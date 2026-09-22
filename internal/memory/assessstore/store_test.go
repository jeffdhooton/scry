package assessstore

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/distill"
	"github.com/jeffdhooton/scry/internal/memory/extract"
	"github.com/jeffdhooton/scry/internal/memory/store"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDurableClaimsDedupReplayAndExpiry(t *testing.T) {
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	s, e := Open(dir, Options{Now: func() time.Time { return now }})
	if e != nil {
		t.Fatal(e)
	}
	src, e := s.Capture(Source{EpisodeID: "ep", Text: "User: a fact", OccurredAt: now})
	if e != nil {
		t.Fatal(e)
	}
	result := extract.Result{EpisodeSummary: "summary", Facts: []extract.Fct{{Fact: "a fact", Confidence: .9}}}
	jobs, e := s.Enqueue(src.ID, result, Versions{Model: "jev-1.13.0", Rubric: "v2", ContextPolicy: "v1"})
	if e != nil || len(jobs) != 1 {
		t.Fatal(jobs, e)
	}
	again, e := s.Enqueue(src.ID, result, jobs[0].Versions)
	if e != nil || again[0].ID != jobs[0].ID {
		t.Fatal(again, e)
	}
	j, e := s.Claim("worker")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Finish(j.ID, "wrong", Failed, "transport_error", nil); !errors.Is(e, ErrOwnership) {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(dir, Options{Now: func() time.Time { return now }})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	j, e = s.GetJob(j.ID)
	if e != nil || j.Status != Failed || j.Error != "interrupted_delivery_unknown" {
		t.Fatal(j, e)
	}
	if _, e = s.Replay(j.ID); !errors.Is(e, ErrEvidenceUnavailable) {
		t.Fatal(e)
	}
	now = now.Add(31 * 24 * time.Hour)
	if e = s.Cleanup(); e != nil {
		t.Fatal(e)
	}
	src, e = s.GetSource(src.ID)
	if e != nil || !src.Unavailable || src.Text != "" {
		t.Fatal(src, e)
	}
}
func TestCapacityAndPersistentBlock(t *testing.T) {
	now := time.Now()
	dir := t.TempDir()
	s, e := Open(dir, Options{Now: func() time.Time { return now }, Limits: Limits{PendingJobs: 1}})
	if e != nil {
		t.Fatal(e)
	}
	src, _ := s.Capture(Source{EpisodeID: "ep", Text: "hello"})
	_, e = s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "one"}, {Fact: "two"}}}, Versions{})
	if !errors.Is(e, ErrCapacity) {
		t.Fatal(e)
	}
	st, _ := s.Status()
	if st.CoverageGaps["pending_capacity"] != 1 {
		t.Fatal(st)
	}
	if e = s.Block("rate_limited", now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(dir, Options{Now: func() time.Time { return now }})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Resume(); !errors.Is(e, ErrCooldown) {
		t.Fatal(e)
	}
	now = now.Add(time.Hour)
	if e = s.Resume(); e != nil {
		t.Fatal(e)
	}
}

func TestSnapshotRedactionPinningAndPagination(t *testing.T) {
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	s, e := Open(t.TempDir(), Options{Now: func() time.Time { return now }})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	first, e := s.Capture(Source{EpisodeID: "older", Text: "User: Authorization: Bearer sk-abcdefghijklmnopqrstuvwxyz123456", OccurredAt: now})
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(first.Text, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatal("secret retained")
	}
	target, _ := s.Capture(Source{EpisodeID: "target", Text: "User: original", OccurredAt: now})
	result := extract.Result{Facts: []extract.Fct{{Fact: "original", Src: "user", Relation: "said", Dst: "original", Confidence: .75}}}
	jobs, e := s.Enqueue(target.ID, result, Versions{})
	if e != nil {
		t.Fatal(e)
	}
	result.Facts[0].Fact = "mutated"
	j, _ := s.GetJob(jobs[0].ID)
	if j.Candidate.Fact != "original" {
		t.Fatal("mutable original")
	}
	later, _ := s.Capture(Source{EpisodeID: "later", Text: "future", OccurredAt: now})
	page, e := s.Sources(SourceQuery{Cutoff: j.SourceCutoff, Limit: 1})
	if e != nil || len(page.Sources) != 1 || page.Sources[0].ID != first.ID {
		t.Fatal(page, e)
	}
	page, e = s.Sources(SourceQuery{Cutoff: j.SourceCutoff, After: page.Next, Limit: 10})
	if e != nil || len(page.Sources) != 1 || page.Sources[0].ID != target.ID {
		t.Fatal(page, e)
	}
	now = now.Add(31 * 24 * time.Hour)
	if e = s.Cleanup(); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{first.ID, target.ID} {
		v, _ := s.GetSource(id)
		if v.Unavailable {
			t.Fatal("pending evidence expired")
		}
	}
	v, _ := s.GetSource(later.ID)
	if !v.Unavailable {
		t.Fatal("unneeded payload retained")
	}
}
func TestPacketReplayAndReadOnly(t *testing.T) {
	dir := t.TempDir()
	s, e := Open(dir, Options{})
	if e != nil {
		t.Fatal(e)
	}
	src, _ := s.Capture(Source{EpisodeID: "ep", Text: "hello"})
	jobs, _ := s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "hello"}}}, Versions{})
	j, e := s.Claim("one")
	if e != nil {
		t.Fatal(e)
	}
	packet := []byte(`{ "state": {"candidate": "hello"} }`)
	manifest := assess.Manifest{SourceRevision: "1"}
	if e = s.SavePacket(j.ID, "one", packet, manifest); e != nil {
		t.Fatal(e)
	}
	if e = s.SavePacket(j.ID, "one", []byte(`{"changed":true}`), manifest); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	if e = s.Finish(j.ID, "one", Failed, "unsafe https://secret.example", nil); e != nil {
		t.Fatal(e)
	}
	replay, e := s.Replay(j.ID)
	if e != nil || replay.ParentID != j.ID || replay.ID == j.ID || !bytes.Equal(replay.Packet, packet) {
		t.Fatal(replay, e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(dir, Options{ReadOnly: true})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	orig, e := s.GetJob(jobs[0].ID)
	if e != nil || orig.Error != "assessment_error" || !bytes.Equal(orig.Packet, packet) {
		t.Fatal(orig, e)
	}
	if _, e = s.Claim("bad"); !errors.Is(e, ErrReadOnly) {
		t.Fatal(e)
	}
}
func TestCaptureAdapterAndDerivedSummary(t *testing.T) {
	s, e := Open(t.TempDir(), Options{})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	v, e := s.CapturePending(store.PendingEpisode{ID: "ep", SourceRef: "/remote/session#1-3", Text: "one", SourceNamespace: "host", SourceSpanKnown: true, SourceStart: 1, SourceEnd: 3, SourceTurns: []distill.SourceTurn{{Speaker: "user", Text: "one", Start: 1, End: 3}}}, SourceMetadata{})
	if e != nil || v.Namespace != "host" || v.SessionID != "/remote/session" || !v.SpanKnown || len(v.Turns) != 1 {
		t.Fatal(v, e)
	}
	again, e := s.CapturePending(store.PendingEpisode{ID: "ep", SourceRef: "/remote/session#1-3", Text: "one", SourceNamespace: "host", SourceSpanKnown: true, SourceStart: 1, SourceEnd: 3, SourceTurns: []distill.SourceTurn{{Speaker: "user", Text: "one", Start: 1, End: 3}}}, SourceMetadata{})
	if e != nil || again.ID != v.ID || again.Revision != v.Revision {
		t.Fatal(again, e)
	}
	derived, e := s.Capture(Source{EpisodeID: "ep", Kind: "derived_summary", Text: "summary"})
	if e != nil || !derived.SourceUnavailable || derived.Unavailable {
		t.Fatal(derived, e)
	}
	jobs, e := s.Enqueue(derived.ID, extract.Result{Facts: []extract.Fct{{Fact: "summary"}}}, Versions{})
	if e != nil || len(jobs) != 1 || jobs[0].SourceID != derived.ID {
		t.Fatalf("pre-extracted candidate must retain explicitly missing raw source: %+v %v", jobs, e)
	}
}

func TestPayloadPressureExpiresTerminalBeforeRefusal(t *testing.T) {
	now := time.Now()
	s, e := Open(t.TempDir(), Options{Now: func() time.Time { return now }, Limits: Limits{PayloadBytes: 2000}})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	old, e := s.Capture(Source{EpisodeID: "old", Text: strings.Repeat("a", 1400)})
	if e != nil {
		t.Fatal(e)
	}
	now = now.Add(31 * 24 * time.Hour)
	if _, e = s.Capture(Source{EpisodeID: "new", Text: strings.Repeat("b", 1400)}); e != nil {
		t.Fatal(e)
	}
	expired, _ := s.GetSource(old.ID)
	if !expired.Unavailable || expired.Text != "" {
		t.Fatal(expired)
	}
	if _, e = s.Capture(Source{EpisodeID: "overflow", Text: strings.Repeat("c", 1400)}); !errors.Is(e, ErrCapacity) {
		t.Fatal(e)
	}
	st, _ := s.Status()
	if st.PayloadBytes > 2000 || st.CoverageGaps["payload_capacity"] != 1 {
		t.Fatal(st)
	}
}
func TestExpiryRefusesReplayAndDeletesMetadataLater(t *testing.T) {
	now := time.Now()
	s, e := Open(t.TempDir(), Options{Now: func() time.Time { return now }})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	src, _ := s.Capture(Source{EpisodeID: "ep", Text: "hello"})
	s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "hello"}}}, Versions{})
	j, _ := s.Claim("one")
	if e = s.SavePacket(j.ID, "one", []byte(`{"state":"hello"}`), assess.Manifest{}); e != nil {
		t.Fatal(e)
	}
	if e = s.Finish(j.ID, "one", Failed, "transport_error", nil); e != nil {
		t.Fatal(e)
	}
	now = now.Add(31 * 24 * time.Hour)
	if _, e = s.Replay(j.ID); !errors.Is(e, ErrEvidenceUnavailable) {
		t.Fatal(e)
	}
	got, _ := s.GetJob(j.ID)
	if !got.EvidenceUnavailable || len(got.Packet) > 0 || got.PacketHash == "" {
		t.Fatal(got)
	}
	now = now.Add(60 * 24 * time.Hour)
	if e = s.Cleanup(); e != nil {
		t.Fatal(e)
	}
	if _, e = s.GetJob(j.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	st, _ := s.Status()
	if st.PayloadBytes != 0 || st.MetadataBytes != 0 {
		t.Fatal(st)
	}
}
func TestConcurrentClaimsAreExclusiveAndInvalidResultsRefused(t *testing.T) {
	s, e := Open(t.TempDir(), Options{})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	src, _ := s.Capture(Source{EpisodeID: "ep", Text: "hello"})
	s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "hello"}}}, Versions{Model: assess.Model})
	var wg sync.WaitGroup
	results := make(chan Job, 2)
	for _, owner := range []string{"a", "b"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			j, e := s.Claim(owner)
			if e == nil {
				results <- j
			} else if !errors.Is(e, ErrNotFound) {
				t.Error(e)
			}
		}(owner)
	}
	wg.Wait()
	close(results)
	var got []Job
	for j := range results {
		got = append(got, j)
	}
	if len(got) != 1 {
		t.Fatal(got)
	}
	j := got[0]
	if e = s.SavePacket(j.ID, j.Owner, []byte(`{"state":"hello"}`), assess.Manifest{}); e != nil {
		t.Fatal(e)
	}
	if e = s.Finish(j.ID, j.Owner, Completed, "", &assess.Assessment{Model: assess.Model}); e == nil {
		t.Fatal("invalid assessment accepted")
	}
	if e = s.SavePacket(j.ID, j.Owner, []byte(`{"secret":"sk-abcdefghijklmnopqrstuvwxyz123456"}`), assess.Manifest{}); e == nil {
		t.Fatal("secret packet accepted")
	}
}

func TestDescendingSourceSnapshot(t *testing.T) {
	s, e := Open(t.TempDir(), Options{})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var records []Source
	for _, text := range []string{"one", "two", "three", "four"} {
		v, e := s.Capture(Source{EpisodeID: text, Text: text})
		if e != nil {
			t.Fatal(e)
		}
		records = append(records, v)
	}
	page, e := s.Sources(SourceQuery{Cutoff: records[2].Revision, Descending: true, Limit: 2})
	if e != nil || len(page.Sources) != 2 || page.Sources[0].ID != records[2].ID || page.Sources[1].ID != records[1].ID {
		t.Fatal(page, e)
	}
	page, e = s.Sources(SourceQuery{Cutoff: records[2].Revision, Descending: true, After: page.Next, Limit: 2})
	if e != nil || len(page.Sources) != 1 || page.Sources[0].ID != records[0].ID || page.Next != 0 {
		t.Fatal(page, e)
	}
}

func TestCompletionMetadataReservedBeforeDispatch(t *testing.T) {
	s, e := Open(t.TempDir(), Options{Limits: Limits{MetadataBytes: 30000}})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	src, e := s.Capture(Source{EpisodeID: "target", Text: "hello"})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "hello"}, {Fact: "hello again"}}}, Versions{Model: assess.Model})
	if e != nil {
		t.Fatal(e)
	}
	before, _ := s.Status()
	j, e := s.Claim(strings.Repeat("x", 128))
	if e != nil {
		t.Fatal(e)
	}
	claimed, _ := s.Status()
	if claimed.MetadataBytes != before.MetadataBytes {
		t.Fatal("claim consumed unreserved metadata", before.MetadataBytes, claimed.MetadataBytes)
	}
	if e = s.SavePacket(j.ID, j.Owner, []byte(`{"state":"hello"}`), assess.Manifest{}); e != nil {
		t.Fatal(e)
	}
	ready, _ := s.Status()
	if ready.MetadataBytes < 8192 {
		t.Fatal("completion metadata not reserved", ready)
	}
	if e = s.ReserveDispatch(j.ID, j.Owner, 1, time.Time{}); e != nil {
		t.Fatal(e)
	}
	// Concurrent intake tries to fill every remaining metadata byte before result delivery.
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_, e := s.Capture(Source{EpisodeID: fmt.Sprintf("%d-%d", n, i), Text: "other", SourceRef: strings.Repeat("m", 300)})
				if errors.Is(e, ErrCapacity) {
					return
				}
				if e != nil {
					t.Error(e)
					return
				}
			}
		}(n)
	}
	wg.Wait()
	p := .8
	zero := 0.
	one := 1.
	result := assess.Assessment{Model: assess.Model, Usage: assess.Usage{InputTokens: 300, OutputTokens: 0}, LatencyMS: 120, Answers: map[string]assess.Answer{
		"supported": {Type: "noul", Noul: &p}, "durable": {Type: "noul", Noul: &p}, "assertion": {Type: "choice", Choice: "established", Confidence: &one, Probabilities: map[string]*float64{"established": &one, "planned": &zero, "hypothetical": &zero, "denied": &zero, "unclear": &zero}}}}
	if e = s.Finish(j.ID, j.Owner, Completed, "", &result); e != nil {
		t.Fatal("paid result was not retained", e)
	}
	got, e := s.GetJob(j.ID)
	if e != nil || got.Status != Completed || got.Assessment == nil {
		t.Fatal(got, e)
	}
	st, _ := s.Status()
	if st.MetadataBytes > 30000 {
		t.Fatal("metadata cap exceeded", st)
	}
	running, e := s.Claim("failure-owner")
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 100; i++ {
		_, e := s.Capture(Source{EpisodeID: fmt.Sprintf("fill-%d", i), Text: "other", SourceRef: strings.Repeat("m", 300)})
		if errors.Is(e, ErrCapacity) {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
	}
	if e = s.Finish(running.ID, running.Owner, Failed, strings.Repeat("a", 80), nil); e != nil {
		t.Fatal("safe terminal failure was not retained", e)
	}
	// A replay cannot reuse the original attempt's released reservation.
	if _, e = s.Replay(j.ID); !errors.Is(e, ErrCapacity) {
		t.Fatal("replay did not reserve capacity", e)
	}

}
