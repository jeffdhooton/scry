# Corrected private overlay: root-only replica evidence

2026-09-06. This is conservation and typed-recognition evidence, not an independent
code verdict or production approval. The first candidate's independent NO-GO is
retained in `ordered-overlay-code-independent-disproof-2026-09-06.md`. Its corrected
successor remains private and under a fresh exact-pin review.

Corrected manifest SHA-256
`9e44b47b16aaf6ac31c88af1347205789bbe873001e463ed76c5696b38331463`:
600 unchanged e097fa6 baseline files and 39 additions. Relative to the first rejected
candidate, only overlay controls40268dde and lookup57c9b939 implementations changed.
The complete corrected source is frozen at
`/tmp/scry-ordered-overlay-correction-sep06.soHIfk/code`.

## Fresh backup and restores

Mini `/Users/jclaw/.scry/backups/memory-20260906T105609Z.badger`, 77,018,148 bytes,
SHA-256 `9aa54e517da5f9ee9f39406815ecb9d08086005ee675bf6a06ccaaf8a365bdfd`.
Copied and hash-verified under `/tmp/scry-foundation-closure-sep06.8IEPu5/`, then
restored into separate fresh `shared-105609` and `overlay-105609` directories there.
Both preserve all 249,361 raw rows through Load/Open/index/read; framed digest
`585bd8c243d6bd752aa17de889d28618ded205c896e1592020bbf1de0790054b`.
31,500 entities; 82,353 facts (74,448 current, 7,905 historical); 9,487 episodes;
33 pending; 56,765 adjacency; 53,179 alias; 11,401 attestation; 3,199 cursor;
4 alias-rejection; 19 retired slug; 19 retired spelling; 1,397 value evidence;
5 meta rows. `shared-105609` remains unchanged.

## Root probe, never ship

`/tmp/scry-overlay-corrected-replica-sep06.IfmSVO/code` is an exact separately
verified copy of all 600 baseline and 39 corrected candidate files. Only this
separate export adds `internal/memory/store/private_overlay_replica_test.go`, SHA
`ae03ddadae9242a72cc07ba1d0953346053afc07f28ee8a88b4978cd94a4e5c5`.
NEVER integrate this file. It uses a fixed explicit replica path/gate and prints
only counts, digests, static typed outcomes and timings, not actual memory content.
The probe mechanically reuses the previous private fixture with the new snapshot
pins/counts/time and two new synthetic names, checking both names are absent first.
No candidate source or conservation assertion changed for this probe.

Command in that export:
`SCRY_PRIVATE_OVERLAY_REPLICA=105609 CGO_ENABLED=0 go test ./internal/memory/store -run '^TestPrivateOverlayRestored105609$' -count=1 -v`.
PASS on the first run, package 16.133s, log SHA-256
`4723cc9f75ddd0e75221571545af3ea42f9ca9190b561fa752527fde0c959b86` at
`/tmp/scry-overlay-corrected-replica-sep06.IfmSVO/actual-corrected-replica.log`.

- Unadopted existing identity remains typed deferred, while independent synthetic
  utility remains proposed. Two candidates, two projections, 27 witnesses, five
  decisions. All 249,361 raw rows unchanged, zero actual identity/fact writes and
  zero events. Planner 2.727029792s; after reopen 2.631645875s.
- Separate explicit PRIVATE-REPLICA adoption preview preserves all raw bytes.
  Apply adds exactly 31,500 anchors and one marker, no other family, while all
  249,361 original rows remain exact. Manifest 16,463,209 bytes, SHA-256
  `85dcfe11f0ebcbc90f7c72c58ce7a9989a8331ed37a8cfcceec8b6281d8c23c0`.
- After private adoption, the existing identity validates as legacy and supplies
  its descriptive route. Two unmaterialized synthetic candidates, three projections,
  108 witnesses, six decisions. All 280,862 post-adoption raw rows remain unchanged
  by planning, zero writes/events. Planner 2.663757708s; after reopen 2.655303167s.
  Framed digest `72aa0dd3e6d8823ff7e3d4e572a7ebfa6df5edf29b0d1689541352008ef1fcc0`.

Preserve the now-adopted `overlay-105609` replica; the initial-state test must refuse
if rerun there. These are root-only single-call measurements, not p95 or independent
semantic approval. No live adoption, graph repair, deployment or sweep occurred.
The live a06cd7b deployment remains unchanged. Full fact/support/lifecycle/Force,
assertion-preservation, cleanup/recall and two complete grading rounds remain open.
