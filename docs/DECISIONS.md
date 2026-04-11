# scry — decision log

Architectural and scope calls that deserve a durable written record. One
entry per decision. Newest at the top. Each entry must answer: what, why,
what would change our minds.

This file resolves the open questions in `docs/SPEC.md` §15. The PHP
calibration findings live in `docs/PHP_CALIBRATION.md`.

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
