---
name: scry
description: |
  Route symbol lookups through scry (a local code-intelligence daemon) instead of Grep.
  scry answers "where is <symbol> defined / used / called / implemented" in <10ms for
  indexed repos by pre-computing a semantic index (SCIP) at `scry init` time. Use this
  skill whenever you need to find a function, class, method, interface, or any named
  symbol in a codebase — it's dramatically faster and more precise than Grep for symbol
  queries, because it understands scope, types, and references.

  TRIGGER when: user asks "where is X called/used/defined/implemented", "who calls X",
  "what does X call", "find every reference to X", "jump to X"; or when you would
  otherwise reach for Grep to find a function/class/method name; or when searching for
  how a specific identifier is wired through a codebase.

  DO NOT use for: string searches inside comments/docstrings, TODO hunting, error
  message lookups, regex over content, file path patterns, or any non-symbol query —
  use Grep/Glob/Read for those. scry is narrow by design.
allowed-tools:
  - Bash
  - Read
  - Grep
  - Glob
---

# /scry — Symbol lookups through a local semantic index

scry is a single static Go binary that maintains a per-repo BadgerDB index of
every symbol, reference, definition, call edge, and implementation relationship.
It's backed by SCIP (SourceGraph Code Intelligence Protocol) and indexes
TypeScript, JavaScript, Go, and PHP/Laravel. Queries are served by a background
daemon over a Unix socket at `~/.scry/scryd.sock`; auto-spawn on first call.

**Read the repo state first, then route.** Before answering any symbol-like
question, decide which tool is actually best for this query. scry is a narrow
precision instrument — use it where it fits and fall back to Grep where it
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
| "Find every TODO in the codebase" | **Grep** | scry doesn't index comments or strings |
| "Find the error message 'permission denied'" | **Grep** | String match, not symbol |
| "Find all `.yaml` files" | **Glob** | File path patterns |
| "Read `config/app.php`" | **Read** | Opening a file |
| "Find where dotenv is configured" | **Grep first, then scry** | If "dotenv" turns out to be a class/function name, pivot to scry |

## Golden path

1. **Check if the current repo is indexed** before firing a query:
   ```bash
   scry status
   ```
   If the repo is listed under "indexed", proceed. If not, see "First-index behavior" below.

2. **Run the query**:
   ```bash
   scry refs <symbol>     # every reference
   scry defs <symbol>     # every definition
   scry callers <symbol>  # every call site with containing function
   scry callees <symbol>  # every outgoing call from this function
   scry impls <symbol>    # every implementor of this interface/base
   ```
   All commands output JSON by default. Add `--pretty` for human-readable.
   Results include absolute file paths, 1-indexed line + column, and a
   trimmed context line from the source.

3. **Interpret results**:
   - Empty result set with a known-good symbol name → probably a name collision
     (scry matches by display name case-insensitively). Try the fully qualified
     form or pivot to Grep to disambiguate.
   - `"not indexed yet"` RPC error → the watcher is mid-reindex. This window is
     now ~12ms after the 2026-04-10 fix; retry once, then fall back to Grep
     if it persists.
   - Results tagged `"kind": "External"` are symbols referenced but not defined
     inside the project (vendor classes, framework, stdlib). See "Vendor caveat".

## First-index behavior (refuse + suggest)

If `scry status` shows the current repo is NOT indexed, **do not auto-run**
`scry init`. Indexing a real project takes 10-60 seconds and would feel broken
if kicked off silently from inside another query. Instead:

1. Print a one-line notice: `scry: repo not indexed — run 'scry init .' to enable symbol lookups`.
2. Fall back to Grep for the current query.
3. If the user explicitly asks "index this repo" or "scry init", then run it.

The user can always opt in with `scry init .` before starting work in a new repo.
Once indexed, the daemon's fsnotify watcher keeps the index fresh on every save
(atomic reindex swap, ~12ms query unavailability window during rebuilds).

## Vendor / framework caveat

scry's parser synthesizes SymbolRecords for any referenced symbol that scip-php,
scip-go, or scip-typescript didn't emit a SymbolInformation block for — which
covers every vendor/framework/stdlib class. Concretely:

- **`scry refs Illuminate\\Support\\Facades\\DB` works** — returns all 252
  call sites in a typical Laravel app, because the project files reference it
  even though vendor/ isn't walked as source.
- **`scry defs Illuminate\\Support\\Facades\\DB` returns empty** — because the
  definition lives in `vendor/laravel/framework/.../DB.php` which scry never
  indexed as source.

When the user asks "where is X defined" and scry returns empty for a name that
obviously exists, it's almost always a vendor class. Fall back to:
- `Grep` with the class name inside `vendor/` (or `node_modules/`)
- Or `Read` a known file path if you can infer one from the SCIP symbol ID
  (the `Illuminate/Support/Facades/DB#` shape maps to the file path directly)

## Facade resolution (Laravel specific)

scry ships a facade resolver that mirrors every `Auth::user()` ref onto the
backing class's `user()` method. So all three of these queries return the same
75 call sites on a real Laravel app:

```bash
scry refs "Illuminate/Support/Facades/Auth#user"            # the facade method
scry refs "Illuminate/Auth/AuthManager#user"                # the manager
scry refs "Illuminate/Contracts/Auth/Guard#user"            # the contract
```

Useful when the user knows the contract/manager name but not the facade name.
Same pattern for DB, Cache, Log, Mail, Queue, etc. — see `docs/DECISIONS.md`
in the scry repo for the full facade map.

## Reference output shape

Every query returns JSON with this general shape:

```json
{
  "symbol": "processOrder",
  "matches": [
    {
      "symbol_id": "scip-typescript npm . . src/orders/service.ts/processOrder().",
      "display_name": "processOrder",
      "kind": "Method",
      "occurrences": [
        {
          "symbol": "...",
          "file": "src/orders/service.ts",
          "line": 42,
          "column": 14,
          "end_line": 42,
          "end_column": 26,
          "context": "  return processOrder(order, user);",
          "is_definition": false
        }
      ]
    }
  ],
  "total": 1,
  "elapsed_ms": 6
}
```

When summarizing results for the user, lead with the top 5-10 occurrences
as `file:line — context` bullets, not with the raw JSON. If the list is long,
group by file.

## Command reference

```bash
scry init [<repo>]           # Index a repo (default: cwd). 10-60s depending on size.
scry status                  # List all indexed repos and their state.
scry refs <symbol>           # Every reference.
scry defs <symbol>           # Every definition.
scry callers <symbol>        # Every caller with containing function.
scry callees <symbol>        # Every callee of this function.
scry impls <symbol>          # Every implementor of this interface.
scry start [--foreground]    # Explicit daemon start (auto-spawns otherwise).
scry stop                    # Graceful daemon shutdown.
```

Global flags:
- `--repo <path>` — target repo (defaults to cwd)
- `--pretty` — human-readable JSON output
- `-h, --help` — per-command help

## When scry returns nothing

Decision tree:

1. **Name is right, 0 results** → might be an external/vendor symbol. Fall back to Grep.
2. **Name is a substring or pattern** → scry only exact-matches display names. Use Grep.
3. **Name is in a language scry doesn't index yet** (Python, Ruby, Rust, Bash) → Grep.
4. **Repo not indexed** → print the refuse+suggest notice above and use Grep.
5. **Daemon crashed** → `scry status` returns an error. Run `scry start` to restart.

## MCP server mode

scry also ships an MCP server (`scry mcp`) that exposes the same queries as
first-class Claude Code tools — `scry_refs`, `scry_defs`, `scry_callers`,
`scry_callees`, `scry_impls`, `scry_status`. When the MCP server is registered
in Claude Code settings, these tools appear alongside Grep/Glob/Read and you
can route to them directly without reading this skill. The skill is the
fallback path for when MCP isn't configured yet.
