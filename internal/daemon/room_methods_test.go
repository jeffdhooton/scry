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
