package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSweepStaleIndexTrash: SwapNext archives the live index to
// index.db.old.<pid>.<ns> and callers delete it in a fire-and-forget
// goroutine. Those goroutines die with the daemon — 31k orphans / 271 GB
// accumulated that way. Daemon start must sweep any index.db.old.* and stale
// index.db.next / manifest.json.next while leaving the live index alone.
func TestSweepStaleIndexTrash(t *testing.T) {
	home := t.TempDir()
	repoDir := filepath.Join(home, "repos", "abcd1234")

	mk := func(parts ...string) string {
		p := filepath.Join(parts...)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	touch := func(parts ...string) string {
		p := filepath.Join(parts...)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	live := mk(repoDir, "index.db")
	liveFile := touch(live, "000001.sst")
	manifest := touch(repoDir, "manifest.json")

	trash1 := mk(repoDir, "index.db.old.123.456")
	touch(trash1, "000002.sst")
	trash2 := mk(repoDir, "index.db.old.999.111")
	next := mk(repoDir, "index.db.next")
	touch(next, "000003.sst")
	nextManifest := touch(repoDir, "manifest.json.next")

	sweepStaleIndexTrash(home)

	for _, gone := range []string{trash1, trash2, next, nextManifest} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("%s should have been swept, stat err=%v", gone, err)
		}
	}
	for _, kept := range []string{live, liveFile, manifest} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("%s must survive the sweep: %v", kept, err)
		}
	}
}
