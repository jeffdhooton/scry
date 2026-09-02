package resolve

import (
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func putEntity(t *testing.T, st *store.Store, slug, name, typ string, aliases ...string) store.Entity {
	t.Helper()
	e := store.Entity{Slug: slug, Name: name, Type: typ, Aliases: aliases, CreatedAt: time.Now(), LastSeen: time.Now()}
	if err := st.PutEntity(e); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestAdmitAliasRejectsReferencesValuesAndGenerics(t *testing.T) {
	st := openTemp(t)
	e := putEntity(t, st, "hermes-ops", "hermes-ops", "project")
	for _, a := range []string{"the machine", "this box", "box", "you", "I", "the user", "main", "in-progress", "46 GiB", "workspace", "setpoint-wt-9e6jz82r", "hermes-ops", "gw", "a"} {
		ok, reason, err := AdmitAlias(st, e, a, "ep1")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Errorf("AdmitAlias(%q) admitted (%s)", a, reason)
		}
	}
}

func TestAdmitAliasSharingATokenIsImmediate(t *testing.T) {
	st := openTemp(t)
	e := putEntity(t, st, "scry", "scry", "project")
	for _, a := range []string{"scry daemon", "context-stack/scry", "Scry memory", "scryd"} {
		ok, reason, _ := AdmitAlias(st, e, a, "ep1")
		if !ok {
			t.Errorf("AdmitAlias(%q) refused: %s", a, reason)
		}
	}
}

func TestAdmitAliasWithoutSharedTokenNeedsTwoEpisodes(t *testing.T) {
	st := openTemp(t)
	e := putEntity(t, st, "jeff", "Jeff", "person")
	if ok, _, _ := AdmitAlias(st, e, "jclaw", "ep1"); ok {
		t.Fatal("first attestation must not admit")
	}
	if ok, _, _ := AdmitAlias(st, e, "jclaw", "ep1"); ok {
		t.Fatal("the same episode again must not admit")
	}
	if ok, reason, _ := AdmitAlias(st, e, "jclaw", "ep2"); !ok {
		t.Fatalf("second independent episode must admit: %s", reason)
	}
}

func TestAdmitAliasOwnedByAnotherEntity(t *testing.T) {
	st := openTemp(t)
	project := putEntity(t, st, "hermes-ops", "hermes-ops", "project", "hermes ops")
	putEntity(t, st, "mac-mini", "Mac mini", "machine", "mini")
	putEntity(t, st, "hermes", "Hermes", "service")

	// Incompatible type: never, however many episodes say so.
	for i, ep := range []string{"e1", "e2", "e3"} {
		if ok, _, _ := AdmitAlias(st, project, "mini", ep); ok {
			t.Fatalf("episode %d admitted a machine's alias onto a project", i+1)
		}
	}
	// Another entity's own name, compatible type: two attestations.
	tool := putEntity(t, st, "hermes-agent", "hermes-agent", "service")
	if ok, _, _ := AdmitAlias(st, tool, "Hermes", "e1"); ok {
		t.Fatal("one episode must not merge two existing entities")
	}
	if ok, reason, _ := AdmitAlias(st, tool, "Hermes", "e2"); !ok {
		t.Fatalf("two episodes must admit a compatible merge: %s", reason)
	}
}

func TestApply_NameReachingAnIncompatibleEntityByAliasStaysSeparate(t *testing.T) {
	st := openTemp(t)
	putEntity(t, st, "hermes-ops", "hermes-ops", "project", "mini", "Mac mini")
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "e1", Source: "manual", SourceRef: "e1", OccurredAt: at, IngestedAt: at}
	res := extract.Result{EpisodeSummary: "x",
		Entities: []extract.Ent{{Name: "Mac mini", Type: "machine", Description: "the Mac mini"}},
		Facts:    []extract.Fct{{Src: "scry", Relation: "deployed_on", Dst: "Mac mini", Fact: "scry runs on the Mac mini", Confidence: 0.9}},
	}
	stats, err := Apply(st, ep, "", res, DefaultExclusive)
	if err != nil {
		t.Fatal(err)
	}
	if stats.EntitiesCreated < 1 {
		t.Fatalf("stats = %+v, want a new machine entity", stats)
	}
	m, err := st.GetEntity("mac-mini")
	if err != nil || m.Type != "machine" {
		t.Fatalf("mac-mini = %+v, %v", m, err)
	}
	if slug, _, _ := st.ResolveAlias("Mac mini"); slug != "mac-mini" {
		t.Errorf("the machine's own name must now resolve to it, got %q", slug)
	}
	facts, _ := st.FactsAbout("mac-mini", false)
	if len(facts) != 1 {
		t.Errorf("deployed_on fact must land on the machine, got %d", len(facts))
	}
	if ops, _ := st.FactsAbout("hermes-ops", false); len(ops) != 0 {
		t.Errorf("hermes-ops must not receive the machine's fact: %+v", ops)
	}
}

func TestApply_ConceptStubUpgradesToTypedEntity(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	applyOne(t, st, "e1", at, extract.Fct{Src: "scry", Relation: "deployed_on", Dst: "halo", Fact: "x", Confidence: 0.9})
	if e, _ := st.GetEntity("halo"); e.Type != "concept" {
		t.Fatalf("stub type = %q", e.Type)
	}
	ep := store.Episode{ID: "e2", Source: "manual", SourceRef: "e2", OccurredAt: at, IngestedAt: at}
	if _, err := Apply(st, ep, "", extract.Result{EpisodeSummary: "x", Entities: []extract.Ent{{Name: "halo", Type: "machine"}}}, DefaultExclusive); err != nil {
		t.Fatal(err)
	}
	if e, _ := st.GetEntity("halo"); e.Type != "machine" {
		t.Errorf("stub not upgraded: %q", e.Type)
	}
}

func TestSharesToken(t *testing.T) {
	cases := []struct {
		alias string
		names []string
		want  bool
	}{
		{"scry daemon", []string{"scry"}, true},
		{"Mac mini", []string{"hermes-ops"}, false},
		{"jclaw", []string{"Jeff"}, false},
		{"Jermes", []string{"Hermes"}, false},
		{"childscribe", []string{"ChildScribe Laravel"}, true},
		{"gpt-oss", []string{"gpt-oss-120b"}, true},
		{"the", []string{"the-thing"}, false},
	}
	for _, tc := range cases {
		if got := sharesToken(tc.alias, tc.names...); got != tc.want {
			t.Errorf("sharesToken(%q, %v) = %v, want %v", tc.alias, tc.names, got, tc.want)
		}
	}
}

func TestAdmitAliasRefusesHardwareOnProjectsAndDescriptionCommonNouns(t *testing.T) {
	st := openTemp(t)
	ops := store.Entity{Slug: "hermes-ops", Name: "hermes-ops", Type: "project", Description: "Standing Hermes agent on the Mac mini, reachable over the tailnet"}
	_ = st.PutEntity(ops)
	for _, a := range []string{"M4 Mac mini", "jclaws mini", "mini box"} {
		if ok, reason, _ := AdmitAlias(st, ops, a, "e1"); ok {
			t.Errorf("AdmitAlias(%q) admitted a machine name onto a project: %s", a, reason)
		}
	}
	// A description made of common nouns admits nothing in one episode.
	sp := store.Entity{Slug: "setpoint", Name: "setpoint", Type: "project", Description: "loop engine"}
	_ = st.PutEntity(sp)
	if ok, _, _ := AdmitAlias(st, sp, "loop engine", "e1"); ok {
		t.Error("common-noun description tokens must not admit an alias on one episode")
	}
	// A specific description token still does.
	wren := store.Entity{Slug: "wren", Name: "wren", Type: "project", Description: "the wrenops cleaning dispatcher"}
	_ = st.PutEntity(wren)
	if ok, _, _ := AdmitAlias(st, wren, "wrenops", "e1"); !ok {
		t.Error("a specific description token should admit")
	}
}

func TestConceptUpgradeRevalidatesAliases(t *testing.T) {
	st := openTemp(t)
	putEntity(t, st, "mac-mini", "Mac mini", "machine", "mini")
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	// A concept stub collects "mini" over two episodes (compatible as a
	// wildcard), then turns out to be a project.
	for _, id := range []string{"e1", "e2"} {
		ep := store.Episode{ID: id, Source: "manual", SourceRef: id, OccurredAt: at, IngestedAt: at}
		if _, err := Apply(st, ep, "", extract.Result{EpisodeSummary: "x", Entities: []extract.Ent{{Name: "widget", Type: "concept", Aliases: []string{"mini"}}}}, DefaultExclusive); err != nil {
			t.Fatal(err)
		}
	}
	ep := store.Episode{ID: "e3", Source: "manual", SourceRef: "e3", OccurredAt: at, IngestedAt: at}
	if _, err := Apply(st, ep, "", extract.Result{EpisodeSummary: "x", Entities: []extract.Ent{{Name: "widget", Type: "project"}}}, DefaultExclusive); err != nil {
		t.Fatal(err)
	}
	w, _ := st.GetEntity("widget")
	for _, a := range w.Aliases {
		if store.Normalize(a) == "mini" {
			t.Errorf("project widget kept the machine's alias after upgrade: %v", w.Aliases)
		}
	}
	rep, _ := Hygiene(st, true)
	if rep.CrossTypeCollisions != 0 {
		t.Errorf("collisions after upgrade = %d", rep.CrossTypeCollisions)
	}
}
