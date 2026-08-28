package extract

import (
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// Extraction defaults to DeepSeek V4, not Anthropic. The memory sweep runs
// unattended every 30 minutes over every transcript on the machine, so the
// default has to be the cheap provider: an Anthropic default is a bill that
// arrives silently whenever the env is incomplete. defaultModel is pinned to
// an explicit V4 id rather than the floating "deepseek-chat" alias, so the
// sweep cannot be moved onto a different generation by an upstream alias
// repoint.
const (
	defaultBaseURL = "https://api.deepseek.com/anthropic"
	defaultModel   = "deepseek-v4-flash"
	anthropicHost  = "api.anthropic.com"
)

// Provider identifies the Messages-API-compatible endpoint the extractors
// talk to. An empty BaseURL means defaultBaseURL (DeepSeek); setting it
// points extraction at any other provider speaking the same wire format.
// Reaching Anthropic requires naming it explicitly in SCRY_MEMORY_BASE_URL —
// there is no path there by omission.
type Provider struct {
	APIKey  string
	Model   string
	BaseURL string
	// KeyEnv names the environment variable APIKey was read from. It is
	// diagnostic only — so an error about a missing key can say which
	// variable to set.
	KeyEnv string
}

// ProviderFromEnv resolves the extraction provider from the environment:
//
//	SCRY_MEMORY_API_KEY   the key, falling back to DEEPSEEK_API_KEY
//	SCRY_MEMORY_MODEL     the model id; empty defaults to defaultModel
//	SCRY_MEMORY_BASE_URL  override the endpoint; empty means defaultBaseURL
//
// ANTHROPIC_API_KEY is deliberately not consulted: it is present in most of
// these shells for unrelated reasons, and honouring it here is what turns a
// half-configured env into Anthropic spend.
func ProviderFromEnv() Provider {
	key := os.Getenv("SCRY_MEMORY_API_KEY")
	if key == "" {
		key = os.Getenv("DEEPSEEK_API_KEY")
	}
	return Provider{
		APIKey:  key,
		Model:   os.Getenv("SCRY_MEMORY_MODEL"),
		BaseURL: os.Getenv("SCRY_MEMORY_BASE_URL"),
	}
}

// Validate reports why a Provider cannot be used. An explicit BaseURL must
// name its own model: defaultModel is a DeepSeek id, and sending it to a
// different provider fails as an opaque 400 on a request that otherwise looks
// correct.
func (p Provider) Validate() error {
	if p.BaseURL != "" && p.Model == "" {
		return fmt.Errorf("extract: SCRY_MEMORY_BASE_URL is set (%s) but SCRY_MEMORY_MODEL is empty — an endpoint other than the default must name its own model", p.BaseURL)
	}
	return nil
}

// Batched reports whether this provider supports the Message Batches API.
// Only Anthropic does: a compatible endpoint serves v1/messages but not
// v1/messages/batches, so backfill must run serially against one. With
// DeepSeek as the default this is false unless Anthropic is named outright.
func (p Provider) Batched() bool {
	return strings.Contains(p.resolveBaseURL(), anthropicHost)
}

// resolveBaseURL returns p.BaseURL, falling back to defaultBaseURL.
func (p Provider) resolveBaseURL() string {
	if p.BaseURL == "" {
		return defaultBaseURL
	}
	return p.BaseURL
}

// requestOptions builds the SDK options for p. option.WithBaseURL normalizes
// a base URL that has a path but no trailing slash, so both
// "https://host/anthropic" and "https://host/anthropic/" resolve v1/messages
// under the path rather than at the host root.
func (p Provider) requestOptions() []option.RequestOption {
	return []option.RequestOption{
		option.WithAPIKey(p.APIKey),
		option.WithBaseURL(p.resolveBaseURL()),
	}
}

// resolveModel returns p.Model, falling back to defaultModel when empty.
func (p Provider) resolveModel() string {
	if p.Model == "" {
		return defaultModel
	}
	return p.Model
}
