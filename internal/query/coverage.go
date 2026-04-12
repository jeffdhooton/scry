package query

import (
	"github.com/jeffdhooton/scry/internal/store"
)

// Tests returns the coverage status of symbols matching `name`. For each
// matching symbol that has coverage data, it returns a SymbolHit with a
// single synthetic occurrence whose Context field contains the hit count.
// Symbols with no coverage data are omitted from results.
func Tests(s *store.Store, name string) (*Result, error) {
	ids, err := s.LookupSymbolsByName(name)
	if err != nil {
		return nil, err
	}
	res := &Result{Symbol: name, Matches: []SymbolHit{}}
	for _, id := range ids {
		cov, err := s.GetCoverage(id)
		if err != nil {
			return nil, err
		}
		if cov == nil {
			continue // no coverage data for this symbol
		}
		sym, err := s.GetSymbol(id)
		if err != nil {
			return nil, err
		}
		hit := SymbolHit{
			SymbolID: id,
			Occurrences: []*store.OccurrenceRecord{
				{
					Symbol:  id,
					File:    cov.File,
					Line:    cov.Line,
					EndLine: cov.EndLine,
					Context: covContext(cov.HitCount),
				},
			},
		}
		if sym != nil {
			hit.DisplayName = sym.DisplayName
			hit.Kind = sym.Kind
		}
		res.Matches = append(res.Matches, hit)
		res.Total++
	}
	return res, nil
}

func covContext(hitCount int) string {
	if hitCount > 0 {
		return "covered (hit count: " + itoa(hitCount) + ")"
	}
	return "not covered"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	// Simple int-to-string without importing strconv.
	buf := [20]byte{}
	i := len(buf)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
