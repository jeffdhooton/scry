# Episode selector — root evidence, not independent approval

2026-09-06 08:56 UTC. Candidate exported e13f7ae. Full baseline blob comparison:
576 files checked;575 unchanged, only identity_legacy_adoption.go modified by
adding exact io-episode:/io-head: reserved-prefix refusal. Eight new selector
source/test files. All original tests unchanged. Independent code review pending.

Frozen design3fd0d796, complete design review3ee87a6b, corrected executable
contractb2b935484c4decd70779db1cded42c6048b5484aa4df18ab16ba7f3779d07265.
Four explicit corrections: own canonical link/reference ordering; local-only
head counter read claim; private writer returns staged data, not actual commit
success; internally derived historical full-byte-equal content dedup allowed,
without searching history for authority. Root read complete review/test before code.

Source pins, internal/memory/store/:

- identity_episode_selection.go:9566d0d233e5d0dcd32ae5694a8457746354119ff85e2abe1bc982668d221aae
- identity_episode_selection_validate.go:db8a01a1c8e19e5f8c58cee80e244504681f2c5b9b4d00115660b66137981024
- identity_episode_selection_write.go:e312424acda1c797bbcdb5e4e68b7f652f9775021d95549fc5d36b51a9f951fb
- identity_episode_selection_read.go:bcc2bc0c5e911f0e70443f3de35d109e2fab600a46a42fc12e38602ce0d24b78
- identity_legacy_adoption.go:67f1ddfd09774698181019ec07c2a91d228d57a82d771bb304cb425f7d206b2b
- identity_episode_selection_test.go:e78c5fd53ab09d09f52ae7630ee82b0721e49fa4435864d0565e7cff1e04babb
- identity_episode_selection_boundaries_test.go:29fbe21de4f90b9e5747a1a6118bffc53039488a2f330821784b97b488a621df
- identity_episode_selection_storage_test.go:16e94817f308c8432a9ddb216c2e9ed1e28dbab965b5c4ce485417dd0a4acb4e
- identity_episode_selection_completeness_test.go:9ac34d50a8a52ba14a207b4b88b12f9921ce36a722d706f8399f9a5ac2d0d154

Root had no failed tests or source corrections. Successive expanded noncached
targeted runs PASS0.533s/1.634s/1.974s/2.154s. Final13 top-level groups include
full input/slot/birth/link coverage, mixed revisions, canonical permutations,
exact retry, stale identical CAS, same-input improvement, A/B/A omitted-lineage
dedup, nil/empty input, no-birth hint deferral, all closed reasons, local head
counter limitation/overflow, current and immediate predecessor validation,
actual oversized/capacity/commit failures and panic, long-ID token/chunks under
head changes, reopen, adopter malformed/present-empty reserved-family refusal.

Logs:

- episode-selection-initial-targeted.log:d937f54dd6ea4cb5a9a06a137884f9d9c0252ff57d74e480434efafef0c81606
- episode-selection-boundaries.log:77dde220ba986ab4b71f373dc47fbf32d05660637488034c60cd4414842eac44
- episode-selection-storage.log:7ff81a209983fc3fd41e8e3de3b4289d3655dba62ba76fde8c32dcd514851469
- episode-selection-final-targeted.log:3ae2aad464f9f48f12cfb8a86088d88b4c964717da1f39078fc39f8f46f451f2
- episode-selection-full.log:eeae313baab888aece6f455a3ce25c40f288ed7d0d9c3fa8207b3630855f53fa

Full CGO_ENABLED=0 go test ./... -count=1 PASS51.402s store/16.377s resolve/
30.998s daemon, exit0. Full logs include stderr and pipefail. Root no-CGO store
vet PASS. These are builder results, never substituted for independent execution.

Fresh Mini backup /Users/jclaw/.scry/backups/memory-20260906T084717Z.badger:
76,526,706bytes SHA9d3e5c501dab85a1a2378dc5f90bd6278013aa3b35e218b2c624fd997dcf5b1f.
Copied to /tmp/scry-foundation-closure-sep06.8IEPu5; two separate fresh restores
shared-084717 and selection-084717. Both all248,586raw rows exact through direct
load/Open/index/read, digest
3ae18eea90136dc6adb797f935ef5182f004fe741e3d7026251e3f654d06faa8.
en31396 FA82130 current74237 historical7893 ep9475 pq30 adj56618 al53044
ar4 att11325 cur3194 meta5 rs19 rt19 ve1327. No live graph mutation/deployment.

Previously accepted notecaaab1fa6b21d0eee0e98a152395624aece5f21be31b57d16278c2eb8e58fa83
confirmed ingested/absentpending in shared-084717; EP raw
3f1b81aba66e9f6cc364e2f58eaafa5da4ff62d96da77a0ba95242fb837aa64e. No retry.

ROOT-ONLY synthetic structural canary on selection-084717 preserves every248,586
old raw row and adds exactly9 auxiliary rows: ep1/io-input1/io3/io-result2/
io-episode1/io-head1; no en/al/fa/generation writes. Exact retry changes no rows;
close/reopen preserves every old and new row. Selector emits0events. Synthetic
fixture input is inserted separately; this is NOT production atomic EP/Apply or
real extracted-birth/support proof. Selector-only duration465.458microseconds,
package1.340s, not ingestion throughput/remember-p95. Added sorted framed digest
76dd81f385570e00ac56aecfa7db90570ac35fd20d2be5d652004f3a517d0cd4.
Private probe internal/memory/store/private_selection_replica_test.go
SHA063d68ec5156d0406229c182e05b8373ec1a05e1d04b5e3c92eca3318e829465
MUST NEVER integrate. Log selection-root-replica-canary.log
SHA98b014fa4f8fd1d6ecd1c85975d3042b5165c1ef2a36510373ebd4a61f38bdae.

The consensus skill's prescribed Opus/Sonnet Agent interface is unavailable in this
session. Existing fresh-context design/code reviews satisfy the goal's bounded
disproof workflow, not multi-model consensus. No consensus verdict is claimed.

Installed a06cd7b unchanged on both hosts. No live adoption, apply, sweep, provider
probe/configuration change or deployment. Full production registration/support/
dependency/alias/lifecycle/Force/status/cleanup/recall/two-round goals remain open.
