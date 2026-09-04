package store

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

// Value-evidence keys are additive to the schema-1 layout. Older binaries
// ignore them, while newer binaries treat their absence as no contextual
// evidence, so adding the prefix does not require a destructive schema bump.
//
//	ve:<normalized-name> → ValueEvidence (JSON)
const prefixValueEvidence = "ve:"

// ValueEvidence records the extraction episodes that explicitly typed a
// spelling as a value. It is classification evidence, not alias ownership:
// exact entities and explicit identity declarations remain authoritative.
type ValueEvidence struct {
	Normalized string   `json:"normalized"`
	Spellings  []string `json:"spellings,omitempty"`
	Episodes   []string `json:"episodes,omitempty"`
}

// RecordValueEvidence durably records one explicit, context-bearing value
// verdict. Repeated observations and force replays are deduplicated.
func (s *Store) RecordValueEvidence(name, episodeID string) error {
	norm := Normalize(name)
	spelling := strings.TrimSpace(name)
	if norm == "" {
		return nil
	}
	s.maintenanceMu.RLock()
	defer s.maintenanceMu.RUnlock()
	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(prefixValueEvidence + norm)
		evidence := ValueEvidence{Normalized: norm}
		item, err := txn.Get(key)
		if err == nil {
			if err := item.Value(func(val []byte) error { return json.Unmarshal(val, &evidence) }); err != nil {
				return err
			}
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		if spelling != "" && !containsValueEvidenceString(evidence.Spellings, spelling) {
			evidence.Spellings = append(evidence.Spellings, spelling)
		}
		if episodeID != "" && !containsValueEvidenceString(evidence.Episodes, episodeID) {
			evidence.Episodes = append(evidence.Episodes, episodeID)
		}
		encoded, err := json.Marshal(evidence)
		if err != nil {
			return err
		}
		return txn.Set(key, encoded)
	})
}

// GetValueEvidence returns the contextual value evidence for name.
func (s *Store) GetValueEvidence(name string) (ValueEvidence, error) {
	var evidence ValueEvidence
	norm := Normalize(name)
	if norm == "" {
		return ValueEvidence{}, ErrNotFound
	}
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixValueEvidence + norm))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error { return json.Unmarshal(val, &evidence) })
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return ValueEvidence{}, ErrNotFound
	}
	return evidence, err
}

// HasValueEvidence reports whether an earlier extraction explicitly typed
// name (or that value's supplied alias) as a value.
func (s *Store) HasValueEvidence(name string) (bool, error) {
	_, err := s.GetValueEvidence(name)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func containsValueEvidenceString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
