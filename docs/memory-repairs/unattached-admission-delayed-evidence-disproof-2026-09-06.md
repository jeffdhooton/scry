# Delayed attestation contamination: correction to the design verdict

2026-09-06. This addendum corrects invariant 1 and narrows the GO in REPORT.md
(SHA-256 `5fa07e0de58e76514bdbd9dc3014f95ffbdca7be1d0b6c62b8a19ca0abb4769a`).
The initial recommendation was insufficient: ignoring orphan attestation
authority only while en: was absent at transaction start does not prevent
that authority from returning after a supported identity commits.

**Revised verdict: GO for a journal-only primitive without admission behavior.
NO-GO for admission prevention based on journal plus transaction-local early
isolation alone.** A persistent attestation authority boundary is necessary.

## Independent runnable counterexamples

Private source is still the unmodified production a06cd7b export. The only
added file is internal/memory/resolve/delayed_attestation_disproof_test.go.

```sh
go test ./internal/memory/resolve -run 'TestDisproofSupportedAdmissionReactivatesOldAttestationNextEpisode|TestDisproofExcludedLegacyEpisodesStillConsumeAttestationCapacity' -count=1 -v
```

Both tests PASS by reproducing the failures. Only synthetic fixtures are used.
Raw comparisons use full Store.Backup followed by direct Badger Load into a
new private directory and exact ValueCopy comparison of the synthetic att:
key. No new production helper or production implementation was introduced.

The first test uses three episode identities:

1. An old stored episode and orphan attestation propose one unrelated alias
   for a slug that has no entity. This is preserved baseline evidence, not an
   authorized owner or an entity inferred from the alias.
2. A new episode creates an actual runbook identity with a legitimate own-name
   fact. It proposes no aliases. Therefore the proposed early isolation rule
   has no alias invocation to alter. The identity is supported and remains en:.
   The entire original attestation value is byte-identical after this commit.
3. A later episode proposes the old alias on the now-established runbook. The
   existing-entity admission path combines the old episode ID with this new
   episode ID, admits the alias and routes an actual new assertion through it.
   The first assertion is fully equal to its original stored Fact; the old
   attestation episode ID remains present. Preservation alone did not prevent
   unauthorized authority from being reused.

This is also a counterexample to the proposed invariant without implementing
it: step 2 has no alias admission to isolate, and step 3 explicitly takes the
unchanged established-entity branch that the invariant permitted. A hollow
finalizer cannot intervene at either step because the identity has real facts.

The second test saturates a legacy attestation record with eight old episode
IDs, then calls AttestAlias for two fresh episodes. The whole raw record stays
byte-identical: neither fresh ID is retained. A persistent exclusion list that
subtracts old IDs only when reading therefore leaves zero usable new episodes
forever for this saturated key. It is not enough to add a marker and subtract
an old count while retaining the same capped evidence writer.

## Source boundary

`internal/memory/resolve/resolve.go:296` updates an existing entity through the
same admitAliases function as a new declaration. No birth-generation state is
consulted. `internal/memory/resolve/aliases.go:400` calls AttestAlias and admits
the unowned alias when the returned count reaches the threshold.

`internal/memory/store/pending.go:244` AttestAlias identifies evidence by
att:<slug>:<normalized alias>. At `:268` it recognizes duplicate episode IDs;
at `:273` it appends only below maxAttestations (8). Neither the key nor the
value binds those episode IDs to the current entity incarnation. There is no
timestamp in that evidence record from which a safe ownership cutoff could be
derived. Entity CreatedAt is episode time, can be historical, and is not a
transactional birth generation; timestamp comparisons cannot supply the
missing binding.

## Smallest persistent isolation that closes the demonstrated cases

For identities first admitted under the new policy, commit a durable marker
that selects an immutable admission generation, and keep a separate set of
generation-bound alias attestations. This requires both pieces: a persistent
selection rule and a place to preserve fresh evidence independently of the
legacy capped list. The names and on-disk representation remain unapproved.

- Allocate the generation before any alias decision for an entity absent at
  transaction start. Bind it to an exact creation operation, not similarity,
  type compatibility, slug alone, a mutable entity hash, or wall-clock time.
  A deterministic full creation identity must still compare complete bytes
  on key collision. Store the selected generation atomically with supported
  en:, its own evidence, facts and episode marker.
- Every later attestation read and write for that entity follows its durable
  generation, even though en: now exists. It must never fall back to legacy
  att: episode counts because the entity has become established. Missing,
  corrupt or inconsistent required generation state is an explicit refusal,
  not a permission to use old evidence.
- For these newly admitted identities, preserve all preexisting legacy att:
  bytes at their exact old keys. Do not append to, delete, rewrite or silently
  relocate those records. They remain visible unresolved legacy evidence
  without authority for the new generation. New independent attestations go
  into their generation's own records; its cap cannot be consumed by the old
  episodes. This avoids both the delayed threshold contamination and the
  saturated-record starvation without a scan-scale ownership decision.
- An unsupported provisional generation is kept as non-routing structured
  observation evidence. A later assertion starts its own generation; it does
  not inherit the observation's threshold counts. A supported generation's
  marker continues to exist across ordinary metadata updates and Force. Any
  reviewed removal/recreation or merge must explicitly preserve/dispose of
  the generation binding, so a removed identity's generation cannot silently
  be rebound to a new identity at the same slug.
- The legacy semantics exception applies only to identities that predate
  admission-policy adoption and have no new-generation marker. It must not
  be a bypass for new ordinary writers: all supported creation paths must
  atomically install the marker, with final validation before commit. This
  does not migrate, delete or reinterpret the existing hollow graph.

An equivalent design could maintain a permanent exclusion boundary plus a
separate authoritative fresh-evidence ledger. That is not smaller in the
tested important sense: it still needs persistent binding and independent
evidence storage, and keeping the old writer for the shared capped array
fails. A marker alone, a transaction journal alone, restoring old bytes, or
checking en: absence only once cannot implement this boundary.

This design intentionally changes how **newly created** identities can use
evidence, which is an admission decision based on exact transaction history.
It makes no retrospective claim that the old alias was false, belonged to
someone else, or should be removed. Preserve and expose the old record as
unresolved; do not report it as repaired just because it cannot route the
new identity. Existing legacy hollows and claims still need their manifests.

## Limits and next proof

No generation format or prototype was built or graded here. The next
journal-only primitive is still independently useful and can preserve every
existing behavior/test. Any later admission prototype must prove persistent
isolation across supported birth, later alias proposals, duplicate/Force,
eight-entry old records, backup/reopen, unsupported observations, and reviewed
identity lifecycle operations before its contract can be accepted.

Retained old writers do not understand this marker or ledger. They may retain
all opaque bytes while modifying en:/al:/att: under the old policy, thereby
bypassing prevention or creating unmarked identities. An additive schema
does not grant behavioral compatibility. Running such a writer on the store
must remain an explicitly disclosed suspension of prevention; restarting a
modern writer cannot silently classify all intervening additions as reviewed
legacy identities and claim uninterrupted safety. That interval needs exact
reconciliation or a separate writer compatibility boundary. No schema bump,
live cleanup, config change or rollback policy is authorized by this review.

The report's original contract conflict with TestApply_EmptySlugSkipped also
remains. These two private tests do not modify that baseline assertion, invoke
providers, touch a live store, or certify the whole goal.
