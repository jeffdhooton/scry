package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/ingest"
	"github.com/jeffdhooton/scry/internal/memory/queue"
	"github.com/jeffdhooton/scry/internal/memory/store"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// This is a deterministic extraction substitute, not live model-quality proof.
// Both revisions deliberately map to the SAME triple: the existing resolver
// keeps the first fact's wording. Orientation must use the authored revision.
type curatedFixtureExtractor struct{}

func (curatedFixtureExtractor) Extract(_ context.Context, ep distill.RawEpisode, _ []string) (extract.Result, error) {
	if !ep.CwdIsRepo {
		return extract.Result{}, errors.New("lost repository attestation before extraction")
	}
	return extract.Result{
		EpisodeSummary: "Model paraphrase deliberately omits the actual constraint",
		Entities:       []extract.Ent{{Name: "Scry", Type: "project"}, {Name: "Go", Type: "tool"}},
		Facts:          []extract.Fct{{Src: "Scry", Relation: "uses", Dst: "Go", Fact: strings.TrimSpace(ep.Text), Confidence: .9}},
	}, nil
}

type curatedRPCClient struct{ socket string }

func (c curatedRPCClient) call(ctx context.Context, method string, params, out any) error {
	client, err := rpc.Dial(c.socket)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Call(ctx, method, params, out)
}

func (c curatedRPCClient) Enqueue(ctx context.Context, eps []distill.RawEpisode) (int, int, error) {
	var out MemoryEnqueueResult
	err := c.call(ctx, "memory.enqueue.curated.v1", MemoryEnqueueParams{Episodes: eps}, &out)
	return out.Queued, out.Known, err
}

func (c curatedRPCClient) GetCursor(ctx context.Context, path string) (store.Cursor, bool, error) {
	var out MemoryCursorGetResult
	err := c.call(ctx, "memory.cursor.get", MemoryCursorGetParams{Path: path}, &out)
	return out.Cursor, out.Found, err
}

func (c curatedRPCClient) PutCursor(ctx context.Context, cursor store.Cursor) error {
	return c.call(ctx, "memory.cursor.put", cursor, nil)
}

// A separate OS process has no parent test state, source text, or conversation
// history. It knows only an isolated socket and cwd and requests the default
// orientation budget over the actual RPC client.
func TestCuratedOrientationClientProcess(t *testing.T) {
	if os.Getenv("SCRY_CURATED_CLIENT") != "1" {
		t.Skip("subprocess helper")
	}
	var out map[string]string
	err := (curatedRPCClient{os.Getenv("SCRY_CURATED_SOCKET")}).call(context.Background(), "memory.orient", MemoryOrientParams{Cwd: os.Getenv("SCRY_CURATED_CWD")}, &out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}

func freshCuratedOrientation(t *testing.T, socket, cwd string) string {
	t.Helper()
	if binary := os.Getenv("SCRY_CURATED_CLIENT_BINARY"); binary != "" {
		cmd := exec.Command(binary, "memory", "orient", "--cwd", cwd)
		cmd.Env = append(os.Environ(), "SCRY_MEMORY_SOCKET="+socket)
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fresh CLI client: %v: %s", err, data)
		}
		md := strings.TrimSpace(string(data))
		if len(md) > 2000 {
			t.Fatalf("normal CLI orientation budget exceeded: %d", len(md))
		}
		return md
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestCuratedOrientationClientProcess$")
	cmd.Env = append(os.Environ(), "SCRY_CURATED_CLIENT=1", "SCRY_CURATED_SOCKET="+socket, "SCRY_CURATED_CWD="+cwd)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh client: %v: %s", err, data)
	}
	var out map[string]string
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("client output: %v: %s", err, data)
	}
	if len(out["markdown"]) > 2000 {
		t.Fatalf("normal orientation budget exceeded: %d", len(out["markdown"]))
	}
	return out["markdown"]
}

type curatedTestServer struct {
	d            *Daemon
	client       curatedRPCClient
	cancel       context.CancelFunc
	done         chan error
	workerCancel context.CancelFunc
	workerDone   chan struct{}
}

func startCuratedServer(t *testing.T, home string) *curatedTestServer {
	t.Helper()
	d := New(LayoutFor(home))
	d.memExtractor = nil // no configured provider can run
	// Short unique sockets avoid macOS's Unix socket path limit. This test
	// never calls Daemon.Run: no installed daemon, watcher, hook or config.
	socketDir, err := os.MkdirTemp("", "sc-")
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(socketDir, "m.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &curatedTestServer{d: d, client: curatedRPCClient{socket}, cancel: cancel, done: make(chan error, 1)}
	go func() { s.done <- d.server.Serve(ctx, ln) }()
	return s
}

func (s *curatedTestServer) startWorker(t *testing.T) {
	t.Helper()
	st, err := s.d.memoryStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.workerCancel, s.workerDone = cancel, make(chan struct{})
	w := queue.New(queue.Options{Store: st, Extractor: curatedFixtureExtractor{}, Workers: 1, Poll: time.Millisecond, Logf: t.Logf})
	go func() { defer close(s.workerDone); w.Run(ctx) }()
}

func (s *curatedTestServer) close(t *testing.T) {
	t.Helper()
	if s.workerCancel != nil {
		s.workerCancel()
		<-s.workerDone
	}
	s.cancel()
	if err := <-s.done; err != nil {
		t.Error(err)
	}
	s.d.closeMemory()
}

func drainCurated(t *testing.T, s *curatedTestServer, wantEpisodes int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var out MemoryQueueResult
		if err := s.client.call(context.Background(), "memory.queue", nil, &out); err != nil {
			t.Fatal(err)
		}
		st, err := s.d.memoryStore()
		if err != nil {
			t.Fatal(err)
		}
		eps, err := st.AllEpisodes()
		if err != nil {
			t.Fatal(err)
		}
		if out.Ready+out.Backoff+out.Parked == 0 && len(eps) == wantEpisodes {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("queue did not complete accepted source revisions")
}

func TestCuratedConstraintDurableWorkflow(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "scry")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	repo, _ = filepath.EvalSymlinks(repo)
	path := filepath.Join(repo, "constraint.txt")
	originalBytes, err := os.ReadFile("../../docs/memory/curated-constraint.txt")
	if err != nil {
		t.Fatal(err)
	}
	original := string(originalBytes)
	edited := "Scry must build and run all release checks with CGO_ENABLED=0; verify cross-compilation before release.\n"
	s := startCuratedServer(t, home)
	defer func() {
		if s != nil {
			s.close(t)
		}
	}()
	importFile := func(text string) (ingest.Summary, error) {
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		return ingest.File(ctx, ingest.Options{Source: "curated", Repo: repo, Path: path, Daemon: s.client})
	}
	start := time.Now()
	sum, err := importFile(original)
	if err != nil || sum.EpisodesIngested != 1 {
		t.Fatalf("initial import: %+v %v", sum, err)
	}
	var pending MemoryQueueResult
	if err := s.client.call(ctx, "memory.queue", nil, &pending); err != nil {
		t.Fatal(err)
	}
	if pending.Ready != 1 || pending.Items[0].Text != original || !pending.Items[0].CwdIsRepo {
		t.Fatalf("pending source lost: %+v", pending)
	}
	firstID := pending.Items[0].ID
	if md := freshCuratedOrientation(t, s.client.socket, repo); strings.Contains(md, strings.TrimSpace(original)) {
		t.Fatal("unprocessed queue text was presented")
	}
	// Restart with work still pending: the real worker consumes persisted pq:.
	s.close(t)
	s = startCuratedServer(t, home)
	s.startWorker(t)
	drainCurated(t, s, 1)
	s.close(t)
	s = startCuratedServer(t, home)
	md := freshCuratedOrientation(t, s.client.socket, repo)
	_, hash, _ := distill.ParseCuratedRef(distill.CuratedRef(path, original))
	for _, want := range []string{strings.TrimSpace(original), path, hash, repo, "(attested)"} {
		if !strings.Contains(md, want) {
			t.Fatalf("initial orientation lacks %q: %s", want, md)
		}
	}
	t.Logf("CASE original source_to_orientation=%s source=%s revision=%s output_bytes=%d\n%s", time.Since(start), path, hash, len(md), md)
	st, _ := s.d.memoryStore()
	first, err := st.GetEpisode(firstID)
	if err != nil || first.Summary != original || first.SourceRef != distill.CuratedRef(path, original) || first.Cwd != repo || !first.CwdIsRepo {
		t.Fatalf("persisted original: %+v %v", first, err)
	}
	facts, err := st.AllFacts()
	if err != nil || len(facts) != 1 || facts[0].Fact != strings.TrimSpace(original) {
		t.Fatalf("real resolver did not commit fact: %+v %v", facts, err)
	}
	if !reflect.DeepEqual(facts[0].Episodes, []string{firstID}) {
		t.Fatal("fact lost source episode")
	}
	// A failed RPC must leave the real persisted cursor byte-for-byte intact.
	cursorKey := distill.CuratedSource + ":" + distill.MakeID(repo+"\x00"+path)
	before, _, err := s.client.GetCursor(ctx, cursorKey)
	if err != nil {
		t.Fatal(err)
	}
	s.d.server.Register("memory.enqueue.curated.v1", func(context.Context, json.RawMessage) (any, error) { return nil, errors.New("fixture enqueue refusal") })
	if _, err := importFile(edited); err == nil {
		t.Fatal("expected enqueue refusal")
	}
	after, _, err := s.client.GetCursor(ctx, cursorKey)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("failed enqueue advanced cursor")
	}
	s.d.server.Register("memory.enqueue.curated.v1", s.d.handleMemoryEnqueue)
	start = time.Now()
	// Also fail AFTER queue acceptance. Retry must acknowledge the same pq:
	// record and must not create a new observation or re-extract twice.
	s.d.server.Register("memory.cursor.put", func(context.Context, json.RawMessage) (any, error) { return nil, errors.New("fixture cursor refusal") })
	if _, err := importFile(edited); err == nil {
		t.Fatal("expected cursor refusal")
	}
	s.d.server.Register("memory.cursor.put", s.d.handleMemoryCursorPut)
	sum, err = importFile(edited)
	if err != nil || sum.EpisodesIngested != 0 || sum.EpisodesSkipped != 1 {
		t.Fatalf("accepted retry duplicated: %+v %v", sum, err)
	}
	if md := freshCuratedOrientation(t, s.client.socket, repo); !strings.Contains(md, strings.TrimSpace(original)) || strings.Contains(md, strings.TrimSpace(edited)) {
		t.Fatal("pending edit replaced completed evidence")
	}
	s.startWorker(t)
	drainCurated(t, s, 2)
	s.close(t)
	s = startCuratedServer(t, home)
	md = freshCuratedOrientation(t, s.client.socket, repo)
	_, editHash, _ := distill.ParseCuratedRef(distill.CuratedRef(path, edited))
	if !strings.Contains(md, strings.TrimSpace(edited)) || !strings.Contains(md, editHash) || strings.Contains(md, strings.TrimSpace(original)) || strings.Contains(md, hash) {
		t.Fatalf("obsolete source presented as current: %s", md)
	}
	t.Logf("CASE edit source_to_orientation=%s revision=%s output_bytes=%d\n%s", time.Since(start), editHash, len(md), md)
	st, _ = s.d.memoryStore()
	retained, err := st.GetEpisode(firstID)
	if err != nil || !reflect.DeepEqual(first, retained) {
		t.Fatalf("old source evidence changed: %+v %v", retained, err)
	}
	var exported MemoryExportResult
	if err := s.client.call(ctx, "memory.export", nil, &exported); err != nil {
		t.Fatal(err)
	}
	foundOriginal := false
	for _, ep := range exported.Episodes {
		if reflect.DeepEqual(first, ep) {
			foundOriginal = true
		}
	}
	if !foundOriginal {
		t.Fatal("original evidence is not recoverable through the existing export RPC")
	}
	facts, err = st.AllFacts()
	if err != nil || len(facts) != 1 || len(facts[0].Episodes) != 2 || facts[0].Fact != strings.TrimSpace(original) {
		t.Fatalf("legacy coalescing/history policy changed: %+v %v", facts, err)
	}
	before, _, _ = s.client.GetCursor(ctx, cursorKey)
	sum, err = importFile(edited) // changes mtime, not bytes
	after, _, _ = s.client.GetCursor(ctx, cursorKey)
	if err != nil || sum.EpisodesIngested != 0 || sum.EpisodesSkipped != 1 || !reflect.DeepEqual(before, after) {
		t.Fatalf("unchanged import effects: %+v %v", sum, err)
	}
	drainCurated(t, s, 2)
	// Same basename, sibling, and nested repository paths are all foreign.
	for _, foreign := range []string{filepath.Join(t.TempDir(), "scry"), repo + "-other", filepath.Join(repo, "nested")} {
		if err := os.MkdirAll(filepath.Join(foreign, ".git"), 0700); err != nil {
			t.Fatal(err)
		}
		start = time.Now()
		md := freshCuratedOrientation(t, s.client.socket, foreign)
		if strings.Contains(md, "CGO") || strings.Contains(md, path) || strings.Contains(md, "Curated constraint") {
			t.Fatalf("foreign source leaked into %s: %s", foreign, md)
		}
		t.Logf("CASE isolation orientation_latency=%s cwd=%s output_bytes=%d\n%s", time.Since(start), foreign, len(md), md)
	}
	// Reverting bytes is a third observation, not a duplicate of revision 1.
	sum, err = importFile(original)
	if err != nil || sum.EpisodesIngested != 1 {
		t.Fatalf("reversion: %+v %v", sum, err)
	}
	s.startWorker(t)
	drainCurated(t, s, 3)
	s.close(t)
	s = startCuratedServer(t, home)
	md = freshCuratedOrientation(t, s.client.socket, repo)
	if !strings.Contains(md, strings.TrimSpace(original)) || strings.Contains(md, strings.TrimSpace(edited)) {
		t.Fatalf("reversion selected wrong observation: %s", md)
	}
}

// Optional local evidence for the explicitly selected real repository file.
// The source is read-only; every store/socket remains an isolated fixture.
func TestCuratedSelectedRepositoryFile(t *testing.T) {
	repo := os.Getenv("SCRY_CURATED_SELECTED_ROOT")
	if repo == "" {
		t.Skip("set SCRY_CURATED_SELECTED_ROOT explicitly for selected-source evidence")
	}
	repo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, "docs/memory/curated-constraint.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	s := startCuratedServer(t, home)
	defer func() {
		if s != nil {
			s.close(t)
		}
	}()
	start := time.Now()
	sum, err := ingest.File(context.Background(), ingest.Options{Source: "curated", Repo: repo, Path: path, Daemon: s.client})
	if err != nil || sum.EpisodesIngested != 1 {
		t.Fatalf("selected source: %+v %v", sum, err)
	}
	s.startWorker(t)
	drainCurated(t, s, 1)
	s.close(t)
	s = startCuratedServer(t, home)
	md := freshCuratedOrientation(t, s.client.socket, repo)
	_, hash, _ := distill.ParseCuratedRef(distill.CuratedRef(path, string(data)))
	for _, want := range []string{strings.TrimSpace(string(data)), path, hash, repo, "(attested)"} {
		if !strings.Contains(md, want) {
			t.Fatalf("selected source lacks %q: %s", want, md)
		}
	}
	t.Logf("CASE selected-real-file source_to_orientation=%s source=%s revision=%s output_bytes=%d\n%s", time.Since(start), path, hash, len(md), md)
}
