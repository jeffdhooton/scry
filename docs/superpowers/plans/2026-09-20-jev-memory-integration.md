# Jev memory integration implementation

Spec: `docs/superpowers/specs/2026-09-20-jev-memory-integration-design.md`.
Branch: `feat/jev-memory-integration`. Existing uncommitted evaluation work is preserved in place.

## Global constraints

Shadow assessments never control graph writes. Off means no capture/dispatch. Keep v1 fixture/request contracts. All provider traffic is explicit; no live corpus upload until preview and authorization. Use only daemon environment TYPESAFE_API_KEY. No deploy or graph backfill.

## Task 1: Structured request and transport

Own `internal/memory/assess/{context,rubric_v2}.go`, existing client transport and focused tests. Add typed v2 state, provenance and replaceable byte-bound budget accounting. Preserve v1 bytes. Reuse transport with exact packet dispatch, supported model validation, safe typed HTTP errors and Retry-After. Test strict responses, Unicode budgets, redaction, no fixture labels/confidence, redirects and cancellation. Test first, run package tests, report public interfaces.

## Task 2: Durable assessment sidecar

Own `internal/memory/assessstore/`. Separate Badger schema stores immutable redacted source revisions, original candidate occurrences, job claims and exact packets/results. Deduplicate canonical content, freeze source cutoff, bound pending jobs/payloads, enforce retention with injected clock, pin pending evidence, recover uncertain running jobs as failed, explicit linked replay and persistent block/resume. Add behavioral tests for restart, saturation, expiry, unavailable evidence and redaction.

## Task 3: Retrieval and packing

Own `internal/memory/assesscontext/`. Deterministic source-backed selection with complete target, chronological history, source namespace/order/span constraints, attested repository identity, relevant conflicts and derived context. Exclude future/beyond-cutoff/self-derived records. Prefer whole source records; never clip core. Test all context acceptance cases in the spec.

## Task 4: Worker and daemon wiring

Own `internal/memory/assessworker/`, config, queue hook, daemon lifecycle/intake/RPCs and minimal distillation metadata. Two bounded independent workers, persist before dispatch, no retries, safe blocked suspension/cooldowns and shutdown. Hook original extraction before resolver; sidecar failures cannot affect ingestion. Validate assessment config independently of extraction. Behavioral tests prove off/shadow graph and recall equality and extraction independence.

## Task 5: Operator interface

Own CLI assessment subcommands, doctor, operator docs. Preserve file-preview invocation. Route status/list/show/replay/resume over configured memory socket; bounded pagination, explicit context, replay preview default, missing key boolean and actionable message. Test remote routing and safe output.

## Task 6: Integration corpus, evaluation and audit

Version at least 100 distinct candidates across at least 20 episodes, with >=30 candidates in a sealed episode-separated set, labels recorded before inference. Use actual builder and worker; compact/retrieved/20k arms and separate v1/v2 comparisons. Export previews before authorized live calls. Report individual disagreements, all quality/coverage/latency/throughput/cost metrics and limitations. Preserve frozen prior experiments. Review each component and whole integration. Run spec-required Go suite, focused races, vet; audit every acceptance requirement before goal completion.

## Progress

- Tasks 1–6 complete. The user explicitly authorized live evaluation after preview: 600 frozen synthetic requests, 592 completed and eight typed failures, no retries. Development preceded sealed without tuning. Results and remaining quality limitations are in `docs/memory-assess/integration-v1/LIVE_REPORT.md`.
- TypeSafe live API and cookbook read; Context7 consulted. Direct HTTP docs and validated existing responses govern native state and noul/input_tokens wire fields.
- Required Go suite, full focused package races, daemon assessment races, changed-package vet and diff checks pass. All seven frozen evaluation artifact hashes verified.
- Acceptance evidence and live measurements: `docs/memory-assess/INTEGRATION_AUDIT.md`. No deployment or production enablement performed.
- Ruling: keep existing checkout on dedicated integration branch, preserving untracked baseline; no new worktree or reset needed.
