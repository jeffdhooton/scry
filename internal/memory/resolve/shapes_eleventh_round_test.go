package resolve

import "testing"

// The eleventh round measured the previous round's loosening properly:
// against the commit before it, values accepted went from 14 of 55 to 52
// of 55 while real names rejected barely moved. These are the five
// families that gave the ground away — settings bound to a value, run
// timestamps, hyphenated participles, participle-and-adverb pairs, and
// two-word commands — with the real names that must survive beside them.
func TestValuesTheLooseningGaveAway(t *testing.T) {
	values := []string{
		"AUTH_MODE=dev", "DB_CONNECTION=mysql", "STEP_LIMIT=150", "durable=true",
		"amd_iommu=off", "max_concurrent_children=1", "ANTHROPIC_BAA_STATUS=executed",
		"GRADER2-20260903T000246Z-3", "harvest-run-20260903T164057Z",
		"build-failed", "tests-passed", "migration-pending", "release-approved",
		"deploy-blocked", "completed successfully", "failed silently", "passed cleanly",
		"deferred indefinitely", "go vet", "npm ci", "git bisect", "cargo clippy",
		"docker ps", "brew doctor", "make check", "kubectl describe", "git stash pop",
	}
	var missed []string
	for _, v := range values {
		if !NotAnIdentity(v) {
			missed = append(missed, v)
		}
	}
	if len(missed) > 0 {
		t.Errorf("accepted %d/%d values: %q", len(missed), len(values), missed)
	}
	real := []string{
		"waiting room", "trade-off", "hand-off", "boarding pass", "in-house",
		"docker hub", "git worktree", "PHP session", "error boundary",
		"deferred revenue", "merged cells", "customer success", "solid state",
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
}
