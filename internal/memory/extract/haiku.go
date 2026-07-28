package extract

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

const defaultModel = "claude-haiku-4-5"

// Haiku is an Extractor backed by the Anthropic Messages API, defaulting
// to Claude Haiku. The system prompt (stable) and glossary block (volatile)
// are sent as separate system blocks so the stable prefix stays cacheable
// across calls.
type Haiku struct {
	client anthropic.Client
	model  string
}

// NewHaiku builds a Haiku extractor. An empty model defaults to
// "claude-haiku-4-5".
func NewHaiku(apiKey, model string) *Haiku {
	if model == "" {
		model = defaultModel
	}
	return &Haiku{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// buildMessages assembles the system blocks and user message for an
// extraction call. SystemPrompt is first with a cache-control breakpoint
// (stable across calls); the glossary is a second, uncached system block
// (varies per call). The user message is
// "reference_time: <RFC3339>\ncwd: <cwd>\n\n<text>".
func buildMessages(ep distill.RawEpisode, glossary []string) ([]anthropic.TextBlockParam, anthropic.MessageParam) {
	system := []anthropic.TextBlockParam{
		{
			Text:         SystemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		},
		{
			Text: "KNOWN ENTITIES (canonical name: aliases):\n" + strings.Join(glossary, "\n"),
		},
	}

	userText := "reference_time: " + ep.OccurredAt.Format(time.RFC3339) + "\ncwd: " + ep.Cwd + "\n\n" + ep.Text
	userMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(userText))

	return system, userMsg
}

// Extract calls Haiku with the episode text and glossary, parses the JSON
// response into a Result, and retries exactly once (appending the
// assistant's reply and a corrective user turn) if the first response
// fails to parse.
func (h *Haiku) Extract(ctx context.Context, ep distill.RawEpisode, glossary []string) (Result, error) {
	system, userMsg := buildMessages(ep, glossary)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(h.model),
		MaxTokens: 4000,
		System:    system,
		Messages:  []anthropic.MessageParam{userMsg},
	}

	resp, err := h.client.Messages.New(ctx, params)
	if err != nil {
		return Result{}, fmt.Errorf("extract: haiku request failed: %w", err)
	}

	raw := concatText(resp)
	result, parseErr := ParseResult(raw)
	if parseErr == nil {
		return result, nil
	}

	// One corrective retry: echo the assistant's (invalid) reply, then ask
	// for a corrected JSON object.
	params.Messages = append(params.Messages,
		anthropic.NewAssistantMessage(anthropic.NewTextBlock(raw)),
		anthropic.NewUserMessage(anthropic.NewTextBlock(
			fmt.Sprintf("Invalid JSON (%s). Return only the corrected JSON object.", parseErr),
		)),
	)

	resp2, err := h.client.Messages.New(ctx, params)
	if err != nil {
		return Result{}, fmt.Errorf("extract: haiku retry request failed: %w", err)
	}

	raw2 := concatText(resp2)
	result2, parseErr2 := ParseResult(raw2)
	if parseErr2 != nil {
		return Result{}, fmt.Errorf("extract: invalid JSON after retry: %w", parseErr2)
	}
	return result2, nil
}

// concatText joins every text content block in a response, in order.
// Haiku responses are expected to contain exactly one text block, but this
// tolerates more without dropping content.
func concatText(msg *anthropic.Message) string {
	var sb strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String()
}
