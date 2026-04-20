package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func blameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blame <file>",
		Short: "Structured blame for a file or line range",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			startLine, _ := cmd.Flags().GetInt("start-line")
			endLine, _ := cmd.Flags().GetInt("end-line")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "git.blame", &daemon.GitBlameParams{
				Repo: repo, File: args[0], StartLine: startLine, EndLine: endLine,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Int("start-line", 0, "start line (inclusive)")
	cmd.Flags().Int("end-line", 0, "end line (inclusive)")
	return cmd
}

func historyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history [file]",
		Short: "Recent commits (repo-wide or per-file)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			p := daemon.GitHistoryParams{Repo: repo, Limit: limit}
			if len(args) == 1 {
				p.File = args[0]
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "git.history", &p, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Int("limit", 20, "max commits to return")
	return cmd
}

func cochangeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cochange <file>",
		Short: "Files that frequently change alongside target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "git.cochange", &daemon.GitCochangeParams{
				Repo: repo, File: args[0], Limit: limit,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Int("limit", 10, "max results")
	return cmd
}

func hotspotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hotspots",
		Short: "Most-churned files by commit frequency",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "git.hotspots", &daemon.GitHotspotsParams{
				Repo: repo, Limit: limit,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Int("limit", 20, "max results")
	return cmd
}

func contributorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contributors [file]",
		Short: "Main authors (repo-wide or per-file)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			p := daemon.GitContributorsParams{Repo: repo}
			if len(args) == 1 {
				p.File = args[0]
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "git.contributors", &p, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	return cmd
}

func intentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "intent <file>",
		Short: "Commit context for a specific line",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			line, _ := cmd.Flags().GetInt("line")
			if line <= 0 {
				return fmt.Errorf("--line is required and must be positive")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "git.intent", &daemon.GitIntentParams{
				Repo: repo, File: args[0], Line: line,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Int("line", 0, "line number (required)")
	_ = cmd.MarkFlagRequired("line")
	return cmd
}
