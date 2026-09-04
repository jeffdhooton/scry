package doctor

import (
	"os"
	"path/filepath"
	"strings"
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

func TestEvalMemoryStatusCallsOutAProviderRefusal(t *testing.T) {
	now := time.Date(2026, 9, 3, 13, 30, 0, 0, time.UTC)
	recent := now.Add(-5 * time.Minute)
	sweep := now.Add(-10 * time.Minute)
	base := daemon.MemoryStatusResult{
		Models: []string{"glm-5.3-flash", "deepseek-v4-flash"}, WorkerRunning: true,
		LastIngestAt: &recent, LastSweepAt: &sweep, LastExtractOKAt: &recent,
	}

	// The whole chain refused: a running worker is not enough.
	all := base
	all.ModelsRefusing = []string{"glm-5.3-flash", "deepseek-v4-flash"}
	all.AllModelsRefusing = true
	if got := findCheck(t, evalMemoryStatus(&all, now), "memory.extraction"); got.Status != StatusFail {
		t.Errorf("a fully refused chain = %s, want fail: %s", got.Status, got.Detail)
	}

	// One of two refused is a warning, not a failure: the chain still runs.
	one := base
	one.ModelsRefusing = []string{"deepseek-v4-flash"}
	if got := findCheck(t, evalMemoryStatus(&one, now), "memory.extraction"); got.Status != StatusWarn {
		t.Errorf("one refused model = %s, want warn: %s", got.Status, got.Detail)
	}

	// Nothing refused stays a pass.
	if got := findCheck(t, evalMemoryStatus(&base, now), "memory.extraction"); got.Status != StatusPass {
		t.Errorf("a healthy chain = %s: %s", got.Status, got.Detail)
	}
}

func TestEvalMemoryStatusFailsAStoppedQueue(t *testing.T) {
	now := time.Date(2026, 9, 3, 13, 30, 0, 0, time.UTC)
	recent := now.Add(-5 * time.Minute)
	stale := now.Add(-90 * time.Minute)
	res := daemon.MemoryStatusResult{
		Models: []string{"glm-5.3-flash"}, WorkerRunning: true,
		LastIngestAt: &recent, LastSweepAt: &recent,
		QueueReady: 1120, LastExtractOKAt: &stale,
	}
	if got := findCheck(t, evalMemoryStatus(&res, now), "memory.queue"); got.Status != StatusFail {
		t.Errorf("a queue with work and no progress for 90m = %s, want fail: %s", got.Status, got.Detail)
	}

	// An empty queue that has been quiet is idle, not stopped.
	idle := res
	idle.QueueReady = 0
	if got := findCheck(t, evalMemoryStatus(&idle, now), "memory.queue"); got.Status != StatusPass {
		t.Errorf("an empty queue = %s, want pass: %s", got.Status, got.Detail)
	}
}

// A queue that has never extracted anything is the worst case, not an
// exempt one: a pipeline that has produced nothing since it started looks
// identical to one that stopped.
func TestEvalMemoryStatusFailsAQueueThatNeverSucceeded(t *testing.T) {
	now := time.Date(2026, 9, 3, 13, 30, 0, 0, time.UTC)
	recent := now.Add(-5 * time.Minute)
	res := daemon.MemoryStatusResult{
		Models: []string{"glm-5.3-flash"}, WorkerRunning: true,
		LastIngestAt: &recent, LastSweepAt: &recent,
		QueueReady: 1120, LastExtractOKAt: nil,
	}
	if got := findCheck(t, evalMemoryStatus(&res, now), "memory.queue"); got.Status != StatusFail {
		t.Errorf("a queue with work that has never extracted = %s, want fail: %s", got.Status, got.Detail)
	}
}

// The per-source breakdown was recorded from 3f9fbb6 onward and read by
// nothing: a grader found MetaLastSweepReport written once and never loaded.
// A number nobody can see does not answer the question it was added for.
func TestSweepSourcesCheck(t *testing.T) {
	now := time.Now()
	base := &daemon.MemoryStatusResult{
		Models: []string{"m"}, WorkerRunning: true,
		LastIngestAt: &now, LastSweepAt: &now, LastExtractOKAt: &now,
	}
	find := func(cs []Check) *Check {
		for i := range cs {
			if cs[i].ID == "memory.sweep_sources" {
				return &cs[i]
			}
		}
		return nil
	}

	t.Run("absent when the sweep reported no breakdown", func(t *testing.T) {
		if c := find(evalMemoryStatus(base, now)); c != nil {
			t.Errorf("check should not appear: %+v", c)
		}
	})

	t.Run("passes and names every source that produced episodes", func(t *testing.T) {
		res := *base
		res.LastSweepBySource = map[string]int{"claude": 3, "codex": 2, "kimi": 1}
		c := find(evalMemoryStatus(&res, now))
		if c == nil || c.Status != StatusPass {
			t.Fatalf("check = %+v", c)
		}
		for _, want := range []string{"claude 3", "codex 2", "kimi 1"} {
			if !strings.Contains(c.Detail, want) {
				t.Errorf("detail %q missing %q", c.Detail, want)
			}
		}
	})

	t.Run("warns when a source was read and produced nothing", func(t *testing.T) {
		res := *base
		res.LastSweepBySource = map[string]int{"claude": 3, "kimi": 0}
		c := find(evalMemoryStatus(&res, now))
		if c == nil || c.Status != StatusWarn {
			t.Fatalf("a silent source must warn: %+v", c)
		}
		if !strings.Contains(c.Detail, "kimi") || c.Remedy == "" {
			t.Errorf("check = %+v", c)
		}
	})
}
