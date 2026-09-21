package assesseval

import (
	"encoding/json"
	"fmt"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"os"
	"path/filepath"
	"strings"
)

type CalibrationRow struct {
	ID                string  `json:"id"`
	SHA256            string  `json:"request_sha256"`
	Bytes             int     `json:"utf8_bytes"`
	ApproximateTokens int     `json:"bytes_div_four_estimate"`
	ReturnedTokens    int     `json:"returned_input_tokens"`
	EstimateRatio     float64 `json:"estimate_over_returned"`
}
type Calibration struct {
	Method      string                          `json:"method"`
	Limitations string                          `json:"limitations"`
	Rows        []CalibrationRow                `json:"rows"`
	ByArm       map[string]CalibrationAggregate `json:"by_arm"`
}
type CalibrationAggregate struct {
	N             int     `json:"n"`
	Bytes         int     `json:"bytes"`
	Estimated     int     `json:"estimated_tokens"`
	Returned      int     `json:"returned_tokens"`
	Ratio         float64 `json:"estimate_over_returned"`
	BoundFailures int     `json:"byte_bound_failures"`
}

// CalibrateFrozen reconstructs exact v1 requests and refuses hash mismatches. It
// only reads pre-existing frozen reports; no inference occurs.
func CalibrateFrozen(dir string) (Calibration, error) {
	c := Calibration{Method: "utf8-bytes-upper-bound-v1", Limitations: "Historical v1 synthetic calls; repetitive archive is not representative real retrieval. Bytes/4 is approximate, not official tokenization. Returned aggregate input usage does not prove the separate state/longest-question bound. No v2 calibration yet.", ByArm: map[string]CalibrationAggregate{}}
	for _, batch := range []string{"1", "2"} {
		b, e := os.ReadFile(filepath.Join(dir, "batch-"+batch+".json"))
		if e != nil {
			return c, e
		}
		cases, e := assess.LoadCases(strings.NewReader(string(b)))
		if e != nil {
			return c, e
		}
		byID := map[string]assess.Case{}
		for _, v := range cases {
			byID[v.ID] = v
		}
		b, e = os.ReadFile(filepath.Join(dir, "results-batch-"+batch+".json"))
		if e != nil {
			return c, e
		}
		var report assess.Report
		if e = json.Unmarshal(b, &report); e != nil {
			return c, e
		}
		for _, result := range report.Results {
			v, ok := byID[result.ID]
			if !ok {
				return c, fmt.Errorf("missing frozen case %s", result.ID)
			}
			req, e := assess.BuildRequest(v.Episode, v.Fact)
			if e != nil {
				return c, e
			}
			raw, _ := json.Marshal(req)
			hash := digest(raw)
			if hash != result.InputSHA256 {
				return c, fmt.Errorf("frozen request hash mismatch %s", result.ID)
			}
			n := result.Assessment.Usage.InputTokens
			if n <= 0 {
				return c, fmt.Errorf("missing returned usage")
			}
			estimate := (len(raw) + 3) / 4
			c.Rows = append(c.Rows, CalibrationRow{result.ID, hash, len(raw), estimate, n, float64(estimate) / float64(n)})
			arm, _, _ := strings.Cut(result.ID, "::")
			a := c.ByArm[arm]
			a.N++
			a.Bytes += len(raw)
			a.Estimated += estimate
			a.Returned += n
			if len(raw) < n {
				a.BoundFailures++
			}
			c.ByArm[arm] = a
		}
	}
	for arm, a := range c.ByArm {
		a.Ratio = float64(a.Estimated) / float64(a.Returned)
		c.ByArm[arm] = a
	}
	return c, nil
}
