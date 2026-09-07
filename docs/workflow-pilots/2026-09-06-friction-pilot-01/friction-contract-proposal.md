# Scry friction history: pilot contract proposal

This is a proposal, not an implemented interface or a reopened memory-admission backlog.

Scry should retain observations of workflow friction with source attribution.
A review can propose a correction; observations must not automatically become
standing instructions or permission changes.

The pilot uses existing manual memory ingestion. Source inspection confirms that
manual input becomes the exact episode summary after resolution (`internal/memory/queue/queue.go`),
but current episode lookup follows an entity's graph-fact provenance
(`internal/memory/recall/recall.go`). There is no guarantee that every stored
observation is discoverable as an entity-linked graph fact. Queue acknowledgment
proves durable submission, not resolved retrieval or recurrence tracking.

A future narrow feature should meet these behavioral requirements:

- Record an event with caller-supplied event ID, run ID, repository, observation
  time, specific friction, evidence reference/hash and outcome. Preserve observed
  facts separately from inferred causes and proposed corrections.
- Repeating the same event ID with identical data is an idempotent retry. The
  same signature in a new run is a distinct observation. Conflicting content at
  an existing ID must not silently overwrite the original event.
- List events by repository/run/time independently of LLM extraction and graph
  identity. A zero-fact event still appears. Exact IDs support retrieval receipts.
- Count recurrence by distinct observed runs, never by graph edges, repeated
  mentions, retries or different agents describing the same event. First pilot:
  three events, one run; cross-run recurrence is unknown.
- Keep a bounded review report: recurring signature, cited events, impact if
  measured, proposed owner/file change, and verification for the proposed fix.
  Review recommendations remain proposals until accepted; do not automatically
  rewrite skills, instructions or policies.
- Demonstrate close/reopen, exact-event retrieval, retry deduplication, separation
  of two runs, preservation of unresolved observations and no instruction activation.

Do not implement this from a social-media schema sketch. First use this one pilot
to establish whether the observations and review produce useful changes. A separate
implementation contract should choose storage/API boundaries only if the pilot warrants it.
