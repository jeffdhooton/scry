package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func retirementWith(t *testing.T, st *Store, slug string, rewrite func(Fact) Fact) EntityRetirementRequest {
	t.Helper()
	req := EntityRetirementRequest{ID: "reviewed-status", Entity: slug, Why: "reviewed non-identity status value"}
	preview, err := st.PreviewEntityRetirement(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range preview.FactFingerprints {
		req.Replacements = append(req.Replacements, EntityRetirementReplacement{
			OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: rewrite(row.Snapshot), Why: "reviewed fact conversion",
		})
	}
	preview, err = st.PreviewEntityRetirement(req)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Ready {
		t.Fatalf("retirement not ready: %v", preview.Problems)
	}
	req.Expected = preview.Expected
	return req
}

func TestRetireEntityPreservesEveryFactAndRemovesIdentity(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	invalidAt := at.Add(time.Hour)
	for _, entity := range []Entity{
		{Slug: "scry", Name: "Scry", Type: "project"},
		{Slug: "ready", Name: "READY", Type: "concept", Aliases: []string{"ready-status"}},
		{Slug: "runner", Name: "Runner", Type: "tool"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	facts := []Fact{
		{Src: "scry", Relation: "status", Dst: "ready", RawRelation: "has_status", Fact: "Scry is ready", ValidFrom: at, Confidence: .91, Episodes: []string{"e1"}},
		{Src: "ready", Relation: "related_to", Dst: "runner", Fact: "the ready marker came from Runner", ValidFrom: at.Add(time.Minute), InvalidAt: &invalidAt, Confidence: .72, Episodes: []string{"e2", "e3"}},
	}
	for _, fact := range facts {
		if err := st.PutFact(fact); err != nil {
			t.Fatal(err)
		}
	}

	req := retirementWith(t, st, "ready", func(fact Fact) Fact {
		if fact.Src == "scry" {
			fact.Dst = ""
			fact.Value = "READY"
		} else {
			fact.Src = "scry"
		}
		return fact
	})
	preview, err := st.RetireEntity(req)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Applied || preview.CurrentFacts != 1 || preview.InvalidatedFacts != 1 {
		t.Fatalf("receipt = %+v", preview)
	}
	if _, err := st.GetEntity("ready"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retired entity remains: %v", err)
	}
	for _, spelling := range []string{"ready", "READY", "ready-status"} {
		if owner, found, err := st.ResolveAlias(spelling); err != nil || found {
			t.Errorf("retired spelling %q resolves to %q, found=%v err=%v", spelling, owner, found, err)
		}
	}
	got, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(facts) {
		t.Fatalf("fact count changed: got %d want %d", len(got), len(facts))
	}
	for _, fact := range got {
		if fact.Src == "ready" || fact.Dst == "ready" {
			t.Errorf("fact still references retired entity: %+v", fact)
		}
		if fact.Fact == "Scry is ready" && (fact.Value != "READY" || fact.Dst != "" || fact.RawRelation != "has_status") {
			t.Errorf("status was not preserved as an attribute: %+v", fact)
		}
		if fact.Fact == "the ready marker came from Runner" && (fact.InvalidAt == nil || !fact.InvalidAt.Equal(invalidAt) || len(fact.Episodes) != 2) {
			t.Errorf("invalidated fact history changed: %+v", fact)
		}
	}
}

func TestRetireEntityRequiresCompleteImmutableFactReview(t *testing.T) {
	st := openTemp(t)
	at := time.Unix(10, 0).UTC()
	for _, entity := range []Entity{{Slug: "app", Name: "App", Type: "project"}, {Slug: "done", Name: "DONE", Type: "concept"}} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	original := Fact{Src: "app", Relation: "status", Dst: "done", Fact: "App is done", ValidFrom: at, Confidence: .8, Episodes: []string{"ep"}}
	if err := st.PutFact(original); err != nil {
		t.Fatal(err)
	}
	bare, err := st.PreviewEntityRetirement(EntityRetirementRequest{Entity: "done", Why: "status"})
	if err != nil {
		t.Fatal(err)
	}
	if bare.Ready || len(bare.FactFingerprints) != 1 || !strings.Contains(strings.Join(bare.Problems, " "), "no reviewed replacement") {
		t.Fatalf("incomplete review was accepted: %+v", bare)
	}

	row := bare.FactFingerprints[0]
	changed := row.Snapshot
	changed.Dst = ""
	changed.Value = "DONE"
	changed.Fact = "rewritten text"
	bad := EntityRetirementRequest{Entity: "done", Why: "status", Replacements: []EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: changed, Why: "convert"}}}
	preview, err := st.PreviewEntityRetirement(bad)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), "changes fact text") {
		t.Fatalf("payload mutation was accepted: %+v", preview)
	}
}

func TestRetireEntityPreservesAttributeShapeAndValue(t *testing.T) {
	setup := func(t *testing.T) (*Store, EntityRetirementRequest, EntityMergeFactFingerprint) {
		t.Helper()
		st := openTemp(t)
		for _, entity := range []Entity{
			{Slug: "obsolete-status", Name: "Obsolete status", Type: "concept"},
			{Slug: "app", Name: "App", Type: "project"},
			{Slug: "unrelated", Name: "Unrelated", Type: "concept"},
		} {
			if err := st.PutEntity(entity); err != nil {
				t.Fatal(err)
			}
		}
		if err := st.PutFact(Fact{Src: "obsolete-status", Relation: "status", Value: "ORIGINAL_LITERAL", Fact: "legacy status reading", ValidFrom: time.Unix(15, 0).UTC()}); err != nil {
			t.Fatal(err)
		}
		req := EntityRetirementRequest{Entity: "obsolete-status", Why: "reviewed non-identity"}
		preview, err := st.PreviewEntityRetirement(req)
		if err != nil || len(preview.FactFingerprints) != 1 {
			t.Fatalf("preview=%+v err=%v", preview, err)
		}
		return st, req, preview.FactFingerprints[0]
	}

	t.Run("literal mutation", func(t *testing.T) {
		st, req, row := setup(t)
		updated := row.Snapshot
		updated.Src = "app"
		updated.Value = "DIFFERENT_LITERAL"
		req.Replacements = []EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: updated, Why: "bad rewrite"}}
		preview, err := st.PreviewEntityRetirement(req)
		if err != nil {
			t.Fatal(err)
		}
		if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), "changes an existing attribute value") {
			t.Fatalf("attribute literal mutation was accepted: %+v", preview)
		}
	})

	t.Run("attribute to edge", func(t *testing.T) {
		st, req, row := setup(t)
		updated := row.Snapshot
		updated.Src, updated.Dst, updated.Value = "app", "unrelated", ""
		req.Replacements = []EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: updated, Why: "bad rewrite"}}
		preview, err := st.PreviewEntityRetirement(req)
		if err != nil {
			t.Fatal(err)
		}
		if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), "turns an attribute into an edge") {
			t.Fatalf("attribute-to-edge rewrite was accepted: %+v", preview)
		}
	})
}

func TestRetireEntityRefusesMalformedOriginalFact(t *testing.T) {
	st := openTemp(t)
	for _, entity := range []Entity{
		{Slug: "obsolete", Name: "Obsolete", Type: "concept"},
		{Slug: "target", Name: "Target", Type: "project"},
		{Slug: "owner", Name: "Owner", Type: "project"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	legacy := Fact{Src: "obsolete", Relation: "related_to", Dst: "target", Value: "LITERAL_MUST_SURVIVE", Fact: "legacy malformed fact", ValidFrom: time.Unix(17, 0).UTC()}
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(factKey(legacy.Src, legacy.Relation, legacy.KeyDst(), legacy.ValidFrom), encoded); err != nil {
			return err
		}
		return txn.Set(adjKey(legacy.Dst, legacy.Src, legacy.Relation, legacy.ValidFrom), nil)
	}); err != nil {
		t.Fatal(err)
	}
	req := EntityRetirementRequest{Entity: "obsolete", Why: "reviewed non-identity"}
	preview, err := st.PreviewEntityRetirement(req)
	if err != nil || len(preview.FactFingerprints) != 1 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	row := preview.FactFingerprints[0]
	updated := row.Snapshot
	updated.Src, updated.Value = "owner", ""
	req.Replacements = []EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: updated, Why: "attempted cleanup"}}
	preview, err = st.PreviewEntityRetirement(req)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), "malformed with both dst and value") {
		t.Fatalf("malformed original was accepted: %+v", preview)
	}
}

func TestRetireEntityRemovesHollowNodeAndStaleWrongOwnerClaim(t *testing.T) {
	st := openTemp(t)
	status := Entity{Slug: "obsolete-status", Name: "OBSOLETE", Type: "concept", Aliases: []string{"old-state"}}
	other := Entity{Slug: "other", Name: "Other", Type: "project"}
	for _, entity := range []Entity{status, other} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	// Manufacture a stale legacy claim: Other owns the index key but does
	// not list the retired spelling. The reviewed retirement must not keep it.
	if err := st.ClaimAlias("old-state", "other"); err != nil {
		t.Fatal(err)
	}
	req := EntityRetirementRequest{Entity: status.Slug, Why: "reviewed hollow status node"}
	preview, err := st.PreviewEntityRetirement(req)
	if err != nil || !preview.Ready || len(preview.FactFingerprints) != 0 {
		t.Fatalf("hollow retirement preview=%+v err=%v", preview, err)
	}
	req.Expected = preview.Expected
	if _, err := st.RetireEntity(req); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetEntity(status.Slug); !errors.Is(err, ErrNotFound) {
		t.Fatalf("hollow status remains: %v", err)
	}
	if owner, found, err := st.ResolveAlias("old-state"); err != nil || found {
		t.Fatalf("stale wrong-owner claim remains: owner=%q found=%v err=%v", owner, found, err)
	}
}

func TestRetireEntityReviewsAndRemovesStaleAdjacencyReferences(t *testing.T) {
	st := openTemp(t)
	for _, entity := range []Entity{
		{Slug: "obsolete", Name: "Obsolete", Type: "concept"},
		{Slug: "owner", Name: "Owner", Type: "project"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	ghostKey := adjKey("obsolete", "owner", "status", time.Unix(19, 0).UTC())
	if err := st.db.Update(func(txn *badger.Txn) error { return txn.Set(ghostKey, []byte("legacy-ghost")) }); err != nil {
		t.Fatal(err)
	}

	req := EntityRetirementRequest{Entity: "obsolete", Why: "reviewed hollow non-identity"}
	preview, err := st.PreviewEntityRetirement(req)
	if err != nil || preview.Ready || len(preview.Adjacencies) != 1 || !preview.Adjacencies[0].Stale || preview.Adjacencies[0].ValueBase64 != "bGVnYWN5LWdob3N0" || preview.Adjacencies[0].ValueBytes != 12 || !preview.Adjacencies[0].NeedsReview {
		t.Fatalf("stale adjacency was not exposed for review: preview=%+v err=%v", preview, err)
	}
	req.ReviewedAdjacencies = []EntityRetirementAdjacencyReview{{Key: preview.Adjacencies[0].Key, ExpectedSHA256: preview.Adjacencies[0].SHA256, Why: "reviewed stale legacy reverse-index payload"}}
	preview, err = st.PreviewEntityRetirement(req)
	if err != nil || !preview.Ready {
		t.Fatalf("explicit stale adjacency review was refused: preview=%+v err=%v", preview, err)
	}
	req.Expected = preview.Expected
	if _, err := st.RetireEntity(req); err != nil {
		t.Fatal(err)
	}
	if err := st.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(ghostKey)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("stale adjacency remains: %w", err)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRetireEntityReviewsAndRemovesMalformedAdjacencyReferences(t *testing.T) {
	st := openTemp(t)
	for _, entity := range []Entity{
		{Slug: "obsolete", Name: "Obsolete", Type: "concept"},
		{Slug: "owner", Name: "Owner", Type: "project"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	ghostKeys := [][]byte{
		[]byte(prefixAdj + "obsolete:owner:status:not-a-time"),
		[]byte(prefixAdj + "other:obsolete:status"),
	}
	if err := st.db.Update(func(txn *badger.Txn) error {
		for _, key := range ghostKeys {
			if err := txn.Set(key, nil); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	req := EntityRetirementRequest{Entity: "obsolete", Why: "reviewed hollow non-identity"}
	preview, err := st.PreviewEntityRetirement(req)
	if err != nil || preview.Ready || len(preview.Adjacencies) != len(ghostKeys) {
		t.Fatalf("malformed adjacencies were not exposed: preview=%+v err=%v", preview, err)
	}
	for _, row := range preview.Adjacencies {
		if !row.Stale || row.CanonicalFactKey != "" || !row.NeedsReview {
			t.Fatalf("malformed adjacency was not classified stale: %+v", row)
		}
		req.ReviewedAdjacencies = append(req.ReviewedAdjacencies, EntityRetirementAdjacencyReview{Key: row.Key, ExpectedSHA256: row.SHA256, Why: "reviewed malformed reverse-index ghost"})
	}
	preview, err = st.PreviewEntityRetirement(req)
	if err != nil || !preview.Ready {
		t.Fatalf("explicit malformed adjacency reviews were refused: preview=%+v err=%v", preview, err)
	}
	req.Expected = preview.Expected
	if _, err := st.RetireEntity(req); err != nil {
		t.Fatal(err)
	}
	if err := st.db.View(func(txn *badger.Txn) error {
		for _, key := range ghostKeys {
			if _, err := txn.Get(key); !errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("malformed adjacency remains: %q: %w", key, err)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRetireEntityAdjacencyDriftAbortsAtomically(t *testing.T) {
	st := openTemp(t)
	for _, entity := range []Entity{
		{Slug: "obsolete", Name: "Obsolete", Type: "concept"},
		{Slug: "owner", Name: "Owner", Type: "project"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	req := EntityRetirementRequest{Entity: "obsolete", Why: "reviewed hollow non-identity"}
	preview, err := st.PreviewEntityRetirement(req)
	if err != nil || !preview.Ready {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	req.Expected = preview.Expected
	ghostKey := adjKey("obsolete", "owner", "status", time.Unix(21, 0).UTC())
	if err := st.db.Update(func(txn *badger.Txn) error { return txn.Set(ghostKey, nil) }); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RetireEntity(req); err == nil || !strings.Contains(err.Error(), "snapshot changed") {
		t.Fatalf("want adjacency snapshot drift, got %v", err)
	}
	if _, err := st.GetEntity("obsolete"); err != nil {
		t.Fatalf("drifted retirement deleted entity: %v", err)
	}
}

func TestRetireEntitySnapshotDriftAndPostconditionAreAtomic(t *testing.T) {
	newStore := func(t *testing.T) (*Store, EntityRetirementRequest) {
		st := openTemp(t)
		at := time.Unix(20, 0).UTC()
		for _, entity := range []Entity{{Slug: "app", Name: "App", Type: "project"}, {Slug: "failed", Name: "FAILED", Type: "concept"}} {
			if err := st.PutEntity(entity); err != nil {
				t.Fatal(err)
			}
		}
		if err := st.PutFact(Fact{Src: "app", Relation: "status", Dst: "failed", Fact: "App failed", ValidFrom: at, Episodes: []string{"e1"}}); err != nil {
			t.Fatal(err)
		}
		req := retirementWith(t, st, "failed", func(fact Fact) Fact { fact.Dst, fact.Value = "", "FAILED"; return fact })
		return st, req
	}

	t.Run("snapshot drift", func(t *testing.T) {
		st, req := newStore(t)
		entity, _ := st.GetEntity("failed")
		entity.Description = "changed after review"
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
		if _, err := st.RetireEntity(req); err == nil || !strings.Contains(err.Error(), "snapshot changed") {
			t.Fatalf("want snapshot changed, got %v", err)
		}
		if _, err := st.GetEntity("failed"); err != nil {
			t.Fatalf("failed apply retired entity: %v", err)
		}
	})

	t.Run("postcondition callback", func(t *testing.T) {
		st, req := newStore(t)
		if _, err := st.RetireEntityChecked(req, func([]Entity, []Fact) error { return errors.New("injected audit failure") }); err == nil || !strings.Contains(err.Error(), "injected audit failure") {
			t.Fatalf("want callback failure, got %v", err)
		}
		if _, err := st.GetEntity("failed"); err != nil {
			t.Fatalf("callback failure retired entity: %v", err)
		}
		facts, _ := st.AllFacts()
		if len(facts) != 1 || facts[0].Dst != "failed" {
			t.Fatalf("callback failure changed facts: %+v", facts)
		}
	})
}

func TestRetireEntitiesManifestDriftAbortsEveryGroup(t *testing.T) {
	st := openTemp(t)
	at := time.Unix(25, 0).UTC()
	for _, entity := range []Entity{
		{Slug: "app", Name: "App", Type: "project"},
		{Slug: "ready", Name: "READY", Type: "concept"},
		{Slug: "failed", Name: "FAILED", Type: "concept"},
	} {
		if err := st.PutEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	for i, status := range []string{"ready", "failed"} {
		if err := st.PutFact(Fact{Src: "app", Relation: "status", Dst: status, Fact: "App is " + status, ValidFrom: at.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	ready := retirementWith(t, st, "ready", func(fact Fact) Fact { fact.Dst, fact.Value = "", "READY"; return fact })
	ready.ID = "ready-status"
	failed := retirementWith(t, st, "failed", func(fact Fact) Fact { fact.Dst, fact.Value = "", "FAILED"; return fact })
	failed.ID = "failed-status"
	failed.Expected.Entities["failed"] = "reviewed-snapshot-was-corrupted"

	if _, err := st.RetireEntitiesChecked([]EntityRetirementRequest{ready, failed}, nil); err == nil || !strings.Contains(err.Error(), "snapshot changed") {
		t.Fatalf("want atomic snapshot refusal, got %v", err)
	}
	for _, slug := range []string{"ready", "failed"} {
		if _, err := st.GetEntity(slug); err != nil {
			t.Fatalf("failed manifest partially retired %s: %v", slug, err)
		}
	}
	facts, err := st.FactsFrom("app", false)
	if err != nil || len(facts) != 2 || facts[0].Dst == "" || facts[1].Dst == "" {
		t.Fatalf("failed manifest partially rewrote facts: %+v err=%v", facts, err)
	}
}

type blockingDurableBackup struct {
	bytes.Buffer
	syncStarted chan struct{}
	releaseSync chan struct{}
	closed      bool
}

func (w *blockingDurableBackup) Sync() error {
	close(w.syncStarted)
	<-w.releaseSync
	return nil
}

func (w *blockingDurableBackup) Close() error {
	w.closed = true
	return nil
}

func TestBackupAndRetireExcludesConcurrentWritersUntilAtomicApply(t *testing.T) {
	st := openTemp(t)
	if err := st.PutEntity(Entity{Slug: "obsolete", Name: "Obsolete", Type: "concept"}); err != nil {
		t.Fatal(err)
	}
	req := EntityRetirementRequest{Entity: "obsolete", Why: "reviewed hollow non-identity"}
	preview, err := st.PreviewEntityRetirement(req)
	if err != nil || !preview.Ready {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	req.Expected = preview.Expected
	w := &blockingDurableBackup{syncStarted: make(chan struct{}), releaseSync: make(chan struct{})}
	maintenanceDone := make(chan error, 1)
	go func() {
		_, _, err := st.BackupAndRetireEntities(w, []EntityRetirementRequest{req})
		maintenanceDone <- err
	}()
	<-w.syncStarted

	writeStarted := make(chan struct{})
	writeDone := make(chan error, 1)
	go func() {
		close(writeStarted)
		writeDone <- st.PutEntity(Entity{Slug: "concurrent", Name: "Concurrent", Type: "project"})
	}()
	<-writeStarted
	select {
	case err := <-writeDone:
		t.Fatalf("writer crossed backup/apply boundary: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(w.releaseSync)
	if err := <-maintenanceDone; err != nil {
		t.Fatal(err)
	}
	if err := <-writeDone; err != nil {
		t.Fatal(err)
	}
	if !w.closed || w.Len() == 0 {
		t.Fatalf("backup was not durably completed before apply: closed=%v bytes=%d", w.closed, w.Len())
	}
	if _, err := st.GetEntity("obsolete"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retirement did not apply: %v", err)
	}
	if _, err := st.GetEntity("concurrent"); err != nil {
		t.Fatalf("blocked writer did not resume after apply: %v", err)
	}
}

func TestRetireEntityRefusesCollisionSelfLoopAndExternalListing(t *testing.T) {
	setup := func(t *testing.T) (*Store, Fact) {
		st := openTemp(t)
		at := time.Unix(30, 0).UTC()
		for _, entity := range []Entity{{Slug: "app", Name: "App", Type: "project"}, {Slug: "failed", Name: "FAILED", Type: "concept"}} {
			if err := st.PutEntity(entity); err != nil {
				t.Fatal(err)
			}
		}
		fact := Fact{Src: "app", Relation: "status", Dst: "failed", Fact: "App failed", ValidFrom: at}
		if err := st.PutFact(fact); err != nil {
			t.Fatal(err)
		}
		return st, fact
	}

	t.Run("self loop", func(t *testing.T) {
		st, _ := setup(t)
		bare, _ := st.PreviewEntityRetirement(EntityRetirementRequest{Entity: "failed", Why: "status"})
		row := bare.FactFingerprints[0]
		loop := row.Snapshot
		loop.Src, loop.Dst = "app", "app"
		req := EntityRetirementRequest{Entity: "failed", Why: "status", Replacements: []EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: loop, Why: "bad review"}}}
		preview, err := st.PreviewEntityRetirement(req)
		if err != nil {
			t.Fatal(err)
		}
		if preview.Ready || len(preview.SelfLoops) == 0 {
			t.Fatalf("self-loop replacement was accepted: %+v", preview)
		}
	})

	t.Run("key collision", func(t *testing.T) {
		st, original := setup(t)
		attribute := original
		attribute.Dst, attribute.Value = "", "FAILED"
		attribute.Fact = "an existing fact at the replacement key"
		if err := st.PutFact(attribute); err != nil {
			t.Fatal(err)
		}
		bare, _ := st.PreviewEntityRetirement(EntityRetirementRequest{Entity: "failed", Why: "status"})
		row := bare.FactFingerprints[0]
		updated := row.Snapshot
		updated.Dst, updated.Value = "", "FAILED"
		req := EntityRetirementRequest{Entity: "failed", Why: "status", Replacements: []EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: updated, Why: "convert"}}}
		preview, err := st.PreviewEntityRetirement(req)
		if err != nil {
			t.Fatal(err)
		}
		if preview.Ready || len(preview.FactKeyCollisions) == 0 {
			t.Fatalf("colliding replacement was accepted: %+v", preview)
		}
	})

	t.Run("external listing", func(t *testing.T) {
		st, _ := setup(t)
		app, _ := st.GetEntity("app")
		app.Aliases = []string{"FAILED"}
		encoded, _ := json.Marshal(app)
		if err := st.db.Update(func(txn *badger.Txn) error { return txn.Set([]byte(prefixEntity+app.Slug), encoded) }); err != nil {
			t.Fatal(err)
		}
		preview, err := st.PreviewEntityRetirement(EntityRetirementRequest{Entity: "failed", Why: "status"})
		if err != nil {
			t.Fatal(err)
		}
		if preview.Ready || len(preview.ExternalListings) == 0 {
			t.Fatalf("external listing was accepted: %+v", preview)
		}
		row := preview.FactFingerprints[0]
		updated := row.Snapshot
		updated.Dst, updated.Value = "", "FAILED"
		req := EntityRetirementRequest{
			Entity: "failed", Why: "status",
			Replacements:  []EntityRetirementReplacement{{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: updated, Why: "attribute"}},
			RehomeAliases: []EntityRetirementAliasRehome{{Alias: "FAILED", Entity: "app", Why: "reviewed legacy spelling belongs to App"}},
		}
		preview, err = st.PreviewEntityRetirement(req)
		if err != nil || !preview.Ready {
			t.Fatalf("explicit rehome was refused: preview=%+v err=%v", preview, err)
		}
		req.Expected = preview.Expected
		if _, err := st.RetireEntity(req); err != nil {
			t.Fatal(err)
		}
		if owner, found, err := st.ResolveAlias("FAILED"); err != nil || !found || owner != "app" {
			t.Fatalf("reviewed spelling was not rehomed: owner=%q found=%v err=%v", owner, found, err)
		}
	})
}
