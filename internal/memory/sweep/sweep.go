// Package sweep is the cursor-based scanner that makes scry's memory
// ingestion reliable. internal/memory/ingest's hooks are a latency
// optimization — they ingest a transcript the moment a Claude/Codex session
// ends — but hooks can be skipped (a crashed process, a machine asleep at
// the wrong moment, an agent that never fires its Stop hook). The sweep is
// the source of truth: given a set of roots, it walks every candidate file
// or run directory, compares it against the cursor recorded by the last
// ingest, and ingests whatever changed. Run repeatedly (e.g. from cron), it
// converges the memory store on "everything that exists on disk" regardless
// of what hooks did or didn't fire.
package sweep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/ingest"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// DefaultActiveWindow is the recommended activeWindow for Run: files
// modified more recently than this are presumed to be a session still in
// progress (still being written to) and are skipped for this pass rather
// than ingested mid-write. Callers (the CLI) are expected to pass this by
// default; Run itself applies no implicit default so tests can exercise the
// boundary precisely.
const DefaultActiveWindow = 5 * time.Minute

// PerFileTimeout bounds the daemon round trips for one candidate. Before
// this, a whole sweep shared one 30-minute context: once a run of failures
// burned the budget, every remaining cursor lookup died instantly with an
// i/o timeout on the deadline-bound connection, and the log filled with
// hundreds of identical lines. A per-candidate deadline keeps one bad file
// from poisoning the rest.
var PerFileTimeout = 2 * time.Minute

// Reporter is implemented by daemons that record when a sweep finished and
// what it found (memory.sweepReport). It is optional so tests and older
// daemons still work without it.
type Reporter interface {
	SweepReport(ctx context.Context, r Report) error
}

// Report is what Run tells the daemon at the end of a pass.
type Report struct {
	Host          string `json:"host,omitempty"`
	FilesScanned  int    `json:"files_scanned"`
	FilesIngested int    `json:"files_ingested"`
	Episodes      int    `json:"episodes"`
	Errors        int    `json:"errors"`
}

// Roots names the places the sweep looks for memory sources. Each
// field is independently defaulted (via DefaultRoots) when empty, so
// callers can override just the one root they need — tests always override
// every field with temp-dir paths and must never touch the real home
// directory.
type Roots struct {
	// ClaudeGlob matches Claude Code session transcripts, e.g.
	// $HOME/.claude/projects/*/*.jsonl — top-level session files only, not
	// the memory/ subdirectory scry itself writes into.
	ClaudeGlob string
	// CodexGlob matches Codex CLI rollout transcripts, e.g.
	// $HOME/.codex/sessions/*/*/*/rollout-*.jsonl.
	CodexGlob string
	// LoomRuns is a directory whose immediate subdirectories are each one
	// loom run, e.g. $HOME/.loom/runs.
	LoomRuns string
	// KimiGlob matches Kimi Code agent wire logs, e.g.
	// $HOME/.kimi-code/sessions/*/*/agents/*/wire.jsonl.
	KimiGlob string
	// OpenCodeDB is OpenCode's SQLite database, e.g.
	// $HOME/.local/share/opencode/opencode.db. Each session in it is one
	// candidate.
	OpenCodeDB string
}

// DefaultRoots returns the real, machine-wide default roots, rooted at the
// current user's home directory. Tests must never use this directly —
// always construct a Roots pointing at a temp directory instead.
func DefaultRoots() Roots {
	home, _ := os.UserHomeDir()
	return Roots{
		ClaudeGlob: filepath.Join(home, ".claude", "projects", "*", "*.jsonl"),
		CodexGlob:  filepath.Join(home, ".codex", "sessions", "*", "*", "*", "rollout-*.jsonl"),
		LoomRuns:   filepath.Join(home, ".loom", "runs"),
		KimiGlob:   filepath.Join(home, ".kimi-code", "sessions", "*", "*", "agents", "*", "wire.jsonl"),
		OpenCodeDB: filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
	}
}

// withDefaults fills any empty field of r from DefaultRoots(), leaving
// explicitly-set fields untouched.
func (r Roots) withDefaults() Roots {
	d := DefaultRoots()
	if r.ClaudeGlob == "" {
		r.ClaudeGlob = d.ClaudeGlob
	}
	if r.CodexGlob == "" {
		r.CodexGlob = d.CodexGlob
	}
	if r.LoomRuns == "" {
		r.LoomRuns = d.LoomRuns
	}
	if r.KimiGlob == "" {
		r.KimiGlob = d.KimiGlob
	}
	if r.OpenCodeDB == "" {
		r.OpenCodeDB = d.OpenCodeDB
	}
	return r
}

// Result reports what one Run call did across every root.
type Result struct {
	FilesScanned       int // every file/dir examined, regardless of outcome
	FilesIngested      int // changed and ingested (or, in dry-run, would be)
	FilesSkippedActive int // mtime within activeWindow of now
	FilesUnchanged     int // no cursor delta, nothing to do
	Episodes           int // sum of Summary.EpisodesIngested across all ingests
	Errors             []string
}

// Run scans every root, compares each candidate against its stored cursor,
// and ingests whatever changed via ingest.File (distill + enqueue; the
// daemon extracts).
//
// o supplies only Daemon; o.Source and o.Path are ignored — Run sets both
// per candidate as it walks the roots. Each candidate gets its own context
// bounded by PerFileTimeout, derived from ctx. Files with mtime
// within activeWindow of time.Now() are skipped entirely (neither ingested
// nor counted unchanged) since they may still be mid-write. When dryRun is
// true, Run reports what would be ingested (FilesIngested counts
// candidates) without enqueueing anything, and cursors are left untouched.
//
// Files/dirs are processed in sorted path order within each root (claude,
// then codex, then loom) for determinism. A per-file error is appended to
// Result.Errors and does not abort the sweep — every other candidate is
// still processed. Likewise, a failure listing one root (a malformed glob
// pattern, or an unreadable LoomRuns directory) is recorded in
// Result.Errors and does not prevent the other roots from being swept —
// this runs unattended from cron, so one bad root must degrade, not abort
// the whole pass. Run itself returns a non-nil error only for something
// that invalidates the entire call (currently: none: root-listing failures
// are always recoverable per-root).
func Run(ctx context.Context, roots Roots, o ingest.Options, activeWindow time.Duration, dryRun bool) (Result, error) {
	roots = roots.withDefaults()

	var result Result

	claudeFiles, err := filepath.Glob(roots.ClaudeGlob)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", roots.ClaudeGlob, err))
		claudeFiles = nil
	}
	sort.Strings(claudeFiles)

	codexFiles, err := filepath.Glob(roots.CodexGlob)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", roots.CodexGlob, err))
		codexFiles = nil
	}
	sort.Strings(codexFiles)

	loomDirs, err := listLoomRunDirs(roots.LoomRuns)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", roots.LoomRuns, err))
		loomDirs = nil
	}

	kimiFiles, err := filepath.Glob(roots.KimiGlob)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", roots.KimiGlob, err))
		kimiFiles = nil
	}
	sort.Strings(kimiFiles)

	openCodeSessions, err := distill.OpenCodeSessions(roots.OpenCodeDB)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", roots.OpenCodeDB, err))
		openCodeSessions = nil
	}

	now := time.Now()

	for _, path := range claudeFiles {
		perFile(ctx, func(fctx context.Context) { sweepFile(fctx, &result, o, "claude", path, now, activeWindow, dryRun) })
	}
	for _, path := range codexFiles {
		perFile(ctx, func(fctx context.Context) { sweepFile(fctx, &result, o, "codex", path, now, activeWindow, dryRun) })
	}
	for _, dir := range loomDirs {
		perFile(ctx, func(fctx context.Context) { sweepLoomDir(fctx, &result, o, dir, now, activeWindow, dryRun) })
	}
	for _, path := range kimiFiles {
		perFile(ctx, func(fctx context.Context) { sweepFile(fctx, &result, o, "kimi", path, now, activeWindow, dryRun) })
	}
	for _, sess := range openCodeSessions {
		perFile(ctx, func(fctx context.Context) {
			sweepOpenCodeSession(fctx, &result, o, roots.OpenCodeDB, sess, now, activeWindow, dryRun)
		})
	}

	if rep, ok := o.Daemon.(Reporter); ok && !dryRun {
		host, _ := os.Hostname()
		perFile(ctx, func(fctx context.Context) {
			if err := rep.SweepReport(fctx, Report{
				Host: host, FilesScanned: result.FilesScanned, FilesIngested: result.FilesIngested,
				Episodes: result.Episodes, Errors: len(result.Errors),
			}); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("sweep report: %v", err))
			}
		})
	}

	return result, nil
}

// perFile runs fn under a fresh PerFileTimeout-bounded child of ctx.
func perFile(ctx context.Context, fn func(context.Context)) {
	fctx, cancel := context.WithTimeout(ctx, PerFileTimeout)
	defer cancel()
	fn(fctx)
}

// Candidates lists every claude, codex, and loom candidate across roots that
// isn't currently active (mtime within activeWindow of now), with NO
// cursor-based filtering — unlike Run, every matching file/dir is returned
// regardless of what's already been ingested. This is what backfill wants:
// collect everything on disk and let the daemon's commit-side HasEpisode
// idempotency dedupe whatever was already ingested, rather than trusting
// cursors (which only track "how far a normal sweep/ingest got," not "has
// this been backfilled").
//
// Each returned slice is sorted, matching Run's per-root ordering. Per-root
// listing errors (a malformed glob pattern, an unreadable LoomRuns
// directory) and per-candidate stat errors are appended to errs and do not
// prevent the rest of the candidates from being listed.
func Candidates(roots Roots, activeWindow time.Duration) (claudeFiles, codexFiles, loomDirs []string, errs []string) {
	roots = roots.withDefaults()
	now := time.Now()

	claude, err := filepath.Glob(roots.ClaudeGlob)
	if err != nil {
		errs = append(errs, fmt.Sprintf("%s: %v", roots.ClaudeGlob, err))
		claude = nil
	}
	sort.Strings(claude)
	claudeFiles = filterActive(claude, now, activeWindow, &errs)

	codex, err := filepath.Glob(roots.CodexGlob)
	if err != nil {
		errs = append(errs, fmt.Sprintf("%s: %v", roots.CodexGlob, err))
		codex = nil
	}
	sort.Strings(codex)
	codexFiles = filterActive(codex, now, activeWindow, &errs)

	loom, err := listLoomRunDirs(roots.LoomRuns)
	if err != nil {
		errs = append(errs, fmt.Sprintf("%s: %v", roots.LoomRuns, err))
		loom = nil
	}
	loomDirs = filterActive(loom, now, activeWindow, &errs)

	return claudeFiles, codexFiles, loomDirs, errs
}

// filterActive returns the subset of paths whose mtime is at least
// activeWindow before now, appending a stat error for any path to *errs
// rather than aborting. Order is preserved.
func filterActive(paths []string, now time.Time, activeWindow time.Duration, errs *[]string) []string {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			*errs = append(*errs, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		if now.Sub(info.ModTime()) < activeWindow {
			continue
		}
		out = append(out, p)
	}
	return out
}

// listLoomRunDirs returns the sorted, full-path immediate subdirectories of
// root. A missing root is not an error (no runs yet) — it simply yields no
// entries.
func listLoomRunDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// sweepFile handles one episodic (byte-offset-resume) candidate: a claude
// or codex transcript file.
func sweepFile(ctx context.Context, result *Result, o ingest.Options, source, path string, now time.Time, activeWindow time.Duration, dryRun bool) {
	result.FilesScanned++

	info, err := os.Stat(path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}

	if now.Sub(info.ModTime()) < activeWindow {
		result.FilesSkippedActive++
		return
	}

	cursor, found, err := o.Daemon.GetCursor(ctx, path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: get cursor: %v", path, err))
		return
	}

	size := info.Size()
	mtimeChanged := !info.ModTime().Equal(cursor.ModTime)

	truncated := found && mtimeChanged && size < cursor.ProcessedBytes
	changed := !found || size > cursor.ProcessedBytes || truncated

	if !changed {
		result.FilesUnchanged++
		return
	}

	if dryRun {
		result.FilesIngested++
		return
	}

	if truncated {
		// The file is shorter than what we last processed (truncated or
		// rotated out from under us) — reset the cursor to 0 first so
		// ingest.File's offset resume starts from the beginning of the
		// file's current contents instead of seeking past EOF. Episode
		// idempotency on the daemon side dedupes anything re-ingested that
		// was already committed before the rotation.
		if err := o.Daemon.PutCursor(ctx, store.Cursor{Path: path}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: reset cursor: %v", path, err))
			return
		}
	}

	sum, err := ingest.File(ctx, ingest.Options{
		Source: source,
		Path:   path,
		Daemon: o.Daemon,
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}

	result.FilesIngested++
	result.Episodes += sum.EpisodesIngested
}

// sweepLoomDir handles one loom run directory: a wholesale (not
// offset-resumed) candidate whose change detection is dir-mtime-only, since
// a run directory is finished/abandoned state rather than an append-only
// log.
func sweepLoomDir(ctx context.Context, result *Result, o ingest.Options, dir string, now time.Time, activeWindow time.Duration, dryRun bool) {
	result.FilesScanned++

	info, err := os.Stat(dir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", dir, err))
		return
	}

	if now.Sub(info.ModTime()) < activeWindow {
		result.FilesSkippedActive++
		return
	}

	cursor, found, err := o.Daemon.GetCursor(ctx, dir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: get cursor: %v", dir, err))
		return
	}

	changed := !found || !info.ModTime().Equal(cursor.ModTime)
	if !changed {
		result.FilesUnchanged++
		return
	}

	if dryRun {
		result.FilesIngested++
		return
	}

	sum, err := ingest.File(ctx, ingest.Options{
		Source: "loom",
		Path:   dir,
		Daemon: o.Daemon,
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", dir, err))
		return
	}

	result.FilesIngested++
	result.Episodes += sum.EpisodesIngested
}

// sweepOpenCodeSession handles one OpenCode session: a wholesale candidate
// whose change detection is the session's time_updated column, compared
// against the cursor's ModTime. The active window applies to that same
// timestamp, since a session still being driven keeps updating it.
func sweepOpenCodeSession(ctx context.Context, result *Result, o ingest.Options, dbPath string, sess distill.OpenCodeSession, now time.Time, activeWindow time.Duration, dryRun bool) {
	result.FilesScanned++
	ref := distill.OpenCodeRef(dbPath, sess.ID)

	if now.Sub(sess.TimeUpdated) < activeWindow {
		result.FilesSkippedActive++
		return
	}

	cursor, found, err := o.Daemon.GetCursor(ctx, ref)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: get cursor: %v", ref, err))
		return
	}
	if found && cursor.ModTime.Equal(sess.TimeUpdated) {
		result.FilesUnchanged++
		return
	}
	if dryRun {
		result.FilesIngested++
		return
	}

	sum, err := ingest.File(ctx, ingest.Options{Source: "opencode", Path: ref, Daemon: o.Daemon})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", ref, err))
		return
	}
	result.FilesIngested++
	result.Episodes += sum.EpisodesIngested
}
