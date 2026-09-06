package resolve

import (
	"fmt"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

// preserveHistoricalAddress is an exact occupied-address protection, not a
// text-based history matcher. Current-triple coalescing stays unchanged.
func preserveHistoricalAddress(st *store.Store, ep store.Episode, rf resolvedFact, stats *Stats) (bool, error) {
	old, err := st.HistoricalRestatement(store.Fact{Src: rf.src, Relation: rf.fct.Relation, Dst: rf.dst, Value: rf.value, RawRelation: rf.rawRel, Fact: rf.fct.Fact, ValidFrom: rf.validFrom})
	if err != nil || old == nil {
		return false, err
	}
	if rf.fct.ValidFrom != "" {
		_, rfcErr := time.Parse(time.RFC3339, rf.fct.ValidFrom)
		_, dateErr := time.Parse("2006-01-02", rf.fct.ValidFrom)
		if rfcErr != nil && dateErr != nil {
			return false, fmt.Errorf("%w: invalid explicit date cannot identify a historical assertion", store.ErrFactConflict)
		}
	}
	// The exact same start prevents mergeFact's relocation branch. It unions
	// evidence, raises confidence only, and carries InvalidAt through unchanged.
	if err := mergeFact(st, ep, *old, old.ValidFrom, rf.fct.Confidence, stats); err != nil {
		return false, err
	}
	return true, nil
}
