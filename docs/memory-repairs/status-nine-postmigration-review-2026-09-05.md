# Independent post-migration nine-status gate — 2026-09-05

Verdict: **PASS for the exact nine-entry candidate on the fresh post-migration source; contingent live-candidate gate.** All three independent tests passed again on an independently restored source. No failed disproof remains within these nine representation changes. This is not an actual-live apply grade, broad status-cleanup approval, or final memory-goal grade. Final drift conditions below remain mandatory because the lead reports a durable operation note was queued after source capture.

The independently extracted deployed implementation is commit `393eeec79f80d3b4becff276c4fcffd71fa68ac5`. The grader extracted this commit again into this temporary directory and reran its own previously written tests against the new source and newly regenerated manifest. No builder replica or builder helper supplied the result. All writes were confined to this temporary checkout and its disposable stores; no live write, deploy, shared-repository edit, remember, model call, or external-service research occurred.

## Exact new source and candidate

- Source: `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T200715Z.badger`.
- Source size: **77,829,763 bytes**; independently verified SHA-256 `5323fef7b594c748fd61462909cc808f78c1bd0a104da070abdc54cb21e102aa`.
- Manifest: `/tmp/scry-migration0160-fresh-sep05.teYtyC/status-nine-postmigration/manifest-replica-only.json`.
- Manifest SHA-256: `a739d9d95de7ac8a8ff8e92ea33e3f717cb1cdf2696d09327c64dbc6bd94f36a`, identical to the previously reviewed nine-entry candidate.
- Independently generated rollback backup: `independent-preapply.badger`, **77,829,723 bytes**, SHA-256 `f968157251ba29c1333e667825d79999c76c70decc696ce3bebf3b294487c57a`.
- Complete raw logical state before apply and after independent backup restore: SHA-256 `26f22e7a84815864f9032abc29c918f674dd7d76ec014fc518b8e0b1b750981c`.
- Independently predicted and actual complete raw logical state after apply: SHA-256 `e657f6610ee92d269a62f954ba7977a0fbbc47a6c2d1e2cd8b5d2fc612ad1813`.

Raw logical hashes use JSON's sorted map encoding of raw Badger keys and values. Backup-file hashes are separately measured physical backup hashes. The independent backup was synced/closed by the deployed backup-coupled retirement path and restored into a separate store; its entire logical state matched the new source.

## Fresh semantic and ownership check

The source was rescanned for all facts, all entities, all alias claims, complete touching history, and candidate reverse indexes. All nine still have exactly one current incoming fact, zero outgoing facts, and zero historical touching facts. Their entire entity snapshots, descriptions, repo references, timestamps, source episodes, source metadata, source-touching companion facts, and same-episode companions are unchanged from the prior independent grade.

This is not inferred from matching manifest hashes alone: the independently regenerated `semantic-evidence.json` is byte-identical to the previous file, SHA-256 `6c587c4803f8f6004fdd660cc6e7d16344fffc0ea28c1d95c8a38c5ada52e231`. Independent preview records are also byte-identical, SHA-256 `529f6a80140cbce86002bf691d7b86c20395a087c8d0388dd5965a9b7b042114`.

The nine reviewed values remain:

| Candidate | Confirmed semantic role |
|---|---|
| `1-complete-priority-cell` | One-of-24 strict launch-gate measurement on the licensing project. |
| `164-fields-have-field-evidence` | Evidence count on the named backlog artifact. |
| `193-pages-blocked-on-authority-verification` | Disputed numerical backlog assertion, still a measurement. |
| `21-suspect-bodies-removed` | Sanitizer result on the retained ingestion engine. |
| `50-jurisdictions-published` | Seed/publication coverage assertion, not a named artifact. |
| `641-fields-have-content` | Disputed numerical content count, not the backlog identity. |
| `actively-building-maine` | Temporary builder activity on the existing project. |
| `addressing-review-findings` | Implementer's activity state, distinct from the retained review thread/task. |
| `ai-sync-tests-15-passing` | Passing-test measurement, distinct from the retained program and pytest-file identities. |

Every replacement was independently compared against the full original fact, requiring only `Dst=""` and `Value=entity.Name`. Source, relation, raw relation, text, timestamps, validity, confidence, provenance, and any unmodeled raw JSON fields remain unchanged. No source owner is inferred or reassigned. `43-url-sitemap` and all other noncandidate identities remain untouched.

All 51,749 alias claims and 30,436 entities were inspected afresh. Each candidate has exactly its own normalized claim, no extra routing key, no external listing, and no other entity sharing its actual hygiene fold. Each has exactly one empty canonical reverse index, with no stale/malformed/unreviewed extra record. No rehome is needed. Every supplied entity/fact/alias/adjacency expectation matches independently generated previews.

The three earlier factual caveats remain: episode `37eb697c…` says 1,062 authority-blocked pages and 193 content fields, disagreeing with the stored 193/641 assertions; episode `53cff505…` distinguishes fifty seeded jurisdictions from a same-episode assertion that sixteen state/DC guides remained unpublished. This gate approves preservation of those existing assertions as values, not their numerical or publication accuracy. Separate fact correction remains required.

## New baseline and measured effect

| Measurement | Fresh source | After nine retirements |
|---|---:|---:|
| Entities | 30,436 | 30,427 |
| Facts | 80,203 | 80,203 |
| Invalidated facts | 7,792 | 7,792 |
| Current relations | 39 | 39 |
| Cross-type collisions | 482 | 482 |
| Existing self-loop facts | 1,095 | 1,095 |
| Existing dangling-endpoint facts | 1,995 | 1,995 |
| Hollow entities | 2,854 | 2,854 |
| Logical raw keys | 241,337 | 241,328 |

All complete defect lists and the relation set are unchanged, not merely their counts. The migration's changed baseline is independently confirmed: compared with the earlier source, five fewer entities, two fewer collisions, one fewer hollow, and two more alias-index keys. These facts describe the supplied source; this grader did not regrade the migration's actual live operation.

The exact raw retirement delta remains 36 deleted keys and 27 added keys: nine entities, nine alias claims, nine old edge facts, and nine reverse indexes removed; nine value facts, nine slug tombstones, and nine spelling tombstones added. No unrelated key/value changes occur. All 9,321 episodes, four pending entries, attempts, cursors, value evidence, and metadata records remain identical to the source.

The ninth-group drift probe rejected the entire batch with zero events and identical complete raw state. A separate forced postcondition saw all nine tentative conversions, then rejected them; all changes rolled back and zero events escaped. Successful apply emitted exactly 27 events. All retired exact entity/alias lookups are absent. Second previews are nonready/nonapplied with zero writes/events. Generic JSON comparisons independently confirmed no hidden fact field was lost.

## Conditions immediately before live application

1. Wait for the queued operation note to settle; capture and retain the exact fresh stable queue state and source. Do not treat time elapsed as stability. This grade does not assume the post-capture note is graph-inert.
2. Verify the deployed implementation and exact nine-entry payload still match this gate. Require every live preview to be ready and all complete entity/fact/alias/adjacency expectations to equal the reviewed ones. Keep the nonempty backup and apply in the deployed maintenance-locked operation; its transaction must recheck expectations.
3. Recheck closure beyond the manifest hashes: all current and invalidated touching facts, affected alias-index ownership including hidden routing keys, every outside normalized listing and hygiene-fold match, absence of occupied replacement keys, full source entities, source episode contents, and relevant companion context. The manifest fingerprint does not itself hash episode-summary content or all semantic companion context. Any change to those reviewed inputs reopens this grade; do not just copy new fingerprints.
4. If the queued note creates unrelated graph records, measure a new whole-store baseline and calculate the same exact nine-entry delta from that state. Post-apply global counts must be compared with that immediate baseline, not these older totals. Unrelated raw records must remain unchanged by the retirement.
5. Preserve the backup, independently verify the actual apply's exact raw and semantic postconditions, restore-verify the rollback artifact, and perform the second dry run. No further status nodes, corrections to the disputed assertions, or guessed owners are covered.

## Artifacts and tests

The semantic record, full before/after fact/entity exports, `independent-previews.json`, `mechanical-results.json`, `graph-audit.json`, `raw-delta.json`, `raw-payload-and-lookup.json`, and independent backup all reside in this directory. Tests run from this independently extracted checkout:

```text
go test ./internal/memory/store -run '^TestStatusNineIndependent$' -count=1 -v
go test ./internal/memory/resolve -run '^TestStatusNineIndependentGraph$' -count=1 -v
go test ./internal/memory/store -run '^TestStatusNineIndependentRawPayloadAndLookup$' -count=1 -v
```

Results: PASS (25.28 s), PASS (1.57 s), PASS (0.06 s). The first test creates its backup exclusively and should be reproduced only in another fresh temporary directory, adjusting the root constant. The existing replica outputs are retained for inspection.
