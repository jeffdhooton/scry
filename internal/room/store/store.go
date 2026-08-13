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
	"sort"
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

// validTransitions covers UpdateTaskStatus. Claiming (open|abandoned ->
// claimed) happens ONLY through ClaimTask, which requires an agent identity —
// so the update path can never create an ownerless claim.
var validTransitions = map[TaskStatus][]TaskStatus{
	TaskClaimed:    {TaskInProgress, TaskAbandoned},
	TaskInProgress: {TaskReview, TaskAbandoned},
	TaskReview:     {TaskDone, TaskInProgress, TaskAbandoned},
}

func transitionAllowed(from, to TaskStatus) bool {
	for _, t := range validTransitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

func (s *Store) openRoom(roomID string) (*Room, error) {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return nil, err
	}
	if room.Status == RoomClosed {
		return nil, fmt.Errorf("room %q is closed", roomID)
	}
	return room, nil
}

func (s *Store) PostTask(roomID string, t *Task) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.openRoom(roomID); err != nil {
		return nil, err
	}
	if t.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}
	if t.ID == "" {
		t.ID = newID()
	}
	now := time.Now().UTC()
	t.RoomID = roomID
	t.Status = TaskOpen
	t.ClaimedBy = ""
	t.CreatedAt = now
	t.UpdatedAt = now
	if err := s.putJSON(taskPrefix+roomID+":"+t.ID, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) getTask(roomID, taskID string) (*Task, error) {
	var task Task
	if err := s.getJSON(taskPrefix+roomID+":"+taskID, &task); err != nil {
		return nil, fmt.Errorf("task %q not found in room %q", taskID, roomID)
	}
	return &task, nil
}

func (s *Store) ClaimTask(roomID, taskID, agent string) (*Task, error) {
	if agent == "" {
		return nil, fmt.Errorf("agent is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.openRoom(roomID); err != nil {
		return nil, err
	}
	task, err := s.getTask(roomID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskOpen && task.Status != TaskAbandoned {
		return nil, fmt.Errorf("task %q already claimed by %q (status %s)", taskID, task.ClaimedBy, task.Status)
	}
	task.Status = TaskClaimed
	task.ClaimedBy = agent
	task.UpdatedAt = time.Now().UTC()
	if err := s.putJSON(taskPrefix+roomID+":"+task.ID, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Store) UpdateTaskStatus(roomID, taskID string, next TaskStatus) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.openRoom(roomID); err != nil {
		return nil, err
	}
	task, err := s.getTask(roomID, taskID)
	if err != nil {
		return nil, err
	}
	if !transitionAllowed(task.Status, next) {
		return nil, fmt.Errorf("invalid transition %s -> %s for task %q", task.Status, next, taskID)
	}
	task.Status = next
	if next == TaskAbandoned {
		task.ClaimedBy = ""
	}
	task.UpdatedAt = time.Now().UTC()
	if err := s.putJSON(taskPrefix+roomID+":"+task.ID, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Store) ListTasks(roomID string) ([]Task, error) {
	if _, err := s.GetRoom(roomID); err != nil {
		return nil, err
	}
	prefix := []byte(taskPrefix + roomID + ":")
	var tasks []Task
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			var task Task
			if err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &task)
			}); err != nil {
				continue
			}
			tasks = append(tasks, task)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, nil
}

func validKind(k MessageKind) bool {
	switch k {
	case KindStatus, KindHandoff, KindContract, KindReview:
		return true
	}
	return false
}

func seqKey(roomID string) string { return seqPrefix + roomID }
func msgKey(roomID string, seq uint64) string {
	return fmt.Sprintf("%s%s:%020d", msgPrefix, roomID, seq)
}

func (s *Store) PostMessage(roomID string, m *Message) (*Message, error) {
	if m.From == "" {
		return nil, fmt.Errorf("message from is required")
	}
	if m.Body == "" {
		return nil, fmt.Errorf("message body is required")
	}
	if !validKind(m.Kind) {
		return nil, fmt.Errorf("invalid message kind %q (want status|handoff|contract|review)", m.Kind)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.openRoom(roomID); err != nil {
		return nil, err
	}

	err := s.db.Update(func(txn *badger.Txn) error {
		var seq uint64
		item, err := txn.Get([]byte(seqKey(roomID)))
		if err == nil {
			if verr := item.Value(func(val []byte) error {
				_, serr := fmt.Sscanf(string(val), "%d", &seq)
				return serr
			}); verr != nil {
				return verr
			}
		} else if err != badger.ErrKeyNotFound {
			return err
		}
		seq++

		m.Seq = seq
		m.RoomID = roomID
		m.CreatedAt = time.Now().UTC()
		data, err := json.Marshal(m)
		if err != nil {
			return err
		}
		if err := txn.Set([]byte(msgKey(roomID, seq)), data); err != nil {
			return err
		}
		return txn.Set([]byte(seqKey(roomID)), []byte(fmt.Sprintf("%020d", seq)))
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) ReadSince(roomID string, cursor uint64, limit int) ([]Message, uint64, error) {
	if limit <= 0 {
		limit = 50
	}
	if _, err := s.GetRoom(roomID); err != nil {
		return nil, cursor, err
	}
	prefix := []byte(msgPrefix + roomID + ":")
	start := []byte(msgKey(roomID, cursor+1))
	msgs := []Message{}
	next := cursor

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(start); it.ValidForPrefix(prefix); it.Next() {
			if len(msgs) >= limit {
				break
			}
			var m Message
			if err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &m)
			}); err != nil {
				continue
			}
			msgs = append(msgs, m)
			next = m.Seq
		}
		return nil
	})
	if err != nil {
		return nil, cursor, err
	}
	return msgs, next, nil
}
