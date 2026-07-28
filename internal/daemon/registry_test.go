package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/index"
)

// makeIndexedRepo creates a repo directory plus a minimal on-disk index layout
// under scryHome so Registry.Get treats the repo as indexed. Returns the repo
// path it indexed under (the canonical form of dir).
func makeIndexedRepo(t *testing.T, scryHome, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	canon := canonicalPath(dir)
	layout := index.Layout(scryHome, canon)
	if err := os.MkdirAll(layout.BadgerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return canon
}

func TestRegistryGetResolvesSymlinkedPath(t *testing.T) {
	scryHome := t.TempDir()
	base := t.TempDir()

	real := filepath.Join(base, "real-repo")
	canon := makeIndexedRepo(t, scryHome, real)

	link := filepath.Join(base, "link-repo")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	defer r.CloseAll()

	e, err := r.Get(scryHome, link)
	if err != nil {
		t.Fatalf("Get via symlink: %v", err)
	}
	if e.RepoPath != canon {
		t.Errorf("RepoPath = %q, want %q", e.RepoPath, canon)
	}
}

func TestRegistryGetWalksUpToIndexedAncestor(t *testing.T) {
	scryHome := t.TempDir()
	base := t.TempDir()

	repo := filepath.Join(base, "repo")
	canon := makeIndexedRepo(t, scryHome, repo)

	sub := filepath.Join(repo, "internal", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	defer r.CloseAll()

	e, err := r.Get(scryHome, sub)
	if err != nil {
		t.Fatalf("Get from subdirectory: %v", err)
	}
	if e.RepoPath != canon {
		t.Errorf("RepoPath = %q, want %q", e.RepoPath, canon)
	}
}

func TestRegistryGetSymlinkIntoOtherRepo(t *testing.T) {
	// The scribe layout: monorepo/apps/foo is a symlink to a separately
	// indexed repo elsewhere on disk. Querying via the symlinked path must
	// find the target repo's index.
	scryHome := t.TempDir()
	base := t.TempDir()

	target := filepath.Join(base, "target-repo")
	canon := makeIndexedRepo(t, scryHome, target)

	mono := filepath.Join(base, "mono", "apps")
	if err := os.MkdirAll(mono, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(mono, "foo")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	defer r.CloseAll()

	e, err := r.Get(scryHome, link)
	if err != nil {
		t.Fatalf("Get via cross-repo symlink: %v", err)
	}
	if e.RepoPath != canon {
		t.Errorf("RepoPath = %q, want %q", e.RepoPath, canon)
	}
}

func TestRegistryGetUnindexedStillErrors(t *testing.T) {
	scryHome := t.TempDir()
	dir := t.TempDir()

	r := NewRegistry()
	defer r.CloseAll()

	if _, err := r.Get(scryHome, dir); err == nil {
		t.Fatal("expected error for unindexed repo, got nil")
	}
}

func TestRegistryEvictRemovesAliases(t *testing.T) {
	scryHome := t.TempDir()
	base := t.TempDir()

	real := filepath.Join(base, "real-repo")
	canon := makeIndexedRepo(t, scryHome, real)

	link := filepath.Join(base, "link-repo")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	defer r.CloseAll()

	if _, err := r.Get(scryHome, link); err != nil {
		t.Fatalf("Get via symlink: %v", err)
	}
	r.Evict(canon)

	// After eviction the alias must not serve a closed store: a fresh Get
	// through the alias should lazily reopen and succeed.
	e, err := r.Get(scryHome, link)
	if err != nil {
		t.Fatalf("Get via symlink after evict: %v", err)
	}
	if e.RepoPath != canon {
		t.Errorf("RepoPath = %q, want %q", e.RepoPath, canon)
	}
}

func TestRegistrySnapshotDedupesAliases(t *testing.T) {
	scryHome := t.TempDir()
	base := t.TempDir()

	real := filepath.Join(base, "real-repo")
	makeIndexedRepo(t, scryHome, real)

	link := filepath.Join(base, "link-repo")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	defer r.CloseAll()

	if _, err := r.Get(scryHome, real); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(scryHome, link); err != nil {
		t.Fatal(err)
	}
	if got := len(r.Snapshot()); got != 1 {
		t.Errorf("Snapshot returned %d entries, want 1", got)
	}
}

func TestCanonicalPathResolvesSymlinks(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if got, want := canonicalPath(link), canonicalPath(real); got != want {
		t.Errorf("canonicalPath(link) = %q, want %q", got, want)
	}
}

func TestCanonicalPathNormalizesCase(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "MixedCase")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	lower := filepath.Join(base, "mixedcase")
	if _, err := os.Stat(lower); err != nil {
		t.Skip("filesystem is case-sensitive; nothing to normalize")
	}
	got := canonicalPath(lower)
	if !strings.HasSuffix(got, string(filepath.Separator)+"MixedCase") {
		t.Errorf("canonicalPath(%q) = %q, want on-disk casing MixedCase", lower, got)
	}
}

func TestCanonicalPathMissingPathReturnsAbs(t *testing.T) {
	// Paths that don't exist must still resolve to something sane (abs form)
	// rather than erroring — callers pass them straight to error messages.
	p := filepath.Join(t.TempDir(), "does-not-exist")
	if got := canonicalPath(p); got != p {
		t.Errorf("canonicalPath(%q) = %q, want unchanged", p, got)
	}
}
