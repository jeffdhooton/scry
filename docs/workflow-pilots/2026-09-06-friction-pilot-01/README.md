# Scry workflow friction pilot SFW-20260906-01

Outcome: **one pilot completed**. The fresh verifier passed the bounded local
code graph acceptance; the initial friction history was retained in Scry and
retrieved with an exact match to its original 970-character manual summary.
This is one observed run, not evidence of cross-run recurrence.

## What ran

Jeff asked whether Scry should retain running workflow friction and agreed to
one pilot. We selected the completed local graph delivery, provided an independent
fresh-context verifier the original acceptance criteria, and asked it to disprove
completion using actual artifacts and isolated probes. No production code or
existing graph acceptance artifacts were changed; all delivery hashes still match.

[Independent review](../2026-09-06-independent-graph-verifier-01/REVIEW.md): PASS,
no new blocking defect. It reran all four real Go/TypeScript fixture scenarios and
added probes for function-value references and reversed multi-hop paths. Targeted
execution took 9.992 seconds. Total review duration was not measured. The retained
full-suite/vet evidence was inspected, not unnecessarily rerun.

The reviewer independently confirmed the documented limitation that `calls` can
mean a reference to a function value, without an invocation. A future precise
call-graph claim would need stronger evidence or an API distinction. That is an
optional contract improvement, not a newly established failure of the bounded
implementation-relationship acceptance.

## Friction review

The evidence and observations are retained in [friction-events.jsonl](friction-events.jsonl).
Each record has its own event ID, one run ID, cause qualification, evidence hashes,
resolution state and proposed correction. Costs were not measured and remain null.

| Event | Observed outcome | Review disposition |
| --- | --- | --- |
| 01: wrong RPC parameter in harness | Corrected before defect baseline; original failure preserved | Low-impact one-off. Prefer existing typed request objects in new integration tests; insufficient evidence for another global rule. |
| 02: get-tweet / research-policy conflict | Fetch stopped; user provided local article; instructions still conflict | Actionable shared-skill defect. Make the workflow consistent with the governing policy. Any policy exception requires its own explicit decision. No policy edit was applied. |
| 03: builder was sole completion judge | This pilot added a fresh verifier; no blocker found, one known limitation independently characterized | Independent review supplied additional evidence. Propose it for substantial runs; one pass does not prove its defect yield or economic benefit. No standing policy was installed. |

No event is labelled recurring: each has one observed occurrence in this one run.
The verifier's characterization is review evidence, not a second run or an
additional occurrence of the original completion process.

## What Scry retained

Initial episode:
`bca75a60776c4f8d74975a6534d4e81df125f2fbfbe196abf1f94ffcee976a4b`.
The manual API first returned durable queue acknowledgment. An early episode
query returned not found; we did not retry the successful write. Subsequent
`scry_episodes` calls for `Scry workflow friction pilot` returned the complete
original summary with all three event IDs and the evidence-file pointer.
[Exact returned episode](retrieved-episode.json) and [receipts](memory-receipts.json)
record that distinction. It was ingested about 115 seconds after submission;
this single observation is not a latency guarantee.

A separate closeout containing the review result and product proposal was queued
once as `840dcc2b70b1863ab46d0c92f32baf583ea47da520ebb20f0cd942a0e327102f`.
Its queued receipt is retained; its resolved retrieval was not required to claim
successful retrieval of the initial pilot record and is not asserted here.

## Product recommendation

**Yes: Scry should own a queryable friction history.** Existing manual memory
can support this small pilot, but generic extracted graph facts should not be
used as the event ledger or recurrence counter. The current episode API follows
entity-linked fact provenance, so an observation that produces no graph fact
may be stored yet absent from that lookup.

The smallest useful next feature would preserve typed, attributed events and
query them directly by repository, run, signature and time. Stable event IDs
make retries idempotent; recurrence counts distinct runs. A review emits proposed
changes with cited events, while adopted constraints stay separately governed.
See [behavioral contract proposal](friction-contract-proposal.md). This is a
proposal for a separately scoped implementation, not an implemented CLI or a
resumption of the frozen memory admission/identity work.

## Stop and boundaries

The single pass is finished. Only pilot/review artifacts and two explicitly
scoped manual memory records were added. No shared skill/policy/config edits,
schedule, deployment, daemon lifecycle, new memory schema or autonomous
instruction changes were performed. No chat transcripts or unrelated records
were submitted as friction history.
