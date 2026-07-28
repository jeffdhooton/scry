package distill

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
)

// minSubstantiveTurns is the floor below which a session isn't worth
// turning into an episode at all (e.g. a one-line aborted session).
const minSubstantiveTurns = 3

// claudeSource is the Source value stamped on episodes produced by
// ClaudeSession.
const claudeSource = "claude-session"

// rawLine is the loose shape of one Claude Code transcript JSONL line.
// Only the fields distillation cares about are declared; everything else
// (uuid, sessionId, gitBranch, version, ...) is ignored by encoding/json.
type rawLine struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Cwd       string      `json:"cwd"`
	Message   *rawMessage `json:"message"`
}

type rawMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// rawBlock is one entry of a message's content array. Fields irrelevant to
// a given block type (e.g. Name on a text block) simply stay zero-valued.
type rawBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
	// tool_result carries a "content" field too, but it is intentionally
	// not declared here: we never want it decoded into memory as text.
}

// ClaudeSession reads a Claude Code session transcript (JSONL) starting at
// the given byte offset, and returns distilled + redacted episodes plus the
// new offset (== bytes consumed, i.e. how far the caller has now read).
//
// Parsing is tolerant: malformed lines and lines of an unrecognized type
// are skipped silently, never causing an error. Sessions with fewer than
// minSubstantiveTurns substantive turns yield zero episodes, but the
// returned offset still advances to the end of the file so callers don't
// re-read the same dead session on every poll.
//
// Final-line rule: a session file's last line may reach EOF without a
// trailing newline, for two very different reasons - (a) the writer is
// still mid-flush and the rest is coming any moment, or (b) the file has
// stopped growing for good (process crash, abandoned session) and that
// line, missing only its newline, is all there will ever be. Without
// plumbing in file mtime/liveness, the two are told apart by content: a
// record cut off only by a missing trailing newline is still a complete,
// syntactically valid JSON value; a record truncated mid-write essentially
// never is (an unterminated string/object/array). So an unterminated final
// fragment is treated as consumed - and its turn ingested - only if
// json.Valid accepts it whole; otherwise it is left unconsumed for the next
// call to re-read once more bytes land.
func ClaudeSession(path string, offset int64) ([]RawEpisode, int64, error) {
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
			// A real I/O error mid-read: bail out without counting
			// whatever fragment we have as consumed (mirrors the
			// not-newline-terminated handling below). Deliberately
			// returns nil episodes rather than any turns/episodes
			// accumulated so far - a mid-file read error means we can't
			// vouch for the rest of the file either, so the caller should
			// treat this call as having made no forward progress on
			// episodes (pos is still advanced to what we did manage to
			// read in full, so a retry doesn't redo completed work).
			return nil, pos, readErr
		}

		// A newline-terminated line is always "complete". A non-terminated
		// fragment (only possible at EOF) is complete too, but only if it
		// is a whole, valid JSON value on its own - see the "Final-line
		// rule" doc above. A mid-write partial (invalid JSON) is left
		// unconsumed: do NOT advance pos or parse it, so the next call
		// re-reads it from the same offset once the writer finishes it.
		complete := len(lineBytes) > 0 && lineBytes[len(lineBytes)-1] == '\n'
		if !complete {
			if !json.Valid([]byte(strings.TrimSpace(string(lineBytes)))) {
				break
			}
			// Falls through: a complete, valid JSON record that's simply
			// missing its trailing newline - treated exactly like a
			// newline-terminated line below.
		}

		start := pos
		pos += int64(len(lineBytes))

		line := strings.TrimSpace(strings.TrimSuffix(string(lineBytes), "\n"))
		if line != "" {
			if t, ok := parseTurn(line, start, pos); ok && strings.TrimSpace(t.text) != "" {
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

	return chunkTurns(claudeSource, path, turns), pos, nil
}

// parseTurn attempts to interpret one JSONL line as a substantive user or
// assistant turn. It returns ok=false for malformed JSON, unrecognized
// types, or lines with no message - all silently skippable.
func parseTurn(line string, start, end int64) (turn, bool) {
	var rl rawLine
	if err := json.Unmarshal([]byte(line), &rl); err != nil {
		return turn{}, false
	}
	if rl.Type != "user" && rl.Type != "assistant" {
		return turn{}, false
	}
	if rl.Message == nil {
		return turn{}, false
	}

	role := rl.Message.Role
	if role == "" {
		role = rl.Type
	}

	text := extractText(rl.Message.Content)

	var ts time.Time
	if rl.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, rl.Timestamp); err == nil {
			ts = parsed
		}
	}

	return turn{
		role:  role,
		text:  text,
		start: start,
		end:   end,
		ts:    ts,
		cwd:   rl.Cwd,
	}, true
}

// extractText reduces a message's content field (either a plain string, or
// an array of typed blocks) to plain text: text blocks pass through
// verbatim, tool_use blocks become a "[tool: <name>]" breadcrumb, and
// tool_result blocks (and any other unrecognized block type, e.g.
// "thinking") are dropped entirely.
func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}

	var blocks []rawBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if t := strings.TrimSpace(b.Text); t != "" {
				parts = append(parts, t)
			}
		case "tool_use":
			name := b.Name
			if name == "" {
				name = "unknown"
			}
			parts = append(parts, "[tool: "+name+"]")
		default:
			// tool_result content, thinking blocks, and anything else we
			// don't recognize yet: dropped, not just ignored for text -
			// tool_result's own "content" field is never even unmarshaled
			// above since rawBlock has no field for it.
		}
	}
	return strings.Join(parts, "\n")
}
