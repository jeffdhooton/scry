package assessstore

import (
	"bytes"
	"encoding/json"

	"github.com/dgraph-io/badger/v4"
	"github.com/jeffdhooton/scry/internal/memory/assess"
)

// SavePreparation retains an immutable, redacted evidence/budget manifest when
// building a dispatchable request fails. It never creates a packet or packet
// hash, so an unsent oversized core cannot become eligible for packet replay.
func (s *Store) SavePreparation(id, owner string, manifest assess.Manifest) error {
	var clean assess.Manifest
	if err := json.Unmarshal(redactedJSON(manifest), &clean); err != nil {
		return err
	}
	return s.write(func(tx *badger.Txn, st *Status) error {
		var j Job
		if err := get(tx, "j:"+id, &j); err != nil {
			return err
		}
		if j.Status != Running || j.Owner != owner {
			return ErrOwnership
		}
		if len(j.Packet) > 0 || !bytes.Equal(encoded(j.Manifest), encoded(assess.Manifest{})) {
			if !bytes.Equal(encoded(j.Manifest), encoded(clean)) {
				return ErrConflict
			}
			return nil
		}
		old := j
		j.Manifest = clean
		j.UpdatedAt = s.opts.Now()
		if err := s.saveJob(tx, st, &old, j); err != nil {
			return err
		}
		if s.capacity(st) {
			return ErrCapacity
		}
		return nil
	})
}
