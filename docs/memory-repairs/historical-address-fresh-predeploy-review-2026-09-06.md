# Independent immediate 02:47:04 predeployment extension

This extends, and does not modify, `REPORT.md` SHA-256 `2994ab25878fdde7ebc6622f4b1a419208fb933e882045a30edf40f13e556a84`. Candidate remains the exact a06cd7b artifact SHA `7938d05258bb3b08461441b7448a8da4a95374664f42a9b240f22c3327de55ef`. No candidate source change, live write, provider call, retry or deployment occurred during this reviewer extension.

Verdict: **bounded immediate predeployment extension PASS**. Both fresh complete restores preserve all active bytes through candidate reads and actual five-suite comparisons. All twenty baseline/candidate CLI benchmark invocations pass and all per-snapshot compared retrieval fields agree. No additional candidate safety defect was proved. The original whole-goal failures and the disclosed newly parked fact-conflict input remain; no whole-goal PASS is granted.

## Fresh complete restores

The lead supplied a later pair of complete backups. I independently hashed each whole file and called direct Badger Load into new private destinations `fresh-shared` and `fresh-laptop`. Full logical raw maps are equal around candidate Open, AllFacts, Entities, index construction and a bounded recall.

| Backup | Bytes | SHA-256 |
|---|---:|---|
| `/tmp/scry-historical-deploy-sep06.INlBVW/memory-20260906T024704Z.badger` | 75,362,914 | `b966754cee3ed9db038bf757571488f50ee12d038a9b95084cc329b78e057b5c` |
| `/Users/jeff/.scry/backups/memory-20260906T024704Z.badger` | 19,445,024 | `9a00beae074ee888dabe2884477f964561edac6ee574c82bdadc47c3219c5a5a` |

Shared: 246,219 active rows; 81,514 facts, 73,651 current and 7,863 historical; 31,074 entities; 9,425 episodes; 23 pending. Logical digest `e88e1d8a5fb5c922ecd028393e2a09ba2bcdb20eecc383f6c6d3ef750c5bf720`. Laptop: all 83,378 logical rows byte-identical to the prior laptop restore, digest `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`; no semantic or raw-row change despite a different complete-backup file hash.

## Full 02:36:47→02:47:04 shared accounting

Every one of the old 81,504 fact payloads is byte-identical, including all 7,863 historical rows. All 52,557 old alias claims, 42 repair markers, 11,142 old attestations, and 22 previously parked complete inputs remain byte-identical. There are no removed, relocated, backdated, invalidated or rewritten facts in this later interval.

| Family | Equal old rows | Added | Changed | Removed |
|---|---:|---:|---:|---:|
| adjacency | 56,223 | 4 | 0 | 0 |
| alias claims | 52,557 | 13 | 0 | 0 |
| alias rejection | 4 | 0 | 0 | 0 |
| attestations | 11,142 | 4 | 0 | 0 |
| cursors | 3,163 | 0 | 0 | 0 |
| entities | 31,063 | 10 | 1 | 0 |
| episodes | 9,424 | 1 | 0 | 0 |
| facts | 81,504 | 10 | 0 | 0 |
| metadata | 4 | 0 | 1 | 0 |
| pending | 22 | 0 | 1 | 1 |
| retirement spellings | 19 | 0 | 0 | 0 |
| retirement targets | 19 | 0 | 0 | 0 |
| value evidence | 1,018 | 9 | 3 | 0 |

The one changed old entity changes LastSeen only; names, aliases, type, description and all repository refs remain equal. All ten new entities resolve through their exact slugs and all thirteen newly listed names/aliases resolve to their owners. All ten new facts have existing endpoints and episode closure and cite the newly committed episode. Three changed value-evidence records add episodes only, retaining earlier identity, spellings and evidence; episode closure passes. Current relation count remains exactly the documented 39, with no noncanonical current row.

Every complete structural defect set is exactly equal to 02:36:47, including 2,854 zero-fact entities, 2,975 no-current entities, 2,441 dangling endpoint occurrences, 504 wrong-owner listed-spelling occurrences, 27 cross-type listed spellings and 93 current self-loops. All main-report structural failures remain; this interval introduces none of those measured defects.

## Queue closure and caveat

The previously pending source-only note, pending payload SHA `8032194651f7028d704eedcf43e58dce36755e8536bcf34ee8668befd94ab6a0`, is now consumed. Its stored episode SHA is `d84c0ad7167375d01402796c6a34849531bf08e473dd2bc554dbd74d9126237e`; full ID/source/source_ref/occurred_at/cwd/cwd_is_repo match the original captured pending input. The new facts cite this new episode with closed references.

One formerly active input becomes parked, increasing the parked total 22→23. Attempts rise 0→1; only attempts/last_error/parked change. All non-retry fields remain equal. Its pending key SHA is `fc70227858b34ee015ad8e26199827c3dcf10f5b120e345dbeb8adbd6893b074`, final payload SHA `9255443114978e969c0adda7587d4070085b358e6dee35659e9a6a68cc271d9c`. The error matches fact-conflict, not alias-claimed or alias-rejected. Its actual error is hashed, never printed.

The error's occupied-address digest `95205323a70edd13e7c2889a5a554e575c537b9b8d65db3e22d1dee7064b5551` has no matching committed fact in the restored final snapshot. There is no committed episode or citing fact for that failed input. This is consistent with a conflict against a transaction-staged assertion, but the original extraction attempt is not replayed and that cause is not independently proven. The closure helper's `occupied_fact_exact_preserved:false` reflects the absent address, not observed loss of an old fact; the complete raw comparison independently proves every old fact remains equal. No queue retry was attempted or approved by this grade.

## Retrieval and limits

The same actual installed baseline and independently rebuilt exact candidate run the five frozen suites on each fresh replica, top 20. Comparison checks hits, mean rank, maximum bytes, complete miss membership/ranks and hashed previews; full raw maps are captured before/after all suites. No benchmark file or expectation changes. Per-question full-response rank/hash parity was already independently established on both preceding replicas in the main grade; this small extension repeats the actual five-suite matrix and complete raw preservation, not the separate 235-question helper matrix or source tests.

| Suite | Shared hits, both binaries | Shared max bytes | Laptop hits, both binaries | Laptop max bytes |
|---|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 | 12,111 | 22/62 | 10,578 |
| heldout-b | 29/66 | 13,359 | 13/66 | 11,034 |
| probes | 7/7 | 9,974 | 5/7 | 9,771 |
| tuning-strict | 45/50 | 11,542 | 28/50 | 10,470 |
| tuning | 47/50 | 11,542 | 30/50 | 10,470 |

All cap counts are zero. Shared heldout-b mean answer rank changes across baseline snapshots from 5.068965517241379 to 5.103448275862069; this small observed ingestion drift occurs equally in baseline and candidate. Accordingly this extension does not assert old/new snapshot rank equality. Original shared recall floors still fail. Both full raw maps remain byte-identical after their ten actual CLI runs.

After the final benchmark, both installed host paths still independently hash to baseline `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`; the supplied candidate remains `7938d05258bb3b08461441b7448a8da4a95374664f42a9b240f22c3327de55ef`. `extension.sh` records exact serialized commands. `EXTENSION-ARTIFACTS.sha256` pins this addendum and its completed JSON/JSONL evidence. The main report and its original manifest remain unchanged.

This extension inherits every main-report limit: no live daemon/startup/postdeployment/sweep verification, no newly held-out fifty-question grade, no remember p95, no semantic status/value adjudication, no temporal-model or historical-recovery certification, and no complete-goal PASS. Private restored databases contain real data and must not be published. The current queue remains live outside these frozen replicas; the report certifies only the stated captured snapshots and exact artifact.
