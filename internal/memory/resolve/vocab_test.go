package resolve

import (
	"strings"
	"testing"
)

func TestVocabularyIsClosedAndSmall(t *testing.T) {
	if len(Canonical) != 39 {
		t.Fatalf("Canonical has %d relations, want the documented 39", len(Canonical))
	}
	seen := map[string]bool{}
	for _, r := range Canonical {
		if seen[r] {
			t.Errorf("duplicate canonical relation %q", r)
		}
		seen[r] = true
		if normalizeRelation(r) != r {
			t.Errorf("canonical relation %q is not in normalized form", r)
		}
		rel, flip := Map(r)
		if rel != r || flip {
			t.Errorf("Map(%q) = %q flip=%v, a canonical relation must map to itself", r, rel, flip)
		}
	}
	if !IsCanonical(RelUses) || IsCanonical("has_bug") {
		t.Error("IsCanonical wrong")
	}
	if SynonymCount() < 300 {
		t.Errorf("synonym table has only %d entries", SynonymCount())
	}
}

func TestMapObservedRelations(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		flip bool
	}{
		// exact table
		{"has_status", RelStatus, false},
		{"test_status", RelStatus, false},
		{"verdict", RelStatus, false},
		{"used_by", RelUses, true},
		{"owned_by", RelOwns, true},
		{"fixed_by", RelFixes, true},
		{"fixed", RelFixes, false},
		{"reviewed_by", RelReviews, true},
		{"requested_review_from", RelReviews, false},
		{"verified", RelTests, false},
		{"verified_by", RelTests, true},
		{"validates", RelTests, false},
		{"covers", RelTests, false},
		{"passed_gate", RelPasses, false},
		{"approved", RelApproves, false},
		{"blocks", RelBlockedBy, true},
		{"supersedes", RelReplacedBy, true},
		{"replaces", RelReplacedBy, true},
		{"merged_to", RelMergedInto, false},
		{"committed", RelMergedInto, false},
		{"hosts", RelRunsOn, true},
		{"installed_on", RelDeployedOn, false},
		{"deployed", RelDeployedOn, false},
		{"exists_on", RelRunsOn, false},
		{"stored_at", RelLocatedAt, false},
		{"located_in", RelLocatedAt, false},
		{"belongs_to", RelPartOf, false},
		{"includes", RelContains, false},
		{"has", RelContains, false},
		{"has_feature", RelContains, false},
		{"has_bug", RelHasIssue, false},
		{"has_defect", RelHasIssue, false},
		{"had_bug", RelHasIssue, false},
		{"claims", RelAssignedTo, true},
		{"works_on", RelAssignedTo, true},
		{"assigned_to", RelAssignedTo, false},
		{"exposes", RelProvides, false},
		{"returns", RelProvides, false},
		{"created", RelProduces, false},
		{"generates", RelProduces, false},
		{"records", RelDocuments, false},
		{"tracks", RelDocuments, false},
		{"found", RelDocuments, false},
		{"reported", RelDocuments, false},
		{"missing", RelLacks, false},
		{"has_no", RelLacks, false},
		{"needs", RelRequires, false},
		{"contradicts", RelConflictsWith, false},
		{"affects", RelCauses, false},
		{"caused_by", RelCauses, true},
		{"configured_with", RelConfigures, false},
		{"posted_to", RelDocuments, false},
		{"runs", RelCalls, false},
		{"launched_by", RelRunsOn, false},
		{"same_as", RelSameAs, false},
		{"identical_to", RelSameAs, false},
		{"completed", RelImplements, false},
		{"shipped", RelImplements, false},
		// suffix and prefix rules
		{"deployed_to", RelDeployedOn, false},
		{"running_on", RelRunsOn, false},
		{"stored_in", RelLocatedAt, false},
		{"monitored_by", RelMonitors, true},
		{"tested_by", RelTests, true},
		{"has_epic", RelContains, false},
		{"has_security_gap", RelHasIssue, false},
		{"has_handshake_p50", RelStatus, false},
		{"has_gpu_count", RelStatus, false},
		{"does_not_use", RelLacks, false},
		{"never_touches", RelLacks, false},
		{"now_uses", RelUses, false},
		{"will_deploy_to", RelDeployedOn, false},
		{"is_using", RelUses, false},
		// stems
		{"verifying", RelTests, false},
		{"deployment", RelDeployedOn, false},
		{"utilizes", RelUses, false},
		{"orchestrates", RelOwns, false},
		{"notified", RelNotifies, false},
		{"escalates_to", RelNotifies, false},
		{"refactored", RelModifies, false},
		{"enforces_constraint", RelEnforces, false},
		{"forbids", RelEnforces, false},
		{"excluded_from", RelExcludes, false},
		{"skips", RelExcludes, false},
		// fallback
		{"robots_method_now_welcomes", RelRelatedTo, false},
		{"outperforms", RelRelatedTo, false},
		{"", RelRelatedTo, false},
		{"___", RelRelatedTo, false},
	}
	for _, tc := range cases {
		rel, flip := Map(tc.raw)
		if rel != tc.want || flip != tc.flip {
			t.Errorf("Map(%q) = %q flip=%v, want %q flip=%v", tc.raw, rel, flip, tc.want, tc.flip)
		}
		if !IsCanonical(rel) {
			t.Errorf("Map(%q) returned non-canonical %q", tc.raw, rel)
		}
	}
}

func TestStem(t *testing.T) {
	for in, want := range map[string]string{
		"verifies": "verif", "verified": "verif", "verifying": "verify", "verification": "verif",
		"deploys": "deploy", "deployment": "deploy", "uses": "us", "hosts": "host", "runs": "run",
	} {
		if got := stem(in); got != want {
			t.Errorf("stem(%q) = %q, want %q", in, got, want)
		}
	}
}

// Routing a spelling is not asserting that its former and current owners are
// the same identity. Nor does a shared measurement or an unknown compound
// containing an identity verb establish identity.
func TestMapIdentityRequiresWholeRelation(t *testing.T) {
	for _, raw := range []string{
		"aliases_index_to", "rehomes_aliases_to", // observed ingestion failures
		"aliases_registry", "aliasing_configuration", "equal_latency",
		"equivalent_capacity", "duplicate_records", "clones_configuration",
		"identical_measurements", "unknown_synonyms", "unknown_aliases",
		"aliases_by", "same_as_with", "alias_of_to",
	} {
		for _, prefix := range []string{"", "currently_", "was_", "will_"} {
			got, flip := Map(prefix + raw)
			if got == RelSameAs || !IsCanonical(got) || flip {
				t.Errorf("Map(%q) = %q, flip=%v; must not infer identity", prefix+raw, got, flip)
			}
		}
	}
	for _, raw := range []string{"aliases_index_to", "rehomes_aliases_to"} {
		if got, flip := Map(raw); got != Fallback || flip {
			t.Errorf("Map(%q) = %q, flip=%v; untyped routing must retain sentence nuance in fallback", raw, got, flip)
		}
	}
}

func TestMapPreservesEveryExplicitIdentityRelation(t *testing.T) {
	for raw, want := range synonyms {
		if want.rel != RelSameAs {
			continue
		}
		for _, prefix := range []string{"", "currently_", "was_", "will_", "has_been_", "already_is_"} {
			input := " __" + strings.ToUpper(prefix+raw) + "__ "
			got, flip := Map(input)
			if got != want.rel || flip != want.flip {
				t.Errorf("Map(%q) = %q, flip=%v; want explicit identity %q, flip=%v", input, got, flip, want.rel, want.flip)
			}
		}
	}
	for _, raw := range []string{"not_alias_of", "does_not_equal", "is_not_same_as", "never_aliases"} {
		if got, flip := Map(raw); got != RelLacks || flip {
			t.Errorf("Map(%q) = %q, flip=%v; negation must survive", raw, got, flip)
		}
	}
}
