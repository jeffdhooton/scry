# Independent post-supersession preservation extension

Verdict: **scoped source-safety PASS for the revised exact-address contract.** The added check closes the independently reproduced same-episode self-supersession hole. No additional assertion/provenance loss, rollback leak or unintended ingestion regression was proved. The full unchanged no-CGO suite passes in an untouched private copy. This is not a live-store, deployment, integration, temporal-model or whole-goal certification.

The original report remains unchanged at `/tmp/scry-historical-disproof.XiyzHG/REPORT.md`, SHA-256 `7d6712b1053556a4e613cfd8a911493d40d3e28399820ab928332c061c78e2d5`. Its original counterexample and the ac2e166 baseline fixture are also unchanged, both retaining fixture hash `9e7c5f37059c6066a657a91d9dc8bf9e7676a490ae9c95f0e6619a5395fbc9dd`. This extension supersedes only its exclusion of the demonstrated same-input post-supersession reopening path; all broader limits remain.

## Reviewed revision and isolation

Read the appended post-supersession proposal and the production diff against the original pinned review copy. The production delta is one exact-address preservation block immediately after `applySupersedes`, before the remaining fallback merge/exclusive/add branches, for inputs not already merged. Store access and the historical matching helper are unchanged.

Revision pins matched before and after testing:

| Relative path | SHA-256 |
| --- | --- |
| internal/memory/resolve/resolve.go | 43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623 |
| internal/memory/resolve/historical_test.go | 1997f82e81446858270347bb0856f10a5c371fb2f1f2ad6b68de9e92e45a924c |
| internal/memory/resolve/historical.go | 9749aef36ee99197b2b57464cb9b12c7ebd2752d1f8037b7fdade591a4bafb4f |
| internal/memory/store/historical.go | 896ba4d7739eded14bf30df9d5f5afa23c6e2a85f86917f9e495af4dc14c7de6 |
| internal/memory/store/historical_test.go | 04773bd70e86428d94419d28bbda6505a4ac47c0126508da2d2752dfdc285742 |

All work used private copies:

- Independent fixtures and this report: `/tmp/scry-historical-extension.9YyYGd`.
- Untouched full-suite copy: `/tmp/scry-historical-extension-suite.CaPwrh`.
- Negative/control copy of the original candidate: `/tmp/scry-historical-extension-control.Iq4cUu`.

No candidate implementation or existing test was edited. No shared source, live graph, real queue, provider, credential, config, backup, deployment, recovery or integration write occurred. Only synthetic test-created databases were opened. apply_patch wrote new/adapted fixtures and this report; gofmt formatted fixtures. The original evidence directories were read/copied, never overwritten. A final `diff -rq` found the untouched suite copy exactly identical to the frozen revised candidate.

## Independent results

**Self-supersession preservation:** adapted the old counterexample into a preservation assertion in the extension copy, while leaving original evidence intact. For canonical `uses`, status attribute via `measured`, true fallback edge via `aliases_index_to`, and exclusive `replaced_by`, tested both confidence orderings (.95 then .2 and .2 then .95), nanosecond start preservation and a repeated superseding input. Every final assertion remains historical at the episode time, retains its original start and .95 confidence, and has exactly one episode ID. With three inputs, stats are one addition, two merges and one invalidation. The supplied two-input canonical/status cases remain green with one addition, one merge and one invalidation. The fallback fixture explicitly checks actual stored Relation, RawRelation and Dst so the test cannot silently exercise an attribute instead.

**Negative control:** ran that exact preservation matrix against a separate copy of the original candidate. All eight cases fail as expected. Canonical/status/exclusive cases reopen the row; lower-last ordering also lowers confidence. True fallback cases refuse at their occupied historical address. The revision therefore improves the actual demonstrated behavior, not merely a test label.

**Unmatched/ineligible hint controls:** missing target, unmatched raw relation, and future assertion start all remain no-op invalidations for canonical, status and true fallback cases. The final row stays current. Existing canonical/status same-episode metadata replacement remains unchanged; true fallback exact-current merging still retains maximum confidence. The same nine cases pass on the separate original-candidate control. This explicitly avoids claiming the revision repairs all current-row metadata replacement.

**Distinct assertion after hint:** canonical, status and true fallback inputs with different sentences at the same address and a self-targeted supersession hint all refuse with ErrFactConflict. Raw database state, observer events and stats prove complete rollback. A hint is not authority to silently replace a distinct occupied assertion.

**Rollback after the new branch:** for all four relation shapes, an episode first adds A, supersedes and merges A through the new check, then reaches a later distinct same-address collision on another source. Every fabricated Badger key/value byte returns to its before value, including schema/entity/index/episode state; observer events and returned stats are zero. This directly tests rollback after the added branch, separately from the original Phase A historical-merge rollback fixture.

**Original preservation matrix:** all previously independent history/current coexistence, staged Phase B history, historical hint execution, benign out-of-order ingestion, malformed dates outside history, later undated exclusive recurrence, occupied-history assertion mismatch, unsupported raw payload refusal, unrelated ingestion beside unsupported history, and late-conflict full-raw rollback cases pass on the revision. The raw helper remains byte-for-byte unchanged from the original review.

## Fixture mapping correction

The supplied new `measured` self-supersession case on an empty store exercises `status` with literal Caldera, not fallback: `Map("measured")` returns status and Caldera has not been established as an entity. The earlier seeded-history tests do reach fallback because their target already exists and the resolver converts status-on-an-identity to related_to.

The first independent no-op hint batch incorrectly expected fallback's .95 merge behavior for this empty-store measured case and failed three subcases with actual status confidence .2. Inspection identified the test assumption; no production change was made. The matrix was corrected to expect the existing status behavior and expanded with actual `aliases_index_to` fallback plus a structural assertion. All revised cases pass, and the no-op controls independently pass on the original candidate. The root has been told to correct its supplied test coverage/description in a separate test-only revision; the currently frozen measured case must not be described as fallback coverage.

This is a coverage correction, not an implementation rejection. Independent fixtures supply the missing true-fallback evidence for the frozen production hash above.

## Commands and accounting

All test commands used `CGO_ENABLED=0` and `-count=1`; no external provider test was enabled.

1. In `/tmp/scry-historical-extension-suite.CaPwrh`, after copying the frozen candidate and leaving that copy untouched:

   `CGO_ENABLED=0 go test ./... -count=1`

   Exit 0. Every package passed or reported no tests; queue 8.145s, resolve 13.694s, store 13.655s, daemon 27.028s. This is the definitive unchanged-full-suite result, including all existing queue recovery/current merge/backdating tests and the candidate's supplied tests.

2. In `/tmp/scry-historical-extension.9YyYGd`:

   `CGO_ENABLED=0 go test ./internal/memory/resolve -run TestIndependent -count=1 -v`

   Initial batch exited 1 solely for the three incorrect measured/no-op expectations explained above. Final expanded matrix exited 0, package 2.411s. All independent preservation/refusal/rollback tests passed. No production or existing-test edits were made to obtain the pass.

3. In `/tmp/scry-historical-extension-control.Iq4cUu`, which contains the original candidate production plus copies of the revised independent fixtures:

   `CGO_ENABLED=0 go test ./internal/memory/resolve -run TestIndependentPostSupersedesUnmatchedAndIneligibleHints -count=1 -v`

   Exit 0, package 1.052s, nine controls green.

   `CGO_ENABLED=0 go test ./internal/memory/resolve -run TestIndependentSameEpisodeSelfSupersedesPreservation -count=1 -v`

   Exit 1, package 0.852s, all eight negative controls fail as expected for the old hole/refusal behavior.

An earlier full-suite invocation in the working extension copy also exited 0. Independent fixture files were copied into that directory after the command started; it is not used as the definitive unchanged-suite evidence because that timing makes the fixture inclusion ambiguous. The second untouched-copy run above removes that ambiguity. No existing tests were edited in either directory.

Final independent fixture SHA-256 values:

- `independent_historical_test.go`: `104cda3b753d40963ceb0b85e0767343a7a2482da93da6cb84e4343e8033fb87`
- `independent_raw_test.go`: `ed1fd5fd4fee0d4736c807eca2f6bd77db404b976b177acbde4b68c09a7da9be`
- `independent_extension_test.go`: `da8f23b2eecd8216c249d525e41bb083054b0a51df7965f97703836ff8850184`

## Limits

The exact-address check preserves an assertion closed by the same input's supersession hint, while retaining its explicit invalidation. It does not establish semantic intent for contradictory hints, change tuple ambiguity/eligibility, make all episode orderings equivalent, or solve arbitrary temporal identity. Current-triple sentence loss/backdating, direct PutFact metadata replacement, unsupported raw current-row rewriting, general exact FactRef redesign, malformed dates outside touched historical addresses, normalized exclusive-target comparison and legacy UnixNano collisions remain outstanding.

No live snapshot, recall, latency, recovery, migration or deploy was graded. The separate schema-refusal artifact remains separate. This extension approves only the frozen source-level preservation boundary under the tested conditions; no complete-memory-goal or production-readiness claim is granted.
