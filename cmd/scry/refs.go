package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/query"
)

func refsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refs <symbol>",
		Short: "Find all references to a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, "refs", args[0])
		},
	}
}

func defsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "defs <symbol>",
		Short: "Find the definition site(s) of a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, "defs", args[0])
		},
	}
}

func runQuery(cmd *cobra.Command, method, name string) error {
	repo, err := resolveRepo(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var res query.Result
	if err := callDaemon(ctx, method, &daemon.QueryParams{Repo: repo, Name: name}, &res); err != nil {
		return err
	}
	pretty, _ := cmd.Flags().GetBool("pretty")
	return printJSON(res, pretty)
}
