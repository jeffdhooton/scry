# Third ordered overlay — root-only restored-replica evidence

2026-09-06 11:23 UTC. This is conservation evidence, not an independent grade,
production approval, ingestion success claim or whole-goal PASS.

Third frozen candidate manifest:
`76808342886d78e364b6e3f38eb5e98e9d0a4f79bb099a64a42de5979f8cdea1`.
All600 e097fa6 baseline files and41 candidate additions were copied and verified
exactly in `/tmp/scry-overlay-third-replica-sep06.eqm86G/code`. The only extra source,
`internal/memory/store/private_overlay_replica_test.go`, is root-only and MUST NEVER
ship; SHAa4f9b2ca83a251dff08a19e5f3287522271303abed5c88cad127a9bb5171640a.
It differs mechanically from prior probeae03ddad only in snapshot pin, time and
synthetic new names. Existing actual metadata is used only privately, never printed.

## Fresh complete snapshot

Mini backup `/Users/jclaw/.scry/backups/memory-20260906T111946Z.badger`,
77,021,042bytes, SHA-256
`0f4e9dec22c46964cf48896b1e9befdd21803c376ff8ecb994b557477b0ea459`.
Local copy has the same full hash. Restore helper verified the hash and restored
into new `shared-111946` and `overlay-111946` directories beneath
`/tmp/scry-foundation-closure-sep06.8IEPu5`. All249,362 raw key/value rows stayed
exact through restore, indexing, full entity/fact reads and close:

`373eda04e053f7e87ac6b6ea5656dbfd44a2f9363423c7447bd285d2cb40d726`.

Counts: en31500, FA82353 (current74448/history7905), EP9487, pending34,
adj56765, al53179, ar4, att11401, cur3199, meta5, rs19, rt19, ve1397.
The `shared-111946` restore remains untouched apart from read/open-close activity.

## Fixed-entry probe

Explicit env-gated probe ran with contained TMPDIR and no CGO:
`SCRY_PRIVATE_OVERLAY_REPLICA=111946 go test ./internal/memory/store -run
TestPrivateOverlayRestored111946 -count=1 -v`.

PASS first run, package16.623s. Log
`/tmp/scry-overlay-third-replica-sep06.eqm86G/actual-third-replica.log`, SHA
`d5a623e56ff41ad1057aeef15669dae28804c42947c33b1d82794674b5d5efd0`.

Before private adoption, actual existing identity remains typed unadopted deferral;
unrelated synthetic utility remains proposed. Two never-materialized candidates,
two projections,27 witnesses,5 decisions; all249,362 raw rows unchanged, zero
entity/fact writer entries and zero events. Reopen repeats that result. Single
planner calls2.854097583s and2.856715834s are not p95 measurements.

The probe explicitly previews and applies adoption ONLY to its private replica.
Preview changes nothing. Apply adds31,501 anchors/marker and preserves every
original row exactly. Manifest16,463,209bytes, SHA
`85dcfe11f0ebcbc90f7c72c58ce7a9989a8331ed37a8cfcceec8b6281d8c23c0`.
Post-adoption280,863 raw rows, digest
`f41ef824f6caf1d80e19694fdc8dab965ec5d4c11501b9bcecf82b28df390341`.

On that explicitly adopted replica, actual existing legacy identity routes while
both new candidates remain unmaterialized. Three projections,108 witnesses,
6 decisions, all280,863 raw rows unchanged by planning, zero writer entries/events.
Reopen repeats the result. Single calls2.848694417s and2.842271917s. Do not rerun the
untouched-snapshot initial precondition against this now-adopted directory.

## Durable-note closure and limits

Previously accepted note
`acc0f2532a529a1d115a892f0f41ec764cc36ec593b60ce72b706fe5abc2f8ff`
is present in the restored durable pending queue, parked after one fact-conflict
attempt, NOT ingested. Pending raw SHA
`93179b245ea760c9729ccbf67af462ffd945a950d8f25798b18acbbdfaf14451`;
error SHA`f294fc67f260e463bc2dc0f0e2aee54e698354a3ddb93963b74675897c1120ea`.
Only safe status/hash fields were printed. No retry, unpark, overwrite or provider
operation was performed. Durable receipt is not successful graph ingestion.

Both rejected source versions remain preserved. Third independent review is still
running. Installed laptop/Mini remain a06cd7b; no live adoption, deployment, graph
repair or sweep occurred. Fact support/preservation, lifecycle/all-writer enforcement,
cleanup, recall and two full independent grading rounds remain open.
