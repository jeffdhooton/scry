package resolve

import (
	"maps"
	"sort"
	"strings"
)

// The relation vocabulary is closed. The extraction model may write any
// verb it likes; the resolver maps it onto one of these before a fact is
// stored, so a path between two entities means something and a query for
// "what runs on the mini" does not have to guess between deployed_on,
// installed_on, exists_on, hosted_on, and running_on. The model's original
// wording is kept on the fact as RawRelation.
//
// Direction matters. Some raw relations are the inverse of a canonical one
// (used_by is uses with src and dst swapped); Map reports that as flip.
const (
	RelStatus        = "status"
	RelUses          = "uses"
	RelDependsOn     = "depends_on"
	RelRequires      = "requires"
	RelLacks         = "lacks"
	RelBlockedBy     = "blocked_by"
	RelConflictsWith = "conflicts_with"
	RelCauses        = "causes"
	RelFixes         = "fixes"
	RelHasIssue      = "has_issue"
	RelTests         = "tests"
	RelPasses        = "passes"
	RelReviews       = "reviews"
	RelApproves      = "approves"
	RelDecided       = "decided"
	RelReplacedBy    = "replaced_by"
	RelMergedInto    = "merged_into"
	RelDeployedOn    = "deployed_on"
	RelRunsOn        = "runs_on"
	RelLocatedAt     = "located_at"
	RelPartOf        = "part_of"
	RelContains      = "contains"
	RelOwns          = "owns"
	RelAssignedTo    = "assigned_to"
	RelImplements    = "implements"
	RelProvides      = "provides"
	RelProduces      = "produces"
	RelDocuments     = "documents"
	RelMonitors      = "monitors"
	RelConfigures    = "configures"
	RelCalls         = "calls"
	RelReferences    = "references"
	RelTargets       = "targets"
	RelNotifies      = "notifies"
	RelEnforces      = "enforces"
	RelExcludes      = "excludes"
	RelModifies      = "modifies"
	RelSameAs        = "same_as"
	RelRelatedTo     = "related_to"
)

// Canonical is every relation a stored fact may carry, in a stable order.
var Canonical = []string{
	RelStatus, RelUses, RelDependsOn, RelRequires, RelLacks, RelBlockedBy, RelConflictsWith,
	RelCauses, RelFixes, RelHasIssue, RelTests, RelPasses, RelReviews, RelApproves, RelDecided,
	RelReplacedBy, RelMergedInto, RelDeployedOn, RelRunsOn, RelLocatedAt, RelPartOf, RelContains,
	RelOwns, RelAssignedTo, RelImplements, RelProvides, RelProduces, RelDocuments, RelMonitors,
	RelConfigures, RelCalls, RelReferences, RelTargets, RelNotifies, RelEnforces, RelExcludes,
	RelModifies, RelSameAs, RelRelatedTo,
}

// Fallback is where a relation nothing else claims lands. The fact sentence
// still carries the nuance; the edge just stops pretending to be typed.
const Fallback = RelRelatedTo

var canonicalSet = func() map[string]bool {
	m := make(map[string]bool, len(Canonical))
	for _, r := range Canonical {
		m[r] = true
	}
	return m
}()

// IsCanonical reports whether rel is in the closed vocabulary.
func IsCanonical(rel string) bool { return canonicalSet[rel] }

// mapping is one synonym-table entry.
type mapping struct {
	rel  string
	flip bool
}

func m(rel string) mapping   { return mapping{rel: rel} }
func inv(rel string) mapping { return mapping{rel: rel, flip: true} }
func entries(rel string, flip bool, names ...string) map[string]mapping {
	out := make(map[string]mapping, len(names))
	for _, n := range names {
		out[n] = mapping{rel: rel, flip: flip}
	}
	return out
}

// synonyms is the exact-match table, built from the relation names seen in
// the live store on 2026-09-02 (5,595 distinct across 27k facts). Only
// forms the rules below cannot derive need to be here, but common ones are
// listed anyway so the table reads as documentation.
var synonyms = func() map[string]mapping {
	t := map[string]mapping{}
	add := func(rel string, flip bool, names ...string) {
		maps.Copy(t, entries(rel, flip, names...))
	}
	add(RelStatus, false, "status", "has_status", "test_status", "build_status", "verdict", "has_verdict", "state", "has_state", "progress", "result", "has_outcome", "outcome", "health", "is", "is_a", "reports_status", "reported_status", "posted_status", "posted_verdict", "set_status", "writes_status", "has_wrong_status", "marked_passed", "flipped_status", "scored", "scored_suitability", "version", "has_version", "scope", "behavior", "has_behavior", "duration", "file_size", "speed", "has_speed", "strength", "dimensions", "format", "location", "test_count", "has_test_count", "test_result", "test_coverage", "has_coverage", "has_test_coverage", "quality_assessed_by", "costs", "has_ttl", "has_size", "has_capacity", "has_median", "has_handshake_p50", "has_input_price_per_mtok", "has_output_price_per_mtok", "has_active_parameters", "has_bingo_points", "has_monthly_credit_for", "has_value", "has_metric", "measured", "measured_on", "counts", "is_frozen", "frozen", "is_stale", "is_public", "is_open", "is_unmerged", "unmerged", "is_running", "running", "is_built", "fully_built", "is_partially_built", "partially_built", "partial", "not_built", "unbuilt", "unwired", "pending", "blocked", "closed", "archived", "abandoned", "deferred", "stopped", "disabled", "was_fixed", "confirmed_clean", "passed_review")
	add(RelUses, false, "uses", "use", "used", "using", "reads", "read", "reads_from", "consumes", "imports", "fetches", "fetches_from", "polls", "receives", "relies_on", "leverages", "built_with", "built_on", "based_on", "now_uses", "will_use", "should_use", "tested_with", "verified_with", "sources", "accessed_via", "reachable_via", "routes_through", "authenticates", "authenticated_on", "connects_to", "connected_to", "integrated_with", "synced_with", "synced_to", "falls_back_to", "fallback_to", "defaults_to")
	add(RelUses, true, "used_by", "used_in", "used_on", "used_for", "used_as", "consumed_by", "imported_by", "read_by")
	add(RelDependsOn, false, "depends_on", "depends", "dependency", "awaits", "awaiting", "waits_for", "waiting_on", "gated_by", "needed_from", "builds_on", "derives", "derived_from", "extends", "inputs_to", "output_feeds", "feeds")
	add(RelRequires, false, "requires", "require", "required", "needs", "need", "needs_fix", "requires_fix", "requires_change", "requires_data", "requires_file", "requires_skill", "must_implement", "must_pass", "mandates", "expects", "needs_update", "needs_method", "prerequisite")
	add(RelRequires, true, "required_by", "required_for", "needed_by")
	add(RelLacks, false, "lacks", "missing", "has_no", "has_gap", "has_no_remote", "is_missing", "missing_from", "missing_coverage", "missing_feature", "missing_in", "missing_data", "missing_routes", "missing_validation", "missing_constraint", "missing_property_tests", "missing_spec_requirement", "lacks_epic", "lacks_coverage", "does_not_contain", "does_not_cover", "does_not_test", "does_not_support", "does_not_use", "does_not_modify", "does_not_exist", "does_not_apply", "is_absent", "absent_from", "not_on", "never_writes_status", "no_work_produced", "omits", "has_coverage_gap", "has_test_gap", "left_behind")
	add(RelBlockedBy, false, "blocked_by", "blocks_by", "blocked_on", "blocks_on", "is_blocked_by", "stopped_by", "gated_on", "waits_on")
	add(RelBlockedBy, true, "blocks", "block", "unblocks", "gates", "claims_block", "constrains", "prevents", "locks_contract", "locked_contract", "freezes", "freezes_path", "frozen_path", "pins", "pins_block", "reserves_block", "reserves_block_for", "assigns_block", "assigns_span", "owns_block", "assigned_block", "has_migrations_in_block")
	add(RelConflictsWith, false, "conflicts_with", "conflicts", "contradicts", "contradicted_by", "diverges_from", "diverged_from", "differs_from", "disagrees_with", "incompatible_with", "collides_with", "competes_with", "resolves_collision", "violates", "violates_spec", "spec_violation", "breaks", "broken_by")
	add(RelCauses, false, "causes", "caused", "cause", "affects", "throws", "fails", "fails_on", "failed", "failed_on", "failed_due_to", "failed_at", "failing", "fails_to_deploy", "triggers", "triggered", "introduces", "introduced", "introduces_regression", "root_cause", "reduces", "improves", "improved", "forces", "drives", "experienced")
	add(RelCauses, true, "caused_by", "affected_by", "triggered_by", "broken_by")
	add(RelFixes, false, "fixes", "fix", "fixed", "resolved", "resolved_on", "fixed_bug", "fixed_defect", "fixed_issue", "fixed_in", "fixed_to", "fixed_for", "fixed_at", "fixing", "resolves", "resolved_to", "resolved_finding", "repairs", "corrects", "corrected", "patched", "addresses", "addressed", "solves", "solution_is", "cleaned_up", "restored", "reverted", "recomputes", "revalidates", "repaired")
	add(RelFixes, true, "fixed_by", "resolved_by", "has_fix", "patched_by", "corrected_by")
	add(RelHasIssue, false, "has_issue", "has_bug", "has_defect", "had_bug", "had_defect", "had_issue", "has_finding", "has_findings", "has_error", "has_problem", "has_risk", "has_risk_level", "has_limitation", "has_quality_issue", "has_known_issue", "has_conflict", "has_critical_issue", "has_important_issue", "has_minor_issue", "has_unresolved_finding", "has_live_findings", "has_live_finding_on", "has_security_gap", "has_gotcha", "has_missing_table", "bug", "defect_in", "found_bug", "found_defect", "found_defect_in", "found_defects_in", "found_issue", "found_issue_in", "identified_defect", "identified_defect_in", "identified_bugs", "identified_gap", "identified_finding", "identified_assumption", "minor_finding", "flags", "flagged", "misreports", "misreports_status_of", "contains_unverified_claim", "shows_unchecked", "has_wrong_status", "regression")
	add(RelTests, false, "tests", "test", "tested", "verified", "verified_on", "confirmed", "validated", "tested_in", "tested_via", "verifies", "verify", "verified_against", "verified_in", "verified_with", "validates", "validates_against", "validate", "checks", "checked", "check", "covers", "cover", "covers_contract", "covers_branch", "exercises", "proves", "confirms", "audits", "audited", "inspected", "investigates", "researched", "reproduced", "evaluated", "diagnosed", "scans", "measures", "mocks", "ran_tests_on", "adds_tests", "adds_regression_for", "has_test", "has_tests", "has_test_suite", "has_property_tests", "has_test_contract", "enforces_coverage_on")
	add(RelTests, true, "tested_by", "verified_by", "validated_by", "checked_by", "covered_by", "audited_by", "reviewed_by_test")
	add(RelPasses, false, "passes", "passed", "pass", "passes_gate", "passed_gate", "passes_checks", "passed_checks", "passes_tests", "passed_tests", "passes_on", "passed_on", "passed_for", "has_tests_passing", "all_tests_race_green", "green", "satisfies", "meets", "achieves", "achieved", "achieved_result", "fits_on", "respects_boundary")
	add(RelReviews, false, "reviews", "review", "reviewed", "reviewed_on", "reviewing", "re_reviewed", "requested_review", "requests_review", "requested_review_of", "requested_review_from", "requests_review_from", "requested_changes", "returned_changes_on", "posted_review", "posted_review_to", "posted_findings", "records_findings_for", "read_thread", "assigned_reviewer", "cross_reviewed")
	add(RelReviews, true, "reviewed_by", "reviewer", "review_of")
	add(RelApproves, false, "approves", "approve", "approved", "approved_for", "accepted", "accepts", "accepted_contract", "authorized", "authorizes", "authorized_by_flip", "allowed", "allows", "permits", "has_user_authorization", "confirmed_by_flip", "credits", "signed_off", "requested_approval", "requested_approval_for")
	add(RelApproves, true, "approved_by", "accepted_by", "authorized_by", "allowed_by")
	add(RelDecided, false, "decided", "decides", "decide", "decision", "chose", "chosen", "selected", "selects", "selected_for", "determined", "concluded", "prefers", "prioritizes", "recommends", "recommended", "adopted", "committed_to_flip", "rejected", "rejects", "refuses", "declined", "established", "directed", "instructs", "prescribes_workflow", "states", "stated", "says", "asserts", "claims_that", "emphasizes", "plans_to", "proposes", "proposed", "proposed_for", "proposed_contract_to", "proposed_contract_for", "posted_contract", "specifies_contract", "locks_invariant", "scoped_out", "deferred_flip")
	add(RelReplacedBy, false, "replaced_by", "superseded_by", "renamed_to", "moved_to", "migrated_to", "converted_to", "changed_to", "changed_to_use", "folded_onto", "relabeled", "rolled_into_main", "consolidates_into", "deprecated_by", "succeeded_by")
	add(RelReplacedBy, true, "replaces", "replace", "replaced", "supersedes", "supersede", "overrides", "overwrote", "renamed", "consolidates", "absorbs", "deduplicates", "duplicates", "predates", "precedes", "ahead_of", "is_ahead_of", "is_ancestor_of", "revised_before", "branches_from", "is_worktree_of", "diffed_against")
	add(RelMergedInto, false, "merged_into", "merged_to", "merged", "merges", "merges_into", "merged_with", "merges_with", "merged_on", "merged_as", "merges_cleanly", "merges_cleanly_with", "merges_cleanly_into", "committed", "committed_to", "committed_as", "committed_on", "committed_at", "committed_in", "commits", "contains_commit", "has_commit", "has_commits", "deployed_commit", "pushed", "pushed_to", "landed_in", "landed_on", "shipped_via_pr", "submitted", "submitted_to", "submits_to", "exists_on_branch", "on_branch", "has_branch", "has_merged", "split_across", "appended_to", "added_to", "added_in", "rolled_into", "publishes", "publishes_to", "published", "exported_from")
	add(RelMergedInto, true, "merged_by", "committed_by", "pushed_by")
	add(RelDeployedOn, false, "deployed_on", "deployed", "installed", "deployed_to", "deploys_to", "deploys_on", "deploys", "deployed_with", "auto_deploys_on", "installed_on", "installed_at", "relaunched_on", "restarted_on", "launched_from", "launched_on", "unmounted_on", "already_built_on", "partially_built_on", "is_implemented_on", "implemented_on", "active_on", "mounts")
	add(RelRunsOn, false, "runs_on", "running_on", "runs_in", "runs_at", "hosted_on", "hosted_by", "exists_on", "operates_in", "operates_on", "works_in", "working_in", "lives_on", "served_by", "scheduled_on", "configured_on")
	add(RelRunsOn, true, "hosts", "host", "serves_host", "runs_service", "has_service", "has_router", "has_engine", "has_cron_job", "spawns", "launches", "launched", "launched_by_flip")
	add(RelLocatedAt, false, "located_at", "located_in", "located_on", "stored_at", "stored_in", "stores_in", "lives_in", "exists_in", "found_in", "defined_in", "declared_in", "implemented_in", "modified_in", "registered_in", "registered_as", "pinned_in", "configured_in", "configured_at", "documented_in", "tracked_in", "reported_in", "duplicated_in", "created_in", "symlinked_to", "points_to", "points_at", "links_to", "maps_to", "resolved_to_flip", "has_dir", "has_file", "has_launch_point", "in_room", "installs_trigger", "has_route", "has_endpoint", "has_subpath_export", "provides_endpoint", "routes_to", "routes", "forwards_to", "sends_to", "delivers_to", "writes_to", "writes_into", "writes", "wrote", "appends", "saved", "stores", "stored", "persists", "records_to")
	add(RelPartOf, false, "part_of", "is_part_of", "belongs_to", "member_of", "included_in", "included_by", "scoped_to", "has_subproject_flip", "candidate_for", "candidate_for_porting", "source_for", "is_foundation_for", "involved_in", "contributed_to", "applies_to", "applied_to", "concerns", "relates_to", "associated_with", "is_reserved_for", "reserved_for", "created_for", "proposed_for_flip", "selected_for_flip", "used_for_flip")
	add(RelContains, false, "contains", "contain", "includes", "include", "has", "had", "has_feature", "has_field", "has_property", "has_part", "has_component", "has_method", "has_migration", "has_column", "has_columns", "has_schema", "has_table", "has_epic", "has_open_epics", "has_task", "has_tasks", "has_data", "has_structure", "has_surface", "has_seam", "has_api", "has_tool", "has_relation", "has_spec", "has_plan", "has_goal", "has_capability", "has_floor_capability", "has_constraint", "has_rule", "has_check_command", "has_config", "has_setting", "has_composite_fk", "has_unique_index", "has_og_image", "has_llm_explainer", "has_internal_links", "has_audit_files", "has_runs", "has_model", "has_subproject", "has_complete_feature", "has_attribute", "has_placeholder_for", "has_contract", "has_built", "defines", "defines_method", "defines_field", "declares", "holds", "carries", "now_carries", "consists_of", "organized_as", "includes_field", "includes_feature", "includes_validation", "now_includes", "now_includes_method", "will_contain", "will_add", "will_gain", "spans", "threads", "wraps", "encodes", "models", "describes", "depicts", "names", "keys_on", "indexes", "populates", "seeds", "instantiates", "creates_tables", "will_create_table", "installs", "registers", "binds", "wires", "wired_into_flip", "embeds", "bundles")
	add(RelContains, true, "contained_in", "included_in_flip", "wired_into", "embedded_in", "bundled_in")
	add(RelOwns, false, "owns", "own", "owned", "maintains", "manages", "governs", "controls", "coordinates", "orchestrates", "operates", "administers", "leads", "delegates_to_flip", "supplies", "authored", "created_by_flip", "designed", "sponsors", "runs_campaign", "offers", "offers_service", "sells")
	add(RelOwns, true, "owned_by", "maintained_by", "managed_by", "governed_by", "controlled_by", "operated_by", "authored_by", "created_by", "designed_by", "built_by", "generated_by", "produced_by", "written_by", "led_by", "run_by")
	add(RelAssignedTo, false, "assigned_to", "assigned", "assigns", "assigned_reviewer_flip", "delegated_to", "handed_off_to", "dispatched_to", "belongs_to_owner", "reports_to")
	add(RelAssignedTo, true, "claims", "claimed", "claimed_task", "claims_task", "attempted_claim", "requested_claim", "works_on", "worked_on", "working_on", "handles", "handled", "now_handles", "delegates_to", "dispatches", "dispatched", "assigned_by", "took", "owns_task", "picked_up")
	add(RelImplements, false, "implements", "implement", "implemented", "completed", "completed_on", "completed_at", "shipped", "shipped_on", "implemented_as", "implementation_of", "builds", "build", "built", "built_from", "realizes", "fulfills", "delivers", "delivered", "delivers_flip", "ships", "shipped_flip", "adds", "added", "adds_flip", "introduces_feature", "extends_with", "extended_with", "expanded_with", "exposes_feature", "now_supports", "supports", "enables", "enables_access_to", "provides_feature", "follows", "follows_pattern", "conforms_to", "reuses", "mirrors", "matches", "shares_base_with", "resolves_to")
	add(RelImplements, true, "implemented_by", "built_by_flip", "realized_by", "delivered_by", "shipped_by", "fulfilled_by", "supported_by", "enabled_by")
	add(RelProvides, false, "provides", "provide", "provided", "serves", "serve", "served", "exposes", "exports", "returns", "emits", "outputs", "yields", "displays", "renders", "shows", "surfaces", "presents", "answers", "handles_request", "accepts_input", "offers_flip", "gives", "grants", "responds_with", "passes_to", "hands", "supplies_flip", "sets", "set_to")
	add(RelProvides, true, "provided_by", "served_by_flip", "exposed_by", "exported_by", "returned_by", "emitted_by", "rendered_by", "displayed_by", "shown_by")
	add(RelProduces, false, "produces", "produce", "produced", "generates", "generate", "generated", "creates", "create", "created", "created_via", "created_from", "created_issue", "creates_flip", "builds_artifact", "emits_artifact", "writes_file", "captures", "captured", "derives_flip", "extracts", "extracts_from", "extracted_from", "computes", "compiles", "compiled", "packages", "packaged", "spawned", "initiated", "filed", "opened", "opens", "opens_files", "raised", "made", "makes", "results_in", "results_stored_on", "sent", "sends", "sent_email", "logs", "counts_flip", "copies", "copied", "converts", "split", "deleted_by_flip", "corrects_flip")
	add(RelProduces, true, "produced_by", "generated_by_flip", "created_by_produce", "output_of", "derived_by", "computed_by", "compiled_by", "packaged_by", "edited_by", "modified_by", "updated_by", "changed_by", "touched_by", "written_by_flip", "authored_by_flip", "filed_by", "raised_by", "opened_by", "made_by")
	add(RelDocuments, false, "documents", "document", "documented", "records", "record", "recorded", "tracks", "track", "tracked", "reports", "report", "reported", "reported_on", "reports_ready", "notes", "noted", "note", "describes_flip", "explains", "summarizes", "logs_flip", "captures_flip", "observed", "observes", "found", "finds", "identified", "identifies", "detected", "detects", "discovered", "discovers", "diagnosed_flip", "classifies", "classified_as", "categorizes", "labels", "marks", "marks_done", "measures_flip", "specifies", "specifies_flip", "declares_flip", "defines_flip", "outlines", "lists", "enumerates", "asked", "asked_about", "asks", "requested_feature", "requested", "requests", "requested_flip", "mentions", "mentioned", "refers_to", "quotes", "cites", "posted", "posts", "posted_to", "posts_to", "posted_status_to", "posted_findings_flip", "announced", "announces", "shared", "shares", "informs", "informed", "warns", "warned", "alerts_flip", "reviewed_flip")
	add(RelDocuments, true, "documented_by", "recorded_by", "tracked_by", "reported_by", "noted_by", "observed_by", "found_by", "identified_by", "detected_by", "discovered_by", "classified_by", "described_by", "mentioned_by", "posted_by", "announced_by")
	add(RelMonitors, false, "monitors", "monitor", "monitored", "watches", "watch", "watched", "observes_flip", "polls_flip", "probes", "probed", "pings", "checks_health_of", "alerts_on", "guards", "guards_against", "guards_with", "guarded_by_flip", "protects", "protects_flip", "protected_by_flip", "catches", "caught", "detects_fatal", "audits_flip", "supervises", "tracks_flip", "keeps", "preserves", "preserved", "maintains_flip", "retains", "backs_up", "backed_up", "schedules", "scheduled", "scheduled_for", "runs_at_flip", "polls")
	add(RelMonitors, true, "monitored_by", "watched_by", "probed_by", "guarded_by", "protected_by", "supervised_by", "scheduled_by", "preserved_by", "kept_by")
	add(RelConfigures, false, "configures", "configure", "configured", "configured_for", "configured_with", "configuration_of", "sets_flip", "set_to_flip", "pins_flip", "pinned", "tunes", "tuned", "parameterizes", "customizes", "customized", "installs_flip", "wires_flip", "registers_flip", "enables_flip", "disables", "disabled_flip", "toggles", "overrides_flip", "defaults_to_flip", "filters", "filters_on", "filters_by", "excludes_flip", "restricts", "limits", "throttles", "caps", "reserves", "allocates", "assigns_span_flip", "binds_flip", "maps", "routes_flip", "resolves_flip", "prefers_flip")
	add(RelConfigures, true, "configured_by", "tuned_by", "customized_by", "installed_by", "registered_by", "filtered_by", "restricted_by", "limited_by", "throttled_by", "capped_by", "mapped_by", "routed_by")
	add(RelCalls, false, "calls", "call", "called", "invokes", "invoke", "invoked", "runs", "ran", "run", "executes", "execute", "executed", "triggers_flip", "fires", "fired", "starts", "started", "start", "restarts", "restarted", "relaunches", "kicks", "kickstarts", "queries", "queried", "requests_from", "hits", "fetches_flip", "consults", "uses_tool", "shells_out_to", "spawns_flip", "dispatches_flip", "enqueues", "enqueued", "schedules_flip", "submits", "sends_request", "loads", "loaded", "reads_flip", "searches", "searched", "opens_flip", "syncs", "synced", "ingests", "ingested", "imports_flip", "processes", "processed", "handles_flip", "drives_flip", "brings_up", "brought_up", "stops", "stopped_flip", "kills", "killed")
	add(RelCalls, true, "called_by", "invoked_by", "run_by_flip", "executed_by", "triggered_by_flip", "fired_by", "started_by", "restarted_by", "queried_by", "hit_by", "loaded_by", "searched_by", "ingested_by", "processed_by", "handled_by", "driven_by", "stopped_by_flip", "killed_by")
	add(RelReferences, false, "references", "reference", "referenced", "refers", "refers_to_flip", "cites_flip", "quotes_flip", "mentions_flip", "links", "linked_to", "links_to_flip", "points_to_flip", "sees", "saw", "reads_flip2", "consults_flip", "checks_flip", "looks_at", "inspects", "compares", "compared", "compares_to", "compared_with", "diffed_against_flip", "equals_flip", "same_as_flip", "identical_to_flip", "relates", "related", "related_to_flip", "involves", "involved", "concerns_flip", "about", "regarding", "applies_flip", "touches_flip", "affects_flip", "impacts", "impacted", "impacts_flip", "aligns_with", "aligned_with", "corresponds_to", "correlates_with", "parallels", "resembles", "analogous_to", "similar_to", "like", "unlike", "contrasts_with")
	add(RelReferences, true, "referenced_by", "referenced_in", "cited_by", "quoted_by", "mentioned_in", "linked_from", "seen_by", "compared_by", "inspected_by", "looked_at_by")
	add(RelTargets, false, "targets", "target", "targeted", "aims_at", "aims", "aimed_at", "plans", "planned", "plan", "planned_to_run_on", "planned_for", "intends", "intended", "intended_for", "next_target", "goal", "has_goal_flip", "will_connect", "will_use_flip", "will_add_flip", "will_contain_flip", "will_create_table_flip", "will_gain_flip", "should_integrate_into", "should_use_flip", "recommended_for", "candidate_flip", "proposes_flip", "proposed_flip", "seeks", "sought", "pursues", "pursued", "explores", "explored", "considers", "considered", "evaluates", "evaluated_flip", "assesses", "assessed", "scopes", "scoped", "scoped_to_flip", "focuses_on", "focused_on", "prioritizes_flip", "designed_for", "built_for", "made_for", "meant_for", "optimized_for", "tailored_for", "suited_for", "fits", "serves_purpose", "purpose", "for", "toward", "towards")
	add(RelTargets, true, "targeted_by", "aimed_at_by", "planned_by", "intended_by", "sought_by", "pursued_by", "explored_by", "considered_by", "evaluated_by", "assessed_by", "scoped_by", "designed_for_flip", "built_for_flip")
	add(RelNotifies, false, "notifies", "notify", "notified", "alerts", "alert", "alerted", "alerts_on_flip", "pings_flip", "messages", "messaged", "emails", "emailed", "sent_email_flip", "texts", "texted", "calls_user", "pages", "paged", "escalates", "escalated", "escalates_to", "reports_to_flip", "informs_flip", "warns_flip", "signals", "signaled", "signals_flip", "broadcasts", "broadcast", "announces_flip", "publishes_flip", "posts_flip", "delivers_alert", "delivered_alert", "delivered_flip", "forwards", "forwarded", "forwards_to_flip", "relays", "relayed", "routes_alert", "routed_alert", "sends_flip", "sent_flip", "dispatches_alert", "dispatched_alert")
	add(RelNotifies, true, "notified_by", "alerted_by", "messaged_by", "emailed_by", "texted_by", "paged_by", "escalated_by", "signaled_by", "broadcast_by", "forwarded_by", "relayed_by", "informed_by", "warned_by")
	add(RelEnforces, false, "enforces", "enforce", "enforced", "enforces_constraint", "now_enforces", "requires_flip", "mandates_flip", "constrains_flip", "restricts_flip", "governs_flip", "controls_flip", "guards_flip", "protects_flip2", "validates_flip", "checks_flip2", "verifies_flip", "asserts_flip", "guarantees", "guaranteed", "ensures", "ensured", "assures", "assured", "maintains_invariant", "locks", "locked", "locked_by", "locks_invariant_flip", "freezes_flip", "frozen_flip", "pins_flip2", "pinned_flip", "bans", "bans_string", "banned", "forbids", "forbade", "forbidden", "prohibits", "prohibits_use_of", "prohibited", "disallows", "disallowed", "blocks_use_of", "blocks_flip", "prevents_flip", "denies", "denied", "refuses_flip", "rejects_flip", "rejected_flip", "must_not_edit", "must_not", "never", "never_flip", "cannot", "must", "must_flip", "should", "should_flip", "policy", "rule", "has_rule_flip", "invariant", "constraint", "has_constraint_flip")
	add(RelEnforces, true, "enforced_by", "guaranteed_by", "ensured_by", "assured_by", "locked_by_flip", "banned_by", "forbidden_by", "prohibited_by", "disallowed_by", "denied_by", "refused_by", "rejected_by", "constrained_by", "governed_by_flip", "controlled_by_flip", "restricted_by_flip", "mandated_by", "required_by_flip")
	add(RelExcludes, false, "excludes", "exclude", "excluded", "skips", "ignores", "avoids", "avoided", "bypasses", "excludes_from", "excluded_from", "omits_flip", "omitted", "skips_flip", "skipped", "ignores_flip", "ignored", "avoids_flip", "bypasses_flip", "bypassed", "drops_flip", "dropped_flip", "removes_flip", "removed_flip", "strips", "stripped", "filters_out", "filtered_out", "leaves_out", "left_out", "does_not_include", "does_not_cover_flip", "does_not_test_flip", "does_not_use_flip", "does_not_modify_flip", "does_not_apply_flip", "does_not_support_flip", "untouched_by", "unaffected_by", "not_in", "not_part_of", "outside", "outside_of", "beyond", "separates", "separated", "separates_flip", "isolates", "isolated", "isolates_flip", "quarantines", "quarantined", "disables_flip", "disabled_flip2", "turns_off", "turned_off", "unsets", "unset", "clears", "cleared", "resets", "reset", "reverts_flip", "reverted_flip", "rolls_back", "rolled_back", "undoes", "undid", "unmerged_flip", "unmounted", "unmounts", "uninstalls", "uninstalled", "unregisters", "unregistered", "unlinks", "unlinked", "unbinds", "unbound", "unwires", "unwired_flip", "detaches", "detached", "disconnects", "disconnected", "abandons", "abandoned_flip", "defers_flip", "deferred_flip2", "postpones", "postponed", "delays", "delayed", "pauses", "paused", "suspends", "suspended", "halts", "halted", "stops_flip", "stopped_flip2", "terminates", "terminated", "ends", "ended", "closes", "closed_flip", "archives", "archived_flip", "retires", "retired", "sunsets", "sunset", "decommissions", "decommissioned", "deprecates", "deprecated", "obsoletes", "obsoleted")
	add(RelExcludes, true, "excluded_by", "omitted_by", "skipped_by", "ignored_by", "avoided_by", "bypassed_by", "dropped_by", "removed_by", "stripped_by", "filtered_out_by", "left_out_by", "isolated_by", "quarantined_by", "disabled_by", "turned_off_by", "unset_by", "cleared_by", "reset_by", "reverted_by", "rolled_back_by", "undone_by", "unmounted_by", "uninstalled_by", "unregistered_by", "unlinked_by", "unbound_by", "unwired_by", "detached_by", "disconnected_by", "abandoned_by", "deferred_by", "postponed_by", "delayed_by", "paused_by", "suspended_by", "halted_by", "terminated_by", "ended_by", "closed_by", "archived_by", "retired_by", "sunset_by", "decommissioned_by", "deprecated_by_flip", "obsoleted_by")
	add(RelModifies, false, "modifies", "modify", "modified", "changes", "changed", "updates", "updated", "updated_to", "updated_for", "updated_with", "updated_signature", "edits", "edited", "edited_by_implementer", "touches", "reworks", "refactors", "refactored", "rewrites", "removes", "removed", "removed_from", "removed_content", "deletes", "deleted", "deleted_from", "drops", "dropped", "modified_flip", "modifies_flip2", "updates_flip", "updated_flip", "changes_flip", "changed_flip", "edits_flip", "edited_flip", "touches_flip2", "alters", "altered", "adjusts", "adjusted", "tweaks", "tweaked", "revises", "revised", "amends", "amended", "patches", "patched_flip", "refactors_flip", "refactored_flip", "rewrites_flip", "rewrote", "rewritten", "reworks_flip", "reworked", "restructures", "restructured", "reorganizes", "reorganized", "renames", "renamed_flip", "moves", "moved", "moved_flip", "relocates", "relocated", "migrates", "migrated", "upgrades", "upgraded", "downgrades", "downgraded", "bumps", "bumped", "extends_flip", "extended_flip", "expands", "expanded", "expanded_flip", "grows", "grew", "shrinks", "shrank", "trims", "trimmed", "prunes", "pruned", "cleans", "cleaned", "tidies", "tidied", "formats", "formatted", "reformats", "reformatted", "lints", "linted", "fixes_style", "styles", "styled", "polishes", "polished", "hardens", "hardened", "secures", "secured", "optimizes", "optimized", "speeds_up", "sped_up", "accelerates", "accelerated", "simplifies", "simplified", "clarifies", "clarified", "documents_change", "adds_to", "appends_to", "prepends_to", "inserts_into", "inserted_into", "injects_into", "injected_into", "wraps_flip", "wrapped", "unwraps", "unwrapped", "splits", "split_flip", "joins", "joined", "combines", "combined", "unifies", "unified", "normalizes", "normalized", "canonicalizes", "canonicalized", "dedupes", "deduped", "deduplicates_flip", "sorts", "sorted", "orders", "ordered", "reorders", "reordered", "groups", "grouped", "regroups", "regrouped", "flattens", "flattened", "nests", "nested", "batches", "batched", "chunks", "chunked", "paginates", "paginated", "caches", "cached", "memoizes", "memoized", "compresses", "compressed", "decompresses", "decompressed", "encrypts", "encrypted", "decrypts", "decrypted", "hashes", "hashed", "signs", "signed", "verifies_signature", "encodes_flip", "encoded", "decodes", "decoded", "serializes", "serialized", "deserializes", "deserialized", "parses", "parsed", "tokenizes", "tokenized", "renders_flip", "rendered", "compiles_flip", "compiled_flip", "transpiles", "transpiled", "bundles_flip", "bundled", "minifies", "minified", "obfuscates", "obfuscated")
	add(RelModifies, true, "modified_by_flip", "updated_by_flip", "changed_by_flip", "edited_by_flip", "touched_by_flip", "altered_by", "adjusted_by", "tweaked_by", "revised_by", "amended_by", "patched_by_flip", "refactored_by", "rewritten_by", "reworked_by", "restructured_by", "reorganized_by", "renamed_by", "moved_by", "relocated_by", "migrated_by", "upgraded_by", "downgraded_by", "bumped_by", "extended_by", "expanded_by", "trimmed_by", "pruned_by", "cleaned_by", "formatted_by", "linted_by", "polished_by", "hardened_by", "secured_by", "optimized_by", "simplified_by", "clarified_by")
	add(RelSameAs, false, "same_as", "equals", "equal_to", "is_identical_to", "identical_to", "identical", "alias_of", "aliases", "aka", "also_known_as", "synonym_of", "synonymous_with", "equivalent_to", "equivalent", "is_same_as", "duplicate_of", "copy_of", "clone_of", "mirror_of", "matches_flip", "resolves_to_same", "maps_to_same", "renamed_from", "formerly", "formerly_known_as", "previously", "previously_known_as", "was", "was_called", "now_called", "known_as", "called")
	// A handful of raw names collide with flip variants above only because
	// the table was assembled from observed spellings; the last write wins,
	// and every flip-only name carrying a "_flip"/"_flip2" suffix is a
	// placeholder that no raw relation matches. Drop them.
	for k := range t {
		if strings.HasSuffix(k, "_flip") || strings.HasSuffix(k, "_flip2") || strings.HasSuffix(k, "_produce") || strings.HasSuffix(k, "_test") && k != "has_test" {
			delete(t, k)
		}
	}
	// Canonical names map to themselves without flipping, whatever the
	// table above said.
	for _, r := range Canonical {
		t[r] = m(r)
	}
	return t
}()

// prepositions are trailing tokens that name where or how rather than
// what: "deployed_to" is "deployed" with a preposition, not a new verb.
var prepositions = map[string]bool{
	"on": true, "at": true, "in": true, "to": true, "for": true, "with": true, "from": true,
	"into": true, "of": true, "via": true, "against": true, "through": true, "onto": true,
	"under": true, "over": true, "by": true, "as": true, "across": true, "within": true,
}

// stems maps a verb stem (the token with common suffixes removed) to its
// canonical relation. It is consulted after the exact table, for raw
// relations the table never saw ("verifying", "deployment").
var stems = map[string]mapping{
	"deploy": m(RelDeployedOn), "install": m(RelDeployedOn), "launch": m(RelDeployedOn),
	"host": inv(RelRunsOn), "run": m(RelCalls), "serv": m(RelProvides),
	"us": m(RelUses), "utiliz": m(RelUses), "consum": m(RelUses), "import": m(RelUses), "read": m(RelUses),
	"depend": m(RelDependsOn), "rel": m(RelDependsOn),
	"requir": m(RelRequires), "need": m(RelRequires),
	"lack": m(RelLacks), "miss": m(RelLacks),
	"block": inv(RelBlockedBy), "gat": inv(RelBlockedBy), "constrain": inv(RelBlockedBy),
	"conflict": m(RelConflictsWith), "contradict": m(RelConflictsWith), "violat": m(RelConflictsWith), "diverg": m(RelConflictsWith),
	"caus": m(RelCauses), "affect": m(RelCauses), "trigger": m(RelCauses), "fail": m(RelCauses), "throw": m(RelCauses), "break": m(RelCauses), "introduc": m(RelCauses),
	"fix": m(RelFixes), "repair": m(RelFixes), "resolv": m(RelFixes), "correct": m(RelFixes), "patch": m(RelFixes), "address": m(RelFixes), "solv": m(RelFixes), "restor": m(RelFixes), "revert": m(RelFixes),
	"test": m(RelTests), "verif": m(RelTests), "validat": m(RelTests), "check": m(RelTests), "cover": m(RelTests), "audit": m(RelTests), "prov": m(RelTests), "confirm": m(RelTests), "inspect": m(RelTests), "investigat": m(RelTests), "reproduc": m(RelTests), "diagnos": m(RelTests), "scan": m(RelTests), "measur": m(RelTests), "evaluat": m(RelTests), "assess": m(RelTests), "exercis": m(RelTests),
	"pass": m(RelPasses), "satisf": m(RelPasses), "achiev": m(RelPasses), "meet": m(RelPasses),
	"review": m(RelReviews),
	"approv": m(RelApproves), "accept": m(RelApproves), "authoriz": m(RelApproves), "allow": m(RelApproves), "permit": m(RelApproves), "grant": m(RelApproves),
	"decid": m(RelDecided), "choos": m(RelDecided), "chos": m(RelDecided), "select": m(RelDecided), "determin": m(RelDecided), "conclud": m(RelDecided), "prefer": m(RelDecided), "recommend": m(RelDecided), "adopt": m(RelDecided), "reject": m(RelDecided), "propos": m(RelDecided), "agre": m(RelDecided), "settl": m(RelDecided), "commit": m(RelMergedInto),
	"replac": inv(RelReplacedBy), "supersed": inv(RelReplacedBy), "overrid": inv(RelReplacedBy), "renam": m(RelReplacedBy), "migrat": m(RelReplacedBy),
	"merg": m(RelMergedInto), "push": m(RelMergedInto), "land": m(RelMergedInto), "submit": m(RelMergedInto), "publish": m(RelMergedInto),
	"locat": m(RelLocatedAt), "stor": m(RelLocatedAt), "liv": m(RelLocatedAt), "exist": m(RelLocatedAt), "resid": m(RelLocatedAt), "sav": m(RelLocatedAt), "persist": m(RelLocatedAt), "writ": m(RelLocatedAt), "wrot": m(RelLocatedAt), "point": m(RelLocatedAt), "link": m(RelLocatedAt), "rout": m(RelLocatedAt), "forward": m(RelLocatedAt),
	"belong": m(RelPartOf), "member": m(RelPartOf),
	"contain": m(RelContains), "includ": m(RelContains), "hold": m(RelContains), "carr": m(RelContains), "compris": m(RelContains), "consist": m(RelContains), "defin": m(RelContains), "declar": m(RelContains), "embed": m(RelContains), "bundl": m(RelContains), "wrap": m(RelContains), "expos": m(RelProvides),
	"own": m(RelOwns), "maintain": m(RelOwns), "manag": m(RelOwns), "govern": m(RelOwns), "control": m(RelOwns), "coordinat": m(RelOwns), "orchestrat": m(RelOwns), "operat": m(RelOwns), "administ": m(RelOwns), "lead": m(RelOwns), "author": m(RelOwns), "design": m(RelOwns),
	"assign": m(RelAssignedTo), "delegat": inv(RelAssignedTo), "claim": inv(RelAssignedTo), "work": inv(RelAssignedTo), "handl": inv(RelAssignedTo), "dispatch": inv(RelAssignedTo),
	"implement": m(RelImplements), "build": m(RelImplements), "built": m(RelImplements), "realiz": m(RelImplements), "fulfil": m(RelImplements), "deliver": m(RelImplements), "ship": m(RelImplements), "add": m(RelImplements), "support": m(RelImplements), "enabl": m(RelImplements), "follow": m(RelImplements), "conform": m(RelImplements), "reus": m(RelImplements), "mirror": m(RelImplements), "match": m(RelImplements),
	"provid": m(RelProvides), "export": m(RelProvides), "return": m(RelProvides), "emit": m(RelProvides), "output": m(RelProvides), "yield": m(RelProvides), "display": m(RelProvides), "render": m(RelProvides), "show": m(RelProvides), "surfac": m(RelProvides), "present": m(RelProvides), "answer": m(RelProvides), "giv": m(RelProvides), "offer": m(RelProvides), "suppl": m(RelProvides), "set": m(RelProvides),
	"produc": m(RelProduces), "generat": m(RelProduces), "creat": m(RelProduces), "captur": m(RelProduces), "extract": m(RelProduces), "comput": m(RelProduces), "compil": m(RelProduces), "packag": m(RelProduces), "spawn": m(RelProduces), "initiat": m(RelProduces), "fil": m(RelProduces), "open": m(RelProduces), "rais": m(RelProduces), "mak": m(RelProduces), "mad": m(RelProduces), "send": m(RelProduces), "sent": m(RelProduces), "log": m(RelProduces), "cop": m(RelProduces), "convert": m(RelProduces), "delet": m(RelProduces), "remov": m(RelProduces), "drop": m(RelProduces), "refactor": m(RelModifies), "rewrit": m(RelModifies), "edit": m(RelModifies), "modif": m(RelModifies), "chang": m(RelModifies), "updat": m(RelModifies), "touch": m(RelModifies), "alter": m(RelModifies), "adjust": m(RelModifies), "tweak": m(RelModifies), "revis": m(RelModifies), "amend": m(RelModifies), "extend": m(RelModifies), "expand": m(RelModifies), "mov": m(RelModifies), "upgrad": m(RelModifies), "bump": m(RelModifies), "trim": m(RelModifies), "prun": m(RelModifies), "clean": m(RelModifies), "format": m(RelModifies), "lint": m(RelModifies), "polish": m(RelModifies), "harden": m(RelModifies), "secur": m(RelModifies), "optimiz": m(RelModifies), "simplif": m(RelModifies), "clarif": m(RelModifies), "split": m(RelModifies), "join": m(RelModifies), "combin": m(RelModifies), "unif": m(RelModifies), "normaliz": m(RelModifies), "dedup": m(RelModifies), "sort": m(RelModifies), "order": m(RelModifies), "group": m(RelModifies), "flatten": m(RelModifies), "nest": m(RelModifies), "batch": m(RelModifies), "chunk": m(RelModifies), "paginat": m(RelModifies), "cach": m(RelModifies), "compress": m(RelModifies), "encrypt": m(RelModifies), "decrypt": m(RelModifies), "hash": m(RelModifies), "sign": m(RelModifies), "encod": m(RelModifies), "decod": m(RelModifies), "serializ": m(RelModifies), "pars": m(RelModifies), "tokeniz": m(RelModifies), "transpil": m(RelModifies), "minif": m(RelModifies),
	"document": m(RelDocuments), "record": m(RelDocuments), "track": m(RelDocuments), "report": m(RelDocuments), "not": m(RelDocuments), "explain": m(RelDocuments), "summariz": m(RelDocuments), "observ": m(RelDocuments), "find": m(RelDocuments), "found": m(RelDocuments), "identif": m(RelDocuments), "detect": m(RelDocuments), "discover": m(RelDocuments), "classif": m(RelDocuments), "categoriz": m(RelDocuments), "label": m(RelDocuments), "mark": m(RelDocuments), "specif": m(RelDocuments), "outlin": m(RelDocuments), "list": m(RelDocuments), "enumerat": m(RelDocuments), "ask": m(RelDocuments), "request": m(RelDocuments), "mention": m(RelDocuments), "quot": m(RelDocuments), "cit": m(RelDocuments), "post": m(RelDocuments), "announc": m(RelDocuments), "shar": m(RelDocuments), "inform": m(RelDocuments), "warn": m(RelDocuments), "describ": m(RelDocuments), "flag": m(RelHasIssue),
	"monitor": m(RelMonitors), "watch": m(RelMonitors), "prob": m(RelMonitors), "ping": m(RelMonitors), "guard": m(RelMonitors), "protect": m(RelMonitors), "catch": m(RelMonitors), "supervis": m(RelMonitors), "keep": m(RelMonitors), "preserv": m(RelMonitors), "retain": m(RelMonitors), "schedul": m(RelMonitors), "poll": m(RelMonitors),
	"configur": m(RelConfigures), "tun": m(RelConfigures), "parameteriz": m(RelConfigures), "customiz": m(RelConfigures), "toggl": m(RelConfigures), "filter": m(RelConfigures), "restrict": m(RelConfigures), "limit": m(RelConfigures), "throttl": m(RelConfigures), "cap": m(RelConfigures), "reserv": m(RelConfigures), "allocat": m(RelConfigures), "bind": m(RelConfigures), "map": m(RelConfigures), "regist": m(RelConfigures), "wir": m(RelConfigures), "pin": m(RelConfigures),
	"call": m(RelCalls), "invok": m(RelCalls), "execut": m(RelCalls), "fir": m(RelCalls), "start": m(RelCalls), "restart": m(RelCalls), "kick": m(RelCalls), "quer": m(RelCalls), "hit": m(RelCalls), "consult": m(RelCalls), "enqueu": m(RelCalls), "load": m(RelCalls), "search": m(RelCalls), "sync": m(RelCalls), "ingest": m(RelCalls), "process": m(RelCalls), "driv": m(RelCalls), "stop": m(RelCalls), "kill": m(RelCalls),
	"referenc": m(RelReferences), "refer": m(RelReferences), "se": m(RelReferences), "look": m(RelReferences), "compar": m(RelReferences), "relat": m(RelReferences), "involv": m(RelReferences), "concern": m(RelReferences), "impact": m(RelReferences), "align": m(RelReferences), "correspond": m(RelReferences), "correlat": m(RelReferences), "parallel": m(RelReferences), "resembl": m(RelReferences),
	"target": m(RelTargets), "aim": m(RelTargets), "plan": m(RelTargets), "intend": m(RelTargets), "seek": m(RelTargets), "pursu": m(RelTargets), "explor": m(RelTargets), "consider": m(RelTargets), "scop": m(RelTargets), "focus": m(RelTargets), "prioritiz": m(RelTargets),
	"notif": m(RelNotifies), "alert": m(RelNotifies), "messag": m(RelNotifies), "email": m(RelNotifies), "text": m(RelNotifies), "pag": m(RelNotifies), "escalat": m(RelNotifies), "signal": m(RelNotifies), "broadcast": m(RelNotifies), "relay": m(RelNotifies),
	"enforc": m(RelEnforces), "mandat": m(RelEnforces), "guarante": m(RelEnforces), "ensur": m(RelEnforces), "assur": m(RelEnforces), "lock": m(RelEnforces), "freez": m(RelEnforces), "froz": m(RelEnforces), "ban": m(RelEnforces), "forbid": m(RelEnforces), "prohibit": m(RelEnforces), "disallow": m(RelEnforces), "den": m(RelEnforces), "refus": m(RelEnforces), "prevent": m(RelEnforces), "assert": m(RelEnforces),
	"exclud": m(RelExcludes), "omit": m(RelExcludes), "skip": m(RelExcludes), "ignor": m(RelExcludes), "avoid": m(RelExcludes), "bypass": m(RelExcludes), "strip": m(RelExcludes), "isolat": m(RelExcludes), "quarantin": m(RelExcludes), "disabl": m(RelExcludes), "unset": m(RelExcludes), "clear": m(RelExcludes), "reset": m(RelExcludes), "roll": m(RelExcludes), "undo": m(RelExcludes), "unmount": m(RelExcludes), "uninstall": m(RelExcludes), "unregist": m(RelExcludes), "unlink": m(RelExcludes), "unbind": m(RelExcludes), "unwir": m(RelExcludes), "detach": m(RelExcludes), "disconnect": m(RelExcludes), "abandon": m(RelExcludes), "defer": m(RelExcludes), "postpon": m(RelExcludes), "delay": m(RelExcludes), "paus": m(RelExcludes), "suspend": m(RelExcludes), "halt": m(RelExcludes), "terminat": m(RelExcludes), "end": m(RelExcludes), "clos": m(RelExcludes), "archiv": m(RelExcludes), "retir": m(RelExcludes), "sunset": m(RelExcludes), "decommission": m(RelExcludes), "deprecat": m(RelExcludes), "obsolet": m(RelExcludes), "separat": m(RelExcludes),
	"equal": m(RelSameAs), "alias": m(RelSameAs), "synonym": m(RelSameAs), "equival": m(RelSameAs), "duplicat": m(RelSameAs), "clon": m(RelSameAs), "identical": m(RelSameAs),
}

// negationPrefixes turn a verb into its absence.
var negationPrefixes = []string{"does_not_", "did_not_", "do_not_", "not_", "never_", "no_", "un", "is_not_", "was_not_", "cannot_", "can_not_", "wont_", "isnt_", "doesnt_"}

// tensePrefixes say when, not what.
var tensePrefixes = []string{"now_", "still_", "already_", "will_", "was_", "were_", "is_", "are_", "has_been_", "have_been_", "had_", "should_", "must_", "may_", "can_", "could_", "would_", "currently_", "previously_", "formerly_", "recently_", "newly_", "just_", "auto_", "re_"}

// Map returns the canonical relation for a raw (already normalized: lower
// snake_case) relation, and whether src and dst must swap. Unknown
// relations land on Fallback without a flip.
func Map(raw string) (rel string, flip bool) {
	raw = strings.Trim(strings.ToLower(strings.TrimSpace(raw)), "_")
	if raw == "" {
		return Fallback, false
	}
	if mp, ok := synonyms[raw]; ok {
		return mp.rel, mp.flip
	}
	// Negation: "does_not_use", "never_writes" → lacks. Negated forms of
	// canonical relations keep their subject: "does_not_depend_on" is still
	// about what X lacks.
	for _, p := range negationPrefixes {
		if strings.HasPrefix(raw, p) && len(raw) > len(p) && (p != "un" || strings.HasPrefix(raw, "un_")) {
			return RelLacks, false
		}
	}
	// Tense and modality prefixes carry no relation.
	for _, p := range tensePrefixes {
		if strings.HasPrefix(raw, p) && len(raw) > len(p) {
			return Map(raw[len(p):])
		}
	}
	tokens := strings.Split(raw, "_")
	// "has_<noun>" is containment unless the noun says otherwise.
	if tokens[0] == "has" || tokens[0] == "have" || tokens[0] == "had" {
		if len(tokens) == 1 {
			return RelContains, false
		}
		noun := strings.Join(tokens[1:], "_")
		if mp, ok := synonyms["has_"+noun]; ok {
			return mp.rel, mp.flip
		}
		if issueNouns[tokens[1]] {
			return RelHasIssue, false
		}
		if valueNouns[tokens[1]] || strings.HasSuffix(noun, "_count") || strings.HasSuffix(noun, "_size") || strings.HasSuffix(noun, "_price") {
			return RelStatus, false
		}
		return RelContains, false
	}
	// Trailing "_by" is the passive voice: "verified_by" is tests, flipped.
	if n := len(tokens); n >= 2 && tokens[n-1] == "by" {
		rel, flip := Map(strings.Join(tokens[:n-1], "_"))
		if rel == Fallback {
			return Fallback, false
		}
		return rel, !flip
	}
	// A trailing preposition names where or how; the verb still decides.
	if n := len(tokens); n >= 2 && prepositions[tokens[n-1]] {
		verb := strings.Join(tokens[:n-1], "_")
		if verb == "stored" || verb == "located" || verb == "lives" || verb == "exists" || verb == "found" || verb == "defined" || verb == "kept" || verb == "saved" || verb == "recorded" {
			return RelLocatedAt, false
		}
		if verb == "installed" || verb == "deployed" || verb == "running" || verb == "runs" || verb == "hosted" || verb == "mounted" || verb == "launched" {
			if tokens[n-1] == "on" || tokens[n-1] == "at" || tokens[n-1] == "in" || tokens[n-1] == "to" {
				if verb == "running" || verb == "runs" || verb == "hosted" {
					return RelRunsOn, false
				}
				return RelDeployedOn, false
			}
		}
		if rel, flip := Map(verb); rel != Fallback {
			return rel, flip
		}
	}
	// Verb stem on the first token, then on the second for forms like
	// "is_using" where the first token carries no verb.
	for _, tok := range tokens[:min(2, len(tokens))] {
		if mp, ok := lookupStem(tok); ok {
			return mp.rel, mp.flip
		}
	}
	return Fallback, false
}

// lookupStem finds the stem table entry for a token, tolerating the "y"
// that survives stripping "ing" from "verifying".
func lookupStem(tok string) (mapping, bool) {
	s := stem(tok)
	if mp, ok := stems[s]; ok {
		return mp, true
	}
	if strings.HasSuffix(s, "y") {
		if mp, ok := stems[strings.TrimSuffix(s, "y")]; ok {
			return mp, true
		}
	}
	return mapping{}, false
}

// issueNouns are "has_<noun>" objects that mean a problem.
var issueNouns = map[string]bool{
	"bug": true, "bugs": true, "defect": true, "defects": true, "issue": true, "issues": true,
	"error": true, "errors": true, "problem": true, "problems": true, "gap": true, "gaps": true,
	"finding": true, "findings": true, "risk": true, "risks": true, "limitation": true,
	"regression": true, "regressions": true, "failure": true, "failures": true, "warning": true,
	"warnings": true, "smell": true, "smells": true, "vulnerability": true, "vulnerabilities": true,
	"weakness": true, "flaw": true, "flaws": true, "concern": true, "concerns": true,
}

// valueNouns are "has_<noun>" objects that are measurements or states, so
// the fact is an attribute rather than an edge.
var valueNouns = map[string]bool{
	"status": true, "state": true, "version": true, "size": true, "count": true, "price": true,
	"cost": true, "speed": true, "latency": true, "throughput": true, "capacity": true, "memory": true,
	"parameters": true, "score": true, "rating": true, "grade": true, "level": true, "priority": true,
	"severity": true, "duration": true, "age": true, "ttl": true, "quota": true, "budget": true,
	"limit": true, "threshold": true, "percent": true, "percentage": true, "ratio": true, "rate": true,
	"median": true, "mean": true, "p50": true, "p95": true, "p99": true, "uptime": true, "coverage": true,
	"points": true, "credit": true, "credits": true, "balance": true, "value": true, "metric": true,
	"metrics": true, "progress": true, "outcome": true, "verdict": true, "result": true, "health": true,
	"length": true, "width": true, "height": true, "weight": true, "temperature": true,
}

// stem strips common English verb suffixes so "verifies", "verified",
// "verifying", and "verification" all reach "verif".
func stem(tok string) string {
	for _, suf := range []string{"ication", "ations", "ation", "ments", "ment", "ings", "ing", "ies", "ied", "ers", "er", "ed", "es", "s", "e"} {
		if strings.HasSuffix(tok, suf) && len(tok)-len(suf) >= 2 {
			return tok[:len(tok)-len(suf)]
		}
	}
	return tok
}

// SynonymCount is the size of the exact table, for the vocabulary tests.
func SynonymCount() int { return len(synonyms) }

// SortedCanonical returns Canonical sorted, for reports.
func SortedCanonical() []string {
	out := append([]string(nil), Canonical...)
	sort.Strings(out)
	return out
}
