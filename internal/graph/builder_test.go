package graph

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	gitstore "github.com/jeffdhooton/scry/internal/git/store"
	graphstore "github.com/jeffdhooton/scry/internal/graph/store"
	"github.com/jeffdhooton/scry/internal/schema"
	schemastore "github.com/jeffdhooton/scry/internal/schema/store"
	codestore "github.com/jeffdhooton/scry/internal/store"
)

func checkGraphErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClassifySymbolPreservesAuthoritativeKinds(t *testing.T) {
	const p = "scip-test npm pkg 1 "
	for _, tt := range []struct{ kind, id, want string }{
		{"UnspecifiedKind", p + "F().", "function"}, {"", p + "C#", "type"},
		{"External", p + "C#", "type"}, {"UnspecifiedKind", p + "I#m.", "term"},
		{"Class", p + "C#", "class"}, {"Struct", p + "C#", "class"},
		{"Interface", p + "I#", "interface"}, {"Type", p + "T#", "type"},
		{"Function", p + "f.", "function"}, {"Method", p + "I#m.", "function"},
		{"Variable", p + "f().", ""}, {"FutureKind", p + "C#", ""},
		{"UnspecifiedKind", "malformed#", ""}, {"External", "bad F().", ""},
		{"Function", "local 0", ""}, {"UnspecifiedKind", p + "f().(x)", ""},
	} {
		t.Run(tt.kind+tt.id, func(t *testing.T) {
			got := classifySymbol(&codestore.SymbolRecord{Kind: tt.kind, Symbol: tt.id})
			if got != tt.want {
				t.Errorf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestCodeGraphMixedClassificationEndpoints(t *testing.T) {
	const p = "scip-test npm pkg 1 "
	home, repo := t.TempDir(), t.TempDir()
	code, err := codestore.Open(filepath.Join(t.TempDir(), "code"))
	checkGraphErr(t, err)
	defer code.Close()
	w := code.NewWriter()
	for _, s := range []codestore.SymbolRecord{
		{Symbol: p + "Caller().", DisplayName: "Caller", Kind: "Function"},
		{Symbol: p + "Callee().", DisplayName: "Callee", Kind: "UnspecifiedKind"},
		{Symbol: p + "Base#", DisplayName: "Base", Kind: "Interface"},
		{Symbol: p + "Impl#", DisplayName: "Impl", Kind: "UnspecifiedKind"},
		{Symbol: p + "External().", DisplayName: "External", Kind: "External"},
		{Symbol: "malformed#", DisplayName: "Malformed", Kind: "UnspecifiedKind"},
		{Symbol: "local 0", DisplayName: "Local", Kind: "Function"},
	} {
		s := s
		checkGraphErr(t, w.PutSymbol(&s))
	}
	checkGraphErr(t, w.PutOccurrence(&codestore.OccurrenceRecord{Symbol: p + "Caller().", File: "caller.ts", Line: 12, Column: 3, IsDefinition: true}))
	for _, id := range []string{p + "Callee().", p + "External().", "malformed#", "local 0"} {
		checkGraphErr(t, w.PutCalleeEdge(p+"Caller().", &codestore.OccurrenceRecord{Symbol: id, File: "caller.ts", Line: 13}))
	}
	checkGraphErr(t, w.PutImplEdge(p+"Base#", p+"Impl#"))
	checkGraphErr(t, w.Flush())
	_, err = Build(home, repo, Sources{Code: code})
	checkGraphErr(t, err)
	g, err := graphstore.Open(Layout(home, repo).BadgerDir)
	checkGraphErr(t, err)
	defer g.Close()
	nodes, err := g.AllNodes()
	checkGraphErr(t, err)
	edges, err := g.AllEdges()
	checkGraphErr(t, err)
	if len(nodes) != 5 || len(edges) != 3 {
		t.Fatalf("nodes=%+v edges=%+v", nodes, edges)
	}
	for _, e := range edges {
		for _, key := range []string{e.SrcKey, e.DstKey} {
			n, err := g.GetNode(key)
			checkGraphErr(t, err)
			if n == nil {
				t.Errorf("dangling endpoint %s", key)
			}
		}
	}
	ext, err := g.GetNode("function:" + p + "External().")
	checkGraphErr(t, err)
	if ext.File != "" || ext.Line != 0 || ext.Metadata["kind"] != "External" {
		t.Errorf("invented external definition/kind: %+v", ext)
	}
	for _, pair := range [][2]string{{"Impl", "Base"}, {"Base", "Impl"}} {
		path, err := FindPath(g, pair[0], pair[1])
		checkGraphErr(t, err)
		if !path.Found || !reflect.DeepEqual(path.Edges, []string{"implements"}) || len(path.Relationships) != 1 || path.Relationships[0].SrcKey != "type:"+p+"Impl#" {
			t.Errorf("wrong stored relation/direction: %+v", path)
		}
	}
}

func TestGraphGitSchemaContributions(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	gs, err := gitstore.Open(filepath.Join(t.TempDir(), "git"))
	checkGraphErr(t, err)
	defer gs.Close()
	gw := gs.NewWriter()
	checkGraphErr(t, gw.PutChurn("a.go", &gitstore.ChurnRecord{Path: "a.go", CommitCount: 4}))
	checkGraphErr(t, gw.PutCochange("a.go", "b.go", &gitstore.CochangeRecord{FileA: "a.go", FileB: "b.go", Count: 4}))
	checkGraphErr(t, gw.PutContrib("a.go", "fixture@example.invalid", &gitstore.ContribRecord{Author: "Fixture Author", CommitCount: 4}))
	checkGraphErr(t, gw.Flush())
	ss, err := schemastore.Open(filepath.Join(t.TempDir(), "schema"))
	checkGraphErr(t, err)
	defer ss.Close()
	sw := ss.NewWriter()
	for _, table := range []schema.TableRecord{{Name: "parents", Type: "table"}, {Name: "children", Type: "table", ForeignKeys: []schema.ForeignKeyRecord{{ReferencedTable: "parents"}}}} {
		b, err := json.Marshal(table)
		checkGraphErr(t, err)
		checkGraphErr(t, sw.PutTable(table.Name, b))
	}
	checkGraphErr(t, sw.Flush())
	cs, err := codestore.Open(filepath.Join(t.TempDir(), "code"))
	checkGraphErr(t, err)
	defer cs.Close()
	cw := cs.NewWriter()
	checkGraphErr(t, cw.PutSymbol(&codestore.SymbolRecord{Symbol: "scip-test npm pkg 1 Read().", DisplayName: "Read", Kind: "UnspecifiedKind"}))
	checkGraphErr(t, cw.Flush())
	_, err = Build(home, repo, Sources{Code: cs, Git: gs, Schema: ss})
	checkGraphErr(t, err)
	g, err := graphstore.Open(Layout(home, repo).BadgerDir)
	checkGraphErr(t, err)
	defer g.Close()
	nodes, err := g.AllNodes()
	checkGraphErr(t, err)
	edges, err := g.AllEdges()
	checkGraphErr(t, err)
	counts := map[string]int{}
	for _, n := range nodes {
		counts[n.Type]++
	}
	if !reflect.DeepEqual(counts, map[string]int{"author": 1, "file": 2, "table": 2, "function": 1}) {
		t.Errorf("node counts=%v", counts)
	}
	rels := map[string]int{}
	for _, e := range edges {
		rels[e.SourceDomain+":"+e.Type]++
	}
	if !reflect.DeepEqual(rels, map[string]int{"git:changed_with": 1, "schema:fk": 1}) {
		t.Errorf("edge counts=%v", rels)
	}
	for _, tt := range []struct{ from, to, relation string }{{"a.go", "b.go", "changed_with"}, {"children", "parents", "fk"}, {"Read", "parents", ""}} {
		path, err := FindPath(g, tt.from, tt.to)
		checkGraphErr(t, err)
		if tt.relation == "" {
			if path.Found {
				t.Error("fabricated code-to-schema relationship")
			}
		} else if !path.Found || !reflect.DeepEqual(path.Edges, []string{tt.relation}) {
			t.Errorf("path=%+v", path)
		}
	}
}
