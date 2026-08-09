# Partial-Index Observability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Record why a repo index is degraded and stop incidental languages from degrading it in the first place.

**Architecture:** Per-indexer outcomes become a first-class `IndexerResult` recorded in each repo's `manifest.json`. Classification (`missing` vs `failed`) reuses the not-found sentinel errors the `internal/sources/*` packages already export. Language detection grows a second tier so an indexer only degrades repo status when its language is *primary* (declared by a marker file, or ≥10% of source files). The results surface through the existing status RPC and `scry doctor`.

**Tech Stack:** Go 1.26.2, stdlib `testing` only, BadgerDB (untouched here), cobra CLI, JSON-RPC over a Unix socket.

**Source spec:** `docs/superpowers/specs/2026-08-09-partial-index-observability-design.md`

## Global Constraints

- **Go 1.26.2**, module `github.com/jeffdhooton/scry`. No CGO, ever.
- **No new third-party dependencies.** Tests use stdlib `testing`; no testify, no mocks framework.
- **No `store.SchemaVersion` bump.** BadgerDB contents are untouched. The 44 existing repos must keep working against their current on-disk indexes and must not be force-reindexed.
- **`Manifest.Indexers` is additive JSON.** Legacy manifests with no `indexers` key must unmarshal cleanly to `Indexers == nil`, and every consumer must render correctly in that case.
- **No changes inside `internal/sources/*`.** The sentinels needed for classification already exist there.
- **JSON output by default** — this tool's primary user is an AI agent. Human formatting belongs only in `scry doctor`.
- **Verify command:** `go build ./... && go test ./...` — must pass before every commit.

---

### Task 1: `IndexerResult` type, classification, and status derivation

Pure logic, no filesystem. Everything else builds on the names defined here.

**Files:**
- Create: `internal/index/result.go`
- Create: `internal/index/result_test.go`

**Interfaces:**
- Consumes: `typescript.ErrIndexerNotFound`, `golang.ErrIndexerNotFound`, `php.ErrPhpNotFound`, `python.ErrIndexerNotFound` — existing exported sentinels in `internal/sources/*`.
- Produces:
  - `type IndexerResult struct` with fields `Language, Status, Tier string; FileCount int; Share float64; Error, Remedy string`
  - constants `IndexerOK, IndexerMissing, IndexerFailed, IndexerSkipped` (values `"ok"`, `"missing"`, `"failed"`, `"skipped"`)
  - constants `TierPrimary, TierIncidental` (values `"primary"`, `"incidental"`)
  - `func classify(language string, err error) (status, errMsg, remedy string)`
  - `func deriveStatus(results []IndexerResult) string`

- [ ] **Step 1: Write the failing test**

Create `internal/index/result_test.go`:

```go
package index

import (
	"fmt"
	"testing"

	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/python"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		language   string
		err        error
		wantStatus string
		wantRemedy string
	}{
		{
			name:       "nil error is ok",
			language:   "python",
			err:        nil,
			wantStatus: IndexerOK,
			wantRemedy: "",
		},
		{
			name:       "wrapped not-found sentinel is missing, with remedy",
			language:   "python",
			err:        fmt.Errorf("index python: %w", python.ErrIndexerNotFound),
			wantStatus: IndexerMissing,
			wantRemedy: "npm i -g @sourcegraph/scip-python",
		},
		{
			name:       "go sentinel is missing",
			language:   "go",
			err:        fmt.Errorf("run: %w", golang.ErrIndexerNotFound),
			wantStatus: IndexerMissing,
			wantRemedy: "check network access; scry auto-downloads scip-go into ~/.scry/bin",
		},
		{
			name:       "arbitrary error is failed, no remedy",
			language:   "go",
			err:        fmt.Errorf("exit status 2: panic in scip-go"),
			wantStatus: IndexerFailed,
			wantRemedy: "",
		},
		{
			name:       "unknown language with arbitrary error is failed",
			language:   "ruby",
			err:        fmt.Errorf("boom"),
			wantStatus: IndexerFailed,
			wantRemedy: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, errMsg, remedy := classify(tt.language, tt.err)
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if remedy != tt.wantRemedy {
				t.Errorf("remedy = %q, want %q", remedy, tt.wantRemedy)
			}
			if tt.err == nil {
				if errMsg != "" {
					t.Errorf("errMsg = %q, want empty for nil error", errMsg)
				}
			} else if errMsg != tt.err.Error() {
				t.Errorf("errMsg = %q, want %q", errMsg, tt.err.Error())
			}
		})
	}
}

func TestDeriveStatus(t *testing.T) {
	tests := []struct {
		name    string
		results []IndexerResult
		want    string
	}{
		{
			name: "all primary ok is ready",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "typescript", Tier: TierPrimary, Status: IndexerOK},
			},
			want: "ready",
		},
		{
			name: "primary missing is partial",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "python", Tier: TierPrimary, Status: IndexerMissing},
			},
			want: "partial",
		},
		{
			name: "primary failed is partial",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "go", Tier: TierPrimary, Status: IndexerFailed},
			},
			want: "partial",
		},
		{
			name: "incidental skipped alongside ok primaries is ready",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "python", Tier: TierIncidental, Status: IndexerSkipped},
			},
			want: "ready",
		},
		{
			name: "incidental failure never degrades",
			results: []IndexerResult{
				{Language: "php", Tier: TierPrimary, Status: IndexerOK},
				{Language: "python", Tier: TierIncidental, Status: IndexerMissing},
			},
			want: "ready",
		},
		{
			name:    "no results is ready",
			results: nil,
			want:    "ready",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveStatus(tt.results); got != tt.want {
				t.Errorf("deriveStatus = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/index/ -run 'TestClassify|TestDeriveStatus' -v`
Expected: FAIL — compile error, `undefined: IndexerOK`, `undefined: classify`, `undefined: deriveStatus`, `undefined: IndexerResult`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/index/result.go`:

```go
package index

import (
	"errors"

	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/php"
	"github.com/jeffdhooton/scry/internal/sources/python"
	"github.com/jeffdhooton/scry/internal/sources/typescript"
)

// Indexer outcome values recorded in IndexerResult.Status.
const (
	IndexerOK      = "ok"      // ran, output parsed
	IndexerMissing = "missing" // binary not installed
	IndexerFailed  = "failed"  // ran and errored, or output failed to parse
	IndexerSkipped = "skipped" // incidental language, deliberately not invoked
)

// Language tiers recorded in IndexerResult.Tier. Only primary languages can
// degrade a repo's status — see deriveStatus.
const (
	TierPrimary    = "primary"
	TierIncidental = "incidental"
)

// IndexerResult records the outcome of one language indexer for one build.
// Persisted in manifest.json so a degraded index can be diagnosed after the
// fact instead of only from the stderr of the build that produced it.
type IndexerResult struct {
	Language  string  `json:"language"`
	Status    string  `json:"status"`
	Tier      string  `json:"tier"`
	FileCount int     `json:"file_count"`
	Share     float64 `json:"share"`
	Error     string  `json:"error,omitempty"`
	Remedy    string  `json:"remedy,omitempty"`
}

// notFoundSentinels maps a language to the sentinel its source package
// returns when the indexer binary is absent. Classification keys off these
// because "you never installed scip-python" is a one-line fix for the
// operator while "scip-go crashed" is a bug worth reporting — and today both
// surface identically.
var notFoundSentinels = map[string]error{
	"typescript": typescript.ErrIndexerNotFound,
	"go":         golang.ErrIndexerNotFound,
	"php":        php.ErrPhpNotFound,
	"python":     python.ErrIndexerNotFound,
}

// indexerRemedies is the operator-facing fix for a missing indexer. These
// mirror the remedy strings scry doctor already prints for the same tools,
// deliberately: one wording for one problem.
var indexerRemedies = map[string]string{
	"typescript": "npm i -g @sourcegraph/scip-typescript",
	"go":         "check network access; scry auto-downloads scip-go into ~/.scry/bin",
	"php":        "install PHP 8.3+ and ensure `php` is on PATH",
	"python":     "npm i -g @sourcegraph/scip-python",
}

// classify converts one indexer's error into a (status, error, remedy)
// triple. A nil error is ok; a wrapped not-found sentinel is missing and
// carries a remedy; anything else is a genuine failure.
func classify(language string, err error) (status, errMsg, remedy string) {
	if err == nil {
		return IndexerOK, "", ""
	}
	if sentinel, ok := notFoundSentinels[language]; ok && errors.Is(err, sentinel) {
		return IndexerMissing, err.Error(), indexerRemedies[language]
	}
	return IndexerFailed, err.Error(), ""
}

// deriveStatus computes the manifest status from the full result set. Only
// primary languages can degrade a repo: an incidental language whose indexer
// is missing is a fact worth recording, not a reason to call an otherwise
// complete index degraded.
//
// "broken" is never returned. When no indexer produces output at all,
// buildAtLayout returns an error and writes no manifest, so that value in
// Manifest.Status is vestigial.
func deriveStatus(results []IndexerResult) string {
	for _, r := range results {
		if r.Tier != TierPrimary {
			continue
		}
		if r.Status == IndexerMissing || r.Status == IndexerFailed {
			return "partial"
		}
	}
	return "ready"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/index/ -run 'TestClassify|TestDeriveStatus' -v`
Expected: PASS, 11 subtests.

Then run the full suite: `go build ./... && go test ./...`
Expected: PASS (nothing else references these yet).

- [ ] **Step 5: Commit**

```bash
git add internal/index/result.go internal/index/result_test.go
git commit -m "feat(index): IndexerResult with missing-vs-failed classification"
```

---

### Task 2: Two-tier language detection

Replaces the flat 1% threshold. Also records the decision in `docs/DECISIONS.md`, since this is the judgment call the spec turns on.

**Files:**
- Create: `internal/index/detect.go`
- Create: `internal/index/detect_test.go`
- Modify: `internal/index/builder.go` — delete `detectLanguages` (lines 359-425) and `langForExt` (lines 427-435); they move to `detect.go`
- Modify: `docs/DECISIONS.md` — append the decision entry

**Interfaces:**
- Consumes: `TierPrimary`, `TierIncidental` from Task 1.
- Produces:
  - `type DetectedLanguage struct { Language, Tier string; FileCount int; Share float64; Marker string }`
  - `func detectLanguages(repoPath string) ([]DetectedLanguage, error)` — **signature change**, returns tiered results sorted by descending `FileCount` for stable output
  - `func primaryLanguages(dets []DetectedLanguage) []string` — for `Manifest.Languages`
  - `func langForExt(ext string) string` — unchanged behavior, moved

- [ ] **Step 1: Confirm the old signature has exactly one caller**

Run: `grep -rn "detectLanguages" --include='*.go' internal cmd`
Expected: the definition plus exactly one call at `internal/index/builder.go:160`. If more callers appear, every one must be updated in Step 5.

- [ ] **Step 2: Write the failing test**

Create `internal/index/detect_test.go`:

```go
package index

import (
	"os"
	"path/filepath"
	"testing"
)

// writeRepo materializes a fixture repo: each map key is a repo-relative
// path, each value is that file's contents.
func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

// nFiles generates n files named <prefix>N<ext> under dir.
func nFiles(dir, prefix, ext string, n int) map[string]string {
	out := map[string]string{}
	for i := 0; i < n; i++ {
		out[filepath.Join(dir, prefix+string(rune('a'+i%26))+string(rune('a'+i/26))+ext)] = "x"
	}
	return out
}

func merge(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// find returns the DetectedLanguage for lang, or a zero value if absent.
func find(dets []DetectedLanguage, lang string) DetectedLanguage {
	for _, d := range dets {
		if d.Language == lang {
			return d
		}
	}
	return DetectedLanguage{}
}

func TestDetectLanguages_MarkerPromotesLowShare(t *testing.T) {
	// 95 php files, 5 python files (5%), but a pyproject.toml declares
	// Python as a real component.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 95),
		nFiles("scripts", "g", ".py", 5),
		map[string]string{"pyproject.toml": "[project]\nname='x'\n"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	py := find(dets, "python")
	if py.Tier != TierPrimary {
		t.Errorf("python tier = %q, want %q (marker file should promote)", py.Tier, TierPrimary)
	}
	if py.Marker != "pyproject.toml" {
		t.Errorf("python marker = %q, want pyproject.toml", py.Marker)
	}
	if py.FileCount != 5 {
		t.Errorf("python file count = %d, want 5", py.FileCount)
	}
}

func TestDetectLanguages_NoMarkerLowShareIsIncidental(t *testing.T) {
	// The childscribe-beta-r4 shape: a dominant PHP app with a handful of
	// stray .py scripts and no Python marker anywhere.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 95),
		nFiles("scripts", "g", ".py", 5),
		map[string]string{"composer.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if got := find(dets, "python").Tier; got != TierIncidental {
		t.Errorf("python tier = %q, want %q", got, TierIncidental)
	}
	if got := find(dets, "php").Tier; got != TierPrimary {
		t.Errorf("php tier = %q, want %q", got, TierPrimary)
	}
}

func TestDetectLanguages_HighShareWithoutMarkerIsPrimary(t *testing.T) {
	// 70 go files, 30 python files (30%), no python marker. Undeclared but
	// substantial code still deserves an index.
	root := writeRepo(t, merge(
		nFiles("cmd", "f", ".go", 70),
		nFiles("tools", "g", ".py", 30),
		map[string]string{"go.mod": "module x\n"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	py := find(dets, "python")
	if py.Tier != TierPrimary {
		t.Errorf("python tier = %q, want %q (30%% share should promote)", py.Tier, TierPrimary)
	}
	if py.Marker != "" {
		t.Errorf("python marker = %q, want empty (share-promoted, not marker-promoted)", py.Marker)
	}
}

func TestDetectLanguages_BelowMinShareIsAbsent(t *testing.T) {
	// 1 python file in 200 = 0.5%, below the 1% floor.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 199),
		map[string]string{"one.py": "x", "composer.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if find(dets, "python").Language != "" {
		t.Errorf("python should be absent entirely below the 1%% floor, got %+v", find(dets, "python"))
	}
}

func TestDetectLanguages_MarkerDoesNotResurrectBelowFloor(t *testing.T) {
	// A marker file cannot promote a language with no source files at all.
	// Laravel apps ship a package.json but may have zero .ts/.js of their own.
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 100),
		map[string]string{"composer.json": "{}", "package.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if find(dets, "typescript").Language != "" {
		t.Error("typescript should be absent with zero .ts files despite package.json")
	}
	if find(dets, "javascript").Language != "" {
		t.Error("javascript should be absent with zero .js files despite package.json")
	}
}

func TestDetectLanguages_SkipsVendorAndVenv(t *testing.T) {
	root := writeRepo(t, merge(
		nFiles("app", "f", ".php", 100),
		nFiles("vendor/pkg", "v", ".php", 50),
		nFiles(".venv/lib", "w", ".py", 500),
		nFiles("node_modules/dep", "n", ".js", 500),
		map[string]string{"composer.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	if find(dets, "python").Language != "" {
		t.Error("python from .venv must not be counted")
	}
	if find(dets, "javascript").Language != "" {
		t.Error("javascript from node_modules must not be counted")
	}
	if got := find(dets, "php").FileCount; got != 100 {
		t.Errorf("php file count = %d, want 100 (vendor/ excluded)", got)
	}
}

func TestPrimaryLanguages(t *testing.T) {
	dets := []DetectedLanguage{
		{Language: "php", Tier: TierPrimary},
		{Language: "python", Tier: TierIncidental},
		{Language: "typescript", Tier: TierPrimary},
	}
	got := primaryLanguages(dets)
	want := []string{"php", "typescript"}
	if len(got) != len(want) {
		t.Fatalf("primaryLanguages = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("primaryLanguages = %v, want %v", got, want)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/index/ -run 'TestDetectLanguages|TestPrimaryLanguages' -v`
Expected: FAIL — compile error, `undefined: DetectedLanguage`, `undefined: primaryLanguages`, and `detectLanguages` returning the wrong type.

- [ ] **Step 4: Write the implementation**

Create `internal/index/detect.go`:

```go
package index

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// minShare is the file-share floor below which a language is ignored
// entirely — it is not indexed and not recorded.
const minShare = 0.01

// primaryShare is the file-share at which a language is primary even with no
// marker file. Undeclared-but-substantial code still deserves an index.
const primaryShare = 0.10

// languageMarkers maps a language to the root-level files that declare it.
// A marker is the strongest available signal that a language is a real
// component of the repo rather than incidental tooling: a repo with a real
// Python component nearly always declares one, and a Laravel app with a
// handful of stray scripts does not.
var languageMarkers = map[string][]string{
	"typescript": {"package.json", "tsconfig.json"},
	"javascript": {"package.json", "tsconfig.json"},
	"go":         {"go.mod"},
	"php":        {"composer.json"},
	"python":     {"pyproject.toml", "requirements.txt", "setup.py", "Pipfile"},
}

// DetectedLanguage is one language found in a repo, with the evidence that
// decided its tier.
type DetectedLanguage struct {
	Language  string
	Tier      string
	FileCount int
	Share     float64
	Marker    string // marker filename that promoted it; "" if share-promoted
}

// skipDirs are never walked during detection.
var skipDirs = map[string]bool{
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
	// Python runtime / venv / cache directories. Counting their .py files
	// as project code would skew language detection and cause unnecessary
	// indexer invocations on dependency-only trees.
	".venv":         true,
	"venv":          true,
	"env":           true,
	"__pycache__":   true,
	".mypy_cache":   true,
	".pytest_cache": true,
	".ruff_cache":   true,
	".tox":          true,
}

// detectLanguages walks the repo, counts source files by language, and
// assigns each language a tier. A language below minShare is omitted
// entirely. Results are sorted by descending file count so output is stable.
func detectLanguages(repoPath string) ([]DetectedLanguage, error) {
	counts := map[string]int{}
	var total int
	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		switch ext := strings.ToLower(filepath.Ext(d.Name())); ext {
		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
			counts[langForExt(ext)]++
			total++
		case ".go":
			counts["go"]++
			total++
		case ".php":
			counts["php"]++
			total++
		case ".py":
			counts["python"]++
			total++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}

	var out []DetectedLanguage
	for lang, c := range counts {
		share := float64(c) / float64(total)
		if share < minShare {
			continue
		}
		det := DetectedLanguage{
			Language:  lang,
			Tier:      TierIncidental,
			FileCount: c,
			Share:     share,
		}
		if marker := findMarker(repoPath, lang); marker != "" {
			det.Tier = TierPrimary
			det.Marker = marker
		} else if share >= primaryShare {
			det.Tier = TierPrimary
		}
		out = append(out, det)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FileCount != out[j].FileCount {
			return out[i].FileCount > out[j].FileCount
		}
		return out[i].Language < out[j].Language
	})
	return out, nil
}

// findMarker returns the first root-level marker file present for lang, or
// "" if none. Markers are checked at the repo root only — a marker buried in
// a subdirectory describes that subdirectory, not the repo.
func findMarker(repoPath, lang string) string {
	for _, name := range languageMarkers[lang] {
		if _, err := os.Stat(filepath.Join(repoPath, name)); err == nil {
			return name
		}
	}
	return ""
}

// primaryLanguages returns just the primary language names, preserving the
// detection order. This is what Manifest.Languages records, so its existing
// meaning ("the languages this repo is indexed for") is unchanged.
func primaryLanguages(dets []DetectedLanguage) []string {
	var out []string
	for _, d := range dets {
		if d.Tier == TierPrimary {
			out = append(out, d.Language)
		}
	}
	return out
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
```

- [ ] **Step 5: Delete the moved functions from `builder.go`**

Delete `detectLanguages` (the block starting `func detectLanguages(repoPath string) ([]string, error) {`, through its closing brace) and `langForExt` from `internal/index/builder.go`. Leave `contains` in place — Task 3 still uses it.

`builder.go:160` will not compile until Task 3. That is expected; Step 6 tests only the new package files.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go vet ./internal/index/ 2>&1 | head` — expect errors only about `builder.go` using the new `detectLanguages` return type. Then temporarily confirm the detection logic alone:

Run: `go test ./internal/index/ -run 'TestDetectLanguages|TestPrimaryLanguages' -v`
Expected: FAIL TO BUILD until Task 3 lands, because `builder.go:160` still assigns the result to a `[]string`. **Do not commit here — proceed directly to Task 3, which restores the build.** Tasks 2 and 3 share one commit.

- [ ] **Step 7: Record the decision**

Append to `docs/DECISIONS.md`, matching the file's existing entry format:

```markdown
## Two-tier language detection (2026-08-09)

**Decision:** A detected language is *primary* if it has a root-level marker
file (`composer.json`, `go.mod`, `package.json`/`tsconfig.json`,
`pyproject.toml`/`requirements.txt`/`setup.py`/`Pipfile`) or holds ≥10% of
source files. Otherwise, above a 1% floor, it is *incidental*: its indexer is
not invoked and its absence never degrades repo status.

**Why:** The previous flat 1% threshold invoked a full indexer for any
language clearing 1% of files. Measured on `childscribe-beta-r4`: 855 PHP,
110 TS/JS, 37 Python files and no Python marker. Python cleared the bar at
3.7%, `scip-python` was not installed, and the entire repo was reported
`partial` — 855 PHP files' worth of complete index described as degraded
because of 37 incidental scripts. 17 of 44 indexed repos were in this state.

The marker file carries the weight rather than share alone because it is a
statement of intent: a repo with a real component in a language declares it.
Share is the fallback for undeclared-but-substantial code.

**What would change our minds:** A repo with a genuine, sizable component in
a language that declares no marker and sits under 10% — it would be silently
skipped. If that shows up, add the marker filename rather than lowering the
share, or introduce a per-repo override in the manifest.
```

---

### Task 3: Wire results through the builder

Restores the build, records results in the manifest, and stops invoking incidental indexers.

**Files:**
- Modify: `internal/index/builder.go` — `Manifest` struct (line ~34), `buildAtLayout` indexer block (lines ~166-208), status derivation (lines ~314-320)
- Create: `internal/index/builder_test.go`

**Interfaces:**
- Consumes: `IndexerResult`, `classify`, `deriveStatus`, `IndexerOK/Missing/Failed/Skipped`, `TierPrimary/TierIncidental` (Task 1); `detectLanguages`, `primaryLanguages`, `DetectedLanguage` (Task 2).
- Produces: `Manifest.Indexers []IndexerResult` — read by Tasks 4 and 5.

- [ ] **Step 1: Write the failing test**

Create `internal/index/builder_test.go`:

```go
package index

import (
	"encoding/json"
	"testing"
	"time"
)

func TestManifest_LegacyJSONWithoutIndexers(t *testing.T) {
	// A manifest written before this feature must still unmarshal, with
	// Indexers nil, and must not lose fields on re-marshal. The 44 repos
	// already on disk depend on this.
	legacy := `{
	  "schema_version": 2,
	  "repo_path": "/Users/jeff/workspace/example",
	  "languages": ["php", "typescript"],
	  "indexed_at": "2026-08-01T12:00:00Z",
	  "status": "partial",
	  "stats": {"documents": 10, "symbols": 20, "references": 30}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(legacy), &m); err != nil {
		t.Fatalf("unmarshal legacy manifest: %v", err)
	}
	if m.Indexers != nil {
		t.Errorf("Indexers = %v, want nil for a legacy manifest", m.Indexers)
	}
	if m.Status != "partial" {
		t.Errorf("Status = %q, want partial", m.Status)
	}
	if len(m.Languages) != 2 {
		t.Errorf("Languages = %v, want 2 entries", m.Languages)
	}
	if m.Stats.Documents != 10 {
		t.Errorf("Stats.Documents = %d, want 10", m.Stats.Documents)
	}

	// Re-marshal must not emit an empty "indexers" key.
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if _, present := round["indexers"]; present {
		t.Error("re-marshalled legacy manifest must omit the empty indexers key")
	}
}

func TestManifest_IndexersRoundTrip(t *testing.T) {
	m := Manifest{
		SchemaVersion: 2,
		RepoPath:      "/tmp/example",
		Languages:     []string{"php"},
		IndexedAt:     time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC),
		Status:        "ready",
		Indexers: []IndexerResult{
			{Language: "php", Status: IndexerOK, Tier: TierPrimary, FileCount: 855, Share: 0.85},
			{Language: "python", Status: IndexerSkipped, Tier: TierIncidental, FileCount: 37, Share: 0.037},
		},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Manifest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Indexers) != 2 {
		t.Fatalf("Indexers len = %d, want 2", len(got.Indexers))
	}
	if got.Indexers[1].Status != IndexerSkipped || got.Indexers[1].Tier != TierIncidental {
		t.Errorf("second indexer = %+v, want skipped/incidental", got.Indexers[1])
	}
	if got.Indexers[1].FileCount != 37 {
		t.Errorf("FileCount = %d, want 37", got.Indexers[1].FileCount)
	}
}

func TestBuildResults_IncidentalIsSkippedNotInvoked(t *testing.T) {
	// The childscribe shape: PHP dominant with a marker, Python incidental
	// with none. The python entry must be recorded as skipped, and the
	// caller must never have been asked to run it.
	dets := []DetectedLanguage{
		{Language: "php", Tier: TierPrimary, FileCount: 855, Share: 0.855, Marker: "composer.json"},
		{Language: "python", Tier: TierIncidental, FileCount: 37, Share: 0.037},
	}
	var invoked []string
	results := buildResults(dets, func(language string) error {
		invoked = append(invoked, language)
		return nil
	})

	if len(invoked) != 1 || invoked[0] != "php" {
		t.Errorf("invoked = %v, want [php] only", invoked)
	}
	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	py := results[1]
	if py.Language != "python" || py.Status != IndexerSkipped {
		t.Errorf("python result = %+v, want skipped", py)
	}
	if py.Error != "" {
		t.Errorf("skipped entry should carry no error, got %q", py.Error)
	}
	if deriveStatus(results) != "ready" {
		t.Errorf("status = %q, want ready", deriveStatus(results))
	}
}

func TestBuildResults_JavascriptFoldsIntoTypescriptIndexer(t *testing.T) {
	// scip-typescript covers both languages. One invocation, one result,
	// counts summed, tier taken as the stronger of the two.
	dets := []DetectedLanguage{
		{Language: "javascript", Tier: TierPrimary, FileCount: 60, Share: 0.6, Marker: "package.json"},
		{Language: "typescript", Tier: TierIncidental, FileCount: 40, Share: 0.4},
	}
	var invoked []string
	results := buildResults(dets, func(language string) error {
		invoked = append(invoked, language)
		return nil
	})

	if len(invoked) != 1 || invoked[0] != "typescript" {
		t.Errorf("invoked = %v, want [typescript] once", invoked)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].FileCount != 100 {
		t.Errorf("FileCount = %d, want 100 (both languages summed)", results[0].FileCount)
	}
	if results[0].Tier != TierPrimary {
		t.Errorf("Tier = %q, want primary (stronger of the two)", results[0].Tier)
	}
}

func TestBuildResults_MissingPrimaryDegrades(t *testing.T) {
	dets := []DetectedLanguage{
		{Language: "go", Tier: TierPrimary, FileCount: 100, Share: 1.0, Marker: "go.mod"},
	}
	results := buildResults(dets, func(language string) error {
		return golangNotFound()
	})
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Status != IndexerMissing {
		t.Errorf("Status = %q, want missing", results[0].Status)
	}
	if results[0].Remedy == "" {
		t.Error("missing primary indexer must carry a remedy")
	}
	if deriveStatus(results) != "partial" {
		t.Errorf("status = %q, want partial", deriveStatus(results))
	}
}
```

Add this helper at the bottom of `internal/index/builder_test.go`:

```go
// golangNotFound wraps the scip-go sentinel the way buildAtLayout does.
func golangNotFound() error {
	return fmt.Errorf("scip-go: %w", golang.ErrIndexerNotFound)
}
```

and add `"fmt"` plus `"github.com/jeffdhooton/scry/internal/sources/golang"` to that file's imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/index/ -v`
Expected: FAIL — `undefined: buildResults`, plus `Manifest` has no field `Indexers`, plus `builder.go:160` type mismatch from Task 2.

- [ ] **Step 3: Add the `Indexers` field to `Manifest`**

In `internal/index/builder.go`, extend the struct (currently at line ~34):

```go
type Manifest struct {
	SchemaVersion int             `json:"schema_version"`
	RepoPath      string          `json:"repo_path"`
	Languages     []string        `json:"languages"`
	IndexedAt     time.Time       `json:"indexed_at"`
	Status        string          `json:"status"` // "ready" | "partial"
	FailedFiles   int             `json:"failed_files,omitempty"`
	Stats         scip.Stats      `json:"stats"`
	CoverageStats *coverage.Stats `json:"coverage_stats,omitempty"`
	// Indexers records the per-language outcome of this build. Additive:
	// manifests written before this field existed unmarshal with a nil
	// slice, and every consumer must render correctly in that case.
	Indexers []IndexerResult `json:"indexers,omitempty"`
}
```

- [ ] **Step 4: Add `buildResults`**

Add to `internal/index/builder.go`, above `buildAtLayout`:

```go
// indexerFor maps a detected language to the indexer that covers it.
// scip-typescript handles both TypeScript and JavaScript, so both fold into
// a single "typescript" invocation.
func indexerFor(language string) string {
	if language == "javascript" {
		return "typescript"
	}
	return language
}

// buildResults runs one indexer per detected primary language via run, and
// records an IndexerResult for every detected language — including the
// incidental ones that are deliberately never invoked. Results are ordered
// by the indexer's first appearance in dets, so output is stable.
//
// run is injected so the decision logic is testable without a real repo or
// real indexer binaries on PATH.
func buildResults(dets []DetectedLanguage, run func(language string) error) []IndexerResult {
	// Fold detected languages onto their indexer, summing file counts and
	// taking the stronger tier.
	order := []string{}
	agg := map[string]*IndexerResult{}
	for _, d := range dets {
		key := indexerFor(d.Language)
		cur, ok := agg[key]
		if !ok {
			agg[key] = &IndexerResult{
				Language:  key,
				Tier:      d.Tier,
				FileCount: d.FileCount,
				Share:     d.Share,
			}
			order = append(order, key)
			continue
		}
		cur.FileCount += d.FileCount
		cur.Share += d.Share
		if d.Tier == TierPrimary {
			cur.Tier = TierPrimary
		}
	}

	out := make([]IndexerResult, 0, len(order))
	for _, key := range order {
		r := *agg[key]
		if r.Tier != TierPrimary {
			r.Status = IndexerSkipped
			out = append(out, r)
			continue
		}
		r.Status, r.Error, r.Remedy = classify(key, run(key))
		out = append(out, r)
	}
	return out
}
```

- [ ] **Step 5: Rewrite the indexer block in `buildAtLayout`**

Replace lines ~160-208 of `internal/index/builder.go` (from `languages, err := detectLanguages(repoPath)` through the closing brace of the `python` block) with:

```go
	dets, err := detectLanguages(repoPath)
	if err != nil {
		return nil, fmt.Errorf("detect languages: %w", err)
	}
	languages := primaryLanguages(dets)
	if len(languages) == 0 {
		return nil, errors.New("no supported languages detected in repo")
	}

	// Run every primary indexer. Each writes its own scip-<lang>.bin. We
	// collect (language, scipPath) pairs and parse them sequentially after
	// all indexers finish — keeps the BadgerDB write batch contiguous.
	type indexed struct {
		language string
		scipPath string
	}
	var produced []indexed
	binDir := filepath.Join(scryHome, "bin")

	results := buildResults(dets, func(language string) error {
		out := layout.scipPath(language)
		var err error
		switch language {
		case "typescript":
			_, err = typescript.Index(ctx, repoPath, out)
		case "go":
			_, err = golang.Index(ctx, binDir, repoPath, out)
		case "php":
			_, err = php.Index(ctx, binDir, repoPath, out)
		case "python":
			_, err = python.Index(ctx, binDir, repoPath, out)
		default:
			return fmt.Errorf("no indexer for language %q", language)
		}
		if err != nil {
			return err
		}
		produced = append(produced, indexed{language, out})
		return nil
	})

	if len(produced) == 0 {
		// Every indexer failed. Surface the first real error verbatim.
		for _, r := range results {
			if r.Error != "" {
				return nil, errors.New(r.Error)
			}
		}
		return nil, fmt.Errorf("no supported indexer ran on repo languages %v", languages)
	}
```

- [ ] **Step 6: Replace the status derivation**

Replace lines ~314-320 (the `status := "ready"` / `if len(indexerErrs) > 0` block) with:

```go
	status := deriveStatus(results)
	if status != "ready" {
		fmt.Fprintf(os.Stderr, "scry: status=%s\n", status)
		for _, r := range results {
			if r.Status == IndexerMissing || r.Status == IndexerFailed {
				fmt.Fprintf(os.Stderr, "scry:   %s: %s — %s\n", r.Language, r.Status, r.Error)
				if r.Remedy != "" {
					fmt.Fprintf(os.Stderr, "scry:     fix: %s\n", r.Remedy)
				}
			}
		}
	}
```

- [ ] **Step 7: Record results in the manifest**

In the `manifest := &Manifest{...}` literal (~line 323), add:

```go
		Indexers:      results,
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go build ./... && go test ./internal/index/ -v`
Expected: PASS — all of Task 1, Task 2, and Task 3's tests.

If the compiler reports `indexerErrs` declared and not used, delete its `var indexerErrs []error` declaration — it is fully replaced by `results`.

Then the full suite: `go test ./...`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/index/ docs/DECISIONS.md
git commit -m "feat(index): two-tier detection, per-indexer results in the manifest

An incidental language no longer invokes its indexer or degrades repo
status. Every detected language's outcome is recorded in manifest.json."
```

---

### Task 4: Surface results through the status RPC

**Files:**
- Modify: `internal/daemon/methods.go` — `InitResult` (line ~70), `RepoStatusEntry` (line ~144), `handleInit` return (line ~118), both `handleStatus` construction sites (lines ~165 and ~190)

**Interfaces:**
- Consumes: `index.Manifest.Indexers` (Task 3).
- Produces: `indexers` key in the `init` and `status` RPC results, inherited by the `scry_status` MCP tool with no further work.

- [ ] **Step 1: Add the field to `InitResult`**

```go
type InitResult struct {
	Repo      string                `json:"repo"`
	Languages []string              `json:"languages"`
	Status    string                `json:"status"`
	Stats     interface{}           `json:"stats"`
	ElapsedMs int64                 `json:"elapsed_ms"`
	Indexers  []index.IndexerResult `json:"indexers,omitempty"`
}
```

And in `handleInit`'s return literal, add:

```go
		Indexers:  manifest.Indexers,
```

- [ ] **Step 2: Add the field to `RepoStatusEntry`**

```go
type RepoStatusEntry struct {
	Repo      string                `json:"repo"`
	Status    string                `json:"status"`
	Languages []string              `json:"languages,omitempty"`
	IndexedAt time.Time             `json:"indexed_at,omitempty"`
	Indexers  []index.IndexerResult `json:"indexers,omitempty"`
}
```

- [ ] **Step 3: Populate it at both construction sites**

`handleStatus` builds `RepoStatusEntry` twice — once from the in-memory registry (~line 165) and once from the on-disk manifest scan (~line 190). **Both** need `Indexers: manifest.Indexers` / `Indexers: m.Indexers` respectively. Missing the second means repos not yet loaded by this daemon silently lose the field.

- [ ] **Step 4: Verify the build and the wire format**

Run: `go build ./... && go test ./...`
Expected: PASS.

Then exercise it end to end:

```bash
go build -o /tmp/scry-verify ./cmd/scry
/tmp/scry-verify stop || true
/tmp/scry-verify init . 2>&1 | tail -5
/tmp/scry-verify status | python3 -c 'import json,sys; d=json.load(sys.stdin); print([r for r in d["repos"] if "scry" in r["repo"]][0])'
```

Expected: the scry repo entry includes an `indexers` array with a single `{"language":"go","status":"ok","tier":"primary",...}` entry.

- [ ] **Step 5: Commit**

```bash
git add internal/daemon/methods.go
git commit -m "feat(daemon): expose per-indexer results in init and status RPCs"
```

---

### Task 5: Surface results in `scry doctor`

Two changes: the current-repo check names what broke, and a new check connects a globally missing indexer to the repos it degraded. Also updates the README, since this is the last user-visible surface.

**Files:**
- Create: `internal/doctor/impact.go`
- Create: `internal/doctor/impact_test.go`
- Modify: `internal/doctor/doctor.go` — `checkCurrentRepo` (lines ~908-931), `Run` (add the new check after line ~131)
- Modify: `README.md` — "Known limitations"

**Interfaces:**
- Consumes: `index.Manifest.Indexers`, `index.IndexerMissing`, `index.IndexerFailed`, `index.TierPrimary` (Tasks 1 and 3); the existing `Check`/`Status`/`Category` types in `doctor.go`.
- Produces: `func checkIndexerImpact(scryHome string, prior []Check) Check` with ID `indexers.impact`.

- [ ] **Step 1: Write the failing test**

Create `internal/doctor/impact_test.go`:

```go
package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/index"
)

// writeManifests materializes fake ~/.scry/repos/<n>/manifest.json entries
// and returns the scryHome that contains them.
func writeManifests(t *testing.T, manifests ...index.Manifest) string {
	t.Helper()
	home := t.TempDir()
	for i, m := range manifests {
		dir := filepath.Join(home, "repos", "repo"+string(rune('a'+i)))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return home
}

func TestCheckIndexerImpact_CountsOnlyPrimaryRepos(t *testing.T) {
	home := writeManifests(t,
		index.Manifest{
			RepoPath: "/a",
			Indexers: []index.IndexerResult{
				{Language: "python", Tier: index.TierPrimary, Status: index.IndexerMissing},
			},
		},
		index.Manifest{
			RepoPath: "/b",
			Indexers: []index.IndexerResult{
				{Language: "python", Tier: index.TierPrimary, Status: index.IndexerMissing},
			},
		},
		index.Manifest{
			// Incidental — skipped, not degraded. Must not be counted.
			RepoPath: "/c",
			Indexers: []index.IndexerResult{
				{Language: "python", Tier: index.TierIncidental, Status: index.IndexerSkipped},
			},
		},
	)
	prior := []Check{
		{ID: "indexers.scip_python", Status: StatusWarn, Remedy: "npm i -g @sourcegraph/scip-python"},
	}

	got := checkIndexerImpact(home, prior)
	if got.Status != StatusWarn {
		t.Errorf("Status = %v, want Warn", got.Status)
	}
	if !strings.Contains(got.Detail, "2 indexed repo") {
		t.Errorf("Detail = %q, want it to name 2 affected repos", got.Detail)
	}
	if !strings.Contains(got.Detail, "python") {
		t.Errorf("Detail = %q, want it to name python", got.Detail)
	}
	if got.Remedy != "npm i -g @sourcegraph/scip-python" {
		t.Errorf("Remedy = %q, want the install command", got.Remedy)
	}
}

func TestCheckIndexerImpact_NothingMissingPasses(t *testing.T) {
	home := writeManifests(t, index.Manifest{
		RepoPath: "/a",
		Indexers: []index.IndexerResult{
			{Language: "go", Tier: index.TierPrimary, Status: index.IndexerOK},
		},
	})
	prior := []Check{{ID: "indexers.scip_python", Status: StatusPass}}

	got := checkIndexerImpact(home, prior)
	if got.Status != StatusPass {
		t.Errorf("Status = %v, want Pass when no indexer is missing", got.Status)
	}
}

func TestCheckIndexerImpact_MissingButNoReposAffected(t *testing.T) {
	// scip-python missing, but no indexed repo treats Python as primary.
	// Not worth a warning.
	home := writeManifests(t, index.Manifest{
		RepoPath: "/a",
		Indexers: []index.IndexerResult{
			{Language: "php", Tier: index.TierPrimary, Status: index.IndexerOK},
		},
	})
	prior := []Check{{ID: "indexers.scip_python", Status: StatusWarn, Remedy: "npm i -g @sourcegraph/scip-python"}}

	got := checkIndexerImpact(home, prior)
	if got.Status != StatusPass {
		t.Errorf("Status = %v, want Pass when no repo is affected", got.Status)
	}
}
```

Create `internal/doctor/doctor_test.go`:

```go
package doctor

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
)

// seedRepoManifest writes a manifest for repoPath into scryHome using the
// same layout resolution the doctor uses, and returns scryHome.
func seedRepoManifest(t *testing.T, repoPath string, m index.Manifest) string {
	t.Helper()
	home := t.TempDir()
	layout := index.Layout(home, repoPath)
	if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(layout.ManifestPath, b, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return home
}

func TestCheckCurrentRepo_NamesTheFailingIndexer(t *testing.T) {
	repo := t.TempDir()
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:  repo,
		Languages: []string{"php", "python"},
		IndexedAt: time.Now(),
		Status:    "partial",
		Indexers: []index.IndexerResult{
			{Language: "php", Tier: index.TierPrimary, Status: index.IndexerOK},
			{
				Language: "python", Tier: index.TierPrimary, Status: index.IndexerMissing,
				Error:  "scip-python not found on PATH",
				Remedy: "npm i -g @sourcegraph/scip-python",
			},
		},
	})

	got := checkCurrentRepo(home, repo)
	if got.Status != StatusWarn {
		t.Errorf("Status = %v, want Warn", got.Status)
	}
	if !strings.Contains(got.Detail, "python") || !strings.Contains(got.Detail, "missing") {
		t.Errorf("Detail = %q, want it to name the python indexer as missing", got.Detail)
	}
	if got.Remedy != "npm i -g @sourcegraph/scip-python" {
		t.Errorf("Remedy = %q, want the install command", got.Remedy)
	}
}

func TestCheckCurrentRepo_LegacyManifestStillRenders(t *testing.T) {
	// A manifest written before Indexers existed must render as it does
	// today — no panic, no empty breakdown appended.
	repo := t.TempDir()
	home := seedRepoManifest(t, repo, index.Manifest{
		RepoPath:  repo,
		Languages: []string{"go"},
		IndexedAt: time.Now(),
		Status:    "ready",
	})

	got := checkCurrentRepo(home, repo)
	if got.Status != StatusPass {
		t.Errorf("Status = %v, want Pass", got.Status)
	}
	if !strings.Contains(got.Detail, "ready") {
		t.Errorf("Detail = %q, want it to contain the status label", got.Detail)
	}
	if strings.Contains(got.Detail, "—  ") {
		t.Errorf("Detail = %q, has a dangling separator from an empty breakdown", got.Detail)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/doctor/ -v`
Expected: FAIL — `undefined: checkIndexerImpact`, and `TestCheckCurrentRepo_NamesTheFailingIndexer` failing because `Detail` does not mention python.

- [ ] **Step 3: Implement `checkIndexerImpact`**

Create `internal/doctor/impact.go`:

```go
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeffdhooton/scry/internal/index"
)

// checkIDLanguages maps each indexer environment check to the manifest
// language it gates. When one of those checks warns, this check answers the
// question it leaves open: how many indexed repos does that actually degrade?
var checkIDLanguages = map[string]string{
	"indexers.scip_python":     "python",
	"indexers.scip_typescript": "typescript",
	"indexers.scip_go":         "go",
	"indexers.php":             "php",
}

// checkIndexerImpact cross-references the indexer checks already run against
// every manifest on disk. Without it, doctor reports "scip-python not on
// PATH" as a mild environment note while 17 indexed repos silently serve
// degraded results.
func checkIndexerImpact(scryHome string, prior []Check) Check {
	base := Check{
		ID:       "indexers.impact",
		Category: CategoryIndexers,
		Name:     "indexer impact",
	}

	// Which languages are currently unavailable, and how to fix each.
	missing := map[string]string{} // language -> remedy
	for _, c := range prior {
		lang, ok := checkIDLanguages[c.ID]
		if !ok || (c.Status != StatusWarn && c.Status != StatusFail) {
			continue
		}
		missing[lang] = c.Remedy
	}
	if len(missing) == 0 {
		base.Status = StatusPass
		base.Detail = "all indexers available"
		return base
	}

	// Count repos where a missing language is primary.
	affected := map[string]int{}
	reposDir := filepath.Join(scryHome, "repos")
	entries, _ := os.ReadDir(reposDir)
	for _, ent := range entries {
		b, err := os.ReadFile(filepath.Join(reposDir, ent.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m index.Manifest
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		for _, r := range m.Indexers {
			if r.Tier != index.TierPrimary {
				continue
			}
			if _, ok := missing[r.Language]; ok {
				affected[r.Language]++
			}
		}
	}
	if len(affected) == 0 {
		base.Status = StatusPass
		base.Detail = "missing indexers affect no indexed repo"
		return base
	}

	langs := make([]string, 0, len(affected))
	for l := range affected {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	var parts []string
	var remedies []string
	for _, l := range langs {
		parts = append(parts, fmt.Sprintf("%s missing — affects %d indexed repo(s)", l, affected[l]))
		if r := missing[l]; r != "" {
			remedies = append(remedies, r)
		}
	}
	base.Status = StatusWarn
	base.Detail = strings.Join(parts, "; ")
	base.Remedy = strings.Join(remedies, " && ")
	return base
}
```

- [ ] **Step 4: Extend `checkCurrentRepo`**

In `internal/doctor/doctor.go`, replace the `detail := fmt.Sprintf(...)` block and the return at the end of `checkCurrentRepo` (lines ~920-931) with:

```go
	detail := fmt.Sprintf("%s — %d docs, %d symbols, %d refs, indexed %s ago (%s)",
		statusLabel,
		m.Stats.Documents, m.Stats.Symbols, m.Stats.References,
		age, strings.Join(m.Languages, "+"))

	// Name what actually broke. Legacy manifests have no Indexers and fall
	// through with today's output.
	var remedy string
	var degraded []string
	for _, r := range m.Indexers {
		switch r.Status {
		case index.IndexerMissing, index.IndexerFailed:
			degraded = append(degraded, fmt.Sprintf("%s %s (%s)", r.Language, r.Status, r.Error))
			if remedy == "" {
				remedy = r.Remedy
			}
		case index.IndexerSkipped:
			degraded = append(degraded, fmt.Sprintf("%s skipped (incidental, %d files)", r.Language, r.FileCount))
		}
	}
	if len(degraded) > 0 {
		detail += " — " + strings.Join(degraded, "; ")
	}

	return Check{
		ID:       "repo.current",
		Category: CategoryRepo,
		Name:     abs,
		Status:   status,
		Detail:   detail,
		Remedy:   remedy,
	}
```

Add `"github.com/jeffdhooton/scry/internal/index"` to the imports if it is not already there (it is — `checkCurrentRepo` already unmarshals into `index.Manifest`).

- [ ] **Step 5: Wire the new check into `Run`**

In `internal/doctor/doctor.go`, immediately after `r.add(checkScipPython(opts.Timeout))` (line ~131), add:

```go
	r.add(checkIndexerImpact(opts.ScryHome, r.Checks))
```

Order matters: it reads the indexer checks that ran before it. Placing it earlier yields an empty `missing` set and a silent false pass.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go build ./... && go test ./internal/doctor/ -v`
Expected: PASS.

Then the full suite: `go test ./...`
Expected: PASS.

- [ ] **Step 7: Verify against the real machine**

```bash
go build -o /tmp/scry-verify ./cmd/scry
/tmp/scry-verify doctor 2>&1 | grep -A2 "indexer impact"
```

Expected: with `scip-python` absent, either a warning naming the affected repo count, or "missing indexers affect no indexed repo" if no repo has yet been rebuilt with the new manifest format. Both are correct; the count grows as repos are reindexed.

- [ ] **Step 8: Update the README**

In `README.md`, under "Known limitations", replace the bullet list entry about schema/init with an added entry (keep the existing bullets):

```markdown
- **Incidental languages are skipped, not indexed.** A language is indexed only if it has a root-level marker file (`composer.json`, `go.mod`, `package.json`, `pyproject.toml`, …) or holds ≥10% of source files. A Laravel app with a handful of stray `.py` scripts will not run `scip-python`, and will not be reported as degraded for it. `scry doctor` lists skipped languages per repo.
- **`scry status` reports per-indexer outcomes.** Each repo entry carries an `indexers` array recording `ok` / `missing` / `failed` / `skipped` per language, with the remedy for a missing indexer.
```

- [ ] **Step 9: Commit**

```bash
git add internal/doctor/ README.md
git commit -m "feat(doctor): name the failing indexer and count affected repos"
```

---

### Task 6: Reindex and confirm the real-world outcome

The whole point of the work. No new code — this validates the change against the 17 repos that motivated it.

**Files:** none modified.

- [ ] **Step 1: Install the new binary and restart the daemon**

```bash
go build -o ~/go/bin/scry ./cmd/scry
scry stop
scry status >/dev/null   # auto-spawns the daemon
```

- [ ] **Step 2: Reindex a repo that was `partial` for the Python reason**

```bash
scry init /Users/jeff/workspace/childscribe-beta-r4 2>&1 | tail -20
```

Expected: no `status=partial` line on stderr. The `init` result JSON reports `"status":"ready"` with an `indexers` array containing a `php` entry `ok`, a `typescript` entry `ok`, and a `python` entry `skipped`/`incidental` with `file_count: 37`.

If it still reports `partial`, read the per-indexer errors now printed on stderr — that is the second, previously invisible cause, and it is now diagnosable.

- [ ] **Step 3: Reindex a repo that was `partial` for an unknown reason**

```bash
scry init /Users/jeff/workspace/idea-planning 2>&1 | tail -20
```

This repo is `typescript,go` with no Python, so the Python fix cannot explain it. Whatever prints here is new information — record it. Fixing it is **out of scope for this plan**; open it as follow-up work.

- [ ] **Step 4: Confirm doctor connects the dots**

```bash
scry doctor 2>&1 | grep -B1 -A3 "indexer impact"
scry status | python3 -c 'import json,sys; d=json.load(sys.stdin); print(sum(1 for r in d["repos"] if r["status"]=="partial"), "partial of", len(d["repos"]))'
```

Expected: the partial count is unchanged for repos not yet reindexed (they keep their legacy manifests) and drops for each one rebuilt. This is correct behavior — no forced global reindex.

- [ ] **Step 5: Commit nothing, report findings**

There is no code change in this task. Report: which repos flipped to `ready`, and what the still-`partial` repos now say their actual failure is.

---

## Self-Review

**Spec coverage:**

| Spec section | Task |
|---|---|
| §1 Data model — `IndexerResult`, additive manifest field | 1 (type), 3 (manifest field + round-trip test) |
| §2 Classification via existing sentinels, table-driven loop | 1 (`classify`), 3 (`buildResults`) |
| §3 Two-tier detection, marker files, `Languages` stays primary-only | 2 |
| §4 Status derivation, `broken` vestigial | 1 (`deriveStatus`) |
| §5 Surfacing — status RPC, doctor `repo.current`, `indexers.impact`, MCP inherits | 4, 5 |
| Testing — classification, tiering, derivation, round-trip, integration, doctor | 1, 2, 3, 5 |
| Verification — reindex childscribe, re-run doctor on idea-planning | 6 |
| Files touched — README, DECISIONS | 2 (DECISIONS), 5 (README) |

No gaps.

**Deviations from the spec, deliberate:**

- The spec's §5 describes a `scry status --pretty` renderer showing "a line per non-ok indexer." `cmd/scry/status.go` uses `printJSON`, so `--pretty` is indented JSON and no such renderer exists. Task 4 adds the field to the JSON; human formatting stays in `doctor`, per the "JSON output by default" constraint. No renderer is built.
- The spec's §5 does not mention `InitResult`. Task 4 adds the field there too — it is the immediate-feedback path right after `scry init` and the most useful place to see the outcome.
- The spec's integration test asserts "`scip-python` was never invoked." Task 3 achieves this with an injected `run` func in `buildResults` rather than a real filesystem build, which keeps the test hermetic and fast (no indexer binaries on PATH required in CI).

**Type consistency:** `IndexerResult` field names are identical across Tasks 1, 3, 4, and 5. `TierPrimary`/`TierIncidental` and `IndexerOK`/`IndexerMissing`/`IndexerFailed`/`IndexerSkipped` are referenced unqualified inside `package index` and as `index.*` inside `package doctor`, correctly in each. `detectLanguages` returns `[]DetectedLanguage` in Task 2 and is consumed as such in Task 3. `buildResults(dets, run)` signature matches between its definition (Task 3 Step 4) and its call (Task 3 Step 5) and its tests (Task 3 Step 1).

**Known ordering constraint:** Tasks 2 and 3 share a commit. Task 2 changes `detectLanguages`'s signature and deletes it from `builder.go`, which breaks the build until Task 3 rewrites the call site. This is called out explicitly in Task 2 Step 6. A reviewer gating Task 2 independently would see a red build; gate them together.
