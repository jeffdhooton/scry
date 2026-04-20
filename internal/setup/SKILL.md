---
name: scry
description: |
  Route symbol lookups, git intelligence, schema queries, HTTP inspection, and
  cross-domain graph traversal through scry (a local code-intelligence daemon)
  instead of Grep, git commands, manual DB inspection, and log reading. scry is
  a unified tool that answers code, schema, git, HTTP, and architectural
  questions in <10ms via a single daemon and MCP server.

  TRIGGER when: user asks "where is X called/used/defined/implemented", "who
  calls X", "what does X call", "find every reference to X", "jump to X", "who
  wrote this line", "what changed in this file recently", "which files change
  together", "what are the hotspots", "who are the main contributors", "what
  columns does this table have", "what are the foreign keys", "what enums
  exist", "what was the last HTTP request", "show me the graph report", "what
  connects X to Y"; or when you would otherwise reach for Grep to find a
  function/class/method name, git blame/log for history context, or DB clients
  for schema info.

  DO NOT use for: string searches inside comments/docstrings, TODO hunting,
  error message lookups, regex over content, file path patterns, or any
  non-symbol query -- use Grep/Glob/Read for those. scry is narrow by design.
allowed-tools:
  - Bash
  - Read
  - Grep
  - Glob
---

# /scry -- Unified code intelligence through a local semantic index

scry is a single static Go binary that maintains per-repo indexes across five
domains: code (SCIP symbols), git (blame/history/cochange), schema (database
tables/FKs/enums), HTTP (captured request/response pairs), and a unified graph
that connects all domains. Queries are served by a background daemon over a
Unix socket at `~/.scry/scryd.sock`; auto-spawn on first call.

**Read the repo state first, then route.** Before answering any symbol-like
question, decide which tool is actually best for this query. scry is a narrow
precision instrument -- use it where it fits and fall back to Grep where it
doesn't.

## Routing table

| Query shape | Tool | Why |
|---|---|---|
| "Where is function `processOrder` called?" | **scry refs** | Returns every ref occurrence with file:line + context in ms |
| "Where is class `UserController` defined?" | **scry defs** | Returns the definition site(s) |
| "What calls `Auth::user()`?" | **scry refs** | scry's facade resolver wires this to `AuthManager#user` too |
| "What does `processOrder` call internally?" | **scry callees** | Needs the pre-computed call graph |
| "Who calls `processOrder`?" | **scry callers** | Callers = refs with containing function attached |
| "What implements interface `Repository`?" | **scry impls** | Uses SCIP relationship edges |
| "Is this function tested?" | **scry tests** | Coverage index joined against symbol defs |
| "Who wrote this line?" | **scry blame** | Pre-indexed blame, faster than `git blame` |
| "What changed recently in this file?" | **scry history** | Structured commit data with diff stats |
| "What files change with this one?" | **scry cochange** | Co-change coupling analysis |
| "What are the most churned files?" | **scry hotspots** | Ranked by commit frequency |
| "Who knows this code best?" | **scry contributors** | Per-file or repo-wide |
| "Why was this line written?" | **scry intent** | Blame + full commit context |
| "What columns does users have?" | **scry describe** | Database schema from live introspection |
| "What are the foreign keys on orders?" | **scry relations** | FK relationships with direction |
| "Find a table or column named email" | **scry schema-search** | Substring search across tables and columns |
| "What enums exist in the database?" | **scry enums** | Enum types and their values |
| "What was the last API request?" | **scry requests** | Captured HTTP traffic via reverse proxy |
| "Show details of request X" | **scry request** | Full req/res with headers and body |
| "What connects User to /api/auth?" | **scry graph path** | Cross-domain shortest path |
| "What are the god nodes?" | **scry graph report** | Pre-computed architectural summary |
| "Find every TODO in the codebase" | **Grep** | scry doesn't index comments or strings |
| "Find the error message 'permission denied'" | **Grep** | String match, not symbol |
| "Find all `.yaml` files" | **Glob** | File path patterns |
| "Read `config/app.php`" | **Read** | Opening a file |

## Golden path

1. **Check if the current repo is indexed** before firing a query:
   ```bash
   scry status
   ```
   If the repo is listed under "indexed", proceed. If not, see "First-index behavior" below.

2. **Run the query**:
   ```bash
   # Code intelligence
   scry refs <symbol>     # every reference
   scry defs <symbol>     # every definition
   scry callers <symbol>  # every call site with containing function
   scry callees <symbol>  # every outgoing call from this function
   scry impls <symbol>    # every implementor of this interface
   scry tests <symbol>    # test coverage status

   # Git intelligence
   scry blame <file>            # structured blame
   scry history [<file>]        # recent commits
   scry cochange <file>         # co-changed files
   scry hotspots                # most churned files
   scry contributors [<file>]   # main authors
   scry intent <file> --line N  # why was this line written?

   # Schema
   scry describe <table>        # table structure with columns/types/keys
   scry relations <table>       # foreign key relationships
   scry schema-search <term>    # find tables/columns by name
   scry enums [table.column]    # enum types and values

   # HTTP capture
   scry requests [--path /api]  # list captured requests
   scry request <id>            # full request/response detail

   # Graph
   scry graph report            # architectural summary (read first!)
   scry graph query <term>      # search graph nodes
   scry graph path --from X --to Y  # shortest path between nodes
   ```
   All commands output JSON by default. Add `--pretty` for human-readable.

3. **Interpret results**:
   - Empty result set with a known-good symbol name -> probably a name collision
     or a vendor/external symbol. Try the fully qualified form or pivot to Grep.
   - `"not indexed yet"` RPC error -> the watcher is mid-reindex. Retry once.
   - Results tagged `"kind": "External"` are vendor/framework/stdlib symbols.

## First-index behavior (refuse + suggest)

If `scry status` shows the current repo is NOT indexed, **do not auto-run**
`scry init`. Instead:

1. Print a one-line notice: `scry: repo not indexed -- run 'scry init .' to enable symbol lookups`.
2. Fall back to Grep for the current query.
3. If the user explicitly asks "index this repo" or "scry init", then run it.

## Indexing all domains

```bash
scry init .                  # code only (TypeScript, Go, PHP, Python)
scry init --git .            # git history only
scry init --schema --dsn "..." . # database schema
scry init --all .            # code + git + schema (auto-detects DSN from .env)
```

After indexing multiple domains, build the unified graph:
```bash
scry graph build .           # connects code, git, schema, HTTP data
scry graph report            # read the architectural summary
```

## HTTP capture

The HTTP proxy captures request/response pairs from your dev server:
```bash
scry proxy start --port 8089 --target localhost:8000
# Point your app/browser at localhost:8089 instead of :8000
scry requests                # see captured traffic
scry proxy stop              # tear down when done
```

## Command reference

```bash
# Code intelligence
scry init [<repo>]           # Index code. 10-60s depending on size.
scry init --git [<repo>]     # Index git history.
scry init --schema [<repo>]  # Index database schema (--dsn or auto-detect .env).
scry init --all [<repo>]     # Index everything detected.
scry status                  # List all indexed repos and domain states.
scry refs <symbol>           # Every reference.
scry defs <symbol>           # Every definition.
scry callers <symbol>        # Every caller with containing function.
scry callees <symbol>        # Every callee of this function.
scry impls <symbol>          # Every implementor of this interface.
scry tests <symbol>          # Test coverage status for a symbol.

# Git intelligence
scry blame <file>            # Structured blame (--start-line, --end-line).
scry history [<file>]        # Recent commits (--limit N).
scry cochange <file>         # Files that change alongside target (--limit N).
scry hotspots                # Most-churned files (--limit N).
scry contributors [<file>]   # Main authors, ranked by commit count.
scry intent <file> --line N  # Why was this line written?

# Schema
scry describe <table>        # Table structure with columns, types, keys.
scry relations <table>       # Foreign key relationships.
scry schema-search <term>    # Search tables and columns by name.
scry enums [table.column]    # Enum types and their allowed values.

# HTTP capture
scry proxy start [--port 8089 --target localhost:8000]
scry proxy stop
scry requests [--path X --method Y --limit N]
scry request <id>

# Graph
scry graph build [path]      # Build unified cross-domain graph.
scry graph query <term>      # Search graph nodes by name.
scry graph path --from X --to Y  # Shortest path between nodes.
scry graph report            # Pre-computed architectural summary.

# Infrastructure
scry start [--foreground]    # Explicit daemon start.
scry stop                    # Graceful daemon shutdown.
scry setup                   # Install skill + MCP server.
scry doctor                  # Health check.
scry upgrade                 # Update to latest release.
```

Global flags:
- `--repo <path>` -- target repo (defaults to cwd)
- `--pretty` -- human-readable JSON output
- `-h, --help` -- per-command help

## MCP server mode

scry ships an MCP server (`scry mcp`) that exposes all queries as
first-class Claude Code tools:

**Code:** `scry_refs`, `scry_defs`, `scry_callers`, `scry_callees`, `scry_impls`,
`scry_tests`, `scry_status`

**Git:** `scry_blame`, `scry_history`, `scry_cochange`, `scry_hotspots`,
`scry_contributors`, `scry_intent`

**Schema:** `scry_describe`, `scry_relations`, `scry_schema_search`, `scry_enums`

**HTTP:** `scry_requests`, `scry_request`, `scry_http_status`

**Graph:** `scry_graph_query`, `scry_graph_path`, `scry_graph_report`

When the MCP server is registered, these 23 tools appear alongside Grep/Glob/Read
and you can route to them directly without reading this skill.
