package store

import (
	"errors"
	"os"
	"reflect"
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
