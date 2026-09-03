package resolve

import (
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func applyOne(t *testing.T, st *store.Store, id string, at time.Time, facts ...extract.Fct) Stats {
	t.Helper()
	ep := store.Episode{ID: id, Source: "manual", SourceRef: id, OccurredAt: at, IngestedAt: at}
	stats, err := Apply(st, ep, "", extract.Result{EpisodeSummary: id, Facts: facts}, DefaultExclusive)
	if err != nil {
		t.Fatalf("Apply %s: %v", id, err)
	}
	return stats
}

func TestApply_StatusIsAlwaysAnAttribute(t *testing.T) {
	st := openTemp(t)
	t1 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	stats := applyOne(t, st, "e1", t1, extract.Fct{Src: "scry", Relation: "status", Dst: "in-progress", Fact: "scry is in progress", Confidence: 0.9})
	if stats.FactsAdded != 1 || stats.EntitiesCreated != 1 {
		t.Fatalf("stats = %+v, want one fact and only the scry stub", stats)
	}
	if _, err := st.GetEntity("in-progress"); err == nil {
		t.Error("a status value must never become an entity")
	}
	facts := mustFacts(t, st, "scry")
	if len(facts) != 1 || !facts[0].IsAttribute() || facts[0].Value != "in-progress" || facts[0].Relation != RelStatus {
		t.Fatalf("facts = %+v", facts)
	}

	// A new status replaces the old one (exclusive relation), even though
	// both targets are values.
	t2 := t1.Add(24 * time.Hour)
	stats = applyOne(t, st, "e2", t2, extract.Fct{Src: "scry", Relation: "has_status", Dst: "done", Fact: "scry is done", Confidence: 0.9})
	if stats.FactsInvalidated != 1 || stats.FactsAdded != 1 {
		t.Fatalf("second status: %+v", stats)
	}
	current, _ := st.FactsFrom("scry", false)
	if len(current) != 1 || current[0].Value != "done" || current[0].RawRelation != "has_status" {
		t.Fatalf("current = %+v", current)
	}
	// Restating the current value merges rather than adding.
	stats = applyOne(t, st, "e3", t2.Add(time.Hour), extract.Fct{Src: "scry", Relation: "status", Dst: "done", Fact: "still done", Confidence: 0.5})
	if stats.FactsMerged != 1 || stats.FactsAdded != 0 {
		t.Fatalf("restatement: %+v", stats)
	}
}

func TestApply_ValueEndpointsBecomeAttributesOrAreDropped(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	stats := applyOne(t, st, "e1", at,
		extract.Fct{Src: "gpt-oss-120b", Relation: "has_active_parameters", Dst: "51B", Fact: "gpt-oss-120b has 51B active parameters", Confidence: 0.9},
		extract.Fct{Src: "main", Relation: "contains", Dst: "scry", Fact: "main has scry", Confidence: 0.9},
		extract.Fct{Src: "46 GiB", Relation: "related_to", Dst: "in-progress", Fact: "nonsense", Confidence: 0.5},
		extract.Fct{Src: "halo", Relation: "runs_on_branch", Dst: "feat/x", Fact: "halo runs on branch feat/x", Confidence: 0.7},
	)
	if stats.FactsRejected != 1 {
		t.Errorf("FactsRejected = %d, want 1 (value-to-value)", stats.FactsRejected)
	}
	for _, slug := range []string{"51b", "main", "46-gib", "in-progress", "featx"} {
		if _, err := st.GetEntity(slug); err == nil {
			t.Errorf("value %q became an entity", slug)
		}
	}
	gpt, _ := st.FactsFrom("gpt-oss-120b", false)
	if len(gpt) != 1 || gpt[0].Value != "51B" || gpt[0].Relation != RelStatus {
		t.Errorf("measurement fact = %+v, want a status attribute with value 51B", gpt)
	}
	scry, _ := st.FactsFrom("scry", false)
	if len(scry) != 1 || scry[0].Value != "main" || scry[0].Relation != RelContains {
		t.Errorf("flipped value-src fact = %+v, want scry with value main", scry)
	}
	halo, _ := st.FactsFrom("halo", false)
	if len(halo) != 1 || halo[0].Value != "feat/x" {
		t.Errorf("branch fact = %+v", halo)
	}
}

func TestApply_RelationsMapOntoTheClosedVocabulary(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	applyOne(t, st, "e1", at,
		extract.Fct{Src: "childscribe-laravel", Relation: "used_by", Dst: "operations-app", Fact: "the ops app uses laravel", Confidence: 0.9},
		extract.Fct{Src: "scry", Relation: "Installed On!", Dst: "mac-mini", Fact: "scry is installed on the mini", Confidence: 0.9},
		extract.Fct{Src: "scry", Relation: "robots_method_now_welcomes", Dst: "crawlers", Fact: "odd verb", Confidence: 0.4},
	)
	all, _ := st.AllFacts()
	got := map[string]store.Fact{}
	for _, f := range all {
		got[f.Src+" "+f.Relation+" "+f.Dst] = f
		if !IsCanonical(f.Relation) {
			t.Errorf("stored non-canonical relation %q", f.Relation)
		}
	}
	if f, ok := got["operations-app uses childscribe-laravel"]; !ok || f.RawRelation != "used_by" {
		t.Errorf("used_by must flip into uses with the raw relation kept: %+v", got)
	}
	if f, ok := got["scry deployed_on mac-mini"]; !ok || f.RawRelation != "installed_on" {
		t.Errorf("installed_on must map to deployed_on: %+v", got)
	}
	if f, ok := got["scry related_to crawlers"]; !ok || f.RawRelation != "robots_method_now_welcomes" {
		t.Errorf("unknown verbs land on related_to with the raw relation kept: %+v", got)
	}
}

func TestApply_ValueNamedEntitiesAreNotCreated(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "e1", Source: "manual", SourceRef: "e1", OccurredAt: at, IngestedAt: at}
	res := extract.Result{EpisodeSummary: "x", Entities: []extract.Ent{
		{Name: "main", Type: "concept"},
		{Name: "in-progress", Type: "concept"},
		{Name: "scry", Type: "project", Aliases: []string{"v2.1", "done", "scry daemon"}},
	}}
	if _, err := Apply(st, ep, "", res, DefaultExclusive); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"main", "in-progress"} {
		if _, err := st.GetEntity(slug); err == nil {
			t.Errorf("%q must not be created as an entity", slug)
		}
	}
	e, err := st.GetEntity("scry")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Aliases) != 1 || e.Aliases[0] != "scry daemon" {
		t.Errorf("value aliases must be dropped: %v", e.Aliases)
	}
}

func TestApply_FactEndpointsNeverCreateNonIdentityEntities(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	stats := applyOne(t, st, "e1", at,
		extract.Fct{Src: "setpoint-wt-lpj7ikz0 worktree", Relation: "owns", Dst: "port-22460", Fact: "the worktree owns port 22460", Confidence: 0.9},
		extract.Fct{Src: "scry", Relation: "contains", Dst: "plan", Fact: "scry has a plan", Confidence: 0.9},
		extract.Fct{Src: "scry", Relation: "blocked_by", Dst: "a49bec73610fc684", Fact: "blocked by run a49bec73610fc684", Confidence: 0.9},
	)
	// The worktree fact has a value at both ends — a scratch worktree id
	// and a port number — so it says nothing about any identity and is
	// rejected rather than stored.
	if stats.FactsAdded != 2 || stats.FactsRejected != 1 {
		t.Fatalf("stats = %+v, want two facts and one rejected", stats)
	}
	for _, bad := range []string{"setpoint-wt-lpj7ikz0-worktree", "plan", "a49bec73610fc684"} {
		if _, err := st.GetEntity(bad); err == nil {
			t.Errorf("fact endpoint %q became an entity", bad)
		}
	}
	// Nothing is stored for the worktree fact: a scratch worktree id and a
	// port number are both values, so there is no identity to attach it to.
	if port, _ := st.FactsFrom("port-22460", false); len(port) != 0 {
		t.Errorf("a fact with a value at both ends must be dropped, got %+v", port)
	}
	scry, _ := st.FactsFrom("scry", false)
	if len(scry) != 2 {
		t.Fatalf("scry facts = %d, want 2 attributes", len(scry))
	}
	for _, f := range scry {
		if !f.IsAttribute() {
			t.Errorf("expected an attribute, got %+v", f)
		}
	}
	if NotAnIdentity("scry") || !NotAnIdentity("plan") || !NotAnIdentity("main") || !NotAnIdentity("/tmp/x") {
		t.Error("NotAnIdentity wrong")
	}
}

func TestApply_StatusToARealEntityStaysAnEdge(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	putEntity(t, st, "setpoint-fleet", "setpoint fleet", "project")
	applyOne(t, st, "e1", at,
		extract.Fct{Src: "dotfiles", Relation: "status", Dst: "setpoint fleet", Fact: "the fleet applied Vale to commit messages", Confidence: 0.9},
		extract.Fct{Src: "dotfiles", Relation: "status", Dst: "in-progress", Fact: "dotfiles work is in progress", Confidence: 0.9},
	)
	facts, _ := st.FactsFrom("dotfiles", false)
	if len(facts) != 2 {
		t.Fatalf("both facts must be current, got %+v", facts)
	}
	var edge, attr int
	for _, f := range facts {
		if f.IsAttribute() {
			attr++
			if f.Value != "in-progress" {
				t.Errorf("wrong attribute value: %+v", f)
			}
		} else {
			edge++
			if f.Dst != "setpoint-fleet" || f.Relation != RelRelatedTo || f.RawRelation != "status" {
				t.Errorf("status edge to a real entity: %+v", f)
			}
		}
	}
	if edge != 1 || attr != 1 {
		t.Errorf("want one edge and one attribute, got %d and %d", edge, attr)
	}
}
