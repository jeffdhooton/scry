# Independent ChildScribe first alias-batch gate — 2026-09-05

Verdict: bounded replica PASS for the exact 49-row manifests below, using production code independently extracted by git archive from 53fafa91d621190c87245f0b0844270b4fdf44c9. Fresh postdeployment snapshot, exact live preview fingerprints, verified deployment and backup discipline, and the actual live application/postconditions remain pending. This is not a live-apply or full-goal PASS.

Session began with `scry memory orient --cwd .`; the entire goal contract was read. No live writes, deployment, remember, model calls, shared source edits, or mutations to the lead's replicas were performed. Independent helpers are in this archive's cmd/independent-alias, cmd/independent-drift, and cmd/independent-raw. Each restore used a new directory and refused an existing target.

## Exact inputs

| Snapshot | Source SHA-256 | Manifest SHA-256 |
|---|---|---|
| /tmp/scry-qwen-live-sep05.Koj9v3/memory-20260905T182812Z.badger | 24621f12ab33ad0d7b32f96238f1b7854430abf41da9c3ff8b362dd8c33ae851 | 6381c88618d0e891988796b2ae066bb7fe87316ada4ef0d146519e167f0c2d4c |
| /tmp/scry-unalias-deploy-sep05.Od8oCm/live-before-deploy.badger | d1e62fc142e348441ccf4c989070af1a906c1ad7ad8158fb65c272f34ac39aad | 24a765acc82e122c270e812e99993dbe3eff5d8261f88f05a0e065f16eb61ba6 |

The manifest files are respectively /tmp/scry-unalias-replica-sep05.rp5vdY/measurement/manifest-replica-only.json and /tmp/scry-unalias-replica-sep05.rp5vdY/refreshed/manifest-replica-only.json. Both source hashes were independently computed. Both manifests matched independent previews without regenerating or weakening expected inputs. Their identical drop plan hashes are 8ff742fbe570347c726efe1f3ce66bb3a9de63c9e151038324d0e1e538f2250f; differences are participant entity and touching-fact fingerprints.

## Independent measurements

| Measurement | Post-Qwen replica | Predeployment replica |
|---|---:|---:|
| Entities, unchanged | 30,175 | 30,224 |
| Full current + historical facts, exact equality | 79,560 | 79,679 |
| ChildScribe aliases | 92 → 43 | 92 → 43 |
| Cross-type collisions | 489 → 486 | 489 → 486 |
| Entire alias index | 51,434 → 51,394 | 51,500 → 51,460 |
| Entire raw database key count | 239,214 → 239,174 | 239,650 → 239,610 |
| Backup bytes independently restored | 76,886,364 | 77,087,455 |
| Entities with no current OR historical touching fact, unchanged | 2,854 | 2,854 |
| Distinct missing endpoint slugs across all facts, unchanged | 994 | 994 |

All 49 literal removals were independently compared with the source alias list. They cover exactly 45 normalized keys. Normalized duplicate groups are Laravel backend / laravel backend / laravel-backend; Laravel app / laravel app; Client Dashboard / client dashboard. No unreviewed normalized variant is removed. The exact retained 43 alias strings and their order equal the independently constructed remainder of the original list; therefore all 24 ambiguous rows, the five excluded target-metadata proposals, and 14 keep rows remain untouched.

Expected index changes were derived independently from the source map and reviewed rows: remove 40 ChildScribe-owned keys with no outside name/slug/alias listing, rehome exactly laravel-forge and forge-ssh-access to forge, preserve api-project/createapp/api-app as api-owned. A scan of every entity's name, slug and aliases found only those five outside listings for affected keys. The resulting ENTIRE index map equals that expected map. All 49 exact lookups were checked. No outside listing is orphaned by this batch.

Every entity field equals the original except the exact childscribe-laravel alias list. Every full fact equals the original, including endpoints, text, timestamps, invalidation, raw relation, confidence, values and episode provenance. A separate independent raw Badger reader proves all other keys and values unchanged as well: only 43 raw keys differ (40 alias deletions, two alias-owner replacements, one entity record). This covers episode, adjacency, cursor, marker and metadata keys in addition to API-level comparisons.

Each apply-generated backup was restored to its own new directory and compared against a separate fresh restore of the original source backup. The entire raw key/value map is identical, proving rollback at the logical database level; physical SST file bytes are not expected to match. Rollback backup hashes: postqwen 862cb3b72afa702896555b7eb0d8d56121b7eac6a9a4aa0ac195bc9c20a79b24; predeploy 6910671e31509e6a2082431859c31e71056ce49e53126058c78287e679c736b1. Neither the original manifest's second preview nor a new preview of the same rows without expected fingerprints is ready; both leave the full state unchanged. This is refused already-absent input, not a claim that broad hygiene converges.

## Semantic disproof

Reviewed the 92-row audit and its per-alias fact/candidate evidence, independently checked the selected spellings against those facts, read Forge's complete touching-fact evidence, and traced key supporting episodes. Existing index ownership alone was not treated as semantic authority. Explicit drops below remove unqualified roles/surfaces or demonstrably wrong identity labels without inventing a recipient. Row numbers identify the committed audit JSON; every selected row is covered.

| Rows | Selected spelling(s) | Independent reason |
|---|---|---|
| 1,11,24 | Laravel backend; laravel backend; laravel-backend | Battery-designer facts explicitly use this backend role for Cell Saviors. ChildScribe facts do not confer global ownership. |
| 3 | Laravel repo | Local checkout role; exact ChildScribe repository/path aliases survive. |
| 4,37 | Laravel app; laravel app | Advocates is also explicitly a Laravel application. |
| 5 | Lararel Forge | Unattested typo naming a deployment platform, not ChildScribe. No invented typo rehome. |
| 7 | childscribe.com (staging) | Domain/environment label contradicts documented staging child-scribe.com and production childscribe.com. |
| 12 | Laravel Forge | Advocates, Cell Saviors and ChildScribe are deployed on the separately established Forge service. Existing Forge listing and supporting platform facts justify restoration of this claim. |
| 13 | forge-ssh-access | Already explicitly listed by Forge, with Forge server/SSH access documentation and facts across projects; not an application identity. This restores an existing scoped access lookup to the deployment service. No literal fact attests the exact slug; the narrower service/access interpretation is a stated semantic inference. |
| 14,20,21,22 | prod web; staging Laravel; Laravel staging; prod Laravel | Unqualified environment/framework roles; local deployment context does not identify a globally unique application. |
| 26 | Laravel API | Evidence includes Cell Saviors proxy endpoints and ChildScribe local API. |
| 28 | web client | Docket's front-door mount table and driver-core extraction explicitly use this role. |
| 29 | staging domain | Domain role; explicit child-scribe.com context is preserved in facts and scoped aliases. |
| 39,49 | Laravel marketing; Laravel web | Framework/surface role, while narrower ChildScribe product-qualified names survive. |
| 40 | deploy-on-push | Deployment behavior, not an app identity. |
| 41 | production posture | Actual facts concern pnpm-bar-prod login/security rehearsal state. |
| 42,43 | mobile/web side; web voice | Cross-component area and voice feature; facts describe web/engine interactions. |
| 45 | refs/heads/main | Git branch ref explicitly on Docket worktree. |
| 47 | marketing-site epic | Facts identify Docket campaign tasks and merges; generic task/epic role. |
| 48 | forge user | Unix account role, with direct Cell Saviors installation evidence. A user account is not the Laravel app. |
| 52 | fleet worktree | Docket and Setpoint temporary checkout roles. |
| 53,54 | Client Dashboard; client dashboard | Unqualified UI role; no attested unique ownership. Specific dashboard entities remain. |
| 55,63,66 | api project; createApp; api app | Docket/Program Health API and constructor-function evidence excludes ChildScribe. Existing api keys and listings preserved without certifying api as a clean identity. |
| 57 | roll-off software | Industry/software category, no unique product attestation. |
| 59 | driver tree | Exact fact identifies Docket web/src/driver source area. |
| 64 | fleet wave | Execution grouping across multiple tasks. |
| 67 | campaign engine | Facts explicitly identify both Claude and Codex as engines. |
| 69 | fleet architecture | Facts concern Setpoint/Scry stack; architecture topic is not ChildScribe. |
| 70 | fleet skill | Fact explicitly identifies room-worker protocol role. |
| 73 | development build | Generic build flavor. |
| 74 | --dev-client | Fact uses it as an Expo CLI flag. |
| 76 | iOS client | Mobile platform/client role, with native WebRTC evidence. |
| 77 | voice pipeline | Mobile and OpenAI Realtime pipeline spanning components. |
| 79 | production profile | EAS configuration/profile facts. |
| 80 | prod environment | Unqualified environment role. |
| 81 | marketing source | Actual fact describes Docket marketing/office source boundary. |
| 86 | web frontend | Explicit Docket source identity facts, including contaminated ChildScribe endpoints. |
| 87 | Envoyer release deployment | Cell Saviors deployment-pattern context; no exact unique application spelling. |
| 88 | Laravel app symlink | Symlink role distinct from target application; actual scoped path aliases survive. |
| 89 | Laravel Herd project | Explicit instructions refer to multiple projects under Herd, not one app. |

Key provenance includes battery-designer episode 2f376340a10fd03045d1b5989bf9936f55972af8736f388c33bce8f12f4313f0 (Cell Saviors campaign); b35691724204ef2fa0c0de65bb681faf8d5e91aa0b86dcf9d757717b15e57db8 (reference_childscribe_forge_staging.md, Forge SSH endpoints, staging/prod release layout); b7c5c3ccef303b46d9581b42830f20f555eac845478fd1222cd6a68549a6080c (Advocates prod-ssh-access.md); and 472a399815e429f0570f0a764001cd9ccf740bef576b399ccc232aa3a668a1b9 (Hoopless live work via forge@server). Episode summaries were read; original transcripts and credentials were not accessed.

## Limits and pending live checks

The refreshed snapshot's participant metadata drift is LastSeen for all three, and Forge repo-ref rotation. Its eight added/changed participant facts are audit-derived statements, including incorrectly represented `same_as` edges from ChildScribe to api and forge. These were preserved and were NOT used as evidence for semantic ownership. They expose remaining graph contamination, not a failure introduced by this alias batch.

ChildScribe's wrong description, repo refs, contaminated facts, 24 ambiguous aliases, and five unexecuted rehome proposals remain. Existing global hollows, missing endpoints, and 486 collisions remain. No claim of broad cleanliness, honest fresh holdout, two real sweeps, or full-goal convergence is made. The lead's reported fixed-benchmark scores were not independently rerun by this gate.

Before actual live application, review a fresh postdeployment source hash and manifest, confirm the same 49 literal dispositions and 45 normalized keys, compare participant entity/fact fingerprints and all affected global listings/claims to that source, and verify the deployed guarded method and real nonempty backup. Any semantic or listing drift requires renewed review. After apply, compare exact live full facts/entities/indexes, backup restoration evidence, lookups, collision delta and no-mutation second pass, and perform the required live benchmark checks. The two certified snapshot manifests must not be forced through current live drift.

## Fresh postdeployment source addendum — 2026-09-05

Verdict: bounded fresh-source replica gate PASS. This supersedes the pending postdeployment-source portion above only. Actual live application, immediate live expected-input comparison, quiet-queue checks, automatic live backup verification and live postconditions remain pending. Deployment binary verification belongs to the separate deployment gate; this reviewer did not independently inspect either installed binary.

Independently hashed source /tmp/scry-childscribe-live-sep05.XlzT6P/source.badger: SHA-256 b8fda9a1c446d9f03dd8bc3116d1e49fd6020f7cd0e7d728553087b6ae445197, 77,105,082 bytes. Lead identifies capture time as 19:04:41 UTC after the 53fafa9 deployment. Manifest /tmp/scry-childscribe-live-sep05.XlzT6P/reviewed-replica/manifest-replica-only.json independently hashes to 24a765acc82e122c270e812e99993dbe3eff5d8261f88f05a0e065f16eb61ba6 and `cmp` proves it byte-identical to the previously reviewed refreshed manifest.

Repeated all independent-alias and independent-raw checks against new directories under /tmp/scry-childscribe-independent.3JdDSY/postdeploy. The unchanged exact manifest matches the fresh independent preview, with no fingerprint regeneration or weakening. All 30,231 entities and all 79,692 complete current/historical facts satisfy the exact expected-state comparison. The ChildScribe alias list alone changes 92→43. Collisions change 489→486. The complete alias map changes 51,509→51,469 through exactly 40 deletions, two Forge rehomes, and three preserved api claims. All affected global name/slug/alias listings match the previously reviewed owners; all normalized variants remain exactly enumerated. Every exact requested lookup is correct. The entire raw database changes 239,692→239,652 keys, with exactly 43 key/value differences and zero other mutations.

The automatic replica backup is /tmp/scry-childscribe-independent.3JdDSY/postdeploy/rollback.badger, 77,105,074 bytes, SHA-256 19abe1f696e6a81fd34d54c29821162487030882c4285623767c279159cc83d8. It was independently restored and its entire raw key/value map compared with a separate fresh source restore: exact equality. The eight-byte difference from the source's serialized backup size does not represent lost data; complete restored logical-state equality is proven.

Both second previews (original expected inputs and fresh preview without expected inputs) refuse the already-absent aliases and leave the state unchanged. The pre-existing 2,854 entities with no current/historical touching fact and 994 distinct missing endpoint slugs remain exactly unchanged. A separate participant-drift comparison against the prior predeployment original emitted no entity or touching-fact changes for childscribe-laravel, forge or api. Thus the existing semantic review applies unchanged; newly added unrelated graph data does not supply new ownership authority.

Evidence: postdeploy/result.json, postdeploy/raw-result.txt, retained postdeploy replicas and rollback backup, and the independent helper source in this archive. No live reads beyond supplied artifacts, live writes, remember, deployment, model calls, or shared source edits were performed for this addendum. Subsequent ingestion cannot be assumed harmless: the lead must compare the same expected fingerprints/listings/claims immediately before the guarded versioned apply and stop or renew review on drift. Remaining contamination, unresolved alias rows, broad hygiene, benchmarks and full-goal limitations above still apply.

## Fresh post-sweep source addendum — 2026-09-05, 19:30:41 UTC source

Verdict: bounded fresh-source replica gate PASS for the newly regenerated exact manifest. The earlier 19:04 manifest is superseded for this snapshot; its fingerprints must not be reused or forced. This reviewer used the same archived production code 53fafa91d621190c87245f0b0844270b4fdf44c9, not the lead's newer undeployed checkout.

Independently verified /tmp/scry-childscribe-final-sep05.qcoHQ3/source.badger: 77,445,247 bytes, SHA-256 6fb519ce181952aedbcc4f73011e33fc84a70c546f12e5dca4cc3b700f5204a3. The lead identifies the remote source as /Users/jclaw/.scry/backups/memory-20260905T193041Z.badger. Independently verified /tmp/scry-childscribe-final-sep05.qcoHQ3/reviewed-replica/manifest-replica-only.json SHA-256 da2fd1a2397d37dbffcd0074e4caee96e64f5a8255b0213aeb91abf80eab5fc0. Its only differences from the 19:04 manifest are the three participant entity fingerprints and the complete participant fact fingerprint. The 49 literal dispositions/reasons, 45 normalized keys, plan hash, affected claim presence/owners, and complete affected global listings are identical.

An independent two-way comparison of ALL participant entity fields and ALL full current/historical touching fact versions against my restored 19:04 source is saved at post-sweep/participant-drift.json. Participant facts increase 3,775→3,781. Entity changes consist solely of LastSeen on childscribe-laravel, forge and api, from 2026-09-05T18:26:06.336Z to 2026-09-05T18:58:38.747Z. No participant names, types, descriptions, repo refs or alias strings change.

Fact drift consists of six new audit-derived facts from episode 717d965d5282c765275fb29815345ba0bc200d1e7a9d125f04ab81df6bb7a707 and two historical status facts gaining InvalidAt=2026-09-05T18:58:38.747Z. The latter retain all other fields and describe the old 122-alias/2,206-fact counts from episode 945fbdd5334f0874936ad0803a5be7965461832601d627edd1e5bb6ca199ca56. The six additions describe manifest removal, repair, existing Forge/API listings and replica results of 43 aliases/486 collisions; the 43-alias fact is itself already invalidated. These are self-referential audit statements, not new product identity evidence. They do not justify any rehome and were not used as semantic authority. In particular, the current source still has 92 aliases and 489 collisions despite the imprecise newly ingested result statements. No historical fact was removed between the sources; two old versions differ only by added invalidation timestamps. No newly observed semantic evidence contradicts the prior per-spelling review.

Repeated independent-alias, independent-drift-full and independent-raw checks in new post-sweep directories. The exact newly pinned manifest matches the independently restored source preview with no regenerated or weakened expectations. All 30,345 entities match the original metadata except the exact ordered ChildScribe alias remainder, 92→43. All 79,926 full facts, including every invalidated fact and all provenance, match the source exactly after apply. Collisions are 489→486. The complete index is 51,658→51,618, with precisely 40 ChildScribe claims deleted, two existing Forge listings rehomed, and three api claims retained. All outside name/slug/alias listings for affected keys are exactly those reviewed; no foreign listing is orphaned, no extra normalized variant is removed, and every requested exact lookup matches the expected owner or absence. Both second previews refuse the absent aliases without mutation.

Separate raw Badger comparison proves original 240,524 keys → 240,484 keys, with exactly the same 43 changed keys and zero other key/value changes. Entire source restoration from the automatic replica backup is exact at the raw key/value level. Backup: /tmp/scry-childscribe-independent.3JdDSY/post-sweep/rollback.badger, 77,445,207 bytes, SHA-256 d236a1822dcfb911ff5ce3d1ba28897c0551cf55641370a2027cf947d23ce4cd. The 40-byte serialized backup size difference does not represent missing logical data. The same 2,854 hollow entities and 994 distinct missing endpoint slugs remain unchanged.

Evidence is retained in post-sweep/result.json, post-sweep/raw-result.txt, post-sweep/participant-drift.json and the independent original/apply/rollback replica directories. No live writes, remember calls, deployments, model calls, or shared source edits occurred. Lead-reported quiet-queue observations and installed daemon versions are not independently graded here. Immediately before guarded live apply, the lead still must verify the live preview matches this exact manifest and the queue remains suitable; any input drift requires refusal and renewed review. Actual live backup integrity and post-apply full-state/index/lookup/benchmark checks remain pending. Existing contamination, 24 ambiguous aliases, five unexecuted rehome proposals and broad-goal limits remain unchanged.

## Actual live-application backup postcheck — 2026-09-05, 19:35 UTC

Verdict: bounded actual-live-state PASS for the supplied automatic pre-apply and post-apply backups, independently restored and compared in new directories. This closes the pending live state/backup/lookup portion for this exact application; it does not grade live benchmarks, later ingestion, two real sweeps, deployment artifacts, or the full goal. The lead reports guarded application at 19:35:22 UTC, 49 dropped/zero refused, using reviewed production 53fafa9 and exact manifest da2fd1a2397d37dbffcd0074e4caee96e64f5a8255b0213aeb91abf80eab5fc0. This reviewer inspected supplied backups and reconstructed the operation, not the live daemon or original store directories.

Independently computed hashes and sizes:

| Actual artifact under /tmp/scry-childscribe-final-sep05.qcoHQ3/ | Bytes | SHA-256 |
|---|---:|---|
| memory-20260905T193522Z.badger — automatic pre-apply | 77,447,357 | 02c7561517ba179a01b1229c4d5e036100bd23b13a244bce81bd9349cdc73be3 |
| memory-20260905T193530Z.badger — actual post-apply | 77,441,558 | 71c339f10f5a193d4902cb4750f29b985cf8821e2f915565a9dfdadb7601cf3e |

The automatic pre-apply backup restores successfully and its independent preview matches ALL exact reviewed manifest expectations without regeneration or weakening. Independent-alias then applies the same request in its own replica and proves full facts/entities/index postconditions plus backup restoration. Independent-raw separately proves that replica changes exactly the expected raw keys and that its backup restores the entire original raw map. Independent-actual restores the actual post-apply backup into another new directory and compares its entire raw key/value map against the independently predicted repaired map: ZERO differences. This includes every fact, historical record, episode, adjacency, cursor, marker, queue, metadata and alias key, not just the participants.

Actual pre→post results:

- 79,926 full current/historical facts unchanged in every field, including invalidation/provenance; 30,345 entities retain every field except the exact ChildScribe alias removal.
- ChildScribe aliases 92→43 with the exact retained strings and order; 49 reviewed literal removals across 45 normalized keys; cross-type collisions 489→486.
- Entire alias index 51,658→51,618. Exactly 40 claims deleted, laravel-forge and forge-ssh-access rehomed to forge, and api-project/createapp/api-app preserved as api-owned. Every one of the 49 actual requested lookups passed. No foreign listing is orphaned and no unrelated index key changes.
- Entire raw database 240,525→240,485 keys. Exactly 43 changed keys: 40 alias deletions, two alias-owner replacements, one en:childscribe-laravel replacement. No additional actual-post raw differences exist.
- The actual post snapshot's second preview of the exact original manifest refuses the already-absent aliases and leaves its entity/fact/index state unchanged; its complete post-preview raw state still equals the independently predicted map. All previously measured hollow/missing-endpoint sets remain unchanged.

Unrelated state was explicitly inspected, not ignored. The entire actual automatic pre-apply raw map differs from the previously reviewed 19:30:41 source in exactly three keys:

1. meta:last_sweep_at advances to 2026-09-05T19:34:13.76812Z.
2. meta:last_sweep_report records Mac.attlocal.net scanning 94 files with zero ingestions, zero episodes and zero errors.
3. pq:1b9aeed7fd26d799d8f337a18299a488343fa559a3779dca7a4e87d063d6c0c8 is a new manual Cell Saviors/battery-designer audit item, enqueued at 15:33:54.809762-04:00 with attempts=0.

There are no other pre-apply raw changes: the complete graph and the reviewed entity/fact/listing/claim inputs remain identical to the reviewed source. All three unrelated keys survive the actual repair byte-for-byte. Thus the pending item was preserved; there was no hidden concurrent extraction/fact mutation inside the compared pre/post window. This observation does not promise how the pending item behaves in a later sweep.

Evidence: actual-live/actual-result.json, actual-live/actual-vs-predicted-raw.json (empty object), actual-live/reviewed-source-to-actual-pre-raw.json (all three complete differences), actual-live/actual-lookups.json, actual-live/actual-second-preview.json, actual-live/result.json, actual-live/raw-result.txt and the separately restored original/actual-post/independent-repair/rollback directories. No live write/read of original stores, remember, deployment, model call or shared source edit occurred in this postcheck. The 24 ambiguous aliases, five unexecuted rehome proposals, contaminated metadata/facts, 486 collisions and existing global anomalies remain outside this bounded PASS.
