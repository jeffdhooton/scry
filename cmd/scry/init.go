package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [path]",
		Short: "Index a repository",
		Long:  "Ask the scry daemon to index the repository at [path] (or the current directory). The daemon runs the language indexer, writes the BadgerDB store under ~/.scry/repos/<hash>/, and registers the repo for queries.",
		Args:  cobra.MaximumNArgs(1),
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

			fmt.Fprintf(os.Stderr, "scry: indexing %s\n", repo)
			start := time.Now()
			var result daemon.InitResult
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if err := callDaemon(ctx, "init", &daemon.InitParams{Repo: repo}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			fmt.Fprintf(os.Stderr, "scry: indexed in %s\n", time.Since(start).Round(time.Millisecond))
			return printJSON(result, pretty)
		},
	}
}

func absPath(p string) (string, error) {
	info, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", p, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", p)
	}
	return p, nil
}

func printJSON(v any, pretty bool) error {
	enc := json.NewEncoder(os.Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}
