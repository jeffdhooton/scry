package resolve

import (
	"fmt"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

// A fallback edge is intentionally untyped: sharing its endpoints does not
// establish that two sentences restate the same fact. Keep their actual
// timestamps and evidence separate; do not invent extra timestamp precision.
func matchingFactForMerge(st *store.Store, rf resolvedFact) (*store.Fact, error) {
	if rf.fct.Relation != Fallback {
		return currentFact(st, rf.src, rf.fct.Relation, rf.keyDst())
	}
	facts, err := st.FactsFrom(rf.src, false)
	if err != nil {
		return nil, err
	}
	for i := range facts {
		f := &facts[i]
		if f.Relation == Fallback && f.Dst == rf.dst && f.Value == rf.value && f.Fact == rf.fct.Fact && f.RawRelation == rf.rawRel {
			return f, nil
		}
	}
	return nil, nil
}

// Called inside Apply's single write transaction, including before moving
// an exact restatement to an earlier ValidFrom. Historical keys count too.
func requireVacantFallbackKey(st *store.Store, incoming store.Fact) error {
	facts, err := st.FactsFrom(incoming.Src, true)
	if err != nil {
		return err
	}
	for _, f := range facts {
		if f.Relation == incoming.Relation && f.KeyDst() == incoming.KeyDst() && f.ValidFrom.UnixNano() == incoming.ValidFrom.UnixNano() {
			return fmt.Errorf("%w: %s -[%s]-> %s at %s; preserve the queued episode for exact fact review", store.ErrFactConflict, incoming.Src, incoming.Relation, incoming.KeyDst(), incoming.ValidFrom.Format("2006-01-02T15:04:05.999999999Z07:00"))
		}
	}
	return nil
}

func currentFallbackForReference(st *store.Store, src, keyDst, raw string) (*store.Fact, error) {
	facts, err := st.FactsFrom(src, false)
	if err != nil {
		return nil, err
	}
	var match *store.Fact
	for i := range facts {
		f := &facts[i]
		effectiveRaw := f.RawRelation
		if effectiveRaw == "" {
			effectiveRaw = f.Relation
		}
		if f.Relation != Fallback || f.KeyDst() != keyDst || effectiveRaw != raw {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("%w: supersedes %s -[%s]-> %s matches multiple fallback statements", store.ErrFactConflict, src, raw, keyDst)
		}
		match = f
	}
	return match, nil
}
