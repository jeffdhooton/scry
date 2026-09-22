# Evaluate proposed memories with Jev

`scry memory assess --file` is an isolated evaluation command. It compares Jev's
classifications with labels in a supplied JSON file. It does not connect to a
Scry daemon, read the memory store, run the extractor, or change stored facts.

From the Scry repository, preview the bundled synthetic cases:

```sh
go run ./cmd/scry memory assess --file docs/memory-assess/synthetic.json
```

Preview is the default even when `TYPESAFE_API_KEY` is set. It prints the exact
redacted requests, without expected labels. No network call is made.

To deliberately run a paid evaluation later, set `TYPESAFE_API_KEY` in your
shell, then run:

```sh
go run ./cmd/scry memory assess \
  --file docs/memory-assess/synthetic.json --live > /tmp/jev-assessment.json
```

The command sends the supplied episode/fact text to TypeSafe. Start with the
bundled synthetic cases. Scry's existing redactor removes known secret shapes;
it does not remove all confidential information. Review a dataset before sending
it to a new provider. The first hosted run completed all 14 synthetic cases on
2026-09-20; see the [baseline results](results/2026-09-20-baseline.md). These
development examples do not establish production accuracy.

The [expanded evaluation](results/2026-09-20-expanded-evaluation.md) adds 64
challenge cases and 12 repeated-anchor calls, with the original model, rubric
and threshold held fixed. See the [protocol](EXPANDED_EVAL.md) for coverage and
limitations. Preview either additional dataset without network calls:

```sh
go run ./cmd/scry memory assess --file docs/memory-assess/expanded-v1.json
go run ./cmd/scry memory assess --file docs/memory-assess/repeatability-v1.json
```

The [context experiment](context-experiment/NOTES.md) compares short inputs,
JSON-wrapped inputs, added relevant sources and roughly 20k-token inputs across
192 calls. Support agreement increased from 42/48 for compact inputs to 46/48
with relevant sources and 48/48 with long context. Durability remained imperfect.
These are targeted synthetic results, not production accuracy estimates.

## What is assessed

Every episode/fact pair produces one request with three independent questions:

- `supported` (noul): whether the whole proposed fact is supported by the episode,
  preserving attribution, negation, time and uncertainty.
- `durable` (noul): whether the fact would be useful across sessions, assuming it
  were true. This is deliberately independent of support. A valid future plan
  can be durable; incidental progress chatter is usually not.
- `assertion` (choice): how the episode presents the underlying claim:
  `established`, `planned`, `hypothetical`, `denied`, or `unclear`.

An established claim is not independently verified truth. A transcript can say
that something happened without proving it. Scry's normal distillation omits
tool-result bodies; this assessor cannot recover that missing evidence.

The rubric is versioned as `memory-assess-v1`; the model is pinned to
`jev-1.13.0`. Change the rubric version when changing its meaning. Re-evaluate
predictions before changing the model. Choice confidence summarizes the shape
of its probability distribution; it is not interchangeable with the probability
of the chosen option or a measured correctness rate.

## Dataset format

Supply a JSON array of 1–100 cases, at most 8 MiB. IDs must be unique. Both boolean
labels and the assertion label are required. Unknown fields are rejected.
Labels are evaluation expectations, never model instructions.

```json
[
  {
    "id": "keep-future-plan",
    "episode": "User: We plan to move Lantern's cache to Redis next month.",
    "fact": "The team plans to move Lantern's cache to Redis next month.",
    "expected": {
      "supported": true,
      "durable": true,
      "assertion": "planned"
    }
  }
]
```

The 14 bundled cases are synthetic development examples with proposed labels. They include
unsupported claims, plans mistaken for completed work, retractions, durable
preferences, valid negated constraints, and instructions embedded in evidence.
They are not a representative holdout or proof of model accuracy. Review labels
and create a separate held-out set before tuning thresholds or changing memory
admission.

## Report and failures

Live mode emits JSON containing:

- Every completed case's expected labels, probabilities, model, token usage and
  elapsed HTTP time, plus a SHA-256 digest of its redacted request. Source text
  is omitted from reports; case IDs are retained, so keep IDs non-sensitive.
- Correct, false-positive and false-negative counts for support and durability,
  using a diagnostic threshold of 0.5; assertion correctness count.
- Mean Brier scores for support/durability (lower is better).
- Nearest-rank p50/p95 request latency, token totals and estimated API cost.

Divide correctness counts by `completed` for accuracy. When `completed` is zero,
metrics have no observations and their zero values must not be interpreted as
perfect performance. These diagnostics never filter or admit memories.

Price is a recorded assumption: $0.042 per million input tokens, output free.
At that price 10,000 requests averaging 5,000 input tokens cost $2.10. The report
uses actual returned token counts from successfully parsed responses; failed
requests might also be billed, so the estimate is not an invoice. Extraction
costs are separate.

Requests run sequentially with a 15-second timeout. Any HTTP error, redirect,
transport error, model mismatch, or malformed answer stops the run without
retrying or switching models. Successful prior results are emitted as a partial
report, and the process exits nonzero. Compare `requested` with `completed`.
Provider response bodies and credentials are not included in error messages.

## Tests

```sh
go test ./internal/memory/assess
go test ./cmd/scry -run TestMemoryAssess
```

Tests exercise the real HTTP adapter against a local server and the real report
calculation against deliberately chosen responses. They prove integration and
metric behavior, not Jev's classification quality. No API key is required.

## Daemon shadow assessment

The integration and its authorized synthetic live evaluation are documented in
[the acceptance audit](INTEGRATION_AUDIT.md) and
[live results](integration-v1/LIVE_REPORT.md): 592 completed assessments and eight
terminal failures from 600 frozen requests. The results support a shadow trial,
with no recommendation to enforce admission or retention.

The daemon can record Jev assessments of extractor-proposed facts in a separate
sidecar. The default is `memory.assessment.mode: off`. To enable observation on
the store machine, set `memory.assessment.mode: shadow` in its `~/.scry/config.yaml`,
put `TYPESAFE_API_KEY` in the **daemon's** environment, and restart the daemon.
A shell export on a laptop does not reach the remote launchd service. The only
supported production model is pinned to `jev-1.13.0`; assessment never changes
fact admission or extraction. Mode changes require a daemon restart.

All commands below query the configured memory daemon, including a remote
`SCRY_MEMORY_SOCKET` or `memory.socket` tunnel. JSON is the default output:

```sh
scry memory assess status --json
scry memory assess list --episode EPISODE_ID --limit 50 --json
scry memory assess list --after NEXT_CURSOR --limit 50 --json
scry memory assess show ASSESSMENT_ID --json
scry memory assess show ASSESSMENT_ID --include-context --json
scry memory assess replay ASSESSMENT_ID            # preview retained request
scry memory assess replay ASSESSMENT_ID --live     # new linked paid attempt
scry memory assess resume                         # after fixing a blocked dispatch
```

`show` omits source text, extracted facts and the request packet unless
`--include-context` is explicit. Replay preview shows the retained redacted
packet and costs nothing; `--live` queues another charge. Expired or missing
evidence cannot be replayed. A provider refusal can block dispatch, and
`resume` validates the provider cooldown before restarting work.

The target input budget is approximately 20,000 tokens. The local byte-bound
counter intentionally underfills because it is not Jev's tokenizer. A packet
that cannot fit has a recorded truncation rather than an implicit larger call.
Historical episodes are imported from existing **derived summaries**, marked
source unavailable; they are not treated as original transcript evidence.

## API references

Verified against the public documentation on 2026-09-19:

- [TypeSafe HTTP contract](https://docs.typesafe.ai/api)
- [Models, pricing and input limits](https://docs.typesafe.ai/models)
- [Confidence semantics](https://docs.typesafe.ai/confidence)

The pinned model's documented limits are 64k tokens across the full request and
32k across state plus the longest question. The evaluator does not include
TypeSafe's tokenizer; the dataset byte limit is a local file bound, not a promise
that every case fits the provider's context window. The bundled cases are small.
