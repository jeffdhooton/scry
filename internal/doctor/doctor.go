// Package doctor runs environment and integration checks and reports them
// as a structured Report. The goal is "what's wrong with my scry setup?"
// answered in one command: `scry doctor`.
//
// Every check is read-only. doctor never writes to the filesystem, never
// spawns a daemon, never runs `scry init` — it only inspects state. If a
// check wants to verify something that would require side effects, it
// reports a Warn with instructions instead.
//
// The Report is designed for two audiences:
//
//   - Humans running the command interactively. cmd/scry/doctor.go renders
//     the Report as a categorized checklist with ✓ / ⚠ / ✗ markers and
//     per-check remediation hints.
//   - Other tooling (CI scripts, support bots). The --json flag dumps the
//     Report verbatim so a calling script can act on individual check IDs
//     or statuses.
//
// Checks are grouped into four categories so the output stays scannable:
// environment (what machine am I running on), daemon (is scry itself
// alive), indexers (do the per-language tools work), and claude (is the
// Claude Code integration wired up).
package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/install"
	"github.com/jeffdhooton/scry/internal/sources/php"
)

// Status is the result of one check. Pass is green, Warn is yellow
// (something missing but not broken), Fail is red (something required and
// broken), Skip is dim (check wasn't applicable — e.g. language not in repo).
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// Category groups checks for the pretty printer. The order here is the
// order they'll appear in the output.
type Category string

const (
	CategoryEnvironment Category = "Environment"
	CategoryDaemon      Category = "Daemon"
	CategoryIndexers    Category = "Indexers"
	CategoryClaude      Category = "Claude Code integration"
	CategoryRepo        Category = "Current repo"
)

// Check is one diagnostic result. ID is a stable machine-readable key so
// tooling can filter/reference individual checks; Name is the human label.
type Check struct {
	ID       string   `json:"id"`
	Category Category `json:"category"`
	Name     string   `json:"name"`
	Status   Status   `json:"status"`
	Detail   string   `json:"detail"`
	Remedy   string   `json:"remedy,omitempty"`
}

// Report is the full diagnostic output.
type Report struct {
	Checks   []Check `json:"checks"`
	Passed   int     `json:"passed"`
	Warnings int     `json:"warnings"`
	Failed   int     `json:"failed"`
	Skipped  int     `json:"skipped"`
}

// Options controls which checks run.
type Options struct {
	// ScryHome is ~/.scry. Defaults to UserHomeDir/.scry.
	ScryHome string
	// Cwd is the directory to check for "is this repo indexed?". Defaults
	// to the process cwd. Pass a non-repo path to skip repo checks.
	Cwd string
	// Timeout caps any subprocess check (e.g. `claude mcp get`) so a
	// hung subcommand doesn't hang the whole doctor run.
	Timeout time.Duration
}

// Run executes every check and returns a populated Report. Errors are
// returned only for unexpected conditions in doctor itself (e.g. can't
// determine the user's home directory). Individual check failures land in
// the Report as Fail/Warn, not as errors.
func Run(opts Options) (*Report, error) {
	if opts.ScryHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("locate user home: %w", err)
		}
		opts.ScryHome = filepath.Join(home, ".scry")
	}
	if opts.Cwd == "" {
		wd, _ := os.Getwd()
		opts.Cwd = wd
	}
	if opts.Timeout == 0 {
		opts.Timeout = 3 * time.Second
	}

	r := &Report{}
	r.add(checkScryBinary())
	r.add(checkScryHome(opts.ScryHome))
	r.add(checkNOFILE())
	r.add(checkDaemonState(opts.ScryHome))
	r.add(checkPHPInterpreter(opts.Timeout))
	r.add(checkScipTypescript())
	r.add(checkScipGo(opts.ScryHome))
	r.add(checkScipPhpEmbed())
	r.add(checkClaudeCLI(opts.Timeout))
	r.add(checkMCPRegistration(opts.Timeout))
	r.add(checkSkillInstalled())
	r.add(checkGlobalCLAUDEmd())
	r.add(checkCurrentRepo(opts.ScryHome, opts.Cwd))
	return r, nil
}

// add appends a check and updates the summary counters.
func (r *Report) add(c Check) {
	r.Checks = append(r.Checks, c)
	switch c.Status {
	case StatusPass:
		r.Passed++
	case StatusWarn:
		r.Warnings++
	case StatusFail:
		r.Failed++
	case StatusSkip:
		r.Skipped++
	}
}

// ---------------- environment ----------------

func checkScryBinary() Check {
	exe, err := os.Executable()
	if err != nil {
		return Check{
			ID:       "scry.binary",
			Category: CategoryEnvironment,
			Name:     "scry binary",
			Status:   StatusWarn,
			Detail:   "couldn't resolve scry binary path: " + err.Error(),
		}
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return Check{
		ID:       "scry.binary",
		Category: CategoryEnvironment,
		Name:     "scry binary",
		Status:   StatusPass,
		Detail:   exe,
	}
}

func checkScryHome(home string) Check {
	info, err := os.Stat(home)
	if errors.Is(err, os.ErrNotExist) {
		return Check{
			ID:       "env.scry_home",
			Category: CategoryEnvironment,
			Name:     "~/.scry directory",
			Status:   StatusWarn,
			Detail:   home + " does not exist yet",
			Remedy:   "will be created automatically on first `scry init`",
		}
	}
	if err != nil {
		return Check{
			ID:       "env.scry_home",
			Category: CategoryEnvironment,
			Name:     "~/.scry directory",
			Status:   StatusFail,
			Detail:   "stat " + home + ": " + err.Error(),
		}
	}
	if !info.IsDir() {
		return Check{
			ID:       "env.scry_home",
			Category: CategoryEnvironment,
			Name:     "~/.scry directory",
			Status:   StatusFail,
			Detail:   home + " exists but is not a directory",
			Remedy:   "remove " + home + " and re-run `scry init`",
		}
	}
	// Probe writability by creating a temp file.
	tmp, err := os.CreateTemp(home, ".doctor-*.tmp")
	if err != nil {
		return Check{
			ID:       "env.scry_home",
			Category: CategoryEnvironment,
			Name:     "~/.scry directory",
			Status:   StatusFail,
			Detail:   home + " not writable: " + err.Error(),
			Remedy:   "check directory permissions",
		}
	}
	_ = tmp.Close()
	_ = os.Remove(tmp.Name())
	return Check{
		ID:       "env.scry_home",
		Category: CategoryEnvironment,
		Name:     "~/.scry directory",
		Status:   StatusPass,
		Detail:   home,
	}
}

func checkNOFILE() Check {
	var lim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		return Check{
			ID:       "env.nofile",
			Category: CategoryEnvironment,
			Name:     "NOFILE rlimit",
			Status:   StatusWarn,
			Detail:   "getrlimit failed: " + err.Error(),
		}
	}
	detail := fmt.Sprintf("soft=%s hard=%s", rlimitValue(lim.Cur), rlimitValue(lim.Max))
	// macOS ships 256 by default. scry's daemon raises its own soft limit to
	// the hard limit on startup, so 256 isn't strictly fatal — but it's
	// worth flagging at the shell level because `scry init` from the CLI
	// won't benefit and PHP/Composer can trip over it.
	if lim.Cur < 4096 {
		return Check{
			ID:       "env.nofile",
			Category: CategoryEnvironment,
			Name:     "NOFILE rlimit",
			Status:   StatusWarn,
			Detail:   detail + " (low; scry daemon will self-raise on start)",
			Remedy:   "consider `ulimit -n 65536` in your shell rc for CLI comfort",
		}
	}
	return Check{
		ID:       "env.nofile",
		Category: CategoryEnvironment,
		Name:     "NOFILE rlimit",
		Status:   StatusPass,
		Detail:   detail,
	}
}

func rlimitValue(n uint64) string {
	// Linux/darwin both use uint64 max as "unlimited"; render that.
	if n >= 1<<62 {
		return "unlimited"
	}
	return strconv.FormatUint(n, 10)
}

// ---------------- daemon ----------------

func checkDaemonState(scryHome string) Check {
	layout := daemon.LayoutFor(scryHome)
	alive, pid := daemon.AliveDaemon(layout)
	if alive {
		return Check{
			ID:       "daemon.state",
			Category: CategoryDaemon,
			Name:     "scry daemon",
			Status:   StatusPass,
			Detail:   fmt.Sprintf("running (pid %d, socket %s)", pid, layout.SocketPath),
		}
	}
	// Not running is fine — auto-spawns on first query. Only flag if there's
	// a stale socket or PID file that would confuse auto-spawn.
	_, sockErr := os.Stat(layout.SocketPath)
	_, pidErr := os.Stat(layout.PIDPath)
	if sockErr == nil || pidErr == nil {
		return Check{
			ID:       "daemon.state",
			Category: CategoryDaemon,
			Name:     "scry daemon",
			Status:   StatusWarn,
			Detail:   "not running, but stale files exist",
			Remedy:   "run `scry stop` to clean up, or just `scry init <repo>` (auto-cleans)",
		}
	}
	return Check{
		ID:       "daemon.state",
		Category: CategoryDaemon,
		Name:     "scry daemon",
		Status:   StatusPass,
		Detail:   "not running (will auto-spawn on first query)",
	}
}

// ---------------- indexers ----------------

func checkPHPInterpreter(timeout time.Duration) Check {
	bin, err := exec.LookPath("php")
	if err != nil {
		return Check{
			ID:       "indexers.php",
			Category: CategoryIndexers,
			Name:     "php interpreter",
			Status:   StatusWarn,
			Detail:   "not on PATH (required only for indexing PHP repos)",
			Remedy:   "install PHP 8.3+ — Laravel Herd, brew install php, or https://www.php.net",
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "-n", "--version").CombinedOutput()
	if err != nil {
		return Check{
			ID:       "indexers.php",
			Category: CategoryIndexers,
			Name:     "php interpreter",
			Status:   StatusWarn,
			Detail:   bin + " failed: " + err.Error(),
		}
	}
	version := firstLine(string(out))
	// Parse "PHP 8.4.12 (cli)..." for the version number.
	if major, minor, ok := parsePHPVersion(version); ok {
		if major < 8 || (major == 8 && minor < 3) {
			return Check{
				ID:       "indexers.php",
				Category: CategoryIndexers,
				Name:     "php interpreter",
				Status:   StatusFail,
				Detail:   fmt.Sprintf("%s (%d.%d too old — scip-php needs 8.3+)", bin, major, minor),
				Remedy:   "upgrade to PHP 8.3 or newer",
			}
		}
	}
	return Check{
		ID:       "indexers.php",
		Category: CategoryIndexers,
		Name:     "php interpreter",
		Status:   StatusPass,
		Detail:   fmt.Sprintf("%s — %s", bin, version),
	}
}

func parsePHPVersion(versionLine string) (major, minor int, ok bool) {
	// Example: "PHP 8.4.12 (cli) (built: Sep 2 2025 09:22:01) ..."
	fields := strings.Fields(versionLine)
	if len(fields) < 2 || fields[0] != "PHP" {
		return 0, 0, false
	}
	parts := strings.Split(fields[1], ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return maj, min, true
}

func checkScipTypescript() Check {
	bin, err := exec.LookPath("scip-typescript")
	if err != nil {
		return Check{
			ID:       "indexers.scip_typescript",
			Category: CategoryIndexers,
			Name:     "scip-typescript",
			Status:   StatusWarn,
			Detail:   "not on PATH (required only for indexing TS/JS repos)",
			Remedy:   "npm i -g @sourcegraph/scip-typescript",
		}
	}
	return Check{
		ID:       "indexers.scip_typescript",
		Category: CategoryIndexers,
		Name:     "scip-typescript",
		Status:   StatusPass,
		Detail:   bin,
	}
}

func checkScipGo(scryHome string) Check {
	if bin, err := exec.LookPath("scip-go"); err == nil {
		return Check{
			ID:       "indexers.scip_go",
			Category: CategoryIndexers,
			Name:     "scip-go",
			Status:   StatusPass,
			Detail:   bin + " (on PATH)",
		}
	}
	cached := filepath.Join(scryHome, "bin", "scip-go")
	if info, err := os.Stat(cached); err == nil && !info.IsDir() {
		return Check{
			ID:       "indexers.scip_go",
			Category: CategoryIndexers,
			Name:     "scip-go",
			Status:   StatusPass,
			Detail:   cached + " (auto-downloaded)",
		}
	}
	// Not cached yet — report that auto-download is available for the
	// current platform. If the platform isn't supported, flag it.
	if _, ok := install.Indexers()["scip-go"]; ok {
		return Check{
			ID:       "indexers.scip_go",
			Category: CategoryIndexers,
			Name:     "scip-go",
			Status:   StatusPass,
			Detail:   "will auto-download on first Go repo (pinned version, SHA256-verified)",
		}
	}
	return Check{
		ID:       "indexers.scip_go",
		Category: CategoryIndexers,
		Name:     "scip-go",
		Status:   StatusWarn,
		Detail:   "no auto-download recipe and not on PATH",
		Remedy:   "install scip-go manually: go install github.com/sourcegraph/scip-go/cmd/scip-go@latest",
	}
}

func checkScipPhpEmbed() Check {
	size := php.EmbeddedTarballSize()
	if size == 0 {
		return Check{
			ID:       "indexers.scip_php",
			Category: CategoryIndexers,
			Name:     "scip-php (embedded)",
			Status:   StatusFail,
			Detail:   "embedded tarball is empty — this binary was built without the PHP indexer",
			Remedy:   "rebuild scry from a clean checkout with internal/sources/php/scip-php.tar.gz present",
		}
	}
	return Check{
		ID:       "indexers.scip_php",
		Category: CategoryIndexers,
		Name:     "scip-php (embedded)",
		Status:   StatusPass,
		Detail:   fmt.Sprintf("%.1f MB tarball (version %s)", float64(size)/(1024*1024), php.EmbeddedVersion()),
	}
}

// ---------------- claude integration ----------------

func checkClaudeCLI(timeout time.Duration) Check {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return Check{
			ID:       "claude.cli",
			Category: CategoryClaude,
			Name:     "claude CLI",
			Status:   StatusWarn,
			Detail:   "not on PATH — Claude Code integration requires the claude CLI",
			Remedy:   "install Claude Code: https://claude.com/claude-code",
		}
	}
	return Check{
		ID:       "claude.cli",
		Category: CategoryClaude,
		Name:     "claude CLI",
		Status:   StatusPass,
		Detail:   bin,
	}
}

func checkMCPRegistration(timeout time.Duration) Check {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return Check{
			ID:       "claude.mcp",
			Category: CategoryClaude,
			Name:     "scry MCP server",
			Status:   StatusSkip,
			Detail:   "claude CLI not found — skipping registration check",
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, claudeBin, "mcp", "get", "scry").CombinedOutput()
	if err != nil {
		return Check{
			ID:       "claude.mcp",
			Category: CategoryClaude,
			Name:     "scry MCP server",
			Status:   StatusWarn,
			Detail:   "not registered in Claude Code",
			Remedy:   "run `scry setup` to register",
		}
	}
	text := string(out)
	// `claude mcp get scry` output contains "Status: ✓ Connected" on success.
	// Fall back to any-ok behavior if the output format changes.
	status := "registered"
	if strings.Contains(text, "Connected") {
		status = "registered and connected"
	} else if strings.Contains(text, "Failed") {
		status = "registered but failing to connect"
		return Check{
			ID:       "claude.mcp",
			Category: CategoryClaude,
			Name:     "scry MCP server",
			Status:   StatusFail,
			Detail:   status,
			Remedy:   "run `claude mcp get scry` for diagnostics; check the registered Command path exists",
		}
	}
	// Extract the Command: line for the detail so the user can sanity-check
	// which scry binary is registered.
	command := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Command:") {
			command = strings.TrimSpace(strings.TrimPrefix(line, "Command:"))
			break
		}
	}
	detail := status
	if command != "" {
		detail += " — " + command
	}
	return Check{
		ID:       "claude.mcp",
		Category: CategoryClaude,
		Name:     "scry MCP server",
		Status:   StatusPass,
		Detail:   detail,
	}
}

func checkSkillInstalled() Check {
	home, err := os.UserHomeDir()
	if err != nil {
		return Check{
			ID:       "claude.skill",
			Category: CategoryClaude,
			Name:     "scry skill",
			Status:   StatusWarn,
			Detail:   "couldn't resolve home: " + err.Error(),
		}
	}
	path := filepath.Join(home, ".claude", "skills", "scry", "SKILL.md")
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Check{
			ID:       "claude.skill",
			Category: CategoryClaude,
			Name:     "scry skill",
			Status:   StatusWarn,
			Detail:   "not installed",
			Remedy:   "run `scry setup` to install the routing skill",
		}
	}
	if err != nil {
		return Check{
			ID:       "claude.skill",
			Category: CategoryClaude,
			Name:     "scry skill",
			Status:   StatusFail,
			Detail:   "stat " + path + ": " + err.Error(),
		}
	}
	return Check{
		ID:       "claude.skill",
		Category: CategoryClaude,
		Name:     "scry skill",
		Status:   StatusPass,
		Detail:   fmt.Sprintf("%s (%d bytes)", path, info.Size()),
	}
}

func checkGlobalCLAUDEmd() Check {
	home, err := os.UserHomeDir()
	if err != nil {
		return Check{
			ID:       "claude.global_md",
			Category: CategoryClaude,
			Name:     "global CLAUDE.md routing rule",
			Status:   StatusWarn,
			Detail:   "couldn't resolve home: " + err.Error(),
		}
	}
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Check{
			ID:       "claude.global_md",
			Category: CategoryClaude,
			Name:     "global CLAUDE.md routing rule",
			Status:   StatusWarn,
			Detail:   "not present (optional but recommended)",
			Remedy:   "see scry README — a ~15 line global rule makes Claude prefer scry_refs over Grep",
		}
	}
	if err != nil {
		return Check{
			ID:       "claude.global_md",
			Category: CategoryClaude,
			Name:     "global CLAUDE.md routing rule",
			Status:   StatusFail,
			Detail:   "read " + path + ": " + err.Error(),
		}
	}
	mentionsScry := strings.Contains(string(data), "scry") || strings.Contains(string(data), "mcp__scry")
	status := StatusPass
	detail := path
	if !mentionsScry {
		status = StatusWarn
		detail = path + " (present but doesn't mention scry)"
	}
	return Check{
		ID:       "claude.global_md",
		Category: CategoryClaude,
		Name:     "global CLAUDE.md routing rule",
		Status:   status,
		Detail:   detail,
	}
}

// ---------------- current repo ----------------

func checkCurrentRepo(scryHome, cwd string) Check {
	if cwd == "" {
		return Check{
			ID:       "repo.current",
			Category: CategoryRepo,
			Name:     "index state",
			Status:   StatusSkip,
			Detail:   "no cwd provided",
		}
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return Check{
			ID:       "repo.current",
			Category: CategoryRepo,
			Name:     "index state",
			Status:   StatusSkip,
			Detail:   "can't resolve cwd: " + err.Error(),
		}
	}
	layout := index.Layout(scryHome, abs)
	manifestBytes, err := os.ReadFile(layout.ManifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return Check{
			ID:       "repo.current",
			Category: CategoryRepo,
			Name:     abs,
			Status:   StatusWarn,
			Detail:   "not indexed",
			Remedy:   "run `scry init` inside this repo to enable symbol lookups",
		}
	}
	if err != nil {
		return Check{
			ID:       "repo.current",
			Category: CategoryRepo,
			Name:     abs,
			Status:   StatusFail,
			Detail:   "read manifest: " + err.Error(),
		}
	}
	var m index.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return Check{
			ID:       "repo.current",
			Category: CategoryRepo,
			Name:     abs,
			Status:   StatusFail,
			Detail:   "parse manifest: " + err.Error(),
			Remedy:   "run `scry init` to rebuild",
		}
	}
	age := time.Since(m.IndexedAt).Round(time.Minute)
	status := StatusPass
	statusLabel := m.Status
	if statusLabel == "" {
		statusLabel = "ready"
	}
	if statusLabel == "partial" {
		status = StatusWarn
	}
	if statusLabel == "broken" {
		status = StatusFail
	}
	detail := fmt.Sprintf("%s — %d docs, %d symbols, %d refs, indexed %s ago (%s)",
		statusLabel,
		m.Stats.Documents, m.Stats.Symbols, m.Stats.References,
		age, strings.Join(m.Languages, "+"))
	return Check{
		ID:       "repo.current",
		Category: CategoryRepo,
		Name:     abs,
		Status:   status,
		Detail:   detail,
	}
}

// ---------------- helpers ----------------

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
