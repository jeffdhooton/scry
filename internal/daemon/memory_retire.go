package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// MemoryRetireEntitiesParams is a reviewed list of independent non-identity
// retirements. Omitted dry_run is intentionally safe for raw RPC callers.
type MemoryRetireEntitiesParams struct {
	Groups []memstore.EntityRetirementRequest `json:"groups"`
	DryRun *bool                              `json:"dry_run,omitempty"`
}

type MemoryRetireEntitiesResult struct {
	DryRun     bool                               `json:"dry_run"`
	BackupPath string                             `json:"backup_path,omitempty"`
	Applied    int                                `json:"applied"`
	Refused    int                                `json:"refused"`
	Groups     []memstore.EntityRetirementPreview `json:"groups"`
}

func (d *Daemon) handleMemoryRetireEntities(_ context.Context, raw json.RawMessage) (any, error) {
	var params MemoryRetireEntitiesParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if len(params.Groups) == 0 {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "no retirement groups given"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	facts, err := st.AllFacts()
	if err != nil {
		return nil, err
	}
	if err := ValidateMemoryRetirementIsolation(params.Groups, facts); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}

	dryRun := params.DryRun == nil || *params.DryRun
	result := &MemoryRetireEntitiesResult{DryRun: dryRun}
	for _, group := range params.Groups {
		preview, err := st.PreviewEntityRetirement(group)
		if err != nil {
			return nil, err
		}
		result.Groups = append(result.Groups, preview)
		if !preview.Ready {
			result.Refused++
			continue
		}
		if !dryRun && !reflect.DeepEqual(group.Expected, preview.Expected) {
			result.Refused++
			result.Groups[len(result.Groups)-1].Problems = append(result.Groups[len(result.Groups)-1].Problems,
				"apply requires exact reviewed entity, fact, and alias fingerprints")
		}
	}
	if dryRun || result.Refused > 0 {
		return result, nil
	}

	backup, err := d.handleMemoryBackup(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("retire-entities: backup first: %w", err)
	}
	result.BackupPath = backup.(*MemoryBackupResult).Path
	for i, group := range params.Groups {
		preview, err := st.RetireEntity(group)
		if err != nil {
			return nil, fmt.Errorf("retire-entities group %s aborted after %d committed group(s) (backup %s): %w", retirementGroupName(group), result.Applied, result.BackupPath, err)
		}
		result.Groups[i] = preview
		result.Applied++
	}
	return result, nil
}

// ValidateMemoryRetirementIsolation ensures sequential group transactions
// cannot invalidate a later group's reviewed inputs. A fact cannot touch two
// retired nodes, and a replacement cannot point at another node in the same
// manifest.
func ValidateMemoryRetirementIsolation(groups []memstore.EntityRetirementRequest, facts []memstore.Fact) error {
	retired := map[string]string{}
	replacementKeys := map[string]int{}
	ids := map[string]string{}
	for _, group := range groups {
		label := retirementGroupName(group)
		if group.Entity == "" {
			return fmt.Errorf("retirement group %s has no entity", label)
		}
		if group.ID != "" {
			if prior := ids[group.ID]; prior != "" {
				return fmt.Errorf("retirement group id %q is repeated for entities %s and %s", group.ID, prior, group.Entity)
			}
			ids[group.ID] = group.Entity
		}
		if prior := retired[group.Entity]; prior != "" {
			return fmt.Errorf("entity %s appears in retirement groups %s and %s", group.Entity, prior, label)
		}
		retired[group.Entity] = label
	}
	for _, fact := range facts {
		srcGroup, srcRetired := retired[fact.Src]
		dstGroup, dstRetired := retired[fact.Dst]
		if srcRetired && dstRetired && fact.Src != fact.Dst {
			return fmt.Errorf("retirement groups %s and %s share fact %s -- %s -- %s; review them in one later design or separate manifests", srcGroup, dstGroup, fact.Src, fact.Relation, fact.Dst)
		}
	}
	for groupIndex, group := range groups {
		for _, replacement := range group.Replacements {
			fact := replacement.Replacement
			key := fmt.Sprintf("%s\x00%s\x00%s\x00%d", fact.Src, fact.Relation, fact.KeyDst(), fact.ValidFrom.UnixNano())
			if priorIndex, found := replacementKeys[key]; found && priorIndex != groupIndex {
				return fmt.Errorf("retirement groups %s and %s propose the same replacement fact key", retirementGroupName(groups[priorIndex]), retirementGroupName(group))
			}
			replacementKeys[key] = groupIndex
			for _, endpoint := range []string{replacement.Replacement.Src, replacement.Replacement.Dst} {
				if other := retired[endpoint]; endpoint != "" && endpoint != group.Entity && other != "" {
					return fmt.Errorf("retirement group %s replacement points at entity %s retired by group %s", retirementGroupName(group), endpoint, other)
				}
			}
		}
		for _, rehome := range group.RehomeAliases {
			if other := retired[rehome.Entity]; rehome.Entity != "" && rehome.Entity != group.Entity && other != "" {
				return fmt.Errorf("retirement group %s rehomes an alias to entity %s retired by group %s", retirementGroupName(group), rehome.Entity, other)
			}
		}
	}
	return nil
}

func retirementGroupName(group memstore.EntityRetirementRequest) string {
	if group.ID != "" {
		return group.ID
	}
	if group.Entity != "" {
		return group.Entity
	}
	return "<unnamed>"
}
