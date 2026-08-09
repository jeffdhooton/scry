package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/index"
)

// writeManifests materializes fake ~/.scry/repos/<n>/manifest.json entries
// and returns the scryHome that contains them.
func writeManifests(t *testing.T, manifests ...index.Manifest) string {
	t.Helper()
	home := t.TempDir()
	for i, m := range manifests {
		dir := filepath.Join(home, "repos", "repo"+string(rune('a'+i)))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return home
}

func TestCheckIndexerImpact_CountsOnlyPrimaryRepos(t *testing.T) {
	home := writeManifests(t,
		index.Manifest{
			RepoPath: "/a",
			Indexers: []index.IndexerResult{
				{Language: "python", Tier: index.TierPrimary, Status: index.IndexerMissing},
			},
		},
		index.Manifest{
			RepoPath: "/b",
			Indexers: []index.IndexerResult{
				{Language: "python", Tier: index.TierPrimary, Status: index.IndexerMissing},
			},
		},
		index.Manifest{
			// Incidental — skipped, not degraded. Must not be counted.
			RepoPath: "/c",
			Indexers: []index.IndexerResult{
				{Language: "python", Tier: index.TierIncidental, Status: index.IndexerSkipped},
			},
		},
	)
	prior := []Check{
		{ID: "indexers.scip_python", Status: StatusWarn, Remedy: "npm i -g @sourcegraph/scip-python"},
	}

	got := checkIndexerImpact(home, prior)
	if got.Status != StatusWarn {
		t.Errorf("Status = %v, want Warn", got.Status)
	}
	if !strings.Contains(got.Detail, "2 indexed repo") {
		t.Errorf("Detail = %q, want it to name 2 affected repos", got.Detail)
	}
	if !strings.Contains(got.Detail, "python") {
		t.Errorf("Detail = %q, want it to name python", got.Detail)
	}
	if got.Remedy != "npm i -g @sourcegraph/scip-python" {
		t.Errorf("Remedy = %q, want the install command", got.Remedy)
	}
}

func TestCheckIndexerImpact_NothingMissingPasses(t *testing.T) {
	home := writeManifests(t, index.Manifest{
		RepoPath: "/a",
		Indexers: []index.IndexerResult{
			{Language: "go", Tier: index.TierPrimary, Status: index.IndexerOK},
		},
	})
	prior := []Check{{ID: "indexers.scip_python", Status: StatusPass}}

	got := checkIndexerImpact(home, prior)
	if got.Status != StatusPass {
		t.Errorf("Status = %v, want Pass when no indexer is missing", got.Status)
	}
}

func TestCheckIndexerImpact_MissingButNoReposAffected(t *testing.T) {
	// scip-python missing, but no indexed repo treats Python as primary.
	// Not worth a warning.
	home := writeManifests(t, index.Manifest{
		RepoPath: "/a",
		Indexers: []index.IndexerResult{
			{Language: "php", Tier: index.TierPrimary, Status: index.IndexerOK},
		},
	})
	prior := []Check{{ID: "indexers.scip_python", Status: StatusWarn, Remedy: "npm i -g @sourcegraph/scip-python"}}

	got := checkIndexerImpact(home, prior)
	if got.Status != StatusPass {
		t.Errorf("Status = %v, want Pass when no repo is affected", got.Status)
	}
}
