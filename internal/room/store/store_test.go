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
