package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	memstore "github.com/jeffdhooton/scry/internal/memory/store"
)

func TestMemoryMergeEntitiesDryRunThenApply(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, err := d.memoryStore()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	winner := memstore.Entity{Slug: "model", Name: "Model", Type: "tool"}
	loser := memstore.Entity{Slug: "model-stub", Name: "Model", Type: "concept", Aliases: []string{"model draft"}}
	if err := st.PutEntity(winner); err != nil {
		t.Fatal(err)
	}
	// Explicitly manufacture the legacy cross-type collision this command
	// exists to consolidate.
	if err := st.ClaimAlias(loser.Name, loser.Slug); err != nil {
		t.Fatal(err)
	}
	if err := st.PutEntity(loser); err != nil {
		t.Fatal(err)
	}
	if err := st.PutFact(memstore.Fact{Src: loser.Slug, Relation: "status", Value: "ready", Fact: "the model is ready", ValidFrom: at, Episodes: []string{"e1"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutFact(memstore.Fact{Src: winner.Slug, Relation: "uses", Value: "weights", Fact: "the model uses weights", ValidFrom: at.Add(time.Minute), Episodes: []string{"e2"}}); err != nil {
		t.Fatal(err)
	}

	group := memstore.EntityMergeRequest{ID: "model", Survivor: winner.Slug, Retire: []string{loser.Slug}, Why: "reviewed duplicate"}
	dryRun := true
	out, err := d.handleMemoryMergeEntities(context.Background(), mustJSON(t, MemoryMergeEntitiesParams{Groups: []memstore.EntityMergeRequest{group}, DryRun: &dryRun}))
	if err != nil {
		t.Fatal(err)
	}
	dry := out.(*MemoryMergeEntitiesResult)
	if dry.BackupPath != "" || dry.Applied != 0 || dry.Refused != 0 || len(dry.Groups) != 1 || !dry.Groups[0].Ready {
		t.Fatalf("dry run = %+v", dry)
	}
	if dry.Groups[0].CollisionDelta >= 0 {
		t.Fatalf("predicted collision delta = %d, want a reduction", dry.Groups[0].CollisionDelta)
	}
	group.Metadata = &dry.Groups[0].ProposedMetadata
	group.Expected = dry.Groups[0].Expected

	// Raw RPC callers that omit dry_run must not mutate anything.
	out, err = d.handleMemoryMergeEntities(context.Background(), mustJSON(t, MemoryMergeEntitiesParams{Groups: []memstore.EntityMergeRequest{group}}))
	if err != nil {
		t.Fatal(err)
	}
	defaulted := out.(*MemoryMergeEntitiesResult)
	if !defaulted.DryRun || defaulted.Applied != 0 || defaulted.BackupPath != "" {
		t.Fatalf("omitted dry_run did not default safely: %+v", defaulted)
	}
	if _, err := st.GetEntity(loser.Slug); err != nil {
		t.Fatalf("default dry run changed the loser: %v", err)
	}

	apply := false
	out, err = d.handleMemoryMergeEntities(context.Background(), mustJSON(t, MemoryMergeEntitiesParams{Groups: []memstore.EntityMergeRequest{group}, DryRun: &apply}))
	if err != nil {
		t.Fatal(err)
	}
	applied := out.(*MemoryMergeEntitiesResult)
	if applied.Applied != 1 || applied.Refused != 0 || applied.BackupPath == "" || !applied.Groups[0].Applied || !applied.Groups[0].CollisionVerified || applied.Groups[0].ObservedCollisionsAfter == nil {
		t.Fatalf("apply = %+v", applied)
	}
	if _, err := st.GetEntity(loser.Slug); !errors.Is(err, memstore.ErrNotFound) {
		t.Fatalf("loser still exists: %v", err)
	}
	if owner, ok, err := st.ResolveAlias(loser.Slug); err != nil || !ok || owner != winner.Slug {
		t.Fatalf("loser slug resolves to %q, %v, %v", owner, ok, err)
	}
}

func TestMemoryMergeEntitiesPreflightsWholeManifestBeforeBackup(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, _ := d.memoryStore()
	for _, e := range []memstore.Entity{{Slug: "a", Name: "A", Type: "tool"}, {Slug: "b", Name: "B", Type: "concept"}} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	// No facts means the group is hollow and must be refused before backup.
	out, err := d.handleMemoryMergeEntities(context.Background(), mustJSON(t, MemoryMergeEntitiesParams{Groups: []memstore.EntityMergeRequest{{Survivor: "a", Retire: []string{"b"}}}}))
	if err != nil {
		t.Fatal(err)
	}
	res := out.(*MemoryMergeEntitiesResult)
	if res.Refused != 1 || res.Applied != 0 || res.BackupPath != "" {
		t.Fatalf("unsafe manifest reached backup/apply: %+v", res)
	}
	if _, err := st.GetEntity("b"); err != nil {
		t.Fatalf("unsafe manifest changed store: %v", err)
	}
}

func TestMemoryMergeEntitiesUsesSequentialPredictionsAndObservedCounts(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, _ := d.memoryStore()
	at := time.Date(2026, 9, 4, 13, 0, 0, 0, time.UTC)
	groups := make([]memstore.EntityMergeRequest, 0, 2)
	for i, prefix := range []string{"alpha", "beta"} {
		winner := memstore.Entity{Slug: prefix + "-tool", Name: prefix + " tool", Type: "tool", Aliases: []string{"shared " + prefix}}
		loser := memstore.Entity{Slug: prefix + "-stub", Name: prefix + " stub", Type: "concept"}
		if err := st.PutEntity(winner); err != nil {
			t.Fatal(err)
		}
		if err := st.ClaimAlias("shared "+prefix, loser.Slug); err != nil {
			t.Fatal(err)
		}
		loser.Aliases = []string{"shared " + prefix}
		if err := st.PutEntity(loser); err != nil {
			t.Fatal(err)
		}
		if err := st.PutFact(memstore.Fact{Src: loser.Slug, Relation: "status", Value: "ready", Fact: prefix + " is ready", ValidFrom: at.Add(time.Duration(i) * time.Minute)}); err != nil {
			t.Fatal(err)
		}
		groups = append(groups, memstore.EntityMergeRequest{ID: prefix, Survivor: winner.Slug, Retire: []string{loser.Slug}, Why: "reviewed duplicate"})
	}

	dryRun := true
	out, err := d.handleMemoryMergeEntities(context.Background(), mustJSON(t, MemoryMergeEntitiesParams{Groups: groups, DryRun: &dryRun}))
	if err != nil {
		t.Fatal(err)
	}
	dry := out.(*MemoryMergeEntitiesResult)
	if len(dry.Groups) != 2 || dry.Groups[0].CollisionsAfter != dry.Groups[1].CollisionsBefore {
		t.Fatalf("predictions are not sequential: %+v", dry.Groups)
	}
	for i := range groups {
		groups[i].Metadata = &dry.Groups[i].ProposedMetadata
		groups[i].Expected = dry.Groups[i].Expected
	}
	apply := false
	out, err = d.handleMemoryMergeEntities(context.Background(), mustJSON(t, MemoryMergeEntitiesParams{Groups: groups, DryRun: &apply}))
	if err != nil {
		t.Fatal(err)
	}
	applied := out.(*MemoryMergeEntitiesResult)
	if applied.Applied != 2 {
		t.Fatalf("applied = %+v", applied)
	}
	for i, group := range applied.Groups {
		if !group.CollisionVerified || group.ObservedCollisionsAfter == nil || *group.ObservedCollisionsAfter != group.CollisionsAfter {
			t.Fatalf("group %d did not verify predicted collision count: %+v", i, group)
		}
	}
}

func TestLegacyInferredIdentityApplyEndpointsAreDisabled(t *testing.T) {
	d := newTestMemoryDaemon(t)
	if _, err := d.handleMemoryHygiene(context.Background(), mustJSON(t, MemoryHygieneParams{DryRun: false})); err == nil {
		t.Fatal("memory.hygiene still permits unreviewed apply")
	}
	if _, err := d.handleMemoryMigrate(context.Background(), mustJSON(t, MemoryMigrateParams{DryRun: false})); err == nil {
		t.Fatal("memory.migrate still permits inferred identity/value apply")
	}
}
