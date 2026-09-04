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
	// Fact is the sentence the reviewer read. It is compared against the
	// stored text, so a fact re-extracted into different words under the
	// same key is refused rather than moved on the strength of a review
	// that was about something else.
	Fact string `json:"fact,omitempty"`
	To   string `json:"to"`
	// Side names the endpoint to move, "src" (the default) or "dst". A
	// fact can be filed under the wrong entity from either end: "the
	// feedback digest job runs on the hermes Mac mini" was stored with
	// the project as its destination.
	Side string `json:"side,omitempty"`
	Why  string `json:"why,omitempty"`
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
	DryRun     bool   `json:"dry_run"`
	BackupPath string `json:"backup_path,omitempty"`
	Moved      int    `json:"moved"`
	Refused    int    `json:"refused"`
	// Warned counts moves that go ahead onto a destination already
	// asserting the same relation and target in different words.
	Warned  int      `json:"warned"`
	Details []string `json:"details"`
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
		side := m.Side
		if side == "" {
			side = "src"
		}
		if side != "src" && side != "dst" {
			res.Refused++
			res.Details = append(res.Details, "refused: side must be src or dst, got "+side)
			continue
		}
		moving := m.Src
		if side == "dst" {
			moving = m.Dst
		}
		if m.To == "" || m.To == moving {
			res.Refused++
			res.Details = append(res.Details, "refused: destination missing or unchanged: "+m.Relation)
			continue
		}
		if _, err := st.GetEntity(m.To); err != nil {
			res.Refused++
			res.Details = append(res.Details, "refused: destination "+m.To+" does not exist")
			continue
		}
		if (side == "src" && m.To == m.Dst) || (side == "dst" && m.To == m.Src) {
			res.Refused++
			res.Details = append(res.Details, "refused: would be a self-loop: "+m.Src+" -> "+m.Dst)
			continue
		}
		// Current facts only. An invalidated fact is history; moving it
		// rewrites the past rather than repairing the present.
		facts, err := st.FactsFrom(m.Src, false)
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
		if m.Fact != "" && m.Fact != found.Fact {
			res.Refused++
			res.Details = append(res.Details, "refused: the fact now reads differently than the list says: "+truncate(found.Fact, 60))
			continue
		}
		// What the destination already holds decides whether this is a
		// repair or a duplicate. Two of the first fourteen moves proposed
		// against the live store collided with a fact the destination
		// already carried, at the identical valid-from, and RelocateFact
		// resolves that by nudging the timestamp a nanosecond until the
		// key is free. That is a silent answer to a real question, so the
		// question is asked here instead.
		// Both directions, because a dst move lands an edge pointing AT
		// the destination and FactsFrom only walks outward from it.
		dupes, err := st.FactsAbout(m.To, false)
		if err != nil {
			return nil, err
		}
		var clash, sameEdge *memstore.Fact
		for i := range dupes {
			f := &dupes[i]
			if side == "dst" {
				// Moving the far end: a duplicate is the same source
				// asserting the same relation at the new destination.
				if f.Relation != m.Relation || f.Src != m.Src || f.Dst != m.To {
					continue
				}
			} else if f.Src != m.To || f.Relation != m.Relation || f.Dst != m.Dst || f.Value != m.Value {
				continue
			}
			sameEdge = f
			if f.ValidFrom.Equal(found.ValidFrom) {
				clash = f
			}
		}
		if clash != nil {
			res.Refused++
			res.Details = append(res.Details, "refused: "+m.To+" already holds this exact fact at the same time: "+truncate(clash.Fact, 60))
			continue
		}
		if sameEdge != nil {
			res.Warned++
			res.Details = append(res.Details, "warning: "+m.To+" already says "+m.Relation+" "+m.Dst+m.Value+": "+truncate(sameEdge.Fact, 60))
		}
		updated := *found
		if side == "dst" {
			updated.Dst = m.To
		} else {
			updated.Src = m.To
		}
		res.Moved++
		res.Details = append(res.Details, moving+" -> "+m.To+" ("+side+"): "+m.Relation+" "+m.Dst+m.Value)
		if p.DryRun {
			continue
		}
		if err := st.RelocateFact(*found, updated); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// truncate bounds a fact sentence for a report line.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
