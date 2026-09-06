# Private serialized admission — independent CODE disproof

2026-09-06. **GO for the pinned private, uncalled cooperative coordinator under the complete frozen contract. No violation of that bounded contract was proved.** This is not a production admission, ownership/adoption, finalizer policy, live latency, deployment, cleanup, or whole-goal verdict. The private daemon bridge and its tagged test must never integrate or ship.

I ran session orientation before other work and read its output, then read the complete active goal objective, SERIAL_ADMISSION_CONTRACT.md, SERIAL_ADMISSION_DESIGN.md, complete independent design review, and complete archived identity ledger review. I independently exported baseline `4f1d1beb758a8c381a8b5184b790db8dd651ca6c` into `/tmp/scry-serial-code-disproof-xzOvG2` using git archive. I copied only the pinned three modified source files, two supplied store test files, two documents, and private tagged bridge/daemon test using apply_patch; all supplied pins match. I authored one additional test file and this report with apply_patch, formatting only my test using gofmt. A complete git-object comparison verified every baseline file outside the three intended implementation files unchanged, including every original test.

No shared/candidate edits, real store or restored live replica access, providers, live queue writes, room/remember operations, sweeps, configuration changes, real backups, or deployments occurred. All database fixtures are synthetic temporary stores. The actual remember test calls the real handler on its temporary daemon's exact Store, without starting the daemon or a worker. Its extraction-chain environment log announces configuration; the fixture explicitly forces the extractor nil and makes no provider call. The lead's additional replica probe was neither copied nor executed here. No agents were spawned.

## Executed evidence

Successful commands ran with `CGO_ENABLED=0`, `-count=1`, shell pipefail, and complete output retained in logs:

```
go test ./internal/memory/store -run 'TestSerialCodeIndependent|TestSerializedAdmission|TestILIndependentDocumentedLimits' -count=1 -v
go test ./... -count=1
go test -tags scry_private_serial_probe ./internal/daemon -run TestPrivateRealRememberDuringSerializedAdmission -count=1 -v
```

The corrected targeted run passed, exit 0, store 7.145s: six additional independent groups, ten unchanged supplied serialized groups, and the unchanged prior ledger's documented-limits group. The full noncached no-CGO suite passed on its first execution, exit 0: store 48.070s, resolver 15.139s, daemon 28.290s. Tagged actual remember passed on its first execution, exit 0, daemon 0.471s, both body and finalizing, including close/reopen equality of the complete PendingEpisode. These are independently executed results, not the builder's runs.

The first targeted attempt did not compile because my new observer test referred to nonexistent `EntityMergeRequest.Losers` instead of `Retire`. I corrected only that field in my test. Its initial source hash and complete failed log are retained below; no implementation or supplied assertion was changed. That first pipeline lacked pipefail, so its wrapper exit 0 was not treated as a test success; the log explicitly records build failure. An initial read-only integrity script also reused zsh's special `path` variable and consequently could not invoke git; its apparent mismatch output was invalid. I reran with task-prefixed variables, successfully verified all original tests and then every unchanged baseline file. No content was changed by either integrity script.

## Findings

1. **Actual producer coverage is complete for the stated cooperating Store surface.** I searched every non-test store file for Update/NewTransaction/NewWriteBatch/Set/Delete/DropAll/DropPrefix/Load/Flatten/RunValueLogGC, examined mutation bodies and the unlocked helpers' callers, and checked the three-file change against baseline. `update` now holds graph shared around the actual db.Update including commit. Its root callers acquire maintenance shared first: PutEpisode, PutEntity, PutFact/putFactUnlocked, InvalidateFact, DeleteFact, PutCursor, DeleteEntity, ClaimAlias, DropAlias/DropAliasRehome, RelocateFact, RecordValueEvidence, AttestAlias, PutMetaTime, and PutMetaJSON. Empty/input-invalid branches can return earlier but do not mutate. Supplied valid fixture calls cover all these writers during both phases and both owner commit/refusal; exact raw maps remain unchanged until release and actually change afterward. Metadata tests use identity-looking keys.

2. **The remaining direct mutation sites have enclosing boundaries.** MergeEntitiesChecked, retirement variants, BackupAndRepairAliases, and private adoptLegacyInventory retain maintenance exclusive over their raw transactions. Restore retains exclusive maintenance over DropAll, Load, and ensureSchema. Open's other schema call precedes publication. Alias rejection and retirement/repair unlocked helpers have the inspected enclosing callers. Generation/observation/input-revision/outcome/journal/writer/fact-ledger codecs use their facade's transaction; root AtomicWrite owns graph shared throughout that transaction. No new independent raw producer or caller-selectable bypass was introduced.

3. **Admission locks precede the snapshot and survive both phases through commit.** `rootAtomicWrite` orders maintenance shared, graph shared/exclusive, then Badger. The new private entry requests exclusive graph; the old entry and public AtomicWrite request shared graph. Retirement revision remains sampled before waiting and checked after both locks. Facades take no root graph lock, retain their transaction, owner phase and poison state, and therefore do not upgrade or recursively reacquire the root lock. The supplied real preexisting AtomicWrite fixture excludes admission entry and puts its complete commit in the later ledger baseline. Public ClaimAlias cannot create its phantom during the new scope; the original old-harness public phantom still passes unchanged.

4. **Independent contention checks support snapshot serialization without changing ordinary transaction concurrency.** Twenty competing serialized owners wait behind a held finalizer, then each scans the prior alias map and atomically records a predecessor count and link. All twenty succeed with unique contiguous counts 0 through 19 and complete predecessor records. Separate independent tests hold an ordinary AtomicWrite or old owner body while a second public AtomicWrite commits; both old entry kinds remain shared. This does not assert fairness, global event order, or a performance bound.

5. **Only the fixed queue operations bypass graph coordination.** `updatePending` has exactly two callers, PutPending and DeletePending, each concatenating the literal `pq:` prefix and holding maintenance shared. Both delegate facades to ordinary update. Independent tests insert and delete empty, traversal-looking, embedded NUL, invalid UTF-8, newline, and identity/meta-looking IDs while each phase is held; exact keys remain under `pq:` and the complete store returns to its original raw map. Supplied tests prove root queue reads/counts/put/delete progress, queue bytes/deletion survive owner refusal and reopen, and facade queue writes roll back or refuse in body/finalizing/closed/poisoned states. A root AtomicWrite containing only queue work remains coordinated. The actual remember path uses HasEpisode, HasPending, PutPending, nonblocking Kick, and PendingCounts, all on the same root; its tagged test proves completion while each phase remains held and durable reopened input after refusal.

6. **Failure and publication behavior survives the refactor.** Supplied tests exercise errors, nested entry refusal, late writes, body/finalizer panic, genuine Badger ErrConflict, actual transaction capacity failure, malformed entry, exact raw rollback, zero failed events, closed owner state, and subsequent graph/queue progress. An independent nested AtomicWrite writes pending input and panics; its caller recovers and ignores a subsequent queue error. The outer scope still refuses, never finalizes, and exactly preserves all bytes; later queue and metadata writes progress. An independent observer performs an actual valid MergeEntitiesChecked after both ordinary and serialized commit and removes its fixture loser; this proves exclusive maintenance reentry as well as graph reentry after both root locks release. Supplied committed observer panic preserves committed bytes and releases graph access. Existing direct-mutator maintenance/observer restrictions and arbitrary input-error behavior were not redesigned.

7. **Maintenance exclusion and retirement refusal are exercised with valid operations.** Supplied forward cases execute actual merge, checked merge, retirement variants, backup-coupled retirement/alias repair, Restore, and adoption preview/apply after body/finalizer release. Reverse cases pause real merge/retirement postconditions, backup sync, or Restore reading, prove no admission callback enters, and preserve retirement's ErrNotFound refusal after waiting. Adoption reverse evidence is explicitly compositional: the test holds the exact maintenance lock; source takes that lock before adoption's transaction; real forward preview/apply exclusion passes. This is not an actual adopter paused in reverse direction.

## Strict limits retained

The independent raw/post-verification group deliberately demonstrates two counterexamples even under the new coordinator: a separate raw Badger writer creates an alias absent from the ledger's prefix-scan view while both transactions commit; a raw finalizer write after successful ledger verification commits outside the returned Final report. Both PASS only because the complete contract excludes these guarantees. They prevent interpreting the code as universal predicate locking or complete finalizer accounting.

The contract requires one genuine Open-produced root, cooperating supported producers, and nonconcurrent, unescaped facades. Captured-root graph writes called synchronously inside serialized callbacks can deadlock and remain excluded. Raw writers, forged handles, concurrent Close, and other lifecycle misuse remain excluded. The capacity test uses a single synthetic in-memory handle to induce real Badger capacity failure, not to prove coordination between forged/shared handles. Ordinary public ClaimAlias after a scope can still claim a missing owner; serialization supplies no new ownership, authorization, adoption, retirement, support-selection or production lifecycle policy. Metadata coordination does not authorize identity marker replacement.

An admission transaction that reads/writes pending keys can conflict with a root queue update; the supplied actual queue-conflict test confirms rollback while keeping the competitor queue input. Pending exclusive maintenance can block subsequent maintenance readers, including queue operations. No queue progress is promised under pending maintenance or storage/resource failure, and no full-database isolation follows. The actual remember evidence is controlled synthetic progress and durability, not live p95. Metadata-stamping enqueue RPCs may wait even after queue bytes commit. Global observer/index ordering, finalizer untracked writes, raw-write accounting, whole-goal regressions, ownership/adoption correctness, live graph health, cleanup and sweep stability are not certified.

The private entry has no default-production caller: source search finds only its declaration and the private tagged bridge. `go list` excludes `private_serial_probe.go` from default GoFiles. The bridge and daemon test must remain exclusively in the private evidence export; do not integrate either file, even though a build tag currently excludes them. Integration of this bounded change does not route any production producer through serialized admission and must not imply permission to do so.

## Evidence pins

Paths are relative to `/tmp/scry-serial-code-disproof-xzOvG2` unless marked external.

| Artifact | SHA-256 |
| --- | --- |
| SERIAL_ADMISSION_CONTRACT.md | 91071f6561eb70859c5f657b5f4d6198104eb6041cd1c563bb87c02d20bd3817 |
| SERIAL_ADMISSION_DESIGN.md | dedcbb18e986639509e8a64284b75cc6580b382868012139684af326e7908106 |
| External complete design review | 9ac136e5663f75ac5207bf328b0a7cd3b517ac64e8635a6d63a050a09e6acef0 |
| External complete identity ledger review | 618182c8444ea1c794034d4a4af743cafa8390387878c8bcca2ca8e90609b2f5 |
| internal/memory/store/store.go | 8093c39a9214e1dbe5546a8cdd5a8cd8c76836952bf2a8616630eeda3ba8f7b1 |
| internal/memory/store/pending.go | f51c3dec4f7aef2a93e5195d229541ba7253e244606e806fe00d92541df6afb2 |
| internal/memory/store/identity_admission_owner.go | 396389b06177ce1cd4af18f23412eea4e1f747696d93fc794d82025bfc89dac1 |
| internal/memory/store/identity_serial_admission_test.go | 68686ff2824011cd91606bbd6f9c582ed43aea79d56dcb27ff2d749314782d98 |
| internal/memory/store/identity_serial_maintenance_test.go | 4a176605f9d38715dec88e069f4235bfa93c01bd526f1a8a2152bbae1f4ffacf |
| PRIVATE ONLY internal/memory/store/private_serial_probe.go | 8219a560336304e4cc3ada99b8a30722c809d0b59029439c8f0eae788baeb2eb |
| PRIVATE ONLY internal/daemon/private_serial_remember_test.go | ec15e21aa452a6833aa81b80b5f09addce56a750288af25e0bc21ed25e880fe3 |
| Initial independent test before field-name correction | f48f7c541f73ed52676616ca43bf708c55913ea58b5eb33f1ea00940fe01b84c |
| internal/memory/store/serial_code_independent_test.go | bf566fa9dc3879f7b14d08231230cc2fd0acf223831d2a9a2463cc3dbf8b4999 |
| serial-code-targeted.log (initial compile failure) | 2323ade218fdafdfa135802b87f6f7c8e656e57651a06cf06826dfd7c3883b36 |
| serial-code-targeted-corrected.log | 00941fe5d68a58532178f60c28f58c4b119cacc423866e8b3a7dc68db85755c5 |
| serial-code-fullsuite.log | 89f6f4182be4d4dff956dca53bee54db67f384ffd393344b7a33d6e6457ab950 |
| serial-code-real-remember.log | fee714dd41d4dcd022a4a7b16bbb93ca08d0dbaa5a302c496e026cb3bbab95cc |

This report's hash is supplied separately. The lead must read the full report before citing this bounded verdict.
