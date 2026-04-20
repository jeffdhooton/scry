package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/setup"
)

// setupCmd installs scry's Claude Code integration: writes the SKILL.md to
// ~/.claude/skills/scry/ and merges the scry MCP server entry into
// ~/.claude/settings.json. Idempotent — run it twice and nothing changes.
func setupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install Claude Code integration (skill + MCP server)",
		Long: `Installs scry's Claude Code integration so Claude routes symbol
lookups through scry instead of Grep.

Two things get installed:

  1. A skill at ~/.claude/skills/scry/SKILL.md that teaches Claude when to
     pick scry over Grep for symbol queries.
  2. An mcpServers.scry entry in ~/.claude/settings.json that registers this
     scry binary as a Model Context Protocol server. Claude Code spawns it on
     first use.

The command is idempotent: run it again after a scry upgrade to refresh both.
The existing settings.json is backed up before any change.

After running, restart Claude Code so the new skill and MCP tools get picked
up.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			force, _ := cmd.Flags().GetBool("force")
			scryBin, _ := cmd.Flags().GetString("scry-binary")

			res, err := setup.Install(setup.Options{
				ScryBinary: scryBin,
				DryRun:     dryRun,
				Force:      force,
			})
			if err != nil {
				return err
			}
			return printSetupResult(res, dryRun)
		},
	}
	cmd.Flags().Bool("dry-run", false, "show what would happen without writing anything")
	cmd.Flags().Bool("force", false, "overwrite an existing SKILL.md even if it was customized")
	cmd.Flags().String("scry-binary", "", "absolute path to the scry binary to register (defaults to os.Executable())")
	return cmd
}

func printSetupResult(res *setup.Result, dryRun bool) error {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	fmt.Fprintf(os.Stderr, "%sscry skill:    %s (%s)\n", prefix, res.SkillPath, res.SkillAction)
	fmt.Fprintf(os.Stderr, "%sscry mcp:      %s (%s)\n", prefix, res.MCPBinary, res.MCPAction)
	if res.StaleSettings != "" {
		fmt.Fprintf(os.Stderr, "%s  cleaned up:  stale mcpServers.scry in %s\n", prefix, res.StaleSettings)
	}
	for _, name := range res.RemovedMCPs {
		fmt.Fprintf(os.Stderr, "%s  removed old: %s MCP server (now part of scry)\n", prefix, name)
	}
	if res.HookAction != "" {
		fmt.Fprintf(os.Stderr, "%sscry hook:     PreToolUse on Grep|Glob (%s)\n", prefix, res.HookAction)
	}

	if res.MCPAction == "manual" {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "claude CLI not found on PATH. To finish setup, run:")
		fmt.Fprintln(os.Stderr, "  "+res.MCPCommand)
		return nil
	}
	if dryRun {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Dry run complete. Rerun without --dry-run to apply.")
		return nil
	}
	if res.SkillAction == "unchanged" && res.MCPAction == "unchanged" && res.StaleSettings == "" {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Already installed and up to date. Nothing to do.")
		return nil
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Next: restart Claude Code to pick up the new skill and MCP tools.")
	fmt.Fprintln(os.Stderr, "Then try asking it \"where is <some function> called?\" — it should route through scry.")
	return nil
}
