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

func ilIndependentRaw(t *testing.T, st *Store) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	if err := st.view(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			v, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			out[string(it.Item().KeyCopy(nil))] = v
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func ilIndependentSet(t *testing.T, st *Store, k string, v []byte) {
	t.Helper()
	if err := st.update(func(tx *badger.Txn) error { return tx.Set([]byte(k), v) }); err != nil {
		t.Fatal(err)
	}
}

func ilIndependentEntity(t *testing.T, actor string, revision int) []byte {
	t.Helper()
	e := Entity{Slug: actor, Name: "Display " + actor, Aliases: []string{"A/B", "a b", "A/B"}, RepoRefs: []string{"repo-one", "repo-two"}, Description: fmt.Sprint(revision), CreatedAt: time.Unix(1760000000, 17).UTC()}
	v, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func ilIndependentImage(m map[string][]byte, key string) identityImage {
	v, exists := m[key]
	return identityImage{Exists: exists, Value: bytes.Clone(v)}
}

func TestILIndependentGeneratedCompleteReplay(t *testing.T) {
	for _, seed := range []int64{1, 922, 7142} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			st := openTemp(t)
			ilIndependentSet(t, st, "al:empty", []byte{})
			for _, f := range []string{"ar:", "ig:", "il:", "il-consumed:", "meta:identity_", "rs:", "rt:"} {
				ilIndependentSet(t, st, f+"\x00\xff", []byte{0, 255})
			}
			ilIndependentSet(t, st, "outside:unchanged", []byte("unmeasured"))
			whole := ilIndependentRaw(t, st)
			model := map[string][]byte{}
			for k, v := range whole {
				if k != keySchemaVersion && k != "outside:unchanged" {
					model[k] = bytes.Clone(v)
				}
			}
			base := map[string][]byte{}
			for k, v := range model {
				base[k] = bytes.Clone(v)
			}
			entries := []identityWriterEntry{}
			first, final := map[string]identityImage{}, map[string]identityImage{}
			var l *identityMutationLedger
			var report identityMutationReport
			events := 0
			st.SetObserver(func(Event) { events++ })
			err := runIdentityAdmission(st, func(tx *Store) error {
				var err error
				l, err = beginIdentityMutationLedger(tx)
				if err != nil {
					return err
				}
				rng := rand.New(rand.NewSource(seed))
				for n := 0; n < 333; n++ {
					actor := []string{"east", "north", "west"}[rng.Intn(3)]
					key := fmt.Sprintf("al:k-%d", rng.Intn(11))
					value := []byte(actor)
					if n%3 == 0 {
						key = "en:" + actor
						value = ilIndependentEntity(t, actor, n)
					}
					before := ilIndependentImage(model, key)
					if _, ok := first[key]; !ok {
						first[key] = before
					}
					after := identityImage{Exists: true, Value: bytes.Clone(value)}
					keyBuf := []byte(key)
					if before.Exists && n%5 == 0 {
						if strings.HasPrefix(key, "al:") {
							actor = string(before.Value)
						}
						after = identityImage{}
						err = l.delete(actor, keyBuf)
						delete(model, key)
					} else {
						err = l.put(actor, keyBuf, value)
						model[key] = bytes.Clone(value)
					}
					if err != nil {
						return err
					}
					final[key] = after
					entries = append(entries, identityWriterEntry{Sequence: uint64(n), Actor: actor, Key: []byte(key), Before: before, After: after})
					for i := range keyBuf {
						keyBuf[i] = 0
					}
					for i := range value {
						value[i] = 0
					}
				}
				return nil
			}, func(*Store) error { var err error; report, err = l.verify(); return err })
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(report.Baseline.Rows, base) || !reflect.DeepEqual(report.Final.Rows, model) || !reflect.DeepEqual(report.History, identityWriterSnapshot{Entries: entries, Before: first, Final: final}) {
				t.Fatal("exact independent oracle differs")
			}
			for k := range whole {
				if k != keySchemaVersion && k != "outside:unchanged" {
					delete(whole, k)
				}
			}
			for k, v := range model {
				whole[k] = v
			}
			if events != 0 || !reflect.DeepEqual(whole, ilIndependentRaw(t, st)) {
				t.Fatal("unexpected committed bytes/events")
			}
		})
	}
}

func TestILIndependentEntityAndSelectedDrift(t *testing.T) {
	for _, when := range []string{"before", "between", "after", "other"} {
		for _, mode := range []string{"replace", "delete", "empty"} {
			t.Run(when+mode, func(t *testing.T) {
				st := openTemp(t)
				ilIndependentSet(t, st, "en:east", ilIndependentEntity(t, "east", 0))
				original := ilIndependentRaw(t, st)
				var l *identityMutationLedger
				err := runIdentityAdmission(st, func(tx *Store) error {
					var err error
					l, err = beginIdentityMutationLedger(tx)
					if err != nil {
						return err
					}
					tracked := "en:east"
					actor := "east"
					if when == "other" {
						tracked = "en:west"
						actor = "west"
					}
					if when != "before" {
						if err = l.put(actor, []byte(tracked), ilIndependentEntity(t, actor, 1)); err != nil {
							return err
						}
					}
					switch mode {
					case "replace":
						err = tx.txn.Set([]byte("en:east"), ilIndependentEntity(t, "east", 7))
					case "delete":
						err = tx.txn.Delete([]byte("en:east"))
					case "empty":
						err = tx.txn.Set([]byte("en:east"), []byte{})
					}
					if err != nil {
						return err
					}
					if when == "before" || when == "between" {
						_ = l.put("east", []byte("en:east"), ilIndependentEntity(t, "east", 2))
					}
					return nil
				}, func(*Store) error {
					r, err := l.verify()
					if err != errIdentityMutationLedger || !reflect.DeepEqual(r, identityMutationReport{}) {
						t.Fatal("drift accepted or partial report")
					}
					return nil
				})
				if err == nil || !reflect.DeepEqual(original, ilIndependentRaw(t, st)) {
					t.Fatal("drift committed")
				}
			})
		}
	}
	for _, family := range []string{"al:", "ar:", "ig:", "il:", "il-consumed:", "meta:identity_", "rs:", "rt:"} {
		t.Run(family, func(t *testing.T) {
			st := openTemp(t)
			key := family + "\xff\x00"
			ilIndependentSet(t, st, key, []byte{1, 0, 3})
			before := ilIndependentRaw(t, st)
			var l *identityMutationLedger
			err := runIdentityAdmission(st, func(tx *Store) error {
				var err error
				l, err = beginIdentityMutationLedger(tx)
				if err != nil {
					return err
				}
				return tx.txn.Set([]byte(key), []byte{1, 0, 4})
			}, func(*Store) error {
				r, err := l.verify()
				if err != errIdentityMutationLedger || !reflect.DeepEqual(r, identityMutationReport{}) {
					t.Fatal("same-length opaque drift accepted")
				}
				return nil
			})
			if err != errIdentityMutationLedger || !reflect.DeepEqual(before, ilIndependentRaw(t, st)) {
				t.Fatal("opaque control drift committed")
			}
		})
	}
}

func TestILIndependentProjectionIsolation(t *testing.T) {
	st := openTemp(t)
	ilIndependentSet(t, st, "en:east", ilIndependentEntity(t, "east", 0))
	ilIndependentSet(t, st, "al:a/b", []byte("east"))
	var l *identityMutationLedger
	err := runIdentityAdmission(st, func(tx *Store) error {
		var err error
		l, err = beginIdentityMutationLedger(tx)
		if err != nil {
			return err
		}
		if err = l.put("east", []byte("en:east"), ilIndependentEntity(t, "east", 1)); err != nil {
			return err
		}
		return l.put("east", []byte("al:a/b"), []byte("east"))
	}, func(*Store) error {
		r, err := l.verify()
		if err != nil {
			return err
		}
		baseline, _ := json.Marshal(l.baseline)
		private, _ := json.Marshal(l.writer.entries)
		final, _ := json.Marshal(r.Final)
		r.Baseline.Rows["en:east"][0] = 0
		r.Baseline.Entities["east"].Aliases[0] = "changed"
		r.Baseline.Entities["east"].RepoRefs[0] = "changed"
		r.Baseline.Listings["a/b"][0].Spelling = "changed"
		r.Baseline.NaturalListings["ab"][0].Spelling = "changed"
		r.Baseline.IndexTargets["east"][0] = "changed"
		r.Baseline.FamilyCounts["en:"] = 77
		actualFinal, _ := json.Marshal(r.Final)
		if !bytes.Equal(final, actualFinal) {
			t.Fatal("baseline report aliases final")
		}
		for k := range r.Final.Rows {
			r.Final.Rows[k] = []byte("changed")
		}
		r.Final.Entities["east"].RepoRefs[0] = "changed"
		r.Final.Listings["a/b"][0].Spelling = "changed"
		if r.Final.NaturalListings["ab"][0].Spelling != "A/B" {
			t.Fatal("listing projections alias")
		}
		for i := range r.History.Entries {
			r.History.Entries[i].Key[0] = 0
			if len(r.History.Entries[i].Before.Value) > 0 {
				r.History.Entries[i].Before.Value[0] = 0
			}
			r.History.Entries[i].After.Value[0] = 0
		}
		if r.History.Before["al:a/b"].Value[0] != 'e' || r.History.Final["al:a/b"].Value[0] != 'e' {
			t.Fatal("history entries alias maps")
		}
		for _, m := range []map[string]identityImage{r.History.Before, r.History.Final} {
			for _, v := range m {
				if len(v.Value) > 0 {
					v.Value[0] = 0
				}
			}
		}
		gotBase, _ := json.Marshal(l.baseline)
		gotPrivate, _ := json.Marshal(l.writer.entries)
		if !bytes.Equal(baseline, gotBase) || !bytes.Equal(private, gotPrivate) {
			t.Fatal("report mutates owned internals")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ilIndependentRaw(t, st)["en:east"], ilIndependentEntity(t, "east", 1)) {
		t.Fatal("report mutated commit")
	}
}

func TestILIndependentScopeAndPrecedence(t *testing.T) {
	st := openTemp(t)
	before := ilIndependentRaw(t, st)
	for _, mode := range []string{"ordinary", "begin-final", "discarded", "prior-begin", "prior-verify", "writer"} {
		t.Run(mode, func(t *testing.T) {
			prior := errors.New("independent private prior")
			var l *identityMutationLedger
			var local error
			finalized := false
			if mode == "ordinary" {
				err := st.AtomicWrite(func(tx *Store) error { _, local = beginIdentityMutationLedger(tx); return local })
				if err != errIdentityMutationLedger {
					t.Fatal(err)
				}
				return
			}
			err := runIdentityAdmission(st, func(tx *Store) error {
				if mode == "prior-begin" {
					tx.poisonAdmission(prior)
				}
				var err error
				l, err = beginIdentityMutationLedger(tx)
				if mode == "prior-begin" {
					local = err
					return nil
				}
				if err != nil {
					return err
				}
				if err := l.put("east", []byte("al:staged"), []byte("east")); err != nil {
					return err
				}
				if mode == "discarded" {
					tx.txn.Discard()
					local = l.put("east", []byte("al:later"), []byte("east"))
				}
				if mode == "writer" {
					local = l.put("east", []byte("al:bad"), []byte("west"))
				}
				return nil
			}, func(tx *Store) error {
				finalized = true
				if mode == "begin-final" {
					_, local = beginIdentityMutationLedger(tx)
				}
				if mode == "prior-verify" {
					tx.poisonAdmission(prior)
					_, local = l.verify()
				}
				return nil
			})
			if local != errIdentityMutationLedger {
				t.Fatalf("local leaks or wrong sentinel: %v", local)
			}
			want := error(errIdentityMutationLedger)
			if mode == "writer" {
				want = errIdentityWriterHistory
			}
			if mode == "prior-begin" || mode == "prior-verify" {
				want = prior
			}
			if err != want {
				t.Fatalf("outer precedence %v != %v", err, want)
			}
			if (mode == "writer" || mode == "discarded" || mode == "prior-begin") && finalized {
				t.Fatal("poisoned body finalized")
			}
			if !reflect.DeepEqual(before, ilIndependentRaw(t, st)) {
				t.Fatal("refused scope committed")
			}
		})
	}
}

func TestILIndependentTwoActualOwners(t *testing.T) {
	first, second := openTemp(t), openTemp(t)
	var l *identityMutationLedger
	err := runIdentityAdmission(first, func(tx *Store) error {
		var err error
		l, err = beginIdentityMutationLedger(tx)
		if err != nil {
			return err
		}
		secondBefore := ilIndependentRaw(t, second)
		err = runIdentityAdmission(second, func(other *Store) error {
			if err := other.txn.Set([]byte("al:staged"), []byte("east")); err != nil {
				return err
			}
			original := l.st
			l.st = other
			defer func() { l.st = original }()
			if err := l.put("east", []byte("al:foreign"), []byte("east")); err != errIdentityMutationLedger {
				t.Fatal("foreign owner accepted")
			}
			return nil
		}, func(*Store) error { t.Fatal("foreign poisoned owner finalized"); return nil })
		if err != errIdentityMutationLedger || !reflect.DeepEqual(secondBefore, ilIndependentRaw(t, second)) {
			t.Fatal("second owner leaked staged write")
		}
		return l.put("east", []byte("al:own"), []byte("east"))
	}, func(*Store) error { _, err := l.verify(); return err })
	if err != nil {
		t.Fatal("original owner damaged", err)
	}
}

func TestILIndependentConflictAndPanicSuppressEvents(t *testing.T) {
	for _, mode := range []string{"conflict", "body-panic", "final-panic"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			ilIndependentSet(t, st, "al:shared", []byte("east"))
			before := ilIndependentRaw(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			var l *identityMutationLedger
			var err error
			panicked := false
			verified := false
			func() {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				err = runIdentityAdmission(st, func(tx *Store) error {
					var err error
					l, err = beginIdentityMutationLedger(tx)
					if err != nil {
						return err
					}
					if err = l.put("west", []byte("al:shared"), []byte("west")); err != nil {
						return err
					}
					if err = l.put("east", []byte("al:staged"), []byte("east")); err != nil {
						return err
					}
					tx.notify(Event{Kind: "entity", Op: "put", Slug: "synthetic"})
					if mode == "body-panic" {
						panic("synthetic")
					}
					if mode == "conflict" {
						return st.db.Update(func(other *badger.Txn) error { return other.Set([]byte("al:shared"), []byte("competitor")) })
					}
					return nil
				}, func(*Store) error {
					_, err := l.verify()
					verified = err == nil
					if mode == "final-panic" {
						panic("synthetic")
					}
					return err
				})
			}()
			if mode == "conflict" {
				before["al:shared"] = []byte("competitor")
				if !errors.Is(err, badger.ErrConflict) || !verified || panicked {
					t.Fatal("not an actual commit conflict", err)
				}
			} else if !panicked {
				t.Fatal("panic swallowed")
			}
			if events != 0 || !reflect.DeepEqual(before, ilIndependentRaw(t, st)) {
				t.Fatal("rollback leaked writes/events")
			}
		})
	}
}

func TestILIndependentDocumentedLimits(t *testing.T) {
	for _, mode := range []string{"reverted", "post-verify", "public-phantom", "outside"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			var l *identityMutationLedger
			var r identityMutationReport
			err := runIdentityAdmission(st, func(tx *Store) error {
				var err error
				l, err = beginIdentityMutationLedger(tx)
				if err != nil {
					return err
				}
				if err := l.put("east", []byte("al:tracked"), []byte("east")); err != nil {
					return err
				}
				switch mode {
				case "reverted":
					if err := tx.txn.Set([]byte("al:transient"), []byte("west")); err != nil {
						return err
					}
					return tx.txn.Delete([]byte("al:transient"))
				case "public-phantom":
					return st.ClaimAlias("phantom", "missing-owner")
				case "outside":
					return tx.txn.Set([]byte("iga:outside"), []byte("opaque"))
				}
				return nil
			}, func(tx *Store) error {
				var err error
				r, err = l.verify()
				if err != nil {
					return err
				}
				if mode == "post-verify" {
					return tx.txn.Set([]byte("al:later"), []byte("west"))
				}
				return nil
			})
			if err != nil {
				t.Fatal("documented limitation behavior changed", err)
			}
			actual := ilIndependentRaw(t, st)
			if mode == "public-phantom" {
				if _, exists := r.Final.Rows["al:phantom"]; exists || string(actual["al:phantom"]) != "missing-owner" {
					t.Fatal("phantom characterization failed")
				}
			}
			if mode == "post-verify" {
				if _, exists := r.Final.Rows["al:later"]; exists || string(actual["al:later"]) != "west" {
					t.Fatal("post-boundary characterization failed")
				}
			}
			if mode == "outside" {
				if _, exists := r.Final.Rows["iga:outside"]; exists || string(actual["iga:outside"]) != "opaque" {
					t.Fatal("outside-family characterization failed")
				}
			}
		})
	}
}
