# Context ablation, version 1

Freeze this experiment before inference. User asked whether cheap, abundant
context can overcome the first evaluation's errors. All text is fictional;
only the evaluator's input changes. The model, three questions, label meanings,
threshold and HTTP implementation remain frozen from the expanded evaluation.

## Arms

- `compact`: original transcript.
- `organized`: exactly that transcript wrapped as JSON text in a named field.
- `relevant`: the JSON text also includes additional attributed source statements
  making scope, attribution, timing or terminology more explicit.
- `long`: identical relevant packet plus 180 unrelated fictional project records,
  13,860 words. Relevant evidence is first in repetition 1 and last in
  repetition 2. These two calls test position, not identical-input repeatability.

The existing adapter accepts an episode string, so JSON is serialized **inside
state.episode**. This tests a textual evidence packet; it does not test native
nested JSON state or redesigned question instructions. No question is changed.
The organized arm is a minimal formatting control, not an elaborate schema.

There are 24 base cases: 10 previously tested anchors and 14 newly authored cases
in seven pairs. Every case runs twice per arm (192 calls), deterministically
shuffled across two 96-case batches. The fresh cases were written before new
outputs, but target known failure patterns; this is a development challenge set,
not a representative untouched holdout. Pair members and repeated calls are
correlated; 48 calls/arm do not mean 48 independent examples.

Additional source text is deliberately favorable and hand-authored. It often
restates the key evidence explicitly. It does not include expected labels, but
is an optimistic approximation to successfully retrieved context, not proof
that Scry can discover or verify this context in a real history. Some additions
explicitly describe milestone value or transience, so they provide interpretations
and durability cues as well as factual detail. Earlier cases
keep their original labels; additions must not reverse support or status.
Labels remain proposed judgments, with known ambiguity in retention usefulness.

## Reproduce and freeze

Run `python3 docs/memory-assess/context-experiment/build.py` from the repository.
Review sources/labels, then use `scry memory assess --file ...` without `--live`
to preview each batch. Freeze SHA-256 digests of build.py, sources.json, batches,
this protocol, analyze.py, synthetic.json, expanded-v1.json and the
classifier/evaluator before the first call. Do not modify
fixtures, prompts, labels or thresholds after inspecting results.

The manifest and exact generated batches are retained with results. The generator
is local and has no API client or credential handling. A live run uses the existing
Go CLI, sequentially, 15-second timeout, no retries; stop on any error or refusal.
No real memory or transcript is uploaded and no memory admission behavior changes.

## Analysis

Recompute correctness, support FP/FN, Brier scores, actual token usage, estimated
cost and nearest-rank latency from raw results. Split anchors from fresh cases,
show per-case probabilities and corrected/regressed decisions relative to the
compact arm. Compare first/last evidence positions in the long arm separately.
Position comparisons are descriptive: one call per case per position cannot
separate position effects from response variability.
Record missing/partial calls and do not replace errors with invented observations.
Paired calls may share provider caches; neither this nor synthetic distractor
performance establishes general accuracy. Dollar estimates use reported input
tokens at $0.042/M and free output; they are not invoices.
