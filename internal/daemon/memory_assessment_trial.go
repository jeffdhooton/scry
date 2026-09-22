package daemon

import "time"

// The trial captures only new source episodes. Older pending extraction work and
// unknown timestamps never become an implicit historical upload.
func (d *Daemon) assessmentTrialEligible(occurred time.Time) bool {
	t := d.assessment.configuration.Trial
	if t == nil {
		return true
	}
	reason := ""
	now := time.Now()
	switch {
	case now.Before(t.StartsAt):
		reason = "trial_not_started"
	case !now.Before(t.ExpiresAt):
		reason = "trial_expired"
	case occurred.IsZero() || occurred.Before(t.StartsAt):
		reason = "trial_source_before_start"
	case !occurred.Before(t.ExpiresAt):
		reason = "trial_source_outside_window"
	}
	if reason == "" {
		s, err := d.assessmentStore()
		if err != nil {
			reason = "trial_state_unavailable"
		} else {
			st, err := s.Status()
			if err != nil {
				reason = "trial_state_unavailable"
			} else if st.Dispatches >= t.MaxRequests {
				reason = "trial_request_limit"
			}
		}
	}
	if reason != "" {
		d.assessmentGap(reason)
		return false
	}
	return true
}
