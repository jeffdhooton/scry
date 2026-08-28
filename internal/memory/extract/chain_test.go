package extract

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

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
