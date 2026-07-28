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

// Rule 4 (earlier ValidFrom): a merge whose incoming ValidFrom genuinely
// predates the stored fact's (e.g. a backfilled episode) relocates the
// record to the earlier ValidFrom instead of discarding it, and the result
// is still exactly one fact — never a duplicate under the old key.
func TestApply_MergeRule_EarlierValidFromRelocates(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: t3, IngestedAt: t3}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "loom uses deepseek-v4", ValidFrom: "2026-03-01", Confidence: 0.6},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	// Backfill: a later-processed episode restating the same fact but with
	// an earlier real-world valid_from.
	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: t3, IngestedAt: t3}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{Src: "loom", Relation: "uses", Dst: "deepseek-v4", Fact: "loom uses deepseek-v4 (backfilled)", ValidFrom: "2026-01-01", Confidence: 0.8},
		},
	}
	stats2, err := Apply(st, ep2, "", res2, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	if stats2.FactsMerged != 1 || stats2.FactsAdded != 0 {
		t.Fatalf("expected a merge (relocated), not a fresh add: %+v", stats2)
	}

	facts := mustFacts(t, st, "loom")
	if len(facts) != 1 {
		t.Fatalf("expected exactly one fact (relocated, not duplicated), got %d: %+v", len(facts), facts)
	}
	f := facts[0]
	if !f.ValidFrom.Equal(t1) {
		t.Fatalf("expected ValidFrom relocated to the earlier date, got %v", f.ValidFrom)
	}
	if f.Confidence != 0.8 {
		t.Fatalf("expected max confidence 0.8, got %v", f.Confidence)
	}
	if len(f.Episodes) != 2 || !containsString(f.Episodes, "ep-1") || !containsString(f.Episodes, "ep-2") {
		t.Fatalf("expected merged provenance from both episodes, got %+v", f.Episodes)
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

// Rule 5 (regression guard): a fact that merges onto an existing current
// triple in Phase A must still apply its own supersedes hint in Phase B —
// the hint targets a different triple and must not be silently dropped
// just because this fact's own triple already merged.
func TestApply_Supersedes_OnMergedFact(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	ep0 := store.Episode{ID: "ep-0", Source: "manual", SourceRef: "seed", OccurredAt: t1, IngestedAt: t1}
	res0 := extract.Result{
		Facts: []extract.Fct{
			{Src: "a", Relation: "uses", Dst: "b", Fact: "a uses b", Confidence: 0.7},
			{Src: "a", Relation: "uses", Dst: "c", Fact: "a uses c", Confidence: 0.6},
		},
	}
	if _, err := Apply(st, ep0, "", res0, DefaultExclusive); err != nil {
		t.Fatalf("Apply seed: %v", err)
	}

	epX := store.Episode{ID: "ep-x", Source: "manual", SourceRef: "z", OccurredAt: t2, IngestedAt: t2}
	resX := extract.Result{
		Facts: []extract.Fct{
			{
				Src: "a", Relation: "uses", Dst: "b", Fact: "a still uses b", Confidence: 0.9,
				Supersedes: &extract.SupRef{Src: "a", Relation: "uses", Dst: "c"},
			},
		},
	}
	stats, err := Apply(st, epX, "", resX, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if stats.FactsMerged != 1 || stats.FactsInvalidated != 1 || stats.FactsAdded != 0 {
		t.Fatalf("expected merge + supersedes-invalidation, no add: %+v", stats)
	}

	facts := mustFacts(t, st, "a")
	if len(facts) != 2 {
		t.Fatalf("expected exactly two fa: records, got %d: %+v", len(facts), facts)
	}
	var b, c *store.Fact
	for i := range facts {
		switch facts[i].Dst {
		case "b":
			b = &facts[i]
		case "c":
			c = &facts[i]
		}
	}
	if b == nil || b.InvalidAt != nil {
		t.Fatalf("expected (a,uses,b) current (merged, not invalidated): %+v", b)
	}
	if len(b.Episodes) != 2 || !containsString(b.Episodes, "ep-0") || !containsString(b.Episodes, "ep-x") {
		t.Fatalf("expected (a,uses,b) merged provenance from both episodes, got %+v", b.Episodes)
	}
	if b.Confidence != 0.9 {
		t.Fatalf("expected (a,uses,b) confidence maxed to 0.9, got %v", b.Confidence)
	}
	if c == nil || c.InvalidAt == nil {
		t.Fatalf("expected (a,uses,c) invalidated via the merged fact's supersedes hint: %+v", c)
	}
	if !c.InvalidAt.Equal(t2) {
		t.Fatalf("expected (a,uses,c) InvalidAt == epX.OccurredAt, got %v", c.InvalidAt)
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

// Rule 6 (order independence): when one episode contains both a
// restatement of the existing current fact and a flip to a new dst, the
// final state must not depend on which comes first in the slice — the
// restatement always merges (Phase A) before the flip's invalidation
// (Phase B) runs, so the old fact is invalidated carrying merged
// provenance rather than being invalidated-then-recreated from scratch.
func TestApply_ExclusiveFlip_OrderIndependent(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	restatement := extract.Fct{Src: "hermes-mini", Relation: "deployed_on", Dst: "digitalocean", Fact: "still on DO", Confidence: 0.85}
	flip := extract.Fct{Src: "hermes-mini", Relation: "deployed_on", Dst: "jclaws-mac-mini", Fact: "moved to the mini", Confidence: 0.95}

	orderings := map[string][]extract.Fct{
		"restatement_then_flip": {restatement, flip},
		"flip_then_restatement": {flip, restatement},
	}

	for name, facts := range orderings {
		t.Run(name, func(t *testing.T) {
			st := openTemp(t)
			ep0 := store.Episode{ID: "ep-0", Source: "manual", SourceRef: "seed", OccurredAt: t1, IngestedAt: t1}
			res0 := extract.Result{
				Facts: []extract.Fct{
					{Src: "hermes-mini", Relation: "deployed_on", Dst: "digitalocean", Fact: "deployed on DO", Confidence: 0.9},
				},
			}
			if _, err := Apply(st, ep0, "", res0, DefaultExclusive); err != nil {
				t.Fatalf("Apply seed: %v", err)
			}

			epX := store.Episode{ID: "ep-x", Source: "manual", SourceRef: "z", OccurredAt: t2, IngestedAt: t2}
			resX := extract.Result{Facts: facts}
			stats, err := Apply(st, epX, "", resX, DefaultExclusive)
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if stats.FactsMerged != 1 || stats.FactsInvalidated != 1 || stats.FactsAdded != 1 {
				t.Fatalf("expected 1 merged + 1 invalidated + 1 added regardless of order, got %+v", stats)
			}

			all := mustFacts(t, st, "hermes-mini")
			if len(all) != 2 {
				t.Fatalf("expected exactly two fa: records total, got %d: %+v", len(all), all)
			}

			var old, cur *store.Fact
			for i := range all {
				switch all[i].Dst {
				case "digitalocean":
					old = &all[i]
				case "jclaws-mac-mini":
					cur = &all[i]
				}
			}
			if old == nil || old.InvalidAt == nil {
				t.Fatalf("expected digitalocean fact invalidated: %+v", old)
			}
			if !old.InvalidAt.Equal(t2) {
				t.Fatalf("expected InvalidAt == epX.OccurredAt, got %v", old.InvalidAt)
			}
			if len(old.Episodes) != 2 || !containsString(old.Episodes, "ep-0") || !containsString(old.Episodes, "ep-x") {
				t.Fatalf("expected digitalocean fact to carry merged provenance from both episodes, got %+v", old.Episodes)
			}
			if old.Confidence != 0.9 {
				t.Fatalf("expected max confidence 0.9 preserved on the invalidated fact, got %v", old.Confidence)
			}
			if cur == nil || cur.InvalidAt != nil {
				t.Fatalf("expected jclaws-mac-mini fact current: %+v", cur)
			}
			if len(cur.Episodes) != 1 || cur.Episodes[0] != "ep-x" {
				t.Fatalf("expected new fact's provenance to be just this episode, got %+v", cur.Episodes)
			}
		})
	}
}

// Guard: Rules 5/6 must never invalidate a fact before its own ValidFrom —
// if ep.OccurredAt predates it (e.g. a backfilled episode processed after a
// later-dated fact already exists), InvalidAt is clamped to the fact's
// ValidFrom instead.
func TestApply_InvalidAtClampedToValidFrom(t *testing.T) {
	st := openTemp(t)
	validFrom := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	backfillOccurred := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: validFrom, IngestedAt: validFrom}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "deployed_on", Dst: "digitalocean", Fact: "deployed on DO", ValidFrom: "2026-05-01", Confidence: 0.9},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	// A backfilled episode, with an OccurredAt earlier than the existing
	// fact's ValidFrom, triggers an exclusive flip.
	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: backfillOccurred, IngestedAt: backfillOccurred}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "deployed_on", Dst: "jclaws-mac-mini", Fact: "deployed on the mini", ValidFrom: "2026-01-01", Confidence: 0.95},
		},
	}
	if _, err := Apply(st, ep2, "", res2, DefaultExclusive); err != nil {
		t.Fatalf("Apply 2: %v", err)
	}

	facts := mustFacts(t, st, "hermes-mini")
	var old *store.Fact
	for i := range facts {
		if facts[i].Dst == "digitalocean" {
			old = &facts[i]
		}
	}
	if old == nil || old.InvalidAt == nil {
		t.Fatalf("expected digitalocean fact invalidated: %+v", old)
	}
	if !old.InvalidAt.Equal(validFrom) {
		t.Fatalf("expected InvalidAt clamped to the fact's own ValidFrom (%v), got %v", validFrom, old.InvalidAt)
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
