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

func TestExpectMatchesAnyOfAndAllOf(t *testing.T) {
	hit := FactHit{Src: "mac-mini", Relation: "runs_on", Dst: "tailscale",
		Fact: "The hermes box is the jclaws Mac mini at 100.96.45.73, user jclaw"}

	// any_of: one phrasing of the answer is enough.
	e := Expect{AnyOf: []string{"nothing like this", "user jclaw"}}
	if !e.Matches(hit) {
		t.Error("any_of must match when one of its phrasings appears")
	}
	if (Expect{AnyOf: []string{"nothing", "here"}}).Matches(hit) {
		t.Error("any_of must not match when none of its phrasings appear")
	}

	// all_of: the parts of one answer must appear together.
	if !(Expect{AllOf: []string{"100.96.45.73", "jclaw"}}).Matches(hit) {
		t.Error("all_of must match when every part appears")
	}
	if (Expect{AllOf: []string{"100.96.45.73", "61229"}}).Matches(hit) {
		t.Error("all_of must not match when a part is missing")
	}

	// The two combine with the other fields rather than replacing them.
	if (Expect{Entity: "somewhere-else", AnyOf: []string{"user jclaw"}}).Matches(hit) {
		t.Error("a wrong entity must still fail even when the text matches")
	}
	if (Expect{}).Matches(hit) {
		t.Error("an empty expectation names no answering fact and must never match")
	}
}
