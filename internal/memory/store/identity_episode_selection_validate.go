package store

import "bytes"

// Complete structural coverage only. Dispositions are not semantic permission.
func validateEpisodeDescription(b *episodeBasis, r episodeResult, outcomes map[string]identityOutcome) (episodeCounts, error) {
	var counts episodeCounts
	fail := func() (episodeCounts, error) { return episodeCounts{}, errEpisodeSelection }
	if r.EpisodeID != b.input.EpisodeID || r.InputKey != b.key || len(r.Declarations) != len(b.input.Declarations) || len(r.Assertions) != len(b.input.Facts) {
		return fail()
	}
	used := map[string]bool{}
	get := func(key string) (identityOutcome, bool) {
		o, ok := outcomes[key]
		if !ok || key == "" || used[key] {
			return identityOutcome{}, false
		}
		used[key] = true
		return o, true
	}
	for i, d := range r.Declarations {
		if d.Ordinal != i || d.Births == nil {
			return fail()
		}
		if _, err := b.slot(d.ObservationKey, "declaration", i, ""); err != nil {
			return fail()
		}
		switch d.Kind {
		case "identity-mention":
		case "value-mention":
			if len(d.Births) != 0 {
				return fail()
			}
		case "deferred-identity":
			if len(d.Births) != 0 {
				return fail()
			}
			counts.DeclarationDeferred++
		default:
			return fail()
		}
	}
	for i, a := range r.Assertions {
		if a.Ordinal != i {
			return fail()
		}
		if _, err := b.slot(a.SourceObservation, "endpoint", i, "src"); err != nil {
			return fail()
		}
		if _, err := b.slot(a.DestinationObservation, "endpoint", i, "dst"); err != nil {
			return fail()
		}
		switch a.Kind {
		case "committed", "deferred":
			if a.Kind == "committed" {
				if a.Reason != "" {
					return fail()
				}
				counts.Committed++
			} else {
				if a.Reason != "identity-dependency" && a.Reason != "assertion-dependency" {
					return fail()
				}
				counts.Deferred++
			}
			o, ok := get(a.OutcomeKey)
			if !ok || o.Birth != nil || o.AssertionOrdinal == nil || *o.AssertionOrdinal != i || o.Disposition != a.Kind || !outcomeHasLink(o, a.SourceObservation, "primary-src") || !outcomeHasLink(o, a.DestinationObservation, "primary-dst") {
				return fail()
			}
			if b.input.Facts[i].Supersedes != nil {
				for _, role := range []string{"supersedes-src", "supersedes-dst"} {
					if !outcomeHasLink(o, a.SourceObservation, role) && !outcomeHasLink(o, a.DestinationObservation, role) {
						return fail()
					}
				}
			}
		case "non-assertion":
			if a.OutcomeKey != "" || (a.Reason != "empty-relation" && a.Reason != "two-values") {
				return fail()
			}
			counts.NonAssertion++
		case "unresolved":
			if a.OutcomeKey != "" || (a.Reason != "missing-source" && a.Reason != "missing-destination") {
				return fail()
			}
			counts.Unresolved++
		default:
			return fail()
		}
	}
	birthPositions := map[string]int{}
	slugs := map[string]bool{}
	reverse := map[string]map[int]bool{}
	var previous [3]int
	for i, entry := range r.Births {
		if _, ok := birthPositions[entry.FirstObservation]; ok || slugs[entry.Birth.Slug] {
			return fail()
		}
		first, err := b.observation(entry.FirstObservation)
		if err != nil {
			return fail()
		}
		order := observationOrder(first)
		if i > 0 && !orderLess(previous, order) {
			return fail()
		}
		previous = order
		name, role := "", "declaration"
		if first.Declaration != nil {
			name = first.Declaration.Name
		} else {
			role = "primary-" + first.Side
			name = first.Fact.Src
			if first.Side == "dst" {
				name = first.Fact.Dst
			}
		}
		birth := entry.Birth
		if birth.EpisodeID != r.EpisodeID || birth.Occurrence != first.Ordinal || birth.Origin != first.Origin || birth.Name != name || !birth.CreatedAt.Equal(first.OccurredAt) {
			return fail()
		}
		o, ok := get(entry.OutcomeKey)
		if !ok || o.Birth == nil || !sameEpisodeBirth(*o.Birth, birth) || !outcomeHasLink(o, entry.FirstObservation, role) {
			return fail()
		}
		if o.Disposition == "supported" {
			counts.BirthSupported++
		} else if o.Disposition == "no-assertion" {
			counts.BirthNoAssertion++
		} else {
			return fail()
		}
		birthPositions[entry.FirstObservation] = i
		slugs[birth.Slug] = true
		reverse[entry.FirstObservation] = map[int]bool{}
		for _, l := range o.Links {
			if l.Role != "declaration" {
				continue
			}
			obs, err := b.observation(l.Key)
			if err != nil || obs.Origin != "declaration" || obs.Ordinal >= len(r.Declarations) || r.Declarations[obs.Ordinal].ObservationKey != l.Key {
				return fail()
			}
			reverse[entry.FirstObservation][obs.Ordinal] = true
		}
	}
	for ordinal, d := range r.Declarations {
		previous := -1
		for _, ref := range d.Births {
			position, exists := birthPositions[ref]
			if !exists || position <= previous || !reverse[ref][ordinal] {
				return fail()
			}
			delete(reverse[ref], ordinal)
			previous = position
		}
	}
	for _, refs := range reverse {
		if len(refs) != 0 {
			return fail()
		}
	}
	if len(used) != len(outcomes) {
		return fail()
	}
	return counts, nil
}

// This reads and validates a result and all its own linked inputs/outcomes, but
// deliberately does not recurse through the episode predecessor chain.
func readEpisodeResultRow(st *Store, key, id string) (episodeResult, []byte, episodeCounts, map[string]identityOutcome, error) {
	fail := func() (episodeResult, []byte, episodeCounts, map[string]identityOutcome, error) {
		return episodeResult{}, nil, episodeCounts{}, nil, errEpisodeSelection
	}
	raw, exists, err := generationRead(st, []byte(key))
	if err != nil || !exists {
		return fail()
	}
	r, err := decodeEpisodeResult(key, raw, id)
	if err != nil || r.Predecessor == key {
		return fail()
	}
	basis, err := newEpisodeBasis(st, r.InputKey, id)
	if err != nil {
		return fail()
	}
	outcomes := map[string]identityOutcome{}
	keys := []string{}
	for _, a := range r.Assertions {
		if a.OutcomeKey != "" {
			keys = append(keys, a.OutcomeKey)
		}
	}
	for _, b := range r.Births {
		keys = append(keys, b.OutcomeKey)
	}
	for _, key := range keys {
		if _, exists := outcomes[key]; exists {
			return fail()
		}
		o, _, err := readOutcomeRow(st, key, id, true)
		if err != nil {
			return fail()
		}
		o, err = basis.outcome(o, true)
		if err != nil {
			return fail()
		}
		outcomes[key] = o
	}
	counts, err := validateEpisodeDescription(basis, r, outcomes)
	if err != nil {
		return fail()
	}
	return r, raw, counts, outcomes, nil
}

func readEpisodeSelectionTxn(st *Store, id string) (episodeSelection, map[string]identityOutcome, error) {
	fail := func() (episodeSelection, map[string]identityOutcome, error) {
		return episodeSelection{}, nil, errEpisodeSelection
	}
	raw, exists, err := generationRead(st, []byte(episodeHeadKey(id)))
	if err != nil {
		return fail()
	}
	if !exists {
		return episodeSelection{}, map[string]identityOutcome{}, nil
	}
	head, err := decodeEpisodeHead(episodeHeadKey(id), raw, id)
	if err != nil {
		return fail()
	}
	result, resultRaw, counts, outcomes, err := readEpisodeResultRow(st, head.ResultKey, id)
	if err != nil || (head.Revision == 1) != (result.Predecessor == "") {
		return fail()
	}
	if result.Predecessor != "" {
		if result.Predecessor == head.ResultKey {
			return fail()
		}
		if _, _, _, _, err := readEpisodeResultRow(st, result.Predecessor, id); err != nil {
			return fail()
		}
	}
	return episodeSelection{Selected: true, Head: head, HeadRaw: bytes.Clone(raw), Result: result, ResultRaw: bytes.Clone(resultRaw), Counts: counts}, outcomes, nil
}
