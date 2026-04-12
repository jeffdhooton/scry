package coverage

import (
	"fmt"
	"os"

	"github.com/jeffdhooton/scry/internal/store"
)

// Index detects coverage files in the repo, parses them, joins against the
// symbol index, and writes coverage records to the store. Returns stats about
// what was indexed. If no coverage files are found, returns zero stats and no
// error — coverage is optional.
func Index(repoPath string, st *store.Store) (*Stats, error) {
	files := detect(repoPath)
	if len(files) == 0 {
		return &Stats{}, nil
	}
	// Build the definition span index once for all coverage files.
	defIdx, err := buildDefIndex(st)
	if err != nil {
		return nil, fmt.Errorf("build def index for coverage: %w", err)
	}

	combined := &Stats{}
	w := st.NewWriter()

	for _, df := range files {
		parser, ok := parsers[df.format]
		if !ok {
			fmt.Fprintf(os.Stderr, "scry: no coverage parser for format %q\n", df.format)
			continue
		}

		ranges, err := parser(df.path, repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scry: parse coverage %s: %v\n", df.path, err)
			continue
		}

		records := joinCoverageToSymbols(ranges, defIdx)
		for _, rec := range records {
			if err := w.PutCoverage(rec); err != nil {
				return nil, fmt.Errorf("write coverage record: %w", err)
			}
		}

		combined.FilesFound++
		combined.RangesParsed += len(ranges)
		combined.SymbolsCovered += len(records)
		if combined.Format == "" {
			combined.Format = df.format
		} else {
			combined.Format += "+" + df.format
		}
	}

	if err := w.Flush(); err != nil {
		return nil, fmt.Errorf("flush coverage writer: %w", err)
	}
	return combined, nil
}
