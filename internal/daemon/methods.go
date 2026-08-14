package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gitindex "github.com/jeffdhooton/scry/internal/git/index"
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
	d.server.Register("tests", d.handleQuery(query.Tests))
	d.server.Register("init", d.handleInit)
	d.server.Register("status", d.handleStatus)
	d.server.Register("shutdown", d.handleShutdown)
	d.server.Register("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]any{"ok": true, "pid": os.Getpid()}, nil
	})
	d.registerMemoryMethods()
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
	Repo      string                `json:"repo"`
	Languages []string              `json:"languages"`
	Status    string                `json:"status"`
	Stats     interface{}           `json:"stats"`
	ElapsedMs int64                 `json:"elapsed_ms"`
	Indexers  []index.IndexerResult `json:"indexers,omitempty"`
}

func (d *Daemon) handleInit(ctx context.Context, raw json.RawMessage) (any, error) {
	var p InitParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if p.Repo == "" {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "repo is required"}
	}
	// Canonicalize (symlinks, on-disk casing) so a repo indexed via a
	// symlinked or differently-cased path keys to one index, not several.
	abs := canonicalPath(p.Repo)

	start := time.Now()

	// Build into a temp directory so the live store (if any) keeps serving
	// queries throughout the rebuild. This avoids the BadgerDB directory lock
	// contention that happened when Evict+Build raced with in-flight queries.
	manifest, nextLayout, err := index.BuildIntoTemp(ctx, d.scryHome(), abs)
	if err != nil {
		return nil, fmt.Errorf("index build: %w", err)
	}

	// Atomically swap the new store into the live position.
	liveLayout := index.Layout(d.scryHome(), abs)
	trash, err := d.registry.SwapNext(liveLayout, nextLayout)
	if err != nil {
		return nil, fmt.Errorf("swap after init: %w", err)
	}
	if trash != "" {
		go func(p string) { _ = os.RemoveAll(p) }(trash)
	}

	elapsed := time.Since(start)

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
		Indexers:  manifest.Indexers,
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
	Git     []map[string]any   `json:"git,omitempty"`
	Schema  []map[string]any   `json:"schema,omitempty"`
	HTTP    map[string]any     `json:"http,omitempty"`
	Graph   []map[string]any   `json:"graph,omitempty"`
	Version string             `json:"version,omitempty"`
}

type RepoStatusEntry struct {
	Repo string `json:"repo"`
	// Status is what the build recorded in the manifest: "ready" or
	// "partial". EffectiveStatus folds in the two signals derived at report
	// time — see index.EffectiveStatus for the precedence.
	Status string `json:"status"`
	// EffectiveStatus is display-only: "ready" | "partial" | "empty" |
	// "stale". Never written back to the manifest.
	EffectiveStatus string `json:"effective_status"`
	// Stale reports that the index no longer matches the repo — HEAD moved
	// since the build, or (with no HEAD available) a source file is newer
	// than the index.
	Stale bool `json:"stale"`
	// EmptyLanguages are primary languages whose indexer claimed success and
	// produced no symbols anyway.
	EmptyLanguages []string              `json:"empty_languages,omitempty"`
	Languages      []string              `json:"languages,omitempty"`
	IndexedAt      time.Time             `json:"indexed_at,omitempty"`
	Indexers       []index.IndexerResult `json:"indexers,omitempty"`
}

// statusHeadTimeout caps HEAD resolution across a whole status call. `scry
// status` is on the hot path for agents; one wedged git invocation must
// degrade the staleness signal, not the response.
const statusHeadTimeout = 2 * time.Second

// headCache resolves each repo's HEAD at most once per status call. Repos
// reach the status handler from two places (the in-memory registry and the
// on-disk scan), and a future caller may ask for the same repo twice — the
// cache makes "one git call per repo" a property of the type rather than of
// the loop that happens to be written today.
type headCache struct {
	ctx     context.Context
	entries map[string]string
}

func newHeadCache(ctx context.Context) *headCache {
	return &headCache{ctx: ctx, entries: map[string]string{}}
}

// head returns the repo's current HEAD, or "" when it has none (not a git
// checkout, no commits yet) or git couldn't be run. Errors are swallowed
// deliberately: no HEAD means the caller falls back to comparing mtimes.
func (h *headCache) head(repoPath string) string {
	if head, ok := h.entries[repoPath]; ok {
		return head
	}
	head, err := gitindex.HeadCommit(h.ctx, repoPath)
	if err != nil {
		head = ""
	}
	h.entries[repoPath] = head
	return head
}

// repoStatusEntry builds one status row, deriving the stale and empty signals
// from the manifest plus the live repo. Neither signal reads the store or
// triggers a reindex.
func repoStatusEntry(m *index.Manifest, heads *headCache) *RepoStatusEntry {
	head := heads.head(m.RepoPath)
	// Only pay for a tree walk when there is no commit to compare against.
	var newestSource time.Time
	if head == "" {
		newestSource = index.NewestSourceMTime(m.RepoPath)
	}
	stale := index.IsStale(m, head, newestSource)
	// Reads the manifest's own per-language counts — no store access, no git,
	// no walk.
	emptyLanguages := index.EmptyLanguages(m)
	return &RepoStatusEntry{
		Repo:            m.RepoPath,
		Status:          m.Status,
		EffectiveStatus: index.EffectiveStatus(m, stale, emptyLanguages),
		Stale:           stale,
		EmptyLanguages:  emptyLanguages,
		Languages:       m.Languages,
		IndexedAt:       m.IndexedAt,
		Indexers:        m.Indexers,
	}
}

func (d *Daemon) handleStatus(ctx context.Context, raw json.RawMessage) (any, error) {
	res := &StatusResult{PID: os.Getpid()}

	headCtx, cancel := context.WithTimeout(ctx, statusHeadTimeout)
	defer cancel()
	heads := newHeadCache(headCtx)

	// Look at every repo we know about — both the in-memory registry and any
	// repos that have a manifest on disk that the daemon hasn't loaded yet.
	seen := map[string]bool{}
	for _, e := range d.registry.Snapshot() {
		seen[e.RepoPath] = true
		manifest, err := index.LoadManifest(e.Layout)
		if err != nil {
			continue
		}
		// The manifest's recorded repo path can be empty on hand-written or
		// very old manifests; the registry knows the real one.
		if manifest.RepoPath == "" {
			manifest.RepoPath = e.RepoPath
		}
		res.Repos = append(res.Repos, repoStatusEntry(manifest, heads))
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
		res.Repos = append(res.Repos, repoStatusEntry(&m, heads))
	}
	res.Git = d.gitStatusEntries()
	res.Schema = d.schemaStatusEntries()
	res.HTTP = d.httpStatusEntry()
	res.Graph = d.graphStatusEntries()

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
