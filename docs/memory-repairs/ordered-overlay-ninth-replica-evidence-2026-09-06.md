# Ninth ordered overlay — root restored-replica evidence

2026-09-06. Bounded builder conservation check, not independent approval,
normal-write prevention, support/ownership closure, p95 or a full-goal round.

Exact candidate /tmp/scry-ordered-overlay-ninth-sep06.nAculd/code, freeze
2e06afc117db9c961924dffc93202f816b2f9523e53a29997c364c6195247dbb.
Separate root probe export /tmp/scry-overlay-ninth-replica-sep06.dUF8Nm/code
verifies all600 baseline SHA/git blobs and54 addition SHA/byte pins.
Only extra source private_overlay_replica_test.go SHA
f8bf99c037c221888e9a8f963cc19aaac01fba342b2c50edae31927766d4957d.
NEVER SHIP this probe. All authored source/copies/scripts use apply_patch.

Fresh Mini backup /Users/jclaw/.scry/backups/memory-20260906T132039Z.badger,
77,341,054bytes SHA
cb1f9eef28b018a5682193043bbbb27e6a2ce169aaf78e931e33f187c3e2992e,
copied once to fresh local destination and remote/local hashes verified.
Independent fresh restores shared-132039 and overlay-132039 under
/tmp/scry-foundation-closure-sep06.8IEPu5 both preserve250029 raw rows exactly,
digest5916d0aa12ae4bacb9a2c4d764972bd32db773be4f4bcef1487c456a82524d9e.
en31595, FA82532/current74622/history7910, EP9498, pq36, adj56888, al53302,
ar4, att11457, cur3208, meta5, rs19, rt19, ve1466.

Only overlay-132039 is opened by the explicit root probe. Original synthetic input
tests specified existing Scry metadata and two new fixture identities. Before
private adoption the old identity defers while unrelated synthetic proposals remain
usable. After private adoption it routes as legacy. No proposal materializes.

| Stage | Rows / digest | Candidates / projections / witnesses / decisions | Duration |
| --- | --- | --- | --- |
| Unadopted | 250029 / 5916d0aa | 2 / 2 / 30 / 5 | 3.780426542s |
| Unadopted reopened | same | 2 / 2 / 30 / 5 | 3.737900209s |
| Privately adopted | 281625 / 7c47d6d1 | 2 / 3 / 640 / 6 | 3.765633125s |
| Privately adopted reopened | same | 2 / 3 / 640 / 6 | 3.811915208s |

Every planner call checks full raw equality, zero identity/fact writer history,
zero emitted events and no created/final-present candidates. The probe does not
independently inspect individual new ReferenceViews; synthetic unit tests cover
those, and fresh whole-unit review is separate. These timings are single private
planning observations, not normal Apply/remember p95 or cross-version comparison.

Private adoption preview changes nothing. Explicit PRIVATE apply adds31596 anchors/
marker only and preserves all250029 original raw rows. Manifest16,523,323bytes SHA
8038a1d627bbe436b46050290c77abe8161a6858aa0a025aaca93a3b1279401f.
Post-adoption digest
7c47d6d173d45b56a8820beb487220608271837aa5c0ce0982a21ed3f20650b4.
shared-132039 remains untouched; overlay-132039 is now privately adopted. Do not
rerun untouched-snapshot preconditions there or confuse either copy with live state.

CGO_ENABLED=0 SCRY_PRIVATE_OVERLAY_REPLICA=132039 go test ./internal/memory/store
-run TestPrivateOverlayRestored132039 -count=1 -v, TMPDIR inside probe export.
PASS first run20.282s package/19.86s test. Complete log actual-ninth-replica.log is
retained at /tmp/scry-overlay-ninth-replica-sep06.dUF8Nm, SHA
4fc9d01f7e6b63ef1d15b1e8dc93589b2692555e6e82b549f46fb54c32bec08c.
No fixture correction.
Only static allowlisted outcomes, counts, hashes and timings are printed, never
actual fact/description/source text or raw errors. No live graph mutation/deploy.

Separate safe note lookup on untouched shared-132039 confirms
059723bc63eaff3c9eab2772a5d4a4ce98419cfa1fb98c1bc2c02f13990182e6
is INGESTED, absent pending/not parked; actual EP raw SHA
d851156f43a76abac048dd5f9fd476b48e54156d202d739a895a74ae38c2211f.
No retry occurred. Earlier bb92111e was already verified ingested122103; acc0f253
remains closed as parked, not successful ingestion. Ninth independent review is
still in progress and these root checks cannot preempt its verdict.
