// Package index orchestrates git indexing passes for one repo:
// blame every file, parse recent commits, compute co-change and churn stats.
package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gitstore "github.com/jeffdhooton/scry/internal/git/store"
)

// Manifest is the per-repo metadata file written alongside the git index.
type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	RepoPath      string    `json:"repo_path"`
	IndexedAt     time.Time `json:"indexed_at"`
	Status        string    `json:"status"`
	LastCommit    string    `json:"last_commit,omitempty"`
	Depth         int       `json:"depth"`
	Stats         Stats     `json:"stats"`
}

type Stats struct {
	Files      int   `json:"files"`
	Commits    int   `json:"commits"`
	BlameLines int   `json:"blame_lines"`
	ElapsedMs  int64 `json:"elapsed_ms"`
}

// RepoLayout is the resolved on-disk layout for the git index within a repo.
type RepoLayout struct {
	RepoPath     string
	StorageDir   string
	BadgerDir    string
	ManifestPath string
}

const DefaultDepth = 500

// Layout resolves where the git index for repoPath should live under scryHome.
// Git data lives in a separate "git" subdirectory from scry's code index.
func Layout(scryHome, repoPath string) RepoLayout {
	hash := sha256.Sum256([]byte(repoPath))
	short := hex.EncodeToString(hash[:])[:16]
	storage := filepath.Join(scryHome, "repos", short, "git")
	return RepoLayout{
		RepoPath:     repoPath,
		StorageDir:   storage,
		BadgerDir:    filepath.Join(storage, "index.db"),
		ManifestPath: filepath.Join(storage, "manifest.json"),
	}
}

func LoadManifest(layout RepoLayout) (*Manifest, error) {
	b, err := os.ReadFile(layout.ManifestPath)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func WriteManifest(layout RepoLayout, m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(layout.ManifestPath, b, 0o644)
}

// Build indexes git data for a repo: blame every file, parse recent commits,
// compute co-change and churn stats.
func Build(ctx context.Context, scryHome, repoPath string, depth int) (*Manifest, error) {
	if depth <= 0 {
		depth = DefaultDepth
	}

	layout := Layout(scryHome, repoPath)
	if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	st, err := gitstore.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	ver, err := st.SchemaVersionOnDisk()
	if err != nil {
		return nil, err
	}
	if ver != 0 && ver != gitstore.SchemaVersion {
		if err := st.Reset(); err != nil {
			return nil, fmt.Errorf("reset store: %w", err)
		}
	}
	if err := st.SetMeta("schema_version", gitstore.SchemaVersion); err != nil {
		return nil, err
	}

	start := time.Now()

	blameLines, err := indexBlame(ctx, repoPath, st)
	if err != nil {
		return nil, fmt.Errorf("index blame: %w", err)
	}

	commitCount, lastCommit, err := indexLog(ctx, repoPath, depth, st)
	if err != nil {
		return nil, fmt.Errorf("index log: %w", err)
	}

	if err := computeCochange(st); err != nil {
		return nil, fmt.Errorf("compute cochange: %w", err)
	}

	if err := computeChurnAndContrib(st); err != nil {
		return nil, fmt.Errorf("compute churn/contrib: %w", err)
	}

	elapsed := time.Since(start)

	fileCount, _ := st.CountKeys(gitstore.PrefixChurn())

	if err := st.SetMeta("indexed_at", time.Now()); err != nil {
		return nil, err
	}
	if err := st.SetMeta("last_commit", lastCommit); err != nil {
		return nil, err
	}

	manifest := &Manifest{
		SchemaVersion: gitstore.SchemaVersion,
		RepoPath:      repoPath,
		IndexedAt:     time.Now(),
		Status:        "ready",
		LastCommit:    lastCommit,
		Depth:         depth,
		Stats: Stats{
			Files:      fileCount,
			Commits:    commitCount,
			BlameLines: blameLines,
			ElapsedMs:  elapsed.Milliseconds(),
		},
	}

	if err := WriteManifest(layout, manifest); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return manifest, nil
}

// HasGit returns true if the path has a .git directory.
func HasGit(repoPath string) bool {
	info, err := os.Stat(filepath.Join(repoPath, ".git"))
	return err == nil && info.IsDir()
}
