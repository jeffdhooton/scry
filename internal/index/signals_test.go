package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var indexedAt = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func TestIsStale(t *testing.T) {
	before := indexedAt.Add(-time.Hour)
	after := indexedAt.Add(time.Hour)

	tests := []struct {
		name       string
		manifest   *Manifest
		head       string
		newestFile time.Time
		want       bool
	}{
		{
			name:     "head moved since index",
			manifest: &Manifest{IndexedAt: indexedAt, HeadCommit: "aaaa1111"},
			head:     "bbbb2222",
			// A file mtime older than the index must not rescue a moved HEAD:
			// the exact comparison wins whenever both sides have a commit.
			newestFile: before,
			want:       true,
		},
		{
			name:       "head unchanged since index",
			manifest:   &Manifest{IndexedAt: indexedAt, HeadCommit: "aaaa1111"},
			head:       "aaaa1111",
			newestFile: after,
			want:       false,
		},
		{
			name:       "no git, source newer than index",
			manifest:   &Manifest{IndexedAt: indexedAt},
			head:       "",
			newestFile: after,
			want:       true,
		},
		{
			name:       "no git, source older than index",
			manifest:   &Manifest{IndexedAt: indexedAt},
			head:       "",
			newestFile: before,
			want:       false,
		},
		{
			name:       "no git and no file mtime available",
			manifest:   &Manifest{IndexedAt: indexedAt},
			head:       "",
			newestFile: time.Time{},
			want:       false,
		},
		{
			name:       "legacy manifest with no recorded head falls back to mtime",
			manifest:   &Manifest{IndexedAt: indexedAt},
			head:       "bbbb2222",
			newestFile: after,
			want:       true,
		},
		{
			name:       "legacy manifest with no recorded head and untouched files",
			manifest:   &Manifest{IndexedAt: indexedAt},
			head:       "bbbb2222",
			newestFile: before,
			want:       false,
		},
		{
			name:       "repo left git behind: recorded head, none live now",
			manifest:   &Manifest{IndexedAt: indexedAt, HeadCommit: "aaaa1111"},
			head:       "",
			newestFile: after,
			want:       true,
		},
		{
			name:       "nil manifest is never stale",
			manifest:   nil,
			head:       "bbbb2222",
			newestFile: after,
			want:       false,
		},
		{
			name:       "unknown index time cannot be compared",
			manifest:   &Manifest{},
			head:       "",
			newestFile: after,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStale(tt.manifest, tt.head, tt.newestFile); got != tt.want {
				t.Fatalf("IsStale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		stale  bool
		empty  []string
		want   string
	}{
		{name: "clean index", status: StatusReady, want: StatusReady},
		{name: "stale only", status: StatusReady, stale: true, want: StatusStale},
		{name: "empty only", status: StatusReady, empty: []string{"typescript"}, want: StatusEmpty},
		{
			name:   "empty outranks stale",
			status: StatusReady,
			stale:  true,
			empty:  []string{"typescript"},
			want:   StatusEmpty,
		},
		{
			name:   "partial outranks stale",
			status: StatusPartial,
			stale:  true,
			want:   StatusPartial,
		},
		{
			name:   "partial outranks empty and stale together",
			status: StatusPartial,
			stale:  true,
			empty:  []string{"go"},
			want:   StatusPartial,
		},
		{
			name:   "legacy manifest with no status reads as ready",
			status: "",
			want:   StatusReady,
		},
		{
			name:   "legacy manifest with no status still reports stale",
			status: "",
			stale:  true,
			want:   StatusStale,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{Status: tt.status}
			if got := EffectiveStatus(m, tt.stale, tt.empty); got != tt.want {
				t.Fatalf("EffectiveStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEffectiveStatusNilManifest(t *testing.T) {
	if got := EffectiveStatus(nil, true, []string{"go"}); got != StatusReady {
		t.Fatalf("EffectiveStatus(nil) = %q, want %q", got, StatusReady)
	}
}

func TestNewestSourceMTime(t *testing.T) {
	repo := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	writeAt(t, filepath.Join(repo, "a.go"), old)
	writeAt(t, filepath.Join(repo, "src", "b.ts"), recent)
	// Ignored: not a source extension, and newer than everything.
	writeAt(t, filepath.Join(repo, "README.md"), time.Now())
	// Ignored: inside a skipped directory, and newer than everything.
	writeAt(t, filepath.Join(repo, "node_modules", "dep.ts"), time.Now())

	if got := NewestSourceMTime(repo); got.Sub(recent).Abs() > 2*time.Second {
		t.Fatalf("NewestSourceMTime() = %v, want ~%v", got, recent)
	}
}

func TestNewestSourceMTimeNoSources(t *testing.T) {
	repo := t.TempDir()
	writeAt(t, filepath.Join(repo, "README.md"), time.Now())
	if got := NewestSourceMTime(repo); !got.IsZero() {
		t.Fatalf("NewestSourceMTime() = %v, want zero time", got)
	}
}

func TestNewestSourceMTimeMissingRepo(t *testing.T) {
	if got := NewestSourceMTime(filepath.Join(t.TempDir(), "gone")); !got.IsZero() {
		t.Fatalf("NewestSourceMTime() = %v, want zero time", got)
	}
}

func TestManifestHeadCommitRoundTrip(t *testing.T) {
	// A manifest written before head_commit existed unmarshals with "" and
	// re-marshals without the key — the 44 repos already on disk depend on
	// staying diagnosable without a rewrite.
	legacy := `{"schema_version":2,"repo_path":"/tmp/x","status":"ready","stats":{}}`
	var m Manifest
	if err := json.Unmarshal([]byte(legacy), &m); err != nil {
		t.Fatalf("unmarshal legacy manifest: %v", err)
	}
	if m.HeadCommit != "" {
		t.Errorf("HeadCommit = %q, want empty for a legacy manifest", m.HeadCommit)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if _, present := round["head_commit"]; present {
		t.Error("re-marshalled legacy manifest must omit the empty head_commit key")
	}

	m.HeadCommit = "aaaa1111"
	b, err = json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Manifest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.HeadCommit != "aaaa1111" {
		t.Errorf("HeadCommit = %q, want it preserved through a round trip", got.HeadCommit)
	}
}

// writeAt creates a file (and its parents) with a fixed modification time.
func writeAt(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}
