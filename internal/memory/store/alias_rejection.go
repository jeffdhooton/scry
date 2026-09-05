package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

// Additive, intentionally not a SchemaVersion change: old stores have no
// decisions. Older writers ignore these records and must not be used with a
// post-repair store. Only explicit reviewed repair transactions create them.
const prefixAliasRejection = "ar:"

var ErrAliasRejected = errors.New("memory: alias rejected for this entity")

// AliasRejection is negative identity evidence, not a global spelling ban.
// Plan pins the exact reviewed dispositions; multiple literal variants and
// their individual reasons remain visible instead of overwriting evidence.
type AliasRejection struct {
	Entity string `json:"entity"`
	Alias  string `json:"alias"`
	Why    string `json:"why"`
	Plan   string `json:"plan"`
}

func aliasRejectionKey(slug, norm string) string {
	return prefixAliasRejection + slug + ":" + norm
}

func decodeAliasRejections(key string, value []byte) ([]AliasRejection, error) {
	var rows []AliasRejection
	if err := json.Unmarshal(value, &rows); err != nil {
		return nil, fmt.Errorf("memory: malformed alias rejection %q: %w", key, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("memory: empty alias rejection %q", key)
	}
	for _, row := range rows {
		if !validEntitySlug(row.Entity) || Normalize(row.Alias) == "" || strings.TrimSpace(row.Why) == "" || row.Plan == "" || aliasRejectionKey(row.Entity, Normalize(row.Alias)) != key {
			return nil, fmt.Errorf("memory: invalid alias rejection %q", key)
		}
	}
	return rows, nil
}

func aliasRejectedTxn(txn *badger.Txn, slug, norm string) (bool, error) {
	key := aliasRejectionKey(slug, norm)
	item, err := txn.Get([]byte(key))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = item.Value(func(value []byte) error { _, err := decodeAliasRejections(key, value); return err })
	return err == nil, err
}

func checkAliasRejectionTxn(txn *badger.Txn, slug, norm string) error {
	rejected, err := aliasRejectedTxn(txn, slug, norm)
	if err != nil {
		return err
	}
	if rejected {
		return fmt.Errorf("%w: %q on %q", ErrAliasRejected, norm, slug)
	}
	return nil
}

// IsAliasRejected observes the current transaction when called by resolution.
func (s *Store) IsAliasRejected(slug, alias string) (bool, error) {
	var rejected bool
	err := s.view(func(txn *badger.Txn) error {
		var err error
		rejected, err = aliasRejectedTxn(txn, slug, Normalize(alias))
		return err
	})
	return rejected, err
}

// Full scans are confined to reviewed maintenance, never ordinary admission.
func aliasRejectionsTxn(txn *badger.Txn) (map[string][]AliasRejection, error) {
	out := map[string][]AliasRejection{}
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	prefix := []byte(prefixAliasRejection)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := string(it.Item().KeyCopy(nil))
		if err := it.Item().Value(func(value []byte) error {
			rows, err := decodeAliasRejections(key, value)
			if err == nil {
				out[key] = rows
			}
			return err
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func rejectionSubset(all map[string][]AliasRejection, owners map[string]bool) map[string][]AliasRejection {
	out := map[string][]AliasRejection{}
	for key, rows := range all {
		if owners[rows[0].Entity] {
			out[key] = rows
		}
	}
	return out
}

func addAliasRejection(all map[string][]AliasRejection, row AliasRejection) {
	key := aliasRejectionKey(row.Entity, Normalize(row.Alias))
	for _, existing := range all[key] {
		if existing == row {
			return
		}
	}
	all[key] = append(all[key], row)
}

func cloneAliasRejections(all map[string][]AliasRejection) map[string][]AliasRejection {
	out := map[string][]AliasRejection{}
	for key, rows := range all {
		out[key] = append([]AliasRejection(nil), rows...)
	}
	return out
}

// Called only while the repair's maintenance lock excludes marker writers.
// Exact full-state verification also proves no prior decision was lost.
func writeAliasRejectionsTxn(txn *badger.Txn, before, after map[string][]AliasRejection) error {
	for key, rows := range before {
		for _, old := range rows {
			found := false
			for _, current := range after[key] {
				found = found || old == current
			}
			if !found {
				return errors.New("memory: alias repair discarded rejection evidence")
			}
		}
	}
	for key, rows := range after {
		if reflect.DeepEqual(rows, before[key]) {
			continue
		}
		value, err := json.Marshal(rows)
		if err != nil {
			return err
		}
		if _, err := decodeAliasRejections(key, value); err != nil {
			return err
		}
		if err := txn.Set([]byte(key), value); err != nil {
			return err
		}
	}
	actual, err := aliasRejectionsTxn(txn)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, after) {
		return errors.New("memory: alias rejection postcondition failed")
	}
	return nil
}
