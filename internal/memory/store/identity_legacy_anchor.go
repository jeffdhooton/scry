package store

// PRIVATE, UNCALLED pure codec. It neither adopts a store nor authorizes an
// owner, metadata update, merge, retirement or consumed-identity reuse.

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"
)

var errLegacyIdentityAnchor = errors.New("memory: invalid legacy identity anchor")

type legacyIdentityAnchor struct {
	Version   int       `json:"version"`
	Inventory string    `json:"inventory"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	RawHash   string    `json:"raw_hash"`
}

type legacyIdentityConsumption struct {
	Version   int    `json:"version"`
	AnchorKey []byte `json:"anchor_key"`
	Anchor    []byte `json:"anchor"`
	Operation string `json:"operation"`
	Reason    string `json:"reason"`
	Successor string `json:"successor"`
}

func legacyDigestValid(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func legacyRawFingerprint(key, raw []byte) string {
	h := sha256.New()
	var size [8]byte
	for _, part := range [][]byte{key, raw} {
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		_, _ = h.Write(size[:])
		_, _ = h.Write(part)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func decodeLegacyEntity(key, raw []byte) (Entity, error) {
	if !utf8.Valid(key) || !utf8.Valid(raw) {
		return Entity{}, errLegacyIdentityAnchor
	}
	var e Entity
	if json.Unmarshal(raw, &e) != nil {
		return Entity{}, errLegacyIdentityAnchor
	}
	canonical, err := json.Marshal(e)
	if err != nil || !bytes.Equal(raw, canonical) || !validEntitySlug(e.Slug) || e.Name == "" || !utf8.ValidString(e.Name) || !bytes.Equal(key, []byte(prefixEntity+e.Slug)) || e.CreatedAt.IsZero() || !time.Unix(0, e.CreatedAt.UnixNano()).Equal(e.CreatedAt) {
		return Entity{}, errLegacyIdentityAnchor
	}
	return e, nil
}

func encodeLegacyIdentityAnchor(a legacyIdentityAnchor) ([]byte, []byte, error) {
	if a.Version != 1 || !legacyDigestValid(a.Inventory) || !legacyDigestValid(a.RawHash) || !validEntitySlug(a.Slug) || !utf8.ValidString(a.Slug) || a.Name == "" || !utf8.ValidString(a.Name) || a.CreatedAt.IsZero() || !time.Unix(0, a.CreatedAt.UnixNano()).Equal(a.CreatedAt) {
		return nil, nil, errLegacyIdentityAnchor
	}
	a.CreatedAt = a.CreatedAt.UTC()
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, nil, errLegacyIdentityAnchor
	}
	return []byte("il:" + a.Slug), raw, nil
}

func makeLegacyIdentityAnchor(inventory string, key, raw []byte) ([]byte, []byte, error) {
	e, err := decodeLegacyEntity(key, raw)
	if err != nil {
		return nil, nil, err
	}
	return encodeLegacyIdentityAnchor(legacyIdentityAnchor{Version: 1, Inventory: inventory, Slug: e.Slug, Name: e.Name, CreatedAt: e.CreatedAt, RawHash: legacyRawFingerprint(key, raw)})
}

func decodeLegacyIdentityAnchor(key, raw []byte) (legacyIdentityAnchor, error) {
	var a legacyIdentityAnchor
	if json.Unmarshal(raw, &a) != nil {
		return legacyIdentityAnchor{}, errLegacyIdentityAnchor
	}
	expectedKey, canonical, err := encodeLegacyIdentityAnchor(a)
	if err != nil || !bytes.Equal(key, expectedKey) || !bytes.Equal(raw, canonical) {
		return legacyIdentityAnchor{}, errLegacyIdentityAnchor
	}
	return a, nil
}

func matchLegacyIdentityAnchor(anchorKey, anchorRaw, entityKey, entityRaw []byte) error {
	a, err := decodeLegacyIdentityAnchor(anchorKey, anchorRaw)
	if err != nil {
		return err
	}
	e, err := decodeLegacyEntity(entityKey, entityRaw)
	if err != nil {
		return err
	}
	if a.Slug != e.Slug || a.Name != e.Name || !a.CreatedAt.Equal(e.CreatedAt) {
		return errLegacyIdentityAnchor
	}
	return nil
}

func encodeLegacyIdentityConsumption(c legacyIdentityConsumption) ([]byte, []byte, error) {
	if c.Version != 1 || !legacyDigestValid(c.Operation) {
		return nil, nil, errLegacyIdentityAnchor
	}
	a, err := decodeLegacyIdentityAnchor(c.AnchorKey, c.Anchor)
	if err != nil {
		return nil, nil, err
	}
	switch c.Reason {
	case "merge":
		if !validEntitySlug(c.Successor) || c.Successor == a.Slug {
			return nil, nil, errLegacyIdentityAnchor
		}
	case "retire":
		if c.Successor != "" {
			return nil, nil, errLegacyIdentityAnchor
		}
	default:
		return nil, nil, errLegacyIdentityAnchor
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, nil, errLegacyIdentityAnchor
	}
	return []byte("il-consumed:" + a.Slug), raw, nil
}

func decodeLegacyIdentityConsumption(key, raw []byte) (legacyIdentityConsumption, error) {
	var c legacyIdentityConsumption
	if json.Unmarshal(raw, &c) != nil {
		return legacyIdentityConsumption{}, errLegacyIdentityAnchor
	}
	expectedKey, canonical, err := encodeLegacyIdentityConsumption(c)
	if err != nil || !bytes.Equal(key, expectedKey) || !bytes.Equal(raw, canonical) {
		return legacyIdentityConsumption{}, errLegacyIdentityAnchor
	}
	return c, nil
}
