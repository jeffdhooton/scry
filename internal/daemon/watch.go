package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	scryHome    string
	registry    *Registry
	postReindex PostReindexFunc

	// budget bounds the descriptors every repo watcher holds between them.
	// Without it the only limit is maxWatchedDirs, which counts directories
	// and so does not bound descriptors at all on the kqueue backends.
	budget *fdBudget

	mu       sync.Mutex
	watchers map[string]*repoWatcher
	// lastOnDemand throttles re-watching repos the budget keeps evicting.
	lastOnDemand map[string]time.Time
	// clock orders watchers for LRU eviction. Incremented on every Watch,
	// which the daemon calls on init and on the first query for a repo, so
	// the repos being worked in are the ones that keep their watches.
	clock uint64
}

func NewWatcher(scryHome string, registry *Registry) *Watcher {
	return &Watcher{
		scryHome:     scryHome,
		registry:     registry,
		watchers:     map[string]*repoWatcher{},
		lastOnDemand: map[string]time.Time{},
		budget:       newFDBudget(defaultWatchFDBudget()),
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
	if rw, ok := w.watchers[repoPath]; ok {
		w.touchLocked(rw)
		return nil
	}

	// Free capacity before building the watcher so a newly active repo can
	// take descriptors back from repos nobody has touched in a while. Without
	// this the first repos watched would hold the budget forever and the repo
	// actually being edited would get no incremental reindex.
	for w.budget.available() < minViableWatchFDs && w.evictLRULocked(repoPath) {
	}

	rw, err := newRepoWatcher(ctx, w.scryHome, repoPath, w.registry, w.postReindex, w.budget)
	if err != nil {
		return err
	}
	w.watchers[repoPath] = rw
	w.touchLocked(rw)
	return nil
}

// Watching reports whether repoPath currently has a live watcher. Unlike
// Touch it does not disturb the LRU order — it is for status reporting.
func (w *Watcher) Watching(repoPath string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.watchers[repoPath]
	return ok
}

// WatchStats summarises watch coverage for `scry status` and doctor.
type WatchStats struct {
	// Watchers is how many repos currently hold a live watcher.
	Watchers int `json:"watchers"`
	// BudgetUsed/BudgetCap are the shared descriptor reserve. On the kqueue
	// backends one watched directory costs 1 + len(entries) descriptors, so
	// these numbers move much faster than the watched-directory count.
	BudgetUsed int `json:"budget_used"`
	BudgetCap  int `json:"budget_cap"`
	// ProcessFDs is the daemon's actual open descriptor count, which kqueue
	// grows past BudgetUsed as watched directories are written to. The
	// governor evicts watchers while this exceeds half the soft NOFILE limit.
	ProcessFDs int `json:"process_fds"`
}

// Stats returns a point-in-time snapshot of watch coverage.
func (w *Watcher) Stats() WatchStats {
	w.mu.Lock()
	n := len(w.watchers)
	w.mu.Unlock()
	return WatchStats{
		Watchers:   n,
		BudgetUsed: w.budget.used(),
		BudgetCap:  w.budget.cap(),
		ProcessFDs: processFDCount(),
	}
}

// SetBudget resizes the shared descriptor budget.
func (w *Watcher) SetBudget(limit int) {
	w.budget.setLimit(limit)
}

// Touch records that repoPath was used, reporting whether it is currently
// watched. A false result means the caller should start a watcher for it.
func (w *Watcher) Touch(repoPath string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	rw, ok := w.watchers[repoPath]
	if ok {
		w.touchLocked(rw)
	}
	return ok
}

// onDemandCooldown throttles starting a watcher for a repo that the budget
// keeps evicting. Starting one walks the whole repo, so without this a query
// pattern cycling through more repos than the budget can hold would re-walk a
// repo on every query.
const onDemandCooldown = 2 * time.Minute

// ClaimOnDemand reports whether the caller should start a watcher for repoPath
// now, recording the attempt so the next one waits out the cooldown.
func (w *Watcher) ClaimOnDemand(repoPath string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.watchers[repoPath]; ok {
		return false
	}
	if last, ok := w.lastOnDemand[repoPath]; ok && time.Since(last) < onDemandCooldown {
		return false
	}
	w.lastOnDemand[repoPath] = time.Now()
	return true
}

// HasBudgetFor reports whether the shared budget can take another watcher
// without evicting an existing one.
func (w *Watcher) HasBudgetFor() bool {
	return w.budget.available() >= minViableWatchFDs
}

func (w *Watcher) touchLocked(rw *repoWatcher) {
	w.clock++
	rw.lastUsed = w.clock
}

// evictLRULocked closes the least recently used watcher, returning whether it
// evicted anything. except is never evicted — it is the repo being started.
func (w *Watcher) evictLRULocked(except string) bool {
	var victim string
	var oldest uint64
	for path, rw := range w.watchers {
		if path == except {
			continue
		}
		if victim == "" || rw.lastUsed < oldest {
			victim, oldest = path, rw.lastUsed
		}
	}
	if victim == "" {
		return false
	}
	rw := w.watchers[victim]
	delete(w.watchers, victim)
	rw.Close()
	fmt.Fprintf(os.Stderr, "scry: unwatched %s to free descriptors (watch budget %d/%d)\n",
		victim, w.budget.used(), w.budget.cap())
	return true
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

	// doReindex performs one full build+swap. It is a field only so tests can
	// exercise the run loop's scheduling without a real indexer; production
	// always uses runReindex.
	doReindex func(ctx context.Context)

	// budget and reserved track this watcher's claim on the shared descriptor
	// budget so Close returns exactly what the walk took.
	budget   *fdBudget
	reserved atomic.Int64
	// lastUsed orders watchers for eviction; owned by Watcher.mu.
	lastUsed uint64

	lastReindex time.Time
}

func newRepoWatcher(parent context.Context, scryHome, repoPath string, reg *Registry, postReindex PostReindexFunc, budget *fdBudget) (*repoWatcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify new: %w", err)
	}
	spent := addRepoToWatcher(fsw, repoPath, budget, perRepoWatchFDs(budget))
	ctx, cancel := context.WithCancel(parent)
	rw := &repoWatcher{
		repoPath:    repoPath,
		scryHome:    scryHome,
		registry:    reg,
		postReindex: postReindex,
		fsw:         fsw,
		cancel:      cancel,
		done:        make(chan struct{}),
		budget:      budget,
	}
	rw.reserved.Store(int64(spent))
	rw.doReindex = rw.runReindex
	go rw.run(ctx)
	return rw, nil
}

// minViableWatchFDs is the capacity a new watcher wants before it starts. Below
// this it would watch so little of the repo that it is not worth evicting
// another repo's watch for.
const minViableWatchFDs = 512

// perRepoWatchFDs caps what one repo may take from the shared budget so a
// single huge tree cannot starve every other repo. Half the budget, so at least
// two repos can always hold a watch at once.
func perRepoWatchFDs(budget *fdBudget) int {
	if budget == nil {
		return maxWatchedDirs
	}
	half := budget.cap() / 2
	if half < minViableWatchFDs {
		return minViableWatchFDs
	}
	return half
}

// maxWatchedDirs caps the per-repo directory count so a runaway repo can't
// exhaust the daemon's fd budget. Hit roughly the same number Apple's
// Spotlight watches by default.
const maxWatchedDirs = 2048

// addRepoToWatcher recursively adds every non-ignored directory under
// repoPath to the fsnotify watcher. fsnotify is per-directory on Linux/macOS
// so we have to walk it ourselves. It returns the descriptors it reserved,
// which the caller releases when the watcher closes.
//
// Skipping is layered:
//   - exact-name skip list (node_modules, vendor, .git, ...)
//   - any directory beginning with '.' (Spotlight, IDE, hidden caches)
//   - any directory under a Laravel-style storage subtree
//   - hard cap on total watched dirs (warns once when hit)
//   - the shared descriptor budget, and this repo's share of it
//
// The budget is what actually bounds descriptors. maxWatchedDirs counts
// directories, and on the kqueue backends one directory can cost hundreds of
// descriptors, so the dir cap alone let a single repo hold 20k+ of them.
func addRepoToWatcher(fsw *fsnotify.Watcher, repoPath string, budget *fdBudget, perRepoLimit int) int {
	skip := repoSkipDirs()
	added := 0
	spent := 0
	hitCap := false
	hitBudget := false

	_ = filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
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

		// Charge the descriptors this directory will actually cost before
		// opening anything, and stop the walk once the repo has had its share.
		cost := watchDirFDCost(path)
		if spent+cost > perRepoLimit || !budget.tryReserve(cost) {
			if !hitBudget {
				fmt.Fprintf(os.Stderr, "scry: watcher fd budget reached on %s after %d dirs (%d fds); remaining dirs will not get incremental updates\n", repoPath, added, spent)
				hitBudget = true
			}
			return filepath.SkipAll
		}

		if err := fsw.Add(path); err != nil {
			// Best-effort: a single ENOSPC or transient EACCES on a subdir
			// shouldn't kill the whole watch.
			budget.release(cost)
			fmt.Fprintf(os.Stderr, "scry: watcher add %s: %v\n", path, err)
			return nil
		}
		added++
		spent += cost
		return nil
	})
	return spent
}

// repoSkipDirs is the exact-name skip set. It covers package managers, build
// outputs, runtime caches, and the storage trees that Laravel/Rails-style
// frameworks dump tens of thousands of files into.
func repoSkipDirs() map[string]bool {
	return map[string]bool{
		// Package managers and dependency caches
		"node_modules":     true,
		"vendor":           true,
		"bower_components": true,
		// Build outputs
		"dist":   true,
		"build":  true,
		"out":    true,
		"target": true,
		"public": true,
		"_site":  true,
		// Test / coverage outputs
		"coverage": true,
		// Framework runtime trees (Laravel/Rails)
		"storage":   true,
		"bootstrap": true,
		"log":       true,
		"logs":      true,
		"tmp":       true,
		"cache":     true,
		// Python venv detritus
		"__pycache__": true,
		"venv":        true,
		".venv":       true,
		// Generated assets
		"generated": true,
	}
}

// maxDeferredEventPaths caps how many distinct paths the run loop remembers
// from events that arrive while a reindex is in flight. Enough to cover any
// realistic mid-build edit burst; a runaway event storm just stops being
// recorded, which at worst eats an edit until its next save.
const maxDeferredEventPaths = 256

func (rw *repoWatcher) run(ctx context.Context) {
	defer close(rw.done)
	defer closeWatcher(rw.fsw)

	var debounceTimer *time.Timer
	pending := false
	fire := make(chan struct{}, 1)
	arm := func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(debounceWindow, func() {
			select {
			case fire <- struct{}{}:
			default:
			}
		})
	}

	// inflight guards against the loop that kernel-panicked the machine: a
	// reindex's own filesystem side effects (git re-hashing dirty files, an
	// indexer's temp files) arrive as events and used to re-arm the debounce,
	// so every reindex scheduled the next one forever. While a reindex is in
	// flight we record event paths in deferred instead of arming; when it
	// finishes, deferredNeedsReindex decides whether any of them was a real
	// edit worth one catch-up pass.
	inflight := false
	deferred := map[string]struct{}{}
	reindexDone := make(chan struct{}, 1)

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-rw.fsw.Events:
			if !ok {
				return
			}
			// Watch new directories before the relevance filter runs: a
			// directory name has no source extension, so relevantEvent would
			// drop the event and the new tree would get no incremental updates
			// until the next full reindex.
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					rw.watchNewDir(ev.Name)
				}
			}
			if !rw.relevantEvent(ev) {
				continue
			}
			if inflight {
				if len(deferred) < maxDeferredEventPaths {
					deferred[ev.Name] = struct{}{}
				}
				continue
			}
			pending = true
			arm()
		case err, ok := <-rw.fsw.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "scry: watcher error in %s: %v\n", rw.repoPath, err)
		case <-fire:
			if !pending || inflight {
				continue
			}
			if since := time.Since(rw.lastReindex); since < reindexCooldown {
				// Wait out the cooldown, keeping pending set so this attempt
				// isn't lost.
				time.AfterFunc(reindexCooldown-since, func() {
					select {
					case fire <- struct{}{}:
					default:
					}
				})
				continue
			}
			pending = false
			inflight = true
			rw.lastReindex = time.Now()
			// The reindex blocks for seconds; run it on a goroutine so this
			// loop keeps reading events. inflight keeps a second run from
			// starting until reindexDone arrives.
			go func() {
				if rw.doReindex != nil {
					rw.doReindex(ctx)
				}
				select {
				case reindexDone <- struct{}{}:
				default:
				}
			}()
		case <-reindexDone:
			inflight = false
			if pending || rw.deferredNeedsReindex(deferred) {
				pending = true
				arm()
			}
			deferred = map[string]struct{}{}
		}
	}
}

// deferredNeedsReindex reports whether any event that arrived during the
// just-finished reindex looks like a real edit: the file still exists and its
// mtime is newer than the reindex start. The reindex's own side effects fail
// both tests — a temp file is gone by now, and git's atime-only rehash of a
// dirty file (the observed panic-causer, a Chmod dropped by relevantEvent
// anyway) never advances mtime — so they cannot schedule the next cycle.
func (rw *repoWatcher) deferredNeedsReindex(deferred map[string]struct{}) bool {
	for p := range deferred {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.ModTime().After(rw.lastReindex) {
			return true
		}
	}
	return false
}

// watchNewDir adds a directory that appeared after the initial walk, plus any
// directories already inside it — a git checkout or cp -r creates a populated
// tree, not one mkdir event per level. It applies the same skip rules as
// addRepoToWatcher and charges every addition to the budget: a build that
// creates thousands of directories must not grow the watch set past what the
// initial walk was held to. Best-effort: budget exhaustion or a failed Add
// stops the walk, and the next full reindex picks the tree up.
func (rw *repoWatcher) watchNewDir(root string) {
	skip := repoSkipDirs()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if skip[name] || (len(name) > 0 && name[0] == '.') {
			return filepath.SkipDir
		}
		cost := watchDirFDCost(path)
		if rw.budget != nil && !rw.budget.tryReserve(cost) {
			return filepath.SkipAll
		}
		if err := rw.fsw.Add(path); err != nil {
			if rw.budget != nil {
				rw.budget.release(cost)
			}
			return nil
		}
		rw.reserved.Add(int64(cost))
		return nil
	})
}

// relevantEvent filters the noise: editor temp files, dotfiles, and files in
// ignored directories.
func (rw *repoWatcher) relevantEvent(ev fsnotify.Event) bool {
	// Only ops that can change content matter. A Chmod-only event never does —
	// and on the kqueue backends it is how an atime update arrives, which git
	// causes on every dirty file over 32KB it re-hashes via mmap. That was the
	// observed self-trigger: reindex runs git, git reads a dirty source file,
	// the atime bump comes back as Chmod, and the loop never ends.
	if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
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

// runReindex is the production reindex body, invoked from the run loop's
// reindex goroutine with inflight set. rw.lastReindex was stamped by the run
// loop just before this started.
func (rw *repoWatcher) runReindex(ctx context.Context) {
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
	// If the daemon dies first, the startup sweep removes it instead.
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
}

func (rw *repoWatcher) Close() {
	rw.cancel()
	select {
	case <-rw.done:
	case <-time.After(2 * time.Second):
	}
	// run's deferred fsw.Close has returned the descriptors to the OS by now,
	// so give the budget its capacity back.
	if rw.budget != nil {
		if n := rw.reserved.Swap(0); n > 0 {
			rw.budget.release(int(n))
		}
	}
}

// Sentinel returned when the watch loop sees a closed channel during shutdown.
var errWatcherClosed = errors.New("watcher closed")

// fdGovernorInterval is how often the daemon samples its own descriptor use.
// Frequent enough to catch a build campaign expanding the watch set within a
// minute, cheap enough to be irrelevant: one bounded F_GETFD scan.
const fdGovernorInterval = 30 * time.Second

// StartFDGovernor samples the process's descriptor count and evicts watchers
// while it sits above the high-water mark.
//
// The budget alone is not enough. It charges what a directory costs at Add
// time, but on the kqueue backends fsnotify keeps opening descriptors after
// that: when a watched directory is written to, dirChange walks it and starts
// watching every entry that has appeared since. A build campaign creating files
// across watched trees therefore grows the descriptor set well past what the
// walk reserved — which is how a daemon reached ~131k descriptors and starved
// the rest of the machine of the system-wide file table.
//
// Evicting the least recently used watcher is the right lever because watchers
// are the elastic part of the daemon's descriptor use; the BadgerDB stores and
// the socket are not.
func (w *Watcher) StartFDGovernor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(fdGovernorInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.enforceFDCeiling()
			}
		}
	}()
}

// fdHighWaterShare is the fraction of the soft NOFILE limit above which the
// governor starts evicting watchers.
const fdHighWaterShare = 2 // i.e. one half

func (w *Watcher) enforceFDCeiling() {
	ceiling := softNOFILE() / fdHighWaterShare
	count := processFDCount()
	if count <= ceiling {
		return
	}

	fmt.Fprintf(os.Stderr, "scry: %d open descriptors exceeds ceiling %d; evicting watchers\n", count, ceiling)
	for count > ceiling {
		w.mu.Lock()
		evicted := w.evictLRULocked("")
		w.mu.Unlock()
		if !evicted {
			// Nothing left to give back. Whatever is holding the descriptors
			// is not a repo watcher.
			fmt.Fprintf(os.Stderr, "scry: still %d descriptors open with no watchers left to evict\n", count)
			return
		}
		count = processFDCount()
	}
	fmt.Fprintf(os.Stderr, "scry: descriptor use back to %d (ceiling %d)\n", count, ceiling)
}

// closeWatcher releases every descriptor an fsnotify watcher holds.
//
// fsnotify v1.9.0's kqueue backend did not: Close marked the watcher closed and
// only then called Remove for each watched path, but remove() returned early on
// a closed watcher, so it never reached its unix.Close. Closing a watcher
// therefore leaked every descriptor it held — measured at 1154 of 1155 on a
// mid-sized repo — which made evicting a watcher free budget without freeing
// anything real. Fixed upstream in v1.10.0 ("kqueue: drop watches directly in
// Close()", fsnotify/fsnotify#740), so this is a plain Close again; the
// regression test in watch_fd_test.go guards the property.
func closeWatcher(fsw *fsnotify.Watcher) {
	_ = fsw.Close()
}
