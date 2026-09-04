package store

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestLiveStoreEntityRewriteIsOwnershipNeutral exercises PutEntity against a
// restored production backup. Never point SCRY_STORE_CHECK_DIR at the live
// daemon store: this test writes a probe entity and then removes it.
func TestLiveStoreEntityRewriteIsOwnershipNeutral(t *testing.T) {
	dir := os.Getenv("SCRY_STORE_CHECK_DIR")
	if dir == "" {
		t.Skip("SCRY_STORE_CHECK_DIR not set")
	}
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	before, err := st.AliasClaims()
	if err != nil {
		t.Fatal(err)
	}
	entities, err := st.Entities()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entities {
		if err := st.PutEntity(e); err != nil {
			t.Fatalf("rewrite unchanged entity %s: %v", e.Slug, err)
		}
	}
	afterRewrite, err := st.AliasClaims()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, afterRewrite) {
		t.Fatal("rewriting unchanged entities changed alias ownership")
	}

	const probeSlug = "zz-scry-alias-ownership-replica-probe"
	if _, err := st.GetEntity(probeSlug); !errors.Is(err, ErrNotFound) {
		t.Fatalf("replica unexpectedly contains probe slug %s", probeSlug)
	}
	probe := Entity{Slug: probeSlug, Name: probeSlug, Type: "concept"}
	if err := st.PutEntity(probe); err != nil {
		t.Fatal(err)
	}
	var claimed string
	for norm, owner := range before {
		if owner != probeSlug {
			claimed = norm
			break
		}
	}
	if claimed == "" {
		t.Fatal("replica has no existing alias claim to test")
	}
	probe.Aliases = []string{claimed}
	if err := st.PutEntity(probe); !errors.Is(err, ErrAliasClaimed) {
		t.Fatalf("conflicting probe write error = %v; want ErrAliasClaimed", err)
	}
	stored, err := st.GetEntity(probeSlug)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Aliases) != 0 {
		t.Fatalf("conflicting probe write was partial: %+v", stored)
	}
	if err := st.DeleteEntity(probeSlug); err != nil {
		t.Fatal(err)
	}
	final, err := st.AliasClaims()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, final) {
		t.Fatal("replica alias ownership did not return to its exact starting state")
	}
	t.Logf("rewrote %d entities; preserved %d alias claims", len(entities), len(before))
}

// TestLiveStoreMergedEntityPostconditions independently audits a merge that
// was just applied to an offline replica.
func TestLiveStoreMergedEntityPostconditions(t *testing.T) {
	dir := os.Getenv("SCRY_MERGE_CHECK_DIR")
	survivor := os.Getenv("SCRY_MERGE_SURVIVOR")
	retiredCSV := os.Getenv("SCRY_MERGE_RETIRED")
	droppedCSV := os.Getenv("SCRY_MERGE_DROPPED")
	if dir == "" || survivor == "" || retiredCSV == "" {
		t.Skip("SCRY_MERGE_CHECK_DIR, SCRY_MERGE_SURVIVOR, and SCRY_MERGE_RETIRED not set")
	}
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	retired := map[string]bool{}
	for _, slug := range strings.Split(retiredCSV, ",") {
		retired[strings.TrimSpace(slug)] = true
	}
	for slug := range retired {
		if _, err := st.GetEntity(slug); !errors.Is(err, ErrNotFound) {
			t.Errorf("retired entity %s still exists: %v", slug, err)
		}
		if owner, ok, err := st.ResolveAlias(slug); err != nil || !ok || owner != survivor {
			t.Errorf("retired slug %s resolves to %q, %v, %v", slug, owner, ok, err)
		}
	}
	e, err := st.GetEntity(survivor)
	if err != nil {
		t.Fatal(err)
	}
	for _, spelling := range append([]string{e.Slug, e.Name}, e.Aliases...) {
		if owner, ok, err := st.ResolveAlias(spelling); err != nil || !ok || owner != survivor {
			t.Errorf("survivor spelling %q resolves to %q, %v, %v", spelling, owner, ok, err)
		}
	}
	claims, err := st.AliasClaims()
	if err != nil {
		t.Fatal(err)
	}
	for norm, owner := range claims {
		if retired[owner] {
			t.Errorf("alias %q still points at retired entity %s", norm, owner)
		}
	}
	if droppedCSV != "" {
		entities, err := st.Entities()
		if err != nil {
			t.Fatal(err)
		}
		for _, spelling := range strings.Split(droppedCSV, ",") {
			norm := Normalize(strings.TrimSpace(spelling))
			if owner, found := claims[norm]; found {
				t.Errorf("dropped alias %q still resolves to %s", spelling, owner)
			}
			for _, e := range entities {
				if entityListsNormalized(e, norm) {
					t.Errorf("dropped alias %q is still listed by %s", spelling, e.Slug)
				}
			}
		}
	}
	facts, err := st.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	touching := 0
	for _, f := range facts {
		if retired[f.Src] || retired[f.Dst] {
			t.Errorf("fact still points at retired entity: %+v", f)
		}
		if f.Src == survivor || f.Dst == survivor {
			touching++
		}
	}
	if touching == 0 {
		t.Error("survivor is hollow")
	}
	t.Logf("survivor %s has %d touching facts; %d total facts and %d alias claims have no retired endpoint", survivor, touching, len(facts), len(claims))
}
