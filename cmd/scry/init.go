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
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Index a repository",
		Long: `Ask the scry daemon to index the repository at [path] (or the current directory).

By default, indexes code (symbols, references, call graphs). Use flags to index
additional domains:

  scry init --git         Index git history (blame, commits, cochange, hotspots)
  scry init --schema      Index database schema (requires --dsn or --detect-env)
  scry init --all         Index everything detected (code + git + schema if DSN available)`,
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

			gitOnly, _ := cmd.Flags().GetBool("git")
			schemaOnly, _ := cmd.Flags().GetBool("schema")
			all, _ := cmd.Flags().GetBool("all")
			dsn, _ := cmd.Flags().GetString("dsn")
			detectEnv, _ := cmd.Flags().GetBool("detect-env")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			pretty, _ := cmd.Flags().GetBool("pretty")

			type combinedResult struct {
				Code   *daemon.InitResult       `json:"code,omitempty"`
				Git    *daemon.GitInitResult    `json:"git,omitempty"`
				Schema *daemon.SchemaInitResult `json:"schema,omitempty"`
				Graph  *daemon.GraphBuildResult `json:"graph,omitempty"`
			}
			var combined combinedResult

			singleDomain := gitOnly || schemaOnly

			// Index code unless a single domain flag was passed
			if !singleDomain {
				fmt.Fprintf(os.Stderr, "scry: indexing code %s\n", repo)
				start := time.Now()
				var result daemon.InitResult
				if err := callDaemon(ctx, "init", &daemon.InitParams{Repo: repo}, &result); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "scry: code indexed in %s\n", time.Since(start).Round(time.Millisecond))
				combined.Code = &result
			}

			// Index git if --git or --all
			if gitOnly || all {
				depth, _ := cmd.Flags().GetInt("depth")
				fmt.Fprintf(os.Stderr, "scry: indexing git history %s\n", repo)
				start := time.Now()
				var result daemon.GitInitResult
				if err := callDaemon(ctx, "git.init", &daemon.GitInitParams{Repo: repo, Depth: depth}, &result); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "scry: git indexed in %s\n", time.Since(start).Round(time.Millisecond))
				combined.Git = &result
			}

			// Index schema if --schema or --all (when DSN available)
			if schemaOnly || all {
				p := daemon.SchemaInitParams{Project: repo, DSN: dsn, DetectEnv: detectEnv}
				if all && dsn == "" && !detectEnv {
					p.DetectEnv = true
				}
				if schemaOnly && dsn == "" && !detectEnv {
					p.DetectEnv = true
				}
				fmt.Fprintf(os.Stderr, "scry: indexing database schema %s\n", repo)
				start := time.Now()
				var result daemon.SchemaInitResult
				err := callDaemon(ctx, "schema.init", &p, &result)
				if err != nil && all {
					fmt.Fprintf(os.Stderr, "scry: schema indexing skipped: %s\n", err)
				} else if err != nil {
					return err
				} else {
					fmt.Fprintf(os.Stderr, "scry: schema indexed in %s (%d tables)\n", time.Since(start).Round(time.Millisecond), result.TableCount)
					combined.Schema = &result
				}
			}

			// Build the unified graph when --all is used (at least one domain must be indexed)
			if all {
				fmt.Fprintf(os.Stderr, "scry: building unified graph %s\n", repo)
				start := time.Now()
				var graphResult daemon.GraphBuildResult
				err := callDaemon(ctx, "graph.build", &daemon.GraphBuildParams{Repo: repo}, &graphResult)
				if err != nil {
					fmt.Fprintf(os.Stderr, "scry: graph build skipped: %s\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "scry: graph built in %s (%d nodes, %d edges, %d communities)\n",
						time.Since(start).Round(time.Millisecond), graphResult.NodeCount, graphResult.EdgeCount, graphResult.Communities)
					combined.Graph = &graphResult
				}
			}

			// If only one domain was indexed, print just that result
			count := 0
			if combined.Code != nil {
				count++
			}
			if combined.Git != nil {
				count++
			}
			if combined.Schema != nil {
				count++
			}
			if combined.Graph != nil {
				count++
			}
			if count == 1 {
				if combined.Code != nil {
					return printJSON(combined.Code, pretty)
				}
				if combined.Git != nil {
					return printJSON(combined.Git, pretty)
				}
				if combined.Schema != nil {
					return printJSON(combined.Schema, pretty)
				}
				return printJSON(combined.Graph, pretty)
			}
			return printJSON(combined, pretty)
		},
	}
	cmd.Flags().Bool("git", false, "index git history (blame, commits, cochange, hotspots)")
	cmd.Flags().Bool("schema", false, "index database schema (auto-detects DSN from .env)")
	cmd.Flags().Bool("all", false, "index everything detected (code + git + schema)")
	cmd.Flags().Int("depth", 500, "number of commits to index (0 = all)")
	cmd.Flags().String("dsn", "", "database connection string (mysql:// or postgres://)")
	cmd.Flags().Bool("detect-env", false, "auto-detect DSN from .env file")
	return cmd
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
