package coverage

import (
	"sort"

	"github.com/jeffdhooton/scry/internal/store"
)

// defSpan is a function/method definition's line range within a file.
type defSpan struct {
	symbolID    string
	displayName string
	line        int
	endLine     int
}

// buildDefIndex loads all definition records from the store and builds a
// per-file sorted index of definition line spans. This lets the join step
// binary-search for which function a covered line belongs to.
func buildDefIndex(st *store.Store) (map[string][]defSpan, error) {
	idx := map[string][]defSpan{}
	err := st.IterateAllDefs(func(rec *store.OccurrenceRecord) error {
		// Skip defs with no line range — they can't participate in coverage matching.
		if rec.Line <= 0 || rec.EndLine <= 0 {
			return nil
		}
		sym, _ := st.GetSymbol(rec.Symbol)
		displayName := ""
		if sym != nil {
			displayName = sym.DisplayName
		}
		idx[rec.File] = append(idx[rec.File], defSpan{
			symbolID:    rec.Symbol,
			displayName: displayName,
			line:        rec.Line,
			endLine:     rec.EndLine,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort each file's spans by start line for binary search.
	for file, spans := range idx {
		sort.Slice(spans, func(i, j int) bool {
			return spans[i].line < spans[j].line
		})

		// Fix single-line spans: scip-go emits def occurrences where
		// EndLine == Line (just the function signature, not the body).
		// Extend each such span to just before the next definition in the
		// same file, or to a generous default (+500 lines) if it's the last.
		// This lets coverage data within the function body match correctly.
		for i := range spans {
			if spans[i].endLine <= spans[i].line {
				if i+1 < len(spans) {
					spans[i].endLine = spans[i+1].line - 1
				} else {
					spans[i].endLine = spans[i].line + 500
				}
			}
		}
		idx[file] = spans
	}
	return idx, nil
}

// joinCoverageToSymbols maps covered ranges to symbol definitions by checking
// which definition spans contain the covered lines. Returns deduplicated
// coverage records (one per symbol, with accumulated hit counts).
func joinCoverageToSymbols(ranges []CoveredRange, defIdx map[string][]defSpan) []*store.CoverageRecord {
	// Accumulate hits per symbol.
	type accum struct {
		rec      *store.CoverageRecord
		hitCount int
	}
	bySymbol := map[string]*accum{}

	for _, cr := range ranges {
		if cr.Count <= 0 {
			continue // not actually executed
		}
		spans, ok := defIdx[cr.File]
		if !ok {
			continue
		}

		// Find all definition spans that overlap with this covered range.
		for i := range spans {
			sp := &spans[i]
			// Check overlap: covered range [cr.Line, cr.EndLine] intersects
			// definition span [sp.line, sp.endLine].
			if cr.EndLine < sp.line {
				continue
			}
			if cr.Line > sp.endLine {
				continue
			}
			// Overlap — this symbol is covered.
			if a, ok := bySymbol[sp.symbolID]; ok {
				a.hitCount += cr.Count
			} else {
				bySymbol[sp.symbolID] = &accum{
					rec: &store.CoverageRecord{
						Symbol:      sp.symbolID,
						DisplayName: sp.displayName,
						File:        cr.File,
						Line:        sp.line,
						EndLine:     sp.endLine,
					},
					hitCount: cr.Count,
				}
			}
		}
	}

	result := make([]*store.CoverageRecord, 0, len(bySymbol))
	for _, a := range bySymbol {
		a.rec.HitCount = a.hitCount
		result = append(result, a.rec)
	}
	return result
}
