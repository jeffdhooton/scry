# Independent occupied fact-key prevention review

Verdict: PASS for the bounded prospective PutFact protection reviewed below. This is not a clean-store verdict, recovery of the overwritten historical fact, or completion of the memory-quality goal. No blocking violation was proved in this candidate.

The reviewer ran `scry memory orient --cwd .`, read the active goal and full COLLISION_PREVENTION_PROPOSAL.md, independently archived commit 226b3e8, and copied only the proposed production store.go plus its three new tests and explicit raw-corruption merge-test fixture into a private archive. All additional test work and restored stores stayed private. No shared edits, live writes, provider calls, queue actions, deployments, room posts, or durable notes were performed by this reviewer. Credential-bearing records were never printed; source facts used by the replay are identified below only by digest.

## Pins

- Private candidate: `/tmp/scry-fact-collision-independent.t4diGK`.
- Private baseline: `/tmp/scry-fact-collision-baseline.gaJyhl`, separately archived from 226b3e8 without the production patch.
- Production `internal/memory/store/store.go`: SHA-256 `600a299cf8177ebacafaa80cba2c3778a77545b5bf4f487ba972087e50cfd8a6`.
- Independently built candidate CLI: SHA-256 `4ee89adbf867a689e6ef3048ce03cdb58603724fae5e47a6cd6c1f8aaa0f90e1`.
- Pinned deployed baseline CLI `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry`: SHA-256 `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.
- Old source `memory-20260905T235134Z.badger`: verified SHA-256 `7680cdcff3563c4cc2d33577506fc4c4aad4931d8bae1016f90e617bd7ee863e`.
- Fresh source `memory-20260906T001120Z.badger`: verified SHA-256 `5b8e97894bea35a0ba2d1d63a1be01725845814051c8cb5c36273a0d4a5d1d6b`.
- Independent reproduction command source `cmd/collision-grade/main.go`: SHA-256 `d332f1b5bbf9379ccc9617574214173028b2b0f0cb57896852eaa153045550fd`.
- Independent extra adversarial tests `internal/memory/store/fact_collision_independent_test.go`: SHA-256 `c00557a465394b53d09b5977e1242db8733bedc312de5f9efd6d77bacaa71abc`.

## Disproof results

The production change performs one point lookup of the occupied exact key inside the existing write transaction. It compares source, canonical relation, exact destination, exact literal value, raw relation, sentence, and start instant. Any difference returns ErrFactConflict with a hash of the key. It adds no whole-store iteration, migration, ownership inference, deletion, timestamp adjustment, external call, or retention behavior. The guard applies regardless of InvalidAt, covering both current and historical records. Same-instant times in different zones compare equal.

Direct tests passed for current and historical collisions in raw value, sentence, and raw relation. Atomic tests verified full raw-key/value equality and zero observer events after refusal. Independent additions passed for malformed occupied `{`, `null`, `{}`, malformed timestamp JSON, and partial JSON; these failed closed with all bytes and events preserved. An independent same-transaction double insertion also refused the second assertion and rolled back the first. Meaningful InvalidAt, confidence, and episode-list updates to the same assertion remain accepted, including an equal instant represented in a different time zone.

The stale merge fixture now deliberately injects raw altered fact text below PutFact. Its existing snapshot-drift rejection and atomic no-write assertions remain intact. That is a valid corruption fixture, not weakened admission or merge behavior. The full store suite includes complete identity merge, historical preservation, key-collision refusal, unalias backup and drift, and reviewed retirement compatibility; all passed.

## Actual occupied-key reproduction and preservation

Both immutable sources were fully restored independently, with their SHA-256 checked before loading. The original occupied assertion digest is `44805a4cf211693496ac56d89433209a00dfc6e90fc192b0ee4ae2c36ba0e026`; the fresh replacement digest is `1c7aff7ef471dfb0cf756869db1ba8d398e8a11455fcf4e88a55bd7a4bd02f7e`. The harness checked that the old record was historical, the new one current, and their raw bytes differed. It read the new assertion in memory and submitted its exact value, sentence, canonical relation, and validity start through normal resolve.Apply, using a private synthetic review episode. It did not replay the original transcript or claim to reproduce every fact of the original extraction batch.

Unpatched 226b3e8 overwrote the occupied historical payload: Apply succeeded, reported FactsAdded=1 and FactsInvalidated=1, and emitted three events. The resulting replacement digest differs from the source replacement because the reproduction intentionally uses its own episode provenance.

The exact candidate returned ErrFactConflict with zero stats and zero observer events. Every raw key/value pair remained identical, including existing current-fact invalidations that the attempted Apply would otherwise have made. This verifies resolver-wide rollback against real old-store content, not only a fixture.

The full old raw digest is `3800785f9b2c746ff8850e1653ae77335fe842c138b4cead6c7ee505889c2160`, calculated by iterating sorted Badger keys and hashing uint64 big-endian key length, key bytes, uint64 big-endian value length, and value bytes. Prefix counts: adj 55769; al 52112; ar 4; att 10928; cur 3134; en 30725; ep 9371; fa 80844; meta 5; pq 12; rs 19; rt 19; ve 761.

A candidate Backup produced 73,975,028 bytes. Restoring that into another private store preserved the same complete raw digest and every prefix count. Open, index building, and five offline benchmarks also preserved that digest on both the candidate replica and the pinned-baseline roundtrip replica.

Queue coverage passed: the normal worker extraction/application path parks the conflict after exactly one attempt, retaining original pending text and source reference, preserving the old assertion and refusing episode commit. The existing deterministic ErrFactConflict classification already performs this parking; the patch changes no queue policy. Normal shared-token and two-episode alias admission, and reviewed alias rejection through normal Apply, passed their existing focused tests. The restored source's four alias-rejection markers and nineteen retirement markers stayed byte-identical under the real collision and read controls; this review does not claim to have replayed every marker's source episode.

## Recall controls

Independent candidate CLI versus the verified deployed baseline CLI, each against the same complete old raw content:

| Suite | Baseline | Candidate | Maximum payload, both |
| --- | --- | --- | --- |
| heldout-2026-09-03 | 51/62 | 51/62 | 12110 bytes |
| heldout-b | 29/66 | 29/66 | 13369 bytes |
| probes | 7/7 | 7/7 | 10095 bytes |
| tuning-strict | 45/50 | 45/50 | 11528 bytes |
| tuning | 47/50 | 47/50 | 11528 bytes |

Every suite reported over_cap=0. These match the immediate snapshot baseline, but the first two remain below the goal's original 53/62 and 34/66 thresholds. No new recall ranking change or fresh fifty-question grader was part of this bounded task.

## Commands and validation

- `CGO_ENABLED=0 go test ./...` in the independent candidate archive: PASS across all packages.
- `CGO_ENABLED=0 go test ./internal/memory/store -run 'Independent|RejectsDistinctOccupied|AllowsSameAssertion' -count=1 -v`: PASS.
- `CGO_ENABLED=1 go test -race ./internal/memory/store ./internal/memory/resolve ./internal/memory/queue -run 'Fact|Collision|MergeEntities|AliasRepair|Retire|Independent' -count=1`: PASS. CGO here is required solely by the Go race instrument; the normal whole suite and CLI build used CGO_ENABLED=0.
- `CGO_ENABLED=0 go test ./internal/memory/resolve ./internal/memory/queue -run 'ReviewedAliasRejection|AdmitAliasSharing|AdmitAliasWithoutShared|HistoricalCanonical|HistoricalCanonicalCollision' -count=1 -v`: PASS.
- `CGO_ENABLED=0 go run ./cmd/collision-grade /tmp/scry-fact-collision-independent.t4diGK/replica`: candidate actual-replay, raw preservation, backup restoration, index, and five-suite no-write controls PASS. The first harness output used incorrect summary field names for benchmark totals; subsequent independent CLI comparisons above supply the actual complete scores and payloads.
- Baseline archive: `CGO_ENABLED=0 go run ./cmd/collision-grade /tmp/scry-fact-collision-baseline.gaJyhl/replica baseline`: reproduced destructive overwrite.
- Both CLIs: `memory bench --dir <private replica> --file docs/memory-bench/<suite>.json`; outputs filtered to suite/total/hits/max_payload_bytes/over_cap before display.
- Candidate harness `go run ./cmd/collision-grade <replica>/old raw` and corresponding roundtrip raw command after all CLI benchmarks: unchanged complete raw digest and counts.

## Deployment compatibility and limits

This candidate keeps the on-disk schema, key layout, API, metadata-update policy, and queue format unchanged. The corresponding snapshot remains readable with the old binary, as independently demonstrated. No migration is required. Deployment approval is bounded to this exact production change after the lead verifies the integrated source/build, takes and restores a fresh nonempty live backup, retains the previous binaries, and records the deployed hashes. A rollback to the old writer reopens the proven overwrite hole; preserve the newer store and take a fresh backup before any rollback, then avoid draining conflict-bearing writes with the old binary. This review itself performed no deployment.

The already overwritten fact remains lost from the fresh source; prospective prevention does not recover it. Current canonical-triple coalescing before PutFact can still discard a distinct incoming sentence, and identical assertion metadata updates can still replace provenance lists or reopen historical validity according to existing semantics. Those unchanged boundaries are not approved as universally lossless and require separate evaluation before any whole-goal claim. No credential remediation, live alias repair, whole-store ownership audit, benchmark repair, or two-sweep acceptance is included in this PASS.
