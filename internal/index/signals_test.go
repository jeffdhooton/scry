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

func TestEmptyLanguages(t *testing.T) {
	// ok is a language that indexed normally.
	ok := func(lang string, files, symbols int) IndexerResult {
		return IndexerResult{
			Language: lang, Status: IndexerOK, Tier: TierPrimary,
			FileCount: files, SymbolCount: symbols,
		}
	}

	tests := []struct {
		name     string
		indexers []IndexerResult
		want     []string
	}{
		{
			name:     "empty with files is flagged",
			indexers: []IndexerResult{ok("typescript", 412, 0)},
			want:     []string{"typescript"},
		},
		{
			name: "empty with no files is not flagged",
			// Zero symbols from zero files is the right answer, not a
			// failure — a repo with no source in that language at all.
			indexers: []IndexerResult{ok("python", 0, 0)},
			want:     nil,
		},
		{
			name:     "a language that found symbols is not flagged",
			indexers: []IndexerResult{ok("go", 120, 8087)},
			want:     nil,
		},
		{
			name: "a single empty language among healthy ones is flagged alone",
			indexers: []IndexerResult{
				ok("go", 120, 8087),
				ok("typescript", 412, 0),
				ok("php", 88, 1254),
			},
			want: []string{"typescript"},
		},
		{
			name: "every empty language is reported, in manifest order",
			indexers: []IndexerResult{
				ok("typescript", 412, 0),
				ok("go", 120, 900),
				ok("python", 33, 0),
			},
			want: []string{"typescript", "python"},
		},
		{
			name: "an incidental language is never flagged",
			// Incidental languages are deliberately not indexed deeply and
			// can't degrade a repo's status either — see deriveStatus.
			indexers: []IndexerResult{{
				Language: "python", Status: IndexerOK, Tier: TierIncidental,
				FileCount: 37, SymbolCount: 0,
			}},
			want: nil,
		},
		{
			name: "a missing indexer is partial's business, not empty's",
			indexers: []IndexerResult{{
				Language: "typescript", Status: IndexerMissing, Tier: TierPrimary,
				FileCount: 412, Error: "scip-typescript not found",
			}},
			want: nil,
		},
		{
			name: "a failed indexer is partial's business, not empty's",
			// Includes the parse-failure path: the binary ran, its dump
			// wouldn't parse, so counts are zero — but the status is failed.
			indexers: []IndexerResult{{
				Language: "go", Status: IndexerFailed, Tier: TierPrimary,
				FileCount: 120, Error: "parse go scip: unexpected EOF",
			}},
			want: nil,
		},
		{
			name: "a skipped indexer is not flagged",
			indexers: []IndexerResult{{
				Language: "php", Status: IndexerSkipped, Tier: TierIncidental,
				FileCount: 37,
			}},
			want: nil,
		},
		{
			name: "a legacy manifest with no per-language results flags nothing",
			// The 44 repos on disk predate IndexerResult entirely. They must
			// read as not-empty rather than as every-language-empty.
			indexers: nil,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EmptyLanguages(&Manifest{Indexers: tt.indexers})
			if len(got) != len(tt.want) {
				t.Fatalf("EmptyLanguages() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("EmptyLanguages() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestEmptyLanguagesNilManifest(t *testing.T) {
	if got := EmptyLanguages(nil); got != nil {
		t.Fatalf("EmptyLanguages(nil) = %v, want nil", got)
	}
}

func TestEmptyLanguagesRollsUpToRepoStatus(t *testing.T) {
	// The end-to-end shape from the goal: a repo whose indexer claimed success
	// across a non-empty file set and produced nothing must not report ready.
	m := &Manifest{
		Status:    StatusReady,
		IndexedAt: indexedAt,
		Indexers: []IndexerResult{{
			Language: "typescript", Status: IndexerOK, Tier: TierPrimary,
			FileCount: 412, SymbolCount: 0,
		}},
	}
	empty := EmptyLanguages(m)
	if len(empty) != 1 || empty[0] != "typescript" {
		t.Fatalf("EmptyLanguages() = %v, want [typescript]", empty)
	}
	if got := EffectiveStatus(m, false, empty); got != StatusEmpty {
		t.Fatalf("EffectiveStatus() = %q, want %q", got, StatusEmpty)
	}
}

func TestEmptyLanguagesNeedsNoReindex(t *testing.T) {
	// Both signals must be computable from a manifest alone — the 44 repos
	// already on disk are diagnosed by reading them, never by rebuilding
	// first. A manifest pointing at a repo path that no longer exists still
	// yields both answers without touching the filesystem.
	m := &Manifest{
		Status:     StatusReady,
		RepoPath:   filepath.Join(t.TempDir(), "deleted-repo"),
		IndexedAt:  indexedAt,
		HeadCommit: "aaaa1111",
		Indexers: []IndexerResult{{
			Language: "go", Status: IndexerOK, Tier: TierPrimary,
			FileCount: 120, SymbolCount: 0,
		}},
	}
	if got := EmptyLanguages(m); len(got) != 1 {
		t.Fatalf("EmptyLanguages() = %v, want one entry", got)
	}
	if !IsStale(m, "bbbb2222", time.Time{}) {
		t.Fatal("IsStale() = false, want true from the recorded commit alone")
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

// A language that detection counts but the mtime walk ignores would be
// silently un-stale-able: edit only those files and the index never looks
// out of date. Both sides now read langForExt, so this holds by construction
// — the test is here to fail loudly if a future language is wired into only
// one of them.
func TestNewestSourceMTimeSeesEveryDetectedLanguage(t *testing.T) {
	exts := []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".go", ".php", ".py"}
	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			if langForExt(ext) == "" {
				t.Fatalf("langForExt(%q) = \"\", want a language — detection would skip it too", ext)
			}
			repo := t.TempDir()
			touched := time.Now().Add(-1 * time.Hour)
			writeAt(t, filepath.Join(repo, "only"+ext), touched)

			got := NewestSourceMTime(repo)
			if got.IsZero() {
				t.Fatalf("NewestSourceMTime() = zero for a repo holding only %s; edits to %s files would never mark the index stale", ext, ext)
			}
			if got.Sub(touched).Abs() > 2*time.Second {
				t.Fatalf("NewestSourceMTime() = %v, want ~%v", got, touched)
			}
		})
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
