package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func birthFixture(t *testing.T) (identityInputRevision, string, []byte) {
	t.Helper()
	r := identityInputRevision{Version: 1, EpisodeID: "birth-episode", OccurredAt: time.Unix(123456, 789).UTC(), Cwd: "/synthetic", Summary: "synthetic revision",
		Declarations: []observedDeclaration{{Name: "Atlas", Type: "project", Description: "first", Aliases: []string{"A", "A"}, TypeFallback: true}, {Name: "Borealis", Type: "service", Aliases: []string{}}, {Name: "Atlas!", Type: "tool", Description: "different proposed metadata"}},
		Facts:        []observedFact{{Src: "Atlas", Relation: "used_by", Dst: "Borealis", Fact: "synthetic", ValidFrom: "unparsed date", Confidence: 0.75, Supersedes: &observedSupersedes{Src: "Atlas", Relation: "uses", Dst: "Old"}}, {Src: "Atlas", Relation: "related_to", Dst: "Atlas", Fact: "same names different sides"}}}
	raw, key, err := encodeIdentityInput(r)
	if err != nil {
		t.Fatal(err)
	}
	return r, key, raw
}

func birthObservation(t *testing.T, r identityInputRevision, origin string, ordinal int, side string) (string, []byte) {
	t.Helper()
	o := identityObservation{Version: 1, EpisodeID: r.EpisodeID, OccurredAt: r.OccurredAt, Cwd: r.Cwd, Origin: origin, Ordinal: ordinal, Side: side}
	if origin == "declaration" {
		o.Declaration = &r.Declarations[ordinal]
	} else {
		o.Fact = &r.Facts[ordinal]
	}
	raw, key, err := encodeIdentityObservation(o)
	if err != nil {
		t.Fatal(err)
	}
	return key, raw
}

func birthEntity(t *testing.T, h *registeredBirth) ([]byte, []byte) {
	t.Helper()
	e := Entity{Slug: h.birth.Slug, Name: h.birth.Name, Type: "project", CreatedAt: h.birth.CreatedAt, LastSeen: h.birth.CreatedAt, Description: "synthetic staged metadata"}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(prefixEntity + e.Slug), raw
}

func TestBirthRegistrationExactDerivationAndOwnedReports(t *testing.T) {
	st := openTemp(t)
	input, key, raw := birthFixture(t)
	original := bytes.Clone(raw)
	firstKey, firstRaw := birthObservation(t, input, "declaration", 0, "")
	firstCopy := bytes.Clone(firstRaw)
	otherKey, otherRaw := birthObservation(t, input, "declaration", 2, "")
	primaryKey, primaryRaw := birthObservation(t, input, "endpoint", 0, "dst")
	var saved *birthRegistry
	var handle *registeredBirth
	events := 0
	st.SetObserver(func(Event) { events++ })
	report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
		saved = r
		for i := range raw {
			raw[i] = 0
		}
		d, err := r.register(firstKey, firstRaw)
		if err != nil {
			return err
		}
		handle = d.Handle
		if d.Kind != "registered" || handle == nil || handle.birth.Name != "Atlas" || handle.birth.CreatedAt != input.OccurredAt {
			t.Fatal("wrong derived birth")
		}
		for i := range d.Observation.Raw {
			d.Observation.Raw[i] = 0
		}
		k, v := birthEntity(t, handle)
		if err := r.putIdentity(handle.birth.Slug, k, v); err != nil {
			return err
		}
		if err := r.putIdentity("atlas", []byte("al:atlas"), []byte("atlas")); err != nil {
			return err
		}
		retry, err := r.register(firstKey, firstRaw)
		if err != nil || retry.Handle != handle {
			t.Fatal("exact retry after creation changed handle")
		}
		if err := r.link(handle, otherKey, otherRaw, "declaration"); err != nil {
			return err
		}
		if err := r.link(handle, otherKey, otherRaw, "declaration"); err != nil {
			return err
		}
		if err := r.link(handle, primaryKey, primaryRaw, "supersedes-dst"); err != nil {
			return err
		}
		dst, err := r.register(primaryKey, primaryRaw)
		if err != nil {
			return err
		}
		if dst.Handle.birth.Name != "Borealis" || dst.Handle.birth.Origin != "endpoint" || dst.Handle.first.Role != "primary-dst" {
			t.Fatal("original inverse side changed")
		}
		for i := range firstRaw {
			firstRaw[i] = 0
		}
		return nil
	})
	if err != nil || events != 0 || len(report.Candidates) != 2 || !bytes.Equal(report.RevisionRaw, original) {
		t.Fatal("registration report failed", err)
	}
	a, b := report.Candidates[0], report.Candidates[1]
	if !a.Created || !a.FinalPresent || b.Created || b.FinalPresent || len(a.Mentions) != 2 || !bytes.Equal(a.First.Raw, firstCopy) {
		t.Fatal("candidate/materialization accounting wrong")
	}
	before := dpRows(t, st)
	for i := range report.RevisionRaw {
		report.RevisionRaw[i] = 0
	}
	for i := range report.Candidates[0].First.Raw {
		report.Candidates[0].First.Raw[i] = 0
	}
	for i := range report.Candidates[0].Mentions[0].Raw {
		report.Candidates[0].Mentions[0].Raw[i] = 0
	}
	report.Identities.Final.Rows["en:atlas"][0] = 0
	if !bytes.Equal(saved.revisionRaw, original) || !bytes.Equal(handle.first.Raw, firstCopy) || !bytes.Equal(handle.mentions[0].Raw, otherRaw) || !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("returned report aliases private/store state")
	}
	if saved.link(handle, otherKey, otherRaw, "declaration") == nil {
		t.Fatal("closed handle accepted")
	}
}

func TestBirthRegistrationDifferentFirstOccurrencesRefuse(t *testing.T) {
	for _, kind := range []string{"conflicting-declaration", "same-tuple-other-side"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			k1, v1 := birthObservation(t, input, "declaration", 0, "")
			k2, v2 := birthObservation(t, input, "declaration", 2, "")
			if kind == "same-tuple-other-side" {
				k1, v1 = birthObservation(t, input, "endpoint", 1, "src")
				k2, v2 = birthObservation(t, input, "endpoint", 1, "dst")
			}
			before := dpRows(t, st)
			report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				if _, err := r.register(k1, v1); err != nil {
					return err
				}
				if d, err := r.register(k2, v2); err == nil || d.Handle != nil {
					t.Fatal("different occurrence treated as duplicate")
				}
				return nil
			})
			if err == nil || !reflect.DeepEqual(report, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("different origin committed")
			}
		})
	}
}

func TestBirthRegistrationRejectsPriorAndUnregisteredHistory(t *testing.T) {
	for _, kind := range []string{"alias-present", "alias-transient", "entity-transient", "unmatched-entity", "unmatched-alias", "unmatched-transient"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			okey, oraw := birthObservation(t, input, "declaration", 0, "")
			before := dpRows(t, st)
			report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				k, v := []byte("al:atlas"), []byte("atlas")
				if strings.Contains(kind, "entity") || kind == "unmatched-transient" {
					e := Entity{Slug: "atlas", Name: "Atlas", Type: "project", CreatedAt: input.OccurredAt}
					k, v = []byte("en:atlas"), dpJSON(t, e)
				}
				// Deliberately exercise lower-level OWNED recorded writes;
				// the fixed registration freeze may not trust its facade only.
				if err := r.identities.put("atlas", k, v); err != nil {
					return err
				}
				if strings.Contains(kind, "transient") {
					if err := r.identities.delete("atlas", k); err != nil {
						return err
					}
				}
				if !strings.HasPrefix(kind, "unmatched") {
					if d, err := r.register(okey, oraw); err == nil || d.Handle != nil {
						t.Fatal("late registration accepted")
					}
				}
				return nil
			})
			if err == nil || !reflect.DeepEqual(report, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("unregistered history committed")
			}
		})
	}
	for _, kind := range []string{"created-deleted", "recreated", "wrong-name", "wrong-time", "stable-replacement"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			okey, oraw := birthObservation(t, input, "declaration", 0, "")
			before := dpRows(t, st)
			report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				d, err := r.register(okey, oraw)
				if err != nil {
					return err
				}
				k, v := birthEntity(t, d.Handle)
				var e Entity
				json.Unmarshal(v, &e)
				if kind == "wrong-name" {
					e.Name = "Wrong"
					v = dpJSON(t, e)
				}
				if kind == "wrong-time" {
					e.CreatedAt = e.CreatedAt.Add(time.Nanosecond)
					v = dpJSON(t, e)
				}
				if err := r.putIdentity("atlas", k, v); err != nil {
					return err
				}
				if kind == "created-deleted" || kind == "recreated" {
					if err := r.deleteIdentity("atlas", k); err != nil {
						return err
					}
				}
				if kind == "recreated" {
					return r.putIdentity("atlas", k, v)
				}
				if kind == "stable-replacement" {
					e.Name = "Wrong"
					return r.putIdentity("atlas", k, dpJSON(t, e))
				}
				return nil
			})
			if kind == "created-deleted" {
				if err != nil || len(report.Candidates) != 1 || !report.Candidates[0].Created || report.Candidates[0].FinalPresent {
					t.Fatal("transient registered birth accounting wrong")
				}
			} else if err == nil || !reflect.DeepEqual(report, birthRegistryReport{}) {
				t.Fatal("invalid creation coverage accepted")
			}
			if !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("failed/transient scope changed raw bytes")
			}
		})
	}
}

func TestBirthRegistrationTypedDeferralAndPrecedence(t *testing.T) {
	for _, kind := range []string{"current", "historical", "deleted-during-body", "prior-poison", "malformed-observation", "consumed", "retired"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			okey, oraw := birthObservation(t, input, "declaration", 0, "")
			bkey, braw := birthObservation(t, input, "declaration", 1, "")
			old := Fact{Src: "atlas", Relation: "status", Value: "legacy", Fact: "synthetic old assertion", ValidFrom: input.OccurredAt.Add(-time.Hour), Confidence: 0.5}
			if kind == "historical" {
				at := input.OccurredAt
				old.InvalidAt = &at
			}
			// Seed legacy dangling data directly: the public writer rightly
			// rejects a missing source and cannot construct this fixture.
			dpSet(t, st, string(factKey(old.Src, old.Relation, old.KeyDst(), old.ValidFrom)), dpJSON(t, old))
			if kind == "consumed" {
				dpSet(t, st, "il-consumed:atlas", []byte{})
			}
			if kind == "retired" {
				dpSet(t, st, "rs:atlas", []byte{0, 255})
			}
			before := dpRows(t, st)
			prior := errors.New("synthetic earlier error")
			report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				if kind == "prior-poison" {
					r.st.poisonAdmission(prior)
				}
				if kind == "malformed-observation" {
					oraw = append(bytes.Clone(oraw), ' ')
				}
				if kind == "deleted-during-body" {
					if err := r.deleteFact(0, factKey(old.Src, old.Relation, old.KeyDst(), old.ValidFrom)); err != nil {
						return err
					}
				}
				d, err := r.register(okey, oraw)
				if kind == "prior-poison" || kind == "malformed-observation" || kind == "consumed" || kind == "retired" {
					if err == nil || d.Kind != "" || d.Handle != nil {
						t.Fatal("deferral masked real error")
					}
					return nil
				}
				if err != nil || d.Kind != "deferred-preexisting-reference" || d.Handle != nil || r.st.admissionFailure != nil {
					t.Fatal("normal deferral poisoned owner")
				}
				for i := range d.Observation.Raw {
					d.Observation.Raw[i] = 0
				}
				if _, err := r.register(okey, oraw); err != nil {
					return err
				}
				unrelated, err := r.register(bkey, braw)
				if err != nil {
					return err
				}
				k, v := birthEntity(t, unrelated.Handle)
				if err := r.putIdentity("borealis", k, v); err != nil {
					return err
				}
				if kind == "deleted-during-body" {
					return prior
				} // keep old assertion on disk
				return nil
			})
			if kind == "current" || kind == "historical" {
				if err != nil || len(report.Dispositions) != 1 || len(report.Candidates) != 1 || !report.Candidates[0].Created || !bytes.Equal(report.Dispositions[0].Observation.Raw, oraw) {
					t.Fatal("mixed scope lost deferral or valid candidate", err)
				}
				for k, v := range before {
					if !bytes.Equal(dpRows(t, st)[k], v) {
						t.Fatal("mixed deferral changed old data")
					}
				}
			} else {
				if err == nil || !reflect.DeepEqual(report, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
					t.Fatal("error or deliberate rollback committed")
				}
				if (kind == "prior-poison" || kind == "deleted-during-body") && err != prior {
					t.Fatal("prior error precedence lost")
				}
			}
		})
	}
}

func TestBirthRegistrationRealWriteFailureAndCommitConflict(t *testing.T) {
	for _, mode := range []string{"oversized", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithValueThreshold(4096))
			if err != nil {
				t.Fatal("synthetic db open")
			}
			st := &Store{db: db}
			defer st.Close()
			input, key, raw := birthFixture(t)
			okey, oraw := birthObservation(t, input, "declaration", 0, "")
			before := dpRows(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				d, err := r.register(okey, oraw)
				if err != nil {
					return err
				}
				k, v := birthEntity(t, d.Handle)
				if mode == "oversized" {
					var e Entity
					json.Unmarshal(v, &e)
					e.Description = strings.Repeat("synthetic-private ", 1000)
					v = dpJSON(t, e)
					if err := r.putIdentity("atlas", k, v); err == nil || strings.Contains(err.Error(), "synthetic-private") {
						t.Fatal("storage failure missing or unsafe")
					}
					return nil
				}
				if err := r.putIdentity("atlas", k, v); err != nil {
					return err
				}
				if err := r.st.PutEpisode(Episode{ID: "staged"}); err != nil {
					return err
				}
				return st.db.Update(func(other *badger.Txn) error { return other.Set(k, v) })
			})
			if err == nil || !reflect.DeepEqual(report, birthRegistryReport{}) || events != 0 {
				t.Fatal("failure returned usable report or events")
			}
			if mode == "conflict" {
				if !errors.Is(err, badger.ErrConflict) {
					t.Fatal("not real commit conflict")
				}
				if len(dpRows(t, st)) != 1 || dpRows(t, st)["en:atlas"] == nil {
					t.Fatal("owner leaked writes outside competitor")
				}
			} else if !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("storage failure leaked writes")
			}
		})
	}
}
