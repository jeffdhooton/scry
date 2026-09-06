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

// These are independently authored disproofs; fixtures contain synthetic data only.
func independentInputStrings(r *identityInputRevision) []*string {
	return []*string{&r.EpisodeID, &r.Cwd, &r.Summary, &r.Declarations[0].Name, &r.Declarations[0].Type, &r.Declarations[0].Description, &r.Declarations[0].Aliases[0], &r.Facts[0].Src, &r.Facts[0].Relation, &r.Facts[0].Dst, &r.Facts[0].Fact, &r.Facts[0].ValidFrom, &r.Facts[0].Supersedes.Src, &r.Facts[0].Supersedes.Relation, &r.Facts[0].Supersedes.Dst}
}

func TestIndependentInputEveryStringAndFloatFidelity(t *testing.T) {
	for i := 0; i < len(independentInputStrings(newInputFixture())); i++ {
		for _, value := range []string{"", " A\x00\n世界😀\u2028\u2029<&> ", "e\u0301", "\ufffd"} {
			r := inputFixture()
			*independentInputStrings(&r)[i] = value
			raw, key, err := encodeIdentityInput(r)
			if i == 0 && value == "" {
				if err != errIdentityInput {
					t.Fatal("empty episode accepted")
				}
				continue
			}
			if err != nil {
				t.Fatalf("string field %d refused valid bytes", i)
			}
			got, err := decodeIdentityInput(key, raw, r.EpisodeID)
			if err != nil || !reflect.DeepEqual(got, r) {
				t.Fatalf("string field %d lost bytes", i)
			}
		}
		r := inputFixture()
		*independentInputStrings(&r)[i] = "\xff"
		if b, k, e := encodeIdentityInput(r); e != errIdentityInput || b != nil || k != "" {
			t.Fatalf("invalid unicode field %d accepted", i)
		}
	}
	for _, confidence := range []float64{math.Copysign(0, -1), 0, math.SmallestNonzeroFloat64, math.MaxFloat64, -math.MaxFloat64, -2, 1.0000000000000002} {
		r := inputFixture()
		r.Facts[0].Confidence = confidence
		raw, key, err := encodeIdentityInput(r)
		if err != nil {
			t.Fatal("finite confidence refused")
		}
		decoded, err := decodeIdentityInput(key, raw, r.EpisodeID)
		if err != nil || math.Float64bits(decoded.Facts[0].Confidence) != math.Float64bits(confidence) {
			t.Fatal("confidence bits lost")
		}
	}
	for _, n := range []int64{math.MinInt64, math.MaxInt64, -1, 0, 1} {
		r := inputFixture()
		r.OccurredAt = time.Unix(0, n).UTC()
		raw, key, err := encodeIdentityInput(r)
		if err != nil {
			t.Fatal("representable nanosecond boundary refused")
		}
		got, err := decodeIdentityInput(key, raw, r.EpisodeID)
		if err != nil || !got.OccurredAt.Equal(r.OccurredAt) {
			t.Fatal("time boundary lost")
		}
	}
	for _, at := range []time.Time{time.Unix(0, math.MaxInt64).Add(time.Nanosecond), time.Unix(0, math.MinInt64).Add(-time.Nanosecond), time.Time{}, time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC)} {
		r := inputFixture()
		r.OccurredAt = at
		if _, _, err := encodeIdentityInput(r); err != errIdentityInput {
			t.Fatal("unrepresentable time accepted")
		}
	}
	r := inputFixture()
	raw, key, _ := encodeIdentityInput(r)
	first, _ := decodeIdentityInput(key, raw, r.EpisodeID)
	second, _ := decodeIdentityInput(key, raw, r.EpisodeID)
	first.Declarations[0].Aliases[0] = "mutated"
	first.Facts[0].Supersedes.Src = "mutated"
	raw[0] = 0
	if !reflect.DeepEqual(second, r) {
		t.Fatal("decoded buffers shared")
	}
}
func newInputFixture() *identityInputRevision { r := inputFixture(); return &r }

func TestIndependentInputFullObservationFieldMatrix(t *testing.T) {
	r := inputFixture()
	raw, key, _ := encodeIdentityInput(r)
	mutations := []func(*identityInputRevision){
		func(v *identityInputRevision) { v.Declarations[0].TypeFallback = false },
		func(v *identityInputRevision) { v.Declarations[0].Aliases = []string{} },
		func(v *identityInputRevision) { v.Declarations[0].Aliases = nil },
		func(v *identityInputRevision) { v.Facts[0].Supersedes = nil },
		func(v *identityInputRevision) { v.Facts[0].Supersedes = &observedSupersedes{} },
		func(v *identityInputRevision) { v.Facts[0].Confidence = 0 },
		func(v *identityInputRevision) { v.OccurredAt = v.OccurredAt.Add(time.Nanosecond) },
	}
	for i := 0; i < 15; i++ {
		i := i
		mutations = append(mutations, func(v *identityInputRevision) { *independentInputStrings(v)[i] += "different" })
	}
	for i, mutate := range mutations {
		changed := inputFixture()
		mutate(&changed)
		for _, side := range []string{"declaration", "src", "dst"} {
			o := identityObservation{Version: 1, EpisodeID: changed.EpisodeID, OccurredAt: changed.OccurredAt, Cwd: changed.Cwd, Origin: "endpoint", Side: side, Fact: &changed.Facts[0]}
			if side == "declaration" {
				o.Origin = "declaration"
				o.Side = ""
				o.Fact = nil
				o.Declaration = &changed.Declarations[0]
			}
			ob, ok, err := encodeIdentityObservation(o)
			if err != nil {
				t.Fatal("bad synthetic observation")
			}
			original := o
			original.EpisodeID = r.EpisodeID
			original.OccurredAt = r.OccurredAt
			original.Cwd = r.Cwd
			if side == "declaration" {
				original.Declaration = &r.Declarations[0]
			} else {
				original.Fact = &r.Facts[0]
			}
			expected, _, _ := encodeIdentityObservation(original)
			match := matchRevisionObservation(key, raw, ok, ob, r.EpisodeID)
			if bytes.Equal(expected, ob) {
				if match != nil {
					t.Fatalf("unaffected observation %d/%s refused", i, side)
				}
			} else if match != errIdentityInput {
				t.Fatalf("mixed field %d/%s accepted", i, side)
			}
		}
	}
	o := identityObservation{Version: 1, EpisodeID: r.EpisodeID, OccurredAt: r.OccurredAt, Cwd: r.Cwd, Origin: "endpoint", Side: "src", Fact: &r.Facts[0]}
	for _, ordinal := range []int{-1, 1, math.MaxInt} {
		o.Ordinal = ordinal
		ob, _ := json.Marshal(o)
		ok := observationEpisodePrefix(r.EpisodeID) + "endpoint:" + fmt.Sprint(ordinal) + ":src:" + generationDigest(ob)
		if err := matchRevisionObservation(key, raw, ok, ob, r.EpisodeID); err != errIdentityInput {
			t.Fatal("bad ordinal accepted")
		}
	}
	o.Ordinal = 0
	ob, ok, _ := encodeIdentityObservation(o)
	for _, bad := range [][]byte{nil, append(bytes.Clone(ob), ' '), bytes.Replace(ob, []byte(`"src":"Aurora Guide"`), []byte(`"src":"Aurora Guide","src":"Aurora Guide"`), 1)} {
		if err := matchRevisionObservation(key, raw, ok, bad, r.EpisodeID); err != errIdentityInput {
			t.Fatal("malformed observation accepted")
		}
	}
	if matchRevisionObservation(key, raw, ok+"x", ob, r.EpisodeID) != errIdentityInput || matchRevisionObservation(key, append(bytes.Clone(raw), ' '), ok, ob, r.EpisodeID) != errIdentityInput {
		t.Fatal("bad exact address accepted")
	}
}

func TestIndependentInputStrictEveryMissingField(t *testing.T) {
	r := inputFixture()
	r.Declarations[0].Aliases = nil
	r.Declarations[0].TypeFallback = false
	r.Facts[0].Supersedes = nil
	raw, _, _ := encodeIdentityInput(r)
	// Preserve all other canonical bytes and use the exact hash of malformed bytes:
	// neither a stale key nor reordered JSON can accidentally explain refusal.
	fields := []struct {
		name  string
		value any
	}{
		{"version", 1}, {"episode_id", r.EpisodeID}, {"occurred_at", r.OccurredAt}, {"cwd", r.Cwd}, {"summary", r.Summary}, {"declarations", r.Declarations}, {"facts", r.Facts},
		{"name", r.Declarations[0].Name}, {"type", r.Declarations[0].Type}, {"description", r.Declarations[0].Description}, {"aliases", r.Declarations[0].Aliases}, {"type_fallback", false},
		{"src", r.Facts[0].Src}, {"relation", r.Facts[0].Relation}, {"dst", r.Facts[0].Dst}, {"fact", r.Facts[0].Fact}, {"valid_from", r.Facts[0].ValidFrom}, {"confidence", r.Facts[0].Confidence}, {"supersedes", r.Facts[0].Supersedes},
	}
	for _, field := range fields {
		name, _ := json.Marshal(field.name)
		value, _ := json.Marshal(field.value)
		needle := append(append(name, ':'), value...)
		start := bytes.Index(raw, needle)
		if start < 0 {
			t.Fatalf("missing synthetic field %s", field.name)
		}
		end := start + len(needle)
		if raw[end] == ',' {
			end++
		} else if raw[start-1] == ',' {
			start--
		} else {
			t.Fatal("field removal fixture")
		}
		bad := append(bytes.Clone(raw[:start]), raw[end:]...)
		if !json.Valid(bad) {
			t.Fatal("malformed fixture JSON")
		}
		badKey := inputEpisodePrefix(r.EpisodeID) + generationDigest(bad)
		if _, err := decodeIdentityInput(badKey, bad, r.EpisodeID); err != errIdentityInput {
			t.Fatalf("missing %s accepted", field.name)
		}
	}
	// Same-order duplicate fields, including identical zero values, remain noncanonical.
	for _, needle := range []string{`"version":1`, `"type_fallback":false`, `"supersedes":null`, `"aliases":null`} {
		bad := bytes.Replace(raw, []byte(needle), []byte(needle+","+needle), 1)
		if _, err := decodeIdentityInput(inputEpisodePrefix(r.EpisodeID)+generationDigest(bad), bad, r.EpisodeID); err != errIdentityInput {
			t.Fatal("duplicate accepted")
		}
	}
}

func TestIndependentInputProvenanceWriteReadAndPoison(t *testing.T) {
	r := inputFixture()
	raw, key, _ := encodeIdentityInput(r)
	good, _ := json.Marshal(Episode{ID: r.EpisodeID, OccurredAt: r.OccurredAt})
	cases := [][]byte{nil, []byte(`null`), []byte(`{}`), append(bytes.Clone(good), 'x'), bytes.Replace(good, []byte(`"id":`), []byte(`"irrelevant":`), 1), bytes.Replace(good, []byte(`"occurred_at":`), []byte(`"ignored":`), 1), bytes.Replace(good, []byte(r.EpisodeID), []byte("different"), 1), bytes.Replace(good, []byte(`"id":"synthetic-input"`), []byte(`"id":"synthetic-input","ID":"synthetic-input"`), 1), bytes.Replace(good, []byte(`"id":"synthetic-input"`), []byte(`"id":"synthetic-\u0069nput"`), 1), bytes.Replace(good, []byte("123456789Z"), []byte("123456788Z"), 1), append(bytes.Clone(good[:len(good)-1]), []byte(",\"extra\":\"\xff\"}")...)}
	for i, ep := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			st := openTemp(t)
			gradeGenerationSet(t, st, prefixEpisode+r.EpisodeID, ep)
			gradeGenerationSet(t, st, key, raw)
			before := gradeGenerationSnapshot(t, st)
			if p, e := listIdentityInputKeys(st, r.EpisodeID, "", 10, 24576); e != errIdentityInput || !reflect.DeepEqual(p, inputKeyPage{}) {
				t.Fatal("invalid EP read accepted")
			}
			if c, e := readIdentityInputChunk(st, r.EpisodeID, key, 0, 512); e != errIdentityInput || !reflect.DeepEqual(c, inputChunk{}) {
				t.Fatal("invalid EP chunk accepted")
			}
			finalized := false
			err := runIdentityAdmission(st, func(tx *Store) error {
				if e := tx.txn.Set([]byte("opaque:must-rollback"), []byte("synthetic")); e != nil {
					return e
				}
				_, e := putIdentityInput(tx, r)
				if e != errIdentityInput {
					t.Fatal("invalid EP replay accepted")
				}
				return nil
			}, func(*Store) error { finalized = true; return nil })
			if err != errIdentityInput || finalized || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("swallowed provenance failure committed")
			}
		})
	}
	// An EP marker staged in the same facade is valid, and rollback still removes both rows.
	st := openTemp(t)
	before := gradeGenerationSnapshot(t, st)
	marker := errors.New("synthetic rollback")
	err := st.AtomicWrite(func(tx *Store) error {
		if err := tx.txn.Set([]byte(prefixEpisode+r.EpisodeID), good); err != nil {
			return err
		}
		if _, err := putIdentityInput(tx, r); err != nil {
			return err
		}
		if _, err := readIdentityInputChunk(tx, r.EpisodeID, key, 0, 512); err != nil {
			return err
		}
		return marker
	})
	if err != marker || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("staged provenance or rollback failed")
	}
}

func TestIndependentInputCountBoundsCursorAndPinnedChunks(t *testing.T) {
	st := openTemp(t)
	r := inputFixture()
	inputEpisode(t, st, r)
	want := []string{}
	for i := 0; i < 107; i++ {
		r.Summary = fmt.Sprint(i)
		want = append(want, inputPut(t, st, r))
	}
	sort.Strings(want)
	p, err := listIdentityInputKeys(st, r.EpisodeID, "", 100, 24576)
	if err != nil || len(p.Keys) != 100 || !reflect.DeepEqual(p.Keys, want[:100]) || p.Next != want[99] {
		t.Fatal("count cap failed")
	}
	tail, err := listIdentityInputKeys(st, r.EpisodeID, p.Next, 100, 24576)
	if err != nil || !reflect.DeepEqual(tail.Keys, want[100:]) || tail.Next != "" {
		t.Fatal("tail page failed")
	}
	for _, limit := range []int{-1, 0, 101, math.MaxInt} {
		if _, e := listIdentityInputKeys(st, r.EpisodeID, "", limit, 512); e != errIdentityInput {
			t.Fatal("invalid limit accepted")
		}
	}
	for _, budget := range []int{-1, 0, 511, 24577, math.MaxInt} {
		if _, e := listIdentityInputKeys(st, r.EpisodeID, "", 1, budget); e != errIdentityInput {
			t.Fatal("invalid list budget")
		}
		if _, e := readIdentityInputChunk(st, r.EpisodeID, want[0], 0, budget); e != errIdentityInput {
			t.Fatal("invalid chunk budget")
		}
	}
	other := inputFixture()
	other.EpisodeID = "other-synthetic"
	inputEpisode(t, st, other)
	foreign := inputPut(t, st, other)
	for _, cursor := range []string{foreign, want[0] + "x", want[0][:len(want[0])-1], "\xff"} {
		if p, e := listIdentityInputKeys(st, r.EpisodeID, cursor, 1, 512); e != errIdentityInput || !reflect.DeepEqual(p, inputKeyPage{}) {
			t.Fatal("foreign/invalid cursor accepted")
		}
	}
	gradeGenerationSet(t, st, want[0], []byte("corrupt synthetic"))
	if p, e := listIdentityInputKeys(st, r.EpisodeID, want[0], 1, 512); e != errIdentityInput || !reflect.DeepEqual(p, inputKeyPage{}) {
		t.Fatal("corrupt cursor skipped")
	}
	r.Summary = strings.Repeat("😀世界", 5000)
	raw, key, _ := encodeIdentityInput(r)
	inputPut(t, st, r)
	assembled := []byte{}
	offset := 0
	pieces := 0
	for {
		c, e := readIdentityInputChunk(st, r.EpisodeID, key, offset, 512)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.Marshal(c)
		if len(b) > 512 || c.Key != key || c.Offset != offset {
			t.Fatal("bad pinned envelope")
		}
		assembled = append(assembled, c.Data...)
		pieces++
		if pieces == 1 {
			r.Summary = "changed input deleting declarations and facts"
			r.Declarations = nil
			r.Facts = nil
			inputPut(t, st, r)
		}
		if c.NextOffset == -1 {
			break
		}
		if c.NextOffset != offset+len(c.Data) || len(c.Data) == 0 {
			t.Fatal("bad progress")
		}
		offset = c.NextOffset
	}
	if pieces < 2 || !bytes.Equal(assembled, raw) {
		t.Fatal("changed revision spliced pinned record")
	}
	for _, offset := range []int{-1, len(raw) + 1, math.MaxInt} {
		if c, e := readIdentityInputChunk(st, r.EpisodeID, key, offset, 512); e != errIdentityInput || !reflect.DeepEqual(c, inputChunk{}) {
			t.Fatal("invalid offset accepted")
		}
	}
}

func TestIndependentInputOrdinaryPanicClosedAndOwnedStorageError(t *testing.T) {
	st := openTemp(t)
	r := inputFixture()
	inputEpisode(t, st, r)
	before := gradeGenerationSnapshot(t, st)
	var escaped *Store
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		_ = st.AtomicWrite(func(tx *Store) error {
			escaped = tx
			if _, e := putIdentityInput(tx, r); e != nil {
				t.Fatal(e)
			}
			panic("synthetic")
		})
	}()
	if !panicked || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("ordinary panic committed")
	}
	for _, s := range []*Store{nil, {}, escaped} {
		if _, e := readIdentityInputChunk(s, r.EpisodeID, "synthetic", 0, 512); e != errIdentityInput {
			t.Fatal("invalid facade reader")
		}
		if _, e := putIdentityInput(s, r); e != errIdentityInput {
			t.Fatal("invalid facade writer")
		}
	}
	// The established ordinary owner contract requires propagation, unlike admission owners.
	err := st.AtomicWrite(func(tx *Store) error {
		r.Version = 0
		_, e := putIdentityInput(tx, r)
		if e != errIdentityInput {
			t.Fatal("invalid input accepted")
		}
		return tx.txn.Set([]byte("opaque:ordinary-retained"), []byte("synthetic"))
	})
	if err != nil {
		t.Fatal("ordinary scope policy changed")
	}
	db, e := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithMemTableSize(2 << 20).WithValueThreshold(4096))
	if e != nil {
		t.Fatal(e)
	}
	bounded := &Store{db: db}
	defer bounded.Close()
	r = inputFixture()
	inputEpisode(t, bounded, r)
	snap := gradeGenerationSnapshot(t, bounded)
	err = runIdentityAdmission(bounded, func(tx *Store) error {
		if _, e := putIdentityInput(tx, r); e != nil {
			return e
		}
		r.Summary = strings.Repeat("private-synthetic-marker", 2000)
		_, e := putIdentityInput(tx, r)
		if e != errIdentityInput || strings.Contains(e.Error(), "private-synthetic-marker") {
			t.Fatal("actual storage error leaked original bytes")
		}
		return nil
	}, func(*Store) error { t.Fatal("poisoned storage failure finalized"); return nil })
	if err != errIdentityInput || !reflect.DeepEqual(snap, gradeGenerationSnapshot(t, bounded)) {
		t.Fatal("actual late error committed")
	}
	if e := st.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e := listIdentityInputKeys(st, r.EpisodeID, "", 1, 512); e != errIdentityInput {
		t.Fatal("closed root list accepted")
	}
	if _, e := readIdentityInputChunk(st, r.EpisodeID, "synthetic", 0, 512); e != errIdentityInput {
		t.Fatal("closed root chunk accepted")
	}
}
