# Durable revision-conflict implementation

2026-09-06. Private6qYb7K only; no actual memory replica/live use or independent
approval. Main production e097fa6/live a06cd7b unchanged. Full goal remains active.

The fixed applyCompleteAdmission entry now implements its actual changed-input
conflict branch. It clones original input/EP before waiting, acquires the real
cooperative owner, reads actual selected full input/result/head and immutable EP,
and derives acceptance (or V1 acceptance-unknown). Only a genuinely different
input with prior acceptance/unknown takes this branch. It writes canonical new
input plus a canonical immutable io-conflict record, never graph/EP/head changes.
The record binds actual selected input/result keys, full selected-head/EP bytes
and derived unknown status. The finalizer verifies zero EN/AL/FA history, zero
candidate creation/events, and exact complete transaction raw delta. Only after
outer commit succeeds does the caller receive typed errAdmissionRevisionConflict.
Storage failure returns zero result and ordinary static error after rollback.

Normal no-head/same-input/prior-none-accepted materialization remains a temporary
explicit refusal in this SAME private entry; it must be connected next, not shipped
as this incomplete policy. The separate complete draft/replay remains read-only.

First stub49e94e13ce306c33951f71025e3aafd26ff8d22e7f4d8548eca2072937d6b80c is preserved
exactly (reconstructed from its authored content and hash-verified); first actual
durability test FAIL.486s, log40066e5dff93ab2cb1db0e9cf60431ba0c289a3b17be8507b85bdb92235986ee.
Initial implementation first tests PASS.466s, log
e3b27f8f04d0635b924f820bb7b1a4b9833f19768d497cdc6fe29945ce9b3cd0.
Those prove exact two-row delta, old-row conservation, zero events, deduplication
and changed-source-identity refusal without input preservation.

Actual backup/restore/reopen test passes. Expanded V1 tests proved two failures:
same input was mislabeled a changed revision, and a changed-input conflict omitted
the known original input key. Acceptance unknown does not mean input identity is
unknown. FAIL.656s/logddfb2b4b33e07bee97dd3db77b8b544d151c4730ac8f6190116371104821c03d;
first commit source7e9f1eb0/testbb202d86/previous-reader0ff7d015 preserved unchanged.
Source-only correction3809d162 reads the actual V1 input/key/raw, compares exact
SameInput and retains acceptance-unknown. Conflict record uses that actual common
input key for V1 and V2. Unchanged expanded/previous/partial/independent replay
tests PASS1.901s, logd21de578f8a4ef7b5fe80596c074daa4e68e847351d1d914afe111de5d00fcf7.

Additional storage-corruption test stages the new input before encountering a
corrupt immutable conflict occupant: rollback leaves every original row and all
events unchanged, reports no committed conflict, and exposes no payload. A prior
partial declaration with accepted metadata and ZERO FA still blocks revision
replacement. Full conflict suite PASS.683s, log
ef0023eace0d12c1c910580d84d7dc6d18146702fbd23b3f82ad45e6c1f227e5.

Current commit source88ea5dc5c95258313ea991db0ba5c617460ae9b631bd27001512387af8fc3730;
testd33f6e1273c3bff5e1632cc752ae6aeeb8d6822286b06ad4f5d661e719182b6f;
previous-reader3809d162a6f1acd7fc3f026bc64d615e535655c559589f51a3a26edc1d96d16b.
Full uncached no-CGO suite PASS store74.101s/resolve18.783s/daemon28.749s;
final log2b932280ccd97da346545722661137c452425876865bf3e6d9bc50bb275ce8a1; vet PASS.

Next is actual normal materialization/result selection and fixed final inventory
coverage. Private MATERIALIZATION_IMPLEMENTATION_NOTES.md identifies the concrete
ledger constraint: existing identity accounting permits only tracked EN/AL deltas,
so old immediate IG writes cannot bypass it. A composed finalizer must separately
account/prove exact supported lifecycle writes and final complete inventories;
HasEpisode/finishSupported is not support authority. No finalizer or policy can be
selected by callers. Full normal Apply, all-writer/schema floor, actual replica
conservation, independent whole-program review and all original live bars remain.
