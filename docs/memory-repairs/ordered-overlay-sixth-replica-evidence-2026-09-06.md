# Sixth ordered overlay — root-only restored replica evidence

2026-09-06. Bounded read-only planner conservation, not independent approval,
normal-write prevention, support/ownership closure, p95 or a full-goal round.

Candidate /tmp/scry-ordered-overlay-sixth-sep06.IGAgjR/code, freeze
da175dd075a97cf46de4a2dd6724fa0c96131bf16af00fb4f0d71e84035224f7.
Separate actual-probe export /tmp/scry-overlay-sixth-replica-sep06.9EpoZB/code:
600 exact baseline files and48 exact additions verified SHA/git blobs/byte lengths.
Only extra source private_overlay_replica_test.go, SHA
307a013964ddcbb638160faaf46daba06b6b21118fcc3ae2a31a5c37f834fbe5.
NEVER SHIP or integrate the probe. All copies and authored scripts use apply_patch.

Fresh Mini backup /Users/jclaw/.scry/backups/memory-20260906T122103Z.badger,
77,150,279bytes SHA
a49eefd40e6e66af152951a48998eb851120c4bb07d361deda17ed28042baf45,
copied once to a fresh local destination and hash-verified. Independent fresh
restores shared-122103 and overlay-122103 under
/tmp/scry-foundation-closure-sep06.8IEPu5 both preserve249613 raw rows exactly,
digest d037654b89b31a482c31539498593fd5cfb83fb202d53121899425d02eb343fd.
Counts: en31535, FA82420/current74512/historical7908, EP9491, pq35, adj56820,
al53222, ar4, att11423, cur3203, meta5, rs19, rt19, ve1417.

The probe opens only overlay-122103 with synthetic original input and fixture
identities. A specified existing Scry identity must defer before private adoption
and resolve as legacy afterward, while independent synthetic proposed utility
survives. No fact/identity proposal ever materializes through the planner.

| Stage | Raw rows / digest | Candidates / projections / witnesses / decisions | Duration |
| --- | --- | --- | --- |
| Unadopted | 249613 / d037654b | 2 / 2 / 29 / 5 | 2.875326667s |
| Unadopted reopened | same | 2 / 2 / 29 / 5 | 2.904008709s |
| Privately adopted | 281149 / eb75ad9d | 2 / 3 / 639 / 6 | 2.983413459s |
| Privately adopted reopened | same | 2 / 3 / 639 / 6 | 2.884705584s |

Each call asserts complete raw equality, zero identity/fact writer history and
zero events, with neither created nor final-present candidates. Private adoption
preview changes nothing; explicit PRIVATE apply adds31536 anchors/marker only,
preserving every original row. Its manifest16,484,727bytes SHA
7911db3a61151d25df0dd706b8056d3fa3510cb3d65989d088252d71bf1be543.
Post-adoption raw digest
eb75ad9d042332df6d54cdbb86406376f112bcc6e9e4c7205badf07604c0eade.
shared-122103 stays untouched. overlay-122103 is now privately adopted; do not
rerun an untouched-snapshot precondition there or confuse it with the live store.

CGO_ENABLED=0 SCRY_PRIVATE_OVERLAY_REPLICA=122103 go test ./internal/memory/store
-run TestPrivateOverlayRestored122103 -count=1 -v, with TMPDIR inside probe export.
PASS first run, package16.906s. Log actual-sixth-replica.log remains in that export,
SHA623bd5d88c13db55df07633d2585ad4cc0dc5b8507cdfba2f7ba10b4ae762b0a.
No fixture correction or source adjustment. Only allowlisted static outcomes,
counts, digests and timings are printed, never actual stored content or raw errors.

Separately, safe note lookup on untouched shared-122103 verifies previously accepted
bb92111e4194a567c5d7ac71a1d33fc3da784ef3842390723bd4a2f27152ae58 is now INGESTED,
absent pending/not parked, actual EP rawSHA
74f6b72f0358dfb32b9524e5b636093b4652ba6c9b2c08a1611568b6c30bc659.
No retry occurred. Earlier acc0f253 remains closed as parked, not successful ingest.
Live binaries/store, providers and configuration unchanged. Fresh independent sixth
code review is running and these root measurements do not preempt its verdict.
