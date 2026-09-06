package ingest

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// ingestCurated is an explicit single-file import, never a sweep root. Cursors
// are namespaced by both canonical repository and source, independent of other
// importers. Callers serialize imports of the same source, as for other cursors.
func ingestCurated(ctx context.Context, o Options) (Summary, error) {
	if o.Repo == "" || o.Path == "" || o.Force {
		return Summary{}, fmt.Errorf("curated ingestion requires --repo and --path, and does not support --force")
	}
	repo, err := filepath.Abs(o.Repo)
	if err != nil {
		return Summary{}, err
	}
	repo, err = filepath.EvalSymlinks(repo)
	if err != nil {
		return Summary{}, err
	}
	// Check on the source machine; never infer ownership from a basename.
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return Summary{}, fmt.Errorf("curated repository must have a .git entry: %w", err)
	}
	path, err := filepath.Abs(o.Path)
	if err != nil {
		return Summary{}, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return Summary{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Summary{}, err
	}
	if !info.Mode().IsRegular() {
		return Summary{}, fmt.Errorf("curated source must be a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return Summary{}, err
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil {
		return Summary{}, err
	}
	if !info.Mode().IsRegular() {
		return Summary{}, fmt.Errorf("curated source must be a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, distill.MaxCuratedBytes+1))
	if err != nil {
		return Summary{}, err
	}
	ep := distill.RawEpisode{Source: distill.CuratedSource, SourceRef: distill.CuratedRef(path, string(data)), Text: string(data), Cwd: repo, CwdIsRepo: true}
	if err := distill.ValidateCurated(ep); err != nil {
		return Summary{}, err
	}
	_, hash, _ := distill.ParseCuratedRef(ep.SourceRef)
	cursorPath := distill.CuratedSource + ":" + distill.MakeID(repo+"\x00"+path)
	cursor, found, err := o.Daemon.GetCursor(ctx, cursorPath)
	if err != nil {
		return Summary{}, fmt.Errorf("curated cursor: %w", err)
	}
	if found && cursor.ContentHash == hash {
		return Summary{EpisodesSkipped: 1}, nil
	}
	// The predecessor makes A -> B -> A a new observation while retries after
	// an accepted enqueue / failed cursor write reuse the exact episode ID.
	ep.ID = distill.MakeID(cursorPath + "\x00" + cursor.EpisodeID + "\x00" + hash)
	ep.OccurredAt = time.Now().UTC()
	if !ep.OccurredAt.After(cursor.ModTime) {
		ep.OccurredAt = cursor.ModTime.Add(time.Nanosecond)
	}
	sum, err := Enqueue(ctx, o.Daemon, []distill.RawEpisode{ep})
	if err != nil {
		return sum, err
	}
	if sum.EpisodesIngested+sum.EpisodesSkipped != 1 {
		return sum, fmt.Errorf("curated enqueue did not acknowledge the complete source")
	}
	// For this source ModTime is the monotonic observation time, not file mtime.
	// No acceptance, no cursor advancement. Extraction may still be pending.
	err = o.Daemon.PutCursor(ctx, store.Cursor{Path: cursorPath, Size: int64(len(data)), ModTime: ep.OccurredAt, ContentHash: hash, EpisodeID: ep.ID})
	if err != nil {
		return sum, fmt.Errorf("curated put cursor: %w", err)
	}
	return sum, nil
}
