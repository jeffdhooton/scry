# Independent Qwen repair gate

Reviewer: fresh-context `qwen_repair_review_sep05`. Verdict: PASS within scope,
2026-09-05. Original full report and independent harness:
`/tmp/scry-qwen-independent.XNcrkD/REVIEW.md`.

The reviewer independently restored the fresh post-deployment snapshot and
checked code `af77a6a5841ec58970887408480bf1b8452d84ce` in a separate archive.
Exact reviewed inputs (SHA-256):

- `qwen-entity-merge-2026-09-05.json`:
  `cb354c9c605af36f9d394606ae705b91261f9b8fcb39ab148779d54c5346d9c2`
- `qwen-alias-rehome-2026-09-05.json`:
  `60700ac8422b84351f228cbca3a5256b3d2e3bf98c56e7e730aea70474d9c5c1`
- `qwen-repair-evidence-2026-09-05.json`:
  `d4e65c3fc43c21884243ad967ad58eb680870470478a922fa9d07bb8bd780c32`
- Source `memory-20260905T181707Z.badger`, 76,857,630 bytes:
  `853d134fd52d4379d959c35181c5969d20ffa962c7df706a96cca8d4f9d26021`

The evidence JSON deliberately retains its pre-review status and exact bytes;
this report supplies the subsequent verdict without changing the reviewed hash.

## Verified on the independent replica

- Rehome preserves all 79,547 full fact records. Merge changes only explicit
  loser-to-survivor endpoints. All 19 touching Q5 facts retain content,
  timestamps, confidence, provenance and invalidation, including one historical
  fact. Q8 remains a distinct identity.
- All four approved spellings resolve to Q5 and return its 19 total facts.
  Q5, Qwen3 and Qwen3.8 disappear globally from listings and alias indexes.
  The entire alias index matches the independently calculated expected map.
- Only the reviewed Q5 records and Q8's explicitly removed aliases change.
  Neither surviving model is hollow. No new dangling endpoint or hollow appears.
- Collision counts match prediction: 492 before rehome, 491 after rehome,
  489 after merge.
- CLI default merge and daemon dry-run operations are immutable. Both daemon
  applies create nonempty backups that restore exact pre-apply entity, fact
  and alias-index states.
- Repeat merge dry-run is immutable and refuses the absent loser. This is safe
  refusal, not an idempotent-success response or global hygiene convergence.

## Immediate live execution conditions

Keep the queue stable. Before the one-row rehome, verify both entity hashes
from the evidence and that the exact Q5 alias-index owner is still Q8.
Entity hashes alone cannot detect a separate alias claim. After rehome,
compare every full merge fingerprint with the reviewed manifest before apply.
Each apply requires its own actual nonempty backup. Any drift requires a fresh
review; never force the manifest.

This is not overall graph certification: the replica still has 2,854 hollow
entities, 2,441 dangling endpoints, and 489 collisions. Remaining Q8 aliases,
historical fact correctness, recall floors and the full goal remain outstanding.
