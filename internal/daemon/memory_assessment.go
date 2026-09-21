package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/config"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assesscontext"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/jeffdhooton/scry/internal/memory/assessworker"
	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

type assessmentRuntime struct {
	configuration config.Assessment
	configError   string
	keyAvailable  bool
	client        assessworker.Dispatcher
	once          sync.Once
	store         *assessstore.Store
	storeError    error
	mu            sync.Mutex
	worker        *assessworker.Worker
	workerError   string
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	gaps          map[string]uint64
	scopes        map[string]string
}

func (d *Daemon) configureAssessment() {
	a := &d.assessment
	a.configuration = config.DefaultAssessment()
	a.gaps = map[string]uint64{}
	a.scopes = map[string]string{}
	cfg, e := config.Load(d.scryHome())
	if e != nil {
		a.configError = "invalid memory assessment configuration"
		return
	}
	effective, e := cfg.Memory.Assessment.Effective()
	if e != nil {
		a.configError = e.Error()
		return
	}
	a.configuration = effective
	if cfg.MemorySocket() != "" || os.Getenv("SCRY_MEMORY_SOCKET") != "" {
		a.configuration.Mode = "off"
		return
	}
	a.keyAvailable = strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY")) != ""
	if a.keyAvailable && a.configuration.Mode == "shadow" {
		c, e := assess.NewClient(os.Getenv("TYPESAFE_API_KEY"))
		if e == nil {
			c.SetTimeout(effective.Timeout)
			a.client = c
		}
	}
}
func (d *Daemon) assessmentStore() (*assessstore.Store, error) {
	a := &d.assessment
	a.once.Do(func() {
		a.store, a.storeError = assessstore.Open(filepath.Join(d.scryHome(), "memory-assess"), assessstore.Options{ReadOnly: a.configuration.Mode != "shadow" || a.configError != ""})
	})
	return a.store, a.storeError
}
func (d *Daemon) assessmentGap(reason string) {
	a := &d.assessment
	a.mu.Lock()
	a.gaps[reason]++
	a.mu.Unlock()
	log.Printf("memory assessment: coverage gap (%s)", reason)
}
func (d *Daemon) assessmentMetadata(cwd string, attested bool, namespace string) assessstore.SourceMetadata {
	m := assessstore.SourceMetadata{}
	if !attested || cwd == "" || namespace == "" {
		return m
	}
	a := &d.assessment
	a.mu.Lock()
	scope := a.scopes[namespace+"\x00"+cwd]
	a.mu.Unlock()
	if scope != "" {
		m.RepositoryScope = scope
	} else if namespace != "" {
		m.RepositoryScope = "attested-path:" + namespace + ":" + cwd
	}
	return m
}
func (d *Daemon) captureAssessment(ep distill.RawEpisode) {
	if d.assessment.configuration.Mode != "shadow" || d.assessment.configError != "" {
		return
	}
	if !d.assessmentTrialEligible(ep.OccurredAt) {
		return
	}
	s, e := d.assessmentStore()
	if e != nil {
		d.assessmentGap("source_store_unavailable")
		return
	}
	if ep.OccurredAt.IsZero() {
		ep.OccurredAt = time.Now()
	}
	if _, e = s.CaptureRaw(ep, d.assessmentMetadata(ep.Cwd, ep.CwdIsRepo, ep.SourceNamespace)); e != nil {
		d.assessmentGap("source_capture_failed")
	}
}
func (d *Daemon) observeAssessment(p store.PendingEpisode, r extract.Result) {
	if d.assessment.configuration.Mode != "shadow" || d.assessment.configError != "" {
		return
	}
	sourceTime := p.OccurredAt
	if p.SourceTimeUnknown {
		sourceTime = time.Time{}
	}
	if !d.assessmentTrialEligible(sourceTime) {
		return
	}
	s, e := d.assessmentStore()
	if e != nil {
		d.assessmentGap("job_store_unavailable")
		return
	}
	src, e := s.CapturePending(p, d.assessmentMetadata(p.Cwd, p.CwdIsRepo, p.SourceNamespace))
	if e != nil {
		d.assessmentGap("source_capture_failed")
		return
	}
	if _, e = s.Enqueue(src.ID, r, assessstore.Versions{Model: d.assessment.configuration.Model, Rubric: assess.RubricV2, ContextPolicy: assess.ContextPolicyVersion}); e != nil {
		d.assessmentGap("job_capture_failed")
		return
	}
	d.assessment.mu.Lock()
	w := d.assessment.worker
	d.assessment.mu.Unlock()
	if w != nil {
		w.Kick()
	}
}

// Pre-extracted commits contain a derived summary, not the raw episode. Retain
// their candidates too, with an explicit unavailable-source terminal outcome
// when the worker attempts to construct evidence. Never quote a summary as raw.
func (d *Daemon) observeCommittedAssessment(ep store.Episode, r extract.Result) {
	if d.assessment.configuration.Mode != "shadow" || d.assessment.configError != "" {
		return
	}
	if !d.assessmentTrialEligible(ep.OccurredAt) {
		return
	}
	s, e := d.assessmentStore()
	if e != nil {
		d.assessmentGap("job_store_unavailable")
		return
	}
	src, e := s.Capture(assessstore.Source{EpisodeID: ep.ID, Source: ep.Source, SourceRef: ep.SourceRef, Text: ep.Summary, OccurredAt: ep.OccurredAt, Kind: "derived_summary", SourceUnavailable: true, SourceMetadata: assessstore.SourceMetadata{Cwd: ep.Cwd, CwdIsRepo: ep.CwdIsRepo}})
	if e != nil {
		d.assessmentGap("source_capture_failed")
		return
	}
	if _, e = s.Enqueue(src.ID, r, assessstore.Versions{Model: d.assessment.configuration.Model, Rubric: assess.RubricV2, ContextPolicy: assess.ContextPolicyVersion}); e != nil {
		d.assessmentGap("job_capture_failed")
		return
	}
	d.assessment.mu.Lock()
	w := d.assessment.worker
	d.assessment.mu.Unlock()
	if w != nil {
		w.Kick()
	}
}

// prepareAssessmentHistory snapshots existing summaries outside the extraction
// and intake paths. Legacy graph paths lack a producer namespace and cannot
// establish cross-machine repository identity for source-backed assessment.
func (d *Daemon) prepareAssessmentHistory(ctx context.Context, s *assessstore.Store) {
	graph, e := d.memoryStore()
	if e != nil {
		d.assessmentGap("derived_store_unavailable")
		return
	}
	episodes, e := graph.AllEpisodes()
	if e != nil {
		d.assessmentGap("derived_history_unavailable")
		return
	}
	// Stable import order makes snapshot revision selection reproducible.
	sort.Slice(episodes, func(i, j int) bool {
		if !episodes[i].OccurredAt.Equal(episodes[j].OccurredAt) {
			return episodes[i].OccurredAt.Before(episodes[j].OccurredAt)
		}
		return episodes[i].ID < episodes[j].ID
	})
	for _, ep := range episodes {
		if ctx.Err() != nil {
			return
		}
		if ep.Summary == "" {
			continue
		}
		_, e = s.Capture(assessstore.Source{EpisodeID: ep.ID, Source: ep.Source, SourceRef: ep.SourceRef, Text: ep.Summary, OccurredAt: ep.OccurredAt, Kind: "derived_summary", SourceUnavailable: true, SourceMetadata: assessstore.SourceMetadata{Cwd: ep.Cwd, CwdIsRepo: ep.CwdIsRepo}})
		if e != nil {
			d.assessmentGap("derived_capture_failed")
			return
		}
	}
}
func (d *Daemon) startAssessmentWorker(ctx context.Context) {
	a := &d.assessment
	if a.configuration.Mode != "shadow" || a.configError != "" {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	if a.cancel != nil {
		a.mu.Unlock()
		cancel()
		return
	}
	a.cancel = cancel
	a.mu.Unlock()
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		s, e := d.assessmentStore()
		if e != nil {
			d.assessmentGap("worker_store_unavailable")
			return
		}
		if a.configuration.Trial == nil {
			d.prepareAssessmentHistory(ctx, s)
		}
		if ctx.Err() != nil {
			return
		}
		var trial *assessworker.Trial
		var minSourceTime time.Time
		if t := a.configuration.Trial; t != nil {
			trial = &assessworker.Trial{StartsAt: t.StartsAt, ExpiresAt: t.ExpiresAt, MaxRequests: t.MaxRequests}
			minSourceTime = t.StartsAt
		}
		w, e := assessworker.New(assessworker.Options{Store: s, Builder: assesscontext.Builder{Sources: s, Counter: assess.ByteBudgetCounter{TargetTokens: a.configuration.TargetInputTokens}, MinSourceTime: minSourceTime}, Client: a.client, Concurrency: a.configuration.Concurrency, Timeout: a.configuration.Timeout, Trial: trial})
		if e != nil {
			d.assessmentGap("worker_configuration_failed")
			return
		}
		a.mu.Lock()
		a.worker = w
		a.mu.Unlock()
		e = w.Run(ctx)
		a.mu.Lock()
		a.worker = nil
		if e != nil && ctx.Err() == nil {
			a.workerError = "worker_stopped_restart_required"
		}
		a.mu.Unlock()
		if e != nil && ctx.Err() == nil {
			d.assessmentGap("worker_stopped")
		}
	}()
}
func (d *Daemon) closeAssessment() {
	a := &d.assessment
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	a.wg.Wait()
	if a.store != nil {
		_ = a.store.Close()
	}
}

func (d *Daemon) registerAssessmentMethods() {
	d.server.Register("memory.assess.status", d.handleAssessmentStatus)
	d.server.Register("memory.assess.list", d.handleAssessmentList)
	d.server.Register("memory.assess.show", d.handleAssessmentShow)
	d.server.Register("memory.assess.replay", d.handleAssessmentReplay)
	d.server.Register("memory.assess.resume", d.handleAssessmentResume)
}

type AssessmentStatusResult struct {
	Configuration         config.Assessment   `json:"configuration"`
	ConfigurationError    string              `json:"configuration_error,omitempty"`
	KeyAvailable          bool                `json:"key_available"`
	BlockedReason         string              `json:"blocked_reason,omitempty"`
	CredentialRequirement string              `json:"credential_requirement,omitempty"`
	Store                 *assessstore.Status `json:"store,omitempty"`
	CoverageGaps          map[string]uint64   `json:"process_coverage_gaps"`
	SourceAvailable       int                 `json:"source_available"`
	SourceUnavailable     int                 `json:"source_unavailable"`
	Truncated             int                 `json:"truncated"`
	InputTokens           int64               `json:"input_tokens"`
	DailyInputTokens      int64               `json:"daily_input_tokens"`
	EstimatedCostUSD      float64             `json:"estimated_cost_usd"`
	HTTPP50MS             float64             `json:"http_p50_ms"`
	HTTPP95MS             float64             `json:"http_p95_ms"`
	EndToEndP50MS         float64             `json:"end_to_end_p50_ms"`
	EndToEndP95MS         float64             `json:"end_to_end_p95_ms"`
	OldestQueueAgeSeconds float64             `json:"oldest_queue_age_seconds"`
	MetricsTruncated      bool                `json:"metrics_truncated"`
	ErrorCounts           map[string]int      `json:"error_counts"`
}

func percentile(x []float64, p float64) float64 {
	if len(x) == 0 {
		return 0
	}
	sort.Float64s(x)
	i := int(math.Ceil(float64(len(x))*p)) - 1
	if i < 0 {
		i = 0
	}
	return x[i]
}
func (d *Daemon) handleAssessmentStatus(ctx context.Context, _ json.RawMessage) (any, error) {
	a := &d.assessment
	out := &AssessmentStatusResult{Configuration: a.configuration, ConfigurationError: a.configError, KeyAvailable: a.keyAvailable, CoverageGaps: map[string]uint64{}, ErrorCounts: map[string]int{}}
	a.mu.Lock()
	for k, v := range a.gaps {
		out.CoverageGaps[k] = v
	}
	if a.workerError != "" {
		out.BlockedReason = a.workerError
	}
	a.mu.Unlock()
	if a.configError != "" {
		out.BlockedReason = "invalid_configuration"
	}
	if a.configuration.Mode == "shadow" && !a.keyAvailable {
		out.BlockedReason = "missing_credentials"
		out.CredentialRequirement = "Set TYPESAFE_API_KEY in the daemon environment, restart, then explicitly resume assessment."
	}
	s, e := d.assessmentStore()
	if e != nil {
		if os.IsNotExist(e) && a.configuration.Mode == "off" {
			return out, nil
		}
		out.BlockedReason = "sidecar_unavailable"
		return out, nil
	}
	status, e := s.Status()
	if e != nil {
		return nil, errors.New("assessment status unavailable")
	}
	out.Store = &status
	if status.BlockedReason != "" {
		out.BlockedReason = status.BlockedReason
	}
	if t := a.configuration.Trial; t != nil && out.BlockedReason == "" {
		switch {
		case time.Now().Before(t.StartsAt):
			out.BlockedReason = "trial_not_started"
		case !time.Now().Before(t.ExpiresAt):
			out.BlockedReason = "trial_expired"
		case status.Dispatches >= t.MaxRequests:
			out.BlockedReason = "trial_request_limit"
		}
	}
	after := ""
	count := 0
	var httpLatency, e2e []float64
	now := time.Now()
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		page, e := s.List(assessstore.ListQuery{After: after, Limit: 200})
		if e != nil {
			return nil, errors.New("assessment metrics unavailable")
		}
		for _, j := range page.Jobs {
			count++
			if j.Error != "" {
				out.ErrorCounts[j.Error]++
			}
			if j.EvidenceUnavailable || j.TargetSourceUnavailable || j.Error == "evidence_unavailable" {
				out.SourceUnavailable++
			} else {
				out.SourceAvailable++
			}
			out.Truncated += len(j.Manifest.Truncations)
			if j.Status == assessstore.Pending || j.Status == assessstore.Running {
				age := now.Sub(j.CreatedAt).Seconds()
				if age > out.OldestQueueAgeSeconds {
					out.OldestQueueAgeSeconds = age
				}
			}
			if j.Assessment != nil {
				out.InputTokens += int64(j.Assessment.Usage.InputTokens)
				httpLatency = append(httpLatency, j.Assessment.LatencyMS)
				e2e = append(e2e, float64(j.UpdatedAt.Sub(j.CreatedAt).Microseconds())/1000)
				if j.UpdatedAt.UTC().Format("2006-01-02") == now.UTC().Format("2006-01-02") {
					out.DailyInputTokens += int64(j.Assessment.Usage.InputTokens)
				}
			}
		}
		if page.Next == "" {
			break
		}
		if count >= 10000 {
			out.MetricsTruncated = true
			break
		}
		after = page.Next
	}
	out.HTTPP50MS = percentile(httpLatency, .5)
	out.HTTPP95MS = percentile(httpLatency, .95)
	out.EndToEndP50MS = percentile(e2e, .5)
	out.EndToEndP95MS = percentile(e2e, .95)
	out.EstimatedCostUSD = float64(out.InputTokens) * 0.042 / 1e6
	return out, nil
}

type AssessmentListParams struct {
	EpisodeID string `json:"episode_id"`
	After     string `json:"after"`
	Limit     int    `json:"limit"`
}

func (d *Daemon) handleAssessmentList(_ context.Context, raw json.RawMessage) (any, error) {
	var p AssessmentListParams
	if len(raw) > 0 && json.Unmarshal(raw, &p) != nil {
		return nil, errors.New("invalid assessment list parameters")
	}
	s, e := d.assessmentStore()
	if e != nil {
		return nil, errors.New("assessment sidecar unavailable")
	}
	return s.List(assessstore.ListQuery{EpisodeID: p.EpisodeID, After: p.After, Limit: p.Limit})
}

type AssessmentShowParams struct {
	ID             string `json:"id"`
	IncludeContext bool   `json:"include_context"`
}
type AssessmentShowResult struct {
	Job     assessstore.Job      `json:"job"`
	Request json.RawMessage      `json:"request,omitempty"`
	Sources []assessstore.Source `json:"sources,omitempty"`
}

func (d *Daemon) handleAssessmentShow(_ context.Context, raw json.RawMessage) (any, error) {
	var p AssessmentShowParams
	if json.Unmarshal(raw, &p) != nil || p.ID == "" {
		return nil, errors.New("assessment id required")
	}
	s, e := d.assessmentStore()
	if e != nil {
		return nil, errors.New("assessment sidecar unavailable")
	}
	j, e := s.GetJob(p.ID)
	if e != nil {
		return nil, e
	}
	out := AssessmentShowResult{Job: j}
	out.Job.Packet = nil
	out.Job.Extraction = nil
	if p.IncludeContext {
		out.Request = json.RawMessage(j.Packet)
		out.Job.Extraction = j.Extraction
		ids := []string{j.SourceID}
		for _, ref := range j.Manifest.Sources {
			if ref.ID != j.SourceID {
				ids = append(ids, ref.ID)
			}
		}
		for _, id := range ids {
			src, e := s.GetSource(id)
			if e == nil {
				out.Sources = append(out.Sources, src)
			} else {
				out.Sources = append(out.Sources, assessstore.Source{ID: id, Unavailable: true, SourceUnavailable: true})
			}
		}
	}
	return &out, nil
}

type AssessmentReplayParams struct {
	ID   string `json:"id"`
	Live bool   `json:"live"`
}

func (d *Daemon) handleAssessmentReplay(_ context.Context, raw json.RawMessage) (any, error) {
	var p AssessmentReplayParams
	if json.Unmarshal(raw, &p) != nil || p.ID == "" {
		return nil, errors.New("assessment id required")
	}
	s, e := d.assessmentStore()
	if e != nil {
		return nil, errors.New("assessment sidecar unavailable")
	}
	j, e := s.GetJob(p.ID)
	if e != nil {
		return nil, e
	}
	if j.EvidenceUnavailable || len(j.Packet) == 0 {
		return nil, assessstore.ErrEvidenceUnavailable
	}
	h := sha256.Sum256(j.Packet)
	if hex.EncodeToString(h[:]) != j.PacketHash {
		return nil, errors.New("assessment packet integrity failure")
	}
	if !p.Live {
		return map[string]any{"mode": "preview", "another_charge": true, "parent_id": j.ID, "packet_hash": j.PacketHash, "request": json.RawMessage(j.Packet), "manifest": j.Manifest}, nil
	}
	if d.assessment.configuration.Mode != "shadow" || d.assessment.configError != "" {
		return nil, errors.New("live replay requires valid shadow configuration")
	}
	replay, e := s.Replay(j.ID)
	if e != nil {
		return nil, e
	}
	d.assessment.mu.Lock()
	w := d.assessment.worker
	d.assessment.mu.Unlock()
	if w != nil {
		w.Kick()
	}
	replay.Packet = nil
	replay.Extraction = nil
	return &replay, nil
}
func (d *Daemon) handleAssessmentResume(_ context.Context, _ json.RawMessage) (any, error) {
	a := &d.assessment
	if a.configError != "" {
		return nil, fmt.Errorf("assessment: %s", a.configError)
	}
	if a.configuration.Mode != "shadow" {
		return nil, errors.New("assessment resume requires shadow mode")
	}
	if !a.keyAvailable {
		return nil, errors.New("TYPESAFE_API_KEY is required in the daemon environment; restart after configuring it")
	}
	a.mu.Lock()
	w := a.worker
	a.mu.Unlock()
	if w == nil {
		return nil, errors.New("assessment worker is not running")
	}
	if e := w.Resume(); e != nil {
		return nil, e
	}
	return map[string]bool{"resumed": true}, nil
}
