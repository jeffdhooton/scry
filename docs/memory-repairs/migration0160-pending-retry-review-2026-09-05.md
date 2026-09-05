# Exact pending-item retry gate: PASS, conditional on immediate fresh checks

This review covers **one normal-pipeline retry invocation only** for:

`ed50810befba04287fe6676b18db368af6808d32508980ae6e49373009230727`

No retry was executed by this reviewer, in the live store or either restored replica. The five other parked IDs are excluded. This is not permission to retry the entire queue, force-reapply completed episodes, edit facts, change timestamps, nudge keys, or repair aliases implicitly.

The recorded canonical-name routing conflict has a reviewed correction in deployed code `62cf6e0d2d8db9d324da23114df837848e972c8f`. The actual provider's intermediate output remains unknown. The gate permits a single administrative resumption through the ordinary extraction/resolution pipeline; it does not guarantee that the next provider output will succeed or match any previous synthetic probe.

## Independent snapshot evidence

Freshly restored source:

`/tmp/scry-canonical-62cf-deploy-sep05.G82GKc/mini-after-deploy.badger`

Mini source backup: `memory-20260905T205021Z.badger`, supplied size 77,959,020 bytes. Independently verified SHA-256: `a6eea06bf04ab409d37a7a2aeb24ac94c5e259148336300f149e5098384d497e`.

Independent replica: `/tmp/scry-pending-retry-gate-sep05.uZpjww/replica-post-deploy`.

It contains 80,268 facts, 30,460 entities, 9,327 completed episodes, 51,781 alias claims, and seven pending items. Six are parked; this review targets exactly one of those six.

The exact pending item is:

- `Source=manual`, `SourceRef=manual`, `Parked=true`, `Attempts=1`.
- `Force=false`; CWD is empty and CwdIsRepo is false.
- OccurredAt, EnqueuedAt, and NextAttempt all represent `2026-09-05T20:09:19.093118Z`.
- Hints exactly `scry` and `db/migrations/0160_task_evidence_rules.sql`.
- No completed episode with this ID exists.
- LastError exactly: `resolve: memory: alias already claimed: "db/migrations/0160-task-evidence-rules.sql" belongs to 0160-task-evidence-rulessql, not dbmigrations0160-task-evidence-rulessql`.

The full durable text is an operation note about the reviewed six-record SQL migration-file merge, its manifest and rollback backup, preserved facts/spellings, unchanged recall suite results, and the unfinished overall goal. It does not ask for another merge or an alias transfer. `post-review.json` contains the complete untruncated text and all metadata.

Pending-record SHA-256 over its Go JSON serialization: `c51e3cc9a64cd7978db4c8e19711098b9114d93f075b03829608d2090fe8da4e`.

Text-byte SHA-256: `f2dcf3cb971179b59c4833e89c9d5b4e14c3b8a670eaf899018c00d33aa1d31f`.

## Identity and original evidence are intact

Canonical name `db/migrations/0160_task_evidence_rules.sql` has exactly **one** current entity whose canonical name matches under store normalization. Its owner is retained slug `0160-task-evidence-rulessql`, type `tool`, with the reviewed migration-file description, six aliases, and two repo refs. The natural slug `dbmigrations0160-task-evidence-rulessql` has no current entity.

The canonical name, retained slug, and all six listed aliases resolve to that survivor. There is no ambiguous canonical-name owner, stale listing mismatch, or unexplained missing claim in these reviewed spellings.

Independently restoring the original postmerge snapshot `/tmp/scry-migration0160-fresh-sep05.teYtyC/memory-20260905T200715Z.badger` showed that the following remain exactly equal to the postdeployment source:

- Entire survivor entity record, including type, description, aliases, repo refs, creation time, and LastSeen.
- All **nine** file facts, including text, raw relation, endpoints/values, times, confidence, invalidation state, and episode provenance.
- All reviewed spelling-to-owner routes.

Owner-record SHA-256: `f34f3a6527825ff40a450aa44dcb3ee0faeffdc0114145e2f6c157658232567d`.

Nine-file-facts JSON SHA-256: `430ac1928474370d44c908a352abd1d9bde08d6d738c78567be094943d29f369`.

The pending note did not yet exist in the original 20:07:15 snapshot, which is consistent with its 20:09:19 enqueue time; no claim is made that the earlier backup preserved that later note.

Read-only inspection preserved a combined hash of all facts, entities, completed episodes, alias claims, and pending items within each independent restore. Postdeployment replica combined hash before/after: `fef02d1654dd200dfe508e979d5b8c273232018b4f0c17cecc2aae9ae46412b3`.

## Retry implementation and failure behavior

Inspected exact `62cf6e0` source:

- `cmd/scry/memory.go:448`: `retry [episode-id]` passes the supplied ID to `memory.queue.retry`. Omitting the ID selects broad retry behavior and is outside this gate.
- `internal/daemon/memory_queue.go:369`: the handler reads pending items, skips every item whose ID differs, and changes only `Parked=false`, `Attempts=0`, and `NextAttempt=now` on the selected eligible item. It leaves Text, Source, SourceRef, OccurredAt, EnqueuedAt, Hints, CWD, CwdIsRepo, Force, and the prior LastError intact.
- `internal/memory/store/pending.go:64`: PutPending serializes the complete struct; there is no text trimming on the retry write. The separate queue-list endpoint truncates only its response copy, not durable text.
- `internal/memory/queue/queue.go:388`: normal processing gives the extractor the original text/source/time/CWD and appends original hints to its glossary. For manual input, the completed episode summary uses the original text rather than the provider's paraphrase.
- `internal/memory/resolve/resolve.go:85`: Apply runs atomically. Completion is recorded last, and an already completed episode is a no-op when Force is false.
- `internal/memory/queue/queue.go:471`: deterministic alias, retired-entity, invalid-slug, or fact-key conflicts park the original pending payload again. Failure does not delete it. Success deletes the pending entry only after Apply succeeds.

Relevant existing tests were inspected: `TestMemoryQueueListsAndRetriesParkedItems`, `TestDeterministicResolverFailureParksImmediately`, `TestApply_Idempotent`, fallback whole-episode collision rollback, and the canonical-name tests. Earlier independent code gates already exercised deterministic queue conflict payload preservation, idempotency, staged ambiguity rollback, and actual restored SQL-file routing. This bounded review did not rerun a retry handler or invoke an extractor; it restored and read stores only.

“Once-only” here means one explicit retry command for this ID. The ordinary pipeline retains its configured provider fallback and transport backoff behavior; a single command is not a promise of exactly one provider call. Any **new deterministic conflict** must be left parked intact for another review, not repeatedly forced through.

## Required immediate action checks for the lead

Before invoking the one-ID retry, the lead must take and verify a fresh nonempty backup and immediately recheck:

1. The same deployed prevention code is active.
2. This exact pending record still matches the reviewed payload/metadata fingerprint, is parked with one attempt, has Force false, and has no completed episode marker.
3. The canonical owner record, unique normalized canonical-owner group, reviewed spelling routes, and nine original file facts still match the fingerprints above.

Any mismatch invalidates this gate until reviewed. The current retry API does not combine a fingerprint comparison and scheduling reset in one compare-and-swap transaction; the lead's immediate read/write sequencing is therefore a real requirement, not an assertion that this review froze live state.

After one invocation, inspect whether this exact item completed or reparked. On success, verify the completed manual episode retains the full original note and recheck the reviewed file identity, original facts/history, and alias ownership. On any new conflict, preserve the pending record unchanged and stop retrying it. Normal ingestion may add explicit current observations; no authority is inferred to change the reviewed identity or discard old evidence.

The active goal's normal-write prevention and queue-drain workflow supplies authority for this narrowly reviewed once-only resumption after the original cause is fixed. No additional owner choice or historical repair is being authorized. Recall floors remain unchanged and unmet; this gate does not certify retrieval quality.

## Artifacts

Root: `/tmp/scry-pending-retry-gate-sep05.uZpjww`.

- `post-review.json`: exact pending record, current canonical owner/group, all nine original file facts, spelling routes, and complete-state preservation hash. SHA-256 `0964009befe6d67141033e178f8454d07d9d1159f972c2fb4f702e73efc3bb2f`.
- `original-postmerge-review.json`: independent original postmerge comparison.
- `code/`: exact `62cf6e0` archive plus read-only inspection helper `cmd/independent-pending-review/main.go`.
- Two fresh, review-owned replicas.

No retry execution, provider/model call, remember call, live write, deployment, broad queue action, shared-repository edit, or historical repair occurred during this gate.

