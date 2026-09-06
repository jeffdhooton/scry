# Private restatement bridge disproof

Verdict: **do not integrate or deploy this candidate.** The historical-preservation fixtures pass, but the candidate introduces a deterministic ordinary-ingestion refusal, inconsistent supersession admission, and an unresolved recurrence interpretation. Refusing paraphrases instead of erasing their sentences is an intentional preservation difference; parking an identical undated fact solely because workers finish out of order is a separate operational regression. The unchanged suite remains red.

Reviewed the active goal, RESTATEMENT_PROPOSAL.md, candidate source and prior assertion-identity design review. Candidate base is ac2e166. Source SHA-256 values were checked before and after testing:

- fallback.go: `e0dd18560fc669abd3d6f05f2cc6a5e58d1a5b75f20a9dfcd4865880aceaeaca`
- resolve.go: `81d45041bf1d014e6460a84e2c0d54e4006711e42d7c41be975abe1ad6c35ae8`
- restatement_bridge_test.go: `a6c789ceccffae4d93e620c5a1313025335c70a180df900cf47abd5b7d297918`

The prototype `/tmp/scry-restatement-bridge-sep06.gOOJWa` was not edited. Tests ran in a separate private copy, `/tmp/scry-restatement-disproof.Ra5yXf`. Fabricated cases are in its `internal/memory/resolve/independent_disproof_test.go`. Their passing results mean the counterexamples were reproduced, not that the implementation passed review. No real backup, source transcript, credential field, provider, shared graph, queue retry, deployment, config or recovery was accessed or changed. The queue test uses its existing fake extractor and temporary database.

## 1. Identical undated facts are parked by arrival order

`matchingFactForMerge` skips an otherwise exact undated record when the incoming episode precedes the stored start (fallback.go:27). Its subsequent `canonicalCurrent` guard then refuses that same canonical triple (line 37), even though the earlier legacy address is vacant.

Independent deterministic fixture: store `lornwick uses caldera`, sentence `Lornwick uses Caldera for routing.`, starting August 10. Submit the identical sentence without valid_from from an episode one hour earlier. Result is ErrFactConflict, zero stats, unchanged old fact and no added provenance. This needs no paraphrase, malformed input, multiple intervals, or provider variation.

The unchanged `TestProviderOutageThenRecoveryDrainsEverything` independently failed with `ready 0 backoff 0 parked 2`. The parent previously observed three; the number depends on scheduling. All twenty queued items have identical `Fact: "fact"` and distinct occurrence times; concurrent worker completion supplies the ordering trigger. The exact test passes on an independently exported ac2e166 baseline. This is a concrete regression against ingestion recovery, not just a historical-preservation expectation change. Retaining a parked original avoids deleting that input, but does not drain it or make its facts queryable.

## 2. Canonical admission depends on Phase A/B history and slice order

Independent fixture, same endpoints/relation/sentence throughout:

- First assertion has explicit start t1.
- Second has explicit start t2 > t1 and a Supersedes hint naming the same triple.
- Episode occurrence is t3 > t2.

On an empty store, `[first, second]` succeeds with two added facts and one invalidation: second's Phase B hint invalidates first before the repeated merge check, removing `canonicalCurrent`. Reversing the slice refuses the whole episode. Seeding first before applying second also refuses: Phase A sees the current tuple and returns before second's hint can run.

This is stronger than the unchanged tests that use different paraphrases: a replacement can fit the legacy address and pass or fail solely from processing phase. The proposal says Phase B should apply the same policy to assertions added in the episode, but this result has no uniform preflight boundary. It also produces overlapping intervals: first ends at t3, second starts at t2. The choice of episode time for supersession predates this candidate; the newly exposed admission route does not solve it.

Tuple supersession now refuses multiple current matches before checking temporal eligibility. A separate fixture with one old and one future current assertion refuses even though only the old one predates the episode. This is conservative rather than destructive, and matches the proposal's broad ambiguity refusal. It must be a declared policy decision; filtering eligible assertions first would be a different contract. Do not restore first-match selection to make the test green.

## 3. Malformed explicit dates are not explicit valid instants

`parseValidFrom` (resolve.go:895) silently substitutes episode time for an invalid nonempty date. The new comparator uses `fct.ValidFrom != ""` as the explicit-date flag, so malformed dates receive explicit-date matching semantics without having parsed successfully.

Three independent fixtures use `not-a-date`, `2026-02-30`, and one space. When episode time equals an existing historical start, all three are accepted and merged into that historical row. At a different occurrence, the same malformed input can instead conflict with a current canonical triple or add an assertion at an invented fallback start when no current triple blocks it.

The permissive parser is preexisting, but the candidate's explicit-versus-missing identity rule now depends on it. A typed parse result should distinguish absent, valid explicit, and invalid explicit. Invalid explicit dates should not silently become assertion identity. Date-only values and equivalent timezone encodings should retain documented valid parsing behavior. Out-of-range UnixNano addressing remains a separate existing format limitation, not solved here.

## 4. Undated historical matching consumes possible recurrence

Independent exclusive-relation fixture: `lornwick replaced_by caldera` is historical over [t1,t2); `lornwick replaced_by emberbank` is current from t2. At t3, submit the exact old Caldera sentence without valid_from. Candidate reports one merge, zero additions and zero invalidations; Emberbank stays the sole current target. The t3 episode becomes provenance of the closed Caldera interval.

This complies with the proposal's assumption that an undated fact asserts no new start. It also demonstrates the consequence: the same input shape expressing a present return to Caldera is silently consumed as historical evidence. Exact text does not establish which interval a new occurrence attests. A unique historical match is insufficient when recurrence is possible. This is an unresolved semantic choice, not a request to reopen old history or restore destructive coalescing. Explicit assertion/interval references, or refusal where intent is ambiguous, are needed before claiming current-state correctness.

## 5. Rollback held; raw preservation remains incomplete

Independent late-conflict fixture first merges new provenance and higher confidence in Phase A, then applies a supersession invalidation and a new fact in Phase B, then conflicts on another same-episode sentence. All decoded facts and entities exactly match their before snapshots, the episode is absent, returned stats are zero, and the observer receives zero events. The outer AtomicWrite boundary works for this tested path; I did not find a partial-commit leak.

The supplied historical-restatement tests also pass: original start and InvalidAt survive, provenance is unioned and confidence takes the maximum. Ambiguous-history and whole-episode-refusal tests pass unchanged.

These are decoded-field guarantees. Source inspection still shows `FactsFrom` decoding into `store.Fact` and `mergeFact` calling PutFact, which marshals the typed struct and replaces the row. Unknown raw JSON fields are not carried through; current and now historical restatements can erase them. This is an existing general mutation limitation with newly expanded historical reach, not dynamically re-proved here. Direct PutFact still allows same-identity replacement of metadata, including provenance and InvalidAt; fixing resolver matching is not a store-wide preservation guarantee. Existing normalized-literal exclusive-target comparisons and bulk invalidation also remain outside this patch.

## 6. Unchanged test accounting

`go test ./internal/memory/resolve -count=1` failed six top-level tests / seven leaf cases:

| Test | Meaning of failure |
|---|---|
| TestApply_StatusIsAlwaysAnAttribute | Different sentence/raw relation on current status no longer coalesces. Intentional preservation difference; does not prove status-node conversion broke. |
| TestApply_MergeRule | Different sentence and explicit start no longer coalesce. Intentional preservation difference. |
| TestApply_MergeRule_EarlierValidFromRelocates | Changed sentence/earlier explicit start no longer relocates old row. Intentional prevention of destructive backdating. |
| TestApply_Supersedes_OnMergedFact | Paraphrase refusal prevents the episode's unrelated supersession hint. Admission change with operational consequence, not evidence the exact-restatement hint path alone broke. |
| TestApply_ExclusiveFlip_OrderIndependent, both subtests | Paraphrase refuses before flip in both orders. These failures alone do not prove order dependence; the independent dated same-sentence fixture above does. |
| TestApply_RelationNormalized | Different sentence refused; normalization itself still executes. |

Each of these named cases passes on ac2e166, including the backdating test run separately. The independent unchanged queue test fails on candidate and passes on baseline. This review did not run the entire repository suite or live replay/recall; no complete green or live-grade claim is made. No existing test was modified.

## 7. Smallest safe next boundary

Keep the separate schema-refusal artifact independent of this resolver experiment. Do not promote the global comparator/refusal change under the label of a historical-restatement fix.

A historical-only follow-up can target the proven occupied historical row that would otherwise be overwritten/reopened: after successful explicit date parsing, locate the exact full assertion at that exact start, retain its start and InvalidAt, union provenance and take max confidence in the enclosing transaction. Also protect actual occupied historical addresses reached through missing-date fallback from metadata replacement. Refuse unsupported raw payloads or preserve their unknown fields rather than reserializing them away. Do not infer a historical interval from text alone merely because it is unique. Test this bounded branch without replacing the established canonical current coalescing/backdating policy, and clearly retain that policy as an outstanding loss risk rather than claiming the goal passed.

That narrower work can improve preservation without deciding all canonical assertion identity. It cannot solve general sentence loss while simultaneously preserving every existing current-triple coalescing expectation. Global noncoalescing is a separate design/implementation boundary: specify undated observations and recurrence, out-of-order earlier evidence, explicit malformed dates, exact supersession references and eligibility, and an order-independent episode plan before mutation. An address discriminator/FactRef redesign may be required for distinct assertions sharing a legacy address, but changing keys alone does not settle the semantic decisions or backlog admission policy.

Do not “fix” the demonstrated queue regression by dropping the earlier-time guard and silently losing earlier as-of evidence, by sorting worker completion, by nudging timestamps, or by relabeling refusal fixtures as ingestion success. The next resolver candidate needs unchanged-suite accounting, these independent cases, and fresh replica/recall grading before integration. No migration, recovery, deployment or whole-goal PASS is granted.
