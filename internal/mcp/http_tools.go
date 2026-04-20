package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var httpToolDefinitions = []tool{
	{
		Name:        "scry_requests",
		Description: "List recent HTTP requests captured by the scry proxy. Filterable by path substring, method, and status code range. Returns compact summaries with ID, method, path, status, duration, and timestamp. Use to see what API calls are happening at runtime. Requires the proxy to be running (`scry proxy start`).",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":       map[string]any{"type": "string", "description": "Filter by path substring (e.g. '/api/users')."},
				"method":     map[string]any{"type": "string", "description": "Filter by HTTP method (GET, POST, etc.)."},
				"status_min": map[string]any{"type": "integer", "description": "Minimum status code (e.g. 400 for errors only)."},
				"status_max": map[string]any{"type": "integer", "description": "Maximum status code."},
				"limit":      map[string]any{"type": "integer", "description": "Max results (default 20)."},
			},
		}),
	},
	{
		Name:        "scry_request",
		Description: "Get full details for a single captured HTTP request by ID. Returns complete headers, request/response bodies, timing, and status. Use after scry_requests to drill into a specific request.",
		InputSchema: mustMarshal(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "string", "description": "Request ID from scry_requests output."},
			},
			"required": []string{"id"},
		}),
	},
	{
		Name:        "scry_http_status",
		Description: "Check whether the scry HTTP proxy is running, which port it's on, what target it's forwarding to, and how many requests have been captured.",
		InputSchema: mustMarshal(map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}),
	},
}

func (s *Server) callHTTPQuery(ctx context.Context, id json.RawMessage, toolName, rpcMethod string, rawArgs json.RawMessage) {
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
		s.writeToolError(id, fmt.Sprintf("scry %s: %s", rpcMethod, err.Error()))
		return
	}

	result := prettyJSON(raw)
	if toolName == "scry_request" {
		result = formatRequestForAgent(raw)
	}

	logCall(callLogEntry{Timestamp: start.Format(time.RFC3339), Tool: toolName, LatencyMs: time.Since(start).Milliseconds()})
	s.writeToolResult(id, result, false)
}

func formatRequestForAgent(raw json.RawMessage) string {
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		return prettyJSON(raw)
	}

	if headers, ok := req["response_headers"].(map[string]any); ok {
		if ct, ok := headers["Content-Type"]; ok {
			if ctStr, ok := ct.(string); ok && isBinaryContent(ctStr) {
				if body, ok := req["response_body"].(string); ok {
					req["response_body"] = fmt.Sprintf("[binary: %d bytes]", len(body))
				}
			}
		}
	}

	out, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return prettyJSON(raw)
	}
	return string(out)
}

func isBinaryContent(ct string) bool {
	lower := strings.ToLower(ct)
	for _, prefix := range []string{
		"image/", "audio/", "video/", "application/octet-stream",
		"application/pdf", "application/zip", "application/gzip",
	} {
		if strings.HasPrefix(lower, prefix) || strings.Contains(lower, prefix) {
			return true
		}
	}
	return false
}
