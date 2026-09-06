package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Independent synthetic fixture: repeated declarations, three births spanning
// declaration/endpoint origins, two assertions, both hint roles on one original
// side, and one declaration referencing two births. No entity/fact authority.
func disproofSelectionFixture(t *testing.T, st *Store, id string) (identityInputRevision, episodeProposal) {
	t.Helper()
	r := identityInputRevision{Version: 1, EpisodeID: id, OccurredAt: time.Unix(987654321, 123456789).UTC(), Cwd: "/synthetic/界\n", Summary: "independent synthetic", Declarations: []observedDeclaration{{Name: "Orion", Type: "project", Aliases: []string{"O", "O"}, TypeFallback: true}, {Name: "Lyra", Type: "project", Aliases: []string{}}, {Name: "Orion", Type: "tool"}}, Facts: []observedFact{{Src: "Orion", Dst: "Lyra", Relation: "related_to", Fact: "one", Confidence: .7, ValidFrom: "unparsed", Supersedes: &observedSupersedes{Src: "Old", Dst: "Prior", Relation: "related_to"}}, {Src: "Cygnus", Dst: "value", Relation: "status", Fact: "two", Confidence: .2}}}
	p := episodeProposal{Result: episodeResult{Version: 1, EpisodeID: id, Declarations: []episodeDeclaration{}, Assertions: []episodeAssertion{}, Births: []episodeBirth{}}, Outcomes: []identityOutcome{}}
	err := st.AtomicWrite(func(tx *Store) error {
		if err := tx.PutEpisode(Episode{ID: id, OccurredAt: r.OccurredAt}); err != nil {
			return err
		}
		var err error
		p.Result.InputKey, err = putIdentityInput(tx, r)
		if err != nil {
			return err
		}
		for i := range r.Declarations {
			key, err := putIdentityObservation(tx, identityObservation{Version: 1, EpisodeID: id, OccurredAt: r.OccurredAt, Cwd: r.Cwd, Origin: "declaration", Ordinal: i, Declaration: &r.Declarations[i]})
			if err != nil {
				return err
			}
			p.Result.Declarations = append(p.Result.Declarations, episodeDeclaration{Ordinal: i, ObservationKey: key, Kind: "identity-mention", Births: []string{}})
		}
		for i := range r.Facts {
			keys := [2]string{}
			for j, side := range []string{"src", "dst"} {
				keys[j], err = putIdentityObservation(tx, identityObservation{Version: 1, EpisodeID: id, OccurredAt: r.OccurredAt, Cwd: r.Cwd, Origin: "endpoint", Ordinal: i, Side: side, Fact: &r.Facts[i]})
				if err != nil {
					return err
				}
			}
			p.Result.Assertions = append(p.Result.Assertions, episodeAssertion{Ordinal: i, Kind: "deferred", Reason: "assertion-dependency", SourceObservation: keys[0], DestinationObservation: keys[1]})
			ordinal := i
			o := identityOutcome{Version: 1, EpisodeID: id, AssertionOrdinal: &ordinal, Disposition: "deferred", Links: []outcomeLink{{Key: keys[0], Role: "primary-src", Spelling: r.Facts[i].Src, FinalSide: "none", State: "deferred"}, {Key: keys[1], Role: "primary-dst", Spelling: r.Facts[i].Dst, FinalSide: "none", State: "not-materialized"}}}
			if i == 0 {
				o.Links = append(o.Links, outcomeLink{Key: keys[0], Role: "supersedes-src", Spelling: "Old", FinalSide: "none", State: "deferred"}, outcomeLink{Key: keys[0], Role: "supersedes-dst", Spelling: "Prior", FinalSide: "none", State: "deferred"})
			}
			p.Outcomes = append(p.Outcomes, o)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"Orion", "Lyra", "Cygnus"} {
		origin, ordinal, first, role := "declaration", i, "", "declaration"
		if i < 2 {
			first = p.Result.Declarations[i].ObservationKey
		} else {
			origin, ordinal, first, role = "endpoint", 1, p.Result.Assertions[1].SourceObservation, "primary-src"
		}
		b := identityBirth{EpisodeID: id, Occurrence: ordinal, Origin: origin, Name: name, Slug: Slugify(name), CreatedAt: r.OccurredAt}
		p.Result.Births = append(p.Result.Births, episodeBirth{Birth: b, FirstObservation: first})
		o := identityOutcome{Version: 1, EpisodeID: id, Birth: &b, Disposition: "no-assertion", Links: []outcomeLink{{Key: first, Role: role, Spelling: name, FinalSide: "none", State: "not-materialized"}}}
		if i == 0 {
			o.Links = append(o.Links, outcomeLink{Key: p.Result.Declarations[2].ObservationKey, Role: "declaration", Spelling: name, FinalSide: "none", State: "not-materialized"})
		}
		if i == 1 {
			o.Links = append(o.Links, outcomeLink{Key: p.Result.Declarations[0].ObservationKey, Role: "declaration", Spelling: "Orion", FinalSide: "none", State: "not-materialized"})
		}
		p.Outcomes = append(p.Outcomes, o)
	}
	p.Result.Declarations[0].Births = []string{p.Result.Births[1].FirstObservation, p.Result.Births[0].FirstObservation}
	p.Result.Declarations[1].Births = []string{p.Result.Births[1].FirstObservation}
	p.Result.Declarations[2].Births = []string{p.Result.Births[0].FirstObservation}
	return r, p
}

func disproofSelectionPut(st *Store, p episodeProposal, expected episodeHeadExpectation) (episodeSelection, error) {
	var captured episodeSelection
	err := runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
		var err error
		captured, err = putEpisodeSelection(tx, p.Result.EpisodeID, expected, p)
		return err
	})
	return captured, err
}

func disproofSelectionViewRows(t *testing.T, st *Store) map[string][]byte {
	t.Helper()
	rows := map[string][]byte{}
	if err := st.view(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			value, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			rows[string(it.Item().KeyCopy(nil))] = value
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestSelectionIndependentPermutationOwnership(t *testing.T) {
	st := openTemp(t)
	_, p := disproofSelectionFixture(t, st, "independent-order")
	p.Outcomes[2].Materialization = []byte("{\"note\":\"synthetic \\u754c\"}")
	beforeProposal := dpJSON(t, p)
	first, err := disproofSelectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeProposal, dpJSON(t, p)) {
		t.Fatal("caller proposal mutated")
	}
	if first.Counts.Deferred != 2 || first.Counts.BirthNoAssertion != 3 {
		t.Fatal("wrong counts")
	}
	baseline := dpRows(t, st)
	rng := rand.New(rand.NewSource(8472))
	for i := 0; i < 25; i++ {
		rng.Shuffle(len(p.Outcomes), func(i, j int) { p.Outcomes[i], p.Outcomes[j] = p.Outcomes[j], p.Outcomes[i] })
		for j := range p.Outcomes {
			links := p.Outcomes[j].Links
			rng.Shuffle(len(links), func(a, b int) { links[a], links[b] = links[b], links[a] })
		}
		p.Result.Declarations[0].Births[0], p.Result.Declarations[0].Births[1] = p.Result.Declarations[0].Births[1], p.Result.Declarations[0].Births[0]
		for j := range p.Result.Births {
			p.Result.Births[j].Birth.CreatedAt = p.Result.Births[j].Birth.CreatedAt.In(time.FixedZone("seconds", 321))
		}
		for j := range p.Outcomes {
			if p.Outcomes[j].Birth != nil {
				p.Outcomes[j].Birth.CreatedAt = p.Outcomes[j].Birth.CreatedAt.In(time.FixedZone("other", -119))
			}
		}
		got, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: first.HeadRaw})
		if err != nil || !reflect.DeepEqual(first, got) || !reflect.DeepEqual(baseline, dpRows(t, st)) {
			t.Fatalf("permutation %d churned: %v", i, err)
		}
	}
	// Mutate every caller-owned pointer/slice and staged report buffer after staging.
	err = runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
		got, err := putEpisodeSelection(tx, p.Result.EpisodeID, episodeHeadExpectation{Exists: true, Raw: first.HeadRaw}, p)
		if err != nil {
			return err
		}
		got.HeadRaw[0] = 0
		got.ResultRaw[0] = 0
		got.Result.Declarations[0].Births[0] = "changed"
		p.Result.Declarations[0].Births[0] = "changed"
		for i := range p.Outcomes {
			p.Outcomes[i].Links[0].Spelling = "changed"
			if p.Outcomes[i].Birth != nil {
				p.Outcomes[i].Birth.Name = "changed"
			}
			if p.Outcomes[i].AssertionOrdinal != nil {
				*p.Outcomes[i].AssertionOrdinal = 999
			}
			if len(p.Outcomes[i].Materialization) > 0 {
				p.Outcomes[i].Materialization[0] = 0
			}
		}
		return nil
	})
	if err != nil || !reflect.DeepEqual(baseline, dpRows(t, st)) {
		t.Fatal("mutated caller/report aliases store", err)
	}
}

func TestSelectionIndependentCoveragePoisonAndPreflight(t *testing.T) {
	cases := []string{"decl-valid-wrong-slot", "assert-valid-wrong-slot", "swap-declarations", "swap-assertions", "swap-births", "nil-decl-refs", "nil-outcomes", "duplicate-slug", "duplicate-first", "birth-wrong-time", "birth-wrong-origin", "birth-wrong-ordinal", "birth-wrong-name", "first-wrong-side", "missing-birth", "missing-ref", "extra-ref", "duplicate-ref", "deferred-ref", "unknown-decl", "unknown-assert", "unknown-reason", "committed-reason", "missing-hint-src", "missing-hint-dst", "extra-hint-wrong-revision", "extra-birth-wrong-revision", "duplicate-link-role", "duplicate-address-different-outcome", "extra-valid-birth-outcome", "extra-outcome-nonassertion", "missing-outcome", "incoming-result-predecessor", "incoming-birth-key", "incoming-outcome-predecessor", "expected-absent-empty", "expected-present-nil", "corrupt-occupied-outcome"}
	for _, mode := range cases {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			r, p := disproofSelectionFixture(t, st, "independent-refusal")
			expected := episodeHeadExpectation{}
			switch mode {
			case "decl-valid-wrong-slot":
				p.Result.Declarations[0].ObservationKey = p.Result.Declarations[1].ObservationKey
			case "assert-valid-wrong-slot":
				p.Result.Assertions[0].SourceObservation = p.Result.Assertions[1].SourceObservation
			case "swap-declarations":
				p.Result.Declarations[0], p.Result.Declarations[1] = p.Result.Declarations[1], p.Result.Declarations[0]
			case "swap-assertions":
				p.Result.Assertions[0], p.Result.Assertions[1] = p.Result.Assertions[1], p.Result.Assertions[0]
			case "swap-births":
				p.Result.Births[0], p.Result.Births[1] = p.Result.Births[1], p.Result.Births[0]
			case "nil-decl-refs":
				p.Result.Declarations[2].Births = nil
			case "nil-outcomes":
				p.Outcomes = nil
			case "duplicate-slug":
				p.Result.Births[1].Birth = p.Result.Births[0].Birth
				b := p.Result.Births[1].Birth
				p.Outcomes[3].Birth = &b
			case "duplicate-first":
				p.Result.Births[1].FirstObservation = p.Result.Births[0].FirstObservation
			case "birth-wrong-time":
				p.Result.Births[0].Birth.CreatedAt = r.OccurredAt.Add(time.Nanosecond)
				b := p.Result.Births[0].Birth
				p.Outcomes[2].Birth = &b
			case "birth-wrong-origin":
				p.Result.Births[0].Birth.Origin = "endpoint"
				b := p.Result.Births[0].Birth
				p.Outcomes[2].Birth = &b
			case "birth-wrong-ordinal":
				p.Result.Births[0].Birth.Occurrence = 1
				b := p.Result.Births[0].Birth
				p.Outcomes[2].Birth = &b
			case "birth-wrong-name":
				p.Result.Births[0].Birth.Name = "Other"
				p.Result.Births[0].Birth.Slug = "other"
				b := p.Result.Births[0].Birth
				p.Outcomes[2].Birth = &b
			case "first-wrong-side":
				p.Result.Births[2].FirstObservation = p.Result.Assertions[1].DestinationObservation
			case "missing-birth":
				p.Result.Births = p.Result.Births[:2]
			case "missing-ref":
				p.Result.Declarations[2].Births = []string{}
			case "extra-ref":
				p.Result.Declarations[2].Births = append(p.Result.Declarations[2].Births, p.Result.Births[1].FirstObservation)
			case "duplicate-ref":
				p.Result.Declarations[0].Births = append(p.Result.Declarations[0].Births, p.Result.Declarations[0].Births[0])
			case "deferred-ref":
				p.Result.Declarations[2].Kind = "deferred-identity"
			case "unknown-decl":
				p.Result.Declarations[2].Kind = "accepted"
			case "unknown-assert":
				p.Result.Assertions[0].Kind = "ingested"
			case "unknown-reason":
				p.Result.Assertions[0].Reason = "unknown"
			case "committed-reason":
				p.Result.Assertions[0].Kind = "committed"
			case "missing-hint-src":
				p.Outcomes[0].Links = append(p.Outcomes[0].Links[:2], p.Outcomes[0].Links[3])
			case "missing-hint-dst":
				p.Outcomes[0].Links = p.Outcomes[0].Links[:3]
			case "extra-hint-wrong-revision", "extra-birth-wrong-revision":
				f := r.Facts[0]
				f.Fact = "another complete revision"
				var key string
				err := st.AtomicWrite(func(tx *Store) error {
					var err error
					key, err = putIdentityObservation(tx, identityObservation{Version: 1, EpisodeID: r.EpisodeID, Origin: "endpoint", Ordinal: 0, Side: "dst", Cwd: r.Cwd, OccurredAt: r.OccurredAt, Fact: &f})
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				if mode == "extra-hint-wrong-revision" {
					p.Outcomes[0].Links = append(p.Outcomes[0].Links, outcomeLink{Key: key, Role: "supersedes-src", Spelling: "Old", State: "deferred", FinalSide: "none"})
				} else {
					p.Outcomes[2].Links = append(p.Outcomes[2].Links, outcomeLink{Key: key, Role: "primary-dst", Spelling: "Lyra", State: "not-materialized", FinalSide: "none"})
				}
			case "duplicate-link-role":
				p.Outcomes[0].Links = append(p.Outcomes[0].Links, p.Outcomes[0].Links[0])
			case "duplicate-address-different-outcome":
				o := p.Outcomes[1]
				o.Links = append([]outcomeLink{}, o.Links...)
				o.Links[1].State = "deferred"
				p.Outcomes = append(p.Outcomes, o)
			case "extra-valid-birth-outcome":
				o := p.Outcomes[2]
				b := *o.Birth
				b.Occurrence = 2
				o.Birth = &b
				o.Links = o.Links[1:]
				p.Outcomes = append(p.Outcomes, o)
			case "extra-outcome-nonassertion":
				p.Result.Assertions[1].Kind = "non-assertion"
				p.Result.Assertions[1].Reason = "two-values"
			case "missing-outcome":
				p.Outcomes = p.Outcomes[1:]
			case "incoming-result-predecessor":
				p.Result.Predecessor = episodeResultPrefix(r.EpisodeID) + strings.Repeat("a", 64)
			case "incoming-birth-key":
				p.Result.Births[0].OutcomeKey = "injected"
			case "incoming-outcome-predecessor":
				p.Outcomes[0].Predecessor = "injected"
			case "expected-absent-empty":
				expected.Raw = []byte{}
			case "expected-present-nil":
				expected.Exists = true
			case "corrupt-occupied-outcome":
				var key string
				err := st.AtomicWrite(func(tx *Store) error {
					basis, err := newEpisodeBasis(tx, p.Result.InputKey, r.EpisodeID)
					if err != nil {
						return err
					}
					o, err := basis.outcome(p.Outcomes[1], false)
					if err != nil {
						return err
					}
					_, key, err = encodeIdentityOutcome(o)
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				dpSet(t, st, key, []byte("occupied secret synthetic"))
			}
			before := dpRows(t, st)
			err := runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
				got, local := putEpisodeSelection(tx, r.EpisodeID, expected, p)
				if !errors.Is(local, errEpisodeSelection) || !reflect.DeepEqual(got, episodeSelection{}) {
					t.Fatalf("bad proposal accepted or unsafe result: %v", local)
				}
				if local.Error() != errEpisodeSelection.Error() {
					t.Fatal("value-bearing error escaped")
				}
				// Verify entire preflight before writes, inside still-open transaction.
				if !reflect.DeepEqual(before, disproofSelectionViewRows(t, tx)) {
					t.Fatal("preflight refusal staged partial rows")
				}
				return nil
			})
			if !errors.Is(err, errEpisodeSelection) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("caught refusal committed", err)
			}
		})
	}
}

func TestSelectionIndependentMalformedSelectedRecords(t *testing.T) {
	for _, target := range []string{"head", "result", "input", "observation", "outcome", "episode"} {
		for _, mutation := range []string{"empty", "unknown-field", "duplicate-field", "wrong-key-content"} {
			t.Run(target+"/"+mutation, func(t *testing.T) {
				st := openTemp(t)
				_, p := disproofSelectionFixture(t, st, "independent-malformed")
				a, err := disproofSelectionPut(st, p, episodeHeadExpectation{})
				if err != nil {
					t.Fatal(err)
				}
				key := map[string]string{"head": episodeHeadKey(p.Result.EpisodeID), "result": a.Head.ResultKey, "input": p.Result.InputKey, "observation": p.Result.Assertions[0].SourceObservation, "outcome": a.Result.Assertions[0].OutcomeKey, "episode": prefixEpisode + p.Result.EpisodeID}[target]
				raw := dpRows(t, st)[key]
				switch mutation {
				case "empty":
					raw = []byte{}
				case "unknown-field":
					if target == "episode" {
						raw = bytes.Replace(raw, []byte(`"id":`), []byte(`"ID":"wrong","id":`), 1)
					} else {
						raw = append([]byte(`{"synthetic_unknown":true,`), raw[1:]...)
					}
				case "duplicate-field":
					field := `"version":1,`
					if target == "episode" {
						field = `"id":"independent-malformed",`
					}
					raw = append([]byte("{"+field), raw[1:]...)
				case "wrong-key-content":
					raw = bytes.ReplaceAll(raw, []byte("independent-malformed"), []byte("independent-different"))
				}
				dpSet(t, st, key, raw)
				before := dpRows(t, st)
				if _, err := readEpisodeSelection(st, p.Result.EpisodeID); err == nil {
					t.Fatal("malformed selected state read")
				}
				if _, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw}); err == nil {
					t.Fatal("malformed no-op accepted")
				}
				if !reflect.DeepEqual(before, dpRows(t, st)) {
					t.Fatal("malformed state rewritten")
				}
			})
		}
	}
}

func TestSelectionIndependentCaughtThenCachedAndActualCommitConflict(t *testing.T) {
	for _, mode := range []string{"caught-then-cached", "callback-error", "commit-conflict", "panic"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			_, p := disproofSelectionFixture(t, st, "independent-atomic")
			a, err := disproofSelectionPut(st, p, episodeHeadExpectation{})
			if err != nil {
				t.Fatal(err)
			}
			before := dpRows(t, st)
			p.Result.Assertions[1].Reason = "identity-dependency"
			var captured episodeSelection
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				err = runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
					var local error
					captured, local = putEpisodeSelection(tx, p.Result.EpisodeID, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw}, p)
					if local != nil {
						return local
					}
					switch mode {
					case "caught-then-cached":
						bad := p
						bad.Result.Predecessor = a.Head.ResultKey
						if _, err := putEpisodeSelection(tx, p.Result.EpisodeID, episodeHeadExpectation{Exists: true, Raw: captured.HeadRaw}, bad); err == nil {
							t.Fatal("bad call accepted")
						}
						if _, err := putEpisodeSelection(tx, p.Result.EpisodeID, episodeHeadExpectation{Exists: true, Raw: captured.HeadRaw}, p); err == nil {
							t.Fatal("cached no-op escaped poison")
						}
						return nil
					case "callback-error":
						return errors.New("synthetic outer error")
					case "panic":
						panic("synthetic outer panic")
					default:
						return st.db.Update(func(other *badger.Txn) error { return other.Set([]byte(episodeHeadKey(p.Result.EpisodeID)), a.HeadRaw) })
					}
				})
			}()
			if mode == "panic" {
				if recovered != "synthetic outer panic" {
					t.Fatal("panic missing")
				}
			} else if err == nil || recovered != nil {
				t.Fatal("outer failure missing", err)
			}
			if mode == "commit-conflict" && !errors.Is(err, badger.ErrConflict) {
				t.Fatal("not real conflict", err)
			}
			if !captured.Selected || captured.Head.Revision != 2 {
				t.Fatal("staged data fixture not captured")
			}
			if !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("failure changed durable rows")
			}
		})
	}
}

func TestSelectionIndependentChunksAndHistoricalBoundaries(t *testing.T) {
	st := openTemp(t)
	id := strings.Repeat("界\"\n\\", 1400)
	_, p := disproofSelectionFixture(t, st, id)
	a, err := disproofSelectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	token, err := readEpisodeSelectionToken(st, id, 512)
	if err != nil || token.ResultKey != a.Head.ResultKey {
		t.Fatal("bounded token failed", err)
	}
	if len(dpJSON(t, token)) > 512 {
		t.Fatal("token overflow")
	}
	for _, budget := range []int{512, 513, 1024, 24576} {
		var joined []byte
		offset, chunks := 0, 0
		for {
			chunk, err := readEpisodeResultChunk(st, id, token.ResultKey, offset, budget)
			if err != nil || chunk.Offset != offset || len(dpJSON(t, chunk)) > budget {
				t.Fatalf("chunk budget=%d offset=%d: %v", budget, offset, err)
			}
			joined = append(joined, chunk.Data...)
			chunks++
			if budget == 512 && chunks == 1 {
				p.Result.Assertions[1].Reason = "identity-dependency"
				b, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
				if err != nil || b.Head.Revision != 2 {
					t.Fatal("head transition", err)
				}
			}
			if chunk.NextOffset == -1 {
				break
			}
			if chunk.NextOffset != offset+len(chunk.Data) || chunk.NextOffset <= offset {
				t.Fatal("lost progress")
			}
			offset = chunk.NextOffset
		}
		if !bytes.Equal(joined, a.ResultRaw) {
			t.Fatal("historical pin changed")
		}
	}
	for _, args := range [][2]int{{-1, 512}, {len(a.ResultRaw) + 1, 512}, {0, 511}, {0, 24577}, {int(^uint(0) >> 1), 512}} {
		if _, err := readEpisodeResultChunk(st, id, token.ResultKey, args[0], args[1]); err == nil {
			t.Fatal("invalid chunk arguments accepted")
		}
	}
	if _, err := readEpisodeResultChunk(st, "wrong episode", token.ResultKey, 0, 512); err == nil {
		t.Fatal("cross episode chunk accepted")
	}
	end, err := readEpisodeResultChunk(st, id, token.ResultKey, len(a.ResultRaw), 512)
	if err != nil || end.NextOffset != -1 || len(end.Data) != 0 {
		t.Fatal("EOF wrong")
	}
}

func TestSelectionIndependentLocalHistoryAndMaxCounter(t *testing.T) {
	st := openTemp(t)
	r, p := disproofSelectionFixture(t, st, "independent-local")
	a, err := disproofSelectionPut(st, p, episodeHeadExpectation{})
	if err != nil {
		t.Fatal(err)
	}
	p.Result.Assertions[1].Reason = "identity-dependency"
	b, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
	if err != nil {
		t.Fatal(err)
	}
	p.Result.Assertions[1].Reason = "assertion-dependency"
	c, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: b.HeadRaw})
	if err != nil {
		t.Fatal(err)
	}
	// Head count deliberately does not prove history length; the initial result
	// is two links away and may be unavailable while current/immediate remain valid.
	dpSet(t, st, a.Head.ResultKey, []byte("corrupt deeper history"))
	c.Head.Revision = ^uint64(0)
	raw, err := encodeEpisodeHead(c.Head)
	if err != nil {
		t.Fatal(err)
	}
	dpSet(t, st, episodeHeadKey(r.EpisodeID), raw)
	got, err := readEpisodeSelection(st, r.EpisodeID)
	if err != nil || got.Head.Revision != ^uint64(0) {
		t.Fatal("reader claimed recursive/full-count proof", err)
	}
	if _, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: raw}); err != nil {
		t.Fatal("max counter no-op refused", err)
	}
	p.Result.Assertions[1].Reason = "identity-dependency"
	before := dpRows(t, st)
	if _, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: raw}); err == nil {
		t.Fatal("overflow accepted")
	}
	if !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("overflow wrote")
	}
	dpSet(t, st, b.Head.ResultKey, []byte("corrupt immediate"))
	if _, err := readEpisodeSelection(st, r.EpisodeID); err == nil {
		t.Fatal("immediate predecessor ignored")
	}
}

func TestSelectionIndependentAdopterExactPrefixes(t *testing.T) {
	for _, prefix := range []string{"io-episode:", "io-head:"} {
		for i, value := range [][]byte{nil, {}, []byte("malformed synthetic")} {
			t.Run(fmt.Sprintf("%s/%d", prefix, i), func(t *testing.T) {
				st := openTemp(t)
				_, manifest := adoptionFixture(t, st)
				// dpSet(nil) means deletion; use Set directly to establish
				// a present zero-length value for both nil and empty inputs.
				key := prefix + "invalid:key"
				if err := st.db.Update(func(tx *badger.Txn) error { return tx.Set([]byte(key), value) }); err != nil {
					t.Fatal(err)
				}
				if _, present := dpRows(t, st)[key]; !present {
					t.Fatal("fixture row is absent")
				}
				assertAdoptionRefusal(t, st, manifest, false)
				assertAdoptionRefusal(t, st, manifest, true)
			})
		}
	}
}

// Compile-time signatures expose no injected finalizer/body or success flag.
var _ func(*Store, string, episodeHeadExpectation, episodeProposal) (episodeSelection, error) = putEpisodeSelection
var _ = json.Valid

func TestSelectionIndependentChangedStagingOwnsBuffers(t *testing.T) {
	st := openTemp(t)
	_, p := disproofSelectionFixture(t, st, "independent-own-staged")
	p.Outcomes[2].Materialization = []byte(`{"synthetic":"intact"}`)
	var expected episodeSelection
	err := runSerializedIdentityAdmission(st, func(*Store) error { return nil }, func(tx *Store) error {
		got, err := putEpisodeSelection(tx, p.Result.EpisodeID, episodeHeadExpectation{}, p)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(dpJSON(t, got), &expected); err != nil {
			return err
		}
		got.HeadRaw[0] = 0
		got.ResultRaw[0] = 0
		got.Result.Births[0].Birth.Name = "changed"
		got.Result.Declarations[0].Births[0] = "changed"
		got.Result.Assertions[0].SourceObservation = "changed"
		p.Result.Births[0].Birth.Name = "changed"
		p.Result.Declarations[0].Births[0] = "changed"
		p.Outcomes[2].Materialization[0] = 0
		p.Outcomes[2].Birth.Name = "changed"
		*p.Outcomes[0].AssertionOrdinal = 999
		p.Outcomes[0].Links[0].Spelling = "changed"
		inside, _, err := readEpisodeSelectionTxn(tx, p.Result.EpisodeID)
		if err != nil || !reflect.DeepEqual(inside, expected) {
			t.Fatal("caller/report changed staged bytes", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := readEpisodeSelection(st, p.Result.EpisodeID)
	if err != nil || !reflect.DeepEqual(got, expected) {
		t.Fatal("caller/report changed committed bytes", err)
	}
}

func TestSelectionIndependentRealStagingErrors(t *testing.T) {
	for _, mode := range []string{"value-limit", "transaction-capacity"} {
		t.Run(mode, func(t *testing.T) {
			db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(8192))
			if err != nil {
				t.Fatal(err)
			}
			st := &Store{db: db}
			defer st.Close()
			_, p := disproofSelectionFixture(t, st, "independent-storage")
			if mode == "value-limit" {
				p.Outcomes[2].Materialization = dpJSON(t, map[string]string{"synthetic_private": strings.Repeat("secret-canary ", 2000)})
			}
			before := dpRows(t, st)
			err = runSerializedIdentityAdmission(st, func(tx *Store) error {
				if mode == "transaction-capacity" {
					for i := 0; i < 10000; i++ {
						err := tx.txn.Set([]byte(fmt.Sprintf("synthetic-capacity:%06d", i)), bytes.Repeat([]byte{'x'}, 2000))
						if errors.Is(err, badger.ErrTxnTooBig) {
							return nil
						}
						if err != nil {
							return err
						}
					}
					t.Fatal("did not fill synthetic transaction")
				}
				return nil
			}, func(tx *Store) error {
				got, local := putEpisodeSelection(tx, p.Result.EpisodeID, episodeHeadExpectation{}, p)
				if !errors.Is(local, errEpisodeSelection) || !reflect.DeepEqual(got, episodeSelection{}) {
					t.Fatal("staging failure not sanitized", local)
				}
				if strings.Contains(local.Error(), "secret") || strings.Contains(local.Error(), "canary") {
					t.Fatal("staging error leaked raw data")
				}
				if mode == "transaction-capacity" && !errors.Is(local, badger.ErrTxnTooBig) {
					t.Fatal("capacity classification lost")
				}
				return nil
			})
			if !errors.Is(err, errEpisodeSelection) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("caught staging failure committed", err)
			}
		})
	}
}

func TestSelectionIndependentSupportedDescriptionIsNotOwnership(t *testing.T) {
	st := openTemp(t)
	_, p := disproofSelectionFixture(t, st, "independent-description")
	for i := 2; i < len(p.Outcomes); i++ {
		record, _, err := identityBirthRecord(*p.Outcomes[i].Birth)
		if err != nil {
			t.Fatal(err)
		}
		p.Outcomes[i].Disposition = "supported"
		p.Outcomes[i].Generation = record.ID
		p.Outcomes[i].Materialization = []byte(`{"synthetic_description":true}`)
	}
	a, err := disproofSelectionPut(st, p, episodeHeadExpectation{})
	if err != nil || a.Counts.BirthSupported != 3 || a.Counts.BirthNoAssertion != 0 {
		t.Fatal("descriptive supported count failed", err)
	}
	for _, birth := range p.Result.Births {
		if _, err := st.GetEntity(birth.Birth.Slug); !errors.Is(err, ErrNotFound) {
			t.Fatal("selector manufactured live identity")
		}
	}
	// Registration-only enumeration removes all formerly described births. The
	// selector cannot prove that producer choice wrong; integration must do so.
	p.Result.Births = []episodeBirth{}
	for i := range p.Result.Declarations {
		p.Result.Declarations[i].Births = []string{}
	}
	p.Outcomes = p.Outcomes[:2]
	b, err := disproofSelectionPut(st, p, episodeHeadExpectation{Exists: true, Raw: a.HeadRaw})
	if err != nil || b.Head.Revision != 2 || b.Counts.BirthSupported != 0 {
		t.Fatal("fresh-only inventory limitation changed", err)
	}
	if raw := dpRows(t, st)[a.Head.ResultKey]; !bytes.Equal(raw, a.ResultRaw) {
		t.Fatal("omission removed historical description")
	}
}
