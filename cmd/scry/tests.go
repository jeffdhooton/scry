package main

import (
	"github.com/spf13/cobra"
)

func testsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tests <symbol>",
		Short: "Check test coverage for a symbol",
		Long: `Check whether a function or method is covered by tests. Requires a
coverage file (cover.out, coverage-final.json, clover.xml, or coverage.json)
to be present in the repo — generate it with your test runner, then re-run
scry init to index it.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, "tests", args[0])
		},
	}
}
