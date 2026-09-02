package resolve

import (
	"errors"
	"regexp"
	"strings"

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
	if isEphemeralName(n) || isGenericAlias(n) || IsValueName(n) {
		return true
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

	owner, owned, err := st.ResolveAlias(alias)
	if err != nil {
		return false, "", err
	}
	if owned && owner == e.Slug {
		return true, "already indexed to this entity", nil
	}
	if !owned {
		// The alias might still be another entity's slug that was never
		// alias-indexed (older stores); treat a live entity at that slug as
		// the owner.
		if other, gerr := st.GetEntity(store.Slugify(alias)); gerr == nil && other.Slug != e.Slug {
			owner, owned = other.Slug, true
		} else if gerr != nil && !errors.Is(gerr, store.ErrNotFound) {
			return false, "", gerr
		}
	}

	if owned {
		other, err := st.GetEntity(owner)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return false, "", err
		}
		if err == nil && !TypesCompatible(other.Type, e.Type) {
			return false, "owned by " + owner + " of incompatible type " + other.Type, nil
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

	// The entity's own name, aliases, and description are its self-account;
	// an alias that echoes any of them is not a stranger.
	if sharesToken(alias, append([]string{e.Name, e.Description}, e.Aliases...)...) {
		return true, "shares a token with the entity's name or description", nil
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
