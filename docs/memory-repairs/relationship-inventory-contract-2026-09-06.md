# Private complete identity relationship inventory

Baseline8fda77e, 2026-09-06. Read-only, uncalled input to the eventual fixed
finalizer. Not a recognized-identity validator, ownership grant, baseline-defect
exception, undo plan, support proof or full write attribution ledger.

Capture one coherent transaction view of ALL raw records in these exact prefixes:
al:, ar:, en:, ig:, il:, il-consumed:, meta:identity_, rs:, rt:. Retain exact owned
full keys and value bytes, including present-empty values and malformed/opaque
alias/control rows. Absence is map membership=false, never inferred from length.
Do not parse aliases or controls as valid authority. No other families are read
into this inventory; att:/iga:/ep:/fa: evidence, queued input and current results
remain separate proofs. This inventory does not assert complete support inputs.

Every en: row must pass the strict legacy entity decoder. Unknown/lossy/malformed
entity data cannot yield a trustworthy listing projection, so refuse globally
with a static error and no partial result. Do not rewrite it. Legitimate Name/Slug
divergence and duplicate/nil aliases retain that decoder's established behavior.
Return independent parsed Entity values indexed by exact slug.

Project every canonical Name and every alias occurrence, without deduplicating,
as {Slug,Kind:name|alias,Ordinal:-1 for Name or original alias index,Spelling}.
Listings groups by Normalize(exact spelling); NaturalListings separately groups
the SAME occurrences by Slugify(exact spelling). Preserve empty normalized groups
as observations, not grants. Each group is ordered by source en: key then Name,
then alias order. Entity.Slug lookup is independently present in Entities; do not
pretend canonical slug routing is a name/alias listing. These finite projections
do not enumerate arbitrary possible unlisted resolver input spellings.

IndexTargets groups every raw al: key under its complete raw owner value converted
to a string, even invalid UTF-8, missing targets and malformed index keys. Each
group's full keys are byte-lexicographically ordered. This catches unlisted and
untouched keys pointing at a removed identity. A map key here is raw data, NEVER
validated ownership. Full Rows allow exact listing/owner/selector/consumption and
negative-marker comparison, including absence, in a later closure implementation.

Return a count for each fixed family (including zeros), total row count and SHA256
of literal identity-relationship-inventory-v1 followed by NUL, then all selected
records in global byte-lexicographic key order, each full key and value separately
prefixed by unsigned64 big-endian byte length. This is measurement, not digest-only
semantic proof. Retain the bounded identity/control raw map deliberately so future
closure can compare exact bytes; no complete fact graph or transcripts are copied.

All maps, nested arrays and byte slices are owned, with no aliases between raw
Rows and parsed projection data or between independent capture calls. Capture
performs no writes/events and exposes no raw data through errors. Nil/zero/closed
facades refuse. Live AtomicWrite read-your-writes and root read-only capture work;
same private nonconcurrent-facade assumption as other primitives. Raw concurrent
phantoms, transient overwritten changes, all-writer coverage, a finalizer-owned
baseline, authorization and post-undo closure are NOT certified by this scan.

Tests independently frame the raw digest and assert all selected/excluded keys,
duplicate canonical/alias occurrences, malformed alias/control preservation,
empty/missing owners, unlisted/untouched dangling transitions, lifecycle changes,
natural/name distinction, whole raw equality and owned maps, canonical entity
refusal, nil/closed/transaction visibility. Restored real measurements must report
only counts/hashes/time/bytes, never entity facts or raw identity/control values.
