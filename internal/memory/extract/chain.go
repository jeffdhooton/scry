package extract

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

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
}

// NewChain builds a Chain over steps, in order.
func NewChain(steps ...Step) *Chain {
	return &Chain{steps: steps}
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
	for i, s := range c.steps {
		if err := ctx.Err(); err != nil {
			return Result{}, fmt.Errorf("extract: chain stopped before %s: %w", s.Name, err)
		}
		res, err := s.Extractor.Extract(ctx, ep, glossary)
		if err == nil {
			return res, nil
		}
		lastCause = err
		if !errors.Is(err, ErrParse) {
			allParse = false
		}
		failures = append(failures, fmt.Sprintf("%s: %v", s.Name, err))
		if i+1 < len(c.steps) {
			log.Printf("memory: extraction on %s failed (%v) — falling back to %s", s.Name, err, c.steps[i+1].Name)
		}
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
