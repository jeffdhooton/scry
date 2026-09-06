# Private identity/alias actual-writer history

Baseline42356e1, 2026-09-06. Implements only the smallest unit permitted by
independent design review94e6e51f; no production routing, support/undo selection,
ownership grant, full relationship inventory or lifecycle policy.

Begin only on a live admission-owned body-phase Store facade; create its own
reviewed identityJournal. Keep exact facade and owner. Actual private put/delete
methods accept actor slug, key and optional new raw bytes. Actor must be a valid
entity slug. Only en:<valid-slug> and al:<nonempty-valid-UTF8-normalized-spelling>
keys are accepted. No att:, fa:, episode or evidence mutations. Empty/malformed
keys refuse. Existing opaque before-images remain exact, not reserialized.

An en: key must name the actor. A new en: value must pass strict complete canonical
legacy entity decoding and its body slug must equal that actor; legitimate name/
slug divergence remains permitted. This is format/binding validation, not legacy
classification or permission to replace a different identity. For al: put, full
new owner bytes must equal actor. For al: delete, actual old owner bytes must equal
actor. Any deletion of an absent key refuses; no fabricated deletion/no-op actor.
An identical put succeeds and records a mutation, retaining exact before/after.

Read actual current bytes and compare them to the last tracked after-image for
this key if previously touched. Capture through identityJournal BEFORE Set/Delete.
Own every key/value before Badger can retain them. Record successful writes in
global execution order with a generated contiguous sequence starting at0, actor,
exact key, exact before presence/bytes and exact after presence/bytes. An error
records no successful mutation, poisons the owner and forces outer rollback even
when swallowed. Only static errors escape; retain static ErrTxnTooBig/ErrConflict
classifications. Existing prior owner errors keep their outer precedence, while
the primitive returns its own static refusal rather than leaking earlier details.

One freeze in finalizing phase checks actual final bytes/presence of every tracked
key against its last record, and returns owned ordered history plus owned maps of
exact first-before and final images. No partial result on failure. The original
journal remains private; no undo or event suppression is offered here. Reject
body freeze, finalizing writes, repeated freeze, closed scope, wrong owner/facade,
nil/zero capability. Any associated owner is poisoned on failure. The same private
field-corruption/nonconcurrent-use assumptions as the owner/fact ledger apply.

Only the primitive's own en:/al: writes occur; zero observer events are generated.
It does not prove there were no untracked changes to DIFFERENT keys. Complete
baseline/final inventories and relationship closure remain required in the fixed
finalizer before cleanup. It does not certify actor authorization, recognized
identity, permitted alias transfer, retained listing correctness, or which birth
has support. A successful ClaimAlias/DropAlias or a matching actor is not proof.

Tests: multi-actor claim/drop/later-claim history; distinct generated operation
sequences; canonical new entities and opaque old images; unchanged puts; absent
delete refusal; actor/key/value mismatch; malformed/noncanonical keys/values;
first-image capture across recapture; raw state changing between operations or
before freeze; swallowed errors/phase/foreign/closed/rollback/panics; ownership of
caller/output slices; real late storage/value failure; exact full raw/event proof.
No existing producer is routed to this unit by its initial implementation.
