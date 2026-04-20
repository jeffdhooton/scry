# scry

**Unified code intelligence daemon for AI agents.** Pre-computes semantic indexes across five domains — code symbols, git history, database schemas, HTTP traffic, and a cross-domain graph — and exposes them as millisecond-latency local queries. One binary, one daemon, one MCP server. Replaces scry + tome + flume + lore.

> **Status:** Unified binary shipped. TypeScript/JavaScript, Go, PHP/Laravel, Python indexing. Git intelligence (blame, history, cochange, hotspots, contributors). Database schema introspection (MySQL, PostgreSQL). HTTP reverse proxy capture. Unified cross-domain graph with community detection. 23 MCP tools across 5 domains. See [`docs/SPEC.md`](docs/SPEC.md) for the original PRD and [`docs/DECISIONS.md`](docs/DECISIONS.md) for architectural decisions.

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
scry setup        # installs skill + MCP server, removes old tome/flume/lore registrations
scry doctor       # checks every prereq and prints a green/yellow/red checklist
```

`scry doctor` tells you exactly what's missing (PHP for PHP repos, `scip-typescript` for TS repos, `claude` CLI, stale old MCP servers, etc.) and how to fix each one.

`scip-typescript` and `scip-python` are the two indexers that aren't auto-bundled — install separately if you need them:

```bash
npm i -g @sourcegraph/scip-typescript   # for TypeScript / JavaScript repos
npm i -g @sourcegraph/scip-python        # for Python repos (requires Node >=16)
```

`scip-go` auto-downloads into `~/.scry/bin/` on first use against a Go repo (pinned, SHA256-verified). `scip-php` is embedded in the scry binary and extracted on first PHP repo.

**Python gotcha**: `scip-python` 0.6.6's bundled Pyright only recognizes Python 3.10-3.13. If your default `python3` is 3.14+ (common on bleeding-edge Homebrew), scry automatically shims `scip-python` to use the first compatible interpreter it finds on PATH (`python3.13`, `python3.12`, `python3.11`, then `python3.10`).

### Migrating from separate tools

If you previously used tome, flume, and lore as separate binaries, `scry setup` automatically removes their MCP registrations. You can also do it manually:

```bash
claude mcp remove tome
claude mcp remove flume
claude mcp remove lore
scry setup        # re-registers the unified scry MCP server
```

The old binaries can be deleted — all functionality is now in `scry`.

## Quick start

```bash
# Index a repo. The daemon auto-spawns on first call.
cd ~/path/to/your/repo
scry init                    # code symbols (TS, Go, PHP, Python)
scry init --git              # git history (blame, cochange, hotspots)
scry init --all              # everything: code + git + schema (auto-detects DSN)

# Code intelligence
scry refs processOrder       # every reference
scry defs processOrder       # every definition
scry callers processOrder    # call sites with containing function
scry callees processOrder    # outgoing calls
scry impls Repository        # implementors of an interface
scry tests processOrder      # test coverage status

# Git intelligence
scry blame src/handler.go    # structured blame
scry history src/handler.go  # recent commits
scry cochange src/handler.go # co-changed files
scry hotspots                # most churned files
scry contributors            # main authors
scry intent src/handler.go --line 42  # why was this line written?

# Schema (requires --schema or --all during init)
scry describe users          # table structure
scry relations orders        # foreign keys
scry schema-search email     # find tables/columns
scry enums                   # enum types and values

# HTTP capture
scry proxy start --port 8089 --target localhost:8000
# Point your app at localhost:8089 instead of :8000
scry requests --path /api    # list captured traffic
scry request <id>            # full request/response
scry proxy stop

# Cross-domain graph
scry graph build             # build from all indexed domains
scry graph report            # architectural summary: god nodes, communities
scry graph query UserService # find nodes by name
scry graph path --from UserService --to "users table"  # shortest path

# Infrastructure
scry status                  # what repos and domains are indexed?
scry start                   # explicit start (auto-spawned otherwise)
scry stop                    # graceful shutdown
scry setup                   # install skill + MCP server
scry doctor                  # health check
```

Output is JSON by default — this tool's primary user is an AI agent. Pass `--pretty` for human reading.

## What works today

| Feature | Status |
|---|---|
| **Code languages** | TypeScript, JavaScript, Go, PHP (Laravel-aware), Python |
| **Git intelligence** | Blame, history, cochange, hotspots, contributors, intent |
| **Schema** | MySQL and PostgreSQL introspection (tables, columns, FKs, enums) |
| **HTTP capture** | Reverse proxy with request/response recording (30-min TTL) |
| **Unified graph** | Cross-domain nodes and edges, Louvain community detection, BFS path finding |
| **Daemon** | Auto-spawned, Unix socket at `~/.scry/scryd.sock` |
| **JSON-RPC 2.0** | Newline-delimited over Unix socket; methods across 5 domains |
| **MCP server** | 23 tools: 7 code + 6 git + 4 schema + 3 HTTP + 3 graph |
| **Watch loop** | fsnotify per indexed repo, 300ms debounce, atomic reindex swap |
| **Index store** | BadgerDB per domain per repo at `~/.scry/repos/<hash>/` |
| **Auto-download** | `scip-go` (pinned, SHA256-verified). `scip-php` embedded in binary. |
| **Call graph** | Built at index time from SCIP `enclosing_range`. Full on TS, partial on Go. |
| **Implementations** | Built at index time from SCIP `Relationships.is_implementation` |
| **Laravel support** | Non-PSR-4 walker, facade resolver (31 facades), view + config string-refs |
| **Test coverage** | Auto-detects `cover.out`, Istanbul JSON, Clover XML, Python `coverage.json` |
| **Claude Code integration** | Skill routing + 23 MCP tools. `scry setup` handles everything. |

Real-world numbers (measured against `~/herd/advocates`, 400 TS files / 55k LOC):

| Metric | Target | Actual |
|---|---|---|
| Daemon cold spawn | <500ms | ~17ms |
| `scry refs <symbol>` end-to-end (warm) | <10ms p50 | 6-7ms |
| Cold index build, 100k-LOC TS repo | <60s | 9.9s |
| Query unavailability during reindex | (was ~3-15s) | 12ms swap |

## MCP tools reference

All tools use the `scry_` prefix. Registered as a single MCP server via `scry setup`.

| Domain | Tools |
|--------|-------|
| **Code** | `scry_refs`, `scry_defs`, `scry_callers`, `scry_callees`, `scry_impls`, `scry_tests`, `scry_status` |
| **Git** | `scry_blame`, `scry_history`, `scry_cochange`, `scry_hotspots`, `scry_contributors`, `scry_intent` |
| **Schema** | `scry_describe`, `scry_relations`, `scry_schema_search`, `scry_enums` |
| **HTTP** | `scry_requests`, `scry_request`, `scry_http_status` |
| **Graph** | `scry_graph_query`, `scry_graph_path`, `scry_graph_report` |

## Claude Code integration

scry integrates with Claude Code at three levels: MCP tools, a routing skill, and PreToolUse hooks. `scry setup` handles the first two automatically. The hooks are optional but strongly recommended — they're what makes Claude *prefer* scry over raw Grep/git without you having to ask.

### What `scry setup` does

```bash
scry setup
```

1. **Registers the MCP server** — runs `claude mcp add --scope user --transport stdio scry -- <binary> mcp`, making all 23 `scry_*` tools available in every Claude Code session.
2. **Installs the routing skill** — writes `~/.claude/skills/scry/SKILL.md`, a detailed routing table that teaches Claude when to reach for scry vs Grep vs Read. Covers all five domains with example queries.
3. **Cleans up legacy tools** — removes old `tome`, `flume`, `lore` MCP registrations if present.

Verify with:

```bash
claude mcp get scry              # should show Status: ✓ Connected
scry doctor                      # full health check
```

### PreToolUse hooks (recommended)

The MCP tools and skill give Claude the *ability* to use scry, but Claude will still sometimes reach for Grep or `git log` out of habit. PreToolUse hooks intercept those calls and nudge Claude toward scry equivalents — or tell you when a repo isn't indexed yet.

Add these to your `~/.claude/settings.json` under the `hooks.PreToolUse` array:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          {
            "command": "/path/to/scry hook pre-search",
            "statusMessage": "Checking scry index...",
            "type": "command"
          }
        ],
        "matcher": "Grep|Glob"
      },
      {
        "hooks": [
          {
            "command": "/path/to/scry hook pre-git",
            "statusMessage": "Checking scry index...",
            "type": "command"
          }
        ],
        "matcher": "Bash"
      }
    ]
  }
}
```

Replace `/path/to/scry` with the actual binary path (e.g. `$HOME/.local/bin/scry` or the output of `which scry`).

**What each hook does:**

| Hook | Fires on | Behavior |
|------|----------|----------|
| `pre-search` | Every `Grep` or `Glob` call | If the pattern looks like a symbol name (not a regex/glob) and the repo is indexed, nudges Claude to use `scry_refs`/`scry_defs` instead. If a graph is available, also mentions `scry_graph_report`, `scry_graph_query`, and `scry_graph_path`. |
| `pre-git` | Every `Bash` call | If the command is `git blame`, `git log`, `git shortlog`, or `git diff --stat` and git history is indexed, nudges Claude toward `scry_blame`, `scry_history`, `scry_contributors`, `scry_hotspots`, or `scry_cochange`. |

**Unindexed repo behavior:** Both hooks detect when the current repo has no scry index and return a message suggesting `scry init --all`. Claude sees this in its context and will relay the suggestion to you. No silent failures.

### Monitoring usage

MCP call logging writes to `~/.scry/logs/mcp-calls.jsonl`. Every scry MCP tool invocation is recorded with timestamp, tool name, repo, result count, and latency:

```bash
# See what tools Claude is actually calling
cat ~/.scry/logs/mcp-calls.jsonl | jq .

# Count tool usage by name
cat ~/.scry/logs/mcp-calls.jsonl | jq -r .tool | sort | uniq -c | sort -rn

# Check if graph tools are being used
grep graph ~/.scry/logs/mcp-calls.jsonl | jq .
```

If you see zero graph entries after working in an indexed repo, Claude may not be reaching for the graph tools. The `pre-search` hook's graph nudge should help, but you can also explicitly ask Claude to "show me the graph report" or "what connects X to Y" to prime the behavior.

### Global CLAUDE.md guidance

For maximum effect, add a line to your `~/.claude/CLAUDE.md` (or project-level `CLAUDE.md`) that reinforces the preference:

```markdown
Prefer MCP tools over raw alternatives. Use scry for symbol lookups (scry_refs, scry_defs,
scry_callers, scry_callees, scry_impls), git history (scry_blame, scry_history, scry_cochange,
scry_hotspots, scry_contributors, scry_intent), database schemas (scry_describe, scry_relations,
scry_schema_search, scry_enums), HTTP traffic (scry_requests, scry_request), and cross-domain
graph queries (scry_graph_query, scry_graph_path, scry_graph_report).
```

### Full integration checklist

```bash
scry setup                       # MCP server + skill
scry doctor                      # verify prereqs
scry init --all                  # index current repo
# Add hooks to ~/.claude/settings.json (see above)
# Add guidance to ~/.claude/CLAUDE.md (see above)
# Verify: work in Claude Code, check ~/.scry/logs/mcp-calls.jsonl
```

## Known limitations

- **`scip-typescript` requires manual install.** It's an npm package; no auto-download available. Workaround: `npm i -g @sourcegraph/scip-typescript`.
- **Vue Single File Components are not indexed.** scip-typescript only walks `.ts`/`.tsx` files.
- **Symbol kind always reports `UnspecifiedKind`.** scip-typescript v0.4.0 doesn't populate `SymbolInformation.Kind`.
- **`<200ms` incremental update is unreachable.** SCIP indexers are project-wide. Realistic: ~600ms small, ~10s large.
- **`scip-go` `enclosing_range` is partial.** Call graph coverage on Go is best-effort.
- **Graph `queries` edge** (function -> table) is not yet implemented. Currently the graph connects code, git, schema, and HTTP domains via structural edges (calls, implements, changed_with, fk).
- **Schema requires explicit init.** `scry init --schema` or `scry init --all` with a DSN or `.env` file.
- **HTTP proxy is explicit.** `scry proxy start` must be run manually; the daemon doesn't auto-start the proxy.

## Architecture

```
~/.scry/
  scryd.sock                  # one socket, one daemon
  scryd.pid
  repos/<hash>/
    code/index.db             # SCIP symbols, refs, call graph
    git/index.db              # blame, commits, cochange, hotspots
    schema/index.db           # database tables, FKs, enums
    http/                     # captured request/response pairs
    graph/index.db            # unified cross-domain graph
    manifest.json             # per-repo metadata across all domains
```

```
┌────────────────────────────────────────────────────────────────┐
│                         scry CLI                               │
│  refs | defs | blame | describe | requests | graph query ...  │
└───────────────────────────┬────────────────────────────────────┘
                            │ JSON-RPC 2.0 / Unix socket
                            ▼
┌────────────────────────────────────────────────────────────────┐
│                      scry daemon                               │
│   ┌──────────────────────────────────────────────────────┐    │
│   │            JSON-RPC dispatcher (rpc.Server)          │    │
│   │  code.*  git.*  schema.*  http.*  graph.*  ping      │    │
│   └──────────────────────────────────────────────────────┘    │
│                                                                │
│   ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│   │  Code    │ │   Git    │ │  Schema  │ │   HTTP   │        │
│   │ Registry │ │ Registry │ │ Registry │ │  Proxy   │        │
│   └──────────┘ └──────────┘ └──────────┘ └──────────┘        │
│                       ┌──────────┐                             │
│                       │  Graph   │                             │
│                       │ Registry │                             │
│                       └──────────┘                             │
│   ┌──────────────────────────────────────────────────────┐    │
│   │ Index Builders: scip-ts, scip-go, scip-php, scip-py │    │
│   │ Git indexer, Schema introspector, Graph builder      │    │
│   └──────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────────┘
```

## Layout

```
scry/
├── cmd/scry/                  # cobra CLI; one binary
│   ├── main.go                # root command, subcommand wiring
│   ├── init.go                # scry init (code, --git, --schema, --all)
│   ├── refs.go                # refs / defs
│   ├── graph.go               # callers / callees / impls
│   ├── tests.go               # test coverage query
│   ├── blame.go               # git blame/history/cochange/hotspots/contributors/intent
│   ├── schema.go              # describe/relations/schema-search/enums
│   ├── proxy.go               # proxy start/stop, requests, request
│   ├── graphcmd.go            # graph build/query/path/report
│   ├── status.go              # daemon status
│   └── ...                    # start, stop, setup, doctor, upgrade, mcp
├── internal/
│   ├── rpc/                   # JSON-RPC 2.0 server + client
│   ├── daemon/                # daemon lifecycle, registries, methods
│   │   ├── daemon.go          # Run, signals, PID file, socket
│   │   ├── registry.go        # code store registry
│   │   ├── git_registry.go    # git store registry
│   │   ├── schema_registry.go # schema store registry
│   │   ├── graph_methods.go   # graph registry + RPC handlers
│   │   ├── methods.go         # code RPC handlers
│   │   ├── git_methods.go     # git RPC handlers
│   │   ├── schema_methods.go  # schema RPC handlers
│   │   ├── http_methods.go    # HTTP proxy RPC handlers
│   │   └── watch.go           # fsnotify watcher
│   ├── store/                 # code BadgerDB store
│   ├── git/                   # git indexer + store
│   ├── schema/                # schema introspector + store
│   ├── http/                  # HTTP proxy + request store
│   ├── graph/                 # graph builder + query + store
│   ├── mcp/                   # MCP stdio server (23 tools)
│   ├── sources/               # language indexers
│   │   ├── scip/              # SCIP protobuf parser
│   │   ├── typescript/        # scip-typescript
│   │   ├── golang/            # scip-go
│   │   ├── php/               # embedded scip-php + Laravel post-processors
│   │   ├── python/            # scip-python
│   │   └── coverage/          # coverage file parsers
│   ├── index/                 # code build pipeline
│   ├── query/                 # code query engine
│   └── install/               # indexer auto-download
└── docs/
    ├── SPEC.md                # original PRD
    ├── DECISIONS.md           # architectural decisions
    ├── UNIFICATION_SPEC.md    # unification design doc
    └── PHP_CALIBRATION.md     # PHP/Laravel feasibility report
```

## Why a single binary

Four separate tools (scry, tome, flume, lore) shared 90% of their infrastructure: cobra CLI, BadgerDB storage, JSON-RPC 2.0, MCP stdio server, daemon lifecycle. Running four daemons, four sockets, and four MCP servers for one project was wasteful. The unified binary eliminates routing decisions for Claude Code — one MCP server, one tool namespace.

## Author

Built by [Jeff Hooton](https://hooton.codes) · [GitHub](https://github.com/jeffdhooton)
