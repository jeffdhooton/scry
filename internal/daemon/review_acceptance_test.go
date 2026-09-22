package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/index"
	"github.com/jeffdhooton/scry/internal/memory/recall"
	"github.com/jeffdhooton/scry/internal/review"
	"github.com/jeffdhooton/scry/internal/rpc"
	"github.com/jeffdhooton/scry/internal/store"
)

type acceptanceReviewerFunc func(context.Context, review.Snapshot) (review.ReviewOutput, error)

func (f acceptanceReviewerFunc) Review(ctx context.Context, s review.Snapshot) (review.ReviewOutput, error) {
	return f(ctx, s)
}

// This is a deterministic evidence-and-scheduling acceptance test, not a live
// model-quality evaluation. The fixture reviewer refuses incomplete evidence;
// its synthetic response and usage do not demonstrate model reasoning quality.
func TestReviewPermissionRegressionAcceptance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	f := newPermissionAcceptanceFixture(t, ctx)
	home, repo, after, d := f.home, f.repo, f.after, f.daemon
	callers, write := f.callers, f.write
	decision, memoryReads, memoryWrites, memorySocket := f.decision, f.memoryReads, f.memoryWrites, f.memorySocket
	const priorDecision = acceptancePriorDecision
	var modelCalls atomic.Int64
	started := make(chan review.Snapshot, 1)
	release := make(chan struct{})
	reviewer := acceptanceReviewerFunc(func(ctx context.Context, snap review.Snapshot) (review.ReviewOutput, error) {
		modelCalls.Add(1)
		ids, err := acceptancePermissionEvidence(snap, callers, priorDecision, true)
		if err != nil {
			return review.ReviewOutput{}, err
		}
		started <- snap
		select {
		case <-release:
		case <-ctx.Done():
			return review.ReviewOutput{}, ctx.Err()
		}
		return review.ReviewOutput{
			Summary:  "Removing the emergency-service branch changes access for the emergency caller.",
			Findings: []review.Finding{{Severity: "high", Title: "Emergency incident response loses access", Detail: "EmergencyServiceRequest supplies only EmergencyService=true. The changed Allowed function now checks only Member or Admin, contradicting the recorded incident-response exception. Ordinary member and admin callers retain access.", EvidenceIDs: ids}},
			TestGaps: []string{"Assert that emergency-service callers retain access without member/admin credentials."},
			Usage:    review.Usage{Known: true, InputTokens: 900, OutputTokens: 120},
		}, nil
	})
	quiet := time.Second
	service, err := review.NewService(filepath.Join(home, "reviews"), review.Options{Enabled: true, Repos: []string{repo}, QuietPeriod: quiet, PollInterval: time.Millisecond * 10, Timeout: 10 * time.Second, MaxInputBytes: 24000, MaxRequestsPerDay: 2, Retain: 10, Provider: "fixture", Model: "deterministic-evidence-check", InputUSDPerMillion: 1, OutputUSDPerMillion: 2}, reviewer, d.reviewEnricher(true, nil, memorySocket))
	if err != nil {
		t.Fatal(err)
	}
	d.reviewService = service
	d.registerReviewMethods()
	socket := filepath.Join(home, "review.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() { _ = d.server.Serve(ctx, listener) }()

	// Foreground edits reset the quiet checkpoint; even an explicit queue must
	// not cause inference before the newest source state has settled.
	if _, err := service.Queue(repo); err != nil {
		t.Fatal(err)
	}
	observedAt := time.Now()
	if err := service.Tick(ctx, observedAt); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(repo, "permission.go"), after+"// foreground edit settled\n")
	if err := service.Tick(ctx, observedAt.Add(quiet/2)); err != nil {
		t.Fatal(err)
	}
	if err := service.Tick(ctx, observedAt.Add(quiet+quiet/2-time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	if modelCalls.Load() != 0 || service.Status().RequestsToday != 0 {
		t.Fatal("inference occurred before the stable quiet checkpoint")
	}
	tickDone := make(chan error, 1)
	go func() { tickDone <- service.Tick(ctx, observedAt.Add(quiet+quiet/2)) }()
	var reviewed review.Snapshot
	select {
	case reviewed = <-started:
	case err := <-tickDone:
		t.Fatalf("review failed before complete evidence reached reviewer: tick=%v status=%+v", err, service.Status())
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	foregroundStart := time.Now()
	// Unrelated foreground work proceeds while the background model is pending.
	// Keep it outside the observed repository so the reviewed snapshot stays fixed.
	scratch := filepath.Join(home, "foreground-result.txt")
	write(scratch, "foreground work completed during the review\n")
	if b, err := os.ReadFile(scratch); err != nil || !strings.Contains(string(b), "completed") {
		t.Fatalf("foreground work: %s %v", b, err)
	}
	if status := service.Status(); status.RequestsToday != 1 || status.Records != 1 {
		t.Fatalf("missing in-flight reservation: %+v", status)
	}
	foregroundElapsed := time.Since(foregroundStart)
	close(release)
	select {
	case err := <-tickDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	records, err := service.List(ctx, repo)
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	record := records[0]
	if record.State != "completed" || record.Freshness != "current" || !record.Provisional || record.Output == nil || len(record.Output.Findings) != 1 {
		t.Fatalf("incomplete result: %+v", record)
	}
	if record.Snapshot.ID != reviewed.ID || record.ContextID == "" {
		t.Fatal("result lost its captured source or context identity")
	}
	if record.CompletedAt.Before(record.StartedAt) || record.ElapsedMS != record.CompletedAt.Sub(record.StartedAt).Milliseconds() {
		t.Fatalf("invalid completion timing: %+v", record)
	}
	if record.Output.Usage != (review.Usage{Known: true, InputTokens: 900, OutputTokens: 120}) || service.Status().RequestsToday != 1 {
		t.Fatalf("usage/reservation mismatch: %+v", record)
	}
	validIDs := map[string]bool{}
	for _, ev := range reviewed.Evidence {
		validIDs[ev.ID] = true
	}
	for _, id := range record.Output.Findings[0].EvidenceIDs {
		if !validIDs[id] {
			t.Fatalf("ungrounded finding citation %q", id)
		}
	}
	t.Logf("deterministic fixture: completion=%dms foreground_window=%s usage=%+v snapshot=%s", record.ElapsedMS, foregroundElapsed, record.Output.Usage, record.Snapshot.ID)

	// Two independently connected agents retrieve the same immutable review.
	first, err := rpc.Dial(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := rpc.Dial(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	get := func(client *rpc.Client) review.Record {
		t.Helper()
		var got review.Record
		if err := client.Call(ctx, "review.get", map[string]string{"id": record.ID}, &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	one, two := get(first), get(second)
	if one.ID != record.ID || two.ID != record.ID || one.Freshness != "current" || two.Freshness != "current" || !reflect.DeepEqual(one.Output, two.Output) || !reflect.DeepEqual(one.Snapshot, two.Snapshot) {
		t.Fatalf("agents did not retrieve the same current review: %+v / %+v", one, two)
	}
	if len(one.Snapshot.Evidence) < 6 {
		t.Fatal("get did not expose the original captured evidence")
	}
	decision.Store("The emergency-service exception was retired after the separate incident-response migration.")
	if got := get(first); got.Freshness != "stale" {
		t.Fatalf("changed decision did not stale review: %s", got.Freshness)
	}
	decision.Store(priorDecision)
	if got := get(first); got.Freshness != "current" {
		t.Fatalf("restored decision/source should match: %s", got.Freshness)
	}
	write(filepath.Join(repo, "emergency.go"), callers["emergency.go"]+"// caller changed after review\n")
	if got := get(second); got.Freshness != "stale" {
		t.Fatalf("changed caller source did not stale review: %s", got.Freshness)
	}
	if memoryReads.Load() == 0 || memoryWrites.Load() != 0 || modelCalls.Load() != 1 {
		t.Fatalf("read-only retrieval violated: memory reads=%d writes=%d inference=%d", memoryReads.Load(), memoryWrites.Load(), modelCalls.Load())
	}
	if _, err := os.Stat(filepath.Join(home, "memory")); !os.IsNotExist(err) {
		t.Fatalf("review unexpectedly created a local memory store: %v", err)
	}
}

func acceptancePermissionEvidence(snap review.Snapshot, callers map[string]string, decision string, regression bool) ([]string, error) {
	ids := []string{}
	wantAfter := "+func Allowed(u User) bool { return u.Member || u.Admin }"
	if !regression {
		wantAfter = "+func Allowed(u User) bool { return u.EmergencyService || u.Admin || u.Member }"
	}
	foundDiff, foundStructure, foundMemory := false, false, false
	foundCallers := map[string]bool{}
	for _, ev := range snap.Evidence {
		switch ev.Kind {
		case "diff":
			if ev.Path == "permission.go" && strings.Contains(ev.Content, "-func Allowed(u User) bool { return u.Member || u.Admin || u.EmergencyService }") && strings.Contains(ev.Content, wantAfter) {
				foundDiff = true
				ids = append(ids, ev.ID)
			}
		case "structural":
			var structural struct {
				Symbol     string                   `json:"symbol"`
				References []store.OccurrenceRecord `json:"references"`
			}
			if err := json.Unmarshal([]byte(ev.Content), &structural); err != nil {
				return nil, err
			}
			if structural.Symbol != "Allowed" {
				continue
			}
			refs := map[string]bool{}
			for _, ref := range structural.References {
				refs[ref.File] = true
			}
			if len(structural.References) != 3 {
				return nil, fmt.Errorf("expected three indexed callers, got %+v", structural.References)
			}
			for path := range callers {
				if !refs[path] {
					return nil, fmt.Errorf("missing indexed caller %s", path)
				}
			}
			foundStructure = true
			ids = append(ids, ev.ID)
		case "caller":
			if want, ok := callers[ev.Path]; ok && ev.Content == want {
				foundCallers[ev.Path] = true
				ids = append(ids, ev.ID)
			}
		case "memory":
			var fact recall.FactHit
			if err := json.Unmarshal([]byte(ev.Content), &fact); err != nil {
				return nil, err
			}
			if fact.Value == decision && fact.Fact == decision && len(fact.Episodes) == 1 && fact.Episodes[0] == "episode-emergency-access-decision" {
				foundMemory = true
				ids = append(ids, ev.ID)
			}
		}
	}
	if !foundDiff || !foundStructure || !foundMemory || len(foundCallers) != 3 {
		return nil, fmt.Errorf("incomplete permission context: diff=%v structure=%v memory=%v callers=%v warnings=%v evidence=%+v", foundDiff, foundStructure, foundMemory, foundCallers, snap.Warnings, snap.Evidence)
	}
	return ids, nil
}

// Inspect every request rather than registering only known mutation names, so
// an accidental new memory write method is counted and rejected too.
func acceptanceMemoryServer(t *testing.T, ctx context.Context, home string, decision *atomic.Value, reads, writes *atomic.Int64) string {
	t.Helper()
	socket := filepath.Join(home, "memory.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				scanner := bufio.NewScanner(conn)
				for scanner.Scan() {
					var request rpc.Request
					if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
						return
					}
					response := rpc.Response{JSONRPC: "2.0", ID: request.ID}
					if request.Method != "memory.recall" {
						writes.Add(1)
						response.Error = &rpc.Error{Code: rpc.CodeMethodNotFound, Message: "fixture allows recall only"}
					} else {
						reads.Add(1)
						var params MemoryRecallParams
						if err := json.Unmarshal(request.Params, &params); err != nil || !strings.Contains(params.Query, "Allowed") {
							response.Error = &rpc.Error{Code: rpc.CodeInvalidParams, Message: "recall must use changed symbol"}
						} else {
							text := decision.Load().(string)
							result := recall.Result{Query: params.Query, Total: 1, Facts: []recall.FactHit{{Src: "permission-project", Relation: "decision", Value: text, Fact: text, ValidFrom: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), Confidence: 1, Episodes: []string{"episode-emergency-access-decision"}, EpisodeCount: 1}}}
							response.Result, _ = json.Marshal(result)
						}
					}
					if err := json.NewEncoder(conn).Encode(response); err != nil {
						return
					}
					if ctx.Err() != nil {
						return
					}
				}
			}()
		}
	}()
	return socket
}

const acceptancePriorDecision = "Keep the emergency-service exception: incident response must remain available when neither member nor admin credentials are present."

type permissionAcceptanceFixture struct {
	home, repo, after, memorySocket string
	daemon                          *Daemon
	callers                         map[string]string
	decision                        *atomic.Value
	memoryReads, memoryWrites       *atomic.Int64
	write                           func(string, string)
}

func newPermissionAcceptanceFixture(t *testing.T, ctx context.Context) *permissionAcceptanceFixture {
	t.Helper()
	home, err := os.MkdirTemp("/tmp", "scry-review-accept-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	t.Setenv("SCRY_MEMORY_SOCKET", "")
	repo := filepath.Join(home, "permission-project")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	repo, err = filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		c := exec.CommandContext(ctx, "git", append([]string{"-c", "core.hooksPath=/dev/null"}, args...)...)
		c.Dir = repo
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %v: %s", args, e, b)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	git("config", "user.name", "Review Fixture")
	git("config", "user.email", "review@example.com")
	before := "package fixture\ntype User struct { Member, Admin, EmergencyService bool }\nfunc Allowed(u User) bool { return u.Member || u.Admin || u.EmergencyService }\n"
	after := "package fixture\ntype User struct { Member, Admin, EmergencyService bool }\nfunc Allowed(u User) bool { return u.Member || u.Admin }\n"
	callers := map[string]string{
		"member.go":    "package fixture\nfunc MemberRequest() bool { return Allowed(User{Member: true}) }\n",
		"admin.go":     "package fixture\nfunc AdminRequest() bool { return Allowed(User{Admin: true}) }\n",
		"emergency.go": "package fixture\nfunc EmergencyServiceRequest() bool { return Allowed(User{EmergencyService: true}) }\n",
	}
	write(filepath.Join(repo, "permission.go"), before)
	for path, content := range callers {
		write(filepath.Join(repo, path), content)
	}
	git("add", ".")
	git("commit", "-qm", "Seed permission fixture")
	write(filepath.Join(repo, "permission.go"), after)

	// Seed the actual code index, including all three references to the changed
	// definition. Current caller bytes are then read by the real enricher.
	layout := index.Layout(home, repo)
	st, err := store.Open(layout.BadgerDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	writer := st.NewWriter()
	const symbol = "scip-go gomod fixture . Allowed."
	if err := writer.PutSymbol(&store.SymbolRecord{Symbol: symbol, DisplayName: "Allowed", Kind: "Function"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.PutOccurrence(&store.OccurrenceRecord{Symbol: symbol, File: "permission.go", Line: 3, Column: 6, IsDefinition: true}); err != nil {
		t.Fatal(err)
	}
	for path, content := range callers {
		column := strings.Index(strings.Split(content, "\n")[1], "Allowed") + 1
		if err := writer.PutOccurrence(&store.OccurrenceRecord{Symbol: symbol, File: path, Line: 2, Column: column}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{layout: LayoutFor(home), registry: NewRegistry(), server: rpc.NewServer()}
	d.registry.Put(&Entry{RepoPath: repo, Layout: layout, Store: st})

	var decision atomic.Value
	decision.Store(acceptancePriorDecision)
	var memoryReads, memoryWrites atomic.Int64
	memorySocket := acceptanceMemoryServer(t, ctx, home, &decision, &memoryReads, &memoryWrites)

	return &permissionAcceptanceFixture{home: home, repo: repo, after: after, memorySocket: memorySocket, daemon: d, callers: callers, decision: &decision, memoryReads: &memoryReads, memoryWrites: &memoryWrites, write: write}
}

// Run only with explicit authorization and fully specified provider settings.
// SCRY_REVIEW_LIVE_CASE=control selects an equivalent permission refactor;
// the default regression case removes the emergency exception. Each invocation
// reserves at most one request and saves the full record before quality checks.
func TestReviewPermissionLiveAcceptance(t *testing.T) {
	if os.Getenv("SCRY_REVIEW_LIVE") != "1" {
		t.Skip("live model quality evaluation requires SCRY_REVIEW_LIVE=1")
	}
	required := func(name string) string {
		t.Helper()
		value := os.Getenv(name)
		if strings.TrimSpace(value) == "" {
			t.Fatalf("live review requires explicit %s", name)
		}
		return value
	}
	cfg := review.ProviderConfig{Protocol: required("SCRY_REVIEW_LIVE_PROTOCOL"), BaseURL: required("SCRY_REVIEW_LIVE_BASE_URL"), Model: required("SCRY_REVIEW_LIVE_MODEL"), APIKeyEnv: required("SCRY_REVIEW_LIVE_API_KEY_ENV"), MaxOutputTokens: 4096}
	fixtureCase := os.Getenv("SCRY_REVIEW_LIVE_CASE")
	if fixtureCase == "" {
		fixtureCase = "regression"
	}
	if fixtureCase != "regression" && fixtureCase != "control" {
		t.Fatalf("unknown live fixture case %q", fixtureCase)
	}
	provider, err := review.NewProvider(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	f := newPermissionAcceptanceFixture(t, ctx)
	if fixtureCase == "control" {
		f.write(filepath.Join(f.repo, "permission.go"), "package fixture\ntype User struct { Member, Admin, EmergencyService bool }\nfunc Allowed(u User) bool { return u.EmergencyService || u.Admin || u.Member }\n")
	}
	var calls atomic.Int64
	checkedProvider := acceptanceReviewerFunc(func(ctx context.Context, snap review.Snapshot) (review.ReviewOutput, error) {
		if _, err := acceptancePermissionEvidence(snap, f.callers, acceptancePriorDecision, fixtureCase == "regression"); err != nil {
			return review.ReviewOutput{}, err
		}
		calls.Add(1)
		return provider.Review(ctx, snap)
	})
	svc, err := review.NewService(filepath.Join(f.home, "live-reviews"), review.Options{Enabled: true, Repos: []string{f.repo}, Timeout: 120 * time.Second, MaxInputBytes: 24000, MaxOutputTokens: 4096, MaxRequestsPerDay: 1, Retain: 1, Provider: cfg.Protocol, Model: cfg.Model}, checkedProvider, f.daemon.reviewEnricher(true, nil, f.memorySocket))
	if err != nil {
		t.Fatal(err)
	}
	rec, runErr := svc.Run(ctx, f.repo)
	if report := os.Getenv("SCRY_REVIEW_LIVE_REPORT"); report != "" {
		b, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(report, append(b, '\n'), 0600); err != nil {
			t.Fatalf("save live review record: %v", err)
		}
	}
	if runErr != nil {
		t.Fatalf("live review failed (state=%s elapsed=%dms): %v", rec.State, rec.ElapsedMS, runErr)
	}
	if rec.State != "completed" || rec.Output == nil {
		t.Fatalf("live review did not complete: %+v", rec)
	}
	t.Logf("live %s: provider=%s model=%s elapsed=%dms usage=%+v snapshot=%s findings=%+v", fixtureCase, rec.Provider, rec.Model, rec.ElapsedMS, rec.Output.Usage, rec.Snapshot.ID, rec.Output.Findings)
	if calls.Load() != 1 || svc.Status().RequestsToday != 1 || f.memoryReads.Load() == 0 || f.memoryWrites.Load() != 0 {
		t.Fatalf("unexpected live dispatches: calls=%d reservations=%d reads=%d writes=%d", calls.Load(), svc.Status().RequestsToday, f.memoryReads.Load(), f.memoryWrites.Load())
	}
	if !rec.Output.Usage.Known || rec.Output.Usage.InputTokens <= 0 || rec.Output.Usage.OutputTokens <= 0 {
		t.Fatalf("provider did not report real usage: %+v", rec.Output.Usage)
	}
	if fixtureCase == "control" {
		if len(rec.Output.Findings) != 0 {
			t.Fatalf("equivalent permission refactor produced false positives: %+v", rec.Output.Findings)
		}
		return
	}
	kinds := map[string]string{}
	for _, ev := range rec.Snapshot.Evidence {
		kinds[ev.ID] = ev.Kind
	}
	groundedEmergencyFinding := false
	for _, finding := range rec.Output.Findings {
		text := strings.ToLower(finding.Title + " " + finding.Detail)
		if !strings.Contains(text, "emergency") {
			continue
		}
		concrete := false
		for _, term := range []string{"deni", "block", "reject", "remov", "los", "break", "regress", "access"} {
			if strings.Contains(text, term) {
				concrete = true
			}
		}
		code, memory := false, false
		for _, id := range finding.EvidenceIDs {
			switch kinds[id] {
			case "diff", "caller", "structural":
				code = true
			case "memory":
				memory = true
			}
		}
		if concrete && code && memory {
			groundedEmergencyFinding = true
		}
	}
	if !groundedEmergencyFinding {
		t.Fatalf("live reviewer missed a concrete emergency-access regression grounded in code and the prior decision: %+v", rec.Output.Findings)
	}
}
