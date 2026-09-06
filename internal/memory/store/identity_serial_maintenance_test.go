package store

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestSerializedAdmissionRealMaintenanceForward(t *testing.T) {
	for _, phase := range []string{"body", "finalizer"} {
		for _, op := range []string{"merge", "merge-checked", "retire", "retire-checked", "retire-many", "backup-retire", "backup-alias", "restore", "adoption-preview", "adoption-apply"} {
			t.Run(phase+"/"+op, func(t *testing.T) {
				f := ownerValidMaintenance(t)
				var manifest []byte
				if op == "adoption-preview" || op == "adoption-apply" {
					// Adoption requires strict canonical timestamps. Its own
					// fixture has intentionally opaque unrelated preserved rows.
					f.s = openTemp(t)
					_, manifest = adoptionFixture(t, f.s)
				}
				before := dpRows(t, f.s)
				release, admission := serialHold(t, f.s, phase, nil)
				writer, reader := &ownerIOProbe{}, &ownerIOProbe{}
				reader.Buffer.Write(f.backup)
				checks := 0
				started := make(chan struct{})
				maintenance := serialAsync(func() error {
					close(started)
					if manifest != nil {
						_, err := adoptLegacyInventory(f.s, manifest, op == "adoption-apply")
						return err
					}
					return ownerInvokeMaintenance(f.s, f, op, writer, reader, &checks)
				})
				serialSignal(t, started)
				serialBlocked(t, maintenance)
				if !reflect.DeepEqual(before, dpRows(t, f.s)) {
					t.Fatal("maintenance crossed held scope")
				}
				release()
				if admission.wait(t) != nil || maintenance.wait(t) != nil {
					t.Fatal("valid maintenance failed after release")
				}
				after := dpRows(t, f.s)
				if op == "restore" || op == "adoption-preview" {
					if !reflect.DeepEqual(before, after) {
						t.Fatal("restored/preview bytes changed")
					}
				} else if reflect.DeepEqual(before, after) {
					t.Fatal("maintenance had no actual effect")
				}
			})
		}
	}
}

type serialBlockingReader struct {
	reader  io.Reader
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *serialBlockingReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.entered); <-r.release })
	return r.reader.Read(p)
}

func TestSerializedAdmissionActualMaintenanceReverseAndRetirement(t *testing.T) {
	for _, op := range []string{"merge", "retire", "backup-retire", "backup-alias", "restore", "adoption-lock-composition"} {
		t.Run(op, func(t *testing.T) {
			f := ownerValidMaintenance(t)
			entered, gate := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(gate) }) }
			t.Cleanup(release)
			post := func([]Entity, []Fact) error { close(entered); <-gate; return nil }
			maintenance := serialAsync(func() error {
				switch op {
				case "merge":
					_, err := f.s.MergeEntitiesChecked(f.merge, post)
					return err
				case "retire":
					_, err := f.s.RetireEntityChecked(f.retire, post)
					return err
				case "backup-retire":
					w := &blockingDurableBackup{syncStarted: entered, releaseSync: gate}
					_, _, err := f.s.BackupAndRetireEntities(w, []EntityRetirementRequest{f.retire})
					return err
				case "backup-alias":
					w := &blockingDurableBackup{syncStarted: entered, releaseSync: gate}
					_, _, err := f.s.BackupAndRepairAliases(w, f.alias)
					return err
				case "restore":
					return f.s.Restore(&serialBlockingReader{reader: bytes.NewReader(f.backup), entered: entered, release: gate})
				case "adoption-lock-composition":
					// Compositional reverse proof, not an actual paused adopter.
					// Forward tests above execute real preview/apply; source audit
					// verifies that those acquire this exact lock before their txn.
					f.s.maintenanceMu.Lock()
					defer f.s.maintenanceMu.Unlock()
					close(entered)
					<-gate
					return nil
				}
				return errors.New("missing maintenance operation")
			})
			serialSignal(t, entered)
			bodyEntered := make(chan struct{})
			admission := serialAsync(func() error {
				return runSerializedIdentityAdmission(f.s, func(*Store) error { close(bodyEntered); return nil }, func(*Store) error { return nil })
			})
			serialBlocked(t, admission)
			select {
			case <-bodyEntered:
				t.Fatal("admission entered during maintenance")
			default:
			}
			release()
			if maintenance.wait(t) != nil {
				t.Fatal("actual maintenance failed")
			}
			err := admission.wait(t)
			if op == "retire" || op == "backup-retire" {
				if !errors.Is(err, ErrNotFound) {
					t.Fatal("waiting admission lost retirement refusal")
				}
				select {
				case <-bodyEntered:
					t.Fatal("retired snapshot entered callback")
				default:
				}
			} else if err != nil {
				t.Fatal("admission did not resume")
			}
			if serialAsync(func() error { return f.s.PutPending(PendingEpisode{ID: "after"}) }).wait(t) != nil {
				t.Fatal("lock not released")
			}
		})
	}
}

func TestSerializedAdmissionRealCapacityFailureAndObserverPanic(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if err != nil {
		t.Fatal("synthetic db open failed")
	}
	st := &Store{db: db}
	defer st.Close()
	before := dpRows(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	err = runSerializedIdentityAdmission(st, func(tx *Store) error {
		if err := tx.PutEpisode(Episode{ID: "staged"}); err != nil {
			return err
		}
		for i := 0; i < 20000; i++ {
			key := []byte{'x', byte(i >> 8), byte(i)}
			if err := tx.txn.Set(key, bytes.Repeat([]byte{'x'}, 1000)); err != nil {
				return err
			}
		}
		return errors.New("capacity fixture did not reach actual failure")
	}, func(*Store) error { return errors.New("unexpected finalizer") })
	if !errors.Is(err, badger.ErrTxnTooBig) || events != 0 || !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("actual capacity rollback failed")
	}
	if serialAsync(func() error { return st.PutPending(PendingEpisode{ID: "after"}) }).wait(t) != nil {
		t.Fatal("queue lock leaked after capacity failure")
	}
	if serialAsync(func() error { return st.ClaimAlias("after", "owner") }).wait(t) != nil {
		t.Fatal("graph lock leaked after capacity failure")
	}
	st.SetObserver(func(Event) { panic("postcommit synthetic") })
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = runSerializedIdentityAdmission(st, func(tx *Store) error { return tx.PutEpisode(Episode{ID: "committed"}) }, func(*Store) error { return nil })
	}()
	if !panicked || dpRows(t, st)["ep:committed"] == nil {
		t.Fatal("observer panic changed commit semantics")
	}
	if serialAsync(func() error { return st.ClaimAlias("observer-after", "owner") }).wait(t) != nil {
		t.Fatal("observer panic leaked graph lock")
	}
}
