# Provisional identity votes: independent bounded disproof

2026-09-06. Private synthetic review; no shared checkout edits, live store, providers,
sweeps, configuration, memory writes, room writes, or deployment operations.

## Pins

- Exported baseline: `06bcaf0731dd4cf5f833e8a1e97776693d3ee716`.
- Contract: `/tmp/scry-provisional-votes-sep06.3YJMlM/VOTES_CONTRACT.md`, SHA-256 `f4a8b6c46026cbf638bdfe6c2a42ce338f7c4d5a6cca1f698c1d1854425e9cc6`.
- Candidate `internal/memory/store/identity_provisional_votes.go`: SHA-256 `ea8b096747432e56f677a97450168c3690ef646105bd7f5cd98f0b872570b0a3`.
- Candidate `internal/memory/store/identity_provisional_votes_test.go`: SHA-256 `e170dd8117a69c59de251af09f773ea1b2eadd5c6c8b99e7f241fe24d8316215`.
- Independent `internal/memory/store/provisional_independent_disproof_test.go`: SHA-256 `49c7e2d9a1ba3b77ec2f90c6b49b43b33a295c85a6b9af1ade11784e018b5ac1`.

The two candidate files were added to a fresh git archive through apply_patch and
have not been changed. The complete active goal and bounded contract were read.
The original baseline plus candidate full suite passed before independent tests.

## Finding: concurrent raw derived-ledger occupancy is inherited

The broad wording "refuse occupied derived iga rows, never overwrite or inherit
them" is not unconditional. `unoccupied()` at candidate lines 66–78 checks an absent
ledger prefix using the owner's transaction snapshot. It establishes no conflict
with a concurrent insertion at a different ledger key.

`TestIndependentProvisionalConcurrentOrphanLedger`, retained unchanged after its
first failing run, performs this deterministic synthetic sequence:

1. Begin a new provisional birth, propose `current-alias`, and stage its entity/EP.
2. From a separate Badger transaction, insert a canonical `iga:` record for that
   exact generation and alias but `old-orphan-episode`, without writing a selector.
3. Materialize and commit the owner. Both return nil.
4. Load the selected generation in a later transaction. Its evidence includes
   `old-orphan-episode`.

The failure is reproducible, including in the required full suite:

```
--- FAIL: TestIndependentProvisionalConcurrentOrphanLedger
    provisional_independent_disproof_test.go:284: occupied derived ledger committed; inherited concurrent old orphan=true
```

Scope qualification: the competing writer is a raw synthetic Badger transaction.
This does not demonstrate an ordinary production producer for such orphan writes;
production controller/API coverage was explicitly excluded. Concurrent legitimate
births that touch the same selector do conflict. If the intended invariant is only
snapshot-visible occupancy with exclusive generation-ledger writer ownership, that
restriction must be explicit and enforced at the eventual production boundary. The
test is evidence of the limitation, not evidence that the uncalled candidate has
already corrupted a live graph. No candidate fix was attempted.

Concrete normal-writer inspection found only the candidate and
`identity_generation.go` producing `iga:` rows. The latter's
`beginIdentityGeneration` stages the same `ig:` selector before any vote and
therefore conflicts with a competing birth. `loadIdentityGeneration` requires a
present valid selector, and its attester rechecks that selector and entity; it
cannot produce an orphan for an absent selector through its supported call
sequence. Both existing generation entrypoints have no non-test callers in this
baseline. `Restore` takes `maintenanceMu.Lock` (store.go:1194), whereas the outer
`AtomicWrite` holds `maintenanceMu.RLock` (store.go:268), excluding restore during
the admission transaction. No current normal producer of the test's isolated
concurrent orphan was demonstrated.

The smallest integration obligation is that every ledger creation, rewrite,
deletion, import and restoration path obey the selector/owner protocol: every
writer must read or stage the relevant selector in the same transaction so that
concurrent incompatible creation conflicts, or run under exclusive maintenance
coordination that excludes admission. Future producer enumeration and lifecycle
testing must establish this before production approval. An additional prefix
scan in the same snapshot cannot enforce arbitrary raw-writer exclusion. This is
a contract boundary, not a proposed format change or removal of the disproof.

## Verification

- `CGO_ENABLED=0 go test ./... -count=1` before independent additions: PASS. This
  included the candidate's six tests and thirteen phase/failure cases.
- The same full command with all independent tests: FAIL solely on the concurrent
  orphan test; other packages pass. Exact output: `INDEPENDENT_FULL_SUITE.log`.
- Final targeted independent command: eight top-level tests, seven PASS and the
  same one FAIL. Exact output: `INDEPENDENT_TARGETED.log`.
- The failure was also reproduced in a separate targeted run without changing it.

Independent passing coverage includes zero transaction writes at begin/attest;
exact finalization diff containing only one reviewed `ig:` and deduplicated `iga:`
rows; no alias claim; twelve raw EP ambiguity/identity/time refusal cases with
caught-error rollback; seven canonical entity/binding cases, accepting mutable
type/description/repo refs; actual selector commit conflict with complete raw-map
rollback and zero events; real `ErrTxnTooBig` after a selector and a first vote are
visibly staged, followed by complete rollback despite the caught error; complete
birth tuple separation and equal-instant timezone canonicalization; caller value
and selector-buffer independence; expired-session refusal; repeated unselected
identical and changed occurrences; eight legacy votes retained byte-identically;
no unselected-alias inheritance; and a later distinct episode raising the selected
alias count from one to two.

The parent disclosed its initial three candidate-fixture failures from unnormalized
space-containing aliases and subsequent normalization plus phase-counter repairs.
Those are not erased by this review. This review received only the pinned corrected
candidate tests; it did not independently execute the earlier fixture revision.

## Limits and verdict

This review does not certify support policy, observations/dispositions, adoption,
production integration, hollow prevention, or the broader existing generation
helper. In particular, omitting materialization does not undo a body-created entity,
and selecting a fixture with no facts is not a fact-support claim. The existing
generation helper remains available and unchanged.

Initial contract verdict: NO-GO for unconditional cross-writer occupied-ledger /
no-inheritance wording. The candidate obeys the tested single-owner and
snapshot-visible boundaries, but that broader wording is disproved against a
concurrent raw ledger writer. There is no demonstrated production path defect
from this test. Retain the failure and explicitly bound the contract plus require
cooperative-writer enforcement before describing the narrower contract as passed.
No complete augmented-suite PASS or live-goal approval follows.
