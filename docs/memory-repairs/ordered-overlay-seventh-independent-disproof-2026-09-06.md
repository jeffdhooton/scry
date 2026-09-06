# Seventh ordered identity overlay — independent bounded disproof

2026-09-06. NO-GO for the exact private, uncalled, read-only planner. Four fresh fixed-entry cases show that a missing explicit alias claimant's entity absence is consumed by declaration, primary, status and hint routing but is absent from both the route trace and the planner's witness map. The resulting typed deferral and unrelated utility are otherwise correct. This is a finite descriptive-dependency contract failure; it requires neither factual support nor production integration to reproduce or correct.

Reviewed source: `/tmp/scry-ordered-overlay-seventh-sep06.x8Wcyi/code`, baseline `e097fa6`, freeze SHA-256 `c2a28e299466adda29da2bafb22b48276a8eead24cc470ea3b4fbf9465a5264e`. Complete controlling contract SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`. Both frozen and independent exports retain all 600 baseline SHA-256/git-blob pins and all 50 addition SHA-256/byte-length pins at start and end. All relative paths below refer to `/tmp/scry-overlay-seventh-disproof.78QIXK`.

## Executable counterexample

`TestSeventhFreshClaimantEntityWitness` creates Atlas(service) and Cygnus(service) through ordinary synthetic PutEntity fixtures. Actual `ClaimAlias("Polar Manual", "old-owner")` creates a retained claim to an absent owner. The exact synthetic legacy inventory is adopted before planning. An unrelated malformed `il:unrelated-owner` is then added to verify that the planner retains normal local utility instead of imposing global integrity refusal.

Each test supplies complete original input to genuine `planIdentityEpisode` through the unchanged supplied `overlayTestPlan` helper. Each input retains the independent Atlas uses Cygnus assertion. Four consumers exercise the missing claimant:

| Consumer | Additional original input | Actual returned route |
| --- | --- | --- |
| declaration | Polar Manual(service) | deferred / missing-alias-owner, Slug old-owner |
| primary | Polar Manual uses Cygnus | deferred / missing-alias-owner, Slug old-owner |
| status | Atlas status Polar Manual | deferred / missing-alias-owner, Slug old-owner |
| hint | Atlas uses Cygnus supersedes Polar Manual uses Cygnus | deferred / missing-alias-owner, Slug old-owner |

All four preserve `al:polar-manual`, `il:old-owner`, `il-consumed:old-owner`, `ig:old-owner` and `rs:old-owner` dependencies. All four omit `en:old-owner` from their respective Trace.Rows and from the entire Witnesses map. The tests require an explicit witness with Exists=false; absence of a map entry is not an absence witness. No producer chain can supply this missing row because the global witness map does not contain it either.

The counterexample uses a normal admitted legacy fixture and an actual low-level historical claim API, not a caller-selected recognized owner or forged report. There is no malformed relevant lifecycle row in these four failing routes. The required correction is to preserve the missing entity evidence, not to turn this legitimate typed deferral into an error. The helper checks complete raw equality, zero emitted events, exact owned revision bytes, complete original slots, zero actual identity/fact writer history and no materialized candidates. Those checks pass in every leaf. The unrelated assertion remains proposed-resolution. No durable planner write or incorrectly selected owner is demonstrated.

The paired positive cases additionally create `Slug: old-owner, Name: Borealis, Type: machine`, without listing Polar Manual, before adoption. All four consumers then return deferred / unlisted-alias-owner, and all required dependencies including the present `en:old-owner` witness are retained. These four leaves pass. The absent/present pair demonstrates the concrete entity-existence branch on the same explicit claim, while preserving typed deferral and unrelated utility in both states.

## Cause and smallest correction

At `identity_overlay_lookup.go:72`, `claim` branches on `p.entities[c.owner]`. The present branch calls `recognize`, which reads and records en:owner. The absent branch at line 76 calls `missingControls`, which validates lifecycle controls and derived retirement but never records en:owner. At lines 164–176, `lookup` again consumes the same existence distinction to choose missing-alias-owner versus unlisted-alias-owner. Because old-owner differs from the mention's natural slug polar-manual, neither the new naturalRoute gate nor the stale-self-claim relationship path supplies the missing row.

The seventh correction accurately adds natural endpoint evidence. It does not cover the distinct explicit claimant endpoint in this counterexample. The full registry identity inventory permits the planner to calculate the absence internally, but it does not identify that absence as the dependency of this particular routing decision. The frozen contract requires positive and negative witnesses and exact per-route dependencies; complete global capture cannot substitute for those records.

Smallest correction: retain the claimed identity's exact entity existence/absence witness whenever its existence participates in claim routing, including normal missing-owner deferral. Keep its existing complete lifecycle validation, typed outcome and explicit-natural/rehome precedence. Review equivalent missing-endpoint consumers within this same finite witness boundary. Do not select a replacement owner, repair the claim, add factual support, introduce a writer or globally refuse unrelated controls.

## Whole-unit coverage and limits

Session orientation ran first and was read. I read the complete active objective, seventh/sixth/fifth/fourth/third review inputs, controlling contract and embedded independent corrections, all six previous complete NO-GO reports, delayed-birth design/review, ControllerV3 and identity-mutation independent reviews, registration contract and episode-selector contract. The selector's referenced complete design and independent design review were also read. Initial oversized combined reads were followed by separate complete/ranged reads for the affected documents.

All six implementation files were read completely. I inspected the neutral projection/declaration/resolve/alias policy adapters and exact token verifier; original resolver declaration, primary, hint and alias paths; actual registration, serialized owner/coordinator and fixed finalizers; identity relationship inventory and mutation/writer ledgers; fact inventory and ledger; active legacy recognition and stored EP provenance. The separate rejected assertion bridge and FA experiment were not copied or used as candidate evidence.

Every selected unchanged supplied planner test, all six prior independent disproof suites, ordered-contract and delayed-birth characterizations, and policy differential/adaptor checks pass. Thus all 60 preceding failing leaves remain fixed on this exact source. The retained run includes the seventh root's derived-punctuation controls and natural endpoint witnesses, sixth natural identity controls, fifth spelling/listing expansion, exact defective raw alias suffixes, valid unknown extensions and unrelated rejection-pair boundaries.

Broader retained coverage includes complete original declaration/assertion slots and nil/present Supersedes; nil/empty input collections; inverse/value-source original-side maps; actual first discovery and later links; rejected-primary visibility in later primary and earlier hint interpretation; typed failed-declaration persistence and independent trusted replacement; exact-natural, canonical-homonym and distinct-name explicit alias precedence; retired/consumed/unadopted/current/history deferrals with unrelated utility; metadata-only old identity usefulness and full old repository references; alias no-transfer rules; metadata/removal/released-claim producer chains; current episode vote deduplication and generation/legacy evidence isolation; actual stored EP validation including current-input IDs; owned inputs/reports; genuine owner/phase/poison and discarded-transaction read refusal.

The exact neutral verifier passes all 157 declarations / 248 symbols under its documented DTO/name substitutions, with no extra copied declarations. The retained differential reports 2,323 literals, 16 unary predicates, 18,084 alias projection cases and 264 declaration projection cases. The unchanged original resolver corpus and closed vocabulary pass in the complete suite. This establishes mechanical/adaptor parity within the retained corpus, not a new held-out identity/value semantic grade or permission to tune dictionaries. Production lexical deduplication remains later work.

Static inspection finds no raw or ordinary entity/fact/alias/vote/EP writer in the six planner files. The fixed entry owns revision bytes before waiting and supplies its own program to genuine serialized registration. It accepts no selected owner, support flag or caller finalizer and returns no active facade/handle. Registration precedes new proposals; later observations link issued handles. The fixed freeze checks empty writer histories; the actual wrapper validates complete final captured identity inventory and fact conservation. The new fixtures independently preserve all raw rows and zero events.

The retained zero-write Badger characterization passes: a stale zero-write Commit can return nil. No dummy write was introduced. This review covers the fixed program's absence of writers, snapshot/cooperative serialization and final captured-inventory conservation. It does not claim arbitrary raw-writer serialization, transient raw-write detection or production all-writer enforcement.

## Commands and evidence

Every Go command, including gofmt for the sole new test, used `CGO_ENABLED=0` and `TMPDIR=/tmp/scry-overlay-seventh-disproof.78QIXK/test-tmp`. Complete stdout/stderr from tests and checks was retained through tee under `set -o pipefail`. Independent full-suite, retained-suite and vet runs were started in parallel; their durations are not performance measurements.

| Command / log | Result |
| --- | --- |
| `ruby verify-pins.rb copy` -> `initial-pins.log` | PASS both exports: 600 baseline SHA/blob pins and 50 addition SHA/byte pins verified around apply_patch copying |
| `go test ./internal/memory/store -run TestSeventhFresh -count=1 -v` -> `fresh-tests.log` | Four missing-claimant leaves FAIL, four present-unlisted boundaries PASS; store 1.074s |
| `go test ./internal/memory/store ./internal/memory/resolve -run 'TestIdentityOverlay\|TestIndependentOverlay\|TestCorrectedFresh\|TestThirdFresh\|TestFourthFresh\|TestFifthFresh\|TestSixthFresh\|TestOrderedContract\|TestDelayedBirth\|TestPrivateIdentityPolicyPort' -count=1 -v` -> `retained-tests.log` | PASS store 35.279s / resolve 2.266s |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` -> `vet.log` | PASS; empty output |
| `go run /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/verify_policy_port.go . /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/policy-inventory-corrected.json internal/memory/identitypolicy` -> `policy-tokens.log` | PASS 157 declarations / 248 symbols |
| `go test ./... -count=1` -> `full-suite.log` | FAIL only in the same four new leaves; store 103.709s. All other packages PASS, including resolve 31.694s / daemon 47.601s |
| `ruby verify-pins.rb` -> `final-pins.log` | PASS both complete exports plus exact contract, input, six previous reports and neutral verifier/manifest |

No new test compile failure, fixture correction, source revision or assertion weakening occurred. The exact initial formatted test is the final test; both failed logs remain. Once the concrete finite NO-GO was proved, I completed the required bounded coverage/report without expanding the counterexample corpus.

The export was created with fresh `git archive e097fa6 | tar`. All 50 additions were copied via apply_patch after exact checks. Every authored test/script/report/evidence index uses apply_patch; gofmt touched only the new test. No supplied, baseline, frozen or shared file was edited. No actual stores/backups/replicas, SSH/providers, room/remember writes, configuration, deployment, cleanup outside synthetic fixtures or additional agents were used. The root full-suite tail was inspected as background only; it is not this independent grade.

New test SHA-256: `b3bd87b03474627e908ccc3aa6f153df0f2dd5d619b1da1a8046b1af994e6fe5`. Complete test/log/script hashes are in `EVIDENCE_SHA256.txt`; the report hash is returned separately.

NO-GO applies only to this finite private planner. It does not approve FA phases/virtual backdate/hint eligibility, assertion identity/address/content/interval preservation, support/materialization/B votes, final ownership closure, lifecycle/all writers, Force/current results, production lexical deduplication, ControllerV3 replacement, adoption/prevention/deployment/live cleanup, recall, binaries/performance/sweeps or any full-goal clause or grading round. Those remain separate mandatory later obligations.
