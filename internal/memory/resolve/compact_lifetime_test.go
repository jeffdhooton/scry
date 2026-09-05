package resolve

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func compactCacheCount() int {
	compactIdxMu.Lock()
	defer compactIdxMu.Unlock()
	return len(compactIdxBy)
}

func TestApplyReleasesTransactionalCompactCaches(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("rollback=%v", fail), func(t *testing.T) {
			st := openTemp(t)
			putEntity(t, st, "occupied-target", "Unrelated Incumbent", "service")
			if err := RefreshCompactIndex(st); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { forgetCompactIndex(st) })
			parentCache := compactIndex(st)
			baseline := compactCacheCount()
			for i := 0; i < 12; i++ {
				now := time.Now().UTC()
				ep := store.Episode{ID: fmt.Sprintf("cache-lifetime-%d", i), OccurredAt: now, IngestedAt: now}
				res := extract.Result{Entities: []extract.Ent{{Name: "Aurora Atlas", Type: "project", Aliases: []string{fmt.Sprintf("Aurora Atlas Companion %d", i)}}}}
				if fail {
					res.Entities = append(res.Entities, extract.Ent{Name: "Occupied Target", Type: "service"})
				}
				_, err := Apply(st, ep, "", res, DefaultExclusive)
				if (fail && !errors.Is(err, store.ErrAliasClaimed)) || (!fail && err != nil) {
					t.Fatalf("Apply error = %v, rollback=%v", err, fail)
				}
				if n := compactCacheCount(); n != baseline {
					t.Fatalf("episode %d retained transactional caches: before=%d after=%d", i, baseline, n)
				}
				if compactIndex(st) != parentCache {
					t.Fatal("transaction cleanup replaced the parent's independent cache")
				}
				if _, found := parentCache.lookup(compactName("Aurora Atlas")); found {
					t.Fatal("transaction snapshot leaked into parent cache")
				}
			}
		})
	}
}

// Optional scale reproduction: reads a supplied backup into a disposable
// store; never opens a live store, calls a provider, or changes the backup.
func TestApplyCompactCacheLifetimeOnBackup(t *testing.T) {
	path := os.Getenv("SCRY_MEMORY_TEST_BACKUP")
	if path == "" {
		t.Skip("set SCRY_MEMORY_TEST_BACKUP to test a disposable restored replica")
	}
	st := openTemp(t)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := st.Restore(f); err != nil {
		t.Fatal(err)
	}
	_, entities, facts, err := st.Counts()
	if err != nil {
		t.Fatal(err)
	}
	baseline := compactCacheCount()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("Cache Lifetime Cedar Atlas %d", i)
		now := time.Now().UTC()
		ep := store.Episode{ID: fmt.Sprintf("cache-lifetime-scale-%d", i), OccurredAt: now, IngestedAt: now}
		if known, err := st.HasEpisode(ep.ID); err != nil || known {
			t.Fatalf("fixture already present or unreadable: known=%v err=%v", known, err)
		}
		_, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: name, Type: "service", Aliases: []string{name + " service"}}}}, DefaultExclusive)
		if err != nil {
			t.Fatal(err)
		}
		if known, err := st.HasEpisode(ep.ID); err != nil || !known {
			t.Fatalf("episode not committed: known=%v err=%v", known, err)
		}
		if n := compactCacheCount(); n != baseline {
			t.Fatalf("episode %d retained caches: before=%d after=%d", i, baseline, n)
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("replica entities=%d facts=%d; 12 committed episodes; caches %d -> %d; GC heap %d -> %d (delta %d)", entities, facts, baseline, compactCacheCount(), before.HeapAlloc, after.HeapAlloc, int64(after.HeapAlloc)-int64(before.HeapAlloc))
}
