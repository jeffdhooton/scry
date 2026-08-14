package doctor

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
)

// seedRepoManifest writes a manifest for repoPath into scryHome using the
// same layout resolution the doctor uses, and returns scryHome.
func seedRepoManifest(t *testing.T, repoPath string, m index.Manifest) string {
	t.Helper()
	home := t.TempDir()
	layout := index.Layout(home, repoPath)
	if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(layout.ManifestPath, b, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return home
}

func TestCheckCurrentRepo_NamesTheFailingIndexer(t *testing.T) {
	repo := t.TempDir()
	stderr := "full compiler context\nfinal complaint\n"
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:  repo,
		Languages: []string{"php", "python"},
		IndexedAt: time.Now(),
		Status:    "partial",
		Indexers: []index.IndexerResult{
			{Language: "php", Tier: index.TierPrimary, Status: index.IndexerOK},
			{
				Language: "python", Tier: index.TierPrimary, Status: index.IndexerMissing,
				Error:  "scip-python not found on PATH",
				Remedy: "npm i -g @sourcegraph/scip-python",
				Stderr: stderr,
			},
		},
	})

	got := checkCurrentRepo(home, repo)
	if got.Status != StatusWarn {
		t.Errorf("Status = %v, want Warn", got.Status)
	}
	if !strings.Contains(got.Detail, "python") || !strings.Contains(got.Detail, "missing") {
		t.Errorf("Detail = %q, want it to name the python indexer as missing", got.Detail)
	}
	if got.Remedy != "npm i -g @sourcegraph/scip-python" {
		t.Errorf("Remedy = %q, want the install command", got.Remedy)
	}
	if got.Stderr != stderr {
		t.Errorf("Stderr = %q, want full captured tail %q", got.Stderr, stderr)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal check: %v", err)
	}
	if !strings.Contains(string(b), `"stderr":"full compiler context\nfinal complaint\n"`) {
		t.Errorf("JSON = %s, want full stderr field", b)
	}
}

func TestCheckCurrentRepo_LegacyManifestStillRenders(t *testing.T) {
	// A manifest written before Indexers existed must render as it does
	// today — no panic, no empty breakdown appended.
	repo := t.TempDir()
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:  repo,
		Languages: []string{"go"},
		IndexedAt: time.Now(),
		Status:    "ready",
	})

	got := checkCurrentRepo(home, repo)
	if got.Status != StatusPass {
		t.Errorf("Status = %v, want Pass", got.Status)
	}
	if !strings.Contains(got.Detail, "ready") {
		t.Errorf("Detail = %q, want it to contain the status label", got.Detail)
	}
	// Exactly one em-dash is expected: the normal "<status> — <stats>"
	// separator at doctor.go:921. A second one (with either one or two
	// trailing spaces) would mean the empty-breakdown guard at doctor.go:939
	// regressed and appended a dangling " — " with nothing after it.
	if strings.Count(got.Detail, "—") != 1 {
		t.Errorf("Detail = %q, want exactly one em-dash separator", got.Detail)
	}
}

func TestStaleDaemonIndexerCheckNamesRestartRemedy(t *testing.T) {
	got := staleDaemonIndexerCheck("indexers.scip_typescript", "scip-typescript")
	if got.Status != StatusWarn {
		t.Fatalf("Status = %v, want Warn", got.Status)
	}
	want := "scip-typescript is installed but the running daemon started before it; run `scry daemon restart`"
	if got.Detail != want {
		t.Errorf("Detail = %q, want %q", got.Detail, want)
	}
	if got.Remedy != "scry daemon restart" {
		t.Errorf("Remedy = %q, want scry daemon restart", got.Remedy)
	}
}
