package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/memory/distill"
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
		memoryOrientCmd(), memoryRecallCmd(), memoryRememberCmd(), memoryEntitiesCmd(),
		memoryFactsCmd(), memoryInvalidateCmd(), memoryStatusCmd())
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

// HasEpisodes reports which of ids are NOT yet committed to the store — see
// backfillDaemon and runBackfill's use of it to skip re-paying for
// extraction on episodes a previous run already ingested.
func (daemonClient) HasEpisodes(ctx context.Context, ids []string) ([]string, error) {
	var result daemon.MemoryHasEpisodesResult
	err := callDaemon(ctx, "memory.hasEpisodes", &daemon.MemoryHasEpisodesParams{IDs: ids}, &result)
	return result.Missing, err
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
	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Backfill every episode across default roots via the Batch API (50% discount), or serially with --no-batch",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if memoryDormant() {
				fmt.Println(dormantNotice)
				return nil
			}

			sinceStr, _ := cmd.Flags().GetString("since")
			var since time.Time
			if sinceStr != "" {
				t, err := time.Parse("2006-01-02", sinceStr)
				if err != nil {
					return fmt.Errorf("--since: invalid date %q, want YYYY-MM-DD: %w", sinceStr, err)
				}
				since = t
			}
			noBatch, _ := cmd.Flags().GetBool("no-batch")

			apiKey := os.Getenv("SCRY_MEMORY_API_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			model := os.Getenv("SCRY_MEMORY_MODEL")

			// No overall timeout — this is a long-running, potentially
			// hours-spanning job (batches can take up to 24h to end). ctx is
			// still cancellable (Ctrl-C), which both the batch poll loop and
			// the serial fallback respect.
			ctx := context.Background()

			summary, err := runBackfill(ctx, backfillConfig{
				Since:   since,
				NoBatch: noBatch,
				Daemon:  daemonClient{},
				Haiku:   extract.NewHaiku(apiKey, model),
				Batch:   extract.NewBatchRunner(apiKey, model),
			})
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(summary, pretty)
		},
	}
	cmd.Flags().String("since", "", "only backfill episodes that occurred on/after this date (YYYY-MM-DD); default: everything")
	cmd.Flags().Bool("no-batch", false, "use serial Haiku.Extract calls (200ms between requests) instead of the Batch API")
	return cmd
}

// backfillDaemon is the daemon surface runBackfill needs: ingest.Daemon's
// glossary/commit/cursor RPCs plus a batched already-ingested check
// (memory.hasEpisodes) that lets backfill skip re-paying for extraction on
// episodes a previous ingest/sweep/backfill run already committed.
type backfillDaemon interface {
	ingest.Daemon
	HasEpisodes(ctx context.Context, ids []string) ([]string, error)
}

// backfillConfig bundles what runBackfill needs beyond ctx/flags.
type backfillConfig struct {
	Since   time.Time // zero means "everything"
	NoBatch bool
	Daemon  backfillDaemon
	Haiku   *extract.Haiku
	Batch   *extract.BatchRunner
}

// backfillGlossaryLimit mirrors ingest.glossaryLimit / the daemon's own
// default (see defaultGlossaryLimit in internal/daemon/memory_methods.go).
const backfillGlossaryLimit = 200

// backfillSummary is what `memory backfill` prints on success.
type backfillSummary struct {
	FilesScanned         int      `json:"files_scanned"`
	EpisodesFound        int      `json:"episodes_found"`
	EpisodesAlreadyKnown int      `json:"episodes_already_known"`
	EpisodesCommitted    int      `json:"episodes_committed"`
	EpisodesSkipped      int      `json:"episodes_skipped"`
	FilesAdvanced        int      `json:"files_advanced"`
	Errors               []string `json:"errors,omitempty"`
}

// backfillGroup is every episode distilled from one source path/dir, kept
// together so that path's cursor is only advanced once every one of its
// episodes has a definitive outcome (committed or recorded as an error) —
// mirroring ingest.File's "commit failures don't advance the cursor" safety
// property, but at file granularity across a batch that spans many files.
type backfillGroup struct {
	source    string // "claude" | "codex" | "loom"
	path      string
	episodes  []distill.RawEpisode
	newOffset int64 // full-file byte length for claude/codex; unused for loom
	size      int64
	modTime   time.Time
}

// runBackfill discovers every episode across the default roots (ignoring
// cursors entirely — commit-side idempotency dedupes anything already
// ingested), filters by cfg.Since, extracts each episode (via the Batch API
// unless cfg.NoBatch), commits every successfully-extracted episode, and —
// per source file/dir whose every episode resolved — advances that path's
// cursor to reflect it's now fully processed.
func runBackfill(ctx context.Context, cfg backfillConfig) (backfillSummary, error) {
	var summary backfillSummary

	claudeFiles, codexFiles, loomDirs, discoverErrs := sweep.Candidates(sweep.Roots{}, sweep.DefaultActiveWindow)
	summary.Errors = append(summary.Errors, discoverErrs...)
	summary.FilesScanned = len(claudeFiles) + len(codexFiles) + len(loomDirs)

	var groups []backfillGroup
	for _, path := range claudeFiles {
		g, err := backfillClaudeOrCodex(path, "claude", distill.ClaudeSession, cfg.Since)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		groups = append(groups, g)
	}
	for _, path := range codexFiles {
		g, err := backfillClaudeOrCodex(path, "codex", distill.CodexRollout, cfg.Since)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		groups = append(groups, g)
	}
	for _, dir := range loomDirs {
		episodes, err := distill.LoomRun(dir)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		episodes = filterSince(episodes, cfg.Since)
		info, err := os.Stat(dir)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: stat: %v", dir, err))
			continue
		}
		groups = append(groups, backfillGroup{source: "loom", path: dir, episodes: episodes, modTime: info.ModTime()})
	}

	var discovered []distill.RawEpisode
	for _, g := range groups {
		discovered = append(discovered, g.episodes...)
	}
	summary.EpisodesFound = len(discovered)

	if len(discovered) == 0 {
		fmt.Println("nothing to backfill")
		return summary, nil
	}

	// F5: skip re-extracting (re-paying for) episodes a previous
	// ingest/sweep/backfill run already committed to the store —
	// resolve.Apply's own idempotency check would no-op them anyway, so
	// extracting them again buys nothing but a wasted API call.
	discoveredIDs := make([]string, len(discovered))
	for i, ep := range discovered {
		discoveredIDs[i] = ep.ID
	}
	missing, err := cfg.Daemon.HasEpisodes(ctx, discoveredIDs)
	if err != nil {
		return summary, fmt.Errorf("memory backfill: hasEpisodes: %w", err)
	}
	summary.EpisodesAlreadyKnown = len(discovered) - len(missing)

	var allEpisodes []distill.RawEpisode
	for _, g := range groups {
		needExtract, _ := partitionKnownEpisodes(g.episodes, missing)
		allEpisodes = append(allEpisodes, needExtract...)
	}

	var results map[string]extract.Result
	var extractErrs map[string]error
	if len(allEpisodes) == 0 {
		fmt.Println("nothing to extract: every discovered episode is already ingested")
	} else {
		glossary, err := cfg.Daemon.Glossary(ctx, backfillGlossaryLimit)
		if err != nil {
			return summary, fmt.Errorf("memory backfill: glossary: %w", err)
		}

		var fatalErr error
		if cfg.NoBatch {
			results, extractErrs, fatalErr = backfillSerial(ctx, cfg.Haiku, allEpisodes, glossary)
			if fatalErr != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("backfill canceled: %v", fatalErr))
			}
		} else {
			totalBatches := (len(allEpisodes) + extract.MaxBatchSize - 1) / extract.MaxBatchSize
			progress := func(done, total int) {
				batchNum := (done + extract.MaxBatchSize - 1) / extract.MaxBatchSize
				if batchNum < 1 {
					batchNum = 1
				}
				if batchNum > totalBatches {
					batchNum = totalBatches
				}
				fmt.Printf("batch %d/%d: %d/%d done\n", batchNum, totalBatches, done, total)
			}
			results, extractErrs, fatalErr = cfg.Batch.Run(ctx, allEpisodes, glossary, progress)
			if fatalErr != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("batch run: %v", fatalErr))
			}
		}
	}

	missingSet := make(map[string]bool, len(missing))
	for _, id := range missing {
		missingSet[id] = true
	}

	for _, g := range groups {
		resolved := true
		for _, ep := range g.episodes {
			if !missingSet[ep.ID] {
				// Already committed by a previous run: nothing left to do
				// for it, and it counts as resolved for cursor purposes.
				continue
			}
			res, ok := results[ep.ID]
			if ok {
				if _, err := cfg.Daemon.Commit(ctx, memstore.Episode{
					ID:         ep.ID,
					Source:     ep.Source,
					SourceRef:  ep.SourceRef,
					Summary:    res.EpisodeSummary,
					OccurredAt: ep.OccurredAt,
					IngestedAt: time.Now(),
				}, ep.Cwd, res); err != nil {
					summary.Errors = append(summary.Errors, fmt.Sprintf("commit %s: %v", ep.ID, err))
					resolved = false
					continue
				}
				summary.EpisodesCommitted++
				continue
			}
			if extractErr, ok := extractErrs[ep.ID]; ok {
				summary.Errors = append(summary.Errors, fmt.Sprintf("extract %s: %v", ep.ID, extractErr))
				if errors.Is(extractErr, extract.ErrParse) {
					// F1: only a content-level parse failure is
					// skip-and-continue. Anything else (a transport-ish
					// error, canceled/expired batch item) leaves the
					// group's cursor unresolved below, same as an
					// unresolved episode.
					summary.EpisodesSkipped++
					continue
				}
				resolved = false
				continue
			}
			// Neither map has this episode ID: its batch never resolved
			// (Run returned a fatal error before this chunk completed). Leave
			// the file's cursor untouched so a re-run retries it.
			resolved = false
		}

		if !resolved {
			continue
		}

		// F5(b): with --since set, episodes that occurred before the cutoff
		// were filtered out before we ever saw them (see filterSince) — the
		// group above only reflects what made it past that filter, not the
		// whole file. Advancing the cursor here would wrongly mark the file
		// as fully processed and hide those earlier episodes from every
		// future sweep/backfill. Only a --since-less (full) backfill is
		// allowed to retire a file's cursor.
		if !cfg.Since.IsZero() {
			continue
		}

		cursor := memstore.Cursor{Path: g.path, ModTime: g.modTime}
		if g.source != "loom" {
			cursor.Size = g.size
			cursor.ProcessedBytes = g.newOffset
		}
		if err := cfg.Daemon.PutCursor(ctx, cursor); err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("put cursor %s: %v", g.path, err))
			continue
		}
		summary.FilesAdvanced++
	}

	return summary, nil
}

// partitionKnownEpisodes splits episodes into those that still need
// extraction (their ID is present in missing) and those already committed
// by a previous run (ID absent from missing). Pure and side-effect-free so
// it's cheaply unit-testable independent of the daemon RPC that produces
// missing.
func partitionKnownEpisodes(episodes []distill.RawEpisode, missing []string) (needExtract, alreadyKnown []distill.RawEpisode) {
	missingSet := make(map[string]bool, len(missing))
	for _, id := range missing {
		missingSet[id] = true
	}
	for _, ep := range episodes {
		if missingSet[ep.ID] {
			needExtract = append(needExtract, ep)
		} else {
			alreadyKnown = append(alreadyKnown, ep)
		}
	}
	return needExtract, alreadyKnown
}

// backfillClaudeOrCodex distills the full contents of an episodic
// (byte-offset-resume) source from offset 0 — deliberately ignoring
// whatever cursor might exist, since backfill wants every episode on disk —
// and filters by since.
func backfillClaudeOrCodex(path, source string, distillFn func(path string, offset int64) ([]distill.RawEpisode, int64, error), since time.Time) (backfillGroup, error) {
	episodes, newOffset, err := distillFn(path, 0)
	if err != nil {
		return backfillGroup{}, fmt.Errorf("distill: %w", err)
	}
	episodes = filterSince(episodes, since)

	info, err := os.Stat(path)
	if err != nil {
		return backfillGroup{}, fmt.Errorf("stat: %w", err)
	}

	return backfillGroup{
		source:    source,
		path:      path,
		episodes:  episodes,
		newOffset: newOffset,
		size:      info.Size(),
		modTime:   info.ModTime(),
	}, nil
}

// filterSince returns only the episodes that occurred at or after since. A
// zero since (the --since flag was omitted) keeps everything.
func filterSince(episodes []distill.RawEpisode, since time.Time) []distill.RawEpisode {
	if since.IsZero() {
		return episodes
	}
	var out []distill.RawEpisode
	for _, ep := range episodes {
		if !ep.OccurredAt.Before(since) {
			out = append(out, ep)
		}
	}
	return out
}

// backfillSerial extracts every episode one at a time via Haiku.Extract,
// sleeping 200ms between calls (the --no-batch fallback: no batch discount,
// but no wait for a batch to end either). A per-episode extraction error is
// recorded in the error map and does not stop the run.
//
// The third return is fatal-only, mirroring BatchRunner.Run's contract: nil
// on a normal (possibly partially-erroring) completion, or ctx's error if
// the run was canceled mid-loop — surfaced so the caller can record it in
// the summary rather than silently returning a truncated result set.
func backfillSerial(ctx context.Context, h *extract.Haiku, episodes []distill.RawEpisode, glossary []string) (map[string]extract.Result, map[string]error, error) {
	results := make(map[string]extract.Result, len(episodes))
	errs := make(map[string]error)

	for i, ep := range episodes {
		if err := ctx.Err(); err != nil {
			return results, errs, err
		}

		res, err := h.Extract(ctx, ep, glossary)
		if err != nil {
			errs[ep.ID] = err
		} else {
			results[ep.ID] = res
		}
		fmt.Printf("serial: %d/%d done\n", i+1, len(episodes))

		if i < len(episodes)-1 {
			select {
			case <-ctx.Done():
				return results, errs, ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
	}

	return results, errs, nil
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

func memoryRememberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remember <fact>",
		Short: "Store a durable fact in global memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entities, _ := cmd.Flags().GetStringArray("entity")

			// No dormancy gate here: unlike ingest, the daemon holds the
			// extractor for memory.remember and reports Dormant in the
			// result itself, so the CLI just relays that rather than
			// short-circuiting before the call.
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			var result daemon.MemoryRememberResult
			if err := callDaemon(ctx, "memory.remember", &daemon.MemoryRememberParams{
				Fact: args[0], Entities: entities,
			}, &result); err != nil {
				return err
			}
			if result.Dormant {
				fmt.Fprintln(os.Stderr, "memory: daemon is dormant (no API key in its environment) — fact stored as episode only, no graph facts extracted")
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	cmd.Flags().StringArray("entity", nil, "hint entity name to bias resolution (repeatable)")
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
