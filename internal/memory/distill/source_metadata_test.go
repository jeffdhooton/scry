package distill

import (
	"strings"
	"testing"
	"time"
)

func TestSourceMetadataPreservesOverlapAndSpeakers(t *testing.T) {
	turns := []turn{
		{role: "user", text: strings.Repeat("a", 7000), start: 0, end: 8000, ts: time.Unix(100, 0)},
		{role: "assistant", text: "same statement", start: 8000, end: 8100, ts: time.Unix(101, 0)},
		{role: "user", text: "same statement", start: 8100, end: 8200, ts: time.Unix(102, 0)},
		{role: "user", text: strings.Repeat("b", 10000), start: 8200, end: 19000, ts: time.Unix(103, 0)},
	}
	eps := chunkTurns("test-session", "/source/session", turns)
	if len(eps) != 2 {
		t.Fatalf("got %d episodes", len(eps))
	}
	for _, ep := range eps {
		if ep.SourceNamespace == "" || !ep.SourceSpanKnown || len(ep.SourceTurns) == 0 {
			t.Fatalf("missing metadata: %+v", ep)
		}
		if ep.SourceStart != ep.SourceTurns[0].Start || ep.SourceEnd != ep.SourceTurns[len(ep.SourceTurns)-1].End {
			t.Fatal("source span mismatch")
		}
	}
	if eps[0].SourceTurns[1].Speaker == eps[0].SourceTurns[2].Speaker {
		t.Fatal("identical text lost distinct speakers")
	}
	last := eps[0].SourceTurns[len(eps[0].SourceTurns)-1]
	first := eps[1].SourceTurns[0]
	if last != first {
		t.Fatal("overlap has different source identity")
	}
}

func TestSourceTurnMetadataRedacted(t *testing.T) {
	secret := "sk-ant-" + strings.Repeat("a", 40)
	eps := chunkTurns("test-session", "/source/session", []turn{{role: "user", text: "key " + secret, start: 0, end: 100, ts: time.Unix(100, 0)}})
	if strings.Contains(eps[0].Text, secret) || strings.Contains(eps[0].SourceTurns[0].Text, secret) {
		t.Fatal("metadata retained unredacted secret")
	}
}
