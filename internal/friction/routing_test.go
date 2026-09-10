package friction

import (
	"context"
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
