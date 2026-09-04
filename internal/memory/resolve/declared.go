package resolve

import (
	"errors"
	"regexp"
	"strings"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// DeclaredValues is the set of exact names the extraction model typed as a
// value or failed to give a documented identity type. An unknown type is not
// affirmative identity evidence and therefore cannot open the fact-endpoint
// path around resolveEntity. Keys are store.Normalize'd so a fact endpoint
// matches the entity's exact name.
// Aliases are intentionally excluded: one value declaration must not poison
// a separately declared identity that happens to share one of its aliases.
//
// The lexical rules in values.go work on the name alone, which is all they
// have. The model read the episode, so it can tell "main" the branch from
// "main" the service and "46 GiB" the measurement from a machine called
// "46". Where the model expresses a judgement, take it; the rules stay as
// the floor under episodes extracted before this type existed, and under a
// model that forgets to use it.
func DeclaredValues(ents []extract.Ent) map[string]bool {
	var out map[string]bool
	for _, ent := range ents {
		if ent.Type != "value" && (trustedIdentityType(ent) || !untrustedStatusShape(ent.Name)) {
			continue
		}
		if out == nil {
			out = map[string]bool{}
		}
		out[store.Normalize(ent.Name)] = true
	}
	return out
}

func trustedIdentityType(ent extract.Ent) bool {
	if ent.TypeFallback {
		return false
	}
	switch ent.Type {
	case "project", "service", "machine", "tool", "person", "decision", "runbook", "concept":
		return true
	default:
		return false
	}
}

// untrustedStatusShape is deliberately used only when the model did not
// provide a documented type, or when a fact endpoint was not declared at
// all. Explicit identity verdicts remain the context signal that separates
// user_login_failed from validation_failed and PYTHON_ARGCOMPLETE_OK from a
// shouted run verdict.
func untrustedStatusShape(name string) bool {
	if IsValueName(name) {
		return true
	}
	trimmed := strings.TrimSpace(name)
	if !enumTokenRE.MatchString(trimmed) {
		return false
	}
	parts := strings.Split(trimmed, "_")
	last := strings.ToLower(parts[len(parts)-1])
	return trimmed == strings.ToUpper(trimmed) || enumEndings[last]
}

// declaredValue reports whether the model typed name as a value AND the
// store does not already know that name as an entity.
//
// The second half is the guard against a sloppy extraction: one episode
// calling "hermes-ops" a value must not demote an entity that dozens of
// other episodes built. A name the store has never seen has nothing to
// lose, which is exactly the case this is for.
func declaredValue(st *store.Store, declared map[string]bool, name string) bool {
	if !declared[store.Normalize(name)] {
		return false
	}
	exact, err := st.GetEntity(store.Slugify(name))
	if err == nil && store.Normalize(exact.Name) == store.Normalize(name) {
		return false
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return false
	}
	return !namesAnArtifact(name)
}

func exactEstablishedIdentity(st *store.Store, name string) (store.Entity, bool) {
	exact, err := st.GetEntity(store.Slugify(name))
	if err != nil || store.Normalize(exact.Name) != store.Normalize(name) {
		return store.Entity{}, false
	}
	return exact, true
}

// establishedMentionIdentity recognizes an exact identity or an alias whose
// indexed owner actually lists that spelling. The listing check distinguishes
// a durable glossary alias from a stale index claim. Exact name wins first.
func establishedMentionIdentity(st *store.Store, name string, resolvedEntities map[string]string) (string, bool, error) {
	norm := store.Normalize(name)
	if slug := resolvedEntities[norm]; slug != "" {
		return slug, true, nil
	}
	if exact, found := exactEstablishedIdentity(st, name); found {
		return exact.Slug, true, nil
	}
	slug, found, err := st.ResolveAlias(name)
	if err != nil || !found {
		return "", false, err
	}
	owner, err := st.GetEntity(slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	for _, spelling := range append([]string{owner.Name}, owner.Aliases...) {
		if store.Normalize(spelling) == norm {
			return owner.Slug, true, nil
		}
	}
	return "", false, nil
}

// ticketRE matches the ways a ticket, issue or pull request gets named:
// "issue-91", "PR-402", "GH#88", "bug 1204", "SCRY-17".
var ticketRE = regexp.MustCompile(`(?i)^(issue|bug|ticket|story|task|pr|mr|gh|pull request)[-#\s]?\d+$|^[A-Z]{2,10}-\d+$`)

// codeFileRE matches a name ending in a source or document extension.
var codeFileRE = regexp.MustCompile(`(?i)\.(go|ts|tsx|js|jsx|py|php|rb|rs|java|kt|swift|c|h|cpp|cs|sql|sh|yaml|yml|json|toml|md|txt|html|css|vue|proto)$`)

// namesAnArtifact reports whether name is a thing the store should hold as
// an entity no matter what the extraction model called it.
//
// The model is allowed to be sloppy and the resolver is not, and the value
// type is the sloppiest instruction in the prompt: an over-eager verdict on
// a file or a ticket would not merely mislabel it, it would stop the entity
// existing, and a fact between two of them is dropped entirely. Files and
// tickets are the two classes worth defending, because the store holds
// thousands of each and they are exactly what a session talks about.
//
// Deliberately narrow. This is a veto over one instruction, not a second
// opinion on every name; anything not clearly a file or a ticket is left to
// the model's judgement and the lexical rules.
func namesAnArtifact(name string) bool {
	n := strings.TrimSpace(name)
	if n == "" {
		return false
	}
	if ticketRE.MatchString(n) {
		return true
	}
	if codeFileRE.MatchString(n) {
		return true
	}
	// A code position is a value, not the file it points into: the prompt
	// says so, and "queue/outbox.ts:170-173" answers no question a session
	// would ask about a file. Judge it before the path rule, which would
	// otherwise defend it for the slash.
	if codePositionRE.MatchString(n) {
		return false
	}
	// A single-segment absolute path such as /tmp or /etc has no second
	// slash after trimming the root, but is still a durable filesystem
	// identity. Whitespace and URL syntax keep this deliberately path-only.
	if strings.HasPrefix(n, "/") && len(n) > 1 &&
		!strings.ContainsAny(n, " \t") && !strings.Contains(n, "://") {
		return true
	}
	// A path: a slash between two name-ish parts, with no spaces around it.
	// Trim one leading slash so absolute executable paths receive the same
	// artifact protection as relative source paths; URLs remain values.
	pathName := strings.TrimPrefix(n, "/")
	if i := strings.IndexByte(pathName, '/'); i > 0 && i < len(pathName)-1 &&
		!strings.ContainsAny(n, " \t") && !strings.Contains(n, "://") {
		return true
	}
	return false
}

// codePositionRE matches a file with a line or line range stuck on the end.
var codePositionRE = regexp.MustCompile(`:\d+(-\d+)?(,\d+(-\d+)?)*$`)
