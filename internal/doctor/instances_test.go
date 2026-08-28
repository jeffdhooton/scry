package doctor

import (
	"errors"
	"strings"
	"testing"
)

const psSample = `  1234 /sbin/launchd
 43409 /Users/jeff/go/bin/scry start --foreground
 51773 /Users/jeff/go/bin/scry start --foreground
 51786 /Users/jeff/go/bin/scry start --foreground
 60001 /Users/jeff/go/bin/scry mcp
 60002 /Users/jeff/go/bin/scry memory sweep
 60003 /bin/zsh -lc source ~/.secrets.zsh 2>/dev/null; exec /Users/jeff/go/bin/scry start --foreground
 60004 scry start --foreground
 60005 /usr/bin/grep scry start --foreground
`

func TestParseForegroundDaemonPIDs(t *testing.T) {
	got := parseForegroundDaemonPIDs(psSample)
	want := []int{43409, 51773, 51786, 60004}
	if len(got) != len(want) {
		t.Fatalf("pids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pids = %v, want %v", got, want)
		}
	}
}

func TestEvalDaemonInstances(t *testing.T) {
	cases := []struct {
		name      string
		canonical int
		pids      []int
		status    Status
		mentions  []string
	}{
		{"single canonical daemon", 43409, []int{43409}, StatusPass, nil},
		{"no daemon", 0, nil, StatusPass, nil},
		{"orphans beside canonical", 43409, []int{43409, 51773, 51786}, StatusFail, []string{"51773", "51786", "orphan"}},
		{"daemons but none canonical", 0, []int{51773}, StatusFail, []string{"51773"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := evalDaemonInstances(tc.canonical, tc.pids)
			if c.ID != "daemon.instances" {
				t.Errorf("ID = %q", c.ID)
			}
			if c.Status != tc.status {
				t.Errorf("Status = %q, want %q (detail %q)", c.Status, tc.status, c.Detail)
			}
			for _, m := range tc.mentions {
				if !strings.Contains(c.Detail, m) {
					t.Errorf("Detail %q does not mention %q", c.Detail, m)
				}
			}
			if tc.status == StatusFail && c.Remedy == "" {
				t.Error("failing check has no remedy")
			}
		})
	}
}

func TestEvalMemoryUIHealth(t *testing.T) {
	cases := []struct {
		name      string
		canonical int
		probe     uiProbe
		status    Status
		mentions  []string
	}{
		{"served by canonical daemon", 43409, uiProbe{StatusCode: 200, PID: 43409, MemoryOK: true}, StatusPass, nil},
		{"served by orphan", 43409, uiProbe{StatusCode: 200, PID: 51773, MemoryOK: false, MemoryError: "Cannot acquire directory lock"}, StatusFail, []string{"51773", "43409"}},
		{"ui cannot open store", 43409, uiProbe{StatusCode: 200, PID: 43409, MemoryOK: false, MemoryError: "Cannot acquire directory lock"}, StatusFail, []string{"directory lock"}},
		{"non-200", 43409, uiProbe{StatusCode: 500}, StatusFail, []string{"500"}},
		{"daemon up but nothing on the port", 43409, uiProbe{Err: errors.New("connection refused")}, StatusFail, []string{"7279"}},
		{"no daemon", 0, uiProbe{Err: errors.New("connection refused")}, StatusSkip, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := evalMemoryUIHealth(tc.canonical, "127.0.0.1:7279", tc.probe)
			if c.ID != "daemon.memory_ui" {
				t.Errorf("ID = %q", c.ID)
			}
			if c.Status != tc.status {
				t.Errorf("Status = %q, want %q (detail %q)", c.Status, tc.status, c.Detail)
			}
			for _, m := range tc.mentions {
				if !strings.Contains(c.Detail, m) {
					t.Errorf("Detail %q does not mention %q", c.Detail, m)
				}
			}
		})
	}
}
