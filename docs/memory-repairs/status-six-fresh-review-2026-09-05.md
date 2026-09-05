# Independent fresh six-status live-candidate gate — 2026-09-05

**PASS: the exact six-node/seven-fact candidate is ready for the reviewed live retirement workflow on deployed `62cf6e0d2d8db9d324da23114df837848e972c8f`.** The source itself is independently verified quiet, the complete semantic/ownership inputs have been freshly rechecked, and all independent preservation, raw-delta, rollback, backup-restore, lookup, and second-preview checks pass.

This is a fresh pre-apply gate, not a carry-forward of the earlier replica verdict and not an actual-live apply grade. No parked retry, network operation, wider candidate set, source-owner repair, benchmark/sweep result, or final memory-goal completion is approved here. The lead must still apply the exact reviewed payload using its nonempty backup and maintenance-locked expectation recheck, then obtain independent actual-live verification. New affected-input drift reopens review.

## Implementation and exact inputs

The grader independently archived full commit `62cf6e0d2d8db9d324da23114df837848e972c8f` into this directory. The deployment-discipline report was read. Git comparison confirms the production change from previously graded `393eeec` is confined to canonical-name resolution in `internal/memory/resolve/resolve.go`; retirement/store, daemon, and CLI code did not change. The complete resolver production diff was inspected, while the independent retirement tests and exact CLI preview were run from this new archive. The builder helper/replica/inventory were not used as proof. No live mutation, provider/model call, remember call, deployment, shared-repository edit, or external research occurred.

- Source: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/stable-205534-source.badger`, Mini `memory-20260905T205534Z.badger`.
- Source **78,001,227 bytes**, independently verified SHA-256 `2b6486ef39aa9e71148ef33c5b52919e3b6279022426d73d4accb25080d0f78a`.
- Manifest: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/status-six-fresh-replica/manifest-replica-only.json`, SHA-256 **`6b625c190057026d4824e2d89a9146c75697e469e0bb2673d2ffd5e34be7573a`**.
- Independent synced/closed backup: `independent-preapply.badger`, **78,001,219 bytes**, SHA-256 `f72b9f18ca3b4661839e47bd5dc7b8fa4f18bccfa2c393506fd2394c417867da`.
- Full raw source and independently restored backup logical hash: `fd58efd65cf0e902434d780f1eea268b60d77ca5eca21f0ae52d594d2f4a7078`.
- Independently predicted and actual replica post-state raw hash: `44bb2f1188b58e0d878a940da19b2a15642e80eb0897330095bf1f7ed95f6cc5`.

Logical hashes encode sorted maps of all raw Badger keys/values. Physical backup hashes are separate measurements.

## Queue truth and new baseline

On the independently restored source, deployed `Store.PendingCounts` returns **0 ready / 0 backoff / 6 parked** at `2026-09-05T20:55:35Z`. `source-queue-truth.json` records this check. This verifies the backup's own queue, rather than assuming the immediately preceding live CLI observation remained valid during capture.

| Measurement | Fresh source | After six retirements |
|---|---:|---:|
| Entities | 30,473 | 30,467 |
| Facts | 80,299 | 80,299 |
| Historical facts | 7,800 | 7,800 |
| Episodes | 9,329 | 9,329 |
| Current relations | 39 | 39 |
| Cross-type collisions | 484 | 484 |
| Self-loop facts | 1,095 | 1,095 |
| Dangling-endpoint facts | 1,995 | 1,995 |
| Hollow entities | 2,855 | 2,855 |
| Logical raw keys | 241,686 | 241,679 |

The old 482-collision and 7,793-history baselines are not reused. All complete defect lists and the relation set are identical before/after the six retirements, not merely equal in count. These preexisting global failures remain outside this bounded PASS.

## Complete fresh semantic closure

The complete candidate entities, all touching current/historical facts, all source metadata, all five full source episode records, all source-touching facts, and the 82-fact same-episode companion union were regenerated independently from the new source. Every candidate still has no outgoing or historical touching fact; five have one incoming fact, and `indexing-audit-clean` has two.

Matching manifest fingerprints were not accepted as sufficient proof. The entire regenerated semantic-evidence structure was compared with the previously reviewed evidence. The only difference is the `mac-mini` context entity's `last_seen`, advancing from `2026-09-05T15:49:04.871727-04:00` to `2026-09-05T16:53:52.598392-04:00`, appearing in three groups: canonical-route, live-checks, and network-authorization. All its other fields are unchanged. A dedicated independent assertion proves these three occurrences account for every byte-level semantic-structure difference. This host timestamp does not change the six identity/value decisions, source ownership, provenance, or artifact distinction.

- Fresh `semantic-evidence.json` SHA-256: `8a944bff750c40eb17dc8f967ad95685bf44267719caa6e7070cc80f5031f021`.
- Complete 82-companion artifact remains byte-identical: `250df05c75f5634719c128dbdcd25aefa0d0bcd2a3d3837327c7f4c5a710b59f`.
- Complete independent previews remain byte-identical: `e92dfb14b34de01cae3ec79cdfd4ec675eee0f377bc4992de4e601a48ab241c1`.
- `semantic-drift-review.json` records the exact metadata change and semantic judgment.

The renewed decisions are:

| Candidate | Freshly confirmed role |
|---|---|
| `app-tests-32-passing` | Earlier measured release result; original 32-test fact remains separate. |
| `app-tests-34-passing` | Later discovery-tooling result; the same-episode all-gates companion corroborates 34 app/18 ingestion tests. |
| `canonical-route-sitemap-passing` | Successful verification outcome after deployment 52489a6, not the route/sitemap or migration artifact. |
| `indexing-audit-clean` | Audit outcome preserved as literal `indexing audit clean` on both existing sources, `sitemap` and `statelicenselookup`. |
| `live-checks-passing` | Time-specific verification outcome after deployment adb2c25, not the live site or deployment artifact. |
| `network-authorization-pending` | Withheld-authorization state; the prepared plan/runner and explicit authorization requirements remain intact. |

The 32/34 facts retain their distinct text, confidence, values, and common stored day-level timestamp; neither is overwritten or silently invalidated. The `sitemap` entity's existing cross-project scope remains unresolved and unchanged; this representation repair does not endorse its global ownership. The network fact retains relation `blocked_by`, absence of raw relation, and full text saying explicit authorization is required. No fetch/crawl permission is created. `gate-green` remains excluded, with all its records untouched. Named plan/runner, guide, migration, sitemap, and other artifact identities remain preserved.

## Raw state, ownership, and failure probes

The fresh scan covered all 30,473 entities, **51,796 alias claims**, and 80,299 facts. Every candidate's complete spelling/claim set is owned only by itself, with no hidden routing key, outside normalized listing, or outside actual hygiene-fold match. All seven reverse indexes are canonical and empty; no stale, malformed, or unreviewed adjacency exists. Full entity/fact/alias/adjacency expectations match the supplied manifest.

For each replacement, independent comparisons require the entire original fact to remain identical except for `Dst=""` and `Value=retiredEntity.Name`. Generic JSON comparisons also prevent unmodeled raw fields from being dropped. All sources, relations/raw relations, text, timestamps, validity, confidence, and episode provenance are preserved.

The complete actual raw map equals the independently predicted result: **26 deleted records and 19 additions**. Six entity records, six claims, seven original edge facts, and seven reverse indexes are removed; seven value facts and twelve exact retirement markers are added. All unrelated raw records, including six parked entries, episodes, attempts, cursors, prior repair markers, value evidence, and metadata, remain unchanged.

The wrong sixth-group fingerprint test aborts every group with unchanged raw state and zero events. The forced postcondition sees all six removals and all seven replacements inside the transaction, rejects them, and rolls back everything with zero events. The real replica apply creates a nonempty synced/closed backup, independently restored to the complete source state, and emits exactly 20 fact/entity events.

All six retired entity/name lookups are absent. All seven literal values are found under their original source. Both source names resolve correctly to themselves in this snapshot. Both indexing-audit facts and both 32/34 results remain independently retrievable.

The second exact six-entry preview is nonready/nonapplied and causes no write/event. A separately executed CLI dry run from the exact `62cf6e0` archive returns `dry_run=true`, `applied=0`, `refused=6`, each because the retired entity is absent. A subsequent complete raw-hash comparison proves the CLI preview changed nothing. See `exact-cli-second-preview.json` and `cli-preview-no-write.json`.

## Handoff and reproducibility

This fresh source and exact candidate pass the bounded live-candidate gate. Proceed only with the unchanged six-entry payload, the reviewed deployed implementation, a nonempty actual live backup, and the built-in maintenance-locked revalidation/atomic apply. Keep retries and unrelated repair writes sequenced separately. If new processing changes affected entity/fact/alias/adjacency or semantic context, review that change instead of merely refreshing fingerprints. Independently grade the actual pre/post backups after application.

All review artifacts are in this directory: complete before/after entities and facts, semantic evidence/companions, semantic-drift review, queue truth, independent previews, mechanical results, graph audit, exact raw delta, source-value lookups, CLI output/no-write proof, independent backup/restoration, and grader-written tests.

```text
go test ./internal/memory/store -run '^TestStatusSixIndependent$' -count=1 -v
go test ./internal/memory/resolve -run '^TestStatusSixIndependentGraph$' -count=1 -v
go test ./internal/memory/store -run '^TestStatusSixFreshSourceQueue$' -count=1 -v
go run ./cmd/scry memory retire-entities --file /tmp/scry-canonical-62cf-deploy-sep05.G82GKc/status-six-fresh-replica/manifest-replica-only.json --dir /tmp/scry-status-six-fresh-grade.KrMFXo/replica
go test ./internal/memory/store -run '^TestStatusSixExactPreviewNoWrite$' -count=1 -v
```

All final checks PASS. Reproduce the main test in a new temporary directory because it creates backup/output paths exclusively. Actual live application, benchmark regressions, subsequent sweeps, broader status completeness, and global graph quality remain separately graded work.

