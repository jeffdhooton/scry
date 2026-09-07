# How good a brain is scry? Assessment and workflow plan, 2026-09-04

An outside read of the graphing and memory layers as they stand today, measured
against the live shared store on the mini, the laptop's MCP call log, and the
repo at `3b29ae7`. Written to answer two questions: how good is the graph, and how
good is the memory as the brain behind autonomous engineering runs. The last
section is the plan.

Companion documents: [`MEMORY_HANDOFF_2026-09-04.md`](MEMORY_HANDOFF_2026-09-04.md)
(state of the memory-solid run), [`MEMORY_AUDIT_2026-09-02.md`](MEMORY_AUDIT_2026-09-02.md)
(every measurement, appended), [`MEMORY_SPEC.md`](MEMORY_SPEC.md) (the design),
[`DECISIONS.md`](DECISIONS.md).

```
measured    2026-09-04 16:00–16:30 EDT, live daemon on mini via the tunnel socket
store       25,081 entities · 64,192 facts · 7,786 episodes · 2,970 cursors
queue       1,162 ready · 3 in backoff · 0 parked · worker running, chain glm-5.3-flash → deepseek-v4-flash
benchmarks  tuning 47/50 · strict 44/50 · heldout 53/62 · heldout-b 34/66 · probes 7/7
```

---

## 1. Verdict in one table

| Function | Grade | One line |
|---|---|---|
| Unified code graph (`scry_graph_*`) | **F** as a code graph, **C** as a co-change graph | Zero function, class, or interface nodes in any repo's graph. Every node is a file, an author, or a table. Root cause in §2. |
| Ingestion | **B+** | Alive on both machines, loud when it stops, per-source counts visible. Backlog of 1,162 episodes is ~4–5 hours of drain. Two items have retried 464 times on a permanent error. |
| `scry_remember` | **A−** | p50 22 ms, p95 239 ms since 2026-09-02 (was p50 24.5 s, p95 154 s before). Durable through a 22-hour outage. But the fact still passes through the extractor, which mistypes and mangles it. |
| Recall when the asker names the thing | **B+** | 88–95% top-20 on the three name-heavy sets. Mean answer rank 4. Latency 65–97 ms. |
| Recall when the asker describes the thing | **D** | 51.5% on heldout-b, unchanged across six ranking attempts. The honest number for a new agent that does not yet know the vocabulary. |
| Temporal reasoning | **D+** | Invalidation fires only for `status` and `replaced_by`. Everything else accumulates, so "is X deployed" returns both "NOT deployed" and "deployed at commit e9060c4" as current. |
| Entity identity | **C−** | 51% of entities are `concept`. 14% of a random sample hold no current fact. ~900 are mangled file paths. 191 of 291 "persons" are fleet agents. The largest project entity holds another project's deploy facts. |
| Orientation (`memory orient`) | **C** | Shows the last five entities touched in the repo. Does not surface the deploy command, the gate, or the standing rules. The hand-written `MEMORY.md` for the same repo does. |
| Reflex: does memory reach a session unasked? | **F** | No `SessionStart` orient hook, no `SessionEnd` ingest hook, on either machine. The spec called for both. Memory is pull-only, and the pull happens 390 times in four months. |

**The short version.** Scry memory is a good ledger with a decent name index, a
weak associative memory, and no reflex. The store is real, the write path is
durable, and recall works when you already know what to ask for. What it is not
yet is a brain: nothing pushes it into a session, the curated memory agents
actually write (`~/.claude/projects/*/memory/`) bypasses it, and the code graph
that was meant to be the deterministic half is missing its code. The biggest
gains available are wiring and coverage, not ranking. The memory-solid run
spent thirteen rounds on resolver rules; the next run should spend its budget on
the workflow around the store.

---

## 2. The unified graph has no code in it

### What was measured

`scry_graph_report` on this repo: 83 nodes, 175 edges, 22 communities, every
god node a file, `surprising_edges: null`. `scry_graph_query "Recall"` returns
four `.go` files and no function. On docket, the largest graph on the machine:
439 nodes, of which 416 are files; the god node is `web/src/driver/pages.tsx`
with degree 24; communities are co-change clusters.

| Repo | Nodes | Edges | Communities | File nodes |
|---|---|---|---|---|
| docket | 439 | 966 | 99 | 416 |
| ph-develop | 294 | 736 | 62 | — |
| advocates | 285 | 494 | 62 | — |
| scry | 83 | 175 | 22 | 83 |

### Why

`internal/graph/builder.go:181` (`extractCodeNodes`) keeps a symbol only when
`classifySymbolKind(sym.Kind)` returns a non-empty type, and that switch matches
`function`, `method`, `class`, `struct`, `interface`, `module`. The code store
records `Kind` from SCIP's `SymbolInformation.kind`, which neither scip-go nor
scip-typescript populates:

```
$ scry defs AdmitAlias   → kind "UnspecifiedKind"
$ scry defs Build        → kind "UnspecifiedKind" / "External"
$ scry defs Store        → kind "External" / "UnspecifiedKind"
```

So every symbol falls through the `default` case and no code node is written.
`extractCallEdges` then finds no caller keys in `nodeSet` and writes no call
edges. What survives is git (authors, co-change edges from the top-500 hotspots)
and schema tables. Louvain runs over that, and the "architecture report" is a
churn report.

This has been true since the graph shipped. Nothing caught it because the
report still returns plausible-looking file clusters, and because the only
consumer is an agent that cannot tell a co-change community from a call-graph
community.

### Fix

Derive the kind from the SCIP symbol descriptor instead of the optional kind
field. The grammar is fixed: a trailing `().` is a method or function, `#` a
type, `.` a term, `/` a package or namespace. Ten lines in `classifySymbolKind`
or a `KindFromSymbol` helper in the scip parser, plus a rebuild of every graph.
Then `scry_graph_path` between a function and a table means something, and the
`surprising_edges` list stops being null.

Measure after: node counts per type per repo, and one path query per language
(Go function → Go interface, TS component → D1 table, PHP controller → Eloquent
model) that today returns nothing.

---

## 3. The memory store, measured today

### 3.1 Shape

| Measure | 2026-09-02 audit | Today | Note |
|---|---|---|---|
| Entities | 18,945 | 25,081 | +32% in two days; most of it fleet-run chatter |
| Facts | 30,301 | 64,192 | |
| Episodes | 3,615 | 7,786 | kimi and opencode now flow |
| `concept` share | 57% | 51% | still the plurality bucket |
| Entities with no aliases | 11,024 | 15,094 | |
| Entities with no description | — | 4,299 | |
| Slugs of four characters or fewer | — | 350 | `add`, `api`, `auth`, `b1`…`b6`, `at1`…`at6` |
| Most aliases | childscribe-laravel 453 | vitest 118, childscribe-laravel 92, scry 77, wrangler 72 | pruned, still fused |

### 3.2 A random sample of 80 entities

Eighty entities drawn with a fixed seed, each looked up for current facts.

| Finding | Count |
|---|---|
| Hold zero current facts | 11 (14%) |
| Typed `concept` with an empty description | 12 |
| Mangled file paths (`apphttpcontrollersapirealtimeinterviewtoolcontrollerphp55-56`, `resourcesjspagescontentreviewindexvue`) | 4 |
| Fleet agents or tasks typed `person` or `project` (`claude-repair-surcharges-and-fees` is a person with 22 facts; `request-entry-and-mounts` is a project with 25) | 3 |
| Wrong type on a real thing (`apisrcoperationscomplete-taskts` is a `machine`) | 1 |
| Single English words as entities (`required`, `attached`) | 2 |

Store-wide, by heuristic: ~898 entities are file paths with the punctuation
stripped, and 191 of the 291 `person` entities are agents, graders, reviewers,
or task workers. The extractor sees a fleet room transcript and types every
named worker as a person and every task contract as a project.

Relations in the sample are healthy: `uses`, `status`, `related_to`,
`depends_on`, `implements`, `documents` at the top; the closed vocabulary holds.
`related_to` is 7% of edges, which is the price of the fallback.

### 3.3 Retrieval: the five benchmark files, run today

| Set | Score | Mean answer rank | Names the entity |
|---|---|---|---|
| probes | 7/7 | 1.0 | yes |
| tuning | 47/50 (94%) | 4.0 | 98% |
| tuning-strict | 44/50 (88%) | 4.2 | 98% |
| heldout-2026-09-03 | 53/62 (85%) | 5.0 | 88% |
| heldout-b | 34/66 (52%) | 6.2 | 59% |

Latency 65–97 ms mean, max payload 13 KB, nothing over the cap. These match the
handoff exactly; nothing regressed and nothing improved since. None of the five
is genuinely held out anymore; the handoff says so and it stands.

### 3.4 Twelve questions an autonomous engineer would actually ask

Run through `scry memory recall`, judged on whether the top four facts answer
the question.

| Question | Result | What came back |
|---|---|---|
| how do I deploy scry to the mini | good | launchd, GOBIN gotcha, hardened binary, Go version — four relevant facts |
| what is hermes | good | gateway on the mini under launchd, both platforms connected |
| what did we decide about keeping episode transcripts | excellent | the decision, the cost, whose call it is |
| which model chain does memory extraction use | shallow | "runs an ordered model chain" ×3 before the chain is named |
| what is loom | good | and `loom -replaced_by-> setpoint` is right there |
| what is cockpit | good | |
| what is the fleet room protocol | good | |
| how does childscribe get deployed to production | contradictory | "NOT deployed" and "deployed at e9060c4" both current, no dates in the sentences |
| the scheduled job that ingests transcripts every half hour | partial | reached `sweep-agent`, and surfaced a **false** fact: "Claude Code SessionEnd hook fires on every session close" — no such hook exists |
| why does scry forbid CGO | miss | four facts about other things that "forbid" |
| what broke the last time we changed the alias rules | miss | top hit is a junk entity `lasttime` |
| what does the pre-search hook do | miss | flooded by another project's `global-search` |

Eight of twelve are usable, which matches heldout-b within noise. The four
misses fall into three families that ranking cannot fix:

1. **The fact was never extracted.** "No CGO" lives in `CLAUDE.md` and
   `DECISIONS.md`, which nothing ingests. Memory only knows what an assistant
   said out loud in a transcript.
2. **The question describes rather than names.** "the scheduled job that
   ingests…" is `scry-memory-sweep`; the asker did not know that.
3. **A generic word lands on the loudest entity that carries it.** "hook" and
   "search" belong to a Cloudflare project with 1,800 facts.

### 3.5 Time

Only two relations are exclusive:

```go
var DefaultExclusive = map[string]bool{"status": true, "replaced_by": true}
```

`deployed_on` was removed from the set after it invalidated 497 true facts
(staging and production are both true). The consequence is that every entity
accumulates current state it has moved past:

| Entity | Current facts | Invalidated | Current `deployed_on` |
|---|---|---|---|
| childscribe-laravel | 1,558 | 521 | 166 |
| docket | 1,810 | — | 37 |
| mac-mini | 199 | 7 | 51 |
| scry | 1,658 | — | 18 |

`--as-of` works, and it works well on the slice that gets invalidated. On the
rest, the question "what is true now" is answered by a pile the agent has to
sort by `valid_from` itself, and the fact sentences rarely carry the date.

The "never relaunch the docket campaign" rule is a clean example: recall
returns "Campaign relaunched at wave 33" and "Relaunched campaign at 18:38
without checking tree" above "Jeff decided to park the Docket parity campaign."
All three are current. The rule is the third fact and the newest, and nothing
in the ranking prefers it.

### 3.6 The pipeline

- **Backlog.** 2,037 held at 12:57, 1,162 at 16:20: about 255 episodes an hour,
  so the queue clears by evening. Transcript facts arrive three to twenty hours
  after the session. Manual remembers skip the line and arrive in minutes.
- **A permanent error retried as a transient one.** Two items are on attempt
  220 and 464 with a two-minute backoff, failing on
  `memory: alias already claimed: "claude-migration-registry-consolidation" belongs to claude`.
  That is a resolver verdict, not a transport failure. They will retry forever.
- **Halving.** One 36,773-character episode was split after 589 timeouts.
  Whatever the count means, 589 attempts before splitting is not the intended
  budget.
- **Cost of thinking.** GLM-5.3-Flash cannot turn thinking off and takes one to
  six minutes per episode. Every fleet run generates dozens of episodes whose
  content is contract negotiation and status posts.

### 3.7 What memory never sees

| Source | Swept | Effect |
|---|---|---|
| Assistant and user turns in Claude, Codex, Kimi, OpenCode transcripts | yes | the whole store |
| Tool results (test output, errors, file contents) | no, by design | "what broke" facts exist only when the assistant narrated it |
| Tool call bodies (the room post an agent wrote, the file it edited) | no, only the tool name | a fleet's contracts and verdicts survive only as prose restatements |
| scry rooms (245 rooms, 5,859 posts, 60k reads) | no | the highest-signal coordination text on the machine has no path into memory |
| `~/.claude/projects/*/memory/*.md` (285 files, 259 index lines, 161 typed `project`, 22 `feedback`) | seeded once, never again | the curated layer agents maintain by hand is invisible past the seed date |
| `CLAUDE.md`, `AGENTS.md`, `docs/DECISIONS.md`, `README.md` | no | "why no CGO" is a miss; every hard constraint is |
| loom / setpoint runs | yes | |

---

## 4. How agents actually use it

From `~/.scry/logs/mcp-calls.jsonl`, 2026-04-20 to today, 133,682 calls:

| Tool | Calls | Since 2026-09-02 |
|---|---|---|
| scry_read (rooms) | 63,814 | |
| scry_task_list (rooms) | 59,973 | |
| scry_post (rooms) | 5,859 | |
| **scry_remember** | 613 | 339 |
| **scry_recall** | 390 | 207 |
| scry_episodes | 22 | |
| **scry_memory_path** | **0** | |
| scry_defs + scry_refs | 57 | |
| scry_graph_report + scry_graph_query | 30 | |

Read that table as a workflow description:

- Rooms are the working memory of every autonomous run and they are used
  constantly. Long-term memory is consulted ~three times a day, and half of
  those calls in the last three days were the memory-solid run measuring
  itself.
- `scry_memory_path` has never been called by anyone. The "how does A relate
  to B" capability the spec led with has no consumer.
- The code tools are called less than the memory tools. Agents are still
  grepping; the pre-search hook nudges but the numbers say it rarely lands.
- The recall log records `results: 0` on every call, so hit rate in the wild
  cannot be measured from the log.

And the wiring:

- `~/.claude/settings.json` hooks: `PreToolUse` (pre-search, pre-git, cockpit
  status), `Notification`, `Stop`, `UserPromptSubmit`. **No `SessionStart`, no
  `SessionEnd`.** `orient` is never injected. `ingest` runs only from the
  30-minute sweep. The stored fact that says the SessionEnd hook is active is
  wrong and has never been invalidated.
- Two memory systems run side by side. Claude Code's file memory
  (`memory/*.md` + `MEMORY.md` index, loaded every session) is what an agent
  actually reads at start. Scry memory is what it is told to query. The docket
  `MEMORY.md` is 33 lines of exactly the facts orient should produce: the
  deploy command, the gate's coverage trap, "never `git add -A` here," "check
  reflog before resetting main," "do not relaunch the campaign." Scry's orient
  for docket returns five recently-touched entities about migration blocks and
  the overage engine.

This is the gap that matters. The store is better than its usage.

---

## 5. What the memory-solid run got right, kept

Not relitigated here, all stand:

- Facts invalidated, never deleted; every store write behind a backup; replica
  first; a hostile grader before apply.
- Transcripts stay on the model chain; no hosted embeddings; no transcript
  retention (measured: worse at every depth).
- Structure in resolver code with table tests, not prompt wording. The `value`
  type experiment proved the model cannot be trusted with the judgement.
- The closed 39-relation vocabulary.
- Recall ranks facts, caps payload at 24 KB, and demotes restatements.
- No seventh lexical ranking pass. The ceiling is retrieval coverage, and a
  synonym table fitted to a question set is negatively correlated with success
  on the next set.
- Store-scale rules that infer an owner from a sentence are wrong in bulk.
  Repairs go through reviewed lists (`reattach`, `unalias`,
  `merge-entities`).

---

## 6. The plan

Ordered by value per hour. Each item names its measure. P0 is a day's work in
total and moves the grades in §1 more than the last thirteen resolver rounds
did.

### P0 — wiring and coverage (do first, in this order)

**1. Put code back in the code graph.** Derive symbol kind from the SCIP
descriptor suffix; rebuild every graph. Measure: function/class node counts per
repo are non-zero; one cross-domain path per language resolves.
*Half a day.*

**2. Wire the two hooks the spec specified.** `SessionStart` runs
`scry memory orient --cwd $PWD --budget 500` as `additionalContext`;
`SessionEnd` runs `scry memory ingest --source claude --path <transcript> &`.
Both in `~/dotfiles/claude/settings.json` so they sync. Then invalidate the
false "hook is active" fact. Measure: every Claude session shows an orient
block; transcript-to-fact latency drops from hours to minutes for Claude.
*One hour.*

**3. Stop retrying permanent errors.** `ErrAliasClaimed` and every other
resolver verdict parks the item with its reason; only transport and provider
errors back off. Cap the timeout count before halving. Measure: `queue_parked`
carries a reason field; no item has `attempts > 10`. *One hour.*

**4. Sweep the curated memory files.** Add `~/.claude/projects/*/memory/*.md`
as a `note` source with per-file cursors, ingested like a manual remember with
`cwd` set from the project path. These 285 files are the most carefully
written facts on the machine and the only place feedback-type facts live.
Measure: recall for "never relaunch the docket campaign" returns the rule at
rank 1. *Two hours.*

**5. Count recall results in the call log.** The `results` field is zero on
every recall. Log fact count and top score. Measure: the field is populated;
a weekly "recall calls with zero facts" number exists. *Thirty minutes.*

### P1 — make the write path carry structure

**6. Structured remember.** `scry_remember` today sends prose through the same
extractor that types a fleet worker as a person. An agent writing a decision
already knows the subject, the kind, and what it replaces. Add optional typed
fields: `subject` (entity name), `kind` (`decision | runbook | gotcha | state |
preference`), `supersedes` (a fact id or a `(subject, relation)` pair),
`project` (repo path). When present, the daemon writes the fact directly with
`confidence 1.0` and the extractor only runs to pull secondary entities. Then
update the MCP tool description and the global `CLAUDE.md` block so agents use
the fields. Measure: share of manual episodes whose primary fact is typed
`concept` falls from today's majority to near zero; remember-to-fact latency
for the primary fact is the RPC latency. *One day.*

**7. Ingest rooms on close.** A closed room's posts, in order, are one
episode with `source: room`, `cwd` from the room's repo, and the room id as
provenance. Contracts, review verdicts, and handoffs stop depending on whether
an agent restated them in prose. Add a `room` to `EpisodesBySource` and to
doctor. Measure: the memory-solid run's own room (34 posts) produces facts that
recall finds by room id. *Half a day.*

**8. Ingest project docs, deterministically scoped.** A per-repo allowlist
(`CLAUDE.md`, `AGENTS.md`, `README.md`, `docs/DECISIONS.md`, `docs/*_SPEC.md`),
chunked by heading, one episode per heading, re-ingested when the blob hash
changes. This is the one place LLM extraction over prose earns its cost: the
text is already decisions. Measure: "why does scry forbid CGO" returns the
constraint at rank 1; the same for three constraints from other repos. *Half a
day.*

**9. An `agent` entity type.** The extractor and resolver both learn that a
name matching the fleet worker pattern (`claude-*`, `codex-*`, `*-reviewer`,
`*-grader`, `grade-item-*`, room task titles) is an `agent`, never a person or a
project. Retire the 191 existing mis-typings through `merge-entities`. Measure:
`person` entities drop to the ~100 that are people. *Half a day plus a reviewed
migration.*

### P2 — retrieval for askers who do not know the name

The honest ceiling is 52% and no ranking change moves it. Three things that
might, cheapest first, each measured on a **fresh** held-out set written by
someone who has not read `query.go`:

**10. Two-hop recall in the tool contract.** Zero code. The `scry_recall`
description tells the agent: if the top facts do not answer, take the
`entities` header from the first response and re-ask naming the most likely
one. The grader measured that 18 of 27 "unreachable" misses come back at rank
≤20 once the entity is named. Measure: heldout-b re-run with a two-call
protocol. *One hour, then measure.*

**11. Prefer the newest current fact when several share `(src, relation)`.**
A ranking rule, not an invalidation rule: when a question is state-shaped
("is", "now", "current", "deployed", "status") and candidates share subject
and relation, lift the newest and attach `valid_from` to the sentence. This
addresses the childscribe and docket contradictions without touching
exclusivity. Measure: the state-shaped questions in the fresh set. *Half a
day.*

**12. A real local embedding model, if 10 and 11 leave a gap.** The house rules
allow local embeddings; the blocker is CGO. Pure-Go BERT inference exists
(nlpodyssey's cybertron runs MiniLM-class models without cgo). Before writing
any of it into recall, run it through the existing offline harness
(`transcript_corpus_experiment_test.go` pattern) against heldout-b and report
the same table the random-indexing model got. If it does not beat 34/66 at
top-20 by a margin, stop. Binary size and a downloaded model file are the costs
to record in `DECISIONS.md`. *Two days to measure; do not build the integration
first.*

### P3 — identity, the unglamorous rest

**13. Use `merge-entities` on the 41 reviewed cross-type groups**, applying the
reviewer's filtered subset (`rev_subset.json`) rather than the full proposal,
and correcting the four type choices the reviewer flagged. Then the ~40
misfiled `hermes-ops` facts, one sentence at a time. Measure: cross-type
collisions with facts on both sides fall; `qwen3-8-27b-uncensored-q5` no longer
exists as a hollow twin.

**14. Path-shaped names keep their path.** ~900 entities are file paths with
the punctuation stripped. The resolver should recognise a path by shape before
slugging, keep the original as the name, slug from the basename, and set a
`path` attribute. Retire the existing mangled ones through a reviewed list.
Measure: `scry memory entities` has no slug matching the mangled-path
heuristic.

**15. Fleet-run noise as a first-class concern.** The store grew by 6,136
entities in two days, and the random sample says fleet task contracts, workers,
and room posts are a large share of it. Before every room-driven run, the sweep should know
the run's worktree and task names and treat facts scoped entirely to one task
contract as episode-local unless a second episode cites them. This is the
"attested by more than one episode" rule from the audit's improvement 5,
applied to entities rather than aliases. Measure: entity growth per fleet run;
share of entities with exactly one citing episode.

### What I would not do

- Another round of lexical value rules. The handoff's conclusion stands: the
  same string is a value in one episode and a name in another.
- Retain transcripts. Measured worse.
- Widen `DefaultExclusive`. It was tried and it invalidated true facts. Fix
  presentation (item 11) instead.
- Tune ranking against any of the five existing question files. Write a new
  one first.

---

## 7. The target workflow

What a day looks like once P0 and P1 land.

```
session start   SessionStart hook → orient(cwd), ≤500 tokens:
                  standing rules for this repo (from notes + docs sources),
                  last three decisions, open blockers, deploy one-liner.
during work     unknown referent → scry_recall, two-hop if the first pass
                  returns entities but no answer.
                code question → scry_graph_* with real code nodes.
decision made   scry_remember with subject/kind/supersedes; lands in ms.
fleet run       room posts are the working memory; room close → episode.
session end     SessionEnd hook → ingest; facts inside minutes.
every 30 min    sweep covers Codex/Kimi/OpenCode/notes/docs; doctor watches age.
nightly         hygiene report in doctor: hollow entities, path-shaped names,
                  agents typed person, queue items over N attempts, growth.
weekly          fresh held-out bench, both strict and loose, appended to the
                  audit with the store size beside the score.
```

## 8. Numbers to keep on the wall

Six, published by `scry doctor --json` and appended weekly to the audit:

| Number | Today | Direction |
|---|---|---|
| Code nodes in the unified graph, all repos | 0 | up |
| Recall top-20 on the current fresh held-out set (strict) | none exists | write it, then up |
| Sessions with an orient block injected | 0% | 100% |
| `concept` share of entities | 51% | down |
| Entities with zero current facts | ~14% | down |
| Oldest queue item age / max attempts | ~2 days / 464 | hours / ≤10 |

---

## Appendix: how this was measured

- Store counts: `scry memory status --pretty` at 16:20 EDT.
- Entity sample: `scry memory entities` dumped to a file, 80 drawn with
  `random.seed(7)`, each passed to `scry memory facts <slug>`.
- Invalidated counts: `scry memory facts <slug> --all`, counting `invalid_at`.
- Benchmarks: `scry memory bench --file docs/memory-bench/<set>.json --top 20`
  against the live daemon.
- Probe questions: `scry memory recall "<q>"`, top four facts judged by hand.
- Tool usage: `~/.scry/logs/mcp-calls.jsonl`, 133,682 lines, first 2026-04-20.
- Graph: `scry graph report --repo <path>` and the per-repo
  `~/.scry/repos/*/graph/manifest.json`; kinds from `scry defs <symbol>`.
- Hooks: `~/.claude/settings.json` and `~/dotfiles/claude/settings.json`.
- Queue: `scry memory queue --pretty` and the mini's `scryd-launchd.log`.

Measuring this system pollutes it: the recall probes above are now in the
transcript this session will be extracted from. Expect entities named after
the questions.
