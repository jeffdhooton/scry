package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/daemon"
	"github.com/jeffdhooton/scry/internal/rpc"
)

func TestMemoryOrientResolvesCwdBeforeRemoteRPC(t *testing.T) {
	socket := shortSocketPath(t)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan daemon.MemoryOrientParams, 1)
	server := rpc.NewServer()
	server.Register("memory.orient", func(_ context.Context, raw json.RawMessage) (any, error) {
		var p daemon.MemoryOrientParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		received <- p
		return map[string]string{"markdown": "orientation fixture"}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = server.Serve(ctx, listener) }()
	t.Setenv(memorySocketEnv, socket)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, wd},
		{"dot", []string{"--cwd", "."}, wd},
		{"relative", []string{"--cwd", "../scry"}, filepath.Clean(filepath.Join(wd, "../scry"))},
		{"absolute", []string{"--cwd", "/client/repo"}, "/client/repo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := memoryOrientCmd()
			cmd.SetArgs(append(tc.args, "--budget", "731"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			p := <-received
			if p.Cwd != tc.want || p.Budget != 731 {
				t.Fatalf("RPC params = %+v, want cwd=%q budget=731", p, tc.want)
			}
		})
	}
}

func TestDialMemoryDaemonUsesConfiguredSocket(t *testing.T) {
	socket := shortSocketPath(t)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	server := rpc.NewServer()
	server.Register("memory.test", func(_ context.Context, _ json.RawMessage) (any, error) {
		return map[string]bool{"remote": true}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = server.Serve(ctx, listener) }()

	t.Setenv(memorySocketEnv, socket)
	client, err := dialMemoryDaemon()
	if err != nil {
		t.Fatalf("dialMemoryDaemon: %v", err)
	}
	defer client.Close()

	var result map[string]bool
	if err := client.Call(context.Background(), "memory.test", nil, &result); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !result["remote"] {
		t.Fatalf("result = %#v, want remote=true", result)
	}
}

func TestDialMemoryDaemonNamesBrokenConfiguredSocket(t *testing.T) {
	socket := filepath.Join("/tmp", "scry-memory-definitely-missing.sock")
	_ = os.Remove(socket)
	t.Setenv(memorySocketEnv, socket)
	_, err := dialMemoryDaemon()
	if err == nil {
		t.Fatal("dialMemoryDaemon unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), memorySocketEnv) || !strings.Contains(err.Error(), socket) {
		t.Fatalf("error does not identify configured socket: %v", err)
	}
}

func shortSocketPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("/tmp", "scry-memory-*.sock")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		t.Fatalf("close temp socket placeholder: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove temp socket placeholder: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}
