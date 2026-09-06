package store

// PRIVATE, UNCALLED PROTOTYPE. This ledger neither admits graph entities nor
// implements adoption, legacy fallback, observation retention or finalization.

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
)

const identityGenerationPrefix = "ig:"
const identityGenerationAttestPrefix = "iga:"

var errIdentityGeneration = errors.New("memory: identity generation precondition failed")

type identityBirth struct {
	EpisodeID  string    `json:"episode_id"`
	Occurrence int       `json:"occurrence"`
	Slug       string    `json:"slug"`
	Name       string    `json:"name"`
	Origin     string    `json:"origin"`
	CreatedAt  time.Time `json:"created_at"`
}

type identityGenerationRecord struct {
	Version int           `json:"version"`
	ID      string        `json:"id"`
	Birth   identityBirth `json:"birth"`
}

type generationAttestation struct {
	Generation string `json:"generation"`
	Alias      string `json:"alias"`
	EpisodeID  string `json:"episode_id"`
}

type identityGenerationSession struct {
	st          *Store
	record      identityGenerationRecord
	raw         []byte
	episodeID   string
	provisional bool
}

func generationDigest(value []byte) string { return fmt.Sprintf("%x", sha256.Sum256(value)) }

func identityBirthRecord(b identityBirth) (identityGenerationRecord, []byte, error) {
	// JSON replaces invalid UTF-8 with U+FFFD. Refuse it before encoding:
	// accepted complete tuples must not collapse or become unreadable on reload.
	for _, value := range []string{b.EpisodeID, b.Slug, b.Name, b.Origin} {
		if !utf8.ValidString(value) {
			return identityGenerationRecord{}, nil, errIdentityGeneration
		}
	}
	if b.EpisodeID == "" || b.Occurrence < 0 || b.Name == "" || b.Slug == "" || Slugify(b.Name) != b.Slug || b.CreatedAt.IsZero() || (b.Origin != "declaration" && b.Origin != "endpoint") {
		return identityGenerationRecord{}, nil, errIdentityGeneration
	}
	// Timestamp identity is the exact instant, matching the entity consistency
	// check's Time.Equal contract. UTC preserves nanoseconds and offsets with
	// second precision that RFC3339 JSON cannot otherwise roundtrip losslessly.
	// This canonicalizes only the new selector, never entity or fact timestamps.
	b.CreatedAt = b.CreatedAt.UTC()
	// Fixed versioned struct encoding is the complete creation identity. The
	// digest is only its address; reads also reconstruct and compare all bytes.
	birthBytes, err := json.Marshal(struct {
		Version int           `json:"version"`
		Birth   identityBirth `json:"birth"`
	}{1, b})
	if err != nil {
		return identityGenerationRecord{}, nil, err
	}
	record := identityGenerationRecord{Version: 1, ID: generationDigest(birthBytes), Birth: b}
	raw, err := json.Marshal(record)
	return record, raw, err
}

func generationTransaction(st *Store) error {
	if st == nil || st.txn == nil || st.pendingEvents == nil {
		return errIdentityGeneration
	}
	_, err := st.txn.Get([]byte(keySchemaVersion))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil
	}
	return err
}

func generationRead(st *Store, key []byte) ([]byte, bool, error) {
	item, err := st.txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	raw, err := item.ValueCopy(nil)
	return raw, true, err
}

func beginIdentityGeneration(st *Store, b identityBirth) (*identityGenerationSession, error) {
	if err := generationTransaction(st); err != nil {
		return nil, err
	}
	record, raw, err := identityBirthRecord(b)
	if err != nil {
		return nil, err
	}
	for _, key := range []string{prefixEntity + b.Slug, identityGenerationPrefix + b.Slug} {
		_, exists, err := generationRead(st, []byte(key))
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errIdentityGeneration
		}
	}
	prefix := []byte(identityGenerationAttestPrefix + record.ID + ":")
	it := st.txn.NewIterator(badger.DefaultIteratorOptions)
	it.Seek(prefix)
	occupied := it.ValidForPrefix(prefix)
	it.Close()
	if occupied {
		return nil, errIdentityGeneration
	}
	if err := st.txn.Set([]byte(identityGenerationPrefix+b.Slug), bytes.Clone(raw)); err != nil {
		return nil, err
	}
	return &identityGenerationSession{st: st, record: record, raw: raw, episodeID: b.EpisodeID, provisional: true}, nil
}

func loadIdentityGeneration(st *Store, slug, episodeID string) (*identityGenerationSession, error) {
	if err := generationTransaction(st); err != nil {
		return nil, err
	}
	if slug == "" || episodeID == "" || !utf8.ValidString(slug) || !utf8.ValidString(episodeID) {
		return nil, errIdentityGeneration
	}
	raw, exists, err := generationRead(st, []byte(identityGenerationPrefix+slug))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errIdentityGeneration
	}
	var record identityGenerationRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, errIdentityGeneration
	}
	expected, canonical, err := identityBirthRecord(record.Birth)
	if err != nil || record.Birth.Slug != slug || record.Version != 1 || record.ID != expected.ID || !bytes.Equal(raw, canonical) {
		return nil, errIdentityGeneration
	}
	session := &identityGenerationSession{st: st, record: record, raw: raw, episodeID: episodeID}
	if err := session.check(true); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *identityGenerationSession) check(requireEntity bool) error {
	if err := generationTransaction(s.st); err != nil {
		return err
	}
	raw, exists, err := generationRead(s.st, []byte(identityGenerationPrefix+s.record.Birth.Slug))
	if err != nil {
		return err
	}
	if !exists || !bytes.Equal(raw, s.raw) {
		return errIdentityGeneration
	}
	e, err := s.st.GetEntity(s.record.Birth.Slug)
	if errors.Is(err, ErrNotFound) && s.provisional && !requireEntity {
		return nil
	}
	if err != nil {
		return errIdentityGeneration
	}
	if e.Slug != s.record.Birth.Slug || e.Name != s.record.Birth.Name || !e.CreatedAt.Equal(s.record.Birth.CreatedAt) {
		return errIdentityGeneration
	}
	return nil
}

func generationAttestationKey(a generationAttestation) []byte {
	return []byte(identityGenerationAttestPrefix + a.Generation + ":" + generationDigest([]byte(a.Alias)) + ":" + generationDigest([]byte(a.EpisodeID)))
}

func (s *identityGenerationSession) evidence(alias string) (map[string]bool, error) {
	prefix := []byte(identityGenerationAttestPrefix + s.record.ID + ":")
	if alias != "" {
		prefix = append(prefix, []byte(generationDigest([]byte(alias))+":")...)
	}
	episodes := make(map[string]bool)
	it := s.st.txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Item().KeyCopy(nil)
		raw, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var a generationAttestation
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, errIdentityGeneration
		}
		canonical, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		if !utf8.ValidString(a.Alias) || !utf8.ValidString(a.EpisodeID) || a.Generation != s.record.ID || a.Alias == "" || Normalize(a.Alias) != a.Alias || a.EpisodeID == "" || (alias != "" && a.Alias != alias) || !bytes.Equal(raw, canonical) || !bytes.Equal(key, generationAttestationKey(a)) {
			return nil, errIdentityGeneration
		}
		episodes[a.EpisodeID] = true
	}
	return episodes, nil
}

func (s *identityGenerationSession) attest(alias string) (int, error) {
	if err := s.check(!s.provisional); err != nil {
		return 0, err
	}
	if alias == "" || !utf8.ValidString(alias) || strings.TrimSpace(alias) != alias || Normalize(alias) != alias {
		return 0, errIdentityGeneration
	}
	episodes, err := s.evidence(alias)
	if err != nil {
		return 0, err
	}
	if episodes[s.episodeID] {
		return len(episodes), nil
	}
	a := generationAttestation{Generation: s.record.ID, Alias: alias, EpisodeID: s.episodeID}
	raw, err := json.Marshal(a)
	if err != nil {
		return 0, err
	}
	if err := s.st.txn.Set(generationAttestationKey(a), raw); err != nil {
		return 0, err
	}
	return len(episodes) + 1, nil
}

// finishSupported is deliberately explicit in this primitive. A production
// controller would need to make it unavoidable and additionally prove actual
// fact support and observation retention. This method makes no alias claim.
func (s *identityGenerationSession) finishSupported() error {
	if err := s.check(true); err != nil {
		return err
	}
	episodes, err := s.evidence("")
	if err != nil {
		return err
	}
	episodes[s.record.Birth.EpisodeID] = true
	episodes[s.episodeID] = true
	for id := range episodes {
		has, err := s.st.HasEpisode(id)
		if err != nil {
			return err
		}
		if !has {
			return errIdentityGeneration
		}
	}
	s.provisional = false
	return nil
}
