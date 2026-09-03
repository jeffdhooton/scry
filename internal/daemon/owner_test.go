package daemon

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestMain doubles as the entry point for helper daemons: when
// SCRY_DAEMON_TEST_HELPER is set, the test binary plays the role of another
// scry process (a lock owner, a legacy daemon, ...) instead of running tests.
// Takeover has to be exercised across real processes because it signals the
// incumbent — signalling ourselves would end the test run.
func TestMain(m *testing.M) {
	if mode := os.Getenv("SCRY_DAEMON_TEST_HELPER"); mode != "" {
		runOwnerTestHelper(mode)
		return
	}
	os.Exit(m.Run())
}

// helperKeep pins listeners/locks the helper opens so the GC finalizer on
// net.Conn/os.File cannot close them behind our back while we block.
var helperKeep []any

func runOwnerTestHelper(mode string) {
	layout := LayoutFor(os.Getenv("SCRY_TEST_HOME"))
	if os.Getenv("SCRY_TEST_IGNORE_TERM") == "1" {
		signal.Ignore(syscall.SIGTERM)
	}
	listen := func() {
		ln, err := net.Listen("unix", layout.SocketPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "helper listen:", err)
			os.Exit(2)
		}
		helperKeep = append(helperKeep, ln)
	}
	switch mode {
	case "owner":
		// Mimics Daemon.Run: take the ownership lock, publish the PID file,
		// bind the socket (immediately, after a delay, or never).
		own, err := acquireOwnership(layout, takeoverPolicy{
			StartupGrace: time.Second, TermGrace: time.Second, KillGrace: time.Second,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "helper acquire:", err)
			os.Exit(2)
		}
		helperKeep = append(helperKeep, own)
		_ = os.WriteFile(layout.PIDPath, []byte(strconv.Itoa(os.Getpid())), 0o644)
		switch {
		case os.Getenv("SCRY_TEST_NO_BIND") == "1":
		case os.Getenv("SCRY_TEST_BIND_DELAY") != "":
			delay, _ := time.ParseDuration(os.Getenv("SCRY_TEST_BIND_DELAY"))
			go func() {
				time.Sleep(delay)
				listen()
			}()
		default:
			listen()
		}
	case "rawlock":
		// Holds the flock without ever recording a PID: an owner whose
		// identity a contender cannot verify.
		f, err := tryLockFile(layout.LockPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "helper rawlock:", err)
			os.Exit(2)
		}
		helperKeep = append(helperKeep, f)
	case "legacy":
		// A pre-lock daemon: alive, named by the PID file, holding no lock.
		_ = os.WriteFile(layout.PIDPath, []byte(strconv.Itoa(os.Getpid())), 0o644)
		if os.Getenv("SCRY_TEST_NO_BIND") != "1" {
			listen()
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown helper mode", mode)
		os.Exit(2)
	}
	fmt.Println("ready")
	// Not select{}: with no other goroutine the runtime reports a deadlock
	// and exits. Sleeping keeps the default SIGTERM disposition (die).
	for {
		time.Sleep(time.Hour)
	}
}

type helperProc struct {
	cmd    *exec.Cmd
	waited bool
}

// startHelper launches the test binary in helper mode and blocks until it
// reports "ready". The process is killed at test cleanup if still alive.
func startHelper(t *testing.T, mode, home string, extraEnv ...string) *helperProc {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		"SCRY_DAEMON_TEST_HELPER="+mode,
		"SCRY_TEST_HOME="+home,
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper %s: %v", mode, err)
	}
	h := &helperProc{cmd: cmd}
	t.Cleanup(func() {
		if !h.waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "ready" {
		t.Fatalf("helper %s did not report ready: %q, %v", mode, line, err)
	}
	return h
}

func (h *helperProc) pid() int { return h.cmd.Process.Pid }

// wait blocks for the helper to exit and returns the signal that ended it
// (0 if it exited normally).
func (h *helperProc) wait(t *testing.T) syscall.Signal {
	t.Helper()
	done := make(chan struct{})
	go func() {
		_ = h.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatalf("helper pid %d did not exit", h.pid())
	}
	h.waited = true
	ws, ok := h.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if ok && ws.Signaled() {
		return ws.Signal()
	}
	return 0
}

// shortTempHome returns a scry home under /tmp rather than t.TempDir():
// Unix socket paths are capped at 104 bytes on macOS (108 on Linux) and the
// test-name-derived TempDir path blows straight through that.
func shortTempHome(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "scryd-")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func testPolicy(logs *strings.Builder) takeoverPolicy {
	var mu sync.Mutex
	return takeoverPolicy{
		StartupGrace: 300 * time.Millisecond,
		TermGrace:    3 * time.Second,
		KillGrace:    3 * time.Second,
		Logf: func(format string, args ...any) {
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprintf(logs, format+"\n", args...)
		},
	}
}

func lockIsFree(t *testing.T, layout Layout) bool {
	t.Helper()
	f, err := tryLockFile(layout.LockPath)
	if err != nil {
		if errors.Is(err, errLockHeld) {
			return false
		}
		t.Fatalf("tryLockFile: %v", err)
	}
	_ = f.Close()
	return true
}

func socketIdentity(t *testing.T, path string) string {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("no Stat_t")
	}
	return fmt.Sprintf("%d:%d", sys.Dev, sys.Ino)
}

func TestAcquireOwnershipFreshHomeRecordsOwnerPID(t *testing.T) {
	layout := LayoutFor(shortTempHome(t))
	var logs strings.Builder
	own, err := acquireOwnership(layout, testPolicy(&logs))
	if err != nil {
		t.Fatalf("acquireOwnership: %v", err)
	}
	defer own.Release()

	b, err := os.ReadFile(layout.LockPath)
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	if got := strings.TrimSpace(string(b)); got != strconv.Itoa(os.Getpid()) {
		t.Errorf("lock file records pid %q, want %d", got, os.Getpid())
	}
	if lockIsFree(t, layout) {
		t.Error("lock is free while ownership is held")
	}
}

func TestAcquireOwnershipReleaseAllowsSuccessor(t *testing.T) {
	layout := LayoutFor(shortTempHome(t))
	var logs strings.Builder
	first, err := acquireOwnership(layout, testPolicy(&logs))
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	first.Release()
	second, err := acquireOwnership(layout, testPolicy(&logs))
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	second.Release()
	if !lockIsFree(t, layout) {
		t.Error("lock still held after Release")
	}
}

// Done bar #1: many simultaneous starters, exactly one owner. Each winner
// binds the socket right away (as Run does), so the losers observe a healthy
// incumbent and step aside instead of retiring it.
func TestAcquireOwnershipConcurrentStartersOneWinner(t *testing.T) {
	layout := LayoutFor(shortTempHome(t))
	const starters = 16
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners int
		stepped int
		other   []error
	)
	for i := 0; i < starters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var logs strings.Builder
			p := testPolicy(&logs)
			p.StartupGrace = 2 * time.Second
			own, err := acquireOwnership(layout, p)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ln, lerr := net.Listen("unix", layout.SocketPath)
				if lerr != nil {
					other = append(other, fmt.Errorf("winner listen: %w", lerr))
					own.Release()
					return
				}
				helperKeep = append(helperKeep, ln, own)
				winners++
			case errors.Is(err, ErrAlreadyRunning):
				stepped++
			default:
				other = append(other, err)
			}
		}()
	}
	wg.Wait()
	if winners != 1 {
		t.Errorf("winners = %d, want exactly 1", winners)
	}
	if stepped != starters-1 {
		t.Errorf("starters that stepped aside = %d, want %d", stepped, starters-1)
	}
	if len(other) != 0 {
		t.Errorf("unexpected errors: %v", other)
	}
}

// Done bar #2: a healthy incumbent's PID file and socket are left untouched.
func TestAcquireOwnershipHealthyIncumbentUntouched(t *testing.T) {
	home := shortTempHome(t)
	layout := LayoutFor(home)
	h := startHelper(t, "owner", home)
	pidBefore, _ := os.ReadFile(layout.PIDPath)
	sockBefore := socketIdentity(t, layout.SocketPath)

	var logs strings.Builder
	own, err := acquireOwnership(layout, testPolicy(&logs))
	if err == nil {
		own.Release()
		t.Fatal("acquired ownership over a healthy incumbent")
	}
	var are *AlreadyRunningError
	if !errors.As(err, &are) || are.PID != h.pid() {
		t.Fatalf("err = %v, want AlreadyRunningError for pid %d", err, h.pid())
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("err does not match ErrAlreadyRunning: %v", err)
	}

	pidAfter, _ := os.ReadFile(layout.PIDPath)
	if string(pidAfter) != string(pidBefore) {
		t.Errorf("PID file changed: %q -> %q", pidBefore, pidAfter)
	}
	if got := socketIdentity(t, layout.SocketPath); got != sockBefore {
		t.Errorf("socket replaced: %s -> %s", sockBefore, got)
	}
	if !processAlive(h.pid()) {
		t.Error("incumbent was killed")
	}
}

// Done bar #3: an owner that is slow to answer (still binding) is given the
// startup grace, not retired.
func TestAcquireOwnershipWaitsForSlowStartingOwner(t *testing.T) {
	home := shortTempHome(t)
	layout := LayoutFor(home)
	h := startHelper(t, "owner", home, "SCRY_TEST_BIND_DELAY=700ms")

	var logs strings.Builder
	p := testPolicy(&logs)
	p.StartupGrace = 3 * time.Second
	start := time.Now()
	own, err := acquireOwnership(layout, p)
	if err == nil {
		own.Release()
		t.Fatal("acquired ownership from a slow-but-live owner")
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("err = %v, want ErrAlreadyRunning", err)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Errorf("returned after %v, before the owner could have bound", time.Since(start))
	}
	if !processAlive(h.pid()) {
		t.Error("slow owner was killed")
	}
	if strings.Contains(logs.String(), "retiring") {
		t.Errorf("logs show a retirement attempt:\n%s", logs.String())
	}
}

// Done bar #4: an unresponsive owner is TERMed and the successor only takes
// over after the process has exited and the lock is released.
func TestAcquireOwnershipRetiresUnresponsiveOwner(t *testing.T) {
	home := shortTempHome(t)
	layout := LayoutFor(home)
	h := startHelper(t, "owner", home, "SCRY_TEST_NO_BIND=1")

	var logs strings.Builder
	own, err := acquireOwnership(layout, testPolicy(&logs))
	if err != nil {
		t.Fatalf("acquireOwnership: %v\nlogs:\n%s", err, logs.String())
	}
	defer own.Release()

	if sig := h.wait(t); sig != syscall.SIGTERM {
		t.Errorf("incumbent ended by %v, want SIGTERM", sig)
	}
	b, _ := os.ReadFile(layout.LockPath)
	if got := strings.TrimSpace(string(b)); got != strconv.Itoa(os.Getpid()) {
		t.Errorf("lock file records %q after takeover, want our pid %d", got, os.Getpid())
	}
	for _, want := range []string{
		fmt.Sprintf("retiring unresponsive daemon pid %d", h.pid()),
		"SIGTERM delivered",
		"released ownership",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs missing %q:\n%s", want, logs.String())
		}
	}
}

// Done bar #5: an owner that ignores TERM is escalated to KILL after the
// term grace, deterministically, and is never left as an orphan.
func TestAcquireOwnershipEscalatesToKillWhenTermIgnored(t *testing.T) {
	home := shortTempHome(t)
	layout := LayoutFor(home)
	h := startHelper(t, "owner", home, "SCRY_TEST_NO_BIND=1", "SCRY_TEST_IGNORE_TERM=1")

	var logs strings.Builder
	p := testPolicy(&logs)
	p.TermGrace = 400 * time.Millisecond
	own, err := acquireOwnership(layout, p)
	if err != nil {
		t.Fatalf("acquireOwnership: %v\nlogs:\n%s", err, logs.String())
	}
	defer own.Release()

	if sig := h.wait(t); sig != syscall.SIGKILL {
		t.Errorf("incumbent ended by %v, want SIGKILL", sig)
	}
	if !strings.Contains(logs.String(), "SIGKILL") {
		t.Errorf("logs do not record the escalation:\n%s", logs.String())
	}
}

// A lock held by a process that never recorded its identity cannot be
// safely signalled; the contender must fail closed rather than guess.
func TestAcquireOwnershipRefusesUnverifiableLockOwner(t *testing.T) {
	home := shortTempHome(t)
	layout := LayoutFor(home)
	h := startHelper(t, "rawlock", home)

	var logs strings.Builder
	own, err := acquireOwnership(layout, testPolicy(&logs))
	if err == nil {
		own.Release()
		t.Fatal("acquired ownership despite an unverifiable lock owner")
	}
	if errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("err = %v, should not present as a healthy incumbent", err)
	}
	if !processAlive(h.pid()) {
		t.Error("unverifiable owner was signalled")
	}
}

// Transition: a daemon from before the lock existed, healthy on the socket,
// is deferred to (never retired), and the contender leaves no lock behind.
func TestAcquireOwnershipDefersToHealthyLegacyDaemon(t *testing.T) {
	home := shortTempHome(t)
	layout := LayoutFor(home)
	h := startHelper(t, "legacy", home)

	var logs strings.Builder
	own, err := acquireOwnership(layout, testPolicy(&logs))
	if err == nil {
		own.Release()
		t.Fatal("acquired ownership over a healthy legacy daemon")
	}
	var are *AlreadyRunningError
	if !errors.As(err, &are) || are.PID != h.pid() {
		t.Fatalf("err = %v, want AlreadyRunningError for pid %d", err, h.pid())
	}
	if !lockIsFree(t, layout) {
		t.Error("contender left the ownership lock held")
	}
	if !processAlive(h.pid()) {
		t.Error("legacy daemon was killed")
	}
}

// Transition: a legacy daemon that is alive but not answering is retired and
// waited for before the successor proceeds.
func TestAcquireOwnershipRetiresUnresponsiveLegacyDaemon(t *testing.T) {
	home := shortTempHome(t)
	layout := LayoutFor(home)
	h := startHelper(t, "legacy", home, "SCRY_TEST_NO_BIND=1")

	var logs strings.Builder
	own, err := acquireOwnership(layout, testPolicy(&logs))
	if err != nil {
		t.Fatalf("acquireOwnership: %v\nlogs:\n%s", err, logs.String())
	}
	defer own.Release()
	if sig := h.wait(t); sig != syscall.SIGTERM {
		t.Errorf("legacy daemon ended by %v, want SIGTERM", sig)
	}
}

// Done bar #2 and #6 at the Run level: a second Run against a live daemon
// steps aside without touching its files, and when the incumbent exits its
// cleanup cannot unlink the successor's socket — the successor comes up on a
// socket that is still there once the old process has fully returned.
func TestRunSecondInstanceStepsAsideAndSuccessorSurvivesIncumbentCleanup(t *testing.T) {
	t.Setenv("SCRY_MEMORY_UI_ADDR", "off")
	home := shortTempHome(t)
	layout := LayoutFor(home)

	runDaemon := func(ctx context.Context) <-chan error {
		errCh := make(chan error, 1)
		go func() { errCh <- New(layout).Run(ctx) }()
		return errCh
	}
	waitSocket := func() {
		t.Helper()
		// Generous on purpose: under `go test ./...` every package runs at
		// once, and a daemon that opens Badger and builds a search index
		// has taken more than ten seconds to answer on a loaded machine.
		deadline := time.Now().Add(60 * time.Second)
		for !pingSocket(layout.SocketPath) {
			if time.Now().After(deadline) {
				t.Fatal("daemon socket never came up")
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	first := runDaemon(ctx1)
	waitSocket()
	sockFirst := socketIdentity(t, layout.SocketPath)

	// A second start against a healthy incumbent steps aside.
	secondErr := <-runDaemon(context.Background())
	if !errors.Is(secondErr, ErrAlreadyRunning) {
		t.Fatalf("second Run err = %v, want ErrAlreadyRunning", secondErr)
	}
	if got := socketIdentity(t, layout.SocketPath); got != sockFirst {
		t.Errorf("second Run replaced the incumbent's socket")
	}

	// Stop the first while a successor is already trying to start: the
	// successor must wait for the incumbent's cleanup, not race it.
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()
	cancel1()
	third := runDaemon(ctx3)
	if err := <-first; err != nil {
		t.Fatalf("first Run returned %v", err)
	}
	waitSocket()
	if got := socketIdentity(t, layout.SocketPath); got == sockFirst {
		t.Errorf("successor is serving on the incumbent's socket object")
	}
	// Give the old process's deferred cleanup every chance to have run, then
	// confirm the successor's socket and PID file are intact.
	time.Sleep(100 * time.Millisecond)
	if !pingSocket(layout.SocketPath) {
		t.Fatal("successor socket was unlinked by the incumbent's cleanup")
	}
	if b, err := os.ReadFile(layout.PIDPath); err != nil || strings.TrimSpace(string(b)) != strconv.Itoa(os.Getpid()) {
		t.Errorf("successor PID file missing or wrong: %q, %v", b, err)
	}

	cancel3()
	if err := <-third; err != nil {
		t.Fatalf("third Run returned %v", err)
	}
	if !lockIsFree(t, layout) {
		t.Error("ownership lock still held after the last daemon exited")
	}
}

func TestRemoveOwnedPIDFileLeavesForeignPIDAlone(t *testing.T) {
	layout := LayoutFor(shortTempHome(t))
	if err := os.WriteFile(layout.PIDPath, []byte("424242"), 0o644); err != nil {
		t.Fatal(err)
	}
	removeOwnedPIDFile(layout, os.Getpid())
	if _, err := os.Stat(layout.PIDPath); err != nil {
		t.Fatalf("PID file belonging to another process was removed: %v", err)
	}
	if err := os.WriteFile(layout.PIDPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	removeOwnedPIDFile(layout, os.Getpid())
	if _, err := os.Stat(layout.PIDPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("own PID file not removed: %v", err)
	}
}
