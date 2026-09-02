package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/queue"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// glossaryTTL bounds how stale the cached glossary may be. Computing it
// walks every entity's facts — seconds on a store this size — so it never
// runs on a request path; the worker reads whatever the last refresh
// produced and a background refresh runs when that is older than this.
const glossaryTTL = 10 * time.Minute

// glossaryCache holds the "slug: aliases" lines the extractor is shown.
type glossaryCache struct {
	mu         sync.Mutex
	lines      []string
	at         time.Time
	refreshing bool
}

// startMemoryWorker builds and runs the queue worker for the lifetime of
// ctx. With no extractor the daemon is dormant: writes still queue, and a
// restart with a key drains them.
//
// Opening the store happens off the startup path: the socket must answer
// before a Badger open (or a lock held by a retiring incumbent) has run its
// course, so the worker is built in a goroutine that closeMemory waits on.
func (d *Daemon) startMemoryWorker(ctx context.Context) {
	if d.memExtractor == nil {
		log.Printf("memory queue: not started — extraction is dormant; queued writes wait for a configured key")
		return
	}
	d.memQueueWG.Add(1)
	go func() {
		defer d.memQueueWG.Done()
		st, err := d.memoryStore()
		if err != nil {
			log.Printf("memory queue: not started — cannot open store: %v", err)
			return
		}
		if ctx.Err() != nil {
			return
		}
		w := queue.New(queue.Options{
			Store:     st,
			Extractor: d.memExtractor,
			Glossary:  d.glossaryLines,
		})
		d.memQueueMu.Lock()
		d.memQueue = w
		d.memQueueMu.Unlock()
		log.Printf("memory queue: worker started")
		w.Run(ctx)
	}()
}

// memoryWorker returns the running worker, or nil.
func (d *Daemon) memoryWorker() *queue.Worker {
	d.memQueueMu.Lock()
	defer d.memQueueMu.Unlock()
	return d.memQueue
}

// kickMemoryWorker wakes the worker after an enqueue, if one is running.
func (d *Daemon) kickMemoryWorker() {
	if w := d.memoryWorker(); w != nil {
		w.Kick()
	}
}

// glossaryLines returns the cached glossary, triggering a background
// refresh when it is stale. The first call returns whatever exists (possibly
// nothing) rather than blocking an extraction on a full-store walk.
func (d *Daemon) glossaryLines() []string {
	d.memGlossary.mu.Lock()
	lines := d.memGlossary.lines
	stale := time.Since(d.memGlossary.at) > glossaryTTL
	if stale && !d.memGlossary.refreshing {
		d.memGlossary.refreshing = true
		d.memQueueWG.Add(1)
		go func() {
			defer d.memQueueWG.Done()
			d.refreshGlossary()
		}()
	}
	d.memGlossary.mu.Unlock()
	return lines
}

func (d *Daemon) refreshGlossary() {
	st, err := d.memoryStore()
	var lines []string
	if err == nil {
		lines, err = computeGlossary(st, defaultGlossaryLimit)
	}
	d.memGlossary.mu.Lock()
	defer d.memGlossary.mu.Unlock()
	d.memGlossary.refreshing = false
	if err != nil {
		log.Printf("memory: glossary refresh failed: %v", err)
		return
	}
	d.memGlossary.lines = lines
	d.memGlossary.at = time.Now()
}

// computeGlossary ranks entities by current-fact degree and renders the
// top limit as "slug: alias1, alias2" lines.
func computeGlossary(st *memstore.Store, limit int) ([]string, error) {
	entities, err := st.Entities()
	if err != nil {
		return nil, err
	}
	type ranked struct {
		entity memstore.Entity
		degree int
	}
	rankedEntities := make([]ranked, 0, len(entities))
	for _, e := range entities {
		facts, err := st.FactsAbout(e.Slug, false)
		if err != nil {
			return nil, err
		}
		rankedEntities = append(rankedEntities, ranked{entity: e, degree: len(facts)})
	}
	sort.SliceStable(rankedEntities, func(i, j int) bool {
		return rankedEntities[i].degree > rankedEntities[j].degree
	})
	if len(rankedEntities) > limit {
		rankedEntities = rankedEntities[:limit]
	}
	lines := make([]string, 0, len(rankedEntities))
	for _, r := range rankedEntities {
		line := r.entity.Slug
		if len(r.entity.Aliases) > 0 {
			line += ": " + strings.Join(r.entity.Aliases, ", ")
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// --- memory.enqueue ---

// MemoryEnqueueParams carries distilled episodes from a client (the sweep,
// `scry memory ingest`) to the daemon's queue. The client never talks to a
// model.
type MemoryEnqueueParams struct {
	Episodes []distill.RawEpisode `json:"episodes"`
}

// MemoryEnqueueResult reports how many episodes were newly queued and how
// many were already known (queued, or already resolved into the store).
type MemoryEnqueueResult struct {
	Queued int `json:"queued"`
	Known  int `json:"known"`
}

func (d *Daemon) handleMemoryEnqueue(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryEnqueueParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	var res MemoryEnqueueResult
	now := time.Now()
	for _, ep := range p.Episodes {
		if ep.ID == "" {
			return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "episode id is required"}
		}
		queued, err := enqueueEpisode(st, ep, nil, now)
		if err != nil {
			return nil, err
		}
		if queued {
			res.Queued++
		} else {
			res.Known++
		}
	}
	if res.Queued > 0 {
		if err := st.PutMetaTime(memstore.MetaLastIngest, now); err != nil {
			return nil, err
		}
		d.kickMemoryWorker()
	}
	return &res, nil
}

// enqueueEpisode writes ep to the pending queue unless it is already there
// or already resolved. Text is redacted here as a backstop: every distiller
// redacts too, but this is the one door into the store.
func enqueueEpisode(st *memstore.Store, ep distill.RawEpisode, hints []string, now time.Time) (bool, error) {
	if has, err := st.HasEpisode(ep.ID); err != nil || has {
		return false, err
	}
	if has, err := st.HasPending(ep.ID); err != nil || has {
		return false, err
	}
	occurred := ep.OccurredAt
	if occurred.IsZero() {
		occurred = now
	}
	return true, st.PutPending(memstore.PendingEpisode{
		ID: ep.ID, Source: ep.Source, SourceRef: ep.SourceRef, Text: distill.Redact(ep.Text), Cwd: ep.Cwd,
		OccurredAt: occurred, EnqueuedAt: now, NextAttempt: now, Hints: hints,
	})
}

// --- memory.queue / memory.queue.retry ---

// MemoryQueueResult is the queue's state for `scry memory queue`.
type MemoryQueueResult struct {
	Ready   int                       `json:"ready"`
	Backoff int                       `json:"backoff"`
	Parked  int                       `json:"parked"`
	Items   []memstore.PendingEpisode `json:"items"`
}

const queueListLimit = 50

func (d *Daemon) handleMemoryQueue(_ context.Context, _ json.RawMessage) (any, error) {
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	ready, backoff, parked, err := st.PendingCounts(time.Now())
	if err != nil {
		return nil, err
	}
	items, err := st.Pending(queueListLimit)
	if err != nil {
		return nil, err
	}
	for i := range items {
		// The text can be a whole transcript slice; the listing only needs
		// enough to recognise the item.
		if len(items[i].Text) > 200 {
			items[i].Text = items[i].Text[:200] + "…"
		}
	}
	if items == nil {
		items = []memstore.PendingEpisode{}
	}
	return &MemoryQueueResult{Ready: ready, Backoff: backoff, Parked: parked, Items: items}, nil
}

// MemoryQueueRetryParams names one parked item to replay, or every parked
// item when ID is empty.
type MemoryQueueRetryParams struct {
	ID string `json:"id,omitempty"`
}

func (d *Daemon) handleMemoryQueueRetry(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryQueueRetryParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
		}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	items, err := st.Pending(0)
	if err != nil {
		return nil, err
	}
	retried := 0
	now := time.Now()
	for _, it := range items {
		if p.ID != "" && it.ID != p.ID {
			continue
		}
		if !it.Parked && !it.NextAttempt.After(now) {
			continue
		}
		it.Parked = false
		it.Attempts = 0
		it.NextAttempt = now
		if err := st.PutPending(it); err != nil {
			return nil, err
		}
		retried++
	}
	if retried > 0 {
		d.kickMemoryWorker()
	}
	return map[string]int{"retried": retried}, nil
}

// --- memory.sweepReport ---

// MemorySweepReport is what a sweep tells the daemon when it finishes, so
// "when did a sweep last run, and what did it find" is answerable from the
// store rather than from a log file on another machine.
type MemorySweepReport struct {
	Host          string    `json:"host,omitempty"`
	FilesScanned  int       `json:"files_scanned"`
	FilesIngested int       `json:"files_ingested"`
	Episodes      int       `json:"episodes"`
	Errors        int       `json:"errors"`
	FinishedAt    time.Time `json:"finished_at"`
}

func (d *Daemon) handleMemorySweepReport(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemorySweepReport
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	p.FinishedAt = time.Now()
	if err := st.PutMetaTime(memstore.MetaLastSweep, p.FinishedAt); err != nil {
		return nil, err
	}
	if err := st.PutMetaJSON(memstore.MetaLastSweepReport, p); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// --- memory.backup ---

// MemoryBackupParams names where the backup goes; empty means
// ~/.scry/backups/memory-<utc>.badger on the daemon's machine.
type MemoryBackupParams struct {
	Path string `json:"path,omitempty"`
}

// MemoryBackupResult reports the file written.
type MemoryBackupResult struct {
	Path  string `json:"path"`
	Bytes uint64 `json:"bytes"`
}

// handleMemoryBackup streams the live store to a file. It is the first step
// of every migration and cheap enough to run on request.
func (d *Daemon) handleMemoryBackup(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryBackupParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
		}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	path := p.Path
	if path == "" {
		path = filepath.Join(d.scryHome(), "backups", "memory-"+time.Now().UTC().Format("20060102T150405Z")+".badger")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	n, err := st.Backup(f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("backup to %s: %w", path, err)
	}
	log.Printf("memory: backup written to %s (%d bytes)", path, n)
	return &MemoryBackupResult{Path: path, Bytes: n}, nil
}
