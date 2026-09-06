package queue

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestHistoricalCanonicalCollisionParksWithoutLosingInput(t *testing.T) {
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
	p := pending("collision-input", "Complete original input for future exact review.")
	p.OccurredAt = at.Add(2 * time.Hour)
	if err := st.PutPending(p); err != nil {
		t.Fatal(err)
	}
	w := New(Options{Store: st, Extractor: fixedExtractor{result: extract.Result{EpisodeSummary: "different assertion", Facts: []extract.Fct{{Src: "Velatrix", Relation: "status", Dst: "in_progress", Fact: "Workspace review passed but the larger goal remains unfinished.", Confidence: .94, ValidFrom: at.Format(time.RFC3339Nano)}}}}, Poll: time.Hour})
	w.process(context.Background(), p)
	got, err := st.GetPending(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Parked || got.Attempts != 1 || got.Text != p.Text || got.SourceRef != p.SourceRef || !strings.Contains(got.LastError, "occupied fact key sha256=") {
		t.Fatalf("pending=%+v", got)
	}
	facts, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(facts, []store.Fact{old}) {
		t.Fatalf("history changed: %+v", facts)
	}
	if has, err := st.HasEpisode(p.ID); err != nil || has {
		t.Fatalf("episode persisted: %v %v", has, err)
	}
}
