package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

type MemoryUnaliasDrop = memstore.AliasDrop

// A bare list is sufficient to preview. Apply requires the complete review
// fingerprints. Omitted dry_run is safe for raw RPC as well as CLI callers.
type MemoryUnaliasParams struct {
	Drops    []MemoryUnaliasDrop           `json:"drops"`
	Expected *memstore.AliasRepairExpected `json:"expected,omitempty"`
	DryRun   *bool                         `json:"dry_run,omitempty"`
}

type MemoryUnaliasResult struct {
	DryRun     bool                        `json:"dry_run"`
	BackupPath string                      `json:"backup_path,omitempty"`
	Dropped    int                         `json:"dropped"`
	Refused    int                         `json:"refused"`
	Details    []string                    `json:"details"`
	Preview    memstore.AliasRepairPreview `json:"preview"`
}

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
	req := memstore.AliasRepairRequest{Drops: p.Drops, Expected: p.Expected}
	preview, err := st.PreviewAliasRepair(req)
	if err != nil {
		return nil, err
	}
	dryRun := p.DryRun == nil || *p.DryRun
	res := &MemoryUnaliasResult{DryRun: dryRun, Preview: preview, Details: []string{}}
	if !preview.Ready || (!dryRun && p.Expected == nil) {
		res.Refused = len(p.Drops)
		res.Details = append(res.Details, preview.Problems...)
		if !dryRun && p.Expected == nil {
			res.Details = append(res.Details, "apply requires exact reviewed fingerprints from a dry run")
		}
		return res, nil
	}
	for _, drop := range p.Drops {
		detail := drop.Entity + " drops " + drop.Alias
		if drop.RehomeTo != "" {
			detail += " -> " + drop.RehomeTo
		}
		res.Details = append(res.Details, detail)
	}
	if dryRun {
		res.Dropped = len(p.Drops)
		return res, nil
	}
	path, backup, err := d.createMemoryBackupFile("")
	if err != nil {
		return nil, fmt.Errorf("unalias: backup first: %w", err)
	}
	res.BackupPath = path
	_, preview, err = st.BackupAndRepairAliases(backup, req)
	if err != nil {
		// Retain the backup for diagnosis even when drift aborts apply.
		return nil, fmt.Errorf("unalias batch aborted atomically (backup %s): %w", path, err)
	}
	res.Preview = preview
	res.Dropped = len(p.Drops)
	return res, nil
}
