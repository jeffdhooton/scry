# Independent canonical regrade: FAIL

Candidate `1dac187a1e77402e28d92a617de3f18d6da41548`, independently archived in a new temporary directory on 2026-09-05. The previous `a965177` was not deployed; live remains `393eeec`. This gate covers canonical-name resolution and its safeguards only, not deployment or the full memory-quality goal.

The original retained-homonym and generic-reference failures are fixed. A remaining normalization inconsistency still admits generic reference metadata through the new cross-type exception.

## Blocking disproof: the determiner guard sees raw spelling, but identity equality sees normalized spelling

`internal/memory/resolve/resolve.go:347` calls `isDeterminerPhrase(name)` on the unnormalized incoming mention. The canonical equality and reference-word checks already normalize spaces and underscores into hyphens. Consequently a generic reference refused in spaced form is newly accepted in underscore or hyphen form.

The final independent fixture keeps the **same spaced canonical owner** in all variants:

`{Slug:"retained-reference-project", Name:"our own machine", Type:"project", Description:""}`

Apply an extracted declaration with type `machine`, description `The machine has 128 GB of RAM.`, and one of these names:

| Incoming mention | Old `393eeec` | Candidate `1dac187` |
| --- | --- | --- |
| `our own machine` | Refuses, unchanged graph | Refuses, unchanged graph |
| `our_own_machine` | Refuses, unchanged graph | Succeeds; copies machine description to project |
| `our-own-machine` | Refuses, unchanged graph | Succeeds; copies machine description to project |

The same differential holds for canonical owner `this physical host` and mentions `this physical host`, `this_physical_host`, and `this-physical-host`.

For each wrongly accepted variant, candidate stats report `EntitiesUpdated=1`, `EntitiesCreated=0`; the existing record stays type project but receives the machine-specific description and LastSeen `2026-09-05T22:00:00Z`. This is a concrete cross-type metadata update from a reference that is supposed to be rejected, not a merely theoretical naming concern. No valid named machine is being rejected in this fixture: the canonical owner is deliberately a legacy generic phrase, and only the input separator changes.

The fix should apply the existing determiner rule to the same normalization group used by the exception. No additional role-word list, store-wide repair rule, or identity inference is needed to address this specific mismatch.

Reproducer: `TestIndependentUniqueCanonicalDeterminerNormalization` in `code/internal/memory/resolve/independent_unique_canonical_test.go`. `normalized-owner-final.log` records four failures on the candidate; `normalized-owner-old393.log` records all six cases passing refusal/preservation on old production code. The earlier exploratory `extra-independent-tests.log` also found the mismatch when the stored owner used the same separator as the incoming name; the final fixture is stronger because the canonical stored name is fixed and spaced.

## Passing independent checks

- All original independent canonical tests now pass (`original-independent-tests.log`): retained cross-type homonyms refuse; natural exact-slug precedence remains; generic `the machine` refuses; stale ownership, incompatible noncanonical aliases, concept-only aliases, and punctuation/natural-slug collisions remain protected.
- One episode with two distinct existing machine/project identities cannot steal their names as cross-type aliases or fuse their fact endpoints.
- Unique retained canonical concept identities upgrade across normal case/space/underscore/hyphen spellings without duplicate entities or description replacement.
- A new independent staged-write probe constructs the second homonym **inside the same transaction**, restores the claim to the original owner, and calls the real resolver. It detects the staged entity, returns `ErrAliasClaimed`, and rolls back facts/entities/episodes/aliases completely.
- Independent homonym variants using case, surrounding whitespace, underscores, and hyphens are all detected as ambiguity. No owner is selected by type or index position.
- Full committed suite: `go test ./... -skip '^TestIndependent'`, exit 0 (`full-suite.log`). It does not cover the new independent separator-normalization disproof.

## Actual postmigration snapshot

Source `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T200715Z.badger` was hash-verified and freshly restored for each actual-store probe.

SHA-256 before/after: `5323fef7b594c748fd61462909cc808f78c1bd0a104da070abdc54cb21e102aa`.

All six synthetic declarations (`tool`, `concept`, `project`, `machine`, `service`, protected SQL-artifact `value`) successfully resolve canonical name `db/migrations/0160_task_evidence_rules.sql` to retained slug `0160-task-evidence-rulessql`.

Two explicit synthetic facts exercise this canonical name as both source and destination. Each probe preserves all **80,203 original facts**, including history and provenance, all **51,749 alias-index claims**, and all **30,436 entity records** except the intended survivor LastSeen refresh. The survivor's reviewed type/name/description/aliases/repo references/creation time remain unchanged. No new natural-slug identity appears. Each restore stays isolated in this review's directory.

The exact extracted type of the real parked operation note remains **unknown** because the intermediate provider response was not retained. These are mechanism probes, not a replay of that missing output.

## Measured runtime cost

On another fresh restore with 30,436 entities, 40 full exceptional `resolveEntity` transactions were measured. Each resolved the SQL canonical filename with mismatched type `machine`, then deliberately returned a sentinel to roll back. Whole-graph hashes remained identical before/after.

- Median: **70.472 ms**.
- p95 (38th sorted sample of 40): **86.658 ms**.
- Maximum: **122.147 ms**.

This includes the transaction-visible complete entity scan and ordinary resolve/update work. It was measured on the review laptop while the full test suite also ran; it is credible local cost evidence, not a production SLO measurement. Cost remains linear in entity count for each exceptional mention and can accumulate when an episode contains many such mentions. Ordinary compatible canonical mentions and noncanonical alias admission do not take this new scan. No claim is made about live remember p95 or sweep throughput.

## Artifacts and scope

Root `/tmp/scry-canonical-regrade-sep05.Fu54Tq`:

- `code/`: exact `1dac187` archive plus independent tests.
- `original-independent-tests.log`: original disproof corpus and fresh actual-store probes, PASS; SHA-256 `ebefde5d6394b7c62bbc6dafceec84f9352dcd968265c601e2e9f65e850ee714`.
- `extra-independent-tests.log`: staged visibility, normalized homonyms, initial reference probes, and runtime samples.
- `normalized-owner-final.log`: final canonical-owner-fixed regression, FAIL; SHA-256 `87c1d9aa4a91f2a18dfa6be21364ff1bbc4dd1e5e2230d42a31eea019da41021`.
- `normalized-owner-old393.log`: exact final fixture under old production source, PASS.
- `full-suite.log`: committed suite PASS.

The full goal house rules were already read during this continuous independent review session. All code, tests, synthetic writes, restores, and reports were confined to review-owned temporary paths. No shared repository edits, live-store mutations, deployments, model calls, remember calls, or historical repairs occurred. The candidate remains blocked until this proved normalization bypass is fixed and independently regraded.
