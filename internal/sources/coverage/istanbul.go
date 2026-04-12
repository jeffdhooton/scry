package coverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// parseIstanbul parses Istanbul/c8 coverage-final.json (vitest, jest output).
//
// The format is a map of file paths to coverage data:
//
//	{
//	  "/abs/path/to/file.ts": {
//	    "statementMap": { "0": {"start":{"line":1,"column":0},"end":{"line":1,"column":30}} },
//	    "s": { "0": 1 },
//	    "fnMap": { "0": {"name":"foo","decl":{"start":{"line":5},"end":{"line":5}},"loc":{"start":{"line":5},"end":{"line":10}}} },
//	    "f": { "0": 3 }
//	  }
//	}
func parseIstanbul(path, repoPath string) ([]CoveredRange, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var fileCoverage map[string]istanbulFileCoverage
	if err := json.Unmarshal(data, &fileCoverage); err != nil {
		return nil, err
	}

	var ranges []CoveredRange
	for filePath, cov := range fileCoverage {
		relPath := toRelPath(filePath, repoPath)

		// Use statement-level coverage (most granular).
		for id, loc := range cov.StatementMap {
			count := cov.S[id]
			if loc.Start.Line > 0 && loc.End.Line > 0 {
				ranges = append(ranges, CoveredRange{
					File:    relPath,
					Line:    loc.Start.Line,
					EndLine: loc.End.Line,
					Count:   count,
				})
			}
		}
	}
	return ranges, nil
}

type istanbulFileCoverage struct {
	StatementMap map[string]istanbulLocation `json:"statementMap"`
	S            map[string]int              `json:"s"`
}

type istanbulLocation struct {
	Start istanbulPos `json:"start"`
	End   istanbulPos `json:"end"`
}

type istanbulPos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func toRelPath(absPath, repoPath string) string {
	if rel, err := filepath.Rel(repoPath, absPath); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return absPath
}

func init() {
	registerParser("istanbul", func(path, repoPath string) ([]CoveredRange, error) {
		return parseIstanbul(path, repoPath)
	})
}
