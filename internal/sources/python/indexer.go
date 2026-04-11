// Package python shells out to @sourcegraph/scip-python and returns a path
// to the .scip file it produces.
//
// Like the typescript indexer, scip-python is distributed as an npm package
// with no GitHub binary release, so there's no auto-download story — users
// install it manually via `npm i -g @sourcegraph/scip-python` and scry just
// shells out to it. `scry doctor` nags them about the missing prereq.
//
// Two non-obvious things this wrapper has to handle beyond a straight
// shellout:
//
//  1. **Python version pinning.** scip-python 0.6.6 bundles a Pyright build
//     that only recognizes Python 3.10-3.13. On a machine with 3.14 as
//     `python3` (common on bleeding-edge Homebrew), Pyright prints
//     "Python version 3.14 from interpreter is unsupported" and silently
//     produces a 0-document SCIP index. We prepend a PATH shim directory
//     that maps `python` and `python3` to a known-good interpreter (searching
//     3.13 → 3.12 → 3.11 → 3.10). Only if nothing compatible is found do we
//     fall back to whatever's on PATH.
//
//  2. **Project version on non-git projects.** scip-python defaults
//     `--project-version` to the current git revision when the repo is
//     a git checkout. On a non-git directory (scratch project, tarball
//     extraction, etc.) it crashes with a cryptic TypeError inside
//     ScipSymbol.normalizeNameOrVersion. We work around by passing
//     `--project-version 0.0.0` explicitly when .git isn't present.
//
// Everything else (project-name, output path, cwd) is standard scip-python
// CLI usage.
package python

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrIndexerNotFound is returned when scip-python is not on PATH.
var ErrIndexerNotFound = errors.New("scip-python not found on PATH; install with: npm i -g @sourcegraph/scip-python")

// compatiblePythonBinaries lists python executable names scip-python's
// bundled Pyright accepts, from newest supported down. The first one
// that resolves via exec.LookPath (or exists at a candidate venv path)
// wins.
var compatiblePythonBinaries = []string{"python3.13", "python3.12", "python3.11", "python3.10"}

// Index runs scip-python against repoRoot and writes the SCIP output to
// outputPath. Returns the absolute outputPath on success.
//
// scryHomeBin is typically ~/.scry/bin — used as the root for the cached
// PATH shim directory that pins the Python version scip-python sees.
func Index(ctx context.Context, scryHomeBin, repoRoot, outputPath string) (string, error) {
	bin, err := exec.LookPath("scip-python")
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

	// Decide which Python interpreter scip-python should see.
	pythonBin := findCompatiblePython(repoRoot)
	shimDir, err := ensurePythonShim(scryHomeBin, pythonBin)
	if err != nil {
		return "", fmt.Errorf("python PATH shim: %w", err)
	}

	projectName := deriveProjectName(repoRoot)

	args := []string{
		"index",
		"--cwd", repoRoot,
		"--output", outputPath,
		"--project-name", projectName,
		"--quiet",
	}
	// On non-git repos scip-python's auto-version detection crashes.
	// Pass a placeholder so we get a real index.
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); os.IsNotExist(err) {
		args = append(args, "--project-version", "0.0.0")
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = buildEnv(shimDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("scip-python exited non-zero: %w", err)
	}
	if info, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("scip-python reported success but no output at %s: %w", outputPath, err)
	} else if info.Size() < 200 {
		// A <200-byte output is the tell-tale sign of scip-python
		// completing "successfully" while emitting zero documents —
		// usually means it picked up an unsupported Python interpreter
		// or an empty include set. Surface as an error so the caller
		// can report it clearly.
		return "", fmt.Errorf("scip-python produced a %d-byte index at %s (likely zero documents; check python interpreter + project layout)", info.Size(), outputPath)
	}
	return outputPath, nil
}

// buildEnv composes the environment for scip-python: the parent process
// env, with the shim dir prepended to PATH and NODE_OPTIONS set so the
// bundled Pyright doesn't OOM on large projects.
func buildEnv(shimDir string) []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	pathSet := false
	nodeOptsSet := false
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "PATH="):
			out = append(out, "PATH="+shimDir+string(os.PathListSeparator)+kv[len("PATH="):])
			pathSet = true
		case strings.HasPrefix(kv, "NODE_OPTIONS="):
			// Preserve existing options and append ours.
			out = append(out, kv+" --max-old-space-size=8192")
			nodeOptsSet = true
		default:
			out = append(out, kv)
		}
	}
	if !pathSet {
		out = append(out, "PATH="+shimDir)
	}
	if !nodeOptsSet {
		out = append(out, "NODE_OPTIONS=--max-old-space-size=8192")
	}
	return out
}

// findCompatiblePython returns an absolute path to a Python interpreter
// that scip-python's bundled Pyright will accept. Search order:
//
//  1. $VIRTUAL_ENV/bin/python (if the user already activated a venv)
//  2. <repoRoot>/.venv/bin/python, <repoRoot>/venv/bin/python,
//     <repoRoot>/env/bin/python (common in-repo venv locations)
//  3. python3.13, python3.12, python3.11, python3.10 on PATH
//  4. python3 on PATH as a last resort (may be newer than Pyright supports)
//
// Never returns "" — the last-resort falls through to plain "python3" even
// if LookPath fails, because exec will give a clearer error than we'd
// compose here.
func findCompatiblePython(repoRoot string) string {
	if v := os.Getenv("VIRTUAL_ENV"); v != "" {
		candidate := filepath.Join(v, "bin", "python")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	for _, sub := range []string{".venv", "venv", "env"} {
		candidate := filepath.Join(repoRoot, sub, "bin", "python")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	for _, name := range compatiblePythonBinaries {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("python3"); err == nil {
		return p
	}
	return "python3"
}

// ensurePythonShim creates a directory containing `python` and `python3`
// symlinks pointing at pythonBin, returning the directory path so callers
// can prepend it to PATH. The shim is cached per-target under scryHomeBin
// so reruns don't churn: the dir name is a sha256 prefix of the resolved
// python binary path.
func ensurePythonShim(scryHomeBin, pythonBin string) (string, error) {
	if err := os.MkdirAll(scryHomeBin, 0o755); err != nil {
		return "", err
	}
	// Resolve symlinks in the target so cache keys are stable across
	// runs even when the user's PATH picked up a different alias.
	resolved := pythonBin
	if abs, err := exec.LookPath(pythonBin); err == nil {
		resolved = abs
	}
	if real, err := filepath.EvalSymlinks(resolved); err == nil {
		resolved = real
	}
	sum := sha256.Sum256([]byte(resolved))
	shimDir := filepath.Join(scryHomeBin, "python-shim-"+hex.EncodeToString(sum[:])[:12])

	// If the shim already exists and points at the right target, reuse it.
	if info, err := os.Lstat(filepath.Join(shimDir, "python3")); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if existing, err := os.Readlink(filepath.Join(shimDir, "python3")); err == nil && existing == resolved {
				return shimDir, nil
			}
		}
	}

	// (Re)create the shim dir fresh. Tiny, cheap, always correct.
	if err := os.RemoveAll(shimDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		return "", err
	}
	for _, name := range []string{"python", "python3"} {
		if err := os.Symlink(resolved, filepath.Join(shimDir, name)); err != nil {
			return "", err
		}
	}
	return shimDir, nil
}

// projectNameRe matches `name = "foo"` / `name = 'foo'` entries. We only
// look inside the first ~100 lines of pyproject.toml since the [project]
// table is conventionally near the top.
var projectNameRe = regexp.MustCompile(`(?m)^\s*name\s*=\s*['"]([^'"]+)['"]`)

// deriveProjectName returns a project name for scip-python's --project-name
// flag. Search order: pyproject.toml [project].name → setup.cfg
// [metadata].name → setup.py regex → the repo directory basename. This
// becomes part of every symbol id scip-python emits, so stability matters
// more than accuracy — same repo should always produce the same name.
func deriveProjectName(repoRoot string) string {
	// 1. pyproject.toml — the [project] table holds name + version in
	// PEP 621 projects. We scan the whole file for any `name = ...` line
	// before the next [section] header; that's good enough for ~99% of
	// cases. A real TOML parser would be safer but adds a dependency.
	if b, err := os.ReadFile(filepath.Join(repoRoot, "pyproject.toml")); err == nil {
		if name := firstProjectNameIn(string(b), "project"); name != "" {
			return name
		}
	}
	// 2. setup.cfg — [metadata] section holds the name.
	if b, err := os.ReadFile(filepath.Join(repoRoot, "setup.cfg")); err == nil {
		if name := firstProjectNameIn(string(b), "metadata"); name != "" {
			return name
		}
	}
	// 3. setup.py — regex for name="..." or name='...'.
	if b, err := os.ReadFile(filepath.Join(repoRoot, "setup.py")); err == nil {
		if m := projectNameRe.FindStringSubmatch(string(b)); len(m) == 2 {
			return m[1]
		}
	}
	// 4. Repo directory basename.
	return filepath.Base(repoRoot)
}

// firstProjectNameIn scans contents for the first `name = "foo"` line
// occurring inside the [section] table. Returns "" if no match.
func firstProjectNameIn(contents, section string) string {
	header := "[" + section + "]"
	idx := strings.Index(contents, header)
	if idx < 0 {
		return ""
	}
	// Look at lines after the header up to the next [section] header.
	rest := contents[idx+len(header):]
	if end := strings.Index(rest, "\n["); end >= 0 {
		rest = rest[:end]
	}
	if m := projectNameRe.FindStringSubmatch(rest); len(m) == 2 {
		return m[1]
	}
	return ""
}
