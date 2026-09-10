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
