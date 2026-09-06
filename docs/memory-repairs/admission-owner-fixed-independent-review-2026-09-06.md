# Corrected private admission owner: bounded independent review

2026-09-06. **PASS for the exact private outermost transaction-owner harness,
subject to its explicit bounded contract.** The earlier public-maintenance
counterexample is fixed. This is not approval for production controller integration,
support cleanup, observation policy, generation adoption, deployment or any
whole-goal completion claim.

The rejected export and report remain unchanged at
/tmp/scry-owner-independent-sep06.lWNWgh. That rejection attaches to its original
hashes. This correction was independently exported from
87a6d1a262d5d7c62037d66335b9a020f89fd4a6 into the new directory
/tmp/scry-owner-corrected-independent-sep06.Cm5GpS. All five corrected implementation
files and original candidate tests were copied with apply_patch and checked against
the supplied pins. The original independent ordinary-boundary tests and required
maintenance safety regression were copied byte-for-byte from the rejected export;
their hashes are unchanged. The characterization test whose PASS meant reproducing
unsafe behavior was deliberately retained only in the rejected export.

I previously completed session orientation and read its output, the complete active
goal and complete archived Controller V2 review. For this correction I read the
complete amended owner contract, complete owner implementation, added builder test,
and complete correction diff, and scanned all public Store methods and direct DB
uses. No shared/frozen implementation was edited, and no live graph, providers,
sweeps, configuration or deployment was accessed. All reviewed mutations use new
synthetic temporary databases.

## The demonstrated correction

The new refuseAdmissionMaintenance guard refuses any admission-owned facade in
every phase and preserves the first recorded failure. Seven entry boundaries call
it before taking locks, opening another transaction, calling I/O or altering the DB
handle: MergeEntitiesChecked, RetireEntitiesChecked, BackupAndRetireEntities,
BackupAndRepairAliases, Backup, Restore and Close. Public wrappers delegate through
these guarded entries. Static inspection found no remaining public direct-DB write
entry outside these boundaries and ordinary update/AtomicWrite. The ordinary root
Store has no admission owner and therefore retains its existing behavior.

The exact previously failing TestOwnerMaintenanceRequiredSafety now passes both
unchanged cases: the escaped closed facade refuses a valid reviewed merge, and a
finalizer cannot commit that merge before reporting an outer error. Complete raw
key/value maps remain identical. Thus this result fixes the prior counterexample
without narrowing the post-return facade claim.

## Independent expanded tests

owner_corrected_maintenance_independent_test.go constructs real ready previews and
exact reviewed fingerprints for a synthetic merge, retirement with explicit fact
replacement, and alias drop. It also makes a nonempty valid Badger backup. These
inputs are used for all ten public maintenance entrypoints, covering every wrapper:
MergeEntities, MergeEntitiesChecked, RetireEntity, RetireEntityChecked,
RetireEntitiesChecked, BackupAndRetireEntities, BackupAndRepairAliases, Backup,
Restore and Close.

Forty independent refusal cases cover all ten operations in the body, finalizer,
closed-after-success and closed-after-failure phases. Body and finalizer cases
stage an ordinary episode first and deliberately ignore the maintenance refusal.
The owner still refuses commit, preserving every original raw byte. Tests assert
zero observer events, zero backup Write/Sync/Close calls, zero Restore reader calls,
zero maintenance postcondition calls, and correct finalizer invocation count. They
verify the root remains open and unchanged even after refused Restore or Close,
and the escaped ordinary write/callback methods remain refused.

Ten independent positive controls then execute the identical valid operation
inputs through root Stores. Merge, retirement and alias repair actually change
their synthetic graph; backups contain data and obey their durable writer protocol;
Restore reads the valid backup and removes a deliberately added post-backup episode;
Close closes the DB. Checked maintenance callbacks run once. These controls make
the refusal result distinguishable from invalid manifests or disabled functionality.

The unchanged independent ordinary-boundary suite still proves shared nested
facades, exactly one check after the complete body, body/finalizer panic rollback,
caught nested errors and panics poisoning the owner, recursive entry refusal,
ordinary-parent poisoning after refused admission, finalizer ordinary mutator
freeze, expired facade refusal, and unchanged non-admission caught-error behavior.
It compares complete raw maps including opaque unknown bytes and checks zero
events on failure. The unchanged tests still reach a real Badger oversized-key
staging failure and an actual badger.ErrConflict during commit. The latter uses a
synthetic competitor solely for fault injection and verifies only that competitor's
expected bytes survive.

The builder's 30 maintenance boundary cases and four original harness tests also
pass unchanged. The unchanged ordinary AtomicWrite regression passes.

## Test-development correction and exact commands

The first expanded run used an overstrict new assertion requiring the scope
sentinel after a facade already held a different prior failure. It failed the ten
closed-after-failure cases because the implementation correctly preserved the first
error. The amended contract requires refusal/poisoning, not replacement of a prior
error. I corrected only this newly authored grader assertion to require that exact
prior error in those cases. No implementation or reused safety test was changed.
The initial output is retained as corrected-targeted.log.

From this export, with shell pipefail enabled:

```
CGO_ENABLED=0 go test ./internal/memory/store -run 'TestOwner|TestAdmission|TestAtomicWrite' -count=1 -v
CGO_ENABLED=0 go test ./... -count=1
```

The final targeted command passes (exit 0); exact output is
corrected-targeted-final.log. The complete noncached no-CGO suite also passes
(exit 0), including all independent and unchanged original tests; exact output is
corrected-fullsuite.log.

## Scope limits and architectural consequences

The injected finalizer remains a private harness seam. No caller has a production
capability to select or omit policy checks, and no complete store-owned support or
identity policy is implemented. Guarding these public maintenance entries does not
authorize private raw helpers in the finalizer. Separately captured root-store
writes, arbitrary raw transaction misuse, concurrent facade use, public validation
errors occurring before update, and panics recovered wholly inside callback code
remain explicit exclusions; this report makes no claims for them.

Read-only maintenance previews still use independent DB views in the existing
code. They are not a validated transaction-local support oracle. Future finalizer
integration must use a reviewed raw support/identity policy and resolve all existing
controller/adoption/observation obligations. Nothing here approves destructive
finalization, mixed-episode disposition or complete facade cache disposal.

## Exact pins

SHA-256, paths relative to this export unless otherwise specified:

| Artifact | SHA-256 |
| --- | --- |
| Amended frozen OWNER_CONTRACT.md | 141ecaa7c5d1b91fd60bbd69eb1f1ae1077b33b148d561cd7f6941dd6758ce0f |
| internal/memory/store/store.go | 9491d689f4a41c3ea6bb8e2a622b46f04810b49daecd375a68428fd8dc6d8891 |
| internal/memory/store/identity_admission_owner.go | 14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82 |
| internal/memory/store/merge.go | b9e2e4073e93941679e5f04cd8486b78d7ec3c897fff801962a99a627bd9649a |
| internal/memory/store/retire.go | 02c575aa58e7870965d2f5bf9ef2f6d381dcdcd3cfcd68799342db2ef135f0a5 |
| internal/memory/store/unalias.go | 0825ae4f1edbe104989cc5f48c93b7f585f82b73853753ccbf041b3fd8049adc |
| internal/memory/store/identity_admission_owner_test.go | 903298415d0d4a2e65fcbf6864d89fd91f0118d58c73447f33aed6ef3341d449 |
| internal/memory/store/identity_admission_maintenance_test.go | 4947cc48a568ed1d5afc25c591f5965311229dcd8e11c57beadac46222870c97 |
| internal/memory/store/owner_independent_disproof_test.go | 71e2acf902d5c6f16f3941a4135275b222271056f797d369b0edc9c7a00d0d31 |
| internal/memory/store/owner_maintenance_required_safety_test.go | c8d442b59912e49ef76e279332bb882f80f0f3fae8ff9f0f3e910118226f695c |
| internal/memory/store/owner_corrected_maintenance_independent_test.go | 76a6bef8a6ebaa063f9744f63de17abbd6c0628e34613531209c5a0ddfcc8d50 |
| corrected-targeted-final.log | b6375fc4f76f37fcdd9c14106435ec7369eb1c7ac972b08e587920a17a37630d |
| corrected-fullsuite.log | 5a5b0fbc70cbe5e30680e9ca61c8973442584399dbe3f2ea0256862c0d78d8a6 |
