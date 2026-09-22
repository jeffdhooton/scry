package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/assessstore"
	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func assessmentDaemon(t *testing.T, mode string) *Daemon {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte("memory:\n  assessment:\n    mode: "+mode+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYPESAFE_API_KEY", "")
	t.Setenv("SCRY_MEMORY_SOCKET", "")
	d := New(LayoutFor(home))
	d.memExtractor = nil
	t.Cleanup(d.closeMemory)
	return d
}

type assessmentDispatchFunc func(context.Context, []byte, string) (assess.Assessment, error)

func (f assessmentDispatchFunc) Dispatch(ctx context.Context, p []byte, m string) (assess.Assessment, error) {
	return f(ctx, p, m)
}
func awaitAssessment(t *testing.T, cond func() bool) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("assessment integration condition timed out")
}

func TestAssessmentEndToEndDoesNotChangeGraphOrRecallWhileProviderWaits(t *testing.T) {
	var baselineFacts, baselineRecall []byte
	for _, mode := range []string{"off", "shadow"} {
		t.Run(mode, func(t *testing.T) {
			d := assessmentDaemon(t, mode)
			entered := make(chan []byte, 2)
			release := make(chan struct{})
			d.assessment.keyAvailable = true
			d.assessment.client = assessmentDispatchFunc(func(ctx context.Context, p []byte, m string) (assess.Assessment, error) {
				entered <- append([]byte(nil), p...)
				select {
				case <-ctx.Done():
					return assess.Assessment{}, ctx.Err()
				case <-release:
				}
				// Deliberately negative support must have no graph authority.
				low, high, zero := .01, .99, 0.0
				return assess.Assessment{Model: m, Answers: map[string]assess.Answer{"supported": {Type: "noul", Noul: &low}, "durable": {Type: "noul", Noul: &low}, "assertion": {Type: "choice", Choice: "denied", Confidence: &high, Probabilities: map[string]*float64{"established": &low, "planned": &zero, "hypothetical": &zero, "denied": &high, "unclear": &zero}}}, Usage: assess.Usage{InputTokens: 400, OutputTokens: 12}, LatencyMS: 1}, nil
			})
			d.memExtractor = &fakeExtractor{result: extract.Result{EpisodeSummary: "Atlas uses Postgres", Entities: []extract.Ent{{Name: "Atlas", Type: "project"}, {Name: "Postgres", Type: "tool"}}, Facts: []extract.Fct{{Src: "Atlas", Relation: "uses", Dst: "Postgres", Fact: "Atlas uses Postgres", Confidence: .9}}}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d.startAssessmentWorker(ctx)
			d.startMemoryWorker(ctx)
			ep := distill.RawEpisode{ID: "e2e", Source: "manual", SourceRef: "manual:e2e", Text: "User: Atlas uses Postgres", OccurredAt: time.Unix(1700000000, 0)}
			if _, e := d.handleMemoryEnqueue(ctx, mustJSON(t, MemoryEnqueueParams{Episodes: []distill.RawEpisode{ep}})); e != nil {
				t.Fatal(e)
			}
			st, e := d.memoryStore()
			if e != nil {
				t.Fatal(e)
			}
			// Extraction completes while Jev has not been allowed to return.
			awaitAssessment(t, func() bool { has, _ := st.HasEpisode(ep.ID); return has })
			facts, e := st.FactsFrom("atlas", false)
			if e != nil || len(facts) != 1 {
				t.Fatalf("graph: %+v %v", facts, e)
			}
			factsJSON, _ := json.Marshal(facts)
			recalled, e := d.handleMemoryRecall(ctx, mustJSON(t, MemoryRecallParams{Query: "Atlas uses Postgres", AsOf: "2023-11-15T00:00:00Z"}))
			if e != nil {
				t.Fatal(e)
			}
			recalledJSON, _ := json.Marshal(recalled)
			if mode == "off" {
				baselineFacts = factsJSON
				baselineRecall = recalledJSON
				return
			}
			if !bytes.Equal(factsJSON, baselineFacts) || !bytes.Equal(recalledJSON, baselineRecall) {
				t.Fatalf("shadow changed graph or recall:\n%s\n%s\n%s\n%s", factsJSON, baselineFacts, recalledJSON, baselineRecall)
			}
			select {
			case packet := <-entered:
				if !strings.Contains(string(packet), "target_episode") || strings.Contains(string(packet), "confidence") {
					t.Fatalf("invalid source packet: %s", packet)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("provider never called")
			}
			close(release)
			side, e := d.assessmentStore()
			if e != nil {
				t.Fatal(e)
			}
			var id string
			awaitAssessment(t, func() bool {
				page, e := side.List(assessstore.ListQuery{EpisodeID: ep.ID})
				if e != nil || len(page.Jobs) != 1 {
					return false
				}
				id = page.Jobs[0].ID
				return page.Jobs[0].Status == assessstore.Completed
			})
			shown, e := d.handleAssessmentShow(ctx, mustJSON(t, AssessmentShowParams{ID: id, IncludeContext: true}))
			if e != nil {
				t.Fatal(e)
			}
			view := shown.(*AssessmentShowResult)
			if len(view.Request) == 0 || len(view.Sources) == 0 || view.Job.Assessment == nil {
				t.Fatal("completed sample lacks inspectable provenance")
			}
			after, e := st.FactsFrom("atlas", false)
			if e != nil {
				t.Fatal(e)
			}
			afterJSON, _ := json.Marshal(after)
			if !bytes.Equal(afterJSON, baselineFacts) {
				t.Fatal("negative judgment altered graph")
			}
		})
	}
}

func TestAssessmentOffDoesNotCaptureOrCreateSidecar(t *testing.T) {
	d := assessmentDaemon(t, "off")
	ep := distill.RawEpisode{ID: "off", Source: "manual", SourceRef: "manual", Text: "User: test", OccurredAt: time.Now()}
	if _, err := d.handleMemoryEnqueue(context.Background(), mustJSON(t, MemoryEnqueueParams{Episodes: []distill.RawEpisode{ep}})); err != nil {
		t.Fatal(err)
	}
	result, err := d.handleAssessmentStatus(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.(*AssessmentStatusResult).Configuration.Mode != "off" {
		t.Fatal("mode not off")
	}
	if _, err := os.Stat(filepath.Join(d.scryHome(), "memory-assess")); !os.IsNotExist(err) {
		t.Fatal("off opened sidecar")
	}
}

func TestAssessmentRepositoryScopeRequiresNamespaceEvenWithLegacyMapping(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	d.assessment.scopes["/workspace/app"] = "project:atlas"
	a := d.assessmentMetadata("/workspace/app", true, "machine-a")
	b := d.assessmentMetadata("/workspace/app", true, "machine-b")
	unknown := d.assessmentMetadata("/workspace/app", true, "")
	if a.RepositoryScope == "" || a.RepositoryScope == b.RepositoryScope || unknown.RepositoryScope != "" {
		t.Fatalf("namespace conflation: %+v %+v %+v", a, b, unknown)
	}
}

func TestAssessmentResumeRejectsExitedWorker(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	d.assessment.keyAvailable = true
	d.assessment.client = assessmentDispatchFunc(func(context.Context, []byte, string) (assess.Assessment, error) { return assess.Assessment{}, nil })
	ctx, cancel := context.WithCancel(context.Background())
	d.startAssessmentWorker(ctx)
	awaitAssessment(t, func() bool { d.assessment.mu.Lock(); defer d.assessment.mu.Unlock(); return d.assessment.worker != nil })
	cancel()
	d.assessment.wg.Wait()
	_, err := d.handleAssessmentResume(context.Background(), nil)
	if err == nil {
		t.Fatal("resume claimed success without a worker")
	}
}

func TestAssessmentLatencyPercentilesIncludeSlowTail(t *testing.T) {
	if got := percentile([]float64{1, 1000}, .95); got != 1000 {
		t.Fatalf("p95 hid slow request: %v", got)
	}
	if got := percentile([]float64{1, 1000}, .5); got != 1 {
		t.Fatalf("p50=%v", got)
	}
}

func TestAssessmentPreExtractedCommitRetainsCandidateWithMissingSource(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	ep := store.Episode{ID: "pre-extracted", Source: "seed", SourceRef: "remote-source", Summary: "Derived summary only", OccurredAt: time.Unix(100, 0), IngestedAt: time.Now()}
	result := extract.Result{EpisodeSummary: "Derived summary only", Entities: []extract.Ent{{Name: "Atlas", Type: "project"}}, Facts: []extract.Fct{{Src: "Atlas", Relation: "status", Dst: "planned", Fact: "Atlas is planned", Confidence: .9}}}
	if _, e := d.handleMemoryCommit(context.Background(), mustJSON(t, MemoryCommitParams{Episode: ep, Result: result})); e != nil {
		t.Fatal(e)
	}
	side, e := d.assessmentStore()
	if e != nil {
		t.Fatal(e)
	}
	page, e := side.List(assessstore.ListQuery{EpisodeID: ep.ID})
	if e != nil || len(page.Jobs) != 1 {
		t.Fatalf("pre-extracted candidate missing: %+v %v", page, e)
	}
	src, e := side.GetSource(page.Jobs[0].SourceID)
	if e != nil {
		t.Fatal(e)
	}
	if src.Kind != "derived_summary" || !src.SourceUnavailable {
		t.Fatal("summary was represented as direct source")
	}
	status, e := d.handleAssessmentStatus(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	if status.(*AssessmentStatusResult).SourceUnavailable != 1 || status.(*AssessmentStatusResult).SourceAvailable != 0 {
		t.Fatal("derived-only target reported as available direct source")
	}
}

func TestAssessmentSidecarFailureCannotRequeuePrimaryIngestion(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	if e := os.WriteFile(filepath.Join(d.scryHome(), "memory-assess"), []byte("not a directory"), 0600); e != nil {
		t.Fatal(e)
	}
	d.memExtractor = &fakeExtractor{result: extract.Result{EpisodeSummary: "Atlas uses Postgres", Entities: []extract.Ent{{Name: "Atlas", Type: "project"}, {Name: "Postgres", Type: "tool"}}, Facts: []extract.Fct{{Src: "Atlas", Relation: "uses", Dst: "Postgres", Fact: "Atlas uses Postgres", Confidence: .9}}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.startMemoryWorker(ctx)
	ep := distill.RawEpisode{ID: "sidecar-failure", Source: "manual", SourceRef: "manual:sidecar-failure", Text: "User: Atlas uses Postgres", OccurredAt: time.Unix(1700000000, 0)}
	if _, e := d.handleMemoryEnqueue(ctx, mustJSON(t, MemoryEnqueueParams{Episodes: []distill.RawEpisode{ep}})); e != nil {
		t.Fatal(e)
	}
	st, e := d.memoryStore()
	if e != nil {
		t.Fatal(e)
	}
	awaitAssessment(t, func() bool {
		has, _ := st.HasEpisode(ep.ID)
		pending, _ := st.HasPending(ep.ID)
		return has && !pending
	})
	result, e := d.handleAssessmentStatus(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	status := result.(*AssessmentStatusResult)
	if status.BlockedReason != "sidecar_unavailable" || status.CoverageGaps["job_store_unavailable"] != 1 {
		t.Fatalf("missing visible gap: %+v", status)
	}
	facts, e := st.FactsFrom("atlas", false)
	if e != nil || len(facts) != 1 {
		t.Fatal("sidecar failure changed graph")
	}
}

func TestAssessmentShadowCaptureAndMissingKeyAreIndependent(t *testing.T) {
	d := assessmentDaemon(t, "shadow")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.startAssessmentWorker(ctx)
	ep := distill.RawEpisode{ID: "shadow", Source: "manual", SourceRef: "manual", Text: "User: Atlas uses Postgres", OccurredAt: time.Unix(100, 0)}
	if _, err := d.handleMemoryEnqueue(ctx, mustJSON(t, MemoryEnqueueParams{Episodes: []distill.RawEpisode{ep}})); err != nil {
		t.Fatal(err)
	}
	st, err := d.memoryStore()
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.GetPending(ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	d.observeAssessment(p, extract.Result{EpisodeSummary: "test", Facts: []extract.Fct{{Src: "Atlas", Dst: "Postgres", Relation: "uses", Fact: "Atlas uses Postgres", Confidence: .9}}})
	side, err := d.assessmentStore()
	if err != nil {
		t.Fatal(err)
	}
	page, err := side.List(assessstore.ListQuery{EpisodeID: ep.ID})
	if err != nil || len(page.Jobs) != 1 {
		t.Fatalf("jobs: %+v %v", page, err)
	}
	if ok, _ := st.HasPending(ep.ID); !ok {
		t.Fatal("assessment consumed extraction input")
	}
	status, err := d.handleAssessmentStatus(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	res := status.(*AssessmentStatusResult)
	if res.KeyAvailable || res.BlockedReason != "missing_credentials" {
		t.Fatalf("missing key not visible: %+v", res)
	}
	show, err := d.handleAssessmentShow(ctx, mustJSON(t, AssessmentShowParams{ID: page.Jobs[0].ID}))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(show)
	if len(raw) == 0 {
		t.Fatal("missing inspection")
	}
	cancel()
}
