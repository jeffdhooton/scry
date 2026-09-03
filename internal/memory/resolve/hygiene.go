package resolve

import (
	"fmt"
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
	// CrossTypeCollisions is the number of pairs of entities of different
	// kinds that answer to the same name, counting every spelling: the
	// entity's own name, its slug, and its aliases. It counts what the
	// store holds, not what would be left after the pass.
	CrossTypeCollisions int `json:"cross_type_collisions"`
	// CollisionSample shows the first of them, so the number can be read
	// rather than trusted.
	CollisionSample []string `json:"collision_sample,omitempty"`
	// EntitiesUnreferenced counts entities no fact mentions. They are
	// reported, never removed: removing them cost spellings the store
	// answered to, and they are counted out of the collision audit
	// instead.
	EntitiesUnreferenced int `json:"entities_unreferenced"`
	// StubClaimsDropped counts names an untyped stub was holding that a
	// typed entity also answers to.
	StubClaimsDropped int `json:"stub_claims_dropped"`
	// StubsMerged counts untyped stubs folded into the typed entity they
	// were a second spelling of.
	StubsMerged int `json:"stubs_merged"`
	// StubMergeSample names the first of those merges.
	StubMergeSample []string `json:"stub_merge_sample,omitempty"`
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
	// Duplicate spellings first: an untyped stub folded into the typed
	// entity it repeats takes its facts with it, and the alias passes
	// below then see one entity where there were two.
	merged, mergeSample, err := mergeDuplicateStubs(st, entities, dryRun)
	if err != nil {
		return rep, err
	}
	rep.StubsMerged, rep.StubMergeSample = merged, mergeSample

	// A thing named by where it is, is that thing. "Mac mini at
	// 100.96.45.73" is the Mac mini, and it had survived four rounds of
	// grading as a second machine holding one fact, because two typed
	// entities are never merged on a name alone and nothing else looked
	// at the address.
	located, locatedSample, err := mergeLocatedDuplicates(st, dryRun)
	if err != nil {
		return rep, err
	}
	rep.StubsMerged += located
	rep.StubMergeSample = append(rep.StubMergeSample, locatedSample...)
	if located > 0 && !dryRun {
		if entities, err = st.Entities(); err != nil {
			return rep, err
		}
	}
	if merged > 0 && !dryRun {
		if entities, err = st.Entities(); err != nil {
			return rep, err
		}
		// The name index is cached, and the merge just removed entities
		// from it. Without this refresh the passes below hand aliases and
		// facts to slugs that no longer exist, which is where several
		// hundred facts pointing at nothing came from.
		if err := RefreshCompactIndex(st); err != nil {
			return rep, err
		}
	}
	bySlug := make(map[string]store.Entity, len(entities))
	realSlugs := make(map[string]string, len(entities))
	// byForm finds a real entity by any spelling of its name: slug,
	// normalized name ("scry_status" and "scry status" both → scry-status),
	// or compact ("halo1" for "halo-1").
	byForm := map[string]string{}
	for _, e := range entities {
		bySlug[e.Slug] = e
		if isEphemeralName(e.Name) || isEphemeralName(e.Slug) || isGenericEntityName(e.Name) || IsValueName(e.Name) {
			continue
		}
		realSlugs[e.Slug] = e.Name
		byForm[e.Slug] = e.Slug
		byForm[store.Normalize(e.Name)] = e.Slug
		byForm[compact(e.Slug)] = e.Slug
	}
	// nameOf returns the real entity a name spells, if any.
	nameOf := func(name string) string {
		if s, ok := byForm[store.Slugify(name)]; ok {
			return s
		}
		if s, ok := byForm[store.Normalize(name)]; ok {
			return s
		}
		if s, ok := byForm[compact(store.Slugify(name))]; ok {
			return s
		}
		return ""
	}
	// nameTokens lets an alias that is a whole-token subset of another
	// entity's name ("mini" inside "Mac mini") find that entity even when
	// nobody listed the alias verbatim.
	nameTokens := map[string]map[string]bool{}
	for slug := range realSlugs {
		nameTokens[slug] = tokensOf(bySlug[slug].Name)
	}
	// factTexts caches an entity's current fact sentences for the mention
	// counts below.
	factTexts := map[string][]string{}
	textsOf := func(slug string) []string {
		if t, ok := factTexts[slug]; ok {
			return t
		}
		facts, _ := st.FactsAbout(slug, false)
		t := make([]string, 0, len(facts))
		for _, f := range facts {
			t = append(t, f.Fact)
		}
		factTexts[slug] = t
		return t
	}
	mentions := func(name string, texts []string) int {
		re := wordRE(name)
		n := 0
		for _, t := range texts {
			if re.MatchString(t) {
				n++
			}
		}
		return n
	}
	subsetOwner := func(alias string, e store.Entity) string {
		at := tokensOf(alias)
		if len(at) == 0 || !hasSpecificToken(alias) {
			return ""
		}
		// The entity's own name in another spelling ("safe-ai" on safeai,
		// "deepresearch/agent.py" on deepresearch-agent-py) is nobody
		// else's.
		if c := compact(alias); c == compact(e.Slug) || c == compact(e.Name) {
			return ""
		}
		// Another entity's whole name plus that entity's kind words names
		// that entity: "Hermes agent" is the service Hermes even on a
		// project called hermes-ops. This is decided before the own-name
		// guard below, because the shared token IS the other's name.
		for slug, nt := range nameTokens {
			if slug == e.Slug || len(nt) == 0 || TypesCompatible(bySlug[slug].Type, e.Type) {
				continue
			}
			kw := kindWords[bySlug[slug].Type]
			ok := true
			for t := range nt {
				if !at[t] {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			extras := 0
			for t := range at {
				if !nt[t] {
					if !kw[t] {
						ok = false
						break
					}
					extras++
				}
			}
			if ok && extras > 0 {
				return slug
			}
		}
		// An alias that echoes the entity's own name ("jeffdhooton" on Jeff,
		// "Qwen38-27B" on the Qwen model) belongs to it; only a stranger is
		// a candidate for another owner.
		if sharesToken(alias, e.Name) {
			return ""
		}
		texts := textsOf(e.Slug)
		best, bestMentions := "", 0
		for slug, nt := range nameTokens {
			if slug == e.Slug {
				continue
			}
			compatible := TypesCompatible(bySlug[slug].Type, e.Type)
			if !aliasNamesOther(at, nt, compatible) {
				continue
			}
			// Among candidates, the one this entity's own facts actually
			// talk about; a candidate nobody mentions still wins over none.
			m := mentions(bySlug[slug].Name, texts)
			if best == "" || m > bestMentions || (m == bestMentions && slug < best) {
				best, bestMentions = slug, m
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
	// keeper decides, once per alias, which owner keeps it: the entity
	// whose own name it is; else, among owners whose name shares a token
	// with it ("Mac mini" for "mini"), the highest degree; else the highest
	// degree. Every other owner of an incompatible type loses it.
	keeperCache := map[string]string{}
	keeper := func(norm string, slugs []string) string {
		if k, ok := keeperCache[norm]; ok {
			return k
		}
		k := ""
		if s := nameOf(norm); s != "" {
			k = s
		}
		// An owner whose own name plus its kind words compose the alias is
		// the thing the alias names ("Hermes agent" is the service Hermes,
		// not the project hermes-ops, whatever their degrees).
		if k == "" {
			at := tokensOf(norm)
			for _, s := range slugs {
				nt := tokensOf(bySlug[s].Name)
				if len(nt) == 0 || !subsetOf(nt, at) {
					continue
				}
				extrasOK := true
				for t := range at {
					if !nt[t] && !kindWords[bySlug[s].Type][t] {
						extrasOK = false
						break
					}
				}
				if extrasOK {
					k = s
					break
				}
			}
		}
		if k == "" {
			best, bestDeg := "", -1
			for _, s := range slugs {
				if sharesToken(norm, bySlug[s].Name) {
					if d := degree(s); d > bestDeg {
						best, bestDeg = s, d
					}
				}
			}
			if best == "" {
				for _, s := range slugs {
					if d := degree(s); d > bestDeg {
						best, bestDeg = s, d
					}
				}
			}
			k = best
		}
		keeperCache[norm] = k
		return k
	}
	// target picks where a losing entity's facts go: the co-owner its own
	// facts mention most, else the keeper.
	target := func(norm string, slugs []string, e store.Entity) string {
		texts := textsOf(e.Slug)
		best, bestMentions := "", 0
		for _, s := range slugs {
			if s == e.Slug || TypesCompatible(bySlug[s].Type, e.Type) {
				continue
			}
			if m := mentions(bySlug[s].Name, texts); m > bestMentions {
				best, bestMentions = s, m
			}
		}
		if best != "" {
			return best
		}
		return keeper(norm, slugs)
	}

	// grants are aliases split off one entity that name another: the
	// other gets them, so "Hermes agent" still resolves after it leaves
	// hermes-ops.
	grants := map[string][]string{}
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
			if s := nameOf(a); s != "" && s != e.Slug {
				other = s
			}
			split := other != ""
			if !split {
				for _, o := range owners[norm] {
					if o != e.Slug && !TypesCompatible(bySlug[o].Type, e.Type) {
						if keeper(norm, owners[norm]) != e.Slug {
							split, other = true, target(norm, owners[norm], e)
						}
						break
					}
				}
			}
			if !split {
				if o := subsetOwner(a, e); o != "" {
					split, other = true, o
				}
			}
			// An alias the write path would refuse today does not get to
			// stay because it was admitted yesterday. Cleanup and
			// prevention have to agree, or the store keeps what the rule
			// forbids: hermes-ops held "Hermes tmux", "Hermes Slack
			// gateway", and "Jeff's own Hermes" long after the rule that
			// admitted them was replaced.
			if !split && (machineLeak(a, e) || roleLeak(a, e)) {
				// Hardware named on a non-machine, or a role named on a
				// person, with no better owner to hand it to: the alias
				// goes and the facts stay.
				rep.AliasesDropped++
				rep.DroppedAliasList = append(rep.DroppedAliasList, e.Slug+": "+a+" (names another kind of thing)")
				changed = true
				continue
			}
			// Applying the write path's naming rule to stored aliases was
			// tried twice and measured worse than not doing it, both
			// times. The first attempt handed 4,634 aliases to entities
			// whose facts never mention them; the corrected rule handed
			// 7,268, rebuilt a magnet entity from zero to 104 aliases,
			// and destroyed 1,471 spellings the store used to answer to,
			// two of them the Mac mini's. One decision about one alias
			// can be judged; nineteen thousand of them compound every
			// weakness in the rule at once. Cleanup and prevention
			// disagreeing on an alias admitted under an older rule is the
			// lesser problem, and it is the one this keeps.
			if _, live := bySlug[other]; split && other != "" && !live {
				// Whatever named this alias is gone: keep the alias where
				// it is rather than handing it to nothing.
				split, other = false, ""
			}
			if split {
				// Facts move across a type boundary (a machine's facts on a
				// project are plainly misfiled), or to an entity of the same
				// type whose exact name the alias is and which shares nothing
				// with this entity's name (gpt-oss-120b on the Qwen model).
				// "childscribe" facts on childscribe-laravel may be about
				// either, and guessing is worse than a stray alias.
				exactStranger := other != "" && other == store.Slugify(a) && !sharesToken(a, e.Name)
				if other != "" && other != e.Slug && (!TypesCompatible(bySlug[other].Type, e.Type) || exactStranger) {
					n, moves, err := reattachByMention(st, e, a, other, now, dryRun)
					if err != nil {
						return rep, err
					}
					rep.FactsReattached += n
					rep.Reattachments = append(rep.Reattachments, moves...)
				}
				rep.AliasesSplit++
				rep.DroppedAliasList = append(rep.DroppedAliasList, e.Slug+": "+a+" → "+other)
				// Hand the alias over only if the recipient would keep it.
				// An alias that names a third entity comes straight back
				// out on the next pass, and the two passes trade it
				// forever, moving facts each time.
				if other != "" && other != e.Slug && store.Normalize(a) != store.Normalize(bySlug[other].Name) && !neverAlias(a) && !machineLeak(a, bySlug[other]) && !roleLeak(a, bySlug[other]) {
					named, err := namedByKindWords(st, a, bySlug[other])
					if err != nil {
						return rep, err
					}
					if named == "" || named == other {
						grants[other] = append(grants[other], a)
					}
				}
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
			if !isWorkspacePath(r) {
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

	// Hand split aliases to the entities they name.
	if !dryRun {
		for slug, names := range grants {
			e, err := st.GetEntity(slug)
			if err != nil {
				continue
			}
			e.Aliases = unionStrings(e.Aliases, names)
			if err := st.PutEntity(e); err != nil {
				return rep, err
			}
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
			for _, a := range e.Aliases {
				if err := st.ClaimAlias(a, e.Slug); err != nil {
					return rep, err
				}
			}
		}
		// Names last, so an entity's own name always resolves to itself.
		for _, e := range entities {
			if err := st.ClaimAlias(e.Name, e.Slug); err != nil {
				return rep, err
			}
		}
	}
	// Entities nothing says anything about used to be pruned here. It
	// cost 3,215 spellings the store answered to — scry-episodes,
	// 10g-switch, gemini-2.5-pro — for the sake of removing nodes that
	// were only ever noise in a count. They stay, and the audit below
	// counts them out instead.
	referenced, err := referencedSlugs(st)
	if err != nil {
		return rep, err
	}
	rep.EntitiesUnreferenced = 0
	for _, e := range entities {
		if !referenced[e.Slug] {
			rep.EntitiesUnreferenced++
		}
	}

	// An untyped stub never keeps a name a typed entity also answers to.
	// The write path refuses that outright; a stub that collected one
	// earlier keeps it, and every one of the store's identical-alias
	// collisions has a concept on one side. Dropping is safe where
	// transferring was not: the name still resolves, to the entity that
	// has a type.
	dropped, err := dropStubClaims(st, dryRun)
	if err != nil {
		return rep, err
	}
	rep.AliasesDropped += dropped
	rep.StubClaimsDropped = dropped

	// Re-read: an apply changed the aliases, and the audit must count what
	// the store holds now rather than the view the pass started from.
	audited := entities
	if !dryRun {
		if fresh, err := st.Entities(); err == nil {
			audited = fresh
		}
	}
	collisions, sample, conflated := auditNames(audited, realSlugs, referenced)
	rep.CrossTypeCollisions = collisions
	rep.CollisionSample = sample
	rep.Conflated = conflated
	return rep, nil
}

// foldName reduces a name to the form two entities would have to share
// for a reader to call them the same: case, punctuation, spacing, and
// plurals all folded away. "Mac mini", "mac-mini", "macmini", and "Mac
// minis" fold to one string, and so do "halo/flashnext" and
// "halo-flashnext".
func foldName(s string) string {
	var b strings.Builder
	for _, t := range strings.Fields(nonAlnumRE.ReplaceAllString(strings.ToLower(s), " ")) {
		b.WriteString(singular(t))
	}
	return b.String()
}

// auditNames counts the names two entities both answer to. Every spelling
// counts — an entity's own name, its slug, and each alias — because a
// project named the same as a machine is the same fusion as a project
// carrying the machine's name as an alias.
//
// Nothing is subtracted for being about to be cleaned. An earlier version
// skipped, in a dry run, every alias the pass believed it would drop, and
// so could not report a collision it thought it would fix; it returned
// zero for two entities sharing a name byte for byte. A measurement that
// cannot come out nonzero is not a measurement.
//
// A concept is not treated as compatible with a real type here, however
// permissive admission is: a concept stub sharing a machine's name is how
// a machine's name reaches a project, and counting it as harmless is what
// let 370 of them accumulate.
func auditNames(entities []store.Entity, realSlugs map[string]string, referenced map[string]bool) (int, []string, []ConflationReport) {
	type owner struct{ slug, typ, spelling string }
	byName := map[string][]owner{}
	var conflated []ConflationReport
	for _, e := range entities {
		// An entity no fact mentions is not a second identity for a name.
		if len(referenced) > 0 && !referenced[e.Slug] {
			continue
		}
		var collisions []string
		seen := map[string]bool{}
		for _, spelling := range append([]string{e.Name, e.Slug}, e.Aliases...) {
			key := foldName(spelling)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			byName[key] = append(byName[key], owner{e.Slug, e.Type, spelling})
		}
		for _, a := range e.Aliases {
			if name, ok := realSlugs[store.Slugify(a)]; ok && store.Slugify(a) != e.Slug {
				collisions = append(collisions, name)
			}
		}
		if len(collisions) > 0 {
			conflated = append(conflated, ConflationReport{Slug: e.Slug, Name: e.Name, CollidesWith: collisions})
		}
	}
	cross := 0
	var sample []string
	for key, os := range byName {
		if len(os) < 2 {
			continue
		}
		for i := range os {
			for j := i + 1; j < len(os); j++ {
				if sameKind(os[i].typ, os[j].typ) {
					continue
				}
				cross++
				if len(sample) < 40 {
					sample = append(sample, fmt.Sprintf("%s: %s:%s (%q) / %s:%s (%q)",
						key, os[i].typ, os[i].slug, os[i].spelling, os[j].typ, os[j].slug, os[j].spelling))
				}
			}
		}
	}
	sort.Strings(sample)
	sort.Slice(conflated, func(i, j int) bool { return conflated[i].Slug < conflated[j].Slug })
	return cross, sample, conflated
}

// sameKind reports whether two entity types name the same kind of thing.
// Unlike TypesCompatible it gives a concept no wildcard: for counting,
// an untyped stub that shares a machine's name is a collision.
func sameKind(a, b string) bool {
	a, b = strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	return a == b || a == "" || b == ""
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

// aliasNamesOther reports whether an alias (tokens at) plainly names the
// entity whose name has tokens nt. Across a type boundary the name may add
// at most one token to the alias ("Mac mini" for "mini"). For the same
// type the alias must embed the other's whole multi-token name
// ("box2-gpt-oss-120b" embeds "gpt-oss-120b") or be a two-plus-token tail
// of it ("oss-120b" in "gpt-oss-120b").
func aliasNamesOther(at, nt map[string]bool, compatible bool) bool {
	subset := func(a, b map[string]bool) bool {
		for t := range a {
			if !b[t] {
				return false
			}
		}
		return true
	}
	if !compatible {
		return len(nt) <= len(at)+1 && subset(at, nt)
	}
	if len(nt) >= 2 && len(nt) < len(at)+2 && subset(nt, at) {
		return true
	}
	return len(at) >= 2 && len(nt) == len(at)+1 && subset(at, nt)
}

// subsetOf reports whether every token of a is in b.
func subsetOf(a, b map[string]bool) bool {
	for t := range a {
		if !b[t] {
			return false
		}
	}
	return true
}

// compact removes separators so "halo-1", "halo_1", and "halo1" compare
// equal.
func compact(s string) string {
	return nonAlnumRE.ReplaceAllString(strings.ToLower(s), "")
}

var nonAlnumRE = regexp.MustCompile(`[^a-z0-9]+`)

// mergeDuplicateStubs folds a concept stub into the typed entity whose
// name it is another spelling of. "Android App Links" written once as a
// service and once as a concept is one thing said twice, and the store
// answered about it twice, splitting its facts between the two.
//
// Only a group with exactly one typed entity is touched. Two typed
// entities sharing a name are the case this whole item exists to protect
// — a machine and a project are never merged, whatever they are called —
// so those are counted and left for a person.
func mergeDuplicateStubs(st *store.Store, entities []store.Entity, dryRun bool) (int, []string, error) {
	groups := map[string][]store.Entity{}
	for _, e := range entities {
		if isEphemeralName(e.Name) || IsValueName(e.Name) {
			continue
		}
		groups[mergeKey(e.Name)] = append(groups[mergeKey(e.Name)], e)
	}
	merged := 0
	var sample []string
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		var typed []store.Entity
		var stubs []store.Entity
		for _, e := range g {
			if t := strings.ToLower(strings.TrimSpace(e.Type)); t == "" || t == "concept" {
				stubs = append(stubs, e)
				continue
			}
			typed = append(typed, e)
		}
		if len(typed) != 1 || len(stubs) == 0 || !absorbs(typed[0].Type) {
			continue
		}
		for _, stub := range stubs {
			merged++
			if len(sample) < 30 {
				sample = append(sample, stub.Slug+" ("+stub.Type+") -> "+typed[0].Slug+" ("+typed[0].Type+")")
			}
			if dryRun {
				continue
			}
			if err := mergeStub(st, stub, typed[0]); err != nil {
				return merged, sample, err
			}
		}
	}
	sort.Strings(sample)
	return merged, sample, nil
}

// mergeKey is the name two entities must share to be one thing written
// twice. It folds case and punctuation but not plurals: reports.ts and
// report.ts are two files, books and book are a table and a project, and
// API integrations is not the person responsible for api-integration.
// The counting key (foldName) folds more, because two names a reader
// would confuse are worth reporting even when they are not worth
// merging.
func mergeKey(s string) string {
	return nonAlnumRE.ReplaceAllString(strings.ToLower(s), "")
}

// absorbs reports whether a type can take a stub's facts. A person and a
// decision cannot: a person is not the model named after them, and a
// decision is an event rather than a thing that holds a directory's
// facts. Both were survivors of merges that should not have happened.
func absorbs(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "person", "decision", "episode", "event":
		return false
	}
	return true
}

// mergeStub moves a stub's facts, aliases, and repository references onto
// target and removes the stub. A fact that joined the two becomes a self
// loop and is invalidated rather than moved, since a thing does not
// relate to itself.
func mergeStub(st *store.Store, stub, target store.Entity) error {
	facts, err := st.FactsAbout(stub.Slug, true)
	if err != nil {
		return err
	}
	for _, f := range facts {
		updated := f
		if updated.Src == stub.Slug {
			updated.Src = target.Slug
		}
		if updated.Dst == stub.Slug {
			updated.Dst = target.Slug
		}
		if updated.Src == updated.Dst {
			if f.InvalidAt == nil {
				at := time.Now()
				if err := st.InvalidateFact(f.Src, f.Relation, f.KeyDst(), f.ValidFrom, at); err != nil {
					return err
				}
			}
			continue
		}
		if err := st.RelocateFact(f, updated); err != nil {
			return fmt.Errorf("merge %s into %s: %w", stub.Slug, target.Slug, err)
		}
	}
	have := map[string]bool{store.Normalize(target.Name): true}
	for _, a := range target.Aliases {
		have[store.Normalize(a)] = true
	}
	for _, a := range append([]string{stub.Name}, stub.Aliases...) {
		if n := store.Normalize(a); n != "" && !have[n] && !neverAlias(a) {
			have[n] = true
			target.Aliases = append(target.Aliases, a)
		}
	}
	refs := map[string]bool{}
	for _, r := range target.RepoRefs {
		refs[r] = true
	}
	for _, r := range stub.RepoRefs {
		if !refs[r] {
			refs[r] = true
			target.RepoRefs = append(target.RepoRefs, r)
		}
	}
	sort.Strings(target.RepoRefs)
	if err := st.PutEntity(target); err != nil {
		return err
	}
	if err := st.DeleteEntity(stub.Slug); err != nil {
		return err
	}
	for _, a := range append([]string{stub.Name, stub.Slug}, stub.Aliases...) {
		if err := st.ClaimAlias(a, target.Slug); err != nil {
			return err
		}
	}
	return nil
}

// dropStubClaims removes from every concept, and every untyped entity,
// each alias that a typed entity answers to by name or by alias. It
// never moves anything: the typed entity already answers to the name, so
// nothing stops resolving.
func dropStubClaims(st *store.Store, dryRun bool) (int, error) {
	entities, err := st.Entities()
	if err != nil {
		return 0, err
	}
	// Keyed the way the alias index is keyed, not the way collisions are
	// counted. foldName folds plurals and punctuation, so matching on it
	// dropped names a typed entity merely *folds* to: "halo1" went
	// because "halo-1" folds to it, and then nothing resolved "halo1" at
	// all. The index answers on store.Normalize, so that is what decides
	// whether the name still has an owner.
	typed := map[string]string{}
	for _, e := range entities {
		if t := strings.ToLower(strings.TrimSpace(e.Type)); t == "" || t == "concept" {
			continue
		}
		for _, spelling := range append([]string{e.Name}, e.Aliases...) {
			if k := store.Normalize(spelling); k != "" {
				typed[k] = e.Slug
			}
		}
	}
	dropped := 0
	for _, e := range entities {
		if t := strings.ToLower(strings.TrimSpace(e.Type)); t != "" && t != "concept" {
			continue
		}
		kept := make([]string, 0, len(e.Aliases))
		changed := false
		for _, a := range e.Aliases {
			k := store.Normalize(a)
			if owner, ok := typed[k]; ok && owner != e.Slug && k != store.Normalize(e.Name) {
				dropped++
				changed = true
				continue
			}
			kept = append(kept, a)
		}
		if !changed || dryRun {
			continue
		}
		e.Aliases = kept
		if err := st.PutEntity(e); err != nil {
			return dropped, err
		}
	}
	return dropped, nil
}

// referencedSlugs returns every entity slug a fact mentions at either
// end, invalidated facts included.
func referencedSlugs(st *store.Store) (map[string]bool, error) {
	facts, err := st.AllFacts()
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(facts))
	for _, f := range facts {
		out[f.Src] = true
		if f.Dst != "" {
			out[f.Dst] = true
		}
	}
	return out, nil
}

// pruneAge is how long an entity may exist with nothing said about it
// before it is removed. A resolver writes an entity and its facts in one
// pass, so an entity a day old with no fact is one an extraction named
// and then said nothing about.
const pruneAge = 24 * time.Hour

// locatedNameRE splits "Mac mini at 100.96.45.73" into the thing and
// where it is.
var locatedNameRE = regexp.MustCompile(`(?i)^(.*?)\s+(?:at|on|@)\s+(\S+)$`)

// addressRE matches the where: an IP, a host on a local or tailnet
// domain, or a bare port.
var addressRE = regexp.MustCompile(`(?i)^(?:\d{1,3}\.){3}\d{1,3}(?::\d+)?$|^[a-z0-9-]+\.(?:local|lan|ts\.net|internal)$|^:\d{2,5}$`)

// mergeLocatedDuplicates folds an entity named "<thing> at <address>"
// into the thing, when the thing is an entity of the same type with more
// facts. Naming something by where it is does not make it a second
// thing, and the pair is only merged when the address is genuinely an
// address rather than any trailing word.
func mergeLocatedDuplicates(st *store.Store, dryRun bool) (int, []string, error) {
	entities, err := st.Entities()
	if err != nil {
		return 0, nil, err
	}
	facts, err := st.AllFacts()
	if err != nil {
		return 0, nil, err
	}
	count := map[string]int{}
	for _, f := range facts {
		if f.InvalidAt != nil {
			continue
		}
		count[f.Src]++
		if f.Dst != "" {
			count[f.Dst]++
		}
	}
	byName := map[string]store.Entity{}
	for _, e := range entities {
		byName[strings.ToLower(strings.TrimSpace(e.Name))] = e
	}
	merged := 0
	var sample []string
	for _, e := range entities {
		m := locatedNameRE.FindStringSubmatch(e.Name)
		if m == nil || !addressRE.MatchString(m[2]) {
			continue
		}
		target, ok := byName[strings.ToLower(strings.TrimSpace(m[1]))]
		if !ok || target.Slug == e.Slug || !strings.EqualFold(target.Type, e.Type) {
			continue
		}
		if count[target.Slug] <= count[e.Slug] {
			continue
		}
		merged++
		sample = append(sample, e.Slug+" ("+e.Type+") -> "+target.Slug+", named by where it is")
		if dryRun {
			continue
		}
		if err := mergeStub(st, e, target); err != nil {
			return merged, sample, err
		}
	}
	return merged, sample, nil
}
