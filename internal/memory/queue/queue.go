// Package queue is the daemon-side worker that turns pending episodes into
// facts. Clients — `scry_remember`, the sweep, `scry memory ingest` — only
// distill and enqueue; this worker owns every provider call. That puts the
// model chain in exactly one process per store, and it means a provider
// outage defers writes instead of losing them: an item that fails on
// transport waits and is retried, and only an item the models cannot parse
// after several tries is parked, still on disk and replayable.
package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

const (
	// MaxParseAttempts is how many times an episode may fail on content
	// (extract.ErrParse from every model in the chain) before it is parked.
	MaxParseAttempts = 3
	// MaxTimeoutAttempts is how many times a transcript episode may run
	// past its (escalating) deadline before it is parked. A reasoning model
	// on a 16 KB transcript slice can take minutes; one that never finishes
	// in three tries with a tripled deadline is too long for the chain, and
	// retrying it forever would starve everything behind it. Manual items
	// are never parked on a timeout: their text is a sentence, so a timeout
	// is always the provider's fault, and an agent is waiting for them.
	MaxTimeoutAttempts = 3
	// minSplitChars is the shortest text worth splitting when extraction
	// keeps timing out. Below it, a timeout is the provider's problem and
	// splitting would only multiply the calls.
	minSplitChars = 3000
	// maxSplitDepth bounds the recursion: an episode may be halved three
	// times (eighths) before it is parked for good.
	maxSplitDepth = 3

	// manualWorkers are slots reserved for manual episodes, so twenty
	// remembers made during an outage never wait behind transcript slices
	// that are holding every general slot against a hung upstream. GLM
	// takes 80-130 s even on a one-sentence fact (measured 2026-09-02), so
	// eight slots clear twenty remembers in about four minutes.
	manualWorkers = 8

	defaultWorkers     = 24
	defaultItemTimeout = 6 * time.Minute
	defaultPoll        = 30 * time.Second
	backoffBase        = 30 * time.Second
	backoffCap         = 2 * time.Minute

	// The in-flight limit finds itself: it starts low, grows by one after
	// a run of successes, and halves when the provider says 429. Z.ai
	// answered 300 of 300 requests with a rate-limit error at 24 in
	// flight, and each rejection cost a retry with a backoff, so throughput
	// collapsed to a fifth of what 8 in flight had managed.
	startLimit      = 6
	minLimit        = 2
	growAfter       = 8 // consecutive successes before widening by one
	rateBackoffBase = 5 * time.Second
	rateBackoffMax  = 45 * time.Second
)

// itemDeadline grows with each attempt so a slow-but-finishing episode
// gets room: 1x, 2x, 3x the base timeout.
func itemDeadline(base time.Duration, attempts int) time.Duration {
	mult := attempts + 1
	if mult > 3 {
		mult = 3
	}
	return base * time.Duration(mult)
}

// Backoff is the wait before attempt n+1: 30s, 1m, then 2m for good. The
// cap is short on purpose. Twenty writes made during an outage must all
// resolve within ten minutes of the provider returning, and a long tail of
// exponential waits would blow that budget.
func Backoff(attempts int) time.Duration {
	d := backoffBase
	for i := 1; i < attempts && d < backoffCap; i++ {
		d *= 2
	}
	if d > backoffCap {
		d = backoffCap
	}
	return d
}

// Options configures a Worker.
type Options struct {
	Store     *store.Store
	Extractor extract.Extractor
	// Glossary supplies the known-entity lines for extraction. It may be
	// nil. It should be cheap: it runs on every item.
	Glossary func() []string
	// Workers is the number of transcript items extracted concurrently
	// (default 24); manual items have manualWorkers slots of their own.
	Workers int
	// ItemTimeout bounds one item's first attempt (default 6m); later
	// attempts get 2x and 3x.
	ItemTimeout time.Duration
	// Poll is how often the loop looks for ready items when nothing kicks
	// it and no item completes (default 30s). Completions and enqueues
	// wake it immediately.
	Poll time.Duration
	Logf func(format string, args ...any)
}

// Worker drains the pending queue.
type Worker struct {
	o       Options
	kick    chan struct{}
	done    chan struct{} // an item finished; a slot is free
	backoff func(int) time.Duration

	mu       sync.Mutex
	inflight map[string]bool
	limit    int // in-flight ceiling, adapted from the provider's answers
	wins     int // consecutive successes since the last narrowing
}

// New builds a Worker; call Run to start it.
func New(o Options) *Worker {
	if o.Workers <= 0 {
		o.Workers = defaultWorkers
	}
	if o.ItemTimeout <= 0 {
		o.ItemTimeout = defaultItemTimeout
	}
	if o.Poll <= 0 {
		o.Poll = defaultPoll
	}
	if o.Logf == nil {
		o.Logf = log.Printf
	}
	limit := startLimit
	if limit > o.Workers {
		limit = o.Workers
	}
	return &Worker{o: o, kick: make(chan struct{}, 1), done: make(chan struct{}, 1),
		backoff: Backoff, inflight: map[string]bool{}, limit: limit}
}

// narrow halves the in-flight limit; called when the provider rate-limits.
func (w *Worker) narrow() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.wins = 0
	if w.limit <= minLimit {
		return
	}
	w.limit /= 2
	if w.limit < minLimit {
		w.limit = minLimit
	}
	w.o.Logf("memory queue: provider is rate-limiting — narrowing to %d in flight", w.limit)
}

// widen counts a success and grows the limit after a run of them.
func (w *Worker) widen() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.wins++
	if w.wins < growAfter || w.limit >= w.o.Workers {
		return
	}
	w.wins = 0
	w.limit++
	w.o.Logf("memory queue: widening to %d in flight", w.limit)
}

// Limit reports the current in-flight ceiling, for tests and status.
func (w *Worker) Limit() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.limit
}

// rateBackoff is the wait after a rate-limit refusal: short and jittered,
// so a narrowed worker pool does not resynchronise into another burst.
func (w *Worker) rateBackoff() time.Duration {
	d := rateBackoffBase + time.Duration(rand.Int64N(int64(rateBackoffBase)))
	if d > rateBackoffMax {
		d = rateBackoffMax
	}
	return d
}

// Kick wakes the loop early, after an enqueue.
func (w *Worker) Kick() {
	select {
	case w.kick <- struct{}{}:
	default:
	}
}

// Run polls until ctx is done. It never returns early on an error: a
// broken store read is logged and retried on the next tick.
func (w *Worker) Run(ctx context.Context) {
	sem := make(chan struct{}, w.o.Workers)
	manualSem := make(chan struct{}, manualWorkers)
	var wg sync.WaitGroup
	ticker := time.NewTicker(w.o.Poll)
	defer ticker.Stop()
	for {
		w.dispatch(ctx, sem, manualSem, &wg)
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
		case <-w.kick:
		case <-w.done:
		}
	}
}

// dispatch claims every ready item that fits in the free worker slots.
// Manual episodes (scry_remember) go first: an agent is waiting on those,
// while a sweep backlog of transcript slices is nobody's blocker. The rest
// are taken round-robin across sources, oldest first within a source, so
// a handful of Kimi or OpenCode episodes never waits behind thousands of
// Claude ones.
func (w *Worker) dispatch(ctx context.Context, sem, manualSem chan struct{}, wg *sync.WaitGroup) {
	if len(sem) == cap(sem) && len(manualSem) == cap(manualSem) {
		return // every slot busy; listing the queue would be wasted work
	}
	items, err := w.o.Store.Pending(0)
	if err != nil {
		w.o.Logf("memory queue: list pending: %v", err)
		return
	}
	now := time.Now()
	for _, p := range order(items, now) {
		if p.Parked || p.NextAttempt.After(now) {
			continue
		}
		w.mu.Lock()
		if w.inflight[p.ID] {
			w.mu.Unlock()
			continue
		}
		if len(w.inflight) >= w.limit {
			w.mu.Unlock()
			return // at the adaptive ceiling; a completion will wake us
		}
		// A manual item takes a reserved slot, or a free general one; a
		// transcript item only ever takes a general slot.
		var slot chan struct{}
		if p.Source == "manual" {
			select {
			case manualSem <- struct{}{}:
				slot = manualSem
			default:
			}
		}
		if slot == nil {
			select {
			case sem <- struct{}{}:
				slot = sem
			default:
				w.mu.Unlock()
				continue // busy; a differently classed item further on may still fit
			}
		}
		w.inflight[p.ID] = true
		w.mu.Unlock()

		wg.Add(1)
		go func(p store.PendingEpisode, slot chan struct{}) {
			defer wg.Done()
			defer func() {
				w.mu.Lock()
				delete(w.inflight, p.ID)
				w.mu.Unlock()
				<-slot
				select {
				case w.done <- struct{}{}:
				default:
				}
			}()
			w.process(ctx, p)
		}(p, slot)
	}
}

// order returns the ready items in dispatch order: manual first (oldest
// first), then one item per source in turn, NEWEST first within a source:
// a slice of today's session is worth more to the next recall than one
// from last month, and a backlog drains from the present backwards.
func order(items []store.PendingEpisode, now time.Time) []store.PendingEpisode {
	var manual []store.PendingEpisode
	bySource := map[string][]store.PendingEpisode{}
	var sources []string
	for _, p := range items {
		if p.Parked || p.NextAttempt.After(now) {
			continue
		}
		if p.Source == "manual" {
			manual = append(manual, p)
			continue
		}
		if _, ok := bySource[p.Source]; !ok {
			sources = append(sources, p.Source)
		}
		bySource[p.Source] = append(bySource[p.Source], p)
	}
	sort.SliceStable(manual, func(i, j int) bool { return manual[i].EnqueuedAt.Before(manual[j].EnqueuedAt) })
	sort.Strings(sources)
	for _, s := range sources {
		q := bySource[s]
		sort.SliceStable(q, func(i, j int) bool {
			if !q[i].OccurredAt.Equal(q[j].OccurredAt) {
				return q[i].OccurredAt.After(q[j].OccurredAt)
			}
			return q[i].EnqueuedAt.Before(q[j].EnqueuedAt)
		})
	}
	out := append([]store.PendingEpisode{}, manual...)
	for {
		took := false
		for _, s := range sources {
			if q := bySource[s]; len(q) > 0 {
				out = append(out, q[0])
				bySource[s] = q[1:]
				took = true
			}
		}
		if !took {
			break
		}
	}
	return out
}

// process extracts one item and either resolves it or records the failure
// for a later attempt.
func (w *Worker) process(ctx context.Context, p store.PendingEpisode) {
	ictx, cancel := context.WithTimeout(ctx, itemDeadline(w.o.ItemTimeout, p.Attempts))
	defer cancel()

	ep := distill.RawEpisode{ID: p.ID, Source: p.Source, SourceRef: p.SourceRef, Text: p.Text,
		OccurredAt: p.OccurredAt, Cwd: p.Cwd}
	var glossary []string
	if w.o.Glossary != nil {
		glossary = w.o.Glossary()
	}
	glossary = append(glossary, p.Hints...)

	res, err := w.o.Extractor.Extract(ictx, ep, glossary)
	if err != nil {
		if ctx.Err() != nil {
			// The daemon is shutting down; the item stays as it was and the
			// next daemon picks it up. Not a failure of the item.
			return
		}
		w.fail(p, err)
		return
	}

	summary := res.EpisodeSummary
	if p.Source == "manual" {
		// A remembered fact is its own best summary. The model's paraphrase
		// drops the specifics the agent chose to write down.
		summary = p.Text
	}
	cwd := ""
	if p.CwdIsRepo {
		cwd = p.Cwd // attested as a repository by the machine that has it
	}
	stats, err := resolve.ApplyWith(w.o.Store, store.Episode{
		ID: p.ID, Source: p.Source, SourceRef: p.SourceRef, Summary: summary,
		OccurredAt: p.OccurredAt, IngestedAt: time.Now(),
		Cwd: p.Cwd, CwdIsRepo: p.CwdIsRepo,
	}, cwd, res, resolve.DefaultExclusive, resolve.ApplyOptions{Force: p.Force})
	if err != nil {
		w.fail(p, fmt.Errorf("resolve: %w", err))
		return
	}
	w.widen()
	if err := w.o.Store.DeletePending(p.ID); err != nil {
		w.o.Logf("memory queue: delete pending %s: %v", p.ID, err)
	}
	if err := w.o.Store.PutMetaTime(store.MetaLastExtract, time.Now()); err != nil {
		w.o.Logf("memory queue: stamp last extract: %v", err)
	}
	w.o.Logf("memory queue: resolved %s (%s): +%d facts, +%d entities, attempt %d",
		p.ID, p.Source, stats.FactsAdded, stats.EntitiesCreated, p.Attempts+1)
}

// fail records one failed attempt. Parse failures count toward parking,
// and so do timeouts on transcript episodes (too long for the chain).
// Everything else — a refused connection, a 5xx, a store error, and any
// failure on a manual item that is not a parse failure — is retried
// indefinitely, because none of those say anything about the episode.
func (w *Worker) fail(p store.PendingEpisode, cause error) {
	// A rate-limit refusal is not the item's fault and must not spend its
	// budget: the pool narrows, the item waits briefly, and its attempt
	// count stays where it was.
	if extract.IsRateLimited(cause) {
		w.narrow()
		p.NextAttempt = time.Now().Add(w.rateBackoff())
		p.LastError = truncate(cause.Error(), 500)
		if err := w.o.Store.PutPending(p); err != nil {
			w.o.Logf("memory queue: record rate limit for %s: %v", p.ID, err)
		}
		return
	}

	p.Attempts++
	p.LastError = truncate(cause.Error(), 500)
	parse := errors.Is(cause, extract.ErrParse)
	timeout := errors.Is(cause, context.DeadlineExceeded)
	switch {
	case parse && p.Attempts >= MaxParseAttempts:
		p.Parked = true
		w.o.Logf("memory queue: PARKED %s after %d unparseable replies: %v — replay with `scry memory queue retry %s`",
			p.ID, p.Attempts, cause, p.ID)
	case timeout && p.Source != "manual" && p.Attempts >= MaxTimeoutAttempts:
		// An episode the chain cannot finish is halved and re-queued, so
		// its content still lands: each half gets a fresh budget, and the
		// halves carry source refs derived from the parent's. Only after
		// maxSplitDepth halvings is it parked for good.
		if split, err := w.splitPending(p); err != nil {
			w.o.Logf("memory queue: split %s failed: %v", p.ID, err)
		} else if split {
			return
		}
		p.Parked = true
		w.o.Logf("memory queue: PARKED %s after %d timeouts (last deadline %s): too long for the chain — replay with `scry memory queue retry %s`",
			p.ID, p.Attempts, itemDeadline(w.o.ItemTimeout, p.Attempts-1), p.ID)
	default:
		p.NextAttempt = time.Now().Add(w.backoff(p.Attempts))
		kind := "transport"
		if parse {
			kind = "parse"
		} else if timeout {
			kind = "timeout"
		}
		w.o.Logf("memory queue: %s failure on %s (attempt %d, retry in %s): %v",
			kind, p.ID, p.Attempts, w.backoff(p.Attempts).Round(time.Second), cause)
	}
	if err := w.o.Store.PutPending(p); err != nil {
		w.o.Logf("memory queue: record failure for %s: %v", p.ID, err)
	}
}

// splitPending halves p's text at a turn boundary and queues both halves
// as new pending episodes, then removes p. It reports false when p is too
// short or already split too deeply, leaving the caller to park it.
func (w *Worker) splitPending(p store.PendingEpisode) (bool, error) {
	if len(p.Text) < minSplitChars || splitDepth(p.SourceRef) >= maxSplitDepth {
		return false, nil
	}
	left, right := splitText(p.Text)
	if left == "" || right == "" {
		return false, nil
	}
	now := time.Now()
	for i, half := range []string{left, right} {
		ref := fmt.Sprintf("%s#split%d", p.SourceRef, i+1)
		child := p
		child.ID = distill.MakeID(ref)
		child.SourceRef = ref
		child.Text = half
		child.Attempts = 0
		child.Parked = false
		child.LastError = "split from " + p.ID + " after repeated timeouts"
		child.EnqueuedAt = now
		child.NextAttempt = now
		if has, err := w.o.Store.HasEpisode(child.ID); err != nil {
			return false, err
		} else if has {
			continue
		}
		if err := w.o.Store.PutPending(child); err != nil {
			return false, err
		}
	}
	if err := w.o.Store.DeletePending(p.ID); err != nil {
		return false, err
	}
	w.o.Logf("memory queue: split %s (%d chars, depth %d) into two halves after %d timeouts",
		p.ID, len(p.Text), splitDepth(p.SourceRef), p.Attempts)
	w.Kick()
	return true, nil
}

// splitText cuts s near the middle, preferring a turn boundary (the blank
// line distill leaves between turns), then a line break, then a space.
func splitText(s string) (string, string) {
	mid := len(s) / 2
	for _, sep := range []string{"\n\n", "\n", " "} {
		if i := strings.LastIndex(s[:mid], sep); i > len(s)/8 {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+len(sep):])
		}
		if i := strings.Index(s[mid:], sep); i >= 0 && mid+i < len(s)-len(s)/8 {
			return strings.TrimSpace(s[:mid+i]), strings.TrimSpace(s[mid+i+len(sep):])
		}
	}
	return "", ""
}

// splitDepth counts the "#splitN" suffixes on a source ref.
func splitDepth(ref string) int { return strings.Count(ref, "#split") }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
