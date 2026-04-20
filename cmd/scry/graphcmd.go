package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func graphCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Unified cross-domain graph",
	}
	cmd.AddCommand(graphBuildCmd())
	cmd.AddCommand(graphQueryCmd())
	cmd.AddCommand(graphPathCmd())
	cmd.AddCommand(graphReportCmd())
	return cmd
}

func graphBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build [path]",
		Short: "Build the unified graph from all indexed domains",
		Long: `Build a cross-domain graph connecting code, schema, git, and HTTP data.
Requires at least one domain to be indexed first (scry init, scry init --git, etc.).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				abs, err := absPath(args[0])
				if err != nil {
					return err
				}
				repo = abs
			}

			fmt.Fprintf(os.Stderr, "scry: building graph %s\n", repo)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			var result daemon.GraphBuildResult
			if err := callDaemon(ctx, "graph.build", &daemon.GraphBuildParams{Repo: repo}, &result); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "scry: graph built in %dms (%d nodes, %d edges, %d communities)\n",
				result.ElapsedMs, result.NodeCount, result.EdgeCount, result.Communities)
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func graphQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query <term>",
		Short: "Search the graph for nodes by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "graph.query", &daemon.GraphQueryParams{
				Repo: repo, Query: args[0],
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func graphPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Find shortest path between two nodes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "graph.path", &daemon.GraphPathParams{
				Repo: repo, From: from, To: to,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().String("from", "", "source node name (required)")
	cmd.Flags().String("to", "", "target node name (required)")
	return cmd
}

func graphReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Show the pre-computed graph report",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "graph.report", &daemon.GraphReportParams{
				Repo: repo,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}
