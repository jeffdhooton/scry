package extract

import (
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// Provider identifies the Messages-API-compatible endpoint the extractors
// talk to. An empty BaseURL means Anthropic itself; setting it points
// extraction at another provider that speaks the same wire format (DeepSeek's
// /anthropic endpoint, a local gateway), which is how the sweep is kept off
// Anthropic pricing without changing the prompt or the parsing.
type Provider struct {
	APIKey  string
	Model   string
	BaseURL string
}

// ProviderFromEnv resolves the extraction provider from the environment:
//
//	SCRY_MEMORY_API_KEY   the key, falling back to ANTHROPIC_API_KEY
//	SCRY_MEMORY_MODEL     the model id; empty defaults to defaultModel, which
//	                      is only meaningful on the Anthropic path (Validate)
//	SCRY_MEMORY_BASE_URL  a non-Anthropic endpoint, e.g.
//	                      https://api.deepseek.com/anthropic
func ProviderFromEnv() Provider {
	key := os.Getenv("SCRY_MEMORY_API_KEY")
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	return Provider{
		APIKey:  key,
		Model:   os.Getenv("SCRY_MEMORY_MODEL"),
		BaseURL: os.Getenv("SCRY_MEMORY_BASE_URL"),
	}
}

// Validate reports why a Provider cannot be used. A custom BaseURL must name
// its own model: defaultModel is an Anthropic model id, and sending it to
// another provider fails as an opaque 400 on a request that otherwise looks
// correct.
func (p Provider) Validate() error {
	if p.BaseURL != "" && p.Model == "" {
		return fmt.Errorf("extract: SCRY_MEMORY_BASE_URL is set (%s) but SCRY_MEMORY_MODEL is empty — a non-Anthropic endpoint must name its own model", p.BaseURL)
	}
	return nil
}

// Batched reports whether this provider supports the Message Batches API.
// Only Anthropic does: a compatible endpoint serves v1/messages but not
// v1/messages/batches, so backfill must run serially against one.
func (p Provider) Batched() bool { return p.BaseURL == "" }

// requestOptions builds the SDK options for p. option.WithBaseURL normalizes
// a base URL that has a path but no trailing slash, so both
// "https://host/anthropic" and "https://host/anthropic/" resolve v1/messages
// under the path rather than at the host root.
func (p Provider) requestOptions() []option.RequestOption {
	opts := []option.RequestOption{option.WithAPIKey(p.APIKey)}
	if p.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(p.BaseURL))
	}
	return opts
}

// resolveModel returns p.Model, falling back to defaultModel when empty.
func (p Provider) resolveModel() string {
	if p.Model == "" {
		return defaultModel
	}
	return p.Model
}
