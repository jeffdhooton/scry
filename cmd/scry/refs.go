package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/query"
	"github.com/jeffdhooton/scry/internal/store"
)

func refsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refs <symbol>",
		Short: "Find all references to a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, args[0], query.Refs)
		},
	}
}

func defsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "defs <symbol>",
		Short: "Find the definition site(s) of a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd, args[0], query.Defs)
		},
	}
}

type queryFn func(*store.Store, string) (*query.Result, error)

func runQuery(cmd *cobra.Command, name string, fn queryFn) error {
	repo, err := resolveRepo(cmd)
	if err != nil {
		return err
	}
	home, err := scryHome()
	if err != nil {
		return err
	}
	layout := index.Layout(home, repo)

	if _, err := os.Stat(layout.BadgerDir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("repo %s is not indexed yet — run `scry init` first", repo)
	} else if err != nil {
		return err
	}

	st, err := store.Open(layout.BadgerDir)
	if err != nil {
		return err
	}
	defer st.Close()

	start := time.Now()
	res, err := fn(st, name)
	if err != nil {
		return err
	}
	res.ElapsedMs = time.Since(start).Milliseconds()

	pretty, _ := cmd.Flags().GetBool("pretty")
	return printJSON(res, pretty)
}
