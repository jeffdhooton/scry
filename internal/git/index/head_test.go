package index

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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

// A HEAD we ran out of time to ask for is NOT the same answer as a repo that
// has no HEAD. Both yield an empty string, so the error is the only thing
// telling them apart, and callers key their mtime fallback off it: guessing
// from timestamps because git was slow reports a repo sitting exactly at its
// indexed commit as stale.
func TestHeadCommitExpiredBudgetIsUnknownNotAbsent(t *testing.T) {
	repo := initRepo(t)
	commit(t, repo, "one.txt")

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)

	head, err := HeadCommit(ctx, repo)
	if head != "" {
		t.Fatalf("HeadCommit() = %q, want empty once the budget is gone", head)
	}
	if err == nil {
		t.Fatal("HeadCommit() error = nil, want an error — a nil error means " +
			"\"this repo has no HEAD\", which sends callers to the mtime fallback")
	}
	if !HeadUnknown(err) {
		t.Errorf("HeadUnknown(%v) = false, want true so callers skip the mtime guess", err)
	}
}

// The dangerous variant of the above: the budget expiring while git is still
// running. Cancelling kills the process, and a killed process reports an
// ExitError — the same error class as "not a repository". Without an explicit
// context check that is laundered into ("", nil), which tells callers this
// repo simply has no HEAD and sends them to the mtime fallback.
//
// A deliberately slow fake git on PATH makes the timing deterministic: the
// call cannot possibly finish inside the budget, so cancellation always lands
// mid-flight rather than before the process starts.
func TestHeadCommitCancelledMidFlightIsUnknownNotAbsent(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\nsleep 30\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	t.Setenv("PATH", bin)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	head, err := HeadCommit(ctx, t.TempDir())
	if head != "" {
		t.Fatalf("HeadCommit() = %q, want empty when the call was cut short", head)
	}
	if err == nil {
		t.Fatal("HeadCommit() error = nil for a git killed mid-call; a nil error " +
			"means \"no HEAD here\" and licenses the mtime fallback")
	}
	if !HeadUnknown(err) {
		t.Errorf("HeadUnknown(%v) = false, want true — the process was killed by "+
			"our own timeout, which is not evidence about the repo", err)
	}
}

// The other way a HEAD goes missing: no git binary at all. That one IS the
// task's "missing git" fallback — mtimes are the only signal left, so callers
// must still use them.
func TestHeadCommitWithoutGitBinaryIsNotUnknown(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("PATH", "")

	head, err := HeadCommit(context.Background(), repo)
	if head != "" {
		t.Fatalf("HeadCommit() = %q, want empty with no git binary", head)
	}
	if err == nil {
		t.Fatal("HeadCommit() error = nil, want the lookup failure reported")
	}
	if HeadUnknown(err) {
		t.Errorf("HeadUnknown(%v) = true, want false — with no git at all, "+
			"mtimes are the only staleness signal and callers must fall back to them", err)
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
