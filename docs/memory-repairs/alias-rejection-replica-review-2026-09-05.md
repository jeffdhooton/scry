# Independent complete-replica alias rejection disproof — 2026-09-05

Bounded result: PASS for the tested prevention and preservation contract. No candidate defect was found. This is not a live repair approval, a rightful ownership finding, or a whole memory-quality goal PASS. No live store was opened, no provider was called, and no shared candidate file was edited. No prior 49-alias backfill was performed.

## Independence and input

The candidate code was independently copied to `code/` before testing. Production source hashes are recorded in `candidate-code-sha256.txt`. The only harness additions are `code/internal/memory/store/independent_audit.go` (read-only raw transaction access) and `code/internal/memory/resolve/independent_live_replica_test.go` (integration test). Those additions are confined to this temporary snapshot.

The complete input backup was independently checked:

- Source: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/memory-20260905T211821Z.badger`
- Bytes: 78,021,343
- SHA-256: `3f09b0d6622d4612811b83fc6738f845063c4d19b07ced41c7b893e9b4421e38`
- Restored counts: 80,308 facts, 30,472 entities, 9,330 episodes, 51,798 claims; also 55,437 reverse edges, 10,782 attestations, 3,115 cursors, 6 pending episodes, 434 value-evidence keys, 15 retired spelling keys, 15 retired slug keys, and 5 metadata keys.

## Exact repair and reversibility

The independent fixture explicitly dropped only `envoyer`, `office dashboard`, and `driver-core worktree` from the complete saved `childscribe-laravel` entity. It assigned no recipient. The fixture used a fresh no-write preview, pinned fingerprints, and an actual filesystem-backed durable preapply backup through `BackupAndRepairAliases`.

The complete raw pre/post maps were compared against an independently constructed exact expected state. Only these seven keys changed:

- `en:childscribe-laravel`: only the reviewed normalized alias removals, preserving every other entity field.
- `al:envoyer`, `al:office-dashboard`, `al:driver-core-worktree`: deleted claims previously owned by ChildScribe.
- `ar:childscribe-laravel:envoyer`, `ar:childscribe-laravel:office-dashboard`, `ar:childscribe-laravel:driver-core-worktree`: explicit owner-specific records with exact literal, reason, and plan fingerprint.

Every other key and value was byte-identical, including every current/historical fact, episode, pending item, adjacency, old attestation, cursor, existing entity, and metadata record. `original-raw.json` and `after-repair-raw.json` preserve the complete raw maps (values encoded as base64). `repair-delta.json`, `fresh-preview.json`, `pinned-manifest.json`, and `apply-receipt.json` preserve the exact review evidence.

`actual-preapply.badger` was 78,021,343 bytes and independently restored into `preapply-restored/`; its complete raw state matched the original exactly. A fresh second preview refused the already-completed repair; a second complete backup/apply attempt also refused with zero database changes. The backup artifact for this refused attempt is retained separately.

## Attempts to bypass prevention

All of the following were exercised against the complete restored graph:

- Direct `PutEntity` of the entire old ChildScribe snapshot returned `ErrAliasRejected`, with zero raw-key changes.
- The same stale snapshot through `AtomicWrite` returned `ErrAliasRejected` and rolled back an entity written earlier in the transaction.
- Literal, uppercase, underscore, and hyphen variants failed direct writes and `ClaimAlias`, preserving complete state.
- Existing attestation bytes survived the repair. Fresh `AdmitAlias` attempts rejected before creating new attestations. The source contains no preexisting attestations for these three exact normalized owner/alias pairs; that absence is recorded explicitly.
- A separate complete restore seeded two normalized attestations per alias before the repair and two more afterwards. All preexisting attestation bytes survived the repair, and further admissions still refused with zero raw-key changes. See `seeded-before-and-after-attestations.json`.
- TWO actual `resolve.Apply` calls each attempted the three aliases on the established ChildScribe identity. They succeeded in saving the two episodes, declined all three aliases, and changed exactly the two new episode keys. No entity metadata, aliases, fact, history, pending item, or other key changed. See `actual-apply-delta.json` and the two stats receipts.
- Close/reopen retained complete state and rejected the stale full entity write again. A real post-repair backup restored into another directory retained complete state and again rejected that stale write.
- A distinct synthetic `Envoyer` tool was representable and resolved independently, demonstrating that the marker is not a global spelling ban. This fixture does not establish rightful live ownership.
- `DropAliasRehome` back onto ChildScribe returned `ErrAliasRejected` with no state change.
- Merges in both orientations involving ChildScribe refused at preview and actual apply, explicitly requiring inheritance support. The refusals preserved every raw key. The marker could not silently disappear through the tested merge path.

## Execution and recall

Final command: `go test ./internal/memory/resolve -run '^TestIndependent' -v -count=1` in the isolated snapshot. Both integration tests PASS; final run 26.330 seconds. `test.log` contains the results.

Focused existing store/resolve/queue alias rejection, alias repair, and deterministic queue failure tests also PASS; see `candidate-regressions.log`.

Provider-free `recall.OfflineRecaller` built an in-memory index in 6.556 seconds. Three queries each took: Envoyer 8.069–8.608 ms; office dashboard 8.066–8.806 ms; driver-core worktree 52.189–52.698 ms; ChildScribe 17.842–18.871 ms. Complete raw state was unchanged by index construction and recall. Results and timings are retained in `recall-*.json` and `summary.json`. These are replica technical observations after the synthetic fixture, not a semantic benchmark or production latency claim.

## Limits

The review does not authorize deploying an older writer onto stores containing `ar:` records; the candidate explicitly documents that older writers ignore the additive namespace. Existing-marker merge inheritance remains deliberately unsupported, and tests establish refusal rather than inheritance correctness. No live remediation, prior 49-alias backfill, provider extraction, or broad memory-quality acceptance is inferred from these bounded checks.
