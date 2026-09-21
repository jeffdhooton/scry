---
name: scry
description: |
  Use scry for semantic code lookups, git history/blame, database schema,
  captured HTTP requests, cross-domain graph questions, and episodic memory.
  Trigger for callers, references, definitions, implementations, co-change,
  contributors, table columns/foreign keys, request details, and prior decisions;
  also use when installing or troubleshooting scry, its daemon, model credentials,
  or optional Jev shadow assessment. Use text/file search for comments, TODOs,
  error strings, regex and file paths instead of scry symbol queries.
allowed-tools:
  - Bash
  - Read
  - Grep
  - Glob
---

# /scry -- Unified code intelligence through a local semantic index

scry is a single static Go binary that maintains indexes across six
domains: code (SCIP symbols), git (blame/history/cochange), schema (database
tables/FKs/enums), HTTP (captured request/response pairs), a unified graph
that connects domains, and global episodic memory. Queries are served by a
background daemon over a Unix socket at `~/.scry/scryd.sock`; auto-spawn on first
call.

**Read the repo state first, then route.** Before answering any symbol-like
question, decide which tool is actually best for this query. scry is a narrow
precision instrument -- use it where it fits and fall back to Grep where it
doesn't.

## Setup and model dependencies

For the full installation guide, use the
[project README](https://github.com/jeffdhooton/scry#setup). Identify the requested
features and the machine owning memory before changing an existing installation.

| Feature | Required setup |
|---|---|
| Code/git/schema/HTTP | No AI API key. Relevant language toolchain/indexer and project dependencies; `git` for history, a reachable database DSN for schema. |
| Memory extraction | A generative model with a Messages-compatible API, valid account/key, model access/credit and outbound HTTPS from the store daemon. No extraction model is bundled. |
| Jev assessment | Working extraction plus separate TypeSafe access to `jev-1.13.0` and `TYPESAFE_API_KEY`; off by default. |
| Memory storage/retrieval | Writable `~/.scry/` and sufficient disk; embedded Badger/local retrieval, no external vector service or GPU. |
| Unattended ingestion | Readable supported transcripts and a separately configured sweep scheduler. |
| Shared memory | Reachable store socket, usually an SSH tunnel; model config and keys live on the store machine. |

**Exact model/key lookup:** without `memory.models` entries in
`~/.scry/config.yaml`, extraction reads `SCRY_MEMORY_API_KEY`, falling back to
`DEEPSEEK_API_KEY` when empty. Defaults are `deepseek-v4-flash` at
`https://api.deepseek.com/anthropic`. `SCRY_MEMORY_MODEL` and
`SCRY_MEMORY_BASE_URL` override these; an explicit base URL requires an explicit
model. An endpoint supporting only OpenAI-style chat completions is insufficient.

A nonempty `memory.models` chain replaces those model/base URL overrides. Each
entry requires `model`; omitted `base_url` means DeepSeek. An explicit
`api_key_env` reads only the named variable; otherwise the default key lookup
applies. Missing primary or fallback keys leave extraction dormant.
`ANTHROPIC_API_KEY` is not implicitly read: explicitly select the Anthropic
endpoint/model and key variable to use it. Jev reads only `TYPESAFE_API_KEY` and
calls `https://api.typesafe.ai/v1/systemone`; it cannot replace extraction.

**Where keys belong:** the store daemon's process environment. Scry does not
load API keys automatically from repository `.env`, shell profiles, a password
manager or YAML. YAML contains variable names, never secret values. For a new
installation, a private mode-600 `~/.config/scry/credentials.env` can hold the
extraction key and optional `TYPESAFE_API_KEY`, plus all configured key variables.
This is a launcher convention; load it explicitly:

```sh
set -a
. "$HOME/.config/scry/credentials.env"
set +a
scry start --foreground
```

Preserve an existing service's credential convention. Use absolute paths in
launchd/systemd and provide the daemon's toolchain PATH. Update its launcher and
restart the correct daemon after configuration/key changes. An export in another
terminal or a second `scry start` does not update a running daemon. Diagnose key
availability without printing keys or dumping the environment.

Memory routing prefers `SCRY_MEMORY_SOCKET`, then `memory.socket`, then the local
socket. A remote socket suppresses local daemon extraction/assessment; configure
models on the store machine and verify the tunnel. `scry setup` installs Claude
Code MCP/skill/hook integration; it does not install a persistent daemon service,
provision AI accounts/keys, create tunnels or schedule sweeps. Use per-host MCP
configuration for other clients, with an absolute binary path.

**Verify only the requested features:**

```sh
scry doctor --json
scry status
scry memory status
scry memory sweep --dry-run
scry memory assess status --json
scry memory assess list --limit 20 --json
```

For code, verify a real symbol query and the host MCP connection. Intentionally
dormant memory is not a code setup failure. For memory, check the intended
`models`, `dormant: false`, `worker_running: true`, ingestion health and successful
extraction of an authorized episode. For Jev, check `configuration.mode: shadow`,
`key_available`, limits, blocked reason and a completed assessment. A nonempty
key is not proof of provider acceptance. Check command availability against the
installed binary; a release may lag the checkout's optional features.

Jev is observational: it stores assessments separately in `~/.scry/memory-assess/`
and does not govern memory admission or ranking. Setup uses `memory.assessment`
in the store's config, a TypeSafe key and a daemon restart. For a small trial,
configure all of `trial.starts_at`, `trial.expires_at` (RFC3339), and
`trial.max_requests`; preserve authorized dates/caps. Dispatch counts are lifetime
reservations, including failed/uncertain calls. Restart/retention do not reset
them; `resume` cannot bypass expiry/cap stops. Follow the README's bounded-trial
link for the complete config. Setting mode `off` and restarting stops calls.

Honor prior authorization for memory/Jev work without repeated approval requests.
A code-only setup does not authorize paid transcript ingestion, historical
backfill or live assessments. `memory backfill` and assessment fixture `--live`
use the calling process's credentials; fixture success does not prove daemon
setup. Do not make paid calls to verify documentation edits.

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
