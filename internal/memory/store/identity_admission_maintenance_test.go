package store

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

type admissionUnusedBackup struct{ calls int }

func (w *admissionUnusedBackup) Write(p []byte) (int, error) { w.calls++; return len(p), nil }
func (w *admissionUnusedBackup) Sync() error                 { w.calls++; return nil }
func (w *admissionUnusedBackup) Close() error                { w.calls++; return nil }

func TestAdmissionMaintenanceRefusesEveryOwnedPhase(t *testing.T) {
	for _, phase := range []string{"body", "finalizing", "closed"} {
		for _, operation := range []string{"merge", "merge-checked", "retire", "retire-checked", "retire-many", "backup-retire", "backup-alias", "backup", "restore", "close"} {
			t.Run(phase+"/"+operation, func(t *testing.T) {
				st := openTemp(t)
				if err := st.PutEntity(Entity{Slug: "original", Name: "Original"}); err != nil {
					t.Fatal(err)
				}
				before := gradeGenerationSnapshot(t, st)
				events := 0
				st.SetObserver(func(Event) { events++ })
				writer := &admissionUnusedBackup{}
				invoke := func(tx *Store) {
					var err error
					switch operation {
					case "merge":
						_, err = tx.MergeEntities(EntityMergeRequest{})
					case "merge-checked":
						_, err = tx.MergeEntitiesChecked(EntityMergeRequest{}, nil)
					case "retire":
						_, err = tx.RetireEntity(EntityRetirementRequest{})
					case "retire-checked":
						_, err = tx.RetireEntityChecked(EntityRetirementRequest{}, nil)
					case "retire-many":
						_, err = tx.RetireEntitiesChecked(nil, nil)
					case "backup-retire":
						_, _, err = tx.BackupAndRetireEntities(writer, nil)
					case "backup-alias":
						_, _, err = tx.BackupAndRepairAliases(writer, AliasRepairRequest{})
					case "backup":
						_, err = tx.Backup(writer)
					case "restore":
						err = tx.Restore(bytes.NewReader(nil))
					case "close":
						err = tx.Close()
					}
					if !errors.Is(err, errIdentityAdmissionScope) {
						t.Fatal("maintenance not refused at owner boundary", err)
					}
				}
				var saved *Store
				err := runIdentityAdmission(st, func(tx *Store) error {
					saved = tx
					if phase == "body" {
						invoke(tx)
					}
					return nil
				}, func(tx *Store) error {
					if phase == "finalizing" {
						invoke(tx)
					}
					return nil
				})
				if phase == "closed" {
					if err != nil {
						t.Fatal(err)
					}
					invoke(saved)
				} else if !errors.Is(err, errIdentityAdmissionScope) {
					t.Fatal("ignored maintenance refusal did not abort outer scope")
				}
				if writer.calls != 0 || events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("maintenance touched bytes, writer or events")
				}
			})
		}
	}
}
