package distill

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// kimiSource is the Source value stamped on episodes produced by KimiWire.
const kimiSource = "kimi-session"

// kimiEvent is the loose shape of one Kimi Code wire.jsonl record. The file
// is an event log, not a message list: a user turn arrives as "turn.prompt",
// the assistant's reply is spread over "context.append_loop_event" records
// whose inner event is "content.part" (text or think pieces) or
// "tool.call", and "turn.ended" closes the turn. Only the fields
// distillation reads are declared.
type kimiEvent struct {
	Type  string          `json:"type"`
	Time  int64           `json:"time"` // ms since epoch
	Input []kimiTextInput `json:"input"`
	Event *kimiLoopEvent  `json:"event"`
}

type kimiTextInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type kimiLoopEvent struct {
	Type string    `json:"type"`
	Name string    `json:"name"` // tool.call
	Part *kimiPart `json:"part"` // content.part
	Step int       `json:"step"` // content.part, tool.call, step.begin/end
}

type kimiPart struct {
	Type string `json:"type"` // "text" | "think"
	Text string `json:"text"`
}

// kimiState is the slice of a session's state.json distillation needs.
type kimiState struct {
	Cwd string `json:"cwd"`
}

// KimiWire reads a Kimi Code agent wire log
// (~/.kimi-code/sessions/<workspace>/<session>/agents/<name>/wire.jsonl)
// from the given byte offset and returns distilled + redacted episodes
// plus the new offset, with the same tolerant parsing and final-line rule
// as ClaudeSession.
//
// Reasoning ("think") parts and tool results never reach an episode; tool
// calls become "[tool: <name>]" breadcrumbs, matching the Claude and Codex
// distillers. The session's working directory lives in state.json two
// directories up, so a resumed call still stamps the right Cwd.
func KimiWire(path string, offset int64) ([]RawEpisode, int64, error) {
	cwd := kimiSessionCwd(path)

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

	// The assistant turn under construction: text pieces and breadcrumbs
	// accumulate until the step ends, the turn ends, or the next prompt
	// arrives. Flushing per step matters: a subagent session is one prompt
	// and dozens of tool-driven steps, and accumulating all of it into a
	// single turn left the session below minSubstantiveTurns, so 112 of
	// 125 Kimi logs on this machine yielded nothing at all.
	var (
		cur      strings.Builder
		curStart int64 = -1
		curTS    time.Time
	)
	flush := func(end int64) {
		if curStart >= 0 && strings.TrimSpace(cur.String()) != "" {
			turns = append(turns, turn{role: "assistant", text: strings.TrimSpace(cur.String()), start: curStart, end: end, ts: curTS, cwd: cwd})
		}
		cur.Reset()
		curStart = -1
	}

	for {
		lineBytes, readErr := reader.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, pos, readErr
		}
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
			var ev kimiEvent
			if err := json.Unmarshal([]byte(line), &ev); err == nil {
				ts := time.UnixMilli(ev.Time)
				switch ev.Type {
				case "turn.prompt":
					flush(start)
					if text := kimiInputText(ev.Input); text != "" {
						turns = append(turns, turn{role: "user", text: text, start: start, end: pos, ts: ts, cwd: cwd})
					}
				case "context.append_loop_event":
					if ev.Event == nil {
						break
					}
					piece := ""
					switch ev.Event.Type {
					case "content.part":
						if ev.Event.Part != nil && ev.Event.Part.Type == "text" {
							piece = ev.Event.Part.Text
						}
					case "tool.call":
						if ev.Event.Name != "" {
							piece = "[tool: " + ev.Event.Name + "]"
						}
					}
					if ev.Event.Type == "step.end" {
						flush(pos)
						break
					}
					if piece != "" {
						if curStart < 0 {
							curStart = start
							curTS = ts
						}
						if cur.Len() > 0 {
							cur.WriteString("\n")
						}
						cur.WriteString(piece)
					}
				case "turn.ended":
					flush(pos)
				}
			}
		}

		if readErr == io.EOF || !complete {
			break
		}
	}
	flush(pos)

	if countSubstantive(turns) < minSubstantiveTurns {
		return nil, pos, nil
	}
	return chunkTurns(kimiSource, path, turns), pos, nil
}

// kimiInputText joins the text pieces of a prompt, skipping injected
// system reminders so a permission-mode banner never counts as a user turn.
func kimiInputText(in []kimiTextInput) string {
	var parts []string
	for _, p := range in {
		if p.Type != "text" {
			continue
		}
		t := strings.TrimSpace(p.Text)
		if t == "" || strings.HasPrefix(t, "<system-reminder>") {
			continue
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, "\n")
}

// kimiSessionCwd reads cwd from the session's state.json (two directories
// above agents/<name>/wire.jsonl). Missing or malformed state yields "".
func kimiSessionCwd(wirePath string) string {
	sessionDir := filepath.Dir(filepath.Dir(filepath.Dir(wirePath)))
	b, err := os.ReadFile(filepath.Join(sessionDir, "state.json"))
	if err != nil {
		return ""
	}
	var st kimiState
	if err := json.Unmarshal(b, &st); err != nil {
		return ""
	}
	return st.Cwd
}

// countSubstantive counts turns with non-empty text.
func countSubstantive(turns []turn) int {
	n := 0
	for _, t := range turns {
		if strings.TrimSpace(t.text) != "" {
			n++
		}
	}
	return n
}
