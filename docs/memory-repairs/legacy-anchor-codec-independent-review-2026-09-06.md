# Exact legacy anchor codec independent review

2026-09-06. Bounded PASS for the private, uncalled pure encoding unit at the hashes below: independent disproof found no violation of ANCHOR_CODEC_CONTRACT.md. This is not approval for legacy adoption, store integration, identity recognition, lifecycle mutation, rollout, or any whole-goal clause.

I ran session orientation, read the complete active goal objective, complete archived Controller V3 independent review (SHA-256 `2521939a134271df7669e073ee836283dd9cbaecc0339fc4df817225f587de10`), complete codec contract, exact source and supplied tests. I exported baseline `fd464c22fd8db22d31edce8bf09457ec2d10f726` into `/tmp/scry-anchor-codec-disproof-sep06.amW4kV`, copied the pinned candidate unit through apply_patch, and authored independent tests and this report only in that export. Shared/root/candidate sources were unchanged. No live graph access, providers, sweeps, configuration, remember, room posts, deployment, or external writes occurred.

## Evidence

The five independent test groups cover:

- Complete Entity and anchor fields: nulls, duplicate fields, missing required fields, case variants, unknown fields, invalid UTF-8, lone UTF-16 surrogate escapes, malformed/empty/nil JSON, trailing values and whitespace, and exact source/anchor key mismatches. Entity aliases/repo_refs retain their existing omitempty rules; explicit null or empty-array source encodings refuse when canonical Entity encoding would omit them. Valid unknown-schema extension is intentionally refused under this strict source contract.
- Exact identity semantics: a legitimate Name that does not slugify to the existing slug is accepted. Type, description, aliases, repository references and LastSeen changes pass matching; changing slug, exact Name or creation instant refuses. Equal creation instants in different minute-offset zones pass. Source offsets are retained in the raw fingerprint while newly encoded anchors use UTC.
- Timestamp boundaries: both signed UnixNano endpoints and Unix epoch round-trip; zero creation, one nanosecond outside either bound, redundant/lost fractional precision, comma fractions, leap-second notation and invalid offset encoding refuse. The supplied test also covers direct anchor encoding from second-offset time zones without losing the instant.
- Digest and ownership: independently constructed unsigned-64 big-endian length framing covers both exact key and value before SHA-256. Partitioned concatenations differ. Inventory and source-digest shapes refuse malformed/uppercase/nonhex/wrong-length values. Mutating caller source/anchor/record buffers does not alter returned encodings or decoded values. Errors are the same static sentinel; refusals preserve supplied bytes.
- Full proposed consumption records: preserve exact original canonical anchor bytes and key, explicit operation digest, reason and successor. Wrong/missing fields, duplicate/unknown/null fields, invalid Unicode, key/body mismatch, wrong versions, malformed operation digests, unknown reasons, same-identity merge targets, absent merge targets, invalid target slugs, retirement with a target, and invalid embedded anchors refuse. Merge and retire can describe different proposals at the same stable consumed address; there is no writer here to choose or replace a stored record.
- Unicode acceptance and rejection: valid multilingual, supplementary-plane, combining, replacement, control and escaped HTML/separator characters round-trip without identity normalization. Every standalone byte 0x80 through 0xff is rejected in each source name/type/description/alias/repository-reference location.

The five supplied test groups also pass. Static call-site inspection found the new private functions only inside their own production source file. There is no Store call, normal-write integration or externally callable adoption path in this unit.

Executed in the independent export:

`CGO_ENABLED=0 go test ./internal/memory/store -run 'TestIndependentAnchor|TestIndependentConsumption|TestLegacyAnchor|TestLegacyConsumption' -count=1`

PASS, including all five new independent groups and five supplied groups. No failing regression was removed or weakened; none was observed.

`CGO_ENABLED=0 go test ./... -count=1`

PASS against the final independent test revision. The earlier full-suite run also passed before the final test-only Unicode extension; the final run is the evidence for the pinned independent test file.

## Limits and next boundary

The stable identity comparison intentionally ignores mutable metadata and RawHash. It is not an alias/type policy, an inventory activity check, a consumption check or owner authorization. The codecs validate syntax and exact canonical records, not whether a supplied operation/inventory/digest represents an approved action. Proposed consumption encoding does not persist an immutable tombstone, prevent overwrite, remove an anchor or enforce lifecycle transitions.

No store reader exists here; consumed-anchor refusal forever is unimplemented and cannot be certified. Atomic reviewed lifecycle changes, complete inventory manifests, adoption markers, existing alias claims, all-writer routing and the old-writer interval remain separate integration work under the archived Controller V3 conditions. No actual snapshot was restored or measured by this reviewer. Root's separate sizing and actual-replica measurements are not independent evidence in this report and are not needed for this pure codec verdict. ANCHOR_DESIGN.md was not graded as implementation.

## Exact pins

| Artifact | SHA-256 |
| --- | --- |
| ANCHOR_CODEC_CONTRACT.md | `20591a73ac9d3a323d63c291fb517cd6f5596fd4f27beeef9b3694c9589c8f94` |
| internal/memory/store/identity_legacy_anchor.go | `7a84b00d46977c2d8ce97c5b29f35763fa347bcd61f967d7178ef42a399a7de7` |
| internal/memory/store/identity_legacy_anchor_test.go | `2558b0a46b4f899244ab40f007df545df592bd0cde3041819ca48c939183bb0a` |
| internal/memory/store/identity_legacy_anchor_independent_test.go | `64fad216ee296a3047349ea3156f9b768e67c8cb6d8dfa14674c0c0cd6df2a41` |

All relative paths refer to the independent export. The final report hash is supplied separately to avoid self-hashing.
