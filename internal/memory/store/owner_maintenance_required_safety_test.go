package store

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestOwnerMaintenanceRequiredSafety(t *testing.T) {
	for _, phase := range []string{"after-return", "finalizer-failure"} {
		t.Run(phase, func(t *testing.T) {
			s := openTemp(t)
			for _, e := range []Entity{{Slug: "model-a", Name: "Model A", Type: "tool"}, {Slug: "model-b", Name: "Model B", Type: "tool"}} {
				if err := s.PutEntity(e); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.PutFact(Fact{Src: "model-b", Relation: "status", Value: "ready", Fact: "Model B is ready", ValidFrom: time.Unix(55, 0).UTC()}); err != nil {
				t.Fatal(err)
			}
			req := EntityMergeRequest{ID: "synthetic-reviewed-maintenance", Survivor: "model-a", Retire: []string{"model-b"}, Why: "Synthetic same identity disposition"}
			preview, err := s.PreviewEntityMerge(req)
			if err != nil || !preview.Ready {
				t.Fatalf("preview %+v %v", preview, err)
			}
			req.Expected, req.Metadata = preview.Expected, &preview.ProposedMetadata
			before := ownerRaw(t, s)
			var escaped *Store
			var maintenanceErr error
			marker := errors.New("reject outer owner")
			err = runIdentityAdmission(s, func(tx *Store) error { escaped = tx; return nil }, func(tx *Store) error {
				if phase == "finalizer-failure" {
					_, maintenanceErr = tx.MergeEntities(req)
					return marker
				}
				return nil
			})
			if phase == "finalizer-failure" && err == nil {
				t.Error("outer failure was lost")
			}
			if phase == "after-return" {
				if err != nil {
					t.Fatal(err)
				}
				_, maintenanceErr = escaped.MergeEntities(req)
			}
			if maintenanceErr == nil {
				t.Error("admission facade accepted public maintenance at " + phase)
			}
			if !reflect.DeepEqual(before, ownerRaw(t, s)) {
				t.Error("public facade maintenance changed committed bytes at " + phase)
			}
		})
	}
}
