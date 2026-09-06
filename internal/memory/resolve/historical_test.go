package resolve

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestHistoricalAddressSelfSupersessionStaysClosed(t *testing.T) {
	for _, rel := range []string{"uses", "measured", "aliases_index_to"} {
		t.Run(rel, func(t *testing.T) {
			st := openTemp(t)
			at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
			end := at.Add(time.Hour)
			first := extract.Fct{Src: "Lornwick", Dst: "Caldera", Relation: rel, Fact: "Lornwick uses Caldera.", ValidFrom: at.Format(time.RFC3339Nano), Confidence: .95}
			second := first
			second.Confidence = .2
			second.Supersedes = &extract.SupRef{Src: first.Src, Dst: first.Dst, Relation: rel}
			stats := applyOne(t, st, "self-supersession", end, first, second)
			if stats.FactsAdded != 1 || stats.FactsMerged != 1 || stats.FactsInvalidated != 1 {
				t.Fatalf("stats %+v", stats)
			}
			got := mustFacts(t, st, "lornwick")
			if len(got) != 1 || got[0].InvalidAt == nil || !got[0].InvalidAt.Equal(end) || !got[0].ValidFrom.Equal(at) || got[0].Confidence != .95 || !reflect.DeepEqual(got[0].Episodes, []string{"self-supersession"}) {
				t.Fatalf("reopened or lowered %+v", got)
			}
			if rel == "aliases_index_to" && (got[0].Relation != Fallback || got[0].RawRelation != rel || got[0].Dst != "caldera") {
				t.Fatalf("fixture did not exercise fallback edge: %+v", got[0])
			}
		})
	}
}

func TestHistoricalAddressRestatement(t *testing.T) {
	for _, relation := range []string{"uses", "measured"} {
		for _, date := range []string{"2026-08-10", "2026-08-10T02:00:00+02:00", ""} {
			t.Run(relation+"/"+date, func(t *testing.T) {
				st := openTemp(t)
				at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
				end := at.Add(time.Hour)
				old := seedFallback(t, st, at, &end)
				if relation == "uses" {
					if err := st.DeleteFact(old.Src, old.Relation, old.KeyDst(), old.ValidFrom); err != nil {
						t.Fatal(err)
					}
					old.Relation, old.RawRelation = "uses", ""
					if err := st.PutFact(old); err != nil {
						t.Fatal(err)
					}
				}
				occurred := at.Add(2 * time.Hour)
				if date == "" {
					occurred = at
				}
				input := extract.Fct{Src: old.Src, Dst: old.Dst, Relation: relation, Fact: old.Fact, ValidFrom: date, Confidence: .9}
				stats := applyOne(t, st, "historical-repeat", occurred, input, input)
				if stats.FactsMerged != 2 || stats.FactsAdded != 0 || stats.FactsInvalidated != 0 {
					t.Fatalf("stats %+v", stats)
				}
				want := old
				want.Confidence = .9
				want.Episodes = append(want.Episodes, "historical-repeat")
				if got := mustFacts(t, st, old.Src); !reflect.DeepEqual(got, []store.Fact{want}) {
					t.Fatalf("history changed %+v", got)
				}
			})
		}
	}
}

func TestHistoricalAddressDoesNotInferUndatedInterval(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := at.Add(time.Hour)
	old := seedFallback(t, st, at, &end)
	stats := applyOne(t, st, "recurrence", end.Add(time.Hour), extract.Fct{Src: old.Src, Dst: old.Dst, Relation: old.RawRelation, Fact: old.Fact, Confidence: .9})
	if stats.FactsAdded != 1 || stats.FactsMerged != 0 {
		t.Fatalf("recurrence consumed %+v", stats)
	}
	got := mustFacts(t, st, old.Src)
	if len(got) != 2 || !reflect.DeepEqual(got[0], old) {
		t.Fatalf("history changed %+v", got)
	}
}

func TestHistoricalAddressMalformedDateRefusesAtomically(t *testing.T) {
	for _, date := range []string{"not-a-date", "2026-02-30", " "} {
		t.Run(date, func(t *testing.T) {
			st := openTemp(t)
			at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
			end := at.Add(time.Hour)
			old := seedFallback(t, st, at, &end)
			before, _ := st.Entities()
			events := 0
			st.SetObserver(func(store.Event) { events++ })
			stats, err := Apply(st, store.Episode{ID: "malformed-date", OccurredAt: at}, "", extract.Result{Entities: []extract.Ent{{Name: "Lornwick", Type: "project"}}, Facts: []extract.Fct{{Src: old.Src, Dst: old.Dst, Relation: old.RawRelation, Fact: old.Fact, ValidFrom: date}}}, DefaultExclusive)
			if !errors.Is(err, store.ErrFactConflict) || stats != (Stats{}) || events != 0 {
				t.Fatalf("not atomic %+v %v %d", stats, err, events)
			}
			after, _ := st.Entities()
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(mustFacts(t, st, old.Src), []store.Fact{old}) {
				t.Fatal("state changed")
			}
			if has, err := st.HasEpisode("malformed-date"); err != nil || has {
				t.Fatalf("episode %v %v", has, err)
			}
		})
	}
}
