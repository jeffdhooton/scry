# Private exact inventory adoption and legacy recognition

Baseline315fa2a. This unit is uncalled, with no CLI, startup adoption, deployment,
semantic cleanup, public lifecycle writer or production admission integration.

A manifest is canonical JSON version1 and entries (non-nil ordered array, empty
allowed), each containing exact en: key and complete original value as base64
bytes. Entries are strictly key-sorted, unique and pass the reviewed legacy source
codec. Reject lossy/noncanonical/unknown/duplicate JSON and invalid source bytes.
The inventory ID is SHA256 of the complete canonical manifest bytes. Return owned
buffers. Capture reads one snapshot and never edits the graph; capture is not
semantic approval. An empty store still requires an explicit manifest/action.

Private adoptLegacyInventory accepts a ROOT Store and exact manifest bytes,
defaults to preview via apply=false. Refuse all transactional facades before
locks or IO; poison an enclosing scope so catching refusal cannot commit it.
Under the root exclusive maintenance lock, one Badger write transaction must
compare the complete current en: set with the manifest, including additions,
removals and every metadata byte. Require all reserved prefixes empty: il:,
il-consumed:, ig:, iga:, io:, io-result:, meta:identity_. Even malformed occupied
rows refuse. No implicit replay: an already adopted store refuses both modes.

Stage one exact named anchor per entry and canonical meta:identity_adoption_v1
marker {version:1,inventory:<id>,entities:<count>}. Verify every staged anchor and
marker. Only these absent keys may be written. Preview actually stages the entire
transaction, then deliberately rolls back: it exercises real transaction/value
limits rather than only estimating sizes. Apply commits once; any preflight,
staging, verification or commit failure returns no report and no candidate writes.
There are zero graph observer events. No DB limit changes or silent batching.
Reports contain only inventory ID, count, proposed key/value bytes, applied flag.
All errors are static; preserve only static ErrTxnTooBig/ErrConflict classes.

Current public graph/queue writers already take the root maintenance read lock;
reviewed merge/retire/unalias/restore take its exclusive lock. Adoption uses that
existing protocol, not Badger phantom detection. Direct private db/txn access,
concurrent Close, and an independently constructed facade bypassing that lock are
outside this unit's coordination contract; they must not be production producers.
Do not infer that a same-snapshot prefix rescan excludes arbitrary raw writers.
No admission policy after the adoption transaction is implemented here: existing
ordinary writers remain unaware, which prohibits live adoption of this unit.

readActiveLegacyIdentity reads one view (read-your-writes on a transaction) and
requires a complete canonical active marker, canonical matching-inventory anchor,
current strict canonical en: record with identical stable name/slug/time, and
absence of BOTH il-consumed:<slug> and ig:<slug>. ANY consumed bytes, even corrupt,
refuse; deleting/recreating en: cannot reactivate a retained consumed anchor.
Mutable metadata alone remains accepted; alias/type guards are separate. Missing
or malformed controls refuse with no partial return. This is legacy recognition,
not generation recognition, alias ownership, actual support or a whole-inventory
integrity audit. It does not certify an arbitrary raw writer cannot forge/delete
controls. No consumer/tombstone persistence API, unconsume or rebind exists here.

Tests must prove canonical manifests, exact drift/extra/missing refusal, all
occupied reserved families, non-writing complete previews, complete successful
apply, immutable original rows and zero events, real late size-failure rollback,
facade refusal/poisoning, concurrent ordinary-writer serialization, marker/anchor
corruption and consumed/recreated identity refusal. Complete real backup replica
measurement and independent review precede any possible shared integration; live
adoption additionally requires complete all-writer lifecycle/controller work.
