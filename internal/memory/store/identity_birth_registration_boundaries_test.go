package store

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func TestBirthRegistrationHandleAndRoleBoundaries(t *testing.T) {
	for _, mode := range []string{"nil", "copied", "foreign", "unknown-role", "wrong-side", "declaration-hint", "missing-hint", "mixed-revision"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			okey, oraw := birthObservation(t, input, "declaration", 0, "")
			var foreign *registeredBirth
			if mode == "foreign" {
				_, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
					d, err := r.register(okey, oraw)
					foreign = d.Handle
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			before := dpRows(t, st)
			report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				d, err := r.register(okey, oraw)
				if err != nil {
					return err
				}
				k, v := birthEntity(t, d.Handle)
				if err := r.putIdentity("atlas", k, v); err != nil {
					return err
				}
				h := d.Handle
				linkKey, linkRaw, role := okey, oraw, "declaration"
				switch mode {
				case "nil":
					h = nil
				case "copied":
					copy := *h
					h = &copy
				case "foreign":
					h = foreign
				case "unknown-role":
					role = "alias-authority"
				case "wrong-side":
					linkKey, linkRaw = birthObservation(t, input, "endpoint", 0, "src")
					role = "primary-dst"
				case "declaration-hint":
					role = "supersedes-src"
				case "missing-hint":
					linkKey, linkRaw = birthObservation(t, input, "endpoint", 1, "src")
					role = "supersedes-src"
				case "mixed-revision":
					input.Declarations[0].Description = "different parsed revision"
					linkKey, linkRaw = birthObservation(t, input, "declaration", 0, "")
				}
				if r.link(h, linkKey, linkRaw, role) == nil {
					t.Fatal("invalid link accepted")
				}
				return nil // catching a structural error must not allow commit
			})
			if err == nil || !reflect.DeepEqual(report, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("invalid link escaped rollback")
			}
		})
	}
}

func TestBirthRegistrationInputAndControlBoundaries(t *testing.T) {
	for _, mode := range []string{"bad-key", "noncanonical", "empty-input", "generation", "anchor", "consumed", "retired-slug", "retired-name", "current-generation", "current-entity"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			okey, oraw := birthObservation(t, input, "declaration", 0, "")
			control := map[string]string{"generation": "ig:atlas", "anchor": "il:atlas", "consumed": "il-consumed:atlas", "retired-slug": "rs:atlas", "retired-name": "rt:atlas"}[mode]
			if control != "" {
				dpSet(t, st, control, []byte{})
			}
			switch mode {
			case "bad-key":
				key += "x"
			case "noncanonical":
				raw = append(bytes.Clone(raw), ' ')
			case "empty-input":
				raw = nil
			}
			before := dpRows(t, st)
			called := false
			report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
				called = true
				// Raw writes model occupied current state; final accounting
				// also refuses it. No registration may be returned first.
				if mode == "current-generation" {
					if err := r.st.txn.Set([]byte("ig:atlas"), []byte{}); err != nil {
						return err
					}
				}
				if mode == "current-entity" {
					e := Entity{Slug: "atlas", Name: "Atlas", Type: "project", CreatedAt: input.OccurredAt}
					if err := r.st.txn.Set([]byte("en:atlas"), dpJSON(t, e)); err != nil {
						return err
					}
				}
				if d, err := r.register(okey, oraw); err == nil || d.Handle != nil {
					t.Fatal("invalid birth accepted")
				}
				return nil
			})
			if (mode == "bad-key" || mode == "noncanonical" || mode == "empty-input") && called {
				t.Fatal("invalid revision invoked body")
			}
			if err == nil || !reflect.DeepEqual(report, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) {
				t.Fatal("invalid input/control committed")
			}
		})
	}
}

func TestBirthRegistrationLifecycleAndPoison(t *testing.T) {
	for _, mode := range []string{"early-freeze", "nested", "ordinary-parent", "body-error", "body-panic", "unregistered-facade", "untracked-entity", "untracked-alias"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			input, key, raw := birthFixture(t)
			okey, oraw := birthObservation(t, input, "declaration", 0, "")
			before := dpRows(t, st)
			prior := errors.New("synthetic body refusal")
			var saved *birthRegistry
			var report birthRegistryReport
			var err error
			var panicked any
			events := 0
			st.SetObserver(func(Event) { events++ })
			func() {
				defer func() { panicked = recover() }()
				if mode == "ordinary-parent" {
					err = st.AtomicWrite(func(tx *Store) error {
						if err := tx.PutEpisode(Episode{ID: "must-rollback"}); err != nil {
							return err
						}
						report, _ = runBirthRegistration(tx, key, raw, func(*birthRegistry) error { t.Fatal("nested body ran"); return nil })
						return nil
					})
					return
				}
				report, err = runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
					saved = r
					d, err := r.register(okey, oraw)
					if err != nil {
						return err
					}
					k, v := birthEntity(t, d.Handle)
					if err := r.putIdentity("atlas", k, v); err != nil {
						return err
					}
					switch mode {
					case "early-freeze":
						if _, err := r.freeze(); err == nil {
							t.Fatal("early freeze accepted")
						}
					case "nested":
						if _, err := runBirthRegistration(r.st, key, raw, func(*birthRegistry) error { t.Fatal("nested body ran"); return nil }); err == nil {
							t.Fatal("nested wrapper accepted")
						}
					case "body-error":
						return prior
					case "body-panic":
						panic(prior)
					case "unregistered-facade":
						if r.putIdentity("unregistered", []byte("al:unregistered"), []byte("unregistered")) == nil {
							t.Fatal("unregistered facade accepted")
						}
					case "untracked-entity":
						e := Entity{Slug: "unregistered", Name: "Unregistered", Type: "project", CreatedAt: input.OccurredAt}
						return r.st.txn.Set([]byte("en:unregistered"), dpJSON(t, e))
					case "untracked-alias":
						return r.st.txn.Set([]byte("al:untracked"), []byte("atlas"))
					}
					return nil
				})
			}()
			if mode == "body-panic" {
				if panicked != prior {
					t.Fatal("body panic did not propagate")
				}
			} else if panicked != nil || err == nil {
				t.Fatal("boundary failed to refuse")
			}
			if mode == "body-error" && err != prior {
				t.Fatal("body error precedence changed")
			}
			if !reflect.DeepEqual(report, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) || events != 0 {
				t.Fatal("failed scope leaked report, rows or events")
			}
			if saved != nil {
				if _, err := saved.register(okey, oraw); err == nil {
					t.Fatal("closed registry reused")
				}
			}
			// A refusal or panic must release the coordinator for later work.
			if _, err := runBirthRegistration(st, key, raw, func(*birthRegistry) error { return nil }); err != nil {
				t.Fatal("coordinator did not recover", err)
			}
		})
	}
	_, key, raw := birthFixture(t)
	if _, err := runBirthRegistration(nil, key, raw, func(*birthRegistry) error { return nil }); err == nil {
		t.Fatal("nil root accepted")
	}
	if _, err := runBirthRegistration(openTemp(t), key, raw, nil); err == nil {
		t.Fatal("nil body accepted")
	}
}

func TestBirthRegistrationExistingAndMechanicalLinkLimit(t *testing.T) {
	st := openTemp(t)
	input, key, raw := birthFixture(t)
	okey, oraw := birthObservation(t, input, "declaration", 0, "")
	bkey, braw := birthObservation(t, input, "declaration", 1, "")
	existing := Entity{Slug: "atlas", Name: "Atlas", Type: "project", CreatedAt: input.OccurredAt.Add(-1)}
	dpSet(t, st, "en:atlas", dpJSON(t, existing))
	before := dpRows(t, st)
	var saved *birthRegistry
	report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
		saved = r
		d, err := r.register(okey, oraw)
		if err != nil || d.Kind != "existing-not-new" || d.Handle != nil {
			t.Fatal("existing identity got new birth authority")
		}
		for i := range d.Observation.Raw {
			d.Observation.Raw[i] = 0
		}
		if _, err := r.register(okey, oraw); err != nil {
			return err
		}
		b, err := r.register(bkey, braw)
		if err != nil {
			return err
		}
		// A mention can be mechanically associated with another candidate,
		// but doing so establishes no alias/index or routing authority.
		return r.link(b.Handle, okey, oraw, "declaration")
	})
	if err != nil || len(report.Dispositions) != 1 || len(report.Candidates) != 1 || report.Dispositions[0].Handle != nil || !bytes.Equal(report.Dispositions[0].Observation.Raw, oraw) || !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("mechanical classification changed graph", err)
	}
	for i := range report.Dispositions[0].Observation.Raw {
		report.Dispositions[0].Observation.Raw[i] = 0
	}
	if !bytes.Equal(saved.dispositions[0].Observation.Raw, oraw) || !bytes.Equal(report.Candidates[0].Mentions[0].Raw, oraw) {
		t.Fatal("disposition projections share mutable storage")
	}
}
