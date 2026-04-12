// Package coverage detects, parses, and indexes test coverage data from
// multiple formats (Go coverprofile, Istanbul JSON, Clover XML, Python
// coverage.json). Coverage data is joined against scry's existing symbol
// definitions to produce per-symbol coverage records.
package coverage

// CoveredRange is a line range that was executed during the test suite.
// All parsers produce this common type.
type CoveredRange struct {
	File    string // relative path within the repo
	Line    int    // start line (1-based)
	EndLine int    // end line (1-based); same as Line for single-line coverage
	Count   int    // execution count (0 = not covered but instrumented)
}

// Stats summarizes what the coverage indexer found.
type Stats struct {
	Format       string `json:"format"`        // e.g. "gocover", "istanbul", "clover", "pycov"
	FilesFound   int    `json:"files_found"`   // coverage data files detected
	RangesParsed int    `json:"ranges_parsed"` // total CoveredRange entries parsed
	SymbolsCovered int  `json:"symbols_covered"` // symbols with at least one covered line
}
