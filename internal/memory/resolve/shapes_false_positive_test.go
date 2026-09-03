package resolve

import "testing"

// The false-positive side, from a grader that probed plausible names
// rather than names already in the store. A rule that rejects real
// things destroys the graph, and these fifteen were being rejected: a
// trunk word anywhere in a name made it a branch, and a name opening
// with a state word made it a status.
func TestPlausibleNamesSurvive(t *testing.T) {
	identities := []string{
		"main-thread-scheduler", "master-key-rotation", "main-street-partners",
		"main-line-billing", "trunk-line-router", "develop-docs-site",
		"develop-mode-toggle", "develop-branch-guard", "trunk-based-development",
		"master-detail-view",
		"pending-payments-api", "ready-check-endpoint", "blocked-domains-list",
		"blocked-user-repository", "waiting_room_ui", "awaiting_signature_queue",
		"needs-assessment-form", "pending_invites_table", "needs_analysis_report",
		"ready_player_service", "blocked_by_id",
	}
	var wrong []string
	for _, id := range identities {
		if NotAnIdentity(id) {
			wrong = append(wrong, id)
		}
	}
	if len(wrong) > 0 {
		t.Errorf("rejected %d/%d real names: %q", len(wrong), len(identities), wrong)
	}
	values := []string{"main trunk", "docket-main", "ph-develop", "staging-develop", "prod-main",
		"ux/onboarding-polish", "seo/resume-page"}
	var missed []string
	for _, v := range values {
		if !NotAnIdentity(v) {
			missed = append(missed, v)
		}
	}
	if len(missed) > 0 {
		t.Errorf("accepted %d branches: %q", len(missed), missed)
	}
}
