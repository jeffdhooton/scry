# Private immutable parsed input revisions

Baseline8fda77e, 2026-09-06. Implements the first unit conditionally permitted by
current episode design reviewa9722fcf, plus pure complete-observation matching.
Uncalled, private; no current result/head, support, birth registration, declaration
completion, graph mutation, transcript retention or production routing.

Version1 input contains full EpisodeID, exact nonzero nanosecond-representable
OccurredAt (normalized to UTC), Cwd, episode Summary, and ordered explicit observed
Declarations and Facts. Preserve every field, TypeFallback, alias order/duplicates,
nil versus empty slices and nil versus present Supersedes. ValidFrom stays unparsed.
Finite confidence is preserved; invalid Unicode or lossy JSON encoding refuses.
No lexical judgments, field truncation, model calls, source text or transcript spans.

Canonical JSON must roundtrip exactly. Full key io-input:<SHA256 episode ID>:<SHA256
canonical bytes> and full bytes are both validated, never digest-only authority.
Decoding refuses unknown/duplicate/missing/noncanonical fields, trailing bytes,
wrong episode/key and lossy representation. Encoder returns owned bytes; decoded
slices/pointers are independently owned. Caller input is never modified.

Pure matchRevisionObservation takes exact canonical revision key/raw and observation
key/raw. Reconstruct the complete expected observation from revision origin/ordinal/
side, full declaration or fact (including Supersedes), cwd, episode and instant;
encode it and compare exact key AND bytes. Reject out-of-range and mismatched
payloads even when episode and ordinal agree. This does not check role/spelling,
outcome coverage, registration inventory or stored provenance; those are additional
result/finalizer obligations. Observation codec remains unchanged.

Private immutable writer requires a live AtomicWrite facade and exact raw episode
ID/instant proof through the reviewed observation provenance validator. It writes
only its own new io-input: row, with owned buffers; exact duplicate is a no-op;
occupied different bytes refuse. No event, EP or graph write. Any error returns
only a static sentinel plus allowlisted ErrTxnTooBig/ErrConflict, and poisons an
admission-owned facade even if swallowed. Ordinary AtomicWrite callers must return
errors (existing owner contract); no new owner policy is implied.

Read/list/chunk operations use one coherent transaction view, validate full rows
and raw episode provenance, and reject closed/nil/zero facades. Root reads are
allowed. Exact existing cursor must validate for the same full episode; count1..100,
JSON byte budget512..24576. Key pages never skip an oversized/corrupt row and return
no partial data on corruption. Chunk envelopes include full key, exact byte offset,
base64 data and NextOffset (-1 at end); complete JSON fits the requested budget.
Offsets at exact end return empty complete data; past-end/negative offsets refuse.
Reassembly is pinned to an immutable key, not a moving current head. This budget
does NOT certify a future CLI/RPC envelope or p95. Large episode IDs are hashed in
keys, not truncated in the stored record. No arbitrary error/key/value content
escapes refusals. No partial writes on ordinary propagated error, owner-poisoned
error, panic or actual commit failure.

This private unit also adds io-input: to the adopter's reserved-prefix refusal;
complete production lifecycle protection remains NO-GO.
Tests must cover complete fidelity, mixed revisions, nil/empty/UTC/Unicode/time,
strict corruption, owner and ordinary scopes, occupied rows, raw EP ambiguity,
pagination/reassembly, owned buffers, actual late storage error privacy, rollback,
panic and persistent reopen/backup restore without changing old raw graph rows.
