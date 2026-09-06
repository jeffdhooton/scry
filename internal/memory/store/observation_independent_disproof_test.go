package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func independentObservation(endpoint bool) identityObservation {
	o := identityObservation{Version: 1, EpisodeID: "episode:独立/🧭", Origin: "declaration", Ordinal: 12, Cwd: "/synthetic/é/e\u0301/\x00/\n", OccurredAt: time.Date(2026, 9, 6, 4, 5, 6, 987654321, time.UTC), Declaration: &observedDeclaration{Name: "Aurora Guide 🧭", Type: "concept", Description: "Exact \"quotes\", <>&, \u2028\u2029 and é/e\u0301.", Aliases: []string{" rejected alias ", "重复", "重复", "", "\x00"}, TypeFallback: true}}
	if endpoint {
		o.Origin, o.Side, o.Declaration = "endpoint", "src", nil
		o.Fact = &observedFact{Src: "DONE_WITH_CONCERNS", Relation: "used_by", Dst: "PYTHON_ARGCOMPLETE_OK", Fact: "原文, inverse/value endpoint before either flip.\n", ValidFrom: "not parsed 2026-02-31", Confidence: math.Nextafter(.9, 1), Supersedes: &observedSupersedes{Src: " prior source ", Relation: "prior inverse", Dst: "prior value"}}
	}
	return o
}

func independentPut(t *testing.T, st *Store, o identityObservation) string {
	t.Helper()
	var key string
	if err := st.AtomicWrite(func(tx *Store) error { var err error; key, err = putIdentityObservation(tx, o); return err }); err != nil {
		t.Fatal(err)
	}
	return key
}

func independentUnchanged(t *testing.T, st *Store, before map[string][]byte) {
	t.Helper()
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("raw store changed")
	}
}

func TestIndependentObservationFullOccurrenceMatrix(t *testing.T) {
	st := openTemp(t)
	var inputs []identityObservation
	for _, id := range []string{"episode:独立/🧭", "separate episode"} {
		for _, ordinal := range []int{0, 1, 2, 9, 10, 100} {
			for _, fallback := range []bool{false, true} {
				for _, aliases := range [][]string{nil, {}, {" rejected alias ", "重复", "重复", ""}} {
					o := independentObservation(false)
					o.EpisodeID, o.Ordinal, o.Declaration.TypeFallback, o.Declaration.Aliases = id, ordinal, fallback, aliases
					inputs = append(inputs, o)
				}
			}
			for _, side := range []string{"src", "dst"} {
				for _, supersedes := range []*observedSupersedes{nil, {}, {Src: "before", Relation: "used_by", Dst: "DONE"}} {
					o := independentObservation(true)
					o.EpisodeID, o.Ordinal, o.Side, o.Fact.Supersedes = id, ordinal, side, supersedes
					inputs = append(inputs, o)
				}
			}
		}
		o := independentObservation(false)
		o.EpisodeID = id
		observationEpisode(t, st, o)
	}
	before := gradeGenerationSnapshot(t, st)
	want := map[string]identityObservation{}
	events := 0
	st.SetObserver(func(Event) { events++ })
	for _, o := range inputs {
		key := independentPut(t, st, o)
		if _, exists := want[key]; exists {
			t.Fatal("distinct occurrence/revision collapsed")
		}
		want[key] = o
	}
	// Repeated description is a distinct revision at the exact same occurrence.
	o := inputs[0]
	d := *o.Declaration
	o.Declaration = &d
	o.Declaration.Description = "second conflicting description"
	want[independentPut(t, st, o)] = o
	after := gradeGenerationSnapshot(t, st)
	if len(after) != len(before)+len(want) {
		t.Fatal("unexpected keys")
	}
	for k, v := range before {
		if !bytes.Equal(after[k], v) {
			t.Fatal("preexisting bytes changed")
		}
	}
	for k := range after {
		if _, old := before[k]; !old && !strings.HasPrefix(k, "io:") {
			t.Fatal("other family written")
		}
	}
	for _, budget := range []int{1100, 1600, 10000} {
		for _, limit := range []int{1, 3, 100} {
			seen := map[string]bool{}
			for _, id := range []string{"episode:独立/🧭", "separate episode"} {
				cursor := ""
				var keys []string
				for n := 0; n <= len(want); n++ {
					page, err := readIdentityObservations(st, id, cursor, limit, budget)
					if err != nil {
						t.Fatal(err)
					}
					body, err := json.Marshal(page)
					if err != nil || len(body) > budget || len(page.Records) > limit {
						t.Fatal("encoded page/count over budget")
					}
					for _, got := range page.Records {
						_, key, err := encodeIdentityObservation(got)
						if err != nil || got.EpisodeID != id || seen[key] || !reflect.DeepEqual(got, want[key]) {
							t.Fatal("lost/changed/duplicated/wrong episode record")
						}
						seen[key] = true
						keys = append(keys, key)
					}
					if page.Next == "" {
						break
					}
					if len(page.Records) == 0 || page.Next <= cursor || page.Next != keys[len(keys)-1] {
						t.Fatal("invalid continuation")
					}
					cursor = page.Next
				}
				if !sort.StringsAreSorted(keys) {
					t.Fatal("not full-byte key order")
				}
			}
			if len(seen) != len(want) {
				t.Fatalf("missing records: %d/%d", len(seen), len(want))
			}
		}
	}
	for _, o := range want {
		independentPut(t, st, o)
	}
	independentUnchanged(t, st, after)
	if events != 0 {
		t.Fatal("observation emitted events")
	}
	if entities, err := st.Entities(); err != nil || len(entities) != 0 {
		t.Fatal("observation became entity")
	}
	if facts, err := st.AllFacts(); err != nil || len(facts) != 0 {
		t.Fatal("observation became fact")
	}
	if _, found, err := st.ResolveAlias(" rejected alias "); err != nil || found {
		t.Fatal("observation routed alias")
	}
}

func TestIndependentObservationEncodingRefusals(t *testing.T) {
	cases := map[string]func(*identityObservation){
		"version":             func(o *identityObservation) { o.Version = 2 },
		"negative ordinal":    func(o *identityObservation) { o.Ordinal = -1 },
		"empty episode":       func(o *identityObservation) { o.EpisodeID = "" },
		"invalid episode":     func(o *identityObservation) { o.EpisodeID = "\xff" },
		"invalid cwd":         func(o *identityObservation) { o.Cwd = "\xff" },
		"invalid name":        func(o *identityObservation) { o.Declaration.Name = "\xff" },
		"invalid type":        func(o *identityObservation) { o.Declaration.Type = "\xff" },
		"invalid description": func(o *identityObservation) { o.Declaration.Description = "\xff" },
		"invalid alias":       func(o *identityObservation) { o.Declaration.Aliases[2] = "\xff" },
		"zero time":           func(o *identityObservation) { o.OccurredAt = time.Time{} },
		"out of range time":   func(o *identityObservation) { o.OccurredAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
		"unknown origin":      func(o *identityObservation) { o.Origin = "other" },
		"declaration side":    func(o *identityObservation) { o.Side = "src" },
		"declaration missing": func(o *identityObservation) { o.Declaration = nil },
		"both kinds":          func(o *identityObservation) { o.Fact = &observedFact{} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			o := independentObservation(false)
			mutate(&o)
			if _, _, err := encodeIdentityObservation(o); !errors.Is(err, errIdentityObservation) {
				t.Fatal("invalid encoding accepted")
			}
		})
	}
	for _, field := range []string{"src", "relation", "dst", "fact", "valid_from", "sup-src", "sup-rel", "sup-dst", "nan", "inf", "side", "missing"} {
		t.Run("endpoint/"+field, func(t *testing.T) {
			o := independentObservation(true)
			switch field {
			case "src":
				o.Fact.Src = "\xff"
			case "relation":
				o.Fact.Relation = "\xff"
			case "dst":
				o.Fact.Dst = "\xff"
			case "fact":
				o.Fact.Fact = "\xff"
			case "valid_from":
				o.Fact.ValidFrom = "\xff"
			case "sup-src":
				o.Fact.Supersedes.Src = "\xff"
			case "sup-rel":
				o.Fact.Supersedes.Relation = "\xff"
			case "sup-dst":
				o.Fact.Supersedes.Dst = "\xff"
			case "nan":
				o.Fact.Confidence = math.NaN()
			case "inf":
				o.Fact.Confidence = math.Inf(1)
			case "side":
				o.Side = "other"
			case "missing":
				o.Fact = nil
			}
			if _, _, err := encodeIdentityObservation(o); !errors.Is(err, errIdentityObservation) {
				t.Fatal("invalid endpoint accepted")
			}
		})
	}
	o := independentObservation(false)
	raw, key, err := encodeIdentityObservation(o)
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{-3599, -1, 0, 1, 86399} {
		x := o
		x.OccurredAt = o.OccurredAt.In(time.FixedZone("synthetic seconds", offset))
		b, k, err := encodeIdentityObservation(x)
		if err != nil || k != key || !bytes.Equal(raw, b) {
			t.Fatal("equal instant changed UTC encoding")
		}
	}
	o.OccurredAt = time.Now()
	if _, _, err := encodeIdentityObservation(o); err != nil {
		t.Fatal("monotonic time failed UTC normalization")
	}
}

func TestIndependentObservationOccupiedBytesAndProvenance(t *testing.T) {
	for _, mode := range []string{"unknown", "duplicate", "noncanonical", "malformed", "different revision", "wrong address", "wrong episode body", "wrong episode time", "missing episode"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			o := independentObservation(false)
			observationEpisode(t, st, o)
			raw, key, err := encodeIdentityObservation(o)
			if err != nil {
				t.Fatal(err)
			}
			readKey := key
			switch mode {
			case "unknown":
				raw = append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"unknown\":{\"opaque\":true}}")...)
			case "duplicate":
				raw = append([]byte("{\"version\":2,"), raw[1:]...)
			case "noncanonical":
				raw = append([]byte(" "), raw...)
			case "malformed":
				raw = []byte("{broken")
			case "different revision":
				x := independentObservation(false)
				x.Declaration.Description = "collision occupant"
				raw, _, _ = encodeIdentityObservation(x)
			case "wrong address":
				readKey = key + "wrong"
			case "wrong episode body":
				ep, _ := st.GetEpisode(o.EpisodeID)
				ep.ID = "foreign"
				b, _ := json.Marshal(ep)
				gradeGenerationSet(t, st, prefixEpisode+o.EpisodeID, b)
			case "wrong episode time":
				ep, _ := st.GetEpisode(o.EpisodeID)
				ep.OccurredAt = ep.OccurredAt.Add(time.Nanosecond)
				b, _ := json.Marshal(ep)
				gradeGenerationSet(t, st, prefixEpisode+o.EpisodeID, b)
			case "missing episode":
				gradeGenerationSet(t, st, prefixEpisode+o.EpisodeID, nil)
			}
			gradeGenerationSet(t, st, readKey, raw)
			before := gradeGenerationSnapshot(t, st)
			if mode != "wrong address" {
				if err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, o); return err }); !errors.Is(err, errIdentityObservation) {
					t.Fatalf("occupied/provenance refusal failed: %v", err)
				}
			}
			if _, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000); !errors.Is(err, errIdentityObservation) {
				t.Fatalf("corrupt read accepted: %v", err)
			}
			independentUnchanged(t, st, before)
		})
	}
}

func TestIndependentObservationOwnsPointersAndReturnedBuffers(t *testing.T) {
	st := openTemp(t)
	o := independentObservation(true)
	observationEpisode(t, st, o)
	raw, key, _ := encodeIdentityObservation(o)
	if err := st.AtomicWrite(func(tx *Store) error {
		if _, err := putIdentityObservation(tx, o); err != nil {
			return err
		}
		o.Fact.Supersedes.Src, o.Fact.Supersedes.Relation, o.Fact.Supersedes.Dst = "mutated", "mutated", "mutated"
		o.Fact.Src, o.Fact.Fact, o.Fact.ValidFrom = "mutated", "mutated", "mutated"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gradeGenerationSnapshot(t, st)[key], raw) {
		t.Fatal("input pointer mutated stored bytes")
	}
	page, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000)
	if err != nil {
		t.Fatal(err)
	}
	page.Records[0].Fact.Supersedes.Dst = "returned pointer mutation"
	if !bytes.Equal(gradeGenerationSnapshot(t, st)[key], raw) {
		t.Fatal("returned pointer mutated stored bytes")
	}
	p2, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := encodeIdentityObservation(p2.Records[0])
	if !bytes.Equal(raw, b) {
		t.Fatal("read reused mutable pointer")
	}
}

func TestIndependentObservationRollbackPanicAndRealStagingFailure(t *testing.T) {
	for _, mode := range []string{"callback error", "panic", "staging failure", "same transaction episode", "staged episode rollback"} {
		t.Run(mode, func(t *testing.T) {
			db, err := badger.Open(badger.DefaultOptions(filepath.Join(t.TempDir(), "db")).WithLogger(nil).WithCompression(0).WithMemTableSize(2 << 20).WithValueThreshold(200000))
			if err != nil {
				t.Fatal(err)
			}
			st := &Store{db: db}
			defer st.Close()
			if err := st.ensureSchema(); err != nil {
				t.Fatal(err)
			}
			o := independentObservation(false)
			if mode != "same transaction episode" && mode != "staged episode rollback" {
				observationEpisode(t, st, o)
			}
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			var escaped *Store
			forced := errors.New("synthetic callback failure")
			panicked := false
			func() {
				defer func() {
					if p := recover(); p != nil {
						if p != forced {
							panic(p)
						}
						panicked = true
					}
				}()
				err = st.AtomicWrite(func(tx *Store) error {
					escaped = tx
					if mode == "same transaction episode" || mode == "staged episode rollback" {
						if err := tx.PutEpisode(Episode{ID: o.EpisodeID, OccurredAt: o.OccurredAt, Cwd: o.Cwd}); err != nil {
							return err
						}
					}
					if mode == "staging failure" {
						if err := tx.txn.Set([]byte("private-filler"), bytes.Repeat([]byte("f"), 190000)); err != nil {
							t.Fatalf("fixture failed too early: %v", err)
						}
						o.Declaration.Description = strings.Repeat("d", 190000)
					}
					if _, err := putIdentityObservation(tx, o); err != nil {
						return err
					}
					switch mode {
					case "panic":
						panic(forced)
					case "callback error", "staged episode rollback":
						return forced
					case "staging failure":
						t.Fatal("primitive Set unexpectedly succeeded")
					}
					return nil
				})
			}()
			if mode == "same transaction episode" {
				if err != nil || events != 1 {
					t.Fatalf("staged episode failed: err=%v events=%d", err, events)
				}
				page, e := readIdentityObservations(st, o.EpisodeID, "", 10, 10000)
				if e != nil || len(page.Records) != 1 {
					t.Fatal("committed observation missing")
				}
			} else {
				if mode == "panic" && !panicked {
					t.Fatal("panic swallowed")
				}
				if mode == "staging failure" && !errors.Is(err, badger.ErrTxnTooBig) {
					t.Fatalf("not actual primitive staging failure: %v", err)
				}
				if (mode == "callback error" || mode == "staged episode rollback") && !errors.Is(err, forced) {
					t.Fatal("callback error lost")
				}
				if events != 0 {
					t.Fatal("failed transaction emitted event")
				}
				independentUnchanged(t, st, before)
			}
			if _, e := putIdentityObservation(escaped, o); e == nil {
				t.Fatal("escaped facade accepted")
			}
			if _, e := putIdentityObservation(st, o); e == nil {
				t.Fatal("nontransactional write accepted")
			}
			if _, e := putIdentityObservation(nil, o); e == nil {
				t.Fatal("nil write accepted")
			}
		})
	}
}

func TestIndependentObservationOversizeCursorAndExactBound(t *testing.T) {
	st := openTemp(t)
	o := independentObservation(false)
	observationEpisode(t, st, o)
	key := independentPut(t, st, o)
	wire, _ := json.Marshal(observationPage{Records: []identityObservation{o}, Next: key})
	if _, err := readIdentityObservations(st, o.EpisodeID, "", 1, len(wire)); err != nil {
		t.Fatal("exact cursor-inclusive budget refused", err)
	}
	_, err := readIdentityObservations(st, o.EpisodeID, "", 1, len(wire)-1)
	if !errors.Is(err, errObservationPageTooSmall) || !strings.Contains(err.Error(), generationDigest([]byte(key))) {
		t.Fatal("oversize not visibly identified")
	}
	for _, cursor := range []string{key + "x", "io:foreign", observationEpisodePrefix(o.EpisodeID)} {
		if _, err := readIdentityObservations(st, o.EpisodeID, cursor, 1, 10000); !errors.Is(err, errIdentityObservation) {
			t.Fatal("invalid cursor accepted")
		}
	}
	for _, pair := range [][2]int{{0, 10000}, {101, 10000}, {1, 31}, {1, (1 << 20) + 1}} {
		if _, err := readIdentityObservations(st, o.EpisodeID, "", pair[0], pair[1]); !errors.Is(err, errIdentityObservation) {
			t.Fatal("invalid bounds accepted")
		}
	}
	// A small row before an oversized row is returned with its own cursor;
	// resumption must repeatedly identify, never advance past, the large row.
	o.Ordinal = 9
	o.Declaration.Description = strings.Repeat("large", 10000)
	largeKey := independentPut(t, st, o)
	page, err := readIdentityObservations(st, o.EpisodeID, "", 10, 1100)
	if err != nil || len(page.Records) != 1 || page.Next != key {
		t.Fatalf("small row not paged before oversize: %v", err)
	}
	for i := 0; i < 2; i++ {
		p, err := readIdentityObservations(st, o.EpisodeID, page.Next, 10, 1100)
		if !errors.Is(err, errObservationPageTooSmall) || p.Next != "" || len(p.Records) != 0 || !strings.Contains(err.Error(), generationDigest([]byte(largeKey))) {
			t.Fatal("oversize hidden or cursor advanced")
		}
	}
}

func TestIndependentObservationBackupLoadOpenReplayAllBytes(t *testing.T) {
	st := openTemp(t)
	o := independentObservation(true)
	observationEpisode(t, st, o)
	independentPut(t, st, o)
	// Unrecognized data and embedded opaque JSON survive along with observations.
	for _, family := range []string{"en:", "al:", "att:", "ig:", "iga:", "fa:", "adj:", "future-family:"} {
		gradeGenerationSet(t, st, family+"synthetic-opaque", []byte("opaque\x00\xff{\"extension\":true}"))
	}
	before := gradeGenerationSnapshot(t, st)
	var backup bytes.Buffer
	if n, err := st.Backup(&backup); err != nil || n == 0 || backup.Len() == 0 {
		t.Fatalf("backup failed: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "restored")
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Load(bytes.NewReader(backup.Bytes()), 16); err != nil {
		t.Fatal(err)
	}
	independentUnchanged(t, &Store{db: db}, before)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	independentUnchanged(t, restored, before)
	independentPut(t, restored, o)
	independentUnchanged(t, restored, before)
	page, err := readIdentityObservations(restored, o.EpisodeID, "", 10, 10000)
	if err != nil || len(page.Records) != 1 || !reflect.DeepEqual(page.Records[0], o) {
		t.Fatal("restore lost structured evidence")
	}
}

func TestIndependentObservationRejectsAmbiguousEpisodeProvenance(t *testing.T) {
	for _, field := range []string{"id", "occurred_at"} {
		t.Run(field, func(t *testing.T) {
			st := openTemp(t)
			o := independentObservation(false)
			observationEpisode(t, st, o)
			beforeEpisode := gradeGenerationSnapshot(t, st)[prefixEpisode+o.EpisodeID]
			conflict := "foreign-episode"
			if field == "occurred_at" {
				conflict = o.OccurredAt.Add(time.Hour).Format(time.RFC3339Nano)
			}
			// The earlier conflicting member must not be silently hidden by the
			// later expected member when establishing key/body provenance.
			raw := append([]byte(fmt.Sprintf("{%q:%q,", field, conflict)), beforeEpisode[1:]...)
			gradeGenerationSet(t, st, prefixEpisode+o.EpisodeID, raw)
			before := gradeGenerationSnapshot(t, st)
			err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, o); return err })
			if !errors.Is(err, errIdentityObservation) {
				t.Errorf("ambiguous episode %s accepted as authoritative provenance: %v", field, err)
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Error("ambiguous episode allowed committed observation")
			}
			// Seed exact canonical evidence so read refusal is checked even if a
			// later candidate fixes the writer independently of the reader.
			observationRaw, observationKey, encodeErr := encodeIdentityObservation(o)
			if encodeErr != nil {
				t.Fatal(encodeErr)
			}
			gradeGenerationSet(t, st, observationKey, observationRaw)
			beforeRead := gradeGenerationSnapshot(t, st)
			page, readErr := readIdentityObservations(st, o.EpisodeID, "", 10, 10000)
			if !errors.Is(readErr, errIdentityObservation) {
				t.Errorf("ambiguous episode %s accepted on read: records=%d err=%v", field, len(page.Records), readErr)
			}
			independentUnchanged(t, st, beforeRead)
		})
	}
}
