package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jeffdhooton/scry/internal/index"
)

// bootstrapWatchers starts a Watcher for the most recently indexed repos whose
// source directory still exists. Stale entries (source dir deleted) are
// silently skipped — the user can clean them up via `scry status` later.
//
// It deliberately does not watch every registered repo. Watching costs
// descriptors — on the kqueue backends, roughly one per file in every watched
// directory — so a machine with a hundred registered repos cannot hold them all
// at once without exhausting the system file table. The shared budget stops the
// walk when it runs out; ordering by index recency means the repos being worked
// in are the ones that get watched, and a query for any other repo starts its
// watcher on demand, evicting the least recently used one if the budget is
// full. Repos without a live watcher still report `stale` in `scry status`.
func (d *Daemon) bootstrapWatchers(ctx context.Context) {
	reposDir := filepath.Join(d.layout.Home, "repos")
	entries, err := os.ReadDir(reposDir)
	if err != nil {
		return
	}

	type candidate struct {
		repoPath  string
		indexedAt int64
	}
	var candidates []candidate

	for _, ent := range entries {
		manifestPath := filepath.Join(reposDir, ent.Name(), "manifest.json")
		b, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var m index.Manifest
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		if _, err := os.Stat(m.RepoPath); err != nil {
			// Source repo gone — skip silently.
			continue
		}
		candidates = append(candidates, candidate{
			repoPath:  m.RepoPath,
			indexedAt: m.IndexedAt.UnixNano(),
		})
	}

	// Most recently indexed first, so the budget is spent on active repos.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].indexedAt > candidates[j].indexedAt
	})

	watched := 0
	for _, c := range candidates {
		if !d.watcher.HasBudgetFor() {
			fmt.Fprintf(os.Stderr, "scry: watch budget spent after %d of %d repos; the rest are watched on demand\n",
				watched, len(candidates))
			return
		}
		if err := d.watcher.Watch(ctx, c.repoPath); err != nil {
			fmt.Fprintf(os.Stderr, "scry: bootstrap watcher %s: %v\n", c.repoPath, err)
			continue
		}
		watched++
	}
}
