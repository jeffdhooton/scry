# Independent immediate predeployment source extension

Verdict: **PASS for the bounded code-only 24eafab deployment gate against the two 00:42:09 UTC backups.** The complete source delta does not prove a new occupied-assertion overwrite, provenance loss, abandoned queued item, marker mutation, existing alias-index transfer, or candidate startup mutation. The prior exact-artifact conditional PASS stands with its actual deployment and postdeployment requirements. This is not a semantic endorsement of all intervening extractions, a live identity-repair gate, or an actual deployment PASS.

Private evidence root remains `/tmp/scry-integrated-predeploy-independent.2D6GWQ`. Only private restores, diagnostics and reports were written. The original REPORT.md and its helper/test source were not changed. Credential-bearing fact values, sentences, keys, derived spellings and source projections remain suppressed. The extension reports hashes, field names, counts and equality booleans.

## Independently verified new inputs and startup preservation

Shared input `/tmp/scry-fact-guard-deploy-sep06.uN9zYJ/memory-20260906T004209Z.badger`: 74,318,735 bytes, independently verified SHA-256 `9f722b1dafd4e0c20984446f79ff02aba9adda0c843f6e8ad80918733ecbb365`.

Laptop input `/Users/jeff/.scry/backups/memory-20260906T004209Z.badger`: 19,445,008 bytes, independently verified SHA-256 `d4e86e0da75439e801c9698cf670cce1ec6e06a92f234b380e8011679f307ac7`.

Both inputs were independently loaded directly with Badger into NEW private directories before candidate schema handling. Full raw key/value captures before and after candidate Open, AllFacts, Entities, offline index construction and an actual recall read are exactly equal. No prefix is filtered and missing keys differ from present empty values.

Shared complete raw SHA is `6e1e10e8a95afd6a5ca63206a0645de188303c9738b1e79014b8e74245a0cffe`: 244,498 keys, 81,057 facts (73,212 current / 7,845 historical), 30,838 entities, 9,388 episodes, 52,258 claims, 55,913 adjacency keys, 10,985 attestations, 3,143 cursors, 857 value-evidence records, five metadata keys, twelve pending records, four alias rejections, nineteen retired-slug and nineteen retirement records.

Laptop remains exactly equal to the prior complete 83,378-key map, raw SHA `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`. The startup/read controls change no bytes on either replica. Those isolated reads are 11,592 and 9,409 bytes respectively; this is not a replacement for the five-suite live comparison.

## Complete shared source drift, 00:30:16 to 00:42:09

The independent harness compares the complete prior map at SHA `4e33ccf0df9f7e35e20d444c4d56f2eda64ef66296594ded88c9f5f3cb3d9802` to the complete fresh map. Counts agree with the root's separately generated delta:

| Family | Added | Changed | Removed |
|---|---:|---:|---:|
| Facts | 91 | 4 | 0 |
| Entities | 55 | 14 | 0 |
| Episodes | 6 | 0 | 0 |
| Alias claims | 69 | 0 | 0 |
| Attestations | 24 | 0 | 0 |
| Adjacency | 63 | 0 | 0 |
| Value evidence | 44 | 5 | 0 |
| Metadata | 0 | 3 | 0 |
| Pending | 0 | 0 | 5 |

All other families are unchanged. Every changed record is retained in a hash-only delta with explicit before/after existence booleans and exact payload digests.

For all four changed preexisting facts, the complete serialized field comparison permits only `invalid_at` or `episodes`:

- Three facts gain InvalidAt, from nil to a time not before ValidFrom. Their source, relation, destination/value, sentence, raw relation, validity start, confidence and all original episode provenance are unchanged. This is preservation of the prior assertion as history; semantic justification for each invalidation was not freshly judged.
- One fact appends a second provenance episode while retaining the original. No other field changes, and the appended ID names one of the six new stored episodes.

No fact key is removed, and no old assertion text/value/identity/start instant is replaced. All 91 new facts have nonempty provenance, cite at least one of the six new stored episodes, and every referenced provenance ID exists. Their source and any entity destination exist in the fresh map. This provides structural/source closure, not line-by-line semantic verification of all new facts.

All five removed pending rows were nonparked. Each has exactly one corresponding newly completed episode with the same ID. Source, SourceRef, OccurredAt, Cwd and CwdIsRepo compare equal to the pending input. Exact payload hashes and equality booleans for all five are retained. The twelve remaining pending records are the same twelve parked records, each preserving its complete old payload byte-for-byte. No parked record disappears, changes attempts/errors/text/provenance, or is silently retried by the measured delta.

The sixth new episode was not present in the prior snapshot's queue. Its source and source_ref are both `manual`, independently checked as a boolean with only hashes in diagnostic output; raw episode SHA is `4fc079fdf71335bc3cb8b958754292ba35f3c2aa9d3cf2b33fde7df85f55dcee`. This is consistent with a new manual input arriving and completing between snapshots. The two snapshot endpoints alone do not prove its unseen enqueue transition, and no such claim is made. It does not leave any prior pending item unexplained.

All 42 protected marker keys and values—four `ar:`, nineteen `rs:` and nineteen `rt:`—are exact. No old alias claim changed owner or disappeared. No old entity, listed alias or repository reference was removed.

The entity drift is **not all last_seen-only**: ten existing entities change only last_seen; three append aliases plus last_seen; one changes type, appends repository references and changes last_seen. The type-change key SHA is `cfab2359760cc32732e0db6fbc64653407deb1041ff448df56e1edd48fc0ca71`; its old type was concept and it had one current touching fact. Its old type hash is `da5e11efa36720a4211ac89acf1479952e99b35636f006a70bcede07495289d6`; new type hash is `86ae35d58a6aa3b5742df94ef9d7162219f0106a911ae1954c1f0604aaec805d`. The complete fact remains preserved, and no old identity or alias claim is merged/deleted/transferred in this delta. The type refinement and appended aliases are not independently semantically endorsed. They occurred in the predeployment source and are not introduced by candidate startup; they do not invalidate the prospective occupied-key/tie-only code gate. A stricter initial diagnostic that assumed existing entity changes were only timestamps/aliases/repo refs refused on this type field; the final diagnostic reports the full field difference instead of concealing it.

## Retained binaries and remaining actual deployment gate

Independent read-only local and SSH hash checks verify:

- `/Users/jeff/go/bin/scry.pre-24eafab-20260906T0048Z` and `/Users/jclaw/.local/bin/scry.pre-24eafab-20260906T0048Z` both retain exact prior SHA `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.
- Both installed executable paths still have that old hash at this check.
- Mini staged `/Users/jclaw/.local/bin/scry.candidate-24eafab-20260906T0048Z` has exact candidate SHA `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`.

The root separately reports its immediate live before suites remain 51/62, 29/66, 7/7, 45/50 and 47/50 with unchanged full miss/rank/mean-rank measurements, max 13,359 bytes, cap violations zero; its 00:46 status still reports 9,388 episodes / 81,057 facts / 30,838 entities and 0 ready / 0 backoff / 12 parked. Those are root-supplied live measurements, not independently rerun suite results in this extension.

The root may proceed with only the previously pinned executable replacement and two launchd restarts, then must independently prove actual installed bytes/new processes, immediate complete post-backup restoration/delta preservation, expected service roles, live suites and later actual sweeps. No blanket source cleanliness, history recovery, credential remediation, identity-manifest apply, benchmark-floor restoration or whole-goal completion is granted. The previous report's rollback restrictions and all outstanding source/recall defects still apply.

## Reproduction

Use the unchanged `integrated-grade restore SOURCE NEW_DIRECTORY SHA256` for each input. Build the added independent helper with `CGO_ENABLED=0 go build -o ../freshness-grade ./cmd/freshness-grade` from private `code/`. Run `../freshness-grade ../shared ../shared-004209` for the full shared delta and `../freshness-grade ../laptop ../laptop-004209` for laptop exact equality. The helper emits only hashes, field names, counts and booleans; it never emits complete fact or source values.

Evidence: `shared-004209-restore.json`, `laptop-004209-restore.json`, `freshness-004209-delta.json`, `freshness-004209-laptop-delta.json`, and `code/cmd/freshness-grade/main.go`. These are distinct from the original review's pinned artifacts.

| Artifact | SHA-256 |
|---|---|
| shared-004209-restore.json | `c9c719d29f38047ebbed0a807d52bf80c04a247e9646925c12b991f0de560853` |
| laptop-004209-restore.json | `807f366da5e2aad3af9c05eddeb84e5a9ace42e36a3826718bf717635564b96e` |
| freshness-004209-delta.json | `75afb42ad097776aa062644427c8a3a77396acb71b99baff2904efc2642d562e` |
| freshness-004209-laptop-delta.json | `6d0351d1bed4fdf15582b1bfa11333a670bc700b4361dbcd0c56411800fbd83f` |
| cmd/freshness-grade/main.go | `dceee8b732a37f4fa448ce3c40115ae0f4af14feaab098964f00750fff635fe5` |
