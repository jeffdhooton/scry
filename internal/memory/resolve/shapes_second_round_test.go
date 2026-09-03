package resolve

import "testing"

// A second grader's hand-picked lists, taken from the live store after
// the first grader's list had been closed. Two rounds of this is the
// point: the rules must generalise past whichever names a grader
// happened to look at, and every family here had a neighbour the first
// round's rules accepted.
func TestNotAnIdentitySecondRound(t *testing.T) {
	values := []string{
		"five", "two", "zero", "seven", "a dozen", "four test files", "three failures",
		"five Playwright specs", "eight design principles", "one hour", "two weeks",
		"five minute timeout", "ninety percent", "confirmed", "unaudited", "claimed",
		"restored", "untouched", "truncated", "preserved", "behind", "ahead", "nope",
		"maybe", "Sep 2025", "Jan 2025", "Feb 14", "Q3", "last Tuesday",
		"goal/jbuild-gateway-first-flight", "proof/two-clicks", "seo/resume-page-and-content",
		"commit-0390547", "commit e8ebdac", "daemon-939114", "Latency probe 191050-03",
		"Decision 1", "PR #32, #33, #34", "ports 4300, 3301, 5373",
		`"greeting" | "follow_up" | "transition" | "close"`, `action: "greeting"`,
		"One failing language must never abort the whole index build",
		"Detect and report stale and empty indexes instead of a silent green",
		"a parent can create a spontaneous journal entry from mobile",
		"port :4290", "color #D4793C", "half a gigabyte", "twice as fast",
		"0050-0099, 0100-0109", "feedback system, SMS verification, notification preferences",
		"confirmed status ok",
	}
	var missed []string
	for _, v := range values {
		if !NotAnIdentity(v) {
			missed = append(missed, v)
		}
	}
	if len(missed) > 0 {
		t.Errorf("accepted %d/%d as identities: %q", len(missed), len(values), missed)
	}

	// "3M" is deliberately absent: a digit followed by one capital is a
	// unit far more often than a brand here, and 51B, 8B, and 120B are
	// parameter counts that appear constantly. The brand loses the tie.
	identities := []string{
		"modernc.org/sqlite", "gpt-oss-120b", "42 CFR Part 2", "303 Magazine",
		"4Runner", "claude-sonnet-4-20250514", "max_loaded_models", "valid_from",
		"2026-07-17-loom-multi-engine-executors", "0030_price_books",
		"internal/memory/resolve/shapes.go", "Boeing 737 MAX", "iPhone 17 Pro", "Five Guys",
		"packages/shared", "halo2", "SPEC.md", "1Password", "10GbE switch",
		"docket-staging", "hoopless-production", "deepseek-v4", "mac-mini", "hermes-ops",
		"abandoned cart queue", "3d orientation test", "36px card layout", "admissions feature",
		"cockpit-daemon", "childscribe-laravel", "cellsaviors", "scry", "jeff", "tailscale",
		"hermeswatch", "docket", "github.com/dgraph-io/badger", "Open WebUI", "main-branch-policy",
	}
	var wrong []string
	for _, id := range identities {
		if NotAnIdentity(id) {
			wrong = append(wrong, id)
		}
	}
	if len(wrong) > 0 {
		t.Errorf("rejected %d/%d real identities: %q", len(wrong), len(identities), wrong)
	}
}
