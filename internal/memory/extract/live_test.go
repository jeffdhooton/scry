package extract

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// TestHaikuLive hits the real Anthropic API. Gated behind SCRY_LIVE_TEST=1
// plus an API key (SCRY_MEMORY_API_KEY or ANTHROPIC_API_KEY) so it never
// runs in the normal test suite or CI by accident.
func TestHaikuLive(t *testing.T) {
	if os.Getenv("SCRY_LIVE_TEST") != "1" {
		t.Skip("set SCRY_LIVE_TEST=1 to run live Haiku extraction tests")
	}
	provider := ProviderFromEnv()
	if provider.APIKey == "" {
		t.Skip("set SCRY_MEMORY_API_KEY or ANTHROPIC_API_KEY to run live Haiku extraction tests")
	}
	if err := provider.Validate(); err != nil {
		t.Fatalf("provider: %v", err)
	}

	h := NewHaiku(provider)

	ep := distill.RawEpisode{
		Source: "claude-session",
		Text: "User: let's deploy book-system to the hermes mini tonight.\n" +
			"Assistant: sure, I'll ssh into the hermes mini and pull the latest book-system release, then restart the service.\n" +
			"User: great, ping me once it's live.\n" +
			"Assistant: deployed book-system to the hermes mini and restarted the service; it's live now.",
		OccurredAt: time.Now(),
		Cwd:        "/Users/jeff/workspace/book-system",
	}
	glossary := []string{"hermes-mini: jclaws-mac-mini", "book-system: canonical book pipeline"}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := h.Extract(ctx, ep, glossary)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if len(result.Entities) < 1 {
		t.Errorf("Entities = %+v, want at least 1", result.Entities)
	}
	if len(result.Facts) < 1 {
		t.Errorf("Facts = %+v, want at least 1", result.Facts)
	}
}
