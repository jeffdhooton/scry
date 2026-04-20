// Package git provides the git intelligence domain registry and RPC handlers
// for the scry daemon.
package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	gitindex "github.com/jeffdhooton/scry/internal/git/index"
	gitstore "github.com/jeffdhooton/scry/internal/git/store"
)

// Registry is a per-daemon cache of opened git BadgerDB stores keyed by
// absolute repo path.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*Entry
}

type Entry struct {
	RepoPath string
	Layout   gitindex.RepoLayout
	Store    *gitstore.Store
}

func NewRegistry() *Registry {
	return &Registry{entries: map[string]*Entry{}}
}

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
	layout := gitindex.Layout(scryHome, abs)
	if _, err := os.Stat(layout.BadgerDir); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("repo %s has no git index — run `scry init --git` first", abs)
	} else if err != nil {
		return nil, err
	}
	st, err := gitstore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open git store: %w", err)
	}
	e = &Entry{RepoPath: abs, Layout: layout, Store: st}
	r.entries[abs] = e
	return e, nil
}

func (r *Registry) Put(e *Entry) {
	r.mu.Lock()
	r.entries[e.RepoPath] = e
	r.mu.Unlock()
}

func (r *Registry) Evict(repoPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[repoPath]; ok {
		_ = e.Store.Close()
		delete(r.entries, repoPath)
	}
}

func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		_ = e.Store.Close()
	}
	r.entries = map[string]*Entry{}
}

func (r *Registry) Snapshot() []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Entry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}
