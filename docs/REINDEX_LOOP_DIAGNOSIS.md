# Reindex loop diagnosis — watcher self-trigger caused 4 kernel panics

**Date:** 2026-08-22. **Status: ALL THREE FIXES SHIPPED** (same day, scry-repo session).
Written by the Claude Code session that diagnosed the panics (dotfiles session); the
scry-repo session confirmed the root cause and landed the fixes. See the three
2026-08-22 entries in `docs/DECISIONS.md` for the decisions and trade-offs.

## Confirmed root cause (differs from the sketch below)

Reproduced deterministically in a sandboxed daemon: one `touch` of a `.go` file in
`idea-planning` produced 46 reindexes in 90 seconds. An instrumented watcher running
during the live loop captured the sustaining events: repeated **Chmod-only** events on
exactly two files — the repo's git-dirty files **larger than 32 KB**. Mechanism: some
step of every reindex runs a git operation that re-hashes dirty working-tree files;
git mmaps files ≥ 32 KB (`SMALL_FILE_SIZE`), the mmap read fires an APFS atime update
as a `NOTE_ATTRIB` kevent, fsnotify's kqueue backend delivers it as `Chmod`, and
`relevantEvent` accepted Chmod on source extensions. Not a create+delete temp file as
guessed in the sketch. Fixes: `relevantEvent` now requires a content op
(`Create|Write|Remove|Rename`); events arriving while a reindex is in flight are
deferred and only re-arm one catch-up if a deferred file still exists with mtime newer
than the build start. Note: a bare `touch` (attribute-only) no longer triggers a
reindex — verify with a real content write.

Fix 2 shipped as `sweepStaleIndexTrash` (synchronous in `Daemon.Run`). Fix 3's blame
of `scry mcp` was wrong — `lsof` showed MCP holds no graph DB; the daemon raced
itself (overlapping rebuild goroutines + `Get` reopening mid-build). Shipped as
`GraphRegistry.BeginBuild`/`EndBuild`; post-reindex graph rebuilds now succeed.

---

Original handoff below, kept for the panic forensics.

## What happened

Jeff's MacBook Pro kernel-panicked 4 times (watchdog timeout, ~every 36h of uptime).
Panic signature, identical in both captured reports:

```
panic: watchdog timeout: no checkins from watchdogd in 94 seconds
Compressor Info: 58% of compressed pages limit (OK) and 100% of segments limit (BAD) with 74 swapfiles
```

The daemon's file-watcher re-triggered a full reindex of `~/workspace/idea-planning`
**368,959 times** (count of `reindexing /Users/jeff/workspace/idea-planning` in
`~/.scry/scryd.log`). Each cycle rebuilt the BadgerDB from scratch. Sustained effect:

- ~11 MB/s of file-backed memory dirtied for 13+ hours (see
  `/Library/Logs/DiagnosticReports/scry_2026-08-22-034727_*.diag`: 549.76 GB over 47,413s,
  all in `badger.(*DB).startMemoryFlush`).
- Daemon RSS leaked ~4.6 MB/s while churning (measured: 5.25→5.46 GB over 45s at 3h uptime).
- After ~36h the VM compressor exhausted its segment limit → userspace froze
  (opendirectoryd userspace_watchdog_timeout spins in the minutes before each panic) →
  watchdogd starved → kernel watchdog panic.
- Side damage: 31,542 orphaned `index.db.old.*` dirs, **271 GB** (cleaned up 2026-08-22;
  `~/.scry` went 285 GB → 14 GB).

## The loop mechanism

1. `idea-planning` mixes Go + TS. `scip-typescript` was missing from the daemon's PATH, so
   every build ended `status: partial`. (Mitigated: installed it and symlinked
   `node` + `scip-typescript` into `/opt/homebrew/bin`, which IS on the launchd PATH.)
2. Observed in `scryd.log`: every completed reindex is immediately followed by
   `scry: reindexing <repo> (file change detected)` — with **zero user file changes**
   (verified: no file in the repo, including `.git/`, newer than boot). The reindex
   pipeline itself generates fsnotify events inside the watched repo that survive
   `relevantEvent` filtering, so each reindex schedules the next. The 2s
   `reindexCooldown` never breaks the cycle because a full reindex takes longer than 2s.
3. The loop only stopped tonight because the fd governor evicted the hot repos
   (`unwatched ... to free descriptors`). That is luck, not a fix — it looped 35 more
   times within 30 minutes of tonight's boot before the budget tripped.

## Fixes needed (in priority order)

### 1. Watcher must not react to its own reindex (the panic-causer)

`internal/daemon/watch.go` — `run()` event loop + `maybeReindex()`. Events that arrive
while a reindex of the same repo is in flight (and any pending debounce armed by them)
must not schedule another reindex. Sketch: set an in-flight flag when the reindex
goroutine starts; in the `fire` handler, drop the pending bit if the events all arrived
during the in-flight window. A user edit made mid-reindex is the corner case — decide
whether to eat it (next real event catches up) or re-arm once. Also worth identifying
WHICH path generates the self-events and adding it to `relevantEvent`/skip rules as
defense in depth — nothing in the repo looked modified afterward, so it's likely a
create+delete of a temp file with an indexed extension.

### 2. Startup sweep of stale trash

`Registry.SwapNext` (`internal/daemon/registry.go`) archives live → `index.db.old.<pid>.<ns>`
and callers delete it in a fire-and-forget goroutine (`methods.go` ~line 104,
`watch.go` phase 3). Those goroutines die with the daemon: ~8% of 369k cycles left
orphans (31,542 dirs / 271 GB). Fix: on daemon start (and/or per-repo open), delete any
`index.db.old.*` and stale `index.db.next` under the repo's layout dir. They are rotate-
then-delete garbage by design; nothing reads them.

### 3. Graph store lock conflict

Every post-reindex graph rebuild currently fails:

```
scry: graph rebuild ... failed: open graph store: open badger at ".../graph/index.db":
Cannot acquire directory lock ... Another process is using this Badger database.
```

The `scry mcp` process holds the graph DB open while the daemon's
`rebuildGraphAsync` (`internal/daemon/graph_methods.go`) tries to open it. Graph data is
silently going stale on every reindex. Needs single-owner access (daemon owns it, MCP
reads through the daemon) or an open/close-per-query discipline.

## Constraints

- The working tree already has ~700 uncommitted lines of fd-budget/governor work in
  these same files (`watch.go`, `registry.go`, `rlimit.go`, `methods.go`, `doctor.go`).
  Build on top of it; don't revert or reformat it.
- Repro/verification for fix 1: watch `~/.scry/scryd.log` while touching one `.go` file
  in a watched repo — exactly one `reindexing (file change detected)` line must appear,
  and no `index.db.old.*` dir may persist afterward. The old failure mode is a
  `reindexing` line every ~2-40s with zero user edits.
- Verification for fix 2: create fake `index.db.old.123.456` dirs under a repo layout,
  restart the daemon, confirm they're gone.
- `kern.num_files` sat at ~27k with the daemon holding ~15k fds; keep the governor's
  budget in mind when adding watches or opens.
