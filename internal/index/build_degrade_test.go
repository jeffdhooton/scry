package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	scipbindings "github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"

	"github.com/jeffdhooton/scry/internal/sources/golang"
	"github.com/jeffdhooton/scry/internal/sources/typescript"
)

// These tests exercise a whole build without any indexer binary on the
// machine. Every indexer invocation goes through the injected indexerRunner
// seam, and the "output" it produces is a SCIP protobuf we synthesize here —
// so nothing below needs scip-typescript, scip-go, scip-python, php or npm.

// goTSRepo is a repo where both "go" and "typescript" detect as primary:
// each has a marker file and a source file, so each holds half the files.
func goTSRepo(t *testing.T) string {
	t.Helper()
	return writeRepo(t, map[string]string{
		"go.mod":       "module example.com/x\n\ngo 1.23\n",
		"main.go":      "package main\n\nfunc main() {}\n",
		"package.json": `{"name":"x"}`,
		"app.ts":       "export function thing() {}\n",
	})
}

// writeSCIPDump writes a real, parseable SCIP index to path: one document
// declaring one symbol, with one definition occurrence and one reference
// occurrence. scip.Parse reports Documents=1, Symbols=1, Definitions=1,
// References=1 for it.
func writeSCIPDump(t *testing.T, path, projectRoot, relPath, symbol string) {
	t.Helper()
	idx := &scipbindings.Index{
		Metadata: &scipbindings.Metadata{
			ProjectRoot:          "file://" + projectRoot,
			TextDocumentEncoding: scipbindings.TextEncoding_UTF8,
		},
		Documents: []*scipbindings.Document{{
			RelativePath: relPath,
			Symbols: []*scipbindings.SymbolInformation{{
				Symbol:      symbol,
				DisplayName: "thing",
				Kind:        scipbindings.SymbolInformation_Function,
			}},
			Occurrences: []*scipbindings.Occurrence{
				{
					Symbol:      symbol,
					Range:       []int32{0, 0, 0, 5},
					SymbolRoles: int32(scipbindings.SymbolRole_Definition),
				},
				{
					Symbol: symbol,
					Range:  []int32{2, 0, 2, 5},
				},
			},
		}},
	}
	b, err := proto.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal scip index: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write scip dump: %v", err)
	}
}

// fakeIndexer builds an indexerRunner from a per-language behavior map. A nil
// behavior means "succeed, writing a valid SCIP dump". Recorded invocations
// are appended to invoked.
type indexerBehavior func(out string) error

func fakeIndexer(t *testing.T, repoPath string, behaviors map[string]indexerBehavior, invoked *[]string) indexerRunner {
	t.Helper()
	return func(_ context.Context, language, _, _, out string) error {
		*invoked = append(*invoked, language)
		if b, ok := behaviors[language]; ok && b != nil {
			return b(out)
		}
		writeSCIPDump(t, out, repoPath, language+"/file", "scip go . . `"+language+"`/thing().")
		return nil
	}
}

// failWith returns a behavior that fails without writing any output, the way
// a missing or crashing indexer binary does.
func failWith(err error) indexerBehavior {
	return func(string) error { return err }
}

// writeGarbage returns a behavior that "succeeds" but leaves a dump that
// cannot be parsed — the scip.Parse failure this task exists to contain.
func writeGarbage(t *testing.T) indexerBehavior {
	return func(out string) error {
		if err := os.WriteFile(out, []byte("this is not a SCIP protobuf"), 0o644); err != nil {
			t.Fatalf("write garbage dump: %v", err)
		}
		return nil
	}
}

// resultFor finds one language's result, failing the test if it is absent.
func resultFor(t *testing.T, m *Manifest, language string) IndexerResult {
	t.Helper()
	for _, r := range m.Indexers {
		if r.Language == language {
			return r
		}
	}
	t.Fatalf("no indexer result for %q in %+v", language, m.Indexers)
	return IndexerResult{}
}

func TestBuildAtLayout_ParseFailureCostsOnlyThatLanguage(t *testing.T) {
	// The builder.go:317 bug: scip-go's dump parses fine, scip-typescript's
	// is corrupt, and the whole build used to return an error — throwing away
	// the Go index that was already sitting in the store and leaving the
	// previous manifest in place to lie about it.
	repo := goTSRepo(t)
	scryHome := t.TempDir()
	layout := Layout(scryHome, repo)

	var invoked []string
	run := fakeIndexer(t, repo, map[string]indexerBehavior{
		"typescript": writeGarbage(t),
	}, &invoked)

	m, err := buildAtLayout(context.Background(), scryHome, repo, layout, run)
	if err != nil {
		t.Fatalf("build must not fail because one language's dump is corrupt: %v", err)
	}
	if len(invoked) != 2 {
		t.Errorf("invoked = %v, want both languages", invoked)
	}
	if m.Status != "partial" {
		t.Errorf("Status = %q, want partial", m.Status)
	}

	goRes := resultFor(t, m, "go")
	if goRes.Status != IndexerOK {
		t.Errorf("go status = %q (%s), want ok — its dump parsed", goRes.Status, goRes.Error)
	}
	if goRes.DocumentCount != 1 || goRes.SymbolCount != 1 || goRes.DefinitionCount != 1 || goRes.ReferenceCount != 1 {
		t.Errorf("go counts = %+v, want 1 of each", goRes)
	}

	tsRes := resultFor(t, m, "typescript")
	if tsRes.Status != IndexerFailed {
		t.Errorf("typescript status = %q, want failed", tsRes.Status)
	}
	if !strings.Contains(tsRes.Error, "parse typescript scip") {
		t.Errorf("typescript error = %q, want the wrapped parse error", tsRes.Error)
	}
	if tsRes.DocumentCount != 0 || tsRes.SymbolCount != 0 || tsRes.DefinitionCount != 0 || tsRes.ReferenceCount != 0 {
		t.Errorf("failed language must carry zero counts, got %+v", tsRes)
	}
	// A parse failure means the indexer is installed and ran, so it must not
	// inherit the "go install scip-typescript" remedy — it needs its own.
	if tsRes.Remedy != parseFailureRemedy {
		t.Errorf("typescript remedy = %q, want the parse-failure remedy", tsRes.Remedy)
	}

	// The surviving language's records are aggregated and the corrupt one's
	// are not.
	if m.Stats.Documents != 1 || m.Stats.Symbols != 1 {
		t.Errorf("Stats = %+v, want only the go dump's records", m.Stats)
	}

	// And the manifest is on disk, not just in the return value.
	onDisk, err := LoadManifest(layout)
	if err != nil {
		t.Fatalf("manifest must be written when a language fails: %v", err)
	}
	if onDisk.Status != "partial" || len(onDisk.Indexers) != 2 {
		t.Errorf("on-disk manifest = %+v, want partial with 2 indexer results", onDisk)
	}
}

func TestBuildAtLayout_EveryIndexerFailingStillWritesAManifest(t *testing.T) {
	// The reproduction: no indexer installed, `scry init` returns
	// "rpc error -32603" and writes nothing, so there is no artifact telling
	// the operator what to install.
	repo := goTSRepo(t)
	scryHome := t.TempDir()
	layout := Layout(scryHome, repo)

	var invoked []string
	run := fakeIndexer(t, repo, map[string]indexerBehavior{
		"go":         failWith(fmt.Errorf("scip-go: %w", golang.ErrIndexerNotFound)),
		"typescript": failWith(fmt.Errorf("scip-typescript: %w", typescript.ErrIndexerNotFound)),
	}, &invoked)

	m, err := buildAtLayout(context.Background(), scryHome, repo, layout, run)
	if err != nil {
		t.Fatalf("a build where every indexer is missing must still produce a manifest, got error: %v", err)
	}
	if m.Status != "partial" {
		t.Errorf("Status = %q, want partial", m.Status)
	}
	if m.Stats.Documents != 0 || m.Stats.Symbols != 0 {
		t.Errorf("Stats = %+v, want zero — nothing ingested", m.Stats)
	}
	for _, lang := range []string{"go", "typescript"} {
		r := resultFor(t, m, lang)
		if r.Status != IndexerMissing {
			t.Errorf("%s status = %q, want missing", lang, r.Status)
		}
		if r.Error == "" {
			t.Errorf("%s must record why it failed", lang)
		}
		if r.Remedy == "" {
			t.Errorf("%s must record how to fix it — that is the whole point of writing the manifest", lang)
		}
		if r.SymbolCount != 0 {
			t.Errorf("%s never ran, so its counts must be zero, got %+v", lang, r)
		}
	}

	if _, err := LoadManifest(layout); err != nil {
		t.Fatalf("manifest must exist on disk after a total failure: %v", err)
	}
}

func TestBuildAtLayout_AllLanguagesOKIsReadyWithPerLanguageCounts(t *testing.T) {
	repo := goTSRepo(t)
	scryHome := t.TempDir()
	layout := Layout(scryHome, repo)

	var invoked []string
	run := fakeIndexer(t, repo, nil, &invoked)

	m, err := buildAtLayout(context.Background(), scryHome, repo, layout, run)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if m.Status != "ready" {
		t.Errorf("Status = %q, want ready", m.Status)
	}
	// Per-language counts must sum to the aggregate, so a consumer can trust
	// either view.
	var docs, syms, defs, refs int
	for _, r := range m.Indexers {
		docs += r.DocumentCount
		syms += r.SymbolCount
		defs += r.DefinitionCount
		refs += r.ReferenceCount
	}
	if docs != m.Stats.Documents || syms != m.Stats.Symbols ||
		defs != m.Stats.Definitions || refs != m.Stats.References {
		t.Errorf("per-language counts (%d/%d/%d/%d) do not sum to Stats %+v",
			docs, syms, defs, refs, m.Stats)
	}
	if docs != 2 {
		t.Errorf("DocumentCount total = %d, want 2 (one per language)", docs)
	}
}

func TestBuildAtLayout_AbortsOnlyWhenTheOutcomeIsUntrustworthy(t *testing.T) {
	// The complement of every other test in this file. A failing *language*
	// degrades; a failure that leaves us unable to say what the store holds
	// aborts with no manifest, because a manifest we can't stand behind is
	// worse than none. Each case below sabotages one such point.
	//
	// Note what "no manifest" protects: the previous build's manifest stays
	// on disk untouched, so a reader sees the last outcome we could vouch
	// for rather than a fresh one we couldn't.
	cases := []struct {
		name string
		// sabotage runs after Layout but before the build, and breaks one
		// step of the trustworthy path.
		sabotage func(t *testing.T, layout RepoLayout)
	}{
		{
			// A regular file where the storage dir belongs: MkdirAll fails
			// before anything at all has run.
			name: "storage dir uncreatable",
			sabotage: func(t *testing.T, layout RepoLayout) {
				if err := os.MkdirAll(filepath.Dir(layout.StorageDir), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(layout.StorageDir, []byte("not a directory"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			// A regular file where the BadgerDB dir belongs: store.Open
			// fails after detection but before any ingest.
			name: "store un-openable",
			sabotage: func(t *testing.T, layout RepoLayout) {
				if err := os.MkdirAll(layout.StorageDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(layout.BadgerDir, []byte("not a directory"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			// A directory where the manifest file belongs. This is the case
			// that matters most: every indexer ran, every dump ingested, and
			// we still must not report success — the build did real work
			// that no reader will ever be able to see.
			name: "manifest unwritable",
			sabotage: func(t *testing.T, layout RepoLayout) {
				if err := os.MkdirAll(layout.ManifestPath, 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := goTSRepo(t)
			scryHome := t.TempDir()
			layout := Layout(scryHome, repo)
			tc.sabotage(t, layout)

			var invoked []string
			run := fakeIndexer(t, repo, nil, &invoked)

			if _, err := buildAtLayout(context.Background(), scryHome, repo, layout, run); err == nil {
				t.Fatal("an untrustworthy outcome must abort the build with an error")
			}
			if _, err := LoadManifest(layout); err == nil {
				t.Error("no manifest may be written when we cannot say what the store contains")
			}
		})
	}
}

func TestBuildResults_PerLanguageOutcomes(t *testing.T) {
	// The decision logic in isolation: which languages get invoked, what
	// status each one lands on, and whether one language's error leaks into
	// another's result.
	goAndTS := []DetectedLanguage{
		{Language: "go", Tier: TierPrimary, FileCount: 50, Share: 0.5, Marker: "go.mod"},
		{Language: "typescript", Tier: TierPrimary, FileCount: 50, Share: 0.5, Marker: "package.json"},
	}

	tests := []struct {
		name        string
		dets        []DetectedLanguage
		errs        map[string]error
		wantInvoked []string
		wantStatus  map[string]string
		wantRemedy  map[string]bool
		wantDerived string
	}{
		{
			name:        "every primary succeeds",
			dets:        goAndTS,
			wantInvoked: []string{"go", "typescript"},
			wantStatus:  map[string]string{"go": IndexerOK, "typescript": IndexerOK},
			wantDerived: "ready",
		},
		{
			name: "one language failing leaves the other untouched",
			dets: goAndTS,
			errs: map[string]error{"go": fmt.Errorf("exit status 1: scip-go panicked")},
			// Both are still invoked: one indexer's failure must never stop
			// the next one from running.
			wantInvoked: []string{"go", "typescript"},
			wantStatus:  map[string]string{"go": IndexerFailed, "typescript": IndexerOK},
			wantRemedy:  map[string]bool{"go": false},
			wantDerived: "partial",
		},
		{
			name:        "one language missing carries a remedy, the other still runs",
			dets:        goAndTS,
			errs:        map[string]error{"typescript": fmt.Errorf("run: %w", typescript.ErrIndexerNotFound)},
			wantInvoked: []string{"go", "typescript"},
			wantStatus:  map[string]string{"go": IndexerOK, "typescript": IndexerMissing},
			wantRemedy:  map[string]bool{"typescript": true},
			wantDerived: "partial",
		},
		{
			name: "every primary fails",
			dets: goAndTS,
			errs: map[string]error{
				"go":         fmt.Errorf("scip-go: %w", golang.ErrIndexerNotFound),
				"typescript": fmt.Errorf("scip-typescript: %w", typescript.ErrIndexerNotFound),
			},
			wantInvoked: []string{"go", "typescript"},
			wantStatus:  map[string]string{"go": IndexerMissing, "typescript": IndexerMissing},
			wantRemedy:  map[string]bool{"go": true, "typescript": true},
			wantDerived: "partial",
		},
		{
			name: "an incidental language is never invoked and never degrades",
			dets: []DetectedLanguage{
				{Language: "go", Tier: TierPrimary, FileCount: 99, Share: 0.99, Marker: "go.mod"},
				{Language: "python", Tier: TierIncidental, FileCount: 1, Share: 0.01},
			},
			wantInvoked: []string{"go"},
			wantStatus:  map[string]string{"go": IndexerOK, "python": IndexerSkipped},
			wantDerived: "ready",
		},
		{
			name: "a failing incidental language does not degrade the repo",
			dets: []DetectedLanguage{
				{Language: "go", Tier: TierPrimary, FileCount: 99, Share: 0.99, Marker: "go.mod"},
				{Language: "python", Tier: TierIncidental, FileCount: 1, Share: 0.01},
			},
			errs:        map[string]error{"python": fmt.Errorf("boom")},
			wantInvoked: []string{"go"},
			wantStatus:  map[string]string{"go": IndexerOK, "python": IndexerSkipped},
			wantDerived: "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var invoked []string
			results := buildResults(tt.dets, func(language string) error {
				invoked = append(invoked, language)
				return tt.errs[language]
			})

			if strings.Join(invoked, ",") != strings.Join(tt.wantInvoked, ",") {
				t.Errorf("invoked = %v, want %v", invoked, tt.wantInvoked)
			}
			byLang := map[string]IndexerResult{}
			for _, r := range results {
				byLang[r.Language] = r
			}
			for lang, want := range tt.wantStatus {
				got, ok := byLang[lang]
				if !ok {
					t.Fatalf("no result recorded for %q", lang)
				}
				if got.Status != want {
					t.Errorf("%s status = %q, want %q", lang, got.Status, want)
				}
				if want == IndexerOK && got.Error != "" {
					t.Errorf("%s succeeded but carries error %q — another language's failure leaked", lang, got.Error)
				}
			}
			for lang, wantRemedy := range tt.wantRemedy {
				if hasRemedy := byLang[lang].Remedy != ""; hasRemedy != wantRemedy {
					t.Errorf("%s remedy present = %v, want %v", lang, hasRemedy, wantRemedy)
				}
			}
			// buildResults never fills ingest counts; only a real parse does.
			for _, r := range results {
				if r.DocumentCount != 0 || r.SymbolCount != 0 || r.DefinitionCount != 0 || r.ReferenceCount != 0 {
					t.Errorf("%s carries ingest counts before any parse ran: %+v", r.Language, r)
				}
			}
			if got := deriveStatus(results); got != tt.wantDerived {
				t.Errorf("deriveStatus = %q, want %q", got, tt.wantDerived)
			}
		})
	}
}
