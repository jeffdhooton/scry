package store

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestLegacyAdoptionActualValueErrorSanitized(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithValueThreshold(4096))
	if err != nil {
		t.Fatal("synthetic DB open failed")
	}
	st := &Store{db: db}
	t.Cleanup(func() { st.Close() })
	small := Entity{Slug: "aaa", Name: "Synthetic small first anchor", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	raw, _ := json.Marshal(small)
	gradeGenerationSet(t, st, "en:aaa", raw)
	large := small
	large.Slug = "zzz"
	found := false
	for n := 3500; n < 4200; n++ {
		large.Name = strings.Repeat("s", n)
		raw, _ = json.Marshal(large)
		_, ar, err := makeLegacyIdentityAnchor(strings.Repeat("a", 64), []byte("en:zzz"), raw)
		if err != nil {
			t.Fatal("synthetic anchor failed")
		}
		if len(raw) <= 4096 && len(ar) > 4096 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("fixture does not cross actual value boundary")
	}
	gradeGenerationSet(t, st, "en:zzz", raw)
	manifest, err := captureLegacyInventory(st)
	if err != nil {
		t.Fatal(err)
	}
	before := gradeGenerationSnapshot(t, st)
	for _, apply := range []bool{false, true} {
		report, err := adoptLegacyInventory(st, manifest, apply)
		if err != errLegacyAdoption || report != (legacyAdoptionReport{}) {
			t.Fatal("unsafe or misclassified value error")
		}
		if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("early anchor leaked after late failure")
		}
	}
}

// Root-only actual replica measurement, never run on a live/original directory.
// This test deliberately APPLIES only to the explicitly restored disposable DB.
func TestPrivateLegacyAdoptionReplica(t *testing.T) {
	path := os.Getenv("SCRY_PRIVATE_ADOPTION_REPLICA")
	if path == "" {
		t.Skip("explicit separate restored replica only")
	}
	if !strings.HasPrefix(path, "/tmp/scry-legacy-adoption-") || !strings.Contains(path, "/replica-") {
		t.Fatal("refuse nonprivate replica path")
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal("private replica open failed")
	}
	defer st.Close()
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	manifest, err := captureLegacyInventory(st)
	if err != nil {
		t.Fatal("private capture failed")
	}
	m, id, err := decodeLegacyInventory(manifest)
	if err != nil {
		t.Fatal("private manifest decode failed")
	}
	started := time.Now()
	preview, err := adoptLegacyInventory(st, manifest, false)
	if err != nil {
		t.Fatal("private full preview refused")
	}
	previewTime := time.Since(started)
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) || events != 0 {
		t.Fatal("private preview wrote")
	}
	started = time.Now()
	report, err := adoptLegacyInventory(st, manifest, true)
	if err != nil {
		t.Fatal("private exact apply refused")
	}
	applyTime := time.Since(started)
	preview.Applied = true
	if report != preview {
		t.Fatal("private preview/apply differ")
	}
	after := gradeGenerationSnapshot(t, st)
	if len(after) != len(before)+len(m.Entries)+1 || events != 0 {
		t.Fatal("private write set/count/events differ")
	}
	for key, value := range before {
		if !bytes.Equal(value, after[key]) {
			t.Fatal("private original record changed")
		}
	}
	for _, entry := range m.Entries {
		e, _ := decodeLegacyEntity(entry.Key, entry.Value)
		a, got, err := readActiveLegacyIdentity(st, e.Slug)
		if err != nil || a.Inventory != id || !reflect.DeepEqual(got, e) {
			t.Fatal("private adopted identity not recognized")
		}
	}
	if !reflect.DeepEqual(after, gradeGenerationSnapshot(t, st)) {
		t.Fatal("private recognition changed store")
	}
	t.Logf("original_rows=%d entities=%d manifest_bytes=%d inventory_sha256=%s proposed_kv_bytes=%d preview=%s apply=%s original_raw_equal=true events=0; private replica only, NOT live/all-writer approval", len(before), len(m.Entries), len(manifest), id, report.Bytes, previewTime, applyTime)
}
