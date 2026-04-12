package coverage

import (
	"os"
	"path/filepath"
)

type detectedFile struct {
	path   string // absolute path to the coverage file
	format string // parser name: "gocover", "istanbul", "clover", "pycov"
}

// detect searches well-known paths relative to repoPath for coverage files.
// Returns at most one file per format (first match wins within each format).
func detect(repoPath string) []detectedFile {
	candidates := []struct {
		paths  []string
		format string
	}{
		{
			paths:  []string{"cover.out", "coverage.out"},
			format: "gocover",
		},
		{
			paths:  []string{"coverage/coverage-final.json", "coverage-final.json"},
			format: "istanbul",
		},
		{
			paths:  []string{"coverage.xml", "clover.xml", "build/logs/clover.xml"},
			format: "clover",
		},
		{
			paths:  []string{"coverage.json", "htmlcov/coverage.json"},
			format: "pycov",
		},
	}

	var found []detectedFile
	for _, c := range candidates {
		for _, rel := range c.paths {
			abs := filepath.Join(repoPath, rel)
			if _, err := os.Stat(abs); err == nil {
				found = append(found, detectedFile{path: abs, format: c.format})
				break // first match per format
			}
		}
	}
	return found
}
