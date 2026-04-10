package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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
