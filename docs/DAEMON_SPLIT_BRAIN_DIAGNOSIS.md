# Scry daemon split-brain diagnosis

**Date:** 2026-08-28. **Status:** Code fix landed the same day (see
`docs/DECISIONS.md` "Daemon ownership" and `internal/daemon/owner.go`); the
regression tests below are in `internal/daemon/owner_test.go` and
`internal/doctor/instances_test.go`. Live recovery was then performed with the
fixed binary using the runbook in this file.

Evidence below came from the running MacBook, not a synthetic reproduction.
Treat the process, socket, port, lock, launchd, and log observations as facts.
The exact concurrent-start sequence is a source-backed inference that still
needs a deterministic regression test.

## Executive summary

Scry was not offline. It was split across three surviving
`scry start --foreground` processes:

| PID | Started | Role observed |
| --- | --- | --- |
| `43409` | 2026-08-28 08:31:34 | Canonical daemon reported by `scry status`; owned the current Unix socket and the memory BadgerDB |
| `51773` | 2026-08-27 23:03:46 | Orphan daemon; owned the live memory UI port `127.0.0.1:7279` |
| `51786` | 2026-08-27 23:03:47 | Orphan daemon; retained watchers and database handles |

This divided ownership caused the visible memory failure:

1. HTTP requests reached PID `51773` on port 7279.
2. Its UI handler lazily attempted to open `~/.scry/memory`.
3. PID `43409` already held Badger's exclusive directory lock.
4. The UI returned HTTP 500, making memory appear offline even though the
   canonical RPC daemon and memory queries were working.

At the same time, the canonical daemon's extractor was **dormant** because
its environment lacked a configured provider API key. That is separate from
the UI failure: recall/orient/status still worked, but daemon-side extraction
could not turn new episodes into graph facts.

The likely systemic cause is a non-atomic daemon takeover protocol combined
with two possible process owners: launchd `KeepAlive` and client/MCP
auto-spawn. A replacement sends SIGTERM to only the PID currently recorded in
the PID file, does not wait for it to exit, removes the socket pathname, and
binds a new listener. Concurrent replacements can therefore overwrite the PID
file and socket pathname while earlier processes remain alive and become
unaddressable orphans.

## User-visible symptoms

The permanent memory UI was listening but unhealthy:

```text
$ curl -D - http://127.0.0.1:7279/
HTTP/1.1 500 Internal Server Error

open badger at "/Users/jeff/.scry/memory": Cannot acquire directory lock on
"/Users/jeff/.scry/memory". Another process is using this Badger database.
err: resource temporarily unavailable
```

The core daemon was reachable:

```text
$ scry doctor
Summary: 17 passed, 3 warnings, 0 failed

$ scry memory status --pretty
{
  "episodes": 2609,
  "entities": 13748,
  "facts": 20456,
  "dormant": true,
  "cursors": 1321
}
```

`scry memory orient --cwd .` also completed successfully. Therefore:

- "daemon offline" is not an accurate description;
- "memory UI returns 500 because it is served by an orphan daemon" is
  accurate; and
- "canonical daemon extraction is dormant because its environment lacks a
  provider key" is independently accurate.

## Confirmed live evidence

### Three foreground daemons survived

```text
43409  PPID 1  /Users/jeff/go/bin/scry start --foreground
51773  PPID 1  /Users/jeff/go/bin/scry start --foreground
51786  PPID 1  /Users/jeff/go/bin/scry start --foreground
```

All three referenced the same pathname, `~/.scry/scryd.sock`, through
different Unix socket objects. Removing and rebinding a Unix socket pathname
does not close a listener that already owns the prior socket object, so this
is possible after a takeover race.

`scry status` reported PID `43409`, and `~/.scry/scryd.pid` contained `43409`.
The PID file modification time matched that process's start time.

### Socket, UI, and memory ownership were divided

- PID `43409` served the current `~/.scry/scryd.sock` RPC endpoint.
- PID `51773` was the only `scry` process listening on TCP port 7279.
- PID `43409` had `~/.scry/memory` open and held Badger's directory lock.
- A request handled by PID `51773` could not open the same store and returned
  HTTP 500.
- PID `51786` held repository database and watcher resources despite serving
  neither the current RPC socket nor the UI port.

The scheduled `scry memory sweep` process was active and had an established
provider HTTPS connection, but it was **not** the memory lock owner. Do not
blame or terminate the sweep as the primary fix for this incident.

### Launchd was crash-looping around an externally owned daemon

`launchctl print gui/501/com.jhoot.scryd` showed:

```text
state = spawn scheduled
runs = 1199
last exit code = 1
properties = keepalive | runatload
```

The LaunchAgent runs:

```text
/bin/zsh -lc 'source ~/.secrets.zsh 2>/dev/null;
  exec /Users/jeff/go/bin/scry start --foreground'
```

Its log repeated every ten seconds:

```text
scry: scry daemon already running (pid 43409,
socket /Users/jeff/.scry/scryd.sock)
```

The launchd job was therefore not the owner of canonical PID `43409`; it kept
starting, detecting the other daemon, exiting 1, and being restarted by
`KeepAlive`.

### Descriptor and database damage extended beyond the UI

Approximate `lsof` line counts at diagnosis time were:

```text
PID 43409   16,372
PID 51773   14,355
PID 51786   14,304
```

The shared daemon log also contained repeated Badger lock failures for repo
indexes and graph indexes, especially the `docket` store. Multiple daemons
were independently retaining watchers and stores, recreating the multiplier
described in `docs/FD_LEAK_DIAGNOSIS.md`.

## Log chronology

The daemon log records the takeover failure mode before the current state:

```text
scry: retiring unresponsive daemon pid 80406 before taking over .../scryd.sock
scry: retiring unresponsive daemon pid 80406 before taking over .../scryd.sock
scry: listen unix .../scryd.sock: bind: address already in use
scry: retiring unresponsive daemon pid 51663 before taking over .../scryd.sock
scry: retiring unresponsive daemon pid 51709 before taking over .../scryd.sock
...
scry: retiring unresponsive daemon pid 51772 before taking over .../scryd.sock
scry: retiring unresponsive daemon pid 51774 before taking over .../scryd.sock
...
```

The rapid succession and skipped surviving PIDs are important. Once a later
starter overwrites the PID file, an earlier living daemon is no longer the PID
that `retireUnresponsiveDaemon` can discover. It becomes an orphan even though
it still owns descriptors, stores, and possibly port 7279.

When canonical PID `43409` started, the log said:

```text
2026/08/28 08:31:34 memory: extraction DORMANT — no API key in the environment.
2026/08/28 08:31:34 memory-ui: listen 127.0.0.1:7279: bind: address already in use
  (live memory UI disabled this run)
```

That explains both present conditions directly:

- the canonical process could not claim the UI because orphan PID `51773`
  already owned it; and
- the canonical process did not inherit the secret-bearing environment that
  the LaunchAgent was designed to supply.

## Source-level failure mechanism

The relevant path is `internal/daemon/daemon.go` in `Daemon.Run`:

1. `aliveDaemonPID` checks the PID file and pings the current socket.
2. If the ping fails, `retireUnresponsiveDaemon` sends SIGTERM to the PID from
   the PID file.
3. It does not wait for process exit, verify shutdown, or escalate after a
   deadline.
4. Startup immediately removes `scryd.sock`.
5. It binds a new socket and overwrites `scryd.pid`.

There is no separate, process-lifetime single-instance lock around that
check/retire/remove/bind/write sequence. The check and takeover are therefore
not atomic across concurrent starters.

The memory UI in `internal/daemon/memory_ui.go` is intentionally best-effort:
a port conflict disables the UI for that daemon without stopping RPC startup.
That is normally reasonable, but it makes split-brain ownership persistent:
the replacement can become canonical while an orphan continues to own the UI.

The UI handler calls `d.memoryExport()`, which calls the serving daemon's lazy
`d.memoryStore()`. It does not proxy the export through whichever process owns
the canonical RPC socket. Consequently, a UI served by an orphan necessarily
contends for Badger's exclusive memory-store lock.

## Likely trigger

The evidence supports, but does not yet deterministically prove, this trigger:

1. An existing daemon becomes slow or temporarily fails the short liveness
   ping under descriptor/index pressure.
2. More than one actor attempts recovery. Current candidates are launchd
   `KeepAlive` and MCP/client auto-spawn.
3. Two starters both observe the daemon as unavailable before either takeover
   has stabilized.
4. Each signals only the PID visible at that instant, removes/rebinds the same
   socket pathname, and writes its own PID.
5. One or more earlier processes survive but disappear from the PID-file-based
   control path.

The nearly simultaneous start times for PIDs `51773` and `51786` support a
concurrent-start race. A regression test should establish the exact
interleaving rather than relying only on this inference.

## Safe live recovery runbook

Do not start by deleting Badger lock files or `scryd.sock`. The locks are held
by live processes; deleting lock files can permit unsafe concurrent database
access, and deleting the socket alone does not close old Unix listeners.

Also do not use a broad `pkill scry`: many `scry mcp` processes are legitimate
stdio servers owned by active Codex/Claude sessions.

### 1. Stop automatic daemon restarts first

Temporarily boot out only the daemon LaunchAgent so `KeepAlive` cannot race the
cleanup:

```sh
launchctl bootout "gui/$(id -u)/com.jhoot.scryd"
```

If service-target bootout is unsupported in the current launchctl state, use
the explicit plist path:

```sh
launchctl bootout "gui/$(id -u)" \
  "$HOME/Library/LaunchAgents/com.jhoot.scryd.plist"
```

### 2. Re-resolve exact foreground daemon PIDs

The incident PIDs may be stale by repair time. Resolve and inspect the exact
targets before signaling:

```sh
pgrep -af '/Users/jeff/go/bin/scry start --foreground'
ps -p <comma-separated-pids> -o pid=,ppid=,lstart=,command=
```

Exclude every `scry mcp`, `scry memory sweep`, and unrelated command.

### 3. Stop the canonical daemon, then exact orphans

Try the supported stop command first, then signal only validated foreground
daemon PIDs that remain:

```sh
scry stop
kill -TERM <validated-daemon-pid> [...]
```

Wait and recheck. If a validated foreground daemon ignores TERM after a
reasonable shutdown window, collect a stack/sample before considering KILL;
that shutdown hang is useful evidence for the code fix.

### 4. Verify the machine is clean before relaunch

All of these should show no foreground daemon owner:

```sh
pgrep -af '/Users/jeff/go/bin/scry start --foreground'
lsof -nP -iTCP:7279 -sTCP:LISTEN
lsof -nP -U | rg '/Users/jeff/.scry/scryd.sock'
lsof -nP +D "$HOME/.scry/memory" | head
```

Only after no live process owns it, remove a genuinely stale socket or PID
file if the supported startup path cannot do so itself.

### 5. Start exactly one authority with secrets

Bootstrap the secret-sourcing LaunchAgent once:

```sh
launchctl bootstrap "gui/$(id -u)" \
  "$HOME/Library/LaunchAgents/com.jhoot.scryd.plist"
```

Do not manually run `scry start` at the same time.

### 6. Verify repaired live state

Acceptance checks:

```sh
pgrep -af '/Users/jeff/go/bin/scry start --foreground'
scry doctor
scry memory status --pretty
curl -fsS http://127.0.0.1:7279/ >/dev/null
curl -fsS http://127.0.0.1:7279/data.json | jq '.entities | length'
launchctl print "gui/$(id -u)/com.jhoot.scryd"
```

Expected:

- exactly one foreground daemon;
- that same process owns `scryd.sock`, port 7279, and the memory DB;
- memory UI returns 200;
- memory status is not dormant and reports the configured model chain;
- launchd shows the service running instead of repeatedly exiting; and
- daemon/log descriptor counts remain bounded.

## Required code fix

### 1. Add a true process-lifetime single-instance lock

Use a dedicated lock file, separate from Badger and the Unix socket, acquired
before any takeover mutation and held for the daemon's entire lifetime. The
lock must cover the full check/retire/wait/remove/bind/write sequence.

The exact design must handle an alive-but-unresponsive lock owner. A safe
protocol is:

1. Try to acquire the daemon ownership lock.
2. If held, ping the canonical socket.
3. If healthy, exit successfully without starting another daemon.
4. If unhealthy, identify and TERM the verified lock owner.
5. Wait with a deadline for both process exit and lock release.
6. Capture diagnostics and explicitly decide whether to escalate.
7. Only after owning the lock may a process remove stale runtime files and
   bind the canonical socket/UI.

A plain PID file is not sufficient synchronization and is vulnerable to PID
reuse. Record process identity/start time if it remains part of takeover.

### 2. Choose one daemon-start authority

Launchd `KeepAlive` and detached client auto-spawn should not independently
race to own the same service. Recommended macOS behavior:

- when the LaunchAgent is installed, clients ask launchd to start/kickstart
  it and then retry the socket;
- clients do not spawn a second detached foreground daemon themselves; and
- direct auto-spawn remains only as a fallback when no service manager is
  installed.

This also ensures the daemon consistently inherits the secret-sourcing
environment instead of becoming dormant depending on who happened to win.

### 3. Make shutdown and takeover observable

`retireUnresponsiveDaemon` currently ignores the result of `Signal` and does
not wait. It should log:

- why the liveness ping failed;
- the target's verified identity;
- TERM delivery success/failure;
- time to exit and release ownership;
- timeout/escalation; and
- the new daemon's successful acquisition of socket, UI, and memory store.

### 4. Prevent stale owners from removing a successor's runtime files

Deferred unconditional removal of shared `scryd.pid` and `scryd.sock` is
unsafe if a successor can replace those paths before the old process exits.
Cleanup should verify that the file/socket still belongs to the exiting
process, or the single-instance lock must make replacement-before-exit
impossible.

### 5. Treat UI ownership as a health invariant

The UI may remain best-effort for user-selected port conflicts, but a default
port conflict with another `scry start --foreground` process should be a loud
daemon-health failure. `scry doctor` should report when:

- the canonical socket PID differs from the 7279 listener PID;
- more than one foreground daemon exists;
- the UI returns non-200; or
- the UI process cannot open the memory store.

## Regression tests / done bar

The fix is not complete until fresh tests prove:

1. **Concurrent start:** launch many daemon starters simultaneously; exactly
   one process survives and owns socket, PID file, UI, and memory DB.
2. **Healthy incumbent:** a second start exits cleanly without mutating the
   incumbent's socket or PID file.
3. **Slow ping:** a temporarily slow but live daemon does not get orphaned by
   overlapping takeovers.
4. **Unresponsive incumbent:** replacement waits for verified shutdown and
   lock release before binding.
5. **TERM timeout:** a process that ignores TERM cannot be silently orphaned;
   timeout and escalation behavior are deterministic and tested.
6. **Old-process cleanup:** an exiting old process cannot unlink a successor's
   socket or PID file.
7. **Launchd + client race:** simultaneous service-manager and client start
   requests converge on one process.
8. **Environment inheritance:** the winning daemon has the configured memory
   providers and `scry memory status` is not dormant.
9. **UI/store unity:** an HTTP export and an RPC memory query are served from
   the same daemon-owned store without a Badger lock error.
10. **Resource bound:** failed contenders release all watchers, stores,
    sockets, and file descriptors promptly.

Live acceptance should then run for at least several launchd/MCP reconnect
cycles with exactly one foreground daemon and no repeated "already running",
"retiring unresponsive daemon", port-7279 conflict, or Badger directory-lock
errors in new log output.

## Related files

- `internal/daemon/daemon.go` — liveness check, takeover, socket/PID ownership,
  signal handling, shutdown
- `internal/daemon/memory_ui.go` — best-effort port binding and per-process UI
  handler
- `internal/daemon/memory_methods.go` — lazy memory-store ownership/export
- `cmd/scry/start.go` — foreground vs detached startup
- `cmd/scry/mcp.go` and client dial/spawn paths — auto-spawn behavior
- `docs/FD_LEAK_DIAGNOSIS.md` — prior observation that multiple daemons
  multiply descriptor pressure
- `docs/REINDEX_LOOP_DIAGNOSIS.md` — related Badger/reindex failure history

## Worktree warning

At diagnosis time the scry worktree already contained substantial unrelated
modified and untracked work, including changes in `internal/daemon/daemon.go`.
Preserve it. Do not reset, revert, or broadly reformat the tree while applying
this fix.
