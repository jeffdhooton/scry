package resolve

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/store"
)

// HygieneReport summarises what a Hygiene pass changed.
type HygieneReport struct {
	EntitiesScanned int `json:"entities_scanned"`
	EntitiesChanged int `json:"entities_changed"`
	AliasesDropped  int `json:"aliases_dropped"`
	RepoRefsDropped int `json:"repo_refs_dropped"`
	// AliasesSplit counts aliases removed from an entity because another
	// entity of an incompatible type owns them, or because they are
	// another entity's own name.
	AliasesSplit int `json:"aliases_split"`
	// FactsReattached counts facts moved from an entity to the entity whose
	// name the fact text actually mentions.
	FactsReattached int `json:"facts_reattached"`
	// SelfLoopsInvalidated counts current facts whose src and dst were the
	// same entity.
	SelfLoopsInvalidated int `json:"self_loops_invalidated"`
	// CrossTypeCollisions is the number of aliases shared by entities of
	// incompatible types after the pass. It must be zero after an apply.
	CrossTypeCollisions int `json:"cross_type_collisions"`
	// Conflated are entities carrying an alias that is ALSO another entity's
	// own name, of a compatible type, after the pass. Reported only; a
	// compatible-type overlap may be a genuine duplicate that a human
	// should merge.
	Conflated []ConflationReport `json:"conflated,omitempty"`
	// EphemeralEntities are entities whose own name is a run artifact. They
	// are REPORTED, not deleted: facts reference entities by slug as src and
	// dst, so removing one would orphan its facts.
	EphemeralEntities []string `json:"ephemeral_entities,omitempty"`
	DroppedAliasList  []string `json:"dropped_aliases,omitempty"`
	Reattachments     []string `json:"reattachments,omitempty"`
}

// ConflationReport names an entity that carries another entity's name as an
// alias — two identities fused into one.
type ConflationReport struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	CollidesWith []string `json:"collides_with"`
}

// Hygiene repairs entities that were written before the write path had
// its current rules. The prevention fixes stop new pollution; this cleans
// what is already stored:
//
//   - aliases that are run artifacts, role words, values, or references
//     ("the machine", "you")
//   - aliases that are another entity's own name, or that an entity of an
//     incompatible type owns: the alias is split off, and current facts on
//     the entity whose text names the other entity move to it
//   - repo refs pointing at directories that are no longer repositories,
//     and refs beyond maxRepoRefs
//   - self-loop facts, invalidated
//   - entities whose own name is a run artifact (reported, not deleted)
//
// dryRun reports without writing. Callers that apply must have taken a
// backup: the daemon's memory.hygiene handler does.
func Hygiene(st *store.Store, dryRun bool) (HygieneReport, error) {
	var rep HygieneReport
	entities, err := st.Entities()
	if err != nil {
		return rep, err
	}
	bySlug := make(map[string]store.Entity, len(entities))
	realSlugs := make(map[string]string, len(entities))
	for _, e := range entities {
		bySlug[e.Slug] = e
		if isEphemeralName(e.Name) || isEphemeralName(e.Slug) || isGenericEntityName(e.Name) || IsValueName(e.Name) {
			continue
		}
		realSlugs[e.Slug] = e.Name
	}
	// nameTokens lets an alias that is a whole-token subset of another
	// entity's name ("mini" inside "Mac mini") find that entity even when
	// nobody listed the alias verbatim.
	nameTokens := map[string]map[string]bool{}
	for slug := range realSlugs {
		nameTokens[slug] = tokensOf(bySlug[slug].Name)
	}
	subsetOwner := func(alias string, e store.Entity) string {
		at := tokensOf(alias)
		if len(at) == 0 {
			return ""
		}
		best, bestSize := "", 0
		for slug, nt := range nameTokens {
			if slug == e.Slug || TypesCompatible(bySlug[slug].Type, e.Type) {
				continue
			}
			ok := true
			for t := range at {
				if !nt[t] {
					ok = false
					break
				}
			}
			if ok && (best == "" || len(nt) < bestSize) {
				best, bestSize = slug, len(nt)
			}
		}
		return best
	}
	degree := func(slug string) int {
		facts, err := st.FactsAbout(slug, false)
		if err != nil {
			return 0
		}
		return len(facts)
	}

	// Pass 1: who lists which alias. Keys are normalized aliases.
	owners := map[string][]string{}
	for _, e := range entities {
		for _, a := range e.Aliases {
			norm := store.Normalize(a)
			if norm == "" || norm == store.Normalize(e.Name) {
				continue
			}
			owners[norm] = append(owners[norm], e.Slug)
		}
	}
	// keeper decides which of several owners keeps a shared alias: the one
	// whose own name it is, else one sharing a token with it, else the
	// highest degree.
	keeper := func(norm string, slugs []string) string {
		if s, ok := realSlugs[store.Slugify(norm)]; ok && s != "" {
			return store.Slugify(norm)
		}
		for _, s := range slugs {
			if sharesToken(norm, bySlug[s].Name) {
				return s
			}
		}
		best, bestDeg := "", -1
		for _, s := range slugs {
			if d := degree(s); d > bestDeg {
				best, bestDeg = s, d
			}
		}
		return best
	}

	now := time.Now()
	for _, e := range entities {
		rep.EntitiesScanned++

		if isEphemeralName(e.Name) || isEphemeralName(e.Slug) || isGenericEntityName(e.Name) {
			// Flagged, never removed — its facts would be orphaned.
			rep.EphemeralEntities = append(rep.EphemeralEntities, e.Name)
		}

		changed := false
		kept := make([]string, 0, len(e.Aliases))
		for _, a := range e.Aliases {
			norm := store.Normalize(a)
			if neverAlias(a) || norm == store.Normalize(e.Name) {
				rep.AliasesDropped++
				rep.DroppedAliasList = append(rep.DroppedAliasList, e.Slug+": "+a)
				changed = true
				continue
			}

			// Another entity's own name, or an alias an incompatible-type
			// entity also lists: split it off this entity, moving the facts
			// that plainly meant the other entity.
			other := ""
			if s, ok := realSlugs[store.Slugify(a)]; ok && store.Slugify(a) != e.Slug && s != "" {
				other = store.Slugify(a)
			}
			split := other != ""
			if !split {
				for _, o := range owners[norm] {
					if o != e.Slug && !TypesCompatible(bySlug[o].Type, e.Type) && keeper(norm, owners[norm]) != e.Slug {
						split, other = true, keeper(norm, owners[norm])
						break
					}
				}
			}
			if !split {
				if o := subsetOwner(a, e); o != "" {
					split, other = true, o
				}
			}
			if split {
				if other != "" && other != e.Slug {
					n, moves, err := reattachByMention(st, e, a, other, now, dryRun)
					if err != nil {
						return rep, err
					}
					rep.FactsReattached += n
					rep.Reattachments = append(rep.Reattachments, moves...)
				}
				rep.AliasesSplit++
				rep.DroppedAliasList = append(rep.DroppedAliasList, e.Slug+": "+a+" → "+other)
				changed = true
				continue
			}
			kept = append(kept, a)
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
		e.Aliases = kept
		e.RepoRefs = keptRefs
		if err := st.PutEntity(e); err != nil {
			return rep, err
		}
	}

	// Self-loops: a fact from an entity to itself says nothing.
	all, err := st.AllFacts()
	if err != nil {
		return rep, err
	}
	for _, f := range all {
		if f.InvalidAt != nil || f.Dst == "" || f.Src != f.Dst {
			continue
		}
		rep.SelfLoopsInvalidated++
		if dryRun {
			continue
		}
		if err := st.InvalidateFact(f.Src, f.Relation, f.KeyDst(), f.ValidFrom, now); err != nil {
			return rep, err
		}
	}

	// Ownership pass: every entity's own name resolves to itself, and every
	// kept alias resolves to its keeper. Then count what is left.
	if !dryRun {
		entities, err = st.Entities()
		if err != nil {
			return rep, err
		}
		for _, e := range entities {
			if err := st.ClaimAlias(e.Name, e.Slug); err != nil {
				return rep, err
			}
		}
		for _, e := range entities {
			for _, a := range e.Aliases {
				if err := st.ClaimAlias(a, e.Slug); err != nil {
					return rep, err
				}
			}
		}
	}
	collisions, conflated := auditAliases(st, entities, realSlugs, dryRun)
	rep.CrossTypeCollisions = collisions
	rep.Conflated = conflated
	return rep, nil
}

// auditAliases counts aliases shared across incompatible types and lists
// compatible-type overlaps with another entity's own name. In a dry run
// the entities passed in still carry the aliases the pass would drop, so
// the count is computed on the post-drop view.
func auditAliases(st *store.Store, entities []store.Entity, realSlugs map[string]string, dryRun bool) (int, []ConflationReport) {
	type owner struct{ slug, typ string }
	byAlias := map[string][]owner{}
	var conflated []ConflationReport
	for _, e := range entities {
		var collisions []string
		for _, a := range e.Aliases {
			norm := store.Normalize(a)
			if norm == "" || norm == store.Normalize(e.Name) {
				continue
			}
			if dryRun && (neverAlias(a) || realSlugs[store.Slugify(a)] != "" && store.Slugify(a) != e.Slug) {
				continue // would be dropped or split
			}
			byAlias[norm] = append(byAlias[norm], owner{e.Slug, e.Type})
			if name, ok := realSlugs[store.Slugify(a)]; ok && store.Slugify(a) != e.Slug {
				collisions = append(collisions, name)
			}
		}
		if len(collisions) > 0 {
			conflated = append(conflated, ConflationReport{Slug: e.Slug, Name: e.Name, CollidesWith: collisions})
		}
	}
	cross := 0
	for _, os := range byAlias {
		if len(os) < 2 {
			continue
		}
		for i := range os {
			for j := i + 1; j < len(os); j++ {
				if !TypesCompatible(os[i].typ, os[j].typ) {
					cross++
				}
			}
		}
	}
	sort.Slice(conflated, func(i, j int) bool { return conflated[i].Slug < conflated[j].Slug })
	return cross, conflated
}

// reattachByMention moves current facts on e whose text mentions alias as
// a whole word — and does not mention e's own name — to entity other.
func reattachByMention(st *store.Store, e store.Entity, alias, other string, now time.Time, dryRun bool) (int, []string, error) {
	facts, err := st.FactsAbout(e.Slug, false)
	if err != nil {
		return 0, nil, err
	}
	aliasRE := wordRE(alias)
	selfRE := wordRE(e.Name)
	slugRE := wordRE(e.Slug)
	moved := 0
	var moves []string
	for _, f := range facts {
		if f.Src == f.Dst {
			continue
		}
		text := f.Fact
		if !aliasRE.MatchString(text) || selfRE.MatchString(text) || slugRE.MatchString(text) {
			continue
		}
		updated := f
		if f.Src == e.Slug {
			updated.Src = other
		}
		if f.Dst == e.Slug {
			updated.Dst = other
		}
		if updated.Src == updated.Dst {
			continue
		}
		moved++
		moves = append(moves, e.Slug+" → "+other+": "+truncateText(f.Fact, 80))
		if dryRun {
			continue
		}
		if err := st.RelocateFact(f, updated); err != nil {
			return moved, moves, err
		}
	}
	return moved, moves, nil
}

// wordRE matches name as a whole word, case-insensitively, treating "-",
// "_", and spaces as equivalent separators.
func wordRE(name string) *regexp.Regexp {
	parts := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), "-", " "), "_", " "))
	if len(parts) == 0 {
		return regexp.MustCompile(`$^`)
	}
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile(`(?i)(?:^|[^a-z0-9])` + strings.Join(quoted, `[\s_-]+`) + `(?:$|[^a-z0-9])`)
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
