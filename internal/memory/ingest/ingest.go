// Package ingest is the client-side half of memory ingestion: it distills
// a source (a Claude Code, Codex, Kimi, or OpenCode transcript, a loom run
// directory, or a hand-authored seed markdown file) into redacted
// episodes, hands them to the daemon's queue, and advances a cursor so
// re-runs make forward progress instead of reprocessing what is done.
//
// Extraction is not this package's job any more. The daemon that owns the
// store runs the model chain (internal/memory/queue), so the sweep on a
// laptop needs no API key and no provider config, and a provider outage
// defers writes instead of failing the sweep.
//
// The package depends only on the small Daemon interface so it is testable
// without a running daemon — cmd/scry supplies the real implementation (a
// daemonClient wrapping callMemoryDaemon).
package ingest

import (
	"context"
	"fmt"
	"os"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// EnqueueBatch bounds how many episodes ride in one memory.enqueue call so
// a long transcript does not turn into a single multi-megabyte request.
const EnqueueBatch = 50

// Daemon is the set of memory-domain RPCs File needs. Implemented in
// cmd/scry by daemonClient; faked in tests.
type Daemon interface {
	// Enqueue hands distilled episodes to the daemon's queue. It reports
	// how many were newly queued and how many the daemon already knew
	// (queued earlier, or already resolved into the store).
	Enqueue(ctx context.Context, episodes []distill.RawEpisode) (queued, known int, err error)
	GetCursor(ctx context.Context, path string) (store.Cursor, bool, error)
	PutCursor(ctx context.Context, c store.Cursor) error
}

// Options configures a single File call.
type Options struct {
	// Force re-reads the source from the beginning and re-applies episodes
	// the daemon already holds, to repair ones resolved before a fix. The
	// cursor is still advanced afterwards.
	Force  bool
	Source string // "claude" | "codex" | "kimi" | "opencode" | "loom" | "seed" (opencode paths are OpenCodeRefs)
	Path   string
	Daemon Daemon
}

// Summary reports what one File call did.
type Summary struct {
	EpisodesIngested int // newly queued at the daemon
	EpisodesSkipped  int // already known to the daemon
}

// offsetDistillFunc is the shape shared by the byte-offset-resume
// distillers (distill.ClaudeSession, distill.CodexRollout, distill.KimiWire).
type offsetDistillFunc func(path string, offset int64) ([]distill.RawEpisode, int64, error)

// File ingests one source path from its cursor offset (episodic sources) or
// wholesale (loom, seed, opencode). It returns per-file counts and advances
// the cursor only once every episode has been accepted by the daemon, so a
// failed enqueue leaves the cursor where it was and the next run retries.
func File(ctx context.Context, o Options) (Summary, error) {
	switch o.Source {
	case "claude":
		return ingestOffset(ctx, o, distill.ClaudeSession)
	case "codex":
		return ingestOffset(ctx, o, distill.CodexRollout)
	case "kimi":
		return ingestOffset(ctx, o, distill.KimiWire)
	case "opencode":
		return ingestOpenCode(ctx, o)
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

// ingestOffset handles the byte-offset-resume sources: resume from the
// stored cursor's ProcessedBytes, distill only the new bytes, enqueue, then
// advance the cursor to the new offset.
func ingestOffset(ctx context.Context, o Options, distillFn offsetDistillFunc) (Summary, error) {
	cursor, found, err := o.Daemon.GetCursor(ctx, o.Path)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: get cursor: %w", err)
	}
	var startOffset int64
	if found && !o.Force {
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

	sum, err := Enqueue(ctx, o.Daemon, episodes)
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
// distill everything, enqueue it, then record a cursor keyed on the path
// with ModTime set to the path's own mtime so the sweep can detect change.
func ingestWholesale(ctx context.Context, o Options, distillFn func() ([]distill.RawEpisode, error)) (Summary, error) {
	episodes, err := distillFn()
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: distill %s: %w", o.Path, err)
	}

	sum, err := Enqueue(ctx, o.Daemon, episodes)
	if err != nil {
		return sum, err
	}

	info, err := os.Stat(o.Path)
	if err != nil {
		return sum, fmt.Errorf("ingest: stat %s: %w", o.Path, err)
	}
	if err := o.Daemon.PutCursor(ctx, store.Cursor{
		Path:    o.Path,
		ModTime: info.ModTime(),
	}); err != nil {
		return sum, fmt.Errorf("ingest: put cursor: %w", err)
	}
	return sum, nil
}

// Enqueue sends episodes to the daemon in batches of EnqueueBatch. A
// failed batch aborts with the counts so far; the caller must not advance
// its cursor past that point.
func Enqueue(ctx context.Context, d Daemon, episodes []distill.RawEpisode) (Summary, error) {
	var sum Summary
	for start := 0; start < len(episodes); start += EnqueueBatch {
		end := min(start+EnqueueBatch, len(episodes))
		queued, known, err := d.Enqueue(ctx, episodes[start:end])
		if err != nil {
			return sum, fmt.Errorf("ingest: enqueue episodes %d-%d: %w", start, end, err)
		}
		sum.EpisodesIngested += queued
		sum.EpisodesSkipped += known
	}
	return sum, nil
}

// ingestOpenCode handles one OpenCode session, addressed by an OpenCodeRef
// ("opencode:<db>:<session>") rather than a file. The whole session is
// distilled each time; episode ids are deterministic so the daemon dedupes
// what it already has. The cursor's ModTime is the session's own
// time_updated, which is what the sweep compares against.
func ingestOpenCode(ctx context.Context, o Options) (Summary, error) {
	dbPath, sessionID, ok := distill.ParseOpenCodeRef(o.Path)
	if !ok {
		return Summary{}, fmt.Errorf("ingest: %q is not an opencode ref", o.Path)
	}
	episodes, updated, err := distill.OpenCodeSessionEpisodes(dbPath, sessionID)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: distill %s: %w", o.Path, err)
	}
	sum, err := Enqueue(ctx, o.Daemon, episodes)
	if err != nil {
		return sum, err
	}
	if err := o.Daemon.PutCursor(ctx, store.Cursor{Path: o.Path, ModTime: updated}); err != nil {
		return sum, fmt.Errorf("ingest: put cursor: %w", err)
	}
	return sum, nil
}
