package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
)

// newTestMemoryDaemon builds a Daemon rooted at a temp scry home, with no
// extractor configured (dormant) unless the caller sets one afterward. It is
// never Run(), so only the memory store (lazily opened) and its handlers are
// exercised.
func newTestMemoryDaemon(t *testing.T) *Daemon {
	t.Helper()
	home := t.TempDir()
	d := New(LayoutFor(home))
	// Force dormant regardless of the test runner's environment, so tests
	// are deterministic even if SCRY_MEMORY_API_KEY/ANTHROPIC_API_KEY happen
	// to be set.
	d.memExtractor = nil
	t.Cleanup(func() {
		d.closeMemory()
	})
	return d
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// fakeExtractor is a canned Extractor for exercising memory.remember's
// non-dormant path without hitting the real Anthropic API. calls records
// every RawEpisode passed in, so tests can assert on what memory.remember
// actually sent to extraction (e.g. that it was redacted first).
type fakeExtractor struct {
	result extract.Result
	err    error
	delay  time.Duration
	mu     sync.Mutex
	calls  []distill.RawEpisode
}

func (f *fakeExtractor) Extract(_ context.Context, ep distill.RawEpisode, _ []string) (extract.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, ep)
	f.mu.Unlock()
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return f.result, f.err
}

func TestMemoryCommitThenRecallRoundtrip(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()

	now := time.Now()
	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{
			ID:         "ep-1",
			Source:     "manual",
			SourceRef:  "manual",
			Summary:    "book-system deployed to hermes-mini",
			OccurredAt: now,
			IngestedAt: now,
		},
		Cwd: "",
		Result: extract.Result{
			EpisodeSummary: "book-system deployed to hermes-mini",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service", Description: "canonical book pipeline"},
				{Name: "hermes-mini", Type: "machine", Description: "mac mini running hermes"},
			},
			Facts: []extract.Fct{
				{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "book-system runs on hermes-mini", Confidence: 0.9},
			},
		},
	}

	res, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams))
	if err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}
	stats, ok := res.(resolve.Stats)
	if !ok {
		t.Fatalf("handleMemoryCommit result type = %T, want resolve.Stats", res)
	}
	if stats.EntitiesCreated != 2 {
		t.Errorf("EntitiesCreated = %d, want 2", stats.EntitiesCreated)
	}
	if stats.FactsAdded != 1 {
		t.Errorf("FactsAdded = %d, want 1", stats.FactsAdded)
	}

	// recall should find book-system.
	recallRes, err := d.handleMemoryRecall(ctx, mustJSON(t, MemoryRecallParams{Query: "book-system"}))
	if err != nil {
		t.Fatalf("handleMemoryRecall: %v", err)
	}
	b, err := json.Marshal(recallRes)
	if err != nil {
		t.Fatalf("marshal recall result: %v", err)
	}
	var hits struct {
		Entities []struct {
			Slug string `json:"slug"`
		} `json:"entities"`
		Facts []struct {
			Relation string `json:"relation"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(b, &hits); err != nil {
		t.Fatalf("unmarshal recall result: %v", err)
	}
	if len(hits.Entities) == 0 || hits.Entities[0].Slug != "book-system" {
		t.Fatalf("recall hits = %+v, want one hit for book-system", hits)
	}
	if len(hits.Facts) != 1 || hits.Facts[0].Relation != "deployed_on" {
		t.Fatalf("recall facts = %+v, want one deployed_on fact", hits.Facts)
	}
}

func TestMemoryRecallFindsCommittedEntity(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-2", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "hermes-mini", Type: "machine"},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	res, err := d.handleMemoryRecall(ctx, mustJSON(t, MemoryRecallParams{Query: "hermes-mini"}))
	if err != nil {
		t.Fatalf("handleMemoryRecall: %v", err)
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal recall result: %v", err)
	}
	var hits struct {
		Entities []struct {
			Slug string `json:"slug"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(b, &hits); err != nil {
		t.Fatalf("unmarshal recall result: %v", err)
	}
	if len(hits.Entities) == 0 || hits.Entities[0].Slug != "hermes-mini" {
		t.Fatalf("recall hits = %+v, want one hit for hermes-mini", hits)
	}
}

func TestMemoryGlossaryContainsCommittedEntity(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-3", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service", Aliases: []string{"book pipeline"}},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	res, err := d.handleMemoryGlossary(ctx, mustJSON(t, MemoryGlossaryParams{}))
	if err != nil {
		t.Fatalf("handleMemoryGlossary: %v", err)
	}
	lines, ok := res.([]string)
	if !ok {
		t.Fatalf("handleMemoryGlossary result type = %T, want []string", res)
	}
	want := "book-system: book pipeline"
	found := false
	for _, l := range lines {
		if l == want {
			found = true
		}
	}
	if !found {
		t.Errorf("glossary lines = %v, want to contain %q", lines, want)
	}
}

func TestMemoryGlossaryOmitsTrailingColonWhenNoAliases(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-4", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "solo-entity", Type: "concept"},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	res, err := d.handleMemoryGlossary(ctx, mustJSON(t, MemoryGlossaryParams{}))
	if err != nil {
		t.Fatalf("handleMemoryGlossary: %v", err)
	}
	lines := res.([]string)
	found := false
	for _, l := range lines {
		if l == "solo-entity" {
			found = true
		}
		if l == "solo-entity:" || l == "solo-entity: " {
			t.Errorf("glossary line for no-alias entity has trailing separator: %q", l)
		}
	}
	if !found {
		t.Errorf("glossary lines = %v, want to contain bare %q", lines, "solo-entity")
	}
}

func TestMemoryStatusCounts(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-5", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service"},
				{Name: "hermes-mini", Type: "machine"},
			},
			Facts: []extract.Fct{
				{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "runs there", Confidence: 0.8},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	res, err := d.handleMemoryStatus(ctx, nil)
	if err != nil {
		t.Fatalf("handleMemoryStatus: %v", err)
	}
	status, ok := res.(*MemoryStatusResult)
	if !ok {
		t.Fatalf("handleMemoryStatus result type = %T, want *MemoryStatusResult", res)
	}
	if status.Episodes != 1 {
		t.Errorf("Episodes = %d, want 1", status.Episodes)
	}
	if status.Entities != 2 {
		t.Errorf("Entities = %d, want 2", status.Entities)
	}
	if status.Facts != 1 {
		t.Errorf("Facts = %d, want 1", status.Facts)
	}
	if !status.Dormant {
		t.Errorf("Dormant = false, want true (no extractor configured)")
	}
	if status.Cursors != 0 {
		t.Errorf("Cursors = %d, want 0", status.Cursors)
	}
}

func TestMemoryCursorRoundtrip(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()

	// Before Put: Found should be false.
	getRes, err := d.handleMemoryCursorGet(ctx, mustJSON(t, MemoryCursorGetParams{Path: "/some/file.jsonl"}))
	if err != nil {
		t.Fatalf("handleMemoryCursorGet (before put): %v", err)
	}
	before := getRes.(*MemoryCursorGetResult)
	if before.Found {
		t.Errorf("Found = true before any Put, want false")
	}

	cursor := memstore.Cursor{
		Path:           "/some/file.jsonl",
		Size:           1024,
		ModTime:        time.Now().Truncate(time.Second),
		ProcessedBytes: 512,
	}
	if _, err := d.handleMemoryCursorPut(ctx, mustJSON(t, cursor)); err != nil {
		t.Fatalf("handleMemoryCursorPut: %v", err)
	}

	getRes2, err := d.handleMemoryCursorGet(ctx, mustJSON(t, MemoryCursorGetParams{Path: "/some/file.jsonl"}))
	if err != nil {
		t.Fatalf("handleMemoryCursorGet (after put): %v", err)
	}
	after := getRes2.(*MemoryCursorGetResult)
	if !after.Found {
		t.Fatalf("Found = false after Put, want true")
	}
	if after.Cursor.ProcessedBytes != 512 {
		t.Errorf("ProcessedBytes = %d, want 512", after.Cursor.ProcessedBytes)
	}
	if after.Cursor.Size != 1024 {
		t.Errorf("Size = %d, want 1024", after.Cursor.Size)
	}
}

func TestMemoryRememberDormantQueuesWithoutError(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()

	res, err := d.handleMemoryRemember(ctx, mustJSON(t, MemoryRememberParams{Fact: "we deploy from the hermes mini"}))
	if err != nil {
		t.Fatalf("handleMemoryRemember (dormant): %v", err)
	}
	result, ok := res.(*MemoryRememberResult)
	if !ok {
		t.Fatalf("handleMemoryRemember result type = %T, want *MemoryRememberResult", res)
	}
	if !result.Dormant || !result.Queued || result.EpisodeID == "" || result.QueueDepth != 1 {
		t.Errorf("result = %+v, want dormant, queued, depth 1", result)
	}

	statusRes, err := d.handleMemoryStatus(ctx, nil)
	if err != nil {
		t.Fatalf("handleMemoryStatus: %v", err)
	}
	status := statusRes.(*MemoryStatusResult)
	if status.Episodes != 0 || status.QueueReady != 1 || !status.Dormant || status.WorkerRunning {
		t.Errorf("status = %+v, want 0 episodes, 1 queued, dormant, no worker", status)
	}
}

func TestMemoryRememberReturnsBeforeExtractionAndResolvesInBackground(t *testing.T) {
	d := newTestMemoryDaemon(t)
	d.memExtractor = &fakeExtractor{
		delay: 300 * time.Millisecond,
		result: extract.Result{
			EpisodeSummary: "book-system deployed to hermes-mini",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service"},
				{Name: "hermes-mini", Type: "machine"},
			},
			Facts: []extract.Fct{
				{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "runs there", Confidence: 0.9},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.startMemoryWorker(ctx)
	for d.memoryWorker() == nil {
		time.Sleep(5 * time.Millisecond)
	}

	start := time.Now()
	res, err := d.handleMemoryRemember(ctx, mustJSON(t, MemoryRememberParams{Fact: "book-system deployed to hermes-mini"}))
	if err != nil {
		t.Fatalf("handleMemoryRemember: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("remember took %s; it must return before extraction runs", elapsed)
	}
	result := res.(*MemoryRememberResult)
	if result.Dormant || !result.Queued {
		t.Errorf("result = %+v", result)
	}

	st, _ := d.memoryStore()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if has, _ := st.HasEpisode(result.EpisodeID); has {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	facts, _ := st.FactsFrom("book-system", false)
	if len(facts) != 1 {
		t.Fatalf("facts on book-system = %d, want 1 resolved in the background", len(facts))
	}
	if has, _ := st.HasPending(result.EpisodeID); has {
		t.Error("episode still pending after resolution")
	}
	cancel()
	d.closeMemory()
}

func TestMemoryRememberSameFactSameDayIsOneEpisode(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	first, _ := d.handleMemoryRemember(ctx, mustJSON(t, MemoryRememberParams{Fact: "scry lives on the mini"}))
	second, _ := d.handleMemoryRemember(ctx, mustJSON(t, MemoryRememberParams{Fact: "scry lives on the mini"}))
	a, b := first.(*MemoryRememberResult), second.(*MemoryRememberResult)
	if a.EpisodeID != b.EpisodeID || a.Known || !b.Known || b.QueueDepth != 1 {
		t.Errorf("first = %+v, second = %+v; want the same episode, second marked known", a, b)
	}
}

// TestMemoryRememberRedactsFact covers F3: memory.remember must redact
// secret-shaped text out of p.Fact before it is stored in the queue, so it
// never reaches a model or the episode summary un-redacted.
func TestMemoryRememberRedactsFact(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()

	secretFact := "the deploy key is sk-abcdefghijklmnopqrstuvwxyz123456"
	res, err := d.handleMemoryRemember(ctx, mustJSON(t, MemoryRememberParams{Fact: secretFact}))
	if err != nil {
		t.Fatalf("handleMemoryRemember: %v", err)
	}
	st, err := d.memoryStore()
	if err != nil {
		t.Fatalf("memoryStore: %v", err)
	}
	p, err := st.GetPending(res.(*MemoryRememberResult).EpisodeID)
	if err != nil {
		t.Fatalf("GetPending: %v", err)
	}
	if strings.Contains(p.Text, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Errorf("un-redacted secret queued: %q", p.Text)
	}
	if !strings.Contains(p.Text, "[REDACTED]") {
		t.Errorf("expected the queued text to contain the redaction marker; got: %q", p.Text)
	}
}

func TestMemoryEnqueueDedupesAgainstStoreAndQueue(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	st, _ := d.memoryStore()
	now := time.Now()
	if err := st.PutEpisode(memstore.Episode{ID: "known", Source: "claude-session", OccurredAt: now, IngestedAt: now}); err != nil {
		t.Fatal(err)
	}
	params := MemoryEnqueueParams{Episodes: []distill.RawEpisode{
		{ID: "known", Source: "claude-session", Text: "already resolved"},
		{ID: "new1", Source: "claude-session", Text: "fresh", OccurredAt: now},
		{ID: "new1", Source: "claude-session", Text: "fresh again", OccurredAt: now},
	}}
	res, err := d.handleMemoryEnqueue(ctx, mustJSON(t, params))
	if err != nil {
		t.Fatalf("handleMemoryEnqueue: %v", err)
	}
	r := res.(*MemoryEnqueueResult)
	if r.Queued != 1 || r.Known != 2 {
		t.Errorf("enqueue = %+v, want queued 1 known 2", r)
	}
	if _, found, _ := st.GetMetaTime(memstore.MetaLastIngest); !found {
		t.Error("MetaLastIngest not stamped after a queued episode")
	}
	if _, err := d.handleMemoryEnqueue(ctx, mustJSON(t, MemoryEnqueueParams{Episodes: []distill.RawEpisode{{Text: "no id"}}})); err == nil {
		t.Error("enqueue without an id must be rejected")
	}
}

func TestMemoryQueueListsAndRetriesParkedItems(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	st, _ := d.memoryStore()
	now := time.Now()
	_ = st.PutPending(memstore.PendingEpisode{ID: "parked", EnqueuedAt: now, NextAttempt: now, Parked: true, Attempts: 3, Text: strings.Repeat("x", 500)})
	_ = st.PutPending(memstore.PendingEpisode{ID: "ready", EnqueuedAt: now, NextAttempt: now})

	res, err := d.handleMemoryQueue(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := res.(*MemoryQueueResult)
	if q.Ready != 1 || q.Parked != 1 || len(q.Items) != 2 || len(q.Items[0].Text) > 210 {
		t.Errorf("queue = ready %d parked %d items %d (text %d)", q.Ready, q.Parked, len(q.Items), len(q.Items[0].Text))
	}

	rr, err := d.handleMemoryQueueRetry(ctx, mustJSON(t, MemoryQueueRetryParams{ID: "parked"}))
	if err != nil {
		t.Fatal(err)
	}
	if rr.(map[string]int)["retried"] != 1 {
		t.Errorf("retry = %v, want 1", rr)
	}
	p, _ := st.GetPending("parked")
	if p.Parked || p.Attempts != 0 {
		t.Errorf("after retry: %+v", p)
	}
}

func TestMemorySweepReportStampsLastSweep(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	if _, err := d.handleMemorySweepReport(ctx, mustJSON(t, MemorySweepReport{Host: "laptop", FilesScanned: 10, FilesIngested: 2})); err != nil {
		t.Fatal(err)
	}
	statusRes, _ := d.handleMemoryStatus(ctx, nil)
	status := statusRes.(*MemoryStatusResult)
	if status.LastSweepAt == nil || time.Since(*status.LastSweepAt) > time.Minute {
		t.Errorf("LastSweepAt = %v", status.LastSweepAt)
	}
	st, _ := d.memoryStore()
	var rep MemorySweepReport
	if found, _ := st.GetMetaJSON(memstore.MetaLastSweepReport, &rep); !found || rep.FilesIngested != 2 || rep.Host != "laptop" {
		t.Errorf("stored report = %+v found=%v", rep, found)
	}
}

func TestMemoryFactsUnknownEntityReturnsError(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()

	if _, err := d.handleMemoryFacts(ctx, mustJSON(t, MemoryFactsParams{Slug: "does-not-exist"})); err == nil {
		t.Fatal("handleMemoryFacts: expected error for unknown entity, got nil")
	}
}

func TestMemoryEpisodesUnknownEntityReturnsError(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()

	if _, err := d.handleMemoryEpisodes(ctx, mustJSON(t, MemoryEpisodesParams{Entity: "does-not-exist"})); err == nil {
		t.Fatal("handleMemoryEpisodes: expected error for unknown entity, got nil")
	}
}

// TestMemoryFactsAndEpisodesResolveAlias covers F6: memory.facts and
// memory.episodes must accept an alias, not just an exact slug, since their
// MCP tool descriptions (scry_recall et al.) promise name-or-slug input.
func TestMemoryFactsAndEpisodesResolveAlias(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-alias", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "book-system deployed to hermes-mini",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service", Aliases: []string{"book-sys"}},
				{Name: "hermes-mini", Type: "machine"},
			},
			Facts: []extract.Fct{
				{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "runs there", Confidence: 0.8},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	factsRes, err := d.handleMemoryFacts(ctx, mustJSON(t, MemoryFactsParams{Slug: "book-sys"}))
	if err != nil {
		t.Fatalf("handleMemoryFacts(alias %q): %v", "book-sys", err)
	}
	facts := factsRes.([]memstore.Fact)
	if len(facts) != 1 {
		t.Fatalf("facts via alias = %d, want 1", len(facts))
	}

	episodesRes, err := d.handleMemoryEpisodes(ctx, mustJSON(t, MemoryEpisodesParams{Entity: "book-sys"}))
	if err != nil {
		t.Fatalf("handleMemoryEpisodes(alias %q): %v", "book-sys", err)
	}
	eps := episodesRes.([]memstore.Episode)
	if len(eps) != 1 {
		t.Fatalf("episodes via alias = %d, want 1", len(eps))
	}
	if eps[0].ID != "ep-alias" {
		t.Errorf("episode ID = %q, want %q", eps[0].ID, "ep-alias")
	}
}

// TestMemoryHasEpisodes covers F5's RPC: it must report exactly the
// requested IDs that are NOT yet committed, leaving already-known ones out
// of Missing.
func TestMemoryHasEpisodes(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "known-ep", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result:  extract.Result{EpisodeSummary: "seed"},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	res, err := d.handleMemoryHasEpisodes(ctx, mustJSON(t, MemoryHasEpisodesParams{
		IDs: []string{"known-ep", "unknown-ep-1", "unknown-ep-2"},
	}))
	if err != nil {
		t.Fatalf("handleMemoryHasEpisodes: %v", err)
	}
	result, ok := res.(*MemoryHasEpisodesResult)
	if !ok {
		t.Fatalf("handleMemoryHasEpisodes result type = %T, want *MemoryHasEpisodesResult", res)
	}
	want := map[string]bool{"unknown-ep-1": true, "unknown-ep-2": true}
	if len(result.Missing) != len(want) {
		t.Fatalf("Missing = %v, want exactly %v", result.Missing, want)
	}
	for _, id := range result.Missing {
		if !want[id] {
			t.Errorf("Missing contains unexpected id %q", id)
		}
		if id == "known-ep" {
			t.Errorf("Missing contains %q, which was already committed", id)
		}
	}
}

func TestMemoryExportIncludesEntitiesFactsAndEpisodesWithHistory(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-export-1", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service"},
				{Name: "hermes-mini", Type: "machine"},
			},
			Facts: []extract.Fct{
				{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "runs there", Confidence: 0.8},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	// Invalidate the fact so the export must surface both a current and a
	// superseded fact (the UI shows history, per the RPC's contract).
	if _, err := d.handleMemoryInvalidate(ctx, mustJSON(t, MemoryInvalidateParams{
		Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini",
	})); err != nil {
		t.Fatalf("handleMemoryInvalidate: %v", err)
	}

	res, err := d.handleMemoryExport(ctx, nil)
	if err != nil {
		t.Fatalf("handleMemoryExport: %v", err)
	}
	export, ok := res.(*MemoryExportResult)
	if !ok {
		t.Fatalf("handleMemoryExport result type = %T, want *MemoryExportResult", res)
	}

	if len(export.Entities) != 2 {
		t.Errorf("Entities = %d, want 2: %+v", len(export.Entities), export.Entities)
	}
	if len(export.Episodes) != 1 || export.Episodes[0].ID != "ep-export-1" {
		t.Errorf("Episodes = %+v, want exactly [ep-export-1]", export.Episodes)
	}
	if len(export.Facts) != 1 {
		t.Fatalf("Facts = %d, want 1 (the sole fact, now invalidated), got %+v", len(export.Facts), export.Facts)
	}
	if export.Facts[0].InvalidAt == nil {
		t.Errorf("Facts[0].InvalidAt = nil, want set — export must include invalidated facts for history")
	}
	if export.GeneratedAt.IsZero() {
		t.Errorf("GeneratedAt is zero, want set")
	}
}

func TestMemoryInvalidateAllCurrentMatches(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	now := time.Now()

	commitParams := MemoryCommitParams{
		Episode: memstore.Episode{ID: "ep-6", Source: "manual", SourceRef: "manual", OccurredAt: now, IngestedAt: now},
		Result: extract.Result{
			EpisodeSummary: "seed",
			Entities: []extract.Ent{
				{Name: "book-system", Type: "service"},
				{Name: "hermes-mini", Type: "machine"},
			},
			Facts: []extract.Fct{
				{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini", Fact: "runs there", Confidence: 0.8},
			},
		},
	}
	if _, err := d.handleMemoryCommit(ctx, mustJSON(t, commitParams)); err != nil {
		t.Fatalf("handleMemoryCommit: %v", err)
	}

	res, err := d.handleMemoryInvalidate(ctx, mustJSON(t, MemoryInvalidateParams{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini"}))
	if err != nil {
		t.Fatalf("handleMemoryInvalidate: %v", err)
	}
	counts, ok := res.(map[string]int)
	if !ok {
		t.Fatalf("handleMemoryInvalidate result type = %T, want map[string]int", res)
	}
	if counts["invalidated"] != 1 {
		t.Errorf("invalidated = %d, want 1", counts["invalidated"])
	}

	facts, err := d.handleMemoryFacts(ctx, mustJSON(t, MemoryFactsParams{Slug: "book-system"}))
	if err != nil {
		t.Fatalf("handleMemoryFacts: %v", err)
	}
	current := facts.([]memstore.Fact)
	if len(current) != 0 {
		t.Errorf("current facts after invalidate = %+v, want none", current)
	}
}

func TestMemoryBackupWritesAFile(t *testing.T) {
	d := newTestMemoryDaemon(t)
	ctx := context.Background()
	st, _ := d.memoryStore()
	_ = st.PutEpisode(memstore.Episode{ID: "e1", Source: "manual"})
	res, err := d.handleMemoryBackup(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := res.(*MemoryBackupResult)
	if !strings.HasPrefix(r.Path, filepath.Join(d.layout.Home, "backups", "memory-")) || r.Bytes == 0 {
		t.Errorf("backup = %+v", r)
	}
	if _, err := os.Stat(r.Path); err != nil {
		t.Errorf("backup file missing: %v", err)
	}
}
