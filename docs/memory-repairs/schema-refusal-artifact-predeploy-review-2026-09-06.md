# Independent exact-artifact schema refusal predeployment review

2026-09-06 UTC. Review workspace: `/tmp/scry-schema-independent.wksyLD`.

Verdict: bounded predeployment PASS for the exact schema-1 startup-refusal artifact. I could not disprove its safety within this scope: independent rebuild, full no-CGO suite, both complete backup restores, all-record preservation, exact-binary refusal fixtures, five-suite comparisons on both snapshots, and all 235 shared per-question rank/payload comparisons passed. Whole-goal completion is rejected: the original held-out floors remain unmet. Deployment itself, fresh live drift checks, and postdeployment verification remain the lead agent's subsequent responsibilities.

## Scope and independent provenance

I read the complete active goal at `/Users/jeff/.codex/attachments/4eb02e47-fa9c-4a97-a378-3e707c35e8e2/goal-objective.md`, ran memory orientation, read both committed schema-refusal independent reports, inspected the production source diff and refusal/restore/benchmark implementations, and conducted new measurements. Existing reports and lead-agent results were not used as proof of this artifact's behavior.

Source was independently exported using `git archive a078240 | tar -x -C /tmp/scry-schema-independent.wksyLD/source`. The full revision is `a07824007973301066b36d1b97c398d9ea081a95`. The baseline source was separately exported from `24eafab`. Between those revisions, the only production change is `internal/memory/store/store.go`: schema-policy comments, ErrSchemaMismatch, and transactional ensureSchema replacement. The schema stays 1. The resolver, recall, CLI, go.mod and go.sum are unchanged. The abandoned restatement design is not present in this production diff.

Candidate store.go SHA-256: `4b18a0037534aa0111b3d8fede8883dbf4bfa3ffe955b3ff15ebd0d07fa38f88`.

Independent build command, from the exported source:

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=a078240' -o /tmp/scry-schema-independent.wksyLD/scry ./cmd/scry
CGO_ENABLED=0 go test ./... > /tmp/scry-schema-independent.wksyLD/full-tests.log 2>&1
```

Go reports `go1.26.2 darwin/arm64`. Build metadata confirms no CGO and trimpath. The full suite exited 0, with 27 test-bearing packages reporting success. This was run before adding the private reviewer helper to the exported source.

The independently built binary and the candidate `/tmp/scry-schema-refusal-deploy-sep06.uUcaFo/scry` both hash to `7783216755045eb365e0e0cd281ead8e2842da1b8fb697dfb42158c232cbb7e7`. Baseline `/tmp/scry-fact-guard-deploy-sep06.uN9zYJ/scry` independently hashes to `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`.

## Private complete restores and logical preservation

Both destinations were nonexistent before the reviewer helper verified the full backup hash, loaded directly with Badger Load, and enumerated all active logical key/value records. The helper then used candidate store.Open, AllFacts, Entities, offline index construction, and local recall, closed the store, and compared the entire key-to-bytes map using reflect.DeepEqual. This bypasses the application's destructive populated Restore. Every record family was included, including opaque/additive records and invalidated facts. The length-framed digest hashes key length, key, value length, value in iterator order.

| Snapshot | Backup bytes | Backup SHA-256 | Logical rows | Facts (current / historical) | Entities | Read payload bytes |
|---|---:|---|---:|---|---:|---:|
| Shared | 74,611,705 | `4045ad6ebf4b632907c361ddf530bbe5355bb5236eb01a7f474b549f43e886b7` | 244,942 | 81,174 (73,326 / 7,848) | 30,900 | 11,474 |
| Laptop | 19,445,000 | `5a5e7c22851b55b0e150d2ffaa9f22846c2f28644b45020e8ce830fe36aa4cd1` | 83,378 | 21,004 (18,495 / 2,509) | 14,200 | 9,409 |

All raw records were exactly equal before/after opening/index/read and before/after all ten artifact benchmark invocations for each snapshot. Shared digest: `29c3983bd20aefc168129d05cb6de777219b8fda6d3fdb7c213c1e210947006d`. Laptop digest: `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`.

The shared source is `/tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T013401Z.badger`; laptop source is `/Users/jeff/.scry/backups/memory-20260906T013402Z.badger`. Source files were read only. Destinations are this review directory's `shared` and `laptop`.

## Exact deployed/candidate artifact benchmark comparison

Each artifact ran `memory bench --dir <private-store> --file <exported-source>/docs/memory-bench/<suite>.json --top 20` against the same unchanged private store. All five suites were run on both snapshots. Full JSON results, complete miss membership/ranks, mean answer rank, payload statistics and latency are retained in `benchmarks.jsonl` and `laptop-benchmarks.jsonl`. Only miss preview strings were replaced with their SHA-256 hashes in memory before saving, because underlying facts may contain historical credentials. No fact text, raw value-derived key, or source payload was emitted. Questions are the existing benchmark questions, not generated from source payloads.

| Suite | Shared old = candidate | Shared mean answer rank | Shared max bytes | Laptop old = candidate | Laptop max bytes |
|---|---:|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 | 4.8431372549019605 | 12,110 | 22/62 | 10,578 |
| heldout-b | 29/66 | 5.068965517241379 | 13,369 | 13/66 | 11,034 |
| probes | 7/7 | 1 | 9,962 | 5/7 | 9,771 |
| tuning-strict | 45/50 | 4.511111111111111 | 11,539 | 28/50 | 10,470 |
| tuning | 47/50 | 3.978723404255319 | 11,539 | 30/50 | 10,470 |

The helper asserts hits, totals, top=20, mean rank, complete miss membership/ranks, zero cap exceedances, and whole-store raw equality. An additional jq comparison proves every result field excluding latency is equal between binaries on each snapshot, including all hashed miss previews and payload measurements. Shared maxima remain below 24,576 bytes, but 51/62 and 29/66 still miss the original 53/62 and 34/66 floors. This artifact does not repair those existing failures.

The private rank helper was separately compiled against the exported baseline and candidate sources. Both ran all 235 questions, each suite with a fresh offline index, against the unchanged shared replica. `cmp baseline-ranks.jsonl candidate-ranks.jsonl` exited 0: every question identifier hash, answer rank (including misses), individual payload size, and the final raw-preservation record were byte-identical. These are helper-built source rank measurements complementing the exact-artifact CLI benchmarks; they are not a claim that the shipped bench CLI emits every successful individual rank.

## Exact-binary adversarial startup checks

The reviewer created private Badger databases with a 512 KiB opaque binary value, a binary key with empty value, and synthetic additive metadata. The exact candidate artifact was invoked through offline memory bench on eight incompatible marker states: numeric 999, absent marker, zero, null, malformed JSON, quoted 1, float 1.0, and future version 2. Every command failed with the expected schema-refusal message; every complete raw key/value map remained equal. Error outputs were recorded only as hashes.

The exact deployed old artifact was invoked against its own identical numeric-999 fixture. It succeeded and erased all three opaque records, leaving only the rewritten schema-1 marker. This independently reproduces the prior wipe using the retained artifact rather than relying only on a source-level test.

Source reasoning supports the observed boundary: marker read, complete iterator emptiness test, and optional initialization share one transaction. Populated missing-marker and incompatible-marker paths return before Set; supported schema does not write; no Delete or DropAll remains in ensureSchema. Open closes its database on schema failure. Existing source tests cover repeated opens and supported empty initialization in the passing full suite.

## Commands, helper and evidence

The inspected lead helper's restore/raw/bench routines were reused in the reviewer's private helper, with unused replay/interaction/metadata code removed. The reviewer added exact-binary opaque fixtures and per-question rank output. Helper path: `source/cmd/independent-evidence/main.go`; SHA-256 `64f0951d2a9b1fdb3fbbef09521c8dfe2aefb4d16077912de73a84d1b5690046`. The baseline helper has identical executable content and one additional trailing blank line; SHA-256 `72dab6bd8ccff08b76ff1619cb654c1d7ce8358df8b3b7759a8c90ee538104c7`.

```sh
# From each exported source, build the respective private evidence executable.
CGO_ENABLED=0 go build -o /tmp/scry-schema-independent.wksyLD/evidence ./cmd/independent-evidence
CGO_ENABLED=0 go build -o /tmp/scry-schema-independent.wksyLD/baseline-evidence ./cmd/independent-evidence
/tmp/scry-schema-independent.wksyLD/evidence restore /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/memory-20260906T013401Z.badger /tmp/scry-schema-independent.wksyLD/shared 4045ad6ebf4b632907c361ddf530bbe5355bb5236eb01a7f474b549f43e886b7
/tmp/scry-schema-independent.wksyLD/evidence restore /Users/jeff/.scry/backups/memory-20260906T013402Z.badger /tmp/scry-schema-independent.wksyLD/laptop 5a5e7c22851b55b0e150d2ffaa9f22846c2f28644b45020e8ce830fe36aa4cd1
/tmp/scry-schema-independent.wksyLD/evidence bench /tmp/scry-schema-independent.wksyLD/shared /tmp/scry-schema-independent.wksyLD/source /tmp/scry-fact-guard-deploy-sep06.uN9zYJ/scry /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/scry
/tmp/scry-schema-independent.wksyLD/evidence bench /tmp/scry-schema-independent.wksyLD/laptop /tmp/scry-schema-independent.wksyLD/source /tmp/scry-fact-guard-deploy-sep06.uN9zYJ/scry /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/scry
/tmp/scry-schema-independent.wksyLD/evidence fixtures /tmp/scry-schema-independent.wksyLD /tmp/scry-fact-guard-deploy-sep06.uN9zYJ/scry /tmp/scry-schema-refusal-deploy-sep06.uUcaFo/scry /tmp/scry-schema-independent.wksyLD/source/docs/memory-bench/probes.json
/tmp/scry-schema-independent.wksyLD/baseline-evidence ranks /tmp/scry-schema-independent.wksyLD/shared /tmp/scry-schema-independent.wksyLD/source
/tmp/scry-schema-independent.wksyLD/evidence ranks /tmp/scry-schema-independent.wksyLD/shared /tmp/scry-schema-independent.wksyLD/source
```

Outputs were redirected to the corresponding files below. Build, test, restore, fixture and completed benchmark commands exited 0.

| Evidence | SHA-256 |
|---|---|
| full-tests.log | `03711d8e14eeefb36264ed8dc5987827db187b8233b34f79bbd89258e4a903ea` |
| shared-restore.json | `ee78338c8cb44b467ef177f3631673773fd5682cb2a97901d04c7a6c3a6cb193` |
| laptop-restore.json | `d11be9f528d4ddf6e7192dceaddb7a4673ccc15ff54a7fadc0a7bbaf5f84fb82` |
| benchmarks.jsonl | `f47a85d618f29006d882074d4dc9396fa24d7a98271ba292367910105e1624cf` |
| laptop-benchmarks.jsonl | `97ab91dbc54efae9c2294b09acb5a0aee68581c80d4ca9a7d96d5c8a8e4edf33` |
| binary-fixtures.jsonl | `1490af35e789e3fbd7e32b0e584cbee7a73f62728c39ed39564b029ff5f89ba4` |
| baseline-ranks.jsonl | `442343dcbac85aa4604ef4d9de30a0c4008601f5b0b28ce4e0a10a253e63cb7d` |
| candidate-ranks.jsonl | `442343dcbac85aa4604ef4d9de30a0c4008601f5b0b28ce4e0a10a253e63cb7d` |

At the lead's request, I independently hashed the retained local predeployment binary `/Users/jeff/go/bin/scry.pre-a078240-20260906T0146Z` with `shasum -a 256`; it matches the baseline hash `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`. I also ran `ssh jclaw@mini 'shasum -a 256 /Users/jclaw/.local/bin/scry.pre-a078240-20260906T0146Z'`; the retained Mini binary independently matches the same hash. Only these exact files were read remotely; no remote changes were made.

## Limitations and required deployment boundary

This review performs no live mutation, provider call, queue retry, daemon restart, configuration change, deployment, builder/shared-checkout edit, or credential output. All reviewer-created sources, reports and fixture/replica stores are in the private review directory. It does not certify the current live state or subsequent sweeps, ingestion, remember p95, credential remediation, identity cleanup, or the entire active goal.

Preservation means active logical records and their exact bytes. Badger may change physical files/internal metadata when opened; deleted/expired keys and historical LSM versions are not included in an ordinary logical iterator. No disk-fault or corrupt-backup recovery proof is claimed.

Populated Restore is still unsafe: it calls DropAll before validating/loading input and checks the schema afterwards. This review neither uses nor approves populated Restore. The old retained binary remains demonstrably destructive on numeric schema mismatch, and additive writer-floor metadata is not enforced. A schema bump, storage-format migration, or rollback across formats requires a separate reviewed compatibility plan. The permitted rollout remains exactly this schema-1 refusal binary, after the lead verifies fresh nonempty backups, retained prior binaries, and live drift, followed by actual deployment verification. Do not interpret the bounded verdict as approval for the abandoned assertion redesign or any store repair.
