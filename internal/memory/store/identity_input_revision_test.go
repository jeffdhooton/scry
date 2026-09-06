package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func inputFixture() identityInputRevision {
	return identityInputRevision{Version: 1, EpisodeID: "synthetic-input", OccurredAt: time.Date(2026, 9, 6, 1, 2, 3, 123456789, time.UTC), Cwd: "/synthetic/世界", Summary: "Parsed summary 😀", Declarations: []observedDeclaration{{Name: "Aurora Guide", Type: "project", Description: "Synthetic declaration", Aliases: []string{"Aurora", "Aurora", "世界"}, TypeFallback: true}, {Name: "Atlas", Aliases: []string{}}}, Facts: []observedFact{{Src: "Aurora Guide", Relation: "uses", Dst: "Atlas", Fact: "Synthetic assertion", ValidFrom: "still unparsed", Confidence: .81, Supersedes: &observedSupersedes{Src: "Aurora Guide", Relation: "uses", Dst: "Former Atlas"}}}}
}

func inputEpisode(t *testing.T, st *Store, r identityInputRevision) {
	t.Helper()
	raw, err := json.Marshal(Episode{ID: r.EpisodeID, OccurredAt: r.OccurredAt, Summary: "Existing episode summary"})
	if err != nil {
		t.Fatal(err)
	}
	gradeGenerationSet(t, st, prefixEpisode+r.EpisodeID, raw)
}

func inputPut(t *testing.T, st *Store, r identityInputRevision) string {
	t.Helper()
	var key string
	if err := st.AtomicWrite(func(tx *Store) error { var err error; key, err = putIdentityInput(tx, r); return err }); err != nil {
		t.Fatal(err)
	}
	return key
}

func inputAssemble(t *testing.T, st *Store, r identityInputRevision, key string, budget int) []byte {
	t.Helper()
	all := []byte{}
	for offset, n := 0, 0; n < 10000; n++ {
		chunk, err := readIdentityInputChunk(st, r.EpisodeID, key, offset, budget)
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(chunk)
		if len(encoded) > budget || chunk.Key != key || chunk.Offset != offset {
			t.Fatal("bad envelope")
		}
		all = append(all, chunk.Data...)
		if chunk.NextOffset == -1 {
			return all
		}
		if chunk.NextOffset != offset+len(chunk.Data) || chunk.NextOffset <= offset {
			t.Fatal("no progress")
		}
		offset = chunk.NextOffset
	}
	t.Fatal("nonterminating reassembly")
	return nil
}

func TestInputRevisionFidelityAndDistinctInputs(t *testing.T) {
	r := inputFixture()
	raw, key, err := encodeIdentityInput(r)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeIdentityInput(key, raw, r.EpisodeID)
	if err != nil || !reflect.DeepEqual(r, decoded) {
		t.Fatal("lost explicit parsed fields")
	}
	decoded.Declarations[0].Aliases[0] = "changed"
	decoded.Facts[0].Supersedes.Dst = "changed"
	if r.Declarations[0].Aliases[0] != "Aurora" || r.Facts[0].Supersedes.Dst != "Former Atlas" {
		t.Fatal("decoded data not owned")
	}
	zoned := r
	zoned.OccurredAt = r.OccurredAt.In(time.FixedZone("seconds", 3723))
	b, k, err := encodeIdentityInput(zoned)
	if err != nil || k != key || !bytes.Equal(b, raw) || zoned.OccurredAt.Location() == time.UTC {
		t.Fatal("UTC fidelity or caller mutation")
	}
	for _, change := range []func(*identityInputRevision){
		func(v *identityInputRevision) { v.Summary += "x" }, func(v *identityInputRevision) { v.Cwd += "x" },
		func(v *identityInputRevision) { v.OccurredAt = v.OccurredAt.Add(time.Nanosecond) },
		func(v *identityInputRevision) { v.Declarations[0].TypeFallback = false },
		func(v *identityInputRevision) { v.Declarations[0].Aliases = []string{"Aurora", "世界", "Aurora"} },
		func(v *identityInputRevision) { v.Declarations[1].Aliases = nil },
		func(v *identityInputRevision) { v.Facts[0].ValidFrom += "x" },
		func(v *identityInputRevision) { v.Facts[0].Supersedes = nil },
		func(v *identityInputRevision) { v.Facts[0].Confidence = .82 },
	} {
		v := inputFixture()
		change(&v)
		b, k, err := encodeIdentityInput(v)
		if err != nil || k == key || bytes.Equal(b, raw) {
			t.Fatal("changed revision collapsed")
		}
	}
	empty := r
	empty.Declarations = nil
	empty.Facts = nil
	b, k, err = encodeIdentityInput(empty)
	if err != nil {
		t.Fatal(err)
	}
	empty.Declarations = []observedDeclaration{}
	empty.Facts = []observedFact{}
	b2, k2, err := encodeIdentityInput(empty)
	if err != nil || k == k2 || bytes.Equal(b, b2) {
		t.Fatal("nil/empty arrays collapsed")
	}
}

func TestInputRevisionStrictMalformed(t *testing.T) {
	r := inputFixture()
	raw, key, _ := encodeIdentityInput(r)
	for _, bad := range [][]byte{nil, []byte("null"), []byte("{}"), []byte("[]"), append([]byte(" "), raw...), append(bytes.Clone(raw), ' '), append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"version":1}`)...), append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"extra":true}`)...), bytes.Replace(raw, []byte(`"type_fallback":true`), []byte(`"type_fallback":true,"type_fallback":true`), 1), bytes.Replace(raw, []byte(`"summary":`), []byte(`"Summary":`), 1), bytes.Replace(raw, []byte(`"summary":`), []byte(`"ignored":`), 1), bytes.Replace(raw, []byte("Parsed summary 😀"), []byte(`\ud800`), 1)} {
		if _, err := decodeIdentityInput(key, bad, r.EpisodeID); err != errIdentityInput {
			t.Fatal("corrupt canonical row accepted")
		}
	}
	if _, err := decodeIdentityInput(key+"x", raw, r.EpisodeID); err == nil {
		t.Fatal("wrong key")
	}
	if _, err := decodeIdentityInput(key, raw, "other"); err == nil {
		t.Fatal("wrong episode")
	}
	for _, change := range []func(*identityInputRevision){
		func(v *identityInputRevision) { v.Version = 0 }, func(v *identityInputRevision) { v.EpisodeID = "" },
		func(v *identityInputRevision) { v.OccurredAt = time.Time{} }, func(v *identityInputRevision) { v.OccurredAt = time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC) },
		func(v *identityInputRevision) { v.Summary = "\xff" }, func(v *identityInputRevision) { v.Cwd = "\xff" },
		func(v *identityInputRevision) { v.Declarations[0].Aliases[0] = "\xff" }, func(v *identityInputRevision) { v.Declarations[0].Name = "\xff" },
		func(v *identityInputRevision) { v.Facts[0].Supersedes.Relation = "\xff" }, func(v *identityInputRevision) { v.Facts[0].Confidence = math.NaN() },
		func(v *identityInputRevision) { v.Facts[0].Confidence = math.Inf(1) },
	} {
		v := inputFixture()
		change(&v)
		if _, _, err := encodeIdentityInput(v); err != errIdentityInput {
			t.Fatal("lossy encoding accepted")
		}
	}
}

func TestInputRevisionFullObservationMatching(t *testing.T) {
	r := inputFixture()
	raw, key, _ := encodeIdentityInput(r)
	for _, origin := range []string{"declaration", "src", "dst"} {
		o := identityObservation{Version: 1, EpisodeID: r.EpisodeID, OccurredAt: r.OccurredAt, Cwd: r.Cwd, Ordinal: 0, Origin: "endpoint", Side: origin, Fact: &r.Facts[0]}
		if origin == "declaration" {
			o.Origin = "declaration"
			o.Side = ""
			o.Fact = nil
			o.Declaration = &r.Declarations[0]
		}
		ob, ok, err := encodeIdentityObservation(o)
		if err != nil || matchRevisionObservation(key, raw, ok, ob, r.EpisodeID) != nil {
			t.Fatal("matching full input refused")
		}
		for _, change := range []string{"payload", "cwd", "time", "ordinal", "supersedes-or-fallback"} {
			v, _ := decodeIdentityObservation(ok, ob, r.EpisodeID)
			switch change {
			case "payload":
				if v.Fact != nil {
					v.Fact.Fact += "different revision"
				} else {
					v.Declaration.Aliases = nil
				}
			case "cwd":
				v.Cwd += "other"
			case "time":
				v.OccurredAt = v.OccurredAt.Add(time.Nanosecond)
			case "ordinal":
				v.Ordinal = 100
			case "supersedes-or-fallback":
				if v.Fact != nil {
					v.Fact.Supersedes.Dst += "other"
				} else {
					v.Declaration.TypeFallback = false
				}
			}
			vb, vk, err := encodeIdentityObservation(v)
			if err != nil {
				t.Fatal("invalid mismatch fixture")
			}
			if matchRevisionObservation(key, raw, vk, vb, r.EpisodeID) != errIdentityInput {
				t.Fatal("mixed full revision accepted")
			}
		}
	}
}

func TestInputRevisionStoreInspectionAndNoRouting(t *testing.T) {
	st := openTemp(t)
	r := inputFixture()
	r.EpisodeID = strings.Repeat("synthetic-世界", 1000)
	inputEpisode(t, st, r)
	gradeGenerationSet(t, st, "opaque:sentinel", []byte{255, 0})
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	want := map[string][]byte{}
	for i := 0; i < 5; i++ {
		r.Summary = fmt.Sprint(i) + strings.Repeat("世界😀", 4000)
		key := inputPut(t, st, r)
		raw, _, _ := encodeIdentityInput(r)
		want[key] = raw
		if inputPut(t, st, r) != key {
			t.Fatal("replay changed key")
		}
		for _, budget := range []int{512, 1024, 24576} {
			if !bytes.Equal(inputAssemble(t, st, r, key, budget), raw) {
				t.Fatal("chunk data loss")
			}
		}
		end, err := readIdentityInputChunk(st, r.EpisodeID, key, len(raw), 512)
		if err != nil || end.NextOffset != -1 || len(end.Data) != 0 {
			t.Fatal("bad exact end")
		}
		if _, err := readIdentityInputChunk(st, r.EpisodeID, key, len(raw)+1, 512); err != errIdentityInput {
			t.Fatal("past end")
		}
	}
	expected := []string{}
	for k := range want {
		expected = append(expected, k)
	}
	sort.Strings(expected)
	for _, limit := range []int{1, 100} {
		got := []string{}
		cursor := ""
		for i := 0; i < 10; i++ {
			p, err := listIdentityInputKeys(st, r.EpisodeID, cursor, limit, 512)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(p)
			if len(b) > 512 {
				t.Fatal("page too large")
			}
			got = append(got, p.Keys...)
			if p.Next == "" {
				break
			}
			if p.Next == cursor {
				t.Fatal("stalled page")
			}
			cursor = p.Next
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatal("page omission or duplicate")
		}
	}
	for _, cursor := range []string{"missing", expected[0] + "x"} {
		p, err := listIdentityInputKeys(st, r.EpisodeID, cursor, 100, 512)
		if err != errIdentityInput || !reflect.DeepEqual(p, inputKeyPage{}) {
			t.Fatal("bad cursor accepted")
		}
	}
	after := gradeGenerationSnapshot(t, st)
	if len(after) != len(before)+5 || events != 0 {
		t.Fatal("unexpected graph write/events")
	}
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("old bytes altered")
		}
	}
	gradeGenerationSet(t, st, expected[4], []byte("corrupt"))
	if p, err := listIdentityInputKeys(st, r.EpisodeID, "", 100, 24576); err != errIdentityInput || !reflect.DeepEqual(p, inputKeyPage{}) {
		t.Fatal("partial corrupt page")
	}
}

func TestInputRevisionScopeProvenanceAndRollback(t *testing.T) {
	for _, kind := range []string{"missing-ep", "wrong-time", "duplicate-id", "occupied", "owner-swallowed", "panic", "rollback", "success"} {
		t.Run(kind, func(t *testing.T) {
			st := openTemp(t)
			r := inputFixture()
			inputEpisode(t, st, r)
			raw, key, _ := encodeIdentityInput(r)
			if kind == "missing-ep" {
				r.EpisodeID = "missing"
			}
			if kind == "wrong-time" {
				r.OccurredAt = r.OccurredAt.Add(time.Nanosecond)
			}
			if kind == "duplicate-id" {
				ep := gradeGenerationSnapshot(t, st)[prefixEpisode+r.EpisodeID]
				gradeGenerationSet(t, st, prefixEpisode+r.EpisodeID, append(bytes.Clone(ep[:len(ep)-1]), []byte(`,"id":"synthetic-input"}`)...))
			}
			if kind == "occupied" {
				gradeGenerationSet(t, st, key, append(bytes.Clone(raw), ' '))
			}
			before := gradeGenerationSnapshot(t, st)
			var escaped *Store
			var err error
			panicked := false
			func() {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				err = runIdentityAdmission(st, func(tx *Store) error {
					escaped = tx
					if err := tx.txn.Set([]byte("opaque:staged"), []byte("staged")); err != nil {
						return err
					}
					if kind == "owner-swallowed" {
						r.Version = 0
						_, _ = putIdentityInput(tx, r)
						return nil
					}
					if _, err := putIdentityInput(tx, r); err != nil {
						return err
					}
					if kind == "panic" {
						panic("synthetic panic")
					}
					if kind == "rollback" {
						return errors.New("synthetic rollback")
					}
					return nil
				}, func(*Store) error { return nil })
			}()
			if kind == "success" {
				if err != nil || panicked {
					t.Fatal("success refused")
				}
			} else {
				if (err == nil && !panicked) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
					t.Fatal("failed scope committed")
				}
			}
			if _, err := putIdentityInput(escaped, r); err != errIdentityInput {
				t.Fatal("closed writer accepted")
			}
			if _, err := listIdentityInputKeys(escaped, r.EpisodeID, "", 1, 512); err != errIdentityInput {
				t.Fatal("closed reader accepted")
			}
		})
	}
	for _, st := range []*Store{nil, {}} {
		if _, err := putIdentityInput(st, inputFixture()); err != errIdentityInput {
			t.Fatal("empty writer")
		}
		if _, err := listIdentityInputKeys(st, "x", "", 1, 512); err != errIdentityInput {
			t.Fatal("empty reader")
		}
	}
	st := openTemp(t)
	if _, err := putIdentityInput(st, inputFixture()); err != errIdentityInput {
		t.Fatal("root writer")
	}
}

func TestInputRevisionActualLateStorageErrors(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if err != nil {
		t.Fatal("synthetic db refused")
	}
	st := &Store{db: db}
	defer st.Close()
	r := inputFixture()
	inputEpisode(t, st, r)
	before := gradeGenerationSnapshot(t, st)
	for _, kind := range []string{"value", "capacity"} {
		staged := 0
		err := runIdentityAdmission(st, func(tx *Store) error {
			if _, err := putIdentityInput(tx, r); err != nil {
				return err
			}
			staged++
			v := inputFixture()
			if kind == "value" {
				v.Summary = strings.Repeat("synthetic-private-value ", 1000)
				_, err := putIdentityInput(tx, v)
				if err != errIdentityInput {
					t.Fatal("unsafe actual value error")
				}
				return nil
			}
			for i := 0; i < 2000; i++ {
				v.Summary = fmt.Sprint(i) + strings.Repeat("x", 2000)
				if _, err := putIdentityInput(tx, v); err != nil {
					if !errors.Is(err, badger.ErrTxnTooBig) {
						t.Fatal("capacity classification lost")
					}
					return nil
				}
				staged++
			}
			t.Fatal("capacity fixture did not fail")
			return nil
		}, func(*Store) error { t.Fatal("storage failure finalized"); return nil })
		if staged == 0 || !errors.Is(err, errIdentityInput) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("late failure leaked write")
		}
	}
}

func TestInputRevisionAdoptionReservedPrefix(t *testing.T) {
	for _, key := range []string{"io-input:", "io-input:malformed", "io-input:malformed:body"} {
		for _, apply := range []bool{false, true} {
			st := openTemp(t)
			_, manifest := adoptionFixture(t, st)
			gradeGenerationSet(t, st, key, []byte{})
			assertAdoptionRefusal(t, st, manifest, apply)
		}
	}
}
