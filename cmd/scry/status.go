package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/index"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show index status for the current repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo(cmd)
			if err != nil {
				return err
			}
			home, err := scryHome()
			if err != nil {
				return err
			}
			layout := index.Layout(home, repo)
			manifest, err := index.LoadManifest(layout)
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("repo %s is not indexed yet — run `scry init` first", repo)
			} else if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			return printJSON(manifest, pretty)
		},
	}
}
