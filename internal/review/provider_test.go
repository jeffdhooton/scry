package review

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const validReviewJSON = `{"summary":"Changed lookup behavior.","findings":[{"severity":"high","title":"Nil lookup panics","detail":"The caller dereferences a missing result.","evidence_ids":["e1"]}],"test_gaps":["Add a missing-result test."]}`

func providerSnapshot() Snapshot {
	return Snapshot{ID: "snapshot1", Evidence: []Evidence{{ID: "e1", Kind: "diff", Path: "a.go", Content: "untrusted source"}}}
}

func providerConfig(url, protocol string) ProviderConfig {
	return ProviderConfig{Protocol: protocol, BaseURL: url, Model: "explicit-model", APIKeyEnv: "SCRY_REVIEW_TEST_KEY", MaxOutputTokens: 1024}
}

func writeProviderResponse(w http.ResponseWriter, protocol, content, stop string) {
	w.Header().Set("Content-Type", "application/json")
	if protocol == "anthropic" {
		json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]string{"type": "text", "text": content}}, "stop_reason": stop, "usage": map[string]int{"input_tokens": 123, "output_tokens": 45}})
	} else {
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": stop, "message": map[string]string{"role": "assistant", "content": content}}}, "usage": map[string]int{"prompt_tokens": 123, "completion_tokens": 45}})
	}
}

func TestProviderWireProtocols(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "test-secret")
	for _, protocol := range []string{"anthropic", "openai"} {
		t.Run(protocol, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if r.Method != http.MethodPost || body["model"] != "explicit-model" {
					t.Error("unexpected method/model")
				}
				for _, key := range []string{"tools", "functions", "tool_choice"} {
					if _, ok := body[key]; ok {
						t.Errorf("unexpected tool capability %s", key)
					}
				}
				encoded, _ := json.Marshal(body)
				if !strings.Contains(string(encoded), "untrusted") || !strings.Contains(string(encoded), "evidence_ids") {
					t.Error("missing prompt boundaries")
				}
				stop := "stop"
				if protocol == "anthropic" {
					if r.URL.Path != "/v1/messages" || r.Header.Get("X-Api-Key") != "test-secret" || r.Header.Get("Anthropic-Version") != "2023-06-01" || body["max_tokens"] != float64(1024) {
						t.Error("incorrect Messages request")
					}
					stop = "end_turn"
				} else if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-secret" || body["max_completion_tokens"] != float64(1024) {
					t.Error("incorrect chat request")
				}
				writeProviderResponse(w, protocol, validReviewJSON, stop)
			}))
			defer server.Close()
			p, err := NewProvider(providerConfig(server.URL+"/v1", protocol))
			if err != nil {
				t.Fatal(err)
			}
			out, err := p.Review(context.Background(), providerSnapshot())
			if err != nil {
				t.Fatal(err)
			}
			if out.Summary == "" || len(out.Findings) != 1 || !out.Usage.Known || out.Usage.InputTokens != 123 || out.Usage.OutputTokens != 45 {
				t.Fatalf("unexpected review: %+v", out)
			}
		})
	}
}

func TestProviderConfiguration(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret")
	for _, mutate := range []func(*ProviderConfig){
		func(c *ProviderConfig) { c.Protocol = "" }, func(c *ProviderConfig) { c.Protocol = "other" },
		func(c *ProviderConfig) { c.BaseURL = "" }, func(c *ProviderConfig) { c.BaseURL = "http://example.com" },
		func(c *ProviderConfig) { c.BaseURL = "https://user:secret@example.com" }, func(c *ProviderConfig) { c.BaseURL = "https://example.com?key=secret" },
		func(c *ProviderConfig) { c.Model = "" }, func(c *ProviderConfig) { c.APIKeyEnv = "" },
		func(c *ProviderConfig) { c.MaxOutputTokens = 0 }, func(c *ProviderConfig) { c.MaxOutputTokens = 1000000 },
	} {
		c := providerConfig("https://example.com", "anthropic")
		mutate(&c)
		if _, err := NewProvider(c); err == nil {
			t.Errorf("accepted invalid config: %+v", c)
		}
	}
	t.Setenv("SCRY_REVIEW_TEST_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "implicit-secret")
	if _, err := NewProvider(providerConfig("https://example.com", "anthropic")); err == nil {
		t.Error("used implicit key")
	}
}

func TestProviderRejectsInvalidOutput(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret")
	outputs := []string{
		`{}`, `null`, `{"summary":"ok","findings":[],"test_gaps":null}`,
		strings.Replace(validReviewJSON, `"high"`, `"unknown"`, 1),
		strings.Replace(validReviewJSON, `["e1"]`, `["missing"]`, 1),
		strings.Replace(validReviewJSON, `["e1"]`, `[]`, 1),
		strings.Replace(validReviewJSON, `"summary":`, `"extra":1,"summary":`, 1),
		strings.Replace(validReviewJSON, `"summary":`, `"summary":"duplicate","summary":`, 1),
		strings.Replace(validReviewJSON, `"summary":`, `"Summary":`, 1),
		strings.Replace(validReviewJSON, `"title":`, `"Title":`, 1),
		validReviewJSON + ` {}`, "```json\n" + validReviewJSON + "\n```", validReviewJSON[:len(validReviewJSON)-3],
	}
	for _, output := range outputs {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeProviderResponse(w, "anthropic", output, "end_turn")
		}))
		p, err := NewProvider(providerConfig(server.URL, "anthropic"))
		if err != nil {
			t.Fatal(err)
		}
		out, err := p.Review(context.Background(), providerSnapshot())
		server.Close()
		if err == nil || out.Summary != "" {
			t.Errorf("accepted invalid output %s", output)
		}
	}
}

func TestProviderUsageOnFailureAndMissingUsage(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProviderResponse(w, "anthropic", validReviewJSON, "max_tokens")
	}))
	p, _ := NewProvider(providerConfig(server.URL, "anthropic"))
	out, err := p.Review(context.Background(), providerSnapshot())
	server.Close()
	if err == nil || !out.Usage.Known || out.Usage.OutputTokens != 45 {
		t.Fatalf("lost usage for billed failed response: %+v, %v", out, err)
	}
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]string{"type": "text", "text": validReviewJSON}}, "stop_reason": "end_turn"})
	}))
	defer server.Close()
	p, _ = NewProvider(providerConfig(server.URL, "anthropic"))
	out, err = p.Review(context.Background(), providerSnapshot())
	if err != nil || out.Usage.Known {
		t.Fatalf("invented absent usage: %+v, %v", out, err)
	}
}

func TestProviderRejectsIncompleteAndToolResponses(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret")
	for _, protocol := range []string{"anthropic", "openai"} {
		for _, stop := range []string{"", "max_tokens", "length", "tool_use", "tool_calls", "content_filter", "refusal"} {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeProviderResponse(w, protocol, validReviewJSON, stop)
			}))
			p, _ := NewProvider(providerConfig(server.URL, protocol))
			_, err := p.Review(context.Background(), providerSnapshot())
			server.Close()
			if err == nil {
				t.Errorf("accepted %s/%s", protocol, stop)
			}
		}
	}
}

func TestProviderFailuresAreSafeAndNeverRetried(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret-that-must-not-leak")
	for _, status := range []int{http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusTemporaryRedirect} {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("Location", "/secret-that-must-not-leak")
			w.WriteHeader(status)
			io.WriteString(w, "secret-that-must-not-leak")
		}))
		p, _ := NewProvider(providerConfig(server.URL, "anthropic"))
		_, err := p.Review(context.Background(), providerSnapshot())
		server.Close()
		if err == nil || strings.Contains(err.Error(), "secret-that-must-not-leak") || calls.Load() != 1 {
			t.Fatalf("unsafe failure: calls=%d err=%v", calls.Load(), err)
		}
		var statusErr *ProviderHTTPError
		if !errors.As(err, &statusErr) || statusErr.StatusCode != status {
			t.Fatalf("provider HTTP status unavailable: %v", err)
		}
	}
}

func TestProviderResponseBoundAndCancellation(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, strings.Repeat("x", 2<<20)) }))
	defer server.Close()
	p, _ := NewProvider(providerConfig(server.URL, "anthropic"))
	if _, err := p.Review(context.Background(), providerSnapshot()); err == nil {
		t.Error("accepted oversized response")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	if _, err := p.Review(ctx, providerSnapshot()); err == nil {
		t.Error("ignored canceled context")
	}
}

func TestProviderRejectsAmbiguousEnvelope(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret")
	content, _ := json.Marshal(validReviewJSON)
	responses := []struct{ protocol, body string }{
		{"anthropic", `{"stop_reason":"max_tokens","stop_reason":"end_turn","content":[{"type":"text","text":` + string(content) + `}]}`},
		{"anthropic", `{"stop_reason":"end_turn","content":[{"type":"tool_use","text":` + string(content) + `}]}`},
		{"openai", `{"choices":[{"finish_reason":"stop","message":{"content":` + string(content) + `,"tool_calls":[{"type":"function"}]}}]}`},
		{"openai", `{"choices":[{"finish_reason":"stop","message":{"content":` + string(content) + `,"refusal":"refused"}}]}`},
	}
	for _, response := range responses {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, response.body) }))
		p, _ := NewProvider(providerConfig(server.URL, response.protocol))
		_, err := p.Review(context.Background(), providerSnapshot())
		server.Close()
		if err == nil {
			t.Errorf("accepted ambiguous envelope: %s", response.body)
		}
	}
}

func TestProviderThinkingBlocksAreNotReviewContent(t *testing.T) {
	t.Setenv("SCRY_REVIEW_TEST_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		json.NewDecoder(r.Body).Decode(&request)
		if _, ok := request["thinking"]; ok {
			t.Error("provider unexpectedly configures thinking")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"stop_reason": "end_turn",
			"content": []any{
				map[string]string{"type": "thinking", "thinking": "not review JSON; must not be stored"},
				map[string]string{"type": "redacted_thinking", "data": "opaque"},
				map[string]string{"type": "text", "text": validReviewJSON},
			},
		})
	}))
	defer server.Close()
	p, _ := NewProvider(providerConfig(server.URL, "anthropic"))
	out, err := p.Review(context.Background(), providerSnapshot())
	if err != nil || out.Summary != "Changed lookup behavior." {
		t.Fatalf("thinking blocked text review: %+v, %v", out, err)
	}
}
