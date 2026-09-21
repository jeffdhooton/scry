package assess

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
)

func boolLabel(v bool) *bool { return &v }
func labeledCase(id string, support, durable bool, assertion string) Case {
	return Case{ID: id, Episode: "User: sample evidence", Fact: "Proposed memory", Expected: Labels{Supported: boolLabel(support), Durable: boolLabel(durable), Assertion: assertion}}
}

type sequenceAssessor struct {
	responses []Assessment
	calls     int
	err       error
}

func (s *sequenceAssessor) Assess(context.Context, string, string) (Assessment, error) {
	s.calls++
	if s.calls > len(s.responses) {
		return Assessment{}, s.err
	}
	return s.responses[s.calls-1], nil
}
func exampleAssessment(t *testing.T) Assessment {
	t.Helper()
	var a Assessment
	if err := json.Unmarshal([]byte(validResponse), &a); err != nil {
		t.Fatal(err)
	}
	return a
}

// Hand-derived confusion counts and Brier scores catch reversed labels,
// optimistic metrics, lost token usage and wrong cost units.
func TestEvaluateMeasuresPredictionsAgainstIndependentLabels(t *testing.T) {
	cases := []Case{labeledCase("valid", true, true, "established"), labeledCase("plan-as-completion", false, true, "planned")}
	a := exampleAssessment(t)
	a.LatencyMS = 100
	b := exampleAssessment(t)
	b.LatencyMS = 300
	engine := &sequenceAssessor{responses: []Assessment{a, b}}
	report, err := Evaluate(context.Background(), cases, engine)
	if err != nil {
		t.Fatal(err)
	}
	if report.Completed != 2 || report.Metrics.Supported.Correct != 1 || report.Metrics.Supported.FalsePositive != 1 || report.Metrics.Supported.FalseNegative != 0 || report.Metrics.AssertionCorrect != 1 {
		t.Fatalf("metrics=%+v", report.Metrics)
	}
	if math.Abs(report.Metrics.Supported.BrierScore-.4181) > 1e-8 || math.Abs(report.Metrics.Durable.BrierScore-.04) > 1e-8 {
		t.Fatalf("Brier=%+v", report.Metrics)
	}
	if report.Usage.InputTokens != 2000 || report.Usage.OutputTokens != 60 || math.Abs(report.EstimatedUSD-.000084) > 1e-10 || report.P95LatencyMS != 300 {
		t.Fatalf("accounting=%+v", report)
	}
	if len(report.Results[0].InputSHA256) != 64 || report.Results[0].InputSHA256 != report.Results[1].InputSHA256 {
		t.Fatal("input digest should depend on evidence, not labels or case id")
	}
	raw, _ := json.Marshal(report)
	if strings.Contains(string(raw), "sample evidence") || strings.Contains(string(raw), "Proposed memory") {
		t.Fatal("report leaked source text")
	}
}

func TestEvaluateCountsMissedValidFacts(t *testing.T) {
	a := exampleAssessment(t)
	low := .1
	a.Answers["supported"] = Answer{Type: "noul", Noul: &low}
	r, err := Evaluate(context.Background(), []Case{labeledCase("valid", true, true, "established")}, &sequenceAssessor{responses: []Assessment{a}})
	if err != nil || r.Metrics.Supported.FalseNegative != 1 || r.Metrics.Supported.Correct != 0 {
		t.Fatalf("report=%+v err=%v", r, err)
	}
}

func TestEvaluatePreservesPartialResultsAndStopsAtFirstFailure(t *testing.T) {
	engine := &sequenceAssessor{responses: []Assessment{exampleAssessment(t)}, err: errors.New("provider unavailable")}
	cases := []Case{labeledCase("a", true, true, "established"), labeledCase("b", false, true, "planned"), labeledCase("c", false, true, "planned")}
	r, err := Evaluate(context.Background(), cases, engine)
	if err == nil || r.Completed != 1 || r.Requested != 3 || engine.calls != 2 || len(r.Results) != 1 {
		t.Fatalf("report=%+v calls=%d err=%v", r, engine.calls, err)
	}
}

func TestLoadCasesRejectsIncompleteOrAmbiguousLabels(t *testing.T) {
	for _, input := range []string{
		`[]`,
		`[{"id":"a","episode":"e","fact":"f","expected":{"durable":true,"assertion":"established"}}]`,
		`[{"id":"a","episode":"e","fact":"f","expected":{"supported":true,"durable":true,"assertion":"nonsense"}}]`,
		`[{"id":"a","episode":"e","fact":"f","expected":{"supported":true,"durable":true,"assertion":"established"},"typo":true}]`,
		`[{"id":"a","episode":"e","fact":"f","expected":{"supported":true,"durable":true,"assertion":"established"}}] {}`,
	} {
		if _, err := LoadCases(strings.NewReader(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}

func TestEvaluateValidatesWholeDatasetBeforeAnyRequests(t *testing.T) {
	engine := &sequenceAssessor{}
	cases := []Case{labeledCase("duplicate", true, true, "established"), labeledCase("duplicate", true, true, "established")}
	if _, err := Evaluate(context.Background(), cases, engine); err == nil || engine.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, engine.calls)
	}
}

func TestSyntheticDatasetIncludesPlansAndValidMemories(t *testing.T) {
	f, err := os.Open("../../../docs/memory-assess/synthetic.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cases, err := LoadCases(f)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]bool{}
	positive, negative := false, false
	for _, c := range cases {
		statuses[c.Expected.Assertion] = true
		if *c.Expected.Supported {
			positive = true
		} else {
			negative = true
		}
	}
	if len(cases) < 10 || len(statuses) != 5 || !positive || !negative {
		t.Fatalf("insufficient coverage: %d cases, %v", len(cases), statuses)
	}
}
