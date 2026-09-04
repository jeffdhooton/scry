package resolve

import "testing"

// TestNamesGuard pins both directions at once. Eleven rounds of graders
// have moved these rules back and forth, and every round that fixed one
// direction cost the other until both were measured together.
func TestNamesGuard(t *testing.T) {
	// Real names that must survive.
	for _, n := range []string{
		"scry", "hermes-ops", "Mac mini", "halo2", "Chrome OS", "llama.cpp GPU build",
		"Ready Player One", "Blocked Punt Media", "Ready Set Go",
		"Python 3.13 shim", "Go 1.23 toolchain", "curl 8.11 HTTP3", "Node 22 runtime",
		"petscribe", "cockpit", "accounting-sync", "childscribe-mobile",
		"api/src/routes/payments.ts", "docs/SPEC.md", "issue-91", "PR-402",
		"gpt-oss-120b", "Qwen3.8-Flash-Next", "buildctl", "trawl",
		"scry: a code intelligence daemon for AI agents",
		"9:00 standup",
		// Namespaced identifiers: npm and artisan scripts, skills, model
		// tags, meta properties. A replica dry run caught 82 of these on
		// their way to being retired as colon-bound settings.
		"db:seed", "blog:audit-links", "superpowers:test-driven-development",
		"qwen3.5:9b", "gpt-oss:120b", "og:image", "test:coverage",
		"setpoint-qwen:latest", "chapters:generate-monthly",
		// Feature flags: the flag is the thing, true or false is its value.
		"BATTERY_DESIGNER_ENABLED", "MEMORY_SHARING_ENABLED",
		// Environment variables and ordinary snake_case names.
		"SCRY_MEMORY_SOCKET", "QUICKBOOKS_CLIENT_SECRET", "PORT", "user_login_failed",
	} {
		if IsValueName(n) {
			t.Errorf("REJECTED a real name: %q", n)
		}
	}
	// Values that must not become entities.
	for _, n := range []string{
		"in-progress", "In Progress", "Completed Successfully", "Ready With Caveats",
		"46 GiB", "main", "build-failed", "completed successfully",
		"SCRY_MEMORY_UI_ADDR=off", "GRADER2-20260903T000246Z-3",
		"think:false", "onDelete: set null",
		"Cache-Control: public, max-age=3600",
		"go vet", "npm ci", "git bisect",
		"https://example.com/x",
		// Enum members, by shape rather than by word list.
		"QUALITY_OK", "SPEC_OK", "VALIDATION_FAILED", "CHANGES_REQUIRED",
		"attempt_status_pending", "attempt_status_unknown",
		// A branch keeps being one under a preposition.
		"on feature/demo-account-seeder",
		// Colon-bound settings, where the value side is a literal.
		"turn_detection: null", "calendar_context: {}",
		// Deliberately NOT here: "spineWidth:0.25in". Tightening the colon
		// rule to an explicit literal on the value side gave that up in
		// exchange for the 82 namespaced identifiers above, which is the
		// right way round to be wrong.
	} {
		if !IsValueName(n) {
			t.Errorf("ACCEPTED a value: %q", n)
		}
	}
}
