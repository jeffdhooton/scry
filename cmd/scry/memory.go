package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/ingest"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/memory/sweep"
)

// dormantNotice is printed (with exit 0, not an error) by ingest/sweep/
// backfill when neither API key env var is set — the memory domain is
// opt-in, so an unconfigured key is a no-op, not a failure.
const dormantNotice = "memory: dormant (no ANTHROPIC_API_KEY / SCRY_MEMORY_API_KEY)"

// memoryDormant reports whether no API key is configured for extraction:
// SCRY_MEMORY_API_KEY, falling back to ANTHROPIC_API_KEY — mirroring
// buildMemoryExtractor in internal/daemon/daemon.go exactly, since the CLI's
// ingest pipeline and the daemon's memory.remember must agree on when the
// domain is "live".
func memoryDormant() bool {
	return os.Getenv("SCRY_MEMORY_API_KEY") == "" && os.Getenv("ANTHROPIC_API_KEY") == ""
}

// memoryCmd is the `scry memory` command tree: global episodic memory graph
// (as opposed to the per-repo code intelligence commands elsewhere in this
// package).
func memoryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "memory", Short: "Global episodic memory graph"}
	cmd.AddCommand(memoryIngestCmd(), memorySweepCmd(), memoryBackfillCmd(),
		memoryOrientCmd(), memoryRecallCmd(), memoryEntitiesCmd(), memoryFactsCmd(),
		memoryInvalidateCmd(), memoryStatusCmd())
	return cmd
}

// --- daemonClient: ingest.Daemon implemented over callDaemon ---

// daemonClient implements ingest.Daemon by calling the daemon's memory.*
// RPCs, using exactly the param/result structs internal/daemon/memory_methods.go
// defines (task 7's RPC contract).
type daemonClient struct{}

var _ ingest.Daemon = daemonClient{}

func (daemonClient) Glossary(ctx context.Context, limit int) ([]string, error) {
	var result []string
	err := callDaemon(ctx, "memory.glossary", &daemon.MemoryGlossaryParams{Limit: limit}, &result)
	return result, err
}

func (daemonClient) Commit(ctx context.Context, ep memstore.Episode, cwd string, res extract.Result) (resolve.Stats, error) {
	var stats resolve.Stats
	err := callDaemon(ctx, "memory.commit", &daemon.MemoryCommitParams{
		Episode: ep, Cwd: cwd, Result: res,
	}, &stats)
	return stats, err
}

func (daemonClient) GetCursor(ctx context.Context, path string) (memstore.Cursor, bool, error) {
	var result daemon.MemoryCursorGetResult
	err := callDaemon(ctx, "memory.cursor.get", &daemon.MemoryCursorGetParams{Path: path}, &result)
	return result.Cursor, result.Found, err
}

func (daemonClient) PutCursor(ctx context.Context, c memstore.Cursor) error {
	// memory.cursor.put takes the Cursor itself as params (no wrapper
	// struct) — see handleMemoryCursorPut, which unmarshals raw straight
	// into a memstore.Cursor.
	var result map[string]any
	return callDaemon(ctx, "memory.cursor.put", &c, &result)
}

// --- ingest ---

func memoryIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Distill, extract, and commit one transcript/run/seed source",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if memoryDormant() {
				fmt.Println(dormantNotice)
				return nil
			}

			source, _ := cmd.Flags().GetString("source")
			switch source {
			case "claude", "codex", "loom", "seed":
			default:
				return fmt.Errorf("--source must be one of claude|codex|loom|seed, got %q", source)
			}
			path, _ := cmd.Flags().GetString("path")

			apiKey := os.Getenv("SCRY_MEMORY_API_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			extractor := extract.NewHaiku(apiKey, os.Getenv("SCRY_MEMORY_MODEL"))

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			sum, err := ingest.File(ctx, ingest.Options{
				Source:    source,
				Path:      path,
				Extractor: extractor,
				Daemon:    daemonClient{},
			})
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(sum, pretty)
		},
	}
	cmd.Flags().String("source", "", "source type: claude|codex|loom|seed (required)")
	cmd.Flags().String("path", "", "path to the transcript file, run directory, or seed markdown file (required)")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}

// --- sweep (Task 9) / backfill (stub, filled in by Task 10) ---

func memorySweepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Scan default roots (Claude/Codex transcripts, loom runs) for new episodes and ingest deltas",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if memoryDormant() {
				fmt.Println(dormantNotice)
				return nil
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")

			apiKey := os.Getenv("SCRY_MEMORY_API_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			extractor := extract.NewHaiku(apiKey, os.Getenv("SCRY_MEMORY_MODEL"))

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			result, err := sweep.Run(ctx, sweep.Roots{}, ingest.Options{
				Extractor: extractor,
				Daemon:    daemonClient{},
			}, sweep.DefaultActiveWindow, dryRun)
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Bool("dry-run", false, "report what would be ingested without extracting or committing anything")
	return cmd
}

func memoryBackfillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backfill",
		Short: "Backfill a source's full history (not yet implemented)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if memoryDormant() {
				fmt.Println(dormantNotice)
				return nil
			}
			return errors.New("memory backfill: implemented in a later task")
		},
	}
}

// --- query verbs: thin callDaemon wrappers ---

func memoryOrientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orient",
		Short: "Short markdown orientation blurb for a working directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := cmd.Flags().GetString("cwd")
			if cwd == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				cwd = wd
			}
			budget, _ := cmd.Flags().GetInt("budget")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result struct {
				Markdown string `json:"markdown"`
			}
			if err := callDaemon(ctx, "memory.orient", &daemon.MemoryOrientParams{
				Cwd: cwd, Budget: budget,
			}, &result); err != nil {
				return err
			}
			// Orient prints markdown raw, not JSON — it's meant to be read
			// directly (e.g. injected into an agent's context), not parsed.
			fmt.Println(result.Markdown)
			return nil
		},
	}
	cmd.Flags().String("cwd", "", "working directory to orient for (default: process cwd)")
	cmd.Flags().Int("budget", 2000, "approximate character budget for the blurb")
	return cmd
}

func memoryRecallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall <query>",
		Short: "Fuzzy entity search, optionally as-of a point in time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			asOf, _ := cmd.Flags().GetString("as-of")
			limit, _ := cmd.Flags().GetInt("limit")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "memory.recall", &daemon.MemoryRecallParams{
				Query: args[0], AsOf: asOf, Limit: limit,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().String("as-of", "", "RFC3339 timestamp; empty means current")
	cmd.Flags().Int("limit", 5, "max results")
	return cmd
}

func memoryEntitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entities",
		Short: "List entities, optionally filtered by type",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			typ, _ := cmd.Flags().GetString("type")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "memory.entities", &daemon.MemoryEntitiesParams{
				Type: typ,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().String("type", "", "filter by entity type (project|service|machine|tool|person|decision|runbook|concept)")
	return cmd
}

func memoryFactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts <slug>",
		Short: "Every fact about a single entity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "memory.facts", &daemon.MemoryFactsParams{
				Slug: args[0], IncludeInvalid: all,
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().Bool("all", false, "include invalidated (historical) facts")
	return cmd
}

func memoryInvalidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "invalidate <src> <relation> <dst>",
		Short: "Invalidate every current fact matching the exact triple",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "memory.invalidate", &daemon.MemoryInvalidateParams{
				Src: args[0], Relation: args[1], Dst: args[2],
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func memoryStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Memory domain counts and dormancy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result daemon.MemoryStatusResult
			if err := callDaemon(ctx, "memory.status", nil, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}
