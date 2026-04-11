# scry — instructions for the building agent

This file is loaded automatically by Claude Code in this directory. Read it first.

## What this is

A code intelligence daemon for AI agents. Pre-computes a semantic index of every repo (symbols, references, call graphs, types, dependencies) and exposes it as a millisecond-latency local API. Replaces the Read+Grep+Glob cycle that eats 30-50% of every Claude Code session.

> **The full PRD lives in [`docs/SPEC.md`](docs/SPEC.md). Read it before writing any code.** It is 712 lines, opinionated, and self-contained.

## Status (2026-04-10)

- **P0 shipped:** TypeScript indexing, BadgerDB store, `init`/`refs`/`defs`/`status`, CLI-direct (no daemon).
- **P1 shipped:** daemon mode, Unix socket + JSON-RPC 2.0, auto-spawn, fsnotify watch loop with background reindex, Go support, auto-download for `scip-go`, `callers`/`callees`/`impls`, the call graph and implementations indexes.
- **P2 PHP — all four post-processors landed:** scip-php is vendored as an embedded directory tree (not a PHAR — the calibration's PHAR plan failed on autoloader collisions and PHP 8.4 keyword shims; see `docs/DECISIONS.md`). Extracted on first use into `~/.scry/bin/scip-php-<sha>/`. (1) The Laravel non-PSR-4 walker scans `routes/`, `config/`, `database/migrations/`, `bootstrap/` after scip-php and binds ~98% of `::class` refs (1254/1283 on hoopless_crm). (2) The facade resolver hardcodes 31 Illuminate facade -> backing-class mappings and emits ~5k synthetic ref edges so `scry refs AuthManager::user` finds the `Auth::user()` call sites. (3+4) The string-ref walker walks every project .php file for `view('foo.bar')` and `config('foo.bar')` calls and emits synthetic blade-file and config-file ref edges (7 view + 280 config refs on hoopless_crm). The scip parser now also synthesizes SymbolRecords for occurrence-only symbols (vendor/Illuminate/etc.) so they're queryable by name. **PHP P2 is feature-complete.**
- **Reindex window fix shipped:** the watcher's reindex path now uses `index.BuildIntoTemp` to write to a sibling temp directory while the live store keeps serving. `Registry.SwapNext` performs a single ~12ms close+rename+open at the end. Measured on hoopless_crm: 1449 successful queries during a 75s reindex window (which spans a 48s rebuild) with zero failures. Replaces the deferred decision in `docs/DECISIONS.md`.

The current code, layout, commands, and known limitations are documented in [`README.md`](README.md). Read that first for orientation.

## Where to start (continuing the project)

1. **Read [`README.md`](README.md)** — one page covering current state, commands, and gotchas.
2. **Read [`docs/DECISIONS.md`](docs/DECISIONS.md)** — every architectural call made so far, with reasoning and what would change our minds. Anything you're tempted to relitigate is probably already in here.
3. **Read [`docs/PHP_CALIBRATION.md`](docs/PHP_CALIBRATION.md)** before touching PHP. It already validated `scip-php` against a real Laravel app and re-scoped what the P2 post-processor needs to do; the spec's §11.1 is now superseded by this report.
4. **Read [`docs/SPEC.md`](docs/SPEC.md)** if you need the original PRD context. Treat it as read-only history — decisions made *after* the PRD live in `DECISIONS.md`, not here.
5. **For new work:** pick from the P2 list above. The §13 build phases in the spec are the canonical roadmap.

## Hard constraints (don't relitigate)

- **Language: Go 1.23+.** Same stack as `~/workspace/trawl` (a sibling project — borrow infrastructure decisions wholesale).
- **No CGO. Ever.** Single static binary, cross-compile freely. Forces `modernc.org/sqlite` if SQLite is ever needed (though BadgerDB is the spec default).
- **No telemetry, no network calls** except to download language indexers (`scip-typescript`, `scip-go`, etc.) into `~/.scry/bin/`.
- **JSON output by default.** This tool's primary user is an AI agent, not a human. Add `--pretty` for human reading.
- **Local-only.** No cloud, no shared state, no daemon manager required.
- **One binary, one daemon per user, many indexed repos.** Auto-spawn the daemon on first CLI call.

## Language priority for v1

1. **TypeScript / JavaScript** — P0, must work. Indexer: `scip-typescript` (npm package, Sourcegraph first-party).
2. **Go** — P1. Indexer: `scip-go` (Sourcegraph first-party).
3. **PHP / Laravel** — P1 engine, P2 Laravel post-processor. **This is a primary user stack — see §11.1 of the SPEC for the full strategy.** Do not defer it. Do not bucket it as "future work."

Python and Bash were originally in v1 but were demoted because no one explicitly asked for them and they would have eaten the scope budget that PHP/Laravel needs.

## Why PHP matters here

The user actively builds in PHP/Laravel as one of their primary stacks. Most code intelligence and dev tools treat PHP as an afterthought. Building PHP/Laravel support correctly the first time is a deliberate priority, not a nice-to-have.

The spec's §11.1 lays out the strategy: ship `davidrjenni/scip-php` as the engine in P1 (with caveats clearly documented), then build a Laravel-aware post-processor in P2 that handles the dynamic patterns scip-php can't statically resolve — facade resolution (`Auth::user()` → `Illuminate\Auth\AuthManager::user()`), service container bindings (`app(FooService::class)`), Eloquent relationships (`hasMany`/`belongsTo`), route closures (`Route::get('/users', [UserController::class, 'index'])`), view templates (`view('users.show', [...])`), service provider bindings.

**Day-1 calibration exercise (recommended):** before writing P0 code, clone `davidrjenni/scip-php` and run it against a real Laravel app. See exactly what it catches and what it misses. That tells you how much the Laravel post-processor needs to do in P2 — better to know on day one than discover the gap in week six. The user has Laravel codebases on this machine; ask which one to test against.

## When you make decisions

The spec deliberately leaves some things open (§15). When you decide them:

1. **Document the choice in `docs/DECISIONS.md`** (create the file if it doesn't exist).
2. **Include the reasoning**, not just the verdict.
3. **Move on.** Don't relitigate it later unless data changes.

This is the same discipline used in the sibling project `~/workspace/trawl` — see its `docs/DECISIONS.md` for the format.

## Sibling project: trawl

`~/workspace/trawl` is the sibling project. Same architectural philosophy (agent-first, single binary, Go, no CGO, BadgerDB, daemon model), same operational story. **Borrow tech-stack decisions wholesale** — no need to re-evaluate `cobra` vs other CLI frameworks, `zerolog` vs other loggers, etc. If trawl picked it, scry uses it too unless there's a specific reason not to.

The two projects will eventually share infrastructure — possibly via a `~/workspace/agent-cli-kit` library if patterns get duplicated enough to justify extraction. Don't extract prematurely.

## How to ask for help

If you genuinely need user input on a non-obvious decision (architecture trade-off, scope question, taste call), use `AskUserQuestion`. Don't burn an hour spinning on a question the user could answer in 30 seconds.

Examples of good questions:
- "scip-php has experimental quality on Laravel facades. Should P1 ship behind a `--php experimental` flag, or just enable it by default with caveats in the docs?"
- "The Laravel post-processor needs to walk the Eloquent model graph. Should it run during indexing (slower index, faster queries) or lazily on first query (faster index, slower first query)?"

Examples of bad questions:
- "What CLI framework should I use?" (Spec answers this. Use cobra.)
- "Should we support PHP?" (Spec answers this. Yes, P1.)
- "Where should the daemon log?" (Spec lists this as an open question for you to decide and document, §15 #2.)

## What "done" means for the next session

P0, P1, and the first half of PHP P2 are shipped. The next session should pick a phase from the §13 list and own it end-to-end. Suggested next bites in priority order:

1. **Claude Code skill + MCP integration** — scry is purely a CLI today and Claude Code doesn't route to it. The natural shape: a skill at `~/.claude/skills/scry/SKILL.md` that prefers `scry refs <symbol>` over Grep for symbol lookups, plus a slash command like `/scry-here` to init cwd. Then an MCP server exposing scry queries as first-class tools so Claude routes automatically. This is the next step per the user-agreed plan stored in memory `scry roadmap order (2026-04)`.
2. **Vue SFC extraction** — call sites in `.vue` files are invisible today and that's a real productivity gap on Inertia/Vue stacks. Pre-extract `<script>` blocks into virtual TS files before invoking scip-typescript.
3. **Python and Ruby** — both have SCIP indexers (`scip-python`, `scip-ruby`); should be straightforward additions following the pattern in `internal/sources/golang`.
4. **Rust** — `scip-rust` for Cargo monorepos.
5. **PHP P3 polish** (lower priority): receiver-aware string-ref matching (only match `view()` at global scope, not when called as a method), custom user facades, Eloquent property/relationship semantics. None of these have hit a real pain point yet.

Don't try to ship more than one of these in a single session. The phased delivery cadence is in the spec for a reason.

**PHP toolchain note:** to rebuild the embedded scip-php tarball at `internal/sources/php/scip-php.tar.gz`, clone `davidrjenni/scip-php` at the pinned commit, run `composer install --no-dev`, re-apply the `src/Composer/Composer.php` patch (re-prepend scip-php's bundled autoloader after the project's so its `nikic/php-parser` wins — see `docs/DECISIONS.md`), prune `tests/`, `Tests/`, `.github/`, `*.md`, and `*.dist`, then `tar -czf` the tree. There is no automation script yet; one belongs in `scripts/build-scip-php.sh` when someone needs to bump the pin.
