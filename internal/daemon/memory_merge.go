package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/jeffdhooton/scry/internal/memory/resolve"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// MemoryMergeEntitiesParams is a reviewed list of disjoint identity groups.
// Dry run is the default at the CLI and returns the metadata and fingerprints
// that must be copied into the manifest before apply.
type MemoryMergeEntitiesParams struct {
	Groups []memstore.EntityMergeRequest `json:"groups"`
	DryRun *bool                         `json:"dry_run,omitempty"`
}

type MemoryMergeGroupResult struct {
	memstore.EntityMergePreview
	CollisionsBefore        int  `json:"collisions_before"`
	CollisionsAfter         int  `json:"collisions_after"`
	CollisionDelta          int  `json:"collision_delta"`
	ObservedCollisionsAfter *int `json:"observed_collisions_after,omitempty"`
	CollisionVerified       bool `json:"collision_verified"`
}

type MemoryMergeEntitiesResult struct {
	DryRun     bool                     `json:"dry_run"`
	BackupPath string                   `json:"backup_path,omitempty"`
	Applied    int                      `json:"applied"`
	Refused    int                      `json:"refused"`
	Groups     []MemoryMergeGroupResult `json:"groups"`
}

// handleMemoryMergeEntities previews or atomically applies each reviewed
// group. Apply preflights the complete manifest before taking a backup; each
// group is then one Badger transaction with the fingerprints checked again
// inside it.
func (d *Daemon) handleMemoryMergeEntities(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryMergeEntitiesParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if len(p.Groups) == 0 {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "no merge groups given"}
	}
	if err := ValidateMemoryMergeGroups(p.Groups); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	dryRun := p.DryRun == nil || *p.DryRun
	res := &MemoryMergeEntitiesResult{DryRun: dryRun}

	// Preflight every group against one read snapshot before any apply. This
	// catches incomplete reviews without leaving earlier groups committed.
	entities, err := st.Entities()
	if err != nil {
		return nil, err
	}
	facts, err := st.AllFacts()
	if err != nil {
		return nil, err
	}
	if err := ValidateMemoryMergeIsolation(p.Groups, facts); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	simEntities, simFacts := entities, facts
	for _, group := range p.Groups {
		preview, err := st.PreviewEntityMerge(group)
		if err != nil {
			return nil, err
		}
		gr := NewMemoryMergeGroupResult(preview, simEntities, simFacts)
		res.Groups = append(res.Groups, gr)
		if !preview.Ready {
			res.Refused++
			continue
		}
		if !dryRun && (group.Metadata == nil || !reflect.DeepEqual(group.Expected, preview.Expected)) {
			res.Refused++
			res.Groups[len(res.Groups)-1].Problems = append(res.Groups[len(res.Groups)-1].Problems,
				"apply requires reviewed metadata and exact current fingerprints")
		}
		if preview.Ready {
			simEntities, simFacts = resolve.SimulateEntityMerge(simEntities, simFacts, preview)
		}
	}
	if dryRun || res.Refused > 0 {
		return res, nil
	}

	b, err := d.handleMemoryBackup(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("merge-entities: backup first: %w", err)
	}
	res.BackupPath = b.(*MemoryBackupResult).Path
	for i, group := range p.Groups {
		preview, err := st.MergeEntities(group)
		if err != nil {
			return nil, fmt.Errorf("merge-entities group %s after %d committed group(s): %w", mergeGroupName(group), res.Applied, err)
		}
		res.Groups[i].EntityMergePreview = preview
		observedEntities, err := st.Entities()
		if err != nil {
			return nil, fmt.Errorf("merge-entities group %s committed but collision recount failed (backup %s): %w", mergeGroupName(group), res.BackupPath, err)
		}
		observedFacts, err := st.AllFacts()
		if err != nil {
			return nil, fmt.Errorf("merge-entities group %s committed but collision recount failed (backup %s): %w", mergeGroupName(group), res.BackupPath, err)
		}
		observed := resolve.CrossTypeCollisionCount(observedEntities, observedFacts)
		res.Groups[i].ObservedCollisionsAfter = &observed
		if observed != res.Groups[i].CollisionsAfter {
			return nil, fmt.Errorf("merge-entities group %s collision verification failed after commit: predicted %d, observed %d (restore backup %s before retrying)", mergeGroupName(group), res.Groups[i].CollisionsAfter, observed, res.BackupPath)
		}
		res.Groups[i].CollisionVerified = true
		res.Applied++
	}
	return res, nil
}

// ValidateMemoryMergeIsolation refuses manifests whose groups touch the same
// fact. The first group would otherwise change the second group's reviewed
// fact fingerprint after whole-manifest preflight and cause a partial apply.
func ValidateMemoryMergeIsolation(groups []memstore.EntityMergeRequest, facts []memstore.Fact) error {
	member := map[string]int{}
	for i, group := range groups {
		member[group.Survivor] = i
		for _, slug := range group.Retire {
			member[slug] = i
		}
	}
	for _, fact := range facts {
		srcGroup, srcOK := member[fact.Src]
		dstGroup, dstOK := member[fact.Dst]
		if srcOK && dstOK && srcGroup != dstGroup {
			return fmt.Errorf("merge groups %s and %s share fact %s -- %s -- %s; apply them in separate reviewed manifests", mergeGroupName(groups[srcGroup]), mergeGroupName(groups[dstGroup]), fact.Src, fact.Relation, fact.Dst)
		}
	}
	return nil
}

// ValidateMemoryMergeGroups rejects overlapping groups before any backup or
// write. The offline CLI and daemon share this manifest-level guard.
func ValidateMemoryMergeGroups(groups []memstore.EntityMergeRequest) error {
	seen := map[string]string{}
	for _, group := range groups {
		label := group.ID
		if label == "" {
			label = group.Survivor
		}
		for _, slug := range append([]string{group.Survivor}, group.Retire...) {
			if prior := seen[slug]; prior != "" {
				return fmt.Errorf("entity %s appears in merge groups %s and %s", slug, prior, label)
			}
			seen[slug] = label
		}
	}
	return nil
}

// NewMemoryMergeGroupResult adds the collision prediction to a store preview.
// It is exported so the CLI's offline replica path and daemon use one metric.
func NewMemoryMergeGroupResult(preview memstore.EntityMergePreview, entities []memstore.Entity, facts []memstore.Fact) MemoryMergeGroupResult {
	before, after := resolve.PredictEntityMergeCollisionCounts(entities, facts, preview)
	return MemoryMergeGroupResult{EntityMergePreview: preview, CollisionsBefore: before, CollisionsAfter: after, CollisionDelta: after - before}
}

func mergeGroupName(group memstore.EntityMergeRequest) string {
	if strings.TrimSpace(group.ID) != "" {
		return group.ID
	}
	retired := append([]string(nil), group.Retire...)
	sort.Strings(retired)
	return group.Survivor + "<-" + strings.Join(retired, ",")
}
