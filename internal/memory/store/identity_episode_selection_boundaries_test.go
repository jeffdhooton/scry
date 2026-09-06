package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestEpisodeSelectionCoverageAndLinkRefusals(t *testing.T) {
	for _, mode := range []string{"missing-declaration", "wrong-declaration-slot", "missing-assertion", "wrong-assertion-slot", "wrong-primary-side", "missing-outcome", "extra-outcome", "duplicate-outcome", "missing-birth-outcome", "missing-forward-birth", "missing-reverse-birth", "duplicate-birth-ref", "value-with-birth", "duplicate-birth", "wrong-first", "wrong-birth-name", "missing-primary", "unknown-reason", "nonassertion-with-outcome", "proposal-predecessor", "proposal-outcome-key", "nil-array", "mixed-birth-link"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			input, p := selectionFixture(t, st, "coverage", true)
			switch mode {
			case "missing-declaration":
				p.Result.Declarations = []episodeDeclaration{}
			case "wrong-declaration-slot":
				p.Result.Declarations[0].Ordinal = 1
			case "missing-assertion":
				p.Result.Assertions = []episodeAssertion{}
			case "wrong-assertion-slot":
				p.Result.Assertions[0].Ordinal = 1
			case "wrong-primary-side":
				p.Result.Assertions[0].SourceObservation = p.Result.Assertions[0].DestinationObservation
			case "missing-outcome":
				p.Outcomes = p.Outcomes[1:]
			case "extra-outcome":
				o := p.Outcomes[0]
				ordinal := 999
				o.AssertionOrdinal = &ordinal
				p.Outcomes = append(p.Outcomes, o)
			case "duplicate-outcome":
				p.Outcomes = append(p.Outcomes, p.Outcomes[0])
			case "missing-birth-outcome":
				p.Outcomes = p.Outcomes[:1]
			case "missing-forward-birth":
				p.Result.Declarations[0].Births = []string{}
			case "missing-reverse-birth":
				p.Outcomes[1].Links = p.Outcomes[1].Links[:1]
			case "duplicate-birth-ref":
				p.Result.Declarations[0].Births = append(p.Result.Declarations[0].Births, p.Result.Declarations[0].Births[0])
			case "value-with-birth":
				p.Result.Declarations[0].Kind = "value-mention"
			case "duplicate-birth":
				p.Result.Births = append(p.Result.Births, p.Result.Births[0])
			case "wrong-first":
				p.Result.Births[0].FirstObservation = p.Result.Assertions[0].SourceObservation
			case "wrong-birth-name":
				p.Result.Births[0].Birth.Name = "Other"
				p.Result.Births[0].Birth.Slug = "other"
			case "missing-primary":
				p.Outcomes[0].Links = p.Outcomes[0].Links[1:]
			case "unknown-reason":
				p.Result.Assertions[0].Reason = "success-anyway"
			case "nonassertion-with-outcome":
				p.Result.Assertions[0].Kind = "non-assertion"
				p.Result.Assertions[0].Reason = "two-values"
			case "proposal-predecessor":
				p.Outcomes[0].Predecessor = "caller-selected"
			case "proposal-outcome-key":
				p.Result.Assertions[0].OutcomeKey = "caller-selected"
			case "nil-array":
				p.Result.Births = nil
			case "mixed-birth-link":
				input.Facts[0].Fact = "different revision at same ordinal"
				_, _, facts := selectionInput(t, st, input)
				p.Outcomes[1].Links[0].Key = facts[0][0]
			}
			before := dpRows(t, st)
			var staged episodeSelection
			err := runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
				var err error
				staged, err = putEpisodeSelection(tx, "coverage", episodeHeadExpectation{}, p)
				if err == nil {
					t.Fatal("invalid proposal accepted")
				}
				return nil // caught validation error must still poison
			})
			if err == nil || !reflect.DeepEqual(staged, episodeSelection{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("invalid proposal escaped refusal")
			}
		})
	}
}

func TestEpisodeSelectionClosedDispositionCounts(t *testing.T) {
	for _, kind := range []string{"non-assertion", "unresolved"} {
		reasons := []string{"empty-relation", "two-values"}
		if kind == "unresolved" {
			reasons = []string{"missing-source", "missing-destination"}
		}
		for _, reason := range reasons {
			t.Run(reason, func(t *testing.T) {
				st := openTemp(t)
				_, p := selectionFixture(t, st, "counts", false)
				p.Outcomes = []identityOutcome{}
				p.Result.Assertions[0].Kind = kind
				p.Result.Assertions[0].Reason = reason
				p.Result.Declarations[0].Kind = "deferred-identity"
				got, err := selectionPut(st, p, episodeHeadExpectation{})
				if err != nil || got.Counts.Committed != 0 || got.Counts.Deferred != 0 || got.Counts.NonAssertion+got.Counts.Unresolved != 1 || got.Counts.DeclarationDeferred != 1 {
					t.Fatal("closed description miscounted", err)
				}
				if kind == "unresolved" && got.Counts.Unresolved != 1 {
					t.Fatal("unresolved silently treated as completed")
				}
			})
		}
	}
}

func TestEpisodeSelectionPhasePoisonAndMalformedCurrent(t *testing.T) {
	for _, mode := range []string{"root", "body", "ordinary", "closed", "prior-poison", "cached-poison", "bad-head", "bad-result", "bad-input", "bad-outcome", "noncanonical-selected-links", "expected-empty-present"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			_, p := selectionFixture(t, st, "phase", false)
			var expected episodeHeadExpectation
			if mode == "cached-poison" || mode == "bad-head" || mode == "bad-result" || mode == "bad-input" || mode == "bad-outcome" || mode == "noncanonical-selected-links" {
				first, err := selectionPut(st, p, episodeHeadExpectation{})
				if err != nil {
					t.Fatal(err)
				}
				expected = episodeHeadExpectation{Exists: true, Raw: first.HeadRaw}
				switch mode {
				case "bad-head":
					dpSet(t, st, episodeHeadKey("phase"), append(bytes.Clone(first.HeadRaw), ' '))
				case "bad-result":
					dpSet(t, st, first.Head.ResultKey, []byte("{}"))
				case "bad-input":
					dpSet(t, st, p.Result.InputKey, []byte("{}"))
				case "bad-outcome":
					dpSet(t, st, first.Result.Assertions[0].OutcomeKey, []byte("{}"))
				case "noncanonical-selected-links":
					// Point a fully canonical result/head at an old-valid but
					// selector-noncanonical outcome permutation.
					var o identityOutcome
					json.Unmarshal(dpRows(t, st)[first.Result.Assertions[0].OutcomeKey], &o)
					o.Links[0], o.Links[1] = o.Links[1], o.Links[0]
					oraw, okey, err := encodeIdentityOutcome(o)
					if err != nil {
						t.Fatal(err)
					}
					dpSet(t, st, okey, oraw)
					first.Result.Assertions[0].OutcomeKey = okey
					rraw, rkey, err := encodeEpisodeResult(first.Result)
					if err != nil {
						t.Fatal(err)
					}
					dpSet(t, st, rkey, rraw)
					first.Head.ResultKey = rkey
					hraw, err := encodeEpisodeHead(first.Head)
					if err != nil {
						t.Fatal(err)
					}
					dpSet(t, st, episodeHeadKey("phase"), hraw)
					expected.Raw = hraw
				}
			}
			if mode == "expected-empty-present" {
				expected = episodeHeadExpectation{Exists: true, Raw: []byte{}}
			}
			before := dpRows(t, st)
			prior := errors.New("synthetic first poison")
			call := func(tx *Store) error { _, err := putEpisodeSelection(tx, "phase", expected, p); return err }
			var err error
			switch mode {
			case "root":
				err = call(st)
			case "body":
				err = runSerializedIdentityAdmission(st, func(tx *Store) error {
					if call(tx) == nil {
						t.Fatal("body accepted")
					}
					return nil
				}, func(*Store) error { t.Fatal("poisoned finalizer ran"); return nil })
			case "ordinary":
				err = st.AtomicWrite(func(tx *Store) error {
					if call(tx) == nil {
						t.Fatal("ordinary accepted")
					}
					return nil
				})
			case "closed":
				var saved *Store
				_ = runSerializedIdentityAdmission(st, func(tx *Store) error { saved = tx; return nil }, func(*Store) error { return nil })
				err = call(saved)
			default:
				err = runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
					if mode == "prior-poison" || mode == "cached-poison" {
						tx.poisonAdmission(prior)
					}
					if call(tx) == nil {
						t.Fatal("invalid current accepted")
					}
					return nil
				})
			}
			if err == nil || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("invalid phase/current changed state")
			}
			if (mode == "prior-poison" || mode == "cached-poison") && err != prior {
				t.Fatal("prior poison changed")
			}
		})
	}
}
