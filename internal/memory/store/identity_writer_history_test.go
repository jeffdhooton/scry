package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func writerEntity(t *testing.T, slug string) ([]byte, []byte) {
	t.Helper()
	e := Entity{Slug: slug, Name: "Explicit " + slug, Type: "project", CreatedAt: time.Date(2026, 9, 6, 0, 0, 0, 1, time.UTC)}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(prefixEntity + slug), raw
}

func TestWriterHistoryMultiActorExactSequence(t *testing.T) {
	st := openTemp(t)
	gradeGenerationSet(t, st, "opaque:old", []byte{255, 0})
	before := gradeGenerationSnapshot(t, st)
	var h *identityWriterHistory
	var snapshot identityWriterSnapshot
	events := 0
	st.SetObserver(func(Event) { events++ })
	err := runIdentityAdmission(st, func(tx *Store) error {
		var err error
		h, err = newIdentityWriterHistory(tx)
		if err != nil {
			return err
		}
		for _, slug := range []string{"atlas", "borealis"} {
			key, raw := writerEntity(t, slug)
			if err := h.put(slug, key, raw); err != nil {
				return err
			}
		}
		if err := h.put("atlas", []byte("al:shared"), []byte("atlas")); err != nil {
			return err
		}
		if err := h.put("atlas", []byte("al:shared"), []byte("atlas")); err != nil {
			return err
		}
		if err := h.delete("atlas", []byte("al:shared")); err != nil {
			return err
		}
		return h.put("borealis", []byte("al:shared"), []byte("borealis"))
	}, func(*Store) error { var err error; snapshot, err = h.freeze(); return err })
	if err != nil || len(snapshot.Entries) != 6 || len(snapshot.Before) != 3 || snapshot.Before["al:shared"].Exists || string(snapshot.Final["al:shared"].Value) != "borealis" {
		t.Fatal("wrong multi-actor history", err)
	}
	for i, actor := range []string{"atlas", "borealis", "atlas", "atlas", "atlas", "borealis"} {
		if snapshot.Entries[i].Sequence != uint64(i) || snapshot.Entries[i].Actor != actor {
			t.Fatal("operation attribution changed")
		}
	}
	if !snapshot.Entries[3].Before.Exists || !bytes.Equal(snapshot.Entries[3].Before.Value, snapshot.Entries[3].After.Value) || snapshot.Entries[4].After.Exists || snapshot.Entries[5].Before.Exists {
		t.Fatal("noop/delete/reclaim states lost")
	}
	after := gradeGenerationSnapshot(t, st)
	if len(after) != len(before)+3 || events != 0 || !bytes.Equal(before["opaque:old"], after["opaque:old"]) {
		t.Fatal("wrong graph write set/events")
	}
	// This test only records the later owner. No undo or ownership policy ran.
}

func TestWriterHistoryOpaqueBeforeImagesAndOwnedOutput(t *testing.T) {
	st := openTemp(t)
	key, raw := writerEntity(t, "atlas")
	opaque := []byte{0, 255, 1}
	gradeGenerationSet(t, st, string(key), opaque)
	before := gradeGenerationSnapshot(t, st)
	var h *identityWriterHistory
	forced := errors.New("synthetic rollback")
	err := runIdentityAdmission(st, func(tx *Store) error {
		var err error
		h, err = newIdentityWriterHistory(tx)
		if err != nil {
			return err
		}
		k, v := bytes.Clone(key), bytes.Clone(raw)
		if err := h.put("atlas", k, v); err != nil {
			return err
		}
		for i := range k {
			k[i] = 0
		}
		for i := range v {
			v[i] = 0
		}
		if err := h.delete("atlas", key); err != nil {
			return err
		}
		return h.put("atlas", key, raw)
	}, func(*Store) error {
		snap, err := h.freeze()
		if err != nil {
			return err
		}
		if !bytes.Equal(snap.Before[string(key)].Value, opaque) || !bytes.Equal(snap.Entries[0].Before.Value, opaque) || !bytes.Equal(snap.Final[string(key)].Value, raw) {
			t.Fatal("opaque first image or new bytes lost")
		}
		for _, entry := range snap.Entries {
			for i := range entry.Key {
				entry.Key[i] = 0
			}
			for i := range entry.Before.Value {
				entry.Before.Value[i] = 0
			}
			for i := range entry.After.Value {
				entry.After.Value[i] = 0
			}
		}
		for _, image := range snap.Before {
			for i := range image.Value {
				image.Value[i] = 0
			}
		}
		for _, image := range snap.Final {
			for i := range image.Value {
				image.Value[i] = 0
			}
		}
		if !bytes.Equal(h.journal.before[string(key)].Value, opaque) || !bytes.Equal(h.final[string(key)].Value, raw) || !bytes.Equal(h.entries[0].Key, key) {
			t.Fatal("output aliases private history")
		}
		return forced
	})
	if err != forced || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("opaque rollback changed original")
	}
}

func TestWriterHistoryMalformedOperationsPoison(t *testing.T) {
	for _, kind := range []string{"nil-key", "empty-key", "foreign-family", "bad-actor", "wrong-entity-actor", "wrong-alias-owner", "wrong-delete-actor", "absent-delete", "bad-normalization", "empty-alias", "bad-utf8-key", "null-entity", "unknown-entity", "empty-value", "stale-next", "stale-freeze", "early-freeze", "late-write", "repeated-freeze", "prior-poison"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			before := gradeGenerationSnapshot(t, st)
			var h *identityWriterHistory
			err := runIdentityAdmission(st, func(tx *Store) error {
				var err error
				h, err = newIdentityWriterHistory(tx)
				if err != nil {
					return err
				}
				if err := h.put("atlas", []byte("al:shared"), []byte("atlas")); err != nil {
					return err
				}
				key, raw := writerEntity(t, "atlas")
				switch kind {
				case "nil-key":
					_ = h.put("atlas", nil, raw)
				case "empty-key":
					_ = h.put("atlas", []byte{}, raw)
				case "foreign-family":
					_ = h.put("atlas", []byte("att:atlas:x"), raw)
				case "bad-actor":
					_ = h.put("Atlas", key, raw)
				case "wrong-entity-actor":
					_ = h.put("borealis", key, raw)
				case "wrong-alias-owner":
					_ = h.put("atlas", []byte("al:other"), []byte("borealis"))
				case "wrong-delete-actor":
					_ = h.delete("borealis", []byte("al:shared"))
				case "absent-delete":
					_ = h.delete("atlas", []byte("al:absent"))
				case "bad-normalization":
					_ = h.put("atlas", []byte("al:NOT normalized"), []byte("atlas"))
				case "empty-alias":
					_ = h.put("atlas", []byte("al:"), []byte("atlas"))
				case "bad-utf8-key":
					_ = h.put("atlas", []byte("al:bad-\xff"), []byte("atlas"))
				case "null-entity":
					_ = h.put("atlas", key, []byte("null"))
				case "unknown-entity":
					_ = h.put("atlas", key, append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":true}`)...))
				case "empty-value":
					_ = h.put("atlas", key, nil)
				case "stale-next", "stale-freeze":
					if err := tx.txn.Set([]byte("al:shared"), []byte("foreign")); err != nil {
						return err
					}
					if kind == "stale-next" {
						_ = h.put("atlas", []byte("al:shared"), []byte("atlas"))
					}
				case "early-freeze":
					_, _ = h.freeze()
				case "prior-poison":
					tx.poisonAdmission(errors.New("private earlier failure"))
					if err := h.put("atlas", key, raw); err != errIdentityWriterHistory {
						t.Fatal("prior error leaked from helper")
					}
				}
				return nil
			}, func(*Store) error {
				if kind == "late-write" {
					_ = h.put("atlas", []byte("al:later"), []byte("atlas"))
					return nil
				}
				snap, err := h.freeze()
				if err != nil && !reflect.DeepEqual(snap, identityWriterSnapshot{}) {
					t.Fatal("partial refusal snapshot")
				}
				if kind == "repeated-freeze" {
					_, _ = h.freeze()
				}
				return nil
			})
			if err == nil || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("caught refusal committed")
			}
		})
	}
}

func TestWriterHistoryScopesAndPanic(t *testing.T) {
	st := openTemp(t)
	if _, err := newIdentityWriterHistory(nil); err != errIdentityWriterHistory {
		t.Fatal("nil accepted")
	}
	if _, err := newIdentityWriterHistory(st); err != errIdentityWriterHistory {
		t.Fatal("root accepted")
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		_, err := newIdentityWriterHistory(tx)
		if err != errIdentityWriterHistory {
			t.Fatal("ordinary accepted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"body", "finalizer", "foreign-owner", "foreign-journal", "success"} {
		before := gradeGenerationSnapshot(t, st)
		var h *identityWriterHistory
		panicked := false
		var result error
		func() {
			defer func() {
				if recover() != nil {
					panicked = true
				}
			}()
			result = runIdentityAdmission(st, func(tx *Store) error {
				var err error
				h, err = newIdentityWriterHistory(tx)
				if err != nil {
					return err
				}
				if phase == "foreign-owner" {
					h.owner = &identityAdmissionOwner{phase: admissionBody}
				}
				if phase == "foreign-journal" {
					h.journal = &identityJournal{st: st}
				}
				if err := h.put("atlas", []byte("al:shared"), []byte("atlas")); err != nil {
					return err
				}
				if phase == "body" {
					panic("synthetic body panic")
				}
				return nil
			}, func(*Store) error {
				if _, err := h.freeze(); err != nil {
					return err
				}
				if phase == "finalizer" {
					panic("synthetic finalizer panic")
				}
				return nil
			})
		}()
		if phase == "body" || phase == "finalizer" {
			if !panicked {
				t.Fatal("panic missing")
			}
		} else if phase == "success" {
			if result != nil {
				t.Fatal(result)
			}
		} else if result != errIdentityWriterHistory {
			t.Fatal("foreign capability accepted")
		}
		if phase != "success" && !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("failed scope changed store")
		}
		if err := h.put("atlas", []byte("al:later"), []byte("atlas")); err != errIdentityWriterHistory {
			t.Fatal("closed put accepted")
		}
		if _, err := h.freeze(); err != errIdentityWriterHistory {
			t.Fatal("closed freeze accepted")
		}
	}
	for _, h := range []*identityWriterHistory{nil, {}} {
		if err := h.delete("atlas", []byte("al:x")); err != errIdentityWriterHistory {
			t.Fatal("empty capability accepted")
		}
	}
}

func TestWriterHistoryActualLateStorageFailure(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if err != nil {
		t.Fatal("synthetic DB failed")
	}
	st := &Store{db: db}
	defer st.Close()
	before := gradeGenerationSnapshot(t, st)
	for _, kind := range []string{"capacity", "value", "key"} {
		staged := 0
		err := runIdentityAdmission(st, func(tx *Store) error {
			h, err := newIdentityWriterHistory(tx)
			if err != nil {
				return err
			}
			if err := h.put("atlas", []byte("al:first"), []byte("atlas")); err != nil {
				return err
			}
			staged++
			switch kind {
			case "capacity":
				for i := 0; i < 20000; i++ {
					if err := h.put("atlas", []byte(fmt.Sprintf("al:synthetic-%06d", i)), []byte("atlas")); err != nil {
						if !errors.Is(err, badger.ErrTxnTooBig) {
							t.Fatal("real capacity class lost")
						}
						return nil
					}
					staged++
				}
				t.Fatal("capacity fixture did not fail")
			case "value":
				key, raw := writerEntity(t, "atlas")
				var e Entity
				json.Unmarshal(raw, &e)
				e.Description = strings.Repeat("synthetic-private-value ", 1000)
				raw, _ = json.Marshal(e)
				if err := h.put("atlas", key, raw); err != errIdentityWriterHistory {
					t.Fatal("unsafe value error")
				}
			case "key":
				if err := h.put("atlas", []byte("al:"+strings.Repeat("x", 70000)), []byte("atlas")); err != errIdentityWriterHistory {
					t.Fatal("unsafe key error")
				}
			}
			return nil
		}, func(*Store) error { t.Fatal("failed write reached freeze"); return nil })
		if staged == 0 || !errors.Is(err, errIdentityWriterHistory) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("late error partially committed")
		}
	}
}
