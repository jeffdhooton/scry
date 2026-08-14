// Package sourceexec provides the shared subprocess error contract used by
// shell-out source indexers.
package sourceexec

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// MaxStderrBytes is the amount of trailing stderr retained from a failed
// indexer. Compiler diagnostics generally put the useful complaint last.
const MaxStderrBytes = 8 * 1024

const truncationMarker = "...[truncated, showing last 8192 bytes]\n"

// ExitError describes a child indexer that started but exited unsuccessfully.
// Stderr contains a bounded tail of the child's stderr.
type ExitError struct {
	Tool   string
	Err    error
	Stderr string
}

func (e *ExitError) Error() string {
	base := fmt.Sprintf("%s exited non-zero: %v", e.Tool, e.Err)
	if line := lastLine(e.Stderr); line != "" {
		return base + ": " + line
	}
	return base
}

// Unwrap preserves the underlying exec error for errors.Is/errors.As callers.
func (e *ExitError) Unwrap() error { return e.Err }

// Run executes cmd, mirrors its output to log, and returns an ExitError with a
// bounded stderr tail when the child exits unsuccessfully.
func Run(cmd *exec.Cmd, tool string, log io.Writer) error {
	if log == nil {
		log = io.Discard
	}
	var tail tailWriter
	cmd.Stdout = log
	cmd.Stderr = io.MultiWriter(log, &tail)
	if err := cmd.Run(); err != nil {
		return &ExitError{Tool: tool, Err: err, Stderr: tail.String()}
	}
	return nil
}

func lastLine(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}

// tailWriter retains only the last MaxStderrBytes while still reporting every
// input byte as written, making it safe to place behind io.MultiWriter.
type tailWriter struct {
	buf       []byte
	truncated bool
}

func (w *tailWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		return 0, nil
	}
	if n >= MaxStderrBytes {
		w.truncated = w.truncated || len(w.buf) > 0 || n > MaxStderrBytes
		w.buf = append(w.buf[:0], p[n-MaxStderrBytes:]...)
		return n, nil
	}
	if overflow := len(w.buf) + n - MaxStderrBytes; overflow > 0 {
		w.truncated = true
		copy(w.buf, w.buf[overflow:])
		w.buf = w.buf[:len(w.buf)-overflow]
	}
	w.buf = append(w.buf, p...)
	return n, nil
}

func (w *tailWriter) String() string {
	if len(w.buf) == 0 {
		return ""
	}
	if w.truncated {
		return truncationMarker + string(w.buf)
	}
	return string(w.buf)
}
