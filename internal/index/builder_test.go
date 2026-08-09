package index

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/sources/golang"
)

func TestManifest_LegacyJSONWithoutIndexers(t *testing.T) {
	// A manifest written before this feature must still unmarshal, with
	// Indexers nil, and must not lose fields on re-marshal. The 44 repos
	// already on disk depend on this.
	legacy := `{
	  "schema_version": 2,
	  "repo_path": "/Users/jeff/workspace/example",
	  "languages": ["php", "typescript"],
	  "indexed_at": "2026-08-01T12:00:00Z",
	  "status": "partial",
	  "stats": {"documents": 10, "symbols": 20, "references": 30}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(legacy), &m); err != nil {
		t.Fatalf("unmarshal legacy manifest: %v", err)
	}
	if m.Indexers != nil {
		t.Errorf("Indexers = %v, want nil for a legacy manifest", m.Indexers)
	}
	if m.Status != "partial" {
		t.Errorf("Status = %q, want partial", m.Status)
	}
	if len(m.Languages) != 2 {
		t.Errorf("Languages = %v, want 2 entries", m.Languages)
	}
	if m.Stats.Documents != 10 {
		t.Errorf("Stats.Documents = %d, want 10", m.Stats.Documents)
	}

	// Re-marshal must not emit an empty "indexers" key.
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if _, present := round["indexers"]; present {
		t.Error("re-marshalled legacy manifest must omit the empty indexers key")
	}
}

func TestManifest_IndexersRoundTrip(t *testing.T) {
	m := Manifest{
		SchemaVersion: 2,
		RepoPath:      "/tmp/example",
		Languages:     []string{"php"},
		IndexedAt:     time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC),
		Status:        "ready",
		Indexers: []IndexerResult{
			{Language: "php", Status: IndexerOK, Tier: TierPrimary, FileCount: 855, Share: 0.85},
			{Language: "python", Status: IndexerSkipped, Tier: TierIncidental, FileCount: 37, Share: 0.037},
		},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Manifest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Indexers) != 2 {
		t.Fatalf("Indexers len = %d, want 2", len(got.Indexers))
	}
	if got.Indexers[1].Status != IndexerSkipped || got.Indexers[1].Tier != TierIncidental {
		t.Errorf("second indexer = %+v, want skipped/incidental", got.Indexers[1])
	}
	if got.Indexers[1].FileCount != 37 {
		t.Errorf("FileCount = %d, want 37", got.Indexers[1].FileCount)
	}
}

func TestBuildResults_IncidentalIsSkippedNotInvoked(t *testing.T) {
	// The childscribe shape: PHP dominant with a marker, Python incidental
	// with none. The python entry must be recorded as skipped, and the
	// caller must never have been asked to run it.
	dets := []DetectedLanguage{
		{Language: "php", Tier: TierPrimary, FileCount: 855, Share: 0.855, Marker: "composer.json"},
		{Language: "python", Tier: TierIncidental, FileCount: 37, Share: 0.037},
	}
	var invoked []string
	results := buildResults(dets, func(language string) error {
		invoked = append(invoked, language)
		return nil
	})

	if len(invoked) != 1 || invoked[0] != "php" {
		t.Errorf("invoked = %v, want [php] only", invoked)
	}
	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	py := results[1]
	if py.Language != "python" || py.Status != IndexerSkipped {
		t.Errorf("python result = %+v, want skipped", py)
	}
	if py.Error != "" {
		t.Errorf("skipped entry should carry no error, got %q", py.Error)
	}
	if deriveStatus(results) != "ready" {
		t.Errorf("status = %q, want ready", deriveStatus(results))
	}
}

func TestBuildResults_JavascriptFoldsIntoTypescriptIndexer(t *testing.T) {
	// scip-typescript covers both languages. One invocation, one result,
	// counts summed, tier taken as the stronger of the two.
	dets := []DetectedLanguage{
		{Language: "javascript", Tier: TierPrimary, FileCount: 60, Share: 0.6, Marker: "package.json"},
		{Language: "typescript", Tier: TierIncidental, FileCount: 40, Share: 0.4},
	}
	var invoked []string
	results := buildResults(dets, func(language string) error {
		invoked = append(invoked, language)
		return nil
	})

	if len(invoked) != 1 || invoked[0] != "typescript" {
		t.Errorf("invoked = %v, want [typescript] once", invoked)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].FileCount != 100 {
		t.Errorf("FileCount = %d, want 100 (both languages summed)", results[0].FileCount)
	}
	if results[0].Tier != TierPrimary {
		t.Errorf("Tier = %q, want primary (stronger of the two)", results[0].Tier)
	}
}

func TestBuildResults_MissingPrimaryDegrades(t *testing.T) {
	dets := []DetectedLanguage{
		{Language: "go", Tier: TierPrimary, FileCount: 100, Share: 1.0, Marker: "go.mod"},
	}
	results := buildResults(dets, func(language string) error {
		return golangNotFound()
	})
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Status != IndexerMissing {
		t.Errorf("Status = %q, want missing", results[0].Status)
	}
	if results[0].Remedy == "" {
		t.Error("missing primary indexer must carry a remedy")
	}
	if deriveStatus(results) != "partial" {
		t.Errorf("status = %q, want partial", deriveStatus(results))
	}
}

// golangNotFound wraps the scip-go sentinel the way buildAtLayout does.
func golangNotFound() error {
	return fmt.Errorf("scip-go: %w", golang.ErrIndexerNotFound)
}
