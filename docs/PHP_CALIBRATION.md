# PHP / Laravel calibration — day-1

**Date:** 2026-04-10
**Target:** `~/herd/hoopless_crm` — Laravel 12, PHP 8.4, ~1199 PHP files / 174k LOC in `app/`
**Indexer:** [`davidrjenni/scip-php`](https://github.com/davidrjenni/scip-php) main branch (`97a2d8d`, 2026-03-31)
**Recommended by:** SPEC §11.1, "day-1 calibration exercise"

This is the spec-recommended exercise: clone `scip-php`, run it against a real Laravel app, document what it catches and what it misses, and use that to size the P2 Laravel post-processor before week 6 instead of after.

---

## TL;DR

scip-php is **better than the spec implied** at the static-PHP layer, and **more constrained than the spec implied** at the install layer. The maintained main branch is real (PHP 8.4 support, recent fixes), but it has never been released to Packagist — the only published version is **v0.0.2 from 2023**, which has a security advisory and chokes on PHP 8.4 syntax.

Once running, scip-php captures everything that `::class` references and `use` statements give you for free, including facade calls (as refs to the facade class itself, not the backing class) and `app(Foo::class)` container resolution. The real gap is **files outside PSR-4 paths** — routes, migrations, config, blade templates — which scip-php skips entirely. For hoopless_crm, that's **1168 `::class` references in `routes/` alone** that the index never sees.

The P2 post-processor scope is therefore not "rebuild what's missing" but "fill three specific holes." Sized below.

---

## Install reality

The spec assumes you can `composer require davidrjenni/scip-php` and go. You can't, cleanly:

| Path | Result |
|---|---|
| `composer require --dev davidrjenni/scip-php` (Packagist v0.0.2) | Refused — `google/protobuf ^3.22` has security advisory PKSA-tcfz-w4fm-hhk9, blocked by composer audit. |
| `composer require --dev davidrjenni/scip-php:dev-main` via VCS | Works with `-W` to allow related upgrades. Bumps `nikic/php-parser` 5.4.0 → 5.7.0, `phpdoc-parser` 2.1.0 → 2.3.2, installs google/protobuf v5.34.1, `composer/class-map-generator`, `jetbrains/phpstorm-stubs`. ~10 package changes total. |
| `docker run davidrjenni/scip-php:latest` | The published image is still v0.0.2 (2023-04-23). PHP 8.2.5. Crashes on hoopless_crm with `verifyModifier` undefined — the bundled php-parser is too old for PHP 8.4 syntax. |
| Run scip-php from its own clone, point at external project | Crashes with `Int_::KIND_INT undefined` — autoloader collision: scip-php loads the target's `vendor/autoload.php` to resolve user symbols, but the target's `nikic/php-parser` v5.4.0 gets autoloaded first and the v5.7.0 internals scip-php needs are gone. PHP can't host two versions of one package in one process. |

**Implication for scry's auto-download flow (SPEC §9 step 2):**

The "vendor a wrapped binary" option (PHAR via box-project/box) from §11.2 is more important than the spec made it sound — it's the only path that avoids the autoloader collision. We need scip-php to ship as a self-contained PHAR with its dependencies frozen, mounted into the target project's working directory but **not** linked into its autoloader for its own internals. That's exactly what the PHAR `_composer_autoload_path` indirection in scip-php's `bin/scip-php` enables, but only if the PHAR is built fresh from main, not from the v0.0.2 release.

**Concrete scry-side action items:**

1. P1 indexer-install for PHP must build a PHAR from a pinned `scip-php` main commit (currently `97a2d8d`), not download a Packagist tarball or pull the docker image.
2. Pin and document the commit; refresh quarterly. Treat scip-php like an unreleased dependency we vendor.
3. The "experimental" flag from the SPEC should default to OFF until the install path is reproducible across at least three Laravel codebases.

---

## Run characteristics on hoopless_crm

Once installed via `composer require --dev davidrjenni/scip-php:dev-main -W`:

| Metric | Value |
|---|---|
| Wall time, full index | **45s** (42.6s user, single-threaded) |
| Output `index.scip` size | 14 MB |
| Documents indexed | **1340** |
| Definitions | **18 892** |
| Occurrences (refs + defs combined) | **69 178** |
| Median occurrences/document | 31 |
| Median definitions/document | 8 |
| Files in `app/` (PSR-4 root) | 1199 indexed |
| Files in `database/factories/` | indexed via `Database\Factories\` PSR-4 |
| Files in `tests/` | indexed via `Tests\` PSR-4 |
| Files in `routes/`, `database/migrations/`, `config/`, `bootstrap/`, `resources/views/` | **0 indexed** |

For SPEC §10's perf target ("Cold index build, 100k-LOC TS repo: <60s"), 174k LOC of PHP in 45s is comfortably on track — though scip-php is single-threaded and we can't easily parallelize it.

---

## What scip-php catches

Verified by inspecting the SCIP output for known patterns in hoopless_crm:

### Static class references — ✅ full coverage

- `use App\Models\User;` and similar `use` statements
- `User::class` — every `::class` literal becomes a ref to the named class
- Type hints in parameters, returns, properties: `function foo(User $u): Activity`
- `extends`, `implements`, `instanceof`
- Method calls on typed receivers: `$user->name` is captured because `$user` is statically typed via PHPDoc or type hint

### Facade calls — ✅ captured (as refs to the facade class)

`Auth::user()` produces two occurrences:
- Range covering `Auth` → `Illuminate/Support/Facades/Auth#`
- Range covering `user` → `Illuminate/Support/Facades/Auth#user().`

This means `scry refs Illuminate/Support/Facades/Auth#user()` would correctly return all 177 `Auth::user()` call sites in hoopless_crm.

The catch: **the ref does not point to the backing class.** `Illuminate\Auth\AuthManager::user()` and `Illuminate\Contracts\Auth\Guard::user()` show no refs from facade calls. An agent asking "where is `AuthManager::user` called?" gets the wrong answer.

### Container resolution `app(Foo::class)` — ✅ captured

`app(\App\Services\BraveSearchService::class)` produces:
- Ref to `app()` (the helper function in `laravel/framework`)
- Ref to `App/Services/BraveSearchService#` — via the `::class` literal

So `scry refs BraveSearchService` correctly finds container-resolved instantiation sites. **For hoopless_crm specifically, this is fully solved** — there are zero `bind()` / `singleton()` calls in `app/Providers/`, so the codebase relies entirely on auto-resolution and `app(Foo::class)`. The "service container binding" gap the SPEC worries about isn't a real gap for this codebase.

### Eloquent relationship targets — ✅ captured (the static half)

`return $this->belongsTo(User::class);` produces:
- Ref to `Illuminate/Database/Eloquent/Relations/BelongsTo#` (the return type)
- Ref to `App/Models/User#` (via `User::class`)
- Definition of `Activity#user().` (the method itself)

So `scry refs User` will return the `belongsTo(User::class)` line as a User reference. That's the half of an Eloquent edge that static analysis can give you.

### What it captures in numbers

| Pattern | Occurrences in code | Captured |
|---|---|---|
| `::class` references in `app/`, `database/factories/`, `tests/` | thousands | ✅ all |
| `Auth::*` facade calls (in indexed files) | 177 | ✅ all (as refs to facade class) |
| `app(Foo::class)` container helper | dozens | ✅ all |
| Eloquent relation declarations (`belongsTo`/`hasMany`/etc) | 431 across 129 models | ✅ static half (the related model class) |

---

## What scip-php misses

### 1. Files outside PSR-4 paths — ❌ completely unindexed

This is the **biggest** gap and the SPEC understated it.

| Directory | Files | LOC | Status |
|---|---|---|---|
| `routes/` | 23 PHP files | 2510 | unindexed |
| `database/migrations/` | many | many | unindexed |
| `config/` | many | many | unindexed |
| `bootstrap/` | a few | a few | unindexed |
| `resources/views/` | 14 blade files | — | unindexed (not PHP at all from scip-php's POV) |

`routes/web.php` alone has **1168 `::class` references** to controller classes. None of them appear in the SCIP index. So `scry refs UserController` will miss every `Route::get('/users', [UserController::class, 'index'])` line — and routes are exactly where an agent goes to figure out what URL maps to what controller.

This is the single highest-leverage thing the post-processor must fix.

### 2. Facade → backing-class edge — ❌ missing

`Auth::user()` resolves only to `Facades\Auth::user`, not to `AuthManager::user` or `Guard::user`. An agent that knows the contract name, not the facade name, gets the wrong answer.

For hoopless_crm with 177 `Auth::*` calls alone, this matters. Across all of Illuminate facades (Cache, DB, Mail, Queue, Session, Storage, etc.) it matters more.

### 3. String-keyed references — ❌ missing

| Pattern | Count in `app/` | Captured |
|---|---|---|
| `view('users.show', ...)` | 7 | ❌ |
| `config('mail.from.address')` | **300** | ❌ |
| `route('proposals.public')` | (not measured) | ❌ |
| `__('messages.welcome')` / `trans(...)` | (not measured) | ❌ |

The `view()` calls are low volume in this codebase (7), but `config()` calls are high volume (300) and the rdeps "what files reference config key X" question is genuinely useful.

### 4. Eloquent property/relationship semantics — ❌ missing (and probably fine to defer)

`$activity->user->name` doesn't link `user` to `App\Models\User` and `name` to a column. To add this, the post-processor would have to walk every model's relation methods and synthesize property→target edges. Doable but high-effort and the static `belongsTo(User::class)` half already covers most user intents.

---

## Sizing the P2 Laravel post-processor

The SPEC's §11.1 listed six things for P2 to handle. Re-prioritized by what hoopless_crm actually shows:

| # | Pattern | SPEC priority | Real impact | Effort estimate |
|---|---|---|---|---|
| 1 | **Index `routes/`, `migrations/`, `config/`, `bootstrap/`** as flat PHP files (capture `::class` and `use` references the same way scip-php does, just from outside the PSR-4 walker) | not in SPEC | **Huge.** 1168 ref losses in `routes/` alone for hoopless_crm | **Small-medium.** Walk the directories, parse each file with `nikic/php-parser`, emit synthesized SCIP `Document` records. Reuses scip-php's symbol shape. ~1-2 days. |
| 2 | **Facade → backing-class edges** | listed | Real, broad — ~hundreds of facade calls in any Laravel app | **Medium.** Walk every facade class, read `getFacadeAccessor()` (often a string returning a service container alias), look up the alias in container service map, add a synthesized ref edge from each facade method occurrence to the backing-class method. Some facades are easy (`getFacadeAccessor` returns a literal string), some are hard (returns dynamic). Cover the top ~30 framework facades and call it done. ~3-5 days. |
| 3 | **`view('foo.bar', ...)` → blade file edge** | listed | Low volume in hoopless_crm (7) but probably higher in non-API-heavy apps. Adds blade files as a new "document" kind. | **Small.** String literal extraction. Synthesized `Document` for each blade template. ~1 day. |
| 4 | **`config('foo.bar')` → config file edge** | not in SPEC | High volume in hoopless_crm (300). Useful for "where is this config key read." | **Small.** ~1 day. |
| 5 | **Service container `bind`/`singleton`** | listed | **Zero impact for hoopless_crm.** No bindings in providers. May matter elsewhere. | **Defer until a codebase actually needs it.** |
| 6 | **Eloquent property/relationship semantics** (`$user->posts->first()->title`) | listed | Real but high effort and not the worst gap | **Defer to P3.** Static `belongsTo(Foo::class)` ref already covers most agent intents. |

**Recommendation:** scope P2 to items 1-4. Item 1 is the unblock. Items 2-4 each add a discrete value layer.

Items 5 and 6 should not be in v1's PHP P2.

---

## Indexer engine: don't replace yet

The SPEC §11.1 reserved a "P3: replace engine with Phpactor" escape hatch. Based on calibration:

- scip-php main branch is **maintained well enough** for v1. Recent commits show real fixes (PHP 8.4 dockerfile bump, phpstan v2 upgrade, php-parser/phpdoc-parser bumps with code changes, type annotation improvements). It is not abandoned.
- The static-analysis layer scip-php gives us is **already good** at the things static analysis can do.
- The gaps are not "the engine is wrong" — they're "static analysis can't see strings or non-PSR-4 files." Replacing the engine wouldn't help with those.

**Verdict:** stay with scip-php. Defer the Phpactor swap until/unless a specific failure mode forces it. Make the engine swappable (per SPEC), but don't preemptively swap.

---

## Action items going into P0

1. **No P0 work needed for PHP** — P0 is TypeScript only. PHP work starts in P1.
2. **Pin scip-php to commit `97a2d8d`** when P1 lands. Document in `internal/sources/php/README.md`.
3. **Build the PHAR distribution flow first**, before writing any of the post-processor. The PHAR is the only install path that doesn't collide with the target project's autoloader.
4. **P2 scope (revised):** add a "non-PSR-4 file walker" (item 1 above), facade resolver (item 2), view template ref (item 3), config key ref (item 4). Skip container bindings and Eloquent property semantics for v1.
5. **Refresh this calibration** when scip-php's pinned commit moves, when Laravel's facade-accessor patterns change materially, or when a new target codebase reveals a pattern this one didn't.

---

## Reproducing

```bash
# Clone scip-php (use main, not the v0.0.2 release)
git clone https://github.com/davidrjenni/scip-php /tmp/scip-php
cd /tmp/scip-php && composer install --quiet

# Install into target Laravel project (modifies composer.json/lock — back up first)
cp ~/herd/hoopless_crm/composer.{json,lock} /tmp/backup-
cd ~/herd/hoopless_crm
composer config repositories.scip-php vcs https://github.com/davidrjenni/scip-php
composer require --dev "davidrjenni/scip-php:dev-main" -W --no-scripts

# Run
./vendor/bin/scip-php   # writes ./index.scip

# Inspect with sourcegraph scip CLI
go install github.com/sourcegraph/scip/cmd/scip@latest   # may need clone+build (replace directives)
scip stats index.scip
scip print --json index.scip > index.json
jq '.documents[] | select(.relative_path=="app/Models/Activity.php") | .occurrences | length' index.json

# Restore
cp /tmp/backup-composer.{json,lock} ~/herd/hoopless_crm/
cd ~/herd/hoopless_crm && composer install --quiet
```
