package resolve

import (
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// DeclaredValues is the set of names the extraction model typed as "value":
// things that describe something rather than being something. Keys are
// store.Normalize'd so a fact endpoint matches whatever spelling the entity
// list used.
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
		if ent.Type != "value" {
			continue
		}
		if out == nil {
			out = map[string]bool{}
		}
		out[store.Normalize(ent.Name)] = true
		for _, a := range ent.Aliases {
			out[store.Normalize(a)] = true
		}
	}
	return out
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
	if _, found, err := st.ResolveAlias(name); err != nil || found {
		return false
	}
	return true
}
