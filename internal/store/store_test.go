package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "badger")
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSchemaVersionRoundTrip(t *testing.T) {
	s := newTestStore(t)
	v, err := s.SchemaVersionOnDisk()
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	if v != 0 {
		t.Fatalf("fresh store should report 0, got %d", v)
	}
	if err := s.SetMeta("schema_version", SchemaVersion); err != nil {
		t.Fatalf("write: %v", err)
	}
	v, err = s.SchemaVersionOnDisk()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if v != SchemaVersion {
		t.Fatalf("got %d, want %d", v, SchemaVersion)
	}
}

func TestSymbolAndOccurrenceRoundTrip(t *testing.T) {
	s := newTestStore(t)
	w := s.NewWriter()

	sym := &SymbolRecord{
		Symbol:      "scip-typescript . pkg 1.0 src/foo.ts/processOrder().",
		DisplayName: "processOrder",
		Kind:        "Function",
	}
	if err := w.PutSymbol(sym); err != nil {
		t.Fatalf("PutSymbol: %v", err)
	}
	if err := w.PutFileSymbol("src/foo.ts", sym.Symbol); err != nil {
		t.Fatalf("PutFileSymbol: %v", err)
	}

	def := &OccurrenceRecord{
		Symbol:       sym.Symbol,
		File:         "src/foo.ts",
		Line:         10,
		Column:       5,
		IsDefinition: true,
		Context:      "function processOrder(o: Order) {",
	}
	if err := w.PutOccurrence(def); err != nil {
		t.Fatalf("PutOccurrence def: %v", err)
	}

	for i := 0; i < 3; i++ {
		ref := &OccurrenceRecord{
			Symbol: sym.Symbol,
			File:   "src/bar.ts",
			Line:   20 + i,
			Column: 8,
		}
		if err := w.PutOccurrence(ref); err != nil {
			t.Fatalf("PutOccurrence ref %d: %v", i, err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	ids, err := s.LookupSymbolsByName("processOrder")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(ids) != 1 || ids[0] != sym.Symbol {
		t.Fatalf("LookupSymbolsByName: got %v", ids)
	}

	got, err := s.GetSymbol(sym.Symbol)
	if err != nil || got == nil {
		t.Fatalf("GetSymbol: %v %v", got, err)
	}
	if got.DisplayName != "processOrder" {
		t.Fatalf("display: %q", got.DisplayName)
	}

	var refCount int
	if err := s.IterateRefs(sym.Symbol, func(rec *OccurrenceRecord) error {
		refCount++
		if rec.IsDefinition {
			t.Fatalf("ref iteration returned a definition")
		}
		return nil
	}); err != nil {
		t.Fatalf("IterateRefs: %v", err)
	}
	if refCount != 3 {
		t.Fatalf("refCount=%d, want 3", refCount)
	}

	var defCount int
	if err := s.IterateDefs(sym.Symbol, func(rec *OccurrenceRecord) error {
		defCount++
		return nil
	}); err != nil {
		t.Fatalf("IterateDefs: %v", err)
	}
	if defCount != 1 {
		t.Fatalf("defCount=%d, want 1", defCount)
	}
}

func TestCaseInsensitiveLookup(t *testing.T) {
	s := newTestStore(t)
	w := s.NewWriter()
	for _, name := range []string{"OrderService", "orderService"} {
		if err := w.PutSymbol(&SymbolRecord{Symbol: "sym:" + name, DisplayName: name}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	ids, err := s.LookupSymbolsByName("orderservice")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 case-insensitive matches, got %v", ids)
	}
}
