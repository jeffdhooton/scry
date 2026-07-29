// Package extract turns a distilled episode into a small knowledge-graph
// JSON document via an LLM, behind the Extractor interface. The concrete
// implementation (Haiku) is a thin wrapper around the Anthropic Messages
// API with a stable, cache-control-eligible system prompt.
package extract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// ErrParse is wrapped into whatever Extract (or a BatchRunner result) returns
// when the model's output could not be turned into a valid Result — after
// Haiku.Extract's one corrective retry, or for a batch item that "succeeded"
// at the API level but whose text didn't parse. Callers (the ingest
// pipeline, backfill) use errors.Is(err, ErrParse) to distinguish this
// content-level failure — safe to skip just the one episode and move on —
// from context/transport failures (a canceled context, a request that never
// got a response), which must abort the whole run instead of silently
// treating unprocessed episodes as done.
var ErrParse = errors.New("extract: parse failed")

// Result is the knowledge graph extracted from a single episode.
type Result struct {
	EpisodeSummary string `json:"episode_summary"`
	Entities       []Ent  `json:"entities"`
	Facts          []Fct  `json:"facts"`
}

// Ent is one entity mentioned in an episode.
type Ent struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases,omitempty"`
}

// Fct is one fact relating two entities.
type Fct struct {
	Src        string  `json:"src"`
	Relation   string  `json:"relation"`
	Dst        string  `json:"dst"`
	Fact       string  `json:"fact"`
	ValidFrom  string  `json:"valid_from,omitempty"` // RFC3339 date or empty
	Confidence float64 `json:"confidence"`
	Supersedes *SupRef `json:"supersedes,omitempty"`
}

// SupRef names an earlier fact that Fct supersedes.
type SupRef struct{ Src, Relation, Dst string }

// Extractor turns a distilled episode into a Result. glossary is the list
// of known entity names (with aliases) to prefer when resolving src/dst.
type Extractor interface {
	Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (Result, error)
}

// allowedEntityTypes mirrors the "type is one of: ..." rule in SystemPrompt.
var allowedEntityTypes = map[string]bool{
	"project":  true,
	"service":  true,
	"machine":  true,
	"tool":     true,
	"person":   true,
	"decision": true,
	"runbook":  true,
	"concept":  true,
}

// ParseResult unmarshals raw model output into a Result. It tolerates a
// ```json ... ``` (or bare ``` ... ```) fence around the JSON object, and
// validates the parsed structure: episode_summary must be non-empty, every
// entity's type must be in the allowed set, and every fact must carry a
// confidence value. Errors are descriptive enough to feed back to the model
// for a corrective retry.
//
// If the fence-stripped text still doesn't parse (e.g. the model wrote prose
// before/after the JSON, or wrapped it in markdown headers instead of a code
// fence), ParseResult falls back to extracting the outermost JSON object —
// the substring from the first '{' to the last '}' inclusive — and retries
// unmarshal+validation once against that substring. This rescues
// prose-wrapped and header-wrapped JSON; truncated JSON (no closing '}') is
// correctly left to fail, since the payload is genuinely incomplete. If the
// fallback also fails, the original error is returned so the caller sees the
// first, more diagnostic failure.
func ParseResult(raw string) (Result, error) {
	text := stripFences(raw)

	result, err := parseResultStrict(text)
	if err == nil {
		return result, nil
	}

	if start, end := strings.IndexByte(text, '{'), strings.LastIndexByte(text, '}'); start >= 0 && end > start {
		if fallback, fallbackErr := parseResultStrict(text[start : end+1]); fallbackErr == nil {
			return fallback, nil
		}
	}

	return Result{}, err
}

// parseResultStrict unmarshals text (already fence-stripped, or a
// first-'{'-to-last-'}' substring) into a Result and validates it. It does
// not attempt any further recovery.
func parseResultStrict(text string) (Result, error) {
	var result Result
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return Result{}, fmt.Errorf("extract: invalid JSON: %w", err)
	}

	// The typed unmarshal above can't distinguish an omitted field from an
	// explicit zero value, so re-parse loosely to check presence.
	var loose struct {
		EpisodeSummary *string          `json:"episode_summary"`
		Entities       []map[string]any `json:"entities"`
		Facts          []map[string]any `json:"facts"`
	}
	if err := json.Unmarshal([]byte(text), &loose); err != nil {
		return Result{}, fmt.Errorf("extract: invalid JSON: %w", err)
	}

	if loose.EpisodeSummary == nil || strings.TrimSpace(*loose.EpisodeSummary) == "" {
		return Result{}, fmt.Errorf("extract: episode_summary is required and must be non-empty")
	}

	for i, e := range loose.Entities {
		typ, _ := e["type"].(string)
		if !allowedEntityTypes[typ] {
			return Result{}, fmt.Errorf("extract: entities[%d] has unknown type %q (want one of project|service|machine|tool|person|decision|runbook|concept)", i, typ)
		}
	}

	for i, f := range loose.Facts {
		if _, ok := f["confidence"]; !ok {
			return Result{}, fmt.Errorf("extract: facts[%d] is missing confidence", i)
		}
	}

	return result, nil
}

// stripFences removes a surrounding ```json ... ``` or ``` ... ``` fence,
// if present, and trims surrounding whitespace. Text without a fence is
// returned unchanged (after trimming).
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	rest := s[3:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		// Everything up to the first newline is an optional language tag
		// (e.g. "json") or blank — never JSON content itself.
		rest = rest[nl+1:]
	}
	rest = strings.TrimSpace(rest)
	rest = strings.TrimSuffix(rest, "```")
	return strings.TrimSpace(rest)
}
