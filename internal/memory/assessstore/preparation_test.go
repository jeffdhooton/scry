package assessstore

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/extract"
)

func TestSavePreparationRequiresOwnershipAndPreservesImmutableManifest(t *testing.T) {
	s, e := Open(t.TempDir(), Options{})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	src, e := s.Capture(Source{EpisodeID: "prep", Text: "User: Atlas uses Postgres", OccurredAt: time.Now()})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "Atlas uses Postgres"}}}, Versions{Model: assess.Model, Rubric: assess.RubricV2, ContextPolicy: assess.ContextPolicyVersion})
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.Claim("owner")
	if e != nil {
		t.Fatal(e)
	}
	manifest := assess.Manifest{ContextPolicy: assess.ContextPolicyVersion, Budget: assess.BudgetReport{Method: "utf8_byte_upper_bound", DispatchBoundTokens: 70000}, MissingRaw: []string{"api_key=sk-abcdefghijklmnopqrstuvwxyz1234567890"}}
	if e = s.SavePreparation(j.ID, "wrong", manifest); !errors.Is(e, ErrOwnership) {
		t.Fatalf("ownership: %v", e)
	}
	before, e := s.Status()
	if e != nil {
		t.Fatal(e)
	}
	s.opts.Limits.MetadataBytes = before.MetadataBytes
	if e = s.SavePreparation(j.ID, "owner", manifest); !errors.Is(e, ErrCapacity) {
		t.Fatalf("capacity: %v", e)
	}
	got, _ := s.GetJob(j.ID)
	if got.Manifest.ContextPolicy != "" {
		t.Fatal("capacity failure partially persisted")
	}
	s.opts.Limits.MetadataBytes = 128 << 20
	if e = s.SavePreparation(j.ID, "owner", manifest); e != nil {
		t.Fatal(e)
	}
	got, _ = s.GetJob(j.ID)
	if got.Manifest.Budget.DispatchBoundTokens != 70000 || got.Manifest.ContextPolicy != assess.ContextPolicyVersion || len(got.Packet) != 0 || got.PacketHash != "" {
		t.Fatal("preparation metadata not preserved")
	}
	if strings.Contains(strings.Join(got.Manifest.MissingRaw, " "), "abcdefghijklmnopqrstuvwxyz") {
		t.Fatal("manifest secret not redacted")
	}
	if e = s.SavePreparation(j.ID, "owner", manifest); e != nil {
		t.Fatalf("identical save: %v", e)
	}
	changed := manifest
	changed.Budget.DispatchBoundTokens++
	if e = s.SavePreparation(j.ID, "owner", changed); !errors.Is(e, ErrConflict) {
		t.Fatalf("immutable preparation: %v", e)
	}
	if e = s.Finish(j.ID, "owner", Oversize, "core_evidence_oversize", nil); e != nil {
		t.Fatal(e)
	}
	if e = s.SavePreparation(j.ID, "owner", manifest); !errors.Is(e, ErrOwnership) {
		t.Fatalf("terminal overwrite: %v", e)
	}
	if _, e = s.Replay(j.ID); !errors.Is(e, ErrEvidenceUnavailable) {
		t.Fatalf("unsent preparation replayable: %v", e)
	}
}
