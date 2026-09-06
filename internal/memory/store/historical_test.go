package store

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestHistoricalRestatementRefusesUnsupportedRaw(t *testing.T) {
	for _, mode := range []string{"unknown", "whitespace", "duplicate", "wrong-key", "exact"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			at := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
			end := at.Add(time.Hour)
			old := Fact{Src: "lornwick", Relation: "status", Value: "in-progress", Fact: "Lornwick remains in progress.", ValidFrom: at, InvalidAt: &end, Confidence: .8, Episodes: []string{"original"}}
			raw, err := json.Marshal(old)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "unknown":
				raw = append(raw[:len(raw)-1], []byte(",\"future\":{\"evidence\":42}}")...)
			case "whitespace":
				raw = append([]byte(" \n"), raw...)
			case "duplicate":
				raw = append(raw[:len(raw)-1], []byte(",\"confidence\":0.8}")...)
			case "wrong-key":
				changed := old
				changed.ValidFrom = at.Add(time.Second)
				raw, _ = json.Marshal(changed)
			}
			key := factKey(old.Src, old.Relation, old.KeyDst(), old.ValidFrom)
			if err := st.db.Update(func(tx *badger.Txn) error { return tx.Set(key, raw) }); err != nil {
				t.Fatal(err)
			}
			before := collisionRawState(t, st)
			got, err := st.HistoricalRestatement(old)
			if mode == "exact" {
				if err != nil || got == nil || !reflect.DeepEqual(*got, old) {
					t.Fatalf("exact %v %+v", err, got)
				}
			} else if !errors.Is(err, ErrFactConflict) || got != nil {
				t.Fatalf("unsupported accepted %v", err)
			}
			if !reflect.DeepEqual(before, collisionRawState(t, st)) {
				t.Fatal("raw rows changed")
			}
		})
	}
}
