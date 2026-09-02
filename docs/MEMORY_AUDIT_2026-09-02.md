# Memory domain audit, 2026-09-02

**Status:** findings only. Nothing in this document has been fixed yet. The
goal file that acts on it lives at
`~/dotfiles/ai/prompts/2026-09-02-scry-memory-solid.md`.

**Question asked:** agents (Claude Code, Codex, Kimi, OpenCode) do not seem
to be storing enough in the graph, and `scry_recall` does not seem to return
the right facts. Is the graph good enough, and where does it need work?

**Short answer:** two things are broken outright (ingestion on the laptop is
dead, and a failed `scry_remember` loses the fact), and underneath them recall
has a structural ceiling: it finds entities by substring and dumps every fact
on them, with no ranking and no search over fact text. The temporal graph
itself (provenance, invalidation, as-of) works. The retrieval layer and the
entity-resolution rules are what need building.

## How this was measured

- Store: the shared store on the Mac mini, exported through the daemon's
  live UI (`http://127.0.0.1:7279/data.json` on the mini) and via
  `scry memory entities` / `scry memory recall` over the tunnel socket
  `~/.scry/shared-memory.sock`.
- Call traffic: `~/.scry/logs/mcp-calls.jsonl` on the laptop (133,082 lines,
  written by every `scry mcp` process, so it covers Claude, Codex, Kimi and
  OpenCode calls that go through this machine).
- Sweep: `/tmp/scry-memory-sweep.log` on the laptop (27 MB).
- Daemon: `~/.scry/logs/scryd-launchd.log` on the mini.
- Code: `internal/memory/recall/recall.go`, `internal/memory/resolve/resolve.go`,
  `internal/daemon/memory_methods.go`, `internal/rpc/rpc.go`,
  `internal/memory/extract/haiku.go`, `cmd/scry/memory.go`.

## What has been worked on recently

Every memory commit since late August was write-path hardening:

| Date | Commit | Change |
|---|---|---|
| 08-28 | 801d14b | Shared memory daemon socket (one store on the mini, tunnel from the laptop) |
| 08-28 | 11eab5e | Ordered extraction model chain in `~/.scry/config.yaml` |
| 08-31 | 9ef369a | Keep the fact when a model invents a type |
| 09-01 | 7fc82ec | Stop losing memory writes to reasoning (thinking ate the output budget) |
| 09-01 | 816e18a | Make disabling thinking best effort (Z.ai rejects the field) |

Nothing has touched recall ranking, entity resolution, relation vocabulary, or
coverage of other agents since the domain shipped on 2026-07-28.

## Topology as deployed

- The store lives on the mini at `/Users/jclaw/.scry/memory`, served by the
  launchd agent `ai.jermes.scryd`, binary `scry-816e18a`.
- The laptop reaches it through `com.jhoot.scry-memory-tunnel`, an SSH
  forward from `~/.scry/shared-memory.sock` to the mini's `scryd.sock`.
- Every MCP host on the laptop (Claude Code via `~/.claude.json`, Codex via
  `~/.codex/config.toml`, OpenCode via `dotfiles/opencode/opencode.json`,
  Kimi via `~/.kimi-code/mcp.json`) registers `scry-memory` with
  `SCRY_MEMORY_SOCKET` pointing at the tunnel socket.
- `com.jhoot.scry-memory-sweep` runs `scry memory sweep` on the laptop every
  30 minutes. The sweep distills and extracts on the laptop, using the
  laptop's `~/.scry/config.yaml`, and commits results to the mini over the
  tunnel.
- The mini's config chain is `glm-5.3-flash` (Z.ai) then `deepseek-v4-flash`.
  The laptop's config chain is still `deepseek-v4-flash` then
  `deepseek-v4-pro`. `Z_AI_API_KEY` is already exported in the laptop's
  `~/.secrets.zsh`. The legacy `SCRY_MEMORY_API_KEY`, `SCRY_MEMORY_MODEL` and
  `SCRY_MEMORY_BASE_URL` exports are also still there; the daemon ignores
  them when `config.yaml` has a `memory.models` list, and logs that it does.
- The spec's `SessionStart` orient hook and `SessionEnd` ingest hook were
  never wired into `~/.claude/settings.json`. Only `pre-search`, `pre-git`
  and cockpit hooks exist. Claude sessions reach memory through the sweep
  and explicit `scry_remember` only.

## Finding 1: laptop ingestion has been dead since 2026-09-01 13:34

DeepSeek returned `402 Insufficient Balance` starting 2026-09-01. The mini's
config was switched to GLM that afternoon. The laptop's was not. Because the
sweep extracts client-side, every laptop sweep since then fails on every
new transcript.

Per run, from the sweep log (44 runs logged since 08-31):

| Measure | Typical value |
|---|---|
| Files scanned | 2,021 to 2,026 |
| Files ingested | 0 (last non-zero: 1 file at 09-01 13:34) |
| Errors: 402 from DeepSeek | about 475 |
| Errors: `write unix ... i/o timeout` | 380 |
| Errors: other | about 90 |
| Lines of 402 noise in the log | 12,696 |

The 380 socket timeouts are not a tunnel or daemon fault. The sweep's whole
run shares one 30-minute context (`cmd/scry/memory.go`, sweep command). The
402 attempts burn the budget, the deadline passes, and every remaining cursor
lookup fails instantly with an i/o timeout on the deadline-bound connection.

Consequence: no Claude or Codex transcript from 09-01 onward is in the graph.
The 959 episodes ingested on 09-01 came from the mini's own backfill, not the
laptop.

## Finding 2: a failed `scry_remember` loses the fact

`handleMemoryRemember` in `internal/daemon/memory_methods.go` builds the
manual episode, then calls the extractor, then `resolve.Apply`. The episode
is only persisted by `Apply` (Rule 7) after a successful extraction. The
comment on the error path says "The episode is already stored"; it is not,
except on the dormant path. A provider error, a timeout, or any non-parse
failure returns an error with nothing written. Only `extract.ErrParse`
produces a dead-letter file.

Compounding it, remember latency is extraction-bound because the provider
chain since 08-06 reasons before answering:

| Period | remember p50 | Worst | Calls over 60 s |
|---|---|---|---|
| 07-29 to 08-05 (Haiku) | 5 to 8 s | 13 s | 0 of 30 |
| 08-06 to 09-02 (DeepSeek, then GLM) | 40 to 130 s | 607 s | 82 of 246 |

The remember call that recorded this audit ran past the 120 s client
threshold and was backgrounded. Codex's default tool timeout is 60 s. The RPC
server (`internal/rpc/rpc.go`, `serveConn`) hands handlers the server's root
context, not a per-connection one, so the daemon finishes the write after
the client has given up; the agent sees a failure and either drops the fact
or retries and stores a duplicate episode (the episode id includes the wall
clock, so retries never dedupe).

Remember does not use the store glossary at all: the handler passes the
caller's entity hints as the glossary. Latency is purely provider reasoning
time on a short prompt, not glossary size.

## Finding 3: recall is entity substring match with no fact ranking

`recall.Query` tokenises the query into 3+ character tokens, matches each as
a substring against every entity's slug, name and aliases (a full scan of
18,945 entities), ranks exact matches above substring matches, takes the top
five entities, and returns every current fact on each of them. Fact text is
never searched. Nothing caps the payload.

Probe results against the live store (limit 5, defaults):

| Query | Entities matched | Facts returned | Payload | Why |
|---|---|---|---|---|
| `hermes deploy` | 5 | 3,434 | 1.18 MB | `deploy` matched an alias of childscribe-laravel (2,668 facts) |
| `Z_AI_API_KEY` | 5 | 1,864 | 635 KB | `key` matched an alias of the Jeff entity (1,847 facts) |
| `memory` | 5 | 798 | 293 KB | Memory Book, scry, Operations suite |
| `scry` | 5 | 550 | 203 KB | scry alone has 540 facts |
| `why did we switch off deepseek` | 5 | 241 | 92 KB | matched a Qwen entity and a 10 GbE switch |
| `cockpit` | 5 | 95 | 43 KB | the good case |
| `GLM-5.3-Flash` | 5 | 658 | 241 KB | also matched childscribe-engine-core (477 facts) |

MCP hosts truncate results this size. The agent sees an arbitrary slice,
often not the fact it needed, and concludes the graph does not know.
`scry_recall` also has no `limit` on facts, only on entities, so the
`limit` parameter cannot fix this.

## Finding 4: graph shape

Store contents on 2026-09-02:

| Measure | Value |
|---|---|
| Entities | 18,945 |
| Facts total / current / invalidated | 30,301 / 27,346 / 2,955 |
| Episodes | 3,615 |
| Episodes by source | claude-session 2,313; codex-session 878; manual 222; seed 117; loom-run 85; kimi 0; opencode 0 |
| Current facts by source | claude 19,603; codex 4,897; manual 1,285; seed 984; loom 750 |
| Cursors | 1,732 |

Entity degree (current facts touching the entity):

| Bucket | Entities |
|---|---|
| 0 facts | 2,648 |
| exactly 1 fact | 10,136 |
| 5 or more facts | 1,779 |

Two thirds of the graph is stubs and leaves. Entity types: concept 10,822;
tool 3,055; project 1,592; service 1,391; decision 1,124; runbook 482;
machine 329; person 150. 11,024 entities have no aliases; 45 have twenty or
more; childscribe-laravel has 453.

Highest-degree nodes: childscribe-laravel 2,838; jeff 1,894; hermes-ops 657;
codex-reviewer 595; scry 559; childscribe-engine-core 482; childscribe-mobile
414; wren-home-cleaning 386; **main 374** (the git branch); claude-code 329;
operations-north-star 313; **in-progress 241** (a status value).

Relations: 5,586 distinct names across 27,346 current facts. The top of the
list is sane (`status` 3,151; `uses` 2,500; `depends_on` 1,371; `blocked_by`
1,035; `decided` 700; `deployed_on` 532) but the tail is unbounded:
`has_status`, `has`, `contains`, `quant_of`, `faster_than`, `launched_by`,
`stored_at`, `measured_on`, `has_monthly_credit_for`, and thousands more.
Path traversal over an uncontrolled vocabulary is close to meaningless.

Confidence is uninformative: 26,091 facts are at 0.9 or above, 34 below 0.6.

## Finding 5: entity resolution merges distinct things

Alias merging in `resolve` accepts whatever alias the extractor emits and
merges on first sight. Observed results:

- `hermes-ops` (type project) carries the aliases `Hermes`, `mac-mini`,
  `mini`, `Mac Mini`, `Helm`, `HelmTerminal`. The agent, the machine, the
  terminal app, and the ops project are one node.
- `qwen38-27b-uncensored-q8` (type tool) carries `gpt-oss-120b`,
  `gpt-oss-120b-Q4_K_M`, `box2-gpt-oss-120b`, `gpt-oss`, `oss-120b`. The
  2026-09-02 13:34 remember about gpt-oss-120b on halo2 was written as facts
  about the Qwen model. That is a wrong answer at recall time.
- `in-progress` (type concept) carries `partial`, `product surface`,
  `voice-of-customer`.
- Value entities: `51b-active-parameters`, `46-gib-spare-memory`,
  `places-per-request-pricing`, plus `main`. Numbers, status values and
  branch names become nodes.
- Self-loops exist (`glm-53-flash-ud-q2-k-xl -[status]-> glm-53-flash-ud-q2-k-xl`).

Hygiene rules already in `resolve.go` (`isEphemeralName`,
`isGenericEntityName`, `isGenericAlias`) catch run artifacts and generic
names but do not gate merges on evidence or type.

## Finding 6: what agents actually store

Manual episodes (from `scry_remember`): 222 stored on the mini. The laptop
MCP log shows 276 remember calls in the same window; the gap is the
write-loss in Finding 2 plus pre-08-28 calls that went to the laptop-local
store before the socket was shared. 38 of 222 manual episodes produced zero
facts. Average 5.85 facts per manual episode. Summary length p50 470
characters, p90 851, max 1,468: agents write paragraphs, and extraction
decomposes them reasonably when it runs.

Kimi and OpenCode contribute only through explicit remembers. There is no
distiller for `~/.kimi-code/sessions` or OpenCode's session store, so the
sweep never sees them.

## RAG versus graph, and the verdict

RAG retrieves text chunks by similarity and lets the model reason over
them. It is robust to vocabulary mismatch and needs no extraction, but it
returns passages rather than facts, cannot answer "what changed since" or
"how does A relate to B", and dedupes nothing. A knowledge graph stores
extracted entities and typed, time-stamped facts with provenance and
invalidation, but is only as good as extraction and entity resolution, and
needs its own retrieval layer to find the right node. Working systems
(Graphiti, Zep) are hybrids: a temporal graph for structure plus lexical or
embedding search over entity names and fact text for entry points.

Scry today is a graph with a lexical entry-point finder and no fact-level
retrieval. The graph half works. The retrieval half and the resolution rules
are the missing pieces, and both are bounded work.

## Improvements, in priority order

1. **Unbreak ingestion.** Align the laptop's model chain with the mini's, or
   move extraction into the daemon so there is one config to keep correct.
   Give the sweep a per-file deadline instead of one 30-minute context. Make
   "hours since last successful ingest" a doctor check that fails.
2. **Make remember durable.** Persist the manual episode before extraction,
   return in milliseconds, resolve facts asynchronously with retry from a
   queue. Provider outages defer facts instead of losing them, and client
   timeouts stop mattering.
3. **Add fact-level retrieval.** Index fact text and entity names (BM25 or
   local embeddings; no new third party), rank facts not entities, cap the
   payload, return matched entities as headers.
4. **Constrain the relation vocabulary and reject value entities.** A fixed
   set of 20 to 40 relations with a mapping table; numbers, status values,
   and branch names never become entities. Resolver rules with table tests,
   not prompt wording.
5. **Tighten alias merging.** An alias must be attested by more than one
   episode before it can merge two existing entities; never merge across
   types like machine and project. Run hygiene to split hermes-ops and the
   Qwen entity.
6. **Cover Kimi and OpenCode.** Distillers for their session stores, ingested
   by the same sweep.

---

## Re-measurement, 2026-09-02 evening (after the run)

Appended, not overwritten: the findings above are the "before". Everything
below was measured against the live shared store on the mini and the real
logs after PR #7 (`9fd4385`) was deployed to both machines, plus the queue
tuning that followed. Reproduce with the commands shown.

### Store

| Measure | Before | After |
|---|---|---|
| Entities | 18,945 | 18,037 (917 value-named entities retired) |
| Facts (total) | 30,301 | 30,335 → growing again as the queue drains |
| Distinct relations on current facts | 5,586 | 39 (`scry memory migrate` dry run: `non_canonical_after: 0`) |
| Attribute facts (value targets, not nodes) | 0 | 7,705 converted at migration |
| Cross-type alias collisions | not measured; hermes-ops alone carried 130 aliases | 0 (`scry memory hygiene` dry run) |
| Self-loops | present | 726 invalidated |
| Backup before migration | — | `/Users/jclaw/.scry/backups/memory-20260902T230320Z.badger`, 62.9 MB |

Migration report (`scry memory migrate --apply`, 19.5 s on the mini): 30,335
facts scanned, 15,233 relations rewritten (1,317 flipped to the canonical
direction, 1,561 on `related_to`), 917 value entities retired with 7,705
facts converted and 154 value-to-value facts invalidated, 3,464 reference
and generic aliases dropped, 4,012 aliases split away from entities of
another type or bearing another entity's name, 4,520 facts reattached
across type boundaries, hygiene converged in 2 passes. A second run
reports zero changes.

Relation distribution after migration (current facts): status 6,450; uses
3,055; contains 2,178; related_to 1,571; depends_on 1,484; blocked_by 1,378;
decided 1,033; tests 955; deployed_on 950; documents 920; implements 878;
fixes 662; lacks 656; merged_into 653; owns 555; located_at 543; requires
521; has_issue 518; produces 455; modifies 452; provides 451; replaced_by
447; reviews 387; assigned_to 349; calls 324; causes 302; passes 296;
runs_on 274; enforces 232; approves 215; part_of 205; monitors 196; targets
191; excludes 175; configures 139; conflicts_with 137; references 74;
same_as 56; notifies 18.

Identities: `hermes-ops` (project) keeps 195 facts; `hermes` (service) 194;
`mac-mini` (machine) 68, up from 7; `amd-halo` 11; `gpt-oss-120b` (tool)
23 as its own entity; the Qwen model carries no gpt-oss alias. `mini`
resolves to `mac-mini`, `Hermes` to `hermes`, `gpt-oss-120b` to itself.

### Ingestion

| Measure | Before | After |
|---|---|---|
| Laptop sweep files ingested | 0 since 09-01 13:34 | 1,090 of 2,157 scanned, 4,773 episodes queued, `Errors: null` (first run of the new agent, 23:04) |
| Mini sweep | none existed | 47 of 47 files, 24 episodes, no errors (`ai.jermes.scry-memory-sweep`) |
| 402 lines in the sweep log | ~475 per run | 0 (the sweep no longer calls a provider) |
| Socket timeouts per run | 380 | 0 (per-file 2-minute deadlines) |
| Places the chain is configured | 2 (laptop and mini config.yaml, diverged) | 1 (mini `config.yaml`; the laptop's names only `memory.socket`) |
| Sources swept | claude, codex, loom | claude, codex, kimi, opencode, loom |
| `scry doctor` | no memory checks | Memory section: daemon reachable, chain + worker, hours since last ingest (fails past 6h), last sweep, queue |

### Remember

Twenty `scry_remember` calls through the real `scry mcp --profile memory`
server from the laptop, as recorded in `~/.scry/logs/mcp-calls.jsonl`:

| Measure | Before | After |
|---|---|---|
| p50 | 40–130 s | 183 ms |
| p95 | — | 277 ms |
| max | 607 s | 420 ms |
| Behaviour on provider failure | fact lost unless the failure was a parse error | queued on disk; retried with backoff; parked only after three unparseable replies or three timeouts, replayable |

### Recall

The seven audit probes against the live store (`scry memory bench --file
docs/memory-bench/probes.json --top 5`): 7 of 7 place a fact from the
intended entity in the top five; mean payload 4.2 KB, max 4.8 KB; mean
answer rank 1.7.

| Query | Before (facts / payload) | After (payload, answer rank) |
|---|---|---|
| `hermes deploy` | 3,434 / 1.18 MB | 9.5 KB, hermes-ops fact in top 5 |
| `Z_AI_API_KEY` | 1,864 / 635 KB | 9.0 KB, the key's fact in top 5 |
| `memory` | 798 / 293 KB | 8.7 KB |
| `scry` | 550 / 203 KB | 9.4 KB |
| `why did we switch off deepseek` | 241 / 92 KB (Qwen, a 10 GbE switch) | 9.6 KB, the 402 decision in top 5 |
| `cockpit` | 95 / 43 KB | 9.0 KB |
| `GLM-5.3-Flash` | 658 / 241 KB | 8.4 KB |

Fifty-question tuning set (`docs/memory-bench/tuning.json`, written by a
fresh sub-agent from 49 distinct episodes and 41 entities, 18 easy / 22
medium / 10 hard): 46 of 50 answering facts in the top 20 (bar: 45), mean
answer rank 3.8, mean payload 9.2 KB, max 10.3 KB, nothing over 24 KB.
The four misses are paraphrases with no lexical overlap ("model cost split
across pipeline stages", "watchdog outside the harness").

### Queue drain

The first laptop sweep queued 4,773 backlog episodes at once. GLM-5.3-Flash
cannot disable thinking and takes roughly three minutes on a 16 KB
transcript slice, so the backlog drains over the following day at twelve
workers. Manual remembers are dispatched ahead of it, and sources are
taken round-robin so the Kimi and OpenCode episodes do not wait behind
the Claude ones. `scry memory status` reports `queue_ready`,
`queue_backoff`, `queue_parked`, and the last successful extraction.
