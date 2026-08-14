package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Repo status values. Only StatusReady and StatusPartial are ever written to
// a manifest — they describe what the build itself produced. StatusStale and
// StatusEmpty are DERIVED at report time from the manifest plus the live repo,
// so diagnosing them never requires a reindex and never rewrites the manifest.
const (
	StatusReady   = "ready"
	StatusPartial = "partial"
	StatusStale   = "stale"
	StatusEmpty   = "empty"
)

// IsStale reports whether the index described by m no longer reflects the
// repo's current contents.
//
// The exact comparison is by commit: if we recorded a HEAD at build time and
// can resolve one now, the index is stale precisely when HEAD has moved. That
// is immune to clock skew, to a rebuild that ran long, and to a checkout that
// rewinds to an older commit.
//
// When either side has no commit — the repo isn't a git checkout, git is
// unavailable, or the manifest predates HeadCommit — we fall back to comparing
// the newest source file's mtime against IndexedAt. Callers pass the zero time
// for newestSourceMTime when they don't have it; with neither signal available
// we report not-stale rather than crying wolf on every status call.
func IsStale(m *Manifest, currentHead string, newestSourceMTime time.Time) bool {
	if m == nil {
		return false
	}
	if currentHead != "" && m.HeadCommit != "" {
		return currentHead != m.HeadCommit
	}
	if newestSourceMTime.IsZero() || m.IndexedAt.IsZero() {
		return false
	}
	return newestSourceMTime.After(m.IndexedAt)
}

// EmptyLanguages returns the primary languages whose indexer reported success
// and produced no symbols anyway, across a non-zero count of detected source
// files. That combination is a failure wearing a success label: the tool ran,
// exited 0, and the store it filled is empty.
//
// The predicate leans on a guarantee from the builder (see IndexerResult's
// count fields): the per-language counts are populated only on the success
// path, so they are zero for every status other than IndexerOK. A language
// that never ran therefore can't be mistaken for one that ran and found
// nothing.
//
// The three conditions each rule out a legitimate zero:
//   - TierPrimary: an incidental language is deliberately not indexed deeply,
//     and it can't degrade a repo's status anyway (see deriveStatus).
//   - IndexerOK: missing and failed languages are already reported as
//     "partial", which outranks empty. Flagging them here would double-report
//     one problem under two names.
//   - FileCount > 0: zero symbols from zero files is the correct answer, not a
//     failure. This is the case that keeps a repo with, say, a single stray
//     .py file from being called empty forever.
//
// Order follows the manifest's own result order so the output is stable across
// calls. Returns nil (not an empty slice) when nothing is empty, so callers can
// test with len() and the daemon's omitempty drops the field entirely.
func EmptyLanguages(m *Manifest) []string {
	if m == nil {
		return nil
	}
	if countsUnrecorded(m) {
		return nil
	}
	var empty []string
	for _, r := range m.Indexers {
		if r.Tier != TierPrimary || r.Status != IndexerOK {
			continue
		}
		if r.SymbolCount == 0 && r.FileCount > 0 {
			empty = append(empty, r.Language)
		}
	}
	return empty
}

// countsUnrecorded reports that this manifest predates the per-language count
// fields, so its zero SymbolCounts mean "never written" rather than "produced
// nothing".
//
// The distinction is invisible on any single language — an absent int and a
// recorded zero are the same value — but it is decidable in aggregate. The
// builder assigns each language's SymbolCount from the very same scip.Stats it
// adds into Manifest.Stats, so on any manifest written since the fields
// existed, a positive total implies at least one positive per-language count.
// Symbols in the total with none against any language therefore proves the
// fields were absent when this manifest was written.
//
// This is not a cosmetic guard. Every index sitting on disk today is in this
// shape, and without it EMPTY reports each one as a failure: on one real
// machine, 45 healthy repos flagged at once. That would replace the silent
// green this task removes with a false alarm, which is worse — a signal that
// cries wolf on everything gets ignored, including when it is right.
func countsUnrecorded(m *Manifest) bool {
	if m.Stats.Symbols <= 0 {
		// A zero total is consistent with the counts being present and the
		// build genuinely producing nothing, which is exactly what EMPTY is
		// for. Nothing to suppress.
		return false
	}
	for _, r := range m.Indexers {
		if r.SymbolCount > 0 {
			return false
		}
	}
	return true
}

// EffectiveStatus folds the manifest's recorded status together with the two
// derived signals into the single label a user should see.
//
// Precedence is partial > empty > stale > ready. A partial build is the
// loudest fact: languages are missing outright, and that is worth saying
// before "some other language indexed nothing" or "this is a commit behind".
// Empty outranks stale because an empty language is broken now and a reindex
// at the current commit will not fix it, while stale is fixed by exactly that.
func EffectiveStatus(m *Manifest, stale bool, emptyLanguages []string) string {
	if m == nil {
		return StatusReady
	}
	// Anything the builder recorded other than "ready" (today only "partial",
	// plus the vestigial "broken" in old manifests) already outranks both
	// derived signals, so pass it through untouched.
	if m.Status != "" && m.Status != StatusReady {
		return m.Status
	}
	if len(emptyLanguages) > 0 {
		return StatusEmpty
	}
	if stale {
		return StatusStale
	}
	return StatusReady
}

// NewestSourceMTime returns the most recent modification time across the
// repo's source files, or the zero time if the repo has none (or can't be
// walked). It uses the same extension set and skip list as language
// detection, so a churning node_modules or build directory doesn't make every
// index look stale.
//
// This walks the tree, so it is only the fallback: callers with a comparable
// pair of commits should compare those instead and never call this.
//
// "Source file" here means exactly what langForExt names, which is what
// detection counts — the two must not drift, or a repo could be detected as
// having a language whose edits never register as staleness.
//
// The walk stops early when ctx is done, returning the newest mtime found so
// far. That truncation is deliberately safe in one direction only: seeing
// fewer files can lower the maximum, so it can turn a stale repo into a
// silent one, but it can never invent a file newer than the index. A bounded
// walk therefore under-reports staleness rather than fabricating it — the
// same tradeoff HEAD resolution makes when its budget expires.
func NewestSourceMTime(ctx context.Context, repoPath string) time.Time {
	var newest time.Time
	// Checking the context per file would dominate the walk on a large tree,
	// where each entry is otherwise a cheap already-cached stat.
	const checkEvery = 512
	seen := 0
	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort, same as detectLanguages
		}
		seen++
		if seen%checkEvery == 0 && ctx.Err() != nil {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if langForExt(strings.ToLower(filepath.Ext(d.Name()))) == "" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return time.Time{}
	}
	return newest
}
