package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/store"
)

// debounceWindow is how long we wait after the last filesystem event before
// triggering a reindex. Long enough to coalesce editor save sequences (write
// + rename + chmod) and burst saves across multiple files in a refactor; short
// enough that the staleness ceiling stays sub-second.
const debounceWindow = 300 * time.Millisecond

// reindexCooldown is the minimum spacing between two reindex runs for the
// same repo. Prevents rapid-fire saves from queuing back-to-back rebuilds —
// the second build re-reads everything anyway.
const reindexCooldown = 2 * time.Second

// Watcher manages one fsnotify watcher per indexed repo and triggers a
// background reindex on relevant filesystem changes.
//
// Why background full reindex instead of single-file incremental: scip-typescript
// (and most SCIP indexers) do not expose a single-file index mode — type
// resolution is project-wide. The only correctness-preserving option today is
// to re-run the indexer over the whole repo. We do that on a goroutine so
// the watch loop keeps reading events.
//
// Known limitation: BadgerDB takes an exclusive directory lock, so during the
// reindex window (~3s for a 50k-LOC TS repo) queries against the same repo
// see "not indexed yet" until the new store is opened and Put back into the
// registry. Acceptable for v1; the long-term fix is build-into-temp-dir +
// atomic rename, which is enough complexity to justify deferring until we
// have data showing the gap matters.
type Watcher struct {
	scryHome string
	registry *Registry

	mu       sync.Mutex
	watchers map[string]*repoWatcher
}

func NewWatcher(scryHome string, registry *Registry) *Watcher {
	return &Watcher{
		scryHome: scryHome,
		registry: registry,
		watchers: map[string]*repoWatcher{},
	}
}

// Watch starts watching repoPath. If a watcher already exists for this repo,
// it's a no-op.
func (w *Watcher) Watch(ctx context.Context, repoPath string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.watchers[repoPath]; ok {
		return nil
	}
	rw, err := newRepoWatcher(ctx, w.scryHome, repoPath, w.registry)
	if err != nil {
		return err
	}
	w.watchers[repoPath] = rw
	return nil
}

// Unwatch stops watching one repo and releases its fsnotify resources.
func (w *Watcher) Unwatch(repoPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if rw, ok := w.watchers[repoPath]; ok {
		rw.Close()
		delete(w.watchers, repoPath)
	}
}

// Close shuts down every watcher.
func (w *Watcher) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, rw := range w.watchers {
		rw.Close()
	}
	w.watchers = map[string]*repoWatcher{}
}

// repoWatcher is one fsnotify watcher tied to one repo, plus the goroutine
// that debounces events and triggers reindex.
type repoWatcher struct {
	repoPath string
	scryHome string
	registry *Registry

	fsw    *fsnotify.Watcher
	cancel context.CancelFunc
	done   chan struct{}

	lastReindex time.Time
}

func newRepoWatcher(parent context.Context, scryHome, repoPath string, reg *Registry) (*repoWatcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify new: %w", err)
	}
	if err := addRepoToWatcher(fsw, repoPath); err != nil {
		_ = fsw.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	rw := &repoWatcher{
		repoPath: repoPath,
		scryHome: scryHome,
		registry: reg,
		fsw:      fsw,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	go rw.run(ctx)
	return rw, nil
}

// maxWatchedDirs caps the per-repo directory count so a runaway repo can't
// exhaust the daemon's fd budget. Hit roughly the same number Apple's
// Spotlight watches by default.
const maxWatchedDirs = 2048

// addRepoToWatcher recursively adds every non-ignored directory under
// repoPath to the fsnotify watcher. fsnotify is per-directory on Linux/macOS
// so we have to walk it ourselves.
//
// Skipping is layered:
//   - exact-name skip list (node_modules, vendor, .git, ...)
//   - any directory beginning with '.' (Spotlight, IDE, hidden caches)
//   - any directory under a Laravel-style storage subtree
//   - hard cap on total watched dirs (warns once when hit)
func addRepoToWatcher(fsw *fsnotify.Watcher, repoPath string) error {
	skip := repoSkipDirs()
	added := 0
	hitCap := false
	return filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != repoPath {
			if skip[name] {
				return filepath.SkipDir
			}
			// Anything starting with a '.' is hidden infrastructure
			// (.git, .vscode, .next, .turbo, .idea, .gradle, .pnpm-store ...).
			if len(name) > 0 && name[0] == '.' {
				return filepath.SkipDir
			}
		}
		if added >= maxWatchedDirs {
			if !hitCap {
				fmt.Fprintf(os.Stderr, "scry: watcher reached %d-dir cap on %s; further dirs will not get incremental updates\n", maxWatchedDirs, repoPath)
				hitCap = true
			}
			return filepath.SkipDir
		}
		if err := fsw.Add(path); err != nil {
			// Best-effort: a single ENOSPC or transient EACCES on a subdir
			// shouldn't kill the whole watch.
			fmt.Fprintf(os.Stderr, "scry: watcher add %s: %v\n", path, err)
			return nil
		}
		added++
		return nil
	})
}

// repoSkipDirs is the exact-name skip set. It covers package managers, build
// outputs, runtime caches, and the storage trees that Laravel/Rails-style
// frameworks dump tens of thousands of files into.
func repoSkipDirs() map[string]bool {
	return map[string]bool{
		// Package managers and dependency caches
		"node_modules": true,
		"vendor":       true,
		"bower_components": true,
		// Build outputs
		"dist":         true,
		"build":        true,
		"out":          true,
		"target":       true,
		"public":       true,
		"_site":        true,
		// Test / coverage outputs
		"coverage":     true,
		// Framework runtime trees (Laravel/Rails)
		"storage":      true,
		"bootstrap":    true,
		"log":          true,
		"logs":         true,
		"tmp":          true,
		"cache":        true,
		// Python venv detritus
		"__pycache__":  true,
		"venv":         true,
		".venv":        true,
		// Generated assets
		"generated":    true,
	}
}

func (rw *repoWatcher) run(ctx context.Context) {
	defer close(rw.done)
	defer rw.fsw.Close()

	var debounceTimer *time.Timer
	pending := false
	fire := make(chan struct{}, 1)

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-rw.fsw.Events:
			if !ok {
				return
			}
			if !rw.relevantEvent(ev) {
				continue
			}
			pending = true
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceWindow, func() {
				select {
				case fire <- struct{}{}:
				default:
				}
			})

			// If the event created a new directory, add it to the watcher so
			// future files inside it generate events too. Best-effort.
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					name := filepath.Base(ev.Name)
					if !repoSkipDirs()[name] {
						_ = rw.fsw.Add(ev.Name)
					}
				}
			}
		case err, ok := <-rw.fsw.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "scry: watcher error in %s: %v\n", rw.repoPath, err)
		case <-fire:
			if !pending {
				continue
			}
			pending = false
			rw.maybeReindex(ctx)
		}
	}
}

// relevantEvent filters the noise: editor temp files, dotfiles, and files in
// ignored directories.
func (rw *repoWatcher) relevantEvent(ev fsnotify.Event) bool {
	if ev.Op == 0 {
		return false
	}
	name := filepath.Base(ev.Name)
	// Editor swap and lock files
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") {
		return false
	}
	if strings.HasSuffix(name, ".swp") || strings.HasSuffix(name, ".swx") {
		return false
	}
	// Inside an ignored directory anywhere in the path
	rel, err := filepath.Rel(rw.repoPath, ev.Name)
	if err != nil {
		return false
	}
	for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
		if repoSkipDirs()[segment] {
			return false
		}
	}
	// Only react to source-file extensions we actually index
	switch strings.ToLower(filepath.Ext(name)) {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".go", ".php":
		return true
	}
	return false
}

func (rw *repoWatcher) maybeReindex(ctx context.Context) {
	if time.Since(rw.lastReindex) < reindexCooldown {
		// Schedule one more attempt after the cooldown so we still pick up
		// changes that arrived during the cooldown window.
		time.AfterFunc(reindexCooldown, func() { rw.maybeReindex(ctx) })
		return
	}
	rw.lastReindex = time.Now()

	// The reindex itself blocks for seconds; do it on a goroutine so the
	// debounce loop keeps reading events. We rely on rw.lastReindex to keep
	// concurrent runs from stacking up.
	go func() {
		fmt.Fprintf(os.Stderr, "scry: reindexing %s (file change detected)\n", rw.repoPath)
		// BadgerDB takes an exclusive lock per directory. Evict the live store
		// from the registry so its handle is closed before index.Build opens
		// its own.
		rw.registry.Evict(rw.repoPath)
		manifest, err := index.Build(ctx, rw.scryHome, rw.repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: reindex %s failed: %v\n", rw.repoPath, err)
			return
		}
		// Atomic swap: open the freshly built store and replace the registry
		// entry. The old store is closed by Registry.Put through Evict.
		layout := index.Layout(rw.scryHome, rw.repoPath)
		st, err := store.Open(layout.BadgerDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: reopen store after reindex %s: %v\n", rw.repoPath, err)
			return
		}
		rw.registry.Evict(rw.repoPath)
		rw.registry.Put(&Entry{RepoPath: rw.repoPath, Layout: layout, Store: st})
		fmt.Fprintf(os.Stderr, "scry: reindexed %s in %s (%d docs, %d refs)\n",
			rw.repoPath,
			time.Since(rw.lastReindex).Round(time.Millisecond),
			manifest.Stats.Documents,
			manifest.Stats.References,
		)
	}()
}

func (rw *repoWatcher) Close() {
	rw.cancel()
	select {
	case <-rw.done:
	case <-time.After(2 * time.Second):
	}
}

// Sentinel returned when the watch loop sees a closed channel during shutdown.
var errWatcherClosed = errors.New("watcher closed")
