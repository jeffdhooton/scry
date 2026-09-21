# Context ablation results — 2026-09-20

All 192 calls completed. Pinned Jev 1.13.0, unchanged memory-assess-v1 rubric, 0.5 diagnostic threshold. All cases and supplemental sources are fictional. Labels were frozen before inference and not revised afterward.

## All calls

| Context | Support | FP / FN | Durable | Assertion | Input tokens/call | p50 / p95 ms | Cost USD |
| --- | --- | --- | --- | --- | --- | --- | --- |
| compact | 42/48 | 3 / 3 | 42/48 | 42/48 | 700 | 156.2 / 323.6 | $0.001411 |
| organized | 44/48 | 2 / 2 | 44/48 | 44/48 | 707 | 179.0 / 283.3 | $0.001426 |
| relevant | 46/48 | 0 / 2 | 45/48 | 48/48 | 761 | 171.3 / 264.9 | $0.001534 |
| long | 48/48 | 0 / 0 | 45/48 | 48/48 | 20,349 | 283.7 / 471.7 | $0.041024 |

## Anchor cases

| Context | Calls | Support | FP / FN | Durable | Assertion | Support Brier |
| --- | --- | --- | --- | --- | --- | --- |
| compact | 20 | 18 | 1 / 1 | 16 | 15 | 0.0843 |
| organized | 20 | 20 | 0 / 0 | 18 | 16 | 0.0426 |
| relevant | 20 | 20 | 0 / 0 | 19 | 20 | 0.0104 |
| long | 20 | 20 | 0 / 0 | 19 | 20 | 0.0150 |

## Fresh cases

| Context | Calls | Support | FP / FN | Durable | Assertion | Support Brier |
| --- | --- | --- | --- | --- | --- | --- |
| compact | 28 | 24 | 2 / 2 | 26 | 27 | 0.1057 |
| organized | 28 | 24 | 2 / 2 | 26 | 28 | 0.0798 |
| relevant | 28 | 26 | 0 / 2 | 26 | 28 | 0.0551 |
| long | 28 | 28 | 0 / 0 | 26 | 28 | 0.0395 |

## Changes relative to compact

A fix/regression is counted per paired case/repetition, separately for each judgment. These are correlated observations, not independent trial counts.

| Context | Support fixes / regressions | Durability fixes / regressions | Assertion fixes / regressions |
| --- | --- | --- | --- |
| organized | 2 / 0 | 2 / 0 | 2 / 0 |
| relevant | 4 / 0 | 3 / 0 | 6 / 0 |
| long | 6 / 0 | 3 / 0 | 6 / 0 |

## Long-context evidence position

| Position | Support | Durable | Assertion |
| --- | --- | --- | --- |
| first | 24/24 | 23/24 | 24/24 |
| last | 24/24 | 22/24 | 24/24 |

## Per-case support probabilities

Each cell lists r1 / r2. Expected labels remain fixed across all arms.

| Case | Expected | Compact | Organized | Relevant | Long |
| --- | --- | --- | --- | --- | --- |
| corrections--retracted-host--a | True | 0.86 / 0.85 | 0.87 / 0.87 | 0.90 / 0.90 | 0.93 / 0.90 |
| corrections--retracted-host--b | False | 0.55 / 0.49 | 0.16 / 0.21 | 0.07 / 0.08 | 0.04 / 0.04 |
| scope--actor--a | True | 0.48 / 0.54 | 0.72 / 0.75 | 0.95 / 0.96 | 0.95 / 0.77 |
| scope--actor--b | False | 0.02 / 0.02 | 0.01 / 0.01 | 0.01 / 0.01 | 0.01 / 0.02 |
| durable-preference | True | 0.78 / 0.79 | 0.83 / 0.82 | 0.88 / 0.90 | 0.93 / 0.90 |
| completed-deployment | True | 0.91 / 0.90 | 0.93 / 0.93 | 0.90 / 0.87 | 0.93 / 0.76 |
| preserve-negated-constraint | True | 0.53 / 0.51 | 0.55 / 0.55 | 0.88 / 0.89 | 0.93 / 0.70 |
| transient-progress | True | 0.87 / 0.86 | 0.74 / 0.76 | 0.79 / 0.79 | 0.83 / 0.88 |
| attribution--uncertain-belief--a | True | 0.93 / 0.93 | 0.95 / 0.95 | 0.94 / 0.94 | 0.95 / 0.91 |
| attribution--uncertain-belief--b | False | 0.02 / 0.02 | 0.01 / 0.01 | 0.01 / 0.01 | 0.01 / 0.01 |
| fresh--environment--a | True | 0.96 / 0.96 | 0.97 / 0.96 | 0.96 / 0.96 | 0.97 / 0.56 |
| fresh--environment--b | False | 0.01 / 0.01 | 0.01 / 0.01 | 0.01 / 0.01 | 0.01 / 0.01 |
| fresh--rollout--a | True | 0.94 / 0.94 | 0.96 / 0.96 | 0.94 / 0.94 | 0.94 / 0.91 |
| fresh--rollout--b | False | 0.01 / 0.02 | 0.01 / 0.01 | 0.01 / 0.01 | 0.01 / 0.01 |
| fresh--permission--a | True | 0.89 / 0.89 | 0.90 / 0.92 | 0.88 / 0.90 | 0.91 / 0.74 |
| fresh--permission--b | False | 0.01 / 0.01 | 0.01 / 0.01 | 0.01 / 0.01 | 0.01 / 0.01 |
| fresh--correction--a | True | 0.19 / 0.34 | 0.38 / 0.48 | 0.45 / 0.47 | 0.69 / 0.69 |
| fresh--correction--b | False | 0.83 / 0.81 | 0.67 / 0.72 | 0.43 / 0.46 | 0.24 / 0.18 |
| fresh--vendor--a | True | 0.87 / 0.87 | 0.82 / 0.87 | 0.95 / 0.92 | 0.97 / 0.92 |
| fresh--vendor--b | False | 0.02 / 0.02 | 0.02 / 0.02 | 0.01 / 0.01 | 0.01 / 0.01 |
| fresh--absent--a | True | 0.97 / 0.96 | 0.95 / 0.94 | 0.94 / 0.92 | 0.95 / 0.76 |
| fresh--absent--b | False | 0.02 / 0.02 | 0.02 / 0.02 | 0.02 / 0.01 | 0.01 / 0.01 |
| fresh--retention--a | True | 0.57 / 0.63 | 0.55 / 0.57 | 0.52 / 0.51 | 0.67 / 0.51 |
| fresh--retention--b | True | 0.82 / 0.69 | 0.69 / 0.78 | 0.87 / 0.86 | 0.91 / 0.66 |

## Remaining disagreements

| Context | Case / repetition | Dimension | Expected | Observed |
| --- | --- | --- | --- | --- |
| compact | attribution--uncertain-belief--a / r1 | durable | True | 0.49 |
| compact | attribution--uncertain-belief--a / r2 | durable | True | 0.49 |
| compact | attribution--uncertain-belief--b / r1 | assertion | denied | unclear |
| compact | attribution--uncertain-belief--b / r2 | assertion | denied | unclear |
| compact | completed-deployment / r1 | durable | True | 0.4 |
| compact | completed-deployment / r2 | durable | True | 0.36 |
| compact | corrections--retracted-host--b / r1 | supported | False | 0.55 |
| compact | corrections--retracted-host--b / r2 | assertion | denied | established |
| compact | durable-preference / r1 | assertion | established | planned |
| compact | durable-preference / r2 | assertion | established | planned |
| compact | fresh--correction--a / r1 | supported | True | 0.19 |
| compact | fresh--correction--a / r2 | supported | True | 0.34 |
| compact | fresh--correction--b / r1 | supported | False | 0.83 |
| compact | fresh--correction--b / r2 | supported | False | 0.81 |
| compact | fresh--permission--b / r1 | durable | True | 0.4 |
| compact | fresh--permission--b / r2 | durable | True | 0.45 |
| compact | fresh--vendor--b / r2 | assertion | denied | unclear |
| compact | scope--actor--a / r1 | supported | True | 0.48 |
| long | attribution--uncertain-belief--a / r2 | durable | True | 0.45 |
| long | fresh--permission--b / r1 | durable | True | 0.23 |
| long | fresh--permission--b / r2 | durable | True | 0.2 |
| organized | attribution--uncertain-belief--b / r1 | assertion | denied | unclear |
| organized | attribution--uncertain-belief--b / r2 | assertion | denied | unclear |
| organized | completed-deployment / r1 | durable | True | 0.4 |
| organized | completed-deployment / r2 | durable | True | 0.38 |
| organized | durable-preference / r1 | assertion | established | planned |
| organized | durable-preference / r2 | assertion | established | planned |
| organized | fresh--correction--a / r1 | supported | True | 0.38 |
| organized | fresh--correction--a / r2 | supported | True | 0.48 |
| organized | fresh--correction--b / r1 | supported | False | 0.67 |
| organized | fresh--correction--b / r2 | supported | False | 0.72 |
| organized | fresh--permission--b / r1 | durable | True | 0.42 |
| organized | fresh--permission--b / r2 | durable | True | 0.39 |
| relevant | attribution--uncertain-belief--a / r1 | durable | True | 0.49 |
| relevant | fresh--correction--a / r1 | supported | True | 0.45 |
| relevant | fresh--correction--a / r2 | supported | True | 0.47 |
| relevant | fresh--permission--b / r1 | durable | True | 0.35 |
| relevant | fresh--permission--b / r2 | durable | True | 0.36 |

Total: 1,080,808 input tokens; 19,238 output tokens; estimated $0.045394. Cost uses returned usage and $0.042/M input, free output.

## Limits

This is a small, targeted synthetic development experiment. Repetitions and paired facts share evidence; aggregate call counts overstate the number of independent examples. Supplemental statements are hand-authored clarifications, sometimes restating the decisive fact. This tests what happens when useful context is available, not whether retrieval can find it accurately. Sources are serialized JSON text inside state.episode, not native nested state. Synthetic archive records are repetitive and easier to separate than real, conflicting project history. Position differences are confounded with per-call variation. Fresh cases target familiar error patterns and are not a representative holdout. No production admission or retention rule changed. See [protocol](PROTOCOL.md) and manifest for reproducibility.
