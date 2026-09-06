package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func referenceFixture() Fact {
	return Fact{Src: "source", Relation: "uses", Dst: "target", RawRelation: "utilizes", Fact: "A synthetic source uses a target.", ValidFrom: time.Date(2026, 9, 6, 4, 0, 0, 123, time.UTC), Confidence: .875, Episodes: []string{"synthetic-episode"}}
}

func referenceBytes(t *testing.T, f Fact) ([]byte, []byte) {
	t.Helper()
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	return factKey(f.Src, f.Relation, f.KeyDst(), f.ValidFrom), raw
}

func TestIdentityReferencesCountAllRawEndpoints(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw) // Intentionally no en:/adj:.
	f.InvalidAt = &f.ValidFrom
	f.Src, f.Dst = "elsewhere", "target"
	key, raw = referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	f.InvalidAt = nil
	f.Src, f.Dst, f.Value = "source", "", "target"
	key, raw = referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	f.Src, f.Dst, f.Value = "source", "source", ""
	key, raw = referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	report, err := scanIdentityFactReferences(st, []string{"source", "target", "elsewhere", "absent", "source"})
	want := map[string]identityReferenceCount{"source": {Current: 3}, "target": {Current: 1, Historical: 1}, "elsewhere": {Historical: 1}, "absent": {}}
	if err != nil || report.Scanned != 4 || !reflect.DeepEqual(report.References, want) {
		t.Fatal("wrong raw reference counts")
	}
	if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("read changed store")
	}
}

func TestIdentityReferencesRejectKnownAmbiguity(t *testing.T) {
	f := referenceFixture()
	at := f.ValidFrom.Add(time.Second)
	f.InvalidAt = &at
	key, raw := referenceBytes(t, f)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for name, value := range fields {
		for _, spelling := range []string{name, strings.ToUpper(name), "\\u" + fmt.Sprintf("%04x", name[0]) + name[1:]} {
			t.Run("duplicate-"+name+"-"+spelling, func(t *testing.T) {
				bad := append(bytes.Clone(raw[:len(raw)-1]), []byte(",\""+spelling+"\":"+string(value)+"}")...)
				if _, err := decodeIdentityReference(key, bad); !errors.Is(err, errIdentityFactReferences) {
					t.Fatal("duplicate accepted")
				}
			})
		}
	}
	for _, field := range []string{"src", "relation", "dst", "fact", "valid_from", "confidence", "episodes"} {
		t.Run("missing-"+field, func(t *testing.T) {
			copyFields := map[string]json.RawMessage{}
			for k, v := range fields {
				if k != field {
					copyFields[k] = v
				}
			}
			bad, _ := json.Marshal(copyFields)
			if _, err := decodeIdentityReference(key, bad); err == nil {
				t.Fatal("missing field accepted")
			}
		})
	}
	for _, field := range []string{"src", "relation", "dst", "fact", "valid_from", "confidence", "raw_relation"} {
		t.Run("null-"+field, func(t *testing.T) {
			copyFields := map[string]json.RawMessage{}
			for k, v := range fields {
				copyFields[k] = v
			}
			copyFields[field] = json.RawMessage("null")
			bad, _ := json.Marshal(copyFields)
			if _, err := decodeIdentityReference(key, bad); err == nil {
				t.Fatal("null scalar accepted")
			}
		})
	}
	cases := [][]byte{
		[]byte("null"), []byte("[]"), append(bytes.Clone(raw), []byte(" {}")...), raw[:len(raw)-1],
		bytes.Replace(raw, []byte(`"src":"source"`), []byte(`"src":"wrong"`), 1),
		bytes.Replace(raw, []byte(`"dst":"target"`), []byte(`"dst":""`), 1),
		bytes.Replace(raw, []byte(`"dst":"target"`), []byte(`"dst":"target","value":"status"`), 1),
		bytes.Replace(raw, []byte(`"src":"source"`), []byte(`"src":"invalid:source"`), 1),
		bytes.Replace(raw, []byte(`"relation":"uses"`), []byte(`"relation":"bad:relation"`), 1),
		bytes.Replace(raw, []byte(`"fact":"A synthetic source uses a target."`), []byte(`"fact":"secret-\ud800"`), 1),
		bytes.Replace(raw, []byte(`"fact":"A synthetic source uses a target."`), []byte("\"fact\":\"secret-\xff\""), 1),
	}
	for i, bad := range cases {
		t.Run(fmt.Sprint("malformed-", i), func(t *testing.T) {
			if _, err := decodeIdentityReference(key, bad); !errors.Is(err, errIdentityFactReferences) || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "source") {
				t.Fatal("accepted ambiguity or leaked content")
			}
		})
	}
	for _, badKey := range [][]byte{[]byte("fa:other:uses:target:0"), append(bytes.Clone(key), '0'), []byte("fa:\xff")} {
		if _, err := decodeIdentityReference(badKey, raw); err == nil {
			t.Fatal("bad address accepted")
		}
	}
}

func TestIdentityReferencesUnknownExtensionsAndTime(t *testing.T) {
	f := referenceFixture()
	f.Fact = "A café uses 世界."
	for _, at := range []time.Time{f.ValidFrom, f.ValidFrom.In(time.FixedZone("offset", 3600)), time.Unix(0, math.MinInt64).UTC(), time.Unix(0, math.MaxInt64).UTC()} {
		f.ValidFrom = at
		key, raw := referenceBytes(t, f)
		raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"extension":1e999999,"extension":{"dst":"hidden","dst":"other"}}`)...)
		got, err := decodeIdentityReference(key, raw)
		if err != nil || !reflect.DeepEqual(got.Episodes, f.Episodes) || got.Fact != f.Fact || !got.ValidFrom.Equal(f.ValidFrom) {
			t.Fatal("valid extension or timestamp rejected")
		}
	}
	for _, at := range []time.Time{time.Time{}, time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC), time.Unix(0, math.MinInt64).Add(-time.Nanosecond), time.Unix(0, math.MaxInt64).Add(time.Nanosecond)} {
		f.ValidFrom = at
		key, raw := referenceBytes(t, f)
		if _, err := decodeIdentityReference(key, raw); err == nil {
			t.Fatal("overflowed temporal address accepted")
		}
	}
	f = referenceFixture()
	key, raw := referenceBytes(t, f)
	raw = bytes.Replace(raw, []byte(`"src"`), []byte(`"\u0073RC"`), 1)
	if _, err := decodeIdentityReference(key, raw); err != nil {
		t.Fatal("unambiguous escaped/case field rejected")
	}
}

func TestIdentityReferencesNoPartialResultAndTransactionalView(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	gradeGenerationSet(t, st, string(key), raw)
	before := gradeGenerationSnapshot(t, st)
	forced := errors.New("synthetic rollback")
	var escaped *Store
	err := st.AtomicWrite(func(tx *Store) error {
		escaped = tx
		if err := tx.txn.Delete(key); err != nil {
			return err
		}
		r, err := scanIdentityFactReferences(tx, []string{"source"})
		if err != nil || r.Scanned != 0 || r.References["source"].Current != 0 {
			t.Fatal("staged deletion invisible")
		}
		if err := tx.txn.Set(key, raw); err != nil {
			return err
		}
		r, err = scanIdentityFactReferences(tx, []string{"source"})
		if err != nil || r.Scanned != 1 || r.References["source"].Current != 1 {
			t.Fatal("staged insertion invisible")
		}
		if err := tx.txn.Set([]byte("fa:zz-unrelated"), []byte(`{"dst":"sensitive"}`)); err != nil {
			return err
		}
		r, err = scanIdentityFactReferences(tx, []string{"source"})
		if err == nil || r.Scanned != 0 || r.References != nil || strings.Contains(err.Error(), "sensitive") {
			t.Fatal("partial report or leaked content")
		}
		return forced
	})
	if !errors.Is(err, forced) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("read affected rollback")
	}
	if _, err := scanIdentityFactReferences(escaped, nil); err == nil {
		t.Fatal("expired facade accepted")
	}
	if _, err := scanIdentityFactReferences(nil, nil); err == nil {
		t.Fatal("nil store accepted")
	}
	for _, slug := range []string{"", "Bad Slug", "bad:slug", "invalid-\xff"} {
		if _, err := scanIdentityFactReferences(st, []string{slug}); err == nil {
			t.Fatal("invalid requested slug accepted")
		}
	}
}

func TestIdentityReferencesPrivateReplicaMeasurement(t *testing.T) {
	path := os.Getenv("SCRY_PRIVATE_REFERENCE_REPLICA")
	if path == "" {
		t.Skip("explicit private replica only")
	}
	st, err := Open(path)
	if err != nil {
		t.Fatal("private replica open failed")
	}
	defer st.Close()
	before := gradeGenerationSnapshot(t, st)
	started := time.Now()
	r, scanErr := scanIdentityFactReferences(st, []string{"scry"})
	elapsed := time.Since(started)
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("replica bytes changed")
	}
	if scanErr != nil {
		t.Logf("REFUSED (no writes): %v; elapsed=%s", scanErr, elapsed)
		return
	}
	t.Logf("validated_rows=%d elapsed=%s; zero raw changes", r.Scanned, elapsed)
}
