package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func independentInventoryDigest(snapshot map[string][]byte) string {
	keys := []string{}
	for key := range snapshot {
		if strings.HasPrefix(key, "fa:") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var b bytes.Buffer
	b.WriteString("identity-reference-inventory-v1")
	b.WriteByte(0)
	for _, key := range keys {
		binary.Write(&b, binary.BigEndian, uint64(len(key)))
		b.WriteString(key)
		binary.Write(&b, binary.BigEndian, uint64(len(snapshot[key])))
		b.Write(snapshot[key])
	}
	sum := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(sum[:])
}

func TestReferenceInventoryCompleteCountsAndExactDigest(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	put := func(f Fact) { key, raw := referenceBytes(t, f); gradeGenerationSet(t, st, string(key), raw) }
	put(f)
	f.Src = "historic-only"
	f.InvalidAt = &f.ValidFrom
	put(f)
	f.Src, f.Dst, f.Value = "attribute-source", "", "not-an-endpoint"
	f.InvalidAt = nil
	put(f)
	f.Src, f.Dst, f.Value = "loop", "loop", ""
	put(f)
	gradeGenerationSet(t, st, "opaque:unknown", []byte{255, 0})
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	r, err := scanIdentityReferenceInventory(st)
	want := map[string]identityReferenceCount{"source": {Current: 1}, "target": {Current: 1, Historical: 1}, "historic-only": {Historical: 1}, "attribute-source": {Current: 1}, "loop": {Current: 1}}
	if err != nil || r.Scanned != 4 || !reflect.DeepEqual(r.References, want) || r.Digest != independentInventoryDigest(before) {
		t.Fatal("wrong inventory")
	}
	slugs := []string{}
	for slug := range want {
		slugs = append(slugs, slug)
	}
	old, err := scanIdentityFactReferences(st, slugs)
	if err != nil || old.Scanned != r.Scanned || !reflect.DeepEqual(old.References, r.References) {
		t.Fatal("original checker differs")
	}
	r.References["source"] = identityReferenceCount{Current: 999}
	next, err := scanIdentityReferenceInventory(st)
	if err != nil || !reflect.DeepEqual(next.References, want) {
		t.Fatal("returned maps alias")
	}
	if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("read mutated graph")
	}
}

func TestReferenceInventoryDigestSeesAllRetainedBytes(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	base, err := scanIdentityReferenceInventory(st)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"sentence", "confidence", "provenance", "raw-relation", "unknown", "unknown-duplicate", "space"} {
		g := f
		changed := bytes.Clone(raw)
		switch kind {
		case "sentence":
			g.Fact += " Updated."
			changed, _ = json.Marshal(g)
		case "confidence":
			g.Confidence = .25
			changed, _ = json.Marshal(g)
		case "provenance":
			g.Episodes = []string{"other"}
			changed, _ = json.Marshal(g)
		case "raw-relation":
			g.RawRelation = "different"
			changed, _ = json.Marshal(g)
		case "unknown":
			changed = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":{"number":1e99999}}`)...)
		case "unknown-duplicate":
			changed = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":1,"unknown":2}`)...)
		case "space":
			changed = append([]byte("\n"), raw...)
		}
		gradeGenerationSet(t, st, string(key), changed)
		r, err := scanIdentityReferenceInventory(st)
		if err != nil || r.Digest == base.Digest || !reflect.DeepEqual(r.References, base.References) || r.Digest != independentInventoryDigest(gradeGenerationSnapshot(t, st)) {
			t.Fatal("raw change not fingerprinted", kind)
		}
	}
	gradeGenerationSet(t, st, string(key), raw)
	r, err := scanIdentityReferenceInventory(st)
	if err != nil || !reflect.DeepEqual(r, base) {
		t.Fatal("restored raw bytes changed identity")
	}
}

func TestReferenceInventoryRefusalsAndTransaction(t *testing.T) {
	st := openTemp(t)
	base, err := scanIdentityReferenceInventory(st)
	if err != nil || base.Scanned != 0 || len(base.References) != 0 || base.Digest != independentInventoryDigest(gradeGenerationSnapshot(t, st)) {
		t.Fatal("empty inventory incorrect")
	}
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	for _, bad := range [][]byte{[]byte("null"), []byte(`{"src":"sensitive-source"}`), append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"src":"other"}`)...), append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":"\ud800"}`)...)} {
		gradeGenerationSet(t, st, "fa:zz-unrelated", bad)
		before := gradeGenerationSnapshot(t, st)
		r, err := scanIdentityReferenceInventory(st)
		if !errors.Is(err, errIdentityFactReferences) || !reflect.DeepEqual(r, identityReferenceInventory{}) || strings.Contains(err.Error(), "sensitive-source") {
			t.Fatal("malformed record accepted/partial/leaky")
		}
		if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("refusal wrote")
		}
	}
	if err := st.db.Update(func(tx *badger.Txn) error { return tx.Delete([]byte("fa:zz-unrelated")) }); err != nil {
		t.Fatal(err)
	}
	before := gradeGenerationSnapshot(t, st)
	forced := errors.New("synthetic rollback")
	var escaped *Store
	err = st.AtomicWrite(func(tx *Store) error {
		escaped = tx
		initial, err := scanIdentityReferenceInventory(tx)
		if err != nil || initial.Scanned != 1 {
			t.Fatal("initial transaction inventory")
		}
		f.InvalidAt = &f.ValidFrom
		invalid, _ := json.Marshal(f)
		if err := tx.txn.Set(key, invalid); err != nil {
			return err
		}
		historical, err := scanIdentityReferenceInventory(tx)
		if err != nil || historical.References[f.Src].Historical != 1 || historical.References[f.Src].Current != 0 || historical.Digest == initial.Digest {
			t.Fatal("staged invalidation unseen")
		}
		f.Src = "new-source"
		f.InvalidAt = nil
		nextKey, nextRaw := referenceBytes(t, f)
		if err := tx.txn.Set(nextKey, nextRaw); err != nil {
			return err
		}
		inserted, err := scanIdentityReferenceInventory(tx)
		if err != nil || inserted.Scanned != 2 || inserted.References["new-source"].Current != 1 {
			t.Fatal("staged insert unseen")
		}
		if err := tx.txn.Delete(key); err != nil {
			return err
		}
		deleted, err := scanIdentityReferenceInventory(tx)
		if err != nil || deleted.Scanned != 1 || deleted.References["source"] != (identityReferenceCount{}) {
			t.Fatal("staged delete unseen")
		}
		return forced
	})
	if err != forced || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("measurement escaped rollback")
	}
	if r, err := scanIdentityReferenceInventory(escaped); err != errIdentityFactReferences || !reflect.DeepEqual(r, identityReferenceInventory{}) {
		t.Fatal("closed facade accepted")
	}
	if r, err := scanIdentityReferenceInventory(nil); err != errIdentityFactReferences || !reflect.DeepEqual(r, identityReferenceInventory{}) {
		t.Fatal("nil accepted")
	}
}

func TestPrivateReferenceInventoryReplica(t *testing.T) {
	path := os.Getenv("SCRY_PRIVATE_REFERENCE_INVENTORY_REPLICA")
	if path == "" {
		t.Skip("explicit private restored replica only")
	}
	if !strings.HasPrefix(path, "/tmp/scry-") {
		t.Fatal("refuse nonprivate path")
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal("private replica open failed")
	}
	defer st.Close()
	before := gradeGenerationSnapshot(t, st)
	started := time.Now()
	r, err := scanIdentityReferenceInventory(st)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatal("private raw inventory refused")
	}
	if r.Digest != independentInventoryDigest(before) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("private inventory/raw mismatch")
	}
	t.Logf("scanned=%d endpoint_slugs=%d fact_digest=%s elapsed=%s original_rows=%d raw_equal=true; root point-in-time measurement, not p95/support authority", r.Scanned, len(r.References), r.Digest, elapsed, len(before))
}
