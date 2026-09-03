package resolve

import "testing"

// The names below were hand-picked by a grader from the live store, not
// written to match the rules. Each is judged by shape: what kind of thing
// the name is, never which particular name it is.
func TestNotAnIdentityByShape(t *testing.T) {
	// Four names moved to the accepted side deliberately, each traded
	// for several real ones a grader chose independently of the
	// store. "verified decision" and "in-progress tasks" went with
	// the two-word verdict rule, which was rejecting deferred
	// revenue, merged cells, verified account, approved vendor,
	// failed payment, canceled subscription and passed ball. "works
	// as expected" and "no longer needed" went with the three-word
	// message rule, which was rejecting error correcting code,
	// missing middle housing, no reply address, expected goals
	// metric, still life painting and no code platform. A rule that
	// rejects real things destroys the graph; a rule that misses a
	// value leaves one extra node.
	values := []string{
		// Branches, and projects compounded with a branch.
		"docket-main", "childscribe-mobile main", "fleet-smoke main", "scribe main",
		"advocates-main", "program-health-main", "program-health-develop", "staging-develop",
		"prod-main", "cs-merge-main", "wren-main", "ph-develop", "main tip", "main b9f73b5",
		"main commit 4db5a91a030fe2c878d883d64b79e5e6e473121a", "wt/apply", "wt/bench",
		"feat/*", "release/*", "feature/notification-method", "loop/llm-unify prior work",
		"feat/x", "fix/123-thing",
		// Counts, measurements, ratios, and scores.
		"15 relations", "16 processes", "308 in-repo references", "9559 fds", "4 engine fixes",
		"37 tools across 7 domains", "10 of 12", "3 of 12", "12 prod children", "~44 locations",
		"65,536-token context", "100 (flat minimum with ceiling-buffer)", "3-0 unanimous victory",
		"11-tok-per-sec-shallow", "stall at 0%", "29 verbatim-quoted moments", "18 products",
		"8 microservices", "42 dashboards", "6 queues", "engine prod v20", "7a",
		// Judgements and states.
		"unblocked", "approved with audit trail", "Publish already in progress.",
		"greenlit", "in-flight", "at risk",
		// Opaque handles a run named after itself.
		"commit-0998bef", "commit d9a151023fbb21843460a0d7a41f2d7dc18fa179", "829f81",
		"task-1ade2583a2ad", "room 1537e88d31a5", "codex-01a019fc", "0998bef",
		"deploy-evidence-0e30ccc", "repair-checkpoint-52cffba",
	}
	for _, v := range values {
		if !NotAnIdentity(v) {
			t.Errorf("NotAnIdentity(%q) = false, want true: this is a value, not a thing", v)
		}
	}

	identities := []string{
		"packages/shared", "internal/memory/resolve", "gpt-oss-120b", "halo2", "SPEC.md",
		"4Runner", "303 Magazine", "42 CFR Part 2", "1Password", "10GbE switch",
		"docket-staging", "hoopless-production", "deepseek-v4", "claude-opus-5",
		"main-branch-policy", "0030-price-books", "mac-mini", "hermes-ops", "cockpit-daemon",
		"3 nodes cluster design", "childscribe-laravel", "cellsaviors", "scry", "jeff",
		"scip-typescript", "BadgerDB", "Z_AI_API_KEY", "tailscale", "forge",
		"memory-sweep launch agent", "childscribe-engine-core", "hermeswatch",
		"scry memory queue", "docket",
	}
	for _, id := range identities {
		if NotAnIdentity(id) {
			t.Errorf("NotAnIdentity(%q) = true, want false: this names a real thing", id)
		}
	}
}
