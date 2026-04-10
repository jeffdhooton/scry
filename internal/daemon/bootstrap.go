package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeffdhooton/scry/internal/index"
)

// bootstrapWatchers walks ~/.scry/repos and starts a Watcher for every repo
// whose source directory still exists. Stale entries (source dir deleted) are
// silently skipped — the user can clean them up via `scry status` later.
func (d *Daemon) bootstrapWatchers(ctx context.Context) {
	reposDir := filepath.Join(d.layout.Home, "repos")
	entries, err := os.ReadDir(reposDir)
	if err != nil {
		return
	}
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
		if err := d.watcher.Watch(ctx, m.RepoPath); err != nil {
			fmt.Fprintf(os.Stderr, "scry: bootstrap watcher %s: %v\n", m.RepoPath, err)
		}
	}
}
