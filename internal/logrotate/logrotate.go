// Package logrotate bounds the size of scry's append-only logs.
//
// Rotation happens when the file is opened rather than when it is written.
// That is a deliberate constraint, not an oversight: the daemon log is handed
// to an exec'd child as its stdout, so the writer has to be a real *os.File
// that the child can inherit. A wrapper that checked size on every Write would
// make os/exec fall back to a pipe and a copying goroutine in a parent process
// that exits immediately, which would lose the log entirely.
//
// Open-time rotation bounds the common cases: mcp-calls.jsonl is opened once
// per call, and the daemon log is opened once per daemon start.
package logrotate

import (
	"fmt"
	"os"
)

// OpenAppend opens path for appending, first rotating it aside if it has
// reached maxBytes. It keeps at most keep rotated generations, named
// path.1 (newest) through path.<keep> (oldest).
//
// A rotation failure is never fatal: logging is best-effort, so the file is
// opened regardless and an oversized log is preferred to a lost one.
func OpenAppend(path string, maxBytes int64, keep int, perm os.FileMode) (*os.File, error) {
	if maxBytes > 0 && keep > 0 {
		if info, err := os.Stat(path); err == nil && info.Size() >= maxBytes {
			rotate(path, keep)
		}
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, perm)
}

// rotate shifts path.N to path.N+1, drops what falls off the end, and moves
// path itself to path.1.
func rotate(path string, keep int) {
	oldest := fmt.Sprintf("%s.%d", path, keep)
	_ = os.Remove(oldest)

	for i := keep - 1; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", path, i)
		to := fmt.Sprintf("%s.%d", path, i+1)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		_ = os.Rename(from, to)
	}
	_ = os.Rename(path, path+".1")
}
