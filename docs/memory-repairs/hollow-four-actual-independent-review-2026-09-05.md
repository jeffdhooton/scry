# Independent actual-live four-hollow retirement review — 2026-09-05

**PASS for the exact four-record actual retirement represented by the automatic 22:53:02 backup and 22:53:12 post snapshot on d1f0a958608389a385ac9a9f57ec1eb941015a10.** The entire actual post-state equals the independently predicted 16-key change. Every other raw key/value survives unchanged. Independent live reads confirm the four records are absent and another preview refuses all four.

This is a bounded actual-operation verdict. The broader goal does not pass: held-out scores remain 51/62 and 29/66, below the cited 53/34 floors, and existing graph defects remain. No additional retirement, merge, semantic rewrite, retry, or queue operation is approved by this report. This reviewer performed no live write, provider request, install, deployment, or shared-repository edit. Synthetic refusals and repeat apply ran only in private replicas. The separate ChildScribe actual-live report remains its own verdict.

## Exact inputs and independent method

Fresh private `git archive d1f0a958608389a385ac9a9f57ec1eb941015a10` at `/tmp/scry-hollow-four-actual-independent.QwKeRi` supplies production source. I read the complete d1 gate, post-ChildScribe extension, immediate-freshness report, and exact manifest. I inspected and reused the earlier independent grader's graph and retained-recall harnesses, adding a new actual-snapshot raw/restore/prediction/refusal grader. Builder helpers and builder inventories were not used as proof.

Manifest `reviewed-manifest.json`, copied from `docs/memory-repairs/hollow-four-batch-2026-09-05.json`, has independently checked SHA-256 `8908910eae87bf9b1f288af86ccc4c2693a0173becb521ee13acd1d21ba9ffd5`.

| Artifact | Bytes | Independently verified SHA-256 |
|---|---:|---|
| Actual automatic backup `memory-20260905T225302Z.badger` | 73,551,199 | `cf7843f7c145b8af0497fc2409b7eb743e62bc4681f48fd0f23323931b13410a` |
| Actual post snapshot `memory-20260905T225312Z.badger` | 73,553,026 | `efa7466e022fb6a67918876aba8960eff9ae4aa1b1027eeeac52dd41ead9058a` |
| Final preapply freshness `memory-20260905T225213Z.badger` | — | `ccb289050456745336f1e33d561dde206de0c5895856c80fe0e875e1e055a9af` |
| Reviewed immediate source `memory-20260905T224743Z.badger` | — | `bcb1b2b5a6817f88f82adfee194566353e9d6348b380d67574a0e5358611b4a2` |

Local source files are under `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/`. Read-only SSH independently checked the actual two originals at `/Users/jclaw/.scry/backups/`; both hashes and sizes match the local copies.

The pinned binary `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry`, installed laptop `/Users/jeff/go/bin/scry`, and Mini `/Users/jclaw/.local/bin/scry` independently hash to `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`. Mini reports `scry d1f0a95`. Both retained siblings `scry.pre-d1f0a95-20260905T2156Z` independently hash to `821358499706bd9388b63a4368bb5320fc1bac61f12c93df48a158667b19bc14`. This verifies installed artifacts; the preceding deployment review supplies process/deployment history.

Each of the four backup inputs above was loaded directly through Badger into its own newly created directory **before candidate Store.Open**. The grader enumerated every logical key and copied every value without prefix filtering. Exact map equality preserves the distinction between absent keys and present empty values. The actual automatic backup, final freshness, and reviewed immediate source are completely equal across **242,798 keys**. Therefore no unreviewed semantic or graph delta entered before this actual operation.

Separately, candidate `Store.Restore` restored the actual automatic backup into `backup-restored/`; full-map comparison and closing/reopening proved it identical to the direct Badger load. Candidate opening and reopening the independently loaded actual post-state in `replica/` likewise preserved every raw value.

Logical hashes below use SHA-256 of Go JSON serialization of the complete sorted raw string map, matching the earlier independent grader's convention. They are distinct from physical backup hashes and from the lead's separately framed raw-stream hash; equality is checked on complete maps, not inferred only from hashes:

- Reviewed immediate source, final freshness, actual automatic backup, and independently restored/reopened backup: `6826fefb50103670eebf2dfb3921dedf004dd771c8cf41320aef04c1a219acf1`.
- Independently predicted post-map and actual post-map: `d28650504ec495c511b4c38a276c16166fac16af69e448512ecfaf13c94d42d6`.

## Exact preservation and semantic closure

The predicted mutation was constructed independently from the reviewed manifest and actual pre-state: remove four `en:` records and their four self-owned `al:` claims; add four `rs:` and four `rt:` records with the exact reviewed entity, ID, and reason. No fact, replacement, rehome, or reverse-index mutation is predicted. Actual post-state equals that prediction exactly. `actual-raw-delta.json` contains all 16 records with presence flags and full before/after bytes.

All **80,602 facts**, including **7,820 historical facts**, all provenance and validity fields, **9,354 episodes**, all three existing `ar:` decisions, every other alias claim, pending payload, cursor, metadata item, attestation, value-evidence record, adjacency, and preexisting retirement marker are byte-identical. Complete counts and per-prefix hashes are in `all-prefix-preservation.json`; complete map equality is the underlying preservation assertion.

Every original selected entity has zero current/historical fact endpoints, zero encoded raw fact-key references, zero raw reverse-index references, no external normalized spelling listing, and exactly its self-owned spelling claim. Full hygiene-fold enumeration also contains only the selected owner for each of its spellings. The actual preapply previews are ready and exactly match every manifest expectation, including rejection fingerprints; they do not write.

Complete pre-state equality carries forward the reviewed original episodes, all companion facts and contextual identities, and the semantic decisions: `all-tasks-implemented` is the ten-task/432-test outcome with separate blockers retained; `api-2312-tests-passing` is the measured empty-router baseline with real API/test identities retained; `engine-unavailable` is the historical worker outcome with original invalidation retained; `git-diff-check-clean` is the check outcome with sitemap artifact and QA constraints retained. I did not re-distill the original conversations; their semantic review remains in the preceding independent gate, and this review independently proves no intervening raw input changed. No inference about other hollow records follows.

## Complete graph inventories and refusal checks

| Inventory | Actual backup | Actual post |
|---|---:|---:|
| Facts | 80,602 | 80,602 |
| Entities | 30,611 | 30,607 |
| Hollow entities | 2,854 | 2,850 |
| Cross-type collision pairs | 484 | 484 |
| Conservative listing/claim defects | 4,433 | 4,433 |
| Dangling-endpoint facts | 1,995 | 1,995 |
| Self-loop facts | 1,096 | 1,096 |
| Current relations | 39 | 39 |
| Queue at 22:53:13 UTC | 0 ready / 0 backoff / 10 parked | same |

Every inventory is complete and uncapped. Complete collision-pair, listing/claim-defect, loop, dangling-fact, and current-relation inventories are equal; the hollow list is precisely the prior list minus these four. The 1,995 measure counts facts with missing endpoints, not 2,441 missing endpoints. Protected `43-url-sitemap` remains identical. Full lists are in `graph-audit.json` and `complete-defect-lists.json`.

All three literal ChildScribe rejection records, including exact plan `274e1e580ec0ac70464546990f4653f9956432071ffacde89d5d0487125f543f`, survive unchanged. On actual post replicas before and after reopen, ordinary stale `PutEntity` alias attempts and `ClaimAlias` attempts each refuse with `ErrAliasRejected`. Original entity recreation, same-slug/different-name recreation, and new-slug/retired-name recreation each refuse with `ErrEntityRetired`. Repeat Store apply and preview also refuse. All probes emit zero events and preserve the complete raw map.

The exact pinned CLI independently returns `applied=0, refused=4` on both private repeat preview and private repeat `--apply`, with all four records absent and no backup created. A separate read-only live preview also returns `dry_run=true, applied=0, refused=4`, with the same four absence reasons. No second live apply was issued by this reviewer.

Before retirement, canonical/name routing for all selected records resolves to the original entity and returns zero current/historical facts. After retirement, all routes and entities are absent. Independent live `memory facts <name> --all` calls for the five unique canonical/name spellings all return `memory: not found`; complete CLI output is in `live-exact-facts.json`.

Old marker-unaware binaries remain unsuitable for writing to this store now that the three ChildScribe rejection markers exist. Keeping their binaries does not make a binary-only downgrade safe, and this report does not approve restoring a pre-marker backup over intervening memory.

## Retrieval and reproduction

All four retained assertions appear at rank **1 before and after** in fresh local indexes. Their exact target records, queries, and historical times also equal the preceding independent review's probes. The unavailable-worker query uses `2026-08-20T00:00:00.000000001Z`, strictly within its original interval. Largest retained-probe response is **12,506 bytes**. Complete target evidence and responses are in `retained-assertion-recall.json`.

Five independent actual-snapshot controls use the pinned binary, fresh offline indexes, and suite files verified byte-identical to the shared repository. The command for each phase/suite is:

```sh
/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry memory bench --dir /tmp/scry-hollow-four-actual-independent.QwKeRi/PHASE --file /tmp/scry-hollow-four-actual-independent.QwKeRi/docs/memory-bench/SUITE.json
```

`PHASE` is `backup-restored` or `replica`. `SUITE` is respectively `heldout-2026-09-03`, `heldout-b`, `probes`, `tuning-strict`, and `tuning`. Before and after hits are respectively **51/62, 29/66, 7/7, 45/50, 47/50**. All ten runs have zero over-cap responses. Largest payload is **13,371 bytes before / 13,370 after**. Small payload-size differences do not change scores or cross the cap. Exact commands, suite hashes, misses, sizes and timings are retained in `bench-*-summary.json` and individual result files.

Passing independent commands, run from this archive:

```sh
go test ./internal/memory/store -run '^TestHollowActualRawAndGuards$' -count=1 -v
go test ./internal/memory/resolve -run '^TestHollow(FourIndependentGraph|CompleteDefects)$' -count=1 -v
go test ./internal/memory/recall -run '^TestHollowFourRetainedAssertionRecall$' -count=1 -v
node run-actual-controls.mjs backup-restored
node run-actual-controls.mjs replica
node check-live-facts.mjs
go test ./internal/memory/store -run '^TestHollowActualFinalNoWrite$' -count=1 -v
```

The final check after all private previews, refusals, recall and benchmark work confirms both complete raw hashes still equal their original actual pre/post values. Full results are in `final-after-cli-recall-bench-no-write.json`. Reproducing the initial restore test requires a new private archive and adjusted root constant, because its direct-load targets must begin empty. No test assertion failed during this review. Two exploratory source-path lookups used absent filenames and were corrected without mutation.

The lead reports the live apply returned four applied/zero refused with the automatic backup path. I did not execute or witness that original call; I independently verified its actual backup and actual post outcome, remote artifact integrity, and subsequent live absence/preview behavior. This review does not claim continued equality after future ingestion, improved global recall floors, or resolution of broader graph defects. The user-owned untracked assessment was left intact.
