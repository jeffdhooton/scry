// Package scip parses a SCIP protobuf index produced by an upstream indexer
// (scip-typescript, scip-go, scip-php, ...) and writes normalized records into
// a scry store.
//
// The on-disk SCIP format is documented at
// https://github.com/scip-code/scip/blob/main/scip.proto. We rely on the Go
// bindings published at github.com/scip-code/scip/bindings/go/scip.
//
// SCIP positions are 0-indexed; we convert to 1-indexed before storing because
// every other surface in scry (CLI output, error messages, agent expectations)
// is 1-indexed.
package scip

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	scipbindings "github.com/scip-code/scip/bindings/go/scip"

	"github.com/jeffdhooton/scry/internal/store"
)

// Parse reads scipPath, walks every Document in it, and writes the normalized
// records into st through a single batched writer. projectRoot is the absolute
// path the relative paths in the SCIP file are anchored to; if empty we read
// it from the index Metadata.
func Parse(ctx context.Context, scipPath string, st *store.Store) (Stats, error) {
	f, err := os.Open(scipPath)
	if err != nil {
		return Stats{}, fmt.Errorf("open scip file: %w", err)
	}
	defer f.Close()

	w := st.NewWriter()
	stats := Stats{}
	var projectRoot string
	// seenSymbols tracks every symbol id we've already PutSymbol'd for in this
	// indexing run. We use it to (a) avoid re-writing duplicate SymbolRecords
	// across documents and (b) decide whether an occurrence-only symbol needs
	// a synthesized SymbolRecord at the end of processing.
	seenSymbols := map[string]bool{}

	visitor := &scipbindings.IndexVisitor{
		VisitMetadata: func(_ context.Context, m *scipbindings.Metadata) error {
			projectRoot = strings.TrimPrefix(m.GetProjectRoot(), "file://")
			if err := st.SetMeta("project_root", projectRoot); err != nil {
				return fmt.Errorf("set project_root: %w", err)
			}
			return nil
		},
		VisitDocument: func(_ context.Context, d *scipbindings.Document) error {
			return processDocument(d, projectRoot, w, &stats, seenSymbols)
		},
	}
	if err := visitor.ParseStreaming(ctx, f); err != nil {
		return stats, fmt.Errorf("parse scip stream: %w", err)
	}
	if err := w.Flush(); err != nil {
		return stats, fmt.Errorf("flush writer: %w", err)
	}
	return stats, nil
}

// Stats is what Parse returns to the caller for logging.
type Stats struct {
	Documents       int
	Symbols         int
	Definitions     int
	References      int
	CallEdges       int
	Implementations int
}

// scope is one definition occurrence with an enclosing AST range, used to
// resolve "what function contains this position?" inside a single document.
type scope struct {
	symbolID  string
	startLine int
	startCol  int
	endLine   int
	endCol    int
}

// contains reports whether (line, col) falls inside this scope's range.
// SCIP positions are 0-indexed; this function works in either basis as long
// as it's used consistently.
func (s scope) contains(line, col int) bool {
	if line < s.startLine || line > s.endLine {
		return false
	}
	if line == s.startLine && col < s.startCol {
		return false
	}
	if line == s.endLine && col > s.endCol {
		return false
	}
	return true
}

// area returns a comparable size for the scope's bounding box. Used to pick
// the smallest enclosing scope when several match (innermost wins).
func (s scope) area() int {
	return (s.endLine-s.startLine)*1_000_000 + (s.endCol - s.startCol)
}

func processDocument(d *scipbindings.Document, projectRoot string, w *store.Writer, stats *Stats, seenSymbols map[string]bool) error {
	stats.Documents++

	// Read source file once so we can attach a context line to every occurrence.
	// SCIP indexers don't include the source text by default, and resolving line
	// content from the file system is dramatically cheaper than re-running the
	// parser on every query.
	sourceLines := readSourceLines(filepath.Join(projectRoot, d.GetRelativePath()))

	// Symbols defined in this document. Also harvest implementation
	// relationships at the same time.
	for _, si := range d.GetSymbols() {
		// SCIP local symbols ("local N") are document-scoped — the same id
		// in different documents represents different variables. We don't
		// expose locals as cross-file query targets (agents rarely ask "find
		// every use of a local variable named `i`") so skip them in the name
		// index entirely.
		if isLocalSymbol(si.GetSymbol()) {
			continue
		}
		if !seenSymbols[si.GetSymbol()] {
			rec := &store.SymbolRecord{
				Symbol:        si.GetSymbol(),
				DisplayName:   displayName(si),
				Kind:          si.GetKind().String(),
				Documentation: strings.Join(si.GetDocumentation(), "\n"),
			}
			if err := w.PutSymbol(rec); err != nil {
				return err
			}
			seenSymbols[si.GetSymbol()] = true
			stats.Symbols++
		}
		if err := w.PutFileSymbol(d.GetRelativePath(), si.GetSymbol()); err != nil {
			return err
		}

		// Implementation edges. SCIP records "Dog implements Animal" by
		// putting a Relationship on Dog's SymbolInformation with
		// Symbol="Animal#" and IsImplementation=true. We invert that into
		// impl:<animal>:<dog> for fast `scry impls Animal#` lookups.
		for _, rel := range si.GetRelationships() {
			if !rel.GetIsImplementation() {
				continue
			}
			if err := w.PutImplEdge(rel.GetSymbol(), si.GetSymbol()); err != nil {
				return err
			}
			stats.Implementations++
		}
	}

	// First pass over occurrences: collect every def-occurrence with an
	// enclosing range. These define the per-document "scopes" we use to
	// resolve containing_symbol on each ref. scip-typescript populates
	// enclosing_range; scip-go does not, so this list will be empty for Go
	// documents and containing_symbol will fall back to "".
	var scopes []scope
	for _, occ := range d.GetOccurrences() {
		if isLocalSymbol(occ.GetSymbol()) {
			continue
		}
		isDef := (occ.GetSymbolRoles() & int32(scipbindings.SymbolRole_Definition)) != 0
		if !isDef {
			continue
		}
		er := occ.GetEnclosingRange()
		if len(er) == 0 {
			continue
		}
		sl, sc, el, ec := decodeRange(er)
		scopes = append(scopes, scope{
			symbolID:  occ.GetSymbol(),
			startLine: sl,
			startCol:  sc,
			endLine:   el,
			endCol:    ec,
		})
	}

	// Second pass: write occurrences and derive containing_symbol + callee
	// edges.
	for _, occ := range d.GetOccurrences() {
		if isLocalSymbol(occ.GetSymbol()) {
			continue
		}
		// Synthesize a SymbolRecord for any occurrence whose symbol id was not
		// declared in any document's SymbolInformation list. This happens for
		// every external reference (Illuminate facades, vendor classes,
		// stdlib types) — scip-php and scip-go only emit SymbolInformation for
		// definitions inside the indexed source tree, but the source tree
		// references thousands of external symbols. Without this synthesis
		// `scry refs DB` would return zero even though every Laravel app calls
		// it constantly.
		if !seenSymbols[occ.GetSymbol()] {
			rec := &store.SymbolRecord{
				Symbol:      occ.GetSymbol(),
				DisplayName: deriveDisplayName(occ.GetSymbol()),
				Kind:        "External",
			}
			if err := w.PutSymbol(rec); err != nil {
				return err
			}
			seenSymbols[occ.GetSymbol()] = true
			stats.Symbols++
		}
		startLine, startCol, endLine, endCol := decodeRange(occ.GetRange())
		isDef := (occ.GetSymbolRoles() & int32(scipbindings.SymbolRole_Definition)) != 0

		containing := smallestEnclosingScope(scopes, startLine, startCol, occ.GetSymbol(), isDef)
		var containingID string
		if containing != nil {
			containingID = containing.symbolID
		}

		rec := &store.OccurrenceRecord{
			Symbol:           occ.GetSymbol(),
			File:             d.GetRelativePath(),
			Line:             startLine + 1,
			Column:           startCol + 1,
			EndLine:          endLine + 1,
			EndColumn:        endCol + 1,
			IsDefinition:     isDef,
			Context:          contextLine(sourceLines, startLine),
			ContainingSymbol: containingID,
		}
		if err := w.PutOccurrence(rec); err != nil {
			return err
		}
		if isDef {
			stats.Definitions++
		} else {
			stats.References++
			// Emit a callee edge: containing function calls this occurrence's
			// symbol. Skip self-edges (containing == ref symbol, e.g.
			// recursive references).
			if containingID != "" && containingID != occ.GetSymbol() {
				if err := w.PutCalleeEdge(containingID, rec); err != nil {
					return err
				}
				stats.CallEdges++
			}
		}
	}
	return nil
}

// smallestEnclosingScope picks the deepest scope that contains (line, col).
// When isDef is true the scope MUST not be the symbol itself — a function
// definition is not "contained by" its own body, even though the def
// occurrence's range is inside its body. Without this guard, a recursive
// helper that defines itself looks like a self-edge.
func smallestEnclosingScope(scopes []scope, line, col int, occSymbol string, isDef bool) *scope {
	var best *scope
	for i := range scopes {
		s := &scopes[i]
		if isDef && s.symbolID == occSymbol {
			continue
		}
		if !s.contains(line, col) {
			continue
		}
		if best == nil || s.area() < best.area() {
			best = s
		}
	}
	return best
}

// isLocalSymbol returns true for SCIP local symbol ids ("local N") which are
// scoped to a single document and would collide if stored under a global
// keyspace.
func isLocalSymbol(symbol string) bool {
	return strings.HasPrefix(symbol, "local ")
}

// decodeRange unpacks the SCIP packed range encoding (3 or 4 ints, 0-indexed).
//
// SCIP encodes a range as either:
//   - [startLine, startChar, endLine, endChar]
//   - [startLine, startChar, endChar]   (single-line; endLine == startLine)
func decodeRange(r []int32) (sl, sc, el, ec int) {
	switch len(r) {
	case 4:
		return int(r[0]), int(r[1]), int(r[2]), int(r[3])
	case 3:
		return int(r[0]), int(r[1]), int(r[0]), int(r[2])
	default:
		return 0, 0, 0, 0
	}
}

// displayName returns SymbolInformation.display_name if present, otherwise
// derives a sensible name from the symbol id's last descriptor. Some indexers
// (notably older scip-typescript builds) don't populate display_name, and we
// must always have something queryable.
func displayName(si *scipbindings.SymbolInformation) string {
	if n := si.GetDisplayName(); n != "" {
		return n
	}
	return deriveDisplayName(si.GetSymbol())
}

// deriveDisplayName parses a SCIP symbol id and returns the name of the last
// descriptor. Falls back to the raw id if parsing fails.
func deriveDisplayName(symbol string) string {
	parsed, err := scipbindings.ParseSymbol(symbol)
	if err != nil || len(parsed.GetDescriptors()) == 0 {
		return symbol
	}
	return parsed.GetDescriptors()[len(parsed.GetDescriptors())-1].GetName()
}

// readSourceLines reads a file once and returns its lines as a slice. Returns
// nil on error — context-line attachment is best-effort, never fatal.
func readSourceLines(absPath string) []string {
	f, err := os.Open(absPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func contextLine(lines []string, lineIdx int) string {
	if lineIdx < 0 || lineIdx >= len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[lineIdx])
}
