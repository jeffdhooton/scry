# Independent fact mutation ledger disproof — 2026-09-06

Bounded verdict: PASS; no violation of the pinned private ledger contract was
proven. This covers exact transaction-local fa: mutation accounting, complete
baseline reconstruction, owned reports, and refusal/rollback behavior. It does
not certify assertion support, production integration, all-writer enforcement,
identity ownership, deployment, live repair or any complete memory-quality goal.

I ran session orientation and read its output, the complete active objective,
the full frozen ledger contract/source/supplied tests, the reviewed outer-owner
contract and corrected review, the complete reference inventory contract/review,
the strict decoder correction review, and the underlying owner, inventory,
decoder and transaction-guard implementations. No filesystem AGENTS.md was found
on the applicable ancestor paths or in the exported source tree; the supplied
AGENTS instructions applied. The owner contract read from the original candidate
has the exact reviewed SHA256 141ecaa7c5d1b91fd60bbd69eb1f1ae1077b33b148d561cd7f6941dd6758ce0f.

I exported exact baseline 099585964fe6d049520583358f491959b0ba2dcb into the fresh
private directory /tmp/scry-fact-ledger-independent-sep06.1EvdxR. I copied the three
pinned candidate files using apply_patch and verified their hashes before and
after testing. I authored one independent test file only in this export. The
candidate and shared worktree were not edited. All databases used were synthetic.

## Independent evidence

The independent test file defines its own fact fixtures, exact complete raw
snapshot reader, state transition model and independently framed digest. The
digest uses explicit big-endian byte shifts and byte-sorted keys; it does not call
the candidate's inventory or reconstruction implementation.

- Six deterministic sequences of 240 mutations each produce 1,440 exact checked
  operation records. Each record's key, before presence/bytes, after presence/bytes
  and ordinal matches the independent model in execution order. Cases include
  current and historical attributes, edges and self-loops, repeated ordinals,
  no-op puts, deletion/reinsertion and fresh keys. Successful writes through nested
  ordinary AtomicWrite retain the same facade. The final complete raw map and
  independently computed fact count/digest match the committed result.
- Every key/body caller buffer is overwritten after the operation. Every mutable
  report key/before/after buffer and final reference-map entry is mutated after
  verification. Internal history and committed raw bytes remain correct. Opaque
  repeated unknown fields, a huge exponent, a valid surrogate pair, and known
  episodes null-versus-empty arrays retain exact bytes. Opaque non-fa: bytes and an
  existing empty non-fa: value remain unchanged. No helper stages an observer event
  and no observer receives one on success.
- Thirty substantive tamper combinations cover untracked deletion, different valid
  bytes, whitespace-only differences and unknown-extension changes on original and
  initially absent keys, before their first tracked write, after their last write,
  between writes and on untouched keys. Every case first stages a separate valid
  ledger mutation, catches the refusal, and still gets exact whole-store rollback
  with zero events. Two absent-to-absent deletion combinations perform no actual
  byte change and are excluded from the thirty-case count. Restoring the original
  bytes with the first tracked write after an untracked change still refuses,
  proving reconstruction checks the captured baseline rather than just final state.
- Sixteen refusal cases after a successful mutation cover nil/empty key or body,
  key/address mismatch, duplicated known field, unsupported historical time,
  unmatched unknown surrogate, negative ordinal, missing deletion, early verify,
  late begin, repeated verify, explicit body/finalizer errors and earlier poison.
  Ignored ledger errors poison the owner and yield no partial reports. The ledger
  returns its static sentinel while the enclosing owner correctly retains an
  unrelated earlier error rather than replacing it.
- Independent simultaneous lifetimes on two synthetic Stores keep uncommitted
  writes invisible outside their transaction. Committing the second owner and
  either committing or aborting the first preserves isolation. Nil/zero, committed
  and aborted escaped ledger capabilities refuse put/delete/verify. Presenting a
  genuine foreign facade with the original owner identity refuses and poisons that
  target scope without poisoning the separate legitimate original owner. This is
  an explicit capability mismatch probe, not arbitrary internal ledger forgery.
- Body and finalizer panics roll back all raw bytes and expire the ledger. A real
  Badger commit conflict, created by a controlled synthetic competing write after
  verification, preserves exactly that competitor's expected value and none of the
  ledger's writes. A real Badger transaction capacity failure after successful
  mutations preserves the static ErrTxnTooBig classification, leaks no synthetic
  input, and rolls back exactly even when caught and ignored.

The unchanged supplied tests also pass, including guaranteed original deletions
ordered before/between/after retained originals, malformed baseline/final records,
actual oversized-value failure following a successful mutation, ordinary/root
facade refusal, finalizer late-put refusal and caller-buffer/report ownership.

Static inspection confirms the ledger itself is the transaction Set/Delete writer
for fa:, checks exact current raw bytes before each successful operation, and has
no mutation/event API calls for adjacent indexes or other prefixes. The final
verification performs a complete strict raw scan followed by an ordered
reconstruction pass. It retains touched-key states/history and reference counts,
not a complete raw baseline map or Fact slice. There are no production call sites.

## Commands and results

Executed in the private export with pipefail:

```
CGO_ENABLED=0 go test ./internal/memory/store -run '^TestIndependentLedger' -count=1 -v
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestIndependentLedger|TestFactLedger' -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

All pass with exit 0. The final targeted suite takes 2.990s. The final complete
noncached no-CGO suite includes the final independent tests: store 36.311s,
resolve 15.483s, daemon 28.696s. No failing disproof was found, removed, weakened or
converted into an expected failure. Initial and final test logs are both retained.

## Exact pins

SHA256, paths relative to this private export:

| Artifact | SHA256 |
| --- | --- |
| FACT_LEDGER_CONTRACT.md | c8e6074c37e8c89eb3eac34f610e00abc14abb08ee6917e73f4e8c062d2fca95 |
| internal/memory/store/identity_fact_ledger.go | 2040e19a3e0bcf505b6faf8253551e687f6d4dc2059823aeeb744536cb8697c7 |
| internal/memory/store/identity_fact_ledger_test.go | d43613c5f223eac5a5e4321e9caf5b79146282699a7532032709babd33ab3967 |
| internal/memory/store/fact_ledger_independent_disproof_test.go | 2786ed6842b225fb63de7a8fad8e936d39bf24a41aead48606396048f3d4b315 |
| internal/memory/store/identity_admission_owner.go | 14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82 |
| internal/memory/store/identity_reference_inventory.go | d80563bb046ef9721bf42ba2199d6207f38f7629b2c9196a129accefa24877d4 |
| internal/memory/store/identity_fact_references.go | c28aed0f6debabbd1a9d1ff4608b7a88bb4129e59e3eea341b7603d6b6aa3e3e |
| independent-targeted-initial.log | bdeeaec07a959f08a216791fb24e15ba6e53eb0f25398918dc025e6862f2242e |
| independent-targeted-final.log | 7da4ac17f4a8c4f370c823fd99e347f4c0edbc72b2b701908b1c9fca7846d685 |
| independent-fullsuite.log | ee4be898a6430d62a9fe51618ad796188204aa7bf20c7db36e0b807a68cb9fa6 |

## Limits

Ordinals remain descriptive. Actual parsed occurrence and dependency semantics,
preservation of distinct assertions, baseline-zero birth selection, adjacent-index
updates, identity/alias journal attribution and a fixed Store-owned finalizer are
still separate unimplemented production requirements. This unit must not be used
as a support certificate or permission to replace/drop a distinct assertion.

The contract intentionally excludes arbitrary same-package corruption of private
fields, completely reverted transient raw changes, concurrent raw-snapshot
phantoms and raw writes performed after private verification. The controlled real
conflict test does not certify phantom immunity. Nil/zero capabilities have no
associated owner to poison; this review does not reinterpret them as authority
for another active scope. No all-writer integration or public selectable finalizer
is approved by this verdict.

The root separately reported a successful actual restored-replica measurement.
I did not open that replica, run that measurement or inspect real raw data; the
root's result is not independent actual-store evidence from this review. No p95,
live graph, semantic support, cleanup, recall or full-goal result follows. No live
service/provider/configuration/room/remember/sweep/deployment operation was used.
The overall objective remains open.
