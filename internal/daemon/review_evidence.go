package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/memory/recall"
	"github.com/jeffdhooton/scry/internal/review"
	"github.com/jeffdhooton/scry/internal/rpc"
	"github.com/jeffdhooton/scry/internal/store"
)

// Structural results are retrieval hints from the index, paired with current
// source bytes. We never present an index timestamp as a source snapshot.
func (d *Daemon) reviewEnricher(includeMemory bool, exclude []string, memorySocket string) review.Enricher {
	return func(ctx context.Context, snap *review.Snapshot) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		changed := map[string]bool{}
		for _, f := range snap.ChangedFiles {
			if !review.PathExcluded(f, exclude) {
				changed[f] = true
			}
		}
		symbols := []string{}
		names := []string{}
		entry, err := d.registry.Get(d.scryHome(), snap.Repository)
		if err != nil {
			snap.Warnings = append(snap.Warnings, "structural index unavailable; caller coverage incomplete")
		} else {
			d.registry.mu.RLock()
			// Registry swaps close stores under the write lock; hold the read lock
			// across traversal to keep all occurrences from one index generation.
			current := d.registry.entries[entry.RepoPath]
			if current != entry {
				d.registry.mu.RUnlock()
				return errors.New("code index changed during review evidence assembly")
			}
			found := map[string]bool{}
			err = entry.Store.IterateAllDefs(func(o *store.OccurrenceRecord) error {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if changed[o.File] && !found[o.Symbol] && len(symbols) < 12 {
					found[o.Symbol] = true
					symbols = append(symbols, o.Symbol)
				}
				return nil
			})
			sort.Strings(symbols)
			sourceFiles := map[string]bool{}
			structuralBytes := 0
			for _, symbol := range symbols {
				sym, e := entry.Store.GetSymbol(symbol)
				if e != nil {
					err = e
					break
				}
				if sym == nil {
					continue
				}
				names = append(names, sym.DisplayName)
				refs := []store.OccurrenceRecord{}
				e = entry.Store.IterateRefs(symbol, func(o *store.OccurrenceRecord) error {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if len(refs) < 30 && !review.PathExcluded(o.File, exclude) {
						copy := *o
						copy.Context = ""
						refs = append(refs, copy)
						sourceFiles[o.File] = true
					}
					return nil
				})
				if e != nil {
					err = e
					break
				}
				// Symbol documentation/context may contain stale source or secrets; only
				// names and locations leave the index. Current caller bytes are below.
				b, _ := json.Marshal(map[string]any{"symbol": sym.DisplayName, "references": refs, "coverage": "indexed references; may be incomplete or stale; confirm against captured source"})
				if structuralBytes+len(b) <= 6000 {
					structuralBytes += len(b)
					snap.Evidence = append(snap.Evidence, review.Evidence{ID: reviewEvidenceID("structural", symbol), Kind: "structural", Content: string(b)})
				}
			}
			d.registry.mu.RUnlock()
			if err != nil {
				snap.Warnings = append(snap.Warnings, "structural lookup incomplete")
			}
			files := []string{}
			for f := range sourceFiles {
				if !changed[f] {
					files = append(files, f)
				}
			}
			sort.Strings(files)
			for i, f := range files {
				if i >= 6 {
					snap.Warnings = append(snap.Warnings, "caller source limit reached")
					break
				}
				content, e := review.ReadEvidenceFile(ctx, snap.Repository, f, 1000)
				if e != nil {
					snap.Warnings = append(snap.Warnings, "caller source unavailable: "+f)
					continue
				}
				snap.Evidence = append(snap.Evidence, review.Evidence{ID: reviewEvidenceID("caller", f), Kind: "caller", Path: f, Content: content})
			}
			if m, e := index.LoadManifest(entry.Layout); e != nil || m.HeadCommit != snap.Head {
				snap.Warnings = append(snap.Warnings, "index freshness unverified; structural locations are retrieval hints")
			}
		}
		if includeMemory {
			sort.Strings(names)
			parts := []string{filepath.Base(snap.Repository)}
			parts = append(parts, names...)
			// File basenames help retrieval when a language index is unavailable.
			if len(names) == 0 {
				for _, f := range snap.ChangedFiles {
					if !review.PathExcluded(f, exclude) && len(parts) < 6 {
						parts = append(parts, filepath.Base(f))
					}
				}
			}
			query := strings.Join(parts, " ") + " decision exception constraint"
			if len(query) > 1000 {
				query = query[:1000]
			}
			result, e := d.reviewRecall(ctx, memorySocket, query)
			if e != nil {
				snap.Warnings = append(snap.Warnings, "memory unavailable; prior decisions not checked")
			} else {
				facts := []review.Evidence{}
				memoryBytes := 0
				for _, f := range result.Facts {
					if f.InvalidAt != nil {
						continue
					}
					f.Score = 0
					b, _ := json.Marshal(f)
					if len(b) > 4000 || memoryBytes+len(b) > 8000 {
						snap.Warnings = append(snap.Warnings, "large memory fact omitted")
						continue
					}
					memoryBytes += len(b)
					id := reviewEvidenceID("memory", f.Src+f.Relation+f.Dst+f.Value+f.ValidFrom.Format(time.RFC3339Nano))
					facts = append(facts, review.Evidence{ID: id, Kind: "memory", Content: string(b)})
				}
				sort.Slice(facts, func(i, j int) bool { return facts[i].ID < facts[j].ID })
				snap.Evidence = append(snap.Evidence, facts...)
			}
		}
		return ctx.Err()
	}
}
func reviewEvidenceID(kind, value string) string {
	h := sha256.Sum256([]byte(value))
	return kind + ":" + hex.EncodeToString(h[:8])
}
func (d *Daemon) reviewRecall(ctx context.Context, socket, query string) (recall.Result, error) {
	if env := os.Getenv("SCRY_MEMORY_SOCKET"); env != "" {
		socket = env
	}
	if strings.HasPrefix(socket, "~/") {
		if home, e := os.UserHomeDir(); e == nil {
			socket = filepath.Join(home, socket[2:])
		}
	}
	params := MemoryRecallParams{Query: query, Limit: 4}
	if socket != "" {
		c, e := rpc.Dial(socket)
		if e != nil {
			return recall.Result{}, e
		}
		defer c.Close()
		var out recall.Result
		e = c.Call(ctx, "memory.recall", params, &out)
		return out, e
	}
	b, _ := json.Marshal(params)
	v, e := d.handleMemoryRecall(ctx, b)
	if e != nil {
		return recall.Result{}, e
	}
	out, ok := v.(recall.Result)
	if !ok {
		return out, fmt.Errorf("unexpected memory response")
	}
	return out, nil
}
