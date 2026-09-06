package store

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "badger")
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func putTestEntities(t *testing.T, s *Store, slugs ...string) {
	t.Helper()
	for _, slug := range slugs {
		if _, err := s.GetEntity(slug); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			t.Fatal(err)
		}
		if err := s.PutEntity(Entity{Slug: slug, Name: slug, Type: "concept"}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpenCloseAndSchemaRefusal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "badger")

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.PutEpisode(Episode{ID: "e1", Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}); err != nil {
		t.Fatalf("PutEpisode: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Simulate a stale schema version written by an older build.
	raw, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	if err := raw.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("meta:schema_version"), []byte("999"))
	}); err != nil {
		t.Fatalf("raw write: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("raw close: %v", err)
	}

	s2, err := Open(dir)
	if s2 != nil {
		_ = s2.Close()
		t.Fatal("incompatible store was opened")
	}
	if !errors.Is(err, ErrSchemaMismatch) {
		t.Fatalf("wanted schema refusal, got %v", err)
	}
	raw, err = badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if err := raw.View(func(txn *badger.Txn) error { _, err := txn.Get([]byte("ep:e1")); return err }); err != nil {
		t.Fatalf("schema refusal lost the original episode: %v", err)
	}
}

func TestEpisodePutHasGet(t *testing.T) {
	s := openTemp(t)

	has, err := s.HasEpisode("e1")
	if err != nil {
		t.Fatalf("HasEpisode: %v", err)
	}
	if has {
		t.Fatalf("expected no episode before put")
	}

	occurred := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	ingested := time.Date(2026, 7, 1, 12, 5, 0, 0, time.UTC)
	e := Episode{
		ID:         "e1",
		Source:     "claude-session",
		SourceRef:  "/path/file.jsonl#L120-L340",
		Summary:    "did a thing",
		OccurredAt: occurred,
		IngestedAt: ingested,
	}
	if err := s.PutEpisode(e); err != nil {
		t.Fatalf("PutEpisode: %v", err)
	}

	has, err = s.HasEpisode("e1")
	if err != nil {
		t.Fatalf("HasEpisode: %v", err)
	}
	if !has {
		t.Fatalf("expected episode after put")
	}

	got, err := s.GetEpisode("e1")
	if err != nil {
		t.Fatalf("GetEpisode: %v", err)
	}
	if got != e {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, e)
	}

	if _, err := s.GetEpisode("missing"); err != ErrNotFound {
		t.Fatalf("GetEpisode missing: want ErrNotFound, got %v", err)
	}
}

func TestAtomicWriteCommitsOrRollsBackAsOneUnit(t *testing.T) {
	s := openTemp(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	episode := Episode{ID: "atomic-episode", Source: "manual", SourceRef: "test", OccurredAt: now, IngestedAt: now}
	alpha := Entity{Slug: "alpha", Name: "Alpha", Type: "project"}
	beta := Entity{Slug: "beta", Name: "Beta", Type: "service"}
	fact := Fact{Src: alpha.Slug, Relation: "uses", Dst: beta.Slug, Fact: "Alpha uses Beta", ValidFrom: now, Episodes: []string{episode.ID}}

	var observed []Event
	s.SetObserver(func(event Event) { observed = append(observed, event) })
	injected := errors.New("injected apply failure")
	err := s.AtomicWrite(func(tx *Store) error {
		if err := tx.PutEntity(alpha); err != nil {
			return err
		}
		if err := tx.PutEntity(beta); err != nil {
			return err
		}
		if got, err := tx.GetEntity(alpha.Slug); err != nil || got.Name != alpha.Name {
			t.Fatalf("transaction did not read its entity write: got=%+v err=%v", got, err)
		}
		if owner, found, err := tx.ResolveAlias(alpha.Name); err != nil || !found || owner != alpha.Slug {
			t.Fatalf("transaction did not read its alias write: owner=%q found=%v err=%v", owner, found, err)
		}
		if err := tx.PutFact(fact); err != nil {
			return err
		}
		if facts, err := tx.FactsFrom(alpha.Slug, true); err != nil || len(facts) != 1 || facts[0].Dst != beta.Slug {
			t.Fatalf("transaction did not read its fact write: facts=%+v err=%v", facts, err)
		}
		if err := tx.PutEpisode(episode); err != nil {
			return err
		}
		if len(observed) != 0 {
			t.Fatalf("observer saw uncommitted events: %+v", observed)
		}
		return injected
	})
	if !errors.Is(err, injected) {
		t.Fatalf("AtomicWrite error = %v, want injected failure", err)
	}
	for _, slug := range []string{alpha.Slug, beta.Slug} {
		if _, err := s.GetEntity(slug); !errors.Is(err, ErrNotFound) {
			t.Fatalf("rolled-back entity %q remains: %v", slug, err)
		}
	}
	if owner, found, err := s.ResolveAlias(alpha.Name); err != nil || found {
		t.Fatalf("rolled-back alias remains: owner=%q found=%v err=%v", owner, found, err)
	}
	if facts, err := s.FactsFrom(alpha.Slug, true); err != nil || len(facts) != 0 {
		t.Fatalf("rolled-back fact remains: facts=%+v err=%v", facts, err)
	}
	if has, err := s.HasEpisode(episode.ID); err != nil || has {
		t.Fatalf("rolled-back episode remains: has=%v err=%v", has, err)
	}
	if len(observed) != 0 {
		t.Fatalf("observer saw rolled-back events: %+v", observed)
	}

	if err := s.AtomicWrite(func(tx *Store) error {
		if err := tx.PutEntity(alpha); err != nil {
			return err
		}
		if err := tx.PutEntity(beta); err != nil {
			return err
		}
		if err := tx.PutFact(fact); err != nil {
			return err
		}
		if err := tx.PutEpisode(episode); err != nil {
			return err
		}
		if len(observed) != 0 {
			t.Fatalf("observer saw events before commit: %+v", observed)
		}
		return nil
	}); err != nil {
		t.Fatalf("AtomicWrite commit: %v", err)
	}
	if len(observed) != 4 {
		t.Fatalf("observer events after commit = %d, want 4: %+v", len(observed), observed)
	}
}

func TestAllEpisodes(t *testing.T) {
	s := openTemp(t)
	now := time.Now()

	episodes := []Episode{
		{ID: "e2", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		{ID: "e1", Source: "claude-session", SourceRef: "/path/file.jsonl#L1-L10", OccurredAt: now, IngestedAt: now},
	}
	for _, e := range episodes {
		if err := s.PutEpisode(e); err != nil {
			t.Fatalf("PutEpisode(%s): %v", e.ID, err)
		}
	}

	got, err := s.AllEpisodes()
	if err != nil {
		t.Fatalf("AllEpisodes: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 episodes, got %d", len(got))
	}
	if got[0].ID != "e1" || got[1].ID != "e2" {
		t.Fatalf("AllEpisodes not sorted by ID: %+v", got)
	}
}

func TestEntityPutAndAliasResolution(t *testing.T) {
	s := openTemp(t)

	e := Entity{
		Slug:        "hermes-mini",
		Name:        "Hermes mini",
		Type:        "machine",
		Description: "the mac mini running Hermes",
		Aliases:     []string{"the mini"},
		CreatedAt:   time.Now(),
		LastSeen:    time.Now(),
	}
	if err := s.PutEntity(e); err != nil {
		t.Fatalf("PutEntity: %v", err)
	}

	got, err := s.GetEntity("hermes-mini")
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	if got.Name != e.Name {
		t.Fatalf("GetEntity mismatch: %+v", got)
	}

	slug, ok, err := s.ResolveAlias("Hermes Mini")
	if err != nil {
		t.Fatalf("ResolveAlias(name): %v", err)
	}
	if !ok || slug != "hermes-mini" {
		t.Fatalf("ResolveAlias(name) = %q, %v; want hermes-mini, true", slug, ok)
	}

	slug, ok, err = s.ResolveAlias("the mini")
	if err != nil {
		t.Fatalf("ResolveAlias(alias): %v", err)
	}
	if !ok || slug != "hermes-mini" {
		t.Fatalf("ResolveAlias(alias) = %q, %v; want hermes-mini, true", slug, ok)
	}

	_, ok, err = s.ResolveAlias("nonexistent thing")
	if err != nil {
		t.Fatalf("ResolveAlias(missing): %v", err)
	}
	if ok {
		t.Fatalf("expected no resolution for unknown alias")
	}
}

func TestPutEntityAndFactRejectDelimiterAmbiguousSlugs(t *testing.T) {
	s := openTemp(t)
	if err := s.PutEntity(Entity{Slug: "bad:slug", Name: "Bad slug", Type: "concept"}); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("PutEntity error = %v, want ErrInvalidSlug", err)
	}
	if _, err := s.GetEntity("bad:slug"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid entity was partially written: %v", err)
	}
	if owner, found, err := s.ResolveAlias("Bad slug"); err != nil || found {
		t.Fatalf("invalid entity claimed alias: owner=%q found=%v err=%v", owner, found, err)
	}
	if err := s.PutFact(Fact{Src: "bad:slug", Relation: "status", Value: "ready", ValidFrom: time.Now()}); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("PutFact error = %v, want ErrInvalidSlug", err)
	}
}

func TestPutEntityCannotOverwriteANameAtACollidingSlug(t *testing.T) {
	s := openTemp(t)
	original := Entity{Slug: "ready-after-fixes", Name: "Unrelated Owner", Type: "concept", Description: "keep"}
	if err := s.PutEntity(original); err != nil {
		t.Fatal(err)
	}
	replacement := original
	replacement.Name = "READY_AFTER_FIXES"
	replacement.Description = "must not persist"
	if err := s.PutEntity(replacement); !errors.Is(err, ErrAliasClaimed) {
		t.Fatalf("colliding name update error = %v, want ErrAliasClaimed", err)
	}
	got, err := s.GetEntity(original.Slug)
	if err != nil || got.Name != original.Name || got.Description != original.Description {
		t.Fatalf("colliding write mutated owner: got=%+v err=%v", got, err)
	}
}

func TestPutEntityPrunesStaleAliases(t *testing.T) {
	s := openTemp(t)
	now := time.Now()

	if err := s.PutEntity(Entity{Slug: "x", Name: "X", Type: "concept",
		Aliases: []string{"a", "b"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity initial: %v", err)
	}

	if err := s.PutEntity(Entity{Slug: "x", Name: "X", Type: "concept",
		Aliases: []string{"b", "c"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity update: %v", err)
	}

	if _, ok, err := s.ResolveAlias("a"); err != nil {
		t.Fatalf("ResolveAlias(a): %v", err)
	} else if ok {
		t.Fatalf("stale alias %q should no longer resolve", "a")
	}

	slug, ok, err := s.ResolveAlias("b")
	if err != nil {
		t.Fatalf("ResolveAlias(b): %v", err)
	}
	if !ok || slug != "x" {
		t.Fatalf("ResolveAlias(b) = %q, %v; want x, true", slug, ok)
	}

	slug, ok, err = s.ResolveAlias("c")
	if err != nil {
		t.Fatalf("ResolveAlias(c): %v", err)
	}
	if !ok || slug != "x" {
		t.Fatalf("ResolveAlias(c) = %q, %v; want x, true", slug, ok)
	}
}

func TestPutEntityRequiresExplicitAliasTransfer(t *testing.T) {
	s := openTemp(t)
	now := time.Now()

	// x owns "shared".
	if err := s.PutEntity(Entity{Slug: "x", Name: "X", Type: "concept",
		Aliases: []string{"shared"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity x: %v", err)
	}

	// An ordinary write cannot move x's claim to y. The transaction also
	// cannot leave behind the partly-written y entity.
	err := s.PutEntity(Entity{Slug: "y", Name: "Y", Type: "concept",
		Aliases: []string{"shared"}, CreatedAt: now, LastSeen: now})
	if !errors.Is(err, ErrAliasClaimed) {
		t.Fatalf("PutEntity y error = %v; want ErrAliasClaimed", err)
	}
	if _, err := s.GetEntity("y"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("conflicting PutEntity wrote y: %v", err)
	}

	slug, ok, err := s.ResolveAlias("shared")
	if err != nil {
		t.Fatalf("ResolveAlias(shared) after y claims it: %v", err)
	}
	if !ok || slug != "x" {
		t.Fatalf("ResolveAlias(shared) = %q, %v; want x, true", slug, ok)
	}

	// The explicit transfer path can move it after y exists.
	if err := s.PutEntity(Entity{Slug: "y", Name: "Y", Type: "concept",
		CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity y without conflicting alias: %v", err)
	}
	if err := s.ClaimAlias("shared", "y"); err != nil {
		t.Fatalf("ClaimAlias(shared, y): %v", err)
	}

	// Re-putting x while its legacy entity record still lists "shared" is
	// allowed, but must preserve y's claim. This is the shape found in the
	// live store and lets unrelated metadata repairs continue.
	if err := s.PutEntity(Entity{Slug: "x", Name: "X", Type: "concept",
		Aliases: []string{"shared"}, Description: "updated", CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity legacy x update: %v", err)
	}

	slug, ok, err = s.ResolveAlias("shared")
	if err != nil {
		t.Fatalf("ResolveAlias(shared) after legacy update: %v", err)
	}
	if !ok || slug != "y" {
		t.Fatalf("legacy update moved shared: got %q, %v; want y, true", slug, ok)
	}

	// Dropping x's stale listing likewise must not delete y's mapping.
	if err := s.PutEntity(Entity{Slug: "x", Name: "X", Type: "concept",
		Aliases: nil, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity x again: %v", err)
	}

	slug, ok, err = s.ResolveAlias("shared")
	if err != nil {
		t.Fatalf("ResolveAlias(shared) after re-put x: %v", err)
	}
	if !ok || slug != "y" {
		t.Fatalf("re-putting x must not clobber y's alias: got %q, %v; want y, true", slug, ok)
	}
}

func TestPutEntityRejectsNewAliasOnExistingEntity(t *testing.T) {
	s := openTemp(t)
	now := time.Now()
	if err := s.PutEntity(Entity{Slug: "x", Name: "X", Type: "concept", CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutEntity(Entity{Slug: "y", Name: "Y", Type: "concept", Aliases: []string{"shared"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatal(err)
	}

	err := s.PutEntity(Entity{Slug: "x", Name: "X", Type: "concept", Aliases: []string{"shared"}, Description: "must not persist", CreatedAt: now, LastSeen: now})
	if !errors.Is(err, ErrAliasClaimed) {
		t.Fatalf("PutEntity x error = %v; want ErrAliasClaimed", err)
	}
	x, err := s.GetEntity("x")
	if err != nil {
		t.Fatal(err)
	}
	if x.Description != "" || len(x.Aliases) != 0 {
		t.Fatalf("conflicting update was not atomic: %+v", x)
	}
}

func TestEntitiesFullScan(t *testing.T) {
	s := openTemp(t)

	now := time.Now()
	entities := []Entity{
		{Slug: "zeta", Name: "Zeta", Type: "concept", CreatedAt: now, LastSeen: now},
		{Slug: "alpha", Name: "Alpha", Type: "concept", CreatedAt: now, LastSeen: now},
	}
	for _, e := range entities {
		if err := s.PutEntity(e); err != nil {
			t.Fatalf("PutEntity(%s): %v", e.Slug, err)
		}
	}

	got, err := s.Entities()
	if err != nil {
		t.Fatalf("Entities: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entities, got %d", len(got))
	}
	if got[0].Slug != "alpha" || got[1].Slug != "zeta" {
		t.Fatalf("Entities not sorted by slug: %+v", got)
	}
}

func TestEntitiesByRepoRef(t *testing.T) {
	s := openTemp(t)
	now := time.Now()

	if err := s.PutEntity(Entity{Slug: "book-system", Name: "book-system", Type: "project",
		RepoRefs: []string{"/Users/jeff/workspace/book-system"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity: %v", err)
	}
	if err := s.PutEntity(Entity{Slug: "loom", Name: "loom", Type: "project",
		RepoRefs: []string{"/Users/jeff/workspace/loom"}, CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity: %v", err)
	}

	got, err := s.EntitiesByRepoRef("/Users/jeff/workspace/book-system")
	if err != nil {
		t.Fatalf("EntitiesByRepoRef: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "book-system" {
		t.Fatalf("EntitiesByRepoRef mismatch: %+v", got)
	}

	got, err = s.EntitiesByRepoRef("/no/such/path")
	if err != nil {
		t.Fatalf("EntitiesByRepoRef(none): %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %+v", got)
	}
}

func TestFactPutAndFactsFromFiltersInvalidated(t *testing.T) {
	s := openTemp(t)
	putTestEntities(t, s, "book-system", "postgres", "old-host")
	v1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	valid := Fact{Src: "book-system", Relation: "uses", Dst: "postgres", Fact: "book-system uses postgres",
		ValidFrom: v1, Confidence: 0.8, Episodes: []string{"e1"}}
	if err := s.PutFact(valid); err != nil {
		t.Fatalf("PutFact: %v", err)
	}

	invalidAt := v1.AddDate(0, 0, 10)
	invalidated := Fact{Src: "book-system", Relation: "deployed_on", Dst: "old-host", Fact: "book-system ran on old-host",
		ValidFrom: v1, InvalidAt: &invalidAt, Confidence: 0.7, Episodes: []string{"e2"}}
	if err := s.PutFact(invalidated); err != nil {
		t.Fatalf("PutFact: %v", err)
	}

	cur, err := s.FactsFrom("book-system", false)
	if err != nil {
		t.Fatalf("FactsFrom(false): %v", err)
	}
	if len(cur) != 1 || cur[0].Relation != "uses" {
		t.Fatalf("want only the valid fact, got %+v", cur)
	}

	all, err := s.FactsFrom("book-system", true)
	if err != nil {
		t.Fatalf("FactsFrom(true): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want both facts including invalidated, got %d: %+v", len(all), all)
	}
}

func TestFactSameKeyDifferentValidFromCoexist(t *testing.T) {
	s := openTemp(t)
	putTestEntities(t, s, "book-system", "hermes-mini")
	v1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	f1 := Fact{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "first stint",
		ValidFrom: v1, Confidence: 0.9, Episodes: []string{"e1"}}
	f2 := Fact{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "second stint",
		ValidFrom: v2, Confidence: 0.9, Episodes: []string{"e2"}}
	if err := s.PutFact(f1); err != nil {
		t.Fatalf("PutFact f1: %v", err)
	}
	if err := s.PutFact(f2); err != nil {
		t.Fatalf("PutFact f2: %v", err)
	}

	got, err := s.FactsFrom("book-system", true)
	if err != nil {
		t.Fatalf("FactsFrom: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 coexisting facts with different ValidFrom, got %d: %+v", len(got), got)
	}
}

func TestFactsAboutUsesReverseIndex(t *testing.T) {
	s := openTemp(t)
	putTestEntities(t, s, "book-system", "hermes-mini")
	v1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	f := Fact{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "book-system runs on the mini",
		ValidFrom: v1, Confidence: 0.9, Episodes: []string{"e1"}}
	if err := s.PutFact(f); err != nil {
		t.Fatalf("PutFact: %v", err)
	}

	about, err := s.FactsAbout("hermes-mini", true)
	if err != nil {
		t.Fatalf("FactsAbout(dst): %v", err)
	}
	if len(about) != 1 || about[0].Dst != "hermes-mini" {
		t.Fatalf("FactsAbout should find fact by dst via adj index: %+v", about)
	}

	about, err = s.FactsAbout("book-system", true)
	if err != nil {
		t.Fatalf("FactsAbout(src): %v", err)
	}
	if len(about) != 1 || about[0].Src != "book-system" {
		t.Fatalf("FactsAbout should find fact by src too: %+v", about)
	}
}

func TestAllFacts(t *testing.T) {
	s := openTemp(t)
	putTestEntities(t, s, "book-system", "postgres", "old-host")
	v1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	current := Fact{Src: "book-system", Relation: "uses", Dst: "postgres", Fact: "book-system uses postgres",
		ValidFrom: v1, Confidence: 0.8, Episodes: []string{"e1"}}
	if err := s.PutFact(current); err != nil {
		t.Fatalf("PutFact current: %v", err)
	}

	invalidAt := v1.AddDate(0, 0, 10)
	superseded := Fact{Src: "book-system", Relation: "deployed_on", Dst: "old-host", Fact: "book-system ran on old-host",
		ValidFrom: v1, InvalidAt: &invalidAt, Confidence: 0.7, Episodes: []string{"e2"}}
	if err := s.PutFact(superseded); err != nil {
		t.Fatalf("PutFact superseded: %v", err)
	}

	got, err := s.AllFacts()
	if err != nil {
		t.Fatalf("AllFacts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("AllFacts should include invalidated facts too, want 2, got %d: %+v", len(got), got)
	}
	var sawCurrent, sawSuperseded bool
	for _, f := range got {
		if f.Relation == "uses" && f.InvalidAt == nil {
			sawCurrent = true
		}
		if f.Relation == "deployed_on" && f.InvalidAt != nil {
			sawSuperseded = true
		}
	}
	if !sawCurrent || !sawSuperseded {
		t.Fatalf("AllFacts missing expected entries: %+v", got)
	}
}

func TestFactInvalidation(t *testing.T) {
	s := openTemp(t)
	putTestEntities(t, s, "book-system", "hermes-mini")
	v1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f := Fact{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini",
		Fact: "book-system runs on the mini", ValidFrom: v1, Confidence: 0.9, Episodes: []string{"e1"}}
	if err := s.PutFact(f); err != nil {
		t.Fatal(err)
	}
	at := v1.AddDate(0, 1, 0)
	if err := s.InvalidateFact("book-system", "deployed_on", "hermes-mini", v1, at); err != nil {
		t.Fatal(err)
	}
	cur, _ := s.FactsFrom("book-system", false)
	if len(cur) != 0 {
		t.Fatalf("want 0 current facts, got %d", len(cur))
	}
	all, _ := s.FactsFrom("book-system", true)
	if len(all) != 1 || all[0].InvalidAt == nil {
		t.Fatalf("invalidated fact must survive: %+v", all)
	}
}

func TestDeleteFact(t *testing.T) {
	s := openTemp(t)
	putTestEntities(t, s, "book-system", "hermes-mini")
	v1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f := Fact{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini",
		Fact: "book-system runs on the mini", ValidFrom: v1, Confidence: 0.9, Episodes: []string{"e1"}}
	if err := s.PutFact(f); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteFact("book-system", "deployed_on", "hermes-mini", v1); err != nil {
		t.Fatalf("DeleteFact: %v", err)
	}

	all, err := s.FactsFrom("book-system", true)
	if err != nil {
		t.Fatalf("FactsFrom: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("want 0 facts after delete, got %d: %+v", len(all), all)
	}

	about, err := s.FactsAbout("hermes-mini", true)
	if err != nil {
		t.Fatalf("FactsAbout: %v", err)
	}
	if len(about) != 0 {
		t.Fatalf("want the adj: mirror deleted too, got %d: %+v", len(about), about)
	}

	if err := s.DeleteFact("book-system", "deployed_on", "hermes-mini", v1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound deleting an already-absent fact, got %v", err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	s := openTemp(t)

	_, ok, err := s.GetCursor("/path/to/session.jsonl")
	if err != nil {
		t.Fatalf("GetCursor(missing): %v", err)
	}
	if ok {
		t.Fatalf("expected no cursor before put")
	}

	c := Cursor{
		Path:           "/path/to/session.jsonl",
		Size:           4096,
		ModTime:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		ProcessedBytes: 2048,
	}
	if err := s.PutCursor(c); err != nil {
		t.Fatalf("PutCursor: %v", err)
	}

	got, ok, err := s.GetCursor(c.Path)
	if err != nil {
		t.Fatalf("GetCursor: %v", err)
	}
	if !ok || got != c {
		t.Fatalf("cursor round trip mismatch: ok=%v got=%+v want=%+v", ok, got, c)
	}

	all, err := s.Cursors()
	if err != nil {
		t.Fatalf("Cursors: %v", err)
	}
	if len(all) != 1 || all[0] != c {
		t.Fatalf("Cursors mismatch: %+v", all)
	}
}

func TestCounts(t *testing.T) {
	s := openTemp(t)
	now := time.Now()

	if err := s.PutEpisode(Episode{ID: "e1", Source: "manual", SourceRef: "x", OccurredAt: now, IngestedAt: now}); err != nil {
		t.Fatalf("PutEpisode: %v", err)
	}
	if err := s.PutEntity(Entity{Slug: "a", Name: "A", Type: "concept", CreatedAt: now, LastSeen: now}); err != nil {
		t.Fatalf("PutEntity: %v", err)
	}
	if err := s.PutFact(Fact{Src: "a", Relation: "rel", Value: "b", Fact: "f", ValidFrom: now, Confidence: 1, Episodes: []string{"e1"}}); err != nil {
		t.Fatalf("PutFact: %v", err)
	}

	episodes, entities, facts, err := s.Counts()
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if episodes != 1 || entities != 1 || facts != 1 {
		t.Fatalf("Counts = %d,%d,%d; want 1,1,1", episodes, entities, facts)
	}
}

func TestNormalizeAndSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Hermes Mini", "hermes-mini"},
		{"the_mini  thing", "the-mini-thing"},
	}
	for _, c := range cases {
		if got := Normalize(c.in); got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	if got := Slugify("Hermes-Ops LeadGen!"); got != "hermes-ops-leadgen" {
		t.Errorf("Slugify(%q) = %q, want %q", "Hermes-Ops LeadGen!", got, "hermes-ops-leadgen")
	}
}

// ~/.scry/backups filled with 44-byte files named memory-pre-restore-*,
// each of which had reported success. A backup that captures nothing from
// a store that holds something is a failure, because the caller is about
// to migrate or wipe on the strength of it.
func TestBackupRefusesToClaimItSavedAnEmptyFile(t *testing.T) {
	t.Run("an empty store backs up to a header and that is fine", func(t *testing.T) {
		st := openTemp(t)
		var buf bytes.Buffer
		n, err := st.Backup(&buf)
		if err != nil {
			t.Fatalf("backing up an empty store must succeed: %v", err)
		}
		if n > headerOnlyBackup {
			t.Errorf("an empty store wrote %d bytes", n)
		}
	})

	t.Run("a store with contents must write more than a header", func(t *testing.T) {
		st := openTemp(t)
		if err := st.PutEntity(Entity{Slug: "scry", Name: "scry", Type: "project"}); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		n, err := st.Backup(&buf)
		if err != nil {
			t.Fatalf("Backup: %v", err)
		}
		if n <= headerOnlyBackup {
			t.Fatalf("a store with an entity wrote only %d bytes", n)
		}
	})

	t.Run("a writer that swallows everything is reported as a failure", func(t *testing.T) {
		st := openTemp(t)
		if err := st.PutEntity(Entity{Slug: "scry", Name: "scry", Type: "project"}); err != nil {
			t.Fatal(err)
		}
		_, err := st.Backup(swallowWriter{})
		if err == nil {
			t.Fatal("a backup that captured nothing reported success")
		}
		if !strings.Contains(err.Error(), "nothing was captured") {
			t.Errorf("error should say what went wrong, got %v", err)
		}
	})
}

// swallowWriter accepts every byte and keeps none, standing in for the
// conditions that produced the 44-byte files.
type swallowWriter struct{}

func (swallowWriter) Write(p []byte) (int, error) { return 0, nil }
