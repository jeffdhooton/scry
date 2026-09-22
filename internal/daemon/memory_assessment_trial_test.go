package daemon

import (
	"context"
	"github.com/jeffdhooton/scry/internal/config"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
	"testing"
	"time"
)

func TestAssessmentTrialExcludesBacklogAndStopsCapture(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	begin := time.Now().Add(-time.Minute)
	d.assessment.configuration.Trial = &config.AssessmentTrial{StartsAt: begin, ExpiresAt: time.Now().Add(time.Hour), MaxRequests: 1}
	r := extract.Result{Facts: []extract.Fct{{Fact: "Atlas uses Postgres"}}}
	d.captureAssessment(distill.RawEpisode{ID: "old", Text: "old raw must not persist", OccurredAt: begin.Add(-time.Second)})
	d.observeAssessment(store.PendingEpisode{ID: "old", Text: "old raw must not persist", OccurredAt: begin.Add(-time.Second)}, r)
	d.observeCommittedAssessment(store.Episode{ID: "old-derived", Summary: "old derived must not persist", OccurredAt: begin.Add(-time.Second)}, r)
	s, e := d.assessmentStore()
	if e != nil {
		t.Fatal(e)
	}
	st, _ := s.Status()
	if st.Revision != 0 {
		t.Fatal("backlog captured")
	}
	d.observeAssessment(store.PendingEpisode{ID: "new", Text: "User: Atlas uses Postgres", OccurredAt: time.Now()}, r)
	page, e := s.List(assessstore.ListQuery{Limit: 10})
	if e != nil || len(page.Jobs) != 1 {
		t.Fatalf("new job missing: %v %+v", e, page)
	}
	d.assessment.configuration.Trial.ExpiresAt = time.Now().Add(-time.Second)
	d.observeAssessment(store.PendingEpisode{ID: "later", Text: "raw after expiry", OccurredAt: time.Now()}, r)
	st, _ = s.Status()
	if st.Revision != 1 {
		t.Fatalf("capture continued after expiry: %+v", st)
	}
	out, e := d.handleAssessmentStatus(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	if out.(*AssessmentStatusResult).CoverageGaps["trial_expired"] == 0 {
		t.Fatal("ineligible source not observable")
	}
}

func TestAssessmentStatusGroupsSafeFailures(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	d.assessment.keyAvailable = true
	d.observeAssessment(store.PendingEpisode{ID: "new", Text: "User: Atlas uses Postgres", OccurredAt: time.Now()}, extract.Result{Facts: []extract.Fct{{Fact: "Atlas uses Postgres"}}})
	s, _ := d.assessmentStore()
	j, e := s.Claim("test")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Finish(j.ID, "test", assessstore.Failed, "provider_response_json", nil); e != nil {
		t.Fatal(e)
	}
	out, e := d.handleAssessmentStatus(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	if out.(*AssessmentStatusResult).ErrorCounts["provider_response_json"] != 1 {
		t.Fatal("failure category absent from status")
	}
}

func TestAssessmentTrialRetainsUnknownSourceTimeAcrossIntake(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	d.assessment.configuration.Trial = &config.AssessmentTrial{StartsAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour), MaxRequests: 100}
	st, e := d.memoryStore()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = enqueueEpisode(st, distill.RawEpisode{ID: "undated", Source: "manual", Text: "An undated old transcript"}, nil, time.Now(), false); e != nil {
		t.Fatal(e)
	}
	p, e := st.GetPending("undated")
	if e != nil {
		t.Fatal(e)
	}
	d.observeAssessment(p, extract.Result{Facts: []extract.Fct{{Fact: "An undated old fact"}}})
	side, _ := d.assessmentStore()
	status, _ := side.Status()
	if status.Revision != 0 {
		t.Fatal("unknown original source time became eligible after intake normalization")
	}
	if p.OccurredAt.IsZero() {
		t.Fatal("primary extraction compatibility changed")
	}
}
