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
