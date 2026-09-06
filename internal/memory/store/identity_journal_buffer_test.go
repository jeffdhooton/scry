package store

import (
	"bytes"
	"errors"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

// The caller keeps the undo key until restore returns, then reuses its buffer.
// A whitelist checked before restore must not become a fact write at commit.
func TestDisproofJournalRestoreRetainsCallerKeyAndWritesFactFamily(t *testing.T) {
	st := openTemp(t)
	entityKey := []byte("en:opaque")
	factKey := []byte("fa:opaque")
	original := []byte("original identity bytes")
	fact := []byte("fact bytes that must never change")
	if err := st.db.Update(func(tx *badger.Txn) error {
		if err := tx.Set(entityKey, original); err != nil {
			return err
		}
		return tx.Set(factKey, fact)
	}); err != nil {
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
		if err := tx.txn.Set([]byte("en:opaque"), []byte("changed")); err != nil {
			return err
		}
		if err := j.restore([]identityUndo{{Key: key, Expected: identityImage{Exists: true, Value: []byte("changed")}}}); err != nil {
			return err
		}
		copy(key, "fa:opaque")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(factKey)
		if err != nil {
			return err
		}
		got, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		if !bytes.Equal(got, fact) {
			t.Fatalf("restore escaped en:/al:/att: whitelist through caller key reuse; fact value changed to identity before-image")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDisproofJournalRestoreRetainsCallerKeyAndDeletesFactFamily(t *testing.T) {
	st := openTemp(t)
	if err := st.db.Update(func(tx *badger.Txn) error { return tx.Set([]byte("fa:opaque"), []byte("preserve")) }); err != nil {
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
		if err := tx.txn.Set([]byte("en:opaque"), []byte("new")); err != nil {
			return err
		}
		if err := j.restore([]identityUndo{{Key: key, Expected: identityImage{Exists: true, Value: []byte("new")}}}); err != nil {
			return err
		}
		copy(key, "fa:opaque")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.db.View(func(tx *badger.Txn) error {
		_, err := tx.Get([]byte("fa:opaque"))
		if errors.Is(err, badger.ErrKeyNotFound) {
			t.Fatal("restore escaped whitelist and deleted untouched fact key after caller key reuse")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
