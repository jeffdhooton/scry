// Package assesscontext builds deterministic, source-backed assessment packets.
// It never invokes recall's public payload formatter or an external provider.
package assesscontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/jeffdhooton/scry/internal/memory/search"
)

var ErrOversize = errors.New("assesscontext: required core exceeds budget")

const maxSourceRecords = 4096
const maxRetrievedBytes = 32 << 20

type SourceReader interface {
	GetSource(string) (assessstore.Source, error)
	Sources(assessstore.SourceQuery) (assessstore.SourcePage, error)
}
type Builder struct {
	Sources SourceReader
	Counter assess.BudgetCounter
	// Optional trial lower bound, applied to raw and derived evidence alike.
	MinSourceTime time.Time
}
type Packet struct {
	Request  []byte          `json:"request"`
	Manifest assess.Manifest `json:"manifest"`
}
type ranked struct {
	source assessstore.Source
	text   string
	score  int
	reason string
}

func opaque(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func sourceRecord(s assessstore.Source, text string) assess.SourceRecord {
	r := assess.SourceRecord{ID: s.ID, Text: text, OccurredAt: s.OccurredAt.UTC().Format(time.RFC3339Nano)}
	if s.OrderKnown {
		r.SourceOrder = s.Order
	}
	return r
}

func (b Builder) Build(ctx context.Context, j assessstore.Job) (Packet, error) {
	var packet Packet
	if b.Sources == nil {
		return packet, errors.New("assesscontext: source reader required")
	}
	if j.Versions.Model != assess.Model || j.Versions.Rubric != assess.RubricV2 || j.Versions.ContextPolicy != assess.ContextPolicyVersion {
		return packet, errors.New("assesscontext: unsupported job versions")
	}
	target, err := b.Sources.GetSource(j.SourceID)
	if err != nil {
		return packet, err
	}
	if target.Unavailable || target.SourceUnavailable || target.Kind == "derived_summary" || target.Text == "" || target.Revision > j.SourceCutoff {
		return packet, assessstore.ErrEvidenceUnavailable
	}
	if target.OccurredAt.IsZero() {
		return packet, errors.New("assesscontext: target time unavailable")
	}
	if !b.MinSourceTime.IsZero() && target.OccurredAt.Before(b.MinSourceTime) {
		return packet, assessstore.ErrEvidenceUnavailable
	}
	counter := b.Counter
	if counter == nil {
		counter = assess.ByteBudgetCounter{}
	}
	state := assess.Context{
		Candidate:     assess.Candidate{Text: j.Candidate.Fact, SourceMention: j.Candidate.Src, RelationMention: j.Candidate.Relation, DestinationMention: j.Candidate.Dst, TemporalClaim: j.Candidate.ValidFrom},
		TargetEpisode: sourceRecord(target, target.Text), SourceHistory: []assess.SourceRecord{}, DerivedContext: []assess.DerivedRecord{},
		EvaluationScope: assess.EvaluationScope{TargetTime: target.OccurredAt.UTC().Format(time.RFC3339Nano), OrderingPolicy: "Evidence at target time only; same-session history requires proven preceding order. Target includes all internal corrections.", SourceGaps: []string{}},
	}
	if target.RepositoryScope != "" {
		state.EvaluationScope.RepositoryScope = opaque(target.RepositoryScope)
	} else {
		state.EvaluationScope.SourceGaps = append(state.EvaluationScope.SourceGaps, "repository_scope_unavailable")
	}
	if target.Namespace == "" {
		state.EvaluationScope.SourceGaps = append(state.EvaluationScope.SourceGaps, "source_namespace_unavailable")
	}
	manifest := assess.Manifest{ContextPolicy: j.Versions.ContextPolicy, SourceRevision: fmt.Sprint(j.SourceCutoff), Sources: []assess.EvidenceRef{{ID: target.ID, Hash: target.Digest, Reason: "complete_target", Rank: 0}}}
	measure := func() (assess.RequestV2, assess.BudgetReport, error) {
		r, e := assess.BuildRequestV2(state)
		if e != nil {
			return r, assess.BudgetReport{}, e
		}
		n, e := counter.Count(r)
		return r, n, e
	}
	fits := func(n assess.BudgetReport) bool { return n.WithinCeilings() && n.EstimatedTokens <= n.TargetTokens }
	_, core, err := measure()
	if err != nil {
		return packet, err
	}
	manifest.Budget = core
	if !fits(core) {
		return Packet{Manifest: manifest}, ErrOversize
	}
	queryTokens := search.Tokenize(j.Candidate.Src + " " + j.Candidate.Dst + " " + j.Candidate.Fact)
	tokenSet := map[string]bool{}
	for _, t := range queryTokens {
		tokenSet[t] = true
	}
	var candidates []ranked
	var after uint64
	scanned, totalBytes := 0, 0
	exclude := func(s assessstore.Source, reason string) {
		manifest.Excluded = append(manifest.Excluded, assess.ExcludedEvidence{ID: s.ID, Reason: reason})
	}
	for {
		if err := ctx.Err(); err != nil {
			return packet, err
		}
		page, e := b.Sources.Sources(assessstore.SourceQuery{Cutoff: j.SourceCutoff, After: after, Limit: 64, Descending: true})
		if e != nil {
			return packet, e
		}
		limited := false
		for _, s := range page.Sources {
			scanned++
			totalBytes += len(s.Text)
			if scanned > maxSourceRecords || totalBytes > maxRetrievedBytes {
				limited = true
				break
			}
			if s.ID == target.ID {
				continue
			}
			if s.EpisodeID == j.EpisodeID {
				exclude(s, "own_extraction")
				continue
			}
			if s.Revision > j.SourceCutoff {
				exclude(s, "beyond_snapshot")
				continue
			}
			if s.OccurredAt.IsZero() {
				exclude(s, "source_time_unavailable")
				continue
			}
			if !b.MinSourceTime.IsZero() && s.OccurredAt.Before(b.MinSourceTime) {
				exclude(s, "before_source_window")
				continue
			}
			if s.OccurredAt.After(target.OccurredAt) {
				exclude(s, "future_source")
				continue
			}
			if s.Unavailable || s.Text == "" {
				exclude(s, "payload_unavailable")
				manifest.MissingRaw = append(manifest.MissingRaw, s.ID)
				continue
			}
			sameSession := s.Source == target.Source && s.SessionID != "" && s.SessionID == target.SessionID
			if !sameSession && recognizableNeighbor(s, target) {
				exclude(s, "source_identity_or_order_unavailable")
				state.EvaluationScope.SourceGaps = append(state.EvaluationScope.SourceGaps, "source_identity_or_order_unavailable")
				continue
			}
			if sameSession && (s.Namespace == "" || target.Namespace == "" || s.Namespace != target.Namespace) {
				exclude(s, "source_namespace_ambiguous")
				continue
			}
			if target.RepositoryScope != s.RepositoryScope && (target.RepositoryScope != "" || s.RepositoryScope != "") {
				exclude(s, "repository_scope_mismatch")
				continue
			}
			text := s.Text
			reason := "related_source"
			score := 0
			if sameSession {
				var why string
				text, why = precedingText(s, target)
				if why != "" {
					exclude(s, why)
					state.EvaluationScope.SourceGaps = append(state.EvaluationScope.SourceGaps, why)
					continue
				}
				reason = "preceding_session"
				score = 100000
			}
			seen := map[string]bool{}
			for _, token := range search.Tokenize(text) {
				if tokenSet[token] && !seen[token] {
					score++
					seen[token] = true
				}
			}
			for _, name := range []string{j.Candidate.Src, j.Candidate.Dst} {
				if name != "" && strings.Contains(strings.ToLower(text), strings.ToLower(name)) {
					score += 1000
				}
			}
			if score == 0 {
				exclude(s, "no_lexical_relevance")
				continue
			}
			if s.Kind == "derived_summary" || s.SourceUnavailable {
				reason = "derived_summary"
				score -= 1000000
			}
			candidates = append(candidates, ranked{s, text, score, reason})
		}
		if limited {
			state.EvaluationScope.SourceGaps = append(state.EvaluationScope.SourceGaps, "retrieval_scan_limit")
			manifest.Truncations = append(manifest.Truncations, "retrieval_scan_limit")
			break
		}
		if page.Next == 0 {
			break
		}
		if after != 0 && page.Next >= after {
			return packet, errors.New("assesscontext: invalid source cursor")
		}
		after = page.Next
	}
	// Source gaps are part of required scope, not history that may be clipped.
	sort.Strings(state.EvaluationScope.SourceGaps)
	state.EvaluationScope.SourceGaps = unique(state.EvaluationScope.SourceGaps)
	_, core, err = measure()
	if err != nil {
		return packet, err
	}
	if !fits(core) {
		manifest.Budget = core
		return Packet{Manifest: manifest}, ErrOversize
	}
	sort.Slice(candidates, func(i, k int) bool {
		a, z := candidates[i], candidates[k]
		if a.score != z.score {
			return a.score > z.score
		}
		if !a.source.OccurredAt.Equal(z.source.OccurredAt) {
			return a.source.OccurredAt.After(z.source.OccurredAt)
		}
		return a.source.ID < z.source.ID
	})
	rawEpisodes := map[string]bool{}
	selectedSources := []assessstore.Source{target}
	for rank, c := range candidates {
		derived := c.reason == "derived_summary"
		if !derived {
			text, why := deduplicateSpans(c.source, selectedSources)
			if why != "" {
				exclude(c.source, why)
				continue
			}
			c.text = text
		}
		if derived && rawEpisodes[c.source.EpisodeID] {
			exclude(c.source, "raw_source_preferred")
			continue
		}
		if derived {
			state.DerivedContext = append(state.DerivedContext, assess.DerivedRecord{ID: c.source.ID, Text: c.text, Kind: "derived_summary", Provenance: "episode:" + opaque(c.source.EpisodeID), Limitation: "source_unavailable=true; extractor summary, not a quotation or independent corroboration"})
		} else {
			state.SourceHistory = append(state.SourceHistory, sourceRecord(c.source, c.text))
		}
		_, budget, e := measure()
		if e != nil {
			return packet, e
		}
		if !fits(budget) {
			if derived {
				state.DerivedContext = state.DerivedContext[:len(state.DerivedContext)-1]
			} else {
				state.SourceHistory = state.SourceHistory[:len(state.SourceHistory)-1]
			}
			exclude(c.source, "whole_record_budget")
			manifest.Truncations = append(manifest.Truncations, c.source.ID)
			continue
		}
		manifest.Sources = append(manifest.Sources, assess.EvidenceRef{ID: c.source.ID, Hash: c.source.Digest, Reason: c.reason, Rank: rank + 1})
		if derived {
			manifest.MissingRaw = append(manifest.MissingRaw, c.source.ID)
		} else {
			rawEpisodes[c.source.EpisodeID] = true
			selectedSources = append(selectedSources, c.source)
		}
	}
	sort.Slice(state.SourceHistory, func(i, k int) bool {
		a, z := state.SourceHistory[i], state.SourceHistory[k]
		// RFC3339 string comparison fails for differing fractional precision.
		ta, _ := time.Parse(time.RFC3339Nano, a.OccurredAt)
		tz, _ := time.Parse(time.RFC3339Nano, z.OccurredAt)
		if !ta.Equal(tz) {
			return ta.Before(tz)
		}
		if a.SourceOrder != z.SourceOrder {
			return a.SourceOrder < z.SourceOrder
		}
		return a.ID < z.ID
	})
	r, n, e := measure()
	if e != nil {
		return packet, e
	}
	manifest.Budget = n
	packet.Request, e = json.Marshal(r)
	packet.Manifest = manifest
	return packet, e
}

func recognizableNeighbor(a, b assessstore.Source) bool {
	if a.Source != b.Source || a.SourceRef == "" || b.SourceRef == "" || a.Source == "manual" {
		return false
	}
	x, _, _ := strings.Cut(a.SourceRef, "#")
	y, _, _ := strings.Cut(b.SourceRef, "#")
	return x != "" && x == y
}

func deduplicateSpans(s assessstore.Source, selected []assessstore.Source) (string, string) {
	var overlapping []assessstore.Source
	for _, p := range selected {
		if s.Namespace != "" && s.Namespace == p.Namespace && s.Source == p.Source && s.SessionID != "" && s.SessionID == p.SessionID && s.SpanKnown && p.SpanKnown && s.Start < p.End && p.Start < s.End {
			overlapping = append(overlapping, p)
		}
	}
	if len(overlapping) == 0 {
		return s.Text, ""
	}
	if len(s.Turns) == 0 {
		return "", "overlap_order_unavailable"
	}
	var lines []string
	for _, turn := range s.Turns {
		duplicate := false
		for _, p := range overlapping {
			if turn.Start >= p.End || turn.End <= p.Start {
				continue
			}
			matched := false
			for _, other := range p.Turns {
				if turn == other {
					matched = true
					break
				}
			}
			if !matched {
				return "", "overlap_identity_unavailable"
			}
			duplicate = true
		}
		if !duplicate {
			lines = append(lines, turn.Speaker+": "+turn.Text)
		}
	}
	if len(lines) == 0 {
		return "", "duplicate_source_span"
	}
	return strings.Join(lines, "\n\n"), ""
}

// Overlap removal uses original source spans and removes only turns that are
// already present in the complete target. Equal text by different speakers is
// never an identity test. Ambiguous partial spans are omitted with a visible gap.
func precedingText(s, target assessstore.Source) (string, string) {
	if s.SpanKnown && target.SpanKnown {
		if s.Start >= target.Start {
			return "", "nonpreceding_source_span"
		}
		if s.End <= target.Start {
			return s.Text, ""
		}
		if s.End > target.End || len(s.Turns) == 0 || len(target.Turns) == 0 {
			return "", "overlap_order_unavailable"
		}
		var lines []string
		for _, turn := range s.Turns {
			if turn.End <= target.Start {
				lines = append(lines, turn.Speaker+": "+turn.Text)
				continue
			}
			matched := false
			for _, other := range target.Turns {
				if turn == other {
					matched = true
					break
				}
			}
			if !matched {
				return "", "overlap_identity_unavailable"
			}
		}
		if len(lines) == 0 {
			return "", "duplicate_source_span"
		}
		return strings.Join(lines, "\n\n"), ""
	}
	if s.OrderKnown && target.OrderKnown && s.Order < target.Order {
		return s.Text, ""
	}
	return "", "source_order_unavailable"
}
func unique(in []string) []string {
	out := in[:0]
	for _, v := range in {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}
