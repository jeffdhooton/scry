package resolve

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

// HygieneReport summarises what a Hygiene pass changed.
type HygieneReport struct {
	EntitiesScanned int `json:"entities_scanned"`
	EntitiesChanged int `json:"entities_changed"`
	AliasesDropped  int `json:"aliases_dropped"`
	RepoRefsDropped int `json:"repo_refs_dropped"`
	// Conflated are entities carrying an alias that is ALSO another entity's
	// own name — the precise signature of two identities fused into one.
	// Counting repo refs was tried and abandoned: a ref is added merely
	// because an entity was mentioned in a session running there, so shared
	// concepts (anthropic-api, spanning four repos) look identical to genuine
	// fusion. An alias colliding with a real entity is unambiguous.
	//
	// Reported only — splitting is a judgement call about which facts belong
	// where, and no heuristic should make it silently.
	Conflated []ConflationReport `json:"conflated,omitempty"`
	// EphemeralEntities are entities whose own name is a run artifact. They
	// are REPORTED, not deleted: facts reference entities by slug as src and
	// dst, so removing one would orphan its facts. Cleaning them up is a
	// deliberate call with the list in hand.
	EphemeralEntities []string `json:"ephemeral_entities,omitempty"`
	DroppedAliasList  []string `json:"dropped_aliases,omitempty"`
}

// Hygiene repairs entities that were written before the write path rejected
// run artifacts. The prevention fixes stop new pollution; this cleans what is
// already stored:
//
//   - aliases that are temp worktrees, scratch paths, or bare hex ids
//   - repo refs pointing at directories that are no longer repositories
//   - repo refs beyond maxRepoRefs, newest kept
//   - entities whose own name is a run artifact (reported, not deleted)
//
// dryRun reports without writing, because this deletes recorded history and
// the first run should be readable before it is trusted.
func Hygiene(st *store.Store, dryRun bool) (HygieneReport, error) {
	var rep HygieneReport
	entities, err := st.Entities()
	if err != nil {
		return rep, err
	}
	// Two passes. The first decides which slugs are real identities, because
	// a collision against a junk entity ("plan", "commit") is not evidence of
	// anything — and junk can appear anywhere in the list, so it cannot be
	// decided while walking.
	realSlugs := make(map[string]string, len(entities))
	for _, e := range entities {
		if isEphemeralName(e.Name) || isEphemeralName(e.Slug) || isGenericEntityName(e.Name) {
			continue
		}
		realSlugs[e.Slug] = e.Name
	}

	for _, e := range entities {
		rep.EntitiesScanned++

		if isEphemeralName(e.Name) || isEphemeralName(e.Slug) || isGenericEntityName(e.Name) {
			// Flagged, never removed — its facts would be orphaned.
			rep.EphemeralEntities = append(rep.EphemeralEntities, e.Name)
		}

		changed := false

		keptAliases := make([]string, 0, len(e.Aliases))
		for _, a := range e.Aliases {
			if isEphemeralName(a) || isGenericAlias(a) {
				rep.AliasesDropped++
				rep.DroppedAliasList = append(rep.DroppedAliasList, a)
				changed = true
				continue
			}
			keptAliases = append(keptAliases, a)
		}

		keptRefs := make([]string, 0, len(e.RepoRefs))
		for _, r := range e.RepoRefs {
			if !strings.HasPrefix(r, "/Users/") {
				rep.RepoRefsDropped++
				changed = true
				continue
			}
			if _, statErr := os.Stat(filepath.Join(r, ".git")); statErr != nil {
				rep.RepoRefsDropped++
				changed = true
				continue
			}
			keptRefs = append(keptRefs, r)
		}
		if len(keptRefs) > maxRepoRefs {
			rep.RepoRefsDropped += len(keptRefs) - maxRepoRefs
			keptRefs = keptRefs[len(keptRefs)-maxRepoRefs:]
			changed = true
		}

		// An alias that is another entity's own name means two identities are
		// fused. Check the KEPT aliases, so junk already being dropped is not
		// also reported.
		var collisions []string
		for _, a := range keptAliases {
			slug := store.Slugify(a)
			if slug == e.Slug {
				continue
			}
			if name, ok := realSlugs[slug]; ok {
				collisions = append(collisions, name)
			}
		}
		if len(collisions) > 0 {
			rep.Conflated = append(rep.Conflated, ConflationReport{
				Slug: e.Slug, Name: e.Name, CollidesWith: collisions,
			})
		}

		if !changed {
			continue
		}
		rep.EntitiesChanged++
		if dryRun {
			continue
		}
		e.Aliases = keptAliases
		e.RepoRefs = keptRefs
		if err := st.PutEntity(e); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

// ConflationReport names an entity that carries another entity's name as an
// alias — two identities fused into one.
type ConflationReport struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	CollidesWith []string `json:"collides_with"`
}
