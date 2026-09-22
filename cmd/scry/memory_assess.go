package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/spf13/cobra"
)

func memoryAssessCmd() *cobra.Command {
	var file string
	var live bool
	cmd := &cobra.Command{
		Use:   "assess",
		Short: "Evaluate proposed memories with Jev (request preview by default)",
		Long: `With --file, evaluate labeled episode/fact pairs independently of the memory store.
The status, list, show, replay and resume subcommands use the configured memory daemon.

By default, print the redacted requests without making network calls. --live
sends those requests to TypeSafe using TYPESAFE_API_KEY and incurs API charges.
Use synthetic data first. Redaction only covers known secret patterns.

The JSON report compares predictions with the supplied labels; it does not
admit, discard, or change memories. File evaluation contacts no daemon or memory store.
Runs stop on the first error, writing completed results before exiting nonzero.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("assess: --file is required for dataset evaluation")
			}
			f, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("assess: open dataset: %w", err)
			}
			cases, err := assess.LoadCases(f)
			f.Close()
			if err != nil {
				return err
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if !live {
				type item struct {
					ID      string         `json:"id"`
					Request assess.Request `json:"request"`
				}
				preview := struct {
					Mode          string `json:"mode"`
					RubricVersion string `json:"rubric_version"`
					Requests      []item `json:"requests"`
				}{Mode: "preview", RubricVersion: assess.RubricVersion, Requests: make([]item, 0, len(cases))}
				for _, c := range cases {
					req, err := assess.BuildRequest(c.Episode, c.Fact)
					if err != nil {
						return err
					}
					preview.Requests = append(preview.Requests, item{c.ID, req})
				}
				return encoder.Encode(preview)
			}
			client, err := assess.NewClient(os.Getenv("TYPESAFE_API_KEY"))
			if err != nil {
				return err
			}
			report, runErr := assess.Evaluate(cmd.Context(), cases, client)
			if err := encoder.Encode(report); err != nil {
				return err
			}
			return runErr
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "JSON array of labeled synthetic episode/fact pairs (required)")
	cmd.Flags().BoolVar(&live, "live", false, "send the supplied cases to TypeSafe and incur API charges")
	cmd.AddCommand(assessmentStatusCmd(), assessmentListCmd(), assessmentShowCmd(), assessmentReplayCmd(), assessmentResumeCmd())
	return cmd
}

func assessmentCall(cmd *cobra.Command, method string, params, out any) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	if err := callMemoryDaemon(ctx, method, params, out); err != nil {
		return fmt.Errorf("assess: %s: %w", method, err)
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
}

func assessmentStatusCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "status", Short: "Show assessment configuration and health", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var out daemon.AssessmentStatusResult
			return assessmentCall(cmd, "memory.assess.status", nil, &out)
		}}
	cmd.Flags().Bool("json", false, "output JSON (the default)")
	return cmd
}

func assessmentListCmd() *cobra.Command {
	var episode, after string
	var limit int
	cmd := &cobra.Command{Use: "list", Short: "List assessment jobs", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if limit < 1 || limit > 200 {
				return fmt.Errorf("assess list: --limit must be between 1 and 200")
			}
			var out assessstore.JobPage
			return assessmentCall(cmd, "memory.assess.list", &daemon.AssessmentListParams{EpisodeID: episode, After: after, Limit: limit}, &out)
		}}
	cmd.Flags().StringVar(&episode, "episode", "", "filter by episode ID")
	cmd.Flags().StringVar(&after, "after", "", "page cursor from previous result")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum jobs per page (1–200)")
	cmd.Flags().Bool("json", false, "output JSON (the default)")
	return cmd
}

func assessmentShowCmd() *cobra.Command {
	var include bool
	cmd := &cobra.Command{Use: "show ASSESSMENT_ID", Short: "Show assessment and provenance", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out daemon.AssessmentShowResult
			return assessmentCall(cmd, "memory.assess.show", &daemon.AssessmentShowParams{ID: args[0], IncludeContext: include}, &out)
		}}
	cmd.Flags().BoolVar(&include, "include-context", false, "include locally retained redacted evidence")
	cmd.Flags().Bool("json", false, "output JSON (the default)")
	return cmd
}

func assessmentReplayCmd() *cobra.Command {
	var live bool
	cmd := &cobra.Command{Use: "replay ASSESSMENT_ID", Short: "Preview or dispatch a linked replay", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out json.RawMessage
			params := &daemon.AssessmentReplayParams{ID: args[0], Live: live}
			if !live {
				return assessmentCall(cmd, "memory.assess.replay", params, &out)
			}
			// A live replay creates a new paid job. A lost response after the
			// daemon commits it must not cause an automatic second attempt.
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			if err := callMemoryDaemonOnce(ctx, "memory.assess.replay", params, &out); err != nil {
				if restarting(err) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("assess: live replay delivery unknown; inspect `scry memory assess list --json` before retrying: %w", err)
				}
				return fmt.Errorf("assess: memory.assess.replay: %w", err)
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}}
	cmd.Flags().BoolVar(&live, "live", false, "dispatch a new paid assessment attempt")
	cmd.Flags().Bool("json", false, "output JSON (the default)")
	return cmd
}

func assessmentResumeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "resume", Short: "Resume blocked assessment dispatch", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var out map[string]bool
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			if err := callMemoryDaemonOnce(ctx, "memory.assess.resume", nil, &out); err != nil {
				if restarting(err) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("assess: resume delivery unknown; inspect `scry memory assess status --json` before retrying: %w", err)
				}
				return fmt.Errorf("assess: memory.assess.resume: %w", err)
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}}
	cmd.Flags().Bool("json", false, "output JSON (the default)")
	return cmd
}
