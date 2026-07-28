// Package ingest is the CLI-side ingest pipeline: it distills a source
// (a Claude Code or Codex CLI transcript, a loom run directory, or a
// hand-authored seed markdown file), extracts a knowledge graph from each
// resulting episode via an LLM, and commits the result to the daemon's
// memory store, advancing a cursor so re-runs make forward progress instead
// of reprocessing what's already been ingested.
//
// The package depends only on small interfaces (Daemon, extract.Extractor)
// so it is testable without a running daemon or network access — cmd/scry
// supplies the real implementations (a daemonClient wrapping callDaemon, and
// extract.NewHaiku).
package ingest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/resolve"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// glossaryLimit is the number of "slug: aliases" lines fetched once per File
// call, mirroring the daemon's own default (see defaultGlossaryLimit in
// internal/daemon/memory_methods.go).
const glossaryLimit = 200

// Daemon is the small set of memory-domain RPCs File needs. Implemented in
// cmd/scry by a daemonClient wrapping callDaemon; faked in tests.
type Daemon interface {
	Glossary(ctx context.Context, limit int) ([]string, error)
	Commit(ctx context.Context, ep store.Episode, cwd string, res extract.Result) (resolve.Stats, error)
	GetCursor(ctx context.Context, path string) (store.Cursor, bool, error)
	PutCursor(ctx context.Context, c store.Cursor) error
}

// Options configures a single File ingest call.
type Options struct {
	Source    string // "claude" | "codex" | "loom" | "seed"
	Path      string
	Extractor extract.Extractor
	Daemon    Daemon
}

// Summary reports what one File call did.
type Summary struct {
	EpisodesIngested int
	EpisodesSkipped  int
	Stats            resolve.Stats
}

// add accumulates stats from one successful commit into the running Summary.
func (s *Summary) add(stats resolve.Stats) {
	s.EpisodesIngested++
	s.Stats.EntitiesCreated += stats.EntitiesCreated
	s.Stats.EntitiesUpdated += stats.EntitiesUpdated
	s.Stats.FactsAdded += stats.FactsAdded
	s.Stats.FactsInvalidated += stats.FactsInvalidated
	s.Stats.FactsMerged += stats.FactsMerged
}

// offsetDistillFunc is the shape shared by distill.ClaudeSession and
// distill.CodexRollout.
type offsetDistillFunc func(path string, offset int64) ([]distill.RawEpisode, int64, error)

// File ingests one transcript/run/seed path from its cursor offset (episodic
// sources: claude, codex) or wholesale (loom, seed). It returns per-file
// stats and advances the cursor on success.
//
// Extraction errors are classified: a content-level parse failure (the
// model's output never became valid JSON even after Extract's own retry,
// wrapped in extract.ErrParse) is non-fatal — that one episode is skipped
// (counted in Summary.EpisodesSkipped) and ingestion continues with the
// next one. Every other extraction error — a canceled/deadline-exceeded
// context, a request that never got a response, or anything else not
// wrapping extract.ErrParse — aborts the whole call immediately, exactly
// like a commit error: the error is returned and the cursor is NOT
// advanced, so a retry picks back up from the last successfully-committed
// episode instead of silently treating the unprocessed remainder as done.
func File(ctx context.Context, o Options) (Summary, error) {
	switch o.Source {
	case "claude":
		return ingestOffset(ctx, o, distill.ClaudeSession)
	case "codex":
		return ingestOffset(ctx, o, distill.CodexRollout)
	case "loom":
		return ingestWholesale(ctx, o, func() ([]distill.RawEpisode, error) {
			return distill.LoomRun(o.Path)
		})
	case "seed":
		return ingestWholesale(ctx, o, func() ([]distill.RawEpisode, error) {
			ep, err := distill.SeedMarkdown(o.Path)
			if err != nil {
				return nil, err
			}
			return []distill.RawEpisode{ep}, nil
		})
	default:
		return Summary{}, fmt.Errorf("ingest: unknown source %q", o.Source)
	}
}

// ingestOffset handles the byte-offset-resume sources (claude, codex):
// resume from the stored cursor's ProcessedBytes, distill only the new
// bytes, commit each resulting episode, then advance the cursor to the new
// offset — but only once every commit has succeeded.
func ingestOffset(ctx context.Context, o Options, distillFn offsetDistillFunc) (Summary, error) {
	cursor, found, err := o.Daemon.GetCursor(ctx, o.Path)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: get cursor: %w", err)
	}
	var startOffset int64
	if found {
		startOffset = cursor.ProcessedBytes
	}

	episodes, newOffset, err := distillFn(o.Path, startOffset)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: distill %s: %w", o.Path, err)
	}

	info, err := os.Stat(o.Path)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: stat %s: %w", o.Path, err)
	}

	sum, err := commitEpisodes(ctx, o, episodes)
	if err != nil {
		return sum, err
	}

	if err := o.Daemon.PutCursor(ctx, store.Cursor{
		Path:           o.Path,
		Size:           info.Size(),
		ModTime:        info.ModTime(),
		ProcessedBytes: newOffset,
	}); err != nil {
		return sum, fmt.Errorf("ingest: put cursor: %w", err)
	}
	return sum, nil
}

// ingestWholesale handles the always-read-in-full sources (loom, seed):
// distill everything, commit it, then record a cursor keyed on the path with
// ProcessedBytes 0 and ModTime set to the path's own mtime (a directory's
// mtime for loom, a file's mtime for seed). Change detection based on that
// mtime is Task 9's job (sweep) — File here only records it.
func ingestWholesale(ctx context.Context, o Options, distillFn func() ([]distill.RawEpisode, error)) (Summary, error) {
	episodes, err := distillFn()
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: distill %s: %w", o.Path, err)
	}

	sum, err := commitEpisodes(ctx, o, episodes)
	if err != nil {
		return sum, err
	}

	info, err := os.Stat(o.Path)
	if err != nil {
		return sum, fmt.Errorf("ingest: stat %s: %w", o.Path, err)
	}
	if err := o.Daemon.PutCursor(ctx, store.Cursor{
		Path:           o.Path,
		Size:           0,
		ModTime:        info.ModTime(),
		ProcessedBytes: 0,
	}); err != nil {
		return sum, fmt.Errorf("ingest: put cursor: %w", err)
	}
	return sum, nil
}

// commitEpisodes fetches the glossary once, then extracts and commits each
// episode in order. A content-level parse failure (extract.ErrParse) skips
// just that episode; every other extraction failure — context errors and
// transport-ish failures alike — aborts immediately, same as a commit
// failure, returning the stats accumulated so far alongside the error so the
// caller does NOT advance the cursor past unprocessed episodes.
func commitEpisodes(ctx context.Context, o Options, episodes []distill.RawEpisode) (Summary, error) {
	var sum Summary

	glossary, err := o.Daemon.Glossary(ctx, glossaryLimit)
	if err != nil {
		return sum, fmt.Errorf("ingest: glossary: %w", err)
	}

	for _, ep := range episodes {
		res, err := o.Extractor.Extract(ctx, ep, glossary)
		if err != nil {
			if errors.Is(err, extract.ErrParse) {
				sum.EpisodesSkipped++
				continue
			}
			// Context errors (ctx.Err() non-nil, including
			// context.DeadlineExceeded/Canceled) and any other
			// transport-ish or unclassified failure: stop processing this
			// file's remaining episodes entirely rather than skipping past
			// them, so they aren't permanently hidden from the next sweep.
			return sum, fmt.Errorf("ingest: extract episode %s: %w", ep.ID, err)
		}

		storeEp := store.Episode{
			ID:         ep.ID,
			Source:     ep.Source,
			SourceRef:  ep.SourceRef,
			Summary:    res.EpisodeSummary,
			OccurredAt: ep.OccurredAt,
			IngestedAt: time.Now(),
		}

		stats, err := o.Daemon.Commit(ctx, storeEp, ep.Cwd, res)
		if err != nil {
			return sum, fmt.Errorf("ingest: commit episode %s: %w", ep.ID, err)
		}
		sum.add(stats)
	}

	return sum, nil
}
