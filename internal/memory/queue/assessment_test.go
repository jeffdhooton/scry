package queue

import (
	"context"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestAssessmentHookObservesOriginalBeforeResolutionAndCannotMutateGraph(t *testing.T) {
	st := openTemp(t)
	p := pending("assessment-hook", "User: scry runs on mini")
	if err := st.PutPending(p); err != nil {
		t.Fatal(err)
	}
	result := extract.Result{EpisodeSummary: "original", Entities: []extract.Ent{{Name: "scry", Type: "project"}, {Name: "mini", Type: "machine"}}, Facts: []extract.Fct{{Src: "scry", Relation: "deployed_on", Dst: "mini", Fact: "scry runs on mini", Confidence: .9}, {Src: "scry", Relation: "related_to", Dst: "scry", Fact: "original self relation", Confidence: .8}}}
	calls := 0
	w := New(Options{Store: st, Extractor: fixedExtractor{result}, OnExtracted: func(got store.PendingEpisode, r extract.Result) {
		calls++
		if has, err := st.HasEpisode(p.ID); err != nil || has {
			t.Errorf("hook ran after graph: %v %v", has, err)
		}
		if got.Text != p.Text || len(r.Facts) != 2 || r.Facts[1].Fact != "original self relation" {
			t.Errorf("lost original candidate: %+v", r)
		}
		r.Facts[0].Fact = "mutated by observer"
		r.Entities[0].Name = "observer-owned"
	}})
	w.process(context.Background(), p)
	if calls != 1 {
		t.Fatalf("hook calls %d", calls)
	}
	facts, err := st.FactsFrom("scry", false)
	if err != nil || len(facts) != 2 || facts[0].Fact != "scry runs on mini" || facts[1].Fact != "original self relation" {
		t.Fatalf("observer changed graph: %+v %v", facts, err)
	}
	if has, _ := st.HasPending(p.ID); has {
		t.Fatal("observer prevented ingest")
	}
}

func TestSplitDoesNotMisattributeWholeEpisodeSpansToHalf(t *testing.T) {
	st := openTemp(t)
	p := pending("span-split", strings.Repeat("User: a long statement\n\n", 500))
	p.SourceNamespace = "remote"
	p.SourceSpanKnown = true
	p.SourceStart = 100
	p.SourceEnd = 20000
	p.SourceTurns = []distill.SourceTurn{{Speaker: "user", Text: p.Text, Start: 100, End: 20000}}
	w := New(Options{Store: st, Extractor: &fakeExtractor{}})
	ok, err := w.splitPending(p)
	if err != nil || !ok {
		t.Fatalf("split %v %v", ok, err)
	}
	children, err := st.Pending(0)
	if err != nil || len(children) != 2 {
		t.Fatalf("children: %d %v", len(children), err)
	}
	for _, child := range children {
		if child.SourceSpanKnown || len(child.SourceTurns) != 0 || child.SourceStart != 0 || child.SourceEnd != 0 {
			t.Fatal("split asserted false span identity")
		}
		if child.SourceNamespace != "remote" {
			t.Fatal("split lost namespace")
		}
	}
}

func TestAssessmentHookNotCalledOnExtractionFailure(t *testing.T) {
	st := openTemp(t)
	p := pending("failed-extraction", "source")
	calls := 0
	w := New(Options{Store: st, Extractor: &fakeExtractor{errs: []error{extract.ErrParse}}, OnExtracted: func(store.PendingEpisode, extract.Result) { calls++ }})
	w.process(context.Background(), p)
	if calls != 0 {
		t.Fatal("assessment captured failed extraction")
	}
}
