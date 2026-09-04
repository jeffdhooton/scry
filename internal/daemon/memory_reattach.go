package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	memstore "github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// MemoryReattachMove names one fact and the entity it should hang from.
// The fact is identified exactly, by the same four fields that key it, so a
// move can never match a fact the caller did not read.
type MemoryReattachMove struct {
	Src       string    `json:"src"`
	Relation  string    `json:"relation"`
	Dst       string    `json:"dst,omitempty"`
	Value     string    `json:"value,omitempty"`
	ValidFrom time.Time `json:"valid_from"`
	To        string    `json:"to"`
	Why       string    `json:"why,omitempty"`
}

// MemoryReattachParams is a reviewed list of moves. DryRun reports what
// would happen and writes nothing.
type MemoryReattachParams struct {
	Moves  []MemoryReattachMove `json:"moves"`
	DryRun bool                 `json:"dry_run"`
}

// MemoryReattachResult reports what was moved and what was refused, with a
// reason per refusal, so a partially-valid list is still useful.
type MemoryReattachResult struct {
	DryRun     bool     `json:"dry_run"`
	BackupPath string   `json:"backup_path,omitempty"`
	Moved      int      `json:"moved"`
	Refused    int      `json:"refused"`
	Details    []string `json:"details"`
}

// handleMemoryReattach moves named facts from one entity to another.
//
// This is deliberately not a rule. Three attempts to move facts at store
// scale by inferring the right owner from a fact's text were built,
// measured, and thrown away — the reasoning is in docs/DECISIONS.md under
// "Facts cannot be refiled by what their sentence names" and "The write
// path's alias test cannot be run over stored aliases". What survives of
// that work is this: a caller reads the facts, decides one at a time, and
// hands in a list that a human or a grader can check line by line.
//
// Every move is verified against the stored fact before it is applied. A
// fact whose text has changed, or that has been invalidated, or whose
// destination does not exist, is refused rather than guessed at.
func (d *Daemon) handleMemoryReattach(_ context.Context, raw json.RawMessage) (any, error) {
	var p MemoryReattachParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	if len(p.Moves) == 0 {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "no moves given"}
	}
	st, err := d.memoryStore()
	if err != nil {
		return nil, err
	}
	res := &MemoryReattachResult{DryRun: p.DryRun}

	if !p.DryRun {
		b, err := d.handleMemoryBackup(context.Background(), nil)
		if err != nil {
			return nil, fmt.Errorf("reattach: backup first: %w", err)
		}
		res.BackupPath = b.(*MemoryBackupResult).Path
	}

	for _, m := range p.Moves {
		if m.To == "" || m.To == m.Src {
			res.Refused++
			res.Details = append(res.Details, "refused: destination missing or unchanged: "+m.Relation)
			continue
		}
		if _, err := st.GetEntity(m.To); err != nil {
			res.Refused++
			res.Details = append(res.Details, "refused: destination "+m.To+" does not exist")
			continue
		}
		if m.To == m.Dst {
			res.Refused++
			res.Details = append(res.Details, "refused: would be a self-loop: "+m.Src+" -> "+m.Dst)
			continue
		}
		facts, err := st.FactsFrom(m.Src, true)
		if err != nil {
			return nil, err
		}
		var found *memstore.Fact
		for i := range facts {
			f := &facts[i]
			if f.Relation == m.Relation && f.Dst == m.Dst && f.Value == m.Value && f.ValidFrom.Equal(m.ValidFrom) {
				found = f
				break
			}
		}
		if found == nil {
			res.Refused++
			res.Details = append(res.Details, "refused: no current fact "+m.Src+" -["+m.Relation+"]-> "+m.Dst+m.Value+" at that time")
			continue
		}
		updated := *found
		updated.Src = m.To
		res.Moved++
		res.Details = append(res.Details, m.Src+" -> "+m.To+": "+m.Relation+" "+m.Dst+m.Value)
		if p.DryRun {
			continue
		}
		if err := st.RelocateFact(*found, updated); err != nil {
			return nil, err
		}
	}
	return res, nil
}
