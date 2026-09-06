package store

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestGradeJournalCaptureCopiesKeyAndImage(t *testing.T) {
	st := openTemp(t)
	old := []byte(" {\"x\":1,\"x\":2,\"unknown\":{\"future\":[null,true]}}\n")
	if err := st.db.Update(func(tx *badger.Txn) error { return tx.Set([]byte("en:opaque"), old) }); err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		key := []byte("en:opaque")
		if err := j.capture(key); err != nil {
			return err
		}
		copy(key, "fa:opaque")
		image, err := j.image([]byte("en:opaque"))
		if err != nil {
			return err
		}
		image.Value[0] = 'X'
		if err := tx.txn.Set([]byte("en:opaque"), []byte("later")); err != nil {
			return err
		}
		if err := j.capture([]byte("en:opaque")); err != nil {
			return err
		}
		if err := j.restore([]identityUndo{{Key: []byte("en:opaque"), Expected: identityImage{Exists: true, Value: []byte("later")}}}); err != nil {
			return err
		}
		got, err := j.image([]byte("en:opaque"))
		if err != nil {
			return err
		}
		if !got.Exists || !bytes.Equal(got.Value, old) {
			t.Fatal("capture key/value aliasing or recapture lost exact original")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGradeJournalAbsentZeroAndCompletePreflight(t *testing.T) {
	st := openTemp(t)
	if err := st.db.Update(func(tx *badger.Txn) error { return tx.Set([]byte("al:zero"), nil) }); err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		for _, key := range []string{"al:zero", "al:absent", "en:first"} {
			if err := j.capture([]byte(key)); err != nil {
				return err
			}
		}
		for _, key := range []string{"al:zero", "al:absent", "en:first"} {
			if err := tx.txn.Set([]byte(key), []byte("replacement")); err != nil {
				return err
			}
		}
		good := identityImage{Exists: true, Value: []byte("replacement")}
		cases := [][]identityUndo{
			{{Key: []byte("en:first"), Expected: good}, {Key: []byte("al:uncaptured"), Expected: identityImage{}}},
			{{Key: []byte("en:first"), Expected: good}, {Key: []byte("en:first"), Expected: good}},
			{{Key: []byte("en:first"), Expected: good}, {Key: []byte("al:zero"), Expected: identityImage{Exists: true, Value: []byte("stale")}}},
			{{Key: []byte("en:first"), Expected: good}, {Key: []byte("fa:outside"), Expected: identityImage{}}},
		}
		for _, undo := range cases {
			if err := j.restore(undo); !errors.Is(err, errIdentityJournal) {
				t.Fatalf("bad preflight = %v", err)
			}
			got, err := j.image([]byte("en:first"))
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, good) {
				t.Fatal("precondition failure restored earlier entry")
			}
		}
		if err := j.restore([]identityUndo{{Key: []byte("al:zero"), Expected: good}, {Key: []byte("al:absent"), Expected: good}}); err != nil {
			return err
		}
		zero, err := j.image([]byte("al:zero"))
		if err != nil {
			return err
		}
		absent, err := j.image([]byte("al:absent"))
		if err != nil {
			return err
		}
		if !zero.Exists || len(zero.Value) != 0 || absent.Exists {
			t.Fatal("zero bytes conflated with absent")
		}
		if err := j.restore([]identityUndo{{Key: []byte("al:zero"), Expected: identityImage{}}}); !errors.Is(err, errIdentityJournal) {
			t.Fatal("existing zero accepted as absent")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGradeJournalOnlyExplicitAbsentEntityEvents(t *testing.T) {
	st := openTemp(t)
	if err := st.PutEntity(Entity{Slug: "existing", Name: "Existing", Type: "service"}); err != nil {
		t.Fatal(err)
	}
	var observed []Event
	st.SetObserver(func(e Event) { observed = append(observed, e) })
	want := []Event{{Kind: "entity", Op: "put", Entity: Entity{Slug: "existing"}}, {Kind: "entity", Op: "put", Entity: Entity{Slug: "other"}}, {Kind: "fact", Op: "put", Entity: Entity{Slug: "gone"}}, {Kind: "episode", Op: "put", Entity: Entity{Slug: "gone"}}, {Kind: "entity", Op: "delete", Slug: "other"}}
	if err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		for _, slug := range []string{"gone", "existing"} {
			if err := j.capture([]byte("en:" + slug)); err != nil {
				return err
			}
		}
		events := append([]Event{{Kind: "entity", Op: "put", Entity: Entity{Slug: "gone"}}, {Kind: "entity", Op: "delete", Slug: "gone"}}, want...)
		for _, e := range events {
			tx.notify(e)
		}
		if err := j.suppressAbsentEntityEvents([]string{"gone", "existing"}); !errors.Is(err, errIdentityJournal) {
			t.Fatal("existing entity suppression accepted")
		}
		if !reflect.DeepEqual(*tx.pendingEvents, events) {
			t.Fatal("failed event preflight changed buffer")
		}
		if err := j.suppressAbsentEntityEvents([]string{"gone", "uncaptured"}); !errors.Is(err, errIdentityJournal) {
			t.Fatal("uncaptured entity suppression accepted")
		}
		return j.suppressAbsentEntityEvents([]string{"gone", "gone"})
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(observed, want) {
		t.Fatal("event filtering lost unrelated/existing/fact/episode event or leaked absent entity")
	}
}

func TestGradeJournalRestoreWriteErrorOuterRollback(t *testing.T) {
	st := openTemp(t)
	var observed int
	st.SetObserver(func(Event) { observed++ })
	err := st.AtomicWrite(func(tx *Store) error {
		j, err := newIdentityJournal(tx)
		if err != nil {
			return err
		}
		first := []byte("en:first")
		oversized := []byte("en:" + strings.Repeat("x", 65000))
		for _, key := range [][]byte{first, oversized} {
			if err := j.capture(key); err != nil {
				return err
			}
		}
		if err := tx.PutEntity(Entity{Slug: "first", Name: "First", Type: "service"}); err != nil {
			return err
		}
		current, err := j.image(first)
		if err != nil {
			return err
		}
		err = j.restore([]identityUndo{{Key: first, Expected: current}, {Key: oversized, Expected: identityImage{}}})
		if err == nil || !strings.Contains(err.Error(), "identity journal restore") {
			t.Fatalf("expected staging failure, got %v", err)
		}
		// Preflight errors promise no staged changes; storage errors instead
		// rely on the caller returning the error to roll the outer write back.
		after, readErr := j.image(first)
		if readErr != nil {
			return readErr
		}
		if after.Exists {
			t.Fatal("fixture did not exercise a partial staged restore before storage error")
		}
		return err
	})
	if err == nil || observed != 0 {
		t.Fatal("storage failure was swallowed or emitted events")
	}
	if _, err := st.GetEntity("first"); !errors.Is(err, ErrNotFound) {
		t.Fatal("outer rollback leaked entity")
	}
	if _, found, err := st.ResolveAlias("First"); err != nil || found {
		t.Fatal("outer rollback leaked claim")
	}
}

func TestGradeJournalPanicAndNestedScope(t *testing.T) {
	st := openTemp(t)
	var escaped *identityJournal
	var observed int
	st.SetObserver(func(Event) { observed++ })
	func() {
		defer func() {
			if recover() != "synthetic panic" {
				t.Fatal("missing panic")
			}
		}()
		_ = st.AtomicWrite(func(tx *Store) error {
			return tx.AtomicWrite(func(nested *Store) error {
				if nested != tx {
					t.Fatal("nested transaction changed facade")
				}
				var err error
				escaped, err = newIdentityJournal(nested)
				if err != nil {
					return err
				}
				if err := escaped.capture([]byte("en:panic")); err != nil {
					return err
				}
				if err := nested.PutEntity(Entity{Slug: "panic", Name: "Panic", Type: "service"}); err != nil {
					return err
				}
				panic("synthetic panic")
			})
		})
	}()
	if observed != 0 {
		t.Fatal("panic leaked callbacks")
	}
	if _, err := st.GetEntity("panic"); !errors.Is(err, ErrNotFound) {
		t.Fatal("panic committed identity")
	}
	if err := escaped.capture([]byte("en:panic")); !errors.Is(err, badger.ErrDiscardedTxn) {
		t.Fatalf("expired capture = %v", err)
	}
	if err := escaped.restore(nil); !errors.Is(err, badger.ErrDiscardedTxn) {
		t.Fatalf("expired restore = %v", err)
	}
	if err := escaped.suppressAbsentEntityEvents(nil); !errors.Is(err, badger.ErrDiscardedTxn) {
		t.Fatalf("expired suppression = %v", err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		if err := escaped.capture([]byte("en:second")); !errors.Is(err, badger.ErrDiscardedTxn) {
			t.Fatal("journal rebound to later transaction")
		}
		var err error
		escaped, err = newIdentityJournal(tx)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := escaped.restore(nil); !errors.Is(err, badger.ErrDiscardedTxn) {
		t.Fatal("successful transaction journal remained live")
	}
}
