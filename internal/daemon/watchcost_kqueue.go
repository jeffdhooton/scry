//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package daemon

import "os"

// watchesEveryEntry reports whether adding a directory to the watcher also
// costs a descriptor per entry inside it. True on the kqueue backends.
const watchesEveryEntry = true

// watchDirFDCost estimates the descriptors fsnotify will hold after adding
// path.
//
// The kqueue backend cannot watch a directory as a unit the way inotify does.
// fsnotify compensates in watchDirectoryFiles: after opening the directory it
// opens every entry inside it — files and subdirectories alike — so it can
// report per-file events. The cost of adding one directory is therefore
// 1 + len(entries), not 1.
//
// An unreadable directory is charged 1: the Add will either fail or watch only
// the directory itself.
func watchDirFDCost(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 1
	}
	return 1 + len(entries)
}
