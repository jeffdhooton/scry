package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/friction"
	"github.com/jeffdhooton/scry/internal/rpc"
	"github.com/spf13/cobra"
)

func frictionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "friction", Short: "Record and review explicit workflow friction"}
	cmd.PersistentFlags().String("socket", "", "use this journal daemon socket directly (no auto-spawn)")
	record := &cobra.Command{
		Use: "record <event.json|->", Short: "Retain one immutable event; identical event-ID retries are safe", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var reader io.Reader = cmd.InOrStdin()
			if args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer f.Close()
				reader = f
			}
			b, err := io.ReadAll(io.LimitReader(reader, friction.MaxEventBytes+1))
			if err != nil {
				return err
			}
			if len(b) > friction.MaxEventBytes {
				return fmt.Errorf("event input exceeds %d bytes", friction.MaxEventBytes)
			}
			var e friction.Event
			if err := friction.Decode(b, &e); err != nil {
				return err
			}
			if err := e.Validate(); err != nil {
				return err
			}
			// The authored repository is part of event identity; never silently
			// replace it with the client's cwd or a global --repo flag.
			if cmd.Flags().Changed("repo") {
				return fmt.Errorf("record uses repository in the event; omit --repo")
			}
			return runFriction(cmd, "record", e)
		},
	}
	get := &cobra.Command{
		Use: "get <event-id>", Short: "Retrieve an exact event without model extraction", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := friction.ValidateID(args[0]); err != nil {
				return err
			}
			return runFriction(cmd, "get", daemon.FrictionGetParams{EventID: args[0]})
		},
	}
	cmd.AddCommand(record, get)
	for _, action := range []string{"list", "review"} {
		sub := &cobra.Command{
			Use: action, Short: "Query the journal for this repository (review returns proposals only)", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				repo, err := resolveRepo(cmd)
				if err != nil {
					return err
				}
				f := friction.Filter{Repository: repo}
				f.RunID, _ = cmd.Flags().GetString("run-id")
				f.Signature, _ = cmd.Flags().GetString("signature")
				f.Since, _ = cmd.Flags().GetString("since")
				f.Until, _ = cmd.Flags().GetString("until")
				if action == "list" {
					f.After, _ = cmd.Flags().GetString("after")
					f.Limit, _ = cmd.Flags().GetInt("limit")
				}
				if err := f.Validate(); err != nil {
					return err
				}
				return runFriction(cmd, action, f)
			},
		}
		sub.Flags().String("run-id", "", "exact run ID")
		sub.Flags().String("signature", "", "exact friction signature")
		sub.Flags().String("since", "", "inclusive RFC3339 observation time")
		sub.Flags().String("until", "", "exclusive RFC3339 observation time")
		if action == "list" {
			sub.Flags().String("after", "", "event-ID cursor from next_after (keep the same filters)")
			sub.Flags().Int("limit", friction.MaxResults, "page size (1–100)")
		}
		cmd.AddCommand(sub)
	}
	return cmd
}

func runFriction(cmd *cobra.Command, action string, params any) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	var result json.RawMessage
	socket, _ := cmd.Flags().GetString("socket")
	if socket != "" {
		c, err := rpc.Dial(socket)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Call(ctx, "friction."+action, params, &result); err != nil {
			return err
		}
	} else if err := callDaemon(ctx, "friction."+action, params, &result); err != nil {
		return err
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	pretty, _ := cmd.Flags().GetBool("pretty")
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(result)
}
