# Sixth ordered identity overlay — independent bounded disproof

2026-09-06. NO-GO for this exact private, uncalled, read-only planner. Six fresh fixed-entry cases show that fact-only primary, status and hint consumers follow a listed alias while concealing malformed controls at its actual natural-slug endpoint. The latest correction validates that endpoint in proposal relationships, but those consumers do not invoke the relationship collector. This is a finite-contract failure; no support or FA implementation is required to demonstrate it.

Reviewed source: `/tmp/scry-ordered-overlay-sixth-sep06.IGAgjR/code`, baseline `e097fa6`, freeze SHA-256 `da175dd075a97cf46de4a2dd6724fa0c96131bf16af00fb4f0d71e84035224f7`. Controlling complete contract SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`. Both frozen and independent exports retain all 600 baseline SHA-256/git-blob pins and all 48 addition SHA-256/byte-length pins at start and end. All relative paths below refer to `/tmp/scry-overlay-sixth-disproof.lURntb`.

## Executable failure

`TestSixthFreshLookupAliasNaturalControls` creates Atlas(service), listing Atlas box, and Cygnus(service), through ordinary synthetic PutEntity fixtures. A second variant additionally creates an entity whose actual Slug is atlas-box but whose Name is Borealis and Type is machine. The exact synthetic legacy inventory is adopted before planning. The fixture then sets malformed `il:atlas-box` to `{"broken":true}`. In the first variant that natural identity is absent; in the second, the malformed bytes replace its anchor. Both variants are required malformed-control inputs, not invented recognized owners.

Each consumer uses only original facts, with no declarations or metadata proposals:

| Consumer | Original input | Result, in both endpoint variants |
| --- | --- | --- |
| primary | Atlas box uses Cygnus | nil error; nonzero plan with one assertion and zero decisions |
| status | Cygnus status Atlas box | nil error; nonzero plan with one assertion and zero decisions |
| hint | Atlas uses Cygnus, superseding Atlas box uses Cygnus | nil error; nonzero plan with one assertion and zero decisions |

All six must refuse with the sanitized planner sentinel and zero report. The unchanged supplied `independentOverlayError` helper invokes genuine `planIdentityEpisode` and first checks complete raw equality and zero emitted events. Those conservation checks pass in every failing leaf. The demonstrated failure is the planner's returned result, not durable mutation.

At `identity_overlay_lookup.go:113`, lookup checks whether the natural occupant's canonical name matches. An absent or differently named occupant falls through. At lines 127–133 it validates the explicit claim's owner Atlas, checks that Atlas lists the spelling, and returns `recognize(atlas)`. It never reaches the natural endpoint recognition at line 136 or missingControls below it. `recognize(atlas)` validates Atlas's canonical spelling and lifecycle, not atlas-box. The complete global captured inventory contains the malformed anchor, but global capture supplies neither validation nor the consumer's dependency witness.

The sixth correction is in `relationships`, reached by candidate/metadata/alias producers. A fact-only recognized endpoint or hint has no such producer, so that correction does not cover the actual fixed-entry route above. The malformed natural state cannot be made irrelevant merely because explicit alias precedence chooses a different target: the frozen contract explicitly preserves rehome precedence while requiring malformed relevant controls to remain errors.

Smallest required correction: validate and retain the relevant natural identity endpoint and its control dependencies in lookup consumers before the explicit alias shortcut succeeds, while keeping validation separate from route selection. Review equivalent lookup shortcuts as part of the same finite validation boundary. Do not replace a valid explicit rehome with the natural occupant, infer ownership, repair a retained listing or impose a global integrity refusal.

`TestSixthFreshValidRehomeAndUnrelatedControlsRetainUtility` is the positive boundary. It keeps the existing Atlas box alias, adds valid `rs:atlas-box` with no rt classification, and places malformed bytes only at `il:unrelated-box`. The ordinary primary still resolves to Atlas; the status becomes related_to Atlas; the hint still resolves to Atlas. Complete raw equality, zero events and no materialized candidates hold. This fixture establishes a valid retained control shape; it does not claim recovery of a historical repair manifest. Actual synthetic retirement/rehome API cases are additionally retained in the supplied tests.

## Whole-unit coverage

Session orientation ran first and was read. I read the complete active objective, sixth/fifth/fourth/third review inputs, controlling contract and embedded independent corrections, all five earlier complete NO-GO reports, delayed-birth design/review, ControllerV3 and identity-mutation reviews, and registration/episode-selector contracts. The selector's referenced design and independent design review were also read. A combined contract read was truncated and was followed by complete bounded ranged reads. One initial read guessed an absent third-input path; the correct exact referenced path was then read. Neither diagnostic changed files.

All six implementation files were read completely. I inspected the neutral projection/declaration/resolve/alias policy adapters and exact token verifier, original resolver declaration/primary/hint/alias paths, actual registration/serialized owner/finalizers, identity relationship inventory, identity writer history/mutation ledger, fact inventory/ledger, active legacy recognition and stored EP provenance. No separate assertion bridge or FA experiment was copied or graded here.

Every selected unchanged supplied planner test, all five prior independent disproof suites, ordered-contract and delayed-birth characterizations, and policy differential/adaptor checks pass. The prior 20, 4, 19, 4 and 7 failing leaves therefore remain fixed. Retained tests cover the latest derived-natural punctuation/control cases, exact malformed raw alias suffixes, valid unknown control extensions and unrelated rejection pairs.

Broader retained coverage includes original declaration/assertion slots and full observations; nil/present Supersedes and nil/empty input collections; inverse and value-source original-side maps; actual first discovery and later links; rejected-primary visibility to later primary and earlier hint records; typed failed-declaration persistence and independently trusted replacement; exact-natural, canonical-homonym and distinct-name explicit alias precedence; retired/consumed/unadopted/current/history typed deferrals with unrelated utility; metadata-only old identity utility and complete old repository references; no alias transfer; projection and released-claim producer chains; current episode vote deduplication and generation/legacy separation; actual stored EP validation even for current-input IDs; owned input/report buffers; real owner/phase/poison and discarded-transaction read boundaries.

The neutral verifier passes all 157 declarations / 248 symbols, with no extra copied declarations under its documented DTO/name substitutions. The retained differential reports 2,323 literals, 16 unary predicates, 18,084 alias projection cases and 264 declaration projection cases. The unchanged original resolver corpus and closed vocabulary pass. This establishes mechanical/adaptor parity in the retained corpus, not a new held-out identity/value semantic grade or permission to tune dictionaries. Production lexical deduplication remains later work.

Static inspection found no raw or ordinary entity/fact/alias/vote/EP writer in the six planner files. The fixed entry owns revision bytes before waiting, supplies its own program to genuine serialized registration, and accepts no selected owner, support flag or caller finalizer. Registration precedes new proposals; later original observations link issued handles. No active facade/handle is returned. The planner freeze checks empty writer histories; the actual wrapper validates final captured identity inventory and FA conservation. Fresh success/error-expected fixtures preserve full raw rows and zero events.

The retained zero-write Badger characterization passes: a stale zero-write Commit can return nil. No dummy write was introduced. This review covers this fixed program's absence of writers, snapshot/cooperative serialization and final captured-inventory conservation. It does not claim arbitrary raw-writer serialization, transient raw-write detection or production all-writer enforcement.

## Commands and retained evidence

All Go commands, including gofmt for the new test, used `CGO_ENABLED=0` and `TMPDIR=/tmp/scry-overlay-sixth-disproof.lURntb/test-tmp`. Tests/checks retained complete stdout/stderr through tee under `set -o pipefail`.

| Command / log | Result |
| --- | --- |
| `ruby verify-pins.rb copy` → `initial-pins.log` | PASS both exports; 600 baseline SHA/blob pins and 48 additions verified before/after apply_patch copying |
| `go test ./internal/memory/store -run TestSixthFresh -count=1 -v` → `fresh-tests.log` | Initial six malformed-control leaves FAIL; store 0.636s |
| `go test ./internal/memory/store ./internal/memory/resolve -run 'TestIdentityOverlay\|TestIndependentOverlay\|TestCorrectedFresh\|TestThirdFresh\|TestFourthFresh\|TestFifthFresh\|TestOrderedContract\|TestDelayedBirth\|TestPrivateIdentityPolicyPort' -count=1 -v` → `retained-tests.log` | PASS store 12.909s / resolve 2.370s |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` → `vet.log` | PASS; empty output |
| `go run /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/verify_policy_port.go . /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/policy-inventory-corrected.json internal/memory/identitypolicy` → `policy-tokens.log` | PASS 157 declarations / 248 symbols |
| `go test ./... -count=1` → `full-suite.log` | FAIL only in the six new leaves; store 73.889s. All other packages PASS, including resolve 17.293s / daemon 29.044s |
| `go test ./internal/memory/store -run TestSixthFresh -count=1 -v` → `fresh-final.log` | Same six leaves FAIL; fresh valid-rehome/unrelated-control boundary PASS; store 0.593s |
| `ruby verify-pins.rb` → `final-pins.log` | PASS both complete source exports, contract, current input, all five previous reports and exact neutral verifier/manifest |

No executed test had a compile failure or fixture correction. The initial failing source remains `fresh-initial-source.go.txt`; subsequent source only adds the positive rehome/unrelated-control test. All initial assertions remain unchanged. All failed logs remain. Once this concrete finite NO-GO was established, I completed bounded required coverage and reporting without further counterexample expansion.

The export was created by fresh `git archive e097fa6 | tar`. All 48 additions were copied via apply_patch after exact checks. Every authored test/script/report/index uses apply_patch; gofmt touched only the new test. No supplied/baseline/frozen/shared files were edited. No actual stores/backups/replicas, SSH/provider calls, room/remember writes, configuration, deployment, cleanup outside synthetic test fixtures or additional agents were used. The root full-suite tail was inspected only as background, not used as the independent grade.

Test SHA-256: `4299de2f24461d3b9d7cb3fbaa8bae88bf8254d5aeed20d3cf3e38369147b328`. Complete new test/source/log/script hashes are in `EVIDENCE_SHA256.txt`; report hash is returned separately.

NO-GO applies only to this finite private planner. It does not approve FA phases or virtual backdate/hint eligibility, assertion identity/address/content/interval preservation, support/materialization/B votes, final ownership closure, lifecycle/all writers, Force/current results, production lexical deduplication, ControllerV3 replacement, adoption/prevention/deployment/live cleanup, recall, binaries/performance/sweeps or any full-goal clause or grading round. Those remain separate mandatory later obligations.
