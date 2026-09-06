# Ordered identity overlay — independent code disproof

2026-09-06. **NO-GO for the frozen private, uncalled, read-only planner.** Three related control/evidence failures violate its finite contract. Six independent test groups contain twenty failing leaf cases. This is not a rejection of the delayed-materialization architecture or a request for another foundational unit. Correct this unit, retain these failures, and obtain a fresh exact-pin review.

The reviewed baseline is `e097fa6cb2e5c5ff8afcbd2b21c163d879856d84`. Frozen manifest `OVERLAY_CODE_FREEZE.json` has SHA-256 `680539f604776a678f2bc1c0eb1a1e61a1b1b04617693cc5d6564a81bc7f1be4`; controlling complete contract has SHA-256 `e56d63aa5b3461b8a4416f32697e9477f4afdd040d15e5c0e4e3cbcf099856ef`. All candidate bytes remained unchanged throughout this review.

The independent export is `/tmp/scry-overlay-code-disproof.kdquPO`. Unless otherwise stated, paths below are relative to that directory.

## Proven violations

### 1. Missing alias identities bypass relevant malformed lifecycle controls

`identity_overlay_lookup.go:165` returns `deferred/missing-alias-owner` when an explicit alias index points to an absent owner. It never calls lifecycle validation for that owner. `relationships` can read the owner's malformed raw controls into witnesses, but reading does not validate them. At lines 171–192, absent natural/normalized/compact alias targets are likewise ignored without examining their controls. The final result is `absent`, allowing an immediate lexical alias proposal.

The ordinary `lookup` path has the corresponding gap: a missing indexed owner's spelling is skipped as an unlisted route, and only the mention's natural identity controls are evaluated. A later canonical-claim conflict returns a normal deferral; status and hint paths can become attributes before that claim check. Malformed relevant owner controls are therefore concealed by an ordinary result.

Independent tests:

- `TestIndependentOverlayMissingAliasOwnerMalformedControls`: seed `al:polar-manual -> absent-owner` and malformed `il:`, `il-consumed:`, `ig:` or `rs:` for `absent-owner`. Declare Atlas(service) with optional alias Polar Manual. All four return a nonzero successful plan with a deferred alias instead of the required sanitized error and zero report.
- `TestIndependentOverlayAbsentAliasNaturalMalformedControls`: place the same four malformed control families at absent `atlas-archive`, then declare Atlas(service) with alias Atlas Archive. All four return a successful plan, with the alias proposed through the own-name containment rule.
- `TestIndependentOverlayMissingClaimOwnerMalformedAcrossRoutes`: malformed `il:absent-owner` behind `al:polar-manual` is missed independently through declaration, ordinary primary, status primary, and supersedes hint routes. All four return success instead of a sanitized refusal.

These are twelve fixed-entry cases, not forged overlay objects or caller-selected owners. The synthetic stores deliberately contain malformed retained controls, which the contract explicitly requires the fixed reader to distinguish from normal absence/deferral. Every case first verifies byte-for-byte raw preservation and zero emitted events. Failures concern the returned authority/dependency description, not a durable mutation.

Required correction: route relevant indexed and natural identities through the complete typed lifecycle/control check before choosing ordinary conflict/absence/value results. Preserve the distinction between validating an old owner and selecting that owner. Do not repair controls or choose a replacement owner.

### 2. A fresh alias proposal can bypass a retired slug without a reviewed rehome

`TestIndependentOverlayAbsentAliasRetiredSlugDefers` seeds valid `rs:atlas-archive` with no `rt:atlas-archive`. A new Atlas declaration proposes Atlas Archive as an alias. The alias is accepted as `contains-own-name`; a later Atlas Archive primary resolves to candidate Atlas rather than remaining dependency-deferred. This differs from the contract's narrowly preserved explicit alias to an already reviewed rehome recipient.

The stronger `TestIndependentOverlayRealRetirementThenAliasRemoval` reproduces the same outcome using actual synthetic-store APIs:

1. Create old `atlas-archive` with canonical name Retired Verdict, and service Recipient listing Atlas Archive.
2. Preview and execute the explicit retirement that rehomes Atlas Archive to Recipient.
3. Remove Recipient's alias with actual `DropAlias`.
4. Adopt the remaining synthetic legacy inventory, then invoke the fixed planner with a new Atlas declaration proposing Atlas Archive and a primary using that spelling.

The retirement and removal leave a valid `rs:atlas-archive`, no corresponding value classification, and no remaining explicit alias route. The planner nevertheless returns `proposed-resolution`, source candidate `atlas`. No raw control-row fabrication is needed for this case. Both retirement tests fail while raw rows and events remain unchanged during planning.

Required correction: keep an existing validated reviewed rehome route usable, but do not let a newly inferred/proposed alias manufacture a new rehome around an absent retired identity. This shares the absent-alias control gap in finding 1. A blanket veto of every spelling whose natural slug was ever retired would break the already passing legitimate rehome case and is not the requested correction.

### 3. Stored evidence borrows the current input's episode exception

`identity_overlay_controls.go:134` returns nil whenever an evidence episode ID equals `registry.revision.EpisodeID`. That exemption is described as applying to this input's proposed vote. However, `valueEvidence`, the legacy `att:` reader, and the generation-specific `iga:` reader call the same helper for evidence already present in the baseline store.

`TestIndependentOverlayStoredEvidenceCannotBorrowCurrentInputEP` has six failing cases: value evidence, legacy votes, and current-generation votes, each with the corresponding `ep:<input ID>` either absent or malformed. All return successful nonzero reports. The value fixture stores `ve:nimbus-core` listing Nimbus Core and only the input's episode ID; despite no valid stored episode proof, its primary becomes a value attribute.

This is not a duplicate-current-vote claim. Vote deduplication still prevents a second current vote, and generation/legacy evidence separation otherwise passes. The violation is treating stored evidence as a new in-memory proposal and skipping its required validation and witness. A proposed current vote can be added separately after validating the stored evidence; it does not need a durable EP claim of its own.

Required correction: validate all episode references obtained from stored rows, including matching-current IDs. Retain actual EP identity/time bytes as dependencies. Keep the current proposed vote separate from those reads.

## Independent coverage and retained passing evidence

I ran `scry memory orient --cwd .` first and read its output. I read the complete active goal, review boundary, controlling contract and incorporated design corrections; the delayed-birth proposal/review and their full characterization tests; the ControllerV3 and identity-mutation reviews; and registration/selector contracts. I inspected all six planner implementation files, supplied planner tests, lexical adapters, the original resolver declaration/alias/primary/hint routines, the genuine registration/owner/finalizer path, relationship and fact inventories/ledgers, recognition/provenance readers, and relevant original store/retirement writers. The complete mechanical port was additionally read and compared by the token verifier across its source manifest; all 157 declarations/248 symbols match under only the documented DTO/name substitutions.

The new passing boundary tests independently cover:

- Distinct canonical names sharing a listed alias follow the valid explicit index owner and retain both old listing witnesses; they are not misclassified as canonical homonyms.
- Discovering Cobalt Atlas before a Borealis alias proposal blocks the later proposal; placing discovery after the proposal preserves that earlier temporal decision. Candidate veto dependencies are retained.
- Inverse relation followed by value-source inversion restores the correct original side. A later rejected primary supplies the earlier hint's candidate. Empty-relation and two-value slots retain complete originals and nil/present SupRef information.
- Failed incompatible declarations cannot expose their aliases/description; a later unknown declaration does not erase the blocked binding; earlier useful Cobalt proposals remain usable and status cannot erase the failed declaration dependency.
- Registry/plan revision bytes, baseline/final inventory rows, candidate/decision observations and primary/hint observations do not share mutable returned buffers.
- Valid retirement aliases with an unrelated retired entity name, optional ID and nested unknown extension remain valid; full raw extension witnesses survive.

The new independent policy test compares fresh synthetic spacing, punctuation, Unicode, artifact/status/path and lexical combinations through original and ported Entity/Declaration adapters, including fallback and unusual type spellings. It also checks the unchanged closed 39-relation vocabulary. This is adapter parity, not a held-out semantic quality grade or permission to tune the word lists.

Every supplied planner group passed independently, including typed failed-declaration replacement; natural exact precedence and canonical homonyms; unsupported/inverse/same-slug discovery; old metadata without a same-episode fact and preservation of all repo refs; current/history-only deferral; stale unlisted claims; generation evidence isolation and malformed relevant generation controls; repeated current votes; report/input ownership/reopen; complete original slots; the 102 invalid-phase shortcut cases; and the genuine discarded-transaction read failure. The original eight delayed-birth and six ordered-contract characterization groups were copied byte-for-byte and passed. Their Phase-A/backdate and current-sentence-loss cases remain baseline characterizations of unresolved later work, not overlay successes.

Static inspection found no raw/normal identity, fact, alias-vote or EP writer in the six planner files. Candidate creation goes through the registry before metadata proposals; later uses link its issued handle. The fixed entry exposes no caller callback, selected owner, support flag or finalizer. It returns no active facade or nonnil registered handle. Full raw equality and zero events hold in the independent successful and failing normal planner fixtures. The underlying genuine registry verifies complete captured final identity/fact inventories and empty writer histories.

The zero-write Badger transaction characterization passes: stale read-only Commit returns nil, while a genuinely writing synthetic transaction can conflict. This review makes no claim of a zero-write commit-conflict check, untracked transient-write detection, arbitrary forged-private-state integrity, concurrent escaped facade safety, or a production all-writer certificate. Supplied private phase/error tests are harness probes; the actual counterexamples above all enter through `planIdentityEpisode`.

## Commands and outcomes

All Go commands ran from the independent export with `CGO_ENABLED=0`, shell `set -o pipefail`, and complete stdout/stderr retained with `tee`.

| Command / log | Result |
| --- | --- |
| `go test ./... -count=1` → `independent-full-suite.log` | Frozen candidate plus supplied tests PASS before new tests: store 66.020s, resolve 16.467s, daemon 29.382s. |
| `go test ./internal/memory/store -run TestIndependentOverlay -count=1 -v` → `independent-tests-initial.log` | Initial three groups fail, nine leaf cases. |
| Same command → `independent-tests-expanded.log` | First four groups fail, fifteen leaf cases. |
| `go test ./internal/memory/store -run 'TestIndependentOverlay(Explicit\|Ordered\|Double\|Failed\|Report\|Valid)' -count=1 -v` → `independent-boundaries-initial.log` | Six new boundary groups PASS, store 0.665s. |
| `go test ./internal/memory/store -run TestIndependentOverlayRealRetirement -count=1 -v` → `independent-actual-retirement.log` | Actual retirement/removal case FAIL, store 0.447s. |
| `go test ./internal/memory/store -run TestIndependentOverlayMissingClaimOwnerMalformedAcrossRoutes -count=1 -v` → `independent-missing-owner-routes.log` | Four route cases FAIL, store 0.583s. |
| `go test ./internal/memory/resolve ./internal/memory/store -run 'TestIdentityOverlay\|TestOrderedContract\|TestDelayedBirth\|TestPrivateIdentityPolicyPort' -count=1 -v` → `independent-supplied-and-characterizations.log` | All selected supplied tests/characterizations PASS; resolve 2.098s, store 5.271s. Differential corpus expanded to 2,295 literals because the byte-copied characterizations were now present. |
| `go run /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/verify_policy_port.go . /tmp/scry-ordered-overlay-contract-sep06.Hc7wdj/policy-inventory-corrected.json internal/memory/identitypolicy` → `independent-policy-token-verification.log` | Exact mechanical token PASS, 157 declarations/248 symbols; no extra copied declarations. |
| `go test ./internal/memory/resolve -run TestIndependentOverlayPurePolicyAdapters -count=1 -v` → `independent-policy-adapters.log` | New adapter test PASS, resolve 0.603s. |
| `go vet ./internal/memory/store ./internal/memory/identitypolicy ./internal/memory/resolve` → `independent-vet.log` | PASS, no output. |
| `go test ./... -count=1` → `independent-final-full-suite.log` | Intermediate full suite FAIL only in the first five new failing groups; store 65.744s. Last route group was added afterward. |
| `go test ./... -count=1` → `independent-complete-full-suite.log` | Complete final test set FAIL only in six new groups/twenty leaves; store 65.547s. All other packages pass, including resolve 16.826s, daemon 27.810s. |
| `ruby verify-independent-pins.rb` → `independent-final-pin-verification.log` | All 600 baseline SHA-256/git-blob hashes and 32 candidate SHA-256 hashes match in BOTH own and frozen exports; contract/manifest pins match. |

No assertion was weakened and no fixture correction was needed. The initial failing test source is retained as `independent-tests-initial.go.txt`; subsequent test changes only added cases and shortened overly verbose synthetic failure output. Every initial failed assertion remains in the final test source. Every failed log remains. No candidate implementation or supplied test was edited. Root later reported additional experiments in its separate correction export; I did not use those reports as independent evidence or grade their source.

## Source ownership, pins and verdict limits

I created the export with `git archive e097fa6 | tar` into a fresh mktemp directory and copied only the 32 manifest additions using apply_patch. The three referenced characterization files were subsequently added byte-for-byte with apply_patch. All authored tests, this report and pin-verification script were created with apply_patch; gofmt only formatted new tests. Logs were retained by tee. No shared/frozen source edits, actual stores/backups/replicas, providers, SSH, live adoption, deployment, sweeps, configuration, rooms, memory writes, or subagents were used. Synthetic adoption/retirement/removal calls exist only to establish test baselines.

The full 32-source pin list is in `independent-final-pin-verification.log`. Exact test, script, source and log SHA-256 values are in `EVIDENCE_SHA256.txt`; report hash is supplied separately to avoid self-reference. New tests are:

- `internal/memory/store/overlay_independent_disproof_test.go` — SHA-256 `a4bcec34e7ffd2f86cc93591ba58e089f130f94b9d329cc63c18100295af6835`.
- `internal/memory/store/overlay_independent_boundaries_test.go` — SHA-256 `8394d3cc245032835273ca8cf0ace4ee66166d42965a773c28a3f261fb03c499`.
- `internal/memory/resolve/overlay_independent_policy_test.go` — SHA-256 `2cb00360626c068fc0ed61f091d168104b697c479870e927cda75ab5f3f84cd5`.

This verdict does not approve either FA phase, virtual backdate/hint eligibility, assertion-address/content collision handling, final factual support, materialization or B votes, final ownership closure, generation/legacy adoption or all-writer enforcement, Force/current-result integration, ingestion counts, ControllerV3 replacement, prevention deployment, live repairs/cleanup, recall, binaries/performance/sweeps, or any whole-goal completion clause. Before production integration, the duplicate lexical policy must still be deduplicated behind the neutral package and the unchanged corpus rerun. These are separate later obligations, not extra prerequisites invented to review this private unit.
