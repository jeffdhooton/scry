package recall

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

func TestCuratedPresentationUsesObservedRevisionWithinNormalBudget(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now().UTC()
	repo, path := "/fixture/scry", "/fixture/scry/constraint.txt"
	// Deliberately store the older observation LAST. Completion order is not
	// edit order, and no source filesystem is available to orientation.
	for _, tc := range []struct {
		id, text           string
		observed, ingested time.Time
	}{
		{"new", strings.Repeat("Updated rule. ", 40), now, now},
		{"old", "Obsolete rule", now.Add(-time.Hour), now.Add(time.Hour)},
	} {
		ep := store.Episode{ID: tc.id, Source: distill.CuratedSource, SourceRef: distill.CuratedRef(path, tc.text), Summary: tc.text, Cwd: repo, CwdIsRepo: true, OccurredAt: tc.observed, IngestedAt: tc.ingested}
		if err := st.PutEpisode(ep); err != nil {
			t.Fatal(err)
		}
	}
	// Saturate both ordinary sections so the source rule must actually win
	// the existing trimming order, not just fit in an otherwise empty store.
	for i := 0; i < 20; i++ {
		slug := fmt.Sprintf("noise-%d", i)
		owner := repo
		if i%2 == 0 {
			owner = "/fixture/elsewhere"
		}
		if err := st.PutEntity(store.Entity{Slug: slug, Name: slug, Type: "project", RepoRefs: []string{owner}, CreatedAt: now, LastSeen: now}); err != nil {
			t.Fatal(err)
		}
		if err := st.PutFact(store.Fact{Src: slug, Relation: "uses", Value: "noise", Fact: strings.Repeat("ordinary noise ", 20), ValidFrom: now, Confidence: .9}); err != nil {
			t.Fatal(err)
		}
	}
	md, err := Orient(st, repo, 0, now)
	if err != nil || len(md) > 2000 || !strings.Contains(md, strings.TrimSpace(strings.Repeat("Updated rule. ", 40))) || !strings.Contains(md, path) || strings.Contains(md, "Obsolete rule") {
		t.Fatalf("bad budget/revision selection: %v\n%s", err, md)
	}
}

func TestCuratedPresentationRejectsUnattestedOrAlteredEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		attested   bool
	}{
		{"no attestation", "Authored rule", false}, {"paraphrased", "Different summary", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, err := store.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			ep := store.Episode{ID: "rule", Source: distill.CuratedSource, SourceRef: distill.CuratedRef("/fixture/scry/rule.txt", "Authored rule"), Summary: tc.text, Cwd: "/fixture/scry", CwdIsRepo: tc.attested, OccurredAt: time.Now()}
			if err := st.PutEpisode(ep); err != nil {
				t.Fatal(err)
			}
			md, err := Orient(st, ep.Cwd, 0, time.Now())
			if err != nil || strings.Contains(md, "rule") || strings.Contains(md, "summary") {
				t.Fatalf("unsupported evidence presented: %v %s", err, md)
			}
		})
	}
}
