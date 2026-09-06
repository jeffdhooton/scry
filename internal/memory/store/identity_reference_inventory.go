package store

// PRIVATE, UNCALLED baseline measurement. Counts and digest do not authorize an
// owner or certify that the admission scope produced the final supporting facts.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/dgraph-io/badger/v4"
)

type identityReferenceInventory struct {
	Scanned    uint64
	Digest     string
	References map[string]identityReferenceCount
}

func scanIdentityReferenceInventory(st *Store) (identityReferenceInventory, error) {
	if st == nil {
		return identityReferenceInventory{}, errIdentityFactReferences
	}
	if st.txn != nil {
		if err := generationTransaction(st); err != nil {
			return identityReferenceInventory{}, errIdentityFactReferences
		}
	}
	report := identityReferenceInventory{References: make(map[string]identityReferenceCount)}
	digest := sha256.New()
	_, _ = digest.Write([]byte("identity-reference-inventory-v1\x00"))
	var length [8]byte
	err := st.view(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(prefixFact)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			var fact Fact
			if err := it.Item().Value(func(raw []byte) error {
				var err error
				fact, err = decodeIdentityReference(key, raw)
				if err != nil {
					return err
				}
				for _, part := range [][]byte{key, raw} {
					binary.BigEndian.PutUint64(length[:], uint64(len(part)))
					_, _ = digest.Write(length[:])
					_, _ = digest.Write(part)
				}
				return nil
			}); err != nil {
				return err
			}
			report.Scanned++
			for side, slug := range []string{fact.Src, fact.Dst} {
				if slug == "" || (side == 1 && slug == fact.Src) {
					continue
				}
				count := report.References[slug]
				if fact.InvalidAt == nil {
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
		if errors.Is(err, errIdentityFactReferences) {
			return identityReferenceInventory{}, err
		}
		return identityReferenceInventory{}, errIdentityFactReferences
	}
	report.Digest = hex.EncodeToString(digest.Sum(nil))
	return report, nil
}
