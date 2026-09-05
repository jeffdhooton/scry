# Independent latest-source nine-status gate — 2026-09-05

**PASS for the exact nine-entry replica operation and complete graph/source drift proof. Stable-source claim disproved: live-candidate readiness remains OPEN, with no live apply clearance.**

The latest supplied backup contains **one ready episode, zero backoff, and five parked**, not a quiet 0/0/5 source. The graph itself remains exactly identical to the independently reviewed post-migration source. This distinguishes a proven queue-stability failure from the nine retirement operations, which passed all independent mechanical and semantic checks again.

This report covers deployed commit `393eeec79f80d3b4becff276c4fcffd71fa68ac5`, independently extracted into this temporary directory. It does not review the lead's subsequent in-progress resolver changes, approve any deployment, or grade the full memory goal. No live writes, deployments, shared-repository edits, remember calls, external research, or model calls were performed by this grader.

## Exact inputs and hashes

- Latest source: `/tmp/scry-migration0160-fresh-sep05.teYtyC/status-fresh-source.badger`, corresponding to Mini `memory-20260905T201333Z.badger`.
- Source: **77,838,184 bytes**, independently verified SHA-256 `1d6fec70035c631eff4b13e4c86c65d3869501bb1e16a88ebcd9b844078a3459`.
- Manifest: `/tmp/scry-migration0160-fresh-sep05.teYtyC/status-final-source-replica/manifest-replica-only.json`, SHA-256 `a739d9d95de7ac8a8ff8e92ea33e3f717cb1cdf2696d09327c64dbc6bd94f36a`.
- Independent nonempty backup: `independent-preapply.badger`, **77,838,168 bytes**, SHA-256 `f75d942520b11eaf151685bc12757ec9ec89ffa3b12d83fe8dc54f87263f3308`.
- Entire raw source state and restored independent backup hash: `f99e8c500fd081b06ebdd56da3576227296ea5ddfe7ae36e26f50180be8b27a3`.
- Independently predicted and actual raw post-apply state hash: `1ebb4aaa907d9905fbefd5d858b7490671c0737286d6fe0608402a2d6d82e6fd`.

Raw logical hashes are sorted JSON encodings of complete raw key/value maps; backup-file hashes are separately measured.

## Full drift proof and queue finding

Compared with the reviewed 20:07:15 post-migration source, exactly four records differ:

1. New cursor `cur:e7fd02c034ae5eb0692e3849e07cbeb252b2bed2dff0c4da5c7ee0d6767ffa9a`, recording 1,070,157 processed bytes of a cadformats Claude session.
2. `meta:last_ingest_at` advanced from `2026-09-05T19:43:25.183891Z` to `2026-09-05T20:13:33.188335Z`.
3. New parked operation-note record `pq:ed50810befba04287fe6676b18db368af6808d32508980ae6e49373009230727`, retaining its full episode after an alias ownership refusal involving the canonical migration path.
4. New **ready** record `pq:ff7c244169ca5edf19ecd06b82e3831c0d2010536d01cae966c30adc55848d36`, source `claude-session`, working directory `/Users/jeff/workspace/cadformats`, attempts zero, parked false. Both enqueue and next-attempt time are `2026-09-05T20:13:33.188335Z`.

The deployed store's `PendingCounts` independently returns **1 ready / 0 backoff / 5 parked** at that timestamp. `source-queue-truth.json` records this evidence. Six `pq:` keys are present. The earlier 0/0/5 status observation therefore cannot describe the captured backup. The lead acknowledged the overlapping arrival and retained the no-live-apply hold.

Every other raw record is identical to the prior source. Excluding cursor/attempt/queue/metadata namespaces, all **227,461** raw records have identical maps with SHA-256 `64b68f8baf4c1ecf35e05db0db1c9b8344c7d6fb477e902f36db26ea0ad69fea`. This includes every entity, fact, reverse index, alias claim, episode, and value-evidence record. Attempt records were separately compared and did not change. The initial drift probe intentionally rejected the unexpected cursor namespace; after inspecting its exact session path, byte count, and matching enqueue event, the grader classified it as ingestion bookkeeping and reran the comparison. No graph drift was waived.

## Semantic and ownership closure

The nine candidates, their complete metadata, all current/historical touching facts, complete source episodes, all source-touching and same-episode companion facts, and companion entity metadata were regenerated from this source. They are byte-identical to the preceding independent semantic evidence: SHA-256 `6c587c4803f8f6004fdd660cc6e7d16344fffc0ea28c1d95c8a38c5ada52e231`. Independent previews are byte-identical too: SHA-256 `529f6a80140cbce86002bf691d7b86c20395a087c8d0388dd5965a9b7b042114`.

The reviewed entries remain `1-complete-priority-cell`, `164-fields-have-field-evidence`, `193-pages-blocked-on-authority-verification`, `21-suspect-bodies-removed`, `50-jurisdictions-published`, `641-fields-have-content`, `actively-building-maine`, `addressing-review-findings`, and `ai-sync-tests-15-passing`. Each is a measurement or activity state, has exactly one current incoming fact, no outgoing fact, and no historical touching fact. No source relocation or guessed owner is involved.

All nine normalized claim sets still resolve only to their respective candidates, with no hidden extra routing key, outside listing, or outside hygiene-fold match across all 51,749 claims and 30,436 entities. All candidate reverse indexes are the nine empty canonical mirrors. The source entities, named backlog JSON artifact, rendering review/task, actual ai-sync pytest artifact, and deliberately excluded `43-url-sitemap` remain intact.

All three factual caveats remain open and unchanged: the stored 193-blocked-pages and 641-content-fields assertions disagree with the source episode's 1,062/193 counts, and fifty seeded jurisdictions must not be equated with verified publication coverage where same-episode evidence says sixteen guides remained unpublished. The operation preserves these existing assertions as values without correcting or endorsing them.

## Independent apply/rollback result

| Measurement | Latest source | After nine retirements |
|---|---:|---:|
| Entities | 30,436 | 30,427 |
| Facts | 80,203 | 80,203 |
| Historical facts | 7,792 | 7,792 |
| Current relations | 39 | 39 |
| Cross-type collisions | 482 | 482 |
| Self-loop facts | 1,095 | 1,095 |
| Dangling-endpoint facts | 1,995 | 1,995 |
| Hollow entities | 2,854 | 2,854 |
| Logical raw keys | 241,340 | 241,331 |

Complete defect lists and relation sets are identical before/after. All preexisting global defects remain failures of the broader goal.

The complete raw delta is precisely 36 deletions and 27 additions: nine entities, claims, old edge facts, and reverse indexes removed; nine literal attribute facts and eighteen retirement markers added. No unrelated record changes. In particular, the six pending records and new cursor are preserved by retirement.

Independent assertions verify `Dst` becomes empty and `Value` becomes the exact display name, preserving the source, relation/raw relation, text, validity, timestamps, confidence, provenance, and any otherwise unmodeled JSON fields. A wrong ninth-group fingerprint rolls back the complete batch with no events. A forced postcondition observes all nine tentative conversions, rejects them, and rolls back all raw changes without events. Successful apply emits 27 events. Its synced/closed nonempty backup restores to exactly the pre-apply state. Retired exact entity/alias lookups are absent. The second preview writes nothing and emits no events.

## Remaining live prerequisite

Let ordinary processing settle and capture another fresh source with its queue truth checked from the backup itself. Recheck every candidate, source entity, touching current/history fact, alias/outside listing/hygiene fold, adjacency, replacement-key occupant, episode, and semantic companion against this evidence. A new unrelated episode may change the global baseline even if all nine fingerprints remain equal; recompute the full expected raw delta from that immediate baseline. Any affected input change reopens review and must not be handled by merely refreshing hashes. Use the reviewed deployed implementation, maintenance-locked backup/apply, and independent actual-live postconditions.

The parked migration-note resolver gap is a separate normal-write issue. This report neither grades a proposed fix nor permits changing code beneath these store checks without the corresponding review.

## Artifacts and verification

This directory contains the fresh independent replica, independently restored backup, complete before/after fact/entity JSON, semantic evidence, previews, raw-delta record, `mechanical-results.json`, `graph-audit.json`, `raw-payload-and-lookup.json`, `source-drift.json`, `pending-final-source.json`, `source-queue-truth.json`, and grader-written tests.

```text
go test ./internal/memory/store -run '^TestStatusNineIndependent$' -count=1 -v
go test ./internal/memory/resolve -run '^TestStatusNineIndependentGraph$' -count=1 -v
go test ./internal/memory/store -run '^TestStatusNineIndependentRawPayloadAndLookup$' -count=1 -v
go test ./internal/memory/store -run '^TestStatusNineFinalSourceDrift$' -count=1 -v
```

All final probes pass; the queue probe's passing assertion explicitly proves the source is not quiet. The first test must be reproduced in a fresh directory because it creates output and backup paths exclusively.
