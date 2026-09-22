package assesseval

import (
	"context"
	"encoding/json"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFrozenCorpus(t *testing.T) {
	c, e := Load("../../../docs/memory-assess/integration-v1/corpus.json")
	if e != nil {
		t.Fatal(e)
	}
	if len(c.Cases) < 100 {
		t.Fatal("small corpus")
	}
	if e = c.Validate(); e != nil {
		t.Fatal(e)
	}
}
func TestPipeline(t *testing.T) {
	c, e := Load("../../../docs/memory-assess/integration-v1/corpus.json")
	if e != nil {
		t.Fatal(e)
	}
	c.Cases = c.Cases[:4]
	r, e := Run(context.Background(), c, Options{Mode: "mock", Arms: []string{"compact", "relevant", "target20k"}, Rubrics: []string{"memory-assess-v1", "memory-assess-v2"}})
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Results) != 24 {
		t.Fatal(len(r.Results))
	}
	for _, v := range r.Results {
		if v.Status != "completed" || v.PacketHash == "" {
			t.Fatalf("%+v", v)
		}
		s := string(v.Packet)
		for _, bad := range []string{"expected", "confidence", c.Cases[0].ID} {
			if strings.Contains(s, bad) {
				t.Fatalf("leak %s", bad)
			}
		}
	}
	if r.InputTokens != nil || r.EstimatedUSD != nil {
		t.Fatal("mock reports real billing")
	}
	b, _ := json.Marshal(r)
	if e = os.WriteFile(t.TempDir()+"/report.json", b, 0600); e != nil {
		t.Fatal(e)
	}
}
func TestMetrics(t *testing.T) {
	m := Binary{}
	m.Add(.8, false)
	m.Add(.2, true)
	m.Finalize()
	if m.FP != 1 || m.FN != 1 || m.Brier < .639 || m.Brier > .641 {
		t.Fatalf("%+v", m)
	}
}

func TestPreviewCorpusBudgetsAndRetrieval(t *testing.T) {
	c, e := Load("../../../docs/memory-assess/integration-v1/corpus.json")
	if e != nil {
		t.Fatal(e)
	}
	r, e := Run(context.Background(), c, Options{Mode: "preview"})
	if e != nil {
		t.Fatal(e)
	}
	if r.Counts["previewed"] != 600 {
		t.Fatal(r.Counts)
	}
	unicode, derived, history := false, false, false
	for _, v := range r.Results {
		if !v.Manifest.Budget.WithinCeilings() {
			t.Fatal("unsafe budget", v.CaseID)
		}
		if strings.Contains(string(v.Packet), "接続できません") {
			unicode = true
		}
		if v.Arm == "compact" && len(v.Manifest.Sources) != 1 {
			t.Fatal("compact contains history")
		}
		if v.Arm != "compact" && len(v.Manifest.Sources) > 1 {
			history = true
		}
		if len(v.Manifest.MissingRaw) > 0 {
			derived = true
		}
		if digest(v.Packet) != v.PacketHash {
			t.Fatal("persisted hash mismatch")
		}
	}
	if !unicode || !derived || !history {
		t.Fatalf("coverage unicode=%v derived=%v history=%v", unicode, derived, history)
	}
}
func TestFrozenCalibration(t *testing.T) {
	c, e := CalibrateFrozen("../../../docs/memory-assess/context-experiment")
	if e != nil {
		t.Fatal(e)
	}
	if len(c.Rows) != 192 {
		t.Fatal(len(c.Rows))
	}
	for _, a := range c.ByArm {
		if a.BoundFailures != 0 {
			t.Fatal(a)
		}
	}
}
func TestNearestRankLatency(t *testing.T) {
	a, _ := (ScriptedMock{}).Dispatch(context.Background(), []byte("first"), "jev-1.13.0")
	b := a
	a.LatencyMS = 1
	b.LatencyMS = 1000
	r := Report{Mode: "mock", Counts: map[string]int{}, Metrics: map[string]*Metrics{}, Results: []Result{{Status: "completed", Assessment: &a, EndToEndMS: 1}, {Status: "completed", Assessment: &b, EndToEndMS: 1000}}}
	r.summarize()
	if r.HTTPP95 != 1000 || r.HTTPP50 != 1 || r.EndToEndP95 != 1000 {
		t.Fatal(r.HTTPP50, r.HTTPP95, r.EndToEndP95)
	}
}

type blockingDispatcher struct{ calls atomic.Int32 }

func (d *blockingDispatcher) Dispatch(context.Context, []byte, string) (assess.Assessment, error) {
	d.calls.Add(1)
	return assess.Assessment{}, &assess.HTTPError{StatusCode: 429}
}
func TestRefusalStopsAllArms(t *testing.T) {
	c, e := Load("../../../docs/memory-assess/integration-v1/corpus.json")
	if e != nil {
		t.Fatal(e)
	}
	c.Cases = c.Cases[:5]
	d := &blockingDispatcher{}
	r, e := Run(context.Background(), c, Options{Mode: "live", Client: d})
	if e == nil {
		t.Fatal("blocked run returned success")
	}
	if len(r.Results) != 5 || len(r.Observations) != 2 || d.calls.Load() > 2 {
		t.Fatalf("continued after refusal: rows %d calls %d", len(r.Results), d.calls.Load())
	}
	if r.Counts["blocked"] == 0 || r.ErrorCounts["provider_http_429"] == 0 {
		t.Fatal("failure missing", r.Counts, r.ErrorCounts)
	}
}
func TestLabelsDoNotAffectPacketOrMock(t *testing.T) {
	c, e := Load("../../../docs/memory-assess/integration-v1/corpus.json")
	if e != nil {
		t.Fatal(e)
	}
	c.Cases = c.Cases[:1]
	o := Options{Mode: "mock", Arms: []string{"relevant"}, Rubrics: []string{"memory-assess-v2"}}
	a, e := Run(context.Background(), c, o)
	if e != nil {
		t.Fatal(e)
	}
	c.Cases[0].Expected = Labels{false, false, "hypothetical"}
	b, e := Run(context.Background(), c, o)
	if e != nil {
		t.Fatal(e)
	}
	if a.Results[0].PacketHash != b.Results[0].PacketHash {
		t.Fatal("labels influence packet")
	}
	aa, _ := json.Marshal(a.Results[0].Assessment)
	bb, _ := json.Marshal(b.Results[0].Assessment)
	if string(aa) != string(bb) {
		t.Fatal("mock echoes labels")
	}
}
