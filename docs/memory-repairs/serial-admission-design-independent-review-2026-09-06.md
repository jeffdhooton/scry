# Narrow graph-writer coordination — independent DESIGN disproof

2026-09-06. **GO to implement the pinned narrow design. No design-contract violation was proved. This is not a code, integration, production routing, deployment, live latency, or whole-goal grade.** The new coordinator does not exist in the baseline inspected or tested here. Integration remains NO-GO until the promised implementation evidence and fresh code disproof exist.

I read the complete active objective, the complete design at `/tmp/scry-serial-admission-sep06.wDNcu0/SERIAL_ADMISSION_DESIGN.md`, and the complete independent identity ledger report at `/tmp/scry-ledger-disproof-PmmAQv/IDENTITY_LEDGER_INDEPENDENT_REVIEW.md`. I read its actual public-phantom test. Session orientation preceded other work. I independently exported commit `699da3bd9cc271e266f7c53b5f5ad78e7f124fa0` into `/tmp/scry-serial-design-disproof-ZoX9By`, then authored only a synthetic characterization test and this report there using apply_patch. Formatting used gofmt. Test commands generated the retained logs. No candidate/shared source edits, real stores/replicas, provider calls, live queue writes, backups of real data, deployment, room, memory, or configuration writes were performed. No agents were spawned.

## Findings

1. **Producer inventory supports the proposed exclusion.** I searched every non-test store source for `Update`, `NewTransaction`, `NewWriteBatch`, `Set`, `Delete`, `DropAll`, `DropPrefix`, `Load`, `Flatten`, and `RunValueLogGC`, then inspected the actual mutation bodies and helper callers. Root `update` covers PutEpisode, PutEntity, PutFact/putFactUnlocked, InvalidateFact, DeleteFact, PutCursor, DeleteEntity, ClaimAlias, DropAlias/DropAliasRehome, RelocateFact, RecordValueEvidence, AttestAlias, PutMetaTime and PutMetaJSON. All production root callers currently acquire maintenance shared first. PutPending and DeletePending are the only proposed exceptions; literal `pq:` prefix concatenation cannot become an `al:`, `en:`, fact, or metadata key through a malicious ID. Empty/no-op and input-validation branches can return before the graph lock because they do not produce writes; blocking tests must use valid requests that actually mutate.

2. **Direct mutation sites have an identified enclosing boundary.** MergeEntitiesChecked, retirement variants, BackupAndRepairAliases, and private adoptLegacyInventory acquire maintenance exclusive before their raw transactions. Their private unlocked helpers have only the audited enclosing callers. Restore holds maintenance exclusive across DropAll, Load, and ensureSchema. Initialization's other ensureSchema caller runs before Open publishes the handle. `deleteAliasIfOwnedBy` runs within update transactions; `writeAliasRejectionsTxn` runs within merge/alias-repair maintenance. The private generation, observation, input-revision, outcome, identity journal, identity writer, and fact-ledger writes use a live facade transaction. Root AtomicWrite's proposed shared graph lock covers them for the entire transaction, regardless of whether a raw private codec calls update. Serialization does not validate these codecs' policy or account for their post-verification changes.

3. **Lock placement is sufficient within the stated one-root contract.** Acquiring maintenance shared, then graph exclusive, before creating the admission snapshot excludes existing and subsequent root update/AtomicWrite transactions through their actual commit. Maintenance-exclusive writers are excluded by the outer lock without reacquiring graph. Facade methods must continue to use their own transaction and never acquire the root graph lock. Retirement revision must be sampled before waiting and checked after both locks, including the new serialized entry if transaction mechanics are refactored. Taking graph exclusive and then calling root AtomicWrite would deadlock; extracting common mechanics is acceptable only if lock ownership remains explicit. Captured-root mutation during a serialized callback, raw foreign transactions, forged handles, escaped/concurrent facades, and concurrent Close remain excluded; the proposal does not promise predicate locking for those cases.

4. **The actual remember path supports the queue claim, with precise limits.** `handleMemoryRemember` obtains the daemon's single lazy Store, redacts text, derives the episode ID, calls enqueueEpisode (`HasEpisode`, `HasPending`, `PutPending`), wakes the worker, and calls PendingCounts. Worker Kick is a nonblocking channel send. Neither this handler nor enqueueEpisode wraps the request in AtomicWrite or writes metadata. The reads use Store.view without the proposed graph lock. Root pending writes can therefore avoid a new direct graph-lock dependency. By contrast, `handleMemoryEnqueue` writes MetaLastIngest after inserting queue items: that RPC may wait for graph access even after its queue bytes commit. Do not generalize this verdict to all ingestion RPC latency. Queue scanning cost, Badger disk/resource pressure, daemon mutex contention, and provider/extraction health remain unmeasured.

5. **Maintenance can still indirectly delay queue admission.** A pending maintenance-exclusive lock can block later maintenance readers, including PutPending/DeletePending, even while serialized admission holds maintenance shared. The frozen design expressly excludes queue-progress guarantees when maintenance is pending. No conflict with the narrow promise was found. A serialized body that reads or writes pq keys can also experience ordinary Badger conflict with concurrent root queue updates; graph serialization is not whole-database isolation or a guarantee that every admission commits.

6. **Facade and observer requirements are necessary, not optional polish.** A queue helper must delegate facade operations to ordinary update so owner finalizing/closed/poison refusal and transaction rollback remain intact. Merely testing a root pending write does not establish this. Root atomic events must publish only after maintenance and graph locks release. Existing direct mutators notify after update returns while still holding their existing maintenance shared lock; the new update lock must already be released. Existing maintenance/observer reentrancy limitations are not repaired by this design. No global event ordering is promised. A successful observer panic leaves committed bytes; failed scopes emit no events.

## Independently executed baseline evidence

All database fixtures are newly created temporary synthetic stores. The export contains no implementation of graphWriteMu or runSerializedIdentityAdmission.

Commands used `CGO_ENABLED=0`, `-count=1`, verbose output, and shell pipefail:

```
go test ./internal/memory/store -run 'TestSerialDesignBaseline|TestAdmissionOwner|TestAdmissionMaintenance|TestOwnerIndependent|TestPending' -count=1 -v
go test ./internal/daemon -run 'TestMemoryRemember|TestMemoryEnqueueDedupesAgainstStoreAndQueue' -count=1 -v
```

The corrected store run passed, exit 0, 2.088s, including the unchanged owner, maintenance refusal, pending, and independent owner failure/conflict tests. Three authored baseline groups demonstrate: (a) public ClaimAlias creates a missing-owner alias absent from the owner transaction's full scans, while both the owner staged episode and competitor alias commit; (b) root queue insert/delete/count/read complete during a held old owner scope and survive that scope's intentional rollback and orderly close/reopen; (c) facade queue put/delete during finalization poison the owner and preserve the exact complete raw map. These characterize existing behavior; they do not test the proposed coordinator.

The initial store command failed one newly authored characterization because its finalizer called exact ResolveAlias on the competitor's key, registering a point read and correctly causing Badger ErrConflict at commit. All other selected tests passed in that run. I corrected that new test to inspect the full scanned raw map in finalizing, matching the prefix-scan phantom claim; the initial failing log is retained, and no supplied test or implementation was changed. This distinction matters: point reads can detect this specific competitor while prefix scanning alone does not establish the needed producer boundary. The initial authored source hash was `8108ec5276b32a70d6a979d8bc6a02ecbbfdcfcc92b0f2bd55f107670059eb0b`; its only subsequent edit replaced the finalizer's exact ResolveAlias with scanned-map membership and nil return.

The actual daemon baseline run passed, exit 0, 0.884s, including dormant queueing, return before fake extraction, same-day deduplication, redaction, and enqueue deduplication. Its configured-provider log label did not invoke a provider: the active worker test installs fakeExtractor and the other selected tests start no extraction worker. These are baseline handler tests, not tests under the new lock. I did not run a full suite for this design-only review.

## Concrete implementation acceptance conditions

- Preserve the complete design limitations, all ordinary ownership behavior, the old owner harness, and its unchanged public-phantom negative characterization. The new entry must remain private and uncalled by production until the fixed lifecycle policy receives separate review.
- Show each audited public mutator's valid write path blocks during both body and finalizing and executes only after success/failure releases the scope. Show a preexisting public transaction prevents serialized snapshot entry. Include both metadata setters and identity-looking metadata names. No graph writer may create an unseen alias inside the serialized snapshot window.
- Prove root PutPending/DeletePending and actual memory.remember progress under controlled scheduling, in both body and finalizing, with durable reopened pq bytes. Prove facade queue methods still poison/refuse late operations and roll back with their owner. Root AtomicWrite containing only queue work remains conservatively coordinated.
- Retain all error precedence, ordinary nested AtomicWrite behavior, owner phase rules, malformed/nested entry refusal, body/finalizer panic rollback, genuine Badger failure behavior, zero failed-transaction events, and postcommit observer root-write progress. Assert subsequent queue and graph writes progress after every failure class.
- Demonstrate actual maintenance operations exclude admission and vice versa using valid preflight fixtures, and retirement waiting still refuses. For adoption, which offers no pause callback, actual forward exclusion plus a reverse test against the exact maintenance lock and source proof of adoption's acquisition can be accepted as compositional evidence. Label it as such; do not call it an actual adoption paused in reverse direction. If this test approach is chosen, narrow the literal test wording that currently requests real preflight paths in both directions.
- A private build-tagged synthetic bridge is an acceptable mechanism for daemon-package tests to call the real private admission entry on the same root handle. Keep it exclusively in the private test export, pin its contents, confirm it is absent from integrated/default production files, and exercise the real handler. It must not substitute a fake lock or introduce a production admission/finalizer API.
- Run the full noncached no-CGO suite on exact candidate source, retain unchanged prior tests, and obtain fresh-context code disproof. No full-goal, live p95, maintenance isolation, fixed-policy safety, or deployment verdict follows from this design GO.

## Evidence pins

Paths below are relative to this private export except the first two documents and prior ledger test.

| Artifact | SHA-256 |
| --- | --- |
| Frozen SERIAL_ADMISSION_DESIGN.md | dedcbb18e986639509e8a64284b75cc6580b382868012139684af326e7908106 |
| Complete IDENTITY_LEDGER_INDEPENDENT_REVIEW.md | 618182c8444ea1c794034d4a4af743cafa8390387878c8bcca2ca8e90609b2f5 |
| Prior identity_ledger_independent_test.go | e2017062252cd8a4d06b93d218762a2fdfc015fe1505efa59225eaaa622f1455 |
| internal/memory/store/store.go | 9491d689f4a41c3ea6bb8e2a622b46f04810b49daecd375a68428fd8dc6d8891 |
| internal/memory/store/pending.go | 718ec2e5d1929bf6448961ed406bbfe8b6d585861204c067cccef5e86aa6650f |
| internal/memory/store/value_evidence.go | 7a9d822b581f8adeadad9af6e4cc40a612436e62d2a963e4b2e3b6b00cbf32e1 |
| internal/memory/store/merge.go | b9e2e4073e93941679e5f04cd8486b78d7ec3c897fff801962a99a627bd9649a |
| internal/memory/store/retire.go | 02c575aa58e7870965d2f5bf9ef2f6d381dcdcd3cfcd68799342db2ef135f0a5 |
| internal/memory/store/unalias.go | 0825ae4f1edbe104989cc5f48c93b7f585f82b73853753ccbf041b3fd8049adc |
| internal/memory/store/alias_rejection.go | 297d02a25663597bcfb692e0d928cca224b5ec38b918682558535ed11a2cee78 |
| internal/memory/store/identity_legacy_adoption.go | 9ac1ea5299c001e9454c49a54dad0dcb18ed54f04ab9922f04dd02bd7874c3d7 |
| internal/memory/store/identity_admission_owner.go | 14821246232b3210476778b12bd3366957b35ca296ac2b4e13a1f388d20cff82 |
| internal/memory/store/identity_admission_owner_test.go | 903298415d0d4a2e65fcbf6864d89fd91f0118d58c73447f33aed6ef3341d449 |
| internal/memory/store/identity_admission_maintenance_test.go | 4947cc48a568ed1d5afc25c591f5965311229dcd8e11c57beadac46222870c97 |
| internal/memory/store/owner_independent_disproof_test.go | 71e2acf902d5c6f16f3941a4135275b222271056f797d369b0edc9c7a00d0d31 |
| internal/memory/store/pending_test.go | 6cb1f67a5eed5d57fb388e5c3ff2a760a2cb7d95fb7cc8a6de2d3b04827cdbd2 |
| internal/memory/store/serial_design_characterization_test.go | cc790489ca33c30fdaf7309557fb545492b8dac811d3ebc347a96c90092b97db |
| internal/daemon/memory_methods.go | d200141a6dae07b86e30fee0a3057b5ef7bf8f6e40c270f61b3fe97d7cff77da |
| internal/daemon/memory_queue.go | b1a5f2294da3baf5286ebd11746dc7115b43ae8e6b1e6bda764b7c832029d13e |
| internal/daemon/daemon.go | 05ad387ca54775cf933baf2023a845e4bb3dc9f48b49d5cdeb9bdea8e42bfdc5 |
| internal/daemon/memory_methods_test.go | ad67c8e7aab63df68bcd18590c4fe188f0b10f6328e32514b4aeab9482cc54fb |
| internal/memory/store/identity_generation.go | e07c6c50affce548b31a94dc2e94d665a49ccab1d246f9433f49277535619592 |
| internal/memory/store/identity_outcome.go | 7b40e318a2c323efed48a5d40c79853fd61e1455e24265903300e7f3cc7c59e8 |
| internal/memory/store/identity_fact_ledger.go | 2040e19a3e0bcf505b6faf8253551e687f6d4dc2059823aeeb744536cb8697c7 |
| internal/memory/store/identity_journal.go | d961d53c2c5fcc69b110987788c01837e79534ac381548c655fe463e020e3a80 |
| internal/memory/store/identity_input_revision.go | b69e6daf91e98fba34166fe949c374d65f9196831d49ddcd11b9650364e59908 |
| internal/memory/store/identity_writer_history.go | 25022d38ec62f80a4eb1ef1759938c2e651c0f8ed864e563ce40ad092eb2d148 |
| internal/memory/store/identity_observation.go | 83a39c03435185963a4a2931f94f30f03a7bd1a86c6e4c545ece95580ad9e0ab |
| serial-design-characterization.log (initial failed characterization) | 15d10ae45d0be76db202ec91c4d665e4f02ea739681fd8e92566026f24ce986a |
| serial-design-characterization-corrected.log (passing) | e9c0736ff86ce6871ffe16cde791b4ebdd44abe89f48c3709c7c821a618f527f |
| serial-design-daemon-baseline.log (passing) | 907682ec93a551d868a81e69e7f43a97b1a62eb626b1889c01462575f55a57cf |

This report's hash is supplied separately. Root must read the complete report before citing its verdict.
