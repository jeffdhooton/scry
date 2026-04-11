package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jeffdhooton/scry/internal/setup"
)

// FixResult summarizes one remediation attempt. Always paired with a
// check ID so the caller can correlate it back to the original Report.
type FixResult struct {
	CheckID string `json:"check_id"`
	Name    string `json:"name"`
	Status  Status `json:"status"` // pass = fix succeeded; warn = tried but didn't fully fix; skip = no fixer
	Action  string `json:"action"` // short human-readable action
	Detail  string `json:"detail"` // longer explanation or error message
}

// FixOptions mirrors Options but adds toggles specific to remediation.
type FixOptions struct {
	Options
	// AllowNetwork permits fixes that go out to npm, GitHub, etc.
	// Defaults false so a `scry doctor --fix` in an air-gapped CI
	// doesn't silently hang on network calls.
	AllowNetwork bool
}

// RunFixes iterates the checks in the Report and applies a remediation
// for each one that has a fixer registered. Fixes are fast (<1s each)
// and side-effectful but reversible where possible. Any fix that would
// take "real time" (running `scry init`, indexing a repo, installing PHP)
// is skipped with a message rather than executed — the first rule of
// scry doctor --fix is no surprising 50-second waits.
func RunFixes(r *Report, opts FixOptions) []FixResult {
	if opts.ScryHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			opts.ScryHome = filepath.Join(home, ".scry")
		}
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}

	var results []FixResult
	for _, chk := range r.Checks {
		// Only attempt fixes for checks that are actually broken.
		if chk.Status != StatusFail && chk.Status != StatusWarn {
			continue
		}
		fix, ok := fixers[chk.ID]
		if !ok {
			continue
		}
		results = append(results, fix(chk, opts))
	}
	return results
}

// fixers maps check IDs to remediation functions. New checks that want
// to participate in --fix add an entry here. Omitting an entry means
// "no auto-fix available" — the check stays broken and the user reads
// the Remedy line.
var fixers = map[string]func(Check, FixOptions) FixResult{
	"env.scry_home":    fixScryHome,
	"daemon.state":     fixStaleDaemon,
	"claude.mcp":       fixRunSetup,
	"claude.skill":     fixRunSetup,
	"claude.global_md": fixGlobalCLAUDEmd,
}

// ---------------- individual fixers ----------------

func fixScryHome(chk Check, opts FixOptions) FixResult {
	if err := os.MkdirAll(opts.ScryHome, 0o755); err != nil {
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusWarn,
			Action:  "mkdir",
			Detail:  err.Error(),
		}
	}
	return FixResult{
		CheckID: chk.ID,
		Name:    chk.Name,
		Status:  StatusPass,
		Action:  "mkdir",
		Detail:  "created " + opts.ScryHome,
	}
}

// fixStaleDaemon nukes stray ~/.scry/scryd.sock / scryd.pid files left
// over from a crashed daemon. It does NOT kill a live daemon — the
// check only fires when the daemon isn't running but stale files exist.
func fixStaleDaemon(chk Check, opts FixOptions) FixResult {
	removed := 0
	for _, rel := range []string{"scryd.sock", "scryd.pid"} {
		path := filepath.Join(opts.ScryHome, rel)
		if err := os.Remove(path); err == nil {
			removed++
		} else if !errors.Is(err, os.ErrNotExist) {
			return FixResult{
				CheckID: chk.ID,
				Name:    chk.Name,
				Status:  StatusWarn,
				Action:  "cleanup stale daemon files",
				Detail:  "remove " + path + ": " + err.Error(),
			}
		}
	}
	return FixResult{
		CheckID: chk.ID,
		Name:    chk.Name,
		Status:  StatusPass,
		Action:  "cleanup stale daemon files",
		Detail:  fmt.Sprintf("removed %d stale file(s)", removed),
	}
}

// fixRunSetup delegates to setup.Install with Force=true so the embedded
// SKILL.md and MCP registration are both refreshed. This covers both the
// "skill missing" and "MCP not registered" checks — either symptom is
// usually remedied by running setup.
func fixRunSetup(chk Check, opts FixOptions) FixResult {
	res, err := setup.Install(setup.Options{Force: true})
	if err != nil {
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusWarn,
			Action:  "scry setup",
			Detail:  err.Error(),
		}
	}
	detail := fmt.Sprintf("skill=%s, mcp=%s", res.SkillAction, res.MCPAction)
	if res.MCPAction == "manual" {
		// claude CLI not on PATH — setup printed the manual command but
		// we can't actually register the server ourselves.
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusWarn,
			Action:  "scry setup",
			Detail:  detail + " (claude CLI not found — MCP not registered)",
		}
	}
	return FixResult{
		CheckID: chk.ID,
		Name:    chk.Name,
		Status:  StatusPass,
		Action:  "scry setup",
		Detail:  detail,
	}
}

// fixGlobalCLAUDEmd writes a minimal ~/.claude/CLAUDE.md with the scry
// routing rule, IF the file doesn't already exist. If it does exist but
// doesn't mention scry, we leave it alone — the user may have their own
// global instructions and auto-appending risks stepping on them.
func fixGlobalCLAUDEmd(chk Check, opts FixOptions) FixResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusWarn,
			Action:  "write ~/.claude/CLAUDE.md",
			Detail:  err.Error(),
		}
	}
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if _, err := os.Stat(path); err == nil {
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusSkip,
			Action:  "write ~/.claude/CLAUDE.md",
			Detail:  "file already exists but doesn't mention scry — not touching it (edit manually)",
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusWarn,
			Action:  "write ~/.claude/CLAUDE.md",
			Detail:  err.Error(),
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusWarn,
			Action:  "write ~/.claude/CLAUDE.md",
			Detail:  err.Error(),
		}
	}
	if err := os.WriteFile(path, []byte(globalCLAUDEmd), 0o644); err != nil {
		return FixResult{
			CheckID: chk.ID,
			Name:    chk.Name,
			Status:  StatusWarn,
			Action:  "write ~/.claude/CLAUDE.md",
			Detail:  err.Error(),
		}
	}
	return FixResult{
		CheckID: chk.ID,
		Name:    chk.Name,
		Status:  StatusPass,
		Action:  "write ~/.claude/CLAUDE.md",
		Detail:  "wrote " + path,
	}
}

// globalCLAUDEmd is the minimal content written by --fix when the user
// has no existing ~/.claude/CLAUDE.md. Keeping it ~15 lines so it adds
// minimal context cost to every conversation. Matches the content we
// hand-wrote earlier in this project.
const globalCLAUDEmd = `# Global instructions

## Tool routing

**For symbol lookups in a scry-indexed repo, use scry's MCP tools instead of Grep.**

- ` + "`mcp__scry__scry_refs`" + ` — where is this function/class/method/interface used? Accepts compound forms like ` + "`DB::table`" + `, ` + "`auth->user`" + `, ` + "`client.Connect`" + ` — the MCP layer parses them and filters by container class.
- ` + "`mcp__scry__scry_defs`" + ` — where is it defined?
- ` + "`mcp__scry__scry_callers`" + ` / ` + "`mcp__scry__scry_callees`" + ` — call graph lookups (who calls X / what does X call).
- ` + "`mcp__scry__scry_impls`" + ` — interface and base-class implementors.
- ` + "`mcp__scry__scry_status`" + ` — check whether the current repo is indexed before running any of the above.

Grep is still the right tool for: string matches inside comments/docstrings, error messages, regex patterns, TODO hunting, file path patterns (use Glob), or any repo scry hasn't indexed. Check ` + "`scry_status`" + ` first — if the repo isn't listed, fall back to Grep silently. **Do not auto-run ` + "`scry init`" + `**; it takes 10-60 seconds and feels broken when triggered from inside another query. If the user wants it indexed they'll ask.
`

// ensureCmdContext is a shared helper for fixes that shell out. Returns
// a timeout-bounded context matching opts.Timeout. Kept here so we have
// a single place to tighten timeouts if fixes start hanging.
func ensureCmdContext(opts FixOptions) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), opts.Timeout)
}

// assertCommand is a tiny sanity check: returns an error if the given
// command isn't on PATH. Used by network-dependent fixers that want to
// bail early with a clear message.
//
//nolint:unused
func assertCommand(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not on PATH", name)
	}
	return nil
}
