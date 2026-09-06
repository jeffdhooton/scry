package store

// PRIVATE, UNCALLED preservation codec. Descriptive outcomes are not support,
// identity authority, current-state selection or admission policy.

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
)

const identityOutcomePrefix = "io-result:"

var errIdentityOutcome = errors.New("memory: invalid identity outcome")

type outcomeLink struct {
	Key          string `json:"key"`
	Role         string `json:"role"`
	FinalSide    string `json:"final_side"`
	Spelling     string `json:"spelling"`
	ResolvedSlug string `json:"resolved_slug"`
	State        string `json:"state"`
}

type identityOutcome struct {
	Version          int            `json:"version"`
	EpisodeID        string         `json:"episode_id"`
	Birth            *identityBirth `json:"birth"`
	AssertionOrdinal *int           `json:"assertion_ordinal"`
	Disposition      string         `json:"disposition"`
	Links            []outcomeLink  `json:"links"`
	Materialization  []byte         `json:"materialization"`
	Generation       string         `json:"generation"`
	Predecessor      string         `json:"predecessor"`
}

type outcomeKeyPage struct {
	Keys []string `json:"keys"`
	Next string   `json:"next"`
}

type outcomeChunk struct {
	Key        string `json:"key"`
	Offset     int    `json:"offset"`
	NextOffset int    `json:"next_offset"`
	Data       []byte `json:"data"`
}

func outcomeEpisodePrefix(id string) string {
	return identityOutcomePrefix + generationDigest([]byte(id)) + ":"
}

func encodeIdentityOutcome(o identityOutcome) ([]byte, string, error) {
	if o.Version != 1 || o.EpisodeID == "" || !utf8.ValidString(o.EpisodeID) || (o.Birth == nil) == (o.AssertionOrdinal == nil) || len(o.Links) == 0 {
		return nil, "", errIdentityOutcome
	}
	if o.Birth != nil {
		record, _, err := identityBirthRecord(*o.Birth)
		if err != nil || record.Birth.EpisodeID != o.EpisodeID {
			return nil, "", errIdentityOutcome
		}
		o.Birth = &record.Birth // Own the normalized value; do not alter caller input.
		switch o.Disposition {
		case "supported":
			if len(o.Materialization) == 0 || o.Generation != record.ID {
				return nil, "", errIdentityOutcome
			}
		case "no-assertion":
			if o.Generation != "" {
				return nil, "", errIdentityOutcome
			}
		default:
			return nil, "", errIdentityOutcome
		}
	} else if *o.AssertionOrdinal < 0 || (o.Disposition != "deferred" && o.Disposition != "committed") || o.Materialization != nil || o.Generation != "" {
		return nil, "", errIdentityOutcome
	}
	if o.Materialization != nil {
		trimmed := bytes.TrimSpace(o.Materialization)
		if len(trimmed) == 0 || trimmed[0] != '{' || !utf8.Valid(o.Materialization) || !json.Valid(o.Materialization) || !referenceStringEscapesValid(o.Materialization) {
			return nil, "", errIdentityOutcome
		}
	}
	seen := map[string]bool{}
	deferred := false
	for _, l := range o.Links {
		if l.Key == "" || seen[l.Key+"\x00"+l.Role] {
			return nil, "", errIdentityOutcome
		}
		seen[l.Key+"\x00"+l.Role] = true
		switch l.Role {
		case "declaration", "primary-src", "primary-dst", "supersedes-src", "supersedes-dst":
		default:
			return nil, "", errIdentityOutcome
		}
		switch l.FinalSide {
		case "src", "dst", "value", "none":
		default:
			return nil, "", errIdentityOutcome
		}
		switch l.State {
		case "resolved":
			if l.FinalSide == "none" || (l.FinalSide == "value" && l.ResolvedSlug != "") || ((l.FinalSide == "src" || l.FinalSide == "dst") && !validEntitySlug(l.ResolvedSlug)) {
				return nil, "", errIdentityOutcome
			}
		case "deferred", "not-materialized":
			if l.ResolvedSlug != "" || l.FinalSide != "none" {
				return nil, "", errIdentityOutcome
			}
			deferred = deferred || l.State == "deferred"
		default:
			return nil, "", errIdentityOutcome
		}
	}
	if o.AssertionOrdinal != nil && ((o.Disposition == "deferred" && !deferred) || (o.Disposition == "committed" && deferred)) {
		return nil, "", errIdentityOutcome
	}
	raw, err := json.Marshal(o)
	if err != nil {
		return nil, "", errIdentityOutcome
	}
	var decoded identityOutcome
	if err := json.Unmarshal(raw, &decoded); err != nil || !reflect.DeepEqual(o, decoded) {
		return nil, "", errIdentityOutcome
	}
	return raw, outcomeEpisodePrefix(o.EpisodeID) + generationDigest(raw), nil
}

func decodeIdentityOutcome(key string, raw []byte, episodeID string) (identityOutcome, error) {
	var o identityOutcome
	if err := json.Unmarshal(raw, &o); err != nil {
		return identityOutcome{}, errIdentityOutcome
	}
	canonical, wantKey, err := encodeIdentityOutcome(o)
	if err != nil || o.EpisodeID != episodeID || key != wantKey || !bytes.Equal(raw, canonical) {
		return identityOutcome{}, errIdentityOutcome
	}
	return o, nil
}

func validateOutcomeLinks(st *Store, o identityOutcome) error {
	birthLinked := o.Birth == nil
	for _, l := range o.Links {
		raw, exists, err := generationRead(st, []byte(l.Key))
		if err != nil || !exists {
			return errIdentityOutcome
		}
		input, err := decodeIdentityObservation(l.Key, raw, o.EpisodeID)
		if err != nil || observationProvenance(st, input) != nil {
			return errIdentityOutcome
		}
		spelling := ""
		switch l.Role {
		case "declaration":
			if input.Declaration == nil || input.Origin != "declaration" {
				return errIdentityOutcome
			}
			spelling = input.Declaration.Name
		case "primary-src", "primary-dst":
			if input.Fact == nil || input.Side != strings.TrimPrefix(l.Role, "primary-") {
				return errIdentityOutcome
			}
			spelling = input.Fact.Src
			if l.Role == "primary-dst" {
				spelling = input.Fact.Dst
			}
		case "supersedes-src", "supersedes-dst":
			if input.Fact == nil || input.Fact.Supersedes == nil {
				return errIdentityOutcome
			}
			spelling = input.Fact.Supersedes.Src
			if l.Role == "supersedes-dst" {
				spelling = input.Fact.Supersedes.Dst
			}
		}
		if spelling != l.Spelling {
			return errIdentityOutcome
		}
		if o.AssertionOrdinal != nil && (input.Fact == nil || input.Ordinal != *o.AssertionOrdinal) {
			return errIdentityOutcome
		}
		if o.Birth != nil && input.Ordinal == o.Birth.Occurrence && input.Origin == o.Birth.Origin && spelling == o.Birth.Name && input.OccurredAt.Equal(o.Birth.CreatedAt) && (l.Role == "declaration" || l.Role == "primary-src" || l.Role == "primary-dst") {
			birthLinked = true
		}
	}
	if !birthLinked {
		return errIdentityOutcome
	}
	return nil
}

func outcomeLineage(o identityOutcome) []byte {
	type inputLink struct{ Key, Role string }
	links := make([]inputLink, len(o.Links))
	for i, l := range o.Links {
		links[i] = inputLink{l.Key, l.Role}
	}
	raw, _ := json.Marshal(struct {
		EpisodeID        string
		Birth            *identityBirth
		AssertionOrdinal *int
		Links            []inputLink
	}{o.EpisodeID, o.Birth, o.AssertionOrdinal, links})
	return raw
}

func readOutcomeRow(st *Store, key, episodeID string, predecessor bool) (identityOutcome, []byte, error) {
	raw, exists, err := generationRead(st, []byte(key))
	if err != nil || !exists {
		return identityOutcome{}, nil, errIdentityOutcome
	}
	o, err := decodeIdentityOutcome(key, raw, episodeID)
	if err != nil || validateOutcomeLinks(st, o) != nil {
		return identityOutcome{}, nil, errIdentityOutcome
	}
	if predecessor && o.Predecessor != "" {
		previous, _, err := readOutcomeRow(st, o.Predecessor, episodeID, false)
		if err != nil || !bytes.Equal(outcomeLineage(previous), outcomeLineage(o)) {
			return identityOutcome{}, nil, errIdentityOutcome
		}
	}
	return o, raw, nil
}

func putIdentityOutcome(st *Store, o identityOutcome) (string, error) {
	if generationTransaction(st) != nil {
		return "", errIdentityOutcome
	}
	raw, key, err := encodeIdentityOutcome(o)
	if err != nil {
		return "", err
	}
	// Use the encoded canonical copy: caller timezone representation is not
	// allowed to change predecessor identity or staged ownership.
	canonical, err := decodeIdentityOutcome(key, raw, o.EpisodeID)
	if err != nil || validateOutcomeLinks(st, canonical) != nil {
		return "", errIdentityOutcome
	}
	if canonical.Predecessor != "" {
		previous, _, err := readOutcomeRow(st, canonical.Predecessor, o.EpisodeID, false)
		if err != nil || !bytes.Equal(outcomeLineage(previous), outcomeLineage(canonical)) {
			return "", errIdentityOutcome
		}
	}
	old, exists, err := generationRead(st, []byte(key))
	if err != nil {
		return "", errIdentityOutcome
	}
	if exists {
		if !bytes.Equal(old, raw) {
			return "", errIdentityOutcome
		}
		return key, nil
	}
	if err := st.txn.Set([]byte(key), bytes.Clone(raw)); err != nil {
		// Some storage refusals include a hex/ASCII dump of the submitted value.
		// Preserve only an allowlisted static classification, never the raw error.
		if errors.Is(err, badger.ErrTxnTooBig) {
			return "", errors.Join(errIdentityOutcome, badger.ErrTxnTooBig)
		}
		return "", errIdentityOutcome
	}
	return key, nil
}

func outcomeView(st *Store, fn func(*Store) error) error {
	if st == nil {
		return errIdentityOutcome
	}
	if st.txn != nil && generationTransaction(st) != nil {
		return errIdentityOutcome
	}
	err := st.view(func(tx *badger.Txn) error { return fn(&Store{db: st.db, txn: tx}) })
	if err != nil {
		return errIdentityOutcome
	}
	return nil
}

func listIdentityOutcomeKeys(st *Store, episodeID, cursor string, limit, maxBytes int) (outcomeKeyPage, error) {
	page := outcomeKeyPage{Keys: []string{}}
	if episodeID == "" || !utf8.ValidString(episodeID) || limit < 1 || limit > 100 || maxBytes < 512 || maxBytes > 24576 {
		return outcomeKeyPage{}, errIdentityOutcome
	}
	prefix := outcomeEpisodePrefix(episodeID)
	err := outcomeView(st, func(tx *Store) error {
		if cursor != "" {
			if _, _, err := readOutcomeRow(tx, cursor, episodeID, true); err != nil {
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
			if _, _, err := readOutcomeRow(tx, key, episodeID, true); err != nil {
				return err
			}
			candidate := outcomeKeyPage{Keys: append(append([]string{}, page.Keys...), key), Next: key}
			raw, _ := json.Marshal(candidate)
			if len(raw) > maxBytes {
				if len(page.Keys) == 0 {
					return errIdentityOutcome
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
		return outcomeKeyPage{}, err
	}
	return page, nil
}

func readIdentityOutcomeChunk(st *Store, episodeID, key string, offset, maxBytes int) (outcomeChunk, error) {
	if episodeID == "" || !utf8.ValidString(episodeID) || offset < 0 || maxBytes < 512 || maxBytes > 24576 {
		return outcomeChunk{}, errIdentityOutcome
	}
	var chunk outcomeChunk
	err := outcomeView(st, func(tx *Store) error {
		_, raw, err := readOutcomeRow(tx, key, episodeID, true)
		if err != nil || offset > len(raw) {
			return errIdentityOutcome
		}
		chunk = outcomeChunk{Key: key, Offset: offset, NextOffset: -1, Data: []byte{}}
		// Binary search an exact byte chunk whose complete JSON/base64
		// envelope fits. Chunks are bytes, not independently valid UTF-8 strings.
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
			return errIdentityOutcome
		}
		chunk.Data = bytes.Clone(raw[offset : offset+lo])
		if offset+lo < len(raw) {
			chunk.NextOffset = offset + lo
		}
		encoded, _ := json.Marshal(chunk)
		if len(encoded) > maxBytes {
			return errIdentityOutcome
		}
		return nil
	})
	if err != nil {
		return outcomeChunk{}, err
	}
	return chunk, nil
}
