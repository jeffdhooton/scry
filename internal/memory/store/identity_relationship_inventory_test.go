package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func relationshipEntity(t *testing.T, st *Store, slug, name string, aliases []string) Entity {
	t.Helper()
	e := Entity{Slug: slug, Name: name, Type: "project", Aliases: aliases, CreatedAt: time.Date(2026, 9, 6, 0, 0, 0, 1, time.UTC)}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	gradeGenerationSet(t, st, "en:"+slug, raw)
	return e
}

func relationshipDigest(rows map[string][]byte) string {
	keys := []string{}
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteString("identity-relationship-inventory-v1")
	buf.WriteByte(0)
	for _, k := range keys {
		binary.Write(&buf, binary.BigEndian, uint64(len(k)))
		buf.WriteString(k)
		binary.Write(&buf, binary.BigEndian, uint64(len(rows[k])))
		buf.Write(rows[k])
	}
	h := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(h[:])
}

func TestRelationshipInventoryExactProjectionAndFamilies(t *testing.T) {
	st := openTemp(t)
	a := relationshipEntity(t, st, "atlas", "Shared", []string{"shared", "Shared", "A/B", "", "世界"})
	b := relationshipEntity(t, st, "borealis", "Separate", []string{"shared"})
	selected := map[string][]byte{"al:shared": []byte("atlas"), "al:unlisted": []byte("atlas"), "al:empty": {}, "al:NOT normalized": []byte("absent"), "al:\xff": {255, 0}, "ar:opaque": {255, 0}, "ig:atlas": {}, "il:atlas": []byte("opaque"), "il-consumed:atlas": {}, "meta:identity_unknown": []byte("opaque"), "rs:atlas": {}, "rt:shared": {}}
	for k, v := range selected {
		gradeGenerationSet(t, st, k, v)
	}
	for _, k := range []string{"fa:excluded", "ep:excluded", "att:excluded", "iga:excluded", "io:excluded", "io-input:excluded", "io-result:excluded", "meta:other", "alike:excluded", "enough:excluded"} {
		gradeGenerationSet(t, st, k, []byte{255, 1})
	}
	before := gradeGenerationSnapshot(t, st)
	selected["en:atlas"] = before["en:atlas"]
	selected["en:borealis"] = before["en:borealis"]
	events := 0
	st.SetObserver(func(Event) { events++ })
	got, err := scanIdentityRelationships(st)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Rows, selected) || got.Scanned != uint64(len(selected)) || got.Digest != relationshipDigest(selected) {
		t.Fatal("inventory bytes/framing mismatch")
	}
	if !reflect.DeepEqual(got.Entities, map[string]Entity{"atlas": a, "borealis": b}) {
		t.Fatal("entity fidelity")
	}
	want := []identityListing{{"atlas", "name", -1, "Shared"}, {"atlas", "alias", 0, "shared"}, {"atlas", "alias", 1, "Shared"}, {"borealis", "alias", 0, "shared"}}
	if !reflect.DeepEqual(got.Listings["shared"], want) || !reflect.DeepEqual(got.NaturalListings["shared"], want) {
		t.Fatal("listing occurrence loss")
	}
	if !reflect.DeepEqual(got.NaturalListings["ab"], []identityListing{{"atlas", "alias", 2, "A/B"}}) || len(got.Listings["a/b"]) != 1 {
		t.Fatal("natural/name routing collapsed")
	}
	if len(got.Listings[""]) != 1 || len(got.NaturalListings[""]) != 2 {
		t.Fatal("empty observation lost")
	}
	if !reflect.DeepEqual(got.IndexTargets["atlas"], []string{"al:shared", "al:unlisted"}) || !reflect.DeepEqual(got.IndexTargets[""], []string{"al:empty"}) || !reflect.DeepEqual(got.IndexTargets[string([]byte{255, 0})], []string{"al:\xff"}) {
		t.Fatal("raw owner/reverse-key loss")
	}
	if _, ok := got.Listings["atlas"]; ok {
		t.Fatal("invented slug listing")
	}
	if len(got.FamilyCounts) != 9 || got.FamilyCounts["al:"] != 5 || got.FamilyCounts["en:"] != 2 {
		t.Fatal("family coverage")
	}
	if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("scan changed graph")
	}
}

func TestRelationshipInventoryOwnedAndReadYourWrites(t *testing.T) {
	st := openTemp(t)
	relationshipEntity(t, st, "atlas", "Atlas", []string{"shared"})
	gradeGenerationSet(t, st, "al:shared", []byte("atlas"))
	before, err := scanIdentityRelationships(st)
	if err != nil {
		t.Fatal(err)
	}
	first, err := scanIdentityRelationships(st)
	if err != nil {
		t.Fatal(err)
	}
	first.Rows["en:atlas"][0] = 0
	first.Rows["al:shared"][0] = 0
	e := first.Entities["atlas"]
	e.Aliases[0] = "mutated"
	first.Listings["shared"][0].Spelling = "mutated"
	first.NaturalListings["shared"][0].Slug = "mutated"
	first.IndexTargets["atlas"][0] = "mutated"
	first.FamilyCounts["en:"] = 100
	fresh, err := scanIdentityRelationships(st)
	if err != nil || !reflect.DeepEqual(before, fresh) {
		t.Fatal("caller corrupted store or previous capture")
	}
	var escaped *Store
	err = st.AtomicWrite(func(tx *Store) error {
		escaped = tx
		if err := tx.txn.Set([]byte("al:later"), []byte("absent")); err != nil {
			return err
		}
		got, err := scanIdentityRelationships(tx)
		if err != nil {
			return err
		}
		if !bytes.Equal(got.Rows["al:later"], []byte("absent")) || got.Scanned != before.Scanned+1 {
			t.Fatal("staged row not visible")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scanIdentityRelationships(escaped); err != errIdentityRelationships {
		t.Fatal("closed facade accepted")
	}
	for _, bad := range []*Store{nil, {}} {
		if v, err := scanIdentityRelationships(bad); err != errIdentityRelationships || !reflect.DeepEqual(v, identityRelationshipInventory{}) {
			t.Fatal("empty facade accepted")
		}
	}
}

func TestRelationshipInventoryDanglingAndLifecycleChangesVisible(t *testing.T) {
	st := openTemp(t)
	relationshipEntity(t, st, "atlas", "Atlas", nil)
	relationshipEntity(t, st, "borealis", "Borealis", []string{"shared"})
	gradeGenerationSet(t, st, "al:shared", []byte("atlas"))
	before, err := scanIdentityRelationships(st)
	if err != nil {
		t.Fatal(err)
	}
	err = st.AtomicWrite(func(tx *Store) error { return tx.txn.Delete([]byte("en:atlas")) })
	if err != nil {
		t.Fatal(err)
	}
	after, err := scanIdentityRelationships(st)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Listings["shared"], after.Listings["shared"]) || !bytes.Equal(before.Rows["al:shared"], after.Rows["al:shared"]) || before.Digest == after.Digest {
		t.Fatal("fixture did not preserve bytes while changing relationship")
	}
	if _, ok := after.Entities["atlas"]; ok || !reflect.DeepEqual(after.IndexTargets["atlas"], []string{"al:shared"}) {
		t.Fatal("absent unlisted index target lost")
	}
	for _, key := range []string{"ig:borealis", "il:borealis", "il-consumed:borealis", "meta:identity_adoption_v1", "ar:borealis:shared", "rs:borealis", "rt:shared"} {
		previous := after.Digest
		gradeGenerationSet(t, st, key, []byte{})
		after, err = scanIdentityRelationships(st)
		if err != nil || after.Digest == previous {
			t.Fatal("present-empty control change hidden")
		}
		if value, exists := after.Rows[key]; !exists || len(value) != 0 {
			t.Fatal("presence lost")
		}
	}
}

func TestRelationshipInventoryMalformedEntityGlobalRefusal(t *testing.T) {
	for _, kind := range []string{"unknown", "duplicate", "space", "key", "invalid", "empty", "zero-time"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			relationshipEntity(t, st, "atlas", "Atlas", nil)
			e := relationshipEntity(t, st, "zed", "Zed", nil)
			gradeGenerationSet(t, st, "al:opaque", []byte{255, 0})
			raw := gradeGenerationSnapshot(t, st)["en:zed"]
			key := "en:zed"
			switch kind {
			case "unknown":
				raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"extra":1}`)...)
			case "duplicate":
				raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"slug":"zed"}`)...)
			case "space":
				raw = append([]byte(" "), raw...)
			case "key":
				key = "en:wrong"
			case "invalid":
				raw = []byte{255}
			case "empty":
				// gradeGenerationSet treats nil as DELETE, not a present empty
				// value. Exercise the malformed existing-row case explicitly.
				raw = []byte{}
			case "zero-time":
				e.CreatedAt = time.Time{}
				raw, _ = json.Marshal(e)
			}
			gradeGenerationSet(t, st, key, raw)
			before := gradeGenerationSnapshot(t, st)
			got, err := scanIdentityRelationships(st)
			if err != errIdentityRelationships || !reflect.DeepEqual(got, identityRelationshipInventory{}) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("partial/unsafe malformed inventory")
			}
		})
	}
}

func TestRelationshipInventoryEmptyDigestAndClosedRoot(t *testing.T) {
	st := openTemp(t)
	got, err := scanIdentityRelationships(st)
	if err != nil || got.Scanned != 0 || got.Digest != relationshipDigest(map[string][]byte{}) || len(got.FamilyCounts) != 9 {
		t.Fatal("empty inventory")
	}
	for family, count := range got.FamilyCounts {
		if count != 0 || (!strings.HasSuffix(family, ":") && family != "meta:identity_") {
			t.Fatal("bad empty family")
		}
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := scanIdentityRelationships(st); err != errIdentityRelationships || !reflect.DeepEqual(got, identityRelationshipInventory{}) {
		t.Fatal("closed root accepted")
	}
}
