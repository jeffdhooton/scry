package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffdhooton/scry/internal/logrotate"
)

const (
	// One line per MCP call, so this fills at the pace the agent works.
	// 16 MB is weeks of history and small enough to grep.
	maxCallLogBytes    = 16 << 20
	callLogGenerations = 3
)

type callLogEntry struct {
	Timestamp string `json:"ts"`
	Tool      string `json:"tool"`
	Symbol    string `json:"symbol,omitempty"`
	Repo      string `json:"repo"`
	Results   int    `json:"results"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

func logCall(entry callLogEntry) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".scry", "logs")
	_ = os.MkdirAll(dir, 0o755)

	f, err := logrotate.OpenAppend(
		filepath.Join(dir, "mcp-calls.jsonl"),
		maxCallLogBytes,
		callLogGenerations,
		0o644,
	)
	if err != nil {
		return
	}
	defer f.Close()

	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(f, "%s\n", b)
}

func extractResultCount(raw json.RawMessage) int {
	var result struct {
		Total int `json:"total"`
	}
	if json.Unmarshal(raw, &result) == nil {
		return result.Total
	}
	return -1
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
