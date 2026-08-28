# fd leak — findings and work-in-progress handoff

Written 2026-08-19. Companion to `docs/FD_LEAK_DIAGNOSIS.md` (the original
incident write-up, from another session).

**Status: work paused mid-stream at the user's request.** The code changes
described below are written, building, and passing tests, but they are
**uncommitted in the working tree** and have **not** been installed or run as a
real daemon. Nothing about the running system was changed. See "State of the
tree" and "What was deliberately not done".

---

## Summary

The incident had **two independent faults**, not the two the original write-up
proposed. Both are reproduced in tests.

1. **The per-repo cap never bounded file descriptors.** On macOS/BSD, fsnotify
   opens one descriptor per *entry* inside every watched directory, not one per
   directory. Combined with watching all 125 registered repos at startup, one
   daemon reached ~131,000 descriptors.
2. **Closing a watcher released nothing.** fsnotify v1.9.0's kqueue `Close()`
   leaks every descriptor it holds. This is an upstream bug.

A third, contributing mechanism: a daemon starved of descriptors fails its own
liveness ping, so the next daemon start orphans it while it keeps every
descriptor. That is the loop behind "four daemons in two hours".

---

## Corrections to the original diagnosis

The measurements in `FD_LEAK_DIAGNOSIS.md` all held up. Two hypotheses did not.

### "The watcher cannot be the source of a single regular-file descriptor" — wrong

The reasoning was that `addRepoToWatcher` adds directories only. True, and it
holds for inotify. It does not hold for kqueue.

fsnotify's kqueue backend cannot watch a directory as a unit. To emit per-file
events, `watchDirectoryFiles` (`backend_kqueue.go:570`) opens a descriptor for
**every entry inside every watched directory**. Adding one directory costs
`1 + len(entries)`.

So the 114,600 REG and 16,256 DIR descriptors are the *same* mechanism. The
walk-and-open paths the write-up pointed at — `memory/distill/codex.go`,
`distill/claude.go`, `sweep/sweep.go`, `schema/envdetect.go`,
`upgrade/upgrade.go` — were **not involved**. I read them; they are not leaking.

Reproduced (now a regression test, `TestWatchRepoStaysWithinFDBudget`):

```
20 directories x 200 files  ->  4021 descriptors
```

20 directories is 1% of `maxWatchedDirs = 2048`.

### "16,256 / 2,048 = 7.94, so eight watchers accumulated" — wrong

That ratio was a coincidence of summing across many repos. Per-repo counts in
the leaking process show no repo holding more than one watcher's worth:

```
2086  ai-engineering-from-scratch
1992  childscribe-mobile
1965  childscribe-mobile-fluid
 897  Herd
 589  AI-VENTURE
 ...
```

The decisive comparison — the two live daemons, same build:

```
pid 5371    10 repos    9,559 fds     up 1d14h
pid 27578   74 repos  133,605 fds     up ~1h
```

Linear in repo count, ~1,700 descriptors per repo. `bootstrapWatchers` watched
every registered repo, and there are 125.

**The long-lived daemon was never healthy.** It had simply bootstrapped before
most repos were registered. The write-up's own "what NOT to conclude" section
warned about exactly this, and it was right to.

### Note on the one repo over cap

`ai-engineering-from-scratch` at 2086 > 2048 comes from the incremental
`fsw.Add` in the event loop, which had no cap check. Minor, but real.

---

## The second fault: `fsnotify.Close()` releases nothing

Not visible from `lsof`, and it only becomes load-bearing once you start
evicting watchers.

`kqueue.Close()` marks the watcher closed via `shared.close()`, and *then*
loops calling `Remove` on each watched path. But `remove()` opens with:

```go
if w.isClosed() { return nil }
```

...which is now true. Every call returns before reaching `unix.Close(info.wd)`.
Only the close-pipe descriptor is freed. Measured:

```
watcher holding 1155 descriptors  ->  Close()  ->  1 released
```

This matters enormously for any fix based on eviction: eviction would free the
*accounting* but not the descriptors, so the daemon would over-commit and drift
upward every time it evicted — strictly worse than not evicting at all. I only
caught it because the first end-to-end measurement showed 53,430 real
descriptors against 16,361 reserved.

**This is an upstream bug in fsnotify v1.9.0 and is worth reporting.** The
workaround is to `Remove` every path *before* `Close`, which takes the working
path through `remove()`.

---

## The daemon-multiplication loop

`AliveDaemon` treats "PID file names a live process, but the socket does not
answer" as a stale socket: it removes the socket and starts anyway, leaving the
old process running forever.

That closes a self-reinforcing loop:

```
daemon starved of descriptors -> cannot accept() -> fails liveness ping
  -> next start declares it stale, takes the socket, orphans it
  -> orphan keeps ~131k descriptors -> machine starves further
```

Evidence: pid 5371 held a unix socket entry for `~/.scry/scryd.sock` at a
*different inode* (`0xbbc1...`) than the live daemon's (`0x438f...`), with the
PID file naming the live one. It was an orphan listening on an unlinked inode.

---

## Changes written (uncommitted)

Diff: ~400 lines across 5 modified files + 4 new files. Scope was confirmed
with the user before starting (they chose budget + LRU eviction over
budget-only).

| File | Change |
|---|---|
| `internal/daemon/watchbudget.go` *(new)* | `fdBudget` (shared reserve/release ceiling) + `processFDCount` via bounded `F_GETFD` scan |
| `internal/daemon/watchcost_kqueue.go` *(new)* | darwin/BSD: `watchDirFDCost = 1 + len(entries)` |
| `internal/daemon/watchcost_other.go` *(new)* | linux/windows: `watchDirFDCost = 1` |
| `internal/daemon/watch.go` | Budget-charged walk; LRU eviction; `Touch`/`ClaimOnDemand`/`HasBudgetFor`; capped incremental add; `closeWatcher` workaround; fd governor |
| `internal/daemon/rlimit.go` | `raiseNOFILE` targets fixed 65536 instead of hard limit; budget sizing |
| `internal/daemon/bootstrap.go` | Lazy, most-recently-indexed-first, stops at budget |
| `internal/daemon/registry.go` | `OnAccess` hook, called outside the lock |
| `internal/daemon/daemon.go` | Wires on-demand watch, governor, budget sizing after `raiseNOFILE`; `retireUnresponsiveDaemon` |
| `internal/daemon/watch_fd_test.go` *(new)* | fd regression tests |
| `internal/daemon/watchbudget_test.go` *(new)* | budget/LRU unit tests |

Design notes, with reasoning and what would change our minds, are in
`docs/DECISIONS.md` — three entries dated 2026-08-19.

### Why a governor as well as a budget

The budget charges what a directory costs *at Add time*. kqueue keeps opening
descriptors afterward: writing into a watched directory makes `dirChange` walk
it and start watching every entry that has appeared since. A build campaign
creating files across watched trees therefore grows the set past whatever the
walk reserved. The governor samples actual process descriptor use every 30s and
evicts LRU watchers while it exceeds half the soft NOFILE limit.

---

## Verification performed

```
go build ./...      OK
go vet ./...        clean
go test ./...       all packages pass
go test -race ./internal/daemon/   pass
```

End-to-end, against the **same 74 real repos** that produced the incident
(watcher exercised directly; no indexing, no daemon started):

```
                     before        after
descriptors          ~131,000      14,239
leaked on close       53,323            0
watchers live            n/a           43 (of 74 attempted, rest on demand)
```

Cost-model accuracy on a real repo: predicted 1328, actual 1152 (ratio 0.87 —
errs conservative, which is the safe direction).

---

## State of the tree

- All changes are **uncommitted**. `git status` shows 5 modified, 4 new source
  files, plus `docs/FD_LEAK_DIAGNOSIS.md` (updated) and this file.
- `internal/daemon/git_methods.go` and `schema_methods.go` show up in
  `gofmt -l`. That is **pre-existing**, not from this work — they are not in
  the diff.
- A test binary was built to the session scratchpad only.

## What was deliberately not done

*(Superseded 2026-08-19, second session: the fix is now installed and running
as the live daemon — see "Open items" above.)*

A peer session (`dotfiles-6f`) running an autonomous build campaign asked for
two constraints, both honoured:

1. **`~/go/bin/scry` was not touched.** Still dated 14 Aug. All builds went to
   the scratchpad.
2. **No daemon was restarted or killed.** The fix has therefore never run as a
   live daemon — that is the main untested surface.

I had drafted a reply to that session correcting their diagnosis (they believed
the leak was in the distill/sweep paths, which it is not) but did not send it.
Worth sending if this work resumes, so they do not chase the wrong code.

## Open items if this resumes

Resolved 2026-08-19 (second session):

1. ~~**Run it as a real daemon.**~~ Done. Fixed build installed to
   `~/go/bin/scry`, all stray daemons retired, new daemon holding ~14.7k fds
   against a 16384 budget with the same 125 repos registered. System open
   files dropped from 218k to 33k on restart.
2. ~~**Report the fsnotify `Close()` bug upstream.**~~ Unnecessary — already
   reported (fsnotify/fsnotify#732) and fixed in v1.10.0 ("kqueue: drop
   watches directly in Close()", PR #740). Upgraded to v1.10.1 and
   `closeWatcher` is a plain `Close()` again; the regression test guards it.
3. ~~**Latent bug, untouched:**~~ Fixed. The incremental add in
   `repoWatcher.run` sat behind `relevantEvent`, which drops directory names
   (no source extension), so newly created directories were never watched.
   Directory creation is now handled before the filter, via `watchNewDir`,
   which walks the new tree (a checkout or `cp -r` creates a populated tree,
   not one event per level) under the same skip rules and budget.
4. **Tune the numbers.** Budget is 1/4 of soft NOFILE clamped to 2048–16384;
   governor ceiling is 1/2 of soft NOFILE; NOFILE target is 65536. Reasonable,
   not empirically tuned. `scry status` now reports `watch` (watchers,
   budget_used/budget_cap, process_fds) so this can be tuned from data.
5. ~~Surface watch coverage~~ Done: `watched` per repo in `scry status`, a
   `watch` summary block, and a `daemon.watch` check in `scry doctor` that
   warns when the budget is >90% used.

---

## Unrelated damage this incident caused

The Go toolchain in this workspace began failing every build, including
`hello world`, with `package strconv is not in std`. The stdlib was intact —
Go's GOROOT index cache had been written during descriptor exhaustion and was
corrupt. **`go clean -cache` fixed it.**

This cost real time and looked exactly like a broken Homebrew Go install. It is
the failure mode the original write-up predicted: the damage surfaced nowhere
near its cause. Worth remembering if other tools on this machine start behaving
strangely — suspect their caches.

## How to re-measure

```sh
sysctl kern.maxfiles kern.num_files
for p in $(pgrep -f "scry start"); do echo "$(lsof -p $p 2>/dev/null | wc -l) $p"; done | sort -rn
lsof -p <pid> | awk '{print $5}' | sort | uniq -c | sort -rn          # REG vs DIR
lsof -p <pid> | awk '{print $NF}' \
  | grep -oE "/Users/[^/]+/workspace/[^/]+" | sort | uniq -c | sort -rn
```

The fd-cost reproduction, without any of the above:

```sh
go test ./internal/daemon/ -run TestWatchRepoStaysWithinFDBudget -v
go test ./internal/daemon/ -run TestCloseWatcherReleasesDescriptors -v
```
