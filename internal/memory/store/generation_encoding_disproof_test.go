package store

import (
	"bytes"
	"testing"
	"time"
)

// These are distinct complete creation tuples, not a cryptographic collision.
// Rejecting malformed text is acceptable; accepting it with identical encoded
// identity is not.
func TestDisproofGenerationFullTupleEncodingIsLossless(t *testing.T) {
	a := identityBirth{EpisodeID: "episode-\xff", Occurrence: 1, Slug: "atlas-guide", Name: "Atlas Guide", Origin: "declaration", CreatedAt: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)}
	b := a
	b.EpisodeID = "episode-\xfe"
	ra, rawA, errA := identityBirthRecord(a)
	rb, rawB, errB := identityBirthRecord(b)
	if errA != nil || errB != nil {
		return
	}
	if ra.ID == rb.ID || bytes.Equal(rawA, rawB) {
		t.Fatal("two accepted distinct birth episode IDs collapsed to identical selector bytes and generation ID")
	}
}

func TestDisproofGenerationAcceptedBirthRemainsLoadable(t *testing.T) {
	st := openTemp(t)
	b := identityBirth{EpisodeID: "episode-\xff", Occurrence: 1, Slug: "atlas-guide", Name: "Atlas Guide", Origin: "declaration", CreatedAt: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)}
	err := st.AtomicWrite(func(tx *Store) error {
		s, err := beginIdentityGeneration(tx, b)
		if err != nil {
			return err
		}
		if err := tx.PutEntity(Entity{Slug: b.Slug, Name: b.Name, CreatedAt: b.CreatedAt}); err != nil {
			return err
		}
		if err := tx.PutFact(Fact{Src: b.Slug, Relation: "status", Value: "ready", Fact: "The synthetic guide is ready.", ValidFrom: b.CreatedAt, Episodes: []string{b.EpisodeID}}); err != nil {
			return err
		}
		if err := tx.PutEpisode(Episode{ID: b.EpisodeID, OccurredAt: b.CreatedAt}); err != nil {
			return err
		}
		return s.finishSupported()
	})
	if err != nil {
		return
	} // Explicit refusal is safe.
	err = st.AtomicWrite(func(tx *Store) error { _, err := loadIdentityGeneration(tx, b.Slug, "later-episode"); return err })
	if err != nil {
		t.Fatalf("successfully finished and committed birth cannot be loaded: %v", err)
	}
}
