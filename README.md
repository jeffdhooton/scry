# scry

**Code intelligence daemon for AI agents.** Pre-computes a semantic index of every repo you work in (symbols, references, definitions, call graphs, implementations) and exposes it as a millisecond-latency local API. Replaces the Read+Grep+Glob cycle that eats 30-50% of every Claude Code session.

> **Status:** P0 + P1 shipped, P2 PHP underway. Single static Go binary. TypeScript/JavaScript, Go, and PHP/Laravel indexing, daemon mode with auto-spawn, JSON-RPC over Unix socket, fsnotify watch loop with background reindex, callers/callees/impls, auto-download for `scip-go`, embedded `scip-php` directory tree (no separate install) plus a Laravel-aware non-PSR-4 file walker that recovers ~1300 `::class` refs in `routes/`, `config/`, `migrations/`, and `bootstrap/` per real Laravel codebase. See [`docs/SPEC.md`](docs/SPEC.md) for the full PRD and [`docs/DECISIONS.md`](docs/DECISIONS.md) for the architectural decisions made along the way.

---

## Quick start

```bash
# Build (no CGO; single static binary)
go build -o scry ./cmd/scry

# Install scip-typescript (npm package, no GitHub release assets)
npm i -g @sourcegraph/scip-typescript

# scip-go is auto-downloaded into ~/.scry/bin/ on first use against a Go repo
# (pinned to v0.1.26, SHA256-verified)

# Index a repo. The daemon auto-spawns on first call and stays warm
# until `scry stop` or logout.
cd ~/path/to/some/typescript/repo
scry init

# Find every reference to a symbol
scry refs processOrder

# The same query, pretty-printed for human reading
scry refs processOrder --pretty

# Other queries
scry defs processOrder
scry callers processOrder        # refs with the containing function exposed
scry callees processOrder        # what does processOrder call?
scry impls Repository            # everything that implements an interface

# Daemon control
scry status                      # what repos are indexed?
scry start                       # explicit start (auto-spawned otherwise)
scry stop                        # graceful shutdown, 5s grace, then SIGKILL
```

Output is JSON by default — this tool's primary user is an AI agent. Pass `--pretty` for human reading. All file paths are absolute, all line/column numbers are 1-indexed.

## What works today

| Feature | Status |
|---|---|
| **Languages** | TypeScript, JavaScript, Go, PHP (Laravel-aware) |
| **Daemon** | Auto-spawned on first CLI call, Unix socket at `~/.scry/scryd.sock` |
| **JSON-RPC 2.0** | Newline-delimited over Unix socket; methods mirror CLI subcommands |
| **Queries** | `init`, `refs`, `defs`, `callers`, `callees`, `impls`, `status`, `start`, `stop` |
| **Index store** | BadgerDB per repo at `~/.scry/repos/<sha256[:16]>/`, schema-versioned, reset-on-bump |
| **Watch loop** | fsnotify watcher per indexed repo, 300ms debounce, background full reindex with atomic registry swap |
| **Auto-download** | `scip-go` (pinned, SHA256-verified). `scip-php` is embedded into the scry binary as a vendored directory tree and extracted on first use. `scip-typescript` is still manual (no GitHub release assets — install via `npm i -g @sourcegraph/scip-typescript`) |
| **Call graph** | Built at index time from SCIP `enclosing_range`. Full coverage on TypeScript, partial on Go |
| **Implementations** | Built at index time from SCIP `Relationships.is_implementation` |
| **Laravel non-PSR-4 walker** | After `scip-php` runs, scry walks `routes/`, `config/`, `database/migrations/`, `bootstrap/` for `::class` refs and joins them to scip-php's symbol IDs. ~98% bind rate on real codebases. |
| **Laravel facade resolver** | Hardcoded map of 31 Illuminate facades to their backing manager/contract classes. After scip-php and the walker, every facade method ref (`Auth::user()`, `DB::table()`, ...) gets synthetic edges to the backing class methods (`AuthManager#user`, `Guard#user`, `DatabaseManager#table`, `Connection#table`). 5129 edges synthesized on hoopless_crm. |
| **Laravel view + config string-ref walker** | Walks every project `.php` file for `view('foo.bar')` and `config('foo.bar')` calls and emits synthetic ref edges to `resources/views/foo/bar.blade.php` and `config/foo.php#bar` symbols. `scry refs services.dataforseo.login` returns every config-call site with file:line and context. 7 view + 280 config refs on hoopless_crm. |
| **External symbol synthesis** | The SCIP parser synthesizes `SymbolRecord`s for any occurrence whose symbol id wasn't declared as `SymbolInformation` in any document. Closes a general gap where vendor / framework / stdlib references were unqueryable by name (`scry refs DB` previously returned 0; now returns the facade symbol with all its occurrences). |

Real-world numbers (measured against `~/herd/advocates`, 400 TS files / 55k LOC):

| Metric | Target | Actual |
|---|---|---|
| Daemon cold spawn (CLI exits, daemon listening) | <500ms | ~17ms |
| `scry refs <symbol>` wall-clock end-to-end (warm) | <10ms p50 | 6-7ms |
| Cold index build, 100k-LOC TS repo | <60s | 9.9s |
| File-edit → query reflects new state | <200ms (spec) | ~600ms small repo / ~10s on advocates (see [§Known limitations](#known-limitations)) |
| Query unavailability during a watcher reindex | (was ~3-15s) | 12ms swap; queries throughout the rebuild succeed |

## Known limitations

- **`scip-typescript` requires manual install.** It's an npm package; the GitHub releases page has no asset binaries to auto-download. Workaround: `npm i -g @sourcegraph/scip-typescript`. We'll revisit if/when an alternative distribution appears.
- **Vue Single File Components are not indexed.** scip-typescript only walks `.ts`/`.tsx` files. For Inertia/Vue stacks like `~/herd/advocates`, this means refs from Vue templates (`<script>` blocks calling composables) don't show up. Fix would require pre-extracting `<script>` content into virtual TS files before invoking scip-typescript.
- **Symbol kind always reports `UnspecifiedKind`.** scip-typescript v0.4.0 doesn't populate `SymbolInformation.Kind`. We surface what's there.
- ~~**Reindex window blocks queries.**~~ **Fixed.** Watcher reindexes now use `index.BuildIntoTemp` to write into `<storage>/index.db.next/` while the live store keeps serving. `Registry.SwapNext` performs a single ~12ms close+rename+open dance at the end. Measured on hoopless_crm: 1449 successful queries during a full 48s reindex with zero failures (slowest single query 84ms).
- **`<200ms` incremental update is unreachable.** The spec target assumed single-file SCIP indexing exists. It doesn't — `scip-typescript` and `scip-go` are project-wide, type-resolution-driven, and offer no `--single-file` mode. Realistic numbers: ~600ms for a tiny project, ~3s for `trawl`-class, ~10-15s for advocates-class. The long-term answer is a tree-sitter overlay for the 95% of queries where syntactic precision is good enough.
- **`scip-go` `enclosing_range` coverage is partial.** Means `containing_symbol` and `callees` are best-effort on Go (we got 197 call edges on trawl, not zero, but coverage is incomplete). TypeScript is full coverage.
- **PHP P2 is feature-complete:** all four post-processors from the calibration are shipped — non-PSR-4 file walker, facade resolver, view template ref, config key ref. See [`docs/PHP_CALIBRATION.md`](docs/PHP_CALIBRATION.md) for the original gap analysis and [`docs/DECISIONS.md`](docs/DECISIONS.md) for why scry ships scip-php as an embedded directory tree (not a PHAR — php-scoper choked on PHP 8.4 keyword shims).

## Architecture in one diagram

```
┌──────────────────────────────────────────────────────────────┐
│                        scry CLI                              │
│  scry refs | defs | callers | callees | impls | status ...  │
└────────────────────────┬─────────────────────────────────────┘
                         │ JSON-RPC 2.0 (newline-delimited JSON)
                         │ ~/.scry/scryd.sock
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                    scry start --foreground                   │
│   ┌────────────────────────────────────────────────────┐    │
│   │            JSON-RPC dispatcher (rpc.Server)        │    │
│   └─────────────────────┬──────────────────────────────┘    │
│                         │                                    │
│   ┌─────────────────────▼──────────────────────────────┐    │
│   │              Query Engine (internal/query)          │    │
│   │   refs | defs | callers | callees | impls          │    │
│   └─────────┬───────────────────────────┬──────────────┘    │
│             │                           │                    │
│   ┌─────────▼─────────┐    ┌───────────▼───────────┐        │
│   │   Store Registry  │    │   File Watcher        │        │
│   │   (one BadgerDB   │◀───│   (fsnotify, 300ms    │        │
│   │    per repo)      │    │    debounce, full     │        │
│   └─────────▲─────────┘    │    reindex on change) │        │
│             │              └───────────────────────┘        │
│   ┌─────────┴─────────────────────────────────────┐         │
│   │           Index Builder                       │         │
│   │   ┌─────────────┐ ┌─────────┐ ┌─────────┐    │         │
│   │   │ scip-ts     │ │ scip-go │ │ scip-php │   │         │
│   │   │ (npm)       │ │ (auto)  │ │ (P2)     │   │         │
│   │   └─────────────┘ └─────────┘ └─────────┘    │         │
│   │   ┌─────────────────────────────────────┐    │         │
│   │   │   SCIP parser (scip-code/scip       │    │         │
│   │   │   bindings) → BadgerDB writer       │    │         │
│   │   └─────────────────────────────────────┘    │         │
│   └─────────────────────────────────────────────────┘         │
└──────────────────────────────────────────────────────────────┘
```

## Layout

```
scry/
├── cmd/scry/                  # cobra CLI; one binary, daemon and client
│   ├── main.go                # root command, version, subcommand wiring
│   ├── daemon.go              # client-side auto-spawn helpers
│   ├── start.go               # `scry start [--foreground]`
│   ├── stop.go                # `scry stop` (RPC + SIGTERM + SIGKILL grace)
│   ├── init.go                # `scry init` — runs through daemon
│   ├── refs.go                # `scry refs` / `scry defs`
│   ├── graph.go               # `scry callers` / `callees` / `impls`
│   └── status.go              # `scry status`
├── internal/
│   ├── rpc/                   # JSON-RPC 2.0 over Unix socket (server + client)
│   ├── daemon/                # daemon lifecycle, registry, watcher, methods
│   │   ├── daemon.go          # Run, signals, PID file, socket
│   │   ├── registry.go        # per-repo BadgerDB cache
│   │   ├── methods.go         # RPC handlers wired to internal/query
│   │   ├── watch.go           # fsnotify per-repo, debounced reindex
│   │   ├── bootstrap.go       # bootstrap watchers from ~/.scry/repos
│   │   └── rlimit.go          # bump RLIMIT_NOFILE on macOS
│   ├── store/                 # BadgerDB-backed index store + tests
│   ├── sources/
│   │   ├── scip/              # SCIP protobuf parser (streaming)
│   │   ├── typescript/        # scip-typescript shellout
│   │   ├── golang/            # scip-go shellout (with auto-download)
│   │   └── php/               # embedded scip-php tree + Laravel non-PSR-4 walker
│   ├── index/                 # build pipeline: detect → run → parse → store
│   ├── query/                 # refs, defs, callers, callees, impls
│   └── install/               # pinned indexer auto-download with SHA256
└── docs/
    ├── SPEC.md                # original PRD (don't edit; this is the source of truth)
    ├── DECISIONS.md           # architectural choices made along the way
    └── PHP_CALIBRATION.md     # day-1 PHP/Laravel feasibility report
```

## Why a daemon

Pre-computation is the entire game. A typical 100k-LOC TypeScript repo has thousands of symbols and tens of thousands of references. Computing those on every query (LSP-style) is impossibly slow — 5 to 30 seconds. Computing them once at index time, storing them in an embedded KV, and querying via index lookup is single-digit milliseconds. The daemon model is required because the cost of building the index has to be amortized across thousands of queries, but only if the index *stays warm* between calls.

## Why Go

Same answers as `~/workspace/trawl`: single static binary, fast cold start, mature ecosystem for daemon patterns (signal handling, fsnotify, Unix sockets), and the file-watching + indexing workload is concurrency-bound rather than CPU-bound. Reuses trawl's tech-stack decisions wholesale (BadgerDB, cobra, zerolog) so the operational story matches.

## Sibling project

[trawl](file:///Users/jhoot/workspace/trawl) — agent-first web scraping, same architectural philosophy. scry borrows trawl's tech-stack decisions wholesale.
