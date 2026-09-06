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
| Leaking aliases dropped from `childscribe-laravel` | 33 | applied 2026-09-04 17:00; benchmarks verified unchanged afterwards |
| The split Qwen model's facts moved onto one entity | 8 | a grader had proved the split caused a wrong recall |

> **The Qwen repair is half a repair.** `reattach` moves fact edges and nothing
> else — not the loser's name, slug, aliases, description or type. So
> `qwen3-8-27b-uncensored-q5` still exists, still answers to `Qwen3.8`, and now
> returns `[]` for both. The split was not healed; it was inverted into a
> one-sided blackout. `scry memory recall` is fuzzy enough to still rank the
> surviving entity first, so the damage is confined to exact `memory facts`
> lookups plus a zero-fact distractor carrying a stale description.

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
- **Item 5's collision clause is not satisfiable with the tools that exist.** This is the
  most important thing in this section and it was established by the last reviewer of the
  run. A 115-move consolidation of the 41 cross-type collisions that have facts on every
  side was built, dry-run, and reviewed line by line. Its verdict was APPLY-A-SUBSET, and
  the numbers underneath it are the point:

  - it takes cross-type collisions from **322 to 281**, not to zero;
  - **7 of the 41 groups do not move the number at all**, because `auditNames` skips
    only entities nothing references, `referencedSlugs` is built from `AllFacts()` which
    includes invalidated facts, and a loser holding one invalidated fact stays counted;
  - it would create **36 hollow entities holding 89 spellings** that return facts today
    and would return `[]`.

  The clause says *no two entities of different types share an alias anywhere in the
  store*. `reattach` cannot satisfy it, because it never touches a name, an alias or a
  type; `unalias` only removes a spelling, which makes lookups worse rather than better.
  **Closing this needs a capability the repo does not have: an entity merge that
  transfers name, aliases and type to the survivor and removes the husk.** Build that
  before attempting the consolidation again.

  The reviewer also caught two groups the mechanical rule got wrong — `homepageredesign`
  fuses ChildScribe's marketing home page with DBA Filing Guide's state-first homepage
  (two different products that fold to one key), and `pr80`'s single moved fact is about
  PR #75. Its error rate on the rule was 2 groups in 41. Its filtered subset is at
  `scratchpad/rev_subset.json` if this is picked up. It also flagged four groups where
  "keep the side with more facts" entrenches a wrong type: `glm53flash` (an LLM survives
  as `project`), `healmemorybookorders` (a console command as `decision`),
  `invoicerepository` (a TS module as `machine`), `termsvue` (a Vue page as `service`
  while the identical `faqvue` and `privacyvue` groups survive as `tool` — the same rule
  giving opposite answers).
- **The alias leak is partly closed.** Sixteen spellings came off `hermes-ops` and 33 off
  `childscribe-laravel`, taking the largest fusion in the store from 122 aliases to 89 —
  the dropped ones named other projects (`docket workspace`, `Mock Docket`,
  `~/workspace/childscribe-mobile`, `legacy loom`, `setpoint orchestrator`) or were
  generic enough to collect anything (`CS`, `RN`, `frontend`, `scratchpad`,
  `product name`). That entity still holds 89 aliases and 2,206 facts, and other entities
  have the same problem. The benchmark run afterwards came back unchanged —
  tuning 47/50, strict 44/50, probes 7/7, max payload 11,480 B — so the prune cost
  nothing measurable. Backup: `memory-20260904T170026Z.badger` on the mini.
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

## Continuation checkpoint — 2026-09-05

The later contract is preserved in `MEMORY_IMPLEMENTATION_GOAL_2026-09-04.md`.
It supersedes the old 41-group proposal, which must not be applied. The
workflow assessment is a follow-on roadmap, not permission to change hooks,
install embeddings, or weaken the ten-clause memory-quality goal.

Prevention is now deployed at `af77a6a` on both machines (18:11 UTC), with
byte-identical no-CGO binaries, SHA-256
`32fdcbad15dd0bb2f87a9987e07ecb887c1fefa7be2a2dbe4a299bfe5f2084e9`.
Previous binaries are retained beside each installed binary as
`scry.pre-af77a6a-20260905T1810Z`. Both actual stores were backed up immediately
before installation as `memory-20260905T181031Z.badger`; the audit records
absolute paths, sizes, hashes, process checks and reviewer evidence.

The last deployment review first caught an unbounded per-episode alias-name
cache leak, then passed the corrected build independently. Existing atomic
Apply, retirement/rehome protection, deterministic queue parking, relative-cwd
orientation and recall metrics are now installed. No live entity repair has
yet been applied in this continuation.

Post-deploy benchmarks match the immediate old-binary measurements: 51/62,
30/66, 7/7, 44/50, 46/50. Three original floors remain missed; do not lower
them or call these fresh held-outs. First observe the remaining queue work,
then take a stable fresh backup and regenerate the Qwen repair. Its Q5 exact
name is currently stolen by the distinct Q8 entity: a reviewed alias rehome
must precede the Q5 merge. Current Qwen preview preserves 19 touching facts,
including one invalidated fact, and drops ambiguous Q5/Qwen3/Qwen3.8 aliases.
The preview is not authorization or an apply-ready live manifest.

### Later checkpoint: Qwen repair applied, 18:28 UTC

The preceding no-live-repair paragraph is superseded by this receipt. Fresh
independent replica review passed the exact September 5 manifests, committed
at `1af0d3b`. After queue stability and immediate fingerprint/alias-owner
checks, the lead rehomed Q5's exact name from Q8, then merged the two Q5
identities. Q8 remains distinct. All four approved exact spellings return 19
facts including one invalidated fact; the former Q5 entity is absent; generic
Q5/Qwen3/Qwen3.8 spellings no longer resolve. Collisions fell 492 to 491 to 489.

The two automatic pre-apply backups are Mini
`memory-20260905T182725Z.badger` and `memory-20260905T182753Z.badger`.
Post-state backup is `memory-20260905T182812Z.badger`; local copies are in
`/tmp/scry-qwen-live-sep05.Koj9v3/`. The audit records absolute paths, sizes,
hashes and full receipts. An independent comparison of these actual live
backups is in progress. The five fixed suites remain 51/62, 30/66, 7/7,
44/50, 46/50: three required floors are still missed.

Next: review the complete remaining collision inventory and every ChildScribe
alias, then Hermes/Mini fact ownership and status retirement. A fresh reviewer
found 92 current ChildScribe aliases, three more than the verified 89-alias
prune checkpoint. That read-only audit is not permission to apply drops.
The deterministic conflict episode remains parked and preserved for explicit
repair/retry. Do not count queue parking, scoped Qwen success, or unchanged
benchmark scores as completion of the ten-clause goal.

### Later checkpoint: guarded alias batches, 18:55 UTC

Qwen's actual-live-backup independent postcheck passed; its complete report is
`memory-repairs/qwen-live-review-2026-09-05.md`. The full remaining graph audit
and every ChildScribe alias disposition are now committed at `2651048`.
All 2,441 missing endpoint occurrences are historical; current missing
endpoints are zero. Preserve those historical facts while repairing their
994 absent endpoint identities. There remain 489 cross-type collision pairs
across 427 folded groups, 2,854 all-history hollows and 89 current self-loops.

Local alias-batch code `1b913a1` fixed unsafe raw RPC default and partial-row
commit behavior. A fresh reviewer then disproved its old-daemon handshake:
a downgrade between connections still wrote through the old handler. The
corrected exact commit `53fafa9` uses `memory.unalias.apply-reviewed.v1` and
passed independent fault/race tests. It requires `{drops,expected}` manifests;
older arrays remain preview-only. Full tests/vet and focused races passed.

The first ChildScribe replica-only batch is 47 reviewed drops plus the two
Forge rehomes. It preserves every fact and foreign API index key, reduces
aliases 92 to 43 and collisions 489 to 486. Replica fixed-suite hits changed
51/30/7/44/46 to 51/30/7/45/47, max payload 13,380 bytes. Twenty-four ambiguous
aliases and five target-metadata rehomes remain excluded. The replica manifest
at `/tmp/scry-unalias-replica-sep05.rp5vdY/measurement/` is not live-approved.

Deployment candidate `/tmp/scry-unalias-deploy-sep05.Od8oCm/scry` has SHA-256
`31f185d70e1439a315a8ea12eaadf4f12d75e77d558edfa0852c890531705aff`.
Actual stores were backed up at 18:53:17. Both installed af77a6a binaries are
retained beside their paths as `scry.pre-53fafa9-20260905T1856Z`. A separate
fresh deployment-discipline gate is running. Neither installation has yet
changed to 53fafa9; no ChildScribe live repair has been applied. Deployment
must precede another fresh snapshot, manifest review and any live batch.

Queue now has two preserved deterministic conflicts, not one: the old
`guard-barrel-and-deep-import-rules` episode and new manual episode
`bec7e834a4912aad973fcf71f1c770d00fc39f9ab781a45e5ef3d134a1c340cb`
(`cadformats-workbench-20260904` alias owned by `workbench`). Both need explicit
identity review and exact retry; do not drop them or replay everything.

### Later checkpoint: guarded deployment complete, 19:15 UTC

Both machines now run the independently reviewed `53fafa9` artifact with
SHA-256 `31f185d70e1439a315a8ea12eaadf4f12d75e77d558edfa0852c890531705aff`.
Installation completed at 19:01 UTC; the previous binaries and both actual
store backups above are retained. The five fixed suites remain
51/62, 30/66, 7/7, 44/50, 46/50. Deployment evidence and reviewer report are
in the audit and `memory-repairs/atomic-alias-deployment-review-2026-09-05.md`.

Fresh stable postdeployment source backup is Mini
`/Users/jclaw/.scry/backups/memory-20260905T190441Z.badger`, 77,105,082 bytes,
SHA-256 `b8fda9a1c446d9f03dd8bc3116d1e49fd6020f7cd0e7d728553087b6ae445197`.
It contains 30,231 entities / 79,692 facts / 9,292 episodes. The exact next
49-row ChildScribe manifest is `memory-repairs/childscribe-alias-batch-2026-09-05.json`,
SHA-256 `24a765acc82e122c270e812e99993dbe3eff5d8261f88f05a0e065f16eb61ba6`.
Separate independent postdeployment replica review passed: full facts unchanged,
92 to 43 aliases, collisions 489 to 486, and precisely 43 raw keys changed.
The automatic backup restored to the entire original raw database. Report:
`memory-repairs/childscribe-alias-batch-review-2026-09-05.md` (room 69).

This is still NOT a live-apply receipt. A subsequent routine sweep queued
new episodes; immediate queue stability and exact expected fingerprints must
pass before apply. A third conflict is preserved/parked: episode
`65030b2a89dfc0f937f1cadb2ae1385629c78e6facd0e431a82aa91f9b5569d4`,
`scry-store` already owned by `scry`. Do not drop or broadly retry conflicts.

Next prevention defect reproduced locally: the relation mapper inferred
`same_as` from `aliases_index_to` and `rehomes_aliases_to` in ordinary
extraction of audit discussion. The resulting ChildScribe-to-API/Forge facts
are preserved by this alias repair, not semantic ownership evidence. A local
whole-relation identity guard and regression tests are in progress; it is
not deployed and no historical relation rewrite is approved.

### Later checkpoint: ChildScribe first batch applied, 19:35 UTC

The earlier pending-apply statements are superseded. Exact 49-row manifest
SHA `da2fd1a2397d37dbffcd0074e4caee96e64f5a8255b0213aeb91abf80eab5fc0`
passed fresh post-sweep source review and guarded live apply on 53fafa9.
Independent actual automatic-pre/post backup comparison passed: all 79,926
facts preserved, 30,345 entities unchanged except aliases 92→43, collisions
489→486, exactly 43 raw keys changed. Mini backups are
`memory-20260905T193522Z.badger` and `memory-20260905T193530Z.badger`;
complete hashes/measurements/report/CLI receipt are appended to the audit.
Live fixed suites are 52/62, 30/66, 7/7, 45/50, 47/50, no payload over cap.
The first two required floors remain missed. All 24 ambiguous aliases and
five unexecuted metadata rehomes remain untouched, as do contaminated facts.

Mapper-only commit `8c2a05d` failed review before deployment: fallback triple
coalescence discarded new routing sentences. Corrected `393eeec` preserves
distinct fallback evidence, refuses occupied exact keys atomically, parks
the intact conflicting episode, and disambiguates raw supersession. Fresh
independent code regrade PASS, with full tests/vet/races. Current binaries
remain 53fafa9; no historical relation rewrite is approved.

Prepared deployment artifact `/tmp/scry-fallback-deploy-sep05.5OX7ef/scry`,
SHA `acfb78186402aec9eef87e71e0b81f6edeac1b9ea46641efd417aaf4368f0bd0`,
is archive-built from 393eeec, no CGO, signature verified. Both installed
53fafa9 binaries are retained beside their paths as
`scry.pre-393eeec-20260905T1939Z`. Fresh actual-store backups are Mini
`memory-20260905T193835Z.badger` and laptop `memory-20260905T193836Z.badger`.
Separate deployment-discipline review is running; do not install before it
passes. No rollback/configuration/retention/provider changes are allowed.

Next collision candidate is FIVE migration-file records, not three. The
first three-record candidate failed semantic closure: it left a hollow
qualified-path machine and two facts on another tool. Expanded replica at
`/tmp/scry-migration0160-sep05.5STYbx/five-member/` passed independent review:
six facts, seven normalized keys, one fewer hollow, 489→487 collisions on
the older 19:04 snapshot. Manifest SHA
`7947eb322e33582fce214183a5a84e5c5639fb869442695d999ad31afd3e265d` is
replica-only, not live-approved. After prevention deploy, regenerate on a
fresh stable post-ChildScribe source and regrade before any live merge.
The full ten-clause goal, remaining collisions/statuses/Hermes ownership,
fresh recall holdout and two consecutive complete grading rounds remain.

### Later checkpoint: prevention deployed, recall check open, 19:47 UTC

393eeec is now installed on both hosts, with the matching hash, rollback
binaries and independently restored actual-store backups documented above.
Independent deployment gate passed. Only the two existing daemon labels
were restarted; laptop PID40300 and Mini PID99227 run the intended paths.
Mini rebuilt its index and is processing real episodes. The actual laptop
store remains dormant as before; no provider/configuration changes occurred.

Postdeploy fixed suites are 52/62, **29/66**, 7/7, 45/50, 47/50. The one-hit
heldout-b decline is under independent attribution review using both old
and new binaries on identical restored snapshots. Live ingestion changed
the graph between measurements, and there are no production read-path
changes; neither observation alone closes the regression. Do not lower
floors, declare recall fixed, or apply the next migration manifest yet.

The new normal write path caught a real same-time fallback assertion
collision and preserved episode
`287c409ed5605855f43f693db9983a45ba74820269b23f17f9296e35901041e8`
for explicit review (`cockpit-attention` to `cockpit-signals`, September 5
midnight). Four parked items now exist, with more normal work processing.
Parking prevents overwrite but is not a completed ingestion result. The
full goal remains active and unfinished.

### Later checkpoint: direct recall attribution complete, 19:57 UTC

The independent same-snapshot old/new comparison is complete (room 77).
Both binaries score 30, 30, then 29/66 on the three immutable sources. The
current SSR answer is intact but ranks 21 after new staging evidence and
vector refresh; a read-only simulation reproduces the displacement. Full
report: `memory-repairs/recall-attribution-2026-09-05.md`. Keep 393eeec;
the original recall floors remain failed, with no benchmark weakening.

The earlier temporary hold pending attribution is now lifted only for
preparing/reviewing the next bounded repair. Refresh the five-member
migration-0160 manifest from a stable postdeployment/post-ChildScribe source
and independently regrade complete inputs before live apply. The older
five-member replica manifest remains stale and must not be applied.

### Later checkpoint: six-record migration repair live, 20:12 UTC

The fresh five-record gate failed before live use: review-session ingestion
had created a sixth explicit SQL-file identity (`docket-migration-0160`)
with three facts. Old fingerprints were unchanged but semantic closure was
not. Expanded SIX-record manifest passed fresh review and was applied at
20:07:09 on 393eeec. Full audit/receipt/reports are under
`memory-repairs/migration0160-*2026-09-05.*` and appended above in the audit.
Do not apply either older three- or five-record manifest.

Independent actual automatic-pre/post backup comparison PASS: all 80,203
facts (7,792 historical) and 9,321 episodes retained, entities 30,441→30,436,
collisions 484→482, one hollow removed, precisely 35 expected raw keys
changed. Eight normalized old spellings now return nine facts. Backups:
Mini `memory-20260905T200709Z.badger` and `memory-20260905T200715Z.badger`.
Fixed suites unchanged: 52/62, 29/66, 7/7, 45/50, 47/50, maximum 13,370
bytes. First two original floors remain failed. Room milestones 78–81.

Nine explicit incoming-only status/measurement candidates passed an
independent replica gate; post-migration source regrading is running.
Candidate lives at `/tmp/scry-migration0160-fresh-sep05.teYtyC/status-nine-postmigration/`.
No status live apply yet. Preserve contradictory counts verbatim and record
separate factual-review gaps; do not conflate structural cleanup with truth
verification. The four parked episodes still require exact conflict review.

Fresh inventory includes 1,011 unique facts on the old 502 status candidates
(156 outgoing) and 856 unique facts touching the Hermes trio. Written rubric
is `memory-repairs/hermes-ownership-rubric-2026-09-05.md`; every fact still
needs its own evidence-backed disposition. Main runtime code remains
393eeec on both machines, all earlier rollback binaries/backups retained.
The full ten-clause goal and both consecutive complete grading rounds remain
unfinished. Continue; this is not a terminal handoff.

### Later checkpoint: nine status values verified, canonical guard regrade — 20:41 UTC

Nine explicit retirements are now live on unchanged 393eeec, with independent
actual-pre/post PASS. Manifest, full receipt and review are committed under
`memory-repairs/status-nine-*2026-09-05.*`. All 80,242 facts including 7,793
historical remain; entities 30,457→30,448, precise expected raw delta only.
Actual backups: Mini `memory-20260905T202710Z.badger` / `202741Z.badger`.
All five fixed suites unchanged at 52/62, 29/66, 7/7, 45/50, 47/50;
maximum 13,356 bytes. First two required floors still fail.

Existing DBA source routing is split: natural `dba-filing-guide` resolves to
`dbafilingguide`, while `DBA filing guide project` retrieves the actual
`dba-filing-guide` source and its intact converted value. This preexisting
defect is separately documented, not silently repaired or called a new loss.

New canonical-name admission code remains undeployed. Independent reviews
rejected a965177 (retained homonyms/generic references) and 1dac187
(determiner separator-normalization bypass). Corrected 62cf6e0 has full Go
suite/vet/resolve+queue race PASS and is undergoing independent regrade.
Never deploy the failed a965177 artifact. Build only the exact newly reviewed
commit, then obtain fresh backup/rollback/deployment gates before installation.
Five parked episodes remain intact, including the migration operation note;
no retry has been authorized by a mere builder test.

Latest immutable post-state inventory is
`/tmp/scry-migration0160-fresh-sep05.teYtyC/status-postlive-inventory/`:
1,002 old-candidate touching facts, 156 outgoing, 857 Hermes-trio facts.
Further status/alias/collision/Hermes review and the full two-round final
grade remain unfinished. The active goal continues.

### Later checkpoint: canonical-name fix deployed, 20:50 UTC

Both machines now run reviewed 62cf6e0, SHA
`821358499706bd9388b63a4368bb5320fc1bac61f12c93df48a158667b19bc14`.
Independent code/replica and separate deployment gates passed; rejected
a965177/1dac187 were never deployed. Previous393eeec executables retained as
`scry.pre-62cf6e0-20260905T2043Z` on both machines. Only the two Scry launchd
services restarted, laptop PID77189 / Mini9886. No configuration changes.
Both real predeployment backups were independently restored; exact paths,
hashes, processes and five-suite JSON are in
`memory-repairs/canonical-name-live-receipt-2026-09-05.json`.

Postdeploy suites unchanged: 52/62,29/66,7/7,45/50,47/50. First two floors
remain failed. Mini is processing normal work; six parked episodes were not
retried. A separate exact retry review of migration note ed50810b is pending.
Immediate post source is Mini `memory-20260905T205021Z.badger`, local
`/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/mini-after-deploy.badger`,
80,268 facts. The real laptop local store remains dormant, unchanged.

Six further explicit status nodes/seven facts passed only replica review
on the older 20:27 source. Fresh live-state review is mandatory before apply.
First-ten Hermes ownership dispositions are in separate proposed review,
not a complete857-fact audit or live repair. Continue the full active goal.

### Later checkpoint: six status values verified, 21:05 UTC

The fresh six-status gate and actual-live grade both passed on 62cf6e0.
Manifest 19ab9ec, SHA
`6b625c190057026d4824e2d89a9146c75697e469e0bb2673d2ffd5e34be7573a`.
Six nodes removed; all seven assertions remain as exact source values, with
all 80,299 facts including 7,800 history preserved. Entities 30,473→30,467;
all complete defect lists unchanged: 484 collisions, 2,855 hollows, 1,095
loops, 1,995 dangling-endpoint facts; 39 current relations. Second exact
preview is a verified no-write. Five suites still 52/62,29/66,7/7,45/50,47/50.

Actual pre/post Mini backups: `memory-20260905T210530Z.badger` and
`memory-20260905T210535Z.badger`; complete receipts/hashes and independent
verdicts are committed under `memory-repairs/status-six-*2026-09-05.*`.
Newest immutable inventory is
`/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/status-six-postlive-inventory/`.
It contains 859 unique Hermes-trio facts. First-ten review committed;
second-ten proposed evidence review is awaiting an independent verdict.

Do NOT blindly retry the parked migration note ed50810b. Its separately
reviewed once-only retry remains held because normal ingestion at 20:51:36
created `migration0160`, a project whose distinct alias key hygiene-folds
with the repaired SQL-file alias. New context is being independently
audited, not silently merged. Other parked items remain untouched. Full
cleanup, recall floors, remaining ownership reviews and both final grading
rounds are still unfinished; keep the active goal running.

### Later checkpoint: durable alias rejection candidate, 21:44 UTC

Prevention review now proves removed aliases can be recreated on deployed 62cf6e0.
This is a reproduced write-path gap, not observed live regrowth. Further alias
cleanup is held. Root candidate adds explicit owner-specific rejection records;
no live markers, new alias drop, pending retry or deployment has occurred.
Read `memory-repairs/alias-rejection-implementation-2026-09-05.md` and the archived
independent alias-reintroduction gap report before continuing. Independent code
and complete-replica reviews now PASS the supported APIs, with dormant legacy
merge inheritance explicitly unapproved. Full tests, vet and race checks pass.
Fresh backup restoration/deployment gate is still pending. Old binaries cannot safely write to a
future marker-bearing store; preserve pre-marker backups and intervening facts.

The four-hollow candidate has a fresh independent replica PASS; the migration-note
retry has an expanded-closure conditional gate. Neither is applied/executed and
neither bypasses fresh closure/code checks. Ordinary ingestion has moved live
state to 80,338 facts / 30,480 entities / 9,332 episodes, queue 0/0/6. Second-ten Hermes
review is committed 1cb5e2f (20/859 total reviewed); collision drift review 8d4c0c8
keeps mixed migration0160 ownership unresolved. All broad goal requirements stay
active; do not stop at this checkpoint or treat a candidate test pass as completion.

### Later checkpoint: alias rejection deployed, 22:11 UTC

Both machines now run d1f0a958, SHA
`290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`,
restarted at 21:58:20 UTC (laptop PID 40814, Mini PID 15220). Prior 62cf6e0
binaries are retained as `scry.pre-d1f0a95-20260905T2156Z`. No hooks/configuration
changed. Independent actual deployment review proves all 242,104 Mini keys and
83,378 laptop keys identical across immediate pre/post backups. Schema remains 1;
there were zero rejection records. See `memory-repairs/alias-rejection-deploy-receipt-2026-09-05.json`
and the actual deployment and recall-attribution reports beside it.

Post suites: 51/62, 29/66, 7/7, 45/50, 47/50, maximum 13,381 bytes, no over-cap
responses. Independent exact old/new binaries both score 52/62 on the earlier
21:44 source and 51/62 on actual pre/post. The Cell Saviors address evidence is
unchanged; fresh-index answer rank moves 18 to 23 with earlier corpus additions.
This disproves deployment causation, not the recall-floor failure. Historical
daemon incremental-index behavior is not established. Both original floors stay OPEN.

Three proposed ChildScribe alias drops remain UNAPPLIED. The original semantic
review blocked inaccurate rejection-reason wording; corrected wording passed on
the 21:44 source. A fresh complete semantic/replica gate is now underway against
22:10:12 source SHA `4a33e4d8b7e2786d3c9a936cf8bf71f7ca61603ecb8f3454b6f26a8dcb946382`
(80,473 facts / 30,551 entities / 9,342 episodes, queue 0/0/8). No prior-alias
rejection backfill exists. After any first live rejection, never let old
marker-unaware binaries write to the marker-bearing store; retain/reconcile later facts.

The four-hollow candidate needs regenerated rejection-aware expectations and a
fresh gate. The migration-note retry remains held: normal old-binary ingestion
added another alias to mixed migration0160, invalidating its earlier closure.
Eight parked payloads remain untouched. Durable deployment note queued once as
`64e172818729c23519f61824b5c2cd068efe77fe876e73af970a8d96738d6136`; never retry
that successful remember. The full goal, remaining ownership audit and both final
grading rounds remain active and unfinished.

### 2026-09-05 22:33 UTC — next exact alias gate

The exact three-alias manifest is now checked in beside the full bounded fresh
review, `memory-repairs/child-three-fresh-gate-2026-09-05.md`. Its independent
PASS extends through 22:21:40, not the subsequent sweep. No live apply yet.
The 22:31:22 backup SHA b669593b978041a646b8c6f3f3dc2cee5eb8cbad0e83d48063915a7a7aaeaea0
contains 80,586 facts / 30,604 entities / 9,353 episodes and queue 0/0/10.
An independent fresh-closure extension is running against that complete source.
Two new parked Cockpit conflicts are untouched. Four empty status records have
a regenerated d1 rejection-aware replica under separate independent review;
if the Child batch goes first, refresh that gate to preserve the three new
rejection records. Original 53/62 and 34/66 floors, remaining aliases and full
Hermes/Mini review, global defects, real sweeps and final grading remain open.

### 2026-09-05 22:38 UTC — three aliases now repaired

Child's envoyer, office dashboard and driver-core worktree aliases were
actually removed at 22:38:02 after fresh semantic, replica and raw-state gates.
The exact committed manifest is `memory-repairs/child-three-alias-batch-2026-09-05.json`;
full live receipt is `memory-repairs/child-three-actual-receipt-2026-09-05.json`.
Canonical 2,106 facts are byte-identical; all 80,586 store facts survive.
The five scores remain 51/29/7/45/47, max 13,361 bytes. Independent actual raw
gate passes exact seven-key change; full actual review report is pending.
Second dry run refuses all three, and broad hygiene remains 484 collisions.

There are NOW three live `ar:` records. Do not run old marker-unaware writer
binaries against this store. No earlier-drop backfill, positive rehome, other
alias disposition or four-record retirement was applied with this operation.
Fresh four-record review source is the later 22:42:38 backup (80,602 facts,
one independently traced external CADFormats episode); preserve the new
rejections. Its manifest remains `memory-repairs/hollow-four-batch-2026-09-05.json`.
The next stops-table semantic gate passed separately, but its exact technical
manifest and fresh full replica remain unreviewed. Third ten Hermes records
have explicit unresolved proposals under independent review; no fact moves.

### 2026-09-05 22:53 UTC — four records now retired

Actual three-alias independent review is fully PASS (report SHA cb8495e1,
archived `memory-repairs/child-three-actual-independent-review-2026-09-05.md`).
After independent four-record freshness checks and another root full raw
comparison, the exact four-record manifest was applied at 22:53:02. Actual
backup `memory-20260905T225302Z.badger` SHA cf7843f7; immediate post
`memory-20260905T225312Z.badger` SHA efa7466e. Full paths/hashes/tool outputs
are in `memory-repairs/hollow-four-actual-receipt-2026-09-05.json`.
80,602 facts / 9,354 episodes unchanged, entities 30,611 to 30,607, queue 0/0/10.
All five scores remain 51/29/7/45/47; second dry run refuses all four.
Independent actual four-record grading is still running, not yet claimed PASS.

Third ten Hermes records now have independent PASS_UNRESOLVED verdicts after
raw-key and wording corrections; no fact moved. The stops-table single-alias
candidate is under fresh technical/semantic review on the post-four backup:
`/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/stops-table-post-four-225312/manifest-replica-only.json`,
SHA 94484a6d7929e98ac42b06aa49b9276b0fc812976bc4cf275613abcf3ffd5c87.
No stops-table live apply, previous 49/33 rejection backfill, queue retry or
broader cleanup occurred. Global 484 collisions and original recall floors
remain open; two final grading rounds have not begun.

### 2026-09-05 23:06 UTC — actual four-record verdict complete

The independent actual four-record review is now PASS and fully archived as
`memory-repairs/hollow-four-actual-independent-review-2026-09-05.md` (SHA
56eaa1a627bd7c6cdd1f1824ac9d2d397012ba0a97a60d4fa97198e5616c7401).
Exact 16-key prediction, full raw preservation, restored backups, all retained
assertions, complete inventories and no-write refusals pass. Independent five
scores remain 51/29/7/45/47; room 107 records the bounded verdict.

The subsequent five-item sweep drained; a combined durable repair note was
queued ONCE as 5c5ce0b27cfaad9b697c9c73aaa6a3cea1ed8ef3df542273839820c22002141b.
Do not retry it. Stops-table technical/semantic review passes only the fixed
22:53:12 source; refresh the complete closure after new ingestion before any
apply. Expo/dev-client review is discovery only, with two missing original
Claude transcripts still unlocated. No new broad cleanup or goal completion.

### 2026-09-05 23:36 UTC — refused stale alias gate; review progress

Stops-table is STILL UNAPPLIED. Independent 23:14:10 source remained fully
equal at 23:23:03; real 23:27 ingestion then changed Child's fact fingerprint.
Immediate 23:33:34 preview refused before any apply command. New full
23:33:53 backup SHA 907b9ae1e3210ac3840271e66ab95e34c9a24838863031812937c161bd076703
has 80,844 facts / 30,725 entities / 9,371 episodes. Private refreshed manifest
9f081e8f6180717ebfe6020adcbcd20bf190a40aa2754ec64e8584fee609c964 is under complete
independent source/replica extension; committed 9b34 manifest is stale.

Fourth-ten Hermes dispositions now independently pass: 2 KEEP, 8 UNRESOLVED,
zero moves. Full report/verdicts and integrated proposal are under
memory-repairs/hermes-fourth-ten-*. Forty records reviewed is not the full
873-record snapshot, and later ingestion still needs review.

Current recall disproof and rejected original-query lexical experiment are
archived. Fixed-source baseline 51/29/7/45/47 regressed to 40/31/7/41/44.
Do not deploy or copy that private candidate into source. Live binary remains
d1f0a95 with three rejection records and 19 retirements. No new remember was
sent after combined note 5c5ce0b2; never retry that successful note.
Both original recall floors, remaining aliases/global defects and final two
fresh grading rounds remain open. Continue the active goal.

### 2026-09-05 23:59 UTC — stops-table applied; finish actual proof next

The 9f081e8f refreshed manifest passed full source/replica and 23:48 freshness
review, then applied at 23:51:33. Child is now 39 aliases, with FOUR rejection
keys; all 19 retirements remain. Complete automatic PRE 235133 SHA32de971a
and immediate POST 235134 SHA7680cdcf are copied locally, hash-verified and
root-restored. Child canonical 2,112 facts are identical; stops table no
longer resolves; second dry run proposes no mutation. All five live suites,
miss sets and mean ranks remain 51/29/7/45/47. Full receipt and reports are
in memory-repairs/child-stops-table-*. Independent ACTUAL review is running;
read its full report and reconcile any disproof before claiming actual PASS.
No repair remember note yet; do not retry prior successful 5c5ce0b2 note.

Relative reason normalization is independently REJECTED: one irrelevant
lexically strong fact can suppress the correct explanation's bonus. Full
report is archived; the private one-hit improvement is not being promoted.
Production ranking and both d1f0a95 binaries remain unchanged.

Expo dev client / dev-client / dev client now have a bounded independent
semantic negative PASS for Child only, no rehome or fact correction. Twelve
available originals establish backend/client separation; five missing
originals remain unverified. Fresh technical manifest and replica review
must account for the new fourth rejection and any ingestion drift before
live authorization. Global cleanliness, all-alias/Hermes reviews, original
recall floors and two final fresh grading rounds remain open. Keep going.

### 2026-09-06 00:07 UTC — actual stops-table gate closed

Independent ACTUAL stops-table PASS is fully archived as
child-stops-table-actual-independent-review-2026-09-05.md, SHA1d25de8e.
Complete actual raw-map prediction, all facts/episodes/twelve queued inputs,
existing marker decisions, real ingestion rejection, reopened second apply
no-write and all ten benchmark controls pass. Room 114 records the verdict.
One successful durable note queued as
3be6deac923caf55d196a90eff0d29bbc4e8c5051c20bfc7177293872bcfce19;
DO NOT retry. Prior 5c5ce0b2 also remains successful and must not be retried.

Dev-client fixed-source technical grading is underway on private manifest
0f1c0555 from post-stops 235134. This cannot approve the later live store,
which has already ingested additional data (hygiene 30,759 entities / 486
collisions). Refresh complete source closure, markers, and all collateral
before any next live mutation. Three literals, two normalized keys, no rehome.

A source-only exact tie-order fix is under test/review: lexical candidate
ties end with Doc.Key and recall ties with existing hitKey. No weight,
synonym, candidate limit or expected-answer change. Production binaries
remain d1f0a95. Its independent grader must test true named-injection and
nil-index paths as well as full restored benchmarks before promotion.

### 2026-09-06 00:26 UTC — fact preservation takes priority

Next dev-client apply is HELD. Its fixed235134 technical PASS is archived,
but fresh001120 backup SHA5b8e9789 has a proven historical assertion overwrite
at a normalized canonical-status key. All old fact text is absent from fresh
facts; the source episode survives. Do not apply private0f1c0555 or02319c37
manifests by merely updating hashes. Read the full source BLOCK report.

Private /tmp/scry-fact-collision-fix-sep06.0MxCkU occupied-key guard now has
independent PASS SHA00c58521 from actual old-store replay, no-CGO/race tests,
full raw/backup/queue/benchmark controls. Integrate exact reviewed files,
verify combined artifact and fresh restored backups before deployment.
This does NOT recover the lost fact or fix current-triple coalescing.

New source contains a plaintext preview credential from password-change
prose; Redact fails to remove it. Never repeat its value/derived spelling or
full sensitive projection. Hash-only evidence is sufficient. No credential
use/rotation/removal occurred; stored-history cleanup requires permission.

Tie-order source change independently passes its bounded scope; full report
recall-exact-tie-independent-review-2026-09-06.md SHA849f12ab. It does not
fix clipped-value identity collisions or unequal-endpoint scoring. Both
production binaries remain d1f0a95. No new remember since successful3be6deac;
do not retry it or5c5ce0b2. No global PASS or final grading. Continue.

### 2026-09-06 00:57 UTC — combined guard deployed, actual grading active

The exact reviewed occupied-key guard and bounded tie-order fix are committed
as24eafab and deployed on BOTH machines at00:49:06UTC. Installed artifact
SHA4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27,
Go1.26.2darwin/arm64/noCGO/trimpath. Laptop PID93994, Mini PID98265.
Both old d1f binaries retain SHA290a14 at their installed path plus
.pre-24eafab-20260906T0048Z. The suffix is a label, not exact retention time.
Never use the older marker-unaware62cf binary as a standalone rollback.

Read complete integrated predeployment and immediate004209 freshness reports
under memory-repairs/fact-guard-*. Both independently pass their bounded
prospective scope. Actual pre004209/post004916 backups are fully copied,
hashed and restored. Root's actual whole-map delta is ONE added pending row
on Mini, with every244498preexisting record unchanged; laptop83378records
remain exactly equal. Immediate live five-suite counts/miss sets match
51/29/7/45/47, cap0max13373. Mean answer ranks improve slightly in the two
heldout suites; independent old-CLI frozen-post measurements reproduce those
same means, so do not attribute the differences to new scoring weights.
Full root receipt is fact-guard-actual-deploy-root-receipt-2026-09-06.json.
Fresh-context actual grader is running; do not count the root receipt as PASS.

Later005430 backupSHA4aa1e86c349dd9f2d972a1b9567de7614bcc5736e3b3cae1e9a1f7cb8153fb9e
contains only one changed pending payload relative to immediate post. That
new manual input was enqueued00:47:13 before deployment and has now parked
after ONE ErrAliasClaimed attempt. It is NOT an ErrFactConflict capture and
NOT successful ingestion. All facts, episodes, old parked payloads and
markers remain byte-identical in root's comparison. Independent later-delta
verification is requested. Do not retry it. Two actual subsequent sweeps,
normal successful extraction and the complete goal bars remain open.

The first private credential regex prototype is independently REJECTED;
full report credential-redaction-independent-rejection-2026-09-06.md.
No redaction changes are in the deployed artifact. A second private parser
experiment at /tmp/scry-credential-parser-sep06.glUWUF has original bounded
tests passing, but the unchanged prior35-case adversarial matrix still has
6 failures (29pass), plus320idempotence checks pass. Read REDACTION_REVISION.md
for exact scope/limits. It is neither integrated nor independently approved;
do not edit expectations to claim a clean suite. Historic credential cleanup
still requires permission, and the old lost assertion is not recovered.

Dev-client three-alias apply remains HELD; no further graph repair follows
from the guard deployment. Child39aliases, ar4 and nineteen retirements are
preserved. Global collisions/hollows/dangling endpoints, Hermes review,
recall floors and final two fresh grading rounds remain open. No new durable
deploy note yet; never retry3be6deac or5c5ce0b2. Continue the active goal.

### 2026-09-06 02:10 UTC — startup safeguard deployed, goal active

This supersedes the preceding deployment-state paragraph, not the original
goal. Both machines now run exact a078240 / binary SHA
7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7,
installed/restarted02:03:32UTC. Laptop PID46487, Mini PID24189. Retained prior
24eafab binaries at installed path plus .pre-a078240-20260906T0146Z each have
SHA4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27.
All older rollback files remain. This change refuses incompatible/missing
schema markers without wiping populated stores; schema stays1. Populated
Restore and old numeric-schema rollback are still unsafe. No format change,
historical recovery, private resolver experiment or live repair was deployed.

Read the complete schema-refusal-artifact/fresh/immediate-predeploy reviews
under memory-repairs. They independently verify exact build, full suite,
all-record restores, old numeric999 wipe/new refusal, original five-suite
parity and immediate015908 preservation. Actual020346 backups are complete,
hashed and restored; root sees all81304facts/7855history and old graph records
unchanged. Only scanner/cursor/retry metadata and new pending inputs differ.
Live scores/misses/mean ranks remain51/29/7/45/47, cap0max13366. An independent
fresh-context ACTUAL grader is running; root receipts are not its final PASS.
Normal successful extraction continues. New020913/020914 backups capture
later activity for subsequent sweep review. Do not mistake parked-input
preservation for successful ingestion, or these scores for original floors.

The previous24eafab actual deployment and two real subsequent scans now have
independent bounded PASS reports archived; all final graph-quality clauses
remain open. See the audit's correction separating slug-only missing-claim
counts from real listed alias defects. Existing canonical backdating and
six-reference truncation were observed again and are not newly approved.

The broad restatement candidate is independently REJECTED (queue recovery
and order-dependent admission failures). A narrower private exact historical
address branch at /tmp/scry-historical-address-sep06.Mbi8Rn has full unchanged
tests and independent source/rollback/supersession PASS; real-replica grading
is now running. Reports and test-only fallback-coverage correction are archived
under historical-address-*. It remains UNINTEGRATED and UNDEPLOYED. Existing
current-triple sentence loss, general temporal identity, new fact addresses
and recovery remain open. Credential parser prototypes are still unapproved;
no historical credential cleanup is authorized.

Source decision note065d17ba was submitted once and is proven ingested;
previous actual fact-guard deployment notef1cb541f is independently ingested.
No new a078240 actual-deploy note at this entry. Never retry those notes or
the earlier IDs. Child dev-client alias apply stays held. Continue the active
goal; no final completion or full two-round grading claim is made.

### 2026-09-06 02:49 UTC — historical source gate; admission remains open

This supersedes the preceding private/integration and reviewer-pending status.
Schema a078240 actual deployment independently PASSES its bounded preservation
gate; full report SHA c3fde6b6071652ddeaf33e76794db8ab21527bf24feffe5f25873cb3c0afa4ba
is archived under memory-repairs. Its later two explicit sweeps FAIL final
graph quality: archived report SHA
3754421fe93521da8f4ce9beaea6fff1aa2ecb44d0ffbbaeac9d91f82665ae74.
All 81,304 original assertions survive those intervals, but one new factless
runbook persists, and held-out floors still fail. Subsequent fresh 02:36:47
inventory finds three new factless runbooks in total, not just the first one.
Other structural defects and existing capped-ref loss/backdating remain open.

Exact historical-address preservation is now SOURCE-INTEGRATED in a06cd7b
after independent source, post-supersession, fallback-coverage and real-replica
reviews; root full no-CGO tests pass. It is NOT DEPLOYED at this entry. Both
installed binaries remain a078240, with the previous rollback files intact.
Candidate /tmp/scry-historical-deploy-sep06.INlBVW/scry is built from an exact
a06cd7b export, Go1.26.2 CGO0 darwin/arm64 trimpath/version ldflags, SHA
7938d05258bb3b08461441b7448a8da4a95374664f42a9b240f22c3327de55ef.
Independent exact-artifact grader /root/historical_artifact_predeploy has
reproduced the hash, full suite, both direct restores, real historical replay
controls and all 235 rank/full-payload comparisons on BOTH fresh replicas.
Final report and 02:47:04 freshness extension are pending; read them before
deploying. Root has independently restored both 02:36 and 02:47 complete
backups with all raw bytes preserved through Open/index/read. No alias apply.

Actual schema deployment note 41fc642ba3b1f0726f842a3a30a2bd82e7e7e4cca5f69ddd8b95afefe09da7fb
is now proven ingested in 02:36:47, absent pending, raw episode SHA
1a62dce9051c451894a3211bb0eef4efe1dbb6833ac9ee19d29ce88687cf3732;
the reviewer independently confirms 17 citing facts with closed provenance.
Source-only historical note be14c4835407470abf98fcf385b85ec93377056a772657b3728657ec135e6f37
was submitted once at approximately02:36:27 and is pending in that snapshot.
NEVER retry either note. Room130 records source integration and later quality
failure. Current repository 7c32a75 adds the full later-sweep report/audit.

Unattached-metadata prevention is a PRIVATE design, not source or store policy:
/tmp/scry-unattached-evidence-sep06.db52ow/PROPOSAL.md,
SHA e8022c81ecb06e193a5925ddab0d9893cf1bb672f7dae1fd49f2a2f89cd0e728.
Root baseline fixtures preserve existing expectations and reproduce declared
metadata with zero facts plus DeleteEntity retaining alias attestations.
Independent /root/unattached_admission_design_disproof is grading architecture;
early fixtures prove old orphan attestations can route a real fact before any
final hollow filter, and deleting a new node can delete an old orphan claim.
Do not implement simple filtering, delete metadata, invent filler facts, or
count hidden evidence as a completed repair. Keep this separate from a06cd7b.
The user's untracked workflow assessment remains untouched. Its hook/install,
old41group and other suggestions do not override the stricter active goal.
