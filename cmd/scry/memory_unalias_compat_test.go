package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jeffdhooton/scry/internal/daemon"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

func TestIndependentUnaliasOldDaemonAndSwap(t *testing.T) {
	for _, swap := range []bool{false, true} {
		t.Run(map[bool]string{false: "old-from-start", true: "downgrade-between-connections"}[swap], func(t *testing.T) {
			socket := shortSocketPath(t)
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			expected := memstore.AliasRepairExpected{Plan: "reviewed-plan", Facts: "reviewed-facts"}
			var calls, writes atomic.Int32
			server := rpc.NewServer()
			server.Register("memory.unalias", func(_ context.Context, raw json.RawMessage) (any, error) {
				var p daemon.MemoryUnaliasParams
				if err := json.Unmarshal(raw, &p); err != nil {
					return nil, err
				}
				n := calls.Add(1)
				if swap && n == 1 {
					return daemon.MemoryUnaliasResult{DryRun: true, Preview: memstore.AliasRepairPreview{Ready: true, Expected: expected}}, nil
				}
				// Previous daemon accepts dry_run but ignores unknown expected fields.
				dry := p.DryRun != nil && *p.DryRun
				if !dry {
					writes.Add(1)
				}
				return map[string]any{"dry_run": dry, "dropped": 1, "refused": 0, "details": []string{}}, nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = server.Serve(ctx, listener) }()
			t.Setenv(memorySocketEnv, socket)
			request := memstore.AliasRepairRequest{Drops: []memstore.AliasDrop{{Entity: "app", Alias: "surface", Why: "reviewed"}}, Expected: &expected}
			data, _ := json.Marshal(request)
			file := filepath.Join(t.TempDir(), "review.json")
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := memoryUnaliasCmd()
			cmd.SetArgs([]string{"--file", file, "--apply"})
			err = cmd.Execute()
			t.Logf("swap=%v calls=%d writes=%d err=%v", swap, calls.Load(), writes.Load(), err)
			if writes.Load() != 0 {
				t.Fatal("old daemon received apply after new-daemon preview; ignored expected inputs")
			}
			if err == nil {
				t.Fatal("old daemon not refused")
			}
		})
	}
}
