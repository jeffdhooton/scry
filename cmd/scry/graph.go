package main

import (
	"github.com/spf13/cobra"
)

func callersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "callers <symbol>",
		Short: "Find call sites for a function (refs with the containing function exposed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, "callers", args[0])
		},
	}
}

func calleesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "callees <symbol>",
		Short: "List the symbols called from inside this function",
		Long: `Walk the call graph for a function and return every callee. The
call graph is built at index time from the SCIP enclosing_range field, so
this query only returns useful results for languages whose indexer populates
that field. As of P1: scip-typescript yes, scip-go no.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, "callees", args[0])
		},
	}
}

func implsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "impls <interface-or-base>",
		Short: "Find every implementation of an interface or base type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, "impls", args[0])
		},
	}
}
