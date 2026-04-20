package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func proxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Manage the HTTP capture proxy",
	}
	cmd.AddCommand(proxyStartCmd())
	cmd.AddCommand(proxyStopCmd())
	return cmd
}

func proxyStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the HTTP capture proxy",
		Long: `Start a reverse proxy that captures all HTTP traffic between your client and
dev server. Requests are stored for 30 minutes and queryable via scry requests.

  scry proxy start                          # default: :8089 → localhost:8000
  scry proxy start --port 9090 --target localhost:3000`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			target, _ := cmd.Flags().GetString("target")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var result daemon.HTTPStartResult
			if err := callDaemon(ctx, "http.start", &daemon.HTTPStartParams{
				Port: port, Target: target,
			}, &result); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "scry: proxy running on :%d → %s\n", result.Port, result.Target)
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Int("port", 8089, "port to listen on")
	cmd.Flags().String("target", "localhost:8000", "upstream server address")
	return cmd
}

func proxyStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the HTTP capture proxy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var result map[string]any
			if err := callDaemon(ctx, "http.stop", struct{}{}, &result); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "scry: proxy stopped\n")
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func requestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests",
		Short: "List captured HTTP requests",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("path")
			method, _ := cmd.Flags().GetString("method")
			limit, _ := cmd.Flags().GetInt("limit")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var result any
			if err := callDaemon(ctx, "http.requests", map[string]any{
				"path": path, "method": method, "limit": limit,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().String("path", "", "filter by path substring")
	cmd.Flags().String("method", "", "filter by HTTP method")
	cmd.Flags().Int("limit", 20, "max results")
	return cmd
}

func requestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "request <id>",
		Short: "Show full details for a captured HTTP request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var result any
			if err := callDaemon(ctx, "http.request", &daemon.HTTPRequestParams{
				ID: args[0],
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}
