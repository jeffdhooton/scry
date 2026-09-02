package recall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBenchScoresTopNPlacementAndPayload(t *testing.T) {
	st, ix := seedRecall(t)
	qs := []Question{
		{Question: "how do we deploy scry to the mini", Expect: Expect{Src: "scry", Relation: "deployed_on", Dst: "mac-mini"}},
		{Question: "why did we switch off deepseek", Expect: Expect{FactSubstring: "402 Insufficient Balance"}},
		{Question: "where is the z.ai key exported", Expect: Expect{Episode: "e1", FactSubstring: "secrets.zsh"}},
		{Question: "what colour is the bikeshed", Expect: Expect{FactSubstring: "no such fact"}},
	}
	rec := func(q string) (Result, error) { return Recall(st, ix, q, nil, 0) }
	res, err := Bench(qs, rec, 20)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 4 || res.Hits != 3 || len(res.Misses) != 1 || res.Misses[0].Rank != -1 {
		t.Errorf("bench = %+v", res)
	}
	if res.MaxPayload > MaxPayloadBytes || res.OverCap != 0 || res.MeanAnswerRank < 1 {
		t.Errorf("payload stats = %+v", res)
	}

	path := filepath.Join(t.TempDir(), "q.json")
	if err := os.WriteFile(path, []byte(`[{"question":"q","expect":{"fact_substring":"x"}},{"question":"bad","expect":{}}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadQuestions(path); err == nil {
		t.Error("a question naming no answer must be rejected")
	}
	if err := os.WriteFile(path, []byte(`[{"question":"q","expect":{"fact_substring":"x"}}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if qs, err := LoadQuestions(path); err != nil || len(qs) != 1 {
		t.Errorf("LoadQuestions = %v, %v", qs, err)
	}
	off, err := OfflineRecaller(st)
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := off("scry deploy mini"); len(r.Facts) == 0 {
		t.Error("offline recaller returned nothing")
	}
}
