package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// alwaysThinksServer imitates a provider that refuses to turn reasoning
// off. Z.ai answers exactly this way for GLM-5.3-Flash:
//
//	[1210] This model always engages in thinking and cannot be disabled;
//	       please use low, high, or max
//
// It records whether each request carried a thinking field so a test can
// assert the retry dropped it.
type alwaysThinksServer struct {
	sawThinking []bool
}

func (s *alwaysThinksServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req map[string]any
	_ = json.Unmarshal(body, &req)
	_, has := req["thinking"]
	s.sawThinking = append(s.sawThinking, has)

	w.Header().Set("Content-Type", "application/json")
	if has {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"type":"error","error":{"type":"invalid_request_error","code":"1210",`+
			`"message":"[1210][This model always engages in thinking and cannot be disabled; please use low, high, or max]"}}`)
		return
	}
	good := `{"episode_summary":"ok","entities":[],"facts":[]}`
	payload := map[string]any{
		"id": "m", "type": "message", "role": "assistant", "model": "test",
		"stop_reason": "end_turn",
		"content":     []map[string]string{{"type": "text", "text": good}},
		"usage":       map[string]int{"input_tokens": 10, "output_tokens": 10},
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// TestDisablingThinkingIsBestEffort is the regression test for making a
// provider-specific optimisation mandatory.
//
// Turning reasoning off saves budget on providers that allow it, but some
// models cannot turn it off and reject the request outright. Losing every
// memory write to an optimisation is far worse than paying for reasoning.
func TestDisablingThinkingIsBestEffort(t *testing.T) {
	server := &alwaysThinksServer{}
	ts := httptest.NewServer(server)
	defer ts.Close()

	h := NewHaiku(Provider{APIKey: "test", Model: "test", BaseURL: ts.URL})

	got, err := h.Extract(context.Background(), testEpisode(), nil)
	if err != nil {
		t.Fatalf("Extract() error = %v, want the write to survive", err)
	}
	if got.EpisodeSummary != "ok" {
		t.Errorf("EpisodeSummary = %q, want %q", got.EpisodeSummary, "ok")
	}
	if len(server.sawThinking) < 2 {
		t.Fatalf("expected a retry without thinking, got %d request(s)", len(server.sawThinking))
	}
	if !server.sawThinking[0] {
		t.Error("first attempt should still try to disable thinking")
	}
	if server.sawThinking[1] {
		t.Error("retry must drop the thinking field entirely")
	}
}

// TestUnrelatedBadRequestStillFails keeps the retry narrow. A 400 that has
// nothing to do with reasoning must not be silently retried.
func TestUnrelatedBadRequestStillFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens too large"}}`)
	}))
	defer ts.Close()

	h := NewHaiku(Provider{APIKey: "test", Model: "test", BaseURL: ts.URL})
	_, err := h.Extract(context.Background(), testEpisode(), nil)
	if err == nil {
		t.Fatal("Extract() error = nil, want an unrelated 400 to fail")
	}
	if !strings.Contains(err.Error(), "max_tokens too large") {
		t.Errorf("error = %v, want the provider's own message", err)
	}
}
