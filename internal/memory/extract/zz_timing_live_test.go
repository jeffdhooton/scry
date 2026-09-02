package extract

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

func TestZZTimingLive(t *testing.T) {
	if os.Getenv("SCRY_TIMING") != "1" {
		t.Skip()
	}
	p := Provider{APIKey: os.Getenv("Z_AI_API_KEY"), Model: "glm-5.3-flash", BaseURL: "https://api.z.ai/api/anthropic"}
	h := NewHaiku(p)
	eps, _, err := distill.ClaudeSession(os.Getenv("SCRY_TIMING_FILE"), 0)
	if err != nil || len(eps) == 0 {
		t.Fatalf("distill: %v %d", err, len(eps))
	}
	full := eps[0]
	for _, size := range []int{4000, 8000, len(full.Text)} {
		ep := full
		if size < len(ep.Text) {
			ep.Text = ep.Text[:size]
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
		start := time.Now()
		res, err := h.Extract(ctx, ep, nil)
		cancel()
		t.Logf("size %d chars: %s err=%v facts=%d", len(ep.Text), time.Since(start).Round(time.Second), err, len(res.Facts))
	}
}
