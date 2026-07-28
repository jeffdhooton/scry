package extract

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// MaxBatchSize is the maximum number of requests the Anthropic Message
// Batches API accepts in a single batch. Run chunks eps into groups of at
// most this size and submits/polls them sequentially.
const MaxBatchSize = 1000

// pollInterval is how often Run polls an in-progress batch's status.
const pollInterval = 60 * time.Second

// BatchRunner submits episodes for extraction via the Anthropic Message
// Batches API — the same prompt/model/params shape as Haiku.Extract, at the
// 50% batch-processing discount, at the cost of latency (batches can take up
// to 24h, though in practice usually far less).
type BatchRunner struct {
	client anthropic.Client
	model  string
}

// NewBatchRunner builds a BatchRunner. An empty model defaults to
// defaultModel, same as NewHaiku.
func NewBatchRunner(apiKey, model string) *BatchRunner {
	if model == "" {
		model = defaultModel
	}
	return &BatchRunner{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// Run submits eps for extraction in chunks of at most MaxBatchSize,
// processing chunks sequentially: submit, poll every pollInterval until the
// batch's processing_status is "ended", then stream and route every result.
//
// The first return maps episode ID to its extracted Result for every
// succeeded request whose response parsed cleanly. The second maps episode
// ID to an error for every request that errored, was canceled, expired, or
// succeeded with a response that failed ParseResult — the two maps are
// disjoint and, on a non-fatal return, their union covers every episode ID
// in eps.
//
// The third return is fatal-only: a batch submission or poll failure (Get
// error, or ctx cancellation) that aborts Run before every chunk has been
// processed. On a fatal return, the first two maps still hold whatever was
// resolved by fully-completed chunks before the failure — callers should
// treat unresolved episode IDs (present in neither map) as not yet
// attempted, safe to retry on the next Run.
//
// progress, if non-nil, is called with the cumulative number of resolved
// episodes so far (across every chunk) and the overall total (len(eps)):
// once after each chunk's results are collected, and periodically (every
// pollInterval) while a chunk is still processing, using that chunk's
// in-flight request count to estimate how many of its requests have
// resolved.
func (b *BatchRunner) Run(ctx context.Context, eps []distill.RawEpisode, glossary []string, progress func(done, total int)) (map[string]Result, map[string]error, error) {
	results := make(map[string]Result)
	errs := make(map[string]error)
	total := len(eps)

	chunks := chunkEpisodes(eps)
	var done int

	for _, chunk := range chunks {
		reqs := buildBatchRequests(chunk, glossary, b.model)

		batch, err := b.client.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{Requests: reqs})
		if err != nil {
			return results, errs, fmt.Errorf("extract: batch submit failed: %w", err)
		}

		// batchID is captured once and never touched by the poll loop's error
		// path — the SDK returns (nil, err) on a Get failure, so reassigning
		// batch itself only on success (via a separate `got` variable) means
		// a transient poll failure (a single 5xx, a rate limit) can never
		// dereference a nil batch on the way out, which would otherwise
		// panic and crash a potentially hours-long backfill.
		batchID := batch.ID

		for batch.ProcessingStatus != anthropic.MessageBatchProcessingStatusEnded {
			select {
			case <-ctx.Done():
				return results, errs, ctx.Err()
			case <-time.After(pollInterval):
			}

			got, err := b.client.Messages.Batches.Get(ctx, batchID)
			if err != nil {
				return results, errs, fmt.Errorf("extract: batch %s: poll failed: %w", batchID, err)
			}
			batch = got

			if progress != nil {
				completedInChunk := len(chunk) - int(batch.RequestCounts.Processing)
				if completedInChunk < 0 {
					completedInChunk = 0
				}
				progress(done+completedInChunk, total)
			}
		}

		var items []batchItem
		stream := b.client.Messages.Batches.ResultsStreaming(ctx, batchID)
		for stream.Next() {
			items = append(items, toBatchItem(stream.Current()))
		}
		if err := stream.Err(); err != nil {
			return results, errs, fmt.Errorf("extract: batch %s: results stream failed: %w", batchID, err)
		}

		chunkResults, chunkErrs := routeBatchResults(items)
		for id, res := range chunkResults {
			results[id] = res
		}
		for id, err := range chunkErrs {
			errs[id] = err
		}

		done += len(chunk)
		if progress != nil {
			progress(done, total)
		}
	}

	return results, errs, nil
}

// chunkEpisodes splits eps into groups of at most MaxBatchSize, preserving
// order. An empty eps yields no chunks.
func chunkEpisodes(eps []distill.RawEpisode) [][]distill.RawEpisode {
	if len(eps) == 0 {
		return nil
	}
	var chunks [][]distill.RawEpisode
	for i := 0; i < len(eps); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(eps) {
			end = len(eps)
		}
		chunks = append(chunks, eps[i:end])
	}
	return chunks
}

// buildBatchRequests constructs one MessageBatchNewParamsRequest per
// episode, identical in shape (system blocks, cache_control, user message,
// model, max_tokens) to the single-shot params Haiku.Extract builds via
// buildMessages — the 50% batch discount applies regardless of caching, so
// there's no reason for the batch request shape to differ. custom_id is the
// episode's ID (a 64-char sha256 hex digest, well within the API's 64-char
// custom_id limit).
func buildBatchRequests(eps []distill.RawEpisode, glossary []string, model string) []anthropic.MessageBatchNewParamsRequest {
	reqs := make([]anthropic.MessageBatchNewParamsRequest, 0, len(eps))
	for _, ep := range eps {
		system, userMsg := buildMessages(ep, glossary)
		reqs = append(reqs, anthropic.MessageBatchNewParamsRequest{
			CustomID: ep.ID,
			Params: anthropic.MessageBatchNewParamsRequestParams{
				Model:     anthropic.Model(model),
				MaxTokens: 4000,
				System:    system,
				Messages:  []anthropic.MessageParam{userMsg},
			},
		})
	}
	return reqs
}

// batchResultKind mirrors MessageBatchResultUnion.Type: which of the four
// terminal outcomes an individual batch response landed in.
type batchResultKind string

const (
	resultSucceeded batchResultKind = "succeeded"
	resultErrored   batchResultKind = "errored"
	resultCanceled  batchResultKind = "canceled"
	resultExpired   batchResultKind = "expired"
)

// batchItem is the pure, testable projection of one line from the batch
// results stream (an anthropic.MessageBatchIndividualResponse): just enough
// to route it, without depending on the SDK's respjson-backed union
// internals in tests.
type batchItem struct {
	CustomID string
	Kind     batchResultKind
	Text     string // concatenated text blocks; populated only when Kind == resultSucceeded
	ErrMsg   string // API error message; populated only when Kind == resultErrored
}

// toBatchItem projects an SDK batch result line into the pure batchItem
// shape routeResult/routeBatchResults operate on.
func toBatchItem(resp anthropic.MessageBatchIndividualResponse) batchItem {
	item := batchItem{
		CustomID: resp.CustomID,
		Kind:     batchResultKind(resp.Result.Type),
	}
	switch item.Kind {
	case resultSucceeded:
		item.Text = concatText(&resp.Result.Message)
	case resultErrored:
		item.ErrMsg = resp.Result.Error.Error.Message
	}
	return item
}

// routeResult resolves one batchItem to either a parsed Result or an error:
// a succeeded item is passed through ParseResult (so a succeeded request
// whose text isn't valid JSON still lands as an error, same as Haiku.Extract
// after its retry fails); errored/canceled/expired items always error.
func routeResult(item batchItem) (Result, error) {
	switch item.Kind {
	case resultSucceeded:
		res, err := ParseResult(item.Text)
		if err != nil {
			return Result{}, fmt.Errorf("extract: batch result %s: %w", item.CustomID, err)
		}
		return res, nil
	case resultErrored:
		return Result{}, fmt.Errorf("extract: batch result %s: errored: %s", item.CustomID, item.ErrMsg)
	case resultCanceled:
		return Result{}, fmt.Errorf("extract: batch result %s: canceled", item.CustomID)
	case resultExpired:
		return Result{}, fmt.Errorf("extract: batch result %s: expired", item.CustomID)
	default:
		return Result{}, fmt.Errorf("extract: batch result %s: unknown result kind %q", item.CustomID, item.Kind)
	}
}

// routeBatchResults applies routeResult to every item, splitting into a
// results map and an errors map keyed by custom_id (episode ID). The two
// maps are disjoint: every item's CustomID lands in exactly one of them.
func routeBatchResults(items []batchItem) (map[string]Result, map[string]error) {
	results := make(map[string]Result, len(items))
	errs := make(map[string]error, len(items))
	for _, item := range items {
		res, err := routeResult(item)
		if err != nil {
			errs[item.CustomID] = err
			continue
		}
		results[item.CustomID] = res
	}
	return results, errs
}
