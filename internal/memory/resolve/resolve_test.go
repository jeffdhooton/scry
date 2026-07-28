package resolve

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func openTemp(t *testing.T) *store.Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "badger")
	st, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func mustFacts(t *testing.T, st *store.Store, slug string) []store.Fact {
	t.Helper()
	facts, err := st.FactsFrom(slug, true)
	if err != nil {
		t.Fatalf("FactsFrom(%s): %v", slug, err)
	}
	return facts
}

// Rule 1: idempotency.
func TestApply_Idempotent(t *testing.T) {
	st := openTemp(t)
	occurred := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
	res := extract.Result{
		Entities: []extract.Ent{{Name: "loom", Type: "project", Description: "loop engine"}},
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "loom uses deepseek-v4", Confidence: 0.9},
		},
	}

	stats1, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply (first): %v", err)
	}
	if stats1.EntitiesCreated != 2 || stats1.FactsAdded != 1 {
		t.Fatalf("unexpected first-apply stats: %+v", stats1)
	}

	_, entitiesBefore, factsBefore, err := st.Counts()
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}

	stats2, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply (second): %v", err)
	}
	if stats2 != (Stats{}) {
		t.Fatalf("expected zero Stats on idempotent re-apply, got %+v", stats2)
	}

	_, entitiesAfter, factsAfter, err := st.Counts()
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if entitiesAfter != entitiesBefore || factsAfter != factsBefore {
		t.Fatalf("re-apply must not write: entities %d->%d facts %d->%d", entitiesBefore, entitiesAfter, factsBefore, factsAfter)
	}
}

// Rule 2: alias-based resolution.
func TestApply_AliasResolution(t *testing.T) {
	st := openTemp(t)
	occurred := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := st.PutEntity(store.Entity{
		Slug: "hermes-mini", Name: "Hermes Mini", Type: "machine",
		Description: "original desc", Aliases: []string{"the mini"},
		CreatedAt: created, LastSeen: created,
	}); err != nil {
		t.Fatalf("seed PutEntity: %v", err)
	}

	ep := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
	res := extract.Result{
		Entities: []extract.Ent{{Name: "the mini", Type: "machine", Description: "updated desc", Aliases: []string{"jclaws-mac-mini"}}},
	}

	stats, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if stats.EntitiesCreated != 0 || stats.EntitiesUpdated != 1 {
		t.Fatalf("expected 0 created / 1 updated, got %+v", stats)
	}

	got, err := st.GetEntity("hermes-mini")
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	if got.Description != "updated desc" {
		t.Fatalf("description not overwritten: %+v", got)
	}
	if !got.CreatedAt.Equal(created) {
		t.Fatalf("CreatedAt should be kept from existing: got %v want %v", got.CreatedAt, created)
	}
	if !got.LastSeen.Equal(occurred) {
		t.Fatalf("LastSeen should be max(existing, occurred): got %v want %v", got.LastSeen, occurred)
	}
	wantAliases := map[string]bool{"the mini": true, "jclaws-mac-mini": true}
	if len(got.Aliases) != len(wantAliases) {
		t.Fatalf("aliases not unioned correctly: %+v", got.Aliases)
	}
	for _, a := range got.Aliases {
		if !wantAliases[a] {
			t.Fatalf("unexpected alias %q in %+v", a, got.Aliases)
		}
	}
}

// Rule 2: description not overwritten when incoming is empty; RepoRefs union
// only for cwd under the user's home.
func TestApply_EntityMerge_EmptyDescriptionAndRepoRefs(t *testing.T) {
	st := openTemp(t)
	occurred := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := st.PutEntity(store.Entity{
		Slug: "loom", Name: "loom", Type: "project",
		Description: "original desc", CreatedAt: created, LastSeen: created,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ep := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
	res := extract.Result{Entities: []extract.Ent{{Name: "loom", Type: "project", Description: ""}}}

	if _, err := Apply(st, ep, "/Users/jeff/workspace/loom", res, DefaultExclusive); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := st.GetEntity("loom")
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	if got.Description != "original desc" {
		t.Fatalf("description should not be overwritten by empty incoming: %+v", got)
	}
	if len(got.RepoRefs) != 1 || got.RepoRefs[0] != "/Users/jeff/workspace/loom" {
		t.Fatalf("expected RepoRefs union with workspace cwd: %+v", got.RepoRefs)
	}

	// Non-workspace cwd must not be unioned in.
	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: occurred, IngestedAt: occurred}
	if _, err := Apply(st, ep2, "/tmp/scratch", res, DefaultExclusive); err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	got2, err := st.GetEntity("loom")
	if err != nil {
		t.Fatalf("GetEntity 2: %v", err)
	}
	if len(got2.RepoRefs) != 1 {
		t.Fatalf("non-workspace cwd should not be unioned: %+v", got2.RepoRefs)
	}
}

// Rule 3: stub entity creation for fact endpoints never seen as entities.
func TestApply_StubEntityCreation(t *testing.T) {
	st := openTemp(t)
	occurred := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
	res := extract.Result{
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "some-new-tool", Fact: "loom uses some-new-tool", Confidence: 0.5},
		},
	}

	stats, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if stats.EntitiesCreated != 2 {
		t.Fatalf("expected 2 stub entities created, got %+v", stats)
	}

	for _, slug := range []string{"loom", "some-new-tool"} {
		e, err := st.GetEntity(slug)
		if err != nil {
			t.Fatalf("GetEntity(%s): %v", slug, err)
		}
		if e.Type != "concept" {
			t.Fatalf("stub entity %s should be type concept, got %q", slug, e.Type)
		}
		if e.Description != "" {
			t.Fatalf("stub entity %s should have empty description, got %q", slug, e.Description)
		}
		if !e.CreatedAt.Equal(occurred) || !e.LastSeen.Equal(occurred) {
			t.Fatalf("stub entity %s CreatedAt/LastSeen should equal ep.OccurredAt: %+v", slug, e)
		}
	}
}

// Rule 4: merge (same src/relation/dst) across episodes, including episode-ID
// dedupe within a single Apply call.
func TestApply_MergeRule(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: t1, IngestedAt: t1}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "loom uses deepseek-v4", ValidFrom: "2026-01-01", Confidence: 0.7},
		},
	}
	stats1, err := Apply(st, ep1, "", res1, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 1: %v", err)
	}
	if stats1.FactsAdded != 1 {
		t.Fatalf("expected FactsAdded=1, got %+v", stats1)
	}

	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: t2, IngestedAt: t2}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "loom still uses deepseek-v4", ValidFrom: "2026-02-01", Confidence: 0.9},
		},
	}
	stats2, err := Apply(st, ep2, "", res2, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	if stats2.FactsMerged != 1 || stats2.FactsAdded != 0 {
		t.Fatalf("expected FactsMerged=1, FactsAdded=0, got %+v", stats2)
	}

	facts := mustFacts(t, st, "loom")
	if len(facts) != 1 {
		t.Fatalf("expected exactly one fact (no duplicate), got %d: %+v", len(facts), facts)
	}
	f := facts[0]
	if f.Confidence != 0.9 {
		t.Fatalf("expected max confidence 0.9, got %v", f.Confidence)
	}
	if !f.ValidFrom.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected earlier ValidFrom kept, got %v", f.ValidFrom)
	}
	if len(f.Episodes) != 2 {
		t.Fatalf("expected both episode IDs recorded, got %+v", f.Episodes)
	}

	// Episode-ID dedupe: re-processing the same triple twice within a single
	// Apply call (same ep.ID both times) must not duplicate the episode ID.
	ep3 := store.Episode{ID: "ep-3", Source: "manual", SourceRef: "z", OccurredAt: t2, IngestedAt: t2}
	res3 := extract.Result{
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "dup 1", Confidence: 0.5},
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "dup 2", Confidence: 0.6},
		},
	}
	stats3, err := Apply(st, ep3, "", res3, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 3: %v", err)
	}
	if stats3.FactsMerged != 2 {
		t.Fatalf("expected both dup facts to merge, got %+v", stats3)
	}
	facts = mustFacts(t, st, "loom")
	if len(facts) != 1 {
		t.Fatalf("still expected exactly one fact, got %d", len(facts))
	}
	if len(facts[0].Episodes) != 3 {
		t.Fatalf("expected ep-3 appended exactly once (dedupe), got %+v", facts[0].Episodes)
	}
}

// Rule 5: supersedes hint invalidates a matching current fact.
func TestApply_Supersedes(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: t1, IngestedAt: t1}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "book-system", Relation: "replaces", Dst: "authorclaw", Fact: "book-system replaces authorclaw", Confidence: 0.8},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: t2, IngestedAt: t2}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{
				Src: "authorclaw", Relation: "status", Dst: "retired", Fact: "authorclaw is retired", Confidence: 0.9,
				Supersedes: &extract.SupRef{Src: "book-system", Relation: "replaces", Dst: "authorclaw"},
			},
		},
	}
	stats2, err := Apply(st, ep2, "", res2, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	if stats2.FactsInvalidated != 1 {
		t.Fatalf("expected FactsInvalidated=1, got %+v", stats2)
	}

	facts := mustFacts(t, st, "book-system")
	if len(facts) != 1 {
		t.Fatalf("expected the superseded fact still present (invalidated, not deleted): %+v", facts)
	}
	if facts[0].InvalidAt == nil {
		t.Fatalf("expected superseded fact to be invalidated")
	}
	if !facts[0].InvalidAt.Equal(t2) {
		t.Fatalf("expected InvalidAt == ep2.OccurredAt, got %v", facts[0].InvalidAt)
	}
}

// Rule 6: exclusive relation flip.
func TestApply_ExclusiveFlip(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: t1, IngestedAt: t1}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "deployed_on", Dst: "digitalocean", Fact: "deployed on DO", Confidence: 0.9},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: t2, IngestedAt: t2}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "deployed_on", Dst: "jclaws-mac-mini", Fact: "deployed on the mini", Confidence: 0.95},
		},
	}
	stats2, err := Apply(st, ep2, "", res2, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	if stats2.FactsInvalidated != 1 || stats2.FactsAdded != 1 {
		t.Fatalf("expected 1 invalidated + 1 added, got %+v", stats2)
	}

	facts := mustFacts(t, st, "hermes-mini")
	if len(facts) != 2 {
		t.Fatalf("expected old fact retained (invalidated) + new fact, got %d: %+v", len(facts), facts)
	}
	var old, cur *store.Fact
	for i := range facts {
		if facts[i].Dst == "digitalocean" {
			old = &facts[i]
		}
		if facts[i].Dst == "jclaws-mac-mini" {
			cur = &facts[i]
		}
	}
	if old == nil || old.InvalidAt == nil {
		t.Fatalf("expected old deployed_on fact invalidated: %+v", old)
	}
	if !old.InvalidAt.Equal(t2) {
		t.Fatalf("expected InvalidAt == ep2.OccurredAt, got %v", old.InvalidAt)
	}
	if cur == nil || cur.InvalidAt != nil {
		t.Fatalf("expected new fact current: %+v", cur)
	}
}

// Rule 6 (negative): non-exclusive relations coexist rather than flipping.
func TestApply_NonExclusiveCoexist(t *testing.T) {
	st := openTemp(t)
	occurred := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
	res := extract.Result{
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "loom uses deepseek-v4", Confidence: 0.8},
			{Src: "loom", Relation: "uses", Dst: "ollama", Fact: "loom uses ollama", Confidence: 0.7},
		},
	}
	stats, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if stats.FactsAdded != 2 || stats.FactsInvalidated != 0 {
		t.Fatalf("expected both uses facts added, none invalidated: %+v", stats)
	}
	facts := mustFacts(t, st, "loom")
	if len(facts) != 2 {
		t.Fatalf("expected both facts current: %+v", facts)
	}
	for _, f := range facts {
		if f.InvalidAt != nil {
			t.Fatalf("non-exclusive facts should not be invalidated: %+v", f)
		}
	}
}

// Rule 3: valid_from parse fallback across all three cases.
func TestApply_ValidFromParseFallback(t *testing.T) {
	occurred := time.Date(2026, 7, 20, 12, 30, 0, 0, time.UTC)

	cases := []struct {
		name      string
		validFrom string
		want      time.Time
	}{
		{"rfc3339", "2026-05-15T10:00:00Z", time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)},
		{"date-only", "2026-05-15", time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)},
		{"garbage-falls-back", "not-a-date", occurred},
		{"empty-falls-back", "", occurred},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := openTemp(t)
			ep := store.Episode{ID: "ep-" + tc.name, Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
			res := extract.Result{
				Facts: []extract.Fct{
					{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "f", ValidFrom: tc.validFrom, Confidence: 0.5},
				},
			}
			if _, err := Apply(st, ep, "", res, DefaultExclusive); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			facts := mustFacts(t, st, "loom")
			if len(facts) != 1 {
				t.Fatalf("expected one fact, got %d", len(facts))
			}
			if !facts[0].ValidFrom.Equal(tc.want) {
				t.Fatalf("ValidFrom = %v, want %v", facts[0].ValidFrom, tc.want)
			}
		})
	}
}

// Empty-slug names must be skipped (never written as an entity or fact
// endpoint), but that does not block resolution of a fact's other, valid
// endpoint — only the unresolvable endpoint and any fact depending on it are
// skipped.
func TestApply_EmptySlugSkipped(t *testing.T) {
	st := openTemp(t)
	occurred := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
	res := extract.Result{
		Entities: []extract.Ent{{Name: "???", Type: "concept", Description: "junk"}},
		Facts: []extract.Fct{
			{Src: "???", Relation: "uses", Dst: "deepseek-v4", Fact: "junk fact", Confidence: 0.5},
		},
	}
	stats, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// The "???" entity itself is skipped, and the fact (whose Src is also
	// "???") is never added — but its valid Dst ("deepseek-v4") still gets a
	// concept stub, since that endpoint resolves fine on its own.
	if stats.EntitiesCreated != 1 || stats.FactsAdded != 0 {
		t.Fatalf("expected only the valid dst stub created, no fact added, got %+v", stats)
	}
	if _, err := st.GetEntity(""); err == nil {
		t.Fatalf("expected no entity written under an empty slug")
	}
	if _, err := st.GetEntity("deepseek-v4"); err != nil {
		t.Fatalf("expected stub entity for the valid dst: %v", err)
	}
	_, entities, facts, err := st.Counts()
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if entities != 1 {
		t.Fatalf("expected exactly one entity written, got entities=%d", entities)
	}
	if facts != 0 {
		t.Fatalf("expected no facts written, got facts=%d", facts)
	}
}
