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

### 2026-09-06 03:07 UTC — actual historical rollout PASS; private journal fix

Supersedes the preceding NOT DEPLOYED status: both machines now run a06cd7b /
7938d05258bb3b08461441b7448a8da4a95374664f42a9b240f22c3327de55ef,
restarted02:54:47–48UTC. Previous a078240 is retained at installedpath plus
.pre-a06cd7b-20260906T0250Z. LaptopPID10158, MiniPID12146. Actual independent
reportSHA227f66914659b8007050c0cfa9656154b9b6f0d036a6638cd8f04edea708cdd6
proves full logical equality across both pre/post pairs, actual process/hash
verification, and old/current five-suite plus235full-response parity on each
post replica. Whole goal remains FAIL; see audit for counts and exact limits.
Room134 records this bounded verdict, not a whole-goal pass.

Actual-deploy note243363ca0a562f795248a6c9a9dd15f491bd74ef32138cff8a80b6d1274a0c41
was accepted ONCE03:06:20UTC; never retry. Source notebe14c483 is independently
ingested by024704. First explicit postsweep03:02:43→03:04:10 completed11episodes,
0errors; full030543pair captured. Second explicit sweep started03:06:20 and
is still running at this entry. Do not count these as clean final rounds.

Private admission design now requires durable generation-bound evidence;
transaction-local orphan isolation alone was independently disproved. Journal
only at /tmp/scry-unattached-evidence-sep06.db52ow/code is UNINTEGRATED. Its first
version was rejected for caller-buffer reuse redirecting undo into fact keys;
fixed source d961d53c2c5fcc69b110987788c01837e79534ac381548c655fe463e020e3a80
owns its full plan. Root targeted/full no-CGO tests pass; independent extension
is running at /tmp/scry-identity-journal-fixed-grade.xiYvzO. Do not integrate it
as admission prevention: generation ledger, structured evidence, complete caller
coverage, cost proof and explicit legacy contract reconciliation are unfinished.

### 2026-09-06 03:36 UTC — sweeps reviewed; generation primitive private

Both explicit posthistorical sweeps are finished and independently reviewed:
historical-address-two-sweeps-independent-review SHA
3fc7d4dbee7426291f6c68d4e7ad9ca0e060fb0f2b2c107227b4f2f2fc93fd2a.
Bounded old-record preservation PASS; whole goal FAIL. First completion03:04:10
is pinned root CLI evidence (background sweep replaced persisted report);
second03:08:23 has matching persisted completion. All six full restores pass.
New zero-fact runbook persists despite aggregate hollows returning2854; all five
scores remain51/29/7/45/47; dry hygiene still487 collision pairs and not no-op.
Actual note243363ca is ingested03:09:13.687952 with15 citing facts, raw episode
SHAa648c31778ef2ec1f314619c1ad0665f79dcefac0ca6117d8a7fbad8f7e21d0b.
Never retry it. Installed artifacts remaina06cd7b unchanged; no new deploy/apply.

Private corrected journal now independently bounded PASS, report81279f03;
still UNINTEGRATED. Private generation primitive initial source3dca5a18 fails
two independent UTF-8 serialization/reload tests; frozen reportc0e64ba1 is
archived. Private fix source3c196ab9d3067c50d26fb839bb65fd7820e0d363da6818ef2a659f93df7680dc
adds strict UTF-8 validation, unchanged reproducers pass; fresh corrected review
is running. All code under/tmp/scry-unattached-evidence-sep06.db52ow/code remains
uncalled, unintegrated, undeployed. Root full-support-scan baseline188–208ms and
~146MB allocated per operation is too costly to treat as accepted hot-path proof.
Continue controller/observation/support and explicit adoption/lifecycle design;
do not infer that a primitive PASS prevents hollows or makes old writers safe.

### 2026-09-06 03:48 UTC — source foundations retained, still no admission

Supersedes the prior wholly-private status:55063da source-integrates uncalled
journal d961d53c;7e33b74 source-integrates uncalled generation e07c6c50 and all
independent regressions. Both are exact reviewed sources, no production caller.
Full integrated no-CGO suite PASSES noncached, existing expectations unchanged.
No schema/adoption/default behavior/live write/deployment occurred; installed
artifacts remaina06cd7b on both machines, prior binaries retained.

Generation3c196ab9 was independently rejected for lossy second-offset timestamp
encoding after its UTF-8 fix. Root's UTC exact-instant correctione07c6c50 now
independently bounded PASS, final reportSHA
43a5eefeba6913350fff44e05ebcff890cf633bfb841108dce62e07527c1a428.
Both rejected versions/reports and all three failing reproducers are preserved.
The correction never changes entity/fact timestamps; malformed legacy state
refuses instead of being repaired. Room137 records this bounded result.

Fresh /root/admission_controller_v2_disproof is reviewing private
/tmp/scry-unattached-evidence-sep06.db52ow/CONTROLLER_V2.md,
b05d1e875bac6ad5c2ae59762eb7c7322a9bb7b530ae8fcdc3e0522b42f07014.
Early review tightens outermost-transaction finalization and malformed raw-fact
support handling; final report pending. Next private unit likely immutable
occurrence-observation storage, not production admission. Keep reviewing full
controller/adoption/lifecycle and utility; overall goal still FAIL, never mark
these foundation commits as prevention deployment or final grading rounds.

### 2026-09-06 04:08 UTC — controller reviewed; observation/owner fixes private

Controller V2 report67218c1c is archived: GO for separately private observation
storage and owner harness; NO-GO integration/adoption. Retain all new-birth
occurrences, supported or unsupported, with original facts before flips;
validate existing fact-only anchors. Finalization belongs to the outermost real
transaction. Duplicate raw fact fields/key-body conflicts must refuse support
cleanup, and dangling references cannot authorize an owner by slug. Full review
and audit record the selected next units and still-open B/adoption mechanics.

Observation/tmp/scry-observation-evidence-sep06.97oku1: initial56f7ce81 rejected
for duplicate raw episode id/time authority, report4ad70548. Corrected83a39c03
uses raw known-field provenance checks and preserves unrelated unknown bytes;
all unchanged reproducers/new controls/full no-CGO suite pass locally. Independent
corrected /root/observation_primitive_disproof running. This code is NOT integrated.

Owner/tmp/scry-admission-owner-sep06.t3WRXJ: initiala4d48fc6/store1fd30285 rejected
because public merge commits through a closed/finalizing facade outside outer
rollback, report4a2c5be3. Corrected owner14821246/store9491d689 plus private
merge/retire/unalias guards refuse public independent maintenance in every owned
phase. Root30 phase/entrypoint checks and full no-CGO suite pass; unchanged
required-safety regressions retained. Independent corrected
/root/admission_owner_harness_disproof running. No shared Store edits or deploy.

Shared87a6d1a still has only uncalled journal/generation foundations; daemons
remainactuala06cd7b, no new live apply. New source-only remember note
406c30f42de41e6b023473390a0e17b6f4eb87347cd91797c35fda46f0a6495c
was accepted ONCE~03:48UTC(depth28); ingestion not checked. NEVERretry. Room139
records private rejections/corrections. Continue the active full goal; no final
quality pass or deployment authority follows from either private-unit verdict.

### 2026-09-06 04:22 UTC — corrected foundations source-integrated

Observation83a39c03 was retained uncalled in cef41bd after corrected independent
PASS b20e0a01. Owner14821246/store9491d689 plus maintenance guards and all safety
regressions are now retained uncalled after corrected independent PASS fce8aec7.
Full integrated no-CGO noncached suite passes. The owner has no production caller;
normal root maintenance remains functional. Room141 and the audit contain the
exact report pins and measured scope. Both initial rejected reports remain.

Actual binaries remain a06cd7b; no live change. Continue with a private raw
fact-reference validator, provisional vote buffer, then separately reviewed
controller/adoption/lifecycle and observable dispositions. Reference existence
must not authorize assigning an old dangling fact to a newly matching slug.
The complete goal still fails; these are not deployment or final grading passes.

### 2026-09-06 04:42 UTC — raw-reference foundation; B remains private

Shared owner06bcaf0 and corrected raw checker c28aed0f are uncalled foundations.
Raw checker independent correction report46277181 passes unchanged failed
regressions plus exhaustive Unicode/duplicate controls; shared full no-CGO suite
passes. Restored031013 root scan81,634facts/793.575333ms/zero raw changes; no p95
or independent actual-store certification. Original rejectiona2db1f0a retained.

B private sourceea8b0967 in/tmp/scry-provisional-votes-sep06.3YJMlM has root
targeted/full PASS but NOT unconditional independent PASS: report21314f8b proves
a raw concurrent iga insertion can evade the snapshot prefix check. No normal
producer demonstrated; seven independent tests pass, one unchanged phantom test
fails. Keep private until complete cooperative selector/owner writer enforcement
is reviewed. Do not remove or invert that failure. Frozen independent export
/tmp/scry-provisional-independent.HfAbxl retains it and full failing output.

ControllerV3 private designc43e0634 in/tmp/scry-controller-v3-sep06.yKRgWZ is under
fresh design disproof, including partial/deferred assertion semantics and exact
adoption lifecycle. No deployment/adoption/live write; daemons stilla06cd7b.

### 2026-09-06 05:09 UTC — outcome reviewed; next exact legacy anchor codec

Corrected outcome7b40e318 is now copied uncalled into shared source with every
independent regression; combined full no-CGO noncached suite passes. Review2cb328a6
closes initialfc7f928b's real disk/memory storage-error input dump (initial8b75b94c).
No current-outcome projection, Apply/CLI integration, support/owner authority or
live effect. ControllerV3 review2521939a is archived: resolve primary/Supersedes
dependencies BEFORE both fact phases; include InvalidateFact/DeleteFact; deferred
assertions need no fabricated birth. Exact per-key journal attribution still open.

B remains private at/tmp/scry-provisional-votes-sep06.3YJMlM. Root-only storage-error
fixa493e99d passes new unchanged6fbec360 regression and earlier builder tests, NOT
independently graded. Initialea8b0967 raw-ledger phantom failure is not fixed; keep
the original frozen failing test/export. All-producer selector coordination remains.

Latest root backup closure/tmp/scry-foundation-closure-sep06.8IEPu5/shared-044441:
76,019,551-byte backupbef6d1b6 restored247,461rows exactly. Prior406c30f4 note now
ingested/absentpending, rawEP64ecb7b9. New5b7c8402c70b7cef62322279a7de6033352e188b8fa0821aafa71a7c1a99d51b
submitted ONCE04:44:41 queued32, still unverified. Never retry either accepted note.

Next private workspace/tmp/scry-legacy-inventory-sep06.sbEk1E contains read-only
sizing and forced-rollback capacity tests:31,227 named anchors fit one default
transaction today (9,171,887 key/valuebytes, maxbatch10,066,329), with zero committed
rows. ANCHOR_DESIGNd7e70022 and purecodec contract20591a73/source7a84b00d remain
unapproved; source tests/review next. No actual adoption/anchor writer exists.
Both daemons stilla06cd7b; no deployment/live repair or final grading pass.

### 2026-09-06 05:27 UTC — legacy codec integrated uncalled

Pure anchor codec7a84b00d plus supplied2558b0a4 and independent64fad216 tests
survive independent review6d2ed4d3 and combined full no-CGO noncached suite.
Report archived under docs/memory-repairs/legacy-anchor-codec-independent-review-2026-09-06.md.
Stable legacy name/slug/time and full original raw fingerprint are encoded;
proposed consumption records preserve the anchor. No adoption, owner authority,
persistent tombstones or consumed-reader enforcement yet. Private next workspace
/tmp/scry-legacy-adoption-sep06.cdtdmm is for exact adoption/reader implementation.

Backup051519 SHA193f19b3 (76,112,807bytes) restored at
/tmp/scry-foundation-closure-sep06.8IEPu5/shared-051519:247,638raw rows exact,
digest9b6626fc. Root now verifies5b7c8402 note ingested/absentpending, rawEPfd029819.
Do not retry it. Room146 records bounded review and root-only closure. Deployed
a06cd7b unchanged; B still private/ungraded correction/raw-phantom caveat unchanged.

### 2026-09-06 05:45 UTC — adoption unit reviewed, not enabled

Exact adopter/active-reader285ae96d now copied uncalled into shared. Independent
review373d92b8 and tests51d07183 prove complete preview/apply preservation on own
restored051519 backup:247,638 originals unchanged,31,247 anchors+one marker,
manifest5292699c,9,178,163 newKVbytes,zero events. Real late errors roll back.
Root combined full no-CGO suite PASS. Report/contract under docs/memory-repairs.
No production caller, public CLI, tombstone writer or ongoing admission policy.
Live adoption is explicitly prohibited until complete lifecycle/controller review.

Complete reference inventoryd80563bb separately reviewed next. Then actual fact
mutation attribution, per-key identity/alias writer history, dependency closure,
Force current projection and all-writer lifecycle gates remain. Include generic
metadata setters protecting reserved markers; no legacy timestamp-nudge helper.
Private next ledger workspace /tmp/scry-fact-ledger-sep06.aWw65K initially empty.
Source notec51e3b8a accepted ONCE queued30 around05:28UTC remains unverified;
never retry. Room147, deployeda06cd7b unchanged, full objective still open.

### 2026-09-06 05:48 UTC — complete reference inventory reviewed

Uncalled d80563bb full raw endpoint-count/digest scan retained with unchanged
suppliede356d5cc/independent54ef0948 tests. Review13e0b68b and final independent
plus combined shared full no-CGO suites PASS. Original strict checker unchanged.
Root actual051519 scan81,892facts/29,388endpointslugs/digeste1c6b989/825.392584ms,
all247,638originalrows unchanged. No independent actual/p95/support certificate.
Room148; report/contract under docs/memory-repairs/reference-inventory-*.

Continue private /tmp/scry-fact-ledger-sep06.aWw65K: FACT_LEDGER_CONTRACT.md now
written, source/tests next. No actual ledger, attribution policy or controller yet.
Prior prototype/review failures and remaining whole-goal gates still apply.

### 2026-09-06 06:05 UTC — fact ledger reviewed; alias history next

Uncalled fact ledger2040e19a/contractc8e6074c, suppliedd43613c5 plus unchanged
independent2786ed68 tests now shared. Review73c4319f/full independent and combined
no-CGO suites PASS. It accounts actual fa: mutations and reconstructs complete
baseline digest, but ordinals are descriptive: no parsed semantics/support or
all-writer integration yet. Root-only restored05151920-mutation no-op preserves
all247,638raw rows,1.8331635s; private measurement testda567608 not integrated.

Identity mutation designaede57c7 in/tmp/scry-identity-mutation-sep06.zzGGzz has
review94e6e51f archived. Next GO only private en:/al: actual-writer actor-history
unit. Full eventual undo/legacy-defect exception must include target identity and
selector/consumption state, ALL retained canonical/alias listings and unlisted
alias keys pointing at changed owners. An unchanged alias can become dangling.
Baseline characterizations897d7f20 stay private, not required bad semantics.

Current episode result design39b32ab7 in/tmp/scry-current-outcome-sep06.0Eu5lS is
only draft. Changed Force extraction may omit old births/ordinals: per-subject
heads alone leave stale current counts. Full structured input/revision must not
lose TypeFallback through extract.Result marshaling; no transcript retention.
Note b2a40dab accepted ONCE queued30 around05:49UTC and priorc51e3b8a remain
unverified; do not retry. Room149; installeda06cd7b unchanged, complete goal open.

### 2026-09-06 06:28 UTC — identity writer retained; input revision next

Uncalled identity writer25022d38, contract04599cdf, suppliedbc75a9fe and independent
284e934c retained after review5b8a969b and independent full no-CGO PASS. Mechanical
actor history only: no safe undo/relationship policy or normal callers. Reports
under docs/memory-repairs/identity-writer-*. Complete lifecycle/controller still open.

Current episode design39b32ab7 now has conditional reviewa9722fcf, archived under
memory-repairs/current-episode-design-independent-review-2026-09-06.md. Read its
complete six conditions before implementation: full observation/input revision
matching, semantic outcome reuse before predecessor links, stable birth inventory
on identical Force, total declaration/fact accounting, exact head CAS and pinned
inspection. Next private structured-input codec can proceed without claiming any
support/current result. Characterization tests0d6732cb stay private (bad baseline
behavior is not a required new regression expectation).

Root restored fresh061537 backupSHAfd22b0c3 with all247,951raw rows unchanged.
c51e3b8a/b2a40dab/c9c7295b now independently of synthetic grading root-verified
ingested and absent pending; exact hashes/counts in audit. Never retry. Room151,
deployed a06cd7b unchanged. All remaining whole-goal gates and two rounds required.

### 2026-09-06 06:47 UTC — parsed input revisions retained

Uncalled input sourceb69e6daf/contract71e1b290 retained after independent review
80eb05f0, unchanged supplied14ce28a2/f5ab5f5e and independent4c670e87 tests/full
no-CGO PASS. Reports under memory-repairs/input-revision-*. Full canonical parsed
revision, exact observation matching, immutable provenance-checked writer and
bounded key/chunk readers only. Adopter now refuses io-input: (SHA9ac1ea52).
Current result/head, total declaration/registration/fact accounting, semantic replay,
fixed support/ownership policy and all-writer lifecycle protection still required.

Private relationship inventoryeae817f6 at/tmp/scry-relationship-inventory-sep06.MukgqY
under independent review. Root corrected only nil-as-delete malformed-empty fixture;
source unchanged, corrected tests/full PASS. Restored061537 read-only scan84,248
selected rows/all247,951rawsame260.756583ms; exact details in audit, no ownership or
independent actual/p95 claim. Root-only replica test58451572 not for integration.
Next composition needs finalizer-owned complete baseline and actual writer history
verification before any post-undo relationship policy, never caller support flags.
Notee7be6597 accepted ONCE queued30, unverified; never retry. Room152, installed
a06cd7b unchanged, full objective and two all-clause grading rounds remain open.

### 2026-09-06 06:53 UTC — relationship inventory retained; attribution next

Uncalled inventoryeae817f6/contractb758a05b retained after independent review46fb43bf,
supplieddfc258a8 plus independent4c1b6a99 unchanged. Eleven targeted groups and
independent/combined full no-CGO suites PASS. Reports under memory-repairs/
relationship-inventory-*. Raw identity/control observation only; no alias or lifecycle
authority. Root-only061537 measurement/fixed fixture history remain in prior audit.

Next /tmp/scry-identity-ledger-sep06.7AWIWi captures its own complete baseline and
actual writer, checks ordered before/after against full final selected raw map.
Source/tests private and unreviewed, no normal callers. Post-verify materialization
and undo must receive separate fixed-policy accounting; full scans do not establish
serializable prefix locks or prevent uncoordinated producer phantoms. Room153,
installeda06cd7b unchanged, complete goal remains active.

### 2026-09-06 07:21 UTC — identity ledger retained; coordination next

Uncalled ledger e43d2b90 retained after independent review618182c8 with unchanged
supplied c3714fb6 and independent e2017062 tests. Full independent and combined
shared no-CGO suites PASS. Contract/report under memory-repairs/identity-ledger-*.
Exact complete-map accounting is not ownership/support/undo or production safety.
The review's public ClaimAlias phantom and post-verification-write counterexamples
remain explicit unclosed integration requirements, with tests preserved unchanged.

Root restored fresh070235 backup SHA dd348721 with all248,206raw rows unchanged;
e7be6597 now confirmed ingested/absent pending, no retry. Exact pins in audit.
Room154, installed a06cd7b unchanged; no live adoption, repair or deployment.
Next private design /tmp/scry-serial-admission-sep06.wDNcu0/SERIAL_ADMISSION_DESIGN.md
SHA dedcbb18 is under fresh-context design disproof. It proposes separate graph
coordination because exclusive maintenanceMu would delay durable remember queue
writes. Do not implement or claim production safety from an unreviewed design.
Fixed finalizer/lifecycle, full cleanup/recall and two all-clause rounds still open.

### 2026-09-06 07:47 UTC — private serialized boundary retained

Coordinator source store8093c39a/pendingf51c3dec/owner396389b0 retained after complete
design9ac136e5 and codecc5b9697 independent reviews. Reports/contracts archived in
memory-repairs/serial-admission-*. All original tests unchanged, supplied68686ff2/
4a176605 and independentbf566fa9 retained. Independent/combined full no-CGO PASS.
New private entry coordinates graph producers, not production policy. Ordinary
remember queue writes remain outside graph lock and synthetic real-handler tests
prove durable progress in both phases. Private bridge/daemon/replica tests NEVER
ship and remain absent from shared source. Raw writers/postverify/captured-root/
concurrent-Close exclusions and old phantom tests remain; lifecycle still required.

Fresh073526 backup SHA794c5638 restored all248,345raw unchanged. Latest note12e12912
now ingested/absentpending; no retry. Root-only complete identity+fact no-op scope
cost2.511723s with every raw row unchanged, not independent/live p95. Audit has pins.
Room156, installeda06cd7b unchanged. No live adoption, cleanup or deployment.

Next design19474eeb /tmp/scry-birth-registration-sep06.vZeC2U/BIRTH_REGISTRATION_DESIGN.md
is under fresh design disproof. No implementation yet. Resolve capture timing,
exact creation coverage and Force inventory requirements before implementing its
smallest coherent scope. Full fixed finalizer, B, dependencies before both fact
phases, lifecycle, cleanup/recall and two all-clause rounds still open.

### 2026-09-06 08:30 UTC — registration accounting retained

Private source509faba4 retained after independent code reviewbb8541d6 and prior
design reviewcd07b5b4. Complete contract/reports/root evidence archived under
memory-repairs/birth-registration-*. Suppliedffd97200/ef5339c7/6085a0ec and
independent61dab1aa tests exact; independent and combined full no-CGO suites PASS.
Initial root legacy-fixture setup failure retained, no source/original-test changes.

Fixed owned input/both-ledger wrapper enforces registration before actual actor
history, exact first observation retries, stable creation coverage, and normal
baseline-reference deferral. No support, alias/lifecycle, Force or EP certificate.
Private registration has no production caller. Unsupported registered entities
still can commit under this finite accounting boundary; do not activate it alone.

Fresh081846 backup SHA5111bb78 restores all248,468raw unchanged. Prior note3da1026d
confirmed ingested/absentpending; no retry. Root-only pure synthetic registration
on own replica2.606648s, no writes/events; not independent/live-p95. Private probe
f43933d2 MUST NEVER ship. Audit and root evidence retain exact pins. Room158,
installeda06cd7b unchanged, no live adoption/repair/deployment.

Next design /tmp/scry-episode-selection-contract-sep06.A6dX3r/EPISODE_SELECTION_CONTRACT_DESIGN.md
SHA3fd0d796 is under independent DESIGN review. Preliminary corrections require
canonical proposed outcome-link and declaration birth-reference order, and a
strictly local head-counter read claim rather than proof of full history length.
No selector implementation yet. Full admission/support/undo/B/lifecycle, cleanup,
recall and two complete grading rounds remain active work.

### 2026-09-06 09:13 UTC — structural episode selector retained

Private selector9566d0d2/db8a01a1/e312424a/bcc2bc0c retained after full independent
code review738af509 and design review3ee87a6b. Complete contract/design/reviews/root
evidence archived in memory-repairs/episode-selection-*. Four supplied test files
and independent28a60251 exact. Adopter67f1ddfd adds only two reserved-prefix refusals.
Root combined shared full no-CGO PASS61.079s store/15.417s resolve/28.382s daemon;
independent full PASS, tenth independent limitation test separately PASS. Initial
reviewer nil-value fixture failures are preserved. No production caller or policy.

Own complete original input/slots, exact expected head CAS, canonical proposed
links/references, immutable current-lineage successor selection and bounded internal
inspection now have executable contracts. STAGED is not committed, head validation
is local not full-chain, supported descriptions are not actual ownership, and a
fresh-only birth inventory can omit old births. These remain fixed-finalizer duties.

Fresh084717 backupSHA9d3e5c50 restored all248,586old raw rows unchanged; prior note
caaab1fa now ingested/absentpending, no retry. Root-only replica selector canary adds
9 synthetic auxiliary rows, all old bytes/exact retry/reopen preserved. No graph
mutation. Private probe063d68ec MUST NEVER ship; exact evidence in audit/root report.
Room160, installeda06cd7b unchanged; no live adoption/repair/deploy/sweep.

Next: read-only ordered identity-resolution overlay, conditional design review
/tmp/scry-delayed-birth-disproof.ul0Moz/DELAYED_BIRTH_DESIGN_REVIEW.md SHAcb15ac10,
fully read with all eight characterizations, room161. Freeze corrections before
implementation: keep every provisional discovery through complete resolution;
separate optional alias rejection from required canonical/endpoint deferral; own
deterministic complete indices rather than elapsed-time global cache; preserve full
alias/lifecycle relationship closure; virtual Phase A precedes Phase B dependencies;
never-staged candidates have Materialization=nil. No controller replacement decision
or production approval yet. Exact current-triple sentence-loss reproduction remains.
Full support/B/lifecycle/Force, assertion identity, cleanup/recall and two all-clause
rounds are still required. Do not stop the unbounded goal at this checkpoint.

### 2026-09-06 11:00 UTC — corrected ordered overlay remains private, under review

Design33b764c9 and complete contracte56d63aa are archived in memory-repairs/ordered-
overlay-*. First code manifest680539f6 received independent **NO-GO e42c9196**:
twenty cases prove missing alias-owner/natural lifecycle validation, unreviewed
retired-slug alias bypass, and stored EP evidence borrowing the current-input
exception. Full report archived; root read all378 new test lines and reproduced
failures. Original rejected export /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj
remains immutable. Room164. No private overlay/pure-policy source integrated yet.

Corrected export /tmp/scry-ordered-overlay-correction-sep06.soHIfk, manifest
9e44b47b16aaf6ac31c88af1347205789bbe873001e463ed76c5696b38331463:
600 unchanged e097fa6 baseline files,39 additions. Only controls40268dde and
lookup57c9b939 implementations differ from rejected source; all prior tests exact.
All40 combined planner groups PASS6.816s; uncached full no-CGO PASS66.861s store/
17.476s resolve/29.141s daemon, vet PASS. A fresh code grader is running; do not
interpret root tests or read-only replica conservation as approval. All first
failures, fixture-only corrections and stale-self-claim regression remain recorded.

Fresh105609 backupSHA9aa54e51 restored all249,361raw rows exactly. Root corrected
replica canary PASS16.133s before/after reopen and explicit PRIVATE-replica adoption,
zero planner writes/events, all old rows preserved. Exact evidence in
memory-repairs/ordered-overlay-corrected-replica-evidence-2026-09-06.md. Private
probeae03ddad MUST NEVER ship. Initial102852 probe14a20f82 likewise never ship.
Both shared-* replicas untouched; overlay-* replicas now hold private anchors and
must not be mistaken for unmodified restores. Live a06cd7b remains unchanged.

Prior cf8f2454 note verified ingested/absentpending in102852. New note
acc0f2532a529a1d115a892f0f41ec764cc36ec593b60ce72b706fe5abc2f8ff
accepted once/depth34, not yet verified; NEVER retry. Next: read the complete fresh
review/tests, fix every proved defect while preserving failed versions, then only
retain an exactly reviewed private unit. Production lexical deduplication, both FA
phases/support/materialization/B/ownership/lifecycle/Force, current assertion loss,
cleanup/recall and two all-clause rounds remain mandatory. Keep the goal active.

### 2026-09-06 11:21 UTC — second overlay NO-GO; third candidate under review

Full second report9076e7ee is archived in memory-repairs/ordered-overlay-second-
independent-disproof-2026-09-06.md; root read it and all new tests54dd58d5. Four
independent leaves fail on immutable second manifest9e44b47b: valid derived-slug
retirement aborts unrelated work (including actual synthetic retirement), and
type-driven alias removal omits its type-upgrade producer. Final independent
contained-temp suite fails only those cases; supplied tests pass. Room165.

Third private export /tmp/scry-ordered-overlay-third-sep06.AnfP5a, manifest
76808342886d78e364b6e3f38eb5e98e9d0a4f79bb099a64a42de5979f8cdea1,
keeps all600 baseline files exact and41 additions. Changed controls2fbc3d24,
aliases35d10852, declarations8f429a7f preflight paired derived rt controls and
record consumed/current metadata producers. Registry and all supplied tests exact.
Root unchanged independent failures reproduced, then extra root tests8e21ac9b
reproduced malformed-control masking, alternate status reinterpretation and missing
metadata/result dependencies. All failures retained; no assertion weakening.

Selected tests PASS7.492s store/2.318s resolve; uncached complete no-CGO suite
PASS66.858s store/16.931s resolve/30.059s daemon, log1ec245b7. Vet and exact157/248
policy tokens PASS. Fresh bounded grader ordered_overlay_third_disproof is running
against frozen76808342, review-input68bdaf9a. Do not integrate before its verdict.
New111946 backup77,021,042bytes SHA0f4e9dec copied and restoring separately for
root replica proof/note closure. Live a06cd7b unchanged; no deployment or repair.

111946 restore/probe completed: all249,362 initial raw rows exact, digest373eda04.
Root probe PASS16.623s before/after reopen and explicit PRIVATE-replica adoption,
zero planner writes/events; adoption adds31501 only, all original rows retained.
Exact evidence memory-repairs/ordered-overlay-third-replica-evidence-2026-09-06.md.
Private probea4f9b2ca MUST NEVER ship; overlay-111946 now privately adopted,
shared-111946 untouched. Noteacc0f253 is durable but PARKED after one fact conflict,
NOT ingested; no retry/unpark, pending raw93179b24. This is not ingestion success.

### 2026-09-06 12:13 UTC — fifth overlay under review; assertion bridge rejected

Third independent NO-GO09ee6c30 and fourth NO-GO7994e1d0 are now archived in
memory-repairs/ordered-overlay-{third,fourth}-independent-disproof-2026-09-06.md.
Third found 19 leaves: malformed canonical retirement hidden by normal lifecycle
deferral, incomplete relationship-owner validation and missing released-claim
producer. Fourth corrected those but still skipped affected aliases' own rt rows
in four consumers. Root reproduced all unchanged failures; rejected exports remain
immutable. Rooms167/169. Fourth full root suite passed; independent disproof wins.

Fifth private export /tmp/scry-ordered-overlay-fifth-sep06.WNftRz/code, manifest
43a61c978988de72b28a238eef0681f71f5c34cf88d43fe4e65b0d1821d48b22,
keeps 600 baseline e097fa6 files and 46 additions. Only lookupb965deec implementation
changes from fourth: scoped worklist covers exact affected rt/rs and actual actor,
claimant and listing ar pairs, including natural-listing counterpart claim addresses.
No owner/spelling cross product or canonicalization of defective raw claim keys.
Valid controls remain evidence, not automatic repair. New root testsbe3daeda retain
all earlier failures and add empty/invalid-UTF8/whitespace exact-key controls.
Selected tests PASS9.022s/2.380s; full uncached no-CGO PASS69.059s store/17.283s
resolve/29.715s daemon, logdb1a6329. Vet and exact157/248 policy tokens PASS.
Fresh bounded ordered_overlay_fifth_disproof is running; review input9e7fd0f0.
No private overlay or neutral-policy source integrated. Do not mistake this for
normal-write prevention, factual support or a full-goal grading round.

Fresh114330 backup77,072,490bytes SHA6cf1d850 restored twice with all249,499 rows
exact, digest036fac91. Root fifth replica probe9b22dee9 (NEVER SHIP), separate
/tmp/scry-overlay-fifth-replica-sep06.z7Eq8b, PASS16.520s before/after reopen and
explicit PRIVATE adoption. Zero planner writes/events; adoption adds31517 only,
all original rows exact. overlay-114330 now privately adopted; shared-114330 remains
untouched. Full replica evidence is archived separately. Live a06cd7b unchanged.

Separate comparator-only current-assertion experiment remains PRIVATE REJECTED:
experiment68987a69 and independent semantic NO-GO86773de0/testf265ef5b archived in
memory-repairs/assertion-bridge-*. It fixes narrow full-assertion collapse but
leaves contradictory exclusive targets, order/observation-time errors, arbitrary
or ineligible SupRef targeting, and unknown-field loss on invalidation; global
malformed-date refusal newly withholds unrelated utility. Original compatibility
failures preserved, no test expectation rewriting. Room168. Next FA work must
compose exact assertion identity, explicit temporal/ambiguity/date rules and
component-level deferral before either metadata or interval mutation.

Note bb92111e4194a567c5d7ac71a1d33fc3da784ef3842390723bd4a2f27152ae58 was accepted
once at11:42:58; verified114330 durable pending, attempts0, not parked/not ingested,
pending raw3cfac8fd. Verify later, never retry. acc0f253 remains closed as parked,
not ingestion success. All cleanup/recall/production integration and two complete
fresh-context rounds remain required; keep the unbounded goal active.

## Continuation — 2026-09-06, after 13:00 UTC

Root remains sole builder/live writer; complete unbounded objective remains active.
Main source is unchanged from e097fa6; subsequent commits contain documentation.
Both live binaries remain a06cd7b/SHA7938d052; no deployment/live graph mutation.
Do not treat the private implementation below as production prevention.

Fifth independent NO-GO a0cb2d9e, sixth NO-GO565ce2d5 and seventh NO-GO52e50249
are now archived in memory-repairs. They successively prove actual natural-owner
controls, fact-only consumer controls, and four missing explicit-claimant entity
absence witnesses. Prior immutable exports/tests and all failed outputs remain.
Rooms170/172/173 record verdicts. Seventh has no actual replica canary.

Eighth private correction: /tmp/scry-ordered-overlay-eighth-sep06.8lun4B/code,
freeze fab44ba27d8922d657d33da54840187651268a2523454dbf2231544792896de7
(600 baseline/51 additions). Only controls880e2629 changes: common missingControls
captures exact en:slug existence/absence before lifecycle checks, including claimant
and inferred-veto consumers. Unchanged seventh test reproduces four FAIL before
correction (.970s). Retained tests PASS10.481/2.370; full uncached no-CGO suite
PASS store70.371/resolve17.482/daemon28.047, log286716006c0f55c688b9fd95daedfc88911669d96f17c307865bfc1c186579c7.
Vet/token157 declarations/248 symbols PASS. Fresh whole-unit reviewer
ordered_overlay_eighth_disproof is active; no independent eighth verdict yet.

Complete assertion plan docs/MEMORY_ASSERTION_ADMISSION_PLAN_2026-09-06.md
SHA20d5ed6f includes corrected v2 cf278586 and full independent addendum88bf6aa6;
initial draft508a70c2/reviewa3c25081 and two characterization tests are archived.
Private ACTUAL composed-program work has begun in
/tmp/scry-complete-admission-sep06.6qYb7K/code, not main. Four new Go source/test
files identity_assertion_admission_plan{,_test}.go and
identity_assertion_admission_temporal{,_test}.go implement incremental original
ordinal/full-identity/address proposals, local deferrals, effective-start exclusive
timelines and frozen eligible-unique hint planning. No normal Apply adapter,
committed outcomes/terminal replay, final support/ownership closure or writer yet.
This export still includes rejected seventh overlay; port the accepted successor
only after exact checks. Do not ship or separately promote these partial stages.

Construction first failed two groups because union-find created empty entries in
the deferred map. Initial source7cf1f5dc and log preserved; source-only correction,
unchanged tests PASS. Temporal first corpus then PASS2.234s including existing
owner tests; log88b143dbdbb76b1eb26fabf05da86929fdd4cfaa1833a77ebb0b52d4ab0274f8.
Both input orders, older-arriving state, equal-start/closed overlap, 1971/2002 hint
eligibility at1980, ambiguity/raw extension/closed-target local deferral, repeated
hints and unrelated failed future hint are covered. Raw equality/events0 throughout.
These are builder planning-stage tests, not actual writes or independent acceptance.

Newest restored backup122103:77,150,279bytes SHAa49eefd4,249613 raw rows digestd037654b.
shared-122103 untouched; overlay-122103 privately adopted by sixth probe only.
Complete root evidence351b1556 is archived. It does not override sixth NO-GO.
bb92111e is now verified INGESTED/absent pending/not parked, EPraw74f6b72f, no retry.
New note059723bc63eaff3c9eab2772a5d4a4ce98419cfa1fb98c1bc2c02f13990182e6
accepted once at depth36, not yet backup-verified: verify later, never retry.
Prioracc0f253 remains parked. All original cleanup/recall/sweep/two-round bars remain.

## Continuation — 2026-09-06, after 13:20 UTC

Plan/design archive committed18a4f1d, shared Go source still unchanged. Eighth
independent NO-GO c79bdbca now archived: eight current/history reference-veto routes
omit exact consumed FA rows. Initial independent dangling-FA fixture correctly
failed ordinary PutFact; only new seed changed to retained raw FA, assertions
unchanged. Root reproduced all8 failures/.748s; room174 records verdict.

Ninth private /tmp/scry-ordered-overlay-ninth-sep06.nAculd/code, freeze
2e06afc117db9c961924dffc93202f816b2f9523e53a29997c364c6195247dbb
(600 baseline/54 additions). Adds exact matching FA witnesses plus scoped reference
queries bound to complete snapshot digest/count, including zero-result queries and
current-only alias veto. Only overlay/controls/aliases change plus new references
source and test. Root new tests pass actual zero query/current/historical alias
branches. Retained PASS10.526/2.334, full uncached PASS69.528/16.800/27.460,
logd8ad22a7bcaaea8fc5295494226cd11a792232173ded58bd25b86fdbc57a8457.
Vet/token PASS. Two compile mistakes preserved: missing slices import f4300e63 and
new test Inventory→actual Final field5786d642; source/import corrections only.
Fresh ordered_overlay_ninth_disproof is active and has reported two preliminary
generation-vote empty-prefix scope failures, not a final report yet. Do not promote.

Fresh132039 backup77,341,054bytes SHAcb1f9eef restores250029 rows/digest5916d0aa
twice. shared-132039 untouched; overlay-132039 now privately adopted by root probe.
Complete root conservation evidence558cfc5b is archived. Separate exact ninth
replica export /tmp/scry-overlay-ninth-replica-sep06.dUF8Nm, private probe f8bf99c0
NEVER SHIP, PASS20.282s before/after reopen/adoption; all original rows exact and
planner events/writers0. Private adoption adds31596 only, post281625/7c47d6d1.
New059723bc is now verified INGESTED/absent pending/not parked, EPrawd851156f.
No retry, live graph mutation, binary deployment or full-goal round.

Complete-admission workspace6qYb7K now has six Go source/test files, adding actual
monotonic FA effect projection to construction/temporal/hint stages. Deferred
primary effects disappear from baseline targets; exact restatement and hint end
combine once; full raw/endpoint/old EP validation covers changing routes; support
inputs derive only from retained incoming assertions. No identity closure, receipts,
terminal replay, writes or normal Apply adapter yet. Source/test hashes and worklist
in its IMPLEMENTATION.md. Selected builder tests PASS1.091s, log8b275817; full
uncached suite PASS70.674/17.537/29.570, logea19bc4d. Export still includes seventh
overlay and cannot ship. Continue the complete program and original goal bars.

## Continuation — 2026-09-06, after 14:00 UTC

Ninth NO-GOa5f5c8fa and tenth independent GOe965ccac are archived unchanged.
Tenth freeze912c4592 passes the complete finite read-only planner contract,
all retained/fresh/full uncached suites, vet and exact policy pins. Room176 records
the bounded verdict; it does not approve assertion admission, writers, deployment
or either full-goal round. Root read report, complete fresh testea16c92f and index.

Private complete-admission6qYb7K now uses all57 exact tenth additions in place of
the rejected seventh overlay;600 baseline pins remain exact. Eight admission WIP
files added ordered identity projection. A new root removal fixture incorrectly
expected frontend removal; preserved failure/source, then used retained Atlas box
case with an explicit removal-proposal precondition, no policy/source change.
Selected tests PASS2.372s. Full post-port uncached suite PASS71.967/17.492/28.928,
log3b73539e. Details in memory-repairs/admission-projection-checkpoint-2026-09-06.md.
Root is now implementing full identity relationship comparison and monotonic
proposal/support closure inside that same private program. No actual writer,
receipt/replay or normal Apply adapter yet. Live source/binaries/graph unchanged;
newest backup and durable-note closure remain132039/059723bc. Keep goal active.

## Continuation — 2026-09-06, after14:38 UTC

Private complete-admission6qYb7K now has16 source/test files. Full detailed evidence
and failure lineage: memory-repairs/admission-ownership-and-receipts-2026-09-06.md.
Ownership2ee642fe compares full listing/natural/index-target identity relationships;
only exactly unchanged defects survive. Projectionc608c2d9 fixes a genuine two-order
type-removal dependency failure without test weakening. Routesad0657c1 refuses
original declaration alias bindings removed before final projection, covering
primary/inverse/hint/value-source-flip and preserving unrelated utility. Full
ownership/routes suite PASS71.571/17.042/27.833, loga0d46617.

Episode pinningb7213c75 keeps actual original EP bytes and rejects changed source/
time/repository identity; first tests PASS.843s, log1cfe2b2b. Prospective assertion
receipts784aa087 independently encode FA disposition and resolved endpoint state,
full identity/address, explicit accepted materialization versus baseline presence,
frozen hint candidates/target/end, combined effects/suppliers and owned transitive
evidence. First tests PASS.622s, logc0a33cb1. No writers or durable V2 results yet;
existing heads/carried occurrences deliberately refuse until replay is connected.
Full16-file uncached suite PASS72.784/17.215/27.893, logec20343f; vet PASS.

Read private REPLAY_IMPLEMENTATION_NOTES.md f9d6f2be before implementing the next
phase. Accepted primary/hint programs must be skipped before identity discovery,
not merely filtered at the writer. Granular accepted metadata/type/alias/vote
actions must not refill or re-evaluate later reviewed changes; partially accepted
declarations cannot be relabeled all-deferred to replace revisions. Need actual
complete V2 result/head decoder, declaration/birth records, terminal Force carry,
durable unselected conflicts, exact materialization/finalizer/normal Apply, then
writer floor and original full-goal bars. No fresh bounded helper review as delivery.

New note276f8621326d7dc947a3b9dc50b84ffaa07eedff54df4e9f2e7ce90ebb13ee26
accepted once depth38, not yet backup-verified; never retry. Latest actual snapshot
remains132039, prior059723bc closure already verified. Live a06cd7b untouched.

## Continuation — 2026-09-06, after15:28 UTC

Private complete-admission6qYb7K now22 admission files plus exact retained fresh
independent tests. See memory-repairs/admission-complete-result-progress-2026-09-06.md
for complete source/failure pins. Frozen18 review c9f6b0b9 NO-GO (room177) proved
deferred address collisions omitted actual occupant evidence. Root86798fee adds
separate AddressBefore and genuine point witness; unchanged root/independent tests
pass. No independent correction approval yet. Frozen DAf00M remains unchanged.

Complete V2 envelope49eddcf9 and actual previous-reader0ff7d015 are private only.
Four new result-validator corruption tests initially failed; first source/test/log
preserved, source-only fix passes assertions. Prior reader first tests PASS.707s:
actual head/result/input/EP binding, exact accepted FA/effect/provenance/closed-end
proof, changed input cannot erase acceptance, old V1 is acceptance-unknown. Reader
tests explicitly seed synthetic result rows, not an implemented writer.
Full uncached no-CGO suite PASS store72.600s/resolve17.131s/daemon29.854s;
log12f7c67c1e53fcb41de834088004899783ea22217cd3766be06315bbe652e51b; vet PASS.

Next remains composed terminal replay BEFORE discovery, granular partial declaration
action carry, complete evidence/action/lifecycle validation, durable changed-input
conflict after successful transaction, exact materialization/fixed final inventories,
normal Apply/policy deduplication and all-writer/schema floor. Existing receipt
builders still refuse prior heads until actual replay is connected. No live changes,
new actual backup, adoption or goal completion. Note276f8621 never retry;132039 remains
latest actual snapshot and059723bc closure already done. Keep full goal active.

## Continuation — 2026-09-06, after16:06 UTC

Private6qYb7K now26 admission files plus four exact independent tests. Read
memory-repairs/terminal-replay-progress-2026-09-06.md for full source/failure pins.
Terminal replay is connected before discovery in the fixed read-only draft wrapper;
completed declarations carry without refilling metadata/aliases/votes. Structural
evidence closure is validated. Partial declarations still refuse before discovery:
replace this WIP guard with granular original-action replay before normal Apply.
Read private PARTIAL_DECLARATION_REPLAY_NOTES.md f2a37d80 before that implementation.

Frozen terminal review0482d10d NO-GO proved four preexisting generation swaps with
identical stable EN tuples. Root5514b8e now requires exact prior ig/il/il-consumed
selectors including absence. Root full suite PASS97.698/20.890/33.036, log8aee2619;
vet PASS. Both unchanged independent test files PASS .947s, logaf9df500. Room178
records review and correction; correction not independently approved. Frozen
khrGn1 source unchanged. No active reviewer remains.

Newest actual backup155247 SHA57d933d5a260ea214d52cd2ab5cee334bb8a3d9d5af349be5986165bb2430552
restored raw-exact250664 rows in shared-155247. Note276f8621 now verified INGESTED,
not pending/parked, attempts0; never retry. This is restore/note evidence ONLY, not
a candidate probe. Last candidate replica check remains132039. Main production
e097fa6/both live a06cd7b unchanged. Next: partial action replay, full semantic
result validation, durable conflict/result selection, actual materialization and
fixed inventories, normal Apply/policy deduplication/all-writer/schema floor, then
remaining original live cleanup and two full independent grading rounds. Keep active.

## Continuation — 2026-09-06, after16:38 UTC

Private6qYb7K now has original partial action replay; temporary refusal guard is
preserved as history and replaced by same-fixture exact no-op success. See
memory-repairs/partial-action-replay-progress-2026-09-06.md for all source/log pins.
Partial source9b017441 and replay2b0b9102 execute only original deferred actions;
receipt merge6fa308b4 carries terminal actions byte-exact. Shared alias predicate/
effect44741865 makes the second overlay successor file. Same-state retries, type
already satisfied, duplicate aliases in both orders, exact old removal present/
absent, later removal refusal and lost spelling across successor retries pass.
Fixtures use synthetic selected rows, not an actual completed writer.

Action validationdba1402e/resultb2da6362 reject twelve proven action-effect corruptions
and three wrong-birth/component corruptions. Root full suite PASS70.257/16.888/29.735,
log31184c34; vet PASS. No independent approval of these successor sources. New tests
and failures preserved; two fixture corrections are fully disclosed in report.

Next: compose actual materialization/result selection and fixed actual inventories
in the same owner, including durable changed-revision conflicts returned AFTER a
successful preservation transaction. Current input/EP/result readers are available;
do not return conflict from inside a transaction that must preserve its new input.
Need more retained-vote/deferred-effect tests, complete hint semantic coherence,
normal Apply/policy deduplication/all-writer/schema floor and original live bars.
No new actual backup/probe/live change/note. Latest155247 only restore/note closure,
last actual candidate132039. Goal active; no final round passed.
