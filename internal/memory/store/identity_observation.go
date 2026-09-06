package store

// PRIVATE UNCALLED prototype: immutable parsed occurrences, never graph routing
// or an admission controller. Materialization/disposition is a separate concern.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
)

const identityObservationPrefix = "io:"

var errIdentityObservation = errors.New("memory: observation precondition failed")
var errObservationPageTooSmall = errors.New("memory: observation exceeds page budget")

type observedDeclaration struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Description  string   `json:"description"`
	Aliases      []string `json:"aliases"`
	TypeFallback bool     `json:"type_fallback"`
}

type observedSupersedes struct {
	Src      string `json:"src"`
	Relation string `json:"relation"`
	Dst      string `json:"dst"`
}

type observedFact struct {
	Src        string              `json:"src"`
	Relation   string              `json:"relation"`
	Dst        string              `json:"dst"`
	Fact       string              `json:"fact"`
	ValidFrom  string              `json:"valid_from"`
	Confidence float64             `json:"confidence"`
	Supersedes *observedSupersedes `json:"supersedes"`
}

type identityObservation struct {
	Version     int                  `json:"version"`
	EpisodeID   string               `json:"episode_id"`
	Origin      string               `json:"origin"`
	Ordinal     int                  `json:"ordinal"`
	Side        string               `json:"side"`
	Cwd         string               `json:"cwd"`
	OccurredAt  time.Time            `json:"occurred_at"`
	Declaration *observedDeclaration `json:"declaration"`
	Fact        *observedFact        `json:"fact"`
}

type observationPage struct {
	Records []identityObservation `json:"records"`
	Next    string                `json:"next"`
}

func observationEpisodePrefix(id string) string {
	return identityObservationPrefix + generationDigest([]byte(id)) + ":"
}

func encodeIdentityObservation(o identityObservation) ([]byte, string, error) {
	if o.Version != 1 || o.EpisodeID == "" || !utf8.ValidString(o.EpisodeID) || o.Ordinal < 0 || o.OccurredAt.IsZero() {
		return nil, "", errIdentityObservation
	}
	switch o.Origin {
	case "declaration":
		if o.Side != "" || o.Declaration == nil || o.Fact != nil {
			return nil, "", errIdentityObservation
		}
	case "endpoint":
		if (o.Side != "src" && o.Side != "dst") || o.Fact == nil || o.Declaration != nil {
			return nil, "", errIdentityObservation
		}
	default:
		return nil, "", errIdentityObservation
	}
	o.OccurredAt = o.OccurredAt.UTC()
	raw, err := json.Marshal(o)
	if err != nil {
		return nil, "", errIdentityObservation
	}
	var decoded identityObservation
	// Equality checks every parsed field, including future text members, aliases
	// and supersession pointers. JSON's replacement of invalid UTF-8 is refusal,
	// not an accepted lossy encoding. UTC has no monotonic/location ambiguity.
	if err := json.Unmarshal(raw, &decoded); err != nil || !reflect.DeepEqual(o, decoded) {
		return nil, "", errIdentityObservation
	}
	key := observationEpisodePrefix(o.EpisodeID) + o.Origin + ":" + strconv.Itoa(o.Ordinal) + ":" + o.Side + ":" + generationDigest(raw)
	return raw, key, nil
}

func decodeIdentityObservation(key string, raw []byte, episodeID string) (identityObservation, error) {
	var o identityObservation
	if err := json.Unmarshal(raw, &o); err != nil {
		return o, errIdentityObservation
	}
	expected, expectedKey, err := encodeIdentityObservation(o)
	if err != nil || key != expectedKey || !bytes.Equal(raw, expected) || o.EpisodeID != episodeID {
		return identityObservation{}, errIdentityObservation
	}
	return o, nil
}

func observationProvenance(st *Store, o identityObservation) error {
	// GetEpisode's ordinary decoder accepts duplicate members by taking the
	// last value. Provenance must not select an identity from ambiguous bytes.
	return st.view(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte(prefixEpisode + o.EpisodeID))
		if err != nil {
			return errIdentityObservation
		}
		return item.Value(func(raw []byte) error {
			if !utf8.Valid(raw) {
				return errIdentityObservation
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			start, err := dec.Token()
			if err != nil || start != json.Delim('{') {
				return errIdentityObservation
			}
			seenID, seenTime := false, false
			for dec.More() {
				token, err := dec.Token()
				if err != nil {
					return errIdentityObservation
				}
				name, ok := token.(string)
				if !ok {
					return errIdentityObservation
				}
				var value json.RawMessage
				if err := dec.Decode(&value); err != nil {
					return errIdentityObservation
				}
				switch {
				case strings.EqualFold(name, "id"):
					if seenID {
						return errIdentityObservation
					}
					seenID = true
					var id string
					if err := json.Unmarshal(value, &id); err != nil || id != o.EpisodeID {
						return errIdentityObservation
					}
					canonical, _ := json.Marshal(id)
					if !bytes.Equal(value, canonical) {
						return errIdentityObservation
					}
				case strings.EqualFold(name, "occurred_at"):
					if seenTime {
						return errIdentityObservation
					}
					seenTime = true
					var at time.Time
					if err := json.Unmarshal(value, &at); err != nil || !at.Equal(o.OccurredAt) {
						return errIdentityObservation
					}
					canonical, err := json.Marshal(at)
					if err != nil || !bytes.Equal(value, canonical) {
						return errIdentityObservation
					}
				}
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim('}') || !seenID || !seenTime {
				return errIdentityObservation
			}
			var extra json.RawMessage
			if err := dec.Decode(&extra); err != io.EOF {
				return errIdentityObservation
			}
			return nil
		})
	})
}

func putIdentityObservation(st *Store, o identityObservation) (string, error) {
	if err := generationTransaction(st); err != nil {
		return "", err
	}
	raw, key, err := encodeIdentityObservation(o)
	if err != nil {
		return "", err
	}
	if err := observationProvenance(st, o); err != nil {
		return "", err
	}
	old, exists, err := generationRead(st, []byte(key))
	if err != nil {
		return "", err
	}
	if exists {
		if !bytes.Equal(old, raw) {
			return "", errIdentityObservation
		}
		return key, nil
	}
	// Both key and raw are newly owned buffers. No alias slice, string backing
	// buffer, or caller supersession pointer is retained by the staged write.
	if err := st.txn.Set([]byte(key), raw); err != nil {
		return "", err
	}
	return key, nil
}

// Page budget covers json.Marshal(observationPage), including the cursor and
// record envelope. It is not a guarantee about a future RPC wrapper's bytes.
// The cursor is the last returned exact key; it must still exist and validate.
// Oversized first records refuse visibly rather than being skipped/truncated.
func readIdentityObservations(st *Store, episodeID, cursor string, limit, maxBytes int) (observationPage, error) {
	page := observationPage{Records: []identityObservation{}}
	if episodeID == "" || !utf8.ValidString(episodeID) || limit < 1 || limit > 100 || maxBytes < 32 || maxBytes > 1<<20 {
		return page, errIdentityObservation
	}
	prefix := observationEpisodePrefix(episodeID)
	if cursor != "" && !strings.HasPrefix(cursor, prefix) {
		return page, errIdentityObservation
	}
	err := st.view(func(tx *badger.Txn) error {
		reader := &Store{db: st.db, txn: tx}
		if cursor != "" {
			raw, exists, err := generationRead(reader, []byte(cursor))
			if err != nil {
				return err
			}
			if !exists {
				return errIdentityObservation
			}
			if _, err := decodeIdentityObservation(cursor, raw, episodeID); err != nil {
				return err
			}
		}
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		start := prefix
		if cursor != "" {
			start = cursor
		}
		it.Seek([]byte(start))
		if cursor != "" && it.ValidForPrefix([]byte(prefix)) && bytes.Equal(it.Item().Key(), []byte(cursor)) {
			it.Next()
		}
		for ; it.ValidForPrefix([]byte(prefix)); it.Next() {
			key := string(it.Item().KeyCopy(nil))
			raw, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			o, err := decodeIdentityObservation(key, raw, episodeID)
			if err != nil {
				return err
			}
			if err := observationProvenance(reader, o); err != nil {
				return err
			}
			candidate := observationPage{Records: append(append([]identityObservation{}, page.Records...), o), Next: key}
			encoded, err := json.Marshal(candidate)
			if err != nil {
				return errIdentityObservation
			}
			if len(encoded) > maxBytes {
				if len(page.Records) == 0 {
					return fmt.Errorf("%w: record key sha256=%s", errObservationPageTooSmall, generationDigest([]byte(key)))
				}
				return nil
			}
			page = candidate
			if len(page.Records) == limit {
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
		return observationPage{}, err
	}
	return page, nil
}
