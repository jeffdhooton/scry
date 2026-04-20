package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var schemaToolDefinitions = []tool{
	{
		Name:        "scry_describe",
		Description: "Describe a database table: columns with types, nullability, defaults, primary keys, indexes, foreign keys, and which tables reference it. Use instead of reading migrations or running DESCRIBE queries. Requires a schema index — run `scry init --schema` first.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table":   map[string]any{"type": "string", "description": "Table name to describe."},
				"project": map[string]any{"type": "string", "description": "Absolute path to the project root. Defaults to cwd."},
			},
			"required": []string{"table"},
		}),
	},
	{
		Name:        "scry_relations",
		Description: "Show the foreign key graph for a table: outgoing FKs (what this table references) and incoming FKs (what references this table). Use to understand data model relationships without reading migrations.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table":   map[string]any{"type": "string", "description": "Table name to inspect."},
				"project": map[string]any{"type": "string", "description": "Absolute path to the project root. Defaults to cwd."},
			},
			"required": []string{"table"},
		}),
	},
	{
		Name:        "scry_schema_search",
		Description: "Search for tables and columns by name substring. Use when you know part of a name but not the exact table — e.g. 'user' finds tables like 'users', 'user_roles' and columns like 'user_id' across all tables.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":   map[string]any{"type": "string", "description": "Search term (case-insensitive substring match)."},
				"project": map[string]any{"type": "string", "description": "Absolute path to the project root. Defaults to cwd."},
			},
			"required": []string{"query"},
		}),
	},
	{
		Name:        "scry_enums",
		Description: "List enum column values. Returns all enum/set columns, or filter by table and column. Use to see valid values for status fields, type columns, etc.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"table":   map[string]any{"type": "string", "description": "Filter by table name (optional)."},
				"column":  map[string]any{"type": "string", "description": "Filter by column name (optional, requires table)."},
				"project": map[string]any{"type": "string", "description": "Absolute path to the project root. Defaults to cwd."},
			},
		}),
	},
}

func (s *Server) callSchemaQuery(ctx context.Context, id json.RawMessage, toolName, rpcMethod string, rawArgs json.RawMessage) {
	start := nowUTC()

	var args struct {
		Table   string `json:"table"`
		Query   string `json:"query"`
		Column  string `json:"column"`
		Project string `json:"project"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		s.writeToolError(id, "invalid arguments: "+err.Error())
		return
	}

	project, err := resolveRepo(args.Project)
	if err != nil {
		s.writeToolError(id, err.Error())
		return
	}

	client, err := s.dial()
	if err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, Repo: project, LatencyMs: time.Since(start).Milliseconds(), Error: "dial failed"})
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()

	params := map[string]any{"project": project}
	if args.Table != "" {
		params["table"] = args.Table
	}
	if args.Query != "" {
		params["query"] = args.Query
	}
	if args.Column != "" {
		params["column"] = args.Column
	}

	var raw json.RawMessage
	if err := client.Call(ctx, rpcMethod, params, &raw); err != nil {
		logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, Repo: project, LatencyMs: time.Since(start).Milliseconds(), Error: err.Error()})
		s.writeToolError(id, fmt.Sprintf("scry %s: %s", rpcMethod, err.Error()))
		return
	}

	logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, Repo: project, LatencyMs: time.Since(start).Milliseconds()})
	s.writeToolResult(id, prettyJSON(raw), false)
}
