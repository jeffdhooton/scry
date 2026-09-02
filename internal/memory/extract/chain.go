package extract

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// CooldownPeriod is how long a step that refused on billing or auth is
// skipped without a network call. A 402 does not clear itself in seconds,
// and asking again on every episode turns one dead key into hundreds of
// identical log lines and a stalled sweep.
const CooldownPeriod = 15 * time.Minute

var billingAuthRE = regexp.MustCompile(`(?i)\b40[123]\b|insufficient balance|unauthorized|forbidden|authentication_error|invalid (x-)?api[- _]?key|payment required`)

// refusedOnBillingOrAuth reports whether err is a provider turning the
// request away for reasons that will not change on the next episode: a
// dead key, no balance, a forbidden endpoint. Rate limits (429) and server
// errors are deliberately excluded; they clear on their own.
func refusedOnBillingOrAuth(err error) bool {
	return err != nil && billingAuthRE.MatchString(err.Error())
}

// Step is one link in a Chain: a named Extractor. The name is the model id
// and exists for logs and errors, so a failure says which model produced it.
type Step struct {
	Name      string
	Extractor Extractor
}

// Chain is an Extractor that tries each step in order and returns the first
// success. It exists because a cheap primary model sometimes returns
// nothing at all — an empty reply, or an entity type it invented — and a
// single-model pipeline dead-letters the episode on the spot. A fallback
// turns that into a slightly more expensive success.
//
// The combined error wraps ErrParse only when every step failed on content.
// A transport failure anywhere means the episode should be retried on the
// next sweep rather than skipped, and callers decide that with
// errors.Is(err, ErrParse).
type Chain struct {
	steps []Step
	now   func() time.Time

	mu      sync.Mutex
	cooling []time.Time // per step: skip until this instant (zero = live)
}

// NewChain builds a Chain over steps, in order.
func NewChain(steps ...Step) *Chain {
	return &Chain{steps: steps, now: time.Now, cooling: make([]time.Time, len(steps))}
}

// cooledDown reports whether step i is currently skipped.
func (c *Chain) cooledDown(i int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now().Before(c.cooling[i])
}

// coolDown marks step i as skipped for CooldownPeriod.
func (c *Chain) coolDown(i int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cooling[i] = c.now().Add(CooldownPeriod)
}

// NewExtractor builds the Extractor for a resolved provider chain: a Haiku
// per provider, wrapped in a Chain. A one-entry chain behaves exactly like
// a bare Haiku.
func NewExtractor(ps Providers) Extractor {
	steps := make([]Step, 0, len(ps.Providers))
	for _, p := range ps.Providers {
		steps = append(steps, Step{Name: p.resolveModel(), Extractor: NewHaiku(p)})
	}
	return NewChain(steps...)
}

// Names lists the step names in order.
func (c *Chain) Names() []string {
	out := make([]string, len(c.steps))
	for i, s := range c.steps {
		out[i] = s.Name
	}
	return out
}

// Extract tries each step until one succeeds. Every failure is logged as it
// happens, naming the model and the one it is falling back to, so the
// daemon log shows how often the primary is being rescued.
func (c *Chain) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (Result, error) {
	var (
		failures  []string
		allParse  = true
		lastCause error
	)
	tried := 0
	for i, s := range c.steps {
		if err := ctx.Err(); err != nil {
			return Result{}, fmt.Errorf("extract: chain stopped before %s: %w", s.Name, err)
		}
		if c.cooledDown(i) {
			allParse = false
			failures = append(failures, fmt.Sprintf("%s: cooling down after a billing/auth refusal", s.Name))
			continue
		}
		tried++
		res, err := s.Extractor.Extract(ctx, ep, glossary)
		if err == nil {
			return res, nil
		}
		lastCause = err
		if !errors.Is(err, ErrParse) {
			allParse = false
		}
		if refusedOnBillingOrAuth(err) {
			c.coolDown(i)
			log.Printf("memory: %s refused on billing/auth (%v) — skipping it for %s", s.Name, err, CooldownPeriod)
		}
		failures = append(failures, fmt.Sprintf("%s: %v", s.Name, err))
		if i+1 < len(c.steps) {
			log.Printf("memory: extraction on %s failed (%v) — falling back to %s", s.Name, err, c.steps[i+1].Name)
		}
	}
	if tried == 0 {
		return Result{}, fmt.Errorf("extract: every model is cooling down after a billing/auth refusal: %s", strings.Join(failures, "; "))
	}
	if len(c.steps) == 1 {
		return Result{}, lastCause
	}
	joined := strings.Join(failures, "; ")
	if allParse {
		return Result{}, fmt.Errorf("extract: all %d models failed: %w: %s", len(c.steps), ErrParse, joined)
	}
	return Result{}, fmt.Errorf("extract: all %d models failed: %s", len(c.steps), joined)
}
