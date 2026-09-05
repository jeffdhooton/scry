# Independent four-hollow gate extension through 22:42:38 — 2026-09-05

**PASS: the exact four-record manifest remains ready for the reviewed atomic retirement workflow on deployed d1f0a958608389a385ac9a9f57ec1eb941015a10, against the complete 22:42:38 source.** Full semantic, raw-store, failure, backup, rejection-preservation and retrieval checks passed. This extends the preceding 22:21:40 review through the actual post-ChildScribe snapshot at 22:38:03 and the later manual ingestion snapshot at 22:42:38.

This is a preapply gate for these four only. It is not the separate actual-ChildScribe verdict, an actual-live four-retirement verdict, a broader graph-quality pass, or permission for other repairs/retries. The lead must preserve the exact reviewed manifest, obtain the actual nonempty locked backup, revalidate immediate inputs, and independently audit actual post-state. Later affected-input or semantic drift reopens this gate. No live writes, provider calls, retries, deployment, or shared-repository edits were performed by this reviewer. The earlier archived REVIEW.md is unchanged.

## Exact sources and artifact

Both extensions use fresh private archives of exact commit `d1f0a958608389a385ac9a9f57ec1eb941015a10`. The manifest copied as `reviewed-manifest.json` is independently SHA-256 **`8908910eae87bf9b1f288af86ccc4c2693a0173becb521ee13acd1d21ba9ffd5`**, identical to `docs/memory-repairs/hollow-four-batch-2026-09-05.json` and the previously reviewed candidate. All complete previews still match the same expectations, including the participant rejection fingerprints.

| Source | Bytes | Independently verified backup SHA-256 |
|---|---:|---|
| Actual post-ChildScribe `memory-20260905T223803Z.badger` | 73,529,181 | `aaadbdb7c65ca869385690476440d84895cb4d2afc1c93310291b2d28e012fd6` |
| Final source `memory-20260905T224238Z.badger` | 73,551,207 | `7e8afae2273606e6213c7f6200369ccdcbd2c1e414c57825d20ae2f671fe678f` |

Both inputs live under `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/`. Post-ChildScribe extension artifacts remain independently archived in `/tmp/scry-hollow-four-postchild-independent.N8LjOJ/`; final extension artifacts are in this report's directory. Exact deployed binary SHA, checked before final CLI use: `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.

For EACH source, the grader restored it into a fresh private replica, independently loaded the source directly with Badger before candidate Store.Open, and compared every raw key/value against candidate-open and candidate-backup-restored maps. No prefix filter was used. Exact map equality distinguishes absent keys from present empty values. Complete restoration and open preserve all keys, including all three rejection markers.

Final independent backup: `independent-preapply.badger`, **73,551,199 bytes**, SHA-256 **`f73bb3cd7c21001d2ee07361a5dde3e9472d4219a353261faac9ce3d0cba3566`**. The exclusive production operation synchronizes and closes it before mutation; independent restoration equals the entire source map. Final logical hashes (sorted JSON of the complete raw map, separately from physical backup hashes) are:

- Source/restored backup: `6826fefb50103670eebf2dfb3921dedf004dd771c8cf41320aef04c1a219acf1`.
- Independently predicted/actual retired replica: `d28650504ec495c511b4c38a276c16166fac16af69e448512ecfaf13c94d42d6`.

## Full intervening closure

The 22:21:40 → 22:38:03 comparison is a complete **391-key** raw delta. It contains 87 new fact keys, four modified fact records, 42 new entities, 12 modified entities, nine new episodes, 55 adjacency additions, 56 alias-claim additions/three removals, three rejection additions, 38 new/one modified attestations, four new/four modified cursors, four metadata changes, two new pending records, and 59 new/eight modified value-evidence records. No fact key or original episode is removed. Complete exact before/after bytes are retained in `complete-222140-to-223803-raw-delta.json`; decoded changes are in `222140-to-223803-analysis.json`.

I read all 91 changed fact assertions, all 54 changed/new entity metadata records, all nine new episode summaries, and the four store metadata changes. The four modified facts comprise three provenance additions (Cockpit→tmux and two Scry→Mini assertions) and one historical invalidation of the existing Scry suite-scores attribute at `2026-09-05T22:00:23.752Z`; that attribute's sentence, value and provenance remain. This invalidation predates the proposed four-retirement transaction and is retained exactly. No blanket endorsement of that unrelated ingestion decision follows.

The complete four-candidate semantic evidence changes only in contextual `codex-reviewer.last_seen` and `statelicenselookup.last_seen`. This conclusion was reached after the entire delta inspection, not presumed from matching candidate fingerprints. Six new facts touch contextual identities, so I expanded to their **three full source episodes and all 29 current/historical companion facts**. Exact original source segments were independently distilled and read in full:

- `531f82c7...`, Claude source 2964038–4212903: Jeff proposes a fuller Cockpit view using herdr and tmux. This is separate from the Hermes task-result remnant.
- `b71e8010...`, Codex source 10267496–10664974: proposed filing and licensing tools, including guided license checks and renewal planning. It concerns future StateLicenseLookup features, not a new identity for the earlier clean Git check.
- `b95561b6...`, Codex source 4540191–5045832: focused CADFormats owner-session review. It explains the reviewer's new last-seen time and does not redefine the unavailable route worker's terminal outcome.

Full derived source records are the three `source-<episode-id>.json` files copied here. The original four domain source transcripts were read in the preceding independent review; their full distilled texts and that report are copied into this directory. The four candidate records, four original source episodes, all 66 original companion facts, surviving assertions, and all other original contextual fields remain unchanged through 22:38:03.

The 22:38:03 → 22:42:38 comparison is a complete **71-key** raw delta: 16 new facts, **two modified existing facts**, seven new entities, two entity last-seen updates, one new episode, ten adjacency additions, eight alias additions, three attestations, 21 value-evidence additions, and one extraction metadata update. No deletion, rejection change, queue change, cursor change, or retirement-marker change occurs. Exact delta: `complete-223803-to-224238-raw-delta.json`; full decoded inventory: `complete-intervening-change-analysis.json`.

The one full stored manual episode `4f5f4353...` reports the CADFormats private workbench release. I read the entire episode, all sixteen new fact records and both modified historical records, and all nine changed/new entity snapshots. The two modifications invalidate CADFormats `dormant-not-deployed` and `review-passed` assertions at the manual episode's occurrence time, retaining all original content and provenance. New facts describe its release, private/public flags, verification, retained files, and credentials handoff location; no credential file was accessed. None names or references a selected remnant, changes its original episode/companion context, or claims its spelling. The four semantic evidence structures are **completely identical** from 22:38:03 to 22:42:38, not merely equal fingerprints.

The final complete raw scans cover every fact payload endpoint, encoded fact key, raw adjacency endpoint, entity spelling listing, and alias claim. Each of the four still has zero current/historical edges, no hidden raw edge/index reference, no outside normalized or hygiene-folded listing, and one self-owned claim. The semantic decisions therefore stand: ten-task completion, measured 2,312-test result, terminal unavailable-worker status, and clean Git check are outcomes; their projects, tools, worker, review blockers, and assertions remain.

## Three durable rejection records

Exactly these source records are present and retained byte-for-byte:

- `ar:childscribe-laravel:driver-core-worktree`
- `ar:childscribe-laravel:envoyer`
- `ar:childscribe-laravel:office-dashboard`

Each retains its complete literal alias, reason and plan `274e1e580ec0ac70464546990f4653f9956432071ffacde89d5d0487125f543f`. The three-record artifact SHA is `75fa48ba80fdfb53fb0a44e5278ba6e5bd5a862b6e48be533d93aa0fa2b42f9d`. They survive source raw-load, candidate open, actual four-retirement replay, independent backup restoration, and closing/reopening both source and retired replicas. Ordinary stale PutEntity alias writes and ClaimAlias attempts fail with ErrAliasRejected for each pair, before and after retirement/reopen, with zero events and complete raw-map equality. The four-retirement operation does not synthesize, delete, alter, inherit, or supersede any alias rejection.

Old marker-unaware binaries must not write to this marker-bearing store. This gate neither approves such rollback nor restores a pre-marker backup over intervening facts.

## Final exact transaction and checks

The final actual replica map equals the independently predicted map with exactly **16 changed keys**: four `en:` and four `al:` deletions, four `rs:` and four `rt:` additions containing exact reviewed IDs/reasons. Every other key/value is identical—including all facts, historical timestamps, provenance, episodes, rejection decisions, pending payloads, cursors, metadata, attestations and indexes.

| Measure | Final source | Retired replica |
|---|---:|---:|
| Facts | 80,602 | 80,602 |
| Historical facts | 7,820 | 7,820 |
| Entities | 30,611 | 30,607 |
| Hollow entities | 2,854 | 2,850 |
| Complete cross-type collision pairs | 484 | 484 |
| Self-loop facts | 1,096 | 1,096 |
| Dangling-endpoint facts | 1,995 | 1,995 |
| Current relations | 39 | 39 |
| Rejection records | 3 | 3 |
| Raw logical keys | 242,798 | 242,798 |

The complete hollow list is exactly the old list minus the four candidates. Complete collision-pair, dangling, loop, and conservative listing/claim-defect inventories are identical; the latter contains 4,433 entries. All four bare result removals are explicitly reviewed; no rule classifies the remaining hollows as statuses. Existing broader defects remain open. New facts referencing existing `no-defects-found` and `disproved` nodes were observed in the earlier interval; those references are not evidence those nodes were newly created, and no cleanup approval for them follows.

Whole-batch rollback passes for wrong fourth-group fact fingerprint, wrong fourth-group rejection fingerprint, and a deliberately failed postcondition after all four deletions. Failed operations produce zero events and zero raw writes. Successful backup-coupled apply produces four delete events and no fact event. Original entities, renamed same-slug entities, and fresh-slug entities using retired names cannot be recreated; all attempts are refused before and after reopen without raw writes.

Final queue classification: **0 ready / 0 backoff / 10 parked** at 22:42:39 UTC. No retry was run. Exact deployed second CLI preview returns dry_run=true, applied=0, refused=4 because all four are absent; final raw-map verification after CLI and recall proves no writes.

All eight before/after local recall probes return their unchanged retained assertion at rank **1**, largest response **12,506 bytes**, below 24 KB. The unavailable-worker query uses its original historical interval at `2026-08-20T00:00:00.000000001Z`; its invalidation remains untouched. Full results, target records and queries are archived. These probes do not replace the five benchmark suites or wider held-out goal grading.

Passing final tests: TestHollowFourIndependent, TestHollowD1RawSource, TestHollowD1RejectionFingerprint, TestHollowPostChildExtension, TestHollowCompleteDefects, TestHollowFourIndependentGraph, TestHollowFourRetainedAssertionRecall, and TestHollowFourCLINoWrite; separately the exact CLI dry run. The preceding post-ChildScribe archive also passed its replica/rejection/recall suites and TestHollowExtensionSources. Original review harnesses were inspected/reused, with new independent full-delta and marker-preservation checks; builder helpers were not used as proof.

All code, manifests, backups, raw maps through restored stores, complete deltas/inventories, semantic evidence, source extracts, defect lists, probes and artifact hashes are retained in the private directories identified above. Fresh reproduction of the exclusive-backup mutation harness requires a new output directory. The lead may proceed only with the exact reviewed four-record workflow and immediate locked revalidation, followed by separate actual-live audit.
