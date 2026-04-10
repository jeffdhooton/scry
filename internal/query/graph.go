package query

import (
	"github.com/jeffdhooton/scry/internal/store"
)

// Callers returns every reference to symbols matching `name`, with the
// containing function exposed via OccurrenceRecord.ContainingSymbol.
//
// In practice this is the same as Refs (every ref is a potential call site)
// but the response makes the containing function visible so the agent can
// jump straight to the caller. For SCIP indexers that don't populate
// enclosing_range (notably scip-go), ContainingSymbol will be empty.
func Callers(s *store.Store, name string) (*Result, error) {
	return Refs(s, name)
}

// Callees returns the call edges originating from any symbol matching `name`.
// Each result item is a callee occurrence: which symbol was called, where in
// the source the call appears, and the source line for context.
//
// Limitation: only populated when the SCIP indexer set enclosing_range on
// definition occurrences. scip-typescript does; scip-go doesn't. For Go
// targets, this returns empty.
func Callees(s *store.Store, name string) (*Result, error) {
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
		if err := s.IterateCallees(id, func(rec *store.OccurrenceRecord) error {
			hit.Occurrences = append(hit.Occurrences, rec)
			return nil
		}); err != nil {
			return nil, err
		}
		res.Matches = append(res.Matches, hit)
		res.Total += len(hit.Occurrences)
	}
	return res, nil
}

// Impls returns every symbol that implements the type/interface matching
// `name`. Both directions of `is_implementation` are walked at index time;
// this is a simple prefix scan.
func Impls(s *store.Store, name string) (*Result, error) {
	ids, err := s.LookupSymbolsByName(name)
	if err != nil {
		return nil, err
	}
	res := &Result{Symbol: name, Matches: []SymbolHit{}}
	for _, baseID := range ids {
		baseSym, err := s.GetSymbol(baseID)
		if err != nil {
			return nil, err
		}
		hit := SymbolHit{
			SymbolID:    baseID,
			Occurrences: []*store.OccurrenceRecord{},
		}
		if baseSym != nil {
			hit.DisplayName = baseSym.DisplayName
			hit.Kind = baseSym.Kind
		}
		implIDs, err := s.IterateImpls(baseID)
		if err != nil {
			return nil, err
		}
		for _, implID := range implIDs {
			// Each impl shows up as a synthesized "occurrence" pointing at
			// the impl's definition site. We resolve the def from the store
			// to get a real file/line/col.
			implSym, _ := s.GetSymbol(implID)
			rec := &store.OccurrenceRecord{Symbol: implID}
			// Pull the first def site if available.
			_ = s.IterateDefs(implID, func(def *store.OccurrenceRecord) error {
				rec.File = def.File
				rec.Line = def.Line
				rec.Column = def.Column
				rec.Context = def.Context
				return errStopIteration
			})
			if implSym != nil {
				rec.ContainingSymbol = implSym.DisplayName
			}
			hit.Occurrences = append(hit.Occurrences, rec)
		}
		res.Matches = append(res.Matches, hit)
		res.Total += len(hit.Occurrences)
	}
	return res, nil
}

// errStopIteration is a sentinel used to break out of an iterator early. The
// store iterator surfaces it back to the caller as an error, which we ignore.
var errStopIteration = stopIterErr{}

type stopIterErr struct{}

func (stopIterErr) Error() string { return "stop" }
