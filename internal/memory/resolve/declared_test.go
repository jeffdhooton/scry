package resolve

import (
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestDeclaredValuesCollectsNamesAndAliases(t *testing.T) {
	got := DeclaredValues([]extract.Ent{
		{Name: "Loom", Type: "project"},
		{Name: "46 GiB", Type: "value"},
		{Name: "In Progress", Type: "value", Aliases: []string{"in-progress"}},
	})
	for _, want := range []string{"46 gib", "in progress", "in-progress"} {
		if !got[store.Normalize(want)] {
			t.Errorf("declared values missing %q: %v", want, got)
		}
	}
	if got[store.Normalize("Loom")] {
		t.Error("a project must not be collected as a value")
	}
}

// The whole point of the value type: names no lexical rule can judge from
// the spelling alone, because the same string is a real entity elsewhere.
func TestApplyDropsEntitiesTheModelTypedAsValues(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-dv", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
	res := extract.Result{
		Entities: []extract.Ent{
			{Name: "scry", Type: "project"},
			{Name: "main", Type: "value", Description: "the branch scry ships from"},
			{Name: "PENDING-og-images", Type: "value", Description: "a queue state"},
		},
		Facts: []extract.Fct{
			{Src: "scry", Relation: "status", Dst: "main", Fact: "scry ships from main", Confidence: 0.9},
		},
	}
	if _, err := Apply(st, ep, "", res, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main", "PENDING-og-images"} {
		if _, found, err := st.ResolveAlias(name); err != nil || found {
			t.Errorf("%q became an entity (found=%v, err=%v)", name, found, err)
		}
	}
	if _, found, err := st.ResolveAlias("scry"); err != nil || !found {
		t.Fatalf("scry must still be an entity: found=%v err=%v", found, err)
	}
	// The fact survives as an attribute of scry rather than vanishing.
	facts := mustFacts(t, st, mustSlug(t, st, "scry"))
	if len(facts) != 1 || facts[0].Dst != "" {
		t.Errorf("want one attribute fact on scry, got %+v", facts)
	}
}

// The guard: one episode's stray "value" verdict must not demote an entity
// other episodes have already built.
func TestApplyKeepsAnEstablishedEntityCalledAValue(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	first := store.Episode{ID: "ep-a", Source: "manual", SourceRef: "a", OccurredAt: at, IngestedAt: at}
	if _, err := Apply(st, first, "", extract.Result{
		Entities: []extract.Ent{{Name: "hermes-ops", Type: "project", Description: "the ops repo"}},
	}, nil); err != nil {
		t.Fatal(err)
	}
	second := store.Episode{ID: "ep-b", Source: "manual", SourceRef: "b", OccurredAt: at.Add(time.Hour), IngestedAt: at.Add(time.Hour)}
	if _, err := Apply(st, second, "", extract.Result{
		Entities: []extract.Ent{{Name: "hermes-ops", Type: "value"}},
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, found, err := st.ResolveAlias("hermes-ops"); err != nil || !found {
		t.Errorf("an established entity was demoted by one value verdict: found=%v err=%v", found, err)
	}
}

// A value-typed endpoint that never appeared in the entity list is judged
// by the lexical rules alone, exactly as before.
func TestApplyStillJudgesUndeclaredNamesLexically(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-lex", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
	res := extract.Result{
		Entities: []extract.Ent{{Name: "trawl", Type: "project"}},
		Facts: []extract.Fct{
			{Src: "trawl", Relation: "status", Dst: "GRADER2-20260903T000246Z-3", Fact: "trawl ran under probe GRADER2-20260903T000246Z-3", Confidence: 0.9},
		},
	}
	if _, err := Apply(st, ep, "", res, nil); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := st.ResolveAlias("GRADER2-20260903T000246Z-3"); found {
		t.Error("a run probe id became an entity without the model's help")
	}
}

func mustSlug(t *testing.T, st *store.Store, name string) string {
	t.Helper()
	slug, found, err := st.ResolveAlias(name)
	if err != nil || !found {
		t.Fatalf("ResolveAlias(%q): found=%v err=%v", name, found, err)
	}
	return slug
}

// The values grader built this case against the first version of the value
// type and it dropped five real identities, deleting issue-91 -> PR-402
// outright because both endpoints were declared values. The prompt no
// longer calls files and tickets values; this pins the resolver's own veto,
// which is what has to hold when the model ignores the prompt.
func TestApplyRefusesAValueVerdictOnFilesAndTickets(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-art", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
	res := extract.Result{
		Entities: []extract.Ent{
			{Name: "scry", Type: "project"},
			{Name: "internal/memory/resolve/declared.go", Type: "value"},
			{Name: "docs/MEMORY_AUDIT_2026-09-02.md", Type: "value"},
			{Name: "issue-91", Type: "value"},
			{Name: "PR-402", Type: "value"},
		},
		Facts: []extract.Fct{
			{Src: "issue-91", Relation: "fixed_by", Dst: "PR-402", Fact: "issue-91 was fixed by PR-402", Confidence: 0.9},
			{Src: "scry", Relation: "contains", Dst: "internal/memory/resolve/declared.go", Fact: "scry contains declared.go", Confidence: 0.9},
		},
	}
	if _, err := Apply(st, ep, "", res, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"internal/memory/resolve/declared.go",
		"docs/MEMORY_AUDIT_2026-09-02.md",
		"issue-91",
		"PR-402",
	} {
		if _, found, err := st.ResolveAlias(name); err != nil || !found {
			t.Errorf("%q was dropped by a value verdict (found=%v, err=%v)", name, found, err)
		}
	}
	// And the fact between two of them must survive as an edge, not vanish.
	// Before the veto, both endpoints were values and the value-to-value
	// rule dropped the fact outright.
	facts, err := st.FactsAbout(mustSlug(t, st, "issue-91"), false)
	if err != nil {
		t.Fatal(err)
	}
	var edge bool
	for _, f := range facts {
		if f.Dst != "" {
			edge = true
		}
	}
	if !edge {
		t.Errorf("issue-91 -> PR-402 was lost: %+v", facts)
	}
}

// The veto covers files and tickets and nothing else. A plain name the
// model wrongly types as a value is still dropped, and the only defence is
// the prompt. Pinned so the limit is a decision rather than a surprise.
func TestAValueVerdictOnAPlainNameIsStillHonoured(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-plain", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
	stats, err := Apply(st, ep, "", extract.Result{
		Entities: []extract.Ent{{Name: "childscribe-mobile", Type: "value"}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ValuesRejected != 1 {
		t.Errorf("ValuesRejected = %d, want 1", stats.ValuesRejected)
	}
	if _, found, _ := st.ResolveAlias("childscribe-mobile"); found {
		t.Error("expected the verdict to be honoured for a plain name")
	}
}

func TestNamesAnArtifactIsNarrow(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		// Defended: files and paths.
		{"internal/memory/resolve/declared.go", true},
		{"docs/MEMORY_AUDIT_2026-09-02.md", true},
		{"cmd/scry", true},
		{"queue/outbox.ts", true},
		{"schema.sql", true},
		// Defended: tickets and pull requests.
		{"issue-91", true},
		{"PR-402", true},
		{"GH#88", true},
		{"bug 1204", true},
		{"SCRY-17", true},

		// Left to the model and the lexical rules: these are the value
		// families the type exists to catch, and the veto must not save them.
		{"in progress", false},
		{"QUALITY_OK", false},
		{"46 GiB", false},
		{"main", false},
		{"think:false", false},
		{"require_approval ON", false},
		{"120-word floor", false},
		{"2026-09-03", false},
		{"hermes-ops", false},
		{"", false},
	}
	for _, c := range cases {
		if got := namesAnArtifact(c.name); got != c.want {
			t.Errorf("namesAnArtifact(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
