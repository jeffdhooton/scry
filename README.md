# scry

**Code intelligence daemon for AI agents.** Pre-computes a semantic index of every repo you work in (symbols, references, definitions, call graphs, implementations) and exposes it as a millisecond-latency local API. Replaces the Read+Grep+Glob cycle that eats 30-50% of every Claude Code session.

> **Status:** P0 + P1 shipped, P2 PHP underway. Single static Go binary. TypeScript/JavaScript, Go, and PHP/Laravel indexing, daemon mode with auto-spawn, JSON-RPC over Unix socket, fsnotify watch loop with background reindex, callers/callees/impls, auto-download for `scip-go`, embedded `scip-php` directory tree (no separate install) plus a Laravel-aware non-PSR-4 file walker that recovers ~1300 `::class` refs in `routes/`, `config/`, `migrations/`, and `bootstrap/` per real Laravel codebase. See [`docs/SPEC.md`](docs/SPEC.md) for the full PRD and [`docs/DECISIONS.md`](docs/DECISIONS.md) for the architectural decisions made along the way.

---

## Install

**One-liner** (darwin / linux, amd64 / arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/jeffdhooton/scry/main/scripts/install.sh | sh
```

Drops the binary at `~/.local/bin/scry`. Customize with `INSTALL_DIR=/usr/local/bin` or pin a version with `SCRY_VERSION=v0.1.0`.

**From source** (requires Go 1.23+):

```bash
go install github.com/jeffdhooton/scry/cmd/scry@latest
```

**Keeping it fresh:** `scry upgrade` downloads the latest release and replaces the running binary in place. `scry upgrade --check` just prints what's available. See [`docs/RELEASING.md`](docs/RELEASING.md) if you're publishing a new version.

**Once installed**, run the post-install setup and verification:

```bash
scry setup        # installs the Claude Code skill + MCP server registration
scry doctor       # checks every prereq and prints a green/yellow/red checklist
```

`scry doctor` tells you exactly what's missing (PHP for PHP repos, `scip-typescript` for TS repos, `claude` CLI, etc.) and how to fix each one. Re-run it any time the setup feels off.

`scip-typescript` and `scip-python` are the two indexers that aren't auto-bundled — install separately if you need them:

```bash
npm i -g @sourcegraph/scip-typescript   # for TypeScript / JavaScript repos
npm i -g @sourcegraph/scip-python        # for Python repos (requires Node ≥16)
```

`scip-go` auto-downloads into `~/.scry/bin/` on first use against a Go repo (pinned, SHA256-verified). `scip-php` is embedded in the scry binary and extracted on first PHP repo.

**Python gotcha**: `scip-python` 0.6.6's bundled Pyright only recognizes Python 3.10–3.13. If your default `python3` is 3.14+ (common on bleeding-edge Homebrew), scry automatically shims `scip-python` to use the first compatible interpreter it finds on PATH (`python3.13`, `python3.12`, `python3.11`, then `python3.10`). If none are installed, `scry doctor` flags it with install instructions. Activate a venv before `scry init` if you want external imports (third-party packages) resolved — scry honors `$VIRTUAL_ENV`, `.venv/`, `venv/`, and `env/` automatically.

### Install the full agent tool suite

scry is part of a suite of local-first dev tools for AI agents. Install and register everything in one shot:

```bash
# Install all four tools
go install github.com/jeffdhooton/scry/cmd/scry@latest
go install github.com/jeffdhooton/flume/cmd/flume@latest
go install github.com/jeffdhooton/tome/cmd/tome@latest
go install github.com/jeffdhooton/lore/cmd/lore@latest

# Register each with Claude Code (one-time, idempotent)
scry setup
flume setup
tome setup
lore setup

# Verify everything is wired up
scry doctor
flume doctor
tome doctor
lore doctor
```

Each tool auto-spawns its daemon on first use. After setup, Claude Code routes queries automatically — symbol lookups go to scry, schema questions go to tome, git history goes to lore, and runtime debugging goes to flume. No manual intervention needed.

| Tool | What it gives your agent |
|------|------------------------|
| **scry** | "Where is this function used?" — in 3ms instead of 30s of grepping |
| **flume** | "What happened on the last request?" — instead of adding print statements |
| **tome** | "What columns does users have?" — in 1 call instead of 3-6 file reads |
| **lore** | "Who changed this and why?" — in 1 call instead of 5 git commands |

## Quick start

```bash
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
scry tests processOrder          # is this function covered by tests?

# Daemon control
scry status                      # what repos are indexed?
scry start                       # explicit start (auto-spawned otherwise)
scry stop                        # graceful shutdown, 5s grace, then SIGKILL

# Claude Code integration
scry setup                       # install skill + MCP server (one-shot, idempotent)
scry mcp                         # stdio MCP server (launched by Claude Code, not humans)
```

Output is JSON by default — this tool's primary user is an AI agent. Pass `--pretty` for human reading. All file paths are absolute, all line/column numbers are 1-indexed.

## What works today

| Feature | Status |
|---|---|
| **Languages** | TypeScript, JavaScript, Go, PHP (Laravel-aware), Python |
| **Daemon** | Auto-spawned on first CLI call, Unix socket at `~/.scry/scryd.sock` |
| **JSON-RPC 2.0** | Newline-delimited over Unix socket; methods mirror CLI subcommands |
| **Queries** | `init`, `refs`, `defs`, `callers`, `callees`, `impls`, `tests`, `status`, `start`, `stop` |
| **Index store** | BadgerDB per repo at `~/.scry/repos/<sha256[:16]>/`, schema-versioned, reset-on-bump |
| **Watch loop** | fsnotify watcher per indexed repo, 300ms debounce, background full reindex with atomic registry swap |
| **Auto-download** | `scip-go` (pinned, SHA256-verified). `scip-php` is embedded into the scry binary as a vendored directory tree and extracted on first use. `scip-typescript` is still manual (no GitHub release assets — install via `npm i -g @sourcegraph/scip-typescript`) |
| **Call graph** | Built at index time from SCIP `enclosing_range`. Full coverage on TypeScript, partial on Go |
| **Implementations** | Built at index time from SCIP `Relationships.is_implementation` |
| **Laravel non-PSR-4 walker** | After `scip-php` runs, scry walks `routes/`, `config/`, `database/migrations/`, `bootstrap/` for `::class` refs and joins them to scip-php's symbol IDs. ~98% bind rate on real codebases. |
| **Laravel facade resolver** | Hardcoded map of 31 Illuminate facades to their backing manager/contract classes. After scip-php and the walker, every facade method ref (`Auth::user()`, `DB::table()`, ...) gets synthetic edges to the backing class methods (`AuthManager#user`, `Guard#user`, `DatabaseManager#table`, `Connection#table`). 5129 edges synthesized on hoopless_crm. |
| **Laravel view + config string-ref walker** | Walks every project `.php` file for `view('foo.bar')` and `config('foo.bar')` calls and emits synthetic ref edges to `resources/views/foo/bar.blade.php` and `config/foo.php#bar` symbols. `scry refs services.dataforseo.login` returns every config-call site with file:line and context. 7 view + 280 config refs on hoopless_crm. |
| **Claude Code integration** | `scry setup` installs a skill at `~/.claude/skills/scry/SKILL.md` (routing instructions for Claude) plus registers scry as a User-scope MCP server by shelling out to `claude mcp add --scope user --transport stdio scry -- <scry-bin> mcp`. Seven tools exposed: `scry_refs`, `scry_defs`, `scry_callers`, `scry_callees`, `scry_impls`, `scry_tests`, `scry_status`. Claude routes symbol queries through scry and falls back to Grep for string/pattern searches. Idempotent; re-runs after a scry upgrade refresh the registered binary path automatically. MCP call logging to `~/.scry/logs/mcp-calls.jsonl` (tool, symbol, latency, result count per call). |
| **Test coverage index** | Auto-detects coverage files (`cover.out`, `coverage-final.json`, `clover.xml`, `coverage.json`) during `scry init`, parses them, and joins covered lines against symbol definitions. `scry tests <symbol>` returns whether a function is covered by tests and with what hit count. Supports Go coverprofile, Istanbul/c8 JSON (vitest/jest), Clover XML (PHPUnit), and Python coverage.json. No coverage files = silent no-op. |
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
│   ├── tests.go               # `scry tests` (coverage query)
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
│   │   ├── php/               # embedded scip-php tree + Laravel non-PSR-4 walker
│   │   ├── python/            # scip-python shellout + PATH shim for Pyright version pinning
│   │   └── coverage/          # coverage file parsers (Go, Istanbul, Clover, Python) + join
│   ├── index/                 # build pipeline: detect → run → parse → store
│   ├── query/                 # refs, defs, callers, callees, impls, tests (coverage)
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

## Part of the agent tool suite

A collection of local-first, single-binary dev tools built for AI coding agents. All share the same architecture: Go, no CGO, BadgerDB, daemon over Unix socket, MCP stdio, millisecond-latency queries. Free, local-only, no cloud.

| Tool | What it does | Status |
|------|-------------|--------|
| **[scry](https://github.com/jeffdhooton/scry)** | Code intelligence — symbols, refs, call graphs, impls, test coverage | Shipped |
| **[flume](https://github.com/jeffdhooton/flume)** | Runtime visibility — HTTP requests, SQL queries, exceptions from dev servers | P0 shipped |
| **[tome](https://github.com/jeffdhooton/tome)** | Schema awareness — DB schemas, API shapes, ORM models, enums | P0 shipped |
| **[lore](https://github.com/jeffdhooton/lore)** | Git intelligence — blame, history, co-change patterns, hotspots | P0 shipped |
