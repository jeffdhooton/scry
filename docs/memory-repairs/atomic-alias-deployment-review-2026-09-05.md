# Independent deployment gate — 2026-09-05

Verdict: bounded PASS for deploying commit `53fafa91d621190c87245f0b0844270b4fdf44c9`, artifact `/tmp/scry-unalias-deploy-sep05.Od8oCm/scry`, SHA-256 `31f185d70e1439a315a8ea12eaadf4f12d75e77d558edfa0852c890531705aff`. No concrete scope, backup, or binary-discipline blocker was found. This is a predeployment gate, not a live repair authorization or certification of the full memory goal.

The reviewer began with `scry memory orient --cwd .`, recalled the deployment runbook, and read the complete goal contract, including its house rules and autonomy. The requested bounded grading delegation and laptop/Mini deployment are expressly authorized. No live writes, deploys, restarts, remember/model calls, configuration edits, or repository edits were performed. Verification writes were confined to this temporary directory and normal Go build/test caches; restored stores were new temporary databases.

## Source and artifact

- Repository HEAD was the specified full SHA. Initial worktree change was only the existing untracked workflow assessment. Later audit/handoff changes appeared while the lead continued; this reviewer did not edit them.
- Independently compared all 329 tracked archive entries, using Git blob hashes, to the deployment archive: zero mismatches.
- Independently rebuilt from that archive with `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=53fafa9'`, writing `scry-rebuilt` here. Its SHA-256 is byte-identical to the candidate above.
- `go version -m` verifies Go 1.26.2, CGO disabled, darwin/arm64, and trimpath. `file` verifies arm64 Mach-O; `codesign --verify --verbose=2` passes. `scry version` reports `53fafa9`.
- Independently ran `go test ./...` in the exact deployment archive: exit 0, all packages pass.
- Production diff from deployed `af77a6a` is confined to reviewed alias-repair implementation and CLI/RPC routing. Other changes are tests and goal evidence/documentation. No ranking, providers, hooks, transcript retention, telemetry, module dependencies, or unrelated product deployment changes appear in that diff.
- Store algorithm SHA-256 is `2f3c74bde5e4f6f9ccaaf902dcbb98befa2ad6e77417b92983d95bd48affc506` in the deployment, independent code-review, and replica archives. Reviewed `/tmp/scry-unalias-regate.gRhAmY/INDEPENDENT_REVIEW.md`: bounded PASS specifically covers this SHA after the daemon downgrade flaw was fixed with a distinct guarded apply RPC method.

## Recoverability before replacement

Both installed binaries still reported `af77a6a`. Their SHA-256 and each retained executable's SHA-256 are `32fdcbad15dd0bb2f87a9987e07ecb887c1fefa7be2a2dbe4a299bfe5f2084e9`.

- Laptop installed `/Users/jeff/go/bin/scry`; retained `/Users/jeff/go/bin/scry.pre-53fafa9-20260905T1856Z`, 32,922,834 bytes, executable.
- Mini installed `/Users/jclaw/.local/bin/scry`; retained `/Users/jclaw/.local/bin/scry.pre-53fafa9-20260905T1856Z`, 32,922,834 bytes, executable.
- Mini backup `/Users/jclaw/.scry/backups/memory-20260905T185317Z.badger`: 77,087,455 bytes, SHA-256 `d1e62fc142e348441ccf4c989070af1a906c1ad7ad8158fb65c272f34ac39aad`. Independently verified the remote file and exact local copy `/tmp/scry-unalias-deploy-sep05.Od8oCm/live-before-deploy.badger`.
- Laptop backup `/Users/jeff/.scry/backups/memory-20260905T185317Z.badger`: 19,445,024 bytes, SHA-256 `9f505bc3a1ea232c5198fc2e30e1dfabc24e3e12d6bb847910ef88f895cf981d`.
- Independently restored both backups using Badger Load into previously nonexistent local temporary directories. Mini: 30,224 entities, 9,291 episodes, 79,679 facts, 239,650 total keys. Laptop: 14,200 entities, 2,689 episodes, 21,004 facts, 83,378 total keys. Both loads and closes succeeded. The retained `restore.go` reproduces these checks against fresh target paths.

The refreshed replica measurement `/tmp/scry-unalias-replica-sep05.rp5vdY/refreshed/summary.json` names the exact current Mini backup. Its 49-row fixture preserves all 79,679 facts with 30,224 entities; the earlier measurement's 79,560 facts belong to the older 18:28 snapshot. The fixture driver checks full fact-object equality. I read that driver and measurement but did not independently reapply the 49 rows or grade their semantic ownership.

## Bounded execution scope and remaining verification

Both existing launchd labels are running and their program arguments execute the intended installed paths. Allowed restart targets are exactly `gui/501/com.jhoot.scryd` on laptop and `gui/501/ai.jermes.scryd` on Mini. Install the same reviewed artifact by atomic replacement, retaining both previous files and backups; do not rebuild separately per host or prune rollback files.

The lead's stated next checks—both installed/running versions and hashes, defaults and queue reasons, five benchmark suites, audit/handoff evidence and room verdict—remain necessary postdeployment work. This gate cannot certify future actions before they occur. A parked queue entry remains preserved pending work, not a successful drain.

No semantic store apply is included in this deployment verdict. The ChildScribe 49-row candidate requires a fresh postdeployment stable snapshot, regenerated exact expected fingerprints, separate semantic/manifest review and dry-run-first verification before its live apply. No graph-wide cleanup, old-binary cleanup, provider change, configuration/hook change, or unrelated restart is included.
