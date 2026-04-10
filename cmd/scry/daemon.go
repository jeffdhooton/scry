package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// dialDaemon opens a client connection to the running daemon, auto-spawning
// it if it isn't already up. The wait budget is 2 seconds total — long enough
// for a cold daemon to come up, short enough to fail loudly if something is
// genuinely wrong.
func dialDaemon() (*rpc.Client, error) {
	home, err := scryHome()
	if err != nil {
		return nil, err
	}
	layout := daemon.LayoutFor(home)

	if alive, _ := daemon.AliveDaemon(layout); !alive {
		if err := spawnDaemon(); err != nil {
			return nil, fmt.Errorf("auto-spawn daemon: %w", err)
		}
		if err := waitForSocket(layout.SocketPath, 2*time.Second); err != nil {
			return nil, fmt.Errorf("daemon did not come up: %w", err)
		}
	}

	c, err := rpc.Dial(layout.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("dial daemon: %w", err)
	}
	return c, nil
}

// spawnDaemon forks the current scry binary as a detached background process
// running `scry start --background`. The child becomes session-leader so it
// survives our exit, and its stdio goes to the daemon log file.
func spawnDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	home, err := scryHome()
	if err != nil {
		return err
	}
	layout := daemon.LayoutFor(home)

	// Daemon log goes to ~/.scry/scryd.log per docs/DECISIONS.md.
	logFile, err := os.OpenFile(layout.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open daemon log: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(exe, "start", "--foreground")
	// Detach from this process group so the child outlives us. Setpgid is
	// portable across darwin/linux/freebsd. We avoid Setsid in case the user
	// runs scry from a TTY shell where it has unintended side effects.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start child: %w", err)
	}
	// Don't Wait — we want the child to outlive us. Release lets the OS reap
	// it when it eventually exits.
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release child: %w", err)
	}
	return nil
}

// waitForSocket polls Dial until the daemon is accepting connections, or the
// budget expires.
func waitForSocket(socketPath string, budget time.Duration) error {
	deadline := time.Now().Add(budget)
	delay := 10 * time.Millisecond
	for {
		if pingSocket(socketPath) {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("timeout waiting for daemon socket")
		}
		time.Sleep(delay)
		if delay < 100*time.Millisecond {
			delay *= 2
		}
	}
}

func pingSocket(path string) bool {
	c, err := rpc.Dial(path)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// callDaemon is a one-shot helper: dial the daemon, send one request, decode
// the result, close the connection.
func callDaemon(ctx context.Context, method string, params, out any) error {
	c, err := dialDaemon()
	if err != nil {
		return err
	}
	defer c.Close()
	return c.Call(ctx, method, params, out)
}
