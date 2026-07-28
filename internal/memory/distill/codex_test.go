package distill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const codexFixturePath = "testdata/codex_rollout.jsonl"

func TestCodexRolloutDistills(t *testing.T) {
	episodes, newOffset, err := CodexRollout(codexFixturePath, 0)
	if err != nil {
		t.Fatalf("CodexRollout returned error: %v", err)
	}

	info, err := os.Stat(codexFixturePath)
	if err != nil {
		t.Fatalf("stat fixture: %v", err)
	}
	if newOffset != info.Size() {
		t.Errorf("offset = %d, want file size %d", newOffset, info.Size())
	}

	if len(episodes) < 1 {
		t.Fatalf("expected at least 1 episode, got 0")
	}

	var combined strings.Builder
	for _, ep := range episodes {
		combined.WriteString(ep.Text)
		combined.WriteString("\n")

		if ep.Source != codexSource {
			t.Errorf("episode Source = %q, want %q", ep.Source, codexSource)
		}
		if ep.Cwd != "/Users/jeff/workspace/fixture-codex-project" {
			t.Errorf("episode %s: Cwd = %q, want session_meta cwd", ep.ID, ep.Cwd)
		}
		if ep.OccurredAt.IsZero() {
			t.Errorf("episode %s: OccurredAt not populated", ep.ID)
		}
		if ep.ID == "" {
			t.Errorf("episode has empty ID")
		}
		if !strings.HasPrefix(ep.SourceRef, codexFixturePath+"#") {
			t.Errorf("episode SourceRef = %q, want prefix %q", ep.SourceRef, codexFixturePath+"#")
		}
	}
	text := combined.String()

	for _, want := range []string{
		"check what tests are failing",                    // user turn 1
		"I ran the test suite and found one failing test", // assistant text
		"[tool: shell]",                    // function_call breadcrumb
		"Will do, appreciate the heads up", // later user text
		"rotate that key immediately",      // later assistant text
	} {
		if !strings.Contains(text, want) {
			t.Errorf("combined episode text missing %q; got: %s", want, text)
		}
	}

	for _, notWant := range []string{
		"FAKEFAKE9876543210ZYXWVUTSRQPONMLKJ", // the fake secret payload
		"AssertionError",                      // function_call_output body dropped entirely
		"FAILED tests/test_foo.py",            // function_call_output body dropped entirely
		"You are Codex, a coding agent",       // developer role message dropped
		"I should run the test suite first",   // reasoning content dropped
	} {
		if strings.Contains(text, notWant) {
			t.Errorf("combined episode text unexpectedly contains %q", notWant)
		}
	}

	t.Run("fewer than 3 substantive turns yields zero episodes but offset still advances", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "short_rollout.jsonl")
		content := `{"timestamp":"2026-07-20T09:00:00Z","type":"session_meta","payload":{"cwd":"/tmp/x"}}
{"timestamp":"2026-07-20T09:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello there"}]}}
{"timestamp":"2026-07-20T09:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi!"}]}}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		episodes, offset, err := CodexRollout(path, 0)
		if err != nil {
			t.Fatalf("CodexRollout error: %v", err)
		}
		if len(episodes) != 0 {
			t.Errorf("expected 0 episodes for < 3 substantive turns, got %d", len(episodes))
		}
		if offset != int64(len(content)) {
			t.Errorf("offset = %d, want %d (offset must still advance)", offset, len(content))
		}
	})
}

func TestCodexRolloutOffsetResume(t *testing.T) {
	content, err := os.ReadFile(codexFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	lines := splitLinesKeepEnds(content)
	if len(lines) < 8 {
		t.Fatalf("fixture has %d lines, expected at least 8", len(lines))
	}

	// Offset lands exactly after the first 6 lines (session_meta, developer
	// message, user message, turn_context, reasoning, malformed line) - a
	// clean line boundary landing right before the function_call breadcrumb.
	var offset int64
	for _, l := range lines[:6] {
		offset += int64(len(l))
	}

	fullEpisodes, fullOffset, err := CodexRollout(codexFixturePath, 0)
	if err != nil {
		t.Fatalf("full parse error: %v", err)
	}

	resumedEpisodes, resumedOffset, err := CodexRollout(codexFixturePath, offset)
	if err != nil {
		t.Fatalf("resumed parse error: %v", err)
	}

	if resumedOffset != int64(len(content)) {
		t.Errorf("resumed offset = %d, want file size %d", resumedOffset, len(content))
	}
	if fullOffset != resumedOffset {
		t.Errorf("full offset %d != resumed offset %d", fullOffset, resumedOffset)
	}

	if len(resumedEpisodes) == 0 {
		t.Fatalf("expected at least 1 episode from resumed parse")
	}
	if len(fullEpisodes) == 0 {
		t.Fatalf("expected at least 1 episode from full parse")
	}

	for _, ep := range resumedEpisodes {
		if strings.Contains(ep.Text, "check what tests are failing") {
			t.Errorf("resumed episode covers first-half content it shouldn't: %s", ep.Text)
		}
		// Even though session_meta is before the resume offset, cwd must
		// still be known - CodexRollout pre-scans the file's first line for
		// it regardless of where the offset starts.
		if ep.Cwd != "/Users/jeff/workspace/fixture-codex-project" {
			t.Errorf("resumed episode Cwd = %q, want session_meta cwd even though session_meta predates the offset", ep.Cwd)
		}
	}

	foundSecondHalf := false
	for _, ep := range resumedEpisodes {
		if strings.Contains(ep.Text, "Will do, appreciate the heads up") {
			foundSecondHalf = true
		}
	}
	if !foundSecondHalf {
		t.Errorf("expected resumed episodes to cover second-half content")
	}
}

// TestCodexRolloutPartialTrailingLineNotConsumed mirrors
// TestClaudeSessionPartialTrailingLineNotConsumed: a live rollout file can be
// read mid-write, so its last line on disk may be a partial fragment (no
// trailing '\n' yet). CodexRollout must not count that fragment's bytes as
// consumed.
func TestCodexRolloutPartialTrailingLineNotConsumed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "live_rollout.jsonl")

	meta := `{"timestamp":"2026-07-20T09:00:00Z","type":"session_meta","payload":{"cwd":"/tmp/live"}}` + "\n"
	line1 := `{"timestamp":"2026-07-20T09:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Hello there, first message."}]}}` + "\n"
	line2 := `{"timestamp":"2026-07-20T09:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hi! Second message here."}]}}` + "\n"
	line3Full := `{"timestamp":"2026-07-20T09:00:03Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Third message, delivered whole only after the writer finishes."}]}}` + "\n"
	line4 := `{"timestamp":"2026-07-20T09:00:04Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Fourth message, wraps it up."}]}}` + "\n"
	line5 := `{"timestamp":"2026-07-20T09:00:05Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Fifth message, closes it out."}]}}` + "\n"

	line3Partial := line3Full[:len(line3Full)/2]

	initial := meta + line1 + line2 + line3Partial
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial fixture: %v", err)
	}

	episodes0, offset0, err := CodexRollout(path, 0)
	if err != nil {
		t.Fatalf("first CodexRollout call error: %v", err)
	}

	wantOffset0 := int64(len(meta) + len(line1) + len(line2))
	if offset0 != wantOffset0 {
		t.Errorf("offset after partial-line read = %d, want %d (end of last complete line, excluding the unterminated fragment)", offset0, wantOffset0)
	}
	if len(episodes0) != 0 {
		t.Errorf("expected 0 episodes before the 3rd turn completes, got %d", len(episodes0))
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	rest := line3Full[len(line3Partial):] + line4 + line5
	if _, err := f.WriteString(rest); err != nil {
		t.Fatalf("append rest of file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close appended file: %v", err)
	}

	episodes1, offset1, err := CodexRollout(path, offset0)
	if err != nil {
		t.Fatalf("resumed CodexRollout call error: %v", err)
	}

	fullContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read final fixture: %v", err)
	}
	if offset1 != int64(len(fullContent)) {
		t.Errorf("offset after resume = %d, want file size %d", offset1, len(fullContent))
	}

	if len(episodes1) == 0 {
		t.Fatalf("expected at least 1 episode once the 3rd and 4th turns are complete")
	}
	var combined strings.Builder
	for _, ep := range episodes1 {
		combined.WriteString(ep.Text)
	}
	text := combined.String()

	if !strings.Contains(text, "Third message, delivered whole only after the writer finishes.") {
		t.Errorf("resumed episodes lost the previously-partial turn; got: %s", text)
	}
	if !strings.Contains(text, "Fourth message, wraps it up.") {
		t.Errorf("resumed episodes missing the newly appended turn; got: %s", text)
	}
	if !strings.Contains(text, "Fifth message, closes it out.") {
		t.Errorf("resumed episodes missing the newly appended turn; got: %s", text)
	}
}

// TestCodexRolloutCompleteFinalLineWithoutNewline mirrors
// TestClaudeSessionCompleteFinalLineWithoutNewline: a rollout file that has
// stopped growing for good whose last record is syntactically complete but
// simply missing its trailing newline must still be ingested.
func TestCodexRolloutCompleteFinalLineWithoutNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abandoned_rollout.jsonl")

	meta := `{"timestamp":"2026-07-20T09:00:00Z","type":"session_meta","payload":{"cwd":"/tmp/dead"}}` + "\n"
	line1 := `{"timestamp":"2026-07-20T09:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Starting a task that will never finish."}]}}` + "\n"
	line2 := `{"timestamp":"2026-07-20T09:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"On it."}]}}` + "\n"
	line3NoNewline := `{"timestamp":"2026-07-20T09:00:03Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Last thing I said before the process died."}]}}`

	content := meta + line1 + line2 + line3NoNewline
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	episodes, offset, err := CodexRollout(path, 0)
	if err != nil {
		t.Fatalf("CodexRollout error: %v", err)
	}

	if offset != int64(len(content)) {
		t.Errorf("offset = %d, want %d (end of file, including the unterminated-but-complete final record)", offset, len(content))
	}

	if len(episodes) == 0 {
		t.Fatalf("expected at least 1 episode (3 substantive turns present)")
	}
	var combined strings.Builder
	for _, ep := range episodes {
		combined.WriteString(ep.Text)
	}
	text := combined.String()

	if !strings.Contains(text, "Last thing I said before the process died.") {
		t.Errorf("final unterminated-but-complete turn was not ingested; got: %s", text)
	}
	if !strings.Contains(text, "Starting a task that will never finish.") {
		t.Errorf("expected first turn's text present; got: %s", text)
	}
}
