package friction

import (
	"context"
	"math"
	"sort"
)

type Proposal struct {
	EventID      string `json:"event_id"`
	Change       string `json:"change"`
	Owner        string `json:"owner,omitempty"`
	File         string `json:"file,omitempty"`
	Verification string `json:"verification,omitempty"`
}

type Group struct {
	Signature                   string     `json:"signature"`
	RunIDs                      []string   `json:"run_ids"`
	DistinctRuns                int        `json:"distinct_runs"`
	Recurring                   bool       `json:"recurring"`
	Events                      []Event    `json:"events"`
	MeasuredEvents              int        `json:"measured_events"`
	MeasuredUserTimeCostSeconds *float64   `json:"measured_user_time_cost_seconds"`
	Proposals                   []Proposal `json:"proposals"`
}

type Review struct {
	Repository          string  `json:"repository"`
	EventCount          int     `json:"event_count"`
	Groups              []Group `json:"groups"`
	RecommendationsOnly bool    `json:"recommendations_only"`
}

// Review groups exact caller-assigned signatures. Counts come only from stored
// events, never the caller's occurrence/prior-run estimates. Refuse truncation
// instead of returning a partial report that looks like a complete review.
func (s *Store) Review(ctx context.Context, f Filter) (*Review, error) {
	if f.After != "" || f.Limit != 0 {
		return nil, invalid("review does not accept after or limit; narrow by run_id, signature or time")
	}
	p, err := s.List(ctx, f)
	if err != nil {
		return nil, err
	}
	if p.NextAfter != "" {
		return nil, ErrReviewTooLarge
	}
	r := &Review{Repository: f.Repository, EventCount: len(p.Events), Groups: []Group{}, RecommendationsOnly: true}
	groups := map[string]*Group{}
	runs := map[string]map[string]bool{}
	for _, e := range p.Events {
		g := groups[e.Signature]
		if g == nil {
			g = &Group{Signature: e.Signature, Events: []Event{}, Proposals: []Proposal{}}
			groups[e.Signature] = g
			runs[e.Signature] = map[string]bool{}
		}
		g.Events = append(g.Events, e)
		runs[e.Signature][e.RunID] = true
		if e.MeasuredUserTimeCostSeconds != nil {
			if g.MeasuredUserTimeCostSeconds == nil {
				g.MeasuredUserTimeCostSeconds = new(float64)
			}
			*g.MeasuredUserTimeCostSeconds += *e.MeasuredUserTimeCostSeconds
			if math.IsInf(*g.MeasuredUserTimeCostSeconds, 0) {
				return nil, invalid("measured cost total overflows for signature %q; narrow the review", e.Signature)
			}
			g.MeasuredEvents++
		}
		if e.ProposedChange != "" {
			g.Proposals = append(g.Proposals, Proposal{e.EventID, e.ProposedChange, e.ProposedOwner, e.ProposedFile, e.ProposedVerification})
		}
	}
	for signature, g := range groups {
		for run := range runs[signature] {
			g.RunIDs = append(g.RunIDs, run)
		}
		sort.Strings(g.RunIDs)
		g.DistinctRuns = len(g.RunIDs)
		g.Recurring = g.DistinctRuns >= 2
		r.Groups = append(r.Groups, *g)
	}
	sort.Slice(r.Groups, func(i, j int) bool {
		if r.Groups[i].DistinctRuns != r.Groups[j].DistinctRuns {
			return r.Groups[i].DistinctRuns > r.Groups[j].DistinctRuns
		}
		return r.Groups[i].Signature < r.Groups[j].Signature
	})
	return r, nil
}
