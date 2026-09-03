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
	// The manual item and the first transcript item are claimed in the same
	// dispatch pass (reserved slot vs general slot), so either may record
	// first; the manual one must not be behind the backlog.
	if order[0] != "manual-1" && order[1] != "manual-1" {
		t.Errorf("manual episode not at the front: %v", order[:3])
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

func TestRepeatedTimeoutsSplitTheEpisodeInsteadOfParking(t *testing.T) {
	st := openTemp(t)
	long := pending("big", strings.Repeat("User: a long turn about the deploy path\n\nAssistant: a long answer\n\n", 120))
	long.Source = "claude-session"
	long.SourceRef = "/tmp/session.jsonl#0-99999"
	if len(long.Text) < minSplitChars {
		t.Fatalf("fixture text is only %d chars", len(long.Text))
	}
	_ = st.PutPending(long)
	fx := &fakeExtractor{delay: time.Second}
	w := New(Options{Store: st, Extractor: fx, Workers: 2, Poll: 10 * time.Millisecond, ItemTimeout: 20 * time.Millisecond})
	w.backoff = func(int) time.Duration { return 0 }
	runFor(t, w, 3*time.Second)

	if !waitUntil(t, 3*time.Second, func() bool { has, _ := st.HasPending("big"); return !has }) {
		p, _ := st.GetPending("big")
		t.Fatalf("parent never split: %+v", p)
	}
	var halves []store.PendingEpisode
	items, _ := st.Pending(0)
	for _, it := range items {
		if strings.Contains(it.SourceRef, "#split") {
			halves = append(halves, it)
		}
	}
	if len(halves) != 2 {
		t.Fatalf("halves = %d, want 2 (of %d pending)", len(halves), len(items))
	}
	joined := halves[0].Text + halves[1].Text
	for _, want := range []string{"a long turn about the deploy path", "a long answer"} {
		if !strings.Contains(joined, want) {
			t.Errorf("split lost content: %q missing", want)
		}
	}
	if halves[0].ID == halves[1].ID || halves[0].ID == "big" {
		t.Error("halves must have their own derived ids")
	}
	if halves[0].Attempts != 0 || halves[0].Parked {
		t.Errorf("halves start fresh: %+v", halves[0])
	}
	if splitDepth(halves[0].SourceRef) != 1 {
		t.Errorf("split depth = %d", splitDepth(halves[0].SourceRef))
	}
}

func TestSplitStopsAtDepthAndShortText(t *testing.T) {
	st := openTemp(t)
	w := New(Options{Store: st, Extractor: &fakeExtractor{}, Poll: time.Hour})
	short := pending("s", "too short to split")
	if ok, err := w.splitPending(short); ok || err != nil {
		t.Errorf("short text: %v %v", ok, err)
	}
	deep := pending("d", strings.Repeat("word ", 2000))
	deep.SourceRef = "/tmp/x.jsonl#0-1#split1#split2#split1"
	if ok, err := w.splitPending(deep); ok || err != nil {
		t.Errorf("max depth: %v %v", ok, err)
	}
	l, r := splitText("User: one\n\nAssistant: two\n\nUser: three\n\nAssistant: four")
	if l == "" || r == "" || strings.Contains(l, "four") {
		t.Errorf("splitText = %q | %q", l, r)
	}
}

func TestRateLimitNarrowsThenWidensAgain(t *testing.T) {
	st := openTemp(t)
	for i := range 60 {
		_ = st.PutPending(pending(fmt.Sprintf("r%02d", i), "fact"))
	}
	limited := errors.New(`POST "https://api.z.ai/api/anthropic/v1/messages": 429 Too Many Requests {"type":"error","error":{"type":"rate_limit_error"}}`)
	if !extract.IsRateLimited(limited) || extract.IsRateLimited(errors.New("500 boom")) {
		t.Fatal("IsRateLimited wrong")
	}
	// Refuse everything until the test says otherwise, so narrowing is not
	// a race with a scripted error list.
	var refusing atomic.Bool
	refusing.Store(true)
	fx := &gatedExtractor{refuse: &refusing, err: limited}

	w := New(Options{Store: st, Extractor: fx, Workers: 24, Poll: 10 * time.Millisecond})
	if w.Limit() != startLimit {
		t.Fatalf("initial limit = %d, want %d", w.Limit(), startLimit)
	}
	runFor(t, w, 20*time.Second)

	if !waitUntil(t, 5*time.Second, func() bool { return w.Limit() <= minLimit }) {
		t.Fatalf("limit never narrowed to the floor under rate limiting (still %d)", w.Limit())
	}
	// A rate-limit refusal must not spend an item's budget.
	items, _ := st.Pending(0)
	spent := 0
	for _, it := range items {
		if it.Attempts > 0 {
			spent++
		}
	}
	if spent > 0 {
		t.Errorf("%d items spent an attempt on a rate-limit refusal", spent)
	}

	refusing.Store(false)
	if !waitUntil(t, 12*time.Second, func() bool { return w.Limit() > minLimit }) {
		t.Errorf("limit never widened again after the provider recovered (still %d)", w.Limit())
	}
}

// gatedExtractor refuses with err while refuse is set, and succeeds after.
type gatedExtractor struct {
	refuse *atomic.Bool
	err    error
	inner  fakeExtractor
}

func (g *gatedExtractor) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (extract.Result, error) {
	if g.refuse.Load() {
		return extract.Result{}, g.err
	}
	return g.inner.Extract(ctx, ep, glossary)
}

func TestSaturatedPoolWidensWhenTheProviderIsQuiet(t *testing.T) {
	st := openTemp(t)
	for i := range 40 {
		_ = st.PutPending(pending(fmt.Sprintf("s%02d", i), "fact"))
	}
	// Every call hangs, so the pool stays saturated and no success can
	// widen it: only the time-based rule can.
	fx := &fakeExtractor{delay: time.Minute}
	w := New(Options{Store: st, Extractor: fx, Workers: 24, Poll: 10 * time.Millisecond, ItemTimeout: time.Minute})
	old := growQuietForTest(t, 40*time.Millisecond)
	defer old()
	runFor(t, w, 3*time.Second)
	if !waitUntil(t, 3*time.Second, func() bool { return w.Limit() >= startLimit+3 }) {
		t.Fatalf("a saturated pool with a quiet provider never widened (limit %d)", w.Limit())
	}
	if w.Limit() > 24 {
		t.Errorf("limit passed the worker ceiling: %d", w.Limit())
	}
}

// growQuietForTest shortens the quiet window and returns a restore func.
func growQuietForTest(t *testing.T, d time.Duration) func() {
	t.Helper()
	old := growQuiet
	growQuiet = d
	return func() { growQuiet = old }
}
