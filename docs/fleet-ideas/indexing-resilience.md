# Indexing that degrades instead of collapsing

scry's code index is the foundation every other domain sits on, and right now it fails in
ways that leave the user with nothing and no way to find out why. A live `scry status`
across ~45 registered repos shows more than a dozen stuck `partial`, thousands of
TypeScript files invisible to `refs`/`defs`/`callers`, and a `graph` that builds zero nodes
for at least one large repo. This work makes indexing resilient and diagnosable. It is
confined to indexing; it does not redesign the query or graph domains.

Four failures were reproduced by hand against the real installation. Fix them.

**One missing indexer aborts the whole build.** `scry init` on a repo with php +
typescript + python returned `rpc error -32603: index build: python: scip-python not found
on PATH` and wrote no manifest at all — the previous stale one survived untouched. The php
and typescript indexers would have succeeded. A primary indexer that cannot run must
degrade that language to a recorded failure and let every other language index; the build
returns `partial`, never nothing. The only thing that should abort a build is a failure that
makes the whole result untrustworthy.

**A failing indexer reports nothing useful.** After installing the binary, the same repo
reported `scip-typescript exited non-zero: exit status 1`. That is the entire diagnostic.
Capture the indexer's stderr (clipped, last N KB — the tail carries the error), record it on
the `IndexerResult`, and surface it through `scry status` and `scry doctor`. A user staring
at "exit status 1" cannot act; a user seeing the compiler's actual complaint can.

**The indexers cannot be provisioned.** `internal/install/install.go` documents that
scip-typescript has no GitHub release assets and so is skipped by `scry install`, leaving a
remedy string the user must run by hand. Both scip-typescript and scip-python install
cleanly from npm (verified: `npm i -g @sourcegraph/scip-typescript` → 0.4.0). Teach
`scry install` and `scry doctor --fix` to provision npm-published indexers when npm is
present, and to say precisely what to run when it is not. A daemon started before the
install will not see the new binary on its PATH — detect that specific case and say so,
rather than reporting the tool as missing forever.

**Stale and empty indexes are invisible.** Many repos were last indexed months ago, and at
least one large TypeScript repo builds a graph with `node_count: 0, edge_count: 0` while
reporting `ready`. A repo whose index is older than its HEAD commit, and an indexer that
produces zero symbols from a non-empty file set, are both failures wearing a success label.
Detect both and report them — a `stale` / `empty` signal in `scry status` and `scry doctor`,
not a silent green.

## Ground rules

- Go, no CGO. The existing store, layout, and manifest shapes stay; this is about how
  failures are handled and reported, not a schema redesign.
- `internal/index/builder.go` already injects the indexer runner (`run func(language string)
  error`) precisely so this logic is testable without real binaries on PATH. Use that seam —
  no test may require scip-typescript, scip-python, or npm to be installed.
- Test command: `go test ./...` from the repo root. `go build ./...` must stay clean.
- Every behavior change needs a direct test: the partial-degrade path, the stderr capture,
  the stale detection, the empty-symbol detection, and the provisioning decision logic.
- Do not touch the room domain (`internal/room`), the memory domain, or the MCP room tools —
  other work is in flight there.
