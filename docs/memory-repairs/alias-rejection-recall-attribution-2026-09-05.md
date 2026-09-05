# Independent recall attribution disproof — 2026-09-05

Verdict: the new Cell Saviors recall miss is reproducible with the OLD deployed binary on the ACTUAL immediate PRE-deployment snapshot. The alias-rejection code deployment is not required to cause the miss. The original recall-floor failure remains OPEN; this report is not whole-goal acceptance or authorization for any live repair.

## Exact controlled comparison

Fresh independent restores were made into this private directory with `audit restore`; the helper refuses an existing destination. It uses the production Store.Restore implementation and does not touch the user's default store/backups. Both exact supplied binaries then ran `memory bench --dir <replica> --file docs/memory-bench/heldout-2026-09-03.json` against each identical snapshot, building fresh indexes offline. No provider, hosted embedding, model download, or ranking/expectation changes were used. Production local random-indexing vectors are part of ordinary offline index construction.

| Complete snapshot | Facts | Old 62cf binary | New d1f0 binary | Address rank, expanded ordinary recall |
|---|---:|---:|---:|---:|
| Earlier baseline mini-pre-deploy.badger | 80,338 | 52/62 | 52/62 | 18; score 33.001 |
| Actual immediate pre mini-immediate-pre.badger | 80,410 | 51/62 | 51/62 | 23; score 31.719 |
| Actual immediate post mini-immediate-post.badger | 80,410 | 51/62 | 51/62 | 23; score 31.719 |

The old/new benchmark JSON is identical on each snapshot after excluding measured latency. The miss is the same exact question: “What tailnet address does the Cell Saviors box answer on?” The fixed benchmark expectation remains `all_of: ["100.99.162.30"]`, top N remains 20. Its benchmark rank -1 means absent from the bounded 20-fact result, not absent from storage. For diagnosis only, unmodified `recall.Recall(..., limit=40)` exposes rank 23. Expanded-recall helper was built from the candidate source; `git diff 62cf6e0d2d8db9d324da23114df837848e972c8f d1f0a958 -- internal/memory/recall internal/memory/search internal/memory/embed` is empty.

## Raw preservation and source identity

Every raw key/value was traversed in Badger sorted-key order, hashed with an unsigned big-endian 64-bit key length, key bytes, unsigned big-endian 64-bit value length, then value bytes. Before/after hashes match for every replica. Actual immediate pre and post are themselves completely raw-identical, including queue and repair metadata, not only graph counts:

- Baseline raw SHA-256: `8fdd3352817c8832f27f9740baf0342f7bc581a83c8f9f2df9549ef45f22771c`.
- Immediate pre and post raw SHA-256: `b88cca9d2244ed7ee212e908d8b928388989bf9b6bf853f4a2397ed54432f72f`.
- Immediate pre/post prefixes: adj 55,503; al 51,856; att 10,801; cur 3,120; en 30,520; ep 9,339; fa 80,410; meta 5; pq 11; rs 15; rt 15; ve 509. No `ar` prefix exists.

The single stored answering fact is byte-equivalent in decoded form in baseline, pre, and post, remains current (no invalid_at), and preserves its exact key and provenance:

`fa:cellsaviors:related_to:~1009916230:2026-08-29T01:41:22.587929Z`

“cellsaviors.com is reachable on the tailnet as cellsaviors-new at 100.99.162.30.” Source `cellsaviors`, relation `related_to`, raw relation `reachable_at`, value `100.99.162.30`, confidence 1, valid_from `2026-08-28T21:41:22.587929-04:00`.

Its unchanged manual episode `3b026fe8e5fedc53353245bbf367d5d8c58913c0ba06135f1f005c089ff4e8e2` explicitly states: “cellsaviors.com is a Laravel application on Laravel Forge at 174.138.111.139, reachable on the tailnet as cellsaviors-new at 100.99.162.30. Hermes monitors it read-only over SSH as the hermeswatch user, which has no sudo.” There is no evidence here to change the expected answer.

## What moved the address below the cutoff

The earlier baseline to actual immediate pre contains 7 newly ingested episodes, 73 added fact keys, 1 removed fact key, and 3 modified existing facts (net +72 facts). All additions precede deployment in the actual pre snapshot. The apparent removed key is the older timestamp for the same scry 62cf6e0 deployment fact; a new key carries the same sentence/value with earlier day-level validity and additional provenance. The 3 modified facts concern CadFormats (one status invalidation, two provenance additions). None modifies the Cell Saviors address or its provenance. Full deltas are in analysis.json; semantic approval of unrelated changes is outside this bounded attribution review.

The newly added fact above the address is:

- Rank 18, score 32.932: `jeff -[documents]-> cellsaviors-branding-overlay`: “Jeff asked how to overlay CellSaviors branding on the extracted cell images.” It cites new episode `6c2459f44a4979f3151c5e26181534d1d3aba8a832d56b359957a94b173900f5`, ingested at 2026-09-05 21:52:52.344471 UTC. Source: `/Users/jeff/.claude/projects/-Users-jeff-workspace-cellsaviors-server/f7e03ffc-6634-45a9-86ce-5b5239a59024.jsonl#8859-182704`. I read the original local transcript: line 8 contains Jeff's actual request to pull images of each cell and overlay CellSaviors branding. Thus this is source-supported ordinary project material, not a fabricated benchmark improvement fact.

Four existing facts formerly below the address also now outrank it:

| Existing fact | Earlier rank / score | Immediate pre rank / score |
|---|---|---|
| cellsaviors owns battery-designer | 19 / 32.989 | 19 / 32.883 |
| remote-hosts-spec requires tailnet-forwarder | 21 / 32.659 | 20 / 32.664 |
| searchconsoleservice related_to cell-saviors | 20 / 32.668 | 21 / 32.560 |
| tailnet-forwarder implements mac-mini | 22 / 32.448 | 22 / 32.452 |

Together these five facts explain the observed position change from 18 to 23. The answer's score falls by 1.282 as the corpus changes; ordinary fresh index construction learns vectors from the current facts and episode summaries. This establishes a corpus-dependent ranking change under unchanged retrieval code, without identifying an exclusive numerical contribution for any single new episode.

The seven new episode IDs and complete source refs/summaries are in analysis.json. They include two Cell Saviors image/branding segments and five Scry/CadFormats review segments, all already present before code deployment. No existing episode record changed.

## Proven vs not proven

Proven: old and new exact binaries agree on each same snapshot; both miss before deployment; all immediate pre/post raw records are identical; the answering fact and evidence remain intact; changed corpus reproduces the drop under fresh offline indexes.

Not claimed: that these fresh offline indexes reproduce the daemon's historical incremental index at 21:44 or 21:57 exactly. No copy of that process's in-memory index was supplied. Whether the pre-restart live index still hit this question and a refresh exposed the corpus shift remains an inference. A binary rollback cannot restore the earlier corpus and the old binary freshly indexed on actual pre/post also misses. The original live recall floor remains OPEN and requires its own source-honest resolution.

## Artifact identity and reproduction

- New exact binary `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry`: SHA-256 `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.
- Old exact binary `/Users/jeff/go/bin/scry.pre-d1f0a95-20260905T2156Z`: SHA-256 `821358499706bd9388b63a4368bb5320fc1bac61f12c93df48a158667b19bc14`.
- Baseline backup SHA-256: `c063d83125a81f096e319c94286958da8f29a388e1e29b73460180a37be4d397`.
- Actual pre backup SHA-256: `1a99896030d3fc0ac0da458923b7baac1af72cbf7542416921f0f4256a20befb`.
- Actual post backup SHA-256: `cc23d752d9804d92256176a8a3b25d01330e14da9c985a5f2f364ceb4272206a`.

All six exact benchmark results: `{baseline,pre,post}-{old,new}-heldout.json`. Full ranked results and stored matches: `*-details.json`. Complete decoded inventories: `*-inventory.json`. Raw checks: `*-raw-{before,after}.json`. Complete parsed fact/episode deltas: `analysis.json`. Reproducible read-only diagnostic helper source: `code/cmd/recall-attribution/main.go`, executable `audit`; analysis script: `analyze.mjs`. Restored fixtures remain in baseline/, pre/, post/ for independent reproduction.

No shared repository edits, live repair writes, queue changes, remote shell, hosted embeddings, or provider calls were performed. Initial memory orientation and recall were read-only context discovery; all attribution experiments used the private restored fixtures.
