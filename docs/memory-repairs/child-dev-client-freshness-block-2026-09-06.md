# Independent fresh-source extension: BLOCK

The 2026-09-06 00:11:20 UTC source fails the preserved-fact safety gate. A historical CADFormats fact has been overwritten at its existing raw key. Separately, the original password-setting user message survives production redaction in a reproduced episode. Neither finding is caused by the proposed Child alias removal, but the fresh snapshot cannot inherit the earlier fixed-source PASS as a safe live-apply gate.

No fresh alias apply, backup-coupled repair, benchmark comparison, deployment, source repair, credential use, rotation, deletion, queue operation, or live write was performed. This is a separate BLOCK report, not a revision of the fixed 23:51:34 report. The lead reports a later prevention candidate; that candidate is outside this review and cannot retroactively recover the lost fact or remove a persisted credential.

## Pinned inputs and independently restored scope

Common input directory: `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3`.

| Input | SHA-256 |
|---|---|
| `memory-20260906T001120Z.badger`, 74,175,654 bytes | `5b8e97894bea35a0ba2d1d63a1be01725845814051c8cb5c36273a0d4a5d1d6b` |
| `dev-client-replica-001120/manifest-replica-only.json` | `02319c379358bf53087e5cde98b31a369df7ea35becbee0f9686f8982a21357b` |
| preceding `memory-20260905T235134Z.badger` | `7680cdcff3563c4cc2d33577506fc4c4aad4931d8bae1016f90e617bd7ee863e` |

Both backups were independently loaded directly with Badger into private replicas under `fresh-001120/`, using the previously archived deployed d1f0a958 source. Complete raw iterations include every family. The prior canonical key-to-base64 map hash remains `0c766cd8dec9d9ea45ee43564cbd6740e30c659d9d7a65a65d032f759c0b4d5a`; the new hash is `33725496009b935f84a622d5d9bfeb3e26415d210e673deea6d79b0b57a41306`.

There are 376 added keys, 42 changed keys and no removed keys. New/changed families: facts 104/7, entities 51/18, episodes 10/0, claims 70/0, adjacency 64/0, attestations 24/0, cursors 5/3, metadata 0/4, pending inputs 1/1 and value evidence 47/9. The fresh source has 244,079 keys, 80,948 facts (73,110 current, 7,838 historical), 30,776 entities, 9,381 episodes, 52,182 claims and thirteen pending inputs. Every existing rejection and retirement record survives exactly: four rejection keys, nineteen retired-slug records and nineteen retirement records.

## Proven historical overwrite

The same raw key exists in both sources:

`fa:cadformats-file-workbench:status:~in-progress:1788566400000000000`

| Field | Prior historical record | Fresh current record |
|---|---|---|
| Value | `in-progress` | `in_progress` |
| ValidFrom | `2026-09-05T00:00:00Z` | same |
| InvalidAt | `2026-09-05T17:41:56.543Z` | absent |
| Confidence | 0.95 | 0.95 |
| Provenance | `0e1566db7e3dc2deca2e43848fb0818dd48cfbef2a4c4fa1d53fd7d08364b0ff` | `a0e8c39d7c9b11a2e36fe77b8c589202af635dae87e6ce1d2a102ed1958a5ab6` |
| Exact raw value SHA | `44805a4cf211693496ac56d89433209a00dfc6e90fc192b0ee4ae2c36ba0e026` | `1c7aff7ef471dfb0cf756869db1ba8d398e8a11455fcf4e88a55bd7a4bd02f7e` |

The old statement says the ZIP/package boundary is complete and the geometry reader is next. The new statement says a PASS covers only the workspace redesign and does not complete the broader workbench goal. They are different assertions from different episodes. This is not a harmless separator normalization, provenance union, or legitimate historical retention.

A complete scan of all 80,948 fresh fact records finds zero copies of the old raw record and zero occurrences of its exact old statement under any other key. The old episode record still exists, which does not preserve the lost fact's validity and provenance. `fresh-001120/overwritten-fact-content.json` and `lost-copy-search.json` contain the complete nonsensitive old/new evidence.

Static inspection of the unchanged deployed code explains this failure: attribute keys normalize both values to the same `~in-progress` slot; the resolver's matching path looks only at current facts; the new-fact vacancy check is restricted to fallback relations; canonical `status` proceeds to PutFact, whose transaction sets the occupied fact key. The raw snapshot disproof does not depend on the lead's separate synthetic reproducer.

I read all seven complete changed old/new fact payloads. Besides this destructive overwrite, two records append provenance and four gain InvalidAt. Those four preserved invalidations are not semantically endorsed: a Docket deployment claim need not refute a migration count, and a later Scry measurement need not refute a previous deployment equality assertion. They remain separate evidence-quality issues, with their original bytes preserved in the fresh snapshot. No corrective disposition follows from this grade.

## Credential boundary, without disclosure

The implicated stored episode is `aba016291510f78fcf37859736143c5081b7a876c86663e434cc3e4672bfb231`, with recorded SourceRef:

`/Users/jeff/.codex/sessions/2026/09/04/rollout-2026-09-04T17-45-42-01a06e62-7ae4-7500-9c05-65f452ee877d.jsonl#56119088-73230901`

The exact raw range is 17,111,813 bytes, SHA `04a12000fc233c5b8fd96fb5ce5bc831f0180f8a2ffbff12c96f4534941485f7`. A private test wrapper bounds the reader to that recorded end and calls unchanged production `parseCodexTurn`, `chunkTurns` and `Redact`. It reproduces the exact stored episode ID and SourceRef. The projection is 13,093 bytes, SHA `70475452dc396f0e39ad4557d938f4c3bbe31d8563a89bb39d6779796e8e388a`; its full text is omitted from review artifacts.

The original user message's syntax is `change [optional words] password to [SECRET]`. Its source record occupies bytes 62,359,021–62,359,451. Its parsed-turn SHA is `37a9c8cc5cab731acbfb975dc1223ff84fc957907fbc70e7464fffd58997d0ca`. Production Redact leaves that complete turn unchanged, and the reproduced projection contains the complete unchanged turn. These assertions are tested as booleans; no password or derived spelling is printed.

`credential-record-hashes.json` records only whole-key and whole-payload hashes/sizes for the two password-related `cadformatscom uses` rows carrying that source ID. It does not claim both rows independently contain the plaintext credential. Entire implicated keys, attribute values, fact text and source projection are suppressed from generated review excerpts. The source and restored database remain the supplied evidence; no credential was used or sent to an external system. `bounded-source-proofs.json` contains the reproducible redaction failure evidence.

## Child-specific evidence remains bounded

The independently regenerated 30 matching facts, 204 companions and 18 stored episodes are physically identical to both the prior closure and the supplied fresh discovery exports, respectively at hashes `0742e21dfbd1aa4da297a49d076dfad2ada7e172dc7af68e693af5c9a1d537f0`, `2a2dc6289c57347d2f32356b1b8a7d8200a17801e99d9f9f4f1c1d47981555d1`, and `9fa75de33835b9f362107a8e5078f4cf6372611887df8ab8aaa3ef02bc6d9aa6`. The five unavailable originals from that domain review remain unavailable/unverified; this extension does not close them.

I independently reconstructed the complete fresh alias Expected. The unchanged plan remains `ee37314aab154b99ff1187cf368a6d3a54828723591a491dda80ac8e08aaa438`. The Child entity fingerprint is now `bf24ff9a8da8d32e97380d2a59df8cb1884ad463459b8ce4f149d52a0e41152d`, and its complete touching-fact fingerprint is `83848794cb056b6d07e7e1ec9461ab6d25060e0bba26648c7808a98d9c7f243a`; all submitted Expected fields match. Child retains 39 aliases, unchanged exact claim/listing ownership and the existing rejection closure.

Child's metadata delta is last_seen only. Its two added touching facts derive solely from the once-sent actual stops-table note `3be6deac923caf55d196a90eff0d29bbc4e8c5051c20bfc7177293872bcfce19`: the alias count of 39 and the completed removal assertion. The latter's raw `removed_alias_from` relation maps to `produces`; that misleading relation is preserved as an existing assertion, not endorsed or used as dev-client identity evidence. There is no new positive recipient or rehome inference.

The alias transformation's narrow structural safety can be investigated separately because it is disjoint from the overwritten CADFormats key. That separation does not make a current safety PASS true. This report therefore grants no fresh alias replica/apply PASS, and no alias operation is used to conceal or repair the source loss.

## Complete structural inventory and review limits

Complete independent before/fresh structural lists are equal under the production normalization semantics: 2,441 dangling endpoint occurrences; 3,869 missing listing claims; 505 wrong-owner listing claims; 28 unlisted claims; 462 normalized multiple-owner listings; 27 normalized multi-type listings; 2,849 entities with no facts; 2,967 with no current facts; and 1,099 self-loops. Missing source episodes, missing/extra adjacency, and dangling alias-index owners are zero. No new defects appear in these particular lists. The overwritten history is real despite unchanged structural defect counts and rising total fact counts. These are not production hygiene's broader collision counts and do not establish graph cleanliness.

The complete raw delta was computed across every family. The old/new changed-fact provenance union contains 33 stored episode IDs; its full one-hop closure contains 748 old/new companion versions. All were extracted and hashed, with sensitive payloads omitted from excerpts. All 22 transcript SourceRefs were opened at exact recorded byte bounds and hashed. Seventeen exact projections reproduce using the public production distiller; five EOF/resume boundary mismatches reproduce using the private bounded-reader wrapper with unchanged parsers/chunker. The remaining eleven are stored manual episodes, not newly reconstructed original transcripts.

Human semantic reading in this extension completed the full seven changed old/new fact payloads, the exact overwrite comparison and lost-copy search, the full incoming `a0e8...` projection, the Child-specific added facts/metadata/note, selected other changed/new entity and fact payloads, and the hash-only password syntax check. It did **not** complete a line-by-line semantic reread of every new fact/entity, all 748 companion versions, or all 22 projections after the conclusive blockers. Automated extraction or hashing is not represented as human semantic verification. The lead explicitly directed finalizing this BLOCK without a whole freshness PASS. Fresh repair/failure/rollback/benchmark suites were accordingly deferred; the earlier fixed-source results are not reused as fresh results.

## Commands and artifacts

From `/tmp/scry-dev-client-technical.QyrtR5`:

```sh
go run ./cmd/freshness-review /tmp/scry-dev-client-technical.QyrtR5/fresh-001120 /tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260906T001120Z.badger /tmp/scry-alias-rejection-deploy-sep05.pPmnn3/dev-client-replica-001120/manifest-replica-only.json /tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260905T235134Z.badger
SCRY_PRIVATE_FRESH_REVIEW=/tmp/scry-dev-client-technical.QyrtR5/fresh-001120 go test ./internal/memory/distill -run TestFreshnessReviewBounded -v
```

The first command reproduces full raw comparisons, independent Expected, domain equality, overwritten-record/lost-copy checks, complete structural inventories and source hashes. The second reproduces the five exact bounded projections and the redaction failure without writing their full text. `fresh-file-hashes.json` pins this report, the private extension helper/test sources and the redacted evidence. The earlier fixed report and its original helper hashes remain intact.

Further work requires independently graded prevention plus separate review of the already-lost fact and credential boundary, followed by a new complete source gate for any alias operation. This report authorizes none of those mutations and does not prescribe silently replacing a timestamp, adding fabricated history, rotating access, or deleting evidence.
