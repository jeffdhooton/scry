// Package install plans and provisions upstream indexer binaries. GitHub
// release binaries are downloaded into ~/.scry/bin; npm-published indexers
// are installed globally with npm.
//
// For GitHub releases, per docs/DECISIONS.md "Auto-download pinned indexers,
// never auto-update":
//   - versions are compiled into the scry binary; users get a stable indexer
//     until they update scry itself
//   - every download is verified against a SHA256 baked into this package
//   - the cache lives at ~/.scry/bin/<name> and is reused across runs
//
// Coverage in P1:
//   - scip-go: Sourcegraph publishes per-platform tarballs on GitHub.
//   - scip-typescript and scip-python: installed from their official npm
//     packages when npm is available.
//   - scip-php: deferred to P2 (PHAR build out of the calibration plan).
package install

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Source describes how an indexer is provisioned.
type Source int

const (
	SourceGitHubRelease Source = iota
	SourceNPM
	SourceManual
)

// Env contains every external fact used to choose a provisioning plan.
// PlanFor deliberately performs no filesystem or PATH probes itself.
type Env struct {
	NPMPath         string
	ToolPath        string
	DaemonStartedAt time.Time
	ToolInstalledAt time.Time
}

// Plan is the side-effect-free provisioning decision for one language tool.
// Tool uses the language keys from index.indexerRemedies: typescript, python,
// go, and php.
type Plan struct {
	Tool       string
	Source     Source
	Package    string
	Command    []string
	Reason     string
	Actionable bool
}

type npmTool struct {
	binary      string
	packageName string
}

var npmTools = map[string]npmTool{
	"typescript": {binary: "scip-typescript", packageName: "@sourcegraph/scip-typescript"},
	"python":     {binary: "scip-python", packageName: "@sourcegraph/scip-python"},
}

// PlanFor decides how to provision tool using only the supplied environment.
// Keeping the probes outside this function makes every branch table-testable
// without npm, an indexer binary, or network access.
func PlanFor(tool string, env Env) Plan {
	if spec, ok := npmTools[tool]; ok {
		plan := Plan{
			Tool:    tool,
			Source:  SourceNPM,
			Package: spec.packageName,
			Command: []string{"npm", "i", "-g", spec.packageName},
		}
		if env.ToolPath != "" {
			plan.Reason = fmt.Sprintf("%s is already installed at %s", spec.binary, env.ToolPath)
			return plan
		}
		if env.NPMPath == "" {
			plan.Reason = fmt.Sprintf("npm is not on PATH; it is required to install %s", spec.binary)
			return plan
		}
		plan.Actionable = true
		plan.Reason = fmt.Sprintf("install %s from npm", spec.binary)
		return plan
	}

	switch tool {
	case "go":
		plan := Plan{Tool: tool, Source: SourceGitHubRelease}
		if env.ToolPath != "" {
			plan.Reason = "scip-go is already installed at " + env.ToolPath
			return plan
		}
		plan.Actionable = true
		plan.Reason = "install the pinned scip-go GitHub release"
		return plan
	case "php":
		if env.ToolPath != "" {
			return Plan{
				Tool:   tool,
				Source: SourceManual,
				Reason: "php is already installed at " + env.ToolPath,
			}
		}
		return Plan{
			Tool:   tool,
			Source: SourceManual,
			Reason: "PHP 8.3+ is a prerequisite and must be installed manually",
		}
	default:
		return Plan{
			Tool:   tool,
			Source: SourceManual,
			Reason: fmt.Sprintf("no provisioning recipe for tool %q", tool),
		}
	}
}

// StaleDaemonPATH reports the specific ordering that leaves a running daemon
// unable to see a newly installed binary even though the current CLI can.
func StaleDaemonPATH(env Env) bool {
	return env.ToolPath != "" &&
		!env.DaemonStartedAt.IsZero() &&
		env.ToolInstalledAt.After(env.DaemonStartedAt)
}

// ProvisionResult describes a completed provisioning attempt.
type ProvisionResult struct {
	Tool    string
	Binary  string
	Path    string
	Version string
}

// Provision executes an actionable npm plan and verifies that the installed
// binary both resolves on PATH and can be invoked. Decision-making remains in
// PlanFor; this function is intentionally the side-effectful boundary.
func Provision(ctx context.Context, plan Plan) (ProvisionResult, error) {
	if !plan.Actionable {
		return ProvisionResult{}, errors.New(plan.Reason)
	}
	if plan.Source != SourceNPM {
		return ProvisionResult{}, fmt.Errorf("provisioning source for %q is not npm", plan.Tool)
	}
	spec, ok := npmTools[plan.Tool]
	if !ok {
		return ProvisionResult{}, fmt.Errorf("no npm provisioning recipe for %q", plan.Tool)
	}
	if len(plan.Command) == 0 {
		return ProvisionResult{}, fmt.Errorf("npm provisioning command for %q is empty", plan.Tool)
	}

	cmd := exec.CommandContext(ctx, plan.Command[0], plan.Command[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return ProvisionResult{}, fmt.Errorf("%s: %w: %s", strings.Join(plan.Command, " "), err, detail)
		}
		return ProvisionResult{}, fmt.Errorf("%s: %w", strings.Join(plan.Command, " "), err)
	}

	bin, err := exec.LookPath(spec.binary)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("npm installed %s, but %s is not on PATH; ensure npm's global bin directory is on PATH", plan.Package, spec.binary)
	}
	verify := exec.CommandContext(ctx, bin, "--version")
	out, err := verify.CombinedOutput()
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("npm installed %s at %s, but invoking %s --version failed: %w: %s", plan.Package, bin, spec.binary, err, strings.TrimSpace(string(out)))
	}
	return ProvisionResult{
		Tool:    plan.Tool,
		Binary:  spec.binary,
		Path:    bin,
		Version: strings.TrimSpace(string(out)),
	}, nil
}

// BinaryName returns the executable associated with a provisioning key.
func BinaryName(tool string) string {
	if spec, ok := npmTools[tool]; ok {
		return spec.binary
	}
	if tool == "go" {
		return "scip-go"
	}
	return tool
}

// Indexer describes one downloadable upstream indexer.
type Indexer struct {
	Name        string                  // binary name as it should appear in ~/.scry/bin/
	Version     string                  // pinned version, e.g. "0.1.26"
	Platforms   map[string]PlatformInfo // key is "<goos>_<goarch>"
	BinaryInTar string                  // path inside the archive of the binary to extract
}

// PlatformInfo is the per-platform download metadata.
type PlatformInfo struct {
	URL    string
	SHA256 string // hex
}

// scipGo is the pinned scip-go release. Bump this when scry releases.
//
// SHA256s lifted from
// https://github.com/sourcegraph/scip-go/releases/download/v0.1.26/scip-go_0.1.26_checksums.txt
var scipGo = Indexer{
	Name:        "scip-go",
	Version:     "0.1.26",
	BinaryInTar: "scip-go",
	Platforms: map[string]PlatformInfo{
		"darwin_amd64": {
			URL:    "https://github.com/sourcegraph/scip-go/releases/download/v0.1.26/scip-go_0.1.26_darwin_amd64.tar.gz",
			SHA256: "768d8048d537f1e2a26735b37fa0481296c6a1010392b2750c88b73716b529cf",
		},
		"darwin_arm64": {
			URL:    "https://github.com/sourcegraph/scip-go/releases/download/v0.1.26/scip-go_0.1.26_darwin_arm64.tar.gz",
			SHA256: "1b87a5e0b2af4e41bc1cc49220e7d3a84a831468ae6944a9574e7d4c1270909c",
		},
		"linux_amd64": {
			URL:    "https://github.com/sourcegraph/scip-go/releases/download/v0.1.26/scip-go_0.1.26_linux_amd64.tar.gz",
			SHA256: "66257b6db74e13c2e756c9abba8e7d34e62eb91d16cdbe087a0b0c170c89c37d",
		},
		"linux_arm64": {
			URL:    "https://github.com/sourcegraph/scip-go/releases/download/v0.1.26/scip-go_0.1.26_linux_arm64.tar.gz",
			SHA256: "bc8e5abb959521912d60181de8922e5158a609a2e9d87e6ed2b7801c11c0efab",
		},
	},
}

// Indexers returns the registry of downloadable indexers.
func Indexers() map[string]Indexer {
	return map[string]Indexer{
		"scip-go": scipGo,
	}
}

// EnsureIndexer guarantees that the binary `name` exists either on PATH or
// inside binDir, downloading the pinned version if needed. Returns the path
// to the runnable binary.
//
// binDir is typically ~/.scry/bin (the daemon supplies this).
func EnsureIndexer(name, binDir string) (string, error) {
	// Already cached locally?
	cached := filepath.Join(binDir, name)
	if info, err := os.Stat(cached); err == nil && !info.IsDir() {
		return cached, nil
	}

	idx, ok := Indexers()[name]
	if !ok {
		return "", fmt.Errorf("no auto-download recipe for indexer %q", name)
	}
	platform := runtime.GOOS + "_" + runtime.GOARCH
	pi, ok := idx.Platforms[platform]
	if !ok {
		return "", fmt.Errorf("no %s release for platform %s", name, platform)
	}

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("create bin dir: %w", err)
	}
	tmp, err := os.CreateTemp(binDir, name+"-download-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := downloadVerified(pi.URL, pi.SHA256, tmp); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := extractBinary(tmp, idx.BinaryInTar, cached); err != nil {
		_ = tmp.Close()
		return "", err
	}
	_ = tmp.Close()
	if err := os.Chmod(cached, 0o755); err != nil {
		return "", err
	}
	return cached, nil
}

// downloadVerified GETs the URL into dst and verifies the SHA256 matches
// expected. Both the bytes and the hash are streamed so memory usage stays
// constant for any binary size.
func downloadVerified(url, expectedSHA256 string, dst io.Writer) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "scry/dev")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %s", url, resp.Status)
	}
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, hasher), resp.Body); err != nil {
		return fmt.Errorf("download body: %w", err)
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	if got != expectedSHA256 {
		return fmt.Errorf("checksum mismatch for %s:\n  expected %s\n  got      %s", url, expectedSHA256, got)
	}
	return nil
}

// extractBinary opens a gzip+tar archive and copies the named file out to
// destPath. Returns an error if the file is not present in the archive.
func extractBinary(archive io.Reader, name, destPath string) error {
	gz, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("binary %q not found in archive", name)
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		// scip-go's tarball lays out the binary at root, but some indexers
		// nest theirs in a directory; match basename as a fallback.
		if hdr.Name == name || filepath.Base(hdr.Name) == name {
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			return out.Close()
		}
	}
}
