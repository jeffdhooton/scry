package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func outcomeFixture(t *testing.T, st *Store, assertion bool) identityOutcome {
	t.Helper()
	input := observationFixture()
	if assertion {
		input.Origin = "endpoint"
		input.Side = "src"
		input.Declaration = nil
		input.Fact = &observedFact{Src: "Aurora Guide", Relation: "uses", Dst: "Polar Manual", Fact: "Synthetic assertion.", ValidFrom: "original unparsed time", Confidence: .8, Supersedes: &observedSupersedes{Src: "Aurora Guide", Relation: "uses", Dst: "Former Manual"}}
	}
	observationEpisode(t, st, input)
	var key string
	if err := st.AtomicWrite(func(tx *Store) error { var err error; key, err = putIdentityObservation(tx, input); return err }); err != nil {
		t.Fatal(err)
	}
	o := identityOutcome{Version: 1, EpisodeID: input.EpisodeID, Disposition: "no-assertion", Birth: &identityBirth{EpisodeID: input.EpisodeID, Occurrence: input.Ordinal, Slug: "aurora-guide", Name: "Aurora Guide", Origin: "declaration", CreatedAt: input.OccurredAt}, Links: []outcomeLink{{Key: key, Role: "declaration", FinalSide: "none", Spelling: "Aurora Guide", State: "not-materialized"}}}
	if assertion {
		ordinal := input.Ordinal
		o.Birth = nil
		o.AssertionOrdinal = &ordinal
		o.Disposition = "deferred"
		o.Links = []outcomeLink{{Key: key, Role: "primary-src", Spelling: "Aurora Guide", FinalSide: "src", ResolvedSlug: "aurora-guide", State: "resolved"}, {Key: key, Role: "supersedes-dst", Spelling: "Former Manual", FinalSide: "none", State: "deferred"}}
	}
	return o
}

func outcomePut(t *testing.T, st *Store, o identityOutcome) string {
	t.Helper()
	var key string
	if err := st.AtomicWrite(func(tx *Store) error { var err error; key, err = putIdentityOutcome(tx, o); return err }); err != nil {
		t.Fatal(err)
	}
	return key
}

func outcomeAssemble(t *testing.T, st *Store, o identityOutcome, key string, budget int) []byte {
	t.Helper()
	all := []byte{}
	offset := 0
	for attempts := 0; attempts < 10000; attempts++ {
		chunk, err := readIdentityOutcomeChunk(st, o.EpisodeID, key, offset, budget)
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(chunk)
		if len(encoded) > budget || chunk.Offset != offset || chunk.Key != key {
			t.Fatal("chunk envelope drift")
		}
		all = append(all, chunk.Data...)
		if chunk.NextOffset == -1 {
			return all
		}
		if chunk.NextOffset != offset+len(chunk.Data) || chunk.NextOffset <= offset {
			t.Fatal("chunk failed to advance")
		}
		offset = chunk.NextOffset
	}
	t.Fatal("chunk reassembly did not terminate")
	return nil
}

func TestOutcomeHintOnlyHasNoBirthOrRouting(t *testing.T) {
	st := openTemp(t)
	o := outcomeFixture(t, st, true)
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	key := outcomePut(t, st, o)
	if replay := outcomePut(t, st, o); replay != key {
		t.Fatal("identical input changed key")
	}
	raw := outcomeAssemble(t, st, o, key, 512)
	got, err := decodeIdentityOutcome(key, raw, o.EpisodeID)
	if err != nil || !reflect.DeepEqual(got, o) || got.Birth != nil || got.Generation != "" {
		t.Fatal("hint-only subject changed")
	}
	after := gradeGenerationSnapshot(t, st)
	if len(after) != len(before)+1 || events != 0 {
		t.Fatal("unexpected write/event")
	}
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("old bytes changed")
		}
	}
	if entities, err := st.Entities(); err != nil || len(entities) != 0 {
		t.Fatal("outcome created identity")
	}
	if _, found, err := st.ResolveAlias("Former Manual"); err != nil || found {
		t.Fatal("outcome acquired routing")
	}
}

func TestOutcomeRevisionHistoryAndOversizeInspection(t *testing.T) {
	st := openTemp(t)
	o := outcomeFixture(t, st, false)
	o.Materialization = []byte(" { \"opaque\":\"" + strings.Repeat("世界😀", 7000) + "\", \"opaque\":2 } ")
	initial := outcomePut(t, st, o)
	want, _, err := encodeIdentityOutcome(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(want) < 24576 {
		t.Fatal("fixture not oversized")
	}
	if got := outcomeAssemble(t, st, o, initial, 512); !bytes.Equal(got, want) {
		t.Fatal("oversized record truncated")
	}
	o.Predecessor = initial
	o.Disposition = "supported"
	record, _, err := identityBirthRecord(*o.Birth)
	if err != nil {
		t.Fatal(err)
	}
	o.Generation = record.ID
	o.Links[0].FinalSide = "src"
	o.Links[0].ResolvedSlug = o.Birth.Slug
	o.Links[0].State = "resolved"
	successor := outcomePut(t, st, o)
	if successor == initial {
		t.Fatal("outcome overwrote history")
	}
	if repeat := outcomePut(t, st, o); repeat != successor {
		t.Fatal("identical successor duplicated")
	}
	// A second branch is retained, not silently chosen as current by the codec.
	o.Materialization = []byte(`{"another":"exact materialization"}`)
	branch := outcomePut(t, st, o)
	seen := map[string]bool{}
	cursor := ""
	for attempt := 0; attempt < 10; attempt++ {
		page, err := listIdentityOutcomeKeys(st, o.EpisodeID, cursor, 1, 512)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(page)
		if len(raw) > 512 || len(page.Keys) > 1 {
			t.Fatal("page exceeds bound")
		}
		for _, key := range page.Keys {
			if seen[key] {
				t.Fatal("duplicate page")
			}
			seen[key] = true
		}
		if page.Next == "" {
			break
		}
		if page.Next == cursor {
			t.Fatal("cursor stalled")
		}
		cursor = page.Next
	}
	if len(seen) != 3 || !seen[initial] || !seen[successor] || !seen[branch] {
		t.Fatal("history omitted")
	}
}

func TestOutcomeLinkAndSubjectRefusals(t *testing.T) {
	for _, mode := range []string{"missing-link", "wrong-role", "wrong-ordinal", "wrong-spelling", "wrong-episode", "duplicate-link", "wrong-birth-time", "missing-birth-link", "bad-state", "deferred-with-owner", "unknown-outcome", "duplicate-outcome", "corrupt-observation", "corrupt-episode", "wrong-predecessor"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			o := outcomeFixture(t, st, mode != "wrong-birth-time" && mode != "missing-birth-link")
			switch mode {
			case "missing-link":
				o.Links[0].Key = "io:absent"
			case "wrong-role":
				o.Links[0].Role = "primary-dst"
			case "wrong-ordinal":
				*o.AssertionOrdinal++
			case "wrong-spelling":
				o.Links[1].Spelling = "Different"
			case "wrong-episode":
				o.EpisodeID = "wrong-episode"
			case "duplicate-link":
				o.Links = append(o.Links, o.Links[0])
			case "wrong-birth-time":
				o.Birth.CreatedAt = o.Birth.CreatedAt.Add(time.Nanosecond)
			case "missing-birth-link":
				o.Birth.Occurrence++
			case "bad-state":
				o.Links[0].State = "owner-inferred"
			case "deferred-with-owner":
				o.Links[1].ResolvedSlug = "guessed-owner"
			case "corrupt-observation":
				gradeGenerationSet(t, st, o.Links[0].Key, []byte(`{"opaque":true}`))
			case "corrupt-episode":
				gradeGenerationSet(t, st, prefixEpisode+o.EpisodeID, []byte(`{"id":"wrong"}`))
			case "wrong-predecessor":
				previous := outcomeFixture(t, st, false)
				o.Predecessor = outcomePut(t, st, previous)
			}
			before := gradeGenerationSnapshot(t, st)
			if mode == "unknown-outcome" || mode == "duplicate-outcome" {
				raw, key, err := encodeIdentityOutcome(o)
				if err != nil {
					t.Fatal(err)
				}
				extra := `,"unknown":true}`
				if mode == "duplicate-outcome" {
					extra = `,"version":1}`
				}
				raw = append(bytes.Clone(raw[:len(raw)-1]), extra...)
				if _, err := decodeIdentityOutcome(key, raw, o.EpisodeID); err == nil {
					t.Fatal("noncanonical outcome accepted")
				}
			} else {
				err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityOutcome(tx, o); return err })
				if err == nil {
					t.Fatal("invalid outcome committed")
				}
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("refusal changed bytes")
			}
		})
	}
}

func TestOutcomeOwnsBuffersAndBackupBytes(t *testing.T) {
	st := openTemp(t)
	o := outcomeFixture(t, st, false)
	o.Materialization = []byte(`{"opaque":"exact"}`)
	want, key, err := encodeIdentityOutcome(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		got, err := putIdentityOutcome(tx, o)
		if err != nil {
			return err
		}
		if got != key {
			t.Fatal("key drift")
		}
		o.Materialization[2] = 'X'
		o.Links[0].Spelling = "mutated"
		o.Birth.Name = "mutated"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	raw := outcomeAssemble(t, st, o, key, 1024)
	if !bytes.Equal(raw, want) {
		t.Fatal("caller mutation changed stored bytes")
	}
	before := gradeGenerationSnapshot(t, st)
	var backup bytes.Buffer
	if n, err := st.Backup(&backup); err != nil || n == 0 {
		t.Fatal("empty backup")
	}
	restored := openTemp(t)
	if err := restored.Restore(bytes.NewReader(backup.Bytes())); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, restored)) {
		t.Fatal("backup lost opaque bytes")
	}
}

func TestOutcomeRollbackAndActualStagingFailure(t *testing.T) {
	for _, mode := range []string{"callback-error", "panic", "staging-failure"} {
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
			o := outcomeFixture(t, st, false)
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			forced := errors.New("synthetic rollback")
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
					if mode == "staging-failure" {
						if err := tx.txn.Set([]byte("synthetic-filler"), bytes.Repeat([]byte("f"), 190000)); err != nil {
							t.Fatal("fixture failed early")
						}
						// Base64 expansion must stay below the value-log threshold;
						// otherwise Badger accounts a small pointer, not this value.
						o.Materialization = []byte(`{"opaque":"` + strings.Repeat("x", 120000) + `"}`)
					}
					if _, err := putIdentityOutcome(tx, o); err != nil {
						return err
					}
					if mode == "staging-failure" {
						t.Fatal("oversized outcome unexpectedly staged")
					}
					if mode == "panic" {
						panic(forced)
					}
					return forced
				})
			}()
			if mode == "staging-failure" && !errors.Is(err, badger.ErrTxnTooBig) {
				t.Fatal("not real staging failure")
			}
			if mode == "panic" && !panicked {
				t.Fatal("panic swallowed")
			}
			if mode == "callback-error" && !errors.Is(err, forced) {
				t.Fatal("error swallowed")
			}
			if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("rollback leaked bytes/events")
			}
		})
	}
}

func TestOutcomeInvalidEncodingAndReaderBounds(t *testing.T) {
	st := openTemp(t)
	o := outcomeFixture(t, st, false)
	key := outcomePut(t, st, o)
	for i, material := range [][]byte{[]byte{}, []byte("null"), []byte(`{"invalid":"\ud800"}`), []byte("{\"invalid\":\"\xff\"}")} {
		candidate := o
		candidate.Materialization = material
		if _, _, err := encodeIdentityOutcome(candidate); err == nil {
			t.Fatal(fmt.Sprint("invalid materialization accepted ", i))
		}
	}
	candidate := o
	candidate.Links = append([]outcomeLink{}, o.Links...)
	candidate.Links[0].Spelling = "bad-\xff"
	if _, _, err := encodeIdentityOutcome(candidate); err == nil {
		t.Fatal("lossy input accepted")
	}
	for _, budget := range []int{0, 511, 24577} {
		if _, err := listIdentityOutcomeKeys(st, o.EpisodeID, "", 1, budget); err == nil {
			t.Fatal("invalid page budget accepted")
		}
		if _, err := readIdentityOutcomeChunk(st, o.EpisodeID, key, 0, budget); err == nil {
			t.Fatal("invalid chunk budget accepted")
		}
	}
	for _, offset := range []int{-1, 1000000} {
		if _, err := readIdentityOutcomeChunk(st, o.EpisodeID, key, offset, 512); err == nil {
			t.Fatal("invalid offset accepted")
		}
	}
	if _, err := listIdentityOutcomeKeys(st, o.EpisodeID, "absent", 1, 512); err == nil {
		t.Fatal("absent cursor accepted")
	}
	if _, err := readIdentityOutcomeChunk(st, "different", key, 0, 512); err == nil {
		t.Fatal("wrong episode accepted")
	}
	var expired *Store
	if err := st.AtomicWrite(func(tx *Store) error { expired = tx; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := listIdentityOutcomeKeys(expired, o.EpisodeID, "", 1, 512); err == nil {
		t.Fatal("expired view accepted")
	}
}
