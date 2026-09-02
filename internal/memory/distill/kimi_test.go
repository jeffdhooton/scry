package distill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const kimiFixture = "testdata/kimi_session/agents/main/wire.jsonl"

func TestKimiWireDistillsPromptsAndAssistantText(t *testing.T) {
	episodes, offset, err := KimiWire(kimiFixture, 0)
	if err != nil {
		t.Fatalf("KimiWire: %v", err)
	}
	info, _ := os.Stat(kimiFixture)
	if offset != info.Size() {
		t.Errorf("offset = %d, want file size %d", offset, info.Size())
	}
	if len(episodes) != 1 {
		t.Fatalf("episodes = %d, want 1", len(episodes))
	}
	ep := episodes[0]
	if ep.Source != "kimi-session" {
		t.Errorf("Source = %q", ep.Source)
	}
	if ep.Cwd != "/Users/jeff/workspace/context-stack/scry" {
		t.Errorf("Cwd = %q, want the session state.json cwd", ep.Cwd)
	}
	if ep.OccurredAt.Year() != 2026 {
		t.Errorf("OccurredAt = %v, want a 2026 timestamp from the prompt event", ep.OccurredAt)
	}
	for _, want := range []string{
		"User: Deploy scry to the mini",
		"Assistant: [tool: Shell]",
		"Built scry-816e18a and copied it",
		"User: Did the daemon come back healthy?",
		"launchctl kickstart -k",
	} {
		if !strings.Contains(ep.Text, want) {
			t.Errorf("episode text missing %q:\n%s", want, ep.Text)
		}
	}
	for _, never := range []string{"I should build first", "SECRET OUTPUT", "sk-abcdefghijklmnopqrstuvwxyz123456", "system-reminder"} {
		if strings.Contains(ep.Text, never) {
			t.Errorf("episode text must not contain %q:\n%s", never, ep.Text)
		}
	}
	if !strings.HasPrefix(ep.SourceRef, kimiFixture+"#") {
		t.Errorf("SourceRef = %q", ep.SourceRef)
	}
}

func TestKimiWireResumesFromOffsetWithNothingNew(t *testing.T) {
	info, _ := os.Stat(kimiFixture)
	episodes, offset, err := KimiWire(kimiFixture, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	if len(episodes) != 0 || offset != info.Size() {
		t.Errorf("resume at EOF: episodes %d offset %d", len(episodes), offset)
	}
}

func TestKimiWireTooShortSessionYieldsNothingButAdvances(t *testing.T) {
	dir := t.TempDir()
	agent := filepath.Join(dir, "sess", "agents", "main")
	if err := os.MkdirAll(agent, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(agent, "wire.jsonl")
	body := `{"type":"turn.prompt","input":[{"type":"text","text":"hi"}],"time":1787187201000}` + "\n" +
		`{"type":"turn.ended","turnId":0,"reason":"completed","time":1787187201500}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	episodes, offset, err := KimiWire(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(episodes) != 0 || offset != int64(len(body)) {
		t.Errorf("short session: episodes %d offset %d want 0 and %d", len(episodes), offset, len(body))
	}
}

func TestKimiSessionCwdMissingStateIsEmpty(t *testing.T) {
	if got := kimiSessionCwd(filepath.Join(t.TempDir(), "a", "agents", "main", "wire.jsonl")); got != "" {
		t.Errorf("cwd = %q, want empty", got)
	}
}
