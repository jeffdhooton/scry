package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func independentOutcomeInput(t *testing.T, st *Store, side string, revision int) (identityObservation, string) {
	t.Helper()
	o := identityObservation{Version: 1, EpisodeID: "independent-sensitive-episode", Origin: "endpoint", Ordinal: 17, Side: side, Cwd: "/synthetic", OccurredAt: time.Date(2026, 9, 6, 4, 5, 6, 123, time.UTC), Fact: &observedFact{Src: "Cedar Atlas", Relation: "uses", Dst: "Birch Journal", Fact: fmt.Sprintf("Exact synthetic assertion revision %d: 世界\n\u0000", revision), ValidFrom: "unparsed input", Confidence: .9, Supersedes: &observedSupersedes{Src: "Original / Source", Relation: "needed_by", Dst: "Old Destination 😀"}}}
	if err := st.PutEpisode(Episode{ID: o.EpisodeID, OccurredAt: o.OccurredAt}); err != nil {
		t.Fatal(err)
	}
	var key string
	if err := st.AtomicWrite(func(tx *Store) error { var err error; key, err = putIdentityObservation(tx, o); return err }); err != nil {
		t.Fatal(err)
	}
	return o, key
}

func independentOutcomeAssertion(o identityObservation, key string) identityOutcome {
	n := o.Ordinal
	return identityOutcome{Version: 1, EpisodeID: o.EpisodeID, AssertionOrdinal: &n, Disposition: "deferred", Links: []outcomeLink{{Key: key, Role: "primary-src", FinalSide: "none", Spelling: o.Fact.Src, State: "deferred"}, {Key: key, Role: "supersedes-src", FinalSide: "none", Spelling: o.Fact.Supersedes.Src, State: "deferred"}, {Key: key, Role: "supersedes-dst", FinalSide: "none", Spelling: o.Fact.Supersedes.Dst, State: "deferred"}}}
}

func independentOutcomePut(t *testing.T, st *Store, o identityOutcome) string {
	t.Helper()
	var key string
	if err := st.AtomicWrite(func(tx *Store) error { var err error; key, err = putIdentityOutcome(tx, o); return err }); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestIndependentOutcomeStagingErrorDoesNotDiscloseInput(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).WithValueThreshold(4096))
	if err != nil {
		t.Fatal(err)
	}
	st := &Store{db: db}
	defer st.Close()
	if err := st.ensureSchema(); err != nil {
		t.Fatal(err)
	}
	input, key := independentOutcomeInput(t, st, "src", 1)
	o := independentOutcomeAssertion(input, key)
	o.Birth = &identityBirth{EpisodeID: input.EpisodeID, Occurrence: input.Ordinal, Slug: "cedar-atlas", Name: input.Fact.Src, Origin: "endpoint", CreatedAt: input.OccurredAt}
	o.AssertionOrdinal = nil
	o.Disposition = "no-assertion"
	o.Materialization = []byte(`{"sensitive":"` + strings.Repeat("synthetic private material", 1000) + `"}`)
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	err = st.AtomicWrite(func(tx *Store) error { _, err := putIdentityOutcome(tx, o); return err })
	if err == nil {
		t.Fatal("fixture did not cause actual staging refusal")
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) || events != 0 {
		t.Fatal("staging refusal leaked writes/events")
	}
	if strings.Contains(err.Error(), "00000000") || strings.Contains(err.Error(), "sensitive") || strings.Contains(err.Error(), "Cedar") {
		t.Fatalf("SAFETY FAILURE: actual Badger staging error disclosed input instead of static reason/digest: %s", err)
	}
}

func TestIndependentOutcomeTwoBirthsOneInputAndRevisionLineage(t *testing.T) {
	st := openTemp(t)
	src, sk := independentOutcomeInput(t, st, "src", 1)
	dst, dk := independentOutcomeInput(t, st, "dst", 1)
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	for _, pair := range []struct {
		input           identityObservation
		key, role, name string
	}{{src, sk, "primary-src", src.Fact.Src}, {dst, dk, "primary-dst", dst.Fact.Dst}} {
		birth := identityBirth{EpisodeID: src.EpisodeID, Occurrence: src.Ordinal, Slug: Slugify(pair.name), Name: pair.name, Origin: "endpoint", CreatedAt: src.OccurredAt}
		record, _, err := identityBirthRecord(birth)
		if err != nil {
			t.Fatal(err)
		}
		o := identityOutcome{Version: 1, EpisodeID: src.EpisodeID, Birth: &birth, Disposition: "supported", Generation: record.ID, Materialization: []byte(" {\"unknown\": [null,{},\"😀\"], \"unknown\":2}\n"), Links: []outcomeLink{{Key: pair.key, Role: pair.role, FinalSide: pair.input.Side, Spelling: pair.name, ResolvedSlug: birth.Slug, State: "resolved"}}}
		independentOutcomePut(t, st, o)
	}
	a := independentOutcomeAssertion(src, sk)
	old := independentOutcomePut(t, st, a)
	if independentOutcomePut(t, st, a) != old {
		t.Fatal("replay drift")
	}
	revised, rk := independentOutcomeInput(t, st, "src", 2)
	b := independentOutcomeAssertion(revised, rk)
	b.Predecessor = old
	if err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityOutcome(tx, b); return err }); err == nil {
		t.Fatal("changed input inherited predecessor lineage")
	}
	b.Predecessor = ""
	newKey := independentOutcomePut(t, st, b)
	if newKey == old {
		t.Fatal("changed input overwrote revision")
	}
	// Same original input, explicit changed disposition and two retained branches.
	a.Predecessor = old
	a.Disposition = "committed"
	for i := range a.Links {
		a.Links[i].State = "not-materialized"
	}
	one := independentOutcomePut(t, st, a)
	a.Links[0].State, a.Links[0].FinalSide, a.Links[0].ResolvedSlug = "resolved", "src", "explicit-descriptive-owner"
	two := independentOutcomePut(t, st, a)
	if one == two {
		t.Fatal("branches collapsed")
	}
	if events != 1 {
		t.Fatalf("only explicit revised PutEpisode may emit event; got %d", events)
	}
	after := gradeGenerationSnapshot(t, st)
	for key, raw := range before {
		if !bytes.Equal(raw, after[key]) {
			t.Fatalf("preexisting row changed: %s", key)
		}
	}
	for key := range after {
		if _, existed := before[key]; !existed && !strings.HasPrefix(key, identityOutcomePrefix) && !strings.HasPrefix(key, identityObservationPrefix) {
			t.Fatalf("unexpected graph write %s", key)
		}
	}
	if entities, err := st.Entities(); err != nil || len(entities) != 0 {
		t.Fatal("descriptive support created entities")
	}
	if _, found, err := st.ResolveAlias("explicit-descriptive-owner"); err != nil || found {
		t.Fatal("annotation changed routing")
	}
}

func TestIndependentOutcomePaginationChunkAndReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "source")
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	input, ik := independentOutcomeInput(t, st, "src", 1)
	base := independentOutcomeAssertion(input, ik)
	keys := []string{}
	for i := 0; i < 27; i++ {
		o := base
		o.Birth = &identityBirth{EpisodeID: input.EpisodeID, Occurrence: input.Ordinal, Slug: "cedar-atlas", Name: input.Fact.Src, Origin: "endpoint", CreatedAt: input.OccurredAt}
		o.AssertionOrdinal = nil
		o.Disposition = "no-assertion"
		o.Materialization = []byte(fmt.Sprintf(" {\"n\": %d, \"opaque\": %q }\n", i, strings.Repeat("😀世界\n", 4500)))
		keys = append(keys, independentOutcomePut(t, st, o))
	}
	sort.Strings(keys)
	for _, budget := range []int{512, 777, 24576} {
		gotKeys := []string{}
		cursor := ""
		for tries := 0; tries < 100; tries++ {
			page, err := listIdentityOutcomeKeys(st, input.EpisodeID, cursor, 100, budget)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(page)
			if len(raw) > budget {
				t.Fatal("page exceeded byte cap")
			}
			gotKeys = append(gotKeys, page.Keys...)
			if page.Next == "" {
				break
			}
			if len(page.Keys) == 0 || page.Next != page.Keys[len(page.Keys)-1] || page.Next <= cursor {
				t.Fatal("cursor not exact/progressive")
			}
			cursor = page.Next
		}
		if !reflect.DeepEqual(keys, gotKeys) {
			t.Fatal("page skipped or duplicated keys")
		}
		var want []byte
		if err := st.view(func(tx *badger.Txn) error {
			item, err := tx.Get([]byte(keys[0]))
			if err != nil {
				return err
			}
			want, err = item.ValueCopy(nil)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if len(want) <= 24576 {
			t.Fatal("fixture not oversized")
		}
		var assembled []byte
		for offset := 0; ; {
			chunk, err := readIdentityOutcomeChunk(st, input.EpisodeID, keys[0], offset, budget)
			if err != nil {
				t.Fatal(err)
			}
			wire, _ := json.Marshal(chunk)
			if len(wire) > budget {
				t.Fatal("chunk envelope exceeded cap")
			}
			var decoded outcomeChunk
			if err := json.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			assembled = append(assembled, decoded.Data...)
			if decoded.NextOffset == -1 {
				break
			}
			if decoded.NextOffset != offset+len(decoded.Data) || decoded.NextOffset <= offset {
				t.Fatal("chunk not progressive")
			}
			offset = decoded.NextOffset
		}
		if !bytes.Equal(want, assembled) {
			t.Fatal("wire reassembly lost bytes")
		}
	}
	before := gradeGenerationSnapshot(t, st)
	var backup bytes.Buffer
	if n, err := st.Backup(&backup); err != nil || n == 0 || backup.Len() == 0 {
		t.Fatal("backup absent")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("reopen changed raw bytes")
	}
	replica := openTemp(t)
	if err := replica.Restore(bytes.NewReader(backup.Bytes())); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, replica)) {
		t.Fatal("restore changed raw bytes")
	}
}

func TestIndependentOutcomeStoredCorruptionAndExactProvenance(t *testing.T) {
	for _, mode := range []string{"missing-episode", "wrong-time", "duplicate-ID", "missing-observation", "observation-unknown", "key-collision", "outcome-unknown", "outcome-duplicate", "outcome-invalid-utf8", "outcome-surrogate", "wrong-role", "wrong-spelling", "wrong-ordinal", "predecessor-corrupt"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			input, ik := independentOutcomeInput(t, st, "src", 1)
			o := independentOutcomeAssertion(input, ik)
			key := independentOutcomePut(t, st, o)
			all := gradeGenerationSnapshot(t, st)
			raw := all[key]
			switch mode {
			case "missing-episode":
				if err := st.db.Update(func(tx *badger.Txn) error { return tx.Delete([]byte(prefixEpisode + input.EpisodeID)) }); err != nil {
					t.Fatal(err)
				}
			case "wrong-time":
				gradeGenerationSet(t, st, prefixEpisode+input.EpisodeID, []byte(fmt.Sprintf(`{"id":%q,"occurred_at":"2020-01-01T00:00:00Z"}`, input.EpisodeID)))
			case "duplicate-ID":
				gradeGenerationSet(t, st, prefixEpisode+input.EpisodeID, []byte(fmt.Sprintf(`{"id":%q,"ID":%q,"occurred_at":%q}`, input.EpisodeID, input.EpisodeID, input.OccurredAt.Format(time.RFC3339Nano))))
			case "missing-observation":
				if err := st.db.Update(func(tx *badger.Txn) error { return tx.Delete([]byte(ik)) }); err != nil {
					t.Fatal(err)
				}
			case "observation-unknown":
				gradeGenerationSet(t, st, ik, append(bytes.Clone(all[ik][:len(all[ik])-1]), []byte(`,"unknown":1}`)...))
			case "key-collision":
				gradeGenerationSet(t, st, key, []byte(`{"different":"synthetic input"}`))
			case "outcome-unknown":
				gradeGenerationSet(t, st, key, append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":1}`)...))
			case "outcome-duplicate":
				gradeGenerationSet(t, st, key, append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"version":1}`)...))
			case "outcome-invalid-utf8":
				gradeGenerationSet(t, st, key, bytes.Replace(raw, []byte("Cedar Atlas"), []byte("\xff"), 1))
			case "outcome-surrogate":
				gradeGenerationSet(t, st, key, bytes.Replace(raw, []byte("Cedar Atlas"), []byte(`\ud800`), 1))
			case "wrong-role", "wrong-spelling", "wrong-ordinal":
				if mode == "wrong-role" {
					o.Links[0].Role = "primary-dst"
				}
				if mode == "wrong-spelling" {
					o.Links[2].Spelling = "normalized-old-destination"
				}
				if mode == "wrong-ordinal" {
					*o.AssertionOrdinal++
				}
				var err error
				raw, key, err = encodeIdentityOutcome(o)
				if err != nil {
					t.Fatal(err)
				}
				gradeGenerationSet(t, st, key, raw)
			case "predecessor-corrupt":
				o.Predecessor = key
				key = independentOutcomePut(t, st, o)
				gradeGenerationSet(t, st, o.Predecessor, []byte(`{}`))
			}
			before := gradeGenerationSnapshot(t, st)
			chunk, err := readIdentityOutcomeChunk(st, input.EpisodeID, key, 0, 24576)
			if err == nil || !reflect.DeepEqual(chunk, outcomeChunk{}) {
				t.Fatal("reader accepted corrupt row or returned partial data")
			}
			page, err := listIdentityOutcomeKeys(st, input.EpisodeID, "", 100, 24576)
			if err == nil || !reflect.DeepEqual(page, outcomeKeyPage{}) {
				t.Fatal("listing accepted corrupt row or returned partial page")
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("reader mutated store")
			}
		})
	}
}

func TestIndependentOutcomeNilOwnershipAndRollback(t *testing.T) {
	st := openTemp(t)
	input, ik := independentOutcomeInput(t, st, "src", 1)
	o := independentOutcomeAssertion(input, ik)
	if _, err := putIdentityOutcome(st, o); err == nil {
		t.Fatal("root accepted write")
	}
	if _, err := putIdentityOutcome(nil, o); err == nil {
		t.Fatal("nil accepted write")
	}
	var expired *Store
	if err := st.AtomicWrite(func(tx *Store) error { expired = tx; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := putIdentityOutcome(expired, o); err == nil {
		t.Fatal("expired accepted write")
	}
	for _, mode := range []string{"return-error", "panic"} {
		before := gradeGenerationSnapshot(t, st)
		forced := errors.New("independent rollback")
		func() {
			defer func() {
				if p := recover(); p != nil && p != forced {
					panic(p)
				}
			}()
			err := st.AtomicWrite(func(tx *Store) error {
				if _, err := putIdentityOutcome(tx, o); err != nil {
					return err
				}
				if mode == "panic" {
					panic(forced)
				}
				return forced
			})
			if mode == "return-error" && !errors.Is(err, forced) {
				t.Fatal("callback error lost")
			}
		}()
		if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
			t.Fatal("rollback lost atomicity")
		}
	}
	want, key, err := encodeIdentityOutcome(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		if _, err := putIdentityOutcome(tx, o); err != nil {
			return err
		}
		*o.AssertionOrdinal = 999
		o.Links[0].Spelling = "caller changed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gradeGenerationSnapshot(t, st)[key], want) {
		t.Fatal("caller pointer/slice mutation altered staged bytes")
	}
	for _, material := range [][]byte{nil, {}, []byte(`null`), []byte(`[]`), []byte(`{"a":"\udc00"}`), []byte("{\"a\":\"\xff\"}")} {
		candidate := independentOutcomeAssertion(input, ik)
		candidate.Materialization = material
		_, _, err := encodeIdentityOutcome(candidate)
		if (material == nil) != (err == nil) {
			t.Fatal("assertion absence/empty materialization conflated")
		}
	}
}
