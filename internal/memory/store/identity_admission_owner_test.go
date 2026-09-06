package store

import (
	"errors"
	"reflect"
	"testing"
)

func TestAdmissionOwnerChecksOnceAfterAllNestedWrites(t *testing.T) {
	st := openTemp(t)
	checks := 0
	events := 0
	st.SetObserver(func(Event) { events++ })
	var escaped *Store
	err := runIdentityAdmission(st, func(tx *Store) error {
		escaped = tx
		if err := tx.AtomicWrite(func(nested *Store) error {
			if nested != tx {
				t.Fatal("nested facade changed")
			}
			return nested.PutEntity(Entity{Slug: "first", Name: "First"})
		}); err != nil {
			return err
		}
		return tx.PutEntity(Entity{Slug: "later", Name: "Later"})
	}, func(tx *Store) error {
		checks++
		for _, slug := range []string{"first", "later"} {
			if _, err := tx.GetEntity(slug); err != nil {
				return err
			}
		}
		if events != 0 {
			t.Fatal("events before commit")
		}
		return nil
	})
	if err != nil || checks != 1 || events != 2 {
		t.Fatal("wrong finalization/commit sequence")
	}
	if err := escaped.PutEntity(Entity{Slug: "escaped", Name: "Escaped"}); err == nil {
		t.Fatal("expired facade wrote")
	}
}

func TestAdmissionOwnerCaughtNestedFailuresCannotCommit(t *testing.T) {
	for _, mode := range []string{"nested-error", "nested-panic", "recursive-admission", "nil-recursive-admission", "final-error", "final-write", "final-nested"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			before := gradeGenerationSnapshot(t, st)
			events := 0
			checks := 0
			st.SetObserver(func(Event) { events++ })
			forced := errors.New("synthetic boundary failure")
			err := runIdentityAdmission(st, func(tx *Store) error {
				if err := tx.PutEntity(Entity{Slug: "staged", Name: "Staged"}); err != nil {
					return err
				}
				switch mode {
				case "nested-error":
					_ = tx.AtomicWrite(func(*Store) error { return forced })
				case "nested-panic":
					func() {
						defer func() {
							if recover() != forced {
								t.Fatal("unexpected panic")
							}
						}()
						_ = tx.AtomicWrite(func(*Store) error { panic(forced) })
					}()
				case "recursive-admission":
					_ = runIdentityAdmission(tx, func(*Store) error { return nil }, func(*Store) error { return nil })
				case "nil-recursive-admission":
					_ = runIdentityAdmission(tx, nil, nil)
				}
				return nil // Deliberately catch nested refusal.
			}, func(tx *Store) error {
				checks++
				switch mode {
				case "final-error":
					return forced
				case "final-write":
					_ = tx.PutEntity(Entity{Slug: "too-late", Name: "Too Late"})
				case "final-nested":
					_ = tx.AtomicWrite(func(*Store) error { t.Fatal("late callback invoked"); return nil })
				default:
					t.Fatal("poisoned body reached finalizer")
				}
				return nil // Deliberately catch finalization-phase refusal.
			})
			if err == nil || events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("caught failure committed bytes/events")
			}
			if checks > 1 {
				t.Fatal("finalizer called twice")
			}
		})
	}
}

func TestAdmissionOwnerRefusesExistingOrdinaryTransaction(t *testing.T) {
	st := openTemp(t)
	before := gradeGenerationSnapshot(t, st)
	err := st.AtomicWrite(func(tx *Store) error {
		if err := tx.PutEntity(Entity{Slug: "ordinary", Name: "Ordinary"}); err != nil {
			return err
		}
		_ = runIdentityAdmission(tx, func(*Store) error { t.Fatal("nested admission body ran"); return nil }, func(*Store) error { t.Fatal("nested finalizer ran"); return nil })
		return nil
	})
	if !errors.Is(err, errIdentityAdmissionScope) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("nested admission refusal did not poison ordinary outer commit")
	}
}

func TestAdmissionOwnerPanicRollbackAndClosedScope(t *testing.T) {
	for _, phase := range []string{"body", "finalizer"} {
		t.Run(phase, func(t *testing.T) {
			st := openTemp(t)
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			var saved *Store
			func() {
				defer func() {
					if recover() != phase {
						t.Fatal("panic not propagated")
					}
				}()
				_ = runIdentityAdmission(st, func(tx *Store) error {
					saved = tx
					if err := tx.PutEntity(Entity{Slug: "staged", Name: "Staged"}); err != nil {
						return err
					}
					if phase == "body" {
						panic(phase)
					}
					return nil
				}, func(*Store) error { panic(phase) })
			}()
			if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("panic escaped rollback")
			}
			if saved.admissionOwner.phase != admissionClosed {
				t.Fatal("panic left scope active")
			}
			if err := saved.AtomicWrite(func(*Store) error { t.Fatal("expired callback ran"); return nil }); err == nil {
				t.Fatal("expired scope accepted")
			}
		})
	}
}
