# Integration evaluation — preview and local mock only

No new provider inference occurred. The 100-candidate corpus is synthetic; real reviewed examples: **0**. This report makes no model-quality or production-readiness claim.

All 600 rubric/arm/case combinations saved inspectable packets and completed with a deliberately scripted local mock. There were no terminal mock failures. All 600 packets remained under the conservative byte ceilings and underfilled their requested token target. Earlier source or derived context was selected in 400 packets; 20 selections explicitly marked missing raw evidence (Thyme summary). No record required budget truncation.

The relevant and 20k arms selected the same short useful histories. This corpus cannot establish the quality or latency benefit of actual 20k-token packets. It tests the configured budget and retrieval integration without manufacturing filler.

## Local integration measurements

| Measurement | Scripted mock value |
|---|---:|
| Completed | 600 |
| Local elapsed time | 496.72 ms |
| Local throughput | 1207.93 jobs/s |
| End-to-end p50 / p95 | 17.56 / 47.03 ms |
| Mock HTTP p50 / p95 | 0 / 0 ms (no HTTP) |
| New provider calls | 0 |
| Returned tokens / charged cost | Not applicable to mock |

Queue snapshots retain actual pending bytes, coverage gaps and oldest pending age before and after each arm. Mock probabilities, confusion matrices, FP/FN, Brier scores and individual disagreements only exercise reporting; they are not Jev quality measurements.

## Historical budget calibration

| Existing frozen v1 arm | Calls | Bytes/4 estimate ÷ returned input tokens | Byte-bound failures |
|---|---:|---:|---:|
| compact | 48 | 0.718 | 0 |
| long | 48 | 1.335 | 0 |
| organized | 48 | 0.719 | 0 |
| relevant | 48 | 0.759 | 0 |

All 192 reconstructed request hashes matched the existing frozen results. Bytes/4 underestimates returned usage in short arms and overestimates it in the repetitive long arm. Keep the conservative byte ceilings. This is approximate historical calibration, not an official tokenizer or v2 calibration. The historical archive is not representative retrieved evidence.

## Concrete live scope awaiting authorization

The preview contains **600 model requests**: 100 synthetic candidates × 3 context arms × 2 separately reported rubrics. Development is 420 requests; sealed final is 180 requests. Summed conservative UTF-8 dispatch bounds are 1,683,484 input tokens, corresponding to approximately **$0.0707** at the adapter’s pinned $0.042 per million input-token rate. This is a preflight budget bound, not returned usage or an invoice; no billing claim is made for the mock. Actual live cost must use returned input usage, and failures may have unreported charges.

Full corpus SHA256: `41306d47b5107ec2fee6c71e829a120819ff908dcf56f3c5f78046a7448fb5c1`. Selected all-cases SHA256: `0b6883aacc4155fda883a9efc2051dbbb3d245950b79635d4e20b0dce004ff7a`. Every packet hash and exact redacted request is in `preview.json`; artifact hashes are in `manifest.json`.

## Rollout boundary

Live evaluation is pending explicit authorization of the concrete packet preview. Independent agent label review is recorded in `REVIEW.md`; human review remains absent. The sealed set contains only six scenario families, so correlated claims limit inference. Before admission or retention enforcement, obtain independently reviewed real examples, acceptable FP/FN thresholds, representative history coverage and a rollback policy. Mock success alone does not justify that change. Recommend shadow-only observation; no enforcing mode.

Artifacts: [corpus](corpus.json), [review](REVIEW.md), [protocol](PROTOCOL.md), [exact packet preview](preview.json), [mock report](mock-report.json), [historical calibration](calibration.json), [hash manifest](manifest.json).
