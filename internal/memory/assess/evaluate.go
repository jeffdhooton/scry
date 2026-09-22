package assess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

const MaxCases = 100
const MaxDatasetBytes = 8 << 20
const InputUSDPerMillion = .042
const DiagnosticThreshold = .5

// Labels are reference judgments, never sent to the classifier. Pointers distinguish
// an intentional negative label from an accidentally omitted field.
type Labels struct {
	Supported *bool  `json:"supported"`
	Durable   *bool  `json:"durable"`
	Assertion string `json:"assertion"`
}

type Case struct {
	ID       string `json:"id"`
	Episode  string `json:"episode"`
	Fact     string `json:"fact"`
	Expected Labels `json:"expected"`
}

func LoadCases(r io.Reader) ([]Case, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxDatasetBytes+1))
	if err != nil {
		return nil, errors.New("assess: cannot read dataset")
	}
	if len(raw) > MaxDatasetBytes {
		return nil, errors.New("assess: dataset exceeds 8 MiB")
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	var cases []Case
	if err := d.Decode(&cases); err != nil {
		return nil, errors.New("assess: invalid dataset JSON or unknown fields")
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, errors.New("assess: trailing data after dataset")
	}
	if err := ValidateCases(cases); err != nil {
		return nil, err
	}
	return cases, nil
}

func ValidateCases(cases []Case) error {
	if len(cases) == 0 || len(cases) > MaxCases {
		return fmt.Errorf("assess: dataset must have 1–%d cases", MaxCases)
	}
	ids := map[string]bool{}
	for i, c := range cases {
		if strings.TrimSpace(c.ID) == "" || ids[c.ID] {
			return fmt.Errorf("assess: case %d has missing or duplicate id", i+1)
		}
		ids[c.ID] = true
		if _, err := BuildRequest(c.Episode, c.Fact); err != nil {
			return fmt.Errorf("assess: case %d: %w", i+1, err)
		}
		if c.Expected.Supported == nil || c.Expected.Durable == nil {
			return fmt.Errorf("assess: case %d needs explicit supported and durable labels", i+1)
		}
		switch c.Expected.Assertion {
		case "established", "planned", "hypothetical", "denied", "unclear":
		default:
			return fmt.Errorf("assess: case %d has invalid assertion label", i+1)
		}
	}
	return nil
}

type Assessor interface {
	Assess(context.Context, string, string) (Assessment, error)
}

type BinaryMetrics struct {
	Correct       int     `json:"correct"`
	FalsePositive int     `json:"false_positive"`
	FalseNegative int     `json:"false_negative"`
	BrierScore    float64 `json:"brier_score"`
}

type Metrics struct {
	Supported        BinaryMetrics `json:"supported"`
	Durable          BinaryMetrics `json:"durable"`
	AssertionCorrect int           `json:"assertion_correct"`
}

type CaseResult struct {
	ID          string     `json:"id"`
	InputSHA256 string     `json:"input_sha256"`
	Expected    Labels     `json:"expected"`
	Assessment  Assessment `json:"assessment"`
}

type Report struct {
	Model               string       `json:"model"`
	RubricVersion       string       `json:"rubric_version"`
	Requested           int          `json:"requested"`
	Completed           int          `json:"completed"`
	DiagnosticThreshold float64      `json:"diagnostic_threshold"`
	Metrics             Metrics      `json:"metrics"`
	Usage               Usage        `json:"usage"`
	InputUSDPerMillion  float64      `json:"input_usd_per_million"`
	EstimatedUSD        float64      `json:"estimated_usd"`
	P50LatencyMS        float64      `json:"p50_latency_ms"`
	P95LatencyMS        float64      `json:"p95_latency_ms"`
	Results             []CaseResult `json:"results"`
}

// Evaluate runs sequentially and retains successful results if a later request
// fails. It measures classification at 0.5 for comparison only; it never gates
// memory. Cost covers reported usage from successful calls, not failed calls.
func Evaluate(ctx context.Context, cases []Case, engine Assessor) (Report, error) {
	r := Report{Model: Model, RubricVersion: RubricVersion, Requested: len(cases), DiagnosticThreshold: DiagnosticThreshold, InputUSDPerMillion: InputUSDPerMillion, Results: []CaseResult{}}
	if err := ValidateCases(cases); err != nil {
		return r, err
	}
	var runErr error
	for i, c := range cases {
		if err := ctx.Err(); err != nil {
			runErr = err
			break
		}
		a, err := engine.Assess(ctx, c.Episode, c.Fact)
		if err != nil {
			runErr = fmt.Errorf("assess: case %d failed: %w", i+1, err)
			break
		}
		if err := a.validate(); err != nil {
			runErr = fmt.Errorf("assess: case %d: %w", i+1, err)
			break
		}
		request, _ := BuildRequest(c.Episode, c.Fact)
		raw, _ := json.Marshal(request)
		hash := sha256.Sum256(raw)
		r.Results = append(r.Results, CaseResult{ID: c.ID, InputSHA256: hex.EncodeToString(hash[:]), Expected: c.Expected, Assessment: a})
	}
	r.summarize()
	return r, runErr
}

func accumulate(m *BinaryMetrics, p float64, label bool) {
	prediction := p >= DiagnosticThreshold
	if prediction == label {
		m.Correct++
	} else if prediction {
		m.FalsePositive++
	} else {
		m.FalseNegative++
	}
	target := 0.0
	if label {
		target = 1
	}
	m.BrierScore += (p - target) * (p - target)
}

func (r *Report) summarize() {
	r.Completed = len(r.Results)
	if r.Completed == 0 {
		return
	}
	latencies := make([]float64, 0, r.Completed)
	for _, item := range r.Results {
		a := item.Assessment
		accumulate(&r.Metrics.Supported, *a.Answers["supported"].Noul, *item.Expected.Supported)
		accumulate(&r.Metrics.Durable, *a.Answers["durable"].Noul, *item.Expected.Durable)
		if a.Answers["assertion"].Choice == item.Expected.Assertion {
			r.Metrics.AssertionCorrect++
		}
		r.Usage.InputTokens += a.Usage.InputTokens
		r.Usage.OutputTokens += a.Usage.OutputTokens
		latencies = append(latencies, a.LatencyMS)
	}
	r.Metrics.Supported.BrierScore /= float64(r.Completed)
	r.Metrics.Durable.BrierScore /= float64(r.Completed)
	r.EstimatedUSD = float64(r.Usage.InputTokens) * InputUSDPerMillion / 1e6
	sort.Float64s(latencies)
	r.P50LatencyMS = latencies[int(math.Ceil(.5*float64(r.Completed)))-1]
	r.P95LatencyMS = latencies[int(math.Ceil(.95*float64(r.Completed)))-1]
}
