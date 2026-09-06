package store

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestIndependentOutcomeDiskStagingErrorDoesNotDiscloseInput(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions(filepath.Join(t.TempDir(), "db")).WithLogger(nil).WithValueLogFileSize(1 << 20))
	if err != nil {
		t.Fatal(err)
	}
	st := &Store{db: db}
	defer st.Close()
	if err := st.ensureSchema(); err != nil {
		t.Fatal(err)
	}
	input, key := independentOutcomeInput(t, st, "src", 1)
	o := independentOutcomeAssertion(input, key)
	o.Birth = &identityBirth{EpisodeID: input.EpisodeID, Occurrence: input.Ordinal, Slug: "cedar-atlas", Name: input.Fact.Src, Origin: "endpoint", CreatedAt: input.OccurredAt}
	o.AssertionOrdinal = nil
	o.Disposition = "no-assertion"
	o.Materialization = []byte(`{"sensitive":"` + strings.Repeat("synthetic private material", 45000) + `"}`)
	before := gradeGenerationSnapshot(t, st)
	events := 0
	st.SetObserver(func(Event) { events++ })
	err = st.AtomicWrite(func(tx *Store) error { _, err := putIdentityOutcome(tx, o); return err })
	if err == nil {
		t.Fatal("fixture did not cause actual disk staging refusal")
	}
	if !reflect.DeepEqual(before, gradeGenerationSnapshot(t, st)) || events != 0 {
		t.Fatal("staging refusal leaked writes/events")
	}
	if strings.Contains(err.Error(), "00000000") || strings.Contains(err.Error(), "sensitive") || strings.Contains(err.Error(), "Cedar") {
		t.Fatalf("SAFETY FAILURE: real disk-backed Badger staging refusal dumps canonical outcome input: %s", err)
	}
}
