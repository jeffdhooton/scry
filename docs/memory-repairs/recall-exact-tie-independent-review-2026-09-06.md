# Exact tie-order independent disproof — 2026-09-06

Verdict: **PASS for the bounded exact-tie source change, with the limits below. Universal recall determinism remains FAIL.** Search resolves equal finite scores and equal validity instants by its unique live document key before clipping. Recall resolves equal rounded scores and equal validity instants by its existing hit key before diversity, when those hit keys differ. The real Sheets failure becomes reproducible across the measured repeats and rebuilds. No question's expected-answer rank regresses in the measured five-suite comparison. This is not recall-floor restoration, deployment approval for an unspecified artifact, fresh held-out grading, or whole-goal approval.

Review archive: `/tmp/scry-exact-tie-disproof-sep06.part2q`. The root remains the sole shared/live writer. I ran required orientation, read the goal, supplied instructions, actual complete four-file diff, exact-tie proposal, prior independent rejection, relevant decision history, and implementation paths. All subsequent writes were private archives, executable diagnostics, reports, synthetic fixture stores, and restores of the supplied backup. No shared source edit, live graph edit, queue edit, room post, remember, provider call, deployment, or new transcript retention occurred.

## Exact inputs and build

- Source HEAD: `3bfa0d21f728d508af30992626a55525876485df`. `exact-code/` is a fresh `git archive` of this exact commit plus ONLY `reviewed.patch`, covering `internal/memory/search/index.go`, `index_test.go`, `internal/memory/recall/query.go`, and `query_test.go`. Untracked user assessment and root review documents were not copied into it.
- Patch SHA-256: `eacea4af06131b17247fa578a7754cbbf8b8423b9c3b5f375896fb1a5147b8d1`.
- Exact final production-file SHA-256: recall `3b269fe49f03526b5f24cd86a53f7c5a6482aa61757667692273635daf4da746`; search `8dcad681f90f1deefc34a36d7434faa93a9f8f2feb402ed2c566bc02c1807725`.
- Pinned baseline CLI: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry`, independently verified SHA-256 `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.
- Clean no-CGO candidate: `candidate-static`, SHA-256 `8f3757a1971ccb856eb470736e3e66f4a41be59490ea9cc4b5c58009a64b4c53`, built in `exact-code/` with `CGO_ENABLED=0 go build -o ../candidate-static ./cmd/scry`, Go 1.26.2 darwin/arm64. `go version -m` confirms `CGO_ENABLED=0`. This private build has no production version ldflag and is not presented as the final deployed artifact.
- An earlier ancillary native-default build, `candidate` (`6c161b0e7d2ec197dade05f0cdbe9ce0b32acd58774d5e403738fb206521bb7c`), ran the initial ten-suite sequence. Its build metadata had CGO enabled, so I independently rebuilt the exact no-CGO artifact and repeated all ten baseline/candidate suite runs on a second fresh restore. The table below is the latter sequence.
- Frozen backup: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260905T235134Z.badger`, independently verified SHA-256 `7680cdcff3563c4cc2d33577506fc4c4aad4931d8bae1016f90e617bd7ee863e`.

The diff changes two comparators only, retaining descending score and descending time before the new lexical key. No weights, relation priors, synonyms, candidate counts, questions, expected answers, facts, vector rules, headers, or cap changed. Neither rejected original-query-only nor relative-reason code is present. Committed production code between the earlier baseline source `14d3057be99827cbdf85529019f2a00e2d8a75be` and this HEAD has no `internal/` or `cmd/` diff. The builder's helper-name compile typo is absent from the reviewed artifact.

## Independent fixtures and tests

New external identifiers include Orenvale, Zephyria, Arden, Veloria and Cedar, with old fixed timestamps and synthetic evidence chosen independently of the benchmark graph. Executable tests are in `code/internal/memory/{search,recall}/independent_exact_test.go`, with copies in `baseline-code/`.

- Search: 41 identical documents, five-place cutoff, eight different insertion/build orders, fifty repeat queries per build, alternating timezone representations of the same nanosecond instant. Every candidate run picks lexical keys 00–04. Baseline fails on its first run with arbitrary tied keys.
- Search precedence: a newer timestamp wins despite its later key; a lexically weaker document remains last despite its newer timestamp and earlier key.
- Recall: twelve equal-weight named entities, identical duplicate text and equal timestamps across zones. Both nil-index and empty-index injection paths run 150 times. Candidate always puts `orenvale-00` first, with byte-identical complete results; baseline fails immediately on a different duplicate representative. Empty index deliberately forces the named-map injection path, independently of lexical search.
- Recall precedence: the higher score wins despite an older date/later key; with equal scores the newer fact wins. Both versions pass this unchanged contract.
- Existing builder tests also exercise reversed lexical insertion and stable duplicate representatives.

The baseline FAIL is an expected regression-test result, not a harness compile failure. Separate limitation fixtures deliberately assert and pass reproduction of preexisting nondeterminism and key collisions, described below. No production workaround was written.

`go test ./...` passed on the exact four-file source before diagnostic additions. `CGO_ENABLED=0 go test ./...` also passed in the clean `exact-code/` archive. Focused `go test -race ./internal/memory/search ./internal/memory/recall` passed with independent fixtures, before diagnostic helpers were added. Existing as-of/history, nil-index and cap tests remain green.

## All five unchanged benchmark suites

`static-bench.sh` runs pinned baseline and independently built no-CGO candidate, offline, top twenty, rebuilding the index for every suite. Every suite uses the exact unchanged JSON in `exact-code/docs/memory-bench/`. Complete results, including every miss, are `static-run-baseline-<suite>.json` and `static-run-static-<suite>.json`.

| Suite | Baseline → candidate hits | Misses each | Mean answer rank, both | Maximum bytes, both | Mean ms before → after |
|---|---:|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 → 51/62 | 11 | 4.843137 | 12,110 | 49.18 → 48.65 |
| heldout-b | 29/66 → 29/66 | 37 | 5.103448 | 13,369 | 39.88 → 40.41 |
| probes | 7/7 → 7/7 | 0 | 1 | 10,095 | 21.86 → 22.29 |
| tuning-strict | 45/50 → 45/50 | 5 | 4.511111 | 11,528 | 43.26 → 45.14 |
| tuning | 47/50 → 47/50 | 3 | 3.978723 | 11,528 | 45.42 → 45.88 |

Zero responses exceed 24,576 bytes. Complete miss arrays are identical. The timings are single local runs, not a production latency confidence interval. The original 53/62 and 34/66 floors remain unmet.

All 235 suite rows also have independent ordinary results and uncapped full fact orders under `baseline-<suite>/NNN.json` and `candidate-<suite>/NNN.json`. `analysis.json` records each unchanged matcher and answer rank: **zero top-twenty losses, zero uncapped answer-rank regressions, and zero answer-rank gains** in this measured comparison. This is individual-question measurement, not inference from equal aggregates.

Full payload equality is not claimed. Three ordinary rows differ: tuning-strict Q10 and tuning Q10 choose a different tied latency-probe representative at rank 18 (`laptop-mcp-host calls`, score 25.608, instead of `uses`, same score/time); tuning-strict Q24 chooses `cs-345` instead of `cs-346` at rank 4, with the corresponding entity header. The answering-fact matcher ranks stay unchanged. 232 uncapped fact lists differ somewhere, primarily because arbitrary tail tie orders now have keys. Seventeen rows change membership at the 4,000 lexical-candidate cutoff; their exact added/removed keys are retained in `analysis.json`. Five common facts also differ in score on four questions through the separate preexisting injection issue below, so I do not attribute every tail difference solely to tie sorting.

Diagnostic helpers are private additions to `code/` and `baseline-code/`; `exact-code/` remains untouched. `IndependentWideRecall` is a copy of each version's Recall with only its name changed and final payload cap omitted, called with limit 100,000. Ordinary results call unmodified production Recall. No score formula, query expansion, or diversity logic changes in the helper. Its ordinary/full prefix cross-check finds zero candidate mismatches and one baseline mismatch (tuning Q24, the known alternating duplicate). The benchmark CLI binaries were built separately from these helpers.

## Real Sheets failure: repeated and rebuilt

Question: “Why did a comma in a note string wreck neighbouring cells with gog sheets update, and what separators are safe?” The expectation is the existing tuning-strict Q24, unchanged.

Each version ran thirty ordinary queries on each of three independently rebuilt indexes against the same frozen replica. Every iteration also measured the entire uncapped fact order and complete 4,000-key lexical candidate order.

- Baseline: two complete result hashes in 90 calls, 44 with `cs-346` and 46 with `cs-345`; each of the three builds exhibits both. Two uncapped fact-order hashes; ninety distinct lexical candidate-order hashes.
- Candidate: one complete result hash, one uncapped fact-order hash, and one lexical candidate-order hash across all ninety calls and all three builds. `cs-345` is the representative every time.
- Candidate full result hash: `e5e783bfeb85593a70895c915dfb905e36a983684d0ec5811f2de5ca19de8d72`; uncapped fact-order hash: `6910376e084706a4565f23cadcf3b9632cbe7d90dd7931c69521fd6529383c20`; candidate-order hash: `bd10b7be932808d7e8246c5614c4fe37cb08f1547cd74420bcb555e07c83f08c`.

This establishes bounded reproducibility for this fixed graph, query, unchanged time-window membership and rebuild path. It does not establish determinism for every query, concurrent graph mutation, arbitrary insertion histories of the embedding model, or future recency-boundary crossings.

## Independently proven limits: universal determinism still fails

**Unequal named-endpoint weights.** Before the new comparator runs, injection traverses `named` as a map. The first endpoint that visits a shared fact scores it using that endpoint's `w`, then sets `seen`; a later endpoint is skipped. A synthetic Veloria/Cedar fixture gives Veloria the exact-query alias weight 2 and Cedar token weight 1. Across 300 identical candidate calls, nil-index scores alternate between 0.1 and 0.2; empty-index scores alternate between 0.725 and 0.95. Baseline reproduces the same values. Sorting afterward cannot recover a score that was selected earlier by map order. This is a separate scoring contract, outside the equal-score tie change.

This is also present on the real frozen corpus. I independently repeated four previously observed questions thirty times per version. All four have one ordinary top-twenty result hash per version, but candidate uncapped orders have **two hashes each**, while baseline has 4, 4, 6 and 8. Concrete score alternations include:

| Existing question | Shared fact | Scores seen in both versions |
|---|---|---:|
| heldout-2026-09-03 Q41 | `jeff owns dbafilingguide` | 12.205 / 16.593 |
| heldout-2026-09-03 Q41 | `jeff documents dbafilingguide` | 12.153 / 16.542 |
| heldout-2026-09-03 Q42 | `dbafilingguide merged_into west-virginia-guide` | 5.830 / 10.548 |
| tuning-strict Q12 | one `halo2 implements rpc-cluster` fact | 12.952 / 16.187 |
| tuning Q21 | `halo2 provides jeff` | 10.396 / 15.305 |

`baseline-real-limitations.jsonl`, `candidate-real-limitations.jsonl`, and `limitations-analysis.json` contain the questions, repeat hashes, timestamps and full selected facts. Some grouped triples also contain different historical fact identities; the listed pairs above were first identified by full identity including timestamp in `analysis.json`, not inferred by merging triples.

**Recall keys can collide.** `hitKey` uses `toHit`'s already clipped Value. Two facts with the same source/relation/time and values consisting of 200 `x` characters followed by `alpha` versus `beta` have distinct search FactKeys but the same clipped hitKey. The independent fixture proves this on both versions. Delimiter-based concatenation is not an injective encoding of unrestricted field strings either. Search's `byKey` ensures distinct live indexed documents have distinct nonempty Doc.Key; recall's hitKey has no corresponding universal uniqueness guarantee. Thus the recall comparator has a deterministic key rule, not a universal total order over all possible source facts. Existing `seen` already uses this colliding key before the new comparator. This patch neither creates nor repairs that semantic collision.

**Other ordering analysis.** BM25 visits query tokens in a deterministic sequence and accumulates each document's contributions in that sequence; map enumeration chooses hit emission order, which the new search comparator now resolves. Synonym extras are sorted. Diversity uses incoming order plus integer/set comparisons, so with fixed scores and distinct final keys it is deterministic. Entity-header totals accumulate through the returned fact slice in order; named-map entries initialize distinct slug totals independently, and headers already break ties by slug. Therefore header maps do not independently defeat the bounded claim once their inputs are fixed. Episode collection starts from ordered returned facts/provenance; it has only a date comparator but its input is fixed on this measured path. Embedding learning accumulates float32 vectors in document order, so different index insertion histories are not promised bit-identical by this tie-only change. Time-dependent recency remains deliberately unchanged.

## Whole-store no-write evidence

Both independently restored directories contain 80,844 facts (73,013 current, 7,831 invalidated), 30,725 entities, 9,371 episodes, 52,112 alias claims, four alias-rejection records, nineteen status-review records and nineteen retirement records. Raw prefixes also include 55,769 adjacency keys, 10,928 attestations, 3,134 cursors, twelve pending records, five metadata records and 761 value-evidence keys.

Full raw key/value captures before and after the initial benchmark, after all full-order/repeat diagnostics, after the extra real limitation probes, and before/after the separate no-CGO benchmark are byte-identical under `cmp`. These preserve every historical fact, provenance, queue/rejection/retirement record and metadata key, not just current assertions.

- JSON file SHA-256 for every capture: `f3587d2d7acfcb876f326ee7abd8b781d8a5ec737394d3d787308e6398b9afc9`.
- Length-prefixed raw key/value stream SHA-256: `3800785f9b2c746ff8850e1653ae77335fe842c138b4cead6c7ee505889c2160`.
- Artifacts: `restore.json`, `static-restore.json`, `raw-before.json`, `raw-after.json`, `raw-last.json`, `raw-final.json`, `static-raw-before.json`, `static-raw-after.json`.

Restore/raw operations reused the inspected read-only audit implementation from `/tmp/scry-recall-floor-disproof-sep05.IYB5c3/audit`; each restore and capture was independently executed. No rejected ranking implementation was inherited.

## Reproduction and evidence hashes

Run `bash static-bench.sh` for the exact no-CGO ten-suite sequence, `bash run.sh` for the initial ancillary sequence, and `bash details.sh` for all question-level orders and ninety-call Sheets repeats. `node analyze.mjs` compares every row. Run `./baseline-limitations replica > baseline-real-limitations.jsonl` and the candidate equivalent, then `node analyze-limitations.mjs`, for the four real nondeterminism probes. Restore commands require a new nonexistent private destination and the frozen backup above; raw checks are `audit raw <directory>` followed by `cmp` against the original capture.

In `exact-code/`, run `CGO_ENABLED=0 go test ./...`. In `code/`, run `go test ./internal/memory/search ./internal/memory/recall -run TestIndependent -v`; the same command in `baseline-code/` must fail the equal-tie regressions. For baseline limitation-only reproduction use the `TestIndependentUnequalEndpoint`, `TestIndependentHitKey`, and `TestIndependentRecallScoreAndDatePrecedence` filters.

| Artifact | SHA-256 |
|---|---|
| `static-all-tests.txt` | `1e0cc2cc0517f09dce8d08ab6e2a95b8b8befdc04807187f170f323cb897ee02` |
| `all-tests.txt` | `0084b323e1bf88af476c87cd7f9fb202f25e5b16470d0de60ebfcec5bc3de20b` |
| `race-tests.txt` | `5c766f38955a22d6308781c9c26990fde8d182ebd9ed18c8f2d98c633c9beb1c` |
| `fixture-results-final.txt` | `4c22259db51fab291dee51475551c3ddb7ae86c1c3b1123924fd7ac050347fea` |
| `baseline-fixture-results.txt` | `1b2cffcbfd33428b6679943f6f931e40fe3f6dc18f058bc0c673345a07d522a7` |
| `analysis.json` | `8e236d53460187791890b4bae171f13d142ec1c7c8e3abdf00bfdb2e88fb57f5` |
| `baseline-repeat.jsonl` | `9ea376850446ff9b08e78ade827e439571b66bfdd8fc9ef5c8ce408d368cadbc` |
| `candidate-repeat.jsonl` | `254144342e110fdcee6a49a7643c4da055af2cf4dc31e5bf8c1c720238f6b445` |
| `limitations-analysis.json` | `1eed7a1bb4c88f08a87eb13d1437f287ea9b08bf5472d543490dadc65a166d77` |
| `baseline-real-limitations.jsonl` | `f37d7dc8edaf9e4e4da6170cc5a2ffc577396d165aa77c166187929d646d7726` |
| `candidate-real-limitations.jsonl` | `94947febb447488c202d1e50a0d172ef1a814ab46c253f2bf4cbd871b4bf21c7` |

The PASS permits the narrow source claim stated at the start. Any later deployment still needs the root's actual binary receipt, fresh backup, live measurements and subsequent-sweep review required by the active goal. The unmet recall floors, new held-out bar and separate injection/key-collision issues are not waived.
