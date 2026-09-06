package store

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func ownerRaw(t *testing.T, s *Store) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	if err := s.view(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			out[string(item.KeyCopy(nil))] = v
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func ownerStage(t *testing.T, s *Store) {
	t.Helper()
	for _, e := range []Entity{{Slug: "alpha", Name: "Alpha", Aliases: []string{"First"}}, {Slug: "beta", Name: "Beta"}} {
		if err := s.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.PutEpisode(Episode{ID: "episode", Summary: "Complete evidence"}); err != nil {
		t.Fatal(err)
	}
	f := Fact{Src: "alpha", Relation: "related_to", Dst: "beta", Fact: "Alpha references Beta", ValidFrom: time.Unix(100, 123).UTC(), Episodes: []string{"episode"}}
	if err := s.PutFact(f); err != nil {
		t.Fatal(err)
	}
}

func ownerAssertRefused(t *testing.T, s *Store) {
	t.Helper()
	if s.admissionOwner.phase != admissionClosed {
		t.Fatal("facade not closed")
	}
	if err := s.PutEpisode(Episode{ID: "escaped"}); err == nil {
		t.Fatal("escaped facade accepted write")
	}
	if err := s.AtomicWrite(func(*Store) error { t.Fatal("escaped callback invoked"); return nil }); err == nil {
		t.Fatal("escaped facade accepted callback")
	}
	if err := runIdentityAdmission(s, nil, nil); err == nil {
		t.Fatal("escaped facade accepted admission")
	}
}

func TestOwnerIndependentCommitOrdering(t *testing.T) {
	s := openTemp(t)
	var escaped *Store
	var order []string
	events := 0
	s.SetObserver(func(Event) {
		if escaped.admissionOwner.phase != admissionClosed {
			t.Fatal("observer before scope closed")
		}
		events++
	})
	err := runIdentityAdmission(s, func(tx *Store) error {
		escaped = tx
		order = append(order, "body")
		if err := tx.AtomicWrite(func(inner *Store) error {
			return inner.AtomicWrite(func(deep *Store) error {
				if deep != tx {
					t.Fatal("nested facade differs")
				}
				ownerStage(t, deep)
				order = append(order, "nested")
				return nil
			})
		}); err != nil {
			return err
		}
		if err := tx.PutEpisode(Episode{ID: "tail"}); err != nil {
			return err
		}
		order = append(order, "tail")
		return nil
	}, func(tx *Store) error {
		order = append(order, "final")
		if events != 0 {
			t.Fatal("event before commit")
		}
		if _, err := tx.GetEpisode("tail"); err != nil {
			return err
		}
		if _, err := tx.GetEntity("beta"); err != nil {
			return err
		}
		return nil
	})
	if err != nil || events != 5 || !reflect.DeepEqual(order, []string{"body", "nested", "tail", "final"}) {
		t.Fatalf("err=%v events=%d order=%v", err, events, order)
	}
	ownerAssertRefused(t, escaped)
}

func TestOwnerIndependentRollbackBoundaries(t *testing.T) {
	for _, mode := range []string{"body-error", "body-panic", "nested-error", "nested-panic", "recursive", "final-error", "final-panic", "final-episode", "final-delete", "final-atomic", "stage-error"} {
		t.Run(mode, func(t *testing.T) {
			s := openTemp(t)
			if err := s.update(func(tx *badger.Txn) error { return tx.Set([]byte("opaque:retained"), []byte{0, 255, 8}) }); err != nil {
				t.Fatal(err)
			}
			before := ownerRaw(t, s)
			var escaped *Store
			events, checks := 0, 0
			s.SetObserver(func(Event) { events++ })
			marker := errors.New("independent sentinel")
			var got error
			var panicked any
			func() {
				defer func() { panicked = recover() }()
				got = runIdentityAdmission(s, func(tx *Store) error {
					escaped = tx
					ownerStage(t, tx)
					switch mode {
					case "body-error":
						return marker
					case "body-panic":
						panic(marker)
					case "nested-error":
						_ = tx.AtomicWrite(func(inner *Store) error {
							if err := inner.PutEpisode(Episode{ID: "nested"}); err != nil {
								return err
							}
							return marker
						})
					case "nested-panic":
						func() {
							defer func() {
								if recover() != marker {
									t.Fatal("wrong nested panic")
								}
							}()
							_ = tx.AtomicWrite(func(*Store) error { panic(marker) })
						}()
					case "recursive":
						_ = tx.AtomicWrite(func(inner *Store) error {
							_ = runIdentityAdmission(inner, nil, nil)
							return nil
						})
					case "stage-error":
						// Real Badger key-size staging failure through an ordinary mutator.
						if err := tx.PutEpisode(Episode{ID: strings.Repeat("x", 65537)}); err == nil {
							t.Fatal("oversized key accepted")
						}
					}
					return nil
				}, func(tx *Store) error {
					checks++
					switch mode {
					case "final-error":
						return marker
					case "final-panic":
						panic(marker)
					case "final-episode":
						_ = tx.PutEpisode(Episode{ID: "late"})
					case "final-delete":
						_ = tx.DeleteEntity("alpha")
					case "final-atomic":
						_ = tx.AtomicWrite(func(*Store) error { t.Fatal("late callback ran"); return nil })
					default:
						t.Fatal("failed body reached finalizer")
					}
					return nil
				})
			}()
			wantPanic := mode == "body-panic" || mode == "final-panic"
			if wantPanic && panicked != marker {
				t.Fatalf("lost panic %v", panicked)
			}
			if !wantPanic && (panicked != nil || got == nil) {
				t.Fatalf("wanted error; err=%v panic=%v", got, panicked)
			}
			if events != 0 || !reflect.DeepEqual(before, ownerRaw(t, s)) {
				t.Fatal("failure changed complete raw map or emitted events")
			}
			wantChecks := 0
			if strings.HasPrefix(mode, "final-") {
				wantChecks = 1
			}
			if checks != wantChecks {
				t.Fatalf("checks %d wanted %d", checks, wantChecks)
			}
			ownerAssertRefused(t, escaped)
		})
	}
}

func TestOwnerIndependentOrdinaryParentPoisonAndBaseline(t *testing.T) {
	for _, admission := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary-caught-error", true: "refused-admission"}[admission], func(t *testing.T) {
			s := openTemp(t)
			before := ownerRaw(t, s)
			events := 0
			s.SetObserver(func(Event) { events++ })
			err := s.AtomicWrite(func(tx *Store) error {
				ownerStage(t, tx)
				_ = tx.AtomicWrite(func(inner *Store) error {
					if admission {
						_ = runIdentityAdmission(inner, func(*Store) error { t.Fatal("nested body ran"); return nil }, func(*Store) error { t.Fatal("nested check ran"); return nil })
					}
					return errors.New("ordinary caught error")
				})
				return nil
			})
			if admission {
				if !errors.Is(err, errIdentityAdmissionScope) || events != 0 || !reflect.DeepEqual(before, ownerRaw(t, s)) {
					t.Fatal("ordinary parent not poisoned")
				}
			} else if err != nil || events != 4 || reflect.DeepEqual(before, ownerRaw(t, s)) {
				t.Fatal("ordinary baseline behavior changed")
			}
		})
	}
}

func TestOwnerIndependentActualCommitConflict(t *testing.T) {
	s := openTemp(t)
	key := []byte("opaque:conflict")
	if err := s.update(func(tx *badger.Txn) error { return tx.Set(key, []byte("before")) }); err != nil {
		t.Fatal(err)
	}
	expected := ownerRaw(t, s)
	expected[string(key)] = []byte("competitor")
	events, checks := 0, 0
	s.SetObserver(func(Event) { events++ })
	var escaped *Store
	err := runIdentityAdmission(s, func(tx *Store) error {
		escaped = tx
		if err := tx.view(func(raw *badger.Txn) error { _, err := raw.Get(key); return err }); err != nil {
			return err
		}
		ownerStage(t, tx)
		return nil
	}, func(*Store) error {
		checks++
		// Fault injection only: a synthetic competitor invalidates the read set.
		// Separately opened transactions are explicitly outside owner confinement.
		return s.db.Update(func(competing *badger.Txn) error { return competing.Set(key, []byte("competitor")) })
	})
	if !errors.Is(err, badger.ErrConflict) || events != 0 || checks != 1 {
		t.Fatalf("err=%v events=%d checks=%d", err, events, checks)
	}
	if !reflect.DeepEqual(expected, ownerRaw(t, s)) {
		t.Fatal("failed owner leaked raw mutations beyond competitor")
	}
	ownerAssertRefused(t, escaped)
}
