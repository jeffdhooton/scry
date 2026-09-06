# Immutable outcome codec — independent correction review

2026-09-06. **PASS for the bounded private, uncalled codec at source `7b40e318`; the reproduced staging-error disclosure is closed.** Both original failed safety tests pass unchanged, additional positive controls pass, and the full noncached CGO-disabled suite passes. This is one bounded correction review, not a second fresh-context goal-grading round or approval for controller integration/deployment.

## Isolation and unchanged failure evidence

I exported baseline `fd464c22fd8db22d31edce8bf09457ec2d10f726` into the new private directory `/tmp/scry-outcome-correction-review-9cXTnD`. Candidate files and both original independent test files were copied using apply_patch, then SHA-256 checked. Only the new positive-control test, this report and captured output were authored in the new export. No shared/root/candidate source changes, live store access, provider calls, sweeps, config changes, memory/room writes or deployments occurred.

The full original objective, archived V3 review and contract were read for the preceding review and their requirements remain the scope. The original failing export `/tmp/scry-outcome-review-fV5kP0` is preserved, including source `fc7f928b`, report hash `8b75b94c08c8648e73de3186353ee46ebd0e22b7d00e782a07a36f53462c3d11`, complete failed log hash `6b33295f096ecabaa1402bf543dc12baaca2b1dd27f1d7a68aa5ec8eab272ed4`, and both unchanged failing-test files. Their hashes match the copies in this review exactly. No failure was erased or test weakened.

The exact implementation diff contains only the Set error boundary. Raw storage errors are no longer returned. `errors.Is(err, badger.ErrTxnTooBig)` selects the two static sentinels `errors.Join(errIdentityOutcome, badger.ErrTxnTooBig)`; every other Set error becomes static `errIdentityOutcome`. The raw original error is neither returned nor wrapped. Other error routes remain as inspected in the original review: codec, provenance and reader errors return static reasons.

## Executed disproof and positive controls

Both unchanged privacy tests now pass against real storage refusals. The memory-backed case still submits the valid 35,853-byte outcome above its 4,096-byte value threshold. The disk-backed case still submits the valid 1,561,189-byte outcome above its 1,048,576-byte value-log limit. These are the same synthetic inputs and actual Set paths that disclosed bytes in the initial candidate. Snapshot equality and zero graph events continue to pass.

New independent `TestIndependentOutcomeCorrectionStaticErrorsAndLargeSuccess` adds four controls:

- Real memory and disk value-limit refusals return an empty key and exactly `errIdentityOutcome.Error()`, satisfy errors.Is for that static sentinel, and are not mislabeled as ErrTxnTooBig. Their errors contain no nonallowlisted details. Every raw key/value stays unchanged and observer events remain zero.
- A real transaction-size refusal uses a 2 MB memtable, 200,000-byte value threshold, 190,000-byte staged filler and 120,000-byte materialization. It reaches outcome staging, keeps both static sentinel classifications and returns exactly their joined static string. The prior filler and outcome roll back together. This independently preserves the copied builder's actual-ErrTxnTooBig regression.
- Each failure case permits a subsequent independent valid outcome write and bounded detail read, showing that sanitization does not break later transactions.
- A disk database with a 4 MB value-log limit accepts a valid outcome larger than 1 MB, with the same materialization size used by the new disk refusal control. Replay returns the identical key. Every complete chunk JSON envelope remains at or below 24,576 bytes, offsets progress without skipping, and reassembled canonical bytes exactly match the precomputed input.

All original bounded preservation checks pass again: birth/assertion separation, primary and hint-only no-birth cases, two births referencing one assertion without graph writes, exact original Supersedes spelling, observation and actual episode validation, wrong links/roles/ordinals, unknown/duplicate/lossy encoding refusal, caller buffer ownership, nil/empty distinction, immutable revisions/predecessors/branches, complete key pagination and byte reassembly, callback error/panic/staging rollback, backup/restore and close/reopen equality, and absence of graph events or routing authority.

## Commands and result

In the new private export:

`CGO_ENABLED=0 go test ./internal/memory/store -run 'Test(IndependentOutcome|Outcome)' -count=1 -v`

PASS, store 3.684s. Seven independent top-level tests (fourteen corruption subcases and four new positive-control subcases), plus the six copied builder tests, pass.

`CGO_ENABLED=0 go test ./... -count=1`

PASS, including store 27.856s and daemon 28.786s. Complete output is `INDEPENDENT_FULL_TEST.log`. No tests used cached results.

## Limits and verdict

No additional contract violation was proved. The correction closes the exact disclosure demonstrated in the initial review while preserving safe classification, transaction atomicity, oversized successful storage and bounded exact reads. PASS applies only to the pinned private uncalled codec.

Materialization, resolved annotations and generation IDs remain descriptive. This review does not certify support, current identity ownership, actual live entity staging, dependency closure, whole-episode Force projection, current-head/branch selection, legacy adoption, controller integration, cleanup, recall performance, two live sweeps or any complete active-goal clause. Those exclusions are part of the reviewed contract, not newly added limitations.

## Pins

| Artifact | SHA-256 |
| --- | --- |
| OUTCOME_CONTRACT.md, unchanged | `e5bd83f2c2e30355fe8fa1642420305c51299ccb40d847853ed3ea619a0f7ab2` |
| internal/memory/store/identity_outcome.go, corrected | `7b40e318a2c323efed48a5d40c79853fd61e1455e24265903300e7f3cc7c59e8` |
| internal/memory/store/identity_outcome_test.go, unchanged | `25407a0dedf43919936b27c99e4709fd4ab2dec6dc745007b783e917317df419` |
| internal/memory/store/identity_outcome_independent_test.go, unchanged | `4d122a3de3a72e7032492d30f15c5e70330c41f47501873554f1cfd6e0884a4f` |
| internal/memory/store/identity_outcome_disk_disproof_test.go, unchanged | `2528870d02209a6eb2c032fc820a6137c42afaa061a38c2e5ae11fd9ae72cf2a` |
| internal/memory/store/identity_outcome_sanitization_independent_test.go, new | `61a3f3e969522d031bd1b698b8de25d214acde59c3200790b923a9368acc85c9` |

Relative paths refer to `/tmp/scry-outcome-correction-review-9cXTnD`. Final report and log hashes are supplied separately.
