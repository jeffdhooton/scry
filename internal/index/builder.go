// Package index orchestrates a full indexing pass for one repo:
//
//  1. resolve the repo's per-user storage directory under ~/.scry/repos/<hash>
//  2. detect languages present in the repo
//  3. run the appropriate language indexer (P0: TypeScript only)
//  4. parse the resulting .scip file into the BadgerDB store
//  5. write the manifest
//
// The result is a "warm" repo: queries against the store return correct
// results immediately.
package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gitindex "github.com/jeffdhooton/scry/internal/git/index"
	"github.com/jeffdhooton/scry/internal/sources/coverage"
	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/php"
	"github.com/jeffdhooton/scry/internal/sources/python"
	"github.com/jeffdhooton/scry/internal/sources/scip"
	"github.com/jeffdhooton/scry/internal/sources/typescript"
	"github.com/jeffdhooton/scry/internal/store"
)

// Manifest is the per-repo metadata file written alongside the BadgerDB index.
type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	RepoPath      string    `json:"repo_path"`
	Languages     []string  `json:"languages"`
	IndexedAt     time.Time `json:"indexed_at"`
	Status        string    `json:"status"` // "ready" | "partial"
	// HeadCommit is the repo's git HEAD when this build started. Empty when
	// the repo is not a git checkout (or has no commits yet). Recorded so
	// staleness is decided by comparing commits rather than clocks — see
	// IsStale. Additive: manifests written before this field existed
	// unmarshal with "" and fall back to the mtime comparison.
	HeadCommit    string          `json:"head_commit,omitempty"`
	FailedFiles   int             `json:"failed_files,omitempty"`
	Stats         scip.Stats      `json:"stats"`
	CoverageStats *coverage.Stats `json:"coverage_stats,omitempty"`
	// Indexers records the per-language outcome of this build. Additive:
	// manifests written before this field existed unmarshal with a nil
	// slice, and every consumer must render correctly in that case.
	Indexers []IndexerResult `json:"indexers,omitempty"`
}

// RepoLayout is the resolved on-disk layout for one repo.
type RepoLayout struct {
	RepoPath     string // absolute path to the source repo
	StorageDir   string // ~/.scry/repos/<hash>
	BadgerDir    string // <StorageDir>/index.db
	ManifestPath string // <StorageDir>/manifest.json
}

// scipPath returns the per-language .scip dump location.
func (l RepoLayout) scipPath(language string) string {
	return filepath.Join(l.StorageDir, "scip-"+language+".bin")
}

// Layout resolves where the live index for repoPath should live under scryHome.
// repoPath must be absolute.
func Layout(scryHome, repoPath string) RepoLayout {
	hash := sha256.Sum256([]byte(repoPath))
	short := hex.EncodeToString(hash[:])[:16]
	storage := filepath.Join(scryHome, "repos", short)
	return RepoLayout{
		RepoPath:     repoPath,
		StorageDir:   storage,
		BadgerDir:    filepath.Join(storage, "index.db"),
		ManifestPath: filepath.Join(storage, "manifest.json"),
	}
}

// NextLayout returns a sibling layout pointing at temp BadgerDir + manifest
// paths next to the live ones. Used by BuildIntoTemp so a watcher reindex
// can write a fresh database without touching the live one — the live store
// keeps serving queries throughout. After the build finishes, the caller
// renames the live and next directories to perform an atomic swap.
//
// Per-language scip dumps stay at their existing path (one writer at a
// time, serialized via the watcher's reindexCooldown).
func NextLayout(layout RepoLayout) RepoLayout {
	return RepoLayout{
		RepoPath:     layout.RepoPath,
		StorageDir:   layout.StorageDir,
		BadgerDir:    filepath.Join(layout.StorageDir, "index.db.next"),
		ManifestPath: filepath.Join(layout.StorageDir, "manifest.json.next"),
	}
}

// Build runs a full index pass for the repo at repoPath.
//
// Behavior:
//   - if the storage directory exists with an outdated schema_version, the
//     BadgerDB is wiped and rebuilt
//   - languages are detected by file extension and tiered: a language is
//     "primary" (its indexer runs) if it has a marker file (package.json,
//     go.mod, composer.json, ...) or clears a 10% file-share floor;
//     otherwise it's "incidental" and recorded but never indexed
//   - per-language .scip dumps are kept on disk so future incremental rebuilds
//     don't have to re-parse the world
//   - status is "ready" if every primary language's indexer succeeded,
//     "partial" if a primary indexer is missing or failed (an incidental
//     language's indexer never being invoked does not degrade status)
//   - one language failing never fails the build: every other language still
//     ingests, and a manifest is written recording what each one did
func Build(ctx context.Context, scryHome, repoPath string) (*Manifest, error) {
	abs, err := absRepoPath(repoPath)
	if err != nil {
		return nil, err
	}
	return buildAtLayout(ctx, scryHome, abs, Layout(scryHome, abs), runInstalledIndexer)
}

// BuildIntoTemp runs a full index pass against repoPath but writes the
// resulting BadgerDB and manifest to a temporary side directory next to the
// live index. The live store is untouched throughout, so concurrent queries
// against it keep working. The caller is responsible for atomically swapping
// the temp output into place after this returns successfully — see
// internal/daemon/watch.go for the pattern.
//
// On entry, any leftover temp directory from a previous failed run is
// removed. On any error the temp directory is left on disk so the next call
// (or a manual cleanup) can inspect it.
func BuildIntoTemp(ctx context.Context, scryHome, repoPath string) (*Manifest, RepoLayout, error) {
	abs, err := absRepoPath(repoPath)
	if err != nil {
		return nil, RepoLayout{}, err
	}
	live := Layout(scryHome, abs)
	next := NextLayout(live)
	// Wipe any leftover temp dir from a prior interrupted run. Otherwise
	// store.Open would reuse stale data.
	if err := os.RemoveAll(next.BadgerDir); err != nil {
		return nil, next, fmt.Errorf("remove stale next badger dir: %w", err)
	}
	_ = os.Remove(next.ManifestPath)
	manifest, err := buildAtLayout(ctx, scryHome, abs, next, runInstalledIndexer)
	if err != nil {
		return nil, next, err
	}
	return manifest, next, nil
}

// absRepoPath normalizes a repo path to absolute form.
func absRepoPath(repoPath string) (string, error) {
	if filepath.IsAbs(repoPath) {
		return repoPath, nil
	}
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repo path: %w", err)
	}
	return abs, nil
}

// indexerFor maps a detected language to the indexer that covers it.
// scip-typescript handles both TypeScript and JavaScript, so both fold into
// a single "typescript" invocation.
func indexerFor(language string) string {
	if language == "javascript" {
		return "typescript"
	}
	return language
}

// indexerRunner runs one language's indexer, writing its SCIP dump to out.
// It is the seam buildAtLayout invokes indexers through, injected so a whole
// build can be exercised without scip-typescript, scip-go, scip-python, php
// or npm anywhere on PATH.
type indexerRunner func(ctx context.Context, language, binDir, repoPath, out string) error

// runInstalledIndexer is the production indexerRunner: it shells out to the
// real indexer binary for the language.
func runInstalledIndexer(ctx context.Context, language, binDir, repoPath, out string) error {
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
	return err
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

	// Re-check tier against the folded share: javascript+typescript may each
	// sit below primaryShare individually (or have no marker file) yet clear
	// the bar once combined, since they run through one indexer.
	for _, r := range agg {
		if r.Tier != TierPrimary && r.Share >= primaryShare {
			r.Tier = TierPrimary
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

// buildAtLayout is the shared body of Build and BuildIntoTemp. It runs
// every applicable indexer through run, parses the SCIP output into the
// BadgerDB at layout.BadgerDir, runs PHP post-processors, and writes the
// manifest to layout.ManifestPath. repoPath must already be absolute.
//
// It degrades per language rather than collapsing. A language whose indexer
// is missing, whose indexer fails, or whose SCIP output won't parse costs
// the build that one language: every other language still ingests, the
// failure is recorded on that language's IndexerResult, and a manifest is
// written with status "partial". That holds all the way down to zero
// languages producing output — that build writes a manifest too, recording
// every language's failure and remedy, because "here is what broke and how
// to fix it" beats an error string with no artifact behind it.
//
// Only an outcome that would make the index untrustworthy aborts with an
// error and no manifest: the storage dir won't create, the store won't open,
// Reset fails, a meta write fails, or the manifest won't write. In each of
// those we cannot describe what the store contains, so we must not claim to.
//
// Because the manifest always describes what the store actually holds, a
// build that ingested nothing leaves an empty store behind rather than the
// previous build's data. Callers that swap a freshly built store into a live
// position (BuildIntoTemp + Registry.SwapNext) therefore promote an empty
// index in that case; the manifest says so.
func buildAtLayout(ctx context.Context, scryHome, repoPath string, layout RepoLayout, run indexerRunner) (*Manifest, error) {
	if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	// Resolve HEAD before any indexer runs, not after: a commit landing
	// mid-build must leave the index looking stale (prompting a reindex)
	// rather than falsely current.
	headCommit, err := gitindex.HeadCommit(ctx, repoPath)
	if err != nil {
		// Not fatal — an index without a recorded HEAD still works, it just
		// falls back to the mtime comparison for staleness.
		fmt.Fprintf(os.Stderr, "scry: resolve HEAD for %s: %v\n", repoPath, err)
	}

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
		if err := run(ctx, language, binDir, repoPath, out); err != nil {
			return err
		}
		produced = append(produced, indexed{language, out})
		return nil
	})

	// Index into results by language, so the ingest loop below can downgrade
	// the one language whose dump won't parse without disturbing the others.
	resultIdx := make(map[string]int, len(results))
	for i, r := range results {
		resultIdx[r.Language] = i
	}

	// Open store, wipe stale data, parse each .scip into the same BadgerDB.
	// Everything from here to the ingest loop is the untrustworthy class: if
	// any of it fails we cannot say what the store holds, so we abort.
	st, err := store.Open(layout.BadgerDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	disk, err := st.SchemaVersionOnDisk()
	if err != nil {
		return nil, fmt.Errorf("read schema version: %w", err)
	}
	if disk != 0 && disk != store.SchemaVersion {
		// Loud reindex per docs/DECISIONS.md "Schema evolution".
		fmt.Fprintf(os.Stderr, "scry: schema upgrade %d -> %d, reindexing %s\n", disk, store.SchemaVersion, repoPath)
	}
	// Always reset before re-ingesting so we don't accumulate stale records
	// from a previous build.
	if err := st.Reset(); err != nil {
		return nil, fmt.Errorf("reset store: %w", err)
	}

	if err := st.SetMeta("schema_version", store.SchemaVersion); err != nil {
		return nil, fmt.Errorf("write schema version: %w", err)
	}
	if err := st.SetMeta("repo_path", repoPath); err != nil {
		return nil, fmt.Errorf("write repo path: %w", err)
	}

	combined := scip.Stats{}
	phpProduced := false
	ingested := 0
	for _, p := range produced {
		stats, err := scip.Parse(ctx, p.scipPath, st)
		if err != nil {
			// This language's dump is unusable. That says nothing about the
			// other languages' dumps, so record it against this language
			// alone and keep ingesting. Note that scip.Parse writes through
			// a batched writer it only flushes on success, so a failed parse
			// leaves at most a partial batch behind, never a torn one.
			if i, ok := resultIdx[p.language]; ok {
				results[i].Status = IndexerFailed
				results[i].Error = fmt.Sprintf("parse %s scip: %v", p.language, err)
				// classify returned no remedy for this language because the
				// indexer itself succeeded. A failure with no remedy is half
				// a diagnosis, so give this path its own.
				results[i].Remedy = parseFailureRemedy
			}
			fmt.Fprintf(os.Stderr, "scry: parse %s scip: %v\n", p.language, err)
			continue
		}
		if i, ok := resultIdx[p.language]; ok {
			results[i].DocumentCount = stats.Documents
			results[i].SymbolCount = stats.Symbols
			results[i].DefinitionCount = stats.Definitions
			results[i].ReferenceCount = stats.References
		}
		combined.Documents += stats.Documents
		combined.Symbols += stats.Symbols
		combined.Definitions += stats.Definitions
		combined.References += stats.References
		combined.CallEdges += stats.CallEdges
		combined.Implementations += stats.Implementations
		ingested++
		if p.language == "php" {
			phpProduced = true
		}
	}

	// Laravel non-PSR-4 walker. scip-php skips routes/, config/,
	// migrations/, bootstrap/ entirely; for hoopless_crm that means
	// 522 ::class controller refs in routes/web.php alone are invisible.
	// The walker reads those files, lexes use statements + ::class refs,
	// and emits synthetic occurrences joined to scip-php's symbols.
	if phpProduced {
		ws, err := php.RunWalker(repoPath, st)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: laravel walker: %v\n", err)
		} else if ws.FilesScanned > 0 {
			fmt.Fprintf(os.Stderr, "scry: laravel walker: %d files, %d ::class refs (%d bound)\n",
				ws.FilesScanned, ws.ClassRefsTotal, ws.ClassRefsBound)
			combined.References += ws.ClassRefsTotal
		}
		fs, err := php.RunFacadeResolver(st)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: facade resolver: %v\n", err)
		} else if fs.FacadesScanned > 0 {
			fmt.Fprintf(os.Stderr, "scry: facade resolver: %d facade methods, %d backing edges\n",
				fs.FacadesScanned, fs.EdgesEmitted)
			combined.References += fs.EdgesEmitted
		}
		ss, err := php.RunStringRefWalker(repoPath, st)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: string-ref walker: %v\n", err)
		} else if ss.FilesScanned > 0 {
			fmt.Fprintf(os.Stderr, "scry: string-ref walker: %d files, %d view refs, %d config refs\n",
				ss.FilesScanned, ss.ViewRefsTotal, ss.ConfigRefsTotal)
			combined.References += ss.ViewRefsTotal + ss.ConfigRefsTotal
		}
	}

	// Coverage indexer: detect and parse coverage files (cover.out,
	// coverage-final.json, clover.xml, coverage.json), join against the
	// definition spans we just indexed, and write per-symbol coverage records.
	// This is a no-op if no coverage files are present.
	var covStats *coverage.Stats
	cs, err := coverage.Index(repoPath, st)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scry: coverage indexer: %v\n", err)
	} else if cs.SymbolsCovered > 0 {
		fmt.Fprintf(os.Stderr, "scry: coverage: %d files, %d ranges, %d symbols covered (%s)\n",
			cs.FilesFound, cs.RangesParsed, cs.SymbolsCovered, cs.Format)
		covStats = cs
	}

	status := deriveStatus(results)
	if status != "ready" {
		fmt.Fprintf(os.Stderr, "scry: status=%s\n", status)
		if ingested == 0 {
			fmt.Fprintf(os.Stderr, "scry:   no language produced a usable index for %s — the store is empty\n", repoPath)
		}
		for _, r := range results {
			if r.Status == IndexerMissing || r.Status == IndexerFailed {
				fmt.Fprintf(os.Stderr, "scry:   %s: %s — %s\n", r.Language, r.Status, r.Error)
				if r.Remedy != "" {
					fmt.Fprintf(os.Stderr, "scry:     fix: %s\n", r.Remedy)
				}
			}
		}
	}

	manifest := &Manifest{
		SchemaVersion: store.SchemaVersion,
		RepoPath:      repoPath,
		Languages:     languages,
		IndexedAt:     time.Now().UTC(),
		Status:        status,
		HeadCommit:    headCommit,
		Stats:         combined,
		CoverageStats: covStats,
		Indexers:      results,
	}
	if err := writeManifest(layout.ManifestPath, manifest); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}
	return manifest, nil
}

// LoadManifest reads an existing manifest, or returns an error if missing.
func LoadManifest(layout RepoLayout) (*Manifest, error) {
	b, err := os.ReadFile(layout.ManifestPath)
	if err != nil {
		return nil, err
	}
	m := &Manifest{}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return m, nil
}

func writeManifest(path string, m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
