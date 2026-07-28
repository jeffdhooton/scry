package distill

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
)

// codexSource is the Source value stamped on episodes produced by
// CodexRollout.
const codexSource = "codex-session"

// codexEnvelope is the loose shape of one Codex rollout JSONL line: every
// record, regardless of type, is a {timestamp, type, payload} envelope.
// Only the fields distillation cares about are declared here.
type codexEnvelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// codexSessionMetaPayload is the payload of a "session_meta" envelope,
// always the first line of a rollout file - it carries the session's
// working directory among other fields we don't need.
type codexSessionMetaPayload struct {
	Cwd string `json:"cwd"`
}

// codexResponseItemPayload is the payload of a "response_item" envelope.
// Its own "type" field discriminates what kind of item it is: "message"
// (role + content, the only kind that carries conversational text),
// "reasoning" (internal chain-of-thought, never surfaced), "function_call"
// / "custom_tool_call" / "web_search_call" / "tool_search_call" (the model
// invoking a tool - kept as a breadcrumb, mirroring Claude's tool_use
// handling), and "function_call_output" / "custom_tool_call_output" /
// "tool_search_output" (the tool's return value - dropped entirely,
// mirroring Claude's tool_result handling).
type codexResponseItemPayload struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"`
}

// codexContentItem is one entry of a message payload's content array.
type codexContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// codexToolCallTypes are response_item payload types that represent the
// model initiating a tool call - kept as a "[tool: <name>]" breadcrumb.
var codexToolCallTypes = map[string]bool{
	"function_call":    true,
	"custom_tool_call": true,
	"web_search_call":  true,
	"tool_search_call": true,
}

// CodexRollout reads a Codex CLI session rollout (JSONL) starting at the
// given byte offset, and returns distilled + redacted episodes plus the
// new offset, following the exact same tolerant-parsing and offset-resume
// semantics as ClaudeSession (see the "Final-line rule" doc on
// ClaudeSession, which applies here unchanged).
//
// Unlike Claude Code transcripts, a rollout's working directory is not
// repeated on every line - it lives only in the "session_meta" envelope,
// which is always the file's first record. So CodexRollout always peeks
// that first line for cwd, independent of the requested offset, meaning
// episodes produced by a resumed (offset > 0) call still carry the correct
// Cwd even though session_meta itself is never re-read past offset 0.
func CodexRollout(path string, offset int64) ([]RawEpisode, int64, error) {
	cwd, err := codexSessionCwd(path)
	if err != nil {
		return nil, offset, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, offset, err
		}
	}

	reader := bufio.NewReader(f)
	pos := offset
	var turns []turn

	for {
		lineBytes, readErr := reader.ReadBytes('\n')

		if readErr != nil && readErr != io.EOF {
			return nil, pos, readErr
		}

		// See ClaudeSession's "Final-line rule" doc: identical logic here.
		complete := len(lineBytes) > 0 && lineBytes[len(lineBytes)-1] == '\n'
		if !complete {
			if !json.Valid([]byte(strings.TrimSpace(string(lineBytes)))) {
				break
			}
		}

		start := pos
		pos += int64(len(lineBytes))

		line := strings.TrimSpace(strings.TrimSuffix(string(lineBytes), "\n"))
		if line != "" {
			if t, ok := parseCodexTurn(line, start, pos, cwd); ok && strings.TrimSpace(t.text) != "" {
				turns = append(turns, t)
			}
		}

		if readErr == io.EOF {
			break
		}
	}

	if len(turns) < minSubstantiveTurns {
		return nil, pos, nil
	}

	return chunkTurns(codexSource, path, turns), pos, nil
}

// codexSessionCwd reads just the rollout file's first line - by Codex CLI
// convention always a "session_meta" envelope - and returns its cwd.
// Tolerant: any read/parse failure or a first line that isn't session_meta
// yields "" rather than an error, matching the tolerant-parsing philosophy
// used throughout this package.
func codexSessionCwd(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	lineBytes, readErr := reader.ReadBytes('\n')
	if readErr != nil && readErr != io.EOF {
		return "", readErr
	}

	line := strings.TrimSpace(string(lineBytes))
	if line == "" {
		return "", nil
	}

	var env codexEnvelope
	if err := json.Unmarshal([]byte(line), &env); err != nil || env.Type != "session_meta" {
		return "", nil
	}

	var meta codexSessionMetaPayload
	if err := json.Unmarshal(env.Payload, &meta); err != nil {
		return "", nil
	}
	return meta.Cwd, nil
}

// parseCodexTurn attempts to interpret one JSONL line as a substantive
// turn: a user/assistant message, or a tool-call breadcrumb. It returns
// ok=false for malformed JSON, non-response_item envelopes, and response
// items that carry no surfaceable text (reasoning, tool outputs, developer/
// system messages) - all silently skippable.
func parseCodexTurn(line string, start, end int64, cwd string) (turn, bool) {
	var env codexEnvelope
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		return turn{}, false
	}
	if env.Type != "response_item" {
		return turn{}, false
	}

	var p codexResponseItemPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return turn{}, false
	}

	var ts time.Time
	if env.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, env.Timestamp); err == nil {
			ts = parsed
		}
	}

	switch {
	case p.Type == "message" && (p.Role == "user" || p.Role == "assistant"):
		text := extractCodexText(p.Content)
		if strings.TrimSpace(text) == "" {
			return turn{}, false
		}
		return turn{role: p.Role, text: text, start: start, end: end, ts: ts, cwd: cwd}, true

	case codexToolCallTypes[p.Type]:
		name := p.Name
		if name == "" {
			name = "unknown"
		}
		return turn{role: "assistant", text: "[tool: " + name + "]", start: start, end: end, ts: ts, cwd: cwd}, true

	default:
		// "message" with role "developer"/"system" (instructions, not
		// conversation), "reasoning" (internal chain-of-thought), and
		// "function_call_output"/"custom_tool_call_output"/
		// "tool_search_output" (tool result bodies): all dropped entirely,
		// exactly as Claude drops tool_result and thinking blocks.
		return turn{}, false
	}
}

// extractCodexText reduces a message payload's content array to plain
// text: "input_text" and "output_text" items pass through verbatim,
// "input_image" and any other item type is dropped.
func extractCodexText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var items []codexContentItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return ""
	}

	var parts []string
	for _, it := range items {
		switch it.Type {
		case "input_text", "output_text":
			if t := strings.TrimSpace(it.Text); t != "" {
				parts = append(parts, t)
			}
		default:
			// input_image and anything else: dropped.
		}
	}
	return strings.Join(parts, "\n")
}
