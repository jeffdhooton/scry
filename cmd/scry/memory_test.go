package main

import (
	"testing"
	"time"

	"github.com/jeffdhooton/scry/internal/memory/distill"
)

// TestPartitionKnownEpisodes covers F5's pure filter helper: episodes whose
// ID appears in missing still need extraction; episodes whose ID does NOT
// appear in missing are already committed from a previous run and should be
// left out of the extraction set entirely.
func TestPartitionKnownEpisodes(t *testing.T) {
	episodes := []distill.RawEpisode{
		{ID: "a", Text: "a"},
		{ID: "b", Text: "b"},
		{ID: "c", Text: "c"},
	}

	needExtract, alreadyKnown := partitionKnownEpisodes(episodes, []string{"b"})

	if len(needExtract) != 1 || needExtract[0].ID != "b" {
		t.Errorf("needExtract = %+v, want exactly episode b", needExtract)
	}
	if len(alreadyKnown) != 2 {
		t.Fatalf("alreadyKnown = %+v, want 2 episodes (a, c)", alreadyKnown)
	}
	gotIDs := map[string]bool{alreadyKnown[0].ID: true, alreadyKnown[1].ID: true}
	if !gotIDs["a"] || !gotIDs["c"] {
		t.Errorf("alreadyKnown IDs = %v, want {a, c}", gotIDs)
	}
}

func TestPartitionKnownEpisodes_EmptyMissingMeansAllKnown(t *testing.T) {
	episodes := []distill.RawEpisode{{ID: "a"}, {ID: "b"}}

	needExtract, alreadyKnown := partitionKnownEpisodes(episodes, nil)

	if len(needExtract) != 0 {
		t.Errorf("needExtract = %+v, want none (nothing missing)", needExtract)
	}
	if len(alreadyKnown) != 2 {
		t.Errorf("alreadyKnown = %+v, want both episodes", alreadyKnown)
	}
}

func TestPartitionKnownEpisodes_AllMissingMeansNoneKnown(t *testing.T) {
	episodes := []distill.RawEpisode{{ID: "a"}, {ID: "b"}}

	needExtract, alreadyKnown := partitionKnownEpisodes(episodes, []string{"a", "b"})

	if len(needExtract) != 2 {
		t.Errorf("needExtract = %+v, want both episodes", needExtract)
	}
	if len(alreadyKnown) != 0 {
		t.Errorf("alreadyKnown = %+v, want none", alreadyKnown)
	}
}

// TestFilterSince covers the existing (previously untested) pure helper
// backfill's --since gating depends on: a zero since keeps everything; a set
// since keeps only episodes at or after it.
func TestFilterSince(t *testing.T) {
	cutoff := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	episodes := []distill.RawEpisode{
		{ID: "before", OccurredAt: cutoff.Add(-time.Hour)},
		{ID: "exact", OccurredAt: cutoff},
		{ID: "after", OccurredAt: cutoff.Add(time.Hour)},
	}

	t.Run("zero since keeps everything", func(t *testing.T) {
		got := filterSince(episodes, time.Time{})
		if len(got) != 3 {
			t.Errorf("filterSince(zero) = %d episodes, want 3", len(got))
		}
	})

	t.Run("set since keeps at-or-after only", func(t *testing.T) {
		got := filterSince(episodes, cutoff)
		if len(got) != 2 {
			t.Fatalf("filterSince(cutoff) = %d episodes, want 2 (exact, after)", len(got))
		}
		ids := map[string]bool{got[0].ID: true, got[1].ID: true}
		if !ids["exact"] || !ids["after"] {
			t.Errorf("filterSince(cutoff) IDs = %v, want {exact, after}", ids)
		}
	})
}
