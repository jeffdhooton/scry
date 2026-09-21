// Package assessworker runs the independently bounded shadow-assessment queue.
// It never retries a provider request or modifies the memory graph.
package assessworker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assesscontext"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
)

type Dispatcher interface {
	Dispatch(context.Context, []byte, string) (assess.Assessment, error)
}
type PacketBuilder interface {
	Build(context.Context, assessstore.Job) (assesscontext.Packet, error)
}
type Options struct {
	Store           *assessstore.Store
	Builder         PacketBuilder
	Client          Dispatcher
	Concurrency     int
	Timeout         time.Duration
	PollInterval    time.Duration
	CleanupInterval time.Duration
	Now             func() time.Time
	Trial           *Trial
}
type Worker struct {
	opts    Options
	owner   string
	kick    chan struct{}
	running atomic.Bool
	// Serialize local dispatch admission, suspension and explicit resume.
	gate      sync.Mutex
	suspended bool
}

func New(o Options) (*Worker, error) {
	if o.Store == nil {
		return nil, errors.New("assessworker: store required")
	}
	if o.Builder == nil {
		return nil, errors.New("assessworker: builder required")
	}
	if o.Concurrency == 0 {
		o.Concurrency = 2
	}
	if o.Concurrency < 1 {
		return nil, errors.New("assessworker: invalid concurrency")
	}
	if o.Timeout == 0 {
		o.Timeout = 15 * time.Second
	}
	if o.Timeout < 0 {
		return nil, errors.New("assessworker: invalid timeout")
	}
	if o.PollInterval == 0 {
		o.PollInterval = time.Second
	}
	if o.PollInterval < 0 {
		return nil, errors.New("assessworker: invalid poll interval")
	}
	if o.CleanupInterval == 0 {
		o.CleanupInterval = time.Minute
	}
	if o.CleanupInterval < 0 {
		return nil, errors.New("assessworker: invalid cleanup interval")
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Trial != nil {
		trial := *o.Trial
		if trial.StartsAt.IsZero() || !trial.ExpiresAt.After(trial.StartsAt) || trial.MaxRequests == 0 {
			return nil, errors.New("assessworker: invalid trial")
		}
		o.Trial = &trial
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, errors.New("assessworker: owner initialization failed")
	}
	return &Worker{opts: o, owner: hex.EncodeToString(id[:]), kick: make(chan struct{}, 1)}, nil
}

// Kick coalesces notifications; durable polling also discovers work after restart.
func (w *Worker) Kick() {
	select {
	case w.kick <- struct{}{}:
	default:
	}
}

// Resume validates the immutable worker configuration and credential availability.
// Replacing credentials or configuration requires constructing a worker at restart.
// Provider access is checked by the next explicit dispatch, never a probe request.
func (w *Worker) Resume() error {
	w.gate.Lock()
	defer w.gate.Unlock()
	if w.opts.Client == nil {
		return errors.New("assessworker: TYPESAFE_API_KEY is required")
	}
	st, err := w.opts.Store.Status()
	if err != nil {
		return errors.New("assessworker: dispatch state unavailable")
	}
	if reason := w.trialStop(st); reason != "" {
		return errors.New("assessworker: " + reason)
	}
	if err := w.opts.Store.Resume(); err != nil {
		return err
	}
	w.suspended = false
	w.Kick()
	return nil
}

// Run blocks until cancellation or a sidecar failure, then waits for all requests
// to observe cancellation before returning. Dispatchers must honor their context.
func (w *Worker) Run(ctx context.Context) error {
	if !w.running.CompareAndSwap(false, true) {
		return errors.New("assessworker: already running")
	}
	defer w.running.Store(false)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if w.opts.Client == nil {
		w.gate.Lock()
		w.suspended = true
		err := w.opts.Store.Block("missing_credentials", time.Time{})
		w.gate.Unlock()
		if err != nil {
			return errors.New("assessworker: suspension persistence failed")
		}
	}
	if err := w.opts.Store.Cleanup(); err != nil {
		return errors.New("assessworker: cleanup failed")
	}
	poll := time.NewTicker(w.opts.PollInterval)
	defer poll.Stop()
	cleanup := time.NewTicker(w.opts.CleanupInterval)
	defer cleanup.Stop()
	done := make(chan error, w.opts.Concurrency)
	active := 0
	defer func() {
		cancel()
		for active > 0 {
			<-done
			active--
		}
	}()
	for {
		if ctx.Err() != nil {
			return nil
		}
		for active < w.opts.Concurrency && ctx.Err() == nil {
			w.gate.Lock()
			if w.suspended {
				w.gate.Unlock()
				break
			}
			if w.opts.Trial != nil {
				st, err := w.opts.Store.Status()
				if err != nil {
					w.gate.Unlock()
					return errors.New("assessworker: dispatch state unavailable")
				}
				if st.BlockedReason == "" {
					if reason := w.trialStop(st); reason != "" {
						if reason == "trial_not_started" {
							w.gate.Unlock()
							break
						}
						err = w.suspendTrial(reason)
						w.gate.Unlock()
						if err != nil {
							return err
						}
						break
					}
				}
			}
			job, err := w.opts.Store.Claim(w.owner)
			w.gate.Unlock()
			if errors.Is(err, assessstore.ErrNotFound) || errors.Is(err, assessstore.ErrBlocked) {
				break
			}
			if err != nil {
				return errors.New("assessworker: claim failed")
			}
			active++
			go func() { done <- w.process(ctx, job) }()
		}
		select {
		case <-ctx.Done():
			return nil
		case err := <-done:
			active--
			if err != nil {
				return err
			}
		case <-w.kick:
		case <-poll.C:
		case <-cleanup.C:
			if err := w.opts.Store.Cleanup(); err != nil {
				return errors.New("assessworker: cleanup failed")
			}
		}
	}
}
func (w *Worker) finish(j assessstore.Job, status assessstore.State, reason string, a *assess.Assessment) error {
	if err := w.opts.Store.Finish(j.ID, w.owner, status, reason, a); err != nil {
		_ = w.opts.Store.RecordGap("assessment_write_failed")
		return errors.New("assessworker: result persistence failed")
	}
	return nil
}
func (w *Worker) process(ctx context.Context, j assessstore.Job) error {
	if t := w.opts.Trial; t != nil {
		src, err := w.opts.Store.GetSource(j.SourceID)
		if err != nil {
			return w.finish(j, assessstore.Failed, "evidence_unavailable", nil)
		}
		if src.OccurredAt.IsZero() || src.OccurredAt.Before(t.StartsAt) || !src.OccurredAt.Before(t.ExpiresAt) {
			return w.finish(j, assessstore.Failed, "trial_source_outside_window", nil)
		}
	}
	if len(j.Packet) == 0 {
		packet, err := w.opts.Builder.Build(ctx, j)
		if err != nil {
			if !reflect.DeepEqual(packet.Manifest, assess.Manifest{}) {
				if saveErr := w.opts.Store.SavePreparation(j.ID, w.owner, packet.Manifest); saveErr != nil {
					_ = w.opts.Store.RecordGap("preparation_write_failed")
					return w.finish(j, assessstore.Failed, "preparation_persistence_failed", nil)
				}
			}
			state, reason := assessstore.Failed, "context_build_failed"
			if errors.Is(err, assesscontext.ErrOversize) {
				state, reason = assessstore.Oversize, "core_evidence_oversize"
			}
			if errors.Is(err, assessstore.ErrEvidenceUnavailable) {
				reason = "evidence_unavailable"
			}
			if ctx.Err() != nil {
				reason = "interrupted_delivery_unknown"
			}
			return w.finish(j, state, reason, nil)
		}
		if err := w.opts.Store.SavePacket(j.ID, w.owner, packet.Request, packet.Manifest); err != nil {
			return w.finish(j, assessstore.Failed, "packet_persistence_failed", nil)
		}
		j.Packet = packet.Request
		j.Manifest = packet.Manifest
	}
	// A replay retains its original packet; validate every retained evidence
	// source against the trial window rather than silently rebuilding the packet.
	if t := w.opts.Trial; t != nil {
		if len(j.Manifest.Sources) == 0 {
			return w.finish(j, assessstore.Failed, "trial_evidence_unverifiable", nil)
		}
		for _, ref := range j.Manifest.Sources {
			src, err := w.opts.Store.GetSource(ref.ID)
			if err != nil || src.Unavailable {
				return w.finish(j, assessstore.Failed, "evidence_unavailable", nil)
			}
			if src.OccurredAt.IsZero() || src.OccurredAt.Before(t.StartsAt) || !src.OccurredAt.Before(t.ExpiresAt) {
				return w.finish(j, assessstore.Failed, "trial_source_outside_window", nil)
			}
		}
	}
	// A job claimed before a different request's refusal must not be dispatched
	// after the persistent suspension has become visible.
	w.gate.Lock()
	if ctx.Err() != nil {
		w.gate.Unlock()
		return w.finish(j, assessstore.Failed, "interrupted_delivery_unknown", nil)
	}
	st, err := w.opts.Store.Status()
	if err != nil {
		w.gate.Unlock()
		return w.finish(j, assessstore.Failed, "dispatch_state_unavailable", nil)
	}
	if w.suspended || st.BlockedReason != "" {
		w.gate.Unlock()
		return w.finish(j, assessstore.Blocked, "dispatch_suspended", nil)
	}
	if ctx.Err() != nil {
		w.gate.Unlock()
		return w.finish(j, assessstore.Failed, "interrupted_delivery_unknown", nil)
	}
	if reason := w.trialStop(st); reason != "" {
		err := w.suspendTrial(reason)
		w.gate.Unlock()
		if err != nil {
			return err
		}
		return w.finish(j, assessstore.Blocked, reason, nil)
	}
	var max uint64
	var until time.Time
	if t := w.opts.Trial; t != nil {
		max, until = t.MaxRequests, t.ExpiresAt
	}
	if err := w.opts.Store.ReserveDispatch(j.ID, w.owner, max, until); err != nil {
		reason := "dispatch_reservation_failed"
		if errors.Is(err, assessstore.ErrDispatchLimit) {
			reason = "trial_request_limit"
		}
		if errors.Is(err, assessstore.ErrTrialExpired) {
			reason = "trial_expired"
		}
		blockErr := w.suspendTrial(reason)
		w.gate.Unlock()
		if blockErr != nil {
			return blockErr
		}
		return w.finish(j, assessstore.Blocked, reason, nil)
	}
	// Admission is the dispatch boundary; concurrently admitted requests may finish
	// after a refusal, but no subsequent requests are admitted.
	requestCtx, cancel := context.WithTimeout(ctx, w.opts.Timeout)
	w.gate.Unlock()
	a, err := w.opts.Client.Dispatch(requestCtx, j.Packet, j.Versions.Model)
	cancel()
	if err != nil {
		state, reason := assessstore.Failed, "provider_request_failed"
		var he *assess.HTTPError
		if errors.As(err, &he) {
			reason = fmt.Sprintf("provider_http_%d", he.StatusCode)
			if he.StatusCode == 401 || he.StatusCode == 403 || he.StatusCode == 429 {
				state = assessstore.Blocked
				retry := time.Time{}
				if he.RetryAfter > 0 {
					retry = w.opts.Now().Add(he.RetryAfter)
				}
				w.gate.Lock()
				w.suspended = true
				blockErr := w.opts.Store.Block(reason, retry)
				w.gate.Unlock()
				if blockErr != nil {
					_ = w.finish(j, state, reason, nil)
					return errors.New("assessworker: suspension persistence failed")
				}
			}
		} else if ctx.Err() != nil {
			reason = "interrupted_delivery_unknown"
		} else if errors.Is(err, context.DeadlineExceeded) {
			reason = "request_timeout_delivery_unknown"
		} else if code := assess.FailureCode(err); code != assess.CodeUnknown {
			reason = "provider_" + string(code)
		}
		return w.finish(j, state, reason, nil)
	}
	if err := a.ValidateFor(j.Versions.Model); err != nil || a.Usage.InputTokens < 0 || a.Usage.OutputTokens < 0 || a.LatencyMS < 0 || math.IsNaN(a.LatencyMS) || math.IsInf(a.LatencyMS, 0) {
		return w.finish(j, assessstore.Failed, "invalid_provider_response", nil)
	}
	return w.finish(j, assessstore.Completed, "", &a)
}
