package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sweepStaleIndexTrash deletes rotate-then-delete garbage under every repo
// storage dir: index.db.old.* archives that SwapNext's fire-and-forget
// cleanup goroutine never got to (it dies with the daemon — 31k orphans and
// 271 GB accumulated that way), plus index.db.next / manifest.json.next left
// by an interrupted build. Nothing reads any of these; the live index.db and
// manifest.json are never touched. Safe to run while the daemon serves: the
// paths are only ever created and consumed by this process, and a reindex
// in flight recreates index.db.next from scratch on entry anyway — the sweep
// runs once at startup, before any watcher can be reindexing.
func sweepStaleIndexTrash(scryHome string) {
	reposDir := filepath.Join(scryHome, "repos")
	repos, err := os.ReadDir(reposDir)
	if err != nil {
		return
	}
	removed := 0
	for _, repo := range repos {
		if !repo.IsDir() {
			continue
		}
		storageDir := filepath.Join(reposDir, repo.Name())
		entries, err := os.ReadDir(storageDir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			name := ent.Name()
			stale := strings.HasPrefix(name, "index.db.old.") ||
				name == "index.db.next" ||
				name == "manifest.json.next"
			if !stale {
				continue
			}
			p := filepath.Join(storageDir, name)
			if err := os.RemoveAll(p); err != nil {
				fmt.Fprintf(os.Stderr, "scry: sweep stale index trash %s: %v\n", p, err)
				continue
			}
			removed++
		}
	}
	if removed > 0 {
		fmt.Fprintf(os.Stderr, "scry: swept %d stale index dirs\n", removed)
	}
}
