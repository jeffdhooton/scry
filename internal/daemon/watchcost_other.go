//go:build !(darwin || freebsd || netbsd || openbsd || dragonfly)

package daemon

// watchesEveryEntry reports whether adding a directory to the watcher also
// costs a descriptor per entry inside it. False everywhere inotify or
// ReadDirectoryChangesW is used.
const watchesEveryEntry = false

// watchDirFDCost estimates the descriptors fsnotify will hold after adding
// path. inotify spends one watch descriptor per directory and covers the
// directory's contents with it, so the cost does not scale with entry count.
func watchDirFDCost(path string) int {
	return 1
}
