package extract

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// countingExtractor records calls and returns a scripted outcome.
type countingExtractor struct {
	calls int
	res   Result
	err   error
}

func (f *countingExtractor) Extract(context.Context, distill.RawEpisode, []string) (Result, error) {
	f.calls++
	return f.res, f.err
}

func parseFailure(reply string) error {
	return fmt.Errorf("extract: invalid JSON after 2 repairs: %w: unexpected end (reply: %q)", ErrParse, reply)
}

func TestChainReturnsPrimaryResultWithoutTouchingFallback(t *testing.T) {
	primary := &countingExtractor{res: Result{EpisodeSummary: "from primary"}}
	fallback := &countingExtractor{res: Result{EpisodeSummary: "from fallback"}}

	res, err := NewChain(Step{"a", primary}, Step{"b", fallback}).Extract(context.Background(), distill.RawEpisode{}, nil)

	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if res.EpisodeSummary != "from primary" {
		t.Errorf("EpisodeSummary = %q, want from primary", res.EpisodeSummary)
	}
	if fallback.calls != 0 {
		t.Errorf("fallback called %d times, want 0", fallback.calls)
	}
}

func TestChainFallsBackWhenPrimaryFails(t *testing.T) {
	primary := &countingExtractor{err: parseFailure("")}
	fallback := &countingExtractor{res: Result{EpisodeSummary: "from fallback"}}

	res, err := NewChain(Step{"a", primary}, Step{"b", fallback}).Extract(context.Background(), distill.RawEpisode{}, nil)

	if err != nil {
		t.Fatalf("Extract() error = %v, want the fallback's success", err)
	}
	if res.EpisodeSummary != "from fallback" {
		t.Errorf("EpisodeSummary = %q, want from fallback", res.EpisodeSummary)
	}
	if primary.calls != 1 || fallback.calls != 1 {
		t.Errorf("calls = primary %d, fallback %d; want 1 and 1", primary.calls, fallback.calls)
	}
}

func TestChainAllParseFailuresWrapsErrParseAndNamesEveryModel(t *testing.T) {
	primary := &countingExtractor{err: parseFailure("")}
	fallback := &countingExtractor{err: parseFailure("{oops")}

	_, err := NewChain(Step{"deepseek-v4-flash", primary}, Step{"deepseek-v4-pro", fallback}).Extract(context.Background(), distill.RawEpisode{}, nil)

	if !errors.Is(err, ErrParse) {
		t.Errorf("error = %v, want it to wrap ErrParse when every model failed on content", err)
	}
	for _, want := range []string{"deepseek-v4-flash", "deepseek-v4-pro", `reply: ""`, `reply: "{oops"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to mention %q", err, want)
		}
	}
}

func TestChainTransportFailureAnywhereIsNotErrParse(t *testing.T) {
	primary := &countingExtractor{err: errors.New("extract: haiku request failed: 502")}
	fallback := &countingExtractor{err: parseFailure("")}

	_, err := NewChain(Step{"a", primary}, Step{"b", fallback}).Extract(context.Background(), distill.RawEpisode{}, nil)

	if err == nil {
		t.Fatal("error = nil, want failure")
	}
	if errors.Is(err, ErrParse) {
		t.Errorf("error = %v wraps ErrParse, but a transport failure in the chain must not be treated as skippable", err)
	}
}

func TestChainStopsWhenContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	primary := &countingExtractor{err: parseFailure("")}
	fallback := &countingExtractor{res: Result{EpisodeSummary: "unreachable"}}
	cancel()

	_, err := NewChain(Step{"a", primary}, Step{"b", fallback}).Extract(ctx, distill.RawEpisode{}, nil)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if fallback.calls != 0 {
		t.Errorf("fallback called %d times after cancellation, want 0", fallback.calls)
	}
}

func TestNewExtractorBuildsOneStepPerProvider(t *testing.T) {
	ps := Providers{Providers: []Provider{
		{APIKey: "k", Model: "deepseek-v4-flash"},
		{APIKey: "k", Model: "deepseek-v4-pro"},
	}}

	ch, ok := NewExtractor(ps).(*Chain)
	if !ok {
		t.Fatalf("NewExtractor() = %T, want *Chain", NewExtractor(ps))
	}
	if got := ch.Names(); strings.Join(got, ",") != "deepseek-v4-flash,deepseek-v4-pro" {
		t.Errorf("Names() = %v", got)
	}
}

func TestChainCoolsDownAStepThatRefusesOnBilling(t *testing.T) {
	primary := &countingExtractor{err: errors.New(`extract: haiku request failed: POST "https://api.deepseek.com/anthropic/v1/messages": 402 Payment Required {"error":{"message":"Insufficient Balance"}}`)}
	fallback := &countingExtractor{res: Result{EpisodeSummary: "from fallback"}}
	ch := NewChain(Step{"deepseek", primary}, Step{"glm", fallback})
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	ch.now = func() time.Time { return now }

	for range 3 {
		res, err := ch.Extract(context.Background(), distill.RawEpisode{}, nil)
		if err != nil || res.EpisodeSummary != "from fallback" {
			t.Fatalf("Extract() = %+v, %v", res, err)
		}
	}
	if primary.calls != 1 {
		t.Errorf("primary called %d times, want 1 (cooling down after the 402)", primary.calls)
	}

	now = now.Add(CooldownPeriod + time.Second)
	_, _ = ch.Extract(context.Background(), distill.RawEpisode{}, nil)
	if primary.calls != 2 {
		t.Errorf("primary called %d times after the cooldown elapsed, want 2", primary.calls)
	}
}

func TestChainAllStepsCoolingIsATransportFailure(t *testing.T) {
	refused := errors.New("401 Unauthorized: invalid api key")
	a := &countingExtractor{err: refused}
	b := &countingExtractor{err: refused}
	ch := NewChain(Step{"a", a}, Step{"b", b})

	_, err := ch.Extract(context.Background(), distill.RawEpisode{}, nil)
	if err == nil || errors.Is(err, ErrParse) {
		t.Fatalf("first Extract() err = %v, want a non-parse failure", err)
	}
	_, err = ch.Extract(context.Background(), distill.RawEpisode{}, nil)
	if err == nil || errors.Is(err, ErrParse) || !strings.Contains(err.Error(), "cooling") {
		t.Fatalf("second Extract() err = %v, want a cooling-down transport error", err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("calls = %d, %d; want 1, 1", a.calls, b.calls)
	}
}

func TestRefusedOnBillingOrAuth(t *testing.T) {
	cases := map[string]bool{
		"402 Payment Required":                         true,
		"Insufficient Balance":                         true,
		"403 Forbidden":                                true,
		"401 Unauthorized":                             true,
		"authentication_error: invalid x-api-key":      true,
		"extract: invalid JSON after 2 repairs":        false,
		"500 Internal Server Error":                    false,
		"context deadline exceeded":                    false,
		"episode 4021 references 4031":                 false,
		"[1210] This model always engages in thinking": false,
		"429 Too Many Requests":                        false,
	}
	for msg, want := range cases {
		if got := refusedOnBillingOrAuth(errors.New(msg)); got != want {
			t.Errorf("refusedOnBillingOrAuth(%q) = %v, want %v", msg, got, want)
		}
	}
}

func TestChainKeepsATimeoutVisibleFromAnyStep(t *testing.T) {
	a := &countingExtractor{err: parseFailure("")}
	b := &countingExtractor{err: fmt.Errorf("extract: haiku request failed: %w", context.DeadlineExceeded)}
	_, err := NewChain(Step{"a", a}, Step{"b", b}).Extract(context.Background(), distill.RawEpisode{}, nil)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrParse) {
		t.Fatalf("err = %v, want a deadline error that is not a parse error", err)
	}
}
