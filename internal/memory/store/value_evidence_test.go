package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestValueEvidencePersistsAndDeduplicatesProvenance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := st.RecordValueEvidence(" READY_AFTER_FIXES ", "episode-a"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordValueEvidence("ready after fixes", "episode-a"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordValueEvidence("ready_after_fixes", "episode-b"); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	evidence, err := st.GetValueEvidence("ready after fixes")
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Normalized != "ready-after-fixes" {
		t.Fatalf("normalized = %q", evidence.Normalized)
	}
	if len(evidence.Episodes) != 2 || evidence.Episodes[0] != "episode-a" || evidence.Episodes[1] != "episode-b" {
		t.Fatalf("episodes = %#v", evidence.Episodes)
	}
	if len(evidence.Spellings) != 3 {
		t.Fatalf("spellings = %#v", evidence.Spellings)
	}
	found, err := st.HasValueEvidence("READY AFTER FIXES")
	if err != nil || !found {
		t.Fatalf("HasValueEvidence: found=%v err=%v", found, err)
	}
	if _, err := st.GetValueEvidence("never-seen"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing evidence error = %v", err)
	}
}
