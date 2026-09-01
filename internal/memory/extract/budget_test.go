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
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// truncatedThenGood serves a first reply that spent its whole budget on
// reasoning -- no text block, stop_reason "max_tokens" -- and a valid result
// on every call after that. It records the max_tokens of each request so a
// test can assert the retry actually asked for more room.
type truncatedThenGood struct {
	budgets []int64
}

func (s *truncatedThenGood) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		MaxTokens int64 `json:"max_tokens"`
	}
	_ = json.Unmarshal(body, &req)
	s.budgets = append(s.budgets, req.MaxTokens)

	w.Header().Set("Content-Type", "application/json")
	if len(s.budgets) == 1 {
		// Reasoning consumed the entire budget. No text block at all.
		fmt.Fprint(w, `{"id":"m","type":"message","role":"assistant",
			"model":"test","stop_reason":"max_tokens",
			"content":[{"type":"thinking","thinking":"deliberating..."}],
			"usage":{"input_tokens":10,"output_tokens":10}}`)
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

func testEpisode() distill.RawEpisode {
	return distill.RawEpisode{
		Text:       "a long fact worth remembering",
		Cwd:        "/tmp",
		OccurredAt: time.Now(),
	}
}

// TestTruncatedReplyRetriesWithMoreRoom is the regression test for silently
// dropped memory writes.
//
// A reply that stopped at max_tokens with nothing but reasoning is not
// malformed JSON -- it is no JSON. Answering it with "your JSON was invalid,
// send it again" cannot help, and it burned both repair attempts, so the
// whole fact was lost. The retry has to buy more room instead.
func TestTruncatedReplyRetriesWithMoreRoom(t *testing.T) {
	server := &truncatedThenGood{}
	ts := httptest.NewServer(server)
	defer ts.Close()

	h := NewHaiku(Provider{APIKey: "test", Model: "test", BaseURL: ts.URL})

	got, err := h.Extract(context.Background(), testEpisode(), nil)
	if err != nil {
		t.Fatalf("Extract() error = %v, want the fact recovered", err)
	}
	if got.EpisodeSummary != "ok" {
		t.Errorf("EpisodeSummary = %q, want %q", got.EpisodeSummary, "ok")
	}
	if len(server.budgets) < 2 {
		t.Fatalf("expected a retry, got %d request(s)", len(server.budgets))
	}
	if server.budgets[1] <= server.budgets[0] {
		t.Errorf("retry budget %d must exceed the first budget %d",
			server.budgets[1], server.budgets[0])
	}
}

// TestExtractionBudgetLeavesRoomForReasoning guards the floor. The extractor
// talks to a reasoning model by default, so a budget sized only for the JSON
// answer is one long episode away from producing nothing at all.
func TestExtractionBudgetLeavesRoomForReasoning(t *testing.T) {
	if extractionMaxTokens < 16000 {
		t.Errorf("extractionMaxTokens = %d, too tight for a reasoning model",
			extractionMaxTokens)
	}
	if retryMaxTokens <= extractionMaxTokens {
		t.Errorf("retryMaxTokens = %d must exceed extractionMaxTokens = %d",
			retryMaxTokens, extractionMaxTokens)
	}
}

// TestTruncatedReplyErrorNamesTheCause keeps the diagnostic that made this
// bug findable: an empty reply says nothing on its own.
func TestTruncatedReplyErrorNamesTheCause(t *testing.T) {
	always := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"m","type":"message","role":"assistant",
			"model":"test","stop_reason":"max_tokens",
			"content":[{"type":"thinking","thinking":"..."}],
			"usage":{"input_tokens":10,"output_tokens":10}}`)
	}))
	defer always.Close()

	h := NewHaiku(Provider{APIKey: "test", Model: "test", BaseURL: always.URL})
	_, err := h.Extract(context.Background(), testEpisode(), nil)
	if err == nil {
		t.Fatal("Extract() error = nil, want failure when every reply truncates")
	}
	if !strings.Contains(err.Error(), "max_tokens") {
		t.Errorf("error = %v, want it to name max_tokens", err)
	}
}
