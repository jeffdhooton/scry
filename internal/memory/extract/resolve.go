package extract

import (
	"fmt"
	"os"

	"github.com/jeffdhooton/scry/internal/config"
)

// Providers is the resolved extraction chain: Providers[0] is tried first
// and each later entry only when the one before it failed. Source names
// where the chain came from ("config.yaml" or "env") so the daemon can say
// which settings it is honouring — and which it is ignoring.
type Providers struct {
	Providers []Provider
	Source    string
}

// ResolveProviders builds the chain from cfg, falling back to the
// environment (ProviderFromEnv) when cfg names no models. A config chain
// replaces SCRY_MEMORY_MODEL and SCRY_MEMORY_BASE_URL outright rather than
// merging with them: a file the user wrote on purpose should mean exactly
// what it says. Keys are still env-only — each entry's api_key_env names the
// variable, defaulting to the SCRY_MEMORY_API_KEY / DEEPSEEK_API_KEY lookup.
func ResolveProviders(cfg config.Config) Providers {
	if len(cfg.Memory.Models) == 0 {
		return Providers{Providers: []Provider{ProviderFromEnv()}, Source: "env"}
	}
	defaultKey := ProviderFromEnv().APIKey
	out := make([]Provider, 0, len(cfg.Memory.Models))
	for _, m := range cfg.Memory.Models {
		key, keyEnv := defaultKey, "SCRY_MEMORY_API_KEY"
		if m.APIKeyEnv != "" {
			key, keyEnv = os.Getenv(m.APIKeyEnv), m.APIKeyEnv
		}
		out = append(out, Provider{APIKey: key, Model: m.Model, BaseURL: m.BaseURL, KeyEnv: keyEnv})
	}
	return Providers{Providers: out, Source: "config.yaml"}
}

// Dormant reports whether the chain cannot run at all because its primary
// has no key. This is the same "no key → dormant" rule as before; a missing
// key on a later entry is a Validate error instead, since that is a
// misconfiguration rather than an opt-out.
func (ps Providers) Dormant() bool {
	return len(ps.Providers) == 0 || ps.Providers[0].APIKey == ""
}

// Validate checks every entry, naming the offending model so a chain with
// one bad link is diagnosable without guessing.
func (ps Providers) Validate() error {
	for i, p := range ps.Providers {
		if err := p.Validate(); err != nil {
			return err
		}
		if i > 0 && p.APIKey == "" {
			return fmt.Errorf("extract: memory.models[%d] (%s) has no API key — %s is empty", i, p.Model, p.KeyEnv)
		}
	}
	return nil
}

// Models lists the chain's model ids in order, for logs and status output.
func (ps Providers) Models() []string {
	out := make([]string, len(ps.Providers))
	for i, p := range ps.Providers {
		out[i] = p.resolveModel()
	}
	return out
}
