package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Options configures a local, opt-in review service. Limits apply to all repos
// together. Reservations include failures and survive daemon restarts.
type Options struct {
	Enabled             bool          `json:"enabled"`
	Repos               []string      `json:"repos"`
	QuietPeriod         time.Duration `json:"quiet_period"`
	PollInterval        time.Duration `json:"poll_interval"`
	Timeout             time.Duration `json:"timeout"`
	MaxInputBytes       int           `json:"max_input_bytes"`
	MaxOutputTokens     int           `json:"max_output_tokens"`
	MaxRequestsPerDay   int           `json:"max_requests_per_day"`
	MaxDailyUSD         float64       `json:"max_daily_usd,omitempty"`
	InputUSDPerMillion  float64       `json:"input_usd_per_million,omitempty"`
	OutputUSDPerMillion float64       `json:"output_usd_per_million,omitempty"`
	Retain              int           `json:"retain"`
	Exclude             []string      `json:"exclude,omitempty"`
	Provider            string        `json:"provider"`
	Model               string        `json:"model"`
}
type Record struct {
	ID               string        `json:"id"`
	Snapshot         Snapshot      `json:"snapshot"`
	ContextID        string        `json:"context_id"`
	State            string        `json:"state"`
	Provisional      bool          `json:"provisional"`
	Freshness        string        `json:"freshness"`
	StartedAt        time.Time     `json:"started_at"`
	CompletedAt      time.Time     `json:"completed_at,omitempty"`
	ElapsedMS        int64         `json:"elapsed_ms"`
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	Output           *ReviewOutput `json:"output,omitempty"`
	Error            string        `json:"error,omitempty"`
	EstimatedCostUSD *float64      `json:"estimated_cost_usd,omitempty"`
	ReservedUSD      float64       `json:"reserved_usd,omitempty"`
	Usage            Usage         `json:"usage"`
}
type observation struct {
	ID    string
	Since time.Time
}
type Enricher func(context.Context, *Snapshot) error
type Service struct {
	mu        sync.Mutex
	runMu     sync.Mutex
	home      string
	opts      Options
	reviewer  Reviewer
	enrich    Enricher
	state     diskState
	observed  map[string]observation
	queued    map[string]bool
	lastCheck map[string]time.Time
	errors    map[string]string
	wake      chan struct{}
}
type Status struct {
	Configuration    Options           `json:"configuration"`
	Ready            bool              `json:"ready"`
	RequestsToday    int               `json:"requests_today"`
	ReservedTodayUSD float64           `json:"reserved_today_usd"`
	Queued           []string          `json:"queued"`
	Errors           map[string]string `json:"errors"`
	Records          int               `json:"records"`
	BlockedReason    string            `json:"blocked_reason,omitempty"`
}

func NewService(home string, opts Options, reviewer Reviewer, enrich Enricher) (*Service, error) {
	if opts.QuietPeriod <= 0 {
		opts.QuietPeriod = 30 * time.Second
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 5 * time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}
	if opts.MaxInputBytes == 0 {
		opts.MaxInputBytes = 48000
	}
	if opts.MaxOutputTokens == 0 {
		opts.MaxOutputTokens = 4096
	}
	if opts.MaxRequestsPerDay == 0 {
		opts.MaxRequestsPerDay = 10
	}
	if opts.Retain == 0 {
		opts.Retain = 100
	}
	if opts.MaxInputBytes < 1024 || opts.MaxInputBytes > 256000 || opts.MaxOutputTokens < 256 || opts.MaxOutputTokens > 16000 || opts.MaxRequestsPerDay < 1 || opts.MaxRequestsPerDay > 1000 || opts.Retain < 1 || opts.Retain > 500 || opts.MaxDailyUSD < 0 || opts.InputUSDPerMillion < 0 || opts.OutputUSDPerMillion < 0 {
		return nil, errors.New("review limits out of range")
	}
	if opts.MaxDailyUSD > 0 && (opts.InputUSDPerMillion <= 0 || opts.OutputUSDPerMillion <= 0) {
		return nil, errors.New("dollar cap requires explicit input and output token rates")
	}
	seen := map[string]bool{}
	repos := []string{}
	for _, p := range opts.Repos {
		if !filepath.IsAbs(p) {
			return nil, errors.New("review repo must be absolute")
		}
		p = filepath.Clean(p)
		if c, e := filepath.EvalSymlinks(p); e == nil {
			p = c
		}
		if !seen[p] {
			seen[p] = true
			repos = append(repos, p)
		}
	}
	opts.Repos = repos
	st, err := loadState(home)
	if err != nil {
		return nil, err
	}
	s := &Service{home: home, opts: opts, reviewer: reviewer, enrich: enrich, state: st, observed: map[string]observation{}, queued: map[string]bool{}, lastCheck: map[string]time.Time{}, errors: map[string]string{}, wake: make(chan struct{}, 1)}
	dirty := false
	for i := range s.state.Records {
		r := &s.state.Records[i]
		if r.State == "running" {
			r.State = "interrupted"
			r.Error = "daemon stopped before a result was saved; reservation retained"
			dirty = true
		}
	}
	if dirty {
		if err = s.saveLocked(); err != nil {
			return nil, err
		}
	}
	return s, nil
}
func (s *Service) allowed(repo string) (string, error) {
	p, e := filepath.Abs(repo)
	if e != nil {
		return "", e
	}
	if c, e := filepath.EvalSymlinks(p); e == nil {
		p = c
	}
	for _, r := range s.opts.Repos {
		if r == p {
			return p, nil
		}
	}
	return "", errors.New("repository is not in review.repos")
}
func (s *Service) Preview(ctx context.Context, repo string) (Snapshot, error) {
	p, e := s.allowed(repo)
	if e != nil {
		return Snapshot{}, e
	}
	snap, e := Capture(ctx, p, CaptureOptions{MaxBytes: s.opts.MaxInputBytes / 2, Exclude: s.opts.Exclude})
	if e != nil {
		return snap, e
	}
	if s.enrich != nil {
		if e = s.enrich(ctx, &snap); e != nil {
			return snap, e
		}
	}
	coverageBudget := min(1024, s.opts.MaxInputBytes/4)
	used := 2 // JSON array brackets; each item also reserves its comma.
	kept := []Evidence{}
	for _, ev := range snap.Evidence {
		b, _ := json.Marshal(ev)
		if used+len(b)+1 > s.opts.MaxInputBytes-coverageBudget {
			snap.Warnings = append(snap.Warnings, "evidence budget reached; some context omitted")
			continue
		}
		kept = append(kept, ev)
		used += len(b) + 1
	}
	snap.Evidence = kept
	if len(snap.Warnings) > 0 {
		// Coverage warnings are evidence too: the model must know when a
		// missing index or budget prevents it from checking the whole change.
		content := "Coverage limitations:\n"
		for _, w := range snap.Warnings {
			if len(content)+len(w)+1 > 700 {
				break
			}
			content += w + "\n"
		}
		ev := Evidence{ID: "coverage", Kind: "coverage", Content: content}
		for {
			b, _ := json.Marshal(ev)
			if len(b) <= coverageBudget {
				break
			}
			runes := []rune(ev.Content)
			if len(runes) == 0 {
				break
			}
			ev.Content = string(runes[:len(runes)-1])
		}
		snap.Evidence = append(snap.Evidence, ev)
	}
	current, e := Fingerprint(ctx, p)
	if e != nil {
		return snap, e
	}
	if current != snap.ID {
		return snap, errors.New("repository changed during evidence assembly")
	}
	return snap, nil
}
func contextID(snap Snapshot) string {
	b, _ := json.Marshal(snap.Evidence)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s *Service) Queue(repo string) (map[string]any, error) {
	p, e := s.allowed(repo)
	if e != nil {
		return nil, e
	}
	if !s.opts.Enabled || s.reviewer == nil {
		return nil, errors.New("background review is not enabled with an available provider")
	}
	s.mu.Lock()
	if s.state.Blocked != "" {
		reason := s.state.Blocked
		s.mu.Unlock()
		return nil, errors.New(reason)
	}
	s.queued[p] = true
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return map[string]any{"repo": p, "state": "queued", "provisional": true}, nil
}
func (s *Service) Run(ctx context.Context, repo string) (Record, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if !s.opts.Enabled || s.reviewer == nil {
		return Record{}, errors.New("background review is disabled or provider unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, s.opts.Timeout)
	defer cancel()
	snap, e := s.Preview(ctx, repo)
	if e != nil {
		return Record{}, e
	}
	if len(snap.ChangedFiles) == 0 {
		return Record{}, errors.New("no uncommitted changes to review")
	}
	hasCode := false
	for _, ev := range snap.Evidence {
		if ev.Kind == "diff" || ev.Kind == "staged_diff" {
			hasCode = true
		}
	}
	if !hasCode {
		return Record{}, errors.New("no reviewable changed-code evidence within the configured limits")
	}
	cid := contextID(snap)
	h := sha256.Sum256([]byte(snap.Repository + "\x00" + snap.ID + "\x00" + cid + "\x00" + s.opts.Provider + "\x00" + s.opts.Model))
	id := hex.EncodeToString(h[:])
	now := time.Now().UTC()
	day := now.Format("2006-01-02")
	reserve := float64(s.opts.MaxInputBytes+8192)*s.opts.InputUSDPerMillion/1e6 + float64(s.opts.MaxOutputTokens)*s.opts.OutputUSDPerMillion/1e6
	s.mu.Lock()
	if s.state.Blocked != "" {
		reason := s.state.Blocked
		s.mu.Unlock()
		return Record{}, errors.New(reason)
	}
	if s.state.Seen[id] {
		for _, r := range s.state.Records {
			if r.ID == id {
				s.mu.Unlock()
				return r, nil
			}
		}
		s.mu.Unlock()
		return Record{}, errors.New("snapshot already reviewed; retained result expired")
	}
	if s.state.Daily[day] >= s.opts.MaxRequestsPerDay {
		s.mu.Unlock()
		return Record{}, errors.New("daily review request cap reached")
	}
	if s.opts.MaxDailyUSD > 0 && s.state.Reserved[day]+reserve > s.opts.MaxDailyUSD {
		s.mu.Unlock()
		return Record{}, errors.New("daily review dollar reservation cap reached")
	}
	rec := Record{ID: id, Snapshot: snap, ContextID: cid, State: "running", Provisional: true, Freshness: "unchecked", StartedAt: now, Provider: s.opts.Provider, Model: s.opts.Model, ReservedUSD: reserve}
	s.state.Daily[day]++
	s.state.Reserved[day] += reserve
	s.state.Seen[id] = true
	s.state.Records = append(s.state.Records, rec)
	if e = s.saveLocked(); e != nil {
		s.mu.Unlock()
		return rec, e
	}
	s.mu.Unlock()
	output, callErr := s.reviewer.Review(ctx, snap)
	rec.Usage = output.Usage
	rec.CompletedAt = time.Now().UTC()
	rec.ElapsedMS = rec.CompletedAt.Sub(now).Milliseconds()
	if output.Usage.Known && s.opts.InputUSDPerMillion > 0 && s.opts.OutputUSDPerMillion > 0 {
		cost := float64(output.Usage.InputTokens)*s.opts.InputUSDPerMillion/1e6 + float64(output.Usage.OutputTokens)*s.opts.OutputUSDPerMillion/1e6
		rec.EstimatedCostUSD = &cost
	}
	if callErr != nil {
		rec.State = "failed"
		rec.Error = callErr.Error()
	} else {
		rec.State = "completed"
		rec.Output = &output
	}
	s.mu.Lock()
	var providerErr *ProviderHTTPError
	if errors.As(callErr, &providerErr) {
		switch providerErr.StatusCode {
		case 401, 402, 403, 429:
			s.state.Blocked = fmt.Sprintf("provider returned HTTP %d; fix availability then run scry review resume (usage is preserved)", providerErr.StatusCode)
		}
	}
	for i := range s.state.Records {
		if s.state.Records[i].ID == id {
			s.state.Records[i] = rec
			break
		}
	}
	e = s.saveLocked()
	s.mu.Unlock()
	if e != nil {
		return rec, e
	}
	return rec, callErr
}

// Tick waits for a stable fingerprint before dispatch. Explicit queue requests
// share this checkpoint so run cannot race a stream of foreground edits.
func (s *Service) Tick(ctx context.Context, now time.Time) error {
	if !s.opts.Enabled || s.reviewer == nil {
		return nil
	}
	s.mu.Lock()
	blocked := s.state.Blocked != ""
	s.mu.Unlock()
	if blocked {
		return nil
	}
	for _, repo := range s.opts.Repos {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fp, e := Fingerprint(ctx, repo)
		if e != nil {
			s.setError(repo, e)
			continue
		}
		s.mu.Lock()
		o, ok := s.observed[repo]
		if !ok || o.ID != fp {
			s.observed[repo] = observation{fp, now}
			delete(s.lastCheck, repo)
			s.mu.Unlock()
			continue
		}
		skip := !s.queued[repo] && !s.lastCheck[repo].IsZero() && now.Sub(s.lastCheck[repo]) < time.Minute
		s.mu.Unlock()
		if skip {
			continue
		}
		if now.Sub(o.Since) < s.opts.QuietPeriod {
			continue
		}
		// Run deduplicates on both source and recalled evidence, allowing a revised
		// supporting decision to trigger a fresh review even with unchanged code.
		_, e = s.Run(ctx, repo)
		s.setError(repo, e)
		s.mu.Lock()
		delete(s.queued, repo)
		s.lastCheck[repo] = now
		s.mu.Unlock()
	}
	return nil
}
func (s *Service) setError(repo string, e error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e == nil {
		delete(s.errors, repo)
	} else {
		s.errors[repo] = e.Error()
	}
}
func (s *Service) Serve(ctx context.Context) {
	timer := time.NewTicker(s.opts.PollInterval)
	defer timer.Stop()
	_ = s.Tick(ctx, time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-timer.C:
			_ = s.Tick(ctx, now)
		case <-s.wake:
			_ = s.Tick(ctx, time.Now())
		}
	}
}
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	day := time.Now().UTC().Format("2006-01-02")
	q := []string{}
	for r := range s.queued {
		q = append(q, r)
	}
	sort.Strings(q)
	errs := map[string]string{}
	for k, v := range s.errors {
		errs[k] = v
	}
	return Status{Configuration: s.opts, Ready: s.opts.Enabled && s.reviewer != nil && s.state.Blocked == "", BlockedReason: s.state.Blocked, RequestsToday: s.state.Daily[day], ReservedTodayUSD: s.state.Reserved[day], Queued: q, Errors: errs, Records: len(s.state.Records)}
}

// Resume is an explicit operator action. Usage and duplicate protection remain.
func (s *Service) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.state.Blocked
	s.state.Blocked = ""
	if e := s.saveLocked(); e != nil {
		s.state.Blocked = old
		return e
	}
	return nil
}
func (s *Service) Get(ctx context.Context, id string) (Record, error) {
	s.mu.Lock()
	var rec Record
	found := false
	for _, r := range s.state.Records {
		if r.ID == id {
			rec = r
			found = true
			break
		}
	}
	s.mu.Unlock()
	if !found {
		return rec, os.ErrNotExist
	}
	rec.Freshness = "unknown"
	snap, e := s.Preview(ctx, rec.Snapshot.Repository)
	if e == nil {
		rec.Freshness = "stale"
		if snap.ID == rec.Snapshot.ID && contextID(snap) == rec.ContextID {
			rec.Freshness = "current"
		}
	}
	return rec, nil
}
func (s *Service) List(ctx context.Context, repo string) ([]Record, error) {
	p, e := s.allowed(repo)
	if e != nil {
		return nil, e
	}
	s.mu.Lock()
	ids := []string{}
	for i := len(s.state.Records) - 1; i >= 0 && len(ids) < 20; i-- {
		if s.state.Records[i].Snapshot.Repository == p {
			ids = append(ids, s.state.Records[i].ID)
		}
	}
	s.mu.Unlock()
	out := []Record{}
	for _, id := range ids {
		r, e := s.Get(ctx, id)
		if e != nil {
			return nil, e
		}
		r.Snapshot.Evidence = nil
		r.Snapshot.Files = nil
		out = append(out, r)
	}
	return out, nil
}
