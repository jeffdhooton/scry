# Vue SFC extraction

**Date:** 2026-08-12
**Status:** approved, ready to plan

## Problem

`scip-typescript` only walks `.ts`/`.tsx` files. Every `.vue` Single File Component is
invisible to scry, and the repo is still reported `ready` with TypeScript indexed — so the gap
is silent.

Measured across the 44 indexed repos on this machine: **~4,500 `.vue` files across 19 repos.**
`hoopless_crm` has 551, `advocates` 315, every `childscribe-*` repo ~250. These are Inertia/Vue
+ Laravel apps — the primary stack here — so this is most of the frontend of most of the repos.

Verified directly rather than assumed: the 1.3 MB TypeScript SCIP dump for `hoopless_crm`
contains **zero** `.vue` documents. Every emitted document path ends in `.ts` or `.d.ts`.

The repos are already configured for it. `hoopless_crm/tsconfig.json` lists
`resources/js/**/*.vue` in `include` and maps `"@/*": ["./resources/js/*"]`. TypeScript is set
up to see these files; scip-typescript is what refuses to walk them.

Script-block shapes are near-uniform, which keeps the extractor simple. Across
`hoopless_crm`'s 551 files: 504 `<script setup lang="ts">`, 40 plain `<script setup>`, one file
with two script blocks, zero template-only, zero plain `<script>`.

## Non-goals

- **`<template>` usage sites.** `<Breadcrumbs :x="y" />` is not indexed. Vue SFC requires
  importing a component before using it in the template, so "which files use `Breadcrumbs.vue`"
  is still answered correctly — references land on the import line, not the tag. Indexing tags
  would need a Vue template parser in Go; deferred to a later phase.
- **`{{ expression }}` and `:prop` binding resolution.** Volar territory. Explicitly out.
- **Blade templates.** The other uncovered surface (~231 files across 18 repos), separate work.
- **Any new indexer binary.** This reuses scip-typescript.

## Validation spike

The riskiest assumptions were tested before this design was written, with a fixture at
`src/Button.vue` (exporting `buttonLabel`) and `src/app.ts` importing it:

1. A `Button.vue.ts` sidecar **is** indexed as a document.
2. `app.ts` was present in the shadow **as a symlink**, and scip-typescript reported its path as
   `src/app.ts` — the shadow-relative path, not the symlink target. A symlink farm does not leak
   real paths into document paths.
3. `import { buttonLabel } from './Button.vue'` **resolved to `Button.vue.ts`** and produced a
   real cross-file reference edge. This was the assumption most likely to sink the approach.

## Design

### 1. Flow

A new `internal/sources/vue` package runs before the TypeScript indexer. When a repo contains
`.vue` files, scip-typescript indexes **the shadow tree instead of the repo**:

```
repo has .vue?  ──no──▶  typescript.Index(repoRoot)          [today's behavior, unchanged]
      │yes
      ▼
vue.BuildShadow(repoRoot, <StorageDir>/vue-shadow)
      │
      ▼
typescript.Index(shadowDir) ──▶ scip.ParseWithOpts(...) ──▶ store
```

One run, not two. The shadow contains symlinks to every `.ts` file, so it covers them as well;
indexing both the repo and the shadow would double-index every `.ts` file and double its
reference counts.

The package's exported surface:

```go
// BuildShadow materializes the shadow tree for repoRoot at shadowDir.
// Returns false (with no error, and no directory created) when repoRoot
// contains no .vue file with an extractable script block, in which case the
// caller indexes repoRoot directly.
func BuildShadow(repoRoot, shadowDir string) (built bool, stats Stats, err error)

// Stats reports what the shadow build did, for logging and the manifest.
type Stats struct {
    VueFiles   int // .vue files found
    Sidecars   int // .vue.ts files written
    Skipped    int // .vue files with no extractable script block
    DirLinks   int // directories symlinked wholesale
    FileLinks  int // individual files symlinked
}

// MapPath rewrites a shadow-relative document path to its repo-relative
// equivalent: "resources/js/Foo.vue.ts" -> "resources/js/Foo.vue".
// Paths that do not end in ".vue.ts" are returned unchanged.
func MapPath(p string) string
```

### 2. The shadow tree

Lives at `<layout.StorageDir>/vue-shadow/`, obtained from a new `RepoLayout.vueShadowPath()`
alongside the existing `scipPath()`. Because it hangs off `RepoLayout`, `BuildIntoTemp`
automatically gets its own shadow under the *next* layout and never disturbs the live one.
Removed at the start of every build, matching how the temp Badger directory is handled.

Construction uses **directory-level symlinks wherever possible**: a directory with no `.vue`
file anywhere beneath it becomes a single symlink. Only directories on a path to a `.vue` file
are materialized as real directories with per-file symlinks inside. For `hoopless_crm` that is
four symlinks (`node_modules/`, `app/`, `vendor/`, `storage/`) plus ~660 materialized entries
under `resources/js/`, instead of tens of thousands.

`node_modules` must be symlinked, **not** skipped — module resolution depends on it. This is the
one place the shadow walker deliberately diverges from `detectLanguages`' skip list.

The `.vue` files themselves are **not** linked into the shadow; only their `.vue.ts` sidecars
are. The spike confirmed TypeScript resolves `./Button.vue` to `Button.vue.ts`, and omitting the
original removes any resolution ambiguity.

### 3. Script extraction and line padding

For each `.vue` file, build a `[]string` with one entry per source line, all empty, then copy
each script block's lines into their **original indices** and join with `\n`.

Line and column numbers in the sidecar are therefore identical to the `.vue` file, so SCIP
occurrence ranges need no remapping at all — only the document path changes. Multiple script
blocks fall out of the same mechanism for free.

Given `Button.vue`:

```
1  <script setup lang="ts">
2  export function buttonLabel(): string {
3      return 'click me';
4  }
5  </script>
6
7  <template>...</template>
```

the sidecar `Button.vue.ts` is one blank line, then lines 2–4 verbatim. `buttonLabel` is on
line 2 in both files.

Only blocks whose `lang` attribute is `ts`, `tsx`, `js`, `jsx`, or absent are extracted; custom
blocks such as `<script lang="json">` (i18n) are ignored. A `.vue` file with no usable block
produces no sidecar and is simply absent from the index.

### 4. SCIP path rewriting

`scip.Parse` keeps its current signature. A new options-carrying variant adds two fields:

- `ProjectRootOverride string` — the real repo path, so the stored `project_root` and all stored
  paths are repo paths rather than shadow paths.
- `PathMapper func(string) string` — strips the trailing `.ts` from `*.vue.ts`.

The mapper applies to every path **stored** (`PutFileSymbol`, occurrence records, document
paths). It must **not** apply to `readSourceLines` at `parse.go:117`, which computes enclosing
scopes for the call graph — that read must resolve against the shadow root and the original
`.vue.ts` name, because that is the file the occurrence ranges describe.

SCIP *symbol* strings will still contain `` `Foo.vue.ts` ``. They are opaque internal
identifiers, never displayed in query output, and self-consistent. Rewriting them buys nothing
and risks breaking cross-references if done partially, so they are left alone.

`project_root` is written to meta but read nowhere in the codebase today (verified), so the
override is for correctness rather than to fix a live bug.

### 5. Detection and tiering

`.vue` maps to a new `"vue"` language in `detectLanguages`, with markers `package.json` and
`tsconfig.json`. `indexerFor("vue")` returns `"typescript"`, folding it into the TypeScript
indexer's single `IndexerResult` — reusing the fold machinery from the partial-index
observability work, including the post-fold `primaryShare` re-check.

This matters independently of indexing: a repo with 250 `.vue` and 10 `.ts` files currently
registers TypeScript at a negligible share. Counting `.vue` fixes the tier.

`Manifest.Languages` will list `vue` alongside `typescript` when both are primary, while
`Indexers` shows one folded `typescript` entry. That asymmetry is intended — languages describe
the repo, indexer results describe what ran.

### 6. Errors and limitations

- A shadow-build failure is returned from `typescript.Index`'s call site and classified
  `failed`, so the repo reports `partial` with the reason recorded. No new error machinery.
- A `.vue` file whose script block cannot be parsed is skipped and counted; the count is logged.
  One bad component never fails the build.
- **Known limitation:** a `</script>` inside a string literal truncates a block early. Present
  in zero files across the scanned repos. Documented in the README, not defended against.
- **Known limitation:** `<template>` usage sites are not indexed; references land on the import.

## Testing

Unit, in `internal/sources/vue`:

- extraction: single block; two blocks; no block; `lang="ts"`, `lang="js"`, no `lang`;
  `lang="json"` ignored; unterminated `<script>` skipped without error
- padding: symbol on source line N appears on line N of the sidecar, for a block that does not
  start at line 1 and for a two-block file
- shadow construction: a directory with no `.vue` beneath it is a symlink; a directory with one
  is a real directory; `node_modules` is symlinked even though `detectLanguages` skips it; the
  `.vue` original is absent from the shadow

Unit, in `internal/index`:

- `.vue` counts toward the `vue` language; `indexerFor("vue") == "typescript"`; a 250-`.vue` /
  10-`.ts` repo yields a primary TypeScript indexer entry

Integration, in `internal/sources/vue` (skipped via `t.Skip` when `scip-typescript` is not on
PATH):

- fixture repo with `Button.vue` exporting a function and `app.ts` importing it → build shadow →
  run scip-typescript → parse with the mapper → assert a reference edge exists whose path is
  `src/Button.vue` (not `.vue.ts`, not a shadow path) at the correct line

## Verification

Reindex `hoopless_crm` and confirm the manifest reports a `vue` language, the TypeScript indexer
`ok`, and a document count that grew by roughly 551. Then `scry refs` a component name and
confirm the hits are `.vue` paths at plausible import lines.

## Files touched

- `internal/sources/vue/extract.go` + test — script-block extraction and line padding
- `internal/sources/vue/shadow.go` + test — shadow tree construction
- `internal/sources/vue/vue_test.go` — the scip-typescript integration test
- `internal/sources/scip/parse.go` — options variant with `ProjectRootOverride` and `PathMapper`
- `internal/index/detect.go` — `.vue` extension, `vue` language, markers
- `internal/index/builder.go` — `indexerFor("vue")`, shadow build before the TypeScript indexer,
  mapper passed to the parser
- `README.md` — remove the "Vue SFC not indexed" limitation, add the two new ones
- `docs/DECISIONS.md` — the shadow-tree and line-padding calls
