package extract

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

func makeEpisodes(n int) []distill.RawEpisode {
	eps := make([]distill.RawEpisode, n)
	for i := range eps {
		eps[i] = distill.RawEpisode{ID: "ep-" + itoa(i), Text: "text"}
	}
	return eps
}

// itoa avoids pulling in strconv just for test fixture IDs.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func TestChunkEpisodes(t *testing.T) {
	cases := []struct {
		name       string
		n          int
		wantChunks []int // lengths of each chunk, in order
	}{
		{"999 fits in one chunk", 999, []int{999}},
		{"1000 fits exactly in one chunk", 1000, []int{1000}},
		{"1001 spills into a second chunk", 1001, []int{1000, 1}},
		{"0 episodes yields no chunks", 0, nil},
		{"2001 spans three chunks", 2001, []int{1000, 1000, 1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := chunkEpisodes(makeEpisodes(tc.n))
			if len(got) != len(tc.wantChunks) {
				t.Fatalf("len(chunks) = %d, want %d (chunks: %v)", len(got), len(tc.wantChunks), chunkLens(got))
			}
			for i, chunk := range got {
				if len(chunk) != tc.wantChunks[i] {
					t.Errorf("chunk[%d] len = %d, want %d", i, len(chunk), tc.wantChunks[i])
				}
			}
			// Every episode must appear in exactly one chunk, in original order.
			var flat []distill.RawEpisode
			for _, chunk := range got {
				flat = append(flat, chunk...)
			}
			if len(flat) != tc.n {
				t.Fatalf("total episodes across chunks = %d, want %d", len(flat), tc.n)
			}
			for i, ep := range flat {
				if ep.ID != "ep-"+itoa(i) {
					t.Errorf("flat[%d].ID = %q, want %q (chunking must preserve order)", i, ep.ID, "ep-"+itoa(i))
				}
			}
		})
	}
}

func chunkLens(chunks [][]distill.RawEpisode) []int {
	lens := make([]int, len(chunks))
	for i, c := range chunks {
		lens[i] = len(c)
	}
	return lens
}

func TestBuildBatchRequests(t *testing.T) {
	eps := []distill.RawEpisode{
		{ID: "abc123", Source: "claude-session", Text: "hello", Cwd: "/tmp/proj"},
		{ID: "def456", Source: "claude-session", Text: "world", Cwd: "/tmp/proj"},
	}
	glossary := []string{"book-system: also known as authorclaw"}

	reqs := buildBatchRequests(eps, glossary, "claude-haiku-4-5")

	if len(reqs) != 2 {
		t.Fatalf("len(reqs) = %d, want 2", len(reqs))
	}

	for i, ep := range eps {
		req := reqs[i]
		if req.CustomID != ep.ID {
			t.Errorf("reqs[%d].CustomID = %q, want %q", i, req.CustomID, ep.ID)
		}
		if req.Params.Model != anthropic.Model("claude-haiku-4-5") {
			t.Errorf("reqs[%d].Params.Model = %q, want claude-haiku-4-5", i, req.Params.Model)
		}
		if req.Params.MaxTokens != 4000 {
			t.Errorf("reqs[%d].Params.MaxTokens = %d, want 4000", i, req.Params.MaxTokens)
		}

		wantSystem, wantUserMsg := buildMessages(ep, glossary)
		if len(req.Params.System) != len(wantSystem) {
			t.Fatalf("reqs[%d].Params.System len = %d, want %d", i, len(req.Params.System), len(wantSystem))
		}
		for j := range wantSystem {
			if req.Params.System[j].Text != wantSystem[j].Text {
				t.Errorf("reqs[%d].Params.System[%d].Text mismatch", i, j)
			}
			if req.Params.System[j].CacheControl.Type != wantSystem[j].CacheControl.Type {
				t.Errorf("reqs[%d].Params.System[%d].CacheControl mismatch: got %q want %q", i, j, req.Params.System[j].CacheControl.Type, wantSystem[j].CacheControl.Type)
			}
		}

		if len(req.Params.Messages) != 1 {
			t.Fatalf("reqs[%d].Params.Messages len = %d, want 1", i, len(req.Params.Messages))
		}
		if req.Params.Messages[0].Content[0].OfText.Text != wantUserMsg.Content[0].OfText.Text {
			t.Errorf("reqs[%d].Params.Messages[0] text mismatch", i)
		}
	}
}

func TestRouteResult(t *testing.T) {
	t.Run("succeeded with valid JSON returns a Result", func(t *testing.T) {
		item := batchItem{
			CustomID: "ep1",
			Kind:     resultSucceeded,
			Text:     `{"episode_summary": "did a thing", "entities": [], "facts": []}`,
		}
		res, err := routeResult(item)
		if err != nil {
			t.Fatalf("routeResult() error = %v, want nil", err)
		}
		if res.EpisodeSummary != "did a thing" {
			t.Errorf("EpisodeSummary = %q", res.EpisodeSummary)
		}
	})

	t.Run("succeeded with malformed JSON returns an error", func(t *testing.T) {
		item := batchItem{
			CustomID: "ep1",
			Kind:     resultSucceeded,
			Text:     "not json at all",
		}
		_, err := routeResult(item)
		if err == nil {
			t.Fatal("routeResult() error = nil, want error for malformed JSON")
		}
	})

	t.Run("errored returns an error mentioning the API message", func(t *testing.T) {
		item := batchItem{
			CustomID: "ep1",
			Kind:     resultErrored,
			ErrMsg:   "rate limited",
		}
		_, err := routeResult(item)
		if err == nil {
			t.Fatal("routeResult() error = nil, want error")
		}
		if got := err.Error(); !contains(got, "rate limited") {
			t.Errorf("error = %q, want it to mention %q", got, "rate limited")
		}
	})

	t.Run("canceled returns an error", func(t *testing.T) {
		item := batchItem{CustomID: "ep1", Kind: resultCanceled}
		_, err := routeResult(item)
		if err == nil {
			t.Fatal("routeResult() error = nil, want error")
		}
	})

	t.Run("expired returns an error", func(t *testing.T) {
		item := batchItem{CustomID: "ep1", Kind: resultExpired}
		_, err := routeResult(item)
		if err == nil {
			t.Fatal("routeResult() error = nil, want error")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}

func TestRouteBatchResults(t *testing.T) {
	items := []batchItem{
		{CustomID: "ok1", Kind: resultSucceeded, Text: `{"episode_summary": "s1", "entities": [], "facts": []}`},
		{CustomID: "ok2", Kind: resultSucceeded, Text: `{"episode_summary": "s2", "entities": [], "facts": []}`},
		{CustomID: "bad-json", Kind: resultSucceeded, Text: "not json"},
		{CustomID: "err1", Kind: resultErrored, ErrMsg: "boom"},
		{CustomID: "canceled1", Kind: resultCanceled},
		{CustomID: "expired1", Kind: resultExpired},
	}

	results, errs := routeBatchResults(items)

	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2 (got %v)", len(results), results)
	}
	if results["ok1"].EpisodeSummary != "s1" || results["ok2"].EpisodeSummary != "s2" {
		t.Errorf("results = %+v", results)
	}

	wantErrKeys := []string{"bad-json", "err1", "canceled1", "expired1"}
	if len(errs) != len(wantErrKeys) {
		t.Fatalf("len(errs) = %d, want %d (got %v)", len(errs), len(wantErrKeys), errs)
	}
	for _, k := range wantErrKeys {
		if errs[k] == nil {
			t.Errorf("errs[%q] = nil, want an error", k)
		}
	}

	// Every custom_id must land in exactly one of the two maps.
	for _, item := range items {
		_, inResults := results[item.CustomID]
		_, inErrs := errs[item.CustomID]
		if inResults == inErrs {
			t.Errorf("custom_id %q: inResults=%v inErrs=%v, want exactly one", item.CustomID, inResults, inErrs)
		}
	}
}

func TestNewBatchRunnerDefaultsModel(t *testing.T) {
	b := NewBatchRunner("test-key", "")
	if b.model != defaultModel {
		t.Errorf("model = %q, want default %q", b.model, defaultModel)
	}

	b2 := NewBatchRunner("test-key", "claude-opus-4")
	if b2.model != "claude-opus-4" {
		t.Errorf("model = %q, want claude-opus-4", b2.model)
	}
}
