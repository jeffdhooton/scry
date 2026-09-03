package resolve

import (
	"regexp"
	"strings"
	"unicode"
)

// Shapes a name can take that make it a value rather than an identity,
// generalised from the shapes a grader found in the live store. The rules
// here are about the form of a name, never about a particular name: a
// list of the store's past mistakes stops working the moment the store
// makes a new one.

var (
	// idNamespaceRE: a name that opens with the kind of thing a run
	// produces, followed by its id. "task-1ade2583a2ad", "room
	// 1537e88d31a5", "codex-01a019fc", "deploy-evidence-0e30ccc".
	idNamespaceRE = regexp.MustCompile(`^(?:task|run|runs|job|room|session|thread|request|approval|ticket|checkpoint|snapshot|artifact|evidence|build|deploy|deployment|commit|sha|hash|worktree|wt|trace|span|correlation)[\s_-]`)
	// branchNamespaceRE: the first path segment of a git branch name.
	// Two segments only: "internal/memory/resolve" is a directory, and
	// "packages/shared" is a module.
	branchNamespaceRE = regexp.MustCompile(`^(?:feat|feature|features|fix|fixes|bugfix|hotfix|chore|refactor|wip|exp|spike|perf|style|revert|dependabot|renovate|codex|claude|kimi|topic|dev|develop|staging|prod|main|master|trunk|origin|upstream|setpoint|loop|attempt\d*|worker|wave|round|pr|issue|opencode|checkpoint|release|releases|wt|worktree)/[^/]+$`)
	// perUnitRE: a rate written out. "11-tok-per-sec-shallow",
	// "20 requests per minute".
	perUnitRE = regexp.MustCompile(`\d[\w.]*[\s_-]*(?:[a-z]+[\s_-])?per[\s_-][a-z]+`)
	// numberThenParenRE: "100 (flat minimum with ceiling-buffer)".
	numberThenParenRE = regexp.MustCompile(`^[~≈<>+-]?[\d.,]+\s*\(`)
	// moneyRE: a name that opens with a currency is an amount.
	moneyRE = regexp.MustCompile(`^[~≈<>+-]?[$€£¥]\s?[\d.,]`)
	// quotedRE: "guard: 'self'", "'./primitives/*': './src/...'" — a
	// quoted literal, or a key with one.
	quotedRE = regexp.MustCompile(`'[^']{2,}'|"[^"]{2,}"`)
	// trailingScoreRE: "Old wins (3-0)".
	trailingScoreRE = regexp.MustCompile(`\(\d{1,3}[-–]\d{1,3}\)$`)
	// scoreRE: "3-0", "2-1 in favour".
	scoreRE = regexp.MustCompile(`^\d{1,3}[-–]\d{1,3}(?:[\s_-]|$)`)
	// itemLabelRE: "7a", "3b" — an item in a list, not a thing.
	itemLabelRE = regexp.MustCompile(`^\d{1,3}[a-z]$`)
	// trailingVersionRE: "engine prod v20", "cockpit v1.2".
	// The version must stand as its own word: "engine prod v20" is a
	// release label, while "deepseek-v4" and "gpt-oss-120b" are the names
	// products are known by.
	trailingVersionRE = regexp.MustCompile(`\s+v\d+(?:\.\d+)*$`)
	// paddedOrdinalRE: "0030-price-books", "0001_init". A zero-padded
	// number orders something; it does not count anything.
	paddedOrdinalRE = regexp.MustCompile(`^0\d{2,}`)
	// snakeIdentRE: "approved_at", "valid_from", "max_loaded_models". A
	// snake_case word is a name in code — a column, a field, a setting —
	// whatever English the words happen to be.
	snakeIdentRE = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)+$`)
	// datedDocRE: "2026-07-17-loom-multi-engine-executors". A name that
	// opens with a date is a dated document, which is a thing.
	datedDocRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[-_ ]\S`)
	// modulePathRE: "modernc.org/sqlite", "github.com/dgraph-io/badger". A
	// host and one path segment name a package, which is a thing. A deeper
	// path is a URL, which locates one.
	modulePathRE = regexp.MustCompile(`^[a-z0-9-]+(?:\.[a-z0-9-]+)+/[a-z0-9._-]+(?:/[a-z0-9._-]+)?$`)
	// brandNumberRE: "7Up", "5Guys". A capital and at least one more
	// letter straight after a digit is a name. Two letters are required
	// because a single one is a unit far more often than a brand in this
	// store: 51B is a parameter count, and 3M loses the tie.
	brandNumberRE = regexp.MustCompile(`^[0-9]+[A-Z][A-Za-z]+$`)
	// slashNameRE: any two-segment name. Git branches are written this way
	// under namespaces nobody can enumerate — goal/, proof/, seo/, ux/ —
	// so the shape is the rule and the exceptions are listed instead.
	slashNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]{1,11}/[a-z0-9._*<>-]+$`)
	// numberedItemRE: "Decision 1", "PR #32", "Wave 4", "port :4290".
	// A pull request and an issue are absent on purpose: "PR #87" names a
	// specific change that sessions say things about, while "Decision 1"
	// and "Wave 4" name a position in a list.
	// A position in a process, with or without a qualifier in front of it:
	// "Gate 3", "iOS build 24", "attempt 3 of 5", "seq 12/15".
	// The qualifier must be a separate word: "iOS build 24" is a build
	// number, while "docket-wave-35" is what a run is called.
	numberedItemRE = regexp.MustCompile(`^(?:\S+ +){0,2}(?:decision|item|step|phase|wave|round|attempt|try|option|slice|lane|gate|tier|stage|sprint|milestone|batch|cohort|epic|checkpoint|revision|build|versioncode|seq|iteration|cycle|question|task)[\s_-]*#?\d+(?:[\s_-]*(?:of|/)[\s_-]*\d+)?$|^ports?[\s_-]*:?\s*\d+`)
	// hashNumberRE: "#42" — one names a change, several make a list.
	hashNumberRE = regexp.MustCompile(`#\d+`)
	// hashColorRE: "#D4793C", "color #D4793C".
	hashColorRE = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`)
	// hexLetterRE distinguishes a colour from an issue number: "#D4793C"
	// carries letters, "#140" does not.
	hexLetterRE = regexp.MustCompile(`[a-fA-F]`)
	// monthYearRE: "Sep 2025", "January 2026", "Feb 14".
	monthYearRE = regexp.MustCompile(`^(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.?[\s_-]+\d{1,4}$`)
	// unAdjectiveRE: "unaudited", "unmerged", "unshipped" — a state written
	// as the negative of a past participle.
	unAdjectiveRE = regexp.MustCompile(`^un[a-z]{3,}(?:ed|able|ible)$`)
	// quarterRE: "Q3", "Q3 2026".
	quarterRE = regexp.MustCompile(`^q[1-4](?:[\s_-]*\d{2,4})?$`)
	// longDigitsRE: a run of six or more digits anywhere in a name is an
	// id — a probe number, a task number, a short sha.
	longDigitsRE = regexp.MustCompile(`^\d{6,}$`)
	// isoDigitsRE: an eight-digit run beginning 19 or 20 is a date, which
	// belongs to names such as claude-sonnet-4-20250514.
	isoDigitsRE = regexp.MustCompile(`^(?:19|20)\d{6}$`)
	// numericListRE: "0050-0099, 0100-0109" — only digits and separators.
	numericListRE = regexp.MustCompile(`^[\d\s,.:_/-]+$`)
	// properPhraseRE: "Five Guys", "Boeing 737 MAX" — every word
	// capitalised is a name, whatever the words mean.
	// Every word capitalised, ending on a word that starts with a letter:
	// "Five Guys", "Boeing 737 MAX". "HTTP 429" ends on a number and is a
	// status code, not a name.
	properPhraseRE = regexp.MustCompile(`^[A-Z0-9][A-Za-z0-9.&'-]*(?:[ -][A-Za-z0-9.&'-]+){0,2}[ -][A-Z][A-Za-z0-9.&'-]*$`)
	// trailingPercentRE: "stall at 0%", "coverage 87.5 %".
	trailingPercentRE = regexp.MustCompile(`[\s_-][\d.,]+\s*%$|^[\d.,]+\s*%$`)
)

// trunkNames are the branch names that a project name is compounded with
// to refer to that project's branch: "docket-main", "ph-develop". Only
// the four that are always branches are listed. An environment name such
// as production or staging compounds into real entity names
// ("hoopless-production"), so it is not here.
// Only main and develop compound into a branch name. Master and trunk
// are ordinary English words first — scrum-master, quiz-master,
// tree-trunk — and reading them as branches cost more than it caught.
var trunkNames = map[string]bool{"main": true, "develop": true}

// allGitWords reports whether every word of a name is git vocabulary.
func allGitWords(words []string) bool {
	for _, w := range words {
		if !trunkNames[w] && !gitNouns[w] && w != "master" && w != "trunk" {
			return false
		}
	}
	return len(words) > 0
}

// gitNouns are the words that follow a branch name when someone is
// talking about the branch itself.
var gitNouns = map[string]bool{
	"tip": true, "head": true, "branch": true, "branches": true, "commit": true,
	"sha": true, "ref": true, "rev": true, "hash": true, "tag": true,
}

// branchNamespaces are the first segments that make a two-segment name a
// branch. The list is open-ended by nature — a team invents a prefix
// whenever it likes — so a name whose head is not here is read as a
// path, which is the cheaper mistake.
var branchNamespaces = map[string]bool{
	"feat": true, "feature": true, "features": true, "fix": true, "fixes": true,
	"bugfix": true, "hotfix": true, "chore": true, "refactor": true, "wip": true,
	"exp": true, "spike": true, "perf": true, "style": true, "revert": true,
	"dependabot": true, "renovate": true, "release": true, "releases": true,
	"topic": true, "dev": true, "develop": true, "staging": true, "prod": true,
	"main": true, "master": true, "trunk": true, "origin": true, "upstream": true,
	"setpoint": true, "loop": true, "worker": true, "wave": true, "round": true,
	"pr": true, "issue": true, "checkpoint": true, "wt": true, "worktree": true,
	"goal": true, "proof": true, "seo": true, "ux": true, "qa": true,
	"codex": true, "claude": true, "kimi": true, "opencode": true, "jeff": true,
	"jclaw": true, "user": true, "users": true, "attempt": true,
}

// codeDirs open a path inside a repository rather than a branch
// namespace. Everything else with one slash in it is read as a branch,
// because branch namespaces cannot be enumerated: a store that knows
// feat/ and loop/ still meets goal/, proof/, seo/, and ux/.
var codeDirs = map[string]bool{
	"packages": true, "internal": true, "src": true, "cmd": true, "docs": true,
	"lib": true, "app": true, "apps": true, "test": true, "tests": true,
	"pkg": true, "api": true, "web": true, "services": true, "scripts": true,
	"tools": true, "config": true, "public": true, "static": true, "assets": true,
	"vendor": true, "node": true, "dist": true, "build": true, "components": true,
	"pages": true, "routes": true, "models": true, "views": true, "db": true,
	"database": true, "migrations": true, "resources": true, "storage": true,
	"bin": true, "etc": true, "usr": true, "var": true, "opt": true, "home": true,
	"users": true, "tmp": true, "private": true, "system": true, "library": true,
}

// wordNumbers are numbers written out. Every rule about quantities was
// gated on a leading digit, so "three failures" walked past all of them
// while "3 failures" did not.
var wordNumbers = map[string]bool{
	"zero": true, "one": true, "two": true, "three": true, "four": true,
	"five": true, "six": true, "seven": true, "eight": true, "nine": true,
	"ten": true, "eleven": true, "twelve": true, "dozen": true, "fifteen": true,
	"twenty": true, "thirty": true, "forty": true, "fifty": true, "sixty": true,
	"seventy": true, "eighty": true, "ninety": true, "hundred": true,
	"thousand": true, "million": true, "billion": true, "half": true,
	"couple": true, "few": true, "several": true, "twice": true, "double": true,
}

// timeUnits follow a number in a duration written out: "one hour".
var timeUnits = map[string]bool{
	"second": true, "seconds": true, "minute": true, "minutes": true,
	"hour": true, "hours": true, "day": true, "days": true, "week": true,
	"weeks": true, "month": true, "months": true, "year": true, "years": true,
	"ms": true, "sec": true, "min": true, "gigabyte": true, "gigabytes": true,
	"megabyte": true, "megabytes": true, "percent": true, "token": true, "tokens": true,
	"terabyte": true, "kilobyte": true,
}

// comparatives follow a bare quantity in a phrase that measures rather
// than names: "twice as fast", "three times slower".
var comparatives = map[string]bool{
	"as": true, "than": true, "fast": true, "faster": true, "slow": true,
	"slower": true, "more": true, "less": true, "better": true, "worse": true,
	"bigger": true, "smaller": true, "longer": true, "shorter": true, "times": true,
}

// weekdays and quarters name a moment, not a thing.
var weekdays = map[string]bool{
	"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
	"friday": true, "saturday": true, "sunday": true, "today": true,
	"yesterday": true, "tomorrow": true, "tonight": true,
}

// irregularPlurals are the count nouns that do not end in s.
var irregularPlurals = map[string]bool{
	"children": true, "people": true, "men": true, "women": true, "feet": true,
	"teeth": true, "indices": true, "matrices": true, "criteria": true, "data": true,
}

// verdictWords open a phrase that reports a judgement rather than naming
// a thing: "verified decision", "approved with audit trail". They are
// kept apart from statusWords because words such as open, active, live,
// and current begin plenty of real names.
var verdictWords = map[string]bool{
	"approved": true, "rejected": true, "verified": true, "unverified": true,
	"validated": true, "blocked": true, "unblocked": true, "merged": true,
	"unmerged": true, "resolved": true, "unresolved": true, "completed": true,
	"failed": true, "passed": true, "pending": true, "deferred": true,
	"cancelled": true, "canceled": true, "superseded": true, "greenlit": true,
	"in-progress": true, "in-review": true, "needs": true, "abandoned": true,
	"postponed": true, "escalated": true, "triaged": true, "reopened": true,
}

// stateOpeners are the two-word states written as a preposition and a
// word: "in flight", "at risk", "on hold".
var stateOpeners = map[string]bool{"in": true, "at": true, "on": true, "off": true, "under": true}

// maxNameChars and maxNameWords bound a name. Past them it is a sentence
// or a paragraph that an extraction mistook for a thing; the longest real
// name in the store is well inside both.
// A name is a noun phrase. Past these it is a sentence someone wrote
// about a thing rather than the name of one. The previous values, 80 and
// 12, were set against the longest name the store then held — 79
// characters and 12 words — so the rule could not fire on anything that
// had survived. These are set against what a name is, and retire the
// sentences that were sitting in the entity table.
const (
	maxNameChars = 56
	maxNameWords = 8
)

// nameWords splits a name the way a reader would, on spaces and the
// punctuation that joins compound names.
func nameWords(n string) []string {
	return strings.FieldsFunc(n, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '\t'
	})
}

// hashLike reports whether a token is a commit sha, a run id, or another
// opaque hexadecimal handle. Six characters and two digits is the floor:
// below it, real names such as halo2 and 120b collide with it.
func hashLike(tok string) bool {
	if len(tok) < 6 || len(tok) > 64 {
		return false
	}
	digits, letters := 0, 0
	for _, r := range tok {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r >= 'a' && r <= 'f':
			letters++
		default:
			return false
		}
	}
	if letters == 0 {
		return false // a run of digits is a number, judged elsewhere
	}
	if len(tok) < 7 {
		return digits >= 2
	}
	return digits >= 1
}

// unitLetters are the letter runs that follow a number as a unit rather
// than as a name: 12GB is a size, 7Up is a drink.
var unitLetters = map[string]bool{
	"b": true, "kb": true, "mb": true, "gb": true, "tb": true, "pb": true,
	"kib": true, "mib": true, "gib": true, "tib": true, "hz": true, "khz": true,
	"mhz": true, "ghz": true, "ms": true, "us": true, "ns": true, "px": true,
	"pt": true, "em": true, "rem": true, "vh": true, "vw": true, "dpi": true,
	"tok": true, "fps": true, "rps": true, "qps": true, "tps": true, "k": true,
	"m": true, "g": true, "t": true, "s": true, "h": true, "d": true, "w": true,
	"x": true, "bn": true, "usd": true, "eur": true, "gbp": true,
}

// numberThenUnit reports whether a name is a number and a unit, which a
// capitalised unit would otherwise pass off as a name: "46 GiB" is a
// quantity, "500 Startups" is a company.
func numberThenUnit(n string) bool {
	words := nameWords(n)
	if len(words) != 2 || !isAllDigits(words[0]) {
		return false
	}
	return unitLetters[words[1]] || countNouns[words[1]] || timeUnits[words[1]]
}

// brandName reports whether a digit-then-capital name is a brand rather
// than a measurement: 7Up is, 12GB is not.
func brandName(s string) bool {
	if !brandNumberRE.MatchString(s) {
		return false
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return !unitLetters[strings.ToLower(s[i:])]
}

// hasHashToken reports whether any word of a name is an opaque handle.
func hasHashToken(n string) bool {
	for _, w := range nameWords(n) {
		if hashLike(strings.Trim(w, "().,:;")) {
			return true
		}
	}
	return false
}

// branchPhrase reports whether a name refers to a git branch: a project
// name compounded with a trunk name ("docket-main", "ph-develop"), a
// namespaced branch ("feat/x", "wt/apply", "loop/llm-unify prior work"),
// or a trunk name carrying a commit ("main b9f73b5").
func branchPhrase(n string) bool {
	if fileExtRE.MatchString(n) || modulePathRE.MatchString(n) {
		return false
	}
	// A two-segment name is a directory unless its first segment is a
	// branch namespace. This default was inverted for a while, on the
	// reasoning that branch namespaces cannot be enumerated, and it cost
	// far more than it caught: terraform/modules, k8s/overlays,
	// helm/charts, proto/billing and eleven of twelve real directories
	// taken from these repositories were all being called branches.
	// Missing a branch leaves one extra node; rejecting a directory
	// destroys a real one.
	if slashNameRE.MatchString(n) {
		if head, _, _ := strings.Cut(n, "/"); branchNamespaces[head] {
			return true
		}
	}
	if namesAThing(n) {
		return false
	}
	words := nameWords(n)
	if len(words) >= 2 {
		trunks, others := 0, 0
		for _, w := range words {
			if trunkNames[w] {
				trunks++
			} else {
				others++
			}
		}
		// A name made only of git words is a branch: "main trunk",
		// "master head".
		if allGitWords(words) {
			return true
		}
		// Otherwise the trunk word must come last, the way a project's
		// branch is written: docket-main, ph-develop. A trunk word in any
		// other position is a word in a name — main-thread-scheduler,
		// master-key-rotation, trunk-based-development — and rejecting
		// those cost real entities.
		if trunks > 0 && trunkNames[words[len(words)-1]] {
			return true
		}
		// Or the trunk word comes first and everything after it is git
		// vocabulary: "main tip", "main branch", "develop head". Anything
		// else after a leading trunk word is a name that starts with an
		// ordinary word.
		if trunks > 0 && trunkNames[words[0]] {
			rest := true
			for _, w := range words[1:] {
				if !gitNouns[w] {
					rest = false
					break
				}
			}
			if rest {
				return true
			}
		}
	}
	if len(words) > 0 && branchNamespaceRE.MatchString(words[0]) {
		return true
	}
	return branchNamespaceRE.MatchString(n)
}

// thingWords end a name that is about something rather than being it:
// "main-branch-policy" is a policy, "3 nodes cluster design" is a design.
// A name ending in one of these is left alone by the shape rules.
var thingWords = map[string]bool{
	"policy": true, "policies": true, "plan": true, "plans": true, "spec": true, "specs": true,
	"design": true, "doc": true, "docs": true, "guide": true, "rule": true, "rules": true,
	"config": true, "script": true, "scripts": true, "tool": true, "service": true, "app": true,
	"repo": true, "project": true, "pipeline": true, "workflow": true, "process": true,
	"strategy": true, "convention": true, "conventions": true, "standard": true, "standards": true,
	"protocol": true, "runbook": true, "checklist": true, "template": true, "schema": true,
	"model": true, "engine": true, "server": true, "daemon": true, "agent": true, "system": true,
	"queue": true, "cart": true, "test": true, "suite": true, "layout": true, "feature": true,
	"form": true, "page": true, "report": true, "index": true, "cache": true, "endpoint": true,
	"route": true, "bucket": true, "topic": true, "stream": true, "surface": true, "flow": true,
	"screen": true, "view": true, "panel": true, "widget": true, "field": true, "column": true,
	"button": true, "job": true, "table": true, "list": true, "gateway": true, "adapter": true,
	"api": true, "apis": true, "endpoints": true, "repository": true,
	"repositories": true, "ui": true, "module": true, "package": true, "library": true,
	"client": true, "worker": true, "workers": true, "handler": true, "controller": true,
	"middleware": true, "hook": true, "hooks": true, "resolver": true, "parser": true,
	"encoder": true, "interface": true, "component": true, "provider": true, "store": true,
}

// namesAThing reports whether a name ends in a word that makes it the
// name of something, whatever shape the rest of it takes.
func namesAThing(n string) bool {
	words := nameWords(n)
	return len(words) > 1 && thingWords[words[len(words)-1]]
}

// plural reports whether a word is a plural count noun.
func plural(w string) bool {
	if irregularPlurals[w] {
		return true
	}
	return len(w) >= 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") && !strings.HasSuffix(w, "us")
}

// numberPhrase reports whether a name is a number and what it counts,
// rather than a thing whose name happens to start with a number. "15
// relations" and "10 of 12" are measurements; "303 Magazine", "4Runner",
// and "42 CFR Part 2" are names.
func numberPhrase(n string) bool {
	if fileExtRE.MatchString(n) || brandName(strings.TrimSpace(n)) {
		return false
	}
	if moneyRE.MatchString(n) {
		return true // an amount of money is a quantity, not a thing
	}
	words := nameWords(strings.TrimPrefix(strings.TrimPrefix(n, "a "), "an "))
	// A name that opens with a quantity is a count even when it ends in a
	// noun; the veto is for names that merely contain one.
	if len(words) > 0 && !wordNumbers[words[0]] && !leadingNumberRE.MatchString(n) && namesAThing(n) {
		return false
	}
	// A number written out counts the same as a number written in digits.
	if len(words) > 0 && wordNumbers[words[0]] {
		if len(words) == 1 {
			return true
		}
		if plural(words[len(words)-1]) {
			return true
		}
		for _, w := range words[1:] {
			if timeUnits[w] || comparatives[w] || countNouns[w] {
				return true
			}
		}
	}
	if !leadingNumberRE.MatchString(n) {
		return false
	}
	if paddedOrdinalRE.MatchString(n) {
		return false
	}
	if numberThenParenRE.MatchString(n) || scoreRE.MatchString(n) || itemLabelRE.MatchString(n) {
		return true
	}
	if quarterRE.MatchString(n) {
		return true
	}
	if len(words) < 2 {
		return false
	}
	// A count noun straight after the number, in a short phrase:
	// "65,536-token context".
	if len(words) <= 3 && (countNouns[words[1]] || plural(words[1]) && len(words) == 2) {
		return true
	}
	// The phrase, or its first clause, ends in what was counted.
	for _, clause := range strings.Split(n, ",") {
		cw := nameWords(strings.TrimSpace(clause))
		if len(cw) >= 2 && plural(cw[len(cw)-1]) {
			return true
		}
	}
	// "10 of 12", "37 tools across 7 domains": two numbers joined by a
	// preposition is an arithmetic statement, not a name.
	numbers := 0
	preposition := false
	for _, w := range words {
		if leadingNumberRE.MatchString(w) {
			numbers++
		}
		switch w {
		case "of", "across", "per", "out", "in", "over":
			preposition = true
		}
	}
	return numbers >= 2 && preposition
}

// sentenceName reports whether a name is prose: too long, too many words,
// more than one sentence, or a list of addresses.
func sentenceName(n string) bool {
	if len(n) > maxNameChars {
		return true
	}
	words := nameWords(n)
	if len(words) > maxNameWords {
		return true
	}
	if strings.Contains(n, "@") && strings.Contains(n, ",") {
		return true
	}
	if strings.HasSuffix(n, ".") && len(words) >= 3 && !fileExtRE.MatchString(n) {
		return true
	}
	return strings.Contains(n, ". ") && len(words) >= 5
}

// verdictPhrase reports whether a name opens with a judgement and says
// little else: "verified decision", "approved with audit trail",
// "in-progress tasks", "at risk", "in flight".
func verdictPhrase(n string) bool {
	if snakeIdentRE.MatchString(n) {
		return false // a field name in code, whatever English it reads as
	}
	// Judged on space-separated words, because a hyphenated compound is
	// one word: "in-progress tasks" is a verdict about tasks, while
	// deferred-revenue-ledger is a ledger, approved-vendor-registry a
	// registry and failed-payment-retrier a retrier. Eighteen of eighteen
	// such names were being rejected while the hyphens were being read as
	// spaces.
	words := strings.Fields(n)
	// Two words, unless a preposition makes it prose: "approved with
	// audit trail" is a verdict, deferred-revenue-ledger is a ledger.
	limit := 2
	for _, w := range words {
		switch w {
		case "with", "for", "in", "on", "at", "by", "of", "to":
			limit = 4
		}
	}
	if len(words) == 0 || len(words) > limit {
		return false
	}
	if namesAThing(n) {
		return false // "abandoned cart queue" is a queue
	}
	if unAdjectiveRE.MatchString(n) {
		return true
	}
	if verdictWords[words[0]] {
		return true
	}
	// "un" + a state: unblocked, unshipped, unmerged.
	if len(words) == 1 && !strings.ContainsAny(words[0], "-_") {
		if rest, ok := strings.CutPrefix(words[0], "un"); ok && statusWords[rest] {
			return true
		}
	}
	if len(words) == 2 && stateOpeners[words[0]] {
		return true
	}
	// The name as written joins a preposition to a state: "in-flight".
	if len(words) == 1 {
		if i := strings.IndexAny(n, "-_"); i > 0 && stateOpeners[n[:i]] && strings.Count(n, "-")+strings.Count(n, "_") == 1 {
			return true
		}
	}
	return false
}

// looksMeasured reports whether a name trails a unit, a percentage, a
// version, or a rate, whatever it opens with.
func looksMeasured(n string) bool {
	if namesAThing(n) {
		return false
	}
	return trailingPercentRE.MatchString(n) || trailingVersionRE.MatchString(n) || perUnitRE.MatchString(n)
}

// shellVerbs open a command line. A command is something you run, not
// something the graph says things about.
var shellVerbs = map[string]bool{
	"npx": true, "npm": true, "pnpm": true, "yarn": true, "bun": true,
	"go": true, "git": true, "curl": true, "wget": true, "brew": true,
	"python": true, "python3": true, "pip": true, "docker": true, "kubectl": true,
	"ssh": true, "scp": true, "rsync": true, "rm": true, "cp": true, "mv": true,
	"cd": true, "ls": true, "cat": true, "sudo": true, "make": true, "cargo": true,
	"composer": true, "php": true, "artisan": true, "rails": true, "bundle": true,
	"launchctl": true, "systemctl": true, "select": true, "insert": true,
	"update": true, "delete": true, "alter": true, "create": true, "drop": true,
}

// errorOpeners begin a message rather than a name: "Failed to parse
// product config", "is missing required field(s)", "We have no reviews
// yet", "not yet started".
var errorOpeners = map[string]bool{
	"failed": true, "invalid": true, "cannot": true, "could": true,
	"unable": true, "missing": true, "unknown": true, "error": true,
	"is": true, "was": true, "were": true, "we": true, "it": true, "there": true,
	"not": true, "no": true, "works": true, "still": true, "already": true,
	"expected": true, "unexpected": true, "warning": true, "must": true,
}

// timePhrases name a moment rather than a thing.
var timePhrases = map[string]bool{
	"last": true, "this": true, "next": true, "overnight": true, "tonight": true,
	"today": true, "yesterday": true, "tomorrow": true, "recently": true,
}

// streetSuffixes end a postal address.
var streetSuffixes = map[string]bool{
	"rd": true, "road": true, "st": true, "street": true, "ave": true, "avenue": true,
	"dr": true, "drive": true, "blvd": true, "ln": true, "lane": true, "way": true,
	"ct": true, "court": true, "pl": true, "place": true, "suite": true, "apt": true,
}

// commandLine reports whether a name is something you would type at a
// shell or a database prompt.
func commandLine(n string) bool {
	words := strings.Fields(n)
	if len(words) < 2 {
		return false
	}
	if shellVerbs[strings.ToLower(words[0])] {
		return true
	}
	return strings.Contains(n, " --") || strings.Contains(n, "&&") || strings.Contains(n, " | ")
}

// messageName reports whether a name reads as a sentence someone was
// shown rather than the name of a thing.
func messageName(n string) bool {
	if len(nameWords(n)) == 1 && timePhrases[nameWords(n)[0]] {
		return true // "overnight", "today"
	}
	// Otherwise a message is prose, and prose has spaces in it. A
	// hyphenated identifier is a name: failed-payment-retrier is a
	// retrier, not a report that a payment failed.
	if !strings.Contains(n, " ") {
		return false
	}
	words := nameWords(strings.Map(func(r rune) rune {
		if r == ',' || r == '.' || r == ':' || r == ';' {
			return ' '
		}
		return r
	}, n))
	if len(words) == 1 && timePhrases[words[0]] {
		return true
	}
	if len(words) < 2 || len(words) > 8 {
		return false
	}
	if errorOpeners[words[0]] {
		return true
	}
	if len(words) <= 3 && timePhrases[words[0]] {
		return true
	}
	// A postal address: a number, then a street.
	if leadingNumberRE.MatchString(n) {
		for _, w := range words {
			if streetSuffixes[w] {
				return true
			}
		}
	}
	return false
}

// chainName reports whether a name is a path drawn between things:
// "User→Family→ChildProfile", "charges.task_id → invoice_lines".
func chainName(n string) bool {
	return strings.Contains(n, "→") || strings.Contains(n, "->") || strings.Contains(n, "⇒")
}

// listName reports whether a name is several values written as one: a
// union of literals, a run of issue numbers, a colour, a numbered item.
func listName(n string) bool {
	if quotedRE.MatchString(n) || trailingScoreRE.MatchString(n) {
		return true
	}
	if strings.Contains(n, "|") || strings.Count(n, ",") >= 2 {
		return true
	}
	if m := hashColorRE.FindString(n); m != "" && hexLetterRE.MatchString(m) {
		return true
	}
	if numberedItemRE.MatchString(n) || monthYearRE.MatchString(n) {
		return true
	}
	if strings.Contains(n, `"`) || numericListRE.MatchString(n) {
		return true
	}
	if quarterRE.MatchString(n) {
		return true
	}
	words := nameWords(n)
	if len(words) == 2 && (ordinalWords[words[0]] || wordNumbers[words[0]]) && weekdays[words[1]] {
		return true
	}
	if len(words) == 1 && weekdays[words[0]] {
		return true
	}
	// One "#42" names a thing; several are a list of them.
	return len(hashNumberRE.FindAllString(n, -1)) >= 2
}

// runArtifact reports whether a name is something a run produced and
// named after itself: an id under a run-shaped namespace, or any name
// carrying an opaque hexadecimal handle.
func runArtifact(n string) bool {
	if hasHashToken(n) {
		return true
	}
	// A long run of digits anywhere is an id: a probe number, a task
	// number. A date is not, and neither is a zero-padded ordinal: a
	// migration named 2026-06-29-000002-create-social-destinations-table
	// is a file, and the store says things about it.
	if !datedDocRE.MatchString(n) {
		for _, w := range nameWords(n) {
			if longDigitsRE.MatchString(w) && !isoDigitsRE.MatchString(w) && !paddedOrdinalRE.MatchString(w) {
				return true
			}
		}
	}
	if idNamespaceRE.MatchString(n) {
		rest := strings.TrimSpace(idNamespaceRE.ReplaceAllString(n, ""))
		for _, w := range nameWords(rest) {
			// Under a run-shaped namespace a bare run of digits is an id
			// too: "commit 2110904" names a commit, not a number.
			if hashLike(w) || uuidRE.MatchString(w) || (len(w) >= 6 && isAllDigits(w)) {
				return true
			}
		}
	}
	return false
}

// isAllDigits reports whether a token is a run of digits.
func isAllDigits(w string) bool {
	if w == "" {
		return false
	}
	for _, r := range w {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ValueShape reports whether a name takes one of the shapes above. It is
// the generalising half of IsValueName: IsValueName knows particular
// forms, this knows kinds of form.
func ValueShape(n string) bool {
	if datedDocRE.MatchString(n) || modulePathRE.MatchString(n) {
		return false
	}
	// Every word capitalised is a name, whatever the words mean: "Five
	// Guys" counts nothing. The shapes a name cannot take are still
	// checked, since a sentence or a commit sha may be capitalised too.
	if properPhraseRE.MatchString(n) {
		return sentenceName(n) || runArtifact(n) || listName(n) || commandLine(n) || chainName(n)
	}
	return sentenceName(n) || runArtifact(n) || branchPhrase(n) ||
		numberPhrase(n) || verdictPhrase(n) || looksMeasured(n) || listName(n) ||
		commandLine(n) || messageName(n) || chainName(n)
}
