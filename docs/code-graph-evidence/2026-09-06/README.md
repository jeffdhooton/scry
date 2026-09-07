# Missing code symbols in the unified graph — local evidence, 2026-09-06

Base revision: `df598c49bca0784983d136f92c732e2cdacc4561` (verified before work).
The delivery is an uncommitted local patch; `delivery-hashes.json` pins its source
and tests. No installed binary, running daemon, live index, memory store, hook,
configuration, deployment or remote branch was changed. The two pre-existing
untracked assessment/goal documents were preserved.

## Result and acceptance checklist

- [x] Reproduced before production edits: both real indexes parse symbols and
  implementation evidence, yet the baseline graph has **0 nodes / 0 edges**.
  `Speaker` returns no match and `Greeter → Speaker` returns `found:false`.
  See [baseline-fixture.log.txt](baseline-fixture.log.txt) and [before-rpc.json](before-rpc.json).
- [x] Preserved producer output, source, versions, commands and SHA-256 hashes.
  [provenance.json](provenance.json), producer logs and decoded `*-index.json`
  are independent of the changed parser and builder. Each fixture's
  `expected.json` specifies exact IDs, locations and relationships from source
  and raw index evidence, and pins the original index and source bytes.
- [x] Missing kind metadata yields conservative code nodes. Explicit kinds win;
  late declarations replace early synthesized/inferred kinds. Referenced
  external metadata with an explicit kind survives protobuf field ordering.
  Invalid fallback descriptors, locals and parameters do not become code nodes.
- [x] `TestIndexedCodeGraphFixture` uses both real indexer outputs through
  `scip.Parse`, Badger code-store persistence, production `graph.build`, and
  production daemon query handlers over isolated Unix sockets. It closes both
  registries, creates a new handler instance, and queries from separate OS
  client processes. No graph records are directly seeded in this test.
- [x] Fresh parse and legacy `UnspecifiedKind` cases return identical node keys
  and edge endpoints. Path responses attribute both endpoints to stored node
  records and return the actual directed edge even for an undirected walk.
  Unrelated nodes remain disconnected, unchanged rebuilds preserve the exact
  node/edge lists, and a second repository returns no fixture symbols.
- [x] Ordinary definitions, references, callers, callees and implementations
  retain the fixture's expected counts and source locations after reopening.
  Go's missing callee evidence stays empty. Mixed authoritative/fallback and
  external endpoints, git co-change, author nodes and schema foreign keys have
  focused regression coverage. Existing graph registry build-window tests pass.
- [x] Baseline and final full Go test/vet gates passed; final tests are uncached.
  This includes the existing curated orientation, memory safety, identity,
  resolver and queue suites. No memory production code or tests changed.
- [x] Before/after counts, exact RPC outputs, timings, commands, revision,
  coverage limits and isolated rebuild/reversal procedure are retained below.

## Changes and limits

The shared SCIP descriptor helper uses the pinned Go binding's full parser,
including escapes and method disambiguators. Its final-descriptor categories are
`Method`, `Type`, `Term`, and `Namespace`. It does not parse prose documentation
or source text to guess a finer kind. Graph grouping maps these to `function`,
`type`, `term`, and `module`. An authoritative `Class`, `Struct`, `Interface`,
`Function`, etc. retains its existing grouping; generic `Type` is now `type`,
rather than incorrectly asserting `class`. Explicit unsupported kinds such as
`Variable` do not trigger descriptor fallback.

The parser fills absent definition kinds and keeps stronger later metadata.
Occurrence-only records still carry `External` unless explicit metadata is
available. The graph builder applies the same helper for old stores with empty,
`UnspecifiedKind` or `External` kinds, and uses the same classification for
nodes, call/reference endpoints and implementation endpoints. External nodes
without definition evidence have no invented file or line. Unreferenced external
metadata is not materialized. Invalid raw identifiers can still be retained by
the existing code-store APIs; fallback does not turn them into graph nodes.

The code store's existing callee records represent **references inside an
indexed enclosing scope**, not proof of an AST call expression. Callable pairs
retain the existing `calls` label (potential calls); pairs involving types,
terms or modules use `references`, so a type annotation or `implements` clause
is not labeled a function call. The ordinary callers/callees APIs are unchanged.
SCIP `is_implementation` remains the authority for implementation edges.

`graph.path` previously guessed edge labels from endpoint types. It now walks
actual stored edges and returns additive `nodes` and `relationships` fields.
The old display fields remain. Search still matches name substrings, and the
walk remains undirected with the existing nine-hop reach. Stored relationship
direction remains explicit in `relationships`; parallel edges use deterministic
stored-key order. No new path is inferred from shared files, spelling, or node
types. Path lookup now reads the edge list, an O(E) operation in addition to the
existing node searches; tiny-fixture timings do not establish large-repo latency.
No persistent schema or language-analysis system was added.

Indexer-specific evidence:

- **Go / scip-go 0.1.26:** all seven global declarations omit `kind`. Source
  `fixture.go:3` defines `Speaker`, `:7` defines `Greeter`, and `:9` supplies
  `Speak() string`, proving the implicit implementation. The index emits both
  type and method implementation records. The interface method uses a `Term`
  suffix (`Speaker#Speak.`); calling it a function from the descriptor alone
  would overclaim. There are no enclosing ranges: `Invoke` has no callee edges
  and no graph path to `Speaker`, despite the call visible in source.
- **TypeScript / scip-typescript 0.4.0 (TypeScript 5.9.3):** source `fixture.ts:1` defines `Speaker`,
  `:5` explicitly says `Greeter implements Speaker`, and `:6` implements `speak`.
  The index emits both implementation edges and scope ranges. The call in
  `invoke` at `:9` supports the fixture's one `calls` edge; its type annotation
  and the class's implements clause support two scope-reference edges.
  The parameter descriptor remains outside the unified graph. Its ordinary
  code-store callee occurrence is retained (three `invoke` callee occurrences).
- These fixtures prove neither Go call-graph coverage nor a TypeScript-to-table
  path. There is no index evidence for a code-to-schema relationship here; the
  regression asserts it remains absent. No new analysis was introduced to fill
  those gaps. The dated September 4 assessment §2 is diagnosis, not authorization
  for its broader examples, live rebuild proposal or memory backlog.

## Measurements

Counts use the actual persisted graph. Legacy-kind cases have the same after
counts. All before type/relation counts are zero.

| Fixture | Before nodes / edges | After nodes by type | After edges by relation |
| --- | --- | --- | --- |
| Go | 0 / 0 | function 3, module 1, term 1, type 2 | implements 2 |
| TypeScript | 0 / 0 | function 4, module 1, type 2 | calls 1, implements 2, references 2 |

Both after queries for `Speaker` return exactly one node. Both after paths are
`Greeter (type) → Speaker (type)`, distance 1, relation `implements`, with exact
SCIP IDs and definition locations. Both unrelated queries remain `found:false`.
See [after-rpc.json](after-rpc.json) for the exact requests/results (including
rebuilds, legacy cases, ordinary code APIs and the empty second repository).
The JSON is extracted without rewriting payloads from
[focused-final.log.txt](focused-final.log.txt). The original logs remain authoritative. Their `.log.txt` suffix keeps them
visible despite the workstation's global `*.log` ignore rule; contents are unchanged.

Single-run timings from those same logs, not latency benchmarks:

| Fixture / operation | Before | After |
| --- | --- | --- |
| Go initial graph.build, handler elapsed | 74 ms | 105 ms |
| TS initial graph.build, handler elapsed | 78 ms | 97 ms |
| Go Speaker query, full fresh client | 29.988 ms | 63.141 ms |
| TS Speaker query, full fresh client | 25.293 ms | 25.941 ms |
| Go implementation path, full fresh client | 8.202 ms | 13.293 ms |
| TS implementation path, full fresh client | 8.500 ms | 8.852 ms |

Client timings include OS process startup, socket RPC and any cold store open;
these are not pure search-computation timings. The recorded Go after rebuild
outlier is 1244 ms versus 105 ms initial build; unchanged node/edge equality
still passed. No production latency or scale claim follows from these fixtures.

## Commands and validation history

Run from the repository root. Existing dependencies and producers were used;
`GOPROXY=off` and `GOTOOLCHAIN=local` prevent test-driven downloads/upgrades.
The exact indexer argv, cwd roots and versions are in `provenance.json`.
The source/index bytes remain under `internal/sources/scip/testdata/{go,typescript}`.

```sh
CGO_ENABLED=0 GOPROXY=off GOTOOLCHAIN=local go test ./...
CGO_ENABLED=0 GOPROXY=off GOTOOLCHAIN=local go vet ./...
CGO_ENABLED=0 GOPROXY=off go test ./internal/daemon -run '^TestIndexedCodeGraphFixture$' -count=1 -v
CGO_ENABLED=0 GOPROXY=off GOTOOLCHAIN=local go test ./internal/graph ./internal/sources/scip ./internal/daemon -run 'TestClassifySymbol|TestCodeGraph|TestGraphGitSchema|TestKindFromSymbol|TestParseKind|TestParseExternalKinds|TestIndexedCodeGraphFixture|TestGraphRegistry' -count=1 -v
CGO_ENABLED=0 GOPROXY=off GOTOOLCHAIN=local go test -count=1 ./...
CGO_ENABLED=0 GOPROXY=off GOTOOLCHAIN=local go vet ./...
```

The first full test and vet commands were run before production edits
(`baseline-test.log.txt`, `baseline-vet.log.txt`, exit 0). The first end-to-end harness
attempt used `symbol` instead of the ordinary query API's `name` parameter;
its failure is retained as `harness-invalid-query-param.log.txt`. After correcting
the harness, `baseline-fixture.log.txt` fails only the missing graph node/edge/query/
path expectations; its ordinary code API assertions pass. No production fix
was present for either baseline run.

`checkpoint-1.log.txt` records the first recovered real graph; `checkpoint-parser.log.txt`
and `checkpoint-2.log.txt` cover metadata precedence, malformed input and expanded
regressions. `focused-final.log.txt` passes all focused acceptance cases.
`final-test.log.txt` and `final-vet.log.txt` record the full gates (including an
uncached 174.173-second memory-store suite). After making the harness compile
against both revisions, `final-current-test.log.txt` and `final-current-vet.log.txt`
validate the exact delivered files; unchanged packages can use the Go test cache. There were no
baseline/environment failures in the full gates and no unrelated repairs.
The old optional parser tests still skip when `/tmp/scry-graph.scip` is absent;
the new checked-in producer-fixture test does not skip or require an indexer.

To decode the unmodified producer bytes independently of Scry's implementation:

```sh
CGO_ENABLED=0 GOPROXY=off go run internal/sources/scip/testdata/inspect.go internal/sources/scip/testdata/go/index.scip
CGO_ENABLED=0 GOPROXY=off go run internal/sources/scip/testdata/inspect.go internal/sources/scip/testdata/typescript/index.scip
```

## Isolated rebuild and reversal

The focused test command above is the reproducible isolated rebuild procedure.
It verifies hashes, copies fixture source into a fresh root, rebases only the
index's `Metadata.ProjectRoot`, parses to a fresh code store, closes it, then
calls `graph.build` through the actual RPC handler. Graph and code layouts use
only that test's temporary home. A new handler instance reopens them and new
client processes invoke `graph.query` / `graph.path`. It rebuilds unchanged data
and checks exact node/edge equality. The legacy case reconstructs only the old
kind fields from the original index metadata before persisting; occurrences and
relationships are untouched. It never invokes `Daemon.Run`, watchers, configured
extractors or the installed daemon, and does not read the historical fixture
root for source attribution. Only its own temporary directories are cleaned up.

For an exact baseline reversal without modifying this worktree, export tracked
revision `df598c49bca0784983d136f92c732e2cdacc4561` into a **new** temporary checkout
using `git archive`, copy the checked-in fixtures and `graph_fixture_test.go`
there, and run the end-to-end command there with an existing Go module cache.
This reversal was executed successfully: see `reversal-command.json` and
`reversal-baseline.log.txt`. All four cases compiled and failed only their missing
graph expectations. The graph assertions fail with zero nodes/edges; the fixture hashes and
ordinary source-query counts remain valid. Return to this worktree and rerun
the same command to rebuild the fixed graph in another fresh home. Retain both
logs. No existing store has to be removed or restored. Never run this procedure
against the live Scry home; deployment and live rebuilding require separate work.

This delivery stops at local review. The September 6 memory reset and the
August 22 `BeginBuild`/`EndBuild` serialization contract remain in force.
