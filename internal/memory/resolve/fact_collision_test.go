package resolve

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestApplyHistoricalCanonicalKeyCollisionRollsBack(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	until := at.Add(time.Hour)
	if err := st.PutEntity(store.Entity{Slug: "velatrix", Name: "Velatrix", Type: "project"}); err != nil {
		t.Fatal(err)
	}
	old := store.Fact{Src: "velatrix", Relation: "status", Value: "in-progress", Fact: "The geometry reader is next.", ValidFrom: at, InvalidAt: &until, Confidence: .95, Episodes: []string{"old-shape"}}
	if err := st.PutFact(old); err != nil {
		t.Fatal(err)
	}
	before, err := st.Entities()
	if err != nil {
		t.Fatal(err)
	}
	ep := store.Episode{ID: "new-shape", Source: "manual", SourceRef: "synthetic:collision", OccurredAt: at.Add(2 * time.Hour), IngestedAt: at.Add(3 * time.Hour)}
	stats, err := Apply(st, ep, "", extract.Result{
		EpisodeSummary: "New exact assertion conflicts with stored history.",
		Entities:       []extract.Ent{{Name: "Sentinel", Type: "tool", Description: "must roll back"}},
		Facts: []extract.Fct{
			{Src: "Velatrix", Relation: "uses", Dst: "Sentinel", Fact: "A different valid edge must also roll back.", Confidence: .9},
			{Src: "Velatrix", Relation: "status", Dst: "in_progress", Fact: "Workspace review passed but the larger goal remains unfinished.", Confidence: .94, ValidFrom: at.Format(time.RFC3339Nano)},
		},
	}, DefaultExclusive)
	if !errors.Is(err, store.ErrFactConflict) || stats != (Stats{}) {
		t.Fatalf("stats=%+v err=%v", stats, err)
	}
	facts, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(facts, []store.Fact{old}) {
		t.Fatalf("history changed: %+v", facts)
	}
	after, err := st.Entities()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rollback changed entities")
	}
	if has, err := st.HasEpisode(ep.ID); err != nil || has {
		t.Fatalf("episode persisted: %v %v", has, err)
	}
}
