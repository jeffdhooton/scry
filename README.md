# scry

**Code intelligence daemon for AI agents.** Pre-computes a semantic index of every repo you work in (symbols, references, call graphs, types, dependencies) and exposes it as a millisecond-latency local API. Replaces the Read+Grep+Glob cycle that eats 30-50% of every Claude Code session.

> Status: spec-only. No code yet. This directory exists so a fresh Claude Code instance can pick up the work.

---

## For the AI agent picking this up

1. **Read [`docs/SPEC.md`](docs/SPEC.md) first.** It is a complete PRD with architecture, tech-stack decisions, build phases, the query API, and a first-commit checklist. Self-contained, opinionated.
2. **Start with Phase P0** (§13 of the spec). The thinnest version: single language (TypeScript), CLI-only (no daemon yet), basic queries against a precomputed SCIP dump. Done when you can run `scry refs <symbol>` against a real TS repo and get back accurate results in <50ms.
3. **Don't relitigate the decisions in §7 (tech stack) or §3 (non-goals).** They are deliberate. If you disagree strongly, write your reasoning into a `docs/DECISIONS.md` and surface it to the user — don't silently override.
4. **Hard constraints:** Go 1.23+, no CGO, single static binary, JSON output by default, local-only (no cloud, no telemetry, no network calls outside of fetching language indexers).

## For humans reading this in a year

The pitch in one paragraph: every Claude Code session burns 30-50% of its tool calls on Read + Grep + Glob just to *find* the relevant code before doing anything with it. That's the slowest, most context-burning, most repetitive part of the agent loop. LSP exists, but LSP is human-first — it powers IDE features (hover, jump-to-definition, autocomplete) that an agent doesn't need. The agent needs *structured queries* like "all callers of function X across the repo, with their containing class and one line of context." scry is a long-running daemon that pre-computes that information and exposes it as a millisecond-latency CLI/API. Same instinct as `trawl` (agent-first, not human-first), applied to code intelligence instead of web scraping.

## Why a daemon

Pre-computation is the entire game. A typical 100k-LOC TypeScript repo has ~5000 symbols, ~30000 references, ~10000 type relationships. Computing those on every query is impossibly slow (~5-30 seconds via LSP). Computing them once when the file changes, storing them in an embedded KV, and querying via index lookup is ~10ms. The daemon model is required because the cost of building the index is amortized across thousands of queries, but only if the index *stays warm* between calls.

## Why Go

Same answers as trawl: single static binary, fast cold start, mature ecosystem for daemon patterns (signal handling, fsnotify, unix sockets), and the file-watching + indexing workload is concurrency-bound rather than CPU-bound. Reuses the same infrastructure decisions (BadgerDB, cobra, zerolog) so the operational story matches trawl exactly.

## Layout

```
scry/
├── README.md          # this file
└── docs/
    └── SPEC.md        # full PRD — read this first
```

Once Phase P0 starts, expect the standard Go layout: `cmd/scry/`, `internal/index/`, `internal/store/`, `internal/query/`, `internal/sources/`.

## Sibling project

[trawl](file:///Users/jhoot/workspace/trawl) — agent-first web scraping, same architectural philosophy. scry borrows trawl's tech-stack decisions wholesale.
