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
	perUnitRE = regexp.MustCompile(`\d[\w.]*[\s_-]*[a-z]+[\s_-]per[\s_-][a-z]+`)
	// numberThenParenRE: "100 (flat minimum with ceiling-buffer)".
	numberThenParenRE = regexp.MustCompile(`^[~≈<>+-]?[\d.,]+\s*\(`)
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
	// trailingPercentRE: "stall at 0%", "coverage 87.5 %".
	trailingPercentRE = regexp.MustCompile(`[\s_-][\d.,]+\s*%$|^[\d.,]+\s*%$`)
)

// trunkNames are the branch names that a project name is compounded with
// to refer to that project's branch: "docket-main", "ph-develop". Only
// the four that are always branches are listed. An environment name such
// as production or staging compounds into real entity names
// ("hoopless-production"), so it is not here.
var trunkNames = map[string]bool{"main": true, "master": true, "develop": true, "trunk": true}

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
const (
	maxNameChars = 80
	maxNameWords = 12
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
	if len(tok) < 8 {
		return digits >= 2
	}
	return digits >= 1
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
	if fileExtRE.MatchString(n) || namesAThing(n) {
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
		if trunks > 0 && others > 0 {
			return true
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
	if !leadingNumberRE.MatchString(n) || fileExtRE.MatchString(n) || namesAThing(n) {
		return false
	}
	if paddedOrdinalRE.MatchString(n) {
		return false
	}
	if numberThenParenRE.MatchString(n) || scoreRE.MatchString(n) || itemLabelRE.MatchString(n) {
		return true
	}
	words := nameWords(n)
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
	words := nameWords(n)
	if len(words) == 0 || len(words) > 4 {
		return false
	}
	if verdictWords[words[0]] {
		return true
	}
	// The judgement may be written as one hyphenated word: "in-progress
	// tasks" splits to three words but reads as two.
	if spaced := strings.Fields(n); len(spaced) > 0 && verdictWords[spaced[0]] {
		return true
	}
	// "un" + a state: unblocked, unshipped, unmerged.
	if len(words) == 1 {
		if rest, ok := strings.CutPrefix(words[0], "un"); ok && statusWords[rest] {
			return true
		}
	}
	if len(words) == 2 && stateOpeners[words[0]] {
		return true
	}
	// The name as written joins a preposition to a state: "in-flight".
	if len(words) == 1 {
		if i := strings.IndexAny(n, "-_"); i > 0 && stateOpeners[n[:i]] {
			return true
		}
	}
	return false
}

// looksMeasured reports whether a name trails a unit, a percentage, a
// version, or a rate, whatever it opens with.
func looksMeasured(n string) bool {
	return trailingPercentRE.MatchString(n) || trailingVersionRE.MatchString(n) || perUnitRE.MatchString(n)
}

// runArtifact reports whether a name is something a run produced and
// named after itself: an id under a run-shaped namespace, or any name
// carrying an opaque hexadecimal handle.
func runArtifact(n string) bool {
	if hasHashToken(n) {
		return true
	}
	if idNamespaceRE.MatchString(n) {
		rest := strings.TrimSpace(idNamespaceRE.ReplaceAllString(n, ""))
		for _, w := range nameWords(rest) {
			if hashLike(w) || uuidRE.MatchString(w) {
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
	if snakeIdentRE.MatchString(n) || datedDocRE.MatchString(n) {
		return false
	}
	return sentenceName(n) || runArtifact(n) || branchPhrase(n) ||
		numberPhrase(n) || verdictPhrase(n) || looksMeasured(n)
}
