# Current recall-floor independent disproof, 2026-09-05

Bounded verdict: **FAIL: original recall floors remain 51/62 versus 53/62 and 29/66 versus 34/66.** This report diagnoses the fixed supplied graph. It does not approve a ranking change, semantic repair, deployment, or whole goal. No new held-out questions were authored.

## Exact baseline and preservation

Source archive: exact `d1f0a958608389a385ac9a9f57ec1eb941015a10`, independently extracted into `code/`. Exact benchmark binary: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry`, verified SHA-256 `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.

The supplied immutable backup `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260905T230854Z.badger` independently matches SHA-256 `188e5bd97b24221c61ac86b4e605f6f05dc0d8184d39be8a464ece4230c785ce`. Fresh restore contains 80,710 facts, 72,885 current facts, 30,658 entities, 9,360 episodes, 52,017 alias claims, three alias rejection records and ten pending records.

| Unchanged suite | Hits | Required floor | Maximum bytes | Over cap |
|---|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 | 53/62 | 12,109 | 0 |
| heldout-b | 29/66 | 34/66 | 13,360 | 0 |
| probes | 7/7 | 7/7 | 10,107 | 0 |
| tuning-strict | 45/50 | 44/50 | 11,530 | 0 |
| tuning | 47/50 | 47/50 | 11,530 | 0 |

These are exact deployed CLI runs, default top twenty, each with a freshly constructed index. The archived question files are unedited. `exact-*.json` preserves full outputs. Full sorted raw key/value maps, with values encoded as base64, are in `raw-before.json`, `raw-after.json`, and `raw-final.json`; all are byte-identical under `cmp`. Their length-prefixed raw-stream SHA-256 is `2ff1ea657a217bd3d038da694d1e18bc2ca34ae7b1758006f3fe79dfc52c6d12`. Read-only diagnostics did not mutate any graph, queue, rejection, or metadata record.

## Where the current misses occur

Diagnostic-only instrumentation captures the actual named entities, expanded query, lexical candidates, ordinary scores before diversity, and complete order after diversity, before response clipping. It leaves the production scoring steps unchanged. For all 128 questions in the two failing suites, the diagnostic first twenty hits were checked field-for-field against ordinary `Recall(...,20)` and matched. This avoids the earlier error of treating a payload-capped `--top 200` result as the complete candidate set. Full traces are `heldout-2026-09-03/001.json` through `062.json` and `heldout-b/001.json` through `066.json`.

| Classification of the 48 misses | First suite | Second suite |
|---|---:|---:|
| No stored fact satisfies unchanged expectation, including history | 2 | 0 |
| Current expectation-matching fact exists but no matching fact enters candidates | 0 | 2 |
| Current matching candidate exists, outside final top twenty | 9 | 35 |

These counts classify the existing exact matcher, not a claim that every matching fact fully answers its question. All 48 questions, expectations, original fact keys/payloads, lexical ranks, pre/post-diversity ranks, and named-entity maps are retained in `analysis.json` and `misses.tsv`. `source-evidence.json` attaches every matching current fact's stored source episode, summary, source reference, and chronology. No expectation was weakened or replaced.

The two absent exact expectations are first-suite Q6 (loopback tunnel) and Q8 (Anthropic output pricing). Q6 requires `loopback origins`, `warp-routing`, and `127.0.0.1/32` together in one fact. The current source-supported `cloudflare-workers-vpc conflicts_with cloudflared` fact contains the first phrase and the failure explanation but not the other two; its manual episode `fed34608…` contains all three. Q8 requires both `25.00 USD` and `$25.00` together. Four current Opus price assertions contain `$25.00`, none contains `25.00 USD`. Their source `51c673b1…` is an agent quoting a cached model table, not a verified present-day price. Ranking alone cannot make either current exact expectation match. This is not evidence authorizing synthetic restatements, source mutation, or expectation edits.

The actual candidate omissions are second-suite Q27 and Q54. Q27's current registry-verification fact explicitly says `0180-0199 remains the single unclaimed block`; original lexical rank **4146** exceeds the 4000-candidate cutoff and its endpoints are not in the named-entity map. A second matching current fact has zero lexical score. The original cited Codex byte range independently contains the single-unclaimed-block assertion. Q54's `mechanical-checker status verified` sentence includes the mechanical writing gate's **100 words**, trigram overlap and syntax templates; original lexical rank **5604** exceeds the same cutoff and its source is not named. Its original transcript path is absent locally; the stored fact/episode remain intact. Neither is a payload-cap miss. I did not widen production candidates or infer that widening would place either in the top twenty.

The recent Cell Saviors and staging misses remain ranking cutoffs at **23** each. Other near misses include first-suite nested-folder query at 27, demo MFA at 30, AMD model-loading at 33, memory `max_tokens` at 35; second-suite worker git rules at 22, DataForSEO pricing at 22, room review at 25, design-session model at 27, and old daemon at 31. The Ollama matcher first hits at 22, but the intended listener sentence itself is **29**; a looser port-only matcher must not be mistaken for semantic success.

## Two general defects worth testing

### 1. Expansion is full-weight ranking evidence despite its candidate-only contract

Production `expand` says original words keep their full weight and synonyms only add candidates. In fact, `Recall` calls `ix.Search(expanded, KindFact, …)` and starts every score at that expanded BM25 score. `Search` treats every non-prefix expansion token exactly like a user token. Named-entity injection also uses `ScoreDoc(expanded, …)`. This is a discrepancy between the intended boundary and actual mechanism, not a missing synonym.

Second-suite Q22 asks which git operations workers may not run. `allowed` expands to `budget cap capped limit quota`, with no budget sense in the question. The same source-supported hard-git-rules fact moves original lexical **1 → expanded lexical 19 → final 25**. Another valid worker-discipline statement is the first exact match, final **22**.

Two unrelated trial-budget facts occupy final ranks 2 and 3: “solid-04/solid-03 is starved behind the $20 budget cap and has not been able to run.” Each rises from original BM25 **2.676** to expanded **23.106**, a **20.430** point gain while matching no git restriction. The hard-rules answer stays at **16.879**. Its exact source is `api-routes-operations-hardening decided fleet-git-discipline`, valid `2026-08-18T19:57:41.75Z`, episode `12fbb0fc…`; full source context is in the evidence file.

This also appears across unrelated questions: dunning bad-input answers move original lexical 44/85 to expanded 280/428, Google OAuth tokens 37→317, and assistant-template obligations 116→368. Expansion also helps other misses (Orbic original 4667→82, AMD original 109→30), so deleting the table wholesale is not supported. A bounded test of separating candidate expansion from ranking evidence would test this general contract without adding, removing, or fitting word entries. No such variant was implemented or scored in this review.

### 2. An unconditional canonical-relation prior overwhelms source-supported explanations

When any `whyWords` token fires, every candidate with a `reasonRelations` canonical relation receives **+8**, independently of whether it explains the asked event. Correct explanations extracted as `calls`, `related_to`, or `status` receive zero. This reads a lossy extraction label as relevance evidence about the question.

For second-suite Q32, “Why did the daemon keep behaving like the old code after I rebuilt it?”, the real Cockpit assertion is lexical rank **3**, ordinary/final rank **31**, score **25.023**. It states the old binary kept answering until `launchctl kickstart -k`; canonical relation is `calls`, raw relation `runs`. Its source episode `1e21e21b…` and original local transcript both confirm the launch agent required kickstart. The transcript records the old HTTP 202 response, the kickstart invocation, and the final explanation that daemon stop was ineffective under launchd.

**29 of the 30 facts ahead of it receive the +8 prior.** Rank 2, an unrelated Scry restart interrupting sweep sockets, has lexical **11.367**, meaning **6.967**, final **30.431**, including +8 and named/recency terms. The actual answer has lexical **17.936**, meaning **6.937**, final **25.023**: better text overlap and nearly identical meaning, but no relation bonus. Rank 1 is “The sweep uses the scry daemon binary which was rebuilt after the code changes,” boosted as `depends_on`; it does not explain the Cockpit process behavior.

The fixed +8 is larger than the lexical margin separating these facts. A bounded test of conditioning or constraining this relation-derived evidence is justified. This is not a proposal to rewrite stored relations, relabel answering facts, or sweep fitted weights. Prior history already records reason-weight sweeps; repeating an unprincipled sweep would not test the identified semantic failure. No alternate ranking was measured here, so no gain or regression promise follows.

## Other observations and limits

Diversity demotes Q34's `ai-syncpy related_to claude-code` answer from **32→4155**, and Q53's `publishsocialpostdraft configures failed` answer from **50→3993**. Pair limits ignore attribute values, and near-duplicate tests may demote source-supported details. Neither would hit twenty before diversity, so this report does not promote diversity to a third proposed fix. Q53's source also identifies the retry behavior as a reviewed defect, rather than an endorsed policy; optimizing its broad matcher without reading that source would be misleading.

The current graph includes review/operation artifacts that compete strongly: the Cell Saviors query ranks “The unrelated queue item in the scry queue belongs to the Cell Saviors project” second, and the fleet-review query ranks a read-only task-listing approval rationale first. These facts must not be discarded to improve scores. Naming and ranking need to tolerate legitimate corpus growth. This diagnostic does not certify all such extraction as semantically correct.

The full historical audit and relevant decisions were read, including rejected relevance feedback, entity-name expansion, graph traversal, coverage weighting, vector retrieval, transcript-corpus experiments, and the correction to the old candidate-ceiling claim. None was repeated as a proposed fix. The two attribution reports remain valid bounded evidence; this review goes beyond them by enumerating current misses and tracing their scoring/candidate paths.

Original-source corroboration was sampled, not performed for every matching fact. `original-source-excerpts.json` distinguishes available transcript evidence, a manual episode whose persisted summary is its source, and the missing writing-gate transcript. Full stored provenance for all misses remains available. These reused benchmarks are not fresh held-out grading. Two subsequent live sweeps and the whole ten-clause goal were not assessed.

## Reproduction

Review root: `/tmp/scry-recall-floor-disproof-sep05.IYB5c3`.

1. Build helper: `cd code && go build -o ../audit ./cmd/recall-floor-audit`.
2. Fresh restore only: `./audit restore <new-private-dir> /tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260905T230854Z.badger`.
3. Raw map: `./audit raw <dir>`; inventory: `./audit inventory <dir>`.
4. Exact baseline: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry memory bench --dir <dir> --file code/docs/memory-bench/<suite>.json` for all five suites above.
5. Detailed diagnostics: `./audit details <dir> code/docs/memory-bench/<suite>.json <existing-private-output-dir>`. Run sequentially; Store.Open owns a Badger lock. Then `node analyze.mjs` and `node evidence.mjs` from this review root.

The helper's private diagnostic flag returns the real candidate order before response-size clipping and captures component evidence. Only this private archive has instrumentation; the exact baseline binary remains unchanged. No shared-repository edits, live write/repair, synthetic memory, room/remember call, provider call, deployment, candidate-count change, synonym change, ranking variant, or expectation edit occurred.
