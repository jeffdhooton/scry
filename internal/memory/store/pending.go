package store

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Pending-queue and meta keys. Both prefixes are additive to the schema-1
// layout: an older binary ignores them and a newer one tolerates their
// absence, so SchemaVersion does not move.
//
//	pq:<id>       → PendingEpisode (JSON)
//	meta:<key>    → RFC3339 timestamp or JSON blob
const (
	prefixPending = "pq:"

	// MetaLastIngest is stamped whenever a transcript-derived episode is
	// queued. It is what `scry doctor` reads to say "hours since the last
	// successful ingest".
	MetaLastIngest = "last_ingest_at"
	// MetaLastSweep is stamped when a sweep completes, whether or not it
	// found anything new.
	MetaLastSweep = "last_sweep_at"
	// MetaLastExtract is stamped when the queue worker resolves an episode
	// into facts, i.e. the provider chain is known to be working.
	MetaLastExtract = "last_extract_ok_at"
	// MetaLastSweepReport holds the last sweep's result JSON.
	MetaLastSweepReport = "last_sweep_report"
)

// PendingEpisode is a distilled episode waiting for extraction. It is the
// durable form of a memory write: `scry_remember` and the sweep both land
// here in milliseconds, and the daemon's queue worker turns it into facts
// when the provider chain is reachable. Nothing is lost when the provider
// is down; the item just waits.
type PendingEpisode struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	SourceRef   string    `json:"source_ref"`
	Text        string    `json:"text"`
	Cwd         string    `json:"cwd,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
	EnqueuedAt  time.Time `json:"enqueued_at"`
	NextAttempt time.Time `json:"next_attempt"`
	Hints       []string  `json:"hints,omitempty"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error,omitempty"`
	// Parked marks an item the worker gave up on after repeated parse
	// failures. It stays in the store, visible in `scry memory queue`, and
	// can be replayed; it is never silently dropped.
	Parked bool `json:"parked,omitempty"`
}

// PutPending writes or replaces a pending episode.
func (s *Store) PutPending(p PendingEpisode) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(prefixPending+p.ID), b)
	})
}

// GetPending returns the pending episode with id, or ErrNotFound.
func (s *Store) GetPending(id string) (PendingEpisode, error) {
	var p PendingEpisode
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixPending + id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error { return json.Unmarshal(val, &p) })
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return PendingEpisode{}, ErrNotFound
	}
	return p, err
}

// HasPending reports whether id is queued.
func (s *Store) HasPending(id string) (bool, error) {
	found := false
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(prefixPending + id))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	return found, err
}

// DeletePending removes id from the queue. Deleting a missing id is not an
// error: the worker may race a manual retry.
func (s *Store) DeletePending(id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(prefixPending + id))
	})
}

// Pending returns queued episodes oldest-first by EnqueuedAt (ties by ID),
// in every state, up to limit (0 = all).
func (s *Store) Pending(limit int) ([]PendingEpisode, error) {
	var out []PendingEpisode
	pb := []byte(prefixPending)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 64
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(pb); it.ValidForPrefix(pb); it.Next() {
			var p PendingEpisode
			if err := it.Item().Value(func(val []byte) error { return json.Unmarshal(val, &p) }); err != nil {
				return err
			}
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].EnqueuedAt.Equal(out[j].EnqueuedAt) {
			return out[i].EnqueuedAt.Before(out[j].EnqueuedAt)
		}
		return out[i].ID < out[j].ID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// PendingCounts splits the queue into ready (claimable now), backoff
// (waiting for NextAttempt), and parked.
func (s *Store) PendingCounts(now time.Time) (ready, backoff, parked int, err error) {
	all, err := s.Pending(0)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, p := range all {
		switch {
		case p.Parked:
			parked++
		case p.NextAttempt.After(now):
			backoff++
		default:
			ready++
		}
	}
	return ready, backoff, parked, nil
}

// PutMetaTime stores t under meta:<key> as RFC3339Nano.
func (s *Store) PutMetaTime(key string, t time.Time) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(prefixMeta+key), []byte(t.UTC().Format(time.RFC3339Nano)))
	})
}

// GetMetaTime reads meta:<key>; found is false when it was never written.
func (s *Store) GetMetaTime(key string) (t time.Time, found bool, err error) {
	var raw string
	err = s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixMeta + key))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return item.Value(func(val []byte) error { raw = string(val); return nil })
	})
	if err != nil || !found {
		return time.Time{}, found, err
	}
	t, err = time.Parse(time.RFC3339Nano, raw)
	return t, true, err
}

// PutMetaJSON stores v as JSON under meta:<key>.
func (s *Store) PutMetaJSON(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(prefixMeta+key), b)
	})
}

// GetMetaJSON decodes meta:<key> into out; found is false when absent.
func (s *Store) GetMetaJSON(key string, out any) (found bool, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixMeta + key))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return item.Value(func(val []byte) error { return json.Unmarshal(val, out) })
	})
	return found, err
}

// --- Alias attestations ---

// prefixAttest keys the episodes that have claimed a given alias for a
// given entity: att:<slug>:<normalized alias> → JSON list of episode ids.
// An alias that would merge two existing entities is only admitted once
// two independent episodes have attested it.
const prefixAttest = "att:"

// maxAttestations bounds the list; two is the threshold, the rest is
// evidence for a human reading the store.
const maxAttestations = 8

// AttestAlias records that episodeID claimed alias norm for slug and
// returns the number of distinct episodes that have done so.
func (s *Store) AttestAlias(slug, norm, episodeID string) (int, error) {
	key := []byte(prefixAttest + slug + ":" + norm)
	var eps []string
	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err == nil {
			if err := item.Value(func(val []byte) error { return json.Unmarshal(val, &eps) }); err != nil {
				return err
			}
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		for _, e := range eps {
			if e == episodeID {
				return nil
			}
		}
		if len(eps) < maxAttestations {
			eps = append(eps, episodeID)
		}
		b, err := json.Marshal(eps)
		if err != nil {
			return err
		}
		return txn.Set(key, b)
	})
	return len(eps), err
}

// AliasAttestations lists the episodes that claimed alias norm for slug.
func (s *Store) AliasAttestations(slug, norm string) ([]string, error) {
	var eps []string
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixAttest + slug + ":" + norm))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error { return json.Unmarshal(val, &eps) })
	})
	return eps, err
}
