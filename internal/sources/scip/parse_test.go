package scip

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	scipbindings "github.com/scip-code/scip/bindings/go/scip"

	"github.com/jeffdhooton/scry/internal/store"
)

func openTempStore(dir string) (*store.Store, error) {
	return store.Open(filepath.Join(dir, "badger"))
}

// TestParseEndToEndOnFixture parses a real scip-typescript file produced
// by /tmp/scry-graph and verifies that impl edges and call edges are
// emitted into a fresh store. This is the regression hook for the bug
// where stats said Implementations=0 even though the SCIP file had four
// relationship-bearing symbols.
func TestParseEndToEndOnFixture(t *testing.T) {
	path := "/tmp/scry-graph.scip"
	if _, err := os.Stat(path); err != nil {
		t.Skip("fixture /tmp/scry-graph.scip not present; skipping")
	}
	dir := t.TempDir()
	st, err := openTempStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	stats, err := Parse(context.Background(), path, st)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("stats: %+v", stats)
	if stats.Implementations == 0 {
		t.Errorf("expected impl edges, got 0")
	}
	if stats.CallEdges == 0 {
		t.Errorf("expected call edges, got 0")
	}
}

// TestStreamingHasRelationships verifies that the streaming visitor surfaces
// SymbolInformation.Relationships. Our impl-edge output was zero in real
// runs, so this is the smallest reproducible test that proves whether the
// SCIP bindings deliver relationships at all.
func TestStreamingHasRelationships(t *testing.T) {
	path := "/tmp/scry-graph.scip"
	if _, err := os.Stat(path); err != nil {
		t.Skip("fixture /tmp/scry-graph.scip not present; skipping")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var symbolCount, withRel int
	visitor := &scipbindings.IndexVisitor{
		VisitDocument: func(_ context.Context, d *scipbindings.Document) error {
			for _, si := range d.GetSymbols() {
				symbolCount++
				if len(si.GetRelationships()) > 0 {
					withRel++
				}
			}
			return nil
		},
	}
	if err := visitor.ParseStreaming(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	t.Logf("symbols=%d withRelationships=%d", symbolCount, withRel)
	if symbolCount == 0 {
		t.Fatal("got zero symbols")
	}
}
