# Independent bounded collision-drift review

Verdict: **the requested snapshots measure 483 → 484 collision pairs, not 482 → 484.** Exactly one pair was added and none removed. The earlier +1 predates the canonical-name deployment. Neither pair demonstrates normalized alias ownership theft. The new `migration0160` project is **semantically UNRESOLVED / mixed abstraction**, not a proven independent homonym and not an approved SQL-file duplicate. The earlier nine-fact/fingerprint-only retry gate is insufficient to establish current semantic closure; keep the pending retry held for that review. No merge, owner choice, repair manifest, retry, live write or full-goal PASS is authorized by this report.

## Pinned inputs and independent method

- Code independently archived from `62cf6e0d2d8db9d324da23114df837848e972c8f` into this directory. Review-only tests were added here, not in the shared repository.
- Before: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/mini-before-deploy.badger`, backup `memory-20260905T204303Z.badger`, 77,937,531 bytes, SHA256 `ef5d6bd0536ef59723a88ed99978b3897112d6844f55f7646b05ef90f1681685`.
- After: same source directory, `memory-20260905T210535Z.badger`, 78,004,674 bytes, SHA256 `ccd3dbd6fc4143458a6682481be5aba422d0ecc29f8f806c1db67c22dc9020b3`.
- Earlier explanatory baseline: `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T202741Z.badger`, SHA256 `b87def29a4334517299b55e9fc288de141f509b873c4afbdfd79bd101208ae06`.

All three were restored into my own new replicas under `results/`. Complete entity and all-fact inventories, including invalidated/history facts, were used to independently enumerate every referenced cross-type collision pair over every name, slug and alias. Enumeration agreed with the deployed code's collision count. Complete alias-claim maps were compared. Full stored source episodes, complete touching facts, full group metadata and all touching endpoint metadata were read. The new episode's 18 companion facts and eight contextual review-artifact identities were also inspected. Three independent tests passed: `TestIndependentCollisionDrift`, `TestIndependentPriorCollisionDrift`, and `TestIndependentEndpointEvidence`. These tests are evidence extraction, not a new resolver regression suite.

| Snapshot | Entities | All facts | Collision groups | Collision pairs |
|---|---:|---:|---:|---:|
| 20:43:03 predeployment | 30,453 | 80,258 | 421 | 483 |
| 21:05:35 latest | 30,467 | 80,299 | 422 | 484 |

Complete lists: [before groups](results/before-groups.json), [after groups](results/after-groups.json), [summary](results/summary.json). The new pair is `migration0160 | 0160-task-evidence-rulessql / migration0160`.

## New postdeployment pair: mixed file / repair identity

The existing tool `0160-task-evidence-rulessql` remains canonically named `db/migrations/0160_task_evidence_rules.sql`. Its description identifies the Docket SQL migration introduced by `a7253c0`, distinct from its table, API code and reservation task. It retains both Scry and Docket repository references and all six aliases. Its eight normalized spelling keys still route to it. Full metadata and all nine touching facts are unchanged between these two snapshots.

The new project `migration0160` has no aliases and only the Scry repository reference. Its description is “Store migration under review; after the fix, every approved spelling key returns nine facts while all 80,203 facts and repository associations are preserved.” Its three facts, all current and none invalidated, are:

1. `migration0160-sixth-record part_of migration0160`: “The sixth record is part of migration0160's identity, so the unchanged five-record fingerprints missed a new part of the identity.”
2. `migration0160 monitors [80,203 facts]`, raw relation `preserves`: “All 80,203 facts survive across the six-record replay and the actual live apply.”
3. `semantic-gate reviews migration0160`, raw relation `reviewed`: the fresh semantic gate compared independently derived baselines and all affected identity inputs against the fresh snapshot and deployed code.

All three derive from episode `8e4571c3e0a78c55f8c251086fce5a8a34b6324407f257aa1904809d9da41672`. Its occurrence/entity creation time is 19:58:55.321 UTC, but ingestion is **20:51:36.887417 UTC**. Thus creation time must not be mistaken for predeployment admission. The deployment receipt records 20:50:08 UTC. Admission occurred after that receipt's deployment time, while the underlying conversation happened before it. These records do not stamp the writing process's executable, model response production time or transaction time; exact binary attribution is not independently proven by snapshot timestamps alone.

The source episode explicitly describes a sixth record **for the migration file**, a corrected six-record manifest, replica replay and actual live merge. I also read the human-readable assistant messages in its exact local source span, `/Users/jeff/.codex/sessions/2026/09/05/rollout-2026-09-05T15-20-53-01a07304-3f7e-7cb1-8ad3-2ff8cf24ab70.jsonl#1764863-2201842` (436,979 bytes, SHA256 `37c0c2e95171c3014503555892cd0ada770dab2ca38f253d8a34cd4567326a43`). At 20:00:35 the reviewer calls the sixth record a record for the migration file. At 20:05:46, 20:10:17 and 20:11:52 the statements about preserving 80,203 facts refer to the six-record candidate, replay and actual apply, not to execution of the Docket SQL migration.

This is therefore not evidence of an unrelated migration or a clean independent homonym. However, merging the whole project into the SQL file would also assert that the SQL file itself preserves the entire Scry graph. The project description and preservation/review facts instead concern the memory-repair operation. The incoming sixth-record edge conflates the represented file with a memory record/repair-group abstraction. The 18 companion facts distinguish five/six-record candidates, manifests, actual apply, review report and the sixth record; they do not establish a single clean referent for this new project. Its endpoint `migration0160-sixth-record` is itself a concept describing the sixth identity record, not automatically another copy of the physical SQL file. **Whole-record ownership remains unresolved.** No endpoint disposition is inferred from sentence wording, name folding, record type, shared source, repository reference or fact count.

Full evidence: [new pair, all 12 touching facts and six complete source episodes](results/new-collision-evidence.json), [all 18 new-episode companion facts and contextual entities](results/migration0160-context.json), [all touching endpoint metadata and spelling owners](results/all-collision-endpoint-metadata.json). All 12 collision-member touching facts are current; the all-fact scan found zero invalidated facts touching these two members. Existing table/API relation wording is retained evidence, not newly endorsed semantics.

## Earlier +1: already present before deployment

The 20:27:41 source has 482 collision pairs. The additional pair present by 20:43:03 is `threemfreader | three-mf-reader / threemf-reader`.

- `three-mf-reader` (tool) explicitly names CADFormats `core/read/3mf.ts` in the workbench copy and earlier StartPart, required-extension and colour-fallback limitations. Three touching facts relate the workbench to this reader and the reader to its two issue records. Source episode `3c83cdc892f2480587325cffbad883f552f46461dd03ce4ee704a029f1142e6b` occurred 05:04:28.866 UTC and was ingested 05:47:50.740899 UTC.
- `threemf-reader` (concept) describes the gated lazy 3MF package/geometry reader with arbitrary StartPart, bounded ZIP sniff, placement/colour handling and export readbacks. Two touching facts say the same CADFormats workbench implements it and it uses the 1e18-domain precision contract. Its complete manual checkpoint episode `f479f897195be54946b8358ae65d00cffa9aede77d1eabe744b1aba4d5babd2c` identifies worktree `/tmp/cadformats-workbench-20260904`, branch `codex/file-workbench` and checkpoint `f98d5a6`. Occurrence is 20:36:23.049144 UTC and ingestion is **20:40:03.616981 UTC**, before both predeployment backup and 62cf deployment.

All five facts are current, with zero invalidated touching facts. These are closely related component descriptions, plausibly earlier/later states of the same reader, not evidence of an unrelated real-world homonym. This bounded snapshot audit does not prove component/version/worktree identity strongly enough to select a merge owner. Different scopes or versions require explicit examination before any repair. It cannot be attributed to the later canonical-name deployment. Full metadata, spellings, five facts and two full episodes: [predeployment evidence](results/predeployment-drift-evidence.json).

## Guard and retry conclusions

`store.Normalize` lowercases and converts spaces/underscores to hyphens, preserving existing hyphens. Hygiene `foldName` removes punctuation boundaries and joins singularized tokens. Consequently:

| Spelling | Normalized key | Owner | Hygiene fold |
|---|---|---|---|
| migration 0160 | migration-0160 | SQL survivor | migration0160 |
| migration0160 | migration0160 | new project | migration0160 |
| three-mf-reader | three-mf-reader | old tool | threemfreader |
| threemf-reader | threemf-reader | new concept | threemfreader |

The deployed canonical-name bypass checks normalized equality with the existing canonical name, not hygiene-fold equality with arbitrary aliases. The new project's spelling is neither the SQL canonical path nor its existing normalized alias key. This collision is outside that exact-name guard's coverage; changing it to merge on the broad fold would not be justified by this evidence.

Across the complete requested snapshots there are **zero new listed-spelling/index-owner anomalies** and **zero existing nonempty alias keys transferred to another nonempty owner**. Of 31 changed claims, 25 are newly occupied keys and six are deleted status keys. Those six deletions are visible in the inventory but their separate repair authorization is not graded here. This is a bounded failure to find an ownership-guard counterexample, not proof that all normal-write paths or all 484 collisions are safe.

The SQL survivor's unchanged nine facts and eight approved normalized keys establish only local stability. They do not dismiss the newly ingested mixed-abstraction project and review records. The semantic closure condition for the pending `ed50810b` retry must be reviewed afresh; this report supplies evidence but does not authorize that retry or any broader repair. No live/shared/provider writes were made, and no memory note was ingested during this audit.

