package resolve

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestCanonicalNameDoesNotDependOnSurvivorSlug(t *testing.T) {
	for _, typ := range []string{"tool", "concept", "project", "service", "machine"} {
		t.Run(typ, func(t *testing.T) {
			st := openTemp(t)
			at := time.Date(2026, 9, 5, 20, 0, 0, 0, time.UTC)
			owner := store.Entity{Slug: "retained-migration-identity", Name: "db/migrations/0160_task_evidence_rules.sql", Type: "tool", Description: "Reviewed SQL file identity", Aliases: []string{"original-migration-file"}, CreatedAt: at, LastSeen: at}
			if err := st.PutEntity(owner); err != nil {
				t.Fatal(err)
			}
			if err := st.PutEntity(store.Entity{Slug: "docket", Name: "docket", Type: "project"}); err != nil {
				t.Fatal(err)
			}
			invalid := at.Add(-time.Hour)
			old := store.Fact{Src: owner.Slug, Relation: RelPartOf, Dst: "docket", Fact: "Preserved historical assertion", ValidFrom: at.Add(-2 * time.Hour), InvalidAt: &invalid, Confidence: .8, Episodes: []string{"historical-source"}}
			if err := st.PutFact(old); err != nil {
				t.Fatal(err)
			}
			name := "db/migrations/0160-task-evidence-rules.sql"
			ep := store.Episode{ID: "canonical-name-" + typ, Source: "manual", SourceRef: "test", OccurredAt: at, IngestedAt: at}
			stats, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: name, Type: typ, Description: "Do not replace reviewed metadata"}}, Facts: []extract.Fct{{Src: name, Relation: RelPartOf, Dst: "docket", Fact: "The reviewed SQL file belongs to Docket.", Confidence: .95}}}, DefaultExclusive)
			if err != nil {
				t.Fatal(err)
			}
			if stats.EntitiesCreated != 0 {
				t.Fatalf("created another identity: %+v", stats)
			}
			got, err := st.GetEntity(owner.Slug)
			if err != nil || !reflect.DeepEqual(got, owner) {
				t.Fatalf("reviewed metadata changed: %+v, %v", got, err)
			}
			if _, err := st.GetEntity(store.Slugify(name)); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("natural-slug duplicate: %v", err)
			}
			facts := mustFacts(t, st, owner.Slug)
			if len(facts) != 2 {
				t.Fatalf("facts = %+v", facts)
			}
			preserved := false
			for _, fact := range facts {
				preserved = preserved || reflect.DeepEqual(fact, old)
				if fact.Src != owner.Slug {
					t.Fatalf("wrong endpoint: %+v", fact)
				}
			}
			if !preserved {
				t.Fatal("historical fact changed")
			}
		})
	}
}

func TestCanonicalNameExceptionDoesNotTrustCrossTypeAlias(t *testing.T) {
	st := openTemp(t)
	owner := store.Entity{Slug: "operations-project", Name: "Operations Repository", Type: "project", Aliases: []string{"Mini Host"}}
	if err := st.PutEntity(owner); err != nil {
		t.Fatal(err)
	}
	ep := store.Episode{ID: "alias-not-canonical", OccurredAt: time.Now()}
	_, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: "Mini Host", Type: "machine"}}}, DefaultExclusive)
	if !errors.Is(err, store.ErrAliasClaimed) {
		t.Fatalf("cross-type alias accepted: %v", err)
	}
	got, err := st.GetEntity(owner.Slug)
	if err != nil || !reflect.DeepEqual(got, owner) {
		t.Fatalf("owner changed: %+v, %v", got, err)
	}
	if exists, err := st.HasEpisode(ep.ID); err != nil || exists {
		t.Fatalf("rejected episode committed: %v, %v", exists, err)
	}
}
