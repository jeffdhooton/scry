package assessstore

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/jeffdhooton/scry/internal/memory/assess"
	"github.com/jeffdhooton/scry/internal/memory/extract"
)

func dispatchJob(t *testing.T, s *Store, name string, packet bool) Job {
	t.Helper()
	src, err := s.Capture(Source{EpisodeID: name, Text: "User: hello"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Enqueue(src.ID, extract.Result{Facts: []extract.Fct{{Fact: "hello"}}}, Versions{Model: assess.Model}); err != nil {
		t.Fatal(err)
	}
	j, err := s.Claim(name)
	if err != nil {
		t.Fatal(err)
	}
	if packet {
		if err := s.SavePacket(j.ID, j.Owner, []byte(`{"state":"hello"}`), assess.Manifest{}); err != nil {
			t.Fatal(err)
		}
	}
	return j
}

func TestDispatchConcurrentLimit(t *testing.T) {
	s, err := Open(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	jobs := make([]Job, 20)
	for i := range jobs {
		jobs[i] = dispatchJob(t, s, fmt.Sprint(i), true)
	}
	var wg sync.WaitGroup
	results := make(chan error, len(jobs))
	for _, j := range jobs {
		wg.Add(1)
		go func(j Job) { defer wg.Done(); results <- s.ReserveDispatch(j.ID, j.Owner, 7, time.Time{}) }(j)
	}
	wg.Wait()
	close(results)
	success, limited := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrDispatchLimit):
			limited++
		default:
			t.Fatal(err)
		}
	}
	st, err := s.Status()
	if err != nil || success != 7 || limited != 13 || st.Dispatches != 7 {
		t.Fatalf("success=%d limited=%d status=%+v err=%v", success, limited, st, err)
	}
}

func TestDispatchRejectsWithoutMutation(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 123, time.UTC)
	for _, tc := range []struct {
		name string
		want error
	}{
		{"owner", ErrOwnership}, {"no_packet", ErrEvidenceUnavailable}, {"duplicate", ErrConflict},
		{"cap", ErrDispatchLimit}, {"expiry", ErrTrialExpired}, {"expired", ErrTrialExpired},
		{"blocked", ErrBlocked}, {"terminal", ErrOwnership}, {"overflow", ErrDispatchLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := Open(t.TempDir(), Options{Now: func() time.Time { return now }})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			j := dispatchJob(t, s, "owner", tc.name != "no_packet")
			owner, max, until := j.Owner, uint64(0), time.Time{}
			switch tc.name {
			case "owner":
				owner = "wrong"
			case "duplicate":
				if err := s.ReserveDispatch(j.ID, owner, 0, until); err != nil {
					t.Fatal(err)
				}
			case "cap":
				other := dispatchJob(t, s, "other", true)
				if err := s.ReserveDispatch(other.ID, other.Owner, 1, until); err != nil {
					t.Fatal(err)
				}
				max = 1
			case "expiry":
				until = now
			case "expired":
				until = now.Add(-time.Nanosecond)
			case "blocked":
				if err := s.Block("trial_paused", time.Time{}); err != nil {
					t.Fatal(err)
				}
			case "terminal":
				if err := s.Finish(j.ID, owner, Failed, "test", nil); err != nil {
					t.Fatal(err)
				}
			case "overflow":
				if err := s.write(func(_ *badger.Txn, st *Status) error { st.Dispatches = math.MaxUint64; return nil }); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := s.Status()
			beforeJob, _ := s.GetJob(j.ID)
			if err := s.ReserveDispatch(j.ID, owner, max, until); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			after, _ := s.Status()
			afterJob, _ := s.GetJob(j.ID)
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeJob, afterJob) {
				t.Fatal("rejection changed durable state")
			}
		})
	}
}

func TestDispatchSurvivesInterruptedDeliveryAndRetention(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 123, time.UTC)
	dir := t.TempDir()
	opts := Options{Now: func() time.Time { return now }}
	s, err := Open(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	j := dispatchJob(t, s, "interrupted", true)
	if err := s.ReserveDispatch(j.ID, j.Owner, 1, now.Add(time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close() }()
	j, err = s.GetJob(j.ID)
	if err != nil || j.Status != Failed || j.Error != "interrupted_delivery_unknown" || !j.DispatchStartedAt.Equal(now) {
		t.Fatalf("job=%+v err=%v", j, err)
	}
	st, err := s.Status()
	if err != nil || st.Dispatches != 1 {
		t.Fatal(st, err)
	}
	now = now.Add(91 * 24 * time.Hour)
	if err := s.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetJob(j.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	st, err = s.Status()
	if err != nil || st.Dispatches != 1 || st.MetadataBytes != 0 {
		t.Fatal(st, err)
	}
	next := dispatchJob(t, s, "next", true)
	if err := s.ReserveDispatch(next.ID, next.Owner, 1, time.Time{}); !errors.Is(err, ErrDispatchLimit) {
		t.Fatal(err)
	}
	if err := s.Resume(); err != nil {
		t.Fatal(err)
	}
	if err := s.ReserveDispatch(next.ID, next.Owner, 1, time.Time{}); !errors.Is(err, ErrDispatchLimit) {
		t.Fatal("resume reset cap", err)
	}
}

func TestDispatchReplayReservesFreshAttempt(t *testing.T) {
	s, err := Open(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	j := dispatchJob(t, s, "first", true)
	if err := s.ReserveDispatch(j.ID, j.Owner, 0, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(j.ID, j.Owner, Failed, "transport_error", nil); err != nil {
		t.Fatal(err)
	}
	r, err := s.Replay(j.ID)
	if err != nil || !r.DispatchStartedAt.IsZero() {
		t.Fatal(r, err)
	}
	r, err = s.Claim("replay")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReserveDispatch(r.ID, r.Owner, 0, time.Time{}); err != nil {
		t.Fatal(err)
	}
	st, err := s.Status()
	if err != nil || st.Dispatches != 2 {
		t.Fatal(st, err)
	}
}

func TestDispatchUsesReservedMetadataCapacity(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 123, time.UTC)
	s, err := Open(t.TempDir(), Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	j := dispatchJob(t, s, "owner", true)
	before, err := s.Status()
	if err != nil {
		t.Fatal(err)
	}
	s.opts.Limits.MetadataBytes = before.MetadataBytes
	if err := s.ReserveDispatch(j.ID, j.Owner, 1, time.Time{}); err != nil {
		t.Fatal(err)
	}
	ready, err := s.Status()
	if err != nil || ready.MetadataBytes != before.MetadataBytes {
		t.Fatal("dispatch consumed unreserved metadata", ready, err)
	}
	if err := s.Finish(j.ID, j.Owner, Failed, "transport_error", nil); err != nil {
		t.Fatal(err)
	}
	after, err := s.Status()
	if err != nil || after.MetadataBytes > before.MetadataBytes || after.Dispatches != 1 {
		t.Fatal(after, err)
	}
}

func TestDispatchZeroClockFailsClosed(t *testing.T) {
	s, err := Open(t.TempDir(), Options{Now: func() time.Time { return time.Time{} }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	j := dispatchJob(t, s, "owner", true)
	if err := s.ReserveDispatch(j.ID, j.Owner, 0, time.Time{}); err == nil {
		t.Fatal("zero timestamp would allow a repeated reservation")
	}
	st, err := s.Status()
	if err != nil || st.Dispatches != 0 {
		t.Fatal(st, err)
	}
}
