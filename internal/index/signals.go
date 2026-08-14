package index

import (
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
// This walks the tree, so it is only the fallback: callers with a git HEAD
// should compare commits instead and never call this.
//
// "Source file" here means exactly what langForExt names, which is what
// detection counts — the two must not drift, or a repo could be detected as
// having a language whose edits never register as staleness.
func NewestSourceMTime(repoPath string) time.Time {
	var newest time.Time
	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort, same as detectLanguages
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
