package store

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func reviewedMerge(t *testing.T, st *Store, survivor string, retire ...string) EntityMergeRequest {
	t.Helper()
	req := EntityMergeRequest{ID: "reviewed-test", Survivor: survivor, Retire: retire, Why: "same identity, reviewed"}
	preview, err := st.PreviewEntityMerge(req)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Ready {
		t.Fatalf("preview not ready: %+v", preview.Problems)
	}
	req.Metadata = &preview.ProposedMetadata
	req.Expected = preview.Expected
	return req
}

func TestMergeEntitiesPreservesCompleteIdentity(t *testing.T) {
	st := openTemp(t)
	created := time.Date(2026, 8, 1, 1, 2, 3, 4, time.UTC)
	seen := created.Add(48 * time.Hour)
	winner := Entity{Slug: "qwen", Name: "Qwen 3.8", Type: "tool", Description: "the model", Aliases: []string{"qwen38"}, RepoRefs: []string{"/repo/a"}, CreatedAt: seen, LastSeen: seen}
	loser := Entity{Slug: "qwen-uncensored", Name: "Qwen 3.8 Uncensored", Type: "concept", Aliases: []string{"Qwen3.8-27B"}, RepoRefs: []string{"/repo/b"}, CreatedAt: created, LastSeen: seen.Add(time.Hour)}
	other := Entity{Slug: "halo", Name: "Halo", Type: "machine"}
	for _, e := range []Entity{winner, loser, other} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	invalidAt := seen.Add(24 * time.Hour)
	facts := []Fact{
		{Src: loser.Slug, Relation: "runs_on", Dst: other.Slug, Fact: "uncensored runs on halo", ValidFrom: created, Confidence: .91, Episodes: []string{"e1"}},
		{Src: other.Slug, Relation: "uses", Dst: loser.Slug, Fact: "halo uses the model", ValidFrom: created.Add(time.Minute), InvalidAt: &invalidAt, Confidence: .72, Episodes: []string{"e2", "e3"}},
		{Src: winner.Slug, Relation: "status", Value: "ready", RawRelation: "state", Fact: "qwen is ready", ValidFrom: seen, Confidence: .8, Episodes: []string{"e4"}},
	}
	for _, f := range facts {
		if err := st.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}

	req := reviewedMerge(t, st, winner.Slug, loser.Slug)
	if req.Metadata.CreatedAt != created || req.Metadata.LastSeen != loser.LastSeen {
		t.Fatalf("proposal did not preserve temporal metadata: %+v", req.Metadata)
	}
	if !reflect.DeepEqual(req.Metadata.RepoRefs, []string{"/repo/a", "/repo/b"}) {
		t.Fatalf("proposal did not union repo refs: %v", req.Metadata.RepoRefs)
	}
	res, err := st.MergeEntities(req)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || res.FactsTouched != 3 || res.CurrentFacts != 2 || res.InvalidatedFacts != 1 {
		t.Fatalf("merge receipt = %+v", res)
	}
	if _, err := st.GetEntity(loser.Slug); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retired entity still exists: %v", err)
	}
	gotFacts, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	if len(gotFacts) != len(facts) {
		t.Fatalf("fact count changed: got %d want %d", len(gotFacts), len(facts))
	}
	for _, f := range gotFacts {
		if f.Src == loser.Slug || f.Dst == loser.Slug {
			t.Fatalf("fact still points at retired entity: %+v", f)
		}
	}
	// Every field except an endpoint changed by the merge is exact.
	wantMoved := facts[0]
	wantMoved.Src = winner.Slug
	wantInvalid := facts[1]
	wantInvalid.Dst = winner.Slug
	for _, want := range []Fact{wantMoved, wantInvalid, facts[2]} {
		found := false
		for _, got := range gotFacts {
			if reflect.DeepEqual(got, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("preserved fact missing: %+v", want)
		}
	}
	for _, spelling := range []string{winner.Name, winner.Slug, "qwen38", loser.Name, loser.Slug, "Qwen3.8-27B"} {
		if owner, ok, err := st.ResolveAlias(spelling); err != nil || !ok || owner != winner.Slug {
			t.Errorf("%q resolves to %q, %v, %v", spelling, owner, ok, err)
		}
	}
}

func TestMergeEntitiesRequiresDeduplicatedCompleteMetadata(t *testing.T) {
	st := openTemp(t)
	for _, e := range []Entity{
		{Slug: "winner", Name: "Winner", Type: "tool", Aliases: []string{"winner alias"}, RepoRefs: []string{"/repo/a"}},
		{Slug: "loser", Name: "Loser", Type: "concept", RepoRefs: []string{"/repo/b"}},
	} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.PutFact(Fact{Src: "loser", Relation: "status", Value: "ready", Fact: "loser ready", ValidFrom: time.Unix(5, 0)}); err != nil {
		t.Fatal(err)
	}
	base, err := st.PreviewEntityMerge(EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}})
	if err != nil || !base.Ready {
		t.Fatalf("base preview = %+v, %v", base.Problems, err)
	}
	metadata := base.ProposedMetadata
	metadata.Aliases = append(metadata.Aliases, "WINNER ALIAS")
	metadata.RepoRefs = []string{"/repo/a", "/repo/a"}
	bad, err := st.PreviewEntityMerge(EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}, Metadata: &metadata})
	if err != nil {
		t.Fatal(err)
	}
	problems := strings.Join(bad.Problems, " ")
	for _, want := range []string{"aliases are not uniquely normalized", "repo refs are not unique", "drops repo ref /repo/b"} {
		if !strings.Contains(problems, want) {
			t.Errorf("metadata problems %q do not contain %q", problems, want)
		}
	}
}

func TestMergeEntitiesRefusesSnapshotDriftAtomically(t *testing.T) {
	st := openTemp(t)
	at := time.Date(2026, 9, 4, 1, 0, 0, 0, time.UTC)
	for _, e := range []Entity{{Slug: "winner", Name: "Winner", Type: "tool"}, {Slug: "loser", Name: "Loser", Type: "concept"}, {Slug: "thing", Name: "Thing", Type: "tool"}} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	f := Fact{Src: "loser", Relation: "uses", Dst: "thing", Fact: "reviewed text", ValidFrom: at, Episodes: []string{"e1"}}
	if err := st.PutFact(f); err != nil {
		t.Fatal(err)
	}
	req := reviewedMerge(t, st, "winner", "loser")
	f.Fact = "text changed after review"
	if err := st.PutFact(f); err != nil {
		t.Fatal(err)
	}
	if _, err := st.MergeEntities(req); err == nil || !strings.Contains(err.Error(), "snapshot changed") {
		t.Fatalf("drift error = %v", err)
	}
	if _, err := st.GetEntity("loser"); err != nil {
		t.Fatalf("failed merge retired loser: %v", err)
	}
	stored, err := st.FactsFrom("loser", true)
	if err != nil || len(stored) != 1 || stored[0].Fact != f.Fact {
		t.Fatalf("failed merge changed facts: %+v, %v", stored, err)
	}
}

func TestMergeEntitiesRefusesSelfLoopsAndKeyCollisions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		facts []Fact
		want  string
	}{
		{name: "self loop", facts: []Fact{{Src: "loser", Relation: "uses", Dst: "winner", Fact: "the twins connect", ValidFrom: time.Unix(1, 0)}}, want: "self-loop"},
		{name: "fact key collision", facts: []Fact{
			{Src: "loser", Relation: "uses", Dst: "thing", Fact: "loser wording", ValidFrom: time.Unix(2, 0)},
			{Src: "winner", Relation: "uses", Dst: "thing", Fact: "winner wording", ValidFrom: time.Unix(2, 0)},
		}, want: "duplicate fact keys"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := openTemp(t)
			for _, e := range []Entity{{Slug: "winner", Name: "Winner", Type: "tool"}, {Slug: "loser", Name: "Loser", Type: "concept"}, {Slug: "thing", Name: "Thing", Type: "tool"}} {
				if err := st.PutEntity(e); err != nil {
					t.Fatal(err)
				}
			}
			for _, f := range tc.facts {
				if err := st.PutFact(f); err != nil {
					t.Fatal(err)
				}
			}
			bare := EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}}
			preview, err := st.PreviewEntityMerge(bare)
			if err != nil {
				t.Fatal(err)
			}
			if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), tc.want) {
				t.Fatalf("preview = %+v; want %q refusal", preview.Problems, tc.want)
			}
			bare.Metadata = &preview.ProposedMetadata
			bare.Expected = preview.Expected
			if _, err := st.MergeEntities(bare); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("apply error = %v; want %q", err, tc.want)
			}
			if _, err := st.GetEntity("loser"); err != nil {
				t.Fatalf("refused merge changed entities: %v", err)
			}
		})
	}
}

func TestMergeEntitiesRequiresReviewedMetadataAndCompleteGroup(t *testing.T) {
	st := openTemp(t)
	for _, e := range []Entity{{Slug: "winner", Name: "Winner", Type: "tool"}, {Slug: "loser", Name: "Loser", Type: "concept", Aliases: []string{"shared spelling"}}, {Slug: "outsider", Name: "Outsider", Type: "service"}} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	// An outside entity listing a group spelling is legacy malformed state.
	if err := st.ClaimAlias("shared spelling", "outsider"); err != nil {
		t.Fatal(err)
	}
	outsider, _ := st.GetEntity("outsider")
	outsider.Aliases = []string{"shared spelling"}
	if err := st.PutEntity(outsider); err != nil {
		t.Fatal(err)
	}
	if err := st.PutFact(Fact{Src: "loser", Relation: "status", Value: "ready", Fact: "loser is ready", ValidFrom: time.Unix(4, 0)}); err != nil {
		t.Fatal(err)
	}
	preview, err := st.PreviewEntityMerge(EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Ready || len(preview.ExternalAliasOwners) == 0 || len(preview.ExternalListings) == 0 {
		t.Fatalf("incomplete group was accepted: %+v", preview)
	}

	// A reviewer may explicitly retire an alias, with a reason, while names
	// and slugs remain mandatory. That leaves the outsider's rightful claim
	// untouched instead of stealing it into the merge.
	req := EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}, DropAliases: []EntityMergeAliasDrop{{Alias: "shared spelling", RehomeTo: "outsider", Why: "generic spelling belongs to outsider"}}}
	preview, err = st.PreviewEntityMerge(req)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Ready || len(preview.DroppedAliases) != 1 {
		t.Fatalf("reviewed alias retirement refused: %+v", preview)
	}
	for _, alias := range preview.ProposedMetadata.Aliases {
		if Normalize(alias) == Normalize("shared spelling") {
			t.Fatalf("proposed metadata kept dropped alias: %v", preview.ProposedMetadata.Aliases)
		}
	}
	req.Metadata = &preview.ProposedMetadata
	req.Expected = preview.Expected
	if _, err := st.MergeEntities(req); err != nil {
		t.Fatal(err)
	}
	if owner, ok, err := st.ResolveAlias("shared spelling"); err != nil || !ok || owner != "outsider" {
		t.Fatalf("reviewed drop stole outsider claim: %q, %v, %v", owner, ok, err)
	}
}

func TestMergeEntitiesRehomesAGroupOwnedDroppedAlias(t *testing.T) {
	st := openTemp(t)
	loser := Entity{Slug: "loser", Name: "Loser", Type: "concept", Aliases: []string{"shared spelling"}}
	outsider := Entity{Slug: "outsider", Name: "Outsider", Type: "service"}
	for _, e := range []Entity{{Slug: "winner", Name: "Winner", Type: "tool"}, loser, outsider} {
		if err := st.PutEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	// Manufacture legacy state in which the outsider lists the spelling but
	// the retiring group still owns its routing key.
	if err := st.ClaimAlias("shared spelling", outsider.Slug); err != nil {
		t.Fatal(err)
	}
	outsider.Aliases = []string{"shared spelling"}
	if err := st.PutEntity(outsider); err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimAlias("shared spelling", loser.Slug); err != nil {
		t.Fatal(err)
	}
	if err := st.PutFact(Fact{Src: loser.Slug, Relation: "status", Value: "ready", Fact: "loser is ready", ValidFrom: time.Unix(4, 0)}); err != nil {
		t.Fatal(err)
	}

	withoutTarget := EntityMergeRequest{Survivor: "winner", Retire: []string{loser.Slug}, DropAliases: []EntityMergeAliasDrop{{Alias: "shared spelling", Why: "belongs to outsider"}}}
	preview, err := st.PreviewEntityMerge(withoutTarget)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), "requires rehome_to") {
		t.Fatalf("drop without rehome was accepted: %+v", preview.Problems)
	}

	req := EntityMergeRequest{Survivor: "winner", Retire: []string{loser.Slug}, DropAliases: []EntityMergeAliasDrop{{Alias: "shared spelling", RehomeTo: outsider.Slug, Why: "belongs to outsider"}}}
	preview, err = st.PreviewEntityMerge(req)
	if err != nil || !preview.Ready {
		t.Fatalf("reviewed rehome preview = %+v, %v", preview.Problems, err)
	}
	req.Metadata = &preview.ProposedMetadata
	req.Expected = preview.Expected
	if _, err := st.MergeEntities(req); err != nil {
		t.Fatal(err)
	}
	if owner, ok, err := st.ResolveAlias("shared spelling"); err != nil || !ok || owner != outsider.Slug {
		t.Fatalf("rehomed alias resolves to %q, %v, %v", owner, ok, err)
	}
}

func TestMergeEntitiesRefusesHollowOrDanglingGroups(t *testing.T) {
	t.Run("hollow", func(t *testing.T) {
		st := openTemp(t)
		for _, e := range []Entity{{Slug: "winner", Name: "Winner", Type: "tool"}, {Slug: "loser", Name: "Loser", Type: "concept"}} {
			if err := st.PutEntity(e); err != nil {
				t.Fatal(err)
			}
		}
		preview, err := st.PreviewEntityMerge(EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}})
		if err != nil {
			t.Fatal(err)
		}
		if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), "hollow") {
			t.Fatalf("hollow merge preview = %+v", preview.Problems)
		}
	})

	t.Run("dangling", func(t *testing.T) {
		st := openTemp(t)
		for _, e := range []Entity{{Slug: "winner", Name: "Winner", Type: "tool"}, {Slug: "loser", Name: "Loser", Type: "concept"}} {
			if err := st.PutEntity(e); err != nil {
				t.Fatal(err)
			}
		}
		if err := st.PutFact(Fact{Src: "loser", Relation: "uses", Dst: "missing", Fact: "loser uses missing", ValidFrom: time.Unix(3, 0)}); err != nil {
			t.Fatal(err)
		}
		preview, err := st.PreviewEntityMerge(EntityMergeRequest{Survivor: "winner", Retire: []string{"loser"}})
		if err != nil {
			t.Fatal(err)
		}
		if preview.Ready || !strings.Contains(strings.Join(preview.Problems, " "), "dangling destination") {
			t.Fatalf("dangling merge preview = %+v", preview.Problems)
		}
	})
}
