package store

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func collisionRawState(t *testing.T, s *Store) map[string]string {
	t.Helper()
	m := map[string]string{}
	if err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			m[string(item.KeyCopy(nil))] = string(v)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPutFactRejectsDistinctOccupiedAssertion(t *testing.T) {
	for _, historical := range []bool{false, true} {
		for _, mode := range []string{"value", "text", "raw-relation", "atomic"} {
			t.Run(mode+"/"+map[bool]string{false: "current", true: "historical"}[historical], func(t *testing.T) {
				st := openTemp(t)
				at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
				until := at.Add(time.Hour)
				if err := st.PutEntity(Entity{Slug: "velatrix", Name: "Velatrix", Type: "project"}); err != nil {
					t.Fatal(err)
				}
				old := Fact{Src: "velatrix", Relation: "status", Value: "in-progress", Fact: "The geometry reader is next.", ValidFrom: at, Confidence: .95, Episodes: []string{"original"}}
				if historical {
					old.InvalidAt = &until
				}
				if err := st.PutFact(old); err != nil {
					t.Fatal(err)
				}
				incoming := old
				switch mode {
				case "value":
					incoming.Value = "in_progress"
				case "text", "atomic":
					incoming.Fact = "A different review assertion."
				case "raw-relation":
					incoming.RawRelation = "has_capacity"
				}
				before := collisionRawState(t, st)
				events := 0
				st.SetObserver(func(Event) { events++ })
				var err error
				if mode == "atomic" {
					err = st.AtomicWrite(func(tx *Store) error {
						if err := tx.PutEntity(Entity{Slug: "sentinel", Name: "Sentinel", Type: "tool"}); err != nil {
							return err
						}
						return tx.PutFact(incoming)
					})
				} else {
					err = st.PutFact(incoming)
				}
				if !errors.Is(err, ErrFactConflict) {
					t.Fatalf("error=%v", err)
				}
				if strings.Contains(err.Error(), incoming.Value) || strings.Contains(err.Error(), incoming.Fact) {
					t.Fatal("conflict diagnostic exposes assertion content")
				}
				if !reflect.DeepEqual(before, collisionRawState(t, st)) || events != 0 {
					t.Fatalf("refusal wrote data or %d events", events)
				}
			})
		}
	}
}

func TestPutFactAllowsSameAssertionMetadataUpdate(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	if err := st.PutEntity(Entity{Slug: "velatrix", Name: "Velatrix", Type: "project"}); err != nil {
		t.Fatal(err)
	}
	f := Fact{Src: "velatrix", Relation: "status", Value: "in-progress", Fact: "A stable assertion.", ValidFrom: at, Confidence: .8, Episodes: []string{"first"}}
	if err := st.PutFact(f); err != nil {
		t.Fatal(err)
	}
	until := at.Add(time.Hour)
	f.InvalidAt = &until
	f.Confidence = .95
	f.Episodes = append(f.Episodes, "second")
	f.ValidFrom = at.In(time.FixedZone("offset", -4*3600))
	if err := st.PutFact(f); err != nil {
		t.Fatal(err)
	}
	got, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("facts=%+v", got)
	}
	a, _ := json.Marshal(f)
	b, _ := json.Marshal(got[0])
	if string(a) != string(b) {
		t.Fatalf("metadata update mismatch: %s %s", a, b)
	}
}
