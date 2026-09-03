// Package migrate applies the resolver's current rules to a store written
// before they existed: the closed relation vocabulary, value endpoints as
// attributes rather than entities, and the alias hygiene. It runs inside
// the daemon that owns the store, takes a backup first, and can be run
// again safely — a second pass finds nothing to do.
package migrate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/resolve"
	"github.com/jeffdhooton/scry/internal/memory/store"
)

// Options configures Run.
type Options struct {
	DryRun bool
	// Backup is called before anything is written; it returns the backup
	// path. Required unless DryRun.
	Backup func() (string, error)
	Logf   func(format string, args ...any)
}

// Report is what one Run did or would do.
type Report struct {
	DryRun     bool   `json:"dry_run"`
	BackupPath string `json:"backup_path,omitempty"`

	FactsScanned       int            `json:"facts_scanned"`
	RelationsRewritten int            `json:"relations_rewritten"`
	RelationsFlipped   int            `json:"relations_flipped"`
	RelationMapping    map[string]int `json:"relation_mapping"` // canonical → facts
	RelationFallback   int            `json:"relation_fallback"`

	// AttributesRestored counts attribute facts whose value turned out to
	// be an identity under the current rules and were turned back into
	// edges to a (re)created entity.
	AttributesRestored  int `json:"attributes_restored"`
	ValueEntities       int `json:"value_entities"`
	ValueFactsConverted int `json:"value_facts_converted"`
	// DeployedOnRestored counts facts that exclusivity invalidated while
	// deployed_on was still an exclusive relation, brought back because a
	// thing is deployed in more than one place at once.
	DeployedOnRestored int `json:"deployed_on_restored"`
	// DanglingEndpoints counts fact endpoints whose entity no longer
	// exists, left behind when an earlier pass retired a value entity.
	DanglingEndpoints int `json:"dangling_endpoints"`
	// InversionsRepaired counts facts that an older episode retired the
	// moment they were written, with a fact older than themselves left
	// current in their place.
	InversionsRepaired int `json:"inversions_repaired"`
	// StatusEdgesRepointed counts status edges between two real entities
	// turned back into related_to edges.
	StatusEdgesRepointed int      `json:"status_edges_repointed"`
	ValueFactsDropped    int      `json:"value_facts_dropped"` // both endpoints values: invalidated
	ValueEntitiesSample  []string `json:"value_entities_sample,omitempty"`

	Hygiene       resolve.HygieneReport `json:"hygiene"`
	HygienePasses int                   `json:"hygiene_passes"`
	// ValueFactsSample lists the first conversions, for reading a dry run.
	ValueFactsSample []string `json:"value_facts_sample,omitempty"`

	NonCanonicalAfter  int      `json:"non_canonical_after"`
	ValueEntitiesAfter int      `json:"value_entities_after"`
	Errors             []string `json:"errors,omitempty"`
	Duration           string   `json:"duration"`
}

// Run performs the migration.
func Run(st *store.Store, o Options) (Report, error) {
	start := time.Now()
	rep := Report{DryRun: o.DryRun, RelationMapping: map[string]int{}}
	logf := o.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if !o.DryRun {
		if o.Backup == nil {
			return rep, fmt.Errorf("migrate: a backup function is required to apply")
		}
		path, err := o.Backup()
		if err != nil {
			return rep, fmt.Errorf("migrate: backup: %w", err)
		}
		rep.BackupPath = path
		logf("migrate: backup at %s", path)
	}

	if err := migrateRelations(st, o.DryRun, &rep); err != nil {
		return rep, err
	}
	logf("migrate: relations: %d scanned, %d rewritten, %d flipped, %d fallback",
		rep.FactsScanned, rep.RelationsRewritten, rep.RelationsFlipped, rep.RelationFallback)

	if err := repairInversions(st, o.DryRun, &rep); err != nil {
		return rep, err
	}
	logf("migrate: order: %d facts restored over an older fact left current in their place", rep.InversionsRepaired)

	if err := restoreDeployedOn(st, o.DryRun, &rep); err != nil {
		return rep, err
	}
	logf("migrate: deployed_on: %d facts restored", rep.DeployedOnRestored)

	if err := migrateValues(st, o.DryRun, &rep); err != nil {
		return rep, err
	}
	logf("migrate: values: %d entities retired, %d facts converted, %d dropped",
		rep.ValueEntities, rep.ValueFactsConverted, rep.ValueFactsDropped)

	// Hygiene converges rather than completes in one pass: a fact moved
	// onto an entity can make one of that entity's aliases newly splittable.
	// Three passes have always been enough; a dry run gets one.
	for pass := 1; pass <= 3; pass++ {
		hyg, err := resolve.Hygiene(st, o.DryRun)
		if err != nil {
			return rep, fmt.Errorf("migrate: hygiene pass %d: %w", pass, err)
		}
		logf("migrate: hygiene pass %d: %d aliases dropped, %d split, %d facts reattached, %d self-loops, %d cross-type collisions left",
			pass, hyg.AliasesDropped, hyg.AliasesSplit, hyg.FactsReattached, hyg.SelfLoopsInvalidated, hyg.CrossTypeCollisions)
		if pass == 1 {
			rep.Hygiene = hyg
		} else {
			rep.Hygiene.AliasesDropped += hyg.AliasesDropped
			rep.Hygiene.AliasesSplit += hyg.AliasesSplit
			rep.Hygiene.FactsReattached += hyg.FactsReattached
			rep.Hygiene.SelfLoopsInvalidated += hyg.SelfLoopsInvalidated
			rep.Hygiene.EntitiesChanged += hyg.EntitiesChanged
			rep.Hygiene.DroppedAliasList = append(rep.Hygiene.DroppedAliasList, hyg.DroppedAliasList...)
			rep.Hygiene.Reattachments = append(rep.Hygiene.Reattachments, hyg.Reattachments...)
			rep.Hygiene.CrossTypeCollisions = hyg.CrossTypeCollisions
			rep.Hygiene.Conflated = hyg.Conflated
		}
		rep.HygienePasses = pass
		if o.DryRun || hyg.AliasesDropped+hyg.AliasesSplit+hyg.FactsReattached+hyg.SelfLoopsInvalidated == 0 {
			break
		}
	}

	if !o.DryRun {
		if err := audit(st, &rep); err != nil {
			return rep, err
		}
	}
	rep.Duration = time.Since(start).Round(time.Millisecond).String()
	return rep, nil
}

// migrateRelations maps every fact's relation onto the vocabulary, swapping
// endpoints for inverse forms, and records the raw relation.
func migrateRelations(st *store.Store, dryRun bool, rep *Report) error {
	facts, err := st.AllFacts()
	if err != nil {
		return err
	}
	for _, f := range facts {
		rep.FactsScanned++
		canon, flip := resolve.Map(f.Relation)
		rep.RelationMapping[canon]++
		if canon == resolve.Fallback && !resolve.IsCanonical(f.Relation) {
			rep.RelationFallback++
		}
		if canon == f.Relation && !flip {
			continue
		}
		rep.RelationsRewritten++
		updated := f
		updated.Relation = canon
		if updated.RawRelation == "" && canon != f.Relation {
			updated.RawRelation = f.Relation
		}
		if flip && f.Dst != "" {
			updated.Src, updated.Dst = f.Dst, f.Src
			rep.RelationsFlipped++
		}
		if dryRun {
			continue
		}
		if err := st.RelocateFact(f, updated); err != nil {
			return fmt.Errorf("relocate %s -[%s]-> %s: %w", f.Src, f.Relation, f.KeyDst(), err)
		}
	}
	return nil
}

// migrateValues turns every fact touching a value-named entity into an
// attribute fact and retires the entity. A `status` edge to a real entity
// becomes an attribute too, since status is always a value.
func migrateValues(st *store.Store, dryRun bool, rep *Report) error {
	entities, err := st.Entities()
	if err != nil {
		return err
	}
	bySlug := make(map[string]store.Entity, len(entities))
	values := map[string]store.Entity{}
	for _, e := range entities {
		bySlug[e.Slug] = e
		if retireable(e.Name) {
			values[e.Slug] = e
		}
	}
	// A fact may point at a slug with no entity behind it: an earlier pass
	// retired the value entity and missed a fact that had been moved onto
	// it. The endpoint is a value in everything but the entity table, so it
	// is treated as one here and the fact becomes an attribute like any
	// other. The slug is its own best name; there is nothing else left of
	// it to read.
	dangling, err := danglingEndpoints(st, bySlug)
	if err != nil {
		return err
	}
	rep.DanglingEndpoints = len(dangling)
	for slug := range dangling {
		values[slug] = store.Entity{Slug: slug, Name: strings.ReplaceAll(slug, "-", " ")}
	}
	rep.ValueEntities = len(values)
	names := make([]string, 0, len(values))
	for _, e := range values {
		names = append(names, e.Name)
	}
	sort.Strings(names)
	if len(names) > 40 {
		names = names[:40]
	}
	rep.ValueEntitiesSample = names

	now := time.Now()
	facts, err := st.AllFacts()
	if err != nil {
		return err
	}
	for _, f := range facts {
		if f.Dst == "" {
			// An attribute whose value is not a value under the current
			// rules (a file path demoted by an earlier, wider pattern) goes
			// back to being an edge to an entity.
			if f.Relation != resolve.RelStatus && f.Value != "" && !retireable(f.Value) {
				slug := store.Slugify(f.Value)
				if slug == "" {
					continue
				}
				rep.AttributesRestored++
				if dryRun {
					continue
				}
				if _, err := st.GetEntity(slug); err != nil {
					if err := st.PutEntity(store.Entity{Slug: slug, Name: f.Value, Type: "concept", CreatedAt: f.ValidFrom, LastSeen: f.ValidFrom}); err != nil {
						return err
					}
				}
				updated := f
				updated.Dst, updated.Value = slug, ""
				if err := st.RelocateFact(f, updated); err != nil {
					return fmt.Errorf("restore %s -[%s]-> %q: %w", f.Src, f.Relation, f.Value, err)
				}
			}
			// An attribute whose subject is itself a value describes
			// nothing: "5 residual test failures" is not a thing that can
			// have a status. The sentence stays readable as an
			// invalidated fact; it is not knowledge about any identity.
			// Skipping these is what left several hundred facts pointing
			// at entities the pass had just deleted.
			if _, ok := values[f.Src]; ok && f.InvalidAt == nil {
				rep.ValueFactsDropped++
				if !dryRun {
					at := time.Now()
					if err := st.InvalidateFact(f.Src, f.Relation, f.KeyDst(), f.ValidFrom, at); err != nil {
						return err
					}
				}
			}
			continue // already an attribute
		}
		srcVal, dstVal := values[f.Src], values[f.Dst]
		_, srcIsValue := values[f.Src]
		_, dstIsValue := values[f.Dst]
		// A status edge to a real identity is not a status: the model used
		// the word loosely. Re-point the relation and keep the edge, rather
		// than making the other entity this one's status and letting
		// exclusivity invalidate it.
		statusEdge := f.Relation == resolve.RelStatus && !srcIsValue && !dstIsValue
		if statusEdge {
			if _, ok := bySlug[f.Dst]; ok {
				updated := f
				updated.Relation = resolve.RelRelatedTo
				if updated.RawRelation == "" {
					updated.RawRelation = resolve.RelStatus
				}
				rep.StatusEdgesRepointed++
				if dryRun {
					continue
				}
				if err := st.RelocateFact(f, updated); err != nil {
					return fmt.Errorf("repoint status edge %s -> %s: %w", f.Src, f.Dst, err)
				}
				continue
			}
		}
		if !srcIsValue && !dstIsValue {
			continue
		}
		updated := f
		switch {
		case srcIsValue && dstIsValue:
			rep.ValueFactsDropped++
			if dryRun || f.InvalidAt != nil {
				continue
			}
			if err := st.InvalidateFact(f.Src, f.Relation, f.KeyDst(), f.ValidFrom, now); err != nil {
				return err
			}
			continue
		case srcIsValue:
			updated.Src = f.Dst
			updated.Dst = ""
			updated.Value = srcVal.Name
		case dstIsValue:
			updated.Dst = ""
			updated.Value = dstVal.Name
		default:
			// Unreachable: the status-edge case is handled above.
			continue
		}
		rep.ValueFactsConverted++
		if len(rep.ValueFactsSample) < 30 {
			rep.ValueFactsSample = append(rep.ValueFactsSample, f.Src+" -["+f.Relation+"]-> "+f.Dst+" ⇒ value "+updated.Value)
		}
		if dryRun {
			continue
		}
		if err := st.RelocateFact(f, updated); err != nil {
			return fmt.Errorf("convert %s -[%s]-> %s: %w", f.Src, f.Relation, f.Dst, err)
		}
	}
	if dryRun {
		return nil
	}
	for slug := range values {
		if err := st.DeleteEntity(slug); err != nil {
			return err
		}
	}
	return nil
}

// audit recounts what the migration promised to remove.
func audit(st *store.Store, rep *Report) error {
	facts, err := st.AllFacts()
	if err != nil {
		return err
	}
	for _, f := range facts {
		if f.InvalidAt == nil && !resolve.IsCanonical(f.Relation) {
			rep.NonCanonicalAfter++
		}
	}
	entities, err := st.Entities()
	if err != nil {
		return err
	}
	for _, e := range entities {
		if retireable(e.Name) {
			rep.ValueEntitiesAfter++
		}
	}
	return nil
}

var pureNumberOrHexRE = regexp.MustCompile(`^\d+$|^[0-9a-f]{7,40}$`)

// retireable is the one predicate both the retire step and the restore
// step use, so a value can never be restored as an entity by one pass and
// retired again by the next.
func retireable(name string) bool {
	return resolve.NotAnIdentity(name) || pureNumberOrHex(store.Slugify(name))
}

// pureNumberOrHex catches entities whose name slugified to a bare number
// or a commit hash ("#140" → "140") even when the name itself has a prefix
// the value detector does not know.
func pureNumberOrHex(slug string) bool { return pureNumberOrHexRE.MatchString(slug) }

// inversionWindow is how soon after a fact begins its retirement counts as
// an artifact of arrival order rather than a real change. Two facts about
// the same thing that genuinely follow one another are minutes apart at
// the very least; two seconds means the retiring episode predated the
// fact it retired.
const inversionWindow = 2 * time.Second

// repairInversions undoes the retirements that arrival order caused.
// Transcripts are swept in whatever order they are found, so a July
// session was routinely resolved after an August fact was stored, and
// exclusivity retired the August fact and left July current. The store
// then answered with the older state.
//
// For an exclusive relation the two swap: the newer fact becomes current
// and the older one ends where the newer begins. For a relation that is
// no longer exclusive the newer fact is simply restored and the older one
// left alone, since both can hold at once.
func repairInversions(st *store.Store, dry bool, rep *Report) error {
	facts, err := st.AllFacts()
	if err != nil {
		return fmt.Errorf("migrate: scan for inversions: %w", err)
	}
	groups := map[string][]store.Fact{}
	for _, f := range facts {
		k := f.Src + "\x00" + f.Relation
		groups[k] = append(groups[k], f)
	}
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		sort.Slice(g, func(i, j int) bool { return g[i].ValidFrom.Before(g[j].ValidFrom) })
		for i := range g {
			f := &g[i]
			if f.InvalidAt == nil || f.InvalidAt.Sub(f.ValidFrom) >= inversionWindow {
				continue
			}
			var winner *store.Fact
			for j := range g {
				w := &g[j]
				if w == f || w.InvalidAt != nil {
					continue
				}
				if f.ValidFrom.Sub(w.ValidFrom) < inversionWindow {
					continue
				}
				if winner == nil || w.ValidFrom.After(winner.ValidFrom) {
					winner = w
				}
			}
			if winner == nil {
				continue
			}
			rep.InversionsRepaired++
			if dry {
				continue
			}
			f.InvalidAt = nil
			if err := st.PutFact(*f); err != nil {
				return fmt.Errorf("migrate: restore %s %s: %w", f.Src, f.Relation, err)
			}
			if !resolve.DefaultExclusive[f.Relation] {
				continue
			}
			at := f.ValidFrom
			winner.InvalidAt = &at
			if err := st.PutFact(*winner); err != nil {
				return fmt.Errorf("migrate: retire %s %s: %w", winner.Src, winner.Relation, err)
			}
		}
	}
	return nil
}

// restoreDeployedOn brings back deployed_on facts that were retired
// because another deployment was recorded. The relation is not exclusive
// any more — a thing runs in more than one place at once — so a fact
// retired in favour of a sibling deployment was retired by a rule that no
// longer exists.
//
// The test is that some other deployment of the same thing had already
// begun when this one was retired. That is the shape Rule 6 left behind,
// and also the shape an explicit "it moved" hint leaves; the two cannot
// be told apart after the fact. Restoring both is the deliberate choice:
// a deployment that really did end will be restated by the next session
// that looks, while a fact silently retired stays gone. 336 of the 369
// retired deployments carried this shape, among them where each web
// application runs.
func restoreDeployedOn(st *store.Store, dry bool, rep *Report) error {
	const relation = "deployed_on"
	starts := map[string][]time.Time{} // src → ValidFrom of every deployment
	var retired []store.Fact
	facts, err := st.AllFacts()
	if err != nil {
		return fmt.Errorf("migrate: scan deployed_on: %w", err)
	}
	for _, f := range facts {
		if f.Relation != relation {
			continue
		}
		starts[f.Src] = append(starts[f.Src], f.ValidFrom)
		if f.InvalidAt != nil {
			retired = append(retired, f)
		}
	}
	// An endpoint that is not an identity is about to be retired by the
	// value pass, so restoring the fact only sets up the same restore
	// next run: 39 facts were being revived and re-retired on every
	// migration.
	ents, err := st.Entities()
	if err != nil {
		return fmt.Errorf("migrate: entities: %w", err)
	}
	real := make(map[string]bool, len(ents))
	for _, e := range ents {
		if !resolve.NotAnIdentity(e.Name) {
			real[e.Slug] = true
		}
	}
	for _, f := range retired {
		if !real[f.Src] || (f.Dst != "" && !real[f.Dst]) {
			continue
		}
		// A thing is not deployed on itself. Hygiene retires self loops,
		// and reviving them here is what made 39 facts come back on every
		// run and go straight out again.
		if f.Src == f.Dst {
			continue
		}
		superseded := false
		for _, start := range starts[f.Src] {
			if start.Equal(f.ValidFrom) {
				continue // itself, or a fact recorded at the same instant
			}
			if !start.After(*f.InvalidAt) {
				superseded = true
				break
			}
		}
		if !superseded {
			continue
		}
		rep.DeployedOnRestored++
		if dry {
			continue
		}
		revived := f
		revived.InvalidAt = nil
		if err := st.PutFact(revived); err != nil {
			return fmt.Errorf("migrate: restore %s deployed_on: %w", f.Src, err)
		}
	}
	return nil
}

// danglingEndpoints returns every slug a current fact points at that has
// no entity record. Invalidated facts are left out: they are history, and
// rewriting history is not this pass's job.
func danglingEndpoints(st *store.Store, bySlug map[string]store.Entity) (map[string]bool, error) {
	facts, err := st.AllFacts()
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, f := range facts {
		if f.InvalidAt != nil {
			continue
		}
		for _, slug := range []string{f.Src, f.Dst} {
			if slug == "" {
				continue
			}
			if _, ok := bySlug[slug]; !ok {
				out[slug] = true
			}
		}
	}
	return out, nil
}
