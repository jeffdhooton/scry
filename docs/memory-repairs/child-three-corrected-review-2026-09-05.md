# Corrected three-alias manifest — independent supplemental verdict

**Bounded PASS** for exact corrected manifest SHA-256 `4af1797522bd675027798d729057f1c80172d4e4ed1d50856040323795e54c4a` on source backup SHA-256 `c063d83125a81f096e319c94286958da8f29a388e1e29b73460180a37be4d397`, using candidate `d1f0a958608389a385ac9a9f57ec1eb941015a10`. This approves only the semantic three-pair disposition and its exact isolated replica result. **Live apply remains held** pending the separate deployment/benchmark attribution and immediate live closure gates described below.

The original `REVIEW.md` and its BLOCK verdict for the old manifest remain retained without edits (SHA-256 `a6147f95c660b9aba395b85c2069698c74ee9dd46203967679119d1f4534f88e`). Its full semantic findings, source review, counterclaim investigation, preservation scope, and exclusions apply here. The old test is retained as `original-child-three-independent-test.go.txt`. This supplement applies only to the new exact manifest at `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/child-three-reason-corrected-replica/manifest-replica-only.json`.

## Resolution of the sole manifest blocker

The literal diff is exactly the intended replacement in Envoyer's reason: “No existing entity lists this spelling” → “No outside existing entity lists this spelling,” plus the resulting expected plan hash change. All drop owners, aliases, ordering, no-rehome dispositions, other reasons, source/owner/fact/listing/claim/rejection fingerprints remain unchanged. The corrected assertion is true: Child is the sole listing and claim owner for each selected normalized spelling, and none has an outside existing listing.

The new independently reproduced plan is `274e1e580ec0ac70464546990f4653f9956432071ffacde89d5d0487125f543f`. Expected was reconstructed from the fresh complete restore before calling the candidate preview. The resulting independent preview is byte-for-byte equal to the root's corrected preview file.

The exact three persisted negative pairs remain justified:

- `(childscribe-laravel, envoyer)`: source concerns Cell Saviors' Envoyer-style production release; it does not establish Child identity, actual Envoyer-service existence, or a positive rehome to Forge/Cell Saviors.
- `(childscribe-laravel, office-dashboard)`: source concerns Docket responsive/coherence/scale-ticket work. Overlapping outside office entities remain distinct and no rightful existing listing is stranded.
- `(childscribe-laravel, driver-core-worktree)`: source explicitly concerns the Docket checkout. The existing driver-core package does not become identical to that worktree by this decision.

The unresolved current `childscribe-laravel uses graph-mutations` counterclaim that incorrectly names ChildScribe in driver-core Task 4 remains raw unchanged. Complete source episode `0c1f60ea...` shows the Docket driver-core checkout and does not identify ChildScribe; preservation is not source correction. Child's polluted survtest description, repository metadata, remaining aliases, all other facts, and historical evidence remain unresolved where previously unresolved. No broader correctness is implied.

## Fresh exact corrected-manifest verification

A new complete independent restore was created under `evidence-corrected/replica`; the previous replica was not reused. The independent seven-key prediction was rebuilt using the corrected manifest and compared against every actual raw key/value after apply:

1. The same one entity record changes only three literal aliases, 43 → 40.
2. The same three `al:` claims are removed.
3. The same three owner-specific `ar:` records are added. All three carry the corrected plan hash; only Envoyer's reason text differs from the old proposal.

The corrected and old proposal post-states differ at exactly the three `ar:` keys, as independently checked. Entity and alias-index bytes are identical between the two proposals. Actual equals independently predicted full raw state, with 241,815 keys before and after.

Passed again on this exact corrected proposal: actual durable backup restored to every original raw key/value; close/reopen preserves state; normalized spelling resolution is absent; direct and atomic ordinary entity reintroduction is rejected without writes; actual resolver admission rejects each literal/normalized spelling from two distinct new episode IDs without changing any raw key; second preview is not ready and second apply refuses without writes. Corrected test command: `go test ./internal/memory/store -run '^TestChildThreeIndependent$' -v -count=1`, PASS in 8.374 seconds. Reviewer test file SHA-256: `d1bd058cb2975219b2f16d6d9b1f54e1705b6c57308142af1dc9ed4fe0b7c243`.

The complete before/after defect lists remain exactly equal to the original review, with no new dangling fact endpoints, missing provenance, adjacency defects, or listing/claim defects. This preserves existing graph defects; it does not declare the graph clean.

All 80,338 facts, 30,480 entities before transformation, 9,332 episodes, all claims, the 2,105 Child touching rows, all 61 source companions, and all three matching facts compare byte-for-byte with the original review exports. All six actual original transcript source byte ranges (the five named episodes plus the Task 4 counterclaim) were freshly reread and their SHA-256 values remain equal to the original source manifests. There is no source drift hidden behind the unchanged database snapshot.

## Corrected closure evidence

- Corrected manifest file SHA-256: `4af1797522bd675027798d729057f1c80172d4e4ed1d50856040323795e54c4a`.
- Corrected plan: `274e1e580ec0ac70464546990f4653f9956432071ffacde89d5d0487125f543f`.
- Before raw closure: `bd796edc3120e711899a6b8d99bd1a3a03390dea95cb961bfd641ca8950be80b` (unchanged).
- Independently predicted = actual corrected raw closure: `363fc0191d79ea9da9906ef42345627429c6cea871920cd2a947f43f8ff3d2bd`.
- Corrected replica pre-apply backup SHA-256: `eadf1cf098458cdca03a9bd2dd8ca2ec52c98352aa45f669f16c284c8709c42f` (restores the exact original logical raw state).
- Full `evidence-corrected/closure.json` file SHA-256: `dee5a341e86581c64fa8c41262bec391dcdc46268aaf3fec1aaf2c5eacf493cc`.
- Corrected `preview.json` SHA-256: `b5937afb4efd1c881b12126b0d37f7be68195ef1fe174ec03945465ae33f9f2f`.
- Corrected `admission-no-write.json` SHA-256: `184887b582841d293039ff5f8580443d8a421d21231e5cb57e7e3db6987c0fb3`.

The unchanged complete evidence and source closures are recorded in the original report and `evidence-verified/source-closure.json` plus `counterclaim-source-closure.json`. Full corrected key-hash maps, expected and actual states, previews, no-write evidence, facts, entities, claims, episodes and defect lists are retained under `evidence-corrected/`.

## Remaining live boundary

The parent reports d1f0a958 deployed to both writers after a separate gate, but this reviewer has not graded those deployments or the actual deployment pre/post raw state. The parent also reports heldout 52 → 51 with a Cell Saviors tailnet-address miss, while the other four suites remain 29/7/45/47. Attribution is pending; this review neither calls that regression resolved nor approves proceeding around it.

Live alias apply remains conditional on the separate deployment/backup/raw-state gate, resolution of that benchmark attribution under the goal's requirements, an immediate full source/graph closure plus code/manifest stability recheck, and independent actual alias pre/post grading plus the five required suites. There is no approval for old 49-removal rejection backfill, other alias changes, source guessing, fact moves, rehomes, or whole-goal completion. Root remains sole live writer; this regrade made no live/provider/shared writes.
