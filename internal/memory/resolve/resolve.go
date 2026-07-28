// Package resolve is the write path that turns an LLM extraction result
// (internal/memory/extract.Result) into canonical entities and temporal
// facts in the memory store (internal/memory/store). It applies seven
// resolution rules, in order, per episode:
//
//  1. Skip if the episode was already ingested (idempotency).
//  2. Resolve/merge each extracted entity (alias-aware).
//  3. Resolve each fact's src/dst to entity slugs (creating concept stubs
//     for names never seen before) and parse its ValidFrom.
//  4. Phase A: merge every fact whose (src, relation, dst) exact-matches a
//     CURRENT stored fact, evaluated across the whole episode before any
//     invalidation runs, so the outcome does not depend on the order facts
//     appear in within the episode.
//  5. Phase B, in slice order, for every fact (merged in Phase A or not):
//     apply the fact's "supersedes" hint, invalidating a matching current
//     fact — independent of what happened to the fact's own triple, since
//     the hint targets an explicit (usually different) triple.
//  6. Phase B, continued, for facts NOT merged in Phase A: for exclusive
//     relations (e.g. deployed_on), invalidate any current fact with the
//     same (src, relation) but a different dst before adding the new one.
//  7. Record the episode.
package resolve

import (
	"errors"
	"regexp"
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
	if err := resolveFacts(st, ep, res.Facts, exclusive, &stats); err != nil {
		return stats, err
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

// resolvedFact is one extract.Fct after Rule 3's endpoint/ValidFrom
// resolution, carried between resolveFacts' phases.
type resolvedFact struct {
	fct       extract.Fct
	src, dst  string
	validFrom time.Time
	merged    bool
}

// resolveFacts implements Rules 3-6 for a whole episode's facts, in two
// phases so the outcome never depends on the order facts appear in within
// the episode:
//
//   - Phase A (Rule 4): every fact whose (src, relation, dst) exact-matches
//     a fact that is current in the store *before* this episode's
//     invalidations run gets merged onto it, regardless of where in the
//     slice it appears.
//   - Phase B (Rules 5-6), in slice order: EVERY fact, merged in Phase A or
//     not, applies its supersedes hint (Rule 5) first — it targets an
//     explicit, usually different, triple, so it must not be skipped just
//     because this fact's own triple already merged. Then, only for facts
//     NOT merged in Phase A and exclusive relations, it invalidates any
//     conflicting current fact before adding itself (Rule 6).
//
// Without the Phase A/B split, a fact that flips an exclusive relation
// (e.g. a new deployed_on target) processed before a same-episode
// restatement of the old target would invalidate-then-recreate the old
// fact from scratch instead of merging onto it, losing its provenance and
// confidence history.
func resolveFacts(st *store.Store, ep store.Episode, facts []extract.Fct, exclusive map[string]bool, stats *Stats) error {
	// Rule 3: resolve every fact's endpoints (creating concept stubs as
	// needed) and ValidFrom up front. Relation is normalized (see
	// normalizeRelation) before anything else touches it: facts whose
	// relation normalizes to empty are dropped here, before any entity stub
	// is created for their src/dst, so a garbage relation costs nothing.
	resolved := make([]resolvedFact, 0, len(facts))
	for _, fct := range facts {
		relation := normalizeRelation(fct.Relation)
		if relation == "" {
			continue
		}
		fct.Relation = relation

		srcSlug, err := ensureEntitySlug(st, ep, fct.Src, stats)
		if err != nil {
			return err
		}
		dstSlug, err := ensureEntitySlug(st, ep, fct.Dst, stats)
		if err != nil {
			return err
		}
		resolved = append(resolved, resolvedFact{
			fct:       fct,
			src:       srcSlug,
			dst:       dstSlug,
			validFrom: parseValidFrom(fct.ValidFrom, ep.OccurredAt),
		})
	}

	// Phase A — Rule 4: merge onto any pre-existing current fact with the
	// same triple, for every fact in the episode, before Phase B's
	// invalidations run.
	for i := range resolved {
		rf := &resolved[i]
		if rf.src == "" || rf.dst == "" {
			continue
		}
		current, err := currentFact(st, rf.src, rf.fct.Relation, rf.dst)
		if err != nil {
			return err
		}
		if current == nil {
			continue
		}
		if err := mergeFact(st, ep, *current, rf.validFrom, rf.fct.Confidence, stats); err != nil {
			return err
		}
		rf.merged = true
	}

	// Phase B — Rules 5 & 6, in slice order.
	for i := range resolved {
		rf := &resolved[i]

		// Rule 5: supersedes hint. Runs for every fact, merged or not — it
		// targets its own (usually different) triple via an explicit ref,
		// independent of what happened to rf's own triple in Phase A.
		if rf.fct.Supersedes != nil {
			if err := applySupersedes(st, ep, *rf.fct.Supersedes, stats); err != nil {
				return err
			}
		}

		if rf.merged {
			continue
		}

		// Rule 6 (exclusive invalidation + add) only applies to facts not
		// already merged in Phase A.
		if rf.src == "" || rf.dst == "" {
			continue
		}

		// Rule 6: exclusive relations invalidate any current fact with the
		// same (src, relation) but a different dst before the new one is
		// added.
		if exclusive[rf.fct.Relation] {
			currents, err := st.FactsFrom(rf.src, false)
			if err != nil {
				return err
			}
			for _, f := range currents {
				if f.Relation == rf.fct.Relation && f.Dst != rf.dst {
					at := clampInvalidAt(ep.OccurredAt, f.ValidFrom)
					if err := st.InvalidateFact(f.Src, f.Relation, f.Dst, f.ValidFrom, at); err != nil {
						return err
					}
					stats.FactsInvalidated++
				}
			}
		}

		newFact := store.Fact{
			Src:        rf.src,
			Relation:   rf.fct.Relation,
			Dst:        rf.dst,
			Fact:       rf.fct.Fact,
			ValidFrom:  rf.validFrom,
			Confidence: rf.fct.Confidence,
			Episodes:   []string{ep.ID},
		}
		if err := st.PutFact(newFact); err != nil {
			return err
		}
		stats.FactsAdded++
	}

	return nil
}

// mergeFact implements Rule 4's write for one match: current is the fact
// already stored under (src, relation, dst); incomingValidFrom and
// incomingConfidence come from the new episode's restatement of it.
//
// Episodes gets ep.ID appended (deduped) and Confidence takes the max of
// the two. ValidFrom is the earlier of the two: if the incoming ValidFrom
// is genuinely before the stored one (e.g. a backfilled episode), the
// record is relocated — since store.Fact's key embeds ValidFrom, that means
// deleting the old (src, relation, dst, oldValidFrom) key and re-putting
// the merged fact under the earlier one, so as-of queries between the two
// dates still see it. If the incoming ValidFrom is the same or later, the
// existing ValidFrom (and its key) is left untouched.
func mergeFact(st *store.Store, ep store.Episode, current store.Fact, incomingValidFrom time.Time, incomingConfidence float64, stats *Stats) error {
	if !containsString(current.Episodes, ep.ID) {
		current.Episodes = append(current.Episodes, ep.ID)
	}
	if incomingConfidence > current.Confidence {
		current.Confidence = incomingConfidence
	}

	if incomingValidFrom.Before(current.ValidFrom) {
		oldValidFrom := current.ValidFrom
		current.ValidFrom = incomingValidFrom
		if err := st.DeleteFact(current.Src, current.Relation, current.Dst, oldValidFrom); err != nil {
			return err
		}
	}

	if err := st.PutFact(current); err != nil {
		return err
	}
	stats.FactsMerged++
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
	at := clampInvalidAt(ep.OccurredAt, current.ValidFrom)
	if err := st.InvalidateFact(srcSlug, ref.Relation, dstSlug, current.ValidFrom, at); err != nil {
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

// relationIllegalRE matches runs of characters outside [a-z0-9_];
// relationRepeatRE collapses any resulting run of underscores (e.g. a
// legitimate "_" abutting a replaced illegal run) down to one.
var (
	relationIllegalRE = regexp.MustCompile(`[^a-z0-9_]+`)
	relationRepeatRE  = regexp.MustCompile(`_+`)
)

// normalizeRelation sanitizes an LLM-produced relation string at the one
// point it crosses from extract.Fct into the store: lowercased, trimmed,
// every run of characters outside [a-z0-9_] collapsed to a single '_', any
// resulting run of underscores collapsed again, and leading/trailing '_'
// trimmed. extract.Fct.Relation is unvalidated free text from the model —
// store.Fact's on-disk keys (factKey/adjKey in internal/memory/store) join
// (src, relation, dst, validFrom) with literal ':' separators, so a relation
// containing ':' would embed extra separators into the key and silently
// corrupt the adj: reverse index (FactsAbout's SplitN(rest, ":", 3)
// misparses it, so the fact quietly vanishes from that side of queries).
// A relation that normalizes to the empty string (e.g. pure punctuation like
// ":::") has nothing usable left; the caller skips the fact entirely rather
// than writing one with no relation.
func normalizeRelation(rel string) string {
	rel = strings.ToLower(strings.TrimSpace(rel))
	rel = relationIllegalRE.ReplaceAllString(rel, "_")
	rel = relationRepeatRE.ReplaceAllString(rel, "_")
	return strings.Trim(rel, "_")
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

// clampInvalidAt returns at, unless at falls before validFrom — in which
// case it returns validFrom instead, so a fact is never invalidated before
// it became valid (zero-width validity beats a negative one).
func clampInvalidAt(at, validFrom time.Time) time.Time {
	if at.Before(validFrom) {
		return validFrom
	}
	return at
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
