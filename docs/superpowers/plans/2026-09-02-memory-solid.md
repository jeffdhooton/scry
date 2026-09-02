# Memory Solid Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make scry's memory graph something every agent on the machine writes to reliably and recalls from accurately: durable async writes, a sweep that never silently stops, a closed relation vocabulary, real identities, and fact-level ranked recall.

**Architecture:** Extraction moves out of the CLI and into the daemon that owns the store. Clients (sweep, `scry_remember`) only distill and enqueue raw episodes; a daemon worker pool extracts and resolves them with retry, so the mini's `config.yaml` is the one place the model chain lives. The resolver gains Go rules for a closed vocabulary, attribute (value) facts, and evidence-gated alias merges, plus a backed-up migration that applies those rules to the existing store. Recall gains an in-memory BM25 index over fact text and entity names, ranks facts (not entities), and caps the payload.

**Tech Stack:** Go 1.23+, BadgerDB v4, anthropic-sdk-go (Messages wire format to Z.ai / DeepSeek), cobra, launchd. No CGO. No new module dependencies: BM25 is hand-rolled, OpenCode's SQLite is read through the `sqlite3` CLI shipped with macOS.

**Spec:** `docs/MEMORY_AUDIT_2026-09-02.md` (findings and numbers) and `~/dotfiles/ai/prompts/2026-09-02-scry-memory-solid.md` (goal, house rules, done bar).

## Global Constraints

- Go, no CGO, no telemetry, single static binary, JSON output by default, local only.
- Transcripts never reach a new third party: extraction stays on the `memory.models` chain in `~/.scry/config.yaml` on the mini (GLM-5.3-Flash on Z.ai, then DeepSeek). Retrieval is lexical.
- Facts are invalidated, never deleted as history. A migration may **relocate** a fact (rewrite its key for the same logical fact, as `mergeFact` already does for `ValidFrom`) but never drops its text or provenance. Every migration takes a Badger backup into `~/.scry/backups` first.
- `store.SchemaVersion` stays at 1. A bump wipes the store. All new key prefixes are additive.
- Structure lives in resolver code with table tests, not prompt wording.
- Every architectural call goes into `docs/DECISIONS.md`. Measurements get appended to `docs/MEMORY_AUDIT_2026-09-02.md`, never overwritten.
- Small PRs against `main`, each green on `go test ./...`.
- Any change to `internal/memory/store` key layout is called out in the PR description (integrator rule).

---

## File map

| Path | Responsibility |
|---|---|
| `internal/memory/store/store.go` | Existing key layout + new `pq:` (pending queue), `meta:` timestamps, `att:` alias attestations; attribute-fact key slot; fact observer hook |
| `internal/memory/queue/queue.go` | Daemon-side worker pool: pull pending episodes, extract, `resolve.Apply`, retry/backoff/park |
| `internal/memory/extract/chain.go` | Per-step cooldown on billing/auth refusals |
| `internal/daemon/memory_methods.go` | `memory.enqueue`, async `memory.remember`, `memory.queue`, `memory.queue.retry`, `memory.sweepReport`, `memory.backup`, `memory.migrate`, richer `memory.status` |
| `internal/daemon/daemon.go` | Start/stop the queue worker; glossary cache |
| `internal/memory/ingest/ingest.go` | Enqueue path replaces client-side extraction for `claude`, `codex`, `loom`, `seed`, `kimi`, `opencode` |
| `internal/memory/sweep/sweep.go` | Kimi + OpenCode roots, per-file deadlines, sweep report |
| `internal/memory/distill/kimi.go` | `wire.jsonl` → episodes |
| `internal/memory/distill/opencode.go` | `opencode.db` sessions → episodes via `sqlite3 -json -readonly` |
| `internal/doctor/memory.go` | `memory.ingest_age`, `memory.queue`, `memory.extraction` checks against the shared memory daemon |
| `internal/memory/resolve/vocab.go` | Closed relation vocabulary + synonym/flip mapping |
| `internal/memory/resolve/values.go` | Value-name detection (numbers, measurements, branches, statuses) |
| `internal/memory/resolve/resolve.go` | Attribute facts, alias admission gates, type compatibility |
| `internal/memory/resolve/hygiene.go` | Hygiene v2: generic/value alias purge, cross-type alias split, mention-based reattachment, self-loop invalidation |
| `internal/memory/migrate/migrate.go` | Backup, relation migration, value-entity migration, report |
| `internal/memory/search/index.go` | In-memory BM25 over facts and entities |
| `internal/memory/recall/recall.go` | `Query` v2: fact-ranked hybrid retrieval with a payload cap |
| `internal/memory/recall/bench.go` | Benchmark runner over a questions file |
| `internal/mcp/memory_tools.go` | Tool descriptions for the new recall shape and async remember |
| `cmd/scry/memory.go` | `queue`, `backup`, `restore`, `migrate`, `bench` verbs; sweep flags |
| `docs/MEMORY_OPS.md` | Topology, plists, deploy path, rollback, for both machines |
| `~/dotfiles/launchd/*.plist` | Tunnel FD limit, sweep log path, mini sweep agent |

---

## Lane A: Durability (done-bar items 1, 2, 6)

### Task A1: Pending queue and meta timestamps in the store

**Files:** Modify `internal/memory/store/store.go`; test `internal/memory/store/store_test.go`.

**Produces:**

```go
type PendingEpisode struct {
    ID, Source, SourceRef, Text, Cwd string
    OccurredAt, EnqueuedAt, NextAttempt time.Time
    Hints []string       // caller glossary hints (remember)
    Attempts int
    LastError string
    Parked bool          // gave up after repeated parse failures; replayable
}
func (s *Store) PutPending(p PendingEpisode) error
func (s *Store) GetPending(id string) (PendingEpisode, error)     // ErrNotFound
func (s *Store) DeletePending(id string) error
func (s *Store) Pending(limit int) ([]PendingEpisode, error)      // oldest EnqueuedAt first, all states
func (s *Store) PendingCounts() (ready, backoff, parked int, err error)
func (s *Store) PutMetaTime(key string, t time.Time) error
func (s *Store) GetMetaTime(key string) (time.Time, bool, error)
const MetaLastIngest = "last_ingest_at"   // last episode enqueued from a transcript
const MetaLastSweep = "last_sweep_at"     // last sweep that completed
const MetaLastExtract = "last_extract_ok_at" // last successful extraction
```

Keys: `pq:<id>` → JSON. `meta:<key>` → RFC3339 string.

**Tests:** put/get/delete round trip; `Pending` ordering by `EnqueuedAt`; counts split by `Parked` and `NextAttempt > now`; meta get on missing key returns found=false.

### Task A2: Daemon worker pool (`internal/memory/queue`)

**Files:** Create `internal/memory/queue/queue.go`, `queue_test.go`.

**Produces:**

```go
type Options struct {
    Store      *store.Store
    Extractor  extract.Extractor      // nil → dormant: worker does not start
    Glossary   func() []string        // cached, may be nil
    Workers    int                    // default 4
    ItemTimeout time.Duration         // default 4m
    Logf       func(string, ...any)
}
type Worker struct{ ... }
func New(o Options) *Worker
func (w *Worker) Run(ctx context.Context)   // blocks until ctx done
func (w *Worker) Kick()                     // wake the poll loop after an enqueue
func Backoff(attempts int) time.Duration   // 30s, 1m, 2m, 2m... (cap 2m)
const MaxParseAttempts = 3
```

Behaviour: every 2s (or on `Kick`) claim up to `Workers` ready items (`!Parked && NextAttempt <= now`) not already in flight; per item: extract with `ctx` bounded by `ItemTimeout`, glossary = `Glossary()` ∪ `Hints`; on success `resolve.Apply` then `DeletePending`, stamp `MetaLastExtract`; on `extract.ErrParse` after all chain steps: `Attempts++`, `Parked = Attempts >= MaxParseAttempts`, `NextAttempt = now + Backoff`; on any other error: `Attempts++`, `NextAttempt = now + Backoff`, never parked. In-flight set prevents double claims. Log one line per state change with the episode id and cause.

**Tests (fake extractor, temp store):** success path deletes pending and writes episode+facts; transport failure keeps item with backoff; three parse failures park; a parked item is not claimed; `Kick` wakes the loop; provider that fails then succeeds resolves all 20 items within a bounded test clock (use short `Backoff` via an unexported var).

### Task A3: Chain cooldown on billing/auth refusals

**Files:** Modify `internal/memory/extract/chain.go`; test `chain_test.go`.

A step whose error message contains a 401/402/403 status (`insufficient balance`, `unauthorized`, `forbidden`, or the SDK's `status 40[123]`) enters a 15-minute cooldown; the chain skips it without a network call and logs once on entry. If every step is cooling, the chain returns a transport error (not `ErrParse`) so the queue retries later.

**Tests:** 402 on step 1 → step 2 called, step 1 skipped on the next call within the window; after the window step 1 is tried again; all cooling → error not wrapping `ErrParse`.

### Task A4: Async `memory.remember` and `memory.enqueue`

**Files:** Modify `internal/daemon/memory_methods.go`, `internal/daemon/daemon.go`; tests in `internal/daemon/memory_methods_test.go` (create).

**Produces (RPC):**

- `memory.enqueue` params `{episodes: [{id, source, source_ref, text, occurred_at, cwd}]}` → `{queued, known}`. Known = already an episode or already pending. Stamps `MetaLastIngest` when `queued > 0`, then `worker.Kick()`.
- `memory.remember` params unchanged → `{queued: true, episode_id, queue_depth, dormant}`. Episode ID = sha256(`manual:` + redacted fact + UTC date). Returns without touching the provider.
- `memory.queue` → `{ready, backoff, parked, items: [...]}` (items capped 50).
- `memory.queue.retry` params `{id?}` → un-parks one or all parked items.
- `memory.sweepReport` params `{files_scanned, files_ingested, episodes, errors}` → stamps `MetaLastSweep`, keeps the last report under `meta:last_sweep_report`.
- `memory.status` adds `queue_ready, queue_backoff, queue_parked, last_ingest_at, last_sweep_at, last_extract_ok_at, worker_running`.

Daemon: build the worker in `New` when the extractor is non-nil, run it from `Run` with the daemon context, glossary cached for 10 minutes (top 200 by degree, computed by a single background refresh, never on the request path).

**Tests:** remember returns in <50ms with a fake extractor that sleeps; same fact same day dedupes to one pending; enqueue dedupes against existing episodes; dormant daemon still queues.

### Task A5: Ingest and sweep enqueue instead of extracting

**Files:** Modify `internal/memory/ingest/ingest.go`, `internal/memory/sweep/sweep.go`, `cmd/scry/memory.go`; tests updated.

`ingest.Daemon` gains `Enqueue(ctx, []distill.RawEpisode) (queued, known int, error)`; `Options.Extractor` is removed from the sweep/ingest path (backfill keeps its own extractor-driven flow). `ingest.File` distills, enqueues in batches of 50, then advances the cursor. `sweep.Run` gives every candidate its own context (`PerFileTimeout`, default 2m) derived from the run context, and calls `Daemon.SweepReport` at the end. The `scry memory sweep` CLI no longer needs a key: dormancy is reported by the daemon.

**Tests:** existing sweep/ingest fakes updated: cursor advances only after a successful enqueue; a per-file timeout does not poison the next file's cursor lookup (fake daemon that sleeps past the per-file deadline on one file); report call happens once with the totals.

### Task A6: Kimi and OpenCode distillers

**Files:** Create `internal/memory/distill/kimi.go`, `kimi_test.go`, `opencode.go`, `opencode_test.go`, testdata fixtures. Modify `sweep.go` roots.

- `KimiWire(path string, offset int64) ([]RawEpisode, int64, error)`: reads `agents/<name>/wire.jsonl`; turns = `context.append_message` user texts whose `origin.kind` is not `injection` and assistant messages (`context.append_message` role assistant, or `turn.*` events carrying assistant text); cwd from the session's `state.json` (two directories up); source `kimi-session`. Same final-line rule as Claude.
- `OpenCodeSessions(dbPath string) ([]OpenCodeSession, error)` lists `{ID, Directory, Title, TimeUpdated}` via `sqlite3 -json -readonly`; `OpenCodeSession(dbPath, sessionID string) ([]RawEpisode, error)` joins `message` and `part` (text parts only, tool parts as breadcrumbs), source `opencode-session`, source ref `opencode:<db>:<session>#<first-msg>-<last-msg>`.
- Sweep roots: `KimiGlob = ~/.kimi-code/sessions/*/*/agents/*/wire.jsonl`, `OpenCodeDB = ~/.local/share/opencode/opencode.db`. OpenCode candidates are per session: cursor path `opencode:<db>:<session>`, change detection on `TimeUpdated` vs `cursor.ModTime`, active window on `TimeUpdated`.

**Tests:** fixture wire.jsonl → expected turns/episodes; injection messages skipped; OpenCode fixture DB built in-test with `sqlite3` (skip test if binary missing) → expected episodes; sweep lists both roots.

### Task A7: Doctor memory checks

**Files:** Create `internal/doctor/memory.go`, `memory_test.go`; modify `doctor.go`.

Dials the memory daemon the same way the CLI does (`SCRY_MEMORY_SOCKET` else the local socket) and calls `memory.status`:

- `memory.extraction`: FAIL when dormant.
- `memory.ingest_age`: FAIL when `last_ingest_at` is older than 6h or absent; detail shows hours.
- `memory.queue`: WARN when `parked > 0` or the oldest ready item is older than 30m.
- `memory.sweep_age`: WARN when `last_sweep_at` older than 2h.

**Tests:** table over status payloads → statuses.

### Task A8: Ops: plists, configs, mini sweep, ops doc

- Tunnel plist: `SoftResourceLimits.NumberOfFiles = 4096`. Sweep plist: log to `~/.scry/logs/memory-sweep.log`. Both copied from `~/dotfiles/launchd/`.
- Mini: `ai.jermes.scry-memory-sweep.plist` every 30 min against the local daemon, log `~/.scry/logs/memory-sweep.log`.
- Laptop `config.yaml`: memory chain removed with a comment pointing at the mini (the laptop daemon's local memory store is unused; its extraction is intentionally dormant).
- `docs/MEMORY_OPS.md` documents all of it plus deploy and rollback.

---

## Lane B: Graph quality (done-bar items 4, 5)

### Task B1: Closed relation vocabulary

**Files:** Create `internal/memory/resolve/vocab.go`, `vocab_test.go`; modify `resolve.go` (apply after `normalizeRelation`), `store.go` (`Fact.RawRelation string json:"raw_relation,omitempty"`).

```go
// Canonical is the closed set. Len <= 40.
var Canonical = []string{...}
// Map returns the canonical relation for raw and whether src/dst swap.
func Map(raw string) (rel string, flip bool)
const Fallback = "related_to"
```

Vocabulary (36): `status`, `uses`, `depends_on`, `blocked_by`, `decided`, `deployed_on`, `runs_on`, `part_of`, `owns`, `implements`, `requires`, `fixes`, `replaced_by`, `has_issue`, `tests`, `passes`, `reviews`, `approves`, `merged_into`, `contains`, `provides`, `documents`, `monitors`, `lacks`, `enforces`, `conflicts_with`, `causes`, `assigned_to`, `configures`, `calls`, `references`, `targets`, `produces`, `located_at`, `alerts`, `related_to`.

Mapping: exact synonym table (~250 observed names incl. flips: `used_by→uses+flip`, `owned_by→owns+flip`, `fixed_by→fixes+flip`, `reviewed_by→reviews+flip`, `hosts→runs_on+flip`, `blocks→blocked_by+flip`, `supersedes→replaced_by+flip`, `has_status/test_status/verdict→status`, `has_bug/has_defect/had_bug→has_issue`, `verified/validates/verifies/checks/covers→tests`, `merged_to/merged/committed→merged_into`, `includes/has/has_feature/has_field/defines/declares→contains`, `exposes/exports/serves/returns/emits/generates/creates→provides`, `records/tracks/reports→documents`, `missing/has_no/has_gap/needs→lacks`, `contradicts→conflicts_with`, `affects/throws/fails→causes`, `claims/claimed/works_on→assigned_to+flip`, `configured_with/specifies→configures`, `installed_on/exists_on/stored_at/mounts→runs_on`, `located_in/implemented_in/belongs_to→part_of`...), then suffix rules (`*_by` → try the base with flip; `has_*` → `contains`; `is_*`/`was_*` → base), then `Fallback`.

**Tests:** every entry in `Canonical` maps to itself; table of 60 observed names → expected; `len(Canonical) <= 40`; unknown → `related_to`.

### Task B2: Attribute facts and value rejection

**Files:** Create `internal/memory/resolve/values.go`, `values_test.go`; modify `store.go`, `resolve.go`, `recall.go` (Path guard), `browse/template.html` (skip empty dst edges), `hygiene.go`.

Store: `Fact.Value string json:"value,omitempty"`; key dst slot = `Dst` or `~` + `Slugify(Value)` when `Dst == ""`; `adj:` written only for real dsts; `Fact.KeyDst() string`. `InvalidateFact`/`DeleteFact` take the key slot (callers pass `f.KeyDst()`).

```go
func IsValueName(name string) bool   // bare number, measurement, version-only, git branch, status word, hex id
func IsStatusWord(name string) bool
```

Resolver: relation `status` always yields an attribute fact (Value = raw dst text). Any relation whose dst `IsValueName` yields an attribute fact. A src that `IsValueName` with a real dst is flipped into an attribute on the dst (`RawRelation` kept). Both value → fact dropped, counted in `Stats.FactsRejected`. Phase A match for attributes compares `Value`; Rule 6 exclusivity applies per `(src, relation)` across attribute values.

**Tests:** table of names (`51b-active-parameters`, `46 GiB`, `v2.1.0`, `main`, `feat/x`, `in-progress`, `0030-price-books` (not a value), `PR #87` (not a value)); resolver writes `scry status=in-progress` with no `in-progress` entity; second status invalidates the first; path traversal ignores attributes.

### Task B3: Alias admission gates

**Files:** Modify `resolve.go`, `store.go` (`att:` keys), tests.

Store: `AttestAlias(slug, norm, episodeID) (count int, error)` under `att:<slug>:<norm>` → JSON list of episode IDs (capped 8). `AliasOwner(norm) (slug string, ok bool, err error)` (existing `ResolveAlias`).

```go
func TypesCompatible(a, b string) bool  // "concept" is a wildcard, otherwise equal
func AdmitAlias(st, entity, alias, episodeID) (admit bool, reason string, err error)
```

Rules, in order: reject ephemeral/generic/value/`the …`/`this …`/pronoun forms; reject if `AliasOwner` is another entity of an incompatible type; if owned by another entity (compatible) or equal to another entity's name, admit only when attestations from distinct episodes ≥ 2; if unowned and it shares a ≥3-char token with the entity's name or an existing alias, admit now; otherwise admit at 2 attestations. Name resolution: an `ent.Name` that resolves through an alias to an entity of an incompatible type does not merge; it gets its own slug. When a typed entity merges onto a `concept` stub the type upgrades.

**Tests:** table per rule; the hermes-ops/mac-mini scenario (project alias "mini" claimed, machine "Mac mini" arrives → separate entity); two-episode attestation flips admission.

### Task B4: Backup, restore, migrate, hygiene v2

**Files:** Create `internal/memory/migrate/migrate.go`, `migrate_test.go`; modify `hygiene.go`, `memory_methods.go`, `cmd/scry/memory.go`.

- `memory.backup {path?}` → Badger `Backup` stream to `~/.scry/backups/memory-<utc>.badger`; returns path and bytes. `scry memory restore --from <file>` runs offline on `~/.scry/memory` (daemon stopped): `DropAll` then `Load`.
- `migrate.Run(st, Options{DryRun, Backup func() (string, error)}) (Report, error)` in order: backup → relations (`vocab.Map`, relocate keys, set `RawRelation`) → values (convert facts touching value entities into attributes; delete emptied value entities and their aliases) → hygiene v2 → self-loop invalidation → report counts per step and per mapping.
- Hygiene v2 adds: generic/value alias purge with the new lists; cross-type alias split (an alias listed on entities of incompatible types is kept only on the entity whose name shares a token with it, else on the highest-degree one; the `al:` key follows); mention-based reattachment (for entity E carrying alias A that is entity B's own name: current facts on E whose text mentions A as a whole word and not E's name/slug are relocated to B); `CrossTypeCollisions int` in the report, which must be 0 after apply.
- `scry memory migrate [--apply]` (default dry run, prints the report).

**Tests:** relation relocation keeps ValidFrom/episodes and sets RawRelation; value migration converts `x status in-progress` and removes the entity; cross-type split scenario; reattachment moves only mention-matching facts; report zero collisions after apply on the fixture.

---

## Lane C: Retrieval (done-bar item 3)

### Task C1: BM25 index

**Files:** Create `internal/memory/search/index.go`, `index_test.go`; modify `store.go` (observer).

```go
type Doc struct { Kind string /*fact|entity*/; Key string; Text string; Slugs []string; ValidFrom time.Time; InvalidAt *time.Time }
type Index struct{...}
func Build(st *store.Store) (*Index, error)
func (ix *Index) Upsert(d Doc); func (ix *Index) Remove(key string)
func (ix *Index) Search(q string, k int, asOf *time.Time) []Hit  // Hit{Key, Score}
func Tokenize(s string) []string   // lowercase, split on non-alnum, keep 2+ chars, split camelCase and snake_case, light stemming (s, ing, ed)
```

Fact doc text = fact sentence + src name + dst name or value + relation words. Entity doc = name + aliases + description. `store.SetObserver(func(ev Event))` fires on `PutFact`, `InvalidateFact`, `DeleteFact`, `PutEntity`; the daemon wires it to the index. BM25 with k1=1.2, b=0.75; exact-phrase bonus.

**Tests:** ranking sanity on a small corpus; camelCase/snake tokenization; as-of filtering; observer keeps the index current.

### Task C2: Recall v2 and payload cap

**Files:** Modify `recall.go`, `recall_test.go`, `memory_methods.go`, `mcp/memory_tools.go`, `cmd/scry/memory.go`.

```go
type Result struct {
    Query    string        `json:"query"`
    Entities []EntityHead  `json:"entities"`  // ≤5: slug, name, type, description, fact_count, score
    Facts    []FactHit     `json:"facts"`     // ≤limit (default 20): fact fields + score + entity names
    Episodes []EpisodeHead `json:"episodes"`  // ≤3: id, source, occurred_at, summary (≤300 chars)
    Truncated bool         `json:"truncated,omitempty"`
}
func Query(st, ix, q string, asOf *time.Time, limit int) (Result, error)
const MaxPayloadBytes = 24 * 1024
```

Scoring: fact BM25 score + 0.5 × entity-match bonus when the fact touches an entity whose name/alias exactly or token-matches the query + small recency term (facts seen in the last 30 days). Entities = top by aggregated fact scores plus direct name matches. Serialize; while over `MaxPayloadBytes`, drop episodes, then trailing facts, set `Truncated`.

MCP `scry_recall`: `limit` = max facts (default 20); description rewritten. `scry_remember` description says it returns immediately and resolves in the background.

**Tests:** seeded store where the answering fact is on a high-degree entity still ranks top-5 for a specific question; payload never exceeds 24KB with 3,000 facts on one entity; as-of works.

### Task C3: Benchmark

**Files:** Create `internal/memory/recall/bench.go`, `cmd/scry/memory.go` (`bench` verb), `docs/memory-bench/tuning.json` (50 questions, builder's set).

Question file: `[{"question": "...", "expect": {"src": "...", "relation": "...", "dst": "..."} | {"fact_substring": "..."} | {"episode": "<id>"}}]`. `scry memory bench --file f.json [--top 20]` prints hits/total, mean payload bytes, max payload, misses with the top-3 returned facts.

---

## Deploy, measure, grade

1. Laptop: `go build -trimpath -ldflags "-X main.version=scry-<sha>" -o ~/go/bin/scry.new && cp ~/go/bin/scry ~/go/bin/scry.pre-<sha> && mv ~/go/bin/scry.new ~/go/bin/scry && launchctl kickstart -k gui/$(id -u)/com.jhoot.scryd`.
2. Mini: `git pull`, same build into `~/.local/bin/scry` with a `.pre-<sha>` copy, `launchctl kickstart -k gui/$(id -u)/ai.jermes.scryd`; install the sweep plist.
3. `scry memory backup` on the mini, then `scry memory migrate` dry run, review, `--apply`, re-run the seven probes, append numbers to the audit doc.
4. Graders: six fresh sub-agents, one per done-bar item, each told to disprove.
