package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Independent fixtures always Set, including nil/empty values. Deletion is
// explicit at the call site; a nil fixture cannot accidentally remove a row.
func riSet(t *testing.T, st *Store, rows map[string][]byte) {
	t.Helper()
	if err := st.update(func(tx *badger.Txn) error {
		for k, v := range rows {
			if err := tx.Set([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func riEntity(t *testing.T, slug, name string, aliases []string) []byte {
	t.Helper()
	raw, err := json.Marshal(Entity{Slug: slug, Name: name, Type: "project", Description: "synthetic", Aliases: aliases, RepoRefs: []string{"/synthetic/repo"}, CreatedAt: time.Date(2026, 9, 6, 1, 2, 3, 4, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func riRaw(t *testing.T, st *Store) map[string][]byte {
	t.Helper()
	rows := map[string][]byte{}
	if err := st.view(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			v, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			rows[string(it.Item().KeyCopy(nil))] = v
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return rows
}

// Whole-map oracle sorts once rather than iterating prefixes, and explicitly
// assembles the eight length bytes rather than calling encoding/binary.
func riDigest(rows map[string][]byte) string {
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	framed := []byte("identity-relationship-inventory-v1\x00")
	for _, k := range keys {
		for _, b := range [][]byte{[]byte(k), rows[k]} {
			n := uint64(len(b))
			for shift := 56; shift >= 0; shift -= 8 {
				framed = append(framed, byte(n>>shift))
			}
			framed = append(framed, b...)
		}
	}
	h := sha256.Sum256(framed)
	return hex.EncodeToString(h[:])
}

func riScan(t *testing.T, st *Store) identityRelationshipInventory {
	t.Helper()
	v, err := scanIdentityRelationships(st)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRIIndependentExactBoundariesAndGlobalFraming(t *testing.T) {
	st := openTemp(t)
	selected := map[string][]byte{"en:a": riEntity(t, "a", "Different", nil)}
	wantCounts := map[string]uint64{"en:": 1}
	for _, p := range []string{"al:", "ar:", "ig:", "il:", "il-consumed:", "meta:identity_", "rs:", "rt:"} {
		selected[p] = []byte{}
		selected[p+"\x00"] = []byte{0, 255, 1}
		selected[p+"\xff"] = bytes.Repeat([]byte{127}, 257)
		wantCounts[p] = 3
	}
	riSet(t, st, selected)
	excluded := map[string][]byte{}
	for _, p := range []string{"al", "al;", "ar;", "en;", "ig;", "iga:", "il", "il-consumed;", "meta:identity", "meta:identity`", "meta:identityZ", "rs;", "rt;", "fa:", "att:", "ep:", "io:", "io-input:", "io-result:"} {
		excluded[p] = []byte{255}
	}
	riSet(t, st, excluded)
	before := riRaw(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	got := riScan(t, st)
	if got.Scanned != uint64(len(selected)) || !reflect.DeepEqual(got.Rows, selected) || !reflect.DeepEqual(got.FamilyCounts, wantCounts) || got.Digest != riDigest(selected) {
		t.Fatal("complete raw/framing/count mismatch")
	}
	if events != 0 || !reflect.DeepEqual(before, riRaw(t, st)) {
		t.Fatal("capture wrote bytes or events")
	}
	if _, present := got.Rows["al:absent"]; present {
		t.Fatal("invented absence")
	}
	if raw, present := got.Rows["al:"]; !present || len(raw) != 0 {
		t.Fatal("lost present empty")
	}
	if !reflect.DeepEqual(got.IndexTargets[""], []string{"al:"}) || !reflect.DeepEqual(got.IndexTargets[string([]byte{0, 255, 1})], []string{"al:\x00"}) {
		t.Fatal("opaque raw reverse ownership lost")
	}
}

func TestRIIndependentOccurrenceAndProjectionIsolation(t *testing.T) {
	st := openTemp(t)
	riSet(t, st, map[string][]byte{"en:z": riEntity(t, "z", "A/B", []string{"a/b", "a/b", "", "世界"}), "en:a": riEntity(t, "a", "a/b", []string{"A/B"}), "en:n": riEntity(t, "n", "Nil", nil)})
	got := riScan(t, st)
	want := []identityListing{{"a", "name", -1, "a/b"}, {"a", "alias", 0, "A/B"}, {"z", "name", -1, "A/B"}, {"z", "alias", 0, "a/b"}, {"z", "alias", 1, "a/b"}}
	if !reflect.DeepEqual(got.Listings["a/b"], want) || !reflect.DeepEqual(got.NaturalListings["ab"], want) {
		t.Fatal("occurrence ordering, duplicate or canonical name lost")
	}
	if len(got.Listings[""]) != 1 || len(got.NaturalListings[""]) != 2 || got.Entities["n"].Aliases != nil {
		t.Fatal("empty/nil observation lost")
	}
	if _, exists := got.Listings["z"]; exists {
		t.Fatal("slug invented as listing")
	}
	other := riScan(t, st)
	got.Listings["a/b"][0].Spelling = "changed"
	if !reflect.DeepEqual(got.NaturalListings["ab"], want) {
		t.Fatal("natural projection shares backing array")
	}
	got.Entities["z"].Aliases[0] = "changed"
	got.Entities["z"].RepoRefs[0] = "changed"
	if got.Listings["a/b"][3].Spelling != "a/b" || !bytes.Equal(got.Rows["en:z"], other.Rows["en:z"]) {
		t.Fatal("parsed entity shares data with listing/raw")
	}
	got.Rows["en:z"][0] = 0
	delete(got.Entities, "n")
	delete(got.NaturalListings, "")
	got.FamilyCounts["en:"] = 99
	if !reflect.DeepEqual(other, riScan(t, st)) {
		t.Fatal("independent capture/store ownership violation")
	}
}

func TestRIIndependentUntouchedOwnerAndLifecycleTransitions(t *testing.T) {
	st := openTemp(t)
	riSet(t, st, map[string][]byte{"en:a": riEntity(t, "a", "A", nil), "en:b": riEntity(t, "b", "B", []string{"shared"}), "al:shared": []byte("a"), "al:unlisted": []byte("a"), "al:missing": []byte("never-present")})
	before := riScan(t, st)
	if err := st.update(func(tx *badger.Txn) error { return tx.Delete([]byte("en:a")) }); err != nil {
		t.Fatal(err)
	}
	after := riScan(t, st)
	if before.Digest == after.Digest || !reflect.DeepEqual(before.Listings["shared"], after.Listings["shared"]) || !reflect.DeepEqual(before.IndexTargets, after.IndexTargets) {
		t.Fatal("unchanged alias relationship to deleted target not observable")
	}
	if _, exists := after.Entities["a"]; exists {
		t.Fatal("deleted entity fabricated")
	}
	for _, key := range []string{"ig:b", "il:b", "il-consumed:b", "meta:identity_adoption_v1", "rs:b", "rt:b", "ar:b:shared"} {
		prior := after
		riSet(t, st, map[string][]byte{key: {}})
		after = riScan(t, st)
		if prior.Digest == after.Digest {
			t.Fatal("lifecycle presence invisible")
		}
		if _, exists := after.Rows[key]; !exists {
			t.Fatal("empty marker absent")
		}
	}
	after.IndexTargets["a"][0] = "mutated"
	if !reflect.DeepEqual(riScan(t, st).IndexTargets, before.IndexTargets) {
		t.Fatal("reverse array shares store memory")
	}
}

func TestRIIndependentSnapshotStagingRollbackAndClosedFacades(t *testing.T) {
	st := openTemp(t)
	riSet(t, st, map[string][]byte{"en:a": riEntity(t, "a", "A", nil), "al:old": []byte("a")})
	base := riScan(t, st)
	stop := errors.New("synthetic rollback")
	var escaped *Store
	err := st.AtomicWrite(func(tx *Store) error {
		escaped = tx
		// Commit a root write after the facade snapshot was established.
		if err := st.db.Update(func(raw *badger.Txn) error { return raw.Set([]byte("rt:external"), []byte("later")) }); err != nil {
			return err
		}
		riSet(t, tx, map[string][]byte{"al:new": []byte("a"), "ig:a": {}})
		if err := tx.txn.Delete([]byte("al:old")); err != nil {
			return err
		}
		got := riScan(t, tx)
		if _, ok := got.Rows["rt:external"]; ok {
			t.Fatal("root newer state leaked into fixed transaction view")
		}
		if _, ok := got.Rows["al:old"]; ok {
			t.Fatal("staged deletion hidden")
		}
		if _, ok := got.Rows["ig:a"]; !ok || string(got.Rows["al:new"]) != "a" {
			t.Fatal("staged writes hidden")
		}
		return stop
	})
	if err != stop {
		t.Fatal(err)
	}
	after := riScan(t, st)
	if _, ok := after.Rows["al:new"]; ok {
		t.Fatal("rollback leaked scan-visible writes")
	}
	if !bytes.Equal(after.Rows["al:old"], base.Rows["al:old"]) || string(after.Rows["rt:external"]) != "later" {
		t.Fatal("fixture visibility mismatch")
	}
	for _, bad := range []*Store{nil, {}, escaped} {
		got, err := scanIdentityRelationships(bad)
		if err != errIdentityRelationships || !reflect.DeepEqual(got, identityRelationshipInventory{}) {
			t.Fatal("invalid facade did not globally refuse")
		}
	}
}

func TestRIIndependentReadOnlyRoot(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	riSet(t, st, map[string][]byte{"en:a": riEntity(t, "a", "A", nil), "rt:empty": {}})
	want := riScan(t, st)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithReadOnly(true))
	if err != nil {
		t.Fatal(err)
	}
	ro := &Store{db: db}
	if !reflect.DeepEqual(want, riScan(t, ro)) {
		t.Fatal("read-only root result mismatch")
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := scanIdentityRelationships(ro)
	if err != errIdentityRelationships || !reflect.DeepEqual(got, identityRelationshipInventory{}) {
		t.Fatal("closed root accepted")
	}
}

func TestRIIndependentMalformedEntitiesRefuseWithoutPartialData(t *testing.T) {
	canonical := riEntity(t, "z", "SecretSyntheticMarker", nil)
	cases := map[string][]byte{
		"present-empty": {}, "invalid-utf8": {255}, "null": []byte("null"),
		"trailing-space":  append(bytes.Clone(canonical), ' '),
		"unknown-field":   append(bytes.Clone(canonical[:len(canonical)-1]), []byte(`,"secret_extension":1}`)...),
		"duplicate-field": append(bytes.Clone(canonical[:len(canonical)-1]), []byte(`,"slug":"z"}`)...),
		"key-mismatch":    riEntity(t, "wrong", "Z", nil),
		"alias-null":      bytes.Replace(canonical, []byte(`,"created_at"`), []byte(`,"aliases":null,"created_at"`), 1),
		"invalid-time":    bytes.Replace(canonical, []byte("2026-09-06T01:02:03.000000004Z"), []byte("0001-01-01T00:00:00Z"), 1),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			st := openTemp(t)
			riSet(t, st, map[string][]byte{"en:a": riEntity(t, "a", "A", nil), "en:z": raw, "al:opaque": {255}, "rt:later": {}})
			before := riRaw(t, st)
			if _, ok := before["en:z"]; !ok {
				t.Fatal("bad fixture: malformed row is absent")
			}
			events := 0
			st.SetObserver(func(Event) { events++ })
			got, err := scanIdentityRelationships(st)
			if err != errIdentityRelationships || !reflect.DeepEqual(got, identityRelationshipInventory{}) {
				t.Fatal("malformed entity allowed partial result")
			}
			if strings.Contains(err.Error(), "SecretSyntheticMarker") || events != 0 || !reflect.DeepEqual(before, riRaw(t, st)) {
				t.Fatal("refusal leaked data or changed store")
			}
		})
	}
}
