package sweep

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/ingest"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

const (
	claudeFixture = "../distill/testdata/claude_session.jsonl"
	codexFixture  = "../distill/testdata/codex_rollout.jsonl"
	loomFixture   = "../distill/testdata/loom_run"
)

// --- fakes: same pattern as internal/memory/ingest's tests ---

type fakeExtractor struct {
	calls []distill.RawEpisode
}

func (f *fakeExtractor) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (extract.Result, error) {
	f.calls = append(f.calls, ep)
	return extract.Result{EpisodeSummary: "summary for " + ep.ID}, nil
}

type fakeDaemon struct {
	glossary      []string
	glossaryCalls int

	commits int

	cursors map[string]store.Cursor
}

func newFakeDaemon() *fakeDaemon {
	return &fakeDaemon{cursors: map[string]store.Cursor{}}
}

func (d *fakeDaemon) Glossary(ctx context.Context, limit int) ([]string, error) {
	d.glossaryCalls++
	return d.glossary, nil
}

func (d *fakeDaemon) Commit(ctx context.Context, ep store.Episode, cwd string, res extract.Result) (resolve.Stats, error) {
	d.commits++
	return resolve.Stats{FactsAdded: 1}, nil
}

func (d *fakeDaemon) GetCursor(ctx context.Context, path string) (store.Cursor, bool, error) {
	c, ok := d.cursors[path]
	return c, ok, nil
}

func (d *fakeDaemon) PutCursor(ctx context.Context, c store.Cursor) error {
	d.cursors[c.Path] = c
	return nil
}

var _ ingest.Daemon = (*fakeDaemon)(nil)

// --- fixture plumbing ---

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("readdir %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dst, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Fatalf("copyDir: nested dirs not supported by this test helper (%s)", e.Name())
		}
		copyFile(t, filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()))
	}
}

// backdate sets both the leaf path's mtime (and, for a directory, the
// directory entry's own mtime) to a time safely outside any activeWindow
// used in these tests, so freshly-copied fixtures aren't skipped as "still
// being written".
func backdate(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

// testRoots lays out a temp directory shaped like the real default roots
// (glob depth matters, exact names don't) and copies in the claude, codex,
// and loom fixtures, backdating everything well outside activeWindow.
type testRoots struct {
	roots      Roots
	claudePath string
	codexPath  string
	loomPath   string
}

func newTestRoots(t *testing.T) testRoots {
	t.Helper()
	tmp := t.TempDir()

	claudePath := filepath.Join(tmp, "claude", "projects", "proj1", "session.jsonl")
	codexPath := filepath.Join(tmp, "codex", "sessions", "2026", "07", "20", "rollout-fixture.jsonl")
	loomPath := filepath.Join(tmp, "loom", "runs", "fixture-loop")

	copyFile(t, claudeFixture, claudePath)
	copyFile(t, codexFixture, codexPath)
	copyDir(t, loomFixture, loomPath)

	old := time.Now().Add(-1 * time.Hour)
	backdate(t, claudePath, old)
	backdate(t, codexPath, old)
	backdate(t, loomPath, old) // dir's own mtime, after its contents are written

	return testRoots{
		roots: Roots{
			ClaudeGlob: filepath.Join(tmp, "claude", "projects", "*", "*.jsonl"),
			CodexGlob:  filepath.Join(tmp, "codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
			LoomRuns:   filepath.Join(tmp, "loom", "runs"),
		},
		claudePath: claudePath,
		codexPath:  codexPath,
		loomPath:   loomPath,
	}
}

func fixtureEpisodeCounts(t *testing.T) (claudeN, codexN int) {
	t.Helper()
	ce, _, err := distill.ClaudeSession(claudeFixture, 0)
	if err != nil {
		t.Fatalf("distill.ClaudeSession: %v", err)
	}
	xe, _, err := distill.CodexRollout(codexFixture, 0)
	if err != nil {
		t.Fatalf("distill.CodexRollout: %v", err)
	}
	if len(ce) == 0 || len(xe) == 0 {
		t.Fatal("fixtures must each produce at least 1 episode")
	}
	return len(ce), len(xe)
}

// --- tests ---

func TestRun_FreshSweepIngestsAllRoots(t *testing.T) {
	tr := newTestRoots(t)
	claudeN, codexN := fixtureEpisodeCounts(t)
	wantEpisodes := claudeN + codexN + 1 // +1 for the single loom episode

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	result, err := Run(context.Background(), tr.roots, ingest.Options{
		Extractor: extractor,
		Daemon:    daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.FilesScanned != 3 {
		t.Errorf("FilesScanned = %d, want 3", result.FilesScanned)
	}
	if result.FilesIngested != 3 {
		t.Errorf("FilesIngested = %d, want 3", result.FilesIngested)
	}
	if result.FilesSkippedActive != 0 {
		t.Errorf("FilesSkippedActive = %d, want 0", result.FilesSkippedActive)
	}
	if result.FilesUnchanged != 0 {
		t.Errorf("FilesUnchanged = %d, want 0", result.FilesUnchanged)
	}
	if result.Episodes != wantEpisodes {
		t.Errorf("Episodes = %d, want %d", result.Episodes, wantEpisodes)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}

	if daemon.glossaryCalls != 3 {
		t.Errorf("glossaryCalls = %d, want 3 (one per ingest.File call)", daemon.glossaryCalls)
	}
	if daemon.commits != wantEpisodes {
		t.Errorf("commits = %d, want %d", daemon.commits, wantEpisodes)
	}
	if len(daemon.cursors) != 3 {
		t.Errorf("cursors stored = %d, want 3", len(daemon.cursors))
	}
	for _, p := range []string{tr.claudePath, tr.codexPath, tr.loomPath} {
		if _, ok := daemon.cursors[p]; !ok {
			t.Errorf("no cursor stored for %s", p)
		}
	}
}

func TestRun_SecondSweepAllUnchanged(t *testing.T) {
	tr := newTestRoots(t)
	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	if _, err := Run(context.Background(), tr.roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	callsAfterFirst := len(extractor.calls)
	commitsAfterFirst := daemon.commits

	result, err := Run(context.Background(), tr.roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if result.FilesUnchanged != 3 {
		t.Errorf("FilesUnchanged = %d, want 3", result.FilesUnchanged)
	}
	if result.FilesIngested != 0 {
		t.Errorf("FilesIngested = %d, want 0", result.FilesIngested)
	}
	if result.Episodes != 0 {
		t.Errorf("Episodes = %d, want 0", result.Episodes)
	}
	if len(extractor.calls) != callsAfterFirst {
		t.Errorf("extractor.calls grew from %d to %d on an unchanged sweep", callsAfterFirst, len(extractor.calls))
	}
	if daemon.commits != commitsAfterFirst {
		t.Errorf("commits grew from %d to %d on an unchanged sweep", commitsAfterFirst, daemon.commits)
	}
}

// buildClaudeLines returns n valid, substantive Claude-transcript JSONL
// lines (alternating user/assistant), each newline-terminated, starting at
// the given timestamp and incrementing by a second per line.
func buildClaudeLines(n int, start time.Time) string {
	roles := []string{"user", "assistant"}
	var out string
	for i := 0; i < n; i++ {
		role := roles[i%2]
		ts := start.Add(time.Duration(i) * time.Second)
		out += fmt.Sprintf(
			`{"type":%q,"timestamp":%q,"cwd":"/tmp/fixture","message":{"role":%q,"content":"turn %d substantive text"}}`+"\n",
			role, ts.Format(time.RFC3339), role, i,
		)
	}
	return out
}

func TestRun_AppendedDeltaOnlyIngestsNewContent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "claude", "projects", "proj1", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	round1 := buildClaudeLines(3, time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC))
	if err := os.WriteFile(path, []byte(round1), 0o644); err != nil {
		t.Fatalf("write round1: %v", err)
	}
	old := time.Now().Add(-2 * time.Hour)
	backdate(t, path, old)

	roots := Roots{
		ClaudeGlob: filepath.Join(tmp, "claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(tmp, "empty-codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   filepath.Join(tmp, "empty-loom", "runs"),
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	if _, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if len(extractor.calls) != 1 {
		t.Fatalf("after round 1, extractor.calls = %d, want 1", len(extractor.calls))
	}
	cursorAfterRound1, found := daemon.cursors[path]
	if !found {
		t.Fatal("no cursor stored after round 1")
	}
	round1Len := int64(len(round1))
	if cursorAfterRound1.ProcessedBytes != round1Len {
		t.Fatalf("cursor.ProcessedBytes after round 1 = %d, want %d", cursorAfterRound1.ProcessedBytes, round1Len)
	}

	// Append the delta: 3 more substantive turns.
	round2 := buildClaudeLines(3, time.Date(2026, 7, 20, 9, 5, 0, 0, time.UTC))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := f.WriteString(round2); err != nil {
		t.Fatalf("append round2: %v", err)
	}
	f.Close()
	backdate(t, path, old.Add(time.Minute)) // still outside activeWindow, but a distinct mtime

	result, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if result.FilesIngested != 1 {
		t.Errorf("FilesIngested = %d, want 1 (only the appended file)", result.FilesIngested)
	}
	if result.Episodes != 1 {
		t.Errorf("Episodes = %d, want 1 (only the delta's episode)", result.Episodes)
	}
	// The prior offset was the resume point: the fake daemon's cursor store
	// carried cursorAfterRound1.ProcessedBytes into this run, and only the
	// delta was distilled from it — not the whole file over again.
	if len(extractor.calls) != 2 {
		t.Fatalf("after round 2, extractor.calls = %d, want 2 total (1 old + 1 new)", len(extractor.calls))
	}

	cursorAfterRound2 := daemon.cursors[path]
	wantLen := int64(len(round1) + len(round2))
	if cursorAfterRound2.ProcessedBytes != wantLen {
		t.Errorf("cursor.ProcessedBytes after round 2 = %d, want %d (full file consumed)", cursorAfterRound2.ProcessedBytes, wantLen)
	}
}

func TestRun_TruncatedFileReingestsFromZero(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "claude", "projects", "proj1", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	round1 := buildClaudeLines(6, time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC))
	if err := os.WriteFile(path, []byte(round1), 0o644); err != nil {
		t.Fatalf("write round1: %v", err)
	}
	old := time.Now().Add(-3 * time.Hour)
	backdate(t, path, old)

	roots := Roots{
		ClaudeGlob: filepath.Join(tmp, "claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(tmp, "empty-codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   filepath.Join(tmp, "empty-loom", "runs"),
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	if _, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	cursorAfterRound1 := daemon.cursors[path]
	if cursorAfterRound1.ProcessedBytes != int64(len(round1)) {
		t.Fatalf("cursor.ProcessedBytes after round 1 = %d, want %d", cursorAfterRound1.ProcessedBytes, len(round1))
	}
	callsAfterRound1 := len(extractor.calls)

	// Simulate truncation/rotation: the file is replaced wholesale by
	// something shorter than what was already processed (e.g. log
	// rotation truncated it, or a new session reused the path).
	round2 := buildClaudeLines(3, time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC))
	if len(round2) >= len(round1) {
		t.Fatalf("test fixture invariant broken: round2 (%d bytes) must be shorter than round1 (%d bytes)", len(round2), len(round1))
	}
	if err := os.WriteFile(path, []byte(round2), 0o644); err != nil {
		t.Fatalf("write round2 (truncated): %v", err)
	}
	backdate(t, path, old.Add(time.Hour)) // distinct mtime, still outside activeWindow

	result, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if result.FilesIngested != 1 {
		t.Errorf("FilesIngested = %d, want 1 (truncated file re-ingested)", result.FilesIngested)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}
	if len(extractor.calls) != callsAfterRound1+1 {
		t.Errorf("extractor.calls = %d, want %d (round2 re-parsed fresh from offset 0)", len(extractor.calls), callsAfterRound1+1)
	}

	cursorAfterRound2 := daemon.cursors[path]
	if cursorAfterRound2.ProcessedBytes != int64(len(round2)) {
		t.Errorf("cursor.ProcessedBytes after truncation = %d, want %d (re-ingested from 0, not resumed from the stale offset)", cursorAfterRound2.ProcessedBytes, len(round2))
	}
}

func TestRun_TouchWithoutAppendCountsUnchanged(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "claude", "projects", "proj1", "session.jsonl")
	copyFile(t, claudeFixture, path)
	old := time.Now().Add(-1 * time.Hour)
	backdate(t, path, old)

	roots := Roots{
		ClaudeGlob: filepath.Join(tmp, "claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(tmp, "empty-codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   filepath.Join(tmp, "empty-loom", "runs"),
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	if _, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	callsAfterFirst := len(extractor.calls)
	cursorAfterFirst := daemon.cursors[path]

	// Touch the file (new mtime, identical size/content) without appending
	// anything — e.g. a backup tool or editor re-saving with no changes.
	backdate(t, path, old.Add(30*time.Minute))

	result, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if result.FilesUnchanged != 1 {
		t.Errorf("FilesUnchanged = %d, want 1 (equal size, different mtime = unchanged)", result.FilesUnchanged)
	}
	if result.FilesIngested != 0 {
		t.Errorf("FilesIngested = %d, want 0", result.FilesIngested)
	}
	if len(extractor.calls) != callsAfterFirst {
		t.Errorf("extractor.calls = %d, want unchanged at %d", len(extractor.calls), callsAfterFirst)
	}
	if daemon.cursors[path] != cursorAfterFirst {
		t.Errorf("cursor mutated on a touch-only change: got %+v, want unchanged %+v", daemon.cursors[path], cursorAfterFirst)
	}
}

func TestRun_ActiveWindowSkipsRecentFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "claude", "projects", "proj1", "session.jsonl")
	copyFile(t, claudeFixture, path)
	// Deliberately leave the mtime at "now" (the copy's natural mtime) to
	// simulate a session still in progress.

	roots := Roots{
		ClaudeGlob: filepath.Join(tmp, "claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(tmp, "empty-codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   filepath.Join(tmp, "empty-loom", "runs"),
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	result, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, DefaultActiveWindow, false)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.FilesSkippedActive != 1 {
		t.Errorf("FilesSkippedActive = %d, want 1", result.FilesSkippedActive)
	}
	if result.FilesIngested != 0 {
		t.Errorf("FilesIngested = %d, want 0", result.FilesIngested)
	}
	if len(extractor.calls) != 0 {
		t.Errorf("extractor.calls = %d, want 0 (active file must not be ingested)", len(extractor.calls))
	}
	if len(daemon.cursors) != 0 {
		t.Errorf("cursors stored = %d, want 0", len(daemon.cursors))
	}
}

func TestRun_UnreadableFileErrorsButContinues(t *testing.T) {
	tmp := t.TempDir()
	goodPath := filepath.Join(tmp, "claude", "projects", "proj-a", "session.jsonl")
	badPath := filepath.Join(tmp, "claude", "projects", "proj-b", "session.jsonl")
	copyFile(t, claudeFixture, goodPath)
	copyFile(t, claudeFixture, badPath)

	old := time.Now().Add(-1 * time.Hour)
	backdate(t, goodPath, old)
	backdate(t, badPath, old)

	if err := os.Chmod(badPath, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(badPath, 0o644) }) // let TempDir cleanup remove it

	roots := Roots{
		ClaudeGlob: filepath.Join(tmp, "claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(tmp, "empty-codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   filepath.Join(tmp, "empty-loom", "runs"),
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	result, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.FilesScanned != 2 {
		t.Errorf("FilesScanned = %d, want 2", result.FilesScanned)
	}
	if result.FilesIngested != 1 {
		t.Errorf("FilesIngested = %d, want 1 (the readable file)", result.FilesIngested)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors = %v, want exactly 1", result.Errors)
	}
	if got := result.Errors[0]; !strings.Contains(got, badPath) {
		t.Errorf("Errors[0] = %q, want it to reference %s", got, badPath)
	}
	if _, ok := daemon.cursors[goodPath]; !ok {
		t.Error("cursor not stored for the readable file")
	}
	if _, ok := daemon.cursors[badPath]; ok {
		t.Error("cursor stored for the unreadable file; must not advance on error")
	}
}

func TestRun_UnreadableLoomRootErrorsButOtherRootsStillSweep(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root ignores directory permissions")
	}

	tmp := t.TempDir()
	claudePath := filepath.Join(tmp, "claude", "projects", "proj1", "session.jsonl")
	copyFile(t, claudeFixture, claudePath)
	old := time.Now().Add(-1 * time.Hour)
	backdate(t, claudePath, old)

	loomRoot := filepath.Join(tmp, "loom", "runs")
	if err := os.MkdirAll(loomRoot, 0o755); err != nil {
		t.Fatalf("mkdir loom root: %v", err)
	}
	if err := os.Chmod(loomRoot, 0o000); err != nil {
		t.Fatalf("chmod loom root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(loomRoot, 0o755) }) // let TempDir cleanup remove it

	roots := Roots{
		ClaudeGlob: filepath.Join(tmp, "claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(tmp, "empty-codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   loomRoot,
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	result, err := Run(context.Background(), roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil (a root-listing failure must not abort the sweep)", err)
	}

	if result.FilesIngested != 1 {
		t.Errorf("FilesIngested = %d, want 1 (claude root still swept despite the loom root failing)", result.FilesIngested)
	}
	if _, ok := daemon.cursors[claudePath]; !ok {
		t.Error("cursor not stored for the claude file; the claude root must still be swept")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors = %v, want exactly 1 (the loom root listing failure)", result.Errors)
	}
	if got := result.Errors[0]; !strings.Contains(got, loomRoot) {
		t.Errorf("Errors[0] = %q, want it to reference the loom root %s", got, loomRoot)
	}
}

func TestRun_DryRunMakesNoExtractorCallsOrCursorWrites(t *testing.T) {
	tr := newTestRoots(t)
	claudeN, codexN := fixtureEpisodeCounts(t)
	_ = claudeN
	_ = codexN

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	result, err := Run(context.Background(), tr.roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, true)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.FilesIngested != 3 {
		t.Errorf("FilesIngested = %d, want 3 (dry-run candidates)", result.FilesIngested)
	}
	if result.Episodes != 0 {
		t.Errorf("Episodes = %d, want 0 (dry-run never actually ingests)", result.Episodes)
	}
	if len(extractor.calls) != 0 {
		t.Errorf("extractor.calls = %d, want 0", len(extractor.calls))
	}
	if daemon.commits != 0 {
		t.Errorf("commits = %d, want 0", daemon.commits)
	}
	if daemon.glossaryCalls != 0 {
		t.Errorf("glossaryCalls = %d, want 0", daemon.glossaryCalls)
	}
	if len(daemon.cursors) != 0 {
		t.Errorf("cursors stored = %d, want 0 (dry-run must not advance cursors)", len(daemon.cursors))
	}

	// A real (non-dry) run afterward must behave exactly as if the dry-run
	// never happened: nothing was skipped as "unchanged".
	result2, err := Run(context.Background(), tr.roots, ingest.Options{
		Extractor: extractor, Daemon: daemon,
	}, time.Minute, false)
	if err != nil {
		t.Fatalf("follow-up Run() error = %v", err)
	}
	if result2.FilesIngested != 3 {
		t.Errorf("follow-up FilesIngested = %d, want 3", result2.FilesIngested)
	}
	if result2.FilesUnchanged != 0 {
		t.Errorf("follow-up FilesUnchanged = %d, want 0", result2.FilesUnchanged)
	}
}

func TestDefaultRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	roots := DefaultRoots()
	want := Roots{
		ClaudeGlob: filepath.Join(home, ".claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(home, ".codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   filepath.Join(home, ".loom", "runs"),
	}
	if roots != want {
		t.Errorf("DefaultRoots() = %+v, want %+v", roots, want)
	}
}

func TestRoots_WithDefaultsOnlyFillsEmptyFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	r := Roots{ClaudeGlob: "/custom/claude/*.jsonl"}
	filled := r.withDefaults()
	if filled.ClaudeGlob != "/custom/claude/*.jsonl" {
		t.Errorf("ClaudeGlob overwritten: got %q", filled.ClaudeGlob)
	}
	if filled.CodexGlob != DefaultRoots().CodexGlob {
		t.Errorf("CodexGlob not defaulted: got %q", filled.CodexGlob)
	}
	if filled.LoomRuns != DefaultRoots().LoomRuns {
		t.Errorf("LoomRuns not defaulted: got %q", filled.LoomRuns)
	}
}
