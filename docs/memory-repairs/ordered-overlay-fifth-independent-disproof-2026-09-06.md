# Fifth ordered identity overlay — independent bounded disproof

2026-09-06. NO-GO for the exact private, uncalled, read-only planner. Its relationship walk expands spelling listings but omits the actual natural-slug identity/control endpoint when that endpoint is absent or has a different canonical name. Seven fresh failing leaves demonstrate this omission through the genuine fixed planner. This is a finite-contract failure; factual support and production integration are not prerequisites added by this review.

Reviewed source: `/tmp/scry-ordered-overlay-fifth-sep06.WNftRz/code`, baseline `e097fa6`, freeze SHA-256 `43a61c978988de72b28a238eef0681f71f5c34cf88d43fe4e65b0d1821d48b22`. Complete controlling contract SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`. The unchanged six implementation files and all supplied tests were reviewed in a fresh git archive. Initial and final verification checks all 600 baseline SHA-256/git-blob hashes and all 46 addition SHA-256/byte lengths in both exports. Final verification additionally checks the review input, contract and all four prior complete NO-GO report pins.

All relative paths below refer to `/tmp/scry-overlay-fifth-disproof.Za4i0d`.

## Executable counterexample

`TestFifthFreshNaturalRouteControlsInRelationships` seeds Atlas with alias `Atlas box`. It then establishes either no entity at `atlas-box`, or an entity with `Slug: atlas-box`, `Name: Borealis`, `Type: machine`. The latter is a legitimate synthetic nonnatural-slug entity; ordinary PutEntity accepts it and the exact synthetic inventory is adopted before planning. There is no fabricated owner or support input to the planner. The fixture places malformed `il:atlas-box` containing `{"broken":true}` after adoption.

Each leaf invokes `planIdentityEpisode` through the unchanged supplied error helper:

| Consumer | Input and observed result, for both natural endpoint variants |
| --- | --- |
| retained-metadata | Atlas(service) receives its first description, retains Atlas box; nil error, one declaration, two decisions |
| already-alias | Atlas(service) repeats Atlas box with its first description; nil error, one declaration, three decisions |
| type-removal | Atlas(concept) becomes service and removes Atlas box through type revalidation; nil error, one declaration, four decisions |

All six must refuse the relevant malformed control with the sanitized planner sentinel and zero report. The test helper first checks full raw equality and zero events; those conservation checks pass in every failing case. No durable planner write is demonstrated.

`TestFifthFreshNaturalOccupantIsDecisionDependency` establishes the same valid adopted `atlas-box` / Borealis occupant without malformed bytes and requests only Atlas metadata. Planning succeeds, but metadata decision 1 omits all five checked dependencies: `en:atlas-box`, `il:atlas-box`, `il-consumed:atlas-box`, `ig:atlas-box`, and `rs:atlas-box`. This is the seventh failing leaf and the same underlying relationship omission, visible even with wholly valid controls. The entity's presence in the complete global inventory is not a dependency witness in that decision.

The passing `TestFifthFreshUnrelatedNaturalControlsStayOpaque` places the malformed anchor at `il:unrelated-box` instead. Atlas metadata and an Atlas uses Cygnus assertion remain useful, the retained alias is not silently repaired, raw rows remain identical and no events emit. The correction must not perform a global integrity refusal.

## Cause and smallest correction

`identity_overlay_lookup.go:347` places the actual explicit spelling's Normalize/Slugify pair on the worklist. At line 391 the walk consults `baseline.Listings[norm]` and `baseline.NaturalListings[natural]`, then validates only the owners collected from claims or these listing occurrences. `identity_relationship_inventory.go:66` builds NaturalListings by Slugify of names and aliases; it does not index the actual entity storage slug. Therefore the test's natural key `atlas-box` contributes Atlas's alias occurrence, but not the `atlas-box` entity named Borealis. When the natural entity is absent, no owner is collected for its lifecycle controls either.

The already-alias shortcut and immediate type-removal path do not subsequently call aliasOwner to discover that missing endpoint. Metadata-only installation likewise depends entirely on the incomplete collector. The fifth correction fixed actual spelling rt/rs and alias-rejection pairs, but those are distinct from lifecycle controls at the derived natural identity route.

Include the actual natural-slug endpoint, including its present/absent entity witness and complete relevant lifecycle controls, when validating affected spelling relationships. Listing expansion alone does not cover that endpoint. Preserve the distinction between validation and owner choice: a valid conflicting/retired/consumed natural route does not itself authorize a transfer, automatically repair an old listing, or invalidate a reviewed explicit rehome. Keep exact-natural precedence, exact-literal retirement VALUE versus derived-route deferral, normal partial utility, and exact malformed raw-key handling. Do not normalize a defective raw alias address into unrelated control addresses. This report proves the stated natural-endpoint omission; it does not claim a complete final support-dependent ownership rule.

## Whole-unit coverage and limits

Session orientation ran first and its output was read. I read the complete active goal, fifth/fourth/third review inputs, controlling contract and embedded independent corrections, all four previous complete NO-GO reports, delayed-birth design/review, ControllerV3 and identity-mutation independent reviews, and birth-registration/episode-selector contracts. Large combined reads with truncated output were followed by the missing individual/ranged reads. All six overlay implementation files were read completely, along with the relevant neutral lexical DTO/declaration/alias/resolve adapters and exact policy verifier. I inspected the actual registration, serialized owner/finalizer, entity and FA inventories, identity and fact writer ledgers, active legacy recognition, stored EP provenance, original declaration/primary/hint routing and alias consumers. No separate assertion-bridge source was copied or treated as candidate evidence.

Every selected unchanged supplied planner, prior independent disproof, delayed-birth characterization, ordered-contract characterization and differential/adaptor test passes. Thus the prior 20, 4, 19 and 4 failing leaves remain fixed on this exact source. The retained run also covers the new root spelling closure tests, including distinct normalized addresses for natural listings, exact defective raw suffixes, valid unknown control extensions, and unrelated rejection pairs.

Broader retained coverage includes complete original declaration/assertion slots, observations and nil/present Supersedes; inverse and value-source original-side maps; actual first discovery and later observation links; rejected-primary visibility to later primary and earlier hint records; typed failed declaration persistence and independently trusted replacement; exact-natural versus canonical-homonym and distinct-name explicit alias precedence; ordinary retired/consumed/unadopted/current/history deferral with unrelated utility; old metadata without same-episode facts and full repo-ref preservation; alias no-transfer rules and provisional release producer chains; current-episode vote deduplication and legacy/generation evidence isolation; actual stored EP checks including current-input IDs; owned input/report buffers; genuine owner/phase/poison guards and discarded-transaction read failure.

The exact policy verifier passes all 157 manifest declarations / 248 symbols with no extra copied declarations under the documented DTO/name substitutions. The retained differential test reports 2,323 literals, 16 unary predicates, 18,084 alias projection cases and 264 declaration projection cases. The unchanged original resolver corpus and closed vocabulary tests pass. This is mechanical/adaptor parity, not a new held-out semantic identity/value grade or permission to tune lexical dictionaries. Production lexical deduplication remains separate later work.

Static inspection finds no raw or ordinary entity/fact/alias/vote/EP writer in the six planner files. The fixed entry owns canonical bytes before waiting and binds its own program to the genuine serialized registration wrapper. It accepts no chosen owners, support flags, caller finalizer or cleanup list and returns no active handle. Registry registration precedes candidate proposals; later original observations link issued handles. The fixed freeze and real wrapper verify empty writer histories and final captured inventories. All fresh successful/error-expected planner fixtures preserve raw rows and zero emitted events.

The retained zero-write Badger characterization passes: a stale zero-write Commit can return nil. No dummy write was introduced. This review claims this fixed program's lack of writes, snapshot/cooperative serialization and final captured-inventory conservation, not arbitrary raw-writer serialization, transient-write detection or production all-writer enforcement.

## Commands and retained evidence

Every Go command used `CGO_ENABLED=0` and `TMPDIR=/tmp/scry-overlay-fifth-disproof.Za4i0d/test-tmp`. Complete stdout/stderr was retained through tee under `set -o pipefail`.

| Command / log | Result |
| --- | --- |
| `ruby verify-pins.rb copy` -> `initial-pins.log` | Both exports: 600 baseline SHA/blob pins; 46 additions checked before and after apply_patch copying |
| `go test ./internal/memory/store -run TestFifthFresh -count=1 -v` -> `fresh-tests.log` | Six malformed-control leaves and one dependency leaf FAIL; unrelated opaque-control test PASS; store 0.717s |
| `go test ./internal/memory/store ./internal/memory/resolve -run 'TestIdentityOverlay\|TestIndependentOverlay\|TestCorrectedFresh\|TestThirdFresh\|TestFourthFresh\|TestOrderedContract\|TestDelayedBirth\|TestPrivateIdentityPolicyPort' -count=1 -v` -> `retained-tests.log` | PASS, store 22.703s / resolve 2.013s |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` -> `vet.log` | PASS; empty output |
| `go run /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/verify_policy_port.go . /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/policy-inventory-corrected.json internal/memory/identitypolicy` -> `policy-tokens.log` | PASS 157 declarations / 248 symbols |
| `go test ./... -count=1` -> `full-suite.log` | FAIL only in the same seven fresh leaves; store 74.938s. All other packages PASS, including resolve 22.349s and daemon 36.055s |
| `ruby verify-pins.rb` -> `final-pins.log` | Both complete source exports remain exact; review input, contract and four prior report pins also match |

There was no fresh fixture correction, compile failure, source revision or assertion weakening. The exact initial failing new test is the final test file, and both failed logs remain. Once the concrete finite failure was established, I completed the bounded required coverage/report instead of expanding the corpus. The verification script was extended only to validate the already-read external review pins. A no-match static writer search returned rg's normal status 1; no source operation resulted.

The export was created with `git archive e097fa6 | tar` into a new mktemp directory. All 46 additions were copied through apply_patch following exact input checks. All authored files use apply_patch, and only the new test was gofmt-formatted. No baseline or supplied file changed. No actual stores/backups/replicas, SSH, provider calls, room/remember writes, configuration, deployment, cleanup outside synthetic test fixtures or additional agents were used.

New test SHA-256: `4346d109b3437bca2f006f8c7a4ad38fab90f17fae8d185f53b40b4f6a559ba4`. Complete test/log/script hashes are in `EVIDENCE_SHA256.txt`; this report's hash is returned separately.

NO-GO applies only to this finite private planner. This review does not approve either FA phase, virtual backdate/hint eligibility, assertion identity/address/content preservation, support/materialization/B votes, final ownership closure, lifecycle/all writers, Force/current results, production lexical deduplication, ControllerV3 replacement, live adoption/prevention/deployment/cleanup, recall, binaries/performance/sweeps or any full-goal clause or grading round. Those remain mandatory later work.
