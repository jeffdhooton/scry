package coverage

import (
	"testing"

	"github.com/jeffdhooton/scry/internal/store"
)

func TestJoinCoverageToSymbols(t *testing.T) {
	defIdx := map[string][]defSpan{
		"pkg/handler.go": {
			{symbolID: "sym:handleRequest", displayName: "handleRequest", line: 10, endLine: 25},
			{symbolID: "sym:handleError", displayName: "handleError", line: 30, endLine: 40},
		},
		"internal/db.go": {
			{symbolID: "sym:query", displayName: "query", line: 5, endLine: 15},
		},
	}

	ranges := []CoveredRange{
		{File: "pkg/handler.go", Line: 12, EndLine: 14, Count: 3},  // inside handleRequest
		{File: "pkg/handler.go", Line: 22, EndLine: 24, Count: 1},  // also inside handleRequest
		{File: "pkg/handler.go", Line: 35, EndLine: 35, Count: 2},  // inside handleError
		{File: "pkg/handler.go", Line: 50, EndLine: 50, Count: 1},  // outside any function
		{File: "internal/db.go", Line: 7, EndLine: 10, Count: 5},   // inside query
		{File: "internal/db.go", Line: 7, EndLine: 10, Count: 0},   // not actually executed
		{File: "missing.go", Line: 1, EndLine: 1, Count: 1},        // file not in index
	}

	records := joinCoverageToSymbols(ranges, defIdx)

	// Should have 3 covered symbols: handleRequest, handleError, query.
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	bySymbol := map[string]*store.CoverageRecord{}
	for _, r := range records {
		bySymbol[r.Symbol] = r
	}

	// handleRequest: two overlapping ranges, counts accumulated (3+1=4).
	hr, ok := bySymbol["sym:handleRequest"]
	if !ok {
		t.Fatal("missing coverage for sym:handleRequest")
	}
	if hr.HitCount != 4 {
		t.Errorf("handleRequest.HitCount = %d, want 4", hr.HitCount)
	}

	// handleError: one range, count=2.
	he, ok := bySymbol["sym:handleError"]
	if !ok {
		t.Fatal("missing coverage for sym:handleError")
	}
	if he.HitCount != 2 {
		t.Errorf("handleError.HitCount = %d, want 2", he.HitCount)
	}

	// query: one range, count=5 (the count=0 range is filtered out).
	q, ok := bySymbol["sym:query"]
	if !ok {
		t.Fatal("missing coverage for sym:query")
	}
	if q.HitCount != 5 {
		t.Errorf("query.HitCount = %d, want 5", q.HitCount)
	}
}

func TestJoinCoverageToSymbolsEmpty(t *testing.T) {
	records := joinCoverageToSymbols(nil, nil)
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}
