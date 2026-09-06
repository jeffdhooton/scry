package store

import (
	"bytes"
	"testing"
	"time"
)

func TestDisproofGenerationAcceptedTimeEncodingIsLossless(t *testing.T) {
	a := gradeGenerationBirth()
	b := a
	a.CreatedAt = time.Date(2026, 9, 6, 1, 2, 3, 0, time.FixedZone("one-second-offset", 1))
	b.CreatedAt = time.Date(2026, 9, 6, 1, 2, 3, 0, time.FixedZone("two-second-offset", 2))
	if a.CreatedAt.Equal(b.CreatedAt) {
		t.Fatal("fixture times must identify different instants")
	}
	ra, rawA, errA := identityBirthRecord(a)
	rb, rawB, errB := identityBirthRecord(b)
	if errA != nil || errB != nil {
		return
	} // Explicit refusal is acceptable.
	if ra.ID == rb.ID || bytes.Equal(rawA, rawB) {
		t.Fatal("two accepted distinct CreatedAt instants collapsed to identical selector bytes and generation ID")
	}
}
