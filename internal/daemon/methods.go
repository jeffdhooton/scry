package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/query"
	"github.com/jeffdhooton/scry/internal/rpc"
	"github.com/jeffdhooton/scry/internal/store"
)

// scryHome is the parent of the daemon's socket directory. The Layout's Home
// field is exactly that.
func (d *Daemon) scryHome() string { return d.layout.Home }

// registerMethods wires every supported RPC method into the server. Method
// names mirror the CLI subcommands one-to-one.
func (d *Daemon) registerMethods() {
	d.server.Register("refs", d.handleQuery(query.Refs))
	d.server.Register("defs", d.handleQuery(query.Defs))
	d.server.Register("callers", d.handleQuery(query.Callers))
	d.server.Register("callees", d.handleQuery(query.Callees))
	d.server.Register("impls", d.handleQuery(query.Impls))
	d.server.Register("init", d.handleInit)
	d.server.Register("status", d.handleStatus)
	d.server.Register("shutdown", d.handleShutdown)
	d.server.Register("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]any{"ok": true, "pid": os.Getpid()}, nil
	})
}

// QueryParams is the shared envelope for refs/defs/symbols/etc. Repo is the
// absolute path to the repo whose index should answer the query.
type QueryParams struct {
	Repo string `json:"repo"`
	Name string `json:"name"`
}

func (d *Daemon) handleQuery(fn func(*store.Store, string) (*query.Result, error)) rpc.HandlerFunc {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var p QueryParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
		}
		if p.Repo == "" || p.Name == "" {
			return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo and name are required"}
		}
		entry, err := d.registry.Get(d.scryHome(), p.Repo)
		if err != nil {
			return nil, err
		}
		return fn(entry.Store, p.Name)
	}
}

// InitParams instructs the daemon to (re)index a repo.
type InitParams struct {
	Repo string `json:"repo"`
}

// InitResult mirrors the manifest plus a wall-clock duration measured by the
// daemon (not by the CLI).
type InitResult struct {
	Repo      string      `json:"repo"`
	Languages []string    `json:"languages"`
	Status    string      `json:"status"`
	Stats     interface{} `json:"stats"`
	ElapsedMs int64       `json:"elapsed_ms"`
}

func (d *Daemon) handleInit(ctx context.Context, raw json.RawMessage) (any, error) {
	var p InitParams
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

	// If the registry already has this repo open, close it before reindexing
	// so the build can wipe and rewrite BadgerDB without lock contention.
	d.registry.Evict(abs)

	start := time.Now()
	manifest, err := index.Build(ctx, d.scryHome(), abs)
	if err != nil {
		return nil, fmt.Errorf("index build: %w", err)
	}
	elapsed := time.Since(start)

	// Re-open the freshly built store and put it into the registry so the next
	// query against this repo doesn't pay the open cost.
	layout := index.Layout(d.scryHome(), abs)
	st, err := store.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("reopen store after init: %w", err)
	}
	d.registry.Put(&Entry{RepoPath: abs, Layout: layout, Store: st})

	// Start watching this repo so future edits trigger background reindex.
	if err := d.watcher.Watch(ctx, abs); err != nil {
		fmt.Fprintf(os.Stderr, "scry: start watcher for %s: %v\n", abs, err)
	}

	return &InitResult{
		Repo:      manifest.RepoPath,
		Languages: manifest.Languages,
		Status:    manifest.Status,
		Stats:     manifest.Stats,
		ElapsedMs: elapsed.Milliseconds(),
	}, nil
}

// StatusParams is empty for "all repos" or specifies a repo for one-repo
// status.
type StatusParams struct {
	Repo string `json:"repo,omitempty"`
}

// StatusResult is the daemon's view of the world.
type StatusResult struct {
	PID     int                `json:"pid"`
	Uptime  string             `json:"uptime,omitempty"`
	Repos   []*RepoStatusEntry `json:"repos"`
	Version string             `json:"version,omitempty"`
}

type RepoStatusEntry struct {
	Repo      string    `json:"repo"`
	Status    string    `json:"status"`
	Languages []string  `json:"languages,omitempty"`
	IndexedAt time.Time `json:"indexed_at,omitempty"`
}

func (d *Daemon) handleStatus(_ context.Context, raw json.RawMessage) (any, error) {
	res := &StatusResult{PID: os.Getpid()}

	// Look at every repo we know about — both the in-memory registry and any
	// repos that have a manifest on disk that the daemon hasn't loaded yet.
	seen := map[string]bool{}
	for _, e := range d.registry.Snapshot() {
		seen[e.RepoPath] = true
		manifest, err := index.LoadManifest(e.Layout)
		if err != nil {
			continue
		}
		res.Repos = append(res.Repos, &RepoStatusEntry{
			Repo:      e.RepoPath,
			Status:    manifest.Status,
			Languages: manifest.Languages,
			IndexedAt: manifest.IndexedAt,
		})
	}

	// Best-effort scan of the on-disk repos directory so the user sees indexed
	// repos even before they've been queried in this daemon's lifetime.
	reposDir := filepath.Join(d.scryHome(), "repos")
	entries, _ := os.ReadDir(reposDir)
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
		if seen[m.RepoPath] {
			continue
		}
		res.Repos = append(res.Repos, &RepoStatusEntry{
			Repo:      m.RepoPath,
			Status:    m.Status,
			Languages: m.Languages,
			IndexedAt: m.IndexedAt,
		})
	}
	return res, nil
}

// handleShutdown asks the daemon to begin a clean shutdown. Reply is sent
// before the actual shutdown so the client doesn't see a connection-reset
// error.
func (d *Daemon) handleShutdown(_ context.Context, _ json.RawMessage) (any, error) {
	go func() {
		// Brief delay so the server has time to flush the response back to the
		// client before we tear down the listener.
		time.Sleep(50 * time.Millisecond)
		d.mu.Lock()
		ln := d.listener
		d.mu.Unlock()
		if ln != nil {
			_ = ln.Close()
		}
	}()
	return map[string]any{"ok": true}, nil
}
