package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Daemon ownership.
//
// Exactly one scry daemon may serve a ~/.scry at a time. Before this file
// existed the only guard was a PID file plus a liveness ping, and the
// takeover sequence (ping, SIGTERM the PID-file process, unlink the socket,
// bind, overwrite the PID file) was not atomic across concurrent starters —
// launchd's KeepAlive and client auto-spawn could both decide the daemon was
// down, each retire only the PID they could see, and leave earlier daemons
// alive but unaddressable. docs/DAEMON_SPLIT_BRAIN_DIAGNOSIS.md has the
// incident: three daemons, one on the socket, one on the UI port, one
// holding watchers.
//
// The fix is a process-lifetime flock on ~/.scry/scryd.lock, acquired before
// any takeover mutation and released only when the daemon exits. flock is
// released by the kernel when the holder dies, so "lock held" means "owner
// alive" without any PID-reuse ambiguity, and a successor cannot touch the
// socket or PID file until the incumbent has fully torn down. The lock file
// is never unlinked: unlinking a lock file lets two processes lock two
// different inodes at the same path.

// ErrAlreadyRunning is the sentinel matched by errors.Is when a start steps
// aside for a healthy incumbent. The concrete error is *AlreadyRunningError.
var ErrAlreadyRunning = errors.New("scry daemon already running")

// AlreadyRunningError reports the healthy incumbent a start deferred to.
type AlreadyRunningError struct {
	PID        int
	SocketPath string
}

func (e *AlreadyRunningError) Error() string {
	return fmt.Sprintf("scry daemon already running (pid %d, socket %s)", e.PID, e.SocketPath)
}

func (e *AlreadyRunningError) Is(target error) bool { return target == ErrAlreadyRunning }

// errLockHeld is returned by tryLockFile when another open file description
// holds the flock.
var errLockHeld = errors.New("ownership lock held")

// takeoverPolicy bounds how a starter treats an incumbent it cannot reach.
type takeoverPolicy struct {
	// StartupGrace is how long a lock holder that is not yet answering on
	// the socket is given before it is considered unresponsive. A fresh
	// daemon binds its socket within milliseconds of taking the lock; this
	// only needs to cover a slow machine, not a hung one.
	StartupGrace time.Duration
	// TermGrace is how long after SIGTERM the incumbent has to exit and
	// release the lock before escalation.
	TermGrace time.Duration
	// KillGrace is how long after SIGKILL to wait for the lock to be
	// released before giving up.
	KillGrace time.Duration
	// Logf receives every takeover decision. Never nil after normalize.
	Logf func(format string, args ...any)
}

// defaultTakeoverPolicy is what Daemon.Run uses. TermGrace exceeds the
// daemon's own DefaultShutdownGrace so a cleanly stopping incumbent is never
// escalated while it is still inside its shutdown window.
func defaultTakeoverPolicy() takeoverPolicy {
	return takeoverPolicy{
		StartupGrace: 3 * time.Second,
		TermGrace:    DefaultShutdownGrace + 5*time.Second,
		KillGrace:    3 * time.Second,
		Logf: func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "scry: "+format+"\n", args...)
		},
	}
}

func (p takeoverPolicy) normalize() takeoverPolicy {
	if p.Logf == nil {
		p.Logf = func(string, ...any) {}
	}
	return p
}

// ownership is the held single-instance lock. Release it exactly once, after
// every other piece of daemon state has been torn down.
type ownership struct {
	layout Layout
	f      *os.File
	once   sync.Once
}

// Release drops the lock. Idempotent.
func (o *ownership) Release() {
	o.once.Do(func() { _ = o.f.Close() })
}

// record writes the owner's PID into the lock file so a contender that finds
// the lock held can identify who to retire.
func (o *ownership) record() error {
	if err := o.f.Truncate(0); err != nil {
		return err
	}
	_, err := o.f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	return err
}

// tryLockFile opens path and takes a non-blocking exclusive flock on it.
// Returns errLockHeld if another open file description holds it.
func tryLockFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errLockHeld
		}
		return nil, fmt.Errorf("flock %s: %w", path, err)
	}
	return f, nil
}

// acquireOwnership makes this process the single daemon for layout, or
// returns why it cannot be:
//
//   - *AlreadyRunningError when a healthy daemon (lock holder or pre-lock
//     legacy daemon) answers on the socket — the caller should exit quietly;
//   - another error when an incumbent holds the lock but cannot be safely
//     retired (unknown identity, ignores SIGKILL).
//
// An incumbent that holds the lock but does not answer within StartupGrace is
// retired: SIGTERM, wait for the lock to be released, escalate to SIGKILL
// after TermGrace. The returned ownership is held until Release.
func acquireOwnership(layout Layout, p takeoverPolicy) (*ownership, error) {
	p = p.normalize()
	if err := os.MkdirAll(layout.Home, 0o755); err != nil {
		return nil, fmt.Errorf("ensure home: %w", err)
	}

	f, err := tryLockFile(layout.LockPath)
	if errors.Is(err, errLockHeld) {
		f, err = contendForLock(layout, p)
	}
	if err != nil {
		return nil, err
	}
	own := &ownership{layout: layout, f: f}

	// The lock is ours, so no lock-aware daemon is alive. A daemon from
	// before the lock existed may still be: it is named by the PID file and
	// holds no lock. Defer to it if it answers; retire it if it does not.
	if err := own.settleLegacyDaemon(p); err != nil {
		own.Release()
		return nil, err
	}
	if err := own.record(); err != nil {
		own.Release()
		return nil, fmt.Errorf("record owner pid: %w", err)
	}
	return own, nil
}

// contendForLock is the path taken when another process holds the lock.
// The holder is given StartupGrace to answer on the socket (it may still be
// binding); a healthy answer means we step aside. Otherwise the holder is
// retired and the lock waited for.
func contendForLock(layout Layout, p takeoverPolicy) (*os.File, error) {
	deadline := time.Now().Add(p.StartupGrace)
	for {
		if pingSocket(layout.SocketPath) {
			pid := readPIDFrom(layout.LockPath)
			if pid <= 0 {
				pid = readPIDFrom(layout.PIDPath)
			}
			return nil, &AlreadyRunningError{PID: pid, SocketPath: layout.SocketPath}
		}
		f, err := tryLockFile(layout.LockPath)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, errLockHeld) {
			return nil, err
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	owner := readPIDFrom(layout.LockPath)
	switch {
	case owner <= 0:
		return nil, fmt.Errorf("ownership lock %s is held by a process that recorded no pid and does not answer on %s; refusing to take over blindly",
			layout.LockPath, layout.SocketPath)
	case owner == os.Getpid():
		// An earlier Run in this same process is still winding down (tests
		// do this; production never does). Wait for it rather than signal
		// ourselves.
		p.Logf("ownership lock held by this process; waiting up to %v for release", p.TermGrace)
		if f := waitForLock(layout.LockPath, p.TermGrace); f != nil {
			return f, nil
		}
		return nil, fmt.Errorf("ownership lock %s still held by this process after %v", layout.LockPath, p.TermGrace)
	}

	p.Logf("retiring unresponsive daemon pid %d before taking over %s (lock %s held, no answer on socket within %v)",
		owner, layout.SocketPath, layout.LockPath, p.StartupGrace)
	return retireLockOwner(layout, p, owner)
}

// retireLockOwner signals pid and waits for the flock it holds to be
// released, escalating from SIGTERM to SIGKILL after TermGrace.
func retireLockOwner(layout Layout, p takeoverPolicy, pid int) (*os.File, error) {
	start := time.Now()
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return nil, fmt.Errorf("SIGTERM pid %d holding %s: %w", pid, layout.LockPath, err)
	}
	p.Logf("SIGTERM delivered to pid %d; waiting up to %v for it to exit and release ownership", pid, p.TermGrace)
	if f := waitForLock(layout.LockPath, p.TermGrace); f != nil {
		p.Logf("pid %d exited and released ownership after %v", pid, time.Since(start).Round(time.Millisecond))
		return f, nil
	}

	p.Logf("pid %d still holds ownership %v after SIGTERM; escalating to SIGKILL (%s)",
		pid, p.TermGrace, describeProcess(pid))
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return nil, fmt.Errorf("SIGKILL pid %d holding %s: %w", pid, layout.LockPath, err)
	}
	if f := waitForLock(layout.LockPath, p.KillGrace); f != nil {
		p.Logf("pid %d released ownership after SIGKILL (%v total)", pid, time.Since(start).Round(time.Millisecond))
		return f, nil
	}
	return nil, fmt.Errorf("pid %d still holds %s %v after SIGKILL; giving up", pid, layout.LockPath, p.KillGrace)
}

// settleLegacyDaemon handles a pre-lock daemon named by the PID file. Called
// with the lock held, so any live process the PID file names is either a
// legacy daemon or an unrelated process that reused the PID; the command
// name check tells them apart.
func (o *ownership) settleLegacyDaemon(p takeoverPolicy) error {
	pid := readPIDFrom(o.layout.PIDPath)
	if pid <= 0 || pid == os.Getpid() || !processAlive(pid) || !looksLikeScryDaemon(pid) {
		return nil
	}
	deadline := time.Now().Add(p.StartupGrace)
	for {
		if pingSocket(o.layout.SocketPath) {
			return &AlreadyRunningError{PID: pid, SocketPath: o.layout.SocketPath}
		}
		if !processAlive(pid) || !looksLikeScryDaemon(pid) {
			return nil
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	p.Logf("retiring unresponsive daemon pid %d before taking over %s (pre-lock daemon named by %s, no answer within %v)",
		pid, o.layout.SocketPath, o.layout.PIDPath, p.StartupGrace)
	start := time.Now()
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("SIGTERM legacy daemon pid %d: %w", pid, err)
	}
	p.Logf("SIGTERM delivered to pid %d; waiting up to %v for exit", pid, p.TermGrace)
	if waitForExit(pid, p.TermGrace) {
		p.Logf("pid %d exited after %v", pid, time.Since(start).Round(time.Millisecond))
		return nil
	}
	p.Logf("pid %d still alive %v after SIGTERM; escalating to SIGKILL (%s)", pid, p.TermGrace, describeProcess(pid))
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("SIGKILL legacy daemon pid %d: %w", pid, err)
	}
	if waitForExit(pid, p.KillGrace) {
		p.Logf("pid %d exited after SIGKILL (%v total)", pid, time.Since(start).Round(time.Millisecond))
		return nil
	}
	return fmt.Errorf("legacy daemon pid %d still alive %v after SIGKILL; giving up", pid, p.KillGrace)
}

// waitForLock polls tryLockFile until it succeeds or the budget expires.
func waitForLock(path string, budget time.Duration) *os.File {
	deadline := time.Now().Add(budget)
	for {
		f, err := tryLockFile(path)
		if err == nil {
			return f
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitForExit polls until pid is gone (or is a zombie awaiting reap, which
// holds no resources) or the budget expires.
func waitForExit(pid int, budget time.Duration) bool {
	deadline := time.Now().Add(budget)
	for {
		if !processAlive(pid) || !looksLikeScryDaemon(pid) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// looksLikeScryDaemon reports whether pid is a live (non-zombie) process
// running the same executable name as this one. It guards the legacy path
// against PID reuse: the PID file is the only thing naming that process,
// and PID files outlive crashes.
func looksLikeScryDaemon(pid int) bool {
	out, err := exec.Command("ps", "-o", "stat=,comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return false
	}
	fields := strings.Fields(string(out))
	if len(fields) < 2 || strings.HasPrefix(fields[0], "Z") {
		return false
	}
	self, err := os.Executable()
	if err != nil {
		return true
	}
	return filepath.Base(fields[len(fields)-1]) == filepath.Base(self)
}

// describeProcess is diagnostic context for an escalation log line.
func describeProcess(pid int) string {
	out, err := exec.Command("ps", "-o", "pid=,stat=,lstart=,command=", "-p", strconv.Itoa(pid)).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return "process not visible to ps"
	}
	return strings.TrimSpace(string(out))
}

// readPIDFrom parses a PID from a file; 0 if absent or malformed.
func readPIDFrom(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(string(bytesTrimSpace(b)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

// removeOwnedPIDFile unlinks the PID file only if it still names pid. With
// the ownership lock this cannot race a successor, but an exiting daemon
// must never be able to unlink a successor's file even if the lock is
// somehow bypassed.
func removeOwnedPIDFile(layout Layout, pid int) {
	if readPIDFrom(layout.PIDPath) != pid {
		return
	}
	_ = os.Remove(layout.PIDPath)
}
