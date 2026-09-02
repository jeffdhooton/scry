package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var memoryToolDefinitions = []tool{
	{
		Name:        "scry_recall",
		Description: "Global cross-session memory: entities (projects, services, machines, people, decisions) with time-stamped facts extracted from past Claude/Codex sessions and loom runs. Use FIRST when the user references a project, machine, or decision not defined in the current context (e.g. 'set this up on hermes'). Returns matched entities, their current facts, and provenance.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Free-text search — entity name, alias, or topic."},
				"as_of": map[string]any{"type": "string", "description": "RFC3339 timestamp. Return facts as they stood at this point in time instead of current. Omit for current state."},
				"limit": map[string]any{"type": "integer", "description": "Max entities to return (default 5)."},
			},
			"required": []string{"query"},
		}),
	},
	{
		Name:        "scry_memory_path",
		Description: "How two remembered entities relate: shortest chain of facts between them (e.g. book-system → deployed_on → hermes-mini).",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"from": map[string]any{"type": "string", "description": "Source entity name or slug."},
				"to":   map[string]any{"type": "string", "description": "Target entity name or slug."},
			},
			"required": []string{"from", "to"},
		}),
	},
	{
		Name:        "scry_episodes",
		Description: "Recent episodes (session/run excerpts) that mention an entity, with summaries and source refs — use to trace when/where something was discussed.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"entity": map[string]any{"type": "string", "description": "Entity name or slug to find episodes for."},
				"limit":  map[string]any{"type": "integer", "description": "Max episodes to return (default 10)."},
			},
			"required": []string{"entity"},
		}),
	},
	{
		Name:        "scry_remember",
		Description: "Store a durable fact in global memory (e.g. a decision, a deploy, a preference with lasting relevance). Use instead of only stating it in prose when the fact should survive this session. Returns immediately once the fact is queued; extraction into graph facts happens in the background and survives provider outages, so never retry a successful call.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"fact": map[string]any{"type": "string", "description": "The fact to remember, in plain prose (e.g. 'book-system deploys to hermes-mini via ssh')."},
				"entities": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Optional glossary hints — canonical entity names mentioned in the fact — to bias entity resolution during extraction.",
				},
			},
			"required": []string{"fact"},
		}),
	},
}

// callMemoryQuery handles memory-domain tool calls that forward directly to
// the daemon. Each memory tool has its own argument shape, so we parse a
// superset struct and forward only the fields present — the daemon validates
// which fields are actually required per RPC.
func (s *Server) callMemoryQuery(ctx context.Context, id json.RawMessage, toolName, rpcMethod string, rawArgs json.RawMessage) {
	start := nowUTC()

	var args struct {
		Query    string   `json:"query"`
		AsOf     string   `json:"as_of"`
		Limit    int      `json:"limit"`
		From     string   `json:"from"`
		To       string   `json:"to"`
		Entity   string   `json:"entity"`
		Fact     string   `json:"fact"`
		Entities []string `json:"entities"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		s.writeToolError(id, "invalid arguments: "+err.Error())
		return
	}

	client, err := s.dial()
	if err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, LatencyMs: time.Since(start).Milliseconds(), Error: "dial failed"})
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()

	// Build the RPC params with whatever fields the caller supplied — the
	// daemon ignores fields its handler doesn't read.
	params := map[string]any{}
	if args.Query != "" {
		params["query"] = args.Query
	}
	if args.AsOf != "" {
		params["as_of"] = args.AsOf
	}
	if args.Limit > 0 {
		params["limit"] = args.Limit
	}
	if args.From != "" {
		params["from"] = args.From
	}
	if args.To != "" {
		params["to"] = args.To
	}
	if args.Entity != "" {
		params["entity"] = args.Entity
	}
	if args.Fact != "" {
		params["fact"] = args.Fact
	}
	if len(args.Entities) > 0 {
		params["entities"] = args.Entities
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
