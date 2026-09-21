package assess

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

const ContextPolicyVersion = "context-policy-v1"
const DefaultTargetInputTokens = 20000
const StateQuestionCeiling = 30000
const RequestCeiling = 60000

// IDs sent to the provider must be opaque; local source paths belong only in
// the retained manifest, never in these records.
type Candidate struct {
	Text               string `json:"text"`
	SourceMention      string `json:"source_mention,omitempty"`
	RelationMention    string `json:"relation_mention,omitempty"`
	DestinationMention string `json:"destination_mention,omitempty"`
	TemporalClaim      string `json:"temporal_claim,omitempty"`
}
type SourceRecord struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Speaker     string `json:"speaker,omitempty"`
	OccurredAt  string `json:"occurred_at,omitempty"`
	SourceOrder int64  `json:"source_order,omitempty"`
}
type DerivedRecord struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	Kind       string `json:"kind"`
	Provenance string `json:"provenance"`
	Limitation string `json:"limitation"`
}
type EvaluationScope struct {
	TargetTime      string   `json:"target_time"`
	RepositoryScope string   `json:"repository_scope,omitempty"`
	SourceGaps      []string `json:"source_gaps,omitempty"`
	OrderingPolicy  string   `json:"ordering_policy,omitempty"`
}
type Context struct {
	Candidate       Candidate       `json:"candidate"`
	TargetEpisode   SourceRecord    `json:"target_episode"`
	SourceHistory   []SourceRecord  `json:"source_history"`
	DerivedContext  []DerivedRecord `json:"derived_context"`
	EvaluationScope EvaluationScope `json:"evaluation_scope"`
}
type EvidenceRef struct {
	ID     string `json:"id"`
	Hash   string `json:"hash"`
	Reason string `json:"reason"`
	Rank   int    `json:"rank"`
}
type ExcludedEvidence struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}
type Manifest struct {
	Sources        []EvidenceRef      `json:"sources"`
	Excluded       []ExcludedEvidence `json:"excluded,omitempty"`
	Truncations    []string           `json:"truncations,omitempty"`
	MissingRaw     []string           `json:"missing_raw,omitempty"`
	SourceRevision string             `json:"source_revision"`
	ContextPolicy  string             `json:"context_policy"`
	Budget         BudgetReport       `json:"budget"`
}

type BudgetReport struct {
	Method                   string `json:"method"`
	DispatchBoundTokens      int    `json:"dispatch_bound_tokens"`
	StateQuestionBoundTokens int    `json:"state_question_bound_tokens"`
	EstimatedTokens          int    `json:"estimated_tokens"`
	TargetTokens             int    `json:"target_tokens"`
}
type BudgetCounter interface {
	Count(RequestV2) (BudgetReport, error)
}
type ByteBudgetCounter struct{ TargetTokens int }

func (c ByteBudgetCounter) Count(r RequestV2) (BudgetReport, error) {
	full, err := json.Marshal(r)
	if err != nil {
		return BudgetReport{}, err
	}
	state, err := json.Marshal(r.State)
	if err != nil {
		return BudgetReport{}, err
	}
	longest := 0
	for _, q := range r.Questions {
		b, e := json.Marshal(q)
		if e != nil {
			return BudgetReport{}, e
		}
		if len(b) > longest {
			longest = len(b)
		}
	}
	target := c.TargetTokens
	if target == 0 {
		target = DefaultTargetInputTokens
	}
	if target < 1 || target > RequestCeiling {
		return BudgetReport{}, errors.New("assess: invalid target token budget")
	}
	// One token per UTF-8 byte is a conservative dispatch bound, even for Unicode.
	return BudgetReport{Method: "utf8-bytes-upper-bound-v1", DispatchBoundTokens: len(full), StateQuestionBoundTokens: len(state) + longest, EstimatedTokens: (len(full) + 3) / 4, TargetTokens: target}, nil
}
func (b BudgetReport) WithinCeilings() bool {
	return b.DispatchBoundTokens <= RequestCeiling && b.StateQuestionBoundTokens <= StateQuestionCeiling
}

var urlPattern = regexp.MustCompile(`(?i)https?://[^\s"']+`)
var quotedPath = regexp.MustCompile(`"(?:/[^"]+|[A-Za-z]:\\[^"]+)"|'(?:/[^']+|[A-Za-z]:\\[^']+)'|` + "`" + `(?:/[^` + "`" + `]+|[A-Za-z]:\\[^` + "`" + `]+)` + "`" + ``)
var barePath = regexp.MustCompile(`(?:^|[\s(=:` + "`" + `])((?:/[^\s,;!?)"'` + "`" + `]+)|(?:[A-Za-z]:\\[^\s,;!?)"'` + "`" + `]+))`)
var opaqueIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)

func pathToken(path string) string {
	sum := sha256.Sum256([]byte(path))
	return "[PATH:" + hex.EncodeToString(sum[:6]) + "]"
}
func cleanNonURL(s string) string {
	s = quotedPath.ReplaceAllStringFunc(s, func(v string) string { return pathToken(v[1 : len(v)-1]) })
	return barePath.ReplaceAllStringFunc(s, func(v string) string {
		prefix := v[:1]
		path := v[1:]
		if strings.HasPrefix(v, "/") || regexp.MustCompile(`^[A-Za-z]:`).MatchString(v) {
			prefix = ""
			path = v
		}
		return prefix + pathToken(path)
	})
}
func clean(s string) string {
	s = distill.Redact(s)
	indexes := urlPattern.FindAllStringIndex(s, -1)
	if len(indexes) == 0 {
		return cleanNonURL(s)
	}
	var out strings.Builder
	last := 0
	for _, span := range indexes {
		out.WriteString(cleanNonURL(s[last:span[0]]))
		out.WriteString(s[span[0]:span[1]])
		last = span[1]
	}
	out.WriteString(cleanNonURL(s[last:]))
	return out.String()
}
func safeID(s string) bool {
	return s != "" && len(s) <= 128 && opaqueIDPattern.MatchString(s) && distill.Redact(s) == s && !strings.Contains(strings.ToLower(s), "secret")
}
func validTime(s string) bool {
	if s == "" {
		return true
	}
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}
func sanitizeRecord(r SourceRecord) (SourceRecord, error) {
	if !safeID(r.ID) {
		return r, errors.New("assess: source ID must be opaque")
	}
	r.Text = clean(r.Text)
	r.Speaker = clean(r.Speaker)
	return r, nil
}
func sanitizeContext(c Context) (Context, error) {
	c.SourceHistory = append([]SourceRecord(nil), c.SourceHistory...)
	c.DerivedContext = append([]DerivedRecord(nil), c.DerivedContext...)
	c.EvaluationScope.SourceGaps = append([]string(nil), c.EvaluationScope.SourceGaps...)
	if !validTime(c.TargetEpisode.OccurredAt) || !validTime(c.EvaluationScope.TargetTime) {
		return c, errors.New("assess: invalid target timestamp")
	}
	if strings.TrimSpace(c.Candidate.Text) == "" || strings.TrimSpace(c.TargetEpisode.Text) == "" {
		return c, errors.New("assess: candidate and target episode must be nonempty")
	}
	var err error
	c.TargetEpisode, err = sanitizeRecord(c.TargetEpisode)
	if err != nil {
		return c, err
	}
	for i := range c.SourceHistory {
		c.SourceHistory[i], err = sanitizeRecord(c.SourceHistory[i])
		if err != nil {
			return c, err
		}
		if !validTime(c.SourceHistory[i].OccurredAt) {
			return c, errors.New("assess: invalid history timestamp")
		}
	}
	for i := range c.DerivedContext {
		d := &c.DerivedContext[i]
		if !safeID(d.ID) {
			return c, errors.New("assess: derived ID must be opaque")
		}
		d.Text = clean(d.Text)
		d.Provenance = clean(d.Provenance)
		d.Limitation = clean(d.Limitation)
		d.Kind = clean(d.Kind)
	}
	c.Candidate.Text = clean(c.Candidate.Text)
	c.Candidate.SourceMention = clean(c.Candidate.SourceMention)
	c.Candidate.RelationMention = clean(c.Candidate.RelationMention)
	c.Candidate.DestinationMention = clean(c.Candidate.DestinationMention)
	c.Candidate.TemporalClaim = clean(c.Candidate.TemporalClaim)
	c.EvaluationScope.RepositoryScope = clean(c.EvaluationScope.RepositoryScope)
	for i := range c.EvaluationScope.SourceGaps {
		c.EvaluationScope.SourceGaps[i] = clean(c.EvaluationScope.SourceGaps[i])
	}
	c.EvaluationScope.OrderingPolicy = clean(c.EvaluationScope.OrderingPolicy)
	return c, nil
}
