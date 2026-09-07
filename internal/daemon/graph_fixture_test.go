package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/git"
	"github.com/jeffdhooton/scry/internal/graph"
	graphstore "github.com/jeffdhooton/scry/internal/graph/store"
	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/query"
	"github.com/jeffdhooton/scry/internal/rpc"
	scipparse "github.com/jeffdhooton/scry/internal/sources/scip"
	"github.com/jeffdhooton/scry/internal/store"
	scip "github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

type graphFixtureExpectation struct {
	SourceSHA256 map[string]string       `json:"source_sha256"`
	IndexSHA256  string                  `json:"index_sha256"`
	Nodes        []graphstore.NodeRecord `json:"nodes"`
	Edges        []graphstore.EdgeRecord `json:"edges"`
	Query        string                  `json:"query"`
	From         string                  `json:"from"`
	To           string                  `json:"to"`
	Unrelated    string                  `json:"unrelated"`
}

// The child knows only a socket and request, and uses the normal RPC client.
func TestGraphFixtureClientProcess(t *testing.T) {
	if os.Getenv("SCRY_GRAPH_FIXTURE_CLIENT") != "1" {
		t.Skip("subprocess helper")
	}
	c, err := rpc.Dial(os.Getenv("SCRY_GRAPH_FIXTURE_SOCKET"))
	if err != nil {
		panic(err)
	}
	defer c.Close()
	var out json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = c.Call(ctx, os.Getenv("SCRY_GRAPH_FIXTURE_METHOD"), json.RawMessage(os.Getenv("SCRY_GRAPH_FIXTURE_PARAMS")), &out)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(out)
	os.Exit(0)
}

func graphFixtureCall(t *testing.T, socket, method string, params, out any) {
	t.Helper()
	b, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestGraphFixtureClientProcess$")
	cmd.Env = append(os.Environ(), "SCRY_GRAPH_FIXTURE_CLIENT=1", "SCRY_GRAPH_FIXTURE_SOCKET="+socket, "SCRY_GRAPH_FIXTURE_METHOD="+method, "SCRY_GRAPH_FIXTURE_PARAMS="+string(b))
	start := time.Now()
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v: %s", method, err, result)
	}
	t.Logf("RPC %s params=%s result=%s client_elapsed=%s", method, b, result, time.Since(start))
	if err := json.Unmarshal(result, out); err != nil {
		t.Fatal(err)
	}
}

func startGraphFixtureServer(t *testing.T, home string) (*Daemon, string, func()) {
	t.Helper()
	// Register the production handlers without constructing providers or watchers.
	d := &Daemon{layout: LayoutFor(home), registry: NewRegistry(), graphRegistry: NewGraphRegistry(), schemaRegistry: NewSchemaRegistry(), server: rpc.NewServer()}
	// New's git registry is needed by graph.build, but no daemon lifecycle runs.
	d.gitRegistry = git.NewRegistry()
	d.registerGraphMethods()
	d.registerMethods()
	dir, err := os.MkdirTemp("/tmp", "sg-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "rpc.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.server.Serve(ctx, ln) }()
	return d, socket, func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
		d.graphRegistry.CloseAll()
		d.registry.CloseAll()
		d.gitRegistry.CloseAll()
		d.schemaRegistry.CloseAll()
	}
}

func TestIndexedCodeGraphFixture(t *testing.T) {
	for _, scenario := range []struct {
		lang   string
		legacy bool
	}{{"go", false}, {"typescript", false}, {"go", true}, {"typescript", true}} {
		lang := scenario.lang
		mode := "parsed"
		if scenario.legacy {
			mode = "legacy-kinds"
		}
		t.Run(lang+"/"+mode, func(t *testing.T) {
			fixture := filepath.Join("..", "sources", "scip", "testdata", lang)
			data, err := os.ReadFile(filepath.Join(fixture, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			var want graphFixtureExpectation
			if err := json.Unmarshal(data, &want); err != nil {
				t.Fatal(err)
			}
			for file, wantHash := range want.SourceSHA256 {
				b, err := os.ReadFile(filepath.Join(fixture, file))
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(b)
				if hex.EncodeToString(sum[:]) != wantHash {
					t.Fatalf("source fixture hash changed: %s", file)
				}
			}
			raw, err := os.ReadFile(filepath.Join(fixture, "index.scip"))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			if hex.EncodeToString(sum[:]) != want.IndexSHA256 {
				t.Fatal("producer fixture hash changed")
			}
			var idx scip.Index
			if err := proto.Unmarshal(raw, &idx); err != nil {
				t.Fatal(err)
			}
			home, repo := t.TempDir(), t.TempDir()
			for _, doc := range idx.Documents {
				b, err := os.ReadFile(filepath.Join(fixture, doc.RelativePath))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(repo, doc.RelativePath), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			// Rebase ONLY metadata; source, symbol IDs, kinds, ranges and relationships
			// remain exactly as emitted by the pinned producer. No /tmp fixture dependency.
			idx.Metadata.ProjectRoot = "file://" + repo
			rebased, err := proto.Marshal(&idx)
			if err != nil {
				t.Fatal(err)
			}
			scipPath := filepath.Join(t.TempDir(), "index.scip")
			if err := os.WriteFile(scipPath, rebased, 0600); err != nil {
				t.Fatal(err)
			}
			code, err := store.Open(index.Layout(home, repo).BadgerDir)
			if err != nil {
				t.Fatal(err)
			}
			stats, err := scipparse.Parse(context.Background(), scipPath, code)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("parser stats=%+v", stats)
			if scenario.legacy {
				// Reconstruct the original parser's kind fields from the unmodified
				// producer metadata. Every occurrence and relationship stays persisted.
				w := code.NewWriter()
				for _, doc := range idx.Documents {
					for _, si := range doc.Symbols {
						rec, err := code.GetSymbol(si.Symbol)
						if err != nil {
							t.Fatal(err)
						}
						if rec == nil {
							continue
						}
						rec.Kind = si.Kind.String()
						if err := w.PutSymbol(rec); err != nil {
							t.Fatal(err)
						}
					}
				}
				if err := w.Flush(); err != nil {
					t.Fatal(err)
				}
			}
			if err := code.Close(); err != nil {
				t.Fatal(err)
			}
			d, socket, closeServer := startGraphFixtureServer(t, home)
			var build GraphBuildResult
			graphFixtureCall(t, socket, "graph.build", GraphBuildParams{Repo: repo}, &build)
			entry, err := d.graphRegistry.Get(home, repo)
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := entry.Store.AllNodes()
			if err != nil {
				t.Fatal(err)
			}
			edges, err := entry.Store.AllEdges()
			if err != nil {
				t.Fatal(err)
			}
			counts, relations := map[string]int{}, map[string]int{}
			for _, n := range nodes {
				counts[n.Type]++
			}
			for _, e := range edges {
				relations[e.Type]++
			}
			t.Logf("node_types=%v edge_relations=%v", counts, relations)
			closeServer()
			// Reopen both persisted stores in a new handler instance, then query from
			// separate client processes with no in-memory parser/build state.
			d, socket, closeServer = startGraphFixtureServer(t, home)
			defer closeServer()
			var found struct {
				Matches []graphstore.NodeRecord `json:"matches"`
				Total   int                     `json:"total"`
			}
			graphFixtureCall(t, socket, "graph.query", GraphQueryParams{Repo: repo, Query: want.Query}, &found)
			// Decode added fields explicitly so this same harness can also compile
			// against the baseline revision for isolated reversal verification.
			var path struct {
				graph.PathResult
				Nodes         []graphstore.NodeRecord `json:"nodes"`
				Relationships []graphstore.EdgeRecord `json:"relationships"`
			}
			graphFixtureCall(t, socket, "graph.path", GraphPathParams{Repo: repo, From: want.From, To: want.To}, &path)
			var unrelated graph.PathResult
			graphFixtureCall(t, socket, "graph.path", GraphPathParams{Repo: repo, From: want.From, To: want.Unrelated}, &unrelated)
			// Log everything before asserting, preserving an informative failing baseline.
			if len(nodes) != len(want.Nodes) {
				t.Errorf("nodes=%d want %d", len(nodes), len(want.Nodes))
			}
			byKey := map[string]graphstore.NodeRecord{}
			for _, n := range nodes {
				byKey[n.Key()] = n
			}
			for _, n := range want.Nodes {
				got, ok := byKey[n.Key()]
				if !ok || got.ID != n.ID || got.Name != n.Name || got.File != n.File || got.Line != n.Line {
					t.Errorf("node %s: got %+v want %+v", n.Key(), got, n)
				}
			}
			if !reflect.DeepEqual(edges, want.Edges) {
				t.Errorf("edges=%+v want %+v", edges, want.Edges)
			}
			if found.Total != 1 || len(found.Matches) != 1 {
				t.Errorf("named query=%+v", found)
			}
			if !path.Found || path.Distance != 1 || !reflect.DeepEqual(path.Edges, []string{"implements"}) {
				t.Errorf("implementation path=%+v", path)
			}
			if path.Found {
				if len(path.Nodes) != 2 || len(path.Relationships) != 1 {
					t.Errorf("unattributed path: %+v", path)
				} else {
					e := path.Relationships[0]
					if e.SrcKey != path.Nodes[0].Key() || e.DstKey != path.Nodes[1].Key() || e.Type != "implements" || e.SourceDomain != "code" {
						t.Errorf("path lost stored endpoints: %+v", path)
					}
					for _, n := range path.Nodes {
						expected := byKey[n.Key()]
						if !reflect.DeepEqual(n, expected) || n.File == "" || n.Line == 0 {
							t.Errorf("path attribution=%+v want %+v", n, expected)
						}
					}
				}
			}
			if unrelated.Found {
				t.Errorf("fabricated unrelated path=%+v", unrelated)
			}
			// Ordinary source query APIs survive the store reopen too.
			for _, method := range []string{"defs", "refs", "callers", "impls", "callees"} {
				var q query.Result
				name := want.Query
				if method == "callees" {
					name = want.Unrelated
				}
				graphFixtureCall(t, socket, method, map[string]string{"repo": repo, "name": name}, &q)
				if method == "callees" {
					if q.Total != 0 {
						t.Errorf("unrelated callees=%+v", q)
					}
				} else {
					expectedTotal := 1
					if lang == "typescript" && (method == "refs" || method == "callers") {
						expectedTotal = 2
					}
					if q.Total != expectedTotal {
						t.Errorf("%s total=%d want %d", method, q.Total, expectedTotal)
					}
					if method == "defs" && q.Total == 1 {
						occ := q.Matches[0].Occurrences[0]
						n := byKey["type:"+occ.Symbol]
						if n.ID != "" && (occ.File != n.File || occ.Line != n.Line || occ.Context == "") {
							t.Errorf("definition attribution=%+v node=%+v", occ, n)
						}
					}
				}
			}
			var callees query.Result
			invoke := "invoke"
			if lang == "go" {
				invoke = "Invoke"
			}
			graphFixtureCall(t, socket, "callees", QueryParams{Repo: repo, Name: invoke}, &callees)
			expectedCallees := 3
			if lang == "go" {
				expectedCallees = 0
			}
			if callees.Total != expectedCallees {
				t.Errorf("indexer callee coverage=%d want %d", callees.Total, expectedCallees)
			}
			if lang == "go" {
				var absent graph.PathResult
				graphFixtureCall(t, socket, "graph.path", GraphPathParams{Repo: repo, From: "Invoke", To: "Speaker"}, &absent)
				if absent.Found {
					t.Error("invented Go scope relationship absent from index")
				}
			}
			graphFixtureCall(t, socket, "graph.build", GraphBuildParams{Repo: repo}, &build)
			entry, err = d.graphRegistry.Get(home, repo)
			if err != nil {
				t.Fatal(err)
			}
			rebuiltNodes, err := entry.Store.AllNodes()
			if err != nil {
				t.Fatal(err)
			}
			rebuiltEdges, err := entry.Store.AllEdges()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(nodes, rebuiltNodes) || !reflect.DeepEqual(edges, rebuiltEdges) {
				t.Error("unchanged rebuild altered nodes or edges")
			}
			other := t.TempDir()
			graphFixtureCall(t, socket, "graph.build", GraphBuildParams{Repo: other}, &build)
			graphFixtureCall(t, socket, "graph.query", GraphQueryParams{Repo: other, Query: want.Query}, &found)
			if found.Total != 0 {
				t.Error("fixture leaked into another repository")
			}
		})
	}
}
