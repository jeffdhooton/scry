# scry — Code Intelligence Daemon for AI Agents

**Status:** Spec / PRD draft
**Audience:** Fresh Claude Code instance building this from scratch
**Working name:** `scry` — committed. (Minor collision with the Crystal-lang LSP server of the same name; see Appendix C.)

---

## 1. Pitch

Every AI coding session today spends 30-50% of its tool calls on `Read`, `Grep`, and `Glob` just to *find* the relevant code before doing anything with it. The agent greps for a symbol, gets back 47 results across 12 files, reads each one, discards 80% of it, and finally has the context to make a change. That round trip is 5-15 seconds and 5000+ tokens *per question*. It happens dozens of times per session.

LSP exists, but LSP is human-first. It was designed for a 60-fps editor showing hover tooltips and autocomplete dropdowns. It is excellent at "what's at line 47 column 12 right now" and bad at "give me a graph of every function that eventually calls `processOrder`, with one line of context for each." Agents need the second thing.

**scry** is a long-running daemon that pre-computes a semantic index of every repository you work in — symbols, references, definitions, call graphs, type relationships, dependencies — and exposes it as a millisecond-latency local API. Agents query it instead of reading and grepping files.

```
Before scry:
  agent: Grep("processOrder")           # 800ms, 47 hits across 12 files
  agent: Read(file1.ts)                 # 200ms, 800 lines
  agent: Read(file2.ts)                 # 200ms, 1200 lines
  agent: Read(file3.ts)                 # 200ms, 600 lines
  ...                                   # repeat ~9 more times
  total: 5-10 seconds, ~5000 tokens of mostly-irrelevant context

After scry:
  agent: scry refs processOrder         # 8ms
  total: 8ms, ~500 tokens of just the relevant lines
```

The wins are 100x latency, 10x token cost, and dramatically higher precision (semantic refs eliminate the false positives from grep matching strings or comments).

scry is built as a **single static Go binary** with a long-running daemon (`scryd`) and a thin client CLI (`scry`). The daemon is auto-spawned on first call and runs in the background until you stop it. Same architectural philosophy as `trawl`: agent-first, not human-first.

---

## 2. The thesis

The developer toolchain is being rewritten for agents, and the rewrites are 10-100x faster because they don't carry the human-UI tax. trawl is one instance (Playwright is human-first, trawl is agent-first). scry is the next: LSP is human-first, scry is agent-first.

The two designs differ in *what they precompute, when, and how they expose it*:

| Concern | LSP (human-first) | scry (agent-first) |
|---|---|---|
| Optimized for | Real-time editor UX, hover, autocomplete | Batch queries, structured output |
| Indexing | Lazy, on-demand | Eager, precomputed |
| Latency target | <100ms (one position at a time) | <10ms (any query, any scale) |
| Output | Pretty hover cards, jump locations | JSON, structured for piping |
| Stateful client | Yes (editor session) | No (CLI is stateless, daemon holds state) |
| Per-language | One LSP server per language | Multi-language, one daemon |
| Deploy | Editor extension | `scry` on PATH |

The two coexist fine — LSP for humans, scry for agents — because they're solving different problems even though they touch the same data.

---

## 3. Goals & Non-Goals

### Goals
- **Sub-10ms p50 query latency** for the common queries (refs, defs, callers, symbols).
- **Sub-100ms incremental update** when a file changes.
- **Pre-computed semantic index**, refreshed by a file watcher. Never on-demand parsing.
- **Multi-language**: TypeScript/JavaScript, Go, Python, Bash for v1.
- **Multi-repo**: one daemon per user, many indexed repos.
- **Single static binary** (no CGO, no runtime, no daemon manager required).
- **Local-only.** No cloud, no telemetry, no network calls outside of fetching language indexers.
- **Agent-first CLI surface.** JSON output by default; `--pretty` for human reading.
- **Drop-in replacement** for the most common Read/Grep patterns. The CLI verbs should map to what an agent actually wants.

### Non-Goals (v1)
- **IDE integration.** Editors have LSP. scry does not need to be an LSP server (though Appendix B explores making it one as a v3 idea).
- **Code completion / autocomplete.** Real-time human typing isn't the use case. Skip.
- **Diagnostics, linting, formatting.** Other tools do this. scry surfaces structure, not opinions.
- **Refactoring engine.** The agent does the refactor — scry just finds the call sites and the agent edits them.
- **Full LSP protocol compatibility.** scry exposes what agents need, not the 80-method LSP surface.
- **Code generation.** Out of scope.
- **Cloud / distributed mode.** scry is local-first. If a team needs shared code intelligence, Sourcegraph already does that.
- **Cross-repo references.** v1 indexes one repo at a time. v2 may add cross-repo for monorepos and tightly-coupled multi-repo setups.
- **Custom DSLs / proprietary languages.** Stick to languages that have well-maintained SCIP indexers or tree-sitter grammars.

---

## 4. Core concepts

### 4.1 The Daemon (`scryd`)

A long-running background process. **One per user.** Manages all indexed repos. Listens on a Unix domain socket at `~/.scry/scryd.sock`. Auto-spawned on the first `scry` CLI call if not running. Runs until explicitly stopped (`scry stop`) or the user logs out.

Why a daemon and not on-demand:
- Index build is expensive (~30-90s for a 100k LOC repo). Re-doing it on every query is impossible.
- Incremental updates need a process that holds the index in memory and watches the filesystem.
- Connection setup over a Unix socket is ~50µs vs ~50ms+ for spawning a new process and parsing arguments. Agents make hundreds of queries per session — the per-query overhead matters.

### 4.2 The Index

A per-repo precomputed semantic index, stored in BadgerDB. Built once via SCIP indexers (Sourcegraph's industry-standard format) plus tree-sitter for syntactic queries, kept fresh via fsnotify-driven incremental updates.

The index contains:
- **Symbols** — every named entity (functions, classes, methods, variables, types, interfaces) with file:line:column, kind, signature, docstring.
- **References** — every use of every symbol, with file:line:column and 1-3 lines of surrounding context.
- **Definitions** — the canonical definition site for every symbol.
- **Call graph** — caller/callee relationships across files.
- **Type relationships** — implements, extends, uses-as-parameter, returns.
- **Dependencies** — file-level and module-level import graph.
- **File metadata** — language, line count, last-indexed timestamp, hash.

Each repo's index lives in its own directory under `~/.scry/repos/<hash>/`. Repos are identified by their root path's SHA256.

### 4.3 The CLI (`scry`)

A thin client. It does no parsing, no indexing — it opens the Unix socket, sends a JSON-RPC request, prints the response. JSON output by default. `--pretty` for human reading.

```bash
scry refs processOrder
scry defs OrderService
scry callers handlePayment
scry callees main
scry impls Repository
scry symbols src/orders/service.ts
scry symbols --query 'order*'
scry hover src/orders/service.ts:47:12
scry deps src/orders/service.ts
scry rdeps src/orders/service.ts
scry graph --from main.ts --depth 3
scry diff main..feature
```

If the daemon isn't running, the CLI auto-spawns it and waits for the socket to come up (up to 2s). If the current repo isn't indexed, the CLI queues an index build and returns either a "still indexing, partial results" response or blocks until ready (configurable via `--wait`).

### 4.4 The Watch Loop

The daemon runs an fsnotify watcher on every indexed repo. On a file change:
1. Debounce ~100ms (don't re-index on every key press if the user is editing)
2. Check the file hash — if unchanged, skip
3. Run an incremental re-index of the affected file via the language's SCIP indexer
4. Update the in-memory index and queue a flush to BadgerDB
5. Invalidate any cached queries that depended on the file

Targets: <100ms incremental update for a single-file edit. The flush to disk is async and never blocks queries.

---

## 5. Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        scry CLI                              │
│   scry refs <symbol> | defs | callers | symbols | ...        │
└────────────────────────┬─────────────────────────────────────┘
                         │ JSON-RPC over Unix socket
                         │ ~/.scry/scryd.sock
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                        scryd                                 │
│   ┌────────────────────────────────────────────────────┐    │
│   │            JSON-RPC dispatcher                     │    │
│   └─────────────────────┬──────────────────────────────┘    │
│                         │                                    │
│   ┌─────────────────────▼──────────────────────────────┐    │
│   │              Query Engine                          │    │
│   │   refs | defs | callers | symbols | hover | ...    │    │
│   └─────────┬───────────────────────────┬──────────────┘    │
│             │                           │                    │
│   ┌─────────▼─────────┐    ┌───────────▼───────────┐        │
│   │   Index Store     │    │   File Watcher        │        │
│   │   (BadgerDB)      │◀───│   (fsnotify +         │        │
│   │                   │    │    debounced reindex) │        │
│   └─────────▲─────────┘    └───────────────────────┘        │
│             │                                                │
│   ┌─────────┴─────────────────────────────────────┐         │
│   │           Indexer Pool                        │         │
│   │   ┌─────────┐ ┌─────────┐ ┌─────────┐         │         │
│   │   │ scip-ts │ │ scip-go │ │ scip-py │  ...    │         │
│   │   └─────────┘ └─────────┘ └─────────┘         │         │
│   │   ┌─────────────────────────────────┐          │         │
│   │   │   tree-sitter (syntactic)       │          │         │
│   │   └─────────────────────────────────┘          │         │
│   └─────────────────────────────────────────────────┘         │
└──────────────────────────────────────────────────────────────┘
```

### Index build flow

1. User runs `scry init` (or first `scry <query>` in an unknown repo)
2. Daemon detects languages present in the repo
3. For each detected language, daemon spawns the appropriate SCIP indexer (`scip-typescript`, `scip-go`, `scip-python`)
4. Indexer produces a `.scip` file (Protobuf-encoded semantic index)
5. Daemon parses the SCIP file, normalizes the data, writes it into BadgerDB
6. Daemon starts the file watcher on the repo
7. Index is "warm" — queries are now <10ms

Cold build for a 100k-LOC TypeScript repo target: <60s. Warm queries from then on: <10ms p50.

---

## 6. The Query API

The CLI surface and the JSON-RPC method names are 1:1. Every query takes optional `--repo <path>` (defaults to the cwd) and `--limit <N>` (defaults to 1000). Output is JSON unless `--pretty` is passed.

### 6.1 Symbol queries

```bash
scry refs <symbol>          # all references to a symbol
scry defs <symbol>          # the definition site(s)
scry symbols [path]         # all symbols in a file or repo
scry symbols --query <pat>  # symbol search with glob/fuzzy match
scry hover <file:L:C>       # symbol info at a position
```

**Output** (`scry refs processOrder`):
```json
{
  "symbol": "processOrder",
  "results": [
    {
      "file": "src/orders/service.ts",
      "line": 47,
      "column": 12,
      "kind": "call",
      "context": "  return processOrder(order, opts);",
      "containing_symbol": "OrderService.handle"
    },
    ...
  ],
  "total": 47,
  "elapsed_ms": 8
}
```

### 6.2 Call graph queries

```bash
scry callers <function>     # who calls this function (1 hop)
scry callees <function>     # what this function calls (1 hop)
scry graph --from <symbol> --depth <N>  # full call graph from a root
```

### 6.3 Type / interface queries

```bash
scry impls <interface>      # everything that implements an interface
scry extends <type>         # everything that extends a base type
scry types <symbol>         # type info for a symbol
```

### 6.4 Dependency queries

```bash
scry deps <file>            # what does this file import
scry rdeps <file>           # what files import this file
scry deps --module <pkg>    # all uses of an external package
```

### 6.5 Diff / change queries

```bash
scry diff <ref1>..<ref2>    # semantic diff between two git refs
                            # (which symbols changed, not which lines)
```

### 6.6 Daemon control

```bash
scry init [path]            # initialize/index a repo
scry status                 # daemon status, indexed repos, memory
scry stop                   # stop the daemon cleanly
scry start                  # start the daemon (mostly auto-spawned)
scry watch <path>           # add a path to the watched repos
scry unwatch <path>         # remove a path
scry reindex [path]         # force a full rebuild
```

### 6.7 Output conventions

- All `file` paths are absolute by default. Pass `--relative` to get repo-relative paths.
- All line/column numbers are 1-indexed (matching what humans and agents expect, not 0-indexed internal LSP convention).
- All `elapsed_ms` is end-to-end including socket round-trip.
- Errors return `{"error": {"code": "...", "message": "..."}}` with sensible HTTP-like codes.
- The JSON shape is **stable**. Versioned via the `--api v1` flag if breaking changes are needed later.

---

## 7. Tech stack (decided — don't relitigate)

| Concern | Choice | Why |
|---|---|---|
| Language | Go 1.23+ | Single binary, daemons, fast startup, same as trawl |
| Indexers | scip-typescript, scip-go, scip-python (shelled out) | Industry standard, Sourcegraph-maintained, accurate |
| AST / syntactic | `tree-sitter` (via `smacker/go-tree-sitter`) | Fast, multi-language grammars, no LSP overhead |
| Storage | `dgraph-io/badger/v4` | Embedded LSM, same as trawl, single dir |
| File watching | `fsnotify/fsnotify` | Standard cross-platform |
| RPC protocol | JSON-RPC 2.0 over Unix domain socket | Simple, agent-friendly, ~50µs round trip |
| CLI | `spf13/cobra` | Standard, same as trawl |
| Logging | `rs/zerolog` | Standard, same as trawl |
| Config | `spf13/viper` | YAML/TOML, same as trawl |
| Hashing | `crypto/sha256` (stdlib) | Repo identity, file change detection |

**Hard constraints:**
- **No CGO.** Single static binary, cross-compile freely. Same as trawl.
- **No network calls** except to download language indexers (`scip-typescript` etc.) on first use, into `~/.scry/bin/`.
- **No telemetry, ever.** This tool reads your source code; trust matters.

---

## 8. Storage layout

```
~/.scry/
├── scryd.sock           # Unix domain socket (daemon listens here)
├── scryd.pid            # daemon PID file
├── scryd.log            # daemon log (zerolog JSON, rotated by size)
├── config.yaml          # user config (optional)
├── bin/                 # downloaded language indexers
│   ├── scip-typescript
│   ├── scip-go
│   └── scip-python
└── repos/
    ├── 8a3f.../
    │   ├── manifest.json    # repo metadata: path, languages, last_indexed
    │   ├── index.db          # BadgerDB
    │   └── scip.bin          # raw SCIP dump (kept for reindex without full re-parse)
    └── ...
```

Every repo gets a directory keyed by `sha256(absolute_repo_path)[:8]`. The manifest tells the daemon which path that hash corresponds to. The BadgerDB inside each repo dir holds the normalized index.

---

## 9. Index build process

For each repo:

1. **Detect languages.** Walk the root, count files by extension. Anything above a 1% threshold gets indexed. (Skip `node_modules`, `vendor`, `dist`, `.git`, `target`, etc. — configurable via `.scryignore`.)

2. **Download indexers if missing.** First time we see TypeScript: download `scip-typescript` from the Sourcegraph release page into `~/.scry/bin/`. Verify the binary's SHA256 against a pinned hash list shipped with scry. Cache forever.

3. **Run indexers in parallel.** Each language's SCIP indexer runs as a subprocess against the repo, producing a `.scip` file in `~/.scry/repos/<hash>/scip-<lang>.bin`.

4. **Parse + normalize.** Read the SCIP protobuf, walk the symbol table, write normalized records into BadgerDB:
   - `sym:<symbol_id>` → symbol record (kind, name, file, line, column, signature)
   - `ref:<symbol_id>:<seq>` → reference record (file, line, column, context, containing_symbol)
   - `def:<symbol_id>` → definition site
   - `file:<path>` → file record (language, line count, hash, indexed_at)
   - `call:<caller_id>:<callee_id>` → call graph edge
   - `impl:<interface_id>:<impl_id>` → implements relationship

5. **Build secondary indices** for fast queries:
   - Symbol name → list of symbol IDs (for `scry refs <name>`)
   - File → list of symbol IDs in that file (for `scry symbols <file>`)
   - Module → list of file IDs (for `scry deps`)

6. **Tree-sitter pass for syntactic enrichment.** Walk every file once with the appropriate tree-sitter grammar to extract per-line context for ref records (the 1-3 surrounding lines that get returned with each ref). This is fast (~50ms per 1000 LOC) and improves the agent UX significantly.

7. **Mark repo as ready.** Write `manifest.json` with `status: ready`, `last_indexed: <timestamp>`, language list, file count.

8. **Start file watcher.** fsnotify on the repo root.

### Incremental update flow

When fsnotify reports a file change:

1. Debounce 100ms
2. Hash the file. If unchanged, skip (handles editor save-and-exit-and-format flows)
3. Run the language indexer with `--single-file <path>` (most SCIP indexers support this)
4. Diff the new symbol set against the old. Add new symbols, remove deleted ones, update references to surviving ones
5. Update BadgerDB
6. Invalidate any in-memory query cache entries that touched this file

Target: <100ms for a single-file change in a 100k-LOC repo.

---

## 10. Performance targets

| Metric | Target |
|---|---|
| Query latency p50 (`scry refs <symbol>`) | <10ms |
| Query latency p99 | <50ms |
| Cold index build, 100k-LOC TS repo | <60s |
| Cold index build, 1M-LOC monorepo | <8min |
| Incremental update, single file | <100ms |
| Daemon RAM, 100k-LOC repo | <500MB |
| Daemon RAM, 1M-LOC repo | <3GB |
| Daemon cold start (auto-spawn) | <500ms |
| Binary size | <30MB |
| Disk usage per 100k-LOC repo | <200MB |

These are *targets*, not hard requirements. If P0 lands at 20ms p50 instead of 10ms, that's still a massive improvement over the Read+Grep baseline. The build agent should measure and document, not chase the exact number.

---

## 11. Languages supported in v1

**P0 (must work):**
- **TypeScript / JavaScript** — highest priority. Indexer: `scip-typescript` (Sourcegraph first-party, npm: `@sourcegraph/scip-typescript`).

**P1 (next):**
- **Go** — easy validation of the multi-language pipeline. Indexer: `scip-go` (Sourcegraph first-party, install: `go install github.com/sourcegraph/scip-go/cmd/scip-go@latest`).
- **PHP** — the harder real-world target. See §11.1 below for the indexer story; this is the wildcard of v1.

**P2 (round out the v1 set):**
- **PHP Laravel-aware enhancements** — see §11.1.

**Deferred to post-v1:**
- Python (`scip-python` exists and is mature; deferred only because no v1 user has explicitly asked for it. Easy to add when someone does.)
- Bash / zsh (no SCIP indexer; would need tree-sitter only, which limits cross-file refs to nothing useful for shell scripts)
- Rust, Ruby, Java/Kotlin, C/C++, Swift, C# — all have SCIP indexers of varying maturity. Add on demand.

The build agent should not implement custom indexers for any of these. Always shell out to the official SCIP indexer if one exists. PHP is the only language in v1 where "the official indexer" doesn't really exist (see below).

### 11.1 PHP — the wildcard

**The honest situation:** Sourcegraph never shipped a first-party `scip-php`. The PHP language is conspicuously missing from their indexer roster despite being one of the top web languages. There is exactly one community implementation:

- **[davidrjenni/scip-php](https://github.com/davidrjenni/scip-php)** — uses `nikic/PHP-Parser` (the canonical PHP AST library), v0.0.2 last tagged April 2023, 16 stars, solo maintainer, essentially in maintenance-mode dormancy.

**The bigger problem is Laravel.** Laravel is one of the most dynamic PHP frameworks in existence. The patterns that matter for cross-file references — facades (`Auth::user()` static-proxy magic), service container resolution (`app(FooService::class)`), Eloquent's `__call`/`__get` magic, route closures bound via reflection — are all *runtime* behaviors that no static AST analysis can resolve without framework-specific knowledge. A naive AST-based indexer like scip-php will silently miss a meaningful fraction of references in any Laravel codebase. The references it *does* find will be correct; the ones it can't statically resolve simply won't appear.

**The strategy:**

**P1: Ship `davidrjenni/scip-php` as the engine, with caveats clearly documented.**
- Pin to a specific commit (do not use the v0.0.2 tag — too stale)
- Ship behind a `php_indexer` config flag so users opt in
- Document the Laravel limitations in `docs/PHP.md` so users aren't surprised
- Cover vanilla PHP well; treat Laravel quality as "best-effort"

**P2: Build a Laravel-aware post-processor.**
- Walks the SCIP output produced by scip-php
- Synthesizes additional symbols and references for the dynamic patterns scip-php misses:
  - **Facade resolution.** `Auth::user()` → resolves through the facade accessor → adds a synthesized ref to `Illuminate\Auth\AuthManager::user()`.
  - **Container bindings.** `app(FooService::class)` and `resolve(FooService::class)` → ref to `FooService`.
  - **Eloquent relationships.** Models declaring `hasMany`/`belongsTo`/etc. → add ref edges to the related models.
  - **Route actions.** `Route::get('/users', [UserController::class, 'index'])` → ref to `UserController::index`.
  - **View templates.** `view('users.show', [...])` → ref to `resources/views/users/show.blade.php` (a kind of pseudo-symbol).
  - **Service provider bindings.** `$this->app->bind(Contract::class, Implementation::class)` → adds a `bind` relationship row.
- Implementation lives in `internal/sources/php/laravel/` as a separate normalization pass after SCIP parsing
- Detection: presence of `composer.json` containing `laravel/framework` triggers the post-processor

**P3 (only if P1+P2 prove insufficient): Replace the engine with Phpactor.**
- [Phpactor](https://github.com/phpactor/phpactor) (1.8k stars, actively maintained, last release December 2025) has a richer internal index than scip-php produces.
- It exposes `phpactor index:build` and `phpactor index:query` but no batch SCIP export — adapting it would require either iterating `index:query` (slow) or linking Phpactor's PHP library directly and walking its internal index.
- Make the PHP engine **swappable** in P1's design so this replacement is an `internal/sources/php/` swap, not a rewrite of the whole pipeline.

**What the user sees:**

```bash
scry refs UserController          # works for the static ~70% of refs in vanilla PHP
                                   # works for ~90%+ of refs after the Laravel post-processor

scry refs FooService               # may miss container-resolved instances in P1
                                   # picked up after the Laravel post-processor in P2
```

The `docs/PHP.md` user guide should explicitly list which patterns are caught and which aren't, with examples, so users have realistic expectations. There is no version of PHP support that's as polished as the TypeScript path because the static analysis ceiling for dynamic PHP is genuinely lower than for typed JS/TS or Go.

### 11.2 Indexer install matrix

| Language | Indexer | Source | Install | Maturity |
|---|---|---|---|---|
| TypeScript / JavaScript | `scip-typescript` | Sourcegraph (first-party) | `npm i -g @sourcegraph/scip-typescript` | Production |
| Go | `scip-go` | Sourcegraph (first-party) | `go install github.com/sourcegraph/scip-go/cmd/scip-go@latest` | Production |
| PHP (vanilla) | `scip-php` (community fork or vendored) | davidrjenni/scip-php on `nikic/PHP-Parser` | `composer require davidrjenni/scip-php` (or vendored binary) | Experimental |
| PHP (Laravel) | scry's Laravel post-processor | scry-internal | bundled | New (P2) |

The auto-download flow in §9 step 2 handles the SCIP-distributed indexers (TypeScript, Go) cleanly. **PHP is the exception** — it's a Composer package, requires PHP runtime + Composer to install, and isn't a standalone binary. The build agent has two options:

1. **Vendor a wrapped binary.** Use `box-project/box` or similar to compile scip-php into a single PHAR archive, ship that as a downloadable artifact in scry's own GitHub releases. Removes the user's PHP/Composer dependency.
2. **Require the user to install PHP + Composer + scip-php themselves.** Document the install. Less invasive, more friction.

Recommendation: option 1 for P1, fall back to option 2 if PHAR packaging proves brittle. The user shouldn't need to know what Composer is to use scry on a PHP project.

---

## 12. Claude Code integration

scry is most valuable when agents actually use it. Three integration paths:

### 12.1 Manual (P0)
Agent learns from `CLAUDE.md` that `scry` is available and prefers it for symbol queries:

```markdown
## Code intelligence

`scry` is available on this machine. Prefer it over Grep/Read for symbol lookups:
- `scry refs <symbol>` instead of grepping for a function name
- `scry defs <symbol>` instead of greping + reading
- `scry callers <function>` to find where a function is called
- `scry symbols <file>` instead of reading a whole file just to see its structure
```

### 12.2 gstack skill wrapper (P1)
A `/scry` skill in gstack that:
- Detects `scry` is installed
- Documents the common query patterns
- Provides natural-language wrappers ("show me everything that calls processOrder")
- Auto-runs `scry init` on the current repo if not yet indexed

### 12.3 Hook-based interception (P2 / experimental)
A Claude Code PreToolUse hook that intercepts `Grep` and `Read` calls:

- If a `Grep` pattern looks like a symbol name (single word, identifier-shaped), check whether scry knows that symbol. If yes, route to scry instead.
- If a `Read` is for a file in an indexed repo and the agent only needs structure (not content), return `scry symbols <file>` results instead.

This is the highest-leverage integration but also the most invasive. It should be optional and clearly documented. Ship it as a separate gstack skill (`/scry-hook`) that the user explicitly enables.

---

## 13. Build phases

### P0 — MVP (1-2 weeks)
The thinnest version that proves the idea.

- [ ] Project scaffold (cobra, viper, zerolog)
- [ ] `internal/store/` BadgerDB-backed index store
- [ ] `internal/sources/scip/` SCIP file parser (consumes the .scip protobuf format)
- [ ] `internal/sources/typescript/` TypeScript indexing (shells out to scip-typescript)
- [ ] `internal/index/` index build pipeline
- [ ] `internal/query/` query engine for the four core queries: `refs`, `defs`, `symbols`, `hover`
- [ ] `cmd/scry/` CLI with: `init`, `refs`, `defs`, `symbols`, `hover`, `status`
- [ ] **No daemon yet.** P0 reads the BadgerDB index directly from the CLI. Slower per-query (cold start ~50ms) but correct and simpler to validate.
- [ ] Manual download of `scip-typescript` (don't auto-fetch yet)
- [ ] Tests against a real OSS TypeScript repo (`microsoft/vscode` is a good stress test)

**Done when:** running `scry init` on a 50k-LOC TypeScript repo completes in <60s and `scry refs <symbol>` returns accurate results in <100ms (CLI cold start included).

### P1 — Daemon mode + Go support (2-4 weeks)
The daemon is what makes scry actually fast.

- [ ] `cmd/scryd/` daemon binary (or merge into `cmd/scry/` with `scry start --daemon`)
- [ ] Unix socket listener
- [ ] JSON-RPC dispatcher
- [ ] CLI auto-spawns daemon if not running
- [ ] In-memory index cache layer above BadgerDB
- [ ] fsnotify watcher per indexed repo
- [ ] Incremental file re-indexing
- [ ] Auto-download `scip-typescript` and `scip-go` on first use, with SHA256 verification
- [ ] Go language support (shell out to `scip-go`)
- [ ] `callers` and `callees` queries (call graph)
- [ ] `impls` query (interface implementors)

**Done when:** the daemon starts in <500ms, `scry refs <symbol>` runs at <10ms p50 over the socket, file edits in a watched repo show up in queries within 200ms.

### P2 — Round out v1 (3-5 weeks)
Production-ready single-user daemon.

- [ ] Python support (`scip-python`)
- [ ] Bash support (tree-sitter only — no SCIP indexer for shell, but the syntactic surface is thin enough that tree-sitter alone is enough)
- [ ] Multi-repo support: daemon manages many repos, queries scope to cwd
- [ ] `scry watch` / `unwatch` / `reindex`
- [ ] `.scryignore` file support
- [ ] `deps` and `rdeps` queries (dependency graph)
- [ ] `scry diff` (semantic diff between git refs)
- [ ] `scry status` shows full daemon state: indexed repos, memory, query stats
- [ ] LaunchAgent (macOS) and systemd unit (Linux) for autostart (optional)
- [ ] gstack `/scry` skill wrapper

**Done when:** scry can be installed on a fresh machine, `scry init` against a polyglot repo (TS + Go + Python + Bash), and all queries return correct results across all four languages. Daemon survives a multi-day idle period without leaks.

### P3 — Intelligence + integration (ongoing)
The features that unlock the second wave of value.

- [ ] PreToolUse hook for Claude Code (intercepts Grep/Read, routes to scry)
- [ ] `scry graph` full call graph traversal with depth control
- [ ] Semantic search via embeddings (find symbols by description, not exact name)
- [ ] `scry hover` enhancement: include docstrings, type definitions, related symbols
- [ ] LSP server mode (expose scry as a real LSP server so editors can use it too — controversial, see Appendix B)
- [ ] Cross-repo references for monorepos and tightly-coupled multi-repo setups
- [ ] Index sharing across users (read-only mode for shared dev machines)

---

## 14. Success criteria

A v1 release is successful if all of these are true:

1. **Latency.** `scry refs <symbol>` over the socket runs at <10ms p50 against a 100k-LOC TypeScript repo.
2. **Accuracy.** Symbol references returned by scry match the references found by VSCode's "Find All References" with ≥98% recall and ≥99% precision on a benchmark suite.
3. **Freshness.** A file edit shows up in query results within 200ms of save.
4. **Resource use.** Daemon RAM stays under 500MB on a 100k-LOC repo over a 24-hour session.
5. **Stability.** Daemon survives 7 days of normal use (idle + bursty queries + file edits) without crashing or leaking.
6. **Languages.** All four v1 languages (TS, Go, Python, Bash) work end-to-end.
7. **Discoverability.** A fresh Claude Code session in a scry-indexed repo, with the `/scry` skill enabled, naturally chooses scry over Grep/Read for symbol lookups.

The most important test is qualitative: **does the user prefer it over Grep in real Claude Code sessions?** If yes, scry is real. If they keep falling back to Grep because scry is too slow, too inaccurate, or too rough around the edges, it isn't.

---

## 15. Open questions for the build agent

These are the decisions I deliberately did NOT make. The build agent should resolve them, document the choice in `docs/DECISIONS.md`, and move on.

1. **One binary or two?** Should `scry` and `scryd` be the same binary with a `--daemon` flag, or two binaries built from one Go module? Lean toward one binary (less ops surface). Document the choice.
2. **Where does the daemon log?** `~/.scry/scryd.log` is the spec default. Rotation policy? Size-based, time-based, both? Pick one.
3. **In-memory cache strategy.** LRU? TTL-based? All-in-memory until RAM cap? Pick the simplest that meets the latency target.
4. **`scip-typescript` distribution.** Pinned version in code? Auto-update? Manual install? Lean toward "auto-download a pinned version on first use, never auto-update without user consent."
5. **Per-repo or global config?** YAML config in `~/.scry/config.yaml` for daemon settings, plus `.scryignore` per-repo for ignore patterns. Same model as `.gitignore`.
6. **What does `scry symbols src/foo.ts` return for a file with 500 symbols?** All of them with `--limit 1000`, paginate above. Document the default.
7. **Test fixtures.** Use a small synthetic TS repo for fast unit tests, plus integration tests against `microsoft/vscode` (large real repo) for accuracy benchmarks. Don't hit the network in CI.
8. **How to handle indexer failures.** scip-typescript can fail on broken TypeScript. Fall back to tree-sitter-only indexing? Skip the file? Mark the repo as partially indexed? Pick one and document.
9. **Migration story for index format changes.** The BadgerDB schema will evolve. Reindex from scratch on schema bump, or write migrations? v1 = reindex from scratch, document loudly when it happens.
10. **Daemon shutdown grace.** What does `scry stop` do if there's a query in flight? 5-second grace period, then SIGKILL. Standard.

---

## 16. What this spec deliberately excludes

If the build agent is tempted to add any of these in v1, **don't**:

- A web UI (CLI is enough)
- Editor integrations beyond optional LSP mode in P3
- Code generation
- Automated refactoring
- Multi-user / shared daemons
- Cloud sync or backup
- Telemetry of any kind
- A plugin system / scripting language
- A query DSL beyond the cobra subcommands
- Real-time autocomplete for human typing
- Diagnostics / linting / formatting
- Cross-repo references in v1 (defer to v2)

These are all reasonable to add later. None of them belong in v1.

---

## 17. First commit checklist

When the build agent picks this up, the first PR should include:

1. `go.mod` with the dependencies from §7
2. `cmd/scry/main.go` with cobra root + version subcommand
3. `internal/store/` package with BadgerDB-backed index store + tests
4. `internal/sources/scip/parser.go` that parses a `.scip` protobuf file
5. `internal/sources/typescript/indexer.go` that shells out to `scip-typescript` and produces a `.scip`
6. `internal/index/builder.go` that orchestrates: detect language → run indexer → parse SCIP → write to store
7. `internal/query/refs.go` that implements the `refs` query against the store
8. `cmd/scry/init.go` and `cmd/scry/refs.go` subcommands
9. `scry init` and `scry refs <symbol>` working end-to-end against a real TypeScript repo
10. README with the pitch from §1 and a quick-start example
11. `docs/DECISIONS.md` populated with the decisions resolved from §15

Everything else in P0/P1/P2 builds on this foundation.

---

## Appendix A: Why Go

Same answers as trawl:
- Single static binary, cross-compile freely
- Mature daemon patterns: signal handling, fsnotify, Unix sockets, all stdlib or stdlib-adjacent
- Concurrency-first runtime — file watcher + indexer pool + query engine all want to run in parallel
- Fast cold start matters because the CLI is invoked thousands of times per session

Considered and rejected:
- **Rust** — fastest but build times slower, ecosystem for daemons thinner, marginal perf wins for an I/O-bound workload
- **Zig** — too immature, ecosystem too thin
- **TypeScript / Bun** — Bun is fast but the daemon model is awkward, FS access has GC pauses, Go fits better
- **Python** — startup time alone disqualifies it for the CLI use case

---

## Appendix B: Why not just be an LSP server?

LSP is the obvious "use what already exists" choice. Why not just expose scry as an LSP server and call it a day?

Three reasons:

1. **LSP is designed around documents and positions, not symbols.** The protocol's primary unit is "what's at position (line, col) in document URI" — that's exactly the wrong unit for an agent that wants to ask "all references to function X across the whole repo." You'd be twisting LSP into shapes it wasn't designed for.

2. **LSP requires per-language servers and editor configuration.** You'd ship one LSP server per language, and the agent (or editor) would have to know which one to start. scry's whole point is one daemon, all languages, one CLI surface.

3. **LSP startup is slow and stateful.** A typical LSP server takes 2-30 seconds to "warm up" before queries return useful results. scry's daemon is always warm because the indexes are precomputed.

LSP and scry are solving the same data problem with different access patterns. Both can coexist — and in fact, **a v3 idea is to expose scry as an LSP server in addition to the JSON-RPC interface**, so editors and agents both benefit. But that's not v1.

---

## Appendix C: Name collision

A Crystal-language LSP server named `scry` exists at `github.com/crystal-lang-tools/scry`. It is:
- Single-language (Crystal only)
- An LSP server (different protocol surface)
- Largely dormant (low recent activity)

We are using the name anyway because:
- Crystal's mindshare is small and the collision is unlikely to cause user confusion
- The function overlap is incidental — both are "code intelligence" but the scope, audience, and protocol are entirely different
- No major dev tool, npm package, or product uses the name in the AI / agent space

If the collision becomes a real problem (e.g., Crystal scry resurges, or scry gets more popular than expected and users are confused), the rename to `scryd-cli` or `scry-agent` is cheap. Document the original name in CHANGELOG and move on.

The decision is reversible. The name is good. Ship it.

---

## Appendix D: Why SCIP and not LSP-as-data

Sourcegraph's SCIP (SCIP Code Intelligence Protocol) is a Protobuf-based format for storing code intelligence data. It's the format used by Sourcegraph's own code search product, and Sourcegraph maintains indexers for most major languages.

Alternatives considered:
- **LSP-as-data (LSIF)** — older Sourcegraph format, deprecated in favor of SCIP. Don't use.
- **TypeScript Compiler API directly** — works for TS but not for other languages, and reimplements what scip-typescript already does
- **Tree-sitter alone** — too shallow, no semantic info (can't resolve cross-file references or types)
- **Custom indexers per language** — way too much work, reinventing what SCIP indexers already do

SCIP is the right choice because:
- One stable format across all languages
- Sourcegraph maintains the indexers; we just consume them
- Protobuf is fast to parse and well-tooled in Go
- The data model is rich: symbols, references, definitions, types, occurrences, all in one schema

The tradeoff: scry depends on SCIP indexer binaries being downloadable. We mitigate this by pinning versions, verifying SHA256, and caching forever in `~/.scry/bin/`.
