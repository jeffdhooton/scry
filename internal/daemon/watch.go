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

// PostReindexFunc is called after a successful code reindex with the repo path.
// The daemon uses this to trigger a background graph rebuild.
type PostReindexFunc func(repoPath string)

// Watcher manages one fsnotify watcher per indexed repo and triggers a
// background reindex on relevant filesystem changes.
//
// Why background full reindex instead of single-file incremental: scip-typescript
// (and most SCIP indexers) do not expose a single-file index mode — type
// resolution is project-wide. The only correctness-preserving option today is
// to re-run the indexer over the whole repo. We do that on a goroutine so
// the watch loop keeps reading events.
//
// The reindex uses build-into-temp-dir: index.BuildIntoTemp writes to
// `<storage>/index.db.next/` while the live store at `<storage>/index.db/`
// keeps serving queries. Once the build finishes, a tiny critical section
// (~milliseconds: close + two renames + open) atomically swaps the new
// directory into place. Total query unavailability collapses from ~3-15s
// per reindex to a sub-50ms gap.
type Watcher struct {
	scryHome     string
	registry     *Registry
	postReindex  PostReindexFunc

	mu       sync.Mutex
	watchers map[string]*repoWatcher
}

func NewWatcher(scryHome string, registry *Registry) *Watcher {
	return &Watcher{
		scryHome:    scryHome,
		registry:    registry,
		watchers:    map[string]*repoWatcher{},
	}
}

// SetPostReindex sets the callback invoked after each successful reindex.
func (w *Watcher) SetPostReindex(fn PostReindexFunc) {
	w.postReindex = fn
}

// Watch starts watching repoPath. If a watcher already exists for this repo,
// it's a no-op.
func (w *Watcher) Watch(ctx context.Context, repoPath string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.watchers[repoPath]; ok {
		return nil
	}
	rw, err := newRepoWatcher(ctx, w.scryHome, repoPath, w.registry, w.postReindex)
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
	repoPath    string
	scryHome    string
	registry    *Registry
	postReindex PostReindexFunc

	fsw    *fsnotify.Watcher
	cancel context.CancelFunc
	done   chan struct{}

	lastReindex time.Time
}

func newRepoWatcher(parent context.Context, scryHome, repoPath string, reg *Registry, postReindex PostReindexFunc) (*repoWatcher, error) {
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
		repoPath:    repoPath,
		scryHome:    scryHome,
		registry:    reg,
		postReindex: postReindex,
		fsw:         fsw,
		cancel:      cancel,
		done:        make(chan struct{}),
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

		// Phase 1 (~3-15s): build the new BadgerDB into a sibling temp dir
		// while the live store keeps serving queries. This is the bulk of the
		// time spent and is fully concurrent with reads.
		manifest, nextLayout, err := index.BuildIntoTemp(ctx, rw.scryHome, rw.repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: reindex %s failed: %v\n", rw.repoPath, err)
			// Leave the temp dir on disk so a developer can inspect it.
			// The next successful run will RemoveAll on entry.
			return
		}

		// Phase 2 (~ms): atomically swap the new directory into place. The
		// query window during which the registry has no entry is just
		// (Evict + 2 renames + Open), typically under 50ms even on a slow
		// disk.
		swapStart := time.Now()
		liveLayout := index.Layout(rw.scryHome, rw.repoPath)
		trash, err := rw.registry.SwapNext(liveLayout, nextLayout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: swap reindex %s failed: %v\n", rw.repoPath, err)
			return
		}

		// Phase 3 (background, no observer): drop the trashed old directory.
		if trash != "" {
			go func(p string) {
				if err := os.RemoveAll(p); err != nil {
					fmt.Fprintf(os.Stderr, "scry: cleanup trash %s: %v\n", p, err)
				}
			}(trash)
		}

		fmt.Fprintf(os.Stderr, "scry: reindexed %s in %s (swap %s, %d docs, %d refs)\n",
			rw.repoPath,
			time.Since(rw.lastReindex).Round(time.Millisecond),
			time.Since(swapStart).Round(time.Millisecond),
			manifest.Stats.Documents,
			manifest.Stats.References,
		)

		if rw.postReindex != nil {
			rw.postReindex(rw.repoPath)
		}
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
