package recall

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

func openTemp(t *testing.T) *store.Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "badger")
	s, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func mustPutEntity(t *testing.T, s *store.Store, e store.Entity) {
	t.Helper()
	if err := s.PutEntity(e); err != nil {
		t.Fatalf("PutEntity(%s): %v", e.Slug, err)
	}
}

func mustPutFact(t *testing.T, s *store.Store, f store.Fact) {
	t.Helper()
	if err := s.PutFact(f); err != nil {
		t.Fatalf("PutFact(%s->%s): %v", f.Src, f.Dst, err)
	}
}

func TestPathTwoHops(t *testing.T) {
	s := openTemp(t)
	now := time.Now().UTC()

	for _, slug := range []string{"loom", "deepseek-v4", "book-system"} {
		mustPutEntity(t, s, store.Entity{
			Slug:      slug,
			Name:      slug,
			Type:      "project",
			CreatedAt: now,
			LastSeen:  now,
		})
	}

	mustPutFact(t, s, store.Fact{
		Src:       "loom",
		Relation:  "uses",
		Dst:       "deepseek-v4",
		Fact:      "loom uses deepseek-v4",
		ValidFrom: now.Add(-2 * time.Hour),
	})
	// book-system -> deepseek-v4 stored the same direction; discovered from
	// deepseek-v4's side via the adj: reverse index, exercising "both
	// directions" BFS traversal.
	mustPutFact(t, s, store.Fact{
		Src:       "book-system",
		Relation:  "uses",
		Dst:       "deepseek-v4",
		Fact:      "book-system uses deepseek-v4",
		ValidFrom: now.Add(-1 * time.Hour),
	})

	chain, err := Path(s, "loom", "book-system")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if len(chain) != 2 {
		t.Fatalf("expected 2-hop chain, got %d hops: %+v", len(chain), chain)
	}
	if chain[0].Src != "loom" || chain[0].Dst != "deepseek-v4" {
		t.Fatalf("expected first hop loom->deepseek-v4, got %+v", chain[0])
	}
	if chain[1].Src != "book-system" || chain[1].Dst != "deepseek-v4" {
		t.Fatalf("expected second hop book-system->deepseek-v4, got %+v", chain[1])
	}
}

func TestPathNotFound(t *testing.T) {
	s := openTemp(t)
	now := time.Now().UTC()
	mustPutEntity(t, s, store.Entity{Slug: "loom", Name: "loom", Type: "project", CreatedAt: now, LastSeen: now})
	mustPutEntity(t, s, store.Entity{Slug: "isolated", Name: "isolated", Type: "project", CreatedAt: now, LastSeen: now})

	if _, err := Path(s, "loom", "isolated"); err != store.ErrNotFound {
		t.Fatalf("expected store.ErrNotFound for disconnected entities, got %v", err)
	}
	if _, err := Path(s, "loom", "nonexistent"); err != store.ErrNotFound {
		t.Fatalf("expected store.ErrNotFound for unknown endpoint, got %v", err)
	}
}

func TestOrientRepoFirstAndBudget(t *testing.T) {
	s := openTemp(t)
	now := time.Now().UTC()
	cwd := "/Users/jeff/workspace/context-stack/scry-memory"

	mustPutEntity(t, s, store.Entity{
		Slug:      "scry-memory",
		Name:      "Scry Memory",
		Type:      "project",
		RepoRefs:  []string{cwd},
		CreatedAt: now,
		LastSeen:  now,
	})
	mustPutFact(t, s, store.Fact{
		Src:       "scry-memory",
		Relation:  "status",
		Dst:       "in-progress",
		Fact:      "scry-memory is under active development",
		ValidFrom: now.Add(-time.Hour),
	})

	for i := 0; i < 49; i++ {
		n := strconv.Itoa(i)
		slug := "entity-" + n
		mustPutEntity(t, s, store.Entity{
			Slug:      slug,
			Name:      "Entity " + n,
			Type:      "tool",
			CreatedAt: now,
			LastSeen:  now.Add(-time.Duration(i) * time.Minute),
		})
		mustPutFact(t, s, store.Fact{
			Src:       slug,
			Relation:  "does",
			Dst:       "something-" + n,
			Fact:      "entity " + n + " does something moderately descriptive",
			ValidFrom: now.Add(-time.Hour),
		})
	}

	out, err := Orient(s, cwd, 500, now)
	if err != nil {
		t.Fatalf("Orient: %v", err)
	}
	if len(out) > 500 {
		t.Fatalf("Orient output exceeds budget: %d chars > 500", len(out))
	}
	if !strings.Contains(out, "Query scry_recall for anything referenced here.") {
		t.Fatalf("expected closing pointer line in output:\n%s", out)
	}
	repoIdx := strings.Index(out, "Scry Memory")
	if repoIdx == -1 {
		t.Fatalf("expected repo entity 'Scry Memory' in output:\n%s", out)
	}
	activeIdx := strings.Index(out, "Entity 0")
	if activeIdx != -1 && activeIdx < repoIdx {
		t.Fatalf("expected repo-matched entity before active entities, repoIdx=%d activeIdx=%d", repoIdx, activeIdx)
	}
	if !strings.HasPrefix(out, "## Memory orientation") {
		t.Fatalf("expected header at start of output:\n%s", out)
	}
}

func TestOrientDefaultBudget(t *testing.T) {
	s := openTemp(t)
	now := time.Now().UTC()
	out, err := Orient(s, "/Users/jeff/workspace/empty-repo", 0, now)
	if err != nil {
		t.Fatalf("Orient: %v", err)
	}
	if len(out) > 2000 {
		t.Fatalf("expected default 2000-char budget, got %d chars", len(out))
	}
	if !strings.Contains(out, "Query scry_recall for anything referenced here.") {
		t.Fatalf("expected closing pointer line even with no entities:\n%s", out)
	}
}

func TestEpisodesLimit(t *testing.T) {
	s := openTemp(t)
	now := time.Now().UTC()
	mustPutEntity(t, s, store.Entity{Slug: "loom", Name: "loom", Type: "project", CreatedAt: now, LastSeen: now})

	ids := []string{"ep-a", "ep-b", "ep-c"}
	occurred := []time.Time{now.Add(-3 * time.Hour), now.Add(-2 * time.Hour), now.Add(-1 * time.Hour)}
	for i, id := range ids {
		if err := s.PutEpisode(store.Episode{
			ID:         id,
			Source:     "seed",
			SourceRef:  id,
			OccurredAt: occurred[i],
			IngestedAt: now,
		}); err != nil {
			t.Fatalf("PutEpisode: %v", err)
		}
	}
	mustPutFact(t, s, store.Fact{
		Src:       "loom",
		Relation:  "note",
		Dst:       "x",
		Fact:      "loom note",
		ValidFrom: now.Add(-time.Hour),
		Episodes:  ids,
	})

	eps, err := Episodes(s, "loom", 2)
	if err != nil {
		t.Fatalf("Episodes: %v", err)
	}
	if len(eps) != 2 {
		t.Fatalf("expected 2 episodes (limit), got %d: %+v", len(eps), eps)
	}
	if eps[0].ID != "ep-c" {
		t.Fatalf("expected most recent episode 'ep-c' first, got %q", eps[0].ID)
	}
	if !sort.SliceIsSorted(eps, func(i, j int) bool { return eps[i].OccurredAt.After(eps[j].OccurredAt) }) {
		t.Fatalf("expected episodes sorted most-recent first, got %+v", eps)
	}
}
