package resolve

import "testing"

// A third grader's hand-picked lists. Each round has found the shapes
// the last round's rules did not reach, so each round's list stays: the
// point of keeping all three is that a rule which passes one list and
// fails the next has absorbed rather than generalised.
func TestNotAnIdentityThirdRound(t *testing.T) {
	values := []string{
		"gate 3", "gate-5", "Tier 1", "Tier 2", "Stage 18", "Gate 4", "Sprint 14",
		"Milestone 3", "Batch 7", "Cohort 2", "epic-4", "Build 4102", "revision 88",
		"iOS build 24", "Android versionCode 23", "attempt 3 of 5",
		"1339 Meadow Rd, Columbus OH 43212", "4 Privet Drive, London",
		"Failed to parse product config", "is missing required field(s)", "Invalid Date",
		"We have no reviews yet", "not yet started", "works as expected", "no longer needed",
		"$12-$30-per-click", "£4.50 per seat", "€200 budget",
		"npx vitest run --project db", "pnpm typecheck && lint && test", "rm -rf /tmp/scratch",
		"SELECT * FROM users", "python3 -m unittest discover -s solver",
		"guard: 'self'", "'./primitives/*': './src/primitives/*.ts'",
		"User→Family→ChildProfile→JournalEntry", "record→stop", "charges.task_id → invoice_lines",
		"setpoint-worktree-m6yzax_e", "jobs_backup_20260612_184343",
		"task_1ade2583a2ad", "commit_0998bef", "deploy_evidence_0e30ccc", "room_1537e88d31a5",
		"run_20260903_141500", "last week", "overnight", "this morning",
		"2026-09-03T14:00:00-04:00", "Old wins (3-0)", "seq 12/15",
	}
	var missed []string
	for _, v := range values {
		if !NotAnIdentity(v) {
			missed = append(missed, v)
		}
	}
	if len(missed) > 0 {
		t.Errorf("accepted %d/%d: %q", len(missed), len(values), missed)
	}

	identities := []string{
		"modernc.org/sqlite", "github.com/dgraph-io/badger", "packages/shared",
		"db/migrations/0350_paperwork_day_rules.sql", "claude-sonnet-4-20250514",
		"gpt-oss-120b", "MobileChildController::store()", "42 CFR Part 2", "RFC 7231",
		"CVE-2024-3094", "SOC 2", "Boeing 737 MAX", "4Runner", "303 Magazine", "WD-40",
		"Route 66", "E*TRADE", "Catch-22", "Studio 54", "PR #87", "CS-186", "REQ-1",
		"~/.scry/config.yaml", "Xbox 360", "max_loaded_models", "valid_from", "approved_at",
		"halo2", "mac-mini", "hermes-ops", "cellsaviors", "scry", "jeff", "SPEC.md",
		"2026-07-17-loom-multi-engine-executors", "0030_price_books", "abandoned cart queue",
		"admissions feature", "36px card layout", "3d orientation test", "Open WebUI",
		"internal/memory/resolve", "hermeswatch", "childscribe-laravel", "1Password",
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
