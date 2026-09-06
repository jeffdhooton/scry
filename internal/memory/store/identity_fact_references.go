package store

// PRIVATE, UNCALLED read-only reference checker. Reference existence is not
// ownership, provenance closure, admission policy or a cleanup authorization.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
)

var errIdentityFactReferences = errors.New("memory: ambiguous fact reference")

type identityReferenceCount struct {
	Current    uint64
	Historical uint64
}

type identityReferenceReport struct {
	Scanned    uint64
	References map[string]identityReferenceCount
}

// Read all known fields separately: unmarshaling a Fact would silently choose
// the last duplicate member. Unknown extensions remain opaque and untouched.
func decodeIdentityReference(key, raw []byte) (Fact, error) {
	fail := func() (Fact, error) {
		return Fact{}, fmt.Errorf("%w: key sha256=%s", errIdentityFactReferences, generationDigest(key))
	}
	if !utf8.Valid(raw) || !utf8.Valid(key) || !referenceStringEscapesValid(raw) {
		return fail()
	}
	var f Fact
	fields := map[string]any{
		"src": &f.Src, "relation": &f.Relation, "dst": &f.Dst,
		"value": &f.Value, "raw_relation": &f.RawRelation, "fact": &f.Fact,
		"valid_from": &f.ValidFrom, "invalid_at": &f.InvalidAt,
		"confidence": &f.Confidence, "episodes": &f.Episodes,
	}
	seen := make(map[string]bool, len(fields))
	dec := json.NewDecoder(bytes.NewReader(raw))
	start, err := dec.Token()
	if err != nil || start != json.Delim('{') {
		return fail()
	}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return fail()
		}
		name, ok := token.(string)
		if !ok {
			return fail()
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return fail()
		}
		for known, target := range fields {
			if !strings.EqualFold(name, known) {
				continue
			}
			if seen[known] {
				return fail()
			}
			seen[known] = true
			if err := json.Unmarshal(value, target); err != nil {
				return fail()
			}
			canonical, err := json.Marshal(target)
			if err != nil {
				return fail()
			}
			var compact bytes.Buffer
			if err := json.Compact(&compact, value); err != nil || !bytes.Equal(compact.Bytes(), canonical) {
				return fail()
			}
			break
		}
	}
	end, err := dec.Token()
	if err != nil || end != json.Delim('}') {
		return fail()
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		return fail()
	}
	for _, required := range []string{"src", "relation", "dst", "fact", "valid_from", "confidence", "episodes"} {
		if !seen[required] {
			return fail()
		}
	}
	if !validEntitySlug(f.Src) || (f.Dst != "" && !validEntitySlug(f.Dst)) || f.Relation == "" || strings.Contains(f.Relation, ":") || (f.Dst == "") == (f.Value == "") {
		return fail()
	}
	// UnixNano overflows silently outside its supported interval. A matching
	// overflowed address cannot prove this fact's temporal identity.
	if !time.Unix(0, f.ValidFrom.UnixNano()).Equal(f.ValidFrom) || !bytes.Equal(key, factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom)) {
		return fail()
	}
	if f.InvalidAt != nil && !time.Unix(0, f.InvalidAt.UnixNano()).Equal(*f.InvalidAt) {
		return fail()
	}
	return f, nil
}

// JSON permits escaped UTF-16 surrogate pairs, but encoding/json substitutes
// U+FFFD for unmatched halves. Validate string syntax across unknown extensions
// too, without decoding their values or changing their original bytes. The JSON
// decoder below remains responsible for the complete document grammar.
func referenceStringEscapesValid(raw []byte) bool {
	inside := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inside = !inside
			continue
		}
		if !inside || raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		unit, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if unit >= 0xdc00 && unit <= 0xdfff {
			return false
		}
		if unit < 0xd800 || unit > 0xdbff {
			continue
		}
		if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return false
		}
		low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return false
		}
		i += 6
	}
	return !inside
}

func scanIdentityFactReferences(st *Store, slugs []string) (identityReferenceReport, error) {
	if st == nil {
		return identityReferenceReport{}, errIdentityFactReferences
	}
	if st.txn != nil {
		if err := generationTransaction(st); err != nil {
			return identityReferenceReport{}, errIdentityFactReferences
		}
	}
	report := identityReferenceReport{References: make(map[string]identityReferenceCount, len(slugs))}
	for _, slug := range slugs {
		if !utf8.ValidString(slug) || !validEntitySlug(slug) {
			return identityReferenceReport{}, errIdentityFactReferences
		}
		report.References[slug] = identityReferenceCount{}
	}
	err := st.view(func(tx *badger.Txn) error {
		prefix := []byte(prefixFact)
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			var f Fact
			err := it.Item().Value(func(raw []byte) error {
				var err error
				f, err = decodeIdentityReference(key, raw)
				return err
			})
			if err != nil {
				return err
			}
			report.Scanned++
			for side, slug := range []string{f.Src, f.Dst} {
				if slug == "" || (side == 1 && slug == f.Src) {
					continue
				}
				count, requested := report.References[slug]
				if !requested {
					continue
				}
				if f.InvalidAt == nil {
					count.Current++
				} else {
					count.Historical++
				}
				report.References[slug] = count
			}
		}
		return nil
	})
	if err != nil {
		// Decoder refusals are already redacted. DB failures must not accidentally
		// expose raw value material through a lower-level diagnostic.
		if errors.Is(err, errIdentityFactReferences) {
			return identityReferenceReport{}, err
		}
		return identityReferenceReport{}, errIdentityFactReferences
	}
	return report, nil
}
