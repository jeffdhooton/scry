package install

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPlanFor(t *testing.T) {
	tests := []struct {
		name           string
		tool           string
		env            Env
		wantSource     Source
		wantPackage    string
		wantCommand    []string
		wantActionable bool
		wantReason     string
	}{
		{
			name:           "typescript via npm",
			tool:           "typescript",
			env:            Env{NPMPath: "/opt/homebrew/bin/npm"},
			wantSource:     SourceNPM,
			wantPackage:    "@sourcegraph/scip-typescript",
			wantCommand:    []string{"npm", "i", "-g", "@sourcegraph/scip-typescript"},
			wantActionable: true,
		},
		{
			name:           "python via npm",
			tool:           "python",
			env:            Env{NPMPath: "/usr/local/bin/npm"},
			wantSource:     SourceNPM,
			wantPackage:    "@sourcegraph/scip-python",
			wantCommand:    []string{"npm", "i", "-g", "@sourcegraph/scip-python"},
			wantActionable: true,
		},
		{
			name:        "npm missing names prerequisite",
			tool:        "typescript",
			wantSource:  SourceNPM,
			wantPackage: "@sourcegraph/scip-typescript",
			wantCommand: []string{"npm", "i", "-g", "@sourcegraph/scip-typescript"},
			wantReason:  "npm",
		},
		{
			name:        "already installed",
			tool:        "python",
			env:         Env{ToolPath: "/usr/local/bin/scip-python"},
			wantSource:  SourceNPM,
			wantPackage: "@sourcegraph/scip-python",
			wantCommand: []string{"npm", "i", "-g", "@sourcegraph/scip-python"},
			wantReason:  "already installed",
		},
		{
			name:           "go release",
			tool:           "go",
			wantSource:     SourceGitHubRelease,
			wantActionable: true,
		},
		{
			name:       "php manual",
			tool:       "php",
			wantSource: SourceManual,
			wantReason: "PHP 8.3+",
		},
		{
			name:       "php already installed",
			tool:       "php",
			env:        Env{ToolPath: "/usr/bin/php"},
			wantSource: SourceManual,
			wantReason: "already installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PlanFor(tt.tool, tt.env)
			if got.Tool != tt.tool {
				t.Errorf("Tool = %q, want %q", got.Tool, tt.tool)
			}
			if got.Source != tt.wantSource {
				t.Errorf("Source = %v, want %v", got.Source, tt.wantSource)
			}
			if got.Package != tt.wantPackage {
				t.Errorf("Package = %q, want %q", got.Package, tt.wantPackage)
			}
			if !reflect.DeepEqual(got.Command, tt.wantCommand) {
				t.Errorf("Command = %#v, want %#v", got.Command, tt.wantCommand)
			}
			if got.Actionable != tt.wantActionable {
				t.Errorf("Actionable = %v, want %v", got.Actionable, tt.wantActionable)
			}
			if tt.wantReason != "" && !strings.Contains(got.Reason, tt.wantReason) {
				t.Errorf("Reason = %q, want it to contain %q", got.Reason, tt.wantReason)
			}
		})
	}
}

func TestStaleDaemonPATH(t *testing.T) {
	started := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		env  Env
		want bool
	}{
		{"installed after daemon start", Env{ToolPath: "/bin/scip-typescript", DaemonStartedAt: started, ToolInstalledAt: started.Add(time.Minute)}, true},
		{"installed before daemon start", Env{ToolPath: "/bin/scip-typescript", DaemonStartedAt: started, ToolInstalledAt: started.Add(-time.Minute)}, false},
		{"tool absent", Env{DaemonStartedAt: started, ToolInstalledAt: started.Add(time.Minute)}, false},
		{"daemon not running", Env{ToolPath: "/bin/scip-typescript", ToolInstalledAt: started.Add(time.Minute)}, false},
		{"install time unknown", Env{ToolPath: "/bin/scip-typescript", DaemonStartedAt: started}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StaleDaemonPATH(tt.env); got != tt.want {
				t.Errorf("StaleDaemonPATH() = %v, want %v", got, tt.want)
			}
		})
	}
}
