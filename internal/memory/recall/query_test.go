package recall

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/search"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func seedRecall(t *testing.T) (*store.Store, *search.Index) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "badger"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ents := []store.Entity{
		{Slug: "scry", Name: "scry", Type: "project", Aliases: []string{"scry daemon"}},
		{Slug: "mac-mini", Name: "Mac mini", Type: "machine", Aliases: []string{"mini"}},
		{Slug: "hermes-ops", Name: "hermes-ops", Type: "project"},
		{Slug: "jeff", Name: "Jeff", Type: "person", Aliases: []string{"key"}},
		{Slug: "childscribe-laravel", Name: "childscribe-laravel", Type: "project", Aliases: []string{"deploy"}},
		{Slug: "glm-53-flash", Name: "GLM-5.3-Flash", Type: "tool"},
	}
	for _, e := range ents {
		e.CreatedAt, e.LastSeen = now, now
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	_ = st.PutEpisode(store.Episode{ID: "ep-deploy", Source: "manual", Summary: "Deployed scry to the mini: build with -trimpath into ~/.local/bin/scry then launchctl kickstart -k gui/$(id -u)/ai.jermes.scryd", OccurredAt: now, IngestedAt: now})
	facts := []store.Fact{
		{Src: "scry", Relation: "deployed_on", Dst: "mac-mini", Fact: "scry is deployed to the mini by building with -trimpath into ~/.local/bin/scry and running launchctl kickstart -k gui/$(id -u)/ai.jermes.scryd", ValidFrom: now, Episodes: []string{"ep-deploy"}},
		{Src: "scry", Relation: "uses", Dst: "glm-53-flash", Fact: "scry switched extraction from DeepSeek to GLM-5.3-Flash because DeepSeek returned 402 Insufficient Balance on 2026-09-01", ValidFrom: now.Add(time.Hour)},
		{Src: "jeff", Relation: "configures", Dst: "scry", Fact: "Z_AI_API_KEY is exported from ~/.secrets.zsh on the laptop and from /Users/jclaw/.hermes/.env on the mini", ValidFrom: now.Add(2 * time.Hour)},
		{Src: "hermes-ops", Relation: "status", Value: "in-progress", Fact: "hermes-ops outreach work is in progress", ValidFrom: now},
	}
	// A high-degree entity with thousands of long facts, the shape that
	// produced 1.18 MB payloads.
	for i := range 3000 {
		facts = append(facts, store.Fact{Src: "childscribe-laravel", Relation: "contains", Dst: "jeff", ValidFrom: now.Add(time.Duration(i) * time.Second),
			Fact: fmt.Sprintf("childscribe-laravel deploy step %d: the key rotation runbook says to run the deploy script on forge and verify the key in the vault, then post to #childscribe-ops", i)})
	}
	for _, f := range facts {
		f.Confidence = 0.9
		if f.Episodes == nil {
			f.Episodes = []string{"e1"}
		}
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	ix, err := search.Build(st)
	if err != nil {
		t.Fatal(err)
	}
	return st, ix
}

func TestRecallAnswersTheQuestionNotTheSubstring(t *testing.T) {
	st, ix := seedRecall(t)
	cases := []struct {
		query   string
		wantSrc string
		wantIn  string
	}{
		{"scry deploy mini", "scry", "launchctl kickstart"},
		{"hermes deploy", "hermes-ops", "hermes-ops"},
		{"why did we switch off deepseek", "scry", "402 Insufficient Balance"},
		{"Z_AI_API_KEY", "jeff", "secrets.zsh"},
		{"GLM-5.3-Flash", "scry", "GLM-5.3-Flash"},
	}
	for _, tc := range cases {
		res, err := Recall(st, ix, tc.query, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Facts) == 0 {
			t.Errorf("%q: no facts", tc.query)
			continue
		}
		found := -1
		for i, f := range res.Facts {
			if i < 5 && f.Src == tc.wantSrc && strings.Contains(f.Fact, tc.wantIn) {
				found = i
				break
			}
		}
		if found < 0 {
			t.Errorf("%q: answering fact not in top 5; top = %q", tc.query, res.Facts[0].Fact)
		}
		b, _ := json.Marshal(res)
		if len(b) > MaxPayloadBytes {
			t.Errorf("%q: payload %d bytes > %d", tc.query, len(b), MaxPayloadBytes)
		}
		if len(res.Facts) > DefaultFactLimit {
			t.Errorf("%q: %d facts > default limit", tc.query, len(res.Facts))
		}
	}
}

func TestRecallCapsAHighDegreeEntity(t *testing.T) {
	st, ix := seedRecall(t)
	res, err := Recall(st, ix, "childscribe-laravel", nil, 500)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(res)
	if len(b) > MaxPayloadBytes {
		t.Fatalf("payload %d bytes > %d", len(b), MaxPayloadBytes)
	}
	if !res.Truncated || res.Total < 3000 {
		t.Errorf("truncated=%v total=%d", res.Truncated, res.Total)
	}
	if len(res.Entities) == 0 || res.Entities[0].Slug != "childscribe-laravel" || res.Entities[0].FactCount < 3000 {
		t.Errorf("entity header = %+v", res.Entities)
	}
}

func TestRecallAsOfAndNilIndex(t *testing.T) {
	st, ix := seedRecall(t)
	past := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	res, err := Recall(st, ix, "scry deploy mini", &past, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Facts) != 0 {
		t.Errorf("as-of before any fact must return nothing: %+v", res.Facts)
	}
	res, err = Recall(st, nil, "scry", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Facts) == 0 || res.Entities[0].Slug != "scry" {
		t.Errorf("nil index fallback: %+v", res)
	}
}

func TestRecallReturnsEpisodesAndAttributes(t *testing.T) {
	st, ix := seedRecall(t)
	res, _ := Recall(st, ix, "scry deploy mini", nil, 0)
	if len(res.Episodes) == 0 || res.Episodes[0].ID != "ep-deploy" {
		t.Errorf("episodes = %+v", res.Episodes)
	}
	res, _ = Recall(st, ix, "hermes-ops in progress", nil, 0)
	if len(res.Facts) == 0 || res.Facts[0].Value != "in-progress" {
		t.Errorf("attribute fact = %+v", res.Facts)
	}
}
