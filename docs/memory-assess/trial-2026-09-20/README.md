# Bounded Jev shadow trial — September 20, 2026

User authorized diagnostic hardening followed by a small trial during normal
agent work. Both laptop and Mini now run the same tested binary:
`scry-6746cbd-jev-trial-20260920`, SHA256
`72b6afbd251608b8826143005738ee7b788e2f9b8c83811bfcd91108969480fc`.
This is a build of the preserved uncommitted integration branch, not a pushed or
merged release. The Mini runs assessment; the laptop continues routing shared
memory to it.

The trial starts **2026-09-20 13:12:56 UTC** and expires **2026-09-21 13:12:56 UTC**
(9:12:56 a.m. Eastern on both dates). Maximum **100 lifetime dispatch attempts**,
concurrency **1**, model **jev-1.13.0**, timeout **15 seconds**. The default target
remains approximately 20k input tokens with conservative byte ceilings. At the
recorded $0.042/million input rate, 100 requests each at the full 60k byte-bound
ceiling correspond to $0.252; actual returned usage is reported separately.

Only new source episodes within the window are eligible. Unknown source times,
the existing 632 parked extraction items, earlier context and old retained replay
evidence are excluded. No historical graph-summary import or backfill runs.
Assessment remains observational: it cannot change admission, resolution,
deletion or recall ranking. The request cap is durable and includes failures and
uncertain deliveries; neither restart nor `resume` can bypass cap or expiry.

## Hardening and verification

- Client failures now carry allowlisted categories through to persisted jobs and
  daemon `error_counts`; arbitrary transport/provider text is discarded.
- Timeout is bounded to 15 seconds. Retry-After arithmetic saturates safely
  rather than overflowing into an early resume.
- Dispatch reservations and exact packets are synchronously persisted before
  network calls. Concurrent reservations, restart, cleanup and replay obey the
  lifetime cap. Automatic retries remain disabled.
- Review exposed and tests reproduced three issues, all fixed before deployment:
  missing timestamps normalized by intake, pretrial context in existing sidecars,
  and future starts requiring manual resume.
- Full memory/daemon/config/CLI/doctor tests, focused store/context/worker/config
  races, daemon assessment races and changed-package vet passed. Independent
  review cleared all findings.
- Initial `scry doctor` reported zero failed checks, with extraction running and
  shadow assessment active. Existing parked-work and local environment warnings
  were retained; they are not assessment failures.

## Inspect and stop

```sh
scry memory assess status --json
scry memory assess list --limit 100 --json
scry memory assess show ASSESSMENT_ID --json
```

Use `--include-context` only when deliberately inspecting retained local source
text. At cap or expiry, `blocked_reason` becomes `trial_request_limit` or
`trial_expired`. Existing graph ingestion continues. To stop early, set the
Mini's `memory.assessment.mode` to `off` and gracefully restart its daemon.

The [setup record](setup.json) contains exact limits and rollback paths. The
[initial assessment status](initial-status.json) and
[initial memory status](initial-memory-status.json) confirm configuration and
primary-worker health. A durable setup note was queued through the normal
extractor as the first trial observation; its episode ID is
`9ef4ec9913213f780695a3b0aad80f594d83646485e00e8c34b248a1b5e792b6`.

The [first observation](first-observation.json) finished with **15 completed,
zero failed** assessments. The [verified assessment snapshot](verified-status.json)
records 15/100 dispatches, 16,789 input tokens, approximately $0.000705 input cost,
HTTP p50/p95 177.515/385.577 ms, no source gaps or truncations, and an empty queue.
One retained request was inspected and its exact bytes verified against its
SHA256; source text was not exported into this repository. The
[verified memory snapshot](verified-memory-status.json) confirms extraction
completed, the worker remained running, and the pre-existing parked count stayed
632. This is an operational smoke check on the setup note, not a representative
quality evaluation. Normal agent work supplies the remaining trial samples.

## Rollback material

- Verified nonempty live graph backup:
  `/Users/jclaw/.scry/backups/memory-20260920T131102Z.badger` (144,403,617 bytes).
- Mini prior binary: `/Users/jclaw/.local/bin/scry.pre-jev-trial-20260920`.
- Laptop prior binary: `/Users/jeff/go/bin/scry.pre-jev-trial-20260920`.
- Mini prior config: `/Users/jclaw/.scry/config.yaml.pre-jev-trial-20260920`.
- Mini credential-environment backup remains on the Mini with mode 0600; no
  credentials are in this repository or YAML. The daemon loads its key through
  its existing environment-file startup path.

Restore the prior binaries/config only if a full binary rollback is needed;
ordinary trial shutdown needs only mode off and restart. Keep the assessment
sidecar for inspection. Normal shutdown requires no graph repair or restore.
