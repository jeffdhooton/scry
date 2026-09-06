package store

// PRIVATE, UNCALLED exact identity/control observation. No ownership validity,
// support certificate, undo policy, phantom immunity or production integration.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

var errIdentityRelationships = errors.New("memory: invalid identity relationship inventory")

type identityListing struct {
	Slug     string
	Kind     string
	Ordinal  int
	Spelling string
}

type identityRelationshipInventory struct {
	Rows            map[string][]byte
	Entities        map[string]Entity
	Listings        map[string][]identityListing
	NaturalListings map[string][]identityListing
	IndexTargets    map[string][]string
	FamilyCounts    map[string]uint64
	Scanned         uint64
	Digest          string
}

func scanIdentityRelationships(st *Store) (identityRelationshipInventory, error) {
	if st == nil || st.db == nil || (st.txn != nil && generationTransaction(st) != nil) {
		return identityRelationshipInventory{}, errIdentityRelationships
	}
	families := []string{"al:", "ar:", "en:", "ig:", "il:", "il-consumed:", "meta:identity_", "rs:", "rt:"}
	// Prefixes are disjoint and none straddles another prefix's key range.
	// Sorting these exact families therefore gives global selected-key order.
	sort.Strings(families)
	out := identityRelationshipInventory{Rows: map[string][]byte{}, Entities: map[string]Entity{}, Listings: map[string][]identityListing{}, NaturalListings: map[string][]identityListing{}, IndexTargets: map[string][]string{}, FamilyCounts: map[string]uint64{}}
	digest := sha256.New()
	_, _ = digest.Write([]byte("identity-relationship-inventory-v1\x00"))
	var length [8]byte
	err := st.view(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for _, family := range families {
			out.FamilyCounts[family] = 0
			prefix := []byte(family)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				key := it.Item().KeyCopy(nil)
				raw, err := it.Item().ValueCopy(nil)
				if err != nil {
					return errIdentityRelationships
				}
				switch family {
				case "en:":
					e, err := decodeLegacyEntity(key, raw)
					if err != nil {
						return errIdentityRelationships
					}
					out.Entities[e.Slug] = e
					add := func(kind string, ordinal int, spelling string) {
						listing := identityListing{Slug: e.Slug, Kind: kind, Ordinal: ordinal, Spelling: spelling}
						norm, natural := Normalize(spelling), Slugify(spelling)
						out.Listings[norm] = append(out.Listings[norm], listing)
						out.NaturalListings[natural] = append(out.NaturalListings[natural], listing)
					}
					add("name", -1, e.Name)
					for i, alias := range e.Aliases {
						add("alias", i, alias)
					}
				case "al:":
					owner := string(raw)
					out.IndexTargets[owner] = append(out.IndexTargets[owner], string(key))
				}
				out.Rows[string(key)] = raw
				out.FamilyCounts[family]++
				out.Scanned++
				for _, part := range [][]byte{key, raw} {
					binary.BigEndian.PutUint64(length[:], uint64(len(part)))
					_, _ = digest.Write(length[:])
					_, _ = digest.Write(part)
				}
			}
		}
		return nil
	})
	if err != nil {
		return identityRelationshipInventory{}, errIdentityRelationships
	}
	// Keep the captured raw alias keys explicit. No normalization or filtering of
	// malformed keys is permitted merely to make this measurement look cleaner.
	for owner := range out.IndexTargets {
		sort.Slice(out.IndexTargets[owner], func(i, j int) bool {
			return strings.Compare(out.IndexTargets[owner][i], out.IndexTargets[owner][j]) < 0
		})
	}
	out.Digest = hex.EncodeToString(digest.Sum(nil))
	return out, nil
}
