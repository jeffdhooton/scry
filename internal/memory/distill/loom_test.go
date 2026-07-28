package distill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const loomFixtureDir = "testdata/loom_run"

func TestLoomRunDistills(t *testing.T) {
	// Deterministic, distinct mtimes so "newest file mtime in the dir" is
	// unambiguous - git checkout doesn't preserve original mtimes, so the
	// test must set them itself rather than relying on whatever land on
	// disk.
	base := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	mtimes := map[string]time.Time{
		"log.md":     base,
		"state.json": base.Add(2 * time.Minute),
		"report.md":  base.Add(5 * time.Minute), // newest
	}
	for name, mt := range mtimes {
		p := filepath.Join(loomFixtureDir, name)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", p, err)
		}
	}

	episodes, err := LoomRun(loomFixtureDir)
	if err != nil {
		t.Fatalf("LoomRun error: %v", err)
	}
	if len(episodes) != 1 {
		t.Fatalf("expected exactly 1 episode per run dir, got %d", len(episodes))
	}
	ep := episodes[0]

	if ep.Source != loomSource {
		t.Errorf("Source = %q, want %q", ep.Source, loomSource)
	}
	if ep.SourceRef != loomFixtureDir {
		t.Errorf("SourceRef = %q, want %q", ep.SourceRef, loomFixtureDir)
	}
	if ep.ID == "" {
		t.Errorf("empty ID")
	}
	wantOccurredAt := mtimes["report.md"]
	if !ep.OccurredAt.Equal(wantOccurredAt) {
		t.Errorf("OccurredAt = %v, want newest mtime %v", ep.OccurredAt, wantOccurredAt)
	}

	for _, want := range []string{
		"fixture-loop",
		"off-by-one error in the pagination helper", // spec goal, from report.md
		"Iterations: 2",
		"iteration 1 [FAIL]",
		"iteration 2 [PASS]",
		"Final status: passed",
	} {
		if !strings.Contains(ep.Text, want) {
			t.Errorf("episode text missing %q; got: %s", want, ep.Text)
		}
	}
}

// TestLoomRunNoReportFallsBackToName covers most real run dirs: report.md is
// only written by loom's deliver() step, so a run that never reached
// delivery (still running, stopped early, budget exhausted) has no goal
// text at all - state.json's "plan" field is generic boilerplate repeated
// on every iteration, not the spec goal. The distiller must still produce
// exactly one episode, using the run's name as the best available stand-in.
func TestLoomRunNoReportFallsBackToName(t *testing.T) {
	dir := t.TempDir()
	state := `{"name":"no-report-run","status":"running","iters":[{"n":1,"plan":"Proceed toward the goal.","summary":"in progress","passed":false,"feedback":"","usd":0,"score":null}],"spent_usd":0}`
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(state), 0o644); err != nil {
		t.Fatalf("write state.json: %v", err)
	}

	episodes, err := LoomRun(dir)
	if err != nil {
		t.Fatalf("LoomRun error: %v", err)
	}
	if len(episodes) != 1 {
		t.Fatalf("expected exactly 1 episode, got %d", len(episodes))
	}
	if !strings.Contains(episodes[0].Text, "no-report-run") {
		t.Errorf("expected fallback goal text to mention the run name; got: %s", episodes[0].Text)
	}
	if !strings.Contains(episodes[0].Text, "Final status: running") {
		t.Errorf("expected final status; got: %s", episodes[0].Text)
	}
}

func TestLoomRunMissingStateErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoomRun(dir); err == nil {
		t.Errorf("expected error for a run dir with no state.json, got nil")
	}
}

// TestLoomRunIDChangesWithContent covers F2: the episode ID must be
// content-sensitive (derived from the newest file mtime in the run dir), not
// just the dir path, so a run dir whose content changes between two
// distills (e.g. a resumed run appending an iteration) produces a distinct
// episode instead of being extracted again at LLM cost and then silently
// dropped by HasEpisode as a duplicate of the stale content.
func TestLoomRunIDChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	state := `{"name":"touch-run","status":"running","iters":[{"n":1,"plan":"p","summary":"first","passed":false,"feedback":"","usd":0,"score":null}],"spent_usd":0}`
	statePath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(statePath, []byte(state), 0o644); err != nil {
		t.Fatalf("write state.json: %v", err)
	}
	t1 := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(statePath, t1, t1); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	first, err := LoomRun(dir)
	if err != nil {
		t.Fatalf("LoomRun (first): %v", err)
	}

	// Touch the file forward in time without changing its logical
	// "iteration" content — mtime alone is what the ID keys off of, mirroring
	// how a resumed run appending to state.json would look from outside.
	t2 := t1.Add(5 * time.Minute)
	if err := os.Chtimes(statePath, t2, t2); err != nil {
		t.Fatalf("chtimes (touch): %v", err)
	}

	second, err := LoomRun(dir)
	if err != nil {
		t.Fatalf("LoomRun (second): %v", err)
	}

	if first[0].ID == second[0].ID {
		t.Errorf("ID unchanged across a content/mtime change: %q", first[0].ID)
	}
}
