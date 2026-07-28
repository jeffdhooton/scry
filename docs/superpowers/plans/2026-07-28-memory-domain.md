# Scry Memory Domain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add scry's fifth domain — a global episodic temporal knowledge graph extracted from Claude Code / Codex transcripts and loom runs, with hook+sweep ingestion and thin-push/pull recall. Spec: `docs/MEMORY_SPEC.md`.

**Architecture:** New `internal/memory/{store,distill,extract,resolve,recall,sweep}` packages. Storage is a single global BadgerDB at `~/.scry/memory/index.db`, opened and owned by the daemon. All LLM work (distill → extract) runs in the **CLI process**; the daemon does storage, resolution, and queries via fast RPCs (`memory.*`), mirroring the existing `git.*`/`graph.*` pattern. MCP tools forward to daemon RPCs exactly like `internal/mcp/git_tools.go`. The dotfiles side (hooks, launchd, CLAUDE.md/AGENTS.md) is Tasks 12–13.

**Tech Stack:** Go 1.26, cobra, BadgerDB v4 (existing deps), `github.com/anthropics/anthropic-sdk-go` (new dep) for Haiku extraction + Batch API.

## Global Constraints

- Module path: `github.com/jeffdhooton/scry`. Repo root: `~/workspace/context-stack/scry`. **Another session may be working on this repo — do all work on a branch `memory-domain` (or a worktree) off `main`, never commit directly to `main`.**
- Follow existing domain patterns exactly: daemon RPCs registered via `d.server.Register("memory.<verb>", d.handleMemoryX)` in a new `internal/daemon/memory_methods.go` (mirror `git_methods.go`); CLI commands use `callDaemon(ctx, "memory.<verb>", &daemon.XParams{...}, &result)` + `printJSON` (mirror `cmd/scry/git.go`); MCP tools are a `memoryToolDefinitions []tool` var + `case` entries in `internal/mcp/server.go` (mirror `git_tools.go`).
- All BadgerDB values are JSON. Memory store schema version starts at 1; on mismatch, wipe and rebuild (same policy as `internal/store`).
- Extraction model: env `SCRY_MEMORY_MODEL`, default `"claude-haiku-4-5"`. API key: `SCRY_MEMORY_API_KEY`, falling back to `ANTHROPIC_API_KEY`. **No key → dormant:** CLI ingest/sweep/backfill print `memory: dormant (no ANTHROPIC_API_KEY / SCRY_MEMORY_API_KEY)` and exit 0; recall/orient/entities/facts/status still work (they don't need the key).
- Prompt-caching caveat: Haiku 4.5's minimum cacheable prefix is **4096 tokens**; our system prompt + glossary will often be below that, so set `cache_control` anyway (harmless, and it kicks in when the glossary grows) but do NOT count on cache reads for cost. The Batch API's 50% discount is the real backfill cost lever.
- Never send raw transcript text to the API without running `distill.Redact` first.
- Memory store home: `filepath.Join(scryHome, "memory")` where `scryHome` is the daemon's existing `d.scryHome()` (`~/.scry`).
- Tests: `go test ./internal/memory/... ./internal/daemon/...`. Every task ends with a passing test run and a commit on the branch. Commit messages: `feat(memory): <what>` and end with `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- Timestamps are `time.Time` marshaled as RFC3339 in JSON. IDs/slugs are lowercase kebab-case.

---

### Task 0: Branch + dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1:** `cd ~/workspace/context-stack/scry && git checkout -b memory-domain`
- [ ] **Step 2:** `go get github.com/anthropics/anthropic-sdk-go@latest && go mod tidy && go build ./...`
  Expected: builds clean.
- [ ] **Step 3:** Commit: `chore(memory): add anthropic-sdk-go dependency`

---

### Task 1: Memory store (BadgerDB)

**Files:**
- Create: `internal/memory/store/store.go`
- Test: `internal/memory/store/store_test.go`

**Interfaces:**
- Consumes: `github.com/dgraph-io/badger/v4` (existing dep; copy open-options idiom from `internal/graph/store/store.go`).
- Produces (used by every later task):

```go
package store

const SchemaVersion = 1

var ErrNotFound = errors.New("memory: not found")

type Episode struct {
    ID         string    `json:"id"`          // sha256 hex of source_ref
    Source     string    `json:"source"`      // claude-session | codex-session | loom-run | seed | manual
    SourceRef  string    `json:"source_ref"`  // path + byte/line span, e.g. "/path/file.jsonl#L120-L340"
    Summary    string    `json:"summary"`
    OccurredAt time.Time `json:"occurred_at"`
    IngestedAt time.Time `json:"ingested_at"`
}

type Entity struct {
    Slug        string    `json:"slug"`
    Name        string    `json:"name"`
    Type        string    `json:"type"` // project|service|machine|tool|person|decision|runbook|concept
    Description string    `json:"description"`
    Aliases     []string  `json:"aliases,omitempty"`
    RepoRefs    []string  `json:"repo_refs,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    LastSeen    time.Time `json:"last_seen"`
}

type Fact struct {
    Src        string     `json:"src"`
    Relation   string     `json:"relation"`
    Dst        string     `json:"dst"`
    Fact       string     `json:"fact"` // one-sentence natural language
    ValidFrom  time.Time  `json:"valid_from"`
    InvalidAt  *time.Time `json:"invalid_at,omitempty"` // nil = current
    Confidence float64    `json:"confidence"`
    Episodes   []string   `json:"episodes"` // provenance episode IDs
}

type Cursor struct {
    Path           string    `json:"path"`
    Size           int64     `json:"size"`
    ModTime        time.Time `json:"mod_time"`
    ProcessedBytes int64     `json:"processed_bytes"`
}

func Open(dir string) (*Store, error) // creates dir; wipes+rebuilds on schema mismatch
func (s *Store) Close() error

func (s *Store) PutEpisode(e Episode) error
func (s *Store) GetEpisode(id string) (Episode, error)
func (s *Store) HasEpisode(id string) (bool, error)

func (s *Store) PutEntity(e Entity) error            // also indexes every alias + the name
func (s *Store) GetEntity(slug string) (Entity, error)
func (s *Store) ResolveAlias(name string) (string, bool, error) // Normalize(name) lookup → slug
func (s *Store) Entities() ([]Entity, error)                    // full scan, sorted by slug
func (s *Store) EntitiesByRepoRef(repoPath string) ([]Entity, error)

func (s *Store) PutFact(f Fact) error
func (s *Store) FactsFrom(slug string, includeInvalid bool) ([]Fact, error) // src == slug
func (s *Store) FactsAbout(slug string, includeInvalid bool) ([]Fact, error) // src OR dst == slug
func (s *Store) InvalidateFact(src, relation, dst string, validFrom, at time.Time) error

func (s *Store) GetCursor(path string) (Cursor, bool, error)
func (s *Store) PutCursor(c Cursor) error
func (s *Store) Cursors() ([]Cursor, error)

func (s *Store) Counts() (episodes, entities, facts int, err error)

func Normalize(name string) string // lowercase, trim, collapse spaces/underscores → "-"
func Slugify(name string) string   // Normalize + strip non [a-z0-9-]
```

Key layout (document in the package comment, same style as `internal/graph/store`):

```
meta:schema_version                                → int
ep:<id>                                            → Episode
en:<slug>                                          → Entity
al:<normalized-name>                               → slug (raw string)
fa:<src>:<relation>:<dst>:<validfrom-unixnano>     → Fact
adj:<dst>:<src>:<relation>:<validfrom-unixnano>    → empty (reverse index for FactsAbout)
cur:<sha256(path)>                                 → Cursor
```

- [ ] **Step 1: Write failing tests** covering: open/close + schema wipe; episode put/has/get; entity put + alias resolution (`ResolveAlias("Hermes Mini")` → `"hermes-mini"` after `PutEntity(Entity{Slug:"hermes-mini", Name:"Hermes mini", Aliases:[]string{"the mini"}})`); fact put + `FactsFrom` filters invalidated when `includeInvalid=false`; `InvalidateFact` sets `InvalidAt`; two facts same `(src,rel,dst)` different `ValidFrom` coexist; cursor round-trip; `EntitiesByRepoRef` matches on exact path element; `Slugify("Hermes-Ops LeadGen!")` == `"hermes-ops-leadgen"`.

```go
func TestFactInvalidation(t *testing.T) {
    s := openTemp(t)
    v1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
    f := store.Fact{Src: "book-system", Relation: "deployed_on", Dst: "hermes-mini",
        Fact: "book-system runs on the mini", ValidFrom: v1, Confidence: 0.9, Episodes: []string{"e1"}}
    if err := s.PutFact(f); err != nil { t.Fatal(err) }
    at := v1.AddDate(0, 1, 0)
    if err := s.InvalidateFact("book-system", "deployed_on", "hermes-mini", v1, at); err != nil { t.Fatal(err) }
    cur, _ := s.FactsFrom("book-system", false)
    if len(cur) != 0 { t.Fatalf("want 0 current facts, got %d", len(cur)) }
    all, _ := s.FactsFrom("book-system", true)
    if len(all) != 1 || all[0].InvalidAt == nil { t.Fatalf("invalidated fact must survive: %+v", all) }
}
```

- [ ] **Step 2:** `go test ./internal/memory/store/` → FAIL (package missing).
- [ ] **Step 3:** Implement `store.go`. Copy the Badger open options and iterator idioms from `internal/graph/store/store.go`. `InvalidateFact` reads the exact `fa:` key, sets `InvalidAt`, writes back.
- [ ] **Step 4:** `go test ./internal/memory/store/` → PASS.
- [ ] **Step 5:** Commit: `feat(memory): global temporal store (episodes, entities, facts, cursors)`

---

### Task 2: Distillation — Claude transcripts, redaction, chunking

**Files:**
- Create: `internal/memory/distill/distill.go` (shared types, chunking, redaction)
- Create: `internal/memory/distill/claude.go`
- Test: `internal/memory/distill/claude_test.go`, `internal/memory/distill/redact_test.go`
- Create: `internal/memory/distill/testdata/claude_session.jsonl` (fixture — see Step 0)

**Interfaces:**
- Produces:

```go
package distill

type RawEpisode struct {
    ID         string    // sha256 hex of SourceRef
    Source     string    // "claude-session" etc.
    SourceRef  string    // "<path>#<startByte>-<endByte>"
    Text       string    // distilled, redacted conversational text
    OccurredAt time.Time
    Cwd        string    // session working directory, "" if unknown
}

// Reads JSONL starting at byte offset; returns episodes + the new offset consumed.
func ClaudeSession(path string, offset int64) ([]RawEpisode, int64, error)
func Redact(text string) string
const maxEpisodeChars = 16000 // ~4k tokens; chunk on turn boundaries with 1-turn overlap
```

- [ ] **Step 0: Ground the parser in a real file.** Run `head -3 $(ls -S ~/.claude/projects/*/*.jsonl | head -1) | python3 -m json.tool` (or read via a Go scratch test) and note the actual field names for: line type, message role, content blocks, timestamp, cwd. The shapes below are the expected ones — **verify and adjust before writing the fixture**:
  - lines are objects with `"type"` (`"user"`, `"assistant"`, others to skip), `"timestamp"` (RFC3339), `"cwd"`, and `"message"` with `"role"` and `"content"` (either a string, or an array of blocks with `"type": "text" | "tool_use" | "tool_result"`).
- [ ] **Step 1:** Build `testdata/claude_session.jsonl` — 8–10 hand-written lines matching the verified shape: 2 user turns, 2 assistant turns with text, 1 assistant turn with a `tool_use` block (name `"Bash"`), 1 `tool_result`-bearing user line, one line with an obviously fake secret (`sk-ant-api03-FAKEFAKE...` ≥ 30 chars), one malformed line (`{truncated`), one irrelevant type. Write failing tests:
  - `TestClaudeSessionDistills`: full-file parse yields ≥1 episode; text contains both user texts and assistant texts; contains breadcrumb `[tool: Bash]`; does NOT contain the tool_result body; does NOT contain the fake secret; `Cwd` and `OccurredAt` populated; malformed line skipped without error.
  - `TestClaudeSessionOffsetResume`: call with `offset` = byte length of first half → episodes only cover the second half; returned offset == file size.
  - `TestChunking`: a synthetic 40k-char conversation yields ≥3 episodes each ≤ `maxEpisodeChars`, split on turn boundaries, consecutive episodes share one overlapping turn, IDs are distinct and deterministic across two runs.
  - `TestRedact` (own file): strips `sk-`-prefixed keys, `Bearer <token>`, `-----BEGIN ... PRIVATE KEY-----` blocks, `ghp_...`, replacing with `[REDACTED]`; leaves normal prose and short hex strings alone.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement. Parsing is tolerant: `json.Unmarshal` each line into a loose struct; skip lines that fail or have unknown type. Keep user/assistant text blocks verbatim; replace `tool_use` blocks with `[tool: <name>]`; drop `tool_result` content entirely. Sessions with < 3 substantive turns return zero episodes (still advance offset). Redact with a small ordered list of `regexp.MustCompile` patterns.
- [ ] **Step 4:** Run → PASS.
- [ ] **Step 5:** Commit: `feat(memory): claude transcript distillation with redaction + chunking`

---

### Task 3: Distillation — Codex, loom, seed

**Files:**
- Create: `internal/memory/distill/codex.go`, `internal/memory/distill/loom.go`, `internal/memory/distill/seed.go`
- Test: `internal/memory/distill/codex_test.go`, `internal/memory/distill/loom_test.go`, `internal/memory/distill/seed_test.go`
- Create: `internal/memory/distill/testdata/codex_rollout.jsonl`, `testdata/loom_run/` (mini fixture dir), `testdata/seed_memory.md`

**Interfaces:**
- Produces (same `RawEpisode` from Task 2):

```go
func CodexRollout(path string, offset int64) ([]RawEpisode, int64, error) // Source: "codex-session"
func LoomRun(dir string) ([]RawEpisode, error)                            // Source: "loom-run"; whole-dir, no offset
func SeedMarkdown(path string) (RawEpisode, error)                        // Source: "seed"
```

- [ ] **Step 0: Ground the parsers.** Codex: `head -5 $(find ~/.codex/sessions -name '*.jsonl' | head -1) | python3 -m json.tool`. Envelope is `{"timestamp","type","payload"}` with types `session_meta` (payload carries cwd + session start), `event_msg`, `response_item` (payload carries role + content items with text) — verify exact payload field names. Loom: `ls ~/.loom/runs/sandbox-fix-add/` and read the run's state/spec files to identify: spec goal, iteration count, gate verdicts, final status (loom's memory spine files — check `~/workspace/loom/loom/memory.py` docstrings for filenames if unclear).
- [ ] **Step 1:** Failing tests, one per source, following the Task 2 pattern:
  - Codex: user + assistant text extracted, tool noise dropped, offset resume works, `Cwd` from `session_meta`.
  - Loom: one fixture run dir → exactly one episode whose text names the goal, the iteration count, each gate verdict, and the final status; `OccurredAt` from the run's newest file mtime.
  - Seed: `testdata/seed_memory.md` has YAML frontmatter (`name`, `description`) + body; episode text = description + body, `OccurredAt` = file mtime, `SourceRef` = path.
- [ ] **Step 2:** FAIL → **Step 3:** implement → **Step 4:** PASS.
- [ ] **Step 5:** Commit: `feat(memory): codex, loom, and seed distillation`

---

### Task 4: Extractor — Haiku client behind an interface

**Files:**
- Create: `internal/memory/extract/extract.go` (types + interface), `internal/memory/extract/prompt.go`, `internal/memory/extract/haiku.go`
- Test: `internal/memory/extract/extract_test.go` (fake-backed), `internal/memory/extract/live_test.go` (gated)

**Interfaces:**
- Consumes: `distill.RawEpisode`.
- Produces:

```go
package extract

type Result struct {
    EpisodeSummary string  `json:"episode_summary"`
    Entities       []Ent   `json:"entities"`
    Facts          []Fct   `json:"facts"`
}
type Ent struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"`
    Description string   `json:"description"`
    Aliases     []string `json:"aliases,omitempty"`
}
type Fct struct {
    Src        string  `json:"src"`
    Relation   string  `json:"relation"`
    Dst        string  `json:"dst"`
    Fact       string  `json:"fact"`
    ValidFrom  string  `json:"valid_from,omitempty"` // RFC3339 date or empty
    Confidence float64 `json:"confidence"`
    Supersedes *SupRef `json:"supersedes,omitempty"`
}
type SupRef struct{ Src, Relation, Dst string }

type Extractor interface {
    Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (Result, error)
}

func NewHaiku(apiKey, model string) *Haiku // model "" → "claude-haiku-4-5"
func ParseResult(raw string) (Result, error) // strips ```json fences, unmarshals, validates
```

`prompt.go` holds the stable system prompt as a `const` (this is the cache-control block — keep it byte-stable):

```go
const SystemPrompt = `Extract a knowledge graph from the episode text.
Return ONLY a JSON object, no prose, matching:
{"episode_summary": "1-2 sentences",
 "entities": [{"name","type","description","aliases":[]}],
 "facts": [{"src","relation","dst","fact","valid_from","confidence","supersedes":{"src","relation","dst"}|null}]}
Rules:
- type is one of: project, service, machine, tool, person, decision, runbook, concept.
- relation is a short snake_case verb phrase: deployed_on, uses, replaced_by, blocked_by, decided, status, owns, depends_on — or another short verb if none fits.
- src/dst reference entity names from this output or the KNOWN ENTITIES glossary; prefer glossary names when the episode clearly refers to them.
- valid_from: RFC3339 date the fact became true, if the text implies one; else omit.
- supersedes: set when the fact contradicts/replaces an earlier fact you can name.
- Never invent facts not supported by the text. Skip trivia (greetings, tool mechanics).
- confidence in [0,1].`
```

- [ ] **Step 1: Failing tests** (no network): a `fakeExtractor` implementing `Extractor` lives in the test file for later tasks to copy; here test `ParseResult`: valid JSON → Result; fenced ` ```json ... ``` ` → Result; junk → error; missing `episode_summary` → error; unknown entity type → error naming the type. Test that `buildMessages(ep, glossary)` (unexported helper returning system blocks + user text) places `SystemPrompt` first with cache control, the glossary as a second system block, and `reference_time: <RFC3339>\ncwd: <cwd>\n\n<text>` as the user message — assert on the returned SDK params' text contents.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implement `haiku.go` with the Go SDK:

```go
func (h *Haiku) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (Result, error) {
    params := anthropic.MessageNewParams{
        Model:     anthropic.Model(h.model),
        MaxTokens: 4000,
        System: []anthropic.TextBlockParam{
            {Text: SystemPrompt, CacheControl: anthropic.NewCacheControlEphemeralParam()},
            {Text: "KNOWN ENTITIES (canonical name: aliases):\n" + strings.Join(glossary, "\n")},
        },
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(
                "reference_time: " + ep.OccurredAt.Format(time.RFC3339) + "\ncwd: " + ep.Cwd + "\n\n" + ep.Text)),
        },
    }
    resp, err := h.client.Messages.New(ctx, params)
    // concat text blocks → ParseResult; on parse error, retry ONCE appending
    // the assistant reply + a user turn: "Invalid JSON (<err>). Return only the corrected JSON object."
}
```
  (Exact SDK type names per `anthropic-sdk-go`; if the compiler rejects a name, fix from the compiler error — do not redesign the call.) `NewHaiku` uses `anthropic.NewClient(option.WithAPIKey(key))`.
- [ ] **Step 4:** `go test ./internal/memory/extract/` → PASS. Add `live_test.go` gated by `SCRY_LIVE_TEST=1` + key present: extract from a 5-line hand-written episode about "deploying book-system to the hermes mini", assert ≥1 entity and ≥1 fact come back. Skip otherwise. Don't run it now.
- [ ] **Step 5:** Commit: `feat(memory): haiku extractor with stable cached prompt`

---

### Task 5: Resolver — canonicalization + temporal invalidation

**Files:**
- Create: `internal/memory/resolve/resolve.go`
- Test: `internal/memory/resolve/resolve_test.go`

**Interfaces:**
- Consumes: `store.*`, `extract.Result`.
- Produces:

```go
package resolve

var DefaultExclusive = map[string]bool{"deployed_on": true, "status": true, "replaced_by": true}

type Stats struct{ EntitiesCreated, EntitiesUpdated, FactsAdded, FactsInvalidated, FactsMerged int }

// Apply is atomic per episode from the caller's perspective and idempotent:
// if st.HasEpisode(ep.ID) it returns zero Stats and nil error without writing.
func Apply(st *store.Store, ep store.Episode, res extract.Result, exclusive map[string]bool) (Stats, error)
```

Resolution rules (implement exactly, in order):
1. Skip if episode already ingested (idempotency).
2. For each extracted entity: `slug := ResolveAlias(name)` else `Slugify(name)`. Missing → create (CreatedAt = ep.OccurredAt). Existing → merge: union aliases, overwrite Description if the new one is non-empty, LastSeen = max(LastSeen, ep.OccurredAt). If `ep.SourceRef`'s session cwd (passed via `store.Episode`-adjacent param — add `Cwd string` argument to `Apply`) is a workspace path, union it into RepoRefs.
3. For each fact: resolve src/dst names → slugs (creating stub entities of type `concept` for names never seen — description empty). ValidFrom = parsed `valid_from` else ep.OccurredAt.
4. **Merge:** if a *current* fact with same `(src, relation, dst)` exists, append ep.ID to its Episodes, keep the earlier ValidFrom, max the Confidence → `FactsMerged++`; do not add a duplicate.
5. **Supersedes hint:** resolve the ref's names; invalidate any current matching fact at ep.OccurredAt → `FactsInvalidated++`.
6. **Exclusive relations:** if `exclusive[relation]` and a current fact exists with same `(src, relation)` but *different* dst → invalidate it at ep.OccurredAt, then add the new fact.
7. Finally `PutEpisode`.

- [ ] **Step 1: Failing table-tests** using a real temp `store.Store` (no LLM): idempotent re-apply; alias-based resolution ("the mini" resolves to existing `hermes-mini`); stub entity creation; merge rule; supersedes; exclusive `deployed_on` flip invalidates old and adds new; non-exclusive `uses` facts coexist; `valid_from` parse fallback.
- [ ] **Step 2:** FAIL → **Step 3:** implement → **Step 4:** PASS.
- [ ] **Step 5:** Commit: `feat(memory): resolver with alias canonicalization and temporal invalidation`

---

### Task 6: Recall — queries, orient, path

**Files:**
- Create: `internal/memory/recall/recall.go`
- Test: `internal/memory/recall/recall_test.go`

**Interfaces:**
- Consumes: `store.*`.
- Produces:

```go
package recall

type EntityHit struct {
    Entity   store.Entity `json:"entity"`
    Facts    []store.Fact `json:"facts"`             // current unless AsOf set
    Episodes []store.Episode `json:"episodes,omitempty"` // most recent ≤5 referencing the entity
}
func Query(st *store.Store, q string, asOf *time.Time, limit int) ([]EntityHit, error)
func Path(st *store.Store, from, to string) ([]store.Fact, error) // BFS over current facts, both directions; ErrNotFound if none
func Episodes(st *store.Store, slug string, limit int) ([]store.Episode, error)
func Orient(st *store.Store, cwd string, budgetChars int) (string, error) // markdown, hard-capped
```

`Query`: tokenize q on non-alphanumerics; a hit is any entity whose slug, name, or alias contains a token (≥3 chars) or the full normalized query. Rank: exact alias match > substring > recency (LastSeen). `asOf` non-nil → include facts where `ValidFrom ≤ asOf` and (`InvalidAt` nil or `> asOf`).

`Orient` output shape (cap default 2000 chars ≈ 500 tokens; truncate whole sections, never mid-line):

```markdown
## Memory orientation
### This repo (<base of cwd>)
- **<name>** (<type>): <fact 1>; <fact 2>; <fact 3>
### Active projects (last 14d)
- <name>: <most recent current fact>
_Query scry_recall for anything referenced here._
```

- [ ] **Step 1: Failing tests** seeding a temp store directly (no LLM): query by alias; as-of time travel returns the invalidated fact; BFS path across 2 hops (`loom → uses → deepseek-v4 ← used_by ← book-system` style, using the adj index in both directions); orient includes repo-matched entity first, respects budget (feed 50 entities, assert `len ≤ budget`), includes the closing pointer line.
- [ ] **Step 2:** FAIL → **Step 3:** implement → **Step 4:** PASS.
- [ ] **Step 5:** Commit: `feat(memory): recall query, as-of, path, and orient`

---

### Task 7: Daemon RPCs

**Files:**
- Create: `internal/daemon/memory_methods.go`
- Modify: `internal/daemon/methods.go` (one line in `registerMethods()`: `d.registerMemoryMethods()`)
- Modify: `internal/daemon/daemon.go` (add `memStore *memstore.Store` + lazy `memoryStore()` accessor; close on shutdown)
- Test: `internal/daemon/memory_methods_test.go`

**Interfaces:**
- Consumes: Tasks 1, 5, 6. Import as `memstore "github.com/jeffdhooton/scry/internal/memory/store"`.
- Produces — RPC methods + param/result structs (CLI and MCP depend on these exact names):

```go
// Registered in registerMemoryMethods():
"memory.commit"     MemoryCommitParams{Episode memstore.Episode; Cwd string; Result extract.Result} → resolve.Stats
"memory.glossary"   MemoryGlossaryParams{Limit int}                → []string  // "slug: alias1, alias2" lines, top-N by fact degree, default 200
"memory.recall"     MemoryRecallParams{Query string; AsOf string; Limit int}  → []recall.EntityHit
"memory.path"       MemoryPathParams{From, To string}             → []memstore.Fact
"memory.episodes"   MemoryEpisodesParams{Entity string; Limit int} → []memstore.Episode
"memory.entities"   MemoryEntitiesParams{Type string}             → []memstore.Entity
"memory.facts"      MemoryFactsParams{Slug string; IncludeInvalid bool} → []memstore.Fact
"memory.invalidate" MemoryInvalidateParams{Src, Relation, Dst string} → map[string]int  // {"invalidated": n} — all current matches
"memory.orient"     MemoryOrientParams{Cwd string; Budget int}    → map[string]string   // {"markdown": ...}
"memory.remember"   MemoryRememberParams{Fact string; Entities []string} → resolve.Stats
"memory.cursor.get" MemoryCursorGetParams{Path string}            → memstore.Cursor (Found bool via wrapper struct)
"memory.cursor.put" memstore.Cursor                               → ok
"memory.status"     (no params)                                    → MemoryStatusResult{Episodes, Entities, Facts int; Dormant bool; Cursors int}
```

`memory.remember`: if the daemon has an extractor (constructed at startup when an API key is in the daemon's env), run the fact text as a tiny `manual` episode (`OccurredAt: now`, ID = sha256 of text+timestamp) through extract→resolve. If dormant, still store the episode (Summary = the fact text) so it's ingestable later, and return zero Stats. The daemon builds its extractor with the same env fallback as the CLI.

- [ ] **Step 1: Failing test** exercising the handlers directly (mirror how `registry_test.go` builds a daemon or call the `handleMemoryX` funcs with a temp scry home): commit an episode with a canned `extract.Result` → recall finds it; glossary lists it; status counts it; cursor round-trips; remember while dormant stores an episode without error.
- [ ] **Step 2:** FAIL → **Step 3:** implement (lazy-open the store on first memory RPC: `memstore.Open(filepath.Join(d.scryHome(), "memory"))`, guarded by `sync.Once`) → **Step 4:** `go test ./internal/daemon/` PASS.
- [ ] **Step 5:** Commit: `feat(memory): daemon memory.* RPCs`

---

### Task 8: Ingest pipeline + CLI (`scry memory ingest|orient|recall|...`)

**Files:**
- Create: `internal/memory/ingest/ingest.go` (CLI-side pipeline: distill → glossary → extract → commit)
- Create: `cmd/scry/memory.go`
- Modify: `cmd/scry/main.go` (register `memoryCmd()` in the root command, next to the other domains)
- Test: `internal/memory/ingest/ingest_test.go`

**Interfaces:**
- Consumes: Tasks 2–4, 7 (`callDaemon` equivalents live in cmd; ingest package takes small interfaces instead so it's testable):

```go
package ingest

type Daemon interface { // implemented in cmd/scry via callDaemon
    Glossary(ctx context.Context, limit int) ([]string, error)
    Commit(ctx context.Context, ep store.Episode, cwd string, res extract.Result) (resolve.Stats, error)
    GetCursor(ctx context.Context, path string) (store.Cursor, bool, error)
    PutCursor(ctx context.Context, c store.Cursor) error
}

type Options struct{ Source, Path string; Extractor extract.Extractor; Daemon Daemon }

// File ingests one transcript/run/seed path from its cursor offset (episodic sources)
// or wholesale (loom/seed). Returns per-file stats. Advances the cursor on success.
func File(ctx context.Context, o Options) (Summary, error)

type Summary struct{ EpisodesIngested, EpisodesSkipped int; Stats resolve.Stats }
```

- Produces: `scry memory` command tree.

- [ ] **Step 1: Failing test** for `ingest.File` with the fake extractor (from Task 4's tests) and a fake Daemon recording calls: Claude fixture path → glossary fetched once, one commit per episode, cursor advanced to file size; second run with same cursor → zero episodes (offset resume); loom fixture → one commit, cursor keyed on the run dir path with Size = 0 semantics (use dir mtime in Cursor.ModTime to detect change).
- [ ] **Step 2:** FAIL → **Step 3:** implement `ingest.go`, then `cmd/scry/memory.go` following `git.go` exactly:

```go
func memoryCmd() *cobra.Command {
    cmd := &cobra.Command{Use: "memory", Short: "Global episodic memory graph"}
    cmd.AddCommand(memoryIngestCmd(), memorySweepCmd(), memoryBackfillCmd(),
        memoryOrientCmd(), memoryRecallCmd(), memoryEntitiesCmd(), memoryFactsCmd(),
        memoryInvalidateCmd(), memoryStatusCmd())
    return cmd
}
```

  - `ingest --source claude|codex|loom|seed --path <p>`: dormant-check first (print notice, exit 0). Build `Haiku` extractor, a `daemonClient` struct implementing `ingest.Daemon` over `callDaemon`, call `ingest.File`, print the Summary JSON. Timeout: `context.WithTimeout(…, 10*time.Minute)`.
  - Query commands are thin `callDaemon` wrappers, one per RPC, mirroring `blameCmd`: `orient [--cwd] [--budget 2000]` prints the markdown raw (not JSON); `recall <query> [--as-of t] [--limit 5]`, `entities [--type t]`, `facts <slug> [--all]`, `invalidate <src> <relation> <dst>`, `status` print JSON via `printJSON`.
- [ ] **Step 4:** `go test ./internal/memory/... && go build ./...` PASS. Manual smoke: `go run ./cmd/scry memory status` (against the running daemon; expect zero counts, `dormant` per env).
- [ ] **Step 5:** Commit: `feat(memory): ingest pipeline + scry memory CLI`

---

### Task 9: Sweep

**Files:**
- Create: `internal/memory/sweep/sweep.go`
- Modify: `cmd/scry/memory.go` (fill in `memorySweepCmd()`)
- Test: `internal/memory/sweep/sweep_test.go`

**Interfaces:**
- Consumes: `ingest`.
- Produces:

```go
package sweep

type Roots struct { // all overridable for tests; defaults from $HOME
    ClaudeGlob string // ~/.claude/projects/*/*.jsonl  (top-level session files only, not memory/)
    CodexGlob  string // ~/.codex/sessions/*/*/*/rollout-*.jsonl
    LoomRuns   string // ~/.loom/runs  (each subdir = one run)
}
type Result struct{ FilesScanned, FilesIngested, FilesSkippedActive, FilesUnchanged int; Episodes int; Errors []string }

// Run scans all roots, compares size+mtime against cursors, and ingests deltas.
// Files with mtime within activeWindow (default 5m) of now are skipped.
// DryRun reports what would be ingested without calling the extractor.
func Run(ctx context.Context, roots Roots, o ingest.Options /* Extractor+Daemon */, activeWindow time.Duration, dryRun bool) (Result, error)
```

Rules: a file is *changed* when no cursor exists, or `Size > cursor.ProcessedBytes`, or `ModTime != cursor.ModTime` with `Size < cursor.ProcessedBytes` (truncated/rotated → re-ingest from 0; episode idempotency dedupes). Loom run dirs are *changed* when dir mtime != cursor.ModTime. Errors on one file are collected into `Result.Errors` and do not abort the sweep.

- [ ] **Step 1: Failing tests** with temp dirs + fixtures from Tasks 2–3: fresh sweep ingests both fixture files; second sweep → all `FilesUnchanged`; appending a line → only the delta ingested; a file with mtime=now skipped as active; unreadable file lands in `Errors` while the rest proceed; dry-run calls no extractor (fake extractor counts calls).
- [ ] **Step 2:** FAIL → **Step 3:** implement + wire `memorySweepCmd()` (`--dry-run` flag; dormant-check; 30-minute timeout).
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit: `feat(memory): idempotent cursor-based sweep`

---

### Task 10: Backfill via Batch API

**Files:**
- Create: `internal/memory/extract/batch.go`
- Modify: `cmd/scry/memory.go` (fill in `memoryBackfillCmd()`)
- Test: `internal/memory/extract/batch_test.go`

**Interfaces:**

```go
package extract

// BatchExtractor submits episodes in batches of ≤1000 requests, polls until
// ended, and returns results keyed by episode ID. Same prompt as Haiku.Extract.
type BatchRunner struct{ /* client, model */ }
func NewBatchRunner(apiKey, model string) *BatchRunner
func (b *BatchRunner) Run(ctx context.Context, eps []distill.RawEpisode, glossary []string,
    progress func(done, total int)) (map[string]Result, map[string]error, error)
```

`memoryBackfillCmd()` flow: `--since <RFC3339 date>` (default: everything), `--no-batch` (falls back to serial `Haiku.Extract` with a 200ms sleep between calls). Uses `sweep.Roots` discovery but ignores cursors' ProcessedBytes for files older than any cursor (i.e. collect ALL episodes from files whose episodes aren't yet ingested — rely on `HasEpisode` idempotency by asking the daemon to commit; commits of known episodes are cheap no-ops). Collect episodes → filter `OccurredAt >= since` → chunk into batches → `Run` → commit each Result via `memory.commit` → advance cursors. Print running progress lines (`batch 2/5: 340/1000 done`).

Batch submission uses the Go SDK's Messages Batches surface: one request per episode with `custom_id` = episode ID and `params` identical to the single-shot call (same system blocks — the 50% discount applies regardless of caching). Poll `processing_status` every 60s until `"ended"`, then stream results and `ParseResult` each succeeded message; errored/expired custom_ids land in the error map. (Exact SDK type names from `anthropic-sdk-go`'s batches package; fix names from compiler errors, keep the flow.)

- [ ] **Step 1: Failing unit tests** for the pure parts (no network): request construction (custom_id mapping, ≤1000 chunking), result routing (succeeded→Result, errored→error map) — factor those into unexported funcs taking/returning plain values so they're testable without the client.
- [ ] **Step 2:** FAIL → **Step 3:** implement → **Step 4:** PASS (`go build ./...` too).
- [ ] **Step 5:** Commit: `feat(memory): batch API backfill`

---

### Task 11: MCP tools

**Files:**
- Create: `internal/mcp/memory_tools.go`
- Modify: `internal/mcp/server.go` (append `memoryToolDefinitions` to the tools list where the other domains are appended; add dispatch cases)
- Test: extend `internal/mcp/server_test.go` if it has a tools/list test; otherwise compile-check.

**Interfaces:**
- Consumes: Task 7 RPC names.
- Produces four MCP tools, defined exactly like `gitToolDefinitions` and dispatched via the existing forward-to-daemon helper pattern (`callGitQuery` style — add `callMemoryQuery`):

| Tool | → RPC | InputSchema properties |
|---|---|---|
| `scry_recall` | `memory.recall` | `query` (string, required), `as_of` (string), `limit` (integer, default 5) |
| `scry_memory_path` | `memory.path` | `from` (string, required), `to` (string, required) |
| `scry_episodes` | `memory.episodes` | `entity` (string, required), `limit` (integer, default 10) |
| `scry_remember` | `memory.remember` | `fact` (string, required), `entities` (array of string) |

Descriptions (verbatim — these are the model-facing trigger text):
- `scry_recall`: "Global cross-session memory: entities (projects, services, machines, people, decisions) with time-stamped facts extracted from past Claude/Codex sessions and loom runs. Use FIRST when the user references a project, machine, or decision not defined in the current context (e.g. 'set this up on hermes'). Returns matched entities, their current facts, and provenance."
- `scry_memory_path`: "How two remembered entities relate: shortest chain of facts between them (e.g. book-system → deployed_on → hermes-mini)."
- `scry_episodes`: "Recent episodes (session/run excerpts) that mention an entity, with summaries and source refs — use to trace when/where something was discussed."
- `scry_remember`: "Store a durable fact in global memory (e.g. a decision, a deploy, a preference with lasting relevance). Use instead of only stating it in prose when the fact should survive this session."

- [ ] **Step 1:** Add definitions + dispatch cases + `callMemoryQuery` (parse per-tool args, forward, wrap result as text JSON exactly like the git path).
- [ ] **Step 2:** `go build ./... && go test ./internal/mcp/` PASS.
- [ ] **Step 3:** Manual smoke once daemon rebuilt/restarted: from a scratch dir, call `scry_recall` via the MCP inspector or a live Claude session; empty result + no error is success at this stage.
- [ ] **Step 4:** Commit: `feat(memory): scry_recall/scry_memory_path/scry_episodes/scry_remember MCP tools`

---

### Task 12: Dotfiles — hooks + launchd sweep

**Files (in `~/dotfiles`, its own commits there):**
- Modify: `claude/settings.json` (hooks)
- Create: `launchd/com.jhoot.scry-memory-sweep.plist`
- Modify: `setup.sh` only if launchd agents aren't already globbed from `launchd/` (check first — the scry-steward plist is there, so loading is likely already handled)

- [ ] **Step 1:** Read `claude/settings.json`'s existing `hooks` blocks and match their exact structure. Add:
  - `SessionStart`: command `sh -c 'scry memory orient --cwd "$(pwd)" --budget 2000 2>/dev/null || true'` — stdout becomes injected context; the `|| true` keeps a broken daemon from blocking session start. (Hook input JSON also carries `cwd`; if the existing hooks parse stdin with jq, prefer `jq -r .cwd`.)
  - `SessionEnd`: command `sh -c 'p=$(jq -r .transcript_path); [ -f "$p" ] && (scry memory ingest --source claude --path "$p" >/dev/null 2>&1 &) ; true'` — backgrounded, never blocks, never errors.
- [ ] **Step 2:** Create the plist (mirror `com.jhoot.scry-steward.plist`'s shape):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>com.jhoot.scry-memory-sweep</string>
  <key>ProgramArguments</key><array>
    <string>/bin/zsh</string><string>-lc</string>
    <string>source ~/.secrets.zsh 2>/dev/null; scry memory sweep</string>
  </array>
  <key>StartInterval</key><integer>1800</integer>
  <key>StandardOutPath</key><string>/tmp/scry-memory-sweep.log</string>
  <key>StandardErrorPath</key><string>/tmp/scry-memory-sweep.log</string>
</dict></plist>
```

- [ ] **Step 3:** Confirm `ANTHROPIC_API_KEY` (or `SCRY_MEMORY_API_KEY`) is exported from `~/.secrets.zsh`. If absent, **stop and ask the user to add it** — do not write a key into any tracked file.
- [ ] **Step 4:** Load: `launchctl unload ~/Library/LaunchAgents/com.jhoot.scry-memory-sweep.plist 2>/dev/null; ln -sf ~/dotfiles/launchd/com.jhoot.scry-memory-sweep.plist ~/Library/LaunchAgents/ && launchctl load ~/Library/LaunchAgents/com.jhoot.scry-memory-sweep.plist` (match however setup.sh installs the steward plist). Verify: `launchctl list | grep scry-memory` and tail the log after a manual `launchctl start com.jhoot.scry-memory-sweep`.
- [ ] **Step 5:** Commit in dotfiles: `feat(memory): claude hooks + launchd sweep for scry memory` (only these files — the dotfiles tree has unrelated pending changes).

---

### Task 13: Dotfiles — forcing blocks (CLAUDE.md + AGENTS.md)

**Files:**
- Modify: `~/.claude/CLAUDE.md` (user-global; not in a repo — apply directly)
- Modify/Create: the Codex-side global instructions file that `ai-sync.py` manages (check `scripts/ai-sync.py` for which path it syncs — likely `~/.agents/AGENTS.md` or similar; put the block wherever Codex reads global guidance)

- [ ] **Step 1:** Append to the scry section of `~/.claude/CLAUDE.md`:

```markdown
**Unknown referent (project/service/machine/person/decision not defined in this conversation or repo):**
→ `scry_recall` FIRST — global memory of past sessions across all projects ("set this up on hermes" → recall "hermes"). Use `scry_episodes` to trace when something was discussed, `scry_memory_path` for how two things relate.

**Durable facts and decisions (deploys, choices made, lasting preferences):**
→ `scry_remember` — store it, don't just say it. This replaces hand-editing memory markdown.
```

- [ ] **Step 2:** Codex side: same block, plus (since Codex has no context-injection hook): `At the start of a session, run \`scry memory orient --cwd .\` and read the output before other work.`
- [ ] **Step 3:** Verify in a fresh Claude session in a scratch directory: the orientation block appears (SessionStart hook), and asking about a seeded entity triggers `scry_recall`.
- [ ] **Step 4:** Commit whichever of these files live in git (dotfiles `ai/` tree per ai-sync conventions).

---

### Task 14: Seed, backfill, verify (operational rollout)

No new code. Run on the built branch binary (`go install ./cmd/scry` or however scry is normally installed — check `which scry` and match).

- [ ] **Step 1:** Restart the daemon so the new RPCs are live (`scry stop && scry start`, matching existing usage).
- [ ] **Step 2:** Seed: `for f in ~/.claude/projects/*/memory/*.md; do scry memory ingest --source seed --path "$f"; done`. Then `scry memory entities` and hand-check: hermes-mini, loom, book-system, scry should exist with sane facts. Fix prompt/resolver and re-run if quality is off (episodes are idempotent; `rm -rf ~/.scry/memory` for a clean slate while iterating).
- [ ] **Step 3:** Recent backfill: `scry memory backfill --since $(date -v-90d +%Y-%m-%d)`. Spot-check with `scry memory recall "hermes"` and `scry memory recall "graph engineering"`. Then full: `scry memory backfill`.
- [ ] **Step 4:** End-to-end: new Claude session in an empty dir → orientation block present → "what do you know about running things on hermes?" → `scry_recall` returns the mini facts with provenance.
- [ ] **Step 5:** Merge: PR `memory-domain` → `main` in the scry repo (coordinate with the other active session; do not merge over their work).

---

## Self-review notes

- **Spec coverage:** data model (T1), distillation incl. redaction/chunking/skip-filter (T2–3), extraction + cached prefix + retry (T4), batch backfill (T10), resolution + invalidation rules 1–3 (T5), recall/orient/path/episodes + as-of (T6), daemon RPCs + dormancy + remember (T7), CLI verbs (T8, T9, T10 — `scry memory` matches the spec's CLI table; `memory.cursor.*` covers sweep state), hooks/launchd (T12), forcing blocks (T13), rollout order (T14). Open seams (mini, loom pre-DISCOVER, cross-graph joins, embeddings) intentionally have no tasks.
- **Known unknowns made explicit as steps:** exact Claude/Codex JSONL field names (T2/T3 Step 0 grounds them against real files before fixtures are written); exact anthropic-sdk-go type names for Messages/Batches (write-then-compile per the SDK guidance); loom run-dir file layout (T3 Step 0); ai-sync's Codex instructions path (T13 Step 2); hook JSON structure (T12 Step 1 reads the existing settings first).
- **Type consistency:** `store.Episode/Entity/Fact/Cursor` defined once in T1 and imported everywhere; `extract.Result` defined in T4 and used by T5/T7/T8/T10; RPC names in T7's table are the single source for T8's CLI and T11's MCP dispatch.
