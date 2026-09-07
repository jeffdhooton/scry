package friction

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

// Store is an append-only journal. It is not a rebuildable index: opening an
// unsupported schema fails without wiping anything. All writes are serialized.
type Store struct {
	db *badger.DB
	mu sync.Mutex
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0).WithSyncWrites(true))
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte("schema"))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return tx.Set([]byte("schema"), []byte("1"))
		}
		if err != nil {
			return err
		}
		return item.Value(func(b []byte) error {
			if string(b) != "1" {
				return fmt.Errorf("unsupported friction schema %q; journal left intact", b)
			}
			return nil
		})
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

type Receipt struct {
	EventID string `json:"event_id"`
	SHA256  string `json:"sha256"`
	Created bool   `json:"created"`
}

func eventKey(id string) []byte { return []byte("event:" + id) }
func repoPrefix(repo string) []byte {
	h := sha256.Sum256([]byte(repo))
	return []byte(fmt.Sprintf("repo:%x:", h))
}

// Record acknowledges only after the synchronous transaction commits. A retry
// with the same typed JSON is a no-op; an ID with changed content is an error.
func (s *Store) Record(e Event) (*Receipt, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	r := &Receipt{EventID: e.EventID, SHA256: hex.EncodeToString(h[:])}
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.db.Update(func(tx *badger.Txn) error {
		item, err := tx.Get(eventKey(e.EventID))
		if err == nil {
			return item.Value(func(old []byte) error {
				if !bytes.Equal(old, b) {
					return ErrConflict
				}
				return nil
			})
		}
		if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		if err := tx.Set(eventKey(e.EventID), b); err != nil {
			return err
		}
		r.Created = true
		return tx.Set(append(repoPrefix(e.Repository), []byte(e.EventID)...), nil)
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func readEvent(tx *badger.Txn, id string) (Event, error) {
	var e Event
	item, err := tx.Get(eventKey(id))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return e, ErrNotFound
	}
	if err != nil {
		return e, err
	}
	err = item.Value(func(b []byte) error { return json.Unmarshal(b, &e) })
	return e, err
}

func (s *Store) Get(id string) (*Event, error) {
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	var e Event
	err := s.db.View(func(tx *badger.Txn) error { var err error; e, err = readEvent(tx, id); return err })
	if err != nil {
		return nil, err
	}
	return &e, nil
}

type Page struct {
	Events    []Event `json:"events"`
	NextAfter string  `json:"next_after,omitempty"`
}

func (s *Store) List(ctx context.Context, f Filter) (*Page, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	limit := f.Limit
	if limit == 0 {
		limit = MaxResults
	}
	p := &Page{Events: []Event{}}
	err := s.db.View(func(tx *badger.Txn) error {
		prefix := repoPrefix(f.Repository)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := tx.NewIterator(opts)
		defer it.Close()
		start := append(append([]byte{}, prefix...), []byte(f.After)...)
		for it.Seek(start); it.ValidForPrefix(prefix); it.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}
			id := string(it.Item().Key()[len(prefix):])
			if id <= f.After {
				continue
			}
			e, err := readEvent(tx, id)
			if err != nil {
				return err
			}
			if !f.matches(e) {
				continue
			}
			if len(p.Events) == limit {
				p.NextAfter = p.Events[len(p.Events)-1].EventID
				break
			}
			p.Events = append(p.Events, e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}
