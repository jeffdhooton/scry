package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the scry daemon",
		Long: `Start the long-running scryd daemon. With no flags, scry start
detaches a background process and returns. With --foreground, scryd runs in
the calling shell — useful for debugging or running under a process supervisor.

The CLI auto-spawns the daemon on first query, so manual start is only needed
for inspecting daemon logs or running under a supervisor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			foreground, _ := cmd.Flags().GetBool("foreground")
			home, err := scryHome()
			if err != nil {
				return err
			}
			layout := daemon.LayoutFor(home)

			if foreground {
				d := daemon.New(layout)
				return d.Run(context.Background())
			}

			if alive, pid := daemon.AliveDaemon(layout); alive {
				fmt.Fprintf(os.Stderr, "scry: daemon already running (pid %d)\n", pid)
				return nil
			}
			if err := spawnDaemon(); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "scry: daemon spawned")
			return nil
		},
	}
	cmd.Flags().Bool("foreground", false, "run the daemon in the foreground (do not detach)")
	return cmd
}
