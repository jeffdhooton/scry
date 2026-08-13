package store

import (
	"fmt"
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

func TestClosedRoomHistoryRemainsReadable(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	task, err := s.PostTask(room.ID, &Task{Title: "t"})
	if err != nil {
		t.Fatalf("post task: %v", err)
	}
	if _, err := s.PostMessage(room.ID, &Message{From: "a", Kind: KindStatus, Body: "done"}); err != nil {
		t.Fatalf("post message: %v", err)
	}
	if _, err := s.CloseRoom(room.ID); err != nil {
		t.Fatalf("close: %v", err)
	}

	msgs, cursor, err := s.ReadSince(room.ID, 0, 50)
	if err != nil {
		t.Fatalf("read closed room: %v", err)
	}
	if len(msgs) != 1 || cursor != 1 {
		t.Fatalf("closed-room read: %d msgs cursor %d", len(msgs), cursor)
	}

	tasks, err := s.ListTasks(room.ID)
	if err != nil {
		t.Fatalf("list closed room: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("closed-room list: %+v", tasks)
	}
}

func TestListTasksUnknownRoomErrors(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.ListTasks("nope"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}
