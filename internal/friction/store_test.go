package friction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func sample(id, run string) Event {
	return Event{EventID: id, RunID: run, Repository: "/fixture/repo", RecordedAt: "2026-09-06T12:00:00-04:00", Signature: "instructions.conflict", Observed: "Two instructions conflict.", Resolution: "Stopped; unresolved.", ResolutionState: "unresolved", Evidence: []string{"session:42"}, ProposedChange: "Clarify the instruction.", ProposedOwner: "maintainer", ProposedFile: "SKILL.md", ProposedVerification: "Repeat the blocked task."}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func TestConcurrentRetryAndConflict(t *testing.T) {
	s := openTestStore(t)
	e := sample("event-1", "run-1")
	var wg sync.WaitGroup
	var created atomic.Int32
	for range 12 {
		wg.Go(func() {
			r, err := s.Record(e)
			if err != nil {
				t.Error(err)
				return
			}
			if r.Created {
				created.Add(1)
			}
		})
	}
	wg.Wait()
	if created.Load() != 1 {
		t.Fatalf("created %d copies", created.Load())
	}
	changed := e
	changed.Resolution = "Pretend fixed"
	if _, err := s.Record(changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed event: %v", err)
	}
	changed = e
	changed.Repository = "/other"
	if _, err := s.Record(changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("ID collision across repos: %v", err)
	}
	got, err := s.Get(e.EventID)
	if err != nil || !reflect.DeepEqual(*got, e) {
		t.Fatalf("lost original: %+v %v", got, err)
	}
	page, err := s.List(context.Background(), Filter{Repository: e.Repository})
	if err != nil || len(page.Events) != 1 {
		t.Fatalf("duplicated: %+v %v", page, err)
	}
}

func TestReviewCountsStoredDistinctRunsAndCitesProposals(t *testing.T) {
	s := openTestStore(t)
	for i, run := range []string{"run-1", "run-1", "run-2"} {
		e := sample(fmt.Sprintf("event-%d", i), run)
		e.DistinctPriorRunsVerified = 999
		e.OccurrencesObservedThisRun = 50
		if i == 0 {
			n := 12.5
			e.MeasuredUserTimeCostSeconds = &n
		}
		if _, err := s.Record(e); err != nil {
			t.Fatal(err)
		}
	}
	other := sample("event-other", "run-3")
	other.Repository = "/other"
	if _, err := s.Record(other); err != nil {
		t.Fatal(err)
	}
	r, err := s.Review(context.Background(), Filter{Repository: "/fixture/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if r.EventCount != 3 || len(r.Groups) != 1 || !r.RecommendationsOnly {
		t.Fatalf("review: %+v", r)
	}
	g := r.Groups[0]
	if g.DistinctRuns != 2 || !g.Recurring || !reflect.DeepEqual(g.RunIDs, []string{"run-1", "run-2"}) {
		t.Fatalf("run counts: %+v", g)
	}
	if g.MeasuredEvents != 1 || g.MeasuredUserTimeCostSeconds == nil || *g.MeasuredUserTimeCostSeconds != 12.5 {
		t.Fatalf("impact: %+v", g)
	}
	if len(g.Proposals) != 3 || g.Proposals[0].EventID != g.Events[0].EventID || g.Proposals[0].Verification == "" {
		t.Fatalf("uncited proposals: %+v", g)
	}
	r, err = s.Review(context.Background(), Filter{Repository: "/fixture/repo", RunID: "run-1"})
	if err != nil || r.Groups[0].DistinctRuns != 1 || r.Groups[0].Recurring {
		t.Fatalf("same run overcounted: %+v %v", r, err)
	}
	r, err = s.Review(context.Background(), Filter{Repository: "/other"})
	if err != nil || r.Groups[0].MeasuredUserTimeCostSeconds != nil {
		t.Fatalf("unknown impact invented: %+v %v", r, err)
	}
}

func TestPaginationTimeBoundsAndReviewRefusesTruncation(t *testing.T) {
	s := openTestStore(t)
	for i := range 101 {
		e := sample(fmt.Sprintf("event-%03d", i), "run-1")
		if i%2 == 0 {
			e.RunID = "run-2"
		}
		if _, err := s.Record(e); err != nil {
			t.Fatal(err)
		}
	}
	f := Filter{Repository: "/fixture/repo", Limit: 7, RunID: "run-2", Since: "2026-09-06T16:00:00Z", Until: "2026-09-06T16:00:01Z"}
	seen := map[string]bool{}
	for {
		p, err := s.List(context.Background(), f)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range p.Events {
			if seen[e.EventID] {
				t.Fatal("duplicate page result")
			}
			seen[e.EventID] = true
		}
		if p.NextAfter == "" {
			break
		}
		f.After = p.NextAfter
	}
	if len(seen) != 51 {
		t.Fatalf("pagination lost events: %d", len(seen))
	}
	f = Filter{Repository: "/fixture/repo", Until: "2026-09-06T16:00:00Z"}
	p, err := s.List(context.Background(), f)
	if err != nil || len(p.Events) != 0 {
		t.Fatalf("until not exclusive: %+v %v", p, err)
	}
	if _, err := s.Review(context.Background(), Filter{Repository: "/fixture/repo"}); !errors.Is(err, ErrReviewTooLarge) {
		t.Fatalf("silent partial review: %v", err)
	}
	if _, err := s.Review(context.Background(), Filter{Repository: "/fixture/repo", After: "event-001"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("accepted review cursor: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.List(ctx, Filter{Repository: "/fixture/repo"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ignored cancellation: %v", err)
	}
}

func TestReviewRejectsCostOverflowWithoutLosingEvents(t *testing.T) {
	s := openTestStore(t)
	for _, id := range []string{"large-1", "large-2"} {
		e := sample(id, id)
		n := 1e308
		e.MeasuredUserTimeCostSeconds = &n
		if _, err := s.Record(e); err != nil {
			t.Fatal(err)
		}
	}
	r, err := s.Review(context.Background(), Filter{Repository: "/fixture/repo"})
	if r != nil || !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "overflows") {
		t.Fatalf("non-JSON numeric total escaped review: %+v %v", r, err)
	}
	p, err := s.List(context.Background(), Filter{Repository: "/fixture/repo"})
	if err != nil || len(p.Events) != 2 {
		t.Fatalf("overflow discarded events: %+v %v", p, err)
	}
}

func TestInvalidEventCannotEraseOrActivate(t *testing.T) {
	s := openTestStore(t)
	for name, change := range map[string]func(*Event){
		"id":          func(e *Event) { e.EventID = "../escape" },
		"run":         func(e *Event) { e.RunID = "" },
		"repo":        func(e *Event) { e.Repository = "relative" },
		"time":        func(e *Event) { e.RecordedAt = "yesterday" },
		"observation": func(e *Event) { e.Observed = " " },
		"evidence":    func(e *Event) { e.Evidence = nil },
		"hash":        func(e *Event) { e.EvidenceSHA256 = map[string]string{"missing": strings.Repeat("a", 64)} },
		"impact":      func(e *Event) { n := -1.0; e.MeasuredUserTimeCostSeconds = &n },
		"approval":    func(e *Event) { e.ChangeApproved = true },
		"oversize":    func(e *Event) { e.Observed = strings.Repeat("x", MaxEventBytes) },
	} {
		t.Run(name, func(t *testing.T) {
			e := sample("invalid", "run")
			change(&e)
			if _, err := s.Record(e); !errors.Is(err, ErrInvalid) {
				t.Fatalf("accepted invalid event: %v", err)
			}
		})
	}
	var e Event
	if err := Decode([]byte(`{"event_id":"id","evidnce":["x"]}`), &e); !errors.Is(err, ErrInvalid) {
		t.Fatal("silently discarded typo")
	}
	if err := Decode([]byte(`{} {}`), &e); !errors.Is(err, ErrInvalid) {
		t.Fatal("accepted second object")
	}
	if _, err := s.Get("invalid"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid event was stored: %v", err)
	}
	if _, err := s.List(context.Background(), Filter{Repository: "/fixture/repo", Limit: 101}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unbounded page accepted: %v", err)
	}
}

func TestReopenExactAndUnsupportedSchemaPreserved(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := sample("event", "run")
	r, err := s.Record(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(e.EventID)
	if err != nil || !reflect.DeepEqual(*got, e) {
		t.Fatalf("reopen: %+v %v", got, err)
	}
	r2, err := s.Record(e)
	if err != nil || r2.Created || r2.SHA256 != r.SHA256 {
		t.Fatalf("retry after reopen: %+v %v", r2, err)
	}
	if err := s.db.Update(func(tx *badger.Txn) error { return tx.Set([]byte("schema"), []byte("future")) }); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if st, err := Open(dir); err == nil {
		st.Close()
		t.Fatal("accepted future schema")
	}
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(eventKey(e.EventID))
		if err != nil {
			return err
		}
		return item.Value(func(b []byte) error {
			var got Event
			if err := json.Unmarshal(b, &got); err != nil {
				return err
			}
			if !reflect.DeepEqual(got, e) {
				t.Fatal("future schema open changed event")
			}
			return nil
		})
	}); err != nil {
		t.Fatal(err)
	}
}
