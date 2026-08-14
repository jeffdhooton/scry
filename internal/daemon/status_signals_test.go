package daemon

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
)

// staticHeads builds a head cache pre-seeded with an answer, so the status
// derivation can be exercised without a real repo on disk.
func staticHeads(repoPath, head string) *headCache {
	h := newHeadCache(context.Background())
	h.entries[repoPath] = head
	return h
}

func TestRepoStatusEntryStaleWhenHeadMoved(t *testing.T) {
	m := &index.Manifest{
		RepoPath:   "/repo",
		Status:     index.StatusReady,
		IndexedAt:  time.Now().Add(-time.Hour),
		HeadCommit: "aaaa1111",
	}
	got := repoStatusEntry(m, staticHeads("/repo", "bbbb2222"))
	if !got.Stale {
		t.Error("Stale = false, want true after HEAD moved")
	}
	if got.EffectiveStatus != index.StatusStale {
		t.Errorf("EffectiveStatus = %q, want %q", got.EffectiveStatus, index.StatusStale)
	}
	// The manifest's own status is reported untouched: "stale" is derived at
	// report time and never written back.
	if got.Status != index.StatusReady {
		t.Errorf("Status = %q, want the manifest's recorded %q", got.Status, index.StatusReady)
	}
}

func TestRepoStatusEntryNotStaleWhenHeadUnchanged(t *testing.T) {
	m := &index.Manifest{
		RepoPath:   "/repo",
		Status:     index.StatusReady,
		IndexedAt:  time.Now().Add(-720 * time.Hour),
		HeadCommit: "aaaa1111",
	}
	got := repoStatusEntry(m, staticHeads("/repo", "aaaa1111"))
	if got.Stale {
		t.Error("Stale = true, want false while HEAD is unchanged")
	}
	if got.EffectiveStatus != index.StatusReady {
		t.Errorf("EffectiveStatus = %q, want %q", got.EffectiveStatus, index.StatusReady)
	}
}

func TestRepoStatusEntryPartialOutranksStale(t *testing.T) {
	m := &index.Manifest{
		RepoPath:   "/repo",
		Status:     index.StatusPartial,
		IndexedAt:  time.Now().Add(-time.Hour),
		HeadCommit: "aaaa1111",
	}
	got := repoStatusEntry(m, staticHeads("/repo", "bbbb2222"))
	if !got.Stale {
		t.Error("Stale = false, want the stale fact still reported")
	}
	if got.EffectiveStatus != index.StatusPartial {
		t.Errorf("EffectiveStatus = %q, want %q to outrank stale", got.EffectiveStatus, index.StatusPartial)
	}
}

func TestRepoStatusEntryFallsBackToMTimeWithoutGit(t *testing.T) {
	repo := t.TempDir()
	src := filepath.Join(repo, "main.go")
	if err := os.WriteFile(src, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	recent := time.Now().Add(-time.Hour)
	if err := os.Chtimes(src, recent, recent); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	m := &index.Manifest{
		RepoPath:  repo,
		Status:    index.StatusReady,
		IndexedAt: time.Now().Add(-48 * time.Hour),
	}
	// An empty cache: the repo is not a git checkout, so head resolution
	// returns "" and the mtime comparison decides.
	got := repoStatusEntry(m, newHeadCache(context.Background()))
	if !got.Stale {
		t.Error("Stale = false, want true when a source file is newer than the index")
	}
	if got.EffectiveStatus != index.StatusStale {
		t.Errorf("EffectiveStatus = %q, want %q", got.EffectiveStatus, index.StatusStale)
	}
}

func TestHeadCacheResolvesEachRepoOnce(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	for _, args := range [][]string{{"add", "main.go"}, {"commit", "-m", "init"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	heads := newHeadCache(context.Background())
	first := heads.head(repo)
	if len(first) != 40 {
		t.Fatalf("head() = %q, want a 40-char sha", first)
	}

	// Destroy the repo. A second git call would now resolve to "" — the same
	// answer coming back proves the status call paid for exactly one.
	if err := os.RemoveAll(filepath.Join(repo, ".git")); err != nil {
		t.Fatalf("rm .git: %v", err)
	}
	if second := heads.head(repo); second != first {
		t.Errorf("head() = %q on the second call, want the cached %q", second, first)
	}
}

func TestHeadCacheEmptyForNonGitPath(t *testing.T) {
	heads := newHeadCache(context.Background())
	if got := heads.head(t.TempDir()); got != "" {
		t.Errorf("head() = %q, want empty for a non-git path", got)
	}
}

func TestRepoStatusEntryJSONFieldNames(t *testing.T) {
	// The room contract names these keys; consumers (scry status, doctor,
	// MCP) read them by name, so a rename is a breaking change.
	m := &index.Manifest{RepoPath: "/repo", Status: index.StatusReady, IndexedAt: time.Now()}
	entry := repoStatusEntry(m, staticHeads("/repo", ""))
	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"stale"`, `"effective_status"`} {
		if !strings.Contains(string(b), key) {
			t.Errorf("status entry JSON = %s, want key %s", b, key)
		}
	}
}
