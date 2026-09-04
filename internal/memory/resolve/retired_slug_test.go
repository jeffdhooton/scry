package resolve

import (
	"errors"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestApply_RehomedSpellingCannotResurrectRetiredSlug(t *testing.T) {
	st := openTemp(t)
	putEntity(t, st, "obsolete", "Retired Verdict", "concept")
	putEntity(t, st, "target-service", "Target Service", "service", "obsolete")
	req := store.EntityRetirementRequest{
		Entity: "obsolete", Why: "reviewed value",
		RehomeAliases: []store.EntityRetirementAliasRehome{{Alias: "obsolete", Entity: "target-service", Why: "existing reviewed alias"}},
	}
	p, err := st.PreviewEntityRetirement(req)
	if err != nil || !p.Ready {
		t.Fatalf("preview=%+v err=%v", p, err)
	}
	req.Expected = p.Expected
	if _, err := st.RetireEntity(req); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	ep := store.Episode{ID: "punctuated-mention", OccurredAt: now, IngestedAt: now}
	stats, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: "Obsolete!", Type: "service"}}}, DefaultExclusive)
	if !errors.Is(err, store.ErrEntityRetired) {
		t.Fatalf("Apply = %+v, %v, want ErrEntityRetired", stats, err)
	}
	if _, err := st.GetEntity("obsolete"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("retired slug recreated: %v", err)
	}
	if _, err := st.GetEpisode(ep.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("failed ingestion committed episode: %v", err)
	}
	// The exact reviewed alias remains an identity, not value evidence.
	ep.ID = "reviewed-mention"
	_, err = Apply(st, ep, "", extract.Result{
		Entities: []extract.Ent{{Name: "Catalog", Type: "project"}},
		Facts:    []extract.Fct{{Src: "Catalog", Relation: "uses", Dst: "obsolete", Fact: "Catalog uses obsolete", Confidence: .9}},
	}, DefaultExclusive)
	if err != nil {
		t.Fatal(err)
	}
	facts := mustFacts(t, st, "catalog")
	if len(facts) != 1 || facts[0].Dst != "target-service" {
		t.Fatalf("reviewed alias lost identity routing: %+v", facts)
	}
}
