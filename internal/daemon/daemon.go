// Package daemon is the long-running scry process. It owns the Unix socket,
// the per-repo BadgerDB stores, the file watchers, and the RPC dispatcher.
//
// One daemon per user. The CLI auto-spawns it on first call. See SPEC §4.1.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/jeffdhooton/scry/internal/rpc"
)

// DefaultShutdownGrace matches docs/DECISIONS.md "Daemon shutdown" — 5s
// from SIGTERM to forceful close.
const DefaultShutdownGrace = 5 * time.Second

// Layout is the on-disk daemon layout under ~/.scry. All paths are absolute.
type Layout struct {
	Home       string // ~/.scry
	SocketPath string // ~/.scry/scryd.sock
	PIDPath    string // ~/.scry/scryd.pid
	LogPath    string // ~/.scry/scryd.log
}

// LayoutFor builds the layout from a scry home directory.
func LayoutFor(home string) Layout {
	return Layout{
		Home:       home,
		SocketPath: filepath.Join(home, "scryd.sock"),
		PIDPath:    filepath.Join(home, "scryd.pid"),
		LogPath:    filepath.Join(home, "scryd.log"),
	}
}

// Daemon is one running scryd process.
type Daemon struct {
	layout   Layout
	registry *Registry
	server   *rpc.Server
	watcher  *Watcher

	mu       sync.Mutex
	listener net.Listener
}

// New constructs a Daemon for the given layout. It does NOT start anything;
// call Run to begin serving.
func New(layout Layout) *Daemon {
	d := &Daemon{
		layout:   layout,
		registry: NewRegistry(),
		server:   rpc.NewServer(),
	}
	d.watcher = NewWatcher(layout.Home, d.registry)
	d.registerMethods()
	return d
}

// Run takes ownership of the process: writes the PID file, opens the socket,
// dispatches RPC calls until ctx is cancelled or SIGTERM/SIGINT arrives, then
// performs a graceful shutdown.
//
// Returns nil on clean shutdown, otherwise the first error that broke the run.
func (d *Daemon) Run(ctx context.Context) error {
	// Raise NOFILE soft limit before doing anything else. fsnotify uses one
	// file descriptor per watched directory, and macOS' default 256 is way
	// below what a single Laravel-class repo needs (1000+ dirs is normal once
	// you count vendor and storage subtrees). The hard limit is usually
	// unlimited; we just need to opt into it.
	raiseNOFILE()

	if err := os.MkdirAll(d.layout.Home, 0o755); err != nil {
		return fmt.Errorf("ensure home: %w", err)
	}

	// Refuse to start if another daemon is already alive on the same socket.
	if alive, pid := d.aliveDaemonPID(); alive {
		return fmt.Errorf("scry daemon already running (pid %d, socket %s)", pid, d.layout.SocketPath)
	}

	// Stale socket from a previous crash — safe to remove now that we've
	// confirmed nothing's listening.
	if err := os.Remove(d.layout.SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale socket: %w", err)
	}

	ln, err := net.Listen("unix", d.layout.SocketPath)
	if err != nil {
		return fmt.Errorf("listen unix %s: %w", d.layout.SocketPath, err)
	}
	if err := os.Chmod(d.layout.SocketPath, 0o600); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	d.mu.Lock()
	d.listener = ln
	d.mu.Unlock()

	if err := os.WriteFile(d.layout.PIDPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		_ = ln.Close()
		return fmt.Errorf("write pid file: %w", err)
	}
	defer os.Remove(d.layout.PIDPath)
	defer os.Remove(d.layout.SocketPath)
	defer d.registry.CloseAll()
	defer d.watcher.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Wire up signal handling FIRST, before any code path that could exhaust
	// file descriptors. Earlier versions panicked here with "pipe failed"
	// when bootstrap watchers ate the fd budget and signal.Notify couldn't
	// open its internal self-pipe.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-runCtx.Done():
		}
	}()

	// Bootstrap: walk ~/.scry/repos and start a watcher for every repo whose
	// source path still exists. Repos whose source dir was deleted stay in
	// the index but get no watcher.
	d.bootstrapWatchers(runCtx)

	serveErr := d.server.Serve(runCtx, ln)
	if serveErr != nil && !errors.Is(serveErr, net.ErrClosed) {
		return serveErr
	}
	return nil
}

// aliveDaemonPID checks the PID file and pings the socket. Returns the PID of
// a confirmed-running daemon, or (false, 0) otherwise.
func (d *Daemon) aliveDaemonPID() (bool, int) {
	return AliveDaemon(d.layout)
}

// AliveDaemon is the standalone version usable from the CLI to decide whether
// to spawn a fresh daemon. Returns (true, pid) if a daemon is currently
// listening on the socket and the PID file matches.
func AliveDaemon(layout Layout) (bool, int) {
	pidBytes, err := os.ReadFile(layout.PIDPath)
	if err != nil {
		// No PID file; quickly try to dial anyway in case someone left one
		// behind.
		if pingSocket(layout.SocketPath) {
			return true, 0
		}
		return false, 0
	}
	pid, err := strconv.Atoi(string(bytesTrimSpace(pidBytes)))
	if err != nil {
		return false, 0
	}
	if !processAlive(pid) {
		return false, 0
	}
	if !pingSocket(layout.SocketPath) {
		return false, 0
	}
	return true, pid
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 doesn't deliver but does check existence/permissions.
	return proc.Signal(syscall.Signal(0)) == nil
}

func pingSocket(path string) bool {
	conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func bytesTrimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}
