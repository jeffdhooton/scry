package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	schemaindex "github.com/jeffdhooton/scry/internal/schema/index"
	schemastore "github.com/jeffdhooton/scry/internal/schema/store"
)

type SchemaRegistry struct {
	mu      sync.RWMutex
	entries map[string]*SchemaEntry
}

type SchemaEntry struct {
	ProjectDir string
	Layout     schemaindex.RepoLayout
	Store      *schemastore.Store
}

func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{entries: map[string]*SchemaEntry{}}
}

func (r *SchemaRegistry) Get(scryHome, projectDir string) (*SchemaEntry, error) {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
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
	layout := schemaindex.Layout(scryHome, abs)
	if _, err := os.Stat(layout.BadgerDir); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("project %s has no schema index — run `scry init --schema` first", abs)
	} else if err != nil {
		return nil, err
	}
	st, err := schemastore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open schema store: %w", err)
	}
	e = &SchemaEntry{ProjectDir: abs, Layout: layout, Store: st}
	r.entries[abs] = e
	return e, nil
}

func (r *SchemaRegistry) Put(e *SchemaEntry) {
	r.mu.Lock()
	r.entries[e.ProjectDir] = e
	r.mu.Unlock()
}

func (r *SchemaRegistry) Evict(projectDir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[projectDir]; ok {
		_ = e.Store.Close()
		delete(r.entries, projectDir)
	}
}

func (r *SchemaRegistry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		_ = e.Store.Close()
	}
	r.entries = map[string]*SchemaEntry{}
}

func (r *SchemaRegistry) Snapshot() []*SchemaEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*SchemaEntry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}
