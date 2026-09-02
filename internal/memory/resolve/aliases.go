package resolve

import (
	"errors"
	"regexp"
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
	if referenceWords[n] || isDeterminerPhrase(n) {
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
	if machineLeak(alias, e) {
		return false, "names hardware on a non-machine", nil
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
		// A concept stub (a wildcard) never takes a typed entity's own
		// name, no matter how many episodes say so: it would swallow it.
		if err == nil && (e.Type == "" || e.Type == "concept") && other.Type != "" && other.Type != "concept" && store.Normalize(alias) == store.Normalize(other.Name) {
			return false, "is the name of the " + other.Type + " " + owner, nil
		}
		n, err := st.AttestAlias(e.Slug, norm, episodeID)
		if err != nil {
			return false, "", err
		}
		if n >= AttestationThreshold {
			return true, "attested by " + itoa(n) + " episodes, would merge with " + owner, nil
		}
		return false, "owned by " + owner + ", attested by " + itoa(n) + " episode(s)", nil
	}

	// The entity's own name and aliases are its self-account; an alias that
	// echoes them is not a stranger. Its description counts only through
	// a specific token — "loop engine" in a description made of two common
	// nouns is not enough, since descriptions name other things too.
	if sharesToken(alias, append([]string{e.Name}, e.Aliases...)...) {
		return true, "shares a token with the entity's name", nil
	}
	if hasSpecificToken(alias) && sharesToken(alias, e.Description) {
		for t := range tokensOf(alias) {
			if !commonNouns[t] && tokensOf(e.Description)[t] {
				return true, "shares a specific token with the entity's description", nil
			}
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
	"mini": true, "box": true, "machine": true, "host": true, "laptop": true, "desktop": true,
	"server": true, "node": true, "mac": true, "workstation": true, "vm": true, "instance": true,
	"macbook": true, "imac": true, "studio": true, "pi": true, "nas": true,
}

// kindWords are the nouns that say what kind of thing an entity is. An
// alias made of another entity's name plus its kind words ("Hermes agent"
// for the service Hermes, "Halo box" for the machine AMD Halo) names that
// entity, whatever it was listed on.
var kindWords = map[string]map[string]bool{
	"service": {"agent": true, "gateway": true, "dashboard": true, "dash": true, "transport": true, "profiles": true, "profile": true, "serve": true, "service": true, "daemon": true, "bot": true, "monitor": true, "cron": true, "job": true, "api": true, "server": true, "app": true, "ai": true, "the": true, "own": true},
	"machine": {"box": true, "machine": true, "mini": true, "host": true, "server": true, "node": true, "laptop": true, "desktop": true, "mac": true, "pc": true, "vm": true, "the": true, "first": true, "second": true, "new": true, "old": true},
	"tool":    {"cli": true, "tool": true, "binary": true, "command": true, "model": true, "engine": true, "the": true, "endpoint": true},
	"project": {"repo": true, "project": true, "codebase": true, "app": true, "the": true},
	"person":  {"the": true, "user": true},
}

// machineLeak reports whether alias names hardware while e is not a
// machine.
func machineLeak(alias string, e store.Entity) bool {
	if e.Type == "machine" || e.Type == "" || e.Type == "concept" {
		return false
	}
	for t := range tokensOf(alias) {
		if machineNouns[t] {
			return true
		}
	}
	return false
}

// RevalidateAliases drops from e every alias that the current rules would
// refuse: reference words, values, hardware nouns on a non-machine, and
// aliases owned by an entity of an incompatible type. It runs when a
// concept stub is upgraded to a real type, because the aliases it collected
// as a wildcard may not belong to what it turned out to be.
func RevalidateAliases(st *store.Store, e *store.Entity) error {
	kept := e.Aliases[:0]
	for _, a := range e.Aliases {
		if neverAlias(a) || machineLeak(a, *e) {
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
	// Compact match against every entity is a full scan; entities are few
	// enough (tens of thousands) that a per-alias scan is affordable only
	// on the write path's rare cold miss, so it is bounded by a cache.
	if slug, found := compactIndex(st).lookup(compactName(alias)); found {
		return slug, true, nil
	}
	return "", false, nil
}

// compactName removes separators so "halo-1", "halo_1", and "halo1"
// compare equal.
func compactName(s string) string { return compact(s) }

// namedByKindWords returns the slug of an entity of a type incompatible
// with e whose name tokens are all in alias and whose remaining tokens are
// that type's kind words, or "".
func namedByKindWords(st *store.Store, alias string, e store.Entity) (string, error) {
	at := tokensOf(alias)
	if len(at) < 2 {
		return "", nil
	}
	// Candidates: entities whose name is one of the alias's tokens or the
	// alias minus one kind word. Look them up by name rather than scanning.
	for t := range at {
		if kindWords[e.Type][t] {
			continue
		}
		owner, ok, err := aliasOwner(st, t)
		if err != nil {
			return "", err
		}
		if !ok || owner == e.Slug {
			continue
		}
		other, err := st.GetEntity(owner)
		if err != nil || TypesCompatible(other.Type, e.Type) {
			continue
		}
		nt := tokensOf(other.Name)
		if len(nt) == 0 {
			continue
		}
		ok2 := true
		for x := range nt {
			if !at[x] {
				ok2 = false
				break
			}
		}
		if !ok2 {
			continue
		}
		extras := 0
		for x := range at {
			if !nt[x] {
				if !kindWords[other.Type][x] {
					ok2 = false
					break
				}
				extras++
			}
		}
		if ok2 && extras > 0 {
			return other.Slug, nil
		}
	}
	return "", nil
}

// compactIdx is a cache of compact entity names → slug, refreshed at most
// once a minute per store.
type compactIdx struct {
	mu    sync.Mutex
	at    time.Time
	names map[string]string
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

func (ci *compactIdx) lookup(key string) (string, bool) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	if key == "" {
		return "", false
	}
	if time.Since(ci.at) > time.Minute {
		return "", false // refreshed lazily by refresh(); a cold miss is fine
	}
	slug, ok := ci.names[key]
	return slug, ok
}

// RefreshCompactIndex rebuilds the compact-name cache for st.
func RefreshCompactIndex(st *store.Store) error {
	entities, err := st.Entities()
	if err != nil {
		return err
	}
	names := make(map[string]string, len(entities))
	for _, e := range entities {
		names[compactName(e.Name)] = e.Slug
		names[compactName(e.Slug)] = e.Slug
	}
	ci := compactIndex(st)
	ci.mu.Lock()
	ci.names, ci.at = names, time.Now()
	ci.mu.Unlock()
	return nil
}
