# Fourth ordered identity overlay — independent bounded code disproof

2026-09-06. NO-GO for the exact private, uncalled, read-only planner. Affected alias spellings still bypass their own malformed retirement controls in four actual fixed-entry consumers. Owner lifecycle validation is now substantially broader, but it does not validate a separate affected spelling's rt row. This is one finite-contract mechanism with four failing leaves, not a demand to implement factual support or production integration before grading this planner.

Reviewed source: `/tmp/scry-ordered-overlay-fourth-sep06.unMwwZ/code`, baseline `e097fa6`, freeze manifest SHA-256 `2e1eaacbe851b8a0e52cafa3ea3b52357bd0552df399f17b421f3480981c3f34`. Controlling complete contract SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`. Both initial and final checks verify all 600 baseline SHA-256/git-blob hashes and all 44 addition SHA-256/byte lengths in the frozen source and this independent export. Final verification also pins the exact review input and all three prior complete NO-GO reports.

All relative paths below refer to `/tmp/scry-overlay-fourth-disproof.IeVc6y`.

## Concrete counterexample

`TestFourthFreshAffectedAliasRetirementValidation` seeds Atlas and adopts the exact synthetic legacy inventory before planning. It then adds malformed `rt:atlas-box` containing `{"broken":true}`. Each leaf invokes the genuine `planIdentityEpisode` through the unchanged supplied error helper, preserving the complete original declaration:

| Consumer | Baseline / input | Actual result |
| --- | --- | --- |
| already-alias | Atlas(service) lists Atlas box; declare Atlas(service), the same alias, and first description | nil error, one declaration, three decisions |
| metadata-retained-alias | Atlas(service) lists Atlas box; declare only Atlas(service) with first description | nil error, one declaration, two decisions |
| type-removal | Atlas(concept) lists Atlas box; declare Atlas(service) with first description | nil error, one declaration, four decisions, including type revalidation removal |
| unlisted-claim | Atlas(service) does not list Atlas box; actual synthetic ClaimAlias assigns it to Atlas; declare Atlas(service) with first description | nil error, one declaration, two decisions |

All four must return the sanitized planner error and zero report under the frozen relevant-malformed-control rule. These are actor spellings explicitly mentioned, preserved, removed, or present in the actor's complete old index-target relationship set. They are not unrelated opaque rows. The unchanged helper checks complete raw equality and zero emitted events before checking the expected refusal; those preservation checks pass in every failing case. No durable planner write is demonstrated.

`identity_overlay_lookup.go:298` begins `relationships`. Its `collect` callback reads the relevant al row and expands normalized/natural listings. The subsequent owner loop recognizes the collected owners or validates their missing identity controls. Here that owner is Atlas, so recognition validates canonical `rt:atlas`; nothing reads or validates `rt:atlas-box`. The affected rt bytes exist in the complete global baseline inventory, but inventory presence is neither validation nor an individual decision's dependency witness.

The explicit repeat-alias path has an additional visible shortcut: `identity_overlay_aliases.go:73` returns `already-alias` before `retiredSpelling(alias)` at line 78. The preceding relationship collector does not close that gap. Metadata installation and immediate type removal also invoke the collector without an independent affected-spelling retirement check.

Smallest required correction: validate and retain the exact affected spelling's relevant retirement control and its paired rs payload throughout relationship consumers, including retained aliases, removals, already-listed input aliases, and old unlisted claims. Do this without normalizing a malformed raw index key into another key. Retain the distinction between validating controls and selecting an owner or authorizing a transfer. Preserve legitimate exact-natural precedence, reviewed explicit rehomes, exact-literal VALUE versus derived-route deferral, normal typed partial utility, and the complete unknown raw extensions. The broader relevant-control audit should address equivalent early-return/relationship paths consistently; this report proves only the rt omission above.

The independent passing `TestFourthFreshUnrelatedRetirementRemainsOpaque` places the same malformed payload at unrelated `rt:unrelated-manual`, then plans Atlas uses Borealis. The primary remains proposed-resolution with complete raw equality and zero events. A global integrity refusal is therefore not the requested correction.

## Whole-unit coverage and limits

I ran and read session orientation first. I read the complete active goal, supplied review inputs, controlling contract with its embedded independent corrections, all three prior NO-GO reports, delayed-birth design/review, ControllerV3 and identity-mutation reviews, and registration/selector contracts. All six overlay implementation files were read completely. I inspected the actual registration/owner/finalizer, relationship and FA inventories and mutation ledgers, active legacy and generation recognition/provenance paths, original declaration/endpoint/hint/alias routing, neutral lexical DTO/adapters and exact token verifier. This is a review of the entire finite unit; it is not limited to the four latest changed files.

Every unchanged selected supplied overlay test, the earlier independent suites, all eight delayed-birth characterization groups, and all six ordered-contract characterization groups pass. The previous 20, 4, and 19 independent failures stay fixed on this source. The latest root relationship cases also pass unchanged. Covered boundaries include complete original declaration/assertion slots and observations, nil/present supersedes, inverse and value-source side mapping, actual first discovery/link order, later rejected-primary visibility to earlier hints, typed failed declaration persistence and replacement, exact-natural/canonical-homonym/distinct-name listed-alias precedence, stale/missing/unlisted/current/history controls, canonical malformed retirement behind normal consumption/retirement, old metadata without same-episode facts, full old repo refs, projection/release producer chains, candidate vote deduplication and old generation/legacy evidence isolation, actual stored EP validation, owned input/report buffers, phase/owner/poison guards and genuine read failure.

The retained differential test reports 2,323 literals, 16 unary predicates, 18,084 alias projection cases and 264 declaration projection cases. The unchanged resolver corpus and neutral port tests pass. The exact verifier confirms 157 source-manifest declarations / 248 symbols with no extra copied declarations under the documented DTO/name substitutions. This is mechanical/adaptor parity, not a fresh held-out identity/value quality grade. No new dictionary, threshold, relation vocabulary, or production lexical deduplication is approved.

Static inspection finds no raw, ordinary entity/fact/alias/vote/EP writer in the six overlay files. The fixed entry owns input before waiting, binds its own program to the genuine serialized registry wrapper, registers before candidate proposals and links later original observations. It accepts no selected owners, support flags or caller finalizer, and returns no active handle. Its own freeze and the actual registry finalizer retain zero writer history and verify final captured inventories. The supplied owned-report tests and all fresh raw-equality checks remain intact.

The retained Badger characterization explicitly shows that stale zero-write Commit can return nil. No dummy write was introduced. This review claims only the fixed program's lack of writes, snapshot/cooperative serialization and final captured-inventory conservation, not arbitrary transient raw-write detection or production all-writer enforcement.

## Commands and evidence

Every Go command used `CGO_ENABLED=0` and `TMPDIR=/tmp/scry-overlay-fourth-disproof.IeVc6y/test-tmp`. Complete stdout/stderr was retained using tee under `set -o pipefail`.

| Command / log | Result |
| --- | --- |
| `ruby copy-and-verify.rb copy` -> `initial-pins.log` | Both exports: all 600 baseline SHA/blob pins; all 44 copied additions validated before/after apply_patch copying |
| `go test ./internal/memory/store -run TestFourthFresh -count=1 -v` -> `fresh-tests.log` | Four affected-rt leaves FAIL; unrelated opaque control PASS; store 0.647s |
| `go test ./internal/memory/store ./internal/memory/resolve -run 'TestIdentityOverlay\|TestIndependentOverlay\|TestCorrectedFresh\|TestThirdFresh\|TestOrderedContract\|TestDelayedBirth\|TestPrivateIdentityPolicyPort' -count=1 -v` -> `retained-tests.log` | PASS; store 20.182s, resolve 2.380s |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` -> `vet.log` | PASS, empty output |
| `go run /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/verify_policy_port.go . /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/policy-inventory-corrected.json internal/memory/identitypolicy` -> `policy-tokens.log` | PASS 157 declarations / 248 symbols |
| `go test ./... -count=1` -> `full-suite.log` | FAIL only in the same new four leaves; store 72.223s. All other packages PASS, including resolve 21.488s and daemon 34.781s |
| `ruby copy-and-verify.rb` -> `final-pins.log` | Both complete source exports remain exact; review/contract/prior reports also match |

No fresh fixture correction, compile failure, source revision, or assertion weakening occurred. The exact first failing test remains the final test file; both failing logs remain. Two read-only attempts guessed absent filenames (`identitypolicy/api.go` and a manifest `additions` field); the actual projection file and `candidate_added` field were then located and inspected. Neither attempt changed files or affected tests. The pin script was later extended only to check the already-read input/report pins.

The private export came from `git archive e097fa6`. All 44 additions were copied with apply_patch after input hash checks; authored files also used apply_patch. Only the new test was gofmt-formatted. No baseline or supplied file changed. No shared/frozen source writes, actual stores/backups/replicas, SSH/providers, room/remember writes, config, deployment, adoption or cleanup outside synthetic fixtures, or extra agents were used. Root's separate replica evidence and assertion bridge experiment were not read as candidate evidence or copied.

The exact test SHA-256 is `cc45df6cc2fcff8f0f5bdde71cd6b66c1bc8a83bff5cf811d43c27ba012ec3f0`. Complete new source/log/script hashes are in `EVIDENCE_SHA256.txt`; this report's hash is supplied separately.

NO-GO applies only to this finite private planner. It does not approve either FA phase, virtual backdate/hint eligibility, assertion identity/address/content preservation, support/materialization/B votes, final ownership closure, lifecycle/all writers, Force/current results, production lexical deduplication, ControllerV3 replacement, live adoption/prevention/deployment/cleanup, recall, binaries/performance/sweeps, or any full-goal clause or grading round. Those remain mandatory later work.
