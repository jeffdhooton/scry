package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/recall"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// defaultGlossaryLimit, defaultRecallLimit, and defaultEpisodesLimit mirror
// the CLI/MCP-layer defaults documented in the memory-domain SDD, applied
// here too so any RPC caller gets a sane result even without specifying a
// limit.
const (
	defaultGlossaryLimit = 200
	defaultRecallLimit   = 5
	defaultEpisodesLimit = 10
)

func (d *Daemon) registerMemoryMethods() {
	d.server.Register("memory.commit", d.handleMemoryCommit)
	d.server.Register("memory.glossary", d.handleMemoryGlossary)
	d.server.Register("memory.recall", d.handleMemoryRecall)
	d.server.Register("memory.path", d.handleMemoryPath)
	d.server.Register("memory.episodes", d.handleMemoryEpisodes)
	d.server.Register("memory.entities", d.handleMemoryEntities)
	d.server.Register("memory.facts", d.handleMemoryFacts)
	d.server.Register("memory.invalidate", d.handleMemoryInvalidate)
	d.server.Register("memory.orient", d.handleMemoryOrient)
	d.server.Register("memory.remember", d.handleMemoryRemember)
	d.server.Register("memory.cursor.get", d.handleMemoryCursorGet)
	d.server.Register("memory.cursor.put", d.handleMemoryCursorPut)
	d.server.Register("memory.hasEpisodes", d.handleMemoryHasEpisodes)
	d.server.Register("memory.status", d.handleMemoryStatus)
}

// closeMemory closes the global memory store, if it was ever opened. Called
// from the daemon's shutdown path alongside closeHTTP and the per-repo
// registries.
func (d *Daemon) closeMemory() {
	if d.memStore != nil {
		_ = d.memStore.Close()
	}
}

// --- memory.commit ---

// MemoryCommitParams carries a pre-extracted episode result (from the CLI's
// ingest pipeline, or memory.remember's own extraction) to be resolved into
// the store.
type MemoryCommitParams struct {
	Episode memstore.Episode `json:"episode"`
	Cwd     string           `json:"cwd,omitempty"`
	Result  extract.Result   `json:"result"`
}

func (d *Daemon) handleMemoryCommit(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryCommitParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Episode.ID == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "episode.id is required"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	stats, err := resolve.Apply(st, p.Episode, p.Cwd, p.Result, resolve.DefaultExclusive)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// --- memory.glossary ---

// MemoryGlossaryParams requests up to Limit "slug: alias1, alias2" lines,
// ranked by current-fact degree (most-connected entities first).
type MemoryGlossaryParams struct {
	Limit int `json:"limit,omitempty"`
}

func (d *Daemon) handleMemoryGlossary(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryGlossaryParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	limit := p.Limit
	if limit <= 0 {
		limit = defaultGlossaryLimit
	}

	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
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

// --- memory.recall ---

// MemoryRecallParams is a fuzzy entity search, optionally as-of a point in
// time. AsOf is RFC3339; empty means "current".
type MemoryRecallParams struct {
	Query string `json:"query"`
	AsOf  string `json:"as_of,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func (d *Daemon) handleMemoryRecall(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryRecallParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Query == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "query is required"}
	}
	var asOf *time.Time
	if p.AsOf != "" {
		t, err := time.Parse(time.RFC3339, p.AsOf)
		if err != nil {
			return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "as_of: " + err.Error()}
		}
		asOf = &t
	}
	limit := p.Limit
	if limit <= 0 {
		limit = defaultRecallLimit
	}

	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	hits, err := recall.Query(st, p.Query, asOf, limit)
	if err != nil {
		return nil, err
	}
	if hits == nil {
		hits = []recall.EntityHit{}
	}
	return hits, nil
}

// --- memory.path ---

// MemoryPathParams asks for the shortest chain of current facts connecting
// two entities.
type MemoryPathParams struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (d *Daemon) handleMemoryPath(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryPathParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.From == "" || p.To == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "from and to are required"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	facts, err := recall.Path(st, p.From, p.To)
	if err != nil {
		// recall.Path returns store.ErrNotFound for unknown endpoints or no
		// path; surface it as a normal RPC error, same as git methods do.
		return nil, err
	}
	if facts == nil {
		facts = []memstore.Fact{}
	}
	return facts, nil
}

// resolveMemorySlug resolves a user-supplied entity name or slug to its
// canonical slug: alias index first, falling back to Slugify when there is
// no alias match — mirroring recall.Path's resolveEndpoint semantics
// exactly, so any RPC accepting an entity identifier honors the same
// name-or-slug contract the MCP tool descriptions (scry_episodes,
// scry_recall's underlying facts lookups) promise callers, not just an
// exact slug.
func resolveMemorySlug(st *memstore.Store, name string) (string, error) {
	slug, ok, err := st.ResolveAlias(name)
	if err != nil {
		return "", err
	}
	if ok {
		return slug, nil
	}
	return memstore.Slugify(name), nil
}

// --- memory.episodes ---

// MemoryEpisodesParams asks for the most-recent episodes referenced by an
// entity's current facts.
type MemoryEpisodesParams struct {
	Entity string `json:"entity"`
	Limit  int    `json:"limit,omitempty"`
}

func (d *Daemon) handleMemoryEpisodes(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryEpisodesParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Entity == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "entity is required"}
	}
	limit := p.Limit
	if limit <= 0 {
		limit = defaultEpisodesLimit
	}

	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	slug, err := resolveMemorySlug(st, p.Entity)
	if err != nil {
		return nil, err
	}
	if _, err := st.GetEntity(slug); err != nil {
		return nil, err
	}
	eps, err := recall.Episodes(st, slug, limit)
	if err != nil {
		return nil, err
	}
	if eps == nil {
		eps = []memstore.Episode{}
	}
	return eps, nil
}

// --- memory.entities ---

// MemoryEntitiesParams optionally filters entities by Type.
type MemoryEntitiesParams struct {
	Type string `json:"type,omitempty"`
}

func (d *Daemon) handleMemoryEntities(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryEntitiesParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	entities, err := st.Entities()
	if err != nil {
		return nil, err
	}
	if p.Type != "" {
		filtered := make([]memstore.Entity, 0, len(entities))
		for _, e := range entities {
			if e.Type == p.Type {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}
	if entities == nil {
		entities = []memstore.Entity{}
	}
	return entities, nil
}

// --- memory.facts ---

// MemoryFactsParams asks for every fact about a single entity (both
// directions: Src == slug or Dst == slug).
type MemoryFactsParams struct {
	Slug           string `json:"slug"`
	IncludeInvalid bool   `json:"include_invalid,omitempty"`
}

func (d *Daemon) handleMemoryFacts(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryFactsParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Slug == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "slug is required"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	slug, err := resolveMemorySlug(st, p.Slug)
	if err != nil {
		return nil, err
	}
	if _, err := st.GetEntity(slug); err != nil {
		return nil, err
	}
	facts, err := st.FactsAbout(slug, p.IncludeInvalid)
	if err != nil {
		return nil, err
	}
	if facts == nil {
		facts = []memstore.Fact{}
	}
	return facts, nil
}

// --- memory.invalidate ---

// MemoryInvalidateParams invalidates every CURRENT fact matching the exact
// (Src, Relation, Dst) triple.
type MemoryInvalidateParams struct {
	Src      string `json:"src"`
	Relation string `json:"relation"`
	Dst      string `json:"dst"`
}

func (d *Daemon) handleMemoryInvalidate(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryInvalidateParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Src == "" || p.Relation == "" || p.Dst == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "src, relation, and dst are required"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	facts, err := st.FactsFrom(p.Src, false)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	invalidated := 0
	for _, f := range facts {
		if f.Relation != p.Relation || f.Dst != p.Dst {
			continue
		}
		if err := st.InvalidateFact(f.Src, f.Relation, f.Dst, f.ValidFrom, now); err != nil {
			return nil, err
		}
		invalidated++
	}
	return map[string]int{"invalidated": invalidated}, nil
}

// --- memory.orient ---

// MemoryOrientParams asks for a short markdown orientation blurb for a
// working directory.
type MemoryOrientParams struct {
	Cwd    string `json:"cwd,omitempty"`
	Budget int    `json:"budget,omitempty"`
}

func (d *Daemon) handleMemoryOrient(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryOrientParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	md, err := recall.Orient(st, p.Cwd, p.Budget, time.Now())
	if err != nil {
		return nil, err
	}
	return map[string]string{"markdown": md}, nil
}

// --- memory.remember ---

// MemoryRememberParams manually seeds a fact into memory. Entities is an
// optional glossary hint list (canonical name : aliases lines) to bias
// entity resolution during extraction.
type MemoryRememberParams struct {
	Fact     string   `json:"fact"`
	Entities []string `json:"entities,omitempty"`
}

// MemoryRememberResult is memory.remember's result. Dormant is true when no
// extractor is configured: in that case the episode is still stored (so
// it's ingestable later) but never resolved into facts, and Stats is
// necessarily its zero value — indistinguishable, if Dormant weren't
// reported, from a call that genuinely extracted zero entities/facts from a
// fact with no durable content. The MCP dispatch (callMemoryQuery) forwards
// this raw over JSON without unmarshaling it into a typed struct, so it
// needs no changes; there is no `scry memory remember` CLI verb to update
// either.
type MemoryRememberResult struct {
	Stats   resolve.Stats `json:"stats"`
	Dormant bool          `json:"dormant"`
}

func (d *Daemon) handleMemoryRemember(ctx context.Context, raw json.RawMessage) (any, error) {
	var p MemoryRememberParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if strings.TrimSpace(p.Fact) == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "fact is required"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}

	// p.Fact is caller-supplied free text handed straight to the extraction
	// API (and stored verbatim as the episode summary); redact it first, same
	// as every other text that reaches distill/extract, so a fact pasted with
	// a live credential in it doesn't leave this process un-redacted.
	redactedFact := distill.Redact(p.Fact)

	now := time.Now()
	sum := sha256.Sum256([]byte(p.Fact + now.Format(time.RFC3339)))
	ep := memstore.Episode{
		ID:         hex.EncodeToString(sum[:]),
		Source:     "manual",
		SourceRef:  "manual",
		Summary:    redactedFact,
		OccurredAt: now,
		IngestedAt: now,
	}

	if d.memExtractor == nil {
		// Dormant: still store the episode so it's ingestable later, but do
		// not resolve it into facts.
		if err := st.PutEpisode(ep); err != nil {
			return nil, err
		}
		return &MemoryRememberResult{Dormant: true}, nil
	}

	rawEp := distill.RawEpisode{
		ID:         ep.ID,
		Source:     ep.Source,
		SourceRef:  ep.SourceRef,
		Text:       redactedFact,
		OccurredAt: now,
	}
	result, err := d.memExtractor.Extract(ctx, rawEp, p.Entities)
	if err != nil {
		return nil, err
	}
	stats, err := resolve.Apply(st, ep, "", result, resolve.DefaultExclusive)
	if err != nil {
		return nil, err
	}
	return &MemoryRememberResult{Stats: stats}, nil
}

// --- memory.cursor.get / memory.cursor.put ---

// MemoryCursorGetParams looks up ingestion progress for a source path.
type MemoryCursorGetParams struct {
	Path string `json:"path"`
}

// MemoryCursorGetResult wraps the cursor with a Found flag so "no cursor
// yet" is distinguishable from a zero-value cursor.
type MemoryCursorGetResult struct {
	Cursor memstore.Cursor `json:"cursor"`
	Found  bool            `json:"found"`
}

func (d *Daemon) handleMemoryCursorGet(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryCursorGetParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Path == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "path is required"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	c, found, err := st.GetCursor(p.Path)
	if err != nil {
		return nil, err
	}
	return &MemoryCursorGetResult{Cursor: c, Found: found}, nil
}

func (d *Daemon) handleMemoryCursorPut(_ context.Context, raw json.RawMessage) (any, error) {
	var c memstore.Cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if c.Path == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "path is required"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	if err := st.PutCursor(c); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// --- memory.hasEpisodes ---

// MemoryHasEpisodesParams asks which of a candidate set of episode IDs are
// already committed to the store.
type MemoryHasEpisodesParams struct {
	IDs []string `json:"ids"`
}

// MemoryHasEpisodesResult reports the subset of the requested IDs that are
// NOT yet in the store — i.e. still Missing, and so still worth paying to
// extract. Backfill uses this to skip re-extracting (and re-paying for)
// episodes a previous ingest/sweep/backfill run already committed;
// resolve.Apply's own HasEpisode idempotency check would no-op them anyway,
// so without this filter the only thing extracting them again buys is a
// wasted API call.
type MemoryHasEpisodesResult struct {
	Missing []string `json:"missing"`
}

func (d *Daemon) handleMemoryHasEpisodes(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryHasEpisodesParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	missing := make([]string, 0, len(p.IDs))
	for _, id := range p.IDs {
		has, err := st.HasEpisode(id)
		if err != nil {
			return nil, err
		}
		if !has {
			missing = append(missing, id)
		}
	}
	return &MemoryHasEpisodesResult{Missing: missing}, nil
}

// --- memory.status ---

// MemoryStatusResult is the daemon's view of the memory domain: counts plus
// whether an extractor is configured.
type MemoryStatusResult struct {
	Episodes int  `json:"episodes"`
	Entities int  `json:"entities"`
	Facts    int  `json:"facts"`
	Dormant  bool `json:"dormant"`
	Cursors  int  `json:"cursors"`
}

func (d *Daemon) handleMemoryStatus(_ context.Context, _ json.RawMessage) (any, error) {
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	episodes, entities, facts, err := st.Counts()
	if err != nil {
		return nil, err
	}
	cursors, err := st.Cursors()
	if err != nil {
		return nil, err
	}
	return &MemoryStatusResult{
		Episodes: episodes,
		Entities: entities,
		Facts:    facts,
		Dormant:  d.memExtractor == nil,
		Cursors:  len(cursors),
	}, nil
}
