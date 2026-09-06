# Independent actual-live stops-table repair grade

**Bounded PASS** for the actual Mini alias-only transaction represented by the automatic 2026-09-05 23:51:33 UTC PRE backup and immediate 23:51:34 UTC POST backup. The actual POST is exactly the independently predicted three-key transformation of the complete actual PRE. Every unrelated raw pair survives. This is not acceptance of the overall memory-quality goal, the remaining Child aliases, any fact relocation, a new owner for the spelling, or later live state.

Evidence and private source archive: `/tmp/stops-actual-independent.5w0cvt`. Source archive is deployed executable commit `d1f0a958608389a385ac9a9f57ec1eb941015a10`; shared repository was inspected at `86d7086286b88e9851e9bb1d27e5a5b5f0c6cae0`. I performed no shared-repository edits, live writes, provider calls, notes, queue retries, deployments, or target-account interactions. Private helpers were adapted only after reading them and were independently executed. Both actual backups and both earlier freshness backups were independently restored in full with direct Badger `Load`, before candidate `Store.Open` could alter them.

## Exact inputs

All backup and binary paths below are under `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/`.

| Input | Bytes | Independently verified SHA-256 |
|---|---:|---|
| `memory-20260905T235133Z.badger` actual PRE | 73,973,203 | `32de971aac86bdb228daf44d94667dff8511bea7a53a92d6dd09944f10019980` |
| `memory-20260905T235134Z.badger` actual POST | 73,975,028 | `7680cdcff3563c4cc2d33577506fc4c4aad4931d8bae1016f90e617bd7ee863e` |
| `memory-20260905T233353Z.badger` reviewed source | 73,971,229 | `907b9ae1e3210ac3840271e66ab95e34c9a24838863031812937c161bd076703` |
| `memory-20260905T234800Z.badger` immediate freshness | 73,971,458 | `9d6b0bf8cb797c8b6649de876c5916cdd566fc9f0e510ef25be77d8e074ed98f` |
| retained deployed `scry` binary | recorded in `input-hashes.json` | `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553` |

Committed manifest `docs/memory-repairs/child-stops-table-batch-2026-09-05.json` SHA-256 is `9f081e8f6180717ebfe6020adcbcd20bf190a40aa2754ec64e8584fee609c964`. Its sole drop is literal `stops table`, owner `childscribe-laravel`, empty rehome, and the complete reviewed literal reason. Plan: `ef7a0970016251e74b41e2db69c5d5c0d7b4b03c49e7a24a0b640df66906d35e`.

The immediate freshness report `/tmp/stops-drained-independent.uoAjxG/FRESHNESS-234800.md` independently hashes to `659094497a5fe1889247cf8412c1765ac921490371cf3d2d732d146abadaef18`. Input lengths and hashes are retained in `evidence/input-hashes.json`.

## Complete freshness drift, including queued input

Actual PRE contains 243,703 raw pairs. Compared with the independently loaded 23:48 source, there is exactly **one added key**, no changed values, and no removals:

`pq:d5d3b5818fe6b1e5776ecd215b985145e9f6321a7a7f1dbb273f00c9050a67a9`.

I decoded and read the full payload. It is a manual CADFormats workbench/deployment note, enqueued at `2026-09-05T19:48:55.384214-04:00`, with `attempts:0` and CADFormats/File workbench hints. It describes UI work, deployment checks, flags and rollback limits, and expressly leaves the broader workbench goal incomplete. It contains no Child/stops-table ownership evidence. It is queued unprocessed input, not an ingested episode or an independently verified account of that other deployment. I did not retry or adjudicate its unrelated assertions. Queue count is therefore **12**, comprising the previous eleven records plus this fresh note.

Compared with 23:33:53, actual PRE additionally changes only the two previously reviewed sweep metadata values: `meta:last_sweep_at` becomes `2026-09-05T23:34:14.576649Z`; `meta:last_sweep_report` becomes the full Mac.attlocal.net report of 94 files scanned, zero ingested, zero episodes and zero errors. Both complete values were read. All other pairs are identical. Complete decoded old/new payloads are retained in `immediate-to-pre-delta.json` and `reviewed-to-pre-decoded-delta.json`.

There is zero fact/entity/episode drift from 23:33:53. The existing semantic disposition remains supported: the Docket FK/table sources do not identify the Child service as a table; the `stops/` directory and migration file are distinct identities; no outside entity lists the exact normalized spelling. Original twelve raw source spans and saved projections, plus all ten additional bounded historical source spans/projections, were independently rehashed and matched to the current stored SourceRefs. This grade carries forward the complete semantic adjudication and explicitly preserved collateral inaccuracies in `child-stops-table-independent-review-2026-09-05.md` and `child-stops-table-sweep-gate-2026-09-05.md`; it does not claim a fresh manual rereading of all historical projections. The only new input was read completely as described above.

Original 164-fact domain closure hash remains `d76ea10656fb14757c0166aa179716611b818e72a6e9255cb064d48955bcdc59`; original twelve-episode metadata hash remains `f00080790020a03adbe54ec21e065d0cbb25600c0c79c86894eec1e50c5dbf87`. The 14 text matches, 76 companion facts, and all 177 broader stop/ticket/migration candidates remain unchanged before the repair. No ownership grounds were inferred from the archived collateral errors.

## Exact actual transformation and preservation

The private Go helper independently reconstructed **every** manifest Expected field from complete raw data, including all current and historical touching facts, every normalized listing, claim presence/owner, Child entity metadata and the complete previous Child rejection records. All fields equal the committed manifest. The touching fact count is 2,112 and its production serialization hash is `7e8b10b870e25afdf3bfb55912ae71f8dbf83e715000e5eed29db08973ceafec`; Child entity hash is `c7293a75b15d988dfd82a421349e8d80399352142e8264a94bcbc859398c115b`. The rejection fingerprint is `1a40bf4d7af133a80d2109949d4f21a45194ec5871a74630aeffac238dfe392a`.

Independent prediction and the actual complete POST agree byte for byte:

1. Replace `en:childscribe-laravel` by removing only the one normalized alias, 40 → 39. All other entity bytes, including description, type, names, repo references and timestamps, remain unchanged after JSON reconstruction.
2. Delete `al:stops-table`, formerly owned by Child.
3. Add `ar:childscribe-laravel:stops-table` with the exact literal alias, owner, manifest reason and plan. Preserve all three prior rejection values.

Go JSON/base64 whole-map hashes, distinct from physical backup hashes:

- Actual PRE: `763325a924b25dcbcfe40342e9510066c288a2f7d7d80b40c37c84f4544d61e6`.
- Independently predicted POST = actual POST: `0c766cd8dec9d9ea45ee43564cbd6740e30c659d9d7a65a65d032f759c0b4d5a`.

Both actual maps contain 243,703 pairs. Every one of 80,844 fact rows remains byte-identical: 73,013 current and 7,831 historical, including full text, invalidations, timestamps, confidence, provenance and both endpoints. All 9,371 episodes, all other 30,724 entities, outside aliases/claims/listings, 55,769 adjacency keys, 10,928 attention keys, 3,134 cursors, all five metadata values, all 12 queue payloads, 761 value-evidence keys, 19 `rs:` and 19 `rt:` markers are preserved. Claims change 52,113 → 52,112; rejection keys 3 → 4. Full family counts and byte comparisons are in `all-prefix-preservation.json`, full raw exports under `evidence/{before,actual,after,old,immediate}`.

A separate candidate replica reproduced the same prediction using a nonempty synced/closed atomic backup, independently restored that backup to the entire actual PRE map, and restored/reopened its output to the entire actual POST map. The extra private backup is 73,973,219 bytes; its differing physical encoding is not semantic drift.

## Reopen, replay and attempted bypasses

Actual POST direct-load/reopen resolves canonical `childscribe-laravel` to Child and returns all 2,112 current/history touching facts. Exact `stops table`, `STOPS_TABLE` and `stops-table` lookups are absent. Whole-map comparisons prove ordinary Open/close and these lookups do not write.

Second `PreviewAliasRepair` against the actual POST reports not ready; the second atomic backed apply refuses and reports not applied. `actual-second-no-write.json` retains both full previews/error and verifies every raw pair unchanged. Equivalent candidate second-pass checks also passed. The stale earlier manifest refuses without writes.

Stale direct Child entity writes and explicit alias claims refuse; atomic variants roll back a preceding sentinel entity entirely. Direct rehome to Child is rejected. Actual merge preview/apply in both Child/stops directions rejects unsupported rejection inheritance and leaves the entire map unchanged. Malformed rejection fixtures (`not json`, `null`, `[]`, `[{}]`) fail closed without mutation. Literal evidence, omission semantics, marker durability, merge preservation and rollback, Expected drift, backup write/sync/close failure, complete-listing requirements and concurrent-writer exclusion tests all pass.

Two stronger **real `resolve.Apply` calls** on a disposable direct restore of the actual POST each include a rejected alias spelling, a distinct synthetic target, a new precise fact sentence with confidence 0.87 and explicit valid_from, and a complete episode with fixed provenance/timestamps. Both calls succeed, retain exact fact text/confidence/timestamps/endpoints and complete episode metadata, and add exactly two facts total. Neither regrows or relists the rejected alias, including after reopen. Every original fact, episode, rejection/retirement marker and queued payload is unchanged. `strong-normal-retention.json` retains both complete new facts, episodes and resolver stats. Separate representability testing shows an independent synthetic identity can own the spelling; this is an implementation-scope test, not a rightful-owner finding.

The optional `TestAliasRejectionRestoredReplica` test skipped because its optional backup environment variable was absent; it is not counted as proof. The stronger actual-source tests above ran. No broader all-repository test run is claimed by this bounded grader.

## Complete structural inventories

Uncapped, sorted **complete defect lists**, not just counts, are equal before/after:

| Existing defect inventory | Before = after |
|---|---:|
| Dangling endpoint occurrences | 2,441 |
| Missing episode / missing adjacency / extra adjacency | 0 / 0 / 0 |
| Missing claims / wrong claims | 3,869 / 505 |
| Dangling claim owners / unlisted claims | 0 / 28 |
| Multiple listings / multi-type exact normalizations | 462 / 27 |
| Zero-fact / zero-current-fact entities | 2,849 / 2,967 |
| Self-loop fact rows | 1,099 |

These defects remain open; this transaction neither introduces a stranded listing nor fixes the wider graph. Full artifacts: `before-defects.json`, `after-defects.json`, `defect-counts.json`.

## All five actual before/after retrieval suites

All ten offline benchmark commands used the exact retained binary hash above against the independently direct-loaded actual PRE and actual POST databases. Suite files were independently hashed and proved equal to current repository files. Complete miss objects, including their associated fields, and exact mean answer ranks match; every response is below both 24,000 bytes and 24 KiB, with over_cap zero.

| Suite | Actual before = after | Mean answer rank, both | Maximum bytes before / after |
|---|---:|---:|---:|
| heldout-2026-09-03 | 51/62 | 4.8431372549019605 | 12,110 / 12,110 |
| heldout-b | 29/66 | 5.103448275862069 | 13,370 / 13,369 |
| probes | 7/7 | 1 | 10,095 / 10,095 |
| tuning-strict | 45/50 | 4.511111111111111 | 11,532 / 11,528 |
| tuning | 47/50 | 3.978723404255319 | 11,532 / 11,528 |

After all ten commands finished, the final reopened full raw-map comparison proved both actual benchmark controls unchanged. The final backed second apply then also proved actual POST unchanged. Full outputs are `bench-{before,after}-*.json`, `bench-comparison.json`, `bench-summary.json`. The goal's 53/62 and 34/66 floors are still missed, exactly as before this transaction.

## Reproduction and evidence identity

Executed in the private archive (use new private database directories for a fresh repetition):

```text
go run ./cmd/stops-actual <evidence> <actual-PRE> <committed-manifest> <233353-backup> <actual-POST> <234800-backup>
node audit.cjs
node bench.cjs
go run ./cmd/stops-strong <evidence> <actual-POST>
go test ./internal/memory/store -run 'TestStopsActualExactExtraPaths|TestStopsActualOldManifestRefuses|TestAliasRejection|TestReviewedAliasDrop' -count=1 -v
go test ./internal/memory/store ./internal/memory/resolve ./internal/daemon -run 'TestAliasRepair|TestReviewedAliasRejectionBeatsAttestationsAndShortcuts|TestMemoryUnalias' -count=1 -v
go test ./internal/memory/store -run 'TestActualPostSecondRepairNoWrite|TestStopsActualFinalReadNoWrite' -count=1 -v
node finalize.cjs
```

The helpers contain exact paths and the independently asserted predicates. `evidence/file-hashes.json` covers full raw/decoded exports, source checks, complete inventories, benchmark results, input identity, and all private helper/test sources. Its SHA-256 is `4883cddbe9ee0ab31fa6957f51fc6def08e49343528d442e1e4820b9732526cf`.

The Mini origin and automatic-backup timing are the root's handoff provenance; I independently verified local bytes and complete restored content, without performing an additional SSH/live-store inspection. This verdict is frozen to those actual inputs and does not establish later live state, two subsequent real sweeps, fresh held-out goal acceptance, a fully clean graph, or approval of other aliases/facts. An old marker-unaware binary remains an unsafe binary-only rollback after these durable rejection records; a complete verified rollback requires compatible code/state discipline. No global heuristic, alias backfill, inferred rehome, fact correction, timestamp change, or broader cleanup is approved by this PASS.
