# Explicit workflow friction journal

Scry retains attributed friction observations in an independent journal. Record
an event when a task exposes a concrete correction, instruction conflict or
verification gap. Reuse the same run ID across agents working the same task.
At the end of the next three real tasks, review what recurred and whether an
approved correction prevented a repeat. No recurrence or time savings are inferred
from silence; checking those outcomes remains part of the manual pilot.

The journal is available through `scry friction record|get|list|review`, the
`friction.*` daemon RPCs, and four corresponding `scry_friction_*` MCP tools.
It requires no indexing, memory facts, provider credentials or extraction worker.
It does not collect automatically, schedule reviews, fetch evidence, or edit
instructions. Review repeats the caller's proposed corrections with event
citations; it does not invent a fix or authorize one.

## Record and retrieve

Write one event JSON object to a file or stdin:

```json
{
  "event_id": "TASK-20260907-01-01",
  "run_id": "TASK-20260907-01",
  "repository": "/absolute/repository/root",
  "recorded_at": "2026-09-07T12:00:00Z",
  "signature": "integration.rpc-parameter-contract",
  "observed": "The integration harness used the wrong request field.",
  "resolution": "Used the existing request type; the request succeeded.",
  "resolution_state": "corrected_in_run",
  "evidence": ["docs/run-01/failed-request.json", "docs/run-01/passing-check.txt"],
  "proposed_change": "Use the existing request type in future integration harnesses.",
  "proposed_owner": "task author",
  "proposed_file": "internal/daemon/example_test.go",
  "proposed_verification": "Run the integration check against the real handler.",
  "measured_user_time_cost_seconds": null,
  "change_approved": false
}
```

```bash
scry friction record event.json
scry friction get TASK-20260907-01-01
scry friction list --repo /absolute/repository/root --run-id TASK-20260907-01
scry friction review --repo /absolute/repository/root
```

The CLI list/review commands default `--repo` to cwd. Record uses the authored
`repository` and rejects `--repo` to prevent silently rewriting attribution.
RPC and MCP use `repository`, not `repo`, and require it for list/review.
Repository identity is the exact absolute clean client path; the service never
resolves symlinks, checks git or opens that repository. Use the same root spelling
on every call. Events may therefore cite repositories not present on the server.

Each successful record returns `{event_id, sha256, created}` after a synchronous
transaction commit. The SHA-256 covers Go's canonical typed JSON serialization of
the event. It does not hash the input file's whitespace or verify evidence bytes.
The timestamp is caller-supplied, so retrying the same ID and field values produces
the same receipt with `created:false`. JSON key order and formatting do not matter;
optional omitted/zero values use the schema's defaults. Text, arrays, supplied
timestamp spelling, evidence references and hashes are preserved.

IDs are unique across the entire journal, including repositories. Changed content
under the same ID fails with RPC code `-32009`; missing IDs return `-32004`.
No event can be overwritten or deleted through this API. Record a separately
identified observation for a later correction and cite the prior event ID in its
evidence; historical outcomes remain historical. Keep the same run ID for updates
within one task. A successful retry is never another occurrence.

Required event fields are shown above through `evidence`. `event_id`, `run_id`
and `signature` each accept 1–200 ASCII letters, digits, `.`, `_`, `:`, `-`, starting
with a letter or digit. At least one nonempty evidence reference is required.
Events are at most 16 KiB when serialized (CLI input files also have a 16 KiB cap).
Unknown fields fail explicitly. Optional fields also include `evidence_sha256`
(reference-to-SHA-256 map), `cause_status`, `cause`, `priority`,
`occurrences_observed_this_run`, and `distinct_prior_runs_verified`.
`change_approved` must be absent or false. Missing measured cost means unknown,
not zero. Record measured costs only, without overlapping the same cost in several
events; the review sums known event costs and reports how many events were measured.

## Routing a correction

An event may name `destination_kind`: exactly one of `fact`, `decision`,
`policy`, `skill`, `worker` or `gate`. The vocabulary is closed and the order is
meaningful — it runs from what an agent might read to what it cannot violate.
Omit the field when the destination is unknown. Nothing infers it from the
authored text, and an unrecognized value is rejected rather than coerced.

Review reports a `routing` block for each signature group. It reads the highest
rung any event in the group named, then counts the distinct run IDs recorded at
that rung:

| Status | Meaning |
|---|---|
| `unrouted` | no event named a destination |
| `holding` | one distinct run at the current rung |
| `outgrown` | two or more distinct runs at the current rung; the next rung is suggested |
| `terminal` | two or more distinct runs at `gate`; no stronger destination exists |

Runs are counted at the rung rather than across the group, so acknowledging a
promotion — a separately identified event at the higher rung, citing the prior
event — does not itself read as another failure. The ladder climbs one rung per
proven failure.

Events that named no destination never count toward a rung, so an unrouted
observation cannot look like evidence that the rung held. A group whose events
name different kinds keeps the highest as current and lists them all in
`kinds_observed`, so inconsistent authoring stays visible.

The verdict is a recommendation with citations. It does not edit an
instruction, approve a change, or write anything outside this journal.

## Review and bounds

List is ordered lexically by event ID, not by time. It returns at most 100 events
per page (configurable `--limit 1..100`). When `next_after` exists, pass it as
`--after` with unchanged filters. Pages are individual snapshots; backfilled events
whose IDs sort before an earlier cursor need a new listing to be discovered.
Both list and review support exact `--run-id`, exact `--signature`, inclusive
RFC3339 `--since`, and exclusive `--until`, interpreted against `recorded_at`.

Review refuses more than 100 matching events and rejects pagination parameters.
Narrow its time/run/signature scope rather than treating a truncated report as
complete. Within that scope, it groups exact signatures, counts distinct stored
run IDs, marks two or more runs as recurring, and includes full source events and
authored proposals. Several agents/events in one run, retries and caller-provided
occurrence/prior-run estimates cannot increase that run count. A reused or
inconsistent signature can still misgroup events; choosing signatures is manual.
Time scopes limit the counts to observations in that scope, not all-time totals.
One-run groups remain visible, including unresolved incidents.

## Storage and rollout

The owning daemon lazily opens `~/.scry/friction` with synchronous Badger writes.
Schema 1 is independent of the memory store. Unsupported schema versions fail
without resetting the journal. No migration, extraction or memory graph admission
policy changes are involved. Concurrent record requests serialize the insertion
and duplicate check; shutdown closes the journal and refuses late requests.

MCP profiles `all` and `local` expose the journal. The `memory` profile continues
to expose only existing memory tools. CLI uses the local daemon by default;
`friction --socket /absolute/socket` selects an explicit daemon directly without
auto-spawn. The journal does not follow `SCRY_MEMORY_SOCKET` or memory.socket.
That keeps this pilot separate from shared memory routing; cross-machine journal
placement is a later operational decision.

This change is local source work. Installing it and restarting the production
daemon/MCP processes remain separate rollout steps. The original pilot JSONL is
read by the integration test into a temporary journal only. Its artificial second
run is test data, not evidence of a real recurrence. The original three events
remain in [the pilot artifact](workflow-pilots/2026-09-06-friction-pilot-01/friction-events.jsonl).

## Acceptance and evidence

The user-approved bar is: load the three pilot observations; restart an isolated
service; retrieve each exactly without extraction; retry without duplicates;
record the same signature in a second run; review reports two distinct runs with
cited evidence and proposals. Preserve unresolved outcomes and never activate an
instruction. A fresh verifier judges these requirements independently.

`internal/daemon/friction_test.go` builds the CLI and runs production handlers in
separate service processes with temporary homes/sockets, tests graceful restart
and process kill, then exercises the real MCP bridge. Store tests additionally
cover concurrent retries, ID collisions, filters/pagination, review overflow,
validation, unknown impact and preservation of an unsupported schema.

```bash
CGO_ENABLED=0 go test ./internal/friction ./internal/daemon ./internal/mcp ./cmd/scry
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go vet ./...
```
