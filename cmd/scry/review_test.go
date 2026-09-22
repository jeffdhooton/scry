package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/rpc"
	"github.com/spf13/cobra"
)

func reviewTestCommand() *cobra.Command {
	root := &cobra.Command{Use: "scry", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("repo", "", "repository")
	root.PersistentFlags().Bool("pretty", false, "pretty output")
	root.AddCommand(reviewCmd())
	return root
}

func TestReviewCommandRoutesJSONToExplicitLocalSocket(t *testing.T) {
	socket := shortSocketPath(t)
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	server := rpc.NewServer()
	requests := make(chan rpc.Request, 10)
	for _, action := range []string{"status", "resume", "preview", "run", "list", "get"} {
		server.Register("review."+action, func(_ context.Context, params json.RawMessage) (any, error) {
			requests <- rpc.Request{Method: "review." + action, Params: params}
			return json.RawMessage(`{"snapshot_id":"snapshot-1","freshness":"stale","provisional":true}`), nil
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = server.Serve(ctx, ln) }()
	t.Setenv(memorySocketEnv, filepath.Join(t.TempDir(), "not-the-review-daemon.sock"))
	repo, err := filepath.Abs("relative-repository")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args   []string
		method string
		params map[string]string
	}{
		{[]string{"status"}, "review.status", map[string]string{}},
		{[]string{"resume"}, "review.resume", map[string]string{}},
		{[]string{"preview", "--repo", "relative-repository"}, "review.preview", map[string]string{"repo": repo}},
		{[]string{"run", "--repo", "relative-repository"}, "review.run", map[string]string{"repo": repo}},
		{[]string{"list", "--repo", "relative-repository"}, "review.list", map[string]string{"repo": repo}},
		{[]string{"get", "review-1"}, "review.get", map[string]string{"id": "review-1"}},
	} {
		for _, pretty := range []bool{false, true} {
			cmd := reviewTestCommand()
			args := append([]string{"review", "--socket", socket}, tc.args...)
			if pretty {
				args = append(args, "--pretty")
			}
			cmd.SetArgs(args)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("%v: %v", args, err)
			}
			var request rpc.Request
			select {
			case request = <-requests:
			default:
				t.Fatalf("command %v returned without routing to the daemon: %s", args, out.String())
			}
			var got map[string]string
			if err := json.Unmarshal(request.Params, &got); err != nil {
				t.Fatal(err)
			}
			if request.Method != tc.method || !reflect.DeepEqual(got, tc.params) {
				t.Fatalf("request=%s %s want %s %+v", request.Method, request.Params, tc.method, tc.params)
			}
			var result map[string]any
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatalf("non-JSON output: %s", out.String())
			}
			if result["snapshot_id"] != "snapshot-1" || result["freshness"] != "stale" || result["provisional"] != true {
				t.Fatalf("lost result metadata: %+v", result)
			}
			if strings.Contains(out.String(), "\n  ") != pretty {
				t.Fatalf("pretty=%v output=%s", pretty, out.String())
			}
		}
	}
}

func TestReviewCommandRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"get"}, {"get", ""}, {"get", "../escape"}, {"get", "has spaces"}, {"get", strings.Repeat("a", 201)},
		{"get", "one", "two"}, {"status", "extra"}, {"resume", "extra"}, {"preview", "extra"}, {"run", "extra"}, {"list", "extra"},
	} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			cmd := reviewTestCommand()
			cmd.SetArgs(append([]string{"review", "--socket", "/nonexistent/review-test.sock"}, args...))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			err := cmd.Execute()
			if err == nil || strings.Contains(err.Error(), "dial unix") {
				t.Fatalf("expected argument validation before dialing, got %v", err)
			}
		})
	}
}

func TestReviewCommandPreservesDaemonErrors(t *testing.T) {
	socket := shortSocketPath(t)
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	server := rpc.NewServer()
	server.Register("review.run", func(context.Context, json.RawMessage) (any, error) {
		return nil, fmt.Errorf("repository is not configured")
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = server.Serve(ctx, ln) }()
	cmd := reviewTestCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"review", "run", "--socket", socket})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "repository is not configured") {
		t.Fatalf("err=%v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("failure printed misleading result: %s", out.String())
	}
}

// Default routing must remain local even when memory is served remotely.
func TestReviewCommandDefaultSocketIgnoresMemoryRoute(t *testing.T) {
	home, err := os.MkdirTemp("/tmp", "scry-review-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv(memorySocketEnv, filepath.Join(home, "remote-memory.sock"))
	dir := filepath.Join(home, ".scry")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("unix", filepath.Join(dir, "scryd.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	server := rpc.NewServer()
	server.Register("review.preview", func(_ context.Context, raw json.RawMessage) (any, error) {
		var params map[string]string
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		if params["repo"] != cwd {
			return nil, fmt.Errorf("repo=%q want cwd %q", params["repo"], cwd)
		}
		return map[string]string{"source": "local"}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = server.Serve(ctx, ln) }()
	cmd := reviewTestCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"review", "preview"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"source":"local"`) {
		t.Fatalf("wrong daemon: %s", out.String())
	}
}
