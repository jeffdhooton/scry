package extract

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// extractionMaxTokens is the output budget for one extraction call.
//
// The default extractor is a reasoning model and reasoning is charged against
// this same budget, so at 8000 a long episode could spend the whole allowance
// thinking and return no text -- stop_reason "max_tokens", content blocks
// [thinking] -- which parses as nothing and lost the memory write outright.
//
// The ceiling cannot simply be raised: the SDK refuses a non-streaming
// request whose budget implies more than ten minutes of work. So extraction
// turns thinking off instead (see Extract) and keeps a budget sized for the
// JSON answer plus room to spare.
const extractionMaxTokens = 16000

// retryMaxTokens is what a truncated reply is retried with. Buying room is
// the only response that can help; a correction prompt cannot, because the
// model never produced anything to correct.
//
// It sits just under the SDK's non-streaming ceiling. That client requires
// streaming once 1h * max_tokens / 128000 exceeds ten minutes, which puts the
// hard limit at 21333 tokens for a request that is not streamed.
const retryMaxTokens = 21000

// Haiku is an Extractor speaking the Anthropic Messages wire format. The
// system prompt (stable) and glossary block (volatile) are sent as separate
// system blocks so the stable prefix stays cacheable across calls. The name
// is historical: it talks to whatever Provider it is given, which by default
// is DeepSeek V4, not Anthropic.
type Haiku struct {
	client anthropic.Client
	model  string
}

// NewHaiku builds a Haiku extractor against p. An empty p.Model defaults to
// defaultModel (DeepSeek V4); see Provider.Validate for why an explicit
// endpoint must name its own model.
func NewHaiku(p Provider) *Haiku {
	return &Haiku{
		client: anthropic.NewClient(p.requestOptions()...),
		model:  p.resolveModel(),
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
		MaxTokens: extractionMaxTokens,
		System:    system,
		Messages:  []anthropic.MessageParam{userMsg},
		// Extraction is a structured-output task against a fixed schema, so
		// chain-of-thought buys nothing and costs the whole budget when the
		// model decides to deliberate. Providers that ignore this field are
		// still covered by the truncation retry below.
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfDisabled: &anthropic.ThinkingConfigDisabledParam{},
		},
	}

	resp, err := h.client.Messages.New(ctx, params)
	if err != nil {
		return Result{}, fmt.Errorf("extract: haiku request failed: %w", err)
	}

	// A reply that ran out of budget before emitting any text is not a
	// malformed answer, it is an absent one. The repair prompts below ask the
	// model to fix JSON it never wrote, so they burn both attempts and lose
	// the fact. Buy room and ask again instead.
	if truncatedBeforeText(resp) {
		params.MaxTokens = retryMaxTokens
		wider, werr := h.client.Messages.New(ctx, params)
		if werr != nil {
			return Result{}, fmt.Errorf("extract: haiku wider-budget request failed: %w", werr)
		}
		resp = wider
	}

	lastResp := resp
	raw := concatText(resp)
	result, parseErr := ParseResult(raw)
	if parseErr == nil {
		return result, nil
	}

	// Two corrective retries. The first echoes the invalid reply and asks for
	// a correction; the second is an explicit repair instruction, because a
	// model that produced prose once tends to produce prose twice. A real
	// fact was lost to a single-retry give-up — the extra attempt is far
	// cheaper than the loss.
	repairs := []string{
		"Invalid JSON (%s). Return only the corrected JSON object.",
		"Still invalid JSON (%s). Return ONLY the JSON object: no prose, no " +
			"explanation, no markdown code fence. Start your reply with { and end it with }.",
	}
	for _, tmpl := range repairs {
		params.Messages = append(params.Messages,
			anthropic.NewAssistantMessage(anthropic.NewTextBlock(raw)),
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				fmt.Sprintf(tmpl, parseErr),
			)),
		)
		next, rerr := h.client.Messages.New(ctx, params)
		if rerr != nil {
			return Result{}, fmt.Errorf("extract: haiku retry request failed: %w", rerr)
		}
		lastResp = next
		raw = concatText(next)
		result, parseErr = ParseResult(raw)
		if parseErr == nil {
			return result, nil
		}
	}
	// The raw reply rides along on the error so the caller can dead-letter the
	// exact text rather than only the parse failure. An empty reply also
	// carries the stop_reason: `reply: ""` alone says nothing about whether
	// the model hit max_tokens, refused, or returned only non-text blocks.
	return Result{}, fmt.Errorf("extract: invalid JSON after 2 repairs: %w: %w (reply: %.400q%s)",
		ErrParse, parseErr, raw, emptyReplyDetail(lastResp, raw))
}

// truncatedBeforeText reports whether a reply hit its output ceiling without
// producing any text. That is the shape a reasoning model returns when
// deliberation consumed the whole budget.
func truncatedBeforeText(msg *anthropic.Message) bool {
	return msg != nil &&
		string(msg.StopReason) == "max_tokens" &&
		concatText(msg) == ""
}

// emptyReplyDetail describes why a reply had no text, for the error message.
func emptyReplyDetail(msg *anthropic.Message, raw string) string {
	if raw != "" || msg == nil {
		return ""
	}
	kinds := make([]string, 0, len(msg.Content))
	for _, block := range msg.Content {
		kinds = append(kinds, string(block.Type))
	}
	return fmt.Sprintf(", stop_reason=%s, content_blocks=[%s]", msg.StopReason, strings.Join(kinds, ","))
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
