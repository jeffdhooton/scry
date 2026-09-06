# Curated constraint: local review evidence and usage

The selected rule is in [`../memory/curated-constraint.txt`](../memory/curated-constraint.txt):

> Scry must build with CGO_ENABLED=0: no CGO, so the single binary can be cross-compiled freely.

It restates the existing `CLAUDE.md` hard constraint and `docs/SPEC.md` §7.
The source's exact SHA-256 is
`1e51679a4a6ab204a50307e33c75d42d436f5d5eedae856ceefb94c7d0eb8578`.
It is selected explicitly; nothing scans this directory automatically.

## Behavior

`ingest.File` reads only the named file inside the named canonical repository.
It requires a local `.git` entry, rejects symlink escapes, nonregular files,
empty/multiline/invalid UTF-8 input, secrets requiring redaction, oversize text
and `--force`. A source is at most 600 bytes; attribution is also bounded so one
accepted rule fits the normal orientation budget. Relative input paths resolve
against the invoking client's working directory, not against `--repo`.

The source goes through the versioned alias of the existing enqueue handler,
the persisted pending queue, the existing extraction worker and ordinary
`resolve.ApplyWith` transaction. The worker retains exact authored text as the
episode summary, as it already does for manual remembers. There is no separate
note store, graph write path, admission controller or new temporal policy.
SourceRef carries the canonical path plus `#sha256=<exact-byte-hash>`; the
existing Cwd/CwdIsRepo fields carry repository provenance through persistence.

Orientation reads completed stored episodes. It verifies text against the
source hash and displays the latest observation for the exact mapped root.
Completion order does not determine revision order. A failed/pending edit does
not replace the last completed source. Earlier episodes keep their original
summary, path, hash, times and attestation. They are accessible through the
existing read-only `memory.export` RPC (all episodes), store `GetEpisode` by ID,
and, when cited by current graph facts, `memory.episodes`.

Unchanged file bytes leave the queue, episodes and cursor unchanged, including
after an mtime-only save. A -> B -> A records the return to A as a new observation.
Enqueue refusal cannot move the cursor. If enqueue succeeds and the cursor
write fails, retry uses the same episode ID and acknowledges the accepted work.
The cursor tracks queue acceptance, not successful model extraction: a provider
failure retains a pending/parked item under the existing queue policy.

## Evidence

Base repository revision: `36dc1e5e42328adaeb7baf5035e8e7def5ae07b5`, plus the
reviewable local diff. No commit, install, push or deployment is part of this run.
The local source manifest and patch in the evidence archive pin the tested bytes.

Archive: `/Users/jeff/scry-local-archives/curated-reset-20260906-8r5n6vsg`.
See [the reset inventory](curated-constraint-reset-2026-09-06.md) for exact private
freeze verification and historical failure evidence. Those experiments were
never used as the implementation base.

`TestCuratedConstraintDurableWorkflow` exercises real ingestion, RPC, pending
storage, queue processing, transactional resolution, store close/reopen and a
separate OS client process with only a socket and cwd. It first restarts with
pending work, processes it, then reopens before orientation. It repeats for an
edited source, tests both enqueue and cursor RPC failures, checks unchanged
replay, retains the original episode and graph provenance, and tests A -> B -> A.
The edited extraction intentionally maps to the same graph triple, demonstrating
correct orientation even while ordinary fact coalescing retains old wording.

`TestCuratedSelectedRepositoryFile` additionally ingests the actual selected file
from this repository into an isolated store and requests orientation at this
repository root after reopen. It is explicitly enabled by
`SCRY_CURATED_SELECTED_ROOT`; ordinary test runs otherwise skip that local case.
All source files are read-only in this case. Mapping/budget tests also cover a
foreign symlink, missing attestation, mismatched summary/hash, older observations
completing last, and a saturated ordinary orientation.

The extractor is a deterministic Go fixture, which intentionally returns an
unhelpful paraphrased summary and real structured entities/facts. These results
prove durable wiring and presentation, not live provider accuracy or latency.
No hosted model was called. Fixture source roots, sockets and stores are isolated.

Measured final run with the actual locally built CLI (`final-focused.log`, SHA-256
`901feb108f3cb1e7056f2d5f7c7a94a63df3384fb54fdc6561e7b980e8706639`):

| Case | Source-to-orientation / query time | Output bytes |
|---|---:|---:|
| Original, including pending restart and final reopen | 914.8 ms | 562 |
| Edited, including accepted retry and final reopen | 164.9 ms | 571 |
| Unrelated root with same basename | 11.6 ms | 71 |
| Sibling root | 9.6 ms | 71 |
| Nested unrelated repository | 10.1 ms | 71 |
| Actual selected repository file, including reopen | 2451.4 ms | 438 |

Full markdown outputs, canonical fixture paths, revision hashes and timings are
in that log. The edit hash is
`f2db747facd05fea29d999a7b3ea31f50e01d2c45bc093ba59a15ce624fac049`.
Both output cases include their exact source and repository; all isolation cases
contain only the ordinary header/footer. These are individual measurements,
not a latency distribution or a full-store performance benchmark. The earlier
checkpoint used a separate test-client process and is retained separately;
the final run uses `review-build/scry` through `SCRY_CURATED_CLIENT_BINARY`.

## Checks and reproduction

Baseline `CGO_ENABLED=0 go test ./...` passed (205.54 s), and
`CGO_ENABLED=0 go vet ./...` passed (1.31 s). Logs and exit codes are retained in
`baseline-*.log` and `baseline-results.json`. The first E2E and boundary runs
also passed; their logs are retained. Checkpoint focused and safety regressions
passed; full checkpoint test/vet passed in 158.63 s / 1.94 s. Final focused,
full tests (37.50 s) and vet (0.95 s) also passed; unchanged package results may
be cached. See `checkpoint-results.json` and `final-results.json` for exit codes,
commands, durations and log hashes. No baseline or new test failures occurred.
Review strengthened the budget fixture to fill both ordinary sections and added
recovery through the public export RPC before that final run.

Exact checkpoint commands, run at the repository root:

```sh
CGO_ENABLED=0 go test ./cmd/scry ./internal/daemon ./internal/memory/ingest ./internal/memory/recall -run Curated -count=1 -v
CGO_ENABLED=0 go test ./internal/memory/resolve ./internal/memory/queue ./internal/memory/store -run 'Historical|History|Alias|Attest|Queue|Park|Repo' -count=1
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go vet ./...
```

The runner also set `GOPROXY=off`, `GOTOOLCHAIN=local`, removed all inherited
`SCRY_`/`GRADER_` live-test/store/socket opt-ins, then set only
`SCRY_CURATED_SELECTED_ROOT=/Users/jeff/workspace/context-stack/scry` for the
explicit isolated source test. The source manifest and logs contain the exact
revision and output evidence. Existing tests were not weakened or removed.

For the final pass, the runner first built a new local artifact with
`CGO_ENABLED=0 go build -o <archive>/review-build/scry ./cmd/scry`, then supplied
`SCRY_CURATED_CLIENT_BINARY=<archive>/review-build/scry` to the same test commands.
Every fresh orientation subprocess used that CLI with the fixture socket and
default budget. `final-results.json` pins the binary SHA-256. This binary was
not copied over an installed or rollback artifact.

## Opt-in usage and reversal

The reproducible local route is the focused test command above, with
`SCRY_CURATED_SELECTED_ROOT` set explicitly to this repository. It creates its
own isolated RPC server, store and deterministic worker; it does not start or
restart an installed daemon. No hook or configuration file changes are needed.

For a separately prepared review binary and an already-running isolated daemon
that supports this source, the CLI procedure is:

```sh
SCRY_MEMORY_SOCKET=/absolute/path/to/isolated.sock /absolute/path/to/review/scry memory ingest \
  --source curated \
  --repo /Users/jeff/workspace/context-stack/scry \
  --path /Users/jeff/workspace/context-stack/scry/docs/memory/curated-constraint.txt
SCRY_MEMORY_SOCKET=/absolute/path/to/isolated.sock /absolute/path/to/review/scry memory queue
# After processing completes, use a new client/session:
SCRY_MEMORY_SOCKET=/absolute/path/to/isolated.sock /absolute/path/to/review/scry memory orient \
  --cwd /Users/jeff/workspace/context-stack/scry
```

Use the same ingest invocation after editing the file. Serialize imports of
this source. An old daemon rejects the versioned enqueue call before any cursor
update. An explicitly configured missing socket fails rather than falling back
to the installed daemon. A dormant daemon retains queued input until its worker
can process it; this change does not configure a provider.

Reversal of this local opt-in is to stop using the review binary and isolated
socket. The commands set no persistent configuration and have no scheduled
ingestion to disable. Leave the isolated evidence intact; no deletion or restore
is necessary. This run grants no authority to run the procedure on a live store.

## Deliberate limits

- This is one authored line, not a Markdown document parser or a home-memory
  crawler. Renaming the file makes a different source; deletion does not withdraw
  a completed rule. Automatic withdrawal is deferred.
- Curated orientation requires the exact canonical mapped repository root.
  A subdirectory, symlink spelling or nested repository does not inherit it;
  pass the canonical root explicitly. No project ownership is guessed.
- Imports of one source must be serialized. The existing cursor API is not a
  concurrent-editor CAS protocol; this slice does not change it.
- The displayed rule is the completed authored source observation. Graph recall
  keeps existing fact/coalescing semantics and may retain older wording. Any
  fact citing a curated episode is excluded from generic orientation bullets,
  even if it also has ordinary episode provenance, to avoid stale or foreign
  rule leakage. Graph evidence is retained and remains queryable.
- The normal budget is unchanged; a caller requesting a smaller budget can omit
  the whole attributed rule. Multiple explicit files can compete for that budget;
  only the selected single-file workflow is the delivery criterion here.
- No new model-quality, power-loss durability, live-store scale or independent
  source-review claim is made. Queue durability and provider/parking behavior
  remain those of the existing engine. Stop here for review.
