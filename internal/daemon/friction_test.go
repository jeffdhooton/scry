package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/friction"
	"github.com/jeffdhooton/scry/internal/mcp"
	"github.com/jeffdhooton/scry/internal/rpc"
)

// Only the production journal handlers run in this child: no providers,
// watchers, live homes, daemon discovery, startup scripts or memory queue.
func TestFrictionServiceProcess(t *testing.T) {
	if os.Getenv("SCRY_FRICTION_TEST_SERVICE") != "1" {
		t.Skip("subprocess helper")
	}
	home := os.Getenv("SCRY_FRICTION_TEST_HOME")
	d := &Daemon{layout: LayoutFor(home), server: rpc.NewServer()}
	d.registerFrictionMethods()
	ln, err := net.Listen("unix", filepath.Join(home, "journal.sock"))
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.server.Serve(ctx, ln) }()
	fmt.Println("READY")
	_, _ = io.Copy(io.Discard, os.Stdin)
	cancel()
	if err := <-done; err != nil {
		panic(err)
	}
	d.closeFriction()
	os.Exit(0)
}

func startFrictionProcess(t *testing.T, home string) (string, func(bool)) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestFrictionServiceProcess$")
	cmd.Env = []string{"HOME=" + home, "SCRY_FRICTION_TEST_SERVICE=1", "SCRY_FRICTION_TEST_HOME=" + home}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	stop := func(kill bool) {
		once.Do(func() {
			if kill {
				_ = cmd.Process.Kill()
			}
			_ = stdin.Close()
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				if err != nil && !kill {
					t.Errorf("service exit: %v %s", err, stderr.String())
				}
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				<-done
				t.Error("service failed to stop")
			}
		})
	}
	t.Cleanup(func() { stop(true) })
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "READY\n" {
			stop(true)
			t.Fatalf("service not ready: %q %s", line, stderr.String())
		}
	case <-time.After(10 * time.Second):
		stop(true)
		t.Fatal("service readiness timeout")
	}
	return filepath.Join(home, "journal.sock"), stop
}

// frictionCLI builds the real command-line tool once and returns a runner bound
// to one service socket. Capturing the socket by value is safe across a restart:
// startFrictionProcess derives it from home, so the path is stable.
func frictionCLI(t *testing.T, home, socket string) func(input []byte, out any, args ...string) []byte {
	t.Helper()
	bin := filepath.Join(home, "scry-test")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/scry")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v %s", err, b)
	}
	return func(input []byte, out any, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		all := append([]string{"friction", "--socket", socket}, args...)
		cmd := exec.CommandContext(ctx, bin, all...)
		cmd.Env = []string{"HOME=" + home, "SCRY_MEMORY_SOCKET=/no-shared-memory"}
		cmd.Stdin = bytes.NewReader(input)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI %v: %v %s", args, err, b)
		}
		if out != nil {
			if err := json.Unmarshal(b, out); err != nil {
				t.Fatalf("CLI JSON: %v %s", err, b)
			}
		}
		return b
	}
}

func TestFrictionPilotThroughRestartedServiceAndCLI(t *testing.T) {
	// Short paths fit macOS's sockaddr_un. The literal pilot repo is metadata,
	// never opened; every actual database and process home is under this temp dir.
	home, err := os.MkdirTemp("/tmp", "sf-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	socket, stop := startFrictionProcess(t, home)
	cli := frictionCLI(t, home, socket)
	raw, err := os.ReadFile("../../docs/workflow-pilots/2026-09-06-friction-pilot-01/friction-events.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var events []friction.Event
	var receipts []friction.Receipt
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte{'\n'}) {
		var e friction.Event
		if err := friction.Decode(line, &e); err != nil {
			t.Fatal(err)
		}
		var receipt friction.Receipt
		cli(line, &receipt, "record", "-")
		if !receipt.Created || receipt.EventID != e.EventID || len(receipt.SHA256) != 64 {
			t.Fatalf("bad receipt: %+v", receipt)
		}
		events = append(events, e)
		receipts = append(receipts, receipt)
	}
	if len(events) != 3 {
		t.Fatalf("pilot fixture has %d events", len(events))
	}
	stop(false)
	socket, stop = startFrictionProcess(t, home)
	for i, want := range events {
		var got friction.Event
		cli(nil, &got, "get", want.EventID)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("event changed across process restart: got %+v want %+v", got, want)
		}
		b, _ := json.Marshal(want)
		var receipt friction.Receipt
		cli(b, &receipt, "record", "-")
		if receipt.Created || receipt.SHA256 != receipts[i].SHA256 {
			t.Fatalf("retry duplicated: %+v", receipt)
		}
	}
	var page friction.Page
	cli(nil, &page, "list", "--repo", events[0].Repository, "--run-id", events[0].RunID)
	if !reflect.DeepEqual(page.Events, events) || page.NextAfter != "" {
		t.Fatalf("exact pilot enumeration: %+v", page)
	}
	var review friction.Review
	cli(nil, &review, "review", "--repo", events[0].Repository)
	if len(review.Groups) != 3 || review.EventCount != 3 {
		t.Fatalf("pilot review: %+v", review)
	}
	for _, g := range review.Groups {
		if g.DistinctRuns != 1 || g.Recurring || g.MeasuredUserTimeCostSeconds != nil {
			t.Fatalf("false recurrence/impact: %+v", g)
		}
	}

	// Artificial second run is test data only, never claimed as a real recurrence.
	second := events[1]
	second.EventID = "TEST-second-run"
	second.RunID = "TEST-run-2"
	second.Observed = "Synthetic second-run repetition for verification."
	b, _ := json.Marshal(second)
	cli(b, nil, "record", "-")
	stop(true)            // a durable receipt must survive process death, too
	_ = os.Remove(socket) // remove only this killed test server's stale temp socket
	socket, stop = startFrictionProcess(t, home)
	defer stop(false)
	cli(nil, &review, "review", "--repo", events[0].Repository)
	if review.EventCount != 4 || review.Groups[0].DistinctRuns != 2 || !review.Groups[0].Recurring || review.Groups[0].Signature != second.Signature {
		t.Fatalf("recurrence lost across crash: %+v", review)
	}
	if review.Groups[0].Events[0].ResolutionState != "workaround_only" || !review.RecommendationsOnly {
		t.Fatalf("unresolved outcome changed: %+v", review)
	}
	t.Logf("PASS: 3 exact pilot events; clean service restart; 3 idempotent retries; synthetic second run persisted through process kill; review reports 2 runs with unresolved evidence and proposals.")

	// Use the real MCP bridge against this same isolated journal.
	srv := mcp.NewWithProfile(func() (mcp.Dialer, error) { return rpc.Dial(socket) }, mcp.ToolProfileLocal)
	request := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "scry_friction_get", "arguments": FrictionGetParams{EventID: events[1].EventID}}}
	b, _ = json.Marshal(request)
	var output bytes.Buffer
	if err := srv.Serve(context.Background(), bytes.NewReader(append(b, '\n')), &output); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Result.IsError || len(result.Result.Content) != 1 {
		t.Fatalf("MCP failed: %s", output.String())
	}
	var got friction.Event
	if err := json.Unmarshal([]byte(result.Result.Content[0].Text), &got); err != nil || !reflect.DeepEqual(got, events[1]) {
		t.Fatalf("MCP lost exact event: %s", output.String())
	}
	if _, err := os.Stat(filepath.Join(home, "memory")); !os.IsNotExist(err) {
		t.Fatal("test opened memory extraction store")
	}
}

func TestFrictionRPCValidationAndClosedStore(t *testing.T) {
	home := t.TempDir()
	d := &Daemon{layout: LayoutFor(home), server: rpc.NewServer()}
	defer d.closeFriction()
	_, err := d.handleFriction(context.Background(), "record", json.RawMessage(`{"event_id":"bad","observd":"typo"}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field swallowed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "friction")); !os.IsNotExist(err) {
		t.Fatal("invalid input opened store")
	}
	d.closeFriction()
	_, err = d.handleFriction(context.Background(), "get", json.RawMessage(`{"event_id":"event"}`))
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("late request reopened store: %v", err)
	}
}

func TestRoutingVerdictReachesReviewThroughCLI(t *testing.T) {
	home, err := os.MkdirTemp("/tmp", "sr-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	socket, stop := startFrictionProcess(t, home)
	defer stop(false)
	cli := frictionCLI(t, home, socket)

	repo := "/fixture/routing"
	record := func(id, run, kind string) {
		t.Helper()
		e := friction.Event{
			EventID: id, RunID: run, Repository: repo,
			RecordedAt: "2026-09-10T12:00:00Z", Signature: "routing.ladder",
			Observed: "The same correction was needed again.", Resolution: "Corrected in run.",
			ResolutionState: "corrected_in_run", Evidence: []string{"session:" + id},
			DestinationKind: kind,
		}
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		var receipt friction.Receipt
		cli(b, &receipt, "record", "-")
		if !receipt.Created {
			t.Fatalf("record %s: %+v", id, receipt)
		}
	}
	record("r-1", "run-1", "fact")
	record("r-2", "run-2", "fact")

	var review friction.Review
	cli(nil, &review, "review", "--repo", repo)
	if len(review.Groups) != 1 {
		t.Fatalf("groups: %+v", review)
	}
	got := review.Groups[0].Routing
	if got.Status != "outgrown" || got.CurrentKind != "fact" || got.SuggestedKind != "decision" {
		t.Fatalf("routing did not survive the daemon and CLI: %+v", got)
	}
	if got.RunsAtCurrent != 2 || got.Rationale == "" {
		t.Fatalf("%+v", got)
	}

	// An unknown kind is refused by the daemon, not silently coerced. The shared
	// cli helper fails the test on a nonzero exit, so run the binary directly here.
	bad := friction.Event{
		EventID: "r-3", RunID: "run-3", Repository: repo,
		RecordedAt: "2026-09-10T12:00:00Z", Signature: "routing.ladder",
		Observed: "x", Resolution: "y", ResolutionState: "unresolved",
		Evidence: []string{"session:r-3"}, DestinationKind: "rule",
	}
	encoded, err := json.Marshal(bad)
	if err != nil {
		t.Fatal(err)
	}
	reject := exec.Command(filepath.Join(home, "scry-test"), "friction", "--socket", socket, "record", "-")
	reject.Env = []string{"HOME=" + home, "SCRY_MEMORY_SOCKET=/no-shared-memory"}
	reject.Stdin = bytes.NewReader(encoded)
	combined, err := reject.CombinedOutput()
	if err == nil {
		t.Fatalf("unknown destination_kind accepted: %s", combined)
	}
	if !bytes.Contains(combined, []byte("destination_kind")) {
		t.Fatalf("rejection did not name the field: %s", combined)
	}
}
