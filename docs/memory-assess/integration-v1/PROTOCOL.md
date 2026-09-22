# Integration evaluation protocol v1

This directory is a synthetic-only, pre-live evaluation package. Existing baseline, expanded and context-experiment results are unchanged. The selected corpus and final packets are frozen by SHA256 in `manifest.json`. Labels were authored, independently reviewed by a separate agent and adjudicated before provider inference; see `REVIEW.md`. There has been no human corpus review and no new Jev inference.

The corpus contains 100 unique candidate claims in 20 distinct target episodes, each with one separate earlier history record. Fourteen scenario families (70 candidates) are development; six different scenario families (30 candidates) are sealed. Related claims and histories stay in one split. Each scenario has its own repository scope, preventing accidental cross-split retrieval. Labels use the full eligible evidence at target time, so compact-arm missing evidence can create reference false negatives. Synthetic historical dates are scenario data; `labels_recorded_at` is the actual UTC authoring/adjudication time.

Three arms use the same candidates:

1. `compact`: the production builder sees the entire target and candidate; the reader exposes no earlier history.
2. `relevant`: production retrieval with a 2,500 approximate-token target.
3. `target20k`: the same retrieval with a 20,000 approximate-token target.

Every arm uses fresh temporary `assessstore` storage, source capture, durable enqueue, the real `assesscontext.Builder`, `assessworker.Worker`, packet persistence before dispatch, result persistence and job inspection. No daemon socket or private memory store is read. Temporary sidecars are removed after export. The compact reader changes available history only. Both richer arms are intentionally short and select identical evidence in this corpus. They test configuration and packet behavior, not the effect of 20k actual input. No archive filler is added.

Rubrics are compared separately: each arm runs both frozen `memory-assess-v1` and native `memory-assess-v2`. The v1 adapter serializes the exact builder-selected target/history/derived/scope fields into the v1 episode string and passes the candidate through the separate v1 fact field. It excludes the candidate from that episode string, preventing self-corroboration. It recomputes the conservative byte ceilings for the final v1 packet. Model pin remains `jev-1.13.0`. Operational case IDs, labels and extractor confidence stay outside model state; source identifiers are opaque hashes. Request hashes cover exact saved bytes.

The default mode uses a no-network dispatcher to exercise packet persistence. Workers finish those preview jobs as failed locally; the exported report identifies successfully saved packets as `previewed`. This is deliberate preview bookkeeping, not a provider failure. `--mock` uses deterministic request-hash probabilities and a valid scripted choice distribution independent of labels. Mock throughput and end-to-end timings measure local harness execution only. Mock HTTP latency is zero and carries no provider performance meaning. Neither preview nor mock exports returned usage or cost fields.

Metrics are grouped by rubric, arm and split. They include support/durability FP/FN and Brier scores at diagnostic 0.5, assertion confusion, every individual disagreement (including retention/durability), retrieval selections/exclusions, source gaps, truncations, underfill, terminal outcomes/errors, nearest-rank HTTP and end-to-end p50/p95, observed queue age and pending bytes, throughput and, only for live runs, daily returned usage and estimated input cost. Full corpus SHA256 and selected-case SHA256 are distinct. Failures without returned usage may still have incurred charges; successful-return cost is not a billing total. Explicit provider refusal stops the entire run across all remaining arms; no automatic retry or fallback occurs.

Byte counting is a conservative UTF-8 bound, not official tokenization. `calibration.json` reconstructs 192 existing frozen v1 requests and verifies every request hash against the old result before comparing bytes/4 to returned usage. The ratio varies substantially by arm; these observations do not establish a universal tokenizer conversion or validate v2 token estimates. The separate state-plus-longest-question bound cannot be validated from aggregate usage alone.

Commands from repository root:

```sh
# Safe default: exports development packets, never calls a provider.
go run ./cmd/jev-memory-eval --out /tmp/development-preview.json
# Inspect all proposed packets, including sealed final set.
go run ./cmd/jev-memory-eval --split all --out /tmp/all-preview.json
# Local integration run; no quality claim.
go run ./cmd/jev-memory-eval --mock --split all --omit-packets --out /tmp/mock-report.json
# Reconstruct already-frozen usage; no new calls.
go run ./cmd/jev-memory-eval --calibrate-from docs/memory-assess/context-experiment --out /tmp/calibration.json
```

Every output path must be new; the CLI refuses overwrites. `--live` is the explicit network boundary and requires `TYPESAFE_API_KEY` in the process environment. A future authorized development run uses `--live --split development`; the sealed final run uses `--live --split sealed` only after all choices are frozen. Do not tune labels or settings from sealed live results. Authorization must refer to the concrete packet preview and selected corpus; existing synthetic experiments do not authorize reading or uploading personal historical memory. No live run was performed for this package.

The parent integration work separately covers daemon intake, extractor mock, off/shadow graph and recall equality, blocked-provider independence and daemon inspection. This standalone runner is not a substitute for that end-to-end acceptance test.
