package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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

	f, err := os.OpenFile(filepath.Join(dir, "mcp-calls.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
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
