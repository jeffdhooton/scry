package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestIndependentCorrectionEveryKnownDuplicate(t *testing.T) {
	f := referenceFixture()
	f.Dst, f.Value = "", "synthetic-attribute"
	f.InvalidAt = &f.ValidFrom
	key,raw := referenceBytes(t,f)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw,&fields); err != nil { t.Fatal("fixture decode failed") }
	if len(fields) != 10 { t.Fatal("fixture does not include every known field") }
	for name,value := range fields {
		for _,spelling := range []string{name,strings.ToUpper(name),fmt.Sprintf(`\u%04x%s`,name[0],name[1:])} {
			candidate := append(bytes.Clone(raw[:len(raw)-1]),[]byte(`,"`+spelling+`":`+string(value)+`}`)...)
			if _,err := decodeIdentityReference(key,candidate); !errors.Is(err,errIdentityFactReferences) { t.Fatal("duplicate known member accepted") }
		}
	}
}
