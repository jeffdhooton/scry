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
	d.server.Register("room.get", d.handleRoomGet)
	d.server.Register("room.list", d.handleRoomList)
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

// RoomProtocolVersion is bumped whenever the room domain gains tools or
// message fields an older `scry mcp` process cannot see. Long-lived MCP
// processes outlive daemon upgrades, and a stale one fails in ways that read
// as a room bug rather than a restart prompt.
//
//	1 — original room domain (create/close/task_*/post/read)
//	2 — room.get, room.list, reply_to, structured verdict fields, publish kind
const RoomProtocolVersion = 2

// --- room.get ---

type RoomGetParams struct {
	RoomID string `json:"room_id,omitempty"`
	RunID  string `json:"run_id,omitempty"`
}

func (d *Daemon) handleRoomGet(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomGetParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, invalidParams(err)
	}
	if p.RoomID == "" && p.RunID == "" {
		return nil, invalidParams(fmt.Errorf("one of room_id or run_id is required"))
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	if p.RoomID != "" {
		return st.GetRoom(p.RoomID)
	}
	return st.FindRoomByRunID(p.RunID)
}

// --- room.list ---

type RoomListParams struct {
	Limit int `json:"limit,omitempty"`
}

func (d *Daemon) handleRoomList(_ context.Context, raw json.RawMessage) (any, error) {
	var p RoomListParams
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	st, err := d.roomStore()
	if err != nil {
		return nil, err
	}
	rooms, err := st.ListRooms()
	if err != nil {
		return nil, err
	}
	if p.Limit > 0 && len(rooms) > p.Limit {
		rooms = rooms[:p.Limit]
	}
	if rooms == nil {
		rooms = []roomstore.Room{}
	}
	return rooms, nil
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
	RoomID   string   `json:"room_id"`
	TaskID   string   `json:"task_id,omitempty"`
	From     string   `json:"from"`
	Kind     string   `json:"kind"`
	Body     string   `json:"body"`
	ReplyTo  uint64   `json:"reply_to,omitempty"`
	Verdict  string   `json:"verdict,omitempty"`
	Severity string   `json:"severity,omitempty"`
	Findings []string `json:"findings,omitempty"`
	PRURL    string   `json:"pr_url,omitempty"`
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
		TaskID:   p.TaskID,
		From:     p.From,
		Kind:     roomstore.MessageKind(p.Kind),
		Body:     p.Body,
		ReplyTo:  p.ReplyTo,
		Verdict:  p.Verdict,
		Severity: p.Severity,
		Findings: p.Findings,
		PRURL:    p.PRURL,
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
