# Actual three-alias repair — independent post-apply grade

**Bounded PASS** for the actual 2026-09-05 22:38:02 UTC three-alias repair, using exact manifest SHA-256 `4ac2ac02f2d855de6e9afa6233f97b9bb1a37e8241b3387bb8dfd13d32a6b652` and candidate commit `d1f0a958608389a385ac9a9f57ec1eb941015a10`. No violation was found of the seven-key prediction, facts-never-discarded requirement, owner-specific negative decisions, actual backup restoration, repeat refusal, or bounded retrieval preservation. This is not whole-goal acceptance: the 53/62 and 34/66 recall floors remain unmet, and global graph defects remain.

The authoritative actual interval is the complete atomic pre-apply backup at 22:38:02 and immediate post-apply snapshot at 22:38:03. A later 22:42:38 snapshot reportedly includes an external writer's new manual episode, 16 new facts, and seven entities. I did not independently grade that later snapshot; neither its additions nor later live state are attributed to the repair or covered by the exact-seven-key statement.

## Isolation and pinned inputs

All reviewer source and replicas are under `/tmp/scry-child-three-actual-independent.677wc2`. I independently extracted `git archive d1f0a958608389a385ac9a9f57ec1eb941015a10`; source archive SHA-256 is `9a0a6c7c4321c2e405f4c53be3e600d30287210986911171369e700b7a5dd516`. Reviewer tests were then added in the private archive. The resulting helper-bearing tree is not described as pristine. No shared repository write, live repair, provider request, queue retry, installation, deployment, or source correction was performed by this grader. The user's untracked workflow assessment was not edited.

I read the complete `child-three-fresh-gate-2026-09-05.md`, `child-three-fresh-extension-2026-09-05.md`, immediate `FRESHNESS-223550.md`, and exact manifest. Semantic source adjudication is inherited from those explicit prior independent gates; this grader does not claim another manual reading of all original transcript spans. Its applicability to the actual apply is independently established by complete raw equality of the actual atomic backup and the latest reviewed immediate snapshot. The prior Scribe root-tsconfig exact-episode reconstruction limitation and all unresolved misattached facts remain disclosed and unchanged.

| Input | Bytes / SHA-256 |
|---|---|
| Actual atomic backup `memory-20260905T223802Z.badger` | 73,526,505 / `2ec037a08d773a6c83ca6c6d1688953c94908c63c73bf463b79430254b0f2f94` |
| Actual immediate post `memory-20260905T223803Z.badger` | 73,529,181 / `aaadbdb7c65ca869385690476440d84895cb4d2afc1c93310291b2d28e012fd6` |
| Reviewed immediate `mini-child-immediate-223550.badger` | SHA `89bfa426f1ae6955fe7cb04fe9bb0fa8472e5e7f089b8cd5e46386fa524eca67` |
| Candidate `scry` | SHA `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553` |

All four supplied files above are under `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3`. Their hashes were independently computed. The actual backup path reported by the live command is `/Users/jclaw/.scry/backups/memory-20260905T223802Z.badger`; root's actual receipt records `dropped:3`, `refused:0`, `applied:true`. File transfer provenance and live apply command execution are evidenced by root's receipt; this grader independently verifies the supplied complete snapshots and separately rechecked installed artifacts/process paths.

Read-only local and `ssh jclaw@mini` checks independently reproduced the candidate hash at `/Users/jeff/go/bin/scry` and `/Users/jclaw/.local/bin/scry`. The two retained `scry.pre-d1f0a95-20260905T2156Z` siblings have SHA `821358499706bd9388b63a4368bb5320fc1bac61f12c93df48a158667b19bc14`. PIDs 40814 and 15220 both still report their installed `scry start --foreground` command and start time September 5 at 17:58:20 EDT. This is artifact/process-path evidence, not a claim of hashing live executable memory.

## Complete actual raw comparison and restoration

Each supplied backup was independently loaded directly into a fresh Badger database before candidate Store.Open. Every key and value was enumerated without any prefix restriction. Candidate Open preserved both direct-loaded actual snapshots exactly. The actual atomic backup equals the complete 22:35:50 reviewed raw state: no drift whatsoever, including queue payloads, cursors, sweep host/report bytes, and source metadata.

I independently reconstructed every Expected field from the actual pre-state: plan `274e1e580ec0ac70464546990f4653f9956432071ffacde89d5d0487125f543f`; Child entity `b7fa1df19b48d1713368d546bda53d32ea6e748292092936fc10b55f26026c59`; complete touching-fact fingerprint `5d879da1900f8e146b5e98858efa44e77445c0ad21b5b21b20b130badb1913fa`; claims, present bits, complete outside listings, and rejection closure. Every field equals the exact reviewed manifest. No other entity lists any of the three exact normalized spellings.

The independently predicted complete map equals the actual immediate post map. Exactly seven keys differ:

1. `en:childscribe-laravel`: remove only the literal `envoyer`, `office dashboard`, and `driver-core worktree` entries, 43 aliases to 40. All other fields and remaining alias order survive exactly.
2. Delete `al:envoyer`, `al:office-dashboard`, and `al:driver-core-worktree`.
3. Add `ar:childscribe-laravel:envoyer`, `ar:childscribe-laravel:office-dashboard`, and `ar:childscribe-laravel:driver-core-worktree`, each with the exact manifest entity, literal alias, reason, and plan.

Both raw maps contain 242,732 keys. Every other original raw byte survives. Prefix counts before/after: facts 80,586/80,586; entities 30,604/30,604; source episodes 9,353/9,353; adjacency 55,606/55,606; positive attestations 10,850/10,850; cursors 3,124/3,124; pending rows 10/10; metadata 5/5; retirement records 15/15 and retired-slug records 15/15; value evidence 607/607. Alias claims fall 51,957 to 51,954 while rejection keys rise zero to three. This includes all fact history and provenance; no selected-prefix omission is used to assert preservation.

`meta:last_sweep_at` remains exactly `2026-09-05T22:34:14.354989Z`. `meta:last_sweep_report` retains host `Mac.attlocal.net`, 94 scanned files, zero ingested/episodes/errors, and `2026-09-05T18:34:14.354989-04:00`, including its exact serialized omission of per-source maps. Queue classification at the actual snapshot time is independently 0 ready / 0 backoff / 10 parked.

The actual live atomic backup was also restored through candidate production Restore into another new directory, compared against every original raw byte, closed, reopened, and compared again. Both restoration and reopen passed. The deployed candidate's existing atomicity tests also passed: preservation/backup; entity, fact, claim-only, outside-listing, plan, and missing-Expected drift; backup write/sync/close failures; concurrent writer exclusion across backup. These fixture tests support the code-path safety properties; the snapshot proof establishes the actual observed outcome.

Complete raw closures (SHA of JSON map from every key to its value SHA-256): before `775d230f16af8ef8572377546374b5d58b7b763df69add3a8114e4ce49837801`; independently predicted and actual post `0d8397603cecfdf16796cb372ef900627543decc848ace6b7bf6b911b12ff854`. These are not length-framed raw-stream hashes.

## Retrieval, negative decisions, and no-op proof

Literal, normalized-hyphen, case/underscore, and padded/repeated-space variants of all three aliases no longer resolve to Child (indeed no owner is returned in the actual post snapshot). Each corresponding owner-specific rejection remains durable. Canonical Child name/slug still resolves. All 2,106 current/history Child-touching facts survive exactly: 1,581 current and 525 historical, with unchanged fingerprint `5d879da1900f8e146b5e98858efa44e77445c0ad21b5b21b20b130badb1913fa`. All 80,586 decoded facts were independently compared too. Root's separate live receipt confirms canonical full-output equality and three exact CLI lookup failures; this grader independently reproduced snapshot lookup behavior.

Direct and atomic entity reintroduction, stale complete Child entity writes, explicit ClaimAlias, and rehome attempts refuse without a single raw mutation. The exact second manifest preview reports not ready and not applied. A private actual-post clone's second BackupAndRepairAliases writes its private backup then refuses application; every database key remains unchanged. The live receipt independently records root's second CLI dry run as dropped zero/refused three/ready false. No live second apply was attempted by this grader.

A separate direct restore of the actual post snapshot tested AdmitAlias for literal and uppercase/underscore variants from two distinct synthetic episode IDs. All refuse with no raw write. Both actual merge directions (Child survivor or loser), supplied fresh Expected and proposed metadata, refuse specifically on marker inheritance; complete raw equality after those probes passed.

That disposable clone then ran two real normal resolver Apply calls proposing all three normalized aliases. Exactly two synthetic episode keys were added and Child's last_seen field changed to the second supplied timestamp; every original fact and every other raw byte survived. No alias regrew, and all three markers remained. A final synthetic distinct Envoyer tool remained technically representable, adding only `en:envoyer` and `al:envoyer` while preserving Child's rejection. This is a private representability test, not an actual Envoyer identity decision or rehome.

## Uncapped inventories

Complete pre/post inventories, including every row, are retained. Structural lists are exactly equal: 2,441 dangling fact endpoints; zero missing source episodes, missing adjacency, or extra adjacency; 3,867 missing listing claims; 505 listing/claim owner mismatches; zero dangling claim owners; 28 claims not listed by their owner; 462 multi-owner normalized spellings; 2,854 entities touched by no current or historical fact; 2,972 without a current fact; 1,096 total self-loops, including 90 current; and 27 exact-normalized multi-type spellings. Zero-fact inventories are structural counts, not independent semantic declarations that each entity should be retired.

The broader production hygiene folding definition yields **484 complete cross-type collision pairs**, unchanged. The independent uncapped pair construction equals candidate CrossTypeCollisionCount and persists every pair, rather than the 40-row production sample.

Every literal alias was classified: 21,981 before, 21,978 after. Never-alias count remains 97; leak count zero; cross-type-claimed count one; revalidation-removal count 98. Named-other heuristic rows fall 6,528 to 6,526 solely because selected Child `driver-core worktree` and `office dashboard` alias rows disappear. Those heuristic suggestions do not authorize package/worktree identity equivalence or an office rehome. Every other classification row is unchanged. The initial grader assertion incorrectly required even these removed alias classifications to remain; it was corrected to require exactly those two selected removals, and the final test passed. This was a grader expectation defect, not a repair defect.

## Actual before/post benchmark controls

The pinned candidate binary independently ran the five unchanged benchmark files against each complete actual snapshot with `scry memory bench --dir <private-actual-pre-or-post> --file docs/memory-bench/<suite>.json`. No daemon or provider was used. Full benchmark outputs and question-file hashes are retained. Scores and the complete missed-question sets are identical before and after:

| Suite | Before → after hits | Before → after maximum payload bytes |
|---|---|---|
| heldout-2026-09-03 | 51/62 → 51/62 | 12,106 → 12,106 |
| heldout-b | 29/66 → 29/66 | 13,360 → 13,361 |
| probes | 7/7 → 7/7 | 10,095 → 10,096 |
| tuning-strict | 45/50 → 45/50 | 11,532 → 11,543 |
| tuning | 47/50 → 47/50 | 11,532 → 11,543 |

All ten runs have zero over-cap results. These reproduce root's live hits, missed-question details, mean answer ranks, and maximum payloads. Live/offline latency naturally differs. Two post-suite mean payload values differ slightly: heldout-b 10,634 live versus 10,633 offline, probes 9,735 live versus 9,738 offline. I do not call the full JSON outputs byte-identical or explain these small mean differences without per-query attribution. There is no score, missed-question, rank, or cap regression. Final direct raw checks after the benchmark and full-classification reads confirm all 242,732 original per-key hashes in each actual replica remain unchanged.

## Reproduction and evidence

Run `node /tmp/scry-child-three-actual-independent.677wc2/reproduce.mjs` to create a fresh sibling directory from the pinned archive, copy only reviewer helpers with a mechanical private-root substitution, and execute the tests and ten offline suite calls. The runner preserves all existing evidence. This convenience runner was written after the individual commands had passed; the underlying individual commands, not a second complete runner invocation, are the executed evidence for this review. It depends on the pinned backup/binary files and exact shared manifest remaining available. The main test is intentionally single-run per evidence directory because its refused-attempt backup uses exclusive creation.

Executed tests: `TestActualLiveIndependent` PASS (7.353 s package); `TestActualFullInventory` PASS (7.172 s final package); `TestActualFinalNoWrite` PASS (1.791 s package); `TestActualNormalAdmission` PASS (4.710 s package). Atomic backup fixture selection passed in 0.690 s package. `node bench-actual.mjs` passed all ten runs and its comparison assertions.

Evidence root is `evidence/`; complete raw before/predicted/actual hash maps, exact changed values, prefix counts, complete defect/classification inventories, Expected, Child facts, lookup observations, refusal previews, private backup, synthetic admission/merge/Apply evidence, and full benchmark results are retained.

Selected exact SHA-256 values:

- `evidence/report.json`: `87768c57b671d333fe24c13ee74555e433a6414b3ffd9f16d7959caa21c63be0`.
- `evidence/actual-raw-delta.json`: `22e7944931b7ed9ccdb89da7ea7ca8cb4f5d9ed1bd44826827151809f8ecd353`.
- `evidence/full-classifications-post.json`: `59911bce2148e70c0dfc377f392325f9d1b2d3fa57fb3120d36d669e21be3dd9`.
- `evidence/bench-comparison.json`: `3e89b38f9f53c96040a134fccb52f49f9b867932541189c00d12b38bdd1b7a87`.
- `evidence/actual-two-normal-apply.json`: `887e7cba40f0964f4949f0048e4bb91c34b258b1ec8cd04c0d7704b1f51da1ec`.
- `evidence/actual-distinct-representability.json`: `48049a8dda61ca68398e7fd52257aee659185f9294d7b3809f4a46f6e9eb8da3`.
- Main actual-raw test source: `0688d4b964c7165c9d018a6c83bdaddcf10e5fdb26f54bfcffb272b0159f1028`.
- Full classification test source: `803d9c0660cd713797a0facb4128671bcda0135966631f37660ee12fabec4c9a`.
- Normal admission test source: `0ab78bf63effc73cda652a6fbfa3c19a83fae038885a6e468266f716fa3a46a8`.
- Benchmark runner source: `d9d5473b766e2a10b26ead5fe18dcbdc217ce4feb3420a4a17e2bf093f610fa6`.

## Rollback and acceptance boundary

These are the first three live rejection records. **Old-binary-only downgrade is now unsafe**: the retained marker-unaware binary can ignore the negative decisions and regrow aliases. Continue with marker-aware code, or separately review a complete restore that reconciles every intervening fact and write. Possession of the actual pre-apply backup proves recoverability of that time, not authority to discard later data.

This PASS covers only Child/`envoyer`, Child/`office dashboard`, and Child/`driver-core worktree` in the exact reviewed batch and actual immediate interval. It grants no approval for prior 49/33-drop backfill, other alias/hollow applies, source corrections, positive rehomes, merges, queue retries, or whole-goal completion. Child's polluted description/repository references and preserved misattached current/history facts remain unresolved. The graph is not declared clean; the original recall floors are not relabeled as passing.
