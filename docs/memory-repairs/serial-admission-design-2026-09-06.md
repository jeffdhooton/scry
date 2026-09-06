# Narrow graph-writer coordination — private design

Baseline 699da3b. This is a concurrency boundary for a future fixed admission
policy, not that policy, a schema change, deployment approval or an ownership rule.
The reviewed uncalled owner harness and ledger remain independently bounded.

## Proven gap and queue constraint

The identity ledger's independent report 618182c8 demonstrates public root
ClaimAlias creating an unseen al: key while the old admission snapshot verifies
and commits. Badger prefix scans are not serializable predicate locks. Coordinate
all cooperating public graph producers around the whole admission transaction.

Do NOT hold maintenanceMu exclusively during admission: PutPending uses its shared
side, and memory.remember calls HasEpisode, HasPending, PutPending, PendingCounts.
Long graph scans must not add a graph-lock dependency to that durable queue path.
Existing exclusive backup/repair/restore maintenance can already delay enqueue;
this proposal does not promise enqueue progress while such maintenance is pending.
It also does not prove live p95, Badger resource independence or provider health.

## Proposed implementation

Add root Store.graphWriteMu sync.RWMutex. Fixed order: maintenanceMu before
graphWriteMu, then Badger transaction. Never upgrade or reacquire the root locks
through a transaction facade. Facades continue using the existing owned txn.

1. Existing root AtomicWrite takes maintenance shared then graph shared through
   snapshot creation, callback and actual commit. Preserve retirementRevision
   sampling BEFORE waiting and checking AFTER both locks. Publish staged events
   only after both locks release, retaining rollback, panic and earlier-error rules.
   Existing nested facade behavior is unchanged.
2. Central root Store.update takes graph shared around db.Update; its existing
   callers already hold maintenance shared. Facade update does not acquire the
   root graph lock and keeps current admission freeze/poison behavior.
3. Only PutPending and DeletePending use a narrowly named private queue-update
   path: on a facade delegate to ordinary update so existing ownership guards
   cannot be bypassed; on a root use db.Update without graph lock while the public
   queue method holds maintenance shared. Their fixed pq: key construction cannot
   target graph families. No caller-selectable public bypass or arbitrary key API.
4. A NEW PRIVATE runSerializedIdentityAdmission entry validates root/non-nil
   callbacks before acquiring locks, then runs the same owner phase protocol as
   the existing harness with maintenance shared and graph EXCLUSIVE held through
   commit. The old uncalled runIdentityAdmission remains a shared-lock harness;
   its existing negative characterization tests remain unchanged. Refactor common
   root transaction/owner mechanics if useful, without exposing a caller-provided
   finalizer publicly or claiming the private callback is a fixed policy.
5. Ordinary root update covers PutEpisode, PutEntity, PutFact, InvalidateFact,
   DeleteFact, PutCursor, DeleteEntity, ClaimAlias, DropAlias/Rehome, RelocateFact,
   AttestAlias, value evidence and BOTH generic metadata setters (including any
   identity_ spelling). Root AtomicWrite covers private raw codecs invoked on its
   facade and arbitrary supported callback composition. An AtomicWrite callback
   doing only queue work is conservatively a graph producer and may wait; the
   daemon's ordinary remember path does not use AtomicWrite.
6. MergeEntitiesChecked, reviewed alias repair, retirement, adoption and Restore
   already hold maintenance EXCLUSIVE across their raw transactions; retain that
   outer exclusion, do not add inverted/reentrant graph locking. ensureSchema is
   initialization before the handle is published. Backup is read-only. Root Close
   is handle lifecycle, remains outside the concurrent-use contract. Audit every
   non-test direct db.Update/txn.Set/Delete/DropAll/Load and every update caller;
   do not rely on this enumerated list without checking the actual source.

## Limits and follow-on requirements

Cooperation is scoped to ONE genuine root Store handle and its supported methods,
with callbacks using only their facade. A captured root mutator called synchronously
from a serialized callback would deadlock and is forbidden by this private contract;
it is not a new resolver capability. Concurrent facade use, forged Store sharing a
db pointer, direct raw Badger writers, concurrent Close and out-of-package unsafe
access remain excluded. Badger's normal directory lock excludes another genuine
Open on the same database; do not disable it. Raw external writers can still cause
phantoms. No claim of universal predicate locking.

Serialization prevents interleaving only. A public ClaimAlias after the scope can
still claim a missing owner under current policy; a full fixed lifecycle policy
must reject that before production admission/adoption. Post-verification finalizer
writes remain a separate exact accounting responsibility. Prefix families outside
the identity relationship inventory need their own evidence/support validation.
Generic metadata coordination does not authorize replacing identity markers.

Observer callbacks run after commit without graph lock. No global cross-writer
event/index publication ordering is claimed; reentrant observers must not deadlock
because of the new graph lock. Existing maintenance/observer behavior is not silently
redesigned here. Failed transactions publish no events. Successful observer panic
does not retroactively roll back committed bytes.

## Required tests before any integration

- Deterministic channels show EVERY covered public producer blocks through both
  body and finalizing, then executes only after commit or failure. Their writes
  cannot create an unseen alias in the serialized snapshot. Test entry both ways:
  an already-running public graph transaction excludes serialized entry too.
- PutPending, DeletePending, HasPending, PendingCounts and the actual synthetic
  memory.remember handler complete while serialized body is held open. Assert
  durable pq bytes on reopen, not merely elapsed time or callback entry. This is
  queue progress under synthetic controlled scheduling, not live p95 evidence.
- Real public ClaimAlias phantom counterexample against the OLD harness still
  passes unchanged; equivalent NEW harness scenario is blocked, then observed
  after serialized commit. No rewritten assertion disguises legacy behavior.
- Root maintenance operations exclude serialized entry in both directions, using
  synthetic fixtures and their real preflight paths. Retirement waiting retains
  refusal. All errors, body/finalizer panics, malformed/nested entry and Badger
  commit failures release locks; pending and graph writes subsequently progress.
- Zero events on failed transactions; successful event callbacks can perform a
  root graph write after commit without new graph-lock deadlock. Exact raw store
  preservation/rollback for failed scopes; no source policy semantics changed.
- Full noncached CGO_ENABLED=0 suite, unchanged prior tests, then fresh-context
  disproof against exact source and this complete contract. No live mutation.

Please disprove this design before implementation, especially missing producers,
lock ordering, queue coupling, facade bypasses, and claims stronger than one-root
cooperative serialization. A design GO is not a code or full-goal verdict.
