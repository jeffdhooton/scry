package extract

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// fakeExtractor is a simple in-memory Extractor for tests in this package
// and for later tasks that need a stand-in for Haiku. Keep it minimal.
type fakeExtractor struct {
	result Result
	err    error
}

func (f *fakeExtractor) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (Result, error) {
	return f.result, f.err
}

var _ Extractor = (*fakeExtractor)(nil)

func TestParseResult(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		raw := `{
			"episode_summary": "Deployed book-system to the hermes mini.",
			"entities": [
				{"name": "book-system", "type": "service", "description": "Book pipeline", "aliases": []}
			],
			"facts": [
				{"src": "book-system", "relation": "deployed_on", "dst": "hermes-mini", "fact": "book-system runs on hermes mini", "confidence": 0.9}
			]
		}`
		result, err := ParseResult(raw)
		if err != nil {
			t.Fatalf("ParseResult() error = %v, want nil", err)
		}
		if result.EpisodeSummary != "Deployed book-system to the hermes mini." {
			t.Errorf("EpisodeSummary = %q", result.EpisodeSummary)
		}
		if len(result.Entities) != 1 || result.Entities[0].Name != "book-system" {
			t.Errorf("Entities = %+v", result.Entities)
		}
		if len(result.Facts) != 1 || result.Facts[0].Confidence != 0.9 {
			t.Errorf("Facts = %+v", result.Facts)
		}
	})

	t.Run("fenced with json tag", func(t *testing.T) {
		raw := "```json\n" + `{
			"episode_summary": "ok",
			"entities": [],
			"facts": []
		}` + "\n```"
		result, err := ParseResult(raw)
		if err != nil {
			t.Fatalf("ParseResult() error = %v, want nil", err)
		}
		if result.EpisodeSummary != "ok" {
			t.Errorf("EpisodeSummary = %q", result.EpisodeSummary)
		}
	})

	t.Run("fenced without json tag", func(t *testing.T) {
		raw := "```\n" + `{
			"episode_summary": "ok",
			"entities": [],
			"facts": []
		}` + "\n```"
		result, err := ParseResult(raw)
		if err != nil {
			t.Fatalf("ParseResult() error = %v, want nil", err)
		}
		if result.EpisodeSummary != "ok" {
			t.Errorf("EpisodeSummary = %q", result.EpisodeSummary)
		}
	})

	t.Run("junk returns error", func(t *testing.T) {
		_, err := ParseResult("not json at all")
		if err == nil {
			t.Fatal("ParseResult() error = nil, want error")
		}
	})

	t.Run("missing episode_summary returns error", func(t *testing.T) {
		raw := `{"entities": [], "facts": []}`
		_, err := ParseResult(raw)
		if err == nil {
			t.Fatal("ParseResult() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "episode_summary") {
			t.Errorf("error = %v, want mention of episode_summary", err)
		}
	})

	t.Run("empty episode_summary returns error", func(t *testing.T) {
		raw := `{"episode_summary": "   ", "entities": [], "facts": []}`
		_, err := ParseResult(raw)
		if err == nil {
			t.Fatal("ParseResult() error = nil, want error")
		}
	})

	// Rejecting the whole result cost the entire fact. A sentence naming an
	// AI model reliably produced type "model", every step of the chain
	// agreed, and the write was dropped rather than stored under a slightly
	// wrong label.
	t.Run("unknown entity type folds into concept instead of failing", func(t *testing.T) {
		raw := `{
			"episode_summary": "ok",
			"entities": [
				{"name": "x", "type": "spaceship", "description": "d"},
				{"name": "y", "type": "machine", "description": "d"}
			],
			"facts": []
		}`
		got, err := ParseResult(raw)
		if err != nil {
			t.Fatalf("ParseResult() error = %v, want the fact kept", err)
		}
		if len(got.Entities) != 2 {
			t.Fatalf("got %d entities, want 2", len(got.Entities))
		}
		if got.Entities[0].Type != "concept" {
			t.Errorf("invented type = %q, want %q", got.Entities[0].Type, "concept")
		}
		if got.Entities[1].Type != "machine" {
			t.Errorf("valid type = %q, want it left alone", got.Entities[1].Type)
		}
	})

	t.Run("missing confidence returns error", func(t *testing.T) {
		raw := `{
			"episode_summary": "ok",
			"entities": [],
			"facts": [{"src": "a", "relation": "uses", "dst": "b", "fact": "a uses b"}]
		}`
		_, err := ParseResult(raw)
		if err == nil {
			t.Fatal("ParseResult() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "confidence") {
			t.Errorf("error = %v, want mention of confidence", err)
		}
	})

	t.Run("prose-wrapped JSON falls back to outermost object", func(t *testing.T) {
		raw := "I extracted the following:\n" + `{
			"episode_summary": "ok",
			"entities": [],
			"facts": []
		}`
		result, err := ParseResult(raw)
		if err != nil {
			t.Fatalf("ParseResult() error = %v, want nil", err)
		}
		if result.EpisodeSummary != "ok" {
			t.Errorf("EpisodeSummary = %q", result.EpisodeSummary)
		}
	})

	t.Run("markdown-header-wrapped JSON falls back to outermost object", func(t *testing.T) {
		raw := "# Result\n" + `{
			"episode_summary": "ok",
			"entities": [],
			"facts": []
		}` + "\ntrailing notes here"
		result, err := ParseResult(raw)
		if err != nil {
			t.Fatalf("ParseResult() error = %v, want nil", err)
		}
		if result.EpisodeSummary != "ok" {
			t.Errorf("EpisodeSummary = %q", result.EpisodeSummary)
		}
	})

	t.Run("truncated JSON still fails with ErrParse-eligible error", func(t *testing.T) {
		raw := `{
			"episode_summary": "ok",
			"entities": [],
			"facts": [{"src": "a", "relation": "uses", "dst": "b", "fact": "a uses b", "confidence": 0.5`
		_, err := ParseResult(raw)
		if err == nil {
			t.Fatal("ParseResult() error = nil, want error for truncated JSON")
		}
	})

	t.Run("nested braces inside strings are handled by the first-to-last brace rule", func(t *testing.T) {
		// The wrapping prose has no braces of its own, so the first '{' and
		// last '}' in the whole text are exactly the JSON object's own
		// delimiters — even though the object contains a string value that
		// itself holds literal '{' and '}' characters (which a naive
		// brace-counting matcher could trip over, but a plain first/last
		// substring cut does not).
		raw := "Here you go:\n" + `{
			"episode_summary": "Discussed the shape {a: 1, b: {c: 2}} in chat.",
			"entities": [],
			"facts": []
		}` + "\nHope that helps."
		result, err := ParseResult(raw)
		if err != nil {
			t.Fatalf("ParseResult() error = %v, want nil", err)
		}
		wantSummary := "Discussed the shape {a: 1, b: {c: 2}} in chat."
		if result.EpisodeSummary != wantSummary {
			t.Errorf("EpisodeSummary = %q, want %q", result.EpisodeSummary, wantSummary)
		}
	})
}

func TestBuildMessages(t *testing.T) {
	occurredAt := time.Date(2026, 7, 28, 12, 30, 0, 0, time.UTC)
	ep := distill.RawEpisode{
		Source:     "claude-session",
		Text:       "User: deploy book-system to the hermes mini\nAssistant: done",
		OccurredAt: occurredAt,
		Cwd:        "/Users/jeff/workspace/book-system",
	}
	glossary := []string{"book-system: also known as authorclaw", "hermes-mini: jclaws-mac-mini"}

	system, userMsg := buildMessages(ep, glossary)

	if len(system) != 2 {
		t.Fatalf("len(system) = %d, want 2", len(system))
	}

	if system[0].Text != SystemPrompt {
		t.Errorf("system[0].Text != SystemPrompt")
	}
	if system[0].CacheControl.Type != "ephemeral" {
		t.Errorf("system[0].CacheControl.Type = %q, want %q", system[0].CacheControl.Type, "ephemeral")
	}

	wantGlossaryBlock := "KNOWN ENTITIES (canonical name: aliases):\n" + strings.Join(glossary, "\n")
	if system[1].Text != wantGlossaryBlock {
		t.Errorf("system[1].Text = %q, want %q", system[1].Text, wantGlossaryBlock)
	}
	if system[1].CacheControl.Type == "ephemeral" {
		t.Errorf("system[1].CacheControl should not be set (glossary block is volatile)")
	}

	if len(userMsg.Content) != 1 || userMsg.Content[0].OfText == nil {
		t.Fatalf("userMsg.Content = %+v, want a single text block", userMsg.Content)
	}
	wantUserText := "reference_time: " + occurredAt.Format(time.RFC3339) + "\ncwd: " + ep.Cwd + "\n\n" + ep.Text
	if got := userMsg.Content[0].OfText.Text; got != wantUserText {
		t.Errorf("userMsg text = %q, want %q", got, wantUserText)
	}
}
