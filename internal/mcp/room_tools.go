package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		Description: "Post a message to the room channel. kind=status for milestones, handoff when your output unblocks a dependent task, contract to negotiate/accept a boundary interface, publish to announce a boundary you built when no negotiation is expected, review to request or respond to cross-review. Thread by task_id, and set reply_to to the seq you are answering. Every post returns its seq — cite that seq when you refer back to a message.",
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
		msg := fmt.Sprintf("scry %s: %s", rpcMethod, err.Error())
		// A method the daemon does not know means the two halves disagree
		// about the protocol. In practice this is always a long-lived `scry
		// mcp` process that outlived a daemon upgrade — say so, because the
		// raw error reads like a room bug.
		if strings.Contains(strings.ToLower(err.Error()), "method not found") {
			msg += "\n\nThis scry MCP process and the scry daemon disagree about the room " +
				"protocol. Restart your MCP connection (in Claude Code: /mcp, then " +
				"reconnect scry) and retry."
		}
		s.writeToolError(id, msg)
		return
	}

	logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, LatencyMs: time.Since(start).Milliseconds()})
	s.writeToolResult(id, prettyJSON(raw), false)
}
