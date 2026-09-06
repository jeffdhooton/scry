# UTF-8 and UTC generation primitive: bounded PASS

2026-09-06. Independent private serialization/database-correctness review of exact
source `e07c6c50affce548b31a94dc2e94d665a49ccab1d246f9433f49277535619592`.
Attempted disproof finds no remaining failure within the tested private primitive
contract. This is not admission, production prevention, format-adoption or deployment
approval.

A third fresh `git archive` of base
`a06cd7b9ef9dbfdd8c19eaccae11afb9d07aa897` was created at
`/tmp/scry-generation-utc-grade.ICMxGb`. Exact source and all prior tests were copied
with apply_patch. No implementation or existing expectation was edited. Both earlier
rejected exports, reports and failing reproducers remain frozen and unchanged.

## Change and evidence

The only source change since rejected `3c196ab...` is
`identity_generation.go:70`: canonicalize the copied birth timestamp to UTC before
encoding. Generation timestamp identity is the exact instant, consistent with
Entity.CreatedAt.Equal; timezone label/offset representation and monotonic clock
data do not select identity. No entity/fact timestamp write was introduced.

All three unchanged serialization reproducers PASS: the two invalid UTF-8
identity/reload tests and the distinct second-offset instant test. The unchanged
38 original independent boundary subcases and 46 independent text cases PASS,
along with the three original builder tests and builder UTF-8 test.

New `generation_utc_independent_test.go` provides:

- `:16`, seven offset subcases: selector JSON preserves exact instants and
  987654321 nanoseconds at positive/negative offsets containing seconds. The
  same instant rendered in four different offsets has identical selector bytes
  and generation ID. A one-nanosecond change and a one-second-offset change in
  otherwise identical wall clocks each remain distinct.
- `:59`, durable reload: a selector born from a second-offset timestamp binds to
  an entity materialized at the same exact instant in a lossless representation.
  Later it also loads when that entity stores an equivalent ordinary hour offset.
  Attestation does not change any preexisting raw key/value. Full Backup, direct
  Badger Load and Open preserve all bytes; replay remains one vote, and a new
  episode gives two. Existing entity, fact, selector and legacy attestation bytes
  stay unchanged across this sequence.
- `:135`, two lossy-entity refusal subcases: supplying the original second-offset
  timestamp directly to the existing Entity JSON writer loses the offset seconds.
  Provisional finish then refuses with full rollback/zero events. If an ordinary
  writer later stores such a lossy timestamp on an already supported entity,
  generation loading refuses while preserving that entire preexisting raw state.

The last result is a refusal boundary, not a fix for legacy Entity serialization.
The primitive does not repair its timestamps or infer the intended original instant.
The durable success fixture explicitly uses a losslessly representable entity
timestamp at the same instant; this is fixture preparation, not hidden behavior
inside the primitive.

The prior unchanged tests retain their earlier coverage: legacy orphan vote counts
1/8, supported birth with no aliases, delayed counts 1 then 2, full fresh IDs beyond
the legacy cap, duplicate/alias/generation isolation, raw backup/reopen, malformed
selector and ledger/key/body/digest refusal, missing episode provenance, stale and
expired handles, owned buffers, provisional occupancy, rollback/panic, actual
selector/ledger staging errors, and unchanged facts/legacy bytes and zero failure
events. Valid Unicode remains lossless and malformed input refuses before staging.

## Completed execution

Commands from the new export, with `set -o pipefail` on recorded pipelines:

```sh
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestGradeGeneration|TestGeneration|TestDisproofGeneration' -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

Targeted command: 19 top-level tests PASS. Full repository command: every tested
package PASSes, including unchanged store, resolve, daemon, queue, recall and sweep.
No package result is cached. Complete logs are `targeted-suite.txt` and
`full-suite.txt`. No prior test or expectation was weakened or skipped.

## SHA-256 pins

| File | SHA-256 |
| --- | --- |
| `internal/memory/store/identity_generation.go` | `e07c6c50affce548b31a94dc2e94d665a49ccab1d246f9433f49277535619592` |
| `internal/memory/store/identity_generation_test.go` | `7d41645de532dc4cb29b6a6586e9a1af31ca5f5ee1fc4036a53e156668224d77` |
| `internal/memory/store/identity_generation_utf8_test.go` | `edd4d2599cf988e43a56f364eff2d2a7ff38977f8ec9ad229febc4a19a745df5` |
| `internal/memory/store/generation_encoding_disproof_test.go` | `7818567e951fa3cce95d49611af8fe41307add1824feadb9fbbf85de497b07af` |
| `internal/memory/store/generation_independent_boundaries_test.go` | `9baa85e3d32fd6dfb9aec01319846fe8ac674264df7fe859c3bee09a35e8a6e0` |
| `internal/memory/store/generation_time_encoding_disproof_test.go` | `77a908997934d12b30d5f3c9f5f285c60a7a4714282d153fdef28b24b71ea19f` |
| `internal/memory/store/generation_utf8_independent_test.go` | `1b1a0b046dcb9574f2778fd8f9f1227fc87cf03bfc3213521641b734804dcabc` |
| `internal/memory/store/generation_utc_independent_test.go` | `46133c1b04a548daedba3ebacee1b1a7e87029c9b44340d827a536e4cceb1d2b` |
| `targeted-suite.txt` | `fb10d84beea15a394e42b896d80467ea4baa53c986c00107554a29e2770a4ae7` |
| `full-suite.txt` | `c9a2571b64d09c70766a79c7b09632694d52b93ad0798dcc03b52cc3de301cbe` |

Prior report hashes verified unchanged:

- `/tmp/scry-generation-independent.LfeFMc/REPORT.md`:
  `c0e64ba1c628a4e1f4d106c48e6cea10b61b7b8cdf44935a1dfd2b286a9021a5`.
- `/tmp/scry-generation-corrected-grade.DB9Vjn/REPORT.md`:
  `b18b1fc17920826c9b0e7af83556ed499d737f46c06fd7656c4571f1a9906dfd`.

## Strictly bounded interpretation

No further primitive fix is proven necessary by this review. Retain these exact
regressions with any future private revision. No production caller uses this
primitive, and this PASS is not a claim that production hollows are prevented.

finishSupported remains a caller convention: it checks binding and episode-key
existence, not fact support, and callers can omit it or mutate afterward. An
unavoidable final controller, observations/useful unresolved evidence, complete
writer and Force coverage, identity lifecycle, safe format/adoption boundary,
legacy fallback decisions and old-writer reconciliation remain separate NO-GO
dependencies. Existing malformed or lossy entity state is refused, not repaired.

No live access, provider call, integration, schema/config change, deployment,
daemon/queue action, live sweep, shared-checkout edit or implementation fix occurred.
