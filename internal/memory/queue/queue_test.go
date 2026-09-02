package queue

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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

// runFor runs w for at most d in the background and, at cleanup, stops it
// and waits for Run to return before the store (registered earlier, so
// cleaned up later) is closed. A worker that outlives its test panics on a
// closed Badger handle.
func runFor(t *testing.T, w *Worker, d time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
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
	runFor(t, w, 2*time.Second)

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
	runFor(t, w, time.Second)
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
	runFor(t, w, 300*time.Millisecond)
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
	runFor(t, w, 2*time.Second)
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
	runFor(t, w, 5*time.Second)
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
	runFor(t, w, time.Second)
	time.Sleep(20 * time.Millisecond)
	_ = st.PutPending(pending("k1", "x"))
	w.Kick()
	if !waitUntil(t, time.Second, func() bool { has, _ := st.HasPending("k1"); return !has }) {
		t.Fatal("Kick did not wake the worker")
	}
}

func TestItemTimeoutRetriesWithMoreRoomThenParks(t *testing.T) {
	st := openTemp(t)
	slow := pending("s1", "x")
	slow.Source = "claude-session"
	_ = st.PutPending(slow)
	fx := &fakeExtractor{delay: time.Second}
	w := New(Options{Store: st, Extractor: fx, Poll: 10 * time.Millisecond, ItemTimeout: 30 * time.Millisecond})
	w.backoff = func(int) time.Duration { return 0 }
	runFor(t, w, 3*time.Second)
	if !waitUntil(t, 3*time.Second, func() bool { p, _ := st.GetPending("s1"); return p.Parked }) {
		p, _ := st.GetPending("s1")
		t.Fatalf("item never parked after repeated timeouts: %+v", p)
	}
	p, _ := st.GetPending("s1")
	if p.Attempts != MaxTimeoutAttempts || !strings.Contains(p.LastError, "deadline") {
		t.Errorf("after timeouts: %+v", p)
	}
	if itemDeadline(time.Minute, 0) != time.Minute || itemDeadline(time.Minute, 1) != 2*time.Minute || itemDeadline(time.Minute, 5) != 3*time.Minute {
		t.Error("itemDeadline escalation wrong")
	}
}

func TestCompletionWakesTheLoopWithoutPolling(t *testing.T) {
	st := openTemp(t)
	for i := range 3 {
		_ = st.PutPending(pending(fmt.Sprintf("c%d", i), "x"))
	}
	w := New(Options{Store: st, Extractor: &fakeExtractor{}, Workers: 1, Poll: time.Hour})
	runFor(t, w, 2*time.Second)
	if !waitUntil(t, 2*time.Second, func() bool { r, _, _, _ := st.PendingCounts(time.Now()); return r == 0 }) {
		t.Fatal("with one worker and an hour poll, completions must wake the loop to take the next item")
	}
}

func TestBackoffCapsAtTwoMinutes(t *testing.T) {
	if Backoff(1) != 30*time.Second || Backoff(2) != time.Minute || Backoff(3) != 2*time.Minute || Backoff(9) != 2*time.Minute {
		t.Errorf("Backoff = %v %v %v %v", Backoff(1), Backoff(2), Backoff(3), Backoff(9))
	}
}

func TestManualEpisodesJumpTheBacklog(t *testing.T) {
	st := openTemp(t)
	base := time.Now().Add(-time.Hour)
	for i := range 30 {
		p := pending(fmt.Sprintf("t%02d", i), "transcript slice")
		p.Source = "claude-session"
		p.EnqueuedAt = base.Add(time.Duration(i) * time.Second)
		_ = st.PutPending(p)
	}
	m := pending("manual-1", "a remembered fact")
	m.EnqueuedAt = time.Now() // youngest of all
	_ = st.PutPending(m)

	var mu sync.Mutex
	var order []string
	fx := &fakeExtractor{delay: 20 * time.Millisecond}
	w := New(Options{Store: st, Extractor: &orderRecorder{inner: fx, mu: &mu, order: &order}, Workers: 1, Poll: 5 * time.Millisecond})
	runFor(t, w, 2*time.Second)
	if !waitUntil(t, 2*time.Second, func() bool { mu.Lock(); defer mu.Unlock(); return len(order) >= 3 }) {
		t.Fatal("nothing processed")
	}
	mu.Lock()
	defer mu.Unlock()
	if order[0] != "manual-1" {
		t.Errorf("first processed = %q, want the manual episode ahead of the backlog: %v", order[0], order[:3])
	}
}

type orderRecorder struct {
	inner extract.Extractor
	mu    *sync.Mutex
	order *[]string
}

func (o *orderRecorder) Extract(ctx context.Context, ep distill.RawEpisode, g []string) (extract.Result, error) {
	o.mu.Lock()
	*o.order = append(*o.order, ep.ID)
	o.mu.Unlock()
	return o.inner.Extract(ctx, ep, g)
}

func TestOrderIsManualFirstThenRoundRobinBySource(t *testing.T) {
	base := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	mk := func(id, src string, age int) store.PendingEpisode {
		return store.PendingEpisode{ID: id, Source: src, OccurredAt: base.Add(time.Duration(age) * time.Second), EnqueuedAt: base.Add(time.Duration(age) * time.Second), NextAttempt: base}
	}
	items := []store.PendingEpisode{
		mk("c1", "claude-session", 1), mk("c2", "claude-session", 2), mk("c3", "claude-session", 3),
		mk("k1", "kimi-session", 50), mk("o1", "opencode-session", 60),
		mk("m2", "manual", 90), mk("m1", "manual", 80),
		{ID: "parked", Source: "manual", Parked: true, EnqueuedAt: base, NextAttempt: base},
		{ID: "later", Source: "kimi-session", EnqueuedAt: base, NextAttempt: base.Add(time.Hour)},
	}
	got := order(items, base.Add(10*time.Minute))
	ids := make([]string, len(got))
	for i, p := range got {
		ids[i] = p.ID
	}
	want := "m1 m2 c3 k1 o1 c2 c1"
	if strings.Join(ids, " ") != want {
		t.Errorf("order = %q, want %q", strings.Join(ids, " "), want)
	}
}

func TestManualItemsNeverParkOnTimeoutAndHaveTheirOwnSlots(t *testing.T) {
	st := openTemp(t)
	// Fill every general slot with transcript items against a hung provider.
	for i := range 3 {
		p := pending(fmt.Sprintf("t%d", i), "long transcript")
		p.Source = "claude-session"
		_ = st.PutPending(p)
	}
	_ = st.PutPending(pending("m1", "a fact"))
	hung := &fakeExtractor{delay: 10 * time.Second}
	w := New(Options{Store: st, Extractor: hung, Workers: 3, Poll: 10 * time.Millisecond, ItemTimeout: 20 * time.Millisecond})
	w.backoff = func(int) time.Duration { return 0 }
	runFor(t, w, 2*time.Second)
	if !waitUntil(t, 2*time.Second, func() bool { p, _ := st.GetPending("m1"); return p.Attempts >= 4 }) {
		p, _ := st.GetPending("m1")
		t.Fatalf("manual item did not get attempts while general slots were hung: %+v", p)
	}
	p, _ := st.GetPending("m1")
	if p.Parked {
		t.Errorf("manual item parked on timeout: %+v", p)
	}
}
