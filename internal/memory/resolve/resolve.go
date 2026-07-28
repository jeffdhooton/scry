// Package resolve is the write path that turns an LLM extraction result
// (internal/memory/extract.Result) into canonical entities and temporal
// facts in the memory store (internal/memory/store). It applies seven
// resolution rules, in order, per episode:
//
//  1. Skip if the episode was already ingested (idempotency).
//  2. Resolve/merge each extracted entity (alias-aware).
//  3. Resolve each fact's src/dst to entity slugs (creating concept stubs
//     for names never seen before) and parse its ValidFrom.
//  4. Merge onto a current fact with the same (src, relation, dst) triple
//     rather than adding a duplicate.
//  5. Apply the fact's "supersedes" hint, invalidating a matching current
//     fact.
//  6. For exclusive relations (e.g. deployed_on), invalidate any current
//     fact with the same (src, relation) but a different dst before adding
//     the new one.
//  7. Record the episode.
package resolve

import (
	"errors"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// DefaultExclusive names relations for which an entity may have at most one
// current fact at a time: extracting a new dst invalidates the old one
// (Rule 6) rather than letting both stand as current.
var DefaultExclusive = map[string]bool{
	"deployed_on": true,
	"status":      true,
	"replaced_by": true,
}

// Stats summarizes the writes performed by one Apply call.
type Stats struct {
	EntitiesCreated  int
	EntitiesUpdated  int
	FactsAdded       int
	FactsInvalidated int
	FactsMerged      int
}

// Apply resolves res into st's entities and facts, attributing everything to
// episode ep. cwd is the working directory of the session the episode came
// from (if known); when it looks like a workspace path (see
// isWorkspacePath), it is unioned into the RepoRefs of every entity touched
// by Rule 2.
//
// Apply is idempotent: if ep has already been ingested (st.HasEpisode), it
// returns a zero Stats and nil error without writing anything.
func Apply(st *store.Store, ep store.Episode, cwd string, res extract.Result, exclusive map[string]bool) (Stats, error) {
	var stats Stats

	// Rule 1: idempotency.
	has, err := st.HasEpisode(ep.ID)
	if err != nil {
		return Stats{}, err
	}
	if has {
		return Stats{}, nil
	}

	// Rule 2: entities.
	for _, ent := range res.Entities {
		if err := resolveEntity(st, ep, cwd, ent, &stats); err != nil {
			return stats, err
		}
	}

	// Rules 3-6: facts.
	for _, fct := range res.Facts {
		if err := resolveFact(st, ep, fct, exclusive, &stats); err != nil {
			return stats, err
		}
	}

	// Rule 7: record the episode last, so a failure above never marks a
	// partially-applied episode as ingested.
	if err := st.PutEpisode(ep); err != nil {
		return stats, err
	}

	return stats, nil
}

// resolveEntity implements Rule 2 for a single extracted entity.
func resolveEntity(st *store.Store, ep store.Episode, cwd string, ent extract.Ent, stats *Stats) error {
	slug, found, err := st.ResolveAlias(ent.Name)
	if err != nil {
		return err
	}
	if !found {
		slug = store.Slugify(ent.Name)
	}
	if slug == "" {
		// Unresolvable name: never write an empty-slug key.
		return nil
	}

	existing, err := st.GetEntity(slug)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}

	if errors.Is(err, store.ErrNotFound) {
		e := store.Entity{
			Slug:        slug,
			Name:        ent.Name,
			Type:        ent.Type,
			Description: ent.Description,
			Aliases:     ent.Aliases,
			CreatedAt:   ep.OccurredAt,
			LastSeen:    ep.OccurredAt,
		}
		if isWorkspacePath(cwd) {
			e.RepoRefs = []string{cwd}
		}
		if err := st.PutEntity(e); err != nil {
			return err
		}
		stats.EntitiesCreated++
		return nil
	}

	// Merge onto the existing entity.
	existing.Aliases = unionStrings(existing.Aliases, ent.Aliases)
	if ent.Description != "" {
		existing.Description = ent.Description
	}
	if ep.OccurredAt.After(existing.LastSeen) {
		existing.LastSeen = ep.OccurredAt
	}
	if isWorkspacePath(cwd) {
		existing.RepoRefs = unionStrings(existing.RepoRefs, []string{cwd})
	}
	if err := st.PutEntity(existing); err != nil {
		return err
	}
	stats.EntitiesUpdated++
	return nil
}

// resolveFact implements Rules 3-6 for a single extracted fact.
func resolveFact(st *store.Store, ep store.Episode, fct extract.Fct, exclusive map[string]bool, stats *Stats) error {
	// Rule 3: resolve endpoints (creating concept stubs as needed) and parse
	// ValidFrom.
	srcSlug, err := ensureEntitySlug(st, ep, fct.Src, stats)
	if err != nil {
		return err
	}
	dstSlug, err := ensureEntitySlug(st, ep, fct.Dst, stats)
	if err != nil {
		return err
	}
	validFrom := parseValidFrom(fct.ValidFrom, ep.OccurredAt)

	merged := false
	if srcSlug != "" && dstSlug != "" {
		// Rule 4: merge onto a current fact with the same triple.
		current, err := currentFact(st, srcSlug, fct.Relation, dstSlug)
		if err != nil {
			return err
		}
		if current != nil {
			if !containsString(current.Episodes, ep.ID) {
				current.Episodes = append(current.Episodes, ep.ID)
			}
			if fct.Confidence > current.Confidence {
				current.Confidence = fct.Confidence
			}
			// ValidFrom (and thus the fact's storage key) is deliberately
			// left untouched: the existing fact was recorded first, so its
			// ValidFrom is the one to keep ("keep the earlier ValidFrom").
			if err := st.PutFact(*current); err != nil {
				return err
			}
			stats.FactsMerged++
			merged = true
		}
	}

	// Rule 5: supersedes hint — independent of what happens to this fact's
	// own triple below.
	if fct.Supersedes != nil {
		if err := applySupersedes(st, ep, *fct.Supersedes, stats); err != nil {
			return err
		}
	}

	if merged || srcSlug == "" || dstSlug == "" {
		return nil
	}

	// Rule 6: exclusive relations invalidate any current fact with the same
	// (src, relation) but a different dst before the new one is added.
	if exclusive[fct.Relation] {
		currents, err := st.FactsFrom(srcSlug, false)
		if err != nil {
			return err
		}
		for _, f := range currents {
			if f.Relation == fct.Relation && f.Dst != dstSlug {
				if err := st.InvalidateFact(f.Src, f.Relation, f.Dst, f.ValidFrom, ep.OccurredAt); err != nil {
					return err
				}
				stats.FactsInvalidated++
			}
		}
	}

	newFact := store.Fact{
		Src:        srcSlug,
		Relation:   fct.Relation,
		Dst:        dstSlug,
		Fact:       fct.Fact,
		ValidFrom:  validFrom,
		Confidence: fct.Confidence,
		Episodes:   []string{ep.ID},
	}
	if err := st.PutFact(newFact); err != nil {
		return err
	}
	stats.FactsAdded++
	return nil
}

// applySupersedes implements Rule 5: resolve the ref's endpoint names
// (without creating stub entities — an unresolved or nonexistent reference
// is simply a no-op) and invalidate the current fact matching its triple, if
// any.
func applySupersedes(st *store.Store, ep store.Episode, ref extract.SupRef, stats *Stats) error {
	srcSlug, err := resolveSlugOnly(st, ref.Src)
	if err != nil {
		return err
	}
	dstSlug, err := resolveSlugOnly(st, ref.Dst)
	if err != nil {
		return err
	}
	if srcSlug == "" || dstSlug == "" {
		return nil
	}

	current, err := currentFact(st, srcSlug, ref.Relation, dstSlug)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	if err := st.InvalidateFact(srcSlug, ref.Relation, dstSlug, current.ValidFrom, ep.OccurredAt); err != nil {
		return err
	}
	stats.FactsInvalidated++
	return nil
}

// ensureEntitySlug resolves name to a slug (alias lookup, else Slugify) and
// guarantees an entity exists at that slug, creating a concept stub
// (Rule 3) if none does. Returns "" without writing anything if name
// slugifies to the empty string.
func ensureEntitySlug(st *store.Store, ep store.Episode, name string, stats *Stats) (string, error) {
	slug, err := resolveSlugOnly(st, name)
	if err != nil {
		return "", err
	}
	if slug == "" {
		return "", nil
	}

	_, err = st.GetEntity(slug)
	if err == nil {
		return slug, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return "", err
	}

	stub := store.Entity{
		Slug:      slug,
		Name:      name,
		Type:      "concept",
		CreatedAt: ep.OccurredAt,
		LastSeen:  ep.OccurredAt,
	}
	if err := st.PutEntity(stub); err != nil {
		return "", err
	}
	stats.EntitiesCreated++
	return slug, nil
}

// resolveSlugOnly resolves name to a slug via the alias index, falling back
// to Slugify. It never creates an entity.
func resolveSlugOnly(st *store.Store, name string) (string, error) {
	slug, found, err := st.ResolveAlias(name)
	if err != nil {
		return "", err
	}
	if !found {
		slug = store.Slugify(name)
	}
	return slug, nil
}

// currentFact returns the current (non-invalidated) fact with the given
// (src, relation, dst) triple, or nil if none exists.
func currentFact(st *store.Store, src, relation, dst string) (*store.Fact, error) {
	facts, err := st.FactsFrom(src, false)
	if err != nil {
		return nil, err
	}
	for i := range facts {
		if facts[i].Relation == relation && facts[i].Dst == dst {
			return &facts[i], nil
		}
	}
	return nil, nil
}

// parseValidFrom parses s as RFC3339, then as a bare date ("2006-01-02"),
// falling back to occurredAt if s is empty or matches neither format.
func parseValidFrom(s string, occurredAt time.Time) time.Time {
	if s == "" {
		return occurredAt
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return occurredAt
}

// isWorkspacePath reports whether cwd looks like a user workspace path
// worth recording as a RepoRef, as opposed to "", "/", or scratch/system
// paths such as "/tmp/...".
func isWorkspacePath(cwd string) bool {
	if cwd == "" {
		return false
	}
	return strings.HasPrefix(cwd, "/Users/")
}

// unionStrings appends elements of incoming to existing that are not
// already present (exact string match), preserving existing's order and the
// order incoming introduces new elements in.
func unionStrings(existing, incoming []string) []string {
	if len(incoming) == 0 {
		return existing
	}
	seen := make(map[string]bool, len(existing))
	out := existing
	for _, s := range existing {
		seen[s] = true
	}
	for _, s := range incoming {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// containsString reports whether s is present in list.
func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
