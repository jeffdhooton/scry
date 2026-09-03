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
	// FactsRejected counts facts dropped because both endpoints were
	// values (a number related to a status word says nothing about any
	// entity).
	FactsRejected int
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
	return ApplyWith(st, ep, cwd, res, exclusive, ApplyOptions{})
}

// ApplyOptions tunes Apply.
type ApplyOptions struct {
	// Force re-applies an episode the store already holds: facts merge onto
	// their current triples and entities refresh (repo refs, aliases). It
	// repairs episodes resolved under an older rule set.
	Force bool
}

// ApplyWith is Apply with options.
func ApplyWith(st *store.Store, ep store.Episode, cwd string, res extract.Result, exclusive map[string]bool, o ApplyOptions) (Stats, error) {
	var stats Stats

	// Rule 1: idempotency.
	has, err := st.HasEpisode(ep.ID)
	if err != nil {
		return Stats{}, err
	}
	if has && !o.Force {
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
	// A run artifact is not an identity. Storing one pollutes recall forever
	// and can never be usefully recalled later. Neither is a value: "main",
	// "in-progress", and "46 GiB" describe things, they are not things.
	if isEphemeralName(ent.Name) || isGenericEntityName(ent.Name) || IsValueName(ent.Name) {
		return nil
	}
	slug, found, err := st.ResolveAlias(ent.Name)
	if err != nil {
		return err
	}
	// A generic name must never pull an entity in by alias: that is the
	// runaway-merge path. Give it its own slug and let it stand alone.
	if found && isGenericAlias(ent.Name) {
		found = false
	}
	// A name that reaches an entity only through an alias, and names a
	// different kind of thing, is a different thing. "Mac mini" (machine)
	// must not merge into hermes-ops (project) because the project once
	// collected "mini" as an alias.
	if found && slug != store.Slugify(ent.Name) {
		owner, gerr := st.GetEntity(slug)
		if gerr == nil && !TypesCompatible(owner.Type, ent.Type) {
			found = false
		}
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

	// A name beats an alias. If another entity of an incompatible type
	// lists this name as an alias, that alias goes: otherwise the two
	// share it and the store grows a cross-type collision every time a new
	// entity is named after something else's nickname.
	if err := claimNameFromAliasHolders(st, ent.Name, slug, ent.Type); err != nil {
		return err
	}

	if errors.Is(err, store.ErrNotFound) {
		e := store.Entity{
			Slug:        slug,
			Name:        ent.Name,
			Type:        ent.Type,
			Description: ent.Description,
			CreatedAt:   ep.OccurredAt,
			LastSeen:    ep.OccurredAt,
		}
		aliases, err := admitAliases(st, e, keepDurable(ent.Aliases), ep.ID)
		if err != nil {
			return err
		}
		e.Aliases = aliases
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
	admitted, err := admitAliases(st, existing, keepDurable(ent.Aliases), ep.ID)
	if err != nil {
		return err
	}
	existing.Aliases = unionStrings(existing.Aliases, admitted)
	// A stub created from a fact endpoint is typed "concept" because nothing
	// knew better. The first mention that does know upgrades it.
	if existing.Type == "" || existing.Type == "concept" {
		if ent.Type != "" && ent.Type != "concept" {
			existing.Type = ent.Type
			// A wildcard-typed stub may have collected aliases a typed
			// entity may not hold (a machine's name on what is now a
			// project). Re-check them against the new type.
			if err := RevalidateAliases(st, &existing); err != nil {
				return err
			}
		}
	}
	// Fill the description, never replace it. Last-writer-wins let a
	// throwaway session in a scratch directory overwrite a real project's
	// description with an observation about the scratch directory. An
	// identity should be stable; correct it deliberately, not by mention.
	if existing.Description == "" && ent.Description != "" {
		existing.Description = ent.Description
	}
	if ep.OccurredAt.After(existing.LastSeen) {
		existing.LastSeen = ep.OccurredAt
	}
	if isWorkspacePath(cwd) {
		existing.RepoRefs = unionStrings(existing.RepoRefs, []string{cwd})
		if len(existing.RepoRefs) > maxRepoRefs {
			existing.RepoRefs = existing.RepoRefs[len(existing.RepoRefs)-maxRepoRefs:]
		}
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
	src, dst  string // dst is "" for an attribute fact
	value     string // set for an attribute fact
	rawRel    string // the model's relation when it differs from fct.Relation
	validFrom time.Time
	merged    bool
}

// keyDst is the store key slot for the fact's target.
func (rf resolvedFact) keyDst() string {
	if rf.dst != "" {
		return rf.dst
	}
	return store.AttrDst(rf.value)
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
		raw := normalizeRelation(fct.Relation)
		if raw == "" {
			continue
		}
		// The vocabulary is closed: the model's verb is mapped onto one of
		// the canonical relations, swapping endpoints when the raw form was
		// the inverse ("used_by" is "uses" the other way round).
		relation, flip := Map(raw)
		if flip {
			fct.Src, fct.Dst = fct.Dst, fct.Src
		}
		fct.Relation = relation
		rawRel := ""
		if raw != relation {
			rawRel = raw
		}

		// Values are never nodes. A fact whose target is a value becomes an
		// attribute of its source; a fact whose source is a value and whose
		// target is an entity is turned around; a fact between two values
		// is about nothing and is dropped.
		// An endpoint that is not an identity (a value, a run artifact, or
		// process vocabulary) must not become a node: before this, a fact
		// endpoint bypassed the entity checks entirely and
		// "setpoint-wt-lpj7ikz0 worktree" became an entity.
		srcIsValue := NotAnIdentity(fct.Src)
		dstIsValue := NotAnIdentity(fct.Dst)
		if relation == RelStatus {
			// "status" almost always points at a state word, and those are
			// attributes. When it points at a real identity the model meant
			// something else by it; keep the edge rather than turning a
			// project into this entity's status, which exclusivity would
			// then invalidate on the next status fact.
			if dstIsValue {
				dstIsValue = true
			} else {
				relation, fct.Relation = RelRelatedTo, RelRelatedTo
				if rawRel == "" {
					rawRel = raw
				}
			}
		}
		switch {
		case srcIsValue && dstIsValue:
			stats.FactsRejected++
			continue
		case srcIsValue:
			fct.Src, fct.Dst = fct.Dst, fct.Src
			dstIsValue = true
		}

		// Both endpoints are resolved (stubs created) before either can
		// veto the fact: an unresolvable src must not stop a valid dst from
		// existing.
		srcSlug, err := ensureEntitySlug(st, ep, fct.Src, stats)
		if err != nil {
			return err
		}
		rf := resolvedFact{fct: fct, src: srcSlug, rawRel: rawRel, validFrom: parseValidFrom(fct.ValidFrom, ep.OccurredAt)}
		if dstIsValue {
			rf.value = strings.TrimSpace(fct.Dst)
		} else {
			dstSlug, err := ensureEntitySlug(st, ep, fct.Dst, stats)
			if err != nil {
				return err
			}
			rf.dst = dstSlug
		}
		if rf.src == "" || (rf.dst == "" && rf.value == "") {
			continue
		}
		resolved = append(resolved, rf)
	}

	// Phase A — Rule 4: merge onto any pre-existing current fact with the
	// same triple, for every fact in the episode, before Phase B's
	// invalidations run.
	for i := range resolved {
		rf := &resolved[i]
		current, err := currentFact(st, rf.src, rf.fct.Relation, rf.keyDst())
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

		// Rule 6: exclusive relations invalidate any current fact with the
		// same (src, relation) but a different target — entity or value —
		// before the new one is added.
		if exclusive[rf.fct.Relation] {
			currents, err := st.FactsFrom(rf.src, false)
			if err != nil {
				return err
			}
			for _, f := range currents {
				if f.Relation == rf.fct.Relation && f.KeyDst() != rf.keyDst() {
					at := clampInvalidAt(ep.OccurredAt, f.ValidFrom)
					if err := st.InvalidateFact(f.Src, f.Relation, f.KeyDst(), f.ValidFrom, at); err != nil {
						return err
					}
					stats.FactsInvalidated++
				}
			}
		}

		newFact := store.Fact{
			Src:         rf.src,
			Relation:    rf.fct.Relation,
			Dst:         rf.dst,
			Value:       rf.value,
			RawRelation: rf.rawRel,
			Fact:        rf.fct.Fact,
			ValidFrom:   rf.validFrom,
			Confidence:  rf.fct.Confidence,
			Episodes:    []string{ep.ID},
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
		if err := st.DeleteFact(current.Src, current.Relation, current.KeyDst(), oldValidFrom); err != nil {
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
// any. ref.Relation is normalized the same way a fact's own Relation is
// (see normalizeRelation) before it's used to look up the target fact —
// stored relations are always normalized, so an un-normalized hint (e.g.
// "Deployed: On!" instead of the stored "deployed_on") would otherwise
// silently miss its target and never invalidate anything. A hint whose
// relation normalizes to empty is skipped outright, same as a fact whose
// own relation does.
func applySupersedes(st *store.Store, ep store.Episode, ref extract.SupRef, stats *Stats) error {
	raw := normalizeRelation(ref.Relation)
	if raw == "" {
		return nil
	}
	relation, flip := Map(raw)
	src, dst := ref.Src, ref.Dst
	if flip {
		src, dst = dst, src
	}
	if IsValueName(src) && !IsValueName(dst) && relation != RelStatus {
		src, dst = dst, src
	}

	srcSlug, err := resolveSlugOnly(st, src)
	if err != nil {
		return err
	}
	if srcSlug == "" {
		return nil
	}
	var keyDst string
	if relation == RelStatus || IsValueName(dst) {
		keyDst = store.AttrDst(dst)
	} else {
		dstSlug, err := resolveSlugOnly(st, dst)
		if err != nil {
			return err
		}
		if dstSlug == "" {
			return nil
		}
		keyDst = dstSlug
	}

	current, err := currentFact(st, srcSlug, relation, keyDst)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	at := clampInvalidAt(ep.OccurredAt, current.ValidFrom)
	if err := st.InvalidateFact(srcSlug, relation, keyDst, current.ValidFrom, at); err != nil {
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
// (src, relation, key dst) triple, or nil if none exists. keyDst is an
// entity slug or store.AttrDst(value).
func currentFact(st *store.Store, src, relation, keyDst string) (*store.Fact, error) {
	facts, err := st.FactsFrom(src, false)
	if err != nil {
		return nil, err
	}
	for i := range facts {
		if facts[i].Relation == relation && facts[i].KeyDst() == keyDst {
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
// paths such as "/tmp/...". Whether the path is a repository is attested
// by the distiller on the machine that has it (RawEpisode.CwdIsRepo); the
// daemon resolving the episode may be on another machine, so it cannot
// stat the path and must not try.
func isWorkspacePath(cwd string) bool {
	if cwd == "" || !strings.HasPrefix(cwd, "/Users/") {
		return false
	}
	for _, bad := range []string{"/tmp/", "/private/tmp/", "/var/folders/", "/private/var/folders/", "/Library/Caches/"} {
		if strings.Contains(cwd, bad) {
			return false
		}
	}
	return true
}

// AddRepoRef unions cwd into refs when it looks like a workspace path,
// keeping the newest maxRepoRefs. Shared with the daemon's repo-ref
// repair so both obey one rule.
func AddRepoRef(refs []string, cwd string) []string {
	if !isWorkspacePath(cwd) {
		return refs
	}
	refs = unionStrings(refs, []string{cwd})
	if len(refs) > maxRepoRefs {
		refs = refs[len(refs)-maxRepoRefs:]
	}
	return refs
}

// maxRepoRefs caps how many repositories one entity may claim. An entity
// that genuinely spans more than a handful of repos is not an entity, it is
// a category, and listing twenty paths helps nobody.
const maxRepoRefs = 6

// ephemeralPatterns match names that are artifacts of a single run rather
// than durable identities: temp worktrees, scratch directories, and bare
// hex ids. These were being promoted to permanent entities and aliases —
// `setpoint-wt-9e6jz82r` became an alias of a real project, and a throwaway
// `/tmp` experiment became an entity.
var (
	tempWorktreeRe = regexp.MustCompile(`(?i)-wt-[a-z0-9]{6,}$|^setpoint-wt-|^loom-wt-`)
	bareHexRe      = regexp.MustCompile(`^[0-9a-f]{8,}$`)
	// Session/thread UUIDs: durable to nothing, and they were being stored as
	// aliases of real agents.
	uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func isEphemeralName(name string) bool {
	n := strings.TrimSpace(name)
	if n == "" {
		return true
	}
	if strings.HasPrefix(n, "/tmp/") || strings.HasPrefix(n, "/private/tmp/") ||
		strings.HasPrefix(n, "/private/var/folders/") || strings.HasPrefix(n, "/var/folders/") {
		return true
	}
	if tempWorktreeRe.MatchString(n) {
		return true
	}
	if bareHexRe.MatchString(strings.ToLower(n)) || uuidRe.MatchString(n) {
		return true
	}
	return false
}

// genericAliases are role words, not identities. They were the mechanism of
// the worst corruption in the graph: once "workspace" and "orchestrator"
// became aliases of one project, every later session that said "the
// workspace" merged into it, fusing three unrelated projects into a single
// entity that claimed four repos.
//
// A role describes what something is doing right now. An identity is what it
// IS. Only identities belong here.
var genericAliases = map[string]bool{
	"agent": true, "app": true, "the app": true, "application": true,
	"cli": true, "codebase": true, "current directory": true, "cwd": true,
	"current worktree": true, "daemon": true, "dir": true, "directory": true,
	"engine": true, "executor": true, "here": true, "loop": true,
	"main": true, "module": true, "north star": true, "orchestrator": true,
	"package": true, "project": true, "the project": true, "repo": true,
	"the repo": true, "repo root": true, "repository": true, "root": true,
	"runner": true, "server": true, "service": true, "session": true,
	"task orchestrator": true, "room orchestrator": true, "tool": true,
	"this repo": true, "worktree": true, "workspace": true,
	"operations app": true, "ops app": true, "the fleet": true,
	"the mini": false, // a real machine nickname — kept deliberately
}

// processNouns are the vocabulary of doing work rather than the names of
// things. Stored as entities they become universal magnets: once "plan"
// exists, every design document in the graph carries it as an alias and
// "collides" with it, and the collision signal becomes useless.
var processNouns = map[string]bool{
	"approved": true, "bug": true, "branch": true, "change": true,
	"commit": true, "design": true, "doc": true, "docs": true,
	"error": true, "failed": true, "failure": true, "feature": true,
	"fix": true, "issue": true, "iteration": true, "note": true,
	"notes": true, "output": true, "passed": true, "phase": true,
	"plan": true, "pr": true, "private": true, "public": true,
	"report": true, "result": true, "results": true, "review": true,
	"spec": true, "status": true, "step": true, "task": true,
	"test": true, "tests": true, "ticket": true, "warning": true,
	"success": true, "done": true, "pending": true, "blocked": true,
}

// numberedStepRe matches "task 2", "phase 1 plan", "step 3", "iteration 4":
// positions in a process, not identities.
var bareNumberRe = regexp.MustCompile(`^\d{1,6}$`)

var numberedStepRe = regexp.MustCompile(`(?i)^(task|phase|step|iteration|wave|round|part)\s+\d+`)

// isGenericEntityName reports whether a name is process vocabulary rather
// than an identity, and so must never be stored as an entity at all.
func isGenericEntityName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.TrimPrefix(n, "the ")
	if n == "" {
		return true
	}
	if processNouns[n] {
		return true
	}
	// A bare number is never an identity. "87" and "88" were entities, each
	// carrying a PR number and a build command as aliases.
	if bareNumberRe.MatchString(n) {
		return true
	}
	if numberedStepRe.MatchString(n) {
		return true
	}
	// "implementation plan", "rollout plan", "test plan" — a qualifier on a
	// process noun is still a process noun.
	if fields := strings.Fields(n); len(fields) == 2 && processNouns[fields[1]] {
		return true
	}
	// Role words, but NOT the sub-path rule below: "packages/shared" and
	// "internal/memory" are real modules you can say things about. That
	// rule exists to stop a directory becoming an ALIAS that merges two
	// projects, which is a different question from whether it is a thing.
	if v, ok := genericAliases[n]; ok {
		return v
	}
	return false
}

// isGenericAlias reports whether a name is a role rather than an identity,
// and so must never be stored as an alias or used to merge entities.
func isGenericAlias(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return true
	}
	if v, ok := genericAliases[n]; ok {
		return v
	}
	// A sub-path is a location, not an identity: "apps/operations" is a
	// directory that exists in more than one repo.
	if strings.Contains(n, "/") && !strings.Contains(n, ".") &&
		!strings.HasPrefix(n, "/") && len(strings.Split(n, "/")) == 2 {
		// Keep owner/repo forms (jeffdhooton/setpoint); drop path forms
		// whose first segment is a common source directory.
		head := strings.Split(n, "/")[0]
		switch head {
		case "apps", "src", "internal", "cmd", "packages", "lib", "pkg",
			"setpoint", "tests", "docs", "scripts":
			return true
		}
	}
	return false
}

// keepDurable drops ephemeral names from a candidate alias list.
func keepDurable(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if isEphemeralName(n) || isGenericAlias(n) || IsValueName(n) {
			continue
		}
		out = append(out, n)
	}
	return out
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

// claimNameFromAliasHolders removes name from the aliases of any entity of
// an incompatible type that lists it, so the entity actually called that
// owns it.
func claimNameFromAliasHolders(st *store.Store, name, slug, typ string) error {
	owner, ok, err := st.ResolveAlias(name)
	if err != nil || !ok || owner == slug {
		return err
	}
	holder, err := st.GetEntity(owner)
	if err != nil {
		return nil
	}
	if TypesCompatible(holder.Type, typ) || store.Normalize(holder.Name) == store.Normalize(name) {
		return nil
	}
	kept := holder.Aliases[:0]
	dropped := false
	for _, a := range holder.Aliases {
		if store.Normalize(a) == store.Normalize(name) {
			dropped = true
			continue
		}
		kept = append(kept, a)
	}
	if !dropped {
		return nil
	}
	holder.Aliases = kept
	if err := st.PutEntity(holder); err != nil {
		return err
	}
	return st.ClaimAlias(name, slug)
}
