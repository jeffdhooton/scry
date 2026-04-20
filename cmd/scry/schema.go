package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func describeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe <table>",
		Short: "Describe a database table (columns, indexes, foreign keys)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "schema.describe", &daemon.SchemaDescribeParams{
				Project: project, Table: args[0],
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func relationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "relations <table>",
		Short: "Show foreign key relationships for a table",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "schema.relations", &daemon.SchemaRelationsParams{
				Project: project, Table: args[0],
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func schemaSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema-search <query>",
		Short: "Search for tables and columns by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "schema.search", &daemon.SchemaSearchParams{
				Project: project, Query: args[0],
			}, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
}

func enumsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enums [table[.column]]",
		Short: "List enum column values",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			p := daemon.SchemaEnumsParams{Project: project}
			if len(args) == 1 {
				table, column := splitTableColumn(args[0])
				p.Table = table
				p.Column = column
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var result any
			if err := callDaemon(ctx, "schema.enums", &p, &result); err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(result, pretty)
		},
	}
	return cmd
}

func splitTableColumn(s string) (string, string) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
