# Complete raw reference inventory, private uncalled extension

Baseline315fa2a. Complete the ControllerV3 baseline scan requirement without
changing the already reviewed requested-slug checker or strict raw decoder.
scanIdentityReferenceInventory reads all fa: in one view, read-your-writes on a
live transactional Store facade. Reject nil/closed facades using the existing
generation transaction guard. Decode EVERY row with decodeIdentityReference;
unrelated malformed records refuse the entire report without partial results.

Return Scanned row count, a newly owned map of every distinct actual source and
nonempty destination slug with separate current/historical counts, and SHA256
of a version-domain prefix plus every exact ordered raw fact key and value.
Count self-loops once; attribute values are never endpoints. A historical fact
counts even with no current counterpart/entity/adjacency/provenance closure.
The digest begins with literal bytes identity-reference-inventory-v1 followed
by NUL. Each key and each raw value is prefixed separately with its length as
unsigned64 big-endian. Iterator exact byte order determines record order.
Changes to any retained fact text, provenance, confidence, unknown extension,
JSON whitespace or endpoint must change the digest, even if counts don't change.

The decoder's losslessness/ambiguity/key rules remain unchanged, including
allowing opaque well-formed unknown extensions. Retain no full Fact slice or raw
key/value map after return. Zero DB writes/events; any error returns zero report.
Only existing redacted decoder errors or static sentinel leave this function.
Returned maps from separate calls never alias. No output prints slugs/raw data.

This is an exact point-in-time count/digest inventory, not ownership, provenance,
mutation attribution, support selection, adoption or permission to create an old
dangling endpoint. A zero baseline count is necessary but not sufficient for a
new birth. Final scans must share the admission transaction and be combined with
the not-yet-implemented complete fact-writer attribution and dependency planner.
No arbitrary concurrent raw-writer phantom resistance or p95 claim.

Test all endpoint kinds, original checker parity, independent digest framing,
metadata/unknown-only differences with unchanged counts, empty/non-FA controls,
malformed unrelated rows/no partial returns, raw byte equality/events, owned
maps, same-transaction staged insert/invalidation/delete and rollback, closed
facades. Measure one restored actual snapshot separately, never print its data.
