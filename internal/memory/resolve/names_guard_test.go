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
		"think:false", "onDelete: set null", "spineWidth:0.25in",
		"Cache-Control: public, max-age=3600",
		"go vet", "npm ci", "git bisect",
		"https://example.com/x",
	} {
		if !IsValueName(n) {
			t.Errorf("ACCEPTED a value: %q", n)
		}
	}
}
