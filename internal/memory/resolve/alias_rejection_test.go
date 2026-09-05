package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestReviewedAliasRejectionBeatsAttestationsAndShortcuts(t *testing.T) {
	st := openTemp(t)
	child := putEntity(t, st, "childscribe", "ChildScribe", "project", "Envoyer")
	for _, id := range []string{"old-A", "old-B"} {
		if _, err := st.AttestAlias(child.Slug, "envoyer", id); err != nil {
			t.Fatal(err)
		}
	}
	req := store.AliasRepairRequest{Drops: []store.AliasDrop{{Entity: child.Slug, Alias: "Envoyer", Why: "Reviewed unrelated name, no inferred recipient"}}}
	p, err := st.PreviewAliasRepair(req)
	if err != nil || !p.Ready {
		t.Fatalf("preview %+v %v", p, err)
	}
	req.Expected = &p.Expected
	backup, err := os.Create(filepath.Join(t.TempDir(), "pre-repair.badger"))
	if err != nil {
		t.Fatal(err)
	}
	if n, _, err := st.BackupAndRepairAliases(backup, req); err != nil || n == 0 {
		t.Fatalf("repair %d %v", n, err)
	}
	clean, err := st.GetEntity(child.Slug)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []store.Entity{clean, child, {Slug: child.Slug, Name: "Envoyer", Type: "tool"}} {
		for i := 0; i < 10; i++ {
			ok, reason, err := AdmitAlias(st, e, "ENVOYER", fmt.Sprintf("fresh-%d", i))
			if err != nil || ok || !strings.Contains(reason, "reviewed rejection") {
				t.Fatalf("admission bypass: %v %s %v", ok, reason, err)
			}
		}
	}
	attestations, err := st.AliasAttestations(child.Slug, "envoyer")
	if err != nil || !reflect.DeepEqual(attestations, []string{"old-A", "old-B"}) {
		t.Fatalf("attestation evidence changed: %v %v", attestations, err)
	}
	oldFacts, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		id := fmt.Sprintf("normal-apply-%d", i)
		ep := store.Episode{ID: id, Source: "manual", SourceRef: id, OccurredAt: time.Unix(int64(100+i), 0).UTC()}
		result := extract.Result{EpisodeSummary: "Synthetic repeated alias assertion", Entities: []extract.Ent{{Name: child.Name, Type: child.Type, Aliases: []string{"Envoyer"}}}}
		if _, err := Apply(st, ep, "", result, DefaultExclusive); err != nil {
			t.Fatal(err)
		}
		if owner, found, err := st.ResolveAlias("Envoyer"); err != nil || found {
			t.Fatalf("normal Apply restored alias: %s %v %v", owner, found, err)
		}
		if _, err := st.GetEpisode(id); err != nil {
			t.Fatalf("declined alias lost episode: %v", err)
		}
	}
	newFacts, err := st.AllFacts()
	if err != nil || !reflect.DeepEqual(oldFacts, newFacts) {
		t.Fatalf("alias rejection changed facts: %v", err)
	}
	// The same spelling can be a legitimate separate identity. This fixture
	// tests permission to represent it, not a live ownership assignment.
	putEntity(t, st, "envoyer", "Envoyer", "tool")
	if owner, found, err := st.ResolveAlias("Envoyer"); err != nil || !found || owner != "envoyer" {
		t.Fatalf("separate identity blocked: %s %v", owner, err)
	}
}
