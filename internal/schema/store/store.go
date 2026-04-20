// Package store wraps the BadgerDB-backed schema cache for one project.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

const (
	SchemaVersion = 1

	prefixMeta  = "meta:"
	prefixTable = "table:"
	prefixFKRef = "fk_ref:"
	prefixEnum  = "enum:"
	prefixName  = "name:"
)

type Store struct {
	db *badger.DB
}

func Open(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir).
		WithLogger(nil).
		WithCompression(0)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger at %q: %w", dir, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Reset() error {
	return s.db.DropAll()
}

func (s *Store) SchemaVersionOnDisk() (int, error) {
	v, err := s.GetMeta("schema_version")
	if errors.Is(err, badger.ErrKeyNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, fmt.Errorf("decode schema_version: %w", err)
	}
	return n, nil
}

func (s *Store) GetMeta(key string) ([]byte, error) {
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixMeta + key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			out = append([]byte{}, val...)
			return nil
		})
	})
	return out, err
}

func (s *Store) SetMeta(key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(prefixMeta+key), b)
	})
}

func (s *Store) GetTable(name string) ([]byte, error) {
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixTable + name))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			out = append([]byte{}, val...)
			return nil
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("table %q not found", name)
	}
	return out, err
}

func (s *Store) ListTables() ([]string, error) {
	prefix := []byte(prefixTable)
	var names []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			k := it.Item().KeyCopy(nil)
			names = append(names, string(k[len(prefix):]))
		}
		return nil
	})
	return names, err
}

func (s *Store) GetForeignKeysTo(table string) ([][]byte, error) {
	prefix := []byte(prefixFKRef + table + ":")
	var results [][]byte
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 32
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				results = append(results, append([]byte{}, val...))
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return results, err
}

func (s *Store) GetEnums(table, column string) ([][]byte, error) {
	var prefix string
	if table == "" {
		prefix = prefixEnum
	} else if column == "" {
		prefix = prefixEnum + table + ":"
	} else {
		key := prefixEnum + table + ":" + column
		var out []byte
		err := s.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(key))
			if err != nil {
				return err
			}
			return item.Value(func(val []byte) error {
				out = append([]byte{}, val...)
				return nil
			})
		})
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return [][]byte{out}, nil
	}

	var results [][]byte
	pb := []byte(prefix)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 32
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				results = append(results, append([]byte{}, val...))
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return results, err
}

func (s *Store) SearchByName(query string) ([]string, error) {
	lower := strings.ToLower(query)
	prefix := []byte(prefixName)
	var matches []string
	seen := map[string]bool{}
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			k := string(it.Item().KeyCopy(nil))
			rest := k[len(prefixName):]
			if strings.Contains(rest, lower) {
				parts := strings.SplitN(rest, ":", 2)
				if len(parts) == 2 && !seen[rest] {
					seen[rest] = true
					matches = append(matches, rest)
				}
			}
		}
		return nil
	})
	return matches, err
}

type Writer struct {
	wb *badger.WriteBatch
}

func (s *Store) NewWriter() *Writer {
	return &Writer{wb: s.db.NewWriteBatch()}
}

func (w *Writer) PutTable(name string, data []byte) error {
	return w.wb.Set([]byte(prefixTable+name), data)
}

func (w *Writer) PutFKRef(referencedTable, srcTable, srcColumn string, data []byte) error {
	key := prefixFKRef + referencedTable + ":" + srcTable + ":" + srcColumn
	return w.wb.Set([]byte(key), data)
}

func (w *Writer) PutEnum(table, column string, data []byte) error {
	key := prefixEnum + table + ":" + column
	return w.wb.Set([]byte(key), data)
}

func (w *Writer) PutName(token, table string) error {
	key := prefixName + strings.ToLower(token) + ":" + table
	return w.wb.Set([]byte(key), nil)
}

func (w *Writer) Flush() error {
	return w.wb.Flush()
}
