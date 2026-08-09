# scry

**Unified code intelligence daemon for AI agents.** Pre-computes semantic indexes across six domains — code symbols, git history, database schemas, HTTP traffic, a cross-domain graph, and a global episodic memory — and exposes them as millisecond-latency local queries. One binary, one daemon, one MCP server. Replaces scry + tome + flume + lore.

> **Status:** Unified binary shipped. TypeScript/JavaScript, Go, PHP/Laravel, Python indexing. Git intelligence (blame, history, cochange, hotspots, contributors). Database schema introspection (MySQL, PostgreSQL). HTTP reverse proxy capture. Unified cross-domain graph with community detection. 27 MCP tools across 6 domains. See [`docs/SPEC.md`](docs/SPEC.md) for the original PRD and [`docs/DECISIONS.md`](docs/DECISIONS.md) for architectural decisions.

---

## Setup

**The whole thing, if you already know what you're doing:**

```bash
curl -fsSL https://raw.githubusercontent.com/jeffdhooton/scry/main/scripts/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
scry setup                   # MCP server + skill + PreToolUse hook for Claude Code
cd ~/path/to/your/repo && scry init --all
scry doctor                  # green/yellow/red checklist; exit 1 if anything failed
```

Everything below is the long version. If you're an agent doing this on someone's behalf, jump to [Setting scry up as an AI agent](#setting-scry-up-as-an-ai-agent).

### 1. Prerequisites

scry itself is a single static Go binary with no runtime dependencies. Prerequisites are per-domain and only matter for the domains you actually use.

| You want | You need | Notes |
|---|---|---|
| Go code indexing | nothing | `scip-go` auto-downloads to `~/.scry/bin/` (pinned, SHA256-verified) |
| PHP / Laravel indexing | `php` 8.1+ on PATH | `scip-php` is embedded in the binary, extracted on first use |
| TypeScript / JS indexing | `npm i -g @sourcegraph/scip-typescript` | npm package, no auto-download available |
| Python indexing | `python3` (3.10–3.13) + `npm i -g @sourcegraph/scip-python` | Node >=16; see the Python gotcha below |
| Git intelligence | `git` on PATH | already there if it's a repo |
| Schema introspection | a reachable MySQL/PostgreSQL DSN | read from `--dsn` or a `.env` file |
| Claude Code integration | the `claude` CLI on PATH | `scry setup` shells out to `claude mcp add` |
| Memory domain | a DeepSeek (or Anthropic-compatible) API key | fully optional; dormant without one |

**Python gotcha:** `scip-python` 0.6.6's bundled Pyright only recognizes Python 3.10–3.13. If your default `python3` is 3.14+ (common on bleeding-edge Homebrew), scry automatically shims `scip-python` to the first compatible interpreter on PATH (`python3.13`, `python3.12`, `python3.11`, then `python3.10`). If none exists, install one.

### 2. Install the binary

**One-liner** (darwin / linux, amd64 / arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/jeffdhooton/scry/main/scripts/install.sh | sh
```

Drops the binary at `~/.local/bin/scry` after verifying its SHA256 against the release manifest. Customize with `INSTALL_DIR=/usr/local/bin` or pin a version with `SCRY_VERSION=v0.1.0`. Make sure the install dir is on your PATH:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc   # or ~/.bashrc
```

**From source** (requires Go 1.23+):

```bash
go install github.com/jeffdhooton/scry/cmd/scry@latest
```

**Staying current:** `scry upgrade` downloads the latest release and replaces the running binary in place; `scry upgrade --check` just prints what's available; `scry upgrade --version v0.2.0` pins a specific tag. After upgrading, re-run `scry setup` so the registered MCP binary path and the installed skill match the new build. See [`docs/RELEASING.md`](docs/RELEASING.md) if you're publishing a version.

### 3. Wire it into your agent

```bash
scry setup
```

Idempotent — safe to re-run after every upgrade. It writes exactly three things and cleans up a fourth:

| Artifact | Path | What it does |
|---|---|---|
| MCP server registration | `~/.claude.json` (via `claude mcp add --scope user --transport stdio scry -- <bin> mcp`) | Makes all 27 `scry_*` tools available in every Claude Code session |
| Routing skill | `~/.claude/skills/scry/SKILL.md` | Teaches Claude when to pick scry over Grep / git / DB clients |
| PreToolUse hook | `~/.claude/settings.json` under `hooks.PreToolUse` | Intercepts `Grep`/`Glob` and nudges toward `scry_refs`/`scry_defs` (see [PreToolUse hooks](#pretooluse-hooks)) |
| Legacy cleanup | — | Removes stale `tome`, `flume`, `lore` MCP registrations and any old `mcpServers.scry` key wrongly left in `settings.json` |

Useful flags: `--dry-run` (print the plan, write nothing), `--force` (overwrite a customized SKILL.md), `--scry-binary /abs/path` (register a specific binary instead of `os.Executable()`).

`settings.json` is backed up before any change. **Restart Claude Code afterwards** so the new MCP server and skill are picked up.

Not using Claude Code? See [Global agent config files](#global-agent-config-files) for Codex, Cursor, Gemini CLI, and Claude Desktop — `scry setup` only handles Claude Code today, the rest is a two-line config paste.

### 4. Index your first repo

The daemon auto-spawns on the first call; you never have to start it manually.

```bash
cd ~/path/to/your/repo
scry init --all              # code + git + schema, auto-detecting what applies
```

| Command | Indexes | Cost |
|---|---|---|
| `scry init` | code symbols only (TS/JS, Go, PHP, Python — auto-detected) | ~10s per 100k LOC |
| `scry init --git` | blame, commits, cochange, hotspots, contributors | seconds; `--depth N` bounds commit count (default 500, `0` = all) |
| `scry init --schema` | database tables, columns, FKs, enums | needs `--dsn mysql://…` / `postgres://…`, or `--detect-env` to read a `.env` |
| `scry init --all` | everything detected, including auto-DSN discovery | the usual choice |

After the first index, a per-repo fsnotify watcher keeps it fresh (300ms debounce, atomic swap — queries keep serving during a rebuild). Indexes live at `~/.scry/repos/<hash>/`. Check what's indexed with `scry status`.

### 5. Optional: enable the memory domain

The memory domain is a *global* (not per-repo) episodic graph built from your past Claude Code / Codex transcripts and loom runs: entities (projects, machines, people, decisions) with time-stamped facts. It's what makes "set this up on hermes" resolvable in a fresh session.

**It is opt-in and dormant by default.** With no API key set, every memory command prints `memory: dormant (no SCRY_MEMORY_API_KEY / DEEPSEEK_API_KEY)` and exits 0 — nothing is extracted, nothing is billed.

To turn it on, export a key:

```bash
export SCRY_MEMORY_API_KEY=sk-…      # DeepSeek key by default
```

| Env var | Default | Meaning |
|---|---|---|
| `SCRY_MEMORY_API_KEY` | — | Extraction API key. Falls back to `DEEPSEEK_API_KEY`. |
| `SCRY_MEMORY_MODEL` | `deepseek-v4-flash` | Model id. **Required** if you set a custom base URL. |
| `SCRY_MEMORY_BASE_URL` | `https://api.deepseek.com/anthropic` | Any Messages-API-compatible endpoint. |
| `SCRY_MEMORY_UI_ADDR` | `127.0.0.1:7279` | Live memory UI address; `off` disables it. Loopback is forced. |

Extraction defaults to DeepSeek, not Anthropic, on purpose: the sweep runs unattended over every transcript on the machine, so the default has to be the cheap provider. `ANTHROPIC_API_KEY` is deliberately **not** consulted — reaching Anthropic requires naming it in `SCRY_MEMORY_BASE_URL` (and then `SCRY_MEMORY_MODEL` is mandatory). The Message Batches API (50% discount, used by `backfill`) is Anthropic-only; against any other endpoint backfill silently falls back to serial extraction.

```bash
scry memory sweep --dry-run   # what would be ingested, without extracting or paying
scry memory sweep             # scan transcript roots, ingest new episodes
scry memory backfill --since 2026-01-01   # one-time historical pass
scry memory status            # counts, cursors, dormancy
scry memory browse            # searchable HTML graph, opened in your browser
```

While the daemon runs it also serves a live, always-fresh version of that UI at **http://127.0.0.1:7279**.

**Scheduling the sweep** — the sweep is the source of truth (hooks are only a latency optimization), so run it on a timer. There's no installer for this yet; on macOS drop a launchd plist at `~/Library/LaunchAgents/com.user.scry-memory-sweep.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.user.scry-memory-sweep</string>
  <key>ProgramArguments</key>
  <array>
    <string>/Users/YOU/.local/bin/scry</string>
    <string>memory</string>
    <string>sweep</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict><key>SCRY_MEMORY_API_KEY</key><string>sk-…</string></dict>
  <key>StartInterval</key><integer>1800</integer>
  <key>StandardErrorPath</key><string>/tmp/scry-memory-sweep.err</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.user.scry-memory-sweep.plist
```

On Linux, a crontab line does the same job: `*/30 * * * * SCRY_MEMORY_API_KEY=sk-… /home/you/.local/bin/scry memory sweep >/dev/null 2>&1`. See [`docs/MEMORY_SPEC.md`](docs/MEMORY_SPEC.md) for the full design.

### 6. Verify

```bash
scry doctor                  # human checklist
scry doctor --json           # machine-readable report
scry doctor --fix            # auto-remediate the fixable checks
```

`scry doctor` is read-only and checks: binary location, `~/.scry` writability, NOFILE rlimit, daemon state (running / stale / clean), every indexer (`php`, `scip-typescript`, `scip-go`, embedded `scip-php`, `python3`, `scip-python`), Claude Code integration (`claude` CLI, MCP registration, skill, PreToolUse hook, global CLAUDE.md rule), stale tome/flume/lore servers, and the current repo's index state. Every failing check prints its own remediation line.

**Exit code is 1 if any check FAILED, 0 otherwise.** Warnings are advisory and don't affect the exit code — "scip-typescript not installed" only matters if you index TypeScript.

`--fix` handles the fast, reversible subset: creating `~/.scry`, clearing a stale daemon PID/socket, re-running `scry setup` for a missing MCP registration or skill, and creating `~/.claude/CLAUDE.md` with the routing rule when that file doesn't exist yet (an existing one is never edited). Anything slow or surprising (installing PHP, running `scry init`) is reported, never executed.

Then confirm end to end:

```bash
claude mcp get scry          # expect: Status: ✓ Connected
scry status                  # expect: your repo, with per-domain index state
scry refs SomeSymbolYouKnow  # expect: occurrences with file:line:col
```

### 7. Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `scry: command not found` | Install dir isn't on PATH. `export PATH="$HOME/.local/bin:$PATH"`. |
| `claude mcp get scry` says "Failed to connect" | Registered binary path is stale (moved or upgraded). Re-run `scry setup`. |
| Claude never calls scry tools | Skill and hook missing or Claude Code wasn't restarted. `scry doctor`, then restart. Add the [global config block](#global-agent-config-files). |
| Empty results for a symbol you can see | Vendor/external symbol, name collision, or repo not indexed. Try the qualified form (`Class::method`), check `scry status`, fall back to Grep. |
| `"not indexed yet"` RPC error | The watcher is mid-reindex. Retry once — the swap takes ~12ms. |
| TypeScript repo indexes zero symbols | `scip-typescript` missing: `npm i -g @sourcegraph/scip-typescript`. |
| Memory commands print "dormant" | No `SCRY_MEMORY_API_KEY` / `DEEPSEEK_API_KEY` in that shell. Note launchd/cron don't inherit your shell env. |
| Stale daemon after a crash | `scry stop` then any command re-spawns it, or `scry doctor --fix`. |
| Everything is weird | `scry stop && rm -rf ~/.scry/repos && scry init --all`. Indexes are disposable. |

### Migrating from separate tools

If you previously used tome, flume, and lore as separate binaries, `scry setup` automatically removes their MCP registrations. You can also do it manually:

```bash
claude mcp remove tome
claude mcp remove flume
claude mcp remove lore
scry setup        # re-registers the unified scry MCP server
```

The old binaries can be deleted — all functionality is now in `scry`.

## Setting scry up as an AI agent

*If you are an AI coding agent installing scry on a user's machine, this section is the whole contract. It is non-interactive, idempotent, and verifiable.*

**Run this, in order:**

```bash
# 1. Install (skip if `scry version` already works)
curl -fsSL https://raw.githubusercontent.com/jeffdhooton/scry/main/scripts/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"

# 2. Register with Claude Code (idempotent; also removes legacy tome/flume/lore)
scry setup

# 3. Index the repo you're working in
cd "$(git rev-parse --show-toplevel)" && scry init --all

# 4. Verify — exits 1 if any check FAILED
scry doctor --json
```

**Then confirm the tools actually answer**, rather than assuming step 4 was enough:

```bash
scry status --pretty                  # repo listed? which domains?
scry defs <a symbol you saw in the code>   # non-empty result set?
```

**Rules for doing this well:**

- **Parse `scry doctor --json`, don't eyeball the pretty output.** Each check carries an `id`, `status` (`pass` / `warn` / `fail`), and a `remedy` string. Act on `fail`; report `warn` to the user only when it's relevant to their stack (a missing `scip-python` is noise in a Go repo).
- **Try `scry doctor --fix` before asking the user for anything.** It resolves missing `~/.scry`, stale daemons, missing MCP registration/skill, and writes a global CLAUDE.md when the user has none — all without prompting. Re-run `scry doctor --json` afterward and only surface what's *still* failing.
- **Never install language indexers silently.** `npm i -g @sourcegraph/scip-typescript` and `scip-python` are global installs on the user's machine — propose them, don't run them unasked. `scip-go` and `scip-php` need no permission (auto-download / embedded).
- **Never enable the memory domain on your own.** It costs money per token and reads every transcript on the machine. Explain it, hand the user the env var, let them decide. Absent a key it stays dormant and harmless.
- **Tell the user to restart Claude Code.** MCP servers and skills are loaded at session start; nothing you did in step 2 is visible in the *current* session.
- **If the host isn't Claude Code**, `scry setup` has nothing to register — do the [per-host MCP config](#mcp-registration-per-host) instead (Codex, Cursor, Gemini CLI, Claude Desktop). Always write an absolute binary path; `~` isn't expanded in those config files.
- **Don't hand-edit `~/.claude.json`.** `scry setup` shells out to `claude mcp add`, which is the supported path. Editing `settings.json` for MCP config is a known dead end — Claude Code doesn't read MCP servers from there.
- **`scry init --all` is minutes, not seconds, on a large repo.** Run it in the background or warn the user before blocking on it.
- **Indexes are disposable.** If anything is inconsistent, `rm -rf ~/.scry/repos` and re-init. There is no user data in there.

**What "done" looks like:** `scry doctor` exits 0, `claude mcp get scry` reports `Status: ✓ Connected`, `scry status` lists the repo, and a real `scry refs <symbol>` query returns occurrences. Anything less, say so plainly — a half-installed scry that silently returns empty results is worse than no scry.

Finally, add the routing block from [Global agent config files](#global-agent-config-files) to the user's global agent config file — without it, the tools exist but the agent won't reach for them.

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

# Global memory (opt-in — see setup step 5)
scry memory recall hermes     # what do I know about this entity?
scry memory remember "book-system deploys to hermes-mini"
scry memory orient            # orientation blurb for the current directory
scry memory browse            # searchable HTML graph
scry memory status            # counts, cursors, dormancy

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
| **Memory** | Global episodic graph from past sessions: entities, time-stamped facts, provenance (opt-in) |
| **Daemon** | Auto-spawned, Unix socket at `~/.scry/scryd.sock` |
| **JSON-RPC 2.0** | Newline-delimited over Unix socket; methods across 6 domains |
| **MCP server** | 27 tools: 7 code + 6 git + 4 schema + 3 HTTP + 3 graph + 4 memory |
| **Watch loop** | fsnotify per indexed repo, 300ms debounce, atomic reindex swap |
| **Index store** | BadgerDB per domain per repo at `~/.scry/repos/<hash>/` |
| **Auto-download** | `scip-go` (pinned, SHA256-verified). `scip-php` embedded in binary. |
| **Call graph** | Built at index time from SCIP `enclosing_range`. Full on TS, partial on Go. |
| **Implementations** | Built at index time from SCIP `Relationships.is_implementation` |
| **Laravel support** | Non-PSR-4 walker, facade resolver (31 facades), view + config string-refs |
| **Test coverage** | Auto-detects `cover.out`, Istanbul JSON, Clover XML, Python `coverage.json` |
| **Claude Code integration** | Skill routing + 27 MCP tools. `scry setup` handles everything. |

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
| **Memory** | `scry_recall`, `scry_remember`, `scry_episodes`, `scry_memory_path` |

The memory tools only return data once the [memory domain is enabled](#5-optional-enable-the-memory-domain); until then they answer from an empty graph.

## Claude Desktop setup

scry works with Claude Desktop (the chat app) as a standard MCP server. Add it to your Claude Desktop config:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "scry": {
      "command": "/path/to/scry",
      "args": ["mcp"]
    }
  }
}
```

Replace `/path/to/scry` with the actual binary path — typically `~/.local/bin/scry` (install script) or the output of `which scry` (Go install). On macOS you must use the full expanded path, not `~`.

After saving, restart Claude Desktop. You'll see the 27 `scry_*` tools available in the toolbox icon. Before using them, index your repo from a terminal:

```bash
cd ~/path/to/your/repo
scry init --all
```

**What you get:** All 27 MCP tools across the six domains — code intelligence, git history, schema introspection, HTTP traffic, graph queries, and global memory. Claude Desktop doesn't support PreToolUse hooks or skills, so you'll need to explicitly ask Claude to use scry tools (e.g. "use scry_refs to find where processOrder is called"). Once Claude sees the tool results, it quickly learns to prefer them.

**Limitations vs Claude Code:** No PreToolUse hooks (Claude Desktop doesn't support hooks), no routing skill, no automatic nudging, and no `~/.claude/CLAUDE.md` auto-loading. The tools themselves work identically.

**Pro tip:** To get Claude Desktop to prefer scry tools without being asked each time, paste the routing table from the [Global agent config files](#global-agent-config-files) section into your first message or use it as a project prompt. This teaches Claude when to reach for `scry_graph_report` vs `scry_refs` vs Grep — the same routing that Claude Code gets automatically from `~/.claude/CLAUDE.md`.

## Claude Code integration

scry integrates with Claude Code (CLI, desktop app, VS Code, and JetBrains extensions) at four levels: MCP tools, a routing skill, PreToolUse hooks, and your global `CLAUDE.md`. `scry setup` handles the first three; the routing block in `CLAUDE.md` is a paste. The hooks and the routing block are what make Claude *prefer* scry over raw Grep/git without you having to ask.

### What `scry setup` does

```bash
scry setup
```

1. **Registers the MCP server** — runs `claude mcp add --scope user --transport stdio scry -- <binary> mcp`, making all 27 `scry_*` tools available in every Claude Code session.
2. **Installs the routing skill** — writes `~/.claude/skills/scry/SKILL.md`, a detailed routing table that teaches Claude when to reach for scry vs Grep vs Read. Covers all six domains with example queries.
3. **Installs the `pre-search` PreToolUse hook** — into `~/.claude/settings.json`, backing the file up first and replacing any stale scry hook entry pointing at an old binary path.
4. **Cleans up legacy tools** — removes old `tome`, `flume`, `lore` MCP registrations if present.

Verify with:

```bash
claude mcp get scry              # should show Status: ✓ Connected
scry doctor                      # full health check
```

### PreToolUse hooks

The MCP tools and skill give Claude the *ability* to use scry, but Claude will still sometimes reach for Grep or `git log` out of habit. PreToolUse hooks intercept those calls and nudge Claude toward scry equivalents — or tell you when a repo isn't indexed yet.

`scry setup` installs the `pre-search` hook (`Grep|Glob`) for you. The `pre-git` hook (`Bash`) is opt-in — add it by hand. Both live in `~/.claude/settings.json` under the `hooks.PreToolUse` array:

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

The hooks intercept Grep and git calls, but Claude also needs to know *when* to reach for scry proactively — especially for graph, memory, and architecture questions where it would otherwise just read files. Paste the routing block from [Global agent config files](#global-agent-config-files) into `~/.claude/CLAUDE.md`, or let `scry doctor --fix` write the file for you if you don't have one yet (it never edits an existing CLAUDE.md).

That routing table is what makes Claude reach for `scry_graph_report` when you ask "what's the architecture?" instead of `ls`-ing directories.

### Full integration checklist

```bash
scry setup                       # MCP server + skill + PreToolUse hook
scry doctor                      # verify prereqs
scry init --all                  # index current repo
# Confirm hooks landed in ~/.claude/settings.json (see above)
# Add the routing block to ~/.claude/CLAUDE.md (see below)
# Restart Claude Code
# Verify: work in Claude Code, check ~/.scry/logs/mcp-calls.jsonl
```

## Global agent config files

MCP registration gives an agent the *ability* to call scry. The routing block below is what makes it actually *reach* for scry instead of Grep, `git log`, or a DB client. Install both.

### The routing block

Paste this into whichever global instruction file your agent reads (paths in the next table). It's the same content `scry setup` installs as a Claude Code skill, condensed to fit a global config file.

```markdown
## scry — use FIRST for code intelligence

scry is a local code intelligence daemon with pre-computed indexes. It answers in <10ms what
Grep/git/file reading takes 30+ seconds to assemble. Always check scry before reaching for
Grep, git commands, or reading files to understand code structure.

### When to use which scry tool

**Starting work in a repo or answering "what is this codebase?":**
→ scry_graph_report — shows architecture: god nodes (highest coupling), communities (feature
clusters), cross-domain connections. Start here for any orientation or onboarding question.
Do NOT read docs or ls directories first.

**"Where is X used/called/defined?":**
→ scry_refs or scry_defs — every reference or definition with file:line:col. Use instead of
Grep for any symbol/identifier lookup. Accepts Mail::to, auth->user, client.Connect forms.

**"What calls X?" / "What does X call?":**
→ scry_callers / scry_callees

**"What implements this interface?" / "Is this function tested?":**
→ scry_impls / scry_tests

**"Who wrote this?" / "What changed recently?" / "Why was this written?":**
→ scry_blame, scry_history, scry_intent — use instead of git blame/git log.

**"What files change together?" / "What are the hotspots?" / "Who maintains this?":**
→ scry_cochange, scry_hotspots, scry_contributors

**"How does X connect to Y?" / "What are the main modules?":**
→ scry_graph_path, scry_graph_query — across code, git, schema, and HTTP domains.

**"What tables/columns/FKs exist?":**
→ scry_describe, scry_relations, scry_schema_search, scry_enums — use instead of DB clients.

**"What HTTP requests happened?":**
→ scry_requests, scry_request

**Unknown referent (a project, service, machine, person, or decision not defined in this
conversation or repo):**
→ scry_recall FIRST — global memory of past sessions across all projects ("set this up on
hermes" → recall "hermes"). scry_episodes traces when it was discussed; scry_memory_path
shows how two things relate.

**Durable facts and decisions (deploys, choices made, lasting preferences):**
→ scry_remember — store it, don't just say it.

### When to fall back to Grep/Read
- String searches in comments, docstrings, error messages
- TODO/FIXME hunting
- Regex pattern matching over file content
- The repo is not indexed (scry_status to check)
- scry returned empty results for a known symbol
```

Drop the memory lines if you didn't enable the memory domain, and the schema/HTTP lines if you never index those.

### Where each host reads it

| Host | Global instruction file | Project-level file |
|---|---|---|
| **Claude Code** | `~/.claude/CLAUDE.md` | `./CLAUDE.md` |
| **Codex CLI** | `~/.codex/AGENTS.md` | `./AGENTS.md` |
| **Cursor** | Settings → Rules → *User Rules* | `.cursor/rules/scry.mdc` |
| **Gemini CLI** | `~/.gemini/GEMINI.md` | `./GEMINI.md` |
| **Claude Desktop** | no global file — paste into project instructions or your first message | — |
| **Windsurf / others** | check the host's docs; most now read `AGENTS.md` | `./AGENTS.md` |

For a repo-scoped variant, put a shorter version in the project file naming the specific symbols and tables that matter in that codebase.

### MCP registration per host

`scry setup` only registers with Claude Code today. `scry mcp` is a standard MCP stdio server, so every other host is a two-line config entry. Use an absolute path — `~` is not expanded in most of these files. Get it with `which scry`.

**Codex CLI** — either `codex mcp add scry -- /abs/path/to/scry mcp`, or in `~/.codex/config.toml`:

```toml
[mcp_servers.scry]
command = "/abs/path/to/scry"
args = ["mcp"]
```

**Cursor** — `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project):

```json
{
  "mcpServers": {
    "scry": { "command": "/abs/path/to/scry", "args": ["mcp"] }
  }
}
```

**Gemini CLI** — `~/.gemini/settings.json`, same `mcpServers` shape as Cursor.

**Claude Desktop** — see [Claude Desktop setup](#claude-desktop-setup) above.

**Anything else** — the generic form is: command `/abs/path/to/scry`, args `["mcp"]`, transport stdio, no env vars required. The daemon auto-spawns on the first tool call.

Config paths are what the vendors document as of this writing; if a host has moved its config, its own docs win. Whatever the host, the check is the same: the tool list should show 27 `scry_*` tools, and a call should return results within milliseconds.

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
  bin/                        # auto-downloaded indexers (scip-go, extracted scip-php)
  logs/mcp-calls.jsonl        # every MCP tool invocation
  memory/                     # global episodic memory graph (not per-repo)
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
│   ├── mcp/                   # MCP stdio server (27 tools)
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
