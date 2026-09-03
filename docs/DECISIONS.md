# scry — decision log

Architectural and scope calls that deserve a durable written record. One
entry per decision. Newest at the top. Each entry must answer: what, why,
what would change our minds.

This file resolves the open questions in `docs/SPEC.md` §15. The PHP
calibration findings live in `docs/PHP_CALIBRATION.md`.

---

## 2026-08-19 — Watchers are bounded by a descriptor budget, not a directory cap

**Decision:** The file watchers are bounded by a shared file-descriptor budget
(a quarter of the soft NOFILE limit, clamped to 2048–16384) rather than by
`maxWatchedDirs` alone. Adding a directory is charged its real cost, which is
platform-specific: on the kqueue backends (macOS, BSD) it is `1 + len(entries)`,
elsewhere `1`. Repos are watched lazily — bootstrap watches the most recently
indexed repos until the budget is spent, and a query for any other repo starts
its watcher on demand, evicting the least recently used one. `raiseNOFILE` now
raises the soft limit to a fixed 65536 instead of the hard limit. A periodic
governor samples the process's actual descriptor count and evicts watchers while
it exceeds half the soft limit.

**Why:** `maxWatchedDirs = 2048` counted directories, which does not bound
descriptors. fsnotify's kqueue backend cannot watch a directory as a unit the
way inotify does, so `watchDirectoryFiles` opens a descriptor for every entry
inside every watched directory. A 20-directory tree — 1% of the cap — measured
4021 descriptors. With 125 registered repos all watched at startup, one daemon
reached ~131,000 descriptors, about 91% of every open file on the machine, and
unrelated processes began failing with `ENFILE` on the system-wide file table.
Raising NOFILE to the hard limit made this possible: macOS reports that limit as
`RLIM_INFINITY`, so the process had no ceiling at all while competing for a
global resource. Measured after the change: the same 74 repos cost 14,239
descriptors instead of ~131,000.

The governor exists because the budget alone is not sufficient. kqueue keeps
opening descriptors after the initial Add — writing into a watched directory
makes `dirChange` start watching every entry that has appeared since — so a
build campaign grows the set past whatever the walk reserved.

**What would change our minds:** an fsnotify release that watches directories
on macOS without a descriptor per entry (FSEvents-backed, say) would remove the
platform asymmetry and let the dir cap bound descriptors again. On a
Linux-only deployment the budget is nearly inert and could be relaxed.

---

## 2026-08-19 — Watches are removed before the fsnotify watcher is closed

**Decision:** `closeWatcher` removes every path from an `fsnotify.Watcher`
before calling `Close`, instead of calling `Close` alone.

**Why:** fsnotify v1.9.0's kqueue backend leaks every descriptor on `Close`.
`Close` marks the watcher closed via `shared.close()` and only then loops
calling `Remove` for each watched path — but `remove()` begins with
`if w.isClosed() { return nil }`, so it returns before reaching its
`unix.Close(info.wd)`. Measured: closing a watcher holding 1155 descriptors
released 1. Only the close-pipe descriptor is actually freed.

This matters more after the budget change than before it. Eviction is the
budget's release valve; if closing a watcher frees the accounting but not the
descriptors, the daemon over-commits and drifts upward every time it evicts —
strictly worse than not evicting at all. Removing the watches first takes the
working path through `remove()`, which closes each directory's descriptor and,
via `watchesInDir`, the per-file descriptors kqueue opened inside it.

**What would change our minds:** an upstream fix. This is a workaround for a
library bug, not a design preference, and `closeWatcher` should collapse back to
`fsw.Close()` once a released fsnotify closes its own descriptors. It is worth
reporting upstream.

---

## 2026-08-19 — A daemon that is alive but not serving is retired, not orphaned

**Decision:** When a starting daemon finds a PID file naming a live process that
fails the socket liveness ping, it sends that process `SIGTERM` before taking
over the socket.

**Why:** The previous behaviour treated "process alive but not answering" as a
stale socket: it removed the socket file and started anyway, leaving the old
process running forever. That is self-reinforcing. A daemon starved of
descriptors cannot `accept`, so it fails the ping, so the next start orphans it
— while it still holds every descriptor its watchers opened. Four daemons were
observed alive within a two-hour window, two of them referencing the same
socket path, each independently watching every indexed repo. The orphan is not
serving anyone by definition of having failed the ping, so terminating it is
strictly better than leaving it holding a five-figure descriptor set.

**What would change our minds:** evidence that a healthy daemon can fail the
200ms ping under normal load. That would make this a liveness-detection problem
rather than an orphan problem, and the ping would need to become more patient
before it is allowed to justify a SIGTERM.

---

## 2026-08-13 — Explicit npm provisioning for TypeScript and Python indexers

**Decision:** Superseding the manual-only portions of the 2026-04-10 and
2026-04-11 indexer-install decisions, scry may install the official
`@sourcegraph/scip-typescript` and `@sourcegraph/scip-python` packages globally
when the user explicitly runs `scry install` or `scry doctor --fix`. The flow
requires `npm` on PATH, invokes the exact documented `npm i -g` command, and
then resolves and invokes the resulting binary with `--version`. If npm is
missing, scry names that prerequisite and prints the command instead of
silently skipping the tool.

A daemon that predates a newly installed binary may retain a PATH that cannot
resolve it. Doctor reports that ordering as a warning with the explicit remedy
`scry daemon restart`; `doctor --fix` performs the restart after provisioning.
The planning decision remains pure and takes all PATH and timestamp facts as
inputs, so tests never need npm, network access, or an installed indexer.

**Why:** Both packages install cleanly through npm, while the previous manual
remedy left an otherwise automated setup broken and allowed a stale daemon to
keep reporting a just-installed indexer as missing. The explicit commands are
the consent boundary for the global install; indexing itself still never
silently installs an npm package.

**What would change our minds:** official, checksum-verifiable standalone
release binaries would move these tools to the pinned GitHub-release path.
Conversely, an npm install that cannot be verified non-interactively would
move the affected tool back to a surfaced manual remedy.

---

## 2026-04-12 — Test coverage: aggregate-only, user-generated, four format parsers

**Decision:** Test coverage indexing ships as a post-processor in the build
pipeline (`internal/sources/coverage/`). Key design calls:

1. **Aggregate coverage only (Phase 1).** Per-test coverage (which *specific*
   test covers which function) would require running each test individually
   or language-specific tooling that isn't mature. Aggregate coverage ("is
   this function hit by *any* test?") is universally available and answers
   the 80% use case.

2. **scry does not run tests.** The user generates coverage files with their
   normal test tooling (`go test -coverprofile`, vitest --coverage, etc.).
   `scry init` detects and consumes the artifacts. Matches the existing
   pattern: scry doesn't run `scip-typescript`, it consumes its output.

3. **Four format parsers:** Go coverprofile (`cover.out`), Istanbul JSON
   (`coverage-final.json` — vitest/jest), Clover XML (`coverage.xml` —
   PHPUnit), Python coverage.json (`coverage.py json`). Auto-detected by
   well-known paths in the repo root.

4. **Single-line def span expansion for scip-go.** scip-go emits definition
   occurrences where EndLine == Line (just the function signature). The
   coverage join extends each such span to the next definition in the same
   file so function body lines match correctly.

5. **Schema version bumped to 2.** New `cov:` key prefix. Forces a clean
   rebuild on existing indexes.

**What would change our minds:** if per-test coverage becomes trivially
available (Go 1.25 test coverage contexts, vitest native per-test output),
add a Phase 2 that writes `cov:<symbol_id>:<test_id>` edges.

---

## 2026-04-12 — MCP call logging to ~/.scry/logs/mcp-calls.jsonl

**Decision:** Every MCP tool call writes a JSONL line to
`~/.scry/logs/mcp-calls.jsonl` with timestamp, tool name, symbol, repo,
result count, latency, and error (if any). Append-only, no rotation.

**Why:** Dogfooding visibility. Lets us see how Claude Code actually uses
scry — which tools get called, how often, what latency looks like in
practice. Zero overhead (one file append per call).

**What would change our minds:** if the log file grows unbounded on heavy
users. Add rotation or size cap when it becomes a real problem.

---

## 2026-04-11 — Python: require manual `npm i -g @sourcegraph/scip-python`, shim Python version at runtime

**Decision:** Python indexing ships via `scip-python` (Sourcegraph's
Pyright fork) as a shell-out from `internal/sources/python/indexer.go`.
The wrapper:

1. Requires the user to `npm i -g @sourcegraph/scip-python` manually —
   matching the scip-typescript precedent, no auto-download.
2. Builds a PATH shim (`~/.scry/bin/python-shim-<sha>/`) that maps
   `python` and `python3` to a compatible interpreter (searches
   `python3.13` → `python3.12` → `python3.11` → `python3.10`), then
   prepends that shim to PATH when running scip-python.
3. Passes `NODE_OPTIONS=--max-old-space-size=8192` to avoid OOM on
   larger projects.
4. Passes `--project-name` derived from `pyproject.toml` →
   `setup.cfg` → `setup.py` → repo dir basename.
5. Passes `--project-version 0.0.0` on non-git repos (scip-python's
   default version detection crashes with a TypeError inside
   `ScipSymbol.normalizeNameOrVersion` when there's no git rev).
6. Respects `$VIRTUAL_ENV`, `.venv/`, `venv/`, `env/` for dependency
   resolution but never fails if no venv is present — the indexer
   degrades gracefully to "project symbols work, external imports
   don't resolve to source" which is still useful.

**Why require manual install:** same reasons as scip-typescript.
scip-python is an npm package with no GitHub binary releases. Auto-
download would require bundling Node + the entire npm tree (thousands
of files, deep dep graph), which is the PHAR rabbit hole with a bigger
radius. `scry doctor` now checks for scip-python on PATH and prints
the exact `npm i -g` line on Warn, keeping the install story surfaced
instead of hidden.

**Why the PATH shim:** scip-python 0.6.6's bundled Pyright only
recognizes Python 3.10-3.13. On a machine with 3.14+ as the system
`python3` (Homebrew's current default), Pyright prints "Python version
3.14 from interpreter is unsupported" and silently emits a 0-document
SCIP index — "successfully wrote SCIP index to index.scip" followed
by 66 bytes of empty metadata. The failure mode is indistinguishable
from success at the CLI level. Three options considered:

1. **Document the limitation + fail loudly** — user has to install an
   older Python and `PATH=...python3.12... scry init`. Works but adds
   ceremony.
2. **Write a `pyrightconfig.json` to the repo** — pollutes the user's
   working tree, overrides any existing config, and has subtle
   conflicts when the project already has `[tool.pyright]` in
   pyproject.toml (the existing sections get ignored).
3. **PATH shim** — a per-target cached symlink dir that makes `python`
   and `python3` point at a compatible interpreter for the duration of
   the scip-python run. Transparent to the user, no repo pollution,
   respects existing pyright configs.

Option 3 won. The shim dir is cached under `~/.scry/bin/python-shim-
<sha256[:12]>/` keyed by the resolved target binary path, so repeated
runs against the same Python reuse the same dir. Cheap to (re)create,
invisible in normal operation, easy to garbage collect.

**Why `--project-version 0.0.0` on non-git repos:** discovered
empirically. On a non-git directory, scip-python crashes with:

```
TypeError: Cannot read properties of undefined (reading 'indexOf')
  at normalizeNameOrVersion (ScipSymbol.ts:23:11)
  at makePackage (symbols.ts:21:23)
```

because its default version detection reads from `git rev-parse`,
which returns undefined outside a git repo, and the downstream code
doesn't guard against that. Passing any non-empty version string
sidesteps the crash. We check for `.git/` at the repo root and only
override when it's absent, so git repos keep their rev-based versions.

**Real numbers on pydantic** (validation target, git clone of
`pydantic/pydantic` at commit `b1bf19445`):

| Metric | Value |
|---|---|
| Cold index | 11.0s wall (scip-python 10.2s + parse ~800ms) |
| Documents | 107 |
| Symbols | 8,087 |
| Definitions | 7,532 |
| References | 35,986 |
| Call edges | 29,375 |
| Implementations | 314 |

Sample queries on the indexed pydantic:

- `scry refs BaseModel` → 137 occurrences across 2 classes
  (`pydantic.main.BaseModel` + the v1 compat shim at
  `pydantic.v1.main.BaseModel`), warm query <10ms.
- `scry defs BaseModel` → both definition sites with accurate
  file:line:context (`pydantic/main.py:119` and `pydantic/v1/main.py:333`).
- `scry callers model_validator` → 4 call edges.
- `scry impls ConfigDict` → 2 matches (v2 + v1 variants).

Symbol shape: `scip-python python pydantic <git-rev> <descriptor>`.
Same five-token pattern as every other SCIP indexer scry ingests, so
the parser, walker pipeline, and MCP tool handlers work unchanged.
Python needed zero changes in `internal/sources/scip/parse.go` or
downstream — the generality of the external-symbol synthesis and the
MCP compound-symbol parser both paid off.

**What we did NOT build:**

- **Framework-aware post-processors.** Python has no single dominant
  framework analogous to Laravel. Django, Flask, FastAPI, Pydantic,
  pytest — each would benefit from different pattern extraction, and
  none are universal enough to justify the P2-level work we did for
  PHP. Defer until a specific pain point appears.
- **Auto-generating a `pyrightconfig.json`.** Considered and rejected:
  see "Why the PATH shim" above. We don't write anything into the
  user's repo.
- **Auto-installing scip-python via `npx --yes`.** Considered for a
  zero-click path but dropped — silent npm installs from inside
  another command feel janky, and the user already needs to know they
  have Python available for indexing to make sense.

**What would change our minds:**

- scip-python's bundled Pyright gets bumped to support Python 3.14+,
  making the PATH shim unnecessary for newer-Python machines. We'd
  keep the shim code for <3.13 fallback but short-circuit when the
  system python is already compatible.
- A real Python repo reveals a persistent symbol-resolution gap that
  mirrors Laravel's non-PSR-4 gap (routes defined in a string key,
  dynamic imports, etc.). At that point we'd add a Python
  post-processor in the same shape as `internal/sources/php/`.
- Sourcegraph publishes a native binary of scip-python (no Node
  required). Would let us auto-download like scip-go.

---

## 2026-04-11 — Claude Code integration via `claude mcp add`, not ~/.claude/settings.json

**Decision:** `scry setup` registers the scry MCP server by shelling out
to `claude mcp add --scope user --transport stdio scry -- <bin> mcp`,
which is Claude Code's official CLI for managing MCP servers. It does
**not** hand-edit any config file directly.

**Why:** Claude Code reads MCP server config from `~/.claude.json`
(~200 KB of session state that Claude Code owns and rewrites
frequently), NOT from `~/.claude/settings.json` (which handles hooks,
enabled plugins, and marketplaces). The first iteration of `scry setup`
wrote to the wrong file. The skill installed correctly, `scry mcp`
spoke the protocol correctly, and the MCP manager UI even showed scry
as connected via background polling — but Claude Code never routed
symbol queries to scry because the current session's tool registry
was snapshotted at startup without scry in it.

The fix has two parts:

1. **Delegate to `claude mcp add`.** That's the documented path;
   Claude Code's internal storage format for `~/.claude.json` is not a
   stable API, and hand-editing a 200 KB state file also risks
   reformatting unrelated keys.
2. **Clean up the wrong-file write if it's still present.** On install,
   `setup.Install` scans `~/.claude/settings.json` for a leftover
   `mcpServers.scry` entry from the earlier buggy iteration, backs the
   file up, and strips the stale key. Best-effort; failure isn't fatal.

**Load-bearing constraint for future MCP-related work:** never silently
edit a config file for a tool the user didn't explicitly target. The
settings.json bug was discovered only because the user tried a query
and noticed Claude fell back to Grep — exactly the kind of silent
failure that burns trust. Any new target (Cursor, Codex, Continue, Zed,
etc.) should go through that tool's official CLI if one exists, or ask
for confirmation before touching its config.

**What would change our minds:** `claude mcp add` grows a hard-to-work-
around limitation (e.g. refuses to register local binary paths in
user scope). At that point we'd fall back to hand-editing
`~/.claude.json` with surgical JSON editing (finding the byte range of
just the `mcpServers` key and replacing its value without re-ordering
the rest), NOT to a full JSON round-trip.

---

## 2026-04-11 — `scry doctor` is read-only by default; `--fix` for fast remediation only

**Decision:** `scry doctor` runs 13 diagnostic checks across
Environment, Daemon, Indexers, Claude Code integration, and the current
repo's index state. Every check is strictly read-only — no subprocess
side effects, no file writes, no daemon spawning. Results are rendered
as a categorized ✓/⚠/✗/— checklist with per-check remediation hints,
and exit 1 if any check fails (warnings are advisory and don't affect
exit code).

A separate `--fix` flag runs auto-remediation for checks that have a
registered fixer. Fixers are capped at <1 second each and may only
perform idempotent, reversible actions. Long operations — running
`scry init` (10-60s), installing PHP, installing `scip-typescript` via
`npm i -g` — are **never** run by `--fix` even when technically
possible. The first rule of `scry doctor --fix` is: no surprising waits.

**Why read-only by default:**
- Diagnostics you can re-run at any time without side effects are safe
  to put in hot paths (bug reports, CI checks, shell rc hooks).
- Side-effectful diagnostics have implicit ordering constraints
  (if check A modifies state, check B has to run after the modification
  settles). Keeping doctor pure means the checks are independent.
- Machine output via `--json` is much easier to reason about when
  running the command never changes disk state.

**Why cap `--fix` actions at <1 second:**
- A surprise 50-second `scry init` triggered from a doctor run would
  feel broken — users run `doctor` to understand state, not to begin
  a lengthy background operation.
- Short fixes are also the ones we can safely re-run on retry without
  cleanup.
- Anything slow should be an explicit command (`scry init`, `npm i -g
  @sourcegraph/scip-typescript`) the user runs deliberately.

Current fixer registry (in `internal/doctor/fix.go`):
- `env.scry_home` — mkdir the directory.
- `daemon.state` — remove stale `scryd.sock` / `scryd.pid`.
- `claude.mcp` — delegate to `setup.Install` with `Force=true`
  (re-registers the MCP server).
- `claude.skill` — same delegation (rewrites the embedded SKILL.md).
- `claude.global_md` — write the 15-line routing rule to
  `~/.claude/CLAUDE.md` **only if the file doesn't already exist**.
  If it exists but doesn't mention scry, the fixer reports Skip with
  "edit manually" — we refuse to touch existing content the user
  already wrote.

After applying fixes, `doctor` re-runs the full check sequence so the
post-fix Report reflects the new state. Fixes are printed below the
refreshed checklist, not as a diff — the before/after is already
visible in "was Warn, now Pass".

**What would change our minds:** a check class appears where the fix
genuinely takes several seconds but is low-friction to automate (e.g.
downloading a pre-built language indexer from an auto-download
recipe). At that point `--fix` gets an `--allow-slow` escape hatch,
not an unconditional "do everything" mode.

---

## 2026-04-11 — Distribution: GoReleaser + install.sh, no Homebrew/Docker yet

**Decision:** scry ships via GoReleaser to GitHub Releases. The
pipeline produces 4 archives (darwin/linux × amd64/arm64) plus a
checksums file on every `v*` tag push. A POSIX shell install script
at `scripts/install.sh` detects the user's platform, queries the
GitHub API for the latest published release, downloads + verifies
SHA256, and drops the binary at `~/.local/bin/scry` (overridable via
`INSTALL_DIR`). End users paste one `curl -fsSL ... | sh` line and
have a working scry.

No Homebrew formula, no Docker image, no npm wrapper, no Debian/RPM
packages. The install story for v1 is "one curl line or go install."

**Why:**
- Three of the four distribution channels (homebrew, docker, .deb)
  require maintaining publication infrastructure and version metadata
  in *addition* to the GitHub release itself. Each one is a place the
  version can drift from the canonical tag.
- scry's target user is developers who already have curl and a
  shell; they don't need brew-level abstractions to install a single
  static binary.
- The one-liner path is the highest-leverage adoption mechanism and
  was the actual blocker for non-Go users pre-v0.1.0.

The GoReleaser config (`.goreleaser.yaml`) uses `CGO_ENABLED=0` and
`-trimpath` for reproducible static builds, injects `main.Version` via
ldflags from the tag, and defaults to `draft: true` so a human eyeballs
the changelog before publishing. The workflow (`.github/workflows/
release.yml`) reads the Go version from `go.mod` so there's no
hardcoded drift.

**Why draft releases by default:**
- First release of anything is where changelog quality matters most.
  A human eyeballing the auto-generated list catches typos, wrong
  commit titles, and missed user-facing notes before users see them.
- Publishing is a 1-click operation (`gh release edit vX.Y.Z
  --draft=false` or the web UI button). Unpublishing after-the-fact
  is awkward.

**Why `scry upgrade` instead of letting users curl | sh again:**
- Upgrade-in-place preserves any shell aliases, PATH customizations,
  and install paths the user chose originally.
- `scry upgrade` can print "you're up to date" and short-circuit
  without re-downloading, which `curl | sh` can't.
- The rename-dance replace handles concurrent `scry` invocations
  gracefully via Unix inode semantics; a fresh `curl | sh` run would
  race with a running daemon.

**What would change our minds:**
- A real user asks for brew / docker / similar because their install
  workflow is centralized. At that point add exactly the channel they
  need, not all of them.
- GoReleaser's output format changes in a breaking way (unlikely —
  they're stable enough that the current config will outlast scry's
  next several major versions).

The full operational checklist — pre-flight, tag, watch workflow,
publish draft, smoke-test — lives in `docs/RELEASING.md`.

---

## 2026-04-11 — Compound symbols (`DB::table`) parsed in the MCP layer, not scry's name index

**Decision:** scry's BadgerDB name index matches by display name
(case-insensitive exact match on the short name, e.g. `table` or `DB`).
When an agent asks `scry_refs("DB::table")`, the MCP server at
`internal/mcp/server.go` splits the compound on `::` / `->` / `.`,
queries the tail (`table`), and filters results whose `symbol_id`
contains the leftmost token (`DB`) as a descriptor segment boundary.
Empty filter results return an honest empty set, not a fall-back to
the class-level query.

**Why parse compounds in MCP and not in the daemon:**
- The CLI already has a precise surface: `scry refs table`. Users who
  know exactly what they want can get it directly.
- Agents (via MCP) naturally phrase queries in method-call notation
  because that's how humans talk about code ("where is DB::table
  called?"). The parsing belongs at the agent-facing surface.
- Keeping the daemon contract strict means future tools that dial
  scry's RPC directly get predictable, exact-match behavior.
- Container filtering at the MCP layer is cheap — it's a substring
  check on already-returned symbol IDs, not a new query.

**Why empty filter results return empty, not fall back:**
- An early iteration fell back to querying the container name
  (`DB::nonexistent` → query `DB`, return 252 DB class references).
  That's a false positive: the agent asked for a nonexistent method
  and got a pile of unrelated class refs it has to sift through.
- Empty results with the original symbol name preserved are a
  truthful "I looked this up and found nothing" answer. Agents can
  decide whether to ask differently or fall back to Grep.

**Container matching:** the filter looks for the container as a
descriptor segment boundary (`/DB#`, `/DB/`, ` DB#`, or ` DB/`),
not just a substring. That's what keeps `DB::table` narrow to
`Illuminate/Support/Facades/DB#table()` and excludes
`Illuminate/Database/Eloquent/Builder#table()` where `DB` is just a
substring of `Database`.

**What would change our minds:** agents start asking for nested
container notation like `Illuminate::Support::DB::table` or
`Illuminate.Support.DB.table` that our `lastIndex` split can't handle.
At that point the parser grows from "find the rightmost operator" to
"tokenize + walk left-to-right," and the container filter becomes a
full-path check instead of a segment-boundary check.

---

## 2026-04-10 — PHP P2: ship scip-php as an embedded directory tree, not a PHAR

**Decision:** scry vendors `davidrjenni/scip-php` (pinned to commit
`97a2d8d`, with one local patch — see below) as a pruned tarball checked
into `internal/sources/php/scip-php.tar.gz` and embedded into the scry
binary via `go:embed`. On first PHP indexing the tarball is extracted
into `~/.scry/bin/scip-php-<sha>/` and we run `php
scip-php-<sha>/bin/scip-php` from within the target repo. The user only
needs `php` (8.3+) on PATH.

The local patch in `src/Composer/Composer.php` re-prepends scip-php's
bundled `nikic/php-parser` to the SPL autoloader after the target
project's autoloader is registered, so scip-php's parser version always
wins. Without the patch, every Laravel project pinning a different
`nikic/php-parser` version causes scip-php to crash with `Int_::KIND_INT
undefined` (or similar) at parse time.

**Why not a PHAR:** the calibration doc recommended a PHAR built via
`humbug/box`, but day-2 implementation found two showstoppers:

1. The PHAR autoloader collision is identical to the directory-tree
   collision — scip-php's `Composer.php` deliberately loads the target
   project's `vendor/autoload.php` to resolve user classes, so its own
   `nikic/php-parser` gets clobbered regardless of whether scip-php is
   delivered as a PHAR or a directory.
2. The standard fix (php-scoper namespace prefixing via box's
   compactor) blew up on PHP 8.4: phpstorm-stubs lists `exit`, `die`,
   `clone`, etc. as functions, so `expose-global-functions => true`
   generates `function exit() { ... }` shims that are syntactically
   invalid because the names are reserved. We tried `exclude-functions`
   regexes; they didn't suppress the shims because the autoload
   generator's `recordedFunctions` is populated through a different
   path. After ~30 minutes spinning on scoper, the directory-tree +
   patch approach was clearly simpler.

The downside of the directory tree: ~14 MB extracted on disk per scry
release vs ~1 MB for a PHAR. Compressed in the embedded tarball it's
2.1 MB, which is fine.

**Why we patch upstream:** the `Composer.php` change is small (~10
lines), trivially re-applied on a `scip-php` rebase, and avoids forking
scip-php in any meaningful sense. We keep the patched tree alongside
the embedded tarball generation script (TODO: write the script).

**What would change our minds:**
- scip-php upstream merges an `--isolated-autoload` mode that registers
  its own deps first.
- A maintained `scip-php` PHAR appears that doesn't collide.
- We add another PHP-aware indexer (e.g. Phpactor) that has cleaner
  isolation properties.

---

## 2026-04-10 — Synthesize SymbolRecords for occurrence-only symbols

**Decision:** When the SCIP parser walks a document's occurrences, if it
encounters a symbol id that has no corresponding `SymbolInformation`
entry in any indexed document, synthesize a `SymbolRecord` with display
name derived from the symbol id's last descriptor and `Kind: "External"`.

**Why:** scip-php (and to a lesser extent scip-go) only emit
`SymbolInformation` for symbols *defined* inside the indexed source
tree. References to vendor classes — every Illuminate facade, every
Eloquent model contract, every PHP stdlib type — appear as occurrences
but produce no symbol record. The result was that `scry refs DB`
returned zero on hoopless_crm even though the codebase has 252
`DB::*` call sites, because the name index never knew the symbol
existed.

The fix is one if-statement in the occurrence loop. On hoopless_crm
the symbol count rose from 20953 → 22190 (1237 external symbols
synthesized) and zero queries that previously worked broke.

**Why this is a SCIP-parser-level fix and not a per-language hack:**
the same gap exists for any indexer that's lazy about emitting
SymbolInformation. scip-go has the same shape for stdlib refs. Future
indexers (Python, Bash) almost certainly will too. Synthesizing in the
parser keeps each indexer wrapper trivial.

**What would change our minds:** an indexer starts emitting full
SymbolInformation for external refs, and the synthesized records
duplicate fields the indexer would otherwise populate (Documentation,
Kind, etc.). At that point we'd switch to "synthesize only if not
already seen", which is what the current code does anyway via the
`seenSymbols` set.

---

## 2026-04-10 — PHP P2: view + config string-ref walker

**Decision:** A second walker pass walks every `.php` file in the project
(skipping `vendor/`, `node_modules/`, `storage/`, `public/`,
`bootstrap/cache/`, and dot-prefixed dirs), runs the existing scanner
over each file, and pulls out any `view('key')` and `config('key')`
calls whose first argument is a string literal. For each match, the
walker synthesizes a SymbolRecord and a ref occurrence, joining the
call site to a stable per-key symbol id.

Symbol shapes:

| Call | Descriptor | Display name |
|---|---|---|
| `view('users.show')` | `resources/views/users/show.blade.php#` | `users.show` |
| `config('mail.from.address')` | `config/mail.php#from.address` | `mail.from.address` |

Real-world numbers on hoopless_crm:

| Metric | Value |
|---|---|
| Files scanned | 1589 |
| `view()` refs | 7 (matches calibration) |
| `config()` refs | 280 (close to calibration's 300) |
| `scry refs pdf.matrix-compare` | 1 (the controller call site) |
| `scry refs services.dataforseo.login` | 6 across services and controllers |

**Why one walker for both:** view and config are the same shape (named
function call with string literal first arg). Doing them in separate
walker passes would walk every file twice. The scanner extension
returns all string-arg call sites in one pass; the walker filters by
recognized function name.

**Why we don't try to verify the file exists on disk:** Laravel's
runtime resolver looks up views/configs through a registered loader,
not by direct path. The walker emits a synthetic symbol whose
descriptor encodes the conventional path, but doesn't check the
filesystem. False positives (string keys that look like view/config
keys but are something else) are bounded by the spec list of
recognized function names.

**Why config splits on the FIRST dot only:** Laravel's `config()`
helper reads `config/<head>.php` for the head segment and treats
the rest as a nested array path inside that file. Splitting on the
first dot mirrors that runtime behavior, giving us a per-file
descriptor (`config/services.php#dataforseo.login`) that can later
join to a config-file walker if we add one.

**Bug fixed during shipping:** the scanner had an infinite loop on
files containing UTF-8 characters past byte 127 inside an
interpolated double-quoted string. The dispatch was
`case isIdentStart(rune(c)):` which widens a `byte` to a `rune` in
the Latin-1 range — `\xE2` → `â` → `IsLetter` returns true. The
identifier reader then called `utf8.DecodeRune` which returned
`RuneError` for the multibyte sequence, produced an empty
identifier, and returned without advancing s.pos. The main loop
would then dispatch on the same byte forever. Fix: a new
`isIdentStartByte` helper that decodes the UTF-8 sequence properly
before deciding, plus a defensive force-advance in the main loop
if the identifier scanner returns without consuming any bytes.
A regression test in `scanner_test.go` covers both the
interpolated-arrow case and a truncated UTF-8 sequence.

**What would change our minds:**
- A real codebase has many false-positive ref hits because some
  user function happens to be named `view` or `config` but takes a
  string literal that isn't a view/config key. At that point we'd
  add receiver-aware matching (only match `view()` at the global
  scope, only match `Config::get()` on the facade). The scanner is
  receiver-blind today.
- The view ref count stays low (7 in hoopless_crm) and the cost of
  walking 1589 files just for view extraction outpaces the value.
  Most Laravel apps with non-API surfaces should have many more.

---

## 2026-04-10 — PHP P2: facade -> backing-class resolver via static map

**Decision:** Hardcode a Go-side map of ~30 Illuminate framework facades
to their backing manager and contract classes (`Auth ->
{AuthManager, Factory, Guard, StatefulGuard}`, `DB -> {DatabaseManager,
Connection}`, etc.). After the non-PSR-4 walker runs, the resolver
walks every `SymbolRecord`, identifies facade method symbols
(`Illuminate/Support/Facades/<X>#method()`), looks up the matching
backing-class methods in the same store, and emits synthetic ref
occurrences from each facade ref site to every backing candidate.

If the backing method does not exist in the store (because nothing in
the user code references it directly), we synthesize a `SymbolRecord`
for it on the fly using the same package + version as the facade —
this keeps `scry refs AuthManager::user` working even when scip-php
never indexed `AuthManager`.

Real-world numbers on hoopless_crm:

| Metric | Value |
|---|---|
| Facade methods scanned | 89 |
| Synthetic backing edges emitted | 5129 |
| `scry refs user` (filtered to AuthManager) | 75 (was 0) |
| `scry refs user` (filtered to Guard contract) | 150 (was 0) |
| `scry refs table` (filtered to DatabaseManager) | 92 (was 0) |
| `scry refs table` (filtered to Connection) | 92 (was 0) |

**Why a static map and not dynamic resolution from
`getFacadeAccessor()`:** the calibration explicitly recommended
"cover the top ~30 facades and call it done." Dynamic resolution would
require parsing every framework facade's source, walking the service
container map, and handling the cases where `getFacadeAccessor()`
returns dynamically — many days of work for marginal gain on the
top 30. The map covers Auth, Cache, Config, Cookie, Crypt, DB, Date,
Event, File, Gate, Hash, Http, Lang, Log, Mail, Notification, Password,
Queue, Redirect, Redis, Request, Response, Route, Schema, Session,
Storage, URL, Validator, View, Bus, Broadcast, Artisan — every facade
shipped with vanilla Laravel.

**Edge multiplication is fine.** Each facade method ref produces N
edges, one per backing candidate. `Auth::user()` therefore creates 4
records (AuthManager, Factory, Guard, StatefulGuard). This is
intentional: an agent might query any of those four names and should
get the call sites either way. Storage cost is trivial (5k entries on
a 22k-symbol store).

**What would change our minds:**
- A real codebase appears that uses a custom facade scry's map
  doesn't cover, AND missing it causes a noticeable agent failure.
  At that point we add a project-level facade resolver that parses
  the user's `AppServiceProvider::register()` for `bind`/`singleton`
  calls.
- The duplication causes false-positive churn in some downstream
  query type (e.g. `scry callers <method>` returning N copies of the
  same site). At that point we deduplicate at query time, not by
  collapsing the resolver.

---

## 2026-04-10 — PHP P2: walk Laravel non-PSR-4 dirs and bind refs to scip-php symbols

**Decision:** After scip-php finishes indexing a PHP repo, scry walks
`routes/`, `database/migrations/`, `config/`, and `bootstrap/` with a
small Go-side PHP scanner (no real parser, just a token-aware walker
that handles strings/comments/heredocs). For each `::class` reference
it finds, it resolves the name against the file's `use` statements,
constructs the corresponding SCIP descriptor (`App/Http/Controllers/
UserController#`), looks up the matching SymbolRecord by the leaf name,
and emits a synthetic ref occurrence joined to scip-php's existing
symbol id. If no matching symbol exists in the store, the walker
synthesizes one tagged with the project's composer package name + lock
content-hash so the ref is still queryable.

Real-world numbers on `~/herd/hoopless_crm` (Laravel 12, ~1199 PHP
files in `app/`):

| Metric | Value |
|---|---|
| Files scanned | 390 |
| `::class` refs found | 1283 |
| Refs bound to existing scip-php symbols | 1254 (98%) |
| Refs synthesized (class not in store) | 29 |
| `scry refs UserSettingsController` before walker | 0 occurrences |
| `scry refs UserSettingsController` after walker | route handler bindings from `routes/settings.php` |

**Why a Go-side scanner instead of running scip-php a second time
with non-PSR-4 paths:** scip-php resolves classes via Composer's
PSR-4 map, not by walking directories. There's no flag to "also index
this loose .php file." The walker is post-processor by design and we
only need `use` statements + `::class` literals — a real PHP parser
would buy us nothing for that target. The Go scanner is ~350 lines
plus a 100-line walker, with unit tests covering string/comment
escape, group use, and absolute names.

**Why not extract more (facades, view, config refs):** SPEC §11.1 and
the calibration doc list four post-processor items; this decision lands
the first one (the file walker, which had the highest measured
leverage — 1168 routes/web.php refs alone in the calibration). The
other three (facade resolver, view template, config key) ride on the
same scaffolding and land in subsequent commits.

**What would change our minds:**
- scip-php upstream learns to index non-PSR-4 files. (Unlikely; the
  whole point of scip-php is that it follows the autoload graph.)
- A class of false matches appears that the simple scanner can't
  distinguish from real refs (e.g. `Foo::class` inside a PHP attribute
  in a way that breaks the index). At that point we upgrade to nikic's
  Go-side `php-tokenizer` port or accept the noise.

---

## 2026-04-10 — Skip SCIP local symbols entirely

**Decision:** When parsing a `.scip` file, drop every `SymbolInformation`
and `Occurrence` whose symbol id starts with `local ` (the SCIP local-symbol
prefix). Don't write them to BadgerDB at all.

**Why:** SCIP local symbols are document-scoped — `local 19` in document A
and `local 19` in document B represent two different variables. The first
P1 build stored them under a global keyspace, which caused `scry refs
concurrency` against trawl to return 83 results from completely unrelated
local variables across the codebase. The bug was only noticed because the
returned occurrences were obviously wrong. Filtering locals entirely is
safer than namespacing them by document because agents almost never ask
"find every use of a local variable named `i`" — local variable
introspection is what an LSP is for.

The size effect is significant: trawl's symbol count dropped from
2487 → 725 (~70% reduction). Most of that mass is method parameters and
function-local declarations.

**What would change our minds:** an agent surface emerges that legitimately
needs cross-occurrence queries on locals (e.g. "highlight every use of
this loop variable in this function for an inline rewrite"). At that point
we'd namespace locals as `<doc>::local <N>` and re-enable, but only inside
a per-document query mode — they should never appear in global lookups.

---

## 2026-04-10 — Defer in-memory cache, BadgerDB is fast enough

**Decision:** Reverse the earlier "all-in-memory until manifest tells us
otherwise" call from the §15 cache-strategy decision. P1 reads BadgerDB
directly per query through the registry. No `map[string]Symbol` overlay,
no LRU, no preload. The store registry only caches the open BadgerDB
*handle*.

**Why:** Measurement after P1 landed shows the daemon serves `scry refs
handle` against advocates (3791 symbols, 26166 references) at 6-7ms
wall-clock end-to-end including process startup, RPC, and JSON marshal.
Single-microsecond BadgerDB lookups dominate the per-query work, not
deserialization. Building an in-memory mirror would add complexity (cache
invalidation on reindex, RAM cap, atomic swap) for no measurable win.

The §15 decision was made before measurement; this entry overrides it.

**What would change our minds:** a query type that requires walking
thousands of records per call (e.g. full call graph traversal at depth 10
across a 1M-LOC monorepo) where BadgerDB iterator overhead becomes the
bottleneck. We'd add the cache for *that query path specifically*, not
globally.

---

## 2026-04-10 — Background full reindex on file change, accept the latency gap

**Decision:** When a file changes in a watched repo, the daemon runs the
*full* SCIP indexer over the *whole* repo on a background goroutine, then
atomically swaps the new BadgerDB store into the registry when it's done.
No single-file incremental, no tree-sitter overlay, no partial updates.

**Why:** The spec target was <200ms for incremental updates. That's
unreachable with the current SCIP indexers — `scip-typescript` and
`scip-go` are project-wide, type-resolution-driven, and offer no
`--single-file` mode. Forcing a single-file path would either be wrong
(partial type resolution) or require us to build a whole new indexer.

Realistic numbers: ~600ms for a tiny project, ~3s for `trawl`, ~10-15s
for `~/herd/advocates`. Documented in `internal/daemon/watch.go`.

The right long-term answer is a tree-sitter overlay that handles 95% of
queries (syntactic precision is enough for "find this name", "find this
class definition") and falls back to the SCIP store for the few queries
that need full type resolution. That's a P3+ effort.

**What would change our minds:** a SCIP indexer publishes a usable
single-file mode, OR a tree-sitter overlay proves cheap enough to ship.

---

## 2026-04-10 — Reindex via build-into-temp-dir + atomic swap (overrides earlier defer)

**Decision:** The watcher's reindex path uses `index.BuildIntoTemp` to
write the new BadgerDB into `<storage>/index.db.next/` while the live
store at `<storage>/index.db/` keeps serving queries. After the build
finishes, `Registry.SwapNext` performs a tiny critical section: close
live store → archive live dir → rename next → live → open new store
→ replace registry entry. The trash dir is removed in the background.

This overrides the earlier "defer the fix" decision. The deferred
hypothesis ("the window is rare in practice") survived right up until
PHP P2 landed and reindexes started routinely taking 45-50s on real
Laravel apps. At that point any save during an ongoing reindex would
guarantee a several-second blackout — cheap to fix, expensive to leave
broken.

Measured on hoopless_crm (1409 docs / 22k symbols / 64k refs):

| Metric | Pre-fix | Post-fix |
|---|---|---|
| Total reindex wall-clock | ~48s | ~48s |
| Query unavailability window | full reindex (~48s) | 12ms (single swap) |
| Queries served during a 75s reindex test | 0 | 1449 |
| Slowest single query during swap | ∞ (errors) | 84ms |

**Why a registry-level swap helper instead of a one-shot rename in the
watcher:** the registry holds the live store handle and the BadgerDB
directory lock. Only the registry can sequence "close live → rename →
open new" inside its mutex without exposing a moment where the
registry has a stale entry pointing at a renamed directory. Putting
the swap inside `Registry.SwapNext` keeps every visible registry
state coherent.

**Why we still archive instead of immediate-delete the old dir:** if
the rename of next → live fails partway through, we want to roll back
to the original state. The archive lets us `os.Rename(trash, live)`
to recover. Background cleanup of the trash dir is best-effort.

**What would change our minds:** if the swap becomes long enough to
matter (e.g. cross-filesystem renames force a copy), we'd need a
stronger atomicity story — maybe a per-repo serial and an in-memory
overlay. None of that is worth doing today.

---

## 2026-04-10 — Bump RLIMIT_NOFILE on daemon startup

**Decision:** The daemon raises its NOFILE soft limit to the hard limit
on startup (`internal/daemon/rlimit.go`). On macOS the soft default is
256, the hard limit is much larger; we just need to opt in.

**Why:** Found via crash. fsnotify uses one fd per watched directory,
and `~/herd/advocates` has ~1500 directories. The first P1 daemon panicked
with `fatal error: pipe failed` because it ran out of fds during the
recursive `WalkDir` add, then `signal.Notify` couldn't open its self-pipe.

The bump is the right call regardless of the watcher behavior — anything
the daemon does at scale (multiple concurrent connections, multiple
indexed repos) is fd-bound. macOS' default is just too low.

**What would change our minds:** nothing reasonable. This is a strict
improvement.

---

## 2026-04-10 — Watcher: aggressive skip list + 2048-dir hard cap

**Decision:** The fsnotify watcher skips an exact-name list (`node_modules`,
`vendor`, `storage`, `public`, `cache`, `tmp`, `dist`, `build`, `coverage`,
`__pycache__`, `venv`, etc.) PLUS every directory name beginning with `.`
(hidden infrastructure: `.git`, `.next`, `.turbo`, `.idea`, `.gradle`,
`.pnpm-store`, etc.), AND caps the total at 2048 directories per repo.
When the cap is hit the watcher logs a warning and continues without
incremental updates for the unwatched portion.

**Why:** Even with NOFILE bumped, watching every directory in a Laravel
or Rails-class repo is wasteful — most subtrees are runtime data
(`storage/wordpress`, `storage/oldpdfs`) that never contain source code.
Skipping them saves fds, reduces fsnotify event volume, and makes the
relevant-event filter faster. The 2048 cap is a defense-in-depth: any
single repo that blows past it is almost certainly indexing something
generated.

**What would change our minds:** a real source tree (not a runtime tree)
needs more than 2048 watched directories. At that point we add a
configurable cap in the daemon config and document it.

---

## 2026-04-10 — Signal handling before watcher bootstrap

**Decision:** `daemon.Run` calls `signal.Notify` *before* calling
`bootstrapWatchers`. The first P1 build did the opposite, which caused
a cascading panic when fd exhaustion in the watcher prevented
`signal.Notify` from opening its self-pipe.

**Why:** Defense-in-depth. Signal handling is process-wide and should be
set up before any code path that could fail. The cost of moving it
earlier is zero; the cost of *not* moving it is a cryptic
"`fatal error: pipe failed`" panic instead of a clean error.

**What would change our minds:** nothing. This is a strict improvement.

---

## 2026-04-10 — Auto-download scip-go yes, scip-typescript no

**Decision:** P1 implements `internal/install` for `scip-go` (pinned to
`v0.1.26`, SHA256-verified, downloaded into `~/.scry/bin/`). It does
*not* implement auto-download for `scip-typescript`. Users still install
that one manually with `npm i -g @sourcegraph/scip-typescript`.

**Why:** scip-go publishes per-platform tarballs as GitHub release assets
with a checksums file. The download flow is straightforward and matches
the §15 "auto-download pinned versions on first use" decision exactly.

scip-typescript is an npm package. Its GitHub release page has *no
binary assets* — only source tarballs. Auto-installing would mean
either:
1. Bundling a node + npm install at first use (too invasive for an agent
   tool)
2. Shelling out to `npx --yes @sourcegraph/scip-typescript@<pinned>`
   (delegates the install to npm but requires the user to have node)
3. Vendoring a pre-built JS bundle inside the scry release (huge,
   couples our release to scip-typescript's)

None of these are a clear win over "user runs `npm i -g` once". The
install instruction is in the README and the indexer wrapper returns a
clear error pointing the user at it.

**What would change our minds:** scip-typescript starts shipping binary
release assets, OR a maintained pre-built single-file bundle appears, OR
we end up bundling node anyway for the gstack `/scry` skill wrapper.

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

---

## 2026-07-28 — Repo resolution: canonical paths + walk-up to nearest indexed ancestor

**Decision:** `Registry.Get` no longer requires the query path to be the
exact string the repo was indexed under. Resolution order: literal
absolute path (back-compat), canonical form (symlinks resolved via
`EvalSymlinks`, each component rewritten to its on-disk casing), then
each canonical ancestor from nearest to farthest. Resolved paths are
cached as alias keys pointing at the same `*Entry`; `Evict`/`SwapNext`
purge aliases by entry identity and `Snapshot` dedupes them.
`handleInit` canonicalizes before building so new indexes always key on
the canonical path.

**Why:** Three real failure modes on the scribe monorepo:
(1) `~/workspace/scribe/apps/childscribe-laravel` is a symlink to
`~/Herd/childscribe` — the indexed repo was unreachable through the
symlink; (2) the symlink target is written lowercase while the index
key was `ChildScribe` — macOS is case-insensitive so both name the same
directory but hash to different `~/.scry/repos/<sha>` layouts; (3) any
query from a subdirectory of an indexed repo failed with "not indexed
yet". The daemon-side fix covers CLI, MCP, and watcher paths at once.

**What would change our minds:** nested indexed repos where walk-up
picks a surprising ancestor (nearest wins today — if a user wants a
subdirectory served by an outer repo's index while the subdirectory has
its own, that's already ambiguous and needs an explicit `--repo`).

---

## 2026-08-09 — Two-tier language detection

**Decision:** A detected language is *primary* if it has a root-level marker
file (`composer.json`, `go.mod`, `package.json`/`tsconfig.json`,
`pyproject.toml`/`requirements.txt`/`setup.py`/`Pipfile`) or holds ≥10% of
source files. Otherwise, above a 1% floor, it is *incidental*: its indexer is
not invoked and its absence never degrades repo status.

**Why:** The previous flat 1% threshold invoked a full indexer for any
language clearing 1% of files. Measured on `childscribe-beta-r4`: 855 PHP,
110 TS/JS, 37 Python files and no Python marker. Python cleared the bar at
3.7%, `scip-python` was not installed, and the entire repo was reported
`partial` — 855 PHP files' worth of complete index described as degraded
because of 37 incidental scripts. 19 of 44 indexed repos were in this state.

The marker file carries the weight rather than share alone because it is a
statement of intent: a repo with a real component in a language declares it.
Share is the fallback for undeclared-but-substantial code.

**What would change our minds:** A repo with a genuine, sizable component in
a language that declares no marker and sits under 10% — it would be silently
skipped. If that shows up, add the marker filename rather than lowering the
share, or introduce a per-repo override in the manifest.

---

## 2026-08-13 — A build degrades per language; only an undescribable store aborts

**Decision:** `index.buildAtLayout` writes a manifest on every build that
reaches the ingest stage, including one where zero languages produced usable
output. A language whose indexer is missing, whose indexer fails, or whose
`.scip` dump won't parse costs that one language: its `IndexerResult` records
the status, error and remedy, every other language still ingests, and the
manifest lands with status `partial`. Only an outcome that leaves us unable
to describe what the store contains still aborts with no manifest — storage
dir uncreatable, store un-openable, `SchemaVersionOnDisk`/`Reset`/`SetMeta`
failure, manifest unwritable.

`IndexerResult` also carries the per-language ingest counts
(`document_count`, `symbol_count`, `definition_count`, `reference_count`)
taken from that language's own `scip.Stats` before aggregation, so a
consumer can tell "indexed and found nothing" from "never ran".

**Why:** Two failure modes collapsed a whole build. `scip.Parse` returning an
error for one language returned an error for the entire build — discarding
the languages that had already ingested and leaving the previous manifest in
place to describe a store that no longer matched it. And a build where every
indexer was missing returned `rpc error -32603` with no artifact at all, so
the operator got an error code instead of the list of things to install. The
per-language remedy strings already existed; they just never reached disk in
the case where they mattered most.

**Consequence worth knowing:** the manifest now always describes what the
store actually holds, which means a build that ingested nothing leaves an
empty store rather than the previous build's data. `Registry.SwapNext` will
promote that empty index. This is the honest reading — the alternative is a
manifest that describes a store it isn't paired with, which is the bug above
in a different costume — but it does mean a total indexer outage during a
watcher reindex empties an index that was previously good. The manifest says
so, in per-language detail.

**Every failure carries a remedy, and three different ones.** A language can
reach zero output three ways, and they need different advice: the binary is
absent (`indexerRemedies` — the install command), the binary ran and exited
non-zero (`indexerFailureRemedy` — run it directly, check the toolchain
version), or the binary succeeded and its dump would not parse
(`parseFailureRemedy` — `scry init --force`, report it if it recurs). The
last two deliberately do *not* reuse the install command: the tool is already
installed, so "npm i -g" is wrong advice, and wrong advice costs more than
none. A failure recorded without any remedy leaves the operator exactly where
the bare `-32603` did, which is the thing this change exists to stop, so
`classify` now returns a remedy for every non-ok status.

**Testing:** the indexer invocation is injected into `buildAtLayout` as an
`indexerRunner`, and the tests synthesize SCIP protobuf dumps directly
(valid ones for the success path, garbage for the parse-failure path). No
test needs scip-typescript, scip-go, scip-python, php or npm on PATH. The
zero-output test is table-driven over all three failure modes above, asserting
each language carries the remedy matching *how* it failed rather than merely
a non-empty one.

**What would change our minds:** if the empty-store-on-total-failure case
bites in practice, the fix belongs in the caller that swaps — `BuildIntoTemp`
plus `SwapNext` can decline to promote a store that ingested nothing — not in
`buildAtLayout`, which should keep the manifest and the store consistent.

---

## 2026-08-13 — Stale and empty are derived at report time, never persisted

**Decision:** `Manifest.Status` on disk keeps exactly two values, `ready` and
`partial` — they describe what the build produced. The two new signals are
computed when status is reported, from the manifest plus the live repo:

- **stale**: the manifest records `head_commit` (the repo's HEAD when the
  build started). A repo is stale when the live HEAD differs. When either
  side has no commit — not a git checkout, no commits yet, or a manifest
  written before `head_commit` existed — the comparison degrades to "is the
  newest source file's mtime after `IndexedAt`".
- **empty**: a primary language whose indexer reported `ok` but produced zero
  symbols across a non-zero detected file count. Read from that language's own
  `IndexerResult` counts, which the builder populates only on the success path
  — so a language that never ran can't be mistaken for one that ran and found
  nothing, which the aggregate `Manifest.Stats` cannot distinguish. Each of
  the three conditions rules out a legitimate zero: incidental languages
  aren't indexed deeply, missing/failed ones are already reported as
  `partial`, and zero symbols from zero files is the correct answer rather
  than a failure.

`scry status` and `scry doctor` fold the manifest status and both signals
into one display label with precedence `partial > empty > stale > ready`
(`index.EffectiveStatus`).

**Why:** A signal that had to be written into the manifest would need a
reindex to be discovered — exactly the thing that is broken. The 44 repos
already on disk with months-old indexes must become diagnosable by *reading*
them, not by rebuilding them first. Deriving also keeps the manifest a record
of one build rather than a mutable status cache that two writers (the builder
and the reporter) both own.

HEAD comparison rather than timestamps because it is exact: it survives clock
skew, a rebuild that ran long, and a checkout that rewinds to an older
commit — all of which a `IndexedAt < newest mtime` test gets wrong. The mtime
walk stays as the fallback for non-git trees and is only paid for when there
is no commit to compare.

Precedence puts `partial` first because a missing indexer is a louder fact
than either derived signal, and `empty` above `stale` because an empty
language is broken at the current commit while a stale one is fixed by
exactly the reindex the label already implies.

**Cost:** one `git rev-parse HEAD` per repo per status call, cached for the
call and bounded by a 2s timeout. `scry status` is on the agent hot path; a
wedged git invocation degrades the staleness signal, not the response.

**"No HEAD" and "couldn't ask" are different answers.** Both come back as an
empty commit string, and only the first one licenses the mtime fallback. If
the timeout fires and we fall back anyway, the safety valve does the opposite
of its job twice over: it pays the most expensive path we have (a full source
tree walk, per repo) to produce a *false* `stale` on every repo touched since
its build — including repos sitting exactly at their indexed commit. On a
machine with many repos, the tail of the list would report stale purely
because git was slow.

So `gitindex.HeadCommit` reports the two cases distinctly and
`gitindex.HeadUnknown(err)` is the single place that tells them apart. It is
true only for a context error. Note that cancellation *kills* git, and a
killed process surfaces as an `*exec.ExitError` — the same error class as
"not a repository" — so `HeadCommit` must check the context before the exit
code or the distinction is lost precisely when it matters. A missing git
binary is deliberately NOT unknown: that is a permanent absence of HEAD, and
mtimes are the correct and only remaining signal.

When the answer is inconclusive, both `scry status` and `scry doctor` report
not-stale and skip the walk. Silence is the honest answer to a question we
never got to ask; an invented finding is not.

**What would change our minds:** if per-repo HEAD resolution ever shows up in
status latency (many repos, cold FS cache), cache the result in the daemon
across calls keyed by `.git/HEAD` mtime rather than re-running git each time.

## 2026-08-13 — Both derived signals must work on the indexes already on disk

**Decision:** the mtime fallback fires whenever there is no *pair* of commits
to compare — not merely when the repo has no live HEAD — and `EmptyLanguages`
suppresses itself on manifests written before the per-language count fields
existed.

**Context:** both signals shipped correct against manifests the new builder
writes, and wrong against every manifest already on disk. On this machine that
is all 45 of them, and the task's opening premise is precisely "a repo indexed
months ago reports ready today". A signal that only starts working after a
reindex fails the requirement that neither may require a reindex to compute.

Two independent bugs, found by running `scry doctor` against a real indexed
repo rather than only against synthesized manifests:

1. **Stale never fired on a legacy manifest.** `IsStale` handles a manifest
   with no `head_commit` correctly — it falls back to mtimes. But both callers
   decided whether to *compute* the mtime by asking whether the REPO had a
   HEAD, not whether the comparison had two commits. A pre-`head_commit`
   manifest in an ordinary git checkout resolves a live HEAD, so the walk was
   skipped, a zero time was passed in, and the repo reported not-stale
   permanently. The condition is now `m.HeadCommit == "" || (head == "" &&
   conclusive)`. Note the asymmetry: when the manifest has no commit we walk
   regardless of how git answered, because git's answer cannot decide anything
   without a recorded commit to compare it against — the `conclusive` guard
   only protects the case where a commit comparison was actually possible.

2. **Empty fired on every legacy manifest.** Those manifests carry per-language
   results with `file_count` but no `symbol_count`, so the predicate
   `SymbolCount == 0 && FileCount > 0` matched a repo that had demonstrably
   indexed 3061 symbols. Absent and zero are the same value per-language, but
   they are separable in aggregate: the builder assigns each language's
   `SymbolCount` from the same `scip.Stats` it sums into `Manifest.Stats`, so a
   positive total with no positive per-language count proves the fields were
   never written. `countsUnrecorded` encodes exactly that and returns no
   languages. This is a narrow suppression — a genuinely empty build reports
   zero in the aggregate too, so it is untouched.

The second bug is the more dangerous one, and it is worth naming why: it
replaces a silent green with a false red across every repo at once. A signal
that flags 45 healthy repos gets ignored, including on the day it is correct.
That failure mode is worse than the one this task set out to fix.

**Cost, and why the walk is now bounded:** fixing (1) means every legacy
manifest takes the mtime path, measured at ~2.9s across the 42 live repos on
this machine. `scry status` is on the hot path for agents, so the status
budget now covers walks as well as git calls, and `NewestSourceMTime` stops
when it expires. Truncation is safe in one direction only: seeing fewer files
can lower the maximum, so a bounded walk can go quiet on a stale repo but can
never invent a file newer than the index. The cost is also transitional — it
disappears per repo as each is reindexed and records a commit.

**What would change our minds:** if the schema version is ever bumped for an
unrelated reason, `countsUnrecorded` could become a straight version check
instead of an aggregate inference. The inference is exact today, but it is a
property of the builder's aggregation, so it needs the comment that says so.

## 2026-08-22 — Watcher self-trigger loop: drop Chmod-only events, defer in-flight events behind an mtime gate

**Decision:** three coupled changes in `internal/daemon/watch.go`:

1. `relevantEvent` requires a content op (`Create|Write|Remove|Rename`). A
   Chmod-only event is dropped no matter the extension.
2. The run loop tracks an in-flight reindex. Events arriving mid-build never
   arm the debounce; their paths are recorded (capped at 256) and, when the
   build finishes, exactly one catch-up reindex is armed **iff** one of those
   files still exists with mtime newer than the build start.
3. Reindex scheduling (debounce, cooldown, in-flight) all lives in the run
   goroutine; the old `maybeReindex`'s unconditional `AfterFunc` re-arm — and
   its data race on `lastReindex` — are gone. The build body is a `doReindex`
   field so tests drive the real scheduling with a fake build.

**Context:** the daemon reindexed `idea-planning` 368,959 times over 13 hours,
dirtying ~11 MB/s until the VM compressor exhausted its segment limit and the
machine kernel-panicked (4 times). Reproduced deterministically: one `touch`
produced 46 reindexes in 90 seconds. The self-trigger, captured live with an
instrumented watcher, is *not* a file the pipeline writes: some step of every
reindex runs a git operation that re-hashes dirty working-tree files, git
mmaps any such file ≥ 32 KB (`SMALL_FILE_SIZE`), and on APFS that read fires
a `NOTE_ATTRIB` (atime) kevent, which fsnotify's kqueue backend delivers as
`Chmod`. Any watched repo with one dirty source file over 32 KB looped
forever; `idea-planning` had two.

The mtime gate decides the mid-build-edit corner named in
`docs/REINDEX_LOOP_DIAGNOSIS.md`: a real edit survives (file exists, mtime
advanced → one catch-up), while every observed self-trigger class fails it —
atime bumps don't advance mtime, and an indexer's create+delete temp file no
longer exists. A hypothetical indexer that rewrites a source-extension file
in-repo on every run would still loop; nothing event-driven can distinguish
that from a user edit, and no indexer we ship does it.

**Consequence:** a bare `touch` (attribute-only, no write) no longer triggers
a reindex — verifying the watcher now requires a real content write. Correct
by construction (content is unchanged, the index is not stale) but a behavior
change from the pre-fix watcher.

**What would change our minds:** an editor whose save path is invisible under
the content-op mask (none known — atomic saves emit Create/Rename), or a
first-party indexer that must write source-extension files into the repo.

## 2026-08-22 — Startup sweep of rotate-then-delete garbage

**Decision:** `sweepStaleIndexTrash` runs synchronously in `Daemon.Run`
before watchers start, deleting `index.db.old.*`, `index.db.next`, and
`manifest.json.next` under every `~/.scry/repos/<hash>/`.

**Context:** `Registry.SwapNext` archives the live store and callers delete
the archive in a fire-and-forget goroutine that dies with the daemon. ~8% of
the 369k loop cycles orphaned their archive: 31,542 dirs, 271 GB. The paths
are rotate-then-delete garbage by design — nothing ever reads them — so the
sweep needs no coordination beyond running before the first reindex can
create a fresh `index.db.next`.

## 2026-08-22 — Graph store: single writer via BeginBuild/EndBuild

**Decision:** `GraphRegistry` gains `BeginBuild`/`EndBuild`. BeginBuild
serializes builds (one mutex across repos), closes the registry's handle so
`graph.Build` can take badger's directory lock, and marks the repo so `Get`
returns "graph rebuild in progress — retry shortly" instead of reopening
mid-build. Both build paths (`graph.build` RPC and the post-reindex rebuild)
run inside the pair; `GraphRegistry.Evict` is gone with its callers.

**Context:** every post-reindex graph rebuild failed with "Cannot acquire
directory lock", so graph data silently went stale on every reindex. The
diagnosis doc blamed the `scry mcp` process, but `lsof` showed MCP holds no
graph DB — it proxies through the daemon. The daemon was racing itself:
rebuild goroutines overlapped with each other (the comment claimed a 30s
debounce that never existed) and with `Get` reopening the store between the
old `Evict` and `graph.Build`'s open. Under loop-era churn a rebuild was
always in flight, so the "race" lost deterministically.

**Trade-off:** graph queries during a rebuild fail fast with a retryable
error instead of serving the stale pre-build graph. Serving stale would need
build-into-temp + swap like the code index; worth doing only if the failing
window (seconds, post-reindex only) annoys in practice.

## 2026-08-23 — Memory extraction: ordered model chain in `~/.scry/config.yaml`

**Decision:** `memory.models` in `~/.scry/config.yaml` is an ordered list
of extraction models (`model`, optional `base_url`, optional `api_key_env`).
`extract.Chain` tries each in order and returns the first success; the
combined error wraps `ErrParse` only when *every* model failed on content,
so a transport failure anywhere still means "retry next sweep" rather than
"skip forever". When the list is present it replaces `SCRY_MEMORY_MODEL` /
`SCRY_MEMORY_BASE_URL` outright (the daemon logs that it is ignoring them);
with no file, the env behaves exactly as before. Keys never live in the
file — `api_key_env` names the variable, defaulting to the existing
`SCRY_MEMORY_API_KEY` / `DEEPSEEK_API_KEY` lookup. This is the first thing
to actually use the `config.yaml` decided on 2026-04-10; it is parsed with
`gopkg.in/yaml.v3`, not viper — one section does not justify the dependency.

**Context:** three dead-letters in a day, all `deepseek-v4-flash`: two
empty replies (`reply: ""` even after both repair turns) and one invented
entity type (`"model"`). Each cost a real fact. `deepseek-v4-pro` sits on
the same endpoint with the same key, so a flash → pro chain rescues those
episodes for a few cents without touching a second provider. Empty replies
now also carry `stop_reason` and the content block types in the error,
because `reply: ""` on its own said nothing about *why*.

**Why replace rather than merge with env:** a file the user wrote on
purpose should mean exactly what it says. Layering env on top ("env is the
primary, config supplies fallbacks") makes the effective chain depend on
which shell spawned the daemon — the exact ambiguity the file exists to
remove.

**What would change our minds:** a fallback that should *not* fire on every
failure (e.g. only on empty replies, never on 4xx) — then `Chain` grows a
predicate. Or per-model batch support beyond Anthropic — today only the
primary is batched in `backfill`; the serial path runs the whole chain.


## 2026-08-28 — Daemon ownership: process-lifetime flock, launchd as the one start authority

The daemon takes an exclusive `flock` on `~/.scry/scryd.lock` before it
touches `scryd.sock` or `scryd.pid`, and holds it until the last deferred
teardown has run (`internal/daemon/owner.go`, wired at the top of
`Daemon.Run`). A starter that finds the lock held gives the holder
`StartupGrace` (3s) to answer on the socket: a healthy answer means it exits
with `ErrAlreadyRunning` (exit 0 from `scry start --foreground`); no answer
means the holder is retired — SIGTERM, wait up to `DefaultShutdownGrace+5s`
for the lock to be released, then SIGKILL and wait 3s more, then give up
with an error. Every step is logged with the target PID, signal delivery,
and time-to-release. A lock held by a process that recorded no PID is never
signalled (fail closed). Pre-lock daemons named only by the PID file get the
same grace/retire treatment, guarded by a `ps` command-name check against
PID reuse. The lock file is never unlinked. An exiting daemon removes the
PID file only if it still names that daemon.

On macOS, when a LaunchAgent whose `ProgramArguments` run `scry start
--foreground` exists in `~/Library/LaunchAgents`, clients that need the
daemon run `launchctl kickstart gui/<uid>/<label>` and retry the socket
instead of spawning a detached process; direct spawn remains the fallback
when there is no agent or launchctl refuses. `scry doctor` gained
`daemon.instances` (count of `scry start --foreground` processes vs the
canonical socket PID) and `daemon.memory_ui` (GET `/health` on the UI port
must be served by the canonical PID and be able to open the memory store).

**Context:** `docs/DAEMON_SPLIT_BRAIN_DIAGNOSIS.md`. Three foreground
daemons survived on the same machine: one on the RPC socket and Badger
memory lock, one on port 7279 (returning 500 because it could not open the
memory store), one holding watchers. launchd `KeepAlive` and client
auto-spawn had both decided the daemon was down; each retired only the PID
it could see, unlinked the socket pathname, and wrote its own PID. Unlinking
a Unix socket path does not close the previous listener, so the earlier
daemons stayed alive and unaddressable. `scry doctor` reported 0 failed.

**Why flock and not a smarter PID file:** the kernel releases a flock when
the holder dies, so "lock held" is proof of a live owner with no PID-reuse
ambiguity and no start-time bookkeeping; and holding it across the whole
teardown makes replacement-before-exit impossible rather than merely
unlikely. **Why launchd is the authority:** it is the only starter that
sources `~/.secrets.zsh`; a client-spawned winner runs dormant. **Why
escalate to SIGKILL:** a daemon that ignores TERM for 10s is hung, and the
alternative — leaving it — is exactly the orphan this fixes.

**What would change our minds:** a Linux service manager in daily use
(systemd `--user`), which would want the same "ask the supervisor" path
with a different discovery; or a daemon shutdown that legitimately needs
longer than 10s, which would raise `TermGrace` rather than remove
escalation.


## 2026-09-02 — Memory writes queue at the daemon; the daemon runs extraction

**Decision:** every memory write lands in a durable pending queue (`pq:`
keys in the memory store) and is extracted by a worker pool inside the
daemon that owns the store. `memory.remember` returns as soon as the
episode is queued, in milliseconds. The sweep and `scry memory ingest`
distill on the client and call `memory.enqueue`; they never touch a
provider. Transport failures retry with a backoff capped at two minutes;
an episode no model can parse after three tries is parked on disk and
replayed with `scry memory queue retry`. A model that answers 401/402/403
is skipped for fifteen minutes. Episode ids for remembers derive from the
fact text and the UTC day.

**Context:** `docs/MEMORY_AUDIT_2026-09-02.md`, findings 1 and 2. A
remember ran 40 to 600 seconds because the provider reasons before
answering; Codex's tool timeout is 60 seconds, so the agent saw a failure
while the daemon finished the write, and a retry stored a second episode.
When extraction failed for any non-parse reason the fact was gone. The
laptop sweep extracted client-side with the laptop's own config, which
still named a DeepSeek chain that had returned 402 for a day: 44 sweeps,
zero files ingested, 12,696 lines of 402 noise, and 380 socket timeouts
because one 30-minute context served the whole run.

**Why the daemon and not the client:** one process per store means one
config to keep correct (the mini's), one place keys have to exist, and a
laptop that needs no provider credentials at all. It also lets the store
know when ingestion last happened, which is what `scry doctor` needs.

**Why a queue and not a background goroutine per call:** the queue is the
durability. A goroutine dies with the daemon; a `pq:` record does not. It
also gives the outage story: the provider comes back, the worker drains.

**Why the schema version stays at 1:** a bump wipes the store. `pq:` (and
`att:`, which the alias-attestation work adds) are additive prefixes an
older binary ignores; `meta:` already existed.

**Known limits:** Badger runs without `SyncWrites`, so a queued write is
durable against a daemon crash but not against power loss in the same
second; the same has always been true of facts. `scry memory backfill` is
the one attended command that still extracts on the client, because it
uses the Anthropic batch API when pointed at Anthropic; the unattended
paths (sweep, ingest, remember) all go through the daemon.

**What would change our minds:** a second store owner (two daemons writing
one Badger dir is impossible anyway), or a provider so fast that inline
extraction is faster than the round trip to the queue. Neither is near.


## 2026-09-02 — OpenCode is read through the sqlite3 CLI

**Decision:** the OpenCode distiller shells out to `sqlite3 -json
-readonly` rather than linking a SQLite driver. Kimi Code's `wire.jsonl`
event log is parsed directly, like the Claude and Codex transcripts.

**Context:** OpenCode moved its sessions into a WAL-mode SQLite database.
The pure-Go driver (`modernc.org/sqlite`) is the sanctioned choice if scry
ever links SQLite, but it adds a large dependency and a minute of compile
time to read eleven rows every half hour. `sqlite3` ships with macOS and
nearly every Linux distribution, handles WAL correctly, and keeps the
binary CGO-free and small. A missing `sqlite3` is a per-root sweep error,
not a crash.

**What would change our minds:** scry needing SQLite for anything on the
hot path, or a machine without `sqlite3` in daily use.


## 2026-09-02 — Doctor fails when memory has not ingested for six hours

**Decision:** `scry doctor` dials the memory daemon (`SCRY_MEMORY_SOCKET`,
then `memory.socket` in `config.yaml`, then the local socket) and reads
`memory.status`. It fails on a dormant chain, a stopped queue worker, or a
last-ingest timestamp older than six hours; it warns on a sweep older than
two hours or a parked queue item.

**Context:** laptop ingestion was dead for a day and nothing said so. The
threshold is the goal file's; six quiet hours on a machine every agent
writes sessions to means the pipeline stopped. Overnight this can fail
honestly; a failing check that is sometimes loud beats one that is never
loud. The `memory.socket` config key exists so the shared-store location
has a home outside every MCP host's private env.


## 2026-09-02 — A closed relation vocabulary of 39 names, mapped in code

**Decision:** every stored fact's relation is one of the 39 names in
`resolve.Canonical` (`internal/memory/resolve/vocab.go`). `resolve.Map`
turns the model's verb into one of them: an exact synonym table built from
the 5,595 relation names observed in the live store, then rules for
negation prefixes (`does_not_use` → `lacks`), tense prefixes (`now_uses`),
`has_<noun>` (issue nouns → `has_issue`, measurement nouns → `status`,
else `contains`), trailing `_by` (passive voice: map the verb and swap the
endpoints), trailing prepositions (`deployed_to`), and finally a verb
stem table. Anything left lands on `related_to`. Inverse forms swap src
and dst (`used_by` → `uses` flipped). The raw relation is kept on the
fact as `raw_relation`. The extraction prompt was not changed.

**Context:** audit finding 4. 5,586 distinct relations across 27k facts,
the top 40 covering 51%. A path over that vocabulary meant nothing and a
query for what runs where had to guess between `deployed_on`,
`installed_on`, `exists_on`, `hosted_on`, and `running_on`. On the live
distribution the mapper types 94% of facts; 5.7% fall back.

**Why code and not prompt wording:** the model is allowed to be sloppy;
the resolver is not. A table with tests is inspectable and versionable;
a prompt is neither, and changing it invalidates the prompt cache.

**Why 39 and not 20:** below about 35 the merges became lies —
`monitors` folding into `documents`, `notifies` into `provides`. The
ceiling in the test is 40.

**What would change our minds:** a recall benchmark showing that a
relation the mapper folds is one agents ask about by name. Then it gets
its own entry, as long as the count stays under 40.


## 2026-09-02 — Values are attributes, never nodes

**Decision:** a fact whose target is a value — a status word, a number or
measurement, a version, a date, a git branch, a commit hash
(`resolve.IsValueName`) — is stored as an attribute fact: `Dst` empty,
`Value` set, the key's dst slot `~<value-slug>`, no reverse index. The
`status` relation is always an attribute. A fact whose source is a value
and whose target is an entity is turned around; one between two values is
dropped and counted. Attribute facts never create entities and path
traversal ignores them.

**Context:** audit finding 5. `main` (the git branch) had 374 facts,
`in-progress` had 241 and had collected "voice-of-customer" as an alias,
`51b-active-parameters` and `46-gib-spare-memory` were nodes.

**Why attributes and not dropping the facts:** "scry status is
in-progress" and "gpt-oss-120b has 51B active parameters" are worth
knowing; only the node was wrong. Fact-level search finds them by text.

**Why `~` in the key slot:** a slug is `[a-z0-9-]`, so an attribute key can
never collide with an edge key, and two different values on the same
(src, relation, valid_from) stay distinct.


## 2026-09-02 — Alias admission needs evidence and matching types

**Decision:** `resolve.AdmitAlias` decides whether an alias may be added
to an entity. Rejected outright: run artifacts, role words, values,
pronouns, and determiner phrases ("the machine"). Rejected always: an
alias owned by an entity of an incompatible type (`concept` is a
wildcard). Admitted at once: an unclaimed alias that shares a token with
the entity's name, aliases, or description. Otherwise — the alias is
another entity's name or alias, or shares nothing with the entity — it
is admitted only after two independent episodes have attested it
(`att:<slug>:<alias>` keys). A name that reaches an entity only through
an alias and names a different type gets its own entity. Concept stubs
upgrade to the first real type that mentions them.

**Context:** audit finding 5 and the live store: hermes-ops (project)
carried "Hermes", "mini", "Mac Mini", "the machine", "box", the Halo box,
and 120 more; the Qwen model carried gpt-oss-120b; the Jeff entity carried
"Claude", "codex", "you", "I". Every one came from a single episode.

**Why two episodes and not one with a confidence threshold:** the model's
confidence is uninformative (26,091 of 27,346 facts sit at 0.9 or
above). Independent repetition is the only evidence the store has.

**What would change our minds:** a legitimate alias that never repeats.
The description rule covers the common case (an entity described as "the
loop engine" admits "loop engine"); a human can always add an alias by
remembering it twice.


## 2026-09-02 — Migration relocates facts under a backup, never deletes text

**Decision:** `scry memory migrate` applies the three rules above to the
existing store. Relations are rewritten by relocating each fact to its
new key (the same operation `mergeFact` already used for ValidFrom), with
provenance, validity, confidence, and the raw relation preserved. When two
facts land on one key (two raw verbs from one episode that map to the same
canonical), the newcomer is shifted by a nanosecond so both keep their own
sentence and validity; nothing is merged away. Value entities have their facts converted to attributes and
are then deleted; facts between two values are invalidated, not deleted.
Hygiene v2 drops reference-word aliases, splits aliases away from
entities of another type or that are another entity's own name (moving
current facts whose text names the other entity), invalidates
self-loops, and re-points every `al:` key at its keeper. A backup is
taken first; a dry run is the default; a second run is a no-op.

**Why relocate rather than invalidate-and-recreate:** a relation rename
is the same logical fact under a corrected label. Invalidating it and
adding a copy would double the fact count and make as-of queries return
both. The house rule protects fact *content* and provenance; both survive
a relocation, and the backup covers the rest.


## 2026-09-02 — Recall ranks facts, not entities, and caps its payload

**Decision:** `memory.recall` finds facts by BM25 over fact sentences and
entity names (`internal/memory/search`, in-memory, rebuilt from the store
at daemon start and kept current through a store observer), boosts facts
that touch an entity the query names, adds a small recency term, returns
twenty facts by default, and trims the serialized result to 24 KB —
episodes first, then trailing facts. Entities come back as headers with a
fact count. `limit` is a fact limit. The MEMORY_SPEC's deferral of
fact-level search is un-deferred by this; its deferral of embeddings
stands.

**Context:** audit finding 3. The old recall matched entities by substring
and returned every current fact on the top five: "hermes deploy" returned
3,434 facts and 1.18 MB because `deploy` was an alias of
childscribe-laravel; "Z_AI_API_KEY" returned 1,864 facts because `key` was
an alias of Jeff. MCP hosts truncated the result and the agent saw an
arbitrary slice.

**Why BM25 and not embeddings:** the house rules allow local embeddings
but forbid hosted ones, and a local model is a new binary dependency for
a corpus of 30k short sentences that lexical ranking handles. On the
migrated store copy the seven audit probes each return under 24 KB with
the intended entity's fact in the top five. Embeddings become worth it
when the benchmark shows paraphrase misses that no synonym rule fixes.

**Why in memory and not on disk:** 30k documents build in about a second;
persisting the index would add a key layout to keep in step with the
store for no gain.

**What would change our minds:** a store an order of magnitude larger,
or a benchmark miss pattern that is semantic rather than lexical.


## 2026-09-02 — The distiller attests the repo; the daemon never stats a path

**Decision:** `RawEpisode.CwdIsRepo` is set by the distiller on the machine
where the session ran (`distill.CwdIsRepo` stats `<cwd>/.git` there) and
travels with the episode through the queue. The resolver records a repo
ref only when the flag is set; `isWorkspacePath` and hygiene no longer
stat anything. `scry memory ingest --force` re-queues episodes the store
already holds so they are re-applied under the current rules.

**Context:** the shared store moved to the mini on 2026-08-28. From then on
every laptop session's cwd was checked for a `.git` directory on the mini,
where `/Users/jeff` does not exist, so no laptop-origin entity received a
repo ref and `scry memory orient` in a laptop repo showed nothing from the
Kimi, OpenCode, or Claude sessions that touched it. The grader for
done-bar item 6 found it.

**Why a flag and not a path map:** the fact to record is "this path was a
repository when the session ran", and only the machine that ran it knows.
A daemon-side allowlist of repo roots would go stale the day a repo is
cloned.


## 2026-09-02 — Kind words, spelling variants, and the alias keeper

**Decision:** an alias that is another entity's name plus that entity's
kind words ("Hermes agent" for the service Hermes, "Halo box" for the
machine AMD Halo, "hermes-agent") names that entity: the write path refuses
it on any other entity of an incompatible type and hygiene splits it off
and grants it to the entity it names. Ownership lookups compare compact
spellings ("halo1" is `halo-1`, "Bryan.Farney" is `bryan-farney`,
"deepresearch/agent.py" is `deepresearch-agent-py`). A concept stub never
takes a typed entity's own name. When several entities list one alias,
the keeper is the one whose name and kind words compose it, else the one
whose own name shares a token with it with the most facts. An entity's
own name in another spelling is never split off it.

**Context:** the item 5 grader watched the live write path put "hermes
agent" back on hermes-ops minutes after the migration had moved it, via
the shares-a-token shortcut, and showed spelling variants crossing types.
A degree-based keeper handed "Hermes agent" to the busier project and the
service and project traded it every pass.

**Why kind words are per type:** "agent" makes a service, "box" makes a
machine, "repo" makes a project. The lists are short and in code
(`resolve.kindWords`), where a table test can hold them.


## 2026-09-03 — Orient shows what happened in this repository

**Decision:** `scry memory orient` ranks a repository's entities by
whether their facts came from a session that ran there (episodes carry
their working directory), then by the day the entity was last seen, then a
typed entity over a bare concept, then how few repositories the entity
claims, then degree. Facts within a bullet prefer the same local
provenance. Bullets are clipped to 150 characters, two facts per entity,
eight entities.

**Context:** the blurb listed entities alphabetically and quoted three
facts at full length, so an orientation for the cleaning-company repo
opened with "36px-card-layout" and a thread-pitch table and named three
things in 2,000 characters. It also could not distinguish the work done
in a repo from a category entity that merely claims six repositories.

**Why local provenance and not a fact count:** an entity like `jeff` or
`docket` touches every repo; the question an orientation answers is "what
happened here", and the episode's own working directory is the only
honest answer to that.

**What would change our minds:** an orientation that needs the whole fact
text (then the budget rises rather than the clip).


## 2026-09-03 — A Kimi step is a turn

**Decision:** `distill.KimiWire` closes the assistant turn at each
`step.end`, not at the end of the whole model turn.

**Context:** a Kimi subagent session is one prompt followed by dozens of
tool-driven steps. Accumulating all of it into a single assistant turn
left such a session with two substantive turns, below the three-turn floor
every distiller applies, so the log produced nothing: 112 of 125 Kimi
logs on this machine, every one larger than 100 KB, were silently
dropped. Per-step flushing matches how a Claude transcript is already
shaped (one message per model call) and takes the same logs to 101 files
and 126 episodes. Reasoning parts are still never stored.

**What would change our minds:** a source whose steps are so fine-grained
that a step is not a unit of work worth its own episode turn.


## 2026-09-03 — One thing is deployed in more than one place

**Decision:** `deployed_on` leaves the exclusive relation set, which now
holds `status` and `replaced_by` only.

**Context:** exclusivity means a new target retires the old one, and it
was applied to deployments. Recording that cockpit reached the mini
therefore retired the fact that cockpit serves its own MCP daemon on a
port, and recording production retired staging. The live store held 497
retired `deployed_on` facts, among them where each web application runs.
A subject has one status and one successor; it does not have one host.

**The repair:** the migration brings back only what exclusivity took. A
retired `deployed_on` fact returns when its `invalid_at` is exactly the
`valid_from` of another `deployed_on` fact from the same entity, which is
the stamp Rule 6 leaves behind. Anything retired for another reason keeps
its invalidation, because a fact is never assumed wrong just because it
is old.

**What would change our minds:** a relation that names a single live
placement (`primary_host`, say) would be exclusive on its own terms
rather than by widening this one.


## 2026-09-03 — An episode may not retire a fact newer than itself

**Decision:** Rule 6 compares the episode's `OccurredAt` against the
current fact's `ValidFrom`. A current fact that is newer stays current,
and the incoming fact is written as already over, valid until the newer
one began. Both sides survive, in the right order.

**Context:** sweeps find transcripts in whatever order the filesystem
hands them over, and the queue works newest-first per source, so a July
session is routinely resolved after an August fact is already stored.
Exclusivity then retired August and left July current. It had happened
984 times: the fact that the app association file went live sat retired
behind an earlier fact saying the CDN had not propagated yet. A third of
every invalidation in the store, 1,776 facts, had been stamped within two
seconds of the fact's own start, which is the signature of this.

**The repair:** the migration swaps an inverted pair for an exclusive
relation, and for a relation that is no longer exclusive restores the
newer fact and leaves the older current. A fact retired long after it
began is a real change and is left alone. Two seconds is the cutoff: two
facts that genuinely follow one another are minutes apart at the least.

**What would change our minds:** a source whose episodes carry no usable
occurrence time, which would make the comparison meaningless.


## 2026-09-03 — Recall demotes restatements

**Decision:** after ranking, recall demotes a fact whose words are mostly
a higher-ranked fact's words, and allows two facts per pair of entities
in the answer window. Demoted facts sit below the distinct ones rather
than disappearing. Numbers are compared before words: two facts that
differ only in an address octet, a port, or a version are never one
sentence.

**Context:** "how does the laptop reach the shared memory graph" returned
six restatements of one load test in its top six and never returned the
sentence that answers it. Twenty sessions saying one thing is still one
thing, and it should not cost twenty of the twenty slots. The numbers
rule came from the opposite failure: without it, two interfaces with
addresses ending .1 and .2 read as the same sentence, and the answer to a
question about one of them was demoted as a duplicate of the other.

**What would change our minds:** a question whose answer genuinely needs
three facts about one pair of entities in the top twenty.


## 2026-09-03 — A benchmark question may name several phrasings

**Decision:** a question in `docs/memory-bench/*.json` may list its answer
under `any_of` rather than a single `fact_substring`.

**Context:** memory keeps more than one sentence for the same thing,
restated by different sessions. Five questions scored as misses while an
equally good answer sat in the top five: the address of the mini, what
Hermes falls back to, when the Laravel app deploys, whether a child's
voice is retained, which hook refused the commits. One of the five named
a fact a later session had superseded, so the question was scoring
against history. Pinning a question to one wording measures the wording,
not the retrieval.

**The risk, stated plainly:** this makes it possible to move the bar
instead of meeting it. The guard is that the graders write their own
held-out questions and never see this file, that the strict question
file is kept beside the loosened one and both numbers are always
reported, and that every alternate is recorded in the audit with the
reason.

**What would change our minds:** a question whose alternates cannot be
justified one by one, or a strict score that stops being reported
alongside the loose one. Either means the file has become a way of
scoring rather than a way of measuring.


## 2026-09-03 — An alias belongs to the entity it names

**Decision:** an alias is the holder's own only when it spells the
holder's name or contains every word of it. An alias that spells another
entity's whole name, and does not spell its holder's, belongs to that
other entity whatever the extra words are, matched past plurals and
spelling variants. A new entity named after another entity's alias takes
that name from it.

**Context:** this replaces the rule from 2026-09-02 in which sharing one
token with the holder's name admitted an alias immediately. A grader
watched the live write path put "Hermes repo", "Hermes tmux", and
"Jeff's own Hermes" onto the hermes-ops project under that rule, which is
how unrelated things fused: one shared word is not evidence.

**Reversal noted:** the earlier entries "Alias admission needs evidence
and matching types" and "Kind words, spelling variants, and the alias
keeper" still hold on evidence and types. What changed is the immediate
path: extras are no longer required to be kind words, because the test is
now which entity the alias names rather than what the extra words are.

**What would change our minds:** a real alias that names a second entity
in passing and belongs to neither — a compound product name, say, where
both halves are already entities.


## 2026-09-03 — A name beats an alias

**Decision:** an entity whose own name is another entity's alias takes
that name, and the alias is dropped from the holder, when the two types
are incompatible.

**Context:** a machine called "Mac mini" appearing after a project had
already collected "Mac mini" as an alias otherwise leaves the alias
pointing at the project forever, and every later mention resolves to the
wrong thing. A name is stronger evidence than an alias: it is what a
session called the thing when introducing it.

**What would change our minds:** an entity created from a passing mention
whose name is wrong, which would then take an alias from the entity that
deserved it. Type incompatibility is the guard.


## 2026-09-03 — A status edge to a real entity is not a status

**Decision:** when a `status` fact points at an entity rather than a
value, the relation becomes `related_to` and the edge is kept.

**Context:** `status` is exclusive, so making one entity another's status
means the next status invalidates it. A model writing "the dotfiles
status is the setpoint fleet" then quietly retires a real relationship. A
status is a value; an edge between two things is a relationship.

**What would change our minds:** nothing seen so far. The conversion is
lossless, and the fact keeps its sentence.


## 2026-09-03 — The queue finds its own concurrency

**Decision:** the extraction worker starts at six items in flight, halves
on a rate-limit refusal down to a floor of two, and widens by one after a
run of successes or after 45 quiet seconds while saturated, up to 24.

**Context:** a fixed 24 produced 300 refusals in 300 attempts against
Z.ai, and a fixed small number wastes a provider that is willing. Neither
the provider's limit nor its mood is knowable in advance, and it changes
with the plan and the hour. Additive growth with multiplicative backoff
is the standard answer and needs no configuration.

**What would change our minds:** a provider that publishes its limit,
which would be worth reading instead of discovering.


## 2026-09-03 — A pass that writes takes a backup and asks

**Decision:** every store-mutating pass takes a backup into
`~/.scry/backups` first and writes only under `--apply`.

**Context:** `scry memory repair-repos` shipped writing by default with an
opt-in `--dry-run` and no backup, and was run against the live shared
store. It was additive and no harm came of it, but the house rule covers
every hygiene pass and the shape of the flag decides what happens on a
mistyped command.

**What would change our minds:** nothing. The cost is one backup file per
repair.


## 2026-09-03 — Doctor measures work done, not workers running

**Decision:** the extraction check fails when the whole model chain is
being refused over billing or authentication, warns when part of it is,
and the queue check fails when it holds work and nothing has been
extracted for thirty minutes — or ever.

**Context:** both provider accounts emptied on 2026-09-03 and the health
check reported a passing chain with a running worker for hours, while the
daemon wrote 12,906 refusal lines. A worker can be perfectly alive and
producing nothing. Thirty minutes is chosen against the queue's own
retry schedule: a backlog that has not moved in thirty minutes is not
between attempts.

**What would change our minds:** a provider outage long enough to be
normal, which would make the check noise rather than news.


## 2026-09-03 — Query expansion has one form that works

**Decision:** the only expansion recall performs is a synonym table of
English words, plus joining each adjacent pair of question words. Two
other forms were built, measured, and deleted.

**What was tried.** Relevance feedback: take the highest-signal words
from the twenty facts a question already reaches and add them to the
query, which is the textbook answer to a vocabulary gap. It made every
question set worse at every weight down to a tenth — the neighbourhood's
words pull the query toward the neighbourhood and away from the one fact
that answers it. Entity-name expansion: add the name of each entity the
question resolves to, on the theory that a question saying "the mobile
app" needs the word childscribe-mobile. Worse again, and for the same
reason: an entity's name brings all of that entity's facts.

**What survives, and why it is narrow.** A synonym maps an English word
to an English word. It never maps a word to the name of an entity: "box"
once expanded to mini and halo, and a grader measured the whole table as
negatively correlated with success on questions it had not been fitted
to, naming that entry as the mechanism. Joining adjacent words is not
really expansion — it is spelling, reaching a name written as one word
from a question that writes it as two.

**Coverage weighting was tried twice and does not help.** Every grader
names polysemy as the failure mode: one common word in the question
collides with a different sense elsewhere in the store, and a document
repeating that word outranks one carrying three of the question's words.
Weighting a document by how much of the question it accounts for is the
textbook answer, and it was measured at weights from one to four on
three question sets, before and after the ranking was rebuilt around a
vector re-rank. It never gained an answer and above weight two it lost
them. Removed both times.

**What is left unsolved, stated plainly.** The remaining misses are
conceptual. A question about someone who "cannot fake his way through" a
subject has to reach a fact about having "no sports domain knowledge",
and no thesaurus and no reweighting of words will do that. Closing it
needs meaning, which means embeddings. The house rules allow local ones;
the mini's model server accepts an embeddings request but has no
embedding model loaded, so that is a decision for the machine's owner
rather than something to arrange quietly.

**What would change our minds:** a local embedding model on the store's
machine, at which point the lexical index becomes the first stage of a
hybrid rather than the whole of it.


## 2026-09-03 — Meaning comes from the store's own word company

**Decision:** every fact carries a 128-dimension vector learned from the
store itself by random indexing, and recall adds a small multiple of the
cosine between the question's vector and the fact's to the word score.

**Context:** the spec deferred embeddings; this run's goal un-defers them.
Three graders in a row showed the same thing: every miss was a ranking
failure with the answering fact live in the store, and the remaining ones
are conceptual rather than lexical. A question about someone who "cannot
fake his way through" a subject has to reach a fact about having "no
sports domain knowledge", and no thesaurus, reweighting, or feedback pass
gets there — all three were built and measured, and two were deleted.

**Why random indexing and not a model:** the house rules allow local
embeddings and forbid a hosted embedding API. A downloaded model is local
but is still a model someone has to install on the store's machine, a
decision that belongs to its owner and a dependency this project does not
want. Random indexing needs none: each word gets a sparse random vector
from its own hash, a word's meaning is the sum of the vectors of the
words it appears with, and a fact is the sum of its words' meanings.
Words that keep the same company end up pointing the same way. It is one
pass, pure Go, no CGO, no network, about two seconds over 46,000 facts
and 30 MB in memory. Episode summaries and entity descriptions are read
as company but never returned, so the model learns from more text than
recall answers with.

**The weight is small on purpose.** Swept from zero to thirty-two against
three question sets: at eight the strict tuning set is unchanged at 44 of
50 while the two sets it was not fitted to gain, 51 to 54 of 62 and 33 to
34 of 66. Above sixteen it starts costing questions that words answer
perfectly well. Meaning is a nudge past a fact that merely repeats a
word, not a replacement for matching one.

**What would change our minds:** an embedding model the store's owner
already runs locally, at which point this becomes the fallback for words
that model has never seen rather than the whole of the semantic layer.

**Graph traversal was tried and is not here.** The reference designs pair
entity and fact search with a hop through the graph, and the goal names
them. Facts on entities one hop from the ones a question names were
scored at every weight from a third to four fifths, and the three
question sets did not move by a single answer at any of them. A
neighbour's facts are already in the lexical candidate window when they
match, and below it when they do not. It is out rather than kept at a
weight where it does nothing, which is the same call made for relevance
feedback and entity-name expansion. What the attempt did surface is that
facts on a named entity were scored without the meaning term while
searched facts had it; that inconsistency is fixed.


## 2026-09-03 — What separates a value from a thing, and what does not

**Decision:** the value rules stay lexical, and their ceiling is stated
rather than papered over. Two alternatives were measured and neither
works.

**Usage statistics do not work.** A value ought to be mentioned once and
never again, and against a list of 50 known values and 29 well-known
entities the signal looks decisive: the values have a median of one fact
and 44 of 50 have a single episode; the entities have a median of 142
facts and none has a single episode. But that comparison is rigged by
the second list. Applied to the store, "one episode and untouched for a
fortnight" selects 9,999 of 18,166 live entities, 55 per cent, and a
random sample of them is full of real things: `PerfLintTest.php`,
`ChapterService::entriesPayloadFor()`, `commit-msg-linter`,
`feedback_submissions table`, `v1-rest-api`, `AWS BAA`. A rarely
mentioned thing and a value are the same shape in the graph. Measured
before it was built, and not built.

**Four rounds of shape rules have a ceiling, and it has been reached.**
Each grader hand-picks names the rules accept, the rules grow to cover
them, and the next grader finds a family one token to the side: digits
but not words, `feat/` but not `goal/`, `approved` but not `confirmed`,
`task-<hex>` but not `task_<hex>`. The fourth round caught none of its
68. The rules are 844 literal tokens across 23 tables, and the honest
description is that they encode what earlier rounds found rather than a
theory of what a name is.

**What the false-positive side is worth — corrected 2026-09-03.** This
entry claimed the false-positive side was "genuinely good", 80 of 81
hard names surviving, and concluded the rules erred in the cheap
direction. A grader showed the measurement was circular: every list of
"hard real names" had been drawn from or checked against the store, and
the store cannot contain a name the rules reject, because such a name
was pruned or never created. Measured on names chosen independently —
invented but plausible, and real directories from these repositories —
the survival rate was **19 of 56**. Sixteen of sixteen two-segment paths
such as `terraform/modules` and `helm/charts` were called branches,
eighteen of eighteen names opening with a verdict word such as
`deferred-revenue-ledger` were called verdicts, and twenty-five of
thirty-two ordinary compounds such as `trade-off` and `start-up` were
called statuses.

So the rules erred in the *expensive* direction, and the sentence
asserting the opposite was the load-bearing reason this item was treated
as nearly closed. Four defaults were inverted in response: a two-segment
name is a directory unless its head is a branch namespace, a verdict
phrase is at most two words unless a preposition makes it prose, a
hyphenated compound is one word rather than a phrase, and a particle
compound is a noun. The test for this lives in
`shapes_real_names_test.go` and is built from names that are *not* in
the store, which is the only version of the test that means anything.

**What would actually close it:** the judgement belongs where the
language is understood. The extraction model reads the transcript and
already names and types every entity; asking it to say whether a name
is a thing or a measurement of one is a question it can answer and a
regex cannot. That is a design change, it costs a provider call per
episode, and it is not something to slip in while the provider is
unreachable — so it is written down here rather than half-built.


## 2026-09-03 — The stemmer files some words twice, and every fix measured worse

**Decision:** the stemmer is left as it is, and the defect is written
down rather than fixed.

**The defect is real.** Stripping a suffix leaves a stem without the
silent e the base form keeps: "merging" becomes "merg" while "merge"
stays "merge", so a question asked in one inflection never meets a fact
written in the other. A grader counted 585 pairs of words split this way
in the live store — "route" in 1,511 documents and "rout" in 2,078,
"worktree" in 1,524 and "worktre" in 213 — and showed four of its six
misses flip to a hit when a single word of the question is changed to
the fact's inflection, with no new information added.

**Three fixes were measured against 250 questions across four sets.**
Dropping the silent e from both sides, so a word has one bucket: 63, 43,
52, 35 against a baseline of 62, 44, 53, 35 — one gained, two lost.
Emitting both forms so nothing is displaced and matches can only be
added: 61, 43, 46, 30, much worse, because duplicating a token inflates
term frequency and document length and distorts the ranking that reads
them. The grader's own additive variants scored 61 and 63. Every one
trades questions one for one.

**Why it stays.** A change that does not measure better does not ship,
which is the same rule applied to relevance feedback, entity-name
expansion, graph traversal, coverage weighting, and vector retrieval
this session. A real fix is a real stemmer — Porter, about two hundred
lines — which would handle the whole family consistently rather than
patching the one suffix. That is worth doing when there is a question
set big enough to tell a real gain from noise; at 250 questions a
one-for-one trade is indistinguishable from either.


## 2026-09-03 — Facts cannot be refiled by what their sentence names

**Decision:** no rule moves a fact from one entity to another on the
strength of what its text mentions. The misfiled facts a grader found —
31 on the hermes-ops project describing the Mac mini or a Halo box —
stay where they are, and the reason is recorded instead.

**What was built and measured.** A migration step that moves a fact's
endpoint when the sentence names exactly one other entity by its whole
distinctive name, never names the entity it is filed under, and the two
are of different kinds. On the live store it proposed **9,329 moves**,
landing facts on entities called `allow`, `setup`, `defined` and
`delivery` — ordinary words that happen to be entity names.

Tightened to hardware only, and only hardware the store says at least
ten things about, it proposed 54. Those were still wrong: `sandbox` is
typed `machine` in this store, so every sentence about a sandbox
*permission* pulled a fact onto it.

**Why the signal is not there.** Two things it depends on are unreliable
at once. Entity types are extraction output, and a grader counted a
large share of the 319 "machines" as worktrees, directories, database
tables and files. And a name distinctive enough to search for is not the
same as a name distinctive enough to move data on: `sandbox`, `delivery`
and `setup` all pass any test of specificity a regex can make.

**What would close it:** correct types. The judgement "this sentence is
about a machine" is one the extraction model can make while it is
reading the transcript and a regex cannot make afterwards — the same
conclusion the value rules reached. Until then a misfiled fact stays
misfiled, and the audit says how many there are rather than pretending
otherwise.

**What would change our minds:** a fact carrying its own subject from
extraction, rather than the resolver inferring one from a sentence.
