# Independent code disproof: reviewed alias rejection prevention

Date: 2026-09-05. Base HEAD: `8d4c0c86fa343e9c7b70caccc3b05b5ab23e2a9f`.
Scope: uncommitted production prevention diff and new rejection implementation; no live store changes, semantic drop decisions, or provider calls. Tests ran in this independent HEAD archive plus the candidate diff and untracked candidate tests. Reviewer added tests only in this archive.

## Verdict

No reachable candidate enforcement blocker found for the supported ordinary resolver/store writes and reviewed repair/merge/retirement APIs. This is a bounded code review result, not approval of live semantic drops, deployment, rollback, or the overall memory-quality goal.

One dormant legacy bypass is independently reproduced below. Keep the legacy hygiene and migration apply paths disabled. The candidate does not make those underlying legacy functions inheritance-safe.

## Reproduced dormant boundary

`internal/memory/resolve/hygiene.go:945` (`mergeStub`) relocates facts, transfers present spellings, writes the target, deletes the source, and claims source spellings without inspecting rejection decisions on the source. It leaves the source's rejection records at the old slug. The new survivor therefore has no inherited rejection.

Reproducer: `internal/memory/resolve/disproof_alias_legacy_test.go`, `TestDisproofDormantLegacyMergeLosesRejectionInheritance`.

1. Create a concept stub `legacy-child` named `Child Scribe`, alias `Envoyer`, and project `real-child` named `ChildScribe`.
2. Apply a reviewed alias repair dropping `Envoyer` from `legacy-child` with backup and current fingerprints.
3. Call the dormant `mergeStub` helper directly on the cleaned stub and project.
4. Confirm the source still has its rejection and the survivor has none.
5. `PutEntity(real-child with Envoyer alias)` succeeds.

This is not a reachable CLI/RPC prevention blocker in this build:

- Non-test callers of `mergeStub` are `mergeDuplicateStubs` and `mergeLocatedDuplicates`, both called by `Hygiene`.
- Non-test callers of `Hygiene` are `handleMemoryHygiene` and `migrate.Run`.
- `internal/daemon/memory_methods.go:491` rejects hygiene apply before reaching the store.
- `internal/daemon/memory_queue.go:607` rejects migration apply before reaching `migrate.Run`.
- `cmd/scry/memory.go:109` rejects `memory migrate --apply`, including offline `--dir` usage.
- Repository-wide call search found no ordinary sweep/startup path reaching either legacy mutating helper.

Do not describe the implementation as making every historical merge function inheritance-safe. Its supported reviewed merge API refuses participants with prior rejection evidence, and its legacy apply entrypoints remain disabled.

## Enforcement and safety evidence

- `store.go:419`: PutEntity checks every normalized current name/alias and the slug against the rejected owner/spelling pair before any writes. Stale ordinary entity writes and AtomicWrite facade writes cannot restore removed spellings.
- `store.go:1196`: ClaimAlias checks the pair before explicit routing writes. Rejection is owner-specific; a distinct legitimate entity can use the spelling.
- `store.go:1263`: low-level rehome validates the target pair in the same transaction as source removal, so refusal rolls back removal. Its lack of a new rejection marker is intentional: the helper has no reviewed reason/manifest.
- `resolve/aliases.go:239`: AdmitAlias reads through the Store facade before own-name/already-listed shortcuts or attestation handling; reviewed rejection declines without accumulating further attestation evidence.
- `queue.go:514`: ErrAliasRejected is classified with the existing permanent resolver failures; errors wrapped with `%w` remain detectable through `errors.Is`.
- `alias_rejection.go:34`: malformed/empty JSON, mismatched entity/alias/key, invalid source slug, missing reason, and empty plan fail closed. A plan is required to be nonempty; it is not a signature/authenticity check. Legitimate production writers use hashJSON of the reviewed disposition input.
- Rejection keys are unambiguous because valid source slugs exclude colons. Alias normalization is shared with routing normalization; spelling variants using case/spaces/underscores/hyphens hit the same pair.
- `unalias.go:123,181,219`: staged markers are considered before a rehome commits; a target dropped in the same batch cannot be used as recipient. The full original state is analyzed before writes, and a modified entity may not retain a newly rejected canonical or alias spelling.
- `unalias.go:240`: exclusive maintenance lock covers backup, Sync, Close, analysis, write, and postconditions. Backup failures abort before marker or graph writes. All entities/facts/claims are compared with exact intended post-state; marker state is also exactly compared, including prior decisions.
- `merge.go:128`: reviewed merges use exclusive maintenance locking, preventing marker phantom inserts by another reviewed writer. Observer dispatch occurs outside this lock. Existing rejection-involved participants (including external disposition entities) make preview unready. Newly reviewed drops mark each group member and explicit DropFrom participants. Dropped spellings retained in proposed metadata are refused.
- `retire.go:307,505`: retirement rehomes check the target pair; expected fingerprints include participant rejection evidence. Retirement does not delete `ar:` evidence.
- `DeleteEntity` leaves `ar:` evidence intact; independent delete/recreate test confirms same-slug recreation with a rejected name fails without changing raw state.
- All production raw entity/routing writers were searched. Beyond the covered paths, Restore is deliberately destructive administrative restoration, not an ordinary admission path. Legacy hygiene composes supported low-level writers but remains subject to the dormant inheritance limitation above.
- No schema version, retention setting, provider configuration, or automatic rejection backfill was changed. Backup streams include the new keys; candidate restore/reopen tests compare complete raw state.

## Explicit operational limitations

- Older binaries ignore additive `ar:` records. An older writer must not be pointed at a marker-bearing store. Reverting the binary is not a safe rejection-preserving rollback. Any pre-marker-backup rollback must be independently verified and reconciled for all later facts/changes before use.
- No automatic backfill exists. An alias removed before this feature is not retrospectively protected unless an explicitly reviewed procedure records that decision; this review does not authorize such decisions.
- The ordinary API `ClaimAlias` remains an explicit routing-transfer primitive, not a general authorization mechanism. Rejection pair checks do not decide the proper owner.
- Reviewed merge backup remains supplied by its caller; each group is atomic, but existing CLI/RPC multi-group merge application is not one backup-to-commit batch transaction. Those existing semantics should not be described as the stronger atomic backup-first AliasRepair semantics.
- Public raw restoration or a future new raw writer can bypass the policy unless it preserves/enforces marker semantics. No general claim about arbitrary direct Badger writes is made.

## Verification

PASS: `go test ./internal/memory/store ./internal/memory/resolve ./internal/memory/queue` (10.964s, 17.543s, 9.592s package results).

PASS: independent `go test ./internal/memory/store ./internal/memory/resolve -run TestDisproof -v`:

- `TestDisproofRejectionSurvivesDeleteRecreate`
- `TestDisproofMismatchedPayloadFailsClosed`
- `TestDisproofStagedRehomeRejectsLaterDrop`
- `TestDisproofDormantLegacyMergeLosesRejectionInheritance` (passing reproduction of the dormant weakness, not an expected safety guarantee).

PASS: `go test -race ./internal/memory/store -run 'AliasRejection|ReviewedAlias|AliasRepair|BackupAndRepair' -count=1` (5.038s).

PASS: `go test ./internal/daemon -run 'TestLegacyInferredIdentityApplyEndpointsAreDisabled|TestMemoryMergeEntities' -count=1` (0.914s), including the production entrypoint guards for dormant legacy mutation.

## Candidate production SHA-256

```
297d02a25663597bcfb692e0d928cca224b5ec38b918682558535ed11a2cee78  internal/memory/store/alias_rejection.go
b6751bdd5375f1826433cb44d27a1d5b3df4915dad0acedc13e1547128e44afc  internal/memory/store/store.go
60d2d2ea5f3aae98ba6e2120e42b2bcb45f4db4ca9dd587aaf58f8526d2f6fb2  internal/memory/store/unalias.go
f462887b76504a8d51cea375d8705b43c4741967968377820b89638dcd85489d  internal/memory/store/merge.go
b76ad3095e7d173a8aa050c5c9497719d76db0ea2906f7e0923b3936f0d62734  internal/memory/store/retire.go
6a96e54997520a10e366f9b1cfc9f3d57beed00c4699de0394456f905e47bdc8  internal/memory/resolve/aliases.go
9f1168cb3da5d6a7c10f1e1d312b526ed5a8fdcad1f980ae1ded54a9365f01e0  internal/memory/queue/queue.go
```
