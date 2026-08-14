package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
)

func TestCheckCurrentRepo_StaleWhenHeadMoved(t *testing.T) {
	repo, head := gitRepoWithCommit(t)
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:   repo,
		Languages:  []string{"go"},
		IndexedAt:  time.Now().Add(-72 * time.Hour),
		Status:     index.StatusReady,
		HeadCommit: "0123456789abcdef0123456789abcdef01234567",
	})

	got := checkCurrentRepo(home, repo)
	if got.Status != StatusWarn {
		t.Errorf("Status = %v, want Warn", got.Status)
	}
	if !strings.Contains(got.Detail, index.StatusStale) {
		t.Errorf("Detail = %q, want it to report the repo as stale", got.Detail)
	}
	if !strings.Contains(got.Detail, head[:8]) || !strings.Contains(got.Detail, "01234567") {
		t.Errorf("Detail = %q, want both the indexed and the live commit", got.Detail)
	}
	if !strings.Contains(got.Detail, "ago") {
		t.Errorf("Detail = %q, want the index age", got.Detail)
	}
	if got.Remedy == "" {
		t.Error("Remedy = empty, want the reindex command")
	}
}

func TestCheckCurrentRepo_NotStaleWhenHeadUnchanged(t *testing.T) {
	repo, head := gitRepoWithCommit(t)
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:   repo,
		Languages:  []string{"go"},
		IndexedAt:  time.Now().Add(-72 * time.Hour),
		Status:     index.StatusReady,
		HeadCommit: head,
	})

	got := checkCurrentRepo(home, repo)
	if got.Status != StatusPass {
		t.Errorf("Status = %v, want Pass", got.Status)
	}
	// An index built three days ago at the commit that is still checked out
	// is current, not stale — age alone must never trip the signal.
	if strings.Contains(got.Detail, index.StatusStale) {
		t.Errorf("Detail = %q, want no stale marker", got.Detail)
	}
	if got.Remedy != "" {
		t.Errorf("Remedy = %q, want none", got.Remedy)
	}
}

func TestCheckCurrentRepo_StaleFallsBackToMTimeWithoutGit(t *testing.T) {
	repo := t.TempDir()
	indexedAt := time.Now().Add(-48 * time.Hour)
	// A source file edited after the index was built, in a directory that is
	// not a git checkout: the mtime comparison is the only signal available.
	touch(t, filepath.Join(repo, "main.go"), time.Now().Add(-1*time.Hour))
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:  repo,
		Languages: []string{"go"},
		IndexedAt: indexedAt,
		Status:    index.StatusReady,
	})

	got := checkCurrentRepo(home, repo)
	if got.Status != StatusWarn {
		t.Errorf("Status = %v, want Warn", got.Status)
	}
	if !strings.Contains(got.Detail, "source files edited") {
		t.Errorf("Detail = %q, want the mtime-based explanation", got.Detail)
	}
}

func TestCheckCurrentRepo_NotStaleWhenSourceUntouched(t *testing.T) {
	repo := t.TempDir()
	touch(t, filepath.Join(repo, "main.go"), time.Now().Add(-96*time.Hour))
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:  repo,
		Languages: []string{"go"},
		IndexedAt: time.Now().Add(-48 * time.Hour),
		Status:    index.StatusReady,
	})

	if got := checkCurrentRepo(home, repo); got.Status != StatusPass {
		t.Errorf("Status = %v (%s), want Pass", got.Status, got.Detail)
	}
}

func TestCheckCurrentRepo_PartialOutranksStale(t *testing.T) {
	repo, _ := gitRepoWithCommit(t)
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:   repo,
		Languages:  []string{"go", "python"},
		IndexedAt:  time.Now().Add(-72 * time.Hour),
		Status:     index.StatusPartial,
		HeadCommit: "0123456789abcdef0123456789abcdef01234567",
		Indexers: []index.IndexerResult{
			{Language: "go", Tier: index.TierPrimary, Status: index.IndexerOK},
			{
				Language: "python", Tier: index.TierPrimary, Status: index.IndexerMissing,
				Error:  "scip-python not found on PATH",
				Remedy: "npm i -g @sourcegraph/scip-python",
			},
		},
	})

	got := checkCurrentRepo(home, repo)
	// The label is the highest-precedence signal...
	if !strings.HasPrefix(got.Detail, index.StatusPartial+" ") {
		t.Errorf("Detail = %q, want it to lead with the partial status", got.Detail)
	}
	// ...but staleness is still reported, and the missing indexer keeps the
	// remedy slot because it is the more urgent fix.
	if !strings.Contains(got.Detail, index.StatusStale) {
		t.Errorf("Detail = %q, want the stale signal reported alongside partial", got.Detail)
	}
	if got.Remedy != "npm i -g @sourcegraph/scip-python" {
		t.Errorf("Remedy = %q, want the missing-indexer remedy", got.Remedy)
	}
}

// gitRepoWithCommit creates a git repo with one commit and returns its path
// and HEAD sha. Skips the test when git is unavailable.
func gitRepoWithCommit(t *testing.T) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		gitRun(t, repo, args...)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	gitRun(t, repo, "add", "main.go")
	gitRun(t, repo, "commit", "-m", "init")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return repo, strings.TrimSpace(string(out))
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func touch(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}
