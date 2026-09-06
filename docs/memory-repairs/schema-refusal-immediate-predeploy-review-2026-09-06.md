# Independent immediate freshness gate, 015908

Verdict: bounded startup-only predeployment PASS extends to the 2026-09-06 01:59:08 UTC shared and laptop backups. No new preservation issue was found that blocks installing the identical schema-1 startup-refusal artifact. This is a raw-freshness extension of REPORT.md SHA-256 `05d36896062e2e159daed3171103b68882ea0ddaf22f5917660eea1ed1b7262c` and FRESH_EXTENSION.md SHA-256 `40fed437ea5d463266fcf10457055a3133e9170662b9c94a59e040a5c124a797`, not actual deployment verification or whole-goal completion.

The candidate was rehashed and remains `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`. No code, ranking, schema or format changed. Per the requested scope, the previously completed full benchmark matrix was not rerun.

Both complete backup hashes were independently verified, then each backup was directly loaded into a previously nonexistent private destination. Candidate Open, AllFacts, Entities, offline index construction and local recall were followed by complete raw key/value map comparison. Both passed exact logical equality.

| Backup | Bytes | SHA-256 | Destination | Raw digest |
|---|---:|---|---|---|
| Shared 015908 | 74,792,395 | `e9a91a9a181bf45f65cb1d5034de80f975d55a14e3b0b710eedd70c2469b4b46` | shared-immediate | `697dfbf8d61e6be8bb61ef0a728f8a1289df495fd9d72b18e2f5271deb8a4b06` |
| Laptop 015908 | 19,445,008 | `01b4a03beea33c64fdcf30ab2da6cba9840c3595d6f6c10cb415535dacba3d21` | laptop-immediate | `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7` |

Private destinations are under `/tmp/scry-schema-independent.wksyLD`. Shared contains 245,413 raw records, 81,304 facts (73,449 current and 7,855 historical), 30,956 entities and 9,409 episodes. Its local recall payload is 11,418 bytes. Laptop remains 83,378 raw records with the same complete raw map as 015255; its payload is 9,409 bytes.

Complete 015255→015908 shared comparison found no removed keys. Additions: 5 facts, 3 reverse-adjacency keys, 1 episode, 1 unparked pending input and 2 value-evidence records. Changes: 4 entity last_seen timestamps only, 1 attestation list retaining all prior provenance, 1 metadata value, and 1 fact's InvalidAt moving from current to historical. That fact's text, identity, timestamps other than InvalidAt, confidence and provenance are unchanged; its safe key SHA-256 is `9b5365bfdc54876ad84edfbc7038787e077d7261ce15738e5961c7ee5db397df`, with old/new payload hashes in immediate-drift.jsonl. This new invalidation is observed predeployment activity, not a candidate Open mutation.

All 7,854 previously historical facts remain byte-identical at their original keys. No assertion is relocated or removed; every old assertion and its provenance remain present. Every existing alias-index claim and all 42 ar/rt/rs repair markers are byte-identical. All existing entity names/types/descriptions/aliases/repo_refs are unchanged. All 15 parked pending records retain their complete raw payloads; none changed, disappeared or was retried. The new unparked input has attempts 0 and no error; key SHA-256 `71226390666152e0e1df6db32023e3b4a6b6ba6ce6d5ae5b9c59751c443a7dc8`, payload SHA-256 `dd8f30763b430c350360bf2a89ef074a3af6f31abbf8da4b73b69dfeb6d80f4f`. Candidate opening/index/read preserves that new input too. The complete laptop drift comparison found zero additions, changes or removals.

Commands used the unchanged, previously inspected and hash-pinned private evidence/drift helpers:

```sh
/tmp/scry-schema-independent.wksyLD/evidence restore /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T015908Z.badger /tmp/scry-schema-independent.wksyLD/shared-immediate e9a91a9a181bf45f65cb1d5034de80f975d55a14e3b0b710eedd70c2469b4b46
/tmp/scry-schema-independent.wksyLD/evidence restore /Users/jeff/.scry/backups/memory-20260906T015908Z.badger /tmp/scry-schema-independent.wksyLD/laptop-immediate 01b4a03beea33c64fdcf30ab2da6cba9840c3595d6f6c10cb415535dacba3d21
/tmp/scry-schema-independent.wksyLD/drift /tmp/scry-schema-independent.wksyLD/shared-final /tmp/scry-schema-independent.wksyLD/shared-immediate
/tmp/scry-schema-independent.wksyLD/drift /tmp/scry-schema-independent.wksyLD/laptop-final /tmp/scry-schema-independent.wksyLD/laptop-immediate
```

All four commands exited 0. Output was redirected to the following private evidence files:

| Evidence | SHA-256 |
|---|---|
| shared-immediate-restore.json | `0280506575103e11cea27daa3d10a0c6e720a033404294454e62278a9ae12799` |
| laptop-immediate-restore.json | `6a55b5e626b5c229905638e400612ff062c5f1313fbde3b909337c449a39b886` |
| immediate-drift.jsonl | `35778f052df9f608488b448f3eb8a9054e50f9a77690ca3e58f825ddd63de4e1` |
| laptop-immediate-drift.jsonl | `62494b9a3897e379bd594b3e278c344208e99d09717212da0b040cd65b92070b` |

No live/shared-store write, provider call, queue retry, deployment or credential/source-payload output occurred. Physical Badger file equality is not claimed. All original exclusions remain: preexisting backdating and repository-reference cap semantics, populated Restore safety, future schema migration, actual deployment verification, unmet original benchmark floors and whole-goal completion. The lead remains responsible for actual immediate backups, deployment and independent postdeployment verification.
