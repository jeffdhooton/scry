# Identity relationship inventory — independent bounded disproof

2026-09-06. GO for the frozen private, uncalled read-only inventory unit only. No contract violation was proved by source inspection or the synthetic executions below. This is not approval for production routing, a fixed finalizer, semantic ownership decisions, or live repair.

I ran the required memory orientation, read the complete active objective, complete inventory contract and archived identity-mutation design review. I independently exported baseline `8fda77e3fd4d4f001ae9038994ab6e12d8bab480` into `/tmp/scry-inventory-independent.tTqsVm`. I copied only the three frozen candidate files through apply_patch and verified all three supplied pins. I read the strict legacy decoder, Store transaction/normalization behavior, generation facade guard, owner harness, and the baseline fixture helper. The archived review read from the checkout matches the export hash below. The implementation and supplied assertions were not edited.

## Executed evidence

`CGO_ENABLED=0 go test ./internal/memory/store -run 'Test(RIIndependent|RelationshipInventory)' -count=1 -v` — PASS, exit 0, package 1.153s. Six newly authored independent groups and five supplied groups passed. The independent fixture setter always uses Set, including for empty values; deletion is explicit.

`CGO_ENABLED=0 go test ./... -count=1` — PASS, exit 0; store package 37.590s. Complete command output is retained in the logs below. There were no failed executions in this independent export. No assertions were weakened, removed, or rewritten following a failure.

The supplied frozen test already contains the corrected malformed-empty fixture. Source inspection confirms baseline `gradeGenerationSet` treats nil as Delete and nonnil `[]byte{}` as Set. My separate present-empty refusal test uses its own always-Set helper and first asserts the malformed row exists. The parent's prior failed full run and corrected replica measurement were not rerun or independently graded here; they cannot be claimed as this review's evidence.

## Findings within the finite contract

1. Complete selected raw families and framing: the independent fixture contains exact bare prefixes, NUL/invalid-UTF-8 suffixes, present-empty values and 257-byte opaque values across all control/alias families, plus a valid entity and excluded neighboring families. Whole raw-map equality, every family count, and total count pass. The independent digest oracle globally sorts the complete expected map once and constructs unsigned 64-bit big-endian length bytes manually. It matches the scan. The implementation's sorted disjoint prefixes do yield global selected-key order, including `il-consumed:` before `il:`. Zero-family counts and the empty digest pass supplied tests.

2. Full occurrences and distinct natural projection: independent entities have divergent slugs/names, shared canonical names, duplicate aliases, empty spelling, non-ASCII spelling and nil aliases. Exact expected source-key/name/alias-ordinal ordering passes for both projections. Natural `ab` remains separate from normalized `a/b`; empty observations remain; no slug-only listing is invented. Parsed RepoRefs, names, aliases and raw bytes retain their independent representation.

3. Raw owner relationships and lifecycle observations: empty, binary, missing and unlisted raw owners remain observations. Deleting an entity leaves its unchanged reverse alias keys visible while Entities records its absence and the digest changes. An unrelated retained entity's listing remains exact. Adding present-empty generation, selector, consumption, adoption and alias-control rows changes inventory membership/digest without interpreting them as authority.

4. Ownership of outputs: mutations to raw buffers, parsed alias/RepoRefs slices, normalized listing arrays, natural projection maps, reverse-key arrays, family counts and entity maps do not corrupt separate captures or the store. Mutating one normalized listing array does not mutate its natural projection. JSON decoding and copied Badger values separate raw data from projections.

5. Actual views and refusal: a transaction created before a subsequent root commit continues to exclude that commit while seeing its own staged alias addition, empty marker and deletion. Rollback preserves root state. A separately opened Badger read-only handle returns the same inventory. Nil, zero, closed root and escaped discarded facades return the same static error and zero result. The scan uses one Store view for all selected records and never invokes update or emits events.

6. Global malformed-entity refusal: nine independent cases include present-empty, invalid UTF-8, null, trailing whitespace, unknown/duplicate fields, key mismatch, lossy alias-null representation and zero creation time. A valid earlier entity plus opaque alias rows never escape as a partial result. Whole-store raw equality and zero events hold, and errors do not expose the synthetic entity marker. The scan delegates canonical validation to the established strict decoder without reconstructing malformed entity rows.

## Limitations

The complete inventory is bounded to `al:`, `ar:`, `en:`, `ig:`, `il:`, `il-consumed:`, `meta:identity_`, `rs:`, and `rt:`. It is neither a complete fact/support inventory nor a grant of validity to opaque control/alias data. It does not enumerate arbitrary unlisted resolver input spellings. The strict existing decoder intentionally refuses noncanonical entity encodings; this review does not expand its accepted legacy domain.

This review does not certify recognition, ownership transfer, reviewed dispositions, baseline-defect exceptions, support/dependency closure, lifecycle policy, finalizer-owned baseline selection, undo/post-undo closure, all-writer coverage, transient overwritten changes, concurrent raw-writer phantom immunity, production integration, performance limits on real stores, recall, or any whole-goal/live clause. Fixed snapshot visibility is tested; broad concurrency safety is not claimed. No private restored data or root-only replica test was read or copied, and no real store statistics are independently certified. No shared/live/provider/config/room/remember/sweep/deploy writes or target-data output were performed.

## Evidence pins

All relative paths refer to `/tmp/scry-inventory-independent.tTqsVm`.

| Artifact | SHA-256 |
| --- | --- |
| `RELATIONSHIP_INVENTORY_CONTRACT.md` | `b758a05b7e0605e545d19c46923fe3554514a6e2ecfd15845eab78415106b449` |
| `internal/memory/store/identity_relationship_inventory.go` | `eae817f6892b5ec6c2151e4467b187fc9eee55311b1705787e773bbbcc679a2e` |
| `internal/memory/store/identity_relationship_inventory_test.go` | `dfc258a8779f28b5267255dfb1372e522e77541468e4c093a661628a6455d214` |
| `internal/memory/store/relationship_inventory_independent_test.go` | `4c1b6a993734fca6ae37890c1ed4cdc4fa833e3c214eac32f2c4b665c9144794` |
| `relationship-inventory-targeted.log` | `2da92a4c6d83ceb2b3557c041300a17c93a69e1f98c78913404c7c781e228d08` |
| `relationship-inventory-full.log` | `c4e21a0bc5b7810c5099e5e135dcedd67045403c30ddcf37e38090837fd63935` |
| `docs/memory-repairs/identity-mutation-design-independent-review-2026-09-06.md` | `94e6e51f71e8caaa49e23e2e7c8a92de556c73099345a4b29d1c904c8219b3d3` |
| `internal/memory/store/identity_legacy_anchor.go` | `7a84b00d46977c2d8ce97c5b29f35763fa347bcd61f967d7178ef42a399a7de7` |
| `internal/memory/store/identity_generation.go` | `e07c6c50affce548b31a94dc2e94d665a49ccab1d246f9433f49277535619592` |
| `internal/memory/store/identity_admission_owner.go` | `14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82` |
| `internal/memory/store/store.go` | `9491d689f4a41c3ea6bb8e2a622b46f04810b49daecd375a68428fd8dc6d8891` |

This report's SHA-256 is supplied separately.
