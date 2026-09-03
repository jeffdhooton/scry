package resolve

import "testing"

func TestIsValueName(t *testing.T) {
	values := []string{
		"51", "0", "3.5", "46 GiB", "46-gib-spare-memory", "51b-active-parameters", "51B active parameters",
		"12GB", "3.5s", "40%", "1,024 rows", "$25", "~200ms", "128k tokens", "8 cores",
		"v2", "v2.1.0", "1.2.3", "1.2.3-rc1", "go1.23", "python3.12",
		"main", "master", "develop", "HEAD", "feat/memory-provider-base-url", "fix/123-thing", "release/1.2", "jeff/wip",
		"801d14b", "7fc82ec288d6e1f83fc450c27c85a33f45d2fd74",
		"2026-09-02", "2026-09-02T12:00:00Z", "12:30", "08:00 AM",
		"in-progress", "In Progress", "in_progress", "done", "partial", "passed", "broken", "healthy", "not-started",
		"needs-work", "spec-compliant", "interviewphase", "status: done", "state=passing", "true", "n/a",
		"", "  ", "\"done\"",
		"setpoint/billing-domain", "loop/add-money-decimal-boundary", "attempt2/x", "current branch", "worker branch", "fix-branch", "the-working-branch", "main branch", "docs/content-system-finalization branch", "HEAD~1",
		"10 users", "20 columns", "6 tests", "83 tests", "80_tests", "275 passing", "116 GB", "409 status", "500 error", "54/54 passing", "1623-tests", "101 passed",
		"completed status", "draft status", "READY status", "VOB Pending", "context_pending status", "awaiting review", "pending review", "build failed", "Phase 1 complete", "Phase 2 in-progress", "step 2 blocked",
		"7fc82ec288d6e1f83fc450c27c85a33f45d2fd74abcdef0123456789abcdef01", "d039fc89-1234-4abc-8def-0123456789ab", "-1", "+5", "1e6", "40 percent", "Q3 2026", "September 2026", "2026-09", "version 1.2.1", "Version 2",
		"116 GB of 125 GB", "4.0-4.3 MB/s", "20.2 tok/s", "284 lines across ~90 files", "8-10", "72h-window", "793 passed, 1 failed", "885 passed / 0 failed", "9,629 passing tests, exit 0", "16 passing, 16 failing", "12 tests, 40 assertions green", "8 new tests passing",
		"build succeeded", "succeeded", "full API suite green", "loop 2 in progress", "Phase 2 tasks 7-8 complete", "needs-env", "needs-investigation", "pending-reload", "pending user decision", "ready for merge", "ready for distribution", "PENDING-og-images",
		"HTTP 429", "500 response", "HTTP 401/403", "route cache 500 error", "checkpoint/ui-polish-2026-08-27", "branch setpoint/ids", "branch 7f67d76", "branch-5cced0b", "main branch at 033ef4c", "loop/* branches", "branch-loop-adopt-problem-envelope-in-api",
		"approval-request-019ffe23-a83c-7791-93ee-682923e18ca3", "jclaw@100.96.45.73", "halo:13306", "0.0.0.0:8787", "jclaws-mac-mini.tail6e45c2.ts.net:10000", "192.168.1.254", "http://localhost:3000/health",
	}
	for _, v := range values {
		if !IsValueName(v) {
			t.Errorf("IsValueName(%q) = false, want true", v)
		}
	}
	identities := []string{
		"scry", "hermes-ops", "Mac mini", "gpt-oss-120b", "Qwen38-27B-Uncensored-Q8", "childscribe-laravel",
		"0030-price-books", "0001-add-embedding-metadata.sql", "PR #87", "Jeff", "codex-reviewer",
		"GLM-5.3-Flash", "deepseek-v4-flash", "Z_AI_API_KEY", "com.jhoot.scryd", "ai.jermes.scryd",
		"memory-book", "Operations suite", "10 GbE switch", "3Dconnexion", "Halo box", "2fa-service",
		"main.go", "main-branch-policy", "01 Token Sheet.dc.html", "3 nodes cluster design", "release-notes", "status-page", "open-webui", "docket-wave-35",
		"docs/DECISIONS.md", "docs/specs/product-spec.md", "tests/Feature/Web/A11yLintTest.php", "tests/bootstrap.php", "setpoint/campaign/runner.py", "build/output", "ci/workflow.yml", "agent/loop.go", "task/queue.ts", "docs/research", "Android version 10", "stall detector", "green-room", "succeeded-webhook-handler",
		"go", "python", "node", "T-Mobile 5G", "PR #87", "issue #140", "branch protection", "feature-branches", "release-notes", "open-webui", "Live.vue", "status-page", "3 nodes cluster design", "2026-08-22 packet capture", "2026-08-06-family-newsletter-delivery.md",
	}
	for _, v := range identities {
		if IsValueName(v) {
			t.Errorf("IsValueName(%q) = true, want false", v)
		}
	}
}
