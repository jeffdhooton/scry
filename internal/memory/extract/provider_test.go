package extract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

func TestProviderFromEnvPrefersMemoryKey(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "memory-key")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")

	if got := ProviderFromEnv().APIKey; got != "memory-key" {
		t.Errorf("APIKey = %q, want memory-key", got)
	}
}

func TestProviderFromEnvFallsBackToAnthropicKey(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")

	if got := ProviderFromEnv().APIKey; got != "anthropic-key" {
		t.Errorf("APIKey = %q, want anthropic-key", got)
	}
}

func TestProviderFromEnvReadsModelAndBaseURL(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "k")
	t.Setenv("SCRY_MEMORY_MODEL", "deepseek-chat")
	t.Setenv("SCRY_MEMORY_BASE_URL", "https://api.deepseek.com/anthropic")

	p := ProviderFromEnv()
	if p.Model != "deepseek-chat" {
		t.Errorf("Model = %q, want deepseek-chat", p.Model)
	}
	if p.BaseURL != "https://api.deepseek.com/anthropic" {
		t.Errorf("BaseURL = %q", p.BaseURL)
	}
}

func TestProviderValidate(t *testing.T) {
	tests := []struct {
		name    string
		p       Provider
		wantErr bool
	}{
		{"anthropic default model", Provider{APIKey: "k"}, false},
		{"anthropic explicit model", Provider{APIKey: "k", Model: "claude-opus-4"}, false},
		{"custom endpoint with model", Provider{APIKey: "k", Model: "deepseek-chat", BaseURL: "https://x/anthropic"}, false},
		{"custom endpoint without model", Provider{APIKey: "k", BaseURL: "https://x/anthropic"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProviderBatched(t *testing.T) {
	if !(Provider{APIKey: "k"}).Batched() {
		t.Error("Anthropic provider should support the Batches API")
	}
	if (Provider{APIKey: "k", Model: "m", BaseURL: "https://x/anthropic"}).Batched() {
		t.Error("custom endpoint must not claim Batches API support")
	}
}

// TestHaikuHonorsBaseURL drives a real Extract against a stub server to pin
// down two things a unit test on the struct can't: that a custom base URL is
// actually used, and that a base URL carrying a path prefix (DeepSeek's
// /anthropic) resolves to <prefix>/v1/messages rather than /v1/messages at
// the host root.
func TestHaikuHonorsBaseURL(t *testing.T) {
	var gotPath, gotModel, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("x-api-key")
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		if strings.Contains(string(body), `"deepseek-chat"`) {
			gotModel = "deepseek-chat"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m","type":"message","role":"assistant","model":"deepseek-chat",` +
			`"content":[{"type":"text","text":"{\"episode_summary\":\"s\",\"entities\":[],\"facts\":[]}"}],` +
			`"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer srv.Close()

	h := NewHaiku(Provider{APIKey: "test-key", Model: "deepseek-chat", BaseURL: srv.URL + "/anthropic"})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.Extract(ctx, distill.RawEpisode{
		Source:     "claude-session",
		Text:       "hello",
		OccurredAt: time.Now(),
		Cwd:        "/tmp",
	}, []string{"foo: bar"})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if gotPath != "/anthropic/v1/messages" {
		t.Errorf("request path = %q, want /anthropic/v1/messages", gotPath)
	}
	if gotModel != "deepseek-chat" {
		t.Errorf("model in request body = %q, want deepseek-chat", gotModel)
	}
	if gotAuth != "test-key" {
		t.Errorf("x-api-key = %q, want test-key", gotAuth)
	}
	if res.EpisodeSummary != "s" {
		t.Errorf("EpisodeSummary = %q, want s", res.EpisodeSummary)
	}
}
