package daemon

import (
	"context"
	"encoding/json"
	"testing"

	memstore "github.com/jeffdhooton/scry/internal/memory/store"
)

func TestMemoryUnaliasRawDefaultIsReadOnly(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, _ := d.memoryStore()
	if err := st.PutEntity(memstore.Entity{Slug: "app", Name: "App", Type: "project", Aliases: []string{"generic frontend"}}); err != nil {
		t.Fatal(err)
	}
	out, err := d.handleMemoryUnalias(context.Background(), json.RawMessage(`{"drops":[{"entity":"app","alias":"generic frontend","why":"Generic role is not an app identity"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if res := out.(*MemoryUnaliasResult); !res.DryRun || res.BackupPath != "" {
		t.Fatalf("omitted dry_run wrote: %+v", res)
	}
	entity, _ := st.GetEntity("app")
	if len(entity.Aliases) != 1 {
		t.Fatal("default call mutated aliases")
	}
}

func TestMemoryUnaliasUnreviewedBatchCannotPartiallyApply(t *testing.T) {
	d := newTestMemoryDaemon(t)
	st, _ := d.memoryStore()
	if err := st.PutEntity(memstore.Entity{Slug: "app", Name: "App", Type: "project", Aliases: []string{"generic frontend"}}); err != nil {
		t.Fatal(err)
	}
	_, err := d.handleMemoryUnalias(context.Background(), json.RawMessage(`{"dry_run":false,"drops":[{"entity":"app","alias":"generic frontend","why":"Generic role"},{"entity":"missing","alias":"other","why":"Invalid later row"}]}`))
	entity, _ := st.GetEntity("app")
	if len(entity.Aliases) != 1 {
		t.Fatalf("earlier row committed despite invalid later row (error %v)", err)
	}
}
