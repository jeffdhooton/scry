# Fixed ordered overlay contract — independent bounded design disproof

2026-09-06. Baseline `e097fa6cb2e5c5ff8afcbd2b21c163d879856d84`.
Reviewed frozen draft `/tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/ORDERED_OVERLAY_CONTRACT_DRAFT.md`, SHA-256 `108850beb6f4cf40048842485d9abfd4ec80fe6a4ae295931d9e6d22f13c3ab9`.

GO for private, uncalled, read-only implementation after incorporating the mandatory corrections below. The draft alone leaves meaningful routing/order choices unspecified, but the fixed Store entry, genuine registration wrapper and neutral pure-policy package form a finite implementable boundary. No actual behavior examined requires unsupported identities to have durable writes. This report is a design verdict, not an implementation PASS, permission to replace ControllerV3, or evidence that the normal write path is safe.

The lead agreed to the corrections during this review. Those messages are clarifications, not bytes attributed to the frozen draft. Freeze them with this report before implementing the contested routing. No additional foundation primitive or support solver is required merely to begin this bounded implementation.

## Evidence and scope

I ran and read `scry memory orient --cwd .` first. I read the full active goal, supplied draft, delayed-birth design and complete independent review, full eight-group characterization source, ControllerV3 and identity-mutation independent reviews, birth-registration contract and episode-selection contract. I inspected actual complete resolver declaration/endpoint/hint/admission/index routines, registration, generation and legacy recognition codecs/readers, owner wrapper, identity/fact inventories and ledgers, store entity/alias/transaction routines, relevant retirement analysis/writer, value evidence and alias rejection readers, and actual local Badger commit behavior.

The private export is `/tmp/scry-ordered-overlay-disproof.ByOqut`, created with `git archive e097fa6`. I authored only new synthetic characterization tests, a retained initial new test source and this report with apply_patch; gofmt formatted new tests. All 600 archived baseline files were compared to their git blob hashes: zero changed or missing. The shared worktree remained at its original four untracked documents. No shared/candidate edits, real stores/replicas, SSH, providers, configuration, backups, rooms, remember, deployment, sweeps or subagents were used.

Five new resolver characterization groups and one new storage characterization pass. The eight preceding delayed-birth groups were copied byte-for-byte into the export and pass; the unchanged original retirement-rehome test passes. The full resolver suite, including every existing inversion/value/artifact/name guard and EmptySlugSkipped expectation, passes. These tests characterize baseline behavior and the design's required distinctions; no overlay exists here. I did not claim a full repository suite or a new implementation grade.

All runs in this review passed initially. The first four-group source and initial log remain retained; the expanded fifth group adds a new characterization without changing previous assertions. No initial failed run was erased, and there was no failure to relabel. Earlier reviews' failed fixtures/logs remain recorded under their original pins; this review does not supersede that evidence.

## Mandatory corrections and concrete disproofs

### 1. Failed declarations need a typed per-spelling binding

`TestOrderedContractFailedDeclarationNeedsTypedBinding` invokes actual baseline component helpers on a synthetic store. Cobalt(service) first admits Cobalt daemon. The later Cobalt daemon(machine) declaration fails its canonical claim with ErrAliasClaimed. If a partial planner catches that conflict and merely omits a successful string-map entry, `resolveSlugOnly("Cobalt daemon")` still returns `cobalt` through the earlier valid alias. Baseline Apply avoids that continuation by rolling back the entire episode. Therefore ordinary string-map omission is not the new partial-utility policy.

Freeze a typed declaration binding indexed by normalized declaration spelling. A required-route conflict writes dependency-deferred for that spelling. Primary lookup must consult it before any successful-declaration or alias fallback, including status/value transformations that might otherwise turn the failure into an attribute. Preserve the earlier Cobalt candidate and all its valid earlier proposals; reject only the failed declaration's proposed authority. Its useful unrelated facts remain inspectable/resolvable. Retain the blocked declaration and predecessor dependencies.

A later declaration is independently evaluated against the full current overlay and baseline controls, never against a deferred entry treated as an owner. Only a later successful explicit trusted identity declaration (documented type, TypeFallback=false) may replace the deferred binding with its validated route. Preserve each declaration's own result and conflict history. Value, unknown and fallback declarations do not erase the deferral. Thus service→machine conflict→service may restore a service binding via that later successful explicit declaration; service→machine→value stays deferred. No rerouting after support selection. This is a deliberate new partial-resolution rule, not baseline parity.

For failed new canonical proposals, perform the necessary claim checks before making aliases/votes routing-visible, or retain a local proposal checkpoint whose failed additions have no authority. Earlier successful proposals for the same candidate cannot be accidentally erased. Rejected proposal descriptions remain in the report. A diagnostic candidate/index occurrence is not an admitted route merely because all candidates remain retained.

### 2. Actual discovery order and canonical report order are different

`TestOrderedContractInverseDiscoveryOriginalSides` reproduces baseline `Atlas used_by Borealis`: the first entity-created observation is Borealis, then Atlas. For `Nimbus_Core used_by Nimbus Core`, both original spellings reach one slug, but the stored first Name is `Nimbus Core`, the original destination. Choosing original src first changes the creation identity even if the final endpoint slugs are the same.

Preserve baseline actual endpoint-discovery order after relation inversion and any value-source inversion. Bind each registration/link to its actual ORIGINAL source or destination observation. Registration must precede all metadata/alias/vote proposals for that candidate. Never call register with the other observation as an alleged exact retry. For same-slug later mentions, use validated links to the first handle.

The registry's producer order is discovery order. A future selector adapter may sort the complete retained birth inventory by each actual first observation's original origin/ordinal/side to satisfy selector canonical ordering. Sorting must not change the first observation, candidate creation tuple, which candidate occupied a slug first, or producer dependencies. This permits the existing selector ordering contract without changing actual resolution. Capture both original observations even when empty relation/two values produces no candidate.

### 3. Hints resolve after ALL primary discoveries

`TestOrderedContractLaterRejectedPrimaryChangesEarlierHint` is a direct execution counterexample. Seed the current attribute `Atlas status "Nimbus Core"`. The first useful fact carries a hint superseding that attribute. Without a later discovery the hint invalidates it. Add a later rejected `!!! uses Nimbus Core` primary: baseline discovers the Nimbus stub while resolving all primaries, then the earlier hint sees Nimbus as known, maps status to related_to, and does NOT invalidate the old attribute. The useful first primary is otherwise unchanged.

Freeze two identity-resolution passes: declarations, then all primary records in order; only after that resolve/annotate hints in original carrying-fact order against the completed identity overlay. Hints never register candidates or create proposals. Retain hints even on descriptive non-assertion/unresolved records, without treating them as executable mutations. An absent hinted name remains absent/deferred; a later primary may legitimately supply an already-discovered provisional candidate to an earlier hint.

Hint dependencies can therefore point to later original primary ordinals. Original ordinal is not a topological authority order. Record the actual production/read sequence and immutable version witnesses. Cyclic eventual support dependencies grant nothing: this unit performs no support closure, exact FA target selection, time-eligibility decision or hint execution. A candidate discovered only by a rejected primary may influence a hint's descriptive interpretation while remaining wholly unsupported. Later fact/support work must not silently reroute that hint after filtering the candidate.

### 4. Separate canonical homonyms from explicit listed aliases

`TestOrderedContractCanonicalHomonymExplicitIndexRoute` establishes two nonnatural-slug identities whose canonical name is Shared Circuit, using explicit ClaimAlias only to construct the synthetic historical state. Actual resolveEntity follows the explicit index owner for a same-type declaration, but its exceptional cross-type path scans the canonical homonyms and refuses. Consequently the draft's general canonical-homonym refusal is a further intentional deviation, not merely a map implementation detail.

Freeze this precedence: a recognized exact-natural canonical identity wins. Otherwise, multiple identities having the same normalized canonical name produce a typed canonical conflict even when al chooses one. Do not select by index order, type, fact counts or lexical similarity. For distinct canonical names that merely share a listed alias, a valid explicit al owner may retain explicit alias precedence, subject to full recognized-control checks and separately recorded relationship conflicts. Do not generalize canonical homonym refusal to every multiply-listed alias.

Complete compact and singular-token indices remain multimaps with deterministic versioning. Inferred ambiguity refuses; kind-name detection only vetoes an optional proposal and records all relevant witnesses, never selects a new owner. A new proposal is visible at its precise ordered transition; later discoveries affect later decisions, not earlier frozen decisions. No elapsed-time/global cache survives the scope. The previously retained Cobalt Atlas cache characterization is still an intentional correction rather than baseline equivalence.

### 5. Retirement validation must bind the actual schema and the chosen identity

The actual retirement writer stores `{entity, id,omitempty, why}` identically at rs:<retired slug> and each rt:<reviewed normalized spelling>. An alias rt key is often unrelated to Normalize(Entity). ID is legitimately optional. Do not invent an ID requirement or demand that an rt alias spell the retired entity name.

Freeze strict relevant decoding: rs key binds Entity exactly; rt key is a nonempty normalized spelling, and its complete payload must equal the matching rs:Entity payload, including valid unknown extensions preserved as raw data. Validate known fields, duplicates, structural/UTF-8 validity and nonempty reason. Missing/mismatched pairing is a typed error, not value evidence. This certifies consistency of the stored classification controls, not recovery of a missing historical manifest. The existing writer emits identical bytes. Sequential valid retirement cannot normally retire the same removed entity again; old rs rows remain. A later retired rehome recipient writes its own matching pair for its spellings, so the rule does not require every historical rs row to match one rt.

The unchanged `TestApply_RehomedSpellingCannotResurrectRetiredSlug` proves another necessary distinction: rs:obsolete may coexist with a valid explicit alias `obsolete` to a different active target-service because the retirement deliberately rehomed that spelling. The absence of rt:obsolete matters. Retired/consumed state belongs to the candidate identity being considered; it must not globally veto every mention whose natural slug equals that old identity. The valid reviewed explicit alias to the different recognized target survives. A natural fallback trying to recreate obsolete still defers, and malformed relevant controls remain errors.

Apply the same typed discipline to all exact/artifact/value/alias/hint paths. A recognized generation requires complete canonical ig key/body, stable entity tuple, actual EP identity/time and no conflicting relevant controls. Existing private generation helpers alone do not prove this. Unadopted old entities are deferred, not new. Absent identities with historical/current FA references defer before registration. No selector outcome or old att row creates authority.

### 6. A zero-write plan cannot produce Badger's write-commit conflict

`TestOrderedContractReadOnlyCommitConflictLimit` reads a synthetic key, commits another transaction's update, and then commits the stale reader. With zero pending writes, Commit returns nil. The otherwise identical case with one synthetic staged write returns ErrConflict. This matches actual dependency source: Badger Commit immediately discards/returns nil when pendingWrites is empty, before commit conflict validation.

Keep the fixed planner's outer error sanitizer, zero report on error, poison checks, owned returns, success/error/panic disposal and full synthetic raw-equality tests. Exercise actual reachable read failures and the sanitizer's allowlisted Conflict/TxnTooBig behavior. Characterize genuine write conflicts through the existing separate writer-wrapper tests if needed, but do not call those real zero-write planner conflicts. Do not add a dummy write merely to satisfy that phrase in the draft. Serialized graph exclusion and snapshot reads are the real boundary here; unrelated queue updates remain outside it. This is a finite test-bar correction, not a new transaction architecture.

## Other challenged boundaries that remain adequate when implemented literally

The acyclic package split is feasible: pure lexical predicates can receive plain lexical entity/declaration DTOs and an exact normalization implementation without importing Store or exposing a caller-selected lookup callback. The reviewed boundary is fixed Store-owned routing over genuine captured inventories. A declaration-level source manifest must include transitive regex/maps/constants and adapters, not only exported function names. Own returned slices/maps; do not export mutable vocabulary tables. Preserve every original resolver test and annotate the explicit cache, canonical-homonym, metadata-reference-preservation and partial-conflict deviations. This report does not grade the lead's separate policy port or AST tooling.

`TestOrderedContractOldMetadataWithoutFactsRetainsUtility` shows that established Atlas may upgrade concept→service and acquire its first description with no fact in the episode. It also reproduces the baseline loss of the oldest repo references when eight references become six. The draft correctly separates old metadata utility from new-candidate support and intentionally preserves all old refs. Do not add a same-episode fact gate to old description/type/LastSeen/repo proposals. Existing bad/missing index relationships may remain only under the precise unchanged-relationship exception, not merely unchanged al bytes.

New-candidate alias evidence must stay in memory and be keyed by genuine candidate plus normalized alias and episode. Repeated declarations cannot enter legacy attestation. A vote proposal is subject to the actual admission branch and early vetoes; recording an original rejected alias observation is not a qualifying vote. Old recognized legacy evidence is separately validated/deduplicated; generations use only their generation-specific evidence. Sufficient votes still do not transfer any baseline owner's key. The private B writer remains unapproved and must not be used as authority.

The draft correctly retains every provisional discovery through all resolution. It must separately retain successful earlier projections, failed declaration bindings, optional alias rejection reasons, producer order and exact baseline witnesses. Full changed-identity relationship dependencies include old unlisted keys and all normalized/natural listing occurrences. Complete final ownership cannot be approved before support selection determines which candidates exist. Thus this unit reports provisional relationships and conflicts without labeling any final set admitted, supported or committed. Descriptive cyclic dependencies are allowed; they cannot certify one another.

Every original declaration and fact slot must appear once in the report with full originals, complete SupRef nil/present information, raw and normalized relation, original-to-final maps and typed dispositions. Unknown relation remains related_to fallback, not invalid relation. Neither failures nor rejected primaries may be omitted to improve counts. Never-staged candidates have no materialization; named versioned proposal projections are sufficient for inspection.

The fixed program must be the only producer: no returned Store/facade/handle, caller finalizer, replacement callback or fake owner projection. Use the genuine registration wrapper, verify zero identity/fact writer history, audit every proposal/link and wrapper final inventory, and expose only owned descriptive data after outer success. Inventory equality proves final conservation of the captured families; static all-producer inspection and full synthetic raw equality establish this particular unit's lack of other writes. It is not proof against an arbitrary untracked transient raw writer or a production all-writer certificate.

## Finite implementation bar and verdict limits

Proceed with the one private read-only unit after freezing this report's corrections. Its independent code review should exercise the existing eight groups plus the six new groups here, original resolver corpus, exact/alias/canonical precedence, typed declaration conflict replacement, current/history-only deferral, retirement rehomes and malformed relevant controls, repeated votes and conflicting metadata, old metadata-only utility, unlisted/stolen/missing claims, original-side/link ordering, late-primary hint visibility, owned reports and zero raw changes/events. No support/fact/executor implementation is needed to make that bar meaningful.

This verdict explicitly does NOT approve: either FA phase or virtual backdate/hint eligibility; assertion-address/content collision handling; final factual support; materialization or B votes; final alias relationship closure; generation/legacy lifecycle adoption or all-writer enforcement; immutable Force/current-result integration; stats/ingested claims; prevention deployment; live repairs, cleanup or recall; binaries/performance/sweeps; replacing ControllerV3; or any full-goal completion clause. The distinct-current-sentence loss and Phase-A backdate characterizations remain unresolved mandatory later work. A read-only design verdict cannot retire them.

## Commands and pins

All test commands ran in the private export with `CGO_ENABLED=0`, `set -o pipefail`, and complete stdout/stderr retained through tee.

- `go test ./internal/memory/resolve -run TestOrderedContract -count=1 -v`: initial four groups PASS (0.679s), expanded five groups PASS (0.532s).
- `go test ./internal/memory/store -run TestOrderedContract -count=1 -v`: one group PASS (0.465s).
- `go test ./internal/memory/resolve -run 'TestDelayedBirth|TestApply_RehomedSpellingCannotResurrectRetiredSlug' -count=1 -v`: eight preceding groups plus unchanged rehome test PASS (0.750s).
- `go test ./internal/memory/resolve -count=1`: initial suite PASS (6.839s), expanded suite PASS (6.710s).

Paths below are relative to `/tmp/scry-ordered-overlay-disproof.ByOqut` unless marked external. Report hash is supplied separately.

| Artifact | SHA-256 |
| --- | --- |
| internal/memory/resolve/ordered_overlay_contract_characterization_test.go | bdab2c3f9cc4009bb7470c3dfbd16f6e973c7ea429b3d2d66492f722d8fd1140 |
| internal/memory/store/ordered_overlay_contract_characterization_test.go | f00c4400b4fe3e70e66fd17eaf18dad79af41f16436a2cb0c32ff61793f83e38 |
| ordered_overlay_contract_characterization_initial.go.txt | 372e98ce1f3ddd597cedf81b44d002a2ae2b0ca8b38ffe3548f7d1d5b64a442a |
| internal/memory/resolve/delayed_birth_baseline_characterization_test.go | d3aacb86ccf92d344f30ae63de0d0c0380fb266168d29469cc3816c4b6e27c29 |
| ordered-contract-characterizations-initial.log | 29baec7b15d5c51ffbce063d5783215664f38e90f9c7cb0d5263d64609d45baf |
| ordered-contract-characterizations-expanded.log | f297fb0c689c9bc7805211b7cf5a46a9571323490ae2e5b7c4ce3874ec175f22 |
| ordered-contract-storage-characterizations-initial.log | ed766074aa38388e8ef763d9d68d0018565046535e3bfb720ab76aae9607bfd7 |
| ordered-contract-preceding-characterizations.log | 592d019628973594a81a25db8224cc6f91f2a602aa86b3e66f7d208abc311d8d |
| ordered-contract-resolver-suite.log | fc05abd241f1e2b0ccbba8e7101e1e7377b5998ad9452e5ed2e253633b24d2aa |
| ordered-contract-resolver-suite-expanded.log | ec2d8b2f0d4a64859ce24b6848aa8b4f23cb1959aee1a9231ef691d0dca2caf8 |
| internal/memory/resolve/resolve.go | 43efc6c16577062cfc497e7c03de8faced93a33ac45b5c9886f932780b2b2623 |
| internal/memory/resolve/aliases.go | 6a96e54997520a10e366f9b1cfc9f3d57beed00c4699de0394456f905e47bdc8 |
| internal/memory/resolve/declared.go | d4172e6594cf900e3d7ddeb2d5a4f67c71fe2313f4a823eae96641652f0c3663 |
| internal/memory/store/store.go | 8093c39a9214e1dbe5546a8cdd5a8cd8c76836952bf2a8616630eeda3ba8f7b1 |
| internal/memory/store/identity_birth_registration.go | 509faba40f6e3f56710d2475b50bddd8091375decff5d80e7cb3aca6147ccb02 |
| external delayed-birth-design-2026-09-06.md | a701291a87c2556f378fa74cb21dbda6732e3ff9e2ded3d7b9b2dcf2709043c0 |
| external delayed-birth-design-independent-review-2026-09-06.md | cb15ac1057e51c6f7dda1dd753e42677f8fab22b725731ed342a30442e0ae1e8 |
| docs/memory-repairs/admission-controller-v3-independent-review-2026-09-06.md | 2521939a134271df7669e073ee836283dd9cbaecc0339fc4df817225f587de10 |
| docs/memory-repairs/identity-mutation-design-independent-review-2026-09-06.md | 94e6e51f71e8caaa49e23e2e7c8a92de556c73099345a4b29d1c904c8219b3d3 |
| docs/memory-repairs/birth-registration-contract-2026-09-06.md | e512d77638a6667f150db4f2aa19c33709a6acfccd87b44f90e1f46624288f15 |
| docs/memory-repairs/episode-selection-contract-2026-09-06.md | b2b935484c4decd70779db1cded42c6048b5484aa4df18ab16ba7f3779d07265 |
