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
		if !isSourceExt(strings.ToLower(filepath.Ext(d.Name()))) {
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

// isSourceExt reports whether ext (lowercased, leading dot) is one of the
// extensions language detection counts.
func isSourceExt(ext string) bool {
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".go", ".php", ".py":
		return true
	}
	return false
}
