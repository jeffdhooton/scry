package store

// PRIVATE, UNCALLED. This transaction and reader do not install admission policy.
// Live adoption is forbidden until every writer enforces the identity lifecycle.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/dgraph-io/badger/v4"
)

const legacyAdoptionMarkerKey = "meta:identity_adoption_v1"

var errLegacyAdoption = errors.New("memory: invalid legacy identity adoption")
var errLegacyAdoptionPreview = errors.New("memory: legacy adoption preview rollback")

type legacyInventoryEntry struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

type legacyInventoryManifest struct {
	Version int                    `json:"version"`
	Entries []legacyInventoryEntry `json:"entries"`
}

type legacyAdoptionMarker struct {
	Version   int    `json:"version"`
	Inventory string `json:"inventory"`
	Entities  int    `json:"entities"`
}

type legacyAdoptionReport struct {
	Inventory string `json:"inventory"`
	Entities  int    `json:"entities"`
	Bytes     uint64 `json:"bytes"`
	Applied   bool   `json:"applied"`
}

func legacyAdoptionFailure(err error) error {
	for _, safe := range []error{badger.ErrTxnTooBig, badger.ErrConflict} {
		if errors.Is(err, safe) {
			return errors.Join(errLegacyAdoption, safe)
		}
	}
	return errLegacyAdoption
}

func decodeLegacyInventory(raw []byte) (legacyInventoryManifest, string, error) {
	var m legacyInventoryManifest
	if json.Unmarshal(raw, &m) != nil || m.Version != 1 || m.Entries == nil {
		return legacyInventoryManifest{}, "", errLegacyAdoption
	}
	canonical, err := json.Marshal(m)
	if err != nil || !bytes.Equal(raw, canonical) {
		return legacyInventoryManifest{}, "", errLegacyAdoption
	}
	for i, entry := range m.Entries {
		if _, err := decodeLegacyEntity(entry.Key, entry.Value); err != nil || (i > 0 && bytes.Compare(m.Entries[i-1].Key, entry.Key) >= 0) {
			return legacyInventoryManifest{}, "", errLegacyAdoption
		}
	}
	digest := sha256.Sum256(raw)
	return m, hex.EncodeToString(digest[:]), nil
}

func captureLegacyInventory(st *Store) ([]byte, error) {
	if st == nil {
		return nil, errLegacyAdoption
	}
	m := legacyInventoryManifest{Version: 1, Entries: []legacyInventoryEntry{}}
	err := st.view(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(prefixEntity)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			raw, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			if _, err := decodeLegacyEntity(key, raw); err != nil {
				return err
			}
			m.Entries = append(m.Entries, legacyInventoryEntry{Key: key, Value: raw})
		}
		return nil
	})
	if err != nil {
		return nil, legacyAdoptionFailure(err)
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, errLegacyAdoption
	}
	return raw, nil
}

func decodeLegacyAdoptionMarker(raw []byte) (legacyAdoptionMarker, error) {
	var m legacyAdoptionMarker
	if json.Unmarshal(raw, &m) != nil || m.Version != 1 || !legacyDigestValid(m.Inventory) || m.Entities < 0 {
		return legacyAdoptionMarker{}, errLegacyAdoption
	}
	canonical, err := json.Marshal(m)
	if err != nil || !bytes.Equal(raw, canonical) {
		return legacyAdoptionMarker{}, errLegacyAdoption
	}
	return m, nil
}

func requireLegacyAbsent(txn *badger.Txn, key []byte) error {
	_, err := txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil
	}
	return errLegacyAdoption
}

func preflightLegacyInventory(txn *badger.Txn, m legacyInventoryManifest) error {
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	for _, family := range []string{"il:", "il-consumed:", "ig:", "iga:", "io:", "io-result:", "io-input:", "meta:identity_"} {
		prefix := []byte(family)
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			return errLegacyAdoption
		}
	}
	prefix := []byte(prefixEntity)
	i := 0
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		if i >= len(m.Entries) || !bytes.Equal(it.Item().Key(), m.Entries[i].Key) {
			return errLegacyAdoption
		}
		raw, err := it.Item().ValueCopy(nil)
		if err != nil {
			return err
		}
		if !bytes.Equal(raw, m.Entries[i].Value) {
			return errLegacyAdoption
		}
		i++
	}
	if i != len(m.Entries) {
		return errLegacyAdoption
	}
	return nil
}

// Root-only, explicitly requested, dry-run by default at the caller. Refusing a
// facade poisons the surrounding scope even if its caller ignores this error.
func adoptLegacyInventory(st *Store, raw []byte, apply bool) (legacyAdoptionReport, error) {
	if st == nil {
		return legacyAdoptionReport{}, errLegacyAdoption
	}
	if st.txn != nil || st.admissionOwner != nil {
		st.poisonAdmission(errLegacyAdoption)
		return legacyAdoptionReport{}, errLegacyAdoption
	}
	m, inventory, err := decodeLegacyInventory(raw)
	if err != nil {
		return legacyAdoptionReport{}, err
	}
	marker, err := json.Marshal(legacyAdoptionMarker{Version: 1, Inventory: inventory, Entities: len(m.Entries)})
	if err != nil {
		return legacyAdoptionReport{}, errLegacyAdoption
	}
	report := legacyAdoptionReport{Inventory: inventory, Entities: len(m.Entries), Bytes: uint64(len(legacyAdoptionMarkerKey) + len(marker)), Applied: apply}
	st.maintenanceMu.Lock()
	defer st.maintenanceMu.Unlock()
	err = st.db.Update(func(txn *badger.Txn) error {
		if err := preflightLegacyInventory(txn, m); err != nil {
			return err
		}
		for _, entry := range m.Entries {
			key, value, err := makeLegacyIdentityAnchor(inventory, entry.Key, entry.Value)
			if err != nil {
				return err
			}
			if err := requireLegacyAbsent(txn, key); err != nil {
				return err
			}
			if err := txn.Set(key, value); err != nil {
				return err
			}
			report.Bytes += uint64(len(key) + len(value))
		}
		if err := requireLegacyAbsent(txn, []byte(legacyAdoptionMarkerKey)); err != nil {
			return err
		}
		if err := txn.Set([]byte(legacyAdoptionMarkerKey), marker); err != nil {
			return err
		}
		for _, entry := range m.Entries {
			key, value, err := makeLegacyIdentityAnchor(inventory, entry.Key, entry.Value)
			if err != nil {
				return err
			}
			item, err := txn.Get(key)
			if err != nil {
				return err
			}
			actual, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			if !bytes.Equal(value, actual) {
				return errLegacyAdoption
			}
		}
		item, err := txn.Get([]byte(legacyAdoptionMarkerKey))
		if err != nil {
			return err
		}
		actual, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		if !bytes.Equal(marker, actual) {
			return errLegacyAdoption
		}
		if !apply {
			return errLegacyAdoptionPreview
		}
		return nil
	})
	if !apply && errors.Is(err, errLegacyAdoptionPreview) {
		return report, nil
	}
	if err != nil {
		return legacyAdoptionReport{}, legacyAdoptionFailure(err)
	}
	return report, nil
}

func readActiveLegacyIdentity(st *Store, slug string) (legacyIdentityAnchor, Entity, error) {
	if st == nil || !validEntitySlug(slug) {
		return legacyIdentityAnchor{}, Entity{}, errLegacyAdoption
	}
	var anchor legacyIdentityAnchor
	var entity Entity
	err := st.view(func(txn *badger.Txn) error {
		for _, prefix := range []string{"il-consumed:", "ig:"} {
			if err := requireLegacyAbsent(txn, []byte(prefix+slug)); err != nil {
				return err
			}
		}
		read := func(key []byte) ([]byte, error) {
			item, err := txn.Get(key)
			if err != nil {
				return nil, err
			}
			return item.ValueCopy(nil)
		}
		raw, err := read([]byte(legacyAdoptionMarkerKey))
		if err != nil {
			return err
		}
		marker, err := decodeLegacyAdoptionMarker(raw)
		if err != nil {
			return err
		}
		ak, ek := []byte("il:"+slug), []byte(prefixEntity+slug)
		ar, err := read(ak)
		if err != nil {
			return err
		}
		anchor, err = decodeLegacyIdentityAnchor(ak, ar)
		if err != nil || anchor.Inventory != marker.Inventory {
			return errLegacyAdoption
		}
		er, err := read(ek)
		if err != nil {
			return err
		}
		if err := matchLegacyIdentityAnchor(ak, ar, ek, er); err != nil {
			return err
		}
		entity, err = decodeLegacyEntity(ek, er)
		return err
	})
	if err != nil {
		return legacyIdentityAnchor{}, Entity{}, legacyAdoptionFailure(err)
	}
	return anchor, entity, nil
}
