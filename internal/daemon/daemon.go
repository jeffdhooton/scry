// Package daemon is the long-running scry process. It owns the Unix socket,
// the per-repo BadgerDB stores, the file watchers, and the RPC dispatcher.
//
// One daemon per user. The CLI auto-spawns it on first call. See SPEC §4.1.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jeffdhooton/scry/internal/config"
	"github.com/jeffdhooton/scry/internal/git"
	scryhttp "github.com/jeffdhooton/scry/internal/http"
	httpstore "github.com/jeffdhooton/scry/internal/http/store"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/queue"
	"github.com/jeffdhooton/scry/internal/memory/search"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	roomstore "github.com/jeffdhooton/scry/internal/room/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// DefaultShutdownGrace matches docs/DECISIONS.md "Daemon shutdown" — 5s
// from SIGTERM to forceful close.
const DefaultShutdownGrace = 5 * time.Second

// Layout is the on-disk daemon layout under ~/.scry. All paths are absolute.
type Layout struct {
	Home       string // ~/.scry
	SocketPath string // ~/.scry/scryd.sock
	PIDPath    string // ~/.scry/scryd.pid
	LockPath   string // ~/.scry/scryd.lock — process-lifetime flock, see owner.go
	LogPath    string // ~/.scry/scryd.log
}

// LayoutFor builds the layout from a scry home directory.
func LayoutFor(home string) Layout {
	return Layout{
		Home:       home,
		SocketPath: filepath.Join(home, "scryd.sock"),
		PIDPath:    filepath.Join(home, "scryd.pid"),
		LockPath:   filepath.Join(home, "scryd.lock"),
		LogPath:    filepath.Join(home, "scryd.log"),
	}
}

// Daemon is one running scryd process.
type Daemon struct {
	layout         Layout
	registry       *Registry
	gitRegistry    *git.Registry
	schemaRegistry *SchemaRegistry
	graphRegistry  *GraphRegistry
	server         *rpc.Server
	watcher        *Watcher

	mu       sync.Mutex
	listener net.Listener

	proxyMu   sync.Mutex
	proxy     *scryhttp.Proxy
	httpStore *httpstore.Store

	memOnce      sync.Once
	memStore     *memstore.Store
	memErr       error
	memExtractor extract.Extractor
	memQueueMu   sync.Mutex
	memQueue     *queue.Worker
	memQueueWG   sync.WaitGroup
	memGlossary  glossaryCache
	memIndexOnce sync.Once
	// memStop closes when the memory domain shuts down, stopping the
	// background relearn of the vector model.
	memStop     chan struct{}
	memStopOnce sync.Once
	memIndex    *search.Index
	memIndexErr error

	memUIMu  sync.Mutex
	memUISrv *http.Server

	roomOnce sync.Once
	roomSt   *roomstore.Store
	roomErr  error
}

// memoryStore lazily opens the global memory store on first use, guarded by
// a sync.Once so concurrent RPCs never race to open it. If the open fails,
// the error is cached and returned on every subsequent call rather than
// retried (a retry loop against a store that can't open is not useful and
// risks panicking on a nil *Store).
func (d *Daemon) memoryStore() (*memstore.Store, error) {
	d.memOnce.Do(func() {
		d.memStore, d.memErr = memstore.Open(filepath.Join(d.scryHome(), "memory"))
	})
	return d.memStore, d.memErr
}

// buildMemoryExtractor constructs the daemon's extract.Extractor from
// ~/.scry/config.yaml (memory.models) falling back to the environment (see
// extract.ResolveProviders). A nil return means the memory domain is
// dormant — memory.remember still stores episodes but never resolves them
// into facts. A missing key, an unusable provider config, and a malformed
// config file all go dormant rather than failing the daemon's startup; the
// CLI surfaces the same misconfiguration loudly, which is where it gets
// noticed.
func buildMemoryExtractor(scryHome string) extract.Extractor {
	cfg, err := config.Load(scryHome)
	if err != nil {
		log.Printf("memory: extraction DORMANT — %v. Episodes will be stored "+
			"but never resolved into facts. Fix the file and restart the daemon.", err)
		return nil
	}
	if sock := cfg.MemorySocket(); sock != "" {
		// This daemon is a client of the shared store served elsewhere;
		// its own memory store is unused and it must not run a model chain
		// of its own, or the chain would be configured in two places and
		// silently diverge (the laptop kept a dead DeepSeek chain for a day
		// this way).
		log.Printf("memory: extraction OFF — memory.socket in %s points at %s; that daemon runs the chain", config.Path(scryHome), sock)
		return nil
	}
	ps := extract.ResolveProviders(cfg)
	// Dormancy must be loud. A daemon restarted without the key went quietly
	// dormant for a whole day: episodes kept being stored and never resolved
	// into facts, and nothing said so until someone went looking.
	if ps.Dormant() {
		log.Printf("memory: extraction DORMANT — no API key in the environment. " +
			"Episodes will be stored but never resolved into facts. Set the " +
			"provider key and restart the daemon.")
		return nil
	}
	if err := ps.Validate(); err != nil {
		log.Printf("memory: extraction DORMANT — unusable provider config: %v. "+
			"Episodes will be stored but never resolved into facts.", err)
		return nil
	}
	log.Printf("memory: extraction chain (%s): %s", ps.Source, strings.Join(ps.Models(), " -> "))
	if ps.Source == "config.yaml" && (os.Getenv("SCRY_MEMORY_MODEL") != "" || os.Getenv("SCRY_MEMORY_BASE_URL") != "") {
		log.Printf("memory: ignoring SCRY_MEMORY_MODEL / SCRY_MEMORY_BASE_URL — memory.models in %s takes precedence", config.Path(scryHome))
	}
	return extract.NewExtractor(ps)
}

// New constructs a Daemon for the given layout. It does NOT start anything;
// call Run to begin serving.
func New(layout Layout) *Daemon {
	d := &Daemon{
		layout:         layout,
		registry:       NewRegistry(),
		gitRegistry:    git.NewRegistry(),
		schemaRegistry: NewSchemaRegistry(),
		graphRegistry:  NewGraphRegistry(),
		server:         rpc.NewServer(),
		memExtractor:   buildMemoryExtractor(layout.Home),
	}
	d.watcher = NewWatcher(layout.Home, d.registry)
	d.watcher.SetPostReindex(d.rebuildGraphAsync)
	d.registerMethods()
	d.registerGitMethods()
	d.registerSchemaMethods()
	d.registerHTTPMethods()
	d.registerGraphMethods()
	d.registerRoomMethods()
	return d
}

// Run takes ownership of the process: writes the PID file, opens the socket,
// dispatches RPC calls until ctx is cancelled or SIGTERM/SIGINT arrives, then
// performs a graceful shutdown.
//
// Returns nil on clean shutdown, otherwise the first error that broke the run.
func (d *Daemon) Run(ctx context.Context) error {
	// Raise NOFILE soft limit before doing anything else. fsnotify needs far
	// more descriptors than macOS' default 256 — on the kqueue backends it
	// holds one per watched *file*, not just per watched directory, so a
	// single Laravel-class repo can want thousands.
	raiseNOFILE()

	// Size the watcher's descriptor budget from the limit we actually got.
	// New() ran before raiseNOFILE, so the budget it built was sized against
	// the pre-raise soft limit.
	d.watcher.SetBudget(defaultWatchFDBudget())

	if err := os.MkdirAll(d.layout.Home, 0o755); err != nil {
		return fmt.Errorf("ensure home: %w", err)
	}

	// Become the one daemon for this home, or step aside for a healthy one
	// (returns *AlreadyRunningError). An incumbent that holds the lock but
	// does not answer is retired and waited for before we proceed — see
	// owner.go for why a PID file alone was not enough. The lock is the
	// first thing acquired and the last thing released: every other
	// teardown below runs before it, so a successor can only start
	// mutating scryd.sock / scryd.pid once this process is fully gone.
	own, err := acquireOwnership(d.layout, defaultTakeoverPolicy())
	if err != nil {
		return err
	}
	defer own.Release()

	// Stale socket from a previous crash. We hold the lock and no healthy
	// daemon answered, so nothing live is behind this pathname.
	if err := os.Remove(d.layout.SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale socket: %w", err)
	}

	ln, err := net.Listen("unix", d.layout.SocketPath)
	if err != nil {
		return fmt.Errorf("listen unix %s: %w", d.layout.SocketPath, err)
	}
	if err := os.Chmod(d.layout.SocketPath, 0o600); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	d.mu.Lock()
	d.listener = ln
	d.mu.Unlock()

	if err := os.WriteFile(d.layout.PIDPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		_ = ln.Close()
		return fmt.Errorf("write pid file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "scry: daemon pid %d owns %s (lock %s)\n", os.Getpid(), d.layout.SocketPath, d.layout.LockPath)
	defer removeOwnedPIDFile(d.layout, os.Getpid())
	defer os.Remove(d.layout.SocketPath)
	defer d.registry.CloseAll()
	defer d.gitRegistry.CloseAll()
	defer d.schemaRegistry.CloseAll()
	defer d.graphRegistry.CloseAll()
	defer d.closeHTTP()
	defer d.closeRooms()
	defer d.closeMemory()
	defer d.closeMemoryUI()
	defer d.watcher.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Wire up signal handling FIRST, before any code path that could exhaust
	// file descriptors. Earlier versions panicked here with "pipe failed"
	// when bootstrap watchers ate the fd budget and signal.Notify couldn't
	// open its internal self-pipe.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-runCtx.Done():
		}
	}()

	// Start a watcher on demand for any repo a query touches, and keep the
	// watcher LRU ordered by real activity so the repos being worked in keep
	// their watches. Starting one walks the repo, so it runs off the query
	// path; Watch itself is serialized and idempotent.
	d.registry.OnAccess = func(repoPath string) {
		if d.watcher.Touch(repoPath) {
			return
		}
		if !d.watcher.ClaimOnDemand(repoPath) {
			return
		}
		go func() {
			if err := d.watcher.Watch(runCtx, repoPath); err != nil {
				fmt.Fprintf(os.Stderr, "scry: watch on demand %s: %v\n", repoPath, err)
			}
		}()
	}

	// Sweep rotate-then-delete garbage from previous daemon lifetimes before
	// any watcher can start a reindex of its own. Synchronous on purpose: it
	// is cheap when there is nothing to do, and when there IS something to do
	// (a dead daemon left hundreds of GB of index.db.old.* archives) freeing
	// the disk beats coming up a few seconds sooner.
	sweepStaleIndexTrash(d.layout.Home)

	// Bootstrap: start watchers for the most recently indexed repos, as many
	// as the descriptor budget affords. The rest are watched on demand above.
	d.bootstrapWatchers(runCtx)

	// Keep the daemon's descriptor use bounded. The budget covers what the
	// watchers reserve at Add time, but the kqueue backends keep opening
	// descriptors on their own as watched directories gain files, so the
	// actual total has to be watched too.
	d.watcher.StartFDGovernor(runCtx)

	// Live memory graph UI, loopback-only, best-effort: a port conflict here
	// must never keep the daemon itself from coming up.
	d.startMemoryUI(runCtx)

	// Drain queued memory writes for as long as the daemon runs. Stopped by
	// runCtx before closeMemory (deferred above) closes the store.
	d.startMemoryWorker(runCtx)

	serveErr := d.server.Serve(runCtx, ln)
	if serveErr != nil && !errors.Is(serveErr, net.ErrClosed) {
		return serveErr
	}
	return nil
}

// AliveDaemon is the standalone version usable from the CLI to decide whether
// to spawn a fresh daemon. Returns (true, pid) if a daemon is currently
// listening on the socket and the PID file matches.
func AliveDaemon(layout Layout) (bool, int) {
	pidBytes, err := os.ReadFile(layout.PIDPath)
	if err != nil {
		// No PID file; quickly try to dial anyway in case someone left one
		// behind.
		if pingSocket(layout.SocketPath) {
			return true, 0
		}
		return false, 0
	}
	pid, err := strconv.Atoi(string(bytesTrimSpace(pidBytes)))
	if err != nil {
		return false, 0
	}
	if !processAlive(pid) {
		return false, 0
	}
	if !pingSocket(layout.SocketPath) {
		return false, 0
	}
	return true, pid
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 doesn't deliver but does check existence/permissions.
	return proc.Signal(syscall.Signal(0)) == nil
}

func pingSocket(path string) bool {
	conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func bytesTrimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}
