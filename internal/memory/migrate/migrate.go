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

	ValueEntities       int      `json:"value_entities"`
	ValueFactsConverted int      `json:"value_facts_converted"`
	ValueFactsDropped   int      `json:"value_facts_dropped"` // both endpoints values: invalidated
	ValueEntitiesSample []string `json:"value_entities_sample,omitempty"`

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
		if resolve.IsValueName(e.Name) || resolve.IsEphemeralName(e.Name) || pureNumberOrHex(e.Slug) {
			values[e.Slug] = e
		}
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
			continue // already an attribute
		}
		srcVal, dstVal := values[f.Src], values[f.Dst]
		_, srcIsValue := values[f.Src]
		_, dstIsValue := values[f.Dst]
		statusEdge := f.Relation == resolve.RelStatus
		if !srcIsValue && !dstIsValue && !statusEdge {
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
		default: // status edge to a real entity
			if _, ok := bySlug[f.Src]; !ok {
				// The source was retired by an earlier pass (a value entity
				// whose edges are already invalidated); nothing to convert.
				continue
			}
			updated.Dst = ""
			updated.Value = bySlug[f.Dst].Name
			if updated.Value == "" {
				updated.Value = f.Dst
			}
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
		if resolve.IsValueName(e.Name) || resolve.IsEphemeralName(e.Name) {
			rep.ValueEntitiesAfter++
		}
	}
	return nil
}

var pureNumberOrHexRE = regexp.MustCompile(`^\d+$|^[0-9a-f]{7,40}$`)

// pureNumberOrHex catches entities whose name slugified to a bare number
// or a commit hash ("#140" → "140") even when the name itself has a prefix
// the value detector does not know.
func pureNumberOrHex(slug string) bool { return pureNumberOrHexRE.MatchString(slug) }
