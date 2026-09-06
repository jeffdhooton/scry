package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func legacyAnchorFixture(t *testing.T) (Entity, []byte, []byte) {
	t.Helper()
	e := Entity{Slug: "retained-slug", Name: "Chosen Canonical Name", Type: "project", Description: "Exact legacy metadata 世界", Aliases: []string{"legacy alias"}, RepoRefs: []string{"/synthetic/repo"}, CreatedAt: time.Date(2026, 9, 6, 1, 2, 3, 456, time.UTC), LastSeen: time.Date(2026, 9, 6, 4, 5, 6, 0, time.UTC)}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return e, []byte(prefixEntity + e.Slug), raw
}

func TestLegacyAnchorExactSourceAndMetadataChanges(t *testing.T) {
	e, key, raw := legacyAnchorFixture(t)
	keyBefore, rawBefore := bytes.Clone(key), bytes.Clone(raw)
	ak, ar, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), key, raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := decodeLegacyIdentityAnchor(ak, ar)
	if err != nil || a.Name != e.Name || a.Slug != e.Slug || !a.CreatedAt.Equal(e.CreatedAt) || a.RawHash != legacyRawFingerprint(key, raw) {
		t.Fatal("anchor lost exact identity")
	}
	if !bytes.Equal(key, keyBefore) || !bytes.Equal(raw, rawBefore) {
		t.Fatal("source bytes mutated")
	}
	if err := matchLegacyIdentityAnchor(ak, ar, key, raw); err != nil {
		t.Fatal("exact identity rejected")
	}
	e.Type = "tool"
	e.Description = "Updated"
	e.Aliases = []string{"changed alias"}
	e.RepoRefs = nil
	e.LastSeen = e.LastSeen.Add(time.Hour)
	updated, _ := json.Marshal(e)
	if err := matchLegacyIdentityAnchor(ak, ar, key, updated); err != nil {
		t.Fatal("metadata-only binding changed")
	}
	for _, field := range []string{"name", "slug", "creation"} {
		other := e
		otherKey := key
		switch field {
		case "name":
			other.Name = "Different Canonical Name"
		case "slug":
			other.Slug = "different-slug"
			otherKey = []byte(prefixEntity + other.Slug)
		case "creation":
			other.CreatedAt = other.CreatedAt.Add(time.Nanosecond)
		}
		otherRaw, _ := json.Marshal(other)
		if err := matchLegacyIdentityAnchor(ak, ar, otherKey, otherRaw); err == nil {
			t.Fatal("identity change accepted")
		}
	}
}

func TestLegacyAnchorSourceRefusals(t *testing.T) {
	_, key, raw := legacyAnchorFixture(t)
	for _, bad := range [][]byte{nil, {}, []byte("null"), []byte("{}"), append(bytes.Clone(raw), ' '), append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":true}`)...), append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"name":"Different"}`)...), bytes.Replace(raw, []byte("Chosen Canonical Name"), []byte(`\ud800`), 1), bytes.Replace(raw, []byte("Chosen Canonical Name"), []byte("bad-\xff"), 1)} {
		before := bytes.Clone(bad)
		if _, _, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), key, bad); !errors.Is(err, errLegacyIdentityAnchor) || err.Error() != errLegacyIdentityAnchor.Error() {
			t.Fatal("invalid source accepted or nonstatic error")
		}
		if !bytes.Equal(before, bad) {
			t.Fatal("rejected source mutated")
		}
	}
	if _, _, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), []byte("en:wrong-key"), raw); err == nil {
		t.Fatal("wrong source key accepted")
	}
	for _, badID := range []string{"", strings.Repeat("A", 64), strings.Repeat("a", 63), strings.Repeat("g", 64), strings.Repeat("a", 63) + "\xff"} {
		if _, _, err := makeLegacyIdentityAnchor(badID, key, raw); err == nil {
			t.Fatal("invalid inventory ID accepted")
		}
	}
}

func TestLegacyAnchorCanonicalAndTimeBoundaries(t *testing.T) {
	e, key, raw := legacyAnchorFixture(t)
	ak, ar, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), key, raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := decodeLegacyIdentityAnchor(ak, ar)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Time{a.CreatedAt.In(time.FixedZone("minute", 3600)), a.CreatedAt.In(time.FixedZone("second", 1)), time.Unix(0, math.MinInt64).UTC(), time.Unix(0, math.MaxInt64).UTC()} {
		other := a
		other.CreatedAt = at
		k, v, err := encodeLegacyIdentityAnchor(other)
		if err != nil {
			t.Fatal("valid instant refused")
		}
		decoded, err := decodeLegacyIdentityAnchor(k, v)
		if err != nil || !decoded.CreatedAt.Equal(at) {
			t.Fatal("exact instant lost")
		}
		if at.Equal(a.CreatedAt) && !bytes.Equal(v, ar) {
			t.Fatal("equal instant changed anchor")
		}
	}
	for _, at := range []time.Time{time.Time{}, time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC), time.Unix(0, math.MinInt64).Add(-time.Nanosecond), time.Unix(0, math.MaxInt64).Add(time.Nanosecond)} {
		other := a
		other.CreatedAt = at
		if _, _, err := encodeLegacyIdentityAnchor(other); err == nil {
			t.Fatal("unsupported timestamp accepted")
		}
	}
	for _, field := range []string{"version", "inventory", "hash", "name", "slug"} {
		other := a
		switch field {
		case "version":
			other.Version = 2
		case "inventory":
			other.Inventory = "wrong"
		case "hash":
			other.RawHash = "wrong"
		case "name":
			other.Name = "bad-\xff"
		case "slug":
			other.Slug = "bad:slug"
		}
		if _, _, err := encodeLegacyIdentityAnchor(other); err == nil {
			t.Fatal("malformed anchor accepted")
		}
	}
	for _, bad := range [][]byte{append(bytes.Clone(ar), ' '), append(bytes.Clone(ar[:len(ar)-1]), []byte(`,"unknown":true}`)...), append(bytes.Clone(ar[:len(ar)-1]), []byte(`,"version":1}`)...)} {
		if _, err := decodeLegacyIdentityAnchor(ak, bad); err == nil {
			t.Fatal("noncanonical anchor accepted")
		}
	}
	if _, err := decodeLegacyIdentityAnchor([]byte("il:wrong"), ar); err == nil {
		t.Fatal("wrong anchor key accepted")
	}
	e.Name = ""
	empty, _ := json.Marshal(e)
	if _, _, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), key, empty); err == nil {
		t.Fatal("empty name accepted")
	}
}

func TestLegacyAnchorFingerprintFramesBothInputs(t *testing.T) {
	key, raw := []byte("ab"), []byte("c")
	var frame bytes.Buffer
	for _, p := range [][]byte{key, raw} {
		if err := binary.Write(&frame, binary.BigEndian, uint64(len(p))); err != nil {
			t.Fatal(err)
		}
		frame.Write(p)
	}
	sum := sha256.Sum256(frame.Bytes())
	if legacyRawFingerprint(key, raw) != hex.EncodeToString(sum[:]) {
		t.Fatal("fingerprint framing differs")
	}
	if legacyRawFingerprint(key, raw) == legacyRawFingerprint([]byte("a"), []byte("bc")) {
		t.Fatal("unframed concatenation collision")
	}
}

func TestLegacyConsumptionCanonicalOwnershipAndRefusals(t *testing.T) {
	_, key, raw := legacyAnchorFixture(t)
	ak, ar, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), key, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"merge", "retire"} {
		c := legacyIdentityConsumption{Version: 1, AnchorKey: bytes.Clone(ak), Anchor: bytes.Clone(ar), Operation: strings.Repeat("b", 64), Reason: reason}
		if reason == "merge" {
			c.Successor = "survivor"
		}
		ck, cr, err := encodeLegacyIdentityConsumption(c)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeLegacyIdentityConsumption(ck, cr)
		if err != nil || !reflect.DeepEqual(decoded, c) {
			t.Fatal("consumption lost original anchor")
		}
		c.AnchorKey[0] = 'X'
		c.Anchor[0] = 'X'
		if _, err := decodeLegacyIdentityConsumption(ck, cr); err != nil {
			t.Fatal("caller input mutation changed encoded record")
		}
		decoded.Anchor[0] = 'X'
		if _, err := decodeLegacyIdentityConsumption(ck, cr); err != nil {
			t.Fatal("decoded result aliases input")
		}
		if _, err := decodeLegacyIdentityConsumption([]byte("il-consumed:wrong"), cr); err == nil {
			t.Fatal("wrong consumption key accepted")
		}
		if _, err := decodeLegacyIdentityConsumption(ck, append(bytes.Clone(cr[:len(cr)-1]), []byte(`,"version":1}`)...)); err == nil {
			t.Fatal("duplicate consumption accepted")
		}
	}
	for _, mode := range []string{"nil-key", "nil-anchor", "wrong-op", "wrong-version", "wrong-reason", "same-target", "missing-target", "retire-target", "bad-target"} {
		c := legacyIdentityConsumption{Version: 1, AnchorKey: ak, Anchor: ar, Operation: strings.Repeat("b", 64), Reason: "merge", Successor: "survivor"}
		switch mode {
		case "nil-key":
			c.AnchorKey = nil
		case "nil-anchor":
			c.Anchor = nil
		case "wrong-op":
			c.Operation = "not-digest"
		case "wrong-version":
			c.Version = 2
		case "wrong-reason":
			c.Reason = "automatic"
		case "same-target":
			c.Successor = "retained-slug"
		case "missing-target":
			c.Successor = ""
		case "retire-target":
			c.Reason = "retire"
		case "bad-target":
			c.Successor = "Invalid Target"
		}
		if _, _, err := encodeLegacyIdentityConsumption(c); err == nil {
			t.Fatal("invalid consumption accepted")
		}
	}
}
