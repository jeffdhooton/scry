package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var gitToolDefinitions = []tool{
	{
		Name:        "scry_blame",
		Description: "Pre-indexed blame data for any file or line range. Returns author, commit hash, date, and commit message per line. Use instead of `git blame` for structured, instant lookups.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file":       map[string]any{"type": "string", "description": "Relative path to the file within the repo."},
				"start_line": map[string]any{"type": "integer", "description": "Start line (inclusive, 1-based). Omit to get all lines."},
				"end_line":   map[string]any{"type": "integer", "description": "End line (inclusive). Omit to get all lines."},
				"repo":       map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"file"},
		}),
	},
	{
		Name:        "scry_history",
		Description: "Recent commits affecting a file, with full messages and diff stats. Repo-wide if no file specified. Use instead of `git log` for structured commit data.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file":  map[string]any{"type": "string", "description": "Relative path to filter commits by. Omit for repo-wide history."},
				"limit": map[string]any{"type": "integer", "description": "Max commits to return (default 20)."},
				"repo":  map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
		}),
	},
	{
		Name:        "scry_cochange",
		Description: "Files that frequently change alongside a target file, ranked by co-occurrence count. Use to discover hidden coupling — files that always change together likely share a dependency.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file":  map[string]any{"type": "string", "description": "Relative path to the target file."},
				"limit": map[string]any{"type": "integer", "description": "Max results (default 10)."},
				"repo":  map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"file"},
		}),
	},
	{
		Name:        "scry_hotspots",
		Description: "Most-churned files in the repo by commit frequency and line changes. Use to identify high-risk files that change often — good candidates for refactoring or extra test coverage.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{"type": "integer", "description": "Max results (default 20)."},
				"repo":  map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
		}),
	},
	{
		Name:        "scry_contributors",
		Description: "Main authors of a file or the whole repo, ranked by commit count and lines changed. Use for 'who wrote this' or 'who knows this area best'.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file": map[string]any{"type": "string", "description": "Relative path to filter by. Omit for repo-wide contributors."},
				"repo": map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
		}),
	},
	{
		Name:        "scry_intent",
		Description: "The commit message and full context for the code at a specific line — why was this line written? Returns the blame record plus the full commit that introduced it.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file": map[string]any{"type": "string", "description": "Relative path to the file."},
				"line": map[string]any{"type": "integer", "description": "Line number (1-based)."},
				"repo": map[string]any{"type": "string", "description": "Absolute path to the repo root. Defaults to cwd."},
			},
			"required": []string{"file", "line"},
		}),
	},
}

// callGitQuery handles git-domain tool calls that forward directly to the
// daemon. Each git tool has its own argument shape, so we parse and forward
// per-tool.
func (s *Server) callGitQuery(ctx context.Context, id json.RawMessage, toolName, rpcMethod string, rawArgs json.RawMessage) {
	start := nowUTC()

	var args struct {
		File      string `json:"file"`
		Repo      string `json:"repo"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
		Limit     int    `json:"limit"`
		Line      int    `json:"line"`
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

	// Build the RPC params with all fields — the daemon ignores unused ones
	params := map[string]any{"repo": repo}
	if args.File != "" {
		params["file"] = args.File
	}
	if args.StartLine > 0 {
		params["start_line"] = args.StartLine
	}
	if args.EndLine > 0 {
		params["end_line"] = args.EndLine
	}
	if args.Limit > 0 {
		params["limit"] = args.Limit
	}
	if args.Line > 0 {
		params["line"] = args.Line
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
