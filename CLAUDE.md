# scry — instructions for the building agent

This file is loaded automatically by Claude Code in this directory. Read it first.

## What this is

A code intelligence daemon for AI agents. Pre-computes a semantic index of every repo (symbols, references, call graphs, types, dependencies) and exposes it as a millisecond-latency local API. Replaces the Read+Grep+Glob cycle that eats 30-50% of every Claude Code session.

> **The full PRD lives in [`docs/SPEC.md`](docs/SPEC.md). Read it before writing any code.** It is 712 lines, opinionated, and self-contained.

## Where to start

1. **Read [`README.md`](README.md)** — one page of orientation.
2. **Read [`docs/SPEC.md`](docs/SPEC.md)** in full. Pay special attention to:
   - §3 (Goals & Non-Goals — what we're explicitly NOT building)
   - §7 (Tech stack — decided, don't relitigate)
   - §11 (Languages supported — TS/JS, Go, PHP/Laravel for v1)
   - §11.1 (PHP — the wildcard. Read this carefully. PHP is the trickiest of the four languages and has a specific multi-phase strategy.)
   - §13 (Build phases — start with P0)
   - §15 (Open questions — these are decisions you should make, document in `docs/DECISIONS.md`, and move on)
   - §17 (First commit checklist — your day-1 deliverable)
3. **Start with Phase P0.** TypeScript/JavaScript only, CLI-only (no daemon yet), basic queries against a precomputed SCIP dump.

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

## What "done" means for the first session

A successful first session ships either:
- **Phase P0 in full** (the first commit checklist in §17 of the spec), OR
- **Enough of P0 to validate the core risk**: `scry init` indexes a real TypeScript repo, `scry refs <symbol>` returns accurate results from the BadgerDB store. Even if the CLI is rough and the test coverage is thin, getting that round-trip working proves the architecture.

Don't try to ship P1/P2/P3 in one session. Phased delivery is in the spec for a reason.
