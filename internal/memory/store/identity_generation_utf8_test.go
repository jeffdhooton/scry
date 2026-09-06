package store

import (
	"bytes"
	"errors"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestGenerationRejectsInvalidUTF8BeforeStaging(t *testing.T) {
	for _, field := range []string{"episode", "name", "slug", "origin"} {
		t.Run(field, func(t *testing.T) {
			st := openTemp(t)
			birth := generationTestBirth()
			switch field {
			case "episode":
				birth.EpisodeID += "\xff"
			case "name":
				birth.Name += "\xff"
			case "slug":
				birth.Slug += "\xff"
			case "origin":
				birth.Origin += "\xff"
			}
			if err := st.AtomicWrite(func(tx *Store) error {
				_, err := beginIdentityGeneration(tx, birth)
				if !errors.Is(err, errIdentityGeneration) {
					t.Fatal("invalid birth accepted")
				}
				// Even a caller continuing after this validation refusal must not
				// find a staged selector. Other errors still require outer abort.
				_, exists, readErr := generationRead(tx, []byte(identityGenerationPrefix+birth.Slug))
				if readErr != nil || exists {
					t.Fatal("invalid birth staged a selector")
				}
				return err
			}); !errors.Is(err, errIdentityGeneration) {
				t.Fatal("expected explicit refusal")
			}
		})
	}
	st := openTemp(t)
	birth := generationTestBirth()
	birth.EpisodeID = "épisode-東京-�"
	generationTestCommitBirth(t, st, birth)
	selector := generationTestRaw(t, st, identityGenerationPrefix+birth.Slug)
	if err := st.AtomicWrite(func(tx *Store) error {
		if _, err := loadIdentityGeneration(tx, birth.Slug, "later-\xff"); !errors.Is(err, errIdentityGeneration) {
			t.Fatal("invalid session ID accepted")
		}
		s, err := loadIdentityGeneration(tx, birth.Slug, "later-épisode")
		if err != nil {
			return err
		}
		if _, err := s.attest("alias-\xff"); !errors.Is(err, errIdentityGeneration) {
			t.Fatal("invalid alias accepted")
		}
		prefix := []byte(identityGenerationAttestPrefix)
		it := tx.txn.NewIterator(badger.DefaultIteratorOptions)
		it.Seek(prefix)
		occupied := it.ValidForPrefix(prefix)
		it.Close()
		if occupied {
			t.Fatal("refused text staged ledger evidence")
		}
		if _, err := s.attest("polar-manual"); err != nil {
			return err
		}
		if err := tx.PutEpisode(Episode{ID: "later-épisode", OccurredAt: birth.CreatedAt}); err != nil {
			return err
		}
		return s.finishSupported()
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(selector, generationTestRaw(t, st, identityGenerationPrefix+birth.Slug)) {
		t.Fatal("valid Unicode selector changed")
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		s, err := loadIdentityGeneration(tx, birth.Slug, "later-épisode")
		if err != nil {
			return err
		}
		n, err := s.attest("polar-manual")
		if n != 1 {
			t.Fatal("valid Unicode episode did not remain deduplicated")
		}
		if err != nil {
			return err
		}
		return s.finishSupported()
	}); err != nil {
		t.Fatal(err)
	}
}
