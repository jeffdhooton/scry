package distill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// loomSource is the Source value stamped on episodes produced by LoomRun.
const loomSource = "loom-run"

// loomIter mirrors loom's IterRecord (loom/loom/memory.py): one entry per
// loop iteration, written to <run-dir>/state.json.
type loomIter struct {
	N        int      `json:"n"`
	Plan     string   `json:"plan"`
	Summary  string   `json:"summary"`
	Passed   bool     `json:"passed"`
	Feedback string   `json:"feedback"`
	Usd      float64  `json:"usd"`
	Score    *float64 `json:"score"`
}

// loomState mirrors loom's RunState (loom/loom/memory.py), the whole
// contents of <run-dir>/state.json.
type loomState struct {
	Name     string     `json:"name"`
	Status   string     `json:"status"`
	Iters    []loomIter `json:"iters"`
	SpentUsd float64    `json:"spent_usd"`
}

// LoomRun distills a single loom run directory (~/.loom/runs/<name>/) into
// exactly one RawEpisode naming the goal, the iteration count, each
// iteration's gate verdict (pass/fail) and its summary, and the run's
// final status. There is no offset/resume concept here - unlike a Claude
// or Codex transcript, a run directory is small, finished (or abandoned)
// state, not an append-only log, so it is always read and distilled in
// full.
//
// The episode ID is content-sensitive (see the mtime comment below): it
// must change whenever the run directory's content changes, so a re-run
// after a content change produces an additive new episode instead of being
// silently dropped as a duplicate of stale content.
func LoomRun(dir string) ([]RawEpisode, error) {
	statePath := filepath.Join(dir, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state loomState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse %s: %w", statePath, err)
	}

	goal := loomGoal(dir, state.Name)

	var b strings.Builder
	fmt.Fprintf(&b, "Loom run %q\n", state.Name)
	fmt.Fprintf(&b, "Goal: %s\n", goal)
	fmt.Fprintf(&b, "Iterations: %d\n", len(state.Iters))
	for _, it := range state.Iters {
		verdict := "FAIL"
		if it.Passed {
			verdict = "PASS"
		}
		fmt.Fprintf(&b, "- iteration %d [%s]: %s\n", it.N, verdict, it.Summary)
	}
	fmt.Fprintf(&b, "Final status: %s\n", state.Status)

	// The ID is derived from dir PLUS the newest mtime among its files, not
	// dir alone: a run directory is re-read in full on every distill call
	// (there is no offset/resume concept here), so a content-independent ID
	// would mean a run whose files changed since the last distill (e.g. a
	// resumed/extended run, or a corrected report.md) gets re-extracted —
	// paying for the LLM call again — only for resolve.Apply's HasEpisode
	// idempotency check to then silently drop the whole result, since the ID
	// looks identical to the already-ingested episode. Mixing the mtime into
	// the ID makes a content change produce a genuinely new episode instead:
	// resolve.Apply merges its facts onto the existing ones by (src,
	// relation, dst) triple, so re-ingesting is additive, not duplicative.
	mtime := newestMtime(dir)
	ep := RawEpisode{
		ID:         makeID(dir + "#" + mtime.UTC().Format(time.RFC3339Nano)),
		Source:     loomSource,
		SourceRef:  dir,
		Text:       Redact(b.String()),
		OccurredAt: mtime,
	}
	return []RawEpisode{ep}, nil
}

// loomGoal returns the spec goal for a run. loom's deliver() step writes
// <run-dir>/report.md with a "**Goal:** <spec.goal>" line (see
// loom/loom/deliver.py's _write_report), but that only happens for runs
// that reached delivery - a run still in progress, stopped early, or
// budget-exhausted before delivery has no report.md at all, and
// state.json's "plan" field is generic boilerplate repeated on every
// iteration, not the original spec goal. For those runs there is simply no
// record of the goal left in the run directory, so the run's name is used
// as the best available stand-in rather than failing distillation outright.
func loomGoal(dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, "report.md"))
	if err != nil {
		return fmt.Sprintf("%s (goal unknown: no report.md)", name)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if g, ok := strings.CutPrefix(line, "**Goal:**"); ok {
			return strings.TrimSpace(g)
		}
	}
	return fmt.Sprintf("%s (goal unknown: report.md missing Goal line)", name)
}

// newestMtime returns the most recent modification time among dir's
// immediate files, or the zero Time if dir has no files or can't be read.
func newestMtime(dir string) time.Time {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}
	}

	var newest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	return newest
}
