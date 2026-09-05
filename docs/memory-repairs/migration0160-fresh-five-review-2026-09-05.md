# FAIL — fresh pre-live semantic gate; five-record mechanics PASS

Do not apply the submitted five-record manifest to the live store. The fresh source contains a sixth explicit identity for this same migration file, with three current facts. Original member fingerprints are unchanged, but the semantic closure is not. No expanded manifest is approved by this report.

Exact reviewed inputs:

- Source: `/tmp/scry-migration0160-fresh-sep05.teYtyC/source.badger`, 77,827,733 bytes, SHA-256 `7261cfff0a7957d825855c430dbbc0f333517e806b7c47d8fc03a66fe8fc1548`. Lead identifies this as the Mini backup `memory-20260905T195752Z.badger` captured after the queue reached zero ready/backoff; I independently verify the local bytes, not the live queue.
- Submitted manifest: `/tmp/scry-migration0160-fresh-sep05.teYtyC/measurement/manifest-replica-only.json`, SHA-256 `7947eb322e33582fce214183a5a84e5c5639fb869442695d999ad31afd3e265d`.
- Implementation: independently archived `393eeec79f80d3b4becff276c4fcffd71fa68ac5` from the repository into this new temporary directory. Merge and hygiene code are unchanged from the earlier reviewed implementation; the inspected store diff adds `ErrFactConflict`.
- My independent pre-apply backup: `results/pre-apply.badger`, 77,827,717 bytes, SHA-256 `3effcb14ed8043d40014ed2041b8963123687245216df6829fadfeb507bd7b66`. It was synced/closed and restored into another new store; every restored raw key/value matches the pre-apply state. Backup file sizes need not match because restore/rebackup rewrites the backup stream; the complete logical bytes were checked.
- Actual Docket migration again hashes to `a806f8bf5eb9dd42e914b4384f0d77a0aad8edc2ea82755e55d0b49eaf535b61`; its sole file-history commit remains `a7253c09d85b43aded2ab6c6f623c11d8b8d508f`.

Proven omission:

`docket-migration-0160` has name `Docket migration 0160`, type `concept`, description explicitly identifying the “Docket migration file 0160” represented by the reviewed records, repo ref `/Users/jeff/workspace/context-stack/scry`, and creation/last-seen `2026-09-05T19:20:55.401Z`. Its normalized spelling is indexed to itself. It has exactly three current facts and no invalidated facts:

1. `part_of docket`: “Migration file 0160 belongs to the Docket project; five distinct scry memory records reference the same file, including a hollow record typed as a machine and a tool record holding two of its facts.”
2. `merged_into` attribute `five-record merge manifest`: the fact describes the five-record replica merge and unified six-fact lookup.
3. `merged_into` attribute `three-record merge manifest`: the fact describes the preceding three-record proposal.

All three cite episode `2d0820415d9d174268f59e2923269f891fa370c5da841065022ec90dc100c11f`, whose summary explicitly describes the independent replica verification, the initial FAIL, expanded bounded PASS, and absence of live approval. Source is `/Users/jeff/.codex/sessions/2026/09/05/rollout-2026-09-05T15-20-53-01a07304-3f7e-7cb1-8ad3-2ff8cf24ab70.jsonl#69120-1756173`, ingested `2026-09-05T15:45:28.523556-04:00`. This establishes that review ingestion produced the record. The historical merge assertions must not be interpreted as proof of live success; their provenance explicitly limits them to proposals/replicas.

This identity judgment does not rest on name similarity. The description names the file, its `part_of docket` assertion explicitly identifies the file as the subject, and the episode ties it to this exact five-record file repair. Its other two facts describe repair history of that same subject. In contrast, `migration-0160-duplicate-group` is explicitly the audit's flagged duplicate group and is the destination of `scry-graph-audit has_issue`; it remains a separate repair-group control. `five-record-regrade-reviewmd`, `three-record-verification-reviewmd`, and `full-five-semantic-closurejson` explicitly identify review artifacts and have artifact-location/role facts. They are separate controls, not migration-file members. Full metadata, facts, and source episodes for these distinctions are in `results/fresh-six-member-disproof.json`.

After applying the submitted five-record manifest in my replica, `Docket migration 0160` still resolves to `docket-migration-0160` and returns only these three facts. The canonical qualified SQL path resolves to the proposed survivor and returns its six facts. This is an experimentally demonstrated identity split remaining after the candidate repair.

Current semantic closure established on this snapshot: the previous five members plus `docket-migration-0160`, nine current facts, zero invalidated facts, and eight normalized keys. Full entity/index scans found no owner or normalized/hygiene-folded listing outside these six for their spellings. The new member brings a later last-seen timestamp and an existing Scry repo ref that a future explicit manifest must review and preserve or otherwise handle through an authorized operation. This report does not silently add it, rewrite its facts, or choose a new metadata policy.

Independent old/new comparison proves the original five full metadata objects, complete touching facts, all associated episodes, every relevant alias owner, and spelling listings are byte-equivalent as decoded JSON. Hence the repeated old hashes and unchanged manifest are real. A broad semantic scan finds the new file identity and separate review records beyond that pinned set. This is exactly why unchanged affected-member hashes alone cannot certify a newly regenerated semantic closure.

Bounded mechanics PASS for the submitted five-record manifest on the fresh source:

- Independently derived baseline: 80,203 facts = 72,411 current + 7,792 invalidated; 30,441 entities; 484 cross-type collisions. After replay: 80,203 facts; 30,437 entities; 482 collisions.
- Six facts touch the five-record group, all current. Four relocate endpoints and two survivor facts remain unchanged. All global current/history fact bytes, raw relation, value, confidence, times, and episode provenance are preserved.
- Complete raw key/value post-state equals my independently constructed expected map. Exactly 25 keys differ, total 241,340 → 241,338. No outside entity metadata, alias route, episode, cursor, fact, or other key changes.
- All four manifest retirees are absent, with no stale endpoint/alias-owner reference. All seven manifest keys and ten literal/normalized test spellings resolve to the survivor and return six facts. The eighth semantic key is the failure described above.
- The only hollow inventory change is removal of `dbmigrations0160-task-evidence-rulessql`: 2,855 → 2,854. Complete dangling-endpoint and self-loop inventories remain unchanged (994 distinct dangling endpoint slugs; 1,095 self-loop facts). No new group hollow, self-loop, or dangling endpoint is introduced.
- Backup restoration matches all raw pre-state bytes. Initial preview, intentionally rejected transactional postcondition, second preview, and rejected repeat apply all leave complete raw state unchanged. The second preview identifies exactly the four missing retirees and is not ready.

Evidence: `results/summary.json`, `results/preview.json`, `results/second-preview.json`, `results/independent-audit.json`, `results/full-five-semantic-closure.json`, and especially `results/fresh-six-member-disproof.json`. Independent tests passed: `TestIndependent0160`, `TestIndependent0160Audit`, `TestIndependent0160FullSemanticClosure`, and `TestIndependent0160FreshClosureEvidence`. Passing mechanical tests do not override the semantic omission.

All writes were confined to this new temporary archive and my own replicas. No live store, builder store, deployment, shared repository, model call, or remember operation was used. Final disposition: FAIL for this exact fresh-source five-record live candidate. Immediate live fingerprint checks and backup remain necessary prerequisites for any later approved manifest, but cannot repair this current closure failure.
