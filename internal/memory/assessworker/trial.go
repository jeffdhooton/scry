package assessworker

import (
	"errors"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"time"
)

type Trial struct {
	StartsAt    time.Time
	ExpiresAt   time.Time
	MaxRequests uint64
}

func (w *Worker) trialStop(st assessstore.Status) string {
	t := w.opts.Trial
	if t == nil {
		return ""
	}
	now := w.opts.Now()
	if now.Before(t.StartsAt) {
		return "trial_not_started"
	}
	if !now.Before(t.ExpiresAt) {
		return "trial_expired"
	}
	if st.Dispatches >= t.MaxRequests {
		return "trial_request_limit"
	}
	return ""
}

// Caller owns gate; persistent suspension survives daemon restart.
func (w *Worker) suspendTrial(reason string) error {
	w.suspended = true
	if err := w.opts.Store.Block(reason, time.Time{}); err != nil {
		return errors.New("assessworker: suspension persistence failed")
	}
	return nil
}
