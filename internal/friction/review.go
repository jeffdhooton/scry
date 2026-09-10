package friction

import (
	"context"
	"fmt"
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

// Routing reports where a signature's corrections were sent and whether that
// destination held. Recommendations only: a verdict cites stored counts and
// never authorizes a change.
type Routing struct {
	Status        string   `json:"status"` // unrouted | holding | outgrown | terminal
	CurrentKind   string   `json:"current_kind,omitempty"`
	KindsObserved []string `json:"kinds_observed"`
	RunsAtCurrent int      `json:"runs_at_current_kind"`
	SuggestedKind string   `json:"suggested_kind,omitempty"`
	Rationale     string   `json:"rationale"`
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
	Routing                     Routing    `json:"routing"`
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
	kindRuns := map[string]map[string]map[string]bool{}
	for _, e := range p.Events {
		g := groups[e.Signature]
		if g == nil {
			g = &Group{Signature: e.Signature, Events: []Event{}, Proposals: []Proposal{}}
			groups[e.Signature] = g
			runs[e.Signature] = map[string]bool{}
		}
		g.Events = append(g.Events, e)
		runs[e.Signature][e.RunID] = true
		if e.DestinationKind != "" {
			if kindRuns[e.Signature] == nil {
				kindRuns[e.Signature] = map[string]map[string]bool{}
			}
			if kindRuns[e.Signature][e.DestinationKind] == nil {
				kindRuns[e.Signature][e.DestinationKind] = map[string]bool{}
			}
			kindRuns[e.Signature][e.DestinationKind][e.RunID] = true
		}
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
		g.Routing = routingFor(kindRuns[signature])
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

// routingFor derives the verdict from stored events alone. Recurrence is counted
// at the current rung, not across the group: acknowledging a promotion lands one
// run at the new rung, and only a second run there proves that rung failed too.
func routingFor(runsByKind map[string]map[string]bool) Routing {
	r := Routing{Status: "unrouted", KindsObserved: []string{}}
	current := -1
	for kind := range runsByKind {
		rung, ok := destinationRung(kind)
		if !ok {
			continue
		}
		r.KindsObserved = append(r.KindsObserved, kind)
		if rung > current {
			current = rung
		}
	}
	if current < 0 {
		r.Rationale = "No event in this group named a destination_kind, so there is no rung to judge."
		return r
	}
	sort.Slice(r.KindsObserved, func(i, j int) bool {
		a, _ := destinationRung(r.KindsObserved[i])
		b, _ := destinationRung(r.KindsObserved[j])
		return a < b
	})
	r.CurrentKind = destinationLadder[current]
	r.RunsAtCurrent = len(runsByKind[r.CurrentKind])
	switch {
	case r.RunsAtCurrent < 2:
		r.Status = "holding"
		r.Rationale = fmt.Sprintf("Routed to %s in one distinct run; that destination has not yet been shown to fail.", r.CurrentKind)
	case current == len(destinationLadder)-1:
		r.Status = "terminal"
		r.Rationale = fmt.Sprintf("Routed to gate and still recurring across %d distinct runs; no stronger destination exists, so treat this as a defect in the gate.", r.RunsAtCurrent)
	default:
		r.Status = "outgrown"
		r.SuggestedKind = destinationLadder[current+1]
		r.Rationale = fmt.Sprintf("Routed to %s and still recurring across %d distinct runs; the evidence supports at least %s.", r.CurrentKind, r.RunsAtCurrent, r.SuggestedKind)
	}
	return r
}
