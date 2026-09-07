package scip

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	scip "github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

func TestKindFromSymbol(t *testing.T) {
	const p = "scip-test npm pkg 1 "
	tests := []struct{ id, want string }{
		{p + "pkg/F().", "Method"}, {p + "pkg/C#F(+1).", "Method"},
		{p + "`path with spaces.ts`/`f``oo`().", "Method"},
		{"scip-test npm my  package 1 `a.b`#", "Type"},
		{p + "C#", "Type"}, {p + "C#field.", "Term"}, {p + "`a/b`/", "Namespace"},
		{p + "f().(p)", ""}, {p + "C#[T]", ""}, {p + "meta:", ""}, {p + "macro!", ""},
		{"local 0", ""}, {"local f().", ""}, {"", ""}, {"F().", ""},
		{p, ""}, {p + "`unterminated#", ""}, {p + "F().junk", ""}, {p + "F(", ""},
		{p + "C#bad@().", ""}, {string([]byte{0xff}) + " npm pkg 1 F().", ""},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := KindFromSymbol(tt.id); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestParseKindPriorityAndConservativeFallback(t *testing.T) {
	const p = "scip-test npm pkg 1 "
	// A reference precedes its declaration; an unspecified declaration precedes
	// an authoritative one. Later weak data must not overwrite either kind.
	idx := &scip.Index{Documents: []*scip.Document{
		{RelativePath: "a.ts", Symbols: []*scip.SymbolInformation{{Symbol: p + "Base#"}}, Occurrences: []*scip.Occurrence{{Symbol: p + "Later#", Range: []int32{0, 0, 5}}, {Symbol: p + "External().", Range: []int32{1, 0, 5}}, {Symbol: "local 0", SymbolRoles: 1, Range: []int32{2, 0, 1}}}},
		{RelativePath: "b.ts", Symbols: []*scip.SymbolInformation{
			{Symbol: p + "Later#", Kind: scip.SymbolInformation_Interface, DisplayName: "Later", Documentation: []string{"authoritative"}},
			{Symbol: p + "Base#", Kind: scip.SymbolInformation_Class},
			{Symbol: p + "Value.", Kind: scip.SymbolInformation_Variable},
			{Symbol: p + "F().", Kind: scip.SymbolInformation_Function},
			{Symbol: p + "Ambiguous#", Documentation: []string{"class is just prose, not kind evidence"}},
			{Symbol: "bad#"}, {Symbol: "local 0", Kind: scip.SymbolInformation_Function},
		}, Occurrences: []*scip.Occurrence{{Symbol: p + "Later#", SymbolRoles: 1, Range: []int32{4, 2, 7}}}},
		{RelativePath: "c.ts", Symbols: []*scip.SymbolInformation{{Symbol: p + "Later#"}, {Symbol: p + "Base#"}}},
	}}
	b, err := proto.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "index.scip")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	st, err := openTempStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	stats, err := Parse(context.Background(), path, st)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{p + "Later#": "Interface", p + "Base#": "Class", p + "Value.": "Variable", p + "F().": "Function", p + "Ambiguous#": "Type", p + "External().": "External", "bad#": "UnspecifiedKind"} {
		got, err := st.GetSymbol(id)
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || got.Kind != want {
			t.Errorf("%s got %+v want %s", id, got, want)
		}
	}
	if stats.Symbols != 7 {
		t.Errorf("distinct symbol count=%d want 7", stats.Symbols)
	}
	if got, err := st.GetSymbol("local 0"); err != nil || got != nil {
		t.Errorf("local persisted: %+v %v", got, err)
	}
	got, err := st.GetSymbol(p + "Later#")
	if err != nil {
		t.Fatal(err)
	}
	if got.Documentation != "authoritative" {
		t.Error("late metadata lost")
	}
}

func TestParseExternalKindsOnlyForReferencedSymbols(t *testing.T) {
	const p = "scip-test npm dependency 1 "
	// Encode external metadata first to exercise legal protobuf field reordering.
	ext, err := proto.Marshal(&scip.Index{ExternalSymbols: []*scip.SymbolInformation{
		{Symbol: p + "API#", Kind: scip.SymbolInformation_Interface},
		{Symbol: p + "Unused#", Kind: scip.SymbolInformation_Class},
	}})
	if err != nil {
		t.Fatal(err)
	}
	docs, err := proto.Marshal(&scip.Index{Documents: []*scip.Document{{RelativePath: "caller.ts", Occurrences: []*scip.Occurrence{{Symbol: p + "API#", Range: []int32{0, 0, 3}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "index.scip")
	if err := os.WriteFile(path, append(ext, docs...), 0600); err != nil {
		t.Fatal(err)
	}
	st, err := openTempStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	stats, err := Parse(context.Background(), path, st)
	if err != nil {
		t.Fatal(err)
	}
	api, err := st.GetSymbol(p + "API#")
	if err != nil {
		t.Fatal(err)
	}
	if api == nil || api.Kind != "Interface" || stats.Symbols != 1 {
		t.Errorf("external metadata lost: %+v stats=%+v", api, stats)
	}
	unused, err := st.GetSymbol(p + "Unused#")
	if err != nil {
		t.Fatal(err)
	}
	if unused != nil {
		t.Error("unreferenced external materialized")
	}
}
