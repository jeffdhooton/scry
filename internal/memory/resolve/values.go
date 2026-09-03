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
		`|^[~≈<>]?[$€£]?\d[\d.,_]*[\s-]*(?:[kmgtp]?i?b|[kmgt]?hz|ms|us|ns|s|sec|secs|seconds?|m|min|mins|minutes?|h|hr|hrs|hours?|d|days?|w|weeks?|%|x|k|m|b|bn|rows?|tokens?|params?|parameters?|cores?|threads?|gpus?|cpus?|nodes?|files?|lines?|fps|rps|qps|tps|req/s|mtok|usd|eur|px|pt|em|rem|vh|vw|dpi|tok)$` +
		`|^[~≈<>]?[$€£]?\d[\d.,_]*[\s-]*(?:[kmgtp]?i?b|[kmgt]?hz|ms|%|params?|parameters?|cores?|threads?|gpus?|cpus?|mtok|k|m|b|bn|x)(?:[\s-][a-z][a-z-]*)*$`)
	// versionRE: "v2", "v1.2.3", "1.2.3", "2026.09", "1.2.3-rc1", "go1.23".
	versionRE = regexp.MustCompile(`^(?:v|go|python|node|php|ruby|java|rust)?\d+(?:\.\d+){1,3}(?:[-+][a-z0-9.]+)?$|^v\d+$`)
	// branchRE: git branch shapes: "feat/x", "fix/123-thing", "release/1.2",
	// "jeff/wip", plus the bare trunk names.
	branchRE = regexp.MustCompile(`^(?:feat|feature|features|fix|fixes|bugfix|hotfix|chore|refactor|wip|exp|spike|perf|style|revert|dependabot|renovate|codex|claude|kimi|jeff|jclaw|user|users|topic|dev|develop|development|staging|prod|production|main|master|trunk|origin|upstream|setpoint|loop|attempt\d*|worker|wave|round|pr|issue|hermes|opencode|checkpoint|release|releases)/[A-Za-z0-9._/<>-]+$`)
	// fileExtRE: a trailing extension makes a path a file, never a branch.
	fileExtRE = regexp.MustCompile(`\.[a-z][a-z0-9]{0,5}$`)
	// branchWordRE: "current branch", "worker branch", "fix-branch",
	// "the-working-branch", "main branch", "docs/x branch", "branch
	// setpoint/ids", "branch-5cced0b", "main branch at 033ef4c".
	branchWordRE = regexp.MustCompile(`(?:^|[\s_-])branch$|^branch[\s_-](?:[a-z0-9._-]+/|[0-9a-f]{7,}|loop|setpoint|fix|feat|feature|wip|main|master)|^head(?:[~^]\d*)?$|^(?:main|master|develop|trunk)[\s_-]branch(?:[\s_-]at[\s_-]|$)|(?:^|\s)branches$`)
	// uuidAnywhereRE: an id embedded in a name ("approval-request-<uuid>").
	uuidAnywhereRE = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	// userAtHostRE: "jclaw@100.96.45.73", "jeff@mini".
	userAtHostRE = regexp.MustCompile(`^[a-z0-9._-]+@[a-z0-9.-]+$`)
	// httpStatusRE: "HTTP 429", "500 response", "route cache 500 error".
	httpStatusRE = regexp.MustCompile(`^(?:http[\s_-]+)?[1-5]\d\d(?:[\s_/-]+(?:or[\s_-]+)?[1-5]\d\d)*(?:[\s_-]+(?:rate|limit|limits|error|errors|status|statuses|code|codes|response|responses|redirect|redirects|auth|regression|regressions|too|many|requests|request|not|found|unauthorized|forbidden|server|timeout|timeouts|bad|gateway|unavailable|conflict|loop|loops)){0,3}$`)
	// hexRE: commit shas up to a full sha256, run ids.
	hexRE = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	// uuidRE: session and thread ids.
	uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	// countRE: "10 users", "83 tests", "80_tests", "275 passing", "409
	// status", "500 error", "54/54 passing", "1623-tests": a number and one
	// or two words whose last word is a count noun or a state.
	countRE = regexp.MustCompile(`^[+-]?\d[\d.,]*[\s_-]+(?:[a-z]+[\s_-]+)?([a-z]+)$`)
	// dimensionRE: "1200x630", "400 x 400".
	dimensionRE = regexp.MustCompile(`^\d+\s*[x×]\s*\d+$`)
	// durationRE: "14h40m", "2m30s", "1h".
	durationRE = regexp.MustCompile(`^\d+h(?:\d+m)?(?:\d+s)?$|^\d+m(?:\d+s)?$|^\d+s$`)
	// ratioRE: "31 / 100 health score", "10 tests / 326 assertions",
	// "54/54 passing", "6/10".
	ratioRE = regexp.MustCompile(`^\d[\d.,]*\s*/\s*\d[\d.,]*(?:[\s_-]+[a-z][a-z-]*)*$`)
	// leadingNumberRE: names that start with a number, for the "number plus
	// a unit, count, or state anywhere" rule ("116 GB of 125 GB", "793
	// passed, 1 failed", "20.2 tok/s", "8-10").
	leadingNumberRE = regexp.MustCompile(`^[~≈<>+-]?\d`)
	numberRangeRE   = regexp.MustCompile(`^\d+(?:\.\d+)?\s*[-–]\s*\d+(?:\.\d+)?[a-z]{0,4}$`)
	unitTokenRE     = regexp.MustCompile(`^\d[\d.,]*(?:[kmgtp]?i?b|[kmgt]?hz|ms|s|h|d|w|%|x|tok|tokens?|px|mb/s|gb/s|kb/s|tok/s|req/s|fps|rps|qps|tps)$|^(?:[kmgtp]?i?b|[kmgt]?hz|ms|us|ns|sec|secs|min|mins|hr|hrs|tok|px|mb/s|gb/s|kb/s|tok/s|req/s|fps|rps|qps|tps|%)$`)
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
	"clicks": true, "impressions": true, "assertions": true, "specs": true, "spec": true, "typos": true,
	"tools": true, "tool": true, "prs": true, "pr": true, "states": true, "state": true, "sources": true,
	"source": true, "repos": true, "repo": true, "blocks": true, "block": true, "epics": true, "epic": true,
	"branches": true, "branch": true, "bookmarks": true, "likes": true, "retweets": true, "replies": true,
	"watchers": true, "watcher": true, "redirects": true, "redirect": true, "responses": true, "response": true,
	"suites": true, "suite": true, "regenerations": true, "score": true, "scores": true,
	"dependents": true, "consumers": true, "callers": true, "usages": true, "instances": true,
	"fails": true, "occurrences": true, "matches": true, "violations": true, "regressions": true,
	"gb": true, "mb": true, "kb": true, "tb": true, "gib": true, "mib": true,
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
	if trunkBranches[n] || (branchRE.MatchString(n) && !fileExtRE.MatchString(n)) || branchWordRE.MatchString(n) {
		return true
	}
	if uuidAnywhereRE.MatchString(n) || userAtHostRE.MatchString(n) || httpStatusRE.MatchString(n) ||
		numberRangeRE.MatchString(n) || dimensionRE.MatchString(n) || durationRE.MatchString(n) || ratioRE.MatchString(n) {
		return true
	}
	if leadingNumberRE.MatchString(n) && !fileExtRE.MatchString(n) &&
		(measurementPhrase(n) || countPhrase(n) || ratioPhrase(n)) {
		return true
	}
	if bareNumberOrMeasureRE.MatchString(n) || versionRE.MatchString(n) || hexRE.MatchString(n) || uuidRE.MatchString(n) || dateRE.MatchString(n) || timeRE.MatchString(n) || endpointRE.MatchString(n) || signedOrSciRE.MatchString(n) {
		return true
	}
	if m := countRE.FindStringSubmatch(n); m != nil && (countNouns[m[1]] || statusWords[m[1]]) {
		return true
	}
	return ValueShape(n)
}

// IsEphemeralName reports whether a name is a run artifact (temp worktree,
// scratch path, bare hex id, session uuid) rather than an identity.
func IsEphemeralName(name string) bool { return isEphemeralName(name) }

// NotAnIdentity reports whether a name must never be an entity: a value, a
// run artifact, or process vocabulary. Every path that could create an
// entity — a named entity in an extraction, or a fact's endpoint — asks
// this first. A fact whose endpoint is not an identity becomes an
// attribute of the other endpoint rather than an edge to a new node.
func NotAnIdentity(name string) bool {
	return IsValueName(name) || isEphemeralName(name) || isGenericEntityName(name)
}

// IsStatusWord reports whether name is a bare state value.
func IsStatusWord(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if statusWords[n] {
		return true
	}
	// A short phrase that ends in a state ("build failed", "Phase 2 tasks
	// 7-8 complete", "full API suite green", "VOB Pending"), names a
	// status ("completed status", "READY status"), or starts with a state
	// or a waiting word ("ready for merge", "pending-reload", "needs env",
	// "awaiting review", "loop 2 in progress").
	words := strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(n))
	if len(words) >= 2 && len(words) <= 6 {
		last := words[len(words)-1]
		if statusWords[last] || last == "status" || last == "state" || last == "succeeded" || last == "green" {
			return true
		}
		if len(words) >= 3 && statusWords[words[len(words)-2]+" "+last] {
			return true
		}
		first := words[0]
		if (first == "awaiting" || first == "pending" || first == "needs" || first == "ready" || first == "blocked" || first == "waiting") && len(words) <= 4 {
			return true
		}
	}
	if n == "succeeded" || n == "failed" {
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

// resultWords end a phrase that reports an outcome: "13 full
// regenerations logged", "5 residual test failures".
var resultWords = map[string]bool{
	"logged": true, "recorded": true, "remaining": true, "total": true, "measured": true,
	"observed": true, "expected": true, "reported": true, "counted": true, "seen": true,
}

// countPhrase reports whether s is a number followed by words that end in
// something countable, a unit, a state, or a value: "23 pre-existing test
// failures", "7 commits on main", "12-tok-per-sec". A phrase ending in a
// noun that names a thing ("3 nodes cluster design", "303 Magazine") is an
// identity and is left alone.
func countPhrase(s string) bool {
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ", ",", " ", "/", " ").Replace(s))
	if len(words) < 2 || len(words) > 8 || !leadingNumberRE.MatchString(words[0]) {
		return false
	}
	last := words[len(words)-1]
	if countNouns[last] || statusWords[last] || resultWords[last] || unitTokenRE.MatchString(last) {
		return true
	}
	return trunkBranches[last]
}

// ratioPhrase reports whether s is two count phrases either side of a
// slash: "10 tests / 326 assertions".
func ratioPhrase(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !countPhrase(p) && !leadingNumberRE.MatchString(p) {
			return false
		}
	}
	return leadingNumberRE.MatchString(strings.TrimSpace(parts[0]))
}

// measurementFillers are the words allowed between the numbers, units,
// counts, and states of a measurement phrase ("116 GB of 125 GB", "284
// lines across ~90 files", "9,629 passing tests, exit 0").
var measurementFillers = map[string]bool{
	"of": true, "on": true, "over": true, "across": true, "in": true, "per": true, "and": true, "or": true,
	"exit": true, "with": true, "at": true, "to": true, "new": true, "total": true, "vs": true, "out": true,
	"about": true, "approx": true, "approximately": true, "roughly": true, "than": true, "more": true, "less": true,
}

// measurementPhrase reports whether n (lowercase, starting with a number)
// is a number followed directly by a unit, count noun, or state, with every
// remaining word a number, unit, count, state, or filler. "3 nodes cluster
// design" fails on "cluster": a title with a leading number is not a
// measurement.
func measurementPhrase(n string) bool {
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ", ",", " ").Replace(n))
	if len(words) < 2 || len(words) > 9 {
		return false
	}
	isNum := func(w string) bool {
		return leadingNumberRE.MatchString(w) && !unitTokenRE.MatchString(w) || numberRangeRE.MatchString(w)
	}
	isMeasure := func(w string) bool { return countNouns[w] || statusWords[w] || unitTokenRE.MatchString(w) }
	if unitTokenRE.MatchString(words[0]) && len(words) <= 3 {
		return true // "72h-window", "3.5s timeout": a quantity qualified by a noun
	}
	if !isMeasure(words[1]) && !(isNum(words[1]) && len(words) >= 3 && isMeasure(words[2])) {
		return false
	}
	for _, w := range words[1:] {
		if isNum(w) || isMeasure(w) || measurementFillers[w] {
			continue
		}
		return false
	}
	return true
}
