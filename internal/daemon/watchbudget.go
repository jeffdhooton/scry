package daemon

import (
	"sync"
	"syscall"
)

// fdBudget is a shared ceiling on the file descriptors every repo watcher may
// hold between them.
//
// It exists because the per-repo directory cap does not bound descriptors. On
// macOS and the BSDs, fsnotify's kqueue backend needs a descriptor per watched
// *entry*, not per watched directory: adding one directory opens a descriptor
// for it and for every file and subdirectory inside it. A repo of 20
// directories — 1% of maxWatchedDirs — can therefore cost 4000+ descriptors,
// and a daemon watching every registered repo consumed ~131k of them, which is
// enough of the system-wide file table to make unrelated processes fail with
// ENFILE.
//
// The budget is charged at Add time and released when a watcher closes, so
// eviction actually returns capacity.
type fdBudget struct {
	mu    sync.Mutex
	limit int
	spent int
}

func newFDBudget(limit int) *fdBudget {
	if limit < 0 {
		limit = 0
	}
	return &fdBudget{limit: limit}
}

// tryReserve claims n descriptors, reporting whether the budget had room. A
// failed reservation claims nothing.
func (b *fdBudget) tryReserve(n int) bool {
	if n <= 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.spent+n > b.limit {
		return false
	}
	b.spent += n
	return true
}

// release returns n descriptors to the budget.
func (b *fdBudget) release(n int) {
	if n <= 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.spent -= n
	if b.spent < 0 {
		b.spent = 0
	}
}

// used reports the currently reserved descriptor count.
func (b *fdBudget) used() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spent
}

// available reports how much of the budget is unclaimed.
func (b *fdBudget) available() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.limit - b.spent
}

// setLimit changes the ceiling. Used once at startup, after the process has
// raised its NOFILE limit and knows what it actually has to spend.
func (b *fdBudget) setLimit(n int) {
	if n < 0 {
		n = 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.limit = n
}

func (b *fdBudget) cap() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.limit
}

// processFDCount reports how many descriptors this process currently holds.
//
// It probes with F_GETFD rather than reading /dev/fd: the fdesc filesystem on
// macOS reports inconsistent results from a process that is concurrently
// opening and closing descriptors, which is exactly the daemon's steady state.
// The scan is bounded so a huge soft limit cannot turn a sample into a stall.
func processFDCount() int {
	limit := softNOFILE()
	if limit > fdScanCeiling {
		limit = fdScanCeiling
	}
	n := 0
	for fd := uintptr(0); fd < uintptr(limit); fd++ {
		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, uintptr(syscall.F_GETFD), 0); errno == 0 {
			n++
		}
	}
	return n
}

// fdScanCeiling bounds processFDCount's probe loop. Well above any healthy
// descriptor count for this daemon, and cheap enough to sample periodically.
const fdScanCeiling = 262144

// softNOFILE returns the process's current soft descriptor limit, or a
// conservative default if it cannot be read.
func softNOFILE() int {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		return 4096
	}
	if lim.Cur > uint64(fdScanCeiling) {
		return fdScanCeiling
	}
	return int(lim.Cur)
}
