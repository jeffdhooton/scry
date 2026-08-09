# Partial-index observability and two-tier language detection

**Date:** 2026-08-09
**Status:** approved, ready to plan

## Problem

17 of 44 indexed repos report `status: "partial"`. Nothing records *why*.

Three defects compound:

1. **Errors are discarded.** `buildAtLayout` collects `indexerErrs`, prints them to stderr at
   `internal/index/builder.go:316`, and writes only the string `"partial"` into the manifest.
   Once indexing finishes, which indexer failed and for what reason is unrecoverable without
   re-running the build and watching stderr. Neither `scry status` nor `scry doctor` can say more
   than "partial".

2. **A missing indexer and a crashed indexer are the same state.** "scip-python was never
   installed" is a one-line fix for the operator; "scip-go crashed parsing a file" is a bug in
   scry. Today both produce identical output. `scry doctor` does flag a globally missing indexer,
   but never connects that to the repos it actually degraded.

3. **A 1% detection threshold triggers full indexer runs on incidental languages.**
   `detectLanguages` (`internal/index/builder.go:363`) admits any language holding ≥1% of source
   files. Measured on `childscribe-beta-r4`: 855 PHP, 110 TS/JS, 37 Python files, and zero Python
   marker files. Python is 3.7% — over the line — so `scip-python` runs, isn't installed, and the
   repo drops to `partial`. 855 PHP files' worth of complete index is reported as degraded because
   of 37 incidental scripts.

The observable result: on the user's primary stack (Laravel + TS), every `childscribe-*` repo has
been serving silently degraded results with no way to learn what was missing.

## Non-goals

- Auto-remediation. `doctor` will report that `scip-python` is missing and which repos it affects;
  it will not install it or trigger re-indexing. (Considered and explicitly scoped out.)
- Fixing whatever causes `idea-planning` (typescript, go — no Python) to be `partial`. That is a
  *second, unknown* cause. This work makes it diagnosable; the fix follows from what it reports.
- Any change to BadgerDB contents or query behavior.

## Design

### 1. Data model

`Manifest` (`internal/index/builder.go:34`) gains one field:

```go
type Manifest struct {
    // ... existing fields unchanged ...
    Indexers []IndexerResult `json:"indexers,omitempty"`
}

// IndexerResult records the outcome of one language indexer for one build.
type IndexerResult struct {
    Language  string  `json:"language"`
    Status    string  `json:"status"`             // ok | missing | failed | skipped
    Tier      string  `json:"tier"`               // primary | incidental
    FileCount int     `json:"file_count"`
    Share     float64 `json:"share"`              // fraction of detected source files
    Error     string  `json:"error,omitempty"`    // set when status is missing|failed
    Remedy    string  `json:"remedy,omitempty"`   // set when status is missing
}
```

Status values:

| Status | Meaning |
|---|---|
| `ok` | indexer ran, output parsed |
| `missing` | indexer binary not installed (`errors.Is` matched a not-found sentinel) |
| `failed` | indexer ran and errored, or its output failed to parse |
| `skipped` | language is incidental; indexer deliberately not invoked |

This is a purely additive field in `manifest.json`. Legacy manifests unmarshal with
`Indexers == nil` and every existing consumer keeps working. **No `store.SchemaVersion` bump** —
the BadgerDB contents are untouched, so the 44 existing repos are not force-reindexed. Repos pick
up the new field on their next natural build; until then they render as they do today.

### 2. Classification reuses existing sentinels

Every source package already exports a not-found sentinel carrying its own install instructions:

- `python.ErrIndexerNotFound` — "scip-python not found on PATH; install with: npm i -g @sourcegraph/scip-python"
- `typescript.ErrIndexerNotFound` — "scip-typescript not found on PATH; install with: npm i -g @sourcegraph/scip-typescript"
- `golang.ErrIndexerNotFound` — "scip-go not found and could not be installed"
- `php.ErrPhpNotFound` — "php interpreter not found on PATH; install PHP 8.3+ to index PHP repos"

Classification is therefore `errors.Is(err, <sentinel>)` → `missing`, anything else → `failed`,
with `Remedy` lifted from the sentinel's message. **No changes inside `internal/sources/*`.**

The four near-identical indexer blocks at `builder.go:172-208` collapse into one table-driven loop
over `{language, runFn, notFoundErr}` so recording is uniform rather than duplicated four times.
This is the only structural refactor in scope.

### 3. Two-tier language detection

`detectLanguages` returns tiered results instead of a flat `[]string`. A language present at ≥1%
of source files is classified:

- **primary** — holds ≥10% of source files, **or** the repo has a marker file for it
- **incidental** — clears 1% but is neither

Marker files:

| Language | Markers |
|---|---|
| typescript / javascript | `package.json`, `tsconfig.json` |
| go | `go.mod` |
| php | `composer.json` |
| python | `pyproject.toml`, `requirements.txt`, `setup.py`, `Pipfile` |

Markers are checked at the repo root only. The marker is the load-bearing signal: a repo with a
real component in a language nearly always declares one, and a repo with 37 stray scripts does not.
The 10% share is the fallback for undeclared-but-substantial code.

Behavior by tier:

- **primary** → indexer runs. `missing` or `failed` ⇒ repo status `partial`.
- **incidental** → indexer is **not invoked**. Recorded as `skipped` with its share and file count.
  Does **not** degrade repo status.

`Manifest.Languages` continues to list primary languages only, preserving its current meaning for
existing consumers. Incidental languages are visible in `Indexers` as `skipped` entries.

Applied to `childscribe-beta-r4`: Python is 3.7% with no marker → incidental → skipped → the repo
returns to `ready`, which is honest, because its PHP and TypeScript indexes are complete.

### 4. Status derivation

Computed from the result set, replacing the `len(indexerErrs) > 0` check at `builder.go:315`:

- `ready` — every primary indexer is `ok`
- `partial` — at least one primary indexer is `missing` or `failed`, and at least one is `ok`

`Manifest.Status` also documents a `broken` value. It is vestigial and stays that way: when no
indexer produces output, `buildAtLayout` returns the first error and writes no manifest at all, so
`broken` is never persisted. This design does not introduce a path that writes it.

### 5. Surfacing

**`scry status`** — `RepoStatusEntry` (`internal/daemon/methods.go:144`) gains
`Indexers []index.IndexerResult` with `json:"indexers,omitempty"`. Both construction sites in
`handleStatus` (the registry loop and the on-disk scan) must set it. `cmd/scry/status.go` renders
via `printJSON`, so `--pretty` is indented JSON — no custom renderer exists and none is added here.

**`scry doctor`** — two changes in `internal/doctor/doctor.go`:

- `checkCurrentRepo` (the `repo.current` check) appends a per-indexer breakdown to `Detail` for
  every non-`ok` entry, and populates `Remedy` from the first `missing` entry. Falls back to
  today's output when `Indexers` is nil (legacy manifest).
- A new check `indexers.impact` under `CategoryIndexers`: for each indexer that the existing
  environment checks found missing, scan `~/.scry/repos/*/manifest.json` and count repos with a
  primary entry for that language. Reports e.g. "scip-python missing — affects 17 indexed repos"
  at `StatusWarn`, with the install command as `Remedy`. `StatusPass` when nothing is missing.

**`scry_status` MCP tool** inherits the new field through the existing RPC; no separate work.

## Testing

Unit, in `internal/index`:

- classification: a wrapped `python.ErrIndexerNotFound` → `missing` with remedy; an arbitrary error
  → `failed` with the message preserved; a `nil` error → `ok`
- tiering: marker-with-low-share → primary; high-share-without-marker → primary;
  low-share-without-marker → incidental; below 1% → absent entirely
- status derivation: all-ok → `ready`; one primary `missing` → `partial`; one primary `failed` →
  `partial`; incidental `skipped` alongside all-ok primaries → `ready`
- manifest round-trip: a legacy manifest JSON with no `indexers` key unmarshals to `Indexers == nil`
  and every existing field is preserved on re-marshal

Integration, in `internal/index`:

- build a fixture repo with a dominant language plus a stray `.py` file and no Python marker;
  assert the manifest is `ready`, carries a `skipped` python entry, and that `scip-python` was
  never invoked

In `internal/doctor`:

- `checkCurrentRepo` against a manifest with a `missing` entry → `StatusWarn`, detail names the
  language, remedy is the install command
- `checkCurrentRepo` against a legacy nil-`Indexers` manifest → today's output, no panic
- `indexers.impact` counts only repos with a *primary* entry for the missing language

## Verification

Beyond the test suite: rebuild the index for `childscribe-beta-r4` and confirm it reports `ready`
with a `skipped` python entry, then re-run `scry doctor` in a repo that stays `partial` (e.g.
`idea-planning`) and confirm it now names the failing indexer and its error — which is the input to
diagnosing that second, still-unknown cause.

## Files touched

- `internal/index/builder.go` — `IndexerResult`, tiered detection, table-driven indexer loop, status derivation
- `internal/index/builder_test.go` — unit + integration coverage
- `internal/daemon/methods.go` — `RepoStatusEntry.Indexers`, both construction sites
- `internal/doctor/doctor.go` — `repo.current` detail, new `indexers.impact` check
- `internal/doctor/doctor_test.go` — doctor coverage
- `README.md` — "Known limitations" note on incidental languages being skipped
- `docs/DECISIONS.md` — the two-tier detection call and why the marker file is the primary signal
