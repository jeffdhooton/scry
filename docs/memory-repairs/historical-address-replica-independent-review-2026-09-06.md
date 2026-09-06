# Independent historical-address replica compatibility grade

Verdict: **scoped replica-compatibility PASS** for the pinned private candidate. No new loss of an existing assertion, provenance, invalidation, raw payload or current recurrence was proved within the exact-address contract. Six literal historical replays on the full shared replica preserve exactly the intended historical evidence update. This is not a whole-goal, integration, deployment, recovery, schema-format or general temporal-model certification. The original recall goal still fails.

## Scope and independence

Read the complete active goal at `/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md`, the full candidate `HISTORICAL_ADDRESS_PROPOSAL.md`, and both repository historical-address independent-review reports dated 2026-09-06. Ran `scry memory orient --cwd .` before other work. Independently inspected the historical helper, store raw guard and surrounding Apply path. Prior reports were context, not results counted as my proof.

All source, builds, helpers, restored stores and this report are under `/tmp/scry-historical-replica-grade.TR8rS0`. Candidate source was copied from `/tmp/scry-historical-address-sep06.Mbi8Rn`; baseline source was independently exported with `git archive a078240`. No production or supplied-test edit was made. New helper files were authored with apply_patch (starting from the inspected prior read-only inventory/restore helper), then gofmt. No subagent was spawned. No live/shared graph, queue, source/provider, configuration, recovery, migration, integration or deployment mutation occurred. Backup files were read only; every restore was direct Badger Load into a previously nonexistent private directory. No populated restore was used. Existing test suites run their own synthetic stores/fakes.

All graph-derived reports are counts and hashes. Raw texts, values, value-derived keys/slugs, source refs and error strings were kept in process memory and not emitted. Helper errors are hashed. The private restored databases necessarily contain the original graph, including its known sensitive content; do not publish these directories or inspect raw rows in a transcript.

## Exact source and artifact pins

The following candidate pins matched before and after grading, both at the supplied source and private copy. Final `diff -rq` found only the added private `cmd/replica-grade` helper directory.

| Relative candidate file | SHA-256 |
| --- | --- |
| internal/memory/resolve/resolve.go | 43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623 |
| internal/memory/resolve/historical.go | 9749aef36ee99197b2b57464cb9b12c7ebd2752d1f8037b7fdade591a4bafb4f |
| internal/memory/resolve/historical_test.go | c9e8d5d2ac411ad45e72fa42c63477b71cbfb347b22dcb346941bf204b5d696e |
| internal/memory/store/historical.go | 896ba4d7739eded14bf30df9d5f5afa23c6e2a85f86917f9e495af4dc14c7de6 |
| internal/memory/store/historical_test.go | 04773bd70e86428d94419d28bbda6505a4ac47c0126508da2d2752dfdc285742 |

Baseline actual CLI `/tmp/scry-schema-refusal-deploy-sep06.uUcaFo/scry`: SHA-256 `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`.

Private `candidate-scry`, built with `CGO_ENABLED=0 go build -o ... ./cmd/scry`: SHA-256 `6d939a03b45b2220fd14d252f3964a286171699688960516ab03e4580a4c2c27`. This is a private build, not the parent's later integration/deployment artifact.

## Real backup preservation

Both complete backups were hashed before loading. After direct restore, raw Badger maps were captured; candidate Store.Open, AllFacts, Entities, offline recall/index construction and a bounded recall read ran; closing and recapturing the full raw map proved exact equality.

| Backup | Bytes | Backup SHA-256 | Raw records | Facts (current) | Entities |
| --- | ---: | --- | ---: | ---: | ---: |
| shared, `/tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T020346Z.badger` | 74921236 | 37187891a0bb081db2f35754df726432caba010e77fba5896c166c2ffa1b866e | 245425 | 81304 (73449) | 30956 |
| laptop, `/Users/jeff/.scry/backups/memory-20260906T020346Z.badger` | 19445008 | 6c19b215d0108dac99ad0f1781c7de0686b973c24044a984827a3ec3827c7027 | 83378 | 21004 (18495) | 14200 |

Shared raw logical digest is `022b8739f013ed98e2b9e96df7a117959fa533335bb43f5b32bfbd047a593f28`; laptop is `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`. The digest encodes length-prefixed sorted key/value bytes, not Badger physical files. Every original key family was included, including pending rows, cursors, retirement/value evidence and metadata. No unknown data was omitted from the raw comparison. Both five-suite CLI runs also preserve these complete maps exactly.

## Unchanged recall comparison

Both actual CLIs ran all five unchanged suites, top 20, on each replica. Every suite has identical hits, mean answer rank, complete miss membership/ranks and maximum payload. All responses remain below 24576 bytes; over-cap count is zero.

| Suite | Shared baseline = candidate | Shared mean answer rank | Shared max bytes | Laptop baseline = candidate |
| --- | --- | ---: | ---: | --- |
| heldout-2026-09-03 | 51/62 | 4.8431372549019605 | 12112 | 22/62 |
| heldout-b | 29/66 | 5.068965517241379 | 13366 | 13/66 |
| probes | 7/7 | 1 | 9962 | 5/7 |
| tuning-strict | 45/50 | 4.511111111111111 | 11533 | 28/50 |
| tuning | 47/50 | 3.978723404255319 | 11533 | 30/50 |

Additionally ran independent source-built offline recaller helpers over all 235 individual shared questions. The complete sanitized rank/payload records and final raw-preservation record compare byte-for-byte (`cmp` exit 0), SHA-256 `e50f2e835bccd590f56c0241702323a57ee58897a636803615fcd15a64e0a822`. CLI output hashes can differ due to fields beyond the asserted comparison; the individual source rank/payload equality supplies the stronger rank check. These are regression checks against immediate baseline, not satisfaction of original 53/62 and 34/66 thresholds. No new held-out set or latency/ingestion/sweep grade was attempted.

## Literal historical replay results

Selection is deterministic over sorted actual stored fact addresses. Historical rows must round-trip through Fact exactly. Endpoint input uses the existing entity's canonical name only when its natural slug and alias index already resolve to that exact stored endpoint. RawRelation, full Value, full assertion text and exact nanosecond ValidFrom are copied into extract.Fct in memory; inverse mappings reverse input endpoints as required. No semantic owner is inferred, changed or supplied to force success. Attribute eligibility additionally requires stable value classification under the current resolver. Every replay uses a deterministic unique synthetic episode and confidence 1 to exercise maximum-confidence handling, then normal `resolve.Apply` with DefaultExclusive.

The original shared snapshot contains 1760 canonical, 180 fallback and 5915 attribute historical facts. Conservative eligible counts are 1287, 145 and 3018 respectively, including 44, 26 and 96 that coexist with a current triple. Remaining rows are outside this bounded selection, not adjudicated or repaired. Full counts and selected address digests are in `shared-selection-final.jsonl`.

| Case | Selected legacy address SHA-256 | Candidate | a078240 baseline |
| --- | --- | --- | --- |
| canonical | bf0a4544b1e0358df1b29a995361842000128b0a0eea0c21675d48892f767281 | exact historical evidence merge | reopens history, replaces old provenance |
| fallback | cbe3a7e22431be22b25d64bfd9ecf5f3ac3b56eb2852688b53484fc83fdab4ec | exact historical evidence merge | atomic fact-conflict refusal |
| attribute | ee80009d9d944f2e499c67a42e9607aad9356e9cf9c6d7f2cef6ddf0813b4351 | exact historical evidence merge | reopens history, replaces old provenance |
| canonical, current coexistence | e8a25753ebd23a1ea4f5ef4030518dae56800605d39651515d1e01b7a4a0c674 | history merge; current untouched | atomic fact-conflict refusal |
| fallback, current coexistence | e24f18802b6c9069bc86d2f35b0438815642934c9f836624c4d45893aaf8e993 | history merge; current untouched | atomic fact-conflict refusal |
| attribute, current coexistence | af6bfde1d81ea333e2d274823024d9bb3dc73cb79080387482f32382ca21437e | history merge; current untouched | adds evidence to current row instead |

Each candidate replay independently compares full before/after raw maps. Its only delta is the exact selected fa payload plus one new ep record. The expected fa is the complete old serialized Fact with only the unique episode union and maximum confidence changed: InvalidAt, start, all content and all old provenance survive exactly. Every other one of 81303 existing facts and every old nonfact row remains byte-identical; no record is deleted; stats are one merge, zero additions, invalidations or entity updates/creations. All current coexistence rows remain untouched. Baseline refusals have zero stats and complete raw equality.

Replays ran sequentially on isolated candidate/baseline full replicas; preceding successful modifications are retained and covered by each next full-map comparison. A final audit against the pristine original shared map proves only six original candidate records changed, all six solely by evidence union/confidence. All 245419 other original rows are byte-identical; zero original records deleted; zero original assertion content/start or invalidation changed; zero old provenance IDs lost; zero confidence lowered. Baseline final audit detects two changed invalidations and two lost provenance IDs across its three modified old records. Full per-key delta hashes are retained in `replay-*.jsonl` and `original-to-{candidate,baseline}.jsonl`.

## Observed inherited attribute boundary and harness correction

The first attribute selector checked source identity and relation mapping but omitted present-day target value classification. Its selected legacy address `89d120032cbaa2aff39782ef4aa50780f9ae2b524134e1fbc4b413b43f795d0d` now resolves its target as an established entity. Normal Apply therefore reaches a different edge address, never the historical attribute address. The initial candidate preservation assertion failed. The baseline adds the same edge. No source fix or owner rewrite was made.

I corrected the selector's eligibility filter and used the independently eligible attribute shown above. Separately repeated the excluded input on a fresh pristine candidate replica without asserting historical-address eligibility. Candidate and baseline produce exactly identical three-record addition deltas (fa, adj, ep, same key/value hashes) and leave all 81304 original facts unchanged. `replay-legacy-attribute-candidate-diagnostic.jsonl` and `replay-attribute-baseline.jsonl` retain that evidence. This preexisting normalization/identity boundary remains outside the scoped PASS; the candidate cannot be advertised as preserving all legacy restatements regardless of endpoint reclassification.

A concurrent read-only selection attempt briefly encountered a private Badger lock while the shared benchmark owned that directory; its error was hashed. Retried only after the benchmark completed. No lock bypass, stale-lock removal or graph rewrite occurred. One harmless report-file lookup used an incorrect private path and failed; required proposal content had already been read completely at its actual path.

## Full-replica synthetic adversarial controls

New synthetic fixtures were added only to the private full shared replica, with endpoint names/aliases and slugs checked absent first. Real payloads were never changed to make a fixture pass.

Four normal Apply episodes exercise first assertion, exact self-superseding restatement, then a later distinct occupied-address collision, using canonical uses, status via measured, actual fallback aliases_index_to, and exclusive replaced_by. Every refusal is ErrFactConflict with zero returned stats, zero observer events and full raw-map equality, including all original real rows and all staged synthetic entities/indexes/episode changes. This extends the prior source matrix into the full replica; it does not claim real queue inputs were replayed.

For unsupported history, seeded two checked synthetic entities and one synthetic historical uses fact, then appended an unknown future_evidence raw JSON field. Captured the entire seed delta (six added records) and proved every preexisting row unchanged. A touched exact restatement refuses with zero stats/events and no raw delta. An unrelated depends_on input succeeds beside it, adds exactly fa/adj/ep, and preserves every preexisting raw record including the unsupported historical payload. Complete delta hashes are in `synthetic.jsonl`. The old separately identified collision snapshot was not replayed in this task; no historical recovery was attempted.

## Commands and verification accounting

All commands below ran within this private review directory or its candidate/baseline source. Every restore destination was absent beforehand. Shell redirection saved only sanitized helper output and test logs.

1. `mktemp -d /tmp/scry-historical-replica-grade.XXXXXX`; copied the frozen candidate to `candidate`; exported baseline with `git archive a078240 | tar -x -C .../baseline`; verified source pins with `shasum -a 256`.
2. `CGO_ENABLED=0 go test ./... -count=1` in candidate: exit 0. Original source/supplied tests unchanged. First run started before helper files were added; a second full run with the helper package present also exited 0 and was saved to `full-suite.log`. No claim depends on helper tests (there are none). Queue, resolver, store and daemon packages all passed. No external provider test was enabled.
3. `CGO_ENABLED=0 go build -o .../candidate-scry ./cmd/scry`; `CGO_ENABLED=0 go build -o .../{candidate,baseline}-helper ./cmd/replica-grade`: all exit 0. Final compare-only helper was rebuilt successfully after the full-suite run; no production change followed grading.
4. `candidate-helper restore BACKUP NEW_PRIVATE_DIR EXPECTED_SHA`: shared, laptop and fresh shared comparison directories. Baseline helper also independently restored its replay replica. The helper requires nonexistent destination, verifies the whole input hash, opens empty Badger, calls Load once and checks complete raw preservation around Store.Open/read/index.
5. `candidate-helper bench REPLICA candidate BASELINE_ACTUAL_CLI CANDIDATE_CLI`: shared and laptop, exit 0. Bench invokes `memory bench --dir REPLICA --file docs/memory-bench/SUITE.json --top 20` for both CLIs, all five suites, and preserves full original raw maps.
6. `candidate-helper select PRISTINE_SHARED`; `candidate-helper replay CANDIDATE_REPLICA KEY_SHA candidate`; `baseline-helper replay BASELINE_REPLICA KEY_SHA baseline`: the six rows listed above. Final eligible runs exit 0. The initial ineligible legacy-attribute candidate expectation exited 2 as documented, followed by independent diagnostic reproduction and selector correction.
7. `candidate-helper synthetic CANDIDATE_REPLICA`: exit 0, all four rollback shapes and both unsupported-payload controls pass.
8. `{candidate,baseline}-helper ranks DISTINCT_PRISTINE_SHARED_REPLICA SOURCE_ROOT`; `cmp shared-ranks-candidate.jsonl shared-ranks-baseline.jsonl`: all exit 0, all 235 ranks and payload sizes identical and raw unchanged. The later diagnostic ran only after candidate ranks finished.
9. `candidate-helper compare-original PRISTINE_SHARED REPLAY_REPLICA`: candidate and baseline final full-original-map audits, both exit 0. Inspected count and complete digest-only delta output.
10. Final source pin checks unchanged; `diff -rq SUPPLIED_CANDIDATE candidate` reports only private helper directory, expected exit 1.

Evidence hashes:

- `full-suite.log`: 056541f9249163eb8c14d3e9ce41b6e9a4676c62e14d65cdda8d026f17aa84af
- `shared-bench.jsonl`: 4c2b4e78246e8e9ac589f97e0e909568452938181e8f4dfa8c5d873c0ebde19d
- `laptop-bench.jsonl`: fa8ae2ceb1eea48fdf755ff88673a9d8061ea47931c831f2b9b73b80ebf6cc46
- `synthetic.jsonl`: 5818c9e94876ae9f08b8fdc96e44506c19586d5c59df7c40a8b71d287d1fc346
- helper `main.go`: f8a6502b75ce9a4aa12c1225a344120e30adcb6dcc39d5178f011f15921b95d1
- helper `replay.go`: 94b644ca6baa081fc34b7955db50e355dfb3015c0aa1c1fb60bb0c7a82973592
- helper `synthetic.go`: 30edacae54a88673c742dd1ea0b108435255cc62786810560232a83a6aea82e7
- helper `compare.go`: 4ec40e8b7b5b3107e46d745e429f66ca7843db75e5107b0c8f26df2b9f07026c

## Limits

This grades the precise candidate historical branch on real restored data plus bounded synthetic controls. It does not close general current-triple coalescing/backdating, direct PutFact replacement, unsupported current-row rewriting, malformed dates outside touched history, normalized identity/exclusive-target ambiguity, exact supersession identity, temporal intent, or legacy UnixNano/address collisions. It repairs no old loss. Literal replay inputs are synthetic episode wrappers around existing facts, not retained transcripts or real pending queue input. Attribute endpoint reclassification is independently reproduced and explicitly excluded above. Actual deployed a078240 schema startup is a separate parent review. No final integration artifact, live daemon/store write, backup/recovery operation, sweep, newly held-out recall bar, latency bar or complete goal is approved by this report.
