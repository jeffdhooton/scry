package store

import (
	"bytes"
	"errors"
	"fmt"
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

func TestBackupAndRestoreRoundTrip(t *testing.T) {
	src := openTemp(t)
	now := time.Now()
	_ = src.PutEntity(Entity{Slug: "scry", Name: "scry", Type: "project", Aliases: []string{"scry daemon"}, CreatedAt: now, LastSeen: now})
	_ = src.PutFact(Fact{Src: "scry", Relation: "deployed_on", Dst: "mini", Fact: "scry runs on the mini", ValidFrom: now, Confidence: 0.9, Episodes: []string{"e1"}})
	_ = src.PutEpisode(Episode{ID: "e1", Source: "manual", OccurredAt: now, IngestedAt: now})
	_ = src.PutPending(PendingEpisode{ID: "p1", EnqueuedAt: now, NextAttempt: now})

	var buf bytes.Buffer
	n, err := src.Backup(&buf)
	if err != nil || n == 0 {
		t.Fatalf("Backup: %d bytes, %v", n, err)
	}

	dst := openTemp(t)
	_ = dst.PutEntity(Entity{Slug: "junk", Name: "junk", Type: "concept"})
	if err := dst.Restore(&buf); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if _, err := dst.GetEntity("junk"); !errors.Is(err, ErrNotFound) {
		t.Error("restore must wipe what was there before")
	}
	if slug, ok, _ := dst.ResolveAlias("scry daemon"); !ok || slug != "scry" {
		t.Errorf("alias index not restored: %q %v", slug, ok)
	}
	facts, _ := dst.FactsAbout("mini", false)
	if len(facts) != 1 {
		t.Errorf("facts about mini after restore = %d, want 1 (adj index restored)", len(facts))
	}
	if has, _ := dst.HasPending("p1"); !has {
		t.Error("pending queue not restored")
	}
	if v, _ := dst.schemaVersionOnDisk(); v != SchemaVersion {
		t.Errorf("schema version after restore = %d", v)
	}
}

func TestAttributeFactsHaveNoReverseIndexAndDistinctKeys(t *testing.T) {
	s := openTemp(t)
	now := time.Now()
	if err := s.PutFact(Fact{Src: "scry", Relation: "status", Fact: "scry is in progress", ValidFrom: now, Confidence: 0.9}); err == nil {
		t.Fatal("a fact with neither dst nor value must be rejected")
	}
	a := Fact{Src: "scry", Relation: "status", Value: "in-progress", Fact: "scry is in progress", ValidFrom: now, Confidence: 0.9, Episodes: []string{"e1"}}
	b := Fact{Src: "scry", Relation: "status", Value: "done", Fact: "scry is done", ValidFrom: now, Confidence: 0.9, Episodes: []string{"e2"}}
	for _, f := range []Fact{a, b} {
		if err := s.PutFact(f); err != nil {
			t.Fatal(err)
		}
	}
	from, _ := s.FactsFrom("scry", false)
	if len(from) != 2 {
		t.Fatalf("two attribute values with the same ValidFrom must not collide: got %d", len(from))
	}
	for _, f := range from {
		if !f.IsAttribute() || f.Dst != "" || f.Value == "" {
			t.Errorf("stored attribute fact = %+v", f)
		}
	}
	about, _ := s.FactsAbout("scry", false)
	if len(about) != 2 {
		t.Errorf("FactsAbout(scry) = %d, want 2", len(about))
	}
	if aboutVal, _ := s.FactsAbout(AttrDst("in-progress"), false); len(aboutVal) != 0 {
		t.Error("a value must not be reachable as a node")
	}
	if err := s.InvalidateFact(a.Src, a.Relation, a.KeyDst(), a.ValidFrom, now.Add(time.Minute)); err != nil {
		t.Fatalf("InvalidateFact by KeyDst: %v", err)
	}
	from, _ = s.FactsFrom("scry", false)
	if len(from) != 1 || from[0].Value != "done" {
		t.Errorf("after invalidation: %+v", from)
	}
	if err := s.DeleteFact(b.Src, b.Relation, b.KeyDst(), b.ValidFrom); err != nil {
		t.Fatalf("DeleteFact by KeyDst: %v", err)
	}
	if IsAttrDst("scry") || !IsAttrDst(AttrDst("x")) {
		t.Error("IsAttrDst wrong")
	}
}

func TestAttestAliasCountsDistinctEpisodes(t *testing.T) {
	s := openTemp(t)
	n, err := s.AttestAlias("hermes-ops", "mini", "ep1")
	if err != nil || n != 1 {
		t.Fatalf("first attestation = %d, %v", n, err)
	}
	if n, _ = s.AttestAlias("hermes-ops", "mini", "ep1"); n != 1 {
		t.Errorf("same episode twice counted %d", n)
	}
	if n, _ = s.AttestAlias("hermes-ops", "mini", "ep2"); n != 2 {
		t.Errorf("second episode = %d, want 2", n)
	}
	eps, _ := s.AliasAttestations("hermes-ops", "mini")
	if len(eps) != 2 {
		t.Errorf("attestations = %v", eps)
	}
	if eps, _ := s.AliasAttestations("nobody", "x"); len(eps) != 0 {
		t.Errorf("missing = %v", eps)
	}
	for i := range 20 {
		_, _ = s.AttestAlias("a", "b", fmt.Sprintf("e%d", i))
	}
	if eps, _ := s.AliasAttestations("a", "b"); len(eps) > maxAttestations {
		t.Errorf("attestation list not capped: %d", len(eps))
	}
}

func TestRelocateFactMovesKeyAndMergesOnCollision(t *testing.T) {
	s := openTemp(t)
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	old := Fact{Src: "a", Relation: "used_by", Dst: "b", Fact: "b uses a", ValidFrom: now, Confidence: 0.5, Episodes: []string{"e1"}}
	_ = s.PutFact(old)
	updated := old
	updated.Src, updated.Dst, updated.Relation, updated.RawRelation = "b", "a", "uses", "used_by"
	if err := s.RelocateFact(old, updated); err != nil {
		t.Fatal(err)
	}
	if from, _ := s.FactsFrom("a", true); len(from) != 0 {
		t.Errorf("old key still present: %+v", from)
	}
	got, _ := s.FactsFrom("b", true)
	if len(got) != 1 || got[0].Relation != "uses" || got[0].RawRelation != "used_by" {
		t.Fatalf("relocated = %+v", got)
	}
	if about, _ := s.FactsAbout("a", true); len(about) != 1 {
		t.Errorf("adj index not moved: %d", len(about))
	}
	// Collision: another fact lands on the same key and merges.
	other := Fact{Src: "c", Relation: "uses_flip", Dst: "d", Fact: "dup", ValidFrom: now, Confidence: 0.9, Episodes: []string{"e2"}}
	_ = s.PutFact(other)
	target := other
	target.Src, target.Dst, target.Relation = "b", "a", "uses"
	if err := s.RelocateFact(other, target); err != nil {
		t.Fatal(err)
	}
	got, _ = s.FactsFrom("b", true)
	if len(got) != 1 || got[0].Confidence != 0.9 || len(got[0].Episodes) != 2 {
		t.Errorf("merge on collision = %+v", got)
	}
	// Edge → attribute relocation drops the reverse index.
	attr := got[0]
	attr.Dst, attr.Value = "", "in-progress"
	if err := s.RelocateFact(got[0], attr); err != nil {
		t.Fatal(err)
	}
	if about, _ := s.FactsAbout("a", true); len(about) != 0 {
		t.Errorf("adj index must be gone after an attribute relocation: %+v", about)
	}
}

func TestDeleteEntityAndClaimAlias(t *testing.T) {
	s := openTemp(t)
	_ = s.PutEntity(Entity{Slug: "main", Name: "main", Type: "concept", Aliases: []string{"trunk"}})
	_ = s.PutEntity(Entity{Slug: "other", Name: "other", Type: "project"})
	if err := s.DeleteEntity("main"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetEntity("main"); !errors.Is(err, ErrNotFound) {
		t.Error("entity still present")
	}
	if _, ok, _ := s.ResolveAlias("trunk"); ok {
		t.Error("alias index still points at the deleted entity")
	}
	if err := s.DeleteEntity("main"); err != nil {
		t.Errorf("deleting twice must be fine: %v", err)
	}
	_ = s.ClaimAlias("shared name", "other")
	if slug, ok, _ := s.ResolveAlias("Shared Name"); !ok || slug != "other" {
		t.Errorf("ClaimAlias = %q %v", slug, ok)
	}
}
