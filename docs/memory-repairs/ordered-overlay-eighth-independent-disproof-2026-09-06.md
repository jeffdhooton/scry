# Eighth ordered identity overlay — independent bounded disproof

2026-09-06. NO-GO for the exact private, uncalled, read-only planner. Eight fresh fixed-entry cases return the correct preexisting-reference deferral but omit the actual current/historical fact responsible for it from the route's dependencies and the complete witness map. This violates the finite descriptive dependency contract. It is not a demand to implement FA planning, support selection or production integration before grading this unit.

Reviewed source: `/tmp/scry-ordered-overlay-eighth-sep06.8lun4B/code`, baseline `e097fa6`, freeze SHA-256 `fab44ba27d8922d657d33da54840187651268a2523454dbf2231544792896de7`. Review input SHA-256 `98738150bae67f9be53d9800aca448a49f755aefd6d975e3fe8ca7a87ca22b3b`; complete controlling contract SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`. Both frozen and independent exports retain all 600 baseline SHA-256/git-blob pins and all 51 addition SHA-256/byte-length pins at start and end. Paths below are relative to `/tmp/scry-overlay-eighth-disproof.Gz9DvD` unless absolute.

## Executable counterexample

`TestEighthFreshReferenceVetoWitness` seeds Atlas(service) and Cygnus(service), then one exact well-formed retained FA row `polar-manual uses atlas`, while `en:polar-manual` is absent. The row contains its original sentence, confidence, validity and episode provenance. Half the cases use a current row; the other half use an invalidated historical row. The complete synthetic legacy inventory is adopted before the planner call. The genuine fact inventory decoder accepts every fixture.

The planner receives only full original input through unchanged `overlayTestPlan`, which invokes `planIdentityEpisode`. Every input includes the unrelated useful `Atlas uses Cygnus` assertion. Four consumers are exercised for each reference state:

| Consumer | Additional input | Correct returned route |
| --- | --- | --- |
| declaration | Polar Manual(service) | deferred / preexisting-reference / polar-manual |
| primary | Polar Manual uses Cygnus | deferred / preexisting-reference / polar-manual |
| status | Atlas status Polar Manual | deferred / preexisting-reference / polar-manual |
| hint | Atlas uses Cygnus supersedes Polar Manual uses Cygnus | deferred / preexisting-reference / polar-manual |

All eight routes omit the actual FA key from their Trace.Rows and from `plan.Witnesses`. Their trace producer lists are empty, so no producer chain supplies the missing evidence. The assertion permits transitive producer representation and checks full stored bytes, not merely a count or digest. It fails with `witness=false positive=false trace-direct=false producer-count=0` in every leaf.

The shared helper independently checks full raw equality, zero emitted events, exact owned revision bytes, complete original slots, empty actual identity/fact writer histories and no materialized candidates. Those checks pass in every leaf, as does the unrelated useful assertion. The demonstrated defect is missing descriptive evidence; no durable mutation or wrong selected owner is demonstrated.

`TestEighthFreshUnrelatedReferenceDoesNotVeto` passes: retained facts referring only to other absent identities do not prevent a new Polar Manual primary from resolving. This preserves the boundary between relevant identity-veto evidence and unrelated baseline rows.

## Cause, contract and correction

At `internal/memory/store/identity_overlay_controls.go:361`, `missingControls` consumes `p.registry.facts.baseline.References[slug]`. A nonzero Current or Historical count produces `preexisting-reference`. Neither that branch nor its callers read the corresponding FA row into a witness or identify a consumed reference-inventory scope in the trace.

The genuine `scanIdentityReferenceInventory` validates FA rows and computes global counts/digest. It does not retain their raw bytes. `identityFactLedgerReport` returns final aggregate inventory plus mutations; mutations are empty here. Thus the returned report has a global reference count and digest, but not the specific observed row on which this route depends. The complete captured identity inventory does not include FA rows either. Global conservation accounting is not a per-decision dependency witness.

This is inside the frozen identity planner's scope: the contract requires complete baseline fact-veto views, exact baseline controls and positive/negative witnesses, and explicit dependencies for every route. The reference check already participates in identity recognition; this review does not ask the overlay to choose exact supersession targets, simulate a temporal phase or execute facts. The earlier seven reviews applied the same inventory-versus-decision distinction to entity/control rows. The eighth missing-owner correction passes its retained tests, but does not address this separate consumed evidence.

Smallest required correction: preserve exact reference evidence and identify it in each decision that consumes the reference query, maintaining current versus historical distinctions and owned bytes. Audit both actual consumers: the proved `missingControls` branch and the separately observed current-reference alias veto at `identity_overlay_aliases.go:120`. The latter was inspected but is not an additional executable finding in this report. A complete zero-reference query also needs an explicit bounded scope/absence representation; an arbitrary absent FA key cannot prove no reference exists. The report does not prescribe one storage representation or require importing later FA planning into this unit. Keep all existing typed outcomes, unrelated utility, read-only behavior and no-owner-transfer rules.

The lead acknowledged this concrete evidence gap while this review was finishing. That acknowledgment is not an implementation correction or evidence of a successor's correctness. The frozen source remained unchanged.

## Whole-unit coverage and limits

Session orientation ran first and was read. I read the complete active objective, eighth through third review inputs, complete controlling contract and embedded independent corrections, all seven previous complete NO-GO reports, delayed-birth design/review, ControllerV3 and identity-mutation independent reviews, registration contract and episode-selector contract, and the selector's referenced complete design/review. Oversized initial combined reads were followed by separate/ranged reads for omitted content.

All six overlay implementation files were read completely. I inspected neutral projection/declaration/resolve/alias adapters and the exact token verifier; original resolver declaration, primary, hint and endpoint paths; actual registration, serialized owner/coordinator and fixed finalizers; complete identity relationship/reference inventories and identity/fact mutation and writer ledgers; active legacy recognition and actual EP provenance. Separate complete-admission WIP and the rejected assertion bridge were not copied or graded.

Every selected unchanged supplied planner test, all seven prior independent disproof suites, ordered-contract and delayed-birth characterizations, and policy differential/adaptor checks pass. All 64 previous independently failing leaves therefore remain fixed. The latest absent explicit-claimant and present-unlisted witness pairs pass unchanged. Retained relationship tests cover derived natural endpoints, counterpart listings, exact malformed raw alias suffixes, relevant paired retirement/rejection controls, valid unknown extensions, and unrelated opaque controls.

Broader retained coverage includes complete original declaration/assertion slots and observations; nil/present Supersedes and nil/empty inputs; inverse and value-source original-side maps; actual first-discovery and later links; rejected-primary visibility to later primaries and earlier hints; typed failed declarations and independently trusted replacement; exact-natural, canonical-homonym and distinct-name explicit alias precedence; retired/consumed/unadopted/current/history deferral with unrelated utility; metadata-only old identity utility and preservation of all old repository references; no alias transfer; metadata/removal/released-claim producer chains; current-episode vote deduplication and legacy/generation isolation; actual stored EP evidence including matching-current input IDs; owned inputs/reports; genuine owner/phase/poison and discarded-transaction read boundaries.

The exact neutral verifier passes all 157 declarations / 248 symbols under only its documented DTO/name substitutions, with no extra copied declarations. The retained differential reports 2,323 literals, 16 unary predicates, 18,084 alias projection cases and 264 declaration projection cases. The unchanged original resolver and closed vocabulary corpora pass in the full suite. This establishes mechanical/adaptor parity within the retained corpus, not a new held-out identity/value quality grade or permission to tune lexical dictionaries. Production deduplication remains later work.

Static inspection finds no raw or ordinary entity/fact/alias/vote/EP writer in the six planner files. The fixed entry owns input before waiting and supplies its own program to genuine serialized registration; it accepts no selected owner, support flag or caller finalizer. Registration precedes candidate proposals; later observations link issued handles. No active facade/handle is returned. The fixed freeze checks empty writer histories, and the real wrapper verifies final captured identity inventory and FA conservation. Fresh fixtures preserve all raw rows and emit zero events.

The retained Badger characterization passes: a stale zero-write Commit can return nil. No dummy write was introduced. This review covers this fixed program's absence of writers, snapshot/cooperative serialization and final captured-inventory conservation. It does not claim arbitrary raw-writer serialization, transient raw-write detection or production all-writer enforcement.

## Commands and evidence

All Go commands, including gofmt for the sole new test, used `CGO_ENABLED=0` and `TMPDIR=/tmp/scry-overlay-eighth-disproof.Gz9DvD/test-tmp`. Test/check stdout and stderr were retained through tee under pipefail. Full suite, retained suite and vet ran concurrently; their durations are not performance measurements.

| Command / log | Result |
| --- | --- |
| `ruby verify-pins.rb copy` → `initial-pins.log` | Both exports: 600 baseline SHA/blob pins and 51 additions verified around apply_patch copying |
| `go test ./internal/memory/store -run TestEighthFresh -count=1 -v` → `fresh-tests.log` | Initial fixture setup FAIL: ordinary PutFact correctly refuses missing source endpoints; no planner finding yet; store .778s |
| Same command after only seed correction → `fresh-corrected-fixture.log` | Eight exact-reference dependency leaves FAIL; unrelated-reference boundary PASS; store .746s |
| Retained planner/prior disproof/characterization/differential selection → `retained-tests.log` | PASS store 22.835s / resolve 1.998s |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` → `vet.log` | PASS, empty output |
| Exact `verify_policy_port.go` with `policy-inventory-corrected.json` → `policy-tokens.log` | PASS 157 declarations / 248 symbols |
| `go test ./... -count=1` → `full-suite.log` | FAIL only in the same eight new leaves; store 75.632s. All other packages PASS, including resolve 22.148s / daemon 36.076s |
| `ruby verify-pins.rb` → `final-pins.log` | Both complete source exports and 11 exact external input/contract/report/verifier pins PASS |

The initial fixture source is retained unchanged as `fresh-initial-fixture-source.go.txt`, alongside its complete failed log. Ordinary PutFact does not create dangling references. I corrected only those new seeds to exact raw FA fixture rows, the same synthetic legacy-state technique used by the supplied current/history tests. No assertion was changed, no supplied test or source was modified, and no compile failure occurred. All subsequent tests use that final source. Once the concrete finite NO-GO was proved, I completed required coverage/reporting without expanding the counterexample corpus.

The export was created using fresh `git archive e097fa6 | tar`. All 51 additions and all authored test/script/report/evidence files use apply_patch; gofmt touched only the new test. One read-only filename guess (`identity_fact_mutation.go`) was absent; the actual fact ledger was located and read. No shared/frozen/supplied/baseline source edits, actual stores/backups/replicas, SSH/providers, room/remember writes, configuration, deployment, cleanup outside synthetic fixtures or additional agents were used. Root's full-suite completion and corrected-tests durations were inspected as background only, never used as this independent grade.

New test SHA-256: `7aca1ce6651ae00359d7e999f0ef322ef3b0682f2a187557da0cba6a816f4ac7`. Full test/log/script hashes are in `EVIDENCE_SHA256.txt`; report hash is returned separately.

NO-GO applies only to this finite private planner. It does not approve FA phases/virtual backdate/hint eligibility, assertion identity/address/content/interval preservation, support/materialization/B votes, final ownership closure, lifecycle/all writers, Force/current results, production lexical deduplication, ControllerV3 replacement, adoption/prevention/deployment/live cleanup, recall, binaries/performance/sweeps or any full-goal clause or grading round. Those remain mandatory later obligations.
