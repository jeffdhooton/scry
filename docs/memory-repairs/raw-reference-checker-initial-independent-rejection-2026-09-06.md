# Independent raw fact-reference checker rejection

Verdict: REJECTED against the frozen contract. This is a bounded review of an
uncalled, read-only raw-reference checker, not owner/admission policy, support
cleanup, merge safety, live graph quality, or performance certification.

## Exact reviewed inputs

- Baseline: `06bcaf0731dd4cf5f833e8a1e97776693d3ee716`.
- Contract: `/tmp/scry-raw-support-sep06.kmOW17/SUPPORT_CONTRACT.md`, SHA-256
  `b425cf14cec45e431b2b8f17b827936b20f455db0c8aa052feaac259cd5d14e8`.
- `internal/memory/store/identity_fact_references.go`, SHA-256
  `b8c6c626120f90c82743396364b7317872284ec86a41f8056772fe8f78d4a886`.
- Builder tests `internal/memory/store/identity_fact_references_test.go`, SHA-256
  `cb65e265c83fd87c1e752f5d8c88a86a21f52c8dcc325a0dc5c46d2a22075789`.
- Independent safety tests `internal/memory/store/reference_independent_disproof_test.go`,
  SHA-256 `fe13e9b86451c718e66e59d1c7c249fbaef2502f11d450e04b81ac5dbb19afd6`.
- Independent export: `/tmp/scry-reference-disproof-sep06.bcFjh8`.

The complete goal objective and contract were read after session orientation.
The baseline was exported independently and only the two pinned candidate files
were copied with apply_patch; the independent tests were then added. Candidate,
baseline, root workspace, and initial independent safety tests remain unchanged.

## Proven counterexamples

1. Unsupported historical timestamp accepted. Decoder line 106 checks the
   UnixNano round trip only for ValidFrom. A canonical nonnil InvalidAt is accepted
   at zero time, year 2500, one nanosecond before the minimum UnixNano instant, and
   one nanosecond after the maximum. All four cases fail
   `TestIndependentReferenceInvalidAtRange` (independent test line 14).
   The contract requires rejection of unsupported timestamps; the checker can
   currently count these malformed records as historical references.
2. Lossy Unicode escapes accepted in extension members. Decoder lines 37 and
   53–86 validate literal UTF-8 and round-trip known values, but leave escaped
   Unicode in unknown members unchecked. Four fixtures containing an unpaired
   surrogate in a top-level unknown name, unknown string value, nested object name,
   or nested array string all pass decoding. These fail
   `TestIndependentReferenceLossyUnknownUnicode` (independent test line 25).
   The contract explicitly refuses lossy Unicode escapes. Unknown fields can
   retain their exact bytes and huge numbers while their JSON string encoding is
   still checked for well-formed surrogate pairs.

These are preserved failing safety regressions, not expected-success assertions.
No graph content or raw key was printed to demonstrate the findings.

## Verification

`CGO_ENABLED=0 go test ./internal/memory/store -run TestIndependentReference -count=1`
failed only the two tests above, with four cases in each. Five other independent
tests passed: known scalar/array losslessness; nullable and empty-field semantics
plus supported historical timestamp boundaries and valid Unicode; escaped Unicode
casefolded known names and duplicate refusal; complete malformed-row validation
even for empty/unrelated requests with raw equality and zero events; closed-store
refusal with no partial report.

`CGO_ENABLED=0 go test ./... -count=1` completed with exit 1. Every package other
than memory/store passed or had no tests. Memory/store failed only the same two
independent tests (eight failing cases); no existing test failure was reported.
The four builder synthetic test functions also exercised exact key/body mismatch,
known duplicates/case/escapes, missing and malformed fields, historical incoming
facts without adjacency, attributes excluded as destinations, self-loop counting,
staged deletes/inserts, rollback, expired transaction facades, all-record failure
without partial reports, opaque extensions, timestamp address bounds, and unchanged
raw storage with zero events.

## Scope and limits

No live store, provider, sweep, configuration, deployment, memory write, or room
write was used. The optional historical replica measurement was not repeated;
the environment-gated builder measurement skipped in the full suite. Root's
reported replica timing and row counts are not independent evidence from this
review and do not certify latency or p95. No cleanup or controller integration is
authorized by this verdict. A correction requires new source pins and rerunning
the unchanged independent safety regressions before any passing claim.
