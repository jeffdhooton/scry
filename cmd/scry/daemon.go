package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/config"
	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/rpc"

	"github.com/jeffdhooton/scry/internal/logrotate"
)

const memorySocketEnv = "SCRY_MEMORY_SOCKET"

const (
	// The daemon only rotates at start, so this is a per-lifetime
	// ceiling rather than a hard bound.
	maxDaemonLogBytes    = 32 << 20
	daemonLogGenerations = 3
)

func daemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the scry daemon",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "restart",
		Short: "Restart the daemon with the current environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := scryHome()
			if err != nil {
				return err
			}
			layout := daemon.LayoutFor(home)
			alive, pid := daemon.AliveDaemon(layout)
			if alive {
				ctx, cancel := context.WithTimeout(cmd.Context(), time.Second)
				err := callDaemon(ctx, "shutdown", nil, nil)
				cancel()
				if err != nil && pid > 0 {
					_ = syscall.Kill(pid, syscall.SIGTERM)
				}
				deadline := time.Now().Add(daemon.DefaultShutdownGrace)
				for time.Now().Before(deadline) {
					if running, _ := daemon.AliveDaemon(layout); !running {
						break
					}
					time.Sleep(50 * time.Millisecond)
				}
				if running, currentPID := daemon.AliveDaemon(layout); running {
					if currentPID > 0 {
						_ = syscall.Kill(currentPID, syscall.SIGKILL)
					}
					return errors.New("daemon did not stop")
				}
			}
			if err := spawnDaemon(); err != nil {
				return err
			}
			if err := waitForSocket(layout.SocketPath, 2*time.Second); err != nil {
				return fmt.Errorf("daemon did not restart: %w", err)
			}
			fmt.Fprintln(os.Stderr, "scry: daemon restarted")
			return nil
		},
	})
	return cmd
}

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
		// 5s rather than 2s: a launchd-mediated start adds a hop, and a
		// starter that must retire an unresponsive incumbent waits for it.
		if err := waitForSocket(layout.SocketPath, 5*time.Second); err != nil {
			return nil, fmt.Errorf("daemon did not come up: %w", err)
		}
	}

	c, err := rpc.Dial(layout.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("dial daemon: %w", err)
	}
	return c, nil
}

// dialMemoryDaemon returns the configured shared-memory daemon when
// SCRY_MEMORY_SOCKET names a Unix socket (typically an SSH StreamLocalForward
// to an always-on machine). Without the variable, memory stays local for
// backwards compatibility.
func dialMemoryDaemon() (*rpc.Client, error) {
	path, source := memorySocket()
	if path == "" {
		return dialDaemon()
	}
	c, err := rpc.Dial(path)
	if err != nil {
		return nil, fmt.Errorf("dial shared memory daemon via %s=%q: %w", source, path, err)
	}
	return c, nil
}

// memorySocket resolves where the shared memory store is served: the
// SCRY_MEMORY_SOCKET environment variable first, then memory.socket in
// ~/.scry/config.yaml. Empty means the local daemon. The second return
// names the source for error messages.
func memorySocket() (path, source string) {
	if p := os.Getenv(memorySocketEnv); p != "" {
		return p, memorySocketEnv
	}
	home, err := scryHome()
	if err != nil {
		return "", ""
	}
	cfg, err := config.Load(home)
	if err != nil {
		return "", ""
	}
	if p := cfg.MemorySocket(); p != "" {
		return p, "config.yaml memory.socket"
	}
	return "", ""
}

// spawnDaemon starts the daemon. When a LaunchAgent supervises the daemon it
// is the one start authority: we ask launchd to kickstart it and never spawn
// a competing detached process (which would race KeepAlive for the socket
// and would run without the secret-bearing environment the agent sources —
// see docs/DAEMON_SPLIT_BRAIN_DIAGNOSIS.md). Without an agent, or if
// launchctl refuses (agent not bootstrapped), the current binary is forked
// as a detached background process running `scry start --foreground`.
func spawnDaemon() error {
	if agent, ok := daemon.FindLaunchAgent(); ok {
		if err := agent.Kickstart(); err == nil {
			fmt.Fprintf(os.Stderr, "scry: asked launchd to start %s\n", agent.Label)
			return nil
		} else {
			fmt.Fprintf(os.Stderr, "scry: %v; spawning daemon directly\n", err)
		}
	}
	return spawnDetachedDaemon()
}

// spawnDetachedDaemon forks the current scry binary as a detached background
// process. The child gets its own process group so it survives our exit,
// and its stdio goes to the daemon log file.
func spawnDetachedDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	home, err := scryHome()
	if err != nil {
		return err
	}
	layout := daemon.LayoutFor(home)

	// Daemon log goes to ~/.scry/scryd.log per docs/DECISIONS.md. The child
	// inherits this handle for its whole life, so rotation can only happen
	// here, at start. One unrotated lifetime reached 296 MB.
	logFile, err := logrotate.OpenAppend(layout.LogPath, maxDaemonLogBytes, daemonLogGenerations, 0o600)
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

// callMemoryDaemon is the memory-domain counterpart to callDaemon. It keeps
// code/git/schema/HTTP/room calls on the local daemon while allowing every
// memory CLI verb—including sweep and orient—to use one remote authority.
func callMemoryDaemon(ctx context.Context, method string, params, out any) error {
	c, err := dialMemoryDaemon()
	if err != nil {
		return err
	}
	defer c.Close()
	return c.Call(ctx, method, params, out)
}
