# Independent fresh three-alias gate — 2026-09-05

Verdict: **bounded PASS** for the exact three-drop manifest SHA-256 `4ac2ac02f2d855de6e9afa6233f97b9bb1a37e8241b3387bb8dfd13d32a6b652` on the complete 22:15:11 UTC snapshot SHA-256 `d020a4599447d238c2bbcb5cff8031e399f3ae9e69d891c5b6fc75bb7e9c8d49`, using deployed candidate commit `d1f0a958608389a385ac9a9f57ec1eb941015a10`. The same manifest independently passed the preceding 22:10:12 snapshot in this review. No candidate violation was proved for these three dispositions or their exact replica changes.

Immediate freshness extension: I independently loaded `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/mini-child-immediate-222140.badger` directly into Badger, verified its exact SHA-256 `35a6e1b8f333fa772ea25537bba2c1979addf25d55eced7dd8f2878558f30d45`, and compared every one of its 242,380 raw key/value hashes to the reviewed 22:15 source. The maps are exactly equal. This extends the same bounded disposition/replica result through 22:21:40 UTC without assuming that matching Expected fields imply complete source equality. Reproducer `TestImmediateFreshIndependent` passed; evidence is `evidence-final/immediate-freshness.json`.

This is a semantic/replica preapply gate, not whole-goal acceptance or permission to broaden the repair. Root remains the only live writer. Immediately before any apply, root must verify a fresh nonempty backup, the exact deployed marker-aware writer hashes and manifest, complete relevant semantic closure including new source companions/outside listings, and the full Expected fingerprints. Any relevant drift requires review; unchanged Expected alone is insufficient. Actual live before/after raw equality to the seven-key prediction, exact lookup, second no-write attempt, and five benchmark suites still require independent post-apply grading. The original recall floors and global graph defects remain open.

## Isolation and pinned inputs

All reviewer code and replicas are under `/tmp/scry-child-three-fresh-independent.G20a7Q`. The source came from an independent `git archive d1f0a958608389a385ac9a9f57ec1eb941015a10`, archive SHA-256 `9a0a6c7c4321c2e405f4c53be3e600d30287210986911171369e700b7a5dd516`. Reviewer helpers were added only after extraction. The directory with those helpers is not called pristine. The shared repository was read only; its untracked workflow assessment remains untouched. No live writes, retries, deployments, provider calls, or shared repository edits occurred.

The exact candidate binary supplied at `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry` hashes to `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`. Separate deployment reports are prerequisites, not work this grader claims to have performed. The root reports both deployed writers have that hash.

The exact manifest is `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/child-three-fresh-221012/manifest-replica-only.json`. Its plan is `274e1e580ec0ac70464546990f4653f9956432071ffacde89d5d0487125f543f`; Child entity fingerprint `b7fa1df19b48d1713368d546bda53d32ea6e748292092936fc10b55f26026c59`; complete touching-fact fingerprint `5d879da1900f8e146b5e98858efa44e77445c0ad21b5b21b20b130badb1913fa`. Both snapshots independently reproduce every Expected field, using separately implemented normalization, JSON hashing, full current/history touching enumeration, outside listings, claim-presence, and rejection closure.

| Snapshot | Exact file SHA-256 | Facts / entities / episodes / raw keys |
|---|---|---|
| `mini-child-fresh-221012.badger` | `4a33e4d8b7e2786d3c9a936cf8bf71f7ca61603ecb8f3454b6f26a8dcb946382` | 80,473 / 30,551 / 9,342 / 242,290 |
| `mini-child-final-221511.badger` | `d020a4599447d238c2bbcb5cff8031e399f3ae9e69d891c5b6fc75bb7e9c8d49` | 80,499 / 30,562 / 9,344 / 242,380 |

Both files are in `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/`. The final source independently has zero alias-rejection records and eight pending rows; PendingCounts at snapshot time is 0 ready, 0 backoff, 8 parked.

## Semantic decision and complete refreshed closure

Only these owner-specific negative identity pairs are approved: Child/`envoyer`, Child/`office-dashboard`, and Child/`driver-core-worktree`. There is no positive rehome, fact move, metadata correction, global alias ban, or inferred Envoyer/Forge/package ownership.

**Envoyer.** Full original Cell Saviors production episode `ea3bd9d6...` records the Envoyer-style releases, the backup path under `/home/forge/cellsaviors.com`, transport work, and the Cell Saviors incident document. Its Child endpoint in the stored production-deployment fact is unsupported by the source. That fact remains unchanged and unresolved. The exact corrected reason says “No outside existing entity lists this spelling,” which is true; Child itself is the sole exact normalized listing and claim owner. The source does not justify inventing an actual Envoyer-service identity or assigning the spelling to Forge or Cell Saviors.

**Office dashboard.** Full episodes `0d82d3a5...`, `7612a8d8...`, and `872b2788...` concern Docket office responsiveness, coherence/Haulyard branding, and the scale-ticket loop. The original companion set includes overtly Docket sentences whose endpoints were folded onto Child; those are not independent proof of Child identity. Overlapping office identities remain distinct and unchanged; all outside names, slugs, aliases and actual claims were enumerated. None lists the exact normalized spelling. Dropping Child's generic surface alias therefore strands no existing outside rightful listing and does not select a positive owner among the office entities.

**Driver-core worktree.** Full episode `38ee8140...` explicitly creates the Docket checkout from `a5458ab`, then builds the package skeleton there. The existing `driver-core` entity describes a platform-neutral package, with aliases `@docket/driver-core` and `driver-core package`; it does not list the selected worktree spelling. Package identity and checkout identity are not equated by this review.

The review preserved and compared every original 2,105 Child-touching row against the fresh store, finding all byte-equivalent in decoded content. Fresh closure is 2,106 rows: 1,581 current and 525 historical. All 61 original source companions are unchanged. Every fresh fact was scanned across serialized content using independently reproduced normalization for exact targets, and a broader Envoyer/office-dashboard/driver-core search returned 128 current/history rows, all examined as assertion text. All entity slugs/names/aliases were scanned, as were all 51,901 final claims. The full inventories and complete hash maps remain available, not just selected matches.

The fresh 21:44 → 22:10 change contains 139 added/modified fact records and one old timestamp key absent (net +135), 84 changed/new entity records, and 10 added episodes. Every changed fact assertion was read, all broad relevant records were inspected, and every new ownership-relevant source was read with its companions. There are now five exact target-matching facts instead of three. The two additions are:

- `independent-alias-replica-review tests envoyer-identity`, source `173bdd22...`: full source is a bounded technical replica review. “Separate Envoyer identity remains representable” describes a synthetic representability check. The new `envoyer-identity` concept lists only “separate Envoyer identity”; neither its metadata nor source establishes an actual Envoyer-service owner or an exact `envoyer` listing.
- `unalias-rejection-gap causes childscribe-laravel`, source `ae62e000...`: full source explicitly reproduces the old unalias gap on an isolated replica and says there were no live changes. Its “Two normal Apply calls restore the Envoyer → ChildScribe alias mapping” statement describes that prior failed prevention experiment, not semantic ownership or evidence that this candidate regrew live. This is the sole new Child-touching record, and remains unchanged.

The 22:10 → 22:15 change contains 26 added facts and two modified existing facts, 19 changed/new entities (net +11), two added manual episodes, and no removed fact keys. I read all 28 changed assertions and both complete manual source summaries: Scry deployment/rollback/benchmark note `64e17281...` and an unrelated CADFormats private-preview authorization note `97f40e9e...`. Neither changes any selected ownership evidence, listings, claims, matching facts, source episodes, or Child-touching record. There was no retry by this reviewer. These fresh global additions are included in the second complete restore and its preservation proof.

## Counterclaims and unresolved data

I extended the prior review beyond its single Task 4 counterclaim. The current `childscribe-laravel uses graph-mutations` sentence naming Child is still contradicted by its full Docket checkout source `0c1f60ea...`. Additional preserved Child records include the invalidated driver-core self-loop, branch 19-ahead/70-behind status, coverage-registration status, four-parity-gaps status, and the current boundary-guard endpoint. Full additional source episodes `b340183c...`, `7aa22f6f...`, `ea7643e4...`, `2dc526ea...`, and `a01a3e95...` describe the Docket checkout, package extraction, dispatch, routing, and webhooks. These statements cannot establish that ChildScribe owns the worktree. All their current/history bytes remain unchanged for separate source correction.

The current `tsconfigdriver-corejson related_to childscribe-laravel` assertion came from a different source, `f08b1478...`, whose exact original span concerns the Scribe chapter branch and a generic untracked root `tsconfig.json`; it does not mention driver-core. This is not evidence for the selected worktree alias. The current candidate distiller did **not** reproduce that old episode ID from either its recorded starting offset or the whole file. I therefore read the complete original span directly, including all user/assistant text, retained its hash, and explicitly mark exact-ID reproduction false. This is a limitation of additional unrelated source reconstruction, not concealed source approval or a basis for repairing that fact.

The complete source closure comprises 14 exact original byte spans and 150 current/history same-episode companions. Thirteen episode IDs and complete distilled texts were reproduced exactly; the fourteenth is the directly read Scribe span described above. All 150 companion assertion texts were read. Raw spans were freshly reread and compared to saved bytes. Source projection excludes tool-result/thinking payloads; this is not a claim of manually reading all raw tool payloads. The original five sources and Task 4 counterclaim retain their prior exact source hashes.

Child's polluted “Default branch of survtest” description, contaminated repository references, and all other remaining aliases remain unresolved where appropriate and untouched. Existing source pollution is not cured or semantically endorsed by these three drops.

## Independent exact replica result

For each snapshot, I first loaded the backup directly with Badger and captured all raw key/value bytes. Opening with candidate Store and separately restoring through candidate Restore both equal that direct raw state, proving the candidate did not silently rewrite the supplied source before the comparison began.

Before apply I independently constructed the whole predicted state. Exactly seven keys change:

1. Replace `en:childscribe-laravel`, removing only the three literal alias entries (43 → 40); every other field remains identical.
2. Delete `al:envoyer`, `al:office-dashboard`, `al:driver-core-worktree`.
3. Add `ar:childscribe-laravel:envoyer`, `ar:childscribe-laravel:office-dashboard`, `ar:childscribe-laravel:driver-core-worktree`, each carrying the exact manifest alias, reason and plan.

Actual complete raw state equals the prediction. No other key changes. All 80,499 facts, all invalidated history, all source/provenance records, every entity field outside the three literals, every unrelated alias index, all 10,812 positive attestation records, adjacency, cursors, queues, rejection/retirement metadata and all other prefixes are preserved byte-for-byte. The actual nonempty durable pre-apply backup was restored and equaled the complete original raw state. Closing and reopening preserved the exact post-state.

Literal/case/underscore/repeated-space lookup variants no longer route to Child; corresponding owner-specific rejection is durable. Stale complete entity writes, direct and atomic alias reintroduction, explicit ClaimAlias and rehome attempts all refuse without raw changes. Both merge directions refuse marker inheritance in preview; actual MergeEntities calls also refuse without any raw write. The exact second preview is not ready; second apply refuses and changes no raw key.

Actual resolver admission from two distinct new episode IDs refuses each literal/normalized spelling without writing any key. A separate complete post-repair clone then ran **two real normal resolver Apply calls**, proposing all three aliases. Both episodes were durably stored; zero facts were added, invalidated, merged or rejected; no aliases regrew; every original raw key stayed identical except the predicted Child last_seen field. The only added keys are the two synthetic episode records. In that disposable probe clone, a distinct synthetic Envoyer tool remains representable while Child's rejection persists, adding only `en:envoyer` and `al:envoyer`. This is a technical representability probe, never a live identity decision.

Final test: `go test ./internal/memory/store -run '^TestChildThreeIndependentFinal$' -v -count=1`, PASS, 9.488 s including package result. Earlier 22:10 test also passed. Additional executable helpers and their output are retained under `cmd/independent-*` and `evidence-final/`.

Complete original/predicted/actual defect lists are equal: 2,441 dangling fact endpoints; 0 missing provenance episodes; 0 missing/extra adjacency; 3,867 listing spellings with missing claims; 505 listing/claim owner mismatches; 0 dangling claim owners; 28 claims unlisted by owner; 462 normalized spellings with multiple listing owners. These are pre-existing final-source defects, not graph-cleanliness claims. Additional independently calculated lists also stay equal: 2,854 entities with no current OR historical fact, 2,972 with no current fact, 1,096 current/history self-loops (90 current), and 27 exact-normalized multi-owner spellings spanning types. The zero-fact lists are structural inventories, not semantic declarations that each record is a hollow defect; the exact-normalized type list is not the broader production hygiene heuristic.

## Artifacts and hashes

All final evidence is in `/tmp/scry-child-three-fresh-independent.G20a7Q/evidence-final/`; 22:10 evidence remains in `evidence/`. Full inventories, complete before/predicted/actual raw key-hash maps, complete defect lists, previews, and durable backups are retained. `analyze.mjs`, `analyze-final.mjs` and `closure-extra.mjs` reproduce semantic drift and supplementary closures.

- Final before raw closure: `55f2bbc28e2262f36d1804dbd2decdcc8dbb4fcc82e0936fef64e7c2a0f02d1e`.
- Final independently predicted = actual raw closure: `8893c0a39555a736e73b0bd65e57eba51c729119e2c88aeedea6e11f80aae241`.
- Final actual pre-apply backup SHA-256: `f704c2f39f039efd617eefa2deae4d2bdd8edfebf8a59e889f2c61e688a913bc`.
- Final `closure.json` SHA-256: `fac5287c05ad5b2d98578b91b2045926c9d400fa32c6b41f1e774a9da177beb1`.
- Complete source manifest SHA-256: `adf9ca8e95897333db4f3e132d148bbc73b222640bd8cbb56db1259adb2a8166`.
- 150-companion file SHA-256: `658dffae750225e03dd51025890edadf859e6110178f395fec8b424e0b89db3c`.
- Reviewer final test source SHA-256: `7b44710ecd54fdc36bdbc49cdf1df502471fed47a97a9612133c2653ebf4addc`.
- Actual-two-Apply evidence SHA-256: `a0dd7127618fae5d5149ebbaa85d8777468d8e759010e340faa6e2fd909ef96d`.
- Distinct-identity evidence SHA-256: `dbaa7dfe6f072a6771848fefc8ade6ef5d0bd718a34f2ddd3600f34a0664e0aa`.

Raw closures hash the JSON map of per-key SHA-256 values using the independent Go harness; they must not be compared directly with another harness's length-framed raw-stream hashes.

**Rollback boundary:** the reviewed final source has zero live rejection markers. Applying this manifest would create the first three. An older marker-unaware binary alone is then an unsafe downgrade: it could ignore the durable decisions and regrow aliases. Keep marker-aware code, or use a separately reviewed full restore that reconciles all intervening writes. Retaining an old binary is not sufficient rollback safety after the first marker.

No approval is given here for the previous 49-drop backfill, stops-table or any other alias disposition, positive rehome, source correction, entity merge, queue retry, broader hygiene apply, or whole-goal completion. The reported five-suite baseline remains 51/29/7/45/47 with original recall floors unmet; this review does not relabel those as passing the original goal.
