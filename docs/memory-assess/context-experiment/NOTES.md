# What the context experiment changes

Richer context materially improved this synthetic evaluation. It is premature to
attribute the earlier failures to a fixed Jev capability limit: the input supplied
to the model matters, and the cost of trying much larger inputs is small.

All 192 requests completed with the original model, rubric and threshold.
There were 24 cases, four arms, two calls per case/arm. Each arm therefore has 48
observations, with repeated and paired examples rather than 48 independent cases.

| Input | Support agreement | Status agreement | Durability agreement | Median HTTP latency |
| --- | --- | --- | --- | --- |
| Original short input | 42/48 | 42/48 | 42/48 | 156 ms |
| Same input wrapped in JSON | 44/48 | 44/48 | 44/48 | 179 ms |
| Added relevant sources | 46/48 | 48/48 | 45/48 | 171 ms |
| Relevant sources plus long archive | 48/48 | 48/48 | 45/48 | 284 ms |

The long arm averaged 20,349 input tokens. Total reported usage across all arms
was 1,080,808 input tokens and 19,238 output tokens, approximately **$0.045394**
at [the documented $0.042/M input price](https://docs.typesafe.ai/models), with
free output. These are estimates from returned usage, not billing records.

The stale-host error improved from support probabilities 0.55/0.49 for compact
inputs to 0.07/0.08 with added relevant sources and 0.04/0.04 in long context.
The ownership paraphrase improved from 0.48/0.54 to 0.95/0.96 with relevant
sources. The new correction pair also improved: rejecting the obsolete public
export claim needed additional context, and accepting the corrected private
export claim crossed 0.5 only in the long arm.

Long context was not uniformly better in probability quality. Its durability
Brier score worsened to 0.1059 versus 0.0769 with relevant context, despite equal
45/48 label agreement. Some support probabilities fell substantially when the
evidence followed the archive (one from 0.97 to 0.56), although no support labels
crossed the diagnostic threshold. One observation per position cannot establish
a position effect independently of response variation.

## Implication for Scry

Test a generous evidence packet before redesigning or replacing the classifier.
Useful inputs include surrounding conversation, speaker attribution, ordered
corrections, entity identity and scope, and related source-backed project facts.
Keep proposed facts and unverified prior memories distinguishable from source
evidence, so adding context does not turn an old extraction error into authority.
The experiment supports spending tokens on context; it does not establish that
arbitrary padding is beneficial or that a 20k-token default is optimal.

This experiment did not implement retrieval or change memory admission. The
existing CLI already supports these larger serialized evidence packets. A next
production-oriented evaluation should obtain context through real retrieval,
preserve provenance, and shadow-score decisions against reviewed examples before
using these scores to discard or overwrite memories.

The biggest limit is that supplemental evidence was hand-authored to clarify
known error patterns. Some additions explicitly explain lasting usefulness.
The archive is synthetic and repetitive. This tests an optimistic supply of
useful context, not the quality of Scry retrieval over real histories. The 14 new
cases target known patterns and are not a representative held-out benchmark.

## Artifacts and reproduction

- [Frozen protocol](PROTOCOL.md) and [manifest](manifest.json).
- [Source cases and added evidence](sources.json); [generator](build.py).
- Exact [batch 1](batch-1.json) and [batch 2](batch-2.json) input files.
- Raw [batch 1 results](results-batch-1.json) and [batch 2 results](results-batch-2.json).
- [Full metrics and disagreements](RESULTS.md); [machine-readable summary](summary.json).

From the repository root, recompute the saved results without network access:

```sh
python3 docs/memory-assess/context-experiment/analyze.py
```

Preview either dataset using `go run ./cmd/scry memory assess --file` followed
by its path. To repeat paid inference, append `--live` with `TYPESAFE_API_KEY` in
the environment and save to a **new** report path. Preserve the frozen reports.
Run batches sequentially and stop on any error; do not automatically retry.

Verification: existing assessor and CLI Go tests passed; both preview payloads
parsed; an independent reviewer checked label preservation, packet invariants
and analysis logic; the analysis independently recomputed metrics from all raw
results and verified every frozen input hash. No production code was changed
for this context experiment.
