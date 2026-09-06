package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIndependentCorrectionEveryUTF16Unit(t *testing.T) {
	key, raw := referenceBytes(t, referenceFixture())
	for unit := 0; unit <= 0xffff; unit++ {
		extension := fmt.Sprintf(`,"extension":"\u%04x"}`, unit)
		candidate := append(bytes.Clone(raw[:len(raw)-1]), extension...)
		before := bytes.Clone(candidate)
		_, err := decodeIdentityReference(key, candidate)
		wantAccept := unit < 0xd800 || unit > 0xdfff
		if (err == nil) != wantAccept { t.Fatalf("single UTF16 unit classification mismatch: index=%d",unit) }
		if !bytes.Equal(candidate,before) { t.Fatal("decoder changed raw bytes") }
	}
}

func TestIndependentCorrectionSurrogatePairMatrix(t *testing.T) {
	key, raw := referenceBytes(t, referenceFixture())
	for offset := 0; offset < 1024; offset++ {
		for _, pair := range [][2]int{{0xd800+offset,0xdc00},{0xd800+offset,0xdfff},{0xd800,0xdc00+offset},{0xdbff,0xdc00+offset}} {
			// Uppercase hex is valid in opaque names and values, even though known
			// string values intentionally require their canonical encoding.
			extension := fmt.Sprintf(`,"\u%04X\u%04X":{"extension":["\u%04X\u%04X"]}}`,pair[0],pair[1],pair[0],pair[1])
			candidate := append(bytes.Clone(raw[:len(raw)-1]),extension...)
			if _,err := decodeIdentityReference(key,candidate); err != nil { t.Fatalf("valid surrogate pair refused: index=%d",offset) }
		}
	}
}

func TestIndependentCorrectionEscapeParityAndOpaqueBytes(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	for parity := 0; parity < 20; parity++ {
		f.ValidFrom = f.ValidFrom.Add(time.Nanosecond)
		key, raw := referenceBytes(t,f)
		value := strings.Repeat(`\`,parity)+`"literal \ud800 / 世界 😀 `+string(rune(0x10000+parity))
		encoded,err := json.Marshal(value)
		if err != nil { t.Fatal("fixture marshal failed") }
		extension := append([]byte(`,"extension":`),encoded...)
		extension = append(extension,[]byte(`,"extension":{"nested":1,"nested":2},"huge":1e999999,"huge":-12345678901234567890123456789012345678901234567890}`)...)
		raw = append(bytes.Clone(raw[:len(raw)-1]),extension...)
		gradeGenerationSet(t,st,string(key),raw)
	}
	before := gradeGenerationSnapshot(t,st)
	events := 0
	st.SetObserver(func(Event){ events++ })
	r,err := scanIdentityFactReferences(st,[]string{f.Src,f.Dst})
	if err != nil || r.Scanned != 20 || r.References[f.Src].Current != 20 || r.References[f.Dst].Current != 20 { t.Fatal("opaque valid controls not counted") }
	if events != 0 || !reflect.DeepEqual(before,gradeGenerationSnapshot(t,st)) { t.Fatal("opaque raw bytes or events changed") }
}

func TestIndependentCorrectionMalformedStagedHistoryFailsWholeScan(t *testing.T) {
	st := openTemp(t)
	f := referenceFixture()
	key,raw := referenceBytes(t,f)
	gradeGenerationSet(t,st,string(key),raw)
	before := gradeGenerationSnapshot(t,st)
	events := 0
	st.SetObserver(func(Event){ events++ })
	stop := errors.New("synthetic rollback")
	err := st.AtomicWrite(func(tx *Store) error {
		at := time.Unix(0,math.MaxInt64).Add(time.Nanosecond).UTC()
		f.InvalidAt = &at
		f.Src = "zz-synthetic"
		badKey,badRaw := referenceBytes(t,f)
		if err := tx.txn.Set(badKey,badRaw); err != nil { return err }
		r,err := scanIdentityFactReferences(tx,[]string{"source"})
		if !errors.Is(err,errIdentityFactReferences) || r.Scanned != 0 || r.References != nil { t.Fatal("staged unsupported history did not fail whole scan") }
		return stop
	})
	if !errors.Is(err,stop) || events != 0 || !reflect.DeepEqual(before,gradeGenerationSnapshot(t,st)) { t.Fatal("failed scan affected rollback or emitted events") }
}
