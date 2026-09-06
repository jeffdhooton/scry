# Complete assertion plan: bounded independent design disproof

2026-09-06. Reviewed draft SHA-256 `508a70c2fbce2d1e9b3e8062577469d5d6144c4d10f53351aa00a3ae4e05bdd3`.

**Not complete as written. The composed approach is feasible, but three concrete corrections are needed: prohibit newly materialized resolved self-loops, make temporal deferral/results representable without falsifying identity resolution, and prevent accepted hints from acquiring new targets during partial Force retry.** None requires returning to triple coalescing or building another uncalled helper as a prerequisite. Implement the corrections as part of the fixed owned program exercised through normal Apply. This review is not a grade of an implementation, the separately reviewed identity overlay, or any production/full-goal state.

## Scope and evidence

I read the full active objective, exact draft, complete archived semantic bridge review and experiment, assertion-identity review, ControllerV3 review and identity-mutation review. Required session orientation ran before other work. I inspected current resolver Apply/Force, endpoint mapping, both fact phases, temporal and fallback selectors, original resolver tests, the store's FA/EP writers, raw inventory, strict input/observation and outcome codecs. All work was in a fresh `git archive 32e1302391e95581f1d88ba342b773d4a765db76` export at `/tmp/scry-complete-fact-design-disproof.pL9J71`. Shared production sources and original tests were untouched.

Only two new characterization tests and this report were authored, with apply_patch and gofmt for the tests. Fabricated stores used t.TempDir with TMPDIR inside this export for every Go command. No actual stores, backups, replicas, SSH, providers, rooms, remember, configuration, deployment, cleanup or additional agents were used. The archived semantic disproof source remains unchanged at its pinned hash. There were no failing tests or fixture corrections in this review; PASS below means the explicitly asserted baseline limitation was reproduced, not that the proposed planner exists.

## 1. Partial Force must not repeat a terminal hint against a new target

The draft freezes candidate sets within a planning execution and makes a completed same-input execution a no-op. Neither rule defines what happens to an already committed component while a different component of the same episode remains deferred.

Concrete timeline, all dates in August 2026:

1. X1 is `Atlas uses Borealis`, sentence S1, interval [1, infinity).
2. Episode E occurs on day 10. Occurrence H is an independently useful `Draco uses Equuleus` assertion with SupRef `Atlas uses Borealis`. H uniquely targets X1 and closes it at 10. An unrelated occurrence R defers, so E is partial. The H result is committed.
3. Another permitted episode subsequently adds X2: `Atlas uses Borealis`, different sentence S2, start 5, current. This nonexclusive relation permits that independent assertion.
4. Force retries the same exact E input to retry R. If it replans H from the new baseline, X1 is ineligible because hint event equals its end. X2 is uniquely eligible. H can now close X2 at 10.

This is successive invalidation by one original hint, despite freezing each individual run's candidates. It also changes H's selected target without a changed input revision or explicit repair authority. The sentence saying already-ended replay gets a separate no-effect result does not specify whether X1's prior target is authoritative over the newly eligible X2.

`TestCompleteFactDesignBaselineForceHintRetargets` reproduces the underlying mechanism through actual baseline ApplyWith: first Apply closes X1, insertion adds X2, identical Force closes X2; both end at E's event. The baseline has no partial outcomes, so this executable is a Force characterization, **not an executed partial-controller disproof**. The four-step partial timeline above demonstrates the specification gap. The draft's completed-result no-op would correctly prevent the baseline test's completed case; it does not address the partial case.

Smallest correction:

- Select and pin the exact original revision and prior complete outcome head before retry planning. Within the same revision, carry forward already committed occurrences, their exact original assertion/materialization references and hint targets/effects. Do not run their hints, exclusivity transitions, votes or metadata writes again merely because another occurrence is deferred.
- Replan only retryable deferred components against revalidated current dependencies. A new component may depend on an existing committed row as baseline, but that does not reopen the old occurrence's effect program.
- An already-ended replay result must name the exact prior target/effect and predecessor result. End equality or a matching triple alone does not establish replay. If materialization has subsequently moved under a reviewed repair, use its explicit lineage or surface stale/conflicting evidence; never silently reacquire a target.
- Identical Force with unchanged dispositions must reuse immutable records and head, with zero duplicate votes/events. Append a successor only when an actual result/revision changes. The draft's unconditional phrase "Force replay appends" should be qualified accordingly.
- Changed extraction under the same EP identity retains the old revision/history and follows an explicit revision-selection rule. A changed revision is not an implicit instruction to undo earlier accepted facts.

## 2. The current strict outcome codec cannot encode the promised truth

Consider an occupied legacy address or equal-start exclusive conflict where Atlas and Borealis are both correctly recognized existing identities. The assertion is deferred solely for an FA reason; neither endpoint is unresolved.

At current source `identity_outcome.go`, assertion outcomes have only `committed`/`deferred` dispositions, and deferred requires at least one link whose State is `deferred`. Such a link must have FinalSide `none` and an empty ResolvedSlug. `TestCompleteFactDesignCodecCannotDescribeResolvedTemporalDeferral` proves that truthful resolved source/destination links are rejected, while changing one resolved route into an unresolved route makes encoding succeed. Doing that in the controller would falsify the observations' resolution outcome.

The same codec explicitly rejects non-nil Materialization for every assertion outcome. `TestCompleteFactDesignCodecCannotRecordAssertionMaterialization` proves a committed assertion encodes with nil and rejects any non-nil materialization. The schema has no exact FA address/full tuple, frozen candidate inventory, hint target/effect, temporal reason or revision/current-result selection fields. These are intentional limitations of an uncalled descriptive identity codec, not a newly discovered production corruption bug. Full original observations cannot reconstruct what an ambiguous hint actually selected after the graph has changed.

Smallest correction: stop promising that the existing outcome shape alone carries complete assertion dispositions. Within the composed implementation, extend/version the occurrence result envelope or bind it to an exact immutable planner result that records assertion-level reason independently of endpoint-resolution State, the original revision and occurrence, exact materialization reference or absence, and hint candidate/target/effect evidence. Preserve existing v1 bytes and bounded readers. The fixed owner must derive and verify those records; a caller-supplied approved-plan record remains prohibited. Non-assertions, rejected optional aliases and no-effect hints need truthful distinct result fields, without inventing an unresolved identity or a successful FA write.

There is a narrower input boundary mismatch too: the draft says capture "confidence bits", but current `observedFact.Confidence` is float64 marshaled directly as JSON. `TestCompleteFactDesignExistingInputCannotPreserveAllConfidenceBits` proves two distinct NaN payloads and positive infinity cannot encode as an input revision. Ordinary valid JSON extraction does not contain those values, so this is not evidence that normal provider inputs lose them. Explicitly choose either a finite canonical-input precondition with safe rejection/queue preservation before this entry, or an exact bits-bearing observation envelope if invalid parsed confidences must receive local durable dispositions. Do not claim the present codec provides the latter.

## 3. Final resolved self-loops need an explicit admission gate

Goal clause 10 forbids the normal path from recreating a self-loop after cleanup. The proposed full assertion checks prohibit unsupported owners and intervals but never reject an edge whose final source and destination are the same recognized identity. Birth support, ownership closure, canonical relations and full tuple/address equality do not imply that rejection.

`TestCompleteFactDesignBaselineAliasSelfLoop` creates recognized Atlas with a valid listed/indexed alias `Atlas Service`, plus Draco and Equuleus. Apply receives `Atlas uses Atlas Service` alongside unrelated `Draco uses Equuleus`. Baseline commits two facts, including `atlas uses atlas`, with no missing endpoints, ownership transfer or birth. This passes all explicitly listed ordinary assertion rules in the draft. Store.PutFact also accepts equal nonempty source/destination. Thus this is an inherited prevention defect omitted from the replacement admission contract, not a reason to reject alias resolution itself.

Smallest correction: after final inverse/value routing and full identity resolution, a newly materialized edge with nonempty destination and Src==Dst is a typed local assertion deferral, preserving original observations and its complete hint component. It contributes no new identity support and cannot produce invalidation effects. Retain the unrelated Draco fact. Do not silently rewrite its relation or invalidate the newly written loop as cleanup. Existing self-loop raw rows remain preserved; exact baseline restatements need an explicit no-new-state-effects policy and cannot certify a clean graph. This check is mechanical equality, not a new lexical rule or an automatic identity-owner choice.

## Remaining temporal/component contract review

I found no additional contradiction in the core effective-start rules, subject to the precision below. These are synthetic timeline deductions from the draft, not tests of an unimplemented planner:

| Input/timeline | Coherent required disposition |
| --- | --- |
| New exclusive B at 5, C at 10, observed 20, either order | B [5,10), C [10,infinity); no slice or observation-time winner. |
| Baseline C starts 10; later observation 20 supplies B starting 5 | B may be historical [5,10); C stays current. |
| New B and C both start 10 | Defer their state component; retain originals and unrelated utility. No confidence/ordinal tie-break. |
| Baseline closed B [1,10); incoming C starts 5 | C's inferred interval overlaps immutable B; defer cohort without changing B. |
| Baseline closed B [1,10); exact B restatement observed 20 | Keep [1,10), max confidence and deduped EPs; no new state event. Passing baseline control reproduces this. |
| Distinct same-target sentences at starts 1 and 5 | May overlap; remain different full assertions. Triple-only SupRef before either end is ambiguous. |
| Literal stage_ready versus stage-ready | Distinct targets even though AttrDst collides; starts at the same occupied address defer rather than coalesce. |
| Hint event 1980; matching starts 1971 and 2002 | Only 1971 is eligible. Full matching precedes eligibility, which precedes uniqueness. |
| Closed [1,20) target; hint event 10 | Target contributes eligibility/ambiguity, but a unique target would require changing immutable history and defers its dependent component. |
| Self hint targets its own newly supplied start equal to event | Zero eligible target; no zero-width interval. Distinct from a graph self-loop. |
| Two explicit-start new assertions point hints at each other, both starts before event | Both candidates can be unique against the same preliminary set; combine actual ends once, then validate final positive intervals. No mutation-order selection. |

Make "preliminary interval" explicit: it means the exact candidate interval derived from the frozen baseline plus proposed exclusive timeline, before any SupRef changes. If any assertion supplying a proposed transition is removed, its connected cohort/effects must be removed too; a retained row cannot keep an end whose only supplier was deferred. Monotonic component removal makes this implementable without recomputing ambiguity into a winner.

Distinguish the frozen matching universe from temporally eligible targets and from component dependency edges. In the 1971/2002 case, a future candidate must not become an arbitrary winner, nor should removing it permit new selection. The text "Include all potential candidates" can conservatively connect an eligible hint to an ineligible future occurrence whose own hint fails, withholding additional utility. If that is intended, expose it as dependency deferral rather than claiming ambiguity; if useful independent components are required to survive, connect only dependencies capable of changing validated eligibility/effects, while retaining the full frozen universe as evidence. An unreadable raw candidate is a separate proof failure. This is a bounded utility clarification, not a demonstrated unsafe mutation.

Raw preservation rules now cover restatement, exclusive closure and SupRef closure; byte round-trip plus exact before-image/key binding defeats the archived unknown-field invalidation loss. Keep malformed inventory refusal distinct from valid unsupported-row component deferral. Validate final adjacency effects as well as FA rows: ordinary PutFact writes adj: and existing stale/missing mirrors must not be silently repaired outside the approved delta. No new address format means sibling/recovery consumers remain a later all-writer gate, as the draft correctly states.

The identity support/ownership direction is coherent: proposals visible during discovery are not support; a retained actual FA may support the originally discovered birth; preexisting dangling references cannot create that identity; unsupported discovery may still be consumed by another valid retained occurrence; optional alias rejection need not destroy that assertion. Complete literal/natural/index relationship closure and exact unchanged-defect equality preserve the controlling identity-mutation review. These remain whole-program proof obligations, not properties of counts or a codec. No opinion about the private fifth overlay's implementation or approval is supplied here.

## Intentional incompatibilities and implementation gate

Effective-start transitions intentionally replace observation-time invalidation. Exact full assertions intentionally replace current-triple sentence loss and earlier-start relocation. Equal-start conflicts intentionally replace an arbitrary/coalesced winner. Malformed explicit dates intentionally change fallback into local deferral. These changes must be reflected in reviewed replacement tests while the original failures remain archived. The bridge's whole-episode malformed-date refusal and its contradictory exclusive rows are not acceptable compatibility behavior. The existing resolver suite passes on the untouched baseline here; that is evidence about the baseline, not a reason to restore lossy expectations in the new program.

The smallest next delivery is the composed private normal-Apply implementation with the three corrections above and explicit preliminary-interval/replay definitions, not another codec-only success. Fresh independent exact-source grading must demonstrate actual committed FA/adj/identity/EP/outcome rows, original occurrence preservation, per-component utility, immutable accepted partial retries/reopen, ownership/birth support, all rewrite-route raw preservation, conflict rollback and zero leaked events. Existing tests must be reconciled explicitly; then run the full no-CGO suite and the objective's actual-backup replica/adoption/all-writer/rollback gates. No integration, deployment, recovery, cleanup or full-goal approval follows from this design review or the baseline passing tests.

## Commands and pins

All commands ran with `TMPDIR=/tmp/scry-complete-fact-design-disproof.pL9J71/tmp CGO_ENABLED=0`:

- `go test ./internal/memory/resolve ./internal/memory/store -run '^TestCompleteFactDesign' -count=1 -v` -> `characterization.log`, PASS, initial five groups.
- After adding the separate Force characterization: `go test ./internal/memory/resolve -run '^TestCompleteFactDesignBaselineForceHintRetargets$' -count=1 -v` -> `force-characterization.log`, PASS.
- `go test ./internal/memory/resolve -count=1` -> `original-plus-characterization-resolver.log`, PASS 6.184s; all original resolver tests unmodified plus the three new resolver groups.

SHA-256, paths relative to this private export unless absolute:

| Artifact | SHA-256 |
| --- | --- |
| Draft `/tmp/scry-complete-fact-plan-sep06.VK8j3B/COMPLETE_FACT_PLAN.md` | `508a70c2fbce2d1e9b3e8062577469d5d6144c4d10f53351aa00a3ae4e05bdd3` |
| Archived semantic review | `86773de0218016c370ea37570f5309500503395dc384f1197cfcc0e1db17b26e` |
| Archived bridge experiment | `68987a69f23688becddc598ecb328640d27644fca173638039714e297b63c670` |
| Archived unchanged semantic disproof test | `f265ef5bbefe0dd24e5b4762943f8117d5bc79d236e8663f027503e6daae3dd1` |
| Archived assertion-identity review | `5f3cc1094bbe45ceb241b49331768884143689d4b51a6244db22d401fb3ad1f9` |
| Archived ControllerV3 review | `2521939a134271df7669e073ee836283dd9cbaecc0339fc4df817225f587de10` |
| Archived identity-mutation review | `94e6e51f71e8caaa49e23e2e7c8a92de556c73099345a4b29d1c904c8219b3d3` |
| `internal/memory/resolve/complete_fact_design_characterization_test.go` | `7bc8620a30ef0f9b39b91d01fbdd8d0344cb1e86d2b0ed92ab98a5d2b79526a3` |
| `internal/memory/store/complete_fact_design_characterization_test.go` | `a9b56e609ac51bf6493b7b03557fd09624392170885ee7609cf82bf0c13744a0` |
| `characterization.log` | `65dadbee349ddf235bd144be97d587bbd32c865aace493d0fd3a2ffa323d79b6` |
| `force-characterization.log` | `92c47cb5a6810775666912e053be3b24b5b4b998ea0fd71ee935f25c36f69797` |
| `original-plus-characterization-resolver.log` | `b8355db999ca853e4b6cd14f53138fa551e75d3df2d78599c6b81138d293e5d2` |
| Baseline `internal/memory/resolve/resolve.go` | `43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623` |
| Unchanged `internal/memory/resolve/resolve_test.go` | `0dd518b0984ba4abc98c946a9e569c68ce39cdf31bb98a7910410961e925cc47` |
| Baseline `internal/memory/store/store.go` | `8093c39a9214e1dbe5546a8cdd5a8cd8c76836952bf2a8616630eeda3ba8f7b1` |
| Baseline `internal/memory/store/identity_outcome.go` | `7b40e318a2c323efed48a5d40c79853fd61e1455e24265903300e7f3cc7c59e8` |
| Baseline `internal/memory/store/identity_input_revision.go` | `b69e6daf91e98fba34166fe949c374d65f9196831d49ddcd11b9650364e59908` |

The final report hash is returned separately to avoid a self-hash mismatch.
