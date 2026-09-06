package store

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestBirthCodeIndependentCaptureAndDeferredDeletion(t *testing.T) {
	for _, historical := range []bool{false, true} {
		t.Run(map[bool]string{false: "current", true: "historical"}[historical], func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			old := Fact{Src: "legacy", Dst: "atlas", Relation: "related_to", Fact: "old destination reference", ValidFrom: input.OccurredAt.Add(-time.Hour), Confidence: 0.5}
			if historical {
				at := input.OccurredAt
				old.InvalidAt = &at
			}
			fk := factKey(old.Src, old.Relation, old.KeyDst(), old.ValidFrom)
			fv := dpJSON(t, old)
			dpSet(t, st, string(fk), fv)
			dpSet(t, st, "al:old spelling", []byte("atlas"))
			obs, obraw := birthObservation(t, input, "endpoint", 1, "dst")
			decl, dv := birthObservation(t, input, "declaration", 0, "")
			other, ov := birthObservation(t, input, "declaration", 1, "")
			var saved *birthRegistry
			out, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				saved = r
				if !bytes.Equal(r.identities.baseline.Rows["al:old spelling"], []byte("atlas")) || r.facts.baseline.Scanned != 1 || len(r.identities.writer.entries) != 0 {
					t.Fatal("baseline not captured before callback")
				}
				if err := r.deleteFact(54321, fk); err != nil {
					return err
				}
				for _, o := range []registeredObservation{{Key: obs, Raw: obraw}, {Key: decl, Raw: dv}, {Key: obs, Raw: obraw}} {
					d, e := r.register(o.Key, o.Raw)
					if e != nil || d.Kind != "deferred-preexisting-reference" || d.Handle != nil || r.st.admissionFailure != nil {
						t.Fatal("baseline reference lost after deletion", e)
					}
					d.Observation.Raw[0] = 0
				}
				d, e := r.register(other, ov)
				if e != nil {
					return e
				}
				k, v := birthEntity(t, d.Handle)
				return r.putIdentity("borealis", k, v)
			})
			if err != nil || len(out.Dispositions) != 2 || len(out.Candidates) != 1 || !out.Candidates[0].Created || out.Facts.Final.Scanned != 0 || len(out.Facts.Mutations) != 1 {
				t.Fatal("mixed deletion accounting failed", err)
			}
			if dpRows(t, st)[string(fk)] != nil || dpRows(t, st)["en:borealis"] == nil || !bytes.Equal(dpRows(t, st)["al:old spelling"], []byte("atlas")) {
				t.Fatal("unexpected committed rows")
			}
			if out.Dispositions[0].Observation.Role != "primary-dst" || out.Dispositions[1].Observation.Role != "declaration" {
				t.Fatal("deferred roles collapsed")
			}
			out.Facts.Mutations[0].Before.Raw[0] = 0
			out.Dispositions[0].Observation.Raw[0] = 0
			if !bytes.Equal(saved.facts.mutations[0].Before.Raw, fv) || !bytes.Equal(saved.dispositions[0].Observation.Raw, obraw) || !bytes.Equal(out.Dispositions[1].Observation.Raw, dv) {
				t.Fatal("returned projections alias")
			}
		})
	}
}

func TestBirthCodeIndependentCachedDecisionsRevalidate(t *testing.T) {
	for _, initial := range []string{"registered", "existing", "deferred"} {
		for _, attack := range []string{"poison", "noncanonical", "different-revision", "occupied"} {
			if attack == "occupied" && initial != "deferred" {
				continue
			}
			t.Run(initial+"/"+attack, func(t *testing.T) {
				st := openTemp(t)
				input, key, raw := birthFixture(t)
				obs, ov := birthObservation(t, input, "declaration", 0, "")
				if initial == "existing" {
					dpSet(t, st, "en:atlas", dpJSON(t, Entity{Slug: "atlas", Name: "Atlas", Type: "project", CreatedAt: input.OccurredAt}))
				}
				if initial == "deferred" {
					f := Fact{Src: "atlas", Relation: "status", Value: "old", Fact: "old", ValidFrom: input.OccurredAt}
					dpSet(t, st, string(factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom)), dpJSON(t, f))
				}
				before := dpRows(t, st)
				prior := errors.New("independent first error")
				out, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
					if _, e := r.register(obs, ov); e != nil {
						return e
					}
					switch attack {
					case "poison":
						r.st.poisonAdmission(prior)
					case "noncanonical":
						ov = append(bytes.Clone(ov), ' ')
					case "different-revision":
						input.Declarations[0].Aliases = []string{"changed"}
						obs, ov = birthObservation(t, input, "declaration", 0, "")
					case "occupied":
						if e := r.st.txn.Set([]byte("rt:atlas"), []byte{}); e != nil {
							return e
						}
					}
					d, e := r.register(obs, ov)
					if !errors.Is(e, errBirthRegistration) || !reflect.DeepEqual(d, birthRegistrationDecision{}) {
						t.Fatal("cached decision bypassed checks", e)
					}
					return nil
				})
				if err == nil || !reflect.DeepEqual(out, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
					t.Fatal("caught failure committed")
				}
				if attack == "poison" && err != prior {
					t.Fatal("first poison lost")
				}
			})
		}
	}
}

func TestBirthCodeIndependentOrderingInterleavedActors(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid-interleaving", true: "late-alias-delete"}[late], func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			a, av := birthObservation(t, input, "declaration", 0, "")
			b, bv := birthObservation(t, input, "declaration", 1, "")
			if late {
				dpSet(t, st, "al:old borealis", []byte("borealis"))
			}
			before := dpRows(t, st)
			out, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				da, e := r.register(a, av)
				if e != nil {
					return e
				}
				ak, ar := birthEntity(t, da.Handle)
				if e = r.putIdentity("atlas", ak, ar); e != nil {
					return e
				}
				if e = r.putIdentity("atlas", []byte("al:a"), []byte("atlas")); e != nil {
					return e
				}
				if late {
					if e = r.identities.delete("borealis", []byte("al:old borealis")); e != nil {
						return e
					}
				}
				db, e := r.register(b, bv)
				if late {
					if e == nil || db.Handle != nil {
						t.Fatal("earlier alias deletion not detected")
					}
					return nil
				}
				if e != nil {
					return e
				}
				if db.Handle.position != 2 {
					t.Fatal("position is not actual identity history length")
				}
				bk, br := birthEntity(t, db.Handle)
				if e = r.putIdentity("borealis", bk, br); e != nil {
					return e
				}
				if e = r.putIdentity("atlas", ak, ar); e != nil {
					return e
				}
				if e = r.deleteIdentity("borealis", bk); e != nil {
					return e
				}
				again, e := r.register(b, bv)
				if e != nil || again.Handle != db.Handle {
					t.Fatal("exact retry after delete failed")
				}
				return nil
			})
			if late {
				if err == nil || !reflect.DeepEqual(out, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
					t.Fatal("late history committed")
				}
				return
			}
			if err != nil || len(out.Candidates) != 2 || out.Candidates[0].Position != 0 || out.Candidates[1].Position != 2 || !out.Candidates[1].Created || out.Candidates[1].FinalPresent {
				t.Fatal("interleaving report failed", err)
			}
		})
	}
}

func TestBirthCodeIndependentOriginalRolesAndCompetingLinks(t *testing.T) {
	st := openTemp(t)
	input, _, _ := birthFixture(t)
	input.Facts = append(input.Facts, observedFact{Src: "90 percent", Relation: "status", Dst: "Workflow", Fact: "value-side flip", ValidFrom: "not a date", Supersedes: &observedSupersedes{Src: "Retired", Relation: "status", Dst: "Yesterday"}})
	raw, key, err := encodeIdentityInput(input)
	if err != nil {
		t.Fatal(err)
	}
	ak, av := birthObservation(t, input, "endpoint", 0, "src")
	bk, bv := birthObservation(t, input, "endpoint", 0, "dst")
	vk, vv := birthObservation(t, input, "endpoint", 2, "src")
	wk, wv := birthObservation(t, input, "endpoint", 2, "dst")
	before := dpRows(t, st)
	out, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
		var hs []*registeredBirth
		for _, o := range []registeredObservation{{Key: ak, Raw: av}, {Key: bk, Raw: bv}, {Key: vk, Raw: vv}, {Key: wk, Raw: wv}} {
			d, e := r.register(o.Key, o.Raw)
			if e != nil {
				return e
			}
			hs = append(hs, d.Handle)
		}
		for _, h := range hs[:2] {
			for _, role := range []string{"primary-src", "supersedes-src", "supersedes-dst"} {
				if e := r.link(h, vk, vv, role); e != nil {
					return e
				}
				if e := r.link(h, vk, vv, role); e != nil {
					return e
				}
			}
		}
		return nil
	})
	if err != nil || len(out.Candidates) != 4 || !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("pure registration wrote graph", err)
	}
	for i, name := range []string{"Atlas", "Borealis", "90 percent", "Workflow"} {
		if out.Candidates[i].Birth.Name != name || out.Candidates[i].Birth.Origin != "endpoint" || out.Candidates[i].Created {
			t.Fatal("original role transformed")
		}
	}
	if len(out.Candidates[0].Mentions) != 3 || len(out.Candidates[1].Mentions) != 3 || out.Candidates[2].First.Role != "primary-src" || out.Candidates[3].First.Role != "primary-dst" {
		t.Fatal("mechanical competing links collapsed")
	}
	out.Candidates[0].Mentions[0].Raw[0] = 0
	if !bytes.Equal(out.Candidates[1].Mentions[0].Raw, vv) || !bytes.Equal(out.Candidates[2].First.Raw, vv) {
		t.Fatal("competing link projections alias")
	}
}

func TestBirthCodeIndependentUntrackedFactFinalizerAndSnapshotOwnership(t *testing.T) {
	for _, untracked := range []bool{false, true} {
		t.Run(map[bool]string{false: "owned-projections", true: "untracked-fact"}[untracked], func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			a, av := birthObservation(t, input, "declaration", 0, "")
			existing := Entity{Slug: "legacy", Name: "Legacy", Aliases: []string{"Prior"}, RepoRefs: []string{"repo"}, CreatedAt: input.OccurredAt}
			dpSet(t, st, "en:legacy", dpJSON(t, existing))
			before := dpRows(t, st)
			f := Fact{Src: "atlas", Relation: "status", Value: "new", Fact: "synthetic", ValidFrom: input.OccurredAt, Episodes: []string{"unproven"}}
			fk, fv := factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom), dpJSON(t, f)
			var saved *birthRegistry
			out, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				saved = r
				d, e := r.register(a, av)
				if e != nil {
					return e
				}
				ek, ev := birthEntity(t, d.Handle)
				if e = r.putIdentity("atlas", ek, ev); e != nil {
					return e
				}
				if untracked {
					return r.st.txn.Set(fk, fv)
				}
				if e = r.putFact(8675309, fk, fv); e != nil {
					return e
				}
				// Repeated stable updates must have distinct owned before/after projections.
				return r.putIdentity("atlas", ek, ev)
			})
			if untracked {
				if err == nil || !reflect.DeepEqual(out, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
					t.Fatal("fixed fact verifier omitted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			out.Identities.Baseline.Entities["legacy"].Aliases[0] = "mutated"
			out.Identities.Baseline.Entities["legacy"].RepoRefs[0] = "mutated"
			out.Identities.Baseline.Rows["en:legacy"][0] = 0
			out.Identities.Final.Entities["legacy"].Aliases[0] = "final-mutated"
			out.Identities.History.Entries[0].After.Value[0] = 0
			out.Identities.History.Final["en:atlas"].Value[0] = 0
			out.Facts.Final.References["atlas"] = identityReferenceCount{Current: 999}
			if saved.identities.baseline.Entities["legacy"].Aliases[0] != "Prior" || saved.identities.baseline.Entities["legacy"].RepoRefs[0] != "repo" || !bytes.Equal(out.Identities.Final.Rows["en:legacy"], before["en:legacy"]) || out.Identities.History.Entries[1].Before.Value[0] != '{' || saved.identities.writer.entries[0].After.Value[0] != '{' {
				t.Fatal("report mutable storage shared")
			}
			if out.Facts.Mutations[0].Ordinal != 8675309 || dpRows(t, st)["ep:"+input.EpisodeID] != nil || dpRows(t, st)["ig:atlas"] != nil {
				t.Fatal("finite report implied provenance/generation")
			}
		})
	}
}

func TestBirthCodeIndependentStableTupleTransientReplacement(t *testing.T) {
	for _, which := range []string{"time", "name", "recreation"} {
		t.Run(which, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			a, av := birthObservation(t, input, "declaration", 0, "")
			before := dpRows(t, st)
			out, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				d, e := r.register(a, av)
				if e != nil {
					return e
				}
				k, v := birthEntity(t, d.Handle)
				if e = r.putIdentity("atlas", k, v); e != nil {
					return e
				}
				if which == "recreation" {
					if e = r.identities.delete("atlas", k); e != nil {
						return e
					}
				} else {
					entity := Entity{Slug: "atlas", Name: "Atlas", CreatedAt: input.OccurredAt, Type: "project"}
					if which == "time" {
						entity.CreatedAt = entity.CreatedAt.Add(time.Nanosecond)
					} else {
						entity.Name = "Atlas!"
					}
					if e = r.identities.put("atlas", k, dpJSON(t, entity)); e != nil {
						return e
					}
				}
				return r.identities.put("atlas", k, v) // restoring final bytes cannot erase invalid history
			})
			if err == nil || !reflect.DeepEqual(out, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("final stable tuple masked invalid history")
			}
		})
	}
}
