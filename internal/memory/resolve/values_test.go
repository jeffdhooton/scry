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
		"main.go", "main-branch-policy", "release-notes", "status-page", "open-webui", "docket-wave-35",
		"go", "python", "node", "T-Mobile 5G",
	}
	for _, v := range identities {
		if IsValueName(v) {
			t.Errorf("IsValueName(%q) = true, want false", v)
		}
	}
}
