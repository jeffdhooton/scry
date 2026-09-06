# Independent schema-startup-refusal disproof

Verdict: bounded source PASS. I could not disprove that the pinned candidate refuses incompatible or missing-marker populated memory stores without changing their active raw logical key/value records, and initializes a schema marker only when no active logical record exists. This is a private source review, not deployment approval, a schema migration approval, or completion of the overall memory-quality goal.

Reviewed on 2026-09-06 UTC by a fresh-context independent reviewer. I read the complete active goal and `/tmp/scry-schema-refusal-sep06.IEy0uh/SCHEMA_REFUSAL_PROPOSAL.md`, inspected the exact changed source and existing/new tests, and used only newly owned private directories for writes. No candidate prototype, shared checkout, live store, provider, queue, credentials, deployment, configuration, or history was modified by this review.

## Exact scope and source

Base: `bccb905aa6de8284ff531648bd67d7122d533ee4`, independently extracted with `git archive` into `baseline/` here. The candidate is a private copy of `/tmp/scry-schema-refusal-sep06.IEy0uh`, excluding the builder's replica. Recursive comparison of `internal/` and `cmd/scry/`, plus comparison of go.mod/go.sum, confirms the only production difference from that base is `internal/memory/store/store.go`. Its diff changes schema-policy comments, adds ErrSchemaMismatch, and replaces ensureSchema; SchemaVersion stays 1. Private evidence helper and reviewer tests are outside shipped production changes.

Pinned SHA-256:

| Artifact | SHA-256 |
| --- | --- |
| Candidate store.go | `757e6af2d37ce4619f0856aea5850e478f73679bb225ec691e70b5fcda8b9be1` |
| Candidate store_test.go | `90a3ca56009afed022f49b15c8793445fa57601ce9681bed9e1e2e174ee8d5f5` |
| Candidate schema_refusal_test.go | `ca3b6a85b74616f47ce050fa1413eff3b500311ee73327c76a4df7dc3259f1c5` |
| Base store.go | `600a299cf8177ebacafaa80cba2c3778a77545b5bf4f487ba972087e50cfd8a6` |
| Reviewer independent_schema_disproof_test.go | `9a26e17d57e9df1872c443abbf8d97660f137fb09f7958d77cbfb8d0b4d6f97e` |
| Reviewer independent_old_wipe_test.go | `acec1e49ce23cb53b8956491293080565a0fd0467fe090c7ebffde128712bc82` |
| Copied schema-evidence/main.go | `0bc52450235e982036f88f5898af9df45c2c16a750791e3d99e3a2798b9cfbed` |

The builder's store.go hash was checked again after execution and remained identical.

## Attempts to disprove the boundary

Independent tests used real on-disk Badger databases. Each comparison enumerates every active raw record into a key-to-bytes map and compares the complete maps; it does not select facts or rely only on counts.

- 23 marker states, each alone/in a populated store, each opened three times: missing, empty bytes, zero, null, negative, future 2, old-wipe trigger 999, maximum integer, overflowing integer, float, exponent, leading-zero integer, plus-prefixed integer, quoted integer, object, array, boolean, concatenated JSON, truncated JSON, invalid binary, a 1 MiB malformed marker, supported 1, and supported 1 with JSON whitespace. Every refusal preserved the complete map. Supported markers retained their original bytes. Only missing marker plus zero records initialized exactly `meta:schema_version=1`.
- Populated fixtures included opaque binary and 512 KiB values, binary keys, nil/empty values, and fake records under ep/en/fa/adj/al/cur/pq/ve/ar/rt/rs/meta/opaque. Separate one-record fixtures used only an empty opaque value, NUL key, 0xff key, or additive writer-floor key; all were recognized as nonempty and refused without initialization.
- Twelve concurrent Open calls tested supported, incompatible 999, missing-marker populated, and brand-new databases. No incompatible handle was admitted and complete logical data remained equal. Concurrent creation converged to exactly one schema-1 marker. Lock contention may correctly refuse a competing Open; this does not claim every simultaneous caller succeeds.
- Closed-database ensureSchema returned ErrDBClosed; initialization on a read-only empty database returned ErrReadOnlyTxn; opening a regular file returned an error and preserved its bytes. Source inspection additionally confirms txn.Get errors, item.Value errors, txn.Set errors, and Update/Commit errors are returned. Physical disk read/write fault injection was not performed; this report does not claim coverage of every storage-device failure.
- An independent test against the archived old source confirmed that Open on marker 999 succeeds, removes all four planted non-marker records, and rewrites the marker to 1. Candidate matrix tests demonstrate refusal and full preservation for the same numeric trigger.

Source reasoning: marker read, complete emptiness check, and optional initialization run in one Badger update transaction. On any present incompatible marker or absent marker with an active record, ensureSchema returns before Set. It contains no DropAll/Delete operation. For supported 1, it performs no Set. The inspected installed Badger Update implementation relays callback errors and discards the transaction; Commit with no pending writes returns without a write. Open closes the database on schema failure. Normal public Open retains Badger's locking, so this review does not posit bypassed locking or concurrent raw writers during initial creation.

## Independently repeated backup proof

The complete backup `/tmp/scry-fact-guard-deploy-sep06.uN9zYJ/memory-20260906T010309Z.badger` was verified as 74,430,346 bytes with SHA-256 `70247d3efb58f6fc2d69768ff24d4b59d304c67c06ce6d2bd4fa00cabef3b208` and loaded directly with Badger Load into the previously nonexistent private `replica-first-sweep/` directory here. The copied helper was read before use; only its restore operation was invoked.

Candidate Open, AllFacts, Entities, offline index construction, and a local recall completed. All 244,694 active raw records were exactly equal before/after, with length-framed SHA-256 `07aacc2b661cfe8d7973b5d2bed50d4ab77e1f296f5512f25139bf660779a4e6`. The reader returned 81,106 facts, 30,863 entities, and an 11,449-byte recall payload. The raw comparison covers all observed families, including 11,010 att records, 73,259 current facts and 7,847 historical facts. Only counts/hashes were emitted for real content. The five benchmark suites were not rerun by this reviewer; the bounded production change affects startup only, and no fresh ranking or whole-goal claim is made.

## Executed verification and evidence

- Candidate: `CGO_ENABLED=0 go test ./internal/memory/store -run 'TestIndependentSchema|TestSchema|TestOpenCloseAndSchemaRefusal|TestBackupRoundtrip' -count=1 -v` — PASS. The final backup regex alternative matches no test name; backup/restore coverage comes from the complete suite below and the separate direct-Load proof, not that alternative.
- Archived baseline: `CGO_ENABLED=0 go test ./internal/memory/store -run TestIndependentOldBaseline999Wipes -count=1 -v` — PASS, reproducing deletion.
- Candidate: `CGO_ENABLED=0 go test ./...` — PASS, including the reviewer fixtures and existing backup/restore tests. All test-bearing packages reported success.

Evidence hashes in this report directory:

| Evidence | SHA-256 |
| --- | --- |
| FOCUSED.log | `58ad2dfdbef97f78e42618bfb6b42b92c4d8b874d3c862572f2fa87c327b3bae` |
| BASELINE.log | `fbb20e83876a28e50136bb17361f94a428366a9e14d404c9dcc91f731dd7049b` |
| FULL.log | `8301194a6b7364b91ed7adc4b6bf92b7d199fafbb3731f19c35dd2219494d04d` |
| RESTORE.json | `11ca65114ca59d1d6ba49977ffc5a9cd98ef034fb6e0ce8fcd04c555d794455c` |

## Remaining limits and finding

One nonblocking documentation defect remains at store.go line 5: the package comment still says “The only automatic wipe is on schema mismatch.” That statement is stale after this change. A comment-only correction should be hash-pinned before promotion.

“Raw equality” here means active logical key/value equality. Badger opening/closing can alter physical database files and internal metadata; historical LSM versions and expired/deleted keys are not the logical rows enumerated by a normal iterator. No physical file-byte preservation claim is made.

Old installed/retained binaries remain dangerous on a different nonzero numeric schema. Additive writer-floor metadata is not enforced by these sources. Restore still calls DropAll before validating/loading input and is not repaired or approved for populated destinations by this review. A future format transition still requires its own compatibility/refusal bridge, reviewed migration and rollback safety plan. This PASS neither closes those design blockers nor approves schema bump, fact-format redesign, live deployment, live repair, or the overall goal.
