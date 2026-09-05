# Independent alias repair gate — 2026-09-05

Verdict: bounded PASS for 53fafa91d621190c87245f0b0844270b4fdf44c9. The preceding commit 1b913a1d6cf5cd9bee95614de570fe1ed5cca4ad FAILED the CLI daemon-downgrade probe. This review does not certify any live repair manifest or the full memory-quality goal.

Reviewed production code was independently extracted with git archive into /tmp/scry-unalias-gate.CJNhlI (initial) and /tmp/scry-unalias-regate.gRhAmY (fixed). Only extra tests/report were added inside those temporary trees. No live writes, deployment, remember calls, model calls, or shared source edits were performed. Session began with the required memory orientation; the complete goal contract and final decisions were read.

## Proven initial violation and fix

At 1b913a1, cmd/scry/memory.go:1479 previews using callMemoryDaemon, then line 1488 uses that helper again for apply. cmd/scry/daemon.go:267 opens a new connection on each call; the helper also retries across restarts. A current preview followed by an old handler therefore sends dry_run:false to a daemon that ignores expected. The independent CLI test observed two legacy-method calls, one unguarded write, and nil CLI error. Reusing the response object also retained preview.ready and expected when the old response omitted preview.

At 53fafa9, the CLI applies only through memory.unalias.apply-reviewed.v1 and clears its response first. The daemon registers that method to the guarded handler. The exact original failure probe now observes one dry preview, zero old-handler writes, and RPC -32601 method-not-found on downgrade. An old daemon present from the start is also refused. An independent successful-new-method test verifies one preview, one versioned write, explicit dry_run:false, and exact drops/expected delivery.

## Independent evidence

All added probes ran with -race. Commands (from fixed archive):

```
go test -race ./cmd/scry ./internal/memory/store ./internal/daemon -run TestIndependentUnalias -count=1 -v
go test -race ./internal/memory/store -run 'TestIndependentUnaliasBackup' -count=1 -v
go test -race ./internal/memory/store -run 'TestIndependentUnalias(Drift|Global)' -count=1 -v
```

The final two commands cover tests added after the first invocation. Every independently added final probe passed:

- Raw RPC omitted/null/true dry_run preserves both aliases and creates no backup; explicit false with absent or empty expected refuses; complete reviewed input applies both rows with a backup.
- Missing, self, non-listing, simultaneously-dropped, and conflicting rehome targets refuse the entire batch. Own-name and later-invalid rows also refuse. Raw key/value snapshots remain identical, with zero observer events.
- Historical incoming fact provenance, phantom outside listing, empty-versus-absent alias index, claim-only owner, plan-only reason, and participant metadata drift all refuse. No raw key/value change or event escapes.
- Rehome removes all normalized source variants exposed in preview and preserves other metadata. Backup restores the exact original raw key/value map.
- A 1 MiB Badger memtable and seven roughly 40 KiB entity records force ErrTxnTooBig after earlier transactional writes fit. All durable keys remain unchanged; no events fire.
- Test-only reflection/unsafe access sets Badger's atomic blockWrites flag after backup close. All transaction writes and postcondition reads are accepted, then actual commit returns ErrBlockedWrites. All durable keys remain unchanged; no events fire; a subsequent ordinary write succeeds after the injected flag is cleared.
- Backup Write, Sync, and Close failures refuse with no key/value changes or events, and close is attempted.
- While backup Sync is held, a concurrent AtomicWrite/ClaimAlias waits; it completes only after backup/apply release.
- A callback can enter AtomicWrite, proving lock release. An injected observer panic propagates after a complete database commit; backup restoration and a subsequent normal write still work.
- Dropping the indexed owner while another listing remains is refused. Explicitly dropping both listings removes the claim. Dropping a source listing whose index already belongs to the target preserves the foreign claim.

Static inspection verifies one transaction for the whole batch, full entity/fact/index postcondition comparison, preserved historical facts, and fingerprint equality inside the writing transaction.

## Limits

Fixture evidence and source inspection only: the lead's real-store replica measurements were not independently repeated here. This does not grade ChildScribe/Forge semantic ownership, approve a live manifest, prove the full ten-clause goal, or certify deployment backup/binary discipline. Real process/power loss and filesystem durability failures were not simulated. The backup writer faults are synthetic; commit failure uses a test-only Badger flag. Observer panic is not recovered by the store and may prevent later observer notifications, although it cannot partially commit this batch; this review verifies commit/lock behavior, not observer-panic recovery.

Store algorithm SHA-256 in both archives: 2f3c74bde5e4f6f9ccaaf902dcbb98befa2ad6e77417b92983d95bd48affc506.
Fixed CLI source SHA-256: 75c343bc6fba8b876fe197195f14b2b8d9409a3c12caa51818ed39ba51b29486.
