# Independent deployment discipline gate: PASS

Date: 2026-09-05. This is a bounded predeployment review, not a deployment receipt or full memory-quality goal pass.

Candidate: `62cf6e0d2d8db9d324da23114df837848e972c8f`.
Reviewed artifact: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/scry`.
Artifact SHA-256: `821358499706bd9388b63a4368bb5320fc1bac61f12c93df48a158667b19bc14`.

## Authorization and scope

I ran memory orientation first and read the complete attached goal objective, supplied AGENTS instructions, and the complete independent code/replica PASS at `/tmp/scry-canonical-finalgate-sep05.uyfOSP/REVIEW.md`. No additional AGENTS.md existed in the checked filesystem ancestor chain. The goal explicitly permits laptop/Mini builds, deployment, backups, and restart of the two launchd daemons when a retained previous binary and verified nonempty backup exist first, with independent review before deployment.

This PASS covers only staging the exact reviewed artifact next to each installed binary, checking its hash/mode, atomically replacing `/Users/jeff/go/bin/scry` and `/Users/jclaw/.local/bin/scry`, and restarting `gui/501/com.jhoot.scryd` and `gui/501/ai.jermes.scryd`, respectively. Both launchd targets were independently read and point to those exact binaries. Preserve all backup and previous-binary files. There is no provider/configuration change, queue retry, semantic repair, manifest application, migration, hook change, or unrelated deployment covered by this gate.

The lead remains responsible for sequencing, live integration and room receipts. I made no shared-repository edits, live writes, restarts, config edits, provider calls or remember calls. The shared worktree's existing untracked workflow assessment was left untouched. All new review artifacts and restored databases are confined to this temporary review directory.

## Exact source, build and production diff

- Independently archived the exact commit. Verified all **356 tracked blobs** against Git object hashes both in my archive and in the supplied artifact's sibling `code` directory. The supplied archive has no extra files; executable tracked modes were checked. Evidence: `verify-blobs.rb`, `blobs.log`.
- Independently rebuilt using `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags='-s -w -X main.Version=62cf6e0' -o independently-built-scry ./cmd/scry`. `cmp` proved byte identity with the proposed artifact. Go metadata confirms Go 1.26.2, no CGO, darwin/arm64 and trimpath. `codesign --verify --verbose=2` passes.
- This is the established single no-CGO macOS executable. Strictly, it is not a fully static Mach-O: `otool -L` shows the same four Apple system libraries/frameworks as the installed previous binary (libSystem, libresolv, CoreFoundation, Security). No runtime dependency was added; the repository's documented build convention calls this CGO-free build static.
- `go test ./...` on the exact archive passed, exit 0. The later independent restoration helper is in `_deployment_probe`, which Go's recursive package discovery excludes, and was not part of the candidate build or full-suite test surface.
- Inspected the entire production diff from live `393eeec79f80d3b4becff276c4fcffd71fa68ac5`: only `internal/memory/resolve/resolve.go` changes production code (45 additions, two deletions); the other Go change is canonical-name regression tests. Remaining changes are documentation/review/manifest artifacts. No dependency, persistence schema, daemon startup, provider, telemetry, retention or embedding code changed.
- The canonical-name exception is limited by existing exact natural-slug precedence, listed ownership, generic/reference checks, and transaction-visible canonical homonym refusal. The independent code/replica report covers prior failed homonym and normalized-reference cases, alias safety, historical fact preservation and real-name regressions. Its three published test-log hashes independently match. Neither rejected predecessor is being deployed separately.

## Live installed binaries and rollback

Read-only local checks and actual `ssh jclaw@mini` checks verified regular executable installed and retained files. All four SHA-256 hashes equal `acfb78186402aec9eef87e71e0b81f6edeac1b9ea46641efd417aaf4368f0bd0`:

- Laptop installed `/Users/jeff/go/bin/scry`.
- Laptop retained `/Users/jeff/go/bin/scry.pre-62cf6e0-20260905T2043Z`.
- Mini installed `/Users/jclaw/.local/bin/scry`.
- Mini retained `/Users/jclaw/.local/bin/scry.pre-62cf6e0-20260905T2043Z`.

Both daemons were running on review (laptop PID 40300, Mini PID 99227). These are predeployment observations only. A binary rollback can atomically reinstall the retained executable and restart the same service. Binary rollback does not reverse facts ingested after deployment; any store rollback must preserve later work and receive its own explicit, scoped review rather than blindly restoring an older snapshot.

## Verified recovery backups

Independently checked source files and downloaded copies as nonempty regular files, with sizes and SHA-256:

| Host | Actual source | Bytes | SHA-256 |
|---|---|---:|---|
| Laptop | `/Users/jeff/.scry/backups/memory-20260905T204304Z.badger` | 19,445,032 | `a760b1a7683f8f098872bc5cb2d3765508bec00aa8117a2f82e8401d951c7c16` |
| Mini | `/Users/jclaw/.scry/backups/memory-20260905T204303Z.badger` | 77,937,531 | `ef5d6bd0536ef59723a88ed99978b3897112d6844f55f7646b05ef90f1681685` |

Matching copied streams: `/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/laptop-before-deploy.badger` and `mini-before-deploy.badger`.

I independently restored **both complete streams** with Badger Load into separate fresh replicas, checked the consumed stream length equals the entire source size, enumerated every restored raw key and value into a length-framed SHA-256, closed/reopened each using the candidate Store, and decoded all entities, current/historical facts, aliases and pending entries. Both completed successfully. Evidence: `_deployment_probe/main.go`, `restore.log`, `laptop-replica/`, `mini-replica/`.

| Baseline | Laptop | Mini |
|---|---:|---:|
| Raw keys | 83,378 | 241,548 |
| Episodes | 2,689 | 9,326 |
| Entities | 14,200 | 30,453 |
| Facts (all history) | 21,004 | 80,258 |
| Invalidated facts | 2,509 | 7,795 |
| Alias claims | 23,159 | 51,771 |
| Ready / backoff / parked at 20:43:05Z | 0 / 0 / 0 | 1 / 0 / 6 |

Laptop raw key-prefix inventory: `adj=21004, al=23159, cur=1321, en=14200, ep=2689, fa=21004, meta=1`.
Mini raw key-prefix inventory: `adj=55417, al=51771, att=10773, cur=3115, en=30453, ep=9326, fa=80258, meta=5, pq=7, rs=9, rt=9, ve=405`.

Restored raw key/value SHA-256:

- Laptop: `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`.
- Mini: `21c6bc60ba0d17a22432c117aaa51cd75131fa88fbfb892045cc78d74e75c7c7`.

The laptop backup is a distinct local database, not a second copy of the default CLI-forwarded Mini database. Its dormant local-store state is expected; this gate gives no authority to change provider configuration. The Mini queue is **not quiet**. The earlier code grader's 80,203-fact fixture is distinct from this 80,258-fact deployment backup; neither proves that a parked provider response has been replayed.

## Outstanding live checks

After installation, the lead must verify both installed hashes, executable signatures/versions, restarted launchd PIDs and service health; confirm the laptop remains in its intended mode; and inspect Mini ingestion/queue errors without silently retrying parked work. Collect the separately assigned immediate pre/post five-suite measurements and append actual deployment evidence and reviewer verdicts to the audit/handoff and authorized room. Any proven regression requires investigation/rollback within the goal's authority.

This review does not grade live recall floors, held-out recall, remember p95, two real sweeps, zero graph defects, or the complete goal's two consecutive rounds. Linear canonical scans measured by the code grader are offline cost evidence only and still require live performance observation.

## Evidence hashes

- `full-suite.log`: `a8d38d808ac700d4158520ca41a45175cebfaac5ed41c0972fb3b84788d0c93f`.
- `restore.log`: `5828e8a1060d2b212f7a077da584a15e85354d980b13fa5c13a3b62ad6dd27c6`.
- `independently-built-scry`: `821358499706bd9388b63a4368bb5320fc1bac61f12c93df48a158667b19bc14`.

No deployment-discipline disproof was found within this exact artifact and scope. **PASS for the proposed bounded deployment; all live acceptance checks remain outstanding.**
