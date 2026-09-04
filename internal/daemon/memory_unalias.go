package daemon

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jeffdhooton/scry/internal/rpc"
)

// MemoryUnaliasDrop names one spelling to take off one entity.
type MemoryUnaliasDrop struct {
	Entity string `json:"entity"`
	Alias  string `json:"alias"`
	Why    string `json:"why,omitempty"`
}

// MemoryUnaliasParams is a reviewed list of alias drops.
type MemoryUnaliasParams struct {
	Drops  []MemoryUnaliasDrop `json:"drops"`
	DryRun bool                `json:"dry_run"`
}

// MemoryUnaliasResult reports what was dropped and what was refused.
type MemoryUnaliasResult struct {
	DryRun     bool     `json:"dry_run"`
	BackupPath string   `json:"backup_path,omitempty"`
	Dropped    int      `json:"dropped"`
	Refused    int      `json:"refused"`
	Details    []string `json:"details"`
}

// handleMemoryUnalias removes named spellings from named entities.
//
// Moving a fact off an entity does not stop the next episode putting it
// back. The project hermes-ops answers to "Hermes Slack gateway" and
// "Jeff's own Hermes", so every extraction that mentions the gateway
// resolves to the project and refiles there — a reviewer's phrase for
// reattaching without this was "bailing a boat with the hole still in it".
//
// Rules refuse these at admission already; what has no safe general answer
// is deciding which stored spellings to strip, because three attempts to do
// that by rule were built, measured, and thrown away (see docs/DECISIONS.md).
// So this takes a list, like reattach: someone reads the aliases and says
// which ones lie.
func (d *Daemon) handleMemoryUnalias(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryUnaliasParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if len(p.Drops) == 0 {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "no drops given"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	res := &MemoryUnaliasResult{DryRun: p.DryRun}
	if !p.DryRun {
		b, err := d.handleMemoryBackup(context.Background(), nil)
		if err != nil {
			return nil, fmt.Errorf("unalias: backup first: %w", err)
		}
		res.BackupPath = b.(*MemoryBackupResult).Path
	}
	for _, drop := range p.Drops {
		e, err := st.GetEntity(drop.Entity)
		if err != nil {
			res.Refused++
			res.Details = append(res.Details, "refused: no entity "+drop.Entity)
			continue
		}
		// The entity's own name is not an alias and must never be dropped.
		if drop.Alias == e.Name || drop.Alias == e.Slug {
			res.Refused++
			res.Details = append(res.Details, "refused: "+drop.Alias+" is the entity's own name")
			continue
		}
		held := false
		for _, a := range e.Aliases {
			if a == drop.Alias {
				held = true
				break
			}
		}
		if !held {
			res.Refused++
			res.Details = append(res.Details, "refused: "+drop.Entity+" does not list "+drop.Alias)
			continue
		}
		res.Dropped++
		res.Details = append(res.Details, drop.Entity+" drops "+drop.Alias)
		if p.DryRun {
			continue
		}
		if _, err := st.DropAlias(drop.Entity, drop.Alias); err != nil {
			return nil, err
		}
	}
	return res, nil
}
