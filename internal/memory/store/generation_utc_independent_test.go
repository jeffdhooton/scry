package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestGradeGenerationUTCExactInstantIdentity(t *testing.T) {
	for _, offset := range []int{1, 2, -1, -2, 3601, -3601, 19801} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			b := gradeGenerationBirth()
			b.CreatedAt = time.Date(2026, 9, 6, 1, 2, 3, 987654321, time.FixedZone("seconds", offset))
			r, raw, err := identityBirthRecord(b)
			if err != nil {
				t.Fatal(err)
			}
			var decoded identityGenerationRecord
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			if !decoded.Birth.CreatedAt.Equal(b.CreatedAt) || decoded.Birth.CreatedAt.Nanosecond() != 987654321 {
				t.Fatal("selector JSON lost instant or nanoseconds")
			}
			if r.Birth.CreatedAt.Location() != time.UTC {
				t.Fatal("selector timestamp is not canonical UTC")
			}
			for _, alternate := range []int{0, 3600, -18000, 1} {
				other := b
				other.CreatedAt = b.CreatedAt.In(time.FixedZone("alternate", alternate))
				r2, raw2, err := identityBirthRecord(other)
				if err != nil || r.ID != r2.ID || !bytes.Equal(raw, raw2) {
					t.Fatal("equal instant selected different generation")
				}
			}
			later := b
			later.CreatedAt = b.CreatedAt.Add(time.Nanosecond)
			r2, raw2, err := identityBirthRecord(later)
			if err != nil || r.ID == r2.ID || bytes.Equal(raw, raw2) {
				t.Fatal("one-nanosecond distinct instant collapsed")
			}
			different := b
			different.CreatedAt = time.Date(2026, 9, 6, 1, 2, 3, 987654321, time.FixedZone("other-seconds", offset+1))
			r3, raw3, err := identityBirthRecord(different)
			if err != nil || r.ID == r3.ID || bytes.Equal(raw, raw3) {
				t.Fatal("distinct second-offset instants collapsed")
			}
		})
	}
}

func TestGradeGenerationUTCSelectorDurableReloadPreservesOtherBytes(t *testing.T) {
	st := openTemp(t)
	b := gradeGenerationBirth()
	b.CreatedAt = time.Date(2026, 9, 6, 1, 2, 3, 987654321, time.FixedZone("second-offset", 1))
	if err := st.AtomicWrite(func(tx *Store) error {
		s, err := beginIdentityGeneration(tx, b)
		if err != nil {
			return err
		}
		// The fixture materializes the same exact instant in a representation
		// that the existing Entity JSON format can preserve.
		if err = tx.PutEntity(Entity{Slug: b.Slug, Name: b.Name, CreatedAt: b.CreatedAt.UTC()}); err != nil {
			return err
		}
		if err = tx.PutFact(Fact{Src: b.Slug, Relation: "status", Value: "ready", Fact: "Synthetic guide is ready.", ValidFrom: b.CreatedAt.UTC(), Confidence: .875, Episodes: []string{b.EpisodeID}}); err != nil {
			return err
		}
		if err = tx.PutEpisode(Episode{ID: b.EpisodeID, OccurredAt: b.CreatedAt.UTC()}); err != nil {
			return err
		}
		return s.finishSupported()
	}); err != nil {
		t.Fatal(err)
	}
	// A supported entity may retain a losslessly representable non-UTC offset.
	e, err := st.GetEntity(b.Slug)
	if err != nil {
		t.Fatal(err)
	}
	e.CreatedAt = b.CreatedAt.In(time.FixedZone("hour-offset", 3600))
	if err = st.PutEntity(e); err != nil {
		t.Fatal(err)
	}
	if _, err = st.AttestAlias(b.Slug, "legacy-alias", "legacy-episode"); err != nil {
		t.Fatal(err)
	}
	before := gradeGenerationSnapshot(t, st)
	gradeGenerationVote(t, st, b.Slug, "next-exact-episode", "north-guide", 1)
	after := gradeGenerationSnapshot(t, st)
	for key, value := range before {
		if !bytes.Equal(value, after[key]) {
			t.Fatalf("primitive changed existing key %q", key)
		}
	}
	var backup bytes.Buffer
	if n, err := st.Backup(&backup); err != nil || n == 0 {
		t.Fatalf("backup: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "direct-restored")
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Load(bytes.NewReader(backup.Bytes()), 16); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if !reflect.DeepEqual(after, gradeGenerationSnapshot(t, r)) {
		t.Fatal("backup/reopen changed raw records")
	}
	gradeGenerationVote(t, r, b.Slug, "next-exact-episode", "north-guide", 1)
	gradeGenerationVote(t, r, b.Slug, "next-new-episode", "north-guide", 2)
	for key, value := range before {
		if !bytes.Equal(value, gradeGenerationSnapshot(t, r)[key]) {
			t.Fatalf("reload changed existing key %q", key)
		}
	}
}

func TestGradeGenerationUTCRefusesLossyEntityTimestamp(t *testing.T) {
	for _, mode := range []string{"birth-finalization", "existing-entity-load"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			b := gradeGenerationBirth()
			b.CreatedAt = time.Date(2026, 9, 6, 1, 2, 3, 987654321, time.FixedZone("second-offset", 1))
			if mode == "existing-entity-load" {
				canonical := b
				canonical.CreatedAt = b.CreatedAt.UTC()
				gradeGenerationBirthCommit(t, st, canonical)
				// Simulate an ordinary existing writer passing the original instant
				// through legacy Entity JSON, which loses offset seconds.
				e, err := st.GetEntity(b.Slug)
				if err != nil {
					t.Fatal(err)
				}
				e.CreatedAt = b.CreatedAt
				if err = st.PutEntity(e); err != nil {
					t.Fatal(err)
				}
				decoded, err := st.GetEntity(b.Slug)
				if err != nil {
					t.Fatal(err)
				}
				if decoded.CreatedAt.Equal(b.CreatedAt) {
					t.Fatal("fixture did not expose legacy Entity serialization loss")
				}
			}
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			err := st.AtomicWrite(func(tx *Store) error {
				if mode == "existing-entity-load" {
					_, err := loadIdentityGeneration(tx, b.Slug, "later")
					return err
				}
				s, err := beginIdentityGeneration(tx, b)
				if err != nil {
					return err
				}
				if err = tx.PutEntity(Entity{Slug: b.Slug, Name: b.Name, CreatedAt: b.CreatedAt}); err != nil {
					return err
				}
				if err = tx.PutEpisode(Episode{ID: b.EpisodeID, OccurredAt: b.CreatedAt.UTC()}); err != nil {
					return err
				}
				return s.finishSupported()
			})
			if !errors.Is(err, errIdentityGeneration) {
				t.Fatalf("lossy entity timestamp did not explicitly refuse: %v", err)
			}
			if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("refusal changed legacy state or emitted events")
			}
		})
	}
}
