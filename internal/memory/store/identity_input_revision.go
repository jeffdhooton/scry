package store

// PRIVATE, UNCALLED structured input preservation. No current selection,
// registration completeness, support authority or production routing.

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
)

const identityInputPrefix = "io-input:"

var errIdentityInput = errors.New("memory: invalid identity input revision")

type identityInputRevision struct {
	Version      int                   `json:"version"`
	EpisodeID    string                `json:"episode_id"`
	OccurredAt   time.Time             `json:"occurred_at"`
	Cwd          string                `json:"cwd"`
	Summary      string                `json:"summary"`
	Declarations []observedDeclaration `json:"declarations"`
	Facts        []observedFact        `json:"facts"`
}

type inputKeyPage struct {
	Keys []string `json:"keys"`
	Next string   `json:"next"`
}

type inputChunk struct {
	Key        string `json:"key"`
	Offset     int    `json:"offset"`
	NextOffset int    `json:"next_offset"`
	Data       []byte `json:"data"`
}

func inputEpisodePrefix(id string) string {
	return identityInputPrefix + generationDigest([]byte(id)) + ":"
}

func encodeIdentityInput(r identityInputRevision) ([]byte, string, error) {
	if r.Version != 1 || r.EpisodeID == "" || !utf8.ValidString(r.EpisodeID) || r.OccurredAt.IsZero() {
		return nil, "", errIdentityInput
	}
	r.OccurredAt = r.OccurredAt.UTC()
	if !time.Unix(0, r.OccurredAt.UnixNano()).Equal(r.OccurredAt) {
		return nil, "", errIdentityInput
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, "", errIdentityInput
	}
	var decoded identityInputRevision
	if json.Unmarshal(raw, &decoded) != nil || !reflect.DeepEqual(r, decoded) {
		return nil, "", errIdentityInput
	}
	return raw, inputEpisodePrefix(r.EpisodeID) + generationDigest(raw), nil
}

func decodeIdentityInput(key string, raw []byte, episodeID string) (identityInputRevision, error) {
	var r identityInputRevision
	if json.Unmarshal(raw, &r) != nil {
		return identityInputRevision{}, errIdentityInput
	}
	canonical, wantKey, err := encodeIdentityInput(r)
	if err != nil || r.EpisodeID != episodeID || key != wantKey || !bytes.Equal(raw, canonical) {
		return identityInputRevision{}, errIdentityInput
	}
	return r, nil
}

// Match full canonical bytes, not just episode/ordinal or hash equality. This
// deliberately supplies no claim about result completeness or role authority.
func matchRevisionObservation(revisionKey string, revisionRaw []byte, observationKey string, observationRaw []byte, episodeID string) error {
	r, err := decodeIdentityInput(revisionKey, revisionRaw, episodeID)
	if err != nil {
		return errIdentityInput
	}
	o, err := decodeIdentityObservation(observationKey, observationRaw, episodeID)
	if err != nil {
		return errIdentityInput
	}
	expected := identityObservation{Version: 1, EpisodeID: r.EpisodeID, OccurredAt: r.OccurredAt, Cwd: r.Cwd, Origin: o.Origin, Ordinal: o.Ordinal, Side: o.Side}
	switch o.Origin {
	case "declaration":
		if o.Ordinal >= len(r.Declarations) {
			return errIdentityInput
		}
		expected.Declaration = &r.Declarations[o.Ordinal]
	case "endpoint":
		if o.Ordinal >= len(r.Facts) {
			return errIdentityInput
		}
		expected.Fact = &r.Facts[o.Ordinal]
	default:
		return errIdentityInput
	}
	raw, key, err := encodeIdentityObservation(expected)
	if err != nil || key != observationKey || !bytes.Equal(raw, observationRaw) {
		return errIdentityInput
	}
	return nil
}

func identityInputFailure(st *Store, err error) error {
	safe := error(errIdentityInput)
	for _, kind := range []error{badger.ErrTxnTooBig, badger.ErrConflict} {
		if errors.Is(err, kind) {
			safe = errors.Join(errIdentityInput, kind)
			break
		}
	}
	if st != nil && st.admissionOwner != nil {
		st.poisonAdmission(safe)
	}
	return safe
}

func inputProvenance(st *Store, r identityInputRevision) error {
	return observationProvenance(st, identityObservation{EpisodeID: r.EpisodeID, OccurredAt: r.OccurredAt})
}

func putIdentityInput(st *Store, r identityInputRevision) (string, error) {
	fail := func(err error) (string, error) { return "", identityInputFailure(st, err) }
	if err := generationTransaction(st); err != nil {
		return fail(err)
	}
	if st.admissionFailure != nil {
		return fail(errIdentityInput)
	}
	raw, key, err := encodeIdentityInput(r)
	if err != nil {
		return fail(err)
	}
	if err := inputProvenance(st, r); err != nil {
		return fail(err)
	}
	old, exists, err := generationRead(st, []byte(key))
	if err != nil {
		return fail(err)
	}
	if exists {
		if !bytes.Equal(old, raw) {
			return fail(errIdentityInput)
		}
		return key, nil
	}
	if err := st.txn.Set([]byte(key), raw); err != nil {
		return fail(err)
	}
	return key, nil
}

func readInputRow(st *Store, key, episodeID string) (identityInputRevision, []byte, error) {
	raw, exists, err := generationRead(st, []byte(key))
	if err != nil || !exists {
		return identityInputRevision{}, nil, errIdentityInput
	}
	r, err := decodeIdentityInput(key, raw, episodeID)
	if err != nil || inputProvenance(st, r) != nil {
		return identityInputRevision{}, nil, errIdentityInput
	}
	return r, raw, nil
}

func inputView(st *Store, fn func(*Store) error) error {
	if st == nil || st.db == nil || (st.txn != nil && generationTransaction(st) != nil) {
		return errIdentityInput
	}
	if err := st.view(func(tx *badger.Txn) error { return fn(&Store{db: st.db, txn: tx}) }); err != nil {
		return errIdentityInput
	}
	return nil
}

func listIdentityInputKeys(st *Store, episodeID, cursor string, limit, maxBytes int) (inputKeyPage, error) {
	if episodeID == "" || !utf8.ValidString(episodeID) || limit < 1 || limit > 100 || maxBytes < 512 || maxBytes > 24576 {
		return inputKeyPage{}, errIdentityInput
	}
	page := inputKeyPage{Keys: []string{}}
	prefix := inputEpisodePrefix(episodeID)
	err := inputView(st, func(tx *Store) error {
		if cursor != "" {
			if _, _, err := readInputRow(tx, cursor, episodeID); err != nil {
				return err
			}
		}
		it := tx.txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		start := prefix
		if cursor != "" {
			start = cursor
		}
		it.Seek([]byte(start))
		if cursor != "" && it.Valid() && string(it.Item().Key()) == cursor {
			it.Next()
		}
		for ; it.ValidForPrefix([]byte(prefix)); it.Next() {
			key := string(it.Item().KeyCopy(nil))
			if _, _, err := readInputRow(tx, key, episodeID); err != nil {
				return err
			}
			candidate := inputKeyPage{Keys: append(append([]string{}, page.Keys...), key), Next: key}
			raw, _ := json.Marshal(candidate)
			if len(raw) > maxBytes {
				if len(page.Keys) == 0 {
					return errIdentityInput
				}
				return nil
			}
			page = candidate
			if len(page.Keys) == limit {
				it.Next()
				if !it.ValidForPrefix([]byte(prefix)) {
					page.Next = ""
				}
				return nil
			}
		}
		page.Next = ""
		return nil
	})
	if err != nil {
		return inputKeyPage{}, err
	}
	return page, nil
}

func readIdentityInputChunk(st *Store, episodeID, key string, offset, maxBytes int) (inputChunk, error) {
	if episodeID == "" || !utf8.ValidString(episodeID) || offset < 0 || maxBytes < 512 || maxBytes > 24576 {
		return inputChunk{}, errIdentityInput
	}
	var chunk inputChunk
	err := inputView(st, func(tx *Store) error {
		_, raw, err := readInputRow(tx, key, episodeID)
		if err != nil || offset > len(raw) {
			return errIdentityInput
		}
		chunk = inputChunk{Key: key, Offset: offset, NextOffset: -1, Data: []byte{}}
		lo, hi := 0, len(raw)-offset
		if hi > maxBytes {
			hi = maxBytes
		}
		for lo < hi {
			mid := (lo + hi + 1) / 2
			trial := chunk
			trial.Data = raw[offset : offset+mid]
			if offset+mid < len(raw) {
				trial.NextOffset = offset + mid
			}
			encoded, _ := json.Marshal(trial)
			if len(encoded) <= maxBytes {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		if lo == 0 && offset < len(raw) {
			return errIdentityInput
		}
		chunk.Data = bytes.Clone(raw[offset : offset+lo])
		if offset+lo < len(raw) {
			chunk.NextOffset = offset + lo
		}
		encoded, _ := json.Marshal(chunk)
		if len(encoded) > maxBytes {
			return errIdentityInput
		}
		return nil
	})
	if err != nil {
		return inputChunk{}, err
	}
	return chunk, nil
}
