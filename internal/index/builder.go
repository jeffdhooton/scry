// Package index orchestrates a full indexing pass for one repo:
//
//  1. resolve the repo's per-user storage directory under ~/.scry/repos/<hash>
//  2. detect languages present in the repo
//  3. run the appropriate language indexer (P0: TypeScript only)
//  4. parse the resulting .scip file into the BadgerDB store
//  5. write the manifest
//
// The result is a "warm" repo: queries against the store return correct
// results immediately.
package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/scip"
	"github.com/jeffdhooton/scry/internal/sources/typescript"
	"github.com/jeffdhooton/scry/internal/store"
)

// Manifest is the per-repo metadata file written alongside the BadgerDB index.
type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	RepoPath      string    `json:"repo_path"`
	Languages     []string  `json:"languages"`
	IndexedAt     time.Time `json:"indexed_at"`
	Status        string    `json:"status"` // "ready" | "partial" | "broken"
	FailedFiles   int       `json:"failed_files,omitempty"`
	Stats         scip.Stats `json:"stats"`
}

// RepoLayout is the resolved on-disk layout for one repo.
type RepoLayout struct {
	RepoPath     string // absolute path to the source repo
	StorageDir   string // ~/.scry/repos/<hash>
	BadgerDir    string // <StorageDir>/index.db
	ManifestPath string // <StorageDir>/manifest.json
}

// scipPath returns the per-language .scip dump location.
func (l RepoLayout) scipPath(language string) string {
	return filepath.Join(l.StorageDir, "scip-"+language+".bin")
}

// Layout resolves where the index for repoPath should live under scryHome.
// repoPath must be absolute.
func Layout(scryHome, repoPath string) RepoLayout {
	hash := sha256.Sum256([]byte(repoPath))
	short := hex.EncodeToString(hash[:])[:16]
	storage := filepath.Join(scryHome, "repos", short)
	return RepoLayout{
		RepoPath:     repoPath,
		StorageDir:   storage,
		BadgerDir:    filepath.Join(storage, "index.db"),
		ManifestPath: filepath.Join(storage, "manifest.json"),
	}
}

// Build runs a full index pass for the repo at repoPath.
//
// Behavior:
//   - if the storage directory exists with an outdated schema_version, the
//     BadgerDB is wiped and rebuilt
//   - languages are detected by file extension; runs every supported indexer
//     present (TypeScript, Go in P1)
//   - per-language .scip dumps are kept on disk so future incremental rebuilds
//     don't have to re-parse the world
//   - status is "ready" if every detected indexer succeeded, "partial" if at
//     least one ran but others failed
func Build(ctx context.Context, scryHome, repoPath string) (*Manifest, error) {
	if !filepath.IsAbs(repoPath) {
		abs, err := filepath.Abs(repoPath)
		if err != nil {
			return nil, fmt.Errorf("resolve repo path: %w", err)
		}
		repoPath = abs
	}

	layout := Layout(scryHome, repoPath)
	if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	languages, err := detectLanguages(repoPath)
	if err != nil {
		return nil, fmt.Errorf("detect languages: %w", err)
	}
	if len(languages) == 0 {
		return nil, errors.New("no supported languages detected in repo")
	}

	// Run every applicable indexer. Each one writes its own scip-<lang>.bin.
	// We collect (language, scipPath) pairs and parse them sequentially after
	// all indexers finish — keeps the BadgerDB write batch contiguous.
	type indexed struct {
		language string
		scipPath string
	}
	var produced []indexed
	var indexerErrs []error

	if contains(languages, "typescript") || contains(languages, "javascript") {
		out := layout.scipPath("typescript")
		if _, err := typescript.Index(ctx, repoPath, out); err != nil {
			indexerErrs = append(indexerErrs, fmt.Errorf("scip-typescript: %w", err))
		} else {
			produced = append(produced, indexed{"typescript", out})
		}
	}
	if contains(languages, "go") {
		out := layout.scipPath("go")
		binDir := filepath.Join(scryHome, "bin")
		if _, err := golang.Index(ctx, binDir, repoPath, out); err != nil {
			indexerErrs = append(indexerErrs, fmt.Errorf("scip-go: %w", err))
		} else {
			produced = append(produced, indexed{"go", out})
		}
	}

	if len(produced) == 0 {
		// Every indexer failed. Surface the first error verbatim.
		if len(indexerErrs) > 0 {
			return nil, indexerErrs[0]
		}
		return nil, fmt.Errorf("no supported indexer ran on repo languages %v", languages)
	}

	// Open store, wipe stale data, parse each .scip into the same BadgerDB.
	st, err := store.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	disk, err := st.SchemaVersionOnDisk()
	if err != nil {
		return nil, fmt.Errorf("read schema version: %w", err)
	}
	if disk != 0 && disk != store.SchemaVersion {
		// Loud reindex per docs/DECISIONS.md "Schema evolution".
		fmt.Fprintf(os.Stderr, "scry: schema upgrade %d -> %d, reindexing %s\n", disk, store.SchemaVersion, repoPath)
	}
	// Always reset before re-ingesting so we don't accumulate stale records
	// from a previous build.
	if err := st.Reset(); err != nil {
		return nil, fmt.Errorf("reset store: %w", err)
	}

	if err := st.SetMeta("schema_version", store.SchemaVersion); err != nil {
		return nil, fmt.Errorf("write schema version: %w", err)
	}
	if err := st.SetMeta("repo_path", repoPath); err != nil {
		return nil, fmt.Errorf("write repo path: %w", err)
	}

	combined := scip.Stats{}
	for _, p := range produced {
		stats, err := scip.Parse(ctx, p.scipPath, st)
		if err != nil {
			return nil, fmt.Errorf("parse %s scip: %w", p.language, err)
		}
		combined.Documents += stats.Documents
		combined.Symbols += stats.Symbols
		combined.Definitions += stats.Definitions
		combined.References += stats.References
		combined.CallEdges += stats.CallEdges
		combined.Implementations += stats.Implementations
	}

	status := "ready"
	if len(indexerErrs) > 0 {
		status = "partial"
		fmt.Fprintf(os.Stderr, "scry: %d indexer(s) failed; status=partial\n", len(indexerErrs))
		for _, e := range indexerErrs {
			fmt.Fprintf(os.Stderr, "scry:   %v\n", e)
		}
	}

	manifest := &Manifest{
		SchemaVersion: store.SchemaVersion,
		RepoPath:      repoPath,
		Languages:     languages,
		IndexedAt:     time.Now().UTC(),
		Status:        status,
		Stats:         combined,
	}
	if err := writeManifest(layout.ManifestPath, manifest); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}
	return manifest, nil
}

// LoadManifest reads an existing manifest, or returns an error if missing.
func LoadManifest(layout RepoLayout) (*Manifest, error) {
	b, err := os.ReadFile(layout.ManifestPath)
	if err != nil {
		return nil, err
	}
	m := &Manifest{}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return m, nil
}

func writeManifest(path string, m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// detectLanguages walks the top of the repo, counts files by extension, and
// returns the language names for any extension above a 1% threshold. P0
// recognizes only TypeScript / JavaScript; other languages are reported but
// the builder ignores them.
func detectLanguages(repoPath string) ([]string, error) {
	counts := map[string]int{}
	var total int
	skip := map[string]bool{
		"node_modules": true,
		".git":         true,
		"dist":         true,
		"build":        true,
		"out":          true,
		"vendor":       true,
		"target":       true,
		".next":        true,
		".turbo":       true,
		"coverage":     true,
	}
	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort
		}
		if d.IsDir() {
			if skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
			counts[langForExt(ext)]++
			total++
		case ".go":
			counts["go"]++
			total++
		case ".php":
			counts["php"]++
			total++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var langs []string
	for lang, c := range counts {
		if total == 0 || float64(c)/float64(total) >= 0.01 {
			langs = append(langs, lang)
		}
	}
	return langs, nil
}

func langForExt(ext string) string {
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	}
	return ""
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
