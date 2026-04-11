package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/mcp"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// mcpCmd runs scry as an MCP stdio server for Claude Code (or any other MCP
// host). Reads newline-delimited JSON-RPC from stdin, writes responses to
// stdout, keeps all logging on stderr. The daemon is auto-spawned on first
// tool call via dialDaemon.
//
// Intended to be launched by a Claude Code mcpServers entry. Not meant to be
// run interactively.
func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run as an MCP stdio server (launched by Claude Code)",
		Long: `Speaks the Model Context Protocol on stdin/stdout so Claude Code
can call scry queries as first-class tools. The daemon is auto-spawned on first
call if it isn't already running.

Not meant to be run interactively. Configure via 'scry setup'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Respect SIGTERM / SIGINT so Claude Code can shut the server
			// down cleanly when it restarts or quits.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
			defer signal.Stop(sigCh)
			go func() {
				<-sigCh
				cancel()
			}()

			dial := func() (mcp.Dialer, error) {
				c, err := dialDaemon()
				if err != nil {
					return nil, err
				}
				return &mcpDialer{c: c}, nil
			}
			srv := mcp.New(dial)
			return srv.Serve(ctx, os.Stdin, os.Stdout)
		},
	}
}

// mcpDialer adapts *rpc.Client to the mcp.Dialer interface. The interface is
// intentionally narrower than rpc.Client so the mcp package doesn't have to
// import internal/rpc.
type mcpDialer struct {
	c *rpc.Client
}

func (d *mcpDialer) Call(ctx context.Context, method string, params, out any) error {
	return d.c.Call(ctx, method, params, out)
}

func (d *mcpDialer) Close() error {
	return d.c.Close()
}
