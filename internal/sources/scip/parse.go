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

	visitor := &scipbindings.IndexVisitor{
		VisitMetadata: func(_ context.Context, m *scipbindings.Metadata) error {
			projectRoot = strings.TrimPrefix(m.GetProjectRoot(), "file://")
			if err := st.SetMeta("project_root", projectRoot); err != nil {
				return fmt.Errorf("set project_root: %w", err)
			}
			return nil
		},
		VisitDocument: func(_ context.Context, d *scipbindings.Document) error {
			return processDocument(d, projectRoot, w, &stats)
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
	Documents   int
	Symbols     int
	Definitions int
	References  int
}

func processDocument(d *scipbindings.Document, projectRoot string, w *store.Writer, stats *Stats) error {
	stats.Documents++

	// Read source file once so we can attach a context line to every occurrence.
	// SCIP indexers don't include the source text by default, and resolving line
	// content from the file system is dramatically cheaper than re-running the
	// parser on every query.
	sourceLines := readSourceLines(filepath.Join(projectRoot, d.GetRelativePath()))

	// Symbols defined in this document.
	for _, si := range d.GetSymbols() {
		rec := &store.SymbolRecord{
			Symbol:        si.GetSymbol(),
			DisplayName:   displayName(si),
			Kind:          si.GetKind().String(),
			Documentation: strings.Join(si.GetDocumentation(), "\n"),
		}
		if err := w.PutSymbol(rec); err != nil {
			return err
		}
		if err := w.PutFileSymbol(d.GetRelativePath(), si.GetSymbol()); err != nil {
			return err
		}
		stats.Symbols++
	}

	// Occurrences in this document — both defs and refs.
	for _, occ := range d.GetOccurrences() {
		startLine, startCol, endLine, endCol := decodeRange(occ.GetRange())
		isDef := (occ.GetSymbolRoles() & int32(scipbindings.SymbolRole_Definition)) != 0
		rec := &store.OccurrenceRecord{
			Symbol:       occ.GetSymbol(),
			File:         d.GetRelativePath(),
			Line:         startLine + 1,
			Column:       startCol + 1,
			EndLine:      endLine + 1,
			EndColumn:    endCol + 1,
			IsDefinition: isDef,
			Context:      contextLine(sourceLines, startLine),
		}
		if err := w.PutOccurrence(rec); err != nil {
			return err
		}
		if isDef {
			stats.Definitions++
		} else {
			stats.References++
		}
	}
	return nil
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
