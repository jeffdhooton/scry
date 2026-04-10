// Package typescript shells out to scip-typescript and returns a path to the
// .scip file it produces.
//
// P0: requires scip-typescript on PATH (manual `npm i -g
// @sourcegraph/scip-typescript`). Auto-download lands in P1.
package typescript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ErrIndexerNotFound is returned when scip-typescript is not on PATH.
var ErrIndexerNotFound = errors.New("scip-typescript not found on PATH; install with: npm i -g @sourcegraph/scip-typescript")

// Index runs scip-typescript against repoRoot and writes the SCIP output to
// outputPath. Returns the absolute outputPath on success.
//
// repoRoot must be an absolute path to a directory containing a tsconfig.json
// (or package.json that scip-typescript can resolve from).
func Index(ctx context.Context, repoRoot, outputPath string) (string, error) {
	bin, err := exec.LookPath("scip-typescript")
	if err != nil {
		return "", ErrIndexerNotFound
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
		"index",
		"--cwd", repoRoot,
		"--output", outputPath,
		"--no-progress-bar",
		"--infer-tsconfig",
	)
	// scip-typescript writes its build log to stdout/stderr; pipe both up so the
	// caller can decide whether to surface them.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("scip-typescript exited non-zero: %w", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("scip-typescript reported success but no output at %s: %w", outputPath, err)
	}
	return outputPath, nil
}
