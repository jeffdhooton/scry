package recall

import (
	"fmt"
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

func TestOrientRanksRecentAndConnectedFirst(t *testing.T) {
	s := openTemp(t)
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	repo := "/Users/jeff/workspace/thing"
	put := func(slug, name, typ string, lastSeen time.Time, facts int) {
		if err := s.PutEntity(store.Entity{Slug: slug, Name: name, Type: typ, RepoRefs: []string{repo}, CreatedAt: lastSeen, LastSeen: lastSeen}); err != nil {
			t.Fatal(err)
		}
		for i := range facts {
			if err := s.PutFact(store.Fact{Src: slug, Relation: "status", Value: fmt.Sprintf("v%d", i), Fact: name + " fact " + fmt.Sprint(i), ValidFrom: lastSeen, Confidence: 0.9}); err != nil {
				t.Fatal(err)
			}
		}
	}
	put("aaa-old-concept", "aaa-old-concept", "concept", now.Add(-30*24*time.Hour), 5)
	put("zzz-new-service", "zzz-new-service", "service", now, 2)
	put("mmm-new-concept", "mmm-new-concept", "concept", now, 9)
	put("empty", "empty", "service", now, 0)

	md, err := Orient(s, repo, 4000, now)
	if err != nil {
		t.Fatal(err)
	}
	iNew := strings.Index(md, "zzz-new-service")
	iConcept := strings.Index(md, "mmm-new-concept")
	iOld := strings.Index(md, "aaa-old-concept")
	if iNew < 0 || iConcept < 0 {
		t.Fatalf("orient missing recent entities:\n%s", md)
	}
	if iNew > iConcept {
		t.Errorf("a typed entity should outrank a concept at the same recency:\n%s", md)
	}
	if iOld >= 0 && iOld < iNew {
		t.Errorf("a month-old entity outranked today's work:\n%s", md)
	}
	if strings.Contains(md, "empty") {
		t.Errorf("an entity with no current fact must not appear:\n%s", md)
	}
}

func TestOrientPrefersWhatHappenedInThisRepo(t *testing.T) {
	s := openTemp(t)
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	repo := "/Users/jeff/workspace/thing"
	// An episode that ran here, from an OpenCode session.
	if err := s.PutEpisode(store.Episode{ID: "oc1", Source: "opencode-session", Summary: "local work",
		OccurredAt: now, IngestedAt: now, Cwd: repo, CwdIsRepo: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutEpisode(store.Episode{ID: "cl1", Source: "claude-session", Summary: "elsewhere",
		OccurredAt: now, IngestedAt: now, Cwd: "/Users/jeff/workspace/other", CwdIsRepo: true}); err != nil {
		t.Fatal(err)
	}
	put := func(slug, typ string, refs []string, facts []store.Fact) {
		if err := s.PutEntity(store.Entity{Slug: slug, Name: slug, Type: typ, RepoRefs: refs, CreatedAt: now, LastSeen: now}); err != nil {
			t.Fatal(err)
		}
		for _, f := range facts {
			f.Src, f.ValidFrom, f.Confidence = slug, now, 0.9
			if err := s.PutFact(f); err != nil {
				t.Fatal(err)
			}
		}
	}
	// A global entity claiming many repos, busier and typed, but nothing
	// of it happened here.
	many := []string{repo, "/a", "/b", "/c", "/d"}
	global := make([]store.Fact, 0, 6)
	for i := range 6 {
		global = append(global, store.Fact{Relation: "status", Value: fmt.Sprintf("g%d", i), Fact: fmt.Sprintf("global fact %d", i), Episodes: []string{"cl1"}})
	}
	put("global-thing", "project", many, global)
	// A small local entity whose fact came from the OpenCode session here.
	put("local-thing", "project", []string{repo}, []store.Fact{
		{Relation: "status", Value: "done", Fact: "the opencode session finished the local thing", Episodes: []string{"oc1"}},
	})

	md, err := Orient(s, repo, 2000, now)
	if err != nil {
		t.Fatal(err)
	}
	iLocal, iGlobal := strings.Index(md, "local-thing"), strings.Index(md, "global-thing")
	if iLocal < 0 {
		t.Fatalf("orient omitted this repo's own work:\n%s", md)
	}
	if iGlobal >= 0 && iGlobal < iLocal {
		t.Errorf("a global entity outranked work done here:\n%s", md)
	}
	if !strings.Contains(md, "the opencode session finished the local thing") {
		t.Errorf("orient must surface the fact from the session that ran here:\n%s", md)
	}
}
