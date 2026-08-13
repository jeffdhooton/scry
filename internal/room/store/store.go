// Package store provides BadgerDB-backed storage for fleet rooms: a task
// board and an append-only, cursor-addressed message channel shared by
// multiple coding agents. All writes are serialized through one mutex —
// claim atomicity and message-sequence allocation depend on it, and write
// volume is far too small for contention to matter.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Key layout:
//
//	room:<roomID>             -> Room
//	task:<roomID>:<taskID>    -> Task
//	msg:<roomID>:<seq %020d>  -> Message
//	seq:<roomID>              -> latest allocated message seq, as %020d
const (
	roomPrefix = "room:"
	taskPrefix = "task:"
	msgPrefix  = "msg:"
	seqPrefix  = "seq:"
)

type Store struct {
	db     *badger.DB
	mu     sync.Mutex // serializes all writes
	stopGC chan struct{}
}

func Open(dir string) (*Store, error) {
	opts := badger.DefaultOptions(dir).
		WithLogger(nil).
		WithCompression(0)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger at %q: %w", dir, err)
	}
	s := &Store{db: db, stopGC: make(chan struct{})}
	go s.gcLoop()
	return s, nil
}

func (s *Store) Close() error {
	close(s.stopGC)
	return s.db.Close()
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

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) CreateRoom(runID, repo string) (*Room, error) {
	room := &Room{
		ID:        newID(),
		RunID:     runID,
		Repo:      repo,
		Status:    RoomOpen,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.putJSON(roomPrefix+room.ID, room); err != nil {
		return nil, err
	}
	return room, nil
}

func (s *Store) GetRoom(id string) (*Room, error) {
	var room Room
	if err := s.getJSON(roomPrefix+id, &room); err != nil {
		return nil, fmt.Errorf("room %q not found", id)
	}
	return &room, nil
}

func (s *Store) CloseRoom(id string) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room, err := s.GetRoom(id)
	if err != nil {
		return nil, err
	}
	if room.Status == RoomClosed {
		return room, nil
	}
	now := time.Now().UTC()
	room.Status = RoomClosed
	room.ClosedAt = &now
	if err := s.putJSON(roomPrefix+room.ID, room); err != nil {
		return nil, err
	}
	return room, nil
}

// --- badger helpers ---

func (s *Store) putJSON(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (s *Store) getJSON(key string, v any) error {
	return s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, v)
		})
	})
}
