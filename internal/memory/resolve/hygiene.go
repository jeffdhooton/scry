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
	for _, e := range entities {
		rep.EntitiesScanned++

		if isEphemeralName(e.Name) || isEphemeralName(e.Slug) {
			// Flagged, never removed — its facts would be orphaned.
			rep.EphemeralEntities = append(rep.EphemeralEntities, e.Name)
		}

		changed := false

		keptAliases := make([]string, 0, len(e.Aliases))
		for _, a := range e.Aliases {
			if isEphemeralName(a) {
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
