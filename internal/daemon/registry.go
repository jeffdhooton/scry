package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/store"
)

// Registry is a per-daemon cache of opened BadgerDB stores keyed by absolute
// repo path. The first query for a repo lazily opens its store and keeps it
// resident for the rest of the daemon lifetime.
//
// In P0 every CLI invocation paid the BadgerDB open cost (~5-15ms). The
// Registry amortizes that to once per daemon lifetime per repo.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*Entry
}

// Entry is one indexed repo as known to the daemon.
type Entry struct {
	RepoPath string
	Layout   index.RepoLayout
	Store    *store.Store
}

func NewRegistry() *Registry {
	return &Registry{entries: map[string]*Entry{}}
}

// Get returns the entry for repoPath, opening the store if necessary. Returns
// an error if the repo is not yet indexed (caller should run init).
func (r *Registry) Get(scryHome, repoPath string) (*Entry, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}
	r.mu.RLock()
	e := r.entries[abs]
	r.mu.RUnlock()
	if e != nil {
		return e, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if e = r.entries[abs]; e != nil {
		return e, nil
	}
	layout := index.Layout(scryHome, abs)
	if _, err := os.Stat(layout.BadgerDir); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("repo %s is not indexed yet — run `scry init` first", abs)
	} else if err != nil {
		return nil, err
	}
	st, err := store.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	e = &Entry{RepoPath: abs, Layout: layout, Store: st}
	r.entries[abs] = e
	return e, nil
}

// Put records an entry that the caller already constructed (e.g. after a
// successful index build).
func (r *Registry) Put(e *Entry) {
	r.mu.Lock()
	r.entries[e.RepoPath] = e
	r.mu.Unlock()
}

// Evict removes one repo from the registry, closing its store. Used by init
// before reindexing so the build can wipe BadgerDB without lock contention.
func (r *Registry) Evict(repoPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[repoPath]; ok {
		_ = e.Store.Close()
		delete(r.entries, repoPath)
	}
}

// SwapNext atomically swaps a freshly built BadgerDB at nextLayout.BadgerDir
// into the live position at liveLayout.BadgerDir, replacing whatever the
// registry currently has for liveLayout.RepoPath. The caller must have
// already finished writing the new database (BuildIntoTemp returns when the
// new store is closed and the directory is consistent on disk).
//
// Sequence inside the registry's write lock:
//
//  1. Close the live store (releases the BadgerDB directory lock).
//  2. Rename live BadgerDir → trash sibling.
//  3. Rename next BadgerDir → live.
//  4. Open the now-live BadgerDir and put a fresh entry into the registry.
//  5. Move the next manifest into place (best effort, not in the lock path).
//
// Returns the path to the archived old directory so the caller can delete
// it asynchronously. Returns an empty string if there was no live entry
// to archive (rare; happens if a watcher fires before the first init).
//
// On any error during the rename phase the function attempts to roll back
// to the original on-disk layout and the registry is left empty for that
// repo. The next query against the repo will lazy-open at the live path.
func (r *Registry) SwapNext(liveLayout, nextLayout index.RepoLayout) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	repo := liveLayout.RepoPath

	// Step 1: close the live store if any. The directory lock is released
	// when Close returns. Without this the rename in step 2 would fail on
	// platforms that hold a lock as a real fd (Linux/macOS via fcntl).
	if e, ok := r.entries[repo]; ok {
		_ = e.Store.Close()
		delete(r.entries, repo)
	}

	// Step 2: archive the live directory. This may not exist if the watcher
	// fired before any init ran — that's fine, we just skip the archive.
	trash := liveLayout.BadgerDir + ".old." + fmt.Sprintf("%d", os.Getpid()) + "." + nowSuffix()
	if _, err := os.Stat(liveLayout.BadgerDir); err == nil {
		if err := os.Rename(liveLayout.BadgerDir, trash); err != nil {
			return "", fmt.Errorf("archive live badger dir: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat live badger dir: %w", err)
	} else {
		trash = "" // nothing to clean up
	}

	// Step 3: promote the next directory.
	if err := os.Rename(nextLayout.BadgerDir, liveLayout.BadgerDir); err != nil {
		// Try to roll back the archive so the live store is at least the
		// previous version. If even that fails the registry is empty for
		// this repo and queries will return "not indexed yet" until the
		// next reindex.
		if trash != "" {
			_ = os.Rename(trash, liveLayout.BadgerDir)
		}
		return "", fmt.Errorf("promote next badger dir: %w", err)
	}

	// Move the next manifest into place. Best effort — a missing manifest
	// only affects `scry status` formatting.
	if _, err := os.Stat(nextLayout.ManifestPath); err == nil {
		if err := os.Rename(nextLayout.ManifestPath, liveLayout.ManifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "scry: promote manifest: %v\n", err)
		}
	}

	// Step 4: open the now-live store and register it. This is the only
	// step where queries against the repo are blocked, and it's a single
	// BadgerDB Open call (typically a few ms).
	st, err := store.Open(liveLayout.BadgerDir)
	if err != nil {
		return trash, fmt.Errorf("open swapped store: %w", err)
	}
	r.entries[repo] = &Entry{
		RepoPath: repo,
		Layout:   liveLayout,
		Store:    st,
	}
	return trash, nil
}

// nowSuffix returns a high-resolution timestamp for unique trash directory
// names. We don't need date formatting — just enough resolution that two
// near-simultaneous reindexes can't collide.
func nowSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// CloseAll closes every store in the registry. Called from daemon shutdown.
func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		_ = e.Store.Close()
	}
	r.entries = map[string]*Entry{}
}

// Snapshot returns a copy of the current entries for status reporting.
func (r *Registry) Snapshot() []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Entry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}
