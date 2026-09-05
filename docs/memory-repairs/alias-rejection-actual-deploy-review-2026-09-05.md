# Independent actual code-only deployment disproof — 2026-09-05

Verdict: bounded PASS for the actual deployment's artifact/process identity, complete logical-state preservation, immutable backup restoration, and pending-payload preservation. No deployment-induced logical mutation exists between the actual 21:57:35 UTC and 21:58:42 UTC snapshots on either machine. All 242,104 Mini keys and all 83,378 laptop keys have identical values before and after. This does not certify semantic alias ownership, authorize a live drop/backfill/retry, or complete the wider memory-quality goal. Five live postdeployment benchmark suites and later ingestion health are the root's separate gates; this reviewer did not independently run them.

## Exact artifact and running processes

Fresh independent `git archive` of commit `d1f0a958608389a385ac9a9f57ec1eb941015a10` into this report's private `code` directory. An independent build using `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=d1f0a95'` matches the proposed artifact byte-for-byte. SHA-256: `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.

Read-only local and independent `ssh jclaw@mini` checks confirm this same hash at `/Users/jeff/go/bin/scry` and `/Users/jclaw/.local/bin/scry`. Both retained siblings `scry.pre-d1f0a95-20260905T2156Z` match prior SHA-256 `821358499706bd9388b63a4368bb5320fc1bac61f12c93df48a158667b19bc14`. Both launchd jobs are running with last exit code zero: laptop `gui/501/com.jhoot.scryd`, PID 40814; Mini `gui/501/ai.jermes.scryd`, PID 15220. Process start times are 2026-09-05 17:58:20 EDT / 21:58:20 UTC. Both process argument vectors name their installed `scry start --foreground`. Launchd's configured program is `/bin/zsh`, a wrapper; it is the process argument vector that identifies the installed scry path.

## Complete restored-state proof

For each of FIVE independently hashed immutable inputs, the harness created a NEW private Badger directory and loaded the complete backup directly, bypassing candidate schema handling. It iterated every key and value without any prefix filter. Candidate Store.Open on that loaded store preserved the complete map exactly. It separately opened another new private directory, invoked candidate Store.Restore, compared every key/value to the raw-loaded source, closed/reopened it, and compared again. Every check passed. Whole-map equality distinguishes absent keys from present empty values. State hashes sort keys and hash each key and value with explicit uint64 big-endian byte-length delimiters.

`go test ./internal/memory/store -run '^TestActualDeploymentIndependent$' -count=1 -v` PASS, test duration 6.07 seconds, package 6.319 seconds. The complete harness is `code/internal/memory/store/actual_deploy_independent_test.go`; per-source inventories are summarized in the five `*-summary.json` files. Complete unfiltered actual deployment delta files `mini-pre-to-post-delta.json` and `laptop-pre-to-post-delta.json` are both `[]`.

| Input | Backup bytes | Backup SHA-256 |
|---|---:|---|
| Mini reviewed 21:53:17 | 78,208,870 | `86fa9d43e579a421ce152a1d14f89ed6443ec4fff20fb9b23abed9a2f9d4a5a4` |
| Mini actual pre 21:57:35 | 78,295,450 | `1a99896030d3fc0ac0da458923b7baac1af72cbf7542416921f0f4256a20befb` |
| Mini actual post 21:58:42 | 73,246,841 | `cc23d752d9804d92256176a8a3b25d01330e14da9c985a5f2f364ceb4272206a` |
| Laptop actual pre 21:57:35 | 19,444,992 | `7326d676acac038179d1243eb9f2f6c8909dbd45ab77b7682c5ec9dc2bf928dd` |
| Laptop actual post 21:58:42 | 19,445,016 | `b122b82e11e15c2d08025a5d3c6176783697c8bce9e5172120765fcafef95988` |

Mini inputs are the three corresponding files in `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3`. Laptop inputs are `/Users/jeff/.scry/backups/memory-20260905T215735Z.badger` and `memory-20260905T215842Z.badger`. Independent SSH additionally verified both actual Mini hashes against originals at `/Users/jclaw/.scry/backups/memory-20260905T215735Z.badger` and `memory-20260905T215842Z.badger`.

Mini actual pre AND post complete logical-state SHA-256: `b88cca9d2244ed7ee212e908d8b928388989bf9b6bf853f4a2397ed54432f72f`. Every prefix has identical counts and values: 80,410 facts (72,603 current, 7,807 historical), 30,520 entities, 9,339 episodes, 51,856 alias claims, 55,503 adjacency records, 10,801 attestations, 3,120 cursors, 509 value-evidence records, 5 metadata keys, 11 pending records, 15 retired spellings and 15 retired slugs. Schema is exactly `1`; rejection (`ar:`) count is zero. No startup wipe, synthesized rejection, altered historical fact, lost episode, claim change, cursor change, or metadata mutation occurred in these snapshots.

Laptop actual pre AND post complete logical-state SHA-256: `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`. Identical 21,004 facts (18,495 current, 2,509 historical), 14,200 entities, 2,689 episodes, 23,159 claims, 21,004 adjacency records, 1,321 cursors, and the schema metadata key. No pending records. This supports its dormant local-memory role during the measured interval; it does not predict future activity.

The smaller physical Mini post-backup proves neither data loss nor a particular compaction mechanism. The complete restored logical maps establish zero logical change despite the 5,048,609-byte physical difference.

## Queue eligibility and all parked payloads

The Mini's actual pre snapshot is 2 ready / 1 backoff / 8 parked at 21:57:35 UTC. Actual post is 3 ready / 0 backoff / 8 parked at 21:58:42 UTC. The raw pending records are entirely unchanged. Record `1e5b18139366b0e7cec72ffaa05cf90613da2cc4ce3e4b1318c8ae2f8bb07c5b` has an unchanged `next_attempt` of 21:57:58.301462 UTC, which falls between those observations. Reclassifying the PRE snapshot at the POST timestamp gives exactly 3/0/8 (`mini-pre-at-post-time.json`). This is time eligibility, not a persisted retry, completion, deletion, arrival, or mutation.

All original eight parked records preserve their complete serialized payloads, including text, source and source_ref, cwd, timestamps, attempts, last_error, parked flag and all other fields. Per-payload hashes are saved in `mini-original-parked-payload-sha256.json`; the unchanged pending records are saved privately in `mini-pending-records.json`. The full map comparison covers every pending record, including the three nonparked ones.

## Reviewed 21:53 source to actual predeployment drift

The independently restored reviewed snapshot matches the previous review's complete logical hash `8d7c3ce653ee98257d4c72e1538b5e053f13e07b9738b048b9fd0d6f5b3de178` and 241,868-key count. The complete delta through the actual pre snapshot is retained as `mini-reviewed-to-pre-delta.json`, with explicit before/after key presence and base64-encoded exact value bytes. There are 260 changed keys: 45 adjacency additions; 42 claim additions; 16 attestation additions; 34 new and 4 modified entities; 5 episode additions; 62 fact additions and 1 removed old fact key; 1 metadata update; 5 pending removals and 3 pending updates; 38 new and 4 modified value-evidence records. No cursor, rejection or retirement-marker changes occurred.

The five removed pending IDs exactly match the five added completed episode IDs: `0f5f38fb...`, `173bdd22...`, `3f36b6b7...`, `b3e21328...`, `c5cc7466...`. They concern four Scry review/repair transcripts and one CellSaviors rendering transcript. The two preexisting six-record parked additions (`479b7049...` and `eca00e8b...`) each record attempt 1 with the existing gemini-api versus gemini claim conflict. The third pending update is the timeout/backoff on `1e5b1813...`, attempt 1 with `extract: chain stopped before deepseek-v4-flash: context deadline exceeded`. No new pending item was added in this interval. All six originally parked records remain unchanged; the two new parked records predate candidate deployment. `meta:last_extract_ok_at` advanced from 21:53:02.081664 UTC to 21:56:56.591812 UTC, before deployment.

The one removed fact key is `fa:scry:deployed_on:~62cf6e0:1788641400000000000`. Its full assertion survives under `fa:scry:deployed_on:~62cf6e0:1788566400000000000`, preserving the same sentence, src/relation/value, raw relation, confidence 1, and original provenance episode. The replacement adds the newly completed `0f5f38fb...` episode and moves valid_from from 20:50 UTC to midnight. This is the existing mergeFact earlier-valid-from rekey behavior in `internal/memory/resolve/resolve.go`; that file is unchanged between old 62cf6e0 and d1f0a958. It is a fact-key relocation with provenance preservation, not an unexplained erased assertion. The 62 newly present facts include this replacement and three newly present historical facts, yielding net +61 facts and +3 historical facts.

The four modified entities are hermes and scry (last_seen only), nanobanana (last_seen plus the CellSaviors repo reference), and migration0160 (last_seen plus alias `migration0160 identity`). New records include `envoyer-identity`, its two alias claims, and an independent review tests edge. These are source-driven changes under the OLD binary, not candidate-created cleanup; no semantic endorsement of the migration0160 alias or new Envoyer identity follows from this deployment pass. No changed key names ChildScribe, office-dashboard or driver-core-worktree; existing `en:envoyer` is unchanged. None of this drift invalidates the exact actual-pre/actual-post identity proof.

## Scope and remaining gates

No live store writes, provider calls, retries, semantic alias operations, backup rollback, deployment, or shared repository edits were performed by this reviewer. Only new private temporary stores/artifacts were written. The shared checkout still has its preexisting untracked `docs/MEMORY_WORKFLOW_ASSESSMENT_2026-09-04.md`.

This pass establishes the actual deployment state through the immediate post snapshot and independent process/hash observation. Later normal ingestion and the root's five live benchmark suites are separate evidence. There is no independent benchmark or latency claim in this report. Zero `ar:` markers in these snapshots means the deployment itself did not introduce the old-writer/marker compatibility hazard; any future first live rejection/backfill must obey the previously reviewed marker-aware rollback constraint and its semantic approval gate. Existing prior alias cleanup remains unprotected by rejection markers until a separately reviewed backfill is applied.
