package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/jeffdhooton/scry/internal/rpc"
	"github.com/spf13/cobra"
)

var reviewIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$`)

func reviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "review", Short: "Inspect provisional background change reviews",
		Long: "Inspect local, snapshot-bound change reviews. Findings are provisional; check snapshot freshness before acting. Reviews never edit code or admit findings into memory.",
	}
	cmd.PersistentFlags().String("socket", "", "use this local review daemon socket directly (no auto-spawn)")
	cmd.AddCommand(&cobra.Command{
		Use: "status", Short: "Show review configuration, activity and usage without inference", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runReview(cmd, "status", map[string]string{}) },
	})
	cmd.AddCommand(&cobra.Command{
		Use: "resume", Short: "Resume reviews after fixing provider availability; keeps daily limits", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runReview(cmd, "resume", map[string]string{}) },
	})
	for _, action := range []struct{ name, description string }{
		{"preview", "Capture local change evidence without model inference"},
		{"run", "Queue asynchronous model inference for an explicitly configured repository within configured spending caps"},
		{"list", "List provisional reviews and snapshot freshness without inference"},
	} {
		cmd.AddCommand(&cobra.Command{
			Use: action.name, Short: action.description, Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				repo, err := resolveRepo(cmd)
				if err != nil {
					return err
				}
				return runReview(cmd, action.name, map[string]string{"repo": repo})
			},
		})
	}
	cmd.AddCommand(&cobra.Command{
		Use: "get <id>", Short: "Retrieve provisional findings, captured evidence and snapshot freshness without inference", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !reviewIDPattern.MatchString(args[0]) {
				return fmt.Errorf("review ID must be 1–200 ASCII letters/digits/._:- and start with a letter or digit")
			}
			return runReview(cmd, "get", map[string]string{"id": args[0]})
		},
	})
	return cmd
}

func runReview(cmd *cobra.Command, action string, params any) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	var result json.RawMessage
	socket, _ := cmd.Flags().GetString("socket")
	if socket != "" {
		client, err := rpc.Dial(socket)
		if err != nil {
			return err
		}
		defer client.Close()
		if err := client.Call(ctx, "review."+action, params, &result); err != nil {
			return err
		}
	} else if err := callDaemon(ctx, "review."+action, params, &result); err != nil {
		return err
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	pretty, _ := cmd.Flags().GetBool("pretty")
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(result)
}
