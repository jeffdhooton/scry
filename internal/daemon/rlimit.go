package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// nofileTarget is the soft NOFILE limit the daemon raises itself to.
//
// It is deliberately a fixed target rather than the hard limit. macOS reports a
// hard limit of RLIM_INFINITY, so raising to the hard limit left the process
// with no ceiling at all — and NOFILE was the only thing bounding one process's
// share of kern.maxfiles, which is a *system-wide* resource. A daemon that lost
// that ceiling grew to ~131k descriptors and other processes on the machine
// started failing to open files with ENFILE.
//
// 65536 is far above what a healthy daemon needs (watch budget, BadgerDB
// stores, sockets) and small enough that even several daemons together stay
// well inside a typical kern.maxfiles of ~491k.
const nofileTarget = 65536

// watchFDBudgetShare is the fraction of the soft NOFILE limit the repo watchers
// may hold between them. The rest is left for BadgerDB stores, the RPC socket,
// indexer subprocesses, and headroom.
const watchFDBudgetShare = 4 // i.e. one quarter

const (
	minWatchFDBudget = 2048
	maxWatchFDBudget = 16384
)

// raiseNOFILE raises the soft NOFILE limit toward nofileTarget. fsnotify needs
// far more descriptors than macOS' default soft cap of 256 — on the kqueue
// backends it holds one per watched file, not just per watched directory.
//
// Errors are non-fatal — we log and continue. Worst case the watcher degrades
// and the fd budget sizes itself down, which is preferable to refusing to
// start.
func raiseNOFILE() {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		fmt.Fprintf(os.Stderr, "scry: getrlimit NOFILE: %v\n", err)
		return
	}
	target := uint64(nofileTarget)
	if lim.Max < target {
		target = lim.Max
	}
	if lim.Cur >= target {
		return
	}
	lim.Cur = target
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		fmt.Fprintf(os.Stderr, "scry: setrlimit NOFILE: %v\n", err)
	}
}

// defaultWatchFDBudget sizes the shared watcher budget from the descriptor
// limit the process actually ended up with.
func defaultWatchFDBudget() int {
	budget := softNOFILE() / watchFDBudgetShare
	if budget < minWatchFDBudget {
		budget = minWatchFDBudget
	}
	if budget > maxWatchFDBudget {
		budget = maxWatchFDBudget
	}
	return budget
}
