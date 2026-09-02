package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/daemon"
)

func findCheck(t *testing.T, checks []Check, id string) Check {
	t.Helper()
	for _, c := range checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("check %s missing from %+v", id, checks)
	return Check{}
}

func TestEvalMemoryStatus(t *testing.T) {
	now := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *time.Time { t := now.Add(-d); return &t }
	healthy := func() *daemon.MemoryStatusResult {
		return &daemon.MemoryStatusResult{
			Models: []string{"glm-5.3-flash", "deepseek-v4-flash"}, WorkerRunning: true,
			LastIngestAt: ago(40 * time.Minute), LastSweepAt: ago(20 * time.Minute), LastExtractOKAt: ago(5 * time.Minute),
		}
	}
	cases := []struct {
		name   string
		mutate func(*daemon.MemoryStatusResult)
		id     string
		want   Status
	}{
		{"healthy extraction", nil, "memory.extraction", StatusPass},
		{"healthy ingest", nil, "memory.ingest_age", StatusPass},
		{"healthy sweep", nil, "memory.sweep_age", StatusPass},
		{"healthy queue", nil, "memory.queue", StatusPass},
		{"dormant fails", func(r *daemon.MemoryStatusResult) { r.Dormant = true }, "memory.extraction", StatusFail},
		{"worker down fails", func(r *daemon.MemoryStatusResult) { r.WorkerRunning = false }, "memory.extraction", StatusFail},
		{"ingest 5h59m passes", func(r *daemon.MemoryStatusResult) { r.LastIngestAt = ago(5*time.Hour + 59*time.Minute) }, "memory.ingest_age", StatusPass},
		{"ingest 6h01m fails", func(r *daemon.MemoryStatusResult) { r.LastIngestAt = ago(6*time.Hour + time.Minute) }, "memory.ingest_age", StatusFail},
		{"ingest never fails", func(r *daemon.MemoryStatusResult) { r.LastIngestAt = nil }, "memory.ingest_age", StatusFail},
		{"sweep 3h warns", func(r *daemon.MemoryStatusResult) { r.LastSweepAt = ago(3 * time.Hour) }, "memory.sweep_age", StatusWarn},
		{"sweep never warns", func(r *daemon.MemoryStatusResult) { r.LastSweepAt = nil }, "memory.sweep_age", StatusWarn},
		{"parked warns", func(r *daemon.MemoryStatusResult) { r.QueueParked = 1 }, "memory.queue", StatusWarn},
		{"deep queue warns", func(r *daemon.MemoryStatusResult) { r.QueueReady = 51 }, "memory.queue", StatusWarn},
		{"shallow queue passes", func(r *daemon.MemoryStatusResult) { r.QueueReady = 5; r.QueueBackoff = 2 }, "memory.queue", StatusPass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := healthy()
			if tc.mutate != nil {
				tc.mutate(res)
			}
			c := findCheck(t, evalMemoryStatus(res, now), tc.id)
			if c.Status != tc.want {
				t.Errorf("%s status = %s, want %s (%s)", tc.id, c.Status, tc.want, c.Detail)
			}
			if c.Status != StatusPass && c.Remedy == "" {
				t.Errorf("%s has no remedy", tc.id)
			}
			if c.Category != CategoryMemory {
				t.Errorf("category = %s", c.Category)
			}
		})
	}
}

func TestMemorySocketResolutionOrder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SCRY_MEMORY_SOCKET", "")
	if p, src := memorySocket(home); src != "local daemon" || p != daemon.LayoutFor(home).SocketPath {
		t.Errorf("no config: %q %q", p, src)
	}
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte("memory:\n  socket: /tmp/shared.sock\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if p, src := memorySocket(home); p != "/tmp/shared.sock" || src != "config.yaml memory.socket" {
		t.Errorf("config: %q %q", p, src)
	}
	t.Setenv("SCRY_MEMORY_SOCKET", "/tmp/env.sock")
	if p, src := memorySocket(home); p != "/tmp/env.sock" || src != "SCRY_MEMORY_SOCKET" {
		t.Errorf("env: %q %q", p, src)
	}
}

func TestCheckMemoryUnreachableIsOneFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SCRY_MEMORY_SOCKET", filepath.Join(home, "missing.sock"))
	checks := checkMemory(home, 200*time.Millisecond)
	if len(checks) != 1 || checks[0].Status != StatusFail || checks[0].ID != "memory.daemon" {
		t.Errorf("checks = %+v", checks)
	}
}
