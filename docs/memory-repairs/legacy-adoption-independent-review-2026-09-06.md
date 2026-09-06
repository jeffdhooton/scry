# Bounded independent legacy-adoption disproof

Verdict: bounded PASS. No violation of the frozen ADOPTION_CONTRACT.md was demonstrated. This verdict does not approve live adoption, lifecycle rollout, admission-policy completeness, semantic cleanup, or the overall memory-solid goal.

Read the full active objective and supplied applicable AGENTS instructions; ran the required orientation before task work. Exported baseline `315fa2a24518995bc62ca54d3cc47f0bac17e48c` with git archive into `/tmp/scry-adoption-disproof-jRBRuU`. Copied the four frozen candidate files through apply_patch. All independent tests and replica writes stayed in this private workspace; candidate/shared paths and live services were untouched.

## Frozen files (SHA256)

- ADOPTION_CONTRACT.md: `4b66855ec03ed7f5120b2ef6bd6d8adbf3cb759eb4c5924be4a3d0e7da2f38b7`
- internal/memory/store/identity_legacy_adoption.go: `285ae96d0886e70a65ba437690890d9fe61edf6cd814341d91bc13d61d83b635`
- internal/memory/store/identity_legacy_adoption_test.go: `064f38a2ad141d1bec923ff6bd1a9bb8496e6f88ee4c8542ce1a851262bad454`
- internal/memory/store/legacy_adoption_storage_test.go: `155e1ce931691988592fd38f0e8d8dd36b93148468b2eb05257df93d4995224d`
- Independent tests, internal/memory/store/independent_adoption_disproof_test.go: `51d0718319d9df78c8578748e51551aa54e5ffc8c8806ec622ede6637d36a236`

## Independent challenges

- Exact preview/apply write sets, all original raw rows unchanged, exact proposed byte count, zero observer events, all identities readable, and replay refusal. Returned manifest and decoded entry buffers do not alias earlier results or input.
- Drift in every entity field; alias and repository-reference order; unknown source fields and whitespace; first/middle/last missing and additional entries.
- All seven reserved families, at empty and nonempty suffixes including invalid UTF-8, with empty, malformed-byte and JSON-null values: 63 occupied-family fixtures refuse both modes without writes.
- Canonical manifests: duplicates, unknown/case-changed keys, invalid numbers, nulls, missing entries, reverse order, duplicate entries, wrong keys/values, trailing whitespace and concatenated documents refuse.
- Canonical marker/anchor controls and current-record corruption refuse. Transactional recognition observes staged controls. Empty/corrupt consumption and generation records refuse; deleting/recreating the original entity inside a transaction cannot reactivate a retained consumed anchor. Rollback restores the prior recognition state.
- Ignored adoption refusal poisons ordinary nested transactions and admission body/finalizer scopes, with staged data rolled back even for an invalid manifest.
- Eighteen ordinary Store producer entry points wait on the root exclusive maintenance lock: entity create/delete, episode, fact put/invalidate/delete/relocate, cursor, alias claim/drop/rehome, pending put/delete, metadata time/JSON, attestation, value evidence and AtomicWrite. Static inspection confirmed private unlocked writers are reached through their locked public wrappers; startup schema initialization is not a concurrent producer. AtomicWrite facades operate under the enclosing root read lock until commit.
- Real late staging failures: independently counted 1,080 accepted anchor writes before transaction size refusal and two accepted anchor writes before an actual value-size error. Both adoption preview and apply return zero reports and leave all raw rows unchanged. Static error sanitation preserves only approved transaction/conflict classes and omits underlying private strings. Closed-store errors are sanitized.

## Commands and evidence

Final independent test version passed `CGO_ENABLED=0 go test ./... -count=1` (exit 0), including supplied candidate tests and independent tests. Full output: `independent-full-suite.log`.

Initial broad custom fixture output is `independent-targeted.log`; its explicit backup test was correctly skipped without the opt-in environment variable. Final late-failure count output is `independent-late-failure.log`, generated with `CGO_ENABLED=0 go test ./internal/memory/store -run '^TestDisproofAdoptionMeasuredLateFailureRollback$' -count=1 -v`.

Actual backup experiment used `SCRY_DISPROOF_RESTORE=1 CGO_ENABLED=0 go test ./internal/memory/store -run '^TestDisproofAdoptionIndependentActualBackup$' -count=1 -v`. The test read and SHA256-verified source `/tmp/scry-foundation-closure-sep06.8IEPu5/memory-20260906T051519Z.badger` as `193f19b3491c74782d7556e3059714cbb7ba65005603ce38f84d70efcf52c256`, then restored into its own new `/tmp/scry-adoption-disproof-jRBRuU/replica-1013414481`. It never opened the builder's replica.

Independent measured result: 247,638 original rows, all byte-identical afterward; 31,247 adopted entities; 16,308,695 manifest bytes; inventory SHA256 `5292699c3a75e5ce4d34ff9bd2a41e2900bf53c0dbf549358780191c04c488c3`; exactly 9,178,163 new key/value bytes; 31,247 anchors plus one marker; zero observer events; every adopted identity recognized; preview non-writing; replay refused. The replica test passed in 3.66 seconds. Full safe count/hash output: `independent-replica.log`.

No failing regression was found or removed. The replica remains available in this private workspace.

## Required limits

Maintenance-lock coordination is not arbitrary raw-writer phantom immunity. Direct private DB/transaction access, forged facades and concurrent Close remain outside the stated contract. Prefix scans do not prove immunity to those actors.

This unit installs no ongoing admission policy and no tombstone/consumption persistence API. Existing normal writers still lack lifecycle enforcement after adoption, so this bounded PASS cannot authorize live adoption. Recognition assumes retained controls; it does not audit the complete inventory or prevent an arbitrary raw writer from forging/deleting them. All-writer lifecycle/controller integration and the separately required reviews remain necessary before rollout.
