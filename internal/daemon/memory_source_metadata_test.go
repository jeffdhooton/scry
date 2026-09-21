package daemon

import (
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

func TestEnqueuePreservesRedactedRemoteSourceMetadata(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, err := d.memoryStore()
	if err != nil {
		t.Fatal(err)
	}
	key := "sk-ant-" + strings.Repeat("x", 40)
	ep := distill.RawEpisode{ID: "metadata", Source: "codex-session", SourceRef: "/unreadable/remote/session#0-200", Text: "User: " + key, OccurredAt: time.Unix(100, 0), SourceNamespace: "remote-host", SourceSpanKnown: true, SourceStart: 0, SourceEnd: 200, SourceTurns: []distill.SourceTurn{{Speaker: "user", Text: key, Start: 0, End: 200}}}
	ok, err := enqueueEpisode(st, ep, nil, time.Now(), false)
	if err != nil || !ok {
		t.Fatalf("enqueue %v %v", ok, err)
	}
	p, err := st.GetPending(ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.SourceNamespace != "remote-host" || !p.SourceSpanKnown || p.SourceStart != 0 || p.SourceEnd != 200 || len(p.SourceTurns) != 1 {
		t.Fatalf("metadata lost: %+v", p)
	}
	if strings.Contains(p.Text, key) || strings.Contains(p.SourceTurns[0].Text, key) {
		t.Fatal("unredacted metadata persisted")
	}
	if ep.SourceTurns[0].Text != key {
		t.Fatal("intake mutated caller")
	}
}
