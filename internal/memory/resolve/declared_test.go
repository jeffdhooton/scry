package resolve

import (
	"fmt"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestDeclaredValuesCollectsNamesAndExplicitValueAliases(t *testing.T) {
	got := DeclaredValues([]extract.Ent{
		{Name: "Loom", Type: "project"},
		{Name: "46 GiB", Type: "value"},
		{Name: "In Progress", Type: "value", Aliases: []string{"workflow-stage"}},
	})
	for _, want := range []string{"46 gib", "in progress"} {
		if !got[store.Normalize(want)] {
			t.Errorf("declared values missing %q: %v", want, got)
		}
	}
	if got[store.Normalize("Loom")] {
		t.Error("a project must not be collected as a value")
	}
	if !got[store.Normalize("workflow-stage")] {
		t.Error("an explicit value alias must be available to classify fact endpoints")
	}
}

func TestParsedShoutedStatusAliasesNeverBecomeRoutingKeys(t *testing.T) {
	for _, status := range []string{"DONE_WITH_CONCERNS", "DIRTY_WORKING_TREE"} {
		st := openTemp(t)
		for i := 0; i < 2; i++ {
			parsed, err := extract.ParseResult(fmt.Sprintf(`{"episode_summary":"alias attestation","entities":[{"name":"review concern handler","type":"service","description":"review service","aliases":[%q]}],"facts":[]}`, status))
			if err != nil {
				t.Fatal(err)
			}
			at := time.Unix(int64(100+i), 0).UTC()
			if _, err := Apply(st, store.Episode{ID: fmt.Sprintf("alias-%s-%d", status, i), Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}, "", parsed, nil); err != nil {
				t.Fatal(err)
			}
		}
		if owner, found, err := st.ResolveAlias(status); err != nil || found {
			t.Fatalf("shouted status %q became routing alias: owner=%q found=%v err=%v", status, owner, found, err)
		}
		parsed, err := extract.ParseResult(fmt.Sprintf(`{"episode_summary":"status report","entities":[{"name":"scry","type":"project","description":"memory system"}],"facts":[{"src":"scry","relation":"reports","dst":%q,"fact":"run status","confidence":0.9}]}`, status))
		if err != nil {
			t.Fatal(err)
		}
		at := time.Unix(103, 0).UTC()
		if _, err := Apply(st, store.Episode{ID: "report-" + status, Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}, "", parsed, nil); err != nil {
			t.Fatal(err)
		}
		facts := mustFacts(t, st, "scry")
		if len(facts) != 1 || facts[0].Dst != "" || facts[0].Value != status {
			t.Fatalf("shouted status %q did not remain an attribute: %+v", status, facts)
		}
	}
}

func TestParsedExplicitValueAliasClassifiesFactEndpoint(t *testing.T) {
	st := openTemp(t)
	parsed, err := extract.ParseResult(`{"episode_summary":"completion report","entities":[{"name":"scry","type":"project","description":"memory system"},{"name":"completion_state","type":"value","description":"run state","aliases":["current_completion"]}],"facts":[{"src":"scry","relation":"reports","dst":"current_completion","fact":"current completion was reported","confidence":0.9}]}`)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Unix(110, 0).UTC()
	if _, err := Apply(st, store.Episode{ID: "value-alias", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}, "", parsed, nil); err != nil {
		t.Fatal(err)
	}
	if _, found, err := st.ResolveAlias("current_completion"); err != nil || found {
		t.Fatalf("value alias became entity: found=%v err=%v", found, err)
	}
	facts := mustFacts(t, st, "scry")
	if len(facts) != 1 || facts[0].Dst != "" || facts[0].Value != "current_completion" {
		t.Fatalf("value alias endpoint did not become attribute: %+v", facts)
	}
}

func TestParsedValueAliasCannotEraseOrdinaryFallbackIdentity(t *testing.T) {
	st := openTemp(t)
	parsed, err := extract.ParseResult(`{"episode_summary":"CLI inventory","entities":[{"name":"completion_state","type":"value","description":"run state","aliases":["SQLite CLI"]},{"name":"SQLite CLI","description":"local database command-line tool"}],"facts":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Entities) != 2 || !parsed.Entities[1].TypeFallback {
		t.Fatalf("test requires parser fallback identity: %+v", parsed.Entities)
	}
	at := time.Unix(120, 0).UTC()
	if _, err := Apply(st, store.Episode{ID: "fallback-alias-conflict", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}, "", parsed, nil); err != nil {
		t.Fatal(err)
	}
	entity, err := st.GetEntity("sqlite-cli")
	if err != nil || entity.Name != "SQLite CLI" || entity.Type != "concept" {
		t.Fatalf("ordinary fallback identity was erased: entity=%+v err=%v", entity, err)
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

func TestContextBearingAmbiguousStatusPairs(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	ep := store.Episode{ID: "ep-ambiguous-status", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
	res := extract.Result{
		Entities: []extract.Ent{
			{Name: "validation_failed", Type: "value", Description: "the CI validation status"},
			{Name: "DONE_WITH_CONCERNS", Type: "value", Description: "the review verdict"},
			{Name: "user_login_failed", Type: "concept", Description: "a durable auth event identifier"},
			{Name: "PYTHON_ARGCOMPLETE_OK", Type: "concept", Description: "a durable protocol marker identifier"},
			{Name: "Ready Player One", Type: "concept", Description: "a cataloged novel"},
			{Name: "46 GiB", Type: "concept", Description: "a memory measurement mislabeled as an identity"},
			{Name: "on feature/example", Type: "concept", Description: "a branch phrase mislabeled as an identity"},
			{Name: "argcomplete", Type: "tool", Description: "the shell completion tool"},
		},
		Facts: []extract.Fct{{
			Src:        "PYTHON_ARGCOMPLETE_OK",
			Relation:   "part_of",
			Dst:        "argcomplete",
			Fact:       "PYTHON_ARGCOMPLETE_OK is an argcomplete protocol marker",
			Confidence: 0.95,
		}},
	}
	if _, err := Apply(st, ep, "", res, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"validation_failed", "DONE_WITH_CONCERNS"} {
		if _, found, err := st.ResolveAlias(name); err != nil || found {
			t.Errorf("context-declared status %q became an entity: found=%v err=%v", name, found, err)
		}
	}
	for _, name := range []string{"46 GiB", "on feature/example"} {
		if _, found, err := st.ResolveAlias(name); err != nil || found {
			t.Errorf("hard value %q bypassed its veto: found=%v err=%v", name, found, err)
		}
	}
	for _, name := range []string{"user_login_failed", "PYTHON_ARGCOMPLETE_OK", "Ready Player One"} {
		if _, found, err := st.ResolveAlias(name); err != nil || !found {
			t.Errorf("context-declared identifier %q was rejected: found=%v err=%v value=%v ephemeral=%v generic=%v", name, found, err, IsValueName(name), isEphemeralName(name), isGenericEntityName(name))
		}
	}
	facts := mustFacts(t, st, mustSlug(t, st, "PYTHON_ARGCOMPLETE_OK"))
	if len(facts) != 1 || facts[0].Dst != mustSlug(t, st, "argcomplete") {
		t.Errorf("context-declared identifier edge was converted or lost: %+v", facts)
	}
}

func TestFallbackOrUnknownTypesCannotCreateSuspiciousIdentities(t *testing.T) {
	for i, entityJSON := range []string{
		`{"name":"QUALITY_OK","description":"the run verdict"}`,
		`{"name":"QUALITY_OK","type":"status","description":"the run verdict"}`,
		`{"name":"validation_failed","description":"the run verdict"}`,
		`{"name":"validation_failed","type":"status","description":"the run verdict"}`,
		`{"name":"DONE_WITH_CONCERNS","description":"the review verdict"}`,
		`{"name":"DONE_WITH_CONCERNS","type":"status","description":"the review verdict"}`,
	} {
		st := openTemp(t)
		parsed, err := extract.ParseResult(`{"episode_summary":"quality run","entities":[` + entityJSON + `],"facts":[]}`)
		if err != nil {
			t.Fatal(err)
		}
		ep := store.Episode{ID: fmt.Sprintf("ep-fallback-%d", i), Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
		if _, err := Apply(st, ep, "", parsed, nil); err != nil {
			t.Fatal(err)
		}
		name := parsed.Entities[0].Name
		if _, found, err := st.ResolveAlias(name); err != nil || found {
			t.Errorf("fallback enum status became an entity: found=%v err=%v parsed=%+v", found, err, parsed.Entities)
		}
	}
	// Raw RPC callers can bypass ParseResult, so an invented direct type must
	// also fail the contextual identity allowlist.
	for i, name := range []string{"QUALITY_OK", "validation_failed", "DONE_WITH_CONCERNS"} {
		st := openTemp(t)
		ep := store.Episode{ID: fmt.Sprintf("ep-direct-unknown-%d", i), Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
		if _, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: name, Type: "status"}}}, nil); err != nil {
			t.Fatal(err)
		}
		if _, found, _ := st.ResolveAlias(name); found {
			t.Errorf("direct invented type created %q", name)
		}
	}
}

func TestUndeclaredStatusDestinationsRemainAttributes(t *testing.T) {
	cases := []struct{ status, relation string }{
		{"validation_failed", "status"},
		{"DONE_WITH_CONCERNS", "has_outcome"},
		{"validation_failed", "produces"},
		{"DONE_WITH_CONCERNS", "reports"},
	}
	for i, tc := range cases {
		st := openTemp(t)
		ep := store.Episode{ID: fmt.Sprintf("ep-undeclared-%d", i), Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
		result := extract.Result{
			Entities: []extract.Ent{{Name: "scry", Type: "project"}},
			Facts:    []extract.Fct{{Src: "scry", Relation: tc.relation, Dst: tc.status, Fact: "Scry reports " + tc.status, Confidence: .9}},
		}
		if _, err := Apply(st, ep, "", result, nil); err != nil {
			t.Fatal(err)
		}
		if _, found, _ := st.ResolveAlias(tc.status); found {
			t.Errorf("undeclared status destination %q became an entity for %s", tc.status, tc.relation)
		}
		facts := mustFacts(t, st, "scry")
		if len(facts) != 1 || facts[0].Dst != "" || facts[0].Value != tc.status {
			t.Errorf("undeclared status %q was not preserved as an attribute: %+v", tc.status, facts)
		}
	}
}

func TestMalformedOrdinaryIdentityFallsBackWithoutBecomingAValue(t *testing.T) {
	for i, name := range []string{"SQLite CLI", "OpenSSH client", "Z shell"} {
		raw := fmt.Sprintf(`{"episode_summary":"tool mention","entities":[{"name":%q,"description":"a durable tool identity"}],"facts":[]}`, name)
		result, err := extract.ParseResult(raw)
		if err != nil {
			t.Fatal(err)
		}
		st := openTemp(t)
		ep := store.Episode{ID: fmt.Sprintf("ep-ordinary-fallback-%d", i), Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
		if _, err := Apply(st, ep, "", result, nil); err != nil {
			t.Fatal(err)
		}
		if _, found, err := st.ResolveAlias(name); err != nil || !found {
			t.Errorf("ordinary fallback identity %q was discarded: found=%v err=%v", name, found, err)
		}
	}
}

func TestEstablishedIdentitySurvivesValueVerdictWithItsEdge(t *testing.T) {
	st := openTemp(t)
	at := time.Now()
	first := store.Episode{ID: "ep-marker-first", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
	if _, err := Apply(st, first, "", extract.Result{Entities: []extract.Ent{
		{Name: "PYTHON_ARGCOMPLETE_OK", Type: "concept"}, {Name: "argcomplete", Type: "tool"},
	}}, nil); err != nil {
		t.Fatal(err)
	}
	second := store.Episode{ID: "ep-marker-second", Source: "manual", SourceRef: "x", OccurredAt: at.Add(time.Minute), IngestedAt: at.Add(time.Minute)}
	result := extract.Result{
		Entities: []extract.Ent{{Name: "PYTHON_ARGCOMPLETE_OK", Type: "value"}, {Name: "argcomplete", Type: "tool"}},
		Facts:    []extract.Fct{{Src: "PYTHON_ARGCOMPLETE_OK", Relation: "part_of", Dst: "argcomplete", Fact: "PYTHON_ARGCOMPLETE_OK is an argcomplete protocol marker", Confidence: .9}},
	}
	if _, err := Apply(st, second, "", result, nil); err != nil {
		t.Fatal(err)
	}
	marker := mustSlug(t, st, "PYTHON_ARGCOMPLETE_OK")
	facts := mustFacts(t, st, marker)
	if len(facts) != 1 || facts[0].Dst != mustSlug(t, st, "argcomplete") {
		t.Errorf("established identity survived but its new edge was demoted: %+v", facts)
	}
	entity, err := st.GetEntity(marker)
	if err != nil || entity.Type != "concept" {
		t.Errorf("value verdict changed established identity metadata: entity=%+v err=%v", entity, err)
	}
}

func TestUndeclaredEstablishedStatusIdentityKeepsItsEdge(t *testing.T) {
	for i, markerName := range []string{"PYTHON_ARGCOMPLETE_OK", "PROTOCOL_NEGOTIATION_READY"} {
		st := openTemp(t)
		at := time.Now()
		first := store.Episode{ID: fmt.Sprintf("ep-undeclared-marker-first-%d", i), Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
		if _, err := Apply(st, first, "", extract.Result{Entities: []extract.Ent{
			{Name: markerName, Type: "concept"}, {Name: "protocol-catalog", Type: "project"},
		}}, nil); err != nil {
			t.Fatal(err)
		}
		second := store.Episode{ID: fmt.Sprintf("ep-undeclared-marker-second-%d", i), Source: "manual", SourceRef: "x", OccurredAt: at.Add(time.Minute), IngestedAt: at.Add(time.Minute)}
		result := extract.Result{
			Entities: []extract.Ent{{Name: "protocol-catalog", Type: "project"}},
			Facts:    []extract.Fct{{Src: markerName, Relation: "part_of", Dst: "protocol-catalog", Fact: markerName + " belongs to the protocol catalog", Confidence: .9}},
		}
		if _, err := Apply(st, second, "", result, nil); err != nil {
			t.Fatal(err)
		}
		marker := mustSlug(t, st, markerName)
		facts := mustFacts(t, st, marker)
		if len(facts) != 1 || facts[0].Dst != mustSlug(t, st, "protocol-catalog") || facts[0].Value != "" {
			t.Errorf("established undeclared marker %q was demoted or inverted: %+v", markerName, facts)
		}
	}
}

func TestUndeclaredFactEndpointPrefersExactIdentityOverStaleAlias(t *testing.T) {
	st := openTemp(t)
	for _, entity := range []store.Entity{
		{Slug: "zephyr", Name: "Zephyr", Type: "service"},
		{Slug: "interceptor", Name: "Interceptor", Type: "service"},
		{Slug: "protocol-catalog", Name: "Protocol Catalog", Type: "project"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.ClaimAlias("Zephyr", "interceptor"); err != nil {
		t.Fatal(err)
	}
	ep := store.Episode{ID: "ep-stale-alias-fact", Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
	result := extract.Result{
		Entities: []extract.Ent{{Name: "Protocol Catalog", Type: "project"}},
		Facts:    []extract.Fct{{Src: "Zephyr", Relation: "part_of", Dst: "Protocol Catalog", Fact: "Zephyr belongs to the protocol catalog", Confidence: .9}},
	}
	if _, err := Apply(st, ep, "", result, nil); err != nil {
		t.Fatal(err)
	}
	if facts := mustFacts(t, st, "zephyr"); len(facts) != 1 || facts[0].Dst != "protocol-catalog" {
		t.Errorf("exact identity lost its fact to stale alias owner: %+v", facts)
	}
	if facts := mustFacts(t, st, "interceptor"); len(facts) != 0 {
		t.Errorf("stale alias owner hijacked exact identity fact: %+v", facts)
	}
}

func TestOwnedStatusShapedAliasRemainsAnIdentityInFacts(t *testing.T) {
	st := openTemp(t)
	for _, entity := range []store.Entity{
		{Slug: "canonical-transport-marker", Name: "Canonical Transport Marker", Type: "concept", Aliases: []string{"TRANSPORT_NEGOTIATION_OK"}},
		{Slug: "protocol-catalog", Name: "Protocol Catalog", Type: "project"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	ep := store.Episode{ID: "ep-owned-status-alias", Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
	result := extract.Result{
		Entities: []extract.Ent{{Name: "Protocol Catalog", Type: "project"}},
		Facts:    []extract.Fct{{Src: "TRANSPORT_NEGOTIATION_OK", Relation: "part_of", Dst: "Protocol Catalog", Fact: "the marker belongs to the protocol catalog", Confidence: .9}},
	}
	if _, err := Apply(st, ep, "", result, nil); err != nil {
		t.Fatal(err)
	}
	if facts := mustFacts(t, st, "canonical-transport-marker"); len(facts) != 1 || facts[0].Dst != "protocol-catalog" || facts[0].Value != "" {
		t.Errorf("owned status-shaped alias was demoted: %+v", facts)
	}
}

func TestSupersedesResolvesEstablishedStatusShapedIdentities(t *testing.T) {
	for _, markerAsSource := range []bool{true, false} {
		st := openTemp(t)
		at := time.Now().Add(-time.Hour)
		for _, entity := range []store.Entity{
			{Slug: "transport-negotiation-ok", Name: "TRANSPORT_NEGOTIATION_OK", Type: "concept"},
			{Slug: "protocol-catalog", Name: "Protocol Catalog", Type: "project"},
			{Slug: "replacement", Name: "Replacement", Type: "concept"},
		} {
			if err := st.PutEntity(entity); err != nil {
				t.Fatal(err)
			}
		}
		src, dst := "TRANSPORT_NEGOTIATION_OK", "Protocol Catalog"
		if !markerAsSource {
			src, dst = dst, src
		}
		old := store.Fact{Src: store.Slugify(src), Relation: "part_of", Dst: store.Slugify(dst), Fact: "old marker relationship", ValidFrom: at, Confidence: .9}
		if err := st.PutFact(old); err != nil {
			t.Fatal(err)
		}
		ep := store.Episode{ID: fmt.Sprintf("ep-supersedes-status-%v", markerAsSource), Source: "manual", SourceRef: "x", OccurredAt: at.Add(time.Minute), IngestedAt: at.Add(time.Minute)}
		result := extract.Result{
			Entities: []extract.Ent{{Name: "Replacement", Type: "concept"}},
			Facts: []extract.Fct{{Src: "Replacement", Relation: "related_to", Dst: "Protocol Catalog", Fact: "replacement observation", Confidence: .9,
				Supersedes: &extract.SupRef{Src: src, Relation: "part_of", Dst: dst}}},
		}
		stats, err := Apply(st, ep, "", result, nil)
		if err != nil {
			t.Fatal(err)
		}
		if stats.FactsInvalidated != 1 {
			t.Errorf("status-shaped supersedes markerAsSource=%v invalidated=%d", markerAsSource, stats.FactsInvalidated)
		}
		facts, err := st.FactsFrom(old.Src, true)
		if err != nil || len(facts) != 1 || facts[0].InvalidAt == nil {
			t.Errorf("old edge remained current markerAsSource=%v facts=%+v err=%v", markerAsSource, facts, err)
		}
	}
}

func TestSupersedesResolvesContextDeclaredAmbiguousValues(t *testing.T) {
	for i, value := range []string{"validation_failed", "DONE_WITH_CONCERNS"} {
		st := openTemp(t)
		at := time.Now().Add(-time.Hour)
		first := store.Episode{ID: fmt.Sprintf("ep-value-first-%d", i), Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
		initial := extract.Result{
			Entities: []extract.Ent{{Name: "Scry", Type: "project"}, {Name: value, Type: "value"}},
			Facts:    []extract.Fct{{Src: "Scry", Relation: "reports", Dst: value, Fact: "old status", Confidence: .9}},
		}
		if _, err := Apply(st, first, "", initial, nil); err != nil {
			t.Fatal(err)
		}
		ep := store.Episode{ID: fmt.Sprintf("ep-supersedes-value-%d", i), Source: "manual", SourceRef: "x", OccurredAt: at.Add(time.Minute), IngestedAt: at.Add(time.Minute)}
		result := extract.Result{
			Entities: []extract.Ent{{Name: "Replacement", Type: "concept"}, {Name: value, Type: "value"}},
			Facts: []extract.Fct{{Src: "Replacement", Relation: "related_to", Dst: "Scry", Fact: "replacement observation", Confidence: .9,
				Supersedes: &extract.SupRef{Src: "Scry", Relation: "reports", Dst: value}}},
		}
		stats, err := Apply(st, ep, "", result, nil)
		if err != nil {
			t.Fatal(err)
		}
		facts, err := st.FactsFrom("scry", true)
		if stats.FactsInvalidated != 1 || err != nil || len(facts) != 1 || facts[0].InvalidAt == nil {
			t.Errorf("declared value %q was not superseded: stats=%+v facts=%+v err=%v", value, stats, facts, err)
		}
	}
}

func TestAmbiguousStatusAliasCannotPoisonLaterFactRouting(t *testing.T) {
	st := openTemp(t)
	for i := 0; i < 2; i++ {
		at := time.Now().Add(time.Duration(i) * time.Minute)
		ep := store.Episode{ID: fmt.Sprintf("ep-status-alias-%d", i), Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
		result := extract.Result{Entities: []extract.Ent{{Name: "validation failed handler", Type: "service", Aliases: []string{"validation_failed"}}}}
		if _, err := Apply(st, ep, "", result, nil); err != nil {
			t.Fatal(err)
		}
	}
	if owner, found, err := st.ResolveAlias("validation_failed"); err != nil || found {
		t.Fatalf("ambiguous status alias became a routing key: owner=%q found=%v err=%v", owner, found, err)
	}
	at := time.Now().Add(3 * time.Minute)
	ep := store.Episode{ID: "ep-status-alias-fact", Source: "manual", SourceRef: "x", OccurredAt: at, IngestedAt: at}
	result := extract.Result{
		Entities: []extract.Ent{{Name: "Scry", Type: "project"}},
		Facts:    []extract.Fct{{Src: "Scry", Relation: "reports", Dst: "validation_failed", Fact: "Scry reports validation failed", Confidence: .9}},
	}
	if _, err := Apply(st, ep, "", result, nil); err != nil {
		t.Fatal(err)
	}
	facts := mustFacts(t, st, "scry")
	if len(facts) != 1 || facts[0].Dst != "" || facts[0].Value != "validation_failed" {
		t.Errorf("ambiguous status alias poisoned fact routing: %+v", facts)
	}
}

func TestArtifactVetoPrecedesBranchAndStatusSpelling(t *testing.T) {
	st := openTemp(t)
	names := []string{"/usr/bin/true", "/usr/bin/false", "/usr/bin/open", "/usr/bin/head", "/usr/bin/yes", "release/mac-arm64"}
	entities := make([]extract.Ent, 0, len(names))
	for _, name := range names {
		entities = append(entities, extract.Ent{Name: name, Type: "tool", Description: "a real executable or directory identity"})
	}
	ep := store.Episode{ID: "ep-artifact-boundary", Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
	if _, err := Apply(st, ep, "", extract.Result{Entities: entities}, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if _, found, err := st.ResolveAlias(name); err != nil || !found {
			t.Errorf("artifact identity %q was rejected: found=%v err=%v", name, found, err)
		}
	}
}

func TestExplicitValueBranchDoesNotUseRelativePathArtifactVeto(t *testing.T) {
	for i, branch := range []string{"feature/example", "fix/123-thing", "release/1.2"} {
		st := openTemp(t)
		ep := store.Episode{ID: fmt.Sprintf("ep-value-branch-%d", i), Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
		result := extract.Result{
			Entities: []extract.Ent{{Name: "scry", Type: "project"}, {Name: branch, Type: "value"}},
			Facts:    []extract.Fct{{Src: "scry", Relation: "status", Dst: branch, Fact: "Scry is on " + branch, Confidence: .9}},
		}
		if _, err := Apply(st, ep, "", result, nil); err != nil {
			t.Fatal(err)
		}
		if _, found, _ := st.ResolveAlias(branch); found {
			t.Errorf("explicit value branch %q became an entity", branch)
		}
		facts := mustFacts(t, st, "scry")
		if len(facts) != 1 || facts[0].Dst != "" || facts[0].Value != branch {
			t.Errorf("branch %q was not preserved as an attribute: %+v", branch, facts)
		}
	}
}

func TestAbsoluteArtifactsSurviveValueVerdict(t *testing.T) {
	st := openTemp(t)
	names := []string{"/usr/bin/sqlite3", "/usr/bin/ssh", "/usr/bin/zipinfo", "/usr/sbin/diskutil", "/usr/bin/swift"}
	entities := make([]extract.Ent, 0, len(names))
	for _, name := range names {
		entities = append(entities, extract.Ent{Name: name, Type: "value", Description: "an executable path"})
	}
	ep := store.Episode{ID: "ep-absolute-artifacts", Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
	if _, err := Apply(st, ep, "", extract.Result{Entities: entities}, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		slug := mustSlug(t, st, name)
		entity, err := st.GetEntity(slug)
		if err != nil || entity.Type != "concept" {
			t.Errorf("absolute artifact %q was not preserved neutrally: entity=%+v err=%v", name, entity, err)
		}
	}
}

func TestDeclaredValueIgnoresStaleAliasOwner(t *testing.T) {
	st := openTemp(t)
	for _, entity := range []store.Entity{
		{Slug: "wrong-owner", Name: "Wrong Owner", Type: "concept", Aliases: []string{"validation_failed"}},
		{Slug: "scry", Name: "Scry", Type: "project"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	ep := store.Episode{ID: "ep-stale-value-alias", Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
	result := extract.Result{
		Entities: []extract.Ent{{Name: "validation_failed", Type: "value", Description: "the run status"}},
		Facts:    []extract.Fct{{Src: "scry", Relation: "status", Dst: "validation_failed", Fact: "Scry validation failed", Confidence: .9}},
	}
	if _, err := Apply(st, ep, "", result, nil); err != nil {
		t.Fatalf("stale alias owner aborted a correctly classified episode: %v", err)
	}
	if _, err := st.GetEntity("validation-failed"); err == nil {
		t.Error("correct value verdict created a status identity")
	}
	facts := mustFacts(t, st, "scry")
	if len(facts) != 1 || facts[0].Dst != "" || facts[0].Value != "validation_failed" {
		t.Errorf("correct value fact was not preserved as an attribute: %+v", facts)
	}
}

func TestValueAliasCannotPoisonSeparateExactIdentity(t *testing.T) {
	st := openTemp(t)
	ep := store.Episode{ID: "ep-value-alias-conflict", Source: "manual", SourceRef: "x", OccurredAt: time.Now(), IngestedAt: time.Now()}
	result := extract.Result{
		Entities: []extract.Ent{
			{Name: "completion_state", Type: "value", Aliases: []string{"PYTHON_ARGCOMPLETE_OK"}},
			{Name: "PYTHON_ARGCOMPLETE_OK", Type: "concept", Description: "a durable protocol marker"},
			{Name: "argcomplete", Type: "tool"},
		},
		Facts: []extract.Fct{{Src: "PYTHON_ARGCOMPLETE_OK", Relation: "part_of", Dst: "argcomplete", Fact: "the marker belongs to argcomplete", Confidence: .9}},
	}
	if _, err := Apply(st, ep, "", result, nil); err != nil {
		t.Fatal(err)
	}
	marker := mustSlug(t, st, "PYTHON_ARGCOMPLETE_OK")
	facts := mustFacts(t, st, marker)
	if len(facts) != 1 || facts[0].Dst != mustSlug(t, st, "argcomplete") {
		t.Errorf("value alias poisoned exact identity edge: %+v", facts)
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
		{"/tmp", true},
		{"/etc", true},
		{"/var", true},
		{"/bin", true},
		{"queue/outbox.ts", true},
		{"schema.sql", true},
		// But a code position points into a file rather than naming one.
		{"queue/outbox.ts:170-173", false},
		{"exception_queue.py:61", false},
		{"memory-graph.test.ts:39,43-57", false},
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
