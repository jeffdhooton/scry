package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// openFDCount reports how many file descriptors this process currently holds.
// It probes with F_GETFD rather than reading /dev/fd, which reports
// inconsistent results from a process that is concurrently opening files.
func openFDCount(t *testing.T) int {
	t.Helper()
	return processFDCount()
}

// buildRepoTree creates a repo with dirs subdirectories, each holding files
// regular files. Names avoid repoSkipDirs and never begin with a dot, so the
// whole tree is eligible for watching.
func buildRepoTree(t *testing.T, dirs, files int) string {
	t.Helper()
	root := t.TempDir()
	for d := range dirs {
		sub := filepath.Join(root, fmt.Sprintf("pkg%03d", d))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		for f := range files {
			p := filepath.Join(sub, fmt.Sprintf("file%03d.ts", f))
			if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

// TestWatchRepoStaysWithinFDBudget is the regression test for the ENFILE
// incident: a daemon watching many repos consumed ~131k descriptors and
// exhausted the system-wide file table.
//
// The per-repo cap counts directories, but on macOS and the BSDs fsnotify's
// kqueue backend needs one descriptor per *file* as well — watchDirectoryFiles
// opens every entry in each watched directory. A tree of 20 directories is far
// under the 2048-dir cap while still costing 4000+ descriptors.
//
// The budget must bound what a single repo can consume regardless of how many
// files its directories hold.
func TestWatchRepoStaysWithinFDBudget(t *testing.T) {
	const budgetLimit = 400
	repo := buildRepoTree(t, 20, 200)

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer fsw.Close()

	before := openFDCount(t)
	budget := newFDBudget(budgetLimit)
	spent := addRepoToWatcher(fsw, repo, budget, budgetLimit)
	after := openFDCount(t)

	delta := after - before
	// Allow generous slack for descriptors the runtime opens incidentally
	// (the readdir in openFDCount, GC-related handles).
	if delta > budgetLimit+64 {
		t.Fatalf("watching one repo opened %d descriptors, budget was %d", delta, budgetLimit)
	}
	if spent > budgetLimit {
		t.Fatalf("budget accounting says %d spent, limit was %d", spent, budgetLimit)
	}
	if budget.used() > budgetLimit {
		t.Fatalf("budget over-committed: used %d, limit %d", budget.used(), budgetLimit)
	}
}

// TestCloseWatcherReleasesDescriptors covers the second half of the leak.
//
// fsnotify v1.9.0's kqueue Close marks the watcher closed before calling Remove
// for each path, and Remove returns early on a closed watcher — so Close alone
// releases nothing. That turns eviction into a trap: it frees budget without
// freeing descriptors, so the daemon over-commits and drifts upward with every
// watcher it closes.
func TestCloseWatcherReleasesDescriptors(t *testing.T) {
	repo := buildRepoTree(t, 8, 60)

	before := openFDCount(t)
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	budget := newFDBudget(1 << 20)
	spent := addRepoToWatcher(fsw, repo, budget, 1<<20)
	if spent < 100 {
		t.Fatalf("test setup: expected the tree to cost real descriptors, got %d", spent)
	}
	watching := openFDCount(t)
	if watching <= before {
		t.Fatal("test setup: watching the repo opened no descriptors")
	}

	closeWatcher(fsw)

	after := openFDCount(t)
	// A couple of descriptors of slack for runtime bookkeeping.
	if after-before > 8 {
		t.Fatalf("closing the watcher leaked %d descriptors (held %d while watching)",
			after-before, watching-before)
	}
}

// TestNewDirectoryGetsWatched covers the gap where a directory created after
// the initial walk never got a watch: the incremental Add sat behind
// relevantEvent, which only passes source-file extensions, so a directory
// create event was dropped before it reached the Add.
func TestNewDirectoryGetsWatched(t *testing.T) {
	repo := buildRepoTree(t, 2, 5)

	w := NewWatcher(t.TempDir(), NewRegistry())
	w.SetBudget(1 << 20)
	if err := w.Watch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	rw := w.watchers[repo]

	// A populated tree appearing at once: nested dir plus a file, the shape a
	// git checkout or cp -r produces.
	fresh := filepath.Join(repo, "fresh", "nested")
	if err := os.MkdirAll(fresh, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fresh, "new.ts"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A skip-named directory must not be watched even when created live.
	if err := os.MkdirAll(filepath.Join(repo, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}

	watched := func(dir string) bool {
		for _, p := range rw.fsw.WatchList() {
			if p == dir {
				return true
			}
		}
		return false
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if watched(filepath.Join(repo, "fresh")) && watched(fresh) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !watched(filepath.Join(repo, "fresh")) {
		t.Fatal("new directory was never added to the watch set")
	}
	if !watched(fresh) {
		t.Fatal("nested new directory was never added to the watch set")
	}
	if watched(filepath.Join(repo, "node_modules")) {
		t.Fatal("skip-named directory was added to the watch set")
	}
}

// TestWatcherEvictionReleasesDescriptors is the same property one level up: an
// eviction has to give the descriptors back, not just the budget.
func TestWatcherEvictionReleasesDescriptors(t *testing.T) {
	repo := buildRepoTree(t, 8, 60)

	before := openFDCount(t)
	w := NewWatcher(t.TempDir(), NewRegistry())
	w.SetBudget(1 << 20)

	if err := w.Watch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if openFDCount(t) <= before {
		t.Fatal("test setup: watching the repo opened no descriptors")
	}

	w.Unwatch(repo)

	if after := openFDCount(t); after-before > 8 {
		t.Fatalf("eviction leaked %d descriptors", after-before)
	}
	if w.budget.used() != 0 {
		t.Fatalf("eviction leaked %d of budget", w.budget.used())
	}
}
