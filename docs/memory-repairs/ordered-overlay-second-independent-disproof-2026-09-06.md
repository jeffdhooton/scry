# Corrected ordered identity planner — independent code disproof

2026-09-06. **NO-GO for the exact frozen private, uncalled, read-only unit.** Two finite-contract violations remain: valid retirement classification at a derived slug causes whole-plan refusal, and type-driven alias removal omits its projection producer dependency. Three fresh failing groups contain four failing leaves. These require corrections to this planner, not another foundation primitive or implementation of factual support merely to grade it.

The reviewed baseline is `e097fa6cb2e5c5ff8afcbd2b21c163d879856d84`. Corrected manifest `/tmp/scry-ordered-overlay-correction-sep06.soHIfk/OVERLAY_CORRECTED_CODE_FREEZE.json` is SHA-256 `9e44b47b16aaf6ac31c88af1347205789bbe873001e463ed76c5696b38331463`. The controlling full contract is SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`; the previous complete NO-GO report is SHA-256 `e42c9196efdfb07491bd53cbcdc981366ddf275694504882ce8ef602fc437a22`. Initial and final verification found all 600 baseline SHA-256/git-blob hashes and 39 addition SHA-256/byte sizes unchanged in both the frozen source and my export.

All relative paths below refer to `/tmp/scry-overlay-corrected-disproof.VyluEa`.

## 1. Valid derived-slug retirement classification refuses unrelated records

`TestCorrectedFreshRetiredCanonicalSlugDoesNotRefuseUnrelatedRecords` places an identical valid retirement payload at `rs:old-verdict` and `rt:polar-manual`. Such a pair means Polar Manual was a reviewed spelling of the removed old-verdict identity. The incoming literal `Polar Manual!` has normalized spelling `polar-manual!` and derived slug `polar-manual`.

The literal itself is not classified by `rt:polar-manual`, so it must not silently become that exact spelling's VALUE decision. But its required candidate route is occupied by a valid retirement control. The planner must retain a typed local dependency deferral and preserve unrelated useful records. Both primary-only and declaration-plus-primary cases instead return the sanitized whole-plan error and a zero report, also hiding the unrelated `Atlas uses Draco` assertion.

The actual path is `lookup` -> `missingControls` -> `candidate` -> registry `register`. `identity_overlay_controls.go:325` checks lifecycle records and FA references for the derived slug, but not its `rt:` classification. The genuine unchanged registration guard at `identity_birth_registration.go:249` then finds `rt:` plus `Normalize(b.Slug)` and correctly refuses registration. The overlay omitted the typed preflight that would prevent reaching this structural error with a normal valid-control conflict.

`TestCorrectedFreshActualRetirementAliasThenAlternateBirth` proves the same state through actual synthetic APIs: create old-verdict with canonical Retired Verdict and alias Polar Manual; preview and execute `RetireEntity` with its exact expected snapshot; adopt the remaining empty synthetic inventory; then plan the alternate literal and unrelated assertion. The actual retirement writer creates the matching `rt:polar-manual` and `rs:old-verdict` rows. Planning still refuses the whole episode. No fabricated lifecycle bytes are needed for this stronger reproduction.

All three failing cases verify raw equality and zero emitted events during the planner call. This is a partial-utility and typed-control failure, not durable corruption by this read-only unit.

Required correction: preflight the full relevant derived-birth control set before registration. A valid derived-spelling conflict must remain a local typed deferral; malformed relevant controls must still be sanitized errors. Preserve exact-spelling VALUE classification, valid exact-natural precedence and existing validated reviewed rehomes. Do not infer a recipient or weaken the registry's structural guard.

## 2. Type-driven alias removal lacks its metadata dependency

`TestCorrectedFreshMetadataProjectionDependencies/upgrade-revalidation` establishes adopted baseline Atlas(concept) with alias Atlas box, then declares Atlas(service) with its first description. The planner records metadata decision 1 changing concept to service, followed by alias-removal decision 2. Decision 2's Before image consumes decision 1's service projection; that service type is what makes Atlas box fail the existing lexical leak predicate. Nevertheless decision 2 has `Trace.Producers=[]`, with no direct or transitive dependency on decision 1.

`identity_overlay_declarations.go:181` installs the type upgrade and immediately calls `revalidate`. `identity_overlay_aliases.go:168` reads the new projection, but line 171 only clones the old parent trace. The immediate leak-removal branch reaches line 202 without recognizing/recording the current actor producer. The frozen report therefore describes a removal dependent on metadata it does not identify as a producer dependency. Merely listing the two decisions in order does not fulfill the contract's explicit exact projection/producer dependency requirement.

The fresh test checks that each decision consuming a previous proposed actor image can reach its producer through the recorded dependency graph. It permits transitive dependencies and does not require one special direct-edge representation. Its ordinary-new-candidate case passes. The upgrade/revalidation case fails. Raw rows and events remain unchanged through the shared read-only test helper.

Required correction: record the exact current actor projection producer before evaluating and recording revalidation proposals, and retain that dependency through freeze. Audit other proposal transitions for the same completeness property. This is descriptive dependency accounting only; no factual support solver or final ownership decision is requested.

## Coverage beyond the two counterexamples

I read session orientation first; the complete active goal, corrected review request, full controlling contract and embedded corrections, complete previous NO-GO report, delayed-birth design/review, ControllerV3 and identity-mutation reviews, registration and selector contracts. I inspected all six planner implementation files, neutral DTO adapters and declaration-token verifier, original resolver declaration/primary/hint/alias behavior, registry/owner/finalizer, identity/fact inventories and ledgers, legacy/generation recognition/provenance, and relevant retirement analysis/writer behavior.

The original twenty independently failing leaves now pass unchanged. The supplied broader correction tests also pass: malformed missing/unlisted claimed owners are validated; absent alias natural identities defer on retirement/current/history controls; stored evidence with the current input's ID must validate an actual EP; a valid current stored EP remains one deduplicated vote with its witness; unresolved missing/unlisted claims cannot become status or hint values; stale self-claims retain the reason and unlisted-key witnesses.

Fresh passing cases add:

- Exact-natural identity precedence over valid missing and unlisted stale alias claims, including status-to-related_to and supersedes lookup.
- Absent compact alias identities with malformed anchors, valid retirement or historical FA references. Malformed controls error; ordinary retirement/reference cases remain alias deferrals.
- Ordinary new-candidate metadata producer-chain completeness, contrasting with the specific failing upgrade/revalidation transition.

The unchanged supplied suites additionally pass original declaration/fact slot preservation, nil/present SupRef and complete original observations, inverse plus value-source inversion, actual first-birth order and later mention links, rejected primary discovery visibility, hints after all primary discoveries, typed failed-declaration binding replacement, canonical homonyms versus exact-natural and distinct-name explicit alias precedence, candidate order/cache refresh, old metadata without same-episode facts and complete old repo refs, no other-owner transfer, generation/legacy vote isolation, malformed relevant lifecycle/provenance, owned reports/input and reopen, 102 shortcut phase/poison probes, closed/wrong scope handling and actual discarded-transaction read errors. The eight delayed-birth and six ordered-contract characterizations remain unchanged and pass; their later FA-phase/content-preservation defects remain baseline characterizations, not overlay fixes.

The neutral port's read-only token verifier passes all 157 manifest declarations / 248 symbols with only its documented DTO/name substitutions and no extra copied declarations. I read the verifier and adapter source. The original resolver corpus and retained differential/adaptor tests pass, including the unchanged 39-relation vocabulary. This establishes mechanical policy parity within these tests; it is not an independent status/name semantic-quality grade or permission to add word rules. The differential corpus reports 2,323 literals, 16 unary predicates, 18,084 alias projection cases and 264 declaration projection cases.

Static inspection found no raw/normal entity, fact, EP, alias or vote writer in the six planner files. The fixed entry owns canonical bytes before waiting, supplies its own program to the genuine serialized registration wrapper, and exposes no chosen owner, support flags, caller callback/finalizer, active facade or handle. Registration precedes new proposal creation; later mentions use links. The final inventories and empty actual writer histories preserve the captured families. Fresh success/error fixtures preserve all raw rows and zero events.

The passing zero-write Badger characterization remains bounded: a stale zero-write Commit returns nil. No dummy write was added, and no actual zero-write commit-conflict guarantee is claimed. Snapshot/cooperative serialization, final captured inventory equality, and this program's absence of writers are the relevant boundary; they do not prove arbitrary untracked transient-write detection or production all-writer enforcement.

## Commands, retained failures and scope

All commands ran in my fresh `git archive e097fa6` export with `CGO_ENABLED=0`. Go test/vet stdout and stderr were retained through `tee` under `set -o pipefail`.

| Command / retained log | Result |
| --- | --- |
| `ruby verify-pins.rb` -> `initial-pins.log`, `final-pins.log` | PASS: 600 baseline SHA-256/git blobs and 39 candidate SHA-256/sizes in both exports; exact contract, manifest and previous report pins. |
| `go test ./... -count=1` -> `supplied-full-suite.log` | Frozen candidate plus every supplied test PASS before fresh tests; store 65.345s, resolve 16.548s, daemon 27.305s. |
| `go test ./internal/memory/store -run TestCorrectedFresh -count=1 -v` -> `fresh-initial.log` | Two derived-slug leaves FAIL; exact-natural and compact-control cases PASS; store 0.697s. |
| Same command -> `fresh-expanded.log` | Final five fresh groups: three groups/four leaves FAIL, six leaves PASS; store 0.768s. |
| `go test ./internal/memory/store ./internal/memory/resolve -run 'TestIdentityOverlay\|TestIndependentOverlay\|TestOrderedContract\|TestDelayedBirth\|TestPrivateIdentityPolicyPort' -count=1 -v` -> `retained-tests.log` | Every selected unchanged supplied test and characterization PASS. |
| `go run /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/verify_policy_port.go . /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/policy-inventory-corrected.json internal/memory/identitypolicy` -> `policy-token-verification.log` | PASS, exact 157 declarations / 248 symbols. |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` -> `independent-vet.log` | PASS with no output. |
| `go test ./... -count=1` -> `final-full-suite.log` | FAIL only in the three fresh store groups/four leaves; store 67.767s. Other packages PASS, resolve 17.836s, daemon 28.480s. |
| Fresh tests with explicit contained TMPDIR -> `fresh-contained.log` | Identical four failing leaves and six passing leaves; store 0.755s. |
| `go test ./... -count=1` with explicit contained TMPDIR -> `final-contained-full-suite.log` | FAIL only in the same three fresh store groups/four leaves; store 65.871s. All other packages PASS, resolve 16.531s, daemon 27.061s. |

An initial shell search had an unmatched filename glob; it changed no files and was replaced by an exact search. No new Go test compile mistake or fixture correction occurred. Fresh source expansion only added the actual-retirement and dependency cases; every initial assertion is unchanged. The initial three-group source is retained as `fresh-initial-source.go.txt`, alongside all failed logs and final source. No candidate implementation or supplied test was edited or assertions weakened.

Scope correction: initial Go runs inherited the system TMPDIR, so their auto-cleaned synthetic `t.TempDir` stores were physically outside the export despite being created solely by my private-export tests. No real store was opened. I disclosed this to the lead and repeated final evidence with `TMPDIR=/tmp/scry-overlay-corrected-disproof.VyluEa/.test-tmp.yw0uV7`, placing runtime test stores under my own export. Earlier runs remain retained rather than relabeled.

All candidate additions, authored tests, script, source snapshot and report were created using apply_patch; gofmt only formatted the new test. No shared/frozen source writes, real stores/backups/replicas, providers, SSH, live adoption/deployments/repairs, memory/room writes, configuration changes or additional agents were used. Synthetic adoption and retirement merely established fixtures.

## Pins and verdict limits

New test `internal/memory/store/overlay_corrected_fresh_test.go`: SHA-256 `54dd58d56c9c4111850020c02aef8c089cf95236ee13eaa38f0b730420b231fb`. Initial source snapshot: `7b4c67fe8090664cf757cf46781cdbd422b0048db29f36bc69a30050f4337612`. Exact source/log/script hashes are retained in `EVIDENCE_SHA256.txt`; the report hash is returned separately. `final-pins.log` includes the complete exact candidate source hash list for both exports.

NO-GO applies to this exact private unit. Neither this review nor any subsequent bounded PASS approves either FA phase, virtual backdate/hint eligibility, assertion identity/address/content preservation, support/materialization/B votes, final alias ownership closure, lifecycle/all-writer enforcement, Force/current-result integration, stats/ingestion, ControllerV3 replacement, live adoption/prevention/deployment/cleanup, recall, binaries/performance/sweeps, or full-goal completion. Before production activation, duplicate lexical policy still requires deduplication behind the neutral package and unchanged-corpus tests. These are later obligations, not added prerequisites for correcting and regrading this unit.
