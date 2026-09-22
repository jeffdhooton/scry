package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

var reviewIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$`)
var reviewToolDefinitions = buildReviewTools()

func buildReviewTools() []tool {
	defs := []tool{}
	for _, entry := range []struct{ action, description string }{
		{"status", "Inspect local background review configuration, activity and usage without model inference."},
		{"preview", "Capture local repository change evidence without model inference. Returns the captured snapshot identity; no findings are produced."},
		{"run", "Explicitly queue asynchronous paid model inference for an allowlisted, configured repository within configured request and token caps. Returns queue state. Findings are provisional and tied to a snapshot; check freshness before acting. Never edits code or admits findings into memory."},
		{"list", "List local provisional reviews and their snapshot freshness without model inference. Errors or stale findings never imply the current code has no issues."},
		{"get", "Retrieve one local review's provisional findings, captured evidence, snapshot identity and freshness without model inference. Stale findings describe the captured snapshot, not necessarily the current code."},
	} {
		props := map[string]any{}
		required := []string{}
		switch entry.action {
		case "preview", "run", "list":
			props["repo"] = map[string]any{"type": "string", "description": "Repository path, made absolute; defaults to the MCP process working directory. Run requires an explicitly configured repository."}
		case "get":
			props["id"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 200, "pattern": reviewIDPattern.String(), "description": "Exact review record ID returned by run or list."}
			required = append(required, "id")
		}
		defs = append(defs, tool{Name: "scry_review_" + entry.action, Description: entry.description,
			InputSchema: mustMarshal(map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}),
		})
	}
	return defs
}

func (s *Server) callReview(ctx context.Context, id json.RawMessage, action string, args json.RawMessage) {
	params, err := reviewParams(action, args)
	if err != nil {
		s.writeToolError(id, err.Error())
		return
	}
	client, err := s.dial()
	if err != nil {
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()
	var result json.RawMessage
	// One dispatch only: a lost run response must not trigger another paid job.
	if err := client.Call(ctx, "review."+action, params, &result); err != nil {
		s.writeToolError(id, err.Error())
		return
	}
	s.writeToolResult(id, string(result), false)
}

func reviewParams(action string, raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, fmt.Errorf("review arguments must be an object")
	}
	field := ""
	switch action {
	case "get":
		field = "id"
	case "preview", "run", "list":
		field = "repo"
	}
	for name := range fields {
		if field == "" || name != field {
			return nil, fmt.Errorf("unexpected review argument %q", name)
		}
	}
	params := map[string]string{}
	if field == "" {
		return params, nil
	}
	var value string
	if supplied, ok := fields[field]; ok {
		if string(supplied) == "null" {
			return nil, fmt.Errorf("%s must be a string", field)
		}
		if err := json.Unmarshal(supplied, &value); err != nil {
			return nil, fmt.Errorf("%s must be a string", field)
		}
	}
	if field == "id" {
		if !reviewIDPattern.MatchString(value) {
			return nil, fmt.Errorf("review ID must be 1–200 ASCII letters/digits/._:- and start with a letter or digit")
		}
	} else {
		var err error
		value, err = resolveRepo(value)
		if err != nil {
			return nil, err
		}
	}
	params[field] = value
	return params, nil
}
