# Assertion bridge independent semantic disproof

2026-09-06. **NO-GO for integration or deployment.** The experiment fixes important full-assertion preservation defects, but its unchanged temporal planner can commit contradictory exclusive targets, select the wrong or arbitrary Supersedes candidate, and erase unsupported raw metadata during invalidation. Globally refusing malformed explicit dates also newly withholds unrelated usable assertions. These conclusions concern actual rows, intervals and committed utility, not changed merge statistics. No additional uncalled primitive is a prerequisite for this verdict.

## Scope and provenance

I read the complete active goal, exact experiment document and freeze manifest, archived assertion-identity design and independent review, Controller V3 independent review, and the relevant resolver/store source. Work occurred in a fresh `git archive 21a78219519e75cc9ca8e367ffcaabcb8ae35626` export at `/tmp/scry-assertion-semantic-disproof.G3lF7z`. The four experiment files were copied with apply_patch and every manifest hash matched. Only independent test/report artifacts were authored there. A second unmodified baseline export beneath `baseline/` received the same independent test file for comparison.

Every store was fabricated under `t.TempDir`, with TMPDIR inside its corresponding private export. Raw metadata fixtures closed their synthetic store, changed its synthetic Badger row, reopened it, and compared that row after Apply. No actual stores, backups, replicas, SSH, providers, memory writes, rooms, configuration, deployment, shared/frozen edits or additional agents were used. Required session orientation was read before work. The original resolver tests were never edited; a directory comparison shows only the two expected modified resolver sources and added frozen bridge test differ from baseline. The independent test file is identical in the experiment and comparison exports.

## Reproduced counterexamples and baseline attribution

The independent test file is `internal/memory/resolve/assertion_semantic_disproof_test.go`; line references below use this export.

| Case | Experiment result | Baseline result | Attribution |
| --- | --- | --- | --- |
| Equal-start exclusive targets, both orders | Two contradictory targets current | One current, but incoming old-target assertion discarded | Newly exposed unsafe temporal result after intentional noncoalescing |
| Two explicit starts observed later, both orders | Last slice item wins; older state can win | Same | Inherited planner defect |
| Later observation with older explicit start | Older state retires newer | Same | Inherited planner defect |
| Distinct normalized-equal status literals | Both current | Incoming full literal/sentence discarded | Underlying target comparator inherited, contradictory result newly exposed |
| Canonical SupRef with two eligible assertions | First assertion invalidated arbitrarily | Same | Inherited ambiguity defect |
| Canonical SupRef first candidate ineligible | Unique eligible candidate missed | Same | Inherited eligibility defect |
| Fallback SupRef one eligible and one future candidate | Entire Apply refused | Same | Inherited eligibility/partial-utility defect |
| Fresh malformed date plus unrelated assertion | Entire Apply refused; unrelated assertion absent | Both assertions committed using date fallback | New overbroad refusal |
| Exact current/historical repeats | Exact start and interval, max confidence, EP union retained | Same for these exact fixtures | Passing preservation control |
| Unknown raw field on exact current restatement | Atomic refusal and unchanged raw row | Field discarded | Experiment fixes this route |
| Unknown raw field on exclusive or SupRef invalidation | Field discarded | Same | Inherited unguarded mutation route |

1. **Equal-start exclusivity is materially broken after preserving both assertions.** Test at line 51 seeds `Atlas replaced_by Borealis`, start August 1, sentence A. An August 10 episode contains distinct undated assertions selecting Borealis and Cygnus. In both input orders, Apply succeeds, retires sentence A at August 10 and writes both new assertions with start August 10 and `InvalidAt=nil`. This is three preserved rows but two contradictory current exclusive targets, not just a different `FactsMerged` count. The baseline coalesces the new Borealis assertion into sentence A, losing its separate text/start; restoring that behavior would violate the goal. The bridge must instead resolve or defer the contradictory component before any metadata or interval mutation.

   Relevant implementation: `resolve.go:641` compares targets and `:644` treats a current row starting at the episode instant as newer/ineligible to invalidate. `:670` only closes the incoming row when its start is strictly earlier than the candidate end. Equal starts therefore leave both current.

2. **Explicit assertion starts and episode time need one stated temporal contract.** Tests at lines 75 and 102 independently vary these dates. A single August 20 episode contains Borealis starting August 5 and Cygnus starting August 10. Borealis-first ends Borealis at August 20 and leaves Cygnus current; Cygnus-first ends Cygnus at August 20 and leaves the older Borealis current. The stored result depends on slice order. A separate seed of Cygnus on August 10 is likewise retired by an August 20 observation whose new Borealis assertion explicitly starts August 5.

   Both behaviors exist at baseline. The current code explicitly uses episode time for automatic invalidation, so August 20 is a real supplied observation instant, not a fabricated timestamp. The test's preferred August 10 boundary is a proposed effective-start contract, not a previously approved specification. The proven unconditional defect is order dependence and failure to distinguish later observation from later effective state. A complete planner should adopt effective-start transitions or conservatively defer when the dates do not prove an interval; it cannot quietly assume that choosing episode time settles the issue. No incoming or existing `ValidFrom` changed in these runs.

3. **Exclusive target equality cannot reuse normalized address equality.** Test at line 117 seeds literal `stage_ready` on August 1 and observes distinct literal `stage-ready` on August 10. Both map to the same AttrDst, but their complete strings differ. The experiment retains both as current status assertions. Baseline appears to pass the test's narrow one-current check only because it loses the incoming literal and sentence through triple coalescing; that is not preservation success. `resolve.go:641` must compare typed full targets (entity identity or full literal) independently from the legacy storage slot. No automatic literal equivalence is justified by punctuation normalization.

4. **Canonical SupRef arbitrarily mutates one of several assertions.** Test at line 132 seeds two current `Atlas uses Borealis` assertions, different sentences and starts August 1 and 5. An August 10 Cygnus assertion includes triple-only SupRef `Atlas uses Borealis`. Both versions are eligible; the first raw-key candidate is invalidated and the other remains current. Apply also commits the Cygnus assertion. `currentFact` at `resolve.go:878` selects the first current triple; `applySupersedes` at `:785` uses it before checking eligibility. This is inherited behavior. A fact-plus-hint atomic contract should defer the ambiguous dependent component before Phase A merges the primary, while letting proven unrelated components commit. The test conservatively expects atomic conflict under the existing all-or-nothing Apply API; that is an acceptable temporary refusal, not a final partial-ingestion solution.

5. **Filter temporal eligibility before checking uniqueness.** Corrected test at line 148 uses same-triple facts starting January 1, 1971 and January 1, 2002, observed by an episode dated January 1, 1980. Their valid UnixNano values sort lexically with 2002 first. There is exactly one temporally eligible candidate. Canonical `uses` selects 2002, decides it is ineligible and silently misses 1971 while committing the new primary. Fallback `synthesizes_with` sees two current candidates and refuses the entire episode before excluding 2002. Both failures reproduce on baseline. The fix is candidate resolution and raw validation, then eligibility filtering, then zero/one/many disposition; selecting first, earliest, latest or highest confidence is not an ambiguity policy.

   **Fixture correction preserved:** the first run used future year 2001, whose key sorts after 1971. Both leaves stopped at a fixture precondition before Apply, so the first log is not evidence for the eligibility defect. The initial test source is preserved verbatim as `assertion_semantic_disproof_test.go.first-run.txt`. Only 2001→2002 changed. Only this corrected group was rerun on the experiment; both leaves then reached Apply and failed semantically. The full corrected corpus was run once on the baseline comparison export. There was no expectation correction or production edit.

6. **Malformed-date refusal has avoidable utility cost.** Test at line 187 uses an empty fact store, a malformed explicit date on `Atlas uses Borealis`, and an unrelated valid `Draco uses Equuleus` assertion. The experimental Apply returns ErrFactConflict, zero Stats and no Draco fact: 1/1 unrelated assertion withheld. Baseline commits both with the existing episode-time fallback. This is a characterization test whose PASS records the experiment's defect; baseline FAIL here records the absence of that refusal. There is no occupied address whose old assertion could be falsely identified. Preserve the old date fallback for a genuinely vacant destination if that remains the product policy, while retaining explicit parse-status information so a malformed date cannot assert exact identity at an occupied address. If malformed dates are instead to defer, defer this dependency component and retain the unrelated assertion; neither policy needs whole-episode refusal.

7. **Raw preservation must cover invalidation as well as restatement.** Test at line 286 adds `"future_metadata":{"keep":"exact"}` to a synthetic canonical raw fact. Exact current restatement correctly refuses in the experiment and retains byte-identical raw bytes. A new exclusive target or an explicit SupRef instead succeeds and rewrites the old row without that field. Both invalidation losses reproduce on baseline; baseline additionally loses the field on current restatement. `Store.InvalidateFact` at `store.go:969` unmarshals and marshals without the experiment's canonical round-trip guard. Its callers never pass through AssertionRestatement for the target being invalidated. Before any such mutation, either refuse unsupported/noncanonical payloads using a raw before-image validation, or use an independently specified raw-preserving patch operation. The smallest implementation is conservative refusal/dependency deferral for unsupported raw content. Merely guarding primary restatements is insufficient.

## Passing controls and intentional incompatibilities

Independent exact-repeat tests at line 200 pass for current and historical rows: two occurrences with confidence .2 preserve old .95, append the new episode once, preserve the old start, preserve historical InvalidAt, and treat `02:00+02:00` as the same instant as midnight UTC. The experiment's frozen tests also passed inside the reproduced full resolver run: changed sentence/raw relation at occupied address refuses atomically, distinct earlier/later starts remain separate, exact same-episode repeats retain maximum confidence, and malformed explicit date cannot falsely identify a current row. These are bounded resolver controls, not a whole-store proof.

Existing full resolver suite still fails eight groups/ten leaves: StatusIsAlwaysAnAttribute, FallbackPreservesDifferentStatements (two offsets), MergeRule, MergeRule_EarlierValidFromRelocates, Supersedes_OnMergedFact, ExclusiveFlip_OrderIndependent (two orders), ValidFromParseFallback/garbage-falls-back, RelationNormalized. The independent run includes the unchanged original tests plus the frozen bridge test and completes FAIL in 6.487s. No `go test ./...` pass, production GO, migration, recovery or whole-goal claim follows.

Most changed merge expectations demand collapsing a different sentence/start and are intentionally incompatible with no-loss identity. They must not be used as a reason to restore backdating or current-triple collapse. The exclusive-order failures and malformed-date rejection include actual adverse behavior, proven above. The tests retain these distinctions; no original assertion was rewritten to turn the suite green.

## Smallest complete FA planner contract to implement next

Implement this as one composed planner exercised through the authorized normal Apply path, without treating another uncalled helper as the delivery outcome:

1. Capture each original occurrence ordinal and complete input, including raw ValidFrom text and full SupRef, before mapping endpoints/relations. Resolve primary and hint identities with explicit roles and validated authority. References to absent or deferred identities participate in the dependency closure; old raw fact endpoints alone cannot authorize identity creation or invalidation.
2. Resolve all reads against the episode's immutable baseline plus a planned exact fact set, before mutating any row. Full assertion identity includes exact endpoints, literal, raw relation, statement and canonical supported start instant. Keep its actual raw address separately. Exact existing current/historical restatements preserve start, content and validity, union EPs and take maximum confidence; duplicate input occurrences retain separate observation links even when materialization is one row. Distinct assertions at an occupied legacy address defer; no timestamp nudge, relocation, guessing or collapse.
3. Use full typed target equality for exclusive relations. Choose and state effective-start transition semantics: differing explicit starts can form an ordered transition only when the context/contract establishes it; equal starts with different targets have no justified winner and defer together. Later arrival or observation does not itself prove a later effective state. Preserve already-closed historical intervals during exact repeats; avoid deriving fresh effects solely because an old assertion is re-observed.
4. SupRef resolution must collect all full-literal/raw-relation candidates, validate their exact raw records and identity endpoints, filter temporal eligibility using the declared hint event time, then apply zero/one/many behavior. Recommend zero eligible = explicit no-target/no-op disposition, one = exact planned target, many = defer dependent component until an exact reviewed assertion reference exists. Triple references must never acquire arbitrary first/latest/best selection. Resolve repeated hint effects against the planned candidate set as a whole so replay or input order cannot choose additional targets.
5. Form dependency components across primary materialization/metadata, every exclusive and SupRef invalidation target, overlapping addresses, ambiguous references, and shared identity mutations. Treat assertion-plus-hint as atomic initially. Classify every component before Phase A metadata writes or Phase B interval writes. Unsupported raw content and unresolved dependencies defer their component; unrelated components still commit alongside durable exact deferred observations/outcomes. All resulting identity support and ownership effects must be verified against this final retained set.
6. Validate exact raw before-images for every planned metadata/interval change, not only new/restated primaries. Refuse unknown, duplicate or unsupported representations unless a specifically reviewed operation preserves them. Apply the final retained plan in one transaction and check exact row/address deltas, intervals, provenance, planned current exclusive targets, observer events and recorded per-component outcomes. Force replay and partial status must remain explicit; an episode marker cannot mean that every assertion was accepted.

The date-policy and hint-time choices above are explicit implementation decisions still needed, not existing tests granted authority to invent intervals. Root can choose them within the preservation goal, then independently test the complete planner. The original compatibility suite remains a promotion blocker until behavior, requirements and tests are reconciled without hiding preservation or utility failures.

## Executed commands and evidence pins

Each command ran with CGO_ENABLED=0 and TMPDIR inside the owning export:

- Experiment: `go test ./internal/memory/resolve -count=1` → `original-plus-frozen-resolver.log`, FAIL.
- Experiment: `go test ./internal/memory/resolve -run '^TestSemantic' -count=1 -v` → `semantic-first-run.log`, FAIL; eligibility leaves are fixture failures in this log.
- Experiment after one fixture correction: `go test ./internal/memory/resolve -run '^TestSemanticSupersedesEligibilityBeforeUniqueness$' -count=1 -v` → `eligibility-corrected-run.log`, FAIL, both semantic counterexamples reached.
- Baseline comparison: `go test ./internal/memory/resolve -run '^TestSemantic' -count=1 -v` → `semantic-baseline-comparison.log`, FAIL; each result is classified above.

SHA-256:

| Artifact | Hash |
| --- | --- |
| Frozen experiment document | `68987a69f23688becddc598ecb328640d27644fca173638039714e297b63c670` |
| Frozen manifest | `c67a9f82b34b4df20364df7b0051a8258738c3bdd1e7c939d07d9abcccb537b0` |
| store/historical.go | `561543185a66cb62bd4745cba743022444b6e0b681662db25cc7be8263572711` |
| resolve/fallback.go | `2fa29873932d48edfe8386de5bd3ac239398e4b5dae9dfc54f6bc437b2970fce` |
| resolve/resolve.go | `b026d1070a571c5a62f82e14c8a04c30786bb9309fedaa9e620b86c9d8bede02` |
| Frozen assertion_preservation_bridge_test.go | `03d60f30e75aca028acb412cef97aa8e286505ab66ae557758b48a268c7b5d5a` |
| Unmodified store/store.go | `8093c39a9214e1dbe5546a8cdd5a8cd8c76836952bf2a8616630eeda3ba8f7b1` |
| Unmodified resolve/historical.go | `9749aef36ee99197b2b57464cb9b12c7ebd2752d1f8037b7fdade591a4bafb4f` |
| Independent test first-run snapshot | `bd59b62f402c46dc630adadb8aa0d11891d69ae505daaa2b4c55a908c16ffade` |
| Independent test corrected source | `f265ef5bbefe0dd24e5b4762943f8117d5bc79d236e8663f027503e6daae3dd1` |
| original-plus-frozen-resolver.log | `4add4a10816a5268a7b1290b35e6d05d7c47649c8a34412a02256c5b74ed90c8` |
| semantic-first-run.log | `463f540495274bcfb8245e781bb5cd9ddfcd9ba647bb3efd1db8362800ccd212` |
| eligibility-corrected-run.log | `158dbca8ccf7f7d06da5c02b8ab02f4491886a95b3bd720e8ebb718762818a51` |
| semantic-baseline-comparison.log | `f14adade885a8a9a97d02f5f6ccdb3b8bccc42876f7965fbd84f8ae7407036a7` |
| Archived assertion-identity independent review | `5f3cc1094bbe45ceb241b49331768884143689d4b51a6244db22d401fb3ad1f9` |
| Archived assertion-identity draft | `035adba51e490f05a49465e26680fbe8f49fa33e3cde058103e4dc589b1db55f` |
| Archived Controller V3 independent review | `2521939a134271df7669e073ee836283dd9cbaecc0339fc4df817225f587de10` |

This report's hash is supplied separately after writing. All failing evidence and both fixture revisions remain present.
