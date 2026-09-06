package store

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestBirthRegistrationActualCapacityFailure(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if err != nil {
		t.Fatal("synthetic db open failed")
	}
	st := &Store{db: db}
	defer st.Close()
	input, key, raw := birthFixture(t)
	okey, oraw := birthObservation(t, input, "declaration", 0, "")
	before := dpRows(t, st)
	events, staged := 0, 0
	st.SetObserver(func(Event) { events++ })
	report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
		d, err := r.register(okey, oraw)
		if err != nil {
			return err
		}
		k, v := birthEntity(t, d.Handle)
		if err := r.putIdentity("atlas", k, v); err != nil {
			return err
		}
		if err := r.st.PutEpisode(Episode{ID: "must-rollback"}); err != nil {
			return err
		}
		for i := 0; i < 20000; i++ {
			if err := r.putIdentity("atlas", []byte(fmt.Sprintf("al:synthetic-capacity-%06d", i)), []byte("atlas")); err != nil {
				if !errors.Is(err, badger.ErrTxnTooBig) {
					t.Fatal("unexpected storage failure category")
				}
				return nil // ignored actual storage error must poison the owner
			}
			staged++
		}
		t.Fatal("fixture did not reach transaction capacity")
		return nil
	})
	if staged == 0 || !errors.Is(err, badger.ErrTxnTooBig) || !reflect.DeepEqual(report, birthRegistryReport{}) || !reflect.DeepEqual(before, dpRows(t, st)) || events != 0 {
		t.Fatal("actual capacity failure leaked state or report")
	}
	if _, err := runBirthRegistration(st, key, raw, func(*birthRegistry) error { return nil }); err != nil {
		t.Fatal("capacity failure retained lock")
	}
}

func TestBirthRegistrationFactAccountingReopenAndReplayLimit(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { st.Close() }()
	input, key, raw := birthFixture(t)
	okey, oraw := birthObservation(t, input, "declaration", 0, "")
	var saved *birthRegistry
	f := Fact{Src: "atlas", Relation: "status", Value: "active", Fact: "synthetic assertion", ValidFrom: input.OccurredAt, Confidence: 0.5, Episodes: []string{input.EpisodeID}}
	fkey, fraw := factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom), dpJSON(t, f)
	report, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
		saved = r
		d, err := r.register(okey, oraw)
		if err != nil {
			return err
		}
		k, v := birthEntity(t, d.Handle)
		if err := r.putIdentity("atlas", k, v); err != nil {
			return err
		}
		// Ordinal labels remain descriptive. This narrow unit is not an
		// assertion-authorization check, even for an out-of-input ordinal.
		if err := r.putFact(999, fkey, fraw); err != nil {
			return err
		}
		return r.st.PutEpisode(Episode{ID: input.EpisodeID, OccurredAt: input.OccurredAt})
	})
	if err != nil || len(report.Facts.Mutations) != 1 || report.Facts.Mutations[0].Ordinal != 999 || !report.Candidates[0].Created {
		t.Fatal("mechanical write accounting failed", err)
	}
	before := dpRows(t, st)
	for i := range report.Facts.Mutations[0].After.Raw {
		report.Facts.Mutations[0].After.Raw[i] = 0
	}
	if !bytes.Equal(saved.facts.mutations[0].After.Raw, fraw) || !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("fact report shares mutable storage")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("reopen changed committed raw rows")
	}
	replay, err := runBirthRegistration(st, key, raw, func(r *birthRegistry) error {
		d, err := r.register(okey, oraw)
		if err != nil || d.Kind != "existing-not-new" || d.Handle != nil {
			t.Fatal("replay invented new registration")
		}
		return nil
	})
	if err != nil || len(replay.Candidates) != 0 || len(replay.Dispositions) != 1 || !reflect.DeepEqual(before, dpRows(t, st)) {
		t.Fatal("replay accounting failed")
	}
	// The first report's birth is NOT automatically the current Force
	// result. Selection/retention must be implemented separately.
}

func TestBirthRegistrationObserverPanicIsPostCommit(t *testing.T) {
	st := openTemp(t)
	_, key, raw := birthFixture(t)
	var report birthRegistryReport
	var caught any
	st.SetObserver(func(Event) { panic("synthetic observer panic") })
	func() {
		defer func() { caught = recover() }()
		report, _ = runBirthRegistration(st, key, raw, func(r *birthRegistry) error { return r.st.PutEpisode(Episode{ID: "committed"}) })
	}()
	if caught != "synthetic observer panic" || !reflect.DeepEqual(report, birthRegistryReport{}) || dpRows(t, st)["ep:committed"] == nil {
		t.Fatal("observer panic changed established postcommit semantics")
	}
	st.SetObserver(nil)
	if _, err := runBirthRegistration(st, key, raw, func(*birthRegistry) error { return nil }); err != nil {
		t.Fatal("observer panic retained lock")
	}
}
