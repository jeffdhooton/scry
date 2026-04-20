package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var graphToolDefinitions = []tool{
	{
		Name:        "scry_graph_query",
		Description: "Search the unified code graph for nodes matching a query. Returns matching functions, classes, tables, files, authors, and endpoints with their connections and degree. Use to explore what exists in the graph before running scry_graph_path.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Search term (case-insensitive name substring)."},
				"repo":  map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"query"},
		}),
	},
	{
		Name:        "scry_graph_path",
		Description: "Find the shortest path between two nodes in the unified graph. Connects across domains — e.g. from a function to a database table, or from an author to an endpoint. Returns the path with node names, types, and edge types.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"from": map[string]any{"type": "string", "description": "Source node name (searched by substring)."},
				"to":   map[string]any{"type": "string", "description": "Target node name (searched by substring)."},
				"repo": map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"from", "to"},
		}),
	},
	{
		Name:        "scry_graph_report",
		Description: "Get the pre-computed graph report: god nodes (highest-degree), surprising cross-domain edges, architectural communities, and suggested queries. Read this first when exploring a new repo to understand its structure before diving into code.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repo": map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
		}),
	},
}

func (s *Server) callGraphQuery(ctx context.Context, id json.RawMessage, toolName, rpcMethod string, rawArgs json.RawMessage) {
	start := nowUTC()

	var args struct {
		Query string `json:"query"`
		From  string `json:"from"`
		To    string `json:"to"`
		Repo  string `json:"repo"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		s.writeToolError(id, "invalid arguments: "+err.Error())
		return
	}

	repo, err := resolveRepo(args.Repo)
	if err != nil {
		s.writeToolError(id, err.Error())
		return
	}

	client, err := s.dial()
	if err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, Repo: repo, LatencyMs: time.Since(start).Milliseconds(), Error: "dial failed"})
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()

	params := map[string]any{"repo": repo}
	if args.Query != "" {
		params["query"] = args.Query
	}
	if args.From != "" {
		params["from"] = args.From
	}
	if args.To != "" {
		params["to"] = args.To
	}

	var raw json.RawMessage
	if err := client.Call(ctx, rpcMethod, params, &raw); err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, Repo: repo, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error()})
		s.writeToolError(id, fmt.Sprintf("scry %s: %s", rpcMethod, err.Error()))
		return
	}

	logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, Repo: repo, LatencyMs: time.Since(start).Milliseconds()})
	s.writeToolResult(id, prettyJSON(raw), false)
}
