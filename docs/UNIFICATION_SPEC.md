# Scry Unification Spec

Collapse scry, tome, flume, and lore into a single `scry` binary. One daemon, one MCP server, one install. Then extend with a graph layer that covers what graphify does — minus the Python, minus the LLM cost for structural passes.

## Why

Four tools share 90% of their infrastructure:

| Layer | scry | tome | flume | lore |
|-------|------|------|-------|------|
| CLI | cobra | cobra | cobra | cobra |
| Storage | BadgerDB | BadgerDB | BadgerDB | BadgerDB |
| RPC | JSON-RPC 2.0 / Unix socket | same | same | same |
| MCP | stdio server | same | same | same |
| Daemon | singleton, auto-spawn, PID file | same | same | same |
| Watcher | fsnotify | — | — | fsnotify |

Running four daemons, four sockets, four MCP servers for one project is wasteful. Claude Code has to know which tool to reach for. A unified binary eliminates that routing decision entirely.

## Architecture

### Single daemon

```
~/.scry/
  scryd.sock                  # one socket
  scryd.pid
  scryd.log
  repos/<hash>/
    code/index.db             # SCIP symbols, refs, call graph
    schema/index.db           # database schema cache
    git/index.db              # blame, commits, cochange, hotspots
    http/index.db             # captured request/response pairs
    graph/index.db            # unified graph (phase 2)
    manifest.json             # per-repo metadata across all domains
```

Each domain gets its own BadgerDB directory within the repo store. This keeps key namespaces clean and allows independent rebuilds without touching other indexes.

### Single MCP server

One `scry mcp` command exposes all tools:

```
# Code intelligence (existing)
scry_defs, scry_refs, scry_callers, scry_callees, scry_impls, scry_tests

# Schema (from tome)
scry_describe, scry_relations, scry_schema_search, scry_enums

# HTTP capture (from flume)
scry_requests, scry_request, scry_http_status

# Git intelligence (from lore)
scry_blame, scry_history, scry_cochange, scry_hotspots, scry_contributors, scry_intent

# Graph (new, phase 2)
scry_graph_query, scry_graph_path, scry_graph_report
```

Tool names use `scry_` prefix uniformly. Claude Code sees one MCP server, one tool namespace.

### CLI subcommands

Group by domain using cobra command groups:

```
scry init [path]              # index code (existing)
scry init --schema [dsn]      # index database schema
scry init --git [path]        # index git history
scry init --all [path]        # index everything detected

# Code intelligence
scry defs <symbol>
scry refs <symbol>
scry callers <symbol>
scry callees <symbol>
scry impls <interface>
scry tests <symbol>

# Schema
scry describe <table>
scry relations <table>
scry enums [table.column]
scry schema-search <term>

# HTTP capture
scry proxy start [--port 8089 --target localhost:8000]
scry proxy stop
scry requests [--path /api --method POST --status 4xx]
scry request <id>

# Git intelligence
scry blame <file> [--lines 10-20]
scry history [file] [--limit 50]
scry cochange <file>
scry hotspots
scry contributors [file]
scry intent <file> --line 42

# Graph (phase 2)
scry graph build [path]
scry graph query "question"
scry graph path <nodeA> <nodeB>
scry graph report

# Infrastructure
scry start / stop / status
scry mcp
scry setup
scry doctor
scry upgrade
scry version
```

### Daemon RPC methods

All four domains register methods on the same RPC server:

```
# Code (existing)
init, refs, defs, callers, callees, impls, tests, status

# Schema (new)
schema.init, schema.describe, schema.relations, schema.search,
schema.enums, schema.refresh

# HTTP (new)
http.start, http.stop, http.requests, http.request, http.status

# Git (new)
git.init, git.blame, git.history, git.cochange, git.hotspots,
git.contributors, git.intent

# Graph (phase 2)
graph.build, graph.query, graph.path, graph.report

# Infrastructure
ping, shutdown
```

Method namespacing (`schema.init` vs `init`) prevents collisions. Existing code intelligence methods keep their unprefixed names for backwards compatibility during migration.

## Migration plan

### Phase 0: Shared infrastructure (do first)

Extract the duplicated packages into a shared `internal/platform/` layer:

```
internal/platform/
  rpc/          # JSON-RPC 2.0 server + client (identical in all 4 tools)
  daemon/       # daemon lifecycle: spawn, PID, socket, signal handling
  store/        # BadgerDB open/close/writer helpers
  mcp/          # MCP stdio protocol, tool registration, call logging
  layout/       # ~/.scry/ directory structure
  watch/        # fsnotify debounce + cooldown wrapper
```

Each domain then becomes a thin layer on top:

```
internal/code/      # existing scry: SCIP indexing, symbol queries
internal/schema/    # from tome: introspection, table/FK/enum queries
internal/http/      # from flume: proxy, request capture, queries
internal/git/       # from lore: blame, log, cochange, churn, contrib
internal/graph/     # new: unified graph builder + query engine
```

### Phase 1: Absorb lore

**Why first:** No external dependencies (just git), same watcher pattern as scry, most natural fit. Lore watches `.git/refs/heads/` while scry watches source files — both can run on the same fsnotify instance.

Steps:
1. Copy `lore/internal/index/` → `scry/internal/git/index/`
2. Copy `lore/internal/store/` → `scry/internal/git/store/` (adapt key prefixes)
3. Register git.* RPC methods in daemon
4. Add `scry_blame`, `scry_history`, `scry_cochange`, `scry_hotspots`, `scry_contributors`, `scry_intent` to MCP tool list
5. Add CLI subcommands: `blame`, `history`, `cochange`, `hotspots`, `contributors`, `intent`
6. Extend `scry init` to detect `.git/` and auto-index history
7. Extend watcher to also monitor `.git/refs/heads/` + `.git/HEAD`
8. Update `scry setup` to register all new MCP tools

**Breaking changes:** None to existing scry users. Lore becomes a deprecated alias that prints "use scry instead."

### Phase 2: Absorb tome

Steps:
1. Copy `tome/internal/schema/` → `scry/internal/schema/`
2. Copy `tome/internal/sources/{mysql,postgres}/` → `scry/internal/schema/sources/`
3. Add `go-sql-driver/mysql` and `jackc/pgx/v5` to go.mod
4. Register schema.* RPC methods
5. Add MCP tools and CLI subcommands
6. DSN detection: `scry init --schema` reads `.env` (same logic as tome's `envdetect.go`)
7. `scry init --all` detects `.env` with DATABASE_URL and auto-indexes schema

**Consideration:** mysql and pgx drivers add ~2MB to the binary. Worth it for zero-config schema awareness. Keep them behind build tags if binary size becomes a concern later.

### Phase 3: Absorb flume

Steps:
1. Copy `flume/internal/daemon/proxy.go` → `scry/internal/http/proxy.go`
2. Copy `flume/internal/store/` → `scry/internal/http/store/`
3. Register http.* RPC methods
4. Add MCP tools and CLI subcommands
5. The proxy listener runs inside the scry daemon (new goroutine)
6. `scry proxy start --port 8089 --target localhost:8000` tells daemon to start capturing
7. `scry proxy stop` tears down the listener

**Consideration:** The HTTP proxy is the only domain that runs a long-lived listener inside the daemon (vs. one-shot indexing). Keep the proxy lifecycle independent — starting the daemon doesn't start the proxy. Explicit `scry proxy start` required.

### Phase 4: Graph layer

This is what closes the gap with graphify. Build a unified graph from data that already exists in the other four indexes.

#### What the graph connects

```
Code nodes:   function, class, interface, module (from SCIP index)
Schema nodes: table, column, foreign_key (from schema index)
Git nodes:    commit, author (from git index)
HTTP nodes:   endpoint, request_pattern (from HTTP capture)
```

#### Edge types

| Edge | Source → Target | Derivation |
|------|----------------|------------|
| `calls` | function → function | SCIP call edges (existing) |
| `implements` | class → interface | SCIP impl edges (existing) |
| `imports` | module → module | SCIP references |
| `queries` | function → table | Static analysis: find SQL/ORM references in function bodies |
| `migrates` | commit → table | Git history: commits that touch migration files |
| `authored_by` | function → author | Blame data: primary author of function's line range |
| `changed_with` | file → file | Cochange data (existing) |
| `serves` | function → endpoint | HTTP capture: map handler functions to observed routes |
| `fk` | table → table | Foreign key relationships (existing) |
| `tested_by` | function → function | Coverage data: test → implementation mapping |

#### Graph storage

In-memory NetworkX-style graph isn't viable in Go without CGO. Instead:

```
graph/index.db keys:
  node:<type>:<id>                    → NodeRecord{type, name, file, line, metadata}
  edge:<src_type>:<src_id>:<edge_type>:<dst_id> → EdgeRecord{type, confidence, source_domain}
  community:<id>                      → CommunityRecord{nodes[], label, cohesion_score}
  report:latest                       → GraphReport{god_nodes[], surprising_edges[], questions[]}
```

All edges carry a `confidence` field (1.0 for SCIP-derived, 0.8 for cochange-derived, etc.) and a `source_domain` tag so the provenance is always clear.

#### Community detection

Use a Go implementation of the Leiden algorithm (or Louvain as fallback). Operate on the edge list directly from BadgerDB — no need to load the full graph into memory. Communities identify architectural boundaries: "these functions, tables, and endpoints form a cohesive module."

#### Graph report

`scry graph report` generates a summary equivalent to graphify's GRAPH_REPORT.md:

- **God nodes**: highest-degree nodes across all types
- **Surprising connections**: cross-domain edges with high confidence (e.g., a function that queries a table it has no import path to)
- **Architectural communities**: clusters with human-readable labels
- **Suggested queries**: pre-computed questions the graph can answer

This report is what the MCP tool `scry_graph_report` returns. Claude Code reads it before exploring.

#### What this doesn't do (and why)

- **No LLM semantic pass**: Graphify uses an LLM to extract concepts from markdown/PDFs. We skip this. The structural pass (SCIP + git + schema + HTTP) provides richer, more reliable edges than LLM inference. If doc ingestion becomes valuable later, add it as an optional `scry graph build --docs` flag.
- **No image/video/audio ingestion**: Out of scope. These are documents, not code intelligence.
- **No cross-repo graphs**: Each repo gets its own graph. Cross-repo edges are a future concern.

## MCP tool behavior changes

### scry_graph_query

```json
{
  "name": "scry_graph_query",
  "input": {"query": "what connects the User model to the /api/auth endpoint?"},
  "output": {
    "paths": [
      {
        "nodes": ["User (table)", "AuthController.login (function)", "/api/auth/login (endpoint)"],
        "edges": ["queries", "serves"],
        "confidence": 0.9
      }
    ],
    "token_cost": 847,
    "raw_file_cost": 62400
  }
}
```

Returns paths through the graph with confidence scores. Shows token cost vs. reading raw files so the agent can judge whether to dig deeper.

### scry_graph_path

```json
{
  "name": "scry_graph_path",
  "input": {"from": "OrderService", "to": "payments table"},
  "output": {
    "shortest_path": ["OrderService.checkout", "PaymentGateway.charge", "payments"],
    "edges": ["calls", "queries"],
    "distance": 2
  }
}
```

### scry_graph_report

Returns the pre-computed report. This is what Claude Code should read first when entering a new repo — equivalent to graphify's PreToolUse hook behavior.

## PreToolUse hook

After `scry setup`, register a PreToolUse hook that fires before Glob and Grep:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Glob|Grep",
        "hooks": [
          {
            "type": "command",
            "command": "scry hook pre-search",
            "statusMessage": "Checking scry graph..."
          }
        ]
      }
    ]
  }
}
```

`scry hook pre-search` checks if a graph report exists for the current repo and returns a short context injection: "Graph report available. God nodes: X, Y, Z. Run scry_graph_report for full context before searching."

This matches graphify's always-on behavior without requiring a separate tool.

## Dependency changes

Current scry dependencies:
- `badger/v4`, `cobra`, `fsnotify`, `scip`

Added by tome absorption:
- `go-sql-driver/mysql`
- `jackc/pgx/v5`

Added by graph layer:
- Community detection algorithm (vendored or small dependency)

Flume and lore add zero new dependencies. Total new deps: 2-3, all well-maintained.

## Binary size estimate

Current scry: ~15MB (mostly BadgerDB + SCIP proto)
After unification: ~18-20MB (mysql/pgx drivers + community detection)

Acceptable for a single static binary.

## What to deprecate

Once scry absorbs everything:

| Old binary | Replacement | Transition |
|-----------|------------|------------|
| `tome` | `scry describe`, `scry relations`, etc. | tome binary prints deprecation notice pointing to scry |
| `flume` | `scry proxy`, `scry requests`, etc. | same |
| `lore` | `scry blame`, `scry history`, etc. | same |

Old MCP tool names (`tome_describe`, `flume_requests`, `lore_blame`) should be kept as aliases for one release cycle, then removed.

## What this gets us

**For the user (you):**
- One `scry setup` instead of four
- One daemon instead of four
- One MCP server in Claude Code config
- `scry init --all` indexes code + schema + git in one shot
- Graph report gives Claude Code architectural awareness without manual prompting

**For Claude Code:**
- Single tool namespace, no routing decisions
- Graph-first exploration: read the report, then targeted queries
- Cross-domain questions answerable without stitching results from four tools
- "Which function queries the users table, was last changed by Jeff, and handles the /api/users endpoint?" — one graph traversal

**vs. graphify:**
- No Python dependency
- No LLM cost for structural analysis
- Richer data: runtime HTTP capture + git history that graphify doesn't have
- Same architectural signal: communities, god nodes, surprising connections
- Single static Go binary, ~20MB, no install friction

## Open questions

1. **Build tags for DB drivers?** If someone never uses schema features, mysql/pgx are dead weight. Build tags would let `go build -tags noschema` produce a smaller binary. Worth the complexity?

2. **Graph rebuild triggers?** Should `scry graph build` run automatically after any domain reindex, or only on explicit request? Auto-rebuild is convenient but adds latency to every file save.

3. **Proxy inside daemon vs. separate process?** The HTTP proxy is long-lived and listens on a port. If it crashes, it shouldn't take down the daemon. Consider running it as a child process managed by the daemon rather than a goroutine.

4. **Cross-domain edge confidence tuning?** The `queries` edge (function → table) requires heuristic SQL/ORM detection. How aggressive should this be? Start conservative (literal table name in string) and expand later.

5. **Community labels?** Leiden gives you clusters but not names. Options: use the highest-degree node's name, use the most common file path prefix, or generate a label from the node types present. No LLM needed.
