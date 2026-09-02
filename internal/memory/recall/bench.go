package recall

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/search"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// A recall benchmark is a list of questions, each written from a real
// episode with a known answering fact. It measures whether that fact is
// among the top N returned, and how big the payloads are. The builder's
// tuning set lives in docs/memory-bench/; a grader writes its own from
// episodes it picks, held out from anything used to tune ranking.

// Question is one benchmark item. Exactly one way of naming the answer is
// needed: a (src, relation, dst|value) triple (any subset, all set fields
// must match), a substring of the fact sentence, or an episode id the fact
// must cite.
type Question struct {
	Question string `json:"question"`
	Expect   Expect `json:"expect"`
	Note     string `json:"note,omitempty"`
}

// Expect names the answering fact.
type Expect struct {
	// Entity matches a fact whose src OR dst is this slug: "a fact from
	// the intended entity", the audit's probe criterion.
	Entity        string `json:"entity,omitempty"`
	Src           string `json:"src,omitempty"`
	Relation      string `json:"relation,omitempty"`
	Dst           string `json:"dst,omitempty"`
	Value         string `json:"value,omitempty"`
	FactSubstring string `json:"fact_substring,omitempty"`
	Episode       string `json:"episode,omitempty"`
}

// Matches reports whether h is the fact e describes.
func (e Expect) Matches(h FactHit) bool {
	if e.Entity != "" && !strings.EqualFold(e.Entity, h.Src) && !strings.EqualFold(e.Entity, h.Dst) {
		return false
	}
	if e.Src != "" && !strings.EqualFold(e.Src, h.Src) {
		return false
	}
	if e.Relation != "" && e.Relation != h.Relation {
		return false
	}
	if e.Dst != "" && !strings.EqualFold(e.Dst, h.Dst) {
		return false
	}
	if e.Value != "" && !strings.EqualFold(e.Value, h.Value) {
		return false
	}
	if e.FactSubstring != "" && !strings.Contains(strings.ToLower(h.Fact), strings.ToLower(e.FactSubstring)) {
		return false
	}
	if e.Episode != "" {
		found := false
		for _, id := range h.Episodes {
			if id == e.Episode {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return e.Entity != "" || e.Src != "" || e.Relation != "" || e.Dst != "" || e.Value != "" || e.FactSubstring != "" || e.Episode != ""
}

// Miss records a question whose answer was not in the top N.
type Miss struct {
	Question string   `json:"question"`
	Rank     int      `json:"rank"` // -1 when absent from the whole result
	Top      []string `json:"top"`  // the first three returned facts
}

// BenchResult is what one run measured.
type BenchResult struct {
	Total          int     `json:"total"`
	Hits           int     `json:"hits"`
	TopN           int     `json:"top_n"`
	HitRate        float64 `json:"hit_rate"`
	MeanPayload    int     `json:"mean_payload_bytes"`
	MaxPayload     int     `json:"max_payload_bytes"`
	OverCap        int     `json:"over_cap"`
	MeanLatencyMs  float64 `json:"mean_latency_ms"`
	Misses         []Miss  `json:"misses,omitempty"`
	MeanAnswerRank float64 `json:"mean_answer_rank"` // over hits, 1-based
}

// Recaller answers one question; the daemon-backed and offline runners
// both satisfy it.
type Recaller func(q string) (Result, error)

// Bench runs every question through recall and scores the top-N placement.
func Bench(qs []Question, recallFn Recaller, topN int) (BenchResult, error) {
	if topN <= 0 {
		topN = DefaultFactLimit
	}
	res := BenchResult{Total: len(qs), TopN: topN}
	var payload, ranks int
	var latency time.Duration
	for _, q := range qs {
		start := time.Now()
		r, err := recallFn(q.Question)
		latency += time.Since(start)
		if err != nil {
			return res, fmt.Errorf("bench: %q: %w", q.Question, err)
		}
		b, _ := json.Marshal(r)
		payload += len(b)
		if len(b) > res.MaxPayload {
			res.MaxPayload = len(b)
		}
		if len(b) > MaxPayloadBytes {
			res.OverCap++
		}
		rank := -1
		for i, h := range r.Facts {
			if q.Expect.Matches(h) {
				rank = i + 1
				break
			}
		}
		if rank > 0 && rank <= topN {
			res.Hits++
			ranks += rank
			continue
		}
		top := make([]string, 0, 3)
		for i, h := range r.Facts {
			if i >= 3 {
				break
			}
			top = append(top, clip(h.Src+" -["+h.Relation+"]-> "+h.Dst+h.Value+": "+h.Fact, 140))
		}
		res.Misses = append(res.Misses, Miss{Question: q.Question, Rank: rank, Top: top})
	}
	if res.Total > 0 {
		res.HitRate = float64(res.Hits) / float64(res.Total)
		res.MeanPayload = payload / res.Total
		res.MeanLatencyMs = float64(latency.Milliseconds()) / float64(res.Total)
	}
	if res.Hits > 0 {
		res.MeanAnswerRank = float64(ranks) / float64(res.Hits)
	}
	return res, nil
}

// LoadQuestions reads a questions file (a JSON array of Question).
func LoadQuestions(path string) ([]Question, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var qs []Question
	if err := json.Unmarshal(b, &qs); err != nil {
		return nil, fmt.Errorf("bench: parse %s: %w", path, err)
	}
	for i, q := range qs {
		if strings.TrimSpace(q.Question) == "" {
			return nil, fmt.Errorf("bench: question %d is empty", i)
		}
		probe := FactHit{Src: q.Expect.Src, Relation: q.Expect.Relation, Dst: q.Expect.Dst, Value: q.Expect.Value, Fact: q.Expect.FactSubstring, Episodes: []string{q.Expect.Episode}}
		if q.Expect.Entity != "" && probe.Src == "" {
			probe.Src = q.Expect.Entity
		}
		if !q.Expect.Matches(probe) {
			return nil, fmt.Errorf("bench: question %d (%q) names no answering fact", i, q.Question)
		}
	}
	return qs, nil
}

// OfflineRecaller answers from a store directory with a freshly built
// index, for benchmarking a copy or a backup without a daemon.
func OfflineRecaller(st *store.Store) (Recaller, error) {
	ix, err := search.Build(st)
	if err != nil {
		return nil, err
	}
	return func(q string) (Result, error) { return Recall(st, ix, q, nil, 0) }, nil
}
