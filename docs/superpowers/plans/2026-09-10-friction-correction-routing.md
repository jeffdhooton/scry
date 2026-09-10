# Friction Correction Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a friction event record which kind of destination its correction belongs in, and let `friction review` report when that destination has demonstrably failed.

**Architecture:** One optional closed-vocabulary field on `friction.Event`, and a derived `Routing` block on each review group. Kinds form an enforcement ladder (`fact` → `gate`); recurrence counted *at the current rung* — not across the group — climbs it one rung per proven failure. No new store, no new state, no new RPC, verb, or MCP tool.

**Tech Stack:** Go 1.26.2, BadgerDB v4, cobra, `encoding/json` with `DisallowUnknownFields`. No CGO.

**Spec:** `docs/superpowers/specs/2026-09-10-friction-correction-routing-design.md`

## Global Constraints

- The new field **must** carry `omitempty`. `Record` stores `json.Marshal(e)` and a retry compares raw stored bytes (`internal/friction/store.go:74-88`). Without `omitempty`, every existing event's bytes change: the three pilot events fail their idempotent retry with `ErrConflict`, and every issued SHA-256 receipt stops matching.
- The vocabulary is closed and ordered exactly: `fact`, `decision`, `policy`, `skill`, `worker`, `gate`. Case-sensitive, exact match, no aliases, no trimming.
- Counts come only from stored events, never from `OccurrencesObservedThisRun` or `DistinctPriorRunsVerified` (existing rule, `internal/friction/review.go:35-37`).
- Nothing infers a kind from prose. Absent stays absent and is reported as `unrouted`.
- No writes outside scry. `ChangeApproved` stays forced-false; the verdict is a recommendation with citations, covered by the existing `Review.RecommendationsOnly = true`.
- Group sort order is unchanged: distinct runs descending, then signature.
- Every verification step runs with `CGO_ENABLED=0`.
- `internal/daemon/friction_test.go:176-178` already asserts pilot receipts survive a retry. It is the end-to-end backstop for the `omitempty` constraint and must stay green throughout.

---

### Task 1: The `destination_kind` field and its closed vocabulary

**Files:**
- Modify: `internal/friction/event.go:30-52` (struct), `internal/friction/event.go:88-140` (`Validate`)
- Test: `internal/friction/routing_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `Event.DestinationKind string`; `destinationLadder []string` (package-level, ordered); `destinationRung(kind string) (int, bool)` returning the ladder index and whether the kind is known. Task 2 uses both.

- [ ] **Step 1: Write the failing tests**

Create `internal/friction/routing_test.go`:

```go
package friction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDestinationKindVocabularyIsClosed(t *testing.T) {
	e := sample("event-1", "run-1")
	if err := e.Validate(); err != nil {
		t.Fatalf("omitted kind must stay valid: %v", err)
	}
	for _, kind := range []string{"fact", "decision", "policy", "skill", "worker", "gate"} {
		e.DestinationKind = kind
		if err := e.Validate(); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	// Near-misses are rejected rather than normalized: a silently coerced kind
	// would land the correction on the wrong rung.
	for _, kind := range []string{"Fact", "gate ", " policy", "rule", "runbook", "FACT"} {
		e.DestinationKind = kind
		if err := e.Validate(); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%q accepted: %v", kind, err)
		}
	}
}

func TestOmittedDestinationKindKeepsStoredBytesAndReceipt(t *testing.T) {
	s := openTestStore(t)
	e := sample("event-1", "run-1")
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	// Without omitempty this key appears, every stored event's bytes change,
	// and retries of existing events become ErrConflict.
	if strings.Contains(string(b), "destination_kind") {
		t.Fatalf("unrouted event serialized the field: %s", b)
	}
	want := sha256.Sum256(b)
	first, err := s.Record(e)
	if err != nil || !first.Created {
		t.Fatalf("record: %v %+v", err, first)
	}
	if first.SHA256 != hex.EncodeToString(want[:]) {
		t.Fatalf("receipt %s want %s", first.SHA256, hex.EncodeToString(want[:]))
	}
	retry, err := s.Record(e)
	if err != nil || retry.Created || retry.SHA256 != first.SHA256 {
		t.Fatalf("retry must be a no-op: %v %+v", err, retry)
	}
}

func TestRoutedEventSerializesTheField(t *testing.T) {
	e := sample("event-1", "run-1")
	e.DestinationKind = "policy"
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"destination_kind":"policy"`) {
		t.Fatalf("routed event lost the field: %s", b)
	}
	var round Event
	if err := Decode(b, &round); err != nil {
		t.Fatalf("decode rejected its own output: %v", err)
	}
	if round.DestinationKind != "policy" {
		t.Fatalf("round trip: %q", round.DestinationKind)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/friction -run 'DestinationKind|RoutedEvent' -v`
Expected: FAIL — `e.DestinationKind undefined (type Event has no field or method DestinationKind)`

- [ ] **Step 3: Add the ladder and the lookup**

In `internal/friction/event.go`, after the `identifier` regexp at line 54:

```go
// destinationLadder orders correction destinations by enforcement strength: a
// fact may be read, a gate cannot be violated. Closed, like the relation
// vocabulary — an open set drifts, and the ladder's ordering is undefined over
// tokens it does not know.
var destinationLadder = []string{"fact", "decision", "policy", "skill", "worker", "gate"}

// destinationRung reports a kind's ladder position and whether it is known.
func destinationRung(kind string) (int, bool) {
	for i, k := range destinationLadder {
		if k == kind {
			return i, true
		}
	}
	return 0, false
}
```

- [ ] **Step 4: Add the field**

In the `Event` struct, immediately after `ProposedVerification` (`event.go:46`):

```go
	DestinationKind             string            `json:"destination_kind,omitempty"`
```

- [ ] **Step 5: Validate it**

In `Validate`, immediately before the `ChangeApproved` check (`event.go:129`):

```go
	if e.DestinationKind != "" {
		if _, ok := destinationRung(e.DestinationKind); !ok {
			return invalid("destination_kind must be one of %s, or omitted", strings.Join(destinationLadder, ", "))
		}
	}
```

`strings` is already imported (`event.go:15`).

- [ ] **Step 6: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/friction -run 'DestinationKind|RoutedEvent' -v`
Expected: PASS, three tests.

- [ ] **Step 7: Verify nothing else regressed**

Run: `CGO_ENABLED=0 go test ./internal/friction ./internal/daemon`
Expected: PASS. `TestFrictionPilotThroughRestartedServiceAndCLI` proves the pilot receipts are unchanged.

- [ ] **Step 8: Commit**

```bash
git add internal/friction/event.go internal/friction/routing_test.go
git commit -m "Add a closed destination kind to friction events"
```

---

### Task 2: The routing verdict on review groups

**Files:**
- Modify: `internal/friction/review.go:9-33` (types), `internal/friction/review.go:38-91` (`Review`)
- Test: `internal/friction/routing_test.go` (extend)

**Interfaces:**
- Consumes: `destinationLadder`, `destinationRung` from Task 1.
- Produces: `Routing` struct with fields `Status`, `CurrentKind`, `KindsObserved`, `RunsAtCurrent`, `SuggestedKind`, `Rationale`; `Group.Routing Routing` (JSON key `routing`); `routingFor(runsByKind map[string]map[string]bool) Routing`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/friction/routing_test.go`, and add `"context"` to its imports:

```go
func routed(id, run, kind string) Event {
	e := sample(id, run)
	e.DestinationKind = kind
	return e
}

func reviewGroup(t *testing.T, events ...Event) Group {
	t.Helper()
	s := openTestStore(t)
	for _, e := range events {
		if _, err := s.Record(e); err != nil {
			t.Fatal(err)
		}
	}
	r, err := s.Review(context.Background(), Filter{Repository: "/fixture/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Groups) != 1 {
		t.Fatalf("expected one signature group, got %d", len(r.Groups))
	}
	return r.Groups[0]
}

func TestUnroutedGroupSuggestsNothing(t *testing.T) {
	g := reviewGroup(t, sample("e-1", "run-1"), sample("e-2", "run-2"))
	if g.Routing.Status != "unrouted" || g.Routing.CurrentKind != "" || g.Routing.SuggestedKind != "" {
		t.Fatalf("%+v", g.Routing)
	}
	if len(g.Routing.KindsObserved) != 0 || g.Routing.Rationale == "" {
		t.Fatalf("%+v", g.Routing)
	}
	// The group is still recurring; routing says nothing about what to do,
	// because nobody said where the correction went.
	if !g.Recurring {
		t.Fatal("recurrence must be unaffected by routing")
	}
}

func TestOneRunAtARungIsHolding(t *testing.T) {
	g := reviewGroup(t, routed("e-1", "run-1", "fact"))
	if g.Routing.Status != "holding" || g.Routing.CurrentKind != "fact" || g.Routing.SuggestedKind != "" {
		t.Fatalf("%+v", g.Routing)
	}
	if g.Routing.RunsAtCurrent != 1 {
		t.Fatalf("runs at rung: %d", g.Routing.RunsAtCurrent)
	}
}

func TestRepeatsInOneRunDoNotClimb(t *testing.T) {
	// Two agents on the same task are one run. Only distinct runs are evidence.
	g := reviewGroup(t, routed("e-1", "run-1", "fact"), routed("e-2", "run-1", "fact"))
	if g.Routing.Status != "holding" || g.Routing.RunsAtCurrent != 1 {
		t.Fatalf("%+v", g.Routing)
	}
}

func TestTwoRunsAtARungIsOutgrown(t *testing.T) {
	g := reviewGroup(t, routed("e-1", "run-1", "fact"), routed("e-2", "run-2", "fact"))
	if g.Routing.Status != "outgrown" || g.Routing.SuggestedKind != "decision" {
		t.Fatalf("%+v", g.Routing)
	}
	if g.Routing.RunsAtCurrent != 2 {
		t.Fatalf("runs at rung: %d", g.Routing.RunsAtCurrent)
	}
}

func TestLadderClimbsOncePerProvenFailure(t *testing.T) {
	e1 := routed("e-1", "run-1", "fact")
	e2 := routed("e-2", "run-2", "fact")
	e3 := routed("e-3", "run-3", "decision")
	e4 := routed("e-4", "run-4", "decision")

	g := reviewGroup(t, e1, e2)
	if g.Routing.Status != "outgrown" || g.Routing.SuggestedKind != "decision" {
		t.Fatalf("two fact runs: %+v", g.Routing)
	}
	// Acknowledging a promotion is a new event at the higher rung. Counting runs
	// across the whole group would read this as a third failure and demand policy
	// for a decision that has not yet had the chance to hold.
	g = reviewGroup(t, e1, e2, e3)
	if g.Routing.Status != "holding" || g.Routing.CurrentKind != "decision" || g.Routing.SuggestedKind != "" {
		t.Fatalf("after acknowledgement: %+v", g.Routing)
	}
	g = reviewGroup(t, e1, e2, e3, e4)
	if g.Routing.Status != "outgrown" || g.Routing.SuggestedKind != "policy" {
		t.Fatalf("after the decision recurred: %+v", g.Routing)
	}
}

func TestGateIsTerminal(t *testing.T) {
	g := reviewGroup(t, routed("e-1", "run-1", "gate"), routed("e-2", "run-2", "gate"))
	if g.Routing.Status != "terminal" || g.Routing.SuggestedKind != "" {
		t.Fatalf("%+v", g.Routing)
	}
	if g.Routing.CurrentKind != "gate" {
		t.Fatalf("%+v", g.Routing)
	}
}

func TestMixedKindsReportEveryRungInOrder(t *testing.T) {
	g := reviewGroup(t,
		routed("e-1", "run-1", "policy"),
		routed("e-2", "run-2", "fact"),
		routed("e-3", "run-3", "worker"),
	)
	if g.Routing.CurrentKind != "worker" || g.Routing.RunsAtCurrent != 1 || g.Routing.Status != "holding" {
		t.Fatalf("%+v", g.Routing)
	}
	want := []string{"fact", "policy", "worker"}
	if len(g.Routing.KindsObserved) != 3 {
		t.Fatalf("%+v", g.Routing)
	}
	for i, kind := range want {
		if g.Routing.KindsObserved[i] != kind {
			t.Fatalf("kinds observed %v want %v", g.Routing.KindsObserved, want)
		}
	}
}

func TestUnroutedEventsDoNotDiluteARung(t *testing.T) {
	// An event that named no destination is not evidence that the rung held.
	g := reviewGroup(t,
		routed("e-1", "run-1", "fact"),
		routed("e-2", "run-2", "fact"),
		sample("e-3", "run-3"),
	)
	if g.Routing.Status != "outgrown" || g.Routing.RunsAtCurrent != 2 || g.Routing.SuggestedKind != "decision" {
		t.Fatalf("%+v", g.Routing)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/friction -run 'Unrouted|Holding|Outgrown|Ladder|Gate|Mixed|Repeats|OneRun|TwoRuns|Dilute' -v`
Expected: FAIL — `g.Routing undefined (type Group has no field or method Routing)`

- [ ] **Step 3: Add the Routing type and the group field**

In `internal/friction/review.go`, after the `Proposal` type (line 15):

```go
// Routing reports where a signature's corrections were sent and whether that
// destination held. Recommendations only: a verdict cites stored counts and
// never authorizes a change.
type Routing struct {
	Status        string   `json:"status"` // unrouted | holding | outgrown | terminal
	CurrentKind   string   `json:"current_kind,omitempty"`
	KindsObserved []string `json:"kinds_observed"`
	RunsAtCurrent int      `json:"runs_at_current_kind"`
	SuggestedKind string   `json:"suggested_kind,omitempty"`
	Rationale     string   `json:"rationale"`
}
```

Add to the `Group` struct, after `Proposals` (line 25):

```go
	Routing Routing `json:"routing"`
```

- [ ] **Step 4: Implement the verdict**

Add to `internal/friction/review.go`, after the `Review` method:

```go
// routingFor derives the verdict from stored events alone. Recurrence is counted
// at the current rung, not across the group: acknowledging a promotion lands one
// run at the new rung, and only a second run there proves that rung failed too.
func routingFor(runsByKind map[string]map[string]bool) Routing {
	r := Routing{Status: "unrouted", KindsObserved: []string{}}
	current := -1
	for kind := range runsByKind {
		rung, ok := destinationRung(kind)
		if !ok {
			continue
		}
		r.KindsObserved = append(r.KindsObserved, kind)
		if rung > current {
			current = rung
		}
	}
	if current < 0 {
		r.Rationale = "No event in this group named a destination_kind, so there is no rung to judge."
		return r
	}
	sort.Slice(r.KindsObserved, func(i, j int) bool {
		a, _ := destinationRung(r.KindsObserved[i])
		b, _ := destinationRung(r.KindsObserved[j])
		return a < b
	})
	r.CurrentKind = destinationLadder[current]
	r.RunsAtCurrent = len(runsByKind[r.CurrentKind])
	switch {
	case r.RunsAtCurrent < 2:
		r.Status = "holding"
		r.Rationale = fmt.Sprintf("Routed to %s in one distinct run; that destination has not yet been shown to fail.", r.CurrentKind)
	case current == len(destinationLadder)-1:
		r.Status = "terminal"
		r.Rationale = fmt.Sprintf("Routed to gate and still recurring across %d distinct runs; no stronger destination exists, so treat this as a defect in the gate.", r.RunsAtCurrent)
	default:
		r.Status = "outgrown"
		r.SuggestedKind = destinationLadder[current+1]
		r.Rationale = fmt.Sprintf("Routed to %s and still recurring across %d distinct runs; the evidence supports at least %s.", r.CurrentKind, r.RunsAtCurrent, r.SuggestedKind)
	}
	return r
}
```

Add `"fmt"` to the imports (`review.go:3-7` currently has `context`, `math`, `sort`).

- [ ] **Step 5: Collect runs per kind and attach the verdict**

In `Review`, declare the accumulator beside `runs` (line 51):

```go
	kindRuns := map[string]map[string]map[string]bool{}
```

Inside the event loop, immediately after `runs[e.Signature][e.RunID] = true` (line 60):

```go
		if e.DestinationKind != "" {
			if kindRuns[e.Signature] == nil {
				kindRuns[e.Signature] = map[string]map[string]bool{}
			}
			if kindRuns[e.Signature][e.DestinationKind] == nil {
				kindRuns[e.Signature][e.DestinationKind] = map[string]bool{}
			}
			kindRuns[e.Signature][e.DestinationKind][e.RunID] = true
		}
```

In the finalize loop, immediately before `r.Groups = append(r.Groups, *g)` (line 82):

```go
		g.Routing = routingFor(kindRuns[signature])
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/friction -v`
Expected: PASS, including the pre-existing review tests.

- [ ] **Step 7: Commit**

```bash
git add internal/friction/review.go internal/friction/routing_test.go
git commit -m "Report when a correction outgrew its destination"
```

---

### Task 3: Offer the field to calling agents

**Files:**
- Modify: `internal/mcp/friction_tools.go:18-37`
- Test: `internal/mcp/friction_tools_test.go` (extend)

**Interfaces:**
- Consumes: the vocabulary from Task 1 (as prose in the description — the MCP layer forwards without normalizing, so the daemon stays the validator).
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Write the failing test**

Append to `internal/mcp/friction_tools_test.go`:

```go
func TestRecordSchemaOffersDestinationKindAsOptional(t *testing.T) {
	var record tool
	for _, td := range frictionToolDefinitions {
		if td.Name == "scry_friction_record" {
			record = td
		}
	}
	if record.Name == "" {
		t.Fatal("record tool missing")
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(record.InputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	desc, ok := schema.Properties["destination_kind"]
	if !ok {
		t.Fatal("destination_kind missing from the record schema")
	}
	// The description is the only place a calling agent learns the vocabulary.
	for _, kind := range []string{"fact", "decision", "policy", "skill", "worker", "gate"} {
		if !strings.Contains(string(desc), kind) {
			t.Fatalf("description omits %q: %s", kind, desc)
		}
	}
	for _, r := range schema.Required {
		if r == "destination_kind" {
			t.Fatal("destination_kind must stay optional")
		}
	}
}
```

Ensure `encoding/json` and `strings` are imported in that test file.

- [ ] **Step 2: Run the test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/mcp -run DestinationKind -v`
Expected: FAIL — "destination_kind missing from the record schema"

- [ ] **Step 3: Add the schema entry**

In `buildFrictionTools`, add to the `fields` map (after the `change_approved` entry, `friction_tools.go:30`):

```go
		"destination_kind": stringField("Optional destination for this correction, exactly one of fact, decision, policy, skill, worker, gate — ordered by how mechanically the correction is enforced. Omit when unknown; never guess. Review reports when a destination keeps failing across distinct runs."),
```

It needs its own entry rather than joining the generic `proposed_*` loop at line 32, because its description has to name the vocabulary.

- [ ] **Step 4: Run the test to verify it passes**

Run: `CGO_ENABLED=0 go test ./internal/mcp -v`
Expected: PASS, including the existing tool-count assertion (which counts tools, not fields, and is unaffected).

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/friction_tools.go internal/mcp/friction_tools_test.go
git commit -m "Offer the destination kind to recording agents"
```

---

### Task 4: Prove routing survives the real service and CLI

**Files:**
- Modify: `internal/daemon/friction_test.go:109-142` (extract the CLI helper), then extend
- Test: same file

**Interfaces:**
- Consumes: everything from Tasks 1–3.
- Produces: `frictionCLI(t *testing.T, home, socket string) func(input []byte, out any, args ...string) []byte`.

- [ ] **Step 1: Extract the CLI helper without changing behavior**

Replace the inline build-and-`cli` block inside `TestFrictionPilotThroughRestartedServiceAndCLI` (lines 118-142) with a call to a new package-level helper, and add the helper to the same file:

```go
// frictionCLI builds the real command-line tool once and returns a runner bound
// to one service socket. The binary lands in home so the temp dir cleans it up.
func frictionCLI(t *testing.T, home, socket string) func(input []byte, out any, args ...string) []byte {
	t.Helper()
	bin := filepath.Join(home, "scry-test")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/scry")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v %s", err, b)
	}
	return func(input []byte, out any, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		all := append([]string{"friction", "--socket", socket}, args...)
		cmd := exec.CommandContext(ctx, bin, all...)
		cmd.Env = []string{"HOME=" + home, "SCRY_MEMORY_SOCKET=/no-shared-memory"}
		cmd.Stdin = bytes.NewReader(input)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI %v: %v %s", args, err, b)
		}
		if out != nil {
			if err := json.Unmarshal(b, out); err != nil {
				t.Fatalf("CLI JSON: %v %s", err, b)
			}
		}
		return b
	}
}
```

In the pilot test, replace lines 118-142 with:

```go
	cli := frictionCLI(t, home, socket)
```

**Why this extraction is safe.** The current inline closure closes over the `socket` *variable*, so the restart at line 166 rebinds what it sees; the extracted helper captures the socket *value* at call time. Those differ only if the path changes across a restart, and it cannot: `startFrictionProcess` returns `filepath.Join(home, "journal.sock")` (`friction_test.go:106`), derived from `home`, which is fixed for the test. Step 2 confirms it.

- [ ] **Step 2: Run the existing test to verify the extraction is behavior-neutral**

Run: `CGO_ENABLED=0 go test ./internal/daemon -run TestFrictionPilotThroughRestartedServiceAndCLI -v`
Expected: PASS, exactly as before the extraction.

- [ ] **Step 3: Write the failing end-to-end test**

Append to `internal/daemon/friction_test.go`:

```go
func TestRoutingVerdictReachesReviewThroughCLI(t *testing.T) {
	home, err := os.MkdirTemp("/tmp", "sr-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	socket, stop := startFrictionProcess(t, home)
	defer stop(false)
	cli := frictionCLI(t, home, socket)

	repo := "/fixture/routing"
	record := func(id, run, kind string) {
		t.Helper()
		e := friction.Event{
			EventID: id, RunID: run, Repository: repo,
			RecordedAt: "2026-09-10T12:00:00Z", Signature: "routing.ladder",
			Observed: "The same correction was needed again.", Resolution: "Corrected in run.",
			ResolutionState: "corrected_in_run", Evidence: []string{"session:" + id},
			DestinationKind: kind,
		}
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		var receipt friction.Receipt
		cli(b, &receipt, "record", "-")
		if !receipt.Created {
			t.Fatalf("record %s: %+v", id, receipt)
		}
	}
	record("r-1", "run-1", "fact")
	record("r-2", "run-2", "fact")

	var review friction.Review
	cli(nil, &review, "review", "--repo", repo)
	if len(review.Groups) != 1 {
		t.Fatalf("groups: %+v", review)
	}
	got := review.Groups[0].Routing
	if got.Status != "outgrown" || got.CurrentKind != "fact" || got.SuggestedKind != "decision" {
		t.Fatalf("routing did not survive the daemon and CLI: %+v", got)
	}
	if got.RunsAtCurrent != 2 || got.Rationale == "" {
		t.Fatalf("%+v", got)
	}

	// An unknown kind is refused by the daemon, not silently coerced. The shared
	// cli helper fails the test on a nonzero exit, so run the binary directly here.
	bad := friction.Event{
		EventID: "r-3", RunID: "run-3", Repository: repo,
		RecordedAt: "2026-09-10T12:00:00Z", Signature: "routing.ladder",
		Observed: "x", Resolution: "y", ResolutionState: "unresolved",
		Evidence: []string{"session:r-3"}, DestinationKind: "rule",
	}
	encoded, err := json.Marshal(bad)
	if err != nil {
		t.Fatal(err)
	}
	reject := exec.Command(filepath.Join(home, "scry-test"), "friction", "--socket", socket, "record", "-")
	reject.Env = []string{"HOME=" + home, "SCRY_MEMORY_SOCKET=/no-shared-memory"}
	reject.Stdin = bytes.NewReader(encoded)
	combined, err := reject.CombinedOutput()
	if err == nil {
		t.Fatalf("unknown destination_kind accepted: %s", combined)
	}
	if !bytes.Contains(combined, []byte("destination_kind")) {
		t.Fatalf("rejection did not name the field: %s", combined)
	}
}
```

Every identifier used here — `bytes`, `exec`, `filepath`, `json`, `os`, `time`, `context`, `friction` — is already imported by this file.

- [ ] **Step 4: Run the test to verify it fails, then passes**

Run: `CGO_ENABLED=0 go test ./internal/daemon -run TestRoutingVerdictReachesReviewThroughCLI -v`

If Tasks 1–3 are complete this passes immediately; it is a regression guard on the pass-through, not a driver of new code. If it fails, the failure names which layer dropped the field.

- [ ] **Step 5: Run the full suite**

```bash
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go vet ./...
```
Expected: PASS. Record any pre-existing failure separately rather than absorbing it.

- [ ] **Step 6: Commit**

```bash
git add internal/daemon/friction_test.go
git commit -m "Prove the routing verdict survives service and CLI"
```

---

### Task 5: Document the field, the ladder, and the rule

**Files:**
- Modify: `docs/friction-journal.md`
- Modify: `docs/DECISIONS.md` (new entry at the top, below the header block)

**Interfaces:**
- Consumes: the final behavior from Tasks 1–4.
- Produces: nothing.

- [ ] **Step 1: Add a routing section to the journal doc**

Insert after the "Record and retrieve" section of `docs/friction-journal.md`, before "Review and bounds":

```markdown
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

The verdict is a recommendation with citations. It does not edit an
instruction, approve a change, or write anything outside this journal.
```

- [ ] **Step 2: Add the decision record**

Insert into `docs/DECISIONS.md` immediately after the `---` that closes the file header (before the 2026-09-06 friction journal entry):

```markdown
## 2026-09-10 — Correction routing is an enforcement ladder counted at the rung

**Decision.** Friction events may name an optional `destination_kind` from a closed,
ordered vocabulary — `fact`, `decision`, `policy`, `skill`, `worker`, `gate` — and review
reports a per-signature `routing` verdict. Recurrence is counted among distinct runs at
the group's highest named rung, not across the whole group. Two runs at a rung mark it
`outgrown` and name the next rung; `gate` is `terminal`. The field carries `omitempty`
so existing stored bytes and issued receipts are unchanged. No new verb, RPC, MCP tool,
store, or write outside scry.

**Why.** The journal could show that a correction recurred but not that its destination
was the wrong one. Repetition is evidence the current rung failed, and the counts needed
to say so were already stored. Counting across the group instead of at the rung would
demand a further promotion the moment one was acknowledged, so the loop would never
settle. The kind is authored rather than inferred, because store-scale rules over
sentences were already measured wrong in bulk.

**What would change our minds.** Evidence from real routed events could justify surfacing
outgrown routings in orientation, a per-repository kind-to-path map, or per-kind
thresholds. Each needs its own scope. Adoption — teaching agents to author the field —
is deliberately not part of this slice.
```

- [ ] **Step 3: Verify the docs build and the commit message lints**

Run: `CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go vet ./...`
Expected: PASS (docs do not affect the build; this confirms the branch is green before the final commit).

- [ ] **Step 4: Commit**

```bash
git add docs/friction-journal.md docs/DECISIONS.md
git commit -m "Document friction correction routing"
```

---

## Done when

- `CGO_ENABLED=0 go test ./...` and `CGO_ENABLED=0 go vet ./...` both pass.
- `TestFrictionPilotThroughRestartedServiceAndCLI` still passes, proving no stored event's bytes or receipts changed.
- A recorded routed event's verdict reaches `scry friction review` output through the real daemon and command-line tool.
- `docs/friction-journal.md` and `docs/DECISIONS.md` describe the field, the ladder, and the counting rule.

Not in this slice, by design: teaching agents to author the field (a dotfiles change), orientation wiring, and any write outside scry.
