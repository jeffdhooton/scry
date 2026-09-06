package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func disproofInventoryRows(t *testing.T, st *Store) map[string][]byte {
	t.Helper()
	rows := map[string][]byte{}
	if err := st.view(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			raw, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			rows[string(it.Item().KeyCopy(nil))] = raw
		}
		return nil
	}); err != nil {
		t.Fatal("snapshot failed")
	}
	return rows
}

func disproofInventoryHash(rows map[string][]byte) string {
	keys := []string{}
	for k := range rows {
		if strings.HasPrefix(k, "fa:") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	framed := []byte("identity-reference-inventory-v1\x00")
	for _, k := range keys {
		for _, data := range [][]byte{[]byte(k), rows[k]} {
			n := uint64(len(data))
			for shift := 56; shift >= 0; shift -= 8 {
				framed = append(framed, byte(n>>shift))
			}
			framed = append(framed, data...)
		}
	}
	sum := sha256.Sum256(framed)
	return hex.EncodeToString(sum[:])
}

func disproofInventoryAssert(t *testing.T, st *Store, expected map[string]identityReferenceCount, n uint64) identityReferenceInventory {
	t.Helper()
	before := disproofInventoryRows(t, st)
	pending := 0
	if st.pendingEvents != nil {
		pending = len(*st.pendingEvents)
	}
	got, err := scanIdentityReferenceInventory(st)
	if err != nil || got.Scanned != n || !reflect.DeepEqual(got.References, expected) || got.Digest != disproofInventoryHash(before) {
		t.Fatal("inventory count or exact-frame digest mismatch")
	}
	requested := []string{"absent-reviewer-slug"}
	for slug := range expected {
		requested = append(requested, slug, slug)
	}
	prior, err := scanIdentityFactReferences(st, requested)
	if err != nil || prior.Scanned != got.Scanned {
		t.Fatal("original scan parity failed")
	}
	for _, slug := range requested {
		if prior.References[slug] != got.References[slug] {
			t.Fatal("original endpoint parity failed")
		}
	}
	if !reflect.DeepEqual(before, disproofInventoryRows(t, st)) {
		t.Fatal("scan wrote raw rows")
	}
	if st.pendingEvents != nil && len(*st.pendingEvents) != pending {
		t.Fatal("scan enqueued events")
	}
	return got
}

func TestIndependentInventoryEndpointMatrix(t *testing.T) {
	st := openTemp(t)
	events := 0
	st.SetObserver(func(Event) { events++ })
	expected := map[string]identityReferenceCount{}
	fixtures := []Fact{}
	for i := 0; i < 123; i++ {
		f := Fact{Src: fmt.Sprintf("source-%02d", i%7), Dst: fmt.Sprintf("target-%02d", i%9), Relation: "uses", Fact: "Synthetic independent inventory sentence.", ValidFrom: time.Date(2026, 9, 6, 8, 0, i, 13, time.UTC), Confidence: .5, Episodes: []string{"missing-episode"}}
		if i%4 == 0 {
			at := f.ValidFrom.Add(time.Minute)
			f.InvalidAt = &at
		}
		if i%3 == 0 {
			f.Dst = f.Src
		}
		if i%5 == 0 {
			f.Dst = ""
			f.Value = "attribute-only-endpoint-looking-slug"
		}
		fixtures = append(fixtures, f)
		endpoints := map[string]bool{f.Src: true}
		if f.Dst != "" {
			endpoints[f.Dst] = true
		}
		for slug := range endpoints {
			c := expected[slug]
			if f.InvalidAt == nil {
				c.Current++
			} else {
				c.Historical++
			}
			expected[slug] = c
		}
	}
	// Deliberately insert opposite to chronological order; address order drives hash.
	for i := len(fixtures) - 1; i >= 0; i-- {
		f := fixtures[i]
		raw, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		if i%11 == 0 {
			raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"extension\":{\"src\":\"opaque-only\",\"x\":1e999999,\"x\":2},\"extension\":[true,null]}")...)
		}
		if i == 0 {
			raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"large\":\""+strings.Repeat("x", 70001)+"\"}")...)
		}
		gradeGenerationSet(t, st, string(factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom)), raw)
	}
	for _, k := range []string{"f", "fa", "fa;other", "fb:bad", "en:fake", "ep:fake", "adj:fake"} {
		gradeGenerationSet(t, st, k, []byte{255, 0, 128})
	}
	got := disproofInventoryAssert(t, st, expected, 123)
	for k := range got.References {
		delete(got.References, k)
	}
	got.References["invented"] = identityReferenceCount{Current: 999}
	disproofInventoryAssert(t, st, expected, 123)
	if events != 0 {
		t.Fatal("scan emitted events")
	}
}

func TestIndependentInventoryRawOnlyChangesAndNonfacts(t *testing.T) {
	st := openTemp(t)
	empty := disproofInventoryAssert(t, st, map[string]identityReferenceCount{}, 0)
	gradeGenerationSet(t, st, "fa;irrelevant", []byte{255, 128, 0})
	gradeGenerationSet(t, st, "unrelated:opaque", []byte("not JSON"))
	again := disproofInventoryAssert(t, st, map[string]identityReferenceCount{}, 0)
	if empty.Digest != again.Digest {
		t.Fatal("nonfacts changed inventory hash")
	}
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	expected := map[string]identityReferenceCount{"source": {Current: 1}, "target": {Current: 1}}
	base := disproofInventoryAssert(t, st, expected, 1)
	variants := [][]byte{
		append([]byte("\n\t "), raw...),
		append(bytes.Clone(raw), []byte("\r\n")...),
		append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"opaque\":{\"u\":\"\\ud83d\\ude00\",\"n\":1e999999}}")...),
		append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"opaque\":1,\"opaque\":1}")...),
	}
	for i := 0; i < 4; i++ {
		g := f
		switch i {
		case 0:
			g.Fact += " Changed sentence."
		case 1:
			g.RawRelation = "different relation text"
		case 2:
			g.Confidence = .125
		case 3:
			g.Episodes = []string{"unretained-other-episode"}
		}
		v, err := json.Marshal(g)
		if err != nil {
			t.Fatal(err)
		}
		variants = append(variants, v)
	}
	seen := map[string]bool{base.Digest: true}
	for _, v := range variants {
		gradeGenerationSet(t, st, string(key), v)
		r := disproofInventoryAssert(t, st, expected, 1)
		if seen[r.Digest] {
			t.Fatal("distinct retained bytes collided")
		}
		seen[r.Digest] = true
	}
	gradeGenerationSet(t, st, string(key), raw)
	restored := disproofInventoryAssert(t, st, expected, 1)
	if !reflect.DeepEqual(restored, base) {
		t.Fatal("identical byte restoration diverged")
	}
}

func TestIndependentInventoryGlobalRefusalMatrix(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	f.Src = "zz-secret-canary"
	badKey, valid := referenceBytes(t, f)
	bads := [][]byte{[]byte("null"), []byte("[]"), append(bytes.Clone(valid), []byte("{}")...), bytes.Replace(valid, []byte("zz-secret-canary"), []byte("key-mismatch"), 1), append([]byte{255}, valid...)}
	fields := []string{"src", "relation", "dst", "value", "raw_relation", "fact", "valid_from", "invalid_at", "confidence", "episodes"}
	for _, field := range fields {
		var members map[string]json.RawMessage
		if err := json.Unmarshal(valid, &members); err != nil {
			t.Fatal(err)
		}
		value, ok := members[field]
		if !ok {
			value = []byte("null")
		}
		// Ensure fields omitted by omitempty first occur, then repeat them.
		base := valid
		if !ok {
			base = append(bytes.Clone(valid[:len(valid)-1]), []byte(fmt.Sprintf(",%q:%s}", field, value))...)
		}
		bads = append(bads, append(bytes.Clone(base[:len(base)-1]), []byte(fmt.Sprintf(",%q:%s}", strings.ToUpper(field), value))...))
	}
	bads = append(bads, append(bytes.Clone(valid[:len(valid)-1]), []byte(",\"opaque\":\"\\ud800\"}")...))
	for i, bad := range bads {
		gradeGenerationSet(t, st, string(badKey), bad)
		before := disproofInventoryRows(t, st)
		events := 0
		st.SetObserver(func(Event) { events++ })
		r, err := scanIdentityReferenceInventory(st)
		if !errors.Is(err, errIdentityFactReferences) || !reflect.DeepEqual(r, identityReferenceInventory{}) {
			t.Fatalf("case %d accepted or partial", i)
		}
		if strings.Contains(err.Error(), "secret-canary") || strings.Contains(err.Error(), "key-mismatch") {
			t.Fatal("raw diagnostic escaped")
		}
		prior, e := scanIdentityFactReferences(st, []string{"source"})
		if !errors.Is(e, errIdentityFactReferences) || !reflect.DeepEqual(prior, identityReferenceReport{}) {
			t.Fatal("refusal diverges from original")
		}
		if events != 0 || !reflect.DeepEqual(before, disproofInventoryRows(t, st)) {
			t.Fatal("refusal mutated")
		}
	}
	gradeGenerationSet(t, st, string(badKey), nil)
	disproofInventoryAssert(t, st, map[string]identityReferenceCount{"source": {Current: 1}, "target": {Current: 1}}, 1)
}

func TestIndependentInventoryTransactionCommitRollbackClosed(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	base := disproofInventoryRows(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	abort := errors.New("independent abort")
	for _, commit := range []bool{false, true} {
		var escaped *Store
		err := st.AtomicWrite(func(tx *Store) error {
			escaped = tx
			disproofInventoryAssert(t, tx, map[string]identityReferenceCount{"source": {Current: 1}, "target": {Current: 1}}, 1)
			at := f.ValidFrom
			f.InvalidAt = &at
			historical, _ := json.Marshal(f)
			if err := tx.txn.Set(key, historical); err != nil {
				return err
			}
			disproofInventoryAssert(t, tx, map[string]identityReferenceCount{"source": {Historical: 1}, "target": {Historical: 1}}, 1)
			g := f
			g.Src = "staged"
			g.Dst = "staged"
			g.InvalidAt = nil
			k, v := referenceBytes(t, g)
			if err := tx.txn.Set(k, v); err != nil {
				return err
			}
			disproofInventoryAssert(t, tx, map[string]identityReferenceCount{"source": {Historical: 1}, "target": {Historical: 1}, "staged": {Current: 1}}, 2)
			if err := tx.txn.Delete(key); err != nil {
				return err
			}
			disproofInventoryAssert(t, tx, map[string]identityReferenceCount{"staged": {Current: 1}}, 1)
			if !commit {
				return abort
			}
			return nil
		})
		if (!commit && err != abort) || (commit && err != nil) {
			t.Fatal("transaction outcome mismatch")
		}
		r, e := scanIdentityReferenceInventory(escaped)
		if e != errIdentityFactReferences || !reflect.DeepEqual(r, identityReferenceInventory{}) {
			t.Fatal("escaped closed scope accepted")
		}
		if !commit && !reflect.DeepEqual(base, disproofInventoryRows(t, st)) {
			t.Fatal("rollback changed raw bytes")
		}
	}
	disproofInventoryAssert(t, st, map[string]identityReferenceCount{"staged": {Current: 1}}, 1)
	if events != 0 {
		t.Fatal("scan emitted events")
	}
	if err := st.Close(); err != nil {
		t.Fatal("close failed")
	}
	r, err := scanIdentityReferenceInventory(st)
	if err != errIdentityFactReferences || !reflect.DeepEqual(r, identityReferenceInventory{}) {
		t.Fatal("closed DB accepted")
	}
}
