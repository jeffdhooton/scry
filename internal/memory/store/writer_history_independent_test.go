package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func independentWriterEntity(t *testing.T, slug string, revision int) []byte {
	t.Helper()
	raw, err := json.Marshal(Entity{Slug: slug, Name: "Legitimate unrelated display", Type: "project", Description: fmt.Sprint(revision), CreatedAt: time.Date(2026, 9, 6, 1, 2, 3, 4, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestIndependentWriterGeneratedReplay(t *testing.T) {
	for _, seed := range []int64{17, 83, 619} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			st := openTemp(t)
			gradeGenerationSet(t, st, "en:atlas", []byte{0, 255, 1})
			gradeGenerationSet(t, st, "al:empty", []byte{})
			gradeGenerationSet(t, st, "att:atlas:sentinel", []byte{255, 0})
			baseline := gradeGenerationSnapshot(t, st)
			model := map[string]identityImage{}
			for k, v := range baseline {
				model[k] = identityImage{Exists: true, Value: bytes.Clone(v)}
			}
			first := map[string]identityImage{}
			expected := []identityWriterEntry{}
			var h *identityWriterHistory
			var snap identityWriterSnapshot
			events := 0
			st.SetObserver(func(Event) { events++ })
			err := runIdentityAdmission(st, func(tx *Store) error {
				var err error
				h, err = newIdentityWriterHistory(tx)
				if err != nil {
					return err
				}
				rng := rand.New(rand.NewSource(seed))
				for i := 0; i < 300; i++ {
					actor := []string{"atlas", "borealis", "cygnus"}[rng.Intn(3)]
					key := fmt.Sprintf("al:key-%d", rng.Intn(7))
					raw := []byte(actor)
					if i%5 == 0 {
						key = "en:" + actor
						raw = independentWriterEntity(t, actor, i)
					}
					if i == 0 {
						key = "en:atlas"
						actor = "atlas"
						raw = independentWriterEntity(t, actor, i)
					}
					if i == 1 {
						key = "al:empty"
					}
					old := model[key]
					after := identityImage{Exists: true, Value: bytes.Clone(raw)}
					if old.Exists && i%4 == 0 && i > 1 {
						if bytes.HasPrefix([]byte(key), []byte("al:")) {
							actor = string(old.Value)
						}
						after = identityImage{}
					}
					if i%11 == 0 && old.Exists && bytes.HasPrefix([]byte(key), []byte("al:")) && len(old.Value) > 0 {
						actor = string(old.Value)
						after = cloneWriterImage(old)
					}
					if _, ok := first[key]; !ok {
						first[key] = cloneWriterImage(old)
					}
					kb, vb := []byte(key), bytes.Clone(after.Value)
					if after.Exists {
						err = h.put(actor, kb, vb)
					} else {
						err = h.delete(actor, kb)
					}
					if err != nil {
						return err
					}
					for j := range kb {
						kb[j] = 0
					}
					for j := range vb {
						vb[j] = 0
					}
					expected = append(expected, identityWriterEntry{Sequence: uint64(i), Actor: actor, Key: []byte(key), Before: cloneWriterImage(old), After: cloneWriterImage(after)})
					model[key] = cloneWriterImage(after)
				}
				return nil
			}, func(*Store) error { var err error; snap, err = h.freeze(); return err })
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(snap.Entries, expected) || !reflect.DeepEqual(snap.Before, first) {
				t.Fatal("generated execution history diverged")
			}
			final := map[string][]byte{}
			for k, v := range model {
				if v.Exists {
					final[k] = v.Value
				}
			}
			if !reflect.DeepEqual(final, gradeGenerationSnapshot(t, st)) || events != 0 {
				t.Fatal("raw commit/event discrepancy")
			}
			for k := range first {
				if !writerImageEqual(snap.Final[k], model[k]) {
					t.Fatal("final mismatch")
				}
			}
			// The unchanged empty before-value must still have presence=true.
			if !snap.Before["al:empty"].Exists || len(snap.Before["al:empty"].Value) != 0 {
				t.Fatal("empty presence lost")
			}
		})
	}
}

func TestIndependentWriterStrictNewValues(t *testing.T) {
	valid := independentWriterEntity(t, "atlas", 0)
	var e Entity
	if err := json.Unmarshal(valid, &e); err != nil {
		t.Fatal(err)
	}
	marshal := func(e Entity) []byte {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	wrongSlug := e
	wrongSlug.Slug = "borealis"
	emptyName := e
	emptyName.Name = ""
	zeroTime := e
	zeroTime.CreatedAt = time.Time{}
	bigTime := e
	bigTime.CreatedAt = time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := map[string][]byte{
		"body-slug": marshal(wrongSlug), "empty-name": marshal(emptyName), "zero-created": marshal(zeroTime), "unrepresentable-time": marshal(bigTime),
		"whitespace": append([]byte(" "), valid...), "trailing": append(bytes.Clone(valid), []byte("{}")...), "duplicate": append(append([]byte{}, valid[:len(valid)-1]...), []byte(`,"slug":"atlas"}`)...),
		"unknown": append(append([]byte{}, valid[:len(valid)-1]...), []byte(`,"private":"secret"}`)...), "invalid-utf8": []byte{'"', 255, '"'}, "object": []byte("{}"), "array": []byte("[]"),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			st := openTemp(t)
			gradeGenerationSet(t, st, "en:atlas", []byte{255, 0})
			before := gradeGenerationSnapshot(t, st)
			err := runIdentityAdmission(st, func(tx *Store) error {
				h, err := newIdentityWriterHistory(tx)
				if err != nil {
					return err
				}
				if err = h.put("atlas", []byte("al:staged"), []byte("atlas")); err != nil {
					return err
				}
				if err = h.put("atlas", []byte("en:atlas"), raw); err != errIdentityWriterHistory {
					t.Fatalf("nonstatic refusal: %v", err)
				}
				if len(h.entries) != 1 {
					t.Fatal("failed mutation recorded")
				}
				return nil
			}, func(*Store) error { t.Fatal("poison reached finalizer"); return nil })
			if err != errIdentityWriterHistory || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("malformed swallowed write committed")
			}
		})
	}
}

func TestIndependentWriterRefusedFamiliesAndDeletes(t *testing.T) {
	for _, key := range []string{"att:atlas:x", "fa:atlas:x", "ep:x", "ig:atlas", "en:", "en:atlas:extra", "al: x", "al:x_foo", "al:\xff"} {
		t.Run(key, func(t *testing.T) {
			st := openTemp(t)
			gradeGenerationSet(t, st, key, []byte("atlas"))
			before := gradeGenerationSnapshot(t, st)
			err := runIdentityAdmission(st, func(tx *Store) error {
				h, err := newIdentityWriterHistory(tx)
				if err != nil {
					return err
				}
				if err = h.delete("atlas", []byte(key)); err != errIdentityWriterHistory {
					t.Fatal("delete accepted")
				}
				return nil
			}, func(*Store) error { t.Fatal("finalized"); return nil })
			if err != errIdentityWriterHistory || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("refusal changed bytes")
			}
		})
	}
	for _, old := range [][]byte{nil, []byte("atlas\x00"), []byte("atlas "), []byte("borealis")} {
		t.Run(fmt.Sprintf("owner-%x", old), func(t *testing.T) {
			st := openTemp(t)
			gradeGenerationSet(t, st, "al:shared", old)
			before := gradeGenerationSnapshot(t, st)
			err := runIdentityAdmission(st, func(tx *Store) error {
				h, err := newIdentityWriterHistory(tx)
				if err != nil {
					return err
				}
				_ = h.delete("atlas", []byte("al:shared"))
				return nil
			}, func(*Store) error { t.Fatal("finalized"); return nil })
			if err != errIdentityWriterHistory || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("partial owner compare")
			}
		})
	}
}

func TestIndependentWriterForeignCapability(t *testing.T) {
	a, b := openTemp(t), openTemp(t)
	before := gradeGenerationSnapshot(t, b)
	err := runIdentityAdmission(a, func(ta *Store) error {
		h, err := newIdentityWriterHistory(ta)
		if err != nil {
			return err
		}
		err = runIdentityAdmission(b, func(tb *Store) error {
			own, err := newIdentityWriterHistory(tb)
			if err != nil {
				return err
			}
			if err = own.put("atlas", []byte("al:staged"), []byte("atlas")); err != nil {
				return err
			}
			foreign := *h
			foreign.st = tb
			if err = foreign.put("atlas", []byte("al:foreign"), []byte("atlas")); err != errIdentityWriterHistory {
				t.Fatal("foreign owner accepted")
			}
			return nil
		}, func(*Store) error { t.Fatal("foreign refusal finalized"); return nil })
		if err != errIdentityWriterHistory || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, b)) {
			t.Fatal("foreign scope committed")
		}
		return nil
	}, func(*Store) error { return nil })
	if err != nil {
		t.Fatal("legitimate independent owner poisoned", err)
	}
}

func TestIndependentWriterStalePresenceAndNoop(t *testing.T) {
	for _, when := range []string{"next", "freeze"} {
		for _, change := range []string{"delete", "create"} {
			t.Run(when+change, func(t *testing.T) {
				st := openTemp(t)
				before := gradeGenerationSnapshot(t, st)
				var h *identityWriterHistory
				err := runIdentityAdmission(st, func(tx *Store) error {
					var err error
					h, err = newIdentityWriterHistory(tx)
					if err != nil {
						return err
					}
					if err = h.put("atlas", []byte("al:shared"), []byte("atlas")); err != nil {
						return err
					}
					if change == "create" {
						if err = h.delete("atlas", []byte("al:shared")); err != nil {
							return err
						}
						err = tx.txn.Set([]byte("al:shared"), []byte{})
					} else {
						err = tx.txn.Delete([]byte("al:shared"))
					}
					if err != nil {
						return err
					}
					if when == "next" {
						_ = h.put("atlas", []byte("al:shared"), []byte("atlas"))
					}
					return nil
				}, func(*Store) error {
					snap, err := h.freeze()
					if err == nil || !reflect.DeepEqual(snap, identityWriterSnapshot{}) {
						t.Fatal("stale presence accepted/partial result")
					}
					return nil
				})
				if err != errIdentityWriterHistory || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("stale key committed")
				}
			})
		}
	}
}

func TestIndependentWriterRealCommitConflict(t *testing.T) {
	st := openTemp(t)
	gradeGenerationSet(t, st, "al:shared", []byte("old-owner"))
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	var h *identityWriterHistory
	err := runIdentityAdmission(st, func(tx *Store) error {
		var err error
		h, err = newIdentityWriterHistory(tx)
		if err != nil {
			return err
		}
		if err = h.put("atlas", []byte("al:shared"), []byte("atlas")); err != nil {
			return err
		}
		if err = h.put("atlas", []byte("al:staged"), []byte("atlas")); err != nil {
			return err
		}
		// Independent synthetic competitor is used solely to force commit conflict.
		return st.db.Update(func(tx *badger.Txn) error { return tx.Set([]byte("al:shared"), []byte("competitor")) })
	}, func(*Store) error { _, err := h.freeze(); return err })
	before["al:shared"] = []byte("competitor")
	if !errors.Is(err, badger.ErrConflict) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) || events != 0 {
		t.Fatal("conflicted writer leaked commit", err)
	}
	if err = h.put("atlas", []byte("al:later"), []byte("atlas")); err != errIdentityWriterHistory {
		t.Fatal("conflicted closed scope accepted")
	}
}
