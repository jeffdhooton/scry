package ingest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

var (
	claudeFixture = filepath.Join("..", "distill", "testdata", "claude_session.jsonl")
	loomFixture   = filepath.Join("..", "distill", "testdata", "loom_run")
	seedFixture   = filepath.Join("..", "distill", "testdata", "seed_memory.md")
)

// fakeDaemon implements Daemon, recording every enqueued episode and every
// cursor write. known marks episode IDs the daemon reports as already
// known; failCall makes the Nth Enqueue call (1-indexed) fail.
type fakeDaemon struct {
	enqueued  []distill.RawEpisode
	batches   []int
	known     map[string]bool
	failCall  int
	failErr   error
	cursors   map[string]store.Cursor
	cursorErr error
}

func newFakeDaemon() *fakeDaemon {
	return &fakeDaemon{cursors: map[string]store.Cursor{}, known: map[string]bool{}}
}

func (d *fakeDaemon) Enqueue(_ context.Context, eps []distill.RawEpisode) (int, int, error) {
	d.batches = append(d.batches, len(eps))
	if d.failCall != 0 && len(d.batches) == d.failCall {
		return 0, 0, d.failErr
	}
	var queued, known int
	for _, ep := range eps {
		if d.known[ep.ID] {
			known++
			continue
		}
		d.known[ep.ID] = true
		d.enqueued = append(d.enqueued, ep)
		queued++
	}
	return queued, known, nil
}

func (d *fakeDaemon) GetCursor(_ context.Context, path string) (store.Cursor, bool, error) {
	if d.cursorErr != nil {
		return store.Cursor{}, false, d.cursorErr
	}
	c, ok := d.cursors[path]
	return c, ok, nil
}

func (d *fakeDaemon) PutCursor(_ context.Context, c store.Cursor) error {
	d.cursors[c.Path] = c
	return nil
}

var _ Daemon = (*fakeDaemon)(nil)

func TestFile_ClaudeSourceEnqueuesAndAdvancesCursor(t *testing.T) {
	wantEpisodes, wantOffset, err := distill.ClaudeSession(claudeFixture, 0)
	if err != nil {
		t.Fatalf("distill.ClaudeSession: %v", err)
	}
	if len(wantEpisodes) == 0 {
		t.Fatal("fixture produced 0 episodes; test needs at least 1")
	}
	info, _ := os.Stat(claudeFixture)

	daemon := newFakeDaemon()
	sum, err := File(context.Background(), Options{Source: "claude", Path: claudeFixture, Daemon: daemon})
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if sum.EpisodesIngested != len(wantEpisodes) || sum.EpisodesSkipped != 0 {
		t.Errorf("summary = %+v, want %d ingested, 0 skipped", sum, len(wantEpisodes))
	}
	for i, got := range daemon.enqueued {
		want := wantEpisodes[i]
		if got.ID != want.ID || got.Source != want.Source || got.SourceRef != want.SourceRef || got.Cwd != want.Cwd || got.Text == "" {
			t.Errorf("enqueued %d = %+v, want %+v", i, got, want)
		}
	}
	cursor, found := daemon.cursors[claudeFixture]
	if !found {
		t.Fatal("cursor not stored")
	}
	if cursor.ProcessedBytes != wantOffset || cursor.Size != info.Size() || cursor.Path != claudeFixture {
		t.Errorf("cursor = %+v, want offset %d size %d", cursor, wantOffset, info.Size())
	}

	// Second run: cursor at EOF, nothing to enqueue.
	sum2, err := File(context.Background(), Options{Source: "claude", Path: claudeFixture, Daemon: daemon})
	if err != nil {
		t.Fatalf("second File() error = %v", err)
	}
	if sum2.EpisodesIngested != 0 || len(daemon.enqueued) != len(wantEpisodes) {
		t.Errorf("second run ingested %d, total enqueued %d", sum2.EpisodesIngested, len(daemon.enqueued))
	}
}

func TestFile_LoomAndSeedRecordModTimeCursors(t *testing.T) {
	for _, tc := range []struct{ source, path string }{{"loom", loomFixture}, {"seed", seedFixture}} {
		info, err := os.Stat(tc.path)
		if err != nil {
			t.Fatalf("stat %s: %v", tc.path, err)
		}
		daemon := newFakeDaemon()
		sum, err := File(context.Background(), Options{Source: tc.source, Path: tc.path, Daemon: daemon})
		if err != nil {
			t.Fatalf("%s: File() error = %v", tc.source, err)
		}
		if sum.EpisodesIngested != 1 || len(daemon.enqueued) != 1 {
			t.Errorf("%s: ingested %d, enqueued %d; want 1 and 1", tc.source, sum.EpisodesIngested, len(daemon.enqueued))
		}
		cursor, found := daemon.cursors[tc.path]
		if !found || cursor.ProcessedBytes != 0 || !cursor.ModTime.Equal(info.ModTime()) {
			t.Errorf("%s: cursor = %+v found=%v, want mtime %v", tc.source, cursor, found, info.ModTime())
		}
	}
}

func TestFile_KnownEpisodesCountAsSkippedAndCursorStillAdvances(t *testing.T) {
	wantEpisodes, _, _ := distill.ClaudeSession(claudeFixture, 0)
	daemon := newFakeDaemon()
	daemon.known[wantEpisodes[0].ID] = true

	sum, err := File(context.Background(), Options{Source: "claude", Path: claudeFixture, Daemon: daemon})
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if sum.EpisodesSkipped != 1 || sum.EpisodesIngested != len(wantEpisodes)-1 {
		t.Errorf("summary = %+v", sum)
	}
	if c, found := daemon.cursors[claudeFixture]; !found || c.ProcessedBytes == 0 {
		t.Error("cursor must advance when the daemon already knew an episode")
	}
}

func TestFile_EnqueueErrorAbortsWithoutAdvancingCursor(t *testing.T) {
	daemon := newFakeDaemon()
	daemon.failCall = 1
	daemon.failErr = errors.New("dial unix: connection refused")

	_, err := File(context.Background(), Options{Source: "claude", Path: claudeFixture, Daemon: daemon})
	if err == nil || !errors.Is(err, daemon.failErr) {
		t.Fatalf("File() error = %v, want the enqueue failure", err)
	}
	if _, found := daemon.cursors[claudeFixture]; found {
		t.Error("cursor was stored despite an enqueue error; must not advance on failure")
	}
}

func TestEnqueue_BatchesAndStopsAtFirstFailure(t *testing.T) {
	episodes := make([]distill.RawEpisode, 0, 120)
	for i := range 120 {
		episodes = append(episodes, distill.RawEpisode{ID: fmt.Sprintf("ep-%03d", i), Text: "x"})
	}
	daemon := newFakeDaemon()
	sum, err := Enqueue(context.Background(), daemon, episodes)
	if err != nil {
		t.Fatal(err)
	}
	if sum.EpisodesIngested != 120 || len(daemon.batches) != 3 || daemon.batches[0] != EnqueueBatch || daemon.batches[2] != 20 {
		t.Errorf("sum = %+v batches = %v", sum, daemon.batches)
	}

	failing := newFakeDaemon()
	failing.failCall = 2
	failing.failErr = context.DeadlineExceeded
	sum, err = Enqueue(context.Background(), failing, episodes)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
	if sum.EpisodesIngested != EnqueueBatch || len(failing.batches) != 2 {
		t.Errorf("after failure: sum = %+v batches = %v (must stop at the failed batch)", sum, failing.batches)
	}
}

func TestFile_UnknownSource(t *testing.T) {
	if _, err := File(context.Background(), Options{Source: "gemini", Path: "x", Daemon: newFakeDaemon()}); err == nil {
		t.Fatal("unknown source must be rejected")
	}
}

func TestFile_CursorErrorIsReported(t *testing.T) {
	daemon := newFakeDaemon()
	daemon.cursorErr = errors.New("i/o timeout")
	if _, err := File(context.Background(), Options{Source: "claude", Path: claudeFixture, Daemon: daemon}); err == nil {
		t.Fatal("cursor lookup failure must surface")
	}
}

func TestFile_ForceRereadsFromTheStart(t *testing.T) {
	wantEpisodes, _, _ := distill.ClaudeSession(claudeFixture, 0)
	daemon := newFakeDaemon()
	if _, err := File(context.Background(), Options{Source: "claude", Path: claudeFixture, Daemon: daemon}); err != nil {
		t.Fatal(err)
	}
	first := len(daemon.enqueued)
	if first == 0 {
		t.Fatal("nothing ingested")
	}
	// Without force the cursor is at EOF: nothing is distilled, so no
	// batch is sent at all.
	batches := len(daemon.batches)
	if _, err := File(context.Background(), Options{Source: "claude", Path: claudeFixture, Daemon: daemon}); err != nil {
		t.Fatal(err)
	}
	if len(daemon.batches) != batches || len(daemon.enqueued) != first {
		t.Errorf("a second plain run re-read the file: batches %v, enqueued %d", daemon.batches, len(daemon.enqueued))
	}
	// With force the whole file is re-read and every episode re-offered.
	daemon.known = map[string]bool{}
	if _, err := File(context.Background(), Options{Force: true, Source: "claude", Path: claudeFixture, Daemon: daemon}); err != nil {
		t.Fatal(err)
	}
	if len(daemon.enqueued) != first+len(wantEpisodes) {
		t.Errorf("force re-offered %d episodes, want %d", len(daemon.enqueued)-first, len(wantEpisodes))
	}
}
