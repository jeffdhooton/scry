package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running scry daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := scryHome()
			if err != nil {
				return err
			}
			layout := daemon.LayoutFor(home)

			alive, pid := daemon.AliveDaemon(layout)
			if !alive {
				fmt.Fprintln(os.Stderr, "scry: no daemon running")
				return nil
			}

			// Try a clean shutdown via RPC first; fall back to SIGTERM if the
			// RPC call fails for any reason.
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if err := callDaemon(ctx, "shutdown", nil, nil); err != nil {
				if pid > 0 {
					_ = syscall.Kill(pid, syscall.SIGTERM)
				}
			}

			// Wait up to the shutdown grace for the socket to disappear.
			deadline := time.Now().Add(daemon.DefaultShutdownGrace)
			for time.Now().Before(deadline) {
				if alive, _ := daemon.AliveDaemon(layout); !alive {
					fmt.Fprintln(os.Stderr, "scry: daemon stopped")
					return nil
				}
				time.Sleep(50 * time.Millisecond)
			}

			// Force-kill after grace.
			if pid > 0 {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
			fmt.Fprintln(os.Stderr, "scry: daemon force-killed after grace period")
			return nil
		},
	}
}
