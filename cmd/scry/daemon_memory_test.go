package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/rpc"
)

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
