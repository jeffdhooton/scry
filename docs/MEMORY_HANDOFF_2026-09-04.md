# Scry memory: handoff, 2026-09-04

State of the run that started from `~/dotfiles/ai/prompts/2026-09-02-scry-memory-solid.md`.
Thirty-nine commits, four of the six done-bar items passing.

Read this with [`MEMORY_AUDIT_2026-09-02.md`](MEMORY_AUDIT_2026-09-02.md) (every round's
measurements, appended never overwritten) and [`DECISIONS.md`](DECISIONS.md) (every
architectural call with its reasoning and what would change our minds).

```
HEAD          63b1399, 39 commits on main, tree clean, go test ./... green
store         22,189 entities · 55,257 facts · 6,824 episodes
queue         2,037 held · 0 parked
deployed      laptop and mini running byte-identical binaries
```

---

## 1. The bar

Six clauses, all six required at once, measured against the live shared store on the
mini and the real logs. Every verdict came from a fresh grader instructed to disprove
it; a failed disproof is the only thing that counts as a pass.

| # | Item | Verdict | Evidence |
|---|---|---|---|
| 1 | Ingestion is alive and loud | **PASS** | laptop 35 ingesting sweeps, mini 19 of 83; 0 provider 402s; 0 socket timeouts in the last 20 sweeps on both; doctor failed loudly for the whole 22-hour outage |
| 2 | Remember is durable and fast | **PASS** | 20/20 accepted, p95 149 ms, 20/20 resolved into facts inside ten minutes of the provider returning, 0 duplicate episodes, 0 dead letters — against a genuine 22-hour outage |
| 3 | Recall answers the question | **PASS** | 54/57 (94.7%) on a grader's own held-out questions, max payload 15,095 B, probes 7/7 at rank 1 |
| 4 | The vocabulary is closed | **FAILS** | 39 relations exactly matching `resolve.Canonical`, all enumerated; bare numbers 0; branches closed; measurements closed — **status values ~42, open** |
| 5 | Identities are distinct | **FAILS** | three entities separate; 34 facts reattached; 16 leaking aliases dropped — **~40 facts still misfiled, 325 collisions** |
| 6 | Every agent is covered | **PASS** | 112 kimi + 44 opencode episodes, 2,102 facts trace to them, orient surfaces them in 6 of 9 repos, provenance verified file-by-file |

---

## 2. What shipped

Thirty-nine commits, each green on `go test ./...`, each deployed to both machines with
the previous binary kept beside the new one.

### Durability and observability

| Commit | What | Why |
|---|---|---|
| `d57d51e` | Retry a memory daemon call three times while the daemon restarts | Deploying restarts the daemon; a sweep in flight logged one error per transcript. 54 errors became 0. |
| `d57d51e` | Break the sweep's totals down by source | A line reading twelve episodes looks healthy while one agent has quietly stopped arriving. |
| `7fbec23` | Surface that breakdown in `memory.status` and `doctor` | It had been written to a key nothing ever read. A number nobody can see answers nothing. |
| `f417b00` | Log what the extractor types as a value | The only way to measure the share of the judgement the model carries. |
| `39d16ff` | Fail a backup that captures nothing | `~/.scry/backups` held 44-byte files reporting success. The rollback guarantee was false for those runs. |
| `e5cb0f2` | Stop one session filling both orient slots | Both facts shown for an entity came from the same episode 54% of the time. |

### Resolver rules

| Commit | Rule | Effect |
|---|---|---|
| `4efa9ca` | `KEY=value` settings, ISO run stamps, hyphenated participles, more shell verbs | Five families the previous loosening had given away |
| `68be4c0` | Enum members by shape; colon settings needing a literal value | Catches `QUALITY_OK`, leaves `SCRY_MEMORY_SOCKET` and `db:seed` alone |
| `40f7f6b` | Tallies, progress ratios, four-digit run stamps | 21 entities retired; `7 Wonders` and `24 Hour Fitness` kept by the title-case guard |
| `26842de` | A number bound to a unit anywhere in the name | 18 measurements retired; `36px card layout` stays a layout |
| `30a3343` | Refuse a path the model mistyped | 7 mangled paths on the store's largest fusion |
| `db718ab` | Guard a concept by what it stands to lose | Closed the concept side of the merge gate, in admission *and* in mention resolution |
| `245face` | Undo four regressions a grader proved | IPv4 read as a version, namespaced flags read as settings, named constants read as enums, capitalised titles read as verdicts |

### New operations

Three commands exist that did not before. All take a reviewed list rather than inferring
anything, all default to a dry run, all take a Badger backup before the first write.

| Command | Takes | Refuses |
|---|---|---|
| `scry memory reattach` | Facts named by src/relation/dst/valid-from **and text**, plus the destination entity and which side to move | a fact whose text has changed; an invalidated fact; a missing destination; a self-loop; an exact key collision on the destination |
| `scry memory unalias` | Spellings to take off named entities | a spelling the entity does not list; the entity's own name; and it never strips an index entry another entity owns |
| `memory.status` → `doctor` | The last sweep's per-source episode counts | — (warns when a source's files were read and produced nothing) |

### Data repairs applied to the live store

| Repair | Size | Reviewed by |
|---|---|---|
| Facts moved off `hermes-ops`, three batches | 34 | a fresh reviewer vetoed batches 1 and 3 before they landed |
| Leaking aliases dropped from `hermes-ops` | 16 | the reviewer's own list of leaks |
| Value entities retired across four migrations | 53 | replica dry run, hand-judged each time |
| The split Qwen model consolidated onto one entity | 8 | a grader had proved the split caused a wrong recall |

---

## 3. What we learned

The findings that outlive the commits. Most were arrived at by building something,
measuring it, and throwing it away.

### A rule that reads a name cannot decide what the name is

Thirteen rounds of lexical rules traded false positives against false negatives without
converging, because the same string is a value in one episode and an identity in
another:

```
main                    the branch   vs  a service called main
hermes-ops              the host     vs  hermes-ops the repo
CHANGES_REQUIRED        a verdict    vs  CMAKE_MINIMUM_REQUIRED a constant
validation_failed       a status     vs  user_login_failed an event
```

False rejection on 4,722 names drawn from **outside** the store is now **0.7%**, down
from 4.0%. Every further catch has been buying itself with a real name. Three of item
4's four named categories are closed; the fourth is where this approach ends.

### `concept` is where the extraction model puts everything it is unsure of

Of 704 entities created in one afternoon under the current rules, **418 came back typed
`concept`** — 59%. Across the whole store it is 51%.

Two resolver rules treated concept as a harmless fallback: `TypesCompatible` granted it
a wildcard, and the merge gates leaned on that. Both were built on a bucket the
extractor fills with statuses and measurements.

> A concept from this extractor carries no information at all, and rules should stop
> reading it as if it did.

### Moving the judgement to the extraction model does not work with this model

A `value` entity type was added on the reasoning that the model reads the episode while
a rule only sees a name. It produces **zero verdicts**. Two live probes against
`glm-5.3-flash`, on an episode written to be full of values, returned 13 and 14 entities
and used the type not once:

```
46 GiB                       → concept
1.4s                         → concept
RELAY_BATCH_SIZE=512         → concept
turn_detection: null         → concept
QUALITY_OK                   → concept
in progress                  → concept
CHANGES_REQUIRED             → concept
relay/batch.ts:88            → concept
feature/telemetry-batching   → concept
```

The obvious theory — that a safety fix reversing "when in doubt choose value" to "do
NOT choose value" had suppressed it — was tested by replacing that line with active
encouragement. No difference. The wording is not the problem.

`deepseek-v4-flash`, second in the chain, has never been probed. The counter is
deployed, so if any model ever uses the type it appears in the daemon log without
anyone running an experiment.

### Store-scale rules that infer an owner are right on their example and wrong in bulk

Four built, measured against a replica, discarded:

| Attempt | Proposed | How it failed |
|---|---|---|
| Refile a fact by what its sentence names | 9,329 moves | landed facts on entities called `allow`, `setup`, `delivery` |
| The same, hardware only | 54 moves | `sandbox` is typed `machine`, so permission facts followed it |
| Transfer aliases at store scale | 4,634 then 7,268 | aliases handed to entities whose facts never mention them |
| Run the write path's alias test over stored aliases | 6,027 splits, 674 moves | `gateway epic → epic`, `payment rows → payment` |

The last inverts for an interesting reason. `namedByKindWords` asks whether an alias is
another entity's name plus a kind word. Asked once about one new alias, the store is the
authority. Asked of 18,582 stored aliases, the store is also the *problem*: it holds
thousands of generic one-word entities, and against that population nearly every
compound alias reads as "that entity plus kind words". The test's precision depends on
the store being clean, which is what it was meant to establish.

What replaced them: judgement applied one fact at a time, written into a list a reviewer
can check line by line, and a tool whose only job is to execute that safely.

### Build the replica first

Every change to the store now gets restored into a scratch copy and dry-run there before
it goes near the live one. This caught a colon rule about to retire 82 entities of which
most were real — npm scripts (`db:seed`), skills
(`superpowers:test-driven-development`), model tags (`qwen3.5:9b`), meta properties
(`og:image`). The colon is how a namespace is spelled at least as often as a setting.
After tightening: 13 entities, every one a value.

### A claim in a comment costs nothing to write and nothing checks it

Three times, something written down as a guarantee was not in the code:

1. A decision entry said admission refuses three Hermes aliases. It **accepts** them —
   `AdmitAlias` returns `"already indexed to this entity"` before the check ever runs.
   The probe behind the claim had called the check directly, against a store where the
   alias was not yet indexed.
2. `reattach`'s comment said a fact whose text had changed would be refused. The move
   struct had no field for the text.
3. The same comment said an invalidated fact would be refused. The lookup passed
   `includeInvalid=true`.

All three were found by a reader, not by a test. The tests now exist.

### A grader's number is evidence, not a verdict

Graders found real, serious things — including a prompt change that would have deleted
2,038 path entities and 376 ticket entities once credit returned. They were also wrong
three times, and each time only checking settled it:

| Claim | What checking found |
|---|---|
| "139 of the 274 are self-loops" | 139 is the count with `hermes-ops` as *destination*. True self-loops on it: 0. Store-wide: 0. |
| "31 kimi logs stranded, 38 episodes lost" | Reset all 125 cursors to offset zero, re-read 83.2 MB: recovered **one** episode. The grader's own port reproduced only 55 of 71 files' refs exactly. |
| "`Done For Now` is a regression" | Admitted before and after — a genuine miss, not a regression. |

### Recall works when the asker names the thing

Not, as this run first concluded, when the words happen to overlap:

| Question set | Names the entity | Top-20 score |
|---|---|---|
| The grader's own 57 | 98% | 94.7% |
| `heldout-2026-09-03` | 88% | 85.5% |
| `heldout-b` | 59% | 51.5% |

`heldout-b` has **higher** question-to-answer word overlap than the set it scores 34
points below. Recall degrades when someone describes a thing whose name they have
forgotten. That is the honest scope statement, and a different claim from the one this
run originally made.

### Six retrieval ideas built and removed on measurement

Relevance feedback, entity-name expansion, graph traversal, coverage weighting (three
separate times), vector retrieval.

Also measured and rejected: retaining episode transcripts to feed the local vector
model. 124.6 MB of real transcript made retrieval **worse at every depth** (11 → 8
answers in the top 20; 35 → 33 in the top 2000). An earlier run of that experiment
showed 11 → 2 and was an artifact — one enormous document per file flattened the
inverse-document-frequency weights. Chunking transcripts to the size of a fact is the
fair comparison.

---

## 4. Corrections

Claims this run made and had to withdraw. The pattern matters more than any one of them:
the confident sentence and the measurement disagreed, every time, in the measurement's
favour.

**"Only the alias-index key matters; folding-level collisions are a metric artifact."**
`machine:qwen3-8-27b-uncensored-q5` and `tool:qwen38-27b-uncensored-q5` are the same
model. Asking the store for it by its exact name returned all eight facts from one twin
and none from the other. Withdrawn; the pair has since been consolidated.
*— round-13 identities grader*

**"27 of 31 recall misses are facts retrieval never returns at any depth."**
`capPayload` trims facts from the tail until the response fits 24 KB, so `--top 200`
never sees 200 facts — it tops out near 65. The plateau was the payload cap. 18 of those
27 return at rank ≤20 once the query names the entity; they were always indexed.
*— item 3 grader*

**"A grader found 429 aliases where a typed entity took a fact-bearing concept's alias."**
Zero of the 429 have that shape: 77 concept→concept, 115 tool→tool, 105 project→project,
80 service→service, no cross-type at all. The guard written for that shape fired on
nothing. *— round-13 identities grader*

**"Orient surfaces Kimi and OpenCode facts in 2 of 5 repos."**
Nine repos, not five — the store is shared with the mini and the laptop-side list was
incomplete. Six of the nine surface such facts, four of them 7–9 each. Undersold, not
oversold. *— item 6 grader*

**"`helm-spawn-agent` and `helm-status` are Hermes agent capabilities."**
Helm is a separate product — a native terminal workspace app. "Agent spawning" was read
as the Hermes agent. Both facts belong on `helm`, and that is where they went.
*— batch 3 pre-apply reviewer*

**"A `MisfiledFacts` counter reports how many facts sit on the wrong entity."**
It reported 5,879 where a hand count gave 25–31. The number did not mean what its name
said, so the counter was removed rather than published. *— caught in review*

---

## 5. Outstanding

### Item 4 — the status-value family

Roughly 42 entities, 26 carrying current facts: `dirty_working_tree`,
`READY-AFTER-FIXES`, `DID_NOT_START`, `validation_failed`, `pause-resume-completed`,
`north-star-wave-lands-clean`.

`participleStates` already holds the right words, and applying it to the end of a
multi-part name would catch most of them. It would also catch `user_login_failed`, which
a grader defended as a real identifier and which the guard test pins as a name. The two
are the same shape.

The shouted half is the same story: `DONE_WITH_CONCERNS` and `PYTHON_ARGCOMPLETE_OK` are
both all-caps identifiers containing a status word. Separating them needs a list of
library prefixes — `CMAKE_`, `CURLOPT_`, `E_`, `WP_` — which is the word-list treadmill
round nine ran and the round-13 grader explicitly warned against running again.

### Item 5 — the rest of the identity work

- **~40 more facts** on `hermes-ops` needing the same judgement, one sentence at a time.
  The tool and the review process are in place; this is unglamorous reading.
- **325 cross-type collisions**, of which **40 have facts on every side** and are the
  ones that can split a recall. Each is one thing spelled twice — `flash-next`/
  `flashnext`, `glm-53-flash`/`glm-5-3-flash`, `faq-vue`/`faqvue`,
  `here-travel-matrix-provider`/`heretravelmatrixprovider`. A **103-move consolidation is
  built and dry-run clean but unreviewed** (`scratchpad/proposal6.json` pattern: keep the
  side with more facts, move everything else). It is larger than anything applied so far
  and must be reviewed before it lands.
- **The alias leak is only half closed.** Sixteen spellings came off `hermes-ops`; other
  entities have the same problem. `childscribe-laravel` holds 130 aliases and 2,206 facts
  and owns the alias index for `docket`, `childscribe-mobile`, `haulyard` and `loom` —
  the largest fusion in the store, invisible to every metric reported here because its
  paths do not fold to any of those names.
- **Two admission holes stay open on purpose.** An entity's own name never passes through
  `AdmitAlias` — `store.PutEntity` writes the index entry unconditionally
  (`store.go:325-329`). And the `"already indexed to this entity"` shortcut
  (`aliases.go:215`) returns before every other check, which is why 0 of 18,581 stored
  aliases are refused by anything. Closing either changes how every entity is created;
  both need their own replica measurement first.

### Decisions parked with Jeff

- **Transcript retention** — answered "not without more thought", and the experiment
  since run supports it: retaining transcripts made retrieval worse, not better.
- **A real local embedding model** is permitted by the house rules (local embeddings
  allowed, hosted ones forbidden) but collides with "no CGO, single static binary".
  Untouched.
- **Rollback binaries** have accumulated to roughly 2 GB on the laptop as
  `~/go/bin/scry.pre-<sha>`. Pruning all but the last few is a one-line cleanup nobody
  has authorised.

---

## 6. Operating notes

### Where things live

| | |
|---|---|
| Store | On the mini. Reached from the laptop through `~/.scry/shared-memory.sock`, an SSH forward to `jclaw@mini:~/.scry/scryd.sock`. |
| Extraction chain | Configured in **one** place: the mini's `~/.scry/config.yaml`. The laptop's config states in a comment that it keeps none on purpose, and the mini's launchd job sets no `SCRY_MEMORY_MODEL` override. |
| Deploy | `go build -o ~/go/bin/scry ./cmd/scry` on the laptop; `GOOS=darwin GOARCH=arm64` build scp'd to the mini, then `launchctl kickstart -k gui/$(id -u)/ai.jermes.scryd`. Previous binary kept as `scry.pre-<sha>` on both. |
| Backups | `~/.scry/backups` on the mini. Every migration, reattach and unalias takes one first, and a backup that captures nothing is now an error. |
| Progress log | scry room `221c0d69ed04` — every grader verdict and every deploy, 34 posts. |

### Two things worth knowing before you run probes

**Measuring this system pollutes it.** Twenty probe remembers created entities out of
their own phrasing — `p95-149ms`, `scry-recall-tuning-strict-score-44-of-50`,
`strict-44-of-50` — and the rules written from that measurement later retired them.
Expect the same and check afterwards.

**None of the five files under `docs/memory-bench/` is genuinely held out.** `tuning` and
`tuning-strict` are the tuning set by name, and git shows both "heldout" sets fed back
into ranking (`66ecd00` fitted the synonym table on `heldout-2026-09-03`; `4648dba`
removed entries using another grader's set). Item 3's verdict rests on 57 questions a
grader wrote after the last ranking change. Any future recall claim needs a fresh set
written by someone who has not seen the ranking code.

### The grading discipline that worked

1. Build the thing.
2. Restore a replica from a live backup; dry-run there; hand-judge a sample.
3. Hand a fresh sub-agent one job: prove this does not meet the bar.
4. Apply only what survives; append the numbers to the audit; record the call in
   `DECISIONS.md` with what would change our minds.

Three bad passes and two false claims were caught at step 2 or 3. Nothing that reached
the live store had to be rolled back.
