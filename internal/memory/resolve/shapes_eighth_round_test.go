package resolve

import "testing"

// An eighth grader's independently chosen names, and the command lines
// that had been walking in beside them. Ordinary English keeps losing to
// rules written for a store that never contained it: spaced particle
// compounds, nouns whose last word is also a state word, terms opening
// with a verdict or a message word, names beginning with a month's first
// three letters, and paths past a length bound.
func TestOrdinaryEnglishSurvivesTheEighthRound(t *testing.T) {
	real := []string{
		"stand up", "trade off", "hand off", "start up", "kick off", "day off",
		"face off", "play off", "count down", "touch down",
		"boarding pass", "mountain pass", "customer success", "standard error",
		"heart failure", "storm warning", "alternating current", "putting green",
		"public good", "junior high", "horse stable", "birthday present", "Xbox Live",
		"deferred revenue", "deferred compensation", "merged cells", "verified account",
		"approved vendor", "passed ball",
		"missing middle housing", "error correcting code", "no reply address",
		"expected goals metric", "still life painting", "no code platform",
		"Marketing 101", "Junction 9", "Novation 61", "Decade 3", "Marathon 26",
		"src/main/java/com/example/service/UserAccountServiceImpl.java",
		"apps/web/src/components/settings/NotificationPreferences.tsx",
		"pr/faq", "users/settings", "issue/template", "qa/handbook",
		"double entry ledger", "failed payment", "canceled subscription",
		"paid in full", "write off",
		"PHP session", "git worktree", "python detection fix", "SSH StreamLocalForward",
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
		"npm run build", "composer install", "git worktree prune", "kubectl get pods",
		"yarn install", "cargo build", "pip install requests", "go mod tidy",
		"systemctl restart nginx", "brew install ffmpeg", "php artisan migrate",
		"build failed", "tests passed", "full API suite green", "Phase 1 complete",
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
