package index

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHeadCommitNotAGitRepo(t *testing.T) {
	head, err := HeadCommit(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("HeadCommit() error = %v, want nil for a non-git path", err)
	}
	if head != "" {
		t.Fatalf("HeadCommit() = %q, want empty for a non-git path", head)
	}
}

func TestHeadCommitMissingPath(t *testing.T) {
	head, err := HeadCommit(context.Background(), filepath.Join(t.TempDir(), "gone"))
	if err != nil {
		t.Fatalf("HeadCommit() error = %v, want nil for a missing path", err)
	}
	if head != "" {
		t.Fatalf("HeadCommit() = %q, want empty for a missing path", head)
	}
}

func TestHeadCommitUnbornHead(t *testing.T) {
	repo := initRepo(t)
	head, err := HeadCommit(context.Background(), repo)
	if err != nil {
		t.Fatalf("HeadCommit() error = %v, want nil for a repo with no commits", err)
	}
	if head != "" {
		t.Fatalf("HeadCommit() = %q, want empty for a repo with no commits", head)
	}
}

func TestHeadCommitTracksHEAD(t *testing.T) {
	repo := initRepo(t)
	commit(t, repo, "one.txt")

	first, err := HeadCommit(context.Background(), repo)
	if err != nil {
		t.Fatalf("HeadCommit() error = %v", err)
	}
	if len(first) != 40 {
		t.Fatalf("HeadCommit() = %q, want a 40-char sha", first)
	}

	commit(t, repo, "two.txt")
	second, err := HeadCommit(context.Background(), repo)
	if err != nil {
		t.Fatalf("HeadCommit() error = %v", err)
	}
	if second == first {
		t.Fatalf("HeadCommit() = %q after a new commit, want a different sha", second)
	}
}

// initRepo creates an empty git repo, skipping the test if git is unavailable.
func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	run(t, repo, "init")
	run(t, repo, "config", "user.email", "test@example.com")
	run(t, repo, "config", "user.name", "test")
	return repo
}

func commit(t *testing.T, repo, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(name), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	run(t, repo, "add", name)
	run(t, repo, "commit", "-m", "add "+name)
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
