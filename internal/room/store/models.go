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
	// KindPublish is a one-way announcement: "here is the shape I built, no
	// reply expected". Self-contained tasks were misusing `contract` for
	// this, which drained propose/accept of meaning for the tasks that
	// actually negotiate.
	KindPublish MessageKind = "publish"
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
	// ReplyTo is the Seq of the message this one answers. Reviewers were
	// already citing "at seq 8 you accepted" in prose; this makes the edge
	// real so a thread can be reconstructed instead of re-read.
	ReplyTo uint64 `json:"reply_to,omitempty"`
	// Structured review/handoff fields. All optional — the prose body stays
	// the human-readable record. These exist because the orchestrator was
	// regex-harvesting verdicts and PR links out of free text.
	Verdict  string   `json:"verdict,omitempty"`  // APPROVED | CHANGES (review only)
	Severity string   `json:"severity,omitempty"` // P0..P3
	Findings []string `json:"findings,omitempty"`
	PRURL    string   `json:"pr_url,omitempty"`
}
