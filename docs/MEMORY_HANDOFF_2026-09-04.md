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
