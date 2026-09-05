package resolve

import (
	"errors"
	"reflect"
	"strings"
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

func TestCanonicalNameRefusesRetainedHomonyms(t *testing.T) {
	st := openTemp(t)
	project := store.Entity{Slug: "atlas-project-id", Name: "Atlas", Type: "project"}
	machine := store.Entity{Slug: "atlas-machine-id", Name: "Atlas", Type: "machine"}
	if err := st.PutEntity(project); err != nil {
		t.Fatal(err)
	}
	// Explicitly construct legacy conflicting ownership, not an ordinary write.
	if err := st.ClaimAlias("Atlas", machine.Slug); err != nil {
		t.Fatal(err)
	}
	if err := st.PutEntity(machine); err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimAlias("Atlas", project.Slug); err != nil {
		t.Fatal(err)
	}
	beforeEntities, err := st.Entities()
	if err != nil {
		t.Fatal(err)
	}
	beforeClaims, err := st.AliasClaims()
	if err != nil {
		t.Fatal(err)
	}
	ep := store.Episode{ID: "ambiguous-retained-homonyms", OccurredAt: time.Now()}
	_, err = Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: "Atlas", Type: "machine"}}, Facts: []extract.Fct{{Src: "Atlas", Relation: RelRunsOn, Dst: "linux", Fact: "The Atlas machine runs Linux.", Confidence: 1}}}, DefaultExclusive)
	if !errors.Is(err, store.ErrAliasClaimed) {
		t.Fatalf("ambiguous routing succeeded: %v", err)
	}
	afterEntities, err := st.Entities()
	if err != nil {
		t.Fatal(err)
	}
	afterClaims, err := st.AliasClaims()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeEntities, afterEntities) || !reflect.DeepEqual(beforeClaims, afterClaims) {
		t.Fatal("ambiguous episode changed identity or routing")
	}
	if facts, err := st.AllFacts(); err != nil || len(facts) != 0 {
		t.Fatalf("ambiguous episode wrote facts: %+v, %v", facts, err)
	}
	if exists, err := st.HasEpisode(ep.ID); err != nil || exists {
		t.Fatalf("ambiguous episode committed: %v, %v", exists, err)
	}
}

func TestCanonicalNameDoesNotPromoteGenericReference(t *testing.T) {
	for _, name := range []string{"the machine", "the_machine", "the-machine", "this box"} {
		t.Run(name, func(t *testing.T) {
			st := openTemp(t)
			owner := store.Entity{Slug: "cedar-retained", Name: name, Type: "project"}
			if err := st.PutEntity(owner); err != nil {
				t.Fatal(err)
			}
			ep := store.Episode{ID: "generic-canonical", OccurredAt: time.Now()}
			stats, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: name, Type: "machine", Description: "Must not fill generic owner metadata"}}}, DefaultExclusive)
			// "this box" is already rejected by the earlier generic-entity
			// gate. The other legacy spellings must retain conflict refusal.
			if name != "this box" && !errors.Is(err, store.ErrAliasClaimed) {
				t.Fatalf("generic canonical accepted: %v", err)
			}
			if stats.EntitiesUpdated != 0 || stats.EntitiesCreated != 0 {
				t.Fatalf("generic reference mutated identities: %+v", stats)
			}
			got, err := st.GetEntity(owner.Slug)
			if err != nil || !reflect.DeepEqual(got, owner) {
				t.Fatalf("generic owner mutated: %+v, %v", got, err)
			}
		})
	}
}

func TestCanonicalNameDeterminerNormalization(t *testing.T) {
	for _, canonical := range []string{"our own machine", "this physical host"} {
		for _, separator := range []string{" ", "_", "-"} {
			mention := strings.ReplaceAll(canonical, " ", separator)
			t.Run(mention, func(t *testing.T) {
				st := openTemp(t)
				owner := store.Entity{Slug: "retained-generic-owner", Name: canonical, Type: "project"}
				if err := st.PutEntity(owner); err != nil {
					t.Fatal(err)
				}
				claims, err := st.AliasClaims()
				if err != nil {
					t.Fatal(err)
				}
				ep := store.Episode{ID: "determiner-normalization", OccurredAt: time.Now()}
				_, err = Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: mention, Type: "machine", Description: "Must not fill project metadata"}}}, DefaultExclusive)
				// A spaced phrase may be skipped by the earlier generic-name
				// gate; normalized variants must not bypass the type refusal.
				if (separator != " " || err != nil) && !errors.Is(err, store.ErrAliasClaimed) {
					t.Fatalf("generic canonical mention accepted: %v", err)
				}
				refused := err != nil
				got, err := st.GetEntity(owner.Slug)
				if err != nil || !reflect.DeepEqual(got, owner) {
					t.Fatalf("owner changed: %+v, %v", got, err)
				}
				afterClaims, err := st.AliasClaims()
				if err != nil || !reflect.DeepEqual(claims, afterClaims) {
					t.Fatalf("claims changed: %+v, %v", afterClaims, err)
				}
				if exists, err := st.HasEpisode(ep.ID); err != nil || (refused && exists) {
					t.Fatalf("refused episode committed: %v, %v", exists, err)
				}
			})
		}
	}
}
