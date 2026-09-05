package store

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type aliasBackup struct {
	bytes.Buffer
	fail           string
	synced, closed bool
}

func (w *aliasBackup) Write(p []byte) (int, error) {
	if w.fail == "write" {
		return 0, errors.New("write failed")
	}
	return w.Buffer.Write(p)
}
func (w *aliasBackup) Sync() error {
	w.synced = true
	if w.fail == "sync" {
		return errors.New("sync failed")
	}
	return nil
}
func (w *aliasBackup) Close() error {
	w.closed = true
	if w.fail == "close" {
		return errors.New("close failed")
	}
	return nil
}

func aliasFixture(t *testing.T) (*Store, AliasRepairRequest) {
	t.Helper()
	s := openTemp(t)
	for _, e := range []Entity{
		{Slug: "app", Name: "App", Type: "project", Aliases: []string{"hosting tool", "generic frontend", "Generic Frontend", "foreign name"}, Description: "do not change", RepoRefs: []string{"/repo"}},
		{Slug: "host", Name: "Host", Type: "tool"},
		{Slug: "other", Name: "Other", Type: "tool"},
	} {
		if err := s.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	// Manufacture reviewed legacy disagreement without automatic inference.
	for _, alias := range []string{"hosting tool", "foreign name"} {
		if err := s.ClaimAlias(alias, "host"); err != nil {
			t.Fatal(err)
		}
	}
	host, _ := s.GetEntity("host")
	host.Aliases = []string{"hosting tool", "foreign name"}
	if err := s.PutEntity(host); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimAlias("hosting tool", "app"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimAlias("foreign name", "host"); err != nil {
		t.Fatal(err)
	}
	at := time.Unix(10, 0).UTC()
	invalid := at.Add(time.Hour)
	for _, f := range []Fact{
		{Src: "app", Relation: "uses", Dst: "host", Fact: "App uses Host", ValidFrom: at, Confidence: .91, Episodes: []string{"e1"}},
		{Src: "host", Relation: "related_to", Dst: "app", Fact: "Historical host fact", RawRelation: "was_host", ValidFrom: at, InvalidAt: &invalid, Confidence: .81, Episodes: []string{"e2"}},
	} {
		if err := s.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	req := AliasRepairRequest{Drops: []AliasDrop{
		{Entity: "app", Alias: "hosting tool", RehomeTo: "host", Why: "Existing distinct host owns this tool identity"},
		{Entity: "app", Alias: "generic frontend", Why: "Generic role, including case variant in preview"},
		{Entity: "app", Alias: "foreign name", Why: "Remove listing, preserve existing Host index"},
	}}
	p, err := s.PreviewAliasRepair(req)
	if err != nil || !p.Ready {
		t.Fatalf("preview %+v %v", p, err)
	}
	req.Expected = &p.Expected
	return s, req
}

func aliasState(t *testing.T, s *Store) string {
	t.Helper()
	es, e := s.Entities()
	if e != nil {
		t.Fatal(e)
	}
	fs, e := s.AllFacts()
	if e != nil {
		t.Fatal(e)
	}
	cs, e := s.AliasClaims()
	if e != nil {
		t.Fatal(e)
	}
	var rs map[string][]AliasRejection
	if err := s.view(func(txn *badger.Txn) error { var err error; rs, err = aliasRejectionsTxn(txn); return err }); err != nil {
		t.Fatal(err)
	}
	return hashJSON([]any{es, fs, cs, rs})
}

func TestAliasRepairAtomicPreservationAndBackup(t *testing.T) {
	s, req := aliasFixture(t)
	before := aliasState(t, s)
	facts, _ := s.AllFacts()
	backup := &aliasBackup{}
	events := 0
	s.SetObserver(func(e Event) {
		if e.Kind != "entity" || e.Op != "put" || e.Entity.Slug != "app" {
			t.Errorf("unexpected event %+v", e)
		}
		if !backup.synced || !backup.closed {
			t.Error("observer before backup close")
		}
		if err := s.AtomicWrite(func(*Store) error { return nil }); err != nil {
			t.Error(err)
		} // callback outside maintenance lock
		events++
	})
	n, p, err := s.BackupAndRepairAliases(backup, req)
	if err != nil || !p.Applied || n == 0 || events != 1 {
		t.Fatalf("apply %+v %v bytes=%d events=%d", p, err, n, events)
	}
	afterFacts, _ := s.AllFacts()
	if !reflect.DeepEqual(facts, afterFacts) {
		t.Fatal("fact metadata changed")
	}
	for _, alias := range []string{"hosting tool", "foreign name"} {
		owner, ok, e := s.ResolveAlias(alias)
		if e != nil || !ok || owner != "host" {
			t.Fatalf("wrong owner %s=%s", alias, owner)
		}
	}
	if _, ok, _ := s.ResolveAlias("generic frontend"); ok {
		t.Fatal("dropped alias still indexed")
	}
	app, _ := s.GetEntity("app")
	if len(app.Aliases) != 0 || app.Description != "do not change" || len(app.RepoRefs) != 1 {
		t.Fatalf("bad metadata %+v", app)
	}
	restored := openTemp(t)
	if err := restored.Restore(bytes.NewReader(backup.Bytes())); err != nil {
		t.Fatal(err)
	}
	if aliasState(t, restored) != before {
		t.Fatal("backup does not restore exact original state")
	}
	p, err = s.PreviewAliasRepair(req)
	if err != nil || p.Ready {
		t.Fatalf("repeat should refuse unchanged absent inputs: %+v %v", p, err)
	}
}

func TestAliasRepairDriftAndBackupFailureNeverWrite(t *testing.T) {
	for _, mode := range []string{"entity", "fact", "claim-only", "outside-listing", "plan", "missing-expected", "write", "sync", "close"} {
		t.Run(mode, func(t *testing.T) {
			s, req := aliasFixture(t)
			switch mode {
			case "entity":
				e, _ := s.GetEntity("host")
				e.Description = "changed"
				if err := s.PutEntity(e); err != nil {
					t.Fatal(err)
				}
			case "fact":
				if err := s.PutFact(Fact{Src: "host", Relation: "status", Value: "changed", Fact: "New review evidence", ValidFrom: time.Unix(30, 0).UTC()}); err != nil {
					t.Fatal(err)
				}
			case "claim-only":
				if err := s.ClaimAlias("hosting tool", "other"); err != nil {
					t.Fatal(err)
				}
			case "outside-listing":
				if err := s.ClaimAlias("generic frontend", "other"); err != nil {
					t.Fatal(err)
				}
				e, _ := s.GetEntity("other")
				e.Aliases = []string{"generic frontend"}
				if err := s.PutEntity(e); err != nil {
					t.Fatal(err)
				}
				if err := s.ClaimAlias("generic frontend", "app"); err != nil {
					t.Fatal(err)
				}
			case "plan":
				req.Drops[0].Why = "Changed after approval"
			case "missing-expected":
				req.Expected = nil
			}
			before := aliasState(t, s)
			backup := &aliasBackup{fail: mode}
			events := 0
			s.SetObserver(func(Event) { events++ })
			_, p, err := s.BackupAndRepairAliases(backup, req)
			if err == nil || p.Applied || events != 0 {
				t.Fatalf("unsafe success %+v %v events=%d", p, err, events)
			}
			if aliasState(t, s) != before {
				t.Fatal("refused batch changed durable state")
			}
		})
	}
}

func TestAliasRepairGlobalDropRequiresEveryListing(t *testing.T) {
	s, req := aliasFixture(t)
	req = AliasRepairRequest{Drops: []AliasDrop{{Entity: "app", Alias: "hosting tool", Why: "Explicit global drop"}}}
	p, err := s.PreviewAliasRepair(req)
	if err != nil || p.Ready {
		t.Fatalf("stranded listing accepted: %+v %v", p, err)
	}
	req.Drops = append(req.Drops, AliasDrop{Entity: "host", Alias: "hosting tool", Why: "Explicit second listing drop"})
	p, err = s.PreviewAliasRepair(req)
	if err != nil || !p.Ready {
		t.Fatalf("complete drop refused: %+v %v", p, err)
	}
	req.Expected = &p.Expected
	_, p, err = s.BackupAndRepairAliases(&aliasBackup{}, req)
	if err != nil || !p.Applied {
		t.Fatalf("apply: %+v %v", p, err)
	}
	if _, ok, _ := s.ResolveAlias("hosting tool"); ok {
		t.Fatal("global drop retained index")
	}
}

func TestAliasRepairExcludesConcurrentWriterAcrossBackup(t *testing.T) {
	s, req := aliasFixture(t)
	w := &blockingDurableBackup{syncStarted: make(chan struct{}), releaseSync: make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, _, err := s.BackupAndRepairAliases(w, req); done <- err }()
	<-w.syncStarted
	wrote := make(chan error, 1)
	go func() { wrote <- s.PutEntity(Entity{Slug: "later", Name: "Later", Type: "project"}) }()
	select {
	case err := <-wrote:
		t.Fatalf("writer crossed backup boundary: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(w.releaseSync)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-wrote; err != nil {
		t.Fatal(err)
	}
}
