package extract

import (
	"strings"
	"testing"

	"github.com/jeffdhooton/scry/internal/config"
)

func TestResolveProvidersWithoutConfigUsesEnv(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "k")
	t.Setenv("SCRY_MEMORY_MODEL", "env-model")
	t.Setenv("SCRY_MEMORY_BASE_URL", "https://env.example/anthropic")

	got := ResolveProviders(config.Config{})

	if got.Source != "env" {
		t.Errorf("Source = %q, want env", got.Source)
	}
	if len(got.Providers) != 1 {
		t.Fatalf("len(Providers) = %d, want 1", len(got.Providers))
	}
	if p := got.Providers[0]; p.APIKey != "k" || p.Model != "env-model" || p.BaseURL != "https://env.example/anthropic" {
		t.Errorf("Providers[0] = %+v, want the env provider", p)
	}
}

func TestResolveProvidersConfigChainIgnoresEnvModel(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "k")
	t.Setenv("SCRY_MEMORY_MODEL", "env-model")
	t.Setenv("SCRY_MEMORY_BASE_URL", "https://env.example/anthropic")
	cfg := config.Config{Memory: config.Memory{Models: []config.Model{
		{Model: "deepseek-v4-flash"},
		{Model: "deepseek-v4-pro"},
	}}}

	got := ResolveProviders(cfg)

	if got.Source != "config.yaml" {
		t.Errorf("Source = %q, want config.yaml", got.Source)
	}
	if len(got.Providers) != 2 {
		t.Fatalf("len(Providers) = %d, want 2", len(got.Providers))
	}
	for i, want := range []string{"deepseek-v4-flash", "deepseek-v4-pro"} {
		p := got.Providers[i]
		if p.Model != want {
			t.Errorf("Providers[%d].Model = %q, want %q", i, p.Model, want)
		}
		if p.BaseURL != "" {
			t.Errorf("Providers[%d].BaseURL = %q, want empty (DeepSeek default) — env base URL must not leak into a config chain", i, p.BaseURL)
		}
		if p.APIKey != "k" {
			t.Errorf("Providers[%d].APIKey = %q, want the default env key", i, p.APIKey)
		}
	}
}

func TestResolveProvidersHonorsAPIKeyEnv(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "deepseek-key")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")
	cfg := config.Config{Memory: config.Memory{Models: []config.Model{
		{Model: "deepseek-v4-flash"},
		{Model: "claude-haiku-4-5", BaseURL: "https://api.anthropic.com", APIKeyEnv: "ANTHROPIC_API_KEY"},
	}}}

	got := ResolveProviders(cfg).Providers

	if got[0].APIKey != "deepseek-key" {
		t.Errorf("Providers[0].APIKey = %q, want deepseek-key", got[0].APIKey)
	}
	if got[1].APIKey != "anthropic-key" || got[1].BaseURL != "https://api.anthropic.com" {
		t.Errorf("Providers[1] = %+v, want the Anthropic key and base URL", got[1])
	}
}

// TestResolvedProvidersValidateReportsFirstBadEntry: a chain entry with no
// key resolvable is as unusable as a missing key was before — and the error
// has to say which entry, because the chain hides which one is empty.
func TestResolvedProvidersValidateNamesEmptyKeyEntry(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_KEY", "")
	cfg := config.Config{Memory: config.Memory{Models: []config.Model{
		{Model: "deepseek-v4-flash"},
		{Model: "claude-haiku-4-5", BaseURL: "https://api.anthropic.com", APIKeyEnv: "ANTHROPIC_API_KEY"},
	}}}

	err := ResolveProviders(cfg).Validate()

	if err == nil {
		t.Fatal("Validate() = nil, want an error for the entry whose ANTHROPIC_API_KEY is empty")
	}
	for _, want := range []string{"claude-haiku-4-5", "ANTHROPIC_API_KEY"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() error = %q, want it to mention %q", err, want)
		}
	}
}

func TestResolvedProvidersDormantWhenNoKey(t *testing.T) {
	t.Setenv("SCRY_MEMORY_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	if got := ResolveProviders(config.Config{}); !got.Dormant() {
		t.Errorf("Dormant() = false, want true with no key in the environment")
	}
	cfg := config.Config{Memory: config.Memory{Models: []config.Model{{Model: "deepseek-v4-flash"}}}}
	if got := ResolveProviders(cfg); !got.Dormant() {
		t.Errorf("Dormant() = false for a config chain with no key, want true")
	}
}
