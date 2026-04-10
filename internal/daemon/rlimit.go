package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// raiseNOFILE bumps the soft NOFILE limit up to the hard limit. fsnotify
// uses one fd per watched directory, and macOS' default soft cap of 256
// is far too low for a real polyglot repo (advocates has ~1500 dirs). The
// hard limit is typically very large; we just need to opt in.
//
// Errors are non-fatal — we log and continue. Worst case the watcher
// degrades and prints "too many open files" warnings, which is preferable
// to refusing to start.
func raiseNOFILE() {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		fmt.Fprintf(os.Stderr, "scry: getrlimit NOFILE: %v\n", err)
		return
	}
	if lim.Cur >= lim.Max {
		return
	}
	lim.Cur = lim.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		fmt.Fprintf(os.Stderr, "scry: setrlimit NOFILE: %v\n", err)
	}
}
