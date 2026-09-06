package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func adoptionFixture(t *testing.T, st *Store) (Entity, []byte) {
	t.Helper()
	e, key, value := legacyAnchorFixture(t)
	gradeGenerationSet(t, st, string(key), value)
	gradeGenerationSet(t, st, "opaque:unknown", []byte{0, 255, 1})
	gradeGenerationSet(t, st, "al:old-listing", []byte("unexplained-owner"))
	gradeGenerationSet(t, st, "fa:retained-opaque", []byte("unparsed-old-fact"))
	manifest, err := captureLegacyInventory(st)
	if err != nil {
		t.Fatal(err)
	}
	return e, manifest
}

func assertAdoptionRefusal(t *testing.T, st *Store, raw []byte, apply bool) {
	t.Helper()
	before := gradeGenerationSnapshot(t, st)
	report, err := adoptLegacyInventory(st, raw, apply)
	if !errors.Is(err, errLegacyAdoption) || report != (legacyAdoptionReport{}) {
		t.Fatal("missing refusal or partial report")
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("refusal changed raw store")
	}
}

func TestLegacyAdoptionFullPreviewApplyAndRead(t *testing.T) {
	st := openTemp(t)
	e, raw := adoptionFixture(t, st)
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	preview, err := adoptLegacyInventory(st, raw, false)
	if err != nil || preview.Applied || preview.Entities != 1 || preview.Bytes == 0 {
		t.Fatal("bad preview", err)
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) || events != 0 {
		t.Fatal("preview committed")
	}
	actual, err := adoptLegacyInventory(st, raw, true)
	want := preview
	want.Applied = true
	if err != nil || actual != want {
		t.Fatal("apply differs", err)
	}
	after := gradeGenerationSnapshot(t, st)
	if len(after) != len(before)+2 || events != 0 {
		t.Fatal("wrong write set/events")
	}
	for key, value := range before {
		if !bytes.Equal(value, after[key]) {
			t.Fatal("old record changed")
		}
	}
	a, got, err := readActiveLegacyIdentity(st, e.Slug)
	if err != nil || !reflect.DeepEqual(got, e) || a.Inventory != preview.Inventory {
		t.Fatal("read does not match adoption", err)
	}
	assertAdoptionRefusal(t, st, raw, false)
	assertAdoptionRefusal(t, st, raw, true)
	got.Description = "new metadata"
	updated, _ := json.Marshal(got)
	gradeGenerationSet(t, st, prefixEntity+e.Slug, updated)
	if _, _, err := readActiveLegacyIdentity(st, e.Slug); err != nil {
		t.Fatal("metadata update refused")
	}
}

func TestLegacyAdoptionManifestCanonicalAndExactDrift(t *testing.T) {
	for _, kind := range []string{"nil", "null", "empty", "duplicate", "unknown", "space", "null-array", "order", "same-key", "missing", "extra", "metadata", "unknown-source"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			e, raw := adoptionFixture(t, st)
			switch kind {
			case "nil":
				raw = nil
			case "null":
				raw = []byte("null")
			case "empty":
				raw = []byte("{}")
			case "duplicate":
				raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"version":1}`)...)
			case "unknown":
				raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"future":0}`)...)
			case "space":
				raw = append(raw, ' ')
			case "null-array":
				raw = []byte(`{"version":1,"entries":null}`)
			case "order", "same-key":
				m, _, _ := decodeLegacyInventory(raw)
				other := m.Entries[0]
				if kind == "order" {
					e.Slug = "aaa"
					other.Key = []byte("en:aaa")
					other.Value, _ = json.Marshal(e)
				}
				m.Entries = append(m.Entries, other)
				raw, _ = json.Marshal(m)
			case "missing":
				if err := st.db.Update(func(tx *badger.Txn) error { return tx.Delete([]byte(prefixEntity + e.Slug)) }); err != nil {
					t.Fatal(err)
				}
			case "extra":
				e.Slug = "extra"
				value, _ := json.Marshal(e)
				gradeGenerationSet(t, st, "en:extra", value)
			case "metadata":
				e.Description += " changed"
				value, _ := json.Marshal(e)
				gradeGenerationSet(t, st, prefixEntity+e.Slug, value)
			case "unknown-source":
				m, _, _ := decodeLegacyInventory(raw)
				value := m.Entries[0].Value
				m.Entries[0].Value = append(bytes.Clone(value[:len(value)-1]), []byte(`,"extension":1}`)...)
				raw, _ = json.Marshal(m)
			}
			assertAdoptionRefusal(t, st, raw, false)
			assertAdoptionRefusal(t, st, raw, true)
		})
	}
	st := openTemp(t)
	raw, err := captureLegacyInventory(st)
	if err != nil || string(raw) != `{"version":1,"entries":[]}` {
		t.Fatal("empty inventory not explicit")
	}
	before := gradeGenerationSnapshot(t, st)
	if r, err := adoptLegacyInventory(st, raw, false); err != nil || r.Entities != 0 {
		t.Fatal("empty preview refused")
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("empty preview wrote")
	}
	if _, err := adoptLegacyInventory(st, raw, true); err != nil {
		t.Fatal("explicit empty adoption failed")
	}
}

func TestLegacyAdoptionReservedAndFacadeRefusal(t *testing.T) {
	for _, key := range []string{"il:", "il-consumed:unknown", "ig:x", "iga:x", "io:x", "io-result:x", "meta:identity_future", legacyAdoptionMarkerKey} {
		st := openTemp(t)
		_, raw := adoptionFixture(t, st)
		gradeGenerationSet(t, st, key, []byte{255})
		assertAdoptionRefusal(t, st, raw, false)
		assertAdoptionRefusal(t, st, raw, true)
	}
	st := openTemp(t)
	_, raw := adoptionFixture(t, st)
	before := gradeGenerationSnapshot(t, st)
	err := st.AtomicWrite(func(tx *Store) error {
		if err := tx.txn.Set([]byte("opaque:staged"), []byte("must-rollback")); err != nil {
			return err
		}
		_, _ = adoptLegacyInventory(tx, raw, true)
		return nil
	})
	if !errors.Is(err, errLegacyAdoption) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("caught nested refusal committed")
	}
	for _, phase := range []string{"body", "finalizer", "closed"} {
		var escaped *Store
		call := func(tx *Store) error { _, _ = adoptLegacyInventory(tx, raw, false); return nil }
		body := func(tx *Store) error {
			escaped = tx
			if phase == "body" {
				return call(tx)
			}
			return nil
		}
		finalizer := func(tx *Store) error {
			if phase == "finalizer" {
				return call(tx)
			}
			return nil
		}
		err := runIdentityAdmission(st, body, finalizer)
		if phase == "closed" {
			if err != nil {
				t.Fatal(err)
			}
			_, err = adoptLegacyInventory(escaped, raw, true)
		}
		if !errors.Is(err, errLegacyAdoption) {
			t.Fatal("owned phase accepted adoption")
		}
		if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("owned refusal changed store")
		}
	}
}

func TestLegacyAdoptionReaderFailsClosed(t *testing.T) {
	for _, kind := range []string{"no-marker", "bad-marker", "wrong-inventory", "no-anchor", "bad-anchor", "no-entity", "renamed", "creation", "selected", "consumed", "corrupt-consumed", "recreated"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			e, raw := adoptionFixture(t, st)
			if _, err := adoptLegacyInventory(st, raw, true); err != nil {
				t.Fatal(err)
			}
			snapshot := gradeGenerationSnapshot(t, st)
			key, value, remove := "", []byte(nil), false
			switch kind {
			case "no-marker":
				key, remove = legacyAdoptionMarkerKey, true
			case "bad-marker":
				key, value = legacyAdoptionMarkerKey, []byte(`{"version":1}`)
			case "wrong-inventory":
				m, _ := decodeLegacyAdoptionMarker(snapshot[legacyAdoptionMarkerKey])
				m.Inventory = strings.Repeat("0", 64)
				value, _ = json.Marshal(m)
				key = legacyAdoptionMarkerKey
			case "no-anchor":
				key, remove = "il:"+e.Slug, true
			case "bad-anchor":
				key, value = "il:"+e.Slug, []byte("null")
			case "no-entity":
				key, remove = prefixEntity+e.Slug, true
			case "renamed":
				e.Name += "changed"
				key = prefixEntity + e.Slug
				value, _ = json.Marshal(e)
			case "creation":
				e.CreatedAt = e.CreatedAt.Add(time.Nanosecond)
				key = prefixEntity + e.Slug
				value, _ = json.Marshal(e)
			case "selected":
				key, value = "ig:"+e.Slug, []byte("even-malformed")
			case "corrupt-consumed":
				key, value = "il-consumed:"+e.Slug, []byte{255}
			case "consumed", "recreated":
				k, v, err := encodeLegacyIdentityConsumption(legacyIdentityConsumption{Version: 1, AnchorKey: []byte("il:" + e.Slug), Anchor: snapshot["il:"+e.Slug], Operation: strings.Repeat("a", 64), Reason: "retire"})
				if err != nil {
					t.Fatal(err)
				}
				key, value = string(k), v
				if kind == "recreated" {
					if err := st.db.Update(func(tx *badger.Txn) error { return tx.Delete([]byte(prefixEntity + e.Slug)) }); err != nil {
						t.Fatal(err)
					}
					gradeGenerationSet(t, st, prefixEntity+e.Slug, snapshot[prefixEntity+e.Slug])
				}
			}
			if remove {
				if err := st.db.Update(func(tx *badger.Txn) error { return tx.Delete([]byte(key)) }); err != nil {
					t.Fatal(err)
				}
			} else {
				gradeGenerationSet(t, st, key, value)
			}
			before := gradeGenerationSnapshot(t, st)
			a, got, err := readActiveLegacyIdentity(st, e.Slug)
			if err != errLegacyAdoption || a != (legacyIdentityAnchor{}) || !reflect.DeepEqual(got, Entity{}) {
				t.Fatal("bad identity accepted/partial returned")
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("read mutated store")
			}
		})
	}
}

func TestLegacyAdoptionActualTransactionLimitRollback(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(64 << 10))
	if err != nil {
		t.Fatal(err)
	}
	st := &Store{db: db}
	t.Cleanup(func() { st.Close() })
	for i := 0; i < 1600; i++ {
		e := Entity{Slug: fmt.Sprintf("identity-%04d", i), Name: "Synthetic exact identity", Type: "project", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		raw, _ := json.Marshal(e)
		gradeGenerationSet(t, st, prefixEntity+e.Slug, raw)
	}
	raw, err := captureLegacyInventory(st)
	if err != nil {
		t.Fatal(err)
	}
	before := gradeGenerationSnapshot(t, st)
	for _, apply := range []bool{false, true} {
		report, err := adoptLegacyInventory(st, raw, apply)
		if !errors.Is(err, badger.ErrTxnTooBig) || !errors.Is(err, errLegacyAdoption) || report != (legacyAdoptionReport{}) {
			t.Fatal("real oversized adoption not classified", err)
		}
		if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("partial anchors survived failed batch")
		}
	}
}

func TestLegacyAdoptionWaitsForOrdinaryWriter(t *testing.T) {
	st := openTemp(t)
	_, raw := adoptionFixture(t, st)
	entered, release := make(chan struct{}), make(chan struct{})
	writer := make(chan error, 1)
	go func() {
		writer <- st.AtomicWrite(func(tx *Store) error {
			close(entered)
			<-release
			return tx.PutEntity(Entity{Slug: "concurrent-new", Name: "Concurrent New", Type: "project", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
		})
	}()
	<-entered
	result := make(chan error, 1)
	go func() { _, err := adoptLegacyInventory(st, raw, true); result <- err }()
	select {
	case <-result:
		close(release)
		<-writer
		t.Fatal("adoption escaped active writer lock")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if err := <-writer; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, errLegacyAdoption) {
			t.Fatal("concurrent added identity not refused")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("adoption deadlocked")
	}
	for key := range gradeGenerationSnapshot(t, st) {
		if strings.HasPrefix(key, "il:") || key == legacyAdoptionMarkerKey {
			t.Fatal("stale adoption committed")
		}
	}
}
