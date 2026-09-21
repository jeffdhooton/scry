package assessworker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assesscontext"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/jeffdhooton/scry/internal/memory/extract"
)

type dispatchFunc func(context.Context, []byte, string) (assess.Assessment, error)

func (f dispatchFunc) Dispatch(c context.Context, b []byte, m string) (assess.Assessment, error) {
	return f(c, b, m)
}
func sidecar(t *testing.T) *assessstore.Store {
	t.Helper()
	s, e := assessstore.Open(filepath.Join(t.TempDir(), "sidecar"), assessstore.Options{})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func enqueue(t *testing.T, s *assessstore.Store, n int, text string) []assessstore.Job {
	t.Helper()
	src, e := s.Capture(assessstore.Source{EpisodeID: fmt.Sprint(n), Text: text, OccurredAt: time.Now()})
	if e != nil {
		t.Fatal(e)
	}
	facts := make([]extract.Fct, n)
	for i := range facts {
		facts[i] = extract.Fct{Fact: fmt.Sprintf("Atlas uses Postgres %d", i)}
	}
	j, e := s.Enqueue(src.ID, extract.Result{Facts: facts}, assessstore.Versions{Model: assess.Model, Rubric: assess.RubricV2, ContextPolicy: assess.ContextPolicyVersion})
	if e != nil {
		t.Fatal(e)
	}
	return j
}
func start(t *testing.T, s *assessstore.Store, d Dispatcher, concurrency int) (*Worker, context.CancelFunc, <-chan error) {
	t.Helper()
	w, e := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: d, Concurrency: concurrency, PollInterval: time.Millisecond, CleanupInterval: time.Hour})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(time.Second):
			t.Error("worker did not stop")
		}
	})
	return w, cancel, done
}
func eventually(t *testing.T, f func() bool) {
	t.Helper()
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		if f() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition did not become true")
}
func terminal(t *testing.T, s *assessstore.Store, id string, state assessstore.State) assessstore.Job {
	t.Helper()
	var j assessstore.Job
	eventually(t, func() bool { j, _ = s.GetJob(id); return j.Status == state })
	return j
}

func TestPersistsPacketBeforeDispatchAndNeverRetries(t *testing.T) {
	s := sidecar(t)
	jobs := enqueue(t, s, 1, "User: Atlas uses Postgres.")
	var calls atomic.Int32
	w, _, _ := start(t, s, dispatchFunc(func(_ context.Context, b []byte, m string) (assess.Assessment, error) {
		attempt := calls.Add(1)
		j, e := s.GetJob(jobs[0].ID)
		if e != nil || (attempt == 1 && (j.Status != assessstore.Running || j.Owner == "")) || j.PacketHash == "" || !bytes.Equal(j.Packet, b) || m != assess.Model {
			t.Error("dispatch preceded durable exact packet")
		}
		return assess.Assessment{}, errors.New("unsafe SECRET provider body")
	}), 1)
	j := terminal(t, s, jobs[0].ID, assessstore.Failed)
	if strings.Contains(j.Error, "SECRET") {
		t.Fatal("unsafe error retained")
	}
	w.Kick()
	time.Sleep(25 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatal("automatic retry")
	}
	replay, e := s.Replay(j.ID)
	if e != nil {
		t.Fatal(e)
	}
	w.Kick()
	terminal(t, s, replay.ID, assessstore.Failed)
	if calls.Load() != 2 {
		t.Fatal("explicit replay missing")
	}
	r, _ := s.GetJob(replay.ID)
	if !bytes.Equal(r.Packet, j.Packet) || r.ParentID != j.ID {
		t.Fatal("replay packet changed")
	}
}
func TestBoundedSlowCallsAndCancellation(t *testing.T) {
	s := sidecar(t)
	jobs := enqueue(t, s, 4, "Atlas uses Postgres.")
	var calls atomic.Int32
	entered := make(chan struct{}, 4)
	_, cancel, _ := start(t, s, dispatchFunc(func(ctx context.Context, _ []byte, _ string) (assess.Assessment, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-ctx.Done()
		return assess.Assessment{}, ctx.Err()
	}), 0)
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("default concurrency not two")
		}
	}
	time.Sleep(15 * time.Millisecond)
	if calls.Load() != 2 {
		t.Fatal("unbounded provider concurrency")
	}
	cancel()
	eventually(t, func() bool {
		st, _ := s.Status()
		return st.Counts[assessstore.Failed] == 2 && st.Counts[assessstore.Pending] == 2
	})
	for _, j := range jobs {
		got, _ := s.GetJob(j.ID)
		if got.Status == assessstore.Failed && got.Error != "interrupted_delivery_unknown" {
			t.Fatalf("wrong shutdown reason: %s", got.Error)
		}
	}
}
func TestProviderBlockAndExplicitResume(t *testing.T) {
	for _, code := range []int{401, 403, 429} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			s := sidecar(t)
			enqueue(t, s, 3, "Atlas uses Postgres.")
			var calls atomic.Int32
			w, _, _ := start(t, s, dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
				calls.Add(1)
				return assess.Assessment{}, &assess.HTTPError{StatusCode: code, RetryAfter: time.Hour}
			}), 1)
			eventually(t, func() bool { st, _ := s.Status(); return st.BlockedReason != "" && st.Counts[assessstore.Blocked] == 1 })
			w.Kick()
			time.Sleep(10 * time.Millisecond)
			if calls.Load() != 1 {
				t.Fatal("new dispatch after refusal")
			}
			if !errors.Is(w.Resume(), assessstore.ErrCooldown) {
				t.Fatal("resume ignored cooldown")
			}
		})
	}
}
func TestMissingCredentialsAndOversize(t *testing.T) {
	t.Run("key", func(t *testing.T) {
		s := sidecar(t)
		enqueue(t, s, 1, "Atlas uses Postgres.")
		w, _, _ := start(t, s, nil, 1)
		eventually(t, func() bool {
			st, _ := s.Status()
			return st.BlockedReason == "missing_credentials" && st.Counts[assessstore.Pending] == 1
		})
		if w.Resume() == nil {
			t.Fatal("resume without credentials")
		}
	})
	t.Run("oversize", func(t *testing.T) {
		s := sidecar(t)
		jobs := enqueue(t, s, 1, strings.Repeat("very long source ", 5000))
		var calls atomic.Int32
		start(t, s, dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
			calls.Add(1)
			return assess.Assessment{}, nil
		}), 1)
		j := terminal(t, s, jobs[0].ID, assessstore.Oversize)
		if j.Manifest.Budget.Method == "" || j.Manifest.Budget.DispatchBoundTokens == 0 || j.Manifest.ContextPolicy != assess.ContextPolicyVersion {
			t.Fatalf("oversize preparation manifest missing: %+v", j.Manifest)
		}
		if len(j.Packet) != 0 || j.PacketHash != "" {
			t.Fatal("oversize job fabricated request packet")
		}
		if _, err := s.Replay(j.ID); !errors.Is(err, assessstore.ErrEvidenceUnavailable) {
			t.Fatalf("unsent oversize core replayable: %v", err)
		}
		if calls.Load() != 0 {
			t.Fatal("oversize dispatched")
		}
	})
}

func validAssessment() assess.Assessment {
	yes, zero := 1.0, 0.0
	return assess.Assessment{Model: assess.Model, Answers: map[string]assess.Answer{
		"supported": {Type: "noul", Noul: &yes}, "durable": {Type: "noul", Noul: &yes},
		"assertion": {Type: "choice", Choice: "established", Confidence: &yes, Probabilities: map[string]*float64{"established": &yes, "planned": &zero, "hypothetical": &zero, "denied": &zero, "unclear": &zero}},
	}, Usage: assess.Usage{InputTokens: 100, OutputTokens: 3}, LatencyMS: 1}
}
func TestInvalidUsageIsTerminalAndDoesNotStopQueue(t *testing.T) {
	s := sidecar(t)
	enqueue(t, s, 2, "Atlas uses Postgres.")
	var calls atomic.Int32
	start(t, s, dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		a := validAssessment()
		if calls.Add(1) == 1 {
			a.Usage.InputTokens = -1
		}
		return a, nil
	}), 1)
	eventually(t, func() bool {
		st, _ := s.Status()
		return st.Counts[assessstore.Completed] == 1 && st.Counts[assessstore.Failed] == 1
	})
}
func TestRestartPreservesPendingAndDoesNotResendUncertainDelivery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart")
	s, e := assessstore.Open(path, assessstore.Options{})
	if e != nil {
		t.Fatal(e)
	}
	enqueue(t, s, 2, "Atlas uses Postgres.")
	uncertain, e := s.Claim("crashed-owner")
	if e != nil {
		t.Fatal(e)
	}
	p, e := (assesscontext.Builder{Sources: s}).Build(context.Background(), uncertain)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SavePacket(uncertain.ID, "crashed-owner", p.Request, p.Manifest); e != nil {
		t.Fatal(e)
	}
	if e = s.Block("provider_http_403", time.Time{}); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = assessstore.Open(path, assessstore.Options{})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	var calls atomic.Int32
	w, _, _ := start(t, s, dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		calls.Add(1)
		return validAssessment(), nil
	}), 1)
	j := terminal(t, s, uncertain.ID, assessstore.Failed)
	if j.Error != "interrupted_delivery_unknown" || !bytes.Equal(j.Packet, p.Request) {
		t.Fatal("uncertain delivery not preserved")
	}
	time.Sleep(15 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatal("restart cleared suspension")
	}
	if e = w.Resume(); e != nil {
		t.Fatal(e)
	}
	eventually(t, func() bool { st, _ := s.Status(); return st.Counts[assessstore.Completed] == 1 })
	if calls.Load() != 1 {
		t.Fatal("uncertain request automatically resent")
	}
	replay, e := s.Replay(j.ID)
	if e != nil {
		t.Fatal(e)
	}
	w.Kick()
	r := terminal(t, s, replay.ID, assessstore.Completed)
	if !bytes.Equal(r.Packet, j.Packet) || r.ParentID != j.ID || calls.Load() != 2 {
		t.Fatal("explicit replay did not retain original packet")
	}
}
func TestTimeoutThenContinuesAndPeriodicRetention(t *testing.T) {
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()) }
	s, e := assessstore.Open(filepath.Join(t.TempDir(), "retention"), assessstore.Options{Now: now, Limits: assessstore.Limits{PayloadAge: time.Hour}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	enqueue(t, s, 2, "Atlas uses Postgres.")
	var calls atomic.Int32
	w, e := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: dispatchFunc(func(ctx context.Context, _ []byte, _ string) (assess.Assessment, error) {
		if calls.Add(1) == 1 {
			<-ctx.Done()
			return assess.Assessment{}, ctx.Err()
		}
		return validAssessment(), nil
	}), Concurrency: 1, Timeout: 10 * time.Millisecond, PollInterval: time.Millisecond, CleanupInterval: 5 * time.Millisecond, Now: now})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	})
	eventually(t, func() bool {
		st, _ := s.Status()
		return st.Counts[assessstore.Completed] == 1 && st.Counts[assessstore.Failed] == 1
	})
	page, e := s.List(assessstore.ListQuery{})
	if e != nil {
		t.Fatal(e)
	}
	for _, j := range page.Jobs {
		if j.Status == assessstore.Failed && j.Error != "request_timeout_delivery_unknown" {
			t.Fatalf("timeout reason %s", j.Error)
		}
	}
	clock.Add(int64(2 * time.Hour))
	eventually(t, func() bool {
		p, _ := s.List(assessstore.ListQuery{})
		for _, j := range p.Jobs {
			if !j.EvidenceUnavailable || len(j.Packet) > 0 {
				return false
			}
		}
		return len(p.Jobs) == 2
	})
}

func TestServerFailureContinuesWithoutRetry(t *testing.T) {
	s := sidecar(t)
	enqueue(t, s, 2, "Atlas uses Postgres.")
	var calls atomic.Int32
	w, _, _ := start(t, s, dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		if calls.Add(1) == 1 {
			return assess.Assessment{}, &assess.HTTPError{StatusCode: 503}
		}
		return validAssessment(), nil
	}), 1)
	eventually(t, func() bool {
		st, _ := s.Status()
		return st.Counts[assessstore.Completed] == 1 && st.Counts[assessstore.Failed] == 1 && st.BlockedReason == ""
	})
	w.Kick()
	time.Sleep(10 * time.Millisecond)
	if calls.Load() != 2 {
		t.Fatal("server failure retried")
	}
}
func TestResumeAfterCooldownUsesRetainedPendingWork(t *testing.T) {
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()) }
	s, e := assessstore.Open(filepath.Join(t.TempDir(), "cooldown"), assessstore.Options{Now: now})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	enqueue(t, s, 2, "Atlas uses Postgres.")
	var calls atomic.Int32
	w, e := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		if calls.Add(1) == 1 {
			return assess.Assessment{}, &assess.HTTPError{StatusCode: 429, RetryAfter: time.Hour}
		}
		return validAssessment(), nil
	}), Concurrency: 1, Now: now, PollInterval: time.Millisecond})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	})
	eventually(t, func() bool { st, _ := s.Status(); return st.Counts[assessstore.Blocked] == 1 })
	if !errors.Is(w.Resume(), assessstore.ErrCooldown) {
		t.Fatal("cooldown bypass")
	}
	clock.Add(int64(time.Hour))
	time.Sleep(10 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatal("automatic resume")
	}
	if e = w.Resume(); e != nil {
		t.Fatal(e)
	}
	eventually(t, func() bool {
		st, _ := s.Status()
		return st.Counts[assessstore.Completed] == 1 && st.Counts[assessstore.Blocked] == 1 && st.BlockedReason == ""
	})
}
func TestClosedSidecarStopsWorkerSafely(t *testing.T) {
	s := sidecar(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	w, err := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		t.Error("called provider with unavailable store")
		return validAssessment(), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if err = w.Run(context.Background()); err == nil || err.Error() != "assessworker: cleanup failed" {
		t.Fatalf("unsafe or missing failure: %v", err)
	}
}
func TestConfigurationValidation(t *testing.T) {
	s := sidecar(t)
	base := Options{Store: s, Builder: assesscontext.Builder{Sources: s}}
	cases := []struct {
		name string
		edit func(*Options)
	}{{"store", func(o *Options) { o.Store = nil }}, {"builder", func(o *Options) { o.Builder = nil }}, {"concurrency", func(o *Options) { o.Concurrency = -1 }}, {"timeout", func(o *Options) { o.Timeout = -time.Second }}, {"poll interval", func(o *Options) { o.PollInterval = -time.Second }}, {"cleanup interval", func(o *Options) { o.CleanupInterval = -time.Second }}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			o := base
			tt.edit(&o)
			if _, err := New(o); err == nil || !strings.Contains(err.Error(), tt.name) {
				t.Fatalf("configuration error missing field: %v", err)
			}
		})
	}
}
