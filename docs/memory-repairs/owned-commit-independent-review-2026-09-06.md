# Bounded independent owned-commit disproof: NO-GO

Reviewed the frozen private `applyCompleteAdmission` entry against the complete 358-line ASSERTION_CONTRACT.md. This verdict concerns the actual private composed transaction only. It is not normal resolve.Apply, production, an all-writer floor, adoption, actual-backup benchmark, or whole-goal approval.

## Finding: unchanged partial replay advances durable selection after its own births

The first independent test run failed `TestOwnedIndependentNewBirthProducerAndTerminalVotes`. Follow-up reduction reproduces the failure with **no declarations or votes**, just two original occurrences using previously absent Draco/Equuleus endpoints:

1. Ordinal 0: Draco uses Equuleus, statement `unsupported first discovery`, raw explicit start `bad-date`, confidence .8.
2. Ordinal 1: Draco uses Equuleus, statement `independent later support`, default episode start, confidence .8.

Both calls execute the real `applyCompleteAdmission` entry against a private Badger store. The first commits ordinal 1, its FA and supported endpoint generations, and durably defers ordinal 0 as `invalid-explicit-start`. Calling the identical original input and EP immediately afterward leaves those dispositions, facts, identities and votes unchanged and emits zero events, but appends a new immutable `io-episode:` result and rewrites `io-head:` to select it. The selected receipt adds predecessor lineage. A third call stabilizes.

The reduction with no declarations appends a 27,021-byte successor result. The one-declaration and two-declaration cases append 106,338 and 146,018 bytes respectively. These sizes are observed in diagnostic-first.log, not estimated. Its logs preserve complete before/after assertion receipt arrays, new selected result addresses, and the changed head address.

The deferred occurrence's original source/destination routes have Kind `candidate`; on the second call they have Kind `generation`, with the same slugs and exact/declared semantics. The initial transaction itself established those generations. `admissionDeferredAssertionSemantic` in `internal/memory/store/identity_assertion_admission_replay.go:287` discards evidence, predecessor, trace and producer positions but retains route Kind. Its comparison therefore prevents exact receipt reuse. `prepareEpisodeResult` in `identity_assertion_admission_result.go:331` treats the resulting receipt difference as a successor. `admissionCommitProgram.finish` then persists that result and advances the head.

This violates the controlling contract's same-input rule: unchanged results reuse immutable records and the exact head; actual disposition change is required to append/select a complete successor. It also violates the explicit own-write partial retry claim exercised by the supplied suite. No repair was attempted, assertions were not softened, and no fixture correction was needed. The no-deferred diagnostic control passes, showing that supported birth creation alone is not enough: retryable deferred receipt recomputation triggers the failure.

## Independent transaction evidence

The unchanged supplied store package suite passes (67.110 seconds). The independent first suite passes all 18 table subcases across preservation and temporal/address components, plus the actual hint replay/backup/reopen/conflict scenario, and fails the new-birth partial replay scenario above. Diagnostic reductions fail with one declaration, two declarations and no declarations; the no-deferred control passes. The separately added transaction suite passes all five top-level tests, including four hint eligibility/refusal subcases.

Actual owned transactions exercised:

- Canonical historical/current exact repeats, exclusive transitions and SupRef rewrites; confidence/provenance and historical interval checks; all four routes with unknown raw FA extensions that defer locally and preserve exact FA bytes; pre-existing opaque adjacency survives every route; unrelated supported births and utility commit.
- Both input orders for ordered exclusive transitions, conflicting equal starts, same legacy address conflicts, same-target distinct sentences and exact duplicate max-confidence coalescing. Exact replay is checked against the complete raw database and zero events.
- A committed hint plus an unrelated deferred date, insertion of a later-arriving eligible target, exact terminal replay, real Badger Backup/Restore and disk close/reopen, followed by durable changed-revision conflict preserving every old row and exactly adding input/conflict records.
- New birth producer dependencies, actual generation rows and exact support references, duplicate generation-domain votes with no legacy vote rows, and unchanged replay (the failing case).
- An existing ownership hole, coalesced declaration metadata and legacy duplicate vote, a deferred type upgrade, later explicitly synthetic ownership repair, actual successor progress, and terminal metadata/assertion suppression. The pending type commits without refilling deliberately cleared already-accepted description.
- Late immutable-result corruption after a read-only predicted result address: the real entry refuses at selection and leaves complete raw rows unchanged, exposes no sensitive sentinel in errors, emits zero events, then succeeds after the synthetic blocker is removed.
- Zero-accepted selected revision: changed EP source witness refuses without mutations; a changed extraction with the original EP can select useful materialization while preserving old input/result/EP bytes.
- SupRef 1971/2002 candidate universe at a 1980 event, temporal filtering before uniqueness, ambiguity, immutable closed history, resolved self-loop local deferral, unrelated actual utility and exact retry.
- Concurrent complete revisions under the real coordinator: exactly one selected result and one durably retained conflict, one selected head at revision 1.

All data is synthetic and private. Supplied fixtures are used for store setup, canonical encoding, time parsing and reading raw rows; assertions and real-entry scenarios are independently authored. No supplied tests or implementation files were changed. No live store, configuration, provider, room, deployment, external service or additional agent was used. Concurrency tests exercise real serialized selection, not an injected mid-transaction Badger CAS conflict; the late rollback test injects a synthetic immutable result occupant, not storage hardware failure. I did not run the whole repository suite or use an actual user backup.

## Exact inventory and reproducibility

Export: `/tmp/scry-owned-independent-sep06.zvCJIZ/code`.
Frozen authority: `/tmp/scry-owned-commit-review-sep06.xfHydo/OWNED_COMMIT_FREEZE.json`, SHA-256 `a8976cf6bb9c99b0e9d76034781647e74048f458903461034540275c70f4bbec`.
Contract: `/tmp/scry-owned-commit-review-sep06.xfHydo/ASSERTION_CONTRACT.md`, SHA-256 `20d5ed6f98903571560f8ffd441ae5a76e6634815992bb90953a5a630218fcb9`.

The private export was created from `git archive e097fa6`; all 99 authored supplied additions were copied through apply_patch. The verifier checked all 600 baseline and 99 supplied addition SHA-256 values in both frozen and exported trees before and after testing. Independent tests add three new files only. Both manifest logs match.

Commands, from the export, all with `CGO_ENABLED=0 TMPDIR=/tmp/scry-owned-independent-sep06.zvCJIZ/tmp`:

- `go test ./internal/memory/store -count=1` (before adding independent tests; supplied-store.log).
- `go test ./internal/memory/store -run '^TestOwnedIndependent' -count=1 -v` (first independent file only; independent-first.log).
- `go test ./internal/memory/store -run '^TestOwnedIndependentBirthReplayDeltaDiagnostic$' -count=1 -v` (diagnostic-first.log).
- `go test ./internal/memory/store -run '^TestOwnedIndependent(Optional|Late|NoAcceptance|HintEligible|Concurrent)' -count=1 -v` (transactions-first.log).

Paths below are relative to `/tmp/scry-owned-independent-sep06.zvCJIZ`:

| Artifact | SHA-256 |
| --- | --- |
| `code/internal/memory/store/owned_independent_disproof_test.go` | `98d0b34616f6b48c4afa9a081e6bdb39642fc1e064f9daad687755cfb5dc9773` |
| `code/internal/memory/store/owned_independent_diagnostic_test.go` | `4ee7b807536afc45eaca4ccc54b234e62c9d40c5c2d800e577949449d2e80ed7` |
| `code/internal/memory/store/owned_independent_transactions_test.go` | `97e0886abc77a49e5ae28918e7afc235310a3444f788804b1bd408e45270100f` |
| `independent-first.log` | `24eed091656b48fdd04ee9d8a4f854bb4c64ade74aeecc1b46a8dce46a9986fb` |
| `diagnostic-first.log` | `89c39d7b847b8bef4f03e6a7c30b3cc74fbb3225562d1fb2ffa5e1e1564dc6f2` |
| `transactions-first.log` | `4fe90d3e134663d438e8234993f64fc48fc90bb8d1a8f5c0cf219468ffb4efc4` |
| `supplied-store.log` | `a56f7b98386bcc3e269ff429131b661363152e17ab6ec455d4f836fba7e391a1` |
| `manifest-before.log` | `b85ee43ac84b170c8d02e77b90659e525896fd1a657327850e7b0b2d5b8afed2` |
| `manifest-after.log` | `b85ee43ac84b170c8d02e77b90659e525896fd1a657327850e7b0b2d5b8afed2` |
| `setup.rb` | `0a15b05358573d696f83895b994c968954a6b1941505d53aab2e738bcc0c7bfc` |

The first independent source and failure hashes were captured immediately in `first-evidence.sha256`. There are no fixture corrections. All failed tests remain unchanged and failing against this frozen source. This is a bounded NO-GO for unchanged partial replay; successful conservation probes do not override that result.
