# Complete preparation — bounded semantic disproof

2026-09-06. **NO-GO for the frozen complete preparation receipts.** One concrete evidence-preservation violation is reproduced: a deferred different-assertion collision omits the actual FA occupying its address. Two different baseline occupants produce byte-identical complete deferred receipts. This finding concerns proposed receipt semantics; no actual fact loss, unauthorized write, or committed-result claim is made.

The root builder supplied this lead during the review. Its original test is independently reproduced here, and a new independent two-store test demonstrates the loss without requiring a particular replacement field. No other semantic violation was proved in this bounded review. The root reports a correction in its separate workspace; that correction was not copied or graded here.

## Frozen source and method

Input: `/tmp/scry-complete-preparation-review-sep06.DAf00M/REVIEW_INPUT.md`, SHA-256 `5575d436fbb2582720480e8b6b54a4499e985a62bf204c3e5bbe891b1e0851e6`. Freeze: `COMPLETE_PREPARATION_FREEZE.json`, SHA-256 `5f40b0261f7c94a6067caa182cad51244de9d7f7b3ad0a66f1ff7dcb329faa10`.

Private export: `/tmp/scry-complete-prep-disproof.9Uo673/code`, freshly created by `git archive e097fa6 | tar`. The 75 additions were copied with apply_patch after verifying their full SHA-256 and byte sizes. All 600 baseline SHA-256/git-blob pins and all 75 addition pins match in both frozen and independent exports at start and end. Supplied sources and tests remain unchanged. Seven supplied external input/contract/verifier/reproducer pins also match at handoff.

Required session orientation ran first and was read. I read the active goal, complete assertion contract including the controlling addendum, assertion bridge experiment and semantic review, ControllerV3/assertion-identity/identity-mutation reviews, complete ordered-overlay contract including its embedded disproof, and complete tenth review. All 18 admission files were read completely. I inspected actual overlay declaration/lookup/primary/hint/control/alias/query implementations, genuine registration and owner/coordinator, strict FA decoding/inventory, actual identity/fact ledgers, and receipt construction. Additional registration, delayed-birth and selector contracts/reviews were read to clarify their narrower provenance and ordering boundaries. Earlier controller/selector drafts do not override the complete assertion contract.

Every executable fixture uses synthetic stores under the private TMPDIR. `freshComplete` accepts a canonical original revision and supported EP bytes, then runs actual pinning, ordered overlay, assertion construction, exclusive timeline, hints, FA effect closure, identity ownership closure, and both receipt methods inside genuine runBirthRegistration. It verifies full raw equality, zero events, and empty returned identity/fact writer histories on success/error. It passes no forged owner, caller support, accepted rows, replacement witnesses or caller finalizer. The unchanged supplied complete fixture is also exercised by the root reproducer.

No actual stores, backups, replicas, SSH, providers, credentials, config, hooks, rooms, remember, deployment, shared/frozen edits, or extra agents were used. All authored tests/scripts/report artifacts were created with apply_patch; gofmt touched only new tests. Console/log output contains synthetic data only.

## Proven violation: deferred collision loses exact occupant evidence

Independent reproducer: `code/internal/memory/store/complete_preparation_occupant_evidence_test.go`, `TestFreshCompleteDistinctOccupantsRequireDistinctReceiptEvidence`. Root-supplied reproducer: `code/internal/memory/store/complete_preparation_root_reproducer_test.go`, `TestAdmissionReceiptOccupiedAddressKeepsActualBeforeImage`.

Both cases seed recognized Atlas/Borealis/Cygnus identities with genuine synthetic legacy adoption, actual old EP proof, and an old `atlas uses cygnus` assertion starting August 1. The incoming assertion has the same address but different full text. The independent variant runs two stores whose old text is respectively `first actual occupying assertion` and `second different occupying assertion`; both receive exactly the same incoming revision plus an unrelated Draco/Equuleus assertion.

Observed results:

- Both conflicting incoming occurrences are correctly deferred, with resolved legacy endpoints, nil materialization and zero proposed effects.
- The unrelated assertion remains prospectively committed, and both runs preserve all actual raw rows and zero events/writer history.
- Neither deferred receipt includes the actual occupied key in `Evidence.Witnesses`; its matching-assertion `Before.Exists` is false, and `Final` is nil.
- Actual old raw hashes differ: `c4205f5c7b233cab588e54d94f22674d16dd92bb57bbf20208b71f4d21715ede` versus `74f41d8f279167316b8533452b4834d0cf8034f253581ef63edaf1ce2639ea2a`.
- The complete deferred receipt bytes are identical, SHA-256 `790f83894f1384c56971f532099c15d0755edba95747065dd0ff22455f145efc` in both runs. Therefore the result does not merely use another field to retain the occupant: the distinction has disappeared from the receipt entirely.

Cause: assertion construction keeps actual baseline records and incoming full identities separately, correctly marking the shared-address component deferred. `receiptFactView` copies only the incoming proposal's `Before`, which denotes absence of that full assertion. `closeFactEffects` skips removed new proposals, so its `validateEffect` never captures the actual address point read for them. Receipt evidence follows only accumulated traces; no matching baseline occupant is added. Internal planner inventories do contain the row, but the owned returned occurrence receipt does not preserve it.

Affected contract: “Exact assertions, actual addresses and metadata” requires actual legacy address/full raw bytes separately from full assertion identity, and exact absence/full before-images as dependencies. “Components, closure and a single materialization” requires complete per-occurrence exact FA/evidence, with truthful resolved endpoints and no reconstructed guesses after replay. This is already a preparation requirement; it does not depend on implementing the later durable writer.

Smallest necessary correction: retain the exact actual address before-state separately from the before-state of the matching full assertion, including a genuine exact point witness for every proposed full assertion address even if its component is already deferred. Keep the different occupant distinct from accepted materialization. Preserve current local deferral and unrelated utility. The root's suggested `AddressBefore` field is one suitable representation; the independent reproducer requires exact evidence, not that particular field name.

Reproduce from `code/` with `TMPDIR=/tmp/scry-complete-prep-disproof.9Uo673/test-tmp CGO_ENABLED=0 go test ./internal/memory/store -run 'TestAdmissionReceiptOccupiedAddressKeepsActualBeforeImage|TestFreshCompleteDistinctOccupantsRequireDistinctReceiptEvidence' -count=1 -v`.

## Other executable challenges and retained fixture correction

Eight fresh top-level groups pass in `fresh-tests-final.log`: six order permutations of same-target overlapping exclusive sentences followed by a different target; inherited contradictory old history while exact metadata repeat and unrelated utility survive; cyclic hints and equal-event ineligible hints; extension/whitespace/duplicate-field raw rows across repeat/exclusive/hint routes; raw-relation and normalized-address/full-literal collisions; an old FA EP reference equal to current input ID with no actual old EP; a future value-source hint; and the same future source with actual unadopted baseline identity. Receipt mutation checks preserve subsequent byte-identical results.

The first fresh run had one incorrect fixture expectation. I assumed the original hint source `Missing` was unresolved. The fixed inherited lexical policy classifies it as a value and reverses the hint onto Borealis; that yields a valid no-target hint and a committed primary. The eligible 1971 target still receives the correct independent hint effect. Initial source `fresh-initial.go.txt`, failed `fresh-tests-initial.log`, and complete `future-diagnostic.log` remain intact. The corrected characterization keeps the same names and original input, explicitly checks this inherited value branch, and expects its no-effect disposition. A separate case adds a real unadopted synthetic `en:missing` row with the same Name, establishing genuine route deferral without tuning the dictionary or changing the question. It passes and does not withhold the earlier eligible unique target. No supplied test or production behavior was changed.

A diagnostic log statement was temporarily added to the new test to capture complete receipts, then removed in the corrected new source. The initial failed source is exact and preserved. An initial Ruby extraction command used invalid slice syntax while copying the root reproducer; it failed before creating that file. Its following gofmt reported the missing new file. The independent occupant test still ran and failed semantically; the corrected copy command then created the exact root test function, which independently fails. Several read-only path guesses failed before locating the referenced files; none changed data or test expectations.

## Executed checks

All Go commands used `CGO_ENABLED=0`, the private TMPDIR above, and complete stdout/stderr retained through tee under pipefail. Exit statuses were returned by the command tool. Durations are outcomes, not performance measurements.

| Check | Outcome / log |
| --- | --- |
| Both exports, all baseline/addition pins | PASS, `initial-pins.log` and `final-pins.log` |
| Unchanged full supplied `go test ./... -count=1` before new tests | PASS all packages; store 70.939s, resolve 17.351s; `full-suite.log` |
| Fresh initial adversarial suite | FAIL one fixture assumption; `fresh-tests-initial.log` |
| Original future fixture full receipt diagnostic | FAIL same fixture assumption; `future-diagnostic.log` |
| Corrected fresh suite plus actual unadopted control | PASS store 1.133s; `fresh-tests-final.log` |
| Retained Admission/Overlay/IdentityOverlay/DelayedBirth/OrderedContract/Policy selection | PASS store 10.518s, resolve 1.185s; `retained-tests.log`; only the newly added root disproof excluded |
| Independent different-occupant evidence test | FAIL with actual differing FA/identical receipt digests; `occupied-evidence-disproof.log` |
| Root-supplied exact reproducer | FAIL; `root-reproducer.log` |
| Full `go test ./... -count=1` including both disproofs | FAIL only the two new disproof tests; store 76.451s, other packages pass; `full-suite-with-disproof.log` |
| Relevant store/identitypolicy/resolve vet | PASS before and after final new tests; `vet.log`, `final-vet.log` |
| Exact pinned policy verifier and corrected manifest | PASS 157 declarations / 248 symbols; `policy-tokens.log` |

## Verdict limits

This NO-GO applies to exact frozen complete preparation receipt evidence. It is not a finding that later writers must already exist, nor approval of those writers. Proposed committed/supported/accepted labels were evaluated as intended dispositions only. VoteRetained remains a qualifying observed vote for a retained actor, not B/legacy/generation writer authority.

No durable V2 codec/selector, Force or changed-revision replay, EP/FA/adj/en/al/ig/iga/legacy vote materializer, normal Apply adapter, final actual inventory checker, production policy deduplication, writer/schema floor, live adoption/cleanup, backup/rollback, benchmark/recall/sweep, or full-goal clause is approved. Root remains sole builder. No unrelated helper prerequisite is imposed. Root's separate correction needs its own concrete verification; nothing in this report certifies it.

All authored test/script/report/log hashes are indexed in `EVIDENCE_SHA256.txt`. This report's hash is supplied separately after writing.
