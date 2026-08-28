package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFDBudgetReserveAndRelease(t *testing.T) {
	b := newFDBudget(100)

	if !b.tryReserve(60) {
		t.Fatal("first reservation should fit")
	}
	if b.used() != 60 || b.available() != 40 {
		t.Fatalf("after reserving 60: used=%d available=%d", b.used(), b.available())
	}
	if b.tryReserve(50) {
		t.Fatal("reservation past the limit should fail")
	}
	if b.used() != 60 {
		t.Fatalf("failed reservation must claim nothing, used=%d", b.used())
	}
	if !b.tryReserve(40) {
		t.Fatal("reservation that exactly fills the budget should fit")
	}

	b.release(100)
	if b.used() != 0 {
		t.Fatalf("after releasing everything: used=%d", b.used())
	}
	// Over-release must not leave the budget negative, which would silently
	// hand out more capacity than exists.
	b.release(50)
	if b.used() != 0 || b.available() != 100 {
		t.Fatalf("over-release corrupted budget: used=%d available=%d", b.used(), b.available())
	}
}

func TestWatchDirFDCostAccountsForEntries(t *testing.T) {
	dir := t.TempDir()
	for i := range 10 {
		if err := os.WriteFile(filepath.Join(dir, string(rune('a'+i))+".ts"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := watchDirFDCost(dir)
	want := 1
	if watchesEveryEntry {
		// One descriptor for the directory, one for each entry inside it.
		want = 11
	}
	if got != want {
		t.Fatalf("watchDirFDCost = %d, want %d (watchesEveryEntry=%v)", got, want, watchesEveryEntry)
	}
}

// newStubWatcher builds a repoWatcher that is already "finished" — Close can
// run against it without a live goroutine.
func newStubWatcher(w *Watcher, path string, lastUsed uint64, reserved int) *repoWatcher {
	done := make(chan struct{})
	close(done)
	rw := &repoWatcher{
		repoPath: path,
		cancel:   func() {},
		done:     done,
		budget:   w.budget,
		lastUsed: lastUsed,
	}
	rw.reserved.Store(int64(reserved))
	return rw
}

func TestEvictLRUChoosesLeastRecentlyUsedAndRefundsBudget(t *testing.T) {
	w := NewWatcher(t.TempDir(), NewRegistry())
	w.SetBudget(1000)

	w.watchers["old"] = newStubWatcher(w, "old", 1, 300)
	w.watchers["mid"] = newStubWatcher(w, "mid", 5, 200)
	w.watchers["new"] = newStubWatcher(w, "new", 9, 100)
	if !w.budget.tryReserve(600) {
		t.Fatal("test setup: budget should hold the stub reservations")
	}

	w.mu.Lock()
	evicted := w.evictLRULocked("")
	w.mu.Unlock()

	if !evicted {
		t.Fatal("expected an eviction")
	}
	if _, still := w.watchers["old"]; still {
		t.Fatal("least recently used watcher should have been evicted")
	}
	if len(w.watchers) != 2 {
		t.Fatalf("expected 2 watchers left, got %d", len(w.watchers))
	}
	// Eviction has to return capacity or it cannot help the caller.
	if w.budget.used() != 300 {
		t.Fatalf("expected 300 reserved after refunding 300, got %d", w.budget.used())
	}
}

func TestEvictLRUNeverEvictsTheExemptRepo(t *testing.T) {
	w := NewWatcher(t.TempDir(), NewRegistry())
	w.SetBudget(1000)
	w.watchers["only"] = newStubWatcher(w, "only", 1, 100)

	w.mu.Lock()
	evicted := w.evictLRULocked("only")
	w.mu.Unlock()

	if evicted {
		t.Fatal("the exempt repo must not be evicted")
	}
	if _, still := w.watchers["only"]; !still {
		t.Fatal("exempt watcher was removed")
	}
}

func TestUnwatchReturnsBudgetSoLaterReposCanBeWatched(t *testing.T) {
	repo := buildRepoTree(t, 4, 50)
	w := NewWatcher(t.TempDir(), NewRegistry())
	w.SetBudget(4096)

	if err := w.Watch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	spent := w.budget.used()
	if spent == 0 {
		t.Fatal("watching a repo should have reserved descriptors")
	}

	w.Unwatch(repo)
	if w.budget.used() != 0 {
		t.Fatalf("Unwatch leaked %d descriptors of budget", w.budget.used())
	}
	w.Close()
}

func TestWatchIsIdempotentAndDoesNotDoubleCharge(t *testing.T) {
	repo := buildRepoTree(t, 3, 40)
	w := NewWatcher(t.TempDir(), NewRegistry())
	w.SetBudget(4096)
	defer w.Close()

	if err := w.Watch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	first := w.budget.used()

	if err := w.Watch(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if w.budget.used() != first {
		t.Fatalf("re-watching the same repo charged the budget twice: %d then %d", first, w.budget.used())
	}
	if len(w.watchers) != 1 {
		t.Fatalf("expected 1 watcher, got %d", len(w.watchers))
	}
}
