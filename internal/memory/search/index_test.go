package search

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestTokenize(t *testing.T) {
	cases := map[string][]string{
		"Why did we switch off DeepSeek?":    {"switch", "deepseek", "deep", "seek"},
		"scry deployed_on mac-mini":          {"scry", "deployed-on", "deploy", "mac-mini", "mac", "mini"},
		"Z_AI_API_KEY":                       {"z-ai-api-key", "ai", "api", "key"},
		"childscribeLaravel uses Laravel 13": {"childscribelaravel", "childscribe", "laravel", "use", "laravel", "13"},
		"switches switched switching":        {"switch", "switch", "switch"},
		"the and for":                        nil,
	}
	for in, want := range cases {
		got := Tokenize(in)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("Tokenize(%q) = %v, want %v", in, got, want)
		}
	}
}

func seed(t *testing.T) (*store.Store, *Index) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "badger"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ents := []store.Entity{
		{Slug: "scry", Name: "scry", Type: "project", Aliases: []string{"scry daemon"}, Description: "code intelligence daemon"},
		{Slug: "mac-mini", Name: "Mac mini", Type: "machine"},
		{Slug: "deepseek-v4-flash", Name: "deepseek-v4-flash", Type: "tool"},
		{Slug: "glm-53-flash", Name: "GLM-5.3-Flash", Type: "tool"},
		{Slug: "childscribe-laravel", Name: "childscribe-laravel", Type: "project", Aliases: []string{"deploy"}},
	}
	for _, e := range ents {
		e.CreatedAt, e.LastSeen = now, now
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	facts := []store.Fact{
		{Src: "scry", Relation: "deployed_on", Dst: "mac-mini", Fact: "scry is deployed on the Mac mini via ssh and launchctl kickstart", ValidFrom: now},
		{Src: "scry", Relation: "uses", Dst: "glm-53-flash", Fact: "scry memory extraction switched from DeepSeek to GLM-5.3-Flash because DeepSeek returned 402 Insufficient Balance", ValidFrom: now.Add(time.Hour)},
		{Src: "scry", Relation: "uses", Dst: "deepseek-v4-flash", Fact: "scry memory extraction uses deepseek-v4-flash", ValidFrom: now.Add(-48 * time.Hour), InvalidAt: ptr(now.Add(time.Hour))},
		{Src: "scry", Relation: "status", Value: "in-progress", Fact: "scry memory work is in progress", ValidFrom: now},
	}
	for i := range 200 {
		facts = append(facts, store.Fact{Src: "childscribe-laravel", Relation: "contains", Dst: "scry", Fact: "childscribe-laravel deploy step " + strings.Repeat("x", i%7) + " runs the deploy script on forge", ValidFrom: now.Add(time.Duration(i) * time.Minute)})
	}
	for _, f := range facts {
		f.Confidence = 0.9
		f.Episodes = []string{"e1"}
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	ix, err := Build(st)
	if err != nil {
		t.Fatal(err)
	}
	return st, ix
}

func ptr(t time.Time) *time.Time { return &t }

func TestSearchRanksTheAnsweringFactFirst(t *testing.T) {
	_, ix := seed(t)
	hits := ix.Search("why did we switch off deepseek", []string{KindFact}, nil, 5)
	if len(hits) == 0 || !strings.Contains(hits[0].Doc.Text, "402 Insufficient Balance") {
		t.Fatalf("top hit = %+v", hits)
	}
	hits = ix.Search("scry deploy mini", []string{KindFact}, nil, 5)
	if len(hits) == 0 || hits[0].Doc.Fact.Dst != "mac-mini" {
		t.Fatalf("top hit for 'scry deploy mini' = %+v", hits[0].Doc.Text)
	}
	if hits := ix.Search("in progress", []string{KindFact}, nil, 5); len(hits) == 0 || hits[0].Doc.Fact.Value != "in-progress" {
		t.Errorf("attribute facts must be searchable by value: %+v", hits)
	}
}

func TestSearchEntitiesAndAsOf(t *testing.T) {
	_, ix := seed(t)
	ents := ix.Search("scry daemon", []string{KindEntity}, nil, 3)
	if len(ents) == 0 || ents[0].Doc.Slugs[0] != "scry" {
		t.Fatalf("entity search = %+v", ents)
	}
	current := ix.Search("deepseek-v4-flash extraction", []string{KindFact}, nil, 5)
	for _, h := range current {
		if h.Doc.Fact.Dst == "deepseek-v4-flash" {
			t.Errorf("invalidated fact returned as current: %s", h.Doc.Text)
		}
	}
	past := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	then := ix.Search("deepseek-v4-flash extraction", []string{KindFact}, &past, 5)
	found := false
	for _, h := range then {
		if h.Doc.Fact.Dst == "deepseek-v4-flash" {
			found = true
		}
	}
	if !found {
		t.Errorf("as-of search must return the fact that was valid then: %+v", then)
	}
}

func TestUpsertAndRemoveKeepTheIndexCurrent(t *testing.T) {
	_, ix := seed(t)
	n := ix.Len()
	f := store.Fact{Src: "scry", Relation: "monitors", Dst: "mac-mini", Fact: "scry doctor watches the tunnel to the mini", ValidFrom: time.Now()}
	ix.Upsert(FactDoc(f, nil))
	if ix.Len() != n+1 {
		t.Fatalf("Len after upsert = %d, want %d", ix.Len(), n+1)
	}
	if hits := ix.Search("doctor watches tunnel", []string{KindFact}, nil, 1); len(hits) == 0 || hits[0].Doc.Key != FactKey(f) {
		t.Fatalf("new fact not found: %+v", hits)
	}
	ix.Upsert(FactDoc(f, nil)) // replace, not duplicate
	if ix.Len() != n+1 {
		t.Errorf("Len after re-upsert = %d", ix.Len())
	}
	ix.Remove(FactKey(f))
	if ix.Len() != n {
		t.Errorf("Len after remove = %d, want %d", ix.Len(), n)
	}
	if hits := ix.Search("doctor watches tunnel", []string{KindFact}, nil, 1); len(hits) != 0 && hits[0].Doc.Key == FactKey(f) {
		t.Error("removed fact still returned")
	}
}
