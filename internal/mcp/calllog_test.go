package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogCallWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, ".scry", "logs")

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	logCall(callLogEntry{
		Timestamp: "2026-04-11T14:32:01Z",
		Tool:      "scry_refs",
		Symbol:    "Registry",
		Repo:      "/tmp/test-repo",
		Results:   14,
		LatencyMs: 3,
	})
	logCall(callLogEntry{
		Timestamp: "2026-04-11T14:32:02Z",
		Tool:      "scry_defs",
		Symbol:    "Server",
		Repo:      "/tmp/test-repo",
		Results:   1,
		LatencyMs: 2,
		Error:     "",
	})

	data, err := os.ReadFile(filepath.Join(logDir, "mcp-calls.jsonl"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var entry callLogEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("unmarshal line 0: %v", err)
	}
	if entry.Tool != "scry_refs" || entry.Symbol != "Registry" || entry.Results != 14 {
		t.Errorf("unexpected first entry: %+v", entry)
	}
}

func TestExtractResultCount(t *testing.T) {
	raw := json.RawMessage(`{"symbol":"foo","matches":[{}],"total":42}`)
	if n := extractResultCount(raw); n != 42 {
		t.Errorf("extractResultCount = %d, want 42", n)
	}

	raw = json.RawMessage(`{"nope": true}`)
	if n := extractResultCount(raw); n != 0 {
		t.Errorf("extractResultCount on missing total = %d, want 0", n)
	}
}
