// Command scry is the code-intelligence CLI for AI agents.
//
// In P0 there is no daemon: the CLI opens the BadgerDB index for the current
// repo directly, runs the query, prints JSON, and exits. P1 will introduce a
// long-running daemon and a Unix socket, with this same binary auto-spawning
// it. See docs/SPEC.md §13.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Version is set by ldflags during release builds. P0 ships as "dev".
var Version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "scry",
		Short: "Code intelligence daemon for AI agents",
		Long: `scry is a code intelligence index for AI agents. It pre-computes
symbols, references, definitions, and call graphs from your repository and
exposes them as millisecond-latency JSON queries — replacing the
Read+Grep+Glob loop that eats most of an agent's time and tokens.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().String("repo", "", "repo root (defaults to cwd)")
	root.PersistentFlags().Bool("pretty", false, "pretty-print JSON output for human reading")

	root.AddCommand(versionCmd())
	root.AddCommand(initCmd())
	root.AddCommand(refsCmd())
	root.AddCommand(defsCmd())
	root.AddCommand(callersCmd())
	root.AddCommand(calleesCmd())
	root.AddCommand(implsCmd())
	root.AddCommand(testsCmd())
	root.AddCommand(blameCmd())
	root.AddCommand(historyCmd())
	root.AddCommand(cochangeCmd())
	root.AddCommand(hotspotsCmd())
	root.AddCommand(contributorsCmd())
	root.AddCommand(intentCmd())
	root.AddCommand(describeCmd())
	root.AddCommand(relationsCmd())
	root.AddCommand(schemaSearchCmd())
	root.AddCommand(enumsCmd())
	root.AddCommand(proxyCmd())
	root.AddCommand(requestsCmd())
	root.AddCommand(requestCmd())
	root.AddCommand(graphCmd())
	root.AddCommand(statusCmd())
	root.AddCommand(startCmd())
	root.AddCommand(stopCmd())
	root.AddCommand(mcpCmd())
	root.AddCommand(setupCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(upgradeCmd())
	root.AddCommand(hookCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "scry:", err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print scry version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("scry", Version)
			return nil
		},
	}
}

// resolveRepo returns the absolute repo root selected by --repo or cwd.
func resolveRepo(cmd *cobra.Command) (string, error) {
	override, _ := cmd.Flags().GetString("repo")
	if override != "" {
		return filepath.Abs(override)
	}
	return os.Getwd()
}

// scryHome returns ~/.scry, creating it if missing.
func scryHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}
	dir := filepath.Join(home, ".scry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create scry home: %w", err)
	}
	return dir, nil
}
