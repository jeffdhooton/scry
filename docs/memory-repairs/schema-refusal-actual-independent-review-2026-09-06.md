# Independent actual-deployment review: schema startup refusal

2026-09-06 UTC. Private review directory: `/tmp/scry-schema-actual-disproof.MJiRYo`.

Verdict: bounded actual-deployment PASS for the exact schema-1 startup-refusal artifact. I could not disprove deployment consistency, complete backup preservation, or retrieval equivalence within this scope: installed/retained hashes, independently rebuilt candidate, all four fresh direct restores, complete raw map/drift comparisons, exact-artifact five-suite comparisons on both post stores, and all 235 individual-question ranks/payloads per host pass. Whole-goal completion is rejected: the original held-out floors remain unmet. This review covers the immediate 01:59:08→02:03:46 UTC backup interval; it does not cover subsequent ingestion or sweeps.

## Independence and scope

I ran `scry memory orient --cwd .`, read the full active goal at `/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md`, and read these three complete predeployment reports: `docs/memory-repairs/schema-refusal-artifact-predeploy-review-2026-09-06.md`, `schema-refusal-fresh-predeploy-review-2026-09-06.md`, and `schema-refusal-immediate-predeploy-review-2026-09-06.md` in the same directory. I inspected the exact source diff and the restore, raw inventory, benchmark, rank, drift and queue-failure implementations. Lead-agent measurements were not used as proof; all measurements below are this reviewer's fresh executions.

Source was independently exported with `git archive a078240 | tar -x -C /tmp/scry-schema-actual-disproof.MJiRYo/source` and `git archive 24eafab | tar -x -C /tmp/scry-schema-actual-disproof.MJiRYo/baseline`. The candidate full revision is `a07824007973301066b36d1b97c398d9ea081a95`. Only `internal/memory/store/store.go` changes in production code; the remaining diff consists of tests and documentation. The schema remains 1. Recall, resolver, benchmark expectations and dependencies are unchanged. Failed restatement and historical-address experiments are absent from the production diff.

The previously graded independent evidence and drift helpers were read fully, copied into this private export, and rebuilt locally. Authored helper changes used apply_patch: hash unexpected errors instead of printing them; emit a hashed row for every added record; compare all pending input fields after excluding only retry metadata; and report fixed error predicates and text lengths. No raw stored keys, facts, values, source references, pending text or errors were emitted. Benchmark fact previews were replaced by hashes in memory before output. Helpers use direct Badger Load into nonexistent destinations, never populated application Restore.

## Actual installed and retained binaries

At 02:07:42 UTC, both installed binaries and this reviewer's independent rebuild had SHA-256 `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`. Both retained prior binaries had SHA-256 `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`.

| Host | Installed executable | PID | Command | launchd service | Start time |
|---|---|---:|---|---|---|
| Laptop | `/Users/jeff/go/bin/scry` | 46487 | `start --foreground` | `gui/501/com.jhoot.scryd` | 2026-09-06 02:03:31 UTC |
| Mini | `/Users/jclaw/.local/bin/scry` | 24189 | `start --foreground` | `gui/501/ai.jermes.scryd` | 2026-09-06 02:03:31 UTC |

The two retained paths are their respective installed paths plus `.pre-a078240-20260906T0146Z`. Both launchd services reported `state = running`, the same PID as ps, and last exit code 0. Their wrapper program is `/bin/zsh`; ps reports the expected Scry executable and arguments. These read-only observations support actual deployment at the reported 02:03:32 completion time; they are not a cryptographic measurement of resident process memory. No older rollback executable was modified or removed by this reviewer.

The independent rebuild command from the candidate export was:

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=a078240' -o /tmp/scry-schema-actual-disproof.MJiRYo/scry ./cmd/scry
```

Host checks used the exact paths above with `shasum -a 256`, `ps -p 46487 -o pid=,lstart=,comm=,args=` and the analogous Mini PID command over `ssh jclaw@mini`. `launchctl print` output was filtered before capture with `rg '^\s*(state =|pid =|program =|runs =|last exit code =)'`. No launchd environment or configuration contents were printed. Complete filtered output is `deployment-check.txt`.

## Four independently verified complete restores

Each full backup was hashed before its nonexistent destination was created. Badger Load consumed the complete stream. The helper then compared every active logical key and value byte before and after candidate Open, AllFacts, Entities, offline index construction and local recall. All four exact map comparisons passed. Digests length-frame each key and value in iterator order.

| Snapshot | Backup bytes | Backup SHA-256 | Raw rows | Raw digest |
|---|---:|---|---:|---|
| Shared pre 015908 | 74,792,395 | `e9a91a9a181bf45f65cb1d5034de80f975d55a14e3b0b710eedd70c2469b4b46` | 245,413 | `697dfbf8d61e6be8bb61ef0a728f8a1289df495fd9d72b18e2f5271deb8a4b06` |
| Shared post 020346 | 74,921,236 | `37187891a0bb081db2f35754df726432caba010e77fba5896c166c2ffa1b866e` | 245,425 | `022b8739f013ed98e2b9e96df7a117959fa533335bb43f5b32bfbd047a593f28` |
| Laptop pre 015908 | 19,445,008 | `01b4a03beea33c64fdcf30ab2da6cba9840c3595d6f6c10cb415535dacba3d21` | 83,378 | `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7` |
| Laptop post 020346 | 19,445,008 | `6c19b215d0108dac99ad0f1781c7de0686b973c24044a984827a3ec3827c7027` | 83,378 | `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7` |

Shared backup paths are `/tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T<timestamp>Z.badger`; laptop paths are `/Users/jeff/.scry/backups/memory-20260906T<timestamp>Z.badger`. Private destinations are `shared-pre`, `shared-post`, `laptop-pre`, and `laptop-post` beneath this review directory.

Shared pre and post each contain 81,304 facts: 73,449 current and 7,855 historical, plus 30,956 entities and 9,409 episodes. Laptop pre and post each contain 21,004 facts: 18,495 current and 2,509 historical, plus 14,200 entities and 2,689 episodes. The bounded local recall payload was 11,418 bytes on each shared snapshot and 9,409 on each laptop snapshot.

## Complete pre/post logical map accounting

There are zero removed keys on either host. Laptop maps are entirely byte-identical. Shared changes are exactly:

| Record family | Added | Changed | Removed |
|---|---:|---:|---:|
| Pending input | 10 | 1 | 0 |
| Scanner cursor | 2 | 2 | 0 |
| Metadata | 0 | 1 | 0 |
| Every other family | 0 | 0 | 0 |

Every one of the 81,304 preexisting shared fact records, including all 7,855 historical facts, is byte-identical at its original key. No fact text, validity timestamp, confidence, endpoint or provenance changed. All 30,956 entities, 52,428 alias-index claims, 9,409 episodes, 11,066 attestations, 56,091 reverse adjacencies, 942 value-evidence records and 42 repair markers (`ar` 4, `rt` 19, `rs` 19) are byte-identical. All 15 old parked pending records are fully byte-identical, including their source inputs and retry metadata.

The changed pending key SHA-256 is `71226390666152e0e1df6db32023e3b4a6b6ba6ce6d5ae5b9c59751c443a7dc8`. Its old payload SHA-256 is `dd8f30763b430c350360bf2a89ef074a3af6f31abbf8da4b73b69dfeb6d80f4f`; new payload is `826bc643aeafedaeb2e45bbbadaceb1077c033dd4de186ed8f33c2c648311749`. Only attempts, last_error and next_attempt changed. Attempts is now 1; it remains unparked. The entire non-retry input, including ID, text, source, source reference, cwd, cwd_is_repo, force, occurrence/enqueue timestamps and hints, compares equal. Its text is 1,781 bytes. The 74-byte error hashes to `1ac44e385cebdbc6ce1d38a5577fa921ee647f2c9513e498180f152e56f2444f`; the safe predicate for `deadline exceeded` is true, while `context canceled` and `connection refused` are false. This fits the existing queue.fail retry-metadata path in `internal/memory/queue/queue.go:467`, but no specific provider failure or restart cause is inferred from timing or this string predicate. There are no removed pending inputs requiring completion accounting.

All ten newly queued inputs are unparked, have attempts 0 and empty errors, and contain 116,559 total text bytes (individual lengths 4,020–15,788). Their complete raw records survive candidate Open/index/read and all old/current artifact benchmark invocations exactly. The evidence stores each new key and full-payload SHA-256, plus input hashes and lengths. This proves preservation of the complete input captured in the post backup; it does not independently reconstruct or certify the upstream transcript's completeness.

The two changed scanner records alter only mod_time, processed_bytes and size. The changed metadata is accounted by old/new hashes. `shared-drift-final.jsonl` contains a hash-only row for every added and changed key, and explicit empty removal accounting. Total scanner records grow 3,154→3,156; pending records grow 16→26 (15 parked, 11 unparked).

## Exact-artifact retrieval comparison on frozen post stores

Both actual installed/retained laptop artifacts ran all five unchanged suites against each same frozen private post store. All twenty CLI invocations succeeded. Complete result fields excluding latency compare equal between artifacts: hit totals, full miss membership/ranks and hashed previews, mean answer rank, mean/max payload, hit rate and cap counts. All active logical rows remain byte-identical before/after each ten-invocation matrix.

| Suite | Shared old = current | Shared mean rank | Shared max bytes | Laptop old = current | Laptop max bytes |
|---|---:|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 | 4.8431372549019605 | 12,112 | 22/62 | 10,578 |
| heldout-b | 29/66 | 5.068965517241379 | 13,366 | 13/66 | 11,034 |
| probes | 7/7 | 1 | 9,962 | 5/7 | 9,771 |
| tuning-strict | 45/50 | 4.511111111111111 | 11,533 | 28/50 | 10,470 |
| tuning | 47/50 | 3.978723404255319 | 11,533 | 30/50 | 10,470 |

Zero responses exceed 24,576 bytes. The original shared floors 53/62 and 34/66 remain failed; the other three original floors pass. No benchmark expectation was edited. `git diff 24eafab a078240 -- docs/memory-bench internal/memory/recall` is empty.

Individual-question rank helpers were compiled separately against baseline and candidate source. All four runs contain all 235 question indices/hashes, complete answer ranks including misses, payload bytes and a complete raw-preservation record. Baseline/candidate output files are byte-identical under cmp on each host's frozen post snapshot; both cmp commands exited 0. Shared and laptop complete raw maps remain unchanged. These source-built helper measurements complement the actual CLI benchmark runs; the shipped CLI itself emits full miss ranks and aggregate successful mean rank rather than every successful question rank. Both hosts' installed and retained executable hashes were checked again after all measurements and still matched the reviewed values.

## Reproduction commands

All source/helper edits are private. Build the evidence helper in each exported revision, and the drift helper in the candidate export:

```sh
CGO_ENABLED=0 go build -o /tmp/scry-schema-actual-disproof.MJiRYo/evidence ./cmd/independent-evidence
CGO_ENABLED=0 go build -o /tmp/scry-schema-actual-disproof.MJiRYo/baseline-evidence ./cmd/independent-evidence
CGO_ENABLED=0 go build -o /tmp/scry-schema-actual-disproof.MJiRYo/drift ./cmd/independent-drift
```

Exact direct-restore invocations (the destinations must not exist):

```sh
/tmp/scry-schema-actual-disproof.MJiRYo/evidence restore /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T015908Z.badger /tmp/scry-schema-actual-disproof.MJiRYo/shared-pre e9a91a9a181bf45f65cb1d5034de80f975d55a14e3b0b710eedd70c2469b4b46
/tmp/scry-schema-actual-disproof.MJiRYo/evidence restore /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T020346Z.badger /tmp/scry-schema-actual-disproof.MJiRYo/shared-post 37187891a0bb081db2f35754df726432caba010e77fba5896c166c2ffa1b866e
/tmp/scry-schema-actual-disproof.MJiRYo/evidence restore /Users/jeff/.scry/backups/memory-20260906T015908Z.badger /tmp/scry-schema-actual-disproof.MJiRYo/laptop-pre 01b4a03beea33c64fdcf30ab2da6cba9840c3595d6f6c10cb415535dacba3d21
/tmp/scry-schema-actual-disproof.MJiRYo/evidence restore /Users/jeff/.scry/backups/memory-20260906T020346Z.badger /tmp/scry-schema-actual-disproof.MJiRYo/laptop-post 6c19b215d0108dac99ad0f1781c7de0686b973c24044a984827a3ec3827c7027
/tmp/scry-schema-actual-disproof.MJiRYo/drift /tmp/scry-schema-actual-disproof.MJiRYo/shared-pre /tmp/scry-schema-actual-disproof.MJiRYo/shared-post
/tmp/scry-schema-actual-disproof.MJiRYo/drift /tmp/scry-schema-actual-disproof.MJiRYo/laptop-pre /tmp/scry-schema-actual-disproof.MJiRYo/laptop-post
```

Exact benchmark/rank commands, each serialized per destination to avoid concurrent Badger opens:

```sh
/tmp/scry-schema-actual-disproof.MJiRYo/evidence bench /tmp/scry-schema-actual-disproof.MJiRYo/shared-post /tmp/scry-schema-actual-disproof.MJiRYo/source /Users/jeff/go/bin/scry.pre-a078240-20260906T0146Z /Users/jeff/go/bin/scry
/tmp/scry-schema-actual-disproof.MJiRYo/evidence bench /tmp/scry-schema-actual-disproof.MJiRYo/laptop-post /tmp/scry-schema-actual-disproof.MJiRYo/source /Users/jeff/go/bin/scry.pre-a078240-20260906T0146Z /Users/jeff/go/bin/scry
/tmp/scry-schema-actual-disproof.MJiRYo/baseline-evidence ranks /tmp/scry-schema-actual-disproof.MJiRYo/shared-post /tmp/scry-schema-actual-disproof.MJiRYo/source
/tmp/scry-schema-actual-disproof.MJiRYo/evidence ranks /tmp/scry-schema-actual-disproof.MJiRYo/shared-post /tmp/scry-schema-actual-disproof.MJiRYo/source
/tmp/scry-schema-actual-disproof.MJiRYo/baseline-evidence ranks /tmp/scry-schema-actual-disproof.MJiRYo/laptop-post /tmp/scry-schema-actual-disproof.MJiRYo/source
/tmp/scry-schema-actual-disproof.MJiRYo/evidence ranks /tmp/scry-schema-actual-disproof.MJiRYo/laptop-post /tmp/scry-schema-actual-disproof.MJiRYo/source
cmp /tmp/scry-schema-actual-disproof.MJiRYo/shared-baseline-ranks.jsonl /tmp/scry-schema-actual-disproof.MJiRYo/shared-candidate-ranks.jsonl
cmp /tmp/scry-schema-actual-disproof.MJiRYo/laptop-baseline-ranks.jsonl /tmp/scry-schema-actual-disproof.MJiRYo/laptop-candidate-ranks.jsonl
```

Each bench helper invokes `<artifact> memory bench --dir <private-store> --file <candidate-export>/docs/memory-bench/<suite>.json --top 20`. Each command's output was redirected into the corresponding private JSON/JSONL below. Non-timing equality was additionally checked separately for each host using:

```sh
jq -sc 'map(select(.kind=="bench")) | group_by(.suite) | map({suite:.[0].suite, runs:length, equal_non_timing: ((.[0].result | del(.mean_latency_ms)) == (.[1].result | del(.mean_latency_ms)))})' <host>-benchmarks.jsonl
```

## Evidence hashes

All paths are relative to this review directory.

| Evidence | SHA-256 |
|---|---|
| deployment-check.txt | `547feeae173a996064365d06d91a4d7a7ca147e436151e77f4c8ef8941be239c` |
| shared-pre-restore.json | `5342a8b47ad6986ec28286d7608fe3d6097d7806dd7ea51b45f1268d9e2928d8` |
| shared-post-restore.json | `6c1ee939080a236957098123c115b9ab7aeb4ed177d07f5d6f97c5cde17ef218` |
| laptop-pre-restore.json | `ea8cc4cf91e06a821ff3ac8b22514955cedf250a4cf795b4b4777acb4620fce6` |
| laptop-post-restore.json | `4bb767668cf19c080de5dce2f935166935c32058de07136eafe45b3d8fbf4eaf` |
| shared-drift-final.jsonl | `cb1076974c67b46f04287e41da49fa44dddad5121329aa58a3afa13fcfd947ba` |
| laptop-drift.jsonl | `62494b9a3897e379bd594b3e278c344208e99d09717212da0b040cd65b92070b` |
| shared-benchmarks.jsonl | `11804c93f70525b26cab430dc42b2d8a0f95835be8e64d92dc1e43e1098716d7` |
| laptop-benchmarks.jsonl | `b7df418c817e04e23795d257b4113f749b31c6085e8fb9a9b51ae537d2485e79` |
| laptop-baseline-ranks.jsonl and laptop-candidate-ranks.jsonl | `cecda83f7e84cba9b8be8fdcac79b5d9d5d3d7784a08a4d8d44eed567aa66e5c` |
| shared-baseline-ranks.jsonl and shared-candidate-ranks.jsonl | `e50f2e835bccd590f56c0241702323a57ee58897a636803615fcd15a64e0a822` |
| source/cmd/independent-evidence/main.go and baseline/cmd/independent-evidence/main.go | `149118419f20933e043b05aca0d43741ad45bea3f43d66b3acd4e137c540b7fe` |
| source/cmd/independent-drift/main.go | `133f857cecf30bee082382c473fe4c9b62b8dffcfb572c49dd8073f037fee5a8` |

Unchanged suite file hashes: heldout-2026-09-03 `fa339bca6770a41e61ac4a3a481ca836887fc2c48fbf72265fe9f3ca8a5bf8c1`; heldout-b `ad92760e4314403a86108442b553e942f6c7b7a1ccdcb78e19f133653efae815`; probes `9a32bf36fc859a24189c959ebc11135dcb46013fcc615ad9904260f442014856`; tuning-strict `af1c280c8350a61e0d0b50e10bd15a3978bb9bc50b119fb4e308fc9616f40074`; tuning `2f51d74eb6cdcd824069d4eab4386d6bc4a62ff08558e9f3ab585d5f7e3df7eb`.

## Limits

This is a bounded actual startup-deployment review, not a full-goal or two-sweep certification. It does not grade live remember p95, fresh fifty-question recall, status-value semantics, successful backlog extraction, historical recovery, or credential remediation. Later snapshots and sweeps belong to a separate review. There were no live/shared writes, queue retries, provider calls, deployments, daemon restarts, configuration changes or shared-checkout edits in this review.

Logical preservation covers all active records in each complete backup and all their bytes; it does not assert physical Badger-file equality, preservation of deleted/expired logical keys, old internal LSM versions, or crash/disk-fault recovery. Candidate startup refusal was adversarially graded in the predeployment reports and is linked here by independent exact binary reproduction; those adversarial fixtures and the full Go test suite were not rerun in this bounded actual-deployment review.

Populated application Restore still performs DropAll before load/validation, the retained old executable is still destructive on a future numeric schema mismatch, and writer-floor metadata is unenforced. No schema bump, format migration, unrelated repair or rollback across formats is approved. Earlier predeployment backdating of two assertions and truncation of four repository-reference lists remain preexisting normal-write risks; the absence of those changes in this immediate interval does not fix or certify their semantics.
