package review

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type reviewerFunc func(context.Context, Snapshot) (ReviewOutput, error)

func (f reviewerFunc) Review(c context.Context, s Snapshot) (ReviewOutput, error) { return f(c, s) }
func serviceRepo(t *testing.T) string {
	t.Helper()
	p := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
		c := exec.Command("git", args...)
		c.Dir = p
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, b, e)
		}
	}
	os.WriteFile(filepath.Join(p, "permission.go"), []byte("package fixture\nfunc Allowed(special bool) bool { return special }\n"), 0600)
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", "Initial"}} {
		c := exec.Command("git", args...)
		c.Dir = p
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, b, e)
		}
	}
	os.WriteFile(filepath.Join(p, "permission.go"), []byte("package fixture\nfunc Allowed(special bool) bool { return false }\n"), 0600)
	return p
}
func serviceOptions(repo string) Options {
	return Options{Enabled: true, Repos: []string{repo}, QuietPeriod: time.Second, PollInterval: time.Second, Timeout: time.Second, MaxInputBytes: 24000, MaxRequestsPerDay: 1, Retain: 20}
}
func TestServiceDebouncesAndPersistsReservation(t *testing.T) {
	repo := serviceRepo(t)
	home := t.TempDir()
	calls := 0
	r := reviewerFunc(func(context.Context, Snapshot) (ReviewOutput, error) {
		calls++
		return ReviewOutput{Summary: "reviewed"}, nil
	})
	s, e := NewService(home, serviceOptions(repo), r, nil)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	ctx := context.Background()
	if e = s.Tick(ctx, now); e != nil {
		t.Fatal(e)
	}
	if calls != 0 {
		t.Fatal("called before quiet period")
	}
	if e = s.Tick(ctx, now.Add(2*time.Second)); e != nil {
		t.Fatal(e)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if e = s.Tick(ctx, now.Add(3*time.Second)); e != nil {
		t.Fatal(e)
	}
	if calls != 1 {
		t.Fatal("duplicate review")
	}
	s, e = NewService(home, serviceOptions(repo), r, nil)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(repo, "other.go"), []byte("package fixture"), 0600)
	if _, e = s.Run(ctx, repo); e == nil {
		t.Fatal("restart replenished request cap")
	}
	if calls != 1 {
		t.Fatal("cap permitted inference")
	}
}
func TestServiceProvisionalFindingsBecomeStale(t *testing.T) {
	repo := serviceRepo(t)
	memory := "Special caller must retain access"
	enrich := func(_ context.Context, s *Snapshot) error {
		s.Evidence = append(s.Evidence, Evidence{ID: "memory:decision", Kind: "memory", Content: memory})
		return nil
	}
	r := reviewerFunc(func(_ context.Context, s Snapshot) (ReviewOutput, error) {
		return ReviewOutput{Summary: "Exception removed", Findings: []Finding{{Severity: "high", Title: "Special caller loses access", Detail: "The decision requires the exception", EvidenceIDs: []string{"memory:decision"}}}}, nil
	})
	s, e := NewService(t.TempDir(), serviceOptions(repo), r, enrich)
	if e != nil {
		t.Fatal(e)
	}
	rec, e := s.Run(context.Background(), repo)
	if e != nil {
		t.Fatal(e)
	}
	got, e := s.Get(context.Background(), rec.ID)
	if e != nil {
		t.Fatal(e)
	}
	if got.Freshness != "current" || !got.Provisional {
		t.Fatalf("bad record: %+v", got)
	}
	memory = "Decision superseded"
	got, e = s.Get(context.Background(), rec.ID)
	if e != nil {
		t.Fatal(e)
	}
	if got.Freshness != "stale" {
		t.Fatal("changed memory remained current")
	}
	memory = "Special caller must retain access"
	os.WriteFile(filepath.Join(repo, "caller.go"), []byte("package fixture"), 0600)
	got, e = s.Get(context.Background(), rec.ID)
	if e != nil {
		t.Fatal(e)
	}
	if got.Freshness != "stale" {
		t.Fatal("changed code remained current")
	}
}
func TestServiceFailuresCountAndDoNotLookClean(t *testing.T) {
	repo := serviceRepo(t)
	r := reviewerFunc(func(context.Context, Snapshot) (ReviewOutput, error) {
		return ReviewOutput{}, errors.New("provider failed")
	})
	s, e := NewService(t.TempDir(), serviceOptions(repo), r, nil)
	if e != nil {
		t.Fatal(e)
	}
	rec, e := s.Run(context.Background(), repo)
	if e == nil || rec.State != "failed" || rec.Error == "" {
		t.Fatalf("expected visible failed record: %+v %v", rec, e)
	}
	if s.Status().RequestsToday != 1 {
		t.Fatal("failed call not reserved")
	}
	if _, e = s.Run(context.Background(), t.TempDir()); e == nil {
		t.Fatal("allowed unconfigured repo")
	}
}

func TestServiceRefusesIncompleteUsageLedger(t *testing.T) {
	for _, body := range []string{`{}`, `{"version":1,"records":[]}`, `{"version":1,"records":[],"daily":{},"reserved":{}}`} {
		home := t.TempDir()
		if e := os.WriteFile(filepath.Join(home, "state.json"), []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := NewService(home, serviceOptions(t.TempDir()), nil, nil); e == nil {
			t.Fatalf("incomplete ledger accepted: %s", body)
		}
	}
}

func TestServiceFailedUsageAndDollarReservationSurviveRestart(t *testing.T) {
	repo := serviceRepo(t)
	home := t.TempDir()
	opts := serviceOptions(repo)
	opts.MaxRequestsPerDay = 10
	opts.InputUSDPerMillion = 1
	opts.OutputUSDPerMillion = 1
	opts.MaxDailyUSD = .04
	r := reviewerFunc(func(context.Context, Snapshot) (ReviewOutput, error) {
		return ReviewOutput{Usage: Usage{Known: true, InputTokens: 120, OutputTokens: 80}}, errors.New("truncated")
	})
	s, e := NewService(home, opts, r, nil)
	if e != nil {
		t.Fatal(e)
	}
	rec, e := s.Run(context.Background(), repo)
	if e == nil || !rec.Usage.Known || rec.EstimatedCostUSD == nil || *rec.EstimatedCostUSD <= 0 {
		t.Fatalf("failed usage lost: %+v %v", rec, e)
	}
	s, e = NewService(home, opts, r, nil)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(repo, "another.go"), []byte("package fixture"), 0600)
	if _, e = s.Run(context.Background(), repo); e == nil {
		t.Fatal("restarted dollar cap permitted another call")
	}
}
func TestServiceCoverageWarningsReachReviewer(t *testing.T) {
	repo := serviceRepo(t)
	s, e := NewService(t.TempDir(), serviceOptions(repo), nil, func(_ context.Context, s *Snapshot) error {
		s.Warnings = append(s.Warnings, "memory unavailable")
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	snap, e := s.Preview(context.Background(), repo)
	if e != nil {
		t.Fatal(e)
	}
	for _, ev := range snap.Evidence {
		if ev.Kind == "coverage" && strings.Contains(ev.Content, "memory unavailable") {
			return
		}
	}
	t.Fatal("model input concealed missing memory")
}
func TestServiceSerializesCallsAcrossRepositories(t *testing.T) {
	a, b := serviceRepo(t), serviceRepo(t)
	opts := serviceOptions(a)
	opts.Repos = append(opts.Repos, b)
	opts.MaxRequestsPerDay = 3
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var active, max atomic.Int32
	r := reviewerFunc(func(ctx context.Context, _ Snapshot) (ReviewOutput, error) {
		n := active.Add(1)
		if n > max.Load() {
			max.Store(n)
		}
		defer active.Add(-1)
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
			return ReviewOutput{}, ctx.Err()
		}
		return ReviewOutput{Summary: "done"}, nil
	})
	s, e := NewService(t.TempDir(), opts, r, nil)
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 2)
	go func() { _, e := s.Run(context.Background(), a); done <- e }()
	<-started
	go func() { _, e := s.Run(context.Background(), b); done <- e }()
	close(release)
	for range 2 {
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	}
	if max.Load() != 1 {
		t.Fatal("parallel provider calls")
	}
}

func TestServiceProviderBlockPersistsWithoutRetry(t *testing.T) {
	repo := serviceRepo(t)
	home := t.TempDir()
	calls := 0
	opts := serviceOptions(repo)
	opts.MaxRequestsPerDay = 3
	r := reviewerFunc(func(context.Context, Snapshot) (ReviewOutput, error) {
		calls++
		return ReviewOutput{}, &ProviderHTTPError{StatusCode: 429}
	})
	s, e := NewService(home, opts, r, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Run(context.Background(), repo); e == nil {
		t.Fatal("expected429")
	}
	if s.Status().Ready || s.Status().BlockedReason == "" {
		t.Fatal("provider not blocked")
	}
	s, e = NewService(home, opts, r, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Queue(repo); e == nil {
		t.Fatal("queue accepted blocked provider")
	}
	if _, e = s.Run(context.Background(), repo); e == nil {
		t.Fatal("blocked provider called")
	}
	if calls != 1 {
		t.Fatal("provider retried")
	}
	if e = s.Resume(); e != nil {
		t.Fatal(e)
	}
	if !s.Status().Ready || s.Status().RequestsToday != 1 {
		t.Fatal("resume reset usage or remained blocked")
	}
}
func TestServiceSmallBudgetRetainsCodeAndBoundsEscapedWarnings(t *testing.T) {
	repo := serviceRepo(t)
	opts := serviceOptions(repo)
	opts.MaxInputBytes = 1024
	enrich := func(_ context.Context, s *Snapshot) error {
		s.Warnings = append(s.Warnings, strings.Repeat("\x01", 600))
		return nil
	}
	s, e := NewService(t.TempDir(), opts, nil, enrich)
	if e != nil {
		t.Fatal(e)
	}
	snap, e := s.Preview(context.Background(), repo)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(snap.Evidence)
	if len(b) > 1024 {
		t.Fatalf("budget exceeded: %d", len(b))
	}
	for _, ev := range snap.Evidence {
		if ev.Kind == "diff" {
			return
		}
	}
	t.Fatal("small budget dropped all code")
}
