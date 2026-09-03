package resolve

import "testing"

// The false-positive side, measured the only way that is not circular:
// on names chosen independently of the store. A grader pointed out that
// every earlier list was drawn from or checked against the graph, which
// cannot hold a name the rules reject, so a 99 per cent survival rate
// measured that way meant nothing. On its independent names the rules
// rejected 37 of 56 real ones, including eleven of twelve real
// directories from these repositories.
func TestRealNamesSurviveTheValueRules(t *testing.T) {
	real := []string{
		"terraform/modules", "k8s/overlays", "helm/charts", "fixtures/users",
		"locales/en-GB", "proto/billing", "charts/api", "queries/reports",
		"seeds/tenants", "contracts/erc20", "themes/dark", "mobile/ios",
		"infra/network", "ansible/roles", "notebooks/eda", "grafana/dashboards",
		"screens/failure-reasons", "grounding/domain-research", "operations/task-state",
		"comparison/decisions", "interview/question-strategy", "e2e/visual",
		"specs/2026-redesign-brief", "ai/failure-modes", "solver/test_domain",
		"mobile/testing-scripts", "paperwork/day-checklist", "launchd/com.jhoot.scryd",
		"deferred-revenue-ledger", "approved-vendor-registry", "rejected-claims-dashboard",
		"merged-tenant-directory", "cancelled-order-webhook", "failed-payment-retrier",
		"resolved-ticket-archive", "verified-email-domain", "deferred-tax-asset",
		"escalated-incident-timeline", "triaged-defect-backlog", "reopened-case-metric",
		"abandoned-mine-dataset", "pending-migration-lock", "passed-inspection-record",
		"completed-lesson-tracker", "blocked-sender-registry", "failed-login-counter",
		"trade-off", "hand-off", "sign-off", "kick-off", "one-off", "drop-off",
		"stand-up", "start-up", "clean-up", "follow-up", "roll-up", "back-up",
		"warm-up", "write-up", "scale-up", "always-on", "add-on", "hands-on",
		"go-live", "round-up", "code-red", "sea-green", "o3-mini-high",
		"scrum-master", "quiz-master", "tree-trunk",
		"1-800-Flowers", "4 Wheel Parts", "500 Startups", "99 Cents Only Stores",
	}
	var wrong []string
	for _, r := range real {
		if NotAnIdentity(r) {
			wrong = append(wrong, r)
		}
	}
	if len(wrong) > 0 {
		t.Errorf("rejected %d/%d real names: %q", len(wrong), len(real), wrong)
	}
	values := []string{"feat/x", "goal/jbuild-gateway", "proof/two-clicks", "seo/resume-page",
		"main trunk", "docket-main", "verified decision", "unblocked", "in-progress tasks"}
	var missed []string
	for _, v := range values {
		if !NotAnIdentity(v) {
			missed = append(missed, v)
		}
	}
	if len(missed) > 0 {
		t.Errorf("accepted %d values: %q", len(missed), missed)
	}
}
