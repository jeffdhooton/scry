// Package upgrade implements `scry upgrade` — a one-command way to pull
// the latest GitHub release tarball, verify its SHA256, and replace the
// running scry binary in place.
//
// Design:
//
//   - Queries the GitHub Releases API (no auth required for public repos)
//     for the latest tag, then constructs the platform-specific archive
//     URL using the same template as .goreleaser.yaml.
//   - Downloads the tarball and the accompanying checksums file, verifies
//     SHA256, extracts `scry` from the archive.
//   - Atomic binary replacement: rename current -> .old, move new into
//     place, then remove .old. If the rename of the new binary fails the
//     .old rename is rolled back so the user isn't left with a stub.
//   - On Unix, replacing a running binary is safe because the kernel
//     keeps the executable's inode alive until the process exits. The
//     running `scry upgrade` process finishes after replacing itself.
//
// What we deliberately don't do:
//
//   - Semver comparison. We treat version strings as opaque: if the
//     current tag == the latest tag, we say "up to date". Otherwise we
//     upgrade. No "you'd be downgrading" detection.
//   - Signing / PGP verification. Checksum verification is enough for
//     v1 — sign artifacts when there's demand.
//   - Cross-platform binary rewriting on Windows (scry is Unix-only).
package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Options controls an upgrade run.
type Options struct {
	// CurrentVersion is the version string the running binary reports
	// (e.g. from main.Version). Compared to the latest tag for the
	// "already up to date" short-circuit.
	CurrentVersion string
	// Repo is the GitHub owner/name. Defaults to jeffdhooton/scry.
	Repo string
	// TargetVersion pins a specific tag. Empty means "latest".
	TargetVersion string
	// BinaryPath is the absolute path of the binary to replace. Defaults
	// to os.Executable().
	BinaryPath string
	// Force proceeds even when CurrentVersion == "dev" (a from-source
	// build without ldflags injection, so version comparison is
	// meaningless).
	Force bool
	// DryRun prints what would happen without downloading or writing.
	DryRun bool
	// Timeout caps the network round-trip.
	Timeout time.Duration
}

// Result summarizes an upgrade run.
type Result struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
	BinaryPath  string `json:"binary_path"`
	Action      string `json:"action"` // "upgraded" | "already-current" | "dry-run"
}

// defaultRepo matches what goes in the README install one-liner.
const defaultRepo = "jeffdhooton/scry"

// Run performs an upgrade end-to-end. Returns nil + an error on any
// unrecoverable failure; on success the Result describes what happened.
func Run(opts Options) (*Result, error) {
	if opts.Repo == "" {
		opts.Repo = defaultRepo
	}
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.BinaryPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("locate running binary: %w", err)
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		opts.BinaryPath = exe
	}

	target := opts.TargetVersion
	if target == "" {
		latest, err := fetchLatestTag(opts.Repo, opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf("look up latest release: %w", err)
		}
		target = latest
	}

	// Current-vs-target comparison. `dev` builds (go install from source
	// without ldflags) force the user to pass --force because we can't
	// reason about them.
	if opts.CurrentVersion != "" && opts.CurrentVersion != "dev" && !opts.Force {
		if normalizeVersion(opts.CurrentVersion) == normalizeVersion(target) {
			return &Result{
				FromVersion: opts.CurrentVersion,
				ToVersion:   target,
				BinaryPath:  opts.BinaryPath,
				Action:      "already-current",
			}, nil
		}
	}
	if opts.CurrentVersion == "dev" && !opts.Force {
		return nil, fmt.Errorf("current binary reports version %q — a dev build without ldflags injection. Re-run with --force to upgrade anyway, or build from source.", opts.CurrentVersion)
	}

	if opts.DryRun {
		return &Result{
			FromVersion: opts.CurrentVersion,
			ToVersion:   target,
			BinaryPath:  opts.BinaryPath,
			Action:      "dry-run",
		}, nil
	}

	archive, err := downloadAndExtract(opts.Repo, target, opts.Timeout)
	if err != nil {
		return nil, err
	}
	defer os.Remove(archive)

	if err := replaceBinary(opts.BinaryPath, archive); err != nil {
		return nil, fmt.Errorf("replace binary: %w", err)
	}

	return &Result{
		FromVersion: opts.CurrentVersion,
		ToVersion:   target,
		BinaryPath:  opts.BinaryPath,
		Action:      "upgraded",
	}, nil
}

// normalizeVersion strips a leading 'v' so "v0.1.0" and "0.1.0" compare
// equal. We don't do any deeper semver parsing.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// fetchLatestTag queries the GitHub Releases API for the latest published
// (non-draft, non-prerelease) release tag.
func fetchLatestTag(repo string, timeout time.Duration) (string, error) {
	url := "https://api.github.com/repos/" + repo + "/releases/latest"
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "scry-upgrade")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no published releases found for %s yet. Once the first `v*` tag is pushed and the release workflow finishes, this command will work.", repo)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024))
		return "", fmt.Errorf("github api %s: %s\n%s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		TagName string `json:"tag_name"`
		Draft   bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode release json: %w", err)
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("no tag_name in github response (repo may have no published releases yet)")
	}
	return payload.TagName, nil
}

// downloadAndExtract fetches the platform-specific archive + its checksum,
// verifies SHA256, extracts the `scry` binary to a temp file, and returns
// the temp path. The caller is responsible for removing it.
func downloadAndExtract(repo, tag string, timeout time.Duration) (string, error) {
	versionNoV := strings.TrimPrefix(tag, "v")
	osName := runtime.GOOS
	archName := runtime.GOARCH
	archive := fmt.Sprintf("scry_%s_%s_%s.tar.gz", versionNoV, osName, archName)
	checksumFile := fmt.Sprintf("scry_%s_checksums.txt", versionNoV)

	baseURL := "https://github.com/" + repo + "/releases/download/" + tag + "/"
	archiveURL := baseURL + archive
	checksumURL := baseURL + checksumFile

	client := &http.Client{Timeout: timeout}

	// 1. Download the archive into memory. Tarballs are small (a few MB).
	archiveBytes, err := httpGet(client, archiveURL)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", archiveURL, err)
	}

	// 2. Download the checksum file and find our archive's line.
	checksumBytes, err := httpGet(client, checksumURL)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", checksumURL, err)
	}
	expected, err := parseChecksum(string(checksumBytes), archive)
	if err != nil {
		return "", err
	}

	// 3. Verify SHA256.
	sum := sha256.Sum256(archiveBytes)
	got := hex.EncodeToString(sum[:])
	if got != expected {
		return "", fmt.Errorf("checksum mismatch for %s: expected %s, got %s", archive, expected, got)
	}

	// 4. Extract `scry` into a temp file sibling to the target location.
	//    We don't know the target yet here, so use a system temp path.
	tmp, err := os.CreateTemp("", "scry-upgrade-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if err := extractScryBinary(archiveBytes, tmp); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func httpGet(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "scry-upgrade")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	// Cap read size at 200 MB so a pathological response can't exhaust
	// memory. scry binaries are <30 MB today.
	return io.ReadAll(io.LimitReader(resp.Body, 200*1024*1024))
}

// parseChecksum finds the line in a GoReleaser checksums.txt matching
// the given archive filename and returns its SHA256 hex.
func parseChecksum(contents, archiveName string) (string, error) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == archiveName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("archive %s not listed in checksum file", archiveName)
}

// extractScryBinary reads a gzip+tar byte stream, finds the `scry` file
// entry, and writes it to dst.
func extractScryBinary(archiveBytes []byte, dst io.Writer) error {
	gz, err := gzip.NewReader(bytesReader(archiveBytes))
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("scry binary not found in archive")
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		// GoReleaser lays the binary at the archive root.
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == "scry" {
			if _, err := io.Copy(dst, tr); err != nil {
				return fmt.Errorf("copy binary: %w", err)
			}
			return nil
		}
	}
}

// bytesReader is a tiny io.Reader over []byte that we prefer to
// importing bytes just for bytes.NewReader — keeps the import set lean.
type bytesReaderT struct {
	b []byte
	i int
}

func bytesReader(b []byte) *bytesReaderT { return &bytesReaderT{b: b} }

func (r *bytesReaderT) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

// replaceBinary atomically replaces the running binary at dst with the
// binary at src. The sequence:
//
//  1. Rename dst -> dst.old.<pid>.<ns>
//  2. Rename src -> dst
//  3. Remove dst.old.<pid>.<ns>
//
// If step 2 fails we roll back step 1. Concurrent readers of dst on Unix
// stay happy because the kernel keeps the executable's inode alive
// until the process exits.
func replaceBinary(dst, src string) error {
	dir := filepath.Dir(dst)
	// Must be on the same filesystem for rename to work. Write the
	// new binary next to the target first.
	newSibling := dst + ".new." + timestamp()
	if err := moveFile(src, newSibling); err != nil {
		return fmt.Errorf("stage new binary: %w", err)
	}
	if err := os.Chmod(newSibling, 0o755); err != nil {
		_ = os.Remove(newSibling)
		return fmt.Errorf("chmod new binary: %w", err)
	}

	oldSibling := dst + ".old." + timestamp()
	// Rename current to .old so we can roll back.
	if err := os.Rename(dst, oldSibling); err != nil {
		_ = os.Remove(newSibling)
		return fmt.Errorf("archive current binary: %w", err)
	}

	// Promote the new binary.
	if err := os.Rename(newSibling, dst); err != nil {
		// Roll back.
		_ = os.Rename(oldSibling, dst)
		_ = os.Remove(newSibling)
		return fmt.Errorf("promote new binary: %w", err)
	}

	// Background remove the old one. Best effort; leftover .old files
	// are harmless.
	_ = os.Remove(oldSibling)
	_ = dir // silence unused
	return nil
}

// moveFile moves src to dst, trying os.Rename first and falling back to
// copy+remove if the source and destination are on different filesystems
// (typical for /tmp on Linux vs /usr/local/bin).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Cross-device fallback.
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
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}

func timestamp() string {
	return fmt.Sprintf("%d.%d", os.Getpid(), time.Now().UnixNano())
}

// Check is a lightweight "what would upgrade do?" helper that doesn't
// touch disk. Useful for `scry upgrade --check` without --dry-run's
// full download simulation.
func Check(opts Options) (latest string, err error) {
	if opts.Repo == "" {
		opts.Repo = defaultRepo
	}
	if opts.Timeout == 0 {
		opts.Timeout = 15 * time.Second
	}
	return fetchLatestTag(opts.Repo, opts.Timeout)
}
