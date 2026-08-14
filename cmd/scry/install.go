package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeffdhooton/scry/internal/install"
)

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [typescript|python...]",
		Short: "Install external SCIP indexers",
		Long: `Install the npm-published SCIP indexers used for TypeScript,
JavaScript, and Python repositories. With no arguments, installs both
scip-typescript and scip-python. Existing installations are left unchanged.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tools, err := installToolArgs(args)
			if err != nil {
				return err
			}
			return provisionTools(cmd.Context(), os.Stdout, tools)
		},
	}
}

func installToolArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return []string{"typescript", "python"}, nil
	}
	tools := make([]string, 0, len(args))
	seen := map[string]bool{}
	for _, arg := range args {
		tool := strings.TrimPrefix(arg, "scip-")
		if tool != "typescript" && tool != "python" {
			return nil, fmt.Errorf("unsupported indexer %q (choose typescript or python)", arg)
		}
		if !seen[tool] {
			seen[tool] = true
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

func provisionTools(parent context.Context, out io.Writer, tools []string) error {
	npmPath, _ := exec.LookPath("npm")
	var failed bool
	for _, tool := range tools {
		binary := install.BinaryName(tool)
		toolPath, _ := exec.LookPath(binary)
		plan := install.PlanFor(tool, install.Env{NPMPath: npmPath, ToolPath: toolPath})
		command := strings.Join(plan.Command, " ")

		if toolPath != "" {
			fmt.Fprintf(out, "%s: already installed at %s\n", binary, toolPath)
			continue
		}
		if !plan.Actionable {
			failed = true
			fmt.Fprintf(out, "%s: %s\n", binary, plan.Reason)
			fmt.Fprintln(out, "  prerequisite: install Node.js/npm and ensure npm is on PATH")
			fmt.Fprintln(out, "  run: "+command)
			continue
		}

		fmt.Fprintf(out, "%s: running %s\n", binary, command)
		ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
		res, err := install.Provision(ctx, plan)
		cancel()
		if err != nil {
			failed = true
			fmt.Fprintf(out, "%s: install failed: %v\n", binary, err)
			fmt.Fprintln(out, "  run: "+command)
			continue
		}
		fmt.Fprintf(out, "%s: installed and verified at %s", binary, res.Path)
		if res.Version != "" {
			fmt.Fprintf(out, " (%s)", res.Version)
		}
		fmt.Fprintln(out)
	}
	if failed {
		return errors.New("one or more indexers could not be installed")
	}
	return nil
}
