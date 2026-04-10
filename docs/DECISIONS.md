# scry — decision log

Architectural and scope calls that deserve a durable written record. One
entry per decision. Newest at the top. Each entry must answer: what, why,
what would change our minds.

This file resolves the open questions in `docs/SPEC.md` §15. The PHP
calibration findings live in `docs/PHP_CALIBRATION.md`.

---

## 2026-04-10 — Vendor scip-php as a PHAR built from a pinned main commit

**Decision:** When P1 lands PHP support, scry will ship `scip-php` as a
PHAR archive built from a pinned `davidrjenni/scip-php` commit (currently
`97a2d8d`, 2026-03-31). Not from Packagist. Not from the docker image.
Not as a `composer require` step the user has to perform.

**Why:** The day-1 calibration (see `docs/PHP_CALIBRATION.md`) verified
three install paths and all of them broke for separate reasons:

1. Packagist `v0.0.2` is from 2023, requires `google/protobuf ^3.22`
   which has security advisory PKSA-tcfz-w4fm-hhk9, and modern Composer
   refuses to install it.
2. The published `davidrjenni/scip-php:latest` docker image is also
   v0.0.2 with bundled PHP 8.2.5 and an old `nikic/php-parser` that
   chokes on PHP 8.4 syntax (`Class_::verifyModifier` undefined).
3. Running scip-php from its own clone, pointed at an external project,
   crashes (`Int_::KIND_INT` undefined) because PHP autoloads
   `nikic/php-parser` from the target's `vendor/` and can't host two
   versions of the same package in one process.

The only install path that worked was `composer require --dev
davidrjenni/scip-php:dev-main -W` from inside the target project. That
modifies the user's `composer.json`/`composer.lock`, which we can't ask
agents or users to do silently. A PHAR with frozen internal dependencies
sidesteps the autoloader collision and gives us a single binary to
download into `~/.scry/bin/`, matching the auto-download flow already
specified for `scip-typescript` and `scip-go`.

**What would change our minds:** scip-php cuts an actual release that
ships with current php-parser, OR a maintained Packagist publish appears,
OR the PHAR build proves brittle in CI (in which case we fall back to a
pinned VCS install with documented `composer.json` modification).

---

## 2026-04-10 — One binary, not two (`scry` is `scryd`)

**Decision (SPEC §15 Q1):** Build a single binary called `scry`. Daemon
mode is `scry start --daemon` or auto-spawned on first CLI call. The
client and the daemon are the same Go program with different entry
points selected by subcommand.

**Why:** Less ops surface, smaller install footprint, one version to
keep in sync. Cobra makes it trivial to gate daemon-only subcommands
behind a flag. The cost of merging the two is negligible — the
client-side code paths are tiny (open socket, send JSON-RPC, print
response). trawl took the same shape.

**What would change our minds:** if the daemon binary balloons past
50 MB because of indexer dependencies and the CLI is invoked thousands
of times per session, separate binaries would amortize startup cost.
We are nowhere near that and the CLI does no parsing in P0 anyway.

---

## 2026-04-10 — Daemon log: `~/.scry/scryd.log`, size-rotated, keep 3

**Decision (SPEC §15 Q2):** zerolog JSON output to `~/.scry/scryd.log`.
Rotate on size: 10 MB per file, keep the most recent 3 (one current +
two backups). No time-based rotation. No external rotator dependency.

**Why:** Size-based rotation is simpler than time-based and matches the
"this tool runs as long as you're working" lifecycle better than a daily
cron. 30 MB total cap is enough to debug a multi-day session and small
enough to fit on any disk. One backup-of-the-backup is the minimum that
survives a rotate-during-crash. No external rotator means no extra
dependency or systemd unit.

**What would change our minds:** users want longer history for
post-incident debugging, in which case bump to 50 MB × 5 backups, or
someone asks for daily rotation for log-shipping reasons.

---

## 2026-04-10 — In-memory cache: all-in-memory until manifest tells us otherwise

**Decision (SPEC §15 Q3):** P0 reads BadgerDB directly per query — no
in-process cache. P1's daemon mode keeps the entire BadgerDB index
loaded into Go maps on warm-up and queries hit the maps directly.
BadgerDB stays as the durable backing store; the in-memory layer is
a read-through mirror, rebuilt from BadgerDB on daemon start.

No LRU. No TTL. The whole index is small enough to live in RAM
(SPEC §10's targets — 500 MB for 100k LOC, 3 GB for 1M LOC — assume
this).

**Why:** The simplest thing that meets the latency target. SCIP indexes
for normal repos are small (hoopless_crm at 174k PHP LOC = 14 MB SCIP
file → maybe 50 MB resident as Go structs). LRU or TTL would buy
nothing at that scale and add invalidation bugs. If a single repo
threatens to blow the RAM budget we'll add a per-repo cap and evict
oldest-touched repo, not LRU within a repo.

**What would change our minds:** indexing a >5M LOC monorepo where the
in-memory representation exceeds 8 GB, OR a query pattern emerges where
re-deserializing BadgerDB records on every query is faster than holding
them resident.

---

## 2026-04-10 — Auto-download pinned indexers, never auto-update

**Decision (SPEC §15 Q4):** P0 requires manual `npm i -g
@sourcegraph/scip-typescript` for the user. P1 auto-downloads
`scip-typescript`, `scip-go`, and the `scip-php` PHAR into `~/.scry/bin/`
on first use, verifying each binary against a SHA256 list compiled into
the scry binary. Pinned versions update only when scry itself is
updated. Never auto-update an indexer behind the user's back.

**Why:** Reproducibility and trust. A code-intelligence tool that
silently swaps its underlying parser changes the meaning of every query
result. The pinned-and-shipped-with-scry model means a given scry
release always produces the same index for the same code. Easier to
reason about, easier to debug, easier to bisect when something breaks.

In P0 we skip auto-download because P0 is "validate the architecture"
not "validate the install story." Manual install is fine when there's
exactly one user (the build agent).

**What would change our minds:** scip-typescript or scip-go ship breaking
fixes that affect correctness — in which case we ship a scry patch
release that bumps the pin.

---

## 2026-04-10 — Global config in `~/.scry/config.yaml`, per-repo `.scryignore`

**Decision (SPEC §15 Q5):** Daemon settings (log level, RAM cap, socket
path, indexer paths) live in `~/.scry/config.yaml` via viper. Per-repo
ignore patterns live in a `.scryignore` file at repo root, gitignore-
style syntax. No per-repo config file beyond `.scryignore`. Defaults are
sensible — most users will never touch either file.

**Why:** Mirrors the `.gitignore` mental model that every developer
already has. Global daemon settings are a singleton concern; per-repo
"don't index this" is a workspace concern. Splitting the two by file
location keeps responsibilities clean.

**What would change our minds:** users want per-repo overrides for
non-ignore settings (e.g., "this repo should always index test files,
that one shouldn't"). If that comes up, add a per-repo `.scry.yaml`
that mirrors a subset of the global schema.

---

## 2026-04-10 — `scry symbols` returns up to `--limit N` (default 1000), paginate above

**Decision (SPEC §15 Q6):** `scry symbols <file>` returns all symbols
in the file by default, capped at `--limit 1000`. If the file has more
than 1000 symbols, the response includes `"truncated": true` and a
`"next_offset"` cursor. Pagination uses `--offset N --limit N`.

**Why:** 1000 symbols covers >99% of real-world files. The truncation
flag is honest about the cap. Cursors instead of opaque tokens because
the underlying storage is ordered and an offset is sufficient — no
need for stable cursor tokens until queries return data that can shift
between requests.

**What would change our minds:** generated files (e.g. protobuf bindings)
routinely exceed the cap and users hit the truncation often. If that
happens, raise the default to 5000.

---

## 2026-04-10 — Test fixtures: synthetic small repo + integration target opt-in

**Decision (SPEC §15 Q7):** Unit tests use a hand-crafted ~15-file
synthetic TypeScript repo committed under `internal/testdata/ts-fixture/`.
Integration tests against a real OSS repo (`microsoft/vscode` is the
SPEC's stress target) live in a separate `_integration_test.go` file
that requires `SCRY_INTEGRATION=1` to run, and the repo is cloned
into a tmp dir on demand, never committed. CI runs the unit suite
only — integration is local-developer.

**Why:** Fast unit tests stay fast. Real-repo accuracy benchmarking
is essential but cannot be in CI without tying CI to a network clone
and to upstream churn that breaks results unrelated to scry changes.

**What would change our minds:** a frozen fixture-repo tarball gets
hosted somewhere (an scry-test-fixtures release) and integration tests
can run against the frozen version in CI without a live clone.

---

## 2026-04-10 — Indexer failures: skip the file, mark repo partial, log loud

**Decision (SPEC §15 Q8):** When `scip-typescript` (or any other
indexer) fails on a single file or batch, scry skips the failing files,
emits a structured warning to the daemon log, marks the repo's manifest
with `"status": "partial"` and a `"failed_files"` count, and continues
indexing. Queries still work; the user can inspect failures via
`scry status --verbose`. Falling back to tree-sitter-only is deferred
to P2+.

**Why:** Refusing to index a 10k-file repo because of 3 broken
TypeScript files is the wrong default. Partial-but-correct beats
nothing-because-perfect. The status flag is honest about coverage so
agents can decide whether to fall back to grep.

**What would change our minds:** a class of failures appears that taints
the rest of the index (e.g., a cross-file type resolution error that
poisons every file referencing the broken type). If that's possible we
mark the whole repo as `"status": "broken"` and refuse queries until a
clean reindex.

---

## 2026-04-10 — Schema evolution: reindex from scratch, version in manifest

**Decision (SPEC §15 Q9):** The BadgerDB schema is versioned via a
`schema_version` integer in each repo's `manifest.json`. When scry
starts and finds an index with an older schema version than its
compiled-in `currentSchemaVersion`, it deletes the BadgerDB directory
and reindexes from scratch. The reindex is announced loudly: log
warning, CLI prints "scry: schema upgrade, reindexing <repo>" before
running, exit nonzero if the reindex fails.

No migration code. No backwards-compatible read paths.

**Why:** Reindexing from scratch is fast (<60s for 100k LOC per the
SPEC targets). Migration code is a long-tail bug factory and an
ongoing maintenance tax for v1 with one user. If reindex takes
multiple minutes for the largest repos, that's a one-time cost per
schema bump and the user can be told to expect it.

**What would change our minds:** v2 onwards if scry has external users
with multi-million-LOC monorepos where reindex takes >30 minutes and
schema changes happen often. At that point, write migrations.

---

## 2026-04-10 — Daemon shutdown: 5 second grace, then SIGKILL

**Decision (SPEC §15 Q10):** `scry stop` sends SIGTERM to the daemon,
which finishes any in-flight queries, flushes pending BadgerDB writes,
closes the socket, and exits. If it doesn't exit within 5 seconds,
`scry stop` sends SIGKILL.

**Why:** Standard. Long enough for a clean shutdown of a normal
workload (queries are <50ms, BadgerDB flush is fast), short enough that
a stuck daemon doesn't make the user wait. Matches what
systemd/supervisord do by default.

**What would change our minds:** a real workload routinely exceeds 5s
to flush (probably means an oversized BadgerDB write batch that should
be split). Fix the underlying issue, don't extend the timeout.

---

## 2026-04-10 — Module path: github.com/jeffdhooton/scry

**Decision:** Use `github.com/jeffdhooton/scry` as the Go module path,
mirroring trawl's `github.com/jeffdhooton/trawl`. Repo is local-only
today; the path is forward-compatible with a public GitHub repo at the
same location.

**Why:** Matches the sibling project. No friction if/when the repo gets
pushed to GitHub. No leaked organization name to rename later.

**What would change our minds:** the project moves under an
organization on GitHub. At that point a one-time `go mod edit -module`
plus an import rewrite handles it.
