package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/graph"
	graphstore "github.com/jeffdhooton/scry/internal/graph/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

func (d *Daemon) registerGraphMethods() {
	d.server.Register("graph.build", d.handleGraphBuild)
	d.server.Register("graph.query", d.handleGraphQuery)
	d.server.Register("graph.path", d.handleGraphPath)
	d.server.Register("graph.report", d.handleGraphReport)
}

// --- graph registry (lightweight, single store per repo) ---

type GraphRegistry struct {
	mu      sync.RWMutex
	entries map[string]*GraphEntry
}

type GraphEntry struct {
	RepoPath string
	Layout   graph.RepoLayout
	Store    *graphstore.Store
}

func NewGraphRegistry() *GraphRegistry {
	return &GraphRegistry{entries: map[string]*GraphEntry{}}
}

func (r *GraphRegistry) Get(scryHome, repoPath string) (*GraphEntry, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
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
	layout := graph.Layout(scryHome, abs)
	if _, err := os.Stat(layout.BadgerDir); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("repo %s has no graph — run `scry graph build` first", abs)
	} else if err != nil {
		return nil, err
	}
	st, err := graphstore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open graph store: %w", err)
	}
	e = &GraphEntry{RepoPath: abs, Layout: layout, Store: st}
	r.entries[abs] = e
	return e, nil
}

func (r *GraphRegistry) Put(e *GraphEntry) {
	r.mu.Lock()
	r.entries[e.RepoPath] = e
	r.mu.Unlock()
}

func (r *GraphRegistry) Evict(repoPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[repoPath]; ok {
		_ = e.Store.Close()
		delete(r.entries, repoPath)
	}
}

func (r *GraphRegistry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		_ = e.Store.Close()
	}
	r.entries = map[string]*GraphEntry{}
}

func (r *GraphRegistry) Snapshot() []*GraphEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*GraphEntry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}

// --- graph.build ---

type GraphBuildParams struct {
	Repo string `json:"repo"`
}

type GraphBuildResult struct {
	Repo        string `json:"repo"`
	Status      string `json:"status"`
	NodeCount   int    `json:"node_count"`
	EdgeCount   int    `json:"edge_count"`
	Communities int    `json:"communities"`
	ElapsedMs   int64  `json:"elapsed_ms"`
}

func (d *Daemon) handleGraphBuild(_ context.Context, raw json.RawMessage) (any, error) {
	var p GraphBuildParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo is required"}
	}
	abs, err := filepath.Abs(p.Repo)
	if err != nil {
		return nil, err
	}

	d.graphRegistry.Evict(abs)

	// Gather available domain stores
	src := graph.Sources{}

	// Code store
	if codeEntry, err := d.registry.Get(d.scryHome(), abs); err == nil {
		src.Code = codeEntry.Store
	}

	// Git store
	if gitEntry, err := d.gitRegistry.Get(d.scryHome(), abs); err == nil {
		src.Git = gitEntry.Store
	}

	// Schema store
	if schemaEntry, err := d.schemaRegistry.Get(d.scryHome(), abs); err == nil {
		src.Schema = schemaEntry.Store
	}

	// HTTP store
	d.proxyMu.Lock()
	src.HTTP = d.httpStore
	d.proxyMu.Unlock()

	start := time.Now()
	manifest, err := graph.Build(d.scryHome(), abs, src)
	if err != nil {
		return nil, fmt.Errorf("graph build: %w", err)
	}
	elapsed := time.Since(start)

	layout := graph.Layout(d.scryHome(), abs)
	st, err := graphstore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("reopen graph store: %w", err)
	}
	d.graphRegistry.Put(&GraphEntry{RepoPath: abs, Layout: layout, Store: st})

	return &GraphBuildResult{
		Repo:        abs,
		Status:      manifest.Status,
		NodeCount:   manifest.NodeCount,
		EdgeCount:   manifest.EdgeCount,
		Communities: manifest.Communities,
		ElapsedMs:   elapsed.Milliseconds(),
	}, nil
}

// --- graph.query (search nodes) ---

type GraphQueryParams struct {
	Repo  string `json:"repo"`
	Query string `json:"query"`
}

func (d *Daemon) handleGraphQuery(_ context.Context, raw json.RawMessage) (any, error) {
	var p GraphQueryParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" || p.Query == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo and query are required"}
	}
	entry, err := d.graphRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}

	nodes, err := entry.Store.SearchNodes(p.Query)
	if err != nil {
		return nil, err
	}

	type nodeWithNeighbors struct {
		graphstore.NodeRecord
		Neighbors []string `json:"neighbors"`
		Degree    int      `json:"degree"`
	}

	var results []nodeWithNeighbors
	for _, n := range nodes {
		neighbors, _ := entry.Store.GetNeighbors(n.Key())
		results = append(results, nodeWithNeighbors{
			NodeRecord: n,
			Neighbors:  neighbors,
			Degree:     len(neighbors),
		})
	}
	if results == nil {
		results = []nodeWithNeighbors{}
	}

	return map[string]any{
		"query":   p.Query,
		"matches": results,
		"total":   len(results),
	}, nil
}

// --- graph.path ---

type GraphPathParams struct {
	Repo string `json:"repo"`
	From string `json:"from"`
	To   string `json:"to"`
}

func (d *Daemon) handleGraphPath(_ context.Context, raw json.RawMessage) (any, error) {
	var p GraphPathParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" || p.From == "" || p.To == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo, from, and to are required"}
	}
	entry, err := d.graphRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}

	result, err := graph.FindPath(entry.Store, p.From, p.To)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// --- graph.report ---

type GraphReportParams struct {
	Repo string `json:"repo"`
}

func (d *Daemon) handleGraphReport(_ context.Context, raw json.RawMessage) (any, error) {
	var p GraphReportParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo is required"}
	}
	entry, err := d.graphRegistry.Get(d.scryHome(), p.Repo)
	if err != nil {
		return nil, err
	}

	report, err := entry.Store.GetReport()
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, &rpc.Error{Code: rpc.CodeInternalError, Message: "no graph report — run `scry graph build` first"}
	}
	return report, nil
}

// rebuildGraphAsync is the PostReindexFunc callback. It debounces graph
// rebuilds: if a graph was built less than 30s ago, it schedules a delayed
// rebuild instead of running immediately.
func (d *Daemon) rebuildGraphAsync(repoPath string) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return
	}

	// Only rebuild if the graph was previously built for this repo.
	layout := graph.Layout(d.scryHome(), abs)
	if _, err := os.Stat(layout.ManifestPath); errors.Is(err, os.ErrNotExist) {
		return
	}

	go func() {
		fmt.Fprintf(os.Stderr, "scry: rebuilding graph for %s (post-reindex)\n", abs)
		start := time.Now()

		src := graph.Sources{}
		if codeEntry, err := d.registry.Get(d.scryHome(), abs); err == nil {
			src.Code = codeEntry.Store
		}
		if gitEntry, err := d.gitRegistry.Get(d.scryHome(), abs); err == nil {
			src.Git = gitEntry.Store
		}
		if schemaEntry, err := d.schemaRegistry.Get(d.scryHome(), abs); err == nil {
			src.Schema = schemaEntry.Store
		}
		d.proxyMu.Lock()
		src.HTTP = d.httpStore
		d.proxyMu.Unlock()

		d.graphRegistry.Evict(abs)
		manifest, err := graph.Build(d.scryHome(), abs, src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: graph rebuild %s failed: %v\n", abs, err)
			return
		}

		st, err := graphstore.Open(layout.BadgerDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: reopen graph store %s: %v\n", abs, err)
			return
		}
		d.graphRegistry.Put(&GraphEntry{RepoPath: abs, Layout: layout, Store: st})

		fmt.Fprintf(os.Stderr, "scry: graph rebuilt %s in %s (%d nodes, %d edges, %d communities)\n",
			abs, time.Since(start).Round(time.Millisecond),
			manifest.NodeCount, manifest.EdgeCount, manifest.Communities)
	}()
}

// graphStatusEntries returns the graph index status for all repos.
func (d *Daemon) graphStatusEntries() []map[string]any {
	var entries []map[string]any
	for _, e := range d.graphRegistry.Snapshot() {
		entry := map[string]any{"repo": e.RepoPath, "domain": "graph"}
		if m, err := graph.LoadManifest(e.Layout); err == nil {
			entry["indexed_at"] = m.IndexedAt
			entry["node_count"] = m.NodeCount
			entry["edge_count"] = m.EdgeCount
			entry["communities"] = m.Communities
		}
		entries = append(entries, entry)
	}

	reposDir := filepath.Join(d.scryHome(), "repos")
	dirs, _ := os.ReadDir(reposDir)
	seen := map[string]bool{}
	for _, e := range d.graphRegistry.Snapshot() {
		seen[e.RepoPath] = true
	}
	for _, dir := range dirs {
		manifestPath := filepath.Join(reposDir, dir.Name(), "graph", "manifest.json")
		if m, err := graph.LoadManifest(graph.RepoLayout{ManifestPath: manifestPath}); err == nil {
			if !seen[m.RepoPath] {
				entries = append(entries, map[string]any{
					"repo":        m.RepoPath,
					"domain":      "graph",
					"indexed_at":  m.IndexedAt,
					"node_count":  m.NodeCount,
					"edge_count":  m.EdgeCount,
					"communities": m.Communities,
				})
			}
		}
	}
	return entries
}
