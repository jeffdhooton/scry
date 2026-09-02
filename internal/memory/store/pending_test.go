package store

import (
	"errors"
	"testing"
	"time"
)

func TestPendingPutGetDelete(t *testing.T) {
	s := openTemp(t)
	now := time.Now().UTC().Truncate(time.Second)
	p := PendingEpisode{
		ID: "p1", Source: "manual", SourceRef: "manual", Text: "scry deploys to the mini",
		OccurredAt: now, EnqueuedAt: now, NextAttempt: now, Hints: []string{"scry"},
	}
	if err := s.PutPending(p); err != nil {
		t.Fatalf("PutPending: %v", err)
	}
	got, err := s.GetPending("p1")
	if err != nil {
		t.Fatalf("GetPending: %v", err)
	}
	if got.Text != p.Text || got.Hints[0] != "scry" || !got.EnqueuedAt.Equal(now) {
		t.Errorf("round trip mismatch: %+v", got)
	}
	has, err := s.HasPending("p1")
	if err != nil || !has {
		t.Fatalf("HasPending = %v, %v; want true", has, err)
	}
	if err := s.DeletePending("p1"); err != nil {
		t.Fatalf("DeletePending: %v", err)
	}
	if _, err := s.GetPending("p1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, GetPending err = %v, want ErrNotFound", err)
	}
	if has, _ := s.HasPending("p1"); has {
		t.Error("HasPending after delete = true")
	}
}

func TestPendingOrderedByEnqueuedAtAndCounts(t *testing.T) {
	s := openTemp(t)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	items := []PendingEpisode{
		{ID: "c", EnqueuedAt: base.Add(2 * time.Minute), NextAttempt: base.Add(2 * time.Minute)},
		{ID: "a", EnqueuedAt: base, NextAttempt: base},
		{ID: "b", EnqueuedAt: base.Add(time.Minute), NextAttempt: base.Add(time.Hour), Attempts: 2},
		{ID: "d", EnqueuedAt: base.Add(3 * time.Minute), NextAttempt: base, Parked: true},
	}
	for _, p := range items {
		if err := s.PutPending(p); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.Pending(0)
	if err != nil {
		t.Fatal(err)
	}
	gotOrder := ""
	for _, p := range all {
		gotOrder += p.ID
	}
	if gotOrder != "abcd" {
		t.Errorf("order = %q, want abcd", gotOrder)
	}
	two, _ := s.Pending(2)
	if len(two) != 2 || two[0].ID != "a" || two[1].ID != "b" {
		t.Errorf("Pending(2) = %+v", two)
	}

	now := base.Add(5 * time.Minute)
	ready, backoff, parked, err := s.PendingCounts(now)
	if err != nil {
		t.Fatal(err)
	}
	if ready != 2 || backoff != 1 || parked != 1 {
		t.Errorf("counts = ready %d backoff %d parked %d; want 2 1 1", ready, backoff, parked)
	}
}

func TestMetaTimeRoundTrip(t *testing.T) {
	s := openTemp(t)
	if _, found, err := s.GetMetaTime(MetaLastIngest); err != nil || found {
		t.Fatalf("missing key: found=%v err=%v", found, err)
	}
	at := time.Date(2026, 9, 2, 12, 34, 56, 0, time.UTC)
	if err := s.PutMetaTime(MetaLastIngest, at); err != nil {
		t.Fatal(err)
	}
	got, found, err := s.GetMetaTime(MetaLastIngest)
	if err != nil || !found || !got.Equal(at) {
		t.Errorf("GetMetaTime = %v, %v, %v", got, found, err)
	}
	// Meta timestamps must not be counted as episodes/entities/facts.
	e, n, f, _ := s.Counts()
	if e+n+f != 0 {
		t.Errorf("Counts after meta write = %d %d %d, want zeros", e, n, f)
	}
}

func TestMetaJSONRoundTrip(t *testing.T) {
	s := openTemp(t)
	type report struct {
		Scanned int `json:"scanned"`
	}
	if err := s.PutMetaJSON("last_sweep_report", report{Scanned: 7}); err != nil {
		t.Fatal(err)
	}
	var got report
	found, err := s.GetMetaJSON("last_sweep_report", &got)
	if err != nil || !found || got.Scanned != 7 {
		t.Errorf("GetMetaJSON = %+v, %v, %v", got, found, err)
	}
	if found, _ := s.GetMetaJSON("nope", &got); found {
		t.Error("missing meta key reported found")
	}
}
