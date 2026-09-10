# Friction correction routing

**Date:** 2026-09-10
**Status:** approved, ready to plan

## Problem

The friction journal (shipped 2026-09-06, `docs/friction-journal.md`) captures a correction
and what the author proposes to do about it: `observed`, `resolution`, `proposed_change`,
`proposed_owner`, `proposed_file`, `proposed_verification`. `friction review` groups events by
exact signature, counts distinct stored run IDs, and marks two or more runs as recurring.

What the journal does not capture is what *kind* of thing a correction should become. An event
can name a target file, but nothing records whether the correction is a one-off fact, a rule
that should apply every time, or something that ought to be mechanically enforced. So the
journal can tell you a correction recurred and cannot tell you that recurrence means its
current destination was the wrong one.

That distinction is the useful half. A correction seen once is knowledge. The same correction
seen three times is evidence that wherever it was written down is not being honoured — and the
fix is not to write it down again in the same place.

Today that judgement happens in a human's head, once, if they happen to notice. The counts
needed to make it mechanically are already computed and already stored.

## Non-goals

- **Writes outside scry.** The journal records proposals and never authorizes one
  (`event.go:129`, and the 2026-09-06 decision: "No automatic collection, policy/skill edits,
  scheduling or shared-memory routing"). Routing classifies and proposes. A human applies.
- **Inferring the kind from prose.** `MEMORY_WORKFLOW_ASSESSMENT_2026-09-04.md` §5 already
  recorded that store-scale rules inferring from a sentence are wrong in bulk, and that the
  `value`-type experiment proved the model cannot be trusted with this class of judgement.
  The kind is authored, validated, and otherwise left alone.
- **A per-repo kind-to-path map.** Where a policy lives is a property of a repository's
  conventions, not of scry. The existing free-form `proposed_file` already carries it.
- **Orientation wiring.** Surfacing outgrown routings in `memory orient` crosses the
  journal/memory boundary the 2026-09-06 decision deliberately kept separate. It is a
  plausible next slice and it needs its own decision.
- **A new verb, RPC, or MCP tool.** Review already answers the question.
- **Automatic collection or scheduled review.** Unchanged from the pilot.

## Data model

One optional field on `Event` (`internal/friction/event.go:30`):

```go
DestinationKind string `json:"destination_kind,omitempty"`
```

Validated against a closed vocabulary; absent is permitted:

```go
var ladder = []string{"fact", "decision", "policy", "skill", "worker", "gate"}
```

The vocabulary is closed for the same reason the 39-relation vocabulary is closed: an open set
drifts (191 mis-typed entities are the standing evidence), and the ladder's ordering is
undefined over tokens it does not know. A `destination_kind` outside the vocabulary is
`ErrInvalid`, consistent with every other field.

### `omitempty` is load-bearing

`Record` stores `json.Marshal(e)` and a retry compares raw stored bytes
(`internal/friction/store.go:74-88`). A new field without `omitempty` changes the marshaled
bytes of every event that does not set it. The three pilot events in
`docs/workflow-pilots/2026-09-06-friction-pilot-01/friction-events.jsonl` would then fail their
idempotent retry with `ErrConflict`, and every previously issued SHA-256 receipt would stop
matching its event.

With `omitempty`, an event that omits the field marshals byte-identically to before. This is a
correctness constraint, not a formatting preference, and it gets an explicit regression test.

## The ladder

Rung order is enforcement strength — how little the agent has to cooperate for the correction
to hold:

| Rung | Kind | The agent... |
|---|---|---|
| 0 | `fact` | might read it |
| 1 | `decision` | can find out why |
| 2 | `policy` | should follow it every time |
| 3 | `skill` | runs it |
| 4 | `worker` | does not run it; it runs itself |
| 5 | `gate` | cannot violate it |

Recurrence is evidence that the current rung failed. Each proven failure moves the correction
one rung toward mechanical enforcement.

## The verdict

Computed per review group, from stored events only — never from the caller's
`occurrences_observed_this_run` or `distinct_prior_runs_verified` estimates, matching the
existing rule at `review.go:35-37`.

```
currentKind   = highest-rung destination_kind among the group's events
runsAtCurrent = count of distinct run_ids among events whose destination_kind == currentKind
```

Four statuses, exhaustive:

| Status | Condition | Suggestion |
|---|---|---|
| `unrouted` | no event in the group carries a kind | none — reported, never guessed |
| `holding` | fewer than 2 distinct runs at the current rung | none |
| `outgrown` | 2 or more distinct runs at the current rung | the next rung up |
| `terminal` | 2 or more distinct runs at `gate` | none — no higher rung exists |

### Why recurrence is counted at the rung, not across the group

The naive rule — recurring group at rung R suggests R+1 — never stops firing. Promoting a
correction from `fact` to `policy` means recording a new event at the higher rung (the journal's
documented pattern: a separately identified observation citing the prior event ID in its
evidence). That new event adds a third distinct run to the group, so the next review would
immediately demand `skill` for a policy that has not yet had the chance to fail.

Counting runs at the current rung fixes this without new state. A promotion acknowledgement
lands exactly one run at the new rung, so `runsAtCurrent` is 1 and the status is `holding`. If
the same signature recurs *again* at `policy`, that is two runs at `policy` — the policy
demonstrably did not hold — and the ladder climbs to `skill`.

One rung per proven failure, driven by new evidence rather than arithmetic over old evidence.
The loop terminates: at `gate` there is no next rung, and a gate that keeps failing is a defect
report, not a routing problem.

### Mixed kinds

A group whose events carry different kinds is not an error. The highest rung wins as
`currentKind`, and `kinds_observed` lists every kind present in rung order, so inconsistent
authoring is visible rather than silently averaged. A reused or inconsistent signature can
already misgroup events (`friction-journal.md`, "Review and bounds"); this makes one more form
of that visible without pretending to repair it.

## Review output

A `Routing` block on `Group` (`internal/friction/review.go:17`):

```go
type Routing struct {
    Status        string   `json:"status"`                    // unrouted | holding | outgrown | terminal
    CurrentKind   string   `json:"current_kind,omitempty"`
    KindsObserved []string `json:"kinds_observed"`            // rung order
    RunsAtCurrent int      `json:"runs_at_current_kind"`
    SuggestedKind string   `json:"suggested_kind,omitempty"`
    Rationale     string   `json:"rationale"`
}
```

`Rationale` is a plain sentence carrying the counts, so the report explains and cites itself
the way the existing review already carries full source events and proposals.

Group sort order is unchanged (`review.go:84-89`): distinct runs descending, then signature.
`Review.RecommendationsOnly` is already `true` and covers the verdict — it is a recommendation
with citations, never an instruction, and never an authorization.

## Surface

No new verb, RPC, or MCP tool, and no CLI or daemon change.

- `destination_kind` joins the `scry_friction_record` field list
  (`internal/mcp/friction_tools.go:18-37`) with a description stating the closed vocabulary and
  that omission is permitted. It needs its own entry rather than the generic `proposed_*` loop,
  because its description must name the vocabulary.
- The daemon decodes straight into `friction.Event`
  (`internal/daemon/friction_methods.go:30-37`), so the struct field is the whole daemon change.
- `runFriction` decodes the daemon's reply into `json.RawMessage` and re-encodes it
  (`cmd/scry/friction.go:103-122`), so `scry friction review` emits the routing block with no
  CLI change. `record` already reads JSON from a file or stdin and needs no new flag.

The MCP bridge forwards the whole object unnormalized (`friction_tools.go:61-76`), so the
schema entry is documentation for the calling agent; the daemon remains the validator.

## Testing

- **Ladder table tests** covering every status, plus mixed kinds, unrouted groups, the
  `gate` terminal case, and single-run groups at each rung.
- **Byte-identity regression:** a pre-change event fixture still marshals to its recorded
  SHA-256, and re-recording a stored event that omits the field returns `created:false` rather
  than `ErrConflict`.
- **Vocabulary validation:** unknown kind rejected with `ErrInvalid`; absent accepted.
- **Promotion sequence:** record at `fact` in two runs, assert `outgrown` suggesting
  `decision`; record the acknowledgement at `decision`; assert `holding`; record a second
  `decision` run; assert `outgrown` suggesting `policy`.
- **End-to-end pass-through** following the existing `internal/daemon/friction_test.go`
  pattern: record a routed event through the real MCP bridge, then assert the routing block
  reaches `scry friction review` stdout unchanged.

```bash
CGO_ENABLED=0 go test ./internal/friction ./internal/daemon ./internal/mcp ./cmd/scry
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go vet ./...
```

## Files

| File | Change |
|---|---|
| `internal/friction/event.go` | field, vocabulary, validation |
| `internal/friction/review.go` | `Routing` struct and computation |
| `internal/friction/routing_test.go` | new — ladder and promotion tables |
| `internal/friction/store_test.go` | byte-identity regression |
| `internal/daemon/friction_test.go` | end-to-end pass-through |
| `internal/mcp/friction_tools.go` | record schema field |
| `docs/friction-journal.md` | document the field, the ladder, the verdict |
| `docs/DECISIONS.md` | new entry in house format |

## Rollout

Local source work. Installing it and restarting the production daemon and MCP processes remain
separate steps, as with the pilot. No live-journal writes, no migration: existing events read
back unchanged with `destination_kind` absent, and their groups report `unrouted` until someone
records a routed observation.

## Follow-on, explicitly out of scope

For routed events to exist, an agent has to author `destination_kind` when it records one.
That is a dotfiles change — the global `CLAUDE.md` friction block or a skill — and it belongs
to its own slice. This spec delivers the mechanism; it does not deliver its adoption.
