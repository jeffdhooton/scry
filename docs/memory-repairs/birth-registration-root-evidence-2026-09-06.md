# Birth registration — root evidence, not independent approval

2026-09-06 08:24 UTC. Private export of e13f7ae; all576 tracked baseline blobs
match git object bytes. Four new Go files only; no production callers or live
deployment. Independent code review is pending separately.

Initial targeted run failed seven deferral cases BEFORE registration: public
PutFact rejected the intentionally absent source. Corrected only the new fixture
to seed an exact raw legacy dangling fact. Source unchanged. Initial failure log
retained SHA7afef8d679b84c597fb0a48e6eabb34913f80814f37762baa68c37b4395fb2b7.

Expanded boundary run PASS1.869s, log
d481d55735362ac8fac01f60fe4555180b7837640a367fd4dd993637c712ab22.
Final targeted12 top-level groups PASS1.929s, log
a25c0a58a386c28b3f4b3c431e7f2529191b18e22b78a9d3f36a65ae1011f826.
Includes actual capacity/commit failures, body/observer panic distinctions, canonical
input/role/handle checks, prior/transient history, stable creation tuple, mixed
normal deferral, complete owned reports, raw reopen and explicit replay limitation.

`CGO_ENABLED=0 go test ./... -count=1` PASS exit0, store51.182s,
resolve15.810s, daemon30.191s. Full log
108d975d88f94c69cfe3dd095af9144a0f7a841177d18edf16f53cf450997dd6.
`CGO_ENABLED=0 go vet ./internal/memory/store` PASS exit0.

Frozen contract e512d77638a6667f150db4f2aa19c33709a6acfccd87b44f90e1f46624288f15;
source509faba40f6e3f56710d2475b50bddd8091375decff5d80e7cb3aca6147ccb02;
supplied main testffd97200bc267672be00bd938046ef84acb9c67758d7875494b49fede775410b;
boundary testef5339c7dfb787276d137df0d77dc22ec7c3748077090678fd8f07e854a71fce;
storage test6085a0ec28947cfe7b2d552bd76967b11f3ece9b4daab120034b62176b9f34fc.
Design19474eeb and design reviewcd07b5b4 remain unchanged. Contract clarifies
body panic rolls back but observer panic is after commit under existing semantics.

Fresh Mini backup /Users/jclaw/.scry/backups/memory-20260906T081846Z.badger:
76,492,348bytes SHA5111bb78a6bd9f33d55f4bff8f67f358a84d212c48ac5d16565933b1e548983c.
Copied exact to /tmp/scry-foundation-closure-sep06.8IEPu5 and restored fresh
shared-081846. DirectLoad/Open/index/read all248,468raw rows identical,
digest109d8a5d1dd5d8c2fcc4fed7677949a365ad85012ebea6499fd9d4f328ec2ef4.
en31374 FA82099 current74206 historical7893 ep9472 pending32;
adj56591 al53017 ar4 att11319 cur3194 meta5 rs19 rt19 ve1323.
This is exact backup/open preservation, not new candidate registration on real data
or a graph-quality/latency grade. Installed a06cd7b remains unchanged on both hosts.

Additional ROOT-ONLY opt-in actual-replica test runs one pure synthetic candidate
registration and exact duplicate mention against shared-081846, with no entity,
fact, observation or other write. Whole fixed scope2.606648s, package3.477s;
all248,468raw rows identical, events0, registered1/materialized0.
Identity inventory84433rows/31374entities digest
fb5e2bb31e53204a8b57600c7d3d30e168bd5e26f5136db575f3ef9a9307b593;
FA82099 digest8cd98448cdac4691e871a12ee93d907793ce489f8347ce88163092436692b225.
Private testf43933d22509f416474765e095737b9983855ef3b993f70de41d29363cc63f69
is internal/memory/store/private_birth_replica_test.go and MUST NEVER integrate.
Log birth-root-replica-read-only.log
0d26e2ec696eab73c35d021b50691c0cc932ffc367e2a26349a34e17165185b4.
This is builder-only read-only compatibility/cost evidence, not independent actual
graph mutation, support, throughput or remember-p95 proof.

Previously accepted note3da1026d61249d973e025d20986632755a77b5e707a7f8083df98a5c3655819f
now confirmed ingested/absentpending in that replica; EP raw SHA
c30c8f9d6c07895646dd1c2480c434eeb91cb8fa90b0cd0ec9ad14102473cd2c.
No retry. No live repair, adoption, sweep, provider/config change or deployment.
Registered unsupported entities can still commit under this private accounting
contract. Actual support/undo/B/current-result/lifecycle policy remain open.
