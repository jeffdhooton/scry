package store

import (
	"bytes"
	"sort"
)

func episodeSemantic(r episodeResult) ([]byte, error) {
	r.Predecessor = ""
	raw, _, err := encodeEpisodeResult(r)
	return raw, err
}

func outcomeSemantic(o identityOutcome) ([]byte, error) {
	o.Predecessor = ""
	raw, _, err := encodeIdentityOutcome(o)
	return raw, err
}

// Returns STAGED descriptive data. Only a future fixed outer wrapper can report
// durable success after its actual commit; this function cannot observe commit.
func putEpisodeSelection(st *Store, id string, expected episodeHeadExpectation, proposal episodeProposal) (episodeSelection, error) {
	fail := func(err error) (episodeSelection, error) { return episodeSelection{}, episodeSelectionFailure(st, err) }
	if st == nil || st.admissionOwner == nil || st.admissionOwner.phase != admissionFinalizing || st.admissionFailure != nil {
		return fail(errEpisodeSelection)
	}
	if err := generationTransaction(st); err != nil {
		return fail(err)
	}
	if proposal.Result.EpisodeID != id || proposal.Result.Predecessor != "" || proposal.Outcomes == nil || (!expected.Exists && expected.Raw != nil) {
		return fail(errEpisodeSelection)
	}
	expected.Raw = bytes.Clone(expected.Raw)
	if expected.Exists {
		if _, err := decodeEpisodeHead(episodeHeadKey(id), expected.Raw, id); err != nil {
			return fail(err)
		}
	}
	// Canonical encode/decode owns all nested slices and normalizes UTC births.
	inputRaw, inputKey, err := encodeEpisodeResult(proposal.Result)
	if err != nil {
		return fail(err)
	}
	r, err := decodeEpisodeResult(inputKey, inputRaw, id)
	if err != nil {
		return fail(err)
	}
	for _, a := range r.Assertions {
		if a.OutcomeKey != "" {
			return fail(errEpisodeSelection)
		}
	}
	for _, b := range r.Births {
		if b.OutcomeKey != "" {
			return fail(errEpisodeSelection)
		}
	}
	basis, err := newEpisodeBasis(st, r.InputKey, id)
	if err != nil {
		return fail(err)
	}
	proposed := make([]identityOutcome, len(proposal.Outcomes))
	for i, o := range proposal.Outcomes {
		if o.Predecessor != "" {
			return fail(errEpisodeSelection)
		}
		proposed[i], err = basis.outcome(o, false)
		if err != nil {
			return fail(err)
		}
	}
	actual, exists, err := generationRead(st, []byte(episodeHeadKey(id)))
	if err != nil || exists != expected.Exists || !bytes.Equal(actual, expected.Raw) {
		return fail(err)
	}
	current, currentOutcomes, err := readEpisodeSelectionTxn(st, id)
	if err != nil {
		return fail(err)
	}
	lineages := map[string]string{}
	for key, o := range currentOutcomes {
		lineage := string(outcomeLineage(o))
		if _, exists := lineages[lineage]; exists {
			return fail(errEpisodeSelection)
		}
		lineages[lineage] = key
	}
	positions := map[string]int{}
	for i, b := range r.Births {
		if _, exists := positions[b.FirstObservation]; exists {
			return fail(errEpisodeSelection)
		}
		positions[b.FirstObservation] = i
	}
	for i := range r.Declarations {
		seen := map[string]bool{}
		for _, ref := range r.Declarations[i].Births {
			if _, exists := positions[ref]; !exists || seen[ref] {
				return fail(errEpisodeSelection)
			}
			seen[ref] = true
		}
		sort.Slice(r.Declarations[i].Births, func(a, b int) bool {
			return positions[r.Declarations[i].Births[a]] < positions[r.Declarations[i].Births[b]]
		})
	}
	outcomes := map[string]identityOutcome{}
	for _, o := range proposed {
		if oldKey, exists := lineages[string(outcomeLineage(o))]; exists {
			old := currentOutcomes[oldKey]
			a, ae := outcomeSemantic(o)
			b, be := outcomeSemantic(old)
			if ae != nil || be != nil {
				return fail(errEpisodeSelection)
			}
			if bytes.Equal(a, b) {
				o = old
			} else {
				o.Predecessor = oldKey
			}
		}
		_, key, err := encodeIdentityOutcome(o)
		if err != nil {
			return fail(err)
		}
		if _, exists := outcomes[key]; exists {
			return fail(errEpisodeSelection)
		}
		if o.Birth != nil {
			found := -1
			for i, b := range r.Births {
				if sameEpisodeBirth(b.Birth, *o.Birth) {
					if found != -1 {
						return fail(errEpisodeSelection)
					}
					found = i
				}
			}
			if found < 0 || r.Births[found].OutcomeKey != "" {
				return fail(errEpisodeSelection)
			}
			r.Births[found].OutcomeKey = key
		} else {
			ordinal := *o.AssertionOrdinal
			if ordinal < 0 || ordinal >= len(r.Assertions) || r.Assertions[ordinal].OutcomeKey != "" {
				return fail(errEpisodeSelection)
			}
			r.Assertions[ordinal].OutcomeKey = key
		}
		outcomes[key] = o
	}
	counts, err := validateEpisodeDescription(basis, r, outcomes)
	if err != nil {
		return fail(err)
	}
	semantic, err := episodeSemantic(r)
	if err != nil {
		return fail(err)
	}
	if current.Selected {
		old, err := episodeSemantic(current.Result)
		if err != nil {
			return fail(err)
		}
		if bytes.Equal(semantic, old) {
			return current, nil
		}
		if current.Head.Revision == ^uint64(0) {
			return fail(errEpisodeSelection)
		}
		r.Predecessor = current.Head.ResultKey
	}
	resultRaw, resultKey, err := encodeEpisodeResult(r)
	if err != nil {
		return fail(err)
	}
	head := episodeHead{Version: 1, EpisodeID: id, ResultKey: resultKey, Revision: 1}
	if current.Selected {
		head.Revision = current.Head.Revision + 1
	}
	headRaw, err := encodeEpisodeHead(head)
	if err != nil {
		return fail(err)
	}
	// Complete preflight of every occupied immutable key precedes every Set.
	keys := make([]string, 0, len(outcomes))
	for key, o := range outcomes {
		raw, want, err := encodeIdentityOutcome(o)
		if err != nil || want != key {
			return fail(errEpisodeSelection)
		}
		old, exists, err := generationRead(st, []byte(key))
		if err != nil || (exists && !bytes.Equal(old, raw)) {
			return fail(err)
		}
		if o.Predecessor != "" {
			prior, _, err := readOutcomeRow(st, o.Predecessor, id, false)
			if err != nil || !bytes.Equal(outcomeLineage(prior), outcomeLineage(o)) {
				return fail(errEpisodeSelection)
			}
		}
		keys = append(keys, key)
	}
	old, exists, err := generationRead(st, []byte(resultKey))
	if err != nil || (exists && !bytes.Equal(old, resultRaw)) {
		return fail(err)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if got, err := putIdentityOutcome(st, outcomes[key]); err != nil || got != key {
			return fail(err)
		}
	}
	if !exists {
		if err := st.txn.Set([]byte(resultKey), bytes.Clone(resultRaw)); err != nil {
			return fail(err)
		}
	}
	if err := st.txn.Set([]byte(episodeHeadKey(id)), bytes.Clone(headRaw)); err != nil {
		return fail(err)
	}
	return episodeSelection{Selected: true, Head: head, HeadRaw: bytes.Clone(headRaw), Result: r, ResultRaw: bytes.Clone(resultRaw), Counts: counts}, nil
}
