# Durable reviewed alias rejection — implementation and verification plan

This is a bounded dependency of the active
[memory-quality goal](../MEMORY_IMPLEMENTATION_GOAL_2026-09-04.md), not a replacement
or weaker completion bar. Status: candidate implementation, not deployed; no live
rejection markers or additional alias removals have been applied.

## Proven failure

Independent restored-source review of deployed `62cf6e0` proves that an explicitly
removed, subsequently unclaimed alias can return through stale `PutEntity` (including
`AtomicWrite`) or repeated ordinary extraction. Two synthetic normal `Apply` calls
restore Envoyer to ChildScribe. In the prior 49-literal removal batch, 40/45 normalized
keys can return through direct writes and 14/45 pass repeated resolver admission.
These are reproduced exposures, not observed live regrowth.

Evidence: [independent gap report](alias-reintroduction-gap-review-2026-09-05.md).
Source: Mini backup `memory-20260905T211821Z.badger`, 78,021,343 bytes, SHA-256
`3f09b0d6622d4612811b83fc6738f845063c4d19b07ced41c7b893e9b4421e38`.

## Invariant and narrow first implementation

- A reviewed negative decision forbids a normalized spelling for one entity, not
  for every entity. No recipient is inferred and no global retired-value marker is
  used. A distinct real Envoyer tool must remain representable.
- Add `ar:<entity slug>:<normalized spelling>` records containing the literal
  dispositions, reasons and exact disposition-plan hash. Preserve multiple literal
  variants/reasons rather than overwriting evidence. Do not bump `SchemaVersion`:
  the existing mismatch behavior wipes the store.
- Only validated explicit `BackupAndRepairAliases` and reviewed merge `DropAliases`
  create these records, atomically with the disposition. Ordinary alias omission,
  heuristic revalidation and low-level `DropAlias` have no review authority and
  create none. Prior positive attestations and source facts remain intact.
- Preflight every incoming name, alias and normalized slug inside `PutEntity`'s
  transaction, including stale and transactional callers. `ClaimAlias`, low-level
  rehome and retirement rehome must not target a rejected pair. Resolver admission
  consults the decision before shortcuts or attestations; typed write conflicts
  park the complete pending episode.
- Fingerprint prior decisions, expose proposed decisions and verify their exact
  post-state. Serialize marker writers with exclusive maintenance locks, publish
  observers after releasing them, and preserve zero-write failure behavior.
- **First-version limitation:** reviewed merge participants with existing rejection
  records are refused. New reviewed merge drops create decisions for the group and
  explicit external `drop_from` identities. Inheritance or supersession is a later
  explicit reviewed design, not an implicit consequence of selecting a survivor.
- Keep legacy broad hygiene/migration apply disabled. This change does not certify
  dormant heuristic merge code, and must not be used as authority to re-enable it.

## Required gates, in order

1. Reproduce stale direct/atomic writes and raw claim regression before the fix.
2. Pass fixtures for normalized variants, own-name shortcuts, old/new/repeated
   attestations, two normal Apply calls, transaction rollback, malformed records,
   backup failures, reopen/restore, other-owner positives, rehome conflicts, merge
   refusal and merge-drop evidence preservation. Verify no review is fabricated
   from ordinary omissions. Run all tests, vet and affected race suites.
3. Restore the verified live source independently. Apply only a disposable explicit
   three-alias technical fixture; compare the entire predicted raw delta, all
   current/history facts, episodes, pending records, metadata and claims. Exercise
   actual normal Apply, stale writes, restore/reopen and exact second no-write.
   Technical fixture PASS is not semantic approval to drop the three live aliases.
4. Fresh-context code and replica graders attempt to disprove the result. Fix
   reachable violations and document unapproved/disabled boundaries. The consensus
   skill's prescribed Opus/Sonnet engines are unavailable; these independent goal
   graders are a disclosed fallback, not a multi-model consensus claim.
5. Before deployment, take and independently restore fresh nonempty Mini and laptop
   backups, verify active code/processes, retain both previous binaries, and compare
   baseline suites. Deploy one reviewed static artifact to both machines; verify
   byte identity, process restart, ingestion and queue state, unchanged graph on
   open, five suites and bounded latency. No marker is synthesized at startup.
6. Only then generate a fresh explicit alias-disposition manifest with complete
   source semantics. The three proposed ChildScribe removals remain held. The prior
   49-row repair requires a separate explicitly reviewed negative-evidence backfill
   capability and fresh manifest; do not replay its stale manifest or infer every
   absent alias is rejected. Such backfill is not implemented by this first change.
7. Each live disposition requires immediate full semantic-closure recheck,
   nonempty automatic backup, independent actual-pre/post raw audit, exact lookups,
   five suites, second no-write and durable receipt. Record marker provenance and
   all expected/predicted/actual changes, including any ordinary-ingestion drift.

## Rollback restriction

Old binaries ignore additive rejection records. Before the first live marker, a
binary-only rollback is possible with the usual verified backup and state review.
After markers exist, **never run an old writer against that marker-bearing store**.
Keep the old binary and verified pre-marker backup, but do not automatically restore
it over later ingestion: reconcile intervening writes first under a reviewed plan,
or deploy a corrected marker-aware binary. Retaining an artifact is not approval
to discard later facts or decisions.

## Other pending work stays separate

The four-hollow-remnant candidate passed a bounded replica gate on `62cf6e0`; it is
not live-applied. The exact parked migration-note retry has a renewed expanded
closure gate, but remains unexecuted. Both require fresh immediate checks; changed
code/fingerprint formats invalidate old apply readiness. Mixed `migration0160`
ownership remains unresolved. Existing five scores remain below the first two
original floors, and broad identity, collision, hollow/dangling and sweep gates are
unfinished. Nothing here marks the active goal complete.
