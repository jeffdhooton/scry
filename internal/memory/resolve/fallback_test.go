package resolve

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func seedFallback(t *testing.T, st *store.Store, at time.Time, invalid *time.Time) store.Fact {
	t.Helper()
	for _, name := range []string{"scry", "hermes-ops"} {
		if err := st.PutEntity(store.Entity{Slug: name, Name: name, Type: "project"}); err != nil {
			t.Fatal(err)
		}
	}
	f := store.Fact{Src: "scry", Dst: "hermes-ops", Relation: Fallback, RawRelation: "measured", Fact: "An earlier independent measurement.", ValidFrom: at, InvalidAt: invalid, Confidence: .8, Episodes: []string{"original"}}
	if err := st.PutFact(f); err != nil {
		t.Fatal(err)
	}
	return f
}

func routingInput() extract.Fct {
	return extract.Fct{Src: "scry", Dst: "hermes-ops", Relation: "aliases_index_to", Fact: "A routing observation, not identity or a measurement.", Confidence: .9}
}

func TestApplyFallbackPreservesDifferentStatements(t *testing.T) {
	for _, offset := range []time.Duration{-time.Hour, time.Hour} {
		t.Run(offset.String(), func(t *testing.T) {
			st := openTemp(t)
			at := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
			old := seedFallback(t, st, at, nil)
			input := routingInput()
			stats := applyOne(t, st, "routing", at.Add(offset), input)
			if stats.FactsAdded != 1 || stats.FactsMerged != 0 {
				t.Fatalf("stats: %+v", stats)
			}
			facts := mustFacts(t, st, "scry")
			if len(facts) != 2 {
				t.Fatalf("lost a statement: %+v", facts)
			}
			for _, f := range facts {
				if f.Fact == old.Fact {
					if !reflect.DeepEqual(f, old) {
						t.Errorf("old changed: %+v", f)
					}
					continue
				}
				if f.Fact != input.Fact || f.RawRelation != input.Relation || f.Relation != Fallback || !f.ValidFrom.Equal(at.Add(offset)) || !reflect.DeepEqual(f.Episodes, []string{"routing"}) {
					t.Errorf("incoming lost: %+v", f)
				}
			}
			stats = applyOne(t, st, "routing-again", at.Add(2*time.Hour), input)
			if stats.FactsMerged != 1 || stats.FactsAdded != 0 {
				t.Fatalf("exact restatement: %+v", stats)
			}
			if got := mustFacts(t, st, "scry"); len(got) != 2 {
				t.Fatalf("restatement duplicated: %+v", got)
			}
		})
	}
}

func TestApplyFallbackKeyCollisionRollsBackWholeEpisode(t *testing.T) {
	for _, historical := range []bool{false, true} {
		t.Run(map[bool]string{false: "current", true: "historical"}[historical], func(t *testing.T) {
			st := openTemp(t)
			at := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
			var invalid *time.Time
			if historical {
				end := at.Add(time.Minute)
				invalid = &end
			}
			old := seedFallback(t, st, at, invalid)
			before, err := st.Entities()
			if err != nil {
				t.Fatal(err)
			}
			ep := store.Episode{ID: "conflicting", OccurredAt: at, IngestedAt: at}
			stats, err := Apply(st, ep, "", extract.Result{Entities: []extract.Ent{{Name: "Additional Project", Type: "project"}}, Facts: []extract.Fct{routingInput()}}, DefaultExclusive)
			if !errors.Is(err, store.ErrFactConflict) || stats != (Stats{}) {
				t.Fatalf("collision should refuse atomically: %+v %v", stats, err)
			}
			after, err := st.Entities()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("entity work escaped failed episode")
			}
			facts := mustFacts(t, st, "scry")
			if !reflect.DeepEqual(facts, []store.Fact{old}) {
				t.Fatalf("fact history changed: %+v", facts)
			}
			present, err := st.HasEpisode(ep.ID)
			if err != nil || present {
				t.Fatalf("failed episode marked complete: %v %v", present, err)
			}
		})
	}
}

func TestApplyFallbackSameEpisodeDuplicateAndConflict(t *testing.T) {
	for _, identical := range []bool{true, false} {
		st := openTemp(t)
		at := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
		input := routingInput()
		second := input
		if !identical {
			second.Fact = "A different routing assertion."
		}
		ep := store.Episode{ID: "same-episode", OccurredAt: at, IngestedAt: at}
		_, err := Apply(st, ep, "", extract.Result{Facts: []extract.Fct{input, second}}, DefaultExclusive)
		if identical {
			if err != nil {
				t.Fatal(err)
			}
			facts := mustFacts(t, st, "scry")
			if len(facts) != 1 || facts[0].Fact != input.Fact {
				t.Fatalf("duplicate: %+v", facts)
			}
		} else {
			if !errors.Is(err, store.ErrFactConflict) {
				t.Fatalf("distinct same-key statements: %v", err)
			}
			facts, e := st.AllFacts()
			if e != nil || len(facts) != 0 {
				t.Fatalf("partial episode: %+v %v", facts, e)
			}
		}
	}
}

func TestApplyFallbackBackfillDoesNotOverwriteHistory(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	end := at.Add(time.Minute)
	seedFallback(t, st, at, &end)
	input := routingInput()
	applyOne(t, st, "newer-routing", at.Add(time.Hour), input)
	before, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	ep := store.Episode{ID: "backfill", OccurredAt: at, IngestedAt: at}
	_, err = Apply(st, ep, "", extract.Result{Facts: []extract.Fct{input}}, DefaultExclusive)
	if !errors.Is(err, store.ErrFactConflict) {
		t.Fatalf("backfill collision: %v", err)
	}
	after, err := st.AllFacts()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("history changed: %+v %v", after, err)
	}
}

func TestApplyFallbackSupersedesRequiresUnambiguousRawReference(t *testing.T) {
	for _, raw := range []string{"aliases_index_to", "unrecognized_relation"} {
		t.Run(raw, func(t *testing.T) {
			st := openTemp(t)
			at := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
			input := routingInput()
			applyOne(t, st, "routing-one", at, input)
			input.Fact = "A different routing assertion."
			applyOne(t, st, "routing-two", at.Add(time.Hour), input)
			before, err := st.AllFacts()
			if err != nil {
				t.Fatal(err)
			}
			ep := store.Episode{ID: "supersession", OccurredAt: at.Add(2 * time.Hour), IngestedAt: at.Add(2 * time.Hour)}
			f := extract.Fct{Src: "scry", Relation: "fixes", Dst: "hermes-ops", Fact: "A later fix.", Confidence: .9, Supersedes: &extract.SupRef{Src: "scry", Relation: raw, Dst: "hermes-ops"}}
			stats, err := Apply(st, ep, "", extract.Result{Facts: []extract.Fct{f}}, DefaultExclusive)
			if raw == "aliases_index_to" {
				if !errors.Is(err, store.ErrFactConflict) {
					t.Fatalf("ambiguous reference accepted: %+v %v", stats, err)
				}
				after, e := st.AllFacts()
				if e != nil || !reflect.DeepEqual(before, after) {
					t.Fatalf("ambiguous supersession changed state: %+v %v", after, e)
				}
			} else if err != nil || stats.FactsInvalidated != 0 || stats.FactsAdded != 1 {
				t.Fatalf("unmatched raw reference should not invalidate another statement: %+v %v", stats, err)
			}
		})
	}
}
