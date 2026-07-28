package ingest

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

const (
	claudeFixture = "../distill/testdata/claude_session.jsonl"
	loomFixture   = "../distill/testdata/loom_run"
	seedFixture   = "../distill/testdata/seed_memory.md"
)

// fakeExtractor is a minimal in-memory Extractor for these tests, following
// the pattern in internal/memory/extract's own tests. errForID lets a
// specific episode fail extraction without failing every episode.
type fakeExtractor struct {
	err      error
	errForID map[string]bool
	calls    []distill.RawEpisode
}

func (f *fakeExtractor) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (extract.Result, error) {
	f.calls = append(f.calls, ep)
	if f.err != nil {
		return extract.Result{}, f.err
	}
	if f.errForID != nil && f.errForID[ep.ID] {
		return extract.Result{}, errors.New("fake extraction error")
	}
	return extract.Result{EpisodeSummary: "summary for " + ep.ID}, nil
}

// commitCall records one call to fakeDaemon.Commit.
type commitCall struct {
	ep  store.Episode
	cwd string
	res extract.Result
}

// fakeDaemon implements ingest.Daemon, recording every call so tests can
// assert on call counts and arguments.
type fakeDaemon struct {
	glossary      []string
	glossaryCalls int

	commits    []commitCall
	commitErr  error // if set, every Commit call after commitErrAfter fails
	failCommit int   // 1-indexed: Commit call number that should fail; 0 = never

	cursors map[string]store.Cursor
}

func newFakeDaemon() *fakeDaemon {
	return &fakeDaemon{cursors: map[string]store.Cursor{}}
}

func (d *fakeDaemon) Glossary(ctx context.Context, limit int) ([]string, error) {
	d.glossaryCalls++
	if limit != 200 {
		return nil, errors.New("unexpected glossary limit")
	}
	return d.glossary, nil
}

func (d *fakeDaemon) Commit(ctx context.Context, ep store.Episode, cwd string, res extract.Result) (resolve.Stats, error) {
	callNum := len(d.commits) + 1
	if d.failCommit != 0 && callNum == d.failCommit {
		return resolve.Stats{}, d.commitErr
	}
	d.commits = append(d.commits, commitCall{ep: ep, cwd: cwd, res: res})
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

var _ Daemon = (*fakeDaemon)(nil)

func TestFile_ClaudeSource(t *testing.T) {
	wantEpisodes, wantOffset, err := distill.ClaudeSession(claudeFixture, 0)
	if err != nil {
		t.Fatalf("distill.ClaudeSession: %v", err)
	}
	if len(wantEpisodes) == 0 {
		t.Fatal("fixture produced 0 episodes; test needs at least 1")
	}

	info, err := os.Stat(claudeFixture)
	if err != nil {
		t.Fatalf("stat fixture: %v", err)
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	sum, err := File(context.Background(), Options{
		Source:    "claude",
		Path:      claudeFixture,
		Extractor: extractor,
		Daemon:    daemon,
	})
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if daemon.glossaryCalls != 1 {
		t.Errorf("glossaryCalls = %d, want 1", daemon.glossaryCalls)
	}
	if len(daemon.commits) != len(wantEpisodes) {
		t.Errorf("commits = %d, want %d (one per episode)", len(daemon.commits), len(wantEpisodes))
	}
	if sum.EpisodesIngested != len(wantEpisodes) {
		t.Errorf("EpisodesIngested = %d, want %d", sum.EpisodesIngested, len(wantEpisodes))
	}
	if sum.EpisodesSkipped != 0 {
		t.Errorf("EpisodesSkipped = %d, want 0", sum.EpisodesSkipped)
	}

	// Each commit's episode must carry the extraction's summary and the
	// distilled episode's identity fields; cwd is passed separately.
	for i, c := range daemon.commits {
		want := wantEpisodes[i]
		if c.ep.ID != want.ID {
			t.Errorf("commit %d ID = %q, want %q", i, c.ep.ID, want.ID)
		}
		if c.ep.Source != want.Source {
			t.Errorf("commit %d Source = %q, want %q", i, c.ep.Source, want.Source)
		}
		if c.ep.SourceRef != want.SourceRef {
			t.Errorf("commit %d SourceRef = %q, want %q", i, c.ep.SourceRef, want.SourceRef)
		}
		if c.ep.Summary != "summary for "+want.ID {
			t.Errorf("commit %d Summary = %q, want extractor's episode_summary", i, c.ep.Summary)
		}
		if c.ep.IngestedAt.IsZero() {
			t.Errorf("commit %d IngestedAt not set", i)
		}
		if c.cwd != want.Cwd {
			t.Errorf("commit %d cwd = %q, want %q", i, c.cwd, want.Cwd)
		}
	}

	// Cursor advanced to file size (byte offset resume).
	cursor, found := daemon.cursors[claudeFixture]
	if !found {
		t.Fatal("cursor not stored")
	}
	if cursor.ProcessedBytes != wantOffset {
		t.Errorf("cursor.ProcessedBytes = %d, want %d", cursor.ProcessedBytes, wantOffset)
	}
	if cursor.Size != info.Size() {
		t.Errorf("cursor.Size = %d, want %d", cursor.Size, info.Size())
	}
	if cursor.Path != claudeFixture {
		t.Errorf("cursor.Path = %q, want %q", cursor.Path, claudeFixture)
	}

	// Second run: cursor already at EOF, so distillation should find
	// nothing new to ingest.
	sum2, err := File(context.Background(), Options{
		Source:    "claude",
		Path:      claudeFixture,
		Extractor: extractor,
		Daemon:    daemon,
	})
	if err != nil {
		t.Fatalf("second File() error = %v", err)
	}
	if sum2.EpisodesIngested != 0 {
		t.Errorf("second run EpisodesIngested = %d, want 0", sum2.EpisodesIngested)
	}
	if len(daemon.commits) != len(wantEpisodes) {
		t.Errorf("second run added commits: total = %d, want unchanged %d", len(daemon.commits), len(wantEpisodes))
	}
	if daemon.glossaryCalls != 2 {
		t.Errorf("glossaryCalls after second run = %d, want 2 (once per File call)", daemon.glossaryCalls)
	}
}

func TestFile_LoomSource(t *testing.T) {
	info, err := os.Stat(loomFixture)
	if err != nil {
		t.Fatalf("stat fixture dir: %v", err)
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	sum, err := File(context.Background(), Options{
		Source:    "loom",
		Path:      loomFixture,
		Extractor: extractor,
		Daemon:    daemon,
	})
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if daemon.glossaryCalls != 1 {
		t.Errorf("glossaryCalls = %d, want 1", daemon.glossaryCalls)
	}
	if len(daemon.commits) != 1 {
		t.Fatalf("commits = %d, want 1 (loom run is always exactly one episode)", len(daemon.commits))
	}
	if sum.EpisodesIngested != 1 {
		t.Errorf("EpisodesIngested = %d, want 1", sum.EpisodesIngested)
	}

	cursor, found := daemon.cursors[loomFixture]
	if !found {
		t.Fatal("cursor not stored")
	}
	if cursor.ProcessedBytes != 0 {
		t.Errorf("cursor.ProcessedBytes = %d, want 0", cursor.ProcessedBytes)
	}
	if !cursor.ModTime.Equal(info.ModTime()) {
		t.Errorf("cursor.ModTime = %v, want dir mtime %v", cursor.ModTime, info.ModTime())
	}
}

func TestFile_SeedSource(t *testing.T) {
	info, err := os.Stat(seedFixture)
	if err != nil {
		t.Fatalf("stat fixture: %v", err)
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{}

	sum, err := File(context.Background(), Options{
		Source:    "seed",
		Path:      seedFixture,
		Extractor: extractor,
		Daemon:    daemon,
	})
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if len(daemon.commits) != 1 {
		t.Fatalf("commits = %d, want 1 (seed file is always exactly one episode)", len(daemon.commits))
	}
	if sum.EpisodesIngested != 1 {
		t.Errorf("EpisodesIngested = %d, want 1", sum.EpisodesIngested)
	}

	cursor, found := daemon.cursors[seedFixture]
	if !found {
		t.Fatal("cursor not stored")
	}
	if cursor.ProcessedBytes != 0 {
		t.Errorf("cursor.ProcessedBytes = %d, want 0", cursor.ProcessedBytes)
	}
	if !cursor.ModTime.Equal(info.ModTime()) {
		t.Errorf("cursor.ModTime = %v, want file mtime %v", cursor.ModTime, info.ModTime())
	}
}

func TestFile_ExtractionErrorSkipsEpisodeButContinues(t *testing.T) {
	wantEpisodes, _, err := distill.ClaudeSession(claudeFixture, 0)
	if err != nil {
		t.Fatalf("distill.ClaudeSession: %v", err)
	}
	if len(wantEpisodes) == 0 {
		t.Fatal("fixture produced 0 episodes; test needs at least 1")
	}

	daemon := newFakeDaemon()
	extractor := &fakeExtractor{errForID: map[string]bool{wantEpisodes[0].ID: true}}

	sum, err := File(context.Background(), Options{
		Source:    "claude",
		Path:      claudeFixture,
		Extractor: extractor,
		Daemon:    daemon,
	})
	if err != nil {
		t.Fatalf("File() error = %v, want nil (extraction errors are skipped, not fatal)", err)
	}
	if sum.EpisodesSkipped != 1 {
		t.Errorf("EpisodesSkipped = %d, want 1", sum.EpisodesSkipped)
	}
	if sum.EpisodesIngested != len(wantEpisodes)-1 {
		t.Errorf("EpisodesIngested = %d, want %d", sum.EpisodesIngested, len(wantEpisodes)-1)
	}
	// Cursor should still advance: the file was fully consumed, only one
	// episode's extraction failed.
	cursor, found := daemon.cursors[claudeFixture]
	if !found {
		t.Fatal("cursor not stored")
	}
	if cursor.ProcessedBytes == 0 {
		t.Errorf("cursor.ProcessedBytes = 0, want file consumed despite the skipped episode")
	}
}

func TestFile_CommitErrorAbortsWithoutAdvancingCursor(t *testing.T) {
	wantEpisodes, _, err := distill.ClaudeSession(claudeFixture, 0)
	if err != nil {
		t.Fatalf("distill.ClaudeSession: %v", err)
	}
	if len(wantEpisodes) == 0 {
		t.Fatal("fixture produced 0 episodes; test needs at least 1")
	}

	daemon := newFakeDaemon()
	daemon.failCommit = 1
	daemon.commitErr = errors.New("boom")
	extractor := &fakeExtractor{}

	_, err = File(context.Background(), Options{
		Source:    "claude",
		Path:      claudeFixture,
		Extractor: extractor,
		Daemon:    daemon,
	})
	if err == nil {
		t.Fatal("File() error = nil, want error from failed commit")
	}
	if _, found := daemon.cursors[claudeFixture]; found {
		t.Error("cursor was stored despite a commit error; must not advance on failure")
	}
}
