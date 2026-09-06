package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestObservationRawProvenanceIdentityFields(t *testing.T) {
	for _, mode := range []string{"uppercase-id-duplicate", "uppercase-time-duplicate", "missing-id", "missing-time", "null-id", "number-time", "trailing-object", "invalid-utf8", "noncanonical-id", "unknown-extension-control"} {
		t.Run(mode, func(t *testing.T) {
			st := openTemp(t)
			o := observationFixture()
			observationEpisode(t, st, o)
			id, _ := json.Marshal(o.EpisodeID)
			at, _ := json.Marshal(o.OccurredAt)
			raw := []byte(`{"id":` + string(id) + `,"occurred_at":` + string(at) + `}`)
			switch mode {
			case "uppercase-id-duplicate":
				raw = append([]byte(`{"ID":`+string(id)+`,`), raw[1:]...)
			case "uppercase-time-duplicate":
				raw = append([]byte(`{"Occurred_At":`+string(at)+`,`), raw[1:]...)
			case "missing-id":
				raw = []byte(`{"occurred_at":` + string(at) + `}`)
			case "missing-time":
				raw = []byte(`{"id":` + string(id) + `}`)
			case "null-id":
				raw = bytes.Replace(raw, id, []byte("null"), 1)
			case "number-time":
				raw = bytes.Replace(raw, at, []byte("123"), 1)
			case "trailing-object":
				raw = append(raw, []byte(` {}`)...)
			case "invalid-utf8":
				raw = append([]byte("{\"opaque\":\"\xff\","), raw[1:]...)
			case "noncanonical-id":
				raw = bytes.Replace(raw, []byte("observation"), []byte(`\u006fbservation`), 1)
			case "unknown-extension-control":
				raw = append([]byte(` {"future":{"nested":[1,true,null]},`), raw[1:]...)
			}
			gradeGenerationSet(t, st, prefixEpisode+o.EpisodeID, raw)
			before := gradeGenerationSnapshot(t, st)
			err := st.AtomicWrite(func(tx *Store) error { _, err := putIdentityObservation(tx, o); return err })
			if mode == "unknown-extension-control" {
				if err != nil {
					t.Fatal("unrelated extension refused", err)
				}
				page, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000)
				if err != nil || len(page.Records) != 1 {
					t.Fatal("extension control unreadable")
				}
				if !bytes.Equal(raw, gradeGenerationSnapshot(t, st)[prefixEpisode+o.EpisodeID]) {
					t.Fatal("episode extension rewritten")
				}
				return
			}
			if !errors.Is(err, errIdentityObservation) || !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("ambiguous provenance accepted or modified")
			}
			obsRaw, key, _ := encodeIdentityObservation(o)
			gradeGenerationSet(t, st, key, obsRaw)
			before = gradeGenerationSnapshot(t, st)
			if _, err := readIdentityObservations(st, o.EpisodeID, "", 10, 10000); !errors.Is(err, errIdentityObservation) {
				t.Fatal("unsafe provenance accepted on read")
			}
			if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) {
				t.Fatal("read modified bytes")
			}
		})
	}
}
