# Frozen generation-ledger primitive: FAIL

2026-09-06. Independent private database-correctness review, not an admission,
deployment or live-state review. Fresh `git archive` of
`a06cd7b9ef9dbfdd8c19eaccae11afb9d07aa897` at
`/tmp/scry-generation-independent.LfeFMc`; the two exact candidate files were
copied with apply_patch. No original source/test expectation was changed. The
candidate and failing tests remain frozen; no implementation fix was made.

## Proven primitive failure

`identity_generation.go:54` accepts nonempty strings without UTF-8 validation.
At `:60` and `:68`, JSON serialization replaces invalid UTF-8 with replacement
characters. The claimed full-tuple encoding therefore loses information before
hashing. `generation_encoding_disproof_test.go:12` supplies distinct EpisodeIDs
`episode-\xff` and `episode-\xfe`; both are accepted and produce identical selector
bytes and generation IDs. This is a serialization collision, not a SHA-256 attack.

The second frozen test at `generation_encoding_disproof_test.go:26` proves a
persistence consequence: a birth with the first ID, an actual own-name fact,
entity and episode passes `finishSupported` and commits. The next transaction
cannot `loadIdentityGeneration`. Deserialization plus canonical reserialization
at `identity_generation.go:144` changes the creation encoding, and the check at
`:145` refuses the selector that this primitive successfully committed.

The smallest next fix is explicit UTF-8 validity checks on all caller-supplied
text accepted into a selector or ledger/session before serialization/staging,
or a deliberately lossless byte encoding. Existing nonempty/normalization checks
alone are insufficient. Keep both reproducers unchanged for corrected review;
add refusal tests for every accepted string boundary and valid Unicode controls.
Normal production episode IDs may be ASCII; this is a demonstrated primitive
input-validation/persistence defect, not evidence that it occurred live.

## Completed boundary evidence

Six independently written test functions with 38 named subcases PASS, as do all
three unchanged builder tests. Independent source is
`internal/memory/store/generation_independent_boundaries_test.go`:

- `:101`: one and eight orphan legacy votes, including opaque unknown JSON bytes,
  remain exactly preserved through supported birth without aliases. Later fresh
  proposals count 1 then 2; 14 complete long fresh IDs survive independently of
  the old cap. Repeated same-episode calls add no votes; aliases and generations
  remain isolated. Direct Badger backup/load followed by Open preserves every
  key/value exactly; replay remains 14 and a new episode yields 15. Existing real
  fact content/provenance is equal and the primitive makes no alias claim.
- `:185`: missing, malformed, unknown-field, wrong-version, wrong-ID, changed-tuple,
  wrong-key/body-slug and noncanonical selectors refuse unchanged. Entity absence,
  exact name drift, one-nanosecond creation-time drift and body-slug drift refuse.
- `:251`: malformed/unknown/noncanonical ledger values, wrong generation/alias,
  empty or mismatched full episode ID, extra key suffix and wrong alias digest
  refuse unchanged. Missing old, birth and current episode provenance causes full
  outer rollback and zero events. These are manufactured key/body mismatches,
  not a claim to have broken a cryptographic hash.
- `:319`: returned errors, panics and a real Badger staging error roll back all
  bytes and events. Caller-owned input buffer reuse does not redirect staged
  ledger data. Attest/finish handles refuse after commit/rollback/panic and in a
  later transaction; a changed selector cannot redirect an active handle.
- `:424`: existing selector/entity, occupied derived ledger, retained selector
  after removal, missing birth provenance and error after provisional selector
  plus ledger staging all refuse/roll back without altering baseline bytes.
- `:467`: deliberately small private Badger transaction capacity makes the
  primitive's own selector Set and ledger Set return `badger.ErrTxnTooBig`.
  Both errors propagate and roll back the entire transaction, with zero events
  and unchanged original facts/legacy records.

Commands run from the frozen export, with `set -o pipefail` for recorded pipelines:

```sh
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestGradeGeneration|TestGeneration' -count=1 -v
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestDisproofGenerationFullTupleEncodingIsLossless|TestDisproofGenerationAcceptedBirthRemainsLoadable' -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

The targeted boundary command PASSes. The focused disproof command FAILs both
frozen assertions. The completed final whole-repository run FAILs only those
two tests in store; every other tested package PASSes, without cached results.
The earlier complete full-suite run also failed only those two assertions; the
final run additionally includes the primitive's own staging-error subcases.

## Exact SHA-256 pins

| File | SHA-256 |
| --- | --- |
| `internal/memory/store/identity_generation.go` | `3dca5a18c9edcbca3e42a5e99833fa1508a2114a130671f5824539e5c0a895f3` |
| `internal/memory/store/identity_generation_test.go` | `7d41645de532dc4cb29b6a6586e9a1af31ca5f5ee1fc4036a53e156668224d77` |
| `internal/memory/store/generation_encoding_disproof_test.go` | `7818567e951fa3cce95d49611af8fe41307add1824feadb9fbbf85de497b07af` |
| `internal/memory/store/generation_independent_boundaries_test.go` | `9baa85e3d32fd6dfb9aec01319846fe8ac674264df7fe859c3bee09a35e8a6e0` |
| `targeted-suite.txt` | `60396362c50bae4b42655004f52eabf21ce7a67f2ea17513c5d35f8b7e2e1900` |
| `frozen-failures.txt` | `1914d29abec3af490c7dca7078de91faba766afad1d6e7d5bc6e44c800cb6d5c` |
| `full-suite-final.txt` | `06c0edf0bc0988d7098d689b09f54d772b3f3ad19794d36393f4ef360128ab22` |
| `full-suite.txt` (earlier run) | `cd6d6ac0fb830ee8212d6d181e2d0062cf625d72351264efaeeec34de689bb31` |

## Strict limits

Reject this exact primitive until the serialization/reload defect is fixed and
independently regraded. Passing database boundaries do not establish production
hollow prevention. `finishSupported` checks binding and episode-key existence,
not actual fact support; callers can omit it or mutate after it. Its convention
is not an unavoidable controller. Admission integration, observations, final
fact support, Force/all-writer coverage, identity lifecycle, legacy adoption and
old-writer reconciliation remain separate NO-GO dependencies. No legacy fallback,
schema bump, retained-writer compatibility or whole-goal pass is approved.

Only synthetic fixtures and local private exports were used. No provider calls,
live graph writes/access, deploy, config, daemon, queue, sweep or shared-checkout
edits occurred. Start-of-session orient was read; applicable task/goal and full
delayed-evidence, corrected journal and prototype contract documents were read.
