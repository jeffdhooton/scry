# Independent integrated artifact and predeployment disproof

Verdict: **bounded conditional PASS for prospective code-only deployment of the exact 24eafab artifact below.** No integrated code, build, restore-preservation, or measured recall regression blocker was proved. Deployment still requires immediately fresh verified backups, retention of both old binaries, installed-byte/process verification, and an independent actual pre/post-state gate. This report is not an actual deployment PASS, source-repair approval, clean-store assertion, credential remediation approval, or completion of the memory-quality goal.

The historical overwrite and credential-redaction failure in the separate fresh-source BLOCK remain real. Preventing future occupied-key overwrites does not recover the lost record or fix redaction. Further identity/alias cleanup cannot inherit a safety PASS from this code-only review.

## Scope and independently pinned build

Private evidence root: `/tmp/scry-integrated-predeploy-independent.2D6GWQ`.

I ran `scry memory orient --cwd .`, read the active goal, both complete independent source reports, the fresh-source BLOCK, both proposals, the complete production diff from deployed d1f0a958, relevant architectural decision/audit entries, and previous deployment reviews. The root remained sole shared/live writer. I made no shared edits, live graph writes, deployments, retries, provider calls, room posts, durable notes, identity repairs, or credential operations. All new diagnostic code and restored stores are private. Read-only installed-binary/process checks were made on both hosts.

An independent `git archive 24eafab441f743da9e27ab1a72debe18c59975c5` supplied the build directory. The preexisting untracked assessment was excluded and untouched. Build and clean-source full tests ran before private diagnostics were added. Only the following production changes exist relative to d1f0a958: the occupied-key assertion guard, search's final document-key comparator, recall's final hit-key comparator, and explanatory comments. Tests and documentation account for the remaining diff. No ranking weights, query files, schema, relation vocabulary, provider configuration, hook configuration, retention settings, or migration were added.

Exact build, Go 1.26.2 darwin/arm64:

```sh
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w -X main.Version=24eafab' -o ../scry ./cmd/scry
../scry version
go version -m ../scry
```

Independent executable and proposed `/tmp/scry-fact-guard-deploy-sep06.uN9zYJ/scry` both have SHA-256 `4a4391090531a7112956dfae99e82633ed49d4e8ca2eb26f0916b5eae55e6b27`. The CLI reports `scry 24eafab`. Embedded build metadata confirms Go 1.26.2, trimpath, CGO_ENABLED=0, GOOS=darwin, GOARCH=arm64, GOARM64=v8.0. No private helper is compiled into that executable.

Verified production SHA-256 values:

| File | SHA-256 |
|---|---|
| store/store.go | `600a299cf8177ebacafaa80cba2c3778a77545b5bf4f487ba972087e50cfd8a6` |
| search/index.go | `8dcad681f90f1deefc34a36d7434faa93a9f8f2feb402ed2c566bc02c1807725` |
| recall/query.go | `3b269fe49f03526b5f24cd86a53f7c5a6482aa61757667692273635daf4da746` |
| fact-key-collision-independent-review-2026-09-06.md | `00c585210df9ea757995fd9614839d5d12e616a7b685cf893f9249e740f568ff` |
| recall-exact-tie-independent-review-2026-09-06.md | `849f12ab0a2e555365e3b2a2803b9ebb3ead2f73f7507381026f7eebe4d92247` |

The pinned old CLI `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/scry` independently hashes to `290a14c04ef0cfa9618db3f1a848bc6f3a343ec28d9c197ea49720eedb30b553`.

## Full direct restoration of both fresh backups

Each input was hashed completely, opened as a backup stream, and loaded with direct Badger `db.Load(f,16)` into a new private directory, before candidate Store.Open could perform schema handling. Every raw key and exact value was captured without any family filter. Candidate Open, AllFacts, Entities, complete offline index construction, an actual recall read, and Close then ran. A second complete capture was compared with the first using exact whole-map equality, including empty values versus absent keys. Both passed. Nothing was migrated, initialized over existing content, or dropped.

| Source | Bytes | Backup SHA-256 |
|---|---:|---|
| Shared `/tmp/scry-alias-rejection-deploy-sep05.pPmnn3/memory-20260906T003016Z.badger` | 74,226,362 | `7248b7da0acae98e970e551143e90949470b34ebbe98a002081ccad883ac3e28` |
| Laptop `/Users/jeff/.scry/backups/memory-20260906T003124Z.badger` | 19,445,008 | `182faa6891c6204de6523dfa6cc05611b0240cec2dbcbb5d755a6ff1a494d790` |

The shared source has 244,151 raw keys: 80,966 facts, 73,125 current / 7,841 historical; 30,783 entities; 9,382 episodes; 52,189 alias claims; 55,850 adjacency keys; 10,961 attestations; 3,143 cursors; 813 value-evidence records; 5 metadata keys; 4 alias rejections; 19 retired-slug and 19 retirement records; and **17 pending records: 12 parked plus 5 nonparked**. A shorthand queue count of 12 is not the total raw pending count. No ready/backoff classification at a particular wall-clock instant is inferred from the parked/nonparked split.

Its complete raw digest before/after is `4e33ccf0df9f7e35e20d444c4d56f2eda64ef66296594ded88c9f5f3cb3d9802`.

The laptop source has 83,378 raw keys: 21,004 facts, 18,495 current / 2,509 historical; 14,200 entities; 2,689 episodes; 23,159 alias claims; 21,004 adjacency keys; 1,321 cursors; one metadata key; no pending or rejection/retirement records. Its complete raw digest before/after is `8efead71b3128363e95c7f39678e3ec9332ae4a7645af03454ccf4780f02c2c7`. It matches the previous independent dormant-laptop logical-state digest. A backup does not prove the future role of a running daemon.

Digests iterate sorted Badger keys and hash uint64 big-endian key length, key bytes, uint64 big-endian value length, and value bytes. Equality is checked on every map entry, not inferred merely from counts or hashes. The shared recall read was 11,539 bytes; laptop 9,409 bytes. Full before/after inventories are saved in `shared-restore.json` and `laptop-restore.json`.

## Integrated disproof tests and actual historical collision

`CGO_ENABLED=0 go test ./...` passed across all packages in the clean independent archive. Its output was returned in the review session; the later focused run is retained as `focused-tests.txt`.

Independent additional Lornwick fixtures cover current and historical occupied keys, each attacked through literal normalization collision, changed statement, and changed raw relation. Each conflict occurs after another write inside AtomicWrite; every preexisting raw key/value remains equal, the earlier write rolls back, and zero observer events are emitted. Each fixture then proves allowed same-assertion metadata updates: changed confidence/provenance, invalidation or reopening, and an equal start instant represented with a +05:30 timezone. These tests do not endorse the existing reopening or provenance-replacement semantics as universally lossless.

Focused combined runs also pass the integrated builder tests for occupied conflicts, metadata updates, resolver-wide rollback, queue parking after one attempt with original input preserved, reviewed alias rejection, search cutoff ties, and recall duplicate ties. The stale merge fixture still injects altered raw text below PutFact and still verifies snapshot-drift refusal; the production admission contract was not weakened to accommodate it.

The actual overwritten assertion was independently replayed through unchanged normal resolve.Apply using the complete restored 23:51:34 source and the exact incoming assertion from 00:11:20. The older backup SHA is `7680cdcff3563c4cc2d33577506fc4c4aad4931d8bae1016f90e617bd7ee863e`; incoming backup SHA is `5b8e97894bea35a0ba2d1d63a1be01725845814051c8cb5c36273a0d4a5d1d6b`. Both were independently hashed and directly restored.

Original raw assertion SHA: `44805a4cf211693496ac56d89433209a00dfc6e90fc192b0ee4ae2c36ba0e026`. Incoming raw assertion SHA: `1c7aff7ef471dfb0cf756869db1ba8d398e8a11455fcf4e88a55bd7a4bd02f7e`. The replay uses its own private episode identity; it does not claim to replay the original complete extraction batch or transcript.

The exact combined candidate returns ErrFactConflict, zero Stats, zero observer events, and complete raw equality across all 243,703 original keys, digest `3800785f9b2c746ff8850e1653ae77335fe842c138b4cead6c7ee505889c2160`. This includes history, original current-fact invalidations that would otherwise occur, all pending payloads, four rejection records and nineteen paired retirement records. The earlier independent source review reproduced the destructive baseline overwrite; that baseline destructive replay was not rerun in this integrated review.

The 00:30:16 occupied replacement is already historical and hashes to `a53f8447bbd81e9582b7389c779a49caea34487023ddc26cc58cd44f26781bdb`. It is not the exact current 00:11:20 incoming payload. An initial fresh-current precondition therefore correctly refused that source; replay was then run against the explicitly pinned 00:11:20 incoming assertion. An earlier overlapping read of the benchmark replica hit Badger's directory lock before mutation; subsequent replay used separate restored directories. Neither refused diagnostic is counted as a successful replay or a product failure.

On another fresh 00:30:16 replica, the harness selected one actual ChildScribe rejection record and submitted its rejected spelling through **two normal Apply episodes**. Both episodes committed; the rejection remained effective, the alias was absent from the entity and never indexed back to it. Every preexisting key/value—including facts, claims, attestations, queues and all decision markers—remained byte-identical. Exactly two private episode keys were added, with no other additions, changes or removals. This represents one stored marker's interaction, not replay of every rejection/retirement source episode.

## Exact CLI before/after controls on fresh shared content

The pinned old CLI and independently rebuilt exact candidate each ran all five unchanged question files using `memory bench --dir <private-shared> --file <suite> --top 20`. Each command constructs its own index. Every suite total, hit count, complete ordered missed-question/rank list, and mean answer rank is identical. Original expectations were not changed.

| Suite | Old → combined | Misses each | Mean answer rank, both | Maximum bytes, both |
|---|---:|---:|---:|---:|
| heldout-2026-09-03 | 51/62 → 51/62 | 11 | 4.862745098039215 | 12,109 |
| heldout-b | 29/66 → 29/66 | 37 | 5.103448275862069 | 13,359 |
| probes | 7/7 → 7/7 | 0 | 1 | 9,983 |
| tuning-strict | 45/50 → 45/50 | 5 | 4.511111111111111 | 11,531 |
| tuning | 47/50 → 47/50 | 3 | 3.978723404255319 | 11,531 |

Every run has zero responses above 24,576 bytes. Full raw equality after all ten commands preserves the shared digest `4e33ccf0...`; every family remains unchanged. `bench-complete-safe.jsonl` retains every benchmark field, including each complete miss question/rank. Miss-preview strings are stored only as SHA-256 values because a graph preview could contain sensitive material. No full fact/value/derived sensitive slug or source projection was printed or saved by this harness.

These measured controls do not restore the original 53/62 and 34/66 floors, constitute a fresh fifty-question grader, prove individual successful-question ranks from aggregates, or claim equal complete recall payloads. The prior source review separately measured all 235 individual uncapped answer ranks, repeated Sheets stability, and independent adversarial tie fixtures. Its bounded evidence and documented limitations remain applicable to the matching production hashes; those larger repeat experiments were not all rerun here.

## Deployment readiness and mandatory remaining evidence

Read-only checks during this review independently confirm old hash `290a14c0...` still installed at laptop `/Users/jeff/go/bin/scry` and Mini `/Users/jclaw/.local/bin/scry`. Both jobs remain running with last exit zero: `gui/501/com.jhoot.scryd` PID 40814 and `gui/501/ai.jermes.scryd` PID 15220, both started September 5 at 17:58:20. Process arguments identify the expected installed executables running `start --foreground`. This review did not install or restart anything.

The proposed plan is safe to proceed only within these exact bounds:

1. Immediately before replacing executables, take fresh nonempty backups of both stores, verify local/remote hashes as appropriate, restore complete content, and reconcile source drift from the pinned review. Retain each currently installed d1f0a958 executable at a unique explicit sibling path and verify its old SHA before replacement. Retention was proposed but not independently performed or certified here.
2. Install only artifact SHA `4a439109...` at the two existing executable paths and restart only the two named launchd jobs. Verify installed bytes, new process identity/start time, expected Mini ingestion and dormant-laptop role. No configuration, hook, provider, queue retry, source-history or identity-manifest change follows from this gate.
3. Take and independently restore immediate complete postdeployment backups. Compare every raw family against the actual predeployment source, explicitly accounting for ordinary queue arrivals, completions, parking, invalidation and other ingestion. Preserve pending text/source/attempt/error metadata and exact historical assertions; rising fact counts are insufficient evidence. A changing queue requires explicit delta accounting, not a quiet-queue assumption.
4. Independently grade actual installed artifact/process state and the complete actual pre/post deltas. Run the five live suites, relevant ingestion/status/durability checks and subsequent real sweeps. The broader goal requires two subsequent real sweeps and two consecutive complete fresh grading rounds, including original floors and the new held-out bar. None of those future observations is pre-certified here.

The retained d1f0a958 binary understands current rejection/retirement markers, but rolling back to it reopens the demonstrated occupied-key overwrite hole. Before any rollback, preserve a fresh backup and later evidence; do not drain conflict-bearing inputs with the old writer. The much older 62cf marker-unaware retained artifact is not a safe standalone rollback. No backup restore may silently discard intervening ingestion.

Universal recall determinism remains unproved and disproved in prior independent fixtures: named-endpoint map visitation can select unequal scores before sorting; clipped/delimiter-based recall hit keys can collide; recency and embedding histories retain their existing behavior. Current canonical-triple coalescing can lose a distinct incoming statement before PutFact, and existing same-assertion metadata semantics remain separate work. The historical overwrite, redaction boundary, live graph defects, source review and all final-goal bars remain open. This report authorizes no cleanup manifest or credential action.

## Reproduction and evidence hashes

Private helper: `code/cmd/integrated-grade/main.go`. Modes are `restore SOURCE NEW_DIRECTORY SHA256`, `raw DIRECTORY`, `metadata DIRECTORY`, `bench DIRECTORY SOURCE_ROOT OLD_CLI NEW_CLI`, `replay OLD_DIRECTORY INCOMING_DIRECTORY`, and `interaction FRESH_PRIVATE_DIRECTORY`. Use separate new directories for independently running operations because Badger requires exclusive directory ownership. Restore mode performs direct load plus full candidate Open/index/read preservation. Benchmark mode invokes the exact separately built CLIs and compares complete scores/miss membership/ranks/caps. Replay and interaction write only their specified private replicas.

Focused command from private `code/`:

```sh
CGO_ENABLED=0 go test ./internal/memory/store ./internal/memory/resolve ./internal/memory/queue ./internal/memory/search ./internal/memory/recall -run 'IndependentIntegrated|RejectsDistinctOccupied|AllowsSameAssertion|HistoricalCanonical|ReviewedAliasRejection|ExactTie|Tied' -count=1 -v
```

| Private evidence | SHA-256 |
|---|---|
| cmd/integrated-grade/main.go | `0bc52450235e982036f88f5898af9df45c2c16a750791e3d99e3a2798b9cfbed` |
| store/integrated_independent_test.go | `eeb5a4fd4c9f44f0a00ea21ab30825628c4d2256f71b474370056e59e1eabb77` |
| shared-restore.json | `28d93600f9283ed77853f2103940d81b396cb283f9d62f7130bc07a191fda722` |
| laptop-restore.json | `c6f3ebdaab8e0fbb86d49b26af4abd614eeee734b3083393ec1fb38f34d82434` |
| bench-complete-safe.jsonl | `c7608add7a764edf0993d074874c61381f2047d583b0da332ef107610071278a` |
| actual-replay.json | `015ca40b5fe427b062549c792dfb503a1549ed34b688c60f1a6869e3b0f11a41` |
| alias-interaction.json | `eae062685de9b816a6b62736594b992995b228208cac70929517c2412f6d9ef8` |
| focused-tests.txt | `c464cbaa1592bdc4a7582b265e8226fc14639124742039edf40ae85b3595ca7b` |
