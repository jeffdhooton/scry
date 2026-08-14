package sourceexec

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestRunCapturesAndClipsStderrTail(t *testing.T) {
	payload := strings.Repeat("x", MaxStderrBytes+137) + "\nFINAL COMPILER COMPLAINT\n"
	cmd := exec.Command("sh", "-c", `printf %s "$1" >&2; exit 23`, "sh", payload)

	err := Run(cmd, "fake-indexer", nil)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("errors.As(%T) = false; want *ExitError", err)
	}
	want := truncationMarker + payload[len(payload)-MaxStderrBytes:]
	if exitErr.Stderr != want {
		t.Fatalf("Stderr mismatch: got %d bytes, want %d", len(exitErr.Stderr), len(want))
	}
	if !strings.HasPrefix(exitErr.Stderr, truncationMarker) {
		t.Fatalf("Stderr = %q..., want truncation marker", exitErr.Stderr[:80])
	}
	if !strings.Contains(err.Error(), "FINAL COMPILER COMPLAINT") {
		t.Fatalf("Error() = %q, want final stderr line", err)
	}
}

func TestRunKeepsShortStderrVerbatim(t *testing.T) {
	cmd := exec.Command("sh", "-c", `printf 'first\nactual complaint\n' >&2; exit 1`)
	err := Run(cmd, "fake-indexer", nil)

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("errors.As(%T) = false; want *ExitError", err)
	}
	if exitErr.Stderr != "first\nactual complaint\n" {
		t.Fatalf("Stderr = %q", exitErr.Stderr)
	}
	if got := err.Error(); !strings.HasSuffix(got, ": actual complaint") {
		t.Fatalf("Error() = %q, want last stderr line", got)
	}
}
