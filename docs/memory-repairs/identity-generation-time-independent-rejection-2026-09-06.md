# UTF-8-corrected generation primitive: remaining tuple-encoding FAIL

2026-09-06. Separate fresh `git archive` export of
`a06cd7b9ef9dbfdd8c19eaccae11afb9d07aa897` at
`/tmp/scry-generation-corrected-grade.DB9Vjn`. Exact private source/tests were
copied with apply_patch; neither implementation nor existing test expectations
were edited. The original rejected directory `/tmp/scry-generation-independent.LfeFMc`
and its report/reproducers remain unchanged.

## UTF-8 correction verified

The source adds validation before tuple serialization (`identity_generation.go:58`),
before accepting load slug/session episode (`:138`), and before attesting an alias
(`:225`). Ledger reads retain complete key/body/canonical-byte checks (`:213`).
Generation IDs are generated hex and checked for equality, not arbitrary accepted
input. Malformed raw JSON text cannot gain acceptance merely because Unmarshal
replaces it: exact canonical byte comparison still refuses the original bytes.

Both original UTF-8 reproducers now PASS unchanged. All six previous independent
boundary tests with 38 named subcases PASS unchanged. The three original builder
tests and new builder UTF-8 test PASS.

An additional independent test file covers 46 new named cases:

- `generation_utf8_independent_test.go:12`: five malformed UTF-8 sequences across
  all seven accepted caller string boundaries (four birth fields, load slug,
  load episode, alias). Each explicitly refuses with no staged persistent state,
  even when this validation-only error is deliberately followed by callback commit.
- `:66`: seven valid Unicode/control-character cases round-trip exact birth names,
  birth/session episode IDs and normalized aliases; replay stays one vote and
  selector bytes remain unchanged. Cases include replacement characters, composed
  and combining accents, NUL, supplementary-plane characters, line separators
  and U+FFFF.
- `:100`: raw invalid UTF-8 and unpaired JSON surrogates in both selector and ledger
  refuse without modifying any key/value.

This supports the specific UTF-8 fix. It does not settle every field in the tuple.

## New frozen serialization counterexample

`generation_time_encoding_disproof_test.go:10` constructs two otherwise identical
births with CreatedAt wall clock `2026-09-06 01:02:03`, one at a fixed offset of
one second and one at two seconds. Go `Time.Equal` confirms that these identify
different instants. Both pass `identityBirthRecord`, yet their selector bytes and
generation IDs are identical. RFC3339 JSON serialization drops the offset seconds
at `identity_generation.go:68`, losing the distinction before the digest at `:75`.

This is another serialization collision, not a cryptographic hash collision.
The failure is narrower than the previous UTF-8 reload failure: this test proves
accepted distinct creation instants collapse in the tuple encoding. It does NOT
prove that such a birth can pass entity-binding finalization and commit a bad
record, nor that production episode timestamps exercise this case.

The smallest next change is to define the creation timestamp identity as the exact
instant, and canonicalize that tuple field to UTC before fixed encoding, preserving
seconds and nanoseconds. That matches the existing Entity.CreatedAt.Equal binding
semantics and need not change stored entity/fact timestamps. Alternatively refuse
non-lossless time encodings explicitly. A new revision must retain the frozen test
and verify distinct second-offset instants, equal instants under different offsets,
nanosecond separation, and persistence. No such implementation was made here.

## Executed tests and outcome

All commands used `CGO_ENABLED=0`, `-count=1`, and `set -o pipefail` for log pipelines:

```sh
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestGradeGeneration|TestGeneration|TestDisproofGeneration' -count=1 -v
CGO_ENABLED=0 go test ./internal/memory/store -run TestDisproofGenerationAcceptedTimeEncodingIsLossless -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

`targeted-suite.txt` contains 15 top-level PASS tests, completed before the time
reproducer was added. `full-suite.txt` is the completed noncached full-suite PASS
from that same stage. The new focused time test FAILs in `frozen-time-failure.txt`.
The final whole-repository run, including that unchanged time reproducer, FAILs
only `TestDisproofGenerationAcceptedTimeEncodingIsLossless` in store; all other
tested packages PASS. Its complete evidence is `full-suite-final.txt`. None of the
package outcomes are cached. Earlier passing logs are not the final verdict.

## SHA-256 pins

| File | SHA-256 |
| --- | --- |
| `internal/memory/store/identity_generation.go` | `3c196ab9d3067c50d26fb839bb65fd7820e0d363da6818ef2a659f93df7680dc` |
| `internal/memory/store/identity_generation_test.go` | `7d41645de532dc4cb29b6a6586e9a1af31ca5f5ee1fc4036a53e156668224d77` |
| `internal/memory/store/identity_generation_utf8_test.go` | `edd4d2599cf988e43a56f364eff2d2a7ff38977f8ec9ad229febc4a19a745df5` |
| `internal/memory/store/generation_encoding_disproof_test.go` | `7818567e951fa3cce95d49611af8fe41307add1824feadb9fbbf85de497b07af` |
| `internal/memory/store/generation_independent_boundaries_test.go` | `9baa85e3d32fd6dfb9aec01319846fe8ac674264df7fe859c3bee09a35e8a6e0` |
| `internal/memory/store/generation_utf8_independent_test.go` | `1b1a0b046dcb9574f2778fd8f9f1227fc87cf03bfc3213521641b734804dcabc` |
| `internal/memory/store/generation_time_encoding_disproof_test.go` | `77a908997934d12b30d5f3c9f5f285c60a7a4714282d153fdef28b24b71ea19f` |
| `targeted-suite.txt` | `c95a6bc2647c9b5b1ea49ac9cc1fa7df96ae41685e9b4a3f7d06a8de6c109d14` |
| `full-suite.txt` (before time test) | `4cba6638de1651a2eeb989cffaed663e4038096b42f0a2111651da66f5db39d4` |
| `frozen-time-failure.txt` | `4ca693769a689f503f70e83eb368d2f9a870d8323e55dd18b6f1c9be6a210451` |
| `full-suite-final.txt` | `24b1bcf28adc3e5a9b3baf6bbc1f5aaf026176b8223a477c6b5274cf9ab554c6` |

## Bounded verdict

Reject exact source `3c196ab...` for its remaining lossless-tuple claim. The UTF-8
failure is fixed; the time-field failure is distinct and remains frozen here.
No live access, provider call, deployment, integration, schema/config change,
shared-checkout edit or implementation fix occurred.

Even a corrected primitive PASS would certify only attempted disproof of this
private selector/ledger contract. Optional finishSupported is not an unavoidable
controller and does not prove fact support. Admission, observations, useful evidence
retention, all-writer/Force coverage, lifecycle, format adoption, legacy fallback,
old-writer reconciliation and production hollow prevention remain unapproved.
