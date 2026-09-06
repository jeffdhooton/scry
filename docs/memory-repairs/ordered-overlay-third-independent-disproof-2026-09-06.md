# Third ordered identity overlay — independent code disproof

2026-09-06. **NO-GO for the exact frozen private, uncalled, read-only planner.** Four mechanisms violate its existing finite contract: ordinary lifecycle deferrals hide malformed canonical retirement controls; relationship consumers retain malformed owner controls without validating them; unlisted index claims do not expand into their complete listing dependency set; and a released provisional claim loses the producer of its absence. Six fresh failing groups contain nineteen failing leaves. None of these findings requires implementing factual support or production integration to correct this unit.

Reviewed baseline `e097fa6cb2e5c5ff8afcbd2b21c163d879856d84`, freeze manifest SHA-256 `76808342886d78e364b6e3f38eb5e98e9d0a4f79bb099a64a42de5979f8cdea1`, complete controlling contract SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`. Both earlier complete NO-GO reports were read, and their source tests remain unchanged. No candidate implementation or supplied assertion was edited.

All relative paths below refer to the independent export `/tmp/scry-overlay-third-disproof.3bkHYs`.

## 1. Consumed/retired alias targets hide malformed canonical retirement controls

`identity_overlay_controls.go:284–290` returns consumed/retired deferrals before reading `rt:Normalize(e.Name)`. This is observable when the entity is reached through a different listed alias: the mention's retirement lookup cannot validate its owner's canonical spelling.

`TestThirdFreshDeferredOwnerCannotMaskCanonicalRetirement` seeds adopted Atlas with listed alias Polar Manual, valid `rs:atlas`, and malformed `rt:atlas`. The fixed planner is then called separately through a Polar Manual declaration, primary, status destination, and supersedes hint. All four return nil error and a nonzero descriptive report. The relevant malformed canonical row is hidden behind an ordinary retired-identity deferral.

`TestThirdFreshConsumedOwnerCanonicalRetirementValidation` produces the equivalent consumed-identity case using an actual encoded consumption record containing the exact adopted anchor. Without malformed retirement bytes, the dependent primary defers and the unrelated Cygnus/Draco primary remains proposed; this control case passes. Adding malformed `rt:atlas` still returns a successful report instead of sanitized error/zero report. This fifth leaf proves the omission is not limited to the retired branch.

The contract requires relevant malformed controls to error rather than be masked by ordinary deferrals. It does not require a stored identity to become trusted merely because an alias points at it. Smallest correction: validate the complete relevant canonical retirement control before returning either normal lifecycle deferral, preserving the ordinary valid-control result and exact literal versus derived-spelling precedence. Do not turn a legitimate deferral into a blanket episode error or infer a rehome.

## 2. Relationship consumers read malformed owner controls without validation

`identity_overlay_lookup.go:322–327` places owner entity/control bytes in witnesses using `read`; it never validates those controls. Two actual producers can reach this collector without the claim reader that otherwise validates the owner.

`TestThirdFreshRevalidationValidatesRelevantMissingOwner` starts with adopted Atlas(concept), alias Atlas box, `al:atlas-box -> absent-owner`, and a malformed control for absent-owner. Declaring Atlas(service) immediately removes Atlas box through the lexical leak branch at `identity_overlay_aliases.go:181`; the claim validation guarded by `if !remove` is skipped. Relationship collection reads the malformed bytes, then the planner successfully reports four decisions, including the removal. All four control families (`il:`, `il-consumed:`, `ig:`, `rs:`) fail the required sanitized-error assertion.

`TestThirdFreshMetadataRelationshipControls` proves the same omission without type refinement or alias removal. Atlas(service) receives its first description while retaining Polar Manual. Its relevant alias relationship points either to a missing index owner or to another existing entity listing Polar Manual. For each of those owner configurations, malformed `il:`, `il-consumed:`, `ig:` or `rs:` is merely collected during the final metadata install. All eight cases return nil error with two decisions.

These twelve leaves concern owners of spellings in the changed identity's explicit relationship set, not unrelated opaque controls elsewhere in the store. The unchanged-defect exception cannot substitute witness presence for validation of relevant malformed identity controls. No new same-episode fact requirement for old metadata is appropriate. Smallest correction: give every relevant relationship consumer the same complete typed control validation, including immediate removals and metadata-only installs. Keep validation separate from choosing an owner, proposing a transfer, or granting support.

## 3. An unlisted index claim omits other listing dependencies

`TestThirdFreshUnlistedClaimIncludesOtherListings` creates Atlas, creates Borealis listing Polar Manual, and explicitly claims Polar Manual for Atlas to establish the synthetic old defect. Atlas does not itself list that spelling. Both identities are adopted before the planner call. An Atlas description-only declaration then changes its projected metadata.

The metadata decision's trace contains `al:polar-manual`, but omits `en:borealis`, `il:borealis`, `il-consumed:borealis`, `ig:borealis` and `rs:borealis`. `relationships` reads all `baseline.IndexTargets[atlas]` at lines 296–299, but only the caller's canonical/alias spellings are subsequently expanded through `Listings` and `NaturalListings`. Thus the old unlisted key never contributes its other listing owner to the decision's dependency set.

The contract explicitly includes all old index keys targeting a changed identity, including unlisted/untouched keys, and every other entity listing the same normalized or natural spelling. Their presence somewhere in the complete global baseline inventory does not make them the recorded dependency set of this metadata proposal. Smallest correction: expand affected unlisted index spellings into the complete normalized/natural listing closure and retain the related owner controls, with the same relevant validation discipline. This is descriptive closure, not approval of a final support-dependent ownership set.

## 4. A released provisional claim loses its producer dependency

`TestThirdFreshReleasedClaimRetainsProducerDependency` uses only original declarations on an empty synthetic store:

1. Atlas(concept) proposes Atlas box as an alias.
2. Atlas(service) upgrades type and removes that alias at decision 6.
3. Atlas box(machine) becomes a new canonical candidate; its declaration result is decision 11, and its useful primary resolves successfully.

The third declaration has no direct or transitive producer dependency on decision 6. The test permits either representation; it does not require one particular direct edge. The cause is `delete(p.claims, norm)` at `identity_overlay_aliases.go:217`: a later `claim` observes absence but has no record of the proposal that made the key absent. Its baseline was also absent, so the normal raw witness supplies no temporal predecessor.

This case does not claim that the later identity should be refused. With no real baseline owner, it can independently qualify. Its descriptive permission nevertheless depends on the earlier provisional release, and the frozen contract requires full ordered producer history. Smallest correction: retain a producer-bearing absence/tombstone (or equivalent exact dependency record) for a released claim and include it when later routing/proposal logic consumes that absence. Such history must never grant permission to transfer a real baseline owner's key.

## Scope and broader passing evidence

I ran and read session orientation before other work. I read the complete review input, active goal, frozen contract with incorporated independent corrections, both complete earlier code NO-GO reports, delayed-birth design and review, ControllerV3 and identity-mutation reviews, and registration/selector contracts. All six implementation files were read completely. I inspected the neutral DTO adapters/token verifier, the original declaration/primary/hint/alias resolution paths, genuine registration/owner/finalizer, relationship and fact inventories/ledgers, legacy/generation recognition and relevant stored evidence codecs.

Every retained supplied overlay test, both earlier independent disproof suites, the original eight delayed-birth groups and six ordered-contract groups pass. The prior twenty and four failing leaves therefore remain fixed on this source. The selected retained run reports store 11.516s and resolve 2.169s. It includes genuine invalid-phase/owner/poison probes, owned input/report tests, discarded-transaction read failure, exact-natural/canonical-homonym/distinct-name explicit alias precedence, failed-declaration binding persistence/replacement, current/history-only deferrals, generation/legacy evidence isolation, one-current-episode votes, malformed stored EP controls, inverse and rejected discovery order, complete original slots and hint visibility.

Fresh passing cases verify a multiple-alias-removal producer chain, final projection producers, first-description/concept-only type evolution, and a hint depending on a candidate discovered by a later rejected primary. The literal/derived retirement matrix preserves exact Polar Manual as a value, while Polar Manual! and Polar Manual? remain typed dependency deferrals in both primaries and hints; unrelated utility and full unknown-extension bytes survive. The valid consumption control likewise preserves partial utility.

The exact pure-policy token verifier passes all 157 manifest declarations / 248 symbols with no extra copied declarations under its documented DTO/name substitutions. The original resolver corpus and retained differential/adaptor checks pass. The differential test reports 2,323 literals, 16 unary predicates, 18,084 alias projection cases and 264 declaration projection cases. This is mechanical/adaptor parity, not a new held-out semantic-quality grade or permission to change dictionaries. Production lexical deduplication remains required later.

Static inspection finds no raw or ordinary entity/fact/alias/vote/EP writer in the six overlay files. The fixed entry owns canonical bytes before waiting, binds its own program to the genuine serialized registration wrapper, supplies no caller-selected owners/support/finalizer, and returns no active handle. New candidates register before proposals; later original observations link their issued handles. Final captured inventories and empty writer histories are verified by the genuine wrapper. Every fresh normal success/error assertion above checks complete raw equality and zero events through the unchanged supplied helpers. No failure here demonstrates a durable write by this planner.

Badger's zero-write Commit limitation remains explicit: the retained characterization confirms stale read-only Commit may return nil. No dummy write, raw-writer immunity, untracked transient-write detection or production all-writer guarantee is claimed. Snapshot/cooperative serialization and this fixed program's lack of writers are the actual boundary.

## Executed commands and retained evidence

All Go commands ran from the independent export with `CGO_ENABLED=0` and `TMPDIR=/tmp/scry-overlay-third-disproof.3bkHYs/test-tmp`. Stdout/stderr were retained with `tee` under `set -o pipefail`. No runtime test store was intentionally placed outside this export.

| Command / log | Result |
| --- | --- |
| `ruby verify-pins.rb` -> `initial-pins.log`, `final-pins.log` | 600 baseline SHA-256/git-blob hashes and 41 addition SHA-256/byte sizes in both independent and frozen exports; exact review-input, manifest, contract and both prior-report pins. |
| `go test ./internal/memory/store -run TestThirdFresh -count=1 -v` -> `fresh-initial.log` | Initial two failing groups / eight failing leaves; multi-removal dependency and retirement matrices PASS. |
| Same command -> `fresh-expanded.log` | Adds consumed and metadata cases; four newly added other-listing fixtures fail during setup, separately identified below. |
| Same command -> `fresh-final.log` | After only fixture setup correction, four failing groups / seventeen failing leaves; five fresh passing leaves. |
| `go test ./internal/memory/store -run 'TestThirdFresh(Unlisted\|Released)' -count=1 -v` -> `fresh-dependencies.log` | Two additional dependency groups FAIL, one leaf each. |
| `go test ./internal/memory/store ./internal/memory/resolve -run 'TestIdentityOverlay\|TestIndependentOverlay\|TestCorrectedFresh\|TestOrderedContract\|TestDelayedBirth\|TestPrivateIdentityPolicyPort' -count=1 -v` -> `retained-tests.log` | All selected unchanged supplied tests and characterizations PASS. |
| `go run /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/verify_policy_port.go . /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/policy-inventory-corrected.json internal/memory/identitypolicy` -> `policy-tokens.log` | Exact mechanical token PASS, 157 declarations / 248 symbols. |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` -> `vet.log`, `final-vet.log` | PASS with no output, including the final fresh source. |
| `go test ./... -count=1` -> `full-suite.log` | Intermediate complete run FAIL only in initial two fresh groups/eight leaves; store 72.951s, resolve 17.347s, daemon 28.731s. |
| `go test ./... -count=1` -> `final-full-suite.log` | FAIL only in six fresh groups / nineteen leaves; store 67.992s. All other packages PASS, including resolve 16.777s and daemon 27.550s. |

The expanded other-listing fixture initially attempted ordinary PutEntity for Borealis while Atlas owned the shared alias. Existing ownership protection correctly refused before planning. I retained that exact source as `fresh-expanded-fixture-source.go.txt` and its failing `fresh-expanded.log`, then added an explicit synthetic `ClaimAlias` before the new entity seed, matching the already retained shared-alias characterization. No assertion changed; all four then reached the fixed planner and reproduced the intended malformed-control error omission. The original source is retained as `fresh-initial-source.go.txt`. All initial tests and assertions remain in final source, including initially passing ones. A read-only attempt to inspect a guessed inventory filename failed because that filename did not exist; the actual relationship inventory was located and read. No files changed from that read attempt.

I created the export with `git archive e097fa6 | tar` into a fresh mktemp directory, then copied all 41 additions via apply_patch after checking their hashes. Authored tests, source snapshots, pin script, report and evidence index use apply_patch; gofmt only formatted new tests. Initial and final pin verification both PASS for all 600 baseline SHA-256/git-blob hashes and all 41 additions in both exports, with exact manifest/contract/review/report pins. All supplied and baseline files remain unchanged. No shared/frozen edits, real store/backup/replica, SSH, providers, configuration, room/remember write, adoption/deployment/cleanup outside synthetic fixtures, or subagents were used.

## Test pins and verdict limits

- `internal/memory/store/overlay_third_fresh_test.go`: SHA-256 `bf1d8a45ba242ebb06aa83239ac4542ebda867b6bce2461b3d86939e895950af`.
- `internal/memory/store/overlay_third_dependencies_test.go`: SHA-256 `f40e35f9f48ea1247bf969818e46bb5edcd9582c9d54fa78b667655ced2fbb87`.
- Initial fresh source snapshot: `3e87c7b74187c2ba9dcd177e4edf21e56393b3856cb2357a65d50db2b7a25df1`.
- Expanded fixture-failure source snapshot: `4c142a89847167e4b024c46e91fdb047c15980d95f78d34d0b926e15addb14c7`.

Complete source/log/script hashes are in `EVIDENCE_SHA256.txt`; the final report hash is delivered separately. NO-GO applies to the exact private unit described above. A subsequent bounded GO cannot approve FA phases or virtual backdate/hint eligibility, assertion identity/address/content preservation, actual support, materialization/B votes, final alias ownership closure, lifecycle/all-writer enforcement, Force/current-result integration, stats/ingestion, ControllerV3 replacement, live adoption/prevention/deployment/cleanup, recall, binaries/performance/sweeps, or any full-goal completion clause. Those are mandatory later obligations, not added prerequisites to reviewing this finite planner.
