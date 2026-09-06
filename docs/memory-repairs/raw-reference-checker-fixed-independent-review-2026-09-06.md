# Independent correction review: raw fact-reference checker

Verdict: PASS for the bounded frozen read-only reference-existence contract after
attempted disproof. The two prior counterexamples are fixed with their original
safety tests unchanged. No additional violation was proved. This review does not
certify ownership, admission, support cleanup, controller integration, live graph
quality, latency, or the complete goal.

## Exact inputs

- Baseline: `06bcaf0731dd4cf5f833e8a1e97776693d3ee716`.
- Unchanged contract `/tmp/scry-raw-support-sep06.kmOW17/SUPPORT_CONTRACT.md`:
  SHA-256 `b425cf14cec45e431b2b8f17b827936b20f455db0c8aa052feaac259cd5d14e8`.
- Corrected `internal/memory/store/identity_fact_references.go`:
  SHA-256 `c28aed0f6debabbd1a9d1ff4608b7a88bb4129e59e3eea341b7603d6b6aa3e3e`.
- Unchanged builder `internal/memory/store/identity_fact_references_test.go`:
  SHA-256 `cb65e265c83fd87c1e752f5d8c88a86a21f52c8dcc325a0dc5c46d2a22075789`.
- Added builder `internal/memory/store/identity_reference_unicode_test.go`:
  SHA-256 `e910103095767deb81b846c7449339005e8908570d902188a23ea66eaa63d8d9`.
- Unchanged original independent `internal/memory/store/reference_independent_disproof_test.go`:
  SHA-256 `fe13e9b86451c718e66e59d1c7c249fbaef2502f11d450e04b81ac5dbb19afd6`.
- Added independent `internal/memory/store/reference_correction_independent_test.go`:
  SHA-256 `4cf2ef0530a2e12e5235100dbd1b374731d679e6e2429ed2a73cf8e5a0da0354`.
- Added independent `internal/memory/store/reference_all_fields_independent_test.go`:
  SHA-256 `f61ac40f1b0136c7a46714c5bb3fd6ea44486de7e3c6e3e6604d8db8a44624de`.
- Separate review export: `/tmp/scry-reference-correction-sep06.dvUaI3`.

The baseline was freshly exported. Only the pinned candidate files and independent
tests were added with apply_patch. The original rejection export remains at
`/tmp/scry-reference-disproof-sep06.bcFjh8`; its original implementation, tests and
report hashes were rechecked unchanged. Its rejection report remains SHA-256
`a2db1f0aa5905efd7452fa3ada7147f64b9ff3e7224a3d00c5a89d4fd0295bbf`.

## Disproof evidence

The original four unsupported InvalidAt cases and four malformed Unicode extension
cases now refuse. The correction checks the optional historical timestamp through
the same exact UnixNano round trip as the address timestamp, and scans JSON string
escapes without decoding or rewriting unknown values.

New independent attempts passed:

- All 65,536 individual escaped UTF-16 code units. Exactly the 2,048 surrogate
  halves refuse; the other 63,488 opaque extension strings remain accepted.
  The decoder leaves each input byte slice unchanged.
- 4,096 valid surrogate-pair cases varying every high and low surrogate against
  both opposite-range boundaries. Uppercase hex, escaped unknown names and nested
  extension string values remain accepted.
- Twenty raw store rows varying slash/quote escape parity and valid Unicode, each
  containing repeated unknown top-level names, nested duplicate members, a huge
  exponent and a huge integer. All twenty count correctly; complete raw snapshots
  remain equal and no observer event is emitted.
- A staged unrelated fact with an out-of-range historical timestamp fails the
  entire transaction-view scan with a zero report. Rollback preserves the complete
  raw snapshot and emits no event.
- All ten known fields present together, including optional value, each duplicated
  using exact, uppercase and escaped-first-character spellings: thirty refusals.

The unchanged independent tests also pass for canonical scalar/array losslessness,
nullable and empty semantics, Unicode casefolded field names, supported historical
timestamp boundaries, valid Unicode, complete scan validation on empty/unrelated
requests, no partial reports, unchanged storage, zero events, and closed stores.
The builder tests retain coverage of raw key/body consistency, historical incoming
references with no adjacency, attributes excluded as destinations, self-loops
counted once, staged writes/deletes, rollback and closed transaction facades.

## Verification

The targeted no-CGO noncached checker suite passed. A complete
`CGO_ENABLED=0 go test ./... -count=1` passed. After adding the final thirty-case
all-known-fields duplicate regression, the same complete command was rerun and
passed with exit 0; memory/store completed in 25.127 seconds. No existing test was
removed, weakened, skipped by modification, or turned into an expected failure.

## Limits

The optional restored-replica measurement was not repeated and its environment-gated
test skipped normally. No independent actual-store or performance claim follows.
No live store, provider, sweep, configuration, deployment, memory write or room write
was used. Test reports expose no real graph keys, values, text or derived slugs.

This is a correction review by the same independent reviewer in a separate export,
not two consecutive fresh-context goal-grading rounds. A successful scan remains a
point-in-time read only. Future integration must independently enforce baseline and
final transaction comparison, preexisting-versus-new reference authorization,
failure poisoning, and prohibition of writes after finalization as required by
the unchanged contract. This verdict authorizes no owner inference or cleanup.
