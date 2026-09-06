# Independent actual a06cd7b deployment preservation grade

2026-09-06 UTC. Verdict: **bounded actual deployment preservation PASS**. Independent disproof found no loss or mutation of active logical store records across the 02:54:47–48 UTC rollout, and no candidate/baseline retrieval difference on either frozen postdeployment store. This is an actual installed-artifact, resident-process-path, startup, complete-backup and frozen-retrieval check. **The complete memory-quality goal still FAILS.**

Ran `scry memory orient --cwd .` before other work. Read the full active objective at `/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md`, supplied AGENTS instructions, and both complete historical-address artifact and immediate-predeployment reports. Their independently checked SHA-256 values are `2994ab25878fdde7ebc6622f4b1a419208fb933e882045a30edf40f13e556a84` and `608921643404d034c96d0e40cec2b203e57a981f8434204d1f6c19ee9cd7b6b9`. Those reports establish earlier context; their PASS verdicts are not substituted for the actual deployment checks below.

## Installed and retained artifacts

Exported `git archive a06cd7b` and `git archive a078240` into this private directory. Rebuilt each from its exact export using Go1.26.2, `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=REVISION'`. Candidate `scry` independently reproduces SHA-256 `7938d05258bb3b08461441b7448a8da4a95374664f42a9b240f22c3327de55ef`; baseline `scry-baseline` independently reproduces `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`. `go version -m` confirms Go1.26.2, CGO disabled, darwin/arm64 and trimpath for the candidate.

Both `/Users/jeff/go/bin/scry` and `jclaw@mini:/Users/jclaw/.local/bin/scry` independently hash to the candidate SHA. Both corresponding `.pre-a06cd7b-20260906T0250Z` retained files hash to the independently rebuilt baseline SHA. Both `.pre-a078240-20260906T0146Z` artifacts remain present with SHA `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`. The host inventories additionally contain 71 laptop and 77 Mini `scry.pre-` files. This reviewer captured their complete names/sizes/hashes but did not have a complete earlier inventory proving that every historical backup name remains; the immediate previous and specifically documented older artifacts are independently verified.

`ps`, filtered `launchctl print`, and `lsof` agree:

| Host | Process | Start, America/New_York | Executable and arguments | launchd |
|---|---:|---|---|---|
| Laptop | 10158 | 2026-09-05 22:54:47 | `/Users/jeff/go/bin/scry start --foreground` | `gui/501/com.jhoot.scryd`, running |
| Mini | 12146 | 2026-09-05 22:54:48 | `/Users/jclaw/.local/bin/scry start --foreground` | `gui/501/ai.jermes.scryd`, running |

Each service uses `/bin/zsh` as its launch program; the live process has exec'd the expected Scry path, corroborated by `lsof`. This proves process start/path plus on-disk artifact identity, not a cryptographic hash of resident process memory. Artifact file modification time is not used as deployment time.

Read actual startup logs in process memory and emitted only fixed event classifications, timestamps, counts and hashes. Mini log `/Users/jclaw/.scry/logs/scryd-launchd.log` records a successful index build at 22:54:55: 122,013 documents, 128,802 words, 81,514 facts, 6.719 seconds, followed by `worker_started`. Laptop `/tmp/scryd-launchd.log` records extraction off and its worker dormant at 22:54:48, matching forwarding to shared extraction. The selected rollout window contains none of the checked index-build/store-open/schema/warmup error patterns. This is bounded log-pattern evidence, not an assertion that arbitrary errors cannot exist.

## Complete fresh pre/post restores

Independently hashed all four complete files before direct Badger Load into nonexistent private destinations `pre-shared`, `post-shared`, `pre-laptop`, `post-laptop`. No populated application Restore was invoked. The Mini's original backup paths were independently hashed over SSH and match the downloaded shared files. Host file timestamps put both pre backups at 22:53:32 and both post backups at 22:55:09 America/New_York, bracketing the observed process starts.

| Snapshot | Complete input path | Bytes | SHA-256 |
|---|---|---:|---|
| Shared pre | `/tmp/scry-historical-deploy-sep06.INlBVW/memory-20260906T025332Z.badger` | 75,362,914 | `5e53e905c2b2eecb5cf7a985c22c88adbc449c28782146765c535723116ed81d` |
| Shared post | `/tmp/scry-historical-deploy-sep06.INlBVW/memory-20260906T025509Z.badger` | 75,362,922 | `2f60f448e8d3844e6fd2692aa69891a693e6cd41145e2c47fac336af8ed7743b` |
| Laptop pre | `/Users/jeff/.scry/backups/memory-20260906T025332Z.badger` | 19,445,016 | `0f3aec53bf5ce13e29ad757f3fa152ed4914ccd74990bffc8b8cb8809cf31aff` |
| Laptop post | `/Users/jeff/.scry/backups/memory-20260906T025509Z.badger` | 19,445,024 | `9a65e98f7a841cb90c17a2f8ba5026be32edf488846dc291e151d98da78655fa` |

The Mini original paths are `/Users/jclaw/.scry/backups/` plus the corresponding shared backup filenames. Full raw maps are captured before/after candidate Store.Open, AllFacts, Entities, offline search-index construction and bounded recall. Every active logical key/value pair is byte-identical around these operations on every restore.

Across the actual rollout, all shared 246,219 active rows remain byte-identical, with length-framed sorted digest `e88e1d8a5fb5c922ecd028393e2a09ba2bcdb20eecc383f6c6d3ef750c5bf720`. This also equals the earlier independently reported 02:47:04 logical digest. Exact shared families: 56,227 adjacency, 52,570 alias claims, 4 rejection markers, 11,146 attestations, 3,163 cursors, 31,074 entities, 9,425 episodes, 81,514 facts, 5 metadata, 23 pending, 19 retirement spellings, 19 retirement targets and 1,030 value-evidence rows. Added, changed and removed counts are zero in every family.

All 81,514 complete fact payloads survive, including 73,651 current and 7,863 historical records: full assertion text, endpoints, value, raw relation, start, invalidation, confidence, provenance and any untyped bytes are preserved by raw equality. All 23 originally parked complete inputs, all 42 repair markers and all 52,570 alias claims remain exactly equal.

All laptop 83,378 rows likewise remain byte-identical, digest `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`: 21,004 adjacency, 23,159 claims, 1,321 cursors, 14,200 entities, 2,689 episodes, 21,004 facts and one schema metadata row. Its 18,495 current and 2,509 historical facts are unchanged. There are no laptop pending or repair-marker rows in these snapshots.

## Actual CLI and full-response parity

Twenty actual CLI commands use the installed candidate and retained previous laptop binaries: both binaries × both frozen post stores × five unchanged benchmark files, top 20. The old/current benchmark files were separately compared and are identical. Every pair matches total, hits, mean answer rank, maximum payload, complete miss membership/ranks and hashed preview content. Complete raw maps remain equal after all CLI commands. Benchmark envelope hashes can differ because mean latency is measured; they are not used to assert response equality.

Independently compiled old/current recaller helpers also run all 235 individual questions against each frozen post store. Both 236-line files per store (235 records plus final raw-preservation inventory) are byte-identical, including question SHA, rank, payload byte count and full serialized recall-response SHA. Shared evidence SHA is `ab42ced88db7a6a08f4a6ac52c1adccdae2077b5d9b91904fba8b3571122a2a1`; laptop is `94e621c81823b18f4814ca78728856b30cae9c874ea4b282c37926a83c5badf7`. All responses remain below 24,576 bytes; all final raw maps remain equal.

| Suite | Shared hits, both | Shared maximum bytes | Laptop hits, both | Laptop maximum bytes |
|---|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 | 12,111 | 22/62 | 10,578 |
| heldout-b | 29/66 | 13,359 | 13/66 | 11,034 |
| probes | 7/7 | 9,974 | 5/7 | 9,771 |
| tuning-strict | 45/50 | 11,542 | 28/50 | 10,470 |
| tuning | 47/50 | 11,542 | 30/50 | 10,470 |

Shared frozen heldout-b mean answer rank is 5.103448275862069 under both old and current binaries. The lead separately reports live resident pre/post mean rank 5.068965517241379→5.103448275862069. I did not independently re-run the lead's live before measurement. The frozen old/current equality, complete stored-byte equality and confirmed actual index rebuild exclude a candidate-only ranking difference in this captured comparison and support index-refresh attribution. They do not prove equality of all live response bytes or reproduce the departed incumbent's exact resident index state.

## Remaining failures and limits

Every independently computed structural defect set is equal across the rollout on both stores. Shared defects remain 2,854 zero-fact entities, 2,975 no-current entities, 2,441 dangling endpoint occurrences, 504 wrong-owner listed-spelling occurrences, 29 unlisted owner claims, 462 multiply listed spellings, 27 cross-type listed spellings, and 1,099 total/93 current self-loops. Shared missing listed claims, dangling claim owners, missing provenance episodes and missing/extra reverse adjacency remain zero. Current relations remain exactly the documented 39, with zero noncanonical current facts. Complete laptop defect sets are also preserved; laptop is not certified structurally clean.

Original shared recall floors 53/62 and 34/66 are still unmet. This grade does not certify historical recovery, temporal semantic correctness, fresh fifty-question recall, remember p95, orient coverage, a hygiene no-op, or two subsequent real sweeps. The existing parked fact-conflict input remains untouched; no provider call or retry was made. General current coalescing/backdating and other explicitly disclosed predeployment contract limits remain outside this actual rollout preservation verdict.

All helper source files were read fully before reuse and copied through apply_patch only into private exports. `verify.sh` serializes same-store reads; independent shared/laptop replicas run concurrently. `summarize.cjs` asserts all completed comparisons. `host-check.cjs` and `log-check.cjs` capture sanitized evidence without launchd environments or raw log/error payloads. No shared source edit, live-store write, deploy, restart, sweep, provider call, queue retry, configuration change or sub-agent delegation occurred. One preliminary build used the shared cwd and was superseded by the independently matching exact-export build; a private-directory Git inspection failed harmlessly and was corrected against the repository. No check was weakened to obtain PASS.

Evidence concerns active logical Badger records, not physical SST/vlog bytes, deleted/expired records, hidden old versions or crash recovery. Private restored databases contain real data and must not be published. `ARTIFACTS.sha256` pins this report, sanitized evidence, helper sources, reproduction scripts and independently built binaries.
