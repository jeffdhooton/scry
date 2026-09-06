package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func ledgerFixture(t *testing.T, st *Store, slug string) ([]byte, []byte) {
	t.Helper()
	f := referenceFixture()
	f.Src = slug
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	return key, raw
}

func TestFactLedgerExactMutationSequenceAndReconstruction(t *testing.T) {
	st := openTemp(t)
	keys, raws := [][]byte{}, [][]byte{}
	for _, slug := range []string{"aaa", "ccc", "eee", "ggg", "zzz"} {
		key, raw := ledgerFixture(t, st, slug)
		keys = append(keys, key)
		raws = append(raws, raw)
	}
	gradeGenerationSet(t, st, "opaque:untouched", []byte{255})
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	var ledger *identityFactLedger
	var report identityFactLedgerReport
	err := runIdentityAdmission(st, func(tx *Store) error {
		var err error
		ledger, err = beginIdentityFactLedger(tx)
		if err != nil {
			return err
		}
		// Deleted originals occur before, between and after untouched originals.
		for _, i := range []int{0, 2, 4} {
			if err := ledger.delete(7, keys[i]); err != nil {
				return err
			}
		}
		var f Fact
		json.Unmarshal(raws[1], &f)
		f.InvalidAt = &f.ValidFrom
		history, _ := json.Marshal(f)
		if err := ledger.put(8, keys[1], history); err != nil {
			return err
		}
		// Noop, delete/reinsert, and new-then-delete retain the complete sequence.
		if err := ledger.put(8, keys[1], history); err != nil {
			return err
		}
		if err := ledger.delete(9, keys[1]); err != nil {
			return err
		}
		if err := ledger.put(10, keys[1], history); err != nil {
			return err
		}
		f.Src = "new"
		f.Dst = ""
		f.Value = "attribute"
		f.InvalidAt = nil
		key, raw := referenceBytes(t, f)
		if err := ledger.put(11, key, raw); err != nil {
			return err
		}
		if err := ledger.delete(12, key); err != nil {
			return err
		}
		f.Src, f.Dst, f.Value = "loop", "loop", ""
		key, raw = referenceBytes(t, f)
		return ledger.put(13, key, raw)
	}, func(tx *Store) error {
		var err error
		report, err = ledger.verify()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Final.Scanned != 3 || report.Final.References["ccc"].Historical != 1 || report.Final.References["loop"].Current != 1 || len(report.Mutations) != 10 {
		t.Fatal("wrong final ledger")
	}
	for i, want := range []int{7, 7, 7, 8, 8, 9, 10, 11, 12, 13} {
		if report.Mutations[i].Ordinal != want {
			t.Fatal("ordinal order changed")
		}
	}
	after := gradeGenerationSnapshot(t, st)
	if report.Final.Digest != independentInventoryDigest(after) || events != 0 || !bytes.Equal(after["opaque:untouched"], before["opaque:untouched"]) {
		t.Fatal("digest/events/unrelated bytes")
	}
	if report.Mutations[0].Before.Exists != true || report.Mutations[0].After.Exists || !bytes.Equal(report.Mutations[0].Before.Raw, raws[0]) {
		t.Fatal("before-image lost")
	}
	report.Mutations[0].Before.Raw[0] = 0
	if !bytes.Equal(ledger.entries[string(keys[0])].before.Raw, raws[0]) {
		t.Fatal("report aliases ledger")
	}
}

func TestFactLedgerUntrackedChangesPoisonAndRollback(t *testing.T) {
	for _, kind := range []string{"untouched-put", "untouched-delete", "unknown-extension", "before-first-write", "after-last-write", "between-tracked-writes", "untracked-insert", "untracked-recreate"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			key, raw := ledgerFixture(t, st, "source")
			before := gradeGenerationSnapshot(t, st)
			var l *identityFactLedger
			err := runIdentityAdmission(st, func(tx *Store) error {
				var err error
				l, err = beginIdentityFactLedger(tx)
				if err != nil {
					return err
				}
				var f Fact
				json.Unmarshal(raw, &f)
				f.Confidence = .125
				changed, _ := json.Marshal(f)
				switch kind {
				case "untouched-put":
					return tx.txn.Set(key, changed)
				case "untouched-delete":
					return tx.txn.Delete(key)
				case "unknown-extension":
					return tx.txn.Set(key, append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"new":1}`)...))
				case "before-first-write":
					if err := tx.txn.Set(key, changed); err != nil {
						return err
					}
					return l.put(0, key, raw)
				case "after-last-write", "between-tracked-writes":
					if err := l.put(0, key, raw); err != nil {
						return err
					}
					if err := tx.txn.Set(key, changed); err != nil {
						return err
					}
					if kind == "between-tracked-writes" {
						_ = l.put(1, key, raw)
					}
					return nil
				case "untracked-insert":
					f.Src = "foreign"
					k, v := referenceBytes(t, f)
					return tx.txn.Set(k, v)
				case "untracked-recreate":
					if err := l.delete(0, key); err != nil {
						return err
					}
					return tx.txn.Set(key, raw)
				}
				return nil
			}, func(*Store) error {
				report, err := l.verify()
				if !reflect.DeepEqual(report, identityFactLedgerReport{}) {
					t.Fatal("partial report on unattributed writes")
				}
				_ = err
				return nil
			})
			if !errors.Is(err, errIdentityFactLedger) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("untracked change committed")
			}
		})
	}
}

func TestFactLedgerCallerOwnershipAndOpaqueRaw(t *testing.T) {
	st := openTemp(t)
	key, raw := ledgerFixture(t, st, "source")
	raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":1e99999,"unknown":{"x":"😀"}}`)...)
	gradeGenerationSet(t, st, string(key), raw)
	before := gradeGenerationSnapshot(t, st)
	var l *identityFactLedger
	forced := errors.New("synthetic rollback")
	err := runIdentityAdmission(st, func(tx *Store) error {
		var err error
		l, err = beginIdentityFactLedger(tx)
		if err != nil {
			return err
		}
		k, v := bytes.Clone(key), bytes.Clone(raw)
		if err := l.put(0, k, v); err != nil {
			return err
		}
		for i := range k {
			k[i] = 0
		}
		for i := range v {
			v[i] = 0
		}
		return nil
	}, func(*Store) error {
		r, err := l.verify()
		if err != nil {
			return err
		}
		if r.Final.Digest != independentInventoryDigest(before) || !bytes.Equal(r.Mutations[0].Key, key) || !bytes.Equal(r.Mutations[0].After.Raw, raw) {
			t.Fatal("exact caller bytes lost")
		}
		return forced
	})
	if err != forced || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("rollback failed")
	}
}

func TestFactLedgerScopeAndMalformedControls(t *testing.T) {
	for _, kind := range []string{"root", "ordinary", "negative-ordinal", "wrong-key", "malformed-put", "missing-delete", "early-verify", "late-put", "twice-verify", "malformed-final", "prior-failure"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			key, raw := ledgerFixture(t, st, "source")
			before := gradeGenerationSnapshot(t, st)
			if kind == "root" {
				if _, err := beginIdentityFactLedger(st); err != errIdentityFactLedger {
					t.Fatal("root ledger accepted")
				}
				return
			}
			if kind == "ordinary" {
				if err := st.AtomicWrite(func(tx *Store) error {
					_, err := beginIdentityFactLedger(tx)
					if err != errIdentityFactLedger {
						t.Fatal("ordinary ledger accepted")
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				return
			}
			var l *identityFactLedger
			err := runIdentityAdmission(st, func(tx *Store) error {
				var err error
				l, err = beginIdentityFactLedger(tx)
				if err != nil {
					return err
				}
				switch kind {
				case "negative-ordinal":
					_ = l.put(-1, key, raw)
				case "wrong-key":
					_ = l.put(0, []byte("en:source"), raw)
				case "malformed-put":
					_ = l.put(0, key, []byte("null"))
				case "missing-delete":
					_ = l.delete(0, []byte("fa:absent"))
				case "early-verify":
					_, _ = l.verify()
				case "malformed-final":
					return tx.txn.Set([]byte("fa:zz"), []byte("null"))
				case "prior-failure":
					tx.poisonAdmission(errors.New("synthetic private detail"))
					if err := l.put(0, key, raw); err != errIdentityFactLedger {
						t.Fatal("existing poison leaked from ledger")
					}
				}
				return nil
			}, func(*Store) error {
				if kind == "late-put" {
					_ = l.put(0, key, raw)
					return nil
				}
				_, _ = l.verify()
				if kind == "twice-verify" {
					_, _ = l.verify()
				}
				return nil
			})
			if err == nil || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("caught scope/data refusal committed")
			}
		})
	}
	st := openTemp(t)
	gradeGenerationSet(t, st, "fa:malformed", []byte("null"))
	before := gradeGenerationSnapshot(t, st)
	err := runIdentityAdmission(st, func(tx *Store) error { _, _ = beginIdentityFactLedger(tx); return nil }, func(*Store) error { return nil })
	if err != errIdentityFactLedger || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("malformed baseline accepted")
	}
}

func TestFactLedgerRealLateSizeFailure(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if err != nil {
		t.Fatal("fixture open failed")
	}
	st := &Store{db: db}
	defer st.Close()
	before := gradeGenerationSnapshot(t, st)
	staged := 0
	err = runIdentityAdmission(st, func(tx *Store) error {
		l, err := beginIdentityFactLedger(tx)
		if err != nil {
			return err
		}
		for i := 0; i < 3000; i++ {
			f := referenceFixture()
			f.Src = fmt.Sprintf("synthetic-%04d", i)
			key, raw := referenceBytes(t, f)
			if err := l.put(i, key, raw); err != nil {
				if !errors.Is(err, badger.ErrTxnTooBig) || strings.Contains(err.Error(), "synthetic-") {
					t.Fatal("bad actual size classification")
				}
				return nil // Ignored failure MUST poison the scope.
			}
			staged++
		}
		return nil
	}, func(*Store) error { t.Fatal("failed ledger reached finalizer"); return nil })
	if staged == 0 || staged == 3000 || !errors.Is(err, badger.ErrTxnTooBig) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("late size failure did not roll back")
	}
	// Separate real value limit after one successful mutation.
	err = runIdentityAdmission(st, func(tx *Store) error {
		l, err := beginIdentityFactLedger(tx)
		if err != nil {
			return err
		}
		f := referenceFixture()
		key, raw := referenceBytes(t, f)
		if err := l.put(0, key, raw); err != nil {
			return err
		}
		f.Src = "large"
		f.Fact = strings.Repeat("private canary ", 1000)
		key, raw = referenceBytes(t, f)
		if err := l.put(1, key, raw); err != errIdentityFactLedger {
			t.Fatal("unsafe real value error")
		}
		return nil
	}, func(*Store) error { t.Fatal("value failure reached finalizer"); return nil })
	if err != errIdentityFactLedger || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("value failure partially committed")
	}
}

func TestFactLedgerClosedAndForeignOwner(t *testing.T) {
	st := openTemp(t)
	key, raw := ledgerFixture(t, st, "source")
	var old *identityFactLedger
	if err := runIdentityAdmission(st, func(tx *Store) error { var err error; old, err = beginIdentityFactLedger(tx); return err }, func(*Store) error { _, err := old.verify(); return err }); err != nil {
		t.Fatal(err)
	}
	if err := old.put(0, key, raw); err != errIdentityFactLedger {
		t.Fatal("closed ledger accepted")
	}
	if _, err := old.verify(); err != errIdentityFactLedger {
		t.Fatal("closed verify accepted")
	}
	before := gradeGenerationSnapshot(t, st)
	err := runIdentityAdmission(st, func(tx *Store) error {
		l, err := beginIdentityFactLedger(tx)
		if err != nil {
			return err
		}
		l.owner = &identityAdmissionOwner{phase: admissionBody}
		_ = l.put(0, key, raw)
		return nil
	}, func(*Store) error { return nil })
	if err != errIdentityFactLedger || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("foreign owner accepted")
	}
}
