package store

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestInputRevisionOwnedStagingAndPersistentRestore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "source")
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := inputFixture()
	inputEpisode(t, st, r)
	gradeGenerationSet(t, st, "opaque:old", []byte{0, 255, 1})
	before := gradeGenerationSnapshot(t, st)
	want, key, _ := encodeIdentityInput(r)
	err = st.AtomicWrite(func(tx *Store) error {
		got, err := putIdentityInput(tx, r)
		if err != nil || got != key {
			t.Fatal("write failed")
		}
		r.Declarations[0].Aliases[0] = "caller mutation"
		r.Facts[0].Supersedes.Dst = "caller mutation"
		chunk, err := readIdentityInputChunk(tx, r.EpisodeID, key, 0, 24576)
		if err != nil || !bytes.Equal(chunk.Data, want) {
			t.Fatal("staged read lost bytes")
		}
		for i := range chunk.Data {
			chunk.Data[i] = 0
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(inputAssemble(t, st, r, key, 512), want) {
		t.Fatal("caller changed staged bytes")
	}
	after := gradeGenerationSnapshot(t, st)
	if len(after) != len(before)+1 {
		t.Fatal("wrong write set")
	}
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("old bytes changed")
		}
	}
	var backup bytes.Buffer
	if _, err := st.Backup(&backup); err != nil || backup.Len() == 0 {
		t.Fatal("nonempty backup failed")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !reflect.DeepEqual(after, gradeGenerationSnapshot(t, st)) || !bytes.Equal(inputAssemble(t, st, r, key, 512), want) {
		t.Fatal("reopen changed bytes")
	}
	restored, err := Open(filepath.Join(t.TempDir(), "restored"))
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if err := restored.Restore(bytes.NewReader(backup.Bytes())); err != nil {
		t.Fatal("restore failed")
	}
	if !reflect.DeepEqual(after, gradeGenerationSnapshot(t, restored)) || !bytes.Equal(inputAssemble(t, restored, r, key, 512), want) {
		t.Fatal("restoration changed bytes")
	}
}

func TestInputRevisionActualCommitConflictAndOrdinaryRollback(t *testing.T) {
	st := openTemp(t)
	r := inputFixture()
	inputEpisode(t, st, r)
	_, key, _ := encodeIdentityInput(r)
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	err := st.AtomicWrite(func(tx *Store) error {
		if _, err := putIdentityInput(tx, r); err != nil {
			return err
		}
		if err := tx.txn.Set([]byte("opaque:staged"), []byte("staged")); err != nil {
			return err
		}
		return st.db.Update(func(other *badger.Txn) error { return other.Set([]byte(key), []byte("synthetic competitor")) })
	})
	before[key] = []byte("synthetic competitor")
	if !errors.Is(err, badger.ErrConflict) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) || events != 0 {
		t.Fatal("conflicted input leaked commit", err)
	}
	err = st.AtomicWrite(func(tx *Store) error {
		if err := tx.txn.Set([]byte("opaque:ordinary"), []byte("staged")); err != nil {
			return err
		}
		_, err := putIdentityInput(tx, r)
		return err
	})
	if err != errIdentityInput || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("ordinary propagated error leaked write")
	}
}
