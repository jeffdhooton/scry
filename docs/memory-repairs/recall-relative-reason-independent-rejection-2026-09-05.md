# Relative reason-prior independent disproof — 2026-09-05

**Verdict: FAIL as a general code fix. Do not promote or deploy this variant.** The one benchmark gain is real, but an independent named fixture proves a regression: an irrelevant, lexically strong new fact suppresses the correct explanation's bonus enough to displace it. The old rule keeps the explanation first on the identical final corpus. The original recall floors remain unmet. This is a bounded private review, not final recall grading or whole-goal approval.

## Scope and exact inputs

Review archive: `/tmp/scry-relative-reason-disproof-sep05.tLciBh`.

- Fresh source archive: `14d3057be99827cbdf85529019f2a00e2d8a75be`. The benchmark candidate was built before adding diagnostic instrumentation, from this archive with only the reviewed `query.go` patch and the builder's two helper tests. Its `query.go` was byte-identical to the supplied experiment file; SHA-256 `7b00ddeb996f10162760d3a3196b1243337cb8ba7ad7b1fad90aa11c9e73809b`.
- Reviewed diff is `reviewed.patch`, SHA-256 `5e685804aa0eb28c69931afca8c4a69680add1374dd442f7ec5e7837202bfe0c`. It changes only the global reason-candidate bonus from `8` to `8 * lexical / bestLexical`, where the best score is the first sorted lexical candidate. The helper clamps nonpositive scores/best to zero and scores above best to best. Entity injection, search, expansion, relation vocabulary, vectors, diversity, caps and questions are untouched.
- Preregistered proposal: `/tmp/scry-recall-scoring-sep05.EHNmnp/REASON_PRIOR_EXPERIMENT.md`, SHA-256 `71ecf0e40a889fb4bdbd347e209ae9ba3552a6a42bf9ecaf4033cdbd6d6d0ae5`.
- Source backup: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260905T230854Z.badger`, independently verified SHA-256 `188e5bd97b24221c61ac86b4e605f6f05dc0d8184d39be8a464ece4230c785ce`.
- Exact baseline CLI: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry`, independently verified SHA-256 `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.
- Independently built candidate CLI: `candidate`, SHA-256 `a96f60769d83309fd4c3e5f5f965b352f193135cbd9c502b8e15c579624dbf28`. This is a private development build, not a production artifact. The supplied root candidate independently verified as `61e0024a33c55edb8ef93bcca4398f65f52994b833414baf5d80b8fdd6b3e0e5`.

I read the applicable instructions, ran required orientation first, inspected the actual patch, the existing recall-floor disproof, and the relevant failed expansion/coverage/vector/weight decisions. The original-query-only experiment was already rejected and is absent from this candidate. This review did not sweep weights, change expectations, create another ranking variant, or author a final fifty-question set.

## Independent design counterexample

The new fixtures use Asterbridge, Blueharbor, Velanora, Dunmere and Quillforge, chosen outside the supplied graph. They were written before execution; a first setup run failed because the fixture lacked required endpoint entities. Only the fixture setup was corrected. No text or score was fitted after observing results.

Question: **“Why did Asterbridge stall during amber zephyr quasar deployment?”**

The correct explanation is `asterbridge causes certificate-expiry`: “Asterbridge stalled because amber certificate expired.” An ordinary status fact says “Asterbridge deployment stall status was recorded during amber testing.” The fixture then adds exactly one unrelated fact on Quillforge: **“Asterbridge stall during amber zephyr quasar deployment is the title of a fictional book.”** It has relation `status` and no endpoint in common with the explanation. All facts have fixed old timestamps. There is no vector model, no episode bonus, and no candidate clipping in this small fixture.

| Fixed-corpus measurement | Old rule | Candidate |
|---|---:|---:|
| Correct explanation rank before outsider | 1 | 1 |
| Correct explanation rank after outsider | 1 | 2 |
| Correct explanation score after outsider | 10.766 | 5.295 |
| Irrelevant book-title score after outsider | 6.772 | 6.772 |
| Correct explanation bonus after outsider | 8 | 2.529040 |

The outsider does change BM25 corpus statistics. This is explicitly separated from the proposed mechanism: after insertion, **both versions have the same answer lexical score 2.140852, the same maximum lexical score 6.772064, and the same 0.625 named-entity term**. The old formula gives `2.140852 + 8 + 0.625 = 10.765852`; the proposed formula gives `2.140852 + 8*2.140852/6.772064 + 0.625 = 5.294892`. Thus the final-corpus reversal is caused by the bonus change, not differing IDF, document length, embeddings, corpus contents or graph ownership between versions. Before insertion the candidate bonus was 5.228104; afterward it is 2.529040.

The denominator also creates pair-order dependence even when the compared lexical scores are held exactly fixed. For reason fact A with score 10 and plain fact B with score 15, A's margin is `10 + 8*10/15 - 15 = +0.333333`. Introducing an independent maximum of 100 changes only that margin to `10 + 8*10/100 - 15 = -4.2`. Under the old rule A's margin is +3 in both cases. This algebraic fixture isolates denominator sensitivity completely from BM25 changes.

The helper is bounded and monotonic **for a fixed denominator**; the tests pass zero/max/scale cases and a 404-point finite-positive monotonicity grid. Those local properties do not imply corpus stability. A lexically strong outlier controls the prior for every other candidate. The highest lexical reason-labeled distractor also still receives all eight points: the independent Velanora explanation is extracted as `status`, while an irrelevant `depends_on` checklist gets +8 and wins. This second fixture demonstrates the original lossy-relation problem remains, rather than proving an additional new regression.

Full executable fixtures and outputs are in `code/internal/memory/recall/independent_disproof_test.go`, `baseline-code/internal/memory/recall/independent_disproof_test.go`, `fixture-results.txt`, and `baseline-fixture-results.txt`. Baseline fixture logging uses the actual old constant bonus; candidate fixture logging uses the proposed bonus. No benchmark-specific name exception is involved.

## Five unchanged suites and individual regressions

Both exact CLIs ran all five suites sequentially against the same fresh restore, top twenty, rebuilding the index for each suite. Full CLI outputs are `baseline-<suite>.json` and `candidate-<suite>.json`.

| Suite | Baseline | Candidate | Required floor | Candidate maximum bytes | Baseline/candidate mean ms |
|---|---:|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 | 51/62 | 53/62 | 12,196 | 48.24 / 48.65 |
| heldout-b | 29/66 | 30/66 | 34/66 | 13,800 | 38.70 / 38.82 |
| probes | 7/7 | 7/7 | 7/7 | 10,094 | 22.29 / 20.57 |
| tuning-strict | 45/50 | 45/50 | 44/50 | 11,530 | 44.22 / 44.64 |
| tuning | 47/50 | 47/50 | 47/50 | 11,530 | 44.66 / 45.12 |

Zero responses exceeded 24 KB. These single-run timings show no material local slowdown; they are not a production latency confidence interval.

Exactly one existing question becomes a top-twenty hit: heldout-b Q32, **“Why did the daemon keep behaving like the old code after I rebuilt it?”**, moves **31 → 4**, before and after diversity. No existing top-twenty benchmark answer is lost. The answering `cockpit calls cockpit-daemon` fact's own score stays **25.023**; it gains because competing reason-labeled facts lose bonus. An unrelated Scry restart fact drops **30.431 → 25.638**. The highest-scoring `sweep-agent depends_on scry` fact remains **43.783** with its full prior and still fails to explain the Cockpit incident.

The gained answer is source-supported. Stored episode `1e21e21bbb57b8a8f89a8ef8c6ed1a3eb5e65e7ae8445e65179a80c167480298` points to `/Users/jeff/.claude/projects/-Users-jeff-workspace-cockpit/5760e80b-50ef-46d1-ad10-8d9d8a888651.jsonl#2119961-2532055`. I independently read that byte range; assistant message `77f69b27-27e1-49c1-b920-a6963b7e3f0e`, timestamp `2026-09-01T20:38:54.060Z`, explicitly explains that daemon stop did not restart the launch-agent process and that the old binary answered until `launchctl kickstart -k`. This is not a lucky match to an unrelated fact.

Aggregate hit counts conceal **16 regressed suite rows representing 14 distinct questions**, using the original matchers and full uncapped orders:

| Suite/question | Answer rank before → after |
|---|---:|
| heldout-2026-09-03 Q9 — offline CRM runner | 15 → 16 |
| heldout-2026-09-03 Q13 — deletion trigger | 8 → 10 |
| heldout-2026-09-03 Q15 — nested repo unindexed | 27 → 44 |
| heldout-2026-09-03 Q35 — small runner build failure | 1 → 2 |
| heldout-2026-09-03 Q44 — Colorado guide destination | 9 → 13 |
| heldout-2026-09-03 Q55 — disappeared memory write | 35 → 59 |
| heldout-b Q5 — spreading GLM across machines | 1 → 2 |
| heldout-b Q15 — sales tax blocker | 55 → 59 |
| heldout-b Q19 — Sentry unusable | 11 → 13 |
| heldout-b Q21 — worker default port | 8 → 17 |
| heldout-b Q39 — voice traffic detour | 522 → 919 |
| tuning-strict and tuning Q38 — outside watchdog | 15 → 18 each |
| tuning-strict and tuning Q40 — private origin parked | 4 → 5 each |
| tuning Q43 — fleet commit hook | 5 → 7 |

These are matcher ranks, not a new semantic grader. For example, the worker-port question also has an explanatory sentence at baseline rank 2 that its unchanged expectation does not accept; the table does not assert all useful port evidence disappeared. The nested-folder regression is direct: the current `subdirectory-query causes daemon-registry` sentence explicitly says exact-path matching omitted ancestors. Its lexical score stays 10.429582 against best 23.055588, so the valid explanation's prior falls from 8 to about 3.619. The lost-write sentence explicitly describes reasoning exhausting the output budget and `stop_reason max_tokens`; it receives a reduced prior as well. Neither original first-suite exact expectation absent from the store is repaired by this change.

## Preservation, candidate boundaries and determinism

The fresh restore contains 80,710 facts, including 72,885 current and 7,825 invalidated facts; 30,658 entities; 9,360 episodes; 52,017 alias claims; three rejection records; and ten pending records. Full raw key/value maps were captured before benchmarking, afterward and after all diagnostics. `raw-before.json`, `raw-after.json`, and `raw-last.json` are byte-identical under `cmp`, file SHA-256 **`fa004f684b246706fc779150d0d6c8c57a4b6c7337f580884745104ed9bb9ba5`**. Their length-prefixed raw-stream SHA-256 is **`2ff1ea657a217bd3d038da694d1e18bc2ca34ae7b1758006f3fe79dfc52c6d12`**. This includes every historical fact, provenance, queue record, rejection and metadata record, not just current assertions.

Recall/search package tests passed for the exact candidate source. The full `go test ./...` passed in the private candidate archive after diagnostic instrumentation was added; its flag defaults false. Existing historical/as-of, nil-index and cap tests remain passing. Full test output is `all-tests.txt`. The instrumentation was added only after building the benchmark artifact and never deployed.

All 235 suite rows have fresh baseline/candidate ordinary results and complete before/after-diversity traces under `baseline-<suite>/NNN.json` and `candidate-<suite>/NNN.json`. **All 181 non-reason questions have byte-identical complete ordinary result objects** across these runs. All 54 reason rows have equal candidate sets. Thirteen non-reason rows differ only in candidate-tail membership despite identical top-twenty results; this is an existing cutoff-tie issue, not a candidate-count or query change.

Universal determinism does **not** pass. The strict production-prefix assertion initially stopped both candidate and baseline diagnostics on the comma/Sheets question. Independent repeats of that same question with one fixed index yielded two distinct full results in 30 calls for each version: baseline 16/14 and candidate 15/15. `cs-345` and `cs-346` carry identical fact text, score and timestamp; one or the other survives duplicate suppression at rank 4. Search sorts tied scores only by timestamp, leaving tied map iteration unresolved, and recall preserves incoming ties. The candidate did not introduce this defect. Final diagnostic helpers log rather than hide mismatches; the final baseline pass logged one mismatch. No whole-goal or deterministic-output PASS is asserted. All 235 question comparisons, candidate-tail differences and repeat hashes are retained in `analysis.json`.

## Evidence index and reproduction

- `analysis.json`: SHA-256 `18e9538874bd905df98cc94c30f39df0c3a8753540f0041b32d0424a07037f75`.
- `fixture-results.txt`: `3988f73cb48325d60f8c231e40b9486915adef724a9905adaec7b183fb0995cb`.
- `baseline-fixture-results.txt`: `add6ce6cdce221e34820be3a2e8de1e5b978970f6edae6bbc9e4f6febe9fabe5`.
- `all-tests.txt`: `cfd1c90a990d0361f07bc9749ad0a1e5e320c34008388c81d6489febd1284c1e`.
- `candidate-repeat.jsonl`: `074d5a0740263c834805aa84f781500d48c5f95a0534c3e4bb6f2584ccc54970`.
- `baseline-repeat.jsonl`: `980750fd11c86da88c0c91c9e81d86eeb2850b1efb4f300b64b3f0a58231e2b6`.

`run.sh` records the exact five-suite sequence and raw comparisons. `details.sh`, `baseline-details.sh`, and `finish-details.sh` record diagnostic commands, including the initial strict-prefix stopping condition and the completing pass. `analyze.mjs` produces the question-level comparison from the unchanged expectations. To reproduce fixtures, run `go test ./internal/memory/recall -run TestIndependent -v` in `code/`, and `go test ./internal/memory/recall -run TestIndependentExternalNames -v` in `baseline-code/`. The latter uses old scoring plus private diagnostics whose flag remains false.

All writes were private build/test/report files, disposable synthetic fixture stores, or restoration of the supplied backup. The restored fixed graph had no logical mutation. Beyond the required startup orientation, no live-store interaction occurred. No shared source edit, user-workflow-file edit, provider call, new memory, room post, live repair, queue change, source-retention change, configuration change or deployment occurred.

The baseline defect justifies studying relation-derived relevance. This particular normalization fails the independent disproof and should be archived as a rejected experiment. The one gain does not restore the original floors, establish fresh held-out performance, or justify restarting a fitted ranking sweep.
