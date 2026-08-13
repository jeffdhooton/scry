# Room Domain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `room` domain to scry — a persistent task board plus an append-only, cursor-addressed message channel — so multiple coding agents (Claude, Codex, Kimi) can coordinate shared work through the scry daemon over MCP.

**Architecture:** Follows scry's established three-layer domain pattern exactly as the HTTP domain does: `internal/room/store/` (BadgerDB storage), `internal/daemon/room_methods.go` (RPC handlers registered on the daemon), `internal/mcp/room_tools.go` (MCP tool definitions + passthrough dispatch). The store serializes all writes through a single mutex (claim atomicity and seq allocation come free; write volume is tiny). Rooms persist after close — they become episode sources for the phase-5 memory graph later.

**Tech Stack:** Go 1.26, BadgerDB v4 (existing dependency), stdlib only otherwise. Parent spec: `~/dotfiles/docs/superpowers/specs/2026-08-12-multi-agent-fleet-design.md` (Component 1).

## Global Constraints

- Work on branch `room-domain` (created from `main` before Task 1; the controller does this). Never commit to `main`.
- No new Go dependencies. `github.com/dgraph-io/badger/v4` is already in go.mod.
- MCP tool names are locked by the fleet spec: `scry_room_create`, `scry_room_close`, `scry_task_post`, `scry_task_claim`, `scry_task_update`, `scry_task_list`, `scry_post`, `scry_read`. RPC method names: `room.create`, `room.close`, `room.task_post`, `room.task_claim`, `room.task_update`, `room.task_list`, `room.post`, `room.read`.
- Task status transitions: `open → claimed → in_progress → review → done`, with `abandoned` reachable from claimed/in_progress/review and re-claimable (abandoned → claimed via claim, not via update).
- Posting tasks or messages to a closed room is an error. Closed rooms and their history are never deleted.
- The global commit-msg hook requires a Capitalized subject line of at most 50 characters. The commit messages below already comply — do not lengthen them. Every commit body ends with: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`
- Full verify for every task: `go test ./internal/room/... ./internal/daemon/... ./internal/mcp/...` and `go build ./...`.
- Do not touch `.claude/worktrees/` (unrelated in-flight work) or the status structs in `internal/daemon/methods.go` (room status reporting is deliberately out of v1 scope).

---

### Task 1: Room store — models and room lifecycle

**Files:**
- Create: `internal/room/store/models.go`
- Create: `internal/room/store/store.go`
- Test: `internal/room/store/store_test.go`

**Interfaces:**
- Produces (used by every later task):
  - `store.Open(dir string) (*Store, error)`, `(*Store).Close() error`
  - `(*Store).CreateRoom(runID, repo string) (*Room, error)` — generates ID, status `open`
  - `(*Store).GetRoom(id string) (*Room, error)` — error contains "not found" for unknown ID
  - `(*Store).CloseRoom(id string) (*Room, error)` — idempotent-safe: closing a closed room returns it unchanged
  - Types `Room`, `RoomStatus` (`RoomOpen`/`RoomClosed`), plus `Task`/`TaskStatus`/`Message`/`MessageKind` model declarations (methods for tasks/messages come in Tasks 2-3)

- [ ] **Step 1: Write the failing test**

```go
package store

import (
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCreateGetCloseRoom(t *testing.T) {
	s := newTestStore(t)

	room, err := s.CreateRoom("run-42", "/Users/jeff/workspace/demo")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if room.ID == "" || room.Status != RoomOpen || room.RunID != "run-42" {
		t.Fatalf("bad room: %+v", room)
	}

	got, err := s.GetRoom(room.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Repo != "/Users/jeff/workspace/demo" {
		t.Fatalf("repo not persisted: %+v", got)
	}

	closed, err := s.CloseRoom(room.ID)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if closed.Status != RoomClosed || closed.ClosedAt == nil {
		t.Fatalf("not closed: %+v", closed)
	}

	// Closing again is not an error and stays closed.
	again, err := s.CloseRoom(room.ID)
	if err != nil {
		t.Fatalf("re-close: %v", err)
	}
	if again.Status != RoomClosed {
		t.Fatalf("re-close changed status: %+v", again)
	}
}

func TestGetRoomNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetRoom("nope")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestRoomsSurviveReopen(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	room, err := s.CreateRoom("run-1", "/repo")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	s2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	got, err := s2.GetRoom(room.ID)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if got.RunID != "run-1" {
		t.Fatalf("lost data: %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd ~/workspace/context-stack/scry && go test ./internal/room/...`
Expected: FAIL to build (package does not exist yet / undefined: Open).

- [ ] **Step 3: Write models.go**

```go
package store

import "time"

type RoomStatus string

const (
	RoomOpen   RoomStatus = "open"
	RoomClosed RoomStatus = "closed"
)

// Room is a shared coordination space for one fleet run: a task board plus
// an append-only message channel. Rooms persist after close so their history
// can feed the memory graph.
type Room struct {
	ID        string     `json:"id"`
	RunID     string     `json:"run_id"`
	Repo      string     `json:"repo"`
	Status    RoomStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
}

type TaskStatus string

const (
	TaskOpen       TaskStatus = "open"
	TaskClaimed    TaskStatus = "claimed"
	TaskInProgress TaskStatus = "in_progress"
	TaskReview     TaskStatus = "review"
	TaskDone       TaskStatus = "done"
	TaskAbandoned  TaskStatus = "abandoned"
)

// Task is one unit of work on the room's board. DependsOn holds task IDs
// that must reach done first; Interfaces carries the boundary/contract stubs
// from the fleet plan.
type Task struct {
	ID         string     `json:"id"`
	RoomID     string     `json:"room_id"`
	Title      string     `json:"title"`
	Body       string     `json:"body,omitempty"`
	DependsOn  []string   `json:"depends_on,omitempty"`
	Interfaces string     `json:"interfaces,omitempty"`
	Status     TaskStatus `json:"status"`
	ClaimedBy  string     `json:"claimed_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type MessageKind string

const (
	KindStatus   MessageKind = "status"
	KindHandoff  MessageKind = "handoff"
	KindContract MessageKind = "contract"
	KindReview   MessageKind = "review"
)

// Message is one entry in a room's append-only channel. Seq is a per-room,
// monotonically increasing cursor; readers poll with ReadSince(cursor).
type Message struct {
	Seq       uint64      `json:"seq"`
	RoomID    string      `json:"room_id"`
	TaskID    string      `json:"task_id,omitempty"`
	From      string      `json:"from"`
	Kind      MessageKind `json:"kind"`
	Body      string      `json:"body"`
	CreatedAt time.Time   `json:"created_at"`
}
```

- [ ] **Step 4: Write store.go (room lifecycle only; task/message methods land in Tasks 2-3)**

```go
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
//   room:<roomID>             -> Room
//   task:<roomID>:<taskID>    -> Task
//   msg:<roomID>:<seq %020d>  -> Message
//   seq:<roomID>              -> latest allocated message seq, as %020d
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
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/room/... -v`
Expected: 3 tests PASS.

- [ ] **Step 6: Full verify and commit**

Run: `go build ./... && go test ./internal/room/... ./internal/daemon/... ./internal/mcp/...`

```bash
git add internal/room/store/
git commit -m "Add room store models and room lifecycle

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Room store — task board with atomic claims

**Files:**
- Modify: `internal/room/store/store.go` (append methods)
- Test: `internal/room/store/store_test.go` (append tests)

**Interfaces:**
- Consumes: Task 1's `Store`, `Task`, `TaskStatus`, `newID`, `putJSON`/`getJSON`.
- Produces:
  - `(*Store).PostTask(roomID string, t *Task) (*Task, error)` — assigns ID (if empty), sets RoomID/Status=`open`/timestamps; error if room missing or closed
  - `(*Store).ClaimTask(roomID, taskID, agent string) (*Task, error)` — atomic; only from `open` or `abandoned`; error message contains "already claimed" on conflict
  - `(*Store).UpdateTaskStatus(roomID, taskID string, next TaskStatus) (*Task, error)` — validates transitions; abandoning clears ClaimedBy; error message contains "invalid transition" otherwise
  - `(*Store).ListTasks(roomID string) ([]Task, error)` — ordered by CreatedAt then ID

- [ ] **Step 1: Write the failing tests**

Append to `store_test.go`:

```go
func TestPostClaimUpdateListTasks(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")

	task, err := s.PostTask(room.ID, &Task{Title: "build API", Interfaces: "GET /leads"})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if task.ID == "" || task.Status != TaskOpen || task.RoomID != room.ID {
		t.Fatalf("bad task: %+v", task)
	}

	claimed, err := s.ClaimTask(room.ID, task.ID, "codex-1")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed.Status != TaskClaimed || claimed.ClaimedBy != "codex-1" {
		t.Fatalf("bad claim: %+v", claimed)
	}

	if _, err := s.ClaimTask(room.ID, task.ID, "kimi-1"); err == nil ||
		!strings.Contains(err.Error(), "already claimed") {
		t.Fatalf("want already-claimed error, got %v", err)
	}

	for _, next := range []TaskStatus{TaskInProgress, TaskReview, TaskDone} {
		if _, err := s.UpdateTaskStatus(room.ID, task.ID, next); err != nil {
			t.Fatalf("update to %s: %v", next, err)
		}
	}

	list, err := s.ListTasks(room.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Status != TaskDone {
		t.Fatalf("bad list: %+v", list)
	}
}

func TestInvalidTransitionRejected(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	task, _ := s.PostTask(room.ID, &Task{Title: "t"})

	// open -> done skips the pipeline.
	if _, err := s.UpdateTaskStatus(room.ID, task.ID, TaskDone); err == nil ||
		!strings.Contains(err.Error(), "invalid transition") {
		t.Fatalf("want invalid-transition error, got %v", err)
	}
}

func TestAbandonedTaskIsReclaimable(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	task, _ := s.PostTask(room.ID, &Task{Title: "t"})
	if _, err := s.ClaimTask(room.ID, task.ID, "agent-a"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	ab, err := s.UpdateTaskStatus(room.ID, task.ID, TaskAbandoned)
	if err != nil {
		t.Fatalf("abandon: %v", err)
	}
	if ab.ClaimedBy != "" {
		t.Fatalf("abandon should clear claim: %+v", ab)
	}
	re, err := s.ClaimTask(room.ID, task.ID, "agent-b")
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if re.ClaimedBy != "agent-b" || re.Status != TaskClaimed {
		t.Fatalf("bad reclaim: %+v", re)
	}
}

func TestConcurrentClaimsExactlyOneWins(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	task, _ := s.PostTask(room.ID, &Task{Title: "contested"})

	const n = 8
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		agent := fmt.Sprintf("agent-%d", i)
		go func() {
			_, err := s.ClaimTask(room.ID, task.ID, agent)
			errs <- err
		}()
	}
	wins := 0
	for i := 0; i < n; i++ {
		if <-errs == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("want exactly 1 winning claim, got %d", wins)
	}
}

func TestPostTaskToClosedRoomFails(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	if _, err := s.CloseRoom(room.ID); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := s.PostTask(room.ID, &Task{Title: "late"}); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Fatalf("want closed-room error, got %v", err)
	}
}
```

Add `"fmt"` and `"sort"`-free imports as needed (`fmt` for the concurrent test).

- [ ] **Step 2: Run to verify the new tests fail**

Run: `go test ./internal/room/...`
Expected: FAIL to build (undefined: PostTask et al.).

- [ ] **Step 3: Implement the task-board methods**

Append to `store.go`:

```go
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
```

Add `"sort"` to store.go imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/room/... -v`
Expected: all tests PASS (Task 1's 3 + these 5). Run the concurrent test with the race detector once: `go test ./internal/room/... -race -run TestConcurrentClaims`
Expected: PASS, no race reports.

- [ ] **Step 5: Full verify and commit**

Run: `go build ./... && go test ./internal/room/... ./internal/daemon/... ./internal/mcp/...`

```bash
git add internal/room/store/
git commit -m "Add room task board with atomic claims

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Room store — channel messages with cursor reads

**Files:**
- Modify: `internal/room/store/store.go` (append methods)
- Test: `internal/room/store/store_test.go` (append tests)

**Interfaces:**
- Consumes: Tasks 1-2.
- Produces:
  - `(*Store).PostMessage(roomID string, m *Message) (*Message, error)` — validates From/Kind/Body; assigns Seq (per-room, starts at 1) and CreatedAt; error if room missing/closed
  - `(*Store).ReadSince(roomID string, cursor uint64, limit int) ([]Message, uint64, error)` — messages with `Seq > cursor` in seq order, up to limit (default 50); second return is the new cursor (last returned Seq, or the input cursor when nothing new)

- [ ] **Step 1: Write the failing tests**

Append to `store_test.go`:

```go
func TestPostAndReadSince(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")

	for i := 1; i <= 5; i++ {
		m, err := s.PostMessage(room.ID, &Message{
			From: "claude-1", Kind: KindStatus, Body: fmt.Sprintf("update %d", i),
		})
		if err != nil {
			t.Fatalf("post %d: %v", i, err)
		}
		if m.Seq != uint64(i) {
			t.Fatalf("want seq %d, got %d", i, m.Seq)
		}
	}

	msgs, cursor, err := s.ReadSince(room.ID, 0, 50)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(msgs) != 5 || cursor != 5 {
		t.Fatalf("want 5 msgs cursor 5, got %d msgs cursor %d", len(msgs), cursor)
	}

	// Incremental read from the returned cursor sees only new messages.
	if _, err := s.PostMessage(room.ID, &Message{From: "codex-1", Kind: KindHandoff, Body: "task 2 unblocked"}); err != nil {
		t.Fatalf("post: %v", err)
	}
	msgs, cursor, err = s.ReadSince(room.ID, cursor, 50)
	if err != nil {
		t.Fatalf("read2: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Kind != KindHandoff || cursor != 6 {
		t.Fatalf("bad incremental read: %d msgs, cursor %d", len(msgs), cursor)
	}

	// Nothing new: empty result, cursor unchanged.
	msgs, cursor, err = s.ReadSince(room.ID, cursor, 50)
	if err != nil {
		t.Fatalf("read3: %v", err)
	}
	if len(msgs) != 0 || cursor != 6 {
		t.Fatalf("want empty read cursor 6, got %d msgs cursor %d", len(msgs), cursor)
	}
}

func TestReadSinceRespectsLimit(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	for i := 0; i < 10; i++ {
		if _, err := s.PostMessage(room.ID, &Message{From: "a", Kind: KindStatus, Body: "m"}); err != nil {
			t.Fatalf("post: %v", err)
		}
	}
	msgs, cursor, err := s.ReadSince(room.ID, 0, 3)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(msgs) != 3 || cursor != 3 {
		t.Fatalf("limit ignored: %d msgs cursor %d", len(msgs), cursor)
	}
}

func TestPostMessageValidation(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")

	if _, err := s.PostMessage(room.ID, &Message{Kind: KindStatus, Body: "x"}); err == nil {
		t.Fatal("want error for missing From")
	}
	if _, err := s.PostMessage(room.ID, &Message{From: "a", Kind: "gossip", Body: "x"}); err == nil {
		t.Fatal("want error for bad kind")
	}
	if _, err := s.CloseRoom(room.ID); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := s.PostMessage(room.ID, &Message{From: "a", Kind: KindStatus, Body: "x"}); err == nil {
		t.Fatal("want error for closed room")
	}
}
```

- [ ] **Step 2: Run to verify the new tests fail**

Run: `go test ./internal/room/...`
Expected: FAIL to build (undefined: PostMessage, ReadSince).

- [ ] **Step 3: Implement channel methods**

Append to `store.go`:

```go
func validKind(k MessageKind) bool {
	switch k {
	case KindStatus, KindHandoff, KindContract, KindReview:
		return true
	}
	return false
}

func seqKey(roomID string) string           { return seqPrefix + roomID }
func msgKey(roomID string, seq uint64) string { return fmt.Sprintf("%s%s:%020d", msgPrefix, roomID, seq) }

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
```

Note: reading from a **closed** room is allowed by design (`ReadSince` checks existence only, not open status) — history stays queryable forever.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/room/... -v`
Expected: all store tests PASS.

- [ ] **Step 5: Full verify and commit**

Run: `go build ./... && go test ./internal/room/... ./internal/daemon/... ./internal/mcp/...`

```bash
git add internal/room/store/
git commit -m "Add room channel with cursor reads

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: Daemon room methods

**Files:**
- Create: `internal/daemon/room_methods.go`
- Modify: `internal/daemon/daemon.go` (struct fields; one registration call; one close call)
- Test: `internal/daemon/room_methods_test.go`

**Interfaces:**
- Consumes: the full `internal/room/store` API from Tasks 1-3; daemon idioms from `internal/daemon/http_methods.go` (handler shape, `rpc.Error` codes) and `daemon.go`'s lazy `memoryStore()` pattern.
- Produces RPC methods (used by Task 5's MCP dispatch): `room.create`, `room.close`, `room.task_post`, `room.task_claim`, `room.task_update`, `room.task_list`, `room.post`, `room.read`. Param shapes are defined below and mirrored by the MCP tool schemas.

- [ ] **Step 1: Write the failing tests**

```go
package daemon

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	roomstore "github.com/jeffdhooton/scry/internal/room/store"
)

func newTestRoomDaemon(t *testing.T) *Daemon {
	t.Helper()
	d := New(LayoutFor(t.TempDir()))
	t.Cleanup(func() { d.closeRooms() })
	return d
}

func callRoom(t *testing.T, d *Daemon, handler func(context.Context, json.RawMessage) (any, error), params any) json.RawMessage {
	t.Helper()
	res, err := handler(context.Background(), mustJSON(t, params))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	out, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	return out
}

func TestRoomLifecycleRoundtrip(t *testing.T) {
	d := newTestRoomDaemon(t)

	var room roomstore.Room
	raw := callRoom(t, d, d.handleRoomCreate, map[string]any{"run_id": "run-9", "repo": "/repo"})
	if err := json.Unmarshal(raw, &room); err != nil {
		t.Fatalf("unmarshal room: %v", err)
	}
	if room.ID == "" || room.Status != roomstore.RoomOpen {
		t.Fatalf("bad room: %+v", room)
	}

	var task roomstore.Task
	raw = callRoom(t, d, d.handleRoomTaskPost, map[string]any{
		"room_id": room.ID, "title": "build API", "interfaces": "GET /leads",
	})
	if err := json.Unmarshal(raw, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}

	raw = callRoom(t, d, d.handleRoomTaskClaim, map[string]any{
		"room_id": room.ID, "task_id": task.ID, "agent": "codex-1",
	})
	var claimed roomstore.Task
	if err := json.Unmarshal(raw, &claimed); err != nil {
		t.Fatalf("unmarshal claim: %v", err)
	}
	if claimed.ClaimedBy != "codex-1" {
		t.Fatalf("bad claim: %+v", claimed)
	}

	// Second claim by another agent surfaces the store's conflict error.
	_, err := d.handleRoomTaskClaim(context.Background(), mustJSON(t, map[string]any{
		"room_id": room.ID, "task_id": task.ID, "agent": "kimi-1",
	}))
	if err == nil || !strings.Contains(err.Error(), "already claimed") {
		t.Fatalf("want already-claimed, got %v", err)
	}

	callRoom(t, d, d.handleRoomPost, map[string]any{
		"room_id": room.ID, "task_id": task.ID, "from": "codex-1",
		"kind": "status", "body": "starting",
	})

	var read struct {
		Messages []roomstore.Message `json:"messages"`
		Cursor   uint64              `json:"cursor"`
	}
	raw = callRoom(t, d, d.handleRoomRead, map[string]any{"room_id": room.ID, "cursor": 0})
	if err := json.Unmarshal(raw, &read); err != nil {
		t.Fatalf("unmarshal read: %v", err)
	}
	if len(read.Messages) != 1 || read.Cursor != 1 {
		t.Fatalf("bad read: %+v", read)
	}

	raw = callRoom(t, d, d.handleRoomClose, map[string]any{"room_id": room.ID})
	var closed roomstore.Room
	if err := json.Unmarshal(raw, &closed); err != nil {
		t.Fatalf("unmarshal close: %v", err)
	}
	if closed.Status != roomstore.RoomClosed {
		t.Fatalf("not closed: %+v", closed)
	}
}

func TestRoomCreateRequiresRunID(t *testing.T) {
	d := newTestRoomDaemon(t)
	_, err := d.handleRoomCreate(context.Background(), mustJSON(t, map[string]any{"repo": "/repo"}))
	if err == nil || !strings.Contains(err.Error(), "run_id") {
		t.Fatalf("want run_id error, got %v", err)
	}
}

func TestRoomTaskListEmpty(t *testing.T) {
	d := newTestRoomDaemon(t)
	var room roomstore.Room
	raw := callRoom(t, d, d.handleRoomCreate, map[string]any{"run_id": "r", "repo": "/x"})
	if err := json.Unmarshal(raw, &room); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw = callRoom(t, d, d.handleRoomTaskList, map[string]any{"room_id": room.ID})
	var tasks []roomstore.Task
	if err := json.Unmarshal(raw, &tasks); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if tasks == nil || len(tasks) != 0 {
		t.Fatalf("want empty non-nil slice, got %#v", tasks)
	}
}
```

- [ ] **Step 2: Run to verify the tests fail**

Run: `go test ./internal/daemon/ -run TestRoom`
Expected: FAIL to build (undefined: handleRoomCreate, closeRooms).

- [ ] **Step 3: Implement room_methods.go**

```go
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	roomstore "github.com/jeffdhooton/scry/internal/room/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

func (d *Daemon) registerRoomMethods() {
	d.server.Register("room.create", d.handleRoomCreate)
	d.server.Register("room.close", d.handleRoomClose)
	d.server.Register("room.task_post", d.handleRoomTaskPost)
	d.server.Register("room.task_claim", d.handleRoomTaskClaim)
	d.server.Register("room.task_update", d.handleRoomTaskUpdate)
	d.server.Register("room.task_list", d.handleRoomTaskList)
	d.server.Register("room.post", d.handleRoomPost)
	d.server.Register("room.read", d.handleRoomRead)
}

// roomStore lazily opens the global room store on first use.
func (d *Daemon) roomStore() (*roomstore.Store, error) {
	d.roomOnce.Do(func() {
		dir := filepath.Join(d.scryHome(), "rooms")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			d.roomErr = fmt.Errorf("create rooms dir: %w", err)
			return
		}
		d.roomSt, d.roomErr = roomstore.Open(dir)
	})
	return d.roomSt, d.roomErr
}

func (d *Daemon) closeRooms() {
	if d.roomSt != nil {
		_ = d.roomSt.Close()
	}
}

func invalidParams(err error) error {
	return &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
}

// --- room.create ---

type RoomCreateParams struct {
	RunID string `json:"run_id"`
	Repo  string `json:"repo"`
}

func (d *Daemon) handleRoomCreate(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomCreateParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RunID == "" {
		return nil, invalidParams(fmt.Errorf("run_id is required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	return st.CreateRoom(p.RunID, p.Repo)
}

// --- room.close ---

type RoomCloseParams struct {
	RoomID string `json:"room_id"`
}

func (d *Daemon) handleRoomClose(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomCloseParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" {
		return nil, invalidParams(fmt.Errorf("room_id is required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	return st.CloseRoom(p.RoomID)
}

// --- room.task_post ---

type RoomTaskPostParams struct {
	RoomID     string   `json:"room_id"`
	Title      string   `json:"title"`
	Body       string   `json:"body,omitempty"`
	DependsOn  []string `json:"depends_on,omitempty"`
	Interfaces string   `json:"interfaces,omitempty"`
}

func (d *Daemon) handleRoomTaskPost(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomTaskPostParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" {
		return nil, invalidParams(fmt.Errorf("room_id is required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	return st.PostTask(p.RoomID, &roomstore.Task{
		Title:      p.Title,
		Body:       p.Body,
		DependsOn:  p.DependsOn,
		Interfaces: p.Interfaces,
	})
}

// --- room.task_claim ---

type RoomTaskClaimParams struct {
	RoomID string `json:"room_id"`
	TaskID string `json:"task_id"`
	Agent  string `json:"agent"`
}

func (d *Daemon) handleRoomTaskClaim(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomTaskClaimParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" || p.TaskID == "" || p.Agent == "" {
		return nil, invalidParams(fmt.Errorf("room_id, task_id, and agent are required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	return st.ClaimTask(p.RoomID, p.TaskID, p.Agent)
}

// --- room.task_update ---

type RoomTaskUpdateParams struct {
	RoomID string `json:"room_id"`
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

func (d *Daemon) handleRoomTaskUpdate(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomTaskUpdateParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" || p.TaskID == "" || p.Status == "" {
		return nil, invalidParams(fmt.Errorf("room_id, task_id, and status are required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	return st.UpdateTaskStatus(p.RoomID, p.TaskID, roomstore.TaskStatus(p.Status))
}

// --- room.task_list ---

type RoomTaskListParams struct {
	RoomID string `json:"room_id"`
}

func (d *Daemon) handleRoomTaskList(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomTaskListParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" {
		return nil, invalidParams(fmt.Errorf("room_id is required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	tasks, err := st.ListTasks(p.RoomID)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		tasks = []roomstore.Task{}
	}
	return tasks, nil
}

// --- room.post ---

type RoomPostParams struct {
	RoomID string `json:"room_id"`
	TaskID string `json:"task_id,omitempty"`
	From   string `json:"from"`
	Kind   string `json:"kind"`
	Body   string `json:"body"`
}

func (d *Daemon) handleRoomPost(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomPostParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" {
		return nil, invalidParams(fmt.Errorf("room_id is required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	return st.PostMessage(p.RoomID, &roomstore.Message{
		TaskID: p.TaskID,
		From:   p.From,
		Kind:   roomstore.MessageKind(p.Kind),
		Body:   p.Body,
	})
}

// --- room.read ---

type RoomReadParams struct {
	RoomID string `json:"room_id"`
	Cursor uint64 `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type RoomReadResult struct {
	Messages []roomstore.Message `json:"messages"`
	Cursor   uint64              `json:"cursor"`
}

func (d *Daemon) handleRoomRead(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomReadParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" {
		return nil, invalidParams(fmt.Errorf("room_id is required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	msgs, cursor, err := st.ReadSince(p.RoomID, p.Cursor, p.Limit)
	if err != nil {
		return nil, err
	}
	if msgs == nil {
		msgs = []roomstore.Message{}
	}
	return &RoomReadResult{Messages: msgs, Cursor: cursor}, nil
}
```

- [ ] **Step 4: Wire into daemon.go**

Three small edits (read the file first to place them exactly):

1. Struct fields — next to the existing `memStore` fields (~line 69), add:

```go
	roomSt   *roomstore.Store
	roomOnce sync.Once
	roomErr  error
```

Add the import `roomstore "github.com/jeffdhooton/scry/internal/room/store"` (and `sync` if not present).

2. Registration — next to `d.registerHTTPMethods()` (~line 120), add:

```go
	d.registerRoomMethods()
```

3. Shutdown — next to `defer d.closeHTTP()` (~line 175), add:

```go
	defer d.closeRooms()
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/daemon/ -run TestRoom -v`
Expected: 3 tests PASS.

- [ ] **Step 6: Full verify and commit**

Run: `go build ./... && go test ./internal/room/... ./internal/daemon/... ./internal/mcp/...`

```bash
git add internal/daemon/room_methods.go internal/daemon/room_methods_test.go internal/daemon/daemon.go
git commit -m "Wire room methods into the scry daemon

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: MCP room tools

**Files:**
- Create: `internal/mcp/room_tools.go`
- Modify: `internal/mcp/server.go` (append definitions to the aggregation at ~line 324-330; add dispatch cases in the tools/call switch at ~line 391-409)

**Interfaces:**
- Consumes: Task 4's RPC methods; mcp package idioms from `internal/mcp/http_tools.go` (`tool` struct, `mustMarshal`, `callHTTPQuery` shape, `logCall`, `writeToolError`/`writeToolResult`, `prettyJSON`).
- Produces: the eight `scry_*` room tools visible to every MCP client.

- [ ] **Step 1: Write room_tools.go**

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var roomToolDefinitions = []tool{
	{
		Name:        "scry_room_create",
		Description: "Create a fleet room: a shared task board + message channel for a multi-agent run. Called by the orchestrator (setpoint), not by workers. Returns the room with its ID.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"run_id": map[string]any{"type": "string", "description": "Fleet run identifier this room belongs to."},
				"repo":   map[string]any{"type": "string", "description": "Absolute path of the repo the run works on."},
			},
			"required": []string{"run_id"},
		}),
	},
	{
		Name:        "scry_room_close",
		Description: "Close a fleet room when its run completes. History is retained forever (it feeds the memory graph); closed rooms reject new tasks and messages but remain readable.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id": map[string]any{"type": "string", "description": "Room ID from scry_room_create."},
			},
			"required": []string{"room_id"},
		}),
	},
	{
		Name:        "scry_task_post",
		Description: "Post a task to a room's board with optional dependency edges (depends_on task IDs) and interface stubs the implementer must honor. Orchestrator-side; workers claim tasks with scry_task_claim.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id":    map[string]any{"type": "string"},
				"title":      map[string]any{"type": "string", "description": "Short imperative task title."},
				"body":       map[string]any{"type": "string", "description": "Full task description."},
				"depends_on": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Task IDs that must be done first."},
				"interfaces": map[string]any{"type": "string", "description": "Boundary contracts this task must honor or negotiate."},
			},
			"required": []string{"room_id", "title"},
		}),
	},
	{
		Name:        "scry_task_claim",
		Description: "Atomically claim a task on the room board. Fails with 'already claimed' if another agent holds it. Claim before starting work; abandoned tasks are re-claimable.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id": map[string]any{"type": "string"},
				"task_id": map[string]any{"type": "string"},
				"agent":   map[string]any{"type": "string", "description": "Your agent identity, e.g. 'codex-1'."},
			},
			"required": []string{"room_id", "task_id", "agent"},
		}),
	},
	{
		Name:        "scry_task_update",
		Description: "Advance a task's status. Valid transitions: claimed→in_progress, in_progress→review, review→done|in_progress; claimed/in_progress/review→abandoned (clears the claim). Claiming itself is only done via scry_task_claim.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id": map[string]any{"type": "string"},
				"task_id": map[string]any{"type": "string"},
				"status":  map[string]any{"type": "string", "enum": []string{"in_progress", "review", "done", "abandoned"}},
			},
			"required": []string{"room_id", "task_id", "status"},
		}),
	},
	{
		Name:        "scry_task_list",
		Description: "List all tasks on a room's board with status, claims, and dependencies. Use to find claimable work and check whether your dependencies are done.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id": map[string]any{"type": "string"},
			},
			"required": []string{"room_id"},
		}),
	},
	{
		Name:        "scry_post",
		Description: "Post a message to the room channel. kind=status for milestones, handoff when your output unblocks a dependent task, contract to negotiate/accept a boundary interface, review to request or respond to cross-review. Thread by task_id.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id": map[string]any{"type": "string"},
				"task_id": map[string]any{"type": "string", "description": "Task this message is about (threads the channel)."},
				"from":    map[string]any{"type": "string", "description": "Your agent identity."},
				"kind":    map[string]any{"type": "string", "enum": []string{"status", "handoff", "contract", "review"}},
				"body":    map[string]any{"type": "string"},
			},
			"required": []string{"room_id", "from", "kind", "body"},
		}),
	},
	{
		Name:        "scry_read",
		Description: "Read room channel messages after your cursor (incremental poll). Returns messages plus the new cursor; pass that cursor next time. Cursor 0 reads from the beginning. Works on closed rooms too (history is permanent).",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id": map[string]any{"type": "string"},
				"cursor":  map[string]any{"type": "integer", "description": "Last seq you have seen; 0 for the start."},
				"limit":   map[string]any{"type": "integer", "description": "Max messages (default 50)."},
			},
			"required": []string{"room_id"},
		}),
	},
}

func (s *Server) callRoomQuery(ctx context.Context, id json.RawMessage, toolName, rpcMethod string, rawArgs json.RawMessage) {
	start := nowUTC()

	client, err := s.dial()
	if err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, LatencyMs: time.Since(start).Milliseconds(), Error: "dial failed"})
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()

	var params any
	if len(rawArgs) > 0 && string(rawArgs) != "{}" {
		var m map[string]any
		if err := json.Unmarshal(rawArgs, &m); err != nil {
			s.writeToolError(id, "invalid arguments: "+err.Error())
			return
		}
		params = m
	} else {
		params = struct{}{}
	}

	var raw json.RawMessage
	if err := client.Call(ctx, rpcMethod, params, &raw); err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error()})
		s.writeToolError(id, fmt.Sprintf("scry %s: %s", rpcMethod, err.Error()))
		return
	}

	logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, LatencyMs: time.Since(start).Milliseconds()})
	s.writeToolResult(id, prettyJSON(raw), false)
}
```

- [ ] **Step 2: Wire into server.go**

Two edits (read the surrounding lines first):

1. In the tool aggregation (~line 324-330), extend the capacity expression with `+len(roomToolDefinitions)` and add:

```go
	all = append(all, roomToolDefinitions...)
```

2. In the tools/call dispatch switch (~line 391-409), add these cases alongside the http/memory ones:

```go
	case "scry_room_create":
		s.callRoomQuery(ctx, req.ID, "scry_room_create", "room.create", p.Arguments)
	case "scry_room_close":
		s.callRoomQuery(ctx, req.ID, "scry_room_close", "room.close", p.Arguments)
	case "scry_task_post":
		s.callRoomQuery(ctx, req.ID, "scry_task_post", "room.task_post", p.Arguments)
	case "scry_task_claim":
		s.callRoomQuery(ctx, req.ID, "scry_task_claim", "room.task_claim", p.Arguments)
	case "scry_task_update":
		s.callRoomQuery(ctx, req.ID, "scry_task_update", "room.task_update", p.Arguments)
	case "scry_task_list":
		s.callRoomQuery(ctx, req.ID, "scry_task_list", "room.task_list", p.Arguments)
	case "scry_post":
		s.callRoomQuery(ctx, req.ID, "scry_post", "room.post", p.Arguments)
	case "scry_read":
		s.callRoomQuery(ctx, req.ID, "scry_read", "room.read", p.Arguments)
	}
```

(Insert the cases inside the existing switch — do not duplicate the closing brace; match the file's existing case ordering style.)

- [ ] **Step 3: Verify tool listing includes the room tools**

Run: `go test ./internal/mcp/ -v 2>&1 | tail -5` and `go vet ./internal/mcp/`
Expected: existing mcp tests still PASS; vet clean. If `server_test.go` has a tools-listing test that asserts specific tools, confirm it still passes (it has no count assertions as of the plan's writing).

- [ ] **Step 4: Full verify and commit**

Run: `go build ./... && go test ./internal/room/... ./internal/daemon/... ./internal/mcp/...`

```bash
git add internal/mcp/room_tools.go internal/mcp/server.go
git commit -m "Expose room domain as scry MCP tools

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```
