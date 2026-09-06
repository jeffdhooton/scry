package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestIndependentCorrectedProvenanceParserBoundary(t *testing.T) {
	o := independentObservation(false)
	id, _ := json.Marshal(o.EpisodeID)
	at, _ := json.Marshal(o.OccurredAt)
	identity := `"id":` + string(id)
	instant := `"occurred_at":` + string(at)
	base := `{` + identity + `,` + instant + `}`
	offset, _ := json.Marshal(o.OccurredAt.In(time.FixedZone("synthetic offset", 5*3600+45*60)))
	type parserCase struct {
		name, raw string
		accept    bool
	}
	cases := []parserCase{
		{"minimal", base, true},
		{"whitespace and order", " \n\t{\n" + instant + " , \n" + identity + "\n}\t\r\n", true},
		{"mixed case single known fields", `{"Id":` + string(id) + `,"Occurred_At":` + string(at) + `}`, true},
		{"escaped known field names", `{"\u0069\u0064":` + string(id) + `,"occurred\u005fat":` + string(at) + `}`, true},
		{"offset exact instant", `{` + identity + `,"occurred_at":` + string(offset) + `}`, true},
		{"unknown duplicate fields", `{` + identity + `,"future":1,"future":2,` + instant + `}`, true},
		{"nested conflicting identity is opaque", `{` + identity + `,"future":{"id":"wrong","ID":null,"occurred_at":"wrong","id":"other"},` + instant + `}`, true},
		{"unknown huge number remains raw", `{` + identity + `,"future":1e99999,` + instant + `}`, true},
		{"unknown string null array", `{` + identity + `,"f1":"🧭\u0000","f2":null,"f3":[true,false,{}],` + instant + `}`, true},
		{"array root", `[` + base + `]`, false},
		{"null root", `null`, false},
		{"string root", `"{}"`, false},
		{"empty object", `{}`, false},
		{"missing close", strings.TrimSuffix(base, "}"), false},
		{"wrong close", strings.TrimSuffix(base, "}") + "]", false},
		{"trailing comma", strings.TrimSuffix(base, "}") + ",}", false},
		{"trailing null", base + " null", false},
		{"trailing garbage", base + " trailing", false},
		{"trailing malformed object", base + " {", false},
		{"trailing unicode whitespace", base + "\u00a0", false},
		{"BOM", "\ufeff" + base, false},
		{"invalid UTF8 unrelated field name", `{` + identity + `,"\xff":1,` + instant + `}`, false},
		{"malformed unknown nested", `{` + identity + `,"future":{"nested":[1,},` + instant + `}`, false},
		{"same id repeated", `{` + identity + `,` + identity + `,` + instant + `}`, false},
		{"same time repeated", `{` + identity + `,` + instant + `,` + instant + `}`, false},
		{"escaped duplicate id", `{` + identity + `,"\u0069d":` + string(id) + `,` + instant + `}`, false},
		{"escaped uppercase duplicate time", `{` + identity + `,` + instant + `,"\u004fccurred_At":` + string(at) + `}`, false},
		{"wrong known member after right", `{` + identity + `,"ID":"foreign",` + instant + `}`, false},
		{"id object", `{"id":{},` + instant + `}`, false},
		{"id array", `{"id":[],` + instant + `}`, false},
		{"id boolean", `{"id":true,` + instant + `}`, false},
		{"time null", `{` + identity + `,"occurred_at":null}`, false},
		{"time object", `{` + identity + `,"occurred_at":{}}`, false},
		{"time array", `{` + identity + `,"occurred_at":[]}`, false},
		{"time bool", `{` + identity + `,"occurred_at":false}`, false},
		{"time invalid date", `{` + identity + `,"occurred_at":"2026-02-31T04:05:06.987654321Z"}`, false},
		{"time wrong nanosecond", `{` + identity + `,"occurred_at":"2026-09-06T04:05:06.987654322Z"}`, false},
		{"time redundant precision", `{` + identity + `,"occurred_at":"2026-09-06T04:05:06.9876543210Z"}`, false},
		{"time noncanonical escape", `{` + identity + `,"occurred_at":"2026-09-06T04:05:06.987654321\u005a"}`, false},
		{"time comma fraction", `{` + identity + `,"occurred_at":"2026-09-06T04:05:06,987654321Z"}`, false},
		{"id equivalent escaped unicode", strings.Replace(base, "独立", `\u72ec\u7acb`, 1), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := openTemp(t)
			gradeGenerationSet(t, st, prefixEpisode+o.EpisodeID, []byte(tc.raw))
			before := gradeGenerationSnapshot(t, st)
			events := 0
			st.SetObserver(func(Event) { events++ })
			err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, o); return err })
			if tc.accept {
				if err != nil {
					t.Fatalf("valid raw policy control refused: %v", err)
				}
				after := gradeGenerationSnapshot(t, st)
				if len(after) != len(before)+1 || !bytes.Equal(after[prefixEpisode+o.EpisodeID], []byte(tc.raw)) {
					t.Fatal("accepted input changed episode bytes/other family")
				}
				page, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000)
				if err != nil || len(page.Records) != 1 || !reflect.DeepEqual(page.Records[0], o) {
					t.Fatalf("accepted input unreadable: %v", err)
				}
				independentPut(t, st, o)
				independentUnchanged(t, st, after)
			} else {
				if !errors.Is(err, errIdentityObservation) {
					t.Fatalf("invalid provenance accepted: %v", err)
				}
				independentUnchanged(t, st, before)
				raw, key, err := encodeIdentityObservation(o)
				if err != nil {
					t.Fatal(err)
				}
				gradeGenerationSet(t, st, key, raw)
				before = gradeGenerationSnapshot(t, st)
				if _, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000); !errors.Is(err, errIdentityObservation) {
					t.Fatalf("independently seeded row accepted on read: %v", err)
				}
				independentUnchanged(t, st, before)
			}
			if events != 0 {
				t.Fatal("provenance operation emitted graph event")
			}
		})
	}
}

func TestIndependentCorrectedProvenanceExtensionRestoreAndTransaction(t *testing.T) {
	st := openTemp(t)
	o := independentObservation(false)
	id, _ := json.Marshal(o.EpisodeID)
	at, _ := json.Marshal(o.OccurredAt)
	episodeRaw := []byte(" \n{" + `"opaque":{"id":"wrong","id":"other","nested":[null,true,1e99999]},"Occurred_At":` + string(at) + `,"ID":` + string(id) + "}\n")
	before := gradeGenerationSnapshot(t, st)
	forced := errors.New("synthetic rollback")
	err := st.AtomicWrite(func(tx *Store) error {
		if err := tx.txn.Set([]byte(prefixEpisode+o.EpisodeID), bytes.Clone(episodeRaw)); err != nil {
			return err
		}
		if _, err := putIdentityObservation(tx, o); err != nil {
			return err
		}
		page, err := readIdentityObservations(tx, o.EpisodeID, "", 10, 10000)
		if err != nil || len(page.Records) != 1 {
			t.Fatal("same-transaction raw provenance not visible")
		}
		return forced
	})
	if !errors.Is(err, forced) {
		t.Fatal(err)
	}
	independentUnchanged(t, st, before)
	if err := st.AtomicWrite(func(tx *Store) error {
		if err := tx.txn.Set([]byte(prefixEpisode+o.EpisodeID), bytes.Clone(episodeRaw)); err != nil {
			return err
		}
		_, err := putIdentityObservation(tx, o)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	before = gradeGenerationSnapshot(t, st)
	var backup bytes.Buffer
	if n, err := st.Backup(&backup); err != nil || n == 0 || backup.Len() == 0 {
		t.Fatal("backup failed")
	}
	dir := filepath.Join(t.TempDir(), "restore")
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
		t.Fatal("raw extension provenance lost on restore/read")
	}
}
