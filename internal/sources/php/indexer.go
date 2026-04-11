// Package php shells out to a vendored copy of davidrjenni/scip-php and returns
// a path to the .scip file it produces.
//
// scip-php is a PHP CLI program. We can't auto-download a "binary" because
// PHP itself isn't bundled. Instead, scry embeds a pruned tarball of
// scip-php's source tree (the pinned commit, all production composer deps
// installed, plus a small patch in src/Composer/Composer.php that re-prepends
// scip-php's bundled nikic/php-parser at the front of the SPL autoloader so it
// wins over whatever version the target Laravel project pins). On first use
// the tarball is extracted into ~/.scry/bin/scip-php-<sha[:12]>/ and we run
// `php scip-php-<sha[:12]>/bin/scip-php` from inside the target repo.
//
// PHP itself must be on PATH. The user is expected to already have PHP for
// any project this indexer is invoked against (it's a PHP project), so we
// don't try to bundle PHP.
package php

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed scip-php.tar.gz
var scipPhpTarball []byte

// ErrPhpNotFound is returned when the `php` interpreter is not on PATH.
var ErrPhpNotFound = errors.New("php interpreter not found on PATH; install PHP 8.3+ to index PHP repos")

// scipPhpVersion is a stable identifier for the embedded scip-php tree. It is
// the SHA256 of the tarball, truncated. Bumping the embedded tarball
// automatically changes the version, which forces a fresh extraction directory
// (the old one is left in place; users can prune ~/.scry/bin/scip-php-* by
// hand if they care).
func scipPhpVersion() string {
	sum := sha256.Sum256(scipPhpTarball)
	return hex.EncodeToString(sum[:])[:12]
}

// Index runs scip-php against repoRoot and writes the SCIP output to
// outputPath. Returns the absolute outputPath on success.
//
// scryHomeBin is typically ~/.scry/bin — the directory under which the
// extracted scip-php tree lives.
//
// repoRoot must be an absolute path to a Composer project (composer.json must
// exist at the root). scip-php derives the package name and version from
// composer.json/composer.lock; we don't pass them explicitly.
func Index(ctx context.Context, scryHomeBin, repoRoot, outputPath string) (string, error) {
	phpBin, err := exec.LookPath("php")
	if err != nil {
		return "", ErrPhpNotFound
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

	if _, err := os.Stat(filepath.Join(repoRoot, "composer.json")); err != nil {
		return "", fmt.Errorf("scip-php requires a composer.json at the repo root: %w", err)
	}

	scipDir, err := ensureExtracted(scryHomeBin)
	if err != nil {
		return "", fmt.Errorf("extract scip-php: %w", err)
	}
	scipBin := filepath.Join(scipDir, "bin", "scip-php")

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	// scip-php hardcodes getcwd() as the project root and always writes
	// ./index.scip there. We chdir into the repo, run, then move the output.
	// The transient file in the user's tree lives for ~12s.
	cmd := exec.CommandContext(ctx, phpBin,
		"-d", "memory_limit=2G",
		scipBin,
	)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("scip-php exited non-zero: %w", err)
	}

	// scip-php writes to ./index.scip — pick it up and move it to outputPath.
	produced := filepath.Join(repoRoot, "index.scip")
	if _, err := os.Stat(produced); err != nil {
		return "", fmt.Errorf("scip-php reported success but no index.scip at %s: %w", produced, err)
	}
	defer os.Remove(produced)
	if err := os.Rename(produced, outputPath); err != nil {
		// Cross-device rename can fail; fall back to copy.
		if err := copyFile(produced, outputPath); err != nil {
			return "", fmt.Errorf("move index.scip to %s: %w", outputPath, err)
		}
	}
	return outputPath, nil
}

// ensureExtracted unpacks the embedded scip-php tarball under scryHomeBin if
// the per-version directory doesn't already exist. Returns the absolute path
// to the extracted root.
func ensureExtracted(scryHomeBin string) (string, error) {
	dir := filepath.Join(scryHomeBin, "scip-php-"+scipPhpVersion())
	marker := filepath.Join(dir, ".extracted")
	if _, err := os.Stat(marker); err == nil {
		return dir, nil
	}

	if err := os.MkdirAll(scryHomeBin, 0o755); err != nil {
		return "", fmt.Errorf("create bin dir: %w", err)
	}
	// Extract into a temp sibling and rename so a partially-extracted tree
	// from a previous interrupted run doesn't get mistaken for ready.
	tmp, err := os.MkdirTemp(scryHomeBin, "scip-php-extract-")
	if err != nil {
		return "", err
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(tmp)
		}
	}()

	if err := untarGz(scipPhpTarball, tmp); err != nil {
		return "", fmt.Errorf("untar: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".extracted"), []byte(scipPhpVersion()), 0o644); err != nil {
		return "", err
	}

	// If a concurrent run got there first, accept its tree and remove ours.
	if _, err := os.Stat(marker); err == nil {
		return dir, nil
	}
	if err := os.Rename(tmp, dir); err != nil {
		// Final dir may have been created concurrently; if so, accept it.
		if _, statErr := os.Stat(marker); statErr == nil {
			return dir, nil
		}
		return "", fmt.Errorf("rename %s -> %s: %w", tmp, dir, err)
	}
	cleanup = false
	return dir, nil
}

func untarGz(blob []byte, destDir string) error {
	gz, err := gzip.NewReader(newByteReader(blob))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		// Defend against tar slip — reject any entry that escapes destDir.
		clean := filepath.Clean(hdr.Name)
		if filepath.IsAbs(clean) || clean == ".." || (len(clean) >= 3 && clean[:3] == "../") {
			return fmt.Errorf("tar entry escapes destination: %q", hdr.Name)
		}
		target := filepath.Join(destDir, clean)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		default:
			// Skip device files, fifos, etc.
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

type byteReader struct {
	b []byte
	i int
}

func newByteReader(b []byte) *byteReader { return &byteReader{b: b} }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
