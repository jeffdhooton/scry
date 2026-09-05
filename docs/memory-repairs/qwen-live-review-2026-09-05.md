# Independent actual-live Qwen repair postcheck — PASS within scope

I restored all three supplied actual-live Badger backups into separate private store directories under /tmp/scry-qwen-independent.XNcrkD/live, using the independent af77a6a5841ec58970887408480bf1b8452d84ce source archive. No live/shared-store writes, remember calls, daemon launches, or model calls occurred.

Verified SHA-256 inputs:

- Before rehome, memory-20260905T182725Z.badger: a207f13b7b9abf2be0955d96b77ac5c6528eae1342c12254cd9eccd2f2200ea2
- Before merge, memory-20260905T182753Z.badger: e4b0ea88788dcc7d47fb5a162d27273b85b87841a4e6f6a48e1f8a64f6a57dfd
- After merge, memory-20260905T182812Z.badger: 24621f12ab33ad0d7b32f96238f1b7854430abf41da9c3ff8b362dd8c33ae851
- Actual receipt docs/memory-repairs/qwen-live-receipt-2026-09-05.json: 7a42237fb6dacb9662edac195c765dcdeecd6b5412b43e70e29910750aa03fd0

The earlier approved merge/rehome manifest hashes remain cb354c9c605af36f9d394606ae705b91261f9b8fcb39ab148779d54c5346d9c2 and 60700ac8422b84351f228cbca3a5256b3d2e3bf98c56e7e730aea70474d9c5c1 respectively.

Independent findings:

- All three backups restore successfully. Their states provide working rollback points before each mutation.
- Actual pre-rehome state contains 30,176 entities, 79,560 facts, and 9,282 episodes. The extra unrelated ingestion is present before both repairs, rather than appearing as an unexplained repair delta.
- Rehome changes exactly the Q8 listing of the exact Q5 name and its alias-index ownership from Q8 to Q5. Every fact, episode, other entity field and other alias-index key is unchanged.
- The actual middle snapshot has exactly the reviewed entity/fact/alias fingerprints. My independently generated full merge preview is JSON-equivalent to the corresponding live receipt preview.
- Final state has 30,175 entities and all 79,560 full fact records. The only fact changes are explicit substitution of the retired Q5 slug with the Q5 survivor at affected endpoints. Text, raw relation, validity, invalidation, confidence, provenance and every other field are preserved. All 9,282 episode records are identical across all three backups.
- Nineteen facts touch the Q5 pair: 18 current, one invalidated. Every approved old spelling resolves to the survivor and retrieves all 19. The loser entity is absent and no alias-index key targets it.
- Q8 remains distinct with 130 facts. Its metadata changes only by removal of the reviewed exact-Q5 and bare-Q5 aliases. The one Q8-originating historical edge to the loser retains Q8 as its source and now targets the Q5 survivor.
- Q5, Qwen3 and Qwen3.8 are globally absent from both entity spellings and alias indexes. The entire final alias-index map equals the independently calculated expected map. Survivor metadata exactly equals the approved manifest; unrelated entities are unchanged.
- Collision counts are independently 492 before rehome, 491 before merge and 489 after merge. The merge's observed -2 matches its prediction.
- No new hollow entity or dangling endpoint appears. The existing 2,854 hollows and 2,441 dangling endpoints remain unchanged.
- Repeating default merge dry-run against the restored actual post-state applies zero groups and refuses the absent loser. This is safe non-mutation, not an idempotent success response.

Evidence: three-snapshot-verification.json, verification.json, independent-merge-preview.json, second-dry-run.json and the before/post-rehome/after JSON snapshots in this directory. The independent harness is /tmp/scry-qwen-independent.XNcrkD/cmd/qwen-independent/main.go.

Verdict: the actual scoped live repair obeys the reviewed identity, alias, historical-preservation and backup requirements. No repair-specific blocker found. This does not certify recall floors, remaining Q8 aliases, global graph cleanliness, or the complete overall goal.
