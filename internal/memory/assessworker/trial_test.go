package assessworker

import (
	"context"
	"errors"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assesscontext"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"sync/atomic"
	"testing"
	"time"
)

func TestTrialCapIncludesFailedCallsAndCannotResume(t *testing.T) {
	s := sidecar(t)
	enqueue(t, s, 8, "User: Atlas uses Postgres.")
	var calls atomic.Int32
	w, e := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		calls.Add(1)
		return assess.Assessment{}, errors.New("unknown transport failure")
	}), Concurrency: 4, PollInterval: time.Millisecond, Trial: &Trial{StartsAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour), MaxRequests: 2}})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	eventually(t, func() bool {
		st, _ := s.Status()
		return st.BlockedReason == "trial_request_limit" && calls.Load() == 2
	})
	if e = w.Resume(); e == nil {
		t.Fatal("resume bypassed cap")
	}
	cancel()
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	st, _ := s.Status()
	if st.Dispatches != 2 || calls.Load() != 2 {
		t.Fatalf("exceeded limit: %+v calls=%d", st, calls.Load())
	}
}

func TestExpiredTrialDoesNotDispatchAndCannotResume(t *testing.T) {
	s := sidecar(t)
	enqueue(t, s, 1, "User: Atlas uses Postgres.")
	var calls atomic.Int32
	w, e := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		calls.Add(1)
		return assess.Assessment{}, errors.New("unexpected")
	}), PollInterval: time.Millisecond, Trial: &Trial{StartsAt: time.Now().Add(-2 * time.Hour), ExpiresAt: time.Now().Add(-time.Hour), MaxRequests: 100}})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	eventually(t, func() bool { st, _ := s.Status(); return st.BlockedReason == "trial_expired" })
	if e = w.Resume(); e == nil {
		t.Fatal("resume bypassed expiry")
	}
	cancel()
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 0 {
		t.Fatal("expired trial made calls")
	}
}

func TestWorkerPersistsTypedClientFailure(t *testing.T) {
	s := sidecar(t)
	jobs := enqueue(t, s, 1, "User: Atlas uses Postgres.")
	invalid := assess.Assessment{Model: "wrong-model"}.ValidateFor(assess.Model)
	start(t, s, dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) { return assess.Assessment{}, invalid }), 1)
	j := terminal(t, s, jobs[0].ID, "failed")
	if j.Error != "provider_model_mismatch" {
		t.Fatalf("lost typed failure: %s", j.Error)
	}
}

func TestTrialWithFutureStartBeginsWithoutManualResume(t *testing.T) {
	s := sidecar(t)
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	begins := time.Unix(0, clock.Load()).Add(time.Second)
	w, e := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		return assess.Assessment{}, errors.New("unused")
	}), PollInterval: time.Millisecond, Now: func() time.Time { return time.Unix(0, clock.Load()) }, Trial: &Trial{StartsAt: begins, ExpiresAt: begins.Add(time.Hour), MaxRequests: 1}})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	time.Sleep(10 * time.Millisecond)
	st, _ := s.Status()
	if st.BlockedReason != "" {
		cancel()
		<-done
		t.Fatal("future start persisted a manual-resume block")
	}
	clock.Store(begins.Add(time.Second).UnixNano())
	// A new job with an eligible source timestamp must run once the clock advances.
	src, e := s.Capture(assessstore.Source{EpisodeID: "scheduled", Text: "User: Atlas uses Postgres", OccurredAt: begins})
	if e != nil {
		t.Fatal(e)
	}
	jobs, e := s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "Atlas uses Postgres"}}}, assessstore.Versions{Model: assess.Model, Rubric: assess.RubricV2, ContextPolicy: assess.ContextPolicyVersion})
	if e != nil {
		t.Fatal(e)
	}
	terminal(t, s, jobs[0].ID, assessstore.Failed)
	cancel()
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}

func TestTrialRejectsRetainedPacketWithPretrialHistory(t *testing.T) {
	s := sidecar(t)
	now := time.Now()
	meta := assessstore.SourceMetadata{RepositoryScope: "repo"}
	_, e := s.Capture(assessstore.Source{EpisodeID: "old", Text: "User: Atlas old secret history", OccurredAt: now.Add(-time.Hour), SourceMetadata: meta})
	if e != nil {
		t.Fatal(e)
	}
	src, e := s.Capture(assessstore.Source{EpisodeID: "target", Text: "User: Atlas uses Postgres", OccurredAt: now, SourceMetadata: meta})
	if e != nil {
		t.Fatal(e)
	}
	jobs, e := s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "Atlas uses Postgres"}}}, assessstore.Versions{Model: assess.Model, Rubric: assess.RubricV2, ContextPolicy: assess.ContextPolicyVersion})
	if e != nil {
		t.Fatal(e)
	}
	var calls atomic.Int32
	w, e := New(Options{Store: s, Builder: assesscontext.Builder{Sources: s}, Client: dispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) {
		calls.Add(1)
		return assess.Assessment{}, errors.New("unexpected")
	}), Trial: &Trial{StartsAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), MaxRequests: 100}})
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.Claim(w.owner)
	if e != nil {
		t.Fatal(e)
	}
	packet, e := (assesscontext.Builder{Sources: s}).Build(context.Background(), j)
	if e != nil {
		t.Fatal(e)
	}
	if len(packet.Manifest.Sources) != 2 {
		t.Fatal("fixture requires old history")
	}
	if e = s.SavePacket(j.ID, w.owner, packet.Request, packet.Manifest); e != nil {
		t.Fatal(e)
	}
	j, _ = s.GetJob(j.ID)
	if e = w.process(context.Background(), j); e != nil {
		t.Fatal(e)
	}
	j, _ = s.GetJob(jobs[0].ID)
	if calls.Load() != 0 || j.Error != "trial_source_outside_window" {
		t.Fatalf("retained packet escaped source cutoff: %+v calls %d", j, calls.Load())
	}
}
