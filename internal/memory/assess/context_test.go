package assess

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestV2PacketRedactsAndBoundsUnicode(t *testing.T) {
	c := Context{Candidate: Candidate{Text: "a sk-abcdefghijklmnopqrstuvwxyz", SourceMention: "/Users/jeff/private"}, TargetEpisode: SourceRecord{ID: "opaque-1", Text: "你好 sk-abcdefghijklmnopqrstuvwxyz", OccurredAt: "2026-09-20T00:00:00Z"}, EvaluationScope: EvaluationScope{TargetTime: "2026-09-20T00:00:00Z"}}
	p, err := BuildRequestV2(c)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), "sk-abcdefghijklmnopqrstuvwxyz") || strings.Contains(string(b), "/Users/jeff") {
		t.Fatalf("unsafe packet: %s", b)
	}
	report, err := ByteBudgetCounter{}.Count(p)
	if err != nil {
		t.Fatal(err)
	}
	if report.DispatchBoundTokens < len(b) {
		t.Fatalf("bound %d < bytes %d", report.DispatchBoundTokens, len(b))
	}
	if report.EstimatedTokens >= report.DispatchBoundTokens {
		t.Fatal("estimate should be distinct from conservative bound")
	}
}

func TestV2SanitizationPreservesURLsAndDistinctPaths(t *testing.T) {
	c := Context{Candidate: Candidate{Text: `See https://example.com/account and /secret and "C:\Users\jeff\private.txt" and "/Users/jeff/My Files/日本語.txt"`}, TargetEpisode: SourceRecord{ID: "src-1", Text: "source /secret"}}
	p, err := BuildRequestV2(c)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	s := string(b)
	if !strings.Contains(s, "https://example.com/account") {
		t.Fatal("URL corrupted:", s)
	}
	for _, raw := range []string{"/secret", "private.txt", "My Files", "日本語.txt"} {
		if strings.Contains(s, raw) {
			t.Fatalf("path leaked %q: %s", raw, s)
		}
	}
	if p.State.Candidate.Text == p.State.TargetEpisode.Text {
		t.Fatal("distinct path evidence conflated")
	}
}
func TestV2RejectsUnsafeIDsAndTimesWithoutMutatingCaller(t *testing.T) {
	base := Context{Candidate: Candidate{Text: "fact"}, TargetEpisode: SourceRecord{ID: "source-1", Text: "text", OccurredAt: "2026-09-20T10:00:00Z"}, SourceHistory: []SourceRecord{{ID: "source-2", Text: "sk-abcdefghijklmnopqrstuvwxyz"}}, DerivedContext: []DerivedRecord{{ID: "derived-1", Text: "/secret", Kind: "fact", Provenance: "source-2", Limitation: "unverified"}}, EvaluationScope: EvaluationScope{TargetTime: "2026-09-20T10:00:00Z", SourceGaps: []string{"/secret"}}}
	if _, err := BuildRequestV2(base); err != nil {
		t.Fatal(err)
	}
	if base.SourceHistory[0].Text != "sk-abcdefghijklmnopqrstuvwxyz" || base.DerivedContext[0].Text != "/secret" || base.EvaluationScope.SourceGaps[0] != "/secret" {
		t.Fatal("mutated caller slices")
	}
	for _, id := range []string{"sk-abcdefghijklmnopqrstuvwxyz", "/secret"} {
		c := base
		c.TargetEpisode.ID = id
		if _, err := BuildRequestV2(c); err == nil {
			t.Fatalf("accepted unsafe ID %q", id)
		}
	}
	for _, timestamp := range []string{"not-a-date", "/secret", "2026-09-20"} {
		c := base
		c.TargetEpisode.OccurredAt = timestamp
		if _, err := BuildRequestV2(c); err == nil {
			t.Fatalf("accepted bad timestamp %q", timestamp)
		}
		c = base
		c.EvaluationScope.TargetTime = timestamp
		if _, err := BuildRequestV2(c); err == nil {
			t.Fatalf("accepted bad target time %q", timestamp)
		}
	}
}

func TestV2RedactsBacktickAndPathPrefix(t *testing.T) {
	c := Context{Candidate: Candidate{Text: "Use `/Users/jeff/private.txt` for storage. path:/Users/jeff/other.txt https://example.com/account"}, TargetEpisode: SourceRecord{ID: "src-1", Text: "source"}}
	p, err := BuildRequestV2(c)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	s := string(b)
	for _, v := range []string{"private.txt", "other.txt"} {
		if strings.Contains(s, v) {
			t.Fatalf("path leaked %s in %s", v, s)
		}
	}
	if !strings.Contains(s, "https://example.com/account") {
		t.Fatalf("URL changed: %s", s)
	}
}
