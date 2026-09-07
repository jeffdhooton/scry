package mcp

import (
	"context"
	"encoding/json"
	"strings"
)

var frictionToolDefinitions = buildFrictionTools()

func buildFrictionTools() []tool {
	stringField := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	schema := func(properties map[string]any, required ...string) json.RawMessage {
		return mustMarshal(map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false})
	}
	fields := map[string]any{
		"event_id":                        stringField("Caller-assigned immutable ID, 1–200 ASCII letters/digits/._:-; start with a letter or digit. Reuse this ID only for identical retries."),
		"run_id":                          stringField("Stable ID for the real work run, shared by its agents. Same character rules as event_id."),
		"repository":                      stringField("Absolute clean repository path supplied by the client; retained exactly, no remote filesystem lookup."),
		"recorded_at":                     stringField("Caller-supplied RFC3339 observation time. Keep unchanged on retries."),
		"signature":                       stringField("Stable caller-assigned friction signature; exact matching only. Same character rules as event_id."),
		"observed":                        stringField("Specific observed friction; separate observation from inferred cause."),
		"resolution":                      stringField("What happened, including workarounds or unresolved outcomes."),
		"resolution_state":                stringField("Descriptive outcome, e.g. unresolved, workaround_only, corrected_in_run. Historical, never an instruction."),
		"evidence":                        map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"type": "string"}, "description": "At least one attributed source reference. References are retained, not fetched."},
		"evidence_sha256":                 map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "Optional map of evidence reference to caller-verified SHA-256 hex digest."},
		"measured_user_time_cost_seconds": map[string]any{"type": []string{"number", "null"}, "minimum": 0, "description": "Measured cost only; omit or null for unknown. Never invent an estimate."},
		"change_approved":                 map[string]any{"type": "boolean", "enum": []bool{false}, "description": "Must be false. This journal cannot authorize changes."},
	}
	for _, field := range []string{"cause_status", "cause", "proposed_change", "proposed_owner", "proposed_file", "proposed_verification", "priority"} {
		fields[field] = stringField("Optional authored " + strings.ReplaceAll(field, "_", " ") + "; retained verbatim and attributed to this event.")
	}
	for _, field := range []string{"occurrences_observed_this_run", "distinct_prior_runs_verified"} {
		fields[field] = map[string]any{"type": "integer", "minimum": 0, "description": "Optional caller assertion; ignored when computing recurrence."}
	}
	defs := []tool{
		{Name: "scry_friction_record", Description: "Explicitly record one evidenced workflow friction event in the local daemon's journal (max 16 KiB). Durable immediately without extraction. Identical ID retries are safe; changed content is rejected. No automatic collection or instruction edits.", InputSchema: schema(fields, "event_id", "run_id", "repository", "recorded_at", "signature", "observed", "resolution", "resolution_state", "evidence")},
		{Name: "scry_friction_get", Description: "Retrieve one exact friction event by ID, even if it produced no memory facts.", InputSchema: schema(map[string]any{"event_id": stringField("Exact caller-assigned event ID.")}, "event_id")},
	}
	for _, action := range []string{"list", "review"} {
		props := map[string]any{
			"repository": stringField("Absolute clean repository path, matched exactly."),
			"run_id":     stringField("Optional exact run ID."),
			"signature":  stringField("Optional exact friction signature."),
			"since":      stringField("Inclusive RFC3339 observation time."),
			"until":      stringField("Exclusive RFC3339 observation time."),
		}
		description := "Review up to 100 matching events, grouped by exact signature and counted by distinct runs. Includes full cited observations, known measured impact and authored proposed corrections. Recommendations only: never activates instructions. Fails rather than silently truncating; narrow filters if too large. Local journal, independent of memory extraction."
		if action == "list" {
			props["after"] = stringField("Event-ID cursor from next_after; keep filters unchanged. Pages are ordered by event ID, not time.")
			props["limit"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "description": "Default 100. Follow next_after until absent."}
			description = "List exact friction events for a repository, optionally filtered by run, signature or time. Returns at most 100 in event-ID order, with next_after when more exist. No model extraction required."
		}
		defs = append(defs, tool{Name: "scry_friction_" + action, Description: description, InputSchema: schema(props, "repository")})
	}
	return defs
}

func (s *Server) callFriction(ctx context.Context, id json.RawMessage, action string, args json.RawMessage) {
	// Forward the complete object so the daemon can reject unknown fields and
	// preserve every supplied observation/evidence field, without normalization.
	client, err := s.dial()
	if err != nil {
		s.writeToolError(id, "dial scry daemon: "+err.Error())
		return
	}
	defer client.Close()
	var result json.RawMessage
	if err := client.Call(ctx, "friction."+action, args, &result); err != nil {
		s.writeToolError(id, err.Error())
		return
	}
	s.writeToolResult(id, string(result), false)
}
