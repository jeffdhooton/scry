package resolve

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

// Alias admission is where identities used to fuse. The write path took
// every alias the model emitted and indexed it on first sight, so one
// session that called the ops project "the mini" made every later mention
// of the machine resolve to the project. hermes-ops ended up with 130
// aliases including the Halo box, "the machine", and "box".
//
// The rules below decide whether an alias may be added to an entity:
//
//   - never: run artifacts, role words, values, and determiner phrases
//     ("the machine", "this box");
//   - never across incompatible types: an alias owned by a machine cannot
//     be claimed by a project (concept is a wildcard, since stubs are
//     created as concept before anything knows better);
//   - immediately, when the alias is unclaimed and shares a token with the
//     entity's own name or an existing alias ("scry daemon" for scry);
//   - otherwise only once two independent episodes have attested it,
//     including every case where the alias already belongs to, or is the
//     name of, another entity.

// AttestationThreshold is how many distinct episodes must claim an alias
// before it can redirect a name that another entity already owns, or that
// shares nothing with the entity's name.
const AttestationThreshold = 2

// TypesCompatible reports whether entities of types a and b may share an
// alias or be merged. "concept" is the fallback bucket the resolver
// assigns when it knows nothing, so it is compatible with everything.
func TypesCompatible(a, b string) bool {
	a, b = strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	if a == b || a == "" || b == "" || a == "concept" || b == "concept" {
		return true
	}
	return false
}

var determinerRE = regexp.MustCompile(`^(?:the|this|that|these|those|my|your|our|their|its|his|her|a|an|some|our own|my own)\s+`)

// isDeterminerPhrase reports whether an alias is a reference rather than
// a name: "the machine", "this box", "my laptop".
func isDeterminerPhrase(name string) bool {
	return determinerRE.MatchString(strings.ToLower(strings.TrimSpace(name)))
}

// pronouns and role nouns that the extractor keeps emitting as aliases of
// whoever happened to be speaking.
var referenceWords = map[string]bool{
	"i": true, "me": true, "my": true, "mine": true, "you": true, "your": true, "yours": true,
	"we": true, "us": true, "our": true, "ours": true, "they": true, "them": true, "their": true,
	"he": true, "him": true, "she": true, "her": true, "it": true, "its": true, "self": true,
	"user": true, "the user": true, "owner": true, "the owner": true, "developer": true, "the developer": true,
	"human": true, "partner": true, "human partner": true, "operator": true, "the operator": true,
	"assistant": true, "the assistant": true, "ai": true, "model": true, "the model": true, "llm": true,
	"agent": true, "the agent": true, "subagent": true, "sub-agent": true, "the subagent": true,
	"reviewer": true, "the reviewer": true, "implementer": true, "the implementer": true, "grader": true, "the grader": true,
	"worker": true, "the worker": true, "author": true, "the author": true, "member": true, "the member": true,
	"peer": true, "controller": true, "the controller": true, "designer": true, "surveyor": true, "assessor": true,
	"machine": true, "the machine": true, "box": true, "the box": true, "this box": true, "host": true, "the host": true,
	"node": true, "the node": true, "laptop": true, "the laptop": true, "desktop": true, "the desktop": true,
	"gateway": true, "the gateway": true, "dashboard": true, "the dashboard": true, "watcher": true, "the watcher": true,
	"config": true, "the config": true, "ops": true, "infra": true, "the infra": true, "stack": true, "the stack": true,
	"robot": true, "bot": true, "the bot": true, "cli": true, "the cli": true, "tool": true, "the tool": true,
	"session": true, "the session": true, "thread": true, "the thread": true, "room": true, "the room": true,
	"channel": true, "the channel": true, "review": true, "the review": true, "contract": true, "the contract": true,
	"gw": true, "a": true, "hc": true, "jh": true,
}

// neverAlias reports whether a candidate alias is rejected outright.
func neverAlias(alias string) bool {
	n := strings.ToLower(strings.TrimSpace(alias))
	if n == "" || len(n) < 2 {
		return true
	}
	if referenceWords[n] || isDeterminerPhrase(n) || ordinalPhrase(n) {
		return true
	}
	if isEphemeralName(n) || isGenericAlias(n) || isGenericEntityName(n) || IsValueName(n) {
		return true
	}
	return false
}

// commonNouns are words that name a kind of thing, not a thing. An alias
// made only of these ("migration", "design spec", "the switch") can be a
// token-subset of dozens of entity names and must never pick one.
var commonNouns = map[string]bool{
	"migration": true, "migrations": true, "file": true, "files": true, "script": true, "scripts": true,
	"table": true, "tables": true, "model": true, "models": true, "page": true, "pages": true,
	"route": true, "routes": true, "service": true, "services": true, "app": true, "apps": true,
	"api": true, "apis": true, "doc": true, "docs": true, "document": true, "test": true, "tests": true,
	"spec": true, "specs": true, "plan": true, "plans": true, "design": true, "engine": true,
	"agent": true, "agents": true, "box": true, "machine": true, "server": true, "client": true,
	"tool": true, "tools": true, "library": true, "package": true, "module": true, "component": true,
	"feature": true, "features": true, "fix": true, "bug": true, "issue": true, "task": true, "tasks": true,
	"project": true, "repo": true, "repository": true, "branch": true, "commit": true, "release": true,
	"config": true, "configuration": true, "setting": true, "settings": true, "switch": true, "router": true,
	"network": true, "system": true, "process": true, "job": true, "run": true, "runs": true, "loop": true,
	// Ordinary nouns a graph names things after, each of which had
	// collected everything said near it: audit, gate, guard, photo,
	// session, chapter, dispatch, finding, evidence.
	"audits": true, "gate": true, "gates": true, "guard": true, "guards": true,
	"photo": true, "photos": true, "session": true, "sessions": true, "chapter": true, "chapters": true,
	"dispatch": true, "finding": true, "findings": true, "evidence": true, "progress": true,
	"outcome": true, "outcomes": true, "gap": true, "gaps": true, "goal": true, "goals": true,
	"waves": true, "epic": true, "epics": true, "shared": true, "build": true, "builds": true,
	"report": true, "audit": true, "review": true, "phase": true, "step": true, "wave": true, "round": true,
	"mobile": true, "web": true, "site": true, "dashboard": true, "gateway": true, "worker": true,
	"the": true, "and": true, "new": true, "old": true, "main": true, "core": true, "base": true,
	"data": true, "memory": true, "storage": true, "store": true, "cache": true, "queue": true, "log": true, "logs": true,
	"campaign": true, "wireframe": true, "canvas": true, "sheet": true, "token": true, "tokens": true,
}

// hasSpecificToken reports whether at least one token of alias is not a
// common noun.
func hasSpecificToken(alias string) bool {
	for t := range tokensOf(alias) {
		if !commonNouns[t] {
			return true
		}
	}
	return false
}

var aliasTokenRE = regexp.MustCompile(`[^a-z0-9]+`)

// stopTokens carry no identity and never count as shared.
var stopTokens = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true, "into": true, "this": true,
	"that": true, "its": true, "our": true, "your": true, "app": true, "new": true, "old": true,
	"via": true, "per": true, "not": true, "are": true, "was": true, "has": true, "have": true,
}

// tokensOf lowercases s and splits it into alphanumeric tokens of three or
// more characters, so "Mac mini" and "hermes-ops" share nothing while
// "scry daemon" and "scry" share "scry".
func tokensOf(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range aliasTokenRE.Split(strings.ToLower(s), -1) {
		if len(t) >= 3 && !stopTokens[t] {
			out[t] = true
		}
	}
	return out
}

// sharesToken reports whether alias shares a token with any of names, or
// is a substring/superstring (4+ chars) of one of them.
func sharesToken(alias string, names ...string) bool {
	at := tokensOf(alias)
	na := strings.ToLower(strings.TrimSpace(alias))
	for _, n := range names {
		for t := range tokensOf(n) {
			if at[t] {
				return true
			}
		}
		nn := strings.ToLower(strings.TrimSpace(n))
		if len(na) >= 4 && len(nn) >= 4 && (strings.Contains(nn, na) || strings.Contains(na, nn)) {
			return true
		}
	}
	return false
}

// AdmitAlias decides whether alias may be added to e on the strength of
// episode episodeID, recording an attestation when the decision is "not
// yet". The reason is for logs and tests.
func AdmitAlias(st *store.Store, e store.Entity, alias, episodeID string) (admit bool, reason string, err error) {
	norm := store.Normalize(alias)
	if norm == "" || norm == store.Normalize(e.Name) {
		return false, "is the entity's own name", nil
	}
	for _, a := range e.Aliases {
		if store.Normalize(a) == norm {
			return false, "already an alias", nil
		}
	}
	if neverAlias(alias) {
		return false, "generic, value, or reference word", nil
	}
	// A leak check never refuses an alias that spells the holder's own
	// name: "Android Studio Ladybug" is Android Studio's, whatever
	// hardware noun it contains, and "Codex Reviewer #2" is that
	// reviewer's.
	if !nameAsWord(alias, e) && compactName(alias) != compactName(e.Name) {
		if roleLeak(alias, e) {
			return false, "names a role rather than a person", nil
		}
		if machineLeak(alias, e) {
			return false, "names hardware on a non-machine", nil
		}
	}

	owner, owned, err := aliasOwner(st, alias)
	if err != nil {
		return false, "", err
	}
	if owned && owner == e.Slug {
		return true, "already indexed to this entity", nil
	}

	// Another entity's name plus its kind words names that entity: "Hermes
	// agent" is the service Hermes, "Halo box" the machine, however many
	// tokens they share with this entity's own name.
	if named, err := namedByKindWords(st, alias, e); err != nil {
		return false, "", err
	} else if named != "" {
		return false, "names " + named + " (its name plus kind words)", nil
	}

	if owned {
		other, err := st.GetEntity(owner)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return false, "", err
		}
		if err == nil && !TypesCompatible(other.Type, e.Type) {
			return false, "owned by " + owner + " of incompatible type " + other.Type, nil
		}
		// A concept stub is a wildcard, so anything it collects becomes a
		// bridge between types. It never takes a name a typed entity
		// already answers to — its own name or one of its aliases — no
		// matter how many episodes say so.
		if err == nil && (e.Type == "" || e.Type == "concept") && other.Type != "" && other.Type != "concept" {
			return false, "already answers for the " + other.Type + " " + owner, nil
		}
		// Within a type, two independent episodes may still merge two
		// entities, which is what the done bar asks for. A grader called
		// this a risk — two tools can swallow each other's names — and it
		// is, but the alternative is refusing every legitimate merge of
		// two spellings of one thing, and merges are recoverable from a
		// backup while a graph of near-duplicates is not.
		n, err := st.AttestAlias(e.Slug, norm, episodeID)
		if err != nil {
			return false, "", err
		}
		if n >= AttestationThreshold {
			return true, "attested by " + itoa(n) + " episodes, would merge with " + owner, nil
		}
		return false, "owned by " + owner + ", attested by " + itoa(n) + " episode(s)", nil
	}

	// Immediate admission is narrow on purpose. Sharing one token with the
	// entity's name used to be enough, and that is how "Hermes repo",
	// "Hermes tmux" and "Jeff's own Hermes" all landed on the hermes-ops
	// project. An alias is the entity's own only when it spells the name,
	// or contains the whole name and adds nothing but this type's kind
	// words.
	an := singularTokens(alias)
	en := singularTokens(e.Name)
	if compactName(alias) == compactName(e.Name) || compactName(alias) == compactName(e.Slug) {
		return true, "another spelling of the entity's own name", nil
	}
	// An alias that spells the entity's own name and adds to it ("scry
	// daemon", "scryd", "context-stack/scry" for scry) is the entity's
	// own. Nothing else is admitted on one episode: the check above has
	// already established that no other entity is named in this alias.
	// An entity named by one word takes only aliases whose extra words
	// describe a kind of thing. "gate" answered to COPPA gate, sex gate
	// and Twilio gate; AUDIT-6 to every audit in the graph; session-ts to
	// every session. This runs before the containment rule below, which
	// is what made an earlier version of the guard unreachable, and it
	// applies to every one-word name rather than to a list of common
	// nouns, since gate was not on that list.
	// Unless the name is an ordinary word, in which case containing it
	// says nothing on its own. Such an alias waits for a second episode
	// rather than being refused outright: refusing outright stopped the
	// Mac mini being called "Mac mini M4 Pro" and the App Store being
	// called "iOS App Store", which is the expensive mistake.
	magnet := genericOneWordName(e.Name) && !extrasAreKindWords(alias, e)
	// The alias must contain the name as a word, not as letters inside
	// one: Bloomberg is not loom, Shalom is not halo, descry is not scry.
	if !magnet && nameAsWord(alias, e) {
		return true, "contains the entity's own name", nil
	}
	if len(en) > 0 && len(an) > len(en) {
		contains := true
		for t := range en {
			if !an[t] {
				contains = false
				break
			}
		}
		// An entity named by one word collects everything said near that
		// word unless the extras describe a kind of thing. AUDIT-6 had
		// gathered 107 aliases this way — every a11y audit, privacy audit
		// and audit seam in the graph — and session-ts every session.
		// "scry daemon" is still scry; "collaboration session" is not
		// session.ts.
		// An ordinary-word name, however many words it has, does not take
		// an alias on one episode just for containing it: "audit gate"
		// would otherwise collect COPPA audit gate and billing audit gate
		// exactly as "gate" collected COPPA gate.
		if contains && magnet {
			contains = false
		}
		if contains {
			return true, "contains every word of the entity's name", nil
		}
	}
	n, err := st.AttestAlias(e.Slug, norm, episodeID)
	if err != nil {
		return false, "", err
	}
	if n >= AttestationThreshold {
		return true, "attested by " + itoa(n) + " episodes", nil
	}
	return false, "shares nothing with the name, attested by " + itoa(n) + " episode(s)", nil
}

// admitAliases filters candidates through AdmitAlias for e.
func admitAliases(st *store.Store, e store.Entity, candidates []string, episodeID string) ([]string, error) {
	var out []string
	for _, a := range candidates {
		ok, _, err := AdmitAlias(st, e, a, episodeID)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, a)
			e.Aliases = append(e.Aliases, a)
		}
	}
	return out, nil
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return strings.TrimLeft(strings.Repeat("0", 0)+string(rune('0'+n/10))+string(rune('0'+n%10)), "0")
}

// machineNouns name hardware. An alias built around one of them on an
// entity that is not a machine ("hermes-mini" on the ops project, "Hermes
// box") is the machine's name leaking onto the project.
var machineNouns = map[string]bool{
	"mini": true, "box": true, "boxe": true, "machine": true, "host": true, "laptop": true, "desktop": true,
	"server": true, "node": true, "mac": true, "workstation": true, "vm": true, "instance": true,
	"macbook": true, "imac": true, "studio": true, "pi": true, "nas": true,
}

// kindWords are the nouns that say what kind of thing an entity is. An
// alias made of another entity's name plus its kind words ("Hermes agent"
// for the service Hermes, "Halo box" for the machine AMD Halo) names that
// entity, whatever it was listed on.
var kindWords = map[string]map[string]bool{
	"service":  {"agent": true, "gateway": true, "dashboard": true, "dash": true, "transport": true, "profiles": true, "profile": true, "serve": true, "service": true, "daemon": true, "bot": true, "monitor": true, "cron": true, "job": true, "api": true, "server": true, "app": true, "ai": true, "the": true, "own": true},
	"machine":  {"box": true, "machine": true, "mini": true, "host": true, "server": true, "node": true, "laptop": true, "desktop": true, "mac": true, "pc": true, "vm": true, "the": true, "first": true, "second": true, "new": true, "old": true},
	"tool":     {"cli": true, "tool": true, "binary": true, "command": true, "model": true, "engine": true, "the": true, "endpoint": true},
	"project":  {"repo": true, "repository": true, "project": true, "codebase": true, "app": true, "the": true, "design": true, "doc": true, "docs": true, "spec": true, "plan": true, "site": true, "web": true},
	"person":   {"the": true, "own": true, "user": true, "account": true},
	"concept":  {"the": true},
	"decision": {"the": true, "decision": true, "call": true},
	"runbook":  {"the": true, "runbook": true, "doc": true, "docs": true, "guide": true},
}

// genericOneWordName reports whether a name is a single ordinary word:
// gate, audit, guard, photo, session. Such a name never carries an alias
// that merely contains it, because everything said near the word ends up
// on it. A single word that is a proper name — scry, hermes, docket —
// still does, so "scryd" and "Scry memory" are scry's.
func genericOneWordName(name string) bool {
	raw := tokensOf(name)
	if len(raw) == 0 || len(raw) > 3 {
		return false
	}
	// Every word ordinary, not just one: "audit gate" and "review
	// session" collect exactly what "audit" and "session" did.
	ordinary := 0
	for t := range raw {
		w := singular(t)
		if commonNouns[t] || commonNouns[w] || anyKindWord[t] || anyKindWord[w] ||
			processNouns[t] || processNouns[w] || thingWords[t] || thingWords[w] {
			ordinary++
		}
	}
	if ordinary != len(raw) {
		return false
	}
	if len(raw) > 1 {
		return true
	}
	for t := range raw {
		w := singular(t)
		if commonNouns[t] || commonNouns[w] || anyKindWord[t] || anyKindWord[w] ||
			processNouns[t] || processNouns[w] || thingWords[t] || thingWords[w] {
			return true
		}
	}
	return false
}

// maxCompoundTail is how much may follow a name in a compound written
// without separators before the compound is a different word.
const maxCompoundTail = 3

// nameAsWord reports whether an alias contains the entity's name as a
// word of its own, in any spelling: "scry daemon" and "context-stack/scry"
// contain scry; "descry" and "Bloomberg" do not contain scry or loom.
func nameAsWord(alias string, e store.Entity) bool {
	n := compactName(e.Name)
	if len(n) < 4 {
		return false
	}
	at := tokensOf(alias)
	for t := range at {
		if compactName(t) == n {
			return true
		}
	}
	// Or the alias carries every word of a multi-word name: "Android
	// Studio Ladybug" is Android Studio's.
	if en := singularTokens(e.Name); len(en) > 1 {
		all := true
		for t := range en {
			if !singularTokens(alias)[t] {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	// A compound written without separators counts only when the name
	// starts it and little follows: "scryd" and "wrenops" are the thing,
	// "Kimikaze" is not kimi. A name at the END of a longer word is
	// never the thing — "heirloom" is not loom, "descry" is not scry —
	// which is the case that matters, since a short name is a suffix of
	// a great many ordinary words.
	ca := compactName(alias)
	return len(ca) > len(n) && len(ca)-len(n) <= maxCompoundTail && strings.HasPrefix(ca, n)
}

// oneWordName reports whether a name is a single word once punctuation
// is set aside: "gate", "AUDIT-6", "session-ts", "photo".
func oneWordName(name string) bool {
	raw := tokensOf(name)
	if len(raw) == 0 {
		return false
	}
	words := 0
	for t := range raw {
		if len(t) >= 3 {
			words++
		}
	}
	return words <= 1
}

// extrasAreKindWords reports whether everything an alias adds to an
// entity's name describes a kind of thing.
func extrasAreKindWords(alias string, e store.Entity) bool {
	en := singularTokens(e.Name)
	kw := kindWords[e.Type]
	for t := range singularTokens(alias) {
		// Compared across inflections, so an entity may be called by its
		// own plural: "gates" is gate's.
		if en[t] || en[singular(t)] || kw[t] || anyKindWord[t] {
			continue
		}
		return false
	}
	return true
}

// distinctiveName reports whether a name is specific enough that an
// alias containing it is about it. A common noun is not, and neither is
// a single word of four characters or fewer.
func distinctiveName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" || commonNouns[n] || anyKindWord[n] || machineNouns[n] {
		return false
	}
	// A file name is its extension plus a word, and the word is usually a
	// common one: DESIGN.md does not own "design system", and settings.py
	// does not own "admin settings".
	if fileExtRE.MatchString(n) {
		return false
	}
	if len(strings.Fields(n)) > 1 {
		return true
	}
	return len(compactName(n)) >= 5
}

// anyKindWord is every type's kind words together. What makes an alias
// belong to the entity it names is that the extra words describe a kind
// of thing rather than a different thing: "Hermes repo" and "Hermes
// agent" both name Hermes, while "COPPA gate" and "sex gate" are two
// different gates.
var anyKindWord = func() map[string]bool {
	out := map[string]bool{}
	for _, kw := range kindWords {
		for w := range kw {
			out[w] = true
		}
	}
	return out
}()

// machineLeak reports whether alias names hardware while e is not a
// machine.
func machineLeak(alias string, e store.Entity) bool {
	if e.Type == "machine" || e.Type == "" || e.Type == "concept" {
		return false
	}
	for t := range singularTokens(alias) {
		if machineNouns[t] {
			return true
		}
	}
	return false
}

// roleNouns name what something does in a piece of work, not who or what
// it is. A person is not their role, and a session is not the person
// running it.
var roleNouns = map[string]bool{
	"agent": true, "subagent": true, "bot": true, "assistant": true,
	"reviewer": true, "grader": true, "worker": true, "session": true,
	"exec": true, "classifier": true, "cohort": true, "suite": true,
	"implementer": true, "operator": true, "runner": true, "executor": true,
	"model": true, "coder": true, "planner": true, "auditor": true,
}

// roleLeak reports whether an alias names a role rather than the person
// holding it. A grader found the person `jeff` carrying "Claude agent",
// "coding-agent", "codex exec", "review subagent", "first grader", and
// "dashboard agent": every agent that had worked for him had been folded
// into him.
func roleLeak(alias string, e store.Entity) bool {
	if e.Type != "person" {
		return false
	}
	for t := range singularTokens(alias) {
		if roleNouns[t] {
			return true
		}
	}
	return strings.HasPrefix(strings.TrimSpace(alias), "/")
}

// ordinalWords open a reference to one of several like things rather than
// a name for one of them.
// Cardinals (one, two, three) and adjectives that begin real names (new,
// old, primary, left) are deliberately absent: "two-factor authentication"
// and "New Relic" are names, and dropping them costs more than the
// references they would catch.
var ordinalWords = map[string]bool{
	"first": true, "second": true, "third": true, "fourth": true, "last": true,
	"next": true, "previous": true, "other": true, "another": true, "both": true,
	"either": true, "each": true, "original": true, "spare": true,
}

// numberedTwinRE matches "box1", "box 2", "node-3": a position in a set,
// which names whichever member the speaker meant at the time.
var numberedTwinRE = regexp.MustCompile(`^(?:box|node|machine|host|server|unit|rig|slot|port|instance)[ _-]?[0-9]{1,2}$`)

// ordinalPhrase reports whether an alias picks one of several like things
// by position. Two Halo machines had been fused under "first Halo",
// "second Halo", "both Halos", "box1", and "box2", each of which names a
// different box depending on who is speaking.
func ordinalPhrase(alias string) bool {
	n := strings.ToLower(strings.TrimSpace(alias))
	if numberedTwinRE.MatchString(n) {
		return true
	}
	if snakeIdentRE.MatchString(n) {
		return false // a field name in code, whatever English it reads as
	}
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(n))
	if len(words) < 2 || len(words) > 3 {
		return false
	}
	return ordinalWords[words[0]]
}

// RevalidateAliases drops from e every alias that the current rules would
// refuse: reference words, values, hardware nouns on a non-machine, and
// aliases owned by an entity of an incompatible type. It runs when a
// concept stub is upgraded to a real type, because the aliases it collected
// as a wildcard may not belong to what it turned out to be.
func RevalidateAliases(st *store.Store, e *store.Entity) error {
	kept := e.Aliases[:0]
	for _, a := range e.Aliases {
		if neverAlias(a) || machineLeak(a, *e) || roleLeak(a, *e) {
			continue
		}
		if owner, ok, err := st.ResolveAlias(a); err != nil {
			return err
		} else if ok && owner != e.Slug {
			if other, gerr := st.GetEntity(owner); gerr == nil && !TypesCompatible(other.Type, e.Type) {
				continue
			}
		}
		kept = append(kept, a)
	}
	e.Aliases = kept
	return nil
}

// aliasOwner finds the entity that owns alias: the alias index first, then
// an entity whose slug spells the alias in any form ("halo1" for halo-1,
// "Bryan.Farney" for bryan-farney, "halo_2" for halo2).
func aliasOwner(st *store.Store, alias string) (string, bool, error) {
	owner, ok, err := st.ResolveAlias(alias)
	if err != nil || ok {
		return owner, ok, err
	}
	singularCompact := ""
	{
		toks := make([]string, 0, 4)
		for t := range singularTokens(alias) {
			toks = append(toks, t)
		}
		sort.Strings(toks)
		singularCompact = strings.Join(toks, "")
	}
	for _, cand := range []string{store.Slugify(alias), store.Normalize(alias), compactName(alias)} {
		if cand == "" {
			continue
		}
		if other, gerr := st.GetEntity(cand); gerr == nil {
			return other.Slug, true, nil
		} else if !errors.Is(gerr, store.ErrNotFound) {
			return "", false, gerr
		}
	}
	// Compact match against every entity name: served from a per-store
	// cache that refreshes itself when older than a minute, so the rule is
	// never silently off after a cold start.
	ci := compactIndex(st)
	if ci.stale() {
		if err := RefreshCompactIndex(st); err != nil {
			return "", false, err
		}
	}
	if slug, found := ci.lookup(compactName(alias)); found {
		return slug, true, nil
	}
	// A plural or reordered spelling of a name ("AMD Halos" for AMD Halo).
	if slug, found := ci.lookupSingular(singularCompact); found {
		return slug, true, nil
	}
	return "", false, nil
}

// compactName removes separators so "halo-1", "halo_1", and "halo1"
// compare equal.
func compactName(s string) string { return compact(s) }

// namedByKindWords returns the slug of another entity whose whole name
// appears in the alias, with every remaining word one of that entity's
// kind words: "Hermes agent" names the service Hermes, "halo boxes" and
// "AMD Halos" name the machine AMD Halo, "State License Lookup repo"
// names that project. Such an alias belongs to the entity it names, not
// to whatever happened to be mentioned alongside it.
func namedByKindWords(st *store.Store, alias string, e store.Entity) (string, error) {
	at := singularTokens(alias)
	if len(at) < 2 {
		return "", nil
	}
	ci := compactIndex(st)
	if ci.stale() {
		if err := RefreshCompactIndex(st); err != nil {
			return "", err
		}
	}
	holder := singularTokens(e.Name)
	holderNamed := len(holder) > 0
	for t := range holder {
		if !at[t] {
			holderNamed = false
			break
		}
	}
	if holderNamed {
		// The alias spells its holder's own name too; that is the holder's
		// business, decided by the rules below.
		return "", nil
	}
	best, bestLen := "", 0
	for _, slug := range ci.candidatesFor(at) {
		if slug == e.Slug {
			continue
		}
		other, err := st.GetEntity(slug)
		if err != nil {
			continue
		}
		ci.mu.Lock()
		nt := ci.toks[slug]
		ci.mu.Unlock()
		if len(nt) == 0 || len(nt) == len(at) {
			continue // the alias IS that name; the owner check covers it
		}
		// An alias that spells another entity's whole name, and does not
		// spell its holder's, belongs to that entity — but only when the
		// words it adds are that type's kind words. "Hermes agent" names
		// the service Hermes; "COPPA gate" and "sex gate" do not name the
		// gate service, and "src/lib/schedule" does not name lib.rs.
		//
		// This constraint used to be skipped whenever the two types
		// differed, on the reasoning that a machine's name on a project is
		// misfiled whatever the extra words. Applied to stored aliases at
		// store scale that handed 4,634 aliases to entities whose facts
		// never mention them. A name plus arbitrary words is a different
		// thing that happens to contain a name.
		// A distinctive name carries the alias with it whatever the extra
		// words are: "Hermes tmux" and "Hermes Slack gateway" are both
		// about Hermes. A name that is a common noun, or a single short
		// word, carries nothing on its own — "COPPA gate" and "sex gate"
		// are two different gates, and "kimi-wire-wave33" is not Kimi —
		// so there the extras must describe a kind of thing.
		// Judged on the tokens the match actually used: "lib.rs" reduces
		// to the single short token "lib", which is no more distinctive
		// than "gate".
		// A name of one word never carries an alias on its own, whatever
		// its length. Calling any five-letter word distinctive is what
		// let guard, photo, session-ts and AUDIT-6 collect everything
		// said near them.
		if !distinctiveName(other.Name) || genericOneWordName(other.Name) {
			kw := kindWords[other.Type]
			ok := true
			for t := range at {
				if !nt[t] && !kw[t] && !anyKindWord[t] {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
		}
		// Prefer the most specific name, and the earlier slug when two are
		// equally specific: ownership of a tied alias was a coin flip per
		// process, because the candidates arrive in Go map order.
		if len(nt) > bestLen || (len(nt) == bestLen && best != "" && slug < best) {
			best, bestLen = slug, len(nt)
		}
	}
	return best, nil
}

// compactIdx caches, per store, what the entity names are: their compact
// spellings and their singularised tokens. AdmitAlias needs both to answer
// "does this alias name some other entity?" without a full scan per call.
type compactIdx struct {
	mu        sync.Mutex
	at        time.Time
	names     map[string]string
	singulars map[string]string          // compact name -> slug
	byTok     map[string][]string        // singular token -> slugs
	toks      map[string]map[string]bool // slug -> its name's singular tokens
}

var (
	compactIdxMu sync.Mutex
	compactIdxBy = map[*store.Store]*compactIdx{}
)

func compactIndex(st *store.Store) *compactIdx {
	compactIdxMu.Lock()
	ci, ok := compactIdxBy[st]
	if !ok {
		ci = &compactIdx{}
		compactIdxBy[st] = ci
	}
	compactIdxMu.Unlock()
	return ci
}

func (ci *compactIdx) stale() bool {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	return time.Since(ci.at) > time.Minute
}

// lookupSingular finds an entity whose name has exactly these singular
// tokens, in any order or plurality.
func (ci *compactIdx) lookupSingular(key string) (string, bool) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	if key == "" {
		return "", false
	}
	slug, ok := ci.singulars[key]
	return slug, ok
}

func (ci *compactIdx) lookup(key string) (string, bool) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	if key == "" {
		return "", false
	}
	slug, ok := ci.names[key]
	return slug, ok
}

// RefreshCompactIndex rebuilds the entity-name caches for st.
func RefreshCompactIndex(st *store.Store) error {
	entities, err := st.Entities()
	if err != nil {
		return err
	}
	names := make(map[string]string, len(entities))
	singulars := make(map[string]string, len(entities))
	byTok := map[string][]string{}
	toks := make(map[string]map[string]bool, len(entities))
	for _, e := range entities {
		names[compactName(e.Name)] = e.Slug
		names[compactName(e.Slug)] = e.Slug
		t := singularTokens(e.Name)
		if len(t) == 0 {
			continue
		}
		toks[e.Slug] = t
		keys := make([]string, 0, len(t))
		for tok := range t {
			byTok[tok] = append(byTok[tok], e.Slug)
			keys = append(keys, tok)
		}
		sort.Strings(keys)
		singulars[strings.Join(keys, "")] = e.Slug
	}
	ci := compactIndex(st)
	ci.mu.Lock()
	ci.names, ci.singulars, ci.byTok, ci.toks, ci.at = names, singulars, byTok, toks, time.Now()
	ci.mu.Unlock()
	return nil
}

// singularTokens lowercases, splits, drops stop words, and strips a
// trailing plural "s" or "es" so "AMD Halos" and "halo boxes" reach the
// same tokens as "AMD Halo" and "Halo box".
func singularTokens(s string) map[string]bool {
	out := map[string]bool{}
	for t := range tokensOf(s) {
		// Both forms. Stemming is lossy in ways that split a word from
		// itself — "cases" reduces to "cas" while "case" stays "case", and
		// "hermes" to "herme" though it is nobody's plural — so the word
		// as written is kept beside its stem and the two sets meet on one
		// or the other.
		out[t] = true
		out[singular(t)] = true
	}
	return out
}

// singular strips an English plural. "es" comes off only after a
// sibilant, which is the rule that makes it "es" in the first place:
// boxes, batches, dishes. Taking it off everything turned gates into
// gat, routes into rout and pages into pag, which stopped an entity
// taking its own plural as an alias and hid 96 collisions from the
// audit.
func singular(t string) string {
	switch {
	case len(t) > 4 && strings.HasSuffix(t, "es") && sibilantBefore(t[:len(t)-2]):
		return t[:len(t)-2]
	case len(t) > 3 && strings.HasSuffix(t, "s") && !strings.HasSuffix(t, "ss") && !strings.HasSuffix(t, "us"):
		return t[:len(t)-1]
	}
	return t
}

// sibilantBefore reports whether a stem ends in the sound that takes
// "es": s, x, z, ch, sh.
func sibilantBefore(stem string) bool {
	if strings.HasSuffix(stem, "ch") || strings.HasSuffix(stem, "sh") {
		return true
	}
	switch {
	case strings.HasSuffix(stem, "s"), strings.HasSuffix(stem, "x"), strings.HasSuffix(stem, "z"):
		return true
	}
	return false
}

// candidatesFor returns slugs whose name tokens are all present in at,
// looked up through the token index rather than by scanning every entity.
func (ci *compactIdx) candidatesFor(at map[string]bool) []string {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for tok := range at {
		for _, slug := range ci.byTok[tok] {
			if seen[slug] {
				continue
			}
			seen[slug] = true
			nt := ci.toks[slug]
			if len(nt) == 0 {
				continue
			}
			ok := true
			for t := range nt {
				if !at[t] {
					ok = false
					break
				}
			}
			if ok {
				out = append(out, slug)
			}
		}
	}
	return out
}
