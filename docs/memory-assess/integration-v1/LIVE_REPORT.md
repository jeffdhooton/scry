# Live integration evaluation — 2026-09-20

User authorized the frozen synthetic evaluation after reviewing its scope and cost. Development ran first, followed by the sealed set, with no intervening rubric, label, packet, or configuration changes. No retries, production daemon restart, private-memory upload, or enforcement occurred.

**592/600 requests completed; 8 failed (1.33%).** All 600 exact request hashes match the frozen preview. All validated responses report `jev-1.13.0`. Failures remain in the denominators for completion reporting and are excluded from quality metrics; no failure is counted as a correct judgment.

## Runtime and cost

| Measure | Development | Sealed | Combined |
| --- | ---: | ---: | ---: |
| Completed | 415/420 | 177/180 | 592/600 |
| Input tokens | 430,770 | 186,739 | 617,509 |
| Output tokens | 41,584 | 17,735 | 59,319 |
| Estimated input cost | 0.01809234 | 0.00784304 | 0.02593538 |
| HTTP p50, ms | 173.65 | 166.89 | 169.69 |
| HTTP p95, ms | 232.76 | 239.78 | 232.76 |
| End-to-end p50, ms | 3251.76 | 1485.35 | 2362.39 |
| End-to-end p95, ms | 5996.25 | 2677.54 | 5858.51 |
| Completed requests/s | 10.92 | 10.56 | 10.81 |

The two runs took 54.76 seconds of combined execution at concurrency 2. End-to-end time includes waiting behind a burst of 70 or 30 jobs per arm; HTTP time measures validated completed requests only. HTTP p95 met the sub-second engineering target for these short synthetic packets. This is not a representative production latency guarantee.

Cost uses returned input tokens at the recorded $0.042/million rate. Output tokens are reported for transparency; the adapter uses input-only pricing. Returned usage totals 617,509 input tokens on 2026-09-20. The eight failed calls may still have incurred unreported charges, so $0.025935378 is not an invoice total.

## Quality by split

Binary judgments use the diagnostic 0.5 cutoff only. FP/FN are counts, Brier is mean squared probability error, and assertion is correct/completed. Reference labels were frozen before inference. Compact-arm judgments are still compared with the full eligible-evidence labels; this intentionally measures the cost of missing history.

| Split | Rubric | Context | Completed | Support FP/FN | Support Brier | Durable FP/FN | Durable Brier | Assertion |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| development | v1 | compact | 69 | 0/18 | 0.2358 | 0/5 | 0.0947 | 50/69 |
| development | v1 | relevant | 70 | 0/17 | 0.1506 | 0/8 | 0.1036 | 60/70 |
| development | v1 | target20k | 67 | 0/15 | 0.1413 | 0/7 | 0.1033 | 56/67 |
| development | v2 | compact | 69 | 0/19 | 0.1903 | 0/15 | 0.1645 | 53/69 |
| development | v2 | relevant | 70 | 0/8 | 0.0911 | 0/20 | 0.1711 | 65/70 |
| development | v2 | target20k | 70 | 0/8 | 0.0936 | 0/23 | 0.1765 | 64/70 |
| sealed | v1 | compact | 29 | 0/6 | 0.1703 | 0/3 | 0.0973 | 22/29 |
| sealed | v1 | relevant | 30 | 0/5 | 0.1356 | 0/3 | 0.1046 | 28/30 |
| sealed | v1 | target20k | 30 | 0/5 | 0.1404 | 0/3 | 0.1076 | 28/30 |
| sealed | v2 | compact | 30 | 0/7 | 0.1699 | 0/8 | 0.1620 | 25/30 |
| sealed | v2 | relevant | 29 | 0/3 | 0.0925 | 0/7 | 0.1702 | 28/29 |
| sealed | v2 | target20k | 29 | 0/2 | 0.0882 | 0/6 | 0.1620 | 27/29 |

Full assertion confusion matrices for every split/arm/rubric and combined results are in [live-summary.json](live-summary.json). Every individual disagreement, including durability, is listed with the claim, expected label, probability/choice and packet hash in [live-disagreements.csv](live-disagreements.csv). Raw reports retain validated distributions and manifests.

## Interpretation

- V2 with relevant retrieval had 11 support false negatives among 99 completed cases, versus 26 among 99 for compact V2. Assertion agreement was 93/99 versus 78/99. Sealed V2 relevant results were 26/29 support agreement and 28/29 assertion agreement. Different failed cases and correlated claims prevent treating these as a clean paired significance test.
- Durability remains unsuitable for enforcement: V2 relevant had 27 false negatives among 99 completed cases (7/29 sealed), compared with 11/100 for V1 relevant. V2 often assigns low durability to denied or uncertain claims despite the instruction to assume accuracy. That is an observed response pattern, not a proved explanation of model reasoning.
- Examples from sealed V2 relevant: the planned Olive clock-change test received support 0.46 and durability 0.48; the Spruce historical NFC normalization statement received support 0.46; the planned Thyme airplane-mode test received support 0.24. A failed Reed delivery was labeled denied where the frozen label was unclear. Labels were not rewritten to fit these outputs.
- No support false positives occurred in completed samples, but these are only 100 synthetic candidates across 20 correlated scenario families, with six sealed families. This does not establish a safe production false-positive rate. Human-reviewed examples and real examples both remain zero.
- Relevant and target20k selected the same short useful histories. Any differences between those arms reflect call variability or missing responses, not evidence that longer context improved performance. All 600 packets were underfilled; no claim about actual 20k-token quality or performance follows.

## Coverage, queues and failures

All 600 packets were persisted before dispatch and remained inspectable in the exports. Earlier raw or derived evidence was selected in 400 packets; 20 selections recorded missing raw evidence from derived-only Thyme history. There were zero budget truncations. Queue snapshots in each raw report retain actual initial/final job counts, pending bytes, metadata/payload bytes, coverage gaps and oldest pending age. Each arm drained to terminal outcomes; no automatic retry or hidden dropped sample occurred.

All eight failures use the existing safe `provider_request_failed` category. This category combines transport, response-read and response-validation errors; retained data cannot establish which subtype occurred. They are not documented HTTP 401/403/429 refusals. No raw response body or secret was retained. More granular safe error codes would improve later diagnosis without changing this frozen run.

| Split | Case | Rubric | Context | Failure |
| --- | --- | --- | --- | --- |
| development | case-14-4 | memory-assess-v1 | compact | provider_request_failed |
| development | case-07-4 | memory-assess-v1 | target20k | provider_request_failed |
| development | case-08-4 | memory-assess-v1 | target20k | provider_request_failed |
| development | case-12-3 | memory-assess-v1 | target20k | provider_request_failed |
| development | case-05-4 | memory-assess-v2 | compact | provider_request_failed |
| sealed | case-19-1 | memory-assess-v1 | compact | provider_request_failed |
| sealed | case-15-2 | memory-assess-v2 | relevant | provider_request_failed |
| sealed | case-18-4 | memory-assess-v2 | target20k | provider_request_failed |

## Recommendation and provenance

The integration is suitable for a separately enabled shadow observation trial. Keep assessment off by default until that rollout is requested. Do not use these judgments to admit, delete, merge, supersede or rank memories. Collect reviewed real examples and diagnose failure categories before proposing any enforcing change; durability disagreement alone rules out automatic retention decisions from this run.

The original [preflight report](REPORT.md), corpus, review, preview and manifest remain unchanged. Its “pending” language describes the preserved pre-run state. The raw live reports retain a generic `quality_claim` string saying labels await independent review; [REVIEW.md](REVIEW.md) records the independent agent review actually performed before inference. No human review is claimed.

Artifacts: [development](development-live.json), [sealed](sealed-live.json), [summary](live-summary.json), [individual disagreements](live-disagreements.csv), [live hashes](live-manifest.json).
