package review

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func snapshotRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	snapshotGit(t, repo, "init", "-q")
	snapshotGit(t, repo, "config", "user.email", "test@example.com")
	snapshotGit(t, repo, "config", "user.name", "Test")
	return repo
}
func snapshotGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}
func snapshotWrite(t *testing.T, repo, path, content string) {
	t.Helper()
	p := filepath.Join(repo, path)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func mustCapture(t *testing.T, repo string, opts CaptureOptions) Snapshot {
	t.Helper()
	s, err := Capture(context.Background(), repo, opts)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestSnapshotIdentityAndChangeKinds(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "a.txt", "original\n")
	snapshotWrite(t, repo, "gone.txt", "delete me\n")
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	base := mustCapture(t, repo, CaptureOptions{})
	if again := mustCapture(t, repo, CaptureOptions{}); again.ID != base.ID {
		t.Fatal("unchanged capture changed identity")
	}
	snapshotWrite(t, repo, "a.txt", "staged\n")
	snapshotGit(t, repo, "add", "a.txt")
	staged := mustCapture(t, repo, CaptureOptions{})
	if staged.ID == base.ID {
		t.Fatal("staged content absent from identity")
	}
	snapshotWrite(t, repo, "a.txt", "unstaged\n")
	snapshotWrite(t, repo, "new.txt", "untracked\n")
	snapshotGit(t, repo, "rm", "-q", "gone.txt")
	s := mustCapture(t, repo, CaptureOptions{})
	if s.ID == staged.ID {
		t.Fatal("unstaged content absent from identity")
	}
	id, err := Fingerprint(context.Background(), repo)
	if err != nil || id != s.ID {
		t.Fatalf("fingerprint mismatch %s %v", id, err)
	}
	for _, p := range []string{"a.txt", "gone.txt", "new.txt"} {
		if !containsSnapshotPath(s.ChangedFiles, p) {
			t.Errorf("missing changed path %q: %v", p, s.ChangedFiles)
		}
	}
	found := false
	for _, f := range s.Files {
		if f.Path == "gone.txt" && f.Mode == "deleted" {
			found = true
		}
	}
	if !found {
		t.Fatal("staged deletion not captured")
	}
	content := ""
	for _, e := range s.Evidence {
		content += e.Content
	}
	for _, want := range []string{"original", "unstaged", "untracked", "delete me"} {
		if !strings.Contains(content, want) {
			t.Errorf("evidence missing %q", want)
		}
	}
	if err := os.Chmod(filepath.Join(repo, "a.txt"), 0755); err != nil {
		t.Fatal(err)
	}
	if mode := mustCapture(t, repo, CaptureOptions{}); mode.ID == s.ID {
		t.Fatal("mode absent from identity")
	}
}
func containsSnapshotPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}
func TestSnapshotExclusionsAndSymlinks(t *testing.T) {
	repo := snapshotRepo(t)
	for _, p := range []string{".env", "nested/.env.local", "credentials.json", "private.pem", "custom/private.txt", "safe.txt"} {
		snapshotWrite(t, repo, p, "SECRET-"+p+"\n")
	}
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	for _, p := range []string{".env", "nested/.env.local", "credentials.json", "private.pem", "custom/private.txt"} {
		snapshotWrite(t, repo, p, "NEW-SECRET-"+p+"\n")
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("SYMLINK-SECRET"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "link")); err != nil {
		t.Fatal(err)
	}
	s := mustCapture(t, repo, CaptureOptions{Exclude: []string{"custom/**"}})
	for _, e := range s.Evidence {
		if strings.Contains(e.Content, "SECRET") || strings.Contains(e.Content, "NEW-SECRET") {
			t.Errorf("secret evidence: %+v", e)
		}
		if e.Path != "link" {
			t.Errorf("excluded path in evidence: %s", e.Path)
		}
	}
	if len(s.Warnings) == 0 {
		t.Fatal("excluded evidence needs warning")
	}
}
func TestSnapshotBudgetAndUnbornHead(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "large.txt", strings.Repeat("abcdef\n", 1000))
	s := mustCapture(t, repo, CaptureOptions{MaxBytes: 100})
	n := 0
	for _, e := range s.Evidence {
		n += len(e.Content)
	}
	if n > 100 || n == 0 {
		t.Fatalf("evidence bytes=%d", n)
	}
	if len(s.Warnings) == 0 {
		t.Fatal("missing truncation warning")
	}
	if !containsSnapshotPath(s.ChangedFiles, "large.txt") {
		t.Fatal("unborn untracked absent")
	}
}
func TestSnapshotIgnoresIgnoredFilesAndHonorsCancellation(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, ".gitignore", "ignored\n")
	before := mustCapture(t, repo, CaptureOptions{})
	snapshotWrite(t, repo, "ignored", "ignore this")
	after := mustCapture(t, repo, CaptureOptions{})
	if before.ID != after.ID {
		t.Fatal("ignored content changed identity")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Capture(ctx, repo, CaptureOptions{}); err == nil {
		t.Fatal("canceled capture succeeded")
	}
}

func TestSnapshotNeverRunsRepositoryFilters(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "file.txt", "before\n")
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	snapshotWrite(t, repo, ".gitattributes", "*.txt filter=unsafe diff=unsafe\n")
	marker := filepath.Join(repo, "EXECUTED")
	snapshotGit(t, repo, "config", "filter.unsafe.clean", "touch "+marker)
	snapshotGit(t, repo, "config", "diff.unsafe.textconv", "touch "+marker)
	snapshotGit(t, repo, "config", "diff.external", "touch "+marker)
	snapshotWrite(t, repo, "file.txt", "after\n")
	s := mustCapture(t, repo, CaptureOptions{})
	if !containsSnapshotPath(s.ChangedFiles, "file.txt") {
		t.Fatal("missing file change")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("repository executable ran: %v", err)
	}
}
func TestSnapshotRejectsTrackedDirectorySymlink(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "nested/file", "original\n")
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	if err := os.RemoveAll(filepath.Join(repo, "nested")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	snapshotWrite(t, outside, "file", "secret\n")
	if err := os.Symlink(outside, filepath.Join(repo, "nested")); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(context.Background(), repo, CaptureOptions{}); err == nil {
		t.Fatal("followed tracked parent symlink")
	}
}
func TestSnapshotConfiguredRecursiveExclusion(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "a/b/c/private.txt", "SECRET\n")
	snapshotWrite(t, repo, "auth.json", "TOKEN\n")
	snapshotWrite(t, repo, "id_ecdsa", "PRIVATE KEY\n")
	s := mustCapture(t, repo, CaptureOptions{Exclude: []string{"**/private.txt"}})
	if len(s.Evidence) != 0 {
		t.Fatalf("secret paths leaked: %+v", s.Evidence)
	}
}
func TestSnapshotHeadAndRenameChangeIdentity(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "old.txt", "same content\n")
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	before := mustCapture(t, repo, CaptureOptions{})
	snapshotGit(t, repo, "commit", "--allow-empty", "-qm", "next")
	changedHead := mustCapture(t, repo, CaptureOptions{})
	if changedHead.ID == before.ID {
		t.Fatal("HEAD absent from fingerprint")
	}
	snapshotGit(t, repo, "mv", "old.txt", "new.txt")
	renamed := mustCapture(t, repo, CaptureOptions{})
	if renamed.ID == changedHead.ID {
		t.Fatal("rename absent from fingerprint")
	}
	if !containsSnapshotPath(renamed.ChangedFiles, "old.txt") || !containsSnapshotPath(renamed.ChangedFiles, "new.txt") {
		t.Fatalf("rename missing: %v", renamed.ChangedFiles)
	}
}

func TestReadEvidenceFileBoundaries(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "safe.txt", "abcdefghij")
	snapshotWrite(t, repo, ".env", "SECRET")
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("OUTSIDE SECRET"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "link")); err != nil {
		t.Fatal(err)
	}
	got, err := ReadEvidenceFile(context.Background(), repo, "safe.txt", 10)
	if err != nil || got != "abcdefghij" {
		t.Fatalf("bounded content %q %v", got, err)
	}
	if _, err := ReadEvidenceFile(context.Background(), repo, ".env", 100); err == nil {
		t.Fatal("secret read allowed")
	}
	if _, err := ReadEvidenceFile(context.Background(), repo, "../escape", 100); err == nil {
		t.Fatal("path traversal allowed")
	}
	got, err = ReadEvidenceFile(context.Background(), repo, "link", 4096)
	if err != nil || got != outside {
		t.Fatalf("symlink target %q %v", got, err)
	}
	if !PathExcluded("nested/.env.local", nil) || !PathExcluded("a/b/private.txt", []string{"**/private.txt"}) {
		t.Fatal("public exclusions missing")
	}
}

func TestSnapshotPermissionChangeAffectsIdentity(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "file", "data\n")
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	before := mustCapture(t, repo, CaptureOptions{})
	if err := os.Chmod(filepath.Join(repo, "file"), 0755); err != nil {
		t.Fatal(err)
	}
	after := mustCapture(t, repo, CaptureOptions{})
	if before.ID == after.ID {
		t.Fatal("executable permission change absent from snapshot")
	}
	if !containsSnapshotPath(after.ChangedFiles, "file") {
		t.Fatal("executable permission change absent from Git changes")
	}
}

func TestSnapshotRejectsConcurrentWrites(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "file", strings.Repeat("a", 16<<20))
	file, err := os.OpenFile(filepath.Join(repo, "file"), os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stop, ready, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		n := byte(0)
		first := true
		for {
			select {
			case <-stop:
				return
			default:
			}
			n++
			_, _ = file.WriteAt([]byte{n}, 0)
			if first {
				close(ready)
				first = false
			}
		}
	}()
	<-ready
	_, err = Capture(context.Background(), repo, CaptureOptions{})
	close(stop)
	<-done
	if err == nil {
		t.Fatal("changing file was accepted as coherent")
	}
}

func TestSnapshotCapturesIndexWhenWorkingTreeMatchesHead(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "file.txt", "HEAD contents\n")
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	base := mustCapture(t, repo, CaptureOptions{})
	snapshotWrite(t, repo, "file.txt", "staged-only contents\n")
	snapshotGit(t, repo, "add", "file.txt")
	snapshotWrite(t, repo, "file.txt", "HEAD contents\n")
	s := mustCapture(t, repo, CaptureOptions{})
	if s.ID == base.ID {
		t.Fatal("index change absent from identity")
	}
	if !containsSnapshotPath(s.ChangedFiles, "file.txt") {
		t.Fatal("staged change absent")
	}
	found := false
	for _, e := range s.Evidence {
		if e.Kind == "staged_diff" && strings.Contains(e.Content, "staged-only contents") {
			found = true
		}
	}
	if !found {
		t.Fatalf("staged evidence absent: %+v", s.Evidence)
	}
	snapshotGit(t, repo, "rm", "--cached", "-f", "file.txt")
	deleted := mustCapture(t, repo, CaptureOptions{})
	if !containsSnapshotPath(deleted.ChangedFiles, "file.txt") {
		t.Fatal("staged deletion with preserved working file absent")
	}
}

func TestSnapshotCleanPrivatePermissionsAreNotGitChanges(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "source.txt", "committed content\n")
	if err := os.Chmod(filepath.Join(repo, "source.txt"), 0600); err != nil {
		t.Fatal(err)
	}
	snapshotGit(t, repo, "add", ".")
	snapshotGit(t, repo, "commit", "-qm", "initial")
	before := mustCapture(t, repo, CaptureOptions{})
	if len(before.ChangedFiles) != 0 || len(before.Evidence) != 0 {
		t.Fatalf("clean 0600 file reported as Git change: %+v", before)
	}
	if err := os.Chmod(filepath.Join(repo, "source.txt"), 0644); err != nil {
		t.Fatal(err)
	}
	after := mustCapture(t, repo, CaptureOptions{})
	if len(after.ChangedFiles) != 0 || len(after.Evidence) != 0 {
		t.Fatalf("nonexecutable permission change reported as Git change: %+v", after)
	}
	if before.ID == after.ID {
		t.Fatal("raw file permissions should remain in snapshot fingerprint")
	}
}

func TestReadEvidenceFileMakesTruncationVisible(t *testing.T) {
	repo := snapshotRepo(t)
	snapshotWrite(t, repo, "large.txt", strings.Repeat("content\n", 100))
	text, err := ReadEvidenceFile(context.Background(), repo, "large.txt", 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > 40 || !strings.HasSuffix(text, "[source truncated]") {
		t.Fatalf("missing bounded truncation marker: %q", text)
	}
	if _, err := ReadEvidenceFile(context.Background(), repo, "large.txt", 4); err == nil {
		t.Fatal("tiny budget silently returned truncated content")
	}
}
