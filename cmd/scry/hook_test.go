package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHookContextDoesNotOverridePermissions(t *testing.T) {
	for _, context := range []string{"", "Use scry for symbol lookups."} {
		t.Run(context, func(t *testing.T) {
			output, err := os.Create(filepath.Join(t.TempDir(), "stdout.json"))
			if err != nil {
				t.Fatal(err)
			}
			defer output.Close()
			original := os.Stdout
			os.Stdout = output
			defer func() { os.Stdout = original }()
			if err := writeHookAllow(context); err != nil {
				t.Fatal(err)
			}
			if _, err := output.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			var result struct {
				Output map[string]any `json:"hookSpecificOutput"`
			}
			if err := json.NewDecoder(output).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if _, exists := result.Output["permissionDecision"]; exists {
				t.Error("context-only hook must leave permission decisions to the host")
			}
			if result.Output["hookEventName"] != "PreToolUse" {
				t.Errorf("unexpected event: %v", result.Output)
			}
			if context != "" && result.Output["additionalContext"] != context {
				t.Errorf("lost hook context: %v", result.Output)
			}
		})
	}
}
