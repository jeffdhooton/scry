package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileIsZeroConfig(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for a missing config.yaml", err)
	}
	if len(cfg.Memory.Models) != 0 {
		t.Errorf("Memory.Models = %v, want empty", cfg.Memory.Models)
	}
}

func TestLoadParsesMemoryModels(t *testing.T) {
	home := t.TempDir()
	write(t, home, `
memory:
  models:
    - model: deepseek-v4-flash
    - model: claude-haiku-4-5
      base_url: https://api.anthropic.com
      api_key_env: ANTHROPIC_API_KEY
`)

	cfg, err := Load(home)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	got := cfg.Memory.Models
	if len(got) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(got))
	}
	if got[0].Model != "deepseek-v4-flash" || got[0].BaseURL != "" || got[0].APIKeyEnv != "" {
		t.Errorf("Models[0] = %+v, want bare deepseek-v4-flash", got[0])
	}
	if got[1].Model != "claude-haiku-4-5" || got[1].BaseURL != "https://api.anthropic.com" || got[1].APIKeyEnv != "ANTHROPIC_API_KEY" {
		t.Errorf("Models[1] = %+v", got[1])
	}
}

func TestLoadRejectsMalformedYAML(t *testing.T) {
	home := t.TempDir()
	write(t, home, "memory: [unclosed")

	if _, err := Load(home); err == nil {
		t.Fatal("Load() error = nil, want a parse error for malformed YAML")
	}
}

func TestLoadRejectsModelEntryWithoutModel(t *testing.T) {
	home := t.TempDir()
	write(t, home, `
memory:
  models:
    - base_url: https://api.anthropic.com
`)

	_, err := Load(home)
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("Load() error = %v, want an error naming the missing model field", err)
	}
}

func write(t *testing.T, home, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
