Verdict: **the draft is sound as a preservation objective but is not yet an implementable migration specification.** The full assertion tuple can distinguish the known collision. It does not, by itself, make downgrade safe, preserve historical restatements, resolve supersession, or update every consumer correctly. The optional discriminator alternative has a smaller storage delta, provided it shares exactly the same resolver and compatibility safeguards.

I reviewed source at `bccb905aa6de8284ff531648bd67d7122d533ee4`. Draft SHA-256: `035adba51e490f05a49465e26680fbe8f49fa33e3cde058103e4dc589b1db55f`. Store source SHA-256: `600a299cf8177ebacafaa80cba2c3778a77545b5bf4f487ba972087e50cfd8a6`. Resolver source SHA-256: `67bd877107eb55ef32953d30f28c2f9c9a9178292dc4a3cc8f363ba678683348`.

This was a source/design disproof. I read the active objective, draft and supplied bounded review evidence. I did not read actual fact records, backups, credential-bearing source projections or pending inputs. No shared edits, live operations, provider calls, notes, recovery, deployment or queue changes occurred.

1. The writer-floor proposal needs a concrete mechanism before any key design is promoted.

   [store.go:264](/Users/jeff/workspace/context-stack/scry/internal/memory/store/store.go:264) invokes `DropAll` for a nonzero numeric schema mismatch. A normal numeric schema increment would therefore cause the current binary to erase the upgraded store on reopening. I independently ran:

   `CGO_ENABLED=0 go test ./internal/memory/store -run '^TestOpenCloseAndSchemaWipe$' -count=1 -v`

   It passed, proving the existing intentionally destructive behavior on fabricated test content.

   An additive `meta:min_writer` record would not protect against these binaries: their admission code does not read it. Deploying a bridge release that respects a floor protects bridge-and-later binaries, but does not retrospectively teach older retained binaries to refuse.

   One concrete avenue worth disproving is a deliberately versioned, non-integer encoding at the existing schema marker: old `schemaVersionOnDisk` attempts JSON decoding into `int` and returns before `DropAll` when decoding fails. A new reader could explicitly recognize the new object encoding. This is a proposed compatibility mechanism, not an approved implementation; it needs actual retained-binary tests and malformed/missing-marker tests. Merely adding a separate marker or incrementing the existing integer is insufficient.

   [store.go:1154](/Users/jeff/workspace/context-stack/scry/internal/memory/store/store.go:1154) introduces a separate restore hazard: `Restore` wipes its destination before loading or checking the restored schema. A startup refusal does not make restoring an incompatible or corrupt backup safe. Specify restoration into a fresh private destination, verification before activation, and an explicit preservation plan for the current destination.

   Also specify that all writer handles are stopped/closed during activation. A check only at `Open` does not constrain a handle opened before a marker changes. “No modification” should distinguish unchanged logical key/value records from incidental physical database-file changes during open/close.

2. Several concrete consumers need changes beyond their draft category labels.

   | Consumer | Proven dependency and required treatment |
   |---|---|
   | [DeleteFact, store.go:958](/Users/jeff/workspace/context-stack/scry/internal/memory/store/store.go:958) | The delete event constructs a four-field fact stub. It lacks sentence, literal, raw relation and any new discriminator. A full-identity search key cannot remove the old document from this event. Read and emit the exact deleted fact/address. |
   | [FactsAbout, store.go:828](/Users/jeff/workspace/context-stack/scry/internal/memory/store/store.go:828) | Deduplication reconstructs legacy keys; reverse lookup reconstructs forward keys from parsed adjacency text. Both must carry the exact address, including malformed/stale adjacency handling. |
   | [fallback.go:30](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/fallback.go:30) | `requireVacantFallbackKey` still rejects every same-triple/same-start sibling, even if the new store can preserve both. Changing only canonical matching leaves fallback assertions blocked. |
   | [memory_reattach.go:124](/Users/jeff/workspace/context-stack/scry/internal/daemon/memory_reattach.go:124) | Finds the first matching endpoint/value/time tuple, then checks the optional sentence. It cannot address siblings distinguished only by raw relation, and can reject the desired sibling because another was visited first. |
   | [memory_retire.go:122](/Users/jeff/workspace/context-stack/scry/internal/daemon/memory_retire.go:122) | Cross-group isolation builds its own four-field replacement key. It needs the same exact identity/address function as storage. |
   | [retire.go:697](/Users/jeff/workspace/context-stack/scry/internal/memory/store/retire.go:697) | Canonical adjacency maps, stale fallback parsing and reference detection each encode the old key. Updating writes alone could hide affected mirrors from review. |
   | [RelocateFact, store.go:1326](/Users/jeff/workspace/context-stack/scry/internal/memory/store/store.go:1326) | Still resolves occupied destinations by repeatedly changing `ValidFrom` by one nanosecond. Its migration/hygiene callers remain reachable. Replace this with an exact-address operation and explicit conflict refusal. |
   | [search/index.go:278](/Users/jeff/workspace/context-stack/scry/internal/memory/search/index.go:278), [recall/query.go:511](/Users/jeff/workspace/context-stack/scry/internal/memory/recall/query.go:511) | Search identity omits assertion text/raw relation; recall identity is constructed after clipping and also omits those fields. Carry a full identity token into hits before clipping, through sorting, injection and deduplication. |
   | [memory_methods.go:361](/Users/jeff/workspace/context-stack/scry/internal/daemon/memory_methods.go:361) | Public invalidation intentionally invalidates every current exact triple match, individually. Decide whether that remains an explicit bulk operation or becomes ambiguity-refusing; a lower-level key change alone cannot decide this contract. |

   Existing migration and hygiene invalidation calls also need exact references: [migrate.go:271](/Users/jeff/workspace/context-stack/scry/internal/memory/migrate/migrate.go:271), [hygiene.go:531](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/hygiene.go:531), [hygiene.go:961](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/hygiene.go:961).

   The public invalidate operation currently lacks one enclosing transaction across all matches. A new ambiguity check must run before any mutation; otherwise discovering a conflict after earlier invalidations would leave a partial operation.

3. Noncoalescing requires an explicit historical and temporal policy.

   [matchingFactForMerge, fallback.go:12](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/fallback.go:12) searches current facts only. Therefore changing its comparator to the full tuple is insufficient. An exact historical assertion can be missed, reconstructed with `InvalidAt=nil` and only the new episode, then accepted by `PutFact` as a same-assertion update. That reopens history and replaces provenance while satisfying the proposed identity equality.

   Match exact assertions across current **and historical** rows. For ordinary restatement, preserve existing validity, union provenance without losing entries, and adopt a declared confidence policy. The existing resolver uses maximum confidence; retaining that is a reasonable bounded default, but direct `PutFact` currently replaces metadata wholesale. Decide whether it remains an explicitly privileged replacement primitive or receives the same preservation checks.

   [resolve.go:553](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/resolve.go:553) performs Phase A merges and [resolve.go:587](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/resolve.go:587) repeats matching in Phase B only for fallback facts. New canonical duplicate assertions within a single episode need the same exact-restatement policy. Repeated facts with conflicting metadata or supersedes hints must not become slice-order-dependent.

   [resolve.go:616](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/resolve.go:616) defines exclusive-target equality through normalized `KeyDst`. Literal values such as fabricated `"stage_ready"` and `"stage-ready"` still compare as the same target after the proposed storage repair. Decide whether exclusivity compares full literals; storage identity must not silently answer that semantic question.

   Temporal tests must vary `ep.OccurredAt` independently of `ValidFrom`. Current exclusive invalidation uses the episode time, while historical insertion uses the assertion start. Test later-arriving older evidence, differing episode/start dates, equal starts, and already-invalidated exact repeats. Removing backdating does not establish these semantics automatically.

   Supersession needs both ambiguity and eligibility rules. [resolve.go:775](/Users/jeff/workspace/context-stack/scry/internal/memory/resolve/resolve.go:775) currently chooses a current triple match before applying the “started before episode” check. Multiple exact assertions make that choice unsafe. Define whether eligibility is filtered first, then require a unique candidate or an explicit assertion reference. Do not silently choose the first, latest or highest-confidence sentence.

4. Full tuple identity is not sufficient to preserve every distinct stored record during migration or merge.

   Two raw records can decode to the same proposed identity while differing in `InvalidAt`, confidence, provenance, unknown fields or JSON representation. Mapping both to one content-derived address would collapse them unless preflight refuses. A hash equality check against the full tuple does not solve this: the tuples actually agree.

   The same condition can arise after reviewed endpoint relocation: two formerly distinct assertions become identical under the new endpoint-inclusive identity. Safe minimal behavior is to refuse the group and require a separate reviewed disposition. Do not union metadata or drop a row as an undocumented merge side effect.

   Distinguish an assertion’s content fingerprint from a stable record identifier. Endpoint-derived content identity necessarily changes during graph repairs. If the discriminator is intended to survive relocation, it is a different concept and needs its own uniqueness and validation rules.

   Define absent versus empty fields, duplicate JSON properties, zero/out-of-range start dates and timestamp encoding. Existing `UnixNano()` key calculations are unsuitable as the universal representation for unrestricted dates. Preserve the original raw timestamp spelling while comparing a documented canonical instant.

5. “Raw fingerprints” are currently decoded-object fingerprints in important repair paths.

   [merge.go:815](/Users/jeff/workspace/context-stack/scry/internal/memory/store/merge.go:815) drops raw keys and unmarshals facts into structs. [merge.go:630](/Users/jeff/workspace/context-stack/scry/internal/memory/store/merge.go:630) fingerprints reserialized structs; [unalias.go:214](/Users/jeff/workspace/context-stack/scry/internal/memory/store/unalias.go:214) does likewise for touching facts.

   An unknown-field-only change can therefore leave a reviewed fingerprint unchanged. Subsequent rewritten facts can lose that unknown field. The draft correctly requires raw migration preservation, but that guarantee must extend to postmigration writes and repairs—or operations touching unsupported payloads must fail closed. Migrating raw bytes and then allowing an ordinary restatement to reserialize away unknown content is not end-to-end preservation.

   A migration manifest also needs a precise policy for malformed rows, unexpected keys, duplicate identities and stale/missing/nonempty adjacency mirrors. “Preserve existing defects visibly” does not say whether such rows are copied unchanged, block migration, or require explicit exceptions. Rebuilding adjacency solely from decoded facts would conceal defects and violate the stated delta.

6. The optional discriminator is the smaller storage transition, with qualifications.

   A bounded additive design can leave existing raw fact records and keys unchanged, use an explicit versioned discriminator for new sibling assertions, and expose one exact `FactRef`/address abstraction to every consumer. Before adding a record, compare the full assertion identity against both legacy and discriminated candidates; otherwise an exact restatement can create duplicate records simply because the formats differ.

   Empty versus nonempty discriminator must be a documented address version, not a caller-chosen loophole to bypass conflicts. Discriminator generation, verification and digest-collision refusal need one implementation. Legacy reads must retain their actual address; deriving an address from payload alone is not enough when the same tuple can appear in both formats.

   This avoids rewriting approximately the entire fact and adjacency population solely to recover one missing assertion. It does **not** eliminate the writer floor, all consumer changes, resolver semantics, unknown-field policy or post-shape grading. Full rekeying offers a simpler eventual single-format invariant, but its larger raw delta and activation/rollback burden are not yet justified by this bounded recovery need.

7. Smallest safe next implementation.

   First implement and independently grade a compatibility/prevention bridge that admits **no new fact format**: replace destructive schema mismatch behavior with explicit refusal in the new binary, concretely specify and fixture-prove legacy refusal, and add full-assertion restatement handling that never backdates, reopens history or loses provenance. Distinct assertions that cannot fit the current storage layout should continue to return the existing deterministic conflict and preserve the entire queued episode.

   That bridge can make progress without a store-wide migration or historical recovery. Before deploying resolver semantics, run the complete fresh replica and required recall controls; refusing more input is not an ingestion-success claim.

   Then implement the exact address abstraction and optional discriminator as a separately reviewed transition, with all consumers above and a compatible rollback binary retained. Only after actual compatible prevention is independently proved should the single-record recovery manifest be regenerated.

   Recovery should add the exact historical payload with its original `InvalidAt`, provenance and other metadata, and verify that every referenced episode exists with the reviewed identity/content. If provenance is missing or differs, adding more records is a new manifest scope; do not invent provenance or replay the source. Pin the current replacement and complete surrounding raw delta, make repeat application an exact no-op, and preserve every intervening write.

The historical assertion remains unrecovered. The occupied-key guard remains useful and should stay active. This review grants no migration, downgrade, live recovery, credential action or whole-goal PASS. The blocking design choices are concrete compatibility encoding/activation, exact historical metadata policy, supersession and exclusive-target semantics, and duplicate-identity/raw-payload handling.
