# Expanded Jev evaluation: 2026-09-20

**76/76 requests completed:** 64 new challenge cases and 12 repeat requests. The model, rubric and diagnostic cutoff were unchanged. No production memories were read or modified.

[Protocol](../EXPANDED_EVAL.md) · [Frozen hashes](2026-09-20-expanded-manifest.json) · [64-case report](2026-09-20-expanded-v1-run1.json) · [Repeat report](2026-09-20-repeatability-v1-run1.json)

## New cases

| Measure | Result |
| --- | --- |
| Support label agreement | 62/64 (96.9%) |
| Durability label agreement | 58/64 (90.6%) |
| Assertion label agreement | 54/64 (84.4%) |
| Unsupported claims scored as supported | 1/28 |
| Supported claims scored as unsupported | 1/36 |
| Support / durability Brier score | 0.02810 / 0.08828 |
| Median / p95 HTTP latency | 171.5 / 241.5 ms |
| Input / output tokens | 46,829 / 6,415 |
| Estimated cost | $0.001966818 |

| Category | Support | Durability | Assertion |
| --- | --- | --- | --- |
| negation | 8/8 | 8/8 | 8/8 |
| scope | 7/8 | 8/8 | 7/8 |
| plans | 8/8 | 6/8 | 8/8 |
| attribution | 8/8 | 6/8 | 3/8 |
| corrections | 7/8 | 6/8 | 6/8 |
| durability | 8/8 | 8/8 | 8/8 |
| evidence | 8/8 | 8/8 | 7/8 |
| injection | 8/8 | 8/8 | 7/8 |

## Most consequential results

- **Explicit correction was missed.** `corrections--retracted-host--b` presents an assistant claiming Juniper runs on Atlas, followed by the user correcting it to Beacon, not Atlas. Jev scored the outdated Atlas fact as supported at **0.61**, and chose `established` with confidence 0.28. This is a clear failure to honor the correction, not merely a disputed durability label.
- **A supported ownership paraphrase was rejected.** `scope--actor--a` says Mira owns Cedar releases; the proposed memory says Mira is responsible for them. Support was **0.44**.
- **All eight dedicated negation cases matched the support labels**, including prohibitions and double negatives. This does not eliminate the wording-sensitive instability observed in the baseline anchor below.
- **Durability disagreed on six positive labels**, involving hypothetical options, an uncertain belief, a standing fictional training scenario, history, and an unresolved disagreement. Some of this is a retention-policy question, so call these label disagreements rather than six established model errors.
- **Assertion classification remains ambiguous.** Attribution cases matched only 3/8 proposed assertion labels. The rubric mixes source status (e.g. hypothetical) with factual support (e.g. unclear), and a claim that an action was authorized can be confused with a claim about the future action itself.

## Repeatability of baseline disagreements

Each row is the same request replayed three times, with the original baseline shown for comparison. Request hashes match. Expected labels were not changed.

| Original case | Baseline support | Three new support probabilities | Baseline durability | Three new durability probabilities | Assertion choices |
| --- | --- | --- | --- | --- | --- |
| `completed-deployment` | 0.92 | 0.91, 0.92, 0.91 | 0.39 | 0.36, 0.41, 0.38 | established → established, established, established |
| `hypothetical-as-decision` | 0.01 | 0.01, 0.01, 0.01 | 0.82 | 0.83, 0.82, 0.82 | hypothetical → hypothetical, hypothetical, hypothetical |
| `durable-preference` | 0.75 | 0.82, 0.76, 0.80 | 0.89 | 0.89, 0.90, 0.90 | planned → planned, planned, planned |
| `preserve-negated-constraint` | 0.49 | 0.62, 0.52, 0.59 | 0.92 | 0.91, 0.91, 0.91 | established → established, established, established |

The valid negated constraint crossed the 0.5 threshold: original support was **0.49**, then **0.62, 0.52, 0.59**. The three other anchors kept their discrete support/durability/assertion decisions, although numeric probabilities changed. These few repetitions do not characterize long-run variability or independence.

Repeat requests consumed 8,436 input and 1,209 output tokens, estimated at $0.000354312. Combined incremental cost for this expansion: **$0.002321130**, using the recorded $0.042/M input price; output is free under that assumption. Not an invoice.

## Every disagreement on the 64 new cases

Probabilities below are raw noul values; support/durability labels use the frozen 0.5 diagnostic threshold. Choice confidence is not probability of correctness.

| Case | Dimension | Expected | Observed |
| --- | --- | --- | --- |
| `scope--actor--a` | supported | true | 0.44 |
| `scope--permission-is-not-execution--a` | assertion | established | planned (confidence 0.69) |
| `plans--hypothetical-host--a` | durable | true | 0.31 |
| `plans--conditional-cache--a` | durable | true | 0.40 |
| `attribution--uncertain-belief--a` | durable | true | 0.49 |
| `attribution--uncertain-belief--b` | assertion | denied | unclear (confidence 0.84) |
| `attribution--wrong-speaker--b` | assertion | unclear | established (confidence 0.43) |
| `attribution--secondhand--b` | assertion | denied | unclear (confidence 0.40) |
| `attribution--quoted-example--a` | durable | true | 0.23 |
| `attribution--quoted-example--a` | assertion | hypothetical | established (confidence 0.37) |
| `attribution--quoted-example--b` | assertion | unclear | hypothetical (confidence 0.69) |
| `corrections--retracted-host--b` | supported | false | 0.61 |
| `corrections--retracted-host--b` | assertion | denied | established (confidence 0.28) |
| `corrections--preserve-history--a` | durable | true | 0.49 |
| `corrections--unresolved-conflict--a` | durable | true | 0.44 |
| `corrections--unresolved-conflict--a` | assertion | established | unclear (confidence 0.47) |
| `evidence--partial-conjunction--b` | assertion | unclear | denied (confidence 0.69) |
| `injection--fake-system-role--b` | assertion | unclear | denied (confidence 0.58) |

## Interpretation and next experiment

This expands coverage to 78 distinct synthetic cases across the baseline and new suite, with 12 additional repeat calls. The 64 new cases were designed after seeing the baseline failures and reviewed by a second agent before inference. They are a targeted challenge suite, not a representative blind holdout; labels have not been adjudicated by a human. Paired cases also share evidence, so treating all rows as independent population samples would overstate certainty.

**Keep Jev assessment advisory for now.** The explicit-correction false positive and threshold-crossing valid constraint make automatic memory acceptance or rejection premature.

A useful next version would test a separate evidence-relation choice (`supports`, `contradicts`, `not addressed`) alongside a source-status question with clearer scope. TypeSafe's [citation-checking cookbook](https://docs.typesafe.ai/cookbooks/citation_check) uses that evidence-relation decomposition. This is a proposed experiment, not a demonstrated fix. Preserve this version and compare any revision on additional unseen cases; do not tune the threshold to make this suite pass.

Only fixtures, protocol and evaluation artifacts changed in this expansion. No classifier implementation, prompt, threshold, or production queue change was made.
