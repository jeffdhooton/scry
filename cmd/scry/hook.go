package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	gitindex "github.com/jeffdhooton/scry/internal/git/index"
	"github.com/jeffdhooton/scry/internal/graph"
	"github.com/jeffdhooton/scry/internal/index"
)

func hookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook",
		Short:  "Claude Code hook handlers (not for direct use)",
		Hidden: true,
	}
	cmd.AddCommand(hookPreSearchCmd())
	cmd.AddCommand(hookPreGitCmd())
	return cmd
}

// hookInput is the JSON Claude Code sends to PreToolUse hooks on stdin.
type hookInput struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	Cwd       string          `json:"cwd"`
}

// hookOutput is the JSON we return to Claude Code on stdout.
type hookOutput struct {
	HookSpecificOutput hookSpecific `json:"hookSpecificOutput"`
}

type hookSpecific struct {
	HookEventName  string `json:"hookEventName"`
	Decision       string `json:"permissionDecision"`
	Context        string `json:"additionalContext,omitempty"`
}

func hookPreSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pre-search",
		Short: "PreToolUse hook: nudge Claude toward scry for symbol lookups",
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil
			}
			var input hookInput
			if err := json.Unmarshal(raw, &input); err != nil {
				return nil
			}

			pattern := extractSearchPattern(input.ToolName, input.ToolInput)
			if pattern == "" || !looksLikeSymbol(pattern) {
				return writeHookAllow("")
			}

			cwd := input.Cwd
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			if cwd == "" {
				return writeHookAllow("")
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return writeHookAllow("")
			}
			scryHome := filepath.Join(home, ".scry")

			layout := index.Layout(scryHome, cwd)
			if _, err := os.Stat(layout.ManifestPath); err != nil {
				return writeHookAllow("scry: this repo is not indexed yet. Run `scry init --all` to enable fast symbol lookups, git intelligence, and architectural graph analysis. Falling back to Grep for now.")
			}

			var context strings.Builder
			context.WriteString(fmt.Sprintf(
				"scry has a pre-computed semantic index for this repo. Before continuing with Grep, try scry_refs(\"%s\") or scry_defs(\"%s\") — they resolve symbols to exact file:line locations with context in <10ms, including cross-file references that Grep misses (facades, interfaces, dynamic dispatch). If scry returns nothing, then fall back to Grep.",
				pattern, pattern,
			))

			graphLayout := graph.Layout(scryHome, cwd)
			if _, err := os.Stat(graphLayout.ManifestPath); err == nil {
				context.WriteString(" This repo also has a unified graph — scry_graph_report shows architectural overview, scry_graph_query searches nodes, scry_graph_path traces connections between components.")
			}

			return writeHookAllow(context.String())
		},
	}
}

func hookPreGitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pre-git",
		Short: "PreToolUse hook: nudge Claude toward scry for git commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil
			}
			var input hookInput
			if err := json.Unmarshal(raw, &input); err != nil {
				return nil
			}

			command := extractBashCommand(input.ToolInput)
			if command == "" {
				return writeHookAllow("")
			}

			suggestion := matchGitCommand(command)
			if suggestion == "" {
				return writeHookAllow("")
			}

			cwd := input.Cwd
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			if cwd == "" {
				return writeHookAllow("")
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return writeHookAllow("")
			}
			scryHome := filepath.Join(home, ".scry")

			// Check if git history is indexed for this repo
			gitLayout := gitindex.Layout(scryHome, cwd)
			if _, err := os.Stat(gitLayout.ManifestPath); err != nil {
				return writeHookAllow("scry: this repo's git history is not indexed yet. Run `scry init --all` to enable fast git intelligence (blame, history, cochange, hotspots, contributors). Falling back to git for now.")
			}

			return writeHookAllow(suggestion)
		},
	}
}

func extractBashCommand(rawInput json.RawMessage) string {
	var input map[string]any
	if err := json.Unmarshal(rawInput, &input); err != nil {
		return ""
	}
	if c, ok := input["command"].(string); ok {
		return strings.TrimSpace(c)
	}
	return ""
}

// matchGitCommand checks if a bash command is a git operation that scry can
// handle better, and returns a nudge string. Returns "" if no match.
func matchGitCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) < 2 || parts[0] != "git" {
		return ""
	}

	switch parts[1] {
	case "blame":
		return "scry has pre-indexed blame data for this repo. Try scry_blame first — it returns structured JSON with author, date, and commit context in <10ms, and scry_intent explains WHY a line was written. Fall back to git blame if scry doesn't cover the file."
	case "log":
		if containsFlag(parts, "--follow") || containsFlag(parts, "--oneline") || containsFlag(parts, "--stat") {
			return "scry has pre-indexed git history for this repo. Try scry_history first — it returns structured commit data with diff stats in <10ms. Also available: scry_cochange (files that change together), scry_hotspots (most churned files). Fall back to git log if you need raw output."
		}
		return "scry has pre-indexed git history for this repo. Try scry_history first — it returns structured JSON with diff stats in <10ms. Also available: scry_cochange (files that change together), scry_hotspots (most churned files), scry_contributors (main authors). Fall back to git log if you need raw output."
	case "shortlog":
		return "scry has pre-indexed contributor data for this repo. Try scry_contributors first — it returns ranked authors by commit count for any file or the whole repo."
	case "diff":
		if containsFlag(parts, "--stat") || containsFlag(parts, "--numstat") {
			return "scry has pre-indexed churn data for this repo. Try scry_hotspots (most changed files) or scry_cochange (files that change together) first — they return structured results in <10ms."
		}
	}
	return ""
}

func containsFlag(parts []string, flag string) bool {
	for _, p := range parts {
		if p == flag {
			return true
		}
	}
	return false
}

func extractSearchPattern(toolName string, rawInput json.RawMessage) string {
	var input map[string]any
	if err := json.Unmarshal(rawInput, &input); err != nil {
		return ""
	}
	switch toolName {
	case "Grep":
		if p, ok := input["pattern"].(string); ok {
			return p
		}
	case "Glob":
		if p, ok := input["pattern"].(string); ok {
			return p
		}
	}
	return ""
}

// looksLikeSymbol returns true if the pattern looks like an identifier lookup
// rather than a regex, string search, or file glob.
func looksLikeSymbol(pattern string) bool {
	if strings.ContainsAny(pattern, "*?[]{}|+\\^$") {
		return false
	}
	if strings.Contains(pattern, " ") {
		return false
	}
	// Must start with a letter or underscore (identifier-like)
	if len(pattern) == 0 {
		return false
	}
	c := pattern[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
		return false
	}
	// At least 3 chars to avoid triggering on short patterns
	if len(pattern) < 3 {
		return false
	}
	return true
}

func writeHookAllow(additionalContext string) error {
	out := hookOutput{
		HookSpecificOutput: hookSpecific{
			HookEventName: "PreToolUse",
			Decision:      "allow",
			Context:       additionalContext,
		},
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}

