package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// These fixtures, snapshots, state transitions and digest framing are independent
// of the builder's test helpers and the implementation's reconstruction routines.
func independentLedgerRow(t *testing.T, n, version int) ([]byte, []byte) {
	t.Helper()
	f := Fact{Src: fmt.Sprintf("independent-%03d", n), Relation: "uses", Dst: "destination", Fact: fmt.Sprintf("Synthetic assertion revision %d.", version), ValidFrom: time.Date(2026, 9, 5, 8, 4, 3, 17, time.UTC), Confidence: .5, Episodes: []string{"synthetic-a", "synthetic-b"}}
	if n%3 == 0 {
		f.Dst, f.Value = "", "literal value"
	}
	if n%3 == 1 {
		f.Dst = f.Src
	}
	if version%2 == 1 {
		v := f.ValidFrom.Add(time.Hour)
		f.InvalidAt = &v
	}
	if version%4 == 0 {
		f.Episodes = nil
	}
	if version%4 == 1 {
		f.Episodes = []string{}
	}
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw[:len(raw)-1], []byte(fmt.Sprintf(`,"opaque":1e99999,"opaque":{"version":%d,"unicode":"\ud83d\ude00"}}`, version))...)
	return factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom), raw
}

func independentLedgerSnapshot(t *testing.T, s *Store) map[string][]byte {
	t.Helper()
	m := map[string][]byte{}
	if err := s.view(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			v, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			m[string(it.Item().KeyCopy(nil))] = v
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return m
}

func independentLedgerDigest(m map[string][]byte) (uint64, string) {
	keys := []string{}
	for k := range m {
		if strings.HasPrefix(k, "fa:") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	framed := []byte("identity-reference-inventory-v1\x00")
	for _, k := range keys {
		for _, part := range [][]byte{[]byte(k), m[k]} {
			n := uint64(len(part))
			for shift := 56; shift >= 0; shift -= 8 {
				framed = append(framed, byte(n>>uint(shift)))
			}
			framed = append(framed, part...)
		}
	}
	digest := sha256.Sum256(framed)
	return uint64(len(keys)), hex.EncodeToString(digest[:])
}

func independentLedgerSeed(t *testing.T, s *Store, n int) {
	t.Helper()
	if err := s.db.Update(func(tx *badger.Txn) error {
		if err := tx.Set([]byte("opaque:outside-facts"), []byte{255, 0, 12}); err != nil {
			return err
		}
		if err := tx.Set([]byte("opaque:empty"), []byte{}); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			k, v := independentLedgerRow(t, i, 0)
			if err := tx.Set(k, v); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestIndependentLedgerModelAndOwnedHistory(t *testing.T) {
	for seed := int64(0); seed < 6; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			s := openTemp(t)
			independentLedgerSeed(t, s, 30)
			model := independentLedgerSnapshot(t, s)
			want := []identityFactMutation{}
			events := 0
			s.SetObserver(func(Event) { events++ })
			var l *identityFactLedger
			var report identityFactLedgerReport
			err := runIdentityAdmission(s, func(tx *Store) error {
				var err error
				l, err = beginIdentityFactLedger(tx)
				if err != nil {
					return err
				}
				rng := rand.New(rand.NewSource(seed))
				for i := 0; i < 240; i++ {
					n := rng.Intn(55)
					k, raw := independentLedgerRow(t, n, i)
					before, exists := model[string(k)]
					m := identityFactMutation{Ordinal: rng.Intn(9), Key: bytes.Clone(k), Before: identityFactState{Exists: exists, Raw: bytes.Clone(before)}}
					if exists && rng.Intn(3) == 0 {
						if err := l.delete(m.Ordinal, k); err != nil {
							return err
						}
						delete(model, string(k))
					} else {
						if exists && rng.Intn(4) == 0 {
							raw = bytes.Clone(before)
						}
						if err := tx.AtomicWrite(func(inner *Store) error {
							if inner != tx {
								t.Fatal("nested facade changed")
							}
							return l.put(m.Ordinal, k, raw)
						}); err != nil {
							return err
						}
						model[string(k)] = bytes.Clone(raw)
						m.After = identityFactState{Exists: true, Raw: bytes.Clone(raw)}
					}
					want = append(want, m)
					for j := range k {
						k[j] = 0
					}
					for j := range raw {
						raw[j] = 0
					}
					if len(*tx.pendingEvents) != 0 {
						t.Fatal("helper staged event")
					}
				}
				return nil
			}, func(tx *Store) error {
				var err error
				report, err = l.verify()
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(report.Mutations, want) {
					t.Fatal("exact ordered before/after/ordinal history differs")
				}
				if !reflect.DeepEqual(independentLedgerSnapshot(t, tx), model) {
					t.Fatal("active owner raw bytes differ")
				}
				count, digest := independentLedgerDigest(model)
				if report.Final.Scanned != count || report.Final.Digest != digest {
					t.Fatal("independent digest mismatch")
				}
				// Mutating all outward mutable surfaces must not alias internal history.
				for _, m := range report.Mutations {
					for i := range m.Key {
						m.Key[i] = 0
					}
					for i := range m.Before.Raw {
						m.Before.Raw[i] = 0
					}
					for i := range m.After.Raw {
						m.After.Raw[i] = 0
					}
				}
				for k := range report.Final.References {
					delete(report.Final.References, k)
				}
				if !reflect.DeepEqual(l.mutations, want) {
					t.Fatal("report aliases internal history")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if events != 0 || !reflect.DeepEqual(model, independentLedgerSnapshot(t, s)) {
				t.Fatal("commit raw equality/events failed")
			}
		})
	}
}

func TestIndependentLedgerUntrackedTransitionMatrix(t *testing.T) {
	for _, stage := range []string{"untouched", "before-first", "after-last", "between"} {
		for _, initial := range []bool{false, true} {
			for _, rogue := range []string{"delete", "new-bytes", "whitespace", "extension"} {
				t.Run(fmt.Sprintf("%s/%t/%s", stage, initial, rogue), func(t *testing.T) {
					if !initial && rogue == "delete" && (stage == "untouched" || stage == "before-first") {
						return
					} // Absent-to-absent has no final byte change.
					s := openTemp(t)
					independentLedgerSeed(t, s, 3)
					n := 1
					if !initial {
						n = 90
					}
					k, original := independentLedgerRow(t, n, 0)
					before := independentLedgerSnapshot(t, s)
					events := 0
					s.SetObserver(func(Event) { events++ })
					var l *identityFactLedger
					err := runIdentityAdmission(s, func(tx *Store) error {
						var err error
						l, err = beginIdentityFactLedger(tx)
						if err != nil {
							return err
						}
						otherK, otherV := independentLedgerRow(t, 99, 1)
						if err = l.put(5, otherK, otherV); err != nil {
							return err
						}
						if stage == "after-last" || stage == "between" {
							if err = l.put(1, k, original); err != nil {
								return err
							}
						}
						var changed []byte
						switch rogue {
						case "delete":
							err = tx.txn.Delete(k)
						case "new-bytes":
							_, changed = independentLedgerRow(t, n, 5)
							err = tx.txn.Set(k, changed)
						case "whitespace":
							changed = append([]byte(" \n"), original...)
							err = tx.txn.Set(k, changed)
						case "extension":
							changed = append(bytes.Clone(original[:len(original)-1]), []byte(`,"extra":null}`)...)
							err = tx.txn.Set(k, changed)
						}
						if err != nil {
							return err
						}
						if stage == "before-first" || stage == "between" {
							_ = l.put(2, k, original)
						}
						return nil
					}, func(*Store) error {
						r, e := l.verify()
						if e == nil || !reflect.DeepEqual(r, identityFactLedgerReport{}) {
							t.Fatal("untracked final change certified")
						}
						return nil
					})
					if err != errIdentityFactLedger || events != 0 || !reflect.DeepEqual(before, independentLedgerSnapshot(t, s)) {
						t.Fatal("ignored failure did not roll back exactly")
					}
				})
			}
		}
	}
}

func TestIndependentLedgerOwnerIsolationAndExpiredCapabilities(t *testing.T) {
	for _, abort := range []bool{false, true} {
		t.Run(fmt.Sprint(abort), func(t *testing.T) {
			a, b := openTemp(t), openTemp(t)
			independentLedgerSeed(t, a, 2)
			independentLedgerSeed(t, b, 2)
			beforeA, beforeB := independentLedgerSnapshot(t, a), independentLedgerSnapshot(t, b)
			var la, lb *identityFactLedger
			forced := errors.New("synthetic outer abort")
			err := runIdentityAdmission(a, func(ta *Store) error {
				var err error
				la, err = beginIdentityFactLedger(ta)
				if err != nil {
					return err
				}
				k, v := independentLedgerRow(t, 70, 1)
				if err = la.put(1, k, v); err != nil {
					return err
				}
				if !reflect.DeepEqual(beforeA, independentLedgerSnapshot(t, a)) {
					t.Fatal("uncommitted ledger visible outside owner")
				}
				return runIdentityAdmission(b, func(tb *Store) error {
					var err error
					lb, err = beginIdentityFactLedger(tb)
					if err != nil {
						return err
					}
					k, v := independentLedgerRow(t, 71, 1)
					return lb.put(1, k, v)
				}, func(*Store) error { _, err := lb.verify(); return err })
			}, func(*Store) error {
				_, err := la.verify()
				if err != nil {
					return err
				}
				if abort {
					return forced
				}
				return nil
			})
			if (abort && err != forced) || (!abort && err != nil) {
				t.Fatal("owner outcome mismatch")
			}
			if reflect.DeepEqual(beforeB, independentLedgerSnapshot(t, b)) {
				t.Fatal("independent owner did not commit")
			}
			if abort && !reflect.DeepEqual(beforeA, independentLedgerSnapshot(t, a)) {
				t.Fatal("outer abort affected isolation")
			}
			for _, l := range []*identityFactLedger{nil, {}, la, lb} {
				k, v := independentLedgerRow(t, 72, 1)
				if err := l.put(0, k, v); err != errIdentityFactLedger {
					t.Fatal("expired or empty put accepted")
				}
				if err := l.delete(0, k); err != errIdentityFactLedger {
					t.Fatal("expired or empty delete accepted")
				}
				r, err := l.verify()
				if err != errIdentityFactLedger || !reflect.DeepEqual(r, identityFactLedgerReport{}) {
					t.Fatal("expired or empty verify accepted")
				}
			}
		})
	}
	// Deliberate foreign-facade presentation exercises the capability identity
	// check, not arbitrary corruption of internal before/after accounting fields.
	a, b := openTemp(t), openTemp(t)
	independentLedgerSeed(t, a, 1)
	independentLedgerSeed(t, b, 1)
	beforeB := independentLedgerSnapshot(t, b)
	var la *identityFactLedger
	if err := runIdentityAdmission(a, func(ta *Store) error {
		var err error
		la, err = beginIdentityFactLedger(ta)
		if err != nil {
			return err
		}
		err = runIdentityAdmission(b, func(tb *Store) error {
			own, err := beginIdentityFactLedger(tb)
			if err != nil {
				return err
			}
			k, v := independentLedgerRow(t, 90, 0)
			if err = own.put(0, k, v); err != nil {
				return err
			}
			foreign := *la
			foreign.st = tb
			if err = foreign.put(0, k, v); err != errIdentityFactLedger {
				t.Fatal("foreign facade accepted")
			}
			return nil
		}, func(*Store) error { t.Fatal("foreign poisoned owner finalized"); return nil })
		if err != errIdentityFactLedger || !reflect.DeepEqual(beforeB, independentLedgerSnapshot(t, b)) {
			t.Fatal("foreign owner failed rollback")
		}
		return nil
	}, func(*Store) error { _, err := la.verify(); return err }); err != nil {
		t.Fatal("separate legitimate owner was poisoned")
	}
}

func TestIndependentLedgerPanicAndRealCommitConflict(t *testing.T) {
	for _, phase := range []string{"body", "finalizer"} {
		t.Run(phase, func(t *testing.T) {
			s := openTemp(t)
			independentLedgerSeed(t, s, 1)
			before := independentLedgerSnapshot(t, s)
			events := 0
			s.SetObserver(func(Event) { events++ })
			var l *identityFactLedger
			panicked := false
			func() {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				_ = runIdentityAdmission(s, func(tx *Store) error {
					var err error
					l, err = beginIdentityFactLedger(tx)
					if err != nil {
						return err
					}
					k, v := independentLedgerRow(t, 77, 1)
					if err = l.put(1, k, v); err != nil {
						return err
					}
					if phase == "body" {
						panic("synthetic panic")
					}
					return nil
				}, func(*Store) error {
					if _, err := l.verify(); err != nil {
						return err
					}
					panic("synthetic panic")
				})
			}()
			if !panicked || events != 0 || !reflect.DeepEqual(before, independentLedgerSnapshot(t, s)) {
				t.Fatal("panic rollback failed")
			}
			if _, err := l.verify(); err != errIdentityFactLedger {
				t.Fatal("panic did not expire ledger")
			}
		})
	}
	t.Run("real-commit-conflict", func(t *testing.T) {
		s := openTemp(t)
		independentLedgerSeed(t, s, 1)
		want := independentLedgerSnapshot(t, s)
		events := 0
		s.SetObserver(func(Event) { events++ })
		var l *identityFactLedger
		err := runIdentityAdmission(s, func(tx *Store) error {
			var err error
			l, err = beginIdentityFactLedger(tx)
			if err != nil {
				return err
			}
			k, v := independentLedgerRow(t, 0, 1)
			if err = l.put(0, k, v); err != nil {
				return err
			}
			k, v = independentLedgerRow(t, 99, 1)
			return l.put(1, k, v)
		}, func(*Store) error {
			if _, err := l.verify(); err != nil {
				return err
			}
			k, v := independentLedgerRow(t, 0, 2)
			want[string(k)] = bytes.Clone(v)
			return s.db.Update(func(competing *badger.Txn) error { return competing.Set(k, v) })
		})
		if !errors.Is(err, badger.ErrConflict) || events != 0 || !reflect.DeepEqual(want, independentLedgerSnapshot(t, s)) {
			t.Fatal("conflicting ledger partially committed")
		}
	})
}

func TestIndependentLedgerRealCapacityFailure(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if err != nil {
		t.Fatal(err)
	}
	s := &Store{db: db}
	defer s.Close()
	independentLedgerSeed(t, s, 2)
	before := independentLedgerSnapshot(t, s)
	events := 0
	s.SetObserver(func(Event) { events++ })
	written := 0
	err = runIdentityAdmission(s, func(tx *Store) error {
		l, err := beginIdentityFactLedger(tx)
		if err != nil {
			return err
		}
		for n := 10; n < 4010; n++ {
			k, v := independentLedgerRow(t, n, 1)
			err = l.put(n, k, v)
			if err != nil {
				if !errors.Is(err, badger.ErrTxnTooBig) || strings.Contains(err.Error(), "independent-") {
					t.Fatal("actual capacity error lost classification or leaked content")
				}
				return nil
			}
			written++
		}
		return nil
	}, func(*Store) error { t.Fatal("ignored capacity failure reached finalizer"); return nil })
	if written == 0 || written == 4000 || !errors.Is(err, badger.ErrTxnTooBig) || events != 0 || !reflect.DeepEqual(before, independentLedgerSnapshot(t, s)) {
		t.Fatal("real late capacity failure did not roll back")
	}
}

func TestIndependentLedgerRefusalAfterSuccessfulMutation(t *testing.T) {
	for _, mode := range []string{"nil-key", "empty-key", "nil-body", "empty-body", "mismatch-address", "duplicate-known", "invalid-history", "unknown-surrogate", "late-begin", "early-verify", "repeated-verify", "negative-delete", "absent-delete", "body-error", "final-error", "prior-poison"} {
		t.Run(mode, func(t *testing.T) {
			s := openTemp(t)
			independentLedgerSeed(t, s, 2)
			before := independentLedgerSnapshot(t, s)
			events := 0
			s.SetObserver(func(Event) { events++ })
			canary := errors.New("SYNTHETIC PRIVATE EARLIER ERROR")
			var l *identityFactLedger
			err := runIdentityAdmission(s, func(tx *Store) error {
				var err error
				l, err = beginIdentityFactLedger(tx)
				if err != nil {
					return err
				}
				k, v := independentLedgerRow(t, 4, 2)
				if err = l.put(7, k, v); err != nil {
					return err
				}
				var e error
				switch mode {
				case "nil-key":
					e = l.put(1, nil, v)
				case "empty-key":
					e = l.put(1, []byte{}, v)
				case "nil-body":
					e = l.put(1, k, nil)
				case "empty-body":
					e = l.put(1, k, []byte{})
				case "mismatch-address":
					other, _ := independentLedgerRow(t, 5, 2)
					e = l.put(1, other, v)
				case "duplicate-known":
					bad := append(bytes.Clone(v[:len(v)-1]), []byte(`,"SRC":"independent-004"}`)...)
					e = l.put(1, k, bad)
				case "invalid-history":
					bad := append(bytes.Clone(v[:len(v)-1]), []byte(`,"invalid_at":"2500-01-01T00:00:00Z"}`)...)
					e = l.put(1, k, bad)
				case "unknown-surrogate":
					bad := append(bytes.Clone(v[:len(v)-1]), []byte(`,"extension":"\ud800"}`)...)
					e = l.put(1, k, bad)
				case "early-verify":
					_, e = l.verify()
				case "negative-delete":
					e = l.delete(-1, k)
				case "absent-delete":
					other, _ := independentLedgerRow(t, 98, 2)
					e = l.delete(1, other)
				case "body-error":
					return canary
				case "prior-poison":
					tx.poisonAdmission(canary)
					e = l.put(1, k, v)
				}
				if e != nil && e != errIdentityFactLedger {
					t.Fatal("ledger leaked arbitrary error")
				}
				return nil
			}, func(tx *Store) error {
				if mode == "late-begin" {
					_, e := beginIdentityFactLedger(tx)
					if e != errIdentityFactLedger {
						t.Fatal("late begin accepted")
					}
					return nil
				}
				_, e := l.verify()
				if e != nil {
					return e
				}
				if mode == "repeated-verify" {
					r, e := l.verify()
					if e != errIdentityFactLedger || !reflect.DeepEqual(r, identityFactLedgerReport{}) {
						t.Fatal("repeat accepted")
					}
				}
				if mode == "final-error" {
					return canary
				}
				return nil
			})
			want := errIdentityFactLedger
			if mode == "body-error" || mode == "final-error" || mode == "prior-poison" {
				want = canary
			}
			if err != want || events != 0 || !reflect.DeepEqual(before, independentLedgerSnapshot(t, s)) {
				t.Fatal("refusal error precedence, event or raw rollback failed")
			}
		})
	}
}
