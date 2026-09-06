package store

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestIdentityJournalExactBeforeImagesAndEvents(t *testing.T) {
	st := openTemp(t)
	claim := []byte(prefixAlias + "aurora")
	if err := st.db.Update(func(tx *badger.Txn) error { return tx.Set(claim, []byte("aurora")) }); err != nil {
		t.Fatal(err)
	}
	var events []Event
	st.SetObserver(func(e Event) { events = append(events, e) })
	if err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		keys := [][]byte{[]byte(prefixEntity + "aurora"), claim, []byte(prefixAttest + "aurora:polar-manual")}
		for _, key := range keys {
			if err := j.capture(key); err != nil {
				return err
			}
		}
		if err := tx.PutEntity(Entity{Slug: "aurora", Name: "Aurora", Type: "runbook"}); err != nil {
			return err
		}
		if _, err := tx.AttestAlias("aurora", "polar-manual", "synthetic-episode"); err != nil {
			return err
		}
		if err := tx.PutEntity(Entity{Slug: "borealis", Name: "Borealis", Type: "service"}); err != nil {
			return err
		}
		if err := tx.PutEpisode(Episode{ID: "synthetic-episode", OccurredAt: time.Unix(1, 0)}); err != nil {
			return err
		}
		undo := make([]identityUndo, 0, len(keys))
		for _, key := range keys {
			expected, err := j.image(key)
			if err != nil {
				return err
			}
			undo = append(undo, identityUndo{Key: key, Expected: expected})
			// Recapture must not replace the original absent/before state.
			if err := j.capture(key); err != nil {
				return err
			}
		}
		if err := j.restore(undo); err != nil {
			return err
		}
		return j.suppressAbsentEntityEvents([]string{"aurora"})
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetEntity("aurora"); !errors.Is(err, ErrNotFound) {
		t.Fatal("provisional entity survived")
	}
	if _, err := st.GetEntity("borealis"); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Kind != "entity" || events[0].Entity.Slug != "borealis" || events[1].Kind != "episode" {
		t.Fatal("uncommitted entity events escaped or unrelated events disappeared")
	}
	if err := st.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(claim)
		if err != nil {
			return err
		}
		value, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		if !bytes.Equal(value, []byte("aurora")) {
			t.Fatal("preexisting orphan claim changed")
		}
		_, err = tx.Get([]byte(prefixAttest + "aurora:polar-manual"))
		if !errors.Is(err, badger.ErrKeyNotFound) {
			t.Fatal("new provisional evidence survived at its routing key")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityJournalRawUnknownBytesAndPreflight(t *testing.T) {
	st := openTemp(t)
	a, b := []byte("en:opaque"), []byte("att:opaque:alias")
	oldA := []byte(" {\"unknown\":1,\"unknown\":2} \n")
	oldB := []byte(" [\"first\", \"second\"] \n")
	if err := st.db.Update(func(tx *badger.Txn) error {
		if err := tx.Set(a, oldA); err != nil {
			return err
		}
		return tx.Set(b, oldB)
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		for _, key := range [][]byte{a, b} {
			if err := j.capture(key); err != nil {
				return err
			}
		}
		for _, key := range [][]byte{a, b} {
			if err := tx.txn.Set(key, []byte("replacement")); err != nil {
				return err
			}
		}
		good := identityImage{Exists: true, Value: []byte("replacement")}
		if err := j.restore([]identityUndo{{Key: a, Expected: good}, {Key: b, Expected: identityImage{}}}); !errors.Is(err, errIdentityJournal) {
			t.Fatal("stale expectation accepted")
		}
		current, err := j.image(a)
		if err != nil {
			return err
		}
		if !bytes.Equal(current.Value, good.Value) {
			t.Fatal("failed complete preflight partially restored first key")
		}
		if err := j.restore([]identityUndo{{Key: a, Expected: good}, {Key: a, Expected: good}}); !errors.Is(err, errIdentityJournal) {
			t.Fatal("duplicate undo accepted")
		}
		return j.restore([]identityUndo{{Key: a, Expected: good}, {Key: b, Expected: good}})
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		for _, pair := range []struct{ k, v []byte }{{a, oldA}, {b, oldB}} {
			got, err := j.image(pair.k)
			if err != nil {
				return err
			}
			if !got.Exists || !bytes.Equal(got.Value, pair.v) {
				t.Fatal("opaque bytes changed")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityJournalScopeLifetimeAndRollback(t *testing.T) {
	st := openTemp(t)
	if _, err := newIdentityJournal(st); !errors.Is(err, errIdentityJournal) {
		t.Fatal("journal outside atomic write accepted")
	}
	var saved *identityJournal
	var events int
	st.SetObserver(func(Event) { events++ })
	failure := errors.New("synthetic final failure")
	err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		saved = j
		for _, key := range [][]byte{nil, []byte("en:"), []byte("fa:forbidden"), []byte("ep:forbidden")} {
			if err := j.capture(key); !errors.Is(err, errIdentityJournal) {
				t.Fatal("out of scope key accepted")
			}
		}
		if err := j.capture([]byte("en:aurora")); err != nil {
			return err
		}
		if err := tx.PutEntity(Entity{Slug: "aurora", Name: "Aurora", Type: "runbook"}); err != nil {
			return err
		}
		if err := j.suppressAbsentEntityEvents([]string{"aurora"}); !errors.Is(err, errIdentityJournal) {
			t.Fatal("present entity event suppression accepted")
		}
		if err := j.restore([]identityUndo{{Key: []byte("al:uncaptured")}}); !errors.Is(err, errIdentityJournal) {
			t.Fatal("uncaptured undo accepted")
		}
		return failure
	})
	if !errors.Is(err, failure) || events != 0 {
		t.Fatal("failed transaction leaked events")
	}
	if _, err := st.GetEntity("aurora"); !errors.Is(err, ErrNotFound) {
		t.Fatal("failed transaction leaked entity")
	}
	if err := saved.capture([]byte("en:aurora")); err == nil {
		t.Fatal("expired capture accepted")
	}
	if err := saved.restore(nil); err == nil {
		t.Fatal("expired restore accepted")
	}
	if err := saved.suppressAbsentEntityEvents(nil); err == nil {
		t.Fatal("expired event suppression accepted")
	}
}
