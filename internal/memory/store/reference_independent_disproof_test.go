package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIndependentReferenceInvalidAtRange(t *testing.T) {
	for i, at := range []time.Time{time.Time{}, time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC), time.Unix(0, math.MinInt64).Add(-time.Nanosecond).UTC(), time.Unix(0, math.MaxInt64).Add(time.Nanosecond).UTC()} {
		f := referenceFixture()
		f.InvalidAt = &at
		key, raw := referenceBytes(t, f)
		if _, err := decodeIdentityReference(key, raw); !errors.Is(err, errIdentityFactReferences) {
			t.Errorf("unsupported optional timestamp accepted: case=%d", i)
		}
	}
}

func TestIndependentReferenceLossyUnknownUnicode(t *testing.T) {
	key, raw := referenceBytes(t, referenceFixture())
	for i, extension := range []string{`,"\ud800":true}`, `,"extension":"\ud800"}`, `,"extension":{"\udfff":true}}`, `,"extension":["\udfff"]}`} {
		candidate := append(bytes.Clone(raw[:len(raw)-1]), extension...)
		if _, err := decodeIdentityReference(key, candidate); !errors.Is(err, errIdentityFactReferences) {
			t.Errorf("lossy Unicode escape accepted: case=%d", i)
		}
	}
}

func TestIndependentReferenceLosslessKnownFields(t *testing.T) {
	f := referenceFixture()
	key, raw := referenceBytes(t, f)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil { t.Fatal("fixture decode failed") }
	cases := []struct{ field, value string }{
		{"confidence", "0.87500000000000000001"}, {"confidence", "8.75e-1"},
		{"episodes", `[null]`}, {"episodes", `["\ud800"]`}, {"episodes", `["synthetic-\u0065pisode"]`},
		{"episodes", `[1]`}, {"value", "null"}, {"invalid_at", `"2026-09-06T04:00:00.0000001230Z"`},
		{"invalid_at", `"2026-09-06T04:00:00+00:00"`}, {"fact", `"\u0041 synthetic source uses a target."`},
	}
	for i, tc := range cases {
		copyFields := make(map[string]json.RawMessage)
		for k,v := range fields { copyFields[k] = v }
		copyFields[tc.field] = json.RawMessage(tc.value)
		candidate, err := json.Marshal(copyFields)
		if err != nil { t.Fatal("fixture marshal failed") }
		if _, err := decodeIdentityReference(key, candidate); err == nil { t.Errorf("lossy or noncanonical known field accepted: case=%d", i) }
	}
}

func TestIndependentReferenceNullableAndEmptySemantics(t *testing.T) {
	f := referenceFixture()
	f.Episodes = nil
	f.Fact = ""
	f.RawRelation = ""
	f.Relation = "outside_current_vocabulary"
	key, raw := referenceBytes(t, f)
	raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"invalid_at":null,"value":"","raw_relation":""}`)...)
	if _, err := decodeIdentityReference(key, raw); err != nil { t.Fatal("canonical nullable or empty fields rejected") }
	for _, at := range []time.Time{time.Unix(0, math.MinInt64).UTC(), time.Unix(0, math.MaxInt64).UTC()} {
		f.InvalidAt = &at
		key, raw = referenceBytes(t, f)
		if _, err := decodeIdentityReference(key, raw); err != nil { t.Fatal("supported optional timestamp boundary rejected") }
	}
	f.InvalidAt = nil
	f.Fact = "世界 😀 café \ufffd <tag> & \u2028"
	f.Episodes = []string{"😀", "\ufffd"}
	key, raw = referenceBytes(t, f)
	if _, err := decodeIdentityReference(key, raw); err != nil { t.Fatal("canonical Unicode rejected") }
}

func TestIndependentReferenceCaseFoldEscapes(t *testing.T) {
	f := referenceFixture()
	f.Value, f.Dst = "synthetic-value", ""
	f.InvalidAt = &f.ValidFrom
	key, raw := referenceBytes(t, f)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil { t.Fatal("fixture decode failed") }
	for name, value := range fields {
		if !strings.Contains(name,"s") { continue }
		folded := strings.ReplaceAll(name,"s",`\u017f`)
		candidate := bytes.Replace(raw, []byte(`"`+name+`"`), []byte(`"`+folded+`"`), 1)
		if _, err := decodeIdentityReference(key,candidate); err != nil { t.Fatal("unambiguous Unicode casefold refused") }
		candidate = append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"`+folded+`":`+string(value)+`}`)...)
		if _, err := decodeIdentityReference(key,candidate); err == nil { t.Fatal("duplicate Unicode casefold accepted") }
	}
}

func TestIndependentReferenceCompleteScanOnEmptyRequest(t *testing.T) {
	st := openTemp(t)
	key,raw := referenceBytes(t,referenceFixture())
	gradeGenerationSet(t,st,string(key),raw)
	gradeGenerationSet(t,st,"fa:zz-synthetic",[]byte(`{"unexpected":true}`))
	before := gradeGenerationSnapshot(t,st)
	events := 0
	st.SetObserver(func(Event){ events++ })
	for _, requested := range [][]string{nil, {"unrelated"}} {
		r,err := scanIdentityFactReferences(st,requested)
		if !errors.Is(err,errIdentityFactReferences) || r.Scanned != 0 || r.References != nil { t.Fatal("empty or unrelated request bypassed fail-closed validation") }
	}
	if events != 0 || !reflect.DeepEqual(before,gradeGenerationSnapshot(t,st)) { t.Fatal("refusal changed raw store or emitted events") }
}

func TestIndependentReferenceClosedStore(t *testing.T) {
	st := openTemp(t)
	if err := st.Close(); err != nil { t.Fatal("close failed") }
	r, err := scanIdentityFactReferences(st,nil)
	if !errors.Is(err,errIdentityFactReferences) || r.Scanned != 0 || r.References != nil { t.Fatal("closed view accepted or returned partial result") }
}
