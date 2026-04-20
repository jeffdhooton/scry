// Package setup installs the scry Claude Code integration: writes the
// embedded SKILL.md to ~/.claude/skills/scry/SKILL.md and registers scry as
// an MCP server via `claude mcp add`.
//
// Design notes:
//
//   - The SKILL.md is version-controlled at internal/setup/SKILL.md and
//     embedded via go:embed so upgrading scry automatically upgrades the
//     skill. Users who want to customize it can edit the installed copy;
//     `scry setup --force` will overwrite it again.
//
//   - MCP registration is done by shelling out to `claude mcp add`, which is
//     Claude Code's official CLI for managing MCP servers. This is MUCH safer
//     than hand-editing ~/.claude.json (which is a 200KB+ file full of
//     unrelated session state that Claude Code owns and rewrites frequently).
//     If `claude` isn't on PATH we print a one-line instruction the user can
//     run manually instead of touching the JSON file directly.
//
//   - A previous version of this package wrote to ~/.claude/settings.json
//     under an `mcpServers` key. That file is for hooks + skill enablement,
//     not MCP servers — Claude Code reads MCP config from ~/.claude.json.
//     On install we clean up any stale entry we may have left in
//     settings.json from that earlier attempt.
package setup

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed SKILL.md
var embeddedSkill []byte

// Options controls the install behavior.
type Options struct {
	// ScryBinary is the absolute path to the scry executable that should be
	// registered in settings.json mcpServers.scry.command. Defaults to
	// os.Executable() at install time.
	ScryBinary string

	// DryRun prints the planned changes without touching disk.
	DryRun bool

	// Force overwrites an existing SKILL.md even if it differs from the
	// embedded copy. Does not affect settings.json merge behavior (which is
	// always a merge).
	Force bool
}

// Result summarizes what Install did, for human-readable output.
type Result struct {
	SkillPath      string   // absolute path to the installed SKILL.md
	SkillAction    string   // "written" | "unchanged" | "dry-run"
	MCPAction      string   // "registered" | "replaced" | "unchanged" | "dry-run" | "manual"
	MCPCommand     string   // the `claude mcp add ...` command we ran (or will run)
	MCPBinary      string   // absolute path to the scry binary that got registered
	ClaudeCLIFound bool     // true if `claude` is on PATH
	StaleSettings  string   // if non-empty: path to settings.json from which we removed a stale entry
	RemovedMCPs    []string // old MCP servers that were unregistered (tome, flume, lore)
	HookAction     string   // "installed" | "unchanged" | "dry-run" | ""
}

// Install performs the full Claude Code integration: SKILL.md + MCP server.
// Returns a Result describing every action. Errors bubble up immediately —
// a partial install is OK because both steps are idempotent.
func Install(opts Options) (*Result, error) {
	claudeHome, err := claudeHomeDir()
	if err != nil {
		return nil, err
	}
	res := &Result{
		SkillPath: filepath.Join(claudeHome, "skills", "scry", "SKILL.md"),
	}

	if err := installSkill(opts, res); err != nil {
		return res, fmt.Errorf("install skill: %w", err)
	}
	if err := cleanupStaleSettings(claudeHome, opts, res); err != nil {
		// Non-fatal: a cleanup failure shouldn't block the real install.
		fmt.Fprintf(os.Stderr, "scry setup: cleanup stale settings.json: %v\n", err)
	}
	cleanupOldMCPServers(opts, res)
	if err := installMCPServer(opts, res); err != nil {
		return res, fmt.Errorf("install mcp server: %w", err)
	}
	if err := installPreToolUseHook(claudeHome, opts, res); err != nil {
		fmt.Fprintf(os.Stderr, "scry setup: install hook: %v\n", err)
	}
	return res, nil
}

// claudeHomeDir returns ~/.claude, creating it if missing. (Most machines
// that run Claude Code already have this directory.)
func claudeHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create claude home: %w", err)
	}
	return dir, nil
}

// ---------------- SKILL.md ----------------

func installSkill(opts Options, res *Result) error {
	dir := filepath.Dir(res.SkillPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	// Short-circuit: if the file exists and matches the embedded copy, no-op.
	existing, readErr := os.ReadFile(res.SkillPath)
	if readErr == nil && bytesEqual(existing, embeddedSkill) && !opts.Force {
		res.SkillAction = "unchanged"
		return nil
	}

	if opts.DryRun {
		res.SkillAction = "dry-run"
		return nil
	}

	// Atomic replace: write to temp sibling, rename over the target.
	tmp, err := os.CreateTemp(dir, "SKILL.md.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(embeddedSkill); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, res.SkillPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	res.SkillAction = "written"
	return nil
}

// ---------------- MCP server registration ----------------

// installMCPServer registers scry as an MCP server via `claude mcp add`.
// The command shape is:
//
//	claude mcp add --scope user --transport stdio scry -- <scry-bin> mcp
//
// If `scry` is already registered and already points at the target binary,
// it's a no-op. If it's registered but pointing somewhere else (e.g. the
// user rebuilt scry in a new location), we `claude mcp remove scry` first
// and re-add.
//
// If `claude` isn't on PATH we print the command the user would need to run
// manually and set res.MCPAction = "manual".
func installMCPServer(opts Options, res *Result) error {
	bin := opts.ScryBinary
	if bin == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate scry binary: %w", err)
		}
		bin = exe
	}
	if abs, err := filepath.Abs(bin); err == nil {
		bin = abs
	}
	res.MCPBinary = bin

	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		res.ClaudeCLIFound = false
		res.MCPCommand = fmt.Sprintf("claude mcp add --scope user --transport stdio scry -- %q mcp", bin)
		res.MCPAction = "manual"
		return nil
	}
	res.ClaudeCLIFound = true
	res.MCPCommand = fmt.Sprintf("%s mcp add --scope user --transport stdio scry -- %q mcp", claudeBin, bin)

	// Inspect current state via `claude mcp get scry`. Parsing is intentionally
	// loose — we only care about whether the command matches our target binary.
	current, currentErr := runClaudeMCP(claudeBin, "get", "scry")
	hasScry := currentErr == nil && len(current) > 0
	commandMatches := hasScry && strings.Contains(current, bin) && strings.Contains(current, " mcp")

	if commandMatches && !opts.Force {
		res.MCPAction = "unchanged"
		return nil
	}

	if opts.DryRun {
		if hasScry {
			res.MCPAction = "replaced (dry-run)"
		} else {
			res.MCPAction = "registered (dry-run)"
		}
		return nil
	}

	if hasScry {
		// Remove the stale entry first. `claude mcp add` refuses to
		// overwrite an existing entry of the same name.
		if _, err := runClaudeMCP(claudeBin, "remove", "scry"); err != nil {
			return fmt.Errorf("remove existing scry entry: %w", err)
		}
	}

	// `claude mcp add -s user -- scry <bin> mcp`
	cmd := exec.Command(claudeBin, "mcp", "add", "--scope", "user", "--transport", "stdio", "scry", "--", bin, "mcp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude mcp add: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	if hasScry {
		res.MCPAction = "replaced"
	} else {
		res.MCPAction = "registered"
	}
	return nil
}

func runClaudeMCP(claudeBin string, args ...string) (string, error) {
	cmd := exec.Command(claudeBin, append([]string{"mcp"}, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ---------------- PreToolUse hook ----------------

// installPreToolUseHook adds a PreToolUse hook for Grep and Glob to
// ~/.claude/settings.json. The hook runs `scry hook pre-search` which checks
// if the current repo is indexed and injects a context nudge telling Claude
// to use scry_refs/scry_defs instead of grep for symbol lookups.
func installPreToolUseHook(claudeHome string, opts Options, res *Result) error {
	bin := res.MCPBinary
	if bin == "" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		bin = exe
	}

	settingsPath := filepath.Join(claudeHome, "settings.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if raw == nil {
		raw = []byte("{}")
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("parse settings.json: %w", err)
	}

	// Parse existing hooks
	type hookEntry struct {
		Type          string `json:"type"`
		Command       string `json:"command"`
		StatusMessage string `json:"statusMessage"`
	}
	type hookMatcher struct {
		Matcher string      `json:"matcher"`
		Hooks   []hookEntry `json:"hooks"`
	}
	type hooksMap struct {
		PreToolUse  []hookMatcher `json:"PreToolUse,omitempty"`
		PostToolUse []hookMatcher `json:"PostToolUse,omitempty"`
	}

	var hooks hooksMap
	if hooksRaw, ok := root["hooks"]; ok {
		if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
			return fmt.Errorf("parse hooks: %w", err)
		}
	}

	hookCommand := bin + " hook pre-search"

	// Check if our hook already exists
	for _, m := range hooks.PreToolUse {
		if m.Matcher == "Grep|Glob" {
			for _, h := range m.Hooks {
				if strings.Contains(h.Command, "hook pre-search") {
					if h.Command == hookCommand {
						res.HookAction = "unchanged"
						return nil
					}
				}
			}
		}
	}

	if opts.DryRun {
		res.HookAction = "dry-run"
		return nil
	}

	// Remove any existing scry hook entries (stale binary path)
	var filtered []hookMatcher
	for _, m := range hooks.PreToolUse {
		if m.Matcher == "Grep|Glob" {
			var kept []hookEntry
			for _, h := range m.Hooks {
				if !strings.Contains(h.Command, "hook pre-search") {
					kept = append(kept, h)
				}
			}
			if len(kept) > 0 {
				m.Hooks = kept
				filtered = append(filtered, m)
			}
		} else {
			filtered = append(filtered, m)
		}
	}

	// Add the new hook
	filtered = append(filtered, hookMatcher{
		Matcher: "Grep|Glob",
		Hooks: []hookEntry{{
			Type:          "command",
			Command:       hookCommand,
			StatusMessage: "Checking scry index...",
		}},
	})
	hooks.PreToolUse = filtered

	// Re-serialize hooks into settings
	hooksJSON, err := json.Marshal(hooks)
	if err != nil {
		return err
	}
	root["hooks"] = hooksJSON

	// Atomic write
	backup := settingsPath + ".bak." + time.Now().Format("20060102-150405")
	if err := os.WriteFile(backup, raw, 0o600); err != nil {
		return fmt.Errorf("backup settings.json: %w", err)
	}
	out, err := marshalIndentOrdered(root)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(claudeHome, "settings.json.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, settingsPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	res.HookAction = "installed"
	return nil
}

// ---------------- cleanup old MCP servers ----------------

// oldMCPServers lists the standalone tools that have been absorbed into scry.
// `scry setup` removes these registrations so Claude Code doesn't see duplicates.
var oldMCPServers = []string{"tome", "flume", "lore"}

// cleanupOldMCPServers removes MCP registrations for tools that have been
// unified into scry (tome, flume, lore). Best-effort: if `claude` isn't on
// PATH or a removal fails, we just skip it.
func cleanupOldMCPServers(opts Options, res *Result) {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return
	}
	for _, name := range oldMCPServers {
		out, err := runClaudeMCP(claudeBin, "get", name)
		if err != nil || len(out) == 0 {
			continue
		}
		if opts.DryRun {
			res.RemovedMCPs = append(res.RemovedMCPs, name+" (dry-run)")
			continue
		}
		if _, err := runClaudeMCP(claudeBin, "remove", name); err != nil {
			fmt.Fprintf(os.Stderr, "scry setup: remove old MCP %q: %v\n", name, err)
			continue
		}
		res.RemovedMCPs = append(res.RemovedMCPs, name)
	}
}

// ---------------- cleanupStaleSettings ----------------

// cleanupStaleSettings removes any leftover mcpServers.scry entry an earlier
// version of this package may have written to ~/.claude/settings.json. MCP
// servers are supposed to live in ~/.claude.json under mcpServers, not in
// settings.json. We silently strip the stale entry so users who ran the old
// `scry setup` don't end up with a dead reference in a file Claude Code
// doesn't even consult for MCP config.
//
// This is best-effort: parse failures are swallowed (the file may have been
// edited to invalid JSON by the user), and a missing file is not an error.
func cleanupStaleSettings(claudeHome string, opts Options, res *Result) error {
	settingsPath := filepath.Join(claudeHome, "settings.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		// Can't parse — don't touch it.
		return nil
	}
	serversRaw, ok := root["mcpServers"]
	if !ok {
		return nil
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(serversRaw, &servers); err != nil {
		return nil
	}
	if _, ok := servers["scry"]; !ok {
		return nil
	}

	if opts.DryRun {
		res.StaleSettings = settingsPath + " (would remove mcpServers.scry)"
		return nil
	}

	delete(servers, "scry")
	if len(servers) == 0 {
		delete(root, "mcpServers")
	} else {
		newServers, err := json.Marshal(servers)
		if err != nil {
			return err
		}
		root["mcpServers"] = newServers
	}

	// Atomic replace. Back up first.
	backup := settingsPath + ".bak." + time.Now().Format("20060102-150405")
	if err := os.WriteFile(backup, raw, 0o600); err != nil {
		return fmt.Errorf("backup settings.json before cleanup: %w", err)
	}
	out, err := marshalIndentOrdered(root)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(claudeHome, "settings.json.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, settingsPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	res.StaleSettings = settingsPath
	return nil
}

// marshalIndentOrdered re-serializes a map[string]json.RawMessage with
// alphabetically-sorted keys and 2-space indent. Used by cleanupStaleSettings
// to rewrite settings.json after removing the stale mcpServers.scry entry.
// This churns key order on first cleanup but stabilizes on subsequent runs.
func marshalIndentOrdered(root map[string]json.RawMessage) ([]byte, error) {
	keys := make([]string, 0, len(root))
	for k := range root {
		keys = append(keys, k)
	}
	sortStrings(keys)

	var out []byte
	out = append(out, '{', '\n')
	for i, k := range keys {
		keyBytes, _ := json.Marshal(k)
		out = append(out, "  "...)
		out = append(out, keyBytes...)
		out = append(out, ':', ' ')
		var v any
		if err := json.Unmarshal(root[k], &v); err != nil {
			return nil, fmt.Errorf("decode %q: %w", k, err)
		}
		vb, err := json.MarshalIndent(v, "  ", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode %q: %w", k, err)
		}
		out = append(out, vb...)
		if i < len(keys)-1 {
			out = append(out, ',')
		}
		out = append(out, '\n')
	}
	out = append(out, '}', '\n')
	return out, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sortStrings is a small bubble sort so the setup package doesn't pull in
// sort from main. The key slices we sort are always small (<20 elements).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// WriteSkillTo is exported for tests: writes the embedded skill to the given
// absolute path, creating parent dirs. Overwrites unconditionally.
func WriteSkillTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, embeddedSkill, 0o644)
}

// EmbeddedSkill returns the embedded SKILL.md bytes for tests and tooling.
func EmbeddedSkill() []byte { return embeddedSkill }
