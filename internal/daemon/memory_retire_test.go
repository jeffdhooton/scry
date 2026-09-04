package daemon

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	memstore "github.com/jeffdhooton/scry/internal/memory/store"
)

func TestMemoryRetireEntitiesDryRunDefaultsSafeAndBacksUpApply(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, err := d.memoryStore()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 4, 14, 0, 0, 0, time.UTC)
	for _, entity := range []memstore.Entity{{Slug: "scry", Name: "Scry", Type: "project"}, {Slug: "done", Name: "DONE", Type: "concept", Aliases: []string{"done-status"}}} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.PutFact(memstore.Fact{Src: "scry", Relation: "status", Dst: "done", RawRelation: "has_status", Fact: "Scry is done", ValidFrom: at, Confidence: .9, Episodes: []string{"ep"}}); err != nil {
		t.Fatal(err)
	}

	group := memstore.EntityRetirementRequest{ID: "done-status", Entity: "done", Why: "reviewed status value"}
	dryRun := true
	out, err := d.handleMemoryRetireEntities(context.Background(), mustJSON(t, MemoryRetireEntitiesParams{Groups: []memstore.EntityRetirementRequest{group}, DryRun: &dryRun}))
	if err != nil {
		t.Fatal(err)
	}
	first := out.(*MemoryRetireEntitiesResult)
	if first.Applied != 0 || first.BackupPath != "" || len(first.Groups) != 1 || len(first.Groups[0].FactFingerprints) != 1 {
		t.Fatalf("first dry run = %+v", first)
	}
	row := first.Groups[0].FactFingerprints[0]
	replacement := row.Snapshot
	replacement.Dst, replacement.Value = "", "DONE"
	group.Replacements = []memstore.EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: replacement, Why: "preserve status as an attribute"}}
	out, err = d.handleMemoryRetireEntities(context.Background(), mustJSON(t, MemoryRetireEntitiesParams{Groups: []memstore.EntityRetirementRequest{group}, DryRun: &dryRun}))
	if err != nil {
		t.Fatal(err)
	}
	reviewed := out.(*MemoryRetireEntitiesResult)
	if reviewed.Refused != 0 || !reviewed.Groups[0].Ready {
		t.Fatalf("reviewed dry run = %+v", reviewed)
	}
	group.Expected = reviewed.Groups[0].Expected

	// A raw caller omitting dry_run receives another preview, never a write.
	out, err = d.handleMemoryRetireEntities(context.Background(), mustJSON(t, MemoryRetireEntitiesParams{Groups: []memstore.EntityRetirementRequest{group}}))
	if err != nil {
		t.Fatal(err)
	}
	defaulted := out.(*MemoryRetireEntitiesResult)
	if !defaulted.DryRun || defaulted.Applied != 0 || defaulted.BackupPath != "" {
		t.Fatalf("omitted dry_run mutated: %+v", defaulted)
	}
	if _, err := st.GetEntity("done"); err != nil {
		t.Fatalf("default dry run retired entity: %v", err)
	}

	apply := false
	out, err = d.handleMemoryRetireEntities(context.Background(), mustJSON(t, MemoryRetireEntitiesParams{Groups: []memstore.EntityRetirementRequest{group}, DryRun: &apply}))
	if err != nil {
		t.Fatal(err)
	}
	applied := out.(*MemoryRetireEntitiesResult)
	if applied.Applied != 1 || applied.Refused != 0 || applied.BackupPath == "" || !applied.Groups[0].Applied {
		t.Fatalf("apply = %+v", applied)
	}
	info, err := os.Stat(applied.BackupPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("backup missing or empty: path=%q info=%v err=%v", applied.BackupPath, info, err)
	}
	if _, err := st.GetEntity("done"); !errors.Is(err, memstore.ErrNotFound) {
		t.Fatalf("status entity remains: %v", err)
	}
	facts, _ := st.FactsFrom("scry", true)
	if len(facts) != 1 || facts[0].Dst != "" || facts[0].Value != "DONE" || facts[0].RawRelation != "has_status" {
		t.Fatalf("fact not preserved as attribute: %+v", facts)
	}
}

func TestMemoryRetireEntitiesPreflightsWholeManifest(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, _ := d.memoryStore()
	at := time.Unix(40, 0).UTC()
	for _, entity := range []memstore.Entity{
		{Slug: "app", Name: "App", Type: "project"},
		{Slug: "ready", Name: "READY", Type: "concept"},
		{Slug: "failed", Name: "FAILED", Type: "concept"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	for i, status := range []string{"ready", "failed"} {
		if err := st.PutFact(memstore.Fact{Src: "app", Relation: "status", Dst: status, Fact: "status " + status, ValidFrom: at.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	ready := memstore.EntityRetirementRequest{Entity: "ready", Why: "status"}
	preview, _ := st.PreviewEntityRetirement(ready)
	row := preview.FactFingerprints[0]
	replacement := row.Snapshot
	replacement.Dst, replacement.Value = "", "READY"
	ready.Replacements = []memstore.EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: replacement, Why: "attribute"}}
	preview, _ = st.PreviewEntityRetirement(ready)
	ready.Expected = preview.Expected

	apply := false
	out, err := d.handleMemoryRetireEntities(context.Background(), mustJSON(t, MemoryRetireEntitiesParams{Groups: []memstore.EntityRetirementRequest{
		ready, {Entity: "failed", Why: "status"},
	}, DryRun: &apply}))
	if err != nil {
		t.Fatal(err)
	}
	result := out.(*MemoryRetireEntitiesResult)
	if result.Applied != 0 || result.Refused != 1 || result.BackupPath != "" {
		t.Fatalf("incomplete second group allowed partial apply: %+v", result)
	}
	for _, slug := range []string{"ready", "failed"} {
		if _, err := st.GetEntity(slug); err != nil {
			t.Fatalf("preflight failure retired %s: %v", slug, err)
		}
	}
}

func TestValidateMemoryRetirementIsolationRejectsSharedFactsAndTargets(t *testing.T) {
	groups := []memstore.EntityRetirementRequest{{ID: "a", Entity: "status-a"}, {ID: "b", Entity: "status-b"}}
	facts := []memstore.Fact{{Src: "status-a", Relation: "related_to", Dst: "status-b"}}
	if err := ValidateMemoryRetirementIsolation(groups, facts); err == nil {
		t.Fatal("shared touching fact was accepted")
	}
	groups[0].Replacements = []memstore.EntityRetirementReplacement{{Replacement: memstore.Fact{Src: "owner", Dst: "status-b"}}}
	if err := ValidateMemoryRetirementIsolation(groups, nil); err == nil {
		t.Fatal("replacement target retired by another group was accepted")
	}
	groups = []memstore.EntityRetirementRequest{
		{ID: "a", Entity: "status-a", Replacements: []memstore.EntityRetirementReplacement{{Replacement: memstore.Fact{Src: "owner", Relation: "status", Value: "done", ValidFrom: time.Unix(1, 0)}}}},
		{ID: "b", Entity: "status-b", Replacements: []memstore.EntityRetirementReplacement{{Replacement: memstore.Fact{Src: "owner", Relation: "status", Value: "done", ValidFrom: time.Unix(1, 0)}}}},
	}
	if err := ValidateMemoryRetirementIsolation(groups, nil); err == nil {
		t.Fatal("cross-group replacement key collision was accepted")
	}
}
