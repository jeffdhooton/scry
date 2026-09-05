# Independent bounded gate: FAIL

Reviewed exact commit `8c2a05df4a7f5dc69b88b3d8422d13b93b52b4ae` against parent `5d4ff50`, on 2026-09-05. This verdict covers the whole-relation identity guard and its normal Apply consequences only. It does not certify the full memory goal or authorize deployment, historical rewrites, or live repairs.

## Blocking finding: newly rerouted claims lose their sentence and raw relation on a current fallback triple

**P1 — `internal/memory/resolve/vocab.go:259` routes rejected identity compounds into `related_to`, exposing them to unconditional triple-based merging at `internal/memory/resolve/resolve.go:511–521`. `mergeFact` at line 611 takes neither incoming sentence nor raw relation. It retains the existing sentence/raw relation, appends the new episode ID, and uses the maximum confidence. The new evidence can therefore disappear while its episode is marked ingested.**

This contradicts the new decision's promise that unknown identity-like compounds retain their sentence and raw relation on the fact, and the goal's fact-preservation requirement. Appending an episode identifier is not preservation of the incoming claim: the old text is now attributed to an episode that supplied different text. The unsafe identity edge is prevented, but this particular normal-write case loses evidence.

### Differential fixture

1. Precreate distinct `origin-console` (project) and `destination-console` (tool).
2. Apply episode `earlier-context` at `2026-09-05T00:00:00Z`: `origin-console related_to destination-console`, text `The origin and destination were discussed at the same meeting.`, confidence 0.7.
3. Apply episode `routing-proof` one hour later: relation `aliases_index_to`, same endpoints, text `The release-monitor alias index already resolves to the distinct destination tool.`, confidence 0.91.

Parent `5d4ff50` adds a separate `same_as` fact carrying the new sentence, raw relation, time, confidence, and episode (`FactsAdded=1`, `FactsMerged=0`). That identity typing is wrong, which this patch correctly aims to prevent.

Candidate `8c2a05d` instead returns `FactsAdded=0`, `FactsMerged=1`. Only the old meeting sentence remains; `RawRelation` remains empty; its confidence becomes 0.91; its episodes become `[earlier-context routing-proof]`. The incoming routing sentence and raw relation are absent from **all** stored facts.

The old general fallback merge behavior predates this commit, but this is a concrete newly affected input: parent preserves the incoming claim as a separate fact; candidate discards it. Retaining the unsafe identity mapping is not the proposed fix. Preserve distinct evidence when multiple raw claims converge on the same canonical triple, with explicit handling of exact time/key collisions and genuine restatements.

Reproductions:

- `new/internal/memory/resolve/independent_sep05_test.go`: `TestIndependentApplyRoutingCollisionKeepsSentence` fails.
- `old/internal/memory/resolve/independent_collision_test.go`: identical fixture passes on parent.
- `parent-collision.log` contains exact parent facts and stats.
- `independent-tests-final.log` contains exact candidate facts and stats.

### Independently restored live snapshot reproduction

Source: `/tmp/scry-childscribe-live-sep05.XlzT6P/source.badger`.

Verified SHA-256 before and after all work: `b8fda9a1c446d9f03dd8bc3116d1e49fd6020f7cd0e7d728553087b6ae445197`.

Independent restored directory: `replica-4202465833` under this report's directory. It contains 79,692 facts before and after the test.

The actual restored graph already contains a current `scry related_to hermes-ops` fact with raw relation `measured`, confidence 1, valid-from `2026-09-03T00:00:00Z`, and two original episode IDs. Its sentence starts `81 facts misfiled on hermes-ops; hygiene reports reattached=0 ...`.

On this isolated replica only, normal Apply received a new synthetic `aliases_index_to` claim on those established endpoints, with the unmistakable replica-only sentence `INDEPENDENT REPLICA-ONLY CASE: the audit's reviewed alias index routes to the distinct Hermes ops project.` and episode `independent-replica-fallback-collision`.

Result: `FactsAdded=0`, `FactsMerged=1`; old `measured` text/raw remain, new episode ID is appended, and the incoming sentence/raw are absent. This demonstrates the failure on the real restored graph shape. It is not a claim that this synthetic sentence was observed in production. No historical fact replay or repair was performed.

## Passing checks

- Independent corpus: 2,154 checks, all 31 effective explicit identity synonyms, all existing tense/modality prefixes, nested valid prefixes, case and surrounding underscore normalization, negation, nonidentity operation compounds, and recursive preposition/passive suffix attacks. No unapproved compound produced `same_as` in the final corpus; existing explicit identities and flip direction survived.
- Canonical vocabulary is unchanged and exactly 39 relations.
- The two observed current ChildScribe routing raw relations both change from `same_as` to `related_to`, without direction reversal.
- Fresh normal Apply on another independently restored live replica preserves incoming sentence, raw relation, source/destination, explicit valid-from, confidence, invalidation state, and episode provenance when there is no existing colliding triple. All 79,692 preexisting facts and 30,231 preexisting entities remain byte-equivalent at JSON serialization.
- Production code diff contains no store iteration, restore, repair, or bulk mutation: one pure mapper wrapper/helper and tests/documentation. Read-only mapping diagnostics do not alter existing facts or entities.
- Parent full `go test ./...`: PASS, exit 0 (`full-suite.log`). Candidate committed test suite using `go test ./... -skip '^TestIndependent'`: PASS, exit 0 (`new-full-suite.log`). The independent final test run exits 1 solely for the two demonstrated loss cases.

Corpus development transparency: the initial exploratory run also tested `matches_flip` as an identity and `will_still_be_` as an established prefix. `matches_flip` is explicitly removed from the effective synonym table; `be_` is not an existing tense prefix. Those initial failures are not counted as regressions in the verdict. The final independent corpus uses the effective table boundary and retained prefixes. Unknown semantic identity paraphrases outside that table remain an intentional conservative limitation; the observed changed live sentences did not prove an actual identity regression.

## Diagnostic existing-raw mapping delta — no historical rewrites

Independent mapping replica: `replica-1141724077`.

- 79,692 total facts: 71,968 current, 7,724 invalidated.
- 30,231 entities.
- 8,946 distinct effective raw strings (`RawRelation`, falling back to stored `Relation` when empty).
- Exactly 19 distinct strings differ between old and new Map: 25 current facts and 2 invalidated facts.
- Every change is `same_as,false` → `related_to,false`; no other mapping delta was observed.
- Four affected current facts already have a same-endpoint current fallback triple (one each under `cloned`, `cloned_at`, `dropped_duplicates`, `duplicated`), underscoring that the collision shape exists.

| Raw string | Current | Invalidated |
| --- | ---: | ---: |
| aliases_index_to | 1 | 0 |
| called_in | 1 | 0 |
| called_within | 1 | 0 |
| cloned | 4 | 0 |
| cloned_at | 1 | 0 |
| cloned_from | 2 | 0 |
| cloned_to | 1 | 0 |
| clones | 2 | 0 |
| clones_full_history | 1 | 0 |
| dropped_duplicates | 2 | 0 |
| duplicate_ack_carries_event | 1 | 0 |
| duplicated | 2 | 0 |
| duplicates_logic_from | 1 | 0 |
| head_equals | 1 | 0 |
| is_byte_identical_to | 0 | 1 |
| left_duplicate_types | 1 | 0 |
| rehomes_aliases_to | 1 | 0 |
| tree_identical_to | 0 | 1 |
| would_clone_duplicate | 2 | 0 |

`live-delta.json` includes every affected original fact, counts, flips, and before/after preservation hashes. No old fact was remapped in storage. Logical JSON SHA-256 before/after diagnostic iteration:

- Facts: `0cc04e05527872130a9526d1d204278b01de9e46c00be4bc7b549a4d4893734d`.
- Entities: `064ac58e9a73597c15cbafdf5d85daa134a79426abfe50a304a27c454efdc120`.

## Evidence and scope

Root: `/tmp/scry-identity-independent-sep05.vUECnl`.

- `new/`: exact candidate git archive, plus independent test and mechanically copied parent pure mapper under `internal/memory/oldvocab`.
- `old/`: exact parent archive plus isolated differential test.
- `live-delta.json` SHA-256: `b952f539089814c5564d0206eadbd4a9f7185ace827f31f3570118570053903f`.
- `independent-tests-final.log` SHA-256: `e1487d77c40bf474fd859e66bcdb3d67ffbb5db546a0a1458a65341dfba3ebeb`.
- `parent-collision.log` SHA-256: `46235c91a1a8390b84667bcbc38ebce0f23dc50dd8f620913599e7fd06d174bd`.

The review began with `scry memory orient --cwd .` and the full goal objective. All code/test/report edits and restores were confined to the new temporary review directory. No shared repository edits, models, remember calls, live-store writes, deployments, historical rewrites, or other reviewers' replica changes occurred. This bounded gate remains FAIL until the proven incoming evidence loss is addressed and independently retested.
