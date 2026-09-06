package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func independentAnchorFixture(t *testing.T) (Entity, []byte, []byte, []byte, []byte) {
	t.Helper()
	e := Entity{Slug: "legacy-kept", Name: "Renamed 世界 Identity", Type: "project", Description: "description", Aliases: []string{"alias"}, RepoRefs: []string{"repo"}, CreatedAt: time.Date(2025, 2, 3, 4, 5, 6, 789123456, time.FixedZone("offset", -19800)), LastSeen: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	key := []byte("en:" + e.Slug)
	ak, ar, err := makeLegacyIdentityAnchor(strings.Repeat("c", 64), key, raw)
	if err != nil {
		t.Fatal(err)
	}
	return e, key, raw, ak, ar
}

func independentJSONField(t *testing.T, raw []byte, field string, replace []byte, omit bool) []byte {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	original, ok := fields[field]
	if !ok {
		t.Fatalf("missing field %s", field)
	}
	marker := append([]byte(`"`+field+`":`), original...)
	if omit {
		if bytes.Contains(raw, append(bytes.Clone(marker), ',')) {
			return bytes.Replace(raw, append(bytes.Clone(marker), ','), nil, 1)
		}
		return bytes.Replace(raw, append([]byte{','}, marker...), nil, 1)
	}
	return bytes.Replace(raw, marker, append([]byte(`"`+field+`":`), replace...), 1)
}

func independentRefusal(t *testing.T, key, raw []byte, check func([]byte, []byte) error) {
	t.Helper()
	kb, rb := bytes.Clone(key), bytes.Clone(raw)
	err := check(key, raw)
	if err != errLegacyIdentityAnchor {
		t.Fatalf("wanted static sentinel, got %v", err)
	}
	if !bytes.Equal(key, kb) || !bytes.Equal(raw, rb) {
		t.Fatal("refused input mutated")
	}
}

func TestIndependentAnchorEveryRawFieldRefusesLoss(t *testing.T) {
	_, key, raw, ak, ar := independentAnchorFixture(t)
	makeCheck := func(k, v []byte) error {
		_, _, err := makeLegacyIdentityAnchor(strings.Repeat("c", 64), k, v)
		return err
	}
	anchorCheck := func(k, v []byte) error { _, err := decodeLegacyIdentityAnchor(k, v); return err }
	for _, subject := range []struct {
		name     string
		key, raw []byte
		fields   []string
		check    func([]byte, []byte) error
	}{
		{"entity", key, raw, []string{"slug", "name", "type", "description", "aliases", "repo_refs", "created_at", "last_seen"}, makeCheck},
		{"anchor", ak, ar, []string{"version", "inventory", "slug", "name", "created_at", "raw_hash"}, anchorCheck},
	} {
		for _, field := range subject.fields {
			for _, kind := range []string{"null", "duplicate", "case", "invalid-unicode", "surrogate"} {
				t.Run(subject.name+"/"+field+"/"+kind, func(t *testing.T) {
					var bad []byte
					switch kind {
					case "null":
						bad = independentJSONField(t, subject.raw, field, []byte("null"), false)
					case "duplicate":
						var fields map[string]json.RawMessage
						json.Unmarshal(subject.raw, &fields)
						bad = append(bytes.Clone(subject.raw[:len(subject.raw)-1]), []byte(`,"`+field+`":`+string(fields[field])+`}`)...)
					case "case":
						bad = bytes.Replace(subject.raw, []byte(`"`+field+`":`), []byte(`"`+strings.ToUpper(field)+`":`), 1)
					case "invalid-unicode":
						bad = independentJSONField(t, subject.raw, field, []byte{'"', 0xff, '"'}, false)
					case "surrogate":
						bad = independentJSONField(t, subject.raw, field, []byte(`"\ud800"`), false)
					}
					independentRefusal(t, subject.key, bad, subject.check)
				})
			}
			// Entity's omitempty metadata is intentionally optional; required fields are not.
			if field != "aliases" && field != "repo_refs" {
				t.Run(subject.name+"/"+field+"/missing", func(t *testing.T) {
					independentRefusal(t, subject.key, independentJSONField(t, subject.raw, field, nil, true), subject.check)
				})
			}
		}
		for _, bad := range [][]byte{nil, {}, []byte("null"), []byte("{}"), []byte("[]"), append(bytes.Clone(subject.raw), ' '), append([]byte(" "), subject.raw...), append(bytes.Clone(subject.raw), subject.raw...), append(bytes.Clone(subject.raw[:len(subject.raw)-1]), []byte(`,"future":null}`)...)} {
			independentRefusal(t, subject.key, bad, subject.check)
		}
		for _, badKey := range [][]byte{nil, {}, []byte("wrong:"), append(bytes.Clone(subject.key), 0), append(bytes.Clone(subject.key), 0xff)} {
			independentRefusal(t, badKey, subject.raw, subject.check)
		}
	}
	for _, field := range []string{"aliases", "repo_refs"} {
		for _, bad := range []string{`[null]`, `["\udfff"]`, `[]`, `["a",null]`} {
			independentRefusal(t, key, independentJSONField(t, raw, field, []byte(bad), false), makeCheck)
		}
	}
}

func TestIndependentAnchorTimeAndMetadataSemantics(t *testing.T) {
	e, key, raw, ak, ar := independentAnchorFixture(t)
	a, err := decodeLegacyIdentityAnchor(ak, ar)
	if err != nil {
		t.Fatal(err)
	}
	if Slugify(e.Name) == e.Slug {
		t.Fatal("fixture must diverge")
	}
	if a.CreatedAt.Location() != time.UTC || !a.CreatedAt.Equal(e.CreatedAt) {
		t.Fatal("not exact UTC instant")
	}
	for _, mutate := range []func(*Entity){func(v *Entity) { v.Type = "tool" }, func(v *Entity) { v.Description = "新 metadata" }, func(v *Entity) { v.Aliases = nil }, func(v *Entity) { v.RepoRefs = []string{} }, func(v *Entity) { v.LastSeen = time.Time{} }, func(v *Entity) { v.CreatedAt = v.CreatedAt.UTC() }, func(v *Entity) { v.CreatedAt = v.CreatedAt.In(time.FixedZone("other", 3600)) }} {
		v := e
		mutate(&v)
		updated, _ := json.Marshal(v)
		if err := matchLegacyIdentityAnchor(ak, ar, key, updated); err != nil {
			t.Fatal("metadata or equal instant refused", err)
		}
	}
	for _, mutate := range []func(*Entity){func(v *Entity) { v.Name += " " }, func(v *Entity) { v.Slug = "other" }, func(v *Entity) { v.CreatedAt = v.CreatedAt.Add(time.Nanosecond) }} {
		v := e
		mutate(&v)
		updated, _ := json.Marshal(v)
		independentRefusal(t, []byte("en:"+v.Slug), updated, func(k, r []byte) error { return matchLegacyIdentityAnchor(ak, ar, k, r) })
	}
	for _, at := range []time.Time{time.Unix(0, math.MinInt64).UTC(), time.Unix(0, math.MaxInt64).UTC(), time.Unix(0, 0).UTC()} {
		v := e
		v.CreatedAt = at
		updated, _ := json.Marshal(v)
		k, r, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), key, updated)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeLegacyIdentityAnchor(k, r)
		if err != nil || !decoded.CreatedAt.Equal(at) {
			t.Fatal("boundary instant lost")
		}
	}
	for _, at := range []time.Time{time.Time{}, time.Unix(0, math.MinInt64).Add(-time.Nanosecond).UTC(), time.Unix(0, math.MaxInt64).Add(time.Nanosecond).UTC()} {
		v := e
		v.CreatedAt = at
		updated, _ := json.Marshal(v)
		independentRefusal(t, key, updated, func(k, r []byte) error {
			_, _, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), k, r)
			return err
		})
	}
	for _, s := range []string{`"2025-02-03T04:05:06.7891234560-05:30"`, `"2025-02-03T04:05:06.7891234561-05:30"`, `"2025-02-03T04:05:06,789123456-05:30"`, `"2025-02-03T04:05:60Z"`, `"2025-02-03T04:05:06+24:00"`} {
		bad := independentJSONField(t, raw, "created_at", []byte(s), false)
		independentRefusal(t, key, bad, func(k, r []byte) error {
			_, _, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), k, r)
			return err
		})
	}
	// An offset is legitimate in the source; canonical anchors must use UTC.
	offset, _ := json.Marshal(e.CreatedAt)
	independentRefusal(t, ak, independentJSONField(t, ar, "created_at", offset, false), func(k, r []byte) error { _, err := decodeLegacyIdentityAnchor(k, r); return err })
}

func TestIndependentAnchorDigestAndOwnership(t *testing.T) {
	_, key, raw, ak, ar := independentAnchorFixture(t)
	framed := make([]byte, 8+len(key)+8+len(raw))
	binary.BigEndian.PutUint64(framed, uint64(len(key)))
	copy(framed[8:], key)
	binary.BigEndian.PutUint64(framed[8+len(key):], uint64(len(raw)))
	copy(framed[16+len(key):], raw)
	sum := sha256.Sum256(framed)
	a, err := decodeLegacyIdentityAnchor(ak, ar)
	if err != nil || a.RawHash != hex.EncodeToString(sum[:]) {
		t.Fatal("digest not exact framed source")
	}
	for _, pair := range [][2][]byte{{nil, nil}, {[]byte("a"), []byte("bc")}, {[]byte("ab"), []byte("c")}, {[]byte("abc"), nil}, {nil, []byte("abc")}} {
		for _, other := range [][2][]byte{{[]byte("x"), pair[1]}, {pair[0], []byte("x")}} {
			if legacyRawFingerprint(pair[0], pair[1]) == legacyRawFingerprint(other[0], other[1]) {
				t.Fatal("input omitted from digest")
			}
		}
	}
	before := a
	for i := range raw {
		raw[i] = 0
	}
	for i := range key {
		key[i] = 0
	}
	if _, err := decodeLegacyIdentityAnchor(ak, ar); err != nil {
		t.Fatal("source aliases output")
	}
	for i := range ar {
		ar[i] = 0
	}
	for i := range ak {
		ak[i] = 0
	}
	if !reflect.DeepEqual(a, before) {
		t.Fatal("decoded anchor aliases caller bytes")
	}
	for _, bad := range []string{"", strings.Repeat("0", 63), strings.Repeat("0", 65), strings.Repeat("A", 64), strings.Repeat("z", 64), strings.Repeat("0", 63) + "\xff"} {
		for _, field := range []string{"inventory", "hash"} {
			b := a
			if field == "inventory" {
				b.Inventory = bad
			} else {
				b.RawHash = bad
			}
			if _, _, err := encodeLegacyIdentityAnchor(b); err != errLegacyIdentityAnchor {
				t.Fatal("invalid digest accepted")
			}
		}
	}
	for _, name := range []string{"", "x\xff"} {
		b := a
		b.Name = name
		if _, _, err := encodeLegacyIdentityAnchor(b); err != errLegacyIdentityAnchor {
			t.Fatal("bad name accepted")
		}
	}
	for _, slug := range []string{"", "Upper", "x:y", "x y", "x\xff", "../x"} {
		b := a
		b.Slug = slug
		if _, _, err := encodeLegacyIdentityAnchor(b); err != errLegacyIdentityAnchor {
			t.Fatal("bad slug accepted")
		}
	}
}

func TestIndependentConsumptionStrictControlsAndOwnership(t *testing.T) {
	_, _, _, ak, ar := independentAnchorFixture(t)
	c := legacyIdentityConsumption{Version: 1, AnchorKey: ak, Anchor: ar, Operation: strings.Repeat("e", 64), Reason: "merge", Successor: "recipient"}
	ck, cr, err := encodeLegacyIdentityConsumption(c)
	if err != nil {
		t.Fatal(err)
	}
	check := func(k, r []byte) error { _, err := decodeLegacyIdentityConsumption(k, r); return err }
	for _, field := range []string{"version", "anchor_key", "anchor", "operation", "reason", "successor"} {
		for _, bad := range [][]byte{[]byte("null"), []byte(`"\ud800"`), []byte{'"', 0xff, '"'}} {
			independentRefusal(t, ck, independentJSONField(t, cr, field, bad, false), check)
		}
		independentRefusal(t, ck, independentJSONField(t, cr, field, nil, true), check)
		var fields map[string]json.RawMessage
		json.Unmarshal(cr, &fields)
		duplicate := append(bytes.Clone(cr[:len(cr)-1]), []byte(`,"`+field+`":`+string(fields[field])+`}`)...)
		independentRefusal(t, ck, duplicate, check)
	}
	for _, bad := range [][]byte{nil, {}, []byte("null"), []byte("{}"), append(bytes.Clone(cr), ' '), append(bytes.Clone(cr[:len(cr)-1]), []byte(`,"unknown":null}`)...)} {
		independentRefusal(t, ck, bad, check)
	}
	for _, key := range [][]byte{nil, {}, []byte("il-consumed:recipient"), ak, append(bytes.Clone(ck), 0xff)} {
		independentRefusal(t, key, cr, check)
	}
	for _, mutation := range []func(*legacyIdentityConsumption){
		func(v *legacyIdentityConsumption) { v.Version = 0 }, func(v *legacyIdentityConsumption) { v.Version = 2 }, func(v *legacyIdentityConsumption) { v.Operation = "" }, func(v *legacyIdentityConsumption) { v.Operation = strings.Repeat("E", 64) }, func(v *legacyIdentityConsumption) { v.Operation = strings.Repeat("e", 63) + "\xff" }, func(v *legacyIdentityConsumption) { v.Reason = "MERGE" }, func(v *legacyIdentityConsumption) { v.Reason = "" }, func(v *legacyIdentityConsumption) { v.Reason = "retire" }, func(v *legacyIdentityConsumption) { v.Successor = "" }, func(v *legacyIdentityConsumption) { v.Successor = "legacy-kept" }, func(v *legacyIdentityConsumption) { v.Successor = "x\xff" }, func(v *legacyIdentityConsumption) { v.Successor = "x:y" }, func(v *legacyIdentityConsumption) { v.AnchorKey = []byte("il:other") }, func(v *legacyIdentityConsumption) { v.Anchor = append(bytes.Clone(v.Anchor), ' ') }, func(v *legacyIdentityConsumption) { v.Anchor = nil },
	} {
		v := c
		mutation(&v)
		if _, _, err := encodeLegacyIdentityConsumption(v); err != errLegacyIdentityAnchor {
			t.Fatal("invalid consumption control accepted")
		}
	}
	retired := c
	retired.Reason = "retire"
	retired.Successor = ""
	rk, rr, err := encodeLegacyIdentityConsumption(retired)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rk, ck) || bytes.Equal(rr, cr) {
		t.Fatal("operation not captured at stable consumed address")
	}
	decoded, err := decodeLegacyIdentityConsumption(ck, cr)
	if err != nil || !reflect.DeepEqual(decoded, c) {
		t.Fatal("full anchor or controls lost")
	}
	beforeKey, beforeAnchor := bytes.Clone(decoded.AnchorKey), bytes.Clone(decoded.Anchor)
	for i := range ak {
		ak[i] = 0
	}
	for i := range ar {
		ar[i] = 0
	}
	if _, err := decodeLegacyIdentityConsumption(ck, cr); err != nil {
		t.Fatal("encode retained supplied anchor")
	}
	for i := range cr {
		cr[i] = 0
	}
	for i := range ck {
		ck[i] = 0
	}
	if !bytes.Equal(decoded.AnchorKey, beforeKey) || !bytes.Equal(decoded.Anchor, beforeAnchor) {
		t.Fatal("decode retained raw input")
	}
	decoded.Anchor[0] = 'X'
	if bytes.Equal(decoded.Anchor, beforeAnchor) {
		t.Fatal("independent snapshot changed with decoded buffer")
	}
}

func TestIndependentAnchorUnicodeRoundTrips(t *testing.T) {
	e, key, _, _, _ := independentAnchorFixture(t)
	for _, name := range []string{"世界", "😀", "e\u0301", "é", "\ufffd", "\x00", "<>&\u2028\u2029"} {
		v := e
		v.Name, v.Description = name, name
		v.Aliases, v.RepoRefs = []string{name}, []string{name}
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		ak, ar, err := makeLegacyIdentityAnchor(strings.Repeat("d", 64), key, raw)
		if err != nil {
			t.Fatalf("valid Unicode refused: %q: %v", name, err)
		}
		a, err := decodeLegacyIdentityAnchor(ak, ar)
		if err != nil || a.Name != name {
			t.Fatal("valid Unicode lost")
		}
	}
	_, _, raw, _, _ := independentAnchorFixture(t)
	for _, field := range []string{"name", "type", "description", "aliases", "repo_refs"} {
		for b := 128; b <= 255; b++ {
			replacement := []byte{'"', byte(b), '"'}
			if field == "aliases" || field == "repo_refs" {
				replacement = append(append([]byte{'['}, replacement...), ']')
			}
			bad := independentJSONField(t, raw, field, replacement, false)
			independentRefusal(t, key, bad, func(k, r []byte) error {
				_, _, err := makeLegacyIdentityAnchor(strings.Repeat("d", 64), k, r)
				return err
			})
		}
	}
}
