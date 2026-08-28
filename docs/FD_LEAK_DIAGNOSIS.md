# scry exhausts the system file table

Written 2026-08-18 from a live incident. Evidence is from a running machine,
not a reproduction — treat the measurements as facts and the hypotheses as
hypotheses.

> **Resolved 2026-08-19.** The measurements below all held up. Two of the
> hypotheses did not, and the section "What the fix turned out to be" at the
> bottom records what was actually wrong, with the reproductions. Read that
> section before acting on the two "where to look" hypotheses, which are
> superseded.

## Symptom

An unrelated Python process died at:

```
OSError: [Errno 23] Too many open files in system
```

Errno 23 is `ENFILE` — the **system-wide** file table, not the per-process
`EMFILE` (errno 24). One process had consumed enough of the global table that
other processes could no longer open anything.

## Measurements

Taken over about two hours while a separate build campaign was running:

```
kern.maxfiles                    491,520
system open files (peak seen)    429,407   (87%)
system open files (later)        167,747   (34%)
scry's share at peak             ~91% of all open files on the machine
```

Per-daemon, at one sampling:

```
pid 5371    fds     9,559    up 1d14h   owns ~/.scry/scryd.sock
pid 52843   fds   131,168    up 1h22m   also shows scryd.sock
pid 1921    fds   131,011    up 2h02m   (exited on its own)
pid 6593    fds   130,934    up 1h03m   (exited on its own)
```

Two observations that matter:

1. **The long-lived daemon is not the offender.** The one that has been up for
   a day and a half sits at ~9.5k fds, which looks like steady state. The
   young ones reach ~131k within an hour.
2. **They exit on their own.** Two of them terminated between samples. So this
   presents as recurring spikes, not a monotonic climb — which makes it easy
   to miss and easy to misattribute.

Breakdown of the worst offender's descriptors:

```
114,600  REG      regular files
 16,256  DIR      directories
    205  PIPE
    102  KQUEUE
```

By repository:

```
20,992  workspace/childscribe-mobile
20,887  workspace/childscribe-mobile-fluid
 4,853  workspace/Herd
 4,543  workspace/AI-VENTURE
 3,962  workspace/ai-engineering-from-scratch
 2,101  workspace/datasets
 1,812  workspace/cellsaviors
 1,494  workspace/childscribe-main
```

Almost no path is held twice (the most-repeated single path appeared 75
times). So this is **many distinct paths opened once each and not released**,
not the same handle opened repeatedly.

## Where to look

`internal/daemon/watch.go`. One `fsnotify.Watcher` per indexed repo,
recursively adding directories:

- `newRepoWatcher()` — creates the watcher, calls `addRepoToWatcher()`
- `addRepoToWatcher()` — "recursively adds every non-ignored directory under
  repoPath"
- `Unwatch()` — "stops watching one repo and releases its fsnotify resources"

`internal/daemon/rlimit.go` already anticipates fd pressure, and its comment
names the mechanism:

> fsnotify uses one fd per watched directory, and macOS' default soft cap of
> 256 is far too low for a real polyglot repo (advocates has ~1500 dirs).

`raiseNOFILE()` raises the process's **soft** NOFILE limit to its **hard**
limit.

## The contradiction worth starting from

`watch.go` defines:

```go
// maxWatchedDirs caps the per-repo directory count so a runaway repo can't
// exhaust the daemon's fd budget.
const maxWatchedDirs = 2048
```

Yet a single repo — `childscribe-mobile` — accounted for **20,992** open
descriptors in one daemon. That is roughly 10x the stated per-repo cap.

Reading `addRepoToWatcher` closes off one hypothesis and sharpens the rest.
The cap **is** enforced — `added >= maxWatchedDirs` returns `filepath.SkipDir`
— and the walk adds **directories only**:

```go
if !d.IsDir() {
    return nil
}
```

So the watcher cannot be the source of a single regular-file descriptor. That
splits the leak in two, and they are probably separate bugs:

### The 16,256 directories — watcher accumulation

```
16,256 DIR / 2,048 per watcher  =  7.94
```

That is very close to eight whole watchers' worth of directories in one
process. The cap holds *per watcher*, so nothing stops a process from holding
many watchers over the same repos. Start at whether a re-index,
re-registration, config reload, or repo re-add creates a second
`repoWatcher` without `Unwatch()`ing the first. `newRepoWatcher` closes `fsw`
only when `addRepoToWatcher` fails; every successful watcher's lifetime is
owned by whoever holds the struct.

### The 114,600 regular files — not the watcher at all

The watcher never opens a regular file, so these come from somewhere else,
and they are 88% of the leak. Look at the indexing and memory paths that walk
repos and open files:

```
internal/memory/distill/codex.go:81,140    os.Open(path)
internal/memory/distill/claude.go:68       os.Open(path)
internal/memory/sweep/sweep.go:216         os.ReadDir(root)
internal/schema/envdetect.go:61            os.Open(path)
internal/upgrade/upgrade.go:389            os.Open(src)
```

Each `os.Open` needs a matching `Close` on **every** return path, including
early error returns. A `defer` inside a loop body also does not run until the
enclosing function returns, which is a common way a walk accumulates handles
that all look correctly deferred.

`scry memory sweep` was observed running as its own process during the
incident, which makes the distill/sweep paths worth reading first.

## The second problem

**Multiple daemons run at once.** Four distinct `scry start --foreground`
processes were observed in a two-hour window, and at one sampling two of them
each showed a reference to `~/.scry/scryd.sock`. Whatever spawns them, each
one independently watches every indexed repo, multiplying the cost. Worth
determining whether a second daemon should be able to start at all when the
socket is already held.

## Why this is worse than it looks

`raiseNOFILE()` raises the per-process limit to the hard maximum. That is
correct for making the watcher work, and it also removes the only thing
bounding a single process's share of a **global** resource. One daemon can
therefore consume the system file table and take down unrelated software.

The failure then surfaces nowhere near its cause: `npm` cannot open a file, a
test runner fails for no stated reason, a build step misreads a file it just
wrote. During this incident the visible casualty was a small HTTP server that
had nothing to do with scry. Anything running unattended overnight will be
diagnosed wrong.

## How to observe it

```sh
sysctl kern.maxfiles
lsof | wc -l                                    # system-wide
lsof | awk '$1=="scry"' | wc -l                 # scry's share
for p in $(pgrep -f "scry start"); do
  echo "$(lsof -p $p 2>/dev/null | wc -l) $p"
done | sort -rn                                 # per-daemon, worst first

lsof -p <pid> | awk '{print $5}' | sort | uniq -c | sort -rn   # REG vs DIR
lsof -p <pid> | awk '{print $NF}' \
  | grep -oE "/Users/[^/]+/workspace/[^/]+" | sort | uniq -c | sort -rn
```

A daemon that has been up an hour and holds six figures of descriptors is
reproducing the bug.

## What is currently in place

Nothing in scry. A watchdog outside it — `~/scripts/fd-watchdog.sh` — polls
every five minutes and sends a Pushover alert when open files cross 70% of
`kern.maxfiles`, naming the worst scry pid. That is detection only. The leak is
unfixed.

## What NOT to conclude

- Do not assume the long-lived daemon is healthy because its count is low; it
  may simply watch fewer repos, or have been started before some repos were
  registered.
- Do not treat the exits as self-healing. The processes died; whether they
  crashed on their own fd exhaustion, were killed, or completed normally was
  not established.
- The per-repo counts are descriptors attributed to paths under that repo, not
  proof that the watcher for that repo opened them.


---

# What the fix turned out to be

Added 2026-08-19, after reproducing both faults in tests and fixing them.

## The split into "16k directories" and "114k regular files" was wrong

The reasoning was that `addRepoToWatcher` adds directories only, so the watcher
"cannot be the source of a single regular-file descriptor." That holds for
inotify. It does not hold here.

fsnotify's kqueue backend — macOS and the BSDs — cannot watch a directory as a
unit. To emit per-file events it opens a descriptor for **every entry inside
every watched directory**, in `watchDirectoryFiles`
(`backend_kqueue.go:570`). Adding one directory costs `1 + len(entries)`, not 1.

So the REG descriptors and the DIR descriptors are the same mechanism, and the
walk-and-open paths in `distill/`, `sweep/`, `envdetect.go` and `upgrade.go`
were not involved at all. Reproduction, now a regression test:

```
20 directories x 200 files  ->  4021 descriptors
```

20 directories is 1% of `maxWatchedDirs`. The cap was never a bound on
descriptors.

## There was no watcher accumulation

`16,256 / 2,048 = 7.94` was read as "eight whole watchers' worth". It was a
coincidence of summing many repos. Direct measurement, per repo, in the leaking
process:

```
2086  ai-engineering-from-scratch
1992  childscribe-mobile
1965  childscribe-mobile-fluid
 897  Herd
 ...
```

No repo holds more than one watcher's worth. The daemon simply watched 74
repos, because `bootstrapWatchers` watched every registered repo and there are
125 of them. The comparison that settles it — the two live daemons, same build:

```
pid 5371    10 repos    9,559 fds     (looked "healthy"; just older than most repos)
pid 27578   74 repos  133,605 fds
```

It is linear in repo count at roughly 1,700 descriptors per repo. The
long-lived daemon was not in steady state; it had simply bootstrapped before
most repos were registered.

## The second fault: closing a watcher released nothing

Not visible from `lsof`, and it only matters once you start evicting.

fsnotify v1.9.0's `kqueue.Close()` marks the watcher closed and *then* loops
calling `Remove` on each watched path. `remove()` opens with
`if w.isClosed() { return nil }`, so every one of those calls returns before
reaching `unix.Close(info.wd)`. Measured:

```
watcher holding 1155 descriptors  ->  Close()  ->  1 released
```

This is an upstream bug. The workaround is to remove the watches first, while
the watcher is still open. It is worth reporting upstream.

## Why the daemons multiplied

`AliveDaemon` treats "PID file names a live process, but the socket does not
answer" as a stale socket: it removes the socket and starts anyway, leaving the
old process running. That closes a loop. A daemon starved of descriptors cannot
`accept`, so it fails the ping, so the next start orphans it — while it keeps
every descriptor its watchers opened. That is the mechanism behind four
daemons in two hours and behind the "they exit on their own" spikes.

## What changed

- Watchers are bounded by a shared descriptor budget, charged the real
  per-platform cost, instead of by a directory count.
- Repos are watched lazily, most-recently-indexed first, with LRU eviction, so
  the repos being worked in hold the watches.
- Watches are removed before the fsnotify watcher is closed.
- `raiseNOFILE` targets a fixed 65536 rather than the hard limit, which macOS
  reports as unlimited.
- A governor samples actual descriptor use and evicts while it is over ceiling,
  because kqueue keeps opening descriptors after the initial Add.
- A daemon that is alive but not serving is retired instead of orphaned.

Measured on the same 74 repos that produced the incident:

```
before   ~131,000 descriptors,  53,323 of them unreleasable
after      14,239 descriptors,       0 leaked on close
```

Full reasoning is in `docs/DECISIONS.md` (three entries dated 2026-08-19).

## One thing this incident caused that looked unrelated

The Go toolchain in this workspace started failing with
`package strconv is not in std` on every build, including `hello world`. The
stdlib was intact; Go's GOROOT index cache had been written during descriptor
exhaustion and was corrupt. `go clean -cache` fixed it. This is exactly the
failure mode the "why this is worse than it looks" section predicted — the
damage surfaced nowhere near its cause, and looked like a broken Go install.
