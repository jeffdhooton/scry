# Jev integration acceptance audit

Date: 2026-09-20. Branch: `feat/jev-memory-integration`.

Implementation, local validation and the separately authorized live evaluation are complete. Existing uncommitted v1 foundation was preserved. No deploy or production enablement occurred. The [live report](integration-v1/LIVE_REPORT.md) records 592 completed assessments and eight typed failures from 600 attempts; every request hash matches the frozen preview.

| Spec deliverable | Evidence and outcome |
| --- | --- |
| Structured contract and preserved baseline | Typed `memory-assess-v2` state and `context-policy-v1`; reused strict HTTP transport. Tests cover malformed responses, model pin, distribution, limits, cancellation, redaction and safe errors. Historical calibration reconstructs all 192 frozen v1 request hashes without changes. |
| Source capture and deterministic context | Namespace and source-order metadata survive intake/extraction. Builder tests cover frozen cutoff, future exclusion, conflicts, overlap by speaker/span, unrelated scopes, missing raw evidence, derived self-corroboration exclusion, Unicode, injected instructions and oversized core. Retrieval is bounded to 4,096 recent records / 32MiB and manifests omissions. |
| Budget handling | Complete core and whole history records; independent conservative byte ceilings, replaceable counter and explicitly approximate target. All 600 frozen packets fit the ceilings. Short histories remain short; this corpus does not test actual 20k-token quality or latency. |
| Durable sidecar and worker | Store tests cover canonical deduplication, claims, restart, packet immutability, linked replay, source pinning, saturation, completion metadata reserve, expiry and read-only inspection. Worker tests cover independent concurrency, persist-before-dispatch, no retries, cancellation, terminal failures, durable refusal/cooldown and explicit resume. |
| Graph and ingestion independence | `TestAssessmentEndToEndDoesNotChangeGraphOrRecallWhileProviderWaits` compares off/shadow graph and recall, and completes extraction while assessment waits. Sidecar failure cannot requeue primary ingestion. Original candidates are copied before resolver mutation. Summary-only commits produce visible missing-source failures. |
| Configuration and operators | Assessment validation cannot disable extraction. Off captures nothing. Daemon RPCs provide status/list/show/replay/resume; CLI tests exercise configured remote socket and one-attempt mutations after response loss. Default inspection omits source bodies. Doctor and operator documentation cover daemon environment credentials, retention, backup and rollback. |
| Corpus and independent labels | Frozen 100 distinct synthetic claims across 20 target episodes; 70 development / 30 sealed candidates in separate scenario families. Independent agent review and adjudication occurred before inference; no human-reviewed or real examples are claimed. |
| Real integration evaluator | Actual builder/store/worker with six separately reported rubric/context arms; 600 inspectable previews and 600 successful scripted mock jobs. Metrics and individual disagreement reporting are tested. All seven artifact hashes match the frozen manifest. |
| Live quality and shadow baseline | Development then sealed ran unchanged against `jev-1.13.0`: 592/600 completed, eight terminal failures, no retries. Live reports include FP/FN, Brier, confusion matrices, all individual disagreements, coverage/gaps, queue snapshots, HTTP/end-to-end latency, throughput and daily returned tokens. Combined HTTP p95 232.76 ms, 617,509 returned input tokens, estimated input cost $0.025935378 excluding unknown failed-call charges. Synthetic-only limitations and recommendation against enforcement are explicit. |

Validation passed:

```sh
go test ./internal/memory/... ./internal/daemon ./internal/config ./cmd/scry ./cmd/jev-memory-eval ./internal/doctor
go test -race ./internal/memory/assess ./internal/memory/assessstore ./internal/memory/assesscontext ./internal/memory/assessworker ./internal/memory/assesseval
go test -race ./internal/daemon -run 'Assessment|Assess|Context|Worker|Store|MissingRaw|Committed|Shadow|SourceMetadata|PreExtracted'
go vet ./internal/config ./internal/memory/assess ./internal/memory/assessstore ./internal/memory/assesscontext ./internal/memory/assessworker ./internal/memory/assesseval ./internal/memory/distill ./internal/memory/queue ./internal/daemon ./internal/doctor ./cmd/scry ./cmd/jev-memory-eval
git diff --check
```

The preserved [preflight report](integration-v1/REPORT.md), [protocol](integration-v1/PROTOCOL.md), [exact packet preview](integration-v1/preview.json) and [hash manifest](integration-v1/manifest.json) define the approved action: 600 synthetic requests, 1,683,484 conservative input-bound tokens, approximately $0.0707 preflight input cost. The [live manifest](integration-v1/live-manifest.json) separately hashes the resulting reports. Existing historical memory remained outside this action.

The frozen development cases ran before the sealed cases without tuning from either run. Every attempted case has an inspectable assessment or typed failure, and all 600 packet hashes were verified. Neither the mock nor the synthetic live corpus alone justifies enforcement; durability false negatives remain substantial. Assessment remains off by default. A production shadow trial is a separate rollout; rollback from shadow is mode off plus daemon restart.
