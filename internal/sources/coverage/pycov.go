package coverage

import (
	"encoding/json"
	"os"
)

// parsePycov parses Python coverage.json (produced by coverage.py json).
//
// Structure:
//
//	{
//	  "files": {
//	    "src/foo.py": {
//	      "executed_lines": [1, 2, 5, 10],
//	      "missing_lines": [3, 4]
//	    }
//	  }
//	}
func parsePycov(path, repoPath string) ([]CoveredRange, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc pycovReport
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var ranges []CoveredRange
	for filePath, fileCov := range doc.Files {
		relPath := toRelPath(filePath, repoPath)
		for _, line := range fileCov.ExecutedLines {
			if line > 0 {
				ranges = append(ranges, CoveredRange{
					File:    relPath,
					Line:    line,
					EndLine: line,
					Count:   1, // coverage.py doesn't report counts in basic mode
				})
			}
		}
	}
	return ranges, nil
}

type pycovReport struct {
	Files map[string]pycovFile `json:"files"`
}

type pycovFile struct {
	ExecutedLines []int `json:"executed_lines"`
	MissingLines  []int `json:"missing_lines"`
}

func init() {
	registerParser("pycov", func(path, repoPath string) ([]CoveredRange, error) {
		return parsePycov(path, repoPath)
	})
}
