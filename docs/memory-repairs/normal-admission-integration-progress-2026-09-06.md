# Normal Apply integration — private and incomplete

2026-09-06. Main production source is unchanged. Root private6qYb7K now changes
the normal resolve.ApplyWith entry to a thin adapter into the same fixed owned
transaction. This is real private integration, not shipped prevention. The private
legacy resolver suite is RED and no tests were removed, rewritten or bypassed.

Before this change, the corrected owned implementation was preserved exactly in
`/tmp/scry-owned-corrected-sep06.7Aah7P/code`, with 600 baseline and 107 additions.
OWNED_CORRECTED_FREEZE.json SHA-256
4d8593e4443909eeadc2da88711207704a5f3c5c3a6896b20ee4ed2780796f58.
That snapshot retains the all-repository green result63ad549e and unchanged
independent tests170b621c. It is not independently approved and remains immutable.

## Connected private behavior

Store.ApplyAdmission accepts complete typed original declarations/facts and actual
EP, preserving TypeFallback, original order, aliases, raw date/confidence and full
nil/present Supersedes. It shares executeCompleteAdmission with the retained raw
test entry; no second transaction, racy post-commit read, caller-selected effects
or legacy Phase A/Phase B fallback. Actual counts derive inside the fixed owner and
are returned only after commit. Storage failures return zero output; committed
revision conflict returns typed exported error plus a bounded distinct outcome.

The bounded outcome separates current selected-input/result/counts/status from
the requested input/conflict record and from THIS attempt's physical graph writes.
Duplicate original occurrences can report two committed assertions and one FA
insertion. Exact replay reports complete selected partial counts with zero new
writes or terminal classifications. Newly terminal non-assertions and explicit
model-value verdicts have separate progress counters; these do not claim durable
value evidence has been written (the missing persistence is a regression below).

The adapter maps every extraction field without lexical mutation. Force grants no
authority to re-execute terminal work or replace accepted input. It validates that
a supplied repository cwd agrees with actual EP attestation, and refuses arbitrary
caller-selected exclusivity. The fixed status/replaced_by vocabulary remains the
temporal authority. Existing callers still need complete source/attestation mapping.

## Tests and failures retained

Shared raw-entry refactor plus unchanged independent/actual entry tests PASS3.439s,
log439b38a838f5fae53e4ab2c5fee9ef62bef6d6ac609c9399527582f81bf98a64.
Initial typed transport tests PASS.586s,
log55497cfb0c83f83ce08f3e8794306357da7bcd69380b5aba875248ccedc40df2.

An additional full declaration-count test found the summary expected an impossible
rejected declaration Kind and omitted valid non-identity outcomes. FAIL.489s,
log804eab24ea37017120a425e59c201a6ae1a25fd085b5f3660f757c0e7c92bd91.
Sourcecdaf9455 and testf8e5bfbf preserved exactly. The unreleased summary field was
corrected to DeclarationsNonIdentity and the same total-count assertion retained
with that field name. No fixture condition or expected total changed. Added progress
tests verify carried value classifications are not counted again. Selected typed
and unchanged independent tests PASS2.030s,
log5645f7be2c56809f307f7ef01cf86b7696af09dc3c708d5f96eef7cf7c08d4ba.

Nonfinite input, malformed original EP text and late immutable-result corruption
all refuse with no rows/events/success counts. The malformed EP-text check passed
without a source change: canonical full-EP validation already rejects the lossy
JSON encoding. Do not describe that as a new proven bug or added production fix.
Logccbb12fb57f3d465a83852faf0846f85f9312d5770525f9df775a94ff33b89f0.

Focused NORMAL Apply/ApplyWith tests PASS.779s,
log300289abdd61ea4565c570bfe39666b099638857f1684a7ef4fda41ca6b3ccd0:
actual partial utility, Force exact replay, durable unselected conflict, original
attested/unattested context and refusal of caller temporal authority.
Complete store package PASS78.073s,
loge48ce135e6a880f8d4e1ae223a2d55b04375336e9b727afafa5837648bd5e978; vet PASS.

Unchanged full legacy resolver suite FAIL13.264s, 71 top-level failing groups,
log9415910c85d1cdb244343e0699cd565e9e6ed3e9c43617da323b31f6c1a95729.
The command used a 60-second timeout; it completed normally with failures, not a
timeout. This is NOT a green full repository. Main remains unchanged and green
at the prior production implementation; this private adapter is not promoted.

## Concrete remaining integration work

Do not classify all 71 failures as obsolete tests. Separate real regressions from
fixtures lacking lifecycle adoption/provenance, old error/status expectations and
explicitly superseded backdating/stub/collision/date behavior. Preserve all original
tests and independent safety assertions. In particular:

- Real regression: normal old Apply records explicit type:value evidence for exact
  names/aliases after established-identity and artifact vetoes. The new overlay
  READS existing ve rows but the complete materializer does not WRITE new ones.
  TestParsedDeclaredValuesCannotAccumulateAliasAttestations recreates a later status
  node; TestSupersedesUsesDurableValueEvidenceWithoutRedeclaration loses later hint
  utility. Add original-observation-bound value-evidence planning, complete durable
  receipts, old-row/provenance preservation and terminal replay under the SAME owner.
  Do not call old RecordValueEvidence after commit or use a new lexical dictionary.
  Its accepted effects must participate in changed-revision acceptance; an EP or
  non-identity label alone is not proof of a previous durable classification write.
- The old queue currently logs/deletes partial selections as resolved and does not
  recognize the new typed revision conflict as a deterministic parking reason.
  Correct public partial/conflict reporting and avoid repeated provider extraction;
  preserve existing transcript-retention constraints. Do not silently discard the
  retained parsed input/result or report a conflicting revision as selected.
- memory.commit still has top-level cwd/attestation separate from nested EP; map
  original source context explicitly and reject contradictory witnesses before the
  new entry. Do not let a workspace-shaped string stand in for attestation.
- Old resolver body and pure policy copies still compile though normal Apply no
  longer calls the old body. Complete neutral-policy deduplication and remove old
  production write routes only with explicit fixture/consumer reconciliation.
- Public bounded V2 stored-result/conflict readers, lifecycle adoption/legacy fixture
  handling, every writer/protected namespace/schema floor, rollback incompatibility,
  exact-source independent review and fresh actual-backup replica proof remain.

Current private source pins:

- store identity_assertion_admission_api.go e3d5d475486d9c773351aea4361a2f65e7dcaff1ad46b9bb76f41fc73b91f6aa
- store API test96ed207e8a5494ae118b7bc479dbf4637ec834610df14ecf6becd0f005cf08cd
- store fixed commit8698bd43d3794a5a34d248522129e07e600b69d3d868b90ec923338a012c4f3b
- resolve admission.go4b646fd86acd00b9721c07de6d2777eebc56dda2bcdb5bd8404334286058c4e7
- resolve admission_test.go2ec86e8006dd8be36cd046cd8702f211cf6efaa15c5f1dfd66f13fa86b9fba0f
- resolve resolve.go8ae8bfa63fb7b80a93e58669ffff102663d6e4f3ce7c55f7a515f25897726e1f

No actual memory snapshot/probe/live store, deploy, adoption or new durable note.
Room last179; all original live repair/benchmark/held-out/sweep and two complete
independent grading-round bars remain open. Full goal remains active.
