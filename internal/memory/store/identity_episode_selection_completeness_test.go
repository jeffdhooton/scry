package store

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestEpisodeSelectionNilEmptyAndUnselected(t *testing.T) {
	st := openTemp(t)
	input := identityInputRevision{Version: 1, EpisodeID: "empty", OccurredAt: time.Unix(123456, 789).UTC()}
	key, _, _ := selectionInput(t, st, input)
	unselected, err := readEpisodeSelection(st, input.EpisodeID)
	if err != nil || unselected.Selected || len(unselected.HeadRaw) != 0 {
		t.Fatal("legacy EP invented current completion")
	}
	p := episodeProposal{Result: episodeResult{Version: 1, EpisodeID: input.EpisodeID, InputKey: key, Declarations: []episodeDeclaration{}, Assertions: []episodeAssertion{}, Births: []episodeBirth{}}, Outcomes: []identityOutcome{}}
	a, err := selectionPut(st, p, episodeHeadExpectation{})
	if err != nil || a.Counts != (episodeCounts{}) {
		t.Fatal("nil input failed", err)
	}
	input.Declarations = []observedDeclaration{}
	input.Facts = []observedFact{}
	other, _, _ := selectionInput(t, st, input)
	if other == key {
		t.Fatal("input revision lost nil/empty")
	}
	p.Result.InputKey = other
	b, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
	if err != nil || b.Head.Revision != 2 || b.Result.Predecessor != a.Head.ResultKey {
		t.Fatal("empty revision incorrectly reused nil result", err)
	}
}

func TestEpisodeSelectionSupersedesCoverageWithoutBirth(t *testing.T) {
	for _, missing := range []string{"", "supersedes-src", "supersedes-dst"} {
		t.Run("missing-"+missing, func(t *testing.T) {
			st := openTemp(t)
			input, p := selectionFixture(t, st, "supersedes", false)
			input.Facts[0].Supersedes = &observedSupersedes{Src: "Previous", Relation: "status", Dst: "Old"}
			key, _, facts := selectionInput(t, st, input)
			p.Result.InputKey = key
			p.Result.Assertions[0].SourceObservation = facts[0][0]
			p.Result.Assertions[0].DestinationObservation = facts[0][1]
			p.Result.Assertions[0].Reason = "assertion-dependency"
			for i := range p.Outcomes[0].Links {
				l := &p.Outcomes[0].Links[i]
				if l.Role == "primary-src" {
					l.Key = facts[0][0]
				} else {
					l.Key = facts[0][1]
				}
			}
			for _, l := range []outcomeLink{{Key: facts[0][0], Role: "supersedes-src", FinalSide: "none", Spelling: "Previous", State: "deferred"}, {Key: facts[0][1], Role: "supersedes-dst", FinalSide: "none", Spelling: "Old", State: "deferred"}} {
				if l.Role != missing {
					p.Outcomes[0].Links = append(p.Outcomes[0].Links, l)
				}
			}
			before := dpRows(t, st)
			got, err := selectionPut(st, p, episodeHeadExpectation{})
			if missing != "" {
				if err == nil || !reflect.DeepEqual(before, dpRows(t, st)) {
					t.Fatal("missing hint role accepted")
				}
				return
			}
			if err != nil || got.Counts.Deferred != 1 || len(got.Result.Births) != 0 {
				t.Fatal("hint-only deferral fabricated birth", err)
			}
		})
	}
}

func TestEpisodeSelectionImmediatePredecessorOwnRevision(t *testing.T) {
	st := openTemp(t)
	input, p := selectionFixture(t, st, "predecessor", false)
	a, err := selectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	input.Facts[0].Fact = "changed fact payload"
	key, _, facts := selectionInput(t, st, input)
	p.Result.InputKey = key
	p.Result.Assertions[0].SourceObservation = facts[0][0]
	p.Result.Assertions[0].DestinationObservation = facts[0][1]
	for i := range p.Outcomes[0].Links {
		l := &p.Outcomes[0].Links[i]
		if l.Role == "primary-src" {
			l.Key = facts[0][0]
		} else {
			l.Key = facts[0][1]
		}
	}
	b, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
	if err != nil {
		t.Fatal(err)
	}
	read, err := readEpisodeSelection(st, "predecessor")
	if err != nil || !bytes.Equal(read.HeadRaw, b.HeadRaw) {
		t.Fatal("predecessor validated against wrong revision", err)
	}
	// Corrupt only the immediate predecessor's own input. The current input
	// and semantic proposal still validate, but current read/retry must refuse.
	dpSet(t, st, a.Result.InputKey, []byte("{}"))
	before := dpRows(t, st)
	if _, err := readEpisodeSelection(st, "predecessor"); err == nil {
		t.Fatal("malformed immediate predecessor ignored")
	}
	if _, err := selectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: b.HeadRaw}); err == nil {
		t.Fatal("no-op masked corrupt predecessor")
	}
	if !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("predecessor refusal changed rows")
	}
}
