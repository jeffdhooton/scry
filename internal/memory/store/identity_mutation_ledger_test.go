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

	"github.com/dgraph-io/badger/v4"
)

func TestIdentityMutationLedgerGeneratedReplay(t *testing.T) {
	st := openTemp(t)
	gradeGenerationSet(t, st, "ar:opaque", []byte{255, 0})
	gradeGenerationSet(t, st, "al:empty", []byte{})
	baseline, err := scanIdentityRelationships(st)
	if err != nil {
		t.Fatal(err)
	}
	model := map[string]identityImage{}
	for k, v := range baseline.Rows {
		model[k] = identityImage{Exists: true, Value: bytes.Clone(v)}
	}
	var ledger *identityMutationLedger
	var report identityMutationReport
	expected := []identityWriterEntry{}
	events := 0
	st.SetObserver(func(Event) { events++ })
	err = runIdentityAdmission(st, func(tx *Store) error {
		var err error
		ledger, err = beginIdentityMutationLedger(tx)
		if err != nil {
			return err
		}
		rng := rand.New(rand.NewSource(718))
		for i := 0; i < 500; i++ {
			actor := []string{"atlas", "borealis", "cygnus"}[rng.Intn(3)]
			key := []byte(fmt.Sprintf("al:shared-%d", rng.Intn(9)))
			value := []byte(actor)
			if i%7 == 0 {
				key, value = writerEntity(t, actor)
			}
			before := cloneWriterImage(model[string(key)])
			after := identityImage{Exists: true, Value: bytes.Clone(value)}
			if i%4 == 0 && before.Exists && strings.HasPrefix(string(key), "al:") {
				actor = string(before.Value)
				after = identityImage{}
			}
			if after.Exists {
				err = ledger.put(actor, key, value)
			} else {
				err = ledger.delete(actor, key)
			}
			if err != nil {
				return err
			}
			expected = append(expected, identityWriterEntry{Sequence: uint64(i), Actor: actor, Key: bytes.Clone(key), Before: before, After: cloneWriterImage(after)})
			model[string(key)] = cloneWriterImage(after)
			for j := range key {
				key[j] = 0
			}
			for j := range value {
				value[j] = 0
			}
		}
		return nil
	}, func(*Store) error { var err error; report, err = ledger.verify(); return err })
	if err != nil || events != 0 || !reflect.DeepEqual(report.Baseline, baseline) || !reflect.DeepEqual(report.History.Entries, expected) {
		t.Fatal("generated accounting diverged", err)
	}
	final := map[string][]byte{}
	for k, v := range model {
		if v.Exists {
			final[k] = v.Value
		}
	}
	if !reflect.DeepEqual(report.Final.Rows, final) || report.Final.Digest != relationshipDigest(final) {
		t.Fatal("complete final raw mismatch")
	}
	actual, err := scanIdentityRelationships(st)
	if err != nil || !reflect.DeepEqual(actual, report.Final) {
		t.Fatal("committed inventory differs")
	}
}

func TestIdentityMutationLedgerAllUntrackedTransitionsRefuse(t *testing.T) {
	for _, when := range []string{"before-first", "after-last", "other-key"} {
		for _, change := range []string{"create", "update", "delete", "present-empty"} {
			t.Run(when+change, func(t *testing.T) {
				st := openTemp(t)
				if change != "create" {
					gradeGenerationSet(t, st, "al:target", []byte("old-owner"))
				}
				before := gradeGenerationSnapshot(t, st)
				var l *identityMutationLedger
				err := runIdentityAdmission(st, func(tx *Store) error {
					var err error
					l, err = beginIdentityMutationLedger(tx)
					if err != nil {
						return err
					}
					tracked := []byte("al:target")
					if when == "other-key" {
						tracked = []byte("al:tracked")
					}
					if when != "before-first" {
						if err := l.put("atlas", tracked, []byte("atlas")); err != nil {
							return err
						}
					}
					switch change {
					case "delete":
						err = tx.txn.Delete([]byte("al:target"))
					case "present-empty":
						err = tx.txn.Set([]byte("al:target"), []byte{})
					default:
						err = tx.txn.Set([]byte("al:target"), []byte("untracked-owner"))
					}
					if err != nil {
						return err
					}
					if when == "before-first" {
						return l.put("atlas", tracked, []byte("atlas"))
					}
					return nil
				}, func(*Store) error {
					got, err := l.verify()
					if err != errIdentityMutationLedger || !reflect.DeepEqual(got, identityMutationReport{}) {
						t.Fatal("untracked write certified")
					}
					return nil
				})
				if err == nil || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("ignored verification refusal committed")
				}
			})
		}
	}
	for _, family := range []string{"ar:", "ig:", "il:", "il-consumed:", "meta:identity_", "rs:", "rt:"} {
		for _, change := range []string{"create", "update", "delete", "present-empty"} {
			t.Run(family+change, func(t *testing.T) {
				st := openTemp(t)
				key := []byte(family + "synthetic")
				if change != "create" {
					gradeGenerationSet(t, st, string(key), []byte("old"))
				}
				before := gradeGenerationSnapshot(t, st)
				var l *identityMutationLedger
				err := runIdentityAdmission(st, func(tx *Store) error {
					var err error
					l, err = beginIdentityMutationLedger(tx)
					if err != nil {
						return err
					}
					if err := l.put("atlas", []byte("al:tracked"), []byte("atlas")); err != nil {
						return err
					}
					switch change {
					case "delete":
						return tx.txn.Delete(key)
					case "present-empty":
						return tx.txn.Set(key, []byte{})
					default:
						return tx.txn.Set(key, []byte("new"))
					}
				}, func(*Store) error { _, _ = l.verify(); return nil })
				if err != errIdentityMutationLedger || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("control drift committed")
				}
			})
		}
	}
}

func TestIdentityMutationLedgerOwnedReportAndNoop(t *testing.T) {
	st := openTemp(t)
	relationshipEntity(t, st, "atlas", "Atlas", []string{"shared"})
	gradeGenerationSet(t, st, "al:shared", []byte("atlas"))
	before := gradeGenerationSnapshot(t, st)
	var l *identityMutationLedger
	err := runIdentityAdmission(st, func(tx *Store) error { var err error; l, err = beginIdentityMutationLedger(tx); return err }, func(*Store) error {
		r, err := l.verify()
		if err != nil {
			return err
		}
		if len(r.History.Entries) != 0 || !reflect.DeepEqual(r.Baseline, r.Final) {
			t.Fatal("zero-operation report")
		}
		original := cloneIdentityRelationshipInventory(l.baseline)
		for _, v := range []*identityRelationshipInventory{&r.Baseline, &r.Final} {
			for _, raw := range v.Rows {
				for i := range raw {
					raw[i] = 0
				}
			}
			v.Entities["atlas"].Aliases[0] = "changed"
			v.Listings["shared"][0].Spelling = "changed"
			v.NaturalListings["shared"][0].Slug = "changed"
			v.IndexTargets["atlas"][0] = "changed"
			v.FamilyCounts["en:"] = 999
		}
		if !reflect.DeepEqual(original, l.baseline) {
			t.Fatal("report mutated private baseline")
		}
		return nil
	})
	if err != nil || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("read-only scope changed state", err)
	}
}

func TestIdentityMutationLedgerScopePoisonAndPanic(t *testing.T) {
	for _, kind := range []string{"bad-put", "bad-delete", "early-verify", "late-put", "repeat-verify", "prior-poison", "body-panic", "finalizer-panic", "foreign-owner", "malformed-baseline", "malformed-final"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			if kind == "malformed-baseline" {
				gradeGenerationSet(t, st, "en:broken", []byte("opaque"))
			}
			before := gradeGenerationSnapshot(t, st)
			var l *identityMutationLedger
			panicked := false
			var err error
			prior := errors.New("synthetic prior error")
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
					if err := l.put("atlas", []byte("al:good"), []byte("atlas")); err != nil {
						return err
					}
					switch kind {
					case "bad-put":
						if err := l.put("atlas", []byte("al:bad"), []byte("borealis")); err != errIdentityMutationLedger {
							t.Fatal("bad local sentinel")
						}
					case "bad-delete":
						_ = l.delete("atlas", []byte("al:absent"))
					case "early-verify":
						_, _ = l.verify()
					case "prior-poison":
						tx.poisonAdmission(prior)
						if err := l.put("atlas", []byte("al:later"), []byte("atlas")); err != errIdentityMutationLedger {
							t.Fatal("prior error leaked locally")
						}
					case "body-panic":
						panic("synthetic body")
					case "foreign-owner":
						l.owner = &identityAdmissionOwner{phase: admissionBody}
						_ = l.delete("atlas", []byte("al:good"))
					case "malformed-final":
						return tx.txn.Set([]byte("en:bad"), []byte("opaque"))
					}
					return nil
				}, func(*Store) error {
					if kind == "late-put" {
						_ = l.put("atlas", []byte("al:later"), []byte("atlas"))
						return nil
					}
					_, err := l.verify()
					if kind == "repeat-verify" {
						_, _ = l.verify()
					}
					if kind == "finalizer-panic" {
						panic("synthetic finalizer")
					}
					return err
				})
			}()
			if err == nil && !panicked {
				t.Fatal("failed scope accepted")
			}
			if kind == "prior-poison" && err != prior {
				t.Fatal("prior precedence lost")
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("scope failure committed")
			}
			if l != nil {
				if err := l.put("atlas", []byte("al:closed"), []byte("atlas")); err != errIdentityMutationLedger {
					t.Fatal("closed accepted")
				}
			}
		})
	}
	st := openTemp(t)
	for _, bad := range []*Store{nil, st, {}} {
		if _, err := beginIdentityMutationLedger(bad); err != errIdentityMutationLedger {
			t.Fatal("unowned begin")
		}
	}
	for _, l := range []*identityMutationLedger{nil, {}} {
		if _, err := l.verify(); err != errIdentityMutationLedger {
			t.Fatal("empty verify")
		}
	}
}

func TestIdentityMutationLedgerActualStorageFailureAndConflict(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if err != nil {
		t.Fatal("synthetic db")
	}
	st := &Store{db: db}
	defer st.Close()
	before := gradeGenerationSnapshot(t, st)
	for _, kind := range []string{"value", "capacity"} {
		err := runIdentityAdmission(st, func(tx *Store) error {
			l, err := beginIdentityMutationLedger(tx)
			if err != nil {
				return err
			}
			if err := l.put("atlas", []byte("al:good"), []byte("atlas")); err != nil {
				return err
			}
			if kind == "value" {
				key, raw := writerEntity(t, "atlas")
				var e Entity
				json.Unmarshal(raw, &e)
				e.Description = strings.Repeat("synthetic-private ", 1000)
				raw, _ = json.Marshal(e)
				if err := l.put("atlas", key, raw); err != errIdentityMutationLedger {
					t.Fatal("unsafe local storage error")
				}
				return nil
			}
			for i := 0; i < 20000; i++ {
				if err := l.put("atlas", []byte(fmt.Sprintf("al:synthetic-%d", i)), []byte("atlas")); err != nil {
					if !errors.Is(err, badger.ErrTxnTooBig) {
						t.Fatal("classification lost")
					}
					return nil
				}
			}
			t.Fatal("capacity did not fail")
			return nil
		}, func(*Store) error { t.Fatal("failed writer finalized"); return nil })
		if err == nil || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("late storage write committed")
		}
	}
	gradeGenerationSet(t, st, "al:shared", []byte("old"))
	before = gradeGenerationSnapshot(t, st)
	var l *identityMutationLedger
	err = runIdentityAdmission(st, func(tx *Store) error {
		var err error
		l, err = beginIdentityMutationLedger(tx)
		if err != nil {
			return err
		}
		if err := l.put("atlas", []byte("al:shared"), []byte("atlas")); err != nil {
			return err
		}
		if err := l.put("atlas", []byte("al:staged"), []byte("atlas")); err != nil {
			return err
		}
		return st.db.Update(func(other *badger.Txn) error { return other.Set([]byte("al:shared"), []byte("competitor")) })
	}, func(*Store) error { _, err := l.verify(); return err })
	before["al:shared"] = []byte("competitor")
	if !errors.Is(err, badger.ErrConflict) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("real conflict leaked scope writes", err)
	}
}
