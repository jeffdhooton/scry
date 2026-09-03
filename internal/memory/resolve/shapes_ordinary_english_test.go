package resolve

import "testing"

// A sixth grader's independently chosen names. Seven families of
// ordinary English were being rejected: preposition compounds, two-word
// nouns opening with a state word, message openers that also open names,
// leading shell verbs without a command's shape, and bare weekdays.
func TestOrdinaryEnglishSurvives(t *testing.T) {
	real := []string{
		"in-house", "in-memory", "in-place", "in-person", "in-network", "in-line",
		"on-call", "on-prem", "on-premise", "on-site", "on-demand", "on-boarding",
		"on-ramp", "off-ramp", "off-site", "off-peak", "at-bat",
		"needs assessment", "needs-assessment", "needs analysis", "waiting room",
		"waiting-room", "pending tray", "pending-tray", "pending intent",
		"blocked shot", "awaiting-arrival", "unresolved-chord",
		"error boundary", "error budget", "expected value", "still life", "no code",
		"no reply", "not found page", "missing person report", "warning banner",
		"unknown caller",
		"docker hub", "rails engine", "drop shadow", "go router", "create account",
		"delete account flow", "update profile screen", "select all", "php artisan",
		"python club", "cat food", "ls colors",
		"monday", "friday", "Monday.com",
		"under armour", "on deck", "in situ", "in vitro", "off broadway",
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
	values := []string{
		"npx vitest run --project db", "pnpm typecheck && lint && test",
		"rm -rf /tmp/scratch", "python3 -m unittest discover -s solver",
		"in progress", "on hold", "at risk", "in-progress", "last Tuesday",
		"failed to parse product config", "cannot verify from diff",
	}
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
