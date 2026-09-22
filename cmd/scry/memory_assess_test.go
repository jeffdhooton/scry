package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/rpc"
)

type assessNoNetwork struct{ calls int }

func (n *assessNoNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	n.calls++
	return nil, errors.New("network forbidden in test")
}

func TestMemoryAssessMutationsDoNotRetryLostResponse(t *testing.T) {
	for _, args := range [][]string{
		{"assess", "replay", "job-1", "--live"},
		{"assess", "resume"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			socket := shortSocketPath(t)
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			var calls atomic.Int32
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					if _, err := bufio.NewReader(conn).ReadBytes('\n'); err == nil {
						calls.Add(1)
					}
					_ = conn.Close() // request arrived; pretend the response was lost
				}
			}()
			t.Setenv(memorySocketEnv, socket)
			cmd := memoryCmd()
			cmd.SetArgs(args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			err = cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "delivery unknown") {
				t.Fatalf("err = %v", err)
			}
			time.Sleep(2200 * time.Millisecond) // previous generic helper retried after 2s
			if got := calls.Load(); got != 1 {
				t.Fatalf("received %d requests, want one", got)
			}
		})
	}
}

func TestMemoryAssessRemoteCommandsAndSafeShow(t *testing.T) {
	socket := shortSocketPath(t)
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	server := rpc.NewServer()
	var show daemon.AssessmentShowParams
	server.Register("memory.assess.status", func(context.Context, json.RawMessage) (any, error) {
		return daemon.AssessmentStatusResult{KeyAvailable: true}, nil
	})
	server.Register("memory.assess.show", func(_ context.Context, raw json.RawMessage) (any, error) {
		if err := json.Unmarshal(raw, &show); err != nil {
			return nil, err
		}
		out := daemon.AssessmentShowResult{}
		if show.IncludeContext {
			out.Request = json.RawMessage(`{"source":"sensitive"}`)
		}
		return out, nil
	})
	server.Register("memory.assess.replay", func(_ context.Context, raw json.RawMessage) (any, error) {
		var p daemon.AssessmentReplayParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return map[string]any{"mode": "preview", "live": p.Live}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = server.Serve(ctx, ln) }()
	t.Setenv(memorySocketEnv, socket)
	for _, tc := range []struct {
		args             []string
		contains, absent string
	}{
		{[]string{"assess", "status", "--json"}, `"key_available":true`, "sensitive"},
		{[]string{"assess", "show", "job-1", "--json"}, `"job"`, "sensitive"},
		{[]string{"assess", "show", "job-1", "--include-context"}, "sensitive", ""},
		{[]string{"assess", "replay", "job-1"}, `"mode":"preview"`, `"live":true`},
	} {
		cmd := memoryCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if !strings.Contains(out.String(), tc.contains) || (tc.absent != "" && strings.Contains(out.String(), tc.absent)) {
			t.Fatalf("%v: %s", tc.args, out.String())
		}
	}
	if show.ID != "job-1" || !show.IncludeContext {
		t.Fatalf("show params: %+v", show)
	}
}

func TestMemoryAssessPreviewsWithoutNetworkEvenWithAPIKey(t *testing.T) {
	network := &assessNoNetwork{}
	prior := http.DefaultTransport
	http.DefaultTransport = network
	t.Cleanup(func() { http.DefaultTransport = prior })
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	cmd := memoryCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"assess", "--file", "../../docs/memory-assess/synthetic.json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Mode     string `json:"mode"`
		Requests []struct {
			ID      string         `json:"id"`
			Request map[string]any `json:"request"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Mode != "preview" || len(result.Requests) < 10 || network.calls != 0 {
		t.Fatalf("preview mode=%s cases=%d network=%d", result.Mode, len(result.Requests), network.calls)
	}
	if strings.Contains(out.String(), "expected") || strings.Contains(out.String(), "test-key") {
		t.Fatal("preview includes labels or credentials")
	}
}

func TestMemoryAssessLiveRequiresExplicitKeyAndValidDataset(t *testing.T) {
	for _, args := range [][]string{
		{"assess", "--live", "--file", "../../docs/memory-assess/synthetic.json"},
		{"assess"},
	} {
		t.Setenv("TYPESAFE_API_KEY", "")
		cmd := memoryCmd()
		cmd.SetArgs(args)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted args %v without required input", args)
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(path, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	network := &assessNoNetwork{}
	prior := http.DefaultTransport
	http.DefaultTransport = network
	t.Cleanup(func() { http.DefaultTransport = prior })
	cmd := memoryCmd()
	cmd.SetArgs([]string{"assess", "--live", "--file", path})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil || network.calls != 0 {
		t.Fatalf("invalid dataset err=%v calls=%d", err, network.calls)
	}
}
