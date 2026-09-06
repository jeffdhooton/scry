package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

// HistoricalRestatement reads only the exact legacy address, including staged
// writes in an enclosing AtomicWrite. A caller that will rewrite the returned
// typed record must not lose unknown JSON fields: refuse any representation
// that does not round-trip byte-for-byte through Fact. This deliberately also
// refuses noncanonical but otherwise supported JSON rather than rewriting it.
// Missing and current rows return nil; no interval is inferred from text.
func (s *Store) HistoricalRestatement(incoming Fact) (*Fact, error) {
	var found *Fact
	key := factKey(incoming.Src, incoming.Relation, incoming.KeyDst(), incoming.ValidFrom)
	err := s.view(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(raw []byte) error {
			var f Fact
			if err := json.Unmarshal(raw, &f); err != nil {
				return fmt.Errorf("%w: unreadable fact address sha256=%x", ErrFactConflict, sha256.Sum256(key))
			}
			if f.InvalidAt == nil {
				return nil
			}
			if f.Src != incoming.Src || f.Relation != incoming.Relation || f.Dst != incoming.Dst || f.Value != incoming.Value || f.RawRelation != incoming.RawRelation || f.Fact != incoming.Fact || !f.ValidFrom.Equal(incoming.ValidFrom) {
				return fmt.Errorf("%w: occupied fact key sha256=%x; preserve the queued episode for exact assertion review", ErrFactConflict, sha256.Sum256(key))
			}
			canonical, err := json.Marshal(f)
			if err != nil || !bytes.Equal(raw, canonical) || !bytes.Equal(key, factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom)) {
				return fmt.Errorf("%w: unsupported historical payload sha256=%x", ErrFactConflict, sha256.Sum256(key))
			}
			found = &f
			return nil
		})
	})
	return found, err
}
