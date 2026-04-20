// Package store provides BadgerDB-backed storage for captured HTTP requests
// with TTL-based expiration.
package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

const keyPrefix = "req:"

type Options struct {
	Dir        string
	TTL        time.Duration
	MaxEntries int
}

func DefaultOptions(dir string) Options {
	return Options{
		Dir:        dir,
		TTL:        30 * time.Minute,
		MaxEntries: 1000,
	}
}

type Store struct {
	db     *badger.DB
	ttl    time.Duration
	stopGC chan struct{}
}

func Open(opts Options) (*Store, error) {
	dbOpts := badger.DefaultOptions(opts.Dir).
		WithLogger(nil).
		WithCompression(0)
	db, err := badger.Open(dbOpts)
	if err != nil {
		return nil, fmt.Errorf("open badger at %q: %w", opts.Dir, err)
	}
	s := &Store{
		db:     db,
		ttl:    opts.TTL,
		stopGC: make(chan struct{}),
	}
	go s.gcLoop()
	return s, nil
}

func (s *Store) Close() error {
	close(s.stopGC)
	return s.db.Close()
}

func (s *Store) Put(req *Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%020d:%s", keyPrefix, req.StartedAt.UnixNano(), req.ID)
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), data).WithTTL(s.ttl)
		return txn.SetEntry(e)
	})
}

func (s *Store) List(f ListFilter) ([]RequestSummary, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	prefix := []byte(keyPrefix)
	var results []RequestSummary

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()

		// Seek to end of "req:" range for reverse iteration
		it.Seek(append(prefix, 0xFF))
		for ; it.ValidForPrefix(prefix); it.Next() {
			if len(results) >= f.Limit {
				break
			}
			var req Request
			err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &req)
			})
			if err != nil {
				continue
			}
			if matchesFilter(&req, &f) {
				results = append(results, req.Summary())
			}
		}
		return nil
	})
	return results, err
}

func (s *Store) Get(id string) (*Request, error) {
	suffix := ":" + id
	prefix := []byte(keyPrefix)
	var result *Request

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()

		it.Seek(append(prefix, 0xFF))
		for ; it.ValidForPrefix(prefix); it.Next() {
			k := string(it.Item().KeyCopy(nil))
			if strings.HasSuffix(k, suffix) {
				return it.Item().Value(func(val []byte) error {
					var req Request
					if err := json.Unmarshal(val, &req); err != nil {
						return err
					}
					result = &req
					return nil
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("request %q not found", id)
	}
	return result, nil
}

func (s *Store) Count() int {
	var n int
	prefix := []byte(keyPrefix)
	_ = s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			n++
		}
		return nil
	})
	return n
}

func (s *Store) gcLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopGC:
			return
		case <-ticker.C:
			_ = s.db.RunValueLogGC(0.5)
		}
	}
}

func matchesFilter(req *Request, f *ListFilter) bool {
	if f.Method != "" && !strings.EqualFold(req.Method, f.Method) {
		return false
	}
	if f.Path != "" && !strings.Contains(req.Path, f.Path) {
		return false
	}
	if f.StatusMin > 0 && req.StatusCode < f.StatusMin {
		return false
	}
	if f.StatusMax > 0 && req.StatusCode > f.StatusMax {
		return false
	}
	return true
}
