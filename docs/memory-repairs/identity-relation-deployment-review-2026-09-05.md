# Independent deployment gate — 2026-09-05

Verdict: bounded PASS for candidate `393eeec79f80d3b4becff276c4fcffd71fa68ac5`, artifact `/tmp/scry-fallback-deploy-sep05.5OX7ef/scry`, SHA-256 `acfb78186402aec9eef87e71e0b81f6edeac1b9ea46641efd417aaf4368f0bd0`. No concrete deployment-scope, backup, or binary-discipline violation was found. This does not certify the full memory goal, the live ingestion/recall bar, or any further semantic repair.

## Contract and scope

Started with `scry memory orient --cwd .`. The complete objective/house-rule/autonomy contract was read earlier in this review session; its current SHA-256 `58d86de427f42022dec6bd7f77fa74f3b3f6285d386c87167ad4e874a70ffc62` is identical to that earlier archive. It expressly allows laptop/Mini deployment after verified nonempty backups and retained previous binaries, with fresh independent disproof before deployment.

Inspected all production diffs from deployed `53fafa9`: bounded normal-write relation-identity guard, fallback evidence preservation, and durable queue parking for exact fact conflicts. Other changes are tests and evidence/decision documents. There is no new schema, migration execution, historical fact rewrite, entity merge, alias transfer, provider, telemetry, hosted embedding, transcript retention, hook/configuration change, push, or unrelated restart in the deployment change. Architectural reasoning is recorded in `docs/DECISIONS.md`.

The independent code regrade `/tmp/scry-fallback-independent-sep05.C4aYAp/REGRADE.md` specifically passes `393eeec` after the rejected `8c2a05d` evidence-loss case was fixed. It includes fresh restored-live-backup probes, independent adverse queue/transaction/key-conflict cases, 2,154 relation boundary cases and the unchanged 39-relation vocabulary. I read that complete report and inspected the bounded source diff; this deployment review does not claim to have rerun its entire adversarial corpus. `fallback.go` matches the independently reviewed archive at SHA-256 `a74bc4e7007f7668212c51b08a21d6b6fbd35cfdcbbf99a5c7e1efd39c57fe3b`.

## Independent source/build/test verification

- Compared all 337 tracked entries in the deployment archive against exact candidate Git blob hashes: zero mismatches.
- Rebuilt independently from the verified archive into `/tmp/scry-fallback-deployment-gate.vtMBbn/scry-rebuilt`, using `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=393eeec' -o ... ./cmd/scry`.
- Independent rebuild is byte-identical to the candidate: SHA-256 `acfb78186402aec9eef87e71e0b81f6edeac1b9ea46641efd417aaf4368f0bd0`.
- Build metadata verifies CGO disabled, darwin/arm64 and trimpath; `file` reports arm64 Mach-O; `codesign --verify --verbose=2` passes; candidate version reports `scry 393eeec`.
- Independently ran `go test ./...` in the exact deployment archive: all packages passed, exit 0.
- Shared HEAD advanced to `3fe1cc64d6a8a9d42de7828d5672fafbb0a37048` during review. Its diff from candidate changes only the ChildScribe manifest and review documents, not production/build source. The certified artifact remains the exact archived `393eeec`, not an arbitrary later rebuild. Existing untracked work was preserved.

## Verified recovery state before replacement

Both currently installed binaries report `53fafa9`. Both installed files and both retained executable files hash to `31f185d70e1439a315a8ea12eaadf4f12d75e77d558edfa0852c890531705aff`.

| Machine | Retained previous binary | Verified size and mode |
| --- | --- | --- |
| Laptop | `/Users/jeff/go/bin/scry.pre-393eeec-20260905T1939Z` | 32,939,778 bytes; executable `-rwxr-xr-x` |
| Mini | `/Users/jclaw/.local/bin/scry.pre-393eeec-20260905T1939Z` | 32,939,778 bytes; executable `-rwxr-xr-x` |

Mini backup `/Users/jclaw/.scry/backups/memory-20260905T193835Z.badger` is 77,468,362 bytes with SHA-256 `88fd5b78303115050d13819701957a871a1fa7a4ab761418c3c42084b43e0abe`. Verified the actual remote file and exact local copy `/tmp/scry-fallback-deploy-sep05.5OX7ef/mini-before-deploy.badger` independently.

Laptop backup `/Users/jeff/.scry/backups/memory-20260905T193836Z.badger` is 19,445,000 bytes with SHA-256 `67a2cb2852fc24ed2a10bc9c837183aeeee7f463c747a75c80552987f86380dd`. Both backups are regular nonempty files with mode `-rw-------`.

Independently restored both backups with Badger Load into new, previously nonexistent databases here; load, iteration and close all succeeded:

| Backup | Entities | Episodes | Facts | Total keys |
| --- | ---: | ---: | ---: | ---: |
| Mini | 30,355 | 9,305 | 79,953 | 240,566 |
| Laptop | 14,200 | 2,689 | 21,004 | 83,378 |

The Mini count is from the newer deployment backup; the lead's earlier 79,926-fact ChildScribe receipt is a different snapshot. This review neither rewrote those facts nor regraded that semantic batch. The independent restore program is retained as `restore.go` in this directory.

## Allowed deployment and outstanding checks

Verified both exact launchd services are running. Their ProgramArguments execute `/Users/jeff/go/bin/scry` and `/Users/jclaw/.local/bin/scry`, respectively. The bounded plan is atomic replacement with the same reviewed artifact, then restart only `gui/501/com.jhoot.scryd` on laptop and `gui/501/ai.jermes.scryd` on Mini. Keep all retained binaries and backups.

After installation the lead must still verify actual installed and running versions/hashes, queue/default configuration behavior, the five benchmark suites and operational health, and append/post deployment evidence as required by the existing contract. This change deliberately preserves and parks exact fact conflicts; additional parked episodes remain unfinished work and cannot be counted as successful ingestion. This predeployment verdict cannot certify those future measurements.

No further semantic repair or historical rewrite is part of this deployment. Every later live repair continues to require its own fresh backup, exact manifest inputs, independent review and dry-run-first verification.

Reviewer performed no live-store writes, daemon restarts, installations, model/remember calls, shared repository edits, or configuration changes. Writes were confined to this temporary review directory and standard build/test caches.
