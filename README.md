# scry

**Code intelligence daemon for AI agents.** Pre-computes a semantic index of every repo you work in (symbols, references, definitions, call graphs, implementations) and exposes it as a millisecond-latency local API. Replaces the Read+Grep+Glob cycle that eats 30-50% of every Claude Code session.

> **Status:** P0 + P1 shipped. Single static Go binary. TypeScript/JavaScript and Go indexing, daemon mode with auto-spawn, JSON-RPC over Unix socket, fsnotify watch loop with background reindex, callers/callees/impls, auto-download for `scip-go`. PHP support is the next milestone (P2). See [`docs/SPEC.md`](docs/SPEC.md) for the full PRD and [`docs/DECISIONS.md`](docs/DECISIONS.md) for the architectural decisions made along the way.

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
| **Languages** | TypeScript, JavaScript, Go |
| **Daemon** | Auto-spawned on first CLI call, Unix socket at `~/.scry/scryd.sock` |
| **JSON-RPC 2.0** | Newline-delimited over Unix socket; methods mirror CLI subcommands |
| **Queries** | `init`, `refs`, `defs`, `callers`, `callees`, `impls`, `status`, `start`, `stop` |
| **Index store** | BadgerDB per repo at `~/.scry/repos/<sha256[:16]>/`, schema-versioned, reset-on-bump |
| **Watch loop** | fsnotify watcher per indexed repo, 300ms debounce, background full reindex with atomic registry swap |
| **Auto-download** | scip-go (pinned, SHA256-verified). scip-typescript is still manual (no GitHub release assets) |
| **Call graph** | Built at index time from SCIP `enclosing_range`. Full coverage on TypeScript, partial on Go |
| **Implementations** | Built at index time from SCIP `Relationships.is_implementation` |

Real-world numbers (measured against `~/herd/advocates`, 400 TS files / 55k LOC):

| Metric | Target | Actual |
|---|---|---|
| Daemon cold spawn (CLI exits, daemon listening) | <500ms | ~17ms |
| `scry refs <symbol>` wall-clock end-to-end (warm) | <10ms p50 | 6-7ms |
| Cold index build, 100k-LOC TS repo | <60s | 9.9s |
| File-edit → query reflects new state | <200ms (spec) | ~600ms small repo / ~10s on advocates (see [§Known limitations](#known-limitations)) |

## Known limitations

- **`scip-typescript` requires manual install.** It's an npm package; the GitHub releases page has no asset binaries to auto-download. Workaround: `npm i -g @sourcegraph/scip-typescript`. We'll revisit if/when an alternative distribution appears.
- **Vue Single File Components are not indexed.** scip-typescript only walks `.ts`/`.tsx` files. For Inertia/Vue stacks like `~/herd/advocates`, this means refs from Vue templates (`<script>` blocks calling composables) don't show up. Fix would require pre-extracting `<script>` content into virtual TS files before invoking scip-typescript.
- **Symbol kind always reports `UnspecifiedKind`.** scip-typescript v0.4.0 doesn't populate `SymbolInformation.Kind`. We surface what's there.
- **Reindex window blocks queries.** Background reindex takes ~3-15s depending on repo size. During that window queries against the same repo return "not indexed yet" because BadgerDB takes an exclusive directory lock and the builder needs it. Documented in `internal/daemon/watch.go`. The fix (build into temp dir + atomic rename) is deferred until measurement shows the gap matters.
- **`<200ms` incremental update is unreachable.** The spec target assumed single-file SCIP indexing exists. It doesn't — `scip-typescript` and `scip-go` are project-wide, type-resolution-driven, and offer no `--single-file` mode. Realistic numbers: ~600ms for a tiny project, ~3s for `trawl`-class, ~10-15s for advocates-class. The long-term answer is a tree-sitter overlay for the 95% of queries where syntactic precision is good enough.
- **`scip-go` `enclosing_range` coverage is partial.** Means `containing_symbol` and `callees` are best-effort on Go (we got 197 call edges on trawl, not zero, but coverage is incomplete). TypeScript is full coverage.
- **PHP is not supported yet.** P2 work. The day-1 calibration in [`docs/PHP_CALIBRATION.md`](docs/PHP_CALIBRATION.md) found that `scip-php` works but needs vendoring as a PHAR (Packagist v0.0.2 is broken on PHP 8.4) and that the biggest Laravel-shaped gap is non-PSR-4 file walking (routes/, migrations/, config/) — `routes/web.php` alone has 1168 `::class` references that the SCIP indexer doesn't see.

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
│   │   └── golang/            # scip-go shellout (with auto-download)
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
