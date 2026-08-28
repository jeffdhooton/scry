package daemon

import (
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/graph"
	graphstore "github.com/jeffdhooton/scry/internal/graph/store"
)

// seedGraphStore creates a valid (empty) graph BadgerDB for repo under home
// so GraphRegistry.Get has something to open.
func seedGraphStore(t *testing.T, home, repo string) graph.RepoLayout {
	t.Helper()
	layout := graph.Layout(home, repo)
	st, err := graphstore.Open(layout.BadgerDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	return layout
}

// TestGraphRegistryBuildWindow is the regression test for the silent graph
// staleness bug: every post-reindex graph rebuild failed with "Cannot acquire
// directory lock" because the registry (or an overlapping rebuild goroutine)
// still held the graph BadgerDB open when graph.Build tried to open it.
//
// The contract: between BeginBuild and EndBuild the registry must not hold
// the store open, must not let Get reopen it, and after EndBuild queries work
// again.
func TestGraphRegistryBuildWindow(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	layout := seedGraphStore(t, home, repo)

	r := NewGraphRegistry()

	// Warm the registry the way a query would.
	if _, err := r.Get(home, repo); err != nil {
		t.Fatalf("initial Get: %v", err)
	}

	r.BeginBuild(repo)

	// The builder must be able to take badger's directory lock now — this
	// Open is exactly what graph.Build does and what kept failing.
	st, err := graphstore.Open(layout.BadgerDir)
	if err != nil {
		t.Fatalf("builder could not open graph store during build window: %v", err)
	}

	// A query arriving mid-build must not reopen the store out from under the
	// builder; it gets a retryable error instead.
	if _, err := r.Get(home, repo); err == nil {
		t.Fatal("Get during a build must fail, not reopen the store")
	} else if !strings.Contains(err.Error(), "in progress") {
		t.Fatalf("Get during a build should say a rebuild is in progress, got: %v", err)
	}

	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = graphstore.Open(layout.BadgerDir)
	if err != nil {
		t.Fatal(err)
	}
	r.EndBuild(repo, &GraphEntry{RepoPath: repo, Layout: layout, Store: st})

	// After the build the registry serves the fresh store.
	e, err := r.Get(home, repo)
	if err != nil {
		t.Fatalf("Get after EndBuild: %v", err)
	}
	if e.Store != st {
		t.Fatal("Get after EndBuild must return the store handed to EndBuild")
	}
}

// TestGraphRegistryEndBuildWithoutEntry: a failed build hands EndBuild no
// entry; the registry must recover — the next Get lazily reopens from disk.
func TestGraphRegistryEndBuildWithoutEntry(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	seedGraphStore(t, home, repo)

	r := NewGraphRegistry()
	r.BeginBuild(repo)
	r.EndBuild(repo, nil)

	if _, err := r.Get(home, repo); err != nil {
		t.Fatalf("Get after failed build must lazily reopen: %v", err)
	}
}
