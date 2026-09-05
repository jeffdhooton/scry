package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestReviewedAliasDropRejectsStaleEntityWrite(t *testing.T) {
	for _, atomic := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "atomic"}[atomic], func(t *testing.T) {
			s, req := aliasFixture(t)
			stale, err := s.GetEntity("app")
			if err != nil {
				t.Fatal(err)
			}
			// Keep the unclaimed normalized spelling. Other fixture aliases
			// have a competing current owner, which independently blocks them.
			stale.Aliases = []string{"generic frontend", "Generic Frontend"}
			if _, _, err := s.BackupAndRepairAliases(&aliasBackup{}, req); err != nil {
				t.Fatal(err)
			}
			before := aliasState(t, s)
			if atomic {
				err = s.AtomicWrite(func(tx *Store) error {
					if err := tx.PutEntity(Entity{Slug: "must-rollback", Name: "Must Rollback"}); err != nil {
						return err
					}
					return tx.PutEntity(stale)
				})
			} else {
				err = s.PutEntity(stale)
			}
			if !errors.Is(err, ErrAliasRejected) {
				t.Fatalf("reviewed alias removal not enforced: %v", err)
			}
			if aliasState(t, s) != before {
				t.Fatal("rejected write changed state")
			}
			if _, err := s.GetEntity("must-rollback"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("atomic write escaped rollback: %v", err)
			}
		})
	}
}

func TestReviewedAliasDropRejectsExplicitClaimBackToWrongOwner(t *testing.T) {
	s, req := aliasFixture(t)
	if _, _, err := s.BackupAndRepairAliases(&aliasBackup{}, req); err != nil {
		t.Fatal(err)
	}
	before := aliasState(t, s)
	if err := s.ClaimAlias("Generic Frontend", "app"); !errors.Is(err, ErrAliasRejected) {
		t.Fatalf("claim bypassed reviewed rejection: %v", err)
	}
	if aliasState(t, s) != before {
		t.Fatal("rejected claim changed state")
	}
}

func rejectionRawState(t *testing.T, s *Store) string {
	t.Helper()
	all := map[string][]byte{}
	if err := s.view(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			v, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			all[string(it.Item().KeyCopy(nil))] = v
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return hashJSON(all)
}

func TestAliasRejectionDurabilityAndOtherIdentity(t *testing.T) {
	s, req := aliasFixture(t)
	if _, _, err := s.BackupAndRepairAliases(&aliasBackup{}, req); err != nil {
		t.Fatal(err)
	}
	var backup bytes.Buffer
	if n, err := s.Backup(&backup); err != nil || n == 0 {
		t.Fatalf("backup %d %v", n, err)
	}
	dir := filepath.Join(t.TempDir(), "restored")
	restored, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.Restore(bytes.NewReader(backup.Bytes())); err != nil {
		t.Fatal(err)
	}
	if rejectionRawState(t, restored) != rejectionRawState(t, s) {
		t.Fatal("backup lost raw state")
	}
	if err := restored.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	for _, spelling := range []string{"generic frontend", "GENERIC_FRONTEND", "generic-frontend"} {
		if rejected, err := restored.IsAliasRejected("app", spelling); err != nil || !rejected {
			t.Fatalf("lost rejection %q %v", spelling, err)
		}
		e, _ := restored.GetEntity("app")
		e.Aliases = []string{spelling}
		before := rejectionRawState(t, restored)
		if err := restored.PutEntity(e); !errors.Is(err, ErrAliasRejected) {
			t.Fatalf("stale variant %q: %v", spelling, err)
		}
		if rejectionRawState(t, restored) != before {
			t.Fatal("rejected variant wrote")
		}
	}
	// Negative ownership is not global retirement or an inferred new owner.
	if err := restored.PutEntity(Entity{Slug: "separate", Name: "Generic Frontend", Type: "tool"}); err != nil {
		t.Fatal(err)
	}
	if owner, ok, err := restored.ResolveAlias("generic frontend"); err != nil || !ok || owner != "separate" {
		t.Fatalf("independent name blocked: %s %v", owner, err)
	}
	if err := restored.ClaimAlias("generic frontend", "separate"); err != nil {
		t.Fatal(err)
	}
	app, _ := restored.GetEntity("app")
	app.Description = "safe metadata-only update"
	if err := restored.PutEntity(app); err != nil {
		t.Fatal(err)
	}
}

func TestAliasRejectionPreservesLiteralReviewEvidence(t *testing.T) {
	s, req := aliasFixture(t)
	req.Drops = append(req.Drops, AliasDrop{Entity: "app", Alias: "Generic Frontend", Why: "Explicit second literal evidence"})
	req.Expected = nil
	p, err := s.PreviewAliasRepair(req)
	if err != nil || !p.Ready {
		t.Fatalf("preview %+v %v", p, err)
	}
	req.Expected = &p.Expected
	if _, _, err := s.BackupAndRepairAliases(&aliasBackup{}, req); err != nil {
		t.Fatal(err)
	}
	if err := s.view(func(txn *badger.Txn) error {
		all, err := aliasRejectionsTxn(txn)
		if err != nil {
			return err
		}
		rows := all[aliasRejectionKey("app", Normalize("generic frontend"))]
		if len(rows) != 2 || rows[0].Why == rows[1].Why || rows[0].Plan != p.Expected.Plan || rows[1].Plan != p.Expected.Plan {
			t.Fatalf("lost literal evidence %+v", rows)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAliasRejectionDoesNotInferFromOrdinaryOmission(t *testing.T) {
	s := openTemp(t)
	e := Entity{Slug: "app", Name: "App", Aliases: []string{"Envoyer"}}
	if err := s.PutEntity(e); err != nil {
		t.Fatal(err)
	}
	e.Aliases = nil
	if err := s.PutEntity(e); err != nil {
		t.Fatal(err)
	}
	if rejected, err := s.IsAliasRejected("app", "Envoyer"); err != nil || rejected {
		t.Fatalf("inferred rejection %v %v", rejected, err)
	}
	e.Aliases = []string{"Envoyer"}
	if err := s.PutEntity(e); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DropAlias("app", "Envoyer"); err != nil {
		t.Fatal(err)
	}
	if rejected, err := s.IsAliasRejected("app", "Envoyer"); err != nil || rejected {
		t.Fatalf("low-level helper fabricated review %v %v", rejected, err)
	}
}

func TestAliasRejectionMalformedMarkerFailsClosed(t *testing.T) {
	for _, value := range []string{"not json", "null", "[]", `[{}]`} {
		t.Run(value, func(t *testing.T) {
			s := openTemp(t)
			if err := s.db.Update(func(txn *badger.Txn) error {
				return txn.Set([]byte(aliasRejectionKey("app", "envoyer")), []byte(value))
			}); err != nil {
				t.Fatal(err)
			}
			before := rejectionRawState(t, s)
			if err := s.PutEntity(Entity{Slug: "app", Name: "App", Aliases: []string{"Envoyer"}}); err == nil {
				t.Fatal("malformed marker ignored")
			}
			if _, err := s.IsAliasRejected("app", "Envoyer"); err == nil {
				t.Fatal("malformed decision admitted")
			}
			if rejectionRawState(t, s) != before {
				t.Fatal("malformed marker changed state")
			}
		})
	}
}

func TestAliasRejectionMergeCannotForgetDecision(t *testing.T) {
	s, req := aliasFixture(t)
	if _, _, err := s.BackupAndRepairAliases(&aliasBackup{}, req); err != nil {
		t.Fatal(err)
	}
	for _, survivor := range []string{"app", "other"} {
		loser := "app"
		if survivor == "app" {
			loser = "other"
		}
		p, err := s.PreviewEntityMerge(EntityMergeRequest{Survivor: survivor, Retire: []string{loser}})
		if err != nil || p.Ready || !strings.Contains(strings.Join(p.Problems, ";"), "inheritance") {
			t.Fatalf("merge can forget rejection: %+v %v", p, err)
		}
	}
}

func TestAliasRejectionMergeDropsPersistAndRollback(t *testing.T) {
	s := openTemp(t)
	for _, e := range []Entity{{Slug: "winner", Name: "Winner", Type: "project"}, {Slug: "loser", Name: "Loser", Type: "project", Aliases: []string{"Envoyer"}}} {
		if err := s.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.PutFact(Fact{Src: "loser", Relation: "status", Value: "old", Fact: "Old assertion", ValidFrom: time.Unix(1, 0).UTC()}); err != nil {
		t.Fatal(err)
	}
	req := EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}, DropAliases: []EntityMergeAliasDrop{{Alias: "Envoyer", Why: "Reviewed unrelated identity"}}}
	p, err := s.PreviewEntityMerge(req)
	if err != nil || !p.Ready {
		t.Fatalf("preview %+v %v", p, err)
	}
	req.Metadata, req.Expected = &p.ProposedMetadata, p.Expected
	before := rejectionRawState(t, s)
	if _, err := s.MergeEntitiesChecked(req, func([]Entity, []Fact) error { return errors.New("deliberate postcondition refusal") }); err == nil {
		t.Fatal("postcondition ignored")
	}
	if rejectionRawState(t, s) != before {
		t.Fatal("rejected merge leaked markers")
	}
	// Observer is outside the exclusive maintenance lock.
	s.SetObserver(func(Event) {
		if err := s.AtomicWrite(func(*Store) error { return nil }); err != nil {
			t.Error(err)
		}
	})
	if _, err := s.MergeEntities(req); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"winner", "loser"} {
		if rejected, err := s.IsAliasRejected(slug, "envoyer"); err != nil || !rejected {
			t.Fatalf("merge dropped rejection %s %v", slug, err)
		}
		if err := s.ClaimAlias("envoyer", slug); !errors.Is(err, ErrAliasRejected) {
			t.Fatalf("merge marker bypass %s %v", slug, err)
		}
	}
}

func TestAliasRejectionRehomeAndSnapshotDrift(t *testing.T) {
	s, req := aliasFixture(t)
	// Introduce a negative decision without changing the pinned entities or
	// claims: model a future reviewed marker-only backfill, not public API.
	row := AliasRejection{Entity: "host", Alias: "hosting tool", Why: "independent prior decision", Plan: "fixture-reviewed-plan"}
	value, _ := json.Marshal([]AliasRejection{row})
	if err := s.db.Update(func(txn *badger.Txn) error { return txn.Set([]byte(aliasRejectionKey("host", "hosting-tool")), value) }); err != nil {
		t.Fatal(err)
	}
	before := rejectionRawState(t, s)
	p, err := s.PreviewAliasRepair(req)
	if err != nil || p.Ready || p.Expected.Rejections == req.Expected.Rejections {
		t.Fatalf("marker drift invisible: %+v %v", p, err)
	}
	if _, _, err := s.BackupAndRepairAliases(&aliasBackup{}, req); err == nil {
		t.Fatal("stale marker snapshot applied")
	}
	if _, err := s.DropAliasRehome("app", "hosting tool", "host"); !errors.Is(err, ErrAliasRejected) {
		t.Fatalf("lower-level rehome bypass: %v", err)
	}
	if rejectionRawState(t, s) != before {
		t.Fatal("rehome refusal changed state")
	}
	// Refusal preserves every old fact and decision, not just counts.
	facts, _ := s.AllFacts()
	if _, _, err := s.BackupAndRepairAliases(&aliasBackup{fail: "close"}, req); err == nil {
		t.Fatal("failed backup accepted")
	}
	afterFacts, _ := s.AllFacts()
	if !reflect.DeepEqual(facts, afterFacts) || rejectionRawState(t, s) != before {
		t.Fatal("failed backup changed state")
	}
}
