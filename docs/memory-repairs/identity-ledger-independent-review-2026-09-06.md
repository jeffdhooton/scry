# Complete identity mutation ledger — independent bounded disproof

2026-09-06. GO for the pinned private, uncalled accounting boundary under its complete frozen contract. No contract violation was proved by the independent synthetic tests or source review. This is not approval for ownership transfers, production routing, undo, materialization, cleanup, deployment or any live/whole-goal clause.

I ran session orientation and read the output before other work, then read the complete active objective and the complete candidate contract/source/supplied tests. I read the full archived relationship inventory, writer history and identity-mutation design reviews (pins below), strict legacy decoder, actual writer/journal source, owner harness, transaction guard, Store view/update/AtomicWrite, normal identity mutation methods and candidate references. The only non-test references to this ledger are its own implementation. No producer is routed through it.

I independently exported baseline `699da3bd9cc271e266f7c53b5f5ad78e7f124fa0` with git archive into `/tmp/scry-ledger-disproof-PmmAQv`, copied only the three frozen additions using apply_patch, and verified every supplied hash. I authored one additional synthetic test file and this report via apply_patch in that private export. I did not change the implementation, supplied tests, candidate directory or shared checkout. No real store/replica, provider, config, room, remember, sweep, deployment or other external write was used. All database operations below use newly created synthetic temporary stores. Test logs retain complete command output.

## Executed evidence

With shell pipefail enabled:

```
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestILIndependent|TestIdentityMutationLedger' -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

The targeted run passed on its first execution, exit 0, store 3.115s: seven independent top-level groups and five unchanged supplied groups. The full noncached no-CGO suite passed on its first execution, exit 0: store 42.082s, resolver 14.820s, daemon 29.231s. There were no failed executions in this export and no assertions were removed or weakened. Root's separately reported test results are not substituted for these executions.

## Independent findings

1. Three deterministic runs exercise 999 operations across three actors, shared aliases and entity keys, including entity deletion/recreation, alias replacement by another actor, repeated revisions and caller-buffer overwrites after every operation. A separate model checks every exact sequence/actor/key/before/after tuple, first/final images, full owned baseline/final selected maps and whole committed store bytes. Existing empty alias presence and opaque binary control keys/values survive. Zero observer events are emitted.

2. Twelve entity drift cases cover replace/delete/present-empty before the first tracked operation, between tracked operations, after the last operation and on another key. Eight more cases change equal-length binary values on invalid-UTF-8/NUL-suffixed alias/control keys across all selected nonentity families. Every refusal rolls back the entire scope; verification returns no partial report. Together with the supplied 40 alias/control transitions, these exercise both presence and byte comparisons, including unchanged map sizes and value lengths. The verifier uses raw full-map equality, not digest/count equivalence.

3. The report isolation test changes baseline raw buffers, parsed alias/RepoRefs arrays, normalized/natural listing arrays, reverse-index arrays and counts; final raw/projection data; and all history entry keys/before/after values and summary-map values. Baseline mutations leave Final unchanged, listing projections remain independent, entry mutations leave history summary maps unchanged, and report mutations leave private captures/history and committed bytes intact. Source inspection also confirms the owned writer clones its journal and final images independently.

4. Ordinary facades, begin during finalizing, discarded transactions, poison before begin/verification, and delegated writer validation errors refuse. A ledger presented with a second genuine live owner's facade poisons that second scope and rolls its staged write back; restoring the original facade leaves the original owner's ledger usable. This deliberately substitutes only the presented facade to test identity binding, not accounting records. Nil/zero/closed, early/repeated verification, late mutation and malformed baseline/final entity cases pass the supplied tests.

5. Earlier-error precedence is explicit and tested: wrapper-local errors are `errIdentityMutationLedger`; a delegated writer validation failure poisons first with `errIdentityWriterHistory`, which remains the outer error. A preexisting owner error remains the outer error while subsequent local wrappers expose only their static sentinel. Ignoring local operation/verification failures still prevents commit and, for body failure, prevents finalization.

6. A real separate Badger writer updates an already observed/staged alias. Verification succeeds in the admission transaction's own view, then actual commit returns ErrConflict. Only the competitor's expected bytes survive. Independently injected staged observer events and both body/finalizer panics also roll back without event publication. The unchanged supplied actual oversized-value and capacity tests pass; capacity retains ErrTxnTooBig classification while arbitrary storage details are replaced by static errors. These are actual database failures, not injected stand-ins for commit.

7. Source inspection confirms begin obtains its own complete relationship inventory and then its own writer; neither is accepted as an argument. Every operation rechecks live owner/body/poison state before delegating actual Set/Delete. Verify requires finalizing, freezes the owned writer, scans final state in that same transaction, replays generated history against the exact baseline, compares complete key membership and raw bytes, and returns an owned report only after all checks. No support/authorization/undo/expected-after inputs exist.

## Demonstrated limits and remaining integration bar

The independent `DocumentedLimits` group intentionally demonstrates four behaviors that PASS because they are explicitly excluded from this finite contract:

- A raw alias creation deleted before the next observation is invisible to the ledger.
- A raw alias write performed after successful verification commits but is absent from the already returned Final report.
- A public root `ClaimAlias("phantom", "missing-owner")` during the admission transaction creates an unseen alias in another committed transaction. The admission verifier and commit both succeed, and its report omits that alias. This is a concrete counterexample to any claim that a full prefix scan alone provides production producer coordination or a serializable predicate lock; the frozen contract explicitly disclaims that claim.
- An `iga:` row lies outside this selected relationship inventory. Its body write commits and is absent from the report, as required by the separate fact/evidence accounting boundary.

These are not contract failures, but they prevent broadening this verdict into production safety. Full actual writer routing/coordination must include public ClaimAlias and all direct mutation sites; every write after this boundary needs separate exact finalizer accounting. No ownership policy, baseline-defect exception, support/dependency selection, undo/post-undo relationship closure, recognized lifecycle/adoption proof, finalizer implementation, real-store performance, recall, prevention, repair or sweep stability is certified here. Static actor attribution is not reviewed authority. Arbitrary forged private state and concurrent use remain excluded assumptions. A successful report is a snapshot/accounting result, not a promise that the surrounding transaction will commit.

## Evidence pins

Relative paths refer to `/tmp/scry-ledger-disproof-PmmAQv`.

| Artifact | SHA-256 |
| --- | --- |
| `IDENTITY_LEDGER_CONTRACT.md` | `f22e3ac0300c7408c1816d5c67ebcdc2e215aacfcc5eab1a74021972e3b6e799` |
| `internal/memory/store/identity_mutation_ledger.go` | `e43d2b90840774ea732a8b22299da5fd3b919d8722b2e7f4525d65e8401d2767` |
| `internal/memory/store/identity_mutation_ledger_test.go` | `c3714fb6dc2f2b18b706142141b60f463771ce2ddbc3124bbfc32c9f1e0d3a76` |
| `internal/memory/store/identity_ledger_independent_test.go` | `e2017062252cd8a4d06b93d218762a2fdfc015fe1505efa59225eaaa622f1455` |
| `identity-ledger-targeted.log` | `251a0327e4d96bc486ae174e59243ebab81e3a9d277255aed725af7116bcd7d9` |
| `identity-ledger-fullsuite.log` | `269175184531e679d1759e5ed48a4013dcfde361fe86a83f35d7e71955395c4c` |
| `docs/memory-repairs/relationship-inventory-independent-review-2026-09-06.md` | `46fb43bf077d65da024025f15d1a2f722cce4843c3bd7e2b9743c9c9f577a4e0` |
| `docs/memory-repairs/identity-writer-independent-review-2026-09-06.md` | `5b8a969b01917f04c9bd3c4b6096352135a8398647295fc113e7927544c1cb2b` |
| `docs/memory-repairs/identity-mutation-design-independent-review-2026-09-06.md` | `94e6e51f71e8caaa49e23e2e7c8a92de556c73099345a4b29d1c904c8219b3d3` |
| `internal/memory/store/identity_relationship_inventory.go` | `eae817f6892b5ec6c2151e4467b187fc9eee55311b1705787e773bbbcc679a2e` |
| `internal/memory/store/identity_writer_history.go` | `25022d38ec62f80a4eb1ef1759938c2e651c0f8ed864e563ce40ad092eb2d148` |
| `internal/memory/store/identity_admission_owner.go` | `14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82` |
| `internal/memory/store/identity_legacy_anchor.go` | `7a84b00d46977c2d8ce97c5b29f35763fa347bcd61f967d7178ef42a399a7de7` |
| `internal/memory/store/store.go` | `9491d689f4a41c3ea6bb8e2a622b46f04810b49daecd375a68428fd8dc6d8891` |

This report's hash is supplied separately. Root must read the complete report before retaining source or citing the verdict.
