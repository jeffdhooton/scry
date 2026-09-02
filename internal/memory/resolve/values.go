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
	bareNumberOrMeasureRE = regexp.MustCompile(`^[~≈<>]?[$€£]?\d[\d.,_]*$` +
		`|^[~≈<>]?[$€£]?\d[\d.,_]*[\s-]*(?:[kmgtp]?i?b|[kmgt]?hz|ms|us|ns|s|sec|secs|seconds?|m|min|mins|minutes?|h|hr|hrs|hours?|d|days?|w|weeks?|%|x|k|m|b|bn|rows?|tokens?|params?|parameters?|cores?|threads?|gpus?|cpus?|nodes?|files?|lines?|fps|rps|qps|tps|req/s|mtok|usd|eur)$` +
		`|^[~≈<>]?[$€£]?\d[\d.,_]*[\s-]*(?:[kmgtp]?i?b|[kmgt]?hz|ms|%|params?|parameters?|cores?|threads?|gpus?|cpus?|mtok|k|m|b|bn|x)(?:[\s-][a-z][a-z-]*)*$`)
	// versionRE: "v2", "v1.2.3", "1.2.3", "2026.09", "1.2.3-rc1", "go1.23".
	versionRE = regexp.MustCompile(`^(?:v|go|python|node|php|ruby|java|rust)?\d+(?:\.\d+){1,3}(?:[-+][a-z0-9.]+)?$|^v\d+$`)
	// branchRE: git branch shapes: "feat/x", "fix/123-thing", "release/1.2",
	// "jeff/wip", plus the bare trunk names.
	branchRE = regexp.MustCompile(`^(?:feat|feature|features|fix|fixes|bugfix|hotfix|chore|refactor|docs?|test|tests|release|releases|wip|exp|spike|ci|build|perf|style|revert|dependabot|renovate|codex|claude|kimi|jeff|jclaw|user|users|topic|dev|develop|development|staging|prod|production|main|master|trunk|origin|upstream|setpoint|loop|attempt\d*|worker|agent|task|wave|round|pr|issue|hermes|opencode)/[A-Za-z0-9._/-]+$`)
	// branchWordRE: "current branch", "worker branch", "fix-branch",
	// "the-working-branch", "main branch", "docs/x branch".
	branchWordRE = regexp.MustCompile(`(?:^|[\s_-])branch$|^head(?:[~^]\d*)?$`)
	// hexRE: commit shas up to a full sha256, run ids.
	hexRE = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	// uuidRE: session and thread ids.
	uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	// countRE: "10 users", "83 tests", "80_tests", "275 passing", "409
	// status", "500 error", "54/54 passing", "1623-tests": a number and one
	// or two words whose last word is a count noun or a state.
	countRE = regexp.MustCompile(`^[+-]?\d[\d.,/_-]*[\s_-]+(?:[a-z]+[\s_-]+)?([a-z]+)$`)
	// signedOrSciRE: "-1", "+5", "1e6", "40 percent", "version 1.2.1",
	// "Version 2", "Q3 2026", "September 2026", "2026-09".
	signedOrSciRE = regexp.MustCompile(`^[+-]\d+(?:\.\d+)?$|^\d+(?:\.\d+)?e[+-]?\d+$|^\d+(?:\.\d+)?\s*percent$|^version\s+\d[\w.]*$|^q[1-4]\s+\d{4}$|^(?:january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{4}$|^\d{4}-\d{2}$`)
)

// countNouns are the words after a number in "83 tests" or "10 users":
// the phrase is a measurement of something, not a thing.
var countNouns = map[string]bool{
	"tests": true, "test": true, "users": true, "user": true, "columns": true, "column": true,
	"rows": true, "row": true, "files": true, "file": true, "lines": true, "line": true,
	"items": true, "item": true, "records": true, "record": true, "entries": true, "entry": true,
	"checks": true, "check": true, "steps": true, "step": true, "status": true, "error": true,
	"errors": true, "warnings": true, "warning": true, "failures": true, "failure": true,
	"passes": true, "commits": true, "commit": true, "tasks": true, "task": true, "facts": true,
	"episodes": true, "entities": true, "tokens": true, "token": true, "requests": true,
	"calls": true, "sessions": true, "session": true, "pages": true, "page": true, "words": true,
	"chars": true, "characters": true, "bytes": true, "seconds": true, "minutes": true, "hours": true,
	"days": true, "weeks": true, "months": true, "years": true, "percent": true, "points": true,
	"routes": true, "endpoints": true, "tables": true, "migrations": true, "issues": true,
	"defects": true, "findings": true, "bugs": true, "cases": true, "results": true, "runs": true,
	"cores": true, "threads": true, "nodes": true, "gpus": true, "cpus": true, "boxes": true,
	"insertions": true, "deletions": true, "attempts": true, "retries": true, "rounds": true,
	"waves": true, "phases": true, "questions": true, "hits": true, "misses": true,
}

var (
	// dateRE: "2026-09-02", "2026-09-02T12:00:00Z", "09/02/2026".
	dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(?:[t ]\d{2}:\d{2}(?::\d{2})?z?)?$|^\d{1,2}/\d{1,2}/\d{2,4}$`)
	// timeRE: "12:30", "08:00 AM".
	timeRE = regexp.MustCompile(`^\d{1,2}:\d{2}(?::\d{2})?\s*(?:am|pm)?$`)
	// endpointRE: "halo:13306", "0.0.0.0:8787", "host.tail6e45c2.ts.net:10000",
	// "192.168.1.254", "http://localhost:3000". Addresses locate a thing;
	// they are not the thing.
	endpointRE = regexp.MustCompile(`^(?:https?://)?[a-z0-9.-]+:\d{2,5}(?:/.*)?$|^\d{1,3}(?:\.\d{1,3}){3}(?::\d+)?$|^(?:https?://)?[a-z0-9-]+(?:\.[a-z0-9-]+)+/[^\s]*$`)
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
	if trunkBranches[n] || branchRE.MatchString(n) || branchWordRE.MatchString(n) {
		return true
	}
	if bareNumberOrMeasureRE.MatchString(n) || versionRE.MatchString(n) || hexRE.MatchString(n) || uuidRE.MatchString(n) || dateRE.MatchString(n) || timeRE.MatchString(n) || endpointRE.MatchString(n) || signedOrSciRE.MatchString(n) {
		return true
	}
	if m := countRE.FindStringSubmatch(n); m != nil && (countNouns[m[1]] || statusWords[m[1]]) {
		return true
	}
	return false
}

// IsEphemeralName reports whether a name is a run artifact (temp worktree,
// scratch path, bare hex id, session uuid) rather than an identity.
func IsEphemeralName(name string) bool { return isEphemeralName(name) }

// IsStatusWord reports whether name is a bare state value.
func IsStatusWord(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if statusWords[n] {
		return true
	}
	// A phrase of up to three words that ends in a state ("build failed",
	// "Phase 1 complete", "VOB Pending", "awaiting review"), or that names
	// a status ("completed status", "READY status").
	words := strings.Fields(strings.ReplaceAll(n, "_", " "))
	if len(words) >= 2 && len(words) <= 3 {
		last := words[len(words)-1]
		if statusWords[last] || last == "status" || last == "state" {
			return true
		}
		if (words[0] == "awaiting" || words[0] == "pending" || words[0] == "needs") && len(words) == 2 {
			return true
		}
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
