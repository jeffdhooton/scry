# Fleet Retro — scry Room Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix retro items 12–17 so the room domain answers the questions agents actually asked it during the first fleets — which room is this run, which message am I replying to, what was the verdict — instead of forcing prose conventions and out-of-band manifests.

**Architecture:** Three layers, bottom-up: the BadgerDB store (`internal/room/store`) gains room lookup by run, a `reply_to` edge, optional structured review fields, and a `publish` message kind; the daemon (`internal/daemon/room_methods.go`) exposes them as RPC methods; the MCP server (`internal/mcp/room_tools.go`) exposes those as tools. Item 16 adds a room-protocol version to the existing `status` RPC so a stale long-lived `scry mcp` process announces itself instead of failing mysteriously. Item 17 is separate and touches only the memory extraction path.

**Tech Stack:** Go (no CGO), BadgerDB v4, the existing `internal/rpc` JSON-RPC server, zerolog.

**Spec:** `docs/retros/2026-08-13-first-fleets.md` (items 12–17), which lives in the setpoint repo at `~/workspace/setpoint/docs/retros/2026-08-13-first-fleets.md`

## Global Constraints

- No CGO. Existing store patterns hold: all writes serialized through `Store.mu`, keys follow the documented `room:` / `task:` / `msg:` / `seq:` layout, values are JSON.
- Every new `Message` and `Task` field is optional (`omitempty`) so records written by the current binary still unmarshal.
- Room history is permanent. Nothing in this plan deletes or rewrites a stored message.
- `go build ./... && go test ./...` must pass before every commit.
- Tool descriptions are prompts, not docs: they are the only instruction the agent sees. Any new tool needs a description that says when to reach for it.
- Consumer note: setpoint's `review_verdict()` (see the setpoint plan, Task 9) already prefers a structured `verdict` field on review messages and falls back to prose. Task 3 below is what makes that field exist; the two land independently and in either order.

---

### Task 1: Find a room without already knowing its ID (retro item 12)

`room.get(run_id)` and a room listing were missing, and it bit three separate times — the viewer, the wave-2 relaunch, and mid-run monitoring — until `room.json` manifests papered over it. The daemon owns this answer.

**Files:**
- Modify: `internal/room/store/store.go`
- Modify: `internal/daemon/room_methods.go`
- Modify: `internal/mcp/room_tools.go`
- Modify: `internal/mcp/server.go` (tool dispatch table — find the `switch` that routes room tool names to RPC methods and add the two new entries alongside `scry_room_create`)
- Test: `internal/room/store/store_test.go`, `internal/daemon/room_methods_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `(*Store).ListRooms() ([]Room, error)` — newest first by `CreatedAt`.
  - `(*Store).FindRoomByRunID(runID string) (*Room, error)` — the newest room for that run; error `room for run %q not found` when none.
  - RPC `room.get` with params `{room_id?, run_id?}` returning a `Room`; RPC `room.list` with params `{limit?}` returning `[]Room`.
  - MCP tools `scry_room_get`, `scry_room_list`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/room/store/store_test.go`:

```go
func TestListRoomsNewestFirst(t *testing.T) {
	s := newTestStore(t)
	first, err := s.CreateRoom("run-a", "/repo")
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := s.CreateRoom("run-b", "/repo")
	if err != nil {
		t.Fatalf("create b: %v", err)
	}

	rooms, err := s.ListRooms()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("want 2 rooms, got %d", len(rooms))
	}
	if rooms[0].ID != second.ID || rooms[1].ID != first.ID {
		t.Fatalf("not newest-first: %s then %s", rooms[0].RunID, rooms[1].RunID)
	}
}

func TestFindRoomByRunIDReturnsTheNewest(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateRoom("sim-hookup", "/repo"); err != nil {
		t.Fatalf("create wave1: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	wave2, err := s.CreateRoom("sim-hookup", "/repo")
	if err != nil {
		t.Fatalf("create wave2: %v", err)
	}

	// Two waves reused the run id; the caller means the current one.
	got, err := s.FindRoomByRunID("sim-hookup")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != wave2.ID {
		t.Fatalf("want newest room %s, got %s", wave2.ID, got.ID)
	}
}

func TestFindRoomByRunIDNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.FindRoomByRunID("nope")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}
```

Add `"time"` to that file's imports.

Add to `internal/daemon/room_methods_test.go` (following the existing helper style in that file for constructing a daemon with a temp home and calling handlers with raw JSON):

```go
func TestHandleRoomGetByRunID(t *testing.T) {
	d := newTestDaemon(t)
	created := mustCreateRoom(t, d, "run-77", "/repo")

	got, err := d.handleRoomGet(context.Background(),
		json.RawMessage(`{"run_id":"run-77"}`))
	if err != nil {
		t.Fatalf("room.get: %v", err)
	}
	room, ok := got.(*roomstore.Room)
	if !ok {
		t.Fatalf("want *Room, got %T", got)
	}
	if room.ID != created.ID {
		t.Fatalf("want room %s, got %s", created.ID, room.ID)
	}
}

func TestHandleRoomGetRequiresOneIdentifier(t *testing.T) {
	d := newTestDaemon(t)
	if _, err := d.handleRoomGet(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("want an error when neither room_id nor run_id is given")
	}
}
```

If `newTestDaemon`/`mustCreateRoom` do not already exist in `room_methods_test.go`, add them modeled on how the existing tests in that file build a daemon and create a room.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/room/... ./internal/daemon/... 2>&1 | head -30`
Expected: FAIL — `s.ListRooms undefined`, `d.handleRoomGet undefined`

- [ ] **Step 3: Implement**

In `internal/room/store/store.go`, after `CloseRoom`:

```go
// ListRooms returns every room, newest first. Rooms are few (one per fleet
// run) and small, so a full scan is the right shape here.
func (s *Store) ListRooms() ([]Room, error) {
	prefix := []byte(roomPrefix)
	var rooms []Room
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			var room Room
			if err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &room)
			}); err != nil {
				continue
			}
			rooms = append(rooms, room)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].CreatedAt.Equal(rooms[j].CreatedAt) {
			return rooms[i].ID < rooms[j].ID
		}
		return rooms[i].CreatedAt.After(rooms[j].CreatedAt)
	})
	return rooms, nil
}

// FindRoomByRunID returns the newest room for a run. Run IDs are not unique:
// a second wave of the same fleet reuses the name, and the caller always
// means the current one.
func (s *Store) FindRoomByRunID(runID string) (*Room, error) {
	rooms, err := s.ListRooms()
	if err != nil {
		return nil, err
	}
	for i := range rooms {
		if rooms[i].RunID == runID {
			return &rooms[i], nil
		}
	}
	return nil, fmt.Errorf("room for run %q not found", runID)
}
```

In `internal/daemon/room_methods.go`, register and implement:

```go
	d.server.Register("room.get", d.handleRoomGet)
	d.server.Register("room.list", d.handleRoomList)
```

```go
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
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, invalidParams(err)
		}
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
```

In `internal/mcp/room_tools.go`, add to `roomToolDefinitions`:

```go
	{
		Name:        "scry_room_get",
		Description: "Look up a fleet room by room_id, or by run_id when you only know the fleet's name. Use this to rejoin or monitor a run whose room ID you do not have — do not go hunting for a room.json manifest on disk. With run_id, returns the newest room for that run (a second wave reuses the name).",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room_id": map[string]any{"type": "string", "description": "Room ID, if you have it."},
				"run_id":  map[string]any{"type": "string", "description": "Fleet run name, e.g. 'sim-hookup-wave2'."},
			},
		}),
	},
	{
		Name:        "scry_room_list",
		Description: "List fleet rooms, newest first, with their run IDs, repos, and open/closed status. Use to find a recent run when you know neither its room ID nor its exact run name.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{"type": "integer", "description": "Max rooms to return."},
			},
		}),
	},
```

Then wire both names to their RPC methods in the MCP server's room-tool dispatch (`scry_room_get` → `room.get`, `scry_room_list` → `room.list`), next to the existing `scry_room_create` → `room.create` entry.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./... 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/room internal/daemon internal/mcp
git commit -m "Answer room lookups by run id natively"
```

---

### Task 2: Make messages addressable (retro item 13)

Reviewers invented structural citation — "at seq 8 you accepted" — because `scry_post` gives them nothing addressable to reference. Seq already exists; make it a first-class citation surface with a real reply edge.

**Files:**
- Modify: `internal/room/store/models.go`
- Modify: `internal/room/store/store.go` (`PostMessage`)
- Modify: `internal/daemon/room_methods.go` (`RoomPostParams`)
- Modify: `internal/mcp/room_tools.go` (`scry_post` and `scry_read` descriptions/schemas)
- Test: `internal/room/store/store_test.go`

**Interfaces:**
- Consumes: nothing from Task 1.
- Produces: `Message.ReplyTo uint64` (`json:"reply_to,omitempty"`); `PostMessage` rejects a `ReplyTo` that does not exist in the room with `reply_to seq %d not found in room %q`; `scry_post` accepts `reply_to`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/room/store/store_test.go`:

```go
func TestPostMessageWithReplyTo(t *testing.T) {
	s := newTestStore(t)
	room, err := s.CreateRoom("run-1", "/repo")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	first, err := s.PostMessage(room.ID, &Message{From: "codex-1", Kind: KindContract, Body: "proposal"})
	if err != nil {
		t.Fatalf("post first: %v", err)
	}
	reply, err := s.PostMessage(room.ID, &Message{
		From: "claude-2", Kind: KindContract, Body: "accepted", ReplyTo: first.Seq,
	})
	if err != nil {
		t.Fatalf("post reply: %v", err)
	}
	if reply.ReplyTo != first.Seq {
		t.Fatalf("reply_to not persisted: %+v", reply)
	}

	msgs, _, err := s.ReadSince(room.ID, 0, 50)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(msgs) != 2 || msgs[1].ReplyTo != first.Seq {
		t.Fatalf("reply_to lost on read: %+v", msgs)
	}
}

func TestPostMessageRejectsUnknownReplyTo(t *testing.T) {
	s := newTestStore(t)
	room, err := s.CreateRoom("run-1", "/repo")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = s.PostMessage(room.ID, &Message{
		From: "codex-1", Kind: KindStatus, Body: "hi", ReplyTo: 42,
	})
	if err == nil || !strings.Contains(err.Error(), "reply_to seq 42 not found") {
		t.Fatalf("want unknown-reply_to error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/room/... 2>&1 | head -20`
Expected: FAIL — `unknown field ReplyTo in struct literal`

- [ ] **Step 3: Implement**

In `models.go`, add to `Message`:

```go
	// ReplyTo is the Seq of the message this one answers. Reviewers were
	// already citing "at seq 8 you accepted" in prose; this makes the edge
	// real so threads can be reconstructed instead of re-read.
	ReplyTo uint64 `json:"reply_to,omitempty"`
```

In `store.go`'s `PostMessage`, inside the `s.db.Update` closure, before allocating the new seq:

```go
		if m.ReplyTo > 0 {
			if _, err := txn.Get([]byte(msgKey(roomID, m.ReplyTo))); err != nil {
				return fmt.Errorf("reply_to seq %d not found in room %q", m.ReplyTo, roomID)
			}
		}
```

In `room_methods.go`, add `ReplyTo uint64 \`json:"reply_to,omitempty"\`` to `RoomPostParams` and pass `ReplyTo: p.ReplyTo` into the `roomstore.Message{...}` literal.

In `room_tools.go`, add to `scry_post`'s properties:

```go
				"reply_to": map[string]any{"type": "integer", "description": "Seq of the message you are answering. Always set this when responding to a contract proposal, a review finding, or a question — it is how the thread stays reconstructable."},
```

and extend the `scry_post` description with: `Every post returns its seq — cite that seq when you refer back to a message, and set reply_to when you answer one.` Extend `scry_read`'s description with: `Each message carries its seq and, when it answers another, reply_to.`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./... 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/room internal/daemon internal/mcp
git commit -m "Give room messages a reply edge"
```

---

### Task 3: Structured verdicts, severities, and deliverable links (retro item 14)

APPROVED/CHANGES, `[P0]`–`[P3]`, and PR URLs all lived in free text; the closing ceremony had to regex-harvest PRs out of prose. Add optional structured fields so the machine-read path stops guessing.

**Files:**
- Modify: `internal/room/store/models.go`
- Modify: `internal/room/store/store.go` (`PostMessage` validation)
- Modify: `internal/daemon/room_methods.go` (`RoomPostParams`)
- Modify: `internal/mcp/room_tools.go` (`scry_post` schema + description)
- Test: `internal/room/store/store_test.go`

**Interfaces:**
- Consumes: `Message.ReplyTo` from Task 2.
- Produces: `Message.Verdict string` (`"APPROVED"|"CHANGES"`, review kind only), `Message.Severity string` (`"P0".."P3"`), `Message.Findings []string`, `Message.PRURL string` — all `omitempty`. `PostMessage` rejects an invalid verdict/severity value, and rejects a verdict on a non-review message.

- [ ] **Step 1: Write the failing tests**

Add to `internal/room/store/store_test.go`:

```go
func TestPostReviewWithStructuredVerdict(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	m, err := s.PostMessage(room.ID, &Message{
		From: "codex-reviewer", Kind: KindReview, Body: "two findings",
		Verdict: "CHANGES", Severity: "P1",
		Findings: []string{"DTO leaks a FIN field", "no RBAC test"},
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if m.Verdict != "CHANGES" || m.Severity != "P1" || len(m.Findings) != 2 {
		t.Fatalf("structured fields lost: %+v", m)
	}
}

func TestPostRejectsInvalidVerdictAndSeverity(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")

	if _, err := s.PostMessage(room.ID, &Message{
		From: "r", Kind: KindReview, Body: "b", Verdict: "LGTM",
	}); err == nil || !strings.Contains(err.Error(), "invalid verdict") {
		t.Fatalf("want invalid-verdict error, got %v", err)
	}
	if _, err := s.PostMessage(room.ID, &Message{
		From: "r", Kind: KindReview, Body: "b", Severity: "urgent",
	}); err == nil || !strings.Contains(err.Error(), "invalid severity") {
		t.Fatalf("want invalid-severity error, got %v", err)
	}
}

func TestVerdictOnlyOnReviewMessages(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	if _, err := s.PostMessage(room.ID, &Message{
		From: "w", Kind: KindStatus, Body: "done", Verdict: "APPROVED",
	}); err == nil || !strings.Contains(err.Error(), "verdict is only valid") {
		t.Fatalf("want verdict-kind error, got %v", err)
	}
}

func TestHandoffCarriesPRURL(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	m, err := s.PostMessage(room.ID, &Message{
		From: "claude-1", Kind: KindHandoff, Body: "mirrors are live",
		PRURL: "https://github.com/x/y/pull/155",
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if m.PRURL != "https://github.com/x/y/pull/155" {
		t.Fatalf("pr_url lost: %+v", m)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/room/... 2>&1 | head -20`
Expected: FAIL — `unknown field Verdict in struct literal`

- [ ] **Step 3: Implement**

In `models.go`, add to `Message`:

```go
	// Structured review/handoff fields. All optional — the prose body stays
	// the human-readable record. These exist because the orchestrator was
	// regex-harvesting verdicts and PR links out of free text.
	Verdict  string   `json:"verdict,omitempty"`  // APPROVED | CHANGES (review only)
	Severity string   `json:"severity,omitempty"` // P0..P3
	Findings []string `json:"findings,omitempty"`
	PRURL    string   `json:"pr_url,omitempty"`
```

In `store.go`, add validators and call them at the top of `PostMessage` (alongside the existing `validKind` check):

```go
func validVerdict(v string) bool { return v == "APPROVED" || v == "CHANGES" }

func validSeverity(s string) bool {
	switch s {
	case "P0", "P1", "P2", "P3":
		return true
	}
	return false
}
```

```go
	if m.Verdict != "" {
		if !validVerdict(m.Verdict) {
			return nil, fmt.Errorf("invalid verdict %q (want APPROVED|CHANGES)", m.Verdict)
		}
		if m.Kind != KindReview {
			return nil, fmt.Errorf("verdict is only valid on review messages, got kind %q", m.Kind)
		}
	}
	if m.Severity != "" && !validSeverity(m.Severity) {
		return nil, fmt.Errorf("invalid severity %q (want P0|P1|P2|P3)", m.Severity)
	}
```

In `room_methods.go`, add the four fields to `RoomPostParams` (`verdict`, `severity`, `findings`, `pr_url`) and pass them into the `roomstore.Message{...}` literal.

In `room_tools.go`, add to `scry_post`'s properties:

```go
				"verdict":  map[string]any{"type": "string", "enum": []string{"APPROVED", "CHANGES"}, "description": "Review verdict. Set it on your FINAL review message — the orchestrator reads this field to decide whether the task is approved, and a task whose review never carries a verdict is reported unreviewed."},
				"severity": map[string]any{"type": "string", "enum": []string{"P0", "P1", "P2", "P3"}, "description": "Severity of the findings in this message."},
				"findings": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "One entry per discrete finding, so they can be tracked individually."},
				"pr_url":   map[string]any{"type": "string", "description": "Deliverable URL, on the status or handoff message that announces it."},
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./... 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/room internal/daemon internal/mcp
git commit -m "Add structured verdict, severity, and link fields"
```

---

### Task 4: A `publish` kind for one-way announcements (retro item 15)

Self-contained tasks used `contract` as a one-way broadcast, which erodes propose/accept semantics for the tasks that genuinely negotiate.

**Files:**
- Modify: `internal/room/store/models.go`
- Modify: `internal/room/store/store.go` (`validKind`)
- Modify: `internal/mcp/room_tools.go` (`scry_post` kind enum + description)
- Test: `internal/room/store/store_test.go`

**Interfaces:**
- Consumes: Task 3's validation block.
- Produces: `KindPublish MessageKind = "publish"`, accepted by `PostMessage` and offered in the `scry_post` enum.

- [ ] **Step 1: Write the failing test**

```go
func TestPublishKindIsAccepted(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	m, err := s.PostMessage(room.ID, &Message{
		From: "codex-1", Kind: KindPublish,
		Body: "endpoints are final: GET /kipu/referrals/referred-out",
	})
	if err != nil {
		t.Fatalf("post publish: %v", err)
	}
	if m.Kind != KindPublish {
		t.Fatalf("kind lost: %+v", m)
	}
}

func TestInvalidKindStillRejected(t *testing.T) {
	s := newTestStore(t)
	room, _ := s.CreateRoom("run-1", "/repo")
	if _, err := s.PostMessage(room.ID, &Message{
		From: "x", Kind: MessageKind("chatter"), Body: "b",
	}); err == nil || !strings.Contains(err.Error(), "invalid message kind") {
		t.Fatalf("want invalid-kind error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/room/... -run Publish 2>&1 | head -20`
Expected: FAIL — `undefined: KindPublish`

- [ ] **Step 3: Implement**

In `models.go`:

```go
const (
	KindStatus   MessageKind = "status"
	KindHandoff  MessageKind = "handoff"
	KindContract MessageKind = "contract"
	KindReview   MessageKind = "review"
	// KindPublish is a one-way announcement: "here is the shape I built,
	// no reply expected". Self-contained tasks were misusing `contract` for
	// this, which drained propose/accept of meaning for the tasks that
	// actually negotiate.
	KindPublish MessageKind = "publish"
)
```

In `store.go`'s `validKind`, add `KindPublish` to the case list and update the error text to `(want status|handoff|contract|publish|review)`.

In `room_tools.go`, add `"publish"` to the `scry_post` kind enum and extend the description: `kind=publish to announce a boundary you built when no negotiation is expected — reserve contract for genuine propose/accept.`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./... 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/room internal/mcp
git commit -m "Add a publish message kind"
```

---

### Task 5: Version-stamp the room protocol (retro item 16)

A long-lived `scry mcp` process predated the room domain and could not see the new tools, which read as a mysterious failure rather than "restart me".

**Files:**
- Modify: `internal/daemon/methods.go` (the `status` result struct and `handleStatus`)
- Modify: `internal/mcp/room_tools.go` (`callRoomQuery`)
- Test: `internal/daemon/room_methods_test.go`

**Interfaces:**
- Consumes: Tasks 1–4 (the protocol whose version this is).
- Produces: `const RoomProtocolVersion = 2` in `internal/daemon`; the `status` RPC result gains `"room_protocol": <int>`; `callRoomQuery` turns a method-not-found error into actionable restart advice.

- [ ] **Step 1: Write the failing test**

Add to `internal/daemon/room_methods_test.go`:

```go
func TestStatusReportsRoomProtocolVersion(t *testing.T) {
	d := newTestDaemon(t)
	got, err := d.handleStatus(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(blob, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	v, ok := m["room_protocol"]
	if !ok {
		t.Fatalf("status has no room_protocol field: %s", blob)
	}
	if int(v.(float64)) != RoomProtocolVersion {
		t.Fatalf("want protocol %d, got %v", RoomProtocolVersion, v)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/daemon/... -run RoomProtocol 2>&1 | head -20`
Expected: FAIL — `undefined: RoomProtocolVersion`

- [ ] **Step 3: Implement**

In `internal/daemon/room_methods.go`, at the top:

```go
// RoomProtocolVersion is bumped whenever the room domain gains tools or
// message fields that an older `scry mcp` process cannot see. Long-lived MCP
// processes outlive daemon upgrades, and a stale one fails in ways that read
// as a room bug rather than a restart prompt.
//
//	1 — original room domain (create/close/task_*/post/read)
//	2 — room.get, room.list, reply_to, structured verdict fields, publish kind
const RoomProtocolVersion = 2
```

In `internal/daemon/methods.go`, add a field to `StatusResult`:

```go
	RoomProtocol int `json:"room_protocol"`
```

and set it where `handleStatus` builds the value at `methods.go:155`:

```go
	res := &StatusResult{PID: os.Getpid(), RoomProtocol: RoomProtocolVersion}
```

In `internal/mcp/room_tools.go`, make the failure legible in `callRoomQuery` — replace the error branch of the `client.Call` with:

```go
	if err := client.Call(ctx, rpcMethod, params, &raw); err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error()})
		msg := fmt.Sprintf("scry %s: %s", rpcMethod, err.Error())
		// A method the daemon does not know means the two halves disagree
		// about the protocol. In practice this is always a long-lived `scry
		// mcp` process that outlived a daemon upgrade — say so, because the
		// raw error reads like a room bug.
		if strings.Contains(strings.ToLower(err.Error()), "method not found") {
			msg += "\n\nThis scry MCP process and the scry daemon disagree about the room protocol. " +
				"Restart your MCP connection (in Claude Code: /mcp, then reconnect scry) and retry."
		}
		s.writeToolError(id, msg)
		return
	}
```

Add `"strings"` to that file's imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./... 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/daemon internal/mcp
git commit -m "Version-stamp the room protocol"
```

---

### Task 6: Stop losing extractions quietly (retro item 17)

In one day the memory domain went silently dormant (the env key was lost on a daemon restart) and separately dropped a fact to `extract: invalid JSON after retry`. Both failures were silent. Make dormancy loud and give extraction a repair retry plus a dead-letter file so nothing is lost without a trace.

**Files:**
- Modify: `internal/daemon/daemon.go` (`buildMemoryExtractor`)
- Modify: `internal/memory/extract/haiku.go` (`Extract`)
- Modify: `internal/daemon/memory_methods.go` (dead-letter on `ErrParse`)
- Test: `internal/memory/extract/provider_test.go` (or a new `haiku_test.go` in the same package)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `Haiku.Extract` attempts up to **two** corrective retries, the second including the parse error and an explicit "return ONLY the JSON object, no prose, no code fence" repair instruction; `(*Daemon).deadLetter(ep distill.RawEpisode, raw string, err error)` writes `<scryHome>/memory/dead-letter/<timestamp>-<episode-id>.json`; `buildMemoryExtractor` logs a warning naming the missing/invalid provider config.

- [ ] **Step 1: Write the failing tests**

Add to the `extract` package (the existing `provider_test.go` already exercises this package; follow its fake-client pattern — if it has no HTTP-level fake, add one with `httptest.NewServer` and point the Anthropic client's base URL at it):

```go
func TestExtractRetriesTwiceBeforeGivingUp(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		// Always invalid JSON in the text block.
		fmt.Fprint(w, `{"content":[{"type":"text","text":"not json"}]}`)
	}))
	defer srv.Close()

	h := newTestHaiku(t, srv.URL)
	_, err := h.Extract(context.Background(), distill.RawEpisode{Summary: "x"}, nil)
	if err == nil || !errors.Is(err, ErrParse) {
		t.Fatalf("want ErrParse, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("want 1 attempt + 2 repair retries = 3 calls, got %d", calls)
	}
}

func TestExtractSucceedsOnTheSecondRepair(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls < 3 {
			fmt.Fprint(w, `{"content":[{"type":"text","text":"still not json"}]}`)
			return
		}
		fmt.Fprint(w, `{"content":[{"type":"text","text":"{\"entities\":[],\"facts\":[]}"}]}`)
	}))
	defer srv.Close()

	h := newTestHaiku(t, srv.URL)
	if _, err := h.Extract(context.Background(), distill.RawEpisode{Summary: "x"}, nil); err != nil {
		t.Fatalf("want success on the second repair, got %v", err)
	}
}
```

Add a `newTestHaiku(t *testing.T, baseURL string) *Haiku` helper that builds a `Haiku` whose Anthropic client points at `baseURL` with a dummy key.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/memory/extract/... 2>&1 | head -20`
Expected: FAIL — `want 1 attempt + 2 repair retries = 3 calls, got 2`

- [ ] **Step 3: Implement**

Rewrite the retry section of `Haiku.Extract` as a loop:

```go
	raw := concatText(resp)
	result, parseErr := ParseResult(raw)
	if parseErr == nil {
		return result, nil
	}

	// Two corrective retries. The first echoes the invalid reply and asks for
	// a correction; the second is an explicit repair instruction, because a
	// model that produced prose once tends to produce prose twice. A fact was
	// lost to a single-retry give-up (retro item 17) — the extra attempt is
	// far cheaper than the loss.
	repairs := []string{
		"Invalid JSON (%s). Return only the corrected JSON object.",
		"Still invalid JSON (%s). Return ONLY the JSON object: no prose, no " +
			"explanation, no markdown code fence. Start your reply with { and end it with }.",
	}
	for _, tmpl := range repairs {
		params.Messages = append(params.Messages,
			anthropic.NewAssistantMessage(anthropic.NewTextBlock(raw)),
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				fmt.Sprintf(tmpl, parseErr),
			)),
		)
		next, err := h.client.Messages.New(ctx, params)
		if err != nil {
			return Result{}, fmt.Errorf("extract: haiku retry request failed: %w", err)
		}
		raw = concatText(next)
		result, parseErr = ParseResult(raw)
		if parseErr == nil {
			return result, nil
		}
	}
	// RawReply is carried on the error so the caller can dead-letter the
	// exact text instead of only the parse failure.
	return Result{}, fmt.Errorf("extract: invalid JSON after 2 repairs: %w: %w (reply: %.400q)",
		ErrParse, parseErr, raw)
```

In `internal/daemon/daemon.go`, make dormancy loud:

```go
func buildMemoryExtractor() extract.Extractor {
	p := extract.ProviderFromEnv()
	if p.APIKey == "" {
		log.Warn().Msg("memory extraction DORMANT: no API key in the environment " +
			"(episodes will be stored but never resolved into facts). Set the " +
			"provider key and restart the daemon.")
		return nil
	}
	if err := p.Validate(); err != nil {
		log.Warn().Err(err).Msg("memory extraction DORMANT: unusable provider config " +
			"(episodes will be stored but never resolved into facts)")
		return nil
	}
	return extract.NewHaiku(p)
}
```

using whatever zerolog import alias the file already uses (`github.com/rs/zerolog/log`; add it if absent).

In `internal/daemon/memory_methods.go`, add the dead letter and call it wherever an extraction error is currently swallowed or logged in the remember/commit path:

```go
// deadLetter records an extraction that could not be parsed. Losing a fact is
// acceptable only if it leaves a trace you can replay (retro item 17).
func (d *Daemon) deadLetter(episodeID string, err error) {
	dir := filepath.Join(d.scryHome(), "memory", "dead-letter")
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		log.Warn().Err(mkErr).Msg("could not create memory dead-letter dir")
		return
	}
	name := fmt.Sprintf("%s-%s.json", time.Now().UTC().Format("20060102T150405Z"), episodeID)
	payload, _ := json.MarshalIndent(map[string]string{
		"episode_id": episodeID,
		"error":      err.Error(),
		"at":         time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if wErr := os.WriteFile(filepath.Join(dir, name), payload, 0o644); wErr != nil {
		log.Warn().Err(wErr).Msg("could not write memory dead-letter file")
		return
	}
	log.Warn().Str("episode", episodeID).Str("file", name).
		Msg("memory extraction failed — episode dead-lettered for replay")
}
```

and wire it into the one extraction call site, `handleMemoryRemember` at `internal/daemon/memory_methods.go:492`. That line reads:

```go
	result, err := d.memExtractor.Extract(ctx, rawEp, p.Entities)
```

Immediately after it, before the existing error handling, add:

```go
	if err != nil && errors.Is(err, extract.ErrParse) {
		// The episode is already stored; only the resolution into facts was
		// lost. Leave a replayable trace rather than dropping it silently.
		d.deadLetter(rawEp.ID, err)
	}
```

Use whatever field on `rawEp` carries its identifier (`distill.RawEpisode`); if it has no ID field, pass the episode's source ref or a hash of its summary instead — the file name only has to be unique and traceable. Add `"time"`, `"os"`, `"errors"` and the zerolog import as needed.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go test ./... 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 5: Verify dormancy is actually loud, by hand**

```bash
go build -o /tmp/scry-test ./cmd/scry
env -u ANTHROPIC_API_KEY /tmp/scry-test daemon run --origin transient 2>&1 | head -5
```

Expected: a `memory extraction DORMANT` warning within the first lines. Stop it with Ctrl-C.

- [ ] **Step 6: Commit**

```bash
git add internal/daemon internal/memory
git commit -m "Make dormancy loud and dead-letter failed extractions"
```

---

### Task 7: Restart the operating session's MCP and mark the retro items done

**Files:**
- Modify: `~/workspace/setpoint/docs/retros/2026-08-13-first-fleets.md`

- [ ] **Step 1: Install the new binary and restart the daemon**

```bash
go install ./cmd/scry && scry daemon restart 2>/dev/null || scry status
```

- [ ] **Step 2: Reconnect the MCP and confirm the new tools resolve**

In Claude Code, run `/mcp` and reconnect `scry`, then confirm `scry_room_list` returns the recent rooms. This is the item-16 loop closing on itself: the stale-process problem is exactly what this step exercises.

- [ ] **Step 3: Annotate the retro**

Append ` — FIXED (<commit subject>)` to items 12–17 in `~/workspace/setpoint/docs/retros/2026-08-13-first-fleets.md`. That file lives in the setpoint repo; commit it there.
