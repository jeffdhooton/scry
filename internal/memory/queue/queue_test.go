package queue

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func openTemp(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "badger"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// fakeExtractor answers from a script: each call pops the next error (nil
// means success) and records the glossary it was handed.
type fakeExtractor struct {
	mu        sync.Mutex
	errs      []error
	calls     int32
	glossary  [][]string
	failUntil time.Time // transport failures before this instant
	delay     time.Duration
}

func (f *fakeExtractor) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (extract.Result, error) {
	atomic.AddInt32(&f.calls, 1)
	f.mu.Lock()
	f.glossary = append(f.glossary, glossary)
	var err error
	if len(f.errs) > 0 {
		err = f.errs[0]
		f.errs = f.errs[1:]
	}
	failUntil := f.failUntil
	f.mu.Unlock()
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return extract.Result{}, ctx.Err()
		}
	}
	if !failUntil.IsZero() && time.Now().Before(failUntil) {
		return extract.Result{}, errors.New("dial tcp: connection refused")
	}
	if err != nil {
		return extract.Result{}, err
	}
	return extract.Result{
		EpisodeSummary: "summary of " + ep.ID,
		Entities:       []extract.Ent{{Name: "scry", Type: "project"}, {Name: "mini", Type: "machine"}},
		Facts:          []extract.Fct{{Src: "scry", Relation: "deployed_on", Dst: "mini", Fact: ep.Text, Confidence: 0.9}},
	}, nil
}

func pending(id, text string) store.PendingEpisode {
	now := time.Now()
	return store.PendingEpisode{ID: id, Source: "manual", SourceRef: "manual", Text: text,
		OccurredAt: now, EnqueuedAt: now, NextAttempt: now}
}

func runFor(t *testing.T, w *Worker, d time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	w.Run(ctx)
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func TestSuccessResolvesAndDeletesPending(t *testing.T) {
	st := openTemp(t)
	if err := st.PutPending(pending("p1", "scry runs on the mini")); err != nil {
		t.Fatal(err)
	}
	fx := &fakeExtractor{}
	w := New(Options{Store: st, Extractor: fx, Glossary: func() []string { return []string{"scry: scry daemon"} },
		Poll: 10 * time.Millisecond})
	go runFor(t, w, 2*time.Second)

	if !waitUntil(t, 2*time.Second, func() bool { has, _ := st.HasPending("p1"); return !has }) {
		t.Fatal("pending item never drained")
	}
	if has, _ := st.HasEpisode("p1"); !has {
		t.Error("episode not recorded after success")
	}
	facts, _ := st.FactsFrom("scry", false)
	if len(facts) != 1 {
		t.Errorf("facts on scry = %d, want 1", len(facts))
	}
	if _, found, _ := st.GetMetaTime(store.MetaLastExtract); !found {
		t.Error("MetaLastExtract not stamped")
	}
	fx.mu.Lock()
	defer fx.mu.Unlock()
	if len(fx.glossary) != 1 || fx.glossary[0][0] != "scry: scry daemon" {
		t.Errorf("glossary handed to extractor = %v", fx.glossary)
	}
}

func TestManualEpisodeKeepsItsTextAsSummary(t *testing.T) {
	st := openTemp(t)
	_ = st.PutPending(pending("m1", "Jeff prefers zsh"))
	w := New(Options{Store: st, Extractor: &fakeExtractor{}, Poll: 10 * time.Millisecond})
	go runFor(t, w, time.Second)
	waitUntil(t, time.Second, func() bool { has, _ := st.HasPending("m1"); return !has })
	ep, err := st.GetEpisode("m1")
	if err != nil {
		t.Fatal(err)
	}
	if ep.Summary != "Jeff prefers zsh" {
		t.Errorf("manual summary = %q, want the fact text", ep.Summary)
	}
}

func TestTransportFailureBacksOffWithoutParking(t *testing.T) {
	st := openTemp(t)
	_ = st.PutPending(pending("p1", "x"))
	fx := &fakeExtractor{errs: []error{errors.New("dial tcp: i/o timeout")}}
	w := New(Options{Store: st, Extractor: fx, Poll: 10 * time.Millisecond})
	w.backoff = func(int) time.Duration { return time.Hour }
	go runFor(t, w, 300*time.Millisecond)
	waitUntil(t, time.Second, func() bool { p, _ := st.GetPending("p1"); return p.Attempts == 1 })
	p, err := st.GetPending("p1")
	if err != nil {
		t.Fatal("item was dropped on a transport failure")
	}
	if p.Parked || p.Attempts != 1 || p.LastError == "" || !p.NextAttempt.After(time.Now().Add(30*time.Minute)) {
		t.Errorf("after transport failure: %+v", p)
	}
	if atomic.LoadInt32(&fx.calls) != 1 {
		t.Errorf("extractor called %d times while in backoff, want 1", fx.calls)
	}
}

func TestParseFailuresParkAfterMaxAttempts(t *testing.T) {
	st := openTemp(t)
	_ = st.PutPending(pending("p1", "x"))
	parse := fmt.Errorf("bad: %w", extract.ErrParse)
	fx := &fakeExtractor{errs: []error{parse, parse, parse, nil}}
	w := New(Options{Store: st, Extractor: fx, Poll: 10 * time.Millisecond})
	w.backoff = func(int) time.Duration { return 0 }
	go runFor(t, w, 2*time.Second)
	if !waitUntil(t, 2*time.Second, func() bool { p, _ := st.GetPending("p1"); return p.Parked }) {
		t.Fatal("item never parked")
	}
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&fx.calls) != MaxParseAttempts {
		t.Errorf("calls = %d, want %d (parked items are not claimed)", fx.calls, MaxParseAttempts)
	}
	if has, _ := st.HasEpisode("p1"); has {
		t.Error("parked item must not be recorded as an episode")
	}
}

func TestProviderOutageThenRecoveryDrainsEverything(t *testing.T) {
	st := openTemp(t)
	for i := range 20 {
		_ = st.PutPending(pending(fmt.Sprintf("p%02d", i), "fact"))
	}
	fx := &fakeExtractor{failUntil: time.Now().Add(200 * time.Millisecond)}
	w := New(Options{Store: st, Extractor: fx, Workers: 4, Poll: 10 * time.Millisecond})
	w.backoff = func(int) time.Duration { return 50 * time.Millisecond }
	go runFor(t, w, 5*time.Second)
	if !waitUntil(t, 5*time.Second, func() bool { r, b, p, _ := st.PendingCounts(time.Now()); return r+b+p == 0 }) {
		r, b, p, _ := st.PendingCounts(time.Now())
		t.Fatalf("queue not drained: ready %d backoff %d parked %d", r, b, p)
	}
	eps, _ := st.AllEpisodes()
	if len(eps) != 20 {
		t.Errorf("episodes = %d, want 20 with no duplicates", len(eps))
	}
}

func TestKickWakesTheLoop(t *testing.T) {
	st := openTemp(t)
	fx := &fakeExtractor{}
	w := New(Options{Store: st, Extractor: fx, Poll: time.Hour})
	go runFor(t, w, time.Second)
	time.Sleep(20 * time.Millisecond)
	_ = st.PutPending(pending("k1", "x"))
	w.Kick()
	if !waitUntil(t, time.Second, func() bool { has, _ := st.HasPending("k1"); return !has }) {
		t.Fatal("Kick did not wake the worker")
	}
}

func TestItemTimeoutIsATransportFailure(t *testing.T) {
	st := openTemp(t)
	_ = st.PutPending(pending("s1", "x"))
	fx := &fakeExtractor{delay: time.Second}
	w := New(Options{Store: st, Extractor: fx, Poll: 10 * time.Millisecond, ItemTimeout: 30 * time.Millisecond})
	w.backoff = func(int) time.Duration { return time.Hour }
	go runFor(t, w, 500*time.Millisecond)
	waitUntil(t, time.Second, func() bool { p, _ := st.GetPending("s1"); return p.Attempts == 1 })
	p, _ := st.GetPending("s1")
	if p.Attempts != 1 || p.Parked {
		t.Errorf("after timeout: %+v", p)
	}
}

func TestBackoffCapsAtTwoMinutes(t *testing.T) {
	if Backoff(1) != 30*time.Second || Backoff(2) != time.Minute || Backoff(3) != 2*time.Minute || Backoff(9) != 2*time.Minute {
		t.Errorf("Backoff = %v %v %v %v", Backoff(1), Backoff(2), Backoff(3), Backoff(9))
	}
}
