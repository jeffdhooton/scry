# Expanded evaluation protocol

This is a targeted challenge suite constructed after the first 14-case run,
not an untouched representative holdout. All inputs are fictional. The original
cases, original report, model (`jev-1.13.0`), rubric (`memory-assess-v1`) and 0.5
diagnostic threshold stay unchanged.

## Datasets

`expanded-v1.json` contains 64 new cases: eight in each group, arranged as four
pairs sharing evidence. Most pairs compare a supported memory with a nearby
unsupported claim. Durability pairs instead contrast two supported statements
with different usefulness across sessions.

| ID prefix | What it challenges |
| --- | --- |
| negation | Prohibitions, double negatives, disabled features, policy versus observed compliance |
| scope | Environment, actor, permission versus execution, explicit exceptions |
| plans | Requested or scheduled work and hypothetical options versus completion claims |
| attribution | Uncertainty, speaker identity, secondhand reports and fictional quotations |
| corrections | Retractions, later success, historical truth and unresolved conflicting accounts |
| durability | Standing preferences, constraints and lessons versus transient conversation |
| evidence | Partially supported conjunctions, numeric details, distant evidence and absent context |
| injection | Quoted commands, fake roles and instructions embedded in candidate facts |

`repeatability-v1.json` repeats the four baseline-disagreement inputs three times
each. The IDs differ, but the episode, fact and expected labels are copied
unchanged from `synthetic.json`. IDs and labels are not sent to Jev. These calls
check short-run output consistency, not generalization; provider caching or
stable inference may make identical responses uninformative about independence.
The original labels remain provisional, including known status/durability
ambiguities, so agreement on these anchors is not a definitive accuracy measure.

## Before inference

1. Review new labels against the frozen rubric without inspecting new model
   outputs. Fix ambiguous wording before freezing the dataset.
2. Use the real command's preview mode to validate both files and ensure expected
   labels stay outside model requests.
3. Record SHA-256 digests of both files and the rubric implementation before the
   first live request. Preserve input order and raw results.

The labels are authored and independently reviewed by agents, not independently
adjudicated by a human. Review cannot eliminate all ambiguity.

## Live run

One 64-request run and one 12-request repeatability run, sequential requests,
15-second per-request timeout, no retries or alternate models. Stop on any error
or provider refusal. Only these synthetic inputs are authorized for this run.

```sh
go run ./cmd/scry memory assess --file docs/memory-assess/expanded-v1.json --live
go run ./cmd/scry memory assess --file docs/memory-assess/repeatability-v1.json --live
```

Save reports separately. Do not automatically execute the second command if the
first fails. Credentials come from the process environment, never command-line
arguments or report files.

## Analysis

Report support, durability and assertion label agreement separately, including
support false positives/false negatives, each category's result, and each
incorrect case's raw probability. Recompute totals independently from per-case
results. Report latency and actual returned token usage with the recorded price
assumption. For repeatability, compare every new answer with its matching
baseline case and disclose unchanged versus varying outputs.

Do not change prompts, labels or thresholds in response to this run and then
present the same examples as fresh evaluation. Any later prompt or threshold
experiment needs versioning and a separate held-out set. None of these results
will admit, discard, invalidate or rewrite stored memories.
