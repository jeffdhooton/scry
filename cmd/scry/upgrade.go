package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/upgrade"
)

// upgradeCmd downloads the latest scry release from GitHub and replaces
// the running binary in place. Idempotent: if the current version
// matches the latest tag, reports "up to date" and exits 0.
//
// The running binary is replaced atomically via rename-dance so a
// concurrent `scry refs` in another shell keeps working (Unix inode
// semantics). After upgrade, the user should rerun `scry setup --force`
// to re-register the new binary path with Claude Code.
func upgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Download the latest scry release and replace this binary",
		Long: `Queries GitHub for the latest published release, downloads the
platform-specific archive, verifies SHA256 against the checksum file, and
replaces this binary in place.

If you installed scry via 'go install' (version "dev"), pass --force.
Upgrade pins the replacement to the latest published tag; use --version
to target a specific tag.

After upgrading, run 'scry setup --force' to re-register the new binary
with Claude Code's MCP config.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetVersion, _ := cmd.Flags().GetString("version")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			force, _ := cmd.Flags().GetBool("force")
			check, _ := cmd.Flags().GetBool("check")

			if check {
				latest, err := upgrade.Check(upgrade.Options{})
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "latest published release: %s\n", latest)
				fmt.Fprintf(os.Stderr, "current binary:            %s\n", Version)
				if Version == latest || Version == "v"+latest || "v"+Version == latest {
					fmt.Fprintln(os.Stderr, "you're up to date.")
				} else {
					fmt.Fprintln(os.Stderr, "upgrade available — run `scry upgrade` to apply.")
				}
				return nil
			}

			res, err := upgrade.Run(upgrade.Options{
				CurrentVersion: Version,
				TargetVersion:  targetVersion,
				DryRun:         dryRun,
				Force:          force,
			})
			if err != nil {
				return err
			}
			return printUpgradeResult(res, dryRun)
		},
	}
	cmd.Flags().String("version", "", "target version tag (defaults to latest)")
	cmd.Flags().Bool("dry-run", false, "show what would happen without downloading or replacing")
	cmd.Flags().Bool("force", false, "upgrade even if the current binary reports version \"dev\"")
	cmd.Flags().Bool("check", false, "print the latest version and exit without downloading")
	return cmd
}

func printUpgradeResult(res *upgrade.Result, dryRun bool) error {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	switch res.Action {
	case "already-current":
		fmt.Fprintf(os.Stderr, "%sscry %s is already the latest version.\n", prefix, res.FromVersion)
	case "dry-run":
		fmt.Fprintf(os.Stderr, "%swould upgrade %s → %s\n", prefix, res.FromVersion, res.ToVersion)
		fmt.Fprintf(os.Stderr, "%s  target binary: %s\n", prefix, res.BinaryPath)
	case "upgraded":
		fmt.Fprintf(os.Stderr, "%supgraded %s → %s\n", prefix, res.FromVersion, res.ToVersion)
		fmt.Fprintf(os.Stderr, "%s  binary:      %s\n", prefix, res.BinaryPath)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Next: run `scry setup --force` to re-register the new binary with Claude Code.")
	default:
		fmt.Fprintf(os.Stderr, "%s%s (%s → %s)\n", prefix, res.Action, res.FromVersion, res.ToVersion)
	}
	return nil
}
