package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/index"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [path]",
		Short: "Index a repository",
		Long:  "Index the repository at [path] (or the current directory) by running the language indexer and writing the result to ~/.scry/repos/<hash>/.",
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
			home, err := scryHome()
			if err != nil {
				return err
			}

			start := time.Now()
			fmt.Fprintf(os.Stderr, "scry: indexing %s\n", repo)
			manifest, err := index.Build(context.Background(), home, repo)
			if err != nil {
				return err
			}
			elapsed := time.Since(start)

			pretty, _ := cmd.Flags().GetBool("pretty")
			out := map[string]any{
				"repo":      manifest.RepoPath,
				"languages": manifest.Languages,
				"status":    manifest.Status,
				"stats":     manifest.Stats,
				"elapsed_ms": elapsed.Milliseconds(),
			}
			return printJSON(out, pretty)
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
