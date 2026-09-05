# Independent exact three-alias review — 2026-09-05

Verdict: **BLOCK the exact manifest `d9c44edc2a7e4a70e9a2be35eb6f124b9afdedd85017f23987bfd81d86a49e1a` for one inaccurate durable reason sentence.** The three owner-specific removals are supported by the reviewed source evidence, and the exact complete-replica mechanics pass. This is not a block on the prevention code.

The `envoyer` reason states: “No existing entity lists this spelling.” That is false in the reviewed pre-state: `childscribe-laravel` lists `envoyer`, and it owns the corresponding claim. The independently derived expected listing is `["childscribe-laravel"]`. The intended, supported statement is “No outside existing entity lists this spelling.” The other two reasons already make that distinction. Because the sentence is persisted verbatim as durable negative identity evidence, correct it before live use and regenerate the plan/manifest/Expected; this review does not approve silently substituting a new manifest.

## Isolation and pinned inputs

The reviewer independently restored the complete supplied backup into `/tmp/scry-child-three-independent.ztce0e/evidence-verified/replica`. No live store, provider, or shared source file was written. The initial code came directly from `git archive d1f0a958608389a385ac9a9f57ec1eb941015a10`, not the root's artifact code directory. Archive SHA-256: `9a0a6c7c4321c2e405f4c53be3e600d30287210986911171369e700b7a5dd516`. Reviewer-only test and source/admission helpers were added after extraction, all under the reviewer's directory; the resulting helper-bearing directory is not described as pristine.

- Source: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/mini-pre-deploy.badger`, SHA-256 `c063d83125a81f096e319c94286958da8f29a388e1e29b73460180a37be4d397`.
- Exact manifest: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/child-three-fresh-replica/manifest-replica-only.json`, SHA-256 `d9c44edc2a7e4a70e9a2be35eb6f124b9afdedd85017f23987bfd81d86a49e1a`.
- Reviewed plan: `185e8c888d98c9c12873d72a30680aad814639e0069cb2d0ad53cc98f79b5ba5`.
- Graph restored: 241,815 raw keys, 30,480 entities, 80,338 facts, 9,332 episodes.

## Semantic disproof work

All five named source episodes were recovered from the actual local transcript files by their exact `source_ref` byte spans. The exact episode IDs were independently reproduced by the candidate distiller. The full distilled episode text was read, not just the database summaries; raw source spans were retained and hashed. These texts intentionally reflect the ingestion projection, which represents tool calls by labels and excludes tool-result/thinking payloads. The review does not claim manual reading of every byte of the much larger raw JSONL tool payloads.

All 61 same-episode companion fact texts were read. All 80,338 fresh facts were scanned across serialized content under independently reproduced normalization for the three spellings; exactly three facts match, with no additional matching fact outside that set. Every one of the 2,105 Child current/history touching rows was included in the preserved evidence closure and full raw equality comparison (1,580 current, 525 invalidated). Broader Child text searches examined ownership/supporting evidence and possible counterclaims beyond the literal three spellings. This is a bounded alias decision, not semantic approval of all 2,105 existing statements.

`envoyer`: `ea3bd9d6...` is a Cell Saviors production remediation episode. Its full text places backup output under `/home/forge/cellsaviors.com`, sends a transport test to the configured Cell Saviors sender inbox, and records results in the Cell Saviors incident document. The 61-companion set includes Cell Saviors deployment, production backup, mail transport, and enrichment-batch facts. The stored `cellsaviors uses childscribe-laravel` fact has an Envoyer-style deployment sentence; its Child destination is unsupported by this source. The drop and negative `(childscribe-laravel, envoyer)` pair are supported. No actual Envoyer service identity is established, and no rehome to Forge or a Cell Saviors entity is inferred. The misattached fact remains exactly unchanged and unresolved.

`office dashboard`: complete episodes `0d82d3a5...`, `7612a8d8...`, and `872b2788...` cover office responsive measurements/implementation, Docket/Haulyard coherence, and the Docket scale-ticket loop. The source paths and repository context are Docket; the coherence source explicitly discusses Docket staging, dispatch trucks, container asset tags, and user-requested Haulyard branding. The matching `office uses playwright` fact carries all three episodes. Outside metadata contains overlapping `office`, `OfficeApp`, `web-office-app`, and `haulyard-office` entities; none lists the exact normalized spelling. The negative Child pair is supported; selection of a positive owner among those entities is neither necessary nor justified here.

`driver-core worktree`: complete episode `38ee8140...` explicitly creates `.claude/worktrees/driver-core` from Docket commit `a5458ab` and develops the driver-core extraction skeleton. Branch/skeleton companions agree with that context. The existing `driver-core` entity describes a platform-neutral package and lists `@docket/driver-core` and `driver-core package`; those do not establish that the package and a checkout are identical. The negative Child pair is supported; no package/worktree merge or rehome is inferred.

An additional counterclaim was found outside the literal matches: current `childscribe-laravel uses graph-mutations` says “Task 4 is part of the driver-core extraction work in the childscribe-laravel project.” Its complete source episode `0c1f60ea06193fb84be196db795f49b584ce409f7c00cec0d519d4263ae208d2` was recovered and read. The source CWD is `/Users/jeff/workspace/docket/.claude/worktrees/driver-core`; its 15,100-byte full distilled text concerns Docket Task 4 compiler-boundary and outbox extraction reviews and does not identify ChildScribe. An episode companion explicitly places Jeff's ownership in driver-core and broader Docket. This counterclaim is additional extraction pollution, not valid ownership evidence. It remains raw unchanged for separate source correction.

Child's own preserved ownership evidence includes the Laravel/Herd app and `apps/childscribe-laravel -> ~/Herd/childscribe` relationship, membership in ChildScribe/Scribe, existing backend/admin functions, and Forge deployment evidence. These are preserved, along with their current/history state and provenance. The entity's current description, “Default branch of survtest; unborn because no commits exist,” is visibly polluted and is **UNRESOLVED**, preserved exactly. Remaining aliases and repository references are not implicitly approved.

Every entity's slug, name, and aliases were independently normalized for outside listing closure, and every claim was inspected. Each selected normalized spelling has exactly one existing listing, Child, and the claim points to Child. Removing it strands no outside existing listing. Relevant complete entity metadata is retained in `entities.json`; all entity metadata except the three literal list entries is unchanged.

## Independent replica predictions and actual checks

The test independently reproduced normalization, serialized hashes, participant/touching sets, listings, claim presence, empty rejection subset, and the Expected object before comparing with the production preview. It did not use production `hashJSON` or repair-analysis helpers to generate its predicted result.

Exactly seven keys were predicted before applying:

- Replace `en:childscribe-laravel`, removing only the three selected entries, aliases 43 → 40, all other fields identical.
- Delete `al:envoyer`, `al:office-dashboard`, and `al:driver-core-worktree`.
- Add `ar:childscribe-laravel:envoyer`, `ar:childscribe-laravel:office-dashboard`, and `ar:childscribe-laravel:driver-core-worktree`, each containing the exact manifest alias, reason and reviewed plan.

Actual post-apply key/value bytes equal that independently constructed full 241,815-key map. Therefore all facts, history, text, provenance, source episodes, adjacency, cursors, pending evidence, unrelated aliases, metadata, and pre-existing marker families remain raw unchanged. No other key was added, removed, or modified. This is stronger than matching counts.

The actual durable pre-apply backup independently restores to the complete original raw state. Backup SHA-256: `16323fd4df59e565907d9886325f7e293b4d63a9b660ab000e300cada170e8ce`. Closing and reopening preserves the exact post-state. Literal, uppercase, underscore, and repeated-space forms no longer resolve through the removed claims and remain rejected for Child. Direct and atomic ordinary entity writes attempting reintroduction fail with `ErrAliasRejected`, without raw writes. Actual resolver admission for all three aliases and normalized variants, from two distinct new episode IDs, refuses them with “explicit reviewed rejection for this entity”; the complete post-state hash map is still identical afterwards. A second exact-manifest preview is not ready; a second apply refuses; both leave the database raw unchanged.

The first scratch test attempt found a reviewer-harness distinction between nil and empty byte slices while cloning zero-length adjacency values. The harness was corrected to preserve the byte slices exactly and a new isolated evidence directory was used; it was not a candidate failure. The corrected test passes in 12.028 seconds. Its source is `internal/memory/store/child_three_independent_test.go`; two reviewer helpers live under `cmd/independent-sources` and `cmd/independent-admission`.

Complete independently generated defect lists before, predicted, and actual are exactly equal:

| Defect | Count |
|---|---:|
| Dangling fact endpoints | 2,441 |
| Missing episode references | 0 |
| Missing / extra adjacency | 0 / 0 |
| Missing alias claims for listings | 3,862 |
| Listing routed to another owner | 505 |
| Dangling claim owner | 0 |
| Claim not listed by its owner | 28 |
| Normalized spellings with multiple listing owners | 462 |

These are pre-existing defects, not claims that the graph is clean.

## Closures and delivery conditions

`evidence-verified/closure.json` contains the full object-closure hashes and counts. Its file SHA-256 is `4445b11350ceb5e06a31ca7873338208327fecab22350ad5c345072c8444efc6`.

- Before raw closure: `bd796edc3120e711899a6b8d99bd1a3a03390dea95cb961bfd641ca8950be80b`.
- Predicted = actual raw closure: `4995c0f4f4b83a866fe72e2a29a437b502a3fbb3b3694d8513db87a9be8c0eb4`.
- Child touching closure: `c1630252a9d94e60d62f2ec3446374f84b647894aea70320d9257b52283237f4`.
- All 61 source-companion closure: `e53ea042ff0b0f33fb8a9f7a41132cab13e3f207892cb8b1bf0d80cd586a8aec`.
- Three matching-fact closure: `fba4f6211b3751794fb3883bd9ab553ee99aa62f59c6135b3f5abbd97cbcd7c2`.
- Defect-list closure: `0fab5e1127c3965f5795377603fe94fd9d288844100bb7670b6477f15d1e7104`.
- Exact five-source span/text hash manifest: `evidence-verified/source-closure.json`, file SHA-256 `ce3e607e119301bc47401dca0e601ece0c8cad30dc5c597686661061b37e818c`.
- Counterclaim source hash manifest: `evidence-verified/counterclaim-source-closure.json`, file SHA-256 `833dd5cf15a1c55f9da784c58c6f8aebf4c0b1315aa559565191910855d76aa4`.

After correcting and independently rechecking the exact durable reason/manifest, any bounded live approval remains conditional on d1f0a958 deployment to both writers with verified backup, an immediate full closure/code/manifest stability recheck, and independent actual pre/post grading plus the five required suites. This review is not deployment approval, permission to backfill the prior 49 removals, approval of other aliases, permission to move facts or repair sources by guessing, or whole-goal approval.
