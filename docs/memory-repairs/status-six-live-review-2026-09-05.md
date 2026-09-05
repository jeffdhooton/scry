# Independent actual-live six-status repair grade — 2026-09-05

**PASS for the bounded actual six-node/seven-fact retirement.** The actual post-apply store exactly equals the independently predicted whole raw map derived from the actual pre-backup. All 80,299 facts, including 7,800 historical facts, are preserved. All seven exact source-value lookups succeed, the six retired names are absent, the second exact CLI preview changes nothing, and complete graph-defect lists are unchanged.

This is an actual repair result, separate from the earlier fresh-candidate PASS. It does not certify broad status completeness, global graph health, any parked retry, network authorization, benchmark/sweep acceptance, or completion of the overall memory goal.

The grader independently archived deployed commit `62cf6e0d2d8db9d324da23114df837848e972c8f`, independently restored both actual backups, reconstructed the exact reviewed operation in another replica, and compared every raw key/value. Builder summaries, root CLI output, and root benchmark claims were not used as proof. All writes occurred only in this temporary checkout and disposable stores; no live write, retry, remember, deployment, provider/model call, or external research occurred.

## Exact actual backups and manifest

- Manifest: `/Users/jeff/workspace/context-stack/scry/docs/memory-repairs/status-six-retirement-2026-09-05.json`, recorded in commit `19ab9ece3251df03a0ee7be484af10bfdd3733f2`, independently measured SHA-256 **`6b625c190057026d4824e2d89a9146c75697e469e0bb2673d2ffd5e34be7573a`**.
- Actual automatic pre-backup: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/memory-20260905T210530Z.badger`, **78,001,464 bytes**, SHA-256 `8bf422a4781869931c7e289caedde5b2749e059a0fcea14a520aaea20e47aa7b`.
- Immediate actual post-backup: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/memory-20260905T210535Z.badger`, **78,004,674 bytes**, SHA-256 `ccd3dbd6fc4143458a6682481be5aba422d0ecc29f8f806c1db67c22dc9020b3`.
- Actual pre / independently restored replica backup raw logical hash: `8d4d1b43c4b4e5df75dbb7cf0ba6b70b762dd92c68a960b5e7268528d4e3cb08`.
- Independently predicted / independently applied replica / actual post raw logical hash: **`445934d17d00081980491450e3814a44c0753bad7423699ae3d1b04f6226f3c6`**.
- Independent backup generated during reconstruction: `independent-preapply.badger`, **78,001,456 bytes**, SHA-256 `cd3aae0ce71845c61649a73988b76375f346011c5bba036a597fa07c29e3e548`; independently restored to the complete actual pre-state.

All backup sizes and file hashes were independently measured. Logical hashes use sorted JSON encodings of complete raw Badger key/value maps and are separate from physical backup-file hashes.

## Reviewed-source to actual-pre drift

The approved 20:55 source has logical hash `fd58efd65cf0e902434d780f1eea268b60d77ca5eca21f0ae52d594d2f4a7078`. Comparing every raw record with the actual pre-backup finds exactly two changes:

1. `meta:last_sweep_at` advanced from `2026-09-05T20:46:48.120982Z` to `2026-09-05T21:04:14.135005Z`.
2. `meta:last_sweep_report` changed from the earlier report (2,344 scanned files, two ingested files, one episode) to a report for `Mac.attlocal.net` with 94 scanned files, zero ingested files, zero episodes, zero errors, finished at `2026-09-05T17:04:14.135005-04:00`.

There is **no cursor drift**. Every entity, fact, episode, alias claim, reverse index, pending record, attempt record, value-evidence record, and existing retirement marker is identical to the approved source. Full before/after values of both sweep records are retained in `stable-to-actual-drift.json`. This metadata change is accounted for in the actual-pre-derived prediction and is not silently normalized away. Both records then remain unchanged across actual retirement.

The independently regenerated complete semantic evidence is byte-identical to the fresh review, SHA-256 `8a944bff750c40eb17dc8f967ad95685bf44267719caa6e7070cc80f5031f021`. This covers all six candidate entities, seven touching facts, five full source episodes, the complete 82-fact same-episode companion union, all source-touching facts, and contextual entity metadata. The actual apply therefore has no unreviewed semantic/input drift despite its different physical backup hash.

## Exact changes and preservation

The six actual retirements are `app-tests-32-passing`, `app-tests-34-passing`, `canonical-route-sitemap-passing`, `indexing-audit-clean`, `live-checks-passing`, and `network-authorization-pending`. They remain the reviewed measurement/activity/authorization-state values, with seven current incoming facts and no outgoing or historical touching fact.

The complete actual delta is **26 deletions and 19 additions**: six entities, six reviewed alias claims, seven original edge facts, and seven canonical reverse indexes removed; seven literal attribute facts and twelve exact durable retirement markers added. The entire actual post-store map equals this independently constructed prediction, including all tombstone payloads. No unrelated raw mutation occurs.

For all seven facts, only the selected `Dst` becomes empty and `Value` becomes the retired entity's exact display name. Independent structured and generic JSON comparisons preserve source, relation, raw relation, text, timestamps, validity, confidence, episode provenance, and any otherwise unmodeled raw field. All current and historical facts outside the seven remain byte-for-byte unchanged.

The 32-test and 34-test results coexist as distinct values with their original fact text/confidence and common stored day-level timestamp. Neither is overwritten, merged, or invalidated. `indexing audit clean` is preserved under both `sitemap` and `statelicenselookup`, with two distinct complete assertions. `network-authorization-pending` keeps relation `blocked_by` and the full statement requiring explicit network authorization; this repair grants none.

The existing broad `sitemap` identity and all its metadata/other facts remain unchanged and are not endorsed as correctly scoped across projects. `gate-green` stays excluded and untouched, as do named plans/runners, routes, migrations, guide artifacts, `43-url-sitemap`, and every other noncandidate entity.

## Actual counts, alias state, and queue

| Measurement | Actual pre | Actual post |
|---|---:|---:|
| Entities | 30,473 | 30,467 |
| Facts | 80,299 | 80,299 |
| Historical facts | 7,800 | 7,800 |
| Episodes | 9,329 | 9,329 |
| Alias-index claims | 51,796 | 51,790 |
| Current relations | 39 | 39 |
| Cross-type collisions | 484 | 484 |
| Self-loop facts | 1,095 | 1,095 |
| Dangling-endpoint facts | 1,995 | 1,995 |
| Hollow entities | 2,855 | 2,855 |
| Logical raw keys | 241,686 | 241,679 |

The complete relation set and full self-loop/dangling/hollow lists are identical before/after, not merely equal in count. Existing global defects remain broader-goal failures.

The independent pre-scan covered all 51,796 alias claims and all entities. Candidate spellings have no hidden routing key, outside normalized listing, or outside actual hygiene-fold match. Exactly six candidate claims are removed, all other claims remain identical, and all six retired name/entity lookups are absent. No stale candidate adjacency remains.

The actual post-store's natural source names `statelicenselookup` and `sitemap` both resolve to themselves. Exact facts lookups through those routes find all seven complete replacements. `actual-source-values.json` contains each preserved fact and the verified source/value route.

At `2026-09-05T21:05:36Z`, the deployed store's `PendingCounts` on each actual backup independently returns **0 ready / 0 backoff / 6 parked**. All parked payloads are preserved byte-for-byte. No retry occurred in this review.

## Backup/rollback and independently repeated CLI

The actual automatic pre-backup is nonempty and independently restorable. On its restoration, an independent wrong-last-group fingerprint probe and a forced postcondition after all six/seven tentative conversions each roll the complete batch back, leave every raw record unchanged, and emit zero observer events. These are replica failure proofs; no live rollback was performed. A new backup-coupled replica apply emits 20 expected fact/entity events and restores its independent backup to the exact original raw map.

Against the independently restored actual post-store, direct repeat previews report all six missing entities and make no write. An additional exact CLI dry run built from the `62cf6e0` archive independently returns `dry_run=true`, `applied=0`, `refused=6`, with each refusal saying the retired entity does not exist:

```text
go run ./cmd/scry memory retire-entities --file /Users/jeff/workspace/context-stack/scry/docs/memory-repairs/status-six-retirement-2026-09-05.json --dir /tmp/scry-status-six-live-grade.rgSZPP/actual-post-restored
```

No `--apply` was supplied. A subsequent whole raw-map hash check proves this exact CLI dry run changed nothing. Full output is in `exact-cli-second-preview.json`; the result is in `actual-cli-preview-no-write.json`.

## Evidence and limits

This directory contains the independently extracted implementation, grader-written tests, actual pre/post restorations, independent reconstructed result/backup, complete before/after and actual entity/fact exports, semantic evidence/companions, previews, mechanical results, graph audit, exact raw delta, stable-to-actual drift, actual source-value checks, actual queue/results, and exact CLI/no-write evidence.

Passing tests: `TestStatusSixIndependent`, `TestStatusSixActualLive`, `TestStatusSixIndependentGraph`, and `TestStatusSixActualCLINoWrite`. The main test creates exclusive output/backup paths and must be reproduced in a fresh directory.

The lead's five post-repair suite figures were not used as independent evidence in this bounded graph-repair grade; benchmarks and subsequent sweep acceptance remain separately reported work. The two observed sweep metadata records establish only accounted operational drift, not a complete two-sweep memory-goal pass. No global completeness or final-goal claim follows from this actual-repair PASS.

