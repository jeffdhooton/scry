# Independent fresh-snapshot extension

2026-09-06 UTC. This extends `REPORT.md` SHA-256 `05d36896062e2e159daed3171103b68882ea0ddaf22f5917660eea1ed1b7262c` for the identical schema-1 startup-refusal artifact `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`.

Verdict: bounded predeployment PASS extends through the independently restored 015255 shared and laptop backups. Both 014746 exact-artifact comparisons, the final 015255 shared exact-artifact comparison, and all private restore/raw preservation and drift accounting passed. This permits the identical schema-1 startup-refusal artifact to proceed through the lead's deployment gate; it does not verify deployment itself. It does not certify the preexisting resolver's timestamp-preservation semantics or whole-goal completion. The independently discovered predeployment drift below must not be described as entirely additive or timestamp preserving.

All work uses new private restores and read-only source backup/retained artifact access. No live store, queue, configuration or provider was changed. The reports contain hashes/counts and fixed field names only for real stored data; no fact text, value-derived key, source payload or error message is emitted.

## Fresh complete backups and direct restores

Each full file hash was independently checked. Each destination was nonexistent before direct Badger Load. The original inspected private `evidence restore` helper then compared complete raw maps before/after candidate Open, entity/fact reads, offline index creation and local recall.

| Backup | Bytes | SHA-256 | Private destination | Raw rows |
|---|---:|---|---|---:|
| Shared 014746 | 74,762,625 | `041b07af4d8774f234f714a27b12f792b2eb5a6365c0fddf5eac035054b116fb` | shared-fresh | 245,373 |
| Laptop 014746 | 19,445,016 | `70b59f513fa0101b48ad57ca99d6a2e52ff7a1f6ce2b998e733c7cf1cb1a1097` | laptop-fresh | 83,378 |
| Shared 015255 | 74,777,550 | `14f86ef0aab9d1058dabe35f46f702f378bab6034971db51a157faead260734d` | shared-final | 245,401 |
| Laptop 015255 | 19,445,000 | `8f6564776d75354738f46c3c5e7af72b8a4a4e607f620876bdac8ac9b2ae1231` | laptop-final | 83,378 |

Shared source paths are `/tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T<timestamp>Z.badger`; laptop source paths are `/Users/jeff/.scry/backups/memory-20260906T<timestamp>Z.badger`. All private destinations are beneath `/tmp/scry-schema-independent.wksyLD/`.

Every Open/index/read comparison passed exact logical key/value equality. Shared 014746 digest is `dd58bd54ec7450dffc52388e95fea4fd32a3aa6783e1655a767e300debd0e479`; shared 015255 is `b95089338c5c09a669935a4137b2cbaac89cbce6d479025283ba9ca1fe1afba7`. Both laptop backups have the same logical digest as the original `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`; complete before/after snapshot comparison proves no laptop logical drift despite different backup-stream hashes.

014746 shared contains 81,290 facts (73,437 current, 7,853 historical), 30,953 entities and 9,407 episodes. 015255 contains 81,299 facts (73,445 current, 7,854 historical), 30,956 entities and 9,408 episodes. Local recall payloads were 11,410 and 11,409 bytes, respectively; laptop payloads 9,409 bytes.

One initial 015255 restore invocation had a reviewer-transcribed incorrect expected hash; the helper refused it before creating/loading the destination. The subsequent invocation used the independently verified correct hash and succeeded. No unverified backup was admitted.

## Complete 013401 to 014746 drift accounting

A new independent `cmd/independent-drift` helper compares complete raw maps and reports every changed/removed key by SHA-256, every changed value by old/new SHA-256, and changed top-level JSON field names. A separate second direct restore of the same 014746 backup (`shared-drift-fresh`) permitted this comparison without sharing an open database with benchmarks.

Added families: adj 94, al 93, att 29, en 53, ep 8, fa 118, pq 1, ve 45. Removed: adj 2, fa 2, pq 6. Changed: att 4, en 27, fa 3, meta 3, pq 1, ve 9. Thus the 116 net additional facts do not mean zero raw fact-key removals.

Both removed facts have exactly one full assertion match under a new key: same source/relation/destination/value/raw relation/text and InvalidAt, with all old provenance still present. Both new keys have earlier ValidFrom and increased confidence; episode counts grow from 1 to 3 and 1 to 2. All cited episodes exist, and both corresponding old adjacency keys are removed with new adjacency keys present.

| Old fact-key SHA-256 | New fact-key SHA-256 | New payload SHA-256 |
|---|---|---|
| `9effacd7aa7b0dc27659a9f3f51c3ab1db8d9b8479a39e36789807dcf530c621` | `beaa38e1e52ac07ceefe5bae32db50f63cafe0203573b34947180fd9f78fc5e5` | `8417ff0f25a0858aca1e8407f839e124ed047b5638f79b10e5f195712d54a945` |
| `23b1fdf3b65790e8ac7c3a8df9dd2e3c3399f4401bf65a8933799cdb2ff0f613` | `d56cc9d3fe1224bf35684543d8f9378cb312c747359e493a9854035c6f957c34` | `3af5961635361f5eba6f8e586a830bf5617a323d4dae146ad9d68d94a2e7c590` |

This shape matches the existing `mergeFact` at `internal/memory/resolve/resolve.go:669`: append provenance, take maximum confidence, choose earlier ValidFrom, delete old key, put merged record. That is an inference from the stored deltas and inspected code, not a captured live call trace. This is preexisting timestamp alteration and is explicitly not a whole-goal historical-preservation PASS. The startup patch does not alter this resolver.

The three changed existing fact values change only InvalidAt from current to historical. Their assertion identity/text/provenance remain unchanged. Every one of the 7,848 facts already historical in the original snapshot is byte-identical at the same key. All preexisting alias-index claims and all 42 ar/rt/rs repair records are byte-identical. Existing attestation changes retain all old provenance.

All 27 changed entities retain their core identity (slug/name/type/description/created_at) and every old exact alias spelling. Four changed entities each remove one old repo_ref while remaining at six refs. For all four, the resulting list exactly equals the last six of the old list union newly present refs, consistent with the unchanged resolver's six-reference cap (`resolve.go:322`, `:959`). Their safe identifiers and capped-reference accounting are retained in `fresh-drift.jsonl` as `entity_delta`. This existing metadata truncation is not authorized or repaired by the startup review.

All 14 previously parked pending records retain their complete raw payloads. Six disappeared unparked pending records each have a completed episode in the new snapshot. One pending record was added. One old unparked record became the fifteenth parked record: key SHA-256 `54a4f9ef66d874164ab757d45ce98a8dfdbdb46ad238843abaff384cd3f73a5d`, new payload SHA-256 `ea6a2a1e75f974cf656b1ec3965669f0174fdbfcfe97604314e21ae7b84a6bbe`. Its source/text/identity/enqueue/occurrence fields are unchanged; only attempts, last_error, next_attempt and parked changed. Attempts is 2, error length 122 bytes, error SHA-256 `eb2d83ed54cddee53d26683b0fd6c8e3c07ebdab0eabc144bd5a6bc9fd94c55d`, fixed safe category `alias_claim`. It was not retried by this reviewer. The raw error is never printed.

## Complete 014746 to 015255 drift accounting

Added: adj 4, al 5, att 1, en 3, ep 1, fa 9, ve 6. Removed: one unparked pq whose completed episode exists. Changed: en 4, fa 2, meta 1, ve 2.

No existing fact key disappears. One changed fact only adds provenance; the other changes only InvalidAt from current to historical. All 7,853 previously historical facts remain byte-identical. All 15 old parked pending payloads, old alias claims and 42 repair markers are byte-identical. No pending work remains other than the 15 parked records in this snapshot. Entity core identities remain equal; three entity changes are last_seen only and the fourth adds aliases plus last_seen. This is snapshot accounting, not a claim of successful extraction of parked work or of two clean sweeps.

## Fresh exact-artifact benchmarks

The original helper invoked both exact retained/candidate artifacts for all five suites against each 014746 private snapshot and compared complete store maps before/after. All ten invocations per snapshot succeeded. Every result field except timing was exactly equal between artifacts, including full miss membership/ranks, hashed miss previews, mean answer rank and payload sizes. All raw records remained equal.

014746 shared scores: 51/62, 29/66, 7/7, 45/50, 47/50. Maximum payloads: 12,112; 13,353; 9,962; 11,541; 11,541 bytes. Laptop scores remain 22/62, 13/66, 5/7, 28/50, 30/50 with its original payload maxima. Zero responses exceed 24,576 bytes. Original held-out floors still fail.

015255 final shared exact-artifact comparison also passed every non-timing result field, full miss membership/ranks and hashed previews, and complete raw preservation. Scores remain 51/62, 29/66, 7/7, 45/50, 47/50; maximum payloads are 12,112; 13,366; 9,961; 11,532; 11,532 bytes, with zero cap exceedances. Each suite's mean answer rank is unchanged from the original report. All 245,401 active logical rows remain exactly equal before/after the ten CLI invocations. Laptop 015255 was not benchmarked again because its complete raw map is identical to the already benchmarked 014746 laptop snapshot. The successful per-question ranks were not separately rerun on the newer shared snapshots; exact-artifact bench output provides full miss ranks and aggregate successful mean ranks here.

## Reproducible commands and evidence

The exact binary build/test/refusal commands and original helper are pinned in REPORT.md. This extension makes no source change to the shipped binary or original helper. New direct restores use `evidence restore <full-source-path> <new-private-destination> <exact-hash>` and write the named restore JSON. Each benchmark uses `evidence bench <private-store> /tmp/scry-schema-independent.wksyLD/source /tmp/scry-fact-guard-deploy-sep06.uN9zYJ/scry /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/scry`.

The new helper was built from the private candidate export with `CGO_ENABLED=0 go build -o /tmp/scry-schema-independent.wksyLD/drift ./cmd/independent-drift`. Complete comparisons are:

```sh
/tmp/scry-schema-independent.wksyLD/drift /tmp/scry-schema-independent.wksyLD/shared /tmp/scry-schema-independent.wksyLD/shared-drift-fresh > /tmp/scry-schema-independent.wksyLD/fresh-drift.jsonl
/tmp/scry-schema-independent.wksyLD/drift /tmp/scry-schema-independent.wksyLD/laptop /tmp/scry-schema-independent.wksyLD/laptop-fresh > /tmp/scry-schema-independent.wksyLD/laptop-fresh-drift.jsonl
/tmp/scry-schema-independent.wksyLD/drift /tmp/scry-schema-independent.wksyLD/shared-fresh /tmp/scry-schema-independent.wksyLD/shared-final > /tmp/scry-schema-independent.wksyLD/final-drift.jsonl
/tmp/scry-schema-independent.wksyLD/drift /tmp/scry-schema-independent.wksyLD/laptop-fresh /tmp/scry-schema-independent.wksyLD/laptop-final > /tmp/scry-schema-independent.wksyLD/laptop-final-drift.jsonl
```

Initial strict assertions intentionally exposed changed validity, removed raw fact keys, confidence changes on relocation and removed repo refs. These were investigated with narrower field-by-field assertions and explicit accounting, not silently interpreted as full preservation. A concurrent private final-drift retry encountered Badger's directory lock while the final benchmark was active; it made no data write and was rerun successfully after the benchmark completed. Final evidence still discloses each observed alteration and does not strengthen the scope of the bounded startup verdict.

| Evidence | SHA-256 |
|---|---|
| shared-fresh-restore.json | `f5110eb4c513e2bf3aa85e94083abb7d3d76fa0c79c61f3ee6978b90e0469b6e` |
| laptop-fresh-restore.json | `241a431cd56888ad83e47488b8a19c854f83529a0d13438a1831cdc2ba4ae284` |
| shared-final-restore.json | `aedc3fa9bfe58b12491e32fc05baf8140c5490af6c26b1f1080d6d0dbbe4ae24` |
| laptop-final-restore.json | `a750164ab74663ed60813e70c352c70fd442d1099cc7349ab1ea0ee8adfd4f67` |
| fresh-benchmarks.jsonl | `8a0511240347c72a5bcccc6c61811da1d238bf2062b65c262d40d0aacd847fe0` |
| laptop-fresh-benchmarks.jsonl | `3cd393c1f4509e15ef04a2af80a18f7bbcaf11664f6d4e2ae27e25e7cdddf2bd` |
| final-benchmarks.jsonl | `f5382ba7b70c7066e55e4cf644450389a8b55bfdd7850d3d1b3e066ee35e9193` |
| fresh-drift.jsonl | `b72178e2e6ca8a833f39847c9b4bd675c9580d5a840acffb0ed89ecfb40d412c` |
| laptop-fresh-drift.jsonl | `62494b9a3897e379bd594b3e278c344208e99d09717212da0b040cd65b92070b` |
| final-drift.jsonl | `afc72dffcfccbf0c2137dcb18cfd91b1dcd46eab1c314f8864066b1f0d909cb0` |
| laptop-final-drift.jsonl | `62494b9a3897e379bd594b3e278c344208e99d09717212da0b040cd65b92070b` |
| source/cmd/independent-drift/main.go | `38d94c3b7a7f8478d244acea0fd7c06a1a9af0f4b17f7456ee2012cec000de32` |

All original limitations remain, especially populated Restore's DropAll-before-validation hazard, destructive older binaries on a future numeric schema, no writer-floor enforcement, no schema bump/migration approval, no live deployment verification, and no whole-goal completion claim.
