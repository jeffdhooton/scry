# Vue SFC Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the ~4,500 `.vue` files across 19 indexed repos queryable by indexing their `<script>` blocks.

**Architecture:** A shadow tree of the repo is built under the repo's scry storage directory: directories with no `.vue` beneath them become single symlinks, and each `.vue` file gets a `<name>.vue.ts` sidecar holding only its script blocks, with every other character blanked to spaces so line *and* column numbers are unchanged. scip-typescript indexes the shadow once (it covers `.ts` files too, via symlinks), and the SCIP parser rewrites `*.vue.ts` document paths back to `*.vue` on the way into the store.

**Tech Stack:** Go 1.26.2, stdlib `testing`, scip-typescript (external, already a prerequisite), `google.golang.org/protobuf` (already in the module graph as indirect).

**Source spec:** `docs/superpowers/specs/2026-08-12-vue-sfc-extraction-design.md`

## Global Constraints

- **Go 1.26.2**, module `github.com/jeffdhooton/scry`. No CGO, ever.
- **No new third-party dependencies.** Task 3's test promotes `google.golang.org/protobuf` from `// indirect` to a direct require in `go.mod` — that is a module-graph line change, not a new dependency, and is the only permitted `go.mod` edit.
- **No `store.SchemaVersion` bump.** Nothing about the on-disk record format changes.
- **`scip.Parse`'s existing signature must not change** — `internal/index/builder.go:315` calls it and other future callers will. Add a variant.
- **Line and column numbers in sidecars must exactly match the `.vue` source.** This is what makes remapping unnecessary; a violation silently produces wrong line numbers in every query result.
- **JSON output by default**; human formatting belongs only in `scry doctor`.
- **Verify command:** `go build ./... && go test ./...` — must pass before every commit.

### Validated by spike (do not re-litigate)

1. A `Foo.vue.ts` sidecar is indexed by scip-typescript as a document.
2. A **symlinked** `.ts` file in the shadow is reported at its shadow-relative path, not the symlink target.
3. `import { x } from './Foo.vue'` resolves to `Foo.vue.ts` and produces a real cross-file reference edge.
4. A plainly `proto.Marshal`-ed `scipbindings.Index` round-trips through `visitor.ParseStreaming` — so Task 3 can build a SCIP fixture in-process.

---

### Task 1: Script-block extraction with position preservation

Pure string logic, no filesystem. The heart of the feature.

**Files:**
- Create: `internal/sources/vue/extract.go`
- Create: `internal/sources/vue/extract_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func ExtractScript(src string) (content string, ok bool)`

**The key idea:** rather than copying script lines into a blank buffer, copy the *entire source* and overwrite every byte outside a script body with a space, leaving `\n` alone. Line count, line numbers, and column numbers are then all trivially identical to the original, including for a single-line `<script>foo()</script>`.

- [ ] **Step 1: Write the failing test**

Create `internal/sources/vue/extract_test.go`:

```go
package vue

import (
	"strings"
	"testing"
)

// lineOf returns the 1-indexed line number containing the first occurrence of
// needle, or 0 if absent.
func lineOf(s, needle string) int {
	i := strings.Index(s, needle)
	if i < 0 {
		return 0
	}
	return strings.Count(s[:i], "\n") + 1
}

// colOf returns the 0-indexed column of the first occurrence of needle.
func colOf(s, needle string) int {
	i := strings.Index(s, needle)
	if i < 0 {
		return -1
	}
	lineStart := strings.LastIndex(s[:i], "\n") + 1
	return i - lineStart
}

const setupTS = `<script setup lang="ts">
import Breadcrumbs from '@/components/Breadcrumbs.vue';

export function buttonLabel(): string {
    return 'click me';
}
</script>

<template>
    <Breadcrumbs :items="crumbs" />
</template>
`

func TestExtractScript_PreservesLineAndColumn(t *testing.T) {
	out, ok := ExtractScript(setupTS)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	// buttonLabel is on line 4 col 16 of the source; it must stay there.
	if got, want := lineOf(setupTS, "buttonLabel"), 4; got != want {
		t.Fatalf("fixture drift: buttonLabel is on line %d of the source, want %d", got, want)
	}
	if got := lineOf(out, "buttonLabel"); got != 4 {
		t.Errorf("buttonLabel on line %d of sidecar, want 4", got)
	}
	if got, want := colOf(out, "buttonLabel"), colOf(setupTS, "buttonLabel"); got != want {
		t.Errorf("buttonLabel at column %d of sidecar, want %d", got, want)
	}
	if got, want := strings.Count(out, "\n"), strings.Count(setupTS, "\n"); got != want {
		t.Errorf("sidecar has %d newlines, source has %d — line count must match", got, want)
	}
}

func TestExtractScript_DropsTemplateAndTags(t *testing.T) {
	out, ok := ExtractScript(setupTS)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	for _, banned := range []string{"<script", "</script>", "<template>", "Breadcrumbs :items"} {
		if strings.Contains(out, banned) {
			t.Errorf("sidecar still contains %q", banned)
		}
	}
	// The import inside the script block must survive.
	if !strings.Contains(out, "import Breadcrumbs from '@/components/Breadcrumbs.vue';") {
		t.Error("sidecar dropped the script block's import")
	}
}

func TestExtractScript_TwoBlocks(t *testing.T) {
	src := `<script lang="ts">
export const NAME = 'x';
</script>

<script setup lang="ts">
const local = NAME;
</script>

<template><div/></template>
`
	out, ok := ExtractScript(src)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if got := lineOf(out, "export const NAME"); got != 2 {
		t.Errorf("NAME on line %d, want 2", got)
	}
	if got := lineOf(out, "const local"); got != 6 {
		t.Errorf("local on line %d, want 6", got)
	}
}

func TestExtractScript_SingleLineBlock(t *testing.T) {
	src := `<template><b/></template>
<script setup>const a = 1;</script>
`
	out, ok := ExtractScript(src)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if got := lineOf(out, "const a"); got != 2 {
		t.Errorf("const a on line %d, want 2", got)
	}
	if got, want := colOf(out, "const a"), colOf(src, "const a"); got != want {
		t.Errorf("const a at column %d, want %d", got, want)
	}
}

func TestExtractScript_LangVariants(t *testing.T) {
	tests := []struct {
		name string
		open string
		want bool
	}{
		{"no lang", `<script setup>`, true},
		{"ts", `<script setup lang="ts">`, true},
		{"tsx", `<script lang="tsx">`, true},
		{"js", `<script lang="js">`, true},
		{"jsx", `<script lang="jsx">`, true},
		{"single quotes", `<script lang='ts'>`, true},
		{"json is not indexable", `<script lang="json">`, false},
		{"yaml is not indexable", `<script lang="yaml">`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := tt.open + "\nconst a = 1;\n</script>\n"
			out, ok := ExtractScript(src)
			if ok != tt.want {
				t.Fatalf("ok = %v, want %v", ok, tt.want)
			}
			if tt.want && !strings.Contains(out, "const a = 1;") {
				t.Error("indexable block was not extracted")
			}
		})
	}
}

func TestExtractScript_NoScriptBlock(t *testing.T) {
	if _, ok := ExtractScript("<template><div/></template>\n"); ok {
		t.Error("ok = true for a template-only component, want false")
	}
}

func TestExtractScript_EmptyBlockIsNotExtractable(t *testing.T) {
	if _, ok := ExtractScript("<script setup></script>\n<template><b/></template>\n"); ok {
		t.Error("ok = true for an empty script block, want false")
	}
}

func TestExtractScript_UnterminatedBlock(t *testing.T) {
	// No closing tag: we cannot know where the block ends, so extract nothing
	// rather than guessing. One malformed component must not fail a build.
	if _, ok := ExtractScript("<script setup lang=\"ts\">\nconst a = 1;\n"); ok {
		t.Error("ok = true for an unterminated script block, want false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sources/vue/ -v`
Expected: FAIL — the package does not exist; `undefined: ExtractScript`.

- [ ] **Step 3: Write the implementation**

Create `internal/sources/vue/extract.go`:

```go
// Package vue makes Vue Single File Components indexable by scip-typescript,
// which only walks .ts/.tsx files and silently ignores .vue entirely.
//
// The approach is a shadow tree (see shadow.go) in which every Foo.vue is
// replaced by a Foo.vue.ts sidecar holding just that component's <script>
// blocks. Sidecars preserve the original byte positions exactly, so SCIP
// occurrence ranges are already correct against the .vue file and only the
// document path has to be rewritten on the way into the store.
package vue

import (
	"regexp"
	"strings"
)

// scriptOpenRe matches a <script> opening tag and captures its attributes.
var scriptOpenRe = regexp.MustCompile(`(?i)<script([^>]*)>`)

// langRe pulls lang="..." (or lang='...') out of a tag's attribute string.
var langRe = regexp.MustCompile(`(?i)\blang\s*=\s*["']([^"']*)["']`)

// closeTag is searched case-insensitively; Vue tooling accepts </SCRIPT>.
const closeTag = "</script>"

// indexableLangs are the script-block languages scip-typescript understands.
// A custom block such as <script lang="json"> (vue-i18n) is not code and is
// left out of the sidecar.
var indexableLangs = map[string]bool{
	"":    true,
	"ts":  true,
	"tsx": true,
	"js":  true,
	"jsx": true,
}

// ExtractScript returns the .vue.ts sidecar content for one component source.
//
// Every byte outside an indexable <script> body is replaced by a space, and
// newlines are left untouched. The result therefore has the same number of
// lines as the source, and every extracted token keeps its original line AND
// column. That is what lets the caller skip range remapping entirely: a SCIP
// occurrence recorded against the sidecar is already correct against the .vue
// file, so only the document path needs rewriting.
//
// ok is false when the component has no indexable, non-empty script block, in
// which case no sidecar should be written and the file is simply not indexed.
func ExtractScript(src string) (string, bool) {
	keep := make([]bool, len(src))
	found := false

	pos := 0
	for pos < len(src) {
		loc := scriptOpenRe.FindStringSubmatchIndex(src[pos:])
		if loc == nil {
			break
		}
		tagEnd := pos + loc[1]
		attrs := src[pos+loc[2] : pos+loc[3]]

		rel := strings.Index(strings.ToLower(src[tagEnd:]), closeTag)
		if rel < 0 {
			// Unterminated block. We cannot know where it ends, and guessing
			// risks pulling template markup into TypeScript, so stop here.
			break
		}
		bodyEnd := tagEnd + rel

		if bodyEnd > tagEnd && indexableLangs[scriptLang(attrs)] {
			for i := tagEnd; i < bodyEnd; i++ {
				keep[i] = true
			}
			if strings.TrimSpace(src[tagEnd:bodyEnd]) != "" {
				found = true
			}
		}
		pos = bodyEnd + len(closeTag)
	}

	if !found {
		return "", false
	}

	out := []byte(src)
	for i := range out {
		if !keep[i] && out[i] != '\n' {
			out[i] = ' '
		}
	}
	return string(out), true
}

// scriptLang returns the lowercased lang attribute of a <script> tag, or ""
// when the attribute is absent.
func scriptLang(attrs string) string {
	m := langRe.FindStringSubmatch(attrs)
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sources/vue/ -v`
Expected: PASS, all subtests.

Then: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/sources/vue/
git commit -m "feat(vue): extract SFC script blocks preserving line and column"
```

---

### Task 2: Shadow tree construction

**Files:**
- Create: `internal/sources/vue/shadow.go`
- Create: `internal/sources/vue/shadow_test.go`

**Interfaces:**
- Consumes: `ExtractScript(src string) (string, bool)` from Task 1.
- Produces:
  - `type Stats struct { VueFiles, Sidecars, Skipped, DirLinks, FileLinks int }`
  - `func BuildShadow(repoRoot, shadowDir string) (bool, Stats, error)`
  - `func MapPath(p string) string`

**The cost model that makes this viable:** a directory containing no `.vue` file anywhere beneath it is symlinked as a single entry. `node_modules`, `vendor`, `.git` and friends are excluded from the `.vue` search entirely, so they can never be materialized — they always become one symlink each. On `hoopless_crm` that is a handful of links plus ~660 real entries under `resources/js/`, instead of walking tens of thousands of files.

- [ ] **Step 1: Write the failing test**

Create `internal/sources/vue/shadow_test.go`:

```go
package vue

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree materializes a fixture repo from a path -> contents map.
func writeTree(t *testing.T, files map[string]string) string {
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

// isSymlink reports whether path exists and is a symlink.
func isSymlink(t *testing.T, path string) bool {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

const componentVue = `<script setup lang="ts">
export const label = 'hi';
</script>
<template><b/></template>
`

func TestBuildShadow_NoVueFilesReturnsFalse(t *testing.T) {
	repo := writeTree(t, map[string]string{
		"src/app.ts":   "export const x = 1;\n",
		"package.json": "{}",
	})
	shadow := filepath.Join(t.TempDir(), "vue-shadow")

	built, _, err := BuildShadow(repo, shadow)
	if err != nil {
		t.Fatalf("BuildShadow: %v", err)
	}
	if built {
		t.Error("built = true for a repo with no .vue files, want false")
	}
	if _, err := os.Stat(shadow); !os.IsNotExist(err) {
		t.Error("shadow directory was created despite there being no .vue files")
	}
}

func TestBuildShadow_SidecarReplacesVueFile(t *testing.T) {
	repo := writeTree(t, map[string]string{
		"src/Button.vue": componentVue,
		"src/app.ts":     "import { label } from './Button.vue';\n",
		"package.json":   "{}",
	})
	shadow := filepath.Join(t.TempDir(), "vue-shadow")

	built, stats, err := BuildShadow(repo, shadow)
	if err != nil {
		t.Fatalf("BuildShadow: %v", err)
	}
	if !built {
		t.Fatal("built = false, want true")
	}
	if stats.VueFiles != 1 || stats.Sidecars != 1 {
		t.Errorf("stats = %+v, want VueFiles=1 Sidecars=1", stats)
	}

	sidecar := filepath.Join(shadow, "src", "Button.vue.ts")
	b, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if !strings.Contains(string(b), "export const label") {
		t.Error("sidecar is missing the script body")
	}
	if strings.Contains(string(b), "<template>") {
		t.Error("sidecar contains template markup")
	}

	// The .vue original must NOT be present — TypeScript would then have two
	// candidates for './Button.vue' and resolution becomes ambiguous.
	if _, err := os.Lstat(filepath.Join(shadow, "src", "Button.vue")); !os.IsNotExist(err) {
		t.Error("the .vue original was linked into the shadow; it must not be")
	}

	// app.ts is an ordinary file next to a .vue, so it is symlinked.
	if !isSymlink(t, filepath.Join(shadow, "src", "app.ts")) {
		t.Error("src/app.ts should be a symlink in the shadow")
	}
}

func TestBuildShadow_DirWithoutVueIsSymlinkedWholesale(t *testing.T) {
	repo := writeTree(t, map[string]string{
		"src/Button.vue":   componentVue,
		"lib/deep/a.ts":    "export const a = 1;\n",
		"lib/deep/b.ts":    "export const b = 2;\n",
		"package.json":     "{}",
	})
	shadow := filepath.Join(t.TempDir(), "vue-shadow")

	built, stats, err := BuildShadow(repo, shadow)
	if err != nil {
		t.Fatalf("BuildShadow: %v", err)
	}
	if !built {
		t.Fatal("built = false, want true")
	}
	// lib/ holds no .vue anywhere, so the whole subtree is ONE link.
	if !isSymlink(t, filepath.Join(shadow, "lib")) {
		t.Error("lib/ should be symlinked wholesale — it contains no .vue")
	}
	if stats.DirLinks < 1 {
		t.Errorf("DirLinks = %d, want at least 1", stats.DirLinks)
	}
	// src/ holds a .vue, so it is a real directory.
	if isSymlink(t, filepath.Join(shadow, "src")) {
		t.Error("src/ should be a real directory — it contains a .vue")
	}
	// The link still resolves to the real content.
	b, err := os.ReadFile(filepath.Join(shadow, "lib", "deep", "a.ts"))
	if err != nil {
		t.Fatalf("read through symlinked dir: %v", err)
	}
	if !strings.Contains(string(b), "export const a") {
		t.Error("symlinked directory does not resolve to real content")
	}
}

func TestBuildShadow_NodeModulesIsLinkedNotWalked(t *testing.T) {
	// A .vue inside node_modules must not drag the whole dependency tree into
	// materialization — node_modules is excluded from the search and therefore
	// always ends up as a single symlink. It must still be present, because
	// TypeScript module resolution depends on it.
	repo := writeTree(t, map[string]string{
		"src/Button.vue":                    componentVue,
		"node_modules/pkg/Widget.vue":       componentVue,
		"node_modules/pkg/index.js":         "module.exports = {};\n",
		"package.json":                      "{}",
	})
	shadow := filepath.Join(t.TempDir(), "vue-shadow")

	built, _, err := BuildShadow(repo, shadow)
	if err != nil {
		t.Fatalf("BuildShadow: %v", err)
	}
	if !built {
		t.Fatal("built = false, want true")
	}
	if !isSymlink(t, filepath.Join(shadow, "node_modules")) {
		t.Error("node_modules should be a single symlink")
	}
	if _, err := os.Stat(filepath.Join(shadow, "node_modules", "pkg", "index.js")); err != nil {
		t.Errorf("node_modules content unreachable through the shadow: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(shadow, "node_modules", "pkg", "Widget.vue.ts")); !os.IsNotExist(err) {
		t.Error("a sidecar was written inside node_modules; it must not be walked")
	}
}

func TestBuildShadow_TemplateOnlyComponentIsSkipped(t *testing.T) {
	repo := writeTree(t, map[string]string{
		"src/Button.vue": componentVue,
		"src/Plain.vue":  "<template><i/></template>\n",
		"package.json":   "{}",
	})
	shadow := filepath.Join(t.TempDir(), "vue-shadow")

	_, stats, err := BuildShadow(repo, shadow)
	if err != nil {
		t.Fatalf("BuildShadow: %v", err)
	}
	if stats.VueFiles != 2 {
		t.Errorf("VueFiles = %d, want 2", stats.VueFiles)
	}
	if stats.Sidecars != 1 {
		t.Errorf("Sidecars = %d, want 1", stats.Sidecars)
	}
	if stats.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", stats.Skipped)
	}
	if _, err := os.Lstat(filepath.Join(shadow, "src", "Plain.vue.ts")); !os.IsNotExist(err) {
		t.Error("a sidecar was written for a component with no script block")
	}
}

func TestBuildShadow_WipesStaleShadow(t *testing.T) {
	repo := writeTree(t, map[string]string{
		"src/Button.vue": componentVue,
		"package.json":   "{}",
	})
	shadow := filepath.Join(t.TempDir(), "vue-shadow")
	if err := os.MkdirAll(filepath.Join(shadow, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stale := filepath.Join(shadow, "src", "Deleted.vue.ts")
	if err := os.WriteFile(stale, []byte("export const gone = 1;\n"), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	if _, _, err := BuildShadow(repo, shadow); err != nil {
		t.Fatalf("BuildShadow: %v", err)
	}
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Error("a sidecar from a previous build survived; the shadow must be wiped first")
	}
}

func TestMapPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"resources/js/Foo.vue.ts", "resources/js/Foo.vue"},
		{"src/app.ts", "src/app.ts"},
		{"src/types/index.d.ts", "src/types/index.d.ts"},
		{"weird.vue.ts.ts", "weird.vue.ts"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := MapPath(tt.in); got != tt.want {
			t.Errorf("MapPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sources/vue/ -run 'TestBuildShadow|TestMapPath' -v`
Expected: FAIL — `undefined: BuildShadow`, `undefined: Stats`, `undefined: MapPath`.

- [ ] **Step 3: Write the implementation**

Create `internal/sources/vue/shadow.go`:

```go
package vue

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// sidecarSuffix is appended to a .vue path to name its TypeScript sidecar.
// ".vue" + ".ts" is deliberate: TypeScript resolving `import './Foo.vue'`
// tries `Foo.vue.ts`, so imports written against the real component resolve
// to the sidecar with no path rewriting in the source.
const sidecarSuffix = ".ts"

// unsearchedDirs are never searched for .vue files. Because a directory is
// only materialized when it contains a .vue, excluding these guarantees they
// are symlinked wholesale — which is both the cheap outcome and the correct
// one: node_modules in particular must be present for TypeScript module
// resolution but must never be walked file by file.
var unsearchedDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"out":          true,
	"target":       true,
	".next":        true,
	".turbo":       true,
	"coverage":     true,
}

// Stats reports what a shadow build did, for logging.
type Stats struct {
	VueFiles  int // .vue files found
	Sidecars  int // .vue.ts sidecars written
	Skipped   int // .vue files with no extractable script block
	DirLinks  int // directories symlinked wholesale
	FileLinks int // individual files symlinked
}

// MapPath rewrites a shadow-relative document path to its repo-relative
// equivalent: "resources/js/Foo.vue.ts" -> "resources/js/Foo.vue". Any other
// path is returned unchanged.
func MapPath(p string) string {
	if strings.HasSuffix(p, ".vue"+sidecarSuffix) {
		return strings.TrimSuffix(p, sidecarSuffix)
	}
	return p
}

// BuildShadow materializes a shadow of repoRoot at shadowDir in which every
// .vue file is replaced by a .vue.ts sidecar holding only its script blocks.
//
// Directories with no .vue file beneath them are symlinked as a single entry,
// so the shadow stays cheap: a Laravel app links node_modules/, vendor/, app/
// and storage/ as one entry each and only materializes its frontend tree.
//
// Returns built=false, creating nothing, when the repo has no .vue file with
// an extractable script block. The caller then indexes repoRoot directly.
//
// Any pre-existing shadowDir is removed first, so a component deleted since
// the last build cannot leave a stale sidecar behind.
func BuildShadow(repoRoot, shadowDir string) (bool, Stats, error) {
	var stats Stats

	hasVue, err := collectVueDirs(repoRoot, &stats)
	if err != nil {
		return false, stats, err
	}
	if len(hasVue) == 0 {
		return false, stats, nil
	}

	if err := os.RemoveAll(shadowDir); err != nil {
		return false, stats, fmt.Errorf("remove stale shadow: %w", err)
	}
	if err := os.MkdirAll(shadowDir, 0o755); err != nil {
		return false, stats, fmt.Errorf("create shadow dir: %w", err)
	}
	if err := materialize(repoRoot, shadowDir, ".", hasVue, &stats); err != nil {
		return false, stats, err
	}
	return true, stats, nil
}

// collectVueDirs walks repoRoot and returns the set of directories (as paths
// relative to repoRoot, with "." for the root) that contain a .vue file at any
// depth. Directories in unsearchedDirs are not descended into.
func collectVueDirs(repoRoot string, stats *Stats) (map[string]bool, error) {
	hasVue := map[string]bool{}
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort: an unreadable subtree is not fatal
		}
		if d.IsDir() {
			if path != repoRoot && unsearchedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".vue") {
			return nil
		}
		stats.VueFiles++
		rel, relErr := filepath.Rel(repoRoot, filepath.Dir(path))
		if relErr != nil {
			return nil
		}
		// Mark this directory and every ancestor up to the root.
		for {
			hasVue[rel] = true
			if rel == "." {
				break
			}
			rel = filepath.Dir(rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan for .vue files: %w", err)
	}
	return hasVue, nil
}

// materialize builds shadowDir/relDir from repoRoot/relDir. Directories in
// hasVue become real directories and are recursed into; everything else is
// symlinked as one entry.
func materialize(repoRoot, shadowDir, relDir string, hasVue map[string]bool, stats *Stats) error {
	entries, err := os.ReadDir(filepath.Join(repoRoot, relDir))
	if err != nil {
		return fmt.Errorf("read dir %s: %w", relDir, err)
	}
	for _, e := range entries {
		rel := filepath.Join(relDir, e.Name())
		src := filepath.Join(repoRoot, rel)
		dst := filepath.Join(shadowDir, rel)

		if e.IsDir() {
			if hasVue[rel] {
				if err := os.MkdirAll(dst, 0o755); err != nil {
					return fmt.Errorf("create shadow dir %s: %w", rel, err)
				}
				if err := materialize(repoRoot, shadowDir, rel, hasVue, stats); err != nil {
					return err
				}
				continue
			}
			if err := os.Symlink(src, dst); err != nil {
				return fmt.Errorf("link dir %s: %w", rel, err)
			}
			stats.DirLinks++
			continue
		}

		if strings.EqualFold(filepath.Ext(e.Name()), ".vue") {
			// Deliberately do NOT link the .vue itself: with both Foo.vue and
			// Foo.vue.ts present, `import './Foo.vue'` has two candidates.
			b, readErr := os.ReadFile(src)
			if readErr != nil {
				stats.Skipped++
				continue
			}
			content, ok := ExtractScript(string(b))
			if !ok {
				stats.Skipped++
				continue
			}
			if err := os.WriteFile(dst+sidecarSuffix, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write sidecar %s: %w", rel, err)
			}
			stats.Sidecars++
			continue
		}

		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("link file %s: %w", rel, err)
		}
		stats.FileLinks++
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sources/vue/ -v`
Expected: PASS, including Task 1's tests.

Then: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/sources/vue/
git commit -m "feat(vue): shadow tree with directory-level symlinks"
```

---

### Task 3: SCIP parse options for path rewriting

**Files:**
- Modify: `internal/sources/scip/parse.go` — `Parse` (line ~31), `processDocument` (line ~111, uses of `GetRelativePath()` at lines 143 and 227)
- Create: `internal/sources/scip/parse_test.go`
- Modify: `go.mod` — promote `google.golang.org/protobuf` from indirect to direct

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `type Opts struct { ProjectRootOverride string; PathMapper func(string) string }`
  - `func ParseWithOpts(ctx context.Context, scipPath string, st *store.Store, opts Opts) (Stats, error)`
  - `func Parse(...)` unchanged in signature, now delegating to `ParseWithOpts` with a zero `Opts`

**The one subtle rule:** `processDocument` uses `d.GetRelativePath()` in exactly three places. Line 117 (`readSourceLines`) must keep using the **unmapped** path against the **SCIP index's own** project root, because that is the file the occurrence ranges describe — the sidecar. Lines 143 (`PutFileSymbol`) and 227 (occurrence `File`) must use the **mapped** path, because those are what queries return. Getting this backwards produces either missing context lines or shadow paths in output.

- [ ] **Step 1: Write the failing test**

Create `internal/sources/scip/parse_test.go`:

```go
package scip

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	scipbindings "github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"

	"github.com/jeffdhooton/scry/internal/store"
)

const fixtureSymbol = "scip-typescript npm demo 1.0.0 src/`Button.vue.ts`/label."

// writeFixture marshals a one-document SCIP index rooted at projectRoot and
// returns the path it was written to.
func writeFixture(t *testing.T, dir, projectRoot string) string {
	t.Helper()
	idx := &scipbindings.Index{
		Metadata: &scipbindings.Metadata{ProjectRoot: "file://" + projectRoot},
		Documents: []*scipbindings.Document{{
			RelativePath: "src/Button.vue.ts",
			Symbols: []*scipbindings.SymbolInformation{{
				Symbol:      fixtureSymbol,
				DisplayName: "label",
			}},
			Occurrences: []*scipbindings.Occurrence{{
				Symbol:      fixtureSymbol,
				Range:       []int32{1, 13, 18},
				SymbolRoles: int32(scipbindings.SymbolRole_Definition),
			}},
		}},
	}
	b, err := proto.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	p := filepath.Join(dir, "index.scip")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return p
}

// openStore opens a scratch store under dir.
func openStore(t *testing.T, dir string) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// collectOccurrences gathers every def and ref occurrence for a symbol. The
// store exposes these as callback iterators, and the record pointer is reused
// between callbacks, so each one is copied out.
func collectOccurrences(t *testing.T, st *store.Store, symbolID string) []store.OccurrenceRecord {
	t.Helper()
	var out []store.OccurrenceRecord
	add := func(o *store.OccurrenceRecord) error {
		out = append(out, *o)
		return nil
	}
	if err := st.IterateDefs(symbolID, add); err != nil {
		t.Fatalf("IterateDefs: %v", err)
	}
	if err := st.IterateRefs(symbolID, add); err != nil {
		t.Fatalf("IterateRefs: %v", err)
	}
	return out
}

// metaString reads a meta key. SetMeta JSON-encodes its value and GetMeta
// returns the raw bytes, so a stored string comes back quoted.
func metaString(t *testing.T, st *store.Store, key string) string {
	t.Helper()
	b, err := st.GetMeta(key)
	if err != nil {
		t.Fatalf("GetMeta(%s): %v", key, err)
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("unmarshal meta %s: %v", key, err)
	}
	return s
}

func TestParseWithOpts_MapsStoredPathsButNotSourceReads(t *testing.T) {
	dir := t.TempDir()

	// The "shadow": holds the sidecar the SCIP ranges actually describe.
	shadow := filepath.Join(dir, "shadow")
	if err := os.MkdirAll(filepath.Join(shadow, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Line 2 (0-indexed 1) is the occurrence's line; cols 13-18 cover "label".
	sidecar := "\nexport const label = 'hi';\n"
	if err := os.WriteFile(filepath.Join(shadow, "src", "Button.vue.ts"), []byte(sidecar), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	repo := filepath.Join(dir, "repo")
	scipPath := writeFixture(t, dir, shadow)
	st := openStore(t, dir)

	stats, err := ParseWithOpts(context.Background(), scipPath, st, Opts{
		ProjectRootOverride: repo,
		PathMapper: func(p string) string {
			if p == "src/Button.vue.ts" {
				return "src/Button.vue"
			}
			return p
		},
	})
	if err != nil {
		t.Fatalf("ParseWithOpts: %v", err)
	}
	if stats.Documents != 1 {
		t.Fatalf("Documents = %d, want 1", stats.Documents)
	}

	// Stored occurrence must carry the MAPPED path.
	occs := collectOccurrences(t, st, fixtureSymbol)
	if len(occs) != 1 {
		t.Fatalf("got %d occurrences, want 1", len(occs))
	}
	if occs[0].File != "src/Button.vue" {
		t.Errorf("stored File = %q, want %q", occs[0].File, "src/Button.vue")
	}

	// The context line proves readSourceLines used the UNMAPPED path against
	// the shadow root — src/Button.vue does not exist anywhere on disk.
	if occs[0].Line != 2 {
		t.Errorf("stored Line = %d, want 2", occs[0].Line)
	}
	if occs[0].Context == "" {
		t.Error("Context is empty — readSourceLines did not find the sidecar")
	}

	if got := metaString(t, st, "project_root"); got != repo {
		t.Errorf("project_root = %q, want the override %q", got, repo)
	}
}

func TestParse_ZeroOptsIsUnchanged(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "Button.vue.ts"),
		[]byte("\nexport const label = 'hi';\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	scipPath := writeFixture(t, dir, root)
	st := openStore(t, dir)

	if _, err := Parse(context.Background(), scipPath, st); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	occs := collectOccurrences(t, st, fixtureSymbol)
	if len(occs) != 1 || occs[0].File != "src/Button.vue.ts" {
		t.Fatalf("occurrences = %+v, want one at src/Button.vue.ts (no mapping)", occs)
	}
	if got := metaString(t, st, "project_root"); got != root {
		t.Errorf("project_root = %q, want the SCIP index's own root %q", got, root)
	}
}
```

The store API used above was verified against `internal/store/store.go` before this plan was written: `IterateDefs(symbolID string, fn func(*OccurrenceRecord) error) error` (line 277), `IterateRefs` (line 272), `GetMeta(key string) ([]byte, error)` (line 120), and `OccurrenceRecord{Symbol, File, Line, Column, EndLine, EndColumn, Context, ContainingSymbol, IsDefinition}` (line 52). There is no `OccurrencesForSymbol`, and `GetMeta` returns JSON-encoded bytes rather than a string — hence the two helpers.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sources/scip/ -v`
Expected: FAIL — `undefined: ParseWithOpts`, `undefined: Opts`.

- [ ] **Step 3: Add `Opts` and `ParseWithOpts`**

In `internal/sources/scip/parse.go`, replace the `Parse` function with:

```go
// Opts customizes how a SCIP index is parsed. The zero value reproduces
// Parse's behavior exactly.
type Opts struct {
	// ProjectRootOverride replaces the project root recorded in the store.
	// Source files are still READ from the root declared inside the SCIP
	// index — only the stored value changes. This exists so an index built
	// against a generated shadow tree records the real repo path. Empty
	// means "use the SCIP index's own root".
	ProjectRootOverride string

	// PathMapper rewrites each document's relative path before it is stored.
	// It is deliberately NOT applied when reading source files off disk: the
	// on-disk file is the one the occurrence ranges describe. Nil means the
	// identity mapping.
	PathMapper func(string) string
}

// Parse reads scipPath, walks every Document in it, and writes the normalized
// records into st through a single batched writer.
func Parse(ctx context.Context, scipPath string, st *store.Store) (Stats, error) {
	return ParseWithOpts(ctx, scipPath, st, Opts{})
}

// ParseWithOpts is Parse with path rewriting. See Opts.
func ParseWithOpts(ctx context.Context, scipPath string, st *store.Store, opts Opts) (Stats, error) {
	f, err := os.Open(scipPath)
	if err != nil {
		return Stats{}, fmt.Errorf("open scip file: %w", err)
	}
	defer f.Close()

	mapPath := opts.PathMapper
	if mapPath == nil {
		mapPath = func(p string) string { return p }
	}

	w := st.NewWriter()
	stats := Stats{}
	var projectRoot string
	// seenSymbols tracks every symbol id we've already PutSymbol'd for in this
	// indexing run. We use it to (a) avoid re-writing duplicate SymbolRecords
	// across documents and (b) decide whether an occurrence-only symbol needs
	// a synthesized SymbolRecord at the end of processing.
	seenSymbols := map[string]bool{}

	visitor := &scipbindings.IndexVisitor{
		VisitMetadata: func(_ context.Context, m *scipbindings.Metadata) error {
			projectRoot = strings.TrimPrefix(m.GetProjectRoot(), "file://")
			stored := projectRoot
			if opts.ProjectRootOverride != "" {
				stored = opts.ProjectRootOverride
			}
			if err := st.SetMeta("project_root", stored); err != nil {
				return fmt.Errorf("set project_root: %w", err)
			}
			return nil
		},
		VisitDocument: func(_ context.Context, d *scipbindings.Document) error {
			return processDocument(d, projectRoot, mapPath, w, &stats, seenSymbols)
		},
	}
	if err := visitor.ParseStreaming(ctx, f); err != nil {
		return stats, fmt.Errorf("parse scip stream: %w", err)
	}
	if err := w.Flush(); err != nil {
		return stats, fmt.Errorf("flush writer: %w", err)
	}
	return stats, nil
}
```

- [ ] **Step 4: Thread the mapper through `processDocument`**

Change the signature at `internal/sources/scip/parse.go:111` to:

```go
func processDocument(d *scipbindings.Document, projectRoot string, mapPath func(string) string, w *store.Writer, stats *Stats, seenSymbols map[string]bool) error {
	stats.Documents++

	// Read source once so we can attach a context line to every occurrence.
	// This uses the UNMAPPED path against the SCIP index's own root: that is
	// the file the occurrence ranges actually describe (for a Vue shadow
	// build, the .vue.ts sidecar). Mapping here would read the wrong file.
	sourceLines := readSourceLines(filepath.Join(projectRoot, d.GetRelativePath()))

	// storedPath is what queries return, so it is the mapped one.
	storedPath := mapPath(d.GetRelativePath())
```

Then replace the two **stored** uses:
- line ~143: `w.PutFileSymbol(d.GetRelativePath(), si.GetSymbol())` → `w.PutFileSymbol(storedPath, si.GetSymbol())`
- line ~227: `File: d.GetRelativePath(),` → `File: storedPath,`

Leave the `readSourceLines` call using `d.GetRelativePath()`.

- [ ] **Step 5: Promote the protobuf dependency**

Run: `go mod tidy`
Expected: `google.golang.org/protobuf` moves out of the `// indirect` block in `go.mod`. No other module is added or removed — verify with `git diff go.mod` and stop if anything else changed.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/sources/scip/ -v`
Expected: PASS.

Then: `go build ./... && go test ./...`
Expected: PASS. `internal/index/builder.go:315` still calls `scip.Parse` with the old signature and must still compile untouched.

- [ ] **Step 7: Commit**

```bash
git add internal/sources/scip/ go.mod go.sum
git commit -m "feat(scip): parse options for project-root override and path mapping"
```

---

### Task 4: Wire the shadow into indexing, detect .vue, document it

**Files:**
- Modify: `internal/index/detect.go` — `languageMarkers` (line ~23), the extension switch in `detectLanguages` (line ~82)
- Modify: `internal/index/builder.go` — `RepoLayout` helper (near line 58), `indexerFor` (line ~160), the indexer closure (line ~250), the parse loop (line ~315)
- Modify: `internal/index/detect_test.go` — add a `.vue` detection case
- Create: `internal/sources/vue/integration_test.go`
- Modify: `README.md:608` — replace the Vue limitation
- Modify: `docs/DECISIONS.md` — append the decision entry

**Interfaces:**
- Consumes: `vue.BuildShadow(repoRoot, shadowDir) (bool, vue.Stats, error)`, `vue.MapPath(string) string` (Task 2); `scip.ParseWithOpts(ctx, path, st, scip.Opts{...})` (Task 3).
- Produces: nothing downstream.

- [ ] **Step 1: Write the failing detection test**

Append to `internal/index/detect_test.go`:

```go
func TestDetectLanguages_VueCountsAsItsOwnLanguage(t *testing.T) {
	// The shape this feature exists for: a Vue-heavy frontend where the .ts
	// files alone would barely register.
	root := writeRepo(t, merge(
		nFiles("resources/js/pages", "p", ".vue", 250),
		nFiles("resources/js", "t", ".ts", 10),
		map[string]string{"package.json": "{}", "tsconfig.json": "{}"},
	))

	dets, err := detectLanguages(root)
	if err != nil {
		t.Fatalf("detectLanguages: %v", err)
	}
	v := find(dets, "vue")
	if v.Language != "vue" {
		t.Fatal("vue was not detected at all")
	}
	if v.FileCount != 250 {
		t.Errorf("vue file count = %d, want 250", v.FileCount)
	}
	if v.Tier != TierPrimary {
		t.Errorf("vue tier = %q, want %q (package.json marker)", v.Tier, TierPrimary)
	}
	if v.Marker != "package.json" && v.Marker != "tsconfig.json" {
		t.Errorf("vue marker = %q, want package.json or tsconfig.json", v.Marker)
	}
}

func TestBuildResults_VueFoldsIntoTypescriptIndexer(t *testing.T) {
	dets := []DetectedLanguage{
		{Language: "vue", Tier: TierPrimary, FileCount: 250, Share: 0.96, Marker: "package.json"},
		{Language: "typescript", Tier: TierIncidental, FileCount: 10, Share: 0.04},
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
	if results[0].Language != "typescript" {
		t.Errorf("folded language = %q, want typescript", results[0].Language)
	}
	if results[0].FileCount != 260 {
		t.Errorf("FileCount = %d, want 260 (vue + ts summed)", results[0].FileCount)
	}
}
```

Note: `writeRepo`, `merge`, `nFiles` and `find` are existing helpers in `detect_test.go`; `buildResults` and `DetectedLanguage` are in `builder.go`/`detect.go`. Read the file before appending to confirm the helper names.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/index/ -run 'TestDetectLanguages_Vue|TestBuildResults_Vue' -v`
Expected: FAIL — vue is not detected (`v.Language` is empty), and `buildResults` invokes `"vue"` rather than folding.

- [ ] **Step 3: Add `.vue` to detection**

In `internal/index/detect.go`, add to `languageMarkers`:

```go
	"vue":        {"package.json", "tsconfig.json"},
```

and add a case to the extension switch inside `detectLanguages`:

```go
		case ".vue":
			counts["vue"]++
			total++
```

In `internal/index/builder.go`, extend `indexerFor`:

```go
// indexerFor maps a detected language to the indexer that covers it.
// scip-typescript handles TypeScript, JavaScript, and — via the generated
// shadow tree in internal/sources/vue — the script blocks of .vue files, so
// all three fold into a single "typescript" invocation.
func indexerFor(language string) string {
	switch language {
	case "javascript", "vue":
		return "typescript"
	}
	return language
}
```

- [ ] **Step 4: Run the detection tests**

Run: `go test ./internal/index/ -run 'TestDetectLanguages|TestBuildResults' -v`
Expected: PASS, including the pre-existing cases.

- [ ] **Step 5: Add the shadow layout helper**

In `internal/index/builder.go`, next to `scipPath` (line ~58):

```go
// vueShadowPath is where the Vue shadow tree for this layout lives. Hanging
// it off RepoLayout means BuildIntoTemp gets its own shadow under the next
// layout and never disturbs the live one.
func (l RepoLayout) vueShadowPath() string {
	return filepath.Join(l.StorageDir, "vue-shadow")
}
```

- [ ] **Step 6: Wire the shadow into the indexer closure**

In `buildAtLayout`, add before the `results := buildResults(...)` call:

```go
	// Vue SFCs are invisible to scip-typescript, which only walks .ts/.tsx.
	// When the repo has any, build a shadow tree whose .vue files are replaced
	// by .vue.ts sidecars and index that instead — it symlinks the .ts files
	// too, so one pass covers both. Indexing the repo as well would
	// double-count every .ts reference.
	tsRoot := repoPath
	vueShadow := false
	if built, vs, err := vue.BuildShadow(repoPath, layout.vueShadowPath()); err != nil {
		fmt.Fprintf(os.Stderr, "scry: vue shadow: %v (indexing without Vue support)\n", err)
	} else if built {
		tsRoot = layout.vueShadowPath()
		vueShadow = true
		fmt.Fprintf(os.Stderr, "scry: vue: %d components, %d sidecars, %d skipped, %d dir links\n",
			vs.VueFiles, vs.Sidecars, vs.Skipped, vs.DirLinks)
	}
```

Then change the `typescript` case in the closure from `repoPath` to `tsRoot`:

```go
		case "typescript":
			_, err = typescript.Index(ctx, tsRoot, out)
```

Add `"github.com/jeffdhooton/scry/internal/sources/vue"` to the imports.

Note the deliberate choice: a shadow-build **error** is logged and indexing continues against the real repo, rather than failing the TypeScript indexer. Vue support is an enhancement; losing it should not turn a working index into a `partial` one.

- [ ] **Step 7: Pass the mapper to the parser**

In the parse loop (line ~315), replace the `scip.Parse` call:

```go
	for _, p := range produced {
		opts := scip.Opts{}
		if vueShadow && p.language == "typescript" {
			// The TypeScript index was built against the shadow: rewrite
			// Foo.vue.ts back to Foo.vue and record the real repo as the root.
			opts = scip.Opts{ProjectRootOverride: repoPath, PathMapper: vue.MapPath}
		}
		stats, err := scip.ParseWithOpts(ctx, p.scipPath, st, opts)
		if err != nil {
			return nil, fmt.Errorf("parse %s scip: %w", p.language, err)
		}
```

Leave the rest of the loop body unchanged.

- [ ] **Step 8: Write the end-to-end integration test**

Create `internal/sources/vue/integration_test.go`:

```go
package vue_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jeffdhooton/scry/internal/sources/scip"
	"github.com/jeffdhooton/scry/internal/sources/typescript"
	"github.com/jeffdhooton/scry/internal/sources/vue"
	"github.com/jeffdhooton/scry/internal/store"
)

// TestShadowIndexEndToEnd is the proof the whole approach works: a real
// scip-typescript run over a shadow tree must produce a reference edge that
// lands on the .vue path, not the sidecar and not a shadow path.
func TestShadowIndexEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("scip-typescript"); err != nil {
		t.Skip("scip-typescript not on PATH")
	}

	repo := t.TempDir()
	files := map[string]string{
		"src/Button.vue": "<script setup lang=\"ts\">\nexport function buttonLabel(): string {\n    return 'click me';\n}\n</script>\n\n<template><button/></template>\n",
		"src/app.ts":     "import { buttonLabel } from './Button.vue';\nconsole.log(buttonLabel());\n",
		"package.json":   `{"name":"vuefixture","version":"1.0.0"}`,
		"tsconfig.json":  `{"compilerOptions":{"target":"ESNext","module":"ESNext","moduleResolution":"bundler","allowJs":true,"skipLibCheck":true},"include":["src/**/*.ts","src/**/*.vue"]}`,
	}
	for rel, content := range files {
		full := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	work := t.TempDir()
	shadow := filepath.Join(work, "vue-shadow")
	built, stats, err := vue.BuildShadow(repo, shadow)
	if err != nil {
		t.Fatalf("BuildShadow: %v", err)
	}
	if !built || stats.Sidecars != 1 {
		t.Fatalf("built=%v stats=%+v, want built with 1 sidecar", built, stats)
	}

	scipPath := filepath.Join(work, "index.scip")
	if _, err := typescript.Index(context.Background(), shadow, scipPath); err != nil {
		t.Fatalf("scip-typescript: %v", err)
	}

	st, err := store.Open(filepath.Join(work, "db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if _, err := scip.ParseWithOpts(context.Background(), scipPath, st, scip.Opts{
		ProjectRootOverride: repo,
		PathMapper:          vue.MapPath,
	}); err != nil {
		t.Fatalf("ParseWithOpts: %v", err)
	}

	// buttonLabel is defined on line 2 of Button.vue and referenced from
	// app.ts. Both must be present, and neither may mention .vue.ts.
	ids, err := st.LookupSymbolsByName("buttonLabel")
	if err != nil {
		t.Fatalf("LookupSymbolsByName: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("buttonLabel was not indexed at all")
	}

	var sawVueDef, sawTSRef bool
	collect := func(o *store.OccurrenceRecord) error {
		switch {
		case o.File == "src/app.ts":
			sawTSRef = true
		case o.File == "src/Button.vue":
			sawVueDef = true
			if o.Line != 2 {
				t.Errorf("definition on line %d of Button.vue, want 2", o.Line)
			}
		}
		if filepath.Base(o.File) == "Button.vue.ts" {
			t.Errorf("an occurrence leaked the sidecar path: %s", o.File)
		}
		return nil
	}
	for _, id := range ids {
		if err := st.IterateDefs(id, collect); err != nil {
			t.Fatalf("IterateDefs: %v", err)
		}
		if err := st.IterateRefs(id, collect); err != nil {
			t.Fatalf("IterateRefs: %v", err)
		}
	}
	if !sawVueDef {
		t.Error("no occurrence recorded against src/Button.vue")
	}
	if !sawTSRef {
		t.Error("no reference recorded from src/app.ts — cross-file resolution failed")
	}
}
```

The accessors used here were verified against `internal/store/store.go`:
`LookupSymbolsByName(name string) ([]string, error)` (line 234), `IterateDefs` (line 277), and
`IterateRefs` (line 272).

- [ ] **Step 9: Run the full suite**

Run: `go build ./... && go test ./... -count=1`
Expected: PASS, including the integration test (scip-typescript is installed on this machine).

- [ ] **Step 10: Update the README**

In `README.md`, replace line 608:

```markdown
- **Vue Single File Components are not indexed.** scip-typescript only walks `.ts`/`.tsx` files.
```

with:

```markdown
- **Vue `<template>` blocks are not indexed.** `<script>` blocks are (via a generated shadow tree), so a component's imports, functions, and types are queryable and `scry refs <Component>` finds every file that imports it — but references land on the import line, not the `<MyComponent />` tag.
- **A `</script>` inside a string literal truncates a Vue script block early.** The extractor is a scanner, not a full parser. No such file exists in any indexed repo.
```

- [ ] **Step 11: Record the decision**

Append to `docs/DECISIONS.md`, matching the file's `## <date> — <Title>` heading convention:

```markdown
## 2026-08-12 — Vue SFC indexing via a shadow tree and blanked sidecars

**Decision:** When a repo contains `.vue` files, build a shadow tree under
`~/.scry/repos/<hash>/vue-shadow/` in which directories with no `.vue` beneath
them are single symlinks and each `Foo.vue` becomes a `Foo.vue.ts` sidecar.
The sidecar is the original file with every byte outside a `<script>` body
replaced by a space. scip-typescript indexes the shadow instead of the repo,
and `scip.ParseWithOpts` rewrites `*.vue.ts` document paths back to `*.vue`.

**Why blanking instead of extracting:** copying script lines into a fresh
buffer requires tracking an offset per block and remapping every SCIP range.
Blanking preserves byte positions exactly, so line and column numbers are
already correct and only the document path changes. Multiple `<script>` blocks
and single-line blocks fall out for free.

**Why a shadow instead of writing sidecars into the repo:** sidecars in the
working tree would show up in `git status`, trigger the file watcher, and
cause HMR rebuilds. Directory-level symlinks keep the shadow cheap — a Laravel
app links `node_modules/`, `vendor/`, `app/` and `storage/` as one entry each.

**Why one indexer pass, not two:** the shadow symlinks `.ts` files, so
indexing both the repo and the shadow would double every TypeScript reference.

**Validated by spike before implementation:** sidecars are indexed; symlinked
`.ts` files report shadow-relative paths rather than symlink targets; and
`import './Foo.vue'` resolves to `Foo.vue.ts`, producing real cross-file
reference edges.

**What would change our minds:** if `<template>` usage sites become the thing
people actually query for, the sidecar approach does not extend to them — that
needs a Vue template compiler and a second synthetic-occurrence pass, closer
to how the Laravel facade resolver works.
```

- [ ] **Step 12: Commit**

```bash
git add internal/index/ internal/sources/vue/ README.md docs/DECISIONS.md
git commit -m "feat(index): index Vue SFC script blocks via a shadow tree"
```

---

### Task 5: Validate against a real repo

No new code. Confirms the feature works on the 551-component repo that motivated it.

**Files:** none modified.

- [ ] **Step 1: Install and restart**

```bash
go build -o ~/go/bin/scry ./cmd/scry
scry stop
scry status >/dev/null
```

- [ ] **Step 2: Reindex hoopless_crm**

```bash
scry init /Users/jeff/Herd/hoopless_crm 2>&1 | tail -20
```

Expected: a `scry: vue: 551 components, ~551 sidecars, ...` line on stderr; `"status":"ready"`; `languages` includes `vue`; the `indexers` array holds one folded `typescript` entry whose `file_count` is roughly 654 (551 vue + 103 ts). Document count should be up several hundred from the pre-change 1482.

- [ ] **Step 3: Query a component**

```bash
scry refs AppSidebarHeader --pretty | head -20
scry defs buttonLabel --pretty 2>/dev/null | head
```

Expected: hits with `.vue` paths. No path anywhere may end in `.vue.ts`, and none may sit under `~/.scry/repos/`. Open one reported `file:line` and confirm the line number actually points at the symbol — this is the end-to-end check that blanking preserved positions.

- [ ] **Step 4: Confirm nothing regressed**

```bash
scry status | python3 -c 'import json,sys; d=json.load(sys.stdin); print(sum(1 for r in d["repos"] if r["status"]=="partial"), "partial of", len(d["repos"]))'
```

Expected: no increase in the partial count versus before this branch.

- [ ] **Step 5: Report, commit nothing**

Report the component count, the document-count delta, and a sample `scry refs` result showing a `.vue` path with a verified-correct line number.

---

## Self-Review

**Spec coverage:**

| Spec section | Task |
|---|---|
| §1 Flow, `BuildShadow`/`Stats`/`MapPath` surface | 2 (surface), 4 (flow wiring) |
| §2 Shadow tree, directory symlinks, node_modules, `.vue` not linked | 2 |
| §2 `vueShadowPath()` on `RepoLayout`, wiped per build | 4 (helper), 2 (wipe) |
| §3 Script extraction, line padding, `lang` filtering, no-block skip | 1 |
| §4 `ProjectRootOverride`, `PathMapper`, `readSourceLines` exemption, symbols untouched | 3 |
| §5 `.vue` → `vue` language, markers, `indexerFor` fold | 4 |
| §6 Shadow failure handling, unparseable component skipped, both limitations | 4 (handling + README) |
| Testing — extraction, padding, shadow construction, detection, integration | 1, 2, 3, 4 |
| Verification — reindex hoopless_crm, `scry refs` a component | 5 |
| Files touched — README, DECISIONS | 4 |

No gaps.

**Deviations from the spec, deliberate:**

- The spec describes building the sidecar by copying script lines into a blank buffer. Task 1 instead blanks non-script bytes to spaces in a copy of the whole source. Same guarantee, strictly stronger: it also preserves **columns** and handles single-line `<script>foo()</script>` blocks, which the line-copy approach drops. Recorded in the DECISIONS entry.
- The spec says a shadow-build failure surfaces as the TypeScript indexer `failed`. Task 4 Step 6 instead logs it and falls back to indexing the real repo. Turning a previously-working index into `partial` because an enhancement failed is the wrong trade — and the branch we just merged exists specifically to stop spurious `partial` states. Called out inline in the task.
- The spec's testing section did not anticipate a unit test for `scip.ParseWithOpts`. Task 3 adds one using an in-process `proto.Marshal`ed fixture, validated by spike. This promotes `google.golang.org/protobuf` from indirect to direct in `go.mod` — permitted explicitly in Global Constraints.

**Type consistency:** `vue.Stats` field names (`VueFiles`, `Sidecars`, `Skipped`, `DirLinks`, `FileLinks`) match between Task 2's definition, its tests, and Task 4's log line. `BuildShadow` returns `(bool, Stats, error)` consistently in Task 2, Task 4 Step 6, and Task 4 Step 8. `MapPath` is used as a bare `func(string) string` value in Task 4 Step 7, matching Task 2's signature. `scip.Opts` field names match between Task 3's definition, Task 4 Step 7, and Task 4 Step 8.

**Store API verified, not guessed:** an early draft of this plan invented `OccurrencesForSymbol` and `SymbolsByName`. Neither exists. The real surface — `LookupSymbolsByName` (store.go:234), the callback iterators `IterateDefs`/`IterateRefs` (store.go:277/272), and `GetMeta` returning JSON-encoded `[]byte` rather than a string (store.go:120) — was read off the source and the test code in Tasks 3 and 4 now uses it directly, with `collectOccurrences` and `metaString` helpers bridging the callback and JSON shapes.
