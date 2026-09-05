# Independent recall attribution: no direct binary regression reproduced

Bounded verdict: **PASS for no direct recall regression caused by the `393eeec` binary on a fixed graph.** The observed additional heldout-b miss is real and reproducible, but both exact production binaries produce it on the later snapshot. The recall acceptance floor remains failed: this attribution does not turn 29/66 into a passing goal result.

## Exact-binary, same-snapshot comparison

The retained old binary and deployed new binary were copied into this review's temporary directory and independently SHA-256 verified:

- Old, `/Users/jeff/go/bin/scry.pre-393eeec-20260905T1939Z`: `31f185d70e1439a315a8ea12eaadf4f12d75e77d558edfa0852c890531705aff`.
- New, `/Users/jeff/go/bin/scry`: `acfb78186402aec9eef87e71e0b81f6edeac1b9ea46641efd417aaf4368f0bd0`.

Each source was independently restored to a new review-only directory. The old and new binaries then ran sequentially on the **same** restored store using:

`<exact-binary> memory bench --dir <independent-replica> --file /Users/jeff/workspace/context-stack/scry/docs/memory-bench/heldout-b.json`

The questions file remained unchanged, SHA-256 `ad92760e4314403a86108442b553e942f6c7b7a1ccdcb78e19f133653efae815`.

| Snapshot | Facts | Current facts | Old hits | New hits | Maximum payload, either binary |
| --- | ---: | ---: | ---: | ---: | ---: |
| ChildScribe post-alias, 19:35:30 UTC | 79,926 | 72,164 | 30/66 | 30/66 | 13,359 bytes |
| Predeployment backup | 79,953 | 72,191 | 30/66 | 30/66 | 13,369 bytes |
| Postdeployment, 19:45:12 UTC | 80,004 | 72,233 | 29/66 | 29/66 | 13,368 bytes |

On each snapshot, the entire old/new benchmark JSON is identical after removing only `mean_latency_ms`: not merely the hit count, but miss identities, reported top facts, answer-rank summary, payload measurements, and cap status agree. All six commands exited 0. Every response stayed below 24 KB.

For each replica, logical fingerprints of **all facts, entities, episodes, and alias-index claims** were identical before testing, after the old binary, and after the new binary. The benchmark calls did not rewrite the graph. This is a logical graph-preservation claim, not a claim that opening Badger leaves every database file byte unchanged.

## Exact lost question and preserved answer

There are no hit changes between the first two snapshots. Exactly one question is newly lost on the third, with no compensating gained question:

> What extra step does staging need after the server-rendered bundle changes?

Its unchanged expectation requires all three strings `daemon-939114`, `supervisorctl`, and `13714` in one answering fact.

The answering fact is still current and **byte-equivalent at JSON serialization in all three snapshots**:

- `childscribe -[deployed_on]-> forge`.
- Sentence: `ChildScribe deployed on Forge; staging daemon-939114 port 13714 requires supervisorctl restart after SSR bundle change.`
- Valid-from `2026-07-21T00:00:00Z`, confidence 1.
- Episode `a7d0b7445ace7e811f89685597c85a43577264cb03ea333072d9467ca67fa11b`.

Its rank is 20 on both earlier graphs, score 23.651 and 23.654 respectively. On the later graph it is rank **21**, score 23.641, as confirmed with a separate read-only 40-fact diagnostic recall. The benchmark's top-20 cutoff explains the miss; no answer was discarded or invalidated.

## Concrete new competitor

The later graph introduces this current fact at rank **14**, score **26.412**:

- `deploy-staging -[deployed_on]-> smoothhauling-staging`, raw relation `deployed_by`.
- Sentence: `Staging deploys run through npm run deploy:staging from the branch tree (build, bundle check, remote migrations, Worker upload, verification).`
- Valid-from `2026-09-05T19:11:10.902Z`, confidence 0.9.
- Episode `05e4cfc366d8e6e200f650432292585062fb8a7dcc173cc361c6eaec6aa907b0`, a Claude session ingested at **19:43:27.555213 UTC**.

The predeployment → later graph delta is 51 added facts, 5 changed facts, **zero removed fact keys**, 22 added/updated entity records, and 4 new episode records. The five changed facts comprise three unrelated status invalidations and two Smooth Hauling provenance additions. The ChildScribe answer is not among them.

Added-fact distribution by new episode:

- 16 facts from the alias-batch audit note, ingested 19:42:18.736074 UTC.
- 8 facts from the identity/fallback decision note, ingested 19:43:22.806572 UTC.
- 3 facts from the Smooth Hauling session above, ingested 19:43:27.555213 UTC.
- 24 facts from the StateLicenseLookup/DBA session, ingested 19:44:33.573365 UTC.

The first three sum to 27 facts, taking the predeployment 79,953 to the reported restart-index count of 79,980. This is consistent with the new staging competitor being present in that index. It is an inference from snapshot contents and ingestion timestamps, not an independently captured image of the daemon's exact in-memory index at the earlier benchmark instant.

## Full rebuild versus incremental vector state

A separate diagnostic used the unchanged candidate search/recall implementation, with all graph stores left read-only. It built a search index from the predeployment snapshot, applied the observed post-snapshot document changes **only to that in-memory index**, and evaluated the staging question against the later store. It then refreshed the deterministic local vectors and compared against an entirely fresh index of the later graph.

| Index state | Answer rank | Answer score | New competitor |
| --- | ---: | ---: | --- |
| Fresh predeployment index | 20 | 23.654 | Absent from graph |
| Post graph lexical updates, predeployment vectors retained | 20 | 23.642 | Outside the returned diagnostic window |
| Same updated index after vector refresh | 21 | 23.641 | Rank 14, score 26.412 |
| Fresh full index of post graph | 21 | 23.641 | Rank 14, score 26.412 |

This reproduces both sides of the hit/miss boundary using graph growth and vector-refresh timing, without changing ranking code. The daemon's existing behavior explains why restart timing matters: `internal/daemon/memory_queue.go:79` builds vectors when the index starts; new facts enter lexical search immediately but wait for the next vector refresh, scheduled every 30 minutes at line 121. `OfflineRecaller` always builds a fresh full index (`internal/memory/recall/bench.go:218`).

The incremental simulation is a controlled model of the mechanism, not an assertion that its event ordering and vector state match the production daemon exactly. The missing exact 79,980-fact backup or in-memory index dump prevents a byte-exact reconstruction of that earlier runtime state. The actual 80,004-fact post snapshot nevertheless reproduces the same lost question with **both** production binaries.

## Code attribution

`git diff 53fafa9..393eeec` has no edits in recall, search, local embedding, benchmark CLI, or daemon index construction. Production changes are the relation guard, distinct fallback-evidence preservation, and deterministic conflict parking; the store change adds an error sentinel.

`resolve.Map` call sites are normal Apply, supersedes resolution, and the explicitly invoked legacy relation-migration path. Recall and search do not call it. The new competitor uses `deployed_by`, which maps through unchanged passive handling to `deployed_on` with a flip; it never reaches the new `same_as` guard or fallback-only evidence branch. Thus its presence is not evidence that the new guard changed that relation's meaning.

The controlled six-run result rules out a direct binary-caused recall difference on these graphs. The concrete rank displacement and vector-refresh simulation support ingestion plus index rebuilding as the cause of the observed additional miss. No rollback recommendation follows from this evidence. The original recall floors still need to be satisfied through separately authorized work; no ranking changes, benchmark weakening, or historical repair was attempted here.

## Sources and reproducibility

Review root: `/tmp/scry-recall-attribution-sep05.zvbXk0`.

Verified backup hashes:

- `/tmp/scry-childscribe-final-sep05.qcoHQ3/memory-20260905T193530Z.badger`: `71c339f10f5a193d4902cb4750f29b985cf8821e2f915565a9dfdadb7601cf3e`.
- `/tmp/scry-fallback-deploy-sep05.5OX7ef/mini-before-deploy.badger`: `88fd5b78303115050d13819701957a871a1fa7a4ab761418c3c42084b43e0abe`.
- Mini `/Users/jclaw/.scry/backups/memory-20260905T194512Z.badger`, downloaded into this review root as `post194512.badger`: `7802b02337a1075a4f03513f0a5a504b119bb5f054cdd07a50e220a07b0954d8`, independently matched against remote SHA-256.

Artifacts:

- `{child193530,predeploy,post194512}-{old,new}-bench.json`: six exact-production-binary benchmark outputs.
- `{child193530,predeploy,post194512}-{before,after-old,after-new,final}.json`: full logical graph fingerprints.
- `{child193530,predeploy,post194512}-details.json`: all 66 question results and all original stored facts matching each expectation.
- `pre-post-graph-index-comparison.json`: complete added/changed fact and entity/episode evidence plus four index-state results; SHA-256 `e32e12a5f6b05b0d515a8965c01f83d7b1de77c22d2883695b4c5bc5f181b4a7`.
- `code/`: exact `393eeec` archive plus the review-only helper under `cmd/independent-recall-audit/main.go`; helper performs fresh restores, logical hashing, read-only diagnostic recall, and in-memory index comparisons.

No live writes, deployment, historical mutations, ranking changes, new questions/expectations, external model calls, remember calls, or shared-repository edits occurred. SSH/SCP were used only to read and copy the supplied immutable backup. Earlier snapshots and other reviewers' replicas were not modified.
