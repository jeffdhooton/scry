package resolve

import (
	"os"
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

func TestNormalizeRelation(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"deployed_on", "deployed_on"},
		{"Deployed: On!", "deployed_on"},
		{"  DEPLOYED   ON  ", "deployed_on"},
		{"deployed_: on", "deployed_on"},
		{":::", ""},
		{"   ", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeRelation(c.in); got != c.want {
			t.Errorf("normalizeRelation(%q) = %q, want %q", c.in, got, c.want)
		}
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
	// Descriptions fill, they do not replace: a later mention of the same
	// entity must not rewrite an identity that is already established.
	if got.Description != "original desc" {
		t.Fatalf("description should be preserved, got: %+v", got)
	}
	if !got.CreatedAt.Equal(created) {
		t.Fatalf("CreatedAt should be kept from existing: got %v want %v", got.CreatedAt, created)
	}
	if !got.LastSeen.Equal(occurred) {
		t.Fatalf("LastSeen should be max(existing, occurred): got %v want %v", got.LastSeen, occurred)
	}
	// "jclaws-mac-mini" shares only the word "mini" with "Hermes Mini", and
	// one shared word is how unrelated things used to fuse, so it waits for
	// a second, independent episode to attest it.
	if len(got.Aliases) != 1 || got.Aliases[0] != "the mini" {
		t.Fatalf("a one-episode alias sharing a single word should be deferred: %+v", got.Aliases)
	}
	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: occurred, IngestedAt: occurred}
	if _, err := Apply(st, ep2, "", res, DefaultExclusive); err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	got, err = st.GetEntity("hermes-mini")
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	wantAliases := map[string]bool{"the mini": true, "jclaws-mac-mini": true}
	if len(got.Aliases) != len(wantAliases) {
		t.Fatalf("a second episode should admit the alias: %+v", got.Aliases)
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

	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	// isWorkspacePath now requires /Users/ AND a real .git, so a temp dir
	// outside the home cannot be a repo ref — assert the guard directly.
	if isWorkspacePath(repoDir) {
		t.Fatalf("temp dir outside /Users should not count as a workspace: %s", repoDir)
	}
	if _, err := Apply(st, ep, repoDir, res, DefaultExclusive); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := st.GetEntity("loom")
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	if got.Description != "original desc" {
		t.Fatalf("description should not be overwritten by empty incoming: %+v", got)
	}
	if len(got.RepoRefs) != 0 {
		t.Fatalf("a non-home path must not become a repo ref: %+v", got.RepoRefs)
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
	// Still zero: neither cwd qualified, so nothing was ever unioned in.
	if len(got2.RepoRefs) != 0 {
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

	// "replaces" is the inverse of the canonical replaced_by, so the fact
	// was stored as authorclaw -[replaced_by]-> book-system and the hint,
	// mapped the same way, still finds it.
	var facts []store.Fact
	for _, f := range mustFacts(t, st, "authorclaw") {
		if f.Relation == RelReplacedBy {
			facts = append(facts, f)
		}
	}
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

// TestApply_SupersedesNormalizesRelation covers the follow-up to F7: stored
// fact relations are always normalized (normalizeRelation), so a supersedes
// hint must normalize ref.Relation the same way before looking up its
// target — otherwise a hint like "Deployed: On!" would silently miss the
// stored "deployed_on" fact and never invalidate it.
func TestApply_SupersedesNormalizesRelation(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: t1, IngestedAt: t1}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "a", Relation: "deployed_on", Dst: "b", Fact: "a deployed on b", Confidence: 0.8},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: t2, IngestedAt: t2}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{
				Src: "a", Relation: "status", Dst: "moved", Fact: "a moved off b", Confidence: 0.9,
				Supersedes: &extract.SupRef{Src: "a", Relation: "Deployed: On!", Dst: "b"},
			},
		},
	}
	stats2, err := Apply(st, ep2, "", res2, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	if stats2.FactsInvalidated != 1 {
		t.Fatalf("expected the normalized supersedes hint to invalidate the deployed_on fact, got %+v", stats2)
	}

	facts := mustFacts(t, st, "a")
	var deployedOn *store.Fact
	for i := range facts {
		if facts[i].Relation == "deployed_on" && facts[i].Dst == "b" {
			deployedOn = &facts[i]
		}
	}
	if deployedOn == nil {
		t.Fatalf("expected the deployed_on fact still present (invalidated, not deleted): %+v", facts)
	}
	if deployedOn.InvalidAt == nil {
		t.Fatalf("expected the deployed_on fact to be invalidated by the (normalized) supersedes hint")
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

// Rule 6: exclusive relation flip. replaced_by is exclusive: a thing has
// one successor. deployed_on is not, so it is no longer the example here.
func TestApply_ExclusiveFlip(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: t1, IngestedAt: t1}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "replaced_by", Dst: "digitalocean", Fact: "deployed on DO", Confidence: 0.9},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: t2, IngestedAt: t2}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "replaced_by", Dst: "jclaws-mac-mini", Fact: "deployed on the mini", Confidence: 0.95},
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
		t.Fatalf("expected old replaced_by fact invalidated: %+v", old)
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

	restatement := extract.Fct{Src: "hermes-mini", Relation: "replaced_by", Dst: "digitalocean", Fact: "still on DO", Confidence: 0.85}
	flip := extract.Fct{Src: "hermes-mini", Relation: "replaced_by", Dst: "jclaws-mac-mini", Fact: "moved to the mini", Confidence: 0.95}

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
					{Src: "hermes-mini", Relation: "replaced_by", Dst: "digitalocean", Fact: "deployed on DO", Confidence: 0.9},
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

// Guard: an episode may not retire a fact that is newer than itself.
// Transcripts arrive in whatever order the sweep finds them, so a July
// session is routinely resolved after an August fact is already stored.
// The August fact stays current and the July one is recorded as having
// ended when August began, rather than the store answering with the
// older state.
func TestApply_OlderEpisodeDoesNotRetireANewerFact(t *testing.T) {
	st := openTemp(t)
	validFrom := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	backfillOccurred := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: validFrom, IngestedAt: validFrom}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "replaced_by", Dst: "digitalocean", Fact: "replaced by DO", ValidFrom: "2026-05-01", Confidence: 0.9},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: backfillOccurred, IngestedAt: backfillOccurred}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{Src: "hermes-mini", Relation: "replaced_by", Dst: "jclaws-mac-mini", Fact: "replaced by the mini", ValidFrom: "2026-01-01", Confidence: 0.95},
		},
	}
	stats, err := Apply(st, ep2, "", res2, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	if stats.FactsInvalidated != 0 {
		t.Errorf("an older episode invalidated %d newer facts, want 0", stats.FactsInvalidated)
	}

	facts := mustFacts(t, st, "hermes-mini")
	var newer, older *store.Fact
	for i := range facts {
		switch facts[i].Dst {
		case "digitalocean":
			newer = &facts[i]
		case "jclaws-mac-mini":
			older = &facts[i]
		}
	}
	if newer == nil || newer.InvalidAt != nil {
		t.Fatalf("the May fact must stay current: %+v", newer)
	}
	if older == nil || older.InvalidAt == nil {
		t.Fatalf("the backfilled January fact must be stored as already over: %+v", older)
	}
	if !older.InvalidAt.Equal(validFrom) {
		t.Errorf("the January fact must end where May begins (%v), got %v", validFrom, older.InvalidAt)
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

// TestApply_RelationNormalized covers F7: an unvalidated LLM relation string
// containing punctuation must be normalized before it ever reaches the
// store (which colon-joins src/relation/dst into its on-disk key), and the
// normalized form must be exactly what later exact-match lookups (Rule 4
// merge, DefaultExclusive) expect — proven here by merging a normalized
// "Deployed: On!" onto an existing plain "deployed_on" fact instead of
// creating a second, duplicate one.
func TestApply_RelationNormalized(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: t1, IngestedAt: t1}
	res1 := extract.Result{
		Facts: []extract.Fct{
			{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "runs there", Confidence: 0.7},
		},
	}
	if _, err := Apply(st, ep1, "", res1, DefaultExclusive); err != nil {
		t.Fatalf("Apply 1: %v", err)
	}

	ep2 := store.Episode{ID: "ep-2", Source: "manual", SourceRef: "y", OccurredAt: t2, IngestedAt: t2}
	res2 := extract.Result{
		Facts: []extract.Fct{
			{Src: "book-system", Relation: "Deployed: On!", Dst: "hermes-mini", Fact: "still runs there", Confidence: 0.9},
		},
	}
	stats2, err := Apply(st, ep2, "", res2, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply 2: %v", err)
	}
	if stats2.FactsMerged != 1 || stats2.FactsAdded != 0 {
		t.Fatalf("expected the normalized relation to merge onto the existing deployed_on fact, got %+v", stats2)
	}

	facts := mustFacts(t, st, "book-system")
	if len(facts) != 1 {
		t.Fatalf("expected exactly one fact (merged, not duplicated), got %d: %+v", len(facts), facts)
	}
	if facts[0].Relation != "deployed_on" {
		t.Fatalf("Relation = %q, want normalized %q", facts[0].Relation, "deployed_on")
	}
}

// TestApply_RelationNormalizesToEmptySkipsFact covers F7's other half: a
// relation that is pure punctuation (nothing left after normalization) must
// be dropped outright — no fact written, and no concept stub created for
// either endpoint, since the fact never gets far enough to touch them.
func TestApply_RelationNormalizesToEmptySkipsFact(t *testing.T) {
	st := openTemp(t)
	occurred := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-1", Source: "manual", SourceRef: "x", OccurredAt: occurred, IngestedAt: occurred}
	res := extract.Result{
		Facts: []extract.Fct{
			{Src: "book-system", Relation: ":::", Dst: "hermes-mini", Fact: "junk relation", Confidence: 0.5},
		},
	}

	stats, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if stats.FactsAdded != 0 || stats.EntitiesCreated != 0 {
		t.Fatalf("expected the fact to be skipped with nothing counted, got %+v", stats)
	}
	_, entities, facts, err := st.Counts()
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if entities != 0 || facts != 0 {
		t.Fatalf("expected nothing written, got entities=%d facts=%d", entities, facts)
	}
}

// --- entity hygiene ---
//
// Three observed corruptions in the live graph, each with its own test:
// a throwaway session overwrote a real project's description; temp worktree
// names became permanent aliases; and repo_refs accumulated four unrelated
// repos on one entity.

func TestApply_DescriptionIsNotClobberedByALaterEpisode(t *testing.T) {
	st := openTemp(t)
	t0 := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	ep1 := store.Episode{ID: "e1", Source: "manual", SourceRef: "a", OccurredAt: t0, IngestedAt: t0}
	if _, err := Apply(st, ep1, "", extract.Result{Entities: []extract.Ent{
		{Name: "cleaning-company", Type: "service", Description: "the residential cleaning operations app"},
	}}, nil); err != nil {
		t.Fatalf("apply 1: %v", err)
	}

	// A throwaway session in an unrelated temp dir mentions the same name.
	t1 := t0.Add(120 * time.Hour)
	ep2 := store.Episode{ID: "e2", Source: "manual", SourceRef: "b", OccurredAt: t1, IngestedAt: t1}
	if _, err := Apply(st, ep2, "", extract.Result{Entities: []extract.Ent{
		{Name: "cleaning-company", Type: "service", Description: "No git remote exists in survtest."},
	}}, nil); err != nil {
		t.Fatalf("apply 2: %v", err)
	}

	got, err := st.GetEntity(store.Slugify("cleaning-company"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Description != "the residential cleaning operations app" {
		t.Fatalf("description was clobbered: %q", got.Description)
	}
}

func TestApply_EphemeralAliasesAreRejected(t *testing.T) {
	st := openTemp(t)
	t0 := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "e1", Source: "manual", SourceRef: "a", OccurredAt: t0, IngestedAt: t0}
	if _, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{
		Name: "setpoint", Type: "project", Description: "loop engine",
		Aliases: []string{
			"setpoint loop engine",        // keep: a real alias (shares "setpoint")
			"setpoint-wt-9e6jz82r",        // drop: a temp worktree
			"/private/var/folders/p2/T/x", // drop: a temp path
			"/tmp/survtest",               // drop: a scratch dir
			"a37497eb9f6b",                // drop: a bare hex id
		},
	}}}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, err := st.GetEntity(store.Slugify("setpoint"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for _, bad := range []string{"setpoint-wt-9e6jz82r", "/private/var/folders/p2/T/x", "/tmp/survtest", "a37497eb9f6b"} {
		if containsString(got.Aliases, bad) {
			t.Fatalf("ephemeral alias survived: %q in %v", bad, got.Aliases)
		}
	}
	if !containsString(got.Aliases, "setpoint loop engine") {
		t.Fatalf("real alias was dropped: %v", got.Aliases)
	}
}

func TestApply_EphemeralEntityNamesAreNotStored(t *testing.T) {
	st := openTemp(t)
	t0 := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "e1", Source: "manual", SourceRef: "a", OccurredAt: t0, IngestedAt: t0}
	if _, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{
		{Name: "setpoint-wt-9e6jz82r", Type: "project", Description: "a temp worktree"},
	}}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := st.GetEntity(store.Slugify("setpoint-wt-9e6jz82r")); err == nil {
		t.Fatal("a temp worktree was stored as an entity")
	}
}

func TestApply_RepoRefsRequireARealRepoAndAreCapped(t *testing.T) {
	st := openTemp(t)
	t0 := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	// A /Users/ path that is not a git repo must not become a repo ref.
	notARepo := t.TempDir()
	ep := store.Episode{ID: "e1", Source: "manual", SourceRef: "a", OccurredAt: t0, IngestedAt: t0}
	if _, err := Apply(st, ep, notARepo, extract.Result{Entities: []extract.Ent{
		{Name: "thing", Type: "project", Description: "d"},
	}}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, _ := st.GetEntity(store.Slugify("thing"))
	if len(got.RepoRefs) != 0 {
		t.Fatalf("non-repo path became a repo ref: %v", got.RepoRefs)
	}
}

func TestEphemeralName_CatchesSessionUUIDs(t *testing.T) {
	// A recall surfaced `019ffe05-2a03-7263-b0c0-e0a8f98cddb8` as an alias of a
	// real agent: a session UUID, durable to nothing.
	for _, s := range []string{
		"019ffe05-2a03-7263-b0c0-e0a8f98cddb8",
		"8f9ed8d2-66a1-4dd3-90c5-f30a531f17d9",
	} {
		if !isEphemeralName(s) {
			t.Fatalf("session uuid not rejected: %s", s)
		}
	}
	for _, s := range []string{"setpoint", "loop engine", "hermes-mini", "program-health"} {
		if isEphemeralName(s) {
			t.Fatalf("durable name wrongly rejected: %s", s)
		}
	}
}

func TestIsGenericAlias_RejectsMergeMagnets(t *testing.T) {
	// Every one of these was a real alias on one entity that had fused
	// setpoint, cleaning-company's operations app, and program-health.
	magnets := []string{
		"workspace", "repo root", "the repo", "current worktree", "worktree",
		"orchestrator", "task orchestrator", "executor", "the project",
		"project", "app", "the app", "operations app", "loop", "north star",
		"repo", "current directory", "root", "engine", "agent", "tool",
	}
	for _, m := range magnets {
		if !isGenericAlias(m) {
			t.Errorf("merge magnet not rejected: %q", m)
		}
	}
	// Sub-paths are locations, not identities.
	for _, p := range []string{"apps/operations", "setpoint/tools", "setpoint/greet-core"} {
		if !isGenericAlias(p) {
			t.Errorf("sub-path not rejected: %q", p)
		}
	}
	// Real identities must survive.
	for _, keep := range []string{
		"setpoint", "loom", "loop engine", "jeffdhooton/setpoint",
		"program-health", "hermes-mini", "childscribe", "scry",
	} {
		if isGenericAlias(keep) {
			t.Errorf("real identity wrongly rejected: %q", keep)
		}
	}
}

func TestApply_GenericAliasesNeverMerge(t *testing.T) {
	st := openTemp(t)
	t0 := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	ep1 := store.Episode{ID: "e1", Source: "manual", SourceRef: "a", OccurredAt: t0, IngestedAt: t0}
	if _, err := Apply(st, ep1, "", extract.Result{Entities: []extract.Ent{
		{Name: "setpoint", Type: "project", Description: "loop engine",
			Aliases: []string{"loop engine", "workspace", "orchestrator"}},
	}}, nil); err != nil {
		t.Fatalf("apply 1: %v", err)
	}

	// A different project calling itself "the orchestrator" must NOT merge in.
	ep2 := store.Episode{ID: "e2", Source: "manual", SourceRef: "b", OccurredAt: t0, IngestedAt: t0}
	if _, err := Apply(st, ep2, "", extract.Result{Entities: []extract.Ent{
		{Name: "orchestrator", Type: "project", Description: "something else entirely"},
	}}, nil); err != nil {
		t.Fatalf("apply 2: %v", err)
	}

	sp, err := st.GetEntity(store.Slugify("setpoint"))
	if err != nil {
		t.Fatalf("get setpoint: %v", err)
	}
	if containsString(sp.Aliases, "workspace") || containsString(sp.Aliases, "orchestrator") {
		t.Fatalf("generic aliases were stored: %v", sp.Aliases)
	}
	if sp.Description != "loop engine" {
		t.Fatalf("a foreign entity merged into setpoint: %q", sp.Description)
	}
}

func TestIsGenericEntityName_RejectsProcessNouns(t *testing.T) {
	// These are all real entities in the live graph. None is an identity;
	// each became a magnet that thousands of documents then "collided" with.
	junk := []string{
		"plan", "the plan", "implementation plan", "commit", "task", "Task 2",
		"Phase 1 plan", "phase 2", "step 3", "iteration 4", "review", "report",
		"approved", "passed", "failed", "private", "status", "result",
		"branch", "tests", "docs", "issue", "bug", "feature",
	}
	for _, j := range junk {
		if !isGenericEntityName(j) {
			t.Errorf("junk entity name not rejected: %q", j)
		}
	}
	keep := []string{
		"setpoint", "program-health", "scry", "hermes-mini", "childscribe",
		"anthropic-api", "cleaning-company", "room-worker", "Bun v1.4.0",
	}
	for _, k := range keep {
		if isGenericEntityName(k) {
			t.Errorf("real identity wrongly rejected: %q", k)
		}
	}
}

func TestApply_GenericEntityNamesAreNotStored(t *testing.T) {
	st := openTemp(t)
	t0 := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "e1", Source: "manual", SourceRef: "a", OccurredAt: t0, IngestedAt: t0}
	if _, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{
		{Name: "plan", Type: "concept", Description: "an implementation plan"},
		{Name: "setpoint", Type: "project", Description: "loop engine"},
	}}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := st.GetEntity(store.Slugify("plan")); err == nil {
		t.Fatal(`"plan" was stored as an entity`)
	}
	if _, err := st.GetEntity(store.Slugify("setpoint")); err != nil {
		t.Fatalf("real entity was lost: %v", err)
	}
}

// Rule 5 obeys the same order as Rule 6: a session cannot report the end
// of something that had not started when it ran.
func TestApply_OlderEpisodeSupersedesHintIsIgnored(t *testing.T) {
	st := openTemp(t)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	epNew := store.Episode{ID: "ep-aug", Source: "manual", SourceRef: "a", OccurredAt: aug, IngestedAt: aug}
	resNew := extract.Result{Facts: []extract.Fct{
		{Src: "childscribe", Relation: "uses", Dst: "drizzle", Fact: "childscribe uses drizzle", Confidence: 0.9},
	}}
	if _, err := Apply(st, epNew, "", resNew, DefaultExclusive); err != nil {
		t.Fatal(err)
	}

	epOld := store.Episode{ID: "ep-jul", Source: "manual", SourceRef: "b", OccurredAt: jul, IngestedAt: jul}
	resOld := extract.Result{Facts: []extract.Fct{{
		Src: "childscribe", Relation: "uses", Dst: "prisma", Fact: "childscribe uses prisma", Confidence: 0.9,
		Supersedes: &extract.SupRef{Src: "childscribe", Relation: "uses", Dst: "drizzle"},
	}}}
	if _, err := Apply(st, epOld, "", resOld, DefaultExclusive); err != nil {
		t.Fatal(err)
	}

	for _, f := range mustFacts(t, st, "childscribe") {
		if f.Dst == "drizzle" && f.InvalidAt != nil {
			t.Fatalf("a July episode retired an August fact: %+v", f)
		}
	}
}
