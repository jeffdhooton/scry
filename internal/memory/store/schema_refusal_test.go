package store

import (
	"errors"
	"github.com/dgraph-io/badger/v4"
	"path/filepath"
	"reflect"
	"testing"
)

func schemaRawRows(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows := map[string][]byte{}
	if err := db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			v, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			rows[string(it.Item().KeyCopy(nil))] = v
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestSchemaRefusalPreservesEveryRawRecord(t *testing.T) {
	for _, marker := range []string{"MISSING", "0", "null", "-1", "2", "999", `"future-format"`, `{}`, `[]`, ``, `not-json`, `1.0`} {
		t.Run(marker, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "db")
			db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Update(func(tx *badger.Txn) error {
				if marker != "MISSING" {
					if err := tx.Set([]byte(keySchemaVersion), []byte(marker)); err != nil {
						return err
					}
				}
				for _, prefix := range []string{"ep:", "en:", "fa:", "adj:", "al:", "pq:", "ar:", "rs:", "rt:", "unknown:"} {
					if err := tx.Set([]byte(prefix+"synthetic"), []byte("opaque-original")); err != nil {
						return err
					}
				}
				return tx.Set([]byte("unknown:empty"), nil)
			}); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			before := schemaRawRows(t, dir)
			for i := 0; i < 2; i++ {
				s, err := Open(dir)
				if s != nil {
					_ = s.Close()
					t.Fatal("opened incompatible data")
				}
				if !errors.Is(err, ErrSchemaMismatch) {
					t.Fatalf("wanted refusal, got %v", err)
				}
				if !reflect.DeepEqual(before, schemaRawRows(t, dir)) {
					t.Fatal("refusal changed raw data")
				}
			}
		})
	}
}

func TestSchemaSupportedOpenIsBytePreserving(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "db")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	before := schemaRawRows(t, dir)
	if string(before[keySchemaVersion]) != "1" || len(before) != 1 {
		t.Fatal("empty initialization has wrong keys")
	}
	for i := 0; i < 2; i++ {
		s, err = Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, schemaRawRows(t, dir)) {
			t.Fatal("supported open changed data")
		}
	}
}
