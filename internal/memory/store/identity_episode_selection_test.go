package store

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func selectionInput(t *testing.T, st *Store, input identityInputRevision) (string, []string, [][2]string) {
	t.Helper()
	var key string
	decls := []string{}
	facts := [][2]string{}
	err := st.AtomicWrite(func(tx *Store) error {
		if err := tx.PutEpisode(Episode{ID: input.EpisodeID, OccurredAt: input.OccurredAt}); err != nil {
			return err
		}
		var err error
		key, err = putIdentityInput(tx, input)
		if err != nil {
			return err
		}
		for i, d := range input.Declarations {
			o := identityObservation{Version: 1, EpisodeID: input.EpisodeID, OccurredAt: input.OccurredAt, Cwd: input.Cwd, Origin: "declaration", Ordinal: i, Declaration: &d}
			k, err := putIdentityObservation(tx, o)
			if err != nil {
				return err
			}
			decls = append(decls, k)
		}
		for i, f := range input.Facts {
			var pair [2]string
			for j, side := range []string{"src", "dst"} {
				o := identityObservation{Version: 1, EpisodeID: input.EpisodeID, OccurredAt: input.OccurredAt, Cwd: input.Cwd, Origin: "endpoint", Ordinal: i, Side: side, Fact: &f}
				k, err := putIdentityObservation(tx, o)
				if err != nil {
					return err
				}
				pair[j] = k
			}
			facts = append(facts, pair)
		}
		return nil
	})
	if err != nil {
		t.Fatal("synthetic input setup", err)
	}
	return key, decls, facts
}

func selectionFixture(t *testing.T, st *Store, id string, birth bool) (identityInputRevision, episodeProposal) {
	t.Helper()
	input := identityInputRevision{Version: 1, EpisodeID: id, OccurredAt: time.Unix(123456, 789).UTC(), Cwd: "/synthetic", Summary: "synthetic", Declarations: []observedDeclaration{{Name: "Atlas", Type: "project", Aliases: []string{"A", "A"}, TypeFallback: true}}, Facts: []observedFact{{Src: "Atlas", Relation: "status", Dst: "active", Fact: "synthetic assertion", Confidence: 0.5}}}
	key, decls, facts := selectionInput(t, st, input)
	p := episodeProposal{Result: episodeResult{Version: 1, EpisodeID: id, InputKey: key, Declarations: []episodeDeclaration{{Ordinal: 0, ObservationKey: decls[0], Kind: "identity-mention", Births: []string{}}}, Assertions: []episodeAssertion{{Ordinal: 0, Kind: "deferred", Reason: "identity-dependency", SourceObservation: facts[0][0], DestinationObservation: facts[0][1]}}, Births: []episodeBirth{}}, Outcomes: []identityOutcome{}}
	ordinal := 0
	p.Outcomes = append(p.Outcomes, identityOutcome{Version: 1, EpisodeID: id, AssertionOrdinal: &ordinal, Disposition: "deferred", Links: []outcomeLink{{Key: facts[0][1], Role: "primary-dst", FinalSide: "none", Spelling: "active", State: "not-materialized"}, {Key: facts[0][0], Role: "primary-src", FinalSide: "none", Spelling: "Atlas", State: "deferred"}}})
	if birth {
		b := identityBirth{EpisodeID: id, Occurrence: 0, Slug: "atlas", Name: "Atlas", Origin: "declaration", CreatedAt: input.OccurredAt}
		p.Result.Declarations[0].Births = []string{decls[0]}
		p.Result.Births = append(p.Result.Births, episodeBirth{Birth: b, FirstObservation: decls[0]})
		p.Outcomes = append(p.Outcomes, identityOutcome{Version: 1, EpisodeID: id, Birth: &b, Disposition: "no-assertion", Links: []outcomeLink{{Key: facts[0][0], Role: "primary-src", FinalSide: "none", Spelling: "Atlas", State: "not-materialized"}, {Key: decls[0], Role: "declaration", FinalSide: "none", Spelling: "Atlas", State: "not-materialized"}}})
	}
	return input, p
}

// Test harness only: private writer returns staged data before outer commit.
func selectionPut(st *Store, p episodeProposal, expected episodeHeadExpectation) (episodeSelection, error) {
	var staged episodeSelection
	err := runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
		var err error
		staged, err = putEpisodeSelection(tx, p.Result.EpisodeID, expected, p)
		return err
	})
	if err != nil {
		return episodeSelection{}, err
	}
	return staged, nil
}

func TestEpisodeSelectionCompleteNoOpAndOwnedResults(t *testing.T) {
	st := openTemp(t)
	_, p := selectionFixture(t, st, "selection", true)
	original := dpJSON(t, p)
	before := dpRows(t, st)
	first, err := selectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, dpJSON(t, p)) {
		t.Fatal("writer changed proposal")
	}
	if !first.Selected || first.Head.Revision != 1 || first.Counts.Deferred != 1 || first.Counts.BirthNoAssertion != 1 || first.Counts.Committed != 0 {
		t.Fatal("wrong complete result")
	}
	after := dpRows(t, st)
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("selection changed old rows")
		}
	}
	second, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: first.HeadRaw})
	if err != nil || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(after, dpRows(t, st)) {
		t.Fatal("identical retry churned", err)
	}
	// Permuted proposal Links are semantically identical under new owned ordering.
	p.Outcomes[0].Links[0], p.Outcomes[0].Links[1] = p.Outcomes[0].Links[1], p.Outcomes[0].Links[0]
	third, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: first.HeadRaw})
	if err != nil || !reflect.DeepEqual(first, third) || !reflect.DeepEqual(after, dpRows(t, st)) {
		t.Fatal("link order caused churn", err)
	}
	read, err := readEpisodeSelection(st, "selection")
	if err != nil || !reflect.DeepEqual(first, read) {
		t.Fatal("read mismatch", err)
	}
	read.HeadRaw[0] = 0
	read.ResultRaw[0] = 0
	read.Result.Declarations[0].Births[0] = "changed"
	if !reflect.DeepEqual(after, dpRows(t, st)) || first.Result.Declarations[0].Births[0] == "changed" {
		t.Fatal("result aliases caller/store")
	}
}

func TestEpisodeSelectionChangedInputAndHistoricalDeduplication(t *testing.T) {
	st := openTemp(t)
	input, p := selectionFixture(t, st, "transition", true)
	a, err := selectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	old := dpRows(t, st)
	input.Summary = "changed extraction removes occurrences"
	input.Declarations = []observedDeclaration{}
	input.Facts = []observedFact{}
	key, _, _ := selectionInput(t, st, input)
	q := episodeProposal{Result: episodeResult{Version: 1, EpisodeID: input.EpisodeID, InputKey: key, Declarations: []episodeDeclaration{}, Assertions: []episodeAssertion{}, Births: []episodeBirth{}}, Outcomes: []identityOutcome{}}
	b, err := selectionPut(st, q, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
	if err != nil || b.Head.Revision != 2 || b.Result.Predecessor != a.Head.ResultKey || b.Counts != (episodeCounts{}) {
		t.Fatal("omission transition failed", err)
	}
	c, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: b.HeadRaw})
	if err != nil || c.Head.Revision != 3 || c.Head.ResultKey == a.Head.ResultKey || c.Result.Predecessor != b.Head.ResultKey {
		t.Fatal("A/B/A reset history", err)
	}
	if c.Result.Assertions[0].OutcomeKey != a.Result.Assertions[0].OutcomeKey || c.Result.Births[0].OutcomeKey != a.Result.Births[0].OutcomeKey {
		t.Fatal("byte-identical historical outcomes not deduplicated")
	}
	for k, v := range old {
		if k == episodeHeadKey(input.EpisodeID) {
			continue
		}
		if !bytes.Equal(v, dpRows(t, st)[k]) {
			t.Fatal("transition changed immutable history")
		}
	}
	before := dpRows(t, st)
	if _, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw}); err == nil {
		t.Fatal("stale semantic no-op accepted")
	}
	if !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("stale refusal changed state")
	}
}

func TestEpisodeSelectionSameInputImprovementAndFreshOnlyReplayLimit(t *testing.T) {
	st := openTemp(t)
	_, p := selectionFixture(t, st, "improvement", true)
	a, err := selectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	p.Result.Assertions[0].Kind = "committed"
	p.Result.Assertions[0].Reason = ""
	o := &p.Outcomes[0]
	o.Disposition = "committed"
	for i := range o.Links {
		l := &o.Links[i]
		l.State = "resolved"
		if l.Role == "primary-src" {
			l.FinalSide = "src"
			l.ResolvedSlug = "atlas"
		} else {
			l.FinalSide = "value"
		}
	}
	b, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
	if err != nil || b.Head.Revision != 2 || b.Counts.Committed != 1 || b.Counts.Deferred != 0 {
		t.Fatal("same-input improvement failed", err)
	}
	if b.Result.Births[0].OutcomeKey != a.Result.Births[0].OutcomeKey {
		t.Fatal("unchanged birth outcome churned")
	}
	var stored identityOutcome
	json.Unmarshal(dpRows(t, st)[b.Result.Assertions[0].OutcomeKey], &stored)
	if stored.Predecessor != a.Result.Assertions[0].OutcomeKey {
		t.Fatal("same lineage predecessor wrong")
	}
	// The structural selector cannot infer omitted prior births from names.
	// This intentionally changed description must not be called Force safety.
	p.Result.Births = []episodeBirth{}
	p.Result.Declarations[0].Births = []string{}
	p.Outcomes = p.Outcomes[:1]
	c, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: b.HeadRaw})
	if err != nil || c.Head.Revision != 3 || len(c.Result.Births) != 0 {
		t.Fatal("structural inventory boundary changed", err)
	}
}
