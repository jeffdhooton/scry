package resolve

import (
	"regexp"
	"strings"
)

// A value is not an identity. "46 GiB", "main", "in-progress", and "0.9"
// describe something; they are not things that can be described. Before
// these rules, every one became an entity: `main` (the git branch) had 374
// facts, `in-progress` had 241 and carried "voice-of-customer" as an alias.
// A fact whose endpoint is a value is stored as an attribute fact (see
// store.Fact.Value) instead of an edge to a value node.

var (
	// bareNumberOrMeasureRE: "51", "3.5", "46 GiB", "51b", "12GB", "3.5s",
	// "40%", "1,024 rows", "51b-active-parameters", "46-gib-spare-memory".
	bareNumberOrMeasureRE = regexp.MustCompile(`^[~≈<>]?[$€£]?\d[\d.,_]*$|^[~≈<>]?[$€£]?\d[\d.,_]*[\s-]*(?:[kmgtp]?i?b|[kmgt]?hz|ms|us|ns|s|sec|secs|seconds?|m|min|mins|minutes?|h|hr|hrs|hours?|d|days?|w|weeks?|%|x|k|m|b|bn|rows?|tokens?|params?|parameters?|cores?|threads?|gpus?|cpus?|nodes?|files?|lines?|fps|rps|qps|tps|req/s|mtok|usd|eur)(?:[\s-][a-z][a-z-]*)*$`)
	// versionRE: "v2", "v1.2.3", "1.2.3", "2026.09", "1.2.3-rc1", "go1.23".
	versionRE = regexp.MustCompile(`^(?:v|go|python|node|php|ruby|java|rust)?\d+(?:\.\d+){1,3}(?:[-+][a-z0-9.]+)?$|^v\d+$`)
	// branchRE: git branch shapes: "feat/x", "fix/123-thing", "release/1.2",
	// "jeff/wip", plus the bare trunk names.
	branchRE = regexp.MustCompile(`^(?:feat|feature|features|fix|fixes|bugfix|hotfix|chore|refactor|docs?|test|tests|release|releases|wip|exp|spike|ci|build|perf|style|revert|dependabot|renovate|codex|claude|kimi|jeff|jclaw|user|users|topic|dev|develop|development|staging|prod|production|main|master|trunk|origin|upstream)/[A-Za-z0-9._/-]+$`)
	// hexRE: commit shas, run ids.
	hexRE = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	// dateRE: "2026-09-02", "2026-09-02T12:00:00Z", "09/02/2026".
	dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(?:[t ]\d{2}:\d{2}(?::\d{2})?z?)?$|^\d{1,2}/\d{1,2}/\d{2,4}$`)
	// timeRE: "12:30", "08:00 AM".
	timeRE = regexp.MustCompile(`^\d{1,2}:\d{2}(?::\d{2})?\s*(?:am|pm)?$`)
)

// trunkBranches are branch names with no slash.
var trunkBranches = map[string]bool{
	"main": true, "master": true, "trunk": true, "develop": true, "dev": true,
	"staging": true, "production": true, "prod": true, "head": true, "origin/main": true,
	"origin/master": true, "release": true, "next": true, "canary": true, "nightly": true,
}

// statusWords are state values. They are the dst of a `status` fact far
// more often than they are anything else, and as entities they became
// magnets: `in-progress` collected "partial", "product surface", and
// "voice-of-customer" as aliases.
var statusWords = map[string]bool{
	"in-progress": true, "in progress": true, "in_progress": true, "inprogress": true, "wip": true,
	"in-review": true, "in review": true, "in_review": true, "under review": true, "reviewing": true,
	"done": true, "complete": true, "completed": true, "finished": true, "closed": true, "resolved": true,
	"pending": true, "queued": true, "scheduled": true, "planned": true, "proposed": true, "draft": true,
	"todo": true, "to-do": true, "to do": true, "backlog": true, "not-started": true, "not started": true, "unstarted": true,
	"blocked": true, "stuck": true, "waiting": true, "on-hold": true, "on hold": true, "paused": true, "deferred": true,
	"open": true, "active": true, "inactive": true, "enabled": true, "disabled": true, "live": true, "running": true, "stopped": true, "idle": true,
	"passed": true, "passing": true, "pass": true, "failed": true, "failing": true, "fail": true, "flaky": true, "skipped": true, "skip": true,
	"green": true, "red": true, "yellow": true, "amber": true, "healthy": true, "unhealthy": true, "degraded": true, "down": true, "up": true,
	"broken": true, "working": true, "fixed": true, "unfixed": true, "regressed": true, "stale": true, "fresh": true, "outdated": true, "current": true,
	"verified": true, "unverified": true, "validated": true, "approved": true, "rejected": true, "accepted": true, "declined": true,
	"merged": true, "unmerged": true, "shipped": true, "released": true, "deployed": true, "undeployed": true, "rolled back": true, "rolled-back": true, "reverted": true,
	"ready": true, "not ready": true, "not-ready": true, "needs-work": true, "needs work": true, "needs-review": true, "needs review": true, "needs-fix": true,
	"partial": true, "partially": true, "incomplete": true, "missing": true, "present": true, "absent": true, "unknown": true, "n/a": true, "none": true, "null": true,
	"ok": true, "okay": true, "good": true, "bad": true, "success": true, "successful": true, "failure": true, "error": true, "warning": true, "critical": true,
	"deprecated": true, "archived": true, "retired": true, "obsolete": true, "experimental": true, "beta": true, "alpha": true, "stable": true, "unstable": true,
	"true": true, "false": true, "yes": true, "no": true, "on": true, "off": true, "high": true, "medium": true, "low": true,
	"spec-compliant": true, "compliant": true, "non-compliant": true, "clean": true, "dirty": true, "empty": true, "full": true,
	"interviewphase": true, "interview-phase": true, "interview phase": true, "phase-1": true, "phase-2": true, "phase 1": true, "phase 2": true,
	"code-quality-review": true, "all-tests-race-green": true, "smoke-fix round": true,
}

// IsValueName reports whether name is a value (a number, measurement,
// version, date, git branch, commit hash, or status word) rather than an
// identity. Such a name never becomes an entity: a fact pointing at it is
// stored as an attribute of the other endpoint.
func IsValueName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.Trim(n, "\"'`")
	if n == "" {
		return true
	}
	if IsStatusWord(n) {
		return true
	}
	if trunkBranches[n] || branchRE.MatchString(n) {
		return true
	}
	if bareNumberOrMeasureRE.MatchString(n) || versionRE.MatchString(n) || hexRE.MatchString(n) || dateRE.MatchString(n) || timeRE.MatchString(n) {
		return true
	}
	return false
}

// IsStatusWord reports whether name is a bare state value.
func IsStatusWord(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if statusWords[n] {
		return true
	}
	// "status: done", "state=passing"
	for _, sep := range []string{":", "="} {
		if i := strings.Index(n, sep); i > 0 {
			head := strings.TrimSpace(n[:i])
			if head == "status" || head == "state" || head == "verdict" || head == "result" {
				return true
			}
		}
	}
	return false
}
