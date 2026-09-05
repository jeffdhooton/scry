# Independent drained-source extension — Child `stops table`

**Bounded PASS for the 23:08:54 source with the separately refreshed candidate manifest `9b34e94a…`. The old `94484a6d…` manifest is BLOCKED by real entity/fact drift and must not be applied.** This report covers only the one owner-specific alias rejection, not broader graph repair or live freshness after this source.

The separately archived four-record actual PASS was read in full. It establishes its own earlier operation; this review does not expand that operation. Root supplied a newer 23:14:10 source during this review, which requires a separate extension before live apply.

## Inputs and isolation

Fresh private archive of deployed commit `d1f0a958608389a385ac9a9f57ec1eb941015a10`: `/tmp/stops-drained-independent.uoAjxG`. I inspected and reused my prior independent raw-map, semantic, refusal and benchmark helpers, with explicit source/count/path updates. No shared-repository or live mutation, provider, queue retry, deployment, installation, or additional durable note occurred. The user-owned workflow assessment remains untouched. Later shared comment/test-only changes are not the executable under review.

Independently verified files under `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3`:

| File | SHA-256 |
|---|---|
| `memory-20260905T230854Z.badger`, 73,724,105 bytes | `188e5bd97b24221c61ac86b4e605f6f05dc0d8184d39be8a464ece4230c785ce` |
| preceding `memory-20260905T225312Z.badger` | `efa7466e022fb6a67918876aba8960eff9ae4aa1b1027eeeac52dd41ead9058a` |
| `stops-table-fresh-230854/manifest-replica-only.json` | `9b34e94a083b90a8eacd48b1e82786680f118e06f7d0d3630dccdc10e28a74b3` |
| deployed binary used for offline suites | `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553` |

Both complete sources were independently loaded with direct Badger Load before candidate Store.Open. Every raw key/value was copied without prefix filtering. Candidate opening and production Restore reproduce the complete input map exactly. Root supplied transfer/live-origin evidence; this independent proof concerns the complete supplied files.

Fresh counts: 80,710 facts, 72,885 current; 30,658 entities; 9,360 episodes; 52,017 alias claims; 55,679 adjacency; 10,879 attestations; 3,128 cursors; ten pending records; five metadata records; three existing rejection keys; 19 each retired-slug/retirement keys; 706 value-evidence records.

## Full semantic drift and original sources

Complete 22:53:12 to 23:08:54 raw delta is 395 added, 23 changed, zero removed keys. Fact-object delta is 113 new/changed versus five old versions: 108 added facts, three provenance extensions, and two old status invalidations. Entity-object delta is 58 new/changed versus seven old versions, net 51 added; all seven existing entity changes are last_seen updates. Six new episodes comprise five Codex reviews and the single combined manual repair note `5c5ce0b2…`. Every changed old/new fact, entity and episode object was read, as were all 111 facts sharing the six fresh episodes. Queue/cursor/meta/value-evidence drift is retained in the full raw maps; no prior rejection or retirement record changed.

I reconstructed and read all five complete new Codex projections, hashing their exact recorded raw spans. Four reproduce from whole-file distillation. `af08fef6…` does not reproduce with whole-file chunk boundaries; resuming from its exact original SourceRef start, 1,695,526, reproduces the exact stored ID and its complete 2,117-byte text using unchanged production distillation. The initial failed reconstruction directory is retained. No minimum-turn threshold was lowered, source ID fabricated, or source file edited. This is the original human/assistant projection with tool-call breadcrumbs; omitted tool-result/reasoning payloads are hashed as part of the raw span, not claimed manually read.

Complete source refs, byte spans, raw/text hashes and reconstruction method: `evidence/new-sources-complete/closure.json`, SHA `e735273eaa141749e8dde5c374286b352e996cc09b14a53f59dc63ba57ed3a0e`. The other complete source is the manual note's stored text, retained and read in `evidence/drift-ep.json`.

The three CADFormats review sources concern private-release login, asset admission, validation and response headers. They introduce no Child, Docket table or competing `stops table` ownership. The two Scry sources are reviews of prior alias/source work. Their assertions are review narration, not independent Docket-domain evidence.

All twelve original domain SourceRef spans and retained full projected texts were rehashed, and their current episode metadata remains exactly equal. All 164 original companion facts remain byte-equal, hash `d76ea10656fb14757c0166aa179716611b818e72a6e9255cb064d48955bcdc59`. Thus the substantive negative continues to rest on the original Docket weight-ticket FK source, six-task plan and final worker report, plus the migration-file versus table versus directory distinction. It does not rest on newly ingested claims that the earlier reviewer passed it.

Fresh separator-variant matching rows grow from five to 14; matching-source companion rows grow from 58 to 76. All nine additions are source-review narration from `568d25e8…`. Complete broad stop/weight-ticket/0043/0112 entity discovery grows from 174 to 177: only three review artifact/claim/decision entities are new. Existing selected metadata changes are Child and Scry last_seen updates. No outside entity lists the exact normalized spelling; its sole listing and claim remain Child.

The new `stops-table-claim` concept is explicitly about a spelling/claim in this graph, not an original database-table entity. Its description's word “formerly” inaccurately implies the attachment already ended, although this preapply snapshot still contains the alias/claim and no fourth rejection. The `stops-table-claim / decided / Child` fact records the review's owner-specific semantic rejection; it does not prove a live mutation. These temporal and review-versus-domain distinctions remain disclosed and preserved. Neither that concept, the review decision, Docket, the migration, nor `stops/` is approved as a positive owner.

## Exact Child drift and attribution of the combined note

Child's sole metadata change is last_seen: `2026-09-05T20:19:06.469Z` → `2026-09-05T19:06:44.403571-04:00`. Its name, type, description, all 40 aliases and their order, repository references and created_at remain unchanged.

Five touching facts are added:

1. Child related_to `childscribe-alias-repair`, raw relation `aliases_repaired_by`, from the combined note. This is the correct memory-repair target context, not a database-table ownership claim.
2. Child status `child-edges-2106`, from the stops review's historical snapshot count.
3. `envoyer-statement` documents Child, from the earlier review's explicit discussion of misleading isolated-failure evidence.
4. `stops-table-claim` decided Child, raw relation `rejected_for`, from the stops review.
5. `three-alias-drop-manifest` targets Child, from the three-alias review.

One prior Child status row, `486-collisions`, gains invalid_at `2026-09-05T22:34:54.303Z`, exactly the new stops review occurrence time. Its source episode remains the original `717d965d…`; it is not rewritten or removed. The new edge-count status does not logically refute an earlier collision count. This is an unresolved status-supersession artifact of ingestion, not evidence that either count is a present domain identity or an approval to correct that row. It predates the combined note's occurrence and is not attributed to that note.

Child now has 2,111 current/history rows: 1,585 current and 526 historical. Fresh typed Expected fact hash is `e476cf64dd83d44ee352f0f95bbd4600612ae68f74c0f8473a4fb019584aef4b`; raw-object array hash is `34238253dd61d76c840244f52357e96941af89c17e435a49b1c5ae938e24972d`. These serialize differently and are not interchangeable.

The combined note has 24 same-episode companion facts. I found no new wrong Child/table endpoint assignment in those 24: the notes describe the two separate memory operations, removed spellings as attribute values, distinct backup paths, preserved fact counts, review receipts and old-binary limitation. The lone new Child relation has the explicit repair context above. It also updates last_seen and supersedes the memory-quality goal's `unfinished` status with `unresolved`, retaining its old row historically. Its existing `childscribe-alias-repair` endpoint still has an older deduplication description; the note did not create or cure that inaccurate prior description. This is not blanket approval of each generic normalized relation such as `produces` for raw `removed_alias`.

All original misattached Docket companion facts, row-copy/no-copy contradictions, out-of-owned-set test disclosure and later migration metadata remain unresolved and preserved. No fact correction, reattachment, invalidation or deletion is approved by this alias-only gate.

## Refreshed exact candidate proof

Before any repair, my independently reconstructed Expected rejected the old manifest. I then verified root's separate refreshed manifest against the full semantic drift above. Its only changes are the two stale Expected fingerprints. Independent reconstruction gives Child entity `c7293a75b15d988dfd82a421349e8d80399352142e8264a94bcbc859398c115b` and fresh touching facts `e476cf64…`; plan remains `ef7a0970016251e74b41e2db69c5d5c0d7b4b03c49e7a24a0b640df66906d35e`, prior rejection closure remains `1a40bf4d7af133a80d2109949d4f21a45194ec5871a74630aeffac238dfe392a`, with unchanged actual claim/presence and sole Child listing.

The fresh candidate privately applies with a synced nonempty 73,724,121-byte backup; its direct restore exactly equals the fresh pre-map. A separately constructed whole-map prediction equals the actual private post-map. Exactly three keys differ: Child aliases 40 → 39 with every other field preserved, deletion of `al:stops-table`, addition of `ar:childscribe-laravel:stops-table` with the exact literal, why and plan. Every other byte survives, including all 80,710 facts and new status invalidations, every source/queue/cursor/meta value, the three earlier rejection keys and all 19/19 retirement markers.

Complete compact JSON maps from keys to base64 values hash before `54358fb6c25c2de1f6c8e19ce44fc31c821954490d91ac9e7f71bab8218f88ac` and predicted/after `baa95b99bcec81710f5bedc12e055bd2912e8e8a47fcf4b8f63edd7152cf8448`. Complete map equality, not hashes alone, underlies the result.

Candidate Open/Restore, pre/post backup restoration, reopen, second preview/backed apply no-write refusal, stale Child/claim atomic rollback, rehome refusal, both actual merge directions, two real normal Apply variants and distinct synthetic representability all pass on private copies. The old manifest was also run through production Preview and backed apply against this fresh source: both refuse on changed fingerprints, preserving the whole map. No live apply was run.

## Structural and retrieval controls

Complete pre/post structural inventories are identical: 2,441 missing-endpoint occurrences; no missing source episodes or missing/extra adjacency; 3,867 missing listing claims; 505 wrong-owner listing claims; zero dangling claim owners; 28 unlisted claims; 462 normalized multi-owner spellings; 27 exact-normalized multi-type spellings; 2,850 entities with no facts; 2,968 without current facts; 1,096 self-loops. Complete rows are retained; no outside listing is stranded. These are structural definitions, not semantic hollow retirement or broader folded-hygiene approval.

Ten pinned offline suite runs preserve scores, missed-question sets and mean answer ranks. Before/after hits are 51/62, 29/66, 7/7, 45/50, 47/50. All over-cap counts are zero. Maximum payloads before → after are respectively 12,109 → 12,110; 13,360 → 13,360; 10,107 → 10,107; 11,530 → 11,530; 11,530 → 11,530. The original acceptance floors remain unmet.

Final complete raw rereads passed after the suites. An initial final-check attempt overlapped the still-running benchmark and encountered Badger's directory lock; it made no database change. I waited for completion and reran that check successfully. An early reporting attempt likewise found the still-unwritten benchmark aggregate; final reporting succeeded on the completed outputs. These were reviewer scheduling errors, not repair failures.

Evidence: `evidence/` retains the initially blocked source review and all fresh original projections. `candidate-evidence/` retains the refreshed direct loads, complete maps, independent Expected, source checks, exact delta, defect lists, old-manifest refusal, merge refusals, backups and all suite outputs. `candidate-evidence/file-hashes.json` pins generated evidence. Full semantic delta file SHA is `7032dd00d32265bb23600f30982985761a317476d03c59b08d35919c012dd9b6`; original twelve-source check file SHA remains `23de70618779d5ab15a40ad2380108716409d146a735e51e84fcb8b497b887c3`; before/after complete defect files both hash `d8b6d9ffa6deddf685ebdefb8979f9d291ba35e5f8299456b609b79e157ca2ec`.

This PASS approves no broad 49/33 backfill, other alias, positive rehome, old 41-group operation, heuristic fact cleanup or goal-completion claim. Old marker-unaware binaries remain unsafe writers. Any live apply requires the newer-source extension, immediate exact input check, atomic backup and a separate actual pre/post audit.
