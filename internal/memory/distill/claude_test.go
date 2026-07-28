package distill

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const fixturePath = "testdata/claude_session.jsonl"

func TestClaudeSessionDistills(t *testing.T) {
	episodes, newOffset, err := ClaudeSession(fixturePath, 0)
	if err != nil {
		t.Fatalf("ClaudeSession returned error: %v", err)
	}

	info, err := os.Stat(fixturePath)
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

		if ep.Source != "claude-session" {
			t.Errorf("episode Source = %q, want %q", ep.Source, "claude-session")
		}
		if ep.Cwd == "" {
			t.Errorf("episode %s: Cwd not populated", ep.ID)
		}
		if ep.OccurredAt.IsZero() {
			t.Errorf("episode %s: OccurredAt not populated", ep.ID)
		}
		if ep.ID == "" {
			t.Errorf("episode has empty ID")
		}
		if !strings.HasPrefix(ep.SourceRef, fixturePath+"#") {
			t.Errorf("episode SourceRef = %q, want prefix %q", ep.SourceRef, fixturePath+"#")
		}
	}
	text := combined.String()

	for _, want := range []string{
		"check the disk usage",        // user turn 1 text
		"plenty of headroom",          // assistant text
		"[tool: Bash]",                // tool_use breadcrumb
		"Will do, thanks",             // later user text
		"rotate that key immediately", // later assistant text
	} {
		if !strings.Contains(text, want) {
			t.Errorf("combined episode text missing %q; got: %s", want, text)
		}
	}

	for _, notWant := range []string{
		"FAKEFAKE1234567890ABCDEFGHIJKL", // the fake secret payload
		"Filesystem",                     // tool_result body must be dropped entirely
		"/dev/disk1s1",                   // tool_result body must be dropped entirely
	} {
		if strings.Contains(text, notWant) {
			t.Errorf("combined episode text unexpectedly contains %q", notWant)
		}
	}

	t.Run("fewer than 3 substantive turns yields zero episodes but offset still advances", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "short_session.jsonl")
		content := `{"type":"user","timestamp":"2026-07-15T10:00:00Z","cwd":"/tmp/x","message":{"role":"user","content":"hello there"}}
{"type":"assistant","timestamp":"2026-07-15T10:00:01Z","cwd":"/tmp/x","message":{"role":"assistant","content":[{"type":"text","text":"hi!"}]}}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		episodes, offset, err := ClaudeSession(path, 0)
		if err != nil {
			t.Fatalf("ClaudeSession error: %v", err)
		}
		if len(episodes) != 0 {
			t.Errorf("expected 0 episodes for < 3 substantive turns, got %d", len(episodes))
		}
		if offset != int64(len(content)) {
			t.Errorf("offset = %d, want %d (offset must still advance)", offset, len(content))
		}
	})
}

func TestClaudeSessionOffsetResume(t *testing.T) {
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	lines := splitLinesKeepEnds(content)
	if len(lines) < 8 {
		t.Fatalf("fixture has %d lines, expected at least 8", len(lines))
	}

	// Offset lands exactly after the first 4 lines (user, assistant text,
	// assistant tool_use, user tool_result) - a clean line boundary that
	// splits the fixture into two halves, each with independent substance.
	var offset int64
	for _, l := range lines[:4] {
		offset += int64(len(l))
	}

	fullEpisodes, fullOffset, err := ClaudeSession(fixturePath, 0)
	if err != nil {
		t.Fatalf("full parse error: %v", err)
	}

	resumedEpisodes, resumedOffset, err := ClaudeSession(fixturePath, offset)
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

	// The offset lands right after line 4 (user, assistant text, assistant
	// tool_use, user tool_result). Only content from those first 4 lines
	// ("check the disk usage" and "I'll run df -h") must be excluded; line
	// 5 onward ("plenty of headroom" etc.) is part of the resumed half.
	for _, ep := range resumedEpisodes {
		if strings.Contains(ep.Text, "check the disk usage") {
			t.Errorf("resumed episode covers first-half content it shouldn't: %s", ep.Text)
		}
		if strings.Contains(ep.Text, "I'll run df -h") {
			t.Errorf("resumed episode covers first-half content it shouldn't: %s", ep.Text)
		}
	}

	foundSecondHalf := false
	for _, ep := range resumedEpisodes {
		if strings.Contains(ep.Text, "Will do, thanks") {
			foundSecondHalf = true
		}
	}
	if !foundSecondHalf {
		t.Errorf("expected resumed episodes to cover second-half content")
	}
}

// TestClaudeSessionPartialTrailingLineNotConsumed guards against a real bug:
// a live transcript file can be read mid-write, so its last line on disk may
// be a partial fragment (no trailing '\n' yet). ClaudeSession must not count
// that fragment's bytes as consumed - if it did, the next call would resume
// reading from the middle of that line, permanently discarding its head
// once the writer finishes it (the parse of the reassembled line would then
// either fail or land on garbage, and the turn would be silently lost).
func TestClaudeSessionPartialTrailingLineNotConsumed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "live_session.jsonl")

	line1 := `{"type":"user","timestamp":"2026-07-15T10:00:00Z","cwd":"/tmp/live","message":{"role":"user","content":"Hello there, first message."}}` + "\n"
	line2 := `{"type":"assistant","timestamp":"2026-07-15T10:00:01Z","cwd":"/tmp/live","message":{"role":"assistant","content":[{"type":"text","text":"Hi! Second message here."}]}}` + "\n"
	line3Full := `{"type":"assistant","timestamp":"2026-07-15T10:00:02Z","cwd":"/tmp/live","message":{"role":"assistant","content":[{"type":"text","text":"Third message, delivered whole only after the writer finishes."}]}}` + "\n"
	line4 := `{"type":"user","timestamp":"2026-07-15T10:00:03Z","cwd":"/tmp/live","message":{"role":"user","content":"Fourth message, wraps it up."}}` + "\n"
	line5 := `{"type":"assistant","timestamp":"2026-07-15T10:00:04Z","cwd":"/tmp/live","message":{"role":"assistant","content":[{"type":"text","text":"Fifth message, closes it out."}]}}` + "\n"

	// Simulate a writer that has only flushed the first half of line 3, with
	// no trailing newline yet.
	line3Partial := line3Full[:len(line3Full)/2]

	initial := line1 + line2 + line3Partial
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial fixture: %v", err)
	}

	episodes0, offset0, err := ClaudeSession(path, 0)
	if err != nil {
		t.Fatalf("first ClaudeSession call error: %v", err)
	}

	wantOffset0 := int64(len(line1) + len(line2))
	if offset0 != wantOffset0 {
		t.Errorf("offset after partial-line read = %d, want %d (end of last complete line, excluding the unterminated fragment)", offset0, wantOffset0)
	}
	// Only 2 substantive turns are complete at this point (< minSubstantiveTurns),
	// so no episodes yet - this call's job is only to prove the fragment
	// wasn't consumed.
	if len(episodes0) != 0 {
		t.Errorf("expected 0 episodes before the 3rd turn completes, got %d", len(episodes0))
	}

	// The writer finishes line 3 and appends two more whole lines (4 and 5,
	// so the resumed portion alone has >= minSubstantiveTurns and actually
	// yields an episode, proving the turn was recovered rather than just
	// checking a byte offset).
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

	episodes1, offset1, err := ClaudeSession(path, offset0)
	if err != nil {
		t.Fatalf("resumed ClaudeSession call error: %v", err)
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

	// The previously-partial turn must be recovered whole, not lost.
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

// TestClaudeSessionCompleteFinalLineWithoutNewline guards the flip side of
// the partial-line case above: a file that has stopped growing for good
// (crash, abandoned session) whose last record is syntactically complete
// but simply missing its trailing newline. That record must still be
// ingested - refusing forever to consume an unterminated fragment would
// silently and permanently lose it, since no more bytes are ever coming.
func TestClaudeSessionCompleteFinalLineWithoutNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abandoned_session.jsonl")

	line1 := `{"type":"user","timestamp":"2026-07-15T10:00:00Z","cwd":"/tmp/dead","message":{"role":"user","content":"Starting a task that will never finish."}}` + "\n"
	line2 := `{"type":"assistant","timestamp":"2026-07-15T10:00:01Z","cwd":"/tmp/dead","message":{"role":"assistant","content":[{"type":"text","text":"On it."}]}}` + "\n"
	// The final line: a complete, valid JSON record, but with NO trailing
	// newline - and none will ever be appended (this file has stopped
	// growing forever, simulating a crash right after the write() call
	// that wrote the record but before the next write() could add '\n').
	line3NoNewline := `{"type":"assistant","timestamp":"2026-07-15T10:00:02Z","cwd":"/tmp/dead","message":{"role":"assistant","content":[{"type":"text","text":"Last thing I said before the process died."}]}}`

	content := line1 + line2 + line3NoNewline
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	episodes, offset, err := ClaudeSession(path, 0)
	if err != nil {
		t.Fatalf("ClaudeSession error: %v", err)
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

func TestChunking(t *testing.T) {
	base := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	turnText := strings.Repeat("lorem ipsum dolor sit amet consectetur ", 20) // ~800 chars

	var turns []turn
	var pos int64
	const n = 60
	for i := 0; i < n; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		start := pos
		end := start + int64(len(turnText))
		pos = end
		turns = append(turns, turn{
			role:  role,
			text:  turnText,
			start: start,
			end:   end,
			ts:    base.Add(time.Duration(i) * time.Second),
			cwd:   "/tmp/example",
		})
	}

	totalChars := 0
	for _, tu := range turns {
		totalChars += len(tu.text)
	}
	if totalChars < 40000 {
		t.Fatalf("synthetic conversation too small: %d chars", totalChars)
	}

	validStarts := map[int64]bool{}
	validEnds := map[int64]bool{}
	for _, tu := range turns {
		validStarts[tu.start] = true
		validEnds[tu.end] = true
	}

	episodes1 := chunkTurns("claude-session", "/fake/path.jsonl", turns)
	episodes2 := chunkTurns("claude-session", "/fake/path.jsonl", turns)

	if len(episodes1) < 3 {
		t.Fatalf("expected >= 3 episodes, got %d", len(episodes1))
	}
	if len(episodes1) != len(episodes2) {
		t.Fatalf("nondeterministic episode count: %d vs %d", len(episodes1), len(episodes2))
	}

	seen := map[string]bool{}
	for i, ep := range episodes1 {
		if len(ep.Text) > maxEpisodeChars {
			t.Errorf("episode %d text length %d exceeds maxEpisodeChars %d", i, len(ep.Text), maxEpisodeChars)
		}
		if ep.ID != episodes2[i].ID {
			t.Errorf("episode %d ID not deterministic across runs: %s vs %s", i, ep.ID, episodes2[i].ID)
		}
		if seen[ep.ID] {
			t.Errorf("duplicate episode ID: %s", ep.ID)
		}
		seen[ep.ID] = true

		start, end := parseSourceRef(t, ep.SourceRef)
		if !validStarts[start] {
			t.Errorf("episode %d start byte %d is not on a turn boundary", i, start)
		}
		if !validEnds[end] {
			t.Errorf("episode %d end byte %d is not on a turn boundary", i, end)
		}
	}

	for i := 0; i+1 < len(episodes1); i++ {
		_, endI := parseSourceRef(t, episodes1[i].SourceRef)
		startNext, _ := parseSourceRef(t, episodes1[i+1].SourceRef)
		if startNext >= endI {
			t.Errorf("episodes %d and %d do not share an overlapping turn: end=%d start=%d", i, i+1, endI, startNext)
		}
	}
}

// TestChunkingTwoOversizedTurnsNoOverlap covers the boundary documented on
// chunkTurns: two turns that each fit within maxEpisodeChars alone but
// together exceed it can't be paired into one episode, so each lands in its
// own single-turn episode with NO overlap between them (overlap is
// structurally impossible there, not a bug). This must terminate (no
// infinite loop chasing an overlap that can never happen) and must not
// duplicate or drop either turn.
func TestChunkingTwoOversizedTurnsNoOverlap(t *testing.T) {
	base := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

	// ~9000 chars each; rendered (with role prefix) each turn alone is
	// under maxEpisodeChars (16000), but any two together exceed it.
	text0 := "TURNZERO-" + strings.Repeat("x", 9000)
	text1 := "TURNONE-" + strings.Repeat("y", 9000)

	turns := []turn{
		{role: "user", text: text0, start: 0, end: int64(len(text0)), ts: base, cwd: "/tmp/big"},
		{role: "assistant", text: text1, start: int64(len(text0)), end: int64(len(text0) + len(text1)), ts: base.Add(time.Second), cwd: "/tmp/big"},
	}

	episodes := chunkTurns("claude-session", "/fake/big.jsonl", turns)

	if len(episodes) != 2 {
		t.Fatalf("expected exactly 2 episodes (one per oversized-pair turn), got %d", len(episodes))
	}
	for i, ep := range episodes {
		if len(ep.Text) > maxEpisodeChars {
			t.Errorf("episode %d text length %d exceeds maxEpisodeChars %d", i, len(ep.Text), maxEpisodeChars)
		}
	}

	// Both turns present exactly once: turn 0's marker only in episode 0,
	// turn 1's marker only in episode 1 - i.e. no duplication and no loss.
	if !strings.Contains(episodes[0].Text, "TURNZERO-") {
		t.Errorf("episode 0 missing turn 0's content")
	}
	if strings.Contains(episodes[0].Text, "TURNONE-") {
		t.Errorf("episode 0 unexpectedly contains turn 1's content (overlap should be impossible here)")
	}
	if !strings.Contains(episodes[1].Text, "TURNONE-") {
		t.Errorf("episode 1 missing turn 1's content")
	}
	if strings.Contains(episodes[1].Text, "TURNZERO-") {
		t.Errorf("episode 1 unexpectedly contains turn 0's content (overlap should be impossible here)")
	}

	// Confirm they really don't overlap in byte range, matching the "no
	// overlap possible" documentation on chunkTurns.
	_, end0 := parseSourceRef(t, episodes[0].SourceRef)
	start1, _ := parseSourceRef(t, episodes[1].SourceRef)
	if start1 != end0 {
		t.Errorf("expected episode 1 to start exactly where episode 0 ended (no overlap): end0=%d start1=%d", end0, start1)
	}
}

// splitLinesKeepEnds splits content into lines, each retaining its trailing
// newline (except possibly the last), mirroring how ClaudeSession consumes
// the file byte-by-byte via bufio.Reader.ReadBytes('\n').
func splitLinesKeepEnds(content []byte) [][]byte {
	parts := bytes.SplitAfter(content, []byte("\n"))
	if len(parts) > 0 && len(parts[len(parts)-1]) == 0 {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func parseSourceRef(t *testing.T, ref string) (start, end int64) {
	t.Helper()
	idx := strings.LastIndex(ref, "#")
	if idx < 0 {
		t.Fatalf("SourceRef %q missing '#'", ref)
	}
	rangePart := ref[idx+1:]
	sepIdx := strings.LastIndex(rangePart, "-")
	if sepIdx < 0 {
		t.Fatalf("SourceRef %q missing '-' in range", ref)
	}
	s, err := strconv.ParseInt(rangePart[:sepIdx], 10, 64)
	if err != nil {
		t.Fatalf("SourceRef %q bad start: %v", ref, err)
	}
	e, err := strconv.ParseInt(rangePart[sepIdx+1:], 10, 64)
	if err != nil {
		t.Fatalf("SourceRef %q bad end: %v", ref, err)
	}
	return s, e
}
