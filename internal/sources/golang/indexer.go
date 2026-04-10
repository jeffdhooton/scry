// Package golang shells out to scip-go and returns a path to the .scip file
// it produces.
//
// Package name is "golang" rather than "go" to avoid the stdlib import name
// collision and the inevitable "package go" parser errors that follow.
//
// P1: requires scip-go on PATH (`go install
// github.com/sourcegraph/scip-go/cmd/scip-go@latest`). Auto-download lands
// alongside the typescript path.
package golang

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jeffdhooton/scry/internal/install"
)

// ErrIndexerNotFound is returned when scip-go is not on PATH and could not
// be auto-installed (typically because the platform isn't covered by the
// pinned release matrix).
var ErrIndexerNotFound = errors.New("scip-go not found and could not be installed")

// Index runs scip-go against repoRoot and writes the SCIP output to
// outputPath. Returns the absolute outputPath on success.
//
// repoRoot must be an absolute path containing a go.mod file (or a parent
// directory thereof — scip-go will walk up).
//
// Resolution order:
//  1. exec.LookPath("scip-go") — user-installed wins
//  2. install.EnsureIndexer("scip-go", scryHomeBin) — pinned auto-download
func Index(ctx context.Context, scryHomeBin, repoRoot, outputPath string) (string, error) {
	bin, err := resolveBinary(scryHomeBin)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(repoRoot) {
		abs, err := filepath.Abs(repoRoot)
		if err != nil {
			return "", fmt.Errorf("resolve repo root: %w", err)
		}
		repoRoot = abs
	}
	if !filepath.IsAbs(outputPath) {
		abs, err := filepath.Abs(outputPath)
		if err != nil {
			return "", fmt.Errorf("resolve output path: %w", err)
		}
		outputPath = abs
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	cmd := exec.CommandContext(ctx, bin,
		"--project-root", repoRoot,
		"--module-root", repoRoot,
		"--repository-root", repoRoot,
		"--output", outputPath,
		"--quiet",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("scip-go exited non-zero: %w", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("scip-go reported success but no output at %s: %w", outputPath, err)
	}
	return outputPath, nil
}

// resolveBinary picks the first usable scip-go: user PATH wins, then
// scryHomeBin (with auto-download).
func resolveBinary(scryHomeBin string) (string, error) {
	if p, err := exec.LookPath("scip-go"); err == nil {
		return p, nil
	}
	p, err := install.EnsureIndexer("scip-go", scryHomeBin)
	if err != nil {
		return "", fmt.Errorf("scip-go not on PATH and auto-install failed: %w", err)
	}
	return p, nil
}
