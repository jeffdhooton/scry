package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAssessmentDefaultsOff(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := cfg.Memory.Assessment.Effective()
	if err != nil || a.Mode != "off" || a.Model != "jev-1.13.0" || a.TargetInputTokens != 20000 || a.Concurrency != 2 || a.Timeout != 15*time.Second {
		t.Fatalf("defaults: %+v %v", a, err)
	}
}

func TestAssessmentValidationDoesNotDisableExtraction(t *testing.T) {
	for _, tc := range []struct{ field, value string }{
		{"mode", "enforce"}, {"model", "jev-latest"}, {"target_input_tokens", "0"}, {"target_input_tokens", "60001"}, {"concurrency", "-1"}, {"timeout", "0s"}, {"timeout", "SECRET-invalid"}, {"concurrency", "SECRET-invalid"},
	} {
		t.Run(tc.field+tc.value, func(t *testing.T) {
			dir := t.TempDir()
			raw := "memory:\n  models:\n    - model: existing-extractor\n  assessment:\n    " + tc.field + ": " + tc.value + "\n"
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(dir)
			if err != nil {
				t.Fatalf("assessment broke extraction config: %v", err)
			}
			if len(cfg.Memory.Models) != 1 || cfg.Memory.Models[0].Model != "existing-extractor" {
				t.Fatal("lost extraction config")
			}
			_, err = cfg.Memory.Assessment.Effective()
			if err == nil || !strings.Contains(err.Error(), "memory.assessment."+tc.field) || strings.Contains(err.Error(), "SECRET") {
				t.Fatalf("unsafe or missing field error: %v", err)
			}
		})
	}
}

func TestAssessmentShadowConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("memory:\n  assessment:\n    mode: shadow\n    concurrency: 3\n    timeout: 10s\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, err := cfg.Memory.Assessment.Effective()
	if err != nil || a.Mode != "shadow" || a.Concurrency != 3 || a.Timeout != 10*time.Second || a.TargetInputTokens != 20000 {
		t.Fatalf("config: %+v %v", a, err)
	}
}
