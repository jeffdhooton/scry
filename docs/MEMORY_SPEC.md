# Scry Memory Spec

**Status:** Design approved 2026-07-28, not yet built.
**Depends on:** Unification phases 0–4 (shipped). Memory is effectively Phase 5.

## Why

Scry's four domains (code, schema, git, HTTP) are deterministic: parsed, never inferred. That was the right call for code — and it leaves a hole no parser can fill. Decisions, intent, cross-project state, and cross-session context live in *conversations*: Claude Code and Codex session transcripts, loom run verdicts. Today that knowledge is either lost at session end or hand-maintained in flat markdown memory files with no structure, no timestamps, and no traversal.

The memory domain adds the episodic half: an LLM-extracted, **temporal** knowledge graph built from session transcripts and loom runs, following the ingest/traverse split (cheap high-volume extraction, careful low-volume recall).

Three target capabilities, in priority order:

1. **Cross-session agent memory** — "What did we decide about X three weeks ago, and did anything since contradict it?"
2. **Cross-run loom memory** — loops that learn: "run N failed the way run M did; skip that path."
3. **Ops orientation** — inside a brand-new repo, "set this up to run on Hermes" resolves: the session knows what Hermes is, where it runs, and how deploys work.

## Non-goals

- Not a replacement for the per-repo deterministic graphs. Memory is a **global** store that links *down* into them.
- No doc/PDF ingestion, no embeddings, no vector search in v1. Recall is entity/alias match + graph traversal.
- No multi-machine sync in v1. The graph lives on the laptop. The mini is a future seam (see Open seams).
- No re-extraction of git history — the git domain already has it; memory links to commits, doesn't re-derive them.

## Architecture

Memory is scry's fifth domain: same binary, same daemon, same socket, same MCP server.

```
internal/memory/
  distill/     transcript JSONL → episode text (pure Go, no LLM)
  extract/     episode text → entities/edges via Haiku (interface-backed)
  resolve/     alias resolution, slug canonicalization, edge upsert + invalidation
  recall/      orient + recall queries, traversal
  sweep/       source cursors, idempotent ingestion of deltas
```

**Storage:** `~/.scry/memory/index.db` (BadgerDB, same idiom as per-repo indexes). Global — deliberately *not* under a repo.

**Dormancy:** no `ANTHROPIC_API_KEY` (or `SCRY_MEMORY_API_KEY` override) → the domain is dormant. CLI verbs print a one-line notice; MCP tools return empty results with a `dormant: true` flag; nothing else in scry is affected.

## Data model

### Episode
The unit of ingestion. Points at source material; does not duplicate it.

```
mem:episode:<id>  →  {
  id            sha256(source_path + span)     // idempotency key
  source        claude-session | codex-session | loom-run | seed | manual
  source_ref    path + line/byte span
  occurred_at   from transcript timestamps
  ingested_at
  summary       ≤ 2 sentences, extractor-written
}
```

### Entity

```
mem:entity:<slug>  →  {
  slug          canonical kebab-case id ("hermes-mini", "loom", "book-system")
  name, type    project | service | machine | tool | person | decision | runbook | concept
  description   evolving one-paragraph summary (extractor may revise)
  aliases       []string
  repo_refs     []string   // workspace repo paths this entity touches → join point to per-repo graphs
  created_at, last_seen
}
mem:alias:<normalized-name>  →  slug
```

### Fact (edge)
Temporal, Graphiti-style: facts are **invalidated, never deleted**.

```
mem:edge:<src>:<relation>:<dst>:<valid_from>  →  {
  src, dst      entity slugs
  relation      short verb phrase ("deployed_on", "replaced_by", "blocked_by", "uses", "decided")
  fact          one-sentence natural-language statement
  valid_from    when the fact became true (extractor-inferred, falls back to episode occurred_at)
  invalid_at    null = current; set when superseded
  confidence    0.0–1.0
  episodes      []episode_id   // provenance
}
```

**Invalidation rules** (applied in `resolve/`, in order):
1. Extractor emits an explicit `supersedes` hint → invalidate the referenced fact.
2. Exclusive relations (`deployed_on`, `status`, `replaced_by`, configurable list): a new fact with the same `(src, relation)` invalidates any current fact with a different `dst`.
3. Otherwise facts coexist. `scry memory invalidate <edge>` exists for manual correction.

Queries default to current facts (`invalid_at == null`); `--as-of <time>` reconstructs past state.

## Distillation (pure Go, runs before any LLM call)

Raw transcripts are ~95% tool payloads. Distillation extracts the conversational spine:

- **Claude JSONL** (`~/.claude/projects/**/*.jsonl`): keep user text and assistant text turns; drop tool_use/tool_result bodies, keep tool *names* as breadcrumbs ("[ran tests]", "[edited setup.sh]"). Keep the session's cwd and git repo as episode context.
- **Codex rollouts** (`~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`): envelope is `{timestamp, type, payload}` with types `session_meta` / `event_msg` / `response_item`. Keep user/assistant messages from `response_item`/`event_msg`; `session_meta` supplies cwd and start time.
- **Loom runs** (`~/.loom/runs/<name>/`): already structured — spec goal, iterations, gate verdicts, final status. Distill to a compact run narrative. High signal, tiny volume.
- **Seed** (`~/.claude/projects/*/memory/*.md`): one-time; each memory file becomes one episode with `occurred_at` from frontmatter/mtime.

**Chunking:** distilled sessions split into episodes of ~4k tokens on turn boundaries, 1-turn overlap.
**Redaction:** a regex pass strips obvious secrets (key-like tokens, `Bearer …`, PEM blocks) before any text leaves the machine.
**Skip filter:** sessions with < 3 substantive turns (permission-prompt stubs, aborted starts) are cursor-advanced without extraction.

## Extraction

**Model:** `claude-haiku-4-5` (config: `memory.model`). Chosen over DeepSeek deliberately: transcripts are the most sensitive data on the machine, and Anthropic/OpenAI already see their respective halves — Haiku for both adds no *new* third party and one extraction path. DeepSeek remains loom's executor brain only.

**Prompt layout (stable-first, variable-last, per the caching split):**

1. **System block, `cache_control: ephemeral`** — extraction schema, relation vocabulary, invalidation-hint rules, canonical-naming rules. Identical every call.
2. **Glossary block** — current entity slugs + aliases (top ~200 by degree). Changes slowly; refreshed per sweep batch, not per episode.
3. **Variable tail** — `reference_time`, episode context (cwd/repo), episode text.

**Output** (JSON, schema-validated, one retry on parse failure):

```json
{
  "episode_summary": "…",
  "entities": [{"name", "type", "description", "aliases": []}],
  "facts": [{"src", "relation", "dst", "fact", "valid_from", "confidence", "supersedes": null}]
}
```

`resolve/` then maps names → slugs via the alias table (create on miss), upserts facts, applies invalidation rules, updates `last_seen` and `repo_refs` (from episode cwd).

**Backfill** uses the Anthropic **Batch API** (50% off, stacks with prompt caching). Estimated one-time cost for full history (1.1 GB Claude + 0.5 GB Codex raw → ~5% survives distillation ≈ 20–40M tokens): **$10–30**. `--since <date>` caps it if desired.

## Ingestion triggers

Hooks for immediacy, a scheduled sweep as the guarantee. **The sweep is the source of truth; hooks are latency optimization.** A missed hook is a non-event.

- **Claude:** `SessionEnd` hook → `scry memory ingest --source claude --path <transcript> &` (fire-and-forget, backgrounded, never blocks the session).
- **Codex:** no session-end hook exists → covered entirely by sweep.
- **Loom:** covered by sweep. (Optional later: a post-run `scry memory ingest` call inside loom — a feeder seam, not a v1 requirement.)
- **Sweep:** `scry memory sweep` scans all source roots, compares against per-file cursors (`mem:cursor:<path>` → size + mtime + processed span), ingests deltas. Skips the transcript of any currently-active session (mtime < 5 min ago) to avoid half-session episodes. Scheduled by launchd: `com.jhoot.scry-memory-sweep.plist`, every 30 minutes. Separate plist from scry-steward (different job, different failure domain).

## Recall

### Pull — MCP tools (on the existing scry MCP server)

| Tool | Contract |
|------|----------|
| `scry_recall` | `query, [as_of], [limit]` → matched entities (alias match), their current facts, provenance episode refs. The workhorse. |
| `scry_memory_path` | `from, to` → shortest fact-chain between two entities ("how does book-system relate to the mini?") |
| `scry_episodes` | `entity, [limit]` → recent episodes touching an entity, with summaries + source refs |
| `scry_remember` | `fact, [entities]` → agent/user-initiated durable fact, ingested as a `manual` episode (replaces hand-editing memory markdown) |

### Push — thin orientation

`scry memory orient --cwd <dir> [--budget 500]` emits compact markdown:

- entities whose `repo_refs` match cwd (or whose slug matches the repo name), each with its 3 most recent current facts
- globally active projects (`last_seen` < 14 days), one line each
- a closing pointer: *"Query scry_recall for anything referenced here."*

Hard token budget (default ~500). Wired as a Claude `SessionStart` hook (additionalContext). Codex has no injection hook → AGENTS.md instructs running `scry memory orient` as the first action in a session.

### Joining down into code graphs

v1: entity `repo_refs` appear in recall output, so the agent can pivot to per-repo `scry_graph_*` tools itself. A native cross-graph `scry_graph_path` join (memory entity → code node) is an explicit later phase — requires nothing in v1's schema beyond `repo_refs`, which it has.

## Forcing usage (Claude & Codex)

- **Global CLAUDE.md** (`~/.claude/CLAUDE.md`): extend the existing "scry FIRST" block — unknown referent (project/service/person/decision not in context) → `scry_recall` before asking or guessing; durable facts/decisions → `scry_remember` instead of prose-only.
- **AGENTS.md** (Codex side, distributed via existing `ai-sync.py` flow): same block + the orient-first instruction.
- **Hooks** (`~/dotfiles/claude/settings.json`): `SessionStart` → orient; `SessionEnd` → ingest.
- MCP tool availability rides the existing scry server entry that `ai-sync.py` already syncs to both tools — zero new MCP wiring.

## CLI

```
scry memory ingest    --source claude|codex|loom|seed --path <p>
scry memory sweep     [--dry-run]
scry memory backfill  [--since <date>] [--no-batch]     # Batch API by default
scry memory orient    [--cwd <dir>] [--budget <tokens>]
scry memory recall    "<query>" [--as-of <time>]
scry memory entities  [--type t] | scry memory facts <slug>
scry memory invalidate <src> <relation> <dst>
scry memory status                                       # cursors, counts, dormancy, last sweep
```

## Testing

- `extract/` behind an interface; unit tests run a fake extractor. Golden fixtures: real (sanitized) Claude/Codex/loom transcript snippets → expected distillation, plus canned extractor JSON → expected graph mutations.
- Invalidation table-tests (exclusive relations, supersedes hints, as-of queries).
- Sweep tests: cursor advance, idempotent re-ingest (same episode id → no dupes), active-session skip, malformed-JSONL resilience.
- One live Haiku smoke test gated behind `SCRY_LIVE_TEST=1`.

## Rollout order

1. Build domain (distill → extract → resolve → store) + CLI + tests
2. Seed from memory markdown; verify entities/facts by hand (`scry memory recall`)
3. Backfill recent 90 days via Batch API; spot-check quality; then full history
4. Enable hooks + launchd sweep
5. Ship CLAUDE.md / AGENTS.md forcing blocks + orient injection
6. Watch a week; tune glossary size, orient budget, exclusive-relation list

## Open seams (explicitly deferred)

- **Mini/multi-machine:** Hermes sessions on the mini generate their own transcripts. Later: sweep-over-ssh or a mini-local scry pushing episode JSON. Nothing in the schema blocks it (episodes already carry source refs).
- **Loom pre-DISCOVER recall:** loom querying `scry memory recall` before its DISCOVER phase — the cross-run learning loop closed from the consumer side.
- **Cross-graph native joins:** memory entity ↔ code node paths inside `scry_graph_path`.
- **Embedding-assisted recall** if alias matching proves too brittle for fuzzy queries. *2026-09-02: fact-level lexical search (BM25) landed instead; see `docs/DECISIONS.md`. Embeddings stay deferred.*

*2026-09-02: the "Mini/multi-machine" seam closed with the shared memory socket (2026-08-28) and daemon-side extraction (2026-09-02); the mini runs its own sweep into the same store.*
