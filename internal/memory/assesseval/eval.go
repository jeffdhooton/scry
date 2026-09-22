// Package assesseval evaluates frozen synthetic cases through the production
// source builder, durable worker and sidecar. It never opens a user's memory.
package assesseval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assesscontext"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/jeffdhooton/scry/internal/memory/assessworker"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"math"
	"os"
	"sort"
	"time"
)

type Episode struct {
	HistoryDerived bool      `json:"history_derived"`
	ID             string    `json:"id"`
	Split          string    `json:"split"`
	Source         string    `json:"source"`
	History        string    `json:"history"`
	Scope          string    `json:"scope"`
	OccurredAt     time.Time `json:"occurred_at"`
}
type Labels struct {
	Supported bool   `json:"supported"`
	Durable   bool   `json:"durable"`
	Assertion string `json:"assertion"`
}
type Case struct {
	ID        string `json:"id"`
	EpisodeID string `json:"episode_id"`
	Split     string `json:"split"`
	Fact      string `json:"fact"`
	Entity    string `json:"entity"`
	Expected  Labels `json:"expected"`
}
type Corpus struct {
	Version          string    `json:"version"`
	Kind             string    `json:"kind"`
	Reviewer         string    `json:"reviewer"`
	LabelsRecordedAt string    `json:"labels_recorded_at"`
	Episodes         []Episode `json:"episodes"`
	Cases            []Case    `json:"cases"`
	Hash             string    `json:"-"`
}

func Load(path string) (Corpus, error) {
	var c Corpus
	b, e := os.ReadFile(path)
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(b, &c)
	c.Hash = digest(b)
	if e == nil {
		e = c.Validate()
	}
	return c, e
}
func (c Corpus) Validate() error {
	if c.Kind != "synthetic" || c.Reviewer == "" || c.LabelsRecordedAt == "" {
		return errors.New("frozen synthetic review metadata required")
	}
	eps := map[string]Episode{}
	for _, e := range c.Episodes {
		if e.ID == "" || e.Source == "" || e.OccurredAt.IsZero() {
			return errors.New("invalid episode")
		}
		if _, ok := eps[e.ID]; ok {
			return errors.New("duplicate episode")
		}
		eps[e.ID] = e
	}
	ids := map[string]bool{}
	facts := map[string]bool{}
	for _, v := range c.Cases {
		ep, ok := eps[v.EpisodeID]
		if !ok || ep.Split != v.Split || ids[v.ID] || facts[v.Fact] || v.Fact == "" {
			return errors.New("invalid case or split")
		}
		ids[v.ID] = true
		facts[v.Fact] = true
		switch v.Expected.Assertion {
		case "established", "planned", "hypothetical", "denied", "unclear":
		default:
			return errors.New("invalid assertion label")
		}
	}
	return nil
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

type Options struct {
	Mode          string
	Arms, Rubrics []string
	Client        assessworker.Dispatcher
}
type Binary struct {
	N     int     `json:"n"`
	FP    int     `json:"fp"`
	FN    int     `json:"fn"`
	Brier float64 `json:"brier"`
}

func (m *Binary) Add(p float64, label bool) {
	m.N++
	y := 0.
	if label {
		y = 1
	}
	m.Brier += (p - y) * (p - y)
	if p >= .5 && !label {
		m.FP++
	}
	if p < .5 && label {
		m.FN++
	}
}
func (m *Binary) Finalize() {
	if m.N > 0 {
		m.Brier /= float64(m.N)
	}
}

type Metrics struct {
	Supported          Binary                    `json:"supported"`
	Durable            Binary                    `json:"durable"`
	AssertionConfusion map[string]map[string]int `json:"assertion_confusion"`
}
type Result struct {
	CompletedAt   time.Time          `json:"completed_at"`
	CaseID        string             `json:"case_id"`
	Split         string             `json:"split"`
	Arm           string             `json:"arm"`
	Rubric        string             `json:"rubric"`
	Status        assessstore.State  `json:"status"`
	Error         string             `json:"error,omitempty"`
	PacketHash    string             `json:"packet_hash"`
	Packet        json.RawMessage    `json:"packet,omitempty"`
	Manifest      assess.Manifest    `json:"manifest"`
	Assessment    *assess.Assessment `json:"assessment,omitempty"`
	Expected      Labels             `json:"expected"`
	Disagreements []string           `json:"disagreements,omitempty"`
	EndToEndMS    float64            `json:"end_to_end_ms"`
}
type Observation struct {
	Arm                string             `json:"arm"`
	Rubric             string             `json:"rubric"`
	Phase              string             `json:"phase"`
	Status             assessstore.Status `json:"store_status"`
	OldestPendingAgeMS float64            `json:"oldest_pending_age_ms"`
}
type Report struct {
	PlannedRequests  int            `json:"planned_requests"`
	Observations     []Observation  `json:"queue_observations"`
	EndToEndP50      float64        `json:"end_to_end_p50_ms"`
	EndToEndP95      float64        `json:"end_to_end_p95_ms"`
	FailureRate      float64        `json:"terminal_failure_rate"`
	ErrorCounts      map[string]int `json:"typed_error_counts"`
	Exclusions       map[string]int `json:"retrieval_exclusions"`
	DailyInputTokens map[string]int `json:"daily_returned_input_tokens,omitempty"`

	Mode              string              `json:"mode"`
	QualityClaim      string              `json:"quality_claim"`
	SelectedDigest    string              `json:"selected_cases_sha256"`
	CorpusHash        string              `json:"corpus_sha256"`
	SyntheticCases    int                 `json:"synthetic_cases"`
	RealReviewedCases int                 `json:"real_reviewed_cases"`
	Requested         int                 `json:"requested"`
	Counts            map[string]int      `json:"completion_counts"`
	Metrics           map[string]*Metrics `json:"metrics_by_rubric_arm_split"`
	Results           []Result            `json:"results"`
	InputTokens       *int                `json:"returned_input_tokens,omitempty"`
	OutputTokens      *int                `json:"returned_output_tokens,omitempty"`
	EstimatedUSD      *float64            `json:"estimated_usd_from_returned_usage,omitempty"`
	ElapsedMS         float64             `json:"elapsed_ms"`
	Throughput        float64             `json:"completed_per_second"`
	HTTPP50           float64             `json:"http_p50_ms"`
	HTTPP95           float64             `json:"http_p95_ms"`
	HistoryCovered    int                 `json:"packets_with_history"`
	MissingRaw        int                 `json:"missing_raw_records"`
	Truncations       int                 `json:"truncated_records"`
	Underfilled       int                 `json:"underfilled_packets"`
}

// reader keeps compact selection inside the real builder, exposing only core.
type reader struct {
	*assessstore.Store
	compact bool
}

func (r reader) Sources(q assessstore.SourceQuery) (assessstore.SourcePage, error) {
	if r.compact {
		return assessstore.SourcePage{}, nil
	}
	return r.Store.Sources(q)
}

type builder struct {
	real   assesscontext.Builder
	rubric string
}

func (b builder) Build(ctx context.Context, j assessstore.Job) (assesscontext.Packet, error) {
	j.Versions.Rubric = assess.RubricV2
	p, e := b.real.Build(ctx, j)
	if e != nil || b.rubric == assess.RubricV2 {
		return p, e
	}
	var v assess.RequestV2
	if e = json.Unmarshal(p.Request, &v); e != nil {
		return p, e
	}
	state, e := json.Marshal(struct {
		Target  assess.SourceRecord    `json:"target_episode"`
		History []assess.SourceRecord  `json:"source_history"`
		Derived []assess.DerivedRecord `json:"derived_context"`
		Scope   assess.EvaluationScope `json:"evaluation_scope"`
	}{v.State.TargetEpisode, v.State.SourceHistory, v.State.DerivedContext, v.State.EvaluationScope})
	if e != nil {
		return p, e
	}
	req, e := assess.BuildRequest(string(state), v.State.Candidate.Text)
	if e != nil {
		return p, e
	}
	p.Request, e = json.Marshal(req)
	if e != nil {
		return p, e
	}
	s, _ := json.Marshal(req.State)
	longest := 0
	for _, q := range req.Questions {
		v, _ := json.Marshal(q)
		if len(v) > longest {
			longest = len(v)
		}
	}
	p.Manifest.Budget.DispatchBoundTokens = len(p.Request)
	p.Manifest.Budget.StateQuestionBoundTokens = len(s) + longest
	p.Manifest.Budget.EstimatedTokens = (len(p.Request) + 3) / 4
	if !p.Manifest.Budget.WithinCeilings() {
		return p, assesscontext.ErrOversize
	}
	return p, nil
}

type preview struct{}

func (preview) Dispatch(context.Context, []byte, string) (assess.Assessment, error) {
	return assess.Assessment{}, errors.New("preview_only")
}

// ScriptedMock depends exclusively on request bytes, never reviewer labels.
type ScriptedMock struct{}

func (ScriptedMock) Dispatch(_ context.Context, b []byte, model string) (assess.Assessment, error) {
	h := sha256.Sum256(b)
	s := float64(h[0]) / 255
	d := float64(h[1]) / 255
	opts := []string{"established", "planned", "hypothetical", "denied", "unclear"}
	choice := opts[int(h[2])%5]
	confidence := .6
	dist := map[string]*float64{}
	for _, v := range opts {
		p := .1
		if v == choice {
			p = .6
		}
		dist[v] = &p
	}
	return assess.Assessment{Model: model, Answers: map[string]assess.Answer{"supported": {Type: "noul", Noul: &s}, "durable": {Type: "noul", Noul: &d}, "assertion": {Type: "choice", Choice: choice, Confidence: &confidence, Probabilities: dist}}}, nil
}
func Run(ctx context.Context, c Corpus, o Options) (r Report, err error) {
	r = Report{Mode: o.Mode, CorpusHash: c.Hash, SyntheticCases: len(c.Cases), Counts: map[string]int{}, Metrics: map[string]*Metrics{}, QualityClaim: "No model-quality claim: preview or scripted integration mock."}
	selected, _ := json.Marshal(struct {
		Episodes []Episode `json:"episodes"`
		Cases    []Case    `json:"cases"`
	}{c.Episodes, c.Cases})
	r.SelectedDigest = digest(selected)
	if e := c.Validate(); e != nil {
		return r, e
	}
	switch o.Mode {
	case "preview":
		o.Client = preview{}
	case "mock":
		o.Client = ScriptedMock{}
	case "live":
		if o.Client == nil {
			return r, errors.New("live dispatcher required")
		}
		r.QualityClaim = "Synthetic evaluation only; agent-authored labels await independent review."
		n, m := 0, 0
		cost := 0.
		r.InputTokens = &n
		r.OutputTokens = &m
		r.EstimatedUSD = &cost
	default:
		return r, errors.New("mode must be preview, mock or live")
	}
	if len(o.Arms) == 0 {
		o.Arms = []string{"compact", "relevant", "target20k"}
	}
	if len(o.Rubrics) == 0 {
		o.Rubrics = []string{assess.RubricVersion, assess.RubricV2}
	}
	r.PlannedRequests = len(c.Cases) * len(o.Arms) * len(o.Rubrics)
	start := time.Now()
	defer func() { r.ElapsedMS = float64(time.Since(start)) / float64(time.Millisecond); r.summarize() }()
	for _, rubric := range o.Rubrics {
		if rubric != assess.RubricVersion && rubric != assess.RubricV2 {
			return r, errors.New("unsupported rubric")
		}
		for _, arm := range o.Arms {
			if arm != "compact" && arm != "relevant" && arm != "target20k" {
				return r, errors.New("unsupported arm")
			}
			rows, e := runArm(ctx, c, o, arm, rubric, &r)
			r.Results = append(r.Results, rows...)
			if e != nil {
				return r, e
			}
		}
	}
	return r, nil
}
func runArm(ctx context.Context, c Corpus, o Options, arm, rubric string, report *Report) ([]Result, error) {
	dir, e := os.MkdirTemp("", "scry-assesseval-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	s, e := assessstore.Open(dir, assessstore.Options{})
	if e != nil {
		return nil, e
	}
	defer s.Close()
	sources := map[string]assessstore.Source{}
	for _, ep := range c.Episodes {
		meta := assessstore.SourceMetadata{Namespace: "synthetic", SessionID: ep.ID, RepositoryScope: ep.Scope, OrderKnown: true, Order: 1}
		if ep.History != "" {
			_, e = s.Capture(assessstore.Source{Kind: historyKind(ep.HistoryDerived), SourceUnavailable: ep.HistoryDerived, EpisodeID: ep.ID + "-prior", Source: "manual", SourceRef: ep.ID + "-prior", Text: ep.History, OccurredAt: ep.OccurredAt.Add(-time.Hour), SourceMetadata: meta})
			if e != nil {
				return nil, e
			}
		}
		meta.Order = 2
		v, e := s.Capture(assessstore.Source{EpisodeID: ep.ID, Source: "manual", SourceRef: ep.ID, Text: ep.Source, OccurredAt: ep.OccurredAt, SourceMetadata: meta})
		if e != nil {
			return nil, e
		}
		sources[ep.ID] = v
	}
	ids := make([]string, len(c.Cases))
	for i, v := range c.Cases {
		jobs, e := s.Enqueue(sources[v.EpisodeID].ID, extract.Result{Facts: []extract.Fct{{Src: v.Entity, Relation: "states", Fact: v.Fact, Confidence: .987654321}}}, assessstore.Versions{Model: assess.Model, Rubric: rubric, ContextPolicy: assess.ContextPolicyVersion})
		if e != nil {
			return nil, e
		}
		ids[i] = jobs[0].ID
	}
	observe := func(phase string) {
		st, err := s.Status()
		if err != nil {
			return
		}
		age := 0.
		for _, id := range ids {
			j, err := s.GetJob(id)
			if err == nil && (j.Status == assessstore.Pending || j.Status == assessstore.Running) {
				a := float64(time.Since(j.CreatedAt)) / float64(time.Millisecond)
				if a > age {
					age = a
				}
			}
		}
		report.Observations = append(report.Observations, Observation{arm, rubric, phase, st, age})
	}
	observe("enqueued")
	defer observe("final")
	target := 20000
	if arm == "relevant" {
		target = 2500
	}
	b := builder{assesscontext.Builder{Sources: reader{s, arm == "compact"}, Counter: assess.ByteBudgetCounter{TargetTokens: target}}, rubric}
	w, e := assessworker.New(assessworker.Options{Store: s, Builder: b, Client: o.Client, Concurrency: 2, PollInterval: time.Millisecond})
	if e != nil {
		return nil, e
	}
	work, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- w.Run(work) }()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	var workerErr error
wait:
	for {
		select {
		case workerErr = <-done:
			break wait
		case <-ctx.Done():
			cancel()
			workerErr = <-done
			break wait
		case <-ticker.C:
			st, err := s.Status()
			if err != nil {
				workerErr = err
				cancel()
				<-done
				break wait
			}
			if st.Counts[assessstore.Pending]+st.Counts[assessstore.Running] == 0 || st.BlockedReason != "" {
				cancel()
				workerErr = <-done
				if st.BlockedReason != "" {
					workerErr = errors.New("provider dispatch blocked; entire evaluation stopped")
				}
				break wait
			}
		}
	}
	cancel()
	if ctx.Err() != nil {
		workerErr = ctx.Err()
	}
	rows := []Result{}
	for i, id := range ids {
		j, e := s.GetJob(id)
		if e != nil {
			return rows, e
		}
		v := c.Cases[i]
		rows = append(rows, Result{CompletedAt: j.UpdatedAt, CaseID: v.ID, Split: v.Split, Arm: arm, Rubric: rubric, Status: j.Status, Error: j.Error, PacketHash: j.PacketHash, Packet: j.Packet, Manifest: j.Manifest, Assessment: j.Assessment, Expected: v.Expected, EndToEndMS: float64(j.UpdatedAt.Sub(j.CreatedAt)) / float64(time.Millisecond)})
	}
	return rows, workerErr
}
func (r *Report) summarize() {
	var lat, e2e []float64
	r.ErrorCounts = map[string]int{}
	r.Exclusions = map[string]int{}
	if r.Mode == "live" {
		r.DailyInputTokens = map[string]int{}
	}
	for i := range r.Results {
		v := &r.Results[i]
		r.Requested++
		status := string(v.Status)
		if r.Mode == "preview" && len(v.Packet) > 0 && v.Assessment == nil {
			status = "previewed"
		}
		r.Counts[status]++
		if v.Error != "" && status != "previewed" {
			r.ErrorCounts[v.Error]++
		}
		for _, ex := range v.Manifest.Excluded {
			r.Exclusions[ex.Reason]++
		}
		if v.Status != assessstore.Pending && v.Status != assessstore.Running {
			e2e = append(e2e, v.EndToEndMS)
		}
		if len(v.Manifest.Sources) > 1 {
			r.HistoryCovered++
		}
		r.MissingRaw += len(v.Manifest.MissingRaw)
		r.Truncations += len(v.Manifest.Truncations)
		if v.Manifest.Budget.EstimatedTokens < v.Manifest.Budget.TargetTokens {
			r.Underfilled++
		}
		if v.Assessment == nil {
			continue
		}
		a := v.Assessment
		lat = append(lat, a.LatencyMS)
		if r.Mode == "live" {
			*r.InputTokens += a.Usage.InputTokens
			r.DailyInputTokens[v.CompletedAt.UTC().Format("2006-01-02")] += a.Usage.InputTokens
			*r.OutputTokens += a.Usage.OutputTokens
		}
		key := fmt.Sprintf("%s/%s/%s", v.Rubric, v.Arm, v.Split)
		m := r.Metrics[key]
		if m == nil {
			m = &Metrics{AssertionConfusion: map[string]map[string]int{}}
			r.Metrics[key] = m
		}
		s, d := *a.Answers["supported"].Noul, *a.Answers["durable"].Noul
		as := a.Answers["assertion"].Choice
		m.Supported.Add(s, v.Expected.Supported)
		m.Durable.Add(d, v.Expected.Durable)
		if m.AssertionConfusion[v.Expected.Assertion] == nil {
			m.AssertionConfusion[v.Expected.Assertion] = map[string]int{}
		}
		m.AssertionConfusion[v.Expected.Assertion][as]++
		if (s >= .5) != v.Expected.Supported {
			v.Disagreements = append(v.Disagreements, "supported")
		}
		if (d >= .5) != v.Expected.Durable {
			v.Disagreements = append(v.Disagreements, "retention_durability")
		}
		if as != v.Expected.Assertion {
			v.Disagreements = append(v.Disagreements, "assertion")
		}
	}
	for _, m := range r.Metrics {
		m.Supported.Finalize()
		m.Durable.Finalize()
	}
	sort.Float64s(e2e)
	if len(e2e) > 0 {
		r.EndToEndP50 = e2e[(len(e2e)-1)/2]
		r.EndToEndP95 = e2e[int(math.Ceil(float64(len(e2e))*.95))-1]
	}
	if r.Requested > 0 {
		r.FailureRate = float64(r.Counts["failed"]+r.Counts["blocked"]+r.Counts["oversize"]) / float64(r.Requested)
	}
	sort.Float64s(lat)
	if len(lat) > 0 {
		r.HTTPP50 = lat[(len(lat)-1)/2]
		r.HTTPP95 = lat[int(math.Ceil(float64(len(lat))*.95))-1]
	}
	if r.ElapsedMS > 0 {
		r.Throughput = float64(r.Counts["completed"]) * 1000 / r.ElapsedMS
	}
	if r.Mode == "live" {
		*r.EstimatedUSD = float64(*r.InputTokens) * assess.InputUSDPerMillion / 1e6
	}
}

func historyKind(derived bool) string {
	if derived {
		return "derived_summary"
	}
	return "raw_source"
}
