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

	defaultWorkers     = 4
	defaultItemTimeout = 4 * time.Minute
	defaultPoll        = 2 * time.Second
	backoffBase        = 30 * time.Second
	backoffCap         = 2 * time.Minute
)

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
	// Workers is the number of items extracted concurrently (default 4).
	Workers int
	// ItemTimeout bounds one item's extraction (default 4m).
	ItemTimeout time.Duration
	// Poll is how often the loop looks for ready items when nothing kicks
	// it (default 2s).
	Poll time.Duration
	Logf func(format string, args ...any)
}

// Worker drains the pending queue.
type Worker struct {
	o       Options
	kick    chan struct{}
	backoff func(int) time.Duration

	mu       sync.Mutex
	inflight map[string]bool
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
	return &Worker{o: o, kick: make(chan struct{}, 1), backoff: Backoff, inflight: map[string]bool{}}
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
	var wg sync.WaitGroup
	ticker := time.NewTicker(w.o.Poll)
	defer ticker.Stop()
	for {
		w.dispatch(ctx, sem, &wg)
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
		case <-w.kick:
		}
	}
}

// dispatch claims every ready item that fits in the free worker slots.
func (w *Worker) dispatch(ctx context.Context, sem chan struct{}, wg *sync.WaitGroup) {
	items, err := w.o.Store.Pending(0)
	if err != nil {
		w.o.Logf("memory queue: list pending: %v", err)
		return
	}
	now := time.Now()
	for _, p := range items {
		if p.Parked || p.NextAttempt.After(now) {
			continue
		}
		w.mu.Lock()
		if w.inflight[p.ID] {
			w.mu.Unlock()
			continue
		}
		select {
		case sem <- struct{}{}:
		default:
			w.mu.Unlock()
			return // every slot busy; the next tick picks up the rest
		}
		w.inflight[p.ID] = true
		w.mu.Unlock()

		wg.Add(1)
		go func(p store.PendingEpisode) {
			defer wg.Done()
			defer func() {
				w.mu.Lock()
				delete(w.inflight, p.ID)
				w.mu.Unlock()
				<-sem
			}()
			w.process(ctx, p)
		}(p)
	}
}

// process extracts one item and either resolves it or records the failure
// for a later attempt.
func (w *Worker) process(ctx context.Context, p store.PendingEpisode) {
	ictx, cancel := context.WithTimeout(ctx, w.o.ItemTimeout)
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
		w.fail(p, err)
		return
	}

	summary := res.EpisodeSummary
	if p.Source == "manual" {
		// A remembered fact is its own best summary. The model's paraphrase
		// drops the specifics the agent chose to write down.
		summary = p.Text
	}
	stats, err := resolve.Apply(w.o.Store, store.Episode{
		ID: p.ID, Source: p.Source, SourceRef: p.SourceRef, Summary: summary,
		OccurredAt: p.OccurredAt, IngestedAt: time.Now(),
	}, p.Cwd, res, resolve.DefaultExclusive)
	if err != nil {
		w.fail(p, fmt.Errorf("resolve: %w", err))
		return
	}
	if err := w.o.Store.DeletePending(p.ID); err != nil {
		w.o.Logf("memory queue: delete pending %s: %v", p.ID, err)
	}
	if err := w.o.Store.PutMetaTime(store.MetaLastExtract, time.Now()); err != nil {
		w.o.Logf("memory queue: stamp last extract: %v", err)
	}
	w.o.Logf("memory queue: resolved %s (%s): +%d facts, +%d entities, attempt %d",
		p.ID, p.Source, stats.FactsAdded, stats.EntitiesCreated, p.Attempts+1)
}

// fail records one failed attempt. Parse failures count toward parking;
// everything else — a timeout, a refused connection, a 5xx, a store error
// — is retried indefinitely, because none of those say anything about the
// episode itself.
func (w *Worker) fail(p store.PendingEpisode, cause error) {
	p.Attempts++
	p.LastError = truncate(cause.Error(), 500)
	parse := errors.Is(cause, extract.ErrParse)
	if parse && p.Attempts >= MaxParseAttempts {
		p.Parked = true
		w.o.Logf("memory queue: PARKED %s after %d unparseable replies: %v — replay with `scry memory queue retry %s`",
			p.ID, p.Attempts, cause, p.ID)
	} else {
		p.NextAttempt = time.Now().Add(w.backoff(p.Attempts))
		kind := "transport"
		if parse {
			kind = "parse"
		}
		w.o.Logf("memory queue: %s failure on %s (attempt %d, retry in %s): %v",
			kind, p.ID, p.Attempts, w.backoff(p.Attempts).Round(time.Second), cause)
	}
	if err := w.o.Store.PutPending(p); err != nil {
		w.o.Logf("memory queue: record failure for %s: %v", p.ID, err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
