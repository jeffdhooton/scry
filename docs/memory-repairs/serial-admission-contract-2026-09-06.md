# Private serialized admission implementation contract

Baseline4f1d1be, additive ledger integration after design baseline699da3b; all
coordination-related baseline source hashes are unchanged. Implements design
dedcbb18 with independent DESIGN review9ac136e5, both to be read in full. No new
production caller, ownership/adoption policy, schema family or live deployment.

Root ordinary update and AtomicWrite take shared graph access; the NEW private
serialized owner takes exclusive graph access. Every path orders maintenance
before graph before Badger snapshot/commit. Facades reuse their existing txn and
keep owner phase/poison semantics. Root atomic events publish after both locks
release. Existing root AtomicWrite and the old private owner remain shared and
their old public-phantom characterization must still pass unchanged.

Only fixed PutPending/DeletePending operations bypass graph locking on a root;
their facade operations delegate to update and cannot bypass rollback/freeze.
Both generic metadata setters remain coordinated, including identity_ keys.
Exclusive maintenance writers remain excluded by maintenanceMu, without taking
the graph lock. Retirement revision is sampled before and checked after both locks.

One genuine root, supported producers, nonconcurrent and unescaped facades only.
Synchronous captured-root writes inside a serialized callback, raw foreign Badger
writes/forged handles/concurrent Close remain excluded. A callback reading or
writing pq keys can conflict with root queue updates; this is not whole-DB isolation.
Queue progress is asserted only absent pending exclusive maintenance/resource
failure, and specifically for memory.remember, not metadata-stamping enqueue RPC.
No live p95 or globally ordered observer/index publication claim. Arbitrary input
errors and committed observer panic retain existing behavior; no global error
privacy rewrite is claimed. Nil root/callback and nested entry refuse as before;
zero/forged root handles are not genuine Open-produced handles.

Testing: all real valid public producer calls block during body AND finalizing,
release after commit/refusal, and an existing public transaction blocks entry.
Actual ClaimAlias phantom excluded by new entry, not removed from old harness.
Queue put/delete/read/count progress, durability on reopen, and facade freeze /
poison / rollback. Actual memory.remember tested with private build-tagged bridge
calling real private entry on same root; bridge/test remain private and absent
from integrated/default production builds. All errors/panics/real commit failure
release locks, suppress failed events, preserve exact bytes; postcommit observer
can call root writer. Ordinary nested writes and retirement waiting unchanged.

Maintenance forward tests exercise valid real operations. Reverse tests use actual
postcondition/backup/reader barriers where available. Adoption reverse exclusion
may be compositional: exact shared maintenance lock gate plus audited acquisition
and actual forward adoption exclusion. Do not describe that as actual paused
reverse adoption. Full unchanged no-CGO suite and fresh code disproof required.
This contract never authorizes alias transfer, missing-owner claims, finalizer
untracked writes, intentional assertion loss, live adoption or broad cleanup.
