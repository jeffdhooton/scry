package store

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

// Opt-in technical regression on an immutable backup, never a live store.
// Every mutation is confined to t.TempDir. This is not semantic approval of
// live alias removals or a backfill of earlier reviewed dispositions.
func TestAliasRejectionRestoredReplica(t *testing.T) {
	source := os.Getenv("SCRY_ALIAS_REJECTION_SOURCE_BACKUP")
	if source == "" {
		t.Skip("immutable source backup not supplied")
	}
	wantHash := os.Getenv("SCRY_ALIAS_REJECTION_SOURCE_SHA256")
	if len(wantHash) != 64 {
		t.Fatal("exact source hash required")
	}
	f, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", h.Sum(nil)) != wantHash {
		t.Fatal("source hash mismatch")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	s := openTemp(t)
	if err := s.Restore(f); err != nil {
		t.Fatal(err)
	}
	facts, err := s.AllFacts()
	if err != nil {
		t.Fatal(err)
	}
	episodes, err := s.AllEpisodes()
	if err != nil {
		t.Fatal(err)
	}
	stale, err := s.GetEntity("childscribe-laravel")
	if err != nil {
		t.Fatal(err)
	}
	req := AliasRepairRequest{}
	for _, spelling := range []string{"envoyer", "office dashboard", "driver-core worktree"} {
		req.Drops = append(req.Drops, AliasDrop{Entity: stale.Slug, Alias: spelling, Why: "Disposable technical regression fixture only; not a live ownership decision"})
	}
	p, err := s.PreviewAliasRepair(req)
	if err != nil || !p.Ready {
		t.Fatalf("preview %+v %v", p, err)
	}
	req.Expected = &p.Expected
	w := &aliasBackup{}
	before := rejectionRawState(t, s)
	if n, _, err := s.BackupAndRepairAliases(w, req); err != nil || n == 0 {
		t.Fatalf("apply %d %v", n, err)
	}
	after := rejectionRawState(t, s)
	for _, atomic := range []bool{false, true} {
		var err error
		if atomic {
			err = s.AtomicWrite(func(tx *Store) error { return tx.PutEntity(stale) })
		} else {
			err = s.PutEntity(stale)
		}
		if !errors.Is(err, ErrAliasRejected) {
			t.Fatalf("stale write atomic=%v: %v", atomic, err)
		}
		if rejectionRawState(t, s) != after {
			t.Fatal("rejected write changed raw snapshot")
		}
	}
	newFacts, err := s.AllFacts()
	if err != nil || !reflect.DeepEqual(facts, newFacts) {
		t.Fatalf("facts changed: %v", err)
	}
	newEpisodes, err := s.AllEpisodes()
	if err != nil || !reflect.DeepEqual(episodes, newEpisodes) {
		t.Fatalf("episodes changed: %v", err)
	}
	restored := openTemp(t)
	if err := restored.Restore(w); err != nil {
		t.Fatal(err)
	}
	if rejectionRawState(t, restored) != before {
		t.Fatal("backup did not restore original complete raw state")
	}
	p, err = s.PreviewAliasRepair(req)
	if err != nil || p.Ready || p.Applied || rejectionRawState(t, s) != after {
		t.Fatalf("second preview not no-write: %+v %v", p, err)
	}
	t.Logf("preserved %d facts and %d episodes; original Child aliases=%d; full raw backup restore, stale direct/atomic refusal and second no-write passed", len(facts), len(episodes), len(stale.Aliases))
}
