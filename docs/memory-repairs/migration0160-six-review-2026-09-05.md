# Bounded PASS — exact six-record fresh-source live candidate

The corrected six-record manifest resolves the semantic omission proved in `/tmp/scry-migration0160-fresh-independent.WXlaaX/REVIEW.md`. That five-record fresh-source candidate remains FAIL. This PASS covers only the exact source, manifest, and implementation below as a live candidate, contingent on a verified nonempty immediate pre-apply backup and rechecking the reviewed live inputs before writing. Any new semantic member or other closure drift requires regeneration and review; unchanged hashes for existing members alone were proved insufficient in the preceding gate. This is not a live apply receipt or a graphwide completion verdict.

Pinned inputs:

- Source `/tmp/scry-migration0160-fresh-sep05.teYtyC/source.badger`, 77,827,733 bytes, SHA-256 `7261cfff0a7957d825855c430dbbc0f333517e806b7c47d8fc03a66fe8fc1548`.
- Manifest `/tmp/scry-migration0160-fresh-sep05.teYtyC/six-member-preserved-refs/manifest-replica-only.json`, SHA-256 `547b324b4a0c83bcec6a345b46a8e2f3e80be5b2178b002d125d2aaf8f29e2c3`.
- Implementation independently extracted into this new temporary archive: `393eeec79f80d3b4becff276c4fcffd71fa68ac5`.
- My independent backup `results/pre-apply.badger`, 77,827,717 bytes, SHA-256 `f7d6f9fbe4ec55d78d2f251e04b6223616c2ae30c4081d4d5f0395cf98bf8c96`.
- Actual SQL file, verified during the immediately preceding fresh-source gate: SHA-256 `a806f8bf5eb9dd42e914b4384f0d77a0aad8edc2ea82755e55d0b49eaf535b61`, sole file-history commit `a7253c09d85b43aded2ab6c6f623c11d8b8d508f`.

Independent work used another fresh archive, another source restore, and another backup restore. Builder replicas and prior reviewer replicas were not mutated. No live/shared store, shared repository edit, deploy, model call, or remember operation was used.

Semantic PASS. The complete six-member source metadata set, nine touching facts, and index ownership exactly match the closure independently proved in the preceding fresh-source disproof. Members are the five original explicit SQL-file records plus `docket-migration-0160`. All nine touching facts are current; no invalidated fact touches any member. Complete scanning of every entity spelling and alias-index owner establishes eight normalized keys with no outside normalized or hygiene-folded owner/listing.

The new concept is the file identity because its description explicitly identifies the Docket migration file, its `part_of docket` fact makes that file the subject, and its episode identifies this exact file-repair history. Its two other facts describe earlier replica/proposal merges. The episode's express “no live apply was approved” context remains preserved. These historical assertions are not upgraded to claims that a live operation happened. The separate `migration-0160-duplicate-group` is the audit's flagged repair group; the review Markdown/JSON records are artifacts. They remain outside the merge, as do the table `task-evidence-rules`, API modules, reservation coordination/range, and encompassing implementation task.

Metadata PASS. Canonical name is `db/migrations/0160_task_evidence_rules.sql`, with artifact-supported `tool` type. Every old basename, name, alias, slug, and routing claim transfers; no alias is dropped. Earliest creation `2026-08-19T02:49:16.995Z` and latest observation `2026-09-05T19:20:55.401Z` are retained. Existing `/Users/jeff/workspace/context-stack/scry` association from review provenance is preserved alongside the independently verified Docket artifact repository. I independently tried a preview omitting the Scry ref: it refused with `metadata drops repo ref /Users/jeff/workspace/context-stack/scry` and complete raw-state equality. No guard weakening is involved. Description remains the explicitly reviewed file identity synthesis. Survivor/type choices follow the artifact evidence, not fact counts.

Mechanics PASS:

- Fresh source baseline independently measured: 80,203 facts (72,411 current and 7,792 invalidated), 30,441 entities, 484 cross-type collisions. Replica after: 80,203 facts, 30,436 entities, 482 collisions.
- Nine group facts are preserved; seven relocate endpoints and two existing survivor facts are unchanged. Every global fact's text, value, raw relation, validity, confidence, and episode provenance is preserved.
- Complete raw key/value post-state matches an independently constructed expected map, covering every forward fact, reverse index, entity, alias route, episode, cursor, and other key. Exactly 35 keys differ; total keys 241,340 → 241,337. No unrelated metadata or index changes.
- All five retired entity keys are absent. No fact endpoint or alias owner still refers to them. Every one of the eight normalized keys and all twelve tested literal/normalized spellings resolve to `0160-task-evidence-rulessql` and return nine facts, including the qualified path, old hollow slug, and `Docket migration 0160`.
- Complete hollow inventory changes only by removal of the full-path machine husk, 2,855 → 2,854. Complete dangling-endpoint and self-loop inventories are unchanged: 994 distinct dangling endpoint slugs and 1,095 self-loop facts remain globally. No affected fact is dangling or a self-loop.
- Nonempty backup was synced/closed before apply. Restoring it into another new store reproduced every pre-apply raw key/value. Logical restore equality, rather than incidental backup-stream byte identity, is the proof.
- Initial preview writes nothing. Intentional transactional postcondition rejection writes nothing. Second preview identifies exactly the five absent retirees and is not ready; repeat apply rejects. Both preserve complete raw state.

Evidence: `results/summary.json`, `results/preview.json`, `results/second-preview.json`, `results/independent-audit.json`, `results/full-six-semantic-closure.json`, and `results/metadata-review.json`. The preceding disproof contains the full additional review-artifact/control evidence and new record's source episode. Tests passed: `TestIndependent0160`, `TestIndependent0160Audit`, `TestIndependent0160FullSemanticClosure`, and `TestIndependent0160MetadataReview`, using review code confined to this temporary archive.

Disposition: PASS for the exact six-record candidate on the pinned fresh source, with the immediate backup/live-input conditions stated above. The submitted five-record predecessor remains rejected, and no future expanded manifest or changed source is automatically approved.
