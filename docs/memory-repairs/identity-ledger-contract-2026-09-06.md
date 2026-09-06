# Private complete identity mutation accounting

Baseline699da3b includes relationship inventoryeae817f6 independently reviewed
under report46fb43bf. Completes a mechanical accounting
boundary, not fixed ownership policy, undo, registration, generation materialization,
observation/result completeness, dependency selection or production routing.

Begin only on a live admission-owner body facade with no earlier failure. Obtain
its OWN complete relationship inventory before any method writes, then its OWN
reviewed identityWriterHistory. No caller baseline, writer, actor authorization,
support flag, undo set or expected-after map is accepted. Same facade/owner binding
and nonconcurrent/private-field-integrity assumptions as those reviewed primitives.

Private put/delete actual methods check body/owner/live/poison state, then delegate
the actual en:/al: operation to the owned writer. Its exact actor/key/new entity
and alias owner validation remains unchanged. All wrapper failures return static
errIdentityMutationLedger plus allowlisted ErrTxnTooBig/ErrConflict and poison the
owned scope. Prior owner errors keep outer precedence; no arbitrary details escape.
The delegated writer may poison first, so its earlier static sentinel remains the
outer error even though this wrapper returns its own static sentinel locally.

Verify once in finalizing. Freeze actual writer history; obtain a complete actual
final relationship inventory in the same transaction view. Start from the exact
owned baseline raw map and replay every generated sequence in order, requiring
each recorded before-presence/bytes equal the replay state. Apply its after-state
to that expected map. Finally require full key-set AND value-byte equality against
the actual final inventory, not just digests/counts. No partial report on refusal.

This detects untracked final en:/al: changes, including before a key's first tracked
write, on other keys, after its last write, and selected lifecycle/control changes
anywhere. The en:/al: writer cannot explain an ig:/il:/consumption/adoption/negative
marker change in the body; those therefore refuse. Finalizer materialization is
separate and may occur only AFTER this boundary under its future exact accounting;
this verifier does NOT certify writes performed later or final graph post-undo.

Return independently owned Baseline, Final and full exact History snapshots. No
output map/byte/array can mutate private captured baseline, writer or another
projection. Repeat/early/late/foreign/nil/closed use refuses and poisons when an
owner exists. Ignored operation/verification failure, panic and actual commit
failure must not commit this scope's writes/events. No event is emitted by this
primitive. Outside selected inventory families are not measured or changed by it;
fact/evidence ledgers remain separate.

The verifier does NOT detect transient raw writes exactly reverted before the
next observed state, arbitrary forged private fields, or concurrent raw-key range
phantoms outside the transaction snapshot. Complete production producer coordination
(including public ClaimAlias) is still required: a full prefix scan is NOT a
serializable predicate lock. Neither this source nor its tests authorize cleanup,
live adoption, alias transfers or intentional assertion loss.

Tests: generated multi-actor replay/full raw map; selected control drift and
untracked create/update/delete/empty-presence before first/after last/on other keys;
scope/owner/poison, all returned ownership, zero operations, opaque controls,
malformed entities, actual storage errors/conflicts, rollback/panics and prior-error
precedence. No normal Store producer is rerouted in this unit.
