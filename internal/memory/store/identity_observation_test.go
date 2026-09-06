package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func observationFixture() identityObservation {
	return identityObservation{Version: 1, EpisodeID: "observation-episode", Origin: "declaration", Ordinal: 0, OccurredAt: time.Date(2026, 9, 6, 1, 2, 3, 123456789, time.UTC), Cwd: "/synthetic/workspace", Declaration: &observedDeclaration{Name: "Aurora Guide", Type: "concept", Description: "First exact description.", Aliases: []string{"Polar Manual", "Polar Manual", "discarded proposal"}, TypeFallback: true}}
}

func observationEpisode(t *testing.T, st *Store, o identityObservation) {
	t.Helper()
	if err := st.PutEpisode(Episode{ID: o.EpisodeID, OccurredAt: o.OccurredAt.UTC(), Cwd: o.Cwd}); err != nil {
		t.Fatal(err)
	}
}

func TestObservationOccurrenceRevisionsAndBoundedPages(t *testing.T) {
	st := openTemp(t)
	o := observationFixture()
	observationEpisode(t, st, o)
	before := gradeGenerationSnapshot(t, st)
	want := map[string][]byte{}
	events := 0
	st.SetObserver(func(Event) { events++ })
	for i := 0; i < 7; i++ {
		o.Ordinal = i / 2 // two explicit content revisions of an occurrence
		o.Declaration.TypeFallback = i%2 == 0
		if err := st.AtomicWrite(func(tx *Store) error {
			key, err := putIdentityObservation(tx, o)
			if err != nil {
				return err
			}
			raw, _, err := encodeIdentityObservation(o)
			if err != nil {
				return err
			}
			want[key] = raw
			second, err := putIdentityObservation(tx, o)
			if second != key {
				t.Fatal("identical replay changed address")
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	cursor := ""
	for pageNo := 0; pageNo < 10; pageNo++ {
		page, err := readIdentityObservations(st, o.EpisodeID, cursor, 2, 1800)
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(page)
		if len(encoded) > 1800 || len(page.Records) > 2 {
			t.Fatal("page exceeded bound")
		}
		for _, got := range page.Records {
			raw, key, err := encodeIdentityObservation(got)
			if err != nil || seen[key] || !bytes.Equal(raw, want[key]) {
				t.Fatal("page lost, changed or duplicated occurrence")
			}
			seen[key] = true
		}
		if page.Next == "" {
			break
		}
		if page.Next == cursor {
			t.Fatal("cursor failed to advance")
		}
		cursor = page.Next
	}
	if len(seen) != len(want) || events != 0 {
		t.Fatal("missing observations or graph events")
	}
	after := gradeGenerationSnapshot(t, st)
	for key, value := range before {
		if !bytes.Equal(value, after[key]) {
			t.Fatal("old record changed")
		}
	}
	if len(after) != len(before)+len(want) {
		t.Fatal("unexpected key family writes")
	}
	if entities, err := st.Entities(); err != nil || len(entities) != 0 {
		t.Fatal("observations became graph nodes")
	}
	if _, found, err := st.ResolveAlias("Polar Manual"); err != nil || found {
		t.Fatal("observation acquired routing")
	}
}

func TestObservationEncodingAndOriginalFactPreservation(t *testing.T) {
	o := observationFixture()
	raw, _, err := encodeIdentityObservation(o)
	if err != nil {
		t.Fatal(err)
	}
	var decoded identityObservation
	if err := json.Unmarshal(raw, &decoded); err != nil || !reflect.DeepEqual(o, decoded) {
		t.Fatal("parsed declaration lost")
	}
	o.Declaration.Aliases = nil
	nilRaw, _, _ := encodeIdentityObservation(o)
	o.Declaration.Aliases = []string{}
	emptyRaw, _, _ := encodeIdentityObservation(o)
	if bytes.Equal(nilRaw, emptyRaw) {
		t.Fatal("nil/empty aliases collapsed")
	}
	o.Declaration.Name = "invalid-\xff"
	if _, _, err := encodeIdentityObservation(o); !errors.Is(err, errIdentityObservation) {
		t.Fatal("lossy text accepted")
	}
	o = observationFixture()
	o.Origin = "endpoint"
	o.Side = "dst"
	o.Declaration = nil
	o.Fact = &observedFact{Src: "in progress", Relation: "used_by", Dst: "Aurora", Fact: "Original inverse assertion, before any flips.", ValidFrom: "unparsed date retained", Confidence: .875, Supersedes: &observedSupersedes{Src: "before", Relation: "uses", Dst: "old"}}
	st := openTemp(t)
	observationEpisode(t, st, o)
	if err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, o); return err }); err != nil {
		t.Fatal(err)
	}
	page, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000)
	if err != nil || len(page.Records) != 1 || !reflect.DeepEqual(page.Records[0], o) {
		t.Fatal("original fact or supersession lost")
	}
	one := o
	one.OccurredAt = o.OccurredAt.In(time.FixedZone("seconds", 1))
	a, keyA, errA := encodeIdentityObservation(o)
	b, keyB, errB := encodeIdentityObservation(one)
	if errA != nil || errB != nil || keyA != keyB || !bytes.Equal(a, b) {
		t.Fatal("equal instant encoding differs")
	}
	one.OccurredAt = one.OccurredAt.Add(time.Nanosecond)
	_, keyC, err := encodeIdentityObservation(one)
	if err != nil || keyC == keyA {
		t.Fatal("nanosecond lost")
	}
}

func TestObservationProvenanceCorruptionAndPageRefusal(t *testing.T) {
	st := openTemp(t)
	o := observationFixture()
	if err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, o); return err }); !errors.Is(err, errIdentityObservation) {
		t.Fatal("missing episode accepted")
	}
	observationEpisode(t, st, o)
	wrong := o
	wrong.OccurredAt = wrong.OccurredAt.Add(time.Second)
	if err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, wrong); return err }); !errors.Is(err, errIdentityObservation) {
		t.Fatal("wrong episode time accepted")
	}
	var key string
	if err := st.AtomicWrite(func(tx *Store) error { var err error; key, err = putIdentityObservation(tx, o); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := readIdentityObservations(st, o.EpisodeID, "", 1, 32); !errors.Is(err, errObservationPageTooSmall) {
		t.Fatal("oversized observation skipped")
	}
	if _, err := readIdentityObservations(st, "other-episode", key, 1, 10000); !errors.Is(err, errIdentityObservation) {
		t.Fatal("foreign cursor accepted")
	}
	raw := gradeGenerationSnapshot(t, st)[key]
	unknown := append(bytes.Clone(raw[:len(raw)-1]), []byte(",\"unknown\":true}")...)
	gradeGenerationSet(t, st, key, unknown)
	before := gradeGenerationSnapshot(t, st)
	if err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, o); return err }); !errors.Is(err, errIdentityObservation) {
		t.Fatal("occupied unknown payload overwritten")
	}
	if _, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000); !errors.Is(err, errIdentityObservation) {
		t.Fatal("unknown payload silently reserialized")
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("refusal changed bytes")
	}
}

func TestObservationOwnedBuffersRollbackAndRestore(t *testing.T) {
	st := openTemp(t)
	o := observationFixture()
	observationEpisode(t, st, o)
	before := gradeGenerationSnapshot(t, st)
	forced := errors.New("synthetic failure")
	events := 0
	st.SetObserver(func(Event) { events++ })
	var escaped *Store
	if err := st.AtomicWrite(func(tx *Store) error {
		escaped = tx
		if _, err := putIdentityObservation(tx, o); err != nil {
			return err
		}
		return forced
	}); !errors.Is(err, forced) {
		t.Fatal("rollback failure")
	}
	if events != 0 || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
		t.Fatal("failed write escaped")
	}
	if _, err := putIdentityObservation(escaped, o); err == nil {
		t.Fatal("expired facade accepted")
	}
	raw, key, err := encodeIdentityObservation(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AtomicWrite(func(tx *Store) error {
		if _, err := putIdentityObservation(tx, o); err != nil {
			return err
		}
		o.Declaration.Aliases[0] = "caller changed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after := gradeGenerationSnapshot(t, st)
	if !bytes.Equal(after[key], raw) {
		t.Fatal("caller mutation changed staged bytes")
	}
	var backup bytes.Buffer
	if n, err := st.Backup(&backup); err != nil || n == 0 {
		t.Fatal("backup failed")
	}
	dir := filepath.Join(t.TempDir(), "restored")
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil).WithCompression(0))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Load(bytes.NewReader(backup.Bytes()), 16); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if !reflect.DeepEqual(after, gradeGenerationSnapshot(t, r)) {
		t.Fatal("restored bytes differ")
	}
	page, err := readIdentityObservations(r, o.EpisodeID, "", 10, 10000)
	if err != nil || len(page.Records) != 1 {
		t.Fatal("restored evidence unreadable")
	}
	got, _, _ := encodeIdentityObservation(page.Records[0])
	if !bytes.Equal(raw, got) {
		t.Fatal("restore changed original input")
	}
}
