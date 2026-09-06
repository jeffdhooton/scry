package store

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func generationTestBirth() identityBirth {
	return identityBirth{EpisodeID: "birth", Occurrence: 0, Slug: "aurora-handbook", Name: "Aurora Handbook", Origin: "declaration", CreatedAt: time.Date(2026, 9, 6, 1, 0, 0, 123, time.UTC)}
}

func generationTestRaw(t *testing.T, st *Store, key string) []byte {
	t.Helper()
	var value []byte
	if err := st.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return value
}

func generationTestCommitBirth(t *testing.T, st *Store, b identityBirth) {
	t.Helper()
	if err := st.AtomicWrite(func(tx *Store) error {
		s, err := beginIdentityGeneration(tx, b)
		if err != nil {
			return err
		}
		if err := tx.PutEntity(Entity{Slug: b.Slug, Name: b.Name, Type: "concept", CreatedAt: b.CreatedAt, LastSeen: b.CreatedAt}); err != nil {
			return err
		}
		if err := tx.PutFact(Fact{Src: b.Slug, Relation: "status", Value: "available", Fact: "The synthetic handbook is available.", ValidFrom: b.CreatedAt, Confidence: 1, Episodes: []string{b.EpisodeID}}); err != nil {
			return err
		}
		if err := tx.PutEpisode(Episode{ID: b.EpisodeID, OccurredAt: b.CreatedAt}); err != nil {
			return err
		}
		return s.finishSupported()
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationIsolatesLegacyAcrossSupportedBirthAndReopen(t *testing.T) {
	for _, legacyCount := range []int{1, 8} {
		t.Run(fmt.Sprint(legacyCount), func(t *testing.T) {
			st := openTemp(t)
			b := generationTestBirth()
			alias := Normalize("Polar Manual")
			for i := 0; i < legacyCount; i++ {
				id := fmt.Sprintf("old-%d", i)
				if _, err := st.AttestAlias(b.Slug, alias, id); err != nil {
					t.Fatal(err)
				}
			}
			key := prefixAttest + b.Slug + ":" + alias
			legacy := generationTestRaw(t, st, key)
			generationTestCommitBirth(t, st, b)
			for i := 1; i <= 2; i++ {
				id := fmt.Sprintf("fresh-%d", i)
				if err := st.AtomicWrite(func(tx *Store) error {
					s, err := loadIdentityGeneration(tx, b.Slug, id)
					if err != nil {
						return err
					}
					for repeat := 0; repeat < 2; repeat++ {
						count, err := s.attest(alias)
						if err != nil {
							return err
						}
						if count != i {
							t.Fatalf("new evidence count %d, want %d", count, i)
						}
					}
					if err := tx.PutEpisode(Episode{ID: id, OccurredAt: b.CreatedAt.Add(time.Duration(i) * time.Hour)}); err != nil {
						return err
					}
					return s.finishSupported()
				}); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(legacy, generationTestRaw(t, st, key)) {
					t.Fatal("legacy attestation bytes changed")
				}
				if _, found, err := st.ResolveAlias("Polar Manual"); err != nil || found {
					t.Fatal("ledger primitive claimed an alias")
				}
			}
			// Normal metadata updates do not select a new generation.
			e, err := st.GetEntity(b.Slug)
			if err != nil {
				t.Fatal(err)
			}
			e.Type = "runbook"
			e.Description = "A retained synthetic description."
			e.LastSeen = e.LastSeen.Add(time.Hour)
			if err := st.PutEntity(e); err != nil {
				t.Fatal(err)
			}
			var backup bytes.Buffer
			if _, err := st.Backup(&backup); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(t.TempDir(), "direct-restored")
			db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Load(bytes.NewReader(backup.Bytes()), 16); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			restored, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer restored.Close()
			if err := restored.AtomicWrite(func(tx *Store) error {
				s, err := loadIdentityGeneration(tx, b.Slug, "fresh-2")
				if err != nil {
					return err
				}
				count, err := s.attest(alias)
				if err != nil {
					return err
				}
				if count != 2 {
					t.Fatal("reopen changed evidence count")
				}
				return s.finishSupported()
			}); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(legacy, generationTestRaw(t, restored, key)) {
				t.Fatal("backup/reopen changed legacy bytes")
			}
		})
	}
}

func TestGenerationRefusesBrokenBindingAndMissingProvenance(t *testing.T) {
	st := openTemp(t)
	b := generationTestBirth()
	var escaped *identityGenerationSession
	err := st.AtomicWrite(func(tx *Store) error {
		s, err := beginIdentityGeneration(tx, b)
		if err != nil {
			return err
		}
		escaped = s
		if _, err := s.attest(Normalize("Polar Manual")); err != nil {
			return err
		}
		if err := tx.PutEntity(Entity{Slug: b.Slug, Name: b.Name, CreatedAt: b.CreatedAt}); err != nil {
			return err
		}
		return s.finishSupported() // The episode was deliberately not written.
	})
	if !errors.Is(err, errIdentityGeneration) {
		t.Fatal("missing provenance accepted")
	}
	if _, err := st.GetEntity(b.Slug); !errors.Is(err, ErrNotFound) {
		t.Fatal("failed birth leaked entity")
	}
	if _, err := escaped.attest(Normalize("Polar Manual")); err == nil {
		t.Fatal("expired generation accepted")
	}
	generationTestCommitBirth(t, st, b)
	selector := generationTestRaw(t, st, identityGenerationPrefix+b.Slug)
	if err := st.db.Update(func(tx *badger.Txn) error {
		return tx.Set([]byte(identityGenerationPrefix+b.Slug), append(bytes.Clone(selector), ' '))
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error { _, err := loadIdentityGeneration(tx, b.Slug, "fresh"); return err }); !errors.Is(err, errIdentityGeneration) {
		t.Fatal("noncanonical selector accepted")
	}
	if !bytes.Equal(append(bytes.Clone(selector), ' '), generationTestRaw(t, st, identityGenerationPrefix+b.Slug)) {
		t.Fatal("unsupported selector rewritten")
	}
}

func TestGenerationAliasAndGenerationSeparation(t *testing.T) {
	st := openTemp(t)
	b := generationTestBirth()
	generationTestCommitBirth(t, st, b)
	b2 := b
	b2.Slug = "borealis-manual"
	b2.Name = "Borealis Manual"
	b2.EpisodeID = "other-birth"
	generationTestCommitBirth(t, st, b2)
	if err := st.AtomicWrite(func(tx *Store) error {
		for _, slug := range []string{b.Slug, b2.Slug} {
			s, err := loadIdentityGeneration(tx, slug, "fresh")
			if err != nil {
				return err
			}
			for _, alias := range []string{"polar-manual", "polar-guide"} {
				n, err := s.attest(alias)
				if err != nil {
					return err
				}
				if n != 1 {
					t.Fatal("alias/generation evidence leaked")
				}
			}
			if err := tx.PutEpisode(Episode{ID: "fresh", OccurredAt: b.CreatedAt}); err != nil {
				return err
			}
			if err := s.finishSupported(); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error { _, err := beginIdentityGeneration(tx, b); return err }); !errors.Is(err, errIdentityGeneration) {
		t.Fatal("supported generation resurrected")
	}
}
