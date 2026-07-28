package daemon

import (
	"context"
	"encoding/json"
	"strings"
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
	calls  []distill.RawEpisode
}

func (f *fakeExtractor) Extract(_ context.Context, ep distill.RawEpisode, _ []string) (extract.Result, error) {
	f.calls = append(f.calls, ep)
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
	var hits []struct {
		Entity struct {
			Slug string `json:"slug"`
		} `json:"entity"`
		Facts []struct {
			Relation string `json:"relation"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(b, &hits); err != nil {
		t.Fatalf("unmarshal recall result: %v", err)
	}
	if len(hits) != 1 || hits[0].Entity.Slug != "book-system" {
		t.Fatalf("recall hits = %+v, want one hit for book-system", hits)
	}
	if len(hits[0].Facts) != 1 || hits[0].Facts[0].Relation != "deployed_on" {
		t.Fatalf("recall facts = %+v, want one deployed_on fact", hits[0].Facts)
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
	var hits []struct {
		Entity struct {
			Slug string `json:"slug"`
		} `json:"entity"`
	}
	if err := json.Unmarshal(b, &hits); err != nil {
		t.Fatalf("unmarshal recall result: %v", err)
	}
	if len(hits) != 1 || hits[0].Entity.Slug != "hermes-mini" {
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
				{Name: "book-system", Type: "service", Aliases: []string{"the book pipeline"}},
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
	want := "book-system: the book pipeline"
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

func TestMemoryRememberDormantStoresEpisodeWithoutError(t *testing.T) {
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
	if !result.Dormant {
		t.Errorf("Dormant = false, want true — a dormant call must be distinguishable from one that genuinely resolved zero stats")
	}
	if result.Stats != (resolve.Stats{}) {
		t.Errorf("stats = %+v, want zero value in dormant mode", result.Stats)
	}

	// Status should show one episode ingested, zero entities/facts (dormant
	// never resolves), and Dormant true.
	statusRes, err := d.handleMemoryStatus(ctx, nil)
	if err != nil {
		t.Fatalf("handleMemoryStatus: %v", err)
	}
	status := statusRes.(*MemoryStatusResult)
	if status.Episodes != 1 {
		t.Errorf("Episodes = %d, want 1", status.Episodes)
	}
	if status.Entities != 0 {
		t.Errorf("Entities = %d, want 0 (dormant remember must not resolve)", status.Entities)
	}
	if !status.Dormant {
		t.Errorf("Dormant = false, want true")
	}
}

func TestMemoryRememberWithExtractorResolvesFacts(t *testing.T) {
	d := newTestMemoryDaemon(t)
	d.memExtractor = &fakeExtractor{
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
	ctx := context.Background()

	res, err := d.handleMemoryRemember(ctx, mustJSON(t, MemoryRememberParams{Fact: "book-system deployed to hermes-mini"}))
	if err != nil {
		t.Fatalf("handleMemoryRemember (with extractor): %v", err)
	}
	result, ok := res.(*MemoryRememberResult)
	if !ok {
		t.Fatalf("handleMemoryRemember result type = %T, want *MemoryRememberResult", res)
	}
	if result.Dormant {
		t.Errorf("Dormant = true, want false — an extractor is configured")
	}
	if result.Stats.EntitiesCreated != 2 {
		t.Errorf("EntitiesCreated = %d, want 2", result.Stats.EntitiesCreated)
	}
	if result.Stats.FactsAdded != 1 {
		t.Errorf("FactsAdded = %d, want 1", result.Stats.FactsAdded)
	}
}

// TestMemoryRememberRedactsFact covers F3: memory.remember must redact
// secret-shaped text out of p.Fact before any of it reaches the extraction
// API or gets stored — both the RawEpisode.Text sent to Extract and the
// episode's stored Summary.
func TestMemoryRememberRedactsFact(t *testing.T) {
	d := newTestMemoryDaemon(t)
	fake := &fakeExtractor{
		result: extract.Result{EpisodeSummary: "noted a token"},
	}
	d.memExtractor = fake
	ctx := context.Background()

	secretFact := "the deploy key is sk-abcdefghijklmnopqrstuvwxyz123456"
	if _, err := d.handleMemoryRemember(ctx, mustJSON(t, MemoryRememberParams{Fact: secretFact})); err != nil {
		t.Fatalf("handleMemoryRemember: %v", err)
	}

	if len(fake.calls) != 1 {
		t.Fatalf("extractor calls = %d, want 1", len(fake.calls))
	}
	if strings.Contains(fake.calls[0].Text, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Errorf("un-redacted secret sent to the extractor: %q", fake.calls[0].Text)
	}
	if !strings.Contains(fake.calls[0].Text, "[REDACTED]") {
		t.Errorf("expected the extractor's input to contain the redaction marker; got: %q", fake.calls[0].Text)
	}

	// The stored episode's Summary must be redacted too — reach past the RPC
	// layer directly into the store (same package) since no RPC exposes a
	// bare episode lookup by ID.
	st, err := d.memoryStore()
	if err != nil {
		t.Fatalf("memoryStore: %v", err)
	}
	ep, err := st.GetEpisode(fake.calls[0].ID)
	if err != nil {
		t.Fatalf("GetEpisode(%s): %v", fake.calls[0].ID, err)
	}
	if strings.Contains(ep.Summary, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Errorf("un-redacted secret stored in episode Summary: %q", ep.Summary)
	}
	if !strings.Contains(ep.Summary, "[REDACTED]") {
		t.Errorf("expected the stored Summary to contain the redaction marker; got: %q", ep.Summary)
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
				{Name: "book-system", Type: "service", Aliases: []string{"bs"}},
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

	factsRes, err := d.handleMemoryFacts(ctx, mustJSON(t, MemoryFactsParams{Slug: "bs"}))
	if err != nil {
		t.Fatalf("handleMemoryFacts(alias %q): %v", "bs", err)
	}
	facts := factsRes.([]memstore.Fact)
	if len(facts) != 1 {
		t.Fatalf("facts via alias = %d, want 1", len(facts))
	}

	episodesRes, err := d.handleMemoryEpisodes(ctx, mustJSON(t, MemoryEpisodesParams{Entity: "bs"}))
	if err != nil {
		t.Fatalf("handleMemoryEpisodes(alias %q): %v", "bs", err)
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
