package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/doctor"
)

// doctorCmd runs `scry doctor` — a read-only environment diagnostic that
// checks every piece of the install (binary, scip indexers, daemon state,
// Claude Code integration, current repo index) and prints a categorized
// pass/warn/fail checklist.
//
// Exit code: 1 if any check failed, 0 otherwise. Warnings don't affect
// exit code — they're advisory (e.g. "scip-typescript not installed" is
// only fatal if you're trying to index a TS repo).
func doctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose scry install and Claude Code integration",
		Long: `Runs a read-only health check across:

  - scry binary location and ~/.scry writability
  - NOFILE rlimit (fsnotify needs headroom on macOS)
  - scry daemon state (running / stale / clean)
  - per-language indexers: php, scip-typescript, scip-go, embedded scip-php
  - Claude Code integration: claude CLI, MCP registration, skill, global CLAUDE.md
  - current repo's index state (if any)

Prints a categorized checklist with remediation hints. Use --json for a
machine-readable report suitable for bug reports or scripts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := scryHome()
			if err != nil {
				return err
			}
			cwd, _ := os.Getwd()

			report, err := doctor.Run(doctor.Options{
				ScryHome: home,
				Cwd:      cwd,
			})
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			fix, _ := cmd.Flags().GetBool("fix")

			var fixes []doctor.FixResult
			if fix {
				fixes = doctor.RunFixes(report, doctor.FixOptions{
					Options: doctor.Options{ScryHome: home, Cwd: cwd},
				})
				// Re-run the diagnostic so the "after" report reflects
				// whatever the fixes changed. Users get a clean view of
				// what's still broken without rerunning the command.
				report, err = doctor.Run(doctor.Options{
					ScryHome: home,
					Cwd:      cwd,
				})
				if err != nil {
					return err
				}
			}

			if jsonOut {
				payload := map[string]any{
					"report": report,
				}
				if fix {
					payload["fixes"] = fixes
				}
				if err := printJSON(payload, true); err != nil {
					return err
				}
				if report.Failed > 0 {
					os.Exit(1)
				}
				return nil
			}
			printDoctorReport(os.Stdout, report)
			if fix {
				printFixResults(os.Stdout, fixes)
			}
			if report.Failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "machine-readable JSON output")
	cmd.Flags().Bool("fix", false, "attempt to auto-remediate failing checks (fast, reversible fixes only)")
	return cmd
}

// printFixResults is the "here's what --fix did" footer, rendered after
// the post-fix Report. Mirrors the check marker style so the two blocks
// read the same way.
func printFixResults(w io.Writer, fixes []doctor.FixResult) {
	if len(fixes) == 0 {
		return
	}
	colors := isTerminal(os.Stdout)
	c := newPalette(colors)
	fmt.Fprintln(w)
	fmt.Fprintln(w, c.bold("Fixes applied"))
	for _, f := range fixes {
		marker, color := markerFor(f.Status)
		label := c.apply(color, marker) + " " + padRight(f.Name, 28)
		fmt.Fprintf(w, "  %s %s\n", label, f.Action)
		if f.Detail != "" {
			fmt.Fprintf(w, "    %s %s\n", c.dim("→"), c.dim(f.Detail))
		}
	}
}

// printDoctorReport renders the report as a grouped, color-coded checklist.
// Colors are emitted only when stdout is a character device (TTY) — piped
// output stays plain for CI-friendliness.
func printDoctorReport(w io.Writer, r *doctor.Report) {
	colors := isTerminal(os.Stdout)
	c := newPalette(colors)

	// Group checks by category, preserving the insertion order from
	// doctor.Run (which already sequences them by Environment → Daemon →
	// Indexers → Claude → Repo).
	var order []doctor.Category
	seen := map[doctor.Category]bool{}
	byCat := map[doctor.Category][]doctor.Check{}
	for _, chk := range r.Checks {
		if !seen[chk.Category] {
			seen[chk.Category] = true
			order = append(order, chk.Category)
		}
		byCat[chk.Category] = append(byCat[chk.Category], chk)
	}

	for i, cat := range order {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s\n", c.bold(string(cat)))
		for _, chk := range byCat[cat] {
			marker, color := markerFor(chk.Status)
			label := c.apply(color, marker) + " " + padRight(chk.Name, 28)
			fmt.Fprintf(w, "  %s %s\n", label, dimIfBlank(c, chk.Detail))
			if chk.Remedy != "" {
				fmt.Fprintf(w, "    %s %s\n", c.dim("→"), c.dim(chk.Remedy))
			}
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s %d passed, %d warnings, %d failed",
		c.bold("Summary:"), r.Passed, r.Warnings, r.Failed)
	if r.Skipped > 0 {
		fmt.Fprintf(w, ", %d skipped", r.Skipped)
	}
	fmt.Fprintln(w)
	if r.Failed > 0 {
		fmt.Fprintln(w, c.apply("red", "scry doctor: some checks failed — see details above"))
	}
}

// isTerminal reports whether f is attached to a character device (i.e. a
// TTY). We avoid golang.org/x/term by inspecting the file mode directly.
// Mirrors how cobra's internals detect interactivity.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// markerFor maps a status to a single-char marker + an ansi color name.
func markerFor(s doctor.Status) (string, string) {
	switch s {
	case doctor.StatusPass:
		return "✓", "green"
	case doctor.StatusWarn:
		return "⚠", "yellow"
	case doctor.StatusFail:
		return "✗", "red"
	case doctor.StatusSkip:
		return "—", "dim"
	}
	return "?", ""
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func dimIfBlank(c *palette, s string) string {
	if s == "" {
		return c.dim("(no detail)")
	}
	return s
}

// palette is a tiny ansi color helper. Keeping it self-contained avoids
// pulling in a color library for a single subcommand.
type palette struct {
	enabled bool
}

func newPalette(enabled bool) *palette { return &palette{enabled: enabled} }

func (p *palette) apply(name, s string) string {
	if !p.enabled {
		return s
	}
	switch name {
	case "green":
		return "\x1b[32m" + s + "\x1b[0m"
	case "yellow":
		return "\x1b[33m" + s + "\x1b[0m"
	case "red":
		return "\x1b[31m" + s + "\x1b[0m"
	case "dim":
		return "\x1b[2m" + s + "\x1b[0m"
	case "bold":
		return "\x1b[1m" + s + "\x1b[0m"
	}
	return s
}

func (p *palette) dim(s string) string  { return p.apply("dim", s) }
func (p *palette) bold(s string) string { return p.apply("bold", s) }
