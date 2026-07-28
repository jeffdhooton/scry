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
