// Package query implements the read side of scry: take a name (or symbol id),
// look it up in the store, and return a structured result that the CLI
// (and later the daemon) marshals to JSON.
//
// P0 has only `refs` and `defs`. `symbols` and `hover` come in P1.
package query

import (
	"github.com/jeffdhooton/scry/internal/store"
)

// Result is the JSON shape returned by the CLI.
type Result struct {
	Symbol    string       `json:"symbol"`
	Matches   []SymbolHit  `json:"matches"`
	Total     int          `json:"total"`
	ElapsedMs int64        `json:"elapsed_ms"`
}

// SymbolHit groups every occurrence under one resolved symbol id. A name like
// "processOrder" can match multiple symbols (e.g. one per file in a polyglot
// repo) so the response always returns a list, even if it's length 1.
type SymbolHit struct {
	SymbolID    string                    `json:"symbol_id"`
	DisplayName string                    `json:"display_name"`
	Kind        string                    `json:"kind,omitempty"`
	Occurrences []*store.OccurrenceRecord `json:"occurrences"`
}

// Refs returns every reference occurrence for symbols whose display name
// matches `name`. Case-insensitive. Definitions are NOT returned — use Defs
// for those.
func Refs(s *store.Store, name string) (*Result, error) {
	return collect(s, name, func(symID string, sink *[]*store.OccurrenceRecord) error {
		return s.IterateRefs(symID, func(rec *store.OccurrenceRecord) error {
			*sink = append(*sink, rec)
			return nil
		})
	})
}

// Defs returns every definition occurrence for symbols matching `name`.
func Defs(s *store.Store, name string) (*Result, error) {
	return collect(s, name, func(symID string, sink *[]*store.OccurrenceRecord) error {
		return s.IterateDefs(symID, func(rec *store.OccurrenceRecord) error {
			*sink = append(*sink, rec)
			return nil
		})
	})
}

func collect(s *store.Store, name string, gather func(string, *[]*store.OccurrenceRecord) error) (*Result, error) {
	ids, err := s.LookupSymbolsByName(name)
	if err != nil {
		return nil, err
	}
	res := &Result{Symbol: name, Matches: []SymbolHit{}}
	for _, id := range ids {
		sym, err := s.GetSymbol(id)
		if err != nil {
			return nil, err
		}
		hit := SymbolHit{
			SymbolID:    id,
			Occurrences: []*store.OccurrenceRecord{},
		}
		if sym != nil {
			hit.DisplayName = sym.DisplayName
			hit.Kind = sym.Kind
		}
		if err := gather(id, &hit.Occurrences); err != nil {
			return nil, err
		}
		res.Matches = append(res.Matches, hit)
		res.Total += len(hit.Occurrences)
	}
	return res, nil
}
