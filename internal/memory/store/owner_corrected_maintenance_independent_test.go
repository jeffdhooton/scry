package store

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"
)

type ownerIOProbe struct {
	bytes.Buffer
	writes, reads, syncs, closes int
}

func (p *ownerIOProbe) Write(b []byte) (int, error) { p.writes++; return p.Buffer.Write(b) }
func (p *ownerIOProbe) Read(b []byte) (int, error)  { p.reads++; return p.Buffer.Read(b) }
func (p *ownerIOProbe) Sync() error                 { p.syncs++; return nil }
func (p *ownerIOProbe) Close() error                { p.closes++; return nil }

type ownerMaintenanceFixture struct {
	s      *Store
	merge  EntityMergeRequest
	retire EntityRetirementRequest
	alias  AliasRepairRequest
	backup []byte
}

func ownerValidMaintenance(t *testing.T) ownerMaintenanceFixture {
	t.Helper()
	s := openTemp(t)
	for _, e := range []Entity{
		{Slug: "survivor", Name: "Survivor", Type: "tool"},
		{Slug: "loser", Name: "Loser", Type: "tool"},
		{Slug: "project", Name: "Project", Type: "project", Aliases: []string{"temporary-surface"}},
		{Slug: "state", Name: "State", Type: "concept"},
	} {
		if err := s.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []Fact{
		{Src: "loser", Relation: "status", Value: "ready", Fact: "The tool is ready", ValidFrom: time.Unix(99, 0).UTC()},
		{Src: "project", Relation: "status", Dst: "state", Fact: "The project has state", ValidFrom: time.Unix(99, 1).UTC()},
	} {
		if err := s.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	m := EntityMergeRequest{ID: "independent-merge", Survivor: "survivor", Retire: []string{"loser"}, Why: "Reviewed synthetic same identity"}
	mp, err := s.PreviewEntityMerge(m)
	if err != nil || !mp.Ready {
		t.Fatalf("merge preview %+v %v", mp, err)
	}
	m.Expected, m.Metadata = mp.Expected, &mp.ProposedMetadata
	r := EntityRetirementRequest{ID: "independent-retire", Entity: "state", Why: "Reviewed synthetic status disposition"}
	rp, err := s.PreviewEntityRetirement(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rp.FactFingerprints {
		f := row.Snapshot
		f.Dst, f.Value = "", "state"
		r.Replacements = append(r.Replacements, EntityRetirementReplacement{OldKey: row.Key, ExpectedSHA256: row.SHA256, Replacement: f, Why: "Reviewed exact endpoint conversion"})
	}
	rp, err = s.PreviewEntityRetirement(r)
	if err != nil || !rp.Ready {
		t.Fatalf("retire preview %+v %v", rp, err)
	}
	r.Expected = rp.Expected
	a := AliasRepairRequest{Drops: []AliasDrop{{Entity: "project", Alias: "temporary-surface", Why: "Reviewed generic surface drop"}}}
	ap, err := s.PreviewAliasRepair(a)
	if err != nil || !ap.Ready {
		t.Fatalf("alias preview %+v %v", ap, err)
	}
	a.Expected = &ap.Expected
	var backup bytes.Buffer
	if n, err := s.Backup(&backup); err != nil || n <= headerOnlyBackup {
		t.Fatalf("backup bytes=%d err=%v", n, err)
	}
	return ownerMaintenanceFixture{s: s, merge: m, retire: r, alias: a, backup: backup.Bytes()}
}

var ownerPublicMaintenance = []string{"merge", "merge-checked", "retire", "retire-checked", "retire-many", "backup-retire", "backup-alias", "backup", "restore", "close"}

func ownerInvokeMaintenance(tx *Store, f ownerMaintenanceFixture, op string, writer, reader *ownerIOProbe, checks *int) error {
	check := func([]Entity, []Fact) error { *checks++; return nil }
	switch op {
	case "merge":
		_, err := tx.MergeEntities(f.merge)
		return err
	case "merge-checked":
		_, err := tx.MergeEntitiesChecked(f.merge, check)
		return err
	case "retire":
		_, err := tx.RetireEntity(f.retire)
		return err
	case "retire-checked":
		_, err := tx.RetireEntityChecked(f.retire, check)
		return err
	case "retire-many":
		_, err := tx.RetireEntitiesChecked([]EntityRetirementRequest{f.retire}, check)
		return err
	case "backup-retire":
		_, _, err := tx.BackupAndRetireEntities(writer, []EntityRetirementRequest{f.retire})
		return err
	case "backup-alias":
		_, _, err := tx.BackupAndRepairAliases(writer, f.alias)
		return err
	case "backup":
		_, err := tx.Backup(writer)
		return err
	case "restore":
		return tx.Restore(reader)
	case "close":
		return tx.Close()
	}
	panic("unknown operation")
}

func TestOwnerCorrectedValidMaintenanceRefusal(t *testing.T) {
	for _, phase := range []string{"body", "finalizer", "closed-success", "closed-failure"} {
		for _, op := range ownerPublicMaintenance {
			t.Run(phase+"/"+op, func(t *testing.T) {
				f := ownerValidMaintenance(t)
				before := ownerRaw(t, f.s)
				writer, reader := &ownerIOProbe{}, &ownerIOProbe{}
				reader.Buffer.Write(f.backup)
				events, checks, finals := 0, 0, 0
				f.s.SetObserver(func(Event) { events++ })
				var saved *Store
				priorFailure := errors.New("deliberate prior failure")
				invoke := func(tx *Store) {
					wantError := errIdentityAdmissionScope
					if phase == "closed-failure" {
						wantError = priorFailure
					}
					if err := ownerInvokeMaintenance(tx, f, op, writer, reader, &checks); !errors.Is(err, wantError) {
						t.Fatalf("required owner refusal, got %v", err)
					}
				}
				err := runIdentityAdmission(f.s, func(tx *Store) error {
					saved = tx
					if phase != "closed-success" {
						if err := tx.PutEpisode(Episode{ID: "staged-before-refusal"}); err != nil {
							return err
						}
					}
					if phase == "body" {
						invoke(tx)
					}
					if phase == "closed-failure" {
						return priorFailure
					}
					return nil
				}, func(tx *Store) error {
					finals++
					if phase == "finalizer" {
						invoke(tx)
					}
					return nil
				})
				if phase == "closed-success" {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil {
					t.Fatal("refused/failed owner committed")
				}
				if phase == "closed-success" || phase == "closed-failure" {
					invoke(saved)
				}
				wantFinals := 0
				if phase == "finalizer" || phase == "closed-success" {
					wantFinals = 1
				}
				if finals != wantFinals {
					t.Fatalf("finals %d want %d", finals, wantFinals)
				}
				if checks != 0 || events != 0 || writer.writes != 0 || writer.syncs != 0 || writer.closes != 0 || reader.reads != 0 {
					t.Fatal("refused maintenance executed callback, I/O or events")
				}
				if !reflect.DeepEqual(before, ownerRaw(t, f.s)) {
					t.Fatal("refused maintenance changed raw store")
				}
				ownerAssertRefused(t, saved)
			})
		}
	}
}

func TestOwnerCorrectedRootMaintenancePositiveControl(t *testing.T) {
	for _, op := range ownerPublicMaintenance {
		t.Run(op, func(t *testing.T) {
			f := ownerValidMaintenance(t)
			before := ownerRaw(t, f.s)
			writer, reader := &ownerIOProbe{}, &ownerIOProbe{}
			reader.Buffer.Write(f.backup)
			checks := 0
			if op == "restore" {
				if err := f.s.PutEpisode(Episode{ID: "extra-before-restore"}); err != nil {
					t.Fatal(err)
				}
			}
			if err := ownerInvokeMaintenance(f.s, f, op, writer, reader, &checks); err != nil {
				t.Fatal(err)
			}
			if op == "close" {
				if _, err := f.s.GetEntity("survivor"); err == nil {
					t.Fatal("root Close had no effect")
				}
				return
			}
			if op == "backup" || op == "restore" {
				if !reflect.DeepEqual(before, ownerRaw(t, f.s)) {
					t.Fatal("root backup/restore raw equality failed")
				}
			} else if reflect.DeepEqual(before, ownerRaw(t, f.s)) {
				t.Fatal("valid root maintenance had no effect")
			}
			if op == "backup" || op == "backup-retire" || op == "backup-alias" {
				if writer.writes == 0 || writer.Len() <= headerOnlyBackup {
					t.Fatal("root backup missing")
				}
			}
			if op == "backup-retire" || op == "backup-alias" {
				if writer.syncs != 1 || writer.closes != 1 {
					t.Fatal("root durable backup protocol changed")
				}
			}
			if op == "restore" && reader.reads == 0 {
				t.Fatal("root Restore did not read backup")
			}
			if op == "merge-checked" || op == "retire-checked" || op == "retire-many" {
				if checks != 1 {
					t.Fatal("root check did not run once")
				}
			}
		})
	}
}
